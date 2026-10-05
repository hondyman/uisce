package analytics

import (
	"context"
	"encoding/json"
	"log"
	"math/rand"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/hondyman/uisce/backend/internal/models"
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
	return s.MarkMaterializingAttempt(ctx, id, "")
}

// MarkMaterializingAttempt starts a deploy/refresh attempt. attemptID tags the
// in-flight write for dual-commit reconcile (CUBE-1.1 / 1.3). Empty attemptID
// keeps legacy callers working.
func (s *PreAggLifecycleService) MarkMaterializingAttempt(ctx context.Context, id uuid.UUID, attemptID string) error {
	return s.updateProps(ctx, id, func(p *models.PreAggProperties) {
		p.LifecycleStatus = models.LifecycleMaterializing
		now := time.Now().UTC()
		p.LastMaterializedAt = &now
		p.LastRefreshStatus = ""
		p.LastRefreshError = ""
		if strings.TrimSpace(attemptID) != "" {
			p.AttemptID = strings.TrimSpace(attemptID)
		}
	})
}

func (s *PreAggLifecycleService) MarkActive(ctx context.Context, id uuid.UUID, stats *models.PreAggStats) error {
	return s.MarkActiveAttempt(ctx, id, "", stats)
}

// MarkActiveAttempt completes a successful refresh. LastRefreshedAt is the
// freshness clock (refresh completion time), compared at query time by
// CubeRouter / EvaluateMVWatermarkStaleness.
func (s *PreAggLifecycleService) MarkActiveAttempt(ctx context.Context, id uuid.UUID, attemptID string, stats *models.PreAggStats) error {
	return s.updateProps(ctx, id, func(p *models.PreAggProperties) {
		p.LifecycleStatus = models.LifecycleActive
		now := time.Now().UTC()
		p.LastRefreshedAt = &now
		p.LastRefreshStatus = "success"
		p.LastRefreshError = ""
		if strings.TrimSpace(attemptID) != "" {
			p.AttemptID = strings.TrimSpace(attemptID)
		}
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
	return s.MarkFailedAttempt(ctx, id, "", err)
}

// MarkFailedAttempt records a failed deploy/refresh for the given attempt.
func (s *PreAggLifecycleService) MarkFailedAttempt(ctx context.Context, id uuid.UUID, attemptID string, err error) error {
	return s.updateProps(ctx, id, func(p *models.PreAggProperties) {
		p.LifecycleStatus = models.LifecycleFailed
		p.LastRefreshStatus = "failed"
		if strings.TrimSpace(attemptID) != "" {
			p.AttemptID = strings.TrimSpace(attemptID)
		}
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

// SchedulerMode controls how PreAggScheduler.Tick treats a due node.
type SchedulerMode string

const (
	// SchedulerModeShadow evaluates due-ness and logs what it would do, but
	// performs no refresh and no lifecycle transition. This is the default so
	// that activating the scheduler cannot change behavior for tenants that
	// never opted into the feature.
	SchedulerModeShadow SchedulerMode = "shadow"

	// SchedulerModeEnabled performs the refresh. Opt-in via
	// PREAGG_SCHEDULER_ENABLED=true.
	SchedulerModeEnabled SchedulerMode = "enabled"
)

// TickReport summarizes one scheduling cycle. It exists so callers (and tests)
// can observe what a tick decided without inferring it from logs.
type TickReport struct {
	Mode         SchedulerMode
	Scanned      int
	SkippedManual int
	NotDue       int
	// Refreshed holds node IDs that were actually refreshed (enabled mode only).
	Refreshed []uuid.UUID
	// WouldRefresh holds node IDs that are due and eligible. In shadow mode
	// this is the entire set of actions taken; in enabled mode it is the same
	// set as Refreshed.
	WouldRefresh []uuid.UUID
	// Failed holds node IDs whose refresh failed.
	Failed []uuid.UUID
	// ParseErrors counts nodes whose properties could not be parsed.
	ParseErrors int
}

// PreAggScheduler handles scheduled refresh of pre-aggregations.
type PreAggScheduler struct {
	db        *sqlx.DB
	lifecycle *PreAggLifecycleService
	preAggSvc *PreAggregationService
	mode      SchedulerMode
	// onFailure, when set, is invoked whenever a refresh transitions a node to
	// failed. A scheduler that fails silently turns staleness from an edge case
	// into the steady state, so failures must be surfaced.
	onFailure func(nodeID uuid.UUID, err error)
	// now is injectable for deterministic tests; defaults to time.Now.
	now func() time.Time
}

func NewPreAggScheduler(db *sqlx.DB, lifecycle *PreAggLifecycleService, preAggSvc *PreAggregationService) *PreAggScheduler {
	return &PreAggScheduler{
		db:        db,
		lifecycle: lifecycle,
		preAggSvc: preAggSvc,
		mode:      SchedulerModeShadow,
		now:       func() time.Time { return time.Now().UTC() },
	}
}

// SetMode selects shadow or enabled behavior. Unknown values are coerced to
// shadow: an unparseable configuration must never silently start refreshing.
func (s *PreAggScheduler) SetMode(m SchedulerMode) {
	if m != SchedulerModeEnabled {
		s.mode = SchedulerModeShadow
		return
	}
	s.mode = SchedulerModeEnabled
}

// Mode reports the current scheduling mode.
func (s *PreAggScheduler) Mode() SchedulerMode { return s.mode }

// SchedulerModeFromEnv maps the PREAGG_SCHEDULER_ENABLED environment value to
// a mode. Only the exact string "true" enables, matching the existing
// CBO_ENABLED feature-flag precedent. Anything else — including a typo such as
// "TRUE" or "1" — resolves to shadow, because a misconfigured flag must never
// silently start refreshing production materializations.
func SchedulerModeFromEnv(value string) SchedulerMode {
	if strings.TrimSpace(value) == "true" {
		return SchedulerModeEnabled
	}
	return SchedulerModeShadow
}

// Enabled reports whether refreshes will actually be performed.
func (s *PreAggScheduler) Enabled() bool { return s.mode == SchedulerModeEnabled }

// SetFailureHook installs the callback invoked on refresh failure.
func (s *PreAggScheduler) SetFailureHook(fn func(nodeID uuid.UUID, err error)) {
	s.onFailure = fn
}

// SetClock overrides the time source (tests).
func (s *PreAggScheduler) SetClock(now func() time.Time) {
	if now != nil {
		s.now = now
	}
}

// Tick runs one scheduling cycle, refreshing due pre-aggregations.
//
// In shadow mode it is strictly read-only: it reports what it would do and
// returns without mutating any node. That property is what makes this safe to
// turn on globally (test #14).
func (s *PreAggScheduler) Tick(ctx context.Context) (*TickReport, error) {
	now := s.now()
	report := &TickReport{Mode: s.mode}

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
		return report, err
	}
	report.Scanned = len(nodes)

	for _, n := range nodes {
		props, err := models.ParsePreAggProperties(n.Properties)
		if err != nil {
			report.ParseErrors++
			continue
		}

		// Skip manual refresh. This is the per-node kill switch: an operator
		// can freeze an individual pre-agg (or cube) regardless of the global
		// mode.
		if props.RefreshStrategy == "manual" {
			report.SkippedManual++
			continue
		}

		// Skip if not due
		if !s.isDueForRefresh(props, now) {
			report.NotDue++
			continue
		}

		// Shadow mode: report the intended action and change nothing. This is
		// the operational evidence that the catalog scan and due-logic work
		// against real data before any refresh is permitted.
		if s.mode != SchedulerModeEnabled {
			report.WouldRefresh = append(report.WouldRefresh, n.ID)
			log.Printf("[PreAggScheduler] shadow: would refresh pre-agg %s (strategy=%q interval_min=%d lifecycle=%q)",
				n.ID, props.RefreshStrategy, props.RefreshIntervalMinutes, props.LifecycleStatus)
			continue
		}

		// Refresh
		report.WouldRefresh = append(report.WouldRefresh, n.ID)
		if err := s.lifecycle.MarkRefreshing(ctx, n.ID); err != nil {
			log.Printf("[PreAggScheduler] mark refreshing %s: %v", n.ID, err)
		}
		err = s.preAggSvc.Refresh(ctx, n.ID)
		if err != nil {
			report.Failed = append(report.Failed, n.ID)
			if markErr := s.lifecycle.MarkFailed(ctx, n.ID, err); markErr != nil {
				log.Printf("[PreAggScheduler] mark failed %s: %v", n.ID, markErr)
			}
			// Surface the failure. Recording it in catalog properties alone
			// lets a broken materialization degrade silently.
			log.Printf("[PreAggScheduler] refresh FAILED for pre-agg %s: %v", n.ID, err)
			if s.onFailure != nil {
				s.onFailure(n.ID, err)
			}
			continue
		}
		report.Refreshed = append(report.Refreshed, n.ID)

		// Mark active (stats would come from StarRocks in production)
		var stats *models.PreAggStats = nil // TODO: Fetch from StarRocks
		if err := s.lifecycle.MarkActive(ctx, n.ID, stats); err != nil {
			log.Printf("[PreAggScheduler] mark active %s: %v", n.ID, err)
		}

		// Schedule next refresh
		if props.RefreshIntervalMinutes > 0 {
			next := now.Add(time.Duration(props.RefreshIntervalMinutes) * time.Minute)
			if err := s.lifecycle.UpdateNextScheduledRefresh(ctx, n.ID, next); err != nil {
				log.Printf("[PreAggScheduler] schedule next %s: %v", n.ID, err)
			}
		}
	}

	return report, nil
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

// jitteredInterval applies ±jitterPercent to base. A flat ticker makes every
// cell scan and refresh on the same instant, which turns a long history into a
// thundering herd against StarRocks. Returns base unchanged for non-positive
// input or zero jitter.
func jitteredInterval(base time.Duration, jitterPercent float64, randSrc func() float64) time.Duration {
	if base <= 0 || jitterPercent <= 0 {
		return base
	}
	if randSrc == nil {
		randSrc = rand.Float64
	}
	// rand.Float64 ∈ [0,1) → factor ∈ [1-jitter, 1+jitter)
	factor := 1 + jitterPercent*(2*randSrc()-1)
	jittered := time.Duration(float64(base) * factor)
	if jittered < time.Millisecond {
		return time.Millisecond
	}
	return jittered
}

// DefaultPreAggTickInterval is the configured cadence at which the scheduler
// scans for due materializations. Intervals below this are not expressible:
// anything faster belongs in the router/staleness layer, the same reasoning as
// the tile refreshInterval floor.
const DefaultPreAggTickInterval = 60 * time.Second

// DefaultPreAggTickJitterPercent is ±20% applied to the tick interval.
const DefaultPreAggTickJitterPercent = 0.20

// Start begins the scheduler loop (blocking). Each cycle is jittered to avoid
// cross-cell alignment. Respects SchedulerMode: in shadow mode it only logs.
func (s *PreAggScheduler) Start(ctx context.Context, interval time.Duration) {
	if interval <= 0 {
		interval = DefaultPreAggTickInterval
	}

	for {
		wait := jitteredInterval(interval, DefaultPreAggTickJitterPercent, nil)
		timer := time.NewTimer(wait)
		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case <-timer.C:
			report, err := s.Tick(ctx)
			if err != nil {
				log.Printf("[PreAggScheduler] tick error: %v", err)
				continue
			}
			if s.mode == SchedulerModeEnabled {
				log.Printf("[PreAggScheduler] tick: scanned=%d refreshed=%d failed=%d skipped_manual=%d not_due=%d",
					report.Scanned, len(report.Refreshed), len(report.Failed), report.SkippedManual, report.NotDue)
			}
		}
	}
}

