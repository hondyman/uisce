package models

import "fmt"

// AggregateRuleSpec defines cross-record tumbling window reconciliation rules.
// Kept strictly outside vm.RuleNode as it operates on multi-row aggregations during
// scheduled pushdown rather than individual record writes.
type AggregateRuleSpec struct {
	Window     string   `json:"window"`      // "1h", "1d" (tumbling only, v1)
	Aggregate  string   `json:"aggregate"`   // "sum" | "count" | "avg" | "max" | "min"
	Field      string   `json:"field"`       // semantic term, MAPS_TO-resolved
	GroupBy    []string `json:"group_by"`    // e.g. ["account_id", "broker_id"]
	TimeColumn string   `json:"time_column"` // semantic term for event timestamp
	Threshold  float64  `json:"threshold"`
	Comparison string   `json:"comparison"`  // "greater_than", "greater_or_equal", "less_than", "less_or_equal", "equals", "not_equals"
}

// Validate checks internal consistency of an AggregateRuleSpec.
func (s *AggregateRuleSpec) Validate() error {
	switch s.Window {
	case "1h", "1d", "hour", "day":
		// valid
	default:
		return fmt.Errorf("unsupported window %q: only '1h' and '1d' tumbling windows supported in v1", s.Window)
	}

	switch s.Aggregate {
	case "sum", "count", "avg", "max", "min":
		// valid
	default:
		return fmt.Errorf("unsupported aggregate %q: must be sum, count, avg, max, or min", s.Aggregate)
	}

	if s.Field == "" && s.Aggregate != "count" {
		return fmt.Errorf("field is required for aggregate %q", s.Aggregate)
	}
	if s.TimeColumn == "" {
		return fmt.Errorf("time_column is required for windowed aggregation")
	}

	switch s.Comparison {
	case "greater_than", "greater_or_equal", "less_than", "less_or_equal", "equals", "not_equals":
		// valid
	default:
		return fmt.Errorf("unsupported comparison %q", s.Comparison)
	}

	return nil
}
