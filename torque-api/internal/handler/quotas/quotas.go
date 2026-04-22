// Package quotas serves the observability surface for org_quotas (S51).
//
//	GET /quotas              — list every (resource, usage, limit) for the tenant (member)
//	GET /quotas/:resource    — single-resource detail (member)
//	PATCH /quotas/:resource  — master-only admin_adjustment / purchased_addons update
//
// Master-only write endpoints are enforced by the router group (see
// cmd/api/main.go). The handler itself is permissive — it relies on
// middleware for RBAC and on repository-level tenant scoping.
package quotas

import (
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/milennials/torque-api/internal/httpx"
	mw "github.com/milennials/torque-api/internal/httpx/middleware"
	quotarepo "github.com/milennials/torque-api/internal/repository/quota"
)

// Handler groups the quota surface.
type Handler struct {
	repo *quotarepo.Repository
}

// New binds the handler.
func New(repo *quotarepo.Repository) *Handler { return &Handler{repo: repo} }

// MemberRoutes mounts the read endpoints — inside the tenant-scoped group.
func (h *Handler) MemberRoutes(r chi.Router) {
	r.Get("/quotas", h.list)
	r.Get("/quotas/{resource}", h.get)
}

// MasterRoutes mounts write endpoints — inside the master-only group.
func (h *Handler) MasterRoutes(r chi.Router) {
	r.Patch("/quotas/{resource}", h.patch)
}

// ---------------- views ---------------------------------------------

type quotaView struct {
	Resource        string `json:"resource"`
	PlanBase        int    `json:"plan_base"`
	PurchasedAddons int    `json:"purchased_addons"`
	AdminAdjustment int    `json:"admin_adjustment"`
	CurrentUsage    int    `json:"current_usage"`
	EffectiveLimit  int    `json:"effective_limit"`
	Remaining       int    `json:"remaining"`
}

func toView(q quotarepo.Quota) quotaView {
	return quotaView{
		Resource: q.ResourceKey, PlanBase: q.PlanBase, PurchasedAddons: q.PurchasedAddons,
		AdminAdjustment: q.AdminAdjustment, CurrentUsage: q.CurrentUsage,
		EffectiveLimit: q.EffectiveLimit, Remaining: q.Remaining,
	}
}

// ---------------- list ----------------------------------------------

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	orgID, _ := mw.OrgIDFrom(r.Context())
	list, err := h.repo.List(r.Context(), orgID)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "INTERNAL", "could not list quotas")
		return
	}
	out := make([]quotaView, len(list))
	for i, q := range list {
		out[i] = toView(q)
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"data": out})
}

// ---------------- get -----------------------------------------------

func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	orgID, _ := mw.OrgIDFrom(r.Context())
	resource := chi.URLParam(r, "resource")
	q, err := h.repo.Get(r.Context(), orgID, resource)
	if errors.Is(err, quotarepo.ErrNotFound) {
		httpx.WriteError(w, http.StatusNotFound, "NOT_FOUND", "quota not configured for this resource")
		return
	}
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "INTERNAL", "could not load quota")
		return
	}
	httpx.WriteJSON(w, http.StatusOK, toView(q))
}

// ---------------- patch (master) ------------------------------------

type patchReq struct {
	AdminAdjustment *int `json:"admin_adjustment,omitempty"`
	PurchasedAddons *int `json:"purchased_addons,omitempty"`
}

func (h *Handler) patch(w http.ResponseWriter, r *http.Request) {
	orgID, _ := mw.OrgIDFrom(r.Context())
	resource := chi.URLParam(r, "resource")
	var body patchReq
	if err := httpx.DecodeJSON(r, &body); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_BODY", "could not parse request")
		return
	}
	if body.AdminAdjustment == nil && body.PurchasedAddons == nil {
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_PATCH",
			"at least one of admin_adjustment, purchased_addons required")
		return
	}
	if body.AdminAdjustment != nil {
		if err := h.repo.SetAdminAdjustment(r.Context(), orgID, resource, *body.AdminAdjustment); err != nil {
			httpx.WriteError(w, http.StatusInternalServerError, "INTERNAL", "could not update admin_adjustment")
			return
		}
	}
	if body.PurchasedAddons != nil {
		if err := h.repo.SetPurchasedAddons(r.Context(), orgID, resource, *body.PurchasedAddons); err != nil {
			httpx.WriteError(w, http.StatusBadRequest, "INVALID_ADDONS", err.Error())
			return
		}
	}
	// Re-read to return the updated state.
	q, err := h.repo.Get(r.Context(), orgID, resource)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "INTERNAL", "could not reload quota")
		return
	}
	httpx.WriteJSON(w, http.StatusOK, toView(q))
}
