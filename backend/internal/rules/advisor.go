package rules

import (
	"fmt"
	"strings"
)

// minFallbackForProposal is the minimum number of fallback hits required
// before the advisor will suggest a Materialized View. Below this volume
// the maintenance cost of the MV outweighs the savings.
const minFallbackForProposal = 1000

// FallbackQueryPattern is the input to the advisor: a summary of one
// query pattern that is currently falling back to the recursive evaluator
// because it is too expensive for the VM fast path.
type FallbackQueryPattern struct {
	TargetBOID    string   `json:"target_bo_id"`
	TargetTable   string   `json:"target_table"`
	GroupByFields []string `json:"group_by_fields"`
	MeasureFields []string `json:"measure_fields"`
	FallbackCount int64    `json:"fallback_count"`
}

// MaterializationProposal is the output of the advisor: a StarRocks MV
// DDL statement that pre-aggregates the hot pattern so subsequent rule
// evaluations can hit the VM fast path instead of the recursive fallback.
type MaterializationProposal struct {
	TargetBOID    string `json:"target_bo_id"`
	SuggestedMV   string `json:"suggested_mv_ddl"`
	EstimatedGain string `json:"estimated_gain"`
}

// AnalyzeFallbackPatterns returns a MaterializationProposal if the
// pattern's FallbackCount clears minFallbackForProposal. Returns nil if
// the pattern is too cold to justify an MV.
//
// MV naming follows the convention mv_auto_{table}_{groupby} so that
// advisor-generated MVs are easily distinguishable from hand-written
// ones in DBA tooling.
func AnalyzeFallbackPatterns(pattern FallbackQueryPattern) (*MaterializationProposal, error) {
	if pattern.TargetTable == "" {
		return nil, fmt.Errorf("target_table is required")
	}
	if len(pattern.GroupByFields) == 0 {
		return nil, fmt.Errorf("at least one group_by_field is required")
	}
	if len(pattern.MeasureFields) == 0 {
		return nil, fmt.Errorf("at least one measure_field is required")
	}
	if pattern.FallbackCount < minFallbackForProposal {
		return nil, nil
	}

	mvName := fmt.Sprintf("mv_auto_%s_%s",
		pattern.TargetTable,
		strings.Join(pattern.GroupByFields, "_"))

	var measureClauses []string
	for _, m := range pattern.MeasureFields {
		measureClauses = append(measureClauses,
			fmt.Sprintf("SUM(%s) AS sum_%s", m, m))
	}

	ddl := fmt.Sprintf(`
CREATE MATERIALIZED VIEW public.%s
BUILD ASYNCHRONOUS
REFRESH DEFERRED MANUAL
DISTRIBUTED BY HASH(%s)
AS
SELECT
    tenant_id,
    %s,
    %s
FROM public.%s
GROUP BY tenant_id, %s;
`,
		mvName,
		pattern.GroupByFields[0],
		strings.Join(pattern.GroupByFields, ", "),
		strings.Join(measureClauses, ", "),
		pattern.TargetTable,
		strings.Join(pattern.GroupByFields, ", "),
	)

	return &MaterializationProposal{
		TargetBOID:    pattern.TargetBOID,
		SuggestedMV:   ddl,
		EstimatedGain: "Reduces fallback latency from ~45ms to <1ms by pre-aggregating hot ratios",
	}, nil
}
