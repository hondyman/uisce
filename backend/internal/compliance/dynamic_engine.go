package compliance

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

// ToleranceConfig defines declarative thresholds without hardcoded conditionals
type ToleranceConfig struct {
	Unit          string  `json:"unit"`
	AlertAbove    float64 `json:"alert_above"`
	AlertBelow    float64 `json:"alert_below"`
	SeverityAbove string  `json:"severity_above"`
	SeverityBelow string  `json:"severity_below"`
}

// ComparisonTargetConfig holds declarative benchmark target topology
type ComparisonTargetConfig struct {
	TargetType    string     `json:"target_type"`
	TargetNodeID  *uuid.UUID `json:"target_node_id,omitempty"`
	BenchmarkCode string     `json:"benchmark_code"`
}

// DynamicComplianceRule represents an opaque graph-driven compliance rule loaded from catalog_node
type DynamicComplianceRule struct {
	RuleID             uuid.UUID              `json:"rule_id"`
	RuleKey            string                 `json:"rule_key"`
	RuleName           string                 `json:"rule_name"`
	RuleCategory       string                 `json:"rule_category"`
	EvaluationMode     string                 `json:"evaluation_granularity"`
	GroupingDimension  string                 `json:"grouping_term_key"`
	FilterAST          string                 `json:"filter_ast"`
	PortfolioMetricAST string                 `json:"portfolio_metric_ast"`
	BenchmarkMetricAST string                 `json:"benchmark_metric_ast"`
	VarianceExpression string                 `json:"variance_expression"`
	ComparisonTarget   ComparisonTargetConfig `json:"comparison_target"`
	Tolerance          ToleranceConfig        `json:"tolerance"`
	ExecutionStrategy  string                 `json:"execution_strategy"`
}

// GroupedComparisonRow represents an evaluated dimension group between portfolio and benchmark
type GroupedComparisonRow struct {
	GroupName       string  `json:"group_name"`
	PortfolioMetric float64 `json:"portfolio_metric"`
	BenchmarkMetric float64 `json:"benchmark_metric"`
	ActiveDelta     float64 `json:"active_delta"`
}

// RuleBreach captures a threshold violation computed purely against graph metadata
type RuleBreach struct {
	RuleID         uuid.UUID `json:"rule_id"`
	RuleKey        string    `json:"rule_key"`
	GroupKey       string    `json:"group_key"`
	PortfolioVal   float64   `json:"portfolio_val"`
	BenchmarkVal   float64   `json:"benchmark_val"`
	ActiveDelta    float64   `json:"active_delta"`
	BreachType     string    `json:"breach_type"`
	ThresholdLimit float64   `json:"threshold_limit"`
	EvaluatedAt    time.Time `json:"evaluated_at"`
}

// DynamicComplianceEngine evaluates compliance rules solely based on graph metadata and AST expressions
type DynamicComplianceEngine struct {
	db *sql.DB
}

// NewDynamicComplianceEngine creates a new graph-driven compliance evaluator
func NewDynamicComplianceEngine(db *sql.DB) *DynamicComplianceEngine {
	return &DynamicComplianceEngine{db: db}
}

// EvaluateRule executes the rule purely based on graph properties without hardcoded conditionals
func (e *DynamicComplianceEngine) EvaluateRule(
	ctx context.Context,
	rule DynamicComplianceRule,
	rows []GroupedComparisonRow,
) []RuleBreach {
	var breaches []RuleBreach
	now := time.Now().UTC()

	for _, row := range rows {
		activeVariance := row.PortfolioMetric - row.BenchmarkMetric

		// Upper threshold evaluation
		if rule.Tolerance.AlertAbove != 0 && activeVariance > rule.Tolerance.AlertAbove {
			breaches = append(breaches, RuleBreach{
				RuleID:         rule.RuleID,
				RuleKey:        rule.RuleKey,
				GroupKey:       row.GroupName,
				PortfolioVal:   row.PortfolioMetric,
				BenchmarkVal:   row.BenchmarkMetric,
				ActiveDelta:    activeVariance,
				BreachType:     rule.Tolerance.SeverityAbove,
				ThresholdLimit: rule.Tolerance.AlertAbove,
				EvaluatedAt:    now,
			})
		}

		// Lower threshold evaluation
		if rule.Tolerance.AlertBelow != 0 && activeVariance < rule.Tolerance.AlertBelow {
			breaches = append(breaches, RuleBreach{
				RuleID:         rule.RuleID,
				RuleKey:        rule.RuleKey,
				GroupKey:       row.GroupName,
				PortfolioVal:   row.PortfolioMetric,
				BenchmarkVal:   row.BenchmarkMetric,
				ActiveDelta:    activeVariance,
				BreachType:     rule.Tolerance.SeverityBelow,
				ThresholdLimit: rule.Tolerance.AlertBelow,
				EvaluatedAt:    now,
			})
		}
	}

	return breaches
}

// LoadRuleFromGraph resolves a compliance rule node and its outgoing graph edges (COMPARES_WITH, GROUPS_BY, FILTERS_BY)
func (e *DynamicComplianceEngine) LoadRuleFromGraph(
	ctx context.Context,
	tenantID uuid.UUID,
	ruleNodeID uuid.UUID,
) (*DynamicComplianceRule, error) {
	if e.db == nil {
		return nil, fmt.Errorf("database connection is nil")
	}

	var nodeName, nodeType string
	var propertiesJSON sql.NullString

	err := e.db.QueryRowContext(ctx, `
		SELECT node_name, COALESCE(catalog_type, ''), properties::text
		FROM catalog_node
		WHERE id = $1 AND (tenant_id = $2 OR tenant_id = (SELECT id FROM public.tenants WHERE gold_copy = true LIMIT 1))
		LIMIT 1
	`, ruleNodeID, tenantID).Scan(&nodeName, &nodeType, &propertiesJSON)

	if err != nil {
		return nil, fmt.Errorf("failed to load compliance rule node: %w", err)
	}

	rule := DynamicComplianceRule{
		RuleID:   ruleNodeID,
		RuleName: nodeName,
	}

	if propertiesJSON.Valid && propertiesJSON.String != "" {
		if err := json.Unmarshal([]byte(propertiesJSON.String), &rule); err != nil {
			return nil, fmt.Errorf("failed to unmarshal rule properties: %w", err)
		}
	}

	// Resolve graph edges: COMPARES_WITH (benchmark target) and GROUPS_BY (semantic term)
	rows, err := e.db.QueryContext(ctx, `
		SELECT 
			e.edge_type_name,
			t.id,
			t.node_name,
			COALESCE(t.catalog_type, '')
		FROM catalog_edge e
		JOIN catalog_node t ON e.object_node_id = t.id
		WHERE e.subject_node_id = $1
		  AND (e.tenant_id = $2 OR e.tenant_id = (SELECT id FROM public.tenants WHERE gold_copy = true LIMIT 1))
	`, ruleNodeID, tenantID)

	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var edgeType, targetIDStr, targetName, targetCatalogType string
			if err := rows.Scan(&edgeType, &targetIDStr, &targetName, &targetCatalogType); err == nil {
				targetUUID, _ := uuid.Parse(targetIDStr)
				switch strings.ToUpper(edgeType) {
				case "COMPARES_WITH":
					rule.ComparisonTarget.TargetNodeID = &targetUUID
					rule.ComparisonTarget.BenchmarkCode = targetName
				case "GROUPS_BY":
					rule.GroupingDimension = targetName
				}
			}
		}
	}

	return &rule, nil
}

// CompileBenchmarkSQL generates the hot/cold pushdown query adhering to Rule 1, 4, 6, and 7
func (e *DynamicComplianceEngine) CompileBenchmarkSQL(
	rule DynamicComplianceRule,
	dimensionColumn string,
	filterExpression string,
) string {
	dimCol := dimensionColumn
	if dimCol == "" {
		dimCol = "industry_sector"
	}

	filterClause := ""
	if filterExpression != "" {
		filterClause = fmt.Sprintf("AND %s", filterExpression)
	}

	return fmt.Sprintf(`WITH portfolio_holdings AS (
    SELECT 
        h.%s AS sector_name,
        SUM(h.market_value) AS sector_mv,
        SUM(SUM(h.market_value)) OVER () AS total_portfolio_mv
    FROM ibor.position h
    WHERE h.tenant_id = :tenant_id
      AND h.account_id = :portfolio_account_id
      %s
      AND h.effective_date = :effective_date
    GROUP BY h.%s
),
benchmark_holdings AS (
    SELECT 
        b.%s AS sector_name,
        SUM(b.market_value) AS sector_mv,
        SUM(SUM(b.market_value)) OVER () AS total_benchmark_mv
    FROM ibor.benchmark_position b
    WHERE b.tenant_id = :tenant_id
      AND b.benchmark_id = :benchmark_id
      %s
      AND b.effective_date = :effective_date
    GROUP BY b.%s
)
SELECT 
    COALESCE(p.sector_name, b.sector_name) AS group_name,
    COALESCE((p.sector_mv / NULLIF(p.total_portfolio_mv, 0)) * 100.0, 0.0) AS portfolio_metric,
    COALESCE((b.sector_mv / NULLIF(b.total_benchmark_mv, 0)) * 100.0, 0.0) AS benchmark_metric,
    (COALESCE((p.sector_mv / NULLIF(p.total_portfolio_mv, 0)) * 100.0, 0.0) - 
     COALESCE((b.sector_mv / NULLIF(b.total_benchmark_mv, 0)) * 100.0, 0.0)) AS active_delta
FROM portfolio_holdings p
FULL OUTER JOIN benchmark_holdings b ON p.sector_name = b.sector_name;`,
		dimCol, filterClause, dimCol,
		dimCol, filterClause, dimCol,
	)
}
