package render

import (
	"fmt"
	"strings"
)

// GeneratePreAggregationDDL creates SQL DDL for the given pre-aggregation.
// If pre.SQL is provided, it will be used as the body. Otherwise a simple
// rollup SELECT is generated that assumes the cubeName refers to an accessible
// relation or view (templates can override with explicit SQL for nested cases).
func (s *Service) GeneratePreAggregationDDL(cubeName string, pre PreAggregation) (string, error) {
	// Pick output object name
	name := pre.PreAggregatedTableName
	if name == "" {
		// safe default
		name = fmt.Sprintf("%s__%s", cubeName, pre.Name)
	}

	// If user provided custom SQL, use it directly
	if strings.TrimSpace(pre.SQL) != "" {
		// wrap in materialized view or table depending on storage
		if pre.Storage == "table" {
			return fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s AS %s;", name, pre.SQL), nil
		}
		// default to materialized view
		return fmt.Sprintf("CREATE MATERIALIZED VIEW IF NOT EXISTS %s AS %s;", name, pre.SQL), nil
	}

	// Fallback rollup generation
	dims := []string{}
	for _, d := range pre.Dimensions {
		dims = append(dims, d)
	}

	aggs := []string{}
	for _, m := range pre.Measures {
		// naive default aggregation: SUM
		aggs = append(aggs, fmt.Sprintf("SUM(%s) AS %s", m, m))
	}

	if len(aggs) == 0 {
		return "", fmt.Errorf("no measures specified for pre-aggregation %s.%s", cubeName, pre.Name)
	}

	selectCols := strings.Join(dims, ", ")
	if selectCols != "" {
		selectCols = selectCols + ", "
	}
	selectCols = selectCols + strings.Join(aggs, ", ")

	// Use cubeName as source relation; templates should override if needed
	body := fmt.Sprintf("SELECT %s FROM %s GROUP BY %s", selectCols, cubeName, strings.Join(dims, ", "))

	if pre.Storage == "table" {
		return fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s AS %s;", name, body), nil
	}
	return fmt.Sprintf("CREATE MATERIALIZED VIEW IF NOT EXISTS %s AS %s;", name, body), nil
}
