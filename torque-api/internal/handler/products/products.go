// Package products serves F11 /api/v1/products endpoints.
//
//   GET    /products           — list (cursor paginated)
//   GET    /products/:id       — detail
//   POST   /products           — create (admin)
//   PATCH  /products/:id       — patch (admin)
//   DELETE /products/:id       — archive soft-delete (admin)
//
// Read is member-accessible (sellers need the catalog). Mutations are
// admin-only (wired by the RequireRole group in main.go).
package products

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/milennials/torque-api/internal/domain"
	"github.com/milennials/torque-api/internal/event"
	"github.com/milennials/torque-api/internal/httpx"
	mw "github.com/milennials/torque-api/internal/httpx/middleware"
	productrepo "github.com/milennials/torque-api/internal/repository/product"
	"github.com/milennials/torque-api/internal/ws"
)

type ReadHandler struct {
	repo *productrepo.Repository
}

type AdminHandler struct {
	repo *productrepo.Repository
	bus  *event.Bus
}

func NewRead(repo *productrepo.Repository) *ReadHandler  { return &ReadHandler{repo: repo} }
func NewAdmin(repo *productrepo.Repository, bus *event.Bus) *AdminHandler {
	return &AdminHandler{repo: repo, bus: bus}
}

func (h *ReadHandler) Routes(r chi.Router) {
	r.Get("/products", h.list)
	r.Get("/products/{id}", h.get)
}
func (h *AdminHandler) Routes(r chi.Router) {
	r.Post("/products", h.create)
	r.Patch("/products/{id}", h.update)
	r.Delete("/products/{id}", h.archive)
}

// -------- views ------------------------------------------------------

type productView struct {
	ID          uuid.UUID       `json:"id"`
	Name        string          `json:"name"`
	Description *string         `json:"description,omitempty"`
	SKU         *string         `json:"sku,omitempty"`
	PriceCents  int64           `json:"price_cents"`
	Currency    string          `json:"currency"`
	IsActive    bool            `json:"is_active"`
	Metadata    json.RawMessage `json:"metadata,omitempty"`
	CreatedBy   *uuid.UUID      `json:"created_by,omitempty"`
	CreatedAt   string          `json:"created_at"`
	UpdatedAt   string          `json:"updated_at"`
}

func toView(p domain.Product) productView {
	var meta json.RawMessage
	if len(p.Metadata) > 0 {
		meta = json.RawMessage(p.Metadata)
	}
	return productView{
		ID: p.ID, Name: p.Name, Description: p.Description, SKU: p.SKU,
		PriceCents: p.PriceCents, Currency: p.Currency, IsActive: p.IsActive,
		Metadata: meta, CreatedBy: p.CreatedBy,
		CreatedAt: p.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt: p.UpdatedAt.UTC().Format(time.RFC3339),
	}
}

// -------- read handlers ---------------------------------------------

func (h *ReadHandler) list(w http.ResponseWriter, r *http.Request) {
	orgID, _ := mw.OrgIDFrom(r.Context())
	q := r.URL.Query()
	opts := productrepo.ListOptions{
		Cursor:     q.Get("cursor"),
		Search:     q.Get("search"),
		ActiveOnly: q.Get("active_only") == "1",
	}
	if ps := q.Get("page_size"); ps != "" {
		if n, err := strconv.Atoi(ps); err == nil {
			opts.Limit = n
		}
	}
	res, err := h.repo.List(r.Context(), orgID, opts)
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_LIST", err.Error())
		return
	}
	out := make([]productView, len(res.Items))
	for i, p := range res.Items {
		out[i] = toView(p)
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{
		"data": out,
		"meta": map[string]any{"next_cursor": nullString(res.NextCursor)},
	})
}

func (h *ReadHandler) get(w http.ResponseWriter, r *http.Request) {
	orgID, _ := mw.OrgIDFrom(r.Context())
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	p, err := h.repo.Get(r.Context(), orgID, id)
	if errors.Is(err, productrepo.ErrNotFound) {
		httpx.WriteError(w, http.StatusNotFound, "NOT_FOUND", "product not found")
		return
	}
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "INTERNAL", "could not load product")
		return
	}
	httpx.WriteJSON(w, http.StatusOK, toView(p))
}

// -------- admin mutations -------------------------------------------

type createReq struct {
	Name        string          `json:"name"`
	Description *string         `json:"description,omitempty"`
	SKU         *string         `json:"sku,omitempty"`
	PriceCents  int64           `json:"price_cents"`
	Currency    string          `json:"currency"`
	Metadata    json.RawMessage `json:"metadata,omitempty"`
}

func (h *AdminHandler) create(w http.ResponseWriter, r *http.Request) {
	sess := mw.MustSession(r.Context())
	orgID, _ := mw.OrgIDFrom(r.Context())
	var body createReq
	if err := httpx.DecodeJSON(r, &body); err != nil {
		decodeError(w, err)
		return
	}
	creator := sess.TeamMemberID
	p, err := h.repo.Create(r.Context(), orgID, productrepo.CreateInput{
		Name: body.Name, Description: body.Description, SKU: body.SKU,
		PriceCents: body.PriceCents, Currency: body.Currency,
		Metadata: body.Metadata, CreatedBy: &creator,
	})
	if err != nil {
		writeProductError(w, err)
		return
	}
	h.publish(orgID, p.ID, "product.created", toView(p))
	httpx.WriteJSON(w, http.StatusCreated, toView(p))
}

type updateReq struct {
	Name        *string         `json:"name,omitempty"`
	Description *string         `json:"description,omitempty"`
	SKU         *string         `json:"sku,omitempty"`
	PriceCents  *int64          `json:"price_cents,omitempty"`
	Currency    *string         `json:"currency,omitempty"`
	IsActive    *bool           `json:"is_active,omitempty"`
	Metadata    json.RawMessage `json:"metadata,omitempty"`
}

func (h *AdminHandler) update(w http.ResponseWriter, r *http.Request) {
	orgID, _ := mw.OrgIDFrom(r.Context())
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	var body updateReq
	if err := httpx.DecodeJSON(r, &body); err != nil {
		decodeError(w, err)
		return
	}
	p, err := h.repo.Update(r.Context(), orgID, id, productrepo.UpdateInput{
		Name: body.Name, Description: body.Description, SKU: body.SKU,
		PriceCents: body.PriceCents, Currency: body.Currency,
		IsActive: body.IsActive, Metadata: body.Metadata,
	})
	if err != nil {
		writeProductError(w, err)
		return
	}
	h.publish(orgID, p.ID, "product.updated", toView(p))
	httpx.WriteJSON(w, http.StatusOK, toView(p))
}

func (h *AdminHandler) archive(w http.ResponseWriter, r *http.Request) {
	orgID, _ := mw.OrgIDFrom(r.Context())
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	if err := h.repo.Archive(r.Context(), orgID, id); err != nil {
		if errors.Is(err, productrepo.ErrNotFound) {
			httpx.WriteError(w, http.StatusNotFound, "NOT_FOUND", "product not found")
			return
		}
		httpx.WriteError(w, http.StatusInternalServerError, "INTERNAL", "could not archive product")
		return
	}
	h.publish(orgID, id, "product.archived", map[string]any{"id": id})
	w.WriteHeader(http.StatusNoContent)
}

// -------- helpers ---------------------------------------------------

func parseID(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_ID", "id must be a uuid")
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

func writeProductError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, productrepo.ErrNotFound):
		httpx.WriteError(w, http.StatusNotFound, "NOT_FOUND", "product not found")
	case errors.Is(err, productrepo.ErrInvalidName):
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_NAME", err.Error())
	case errors.Is(err, productrepo.ErrInvalidPrice):
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_PRICE", err.Error())
	case errors.Is(err, productrepo.ErrInvalidCurrency):
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_CURRENCY", err.Error())
	case errors.Is(err, productrepo.ErrSKUConflict):
		httpx.WriteError(w, http.StatusConflict, "SKU_CONFLICT", err.Error())
	default:
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_PRODUCT", err.Error())
	}
}

func (h *AdminHandler) publish(orgID, id uuid.UUID, evtType string, patch any) {
	h.bus.Publish(ws.Event{
		Type: evtType, TenantID: orgID, EntityType: "product",
		EntityID: &id, Patch: patch, OccurredAt: time.Now().UTC(),
	})
}

func nullString(s string) any {
	if s == "" {
		return nil
	}
	return s
}
