package ws

import (
	"context"
	"encoding/json"
	"sync"
	"sync/atomic"
	"time"

	"github.com/coder/websocket"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

// HubConfig tunes the hub. Zero-value uses safe defaults.
type HubConfig struct {
	// PingInterval bounds the time between server → client pings.
	// The client is expected to respond with a pong; otherwise the conn is
	// torn down by the next PongTimeout.
	PingInterval time.Duration
	// PongTimeout is the maximum silence before we declare the peer dead.
	PongTimeout time.Duration
	// OutboundBuffer is the per-connection write queue. Events beyond this
	// are dropped (DropOldest) — real-time UI prefers fresh state.
	OutboundBuffer int
	// MaxMessageSize caps inbound frames (clients do not send much, but a
	// compromised peer should not be able to DoS the server).
	MaxMessageSize int64
}

// DefaultHubConfig returns production defaults.
func DefaultHubConfig() HubConfig {
	return HubConfig{
		PingInterval:   30 * time.Second,
		PongTimeout:    60 * time.Second,
		OutboundBuffer: 64,
		MaxMessageSize: 64 * 1024, // 64 KB
	}
}

// Hub multiplexes events to tenant-scoped connections.
//
// It is the single source of truth for "who is connected". Register returns a
// Connection handle the caller drives — the hub itself does not own the
// transport, only the routing.
type Hub struct {
	cfg    HubConfig
	logger zerolog.Logger

	mu    sync.RWMutex
	byOrg map[uuid.UUID]map[uuid.UUID]*Conn
}

// NewHub returns a Hub ready to Register connections.
func NewHub(cfg HubConfig, logger zerolog.Logger) *Hub {
	if cfg.PingInterval == 0 {
		cfg = DefaultHubConfig()
	}
	return &Hub{
		cfg:    cfg,
		logger: logger,
		byOrg:  make(map[uuid.UUID]map[uuid.UUID]*Conn),
	}
}

// Conn is the hub's view of one live connection. The transport lives in the
// caller's handler (see handler.go).
type Conn struct {
	id  uuid.UUID
	org uuid.UUID
	hub *Hub

	// out is written by Publish/Broadcast and drained by the writer goroutine
	// attached to the websocket.
	out chan Event

	// lastSeen is updated on every inbound frame (pong, ping, message). The
	// reaper checks this against cfg.PongTimeout.
	lastSeen atomic.Int64 // unix nanos
}

// Register allocates a Conn bound to `orgID`. The caller owns the returned
// Conn and must invoke Unregister on disconnect (deferred).
func (h *Hub) Register(orgID uuid.UUID) *Conn {
	c := &Conn{
		id:  uuid.New(),
		org: orgID,
		hub: h,
		out: make(chan Event, h.cfg.OutboundBuffer),
	}
	c.lastSeen.Store(time.Now().UnixNano())

	h.mu.Lock()
	if _, ok := h.byOrg[orgID]; !ok {
		h.byOrg[orgID] = make(map[uuid.UUID]*Conn)
	}
	h.byOrg[orgID][c.id] = c
	h.mu.Unlock()
	return c
}

// Unregister removes the conn from routing and closes its outbound channel.
// Safe to call twice; subsequent sends are no-ops.
func (h *Hub) Unregister(c *Conn) {
	h.mu.Lock()
	if conns, ok := h.byOrg[c.org]; ok {
		delete(conns, c.id)
		if len(conns) == 0 {
			delete(h.byOrg, c.org)
		}
	}
	h.mu.Unlock()

	// Close outbound chan under a once-guard.
	defer func() { _ = recover() }()
	close(c.out)
}

// Broadcast fans an event to every connection of the matching tenant.
// Non-blocking: a slow client drops the event (DropOldest — UI favors
// freshness).
func (h *Hub) Broadcast(evt Event) {
	if evt.TenantID == uuid.Nil {
		h.logger.Warn().Str("type", evt.Type).Msg("event missing tenant_id; dropped")
		return
	}
	h.mu.RLock()
	conns := make([]*Conn, 0, 4)
	if m, ok := h.byOrg[evt.TenantID]; ok {
		for _, c := range m {
			conns = append(conns, c)
		}
	}
	h.mu.RUnlock()

	for _, c := range conns {
		c.send(evt)
	}
}

// send writes to the outbound channel with DropOldest semantics — we evict
// the stalest event rather than block the hub.
func (c *Conn) send(evt Event) {
	for {
		select {
		case c.out <- evt:
			return
		default:
			select {
			case <-c.out:
			default:
				return
			}
		}
	}
}

// Writer drains c.out and sends each event as a JSON text message. Also pings
// at cfg.PingInterval. Exits cleanly when ctx is cancelled or ws closes.
//
// Returns on non-recoverable error; the caller closes the websocket.
func (c *Conn) Writer(ctx context.Context, ws *websocket.Conn) error {
	ticker := time.NewTicker(c.hub.cfg.PingInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case evt, ok := <-c.out:
			if !ok {
				return nil
			}
			buf, err := json.Marshal(evt)
			if err != nil {
				c.hub.logger.Warn().Err(err).Str("type", evt.Type).Msg("marshal event")
				continue
			}
			wctx, cancel := context.WithTimeout(ctx, 5*time.Second)
			err = ws.Write(wctx, websocket.MessageText, buf)
			cancel()
			if err != nil {
				return err
			}
		case <-ticker.C:
			pctx, cancel := context.WithTimeout(ctx, 5*time.Second)
			err := ws.Ping(pctx)
			cancel()
			if err != nil {
				return err
			}
			// Liveness check: if we have not seen the peer inside PongTimeout,
			// close the conn.
			if c.silentFor() > c.hub.cfg.PongTimeout {
				return errSilent
			}
		}
	}
}

// Reader drains inbound frames, updating lastSeen on each, and exits on the
// first error or close. Inbound content is discarded by design — the client
// never sends events; this goroutine exists to collect pongs and detect dead
// peers.
func (c *Conn) Reader(ctx context.Context, ws *websocket.Conn) error {
	ws.SetReadLimit(c.hub.cfg.MaxMessageSize)
	for {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		// coder/websocket handles pings/pongs below Read, but Read is still
		// the liveness heartbeat — every received frame refreshes lastSeen.
		_, _, err := ws.Read(ctx)
		if err != nil {
			return err
		}
		c.lastSeen.Store(time.Now().UnixNano())
	}
}

// OrgID exposes the conn's tenant (for logs; do NOT route from outside).
func (c *Conn) OrgID() uuid.UUID { return c.org }

// ID exposes the conn id (for logs).
func (c *Conn) ID() uuid.UUID { return c.id }

func (c *Conn) silentFor() time.Duration {
	last := c.lastSeen.Load()
	if last == 0 {
		return 0
	}
	return time.Since(time.Unix(0, last))
}

// errSilent is returned when the client misses the pong deadline.
var errSilent = connectionError("peer silent beyond PongTimeout")

type connectionError string

func (e connectionError) Error() string { return string(e) }
