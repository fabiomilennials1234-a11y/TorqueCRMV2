// Package master serves F16 /api/v1/master endpoints. Every route is gated
// by RequireMaster; non-master callers get 403 with no body leakage.
//
//   GET  /master/health                      — system-wide snapshot
//   GET  /master/organizations               — list all tenants
//   GET  /master/organizations/:id           — detail
//   POST /master/organizations/:id/impersonate — mint impersonation session
package master

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"

	"github.com/milennials/torque-api/internal/domain"
	"github.com/milennials/torque-api/internal/httpx"
	mw "github.com/milennials/torque-api/internal/httpx/middleware"
	"github.com/milennials/torque-api/internal/repository/audit"
	masterrepo "github.com/milennials/torque-api/internal/repository/master"
	userrepo "github.com/milennials/torque-api/internal/repository/user"
	jwtsvc "github.com/milennials/torque-api/internal/service/jwt"
)

// impersonationResolver is the narrow contract the handler needs from
// masterrepo for the impersonate flow. Defined here so tests can supply a
// fake without spinning up Postgres. *masterrepo.Repository satisfies it.
type impersonationResolver interface {
	ImpersonationTarget(ctx context.Context, orgID uuid.UUID) (uuid.UUID, uuid.UUID, error)
}

// auditAppender is the narrow contract the handler needs from audit for
// the impersonate flow. *audit.Repository satisfies it.
type auditAppender interface {
	Append(ctx context.Context, e audit.Entry) error
}

// jwtIssuer is the narrow contract the handler needs from jwtsvc for
// minting the impersonation token. *jwtsvc.Service satisfies it.
type jwtIssuer interface {
	Issue(userID uuid.UUID, sess domain.Session) (string, time.Time, error)
}

type Handler struct {
	repo         *masterrepo.Repository
	users        *userrepo.Repository
	auditRepo    auditAppender
	impResolver  impersonationResolver
	jwt          jwtIssuer
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
		repo: o.Repo, users: o.Users, auditRepo: o.Audit, impResolver: o.Repo, jwt: o.JWT,
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
	// AccessToken is intentionally NOT surfaced in the body — the session
	// cookie is the sole transport, mirroring /auth/login (ADR-003).
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

// impersonate mints a tenant-scoped JWT for the target org AND sets the
// session cookie atomically on the response. The caller is a master user;
// the audit_log row records both actor_user_id (master) and target_org_id
// (tenant entered) before the token is minted.
//
// Invariants (D062 + ADR-003):
//
//   1. Audit-first: if audit_log insert fails, NO token is minted and NO
//      cookie is set. Fail-closed on master cross-org access.
//   2. The JWT carries the master's UserID as `sub` (so audit downstream
//      still ties actions to the real human) + the TARGET org_id + role
//      `admin` + IsMaster=true so RBAC keeps recognizing the master bypass.
//   3. No refresh token is issued for impersonation. Sessions are
//      deliberately short-lived and non-renewable — on expiry the master
//      re-enters via /auth/me (still authenticated as master in cookies
//      rotated by /auth/refresh... which this flow does NOT change).
//   4. The cookie pattern mirrors /auth/login exactly: Path=/,
//      HttpOnly=true, Secure=cookieSecure, SameSite=Strict, Domain=
//      cookieDomain. Diverging any of these silently breaks session
//      continuity.
//
// End-of-impersonation is handled by /auth/logout (which clears all three
// auth cookies) or by the JWT naturally expiring; there is no dedicated
// "stop impersonating" endpoint in this sprint.
func (h *Handler) impersonate(w http.ResponseWriter, r *http.Request) {
	sess := mw.MustSession(r.Context())
	orgID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_ID", "id must be a uuid")
		return
	}
	memberID, userID, err := h.impResolver.ImpersonationTarget(r.Context(), orgID)
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
		RequestID:      mw.RequestIDFrom(r.Context()),
	}); err != nil {
		log.Error().
			Err(err).
			Str("actor_user_id", sess.UserID.String()).
			Str("target_org_id", orgID.String()).
			Msg("master impersonation refused: audit write failed")
		httpx.WriteError(w, http.StatusInternalServerError, "AUDIT_FAILED",
			"impersonation refused: audit log write failed")
		return
	}

	// Build the impersonated session. `sub` is the master's user_id (so the
	// audit trail downstream ties actions to the real human), but org_id +
	// team_member_id + role are the TARGET tenant's. IsMaster stays true so
	// RBAC keeps honoring master bypass inside the impersonated context.
	impSess := domain.Session{
		UserID:         sess.UserID,
		OrganizationID: orgID,
		TeamMemberID:   memberID,
		Role:           domain.RoleAdmin,
		IsMaster:       true,
		UIMode:         sess.UIMode,
	}

	accessRaw, exp, err := h.jwt.Issue(sess.UserID, impSess)
	if err != nil {
		log.Error().
			Err(err).
			Str("actor_user_id", sess.UserID.String()).
			Str("target_org_id", orgID.String()).
			Msg("master impersonation: jwt mint failed post-audit")
		httpx.WriteError(w, http.StatusInternalServerError, "TOKEN_MINT_FAILED",
			"could not mint impersonation token")
		return
	}

	// Atomic cookie swap. Pattern MUST mirror auth.setSessionCookie exactly
	// (Path, HttpOnly, Secure, SameSite, Domain) — any divergence is a
	// session-continuity bug.
	http.SetCookie(w, &http.Cookie{
		Name:     mw.SessionCookieName,
		Value:    accessRaw,
		Path:     "/",
		Domain:   h.cookieDomain,
		Expires:  exp,
		HttpOnly: true,
		Secure:   h.cookieSecure,
		SameSite: http.SameSiteStrictMode,
	})
	// NOTE: we do NOT rotate the refresh cookie. Impersonation is
	// non-renewable by design; once `exp` passes, the client must call
	// /auth/refresh under the master's own (unchanged) refresh chain,
	// which re-mints a master-context access token — effectively ending
	// the impersonation. This is a deliberate blast-radius reduction.

	log.Info().
		Str("actor_user_id", sess.UserID.String()).
		Str("target_org_id", orgID.String()).
		Str("target_team_member_id", memberID.String()).
		Time("expires_at", exp).
		Msg("master impersonation established")

	httpx.WriteJSON(w, http.StatusOK, impersonateView{
		OrganizationID: orgID,
		TeamMemberID:   memberID,
		UserID:         userID,
		ExpiresAt:      exp.UTC().Format(time.RFC3339),
	})
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
