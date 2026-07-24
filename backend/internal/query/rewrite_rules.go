package query

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"regexp"
	"strings"
	"time"
)

// initializeStaticRules sets up the static governance and optimization rules
func (e *RewriteEngine) initializeStaticRules() {
	e.staticRules = []RewriteRule{
		{
			Name:        "remove_disallowed_columns",
			Description: "Remove columns not allowed by the evaluation decision",
			Priority:    100,
			Condition: func(ctx *RewriteContext) bool {
				return ctx.Decision.Decision == "partial" && len(ctx.Decision.AllowedScopes) > 0
			},
			Action: e.removeDisallowedColumns,
		},
		{
			Name:        "inject_tenant_filter",
			Description: "Inject tenant isolation filter",
			Priority:    90,
			Condition: func(ctx *RewriteContext) bool {
				return ctx.TenantID != ""
			},
			Action: e.injectTenantFilter,
		},
		{
			Name:        "inject_row_filters",
			Description: "Inject row-level security filters from pruning hints",
			Priority:    80,
			Condition: func(ctx *RewriteContext) bool {
				return len(ctx.PruningHints.RowFilters) > 0
			},
			Action: e.injectRowFilters,
		},
		{
			Name:        "replace_uncertified_metrics",
			Description: "Replace uncertified metrics with certified alternatives",
			Priority:    70,
			Condition: func(ctx *RewriteContext) bool {
				return e.hasUncertifiedMetrics(ctx)
			},
			Action: e.replaceUncertifiedMetrics,
		},
		{
			Name:        "optimize_column_selection",
			Description: "Select only necessary columns based on pruning hints",
			Priority:    60,
			Condition: func(ctx *RewriteContext) bool {
				return len(ctx.PruningHints.Columns) > 0
			},
			Action: e.optimizeColumnSelection,
		},
	}
}

// initializeAIRules sets up AI-powered optimization rules
func (e *RewriteEngine) initializeAIRules() {
	e.aiRules = []AIRewriteRule{
		{
			Name:        "suggest_join_optimization",
			Description: "Suggest alternative joins for better performance",
			Priority:    50,
			Condition: func(ctx *RewriteContext) bool {
				return strings.Contains(strings.ToUpper(ctx.RewrittenQuery), "JOIN")
			},
			Suggestion: e.suggestJoinOptimization,
		},
		{
			Name:        "suggest_filter_pushdown",
			Description: "Suggest pushing filters down to reduce data scanned",
			Priority:    40,
			Condition: func(ctx *RewriteContext) bool {
				return e.hasSubqueriesOrViews(ctx.RewrittenQuery)
			},
			Suggestion: e.suggestFilterPushdown,
		},
		{
			Name:        "suggest_index_usage",
			Description: "Suggest adding filters to leverage existing indexes",
			Priority:    30,
			Condition: func(ctx *RewriteContext) bool {
				return !strings.Contains(strings.ToUpper(ctx.RewrittenQuery), "WHERE")
			},
			Suggestion: e.suggestIndexUsage,
		},
	}
}

// applyStaticRules applies all static rules in priority order
func (e *RewriteEngine) applyStaticRules(ctx *RewriteContext) error {
	// Sort rules by priority (highest first)
	rules := make([]RewriteRule, len(e.staticRules))
	copy(rules, e.staticRules)

	for i := 0; i < len(rules)-1; i++ {
		for j := i + 1; j < len(rules); j++ {
			if rules[i].Priority < rules[j].Priority {
				rules[i], rules[j] = rules[j], rules[i]
			}
		}
	}

	for _, rule := range rules {
		if rule.Condition(ctx) {
			before := ctx.RewrittenQuery
			if err := rule.Action(ctx); err != nil {
				return fmt.Errorf("rule %s failed: %w", rule.Name, err)
			}

			if before != ctx.RewrittenQuery {
				ctx.AppliedRules = append(ctx.AppliedRules, AppliedRule{
					RuleName:    rule.Name,
					Description: rule.Description,
					Before:      before,
					After:       ctx.RewrittenQuery,
					Reason:      fmt.Sprintf("Applied %s rule", rule.Name),
					Timestamp:   time.Now(),
				})
			}
		}
	}

	return nil
}

// applyAIRules applies AI-powered rules and generates suggestions
func (e *RewriteEngine) applyAIRules(ctx *RewriteContext) ([]RewriteSuggestion, error) {
	var suggestions []RewriteSuggestion

	for _, rule := range e.aiRules {
		if rule.Condition(ctx) {
			suggestion, err := rule.Suggestion(ctx)
			if err != nil {
				// Log error but continue with other rules
				fmt.Printf("AI rule %s failed: %v\n", rule.Name, err)
				continue
			}
			if suggestion != nil {
				suggestions = append(suggestions, *suggestion)
			}
		}
	}

	return suggestions, nil
}

// generatePerformanceTips generates performance optimization tips
func (e *RewriteEngine) generatePerformanceTips(ctx *RewriteContext) []string {
	var tips []string

	query := strings.ToUpper(ctx.RewrittenQuery)

	if !strings.Contains(query, "WHERE") {
		tips = append(tips, "Consider adding WHERE clauses to reduce data scanned")
	}

	if strings.Contains(query, "SELECT *") {
		tips = append(tips, "Avoid SELECT * - specify only needed columns")
	}

	if strings.Contains(query, "ORDER BY") && !strings.Contains(query, "LIMIT") {
		tips = append(tips, "Consider adding LIMIT when using ORDER BY")
	}

	if len(ctx.PruningHints.Columns) > 10 {
		tips = append(tips, "Query selects many columns - consider if all are needed")
	}

	return tips
}

// generateComplianceNotes generates compliance-related notes
func (e *RewriteEngine) generateComplianceNotes(ctx *RewriteContext) []string {
	var notes []string

	if ctx.Decision.Decision == "partial" {
		notes = append(notes, fmt.Sprintf("Query restricted to scopes: %s", strings.Join(ctx.Decision.AllowedScopes, ", ")))
	}

	if len(ctx.PruningHints.RowFilters) > 0 {
		notes = append(notes, "Row-level security filters applied")
	}

	if ctx.TenantID != "" {
		notes = append(notes, fmt.Sprintf("Tenant isolation enforced for: %s", ctx.TenantID))
	}

	return notes
}

// generateRewriteID generates a unique ID for the rewrite operation
func generateRewriteID() string {
	bytes := make([]byte, 16)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

// Static Rule Implementations

func (e *RewriteEngine) removeDisallowedColumns(ctx *RewriteContext) error {
	// Get schema to map scopes to columns
	schema, err := e.schemaProvider.GetAssetSchema(ctx.AssetID)
	if err != nil {
		return fmt.Errorf("failed to get schema: %w", err)
	}

	// Build set of allowed columns
	allowedColumns := make(map[string]bool)
	for _, scope := range ctx.Decision.AllowedScopes {
		if cols, ok := schema.ColumnsByScope[scope]; ok {
			for _, col := range cols {
				allowedColumns[col] = true
			}
		}
	}

	// Parse and modify SELECT clause
	query := ctx.RewrittenQuery
	selectRegex := regexp.MustCompile(`(?i)SELECT\s+(.+?)\s+FROM`)
	matches := selectRegex.FindStringSubmatch(query)
	if len(matches) < 2 {
		return nil // No SELECT clause found
	}

	selectClause := matches[1]
	if strings.ToUpper(strings.TrimSpace(selectClause)) == "*" {
		// Replace * with allowed columns
		var cols []string
		for col := range allowedColumns {
			cols = append(cols, col)
		}
		newSelectClause := strings.Join(cols, ", ")
		ctx.RewrittenQuery = strings.Replace(query, selectClause, newSelectClause, 1)
	} else {
		// Parse individual columns and filter
		columns := strings.Split(selectClause, ",")
		var filteredCols []string
		for _, col := range columns {
			col = strings.TrimSpace(col)
			if allowedColumns[col] {
				filteredCols = append(filteredCols, col)
			}
		}
		newSelectClause := strings.Join(filteredCols, ", ")
		ctx.RewrittenQuery = strings.Replace(query, selectClause, newSelectClause, 1)
	}

	return nil
}

func (e *RewriteEngine) injectTenantFilter(ctx *RewriteContext) error {
	query := strings.ToUpper(ctx.RewrittenQuery)

	if !strings.Contains(query, "WHERE") {
		ctx.RewrittenQuery += fmt.Sprintf(" WHERE tenant_id = '%s'", ctx.TenantID)
	} else {
		ctx.RewrittenQuery = strings.Replace(ctx.RewrittenQuery, " WHERE ", fmt.Sprintf(" WHERE tenant_id = '%s' AND ", ctx.TenantID), 1)
	}

	return nil
}

func (e *RewriteEngine) injectRowFilters(ctx *RewriteContext) error {
	if len(ctx.PruningHints.RowFilters) == 0 {
		return nil
	}

	filters := strings.Join(ctx.PruningHints.RowFilters, " AND ")
	query := strings.ToUpper(ctx.RewrittenQuery)

	if !strings.Contains(query, "WHERE") {
		ctx.RewrittenQuery += " WHERE " + filters
	} else {
		ctx.RewrittenQuery = strings.Replace(ctx.RewrittenQuery, " WHERE ", " WHERE "+filters+" AND ", 1)
	}

	return nil
}

func (e *RewriteEngine) replaceUncertifiedMetrics(ctx *RewriteContext) error {
	// This is a simplified implementation - in practice, you'd have a mapping
	// of uncertified metrics to their certified alternatives
	uncertifiedToCertified := map[string]string{
		"net_margin":   "certified_net_margin",
		"gross_profit": "certified_gross_profit",
	}

	query := ctx.RewrittenQuery
	for uncertified, certified := range uncertifiedToCertified {
		query = strings.ReplaceAll(query, uncertified, certified)
	}

	ctx.RewrittenQuery = query
	return nil
}

func (e *RewriteEngine) optimizeColumnSelection(ctx *RewriteContext) error {
	if len(ctx.PruningHints.Columns) == 0 {
		return nil
	}

	query := ctx.RewrittenQuery
	selectRegex := regexp.MustCompile(`(?i)SELECT\s+(.+?)\s+FROM`)
	matches := selectRegex.FindStringSubmatch(query)
	if len(matches) < 2 {
		return nil
	}

	selectClause := matches[1]
	if strings.ToUpper(strings.TrimSpace(selectClause)) == "*" {
		columns := strings.Join(ctx.PruningHints.Columns, ", ")
		ctx.RewrittenQuery = strings.Replace(query, selectClause, columns, 1)
	}

	return nil
}

// AI Rule Implementations

func (e *RewriteEngine) suggestJoinOptimization(ctx *RewriteContext) (*RewriteSuggestion, error) {
	query := strings.ToUpper(ctx.RewrittenQuery)

	// Simple heuristic: if there are multiple JOINs, suggest optimizing
	joinCount := strings.Count(query, "JOIN")
	if joinCount > 2 {
		return &RewriteSuggestion{
			Description: "Consider optimizing multiple JOINs",
			QueryDiff:   "Multiple JOINs detected - consider restructuring query or adding indexes",
			Confidence:  0.7,
			Reasoning:   "Queries with multiple JOINs can benefit from optimization",
		}, nil
	}

	return nil, nil
}

func (e *RewriteEngine) suggestFilterPushdown(ctx *RewriteContext) (*RewriteSuggestion, error) {
	query := strings.ToUpper(ctx.RewrittenQuery)

	if strings.Contains(query, "SELECT") && strings.Contains(query, "FROM") && strings.Contains(query, "(") {
		return &RewriteSuggestion{
			Description: "Consider pushing filters into subqueries",
			QueryDiff:   "Move WHERE conditions into subqueries to reduce data processed",
			Confidence:  0.8,
			Reasoning:   "Filter pushdown can significantly improve query performance",
		}, nil
	}

	return nil, nil
}

func (e *RewriteEngine) suggestIndexUsage(ctx *RewriteContext) (*RewriteSuggestion, error) {
	return &RewriteSuggestion{
		Description: "Consider adding WHERE conditions",
		QueryDiff:   "Add filters to leverage database indexes",
		Confidence:  0.6,
		Reasoning:   "Indexed columns in WHERE clauses improve query performance",
	}, nil
}

// Helper functions

func (e *RewriteEngine) hasUncertifiedMetrics(ctx *RewriteContext) bool {
	uncertifiedMetrics := []string{"net_margin", "gross_profit"}
	query := strings.ToLower(ctx.RewrittenQuery)

	for _, metric := range uncertifiedMetrics {
		if strings.Contains(query, metric) {
			return true
		}
	}

	return false
}

func (e *RewriteEngine) hasSubqueriesOrViews(query string) bool {
	query = strings.ToUpper(query)
	return strings.Contains(query, "SELECT") && strings.Contains(query, "(") ||
		strings.Contains(query, "VIEW")
}
