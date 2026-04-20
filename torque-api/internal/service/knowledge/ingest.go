// Package knowledge owns the ingest pipeline for F06 Copilot RAG.
//
// For S39, ingest runs inline from the handler (detached goroutine)
// rather than through the operations ledger. Rationale: single-replica
// deploys + small source sizes (< 5MB text) mean the operations queue
// adds indirection without back-pressure benefit. When we go
// multi-replica (and URL fetching lands), this package becomes a
// `worker.Handler` registration and the handler stops spawning.
//
// The ingest contract:
//
//  1. Source row already exists with status='queued' (enqueued by handler).
//  2. Ingest() flips status → ingesting.
//  3. Chunks the payload (ai.Chunk).
//  4. Embeds each chunk's content (ai.Embedder).
//  5. Writes chunks + embeddings in a single tx (repo.InsertChunks).
//  6. Flips status → ready with ingested_at = now(), OR → failed + error.
//
// The pipeline fails closed: any error at step 3-5 leaves the source in
// `failed` so the UI can show it and the user can re-ingest explicitly
// (idempotent — step 5 deletes prior chunks first).
package knowledge

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	agentrepo "github.com/milennials/torque-api/internal/repository/agent"
	"github.com/milennials/torque-api/internal/service/ai"
)

// Service is the ingest orchestrator. All dependencies are interfaces
// so tests can drive the pipeline with fakes.
type Service struct {
	repo     *agentrepo.Repository
	embedder ai.Embedder
	logger   zerolog.Logger
}

// New builds a service with mandatory deps. The embedder may not be nil
// — callers should fall back to ai.MockEmbedder rather than leaving it
// unset, which would NPE at batch time.
func New(repo *agentrepo.Repository, embedder ai.Embedder, logger zerolog.Logger) *Service {
	return &Service{
		repo:     repo,
		embedder: embedder,
		logger:   logger.With().Str("component", "knowledge_ingest").Logger(),
	}
}

// Ingest drives the full pipeline for one source.
//
// The input `text` is the inline content the handler received. When
// the source has kind=url (deferred to S40+), the handler must fetch
// the URL first with HTTPS + SSRF guards, then hand the text here.
func (s *Service) Ingest(ctx context.Context, orgID, sourceID uuid.UUID, text string) error {
	// Transition to ingesting — a failure here is diagnostic only,
	// we continue even if the status flip doesn't persist.
	if err := s.repo.UpdateSourceStatus(ctx, orgID, sourceID, "ingesting", nil, nil); err != nil {
		s.logger.Warn().Err(err).Str("source_id", sourceID.String()).
			Msg("could not flip source to ingesting")
	}

	// Chunk.
	chunks := ai.Chunk(text, ai.DefaultChunkOptions())
	if len(chunks) == 0 {
		msg := "empty source after chunking"
		_ = s.repo.UpdateSourceStatus(ctx, orgID, sourceID, "failed", &msg, nil)
		return errors.New(msg)
	}

	// Embed in batches of 100 (Gemini cap). For S39 typical sources
	// that's one round-trip; larger corpora get sequential batches.
	const batchSize = 100
	contents := make([]string, len(chunks))
	for i, c := range chunks {
		contents[i] = c.Content
	}

	embeddings := make([]ai.Embedding, 0, len(chunks))
	for offset := 0; offset < len(contents); offset += batchSize {
		end := offset + batchSize
		if end > len(contents) {
			end = len(contents)
		}
		batch, err := s.embedder.Embed(ctx, contents[offset:end])
		if err != nil {
			msg := fmt.Sprintf("embed batch [%d:%d]: %v", offset, end, err)
			_ = s.repo.UpdateSourceStatus(ctx, orgID, sourceID, "failed", &msg, nil)
			return fmt.Errorf("embed: %w", err)
		}
		embeddings = append(embeddings, batch...)
	}

	if len(embeddings) != len(chunks) {
		msg := fmt.Sprintf("embedder returned %d vectors for %d chunks", len(embeddings), len(chunks))
		_ = s.repo.UpdateSourceStatus(ctx, orgID, sourceID, "failed", &msg, nil)
		return errors.New(msg)
	}

	// Persist.
	inputs := make([]agentrepo.InsertChunkInput, len(chunks))
	for i, c := range chunks {
		inputs[i] = agentrepo.InsertChunkInput{
			Ord:            c.Ord,
			Content:        c.Content,
			TokensEstimate: c.TokensEstimate,
			Embedding:      embeddings[i],
			Metadata:       json.RawMessage(fmt.Sprintf(`{"embedder":%q}`, s.embedder.Name())),
		}
	}
	if err := s.repo.InsertChunks(ctx, orgID, sourceID, inputs); err != nil {
		msg := err.Error()
		_ = s.repo.UpdateSourceStatus(ctx, orgID, sourceID, "failed", &msg, nil)
		return fmt.Errorf("insert chunks: %w", err)
	}

	now := time.Now().UTC()
	if err := s.repo.UpdateSourceStatus(ctx, orgID, sourceID, "ready", nil, &now); err != nil {
		return fmt.Errorf("mark ready: %w", err)
	}
	s.logger.Info().
		Str("source_id", sourceID.String()).
		Int("chunks", len(chunks)).
		Str("embedder", s.embedder.Name()).
		Msg("source ingested")
	return nil
}

// RunDetached spawns Ingest on a fresh context that survives the
// originating HTTP request. Used by the enqueue handler so the caller
// gets a 202 while the actual work completes in the background.
//
// Panic boundary: a panic in the embedder or repo here would otherwise
// take down the process silently. We recover and mark the source as
// failed so the UI surfaces it.
func (s *Service) RunDetached(orgID, sourceID uuid.UUID, text string) {
	go func() {
		// Generous 10min ceiling. Longer-than-this ingests should move
		// to the worker pool with proper progress reporting.
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
		defer cancel()
		defer func() {
			if rec := recover(); rec != nil {
				msg := fmt.Sprintf("ingest panic: %v", rec)
				s.logger.Error().
					Str("source_id", sourceID.String()).
					Interface("panic", rec).
					Msg("ingest recovered")
				_ = s.repo.UpdateSourceStatus(ctx, orgID, sourceID, "failed", &msg, nil)
			}
		}()
		if err := s.Ingest(ctx, orgID, sourceID, text); err != nil {
			s.logger.Warn().Err(err).Str("source_id", sourceID.String()).Msg("ingest failed")
		}
	}()
}
