package aso

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// ============================================================================
// Drift Signal Types
// ============================================================================

// DriftSignal represents a detected anomaly or drift in optimization performance
type DriftSignal struct {
	ID             uuid.UUID       `json:"id" db:"id"`
	TargetType     TargetType      `json:"target_type" db:"target_type"`
	TargetID       uuid.UUID       `json:"target_id" db:"target_id"`
	TargetName     string          `json:"target_name" db:"target_name"`
	TenantID       *uuid.UUID      `json:"tenant_id,omitempty" db:"tenant_id"`
	Env            string          `json:"env" db:"env"`
	SignalType     DriftSignalType `json:"signal_type" db:"signal_type"`
	Severity       DriftSeverity   `json:"severity" db:"severity"`
	Status         DriftStatus     `json:"status" db:"status"`
	Evidence       json.RawMessage `json:"evidence" db:"evidence"`
	Recommendation string          `json:"recommendation" db:"recommendation"`
	DetectedAt     time.Time       `json:"detected_at" db:"detected_at"`
	ResolvedAt     *time.Time      `json:"resolved_at,omitempty" db:"resolved_at"`
	ResolvedBy     *string         `json:"resolved_by,omitempty" db:"resolved_by"`
	AutoResolved   bool            `json:"auto_resolved" db:"auto_resolved"`
}

// DriftSignalType categorizes the type of drift
type DriftSignalType string

const (
	DriftSignalPatternChange     DriftSignalType = "pattern_change"     // Query patterns changed
	DriftSignalMissRateSpike     DriftSignalType = "miss_rate_spike"    // Pre-agg misses increasing
	DriftSignalLatencyRegression DriftSignalType = "latency_regression" // Latency getting worse
	DriftSignalRefreshFailure    DriftSignalType = "refresh_failure"    // Refresh failing
	DriftSignalSchemaDrift       DriftSignalType = "schema_drift"       // Underlying schema changed
	DriftSignalUsageDecline      DriftSignalType = "usage_decline"      // Usage dropping
	DriftSignalStaleData         DriftSignalType = "stale_data"         // Data freshness degrading
)

// DriftSeverity indicates urgency
type DriftSeverity string

const (
	DriftSeverityLow      DriftSeverity = "low"
	DriftSeverityMedium   DriftSeverity = "medium"
	DriftSeverityHigh     DriftSeverity = "high"
	DriftSeverityCritical DriftSeverity = "critical"
)

// DriftStatus tracks resolution
type DriftStatus string

const (
	DriftStatusOpen         DriftStatus = "open"
	DriftStatusAcknowledged DriftStatus = "acknowledged"
	DriftStatusResolving    DriftStatus = "resolving"
	DriftStatusResolved     DriftStatus = "resolved"
	DriftStatusIgnored      DriftStatus = "ignored"
)

// ============================================================================
// Evidence Types per Signal
// ============================================================================

// PatternChangeEvidence shows query pattern drift
type PatternChangeEvidence struct {
	PreviousGrains  []string `json:"previous_grains"`
	CurrentGrains   []string `json:"current_grains"`
	NewMeasures     []string `json:"new_measures"`
	DroppedMeasures []string `json:"dropped_measures"`
	ChangePercent   float64  `json:"change_percent"`
}

// MissRateEvidence shows pre-agg miss rate spike
type MissRateEvidence struct {
	PreviousMissRate    float64  `json:"previous_miss_rate"` // last 7 days avg
	CurrentMissRate     float64  `json:"current_miss_rate"`  // last 24 hours
	MissIncreasePercent float64  `json:"miss_increase_percent"`
	MissedQueries       int64    `json:"missed_queries"`
	CommonMissPatterns  []string `json:"common_miss_patterns"`
}

// LatencyRegressionEvidence shows latency degradation
type LatencyRegressionEvidence struct {
	BaselineP95Ms     float64  `json:"baseline_p95_ms"` // when optimization applied
	CurrentP95Ms      float64  `json:"current_p95_ms"`  // now
	RegressionPercent float64  `json:"regression_percent"`
	PossibleCauses    []string `json:"possible_causes"`
}

// RefreshFailureEvidence shows refresh problems
type RefreshFailureEvidence struct {
	ConsecutiveFailures int       `json:"consecutive_failures"`
	LastSuccessAt       time.Time `json:"last_success_at"`
	LastError           string    `json:"last_error"`
	RetryAttempts       int       `json:"retry_attempts"`
}

// ============================================================================
// Anomaly Detection Service
// ============================================================================

// AnomalyDetectionService monitors for drift and anomalies
type AnomalyDetectionService interface {
	// ScanForAnomalies checks all optimizations for drift
	ScanForAnomalies(ctx context.Context, env string) ([]DriftSignal, error)

	// CheckOptimization checks a specific optimization for drift
	CheckOptimization(ctx context.Context, optID uuid.UUID) (*DriftSignal, error)

	// GetOpenSignals returns unresolved drift signals
	GetOpenSignals(ctx context.Context, env string, tenantID *uuid.UUID) ([]DriftSignal, error)

	// AcknowledgeSignal marks a signal as acknowledged
	AcknowledgeSignal(ctx context.Context, signalID uuid.UUID, actor string) error

	// ResolveSignal marks a signal as resolved
	ResolveSignal(ctx context.Context, signalID uuid.UUID, actor string, autoResolved bool) error

	// GetSignalHistory returns historical signals for analysis
	GetSignalHistory(ctx context.Context, targetID uuid.UUID, limit int) ([]DriftSignal, error)
}

// anomalyDetectionService implements AnomalyDetectionService
type anomalyDetectionService struct {
	db        *sqlx.DB
	optRepo   ASOOptimizationRepository
	telemetry TelemetryService
	config    *ASOConfig
}

// NewAnomalyDetectionService creates a new anomaly detection service
func NewAnomalyDetectionService(db *sqlx.DB, optRepo ASOOptimizationRepository, telemetry TelemetryService, config *ASOConfig) AnomalyDetectionService {
	if config == nil {
		config = DefaultConfig()
	}
	return &anomalyDetectionService{
		db:        db,
		optRepo:   optRepo,
		telemetry: telemetry,
		config:    config,
	}
}

// ScanForAnomalies checks all optimizations for drift
func (s *anomalyDetectionService) ScanForAnomalies(ctx context.Context, env string) ([]DriftSignal, error) {
	// Get all applied optimizations
	applied := OptStatusApplied
	filter := OptimizationFilter{Env: &env, Status: &applied}
	opts, err := s.optRepo.List(ctx, filter)
	if err != nil {
		return nil, err
	}

	var signals []DriftSignal

	for _, opt := range opts {
		signal, err := s.CheckOptimization(ctx, opt.ID)
		if err != nil {
			continue
		}
		if signal != nil {
			// Persist the signal
			s.persistSignal(ctx, signal)
			signals = append(signals, *signal)
		}
	}

	return signals, nil
}

// CheckOptimization checks a specific optimization for drift
func (s *anomalyDetectionService) CheckOptimization(ctx context.Context, optID uuid.UUID) (*DriftSignal, error) {
	opt, err := s.optRepo.GetByID(ctx, optID)
	if err != nil || opt == nil {
		return nil, err
	}

	// Only check applied optimizations
	if opt.Status != OptStatusApplied {
		return nil, nil
	}

	// Check for various types of drift
	if signal := s.checkMissRateDrift(ctx, opt); signal != nil {
		return signal, nil
	}

	if signal := s.checkLatencyRegression(ctx, opt); signal != nil {
		return signal, nil
	}

	if signal := s.checkRefreshFailures(ctx, opt); signal != nil {
		return signal, nil
	}

	if signal := s.checkUsageDecline(ctx, opt); signal != nil {
		return signal, nil
	}

	return nil, nil
}

// checkMissRateDrift detects pre-agg miss rate spikes
func (s *anomalyDetectionService) checkMissRateDrift(ctx context.Context, opt *ASOOptimization) *DriftSignal {
	// Get miss rate for last 24 hours vs last 7 days baseline
	currentMetrics, err := s.telemetry.GetMissRate(ctx, opt.TargetID, 24*time.Hour)
	if err != nil || currentMetrics == nil {
		return nil
	}

	baselineMetrics, err := s.telemetry.GetMissRate(ctx, opt.TargetID, 7*24*time.Hour)
	if err != nil || baselineMetrics == nil {
		return nil
	}

	// Check if miss rate increased significantly
	spikeThreshold := s.config.Anomaly.MissRateSpikeThreshold
	highThreshold := s.config.Anomaly.MissRateHighThreshold

	if currentMetrics.MissRate > baselineMetrics.MissRate*spikeThreshold && currentMetrics.MissRate > 0.1 {
		evidence := MissRateEvidence{
			PreviousMissRate:    baselineMetrics.MissRate,
			CurrentMissRate:     currentMetrics.MissRate,
			MissIncreasePercent: (currentMetrics.MissRate - baselineMetrics.MissRate) / baselineMetrics.MissRate * 100,
			MissedQueries:       currentMetrics.MissCount,
			CommonMissPatterns:  currentMetrics.CommonMissGrains,
		}
		evidenceJSON, _ := json.Marshal(evidence)

		severity := DriftSeverityMedium
		if currentMetrics.MissRate > highThreshold {
			severity = DriftSeverityHigh
		}

		return &DriftSignal{
			ID:             uuid.New(),
			TargetType:     opt.TargetType,
			TargetID:       opt.TargetID,
			TargetName:     opt.TargetName,
			TenantID:       opt.TenantID,
			Env:            opt.Env,
			SignalType:     DriftSignalMissRateSpike,
			Severity:       severity,
			Status:         DriftStatusOpen,
			Evidence:       evidenceJSON,
			Recommendation: "Consider updating pre-agg definition to cover new query patterns",
			DetectedAt:     time.Now(),
		}
	}

	return nil
}

// checkLatencyRegression detects latency getting worse
func (s *anomalyDetectionService) checkLatencyRegression(ctx context.Context, opt *ASOOptimization) *DriftSignal {
	// Get baseline from optimization record
	var baselineP95 float64
	if opt.P95LatencyMs != nil {
		baselineP95 = *opt.P95LatencyMs
	} else {
		return nil // No baseline to compare
	}

	// Get current latency from telemetry
	currentStats, err := s.telemetry.GetLatencyStats(ctx, opt.TargetID, 24*time.Hour)
	if err != nil || currentStats == nil {
		return nil
	}

	currentP95 := currentStats.P95LatencyMs
	regressionThreshold := s.config.Anomaly.LatencyRegressionThreshold

	// If latency regressed significantly
	if currentP95 > baselineP95*regressionThreshold && baselineP95 > 0 {
		evidence := LatencyRegressionEvidence{
			BaselineP95Ms:     baselineP95,
			CurrentP95Ms:      currentP95,
			RegressionPercent: (currentP95 - baselineP95) / baselineP95 * 100,
			PossibleCauses:    []string{"Data volume increase", "Query pattern change", "Pre-agg staleness"},
		}
		evidenceJSON, _ := json.Marshal(evidence)

		return &DriftSignal{
			ID:             uuid.New(),
			TargetType:     opt.TargetType,
			TargetID:       opt.TargetID,
			TargetName:     opt.TargetName,
			TenantID:       opt.TenantID,
			Env:            opt.Env,
			SignalType:     DriftSignalLatencyRegression,
			Severity:       DriftSeverityHigh,
			Status:         DriftStatusOpen,
			Evidence:       evidenceJSON,
			Recommendation: "Review pre-agg effectiveness and consider tuning",
			DetectedAt:     time.Now(),
		}
	}

	return nil
}

// checkRefreshFailures detects refresh problems
func (s *anomalyDetectionService) checkRefreshFailures(ctx context.Context, opt *ASOOptimization) *DriftSignal {
	// Get refresh status from telemetry
	status, err := s.telemetry.GetRefreshStatus(ctx, opt.TargetID)
	if err != nil || status == nil {
		return nil
	}

	alertThreshold := s.config.Anomaly.RefreshFailureAlertCount
	criticalThreshold := s.config.Anomaly.RefreshFailureCriticalCount

	if status.ConsecutiveFailures >= alertThreshold {
		var lastSuccess time.Time
		if status.LastSuccessAt != nil {
			lastSuccess = *status.LastSuccessAt
		}

		evidence := RefreshFailureEvidence{
			ConsecutiveFailures: status.ConsecutiveFailures,
			LastSuccessAt:       lastSuccess,
			LastError:           status.LastError,
			RetryAttempts:       status.ConsecutiveFailures,
		}
		evidenceJSON, _ := json.Marshal(evidence)

		severity := DriftSeverityMedium
		if status.ConsecutiveFailures >= criticalThreshold {
			severity = DriftSeverityCritical
		}

		return &DriftSignal{
			ID:             uuid.New(),
			TargetType:     opt.TargetType,
			TargetID:       opt.TargetID,
			TargetName:     opt.TargetName,
			TenantID:       opt.TenantID,
			Env:            opt.Env,
			SignalType:     DriftSignalRefreshFailure,
			Severity:       severity,
			Status:         DriftStatusOpen,
			Evidence:       evidenceJSON,
			Recommendation: "Investigate refresh failure and consider rebuilding pre-agg",
			DetectedAt:     time.Now(),
		}
	}

	return nil
}

// checkUsageDecline detects dropping usage
func (s *anomalyDetectionService) checkUsageDecline(ctx context.Context, opt *ASOOptimization) *DriftSignal {
	// Get usage stats from telemetry
	stats, err := s.telemetry.GetUsageStats(ctx, opt.TargetID, 30*24*time.Hour)
	if err != nil || stats == nil {
		return nil
	}

	// Check if significantly under-utilized
	if stats.IsUnderUtilized && stats.DaysSinceLastUse > 30 {
		evidence := map[string]interface{}{
			"total_queries_30d":   stats.TotalQueries,
			"trend_percent":       stats.TrendPercent,
			"days_since_last_use": stats.DaysSinceLastUse,
		}
		evidenceJSON, _ := json.Marshal(evidence)

		return &DriftSignal{
			ID:             uuid.New(),
			TargetType:     opt.TargetType,
			TargetID:       opt.TargetID,
			TargetName:     opt.TargetName,
			TenantID:       opt.TenantID,
			Env:            opt.Env,
			SignalType:     DriftSignalUsageDecline,
			Severity:       DriftSeverityLow,
			Status:         DriftStatusOpen,
			Evidence:       evidenceJSON,
			Recommendation: "Consider deprecating this pre-agg to reduce storage costs",
			DetectedAt:     time.Now(),
		}
	}

	return nil
}

// persistSignal saves a drift signal to the database
func (s *anomalyDetectionService) persistSignal(ctx context.Context, signal *DriftSignal) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO semantic.drift_signal (
			id, target_type, target_id, target_name, tenant_id, env,
			signal_type, severity, status, evidence, recommendation, detected_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12
		) ON CONFLICT (id) DO NOTHING
	`, signal.ID, signal.TargetType, signal.TargetID, signal.TargetName,
		signal.TenantID, signal.Env, signal.SignalType, signal.Severity,
		signal.Status, signal.Evidence, signal.Recommendation, signal.DetectedAt)

	return err
}

// GetOpenSignals returns unresolved drift signals
func (s *anomalyDetectionService) GetOpenSignals(ctx context.Context, env string, tenantID *uuid.UUID) ([]DriftSignal, error) {
	query := `
		SELECT * FROM semantic.drift_signal
		WHERE env = $1 AND status IN ('open', 'acknowledged', 'resolving')
	`
	args := []interface{}{env}

	if tenantID != nil {
		query += ` AND tenant_id = $2`
		args = append(args, *tenantID)
	}

	query += ` ORDER BY severity DESC, detected_at DESC`

	var signals []DriftSignal
	err := s.db.SelectContext(ctx, &signals, query, args...)
	return signals, err
}

// AcknowledgeSignal marks a signal as acknowledged
func (s *anomalyDetectionService) AcknowledgeSignal(ctx context.Context, signalID uuid.UUID, actor string) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE semantic.drift_signal
		SET status = 'acknowledged'
		WHERE id = $1
	`, signalID)
	return err
}

// ResolveSignal marks a signal as resolved
func (s *anomalyDetectionService) ResolveSignal(ctx context.Context, signalID uuid.UUID, actor string, autoResolved bool) error {
	now := time.Now()
	_, err := s.db.ExecContext(ctx, `
		UPDATE semantic.drift_signal
		SET status = 'resolved', resolved_at = $2, resolved_by = $3, auto_resolved = $4
		WHERE id = $1
	`, signalID, now, actor, autoResolved)
	return err
}

// GetSignalHistory returns historical signals for analysis
func (s *anomalyDetectionService) GetSignalHistory(ctx context.Context, targetID uuid.UUID, limit int) ([]DriftSignal, error) {
	var signals []DriftSignal
	err := s.db.SelectContext(ctx, &signals, `
		SELECT * FROM semantic.drift_signal
		WHERE target_id = $1
		ORDER BY detected_at DESC
		LIMIT $2
	`, targetID, limit)
	return signals, err
}
