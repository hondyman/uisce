package engine_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"

	"github.com/hondyman/uisce/backend/internal/domain"
	"github.com/hondyman/uisce/backend/internal/engine"
	"github.com/stretchr/testify/assert"
)

type MockEventPublisher struct {
	mu           sync.Mutex
	PublishCount int
	LastTopic    string
	LastKey      string
	LastPayload  []byte
}

func (m *MockEventPublisher) Publish(topic string, key string, payload []byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.PublishCount++
	m.LastTopic = topic
	m.LastKey = key
	m.LastPayload = payload
	return nil
}

func TestFederatedExecutor_SecurityHardFail(t *testing.T) {
	pgDB, _, _ := sqlmock.New()
	srDB, _, _ := sqlmock.New()
	executor := engine.NewFederatedExecutor(pgDB, srDB)

	// Context MISSING tenant_id
	ctx := context.Background()
	req := domain.ExecutionRequest{
		GeneratedSQL: "SELECT * FROM public.customers",
		TargetEngine: "POSTGRES_HOT",
	}

	_, err := executor.Execute(ctx, req)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "missing tenant_id in context")
}

func TestAuditingExecutor_EmitsTelemetry(t *testing.T) {
	pgDB, pgMock, err := sqlmock.New()
	assert.NoError(t, err)
	defer pgDB.Close()

	srDB, _, _ := sqlmock.New()
	defer srDB.Close()

	// Expect EXPLAIN query
	pgMock.ExpectQuery("^EXPLAIN SELECT").WillReturnRows(sqlmock.NewRows([]string{"QUERY PLAN"}).AddRow("Seq Scan on customers"))
	// Expect ACTUAL query
	pgMock.ExpectQuery("^SELECT").WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))

	baseExecutor := engine.NewFederatedExecutor(pgDB, srDB)
	mockPublisher := &MockEventPublisher{}
	auditExecutor := engine.NewAuditingExecutor(baseExecutor, mockPublisher)

	ctx := context.WithValue(context.Background(), "tenant_id", "tenant-123")
	ctx = context.WithValue(ctx, "user_id", "user-456")

	req := domain.ExecutionRequest{
		BOID:         "bo-789",
		GeneratedSQL: "SELECT * FROM customers",
		TargetEngine: "POSTGRES_HOT",
	}

	rows, err := auditExecutor.Execute(ctx, req)
	assert.NoError(t, err)
	assert.NotNil(t, rows)

	// Wait briefly for non-blocking goroutine to complete
	time.Sleep(50 * time.Millisecond)

	// Verify Event was published asynchronously
	mockPublisher.mu.Lock()
	defer mockPublisher.mu.Unlock()
	assert.Equal(t, 1, mockPublisher.PublishCount)
	assert.Equal(t, "bo_execution_telemetry", mockPublisher.LastTopic)
	assert.Contains(t, string(mockPublisher.LastPayload), "Seq Scan on customers")
}

func TestCleanRoomBenchmarkingPublisher(t *testing.T) {
	mockPublisher := &MockEventPublisher{}
	cleanRoom := engine.NewCleanRoomBenchmarkingPublisher(mockPublisher, 0.02)

	ctx := context.WithValue(context.Background(), "tenant_id", "secret-tenant-777")
	err := cleanRoom.PublishAnonymizedMetric(ctx, "bo-trade-123", "AssetManager", "avg_settlement_duration_sec", 120.0)

	assert.NoError(t, err)
	assert.Equal(t, 1, mockPublisher.PublishCount)
	assert.Equal(t, "bo_benchmark_anonymized", mockPublisher.LastTopic)
	assert.Equal(t, "AssetManager", mockPublisher.LastKey)

	payloadStr := string(mockPublisher.LastPayload)
	assert.Contains(t, payloadStr, "avg_settlement_duration_sec")
	assert.Contains(t, payloadStr, "AssetManager")
	// Verify Rule 7: secret-tenant-777 MUST NOT be leaked in benchmark payload
	assert.NotContains(t, payloadStr, "secret-tenant-777")
}


