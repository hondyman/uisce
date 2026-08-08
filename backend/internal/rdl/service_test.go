package rdl

import (
	"context"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

func TestGetRulesByTenant(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock: %v", err)
	}
	defer db.Close()
	sqlxDB := sqlx.NewDb(db, "postgres")

	tenantID := uuid.New()
	ruleID1 := uuid.New()
	ruleID2 := uuid.New()

	rows := sqlmock.NewRows([]string{"id", "tenant_id", "rule_id", "type", "version", "name", "description", "jurisdiction", "parameters", "expression", "scoring_formula", "wash_sale_config", "substitute_asset_rules", "schedule", "notifications", "active", "audit", "effective_from", "effective_to", "created_at", "updated_at"}).
		AddRow(ruleID1, tenantID, "TLH_001", "tax_loss_harvesting", "1.0", "TLH Basic Rule", "Basic tax loss harvesting rule", "US", `{"min_loss_percentage": 10}`, "input.unrealized_loss_pct >= params.min_loss_percentage", "input.unrealized_loss_usd * 0.35", `{"enabled": true}`, "{}", "{}", "{}", true, "{}", nil, nil, "2025-01-01T00:00:00Z", "2025-01-01T00:00:00Z").
		AddRow(ruleID2, tenantID, "WS_001", "wash_sale", "1.0", "Wash Sale Rule", "Prevent wash sales", "US", "{}", "true", "", `{"enabled": false}`, "{}", "{}", "{}", true, "{}", nil, nil, "2025-01-01T00:00:00Z", "2025-01-01T00:00:00Z")

	mock.ExpectQuery("SELECT \\* FROM rule_definitions WHERE tenant_id").WithArgs(tenantID).WillReturnRows(rows)

	service := NewService(sqlxDB)
	ctx := context.Background()

	rules, err := service.GetRulesByTenant(ctx, tenantID)
	if err != nil {
		t.Fatalf("GetRulesByTenant failed: %v", err)
	}

	if len(rules) != 2 {
		t.Errorf("Expected 2 rules, got %d", len(rules))
	}

	if rules[0].RuleID != "TLH_001" {
		t.Errorf("Expected rule_id TLH_001, got %s", rules[0].RuleID)
	}
	if rules[0].Type != RuleTypeTaxLossHarvesting {
		t.Errorf("Expected type tax_loss_harvesting, got %s", rules[0].Type)
	}
}
