package services

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/jmoiron/sqlx"
)

// TenantConfigService manages per-tenant QoS configurations
type TenantConfigService struct {
	db      *sqlx.DB
	configs map[string]*TenantQoSConfig
	mu      sync.RWMutex
}

// NewTenantConfigService creates a new tenant configuration service
func NewTenantConfigService(db *sqlx.DB) *TenantConfigService {
	return &TenantConfigService{
		db:      db,
		configs: make(map[string]*TenantQoSConfig),
	}
}

// GetConfig retrieves QoS configuration for a tenant
func (s *TenantConfigService) GetConfig(tenantID string) (*TenantQoSConfig, error) {
	s.mu.RLock()
	config, ok := s.configs[tenantID]
	s.mu.RUnlock()

	if ok {
		return config, nil
	}

	// Load from database
	config, err := s.loadConfigFromDB(tenantID)
	if err != nil {
		// Return default config if not found
		return s.getDefaultConfig(tenantID), nil
	}

	s.mu.Lock()
	s.configs[tenantID] = config
	s.mu.Unlock()

	return config, nil
}

// loadConfigFromDB loads tenant configuration from database
func (s *TenantConfigService) loadConfigFromDB(tenantID string) (*TenantQoSConfig, error) {
	var config TenantQoSConfig
	query := `SELECT tenant_id, tier, concurrency_limit, token_rate, burst_tokens,
	                 cpu_limit, memory_limit, cache_ttl, priority, features
	          FROM tenant_configs WHERE tenant_id = $1`

	err := s.db.Get(&config, query, tenantID)
	if err != nil {
		return nil, err
	}

	return &config, nil
}

// getDefaultConfig returns default Bronze tier configuration
func (s *TenantConfigService) getDefaultConfig(tenantID string) *TenantQoSConfig {
	return &TenantQoSConfig{
		TenantID:         tenantID,
		Tier:             TierBronze,
		ConcurrencyLimit: 10,
		TokenRate:        100,
		BurstTokens:      200,
		CPULimit:         10.0,
		MemoryLimit:      100 * 1024 * 1024, // 100MB
		CacheTTL:         5 * time.Minute,
		Priority:         1,
		Features: FeatureFlags{
			AutomationAutoApply:    false,
			ConversationalFeatures: true,
			AdvancedAnalytics:      false,
			CustomIntegrations:     false,
		},
	}
}

// UpdateConfig updates tenant configuration
func (s *TenantConfigService) UpdateConfig(config *TenantQoSConfig) error {
	query := `INSERT INTO tenant_configs (tenant_id, tier, concurrency_limit, token_rate,
	                 burst_tokens, cpu_limit, memory_limit, cache_ttl, priority, features)
	          VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	          ON CONFLICT (tenant_id) DO UPDATE SET
	            tier = EXCLUDED.tier,
	            concurrency_limit = EXCLUDED.concurrency_limit,
	            token_rate = EXCLUDED.token_rate,
	            burst_tokens = EXCLUDED.burst_tokens,
	            cpu_limit = EXCLUDED.cpu_limit,
	            memory_limit = EXCLUDED.memory_limit,
	            cache_ttl = EXCLUDED.cache_ttl,
	            priority = EXCLUDED.priority,
	            features = EXCLUDED.features`

	_, err := s.db.Exec(query, config.TenantID, config.Tier, config.ConcurrencyLimit,
		config.TokenRate, config.BurstTokens, config.CPULimit, config.MemoryLimit,
		config.CacheTTL, config.Priority, config.Features)

	if err != nil {
		return err
	}

	s.mu.Lock()
	s.configs[config.TenantID] = config
	s.mu.Unlock()

	return nil
}

// GetTierConfig returns configuration template for a tier
func (s *TenantConfigService) GetTierConfig(tier TenantTier) *TenantQoSConfig {
	switch tier {
	case TierGold:
		return &TenantQoSConfig{
			Tier:             TierGold,
			ConcurrencyLimit: 100,
			TokenRate:        1000,
			BurstTokens:      2000,
			CPULimit:         50.0,
			MemoryLimit:      1 * 1024 * 1024 * 1024, // 1GB
			CacheTTL:         15 * time.Minute,
			Priority:         10,
			Features: FeatureFlags{
				AutomationAutoApply:    true,
				ConversationalFeatures: true,
				AdvancedAnalytics:      true,
				CustomIntegrations:     true,
			},
		}
	case TierSilver:
		return &TenantQoSConfig{
			Tier:             TierSilver,
			ConcurrencyLimit: 50,
			TokenRate:        500,
			BurstTokens:      1000,
			CPULimit:         25.0,
			MemoryLimit:      500 * 1024 * 1024, // 500MB
			CacheTTL:         10 * time.Minute,
			Priority:         5,
			Features: FeatureFlags{
				AutomationAutoApply:    true,
				ConversationalFeatures: true,
				AdvancedAnalytics:      false,
				CustomIntegrations:     false,
			},
		}
	default: // Bronze
		return &TenantQoSConfig{
			Tier:             TierBronze,
			ConcurrencyLimit: 10,
			TokenRate:        100,
			BurstTokens:      200,
			CPULimit:         10.0,
			MemoryLimit:      100 * 1024 * 1024, // 100MB
			CacheTTL:         5 * time.Minute,
			Priority:         1,
			Features: FeatureFlags{
				AutomationAutoApply:    false,
				ConversationalFeatures: true,
				AdvancedAnalytics:      false,
				CustomIntegrations:     false,
			},
		}
	}
}

// BackgroundJobQueue manages per-tenant background job queues
type BackgroundJobQueue struct {
	queues map[string]*TenantQueue
	mu     sync.RWMutex
}

// TenantQueue represents a per-tenant job queue
type TenantQueue struct {
	tenantID string
	jobs     chan Job
	workers  int
	stopChan chan struct{}
}

// Job represents a background job
type Job struct {
	ID       string
	Type     string
	Payload  interface{}
	Priority int
	Deadline time.Time
}

// NewBackgroundJobQueue creates a new background job queue system
func NewBackgroundJobQueue() *BackgroundJobQueue {
	return &BackgroundJobQueue{
		queues: make(map[string]*TenantQueue),
	}
}

// GetQueue retrieves or creates a queue for a tenant
func (q *BackgroundJobQueue) GetQueue(tenantID string, config *TenantQoSConfig) *TenantQueue {
	q.mu.RLock()
	queue, ok := q.queues[tenantID]
	q.mu.RUnlock()

	if ok {
		return queue
	}

	q.mu.Lock()
	defer q.mu.Unlock()

	// Double-check after acquiring write lock
	if queue, ok := q.queues[tenantID]; ok {
		return queue
	}

	queue = &TenantQueue{
		tenantID: tenantID,
		jobs:     make(chan Job, config.ConcurrencyLimit*2), // Buffer size based on concurrency
		workers:  config.ConcurrencyLimit / 10,              // Workers based on concurrency limit
		stopChan: make(chan struct{}),
	}

	// Start workers
	for i := 0; i < queue.workers; i++ {
		go queue.worker()
	}

	q.queues[tenantID] = queue
	return queue
}

// SubmitJob submits a job to a tenant's queue
func (q *BackgroundJobQueue) SubmitJob(tenantID string, job Job, config *TenantQoSConfig) error {
	queue := q.GetQueue(tenantID, config)

	select {
	case queue.jobs <- job:
		return nil
	default:
		return fmt.Errorf("queue full for tenant %s", tenantID)
	}
}

// worker processes jobs for a tenant
func (tq *TenantQueue) worker() {
	for {
		select {
		case job := <-tq.jobs:
			tq.processJob(job)
		case <-tq.stopChan:
			return
		}
	}
}

// processJob processes a single job
func (tq *TenantQueue) processJob(job Job) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	// Process job based on type
	switch job.Type {
	case "drift_scan":
		// Implement drift scanning logic
	case "bundle_mining":
		// Implement bundle mining logic
	case "cache_invalidation":
		// Implement cache invalidation logic
	default:
		fmt.Printf("Unknown job type: %s\n", job.Type)
	}

	_ = ctx // Use context for timeout handling
}
