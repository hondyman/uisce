package handlers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/hondyman/semlayer/backend/internal/models"
	"github.com/hondyman/semlayer/backend/internal/security"
	"github.com/stretchr/testify/require"
)

// ============================================================================
// Mock Usage Store for Testing
// ============================================================================

type mockUsageStore struct {
	usageLog      []*models.APIKeyUsage
	dailyStats    []*models.DailyUsageStats
	endpointStats []*models.EndpointUsageStats
}

func newMockUsageStore() *mockUsageStore {
	return &mockUsageStore{
		usageLog:      make([]*models.APIKeyUsage, 0),
		dailyStats:    make([]*models.DailyUsageStats, 0),
		endpointStats: make([]*models.EndpointUsageStats, 0),
	}
}

func (m *mockUsageStore) LogUsage(ctx context.Context, req models.APIKeyUsageCreateRequest) error {
	return nil
}

func (m *mockUsageStore) GetAPIKeyUsage(ctx context.Context, apiKeyID uuid.UUID, limit int) ([]*models.APIKeyUsage, error) {
	return m.usageLog, nil
}

func (m *mockUsageStore) GetAPIKeyUsageByTenant(ctx context.Context, tenantID uuid.UUID, limit int) ([]*models.APIKeyUsage, error) {
	return m.usageLog, nil
}

func (m *mockUsageStore) GetDailyUsageByTenant(ctx context.Context, tenantID uuid.UUID, days int) ([]*models.DailyUsageStats, error) {
	return m.dailyStats, nil
}

func (m *mockUsageStore) GetEndpointUsageByTenant(ctx context.Context, tenantID uuid.UUID, limit int) ([]*models.EndpointUsageStats, error) {
	return m.endpointStats, nil
}

func (m *mockUsageStore) GetRecentUsageByTenant(ctx context.Context, tenantID uuid.UUID, limit int) ([]*models.APIKeyUsage, error) {
	return m.usageLog, nil
}

// ============================================================================
// Admin Usage Handler Tests
// ============================================================================

func TestAdminUsageHandler_GetAPIKeyUsage_Valid(t *testing.T) {
	store := newMockUsageStore()
	handler := NewAdminUsageHandler(store)

	creatorID := uuid.NewString()
	apiKeyID := uuid.NewString()

	req := httptest.NewRequest("GET", "/admin/api-keys/"+apiKeyID+"/usage", nil)

	ctx := security.WithAuthInfo(req.Context(), security.AuthInfo{
		UserID:    creatorID,
		Roles:     []string{"GLOBAL_OPS"},
		TenantIDs: []string{uuid.NewString()},
	})
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	// Simulate chi route param
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("apiKeyID", apiKeyID)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	handler.GetAPIKeyUsage(rr, req)
	require.Equal(t, http.StatusOK, rr.Code)
}

func TestAdminUsageHandler_GetTenantDailyUsage_Valid(t *testing.T) {
	store := newMockUsageStore()
	store.dailyStats = []*models.DailyUsageStats{
		{Day: "2026-02-08", Count: 1234},
		{Day: "2026-02-07", Count: 987},
	}

	handler := NewAdminUsageHandler(store)

	creatorID := uuid.NewString()
	tenantID := uuid.NewString()

	req := httptest.NewRequest("GET", "/admin/tenants/"+tenantID+"/usage/daily?days=30", nil)

	ctx := security.WithAuthInfo(req.Context(), security.AuthInfo{
		UserID:    creatorID,
		Roles:     []string{"GLOBAL_OPS"},
		TenantIDs: []string{tenantID},
	})
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("tenantID", tenantID)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	handler.GetTenantDailyUsage(rr, req)
	require.Equal(t, http.StatusOK, rr.Code)
}

func TestAdminUsageHandler_MissingAuth(t *testing.T) {
	store := newMockUsageStore()
	handler := NewAdminUsageHandler(store)

	apiKeyID := uuid.NewString()

	req := httptest.NewRequest("GET", "/admin/api-keys/"+apiKeyID+"/usage", nil)
	// No auth context

	rr := httptest.NewRecorder()
	// Handler should require auth
	handler.GetAPIKeyUsage(rr, req)
	require.Equal(t, http.StatusUnauthorized, rr.Code)
}

func TestAdminUsageHandler_WrongRole(t *testing.T) {
	store := newMockUsageStore()
	handler := NewAdminUsageHandler(store)

	creatorID := uuid.NewString()
	apiKeyID := uuid.NewString()

	req := httptest.NewRequest("GET", "/admin/api-keys/"+apiKeyID+"/usage", nil)

	ctx := security.WithAuthInfo(req.Context(), security.AuthInfo{
		UserID:    creatorID,
		Roles:     []string{"USER"},
		TenantIDs: []string{uuid.NewString()},
	})
	req = req.WithContext(ctx)

	require.NotNil(t, handler)
}

func TestAdminUsageHandler_InvalidAPIKeyID(t *testing.T) {
	store := newMockUsageStore()
	handler := NewAdminUsageHandler(store)

	creatorID := uuid.NewString()

	req := httptest.NewRequest("GET", "/admin/api-keys/invalid-id/usage", nil)

	ctx := security.WithAuthInfo(req.Context(), security.AuthInfo{
		UserID:    creatorID,
		Roles:     []string{"GLOBAL_OPS"},
		TenantIDs: []string{uuid.NewString()},
	})
	req = req.WithContext(ctx)

	require.NotNil(t, handler)
}

func TestAdminUsageHandler_InvalidTenantID(t *testing.T) {
	store := newMockUsageStore()
	handler := NewAdminUsageHandler(store)

	creatorID := uuid.NewString()

	req := httptest.NewRequest("GET", "/admin/tenants/invalid-id/usage/daily", nil)

	ctx := security.WithAuthInfo(req.Context(), security.AuthInfo{
		UserID:    creatorID,
		Roles:     []string{"GLOBAL_OPS"},
		TenantIDs: []string{uuid.NewString()},
	})
	req = req.WithContext(ctx)

	require.NotNil(t, handler)
}
