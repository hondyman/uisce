package analytics

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/hondyman/uisce/backend/internal/models"
	"github.com/jmoiron/sqlx"
)

type WindowEvaluationResult struct {
	RuleID         string            `json:"rule_id"`
	RuleKey        string            `json:"rule_key"`
	RuleName       string            `json:"rule_name"`
	Violations     int               `json:"violations"`
	PersistedCount int               `json:"persisted_count"`
	ViolationsList []ViolationRecord `json:"violations_list,omitempty"`
}

// ExecuteWindowRule executes a tumbling-window aggregation rule against a data-plane table,
// emitting group-level violation records with full remediation queue context.
func ExecuteWindowRule(
	ctx context.Context,
	dataDB *sqlx.DB,
	alphaDB *sqlx.DB,
	spec models.AggregateRuleSpec,
	table string,
	snap *RuleSnapshot,
	rule *EvaluableRule,
	windowStart, windowEnd time.Time,
) (*WindowEvaluationResult, error) {
	if dataDB == nil {
		return nil, fmt.Errorf("dataDB connection is required")
	}

	compiled, err := CompileWindowRuleSQL(spec, table, snap.ColumnMap)
	if err != nil {
		return nil, fmt.Errorf("compile window rule %q: %w", rule.RuleKey, err)
	}

	rows, err := dataDB.QueryxContext(ctx, compiled.Query, snap.TenantID, windowStart, windowEnd)
	if err != nil {
		return nil, fmt.Errorf("execute window aggregation for rule %q: %w", rule.RuleKey, err)
	}
	defer rows.Close()

	res := &WindowEvaluationResult{
		RuleID:   rule.RuleID,
		RuleKey:  rule.RuleKey,
		RuleName: rule.RuleName,
	}

	for rows.Next() {
		rowMap := make(map[string]interface{})
		if err := rows.MapScan(rowMap); err != nil {
			return nil, fmt.Errorf("scan window row: %w", err)
		}

		// Extract window_start, window_end, metric_value
		wStartVal := rowMap["window_start"]
		wEndVal := rowMap["window_end"]
		metricVal := rowMap["metric_value"]

		// Build group dimensions
		groupDimensions := make(map[string]interface{})
		groupSlugParts := make([]string, 0, len(compiled.GroupCols))
		for _, col := range compiled.GroupCols {
			val := rowMap[col]
			groupDimensions[col] = val
			groupSlugParts = append(groupSlugParts, fmt.Sprintf("%v", val))
		}

		groupSlug := strings.Join(groupSlugParts, "_")
		if groupSlug == "" {
			groupSlug = "all"
		}

		wStartTimeStr := fmt.Sprintf("%v", wStartVal)
		recordID := fmt.Sprintf("group:%s:%s:%s", snap.BOName, groupSlug, wStartTimeStr)

		violationContext := map[string]interface{}{
			"group_by":     groupDimensions,
			"window_start": wStartVal,
			"window_end":   wEndVal,
			"metric_value": metricVal,
			"threshold":    spec.Threshold,
			"aggregate":    spec.Aggregate,
			"field":        spec.Field,
			"comparison":   spec.Comparison,
		}

		v := ViolationRecord{
			TenantID:     snap.TenantID,
			RuleID:       rule.RuleID,
			RuleKey:      rule.RuleKey,
			RuleVersion:  rule.RuleVersion,
			RuleName:     rule.RuleName,
			BOKey:        snap.BOName,
			Severity:     rule.Severity,
			RecordID:     recordID,
			Message:      fmt.Sprintf("Window aggregate threshold breached: %s(%s) %s %v (threshold %v)", spec.Aggregate, spec.Field, spec.Comparison, metricVal, spec.Threshold),
			Fields:       compiled.GroupCols,
			Context:      violationContext,
			WriteBlocked: false, // window reconciliations never block OLTP writes
			RuleError:    false,
		}

		res.Violations++
		res.ViolationsList = append(res.ViolationsList, v)

		// Persist to alpha ledger
		if alphaDB != nil {
			if perr := PersistViolation(ctx, alphaDB, v); perr != nil && perr != sql.ErrNoRows {
				return nil, fmt.Errorf("persist window violation: %w", perr)
			}
			res.PersistedCount++
		}
	}

	return res, nil
}
