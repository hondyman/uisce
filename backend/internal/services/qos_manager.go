package services

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"
)

// QoSError represents a QoS-related error
type QoSError struct {
	Code       string
	HTTPStatus int
	Message    string
}

func (e *QoSError) Error() string {
	return fmt.Sprintf("QoS Error: %s (%s)", e.Message, e.Code)
}

// QoSManager handles rate limiting and capacity checks
type QoSManager struct {
	db     *sqlx.DB
	logger *zap.Logger

	// In-memory cache of quotas: TenantID -> Resource -> Limit/Window
	quotas map[string]map[string]QuotaDef
	mu     sync.RWMutex

	// In-memory usage tracker: TenantID:Resource -> count
	// Simplified sliding window or fixed window reset
	usage   map[string]int
	usageMu sync.Mutex

	stopChan chan struct{}
}

type QuotaDef struct {
	Limit  int64
	Window int // Seconds
}

func NewQoSManager(db *sqlx.DB) *QoSManager {
	logger, _ := zap.NewProduction()
	m := &QoSManager{
		db:       db,
		logger:   logger,
		quotas:   make(map[string]map[string]QuotaDef),
		usage:    make(map[string]int),
		stopChan: make(chan struct{}),
	}
	// Start background refresh of quotas
	go m.refreshQuotasLoop()
	// Start background reset of usage (simplified global ticker for MVP)
	go m.resetUsageLoop()
	return m
}

func (m *QoSManager) Stop() {
	close(m.stopChan)
}

func (m *QoSManager) refreshQuotasLoop() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	m.RefreshQuotas() // Initial load
	for {
		select {
		case <-ticker.C:
			m.RefreshQuotas()
		case <-m.stopChan:
			return
		}
	}
}

func (m *QoSManager) resetUsageLoop() {
	// For MVP, we reset all counters every minute.
	// Real implementation would be per-window bucket or token bucket.
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			m.usageMu.Lock()
			m.usage = make(map[string]int) // Clear all
			m.usageMu.Unlock()
		case <-m.stopChan:
			return
		}
	}
}

func (m *QoSManager) RefreshQuotas() {
	m.logger.Info("Refreshing Tenant Quotas")
	rows, err := m.db.Queryx("SELECT tenant_id, resource_name, limit_value, window_seconds FROM tenant_quotas")
	if err != nil {
		m.logger.Error("Failed to refresh quotas", zap.Error(err))
		return
	}
	defer rows.Close()

	newQuotas := make(map[string]map[string]QuotaDef)
	for rows.Next() {
		var t, r string
		var l int64
		var w int
		if err := rows.Scan(&t, &r, &l, &w); err == nil {
			if _, ok := newQuotas[t]; !ok {
				newQuotas[t] = make(map[string]QuotaDef)
			}
			newQuotas[t][r] = QuotaDef{Limit: l, Window: w}
		}
	}

	m.mu.Lock()
	m.quotas = newQuotas
	m.mu.Unlock()
}

// CheckAccess returns true if the tenant is within limits for the resource
func (m *QoSManager) CheckAccess(tenantID string, resource string) (bool, error) {
	// 1. Get Quota
	m.mu.RLock()
	tenantQuotas, ok := m.quotas[tenantID]
	if !ok {
		// Fallback to default tenant policies if specific not found, or Allow if no quota defined
		// Let's check "default" tenant as fallback
		tenantQuotas, ok = m.quotas["default"]
	}
	m.mu.RUnlock()

	if !ok {
		// No quotas defined at all? Allow by default or Deny?
		// Allow for safety in MVP
		return true, nil
	}

	def, ok := tenantQuotas[resource]
	if !ok {
		// Resource not limited
		return true, nil
	}

	// 2. Check Usage
	key := fmt.Sprintf("%s:%s", tenantID, resource)
	m.usageMu.Lock()
	defer m.usageMu.Unlock()

	current := m.usage[key]
	if int64(current) >= def.Limit {
		return false, fmt.Errorf("quota exceeded for %s: %d/%d", resource, current, def.Limit)
	}

	// Increment (this isn't atomic with check in distributed sys, but fine for single instance)
	m.usage[key]++
	return true, nil
}

// CheckQoS performs checks suitable for AccessIntelligenceService
func (m *QoSManager) CheckQoS(ctx context.Context, tenantID string, config *TenantQoSConfig) error {
	// For now, we reuse CheckAccess Logic primarily, checking a general "requests" resource
	// effectively ignoring config for the simple quota check, but we could use config.ConcurrencyLimit here too.

	allowed, err := m.CheckAccess(tenantID, "requests")
	if err != nil {
		return &QoSError{Code: "internal_error", Message: err.Error(), HTTPStatus: 500}
	}
	if !allowed {
		return &QoSError{Code: "rate_limit_exceeded", Message: "Quota exceeded", HTTPStatus: 429}
	}
	return nil
}

// RecordQoSResult updates QoS metrics based on success/failure of a request
func (m *QoSManager) RecordQoSResult(tenantID string, success bool) {
	// Stub for recording metrics or adjusting adaptive limits
}

func (m *QoSManager) GetAllQuotas() map[string]map[string]QuotaDef {
	m.mu.RLock()
	defer m.mu.RUnlock()
	// Return a copy to avoid race conditions
	copy := make(map[string]map[string]QuotaDef)
	for t, resources := range m.quotas {
		copy[t] = make(map[string]QuotaDef)
		for r, def := range resources {
			copy[t][r] = def
		}
	}
	return copy
}

func (m *QoSManager) UpdateQuota(tenantID, resource string, limit int64, window int) error {
	// Update DB
	query := `INSERT INTO tenant_quotas (tenant_id, resource_name, limit_value, window_seconds) 
	          VALUES ($1, $2, $3, $4)
	          ON CONFLICT (tenant_id, resource_name) 
	          DO UPDATE SET limit_value = EXCLUDED.limit_value, window_seconds = EXCLUDED.window_seconds`
	_, err := m.db.Exec(query, tenantID, resource, limit, window)
	if err != nil {
		return err
	}

	// Refresh cache immediately
	m.RefreshQuotas()
	return nil
}
