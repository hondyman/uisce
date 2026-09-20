package api

import (
	"context"
	"database/sql"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
)

func TestFindOrCreateTermNode_FallbackReSelect(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer db.Close()

	svc := NewGlossaryService(context.Background(), db, nil, NewInMemoryJobStore())

	tenantID := uuid.New().String()
	nodeTypeID := uuid.New().String()
	qualifiedPrefix := "semantic_term"
	name := "TestTerm"
	qualifiedPath := qualifiedPrefix + "/" + name

	// Phase 1: pre-check returns no rows (no existing node)
	preCheckRows := sqlmock.NewRows([]string{"id"})
	mock.ExpectQuery(
		`SELECT id FROM catalog_node WHERE node_type_id = \$1 AND tenant_id = \$2 AND \(qualified_path = \$3 OR lower\(node_name\) = lower\(\$4\)\) LIMIT 1`,
	).WithArgs(nodeTypeID, tenantID, qualifiedPath, name).
		WillReturnRows(preCheckRows)

	// Phase 2: INSERT with ON CONFLICT DO NOTHING — no row returned (conflict fired)
	mock.ExpectQuery(
		`INSERT INTO catalog_node.*ON CONFLICT \(tenant_id, qualified_path\) DO NOTHING.*RETURNING id`,
	).WithArgs(name, nodeTypeID, tenantID, sqlmock.AnyArg(), sqlmock.AnyArg(), qualifiedPath).
		WillReturnError(sql.ErrNoRows)

	// Fallback re-SELECT returns the existing row
	fallbackID := uuid.New().String()
	fallbackRows := sqlmock.NewRows([]string{"id"}).AddRow(fallbackID)
	mock.ExpectQuery(
		`SELECT id FROM catalog_node WHERE tenant_id = \$1 AND qualified_path = \$2 LIMIT 1`,
	).WithArgs(tenantID, qualifiedPath).
		WillReturnRows(fallbackRows)

	id, reused, err := svc.findOrCreateTermNode(
		context.Background(),
		tenantID, "none", nodeTypeID,
		qualifiedPrefix, name, "some definition",
	)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if id != fallbackID {
		t.Errorf("id = %q; want %q", id, fallbackID)
	}
	if !reused {
		t.Error("reused = false; want true")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled sqlmock expectations: %v", err)
	}
}

func TestFindOrCreateTermNode_NewInsert(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer db.Close()

	svc := NewGlossaryService(context.Background(), db, nil, NewInMemoryJobStore())

	tenantID := uuid.New().String()
	nodeTypeID := uuid.New().String()
	qualifiedPrefix := "semantic_term"
	name := "BrandNewTerm"
	qualifiedPath := qualifiedPrefix + "/" + name

	// Phase 1: pre-check returns no rows
	preCheckRows := sqlmock.NewRows([]string{"id"})
	mock.ExpectQuery(
		`SELECT id FROM catalog_node WHERE node_type_id = \$1 AND tenant_id = \$2 AND \(qualified_path = \$3 OR lower\(node_name\) = lower\(\$4\)\) LIMIT 1`,
	).WithArgs(nodeTypeID, tenantID, qualifiedPath, name).
		WillReturnRows(preCheckRows)

	// Phase 2: INSERT returns the new id (no conflict)
	newID := uuid.New().String()
	insertRows := sqlmock.NewRows([]string{"id"}).AddRow(newID)
	mock.ExpectQuery(
		`INSERT INTO catalog_node.*ON CONFLICT \(tenant_id, qualified_path\) DO NOTHING.*RETURNING id`,
	).WithArgs(name, nodeTypeID, tenantID, sqlmock.AnyArg(), sqlmock.AnyArg(), qualifiedPath).
		WillReturnRows(insertRows)

	id, reused, err := svc.findOrCreateTermNode(
		context.Background(),
		tenantID, "none", nodeTypeID,
		qualifiedPrefix, name, "",
	)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if id != newID {
		t.Errorf("id = %q; want %q", id, newID)
	}
	if reused {
		t.Error("reused = true; want false (newly inserted)")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled sqlmock expectations: %v", err)
	}
}

func TestFindOrCreateTermNode_PreCheckHit(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer db.Close()

	svc := NewGlossaryService(context.Background(), db, nil, NewInMemoryJobStore())

	tenantID := uuid.New().String()
	nodeTypeID := uuid.New().String()
	qualifiedPrefix := "semantic_term"
	name := "AlreadyExists"
	qualifiedPath := qualifiedPrefix + "/" + name

	// Phase 1: pre-check finds the existing row
	existingID := uuid.New().String()
	preCheckRows := sqlmock.NewRows([]string{"id"}).AddRow(existingID)
	mock.ExpectQuery(
		`SELECT id FROM catalog_node WHERE node_type_id = \$1 AND tenant_id = \$2 AND \(qualified_path = \$3 OR lower\(node_name\) = lower\(\$4\)\) LIMIT 1`,
	).WithArgs(nodeTypeID, tenantID, qualifiedPath, name).
		WillReturnRows(preCheckRows)

	// No INSERT should occur
	id, reused, err := svc.findOrCreateTermNode(
		context.Background(),
		tenantID, "none", nodeTypeID,
		qualifiedPrefix, name, "",
	)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if id != existingID {
		t.Errorf("id = %q; want %q", id, existingID)
	}
	if !reused {
		t.Error("reused = false; want true")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled sqlmock expectations: %v", err)
	}
}
