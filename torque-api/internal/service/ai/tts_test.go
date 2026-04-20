package ai_test

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/milennials/torque-api/internal/service/ai"
)

// --------------- construction --------------------------------------

func TestNewElevenLabsTTS_RejectsBadConfig(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		cfg  ai.ElevenLabsConfig
	}{
		{"empty base", ai.ElevenLabsConfig{APIKey: "k"}},
		{"http base", ai.ElevenLabsConfig{BaseURL: "http://x", APIKey: "k"}},
		{"no api key", ai.ElevenLabsConfig{BaseURL: "https://x"}},
	}
	for _, c := range cases {
		if _, err := ai.NewElevenLabsTTS(c.cfg); err == nil {
			t.Errorf("%s: expected error", c.name)
		}
	}
}

func TestElevenLabsTTS_Name(t *testing.T) {
	t.Parallel()
	p, err := ai.NewElevenLabsTTS(ai.ElevenLabsConfig{BaseURL: "https://x", APIKey: "k"})
	if err != nil {
		t.Fatalf("new: %v", err)
	}
	if p.Name() != "elevenlabs" {
		t.Errorf("got %s", p.Name())
	}
}

// --------------- happy path ----------------------------------------

func TestElevenLabsTTS_Synthesize_Happy(t *testing.T) {
	t.Parallel()
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("xi-api-key") != "secret" {
			t.Errorf("missing api key header")
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("wrong content-type")
		}
		if !strings.Contains(r.URL.Path, "/v1/text-to-speech/VOICE1") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "audio/mpeg")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte{0xff, 0xfb, 0x90, 0x64}) // fake mp3 preamble
	}))
	defer srv.Close()

	p, err := ai.NewElevenLabsTTS(ai.ElevenLabsConfig{
		BaseURL: srv.URL, APIKey: "secret", HTTPClient: srv.Client(),
	})
	if err != nil {
		t.Fatalf("new: %v", err)
	}
	got, err := p.Synthesize(context.Background(), "VOICE1", "Olá")
	if err != nil {
		t.Fatalf("synth: %v", err)
	}
	if !bytes.Equal(got, []byte{0xff, 0xfb, 0x90, 0x64}) {
		t.Errorf("unexpected bytes: %v", got)
	}
}

// --------------- error paths --------------------------------------

func TestElevenLabsTTS_Synthesize_RequiresVoiceAndText(t *testing.T) {
	t.Parallel()
	p, _ := ai.NewElevenLabsTTS(ai.ElevenLabsConfig{BaseURL: "https://x", APIKey: "k"})
	if _, err := p.Synthesize(context.Background(), "", "hi"); !errors.Is(err, ai.ErrBadRequest) {
		t.Errorf("empty voice should be ErrBadRequest")
	}
	if _, err := p.Synthesize(context.Background(), "v", ""); !errors.Is(err, ai.ErrBadRequest) {
		t.Errorf("empty text should be ErrBadRequest")
	}
}

func TestElevenLabsTTS_Synthesize_RejectsOversizeText(t *testing.T) {
	t.Parallel()
	p, _ := ai.NewElevenLabsTTS(ai.ElevenLabsConfig{BaseURL: "https://x", APIKey: "k"})
	huge := strings.Repeat("a", 5001)
	if _, err := p.Synthesize(context.Background(), "v", huge); !errors.Is(err, ai.ErrBadRequest) {
		t.Errorf("> 5000 chars should be ErrBadRequest")
	}
}

func TestElevenLabsTTS_Synthesize_AuthFailed(t *testing.T) {
	t.Parallel()
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer srv.Close()
	p, _ := ai.NewElevenLabsTTS(ai.ElevenLabsConfig{
		BaseURL: srv.URL, APIKey: "k", HTTPClient: srv.Client(),
	})
	_, err := p.Synthesize(context.Background(), "v", "t")
	if !errors.Is(err, ai.ErrAuthFailed) {
		t.Errorf("want ErrAuthFailed, got %v", err)
	}
}

func TestElevenLabsTTS_Synthesize_RateLimited(t *testing.T) {
	t.Parallel()
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer srv.Close()
	p, _ := ai.NewElevenLabsTTS(ai.ElevenLabsConfig{
		BaseURL: srv.URL, APIKey: "k", HTTPClient: srv.Client(),
	})
	_, err := p.Synthesize(context.Background(), "v", "t")
	if !errors.Is(err, ai.ErrRateLimited) {
		t.Errorf("want ErrRateLimited, got %v", err)
	}
}

func TestElevenLabsTTS_Synthesize_ProviderUnavailable(t *testing.T) {
	t.Parallel()
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()
	p, _ := ai.NewElevenLabsTTS(ai.ElevenLabsConfig{
		BaseURL: srv.URL, APIKey: "k", HTTPClient: srv.Client(),
	})
	_, err := p.Synthesize(context.Background(), "v", "t")
	if !errors.Is(err, ai.ErrProviderUnavailable) {
		t.Errorf("want ErrProviderUnavailable, got %v", err)
	}
}

func TestElevenLabsTTS_Synthesize_BadRequestFromProvider(t *testing.T) {
	t.Parallel()
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte("invalid voice_id"))
	}))
	defer srv.Close()
	p, _ := ai.NewElevenLabsTTS(ai.ElevenLabsConfig{
		BaseURL: srv.URL, APIKey: "k", HTTPClient: srv.Client(),
	})
	_, err := p.Synthesize(context.Background(), "v", "t")
	if !errors.Is(err, ai.ErrBadRequest) {
		t.Errorf("want ErrBadRequest, got %v", err)
	}
}

// --------------- MockTTS -------------------------------------------

func TestMockTTS_Deterministic(t *testing.T) {
	t.Parallel()
	m := ai.NewMockTTS()
	a, err := m.Synthesize(context.Background(), "v1", "hello")
	if err != nil {
		t.Fatalf("synth: %v", err)
	}
	b, _ := m.Synthesize(context.Background(), "v1", "hello")
	if !bytes.Equal(a, b) {
		t.Errorf("expected deterministic output")
	}
	c, _ := m.Synthesize(context.Background(), "v1", "world")
	if bytes.Equal(a, c) {
		t.Errorf("expected different output for different text")
	}
	if !bytes.HasPrefix(a, []byte("MP3?")) {
		t.Errorf("mock output missing prefix")
	}
}

func TestMockTTS_RequiresInputs(t *testing.T) {
	t.Parallel()
	m := ai.NewMockTTS()
	if _, err := m.Synthesize(context.Background(), "", "t"); !errors.Is(err, ai.ErrBadRequest) {
		t.Errorf("empty voice should be ErrBadRequest")
	}
	if _, err := m.Synthesize(context.Background(), "v", ""); !errors.Is(err, ai.ErrBadRequest) {
		t.Errorf("empty text should be ErrBadRequest")
	}
}
