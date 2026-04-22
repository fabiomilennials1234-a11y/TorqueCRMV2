package meta

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/milennials/torque-api/internal/service/integration"
)

// We exercise the HTTP + envelope layer directly via (*Provider).get +
// (*Provider).fetchInsights. Store-dependent flows are covered at the
// handler / repo integration tier (a real pgxpool). Unit tests here
// own the wire contract: Graph API field names, error code mapping,
// decimal → cents parsing.

func newProvider(base, appSecret string) *Provider {
	return New(Config{BaseURL: base, AppSecret: appSecret}, nil, nil, nil)
}

// ---- number helpers -------------------------------------------------

func TestDecimalToCents(t *testing.T) {
	t.Parallel()
	cases := []struct {
		in   string
		want int64
	}{
		{"12.34", 1234},
		{"12", 1200},
		{"12.3", 1230},
		{"0", 0},
		{"", 0},
		{"  9.99  ", 999},
		{"-1.50", -150},
		{"12.3456", 1234}, // truncate beyond 2 decimals
		{"abc", 0},
	}
	for _, c := range cases {
		if got := decimalToCents(c.in); got != c.want {
			t.Errorf("decimalToCents(%q) = %d, want %d", c.in, got, c.want)
		}
	}
}

func TestSumLeadActions(t *testing.T) {
	t.Parallel()
	// Action buckets from Graph API. We count anything whose type
	// string contains "lead" (case-insensitive).
	got := sumLeadActions([]graphAction{
		{ActionType: "lead", Value: "3"},
		{ActionType: "onsite_conversion.lead_grouped", Value: "2"},
		{ActionType: "link_click", Value: "10"}, // ignored
		{ActionType: "LEAD", Value: "1"},        // case-insensitive
	})
	if got != 6 {
		t.Fatalf("sumLeadActions: got %d, want 6", got)
	}
}

func TestCPLead(t *testing.T) {
	t.Parallel()
	// spend 1000 cents, 5 leads → 200 cents CPL
	if got := cpLead(1000, 5); got != 200 {
		t.Fatalf("cpLead(1000, 5) = %d", got)
	}
	// Zero leads returns -1 ("unavailable") to distinguish from free.
	if got := cpLead(1000, 0); got != -1 {
		t.Fatalf("cpLead(1000, 0) = %d, want -1", got)
	}
}

// ---- appsecret_proof -----------------------------------------------

func TestAppsecretProof_EmptyWhenNoSecret(t *testing.T) {
	t.Parallel()
	p := newProvider("", "")
	if got := p.appsecretProof("token"); got != "" {
		t.Fatalf("proof without app secret: %q", got)
	}
}

func TestAppsecretProof_HMACOverToken(t *testing.T) {
	t.Parallel()
	p := newProvider("", "app-secret")
	if got := p.appsecretProof("abc"); got == "" || len(got) != 64 {
		// SHA-256 is 32 bytes = 64 hex chars.
		t.Fatalf("proof wrong shape: %q", got)
	}
}

// ---- fetchInsights --------------------------------------------------

func TestGet_HappyPath(t *testing.T) {
	t.Parallel()
	var calls int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if !hasPrefix(r.URL.Path, "/act_123/insights") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		q := r.URL.Query()
		if q.Get("access_token") != "tok" {
			t.Errorf("token missing: %v", q)
		}
		if q.Get("level") != "account" {
			t.Errorf("level: %q", q.Get("level"))
		}
		// Graph preset must travel through as-is.
		if q.Get("date_preset") != "last_7d" {
			t.Errorf("date_preset: %q", q.Get("date_preset"))
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":[{
		  "account_id":"123","account_currency":"BRL",
		  "spend":"100.50","impressions":"1000","clicks":"50",
		  "actions":[{"action_type":"lead","value":"5"}]
		}]}`))
	}))
	defer srv.Close()

	p := newProvider(srv.URL, "")
	rows, err := p.get(context.Background(), "tok", "123", insightParams{
		level: "account", datePreset: "last_7d",
	})
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("rows: %d", len(rows))
	}
	if rows[0].Spend != "100.50" {
		t.Fatalf("spend: %q", rows[0].Spend)
	}
	if calls != 1 {
		t.Fatalf("calls: %d", calls)
	}
}

func TestGet_MapsOAuthErrorToAuthFailed(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":{"code":190,"message":"Invalid OAuth 2.0 Access Token","type":"OAuthException"}}`))
	}))
	defer srv.Close()

	p := newProvider(srv.URL, "")
	_, err := p.get(context.Background(), "tok", "123", insightParams{level: "account", datePreset: "last_7d"})
	if !errors.Is(err, integration.ErrAuthFailed) {
		t.Fatalf("expected ErrAuthFailed, got %v", err)
	}
}

func TestGet_MapsRateLimitToRateLimited(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"error":{"code":17,"message":"User request limit reached","type":"OAuthException"}}`))
	}))
	defer srv.Close()

	p := newProvider(srv.URL, "")
	_, err := p.get(context.Background(), "tok", "123", insightParams{level: "account", datePreset: "last_7d"})
	if !errors.Is(err, integration.ErrRateLimited) {
		t.Fatalf("expected ErrRateLimited, got %v", err)
	}
}

func TestFetchInsights_BuildsRollup(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		w.Header().Set("Content-Type", "application/json")
		switch q.Get("level") {
		case "account":
			_, _ = w.Write([]byte(`{"data":[{
			  "account_id":"123","account_currency":"BRL",
			  "spend":"200.00","impressions":"2000","clicks":"100",
			  "actions":[{"action_type":"lead","value":"10"}]
			}]}`))
		case "campaign":
			_, _ = w.Write([]byte(`{"data":[
			  {"campaign_id":"c1","campaign_name":"Promo",
			   "spend":"150.00","impressions":"1500","clicks":"80",
			   "actions":[{"action_type":"lead","value":"7"}]},
			  {"campaign_id":"c2","campaign_name":"Test",
			   "spend":"50.00","impressions":"500","clicks":"20",
			   "actions":[{"action_type":"lead","value":"3"}]}
			]}`))
		default:
			t.Errorf("unknown level: %s", q.Get("level"))
		}
	}))
	defer srv.Close()

	p := newProvider(srv.URL, "")
	out, err := p.fetchInsights(context.Background(), "tok", integration.InsightsWindow{
		AccountID: "123", DateRange: "last_7d",
	})
	if err != nil {
		t.Fatalf("fetchInsights: %v", err)
	}
	if out.SpendCents != 20000 {
		t.Fatalf("spend: %d", out.SpendCents)
	}
	if out.Leads != 10 {
		t.Fatalf("leads: %d", out.Leads)
	}
	if out.CPLeadCents != 2000 {
		t.Fatalf("cpl: %d", out.CPLeadCents)
	}
	if len(out.Campaigns) != 2 {
		t.Fatalf("campaigns: %d", len(out.Campaigns))
	}
	if out.Campaigns[0].Leads != 7 {
		t.Fatalf("c1 leads: %d", out.Campaigns[0].Leads)
	}
}

// hasPrefix is a tiny helper that sidesteps pulling in strings just
// for the single test assertion.
func hasPrefix(s, p string) bool { return len(s) >= len(p) && s[:len(p)] == p }
