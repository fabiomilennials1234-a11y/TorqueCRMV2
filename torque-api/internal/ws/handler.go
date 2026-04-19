package ws

import (
	"context"
	"errors"
	"net/http"
	"sync"

	"github.com/coder/websocket"
	"github.com/rs/zerolog"

	mw "github.com/milennials/torque-api/internal/httpx/middleware"
)

// Handler upgrades HTTP requests to WebSocket and binds each conn to the hub.
//
// Auth is enforced via the standard Authenticator → RequireAuth chain BEFORE
// the handler runs — by the time we get here, `mw.SessionFrom(ctx)` returns
// the authenticated session (RequireAuth guarantees it).
type Handler struct {
	hub    *Hub
	logger zerolog.Logger

	// allowedOrigins mirrors the CORS whitelist. Empty means "accept same-origin
	// only" (coder/websocket's default). In prod this MUST be set.
	allowedOrigins []string
}

// NewHandler returns a WebSocket upgrade handler bound to the hub.
func NewHandler(hub *Hub, logger zerolog.Logger, allowedOrigins []string) *Handler {
	return &Handler{hub: hub, logger: logger, allowedOrigins: allowedOrigins}
}

// ServeHTTP accepts the WebSocket handshake, registers a Conn with the hub,
// and runs reader/writer until one of them fails.
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	sess, ok := mw.SessionFrom(r.Context())
	if !ok {
		// Defensive — RequireAuth should have rejected.
		http.Error(w, `{"code":"UNAUTHENTICATED","message":"authentication required"}`, http.StatusUnauthorized)
		return
	}

	accept := &websocket.AcceptOptions{
		Subprotocols:   []string{"torque.v1"},
		OriginPatterns: h.allowedOrigins,
	}
	if len(h.allowedOrigins) == 0 {
		// coder/websocket blocks cross-origin by default; be explicit in dev.
		accept.InsecureSkipVerify = false
	}

	ws, err := websocket.Accept(w, r, accept)
	if err != nil {
		h.logger.Warn().Err(err).Msg("ws accept failed")
		return
	}
	// On any return, close the ws; Unregister is called separately.
	defer func() {
		_ = ws.CloseNow()
	}()

	conn := h.hub.Register(sess.OrganizationID)
	defer h.hub.Unregister(conn)

	h.logger.Info().
		Str("conn_id", conn.ID().String()).
		Str("org_id", conn.OrgID().String()).
		Str("user_id", sess.UserID.String()).
		Msg("ws connected")

	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()

	var wg sync.WaitGroup
	wg.Add(2)
	errCh := make(chan error, 2)

	go func() {
		defer wg.Done()
		errCh <- conn.Reader(ctx, ws)
	}()
	go func() {
		defer wg.Done()
		errCh <- conn.Writer(ctx, ws)
	}()

	// First failure cancels the ctx so the other goroutine unblocks.
	firstErr := <-errCh
	cancel()
	wg.Wait()

	if firstErr != nil && !isExpectedClose(firstErr) {
		h.logger.Warn().
			Err(firstErr).
			Str("conn_id", conn.ID().String()).
			Msg("ws closed with error")
	}
}

func isExpectedClose(err error) bool {
	if err == nil || errors.Is(err, context.Canceled) {
		return true
	}
	if websocket.CloseStatus(err) != -1 {
		return true
	}
	return false
}
