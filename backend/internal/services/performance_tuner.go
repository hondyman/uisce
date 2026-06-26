package services

import (
	"context"
	"fmt"
	"math"
	"sync"
	"sync/atomic"
	"time"

	"github.com/hondyman/semlayer/backend/internal/logging"
)

// AutoscalingConfig contains autoscaling configuration
type AutoscalingConfig struct {
	Enabled             bool
	MinReplicas         int
	MaxReplicas         int
	TargetP95Latency    time.Duration
	TargetP99Latency    time.Duration
	ScaleUpThreshold    float64 // percentage above target
	ScaleDownThreshold  float64 // percentage below target
	ScaleUpCooldown     time.Duration
	ScaleDownCooldown   time.Duration
	QueueDepthThreshold int
	HeadroomTarget      float64 // target headroom percentage (0.3 = 30%)
}

// TuningRecommendation represents a tuning recommendation
type TuningRecommendation struct {
	Component       string
	Issue           string
	Recommendation  string
	Priority        string // "critical", "high", "medium", "low"
	Impact          string
	TimeToImplement time.Duration
}

// PerformanceTuner provides automated performance tuning and autoscaling
type PerformanceTuner struct {
	config         *AutoscalingConfig
	perfMonitor    *PerformanceMonitor
	loadTestEngine *LoadTestEngine

	// Scaling state
	currentReplicas int
	lastScaleUp     time.Time
	lastScaleDown   time.Time
	scalingMu       sync.Mutex

	// Tuning analysis
	recommendations   []TuningRecommendation
	recommendationsMu sync.RWMutex

	// Continuous monitoring
	stopChan chan struct{}
	wg       sync.WaitGroup
}

// NewPerformanceTuner creates a new performance tuner
func NewPerformanceTuner(config *AutoscalingConfig, perfMonitor *PerformanceMonitor, loadTestEngine *LoadTestEngine) *PerformanceTuner {
	return &PerformanceTuner{
		config:          config,
		perfMonitor:     perfMonitor,
		loadTestEngine:  loadTestEngine,
		currentReplicas: 1,
		recommendations: make([]TuningRecommendation, 0),
		stopChan:        make(chan struct{}),
	}
}

// Start begins continuous performance monitoring and tuning
func (pt *PerformanceTuner) Start(ctx context.Context) {
	logging.GetLogger().Sugar().Info("Starting performance tuner")

	pt.wg.Add(2)
	go pt.monitorAndTune(ctx)
	go pt.analyzePerformance(ctx)

	logging.GetLogger().Sugar().Info("Performance tuner started")
}

// Stop stops the performance tuner
func (pt *PerformanceTuner) Stop() {
	close(pt.stopChan)
	pt.wg.Wait()
	logging.GetLogger().Sugar().Info("Performance tuner stopped")
}

// monitorAndTune continuously monitors performance and applies tuning
func (pt *PerformanceTuner) monitorAndTune(ctx context.Context) {
	defer pt.wg.Done()

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-pt.stopChan:
			return
		case <-ticker.C:
			pt.analyzeAndTune()
		}
	}
}

// analyzeAndTune performs performance analysis and applies tuning recommendations
func (pt *PerformanceTuner) analyzeAndTune() {
	stats := pt.perfMonitor.GetStats()
	if stats == nil {
		return
	}

	// Analyze cache performance
	pt.analyzeCachePerformance(stats)

	// Analyze QoS performance
	pt.analyzeQoSPerformance()

	// Analyze database performance
	pt.analyzeDatabasePerformance(stats)

	// Check autoscaling needs
	if pt.config.Enabled {
		pt.checkAutoscaling(stats)
	}

	// Log current performance status
	pt.logPerformanceStatus(stats)
}

// analyzeCachePerformance analyzes cache hit rates and makes recommendations
func (pt *PerformanceTuner) analyzeCachePerformance(stats map[string]interface{}) {
	cacheHitRateStr, ok := stats["cache_hit_rate"].(string)
	if !ok {
		return
	}

	var cacheHitRate float64
	fmt.Sscanf(cacheHitRateStr, "%f%%", &cacheHitRate)
	cacheHitRate /= 100

	cacheSize := stats["cache_size"].(int64)
	inflight := stats["cache_inflight"].(int64)

	// Cache hit rate analysis
	if cacheHitRate < 0.85 {
		rec := TuningRecommendation{
			Component:       "cache",
			Issue:           fmt.Sprintf("Cache hit rate is %.1f%% (target: 90%%+)", cacheHitRate*100),
			Recommendation:  "Increase TTL, improve pre-warming, or analyze key diversity",
			Priority:        "high",
			Impact:          "High - affects response latency and backend load",
			TimeToImplement: 2 * time.Hour,
		}
		pt.addRecommendation(rec)
	}

	// Cache size analysis
	if cacheSize > 1000000 { // 1M entries
		rec := TuningRecommendation{
			Component:       "cache",
			Issue:           fmt.Sprintf("Cache size is %d entries - may be too large", cacheSize),
			Recommendation:  "Review cache TTL settings and consider sharding",
			Priority:        "medium",
			Impact:          "Medium - affects memory usage",
			TimeToImplement: 4 * time.Hour,
		}
		pt.addRecommendation(rec)
	}

	// In-flight cache refreshes analysis
	if inflight > 100 {
		rec := TuningRecommendation{
			Component:       "cache",
			Issue:           fmt.Sprintf("%d in-flight cache refreshes - possible stampede", inflight),
			Recommendation:  "Increase singleflight window or batch invalidations",
			Priority:        "critical",
			Impact:          "Critical - causes latency spikes and resource exhaustion",
			TimeToImplement: 1 * time.Hour,
		}
		pt.addRecommendation(rec)
	}
}

// analyzeQoSPerformance analyzes QoS metrics and makes recommendations
func (pt *PerformanceTuner) analyzeQoSPerformance() {
	denials := atomic.LoadInt64(&pt.perfMonitor.qosTokenDenials)
	breakerTrips := atomic.LoadInt64(&pt.perfMonitor.qosBreakerTrips)
	loadShed := atomic.LoadInt64(&pt.perfMonitor.qosLoadShed)

	totalRequests := atomic.LoadInt64(&pt.perfMonitor.totalRequests)

	if totalRequests == 0 {
		return
	}

	denialRate := float64(denials) / float64(totalRequests) * 100
	breakerRate := float64(breakerTrips) / float64(totalRequests) * 100

	// QoS denial analysis
	if denialRate > 5.0 {
		rec := TuningRecommendation{
			Component:       "qos",
			Issue:           fmt.Sprintf("QoS denial rate is %.1f%% (target: <5%%)", denialRate),
			Recommendation:  "Increase token bucket capacity or review rate limits",
			Priority:        "high",
			Impact:          "High - affects user experience",
			TimeToImplement: 30 * time.Minute,
		}
		pt.addRecommendation(rec)
	}

	// Circuit breaker analysis
	if breakerRate > 1.0 {
		rec := TuningRecommendation{
			Component:       "qos",
			Issue:           fmt.Sprintf("Circuit breaker trips: %.1f%% (target: <1%%)", breakerRate),
			Recommendation:  "Investigate upstream service issues or increase breaker thresholds",
			Priority:        "critical",
			Impact:          "Critical - indicates service instability",
			TimeToImplement: 15 * time.Minute,
		}
		pt.addRecommendation(rec)
	}

	// Load shedding analysis
	if loadShed > 0 {
		rec := TuningRecommendation{
			Component:       "qos",
			Issue:           fmt.Sprintf("%d requests load-shed - system under extreme pressure", loadShed),
			Recommendation:  "Immediate scaling required or implement backpressure",
			Priority:        "critical",
			Impact:          "Critical - system protection activated",
			TimeToImplement: 5 * time.Minute,
		}
		pt.addRecommendation(rec)
	}
}

// analyzeDatabasePerformance analyzes database metrics
func (pt *PerformanceTuner) analyzeDatabasePerformance(stats map[string]interface{}) {
	// This would integrate with database connection pool metrics
	// For now, we'll use basic heuristics

	activeRequests := stats["active_requests"].(int64)
	if activeRequests > 100 {
		rec := TuningRecommendation{
			Component:       "database",
			Issue:           fmt.Sprintf("%d active requests - possible DB bottleneck", activeRequests),
			Recommendation:  "Check connection pool size, consider read replicas, or optimize queries",
			Priority:        "high",
			Impact:          "High - affects all requests",
			TimeToImplement: 1 * time.Hour,
		}
		pt.addRecommendation(rec)
	}
}

// checkAutoscaling determines if scaling is needed
func (pt *PerformanceTuner) checkAutoscaling(stats map[string]interface{}) {
	pt.scalingMu.Lock()
	defer pt.scalingMu.Unlock()

	now := time.Now()

	// Get current latency metrics
	// This would need to be enhanced with actual p95/p99 tracking
	activeRequests := stats["active_requests"].(int64)

	// Simple scaling logic based on active requests and queue depth
	shouldScaleUp := activeRequests > int64(pt.config.QueueDepthThreshold)
	shouldScaleDown := activeRequests < int64(pt.config.QueueDepthThreshold/2)

	if shouldScaleUp && pt.currentReplicas < pt.config.MaxReplicas {
		if now.Sub(pt.lastScaleUp) > pt.config.ScaleUpCooldown {
			pt.scaleUp()
			pt.lastScaleUp = now
		}
	} else if shouldScaleDown && pt.currentReplicas > pt.config.MinReplicas {
		if now.Sub(pt.lastScaleDown) > pt.config.ScaleDownCooldown {
			pt.scaleDown()
			pt.lastScaleDown = now
		}
	}
}

// scaleUp increases the number of replicas
func (pt *PerformanceTuner) scaleUp() {
	oldReplicas := pt.currentReplicas
	pt.currentReplicas = int(math.Min(float64(pt.config.MaxReplicas), float64(pt.currentReplicas)*1.5))

	if pt.currentReplicas != oldReplicas {
		logging.GetLogger().Sugar().Infof("Scaling UP: %d -> %d replicas", oldReplicas, pt.currentReplicas)
		// Here you would integrate with your container orchestration system
		// e.g., Kubernetes HPA, AWS ECS, etc.
	}
}

// scaleDown decreases the number of replicas
func (pt *PerformanceTuner) scaleDown() {
	oldReplicas := pt.currentReplicas
	newReplicas := int(math.Max(float64(pt.config.MinReplicas), float64(pt.currentReplicas)*0.8))

	// Ensure we maintain headroom
	headroomRatio := float64(pt.currentReplicas) / float64(newReplicas)
	if headroomRatio < (1 + pt.config.HeadroomTarget) {
		newReplicas = int(float64(pt.currentReplicas) / (1 + pt.config.HeadroomTarget))
		newReplicas = int(math.Max(float64(pt.config.MinReplicas), float64(newReplicas)))
	}

	if newReplicas != oldReplicas {
		pt.currentReplicas = newReplicas
		logging.GetLogger().Sugar().Infof("Scaling DOWN: %d -> %d replicas", oldReplicas, pt.currentReplicas)
		// Here you would integrate with your container orchestration system
	}
}

// analyzePerformance runs periodic deep performance analysis
func (pt *PerformanceTuner) analyzePerformance(ctx context.Context) {
	defer pt.wg.Done()

	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-pt.stopChan:
			return
		case <-ticker.C:
			pt.runDeepAnalysis()
		}
	}
}

// runDeepAnalysis performs comprehensive performance analysis
func (pt *PerformanceTuner) runDeepAnalysis() {
	logging.GetLogger().Sugar().Info("Running deep performance analysis")

	// Analyze tenant-specific performance
	pt.analyzeTenantPerformance()

	// Analyze error patterns
	pt.analyzeErrorPatterns()

	// Generate weekly load test recommendations
	pt.generateLoadTestRecommendations()

	logging.GetLogger().Sugar().Info("Deep performance analysis completed")
}

// analyzeTenantPerformance analyzes per-tenant performance metrics
func (pt *PerformanceTuner) analyzeTenantPerformance() {
	// This would analyze per-tenant metrics from the performance monitor
	// For now, we'll add a placeholder recommendation

	rec := TuningRecommendation{
		Component:       "tenant-isolation",
		Issue:           "Per-tenant performance analysis needed",
		Recommendation:  "Implement detailed per-tenant latency tracking and QoS enforcement",
		Priority:        "medium",
		Impact:          "Medium - improves tenant experience",
		TimeToImplement: 4 * time.Hour,
	}
	pt.addRecommendation(rec)
}

// analyzeErrorPatterns analyzes error patterns for insights
func (pt *PerformanceTuner) analyzeErrorPatterns() {
	// Analyze error patterns from recent logs/metrics
	// This would integrate with logging and metrics systems

	rec := TuningRecommendation{
		Component:       "error-analysis",
		Issue:           "Error pattern analysis needed",
		Recommendation:  "Implement error rate monitoring and alerting",
		Priority:        "low",
		Impact:          "Low - improves observability",
		TimeToImplement: 2 * time.Hour,
	}
	pt.addRecommendation(rec)
}

// generateLoadTestRecommendations generates recommendations for load testing
func (pt *PerformanceTuner) generateLoadTestRecommendations() {
	rec := TuningRecommendation{
		Component:       "load-testing",
		Issue:           "Weekly load testing cadence needed",
		Recommendation:  "Schedule weekly synthetic load tests at 1x and 2x peak load",
		Priority:        "medium",
		Impact:          "Medium - ensures system reliability",
		TimeToImplement: 8 * time.Hour,
	}
	pt.addRecommendation(rec)
}

// addRecommendation adds a tuning recommendation
func (pt *PerformanceTuner) addRecommendation(rec TuningRecommendation) {
	pt.recommendationsMu.Lock()
	defer pt.recommendationsMu.Unlock()

	// Check if recommendation already exists
	for _, existing := range pt.recommendations {
		if existing.Component == rec.Component && existing.Issue == rec.Issue {
			return // Already exists
		}
	}

	pt.recommendations = append(pt.recommendations, rec)
	logging.GetLogger().Sugar().Warnf("TUNING RECOMMENDATION [%s]: %s - %s",
		rec.Priority, rec.Component, rec.Recommendation)
}

// GetRecommendations returns current tuning recommendations
func (pt *PerformanceTuner) GetRecommendations() []TuningRecommendation {
	pt.recommendationsMu.RLock()
	defer pt.recommendationsMu.RUnlock()

	recs := make([]TuningRecommendation, len(pt.recommendations))
	copy(recs, pt.recommendations)
	return recs
}

// ClearRecommendations clears all recommendations
func (pt *PerformanceTuner) ClearRecommendations() {
	pt.recommendationsMu.Lock()
	defer pt.recommendationsMu.Unlock()
	pt.recommendations = pt.recommendations[:0]
}

// logPerformanceStatus logs current performance status
func (pt *PerformanceTuner) logPerformanceStatus(stats map[string]interface{}) {
	activeRequests := stats["active_requests"].(int64)
	cacheHitRate := stats["cache_hit_rate"].(string)
	errorCount := stats["error_count"].(int64)

	logging.GetLogger().Sugar().Infof("Performance Status: active=%d, cache_hit=%s, errors=%d, replicas=%d",
		activeRequests, cacheHitRate, errorCount, pt.currentReplicas)
}

// GetCurrentReplicas returns the current number of replicas
func (pt *PerformanceTuner) GetCurrentReplicas() int {
	pt.scalingMu.Lock()
	defer pt.scalingMu.Unlock()
	return pt.currentReplicas
}

// CreateDefaultConfig creates a default autoscaling configuration
func CreateDefaultConfig() *AutoscalingConfig {
	return &AutoscalingConfig{
		Enabled:             true,
		MinReplicas:         1,
		MaxReplicas:         10,
		TargetP95Latency:    100 * time.Millisecond,
		TargetP99Latency:    200 * time.Millisecond,
		ScaleUpThreshold:    0.8, // 80% of target
		ScaleDownThreshold:  0.3, // 30% of target
		ScaleUpCooldown:     2 * time.Minute,
		ScaleDownCooldown:   5 * time.Minute,
		QueueDepthThreshold: 50,
		HeadroomTarget:      0.4, // 40% headroom
	}
}
