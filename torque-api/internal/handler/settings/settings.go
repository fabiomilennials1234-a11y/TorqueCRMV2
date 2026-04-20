// Package settings serves F15 /api/v1 settings endpoints.
//
//   GET    /organization              — member-accessible profile
//   PATCH  /organization              — admin-only profile patch
//   GET    /webhooks                  — member-accessible list
//   POST   /webhooks                  — admin-only create
//   PATCH  /webhooks/:id              — admin-only patch
//   DELETE /webhooks/:id              — admin-only delete
//   GET    /me/notifications          — own prefs
//   PUT    /me/notifications          — upsert own pref
package settings

import (
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/milennials/torque-api/internal/httpx"
	mw "github.com/milennials/torque-api/internal/httpx/middleware"
	settingsrepo "github.com/milennials/torque-api/internal/repository/settings"
)

type ReadHandler struct {
	repo *settingsrepo.Repository
}

type AdminHandler struct {
	repo *settingsrepo.Repository
}

type MeHandler struct {
	repo *settingsrepo.Repository
}

func NewRead(repo *settingsrepo.Repository) *ReadHandler   { return &ReadHandler{repo: repo} }
func NewAdmin(repo *settingsrepo.Repository) *AdminHandler { return &AdminHandler{repo: repo} }
func NewMe(repo *settingsrepo.Repository) *MeHandler       { return &MeHandler{repo: repo} }

func (h *ReadHandler) Routes(r chi.Router) {
	r.Get("/organization", h.getOrg)
	r.Get("/webhooks", h.listWebhooks)
}

func (h *AdminHandler) Routes(r chi.Router) {
	r.Patch("/organization", h.updateOrg)
	r.Post("/webhooks", h.createWebhook)
	r.Patch("/webhooks/{id}", h.updateWebhook)
	r.Delete("/webhooks/{id}", h.deleteWebhook)
}

func (h *MeHandler) Routes(r chi.Router) {
	r.Get("/me/notifications", h.listPrefs)
	r.Put("/me/notifications", h.setPref)
}

// -------- views ------------------------------------------------------

type orgView struct {
	ID        uuid.UUID `json:"id"`
	Slug      string    `json:"slug"`
	Name      string    `json:"name"`
	LegalName *string   `json:"legal_name,omitempty"`
	CNPJ      *string   `json:"cnpj,omitempty"`
	Timezone  *string   `json:"timezone,omitempty"`
	LogoURL   *string   `json:"logo_url,omitempty"`
	UpdatedAt string    `json:"updated_at"`
}

type webhookView struct {
	ID            uuid.UUID `json:"id"`
	URL           string    `json:"url"`
	Description   *string   `json:"description,omitempty"`
	EventTypes    []string  `json:"event_types"`
	IsActive      bool      `json:"is_active"`
	LastSuccessAt *string   `json:"last_success_at,omitempty"`
	LastFailureAt *string   `json:"last_failure_at,omitempty"`
	LastError     *string   `json:"last_error,omitempty"`
	CreatedAt     string    `json:"created_at"`
}

type prefView struct {
	Channel string `json:"channel"`
	Topic   string `json:"topic"`
	Enabled bool   `json:"enabled"`
}

func toOrgView(o settingsrepo.Organization) orgView {
	return orgView{
		ID: o.ID, Slug: o.Slug, Name: o.Name,
		LegalName: o.LegalName, CNPJ: o.CNPJ, Timezone: o.Timezone, LogoURL: o.LogoURL,
		UpdatedAt: o.UpdatedAt.UTC().Format(time.RFC3339),
	}
}

func toWebhookView(w settingsrepo.WebhookEndpoint) webhookView {
	v := webhookView{
		ID: w.ID, URL: w.URL, Description: w.Description,
		EventTypes: w.EventTypes, IsActive: w.IsActive, LastError: w.LastError,
		CreatedAt: w.CreatedAt.UTC().Format(time.RFC3339),
	}
	if w.LastSuccessAt != nil {
		s := w.LastSuccessAt.UTC().Format(time.RFC3339)
		v.LastSuccessAt = &s
	}
	if w.LastFailureAt != nil {
		s := w.LastFailureAt.UTC().Format(time.RFC3339)
		v.LastFailureAt = &s
	}
	return v
}

// -------- read handlers ---------------------------------------------

func (h *ReadHandler) getOrg(w http.ResponseWriter, r *http.Request) {
	orgID, _ := mw.OrgIDFrom(r.Context())
	o, err := h.repo.GetOrganization(r.Context(), orgID)
	if errors.Is(err, settingsrepo.ErrNotFound) {
		httpx.WriteError(w, http.StatusNotFound, "NOT_FOUND", "organization not found")
		return
	}
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "INTERNAL", "could not load organization")
		return
	}
	httpx.WriteJSON(w, http.StatusOK, toOrgView(o))
}

func (h *ReadHandler) listWebhooks(w http.ResponseWriter, r *http.Request) {
	orgID, _ := mw.OrgIDFrom(r.Context())
	list, err := h.repo.ListWebhooks(r.Context(), orgID)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "INTERNAL", "could not list webhooks")
		return
	}
	out := make([]webhookView, len(list))
	for i, w := range list {
		out[i] = toWebhookView(w)
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"data": out})
}

// -------- admin mutations -------------------------------------------

type updateOrgReq struct {
	Name      *string `json:"name,omitempty"`
	LegalName *string `json:"legal_name,omitempty"`
	CNPJ      *string `json:"cnpj,omitempty"`
	Timezone  *string `json:"timezone,omitempty"`
	LogoURL   *string `json:"logo_url,omitempty"`
}

func (h *AdminHandler) updateOrg(w http.ResponseWriter, r *http.Request) {
	orgID, _ := mw.OrgIDFrom(r.Context())
	var body updateOrgReq
	if err := httpx.DecodeJSON(r, &body); err != nil {
		decodeErr(w, err)
		return
	}
	o, err := h.repo.UpdateOrganization(r.Context(), orgID, settingsrepo.UpdateOrganizationInput{
		Name: body.Name, LegalName: body.LegalName, CNPJ: body.CNPJ,
		Timezone: body.Timezone, LogoURL: body.LogoURL,
	})
	if errors.Is(err, settingsrepo.ErrNotFound) {
		httpx.WriteError(w, http.StatusNotFound, "NOT_FOUND", "organization not found")
		return
	}
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_ORG", err.Error())
		return
	}
	httpx.WriteJSON(w, http.StatusOK, toOrgView(o))
}

type createWebhookReq struct {
	URL         string   `json:"url"`
	Description *string  `json:"description,omitempty"`
	EventTypes  []string `json:"event_types"`
	Secret      string   `json:"secret"`
}

func (h *AdminHandler) createWebhook(w http.ResponseWriter, r *http.Request) {
	sess := mw.MustSession(r.Context())
	orgID, _ := mw.OrgIDFrom(r.Context())
	var body createWebhookReq
	if err := httpx.DecodeJSON(r, &body); err != nil {
		decodeErr(w, err)
		return
	}
	wep, err := h.repo.CreateWebhook(r.Context(), orgID, settingsrepo.CreateWebhookInput{
		URL: body.URL, Description: body.Description, EventTypes: body.EventTypes,
		Secret: body.Secret, CreatedBy: sess.TeamMemberID,
	})
	if errors.Is(err, settingsrepo.ErrURLScheme) {
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_URL", err.Error())
		return
	}
	if errors.Is(err, settingsrepo.ErrURLTaken) {
		httpx.WriteError(w, http.StatusConflict, "URL_TAKEN", err.Error())
		return
	}
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_WEBHOOK", err.Error())
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, toWebhookView(wep))
}

type updateWebhookReq struct {
	Description *string   `json:"description,omitempty"`
	EventTypes  *[]string `json:"event_types,omitempty"`
	IsActive    *bool     `json:"is_active,omitempty"`
}

func (h *AdminHandler) updateWebhook(w http.ResponseWriter, r *http.Request) {
	orgID, _ := mw.OrgIDFrom(r.Context())
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_ID", "id must be a uuid")
		return
	}
	var body updateWebhookReq
	if err := httpx.DecodeJSON(r, &body); err != nil {
		decodeErr(w, err)
		return
	}
	wep, err := h.repo.UpdateWebhook(r.Context(), orgID, id, settingsrepo.UpdateWebhookInput{
		Description: body.Description, EventTypes: body.EventTypes, IsActive: body.IsActive,
	})
	if errors.Is(err, settingsrepo.ErrNotFound) {
		httpx.WriteError(w, http.StatusNotFound, "NOT_FOUND", "webhook not found")
		return
	}
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_WEBHOOK", err.Error())
		return
	}
	httpx.WriteJSON(w, http.StatusOK, toWebhookView(wep))
}

func (h *AdminHandler) deleteWebhook(w http.ResponseWriter, r *http.Request) {
	orgID, _ := mw.OrgIDFrom(r.Context())
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_ID", "id must be a uuid")
		return
	}
	if err := h.repo.DeleteWebhook(r.Context(), orgID, id); err != nil {
		if errors.Is(err, settingsrepo.ErrNotFound) {
			httpx.WriteError(w, http.StatusNotFound, "NOT_FOUND", "webhook not found")
			return
		}
		httpx.WriteError(w, http.StatusInternalServerError, "INTERNAL", "could not delete webhook")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// -------- me/notifications ------------------------------------------

func (h *MeHandler) listPrefs(w http.ResponseWriter, r *http.Request) {
	sess := mw.MustSession(r.Context())
	list, err := h.repo.ListPreferences(r.Context(), sess.TeamMemberID)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "INTERNAL", "could not list preferences")
		return
	}
	out := make([]prefView, len(list))
	for i, p := range list {
		out[i] = prefView{Channel: p.Channel, Topic: p.Topic, Enabled: p.Enabled}
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"data": out})
}

type setPrefReq struct {
	Channel string `json:"channel"`
	Topic   string `json:"topic"`
	Enabled bool   `json:"enabled"`
}

func (h *MeHandler) setPref(w http.ResponseWriter, r *http.Request) {
	sess := mw.MustSession(r.Context())
	var body setPrefReq
	if err := httpx.DecodeJSON(r, &body); err != nil {
		decodeErr(w, err)
		return
	}
	if err := h.repo.SetPreference(r.Context(), sess.TeamMemberID, body.Channel, body.Topic, body.Enabled); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_PREF", err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// -------- helpers ---------------------------------------------------

func decodeErr(w http.ResponseWriter, err error) {
	if httpx.IsBodyTooLarge(err) {
		httpx.WriteError(w, http.StatusRequestEntityTooLarge, "BODY_TOO_LARGE", "body exceeds 1 MiB")
		return
	}
	httpx.WriteError(w, http.StatusBadRequest, "INVALID_BODY", "could not parse request")
}
