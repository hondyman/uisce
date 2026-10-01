package querybuilder

import (
	"context"
	"fmt"
	"strings"

	"github.com/jmoiron/sqlx"
)

// MetricCatalogReconciler synchronizes metric definitions with the catalog graph on startup or reconciliation pass.
type MetricCatalogReconciler struct {
	db *sqlx.DB
}

// NewMetricCatalogReconciler creates a new reconciler.
func NewMetricCatalogReconciler(db *sqlx.DB) *MetricCatalogReconciler {
	return &MetricCatalogReconciler{db: db}
}

// ReconcileAll reconciles all active metrics across tenants into catalog_node and catalog_edge.
// Safe for concurrent runs across multiple instance replicas using advisory locking / upsert semantics.
func (r *MetricCatalogReconciler) ReconcileAll(ctx context.Context) (int, error) {
	if r.db == nil {
		return 0, nil
	}

	// 1. Acquire transaction
	tx, err := r.db.Beginx()
	if err != nil {
		return 0, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// 2. Fetch all active metrics
	var rows []metricDefRow
	query := `SELECT id, tenant_id, name, description, bo_id, catalog_term_id, expression,
	                 grain_allowlist, format_config, variables, materialization_config,
	                 decomposable, content_hash, tags, is_core, status, archived_at,
	                 created_by, created_at, updated_at
	          FROM data_explorer.metric_definition
	          WHERE status = 'active' AND archived_at IS NULL`

	if err := tx.SelectContext(ctx, &rows, query); err != nil {
		return 0, fmt.Errorf("failed to fetch active metrics: %w", err)
	}

	reconciledCount := 0
	for _, row := range rows {
		m := row.toMetricDefinition()
		nodeID, err := SyncMetricToCatalogGraph(ctx, tx, m.TenantID, m)
		if err != nil {
			return reconciledCount, fmt.Errorf("failed reconciling metric %s (%s): %w", m.ID, m.Name, err)
		}

		// Backfill catalog_term_id if not present
		if m.CatalogTermID == nil || *m.CatalogTermID == "" {
			_, err = tx.ExecContext(ctx, `
				UPDATE data_explorer.metric_definition
				SET catalog_term_id = $1, updated_at = NOW()
				WHERE id = $2 AND tenant_id = $3
			`, nodeID, m.ID, m.TenantID)
			if err != nil {
				return reconciledCount, fmt.Errorf("failed backfilling catalog_term_id on metric %s: %w", m.ID, err)
			}
		}
		reconciledCount++
	}

	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("failed to commit reconciliation transaction: %w", err)
	}

	return reconciledCount, nil
}

// ValidateMetricAdoptionPreflight checks if all underlying termNodeIds referenced in a core metric
// resolve to mapped terms in the client tenant. Returns missing term report on error.
func ValidateMetricAdoptionPreflight(m MetricDefinition, clientAvailableTermIDs map[string]bool) error {
	var missingTerms []string
	if m.Expression.TermNodeID != "" {
		if !clientAvailableTermIDs[m.Expression.TermNodeID] {
			missingTerms = append(missingTerms, m.Expression.TermNodeID)
		}
	}

	if len(missingTerms) > 0 {
		return fmt.Errorf("adoption preflight failed: core metric %q references terms [%s] not mapped in target tenant",
			m.Name, strings.Join(missingTerms, ", "))
	}
	return nil
}
