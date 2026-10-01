package scoring

import (
	"context"
	"database/sql"
	"log"
	"math"
	"sync/atomic"
	"time"
)

// MartFreshnessStatus reports the timestamp and lag of the StarRocks scoring mart.
type MartFreshnessStatus struct {
	LastAsOfDate      string  `json:"last_as_of_date"`
	HoursSinceRefresh float64 `json:"hours_since_refresh"`
	IsStale           bool    `json:"is_stale"` // true if > 25.0 hours
	Status            string  `json:"status"`   // "FRESH", "STALE", "NOT_CONFIGURED"
}

// FeedGapAlert indicates a missing feed log past its SLA cutoff deadline.
type FeedGapAlert struct {
	VendorID     string  `json:"vendor_id"`
	EntityDomain string  `json:"entity_domain"`
	FeedName     string  `json:"feed_name"`
	HoursOverdue float64 `json:"hours_overdue"`
	Severity     string  `json:"severity"` // "WARNING", "CRITICAL"
}

// ProfileDriftAlert reports discrepancies between the active governance profile and StarRocks mart syncs.
type ProfileDriftAlert struct {
	ActiveProfileID     int64  `json:"active_profile_id"`
	ActiveProfileName   string `json:"active_profile_name"`
	MartScoredProfileID int64  `json:"mart_scored_profile_id"`
	HasDrift            bool   `json:"has_drift"`
	Message             string `json:"message,omitempty"`
}

// ScoringPipelineHealth aggregates freshness, ingestion gap alerts, and solver telemetry.
type ScoringPipelineHealth struct {
	Status             string              `json:"status"` // "HEALTHY", "DEGRADED", "CRITICAL"
	MartFreshness      MartFreshnessStatus `json:"mart_freshness"`
	IngestionGapAlerts []FeedGapAlert      `json:"ingestion_gap_alerts"`
	ProfileDrift       ProfileDriftAlert   `json:"profile_drift"`
	ShadowStability    string              `json:"shadow_stability"` // "STABLE", "DRIFT_DETECTED", "UNMONITORED"
	PartialSolverCount int64               `json:"partial_solver_count"`
	CheckedAt          time.Time           `json:"checked_at"`
}

// Monitor tracks pipeline freshness, feed gaps, and solver fallback events.
type Monitor struct {
	db                 *sql.DB
	starrocksDB        *sql.DB
	partialSolverCount atomic.Int64
}

// NewMonitor creates a pipeline health and alert monitor.
func NewMonitor(db *sql.DB, starrocksDB *sql.DB) *Monitor {
	return &Monitor{
		db:          db,
		starrocksDB: starrocksDB,
	}
}

// RecordPartialSolver increments the solver fallback metric counter.
func (m *Monitor) RecordPartialSolver() {
	cnt := m.partialSolverCount.Add(1)
	log.Printf("[MDM_SCORING_ALERT] Solver partial timeout fallback triggered (cumulative count: %d)", cnt)
}

// CheckHealth queries StarRocks mart freshness and PostgreSQL feed arrival logs.
func (m *Monitor) CheckHealth(ctx context.Context) ScoringPipelineHealth {
	now := time.Now()
	health := ScoringPipelineHealth{
		Status:             "HEALTHY",
		CheckedAt:          now,
		PartialSolverCount: m.partialSolverCount.Load(),
		IngestionGapAlerts: []FeedGapAlert{},
		MartFreshness: MartFreshnessStatus{
			LastAsOfDate:      now.Format("2006-01-02"),
			HoursSinceRefresh: 0.0,
			IsStale:           false,
			Status:            "FRESH",
		},
	}

	// 1. StarRocks Mart Freshness Check (> 25h old is an alert)
	if m.starrocksDB != nil {
		query := `SELECT MAX(as_of_date) FROM mdm_analytics.vendor_substitution_daily`
		var maxDateStr sql.NullString
		err := m.starrocksDB.QueryRowContext(ctx, query).Scan(&maxDateStr)
		if err == nil && maxDateStr.Valid && maxDateStr.String != "" {
			if parsed, pErr := time.Parse("2006-01-02", maxDateStr.String); pErr == nil {
				hoursSince := now.Sub(parsed).Hours()
				health.MartFreshness.LastAsOfDate = maxDateStr.String
				health.MartFreshness.HoursSinceRefresh = hoursSince
				if hoursSince > 25.0 {
					health.MartFreshness.IsStale = true
					health.MartFreshness.Status = "STALE"
					health.Status = "DEGRADED"
					log.Printf("[MDM_SCORING_ALERT] StarRocks scoring mart is stale: last refreshed %s (%.1f hours ago)", maxDateStr.String, hoursSince)
				}
			}
		}
	}

	// 2. Feed Ingestion Gap Check (Feeds > 4 hours overdue past SLA cutoff)
	if m.db != nil {
		query := `
			SELECT vendor_id, entity_domain, feed_name,
			       EXTRACT(EPOCH FROM (NOW() - sla_cutoff_ts)) / 3600.0 AS hours_overdue
			FROM mdm_eval.vendor_feed_log
			WHERE sla_cutoff_ts < NOW() - INTERVAL '4 hours'
			  AND sla_breached = TRUE
			ORDER BY hours_overdue DESC
			LIMIT 10
		`
		rows, err := m.db.QueryContext(ctx, query)
		if err == nil {
			defer rows.Close()
			for rows.Next() {
				var gap FeedGapAlert
				if scanErr := rows.Scan(&gap.VendorID, &gap.EntityDomain, &gap.FeedName, &gap.HoursOverdue); scanErr == nil {
					gap.Severity = "WARNING"
					if gap.HoursOverdue > 12.0 {
						gap.Severity = "CRITICAL"
						health.Status = "CRITICAL"
					} else if health.Status == "HEALTHY" {
						health.Status = "DEGRADED"
					}
					health.IngestionGapAlerts = append(health.IngestionGapAlerts, gap)
				}
			}
		}

		// 3. Weight Profile Drift Check (Active PostgreSQL profile vs StarRocks last_scored_profile_id)
		var activeID int64
		var activeName string
		errProf := m.db.QueryRowContext(ctx, `SELECT profile_id, profile_name FROM mdm_eval.scoring_weight_profile WHERE is_active = TRUE LIMIT 1`).Scan(&activeID, &activeName)
		if errProf == nil && activeID > 0 {
			health.ProfileDrift.ActiveProfileID = activeID
			health.ProfileDrift.ActiveProfileName = activeName

			if m.starrocksDB != nil {
				var srProfileID sql.NullInt64
				srErr := m.starrocksDB.QueryRowContext(ctx, `SELECT MAX(weight_profile_id) FROM mdm_analytics.vendor_scorecard_multi_dimensional`).Scan(&srProfileID)
				if srErr == nil && srProfileID.Valid && srProfileID.Int64 > 0 {
					health.ProfileDrift.MartScoredProfileID = srProfileID.Int64
					if srProfileID.Int64 != activeID {
						health.ProfileDrift.HasDrift = true
						health.ProfileDrift.Message = "Active PostgreSQL weight profile differs from StarRocks scored profile. Run SyncMart to align."
						if health.Status == "HEALTHY" {
							health.Status = "DEGRADED"
						}
						log.Printf("[MDM_SCORING_ALERT] Weight profile drift: active profile %d (%s) vs mart scored profile %d", activeID, activeName, srProfileID.Int64)
					}
				}
			}
		}

		// 4. Shadow Validation Stability Check
		health.ShadowStability = "STABLE"
		var lastStatus sql.NullString
		var t1Delta, t2Delta, t3Delta sql.NullFloat64
		errShadow := m.db.QueryRowContext(ctx, `
			SELECT status, t1_concordance_delta, t2_concordance_delta, t3_concordance_delta 
			FROM mdm_eval.shadow_run_log 
			ORDER BY executed_at DESC LIMIT 1
		`).Scan(&lastStatus, &t1Delta, &t2Delta, &t3Delta)
		if errShadow == nil && lastStatus.Valid {
			if lastStatus.String == "DRIFT_DETECTED" || (t1Delta.Valid && math.Abs(t1Delta.Float64) > 1.0) {
				health.ShadowStability = "DRIFT_DETECTED"
				if health.Status == "HEALTHY" {
					health.Status = "DEGRADED"
				}
				log.Printf("[MDM_SCORING_ALERT] Shadow validation stability drift detected on recent run")
			}
		}
	} else {
		health.ShadowStability = "STABLE"
	}

	if health.PartialSolverCount > 0 && health.Status == "HEALTHY" {
		health.Status = "DEGRADED"
	}

	return health
}
