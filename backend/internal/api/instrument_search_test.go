package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	jwtmiddleware "github.com/hondyman/uisce/libs/jwt-middleware"
)

func withInstrumentTestAuth(r *http.Request, userID, tenantID string) *http.Request {
	claims := &jwtmiddleware.JWTClaims{
		UserID:   userID,
		TenantID: tenantID,
	}
	ctx := context.WithValue(r.Context(), jwtmiddleware.ClaimsContextKey, claims)
	return r.WithContext(ctx)
}

// 1. Unauthenticated -> 401
func TestInstrumentSearch_Unauthenticated(t *testing.T) {
	db, _, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	handler := NewInstrumentSearchHandler(db)
	req := httptest.NewRequest(http.MethodGet, "/api/instruments/search?q=AAPL", nil)
	rec := httptest.NewRecorder()

	handler.Search(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 Unauthorized, got %d", rec.Code)
	}
}

// 2. Empty query -> empty list
func TestInstrumentSearch_EmptyQuery(t *testing.T) {
	db, _, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	handler := NewInstrumentSearchHandler(db)
	req := httptest.NewRequest(http.MethodGet, "/api/instruments/search?q=", nil)
	req = withInstrumentTestAuth(req, "user-1", "11111111-1111-1111-1111-111111111111")
	rec := httptest.NewRecorder()

	handler.Search(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", rec.Code)
	}

	var resp InstrumentSearchResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed decoding json: %v", err)
	}
	if len(resp.Results) != 0 {
		t.Fatalf("expected empty results, got %d", len(resp.Results))
	}
}

// 3. Invalid asset_class -> 400 Bad Request
func TestInstrumentSearch_InvalidAssetClass(t *testing.T) {
	db, _, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	handler := NewInstrumentSearchHandler(db)
	req := httptest.NewRequest(http.MethodGet, "/api/instruments/search?q=AAPL&asset_class=crypto_token", nil)
	req = withInstrumentTestAuth(req, "user-1", "11111111-1111-1111-1111-111111111111")
	rec := httptest.NewRecorder()

	handler.Search(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 Bad Request, got %d", rec.Code)
	}
}

// 4. Exact Ticker & ISIN ranking priority with tenant override
func TestInstrumentSearch_RankingAndTenantOverride(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	tenantID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	aaplID := uuid.New()
	aaplCustomID := uuid.New()

	// Expect query with tenantID, asset_class="", query="AAPL", limit=10
	mock.ExpectQuery(regexp.QuoteMeta("WITH ranked_securities AS (")).
		WithArgs(tenantID, "", "AAPL", 10).
		WillReturnRows(
			sqlmock.NewRows([]string{"id", "ticker", "security_name", "isin", "subtype_code", "exchange_mic", "match_rank"}).
				AddRow(aaplCustomID, "AAPL", "Apple Inc. (Custom Tenant)", "US0378331005", "equity", "XNAS", 1).
				AddRow(aaplID, "AAPLW", "Apple Warrant", "US0378331099", "equity", "XNAS", 3),
		)

	handler := NewInstrumentSearchHandler(db)
	req := httptest.NewRequest(http.MethodGet, "/api/instruments/search?q=AAPL", nil)
	req = withInstrumentTestAuth(req, "user-1", tenantID.String())
	rec := httptest.NewRecorder()

	handler.Search(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp InstrumentSearchResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed decoding json: %v", err)
	}
	if len(resp.Results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(resp.Results))
	}

	// Exact match ranks first
	if resp.Results[0].Symbol != "AAPL" || resp.Results[0].Name != "Apple Inc. (Custom Tenant)" {
		t.Fatalf("expected custom tenant AAPL to rank first, got %v", resp.Results[0])
	}
	if resp.Results[0].ISIN != "US0378331005" {
		t.Fatalf("expected ISIN US0378331005, got %s", resp.Results[0].ISIN)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sqlmock expectations: %v", err)
	}
}

// 5. Limit cap clamped to max 20
func TestInstrumentSearch_LimitClamping(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	tenantID := uuid.MustParse("11111111-1111-1111-1111-111111111111")

	// Limit requested is 100, but SQL query must receive 20
	mock.ExpectQuery(regexp.QuoteMeta("WITH ranked_securities AS (")).
		WithArgs(tenantID, "equity", "TSLA", 20).
		WillReturnRows(
			sqlmock.NewRows([]string{"id", "ticker", "security_name", "isin", "subtype_code", "exchange_mic", "match_rank"}).
				AddRow(uuid.New(), "TSLA", "Tesla Inc.", "US88160R1014", "equity", "XNAS", 1),
		)

	handler := NewInstrumentSearchHandler(db)
	req := httptest.NewRequest(http.MethodGet, "/api/instruments/search?q=TSLA&limit=100&asset_class=equity", nil)
	req = withInstrumentTestAuth(req, "user-1", tenantID.String())
	rec := httptest.NewRecorder()

	handler.Search(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", rec.Code)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sqlmock expectations: %v", err)
	}
}

// 6. Benchmark & execution latency assertion (<100ms) with mocked high-density response
func TestInstrumentSearch_LatencyBudget(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	tenantID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	rows := sqlmock.NewRows([]string{"id", "ticker", "security_name", "isin", "subtype_code", "exchange_mic", "match_rank"})
	for i := 0; i < 20; i++ {
		rows.AddRow(uuid.New(), fmt.Sprintf("SYM%02d", i), fmt.Sprintf("Synthetic Security %02d", i), fmt.Sprintf("US999999%04d", i), "equity", "XNAS", 3)
	}

	mock.ExpectQuery(regexp.QuoteMeta("WITH ranked_securities AS (")).
		WithArgs(tenantID, "", "SYM", 20).
		WillReturnRows(rows)

	handler := NewInstrumentSearchHandler(db)
	req := httptest.NewRequest(http.MethodGet, "/api/instruments/search?q=SYM&limit=20", nil)
	req = withInstrumentTestAuth(req, "user-1", tenantID.String())
	rec := httptest.NewRecorder()

	start := time.Now()
	handler.Search(rec, req)
	elapsed := time.Since(start)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", rec.Code)
	}
	if elapsed > 100*time.Millisecond {
		t.Fatalf("latency exceeded 100ms budget: took %v", elapsed)
	}
}
