// Package members serves F10 Equipe /api/v1/members endpoints.
//
//   GET    /members                              — list
//   POST   /members                              — add (existing user by email)
//   GET    /members/:id                          — detail
//   PATCH  /members/:id                          — patch (display_name, role, avatar_url, is_active)
//   DELETE /members/:id                          — deactivate (soft)
//   GET    /members/:id/permissions              — list explicit overrides
//   PUT    /members/:id/permissions/:featureKey  — upsert override
//   DELETE /members/:id/permissions/:featureKey  — revert to default
//
// Read surfaces are member-accessible (anyone on the org sees the team).
// All mutations are admin-only (wired by the RequireRole group in main.go).
package members

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/milennials/torque-api/internal/domain"
	"github.com/milennials/torque-api/internal/event"
	"github.com/milennials/torque-api/internal/httpx"
	mw "github.com/milennials/torque-api/internal/httpx/middleware"
	memberrepo "github.com/milennials/torque-api/internal/repository/member"
	quotarepo "github.com/milennials/torque-api/internal/repository/quota"
	"github.com/milennials/torque-api/internal/ws"
)

// ReadHandler serves the read-only, member-accessible surfaces.
type ReadHandler struct {
	repo *memberrepo.Repository
}

// quotaGate is the narrow contract the admin handler needs from the
// quota repository. Kept local (not in the repo package) so tests can
// drop in a fake without spinning a pgxpool, and so the production
// `*quotarepo.Repository` implicitly satisfies it without any casting.
// The underlying middleware uses its own `QuotaReader` interface — that
// one is a strict subset of this one.
type quotaGate interface {
	mw.QuotaReader
	IncrementUsage(ctx context.Context, orgID uuid.UUID, resource string, delta int) (int, error)
}

// AdminHandler serves the admin-only mutations + permission management.
type AdminHandler struct {
	repo  *memberrepo.Repository
	bus   *event.Bus
	quota quotaGate
}

// NewRead binds a read handler.
func NewRead(repo *memberrepo.Repository) *ReadHandler {
	return &ReadHandler{repo: repo}
}

// NewAdmin binds the admin handler.
func NewAdmin(repo *memberrepo.Repository, bus *event.Bus) *AdminHandler {
	return &AdminHandler{repo: repo, bus: bus}
}

// WithQuota attaches the quota repository so POST /members enforces the
// org's team_members cap (S52). nil is accepted — the handler collapses
// back to unbounded add/deactivate (matches pre-S52 behavior for test
// fixtures that don't seed plan_quotas).
func (h *AdminHandler) WithQuota(q *quotarepo.Repository) *AdminHandler {
	if q == nil {
		h.quota = nil
		return h
	}
	h.quota = q
	return h
}

// ReadRoutes mounts read endpoints under the tenant-scoped group.
func (h *ReadHandler) Routes(r chi.Router) {
	r.Get("/members", h.list)
	r.Get("/members/{id}", h.get)
	r.Get("/members/{id}/permissions", h.listOverrides)
}

// AdminRoutes mounts admin endpoints under the admin-only group.
func (h *AdminHandler) Routes(r chi.Router) {
	// S52 — POST /members sits behind the quota gate when wired. The
	// handler also increments usage post-success and decrements on
	// deactivate/update-is_active transitions so the counter tracks the
	// active-member headcount the plan caps enforce.
	if h.quota != nil {
		r.With(mw.RequireQuota(h.quota, quotarepo.ResourceTeamMembers)).Post("/members", h.add)
	} else {
		r.Post("/members", h.add)
	}
	r.Patch("/members/{id}", h.update)
	r.Delete("/members/{id}", h.deactivate)
	r.Put("/members/{id}/permissions/{featureKey}", h.setOverride)
	r.Delete("/members/{id}/permissions/{featureKey}", h.clearOverride)
}

// -------- views ------------------------------------------------------

type memberView struct {
	ID            uuid.UUID  `json:"id"`
	UserID        uuid.UUID  `json:"user_id"`
	Email         string     `json:"email"`
	DisplayName   string     `json:"display_name"`
	Role          string     `json:"role"`
	AvatarURL     *string    `json:"avatar_url,omitempty"`
	IsActive      bool       `json:"is_active"`
	InvitedBy     *uuid.UUID `json:"invited_by,omitempty"`
	InvitedAt     *string    `json:"invited_at,omitempty"`
	JoinedAt      string     `json:"joined_at"`
	DeactivatedAt *string    `json:"deactivated_at,omitempty"`
}

func toView(m domain.TeamMember) memberView {
	v := memberView{
		ID: m.ID, UserID: m.UserID, Email: m.Email,
		DisplayName: m.DisplayName, Role: string(m.Role), AvatarURL: m.AvatarURL,
		IsActive: m.IsActive, InvitedBy: m.InvitedBy,
		JoinedAt: m.JoinedAt.UTC().Format(time.RFC3339),
	}
	if m.InvitedAt != nil {
		s := m.InvitedAt.UTC().Format(time.RFC3339)
		v.InvitedAt = &s
	}
	if m.DeactivatedAt != nil {
		s := m.DeactivatedAt.UTC().Format(time.RFC3339)
		v.DeactivatedAt = &s
	}
	return v
}

// -------- read handlers ---------------------------------------------

func (h *ReadHandler) list(w http.ResponseWriter, r *http.Request) {
	orgID, _ := mw.OrgIDFrom(r.Context())
	includeInactive := r.URL.Query().Get("include_inactive") == "1"
	list, err := h.repo.List(r.Context(), orgID, includeInactive)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "INTERNAL", "could not list members")
		return
	}
	out := make([]memberView, len(list))
	for i, m := range list {
		out[i] = toView(m)
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"data": out})
}

func (h *ReadHandler) get(w http.ResponseWriter, r *http.Request) {
	orgID, _ := mw.OrgIDFrom(r.Context())
	id, ok := parseID(w, r, "id")
	if !ok {
		return
	}
	m, err := h.repo.Get(r.Context(), orgID, id)
	if errors.Is(err, memberrepo.ErrNotFound) {
		httpx.WriteError(w, http.StatusNotFound, "NOT_FOUND", "member not found")
		return
	}
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "INTERNAL", "could not load member")
		return
	}
	httpx.WriteJSON(w, http.StatusOK, toView(m))
}

func (h *ReadHandler) listOverrides(w http.ResponseWriter, r *http.Request) {
	orgID, _ := mw.OrgIDFrom(r.Context())
	id, ok := parseID(w, r, "id")
	if !ok {
		return
	}
	rows, err := h.repo.ListOverrides(r.Context(), orgID, id)
	if errors.Is(err, memberrepo.ErrNotFound) {
		httpx.WriteError(w, http.StatusNotFound, "NOT_FOUND", "member not found")
		return
	}
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "INTERNAL", "could not list overrides")
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"data": rows})
}

// -------- admin mutations -------------------------------------------

type addReq struct {
	Email       string `json:"email"`
	DisplayName string `json:"display_name"`
	Role        string `json:"role"`
}

func (h *AdminHandler) add(w http.ResponseWriter, r *http.Request) {
	sess := mw.MustSession(r.Context())
	orgID, _ := mw.OrgIDFrom(r.Context())
	var body addReq
	if err := httpx.DecodeJSON(r, &body); err != nil {
		decodeError(w, err)
		return
	}
	role := domain.Role(body.Role)
	if !role.IsValid() {
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_ROLE", "role must be admin or membro")
		return
	}
	m, err := h.repo.AddByEmail(r.Context(), orgID, memberrepo.AddByEmailInput{
		Email: body.Email, DisplayName: body.DisplayName, Role: role, InvitedBy: sess.TeamMemberID,
	})
	if errors.Is(err, memberrepo.ErrUserNotFound) {
		httpx.WriteError(w, http.StatusNotFound, "USER_NOT_FOUND", "no active user with this email")
		return
	}
	if errors.Is(err, memberrepo.ErrAlreadyMember) {
		httpx.WriteError(w, http.StatusConflict, "ALREADY_MEMBER", "user is already a member of this org")
		return
	}
	if errors.Is(err, memberrepo.ErrInvalidRole) {
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_ROLE", "role must be admin or membro")
		return
	}
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_MEMBER", err.Error())
		return
	}
	// S52 — increment usage AFTER the create succeeds. Handler failure
	// paths above do not increment. Fire-and-forget: a drift-free usage
	// recount job is the canonical authority for divergence.
	if h.quota != nil {
		_, _ = h.quota.IncrementUsage(r.Context(), orgID, quotarepo.ResourceTeamMembers, 1)
	}
	h.publish(orgID, m.ID, "member.created", toView(m))
	httpx.WriteJSON(w, http.StatusCreated, toView(m))
}

type updateReq struct {
	DisplayName *string `json:"display_name,omitempty"`
	Role        *string `json:"role,omitempty"`
	AvatarURL   *string `json:"avatar_url,omitempty"`
	IsActive    *bool   `json:"is_active,omitempty"`
}

func (h *AdminHandler) update(w http.ResponseWriter, r *http.Request) {
	sess := mw.MustSession(r.Context())
	orgID, _ := mw.OrgIDFrom(r.Context())
	id, ok := parseID(w, r, "id")
	if !ok {
		return
	}
	var body updateReq
	if err := httpx.DecodeJSON(r, &body); err != nil {
		decodeError(w, err)
		return
	}
	in := memberrepo.UpdateInput{
		DisplayName: body.DisplayName, AvatarURL: body.AvatarURL, IsActive: body.IsActive,
	}
	if body.Role != nil {
		role := domain.Role(*body.Role)
		if !role.IsValid() {
			httpx.WriteError(w, http.StatusBadRequest, "INVALID_ROLE", "role must be admin or membro")
			return
		}
		// Admin cannot demote their own role — prevents an org lockout where
		// the last admin strips themselves. Another admin can still do it.
		if id == sess.TeamMemberID && role != domain.RoleAdmin {
			httpx.WriteError(w, http.StatusForbidden, "SELF_DEMOTE", "admins cannot demote themselves")
			return
		}
		in.Role = &role
	}
	// Prevent self-deactivation for the same reason.
	if body.IsActive != nil && !*body.IsActive && id == sess.TeamMemberID {
		httpx.WriteError(w, http.StatusForbidden, "SELF_DEACTIVATE", "admins cannot deactivate themselves")
		return
	}
	// S52 — the team_members quota tracks *active* seat count. A PATCH
	// that flips is_active changes that count, so we snapshot the prior
	// value before Update so we only bump on a real transition.
	var priorActive *bool
	if h.quota != nil && body.IsActive != nil {
		prior, gerr := h.repo.Get(r.Context(), orgID, id)
		if gerr == nil {
			v := prior.IsActive
			priorActive = &v
		}
	}
	m, err := h.repo.Update(r.Context(), orgID, id, in)
	if errors.Is(err, memberrepo.ErrNotFound) {
		httpx.WriteError(w, http.StatusNotFound, "NOT_FOUND", "member not found")
		return
	}
	if errors.Is(err, memberrepo.ErrInvalidRole) {
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_ROLE", "role must be admin or membro")
		return
	}
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "INTERNAL", "could not update member")
		return
	}
	// S52 — settle the quota delta only when is_active actually flipped.
	// Idempotent PATCHes (same is_active echoed back) leave the counter
	// untouched.
	if h.quota != nil && priorActive != nil && body.IsActive != nil && *priorActive != *body.IsActive {
		delta := -1
		if *body.IsActive {
			delta = 1
		}
		_, _ = h.quota.IncrementUsage(r.Context(), orgID, quotarepo.ResourceTeamMembers, delta)
	}
	h.publish(orgID, m.ID, "member.updated", toView(m))
	httpx.WriteJSON(w, http.StatusOK, toView(m))
}

func (h *AdminHandler) deactivate(w http.ResponseWriter, r *http.Request) {
	sess := mw.MustSession(r.Context())
	orgID, _ := mw.OrgIDFrom(r.Context())
	id, ok := parseID(w, r, "id")
	if !ok {
		return
	}
	if id == sess.TeamMemberID {
		httpx.WriteError(w, http.StatusForbidden, "SELF_DEACTIVATE", "admins cannot deactivate themselves")
		return
	}
	if err := h.repo.Deactivate(r.Context(), orgID, id); err != nil {
		if errors.Is(err, memberrepo.ErrNotFound) {
			httpx.WriteError(w, http.StatusNotFound, "NOT_FOUND", "member not found")
			return
		}
		httpx.WriteError(w, http.StatusInternalServerError, "INTERNAL", "could not deactivate member")
		return
	}
	// S52 — reclaim the seat. Deactivate is guarded by `is_active=true`
	// at the repo (line 235), so ErrNotFound above covers both
	// missing-member and already-deactivated — the increment only
	// fires when the row actually flipped.
	if h.quota != nil {
		_, _ = h.quota.IncrementUsage(r.Context(), orgID, quotarepo.ResourceTeamMembers, -1)
	}
	h.publish(orgID, id, "member.deactivated", map[string]any{"id": id})
	w.WriteHeader(http.StatusNoContent)
}

type overrideReq struct {
	Value bool `json:"value"`
}

func (h *AdminHandler) setOverride(w http.ResponseWriter, r *http.Request) {
	sess := mw.MustSession(r.Context())
	orgID, _ := mw.OrgIDFrom(r.Context())
	id, ok := parseID(w, r, "id")
	if !ok {
		return
	}
	featureKey := chi.URLParam(r, "featureKey")
	if featureKey == "" {
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_KEY", "feature key is required")
		return
	}
	var body overrideReq
	if err := httpx.DecodeJSON(r, &body); err != nil {
		decodeError(w, err)
		return
	}
	err := h.repo.SetOverride(r.Context(), orgID, id, featureKey, body.Value, sess.TeamMemberID)
	if errors.Is(err, memberrepo.ErrNotFound) {
		httpx.WriteError(w, http.StatusNotFound, "NOT_FOUND", "member not found")
		return
	}
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_KEY", err.Error())
		return
	}
	h.publish(orgID, id, "member.permission_changed", map[string]any{
		"member_id": id, "feature_key": featureKey, "value": body.Value,
	})
	w.WriteHeader(http.StatusNoContent)
}

func (h *AdminHandler) clearOverride(w http.ResponseWriter, r *http.Request) {
	orgID, _ := mw.OrgIDFrom(r.Context())
	id, ok := parseID(w, r, "id")
	if !ok {
		return
	}
	featureKey := chi.URLParam(r, "featureKey")
	if featureKey == "" {
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_KEY", "feature key is required")
		return
	}
	err := h.repo.ClearOverride(r.Context(), orgID, id, featureKey)
	if errors.Is(err, memberrepo.ErrNotFound) {
		httpx.WriteError(w, http.StatusNotFound, "NOT_FOUND", "member not found")
		return
	}
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "INTERNAL", "could not clear override")
		return
	}
	h.publish(orgID, id, "member.permission_changed", map[string]any{
		"member_id": id, "feature_key": featureKey, "cleared": true,
	})
	w.WriteHeader(http.StatusNoContent)
}

// -------- helpers ---------------------------------------------------

func parseID(w http.ResponseWriter, r *http.Request, name string) (uuid.UUID, bool) {
	id, err := uuid.Parse(chi.URLParam(r, name))
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_ID", name+" must be a uuid")
		return uuid.Nil, false
	}
	return id, true
}

func decodeError(w http.ResponseWriter, err error) {
	if httpx.IsBodyTooLarge(err) {
		httpx.WriteError(w, http.StatusRequestEntityTooLarge, "BODY_TOO_LARGE", "body exceeds 1 MiB")
		return
	}
	httpx.WriteError(w, http.StatusBadRequest, "INVALID_BODY", "could not parse request")
}

func (h *AdminHandler) publish(orgID, id uuid.UUID, evtType string, patch any) {
	h.bus.Publish(ws.Event{
		Type: evtType, TenantID: orgID, EntityType: "member",
		EntityID: &id, Patch: patch, OccurredAt: time.Now().UTC(),
	})
}
