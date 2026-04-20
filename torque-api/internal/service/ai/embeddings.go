package ai

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"strings"
	"time"
)

// ---------- errors + shape -----------------------------------------

// ErrEmbeddingDim is returned when the provider hands back a vector
// whose dimension does not match EmbeddingDim. The caller MUST refuse
// to write it to the DB — mixing dimensions into the same pgvector
// column corrupts the HNSW index.
var ErrEmbeddingDim = errors.New("ai: embedding dimension mismatch")

// EmbeddingDim is the wire-level contract with migration 0019
// (`vector(768)`). Changing this value requires a new migration with
// a vector column of the new dimension plus backfill — never silently
// let a different dim leak into the system.
const EmbeddingDim = 768

// Embedding is a single dense vector in Euclidean space, length ==
// EmbeddingDim. The tenant-scoped HNSW index uses cosine distance
// (`<=>`), so upstream callers do NOT need to L2-normalize; pgvector
// handles normalization internally for `vector_cosine_ops`.
type Embedding [EmbeddingDim]float32

// Embedder is the provider-agnostic surface the ingest worker +
// retrieval service depend on. Two concrete implementations ship with
// S39: GeminiEmbedder (real) and MockEmbedder (deterministic hash
// fallback for tests + key-less dev).
type Embedder interface {
	Name() string
	// Embed returns one vector per input text in the same order.
	// Batch size is the caller's concern — some providers cap at 100;
	// Gemini caps at 250 per batch. Implementations MUST NOT silently
	// chunk; if the batch exceeds their limit, they return an error so
	// the caller decides whether to fan out.
	Embed(ctx context.Context, inputs []string) ([]Embedding, error)
}

// ---------- Gemini implementation ------------------------------------

// GeminiConfig carries everything the real embedder needs.
type GeminiConfig struct {
	BaseURL    string
	APIKey     string
	Model      string        // e.g. "text-embedding-004"
	Timeout    time.Duration // default 15s
	HTTPClient *http.Client
}

// GeminiEmbedder talks to Google's `generativelanguage.googleapis.com`
// batchEmbedContents endpoint. Authentication is by `?key=<API_KEY>`
// query string per the published contract; the API key is sensitive
// and is scrubbed by the Sentry filter (see internal/observability).
type GeminiEmbedder struct {
	cfg    GeminiConfig
	client *http.Client
}

// NewGeminiEmbedder validates config and returns a ready embedder.
func NewGeminiEmbedder(cfg GeminiConfig) (*GeminiEmbedder, error) {
	base := strings.TrimRight(cfg.BaseURL, "/")
	if base == "" {
		return nil, errors.New("gemini: BaseURL required")
	}
	if !strings.HasPrefix(base, "https://") {
		return nil, errors.New("gemini: BaseURL must use https://")
	}
	if cfg.APIKey == "" {
		return nil, errors.New("gemini: APIKey required")
	}
	if cfg.Model == "" {
		return nil, errors.New("gemini: Model required")
	}
	if cfg.Timeout <= 0 {
		cfg.Timeout = 15 * time.Second
	}
	client := cfg.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: cfg.Timeout}
	}
	cfg.BaseURL = base
	return &GeminiEmbedder{cfg: cfg, client: client}, nil
}

// Name implements Embedder.
func (*GeminiEmbedder) Name() string { return "gemini" }

// Embed batches up to 100 inputs in a single batchEmbedContents call.
// Larger batches surface as ErrBadRequest so the caller can fan out
// rather than silently losing rows to provider quota.
func (e *GeminiEmbedder) Embed(ctx context.Context, inputs []string) ([]Embedding, error) {
	if len(inputs) == 0 {
		return nil, nil
	}
	if len(inputs) > 100 {
		return nil, fmt.Errorf("%w: batch > 100 (got %d)", ErrBadRequest, len(inputs))
	}

	type contentPart struct {
		Text string `json:"text"`
	}
	type content struct {
		Parts []contentPart `json:"parts"`
	}
	type requestEntry struct {
		Model   string  `json:"model"`
		Content content `json:"content"`
		// TaskType RETRIEVAL_DOCUMENT tunes the embedding space for
		// document-in-corpus retrieval; the query-side uses
		// RETRIEVAL_QUERY. Leaving it unset gives a general vector
		// that works for both but is measurably weaker at topK.
		TaskType string `json:"taskType,omitempty"`
	}
	type requestBody struct {
		Requests []requestEntry `json:"requests"`
	}

	entries := make([]requestEntry, len(inputs))
	modelPath := "models/" + e.cfg.Model
	for i, in := range inputs {
		entries[i] = requestEntry{
			Model:    modelPath,
			Content:  content{Parts: []contentPart{{Text: in}}},
			TaskType: "RETRIEVAL_DOCUMENT",
		}
	}
	bodyBytes, err := json.Marshal(requestBody{Requests: entries})
	if err != nil {
		return nil, fmt.Errorf("encode embeddings request: %w", err)
	}

	url := fmt.Sprintf("%s/models/%s:batchEmbedContents?key=%s",
		e.cfg.BaseURL, e.cfg.Model, e.cfg.APIKey)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("build embeddings request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	res, err := e.client.Do(req)
	if err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return nil, err
		}
		return nil, ErrProviderUnavailable
	}
	defer res.Body.Close()

	switch {
	case res.StatusCode == http.StatusUnauthorized, res.StatusCode == http.StatusForbidden:
		return nil, ErrAuthFailed
	case res.StatusCode == http.StatusTooManyRequests:
		return nil, ErrRateLimited
	case res.StatusCode >= 500:
		return nil, ErrProviderUnavailable
	case res.StatusCode >= 400:
		raw, _ := io.ReadAll(io.LimitReader(res.Body, 200))
		return nil, fmt.Errorf("%w: %s", ErrBadRequest, strings.TrimSpace(string(raw)))
	}

	type responseEmbedding struct {
		Values []float32 `json:"values"`
	}
	type responseBody struct {
		Embeddings []responseEmbedding `json:"embeddings"`
	}
	var rb responseBody
	if err := json.NewDecoder(res.Body).Decode(&rb); err != nil {
		return nil, fmt.Errorf("decode embeddings response: %w", err)
	}
	if len(rb.Embeddings) != len(inputs) {
		return nil, fmt.Errorf("%w: provider returned %d embeddings for %d inputs",
			ErrBadRequest, len(rb.Embeddings), len(inputs))
	}

	out := make([]Embedding, len(rb.Embeddings))
	for i, emb := range rb.Embeddings {
		if len(emb.Values) != EmbeddingDim {
			return nil, fmt.Errorf("%w: got %d (expected %d)",
				ErrEmbeddingDim, len(emb.Values), EmbeddingDim)
		}
		copy(out[i][:], emb.Values)
	}
	return out, nil
}

// ---------- mock implementation --------------------------------------

// MockEmbedder produces deterministic vectors from SHA-256 of the
// input. Two identical strings always get the same vector — so
// ordering in SimilaritySearch is testable without hitting the
// provider. It is NOT a real semantic embedder; swapping it in for
// production retrieval would make similarity meaningless.
//
// Used when GEMINI_API_KEY is empty (dev bootstrap, unit tests). The
// boot log prints a big warning so prod deployments fail loudly if
// the real key is missing.
type MockEmbedder struct{}

// NewMockEmbedder returns a ready mock.
func NewMockEmbedder() *MockEmbedder { return &MockEmbedder{} }

// Name implements Embedder.
func (*MockEmbedder) Name() string { return "mock" }

// Embed hashes each input with SHA-256 and expands it deterministically
// to EmbeddingDim floats in [-1, 1]. The hash→float expansion uses
// successive SHA-256 iterations so close rewrites of the same string
// land at similar vectors (which helps exercise the cosine path in
// unit tests without needing a real provider).
func (*MockEmbedder) Embed(_ context.Context, inputs []string) ([]Embedding, error) {
	out := make([]Embedding, len(inputs))
	for i, in := range inputs {
		out[i] = deterministicEmbedding(in)
	}
	return out, nil
}

func deterministicEmbedding(s string) Embedding {
	var emb Embedding
	seed := sha256.Sum256([]byte(s))
	// Expand 32 bytes of seed into EmbeddingDim floats by chaining
	// SHA-256 rounds. Each round covers 32 bytes = 8 float32s. We need
	// ceil(768 / 8) = 96 rounds.
	cur := seed
	written := 0
	for written < EmbeddingDim {
		for i := 0; i < len(cur) && written < EmbeddingDim; i += 4 {
			n := binary.BigEndian.Uint32(cur[i : i+4])
			// Map uint32 → float32 in [-1, 1] via u/2^31 - 1.
			emb[written] = float32(n)/math.MaxInt32 - 1
			written++
		}
		cur = sha256.Sum256(cur[:])
	}
	return emb
}
