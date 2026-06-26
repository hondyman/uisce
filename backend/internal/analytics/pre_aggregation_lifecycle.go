package analytics

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/hondyman/semlayer/backend/internal/models"
	"github.com/jmoiron/sqlx"
)

// PreAggLifecycleService manages state transitions for pre-aggregations.
type PreAggLifecycleService struct {
	db *sqlx.DB
}

func NewPreAggLifecycleService(db *sqlx.DB) *PreAggLifecycleService {
	return &PreAggLifecycleService{db: db}
}

func (s *PreAggLifecycleService) updateProps(ctx context.Context, id uuid.UUID, fn func(*models.PreAggProperties)) error {
	var node struct {
		Properties json.RawMessage `db:"properties"`
	}
	err := s.db.GetContext(ctx, &node, `SELECT properties FROM catalog_node WHERE id = $1`, id)
	if err != nil {
		return err
	}

	props, err := models.ParsePreAggProperties(node.Properties)
	if err != nil {
		return err
	}

	fn(props)

	propsJSON, err := json.Marshal(props)
	if err != nil {
		return err
	}

	_, err = s.db.ExecContext(ctx, `UPDATE catalog_node SET properties = $1, updated_at = NOW() WHERE id = $2`, propsJSON, id)
	return err
}

func (s *PreAggLifecycleService) MarkMaterializing(ctx context.Context, id uuid.UUID) error {
	return s.updateProps(ctx, id, func(p *models.PreAggProperties) {
		p.LifecycleStatus = models.LifecycleMaterializing
		now := time.Now().UTC()
		p.LastMaterializedAt = &now
		p.LastRefreshStatus = ""
		p.LastRefreshError = ""
	})
}

func (s *PreAggLifecycleService) MarkActive(ctx context.Context, id uuid.UUID, stats *models.PreAggStats) error {
	return s.updateProps(ctx, id, func(p *models.PreAggProperties) {
		p.LifecycleStatus = models.LifecycleActive
		now := time.Now().UTC()
		p.LastRefreshedAt = &now
		p.LastRefreshStatus = "success"
		p.LastRefreshError = ""
		if stats != nil {
			p.RowCount = &stats.RowCount
			p.SizeBytes = &stats.SizeBytes
		}
	})
}

func (s *PreAggLifecycleService) MarkRefreshing(ctx context.Context, id uuid.UUID) error {
	return s.updateProps(ctx, id, func(p *models.PreAggProperties) {
		p.LifecycleStatus = models.LifecycleRefreshing
	})
}

func (s *PreAggLifecycleService) MarkStale(ctx context.Context, id uuid.UUID, reason string) error {
	return s.updateProps(ctx, id, func(p *models.PreAggProperties) {
		p.LifecycleStatus = models.LifecycleStale
		p.LastRefreshStatus = "stale"
		p.LastRefreshError = reason
	})
}

func (s *PreAggLifecycleService) MarkFailed(ctx context.Context, id uuid.UUID, err error) error {
	return s.updateProps(ctx, id, func(p *models.PreAggProperties) {
		p.LifecycleStatus = models.LifecycleFailed
		p.LastRefreshStatus = "failed"
		if err != nil {
			p.LastRefreshError = err.Error()
		}
	})
}

func (s *PreAggLifecycleService) UpdateNextScheduledRefresh(ctx context.Context, id uuid.UUID, t time.Time) error {
	return s.updateProps(ctx, id, func(p *models.PreAggProperties) {
		p.NextScheduledRefresh = &t
	})
}

// SetIdle marks a pre-aggregation as idle (not yet materialized).
func (s *PreAggLifecycleService) SetIdle(ctx context.Context, id uuid.UUID) error {
	return s.updateProps(ctx, id, func(p *models.PreAggProperties) {
		p.LifecycleStatus = models.LifecycleIdle
	})
}

// --- Invalidation Service ---

// PreAggInvalidationService marks pre-aggregations stale when upstream changes occur.
type PreAggInvalidationService struct {
	db        *sqlx.DB
	lifecycle *PreAggLifecycleService
}

func NewPreAggInvalidationService(db *sqlx.DB, lifecycle *PreAggLifecycleService) *PreAggInvalidationService {
	return &PreAggInvalidationService{db: db, lifecycle: lifecycle}
}

// InvalidateByBO marks all pre-aggregations for a BO as stale.
func (s *PreAggInvalidationService) InvalidateByBO(ctx context.Context, boID uuid.UUID) error {
	// Find pre-aggs linked to this BO via PREAGG_FOR_BO edge
	var preAggIDs []uuid.UUID
	err := s.db.SelectContext(ctx, &preAggIDs, `
		SELECT e.source_node_id
		FROM catalog_edge e
		JOIN catalog_edge_type et ON e.edge_type_id = et.id
		WHERE e.target_node_id = $1 AND et.edge_type_name = 'PREAGG_FOR_BO'
	`, boID)
	if err != nil {
		return err
	}

	for _, id := range preAggIDs {
		_ = s.lifecycle.MarkStale(ctx, id, "BO changed")
	}
	return nil
}

// InvalidateByTerm marks pre-aggregations using a term as stale.
func (s *PreAggInvalidationService) InvalidateByTerm(ctx context.Context, termID uuid.UUID) error {
	var preAggIDs []uuid.UUID
	err := s.db.SelectContext(ctx, &preAggIDs, `
		SELECT e.source_node_id
		FROM catalog_edge e
		JOIN catalog_edge_type et ON e.edge_type_id = et.id
		WHERE e.target_node_id = $1 AND et.edge_type_name = 'PREAGG_USES_TERM'
	`, termID)
	if err != nil {
		return err
	}

	for _, id := range preAggIDs {
		_ = s.lifecycle.MarkStale(ctx, id, "SemanticTerm changed")
	}
	return nil
}

// InvalidateByCalculation marks pre-aggregations using a calculation as stale.
// Also handles transitive dependencies via CALC_USES_CALC edges.
func (s *PreAggInvalidationService) InvalidateByCalculation(ctx context.Context, calcID uuid.UUID) error {
	// 1. Get all dependent calcs (recursive)
	calcIDs := []uuid.UUID{calcID}
	dependentCalcs, err := s.getRecursiveDependents(ctx, calcID, "CALC_USES_CALC")
	if err == nil {
		calcIDs = append(calcIDs, dependentCalcs...)
	}

	// 2. Find pre-aggs using any of these calcs
	preAggSet := make(map[uuid.UUID]bool)
	for _, cid := range calcIDs {
		var ids []uuid.UUID
		err := s.db.SelectContext(ctx, &ids, `
			SELECT e.source_node_id
			FROM catalog_edge e
			JOIN catalog_edge_type et ON e.edge_type_id = et.id
			WHERE e.target_node_id = $1 AND et.edge_type_name = 'PREAGG_USES_CALC'
		`, cid)
		if err == nil {
			for _, id := range ids {
				preAggSet[id] = true
			}
		}
	}

	for id := range preAggSet {
		_ = s.lifecycle.MarkStale(ctx, id, "CalculationTerm changed")
	}
	return nil
}

func (s *PreAggInvalidationService) getRecursiveDependents(ctx context.Context, sourceID uuid.UUID, edgeType string) ([]uuid.UUID, error) {
	// Simple recursive CTE to find all dependents
	var ids []uuid.UUID
	query := `
		WITH RECURSIVE deps AS (
			SELECT e.source_node_id
			FROM catalog_edge e
			JOIN catalog_edge_type et ON e.edge_type_id = et.id
			WHERE e.target_node_id = $1 AND et.edge_type_name = $2
			
			UNION
			
			SELECT e.source_node_id
			FROM catalog_edge e
			JOIN catalog_edge_type et ON e.edge_type_id = et.id
			JOIN deps d ON e.target_node_id = d.source_node_id
			WHERE et.edge_type_name = $2
		)
		SELECT source_node_id FROM deps
	`
	err := s.db.SelectContext(ctx, &ids, query, sourceID, edgeType)
	return ids, err
}

// --- Scheduler ---

// PreAggScheduler handles scheduled refresh of pre-aggregations.
type PreAggScheduler struct {
	db        *sqlx.DB
	lifecycle *PreAggLifecycleService
	preAggSvc *PreAggregationService
}

func NewPreAggScheduler(db *sqlx.DB, lifecycle *PreAggLifecycleService, preAggSvc *PreAggregationService) *PreAggScheduler {
	return &PreAggScheduler{db: db, lifecycle: lifecycle, preAggSvc: preAggSvc}
}

// Tick runs one scheduling cycle, refreshing due pre-aggregations.
func (s *PreAggScheduler) Tick(ctx context.Context) error {
	now := time.Now().UTC()

	// Load all pre_aggregation nodes
	var nodes []struct {
		ID         uuid.UUID       `db:"id"`
		Properties json.RawMessage `db:"properties"`
	}
	err := s.db.SelectContext(ctx, &nodes, `
		SELECT n.id, n.properties
		FROM catalog_node n
		JOIN catalog_node_type nt ON n.node_type_id = nt.id
		WHERE nt.catalog_type_name = 'pre_aggregation'
	`)
	if err != nil {
		return err
	}

	for _, n := range nodes {
		props, err := models.ParsePreAggProperties(n.Properties)
		if err != nil {
			continue
		}

		// Skip manual refresh
		if props.RefreshStrategy == "manual" {
			continue
		}

		// Skip if not due
		if !s.isDueForRefresh(props, now) {
			continue
		}

		// Refresh
		_ = s.lifecycle.MarkRefreshing(ctx, n.ID)
		err = s.preAggSvc.Refresh(ctx, n.ID)
		if err != nil {
			_ = s.lifecycle.MarkFailed(ctx, n.ID, err)
			continue
		}

		// Mark active (stats would come from StarRocks in production)
		var stats *models.PreAggStats = nil // TODO: Fetch from StarRocks
		_ = s.lifecycle.MarkActive(ctx, n.ID, stats)

		// Schedule next refresh
		if props.RefreshIntervalMinutes > 0 {
			next := now.Add(time.Duration(props.RefreshIntervalMinutes) * time.Minute)
			_ = s.lifecycle.UpdateNextScheduledRefresh(ctx, n.ID, next)
		}
	}

	return nil
}

func (s *PreAggScheduler) isDueForRefresh(p *models.PreAggProperties, now time.Time) bool {
	// Stale always needs refresh
	if p.LifecycleStatus == models.LifecycleStale {
		return true
	}

	// Check scheduled time
	if p.NextScheduledRefresh == nil {
		return true // Never scheduled = due
	}
	return !now.Before(*p.NextScheduledRefresh)
}

// Start begins the scheduler loop (blocking).
func (s *PreAggScheduler) Start(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := s.Tick(ctx); err != nil {
				fmt.Printf("PreAggScheduler tick error: %v\n", err)
			}
		}
	}
}
