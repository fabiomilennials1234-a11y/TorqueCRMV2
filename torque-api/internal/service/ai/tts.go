package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// ---------- provider-agnostic TTS surface ---------------------------

// TTS is the interface the playground-preview handler + future
// outbound-audio worker talk to. Provider swap = replace this impl.
type TTS interface {
	Name() string
	// Synthesize renders `text` as mp3 audio using `voiceID`. Returns
	// the complete audio payload in memory (agents cap text at 32 KB
	// per turn — mp3 at 44 kHz mono ~32 KB/s = ~1 MB for a 30s reply).
	Synthesize(ctx context.Context, voiceID, text string) ([]byte, error)
}

// ---------- ElevenLabs adapter --------------------------------------

// ElevenLabsConfig carries the API deps. BaseURL defaults are set by
// the caller (internal/config reads `ELEVENLABS_BASE_URL`).
type ElevenLabsConfig struct {
	BaseURL    string
	APIKey     string
	// ModelID is the ElevenLabs model identifier (e.g. "eleven_multilingual_v2",
	// "eleven_turbo_v2_5"). Default when empty: eleven_multilingual_v2 — our
	// user base is PT-BR so the multilingual model has better phoneme
	// coverage than the monolingual English default.
	ModelID    string
	Timeout    time.Duration
	HTTPClient *http.Client
}

// ElevenLabsTTS is the concrete provider.
type ElevenLabsTTS struct {
	cfg    ElevenLabsConfig
	client *http.Client
}

// NewElevenLabsTTS validates config. BaseURL must be https:// to keep
// parity with the rest of the system's outbound allowlist (S29 invariant).
func NewElevenLabsTTS(cfg ElevenLabsConfig) (*ElevenLabsTTS, error) {
	base := strings.TrimRight(cfg.BaseURL, "/")
	if base == "" {
		return nil, errors.New("elevenlabs: BaseURL required")
	}
	if !strings.HasPrefix(base, "https://") {
		return nil, errors.New("elevenlabs: BaseURL must use https://")
	}
	if cfg.APIKey == "" {
		return nil, errors.New("elevenlabs: APIKey required")
	}
	if cfg.ModelID == "" {
		cfg.ModelID = "eleven_multilingual_v2"
	}
	if cfg.Timeout <= 0 {
		cfg.Timeout = 20 * time.Second
	}
	client := cfg.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: cfg.Timeout}
	}
	cfg.BaseURL = base
	return &ElevenLabsTTS{cfg: cfg, client: client}, nil
}

// Name implements TTS.
func (*ElevenLabsTTS) Name() string { return "elevenlabs" }

// Synthesize POSTs to /v1/text-to-speech/:voice_id and returns the
// mp3 bytes. Enforces provider error taxonomy (401→ErrAuthFailed,
// 429→ErrRateLimited, 5xx→ErrProviderUnavailable, 4xx→ErrBadRequest).
func (p *ElevenLabsTTS) Synthesize(ctx context.Context, voiceID, text string) ([]byte, error) {
	if voiceID == "" {
		return nil, fmt.Errorf("%w: voice_id is required", ErrBadRequest)
	}
	if text == "" {
		return nil, fmt.Errorf("%w: text is required", ErrBadRequest)
	}
	// Provider cap: refuse > 5000 chars client-side. ElevenLabs limits
	// ~5000 chars per request and bills per char; hitting provider
	// rejection is a poor DX compared to a fast local refuse.
	if len(text) > 5000 {
		return nil, fmt.Errorf("%w: text exceeds 5000 chars (%d)", ErrBadRequest, len(text))
	}

	body, err := json.Marshal(map[string]any{
		"text":     text,
		"model_id": p.cfg.ModelID,
		// voice_settings left at provider defaults (stability/similarity).
		// Tunable per-agent in a follow-up if needed.
	})
	if err != nil {
		return nil, fmt.Errorf("encode tts request: %w", err)
	}

	url := fmt.Sprintf("%s/v1/text-to-speech/%s", p.cfg.BaseURL, voiceID)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("build tts request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "audio/mpeg")
	req.Header.Set("xi-api-key", p.cfg.APIKey)

	res, err := p.client.Do(req)
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

	// Cap at 5 MB — a sane ceiling for a single reply. Larger responses
	// are either a provider bug or an oversized text that slipped past
	// the 5000-char guard above.
	const maxAudio = 5 * 1024 * 1024
	buf, err := io.ReadAll(io.LimitReader(res.Body, maxAudio+1))
	if err != nil {
		return nil, fmt.Errorf("read tts response: %w", err)
	}
	if len(buf) > maxAudio {
		return nil, fmt.Errorf("%w: tts response > 5 MB", ErrBadRequest)
	}
	return buf, nil
}

// ---------- MockTTS for dev + tests ---------------------------------

// MockTTS returns deterministic fake mp3 bytes — a 4-byte header
// "MP3?" + sha256 of (voice|text). Plays as silence-or-noise in real
// decoders; the browser Audio element in the Playground preview will
// fail silently. Swap for ElevenLabsTTS as soon as ELEVENLABS_API_KEY
// is set.
type MockTTS struct{}

// NewMockTTS returns a ready mock.
func NewMockTTS() *MockTTS { return &MockTTS{} }

// Name implements TTS.
func (*MockTTS) Name() string { return "mock-tts" }

// Synthesize returns a tiny deterministic blob. Enough to verify the
// download path + content-type without calling the real provider.
func (*MockTTS) Synthesize(_ context.Context, voiceID, text string) ([]byte, error) {
	if voiceID == "" {
		return nil, fmt.Errorf("%w: voice_id required", ErrBadRequest)
	}
	if text == "" {
		return nil, fmt.Errorf("%w: text required", ErrBadRequest)
	}
	// Shape: "MP3?" + hex-encoded first 16 bytes of sha256(voice+"|"+text).
	sum := deterministicEmbedding(voiceID + "|" + text) // reuse hash path
	out := make([]byte, 0, 4+64)
	out = append(out, 'M', 'P', '3', '?')
	// Take first 16 float32s -> 64 bytes. Totally bogus as audio, but
	// deterministic + cheap + never zero-length.
	for i := 0; i < 16; i++ {
		bits := sum[i]
		// Cast float32 to int32 via IEEE bits to get stable bytes.
		b := fmt.Sprintf("%08x", int32(bits*1e6))
		out = append(out, []byte(b)...)
	}
	return out, nil
}
