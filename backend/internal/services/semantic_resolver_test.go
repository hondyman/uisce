package services

import (
	"errors"
	"testing"

	"github.com/hondyman/semlayer/backend/internal/models"
)

// ResolverMockRepo for standalone testing
type ResolverMockRepo struct {
	terms map[string]*models.SemanticTerm
}

func (m *ResolverMockRepo) GetTerm(id string) (*models.SemanticTerm, error) {
	if t, ok := m.terms[id]; ok {
		return t, nil
	}
	return nil, errors.New("term not found")
}

func (m *ResolverMockRepo) GetTermsByTable(tableName string) ([]*models.SemanticTerm, error) {
	return nil, nil
}

func TestResolveToSQL_Materialization(t *testing.T) {
	repo := &ResolverMockRepo{
		terms: map[string]*models.SemanticTerm{
			"holding.market_value_resolved": {
				ID:              "holding.market_value_resolved",
				Type:            models.SemanticTypeCalculated,
				Expression:      "CASE WHEN ... END",
				Materialization: "materialized_table",
			},
			"holding.market_value_virtual": {
				ID:              "holding.market_value_virtual",
				Type:            models.SemanticTypeCalculated,
				Expression:      "1 + 1",
				Materialization: "virtual",
			},
		},
	}

	resolver := NewSemanticResolver(repo)

	// Test 1: Materialized Table Strategy
	sql, err := resolver.ResolveToSQL("holding.market_value_resolved")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := "analytics.holdings_preagg.holding_market_value_resolved"
	if sql != expected {
		t.Errorf("expected %s, got %s", expected, sql)
	}

	// Test 2: Virtual Strategy
	sqlVirtual, err := resolver.ResolveToSQL("holding.market_value_virtual")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// The naive resolveExpression wraps simple math in parens roughly
	if len(sqlVirtual) < 3 {
		t.Errorf("expected non-empty SQL for virtual, got %s", sqlVirtual)
	}
}
