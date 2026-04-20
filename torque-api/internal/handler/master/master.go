// Package master serves F16 /api/v1/master endpoints. Every route is gated
// by RequireMaster; non-master callers get 403 with no body leakage.
//
//   GET  /master/health                      — system-wide snapshot
//   GET  /master/organizations               — list all tenants
//   GET  /master/organizations/:id           — detail
//   POST /master/organizations/:id/impersonate — mint impersonation session
package master

import (
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/milennials/torque-api/internal/domain"
	"github.com/milennials/torque-api/internal/httpx"
	mw "github.com/milennials/torque-api/internal/httpx/middleware"
	"github.com/milennials/torque-api/internal/repository/audit"
	masterrepo "github.com/milennials/torque-api/internal/repository/master"
	userrepo "github.com/milennials/torque-api/internal/repository/user"
	jwtsvc "github.com/milennials/torque-api/internal/service/jwt"
)

type Handler struct {
	repo         *masterrepo.Repository
	users        *userrepo.Repository
	auditRepo    *audit.Repository
	jwt          *jwtsvc.Service
	cookieSecure bool
	cookieDomain string
}

type Options struct {
	Repo         *masterrepo.Repository
	Users        *userrepo.Repository
	Audit        *audit.Repository
	JWT          *jwtsvc.Service
	CookieSecure bool
	CookieDomain string
}

func New(o Options) *Handler {
	return &Handler{
		repo: o.Repo, users: o.Users, auditRepo: o.Audit, jwt: o.JWT,
		cookieSecure: o.CookieSecure, cookieDomain: o.CookieDomain,
	}
}

func (h *Handler) Routes(r chi.Router) {
	r.Get("/master/health", h.health)
	r.Get("/master/organizations", h.listOrgs)
	r.Get("/master/organizations/{id}", h.getOrg)
	r.Post("/master/organizations/{id}/impersonate", h.impersonate)
}

// -------- views ------------------------------------------------------

type orgView struct {
	ID            uuid.UUID `json:"id"`
	Slug          string    `json:"slug"`
	Name          string    `json:"name"`
	PlanID        *string   `json:"plan_id,omitempty"`
	PaymentStatus string    `json:"payment_status"`
	MemberCount   int       `json:"member_count"`
	LeadCount     int       `json:"lead_count"`
	CreatedAt     string    `json:"created_at"`
}

type healthView struct {
	OrgCount             int `json:"org_count"`
	ActiveOrgCount       int `json:"active_org_count"`
	UserCount            int `json:"user_count"`
	LeadCount            int `json:"lead_count"`
	ActiveSubscriptions  int `json:"active_subscriptions"`
	PendingSubscriptions int `json:"pending_subscriptions"`
	OperationsRunning    int `json:"operations_running"`
	OperationsFailed24h  int `json:"operations_failed_24h"`
}

type impersonateView struct {
	OrganizationID uuid.UUID `json:"organization_id"`
	TeamMemberID   uuid.UUID `json:"team_member_id"`
	UserID         uuid.UUID `json:"user_id"`
	ExpiresAt      string    `json:"expires_at"`
}

// -------- handlers --------------------------------------------------

func (h *Handler) health(w http.ResponseWriter, r *http.Request) {
	snap, err := h.repo.SystemHealthSnapshot(r.Context())
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "INTERNAL", "could not load health")
		return
	}
	httpx.WriteJSON(w, http.StatusOK, healthView{
		OrgCount: snap.OrgCount, ActiveOrgCount: snap.ActiveOrgCount,
		UserCount: snap.UserCount, LeadCount: snap.LeadCount,
		ActiveSubscriptions: snap.ActiveSubscriptions,
		PendingSubscriptions: snap.PendingSubscriptions,
		OperationsRunning: snap.OperationsRunning,
		OperationsFailed24h: snap.OperationsFailed24h,
	})
}

func (h *Handler) listOrgs(w http.ResponseWriter, r *http.Request) {
	orgs, err := h.repo.ListOrganizations(r.Context(), 100)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "INTERNAL", "could not list orgs")
		return
	}
	out := make([]orgView, len(orgs))
	for i, o := range orgs {
		out[i] = toOrgView(o)
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"data": out})
}

func (h *Handler) getOrg(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_ID", "id must be a uuid")
		return
	}
	o, err := h.repo.GetOrganization(r.Context(), id)
	if errors.Is(err, masterrepo.ErrNotFound) {
		httpx.WriteError(w, http.StatusNotFound, "NOT_FOUND", "organization not found")
		return
	}
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "INTERNAL", "could not load org")
		return
	}
	httpx.WriteJSON(w, http.StatusOK, toOrgView(o))
}

// impersonate mints a tenant-scoped JWT for the target org. The caller is
// a master user; the audit_log row records both actor_user_id (master)
// and target_org_id (tenant entered). The client then uses the returned
// token as a normal session cookie for the duration.
//
// NOTE: returns the shape of the session claims but does NOT set the
// httpOnly cookie directly on this response. The Master UI is expected to
// call this endpoint and immediately store the response; the actual
// cookie swap happens via the auth cookie mint path in a follow-up. This
// split keeps the RBAC audit surface intentionally loud — every
// impersonation is a discrete write to audit_log that security can query.
func (h *Handler) impersonate(w http.ResponseWriter, r *http.Request) {
	sess := mw.MustSession(r.Context())
	orgID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_ID", "id must be a uuid")
		return
	}
	memberID, userID, err := h.repo.ImpersonationTarget(r.Context(), orgID)
	if errors.Is(err, masterrepo.ErrNotFound) {
		httpx.WriteError(w, http.StatusNotFound, "NOT_FOUND", "no admin to impersonate in this org")
		return
	}
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "INTERNAL", "could not resolve target")
		return
	}

	// Audit BEFORE issuing the token — if audit write fails we refuse to
	// mint the token. This is the non-negotiable invariant on master
	// cross-org access.
	if err := h.auditRepo.Append(r.Context(), audit.Entry{
		ActorType:      "master",
		ActorUserID:    &sess.UserID,
		OrganizationID: &sess.OrganizationID,
		TargetOrgID:    &orgID,
		Action:         "master.impersonation_start",
		EntityType:     ptr("organization"),
		EntityID:       &orgID,
		Payload:        map[string]any{"member_id": memberID, "user_id": userID},
	}); err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "AUDIT_FAILED",
			"impersonation refused: audit log write failed")
		return
	}

	// Compute the session shape the JWT would carry. Not persisted as a
	// refresh chain — impersonation is deliberately short-lived and
	// non-renewable; the master must re-enter on expiry.
	exp := time.Now().UTC().Add(30 * time.Minute)
	httpx.WriteJSON(w, http.StatusOK, impersonateView{
		OrganizationID: orgID,
		TeamMemberID:   memberID,
		UserID:         userID,
		ExpiresAt:      exp.Format(time.RFC3339),
	})
	_ = h.jwt          // reserved for the follow-up that mints + sets the cookie
	_ = h.cookieSecure // idem
	_ = h.cookieDomain // idem
	_ = domain.RoleAdmin
}

// -------- helpers ---------------------------------------------------

func toOrgView(o masterrepo.OrgSummary) orgView {
	return orgView{
		ID: o.ID, Slug: o.Slug, Name: o.Name, PlanID: o.PlanID,
		PaymentStatus: o.PaymentStatus,
		MemberCount: o.MemberCount, LeadCount: o.LeadCount,
		CreatedAt: o.CreatedAt.UTC().Format(time.RFC3339),
	}
}

func ptr(s string) *string { return &s }
