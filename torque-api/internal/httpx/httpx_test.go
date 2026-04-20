package httpx_test

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/milennials/torque-api/internal/httpx"
)

type simple struct {
	Name string `json:"name"`
}

func TestDecodeJSON_RejectsUnknownFields(t *testing.T) {
	t.Parallel()
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"name":"a","extra":1}`))
	var s simple
	if err := httpx.DecodeJSON(req, &s); err == nil {
		t.Fatal("expected error on unknown field")
	}
}

func TestDecodeJSON_HappyPath(t *testing.T) {
	t.Parallel()
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"name":"ok"}`))
	var s simple
	if err := httpx.DecodeJSON(req, &s); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if s.Name != "ok" {
		t.Fatalf("name = %q", s.Name)
	}
}

func TestDecodeJSON_OversizedBodyReportsTooLarge(t *testing.T) {
	t.Parallel()
	// Construct a payload > MaxBodyBytes. Use a long string field.
	big := bytes.Buffer{}
	big.WriteString(`{"name":"`)
	big.WriteString(strings.Repeat("x", int(httpx.MaxBodyBytes)+10))
	big.WriteString(`"}`)

	req := httptest.NewRequest(http.MethodPost, "/", &big)
	var s simple
	err := httpx.DecodeJSON(req, &s)
	if err == nil {
		t.Fatal("expected error for oversized body")
	}
	if !httpx.IsBodyTooLarge(err) {
		t.Fatalf("expected IsBodyTooLarge true, got err=%v", err)
	}
}
