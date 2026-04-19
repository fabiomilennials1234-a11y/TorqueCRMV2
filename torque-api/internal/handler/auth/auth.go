// Package auth implements /auth/login, /auth/logout, /auth/refresh, /auth/me.
//
// Cookies set here are the contract documented in ADR-003:
//
//   __torque_session  — httpOnly, Secure (prod), SameSite=Strict, short-lived JWT.
//   __torque_refresh  — httpOnly, Secure (prod), SameSite=Strict, opaque, Path=/api/v1/auth.
//   __torque_csrf     — NOT httpOnly, Secure (prod), SameSite=Strict, double-submit companion.
//
// Every response that establishes or rotates a session sets all three.
package auth

import (
	"context"
	"errors"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/milennials/torque-api/internal/domain"
	"github.com/milennials/torque-api/internal/httpx"
	mw "github.com/milennials/torque-api/internal/httpx/middleware"
	refreshrepo "github.com/milennials/torque-api/internal/repository/refresh"
	userrepo "github.com/milennials/torque-api/internal/repository/user"
	"github.com/milennials/torque-api/internal/service/jwt"
	"github.com/milennials/torque-api/internal/service/password"
	"github.com/milennials/torque-api/internal/service/permission"
	"github.com/milennials/torque-api/internal/service/token"
)

// Options wires the handler to its collaborators. All fields are required.
type Options struct {
	Pool            *pgxpool.Pool
	Users           *userrepo.Repository
	Refresh         *refreshrepo.Repository
	JWT             *jwt.Service
	Permission      *permission.Resolver
	RefreshTTL      time.Duration
	CookieSecure    bool   // false only in local dev over plain http
	CookieDomain    string // "" for host-only cookies (recommended in dev)
}

// Handler groups the /auth endpoints.
type Handler struct {
	opts Options
}

// New returns a handler from the given options. Panics on missing dependencies
// — wiring mistakes should fail at boot, not at first request.
func New(opts Options) *Handler {
	if opts.Pool == nil || opts.Users == nil || opts.Refresh == nil ||
		opts.JWT == nil || opts.Permission == nil {
		panic("auth.New: missing dependency")
	}
	if opts.RefreshTTL <= 0 {
		panic("auth.New: RefreshTTL must be > 0")
	}
	return &Handler{opts: opts}
}

// Routes mounts /auth/* on the given router. Call on the pre-auth subrouter;
// login and refresh both predate the session cookie.
func (h *Handler) Routes(r chi.Router) {
	r.Post("/auth/login", h.login)
	r.Post("/auth/refresh", h.refresh)
	r.Post("/auth/logout", h.logout)
	// /auth/me is gated behind RequireAuth — mount on the authenticated subrouter.
}

// MeRoute mounts GET /auth/me on the caller's authenticated subrouter.
func (h *Handler) MeRoute(r chi.Router) {
	r.Get("/auth/me", h.me)
}

// -------- DTOs --------------------------------------------------------

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	// Optional: the frontend MAY send a target org slug when the user has
	// more than one membership. Empty = most recent join wins.
	OrganizationSlug string `json:"organization_slug,omitempty"`
}

type loginResponse struct {
	User         meUser         `json:"user"`
	Organization meOrganization `json:"organization"`
}

type meResponse struct {
	User         meUser                     `json:"user"`
	Organization meOrganization             `json:"organization"`
	Permissions  []domain.FeaturePermission `json:"permissions"`
	CSRFToken    string                     `json:"csrf_token"`
	IsMaster     bool                       `json:"is_master"`
}

type meUser struct {
	ID          uuid.UUID     `json:"id"`
	Email       string        `json:"email"`
	DisplayName string        `json:"display_name"`
	Role        domain.Role   `json:"role"`
	UIMode      domain.UIMode `json:"ui_mode"`
}

type meOrganization struct {
	ID            uuid.UUID `json:"id"`
	Slug          string    `json:"slug"`
	Name          string    `json:"name"`
	PlanID        *string   `json:"plan_id"`
	PaymentStatus string    `json:"payment_status"`
	LogoURL       *string   `json:"logo_url"`
}

// -------- login -------------------------------------------------------

func (h *Handler) login(w http.ResponseWriter, r *http.Request) {
	var body loginRequest
	if err := httpx.DecodeJSON(r, &body); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_BODY", "could not parse request")
		return
	}
	email := strings.ToLower(strings.TrimSpace(body.Email))
	if email == "" || body.Password == "" {
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_CREDENTIALS", "email and password are required")
		return
	}

	ctx := r.Context()
	user, err := h.opts.Users.FindByEmail(ctx, email)
	if err != nil && !errors.Is(err, userrepo.ErrNotFound) {
		httpx.WriteError(w, http.StatusInternalServerError, "INTERNAL", "could not process login")
		return
	}
	// Constant-time-ish uniform response: even if the user does not exist,
	// run a dummy bcrypt verify so response time carries no signal.
	if errors.Is(err, userrepo.ErrNotFound) {
		_ = password.Verify("$2a$12$invalidinvalidinvalidinvalidinvalidinvalidinvalidinvalidinva", body.Password)
		httpx.WriteError(w, http.StatusUnauthorized, "INVALID_CREDENTIALS", "email or password is incorrect")
		return
	}
	if err := password.Verify(user.PasswordHash, body.Password); err != nil {
		httpx.WriteError(w, http.StatusUnauthorized, "INVALID_CREDENTIALS", "email or password is incorrect")
		return
	}

	memberships, err := h.opts.Users.Memberships(ctx, user.ID)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "INTERNAL", "could not load memberships")
		return
	}
	if len(memberships) == 0 {
		// Valid user, zero tenants. Today we reject; tomorrow we could return
		// an onboarding link. Keep the semantics tight until UX is decided.
		httpx.WriteError(w, http.StatusForbidden, "NO_MEMBERSHIP", "user has no active organization")
		return
	}

	chosen := pickMembership(memberships, body.OrganizationSlug)
	isMaster, err := h.opts.Users.IsMaster(ctx, user.ID)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "INTERNAL", "could not resolve master flag")
		return
	}

	org, err := h.opts.Users.Organization(ctx, chosen.OrganizationID)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "INTERNAL", "could not load organization")
		return
	}

	sess := domain.Session{
		UserID:         user.ID,
		OrganizationID: chosen.OrganizationID,
		TeamMemberID:   chosen.TeamMemberID,
		Role:           chosen.Role,
		IsMaster:       isMaster,
		UIMode:         user.UIMode,
	}

	if err := h.establishSession(w, r, sess); err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "INTERNAL", "could not establish session")
		return
	}

	// Best-effort last_login update; do not fail the request on error.
	_ = h.opts.Users.TouchLastLogin(ctx, user.ID)

	httpx.WriteJSON(w, http.StatusOK, loginResponse{
		User: meUser{
			ID:          user.ID,
			Email:       user.Email,
			DisplayName: user.DisplayName,
			Role:        chosen.Role,
			UIMode:      user.UIMode,
		},
		Organization: meOrganization{
			ID:            org.ID,
			Slug:          org.Slug,
			Name:          org.Name,
			PlanID:        org.PlanID,
			PaymentStatus: org.PaymentStatus,
			LogoURL:       org.LogoURL,
		},
	})
}

// pickMembership selects the membership matching slug if provided, otherwise
// the most recently joined one (memberships arrive sorted by joined_at DESC).
func pickMembership(memberships []domain.Membership, slug string) domain.Membership {
	if slug == "" {
		return memberships[0]
	}
	for _, m := range memberships {
		if m.OrgSlug == slug {
			return m
		}
	}
	return memberships[0] // fallback — client asked for a slug we cannot honor
}

// -------- refresh -----------------------------------------------------

func (h *Handler) refresh(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie(refreshCookieName)
	if err != nil || cookie.Value == "" {
		httpx.WriteError(w, http.StatusUnauthorized, "UNAUTHENTICATED", "refresh cookie missing")
		return
	}
	tokenHash := token.Hash(cookie.Value)

	ctx := r.Context()
	current, err := h.opts.Refresh.Lookup(ctx, tokenHash)
	switch {
	case errors.Is(err, refreshrepo.ErrNotFound):
		h.clearAuthCookies(w)
		httpx.WriteError(w, http.StatusUnauthorized, "UNAUTHENTICATED", "refresh token invalid")
		return
	case errors.Is(err, refreshrepo.ErrRevoked):
		h.clearAuthCookies(w)
		httpx.WriteError(w, http.StatusUnauthorized, "UNAUTHENTICATED", "refresh token revoked")
		return
	case errors.Is(err, refreshrepo.ErrExpired):
		h.clearAuthCookies(w)
		httpx.WriteError(w, http.StatusUnauthorized, "UNAUTHENTICATED", "refresh token expired")
		return
	case err != nil:
		httpx.WriteError(w, http.StatusInternalServerError, "INTERNAL", "could not process refresh")
		return
	}

	// Reuse detection: used_at is set only when Rotate previously consumed it.
	// Treat as a compromise signal — revoke the whole lineage.
	if current.UsedAt != nil {
		_ = h.opts.Refresh.RevokeChain(ctx, current.ID, "reuse_detected")
		h.clearAuthCookies(w)
		httpx.WriteError(w, http.StatusUnauthorized, "UNAUTHENTICATED", "refresh token reused — session revoked")
		return
	}

	// Rehydrate membership + master flag; org may have changed payment_status.
	mem, err := h.opts.Users.FindMembership(ctx, current.UserID, current.OrganizationID)
	if err != nil {
		_ = h.opts.Refresh.RevokeChain(ctx, current.ID, "membership_revoked")
		h.clearAuthCookies(w)
		httpx.WriteError(w, http.StatusUnauthorized, "UNAUTHENTICATED", "membership revoked")
		return
	}
	isMaster, err := h.opts.Users.IsMaster(ctx, current.UserID)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "INTERNAL", "could not resolve master flag")
		return
	}
	user, err := h.opts.Users.FindByID(ctx, current.UserID)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "INTERNAL", "could not reload user")
		return
	}

	sess := domain.Session{
		UserID:         user.ID,
		OrganizationID: mem.OrganizationID,
		TeamMemberID:   mem.TeamMemberID,
		Role:           mem.Role,
		IsMaster:       isMaster,
		UIMode:         user.UIMode,
	}

	// Mint new refresh, rotate, set cookies.
	newRaw := token.Generate()
	nextRow := refreshrepo.Token{
		UserID:         sess.UserID,
		OrganizationID: sess.OrganizationID,
		TeamMemberID:   sess.TeamMemberID,
		TokenHash:      token.Hash(newRaw),
		ExpiresAt:      time.Now().Add(h.opts.RefreshTTL),
	}
	fp := fingerprint(r)
	if _, err := h.opts.Refresh.Rotate(ctx, current.ID, nextRow, fp); err != nil {
		if errors.Is(err, refreshrepo.ErrReused) {
			_ = h.opts.Refresh.RevokeChain(ctx, current.ID, "reuse_race")
			h.clearAuthCookies(w)
			httpx.WriteError(w, http.StatusUnauthorized, "UNAUTHENTICATED", "refresh token race lost")
			return
		}
		httpx.WriteError(w, http.StatusInternalServerError, "INTERNAL", "could not rotate refresh")
		return
	}

	accessRaw, accessExp, err := h.opts.JWT.Issue(sess.UserID, sess)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "INTERNAL", "could not mint access token")
		return
	}
	h.setSessionCookie(w, accessRaw, accessExp)
	h.setRefreshCookie(w, newRaw, nextRow.ExpiresAt)
	h.setCSRFCookie(w, token.Generate(), nextRow.ExpiresAt)

	httpx.WriteJSON(w, http.StatusOK, map[string]string{"status": "refreshed"})
}

// -------- logout ------------------------------------------------------

func (h *Handler) logout(w http.ResponseWriter, r *http.Request) {
	// Best-effort revoke; even if the cookie is missing or invalid, we always
	// clear client cookies so the browser reflects the intent.
	if cookie, err := r.Cookie(refreshCookieName); err == nil && cookie.Value != "" {
		if current, lerr := h.opts.Refresh.Lookup(r.Context(), token.Hash(cookie.Value)); lerr == nil {
			_ = h.opts.Refresh.RevokeChain(r.Context(), current.ID, "logout")
		}
	}
	h.clearAuthCookies(w)
	w.WriteHeader(http.StatusNoContent)
}

// -------- me ----------------------------------------------------------

func (h *Handler) me(w http.ResponseWriter, r *http.Request) {
	sess := mw.MustSession(r.Context())

	user, err := h.opts.Users.FindByID(r.Context(), sess.UserID)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "INTERNAL", "could not load user")
		return
	}
	org, err := h.opts.Users.Organization(r.Context(), sess.OrganizationID)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "INTERNAL", "could not load organization")
		return
	}
	perms, err := h.opts.Permission.Bundle(r.Context(), sess)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "INTERNAL", "could not load permissions")
		return
	}
	csrf := ""
	if c, err := r.Cookie(mw.CSRFCookieName); err == nil {
		csrf = c.Value
	}

	httpx.WriteJSON(w, http.StatusOK, meResponse{
		User: meUser{
			ID:          user.ID,
			Email:       user.Email,
			DisplayName: user.DisplayName,
			Role:        sess.Role,
			UIMode:      user.UIMode,
		},
		Organization: meOrganization{
			ID:            org.ID,
			Slug:          org.Slug,
			Name:          org.Name,
			PlanID:        org.PlanID,
			PaymentStatus: org.PaymentStatus,
			LogoURL:       org.LogoURL,
		},
		Permissions: perms,
		CSRFToken:   csrf,
		IsMaster:    sess.IsMaster,
	})
}

// -------- session plumbing -------------------------------------------

const refreshCookieName = "__torque_refresh"

// establishSession mints access + refresh + csrf cookies for a fresh login.
func (h *Handler) establishSession(w http.ResponseWriter, r *http.Request, sess domain.Session) error {
	accessRaw, accessExp, err := h.opts.JWT.Issue(sess.UserID, sess)
	if err != nil {
		return err
	}
	refreshRaw := token.Generate()
	csrfRaw := token.Generate()
	refreshRow := refreshrepo.Token{
		UserID:         sess.UserID,
		OrganizationID: sess.OrganizationID,
		TeamMemberID:   sess.TeamMemberID,
		TokenHash:      token.Hash(refreshRaw),
		ExpiresAt:      time.Now().Add(h.opts.RefreshTTL),
	}
	if _, err := h.opts.Refresh.Issue(context.Background(), refreshRow, fingerprint(r)); err != nil {
		return err
	}

	h.setSessionCookie(w, accessRaw, accessExp)
	h.setRefreshCookie(w, refreshRaw, refreshRow.ExpiresAt)
	h.setCSRFCookie(w, csrfRaw, refreshRow.ExpiresAt)
	return nil
}

func (h *Handler) setSessionCookie(w http.ResponseWriter, v string, exp time.Time) {
	http.SetCookie(w, &http.Cookie{
		Name:     mw.SessionCookieName,
		Value:    v,
		Path:     "/",
		Domain:   h.opts.CookieDomain,
		Expires:  exp,
		HttpOnly: true,
		Secure:   h.opts.CookieSecure,
		SameSite: http.SameSiteStrictMode,
	})
}

func (h *Handler) setRefreshCookie(w http.ResponseWriter, v string, exp time.Time) {
	http.SetCookie(w, &http.Cookie{
		Name:     refreshCookieName,
		Value:    v,
		Path:     "/api/v1/auth", // scoped; do not ship on every request
		Domain:   h.opts.CookieDomain,
		Expires:  exp,
		HttpOnly: true,
		Secure:   h.opts.CookieSecure,
		SameSite: http.SameSiteStrictMode,
	})
}

func (h *Handler) setCSRFCookie(w http.ResponseWriter, v string, exp time.Time) {
	http.SetCookie(w, &http.Cookie{
		Name:     mw.CSRFCookieName,
		Value:    v,
		Path:     "/",
		Domain:   h.opts.CookieDomain,
		Expires:  exp,
		HttpOnly: false, // must be readable by JS (double-submit)
		Secure:   h.opts.CookieSecure,
		SameSite: http.SameSiteStrictMode,
	})
}

func (h *Handler) clearAuthCookies(w http.ResponseWriter) {
	epoch := time.Unix(0, 0)
	for _, c := range []http.Cookie{
		{Name: mw.SessionCookieName, Path: "/", Domain: h.opts.CookieDomain, Expires: epoch, MaxAge: -1, HttpOnly: true, Secure: h.opts.CookieSecure, SameSite: http.SameSiteStrictMode},
		{Name: refreshCookieName, Path: "/api/v1/auth", Domain: h.opts.CookieDomain, Expires: epoch, MaxAge: -1, HttpOnly: true, Secure: h.opts.CookieSecure, SameSite: http.SameSiteStrictMode},
		{Name: mw.CSRFCookieName, Path: "/", Domain: h.opts.CookieDomain, Expires: epoch, MaxAge: -1, HttpOnly: false, Secure: h.opts.CookieSecure, SameSite: http.SameSiteStrictMode},
	} {
		c := c
		http.SetCookie(w, &c)
	}
}

func fingerprint(r *http.Request) refreshrepo.Fingerprint {
	fp := refreshrepo.Fingerprint{UserAgent: r.UserAgent()}
	if host, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
		fp.ClientIP = net.ParseIP(host)
	} else {
		fp.ClientIP = net.ParseIP(r.RemoteAddr)
	}
	return fp
}
