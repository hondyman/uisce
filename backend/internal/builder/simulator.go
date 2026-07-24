// Package builder provides impact simulation against historical data
package builder

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"
)

// ImpactReport contains the results of a rule simulation
type ImpactReport struct {
	TotalRecords    int64                    `json:"total_records"`
	AffectedRecords int64                    `json:"affected_records"`
	ImpactPercent   float64                  `json:"impact_percent"`
	SampleMatches   []map[string]interface{} `json:"sample_matches"`
	TimeRange       string                   `json:"time_range"`
	Warning         string                   `json:"warning,omitempty"`
	QueryExecuted   string                   `json:"query_executed,omitempty"`
}

// Simulator performs impact analysis against StarRocks historical data
type Simulator struct {
	StarRocksDB *sql.DB
	TableName   string
}

// NewSimulator creates a new impact simulator
func NewSimulator(starrocksDB *sql.DB, tableName string) *Simulator {
	return &Simulator{
		StarRocksDB: starrocksDB,
		TableName:   tableName,
	}
}

// SimulateRuleImpact runs the rule against historical data and returns impact metrics
func (s *Simulator) SimulateRuleImpact(ctx context.Context, rule UIRule, timeRange string) (*ImpactReport, error) {
	if s.StarRocksDB == nil {
		// Return mock data if no StarRocks connection
		return s.mockSimulation(rule, timeRange), nil
	}

	// Parse time range
	duration, err := parseTimeRange(timeRange)
	if err != nil {
		duration = 24 * time.Hour // Default to 24h
	}
	cutoffTime := time.Now().Add(-duration)

	// Build WHERE clause from conditions
	whereClause, err := s.buildWhereClause(rule.Conditions, rule.Logic)
	if err != nil {
		return nil, fmt.Errorf("failed to build WHERE clause: %w", err)
	}

	// Get total records in time range
	totalQuery := fmt.Sprintf(
		"SELECT COUNT(*) FROM %s WHERE created_at >= ?",
		s.TableName,
	)
	var totalRecords int64
	err = s.StarRocksDB.QueryRowContext(ctx, totalQuery, cutoffTime).Scan(&totalRecords)
	if err != nil && err != sql.ErrNoRows {
		return nil, fmt.Errorf("failed to count total records: %w", err)
	}

	// Get affected records
	affectedQuery := fmt.Sprintf(
		"SELECT COUNT(*) FROM %s WHERE created_at >= ? AND (%s)",
		s.TableName,
		whereClause,
	)
	var affectedRecords int64
	err = s.StarRocksDB.QueryRowContext(ctx, affectedQuery, cutoffTime).Scan(&affectedRecords)
	if err != nil && err != sql.ErrNoRows {
		return nil, fmt.Errorf("failed to count affected records: %w", err)
	}

	// Get sample matches
	sampleQuery := fmt.Sprintf(
		"SELECT * FROM %s WHERE created_at >= ? AND (%s) LIMIT 5",
		s.TableName,
		whereClause,
	)
	samples, err := s.querySamples(ctx, sampleQuery, cutoffTime)
	if err != nil {
		// Non-fatal, just log
		samples = []map[string]interface{}{}
	}

	// Calculate impact
	var impactPercent float64
	if totalRecords > 0 {
		impactPercent = float64(affectedRecords) / float64(totalRecords) * 100
	}

	// Generate warning message
	var warning string
	if impactPercent > 10 {
		warning = fmt.Sprintf("⚠️ CAUTION: This rule would have affected %.1f%% of records (%d out of %d) in the last %s. Consider reviewing the conditions.",
			impactPercent, affectedRecords, totalRecords, timeRange)
	} else if impactPercent > 5 {
		warning = fmt.Sprintf("This rule would have affected %.1f%% of records (%d out of %d) in the last %s.",
			impactPercent, affectedRecords, totalRecords, timeRange)
	} else {
		warning = fmt.Sprintf("This rule would have affected %d records (%.2f%%) in the last %s.",
			affectedRecords, impactPercent, timeRange)
	}

	return &ImpactReport{
		TotalRecords:    totalRecords,
		AffectedRecords: affectedRecords,
		ImpactPercent:   impactPercent,
		SampleMatches:   samples,
		TimeRange:       timeRange,
		Warning:         warning,
		QueryExecuted:   affectedQuery,
	}, nil
}

// buildWhereClause converts conditions to SQL WHERE clause
func (s *Simulator) buildWhereClause(conditions []Condition, logic string) (string, error) {
	if len(conditions) == 0 {
		return "1=1", nil
	}

	clauses := make([]string, 0, len(conditions))
	for _, cond := range conditions {
		clause, err := s.conditionToSQL(cond)
		if err != nil {
			return "", err
		}
		clauses = append(clauses, clause)
	}

	connector := " AND "
	if strings.ToUpper(logic) == "OR" {
		connector = " OR "
	}

	return strings.Join(clauses, connector), nil
}

// conditionToSQL converts a single condition to SQL
func (s *Simulator) conditionToSQL(cond Condition) (string, error) {
	field := cond.Field
	op := cond.Operator
	value := cond.Value

	// Format value for SQL
	var valueStr string
	switch v := value.(type) {
	case string:
		valueStr = fmt.Sprintf("'%s'", strings.ReplaceAll(v, "'", "''"))
	case float64:
		if v == float64(int(v)) {
			valueStr = fmt.Sprintf("%d", int(v))
		} else {
			valueStr = fmt.Sprintf("%f", v)
		}
	case int:
		valueStr = fmt.Sprintf("%d", v)
	case bool:
		if v {
			valueStr = "TRUE"
		} else {
			valueStr = "FALSE"
		}
	case []interface{}:
		items := make([]string, len(v))
		for i, item := range v {
			if s, ok := item.(string); ok {
				items[i] = fmt.Sprintf("'%s'", strings.ReplaceAll(s, "'", "''"))
			} else {
				items[i] = fmt.Sprintf("%v", item)
			}
		}
		valueStr = fmt.Sprintf("(%s)", strings.Join(items, ", "))
	default:
		valueStr = fmt.Sprintf("%v", v)
	}

	// Map operators to SQL
	switch op {
	case ">", "<", ">=", "<=", "=", "==":
		if op == "==" {
			op = "="
		}
		return fmt.Sprintf("%s %s %s", field, op, valueStr), nil
	case "!=":
		return fmt.Sprintf("%s <> %s", field, valueStr), nil
	case "IN":
		return fmt.Sprintf("%s IN %s", field, valueStr), nil
	case "NOT_IN":
		return fmt.Sprintf("%s NOT IN %s", field, valueStr), nil
	case "contains":
		strVal := value.(string)
		return fmt.Sprintf("%s LIKE '%%%s%%'", field, strings.ReplaceAll(strVal, "'", "''")), nil
	case "startsWith":
		strVal := value.(string)
		return fmt.Sprintf("%s LIKE '%s%%'", field, strings.ReplaceAll(strVal, "'", "''")), nil
	case "endsWith":
		strVal := value.(string)
		return fmt.Sprintf("%s LIKE '%%%s'", field, strings.ReplaceAll(strVal, "'", "''")), nil
	default:
		return fmt.Sprintf("%s = %s", field, valueStr), nil
	}
}

// querySamples retrieves sample matching records
func (s *Simulator) querySamples(ctx context.Context, query string, args ...interface{}) ([]map[string]interface{}, error) {
	rows, err := s.StarRocksDB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		return nil, err
	}

	var samples []map[string]interface{}
	for rows.Next() {
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))
		for i := range values {
			valuePtrs[i] = &values[i]
		}

		if err := rows.Scan(valuePtrs...); err != nil {
			continue
		}

		row := make(map[string]interface{})
		for i, col := range columns {
			row[col] = values[i]
		}
		samples = append(samples, row)
	}

	return samples, nil
}

// parseTimeRange converts time range string to duration
func parseTimeRange(tr string) (time.Duration, error) {
	tr = strings.ToLower(strings.TrimSpace(tr))
	switch tr {
	case "1h":
		return time.Hour, nil
	case "6h":
		return 6 * time.Hour, nil
	case "12h":
		return 12 * time.Hour, nil
	case "24h", "1d":
		return 24 * time.Hour, nil
	case "7d", "1w":
		return 7 * 24 * time.Hour, nil
	case "30d", "1m":
		return 30 * 24 * time.Hour, nil
	default:
		return 0, fmt.Errorf("unknown time range: %s", tr)
	}
}

// mockSimulation returns simulated impact data when StarRocks is unavailable
func (s *Simulator) mockSimulation(rule UIRule, timeRange string) *ImpactReport {
	// Generate plausible mock data based on conditions
	totalRecords := int64(10000)

	// Estimate affected records based on conditions
	var affectedRatio float64
	for _, cond := range rule.Conditions {
		switch cond.Operator {
		case ">", ">=":
			affectedRatio += 0.15
		case "<", "<=":
			affectedRatio += 0.15
		case "==":
			affectedRatio += 0.05
		case "IN":
			affectedRatio += 0.10
		default:
			affectedRatio += 0.10
		}
	}
	if affectedRatio > 0.5 {
		affectedRatio = 0.5
	}

	affectedRecords := int64(float64(totalRecords) * affectedRatio)
	impactPercent := affectedRatio * 100

	warning := fmt.Sprintf("This rule would have affected %d records (%.2f%%) in the last %s. (Simulated data)",
		affectedRecords, impactPercent, timeRange)

	return &ImpactReport{
		TotalRecords:    totalRecords,
		AffectedRecords: affectedRecords,
		ImpactPercent:   impactPercent,
		SampleMatches: []map[string]interface{}{
			{"id": "TRD-001", "amount": 1500000, "currency": "USD"},
			{"id": "TRD-002", "amount": 2100000, "currency": "EUR"},
		},
		TimeRange: timeRange,
		Warning:   warning,
	}
}
