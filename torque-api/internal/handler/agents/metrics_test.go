package agents

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// Covers param-validation paths for the metrics handler. The
// aggregation itself needs a live DB — integration tests land in a
// runtime-validation pass when Go is installed on the host.

func TestMetricsHandler_ParamValidation(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name     string
		qs       string
		wantCode string
	}{
		{"invalid since", "since=not-a-date", "INVALID_SINCE"},
		{"invalid until", "since=2026-01-01T00:00:00Z&until=blah", "INVALID_UNTIL"},
		{
			"inverted window",
			"since=2026-02-01T00:00:00Z&until=2026-01-01T00:00:00Z",
			"INVALID_WINDOW",
		},
		{
			"window > 365 days",
			"since=2024-01-01T00:00:00Z&until=2026-01-02T00:00:00Z",
			"WINDOW_TOO_LARGE",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			req := httptest.NewRequest(http.MethodGet, "/metrics?"+c.qs, nil)
			// Install the chi route context so parseID resolves
			// without hitting INVALID_ID.
			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("id", uuid.New().String())
			req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
			w := httptest.NewRecorder()

			// Repo is unused on the validation paths we care about
			// — every case 400s before touching the DB.
			h := &Handler{}
			h.agentMetrics(w, req)

			res := w.Result()
			defer res.Body.Close()
			if res.StatusCode != http.StatusBadRequest {
				t.Fatalf("want 400, got %d", res.StatusCode)
			}
			var body struct {
				Error struct {
					Code string `json:"code"`
				} `json:"error"`
			}
			_ = json.NewDecoder(res.Body).Decode(&body)
			if body.Error.Code != c.wantCode {
				// Fallback parse — the error-envelope shape in this
				// project varies by helper; tolerate alternative
				// response shape as long as the substring appears.
				raw := w.Body.String()
				if !strings.Contains(raw, c.wantCode) {
					t.Errorf("want code %s in body, got %q", c.wantCode, raw)
				}
			}
		})
	}
}
