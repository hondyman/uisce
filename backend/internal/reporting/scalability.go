package reporting

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"golang.org/x/time/rate"
)

// ============================================================================
// RATE LIMITING & THROTTLING
// ============================================================================

// RateLimitConfig defines rate limiting rules
type RateLimitConfig struct {
	// Per-tenant limits
	TenantRequestsPerSecond float64
	TenantBurstSize         int

	// Per-user limits
	UserRequestsPerSecond float64
	UserBurstSize         int

	// Global limits
	GlobalRequestsPerSecond float64
	GlobalBurstSize         int

	// Report generation limits (more restrictive)
	RenderRequestsPerMinute float64
	RenderBurstSize         int

	// Export limits
	ExportRequestsPerHour float64
	ExportMaxConcurrent   int
}

// DefaultRateLimitConfig returns production rate limit settings
func DefaultRateLimitConfig() *RateLimitConfig {
	return &RateLimitConfig{
		TenantRequestsPerSecond: 100,
		TenantBurstSize:         200,
		UserRequestsPerSecond:   20,
		UserBurstSize:           50,
		GlobalRequestsPerSecond: 10000,
		GlobalBurstSize:         20000,
		RenderRequestsPerMinute: 30,
		RenderBurstSize:         10,
		ExportRequestsPerHour:   100,
		ExportMaxConcurrent:     5,
	}
}

// RateLimiter manages rate limiting across tenants and users
type RateLimiter struct {
	config         *RateLimitConfig
	globalLimiter  *rate.Limiter
	tenantLimiters map[uuid.UUID]*rate.Limiter
	userLimiters   map[uuid.UUID]*rate.Limiter
	renderLimiters map[uuid.UUID]*rate.Limiter
	mutex          sync.RWMutex
}

// NewRateLimiter creates a rate limiter
func NewRateLimiter(config *RateLimitConfig) *RateLimiter {
	return &RateLimiter{
		config:         config,
		globalLimiter:  rate.NewLimiter(rate.Limit(config.GlobalRequestsPerSecond), config.GlobalBurstSize),
		tenantLimiters: make(map[uuid.UUID]*rate.Limiter),
		userLimiters:   make(map[uuid.UUID]*rate.Limiter),
		renderLimiters: make(map[uuid.UUID]*rate.Limiter),
	}
}

// AllowRequest checks if a request should be allowed
func (rl *RateLimiter) AllowRequest(ctx context.Context, tenantID uuid.UUID, userID *uuid.UUID) error {
	// Check global limit
	if !rl.globalLimiter.Allow() {
		return &RateLimitError{Type: "global", RetryAfter: time.Second}
	}

	// Check tenant limit
	tenantLimiter := rl.getTenantLimiter(tenantID)
	if !tenantLimiter.Allow() {
		return &RateLimitError{Type: "tenant", TenantID: tenantID, RetryAfter: time.Second}
	}

	// Check user limit if provided
	if userID != nil {
		userLimiter := rl.getUserLimiter(*userID)
		if !userLimiter.Allow() {
			return &RateLimitError{Type: "user", UserID: userID, RetryAfter: time.Second}
		}
	}

	return nil
}

// AllowRender checks if a render request should be allowed
func (rl *RateLimiter) AllowRender(ctx context.Context, tenantID uuid.UUID) error {
	renderLimiter := rl.getRenderLimiter(tenantID)
	if !renderLimiter.Allow() {
		return &RateLimitError{
			Type:       "render",
			TenantID:   tenantID,
			RetryAfter: time.Minute / time.Duration(rl.config.RenderRequestsPerMinute),
		}
	}
	return nil
}

func (rl *RateLimiter) getTenantLimiter(tenantID uuid.UUID) *rate.Limiter {
	rl.mutex.RLock()
	limiter, ok := rl.tenantLimiters[tenantID]
	rl.mutex.RUnlock()

	if ok {
		return limiter
	}

	rl.mutex.Lock()
	defer rl.mutex.Unlock()

	// Double-check after acquiring write lock
	if limiter, ok = rl.tenantLimiters[tenantID]; ok {
		return limiter
	}

	limiter = rate.NewLimiter(
		rate.Limit(rl.config.TenantRequestsPerSecond),
		rl.config.TenantBurstSize,
	)
	rl.tenantLimiters[tenantID] = limiter
	return limiter
}

func (rl *RateLimiter) getUserLimiter(userID uuid.UUID) *rate.Limiter {
	rl.mutex.RLock()
	limiter, ok := rl.userLimiters[userID]
	rl.mutex.RUnlock()

	if ok {
		return limiter
	}

	rl.mutex.Lock()
	defer rl.mutex.Unlock()

	if limiter, ok = rl.userLimiters[userID]; ok {
		return limiter
	}

	limiter = rate.NewLimiter(
		rate.Limit(rl.config.UserRequestsPerSecond),
		rl.config.UserBurstSize,
	)
	rl.userLimiters[userID] = limiter
	return limiter
}

func (rl *RateLimiter) getRenderLimiter(tenantID uuid.UUID) *rate.Limiter {
	rl.mutex.RLock()
	limiter, ok := rl.renderLimiters[tenantID]
	rl.mutex.RUnlock()

	if ok {
		return limiter
	}

	rl.mutex.Lock()
	defer rl.mutex.Unlock()

	if limiter, ok = rl.renderLimiters[tenantID]; ok {
		return limiter
	}

	limiter = rate.NewLimiter(
		rate.Limit(rl.config.RenderRequestsPerMinute/60), // Convert to per-second
		rl.config.RenderBurstSize,
	)
	rl.renderLimiters[tenantID] = limiter
	return limiter
}

// RateLimitError represents a rate limit exceeded error
type RateLimitError struct {
	Type       string
	TenantID   uuid.UUID
	UserID     *uuid.UUID
	RetryAfter time.Duration
}

func (e *RateLimitError) Error() string {
	return fmt.Sprintf("rate limit exceeded (%s), retry after %v", e.Type, e.RetryAfter)
}

// ============================================================================
// RESOURCE QUOTAS
// ============================================================================

// ResourceQuota defines tenant resource limits
type ResourceQuota struct {
	// Storage
	MaxStorageBytes  int64 `json:"max_storage_bytes"`
	UsedStorageBytes int64 `json:"used_storage_bytes"`

	// Reports
	MaxReportDefinitions  int `json:"max_report_definitions"`
	UsedReportDefinitions int `json:"used_report_definitions"`

	// Schedules
	MaxSchedules    int `json:"max_schedules"`
	ActiveSchedules int `json:"active_schedules"`

	// Concurrent renders
	MaxConcurrentRenders int `json:"max_concurrent_renders"`
	CurrentRenders       int `json:"current_renders"`

	// Data retention
	HistoryRetentionDays int `json:"history_retention_days"`

	// Features
	AllowedOutputFormats []string `json:"allowed_output_formats"`
	AIFeaturesEnabled    bool     `json:"ai_features_enabled"`
	AdvancedScheduling   bool     `json:"advanced_scheduling"`
}

// TenantPlan defines subscription tiers
type TenantPlan string

const (
	PlanFree         TenantPlan = "free"
	PlanStarter      TenantPlan = "starter"
	PlanProfessional TenantPlan = "professional"
	PlanEnterprise   TenantPlan = "enterprise"
)

// GetQuotaForPlan returns default quotas for a subscription plan
func GetQuotaForPlan(plan TenantPlan) *ResourceQuota {
	switch plan {
	case PlanFree:
		return &ResourceQuota{
			MaxStorageBytes:      1024 * 1024 * 100, // 100MB
			MaxReportDefinitions: 5,
			MaxSchedules:         2,
			MaxConcurrentRenders: 1,
			HistoryRetentionDays: 7,
			AllowedOutputFormats: []string{"html", "pdf"},
			AIFeaturesEnabled:    false,
			AdvancedScheduling:   false,
		}
	case PlanStarter:
		return &ResourceQuota{
			MaxStorageBytes:      1024 * 1024 * 1024, // 1GB
			MaxReportDefinitions: 25,
			MaxSchedules:         10,
			MaxConcurrentRenders: 3,
			HistoryRetentionDays: 30,
			AllowedOutputFormats: []string{"html", "pdf", "excel"},
			AIFeaturesEnabled:    false,
			AdvancedScheduling:   true,
		}
	case PlanProfessional:
		return &ResourceQuota{
			MaxStorageBytes:      1024 * 1024 * 1024 * 10, // 10GB
			MaxReportDefinitions: 100,
			MaxSchedules:         50,
			MaxConcurrentRenders: 10,
			HistoryRetentionDays: 90,
			AllowedOutputFormats: []string{"html", "pdf", "excel", "csv", "json"},
			AIFeaturesEnabled:    true,
			AdvancedScheduling:   true,
		}
	case PlanEnterprise:
		return &ResourceQuota{
			MaxStorageBytes:      1024 * 1024 * 1024 * 100, // 100GB
			MaxReportDefinitions: -1,                       // Unlimited
			MaxSchedules:         -1,                       // Unlimited
			MaxConcurrentRenders: 50,
			HistoryRetentionDays: 365,
			AllowedOutputFormats: []string{"html", "pdf", "excel", "csv", "json", "xml"},
			AIFeaturesEnabled:    true,
			AdvancedScheduling:   true,
		}
	default:
		return GetQuotaForPlan(PlanFree)
	}
}

// QuotaManager manages tenant resource quotas
type QuotaManager struct {
	quotas map[uuid.UUID]*ResourceQuota
	mutex  sync.RWMutex
}

// NewQuotaManager creates a quota manager
func NewQuotaManager() *QuotaManager {
	return &QuotaManager{
		quotas: make(map[uuid.UUID]*ResourceQuota),
	}
}

// GetQuota returns the quota for a tenant
func (qm *QuotaManager) GetQuota(tenantID uuid.UUID) *ResourceQuota {
	qm.mutex.RLock()
	defer qm.mutex.RUnlock()

	if quota, ok := qm.quotas[tenantID]; ok {
		return quota
	}
	return GetQuotaForPlan(PlanFree)
}

// SetQuota sets the quota for a tenant
func (qm *QuotaManager) SetQuota(tenantID uuid.UUID, quota *ResourceQuota) {
	qm.mutex.Lock()
	defer qm.mutex.Unlock()
	qm.quotas[tenantID] = quota
}

// CheckQuota checks if an operation is allowed under quota
func (qm *QuotaManager) CheckQuota(tenantID uuid.UUID, operation string) error {
	quota := qm.GetQuota(tenantID)

	switch operation {
	case "create_report":
		if quota.MaxReportDefinitions >= 0 && quota.UsedReportDefinitions >= quota.MaxReportDefinitions {
			return &QuotaExceededError{
				Resource: "report_definitions",
				Limit:    quota.MaxReportDefinitions,
				Used:     quota.UsedReportDefinitions,
			}
		}
	case "create_schedule":
		if quota.MaxSchedules >= 0 && quota.ActiveSchedules >= quota.MaxSchedules {
			return &QuotaExceededError{
				Resource: "schedules",
				Limit:    quota.MaxSchedules,
				Used:     quota.ActiveSchedules,
			}
		}
	case "render_report":
		if quota.CurrentRenders >= quota.MaxConcurrentRenders {
			return &QuotaExceededError{
				Resource: "concurrent_renders",
				Limit:    quota.MaxConcurrentRenders,
				Used:     quota.CurrentRenders,
			}
		}
	}

	return nil
}

// QuotaExceededError represents a quota exceeded error
type QuotaExceededError struct {
	Resource string
	Limit    int
	Used     int
}

func (e *QuotaExceededError) Error() string {
	return fmt.Sprintf("quota exceeded for %s: %d/%d", e.Resource, e.Used, e.Limit)
}

// ============================================================================
// CONNECTION POOLING
// ============================================================================

// PoolConfig configures connection pool behavior
type PoolConfig struct {
	// Database pool
	DBMaxConnections     int
	DBMaxIdleConnections int
	DBConnMaxLifetime    time.Duration
	DBConnMaxIdleTime    time.Duration

	// HTTP client pool (for Cube.dev)
	HTTPMaxConnsPerHost int
	HTTPMaxIdleConns    int
	HTTPIdleConnTimeout time.Duration

	// Worker pool
	RenderWorkers   int
	ExportWorkers   int
	ScheduleWorkers int
}

// DefaultPoolConfig returns production pool settings
func DefaultPoolConfig() *PoolConfig {
	return &PoolConfig{
		DBMaxConnections:     100,
		DBMaxIdleConnections: 25,
		DBConnMaxLifetime:    30 * time.Minute,
		DBConnMaxIdleTime:    5 * time.Minute,
		HTTPMaxConnsPerHost:  100,
		HTTPMaxIdleConns:     100,
		HTTPIdleConnTimeout:  90 * time.Second,
		RenderWorkers:        10,
		ExportWorkers:        5,
		ScheduleWorkers:      3,
	}
}

// ============================================================================
// WORKER POOL FOR ASYNC OPERATIONS
// ============================================================================

// WorkerPool manages background workers for async operations
type WorkerPool struct {
	workers  int
	jobQueue chan Job
	results  chan JobResult
	stopCh   chan struct{}
	wg       sync.WaitGroup
}

// Job represents a background job
type Job struct {
	ID       uuid.UUID
	Type     string // render, export, schedule
	TenantID uuid.UUID
	Payload  interface{}
	Priority int // Higher = more urgent
	Created  time.Time
}

// JobResult represents the result of a job
type JobResult struct {
	JobID   uuid.UUID
	Success bool
	Result  interface{}
	Error   error
	Took    time.Duration
}

// NewWorkerPool creates a worker pool
func NewWorkerPool(workers int, queueSize int) *WorkerPool {
	pool := &WorkerPool{
		workers:  workers,
		jobQueue: make(chan Job, queueSize),
		results:  make(chan JobResult, queueSize),
		stopCh:   make(chan struct{}),
	}
	pool.start()
	return pool
}

func (wp *WorkerPool) start() {
	for i := 0; i < wp.workers; i++ {
		wp.wg.Add(1)
		go wp.worker(i)
	}
}

func (wp *WorkerPool) worker(id int) {
	defer wp.wg.Done()

	for {
		select {
		case job := <-wp.jobQueue:
			start := time.Now()
			result := wp.processJob(job)
			result.Took = time.Since(start)
			wp.results <- result
		case <-wp.stopCh:
			return
		}
	}
}

func (wp *WorkerPool) processJob(job Job) JobResult {
	result := JobResult{JobID: job.ID}

	// Job processing logic would go here
	// This is a placeholder - actual implementation would call appropriate handlers

	switch job.Type {
	case "render":
		// Process render job
		result.Success = true
	case "export":
		// Process export job
		result.Success = true
	case "schedule":
		// Process scheduled report
		result.Success = true
	default:
		result.Error = fmt.Errorf("unknown job type: %s", job.Type)
	}

	return result
}

// Submit adds a job to the queue
func (wp *WorkerPool) Submit(job Job) error {
	select {
	case wp.jobQueue <- job:
		return nil
	default:
		return fmt.Errorf("job queue full")
	}
}

// Results returns the results channel
func (wp *WorkerPool) Results() <-chan JobResult {
	return wp.results
}

// Stop gracefully stops the worker pool
func (wp *WorkerPool) Stop() {
	close(wp.stopCh)
	wp.wg.Wait()
	close(wp.jobQueue)
	close(wp.results)
}

// QueueStats returns queue statistics
func (wp *WorkerPool) QueueStats() (queued int, capacity int) {
	return len(wp.jobQueue), cap(wp.jobQueue)
}

// ============================================================================
// CIRCUIT BREAKER FOR EXTERNAL SERVICES
// ============================================================================

// CircuitBreakerState represents the circuit breaker state
type CircuitBreakerState int

const (
	CircuitClosed CircuitBreakerState = iota
	CircuitOpen
	CircuitHalfOpen
)

// CircuitBreaker prevents cascading failures
type CircuitBreaker struct {
	name             string
	failureThreshold int
	successThreshold int
	timeout          time.Duration

	state       CircuitBreakerState
	failures    int
	successes   int
	lastFailure time.Time
	mutex       sync.RWMutex
}

// NewCircuitBreaker creates a circuit breaker
func NewCircuitBreaker(name string, failureThreshold, successThreshold int, timeout time.Duration) *CircuitBreaker {
	return &CircuitBreaker{
		name:             name,
		failureThreshold: failureThreshold,
		successThreshold: successThreshold,
		timeout:          timeout,
		state:            CircuitClosed,
	}
}

// Execute runs a function through the circuit breaker
func (cb *CircuitBreaker) Execute(fn func() error) error {
	if !cb.canExecute() {
		return &CircuitOpenError{Name: cb.name}
	}

	err := fn()
	cb.recordResult(err == nil)
	return err
}

func (cb *CircuitBreaker) canExecute() bool {
	cb.mutex.RLock()
	state := cb.state
	lastFailure := cb.lastFailure
	cb.mutex.RUnlock()

	switch state {
	case CircuitClosed:
		return true
	case CircuitOpen:
		if time.Since(lastFailure) > cb.timeout {
			cb.mutex.Lock()
			cb.state = CircuitHalfOpen
			cb.mutex.Unlock()
			return true
		}
		return false
	case CircuitHalfOpen:
		return true
	}
	return false
}

func (cb *CircuitBreaker) recordResult(success bool) {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()

	if success {
		cb.failures = 0
		cb.successes++
		if cb.state == CircuitHalfOpen && cb.successes >= cb.successThreshold {
			cb.state = CircuitClosed
			cb.successes = 0
		}
	} else {
		cb.successes = 0
		cb.failures++
		cb.lastFailure = time.Now()
		if cb.failures >= cb.failureThreshold {
			cb.state = CircuitOpen
		}
	}
}

// GetState returns the current state
func (cb *CircuitBreaker) GetState() CircuitBreakerState {
	cb.mutex.RLock()
	defer cb.mutex.RUnlock()
	return cb.state
}

// CircuitOpenError indicates the circuit is open
type CircuitOpenError struct {
	Name string
}

func (e *CircuitOpenError) Error() string {
	return fmt.Sprintf("circuit breaker '%s' is open", e.Name)
}
