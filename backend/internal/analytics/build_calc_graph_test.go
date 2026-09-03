package analytics

import (
	"database/sql"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"github.com/hondyman/uisce/backend/internal/boresolver"
	"github.com/hondyman/uisce/backend/models"
)

// TestBuildCalcGraph_ResolvesCalcInCalcFromDB proves calc-in-calc works at
// the persistence layer: net_customer_xirr's formula references
// customer_xirr by NAME, and customer_xirr is a separately stored
// calculation (not a base field) — BuildCalcGraph must recursively pull it
// in as its own CalcNode rather than treating "customer_xirr" as an
// unresolved base field.
func TestBuildCalcGraph_ResolvesCalcInCalcFromDB(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer db.Close()
	svc := &SemanticCalculationService{db: sqlx.NewDb(db, "postgres")}

	root := &models.Calculation{
		ID:      uuid.New(),
		Name:    "net_customer_xirr",
		Formula: "${customer_xirr} - ${hurdle_rate}",
	}

	cols := []string{"id", "node_id", "name", "title", "description", "formula", "engine_type", "return_type", "arguments", "category", "subcategory", "domain_id", "execution_type", "engine", "is_materialized", "tier", "execution_preference"}
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM calculations WHERE name = $1`)).
		WithArgs("customer_xirr").
		WillReturnRows(sqlmock.NewRows(cols).AddRow(
			uuid.New(), uuid.Nil, "customer_xirr", "Customer XIRR", "", "xirr(${cashflow_amount}, ${cashflow_date})", "", "", []byte("null"), "", "", nil, "", "", false, "host_runtime", "auto",
		))
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM calculations WHERE name = $1`)).
		WithArgs("cashflow_amount").
		WillReturnError(sql.ErrNoRows)
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM calculations WHERE name = $1`)).
		WithArgs("cashflow_date").
		WillReturnError(sql.ErrNoRows)
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM calculations WHERE name = $1`)).
		WithArgs("hurdle_rate").
		WillReturnError(sql.ErrNoRows)

	graph, err := svc.BuildCalcGraph(root)
	if err != nil {
		t.Fatalf("BuildCalcGraph failed: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet SQL expectations: %v", err)
	}

	if _, ok := graph.Nodes["customer_xirr"]; !ok {
		t.Fatal("expected customer_xirr to be pulled in as a nested calc node")
	}
	if graph.Nodes["customer_xirr"].IsBaseField {
		t.Error("customer_xirr should be a calc node (has a formula), not a base field")
	}
	if graph.Nodes["customer_xirr"].Formula != "xirr(${cashflow_amount}, ${cashflow_date})" {
		t.Errorf("customer_xirr formula not carried through: %q", graph.Nodes["customer_xirr"].Formula)
	}
	for _, baseField := range []string{"hurdle_rate", "cashflow_amount", "cashflow_date"} {
		node, ok := graph.Nodes[baseField]
		if !ok {
			t.Errorf("expected %s to be present as a base field", baseField)
			continue
		}
		if !node.IsBaseField {
			t.Errorf("expected %s to be IsBaseField=true", baseField)
		}
	}

	// Full end-to-end proof: resolve layers and compile, confirming
	// net_customer_xirr is poisoned to host-runtime (it depends on
	// customer_xirr, which is host-runtime) even though its own formula
	// is pure arithmetic.
	layers, err := graph.ResolveExecutionLayers()
	if err != nil {
		t.Fatalf("ResolveExecutionLayers failed: %v", err)
	}
	gen := &boresolver.BOSQLGenerator{}
	_, hostNodes, err := gen.CompileDeepCalculations(layers, "SELECT 1", []string{"net_customer_xirr"})
	if err != nil {
		t.Fatalf("CompileDeepCalculations failed: %v", err)
	}
	if len(hostNodes) != 2 {
		t.Fatalf("expected 2 host-runtime nodes (customer_xirr, net_customer_xirr), got %d", len(hostNodes))
	}
}
