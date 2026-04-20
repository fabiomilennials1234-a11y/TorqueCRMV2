package ai_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/milennials/torque-api/internal/service/ai"
)

// --------------- MockEmbedder ----------------------------------------

func TestMockEmbedder_DeterministicDim(t *testing.T) {
	t.Parallel()
	em := ai.NewMockEmbedder()
	a, err := em.Embed(context.Background(), []string{"hello"})
	if err != nil {
		t.Fatalf("embed: %v", err)
	}
	b, err := em.Embed(context.Background(), []string{"hello"})
	if err != nil {
		t.Fatalf("embed re-run: %v", err)
	}
	if len(a) != 1 || len(b) != 1 {
		t.Fatalf("unexpected result count: %d/%d", len(a), len(b))
	}
	if a[0] != b[0] {
		t.Errorf("expected deterministic output across runs")
	}
	// Any difference in input must change at least one dimension.
	c, _ := em.Embed(context.Background(), []string{"world"})
	if a[0] == c[0] {
		t.Errorf("expected different vectors for different inputs")
	}
	for i, v := range a[0] {
		if v < -1.0001 || v > 1.0001 {
			t.Errorf("dim %d out of [-1,1]: %f", i, v)
		}
	}
}

func TestMockEmbedder_BatchKeepsOrder(t *testing.T) {
	t.Parallel()
	em := ai.NewMockEmbedder()
	batch, err := em.Embed(context.Background(), []string{"a", "b", "c"})
	if err != nil {
		t.Fatalf("embed: %v", err)
	}
	if len(batch) != 3 {
		t.Fatalf("want 3 vecs, got %d", len(batch))
	}
	singleA, _ := em.Embed(context.Background(), []string{"a"})
	if batch[0] != singleA[0] {
		t.Errorf("batch[0] differs from single a")
	}
}

// --------------- GeminiEmbedder construction ------------------------

func TestNewGeminiEmbedder_RejectsBadConfig(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		cfg  ai.GeminiConfig
	}{
		{"empty base", ai.GeminiConfig{APIKey: "k", Model: "m"}},
		{"http base", ai.GeminiConfig{BaseURL: "http://x", APIKey: "k", Model: "m"}},
		{"no api key", ai.GeminiConfig{BaseURL: "https://x", Model: "m"}},
		{"no model", ai.GeminiConfig{BaseURL: "https://x", APIKey: "k"}},
	}
	for _, c := range cases {
		if _, err := ai.NewGeminiEmbedder(c.cfg); err == nil {
			t.Errorf("%s: expected error", c.name)
		}
	}
}

// --------------- GeminiEmbedder Embed happy path --------------------

func TestGeminiEmbedder_Embed_HappyPath(t *testing.T) {
	t.Parallel()
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("want POST got %s", r.Method)
		}
		if !strings.Contains(r.URL.Path, "batchEmbedContents") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.URL.Query().Get("key") != "secret" {
			t.Errorf("missing api key in query")
		}
		// Decode the request and mirror the count.
		type reqEntry struct {
			Content struct {
				Parts []struct {
					Text string `json:"text"`
				} `json:"parts"`
			} `json:"content"`
		}
		var body struct {
			Requests []reqEntry `json:"requests"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode: %v", err)
		}
		embeddings := make([]map[string]any, len(body.Requests))
		for i := range body.Requests {
			vals := make([]float32, ai.EmbeddingDim)
			// Deterministic stub: position i in slot 0, rest 0.
			vals[0] = float32(i + 1)
			embeddings[i] = map[string]any{"values": vals}
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"embeddings": embeddings})
	}))
	defer srv.Close()

	em, err := ai.NewGeminiEmbedder(ai.GeminiConfig{
		BaseURL: srv.URL, APIKey: "secret", Model: "text-embedding-004",
		HTTPClient: srv.Client(),
	})
	if err != nil {
		t.Fatalf("new: %v", err)
	}
	got, err := em.Embed(context.Background(), []string{"hello", "world"})
	if err != nil {
		t.Fatalf("embed: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("want 2 vecs, got %d", len(got))
	}
	if got[0][0] != 1 || got[1][0] != 2 {
		t.Errorf("order not preserved: %f %f", got[0][0], got[1][0])
	}
}

// --------------- GeminiEmbedder error paths -------------------------

func TestGeminiEmbedder_Embed_RateLimited(t *testing.T) {
	t.Parallel()
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer srv.Close()
	em, _ := ai.NewGeminiEmbedder(ai.GeminiConfig{
		BaseURL: srv.URL, APIKey: "k", Model: "m", HTTPClient: srv.Client(),
	})
	_, err := em.Embed(context.Background(), []string{"x"})
	if !errors.Is(err, ai.ErrRateLimited) {
		t.Errorf("want ErrRateLimited, got %v", err)
	}
}

func TestGeminiEmbedder_Embed_AuthFailed(t *testing.T) {
	t.Parallel()
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer srv.Close()
	em, _ := ai.NewGeminiEmbedder(ai.GeminiConfig{
		BaseURL: srv.URL, APIKey: "k", Model: "m", HTTPClient: srv.Client(),
	})
	_, err := em.Embed(context.Background(), []string{"x"})
	if !errors.Is(err, ai.ErrAuthFailed) {
		t.Errorf("want ErrAuthFailed, got %v", err)
	}
}

func TestGeminiEmbedder_Embed_DimMismatch(t *testing.T) {
	t.Parallel()
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"embeddings": []map[string]any{{"values": []float32{0.1, 0.2, 0.3}}}, // 3 dims, not 768
		})
	}))
	defer srv.Close()
	em, _ := ai.NewGeminiEmbedder(ai.GeminiConfig{
		BaseURL: srv.URL, APIKey: "k", Model: "m", HTTPClient: srv.Client(),
	})
	_, err := em.Embed(context.Background(), []string{"x"})
	if !errors.Is(err, ai.ErrEmbeddingDim) {
		t.Errorf("want ErrEmbeddingDim, got %v", err)
	}
}

func TestGeminiEmbedder_Embed_RejectsLargeBatch(t *testing.T) {
	t.Parallel()
	em, _ := ai.NewGeminiEmbedder(ai.GeminiConfig{
		BaseURL: "https://x", APIKey: "k", Model: "m",
	})
	inputs := make([]string, 101)
	for i := range inputs {
		inputs[i] = "x"
	}
	_, err := em.Embed(context.Background(), inputs)
	if !errors.Is(err, ai.ErrBadRequest) {
		t.Errorf("want ErrBadRequest for oversize batch, got %v", err)
	}
}

func TestGeminiEmbedder_Embed_EmptyInputReturnsNil(t *testing.T) {
	t.Parallel()
	em, _ := ai.NewGeminiEmbedder(ai.GeminiConfig{
		BaseURL: "https://x", APIKey: "k", Model: "m",
	})
	got, err := em.Embed(context.Background(), nil)
	if err != nil {
		t.Errorf("want no error for empty input, got %v", err)
	}
	if got != nil {
		t.Errorf("want nil result for empty input, got %v", got)
	}
}
