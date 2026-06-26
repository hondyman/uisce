package store

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/hondyman/semlayer/backend/internal/models"
	"github.com/jmoiron/sqlx"
)

// APIKeyUsageStore defines operations for API key usage logging
type APIKeyUsageStore interface {
	LogUsage(ctx context.Context, req models.APIKeyUsageCreateRequest) error
	GetAPIKeyUsage(ctx context.Context, apiKeyID uuid.UUID, limit int) ([]*models.APIKeyUsage, error)
	GetAPIKeyUsageByTenant(ctx context.Context, tenantID uuid.UUID, limit int) ([]*models.APIKeyUsage, error)
	GetDailyUsageByTenant(ctx context.Context, tenantID uuid.UUID, days int) ([]*models.DailyUsageStats, error)
	GetEndpointUsageByTenant(ctx context.Context, tenantID uuid.UUID, limit int) ([]*models.EndpointUsageStats, error)
	GetRecentUsageByTenant(ctx context.Context, tenantID uuid.UUID, limit int) ([]*models.APIKeyUsage, error)
}

// apiKeyUsageStoreImpl implements APIKeyUsageStore
type apiKeyUsageStoreImpl struct {
	db *sqlx.DB
}

// NewAPIKeyUsageStore creates a new API key usage store
func NewAPIKeyUsageStore(db *sqlx.DB) APIKeyUsageStore {
	return &apiKeyUsageStoreImpl{db: db}
}

// LogUsage logs an API key usage event
func (s *apiKeyUsageStoreImpl) LogUsage(ctx context.Context, req models.APIKeyUsageCreateRequest) error {
	query := `
		INSERT INTO api_key_usage (api_key_id, user_id, tenant_id, path, method, region, ip_address, user_agent)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`

	_, err := s.db.ExecContext(ctx, query,
		req.APIKeyID, req.UserID, req.TenantID, req.Path, req.Method,
		req.Region, req.IPAddress, req.UserAgent,
	)

	if err != nil {
		return fmt.Errorf("failed to log API key usage: %w", err)
	}

	return nil
}

// GetAPIKeyUsage retrieves usage records for a specific API key
func (s *apiKeyUsageStoreImpl) GetAPIKeyUsage(ctx context.Context, apiKeyID uuid.UUID, limit int) ([]*models.APIKeyUsage, error) {
	if limit <= 0 {
		limit = 100
	}

	query := `
		SELECT id, api_key_id, user_id, tenant_id, path, method, region, ip_address, user_agent, created_at
		FROM api_key_usage
		WHERE api_key_id = $1
		ORDER BY created_at DESC
		LIMIT $2
	`

	var usage []*models.APIKeyUsage
	err := s.db.SelectContext(ctx, &usage, query, apiKeyID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get API key usage: %w", err)
	}

	return usage, nil
}

// GetAPIKeyUsageByTenant retrieves usage records for a specific tenant
func (s *apiKeyUsageStoreImpl) GetAPIKeyUsageByTenant(ctx context.Context, tenantID uuid.UUID, limit int) ([]*models.APIKeyUsage, error) {
	if limit <= 0 {
		limit = 100
	}

	query := `
		SELECT id, api_key_id, user_id, tenant_id, path, method, region, ip_address, user_agent, created_at
		FROM api_key_usage
		WHERE tenant_id = $1
		ORDER BY created_at DESC
		LIMIT $2
	`

	var usage []*models.APIKeyUsage
	err := s.db.SelectContext(ctx, &usage, query, tenantID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get tenant API key usage: %w", err)
	}

	return usage, nil
}

// GetDailyUsageByTenant retrieves daily usage statistics for a tenant
func (s *apiKeyUsageStoreImpl) GetDailyUsageByTenant(ctx context.Context, tenantID uuid.UUID, days int) ([]*models.DailyUsageStats, error) {
	if days <= 0 {
		days = 30
	}

	query := `
		SELECT
			date_trunc('day', created_at)::DATE AS day,
			COUNT(*) AS count
		FROM api_key_usage
		WHERE tenant_id = $1
		AND created_at > now() - INTERVAL '1 day' * $2
		GROUP BY day
		ORDER BY day DESC
	`

	var stats []*models.DailyUsageStats
	err := s.db.SelectContext(ctx, &stats, query, tenantID, days)
	if err != nil {
		return nil, fmt.Errorf("failed to get daily usage: %w", err)
	}

	return stats, nil
}

// GetEndpointUsageByTenant retrieves endpoint usage statistics for a tenant
func (s *apiKeyUsageStoreImpl) GetEndpointUsageByTenant(ctx context.Context, tenantID uuid.UUID, limit int) ([]*models.EndpointUsageStats, error) {
	if limit <= 0 {
		limit = 20
	}

	query := `
		SELECT
			path,
			COUNT(*) AS count
		FROM api_key_usage
		WHERE tenant_id = $1
		GROUP BY path
		ORDER BY count DESC
		LIMIT $2
	`

	var stats []*models.EndpointUsageStats
	err := s.db.SelectContext(ctx, &stats, query, tenantID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get endpoint usage: %w", err)
	}

	return stats, nil
}

// GetRecentUsageByTenant retrieves recent usage records for a tenant
func (s *apiKeyUsageStoreImpl) GetRecentUsageByTenant(ctx context.Context, tenantID uuid.UUID, limit int) ([]*models.APIKeyUsage, error) {
	if limit <= 0 {
		limit = 100
	}

	query := `
		SELECT id, api_key_id, user_id, tenant_id, path, method, region, ip_address, user_agent, created_at
		FROM api_key_usage
		WHERE tenant_id = $1
		ORDER BY created_at DESC
		LIMIT $2
	`

	var usage []*models.APIKeyUsage
	err := s.db.SelectContext(ctx, &usage, query, tenantID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get recent usage: %w", err)
	}

	return usage, nil
}
