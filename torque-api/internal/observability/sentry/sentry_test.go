package sentry

import (
	"strings"
	"testing"

	sentrygo "github.com/getsentry/sentry-go"
)

func TestScrubPII_HeadersAndCookies(t *testing.T) {
	t.Parallel()
	evt := &sentrygo.Event{
		Request: &sentrygo.Request{
			Cookies: "__torque_session=abc; __torque_csrf=xyz",
			Headers: map[string]string{
				"Authorization":  "Bearer hunter2",
				"Cookie":         "session=xyz",
				"X-CSRF-Token":   "deadbeef",
				"User-Agent":     "Mozilla/5.0",
				"X-Request-ID":   "req-123",
			},
			QueryString: "q=hello&api_key=secret&other=ok",
			Data:        `{"password":"hunter2"}`,
		},
		User: sentrygo.User{
			Email:     "user@example.com",
			IPAddress: "203.0.113.42",
		},
	}

	out := scrubPII(evt, nil)
	if out.Request.Cookies != "[Scrubbed]" {
		t.Fatalf("cookies not scrubbed: %q", out.Request.Cookies)
	}
	if out.Request.Headers["Authorization"] != "[Scrubbed]" {
		t.Fatalf("Authorization not scrubbed: %v", out.Request.Headers)
	}
	if out.Request.Headers["Cookie"] != "[Scrubbed]" {
		t.Fatalf("Cookie not scrubbed: %v", out.Request.Headers)
	}
	if out.Request.Headers["User-Agent"] != "Mozilla/5.0" {
		t.Fatalf("non-sensitive headers must be preserved, got %v", out.Request.Headers)
	}
	if !strings.Contains(out.Request.QueryString, "api_key=[Scrubbed]") {
		t.Fatalf("api_key not scrubbed in query: %q", out.Request.QueryString)
	}
	if !strings.Contains(out.Request.QueryString, "q=hello") {
		t.Fatalf("non-sensitive query dropped: %q", out.Request.QueryString)
	}
	if out.Request.Data != "[Scrubbed]" {
		t.Fatalf("body not scrubbed: %v", out.Request.Data)
	}
	if out.User.Email != "[redacted]" {
		t.Fatalf("user email not redacted: %q", out.User.Email)
	}
	if out.User.IPAddress != "203.0.113.0/24" {
		t.Fatalf("IP not truncated: %q", out.User.IPAddress)
	}
}

func TestScrubPII_IPv6Truncation(t *testing.T) {
	t.Parallel()
	evt := &sentrygo.Event{User: sentrygo.User{IPAddress: "2001:db8:85a3::8a2e:370:7334"}}
	out := scrubPII(evt, nil)
	if !strings.HasPrefix(out.User.IPAddress, "2001:db8:85a3") {
		t.Fatalf("IPv6 must keep the first 3 hextets, got %q", out.User.IPAddress)
	}
}

func TestScrubPII_NilEvent(t *testing.T) {
	t.Parallel()
	if scrubPII(nil, nil) != nil {
		t.Fatal("nil event must return nil")
	}
}

func TestIsSensitive(t *testing.T) {
	t.Parallel()
	for _, k := range []string{"password", "API_KEY", "X-CSRF-Token", "Cookie", "email"} {
		if !isSensitive(k) {
			t.Errorf("%q must be sensitive", k)
		}
	}
	for _, k := range []string{"User-Agent", "X-Request-ID", "Content-Type"} {
		if isSensitive(k) {
			t.Errorf("%q must NOT be sensitive", k)
		}
	}
}
