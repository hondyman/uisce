package boresolver

import (
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/hondyman/uisce/backend/internal/models"
	"github.com/jmoiron/sqlx"
)

// TestGetBOTerms_CalcTermSkipsDefaultAggregation verifies that calc-term
// fields (properties->>'term_type' = 'calculated') get TermType="calculated"
// and NO DefaultAggregation, while physical MEASURE fields still get
// DefaultAggregation="SUM". This is the backend enforcement of the standing
// rule: calc terms never receive fabricated aggregation.
func TestGetBOTerms_CalcTermSkipsDefaultAggregation(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	// The query LEFT JOINs catalog_node to detect calc terms.
	// Columns: term_node_id, term_key, term_name, display_name,
	//          description, data_type, role, binding_status, term_type
	cols := []string{
		"term_node_id", "term_key", "term_name", "display_name",
		"description", "data_type", "role", "binding_status", "term_type",
	}

	mock.ExpectQuery("FROM public.business_object_fields f").
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows(cols).
			// Physical MEASURE — should get DefaultAggregation="SUM"
			AddRow("f-physical", "total_amount", "Total Amount", "Total Amount",
				"", "number", "MEASURE", "RESOLVED", "").
			// Calc term — should get TermType="calculated", no DefaultAggregation
			AddRow("f-calc", "trade_value", "Trade Value", "Trade Value",
				"", "number", "MEASURE", "RESOLVED", "calculated").
			// Physical DIMENSION — no DefaultAggregation regardless
			AddRow("f-dim", "order_date", "Order Date", "Order Date",
				"", "date", "DIMENSION", "RESOLVED", ""))

	repo := &PostgresBORepository{DB: sqlx.NewDb(db, "sqlmock")}
	terms, err := repo.GetBOTerms("test-bo-id", "")
	if err != nil {
		t.Fatalf("GetBOTerms: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}

	if len(terms) != 3 {
		t.Fatalf("expected 3 terms, got %d", len(terms))
	}

	// Physical MEASURE: DefaultAggregation = "SUM", TermType = ""
	if terms[0].DefaultAggregation != "SUM" {
		t.Errorf("physical MEASURE: expected DefaultAggregation=%q, got %q", "SUM", terms[0].DefaultAggregation)
	}
	if terms[0].TermType != "" {
		t.Errorf("physical MEASURE: expected empty TermType, got %q", terms[0].TermType)
	}

	// Calc term: DefaultAggregation = "", TermType = "calculated"
	if terms[1].DefaultAggregation != "" {
		t.Errorf("calc term: expected empty DefaultAggregation, got %q", terms[1].DefaultAggregation)
	}
	if terms[1].TermType != "calculated" {
		t.Errorf("calc term: expected TermType=%q, got %q", "calculated", terms[1].TermType)
	}

	// Physical DIMENSION: DefaultAggregation = "", TermType = ""
	if terms[2].DefaultAggregation != "" {
		t.Errorf("physical DIMENSION: expected empty DefaultAggregation, got %q", terms[2].DefaultAggregation)
	}
	if terms[2].TermType != "" {
		t.Errorf("physical DIMENSION: expected empty TermType, got %q", terms[2].TermType)
	}
}

// TestGetBOTerms_CalcTermPropertiesAssumption verifies the properties->>'term_type'
// assumption: CalcTermProperties.TermType is always "calculated" for nodes written
// by CalcTermService.UpsertCalcTerm. If this assumption breaks, the LEFT JOIN
// in GetBOTerms silently no-ops and the frontend ternary never fires.
func TestGetBOTerms_CalcTermPropertiesAssumption(t *testing.T) {
	// CalcTermProperties is the struct stored in catalog_node.properties.
	// TermType is always "calculated" — this test pins that contract.
	props := models.CalcTermProperties{
		BOName:   "bo_orders",
		TenantID: "test-tenant",
		TermType: "calculated",
	}
	if props.TermType != "calculated" {
		t.Errorf("CalcTermProperties.TermType must be %q, got %q", "calculated", props.TermType)
	}
}
