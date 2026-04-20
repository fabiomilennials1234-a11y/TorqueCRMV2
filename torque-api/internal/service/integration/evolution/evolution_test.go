package evolution_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/milennials/torque-api/internal/service/integration"
	"github.com/milennials/torque-api/internal/service/integration/evolution"
)

func TestNew_RejectsBadConfig(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		cfg  evolution.Config
	}{
		{"empty base", evolution.Config{APIKey: "k"}},
		{"http base", evolution.Config{BaseURL: "http://evo.example.com", APIKey: "k"}},
		{"no api key", evolution.Config{BaseURL: "https://evo.example.com"}},
	}
	for _, c := range cases {
		if _, err := evolution.New(c.cfg); err == nil {
			t.Errorf("%s: expected error", c.name)
		}
	}
}

func TestSendMessage_HappyPath(t *testing.T) {
	t.Parallel()
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/message/sendText/inst-1" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Header.Get("apikey") != "secret" {
			t.Errorf("missing apikey header")
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"key":{"id":"wamid.XYZ"}}`))
	}))
	defer srv.Close()

	p, err := evolution.New(evolution.Config{
		BaseURL:    strings.Replace(srv.URL, "http://", "https://", 1),
		APIKey:     "secret",
		HTTPClient: srv.Client(),
	})
	if err != nil {
		t.Fatalf("new: %v", err)
	}

	res, err := p.SendMessage(context.Background(), integration.OutboundMessage{
		ChannelExternalID: "inst-1",
		Recipient:         "+5511999999999",
		Kind:              "text",
		Body:              "oi",
	})
	if err != nil {
		t.Fatalf("send: %v", err)
	}
	if res.ProviderMessageID != "wamid.XYZ" {
		t.Fatalf("unexpected id: %s", res.ProviderMessageID)
	}
}

func TestSendMessage_AuthFailure(t *testing.T) {
	t.Parallel()
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":"bad key"}`))
	}))
	defer srv.Close()

	p, err := evolution.New(evolution.Config{
		BaseURL:    strings.Replace(srv.URL, "http://", "https://", 1),
		APIKey:     "wrong",
		HTTPClient: srv.Client(),
	})
	if err != nil {
		t.Fatalf("new: %v", err)
	}
	_, err = p.SendMessage(context.Background(), integration.OutboundMessage{
		ChannelExternalID: "inst-1", Recipient: "+55", Kind: "text", Body: "x",
	})
	if !errors.Is(err, integration.ErrAuthFailed) {
		t.Fatalf("want ErrAuthFailed, got %v", err)
	}
}

func TestSendMessage_UnsupportedKind(t *testing.T) {
	t.Parallel()
	p, err := evolution.New(evolution.Config{
		BaseURL: "https://x", APIKey: "k",
	})
	if err != nil {
		t.Fatalf("new: %v", err)
	}
	_, err = p.SendMessage(context.Background(), integration.OutboundMessage{
		ChannelExternalID: "inst-1", Recipient: "+55", Kind: "sticker",
	})
	if !errors.Is(err, integration.ErrUnsupported) {
		t.Fatalf("want ErrUnsupported, got %v", err)
	}
}

func TestName(t *testing.T) {
	t.Parallel()
	p, _ := evolution.New(evolution.Config{BaseURL: "https://x", APIKey: "k"})
	if p.Name() != "evolution" {
		t.Fatalf("name: %s", p.Name())
	}
}
