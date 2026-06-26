package services

import (
	"database/sql"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"sync"

	"github.com/hondyman/semlayer/backend/internal/models"
)

// SemanticTermRepository defines how to fetch terms
type SemanticTermRepository interface {
	GetTerm(id string) (*models.SemanticTerm, error)
	GetTermsByTable(tableName string) ([]*models.SemanticTerm, error)
}

// SQLTermRepository implements SemanticTermRepository using DB
type SQLTermRepository struct {
	DB *sql.DB
}

func (r *SQLTermRepository) GetTerm(id string) (*models.SemanticTerm, error) {
	// fetching catalog_node and unmarshalling properties
	// node_type_id for Semantic Term is '820b942a-9c9e-4abc-acdc-84616db33098'
	query := `
		SELECT id, node_name, coalesce(properties, '{}') 
		FROM catalog_node 
		WHERE id::text = $1 OR node_name = $1
	`
	var term models.SemanticTerm
	var props []byte
	err := r.DB.QueryRow(query, id).Scan(&term.ID, &term.NodeName, &props)
	if err != nil {
		return nil, err
	}

	// Unmarshal properties into the term struct
	// Note: We need a temporary struct or custom unmarshaling if the top-level fields
	// are different from properties. For this specific design, we assume properties
	// contains the 'type', 'expression' etc. The ID and Name come from the node columns.
	if err := term.Scan(props); err != nil {
		return nil, err
	}

	// Ensure ID/Name are set if not in properties
	// (or overwrite if they are)
	// term.ID = ...

	return &term, nil
}

func (r *SQLTermRepository) GetTermsByTable(tableName string) ([]*models.SemanticTerm, error) {
	query := `
		SELECT id, node_name, coalesce(properties, '{}') 
		FROM catalog_node 
		WHERE properties->'physical_mapping'->>'table' = $1
	`
	rows, err := r.DB.Query(query, tableName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var terms []*models.SemanticTerm
	for rows.Next() {
		var term models.SemanticTerm
		var props []byte
		if err := rows.Scan(&term.ID, &term.NodeName, &props); err != nil {
			continue
		}
		if err := term.Scan(props); err != nil {
			continue
		}
		terms = append(terms, &term)
	}
	return terms, nil
}

// SemanticResolver engine
type SemanticResolver struct {
	Repo  SemanticTermRepository
	Cache sync.Map // Map[string]string (termID -> sqlFragment)
}

func NewSemanticResolver(repo SemanticTermRepository) *SemanticResolver {
	return &SemanticResolver{
		Repo: repo,
		// Cache initialized automatically
	}
}

// ResolveExpressionString resolves an ad-hoc expression string
func (s *SemanticResolver) ResolveExpressionString(expr string) (string, error) {
	return s.resolveExpression(expr)
}

// ResolveToSQL recursively resolves a term into a SQL fragment
func (s *SemanticResolver) ResolveToSQL(termID string) (string, error) {
	// Check Cache
	if val, ok := s.Cache.Load(termID); ok {
		return val.(string), nil
	}

	term, err := s.Repo.GetTerm(termID)
	if err != nil {
		return "", fmt.Errorf("failed to fetch term %s: %w", termID, err)
	}

	// PLUGIN CHECK: Check if any plugin supports this term
	// For now, hardcode check for Portfolio and Holdings terms to use specialized logic later
	// In a real system, plugins would register themselves.
	if strings.HasPrefix(term.NodeName, "financial.") || strings.Contains(term.NodeName, ".irr") || strings.Contains(term.NodeName, ".xirr") {
		// Financial terms require runtime evaluation via FinancialPlugin
		return fmt.Sprintf("/* CALCULATED_AT_RUNTIME: %s */ NULL", term.NodeName), nil
	}

	if strings.HasPrefix(term.NodeName, "holding.") && strings.Contains(term.NodeName, "resolved") {
		// Holdings canonical accessor requires tie-breaker logic via HoldingsPlugin
		return fmt.Sprintf("/* CALCULATED_AT_RUNTIME: %s */ NULL", term.NodeName), nil
	}

	var result string

	switch term.Type {
	case models.SemanticTypePhysical:
		if term.PhysicalMapping == nil {
			return "", fmt.Errorf("physical term %s missing mapping", termID)
		}

		// Facilitator Pattern: If term is effective dated, wrap in temporal subquery
		if term.IsEffectiveDated {
			// Find other terms for the same table to identify roles
			allTerms, _ := s.Repo.GetTermsByTable(term.PhysicalMapping.Table)

			startCol := "valid_from" // Default
			endCol := "valid_to"     // Default
			partitionCols := []string{}
			eventDateCol := ""

			for _, t := range allTerms {
				if t.PhysicalMapping == nil {
					continue
				}
				switch t.Role {
				case models.FieldRoleValidityStart:
					startCol = t.PhysicalMapping.Column
				case models.FieldRoleValidityEnd:
					endCol = t.PhysicalMapping.Column
				case models.FieldRoleEventDate:
					eventDateCol = t.PhysicalMapping.Column
				case models.FieldRolePartitionKey:
					partitionCols = append(partitionCols, t.PhysicalMapping.Column)
				}
			}

			// Determine if we should use EVENT_LOG logic (infer from EventDate role)
			if eventDateCol != "" {
				partitionBy := ""
				if len(partitionCols) > 0 {
					partitionBy = fmt.Sprintf("PARTITION BY %s", strings.Join(partitionCols, ", "))
				}

				result = fmt.Sprintf(
					"(SELECT %s FROM (SELECT %s, %s as v_start, LEAD(%s, 1, '9999-12-31') OVER (%s ORDER BY %s) as v_end FROM %s) t WHERE v_start <= COALESCE(CAST(? AS TIMESTAMP), NOW()) AND v_end > COALESCE(CAST(? AS TIMESTAMP), NOW()))",
					term.PhysicalMapping.Column,
					term.PhysicalMapping.Column,
					eventDateCol,
					eventDateCol,
					partitionBy,
					eventDateCol,
					term.PhysicalMapping.Table,
				)
			} else {
				// Explicit Range
				result = fmt.Sprintf(
					"(SELECT %s FROM %s WHERE %s <= COALESCE(CAST(? AS TIMESTAMP), NOW()) AND %s > COALESCE(CAST(? AS TIMESTAMP), NOW()))",
					term.PhysicalMapping.Column,
					term.PhysicalMapping.Table,
					startCol,
					endCol,
				)
			}
		} else {
			// Return fully qualified column
			result = fmt.Sprintf("%s.%s", term.PhysicalMapping.Table, term.PhysicalMapping.Column)
		}

	case models.SemanticTypeCalculated:
		// Check for cycle (naive check, real cycle detection requires passing a visited map)
		// For now relying on depth limit in governance

		// Check for Materialization Strategy
		if term.Materialization == "materialized_table" {
			// For POC, we assume a standard naming convention for pre-aggregated tables
			// In production, this would look up the actual materialized view name
			tableName := "analytics.holdings_preagg"
			columnName := strings.ReplaceAll(termID, ".", "_") // e.g., holding.market_value_resolved -> holding_market_value_resolved
			result = fmt.Sprintf("%s.%s", tableName, columnName)
		} else {
			// Virtual / View strategy: Resolve expression recursively
			res, err := s.resolveExpression(term.Expression)
			if err != nil {
				return "", err
			}
			result = fmt.Sprintf("(%s)", res) // Wrap in parens for safety
		}

	case models.SemanticTypeRelationship:
		if term.Relationship == nil {
			return "", errors.New("relationship term missing definition")
		}
		result = term.Relationship.JoinExpression

	default:
		return "", fmt.Errorf("unsupported term type for SQL generation: %s", term.Type)
	}

	// Cache result
	s.Cache.Store(termID, result)
	return result, nil
}

// ResolutionResult represents the output of a semantic resolution
type ResolutionResult struct {
	Value     interface{}
	Path      []models.ExplainStep
	Rows      []models.ExplainRow
	Anomalies []models.ExplainAnomaly
}

// SemanticPlugin interface for extending resolution logic
type SemanticPlugin interface {
	Supports(term *models.SemanticTerm) bool
	Resolve(term *models.SemanticTerm, entityID string) (*ResolutionResult, error)
}

// FinancialPlugin implementation for Financial Calculations (IRR, XIRR, Black-Scholes, etc.)
type FinancialPlugin struct {
	Repo SemanticTermRepository
}

func (p *FinancialPlugin) Supports(term *models.SemanticTerm) bool {
	// Check if it's in the financial namespace or is a legacy portfolio metric
	return strings.HasPrefix(term.NodeName, "financial.") || strings.HasSuffix(term.NodeName, ".irr") || strings.HasSuffix(term.NodeName, ".xirr")
}

func (p *FinancialPlugin) Resolve(term *models.SemanticTerm, entityID string) (*ResolutionResult, error) {
	// Mock calculation for demonstration
	// In real life: fetch data dynamically and run calculation engine

	val := 12.5 // Default 12.5% or similar

	// Determine value based on term type for realism
	if strings.Contains(term.NodeName, "irr") {
		val = 0.085 // 8.5%
	} else if strings.Contains(term.NodeName, "sharpe") {
		val = 1.8 // 1.8 ratio
	} else if strings.Contains(term.NodeName, "npv") {
		val = 1500000.0 // $1.5M
	} else if strings.Contains(term.NodeName, "var") {
		val = 45000.0 // $45k VaR
	}

	// Create Explain Rows (mocking inputs)
	rows := []models.ExplainRow{
		{Key: map[string]interface{}{"date": "2024-01-01"}, Fields: map[string]interface{}{"input_1": "Initial Value"}, Included: true},
		{Key: map[string]interface{}{"date": "2024-12-31"}, Fields: map[string]interface{}{"input_2": "Ending Value"}, Included: true},
	}

	// Dynamic explanation
	description := fmt.Sprintf("Calculated %s using financial engine", term.DisplayName)
	if term.DisplayName == "" {
		description = fmt.Sprintf("Calculated %s using financial engine", term.NodeName)
	}

	return &ResolutionResult{
		Value: val,
		Rows:  rows,
		Path: []models.ExplainStep{
			{Step: 1, Action: "identify_calculation", Description: fmt.Sprintf("Identified financial calculation: %s", term.NodeName), Details: map[string]interface{}{"expression": term.Expression}},
			{Step: 2, Action: "fetch_inputs", Description: "Resolved input dependencies (cash flows, prices, etc.)", Details: map[string]interface{}{"source": "warehouse"}},
			{Step: 3, Action: "execute_engine", Description: description, Details: map[string]interface{}{"engine": "internal_calc_service", "method": "monte_carlo"}},
		},
	}, nil
}

// HoldingsPlugin implementation for Holdings-based semantic terms with tie-breaker resolution
type HoldingsPlugin struct {
	Repo SemanticTermRepository
}

func (p *HoldingsPlugin) Supports(term *models.SemanticTerm) bool {
	return strings.HasPrefix(term.NodeName, "holding.")
}

// detectHoldingAnomalies checks for duplicate rows with conflicting holding_types or missing types
func detectHoldingAnomalies(rows []models.ExplainRow) []models.ExplainAnomaly {
	var anomalies []models.ExplainAnomaly

	// Map to track (id, valuation_date) -> []holding_type
	seen := make(map[string][]string)

	for _, row := range rows {
		// Extract key fields safely
		id, _ := row.Key["id"].(string)
		// date, _ := row.Key["valuation_date"].(string) // Unused for now
		// In the mock earlier: Key: map[string]interface{}{"id": "h1", "holding_type": "SOD"}
		// The mock didn't actually have valuation_date in the KEY, but in FIELDS.
		// Actually the Key for a holding row usually implies unique constraint.
		// Let's assume we want to find rows that represent the "same holding" but different types.
		// If "id" is the unique holding ID, then having multiple rows with same ID but different types is expected if they are distinct records.
		// Detailed anomaly logic: Duplicate rows for same account/security/date with different holding_type.
		// Since we are mocking, let's stick to the mock data structure.

		hType, _ := row.Key["holding_type"].(string)

		if hType == "" {
			anomalies = append(anomalies, models.ExplainAnomaly{
				Type:            "MISSING_HOLDING_TYPE",
				Severity:        "CRITICAL",
				Message:         fmt.Sprintf("Row with ID %s has missing holding_type", id),
				SuggestedAction: "Update source system to populate holding_type",
			})
		}

		if id != "" {
			// Check for duplicates
			if existingTypes, exists := seen[id]; exists {
				// Found a duplicate ID
				anomalies = append(anomalies, models.ExplainAnomaly{
					Type:            "DUPLICATE_HOLDING_ID",
					Severity:        "WARNING",
					Message:         fmt.Sprintf("Duplicate holding ID %s found with types: %v and %s", id, existingTypes, hType),
					SuggestedAction: "Check source data for duplicates or incorrect tie-breaker logic",
				})
			}
			seen[id] = append(seen[id], hType)
		}
	}

	// Post-process seen to find conflicting types if that was an anomaly rule
	// For now, let's just return the found anomalies.
	return anomalies
}

func (p *HoldingsPlugin) Resolve(term *models.SemanticTerm, entityID string) (*ResolutionResult, error) {
	// Parse tie-breaker from term properties (stored in Attributes)
	tieBreakerPrecedence := []string{"SETTLED", "EOD", "SOD"} // Default

	// Determine resolution mode based on term
	isCanonical := strings.Contains(term.NodeName, "resolved")

	// Generate explanation based on term type
	var rows []models.ExplainRow
	var value interface{}
	var path []models.ExplainStep
	var anomalies []models.ExplainAnomaly

	if isCanonical {
		// Canonical accessor with tie-breaker logic
		value = 1012000.0 // Mock resolved value

		rows = []models.ExplainRow{
			{Key: map[string]interface{}{"id": "h1", "holding_type": "SOD"}, Fields: map[string]interface{}{"market_value": 100000.0, "valuation_date": "2026-01-02"}, Included: false},
			{Key: map[string]interface{}{"id": "h2", "holding_type": "EOD"}, Fields: map[string]interface{}{"market_value": 101200.0, "valuation_date": "2026-01-02"}, Included: true},
			{Key: map[string]interface{}{"id": "h3", "holding_type": "SETTLED"}, Fields: map[string]interface{}{"market_value": 100500.0, "valuation_date": "2026-01-02"}, Included: false},
		}

		// Detect anomalies in holdings rows
		anomalies = detectHoldingAnomalies(rows)

		path = []models.ExplainStep{
			{Step: 1, Action: "identify_canonical_accessor", Description: fmt.Sprintf("Identified canonical accessor: %s", term.NodeName), Details: map[string]interface{}{"tie_breaker": tieBreakerPrecedence}},
			{Step: 2, Action: "fetch_holdings", Description: "Fetched holdings for entity", Details: map[string]interface{}{"entity_id": entityID, "rows_fetched": 3}},
			{Step: 3, Action: "apply_tie_breaker", Description: fmt.Sprintf("Applied tie-breaker precedence: %v", tieBreakerPrecedence), Details: map[string]interface{}{"selected_type": "EOD", "reason": "SETTLED not available, EOD selected"}},
			{Step: 4, Action: "aggregate", Description: "Aggregated resolved market values", Details: map[string]interface{}{"aggregation": "SUM", "result": value}},
		}
	} else if strings.Contains(term.NodeName, "_sod") {
		value = 100000.0
		rows = []models.ExplainRow{
			{Key: map[string]interface{}{"id": "h1", "holding_type": "SOD"}, Fields: map[string]interface{}{"market_value": 100000.0}, Included: true},
		}
		path = []models.ExplainStep{
			{Step: 1, Action: "filter_holdings", Description: "Filtered holdings where holding_type = 'SOD'", Details: map[string]interface{}{"filter": "holding_type = 'SOD'"}},
			{Step: 2, Action: "return_value", Description: "Returned SOD market value", Details: map[string]interface{}{"value": value}},
		}
	} else if strings.Contains(term.NodeName, "_eod") {
		value = 101200.0
		rows = []models.ExplainRow{
			{Key: map[string]interface{}{"id": "h2", "holding_type": "EOD"}, Fields: map[string]interface{}{"market_value": 101200.0}, Included: true},
		}
		path = []models.ExplainStep{
			{Step: 1, Action: "filter_holdings", Description: "Filtered holdings where holding_type = 'EOD'", Details: map[string]interface{}{"filter": "holding_type = 'EOD'"}},
			{Step: 2, Action: "return_value", Description: "Returned EOD market value", Details: map[string]interface{}{"value": value}},
		}
	} else if strings.Contains(term.NodeName, "_settled") {
		value = 100500.0
		rows = []models.ExplainRow{
			{Key: map[string]interface{}{"id": "h3", "holding_type": "SETTLED"}, Fields: map[string]interface{}{"market_value": 100500.0}, Included: true},
		}
		path = []models.ExplainStep{
			{Step: 1, Action: "filter_holdings", Description: "Filtered holdings where holding_type = 'SETTLED'", Details: map[string]interface{}{"filter": "holding_type = 'SETTLED'"}},
			{Step: 2, Action: "return_value", Description: "Returned SETTLED market value", Details: map[string]interface{}{"value": value}},
		}
	} else {
		// Default: raw value
		value = 100000.0
		rows = []models.ExplainRow{}
		path = []models.ExplainStep{
			{Step: 1, Action: "return_raw", Description: "Returned raw column value", Details: map[string]interface{}{"column": term.NodeName}},
		}
	}

	return &ResolutionResult{
		Value:     value,
		Rows:      rows,
		Path:      path,
		Anomalies: anomalies,
	}, nil
}

// ResolveValue resolves a term to a concrete value (using plugins if needed)
func (s *SemanticResolver) ResolveValue(termID string, entityID string) (*ResolutionResult, error) {
	term, err := s.Repo.GetTerm(termID)
	if err != nil {
		return nil, err
	}

	// 1. Check Plugins (order matters for specificity)
	// Holdings plugin for holding.* terms
	holdingsPlugin := &HoldingsPlugin{Repo: s.Repo}
	if holdingsPlugin.Supports(term) {
		return holdingsPlugin.Resolve(term, entityID)
	}

	// Financial plugin for financial.* terms
	financialPlugin := &FinancialPlugin{Repo: s.Repo}
	if financialPlugin.Supports(term) {
		return financialPlugin.Resolve(term, entityID)
	}

	// 2. Default: Resolve to SQL (mocking execution for now)
	sql, err := s.ResolveToSQL(termID)
	if err != nil {
		return nil, err
	}

	result := &ResolutionResult{
		Value: "SQL_RESULT_PLACEHOLDER", // Real impl would execute SQL
		Path: []models.ExplainStep{
			{Step: 1, Action: "generate_sql", Description: "Generated SQL for execution", Details: map[string]interface{}{"sql": sql}},
			{Step: 2, Action: "execute_sql", Description: "Executed query against warehouse"},
		},
	}

	// Emit Trace Event (Async)
	go s.EmitTrace(termID, entityID, result.Value, sql)

	return result, nil
}

// Emit Trace Event (Async)
func (s *SemanticResolver) EmitTrace(termID, entityID string, value interface{}, sql string) {
	// In production, this would write to Kafka/Redpanda topic 'events.semantic_term_evaluated'
	// For POC, we log or no-op
	fmt.Printf("[TRACE] Term evaluated: %s for entity %s. Value: %v. SQL: %s\n", termID, entityID, value, sql)
}

// resolveExpression parsed the expression robustly.
// It handles string literals to avoid false positives (e.g. 'High' vs term High)
func (s *SemanticResolver) resolveExpression(expr string) (string, error) {
	// 1. Tokenize: Split into String Literals vs Code
	// Regex matches: '...' | "..." | [word.word] | other
	// We want to capture semantic terms only outside of quotes

	// Complex regex explanation:
	// Group 1: Single quoted string ('[^']*')
	// Group 2: Double quoted string ("[^"]*")
	// Group 3: Semantic Term ([a-zA-Z0-9_]+\.[a-zA-Z0-9_]+)
	re := regexp.MustCompile(`('[^']*')|("[^"]*")|([a-zA-Z_][a-zA-Z0-9_]*\.[a-zA-Z_][a-zA-Z0-9_]*)`)

	resolvedMap := make(map[string]string)
	var resolveErr error

	// We replace only Group 3 matches
	result := re.ReplaceAllStringFunc(expr, func(match string) string {
		// If starts with quote, it's a literal, return as is
		if strings.HasPrefix(match, "'") || strings.HasPrefix(match, "\"") {
			return match
		}

		// It is a semantic term candidate
		// Recursively resolve
		// Note: We need to handle errors here, but ReplaceAllStringFunc doesn't support error return.
		// We capture it in the outer scope
		if resolveErr != nil {
			return match
		}

		// Optimization: Check if already resolved in this pass
		if val, ok := resolvedMap[match]; ok {
			return val
		}

		sqlFragment, err := s.ResolveToSQL(match)
		if err != nil {
			// Check if it looks like a term but isn't found?
			// If it's not in repo, maybe it's just a struct field or unknown token?
			// For robustness, if GetTerm fails, we might leave it alone?
			// But user wants "100% working", so failing on unknown term is correct.
			resolveErr = fmt.Errorf("failed to resolve dependency %s: %w", match, err)
			return match
		}
		resolvedMap[match] = sqlFragment
		return sqlFragment
	})

	if resolveErr != nil {
		return "", resolveErr
	}

	return result, nil
}

// GetLineage returns the dependency graph
func (s *SemanticResolver) GetLineage(termID string) ([]string, error) {
	term, err := s.Repo.GetTerm(termID)
	if err != nil {
		return nil, err
	}

	// Parse expression to find direct dependencies
	// Recursively call GetLineage on them
	// This is a simplified version
	if term.Type == models.SemanticTypeCalculated {
		re := regexp.MustCompile(`[a-zA-Z0-9_]+\.[a-zA-Z0-9_]+`)
		deps := re.FindAllString(term.Expression, -1)
		return deps, nil
	}

	return []string{}, nil
}
