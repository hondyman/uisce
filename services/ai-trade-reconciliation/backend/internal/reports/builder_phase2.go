package reports

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
)

// ============================================================================
// TRANSACTION SUPPORT - Atomic multi-step operations
// ============================================================================

// WithTx wraps a function in a database transaction
// If the function returns an error, the transaction is rolled back
// Otherwise, the transaction is committed
func (rb *ReportBuilder) WithTx(ctx context.Context, fn func(*sql.Tx) error) error {
	tx, err := rb.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	if err := fn(tx); err != nil {
		if rollbackErr := tx.Rollback(); rollbackErr != nil {
			return fmt.Errorf("transaction failed: %w, rollback also failed: %w", err, rollbackErr)
		}
		return fmt.Errorf("transaction failed: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// SaveReportTemplateWithTx saves a report template within a transaction for atomicity
func (rb *ReportBuilder) SaveReportTemplateWithTx(ctx context.Context, template *ReportTemplate) error {
	return rb.WithTx(ctx, func(tx *sql.Tx) error {
		return rb.saveTemplateInTx(tx, template)
	})
}

// saveTemplateInTx is the internal method that uses a provided transaction
func (rb *ReportBuilder) saveTemplateInTx(tx *sql.Tx, template *ReportTemplate) error {
	if template == nil {
		return fmt.Errorf("template cannot be nil")
	}
	if template.ID == uuid.Nil {
		return fmt.Errorf("template ID is required")
	}

	template.UpdatedAt = time.Now()

	sectionsJSON, err := json.Marshal(template.Sections)
	if err != nil {
		return fmt.Errorf("failed to marshal sections: %w", err)
	}
	filtersJSON, err := json.Marshal(template.Filters)
	if err != nil {
		return fmt.Errorf("failed to marshal filters: %w", err)
	}
	rulesJSON, err := json.Marshal(template.Rules)
	if err != nil {
		return fmt.Errorf("failed to marshal rules: %w", err)
	}

	// TODO: Refactor to Hasura GraphQL with transaction support
	// mutation { update_report_templates_by_pk(
	//   pk_columns: {id: $id}
	//   _set: {sections: $sections, filters: $filters, rules: $rules, updated_at: $updated_at}
	// ) { id }}
	// JSONB fields: sections, filters, rules
	// Note: Ensure Hasura transaction support or use optimistic locking pattern
	_, err = tx.ExecContext(context.Background(), `
		UPDATE report_templates 
		SET sections = $1, filters = $2, rules = $3, updated_at = $4
		WHERE id = $5
	`, string(sectionsJSON), string(filtersJSON), string(rulesJSON), template.UpdatedAt, template.ID)

	if err != nil {
		return fmt.Errorf("failed to save template: %w", err)
	}

	return nil
}

// ============================================================================
// CACHING LAYER - Reduce database load with TTL cache
// ============================================================================

// CacheEntry represents a cached item with expiration time
type CacheEntry struct {
	Data      interface{}
	ExpiresAt time.Time
}

// TemplateCache provides in-memory caching for report templates
type TemplateCache struct {
	mu    sync.RWMutex
	cache map[string]*CacheEntry
	ttl   time.Duration
}

// NewTemplateCache creates a new template cache with TTL
func NewTemplateCache(ttl time.Duration) *TemplateCache {
	cache := &TemplateCache{
		cache: make(map[string]*CacheEntry),
		ttl:   ttl,
	}

	// Start cleanup goroutine
	go cache.cleanupExpired()

	return cache
}

// Set stores a value in the cache
func (tc *TemplateCache) Set(key string, value interface{}) {
	tc.mu.Lock()
	defer tc.mu.Unlock()

	tc.cache[key] = &CacheEntry{
		Data:      value,
		ExpiresAt: time.Now().Add(tc.ttl),
	}
}

// Get retrieves a value from the cache
// Returns nil if not found or expired
func (tc *TemplateCache) Get(key string) interface{} {
	tc.mu.RLock()
	defer tc.mu.RUnlock()

	entry, exists := tc.cache[key]
	if !exists {
		return nil
	}

	// Check if expired
	if time.Now().After(entry.ExpiresAt) {
		return nil
	}

	return entry.Data
}

// Delete removes a key from the cache
func (tc *TemplateCache) Delete(key string) {
	tc.mu.Lock()
	defer tc.mu.Unlock()

	delete(tc.cache, key)
}

// Clear removes all entries from the cache
func (tc *TemplateCache) Clear() {
	tc.mu.Lock()
	defer tc.mu.Unlock()

	tc.cache = make(map[string]*CacheEntry)
}

// cleanupExpired periodically removes expired entries
func (tc *TemplateCache) cleanupExpired() {
	ticker := time.NewTicker(tc.ttl / 2)
	defer ticker.Stop()

	for range ticker.C {
		tc.mu.Lock()
		now := time.Now()
		for key, entry := range tc.cache {
			if now.After(entry.ExpiresAt) {
				delete(tc.cache, key)
			}
		}
		tc.mu.Unlock()
	}
}

// ============================================================================
// AUDIT LOGGING - Track all changes for compliance
// ============================================================================

// AuditLog represents a single audit log entry
type AuditLog struct {
	ID         string                 `json:"id"`
	TenantID   string                 `json:"tenant_id"`
	UserID     string                 `json:"user_id"`
	Action     string                 `json:"action"` // create, update, delete
	EntityType string                 `json:"entity_type"`
	EntityID   string                 `json:"entity_id"`
	OldValue   map[string]interface{} `json:"old_value"`
	NewValue   map[string]interface{} `json:"new_value"`
	Reason     string                 `json:"reason"` // Why was this change made
	Timestamp  time.Time              `json:"timestamp"`
	IPAddress  string                 `json:"ip_address"`
	UserAgent  string                 `json:"user_agent"`
}

// AuditLogger logs all changes to a persistent store
type AuditLogger struct {
	db    *sql.DB
	queue chan *AuditLog
	done  chan struct{}
}

// NewAuditLogger creates a new audit logger with async logging
func NewAuditLogger(db *sql.DB, queueSize int) *AuditLogger {
	al := &AuditLogger{
		db:    db,
		queue: make(chan *AuditLog, queueSize),
		done:  make(chan struct{}),
	}

	// Start async logging worker
	go al.worker()

	return al
}

// Log logs a change asynchronously
func (al *AuditLogger) Log(entry *AuditLog) {
	select {
	case al.queue <- entry:
		// Queued
	case <-al.done:
		// Logger is closed
	}
}

// LogSync logs a change synchronously (blocks until written)
func (al *AuditLogger) LogSync(ctx context.Context, entry *AuditLog) error {
	if entry.ID == "" {
		entry.ID = uuid.New().String()
	}
	if entry.Timestamp.IsZero() {
		entry.Timestamp = time.Now()
	}

	// TODO: Refactor to Hasura GraphQL
	// mutation { insert_audit_logs_one(object: {
	//   id, tenant_id, user_id, action, entity_type, entity_id,
	//   old_value, new_value, reason, timestamp, ip_address, user_agent
	// }) { id }}
	// JSONB fields: old_value, new_value (nullable)
	_, err := al.db.ExecContext(ctx, `
		INSERT INTO audit_logs (id, tenant_id, user_id, action, entity_type, entity_id, 
		                         old_value, new_value, reason, timestamp, ip_address, user_agent)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
	`,
		entry.ID, entry.TenantID, entry.UserID, entry.Action, entry.EntityType, entry.EntityID,
		jsonOrNull(entry.OldValue), jsonOrNull(entry.NewValue), entry.Reason, entry.Timestamp,
		entry.IPAddress, entry.UserAgent)

	if err != nil {
		return fmt.Errorf("failed to log audit entry: %w", err)
	}

	return nil
}

// worker processes queued audit logs
func (al *AuditLogger) worker() {
	ctx := context.Background()
	for {
		select {
		case entry := <-al.queue:
			if entry == nil {
				return
			}
			if err := al.LogSync(ctx, entry); err != nil {
				// Log error but don't block
				fmt.Printf("audit log error: %v\n", err)
			}
		case <-al.done:
			return
		}
	}
}

// Close stops the audit logger
func (al *AuditLogger) Close() error {
	close(al.done)
	// Wait for remaining entries to be logged
	time.Sleep(100 * time.Millisecond)
	return nil
}

// ============================================================================
// PERFORMANCE METRICS - Track operation timing and database queries
// ============================================================================

// Metrics tracks operation performance
type Metrics struct {
	mu sync.RWMutex

	// Counters
	QueryCount         int64
	TemplatesSaved     int64
	TemplatesLoaded    int64
	EntitiesDropped    int64
	DropActionsHandled int64

	// Timing (in milliseconds)
	TotalQueryTime         int64
	TotalSaveTime          int64
	TotalLoadTime          int64
	TotalDropTime          int64
	AverageCacheLookupTime int64

	// Cache stats
	CacheHits      int64
	CacheMisses    int64
	CacheEvictions int64
}

// MetricsSnapshot is a thread-safe copy of metrics (no mutex)
type MetricsSnapshot struct {
	QueryCount         int64
	TemplatesSaved     int64
	TemplatesLoaded    int64
	EntitiesDropped    int64
	DropActionsHandled int64
	TotalQueryTime     int64
	TotalSaveTime      int64
	TotalLoadTime      int64
	TotalDropTime      int64
	CacheHits          int64
	CacheMisses        int64
	CacheEvictions     int64
}

// Timer helps measure operation duration
type Timer struct {
	start time.Time
}

// NewTimer creates a new timer
func NewTimer() *Timer {
	return &Timer{start: time.Now()}
}

// Elapsed returns milliseconds since creation
func (t *Timer) Elapsed() int64 {
	return time.Since(t.start).Milliseconds()
}

// MetricsCollector collects and stores metrics
type MetricsCollector struct {
	metrics *Metrics
}

// NewMetricsCollector creates a new metrics collector
func NewMetricsCollector() *MetricsCollector {
	return &MetricsCollector{
		metrics: &Metrics{},
	}
}

// RecordQuery records a query execution
func (mc *MetricsCollector) RecordQuery(durationMs int64) {
	mc.metrics.mu.Lock()
	defer mc.metrics.mu.Unlock()

	mc.metrics.QueryCount++
	mc.metrics.TotalQueryTime += durationMs
}

// RecordTemplateSave records a template save operation
func (mc *MetricsCollector) RecordTemplateSave(durationMs int64) {
	mc.metrics.mu.Lock()
	defer mc.metrics.mu.Unlock()

	mc.metrics.TemplatesSaved++
	mc.metrics.TotalSaveTime += durationMs
}

// RecordTemplateLoad records a template load operation
func (mc *MetricsCollector) RecordTemplateLoad(durationMs int64) {
	mc.metrics.mu.Lock()
	defer mc.metrics.mu.Unlock()

	mc.metrics.TemplatesLoaded++
	mc.metrics.TotalLoadTime += durationMs
}

// RecordDrop records a drag-drop operation
func (mc *MetricsCollector) RecordDrop(durationMs int64) {
	mc.metrics.mu.Lock()
	defer mc.metrics.mu.Unlock()

	mc.metrics.EntitiesDropped++
	mc.metrics.TotalDropTime += durationMs
}

// RecordCacheHit records a cache hit
func (mc *MetricsCollector) RecordCacheHit() {
	mc.metrics.mu.Lock()
	defer mc.metrics.mu.Unlock()

	mc.metrics.CacheHits++
}

// RecordCacheMiss records a cache miss
func (mc *MetricsCollector) RecordCacheMiss() {
	mc.metrics.mu.Lock()
	defer mc.metrics.mu.Unlock()

	mc.metrics.CacheMisses++
}

// GetMetrics returns a copy of current metrics (without mutex)
func (mc *MetricsCollector) GetMetrics() MetricsSnapshot {
	mc.metrics.mu.RLock()
	defer mc.metrics.mu.RUnlock()

	return MetricsSnapshot{
		QueryCount:         mc.metrics.QueryCount,
		TemplatesSaved:     mc.metrics.TemplatesSaved,
		TemplatesLoaded:    mc.metrics.TemplatesLoaded,
		EntitiesDropped:    mc.metrics.EntitiesDropped,
		DropActionsHandled: mc.metrics.DropActionsHandled,
		TotalQueryTime:     mc.metrics.TotalQueryTime,
		TotalSaveTime:      mc.metrics.TotalSaveTime,
		TotalLoadTime:      mc.metrics.TotalLoadTime,
		TotalDropTime:      mc.metrics.TotalDropTime,
		CacheHits:          mc.metrics.CacheHits,
		CacheMisses:        mc.metrics.CacheMisses,
		CacheEvictions:     mc.metrics.CacheEvictions,
	}
}

// AverageQueryTime returns average query execution time in milliseconds
func (mc *MetricsCollector) AverageQueryTime() float64 {
	mc.metrics.mu.RLock()
	defer mc.metrics.mu.RUnlock()

	if mc.metrics.QueryCount == 0 {
		return 0
	}
	return float64(mc.metrics.TotalQueryTime) / float64(mc.metrics.QueryCount)
}

// AverageSaveTime returns average save time in milliseconds
func (mc *MetricsCollector) AverageSaveTime() float64 {
	mc.metrics.mu.RLock()
	defer mc.metrics.mu.RUnlock()

	if mc.metrics.TemplatesSaved == 0 {
		return 0
	}
	return float64(mc.metrics.TotalSaveTime) / float64(mc.metrics.TemplatesSaved)
}

// CacheHitRate returns cache hit rate as percentage
func (mc *MetricsCollector) CacheHitRate() float64 {
	mc.metrics.mu.RLock()
	defer mc.metrics.mu.RUnlock()

	total := mc.metrics.CacheHits + mc.metrics.CacheMisses
	if total == 0 {
		return 0
	}
	return (float64(mc.metrics.CacheHits) / float64(total)) * 100
}

// ============================================================================
// BATCH OPERATIONS - Handle multiple drops in single operation
// ============================================================================

// BatchDropRequest represents a batch of drop operations
type BatchDropRequest struct {
	TemplateID string
	Drops      []DragDropState
}

// BatchDropResult represents the result of batch operations
type BatchDropResult struct {
	Successful int
	Failed     int
	Errors     map[int]error // Maps drop index to error
	Duration   time.Duration
}

// DropEntitiesBatch handles multiple drops in a single atomic operation
func (rb *ReportBuilder) DropEntitiesBatch(ctx context.Context, request BatchDropRequest) (*BatchDropResult, error) {
	result := &BatchDropResult{
		Errors: make(map[int]error),
	}
	timer := NewTimer()

	if err := ValidateUUID(request.TemplateID); err != nil {
		return result, fmt.Errorf("invalid template ID: %w", err)
	}

	if len(request.Drops) == 0 {
		return result, fmt.Errorf("no drops provided")
	}

	// Get template once for batch operation
	template, err := rb.GetReportTemplate(ctx, request.TemplateID)
	if err != nil {
		return result, fmt.Errorf("failed to get template: %w", err)
	}

	// Process all drops within a transaction
	err = rb.WithTx(ctx, func(tx *sql.Tx) error {
		for i, dropState := range request.Drops {
			// Validate each drop state
			if err := ValidateDragDropState(&dropState); err != nil {
				result.Errors[i] = err
				result.Failed++
				continue
			}

			// Find section
			sectionIndex, err := FindSectionByID(template.Sections, dropState.TargetSectionID)
			if err != nil {
				result.Errors[i] = err
				result.Failed++
				continue
			}

			// Execute drop action
			switch dropState.Action {
			case "add_to_table":
				template.Sections[sectionIndex].DroppedEntities = append(
					template.Sections[sectionIndex].DroppedEntities,
					DragDropEntity{
						EntityID:      dropState.SourceEntity.EntityID,
						EntityName:    dropState.SourceEntity.EntityName,
						EntityType:    dropState.SourceEntity.EntityType,
						DataType:      dropState.SourceEntity.DataType,
						DisplayFormat: "raw",
						ColumnWidth:   200,
					},
				)
			case "create_filter":
				template.Filters = append(template.Filters, ReportFilter{
					ID:              uuid.New(),
					FilterType:      GetDefaultFilterType(dropState.SourceEntity.DataType),
					EntityID:        dropState.SourceEntity.EntityID,
					EntityName:      dropState.SourceEntity.EntityName,
					ApplyToSections: []string{dropState.TargetSectionID},
					DroppedFrom:     "drag_drop",
					Operator:        "and",
				})
			case "create_aggregation":
				template.Sections[sectionIndex].AggregationFields = append(
					template.Sections[sectionIndex].AggregationFields,
					AggregationField{
						FieldName:       dropState.SourceEntity.EntityName,
						AggregationType: GetDefaultAggregation(dropState.SourceEntity.EntityType),
						DisplayName:     dropState.SourceEntity.EntityName,
					},
				)
			case "create_rule":
				template.Rules = append(template.Rules, ReportRule{
					ID:               uuid.New(),
					Name:             fmt.Sprintf("Rule for %s", dropState.SourceEntity.EntityName),
					Description:      fmt.Sprintf("Auto-generated rule from %s", dropState.SourceEntity.EntityName),
					EntitiesInvolved: []string{dropState.SourceEntity.EntityID},
					CreatedFrom: []DragDropEntity{
						{
							EntityID:   dropState.SourceEntity.EntityID,
							EntityName: dropState.SourceEntity.EntityName,
							EntityType: dropState.SourceEntity.EntityType,
						},
					},
					IsActive: true,
				})
			}

			result.Successful++
		}

		// Save all changes in single transaction
		return rb.saveTemplateInTx(tx, template)
	})

	result.Duration = time.Duration(timer.Elapsed()) * time.Millisecond

	if err != nil {
		return result, fmt.Errorf("batch operation failed: %w", err)
	}

	return result, nil
}

// ============================================================================
// HELPER FUNCTIONS
// ============================================================================

// jsonOrNull marshals data to JSON or returns NULL for database
func jsonOrNull(data map[string]interface{}) interface{} {
	if data == nil || len(data) == 0 {
		return nil
	}
	b, err := json.Marshal(data)
	if err != nil {
		return nil
	}
	return string(b)
}
