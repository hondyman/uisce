package cbo

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"

	"github.com/hondyman/uisce/backend/internal/events"
)

const (
	defaultWindowMinutes       = 60
	defaultMinSampleCount      = 5
	defaultFailureRateFailover = 0.15
	defaultLatencyDegradedMs   = 2500.0
)

func defaults(cfg *CBOConfig) {
	if cfg.WindowMinutes == 0 {
		cfg.WindowMinutes = defaultWindowMinutes
	}
	if cfg.MinSampleCount == 0 {
		cfg.MinSampleCount = defaultMinSampleCount
	}
	if cfg.FailureRateFailover == 0 {
		cfg.FailureRateFailover = defaultFailureRateFailover
	}
	if cfg.LatencyDegradedMs == 0 {
		cfg.LatencyDegradedMs = defaultLatencyDegradedMs
	}
	if cfg.CacheTTLSeconds == 0 {
		cfg.CacheTTLSeconds = defaultCacheTTLSeconds
	}
}

type TelemetryRouter struct {
	db        *sqlx.DB
	cache     *TelemetryCache
	cfg       *CBOConfig
	logger    CBOLogger
	publisher events.CBOPublisher
}

func NewTelemetryRouter(db *sqlx.DB, redis RedisClient, cfg *CBOConfig, logger CBOLogger) *TelemetryRouter {
	if logger == nil {
		logger = NewNopLogger()
	}
	defaults(cfg)
	return &TelemetryRouter{
		db:     db,
		cache:  NewTelemetryCache(redis, cfg),
		cfg:    cfg,
		logger: logger,
	}
}

func (r *TelemetryRouter) WithPublisher(p events.CBOPublisher) {
	r.publisher = p
}

func (r *TelemetryRouter) GetOptimalFlavor(
	ctx context.Context,
	tenantID string,
	boKey string,
	defaultFlavor string,
) (string, error) {
	if !r.cfg.Enabled {
		return defaultFlavor, nil
	}

	if tenantID == "" {
		return defaultFlavor, nil
	}

	decision, err := r.cache.Get(ctx, tenantID, boKey)
	if err == nil && decision != nil {
		r.logger.Debug("CBO cache hit",
			zap.String("tenant", tenantID),
			zap.String("bo", boKey),
			zap.String("flavor", decision.RecommendedFlavor),
		)
		return decision.RecommendedFlavor, nil
	}

	computed, err := r.computeFromTelemetry(ctx, tenantID, boKey, defaultFlavor)
	if err != nil {
		r.logger.Warn("CBO telemetry compute failed, using default",
			zap.String("tenant", tenantID),
			zap.String("bo", boKey),
			zap.Error(err),
		)
		return defaultFlavor, nil
	}

	_ = r.cache.Set(ctx, computed)

	r.logger.Info("CBO flavor decision",
		zap.String("tenant", tenantID),
		zap.String("bo", boKey),
		zap.String("default_flavor", defaultFlavor),
		zap.String("recommended_flavor", computed.RecommendedFlavor),
		zap.String("reason", string(computed.OverrideReason)),
		zap.Float64("failure_rate", computed.FailureRate),
		zap.Float64("avg_latency_ms", computed.AvgLatencyMs),
		zap.Int("sample_count", computed.SampleCount),
	)

	if computed.RecommendedFlavor != defaultFlavor && r.publisher != nil {
		rerouteEvent := &events.CBORerouteEvent{
			EventID:            uuid.New().String(),
			EventType:          events.EventTypeCBOReroute,
			TenantID:           tenantID,
			BOName:             boKey,
			DefaultFlavor:      defaultFlavor,
			RecommendedFlavor:  computed.RecommendedFlavor,
			OverrideReason:     string(computed.OverrideReason),
			AvgLatencyMs:       computed.AvgLatencyMs,
			FailureRate:        computed.FailureRate,
			CacheHitRate:       computed.CacheHitRate,
			SampleCount:        computed.SampleCount,
			WindowMinutes:      computed.WindowMinutes,
			Timestamp:          time.Now(),
		}
		_ = r.publisher.PublishCBOReroute(ctx, rerouteEvent)
	}

	return computed.RecommendedFlavor, nil
}

type telemetryMetrics struct {
	AvgLatencyMs float64 `db:"avg_latency"`
	CacheHitRate float64 `db:"cache_hit_rate"`
	FailureRate  float64 `db:"failure_rate"`
	SampleCount  int     `db:"sample_count"`
}

func (r *TelemetryRouter) computeFromTelemetry(
	ctx context.Context,
	tenantID string,
	boKey string,
	defaultFlavor string,
) (*TelemetryFlavorDecision, error) {
	query := `
		SELECT
			COALESCE(AVG(execution_time_ms), 0.0) AS avg_latency,
			COALESCE(SUM(CASE WHEN cache_hit THEN 1 ELSE 0 END)::float /
			         NULLIF(COUNT(*), 0), 0.0) AS cache_hit_rate,
			COALESCE(SUM(CASE WHEN error IS NOT NULL THEN 1 ELSE 0 END)::float /
			         NULLIF(COUNT(*), 0), 0.0) AS failure_rate,
			COUNT(*) AS sample_count
		FROM semantic_query_history_v2
		WHERE tenant_id = $1
		  AND cube_name = $2
		  AND created_at >= NOW() - ($3 || ' minutes')::interval
	`

	var m telemetryMetrics
	err := r.db.QueryRowxContext(ctx, query, tenantID, boKey, r.cfg.WindowMinutes).StructScan(&m)
	if err == sql.ErrNoRows || m.SampleCount < r.cfg.MinSampleCount {
		return &TelemetryFlavorDecision{
			TenantID:          tenantID,
			BOName:            boKey,
			DefaultFlavor:     defaultFlavor,
			RecommendedFlavor: defaultFlavor,
			Source:            "rule",
			OverrideReason:    OverrideInsufficientTelemetry,
			SampleCount:       m.SampleCount,
			WindowMinutes:     r.cfg.WindowMinutes,
		}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query telemetry: %w", err)
	}

	reason := OverrideWithinSLO
	recommended := defaultFlavor

	if m.FailureRate > r.cfg.FailureRateFailover {
		reason = OverrideFailoverHighFailure
		if defaultFlavor == FlavorStarRocks {
			recommended = FlavorIceberg
		}
	} else if m.AvgLatencyMs > r.cfg.LatencyDegradedMs {
		reason = OverrideLatencyDegraded
		if defaultFlavor == FlavorStarRocks {
			recommended = FlavorIceberg
		}
	}

	return &TelemetryFlavorDecision{
		TenantID:          tenantID,
		BOName:            boKey,
		DefaultFlavor:     defaultFlavor,
		RecommendedFlavor: recommended,
		Source:            "telemetry",
		OverrideReason:    reason,
		AvgLatencyMs:      m.AvgLatencyMs,
		FailureRate:       m.FailureRate,
		CacheHitRate:      m.CacheHitRate,
		SampleCount:       m.SampleCount,
		WindowMinutes:     r.cfg.WindowMinutes,
	}, nil
}

func (r *TelemetryRouter) GetConfig() *CBOConfig {
	return r.cfg
}

func (r *TelemetryRouter) IsEnabled() bool {
	return r.cfg.Enabled
}

func (r *TelemetryRouter) GetDecision(
	ctx context.Context,
	tenantID string,
	boKey string,
	defaultFlavor string,
) (*TelemetryFlavorDecision, error) {
	if !r.cfg.Enabled || tenantID == "" {
		return &TelemetryFlavorDecision{
			TenantID:          tenantID,
			BOName:            boKey,
			DefaultFlavor:     defaultFlavor,
			RecommendedFlavor: defaultFlavor,
			Source:            "rule",
			OverrideReason:    OverrideWithinSLO,
		}, nil
	}

	if cached, err := r.cache.Get(ctx, tenantID, boKey); err == nil && cached != nil {
		return cached, nil
	}

	return r.computeFromTelemetry(ctx, tenantID, boKey, defaultFlavor)
}
