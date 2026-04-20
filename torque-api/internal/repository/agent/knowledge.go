// Knowledge base extensions for F06 Copilot RAG (S39).
//
// This file complements agent.go — agents own per-tenant configuration;
// this file owns the per-tenant retrievable documents + chunks the
// playground (and eventually production sessions) can search against.
//
// The pgvector-shaped operations (`<=>`, `vector(...)::vector`) live
// here rather than in agent.go so the separation is obvious when
// reviewing the dependency on the 0019 migration.

package agent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/milennials/torque-api/internal/service/ai"
)

// KnowledgeCollection is the UI-facing view of a collection.
type KnowledgeCollection struct {
	ID             uuid.UUID
	OrganizationID uuid.UUID
	Name           string
	Description    *string
	SourceCount    int
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// KnowledgeSource is one ingestable document inside a collection.
type KnowledgeSource struct {
	ID             uuid.UUID
	OrganizationID uuid.UUID
	CollectionID   uuid.UUID
	Kind           string
	Title          string
	URI            *string
	Status         string
	Error          *string
	IngestedAt     *time.Time
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// RetrievedChunk is what SimilaritySearch hands back to the caller:
// a chunk of content plus its cosine distance to the query vector.
// Lower Distance = more similar (0 == identical direction).
type RetrievedChunk struct {
	ID        uuid.UUID
	SourceID  uuid.UUID
	Ord       int
	Content   string
	Distance  float64
}

// ListCollections returns every collection for the tenant with a live
// source_count. Ordered by name (stable) so pagination is predictable.
func (r *Repository) ListCollections(ctx context.Context, orgID uuid.UUID) ([]KnowledgeCollection, error) {
	const q = `
		SELECT c.id, c.organization_id, c.name, c.description,
		       c.created_at, c.updated_at,
		       COALESCE(s.source_count, 0) AS source_count
		  FROM knowledge_collections c
		  LEFT JOIN (
		    SELECT collection_id, COUNT(*) AS source_count
		      FROM knowledge_sources
		     WHERE organization_id = $1
		     GROUP BY collection_id
		  ) s ON s.collection_id = c.id
		 WHERE c.organization_id = $1
		 ORDER BY c.name
	`
	rows, err := r.pool.Query(ctx, q, orgID)
	if err != nil {
		return nil, fmt.Errorf("list collections: %w", err)
	}
	defer rows.Close()
	out := make([]KnowledgeCollection, 0, 4)
	for rows.Next() {
		var c KnowledgeCollection
		if err := rows.Scan(&c.ID, &c.OrganizationID, &c.Name, &c.Description,
			&c.CreatedAt, &c.UpdatedAt, &c.SourceCount); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// GetCollection returns a single collection by id, scoped to tenant.
func (r *Repository) GetCollection(ctx context.Context, orgID, id uuid.UUID) (KnowledgeCollection, error) {
	const q = `
		SELECT id, organization_id, name, description, created_at, updated_at
		  FROM knowledge_collections
		 WHERE id = $1 AND organization_id = $2
		 LIMIT 1
	`
	var c KnowledgeCollection
	err := r.pool.QueryRow(ctx, q, id, orgID).Scan(
		&c.ID, &c.OrganizationID, &c.Name, &c.Description, &c.CreatedAt, &c.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return KnowledgeCollection{}, ErrNotFound
	}
	if err != nil {
		return KnowledgeCollection{}, fmt.Errorf("get collection: %w", err)
	}
	return c, nil
}

// ListSources returns every source inside a collection for the tenant.
func (r *Repository) ListSources(ctx context.Context, orgID, collectionID uuid.UUID) ([]KnowledgeSource, error) {
	const q = `
		SELECT id, organization_id, collection_id, kind, title, uri,
		       status::text, error, ingested_at, created_at, updated_at
		  FROM knowledge_sources
		 WHERE organization_id = $1 AND collection_id = $2
		 ORDER BY created_at DESC
	`
	rows, err := r.pool.Query(ctx, q, orgID, collectionID)
	if err != nil {
		return nil, fmt.Errorf("list sources: %w", err)
	}
	defer rows.Close()
	out := make([]KnowledgeSource, 0, 8)
	for rows.Next() {
		var s KnowledgeSource
		if err := rows.Scan(&s.ID, &s.OrganizationID, &s.CollectionID, &s.Kind, &s.Title, &s.URI,
			&s.Status, &s.Error, &s.IngestedAt, &s.CreatedAt, &s.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

// GetSource returns a single source, scoped to tenant.
func (r *Repository) GetSource(ctx context.Context, orgID, id uuid.UUID) (KnowledgeSource, error) {
	const q = `
		SELECT id, organization_id, collection_id, kind, title, uri,
		       status::text, error, ingested_at, created_at, updated_at
		  FROM knowledge_sources
		 WHERE id = $1 AND organization_id = $2
		 LIMIT 1
	`
	var s KnowledgeSource
	err := r.pool.QueryRow(ctx, q, id, orgID).Scan(
		&s.ID, &s.OrganizationID, &s.CollectionID, &s.Kind, &s.Title, &s.URI,
		&s.Status, &s.Error, &s.IngestedAt, &s.CreatedAt, &s.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return KnowledgeSource{}, ErrNotFound
	}
	if err != nil {
		return KnowledgeSource{}, fmt.Errorf("get source: %w", err)
	}
	return s, nil
}

// UpdateSourceStatus transitions status + optional error + ingested_at.
// The ingest worker calls this at ingestion start (→ingesting), on
// failure (→failed + error), and on success (→ready + ingested_at).
func (r *Repository) UpdateSourceStatus(
	ctx context.Context, orgID, id uuid.UUID, status string, errMsg *string, ingestedAt *time.Time,
) error {
	sets := []string{"status = $3::knowledge_source_status", "error = $4"}
	args := []any{orgID, id, status, errMsg}
	if ingestedAt != nil {
		sets = append(sets, "ingested_at = $5")
		args = append(args, *ingestedAt)
	}
	q := fmt.Sprintf(
		`UPDATE knowledge_sources SET %s WHERE organization_id = $1 AND id = $2`,
		strings.Join(sets, ", "),
	)
	ct, err := r.pool.Exec(ctx, q, args...)
	if err != nil {
		return fmt.Errorf("update source status: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// InsertChunkInput is one row the ingest worker wants to persist.
type InsertChunkInput struct {
	Ord            int
	Content        string
	TokensEstimate int
	Embedding      ai.Embedding
	Metadata       json.RawMessage
}

// InsertChunks upserts chunks for a source in a single transaction.
// Any previous chunks for this source are deleted first — ingest is
// idempotent, so a re-run overwrites rather than duplicating.
//
// The vector is passed as a pgvector string literal (`[0.1,0.2,...]`)
// because pgx doesn't ship a native encoder for the vector type at
// the driver level. The cast `::vector` tells Postgres to parse it.
func (r *Repository) InsertChunks(ctx context.Context, orgID, sourceID uuid.UUID, chunks []InsertChunkInput) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if _, err := tx.Exec(ctx,
		`DELETE FROM knowledge_chunks WHERE organization_id = $1 AND source_id = $2`,
		orgID, sourceID,
	); err != nil {
		return fmt.Errorf("clear previous chunks: %w", err)
	}

	const q = `
		INSERT INTO knowledge_chunks (
		  organization_id, source_id, ord, content, tokens_estimate,
		  embedding, embedding_text, metadata
		) VALUES ($1, $2, $3, $4, $5, $6::vector, NULL, $7::jsonb)
	`
	for _, c := range chunks {
		meta := c.Metadata
		if len(meta) == 0 {
			meta = []byte(`{}`)
		}
		if _, err := tx.Exec(ctx, q,
			orgID, sourceID, c.Ord, c.Content, c.TokensEstimate,
			encodeVector(c.Embedding), meta,
		); err != nil {
			return fmt.Errorf("insert chunk %d: %w", c.Ord, err)
		}
	}

	return tx.Commit(ctx)
}

// SimilaritySearch returns the top-K chunks across every source in the
// agent's bound collection, ordered by cosine distance (ascending).
// Returns (nil, nil) when the agent has no collection bound — callers
// should skip retrieval entirely in that case.
//
// The query is tenant-scoped first (index stays selective), then
// filtered to sources inside the agent's collection. `topK` is clamped
// to [1, 20]; higher values start eating into the prompt budget.
func (r *Repository) SimilaritySearch(
	ctx context.Context, orgID, agentID uuid.UUID, query ai.Embedding, topK int,
) ([]RetrievedChunk, error) {
	if topK < 1 {
		topK = 1
	}
	if topK > 20 {
		topK = 20
	}

	// Which collection is the agent bound to?
	var collectionID *uuid.UUID
	if err := r.pool.QueryRow(ctx,
		`SELECT knowledge_collection_id FROM agents
		  WHERE organization_id = $1 AND id = $2
		  LIMIT 1`,
		orgID, agentID,
	).Scan(&collectionID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("lookup agent collection: %w", err)
	}
	if collectionID == nil {
		return nil, nil
	}

	const q = `
		SELECT k.id, k.source_id, k.ord, k.content,
		       (k.embedding <=> $3::vector) AS distance
		  FROM knowledge_chunks k
		  JOIN knowledge_sources s ON s.id = k.source_id
		 WHERE k.organization_id = $1
		   AND s.collection_id = $2
		   AND s.status = 'ready'
		   AND k.embedding IS NOT NULL
		 ORDER BY k.embedding <=> $3::vector
		 LIMIT $4
	`
	rows, err := r.pool.Query(ctx, q, orgID, *collectionID, encodeVector(query), topK)
	if err != nil {
		return nil, fmt.Errorf("similarity search: %w", err)
	}
	defer rows.Close()
	out := make([]RetrievedChunk, 0, topK)
	for rows.Next() {
		var c RetrievedChunk
		if err := rows.Scan(&c.ID, &c.SourceID, &c.Ord, &c.Content, &c.Distance); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// SetAgentKnowledgeCollection binds (or unbinds) an agent's collection.
// Passing nil unsets the binding, disabling retrieval for the agent.
func (r *Repository) SetAgentKnowledgeCollection(
	ctx context.Context, orgID, agentID uuid.UUID, collectionID *uuid.UUID,
) error {
	// Validate collection ownership when setting; detach path skips.
	if collectionID != nil {
		var ownedBy uuid.UUID
		if err := r.pool.QueryRow(ctx,
			`SELECT organization_id FROM knowledge_collections WHERE id = $1 LIMIT 1`,
			*collectionID,
		).Scan(&ownedBy); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return ErrNotFound
			}
			return fmt.Errorf("collection ownership: %w", err)
		}
		if ownedBy != orgID {
			return ErrNotFound
		}
	}

	ct, err := r.pool.Exec(ctx,
		`UPDATE agents SET knowledge_collection_id = $3
		  WHERE organization_id = $1 AND id = $2`,
		orgID, agentID, collectionID,
	)
	if err != nil {
		return fmt.Errorf("set agent knowledge: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// encodeVector renders an Embedding as a pgvector text literal. Using
// %g (shortest unambiguous) keeps the payload small without losing
// roundtripping precision for float32.
func encodeVector(v ai.Embedding) string {
	var b strings.Builder
	b.Grow(ai.EmbeddingDim * 10)
	b.WriteByte('[')
	for i, f := range v {
		if i > 0 {
			b.WriteByte(',')
		}
		_, _ = fmt.Fprintf(&b, "%g", f)
	}
	b.WriteByte(']')
	return b.String()
}
