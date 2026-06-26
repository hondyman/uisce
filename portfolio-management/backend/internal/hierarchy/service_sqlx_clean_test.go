package hierarchy

import (
	"context"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/jmoiron/sqlx"
)

func TestGetEntityHierarchy_SimpleTree(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to open sqlmock: %v", err)
	}
	defer db.Close()

	dbx := sqlx.NewDb(db, "postgres")
	svc := NewHierarchyServiceSQLXImpl(dbx)

	// expected columns returned by the recursive CTE
	columns := []string{"id", "tenant_id", "model_type", "display_name", "parent_id", "depth", "path_ids_json", "path_names_json"}

	rootID := "root-uuid"
	childID := "child-uuid"

	rows := sqlmock.NewRows(columns).
		AddRow(rootID, "t1", "typeA", "Root", nil, 0, `["root-uuid"]`, `["Root"]`).
		AddRow(childID, "t1", "typeB", "Child", rootID, 1, `["root-uuid","child-uuid"]`, `["Root","Child"]`)

	// The service will run the CTE query once; provide the rows
	mock.ExpectQuery("SELECT id, tenant_id").WillReturnRows(rows)

	root, err := svc.GetEntityHierarchy(context.Background(), rootID, -1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if root == nil {
		t.Fatalf("expected root node, got nil")
	}
	if root.ID != rootID {
		t.Fatalf("expected root id %s, got %s", rootID, root.ID)
	}
	if len(root.Children) != 1 {
		t.Fatalf("expected 1 child, got %d", len(root.Children))
	}
	if root.Children[0].ID != childID {
		t.Fatalf("expected child id %s, got %s", childID, root.Children[0].ID)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sqlmock expectations: %v", err)
	}
}

func TestGetEntityHierarchy_RootNotFound(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to open sqlmock: %v", err)
	}
	defer db.Close()

	dbx := sqlx.NewDb(db, "postgres")
	svc := NewHierarchyServiceSQLXImpl(dbx)

	columns := []string{"id", "tenant_id", "model_type", "display_name", "parent_id", "depth", "path_ids_json", "path_names_json"}
	empty := sqlmock.NewRows(columns)

	mock.ExpectQuery("SELECT id, tenant_id").WillReturnRows(empty)

	_, err = svc.GetEntityHierarchy(context.Background(), "missing-root", -1)
	if err == nil {
		t.Fatalf("expected error for missing root, got nil")
	}
}

func TestGetEntityHierarchy_MaxDepth(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to open sqlmock: %v", err)
	}
	defer db.Close()

	dbx := sqlx.NewDb(db, "postgres")
	svc := NewHierarchyServiceSQLXImpl(dbx)

	columns := []string{"id", "tenant_id", "model_type", "display_name", "parent_id", "depth", "path_ids_json", "path_names_json"}
	rootID := "root-uuid"
	child1 := "child1-uuid"
	child2 := "child2-uuid"

	rows := sqlmock.NewRows(columns).
		AddRow(rootID, "t1", "typeA", "Root", nil, 0, `["root-uuid"]`, `["Root"]`).
		AddRow(child1, "t1", "typeB", "Child1", rootID, 1, `["root-uuid","child1-uuid"]`, `["Root","Child1"]`).
		AddRow(child2, "t1", "typeC", "Child2", child1, 2, `["root-uuid","child1-uuid","child2-uuid"]`, `["Root","Child1","Child2"]`)

	mock.ExpectQuery("SELECT id, tenant_id").WillReturnRows(rows)

	root, err := svc.GetEntityHierarchy(context.Background(), rootID, 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if root == nil {
		t.Fatalf("expected root node, got nil")
	}
	// maxDepth=1 should include only child1 (depth 1), not child2
	if len(root.Children) != 1 {
		t.Fatalf("expected 1 child at depth<=1, got %d", len(root.Children))
	}
	if root.Children[0].ID != child1 {
		t.Fatalf("expected child id %s, got %s", child1, root.Children[0].ID)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sqlmock expectations: %v", err)
	}
}

func TestGetEntityHierarchy_CycleDetection(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to open sqlmock: %v", err)
	}
	defer db.Close()

	dbx := sqlx.NewDb(db, "postgres")
	svc := NewHierarchyServiceSQLXImpl(dbx)

	columns := []string{"id", "tenant_id", "model_type", "display_name", "parent_id", "depth", "path_ids_json", "path_names_json"}
	a := "a-uuid"
	b := "b-uuid"

	// Simulate a cycle: A -> B and B -> A (the CTE can return cycles depending on data)
	rows := sqlmock.NewRows(columns).
		AddRow(a, "t1", "typeA", "A", nil, 0, `["a-uuid"]`, `["A"]`).
		AddRow(b, "t1", "typeB", "B", a, 1, `["a-uuid","b-uuid"]`, `["A","B"]`).
		// also add A as child of B to form a cycle in returned rows
		AddRow(a, "t1", "typeA", "A", b, 2, `["a-uuid","b-uuid","a-uuid"]`, `["A","B","A"]`)

	mock.ExpectQuery("SELECT id, tenant_id").WillReturnRows(rows)

	// Should return without infinite recursion and provide a structure
	root, err := svc.GetEntityHierarchy(context.Background(), a, -1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if root == nil || root.ID != a {
		t.Fatalf("expected root a, got %+v", root)
	}

	// Ensure children are present (cycle handled gracefully in-memory)
	if len(root.Children) == 0 {
		t.Fatalf("expected at least one child for root, got 0")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sqlmock expectations: %v", err)
	}
}
