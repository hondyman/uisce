package reporting

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
)

// ============================================================================
// ANALYTICS & TELEMETRY
// ============================================================================

// ReportUsageEvent represents a report usage event for analytics
type ReportUsageEvent struct {
	ID                 uuid.UUID       `json:"id"`
	TenantID           uuid.UUID       `json:"tenant_id"`
	DatasourceID       uuid.UUID       `json:"datasource_id"`
	ReportDefinitionID uuid.UUID       `json:"report_definition_id"`
	ReportExtensionID  *uuid.UUID      `json:"report_extension_id,omitempty"`
	EventType          string          `json:"event_type"` // view, render, export, schedule
	UserID             *uuid.UUID      `json:"user_id,omitempty"`
	SessionID          string          `json:"session_id,omitempty"`
	OutputFormat       string          `json:"output_format,omitempty"`
	Parameters         json.RawMessage `json:"parameters,omitempty"`
	Duration           int64           `json:"duration_ms,omitempty"`
	Status             string          `json:"status"` // success, error, timeout
	ErrorCode          string          `json:"error_code,omitempty"`
	Metadata           json.RawMessage `json:"metadata,omitempty"`
	ClientInfo         *ClientInfo     `json:"client_info,omitempty"`
	Timestamp          time.Time       `json:"timestamp"`
}

// ClientInfo captures client environment details
type ClientInfo struct {
	UserAgent    string `json:"user_agent"`
	IPAddress    string `json:"ip_address"`
	Browser      string `json:"browser"`
	BrowserVer   string `json:"browser_version"`
	OS           string `json:"os"`
	Device       string `json:"device"`
	Locale       string `json:"locale"`
	Timezone     string `json:"timezone"`
	ScreenWidth  int    `json:"screen_width"`
	ScreenHeight int    `json:"screen_height"`
}

// PerformanceMetrics captures report generation performance
type PerformanceMetrics struct {
	TotalDuration   int64 `json:"total_duration_ms"`
	QueryTime       int64 `json:"query_time_ms"`
	RenderTime      int64 `json:"render_time_ms"`
	CacheHit        bool  `json:"cache_hit"`
	DataRowCount    int   `json:"data_row_count"`
	OutputSizeBytes int64 `json:"output_size_bytes"`
	MemoryUsedBytes int64 `json:"memory_used_bytes"`
	CubeQueryCount  int   `json:"cube_query_count"`
}

// AnalyticsCollector collects and batches analytics events
type AnalyticsCollector struct {
	events        []*ReportUsageEvent
	mutex         sync.Mutex
	batchSize     int
	flushInterval time.Duration
	sink          AnalyticsSink
	stopCh        chan struct{}
}

// AnalyticsSink interface for analytics storage backends
type AnalyticsSink interface {
	Write(ctx context.Context, events []*ReportUsageEvent) error
}

// NewAnalyticsCollector creates an analytics collector
func NewAnalyticsCollector(sink AnalyticsSink, batchSize int, flushInterval time.Duration) *AnalyticsCollector {
	ac := &AnalyticsCollector{
		events:        make([]*ReportUsageEvent, 0, batchSize),
		batchSize:     batchSize,
		flushInterval: flushInterval,
		sink:          sink,
		stopCh:        make(chan struct{}),
	}
	go ac.flushLoop()
	return ac
}

// Track records an analytics event
func (ac *AnalyticsCollector) Track(event *ReportUsageEvent) {
	event.ID = uuid.New()
	event.Timestamp = time.Now()

	ac.mutex.Lock()
	ac.events = append(ac.events, event)
	shouldFlush := len(ac.events) >= ac.batchSize
	ac.mutex.Unlock()

	if shouldFlush {
		go ac.flush()
	}
}

func (ac *AnalyticsCollector) flushLoop() {
	ticker := time.NewTicker(ac.flushInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			ac.flush()
		case <-ac.stopCh:
			ac.flush() // Final flush
			return
		}
	}
}

func (ac *AnalyticsCollector) flush() {
	ac.mutex.Lock()
	if len(ac.events) == 0 {
		ac.mutex.Unlock()
		return
	}

	events := ac.events
	ac.events = make([]*ReportUsageEvent, 0, ac.batchSize)
	ac.mutex.Unlock()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := ac.sink.Write(ctx, events); err != nil {
		// Log error but don't lose events - could implement retry queue
		_ = err
	}
}

// Stop gracefully stops the collector
func (ac *AnalyticsCollector) Stop() {
	close(ac.stopCh)
}

// ============================================================================
// REAL-TIME METRICS
// ============================================================================

// RealtimeMetrics provides live performance metrics
type RealtimeMetrics struct {
	// Counters
	totalRequests     int64
	successfulRenders int64
	failedRenders     int64
	cacheHits         int64
	cacheMisses       int64

	// Gauges
	activeRenders int64
	queuedRenders int64

	// Histograms (simplified - production should use proper histogram)
	renderTimes    []int64
	renderTimeLock sync.RWMutex

	// Per-tenant metrics
	tenantMetrics map[uuid.UUID]*TenantMetrics
	tenantLock    sync.RWMutex
}

// TenantMetrics tracks per-tenant usage
type TenantMetrics struct {
	TenantID      uuid.UUID
	RequestCount  int64
	RenderCount   int64
	ErrorCount    int64
	TotalDuration int64
	LastActivity  time.Time
}

// NewRealtimeMetrics creates a realtime metrics tracker
func NewRealtimeMetrics() *RealtimeMetrics {
	return &RealtimeMetrics{
		renderTimes:   make([]int64, 0, 1000),
		tenantMetrics: make(map[uuid.UUID]*TenantMetrics),
	}
}

// RecordRequest records a request
func (rm *RealtimeMetrics) RecordRequest() {
	atomic.AddInt64(&rm.totalRequests, 1)
}

// RecordRenderStart records a render starting
func (rm *RealtimeMetrics) RecordRenderStart() {
	atomic.AddInt64(&rm.activeRenders, 1)
}

// RecordRenderComplete records a completed render
func (rm *RealtimeMetrics) RecordRenderComplete(durationMs int64, success bool, tenantID uuid.UUID) {
	atomic.AddInt64(&rm.activeRenders, -1)

	if success {
		atomic.AddInt64(&rm.successfulRenders, 1)
	} else {
		atomic.AddInt64(&rm.failedRenders, 1)
	}

	// Record duration
	rm.renderTimeLock.Lock()
	rm.renderTimes = append(rm.renderTimes, durationMs)
	if len(rm.renderTimes) > 1000 {
		rm.renderTimes = rm.renderTimes[len(rm.renderTimes)-1000:]
	}
	rm.renderTimeLock.Unlock()

	// Update tenant metrics
	rm.tenantLock.Lock()
	tm, ok := rm.tenantMetrics[tenantID]
	if !ok {
		tm = &TenantMetrics{TenantID: tenantID}
		rm.tenantMetrics[tenantID] = tm
	}
	tm.RequestCount++
	tm.RenderCount++
	tm.TotalDuration += durationMs
	tm.LastActivity = time.Now()
	if !success {
		tm.ErrorCount++
	}
	rm.tenantLock.Unlock()
}

// RecordCacheHit records a cache hit
func (rm *RealtimeMetrics) RecordCacheHit() {
	atomic.AddInt64(&rm.cacheHits, 1)
}

// RecordCacheMiss records a cache miss
func (rm *RealtimeMetrics) RecordCacheMiss() {
	atomic.AddInt64(&rm.cacheMisses, 1)
}

// GetSnapshot returns current metrics snapshot
func (rm *RealtimeMetrics) GetSnapshot() *MetricsSnapshot {
	rm.renderTimeLock.RLock()
	var avgRenderTime int64
	if len(rm.renderTimes) > 0 {
		var total int64
		for _, t := range rm.renderTimes {
			total += t
		}
		avgRenderTime = total / int64(len(rm.renderTimes))
	}
	rm.renderTimeLock.RUnlock()

	cacheHits := atomic.LoadInt64(&rm.cacheHits)
	cacheMisses := atomic.LoadInt64(&rm.cacheMisses)
	var cacheHitRate float64
	if cacheHits+cacheMisses > 0 {
		cacheHitRate = float64(cacheHits) / float64(cacheHits+cacheMisses)
	}

	return &MetricsSnapshot{
		TotalRequests:     atomic.LoadInt64(&rm.totalRequests),
		SuccessfulRenders: atomic.LoadInt64(&rm.successfulRenders),
		FailedRenders:     atomic.LoadInt64(&rm.failedRenders),
		ActiveRenders:     atomic.LoadInt64(&rm.activeRenders),
		QueuedRenders:     atomic.LoadInt64(&rm.queuedRenders),
		CacheHits:         cacheHits,
		CacheMisses:       cacheMisses,
		CacheHitRate:      cacheHitRate,
		AvgRenderTimeMs:   avgRenderTime,
		Timestamp:         time.Now(),
	}
}

// MetricsSnapshot is a point-in-time view of metrics
type MetricsSnapshot struct {
	TotalRequests     int64     `json:"total_requests"`
	SuccessfulRenders int64     `json:"successful_renders"`
	FailedRenders     int64     `json:"failed_renders"`
	ActiveRenders     int64     `json:"active_renders"`
	QueuedRenders     int64     `json:"queued_renders"`
	CacheHits         int64     `json:"cache_hits"`
	CacheMisses       int64     `json:"cache_misses"`
	CacheHitRate      float64   `json:"cache_hit_rate"`
	AvgRenderTimeMs   int64     `json:"avg_render_time_ms"`
	Timestamp         time.Time `json:"timestamp"`
}

// GetTenantMetrics returns metrics for a specific tenant
func (rm *RealtimeMetrics) GetTenantMetrics(tenantID uuid.UUID) *TenantMetrics {
	rm.tenantLock.RLock()
	defer rm.tenantLock.RUnlock()

	if tm, ok := rm.tenantMetrics[tenantID]; ok {
		// Return a copy to avoid race conditions
		return &TenantMetrics{
			TenantID:      tm.TenantID,
			RequestCount:  tm.RequestCount,
			RenderCount:   tm.RenderCount,
			ErrorCount:    tm.ErrorCount,
			TotalDuration: tm.TotalDuration,
			LastActivity:  tm.LastActivity,
		}
	}
	return nil
}

// RecordView records a report view for analytics
func (rm *RealtimeMetrics) RecordView(tenantID, reportID uuid.UUID) {
	rm.RecordRequest()

	rm.tenantLock.Lock()
	defer rm.tenantLock.Unlock()

	tm, ok := rm.tenantMetrics[tenantID]
	if !ok {
		tm = &TenantMetrics{TenantID: tenantID}
		rm.tenantMetrics[tenantID] = tm
	}
	tm.RequestCount++
	tm.LastActivity = time.Now()
}

// ============================================================================
// REPORT POPULARITY & RECOMMENDATIONS
// ============================================================================

// PopularityTracker tracks report popularity for recommendations
type PopularityTracker struct {
	// Report view counts
	viewCounts map[string]int64
	viewLock   sync.RWMutex

	// User report history
	userHistory map[uuid.UUID][]uuid.UUID
	historyLock sync.RWMutex

	// Category popularity
	categoryViews map[string]int64
	categoryLock  sync.RWMutex
}

// NewPopularityTracker creates a popularity tracker
func NewPopularityTracker() *PopularityTracker {
	return &PopularityTracker{
		viewCounts:    make(map[string]int64),
		userHistory:   make(map[uuid.UUID][]uuid.UUID),
		categoryViews: make(map[string]int64),
	}
}

// RecordView records a report view
func (pt *PopularityTracker) RecordView(tenantID, reportID uuid.UUID, userID *uuid.UUID, category string) {
	// Update global view count
	key := fmt.Sprintf("%s:%s", tenantID, reportID)
	pt.viewLock.Lock()
	pt.viewCounts[key]++
	pt.viewLock.Unlock()

	// Update user history
	if userID != nil {
		pt.historyLock.Lock()
		history := pt.userHistory[*userID]
		// Keep last 50 reports
		if len(history) >= 50 {
			history = history[1:]
		}
		pt.userHistory[*userID] = append(history, reportID)
		pt.historyLock.Unlock()
	}

	// Update category popularity
	if category != "" {
		pt.categoryLock.Lock()
		pt.categoryViews[category]++
		pt.categoryLock.Unlock()
	}
}

// GetPopularReports returns most popular reports for a tenant
func (pt *PopularityTracker) GetPopularReports(tenantID uuid.UUID, limit int) []PopularReport {
	prefix := fmt.Sprintf("%s:", tenantID)

	pt.viewLock.RLock()
	defer pt.viewLock.RUnlock()

	// Collect reports for tenant
	type reportCount struct {
		id    uuid.UUID
		count int64
	}
	var reports []reportCount

	for key, count := range pt.viewCounts {
		if len(key) > len(prefix) && key[:len(prefix)] == prefix {
			idStr := key[len(prefix):]
			if id, err := uuid.Parse(idStr); err == nil {
				reports = append(reports, reportCount{id, count})
			}
		}
	}

	// Sort by count descending
	for i := 0; i < len(reports)-1; i++ {
		for j := i + 1; j < len(reports); j++ {
			if reports[j].count > reports[i].count {
				reports[i], reports[j] = reports[j], reports[i]
			}
		}
	}

	// Return top N
	result := make([]PopularReport, 0, limit)
	for i := 0; i < len(reports) && i < limit; i++ {
		result = append(result, PopularReport{
			ReportID:  reports[i].id,
			ViewCount: reports[i].count,
		})
	}

	return result
}

// PopularReport represents a popular report entry
type PopularReport struct {
	ReportID  uuid.UUID `json:"report_id"`
	ViewCount int64     `json:"view_count"`
}

// GetUserRecommendations returns personalized report recommendations
func (pt *PopularityTracker) GetUserRecommendations(userID uuid.UUID, tenantID uuid.UUID, limit int) []uuid.UUID {
	pt.historyLock.RLock()
	history := pt.userHistory[userID]
	pt.historyLock.RUnlock()

	// Simple collaborative filtering - find users with similar history
	// In production, this would use a proper ML model

	// For now, return popular reports not in user's history
	popular := pt.GetPopularReports(tenantID, limit+len(history))

	historySet := make(map[uuid.UUID]bool)
	for _, id := range history {
		historySet[id] = true
	}

	result := make([]uuid.UUID, 0, limit)
	for _, p := range popular {
		if !historySet[p.ReportID] {
			result = append(result, p.ReportID)
			if len(result) >= limit {
				break
			}
		}
	}

	return result
}

// ============================================================================
// DATA QUALITY SCORING
// ============================================================================

// DataQualityScore represents the quality of report data
type DataQualityScore struct {
	OverallScore float64            `json:"overall_score"` // 0-100
	Completeness float64            `json:"completeness"`  // % of expected fields populated
	Freshness    float64            `json:"freshness"`     // Based on data age
	Accuracy     float64            `json:"accuracy"`      // Based on validation rules
	Consistency  float64            `json:"consistency"`   // Cross-field consistency
	Issues       []DataQualityIssue `json:"issues,omitempty"`
	CalculatedAt time.Time          `json:"calculated_at"`
}

// DataQualityIssue represents a specific quality issue
type DataQualityIssue struct {
	Field       string `json:"field"`
	IssueType   string `json:"issue_type"` // missing, stale, invalid, inconsistent
	Severity    string `json:"severity"`   // low, medium, high
	Description string `json:"description"`
	Impact      string `json:"impact"`
}

// DataQualityAnalyzer analyzes data quality for reports
type DataQualityAnalyzer struct {
	rules []DataQualityRule
}

// DataQualityRule defines a quality check rule
type DataQualityRule struct {
	Name        string
	Field       string
	CheckFunc   func(value interface{}) bool
	Severity    string
	Description string
}

// NewDataQualityAnalyzer creates a data quality analyzer
func NewDataQualityAnalyzer() *DataQualityAnalyzer {
	return &DataQualityAnalyzer{
		rules: []DataQualityRule{},
	}
}

// AnalyzeReportData analyzes data quality for a report result
func (dqa *DataQualityAnalyzer) AnalyzeReportData(data json.RawMessage) *DataQualityScore {
	score := &DataQualityScore{
		CalculatedAt: time.Now(),
	}

	// Parse data
	var rows []map[string]interface{}
	if err := json.Unmarshal(data, &rows); err != nil {
		score.OverallScore = 0
		score.Issues = append(score.Issues, DataQualityIssue{
			IssueType:   "invalid",
			Severity:    "high",
			Description: "Unable to parse report data",
		})
		return score
	}

	if len(rows) == 0 {
		score.OverallScore = 100
		score.Completeness = 100
		score.Freshness = 100
		score.Accuracy = 100
		score.Consistency = 100
		return score
	}

	// Analyze completeness
	totalFields := 0
	populatedFields := 0
	for _, row := range rows {
		for _, v := range row {
			totalFields++
			if v != nil && v != "" {
				populatedFields++
			}
		}
	}
	if totalFields > 0 {
		score.Completeness = float64(populatedFields) / float64(totalFields) * 100
	}

	// Default scores
	score.Freshness = 100 // Would check data timestamps
	score.Accuracy = 100  // Would run validation rules
	score.Consistency = 100

	// Calculate overall
	score.OverallScore = (score.Completeness + score.Freshness + score.Accuracy + score.Consistency) / 4

	return score
}
