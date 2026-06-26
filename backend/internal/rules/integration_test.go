package rules_test

import (
	"context"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/hondyman/semlayer/backend/internal/models"
	"github.com/hondyman/semlayer/backend/internal/rules"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUMARebalanceRulesIntegration(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := rules.NewSQLRuleRepository(db)
	engine := rules.NewRuleEngine(repo)
	require.NoError(t, err)

	rebalanceEngine := rules.NewUMARebalanceRulesEngine(repo, engine)

	t.Run("Dynamic Rule Evaluation", func(t *testing.T) {
		// Mock rule in DB: "input.uma.aum < 1000000" (compliant if AUM < 1M)
		// We will test with AUM = 2M, so it should fail (violation)
		ruleID := uuid.New()
		rows := sqlmock.NewRows([]string{"id", "name", "description", "rule_type", "expression", "severity", "enabled", "created_at", "updated_at"}).
			AddRow(ruleID, "AUM Limit", "Max AUM check", "limit", "input.uma.aum < 1000000", "warning", true, time.Now(), time.Now())

		mock.ExpectQuery("SELECT id, name, description, rule_type, expression, severity, enabled, created_at, updated_at FROM compliance_rules").
			WillReturnRows(rows)

		uma := &models.UMAAccount{
			ID:  "uma-123",
			AUM: 2000000, // 2M > 1M, should violate
		}
		sleeves := []*models.UMASleeve{}
		plan := &models.UMARebalancePlan{}

		violations := rebalanceEngine.EvaluateRebalancePlan(context.Background(), uma, sleeves, plan)

		assert.NotEmpty(t, violations)
		found := false
		for _, v := range violations {
			if v.RuleID == ruleID.String() {
				found = true
				assert.Equal(t, "AUM Limit", v.RuleName)
				assert.Equal(t, "warning", v.Severity)
			}
		}
		assert.True(t, found, "Expected dynamic rule violation not found")
	})
}
