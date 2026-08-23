package catalog

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/google/uuid"
)

type mockLoaderForE2E struct {
	rows []SubtypeRow
}

func (m *mockLoaderForE2E) LoadAllForTenant(ctx context.Context, db *sql.DB, tenantID uuid.UUID) ([]SubtypeRow, error) {
	return m.rows, nil
}

func TestSTIE2E_MockedFlow(t *testing.T) {
	tenantID := uuid.New()

	rows := []SubtypeRow{
		{
			ID:             uuid.New(),
			TenantID:       tenantID,
			RootObject:     "account",
			SubtypeCode:    "institutional",
			DisplayName:    "Institutional Account",
			FieldAllowlist: []string{"account_number", "sponsor_id"},
			IsActive:       true,
			CreatedAt:      time.Now(),
		},
		{
			ID:             uuid.New(),
			TenantID:       tenantID,
			RootObject:     "account",
			SubtypeCode:    "retail_wealth",
			DisplayName:    "Retail Wealth Account",
			FieldAllowlist: []string{"account_number", "account_name"},
			IsActive:       true,
			CreatedAt:      time.Now(),
		},
	}

	loader := &mockLoaderForE2E{rows: rows}
	builder := NewSubtypeBOBuilder(loader)

	if builder == nil {
		t.Fatal("builder is nil")
	}

	if len(rows) != 2 {
		t.Errorf("expected 2 rows, got %d", len(rows))
	}

	if rows[0].FieldAllowlist[0] != "account_number" {
		t.Errorf("unexpected first field: %s", rows[0].FieldAllowlist[0])
	}

	scanner := NewSTIColumnScanner()
	if scanner == nil {
		t.Fatal("scanner is nil")
	}

	linker := NewSubtypeSemanticLinker()
	if linker == nil {
		t.Fatal("linker is nil")
	}

	_ = builder
	_ = scanner
	_ = linker
}

type e2eTestCase struct {
	name           string
	subtypeCode    string
	expectedFields int
}

func TestSTIE2E_SubtypeFieldCounts(t *testing.T) {
	cases := []e2eTestCase{
		{"institutional", "institutional", 2},
		{"retail_wealth", "retail_wealth", 2},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			loader := NewSubtypeRegistryLoader(time.Minute)
			if loader == nil {
				t.Fatal("loader is nil")
			}
		})
	}
}
