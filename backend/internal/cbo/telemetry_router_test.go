package cbo

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"
)

func TestTelemetryRouter_GetOptimalFlavor_Disabled(t *testing.T) {
	db, _, _ := sqlmock.New()
	defer db.Close()

	cfg := &CBOConfig{Enabled: false}
	router := NewTelemetryRouter(sqlx.NewDb(db, "postgres"), nil, cfg, nil)

	flavor, err := router.GetOptimalFlavor(context.Background(), "tenant-disabled", "customers", FlavorStarRocks)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if flavor != FlavorStarRocks {
		t.Errorf("expected STARROCKS when disabled, got %s", flavor)
	}
}

func TestTelemetryRouter_GetOptimalFlavor_EmptyTenant(t *testing.T) {
	db, _, _ := sqlmock.New()
	defer db.Close()

	cfg := &CBOConfig{Enabled: true}
	router := NewTelemetryRouter(sqlx.NewDb(db, "postgres"), nil, cfg, nil)

	flavor, err := router.GetOptimalFlavor(context.Background(), "", "customers", FlavorStarRocks)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if flavor != FlavorStarRocks {
		t.Errorf("expected STARROCKS with empty tenant, got %s", flavor)
	}
}

func TestTelemetryRouter_GetOptimalFlavor_InsufficientSamples(t *testing.T) {
	db, mock, _ := sqlmock.New()
	defer db.Close()

	cfg := &CBOConfig{Enabled: true, MinSampleCount: 10, WindowMinutes: 60}
	router := NewTelemetryRouter(sqlx.NewDb(db, "postgres"), newMockRedis(), cfg, nil)

	rows := sqlmock.NewRows([]string{"avg_latency", "cache_hit_rate", "failure_rate", "sample_count"}).
		AddRow(150.0, 0.5, 0.0, 3)

	mock.ExpectQuery("SELECT").
		WithArgs("tenant-insufficient", "customers", 60).
		WillReturnRows(rows)

	flavor, err := router.GetOptimalFlavor(context.Background(), "tenant-insufficient", "customers", FlavorStarRocks)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if flavor != FlavorStarRocks {
		t.Errorf("expected STARROCKS with insufficient samples, got %s", flavor)
	}

	decision, _ := router.GetDecision(context.Background(), "tenant-insufficient", "customers", FlavorStarRocks)
	if decision.OverrideReason != OverrideInsufficientTelemetry {
		t.Errorf("expected OverrideInsufficientTelemetry, got %s", decision.OverrideReason)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %v", err)
	}
}

func TestTelemetryRouter_GetOptimalFlavor_HighFailureRate(t *testing.T) {
	db, mock, _ := sqlmock.New()
	defer db.Close()

	cfg := &CBOConfig{Enabled: true, MinSampleCount: 5, FailureRateFailover: 0.15, WindowMinutes: 60}
	router := NewTelemetryRouter(sqlx.NewDb(db, "postgres"), newMockRedis(), cfg, nil)

	rows := sqlmock.NewRows([]string{"avg_latency", "cache_hit_rate", "failure_rate", "sample_count"}).
		AddRow(100.0, 0.5, 0.25, 50)

	mock.ExpectQuery("SELECT").
		WithArgs("tenant-failure", "customers", 60).
		WillReturnRows(rows)

	flavor, err := router.GetOptimalFlavor(context.Background(), "tenant-failure", "customers", FlavorStarRocks)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if flavor != FlavorIceberg {
		t.Errorf("expected ICEBERG with high failure rate, got %s", flavor)
	}

	decision, _ := router.GetDecision(context.Background(), "tenant-failure", "customers", FlavorStarRocks)
	if decision.OverrideReason != OverrideFailoverHighFailure {
		t.Errorf("expected OverrideFailoverHighFailure, got %s", decision.OverrideReason)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %v", err)
	}
}

func TestTelemetryRouter_GetOptimalFlavor_HighFailureRateIcebergDefault(t *testing.T) {
	db, mock, _ := sqlmock.New()
	defer db.Close()

	cfg := &CBOConfig{Enabled: true, MinSampleCount: 5, FailureRateFailover: 0.15, WindowMinutes: 60}
	router := NewTelemetryRouter(sqlx.NewDb(db, "postgres"), newMockRedis(), cfg, nil)

	rows := sqlmock.NewRows([]string{"avg_latency", "cache_hit_rate", "failure_rate", "sample_count"}).
		AddRow(100.0, 0.5, 0.25, 50)

	mock.ExpectQuery("SELECT").
		WithArgs("tenant-iceberg-default", "customers", 60).
		WillReturnRows(rows)

	flavor, err := router.GetOptimalFlavor(context.Background(), "tenant-iceberg-default", "customers", FlavorIceberg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if flavor != FlavorIceberg {
		t.Errorf("expected ICEBERG (no failover destination), got %s", flavor)
	}

	decision, _ := router.GetDecision(context.Background(), "tenant-iceberg-default", "customers", FlavorIceberg)
	if decision.OverrideReason != OverrideFailoverHighFailure {
		t.Errorf("expected OverrideFailoverHighFailure, got %s", decision.OverrideReason)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %v", err)
	}
}

func TestTelemetryRouter_GetOptimalFlavor_LatencyDegraded(t *testing.T) {
	db, mock, _ := sqlmock.New()
	defer db.Close()

	cfg := &CBOConfig{Enabled: true, MinSampleCount: 5, FailureRateFailover: 0.15, LatencyDegradedMs: 2500, WindowMinutes: 60}
	router := NewTelemetryRouter(sqlx.NewDb(db, "postgres"), newMockRedis(), cfg, nil)

	rows := sqlmock.NewRows([]string{"avg_latency", "cache_hit_rate", "failure_rate", "sample_count"}).
		AddRow(5000.0, 0.1, 0.05, 50)

	mock.ExpectQuery("SELECT").
		WithArgs("tenant-latency", "customers", 60).
		WillReturnRows(rows)

	flavor, err := router.GetOptimalFlavor(context.Background(), "tenant-latency", "customers", FlavorStarRocks)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if flavor != FlavorIceberg {
		t.Errorf("expected ICEBERG with degraded latency, got %s", flavor)
	}

	decision, _ := router.GetDecision(context.Background(), "tenant-latency", "customers", FlavorStarRocks)
	if decision.OverrideReason != OverrideLatencyDegraded {
		t.Errorf("expected OverrideLatencyDegraded, got %s", decision.OverrideReason)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %v", err)
	}
}

func TestTelemetryRouter_GetOptimalFlavor_WithinSLO(t *testing.T) {
	db, mock, _ := sqlmock.New()
	defer db.Close()

	cfg := &CBOConfig{Enabled: true, MinSampleCount: 5, FailureRateFailover: 0.15, LatencyDegradedMs: 2500, WindowMinutes: 60}
	router := NewTelemetryRouter(sqlx.NewDb(db, "postgres"), newMockRedis(), cfg, nil)

	rows := sqlmock.NewRows([]string{"avg_latency", "cache_hit_rate", "failure_rate", "sample_count"}).
		AddRow(150.0, 0.8, 0.01, 100)

	mock.ExpectQuery("SELECT").
		WithArgs("tenant-slo", "customers", 60).
		WillReturnRows(rows)

	flavor, err := router.GetOptimalFlavor(context.Background(), "tenant-slo", "customers", FlavorStarRocks)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if flavor != FlavorStarRocks {
		t.Errorf("expected STARROCKS when within SLO, got %s", flavor)
	}

	decision, _ := router.GetDecision(context.Background(), "tenant-slo", "customers", FlavorStarRocks)
	if decision.OverrideReason != OverrideWithinSLO {
		t.Errorf("expected OverrideWithinSLO, got %s", decision.OverrideReason)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %v", err)
	}
}

type mockRedis struct {
	data map[string]string
}

func newMockRedis() *mockRedis {
	return &mockRedis{data: make(map[string]string)}
}

func (m *mockRedis) Get(ctx context.Context, key string) (string, error) {
	if val, ok := m.data[key]; ok {
		return val, nil
	}
	return "", nil
}

func (m *mockRedis) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	switch v := value.(type) {
	case string:
		m.data[key] = v
	case []byte:
		m.data[key] = string(v)
	default:
		m.data[key] = fmt.Sprintf("%v", v)
	}
	return nil
}

func (m *mockRedis) Ping(ctx context.Context) error {
	return nil
}

func (m *mockRedis) Close() error {
	return nil
}

func TestTelemetryCache_Get_Set(t *testing.T) {
	redis := newMockRedis()
	cache := NewTelemetryCache(redis, &CBOConfig{CacheTTLSeconds: 60})

	decision := &TelemetryFlavorDecision{
		TenantID:          "tenant-1",
		BOName:           "customers",
		DefaultFlavor:    FlavorStarRocks,
		RecommendedFlavor: FlavorIceberg,
		Source:           "telemetry",
		OverrideReason:   OverrideFailoverHighFailure,
		AvgLatencyMs:     100.0,
		FailureRate:      0.25,
		SampleCount:      50,
	}

	ctx := context.Background()
	err := cache.Set(ctx, decision)
	if err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	retrieved, err := cache.Get(ctx, "tenant-1", "customers")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if retrieved == nil {
		t.Fatal("expected non-nil decision")
	}
	if retrieved.RecommendedFlavor != FlavorIceberg {
		t.Errorf("expected ICEBERG, got %s", retrieved.RecommendedFlavor)
	}
	if retrieved.OverrideReason != OverrideFailoverHighFailure {
		t.Errorf("expected OverrideFailoverHighFailure, got %s", retrieved.OverrideReason)
	}
}

func TestTelemetryCache_CacheKey(t *testing.T) {
	redis := newMockRedis()
	cache := NewTelemetryCache(redis, &CBOConfig{CacheTTLSeconds: 60})

	decision := &TelemetryFlavorDecision{
		TenantID:          "t1",
		BOName:           "b1",
		DefaultFlavor:    FlavorStarRocks,
		RecommendedFlavor: FlavorStarRocks,
		OverrideReason:   OverrideWithinSLO,
	}

	ctx := context.Background()
	_ = cache.Set(ctx, decision)

	expectedKey := "tenant:t1:cbo:b1"
	if _, ok := redis.data[expectedKey]; !ok {
		t.Errorf("expected cache key %s in redis data", expectedKey)
	}
}

func TestTelemetryRouter_CacheHit(t *testing.T) {
	db, _, _ := sqlmock.New()
	defer db.Close()

	redis := newMockRedis()

	decision := &TelemetryFlavorDecision{
		TenantID:          "tenant-cachehit",
		BOName:           "customers",
		DefaultFlavor:    FlavorStarRocks,
		RecommendedFlavor: FlavorIceberg,
		Source:           "telemetry",
		OverrideReason:   OverrideFailoverHighFailure,
		AvgLatencyMs:     100.0,
		FailureRate:      0.25,
		SampleCount:      50,
	}
	encoded, _ := json.Marshal(decision)
	redis.data["tenant:tenant-cachehit:cbo:customers"] = string(encoded)

	cfg := &CBOConfig{Enabled: true}
	router := NewTelemetryRouter(sqlx.NewDb(db, "postgres"), redis, cfg, nil)

	flavor, err := router.GetOptimalFlavor(context.Background(), "tenant-cachehit", "customers", FlavorStarRocks)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if flavor != FlavorIceberg {
		t.Errorf("expected ICEBERG from cache, got %s", flavor)
	}
}

func TestTelemetryRouter_GetDecision(t *testing.T) {
	db, mock, _ := sqlmock.New()
	defer db.Close()

	cfg := &CBOConfig{Enabled: true, MinSampleCount: 5, FailureRateFailover: 0.15, LatencyDegradedMs: 2500, WindowMinutes: 60}
	router := NewTelemetryRouter(sqlx.NewDb(db, "postgres"), newMockRedis(), cfg, nil)

	rows := sqlmock.NewRows([]string{"avg_latency", "cache_hit_rate", "failure_rate", "sample_count"}).
		AddRow(150.0, 0.8, 0.01, 100)

	mock.ExpectQuery("SELECT").
		WithArgs("tenant-decision", "customers", 60).
		WillReturnRows(rows)

	ctx := context.Background()
	decision, err := router.GetDecision(ctx, "tenant-decision", "customers", FlavorStarRocks)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if decision.TenantID != "tenant-decision" {
		t.Errorf("expected tenant-decision, got %s", decision.TenantID)
	}
	if decision.BOName != "customers" {
		t.Errorf("expected customers, got %s", decision.BOName)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %v", err)
	}
}

func TestDefaults(t *testing.T) {
	cfg := &CBOConfig{}
	defaults(cfg)

	if cfg.WindowMinutes != defaultWindowMinutes {
		t.Errorf("expected WindowMinutes %d, got %d", defaultWindowMinutes, cfg.WindowMinutes)
	}
	if cfg.MinSampleCount != defaultMinSampleCount {
		t.Errorf("expected MinSampleCount %d, got %d", defaultMinSampleCount, cfg.MinSampleCount)
	}
	if cfg.FailureRateFailover != defaultFailureRateFailover {
		t.Errorf("expected FailureRateFailover %f, got %f", defaultFailureRateFailover, cfg.FailureRateFailover)
	}
	if cfg.LatencyDegradedMs != defaultLatencyDegradedMs {
		t.Errorf("expected LatencyDegradedMs %f, got %f", defaultLatencyDegradedMs, cfg.LatencyDegradedMs)
	}
}

func TestNewZapLogger(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	cboLogger := NewZapLogger(logger)

	cboLogger.Info("test info")
	cboLogger.Warn("test warn")
	cboLogger.Debug("test debug")
}
