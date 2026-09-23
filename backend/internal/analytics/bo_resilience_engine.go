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

// BOLifecycleStatus defines the Maker-Checker governance state.
type BOLifecycleStatus string

const (
	StatusDraft           BOLifecycleStatus = "DRAFT"
	StatusPendingApproval BOLifecycleStatus = "PENDING_APPROVAL"
	StatusPublished       BOLifecycleStatus = "PUBLISHED"
	StatusDeprecated      BOLifecycleStatus = "DEPRECATED"
)

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
