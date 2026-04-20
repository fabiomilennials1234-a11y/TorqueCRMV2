// Package inbox is the pgx-backed store for F04 channels, conversations and
// messages. Webhook intake + outbound send build on this; conversation list
// + detail read from it.
package inbox

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrNotFound = errors.New("not found")

// Conversation is the list-row shape.
type Conversation struct {
	ID                 uuid.UUID
	OrganizationID     uuid.UUID
	ChannelID          uuid.UUID
	ChannelKind        string
	LeadID             *uuid.UUID
	ExternalThreadID   string
	ContactName        *string
	ContactHandle      *string
	State              string
	AssignedTo         *uuid.UUID
	UnreadCount        int
	LastMessageAt      *time.Time
	LastMessagePreview *string
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

// Message is the detail-row shape.
type Message struct {
	ID              uuid.UUID
	OrganizationID  uuid.UUID
	ConversationID  uuid.UUID
	ExternalID      *string
	Direction       string
	Kind            string
	Body            *string
	MediaURL        *string
	MediaMime       *string
	SentByMemberID  *uuid.UUID
	Status          string
	OccurredAt      time.Time
	DeliveredAt     *time.Time
	ReadAt          *time.Time
	CreatedAt       time.Time
}

// Repository wraps a pgx pool.
type Repository struct {
	pool *pgxpool.Pool
}

// New binds the repository.
func New(pool *pgxpool.Pool) *Repository { return &Repository{pool: pool} }

// -------- channels ----------------------------------------------------

// UpsertChannel creates a channel or updates its name by (org, kind, external_id).
// external_id is REQUIRED — SQL's NULL != NULL means the UNIQUE index would
// silently accept duplicates when externalID is nil. Callers that truly have
// no provider id should pass a stable synthetic token.
func (r *Repository) UpsertChannel(ctx context.Context, orgID uuid.UUID, kind, name string, externalID *string) (uuid.UUID, error) {
	if externalID == nil || *externalID == "" {
		return uuid.Nil, errors.New("external_id is required to prevent duplicate channels")
	}
	const q = `
		INSERT INTO channels (organization_id, kind, name, external_id)
		VALUES ($1, $2::channel_kind, $3, $4)
		ON CONFLICT (organization_id, kind, external_id) DO UPDATE
		  SET name = EXCLUDED.name
		RETURNING id
	`
	var id uuid.UUID
	if err := r.pool.QueryRow(ctx, q, orgID, kind, name, externalID).Scan(&id); err != nil {
		return uuid.Nil, fmt.Errorf("upsert channel: %w", err)
	}
	return id, nil
}

// -------- conversations ----------------------------------------------

// ListConversationsOptions filters the list endpoint.
type ListConversationsOptions struct {
	State      string // "" = any
	AssignedTo *uuid.UUID
	ChannelID  *uuid.UUID
	Limit      int
}

// ListConversations returns the most recently active conversations.
func (r *Repository) ListConversations(ctx context.Context, orgID uuid.UUID, opts ListConversationsOptions) ([]Conversation, error) {
	limit := opts.Limit
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	args := []any{orgID, limit}
	q := `
		SELECT c.id, c.organization_id, c.channel_id, ch.kind::text, c.lead_id,
		       c.external_thread_id, c.contact_name, c.contact_handle,
		       c.state::text, c.assigned_to, c.unread_count,
		       c.last_message_at, c.last_message_preview,
		       c.created_at, c.updated_at
		  FROM conversations c
		  JOIN channels ch ON ch.id = c.channel_id
		 WHERE c.organization_id = $1
	`
	if opts.State != "" {
		args = append(args, opts.State)
		q += fmt.Sprintf(" AND c.state = $%d::conversation_state", len(args))
	}
	if opts.AssignedTo != nil {
		args = append(args, *opts.AssignedTo)
		q += fmt.Sprintf(" AND c.assigned_to = $%d", len(args))
	}
	if opts.ChannelID != nil {
		args = append(args, *opts.ChannelID)
		q += fmt.Sprintf(" AND c.channel_id = $%d", len(args))
	}
	q += " ORDER BY c.last_message_at DESC NULLS LAST, c.id DESC LIMIT $2"

	rows, err := r.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("list conversations: %w", err)
	}
	defer rows.Close()
	out := make([]Conversation, 0, limit)
	for rows.Next() {
		var c Conversation
		if err := rows.Scan(
			&c.ID, &c.OrganizationID, &c.ChannelID, &c.ChannelKind, &c.LeadID,
			&c.ExternalThreadID, &c.ContactName, &c.ContactHandle,
			&c.State, &c.AssignedTo, &c.UnreadCount,
			&c.LastMessageAt, &c.LastMessagePreview,
			&c.CreatedAt, &c.UpdatedAt,
		); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// GetConversation returns one conversation by id within the tenant.
func (r *Repository) GetConversation(ctx context.Context, orgID, id uuid.UUID) (Conversation, error) {
	const q = `
		SELECT c.id, c.organization_id, c.channel_id, ch.kind::text, c.lead_id,
		       c.external_thread_id, c.contact_name, c.contact_handle,
		       c.state::text, c.assigned_to, c.unread_count,
		       c.last_message_at, c.last_message_preview,
		       c.created_at, c.updated_at
		  FROM conversations c
		  JOIN channels ch ON ch.id = c.channel_id
		 WHERE c.organization_id = $1 AND c.id = $2
		 LIMIT 1
	`
	var c Conversation
	err := r.pool.QueryRow(ctx, q, orgID, id).Scan(
		&c.ID, &c.OrganizationID, &c.ChannelID, &c.ChannelKind, &c.LeadID,
		&c.ExternalThreadID, &c.ContactName, &c.ContactHandle,
		&c.State, &c.AssignedTo, &c.UnreadCount,
		&c.LastMessageAt, &c.LastMessagePreview,
		&c.CreatedAt, &c.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return Conversation{}, ErrNotFound
	}
	if err != nil {
		return Conversation{}, fmt.Errorf("get conversation: %w", err)
	}
	return c, nil
}

// UpsertConversation creates or updates a conversation by (channel, external_thread_id).
// Used by webhook intake — the provider supplies the thread id.
//
// Tenant safety: channel_id is validated to belong to organization_id BEFORE
// the insert. A misrouted or spoofed webhook that supplies another tenant's
// channel id cannot create a cross-tenant conversation row.
type UpsertConversationInput struct {
	OrganizationID   uuid.UUID
	ChannelID        uuid.UUID
	ExternalThreadID string
	ContactName      *string
	ContactHandle    *string
	Metadata         json.RawMessage
}

func (r *Repository) UpsertConversation(ctx context.Context, in UpsertConversationInput) (uuid.UUID, error) {
	if in.ExternalThreadID == "" {
		return uuid.Nil, errors.New("external_thread_id is required")
	}
	meta := in.Metadata
	if len(meta) == 0 {
		meta = []byte(`{}`)
	}
	// Ownership pre-flight: channel must belong to the caller's tenant.
	var ownedBy uuid.UUID
	if err := r.pool.QueryRow(ctx,
		`SELECT organization_id FROM channels WHERE id = $1 LIMIT 1`,
		in.ChannelID,
	).Scan(&ownedBy); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return uuid.Nil, ErrNotFound
		}
		return uuid.Nil, fmt.Errorf("channel ownership: %w", err)
	}
	if ownedBy != in.OrganizationID {
		return uuid.Nil, ErrNotFound // never leak cross-tenant existence
	}

	const q = `
		INSERT INTO conversations (
		  organization_id, channel_id, external_thread_id,
		  contact_name, contact_handle, metadata
		) VALUES ($1,$2,$3,$4,$5,$6)
		ON CONFLICT (channel_id, external_thread_id) DO UPDATE
		  SET contact_name   = COALESCE(EXCLUDED.contact_name, conversations.contact_name),
		      contact_handle = COALESCE(EXCLUDED.contact_handle, conversations.contact_handle)
		RETURNING id
	`
	var id uuid.UUID
	if err := r.pool.QueryRow(ctx, q,
		in.OrganizationID, in.ChannelID, in.ExternalThreadID,
		in.ContactName, in.ContactHandle, meta,
	).Scan(&id); err != nil {
		return uuid.Nil, fmt.Errorf("upsert conversation: %w", err)
	}
	return id, nil
}

// AssignConversation sets assigned_to on a conversation.
func (r *Repository) AssignConversation(ctx context.Context, orgID, id uuid.UUID, assignee *uuid.UUID) error {
	ct, err := r.pool.Exec(ctx,
		`UPDATE conversations SET assigned_to = $3 WHERE organization_id = $1 AND id = $2`,
		orgID, id, assignee,
	)
	if err != nil {
		return fmt.Errorf("assign: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// SetConversationState transitions the state field (resolved, archived, etc).
func (r *Repository) SetConversationState(ctx context.Context, orgID, id uuid.UUID, state string) error {
	ct, err := r.pool.Exec(ctx,
		`UPDATE conversations SET state = $3::conversation_state WHERE organization_id = $1 AND id = $2`,
		orgID, id, state,
	)
	if err != nil {
		return fmt.Errorf("set state: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// MarkRead zeroes unread_count on a conversation (user opened the thread).
func (r *Repository) MarkRead(ctx context.Context, orgID, id uuid.UUID) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE conversations SET unread_count = 0 WHERE organization_id = $1 AND id = $2`,
		orgID, id,
	)
	return err
}

// -------- messages ---------------------------------------------------

// AppendMessageInput carries all fields to insert + denormalize on the conv.
type AppendMessageInput struct {
	OrganizationID  uuid.UUID
	ConversationID  uuid.UUID
	ExternalID      *string
	Direction       string // inbound | outbound
	Kind            string
	Body            *string
	MediaURL        *string
	MediaMime       *string
	SentByMemberID  *uuid.UUID
	OccurredAt      time.Time
}

// AppendMessage inserts a message and keeps the parent conversation's
// last_message_* fields in sync + bumps unread_count for inbound. Runs in
// one transaction so WS broadcasts always see consistent state.
//
// Tenant safety: the conversation_id is validated to belong to
// organization_id before the insert. Without this guard a member of org A
// could post a conversation_id belonging to org B and create an orphaned
// row that org A's queries never see but that poisons org B's
// (conversation_id, external_id) dedup index.
func (r *Repository) AppendMessage(ctx context.Context, in AppendMessageInput) (Message, error) {
	if in.Direction != "inbound" && in.Direction != "outbound" {
		return Message{}, fmt.Errorf("invalid direction: %s", in.Direction)
	}
	if in.OccurredAt.IsZero() {
		in.OccurredAt = time.Now().UTC()
	}

	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return Message{}, fmt.Errorf("begin: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// Ownership guard — refuse if conversation is not in the caller's tenant.
	var ownedBy uuid.UUID
	if err := tx.QueryRow(ctx,
		`SELECT organization_id FROM conversations WHERE id = $1 LIMIT 1`,
		in.ConversationID,
	).Scan(&ownedBy); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Message{}, ErrNotFound
		}
		return Message{}, fmt.Errorf("conversation ownership: %w", err)
	}
	if ownedBy != in.OrganizationID {
		return Message{}, ErrNotFound
	}

	var m Message
	err = tx.QueryRow(ctx,
		`INSERT INTO messages (
		   organization_id, conversation_id, external_id, direction, kind,
		   body, media_url, media_mime, sent_by_member_id, occurred_at, status
		 ) VALUES ($1,$2,$3,$4::message_direction,$5::message_kind,$6,$7,$8,$9,$10,
		           CASE WHEN $4='outbound' THEN 'queued' ELSE 'delivered' END::message_status)
		 ON CONFLICT (conversation_id, external_id) DO UPDATE SET status = messages.status
		 RETURNING id, organization_id, conversation_id, external_id, direction::text,
		           kind::text, body, media_url, media_mime, sent_by_member_id,
		           status::text, occurred_at, delivered_at, read_at, created_at`,
		in.OrganizationID, in.ConversationID, in.ExternalID, in.Direction, in.Kind,
		in.Body, in.MediaURL, in.MediaMime, in.SentByMemberID, in.OccurredAt,
	).Scan(
		&m.ID, &m.OrganizationID, &m.ConversationID, &m.ExternalID, &m.Direction,
		&m.Kind, &m.Body, &m.MediaURL, &m.MediaMime, &m.SentByMemberID,
		&m.Status, &m.OccurredAt, &m.DeliveredAt, &m.ReadAt, &m.CreatedAt,
	)
	if err != nil {
		return Message{}, fmt.Errorf("insert message: %w", err)
	}

	// Denormalize onto the conversation for the list view.
	preview := messagePreview(m)
	_, err = tx.Exec(ctx,
		`UPDATE conversations
		    SET last_message_at      = $2,
		        last_message_preview = $3,
		        unread_count = CASE WHEN $4 = 'inbound' THEN unread_count + 1 ELSE unread_count END
		  WHERE organization_id = $5 AND id = $1`,
		in.ConversationID, in.OccurredAt, preview, in.Direction, in.OrganizationID,
	)
	if err != nil {
		return Message{}, fmt.Errorf("update conversation: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return Message{}, fmt.Errorf("commit: %w", err)
	}
	return m, nil
}

// ListMessages returns the message history of a conversation, oldest first.
// S13 will layer cursor pagination; S12 ships a simple limit-based list.
func (r *Repository) ListMessages(ctx context.Context, orgID, convID uuid.UUID, limit int) ([]Message, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	const q = `
		SELECT id, organization_id, conversation_id, external_id,
		       direction::text, kind::text, body, media_url, media_mime,
		       sent_by_member_id, status::text,
		       occurred_at, delivered_at, read_at, created_at
		  FROM messages
		 WHERE organization_id = $1 AND conversation_id = $2
		 ORDER BY occurred_at
		 LIMIT $3
	`
	rows, err := r.pool.Query(ctx, q, orgID, convID, limit)
	if err != nil {
		return nil, fmt.Errorf("list messages: %w", err)
	}
	defer rows.Close()
	out := make([]Message, 0, limit)
	for rows.Next() {
		var m Message
		if err := rows.Scan(
			&m.ID, &m.OrganizationID, &m.ConversationID, &m.ExternalID,
			&m.Direction, &m.Kind, &m.Body, &m.MediaURL, &m.MediaMime,
			&m.SentByMemberID, &m.Status,
			&m.OccurredAt, &m.DeliveredAt, &m.ReadAt, &m.CreatedAt,
		); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

// messagePreview returns at most 200 runes of the body, or a localized
// placeholder for non-text kinds. Rune-safe: WhatsApp/Instagram bodies rich
// in emoji or CJK would otherwise corrupt mid-codepoint if sliced at the
// byte offset, producing invalid UTF-8 that encoding/json replaces with
// U+FFFD downstream.
func messagePreview(m Message) string {
	if m.Body != nil {
		rs := []rune(*m.Body)
		if len(rs) > 200 {
			rs = rs[:200]
		}
		return string(rs)
	}
	switch m.Kind {
	case "image":
		return "[imagem]"
	case "audio":
		return "[áudio]"
	case "video":
		return "[vídeo]"
	case "document":
		return "[documento]"
	case "sticker":
		return "[sticker]"
	default:
		return ""
	}
}
