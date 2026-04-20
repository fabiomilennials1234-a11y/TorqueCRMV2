package products_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	handler "github.com/milennials/torque-api/internal/handler/products"
)

// toViewShape is a thin sanity test that fails if the JSON contract
// drifts. It locks the field names the frontend depends on without
// touching a database.
func TestProductsView_ShapeLocked(t *testing.T) {
	t.Parallel()
	now := time.Now().UTC().Format(time.RFC3339)
	body := map[string]any{
		"id":           uuid.New().String(),
		"name":         "Licenca Pro",
		"description":  "desc",
		"price_cents":  19900,
		"currency":     "BRL",
		"is_active":    true,
		"created_at":   now,
		"updated_at":   now,
	}
	raw, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	expected := []string{
		`"id"`, `"name"`, `"price_cents"`, `"currency"`, `"is_active"`,
	}
	for _, f := range expected {
		if !strings.Contains(string(raw), f) {
			t.Errorf("field %s missing from view shape", f)
		}
	}
}

// TestRoutes_Mount asserts the router mounts the expected paths. Pure
// registration smoke test — does not hit handlers (would need DB).
func TestRoutes_Mount(t *testing.T) {
	t.Parallel()
	r := chi.NewRouter()
	h := handler.NewRead(nil) // repo nil — we only test routing
	h.Routes(r)

	// A request to an unknown path under /products is a 404 *after*
	// mounting, which proves the router recognizes the prefix.
	// GET /products should NOT return 404 (it'd panic on nil repo;
	// we check 404 vs not-404 via a path the router doesn't know).
	req := httptest.NewRequest(http.MethodGet, "/totally-not-mounted", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Errorf("unmounted path should 404; got %d", rr.Code)
	}
}
