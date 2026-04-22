package szchat

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/milennials/torque-api/internal/service/integration"
)

// We exercise postJSON directly via a test HTTP server. Store-dependent
// paths (StoreAPIKey, SendMessageForOrg credential lookup) run on the
// handler/repo tier against a real pgxpool.

func newProvider(base string) *Provider {
	return New(Config{BaseURL: base}, nil, nil, nil)
}

func TestPostJSON_HappyPath(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method: %s", r.Method)
		}
		if r.URL.Path != "/messages" {
			t.Errorf("path: %s", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer k" {
			t.Errorf("auth header: %q", got)
		}
		body, _ := io.ReadAll(r.Body)
		if !strings.Contains(string(body), `"to":"5511999"`) {
			t.Errorf("unexpected body: %s", body)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"msg_1","accepted_at":"2026-01-01T00:00:00Z"}`))
	}))
	defer srv.Close()

	p := newProvider(srv.URL)
	res, err := p.postJSON(context.Background(), "k", "/messages", outboundBody{
		Recipient: "5511999", Kind: "text", Body: "oi",
	})
	if err != nil {
		t.Fatalf("postJSON: %v", err)
	}
	if res.ProviderMessageID != "msg_1" {
		t.Fatalf("id: %q", res.ProviderMessageID)
	}
	if res.AcceptedAt.IsZero() {
		t.Fatal("accepted_at not set")
	}
}

func TestPostJSON_AuthFailureMapsToErrAuthFailed(t *testing.T) {
	t.Parallel()
	for _, code := range []int{http.StatusUnauthorized, http.StatusForbidden} {
		code := code
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(code)
		}))
		p := newProvider(srv.URL)
		_, err := p.postJSON(context.Background(), "k", "/messages", outboundBody{Recipient: "1", Body: "x"})
		if !errors.Is(err, integration.ErrAuthFailed) {
			t.Errorf("code %d: expected ErrAuthFailed, got %v", code, err)
		}
		srv.Close()
	}
}

func TestPostJSON_RateLimitMapsToErrRateLimited(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer srv.Close()

	p := newProvider(srv.URL)
	_, err := p.postJSON(context.Background(), "k", "/messages", outboundBody{Recipient: "1", Body: "x"})
	if !errors.Is(err, integration.ErrRateLimited) {
		t.Fatalf("expected ErrRateLimited, got %v", err)
	}
}

func TestPostJSON_5xxMapsToErrUnreachable(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	p := newProvider(srv.URL)
	_, err := p.postJSON(context.Background(), "k", "/messages", outboundBody{Recipient: "1", Body: "x"})
	if !errors.Is(err, integration.ErrUnreachable) {
		t.Fatalf("expected ErrUnreachable, got %v", err)
	}
}

func TestPostJSON_ProviderErrorBody(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		// 200 with error body — provider-defined failure mode.
		_, _ = w.Write([]byte(`{"error":{"code":"INVALID_CHANNEL","message":"channel not found"}}`))
	}))
	defer srv.Close()

	p := newProvider(srv.URL)
	_, err := p.postJSON(context.Background(), "k", "/messages", outboundBody{Recipient: "1", Body: "x"})
	if err == nil || !strings.Contains(err.Error(), "INVALID_CHANNEL") {
		t.Fatalf("expected INVALID_CHANNEL error, got %v", err)
	}
}

func TestNormalizeKind(t *testing.T) {
	t.Parallel()
	cases := map[string]string{
		"":         "text",
		"text":     "text",
		"IMAGE":    "image",
		"audio":    "audio",
		"video":    "video",
		"document": "document",
		"file":     "document",
		"random":   "text",
	}
	for in, want := range cases {
		if got := normalizeKind(in); got != want {
			t.Errorf("normalizeKind(%q) = %q, want %q", in, got, want)
		}
	}
}
