package vocabulary

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/jmoiron/sqlx"
	"github.com/prometheus/client_golang/prometheus/testutil"
)

func newTestResolver(t *testing.T) (*Resolver, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return &Resolver{db: sqlx.NewDb(db, "postgres")}, mock
}

func TestResolveTerm_ExactMatchHit(t *testing.T) {
	r, mock := newTestResolver(t)
	tenantID := "tenant-1"

	// resolveViaAlias: no rows.
	mock.ExpectQuery(`FROM catalog_node alias_n`).
		WillReturnRows(sqlmock.NewRows([]string{"term_id", "term_name", "matched_token", "matched_via", "tenant_id"}))

	// resolveViaSynonym: no rows.
	mock.ExpectQuery(`FROM catalog_node syn_n`).
		WillReturnRows(sqlmock.NewRows([]string{"term_id", "term_name", "matched_token", "matched_via", "tenant_id"}))

	// resolveViaBusinessTermName: one row.
	mock.ExpectQuery(`FROM catalog_node cn`).
		WillReturnRows(sqlmock.NewRows([]string{"term_id", "term_name", "matched_token", "matched_via", "tenant_id"}).
			AddRow("bt-1", "Portfolio", "portfolio", "DIRECT", tenantID))

	// enrichWithSemanticTerm: no bound semantic term (expected, not an error).
	mock.ExpectQuery(`FROM catalog_node st`).
		WillReturnError(sql.ErrNoRows)

	before := testutil.ToFloat64(ResolutionAttempts.WithLabelValues(tenantID, "business_term_name", "hit"))

	results, err := r.ResolveTerm(context.Background(), tenantID, "portfolio")
	if err != nil {
		t.Fatalf("ResolveTerm returned error: %v", err)
	}
	if len(results) != 1 || results[0].TermName != "Portfolio" {
		t.Fatalf("expected one match for Portfolio, got %+v", results)
	}

	after := testutil.ToFloat64(ResolutionAttempts.WithLabelValues(tenantID, "business_term_name", "hit"))
	if after != before+1 {
		t.Fatalf("expected business_term_name hit counter to increment by 1, went from %v to %v", before, after)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

func TestResolveTerm_Miss(t *testing.T) {
	r, mock := newTestResolver(t)
	tenantID := "tenant-1"

	mock.ExpectQuery(`FROM catalog_node alias_n`).
		WillReturnRows(sqlmock.NewRows([]string{"term_id", "term_name", "matched_token", "matched_via", "tenant_id"}))
	mock.ExpectQuery(`FROM catalog_node syn_n`).
		WillReturnRows(sqlmock.NewRows([]string{"term_id", "term_name", "matched_token", "matched_via", "tenant_id"}))
	mock.ExpectQuery(`FROM catalog_node cn`).
		WillReturnRows(sqlmock.NewRows([]string{"term_id", "term_name", "matched_token", "matched_via", "tenant_id"}))

	before := testutil.ToFloat64(ResolutionAttempts.WithLabelValues(tenantID, "business_term_name", "miss"))

	results, err := r.ResolveTerm(context.Background(), tenantID, "nonexistent-term")
	if err != nil {
		t.Fatalf("ResolveTerm returned error: %v", err)
	}
	if len(results) != 0 {
		t.Fatalf("expected no matches, got %+v", results)
	}

	after := testutil.ToFloat64(ResolutionAttempts.WithLabelValues(tenantID, "business_term_name", "miss"))
	if after != before+1 {
		t.Fatalf("expected business_term_name miss counter to increment by 1, went from %v to %v", before, after)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

func TestResolveTerm_ErrorIsSurfacedAndCounted(t *testing.T) {
	r, mock := newTestResolver(t)
	tenantID := "tenant-1"

	forced := errors.New("connection reset")
	mock.ExpectQuery(`FROM catalog_node alias_n`).WillReturnError(forced)

	before := testutil.ToFloat64(ResolutionAttempts.WithLabelValues(tenantID, "alias", "error"))

	_, err := r.ResolveTerm(context.Background(), tenantID, "portfolio")
	if err == nil {
		t.Fatal("expected ResolveTerm to surface the query error, got nil")
	}
	if !errors.Is(err, forced) {
		t.Fatalf("expected wrapped/forced error, got: %v", err)
	}

	after := testutil.ToFloat64(ResolutionAttempts.WithLabelValues(tenantID, "alias", "error"))
	if after != before+1 {
		t.Fatalf("expected alias error counter to increment by 1, went from %v to %v", before, after)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}
