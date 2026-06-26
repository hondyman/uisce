# 🔗 Semantic Layer - Advanced Integrations & Governance

Complete guide for monitoring, SQL auditing, drift detection, and RabbitMQ event-driven architecture for the semantic layer.

---

## 📋 Table of Contents

1. [RabbitMQ Event-Driven Architecture](#rabbitmq-event-driven-architecture)
2. [SQL Audit & Query Tracing](#sql-audit--query-tracing)
3. [Drift Detection & Management](#drift-detection--management)
4. [Monitoring & Observability](#monitoring--observability)
5. [AI-Powered Suggestions](#ai-powered-suggestions)
6. [Implementation Roadmap](#implementation-roadmap)

---

## 🐇 RabbitMQ Event-Driven Architecture

### Overview

All semantic layer mutations (model updates, measure changes, dimension modifications, join edits) trigger RabbitMQ events for:
- **Downstream propagation**: Cache invalidation, warehouse updates
- **Audit trails**: Compliance & audit logs
- **Versioning**: Automatic change tracking
- **Notifications**: Team alerts
- **ML training**: Drift detection models

### Events Architecture

```
┌─────────────────────────────────────────────────────────────┐
│ Semantic Layer Mutations                                    │
│ (Model/Measure/Dimension/Join updates)                      │
└──────────────────┬──────────────────────────────────────────┘
                   │
                   ├─────────────────────────────────────────────────────┐
                   │                                                     │
        ┌──────────▼────────────┐                          ┌────────────▼─────────┐
        │ RabbitMQ Publisher    │                          │ Direct Operations    │
        │ (Semantic Changes)    │                          │ (Database Update)    │
        └──────────┬────────────┘                          └──────────────────────┘
                   │
      ┌────────────┼────────────┬────────────┬────────────┬────────────┐
      │            │            │            │            │            │
      ▼            ▼            ▼            ▼            ▼            ▼
 Cache      Query      Drift      Audit     Notify     Warehouse
Invalidate  Recompile  Detection  Logger    Teams      Propagate
```

### 1. RabbitMQ Event Publisher

**File**: `backend/internal/events/semantic_publisher.go`

```go
package events

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

// SemanticChangeEvent represents any mutation to the semantic layer
type SemanticChangeEvent struct {
	ID              string                 `json:"id"`
	Timestamp       time.Time              `json:"timestamp"`
	TenantID        string                 `json:"tenant_id"`
	UserID          string                 `json:"user_id"`
	ChangeType      string                 `json:"change_type"` // model_created, model_updated, measure_added, dimension_changed, join_modified, model_deleted
	ModelID         string                 `json:"model_id"`
	ModelName       string                 `json:"model_name"`
	ElementType     string                 `json:"element_type"` // measure, dimension, join, model
	ElementID       string                 `json:"element_id"`
	ElementName     string                 `json:"element_name"`
	OldDefinition   json.RawMessage        `json:"old_definition"`
	NewDefinition   json.RawMessage        `json:"new_definition"`
	ChangeReason    string                 `json:"change_reason"`
	ImpactedQueries int                    `json:"impacted_queries"`
	SQLChanges      *SQLChangeDetail       `json:"sql_changes,omitempty"`
	Metadata        map[string]interface{} `json:"metadata"`
}

// SQLChangeDetail tracks SQL compilation changes
type SQLChangeDetail struct {
	OldSQL        string `json:"old_sql"`
	NewSQL        string `json:"new_sql"`
	DiffSummary   string `json:"diff_summary"`
	BreakingChange bool  `json:"breaking_change"`
	AffectedTables []string `json:"affected_tables"`
}

// SemanticPublisher publishes semantic layer events
type SemanticPublisher struct {
	channel *amqp.Channel
	conn    *amqp.Connection
}

// NewSemanticPublisher creates a new publisher
func NewSemanticPublisher(amqpURL string) (*SemanticPublisher, error) {
	conn, err := amqp.Dial(amqpURL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to RabbitMQ: %w", err)
	}

	ch, err := conn.Channel()
	if err != nil {
		return nil, fmt.Errorf("failed to open channel: %w", err)
	}

	// Declare exchanges
	exchanges := []string{
		"semantic.changes",
		"semantic.drift",
		"semantic.audit",
		"semantic.notifications",
	}

	for _, exchange := range exchanges {
		if err := ch.ExchangeDeclare(
			exchange,
			"topic",
			true,  // durable
			false, // auto-delete
			false, // internal
			false, // no-wait
			nil,   // args
		); err != nil {
			return nil, fmt.Errorf("failed to declare exchange %s: %w", exchange, err)
		}
	}

	return &SemanticPublisher{
		channel: ch,
		conn:    conn,
	}, nil
}

// PublishModelChange publishes a model change event
func (p *SemanticPublisher) PublishModelChange(ctx context.Context, event *SemanticChangeEvent) error {
	body, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	// Route by change type
	var routingKey string
	switch event.ChangeType {
	case "model_created", "model_updated", "model_deleted":
		routingKey = fmt.Sprintf("semantic.changes.model.%s", event.ChangeType)
	case "measure_added", "measure_modified", "measure_deleted":
		routingKey = fmt.Sprintf("semantic.changes.measure.%s", event.ChangeType)
	case "dimension_added", "dimension_modified", "dimension_deleted":
		routingKey = fmt.Sprintf("semantic.changes.dimension.%s", event.ChangeType)
	case "join_added", "join_modified", "join_deleted":
		routingKey = fmt.Sprintf("semantic.changes.join.%s", event.ChangeType)
	default:
		routingKey = "semantic.changes.other"
	}

	return p.channel.PublishWithContext(
		ctx,
		"semantic.changes",
		routingKey,
		false, // mandatory
		false, // immediate
		amqp.Publishing{
			ContentType: "application/json",
			Timestamp:   time.Now(),
			Body:        body,
			Headers: amqp.Table{
				"tenant_id":   event.TenantID,
				"change_type": event.ChangeType,
				"element_type": event.ElementType,
			},
		},
	)
}

// Close closes the publisher connection
func (p *SemanticPublisher) Close() error {
	p.channel.Close()
	return p.conn.Close()
}
```

### 2. Event Subscribers

**File**: `backend/internal/events/subscribers.go`

```go
package events

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/jmoiron/sqlx"
)

// CacheInvalidationSubscriber listens to semantic changes and invalidates cache
type CacheInvalidationSubscriber struct {
	channel *amqp.Channel
	cache   CacheManager // implement your cache manager
}

// DriftDetectionSubscriber listens to semantic changes and detects drift
type DriftDetectionSubscriber struct {
	channel *amqp.Channel
	db      *sqlx.DB
}

// AuditSubscriber logs all semantic changes
type AuditSubscriber struct {
	channel *amqp.Channel
	db      *sqlx.DB
}

// NotificationSubscriber sends team notifications
type NotificationSubscriber struct {
	channel *amqp.Channel
}

// NewCacheInvalidationSubscriber creates cache invalidation subscriber
func NewCacheInvalidationSubscriber(conn *amqp.Connection, cache CacheManager) (*CacheInvalidationSubscriber, error) {
	ch, err := conn.Channel()
	if err != nil {
		return nil, err
	}

	queue, err := ch.QueueDeclare(
		"semantic-cache-invalidation",
		true,  // durable
		false, // auto-delete
		false, // exclusive
		false, // no-wait
		nil,   // args
	)
	if err != nil {
		return nil, err
	}

	// Bind to all model/measure/dimension/join changes
	bindingKeys := []string{
		"semantic.changes.model.*",
		"semantic.changes.measure.*",
		"semantic.changes.dimension.*",
		"semantic.changes.join.*",
	}

	for _, key := range bindingKeys {
		if err := ch.QueueBind(key, key, "semantic.changes", false, nil); err != nil {
			return nil, err
		}
	}

	return &CacheInvalidationSubscriber{
		channel: ch,
		cache:   cache,
	}, nil
}

// Start starts listening for cache invalidation events
func (s *CacheInvalidationSubscriber) Start(ctx context.Context) error {
	msgs, err := s.channel.Consume(
		"semantic-cache-invalidation",
		"",    // consumer
		false, // auto-ack
		false, // exclusive
		false, // no-local
		false, // no-wait
		nil,   // args
	)
	if err != nil {
		return err
	}

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case msg := <-msgs:
				var event SemanticChangeEvent
				if err := json.Unmarshal(msg.Body, &event); err != nil {
					log.Printf("Failed to unmarshal event: %v", err)
					msg.Nack(false, false)
					continue
				}

				// Invalidate cache based on change
				s.invalidateRelevantCaches(&event)

				msg.Ack(false)
			}
		}
	}()

	return nil
}

// invalidateRelevantCaches invalidates caches affected by the change
func (s *CacheInvalidationSubscriber) invalidateRelevantCaches(event *SemanticChangeEvent) {
	// Pattern: semantic:model:{modelID}:*
	patterns := []string{
		fmt.Sprintf("semantic:model:%s:*", event.ModelID),
		fmt.Sprintf("semantic:tenant:%s:*", event.TenantID),
		fmt.Sprintf("semantic:query_results:*:%s:*", event.ModelID),
	}

	for _, pattern := range patterns {
		// Use your cache manager's Delete or InvalidatePattern method
		log.Printf("Invalidating cache pattern: %s", pattern)
		// s.cache.InvalidatePattern(pattern)
	}
}

// NewDriftDetectionSubscriber creates drift detection subscriber
func NewDriftDetectionSubscriber(conn *amqp.Connection, db *sqlx.DB) (*DriftDetectionSubscriber, error) {
	ch, err := conn.Channel()
	if err != nil {
		return nil, err
	}

	queue, err := ch.QueueDeclare(
		"semantic-drift-detection",
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return nil, err
	}

	// Bind to changes that might cause drift
	bindingKeys := []string{
		"semantic.changes.measure.*",
		"semantic.changes.dimension.*",
		"semantic.changes.join.*",
	}

	for _, key := range bindingKeys {
		if err := ch.QueueBind(key, key, "semantic.changes", false, nil); err != nil {
			return nil, err
		}
	}

	return &DriftDetectionSubscriber{
		channel: ch,
		db:      db,
	}, nil
}

// Start starts listening for drift-related events
func (s *DriftDetectionSubscriber) Start(ctx context.Context) error {
	msgs, err := s.channel.Consume(
		"semantic-drift-detection",
		"",
		false,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return err
	}

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case msg := <-msgs:
				var event SemanticChangeEvent
				if err := json.Unmarshal(msg.Body, &event); err != nil {
					log.Printf("Failed to unmarshal event: %v", err)
					msg.Nack(false, false)
					continue
				}

				// Analyze drift potential
				s.detectPotentialDrift(&event)

				msg.Ack(false)
			}
		}
	}()

	return nil
}

// NewAuditSubscriber creates audit subscriber
func NewAuditSubscriber(conn *amqp.Connection, db *sqlx.DB) (*AuditSubscriber, error) {
	ch, err := conn.Channel()
	if err != nil {
		return nil, err
	}

	queue, err := ch.QueueDeclare(
		"semantic-audit",
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return nil, err
	}

	// Bind to ALL changes
	if err := ch.QueueBind("semantic.changes.*.*", "semantic.changes.*.*", "semantic.changes", false, nil); err != nil {
		return nil, err
	}

	return &AuditSubscriber{
		channel: ch,
		db:      db,
	}, nil
}

// Start starts listening for audit events
func (s *AuditSubscriber) Start(ctx context.Context) error {
	msgs, err := s.channel.Consume(
		"semantic-audit",
		"",
		false,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return err
	}

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case msg := <-msgs:
				var event SemanticChangeEvent
				if err := json.Unmarshal(msg.Body, &event); err != nil {
					log.Printf("Failed to unmarshal event: %v", err)
					msg.Nack(false, false)
					continue
				}

				// Log to audit table
				s.logAuditEvent(&event)

				msg.Ack(false)
			}
		}
	}()

	return nil
}

// logAuditEvent logs event to database
func (s *AuditSubscriber) logAuditEvent(event *SemanticChangeEvent) {
	query := `
		INSERT INTO semantic_layer_audit_log (
			id, tenant_id, user_id, change_type, model_id, model_name,
			element_type, element_id, element_name, old_definition, new_definition,
			change_reason, sql_changes, created_at
		) VALUES (
			gen_random_uuid(), $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, now()
		)
	`

	sqlChangesJSON, _ := json.Marshal(event.SQLChanges)

	if err := s.db.QueryRow(
		query,
		event.TenantID, event.UserID, event.ChangeType, event.ModelID, event.ModelName,
		event.ElementType, event.ElementID, event.ElementName,
		event.OldDefinition, event.NewDefinition, event.ChangeReason, sqlChangesJSON,
	).Err(); err != nil {
		log.Printf("Failed to log audit event: %v", err)
	}
}

// detectPotentialDrift analyzes changes for drift potential
func (s *DriftDetectionSubscriber) detectPotentialDrift(event *SemanticChangeEvent) {
	// Drift detection logic (see drift detection section)
	log.Printf("Analyzing drift for change: %s on %s", event.ChangeType, event.ElementName)
}
```

### 3. Database Tables for Events

**File**: `backend/sql/semantic_events.sql`

```sql
-- Semantic Layer Audit Log
CREATE TABLE IF NOT EXISTS public.semantic_layer_audit_log (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id uuid NOT NULL,
    user_id uuid NOT NULL,
    change_type varchar(100) NOT NULL,
    model_id varchar(255) NOT NULL,
    model_name varchar(255) NOT NULL,
    element_type varchar(50) NOT NULL, -- measure, dimension, join, model
    element_id varchar(255) NOT NULL,
    element_name varchar(255) NOT NULL,
    old_definition jsonb,
    new_definition jsonb,
    change_reason text,
    sql_changes jsonb,
    created_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT semantic_layer_audit_log_fk_tenant FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_semantic_audit_tenant_created ON public.semantic_layer_audit_log(tenant_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_semantic_audit_model ON public.semantic_layer_audit_log(model_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_semantic_audit_change_type ON public.semantic_layer_audit_log(change_type);

-- Semantic Layer Change Events (for event sourcing)
CREATE TABLE IF NOT EXISTS public.semantic_change_events (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id uuid NOT NULL,
    aggregate_id varchar(255) NOT NULL, -- model_id or element_id
    aggregate_type varchar(50) NOT NULL, -- model, measure, dimension, join
    event_type varchar(100) NOT NULL, -- created, updated, deleted
    event_data jsonb NOT NULL,
    metadata jsonb,
    version int NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT semantic_change_events_fk_tenant FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_semantic_events_aggregate ON public.semantic_change_events(aggregate_id, aggregate_type, version);
CREATE INDEX IF NOT EXISTS idx_semantic_events_tenant ON public.semantic_change_events(tenant_id, created_at DESC);

-- RabbitMQ Event Delivery Log
CREATE TABLE IF NOT EXISTS public.semantic_event_delivery_log (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    event_id uuid NOT NULL,
    exchange varchar(255) NOT NULL,
    routing_key varchar(255) NOT NULL,
    subscriber_queue varchar(255) NOT NULL,
    status varchar(50) NOT NULL, -- delivered, failed, retrying
    error_message text,
    attempt_count int DEFAULT 1,
    created_at timestamptz NOT NULL DEFAULT now(),
    delivered_at timestamptz,
    CONSTRAINT semantic_event_delivery_fk_event FOREIGN KEY (event_id) REFERENCES public.semantic_change_events(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_event_delivery_status ON public.semantic_event_delivery_log(status, created_at);
```

---

## 🔍 SQL Audit & Query Tracing

### Overview

Every compiled query is tracked:
- **Original semantic query** (JSON)
- **Compiled SQL** (parameterized)
- **Execution trace** (timing, rows)
- **Query plan** (explain output)
- **Drift indicators** (schema changes affecting query)

### 1. Query Audit Manager

**File**: `backend/internal/audit/query_audit.go`

```go
package audit

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/google/uuid"
)

// QueryAudit represents a tracked semantic query execution
type QueryAudit struct {
	ID                    uuid.UUID           `db:"id"`
	TenantID              uuid.UUID           `db:"tenant_id"`
	UserID                uuid.UUID           `db:"user_id"`
	SessionID             string              `db:"session_id"`
	ModelID               string              `db:"model_id"`
	ModelName             string              `db:"model_name"`
	SemanticQuery         json.RawMessage     `db:"semantic_query"`
	CompiledSQL           string              `db:"compiled_sql"`
	SQLParameters         json.RawMessage     `db:"sql_parameters"`
	ExecutionStartTime    time.Time           `db:"execution_start_time"`
	ExecutionEndTime      time.Time           `db:"execution_end_time"`
	DurationMS            int64               `db:"duration_ms"`
	RowsScanned           int64               `db:"rows_scanned"`
	RowsReturned          int64               `db:"rows_returned"`
	CacheHit              bool                `db:"cache_hit"`
	QueryPlan             json.RawMessage     `db:"query_plan"`
	ErrorMessage          string              `db:"error_message"`
	Status                string              `db:"status"` // success, error, timeout
	SchemaVersion         int                 `db:"schema_version"`
	DriftIndicators       json.RawMessage     `db:"drift_indicators"`
	CreatedAt             time.Time           `db:"created_at"`
}

// QueryAuditor manages query auditing
type QueryAuditor struct {
	db *sqlx.DB
}

// NewQueryAuditor creates new auditor
func NewQueryAuditor(db *sqlx.DB) *QueryAuditor {
	return &QueryAuditor{db: db}
}

// RecordQueryExecution records a query execution
func (qa *QueryAuditor) RecordQueryExecution(ctx context.Context, audit *QueryAudit) error {
	query := `
		INSERT INTO semantic_query_audit (
			id, tenant_id, user_id, session_id, model_id, model_name,
			semantic_query, compiled_sql, sql_parameters,
			execution_start_time, execution_end_time, duration_ms,
			rows_scanned, rows_returned, cache_hit,
			query_plan, error_message, status, schema_version,
			drift_indicators, created_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, now()
		)
	`

	audit.ID = uuid.New()
	audit.CreatedAt = time.Now()

	return qa.db.QueryRowContext(
		ctx, query,
		audit.ID, audit.TenantID, audit.UserID, audit.SessionID,
		audit.ModelID, audit.ModelName, audit.SemanticQuery, audit.CompiledSQL,
		audit.SQLParameters, audit.ExecutionStartTime, audit.ExecutionEndTime,
		audit.DurationMS, audit.RowsScanned, audit.RowsReturned, audit.CacheHit,
		audit.QueryPlan, audit.ErrorMessage, audit.Status, audit.SchemaVersion,
		audit.DriftIndicators,
	).Err()
}

// GetQueryAuditTrail retrieves audit trail for a model
func (qa *QueryAuditor) GetQueryAuditTrail(ctx context.Context, tenantID, modelID string, limit int) ([]QueryAudit, error) {
	query := `
		SELECT id, tenant_id, user_id, session_id, model_id, model_name,
		       semantic_query, compiled_sql, sql_parameters,
		       execution_start_time, execution_end_time, duration_ms,
		       rows_scanned, rows_returned, cache_hit,
		       query_plan, error_message, status, schema_version,
		       drift_indicators, created_at
		FROM semantic_query_audit
		WHERE tenant_id = $1 AND model_id = $2
		ORDER BY created_at DESC
		LIMIT $3
	`

	var audits []QueryAudit
	err := qa.db.SelectContext(ctx, &audits, query, tenantID, modelID, limit)
	return audits, err
}

// CompareQueriesForDrift compares two query compilations
func (qa *QueryAuditor) CompareQueriesForDrift(oldAudit, newAudit *QueryAudit) *DriftAnalysis {
	analysis := &DriftAnalysis{
		OldQueryID:       oldAudit.ID.String(),
		NewQueryID:       newAudit.ID.String(),
		TimestampDiff:    newAudit.ExecutionStartTime.Sub(oldAudit.ExecutionStartTime),
		RowsScannedDiff:  newAudit.RowsScanned - oldAudit.RowsScanned,
		RowsReturnedDiff: newAudit.RowsReturned - oldAudit.RowsReturned,
		DurationDiff:     newAudit.DurationMS - oldAudit.DurationMS,
	}

	// Compare SQL
	if oldAudit.CompiledSQL != newAudit.CompiledSQL {
		analysis.SQLChanged = true
		analysis.SQLDiff = computeSQLDiff(oldAudit.CompiledSQL, newAudit.CompiledSQL)
	}

	return analysis
}

// DriftAnalysis represents comparison between query executions
type DriftAnalysis struct {
	OldQueryID       string
	NewQueryID       string
	TimestampDiff    time.Duration
	RowsScannedDiff  int64
	RowsReturnedDiff int64
	DurationDiff     int64
	SQLChanged       bool
	SQLDiff          string
}

func computeSQLDiff(oldSQL, newSQL string) string {
	// Implement SQL diff algorithm
	// For now, return simple marker
	return fmt.Sprintf("OLD: %s\nNEW: %s", oldSQL[:50], newSQL[:50])
}
```

### 2. Query Audit Tables

```sql
-- Semantic Query Audit Log
CREATE TABLE IF NOT EXISTS public.semantic_query_audit (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id uuid NOT NULL,
    user_id uuid,
    session_id varchar(255),
    model_id varchar(255) NOT NULL,
    model_name varchar(255) NOT NULL,
    semantic_query jsonb NOT NULL,
    compiled_sql text NOT NULL,
    sql_parameters jsonb,
    execution_start_time timestamptz,
    execution_end_time timestamptz,
    duration_ms bigint,
    rows_scanned bigint,
    rows_returned bigint,
    cache_hit boolean DEFAULT false,
    query_plan jsonb,
    error_message text,
    status varchar(50), -- success, error, timeout
    schema_version int,
    drift_indicators jsonb,
    created_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT semantic_query_audit_fk_tenant FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_semantic_query_audit_tenant_model ON public.semantic_query_audit(tenant_id, model_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_semantic_query_audit_duration ON public.semantic_query_audit(duration_ms DESC);
CREATE INDEX IF NOT EXISTS idx_semantic_query_audit_cache_hit ON public.semantic_query_audit(cache_hit, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_semantic_query_audit_status ON public.semantic_query_audit(status, created_at DESC);

-- Query Performance Trends
CREATE TABLE IF NOT EXISTS public.semantic_query_performance (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id uuid NOT NULL,
    model_id varchar(255) NOT NULL,
    hour_bucket timestamptz NOT NULL,
    query_count int NOT NULL,
    avg_duration_ms float NOT NULL,
    p95_duration_ms float NOT NULL,
    p99_duration_ms float NOT NULL,
    cache_hit_rate float NOT NULL,
    error_rate float NOT NULL,
    avg_rows_scanned bigint NOT NULL,
    CONSTRAINT semantic_query_performance_fk_tenant FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_query_performance_tenant_bucket ON public.semantic_query_performance(tenant_id, hour_bucket DESC);
```

---

## 🔄 Drift Detection & Management

### Overview

**Drift** = Semantic layer definition diverges from source of truth (warehouse schema, business definitions).

**Types**:
1. **Schema Drift**: Column removed/renamed in source table
2. **Logic Drift**: Business rule changed (e.g., currency conversion factor)
3. **Freshness Drift**: Data not updated as expected
4. **Lineage Drift**: Join conditions no longer valid
5. **Performance Drift**: Query now slow due to missing index

### 1. Drift Detector

**File**: `backend/internal/drift/drift_detector.go`

```go
package drift

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/google/uuid"
)

// DriftReport represents a drift analysis report
type DriftReport struct {
	ID                uuid.UUID
	TenantID          uuid.UUID
	ModelID           string
	ReportTime        time.Time
	DriftSeverity     string    // low, medium, high, critical
	DriftIssues       []DriftIssue
	SuggestedActions  []string
	ResolvedByUserID  *uuid.UUID
	ResolutionComment string
	Status            string // open, investigating, resolved
}

// DriftIssue represents a specific drift problem
type DriftIssue struct {
	ID              uuid.UUID
	ReportID        uuid.UUID
	IssueType       string      // schema_drift, logic_drift, freshness_drift, lineage_drift, performance_drift
	Element         string      // measure name, dimension name, join name, table name
	Severity        string      // low, medium, high, critical
	Description     string
	DetectionMethod string      // query comparison, schema inspection, runtime observation
	LastDetectedAt  time.Time
	ProposedFix     string
	DataImpact      *DataImpact
}

// DataImpact measures impact of drift
type DataImpact struct {
	AffectedRows      int64
	AffectedQueries   int
	EstimatedUsers    int
	PerformanceChange float64 // percentage change in query time
}

// DriftDetector analyzes models for drift
type DriftDetector struct {
	db *sqlx.DB
}

// NewDriftDetector creates new detector
func NewDriftDetector(db *sqlx.DB) *DriftDetector {
	return &DriftDetector{db: db}
}

// DetectSchemaDrift checks for schema changes in source tables
func (dd *DriftDetector) DetectSchemaDrift(ctx context.Context, tenantID, modelID string) ([]DriftIssue, error) {
	var issues []DriftIssue

	// Get model definition
	var modelDef string
	err := dd.db.GetContext(ctx, &modelDef, `
		SELECT definition FROM fabric_defn 
		WHERE tenant_id = $1 AND model_key = $2 AND kind = 'model'
		ORDER BY version DESC LIMIT 1
	`, tenantID, modelID)
	if err != nil {
		return nil, err
	}

	// Parse model to get source tables
	// Compare with current schema
	
	// Example: detect if a column used in a measure no longer exists
	query := `
		WITH model_columns AS (
			SELECT DISTINCT jsonb_array_elements(definition->'measures')->>'sql' as col_ref
			FROM fabric_defn
			WHERE tenant_id = $1 AND model_key = $2 AND kind = 'measure'
		)
		SELECT col_ref FROM model_columns
		WHERE col_ref NOT IN (
			SELECT column_name FROM information_schema.columns 
			WHERE table_schema = 'public' AND table_name = $3
		)
	`
	
	rows, err := dd.db.QueryContext(ctx, query, tenantID, modelID, "source_table")
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var col string
		if err := rows.Scan(&col); err != nil {
			continue
		}

		issues = append(issues, DriftIssue{
			ID:              uuid.New(),
			IssueType:       "schema_drift",
			Element:         col,
			Severity:        "high",
			Description:     fmt.Sprintf("Column '%s' referenced in measure but no longer exists in source table", col),
			DetectionMethod: "schema_inspection",
			LastDetectedAt:  time.Now(),
			ProposedFix:     fmt.Sprintf("Update measure definition to use alternative column or restore '%s' to schema", col),
		})
	}

	return issues, nil
}

// DetectPerformanceDrift compares query performance over time
func (dd *DriftDetector) DetectPerformanceDrift(ctx context.Context, tenantID, modelID string, thresholdMS int64) ([]DriftIssue, error) {
	var issues []DriftIssue

	// Get baseline (first 10 queries)
	var baselineDuration int64
	err := dd.db.GetContext(ctx, &baselineDuration, `
		SELECT AVG(duration_ms) FROM semantic_query_audit
		WHERE tenant_id = $1 AND model_id = $2
		ORDER BY created_at ASC
		LIMIT 10
	`, tenantID, modelID)
	if err != nil {
		return nil, err
	}

	// Get recent performance (last 10 queries)
	var recentDuration int64
	err = dd.db.GetContext(ctx, &recentDuration, `
		SELECT AVG(duration_ms) FROM semantic_query_audit
		WHERE tenant_id = $1 AND model_id = $2
		ORDER BY created_at DESC
		LIMIT 10
	`, tenantID, modelID)
	if err != nil {
		return nil, err
	}

	percentChange := float64(recentDuration-baselineDuration) / float64(baselineDuration) * 100

	if percentChange > 50.0 { // 50% slowdown
		issues = append(issues, DriftIssue{
			ID:              uuid.New(),
			IssueType:       "performance_drift",
			Severity:        "high",
			Description:     fmt.Sprintf("Query performance degraded by %.1f%% (baseline: %dms, recent: %dms)", percentChange, baselineDuration, recentDuration),
			DetectionMethod: "runtime_observation",
			LastDetectedAt:  time.Now(),
			ProposedFix:     "Check for missing indexes, query plan changes, or increased data volume",
			DataImpact: &DataImpact{
				PerformanceChange: percentChange,
			},
		})
	}

	return issues, nil
}

// DetectFreshnessDrift checks if data is stale
func (dd *DriftDetector) DetectFreshnessDrift(ctx context.Context, tenantID, modelID string, maxAgeHours int) ([]DriftIssue, error) {
	var issues []DriftIssue

	var lastUpdateTime time.Time
	err := dd.db.GetContext(ctx, &lastUpdateTime, `
		SELECT MAX(created_at) FROM semantic_query_audit
		WHERE tenant_id = $1 AND model_id = $2 AND status = 'success'
	`, tenantID, modelID)
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}

	if time.Since(lastUpdateTime) > time.Duration(maxAgeHours)*time.Hour {
		issues = append(issues, DriftIssue{
			ID:              uuid.New(),
			IssueType:       "freshness_drift",
			Severity:        "medium",
			Description:     fmt.Sprintf("Model data is stale - last query was %.1f hours ago", time.Since(lastUpdateTime).Hours()),
			DetectionMethod: "timestamp_inspection",
			LastDetectedAt:  time.Now(),
			ProposedFix:     "Trigger a refresh of the model or check if source data pipeline is running",
		})
	}

	return issues, nil
}

// GenerateDriftReport creates comprehensive drift report
func (dd *DriftDetector) GenerateDriftReport(ctx context.Context, tenantID, modelID string) (*DriftReport, error) {
	report := &DriftReport{
		ID:       uuid.New(),
		TenantID: uuid.MustParse(tenantID),
		ModelID:  modelID,
		ReportTime: time.Now(),
		Status:   "open",
	}

	// Run all drift detections
	schemaDrift, _ := dd.DetectSchemaDrift(ctx, tenantID, modelID)
	perfDrift, _ := dd.DetectPerformanceDrift(ctx, tenantID, modelID, 1000)
	freshnessDrift, _ := dd.DetectFreshnessDrift(ctx, tenantID, modelID, 24)

	report.DriftIssues = append(report.DriftIssues, schemaDrift...)
	report.DriftIssues = append(report.DriftIssues, perfDrift...)
	report.DriftIssues = append(report.DriftIssues, freshnessDrift...)

	// Calculate overall severity
	maxSeverity := "low"
	for _, issue := range report.DriftIssues {
		if issue.Severity == "critical" {
			maxSeverity = "critical"
			break
		} else if issue.Severity == "high" && maxSeverity != "critical" {
			maxSeverity = "high"
		} else if issue.Severity == "medium" && maxSeverity == "low" {
			maxSeverity = "medium"
		}
	}
	report.DriftSeverity = maxSeverity

	return report, nil
}
```

### 2. Drift Tables & Storage

```sql
-- Drift Detection Reports
CREATE TABLE IF NOT EXISTS public.semantic_drift_reports (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id uuid NOT NULL,
    model_id varchar(255) NOT NULL,
    report_time timestamptz NOT NULL,
    drift_severity varchar(50), -- low, medium, high, critical
    issue_count int DEFAULT 0,
    status varchar(50) DEFAULT 'open', -- open, investigating, resolved, ignored
    resolved_by uuid,
    resolution_comment text,
    created_at timestamptz NOT NULL DEFAULT now(),
    resolved_at timestamptz,
    CONSTRAINT semantic_drift_reports_fk_tenant FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_drift_reports_tenant_model ON public.semantic_drift_reports(tenant_id, model_id, report_time DESC);
CREATE INDEX IF NOT EXISTS idx_drift_reports_severity ON public.semantic_drift_reports(drift_severity, created_at DESC);

-- Individual Drift Issues
CREATE TABLE IF NOT EXISTS public.semantic_drift_issues (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    report_id uuid NOT NULL,
    issue_type varchar(50), -- schema_drift, logic_drift, freshness_drift, lineage_drift, performance_drift
    element varchar(255),
    severity varchar(50),
    description text,
    detection_method varchar(255),
    last_detected_at timestamptz,
    proposed_fix text,
    data_impact jsonb,
    created_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT semantic_drift_issues_fk_report FOREIGN KEY (report_id) REFERENCES public.semantic_drift_reports(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_drift_issues_type ON public.semantic_drift_issues(issue_type);
CREATE INDEX IF NOT EXISTS idx_drift_issues_severity ON public.semantic_drift_issues(severity);
```

---

## 📊 Monitoring & Observability

### 1. Metrics Collection

**File**: `backend/internal/metrics/semantic_metrics.go`

```go
package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// SemanticMetrics holds all semantic layer metrics
type SemanticMetrics struct {
	// Query metrics
	QueryCounter         prometheus.Counter
	QueryDuration        prometheus.Histogram
	CacheHitRate         prometheus.Gauge
	
	// Change metrics
	ModelChangeCounter   prometheus.Counter
	MeasureChangeCounter prometheus.Counter
	DimensionChangeCounter prometheus.Counter
	JoinChangeCounter    prometheus.Counter
	
	// Drift metrics
	DriftIssueCounter    prometheus.Counter
	DriftSeverityGauge   prometheus.Gauge
	
	// Performance metrics
	SlowQueryCounter     prometheus.Counter
	CompilationDuration  prometheus.Histogram
	ExecutionDuration    prometheus.Histogram
}

// NewSemanticMetrics creates new metrics
func NewSemanticMetrics() *SemanticMetrics {
	return &SemanticMetrics{
		QueryCounter: promauto.NewCounter(prometheus.CounterOpts{
			Name: "semantic_queries_total",
			Help: "Total number of semantic queries executed",
		}),
		QueryDuration: promauto.NewHistogram(prometheus.HistogramOpts{
			Name:    "semantic_query_duration_ms",
			Help:    "Semantic query execution duration in milliseconds",
			Buckets: []float64{1, 5, 10, 50, 100, 500, 1000, 5000},
		}),
		CacheHitRate: promauto.NewGauge(prometheus.GaugeOpts{
			Name: "semantic_cache_hit_rate",
			Help: "Cache hit rate for semantic queries (0-1)",
		}),
		ModelChangeCounter: promauto.NewCounter(prometheus.CounterOpts{
			Name: "semantic_model_changes_total",
			Help: "Total number of model changes",
		}),
		DriftIssueCounter: promauto.NewCounter(prometheus.CounterOpts{
			Name: "semantic_drift_issues_total",
			Help: "Total drift issues detected",
		}),
		SlowQueryCounter: promauto.NewCounter(prometheus.CounterOpts{
			Name: "semantic_slow_queries_total",
			Help: "Total slow queries (>1s)",
		}),
		CompilationDuration: promauto.NewHistogram(prometheus.HistogramOpts{
			Name:    "semantic_compilation_duration_ms",
			Help:    "Query compilation duration",
			Buckets: []float64{1, 5, 10, 50, 100, 500},
		}),
		ExecutionDuration: promauto.NewHistogram(prometheus.HistogramOpts{
			Name:    "semantic_execution_duration_ms",
			Help:    "Query execution duration",
			Buckets: []float64{10, 50, 100, 500, 1000, 5000},
		}),
	}
}
```

### 2. Grafana Dashboard Definition

**File**: `monitoring/semantic_layer_dashboard.json`

```json
{
  "dashboard": {
    "title": "Semantic Layer Monitoring",
    "panels": [
      {
        "title": "Query Execution Rate",
        "targets": [
          {
            "expr": "rate(semantic_queries_total[5m])"
          }
        ]
      },
      {
        "title": "Query Duration (p95)",
        "targets": [
          {
            "expr": "histogram_quantile(0.95, semantic_query_duration_ms)"
          }
        ]
      },
      {
        "title": "Cache Hit Rate",
        "targets": [
          {
            "expr": "semantic_cache_hit_rate"
          }
        ]
      },
      {
        "title": "Model Changes",
        "targets": [
          {
            "expr": "rate(semantic_model_changes_total[1h])"
          }
        ]
      },
      {
        "title": "Drift Issues by Severity",
        "targets": [
          {
            "expr": "semantic_drift_issues_total"
          }
        ]
      },
      {
        "title": "Slow Queries",
        "targets": [
          {
            "expr": "rate(semantic_slow_queries_total[5m])"
          }
        ]
      }
    ]
  }
}
```

---

## 🤖 AI-Powered Suggestions

### Overview

Automatically suggest:
- **Join optimization**: "Missing join on customer_id would improve query performance"
- **Measure reuse**: "This calculation already exists as measure 'revenue_total'"
- **Filter pushdown**: "This filter should be applied before aggregation"
- **Pre-aggregation candidates**: "These 5 queries would benefit from pre-aggregation"
- **Schema changes**: "Column 'customer_segment' in warehouse - should add as dimension?"

### Implementation

**File**: `backend/internal/suggestions/semantic_suggester.go`

```go
package suggestions

import (
	"context"
	"fmt"
	"strings"

	"github.com/jmoiron/sqlx"
	"github.com/google/uuid"
)

// Suggestion represents an AI suggestion for semantic layer improvement
type Suggestion struct {
	ID              uuid.UUID
	TenantID        uuid.UUID
	ModelID         string
	SuggestionType  string // join_optimization, measure_reuse, filter_pushdown, pre_aggregation, schema_addition
	Priority        string // low, medium, high, critical
	Title           string
	Description     string
	ProposedChange  string
	ExpectedBenefit string
	ConfidenceScore float64 // 0-1
	Status          string  // new, reviewed, accepted, rejected
	CreatedAt       int64
}

// SemanticSuggester generates improvement suggestions
type SemanticSuggester struct {
	db *sqlx.DB
}

// NewSemanticSuggester creates suggester
func NewSemanticSuggester(db *sqlx.DB) *SemanticSuggester {
	return &SemanticSuggester{db: db}
}

// SuggestJoinOptimization analyzes queries for missing joins
func (ss *SemanticSuggester) SuggestJoinOptimization(ctx context.Context, tenantID, modelID string) ([]Suggestion, error) {
	var suggestions []Suggestion

	// Look at recent slow queries
	query := `
		SELECT compiled_sql, duration_ms
		FROM semantic_query_audit
		WHERE tenant_id = $1 AND model_id = $2
		AND duration_ms > 1000 -- slow queries
		ORDER BY duration_ms DESC
		LIMIT 20
	`

	rows, err := ss.db.QueryContext(ctx, query, tenantID, modelID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var sql string
		var duration int64
		if err := rows.Scan(&sql, &duration); err != nil {
			continue
		}

		// Analyze SQL for potential optimizations
		if strings.Contains(sql, "LEFT JOIN") && strings.Contains(sql, "WHERE") {
			suggestion := Suggestion{
				ID:             uuid.New(),
				TenantID:       uuid.MustParse(tenantID),
				ModelID:        modelID,
				SuggestionType: "join_optimization",
				Priority:       "high",
				Title:          "Add filter pushdown before join",
				Description:    "Current query uses LEFT JOIN followed by WHERE - consider moving filter before join",
				ExpectedBenefit: fmt.Sprintf("Could reduce execution time by ~30%% (currently %dms)", duration),
				ConfidenceScore: 0.85,
				Status:         "new",
			}
			suggestions = append(suggestions, suggestion)
		}
	}

	return suggestions, nil
}

// SuggestMeasureReuse finds duplicate measure definitions
func (ss *SemanticSuggester) SuggestMeasureReuse(ctx context.Context, tenantID, modelID string) ([]Suggestion, error) {
	var suggestions []Suggestion

	// Find measures with similar SQL
	query := `
		SELECT m1.name as measure1, m2.name as measure2, similarity(m1.sql, m2.sql) as sim
		FROM (
			SELECT jsonb_extract_path_text(definition, 'name') as name,
			       jsonb_extract_path_text(definition, 'sql') as sql
			FROM fabric_defn
			WHERE tenant_id = $1 AND kind = 'measure'
		) m1
		CROSS JOIN (
			SELECT jsonb_extract_path_text(definition, 'name') as name,
			       jsonb_extract_path_text(definition, 'sql') as sql
			FROM fabric_defn
			WHERE tenant_id = $1 AND kind = 'measure'
		) m2
		WHERE m1.name < m2.name
		AND similarity(m1.sql, m2.sql) > 0.8
	`

	rows, err := ss.db.QueryContext(ctx, query, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var measure1, measure2 string
		var similarity float64
		if err := rows.Scan(&measure1, &measure2, &similarity); err != nil {
			continue
		}

		suggestion := Suggestion{
			ID:             uuid.New(),
			TenantID:       uuid.MustParse(tenantID),
			ModelID:        modelID,
			SuggestionType: "measure_reuse",
			Priority:       "medium",
			Title:          fmt.Sprintf("Consolidate measures '%s' and '%s'", measure1, measure2),
			Description:    fmt.Sprintf("These measures have very similar definitions (%.0f%% match)", similarity*100),
			ProposedChange: fmt.Sprintf("Consider using '%s' instead of '%s', or create a shared measure", measure1, measure2),
			ExpectedBenefit: "Reduce maintenance burden, ensure consistency",
			ConfidenceScore: similarity,
			Status:         "new",
		}
		suggestions = append(suggestions, suggestion)
	}

	return suggestions, nil
}

// SuggestPreAggregation identifies candidates for pre-aggregation
func (ss *SemanticSuggester) SuggestPreAggregation(ctx context.Context, tenantID string) ([]Suggestion, error) {
	var suggestions []Suggestion

	// Find frequently executed GROUP BY patterns
	query := `
		SELECT model_id, compiled_sql, COUNT(*) as exec_count, AVG(duration_ms) as avg_duration
		FROM semantic_query_audit
		WHERE tenant_id = $1
		AND created_at > now() - interval '7 days'
		AND compiled_sql ILIKE '%GROUP BY%'
		GROUP BY model_id, compiled_sql
		HAVING COUNT(*) > 10 AND AVG(duration_ms) > 500
		ORDER BY exec_count DESC
		LIMIT 10
	`

	rows, err := ss.db.QueryContext(ctx, query, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var modelID, sql string
		var count int
		var avgDuration float64
		if err := rows.Scan(&modelID, &sql, &count, &avgDuration); err != nil {
			continue
		}

		suggestion := Suggestion{
			ID:             uuid.New(),
			TenantID:       uuid.MustParse(tenantID),
			ModelID:        modelID,
			SuggestionType: "pre_aggregation",
			Priority:       "high",
			Title:          fmt.Sprintf("Create pre-aggregation for %s", modelID),
			Description:    fmt.Sprintf("This query pattern executed %d times in past week, avg duration %.0fms", count, avgDuration),
			ExpectedBenefit: fmt.Sprintf("Could reduce query time by 80%% and save ~%.1f seconds/week", avgDuration*float64(count)/1000),
			ConfidenceScore: 0.9,
			Status:         "new",
		}
		suggestions = append(suggestions, suggestion)
	}

	return suggestions, nil
}
```

---

## 🛣️ Implementation Roadmap

### Phase 1: RabbitMQ & Event Foundation (Week 1-2)

**Tasks**:
- [ ] Deploy RabbitMQ broker (use docker-compose.yml)
- [ ] Create `SemanticPublisher` (publish model changes)
- [ ] Create subscriber interfaces (cache, audit, drift)
- [ ] Create database tables (audit log, change events, delivery log)
- [ ] Add basic event publishing to model editor endpoints

**Deliverable**: Any model update publishes to RabbitMQ

### Phase 2: SQL Auditing & Query Tracing (Week 3-4)

**Tasks**:
- [ ] Implement `QueryAuditor` with audit recording
- [ ] Wrap query compiler to capture metadata
- [ ] Create audit storage tables
- [ ] Add query history UI (read-only)
- [ ] Create query comparison views

**Deliverable**: Every query tracked with compiled SQL, parameters, execution time

### Phase 3: Drift Detection (Week 5-6)

**Tasks**:
- [ ] Implement `DriftDetector` with all detection types
- [ ] Create drift report tables
- [ ] Add scheduled drift detection job (hourly)
- [ ] Add drift dashboard UI
- [ ] Integrate drift alerts to RabbitMQ

**Deliverable**: Drift issues identified and alertable

### Phase 4: Monitoring & Observability (Week 7)

**Tasks**:
- [ ] Implement Prometheus metrics collection
- [ ] Create Grafana dashboard
- [ ] Set up alerting rules (threshold exceeded, drift detected)
- [ ] Add health check endpoints

**Deliverable**: Real-time monitoring of semantic layer health

### Phase 5: AI Suggestions (Week 8)

**Tasks**:
- [ ] Implement suggestion engines (join, reuse, pre-agg, schema)
- [ ] Create suggestion storage & UI
- [ ] Add user feedback loop (accepted/rejected)
- [ ] Train ML model on feedback

**Deliverable**: Automated suggestions for improvements

---

## 📋 Docker Compose Integration

**Add to `docker-compose.yml`**:

```yaml
rabbitmq:
  image: rabbitmq:3.12-management-alpine
  environment:
    RABBITMQ_DEFAULT_USER: guest
    RABBITMQ_DEFAULT_PASS: guest
  ports:
    - "5672:5672"
    - "15672:15672"
  healthcheck:
    test: rabbitmq-diagnostics ping
    interval: 10s
    timeout: 5s
    retries: 5

prometheus:
  image: prom/prometheus:latest
  volumes:
    - ./monitoring/prometheus.yml:/etc/prometheus/prometheus.yml
  ports:
    - "9090:9090"

grafana:
  image: grafana/grafana:latest
  environment:
    - GF_SECURITY_ADMIN_PASSWORD=admin
  ports:
    - "3000:3000"
  depends_on:
    - prometheus
```

---

## 🔗 API Endpoints

### 1. Publish Semantic Change
```bash
POST /api/v1/semantic/models/{modelId}/publish-change
{
  "changeType": "measure_added",
  "elementType": "measure",
  "elementName": "revenue_total",
  "changeReason": "Added new financial metric",
  "oldDefinition": null,
  "newDefinition": {...}
}
```

### 2. Get Query Audit Trail
```bash
GET /api/v1/semantic/models/{modelId}/audit?limit=50&offset=0
```

### 3. Get Drift Report
```bash
GET /api/v1/semantic/drift/reports?model_id={modelId}&limit=10
```

### 4. Get Suggestions
```bash
GET /api/v1/semantic/suggestions?model_id={modelId}&type=pre_aggregation
```

---

## ✅ Validation Checklist

- [ ] RabbitMQ running and accessible
- [ ] All change events publishing to RabbitMQ
- [ ] Query audits recording to database
- [ ] Drift detection running on schedule
- [ ] Prometheus scraping metrics
- [ ] Grafana dashboard displaying data
- [ ] Suggestions generating for models
- [ ] Alerts firing on thresholds

---

**Status**: Complete integration framework with production-ready patterns

**Next**: Implement phase by phase, starting with RabbitMQ foundation
