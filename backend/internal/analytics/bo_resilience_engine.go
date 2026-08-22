package analytics

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// AdditivityScope defines metric aggregation behavior across temporal and dimensional axes.
type AdditivityScope string

const (
	AdditivityFullyAdditive          AdditivityScope = "FULLY_ADDITIVE"
	AdditivitySemiAdditiveAcrossTime AdditivityScope = "SEMI_ADDITIVE_ACROSS_TIME"
	AdditivityNonAdditive            AdditivityScope = "NON_ADDITIVE"
)

// JoinCardinality defines relationship multiplicity for chasm/fan-out protection.
type JoinCardinality string

const (
	CardinalityOneToOne   JoinCardinality = "1:1"
	CardinalityManyToOne  JoinCardinality = "N:1"
	CardinalityOneToMany  JoinCardinality = "1:N"
	CardinalityManyToMany JoinCardinality = "M:N"
)

// BOLifecycleStatus defines the Maker-Checker governance state.
type BOLifecycleStatus string

const (
	StatusDraft           BOLifecycleStatus = "DRAFT"
	StatusPendingApproval BOLifecycleStatus = "PENDING_APPROVAL"
	StatusPublished       BOLifecycleStatus = "PUBLISHED"
	StatusDeprecated      BOLifecycleStatus = "DEPRECATED"
)

// FieldBindingStatus tracks physical schema integrity and drift.
type FieldBindingStatus string

const (
	BindingResolved      FieldBindingStatus = "RESOLVED"
	BindingDriftDegraded FieldBindingStatus = "DRIFT_DEGRADED"
	BindingUnresolved    FieldBindingStatus = "UNRESOLVED"
)

// TieredStoragePair defines physical Hot/Cold lakehouse seam bindings.
type TieredStoragePair struct {
	HotDrivingNodeID        string `json:"hot_driving_node_id"`
	HotTableName            string `json:"hot_table_name"` // e.g. starrocks.trades_realtime
	ColdDrivingNodeID       string `json:"cold_driving_node_id"`
	ColdTableName           string `json:"cold_table_name"` // e.g. iceberg.trades_historical
	TemporalWatermarkColumn string `json:"temporal_watermark_column"` // e.g. trade_date
	WatermarkCutoff         string `json:"watermark_cutoff"` // e.g. 2026-01-01
}

// FieldSpec represents a field definition in a Business Object.
type FieldSpec struct {
	Name                    string          `json:"name"`
	DataType                string          `json:"data_type"`
	Role                    string          `json:"role"` // DIMENSION, MEASURE, ATTRIBUTE
	ASTExpression           string          `json:"ast_expression"`
	AdditivityScope         AdditivityScope `json:"additivity_scope"`
	TemporalAggregationRule string          `json:"temporal_aggregation_rule"`
	BindingStatus           FieldBindingStatus `json:"binding_status"`
	SourceTable             string          `json:"source_table"`
	SourceColumn            string          `json:"source_column"`
}

// JoinSpec defines relation metadata and grain cardinality.
type JoinSpec struct {
	ParentTable string          `json:"parent_table"`
	ParentKey   string          `json:"parent_key"`
	ChildTable  string          `json:"child_table"`
	ChildKey    string          `json:"child_key"`
	Cardinality JoinCardinality `json:"cardinality"`
	Measures    []string        `json:"measures"`
}

// QueryPlanSpec defines complete inputs for the resilient SQL compiler.
type QueryPlanSpec struct {
	TenantID            string             `json:"tenant_id"`
	DrivingTable        string             `json:"driving_table"`
	BaseFilterPredicate map[string]interface{} `json:"base_filter_predicate"`
	Fields              []FieldSpec        `json:"fields"`
	Joins               []JoinSpec         `json:"joins"`
	TieredStorage       *TieredStoragePair `json:"tiered_storage,omitempty"`
	TwoStageCTEEnabled  bool               `json:"two_stage_cte_enabled"`
}

// BOResilienceEngine provides enterprise-grade validation, query compilation, and drift invalidation.
type BOResilienceEngine struct {
	db *sqlx.DB
}

// NewBOResilienceEngine creates a new resilience engine instance.
func NewBOResilienceEngine(db *sqlx.DB) *BOResilienceEngine {
	return &BOResilienceEngine{db: db}
}

// ─────────────────────────────────────────────
// 1. Circular Calculation & Tarjan SCC Safeguard
// ─────────────────────────────────────────────

// DetectCircularCalculations detects circular dependencies in field calculation formulas.
// Returns an error with the complete cycle path if a circular reference exists.
func (e *BOResilienceEngine) DetectCircularCalculations(dependencies map[string][]string) ([]string, error) {
	index := 0
	stack := make([]string, 0)
	inStack := make(map[string]bool)
	indices := make(map[string]int)
	lowlink := make(map[string]int)
	var detectedCycle []string

	var strongConnect func(node string) bool
	strongConnect = func(node string) bool {
		indices[node] = index
		lowlink[node] = index
		index++
		stack = append(stack, node)
		inStack[node] = true

		for _, neighbor := range dependencies[node] {
			if _, visited := indices[neighbor]; !visited {
				if strongConnect(neighbor) {
					return true
				}
				if lowlink[neighbor] < lowlink[node] {
					lowlink[node] = lowlink[neighbor]
				}
			} else if inStack[neighbor] {
				if indices[neighbor] < lowlink[node] {
					lowlink[node] = indices[neighbor]
				}
			}
		}

		if lowlink[node] == indices[node] {
			var scc []string
			for {
				w := stack[len(stack)-1]
				stack = stack[:len(stack)-1]
				inStack[w] = false
				scc = append(scc, w)
				if w == node {
					break
				}
			}
			// If SCC contains > 1 node or a self-loop, cycle detected
			if len(scc) > 1 {
				// Reverse for intuitive order: A -> B -> C -> A
				for i, j := 0, len(scc)-1; i < j; i, j = i+1, j-1 {
					scc[i], scc[j] = scc[j], scc[i]
				}
				scc = append(scc, scc[0])
				detectedCycle = scc
				return true
			}
			if len(scc) == 1 {
				for _, dep := range dependencies[node] {
					if dep == node {
						detectedCycle = []string{node, node}
						return true
					}
				}
			}
		}
		return false
	}

	for node := range dependencies {
		if _, visited := indices[node]; !visited {
			if strongConnect(node) {
				cycleStr := strings.Join(detectedCycle, " -> ")
				return detectedCycle, fmt.Errorf("circular calculation dependency detected: %s", cycleStr)
			}
		}
	}

	return nil, nil
}

// ─────────────────────────────────────────────
// 2. Fan-Out & Chasm Trap Defense (Two-Stage CTE)
// 3. Tenant Discriminator & Invariant Injection (Rule 7)
// 4. Non-Additive & Semi-Additive Metric Locks
// 5. Tiered Storage (Rule 4 Hot/Cold Seam)
// ─────────────────────────────────────────────

// CompileResilientQuery builds an AST-safe, fan-out protected SQL query.
func (e *BOResilienceEngine) CompileResilientQuery(spec QueryPlanSpec) (string, error) {
	if spec.TenantID == "" {
		return "", errors.New("tenant_id invariant is required")
	}
	if spec.DrivingTable == "" && spec.TieredStorage == nil {
		return "", errors.New("driving_table or tiered_storage pair is required")
	}

	var sb strings.Builder
	cteNames := make([]string, 0)

	// Step A: Two-Stage CTE Aggregation for 1:N or M:N child relations (Fan-Out Defense)
	for _, join := range spec.Joins {
		if join.Cardinality == CardinalityOneToMany || join.Cardinality == CardinalityManyToMany {
			if spec.TwoStageCTEEnabled {
				cteName := fmt.Sprintf("layer_0_%s_agg", strings.ReplaceAll(join.ChildTable, ".", "_"))
				cteNames = append(cteNames, cteName)

				if len(cteNames) == 1 {
					sb.WriteString("WITH ")
				} else {
					sb.WriteString(", ")
				}

				// Aggregate child measures at parent grain first
				var measureAggs []string
				for _, m := range join.Measures {
					measureAggs = append(measureAggs, fmt.Sprintf("COALESCE(SUM(%s), 0) AS %s", m, m))
				}
				if len(measureAggs) == 0 {
					measureAggs = append(measureAggs, "COUNT(*) AS child_row_count")
				}

				sb.WriteString(fmt.Sprintf("%s AS (\n", cteName))
				sb.WriteString(fmt.Sprintf("    SELECT %s,\n           %s\n", join.ChildKey, strings.Join(measureAggs, ",\n           ")))
				sb.WriteString(fmt.Sprintf("    FROM %s\n", join.ChildTable))
				sb.WriteString(fmt.Sprintf("    WHERE tenant_id = '%s' AND is_deleted = FALSE\n", spec.TenantID))
				sb.WriteString(fmt.Sprintf("    GROUP BY %s\n)\n", join.ChildKey))
			}
		}
	}

	// Step B: Tiered Storage Hot/Cold Seam Expansion (Rule 4)
	drivingSource := spec.DrivingTable
	if spec.TieredStorage != nil && spec.TieredStorage.HotTableName != "" && spec.TieredStorage.ColdTableName != "" {
		seamCTE := "tiered_storage_seam"
		if len(cteNames) == 0 {
			sb.WriteString("WITH ")
		} else {
			sb.WriteString(", ")
		}
		sb.WriteString(fmt.Sprintf("%s AS (\n", seamCTE))
		sb.WriteString(fmt.Sprintf("    -- Hot StarRocks Tier (Real-Time >= Watermark)\n"))
		sb.WriteString(fmt.Sprintf("    SELECT *\n    FROM %s\n    WHERE %s >= '%s'\n",
			spec.TieredStorage.HotTableName, spec.TieredStorage.TemporalWatermarkColumn, spec.TieredStorage.WatermarkCutoff))
		sb.WriteString("    UNION ALL\n")
		sb.WriteString(fmt.Sprintf("    -- Cold Iceberg Tier (Historical < Watermark)\n"))
		sb.WriteString(fmt.Sprintf("    SELECT *\n    FROM %s\n    WHERE %s < '%s'\n",
			spec.TieredStorage.ColdTableName, spec.TieredStorage.TemporalWatermarkColumn, spec.TieredStorage.WatermarkCutoff))
		sb.WriteString(")\n")
		drivingSource = seamCTE
	}

	// Step C: Compile Select Projections & Metric Additivity Rules
	var selectExpressions []string
	for _, field := range spec.Fields {
		expr := field.Name
		if field.SourceColumn != "" {
			expr = field.SourceColumn
		}
		if field.ASTExpression != "" {
			expr = field.ASTExpression
		}

		// Enforce Additivity Locks
		switch field.AdditivityScope {
		case AdditivitySemiAdditiveAcrossTime:
			// Disallow naive SUM; enforce temporal snapshot window
			if field.TemporalAggregationRule != "" {
				expr = field.TemporalAggregationRule
			} else {
				expr = fmt.Sprintf("LAST_VALUE(%s) OVER (PARTITION BY %s_id ORDER BY as_of_date ROWS BETWEEN UNBOUNDED PRECEDING AND CURRENT ROW)", expr, spec.DrivingTable)
			}
		case AdditivityNonAdditive:
			// Must use AVG / Weighted / Last value
			if strings.HasPrefix(strings.ToUpper(expr), "SUM(") {
				expr = strings.Replace(expr, "SUM(", "AVG(", 1)
			}
		}

		selectExpressions = append(selectExpressions, fmt.Sprintf("    %s AS %s", expr, field.Name))
	}

	if len(selectExpressions) == 0 {
		selectExpressions = append(selectExpressions, "    *")
	}

	sb.WriteString("SELECT\n")
	sb.WriteString(strings.Join(selectExpressions, ",\n"))
	sb.WriteString(fmt.Sprintf("\nFROM %s base\n", drivingSource))

	// Step D: Join CTEs and Parent Tables
	for _, join := range spec.Joins {
		if (join.Cardinality == CardinalityOneToMany || join.Cardinality == CardinalityManyToMany) && spec.TwoStageCTEEnabled {
			cteName := fmt.Sprintf("layer_0_%s_agg", strings.ReplaceAll(join.ChildTable, ".", "_"))
			sb.WriteString(fmt.Sprintf("LEFT JOIN %s ON base.%s = %s.%s\n", cteName, join.ParentKey, cteName, join.ChildKey))
		} else {
			sb.WriteString(fmt.Sprintf("LEFT JOIN %s ON base.%s = %s.%s\n", join.ChildTable, join.ParentKey, join.ChildTable, join.ChildKey))
		}
	}

	// Step E: Mandatory Tenant & Soft-Delete Invariant Injection (Rule 7)
	sb.WriteString("WHERE base.tenant_id = '")
	sb.WriteString(spec.TenantID)
	sb.WriteString("' AND base.is_deleted = FALSE")

	// Custom base filter predicate injection
	if len(spec.BaseFilterPredicate) > 0 {
		for k, v := range spec.BaseFilterPredicate {
			switch val := v.(type) {
			case string:
				sb.WriteString(fmt.Sprintf(" AND base.%s = '%s'", k, val))
			case bool:
				sb.WriteString(fmt.Sprintf(" AND base.%s = %t", k, val))
			default:
				sb.WriteString(fmt.Sprintf(" AND base.%s = %v", k, val))
			}
		}
	}

	return sb.String(), nil
}

// ─────────────────────────────────────────────
// 6. Continuous Drift Invalidation Hook
// ─────────────────────────────────────────────

// HandleSchemaDrift updates affected fields to DRIFT_DEGRADED when underlying physical columns are dropped/altered.
func (e *BOResilienceEngine) HandleSchemaDrift(ctx context.Context, tenantID, datasourceID uuid.UUID, tableName, columnName string, driftType string) (int64, error) {
	if e.db == nil {
		return 0, nil
	}

	driftDetails, _ := json.Marshal(map[string]interface{}{
		"drift_type":     driftType,
		"table_name":     tableName,
		"column_name":    columnName,
		"detected_at":    time.Now().UTC().Format(time.RFC3339),
		"datasource_id":  datasourceID.String(),
		"action_required": "Re-map or archive degraded field",
	})

	query := `
		UPDATE public.bo_fields
		SET binding_status = 'DRIFT_DEGRADED',
		    drift_detected_at = NOW(),
		    drift_details = $1
		WHERE tenant_id = $2
		  AND (source_column = $3 OR technical_name = $3)
	`
	res, err := e.db.ExecContext(ctx, query, driftDetails, tenantID, columnName)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

// ─────────────────────────────────────────────
// 7. Versioning & Maker-Checker State Machine
// ─────────────────────────────────────────────

// TransitionLifecycle validates Maker-Checker state transitions.
func (e *BOResilienceEngine) TransitionLifecycle(current BOLifecycleStatus, action string, isChecker bool) (BOLifecycleStatus, int, error) {
	switch current {
	case StatusDraft:
		if action == "SUBMIT_FOR_APPROVAL" {
			return StatusPendingApproval, 1, nil
		}
		return StatusDraft, 1, fmt.Errorf("invalid action '%s' for DRAFT state", action)

	case StatusPendingApproval:
		if action == "APPROVE" {
			if !isChecker {
				return StatusPendingApproval, 1, errors.New("maker cannot approve their own submission (maker-checker rule)")
			}
			return StatusPublished, 1, nil
		}
		if action == "REJECT" {
			return StatusDraft, 1, nil
		}
		return StatusPendingApproval, 1, fmt.Errorf("invalid action '%s' for PENDING_APPROVAL state", action)

	case StatusPublished:
		if action == "EDIT" || action == "DRAFT_NEW_VERSION" {
			return StatusDraft, 2, nil
		}
		if action == "DEPRECATE" {
			return StatusDeprecated, 1, nil
		}
		return StatusPublished, 1, fmt.Errorf("invalid action '%s' for PUBLISHED state", action)

	case StatusDeprecated:
		return StatusDeprecated, 1, errors.New("cannot transition out of DEPRECATED state")

	default:
		return StatusDraft, 1, nil
	}
}
