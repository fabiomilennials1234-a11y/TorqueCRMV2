package tinyerp

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/google/uuid"

	"github.com/milennials/torque-api/internal/service/integration"
)

// fakeSink captures UpsertBySKU calls so we can assert what the sync
// pipeline forwarded to the product repo without booting a Postgres.
type fakeSink struct {
	mu       sync.Mutex
	inserted int
	updated  int
	rows     []fakeSinkRow
	// next determines what to return for each call.
	nextInserted bool
	returnErr    error
}

type fakeSinkRow struct {
	Name, SKU, Currency string
	PriceCents          int64
}

func (s *fakeSink) UpsertBySKU(_ context.Context, _ uuid.UUID, name, sku string,
	_ *string, priceCents int64, currency string,
) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.returnErr != nil {
		return false, s.returnErr
	}
	s.rows = append(s.rows, fakeSinkRow{Name: name, SKU: sku, PriceCents: priceCents, Currency: currency})
	// Alternate between inserts and updates to cover both paths.
	inserted := s.nextInserted
	s.nextInserted = !s.nextInserted
	if inserted {
		s.inserted++
	} else {
		s.updated++
	}
	return inserted, nil
}

func TestDecimalToCents_TinyERP(t *testing.T) {
	t.Parallel()
	cases := []struct {
		in   string
		want int64
	}{
		{"10.00", 1000},
		{"10,50", 1050}, // BR decimal separator
		{"0", 0},
		{"", 0},
		{"3.9", 390},
		{"-4.20", -420},
	}
	for _, c := range cases {
		if got := decimalToCents(c.in); got != c.want {
			t.Errorf("decimalToCents(%q) = %d, want %d", c.in, got, c.want)
		}
	}
}

func TestListProductsPage_OK(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/produtos.pesquisa.php" {
			t.Errorf("path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"retorno":{"status":"OK","pagina":1,"numero_paginas":2,"produtos":[
		  {"produto":{"id":"1","codigo":"A1","nome":"Aroma","preco":"12.50"}},
		  {"produto":{"id":"2","codigo":"A2","nome":"Bebida","preco":"25.00"}}
		]}}`))
	}))
	defer srv.Close()

	p := New(Config{BaseURL: srv.URL}, nil, nil, nil)
	rows, more, err := p.listProductsPage(context.Background(), "tok", 1)
	if err != nil {
		t.Fatalf("listProductsPage: %v", err)
	}
	if len(rows) != 2 {
		t.Fatalf("rows: %d", len(rows))
	}
	if rows[0].Codigo != "A1" || rows[1].Codigo != "A2" {
		t.Fatalf("rows: %+v", rows)
	}
	if !more {
		t.Fatal("expected more=true (page 1 of 2)")
	}
}

func TestListProductsPage_LastPageHasMoreFalse(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"retorno":{"status":"OK","pagina":3,"numero_paginas":3,"produtos":[]}}`))
	}))
	defer srv.Close()

	p := New(Config{BaseURL: srv.URL}, nil, nil, nil)
	_, more, err := p.listProductsPage(context.Background(), "tok", 3)
	if err != nil {
		t.Fatalf("listProductsPage: %v", err)
	}
	if more {
		t.Fatal("expected more=false (last page)")
	}
}

func TestListProductsPage_AuthError(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"retorno":{"status":"Erro","codigo_erro":6,"erros":[{"erro":"Token inválido"}]}}`))
	}))
	defer srv.Close()

	p := New(Config{BaseURL: srv.URL}, nil, nil, nil)
	_, _, err := p.listProductsPage(context.Background(), "tok", 1)
	if !errors.Is(err, integration.ErrAuthFailed) {
		t.Fatalf("expected ErrAuthFailed, got %v", err)
	}
}

// processSyncRows is a handler-side loop that mirrors
// Provider.SyncProducts inner body — extracted so we can exercise the
// sink forwarding without mocking the credential store.
func processSyncRows(rows []productoInner, sink *fakeSink) SyncProductsResult {
	var res SyncProductsResult
	for _, row := range rows {
		res.Fetched++
		if row.Codigo == "" {
			res.Skipped++
			continue
		}
		inserted, err := sink.UpsertBySKU(context.Background(), uuid.Nil,
			row.Nome, row.Codigo, nil, decimalToCents(row.Preco), "BRL")
		if err != nil {
			res.Skipped++
			continue
		}
		if inserted {
			res.Inserted++
		} else {
			res.Updated++
		}
	}
	return res
}

func TestSink_ForwardsRowsAndCountsInsertsVsUpdates(t *testing.T) {
	t.Parallel()
	rows := []productoInner{
		{Codigo: "A1", Nome: "Aroma", Preco: "10.00"},
		{Codigo: "", Nome: "No SKU", Preco: "5.00"}, // skipped
		{Codigo: "A2", Nome: "Bebida", Preco: "20.00"},
	}
	sink := &fakeSink{}
	res := processSyncRows(rows, sink)

	if res.Fetched != 3 {
		t.Errorf("fetched: %d", res.Fetched)
	}
	if res.Skipped != 1 {
		t.Errorf("skipped: %d", res.Skipped)
	}
	if res.Inserted+res.Updated != 2 {
		t.Errorf("ins+upd: %d", res.Inserted+res.Updated)
	}
	// Verify payload that crossed the sink boundary.
	if len(sink.rows) != 2 {
		t.Fatalf("sink rows: %d", len(sink.rows))
	}
	if sink.rows[0].SKU != "A1" || sink.rows[0].PriceCents != 1000 {
		t.Errorf("row 0: %+v", sink.rows[0])
	}
	if sink.rows[1].SKU != "A2" || sink.rows[1].PriceCents != 2000 {
		t.Errorf("row 1: %+v", sink.rows[1])
	}
}

func TestSink_ErrorCountsAsSkip(t *testing.T) {
	t.Parallel()
	sink := &fakeSink{returnErr: fmt.Errorf("boom")}
	res := processSyncRows([]productoInner{
		{Codigo: "A1", Nome: "X", Preco: "1.00"},
	}, sink)
	if res.Skipped != 1 || res.Inserted != 0 || res.Updated != 0 {
		t.Fatalf("unexpected result: %+v", res)
	}
}
