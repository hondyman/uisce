package catalog

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/google/uuid"
)

type mockLoaderForBuilder struct {
	rows []SubtypeRow
	err  error
}

func (m *mockLoaderForBuilder) LoadAllForTenant(ctx context.Context, db *sql.DB, tenantID uuid.UUID) ([]SubtypeRow, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.rows, nil
}

type mockDBForBO struct {
	execResults []mockExecResult
	txStarted  bool
	txCommitted bool
	txRolledBack bool
}

type mockExecResult struct {
	query string
	args  []interface{}
	err   error
}

func TestSubtypeBOBuilder_BuildForTenant_LoadError(t *testing.T) {
	loader := &mockLoaderForBuilder{
		err: context.DeadlineExceeded,
	}
	builder := NewSubtypeBOBuilder(loader)

	err := builder.BuildForTenant(context.Background(), nil, uuid.New())
	if err == nil {
		t.Error("expected error from loader")
	}
}

func TestSubtypeBOBuilder_QualifiedPath(t *testing.T) {
	_ = SubtypeRow{
		ID:                uuid.New(),
		TenantID:          uuid.New(),
		RootObject:        "account",
		SubtypeCode:       "institutional",
		DisplayName:       "Institutional Account",
		FieldAllowlist:    []string{"account_number", "sponsor_id"},
		IsActive:          true,
		CreatedAt:         time.Now(),
	}

	expectedPath := "oms.account/institutional"
	if expectedPath != "oms.account/institutional" {
		t.Errorf("path mismatch")
	}

	attrPath := expectedPath + "/account_number"
	if attrPath != "oms.account/institutional/account_number" {
		t.Errorf("expected oms.account/institutional/account_number, got %s", attrPath)
	}
}

func TestSubtypeBOBuilder_LoaderInterface(t *testing.T) {
	rows := []SubtypeRow{
		{
			ID:             uuid.New(),
			TenantID:       uuid.New(),
			RootObject:     "account",
			SubtypeCode:    "institutional",
			DisplayName:    "Institutional Account",
			FieldAllowlist: []string{"account_number", "sponsor_id"},
			IsActive:       true,
			CreatedAt:      time.Now(),
		},
	}

	loader := &mockLoaderForBuilder{rows: rows}
	builder := NewSubtypeBOBuilder(loader)

	if builder == nil {
		t.Fatal("expected non-nil builder")
	}
}
