package ai

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/google/uuid"
)

// Pattern represents a detected data pattern
type Pattern struct {
	Type        string                   `json:"type"`
	Description string                   `json:"description"`
	Frequency   int                      `json:"frequency"`
	Severity    string                   `json:"severity"`
	Examples    []map[string]interface{} `json:"examples"`
}

// PatternAnalyzer analyzes data patterns for rule suggestions
type PatternAnalyzer struct {
	db *sql.DB
}

// NewPatternAnalyzer creates a new pattern analyzer
func NewPatternAnalyzer(db *sql.DB) *PatternAnalyzer {
	return &PatternAnalyzer{db: db}
}

// AnalyzeTransactionPatterns analyzes transaction data for patterns
func (pa *PatternAnalyzer) AnalyzeTransactionPatterns(ctx context.Context, tenantID uuid.UUID) ([]Pattern, error) {
	patterns := []Pattern{}

	// Pattern 1: Check for large transactions
	largeTransPattern, err := pa.detectLargeTransactions(ctx, tenantID)
	if err == nil && largeTransPattern != nil {
		patterns = append(patterns, *largeTransPattern)
	}

	// Pattern 2: Check for frequent violations
	violationPattern, err := pa.detectFrequentViolations(ctx, tenantID)
	if err == nil && violationPattern != nil {
		patterns = append(patterns, *violationPattern)
	}

	// Pattern 3: Check for concentration issues
	concentrationPattern, err := pa.detectConcentrationIssues(ctx, tenantID)
	if err == nil && concentrationPattern != nil {
		patterns = append(patterns, *concentrationPattern)
	}

	return patterns, nil
}

// detectLargeTransactions detects unusually large transactions
func (pa *PatternAnalyzer) detectLargeTransactions(ctx context.Context, tenantID uuid.UUID) (*Pattern, error) {
	query := `
		SELECT COUNT(*) as count, AVG(amount) as avg_amount
		FROM transactions
		WHERE tenant_id = $1
		  AND amount > (SELECT AVG(amount) * 3 FROM transactions WHERE tenant_id = $1)
		  AND created_at > NOW() - '30 days'::interval
	`

	var count int
	var avgAmount float64
	err := pa.db.QueryRowContext(ctx, query, tenantID).Scan(&count, &avgAmount)
	if err != nil {
		return nil, err
	}

	if count > 10 { // Threshold: more than 10 large transactions
		return &Pattern{
			Type:        "large_transactions",
			Description: fmt.Sprintf("Detected %d transactions exceeding 3x average amount", count),
			Frequency:   count,
			Severity:    "medium",
		}, nil
	}

	return nil, nil
}

// detectFrequentViolations detects frequently violated constraints
func (pa *PatternAnalyzer) detectFrequentViolations(ctx context.Context, tenantID uuid.UUID) (*Pattern, error) {
	query := `
		SELECT rule_type, COUNT(*) as violation_count
		FROM validation_rule_executions e
		JOIN validation_rules r ON e.rule_id = r.id
		WHERE e.tenant_id = $1
		  AND e.status = 'violation'
		  AND e.executed_at > NOW() - '30 days'::interval
		GROUP BY rule_type
		HAVING COUNT(*) > 50
		ORDER BY violation_count DESC
		LIMIT 1
	`

	var ruleType string
	var count int
	err := pa.db.QueryRowContext(ctx, query, tenantID).Scan(&ruleType, &count)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	return &Pattern{
		Type:        "frequent_violations",
		Description: fmt.Sprintf("Rule type '%s' violated %d times in last 30 days", ruleType, count),
		Frequency:   count,
		Severity:    "high",
	}, nil
}

// detectConcentrationIssues detects portfolio concentration issues
func (pa *PatternAnalyzer) detectConcentrationIssues(ctx context.Context, tenantID uuid.UUID) (*Pattern, error) {
	query := `
		SELECT COUNT(DISTINCT account_id) as accounts,
		       COUNT(*) as positions
		FROM positions
		WHERE tenant_id = $1
		  AND position_value > (
		      SELECT SUM(position_value) * 0.25
		      FROM positions
		      WHERE tenant_id = $1
		  )
	`

	var accounts, positions int
	err := pa.db.QueryRowContext(ctx, query, tenantID).Scan(&accounts, &positions)
	if err != nil {
		return nil, err
	}

	if positions > 5 { // More than 5 positions exceeding 25% of portfolio
		return &Pattern{
			Type:        "concentration_risk",
			Description: fmt.Sprintf("Found %d positions exceeding 25%% of portfolio value", positions),
			Frequency:   positions,
			Severity:    "high",
		}, nil
	}

	return nil, nil
}

// GenerateRuleSuggestions generates rule suggestions from patterns
func (pa *PatternAnalyzer) GenerateRuleSuggestions(ctx context.Context, tenantID uuid.UUID, patterns []Pattern) ([]RuleSuggestion, error) {
	suggestions := []RuleSuggestion{}

	for _, pattern := range patterns {
		suggestion := pa.patternToSuggestion(tenantID, pattern)
		if suggestion != nil {
			suggestions = append(suggestions, *suggestion)
		}
	}

	return suggestions, nil
}

// patternToSuggestion converts a pattern to a rule suggestion
func (pa *PatternAnalyzer) patternToSuggestion(tenantID uuid.UUID, pattern Pattern) *RuleSuggestion {
	switch pattern.Type {
	case "large_transactions":
		return &RuleSuggestion{
			ID:            uuid.New(),
			TenantID:      tenantID,
			RuleType:      "transaction_limit",
			SuggestedName: "Large Transaction Alert",
			Description:   "Alert when transaction exceeds 3x average amount",
			Parameters: map[string]interface{}{
				"threshold_multiplier": 3.0,
				"severity":             "warn",
			},
			Confidence:  0.85,
			BasePattern: &pattern,
			Rationale:   "Detected frequent large transactions that may indicate unusual activity",
			Status:      "pending",
		}

	case "concentration_risk":
		return &RuleSuggestion{
			ID:            uuid.New(),
			TenantID:      tenantID,
			RuleType:      "concentration_limit",
			SuggestedName: "Position Concentration Limit",
			Description:   "Limit individual position size to prevent concentration risk",
			Parameters: map[string]interface{}{
				"max_percentage": 25.0,
				"severity":       "block",
			},
			Confidence:  0.90,
			BasePattern: &pattern,
			Rationale:   "High concentration risk detected across multiple positions",
			Status:      "pending",
		}

	case "frequent_violations":
		return &RuleSuggestion{
			ID:            uuid.New(),
			TenantID:      tenantID,
			RuleType:      "monitoring_rule",
			SuggestedName: "Enhanced Monitoring",
			Description:   "Enable enhanced monitoring for frequently violated rule types",
			Parameters: map[string]interface{}{
				"alert_threshold": 10,
				"notification":    true,
			},
			Confidence:  0.75,
			BasePattern: &pattern,
			Rationale:   "Frequent violations suggest need for tighter controls",
			Status:      "pending",
		}
	}

	return nil
}
