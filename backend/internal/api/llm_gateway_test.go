package api

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestLoadSemanticBundle_RegionMatch(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to open sqlmock: %v", err)
	}
	defer db.Close()

	srv := &Server{DB: db}
	gw := &LLMGateway{server: srv}

	rows := sqlmock.NewRows([]string{"id", "name", "datasource_id", "driving_table", "coalesce"}).AddRow("bo-1", "customers", "ds-1", "customers", 1)
	mock.ExpectQuery("(?s)SELECT .* FROM business_objects bo").WithArgs("customers", "tenant-1", "eu-west").WillReturnRows(rows)

	fieldRows := sqlmock.NewRows([]string{"id", "name", "display_name", "semantic_term", "datasource_id", "table_name", "column_name"}).
		AddRow("field-1", "id", "ID", "Customer ID", "ds-1", "customers", "id")
	mock.ExpectQuery("(?s)SELECT .* FROM bo_fields").WithArgs("bo-1", "tenant-1").WillReturnRows(fieldRows)

	snapshotRows := sqlmock.NewRows([]string{"snapshot_id", "semantic_term_id", "business_term_id", "definition", "version", "metadata", "compliance", "lineage", "created_at"}).AddRow("ss-1", "term-1", "bt-1", "def", "2", "{}", "{}", "{}", time.Now())
	mock.ExpectQuery("(?s)SELECT properties->>'snapshot_id'.* FROM public.catalog_node").WithArgs("tenant-1", "eu-west").WillReturnRows(snapshotRows)

	bundle, err := gw.loadSemanticBundle(context.Background(), "tenant-1", "customers", "eu-west", "v1")
	if err != nil {
		t.Fatalf("expected bundle, got error: %v", err)
	}
	if bundle.BusinessObjectName != "customers" {
		t.Fatalf("unexpected bundle name: %s", bundle.BusinessObjectName)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestProcessQuery_RejectsMissingRegionFromPlanner(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to open sqlmock: %v", err)
	}
	defer db.Close()

	// Create a fake LLM provider that returns a semantic query without region
	tp := &fakeLLMProvider{ret: &SemanticQuery{Datasource: "customers", Select: []string{"id"}, Limit: 10}}

	gw := &LLMGateway{server: &Server{DB: db, GeminiClient: tp}}

	// Expectations for loadSemanticBundle
	rows := sqlmock.NewRows([]string{"id", "name", "datasource_id", "driving_table", "coalesce"}).AddRow("bo-1", "customers", "ds-1", "customers", 1)
	mock.ExpectQuery("(?s)SELECT .* FROM business_objects bo").WithArgs("customers", "tenant-1", "eu-west").WillReturnRows(rows)

	fieldRows := sqlmock.NewRows([]string{"id", "name", "display_name", "semantic_term", "datasource_id", "table_name", "column_name"}).
		AddRow("field-1", "id", "ID", "Customer ID", "ds-1", "customers", "id")
	mock.ExpectQuery("(?s)SELECT .* FROM bo_fields").WithArgs("bo-1", "tenant-1").WillReturnRows(fieldRows)

	snapshotRows := sqlmock.NewRows([]string{"snapshot_id", "semantic_term_id", "business_term_id", "definition", "version", "metadata", "compliance", "lineage", "created_at"}).AddRow("ss-1", "term-1", "bt-1", "def", "2", "{}", "{}", "{}", time.Now())
	mock.ExpectQuery("(?s)SELECT properties->>'snapshot_id'.* FROM public.catalog_node").WithArgs("tenant-1", "eu-west").WillReturnRows(snapshotRows)

	req := &SemanticQueryRequest{Datasource: "customers", Prompt: "List customers", Mode: "exploratory"}
	_, err = gw.ProcessQuery(context.Background(), "tenant-1", "eu-west", req)
	if err == nil {
		t.Fatalf("expected error when planner returns missing region")
	}
}

func TestLoadSemanticBundle_MissingSnapshotForRegion(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to open sqlmock: %v", err)
	}
	defer db.Close()

	srv := &Server{DB: db}
	gw := &LLMGateway{server: srv}

	// Expect BO query
	rows := sqlmock.NewRows([]string{"id", "name", "datasource_id", "driving_table", "coalesce"}).AddRow("bo-1", "customers", "ds-1", "customers", 1)
	mock.ExpectQuery("(?s)SELECT .* FROM business_objects bo").WithArgs("customers", "tenant-1", "eu-west").WillReturnRows(rows)

	fieldRows := sqlmock.NewRows([]string{"id", "name", "display_name", "semantic_term", "datasource_id", "table_name", "column_name"}).
		AddRow("field-1", "id", "ID", "Customer ID", "ds-1", "customers", "id")
	mock.ExpectQuery("(?s)SELECT .* FROM bo_fields").WithArgs("bo-1", "tenant-1").WillReturnRows(fieldRows)

	// Expect snapshot query to return no rows
	mock.ExpectQuery("SELECT properties->>'snapshot_id'.* FROM public.catalog_node").WithArgs("tenant-1", "eu-west").WillReturnError(sql.ErrNoRows)

	_, err = gw.loadSemanticBundle(context.Background(), "tenant-1", "customers", "eu-west", "v1")
	if err == nil {
		t.Fatalf("expected error when snapshot missing for region")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestLoadSemanticBundle_WithSnapshot(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to open sqlmock: %v", err)
	}
	defer db.Close()

	srv := &Server{DB: db}
	gw := &LLMGateway{server: srv}

	// Business object query
	rows := sqlmock.NewRows([]string{"id", "name", "datasource_id", "driving_table", "coalesce"}).AddRow("bo-1", "customers", "ds-1", "customers", 1)
	mock.ExpectQuery("(?s)SELECT .* FROM business_objects bo").WithArgs("customers", "tenant-1", "eu-west").WillReturnRows(rows)

	fieldRows := sqlmock.NewRows([]string{"id", "name", "display_name", "semantic_term", "datasource_id", "table_name", "column_name"}).
		AddRow("field-1", "id", "ID", "Customer ID", "ds-1", "customers", "id")
	mock.ExpectQuery("(?s)SELECT .* FROM bo_fields").WithArgs("bo-1", "tenant-1").WillReturnRows(fieldRows)

	// Snapshot query
	snapshotRows := sqlmock.NewRows([]string{"snapshot_id", "semantic_term_id", "business_term_id", "definition", "version", "metadata", "compliance", "lineage", "created_at"}).AddRow("ss-1", "term-1", "bt-1", "def", "2", "{\"foo\":\"bar\"}", "{}", "{}", time.Now())
	mock.ExpectQuery("(?s)SELECT properties->>'snapshot_id'.* FROM public.catalog_node").WithArgs("tenant-1", "eu-west").WillReturnRows(snapshotRows)

	bundle, err := gw.loadSemanticBundle(context.Background(), "tenant-1", "customers", "eu-west", "v1")
	if err != nil {
		t.Fatalf("expected bundle, got error: %v", err)
	}
	if bundle.Snapshot == nil || bundle.Snapshot.SnapshotID != "ss-1" {
		t.Fatalf("expected snapshot ss-1, got %v", bundle.Snapshot)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

// fakeLLMProvider implements the minimal interface used by LLMGateway for tests
type fakeLLMProvider struct {
	ret *SemanticQuery
	sql string
}

func (f *fakeLLMProvider) GenerateSemanticQuery(ctx context.Context, bundle *SemanticBundle, userPrompt string, mode string, region string) (*SemanticQuery, error) {
	return f.ret, nil
}

func (f *fakeLLMProvider) GenerateSQL(ctx context.Context, bundle *SemanticBundle, q *SemanticQuery) (string, error) {
	return f.sql, nil
}
