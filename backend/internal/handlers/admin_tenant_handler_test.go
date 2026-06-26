package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/hondyman/semlayer/backend/internal/models"
	"github.com/hondyman/semlayer/backend/internal/security"
	"github.com/stretchr/testify/require"
)

// ============================================================================
// Mock Tenant Store for Testing
// ============================================================================

type mockTenantStore struct {
	tenants map[uuid.UUID]*models.Tenant
}

func newMockTenantStore() *mockTenantStore {
	return &mockTenantStore{
		tenants: make(map[uuid.UUID]*models.Tenant),
	}
}

func (m *mockTenantStore) CreateTenant(ctx context.Context, req models.TenantCreateRequest) (*models.Tenant, error) {
	now := time.Now()
	tenant := &models.Tenant{
		ID:            req.ID,
		Name:          req.Name,
		Code:          req.Code,
		Region:        req.Region,
		Plan:          req.Plan,
		MaxRequests:   req.MaxRequests,
		WindowSeconds: req.WindowSeconds,
		IsSuspended:   false,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	m.tenants[req.ID] = tenant
	return tenant, nil
}

func (m *mockTenantStore) GetTenantByID(ctx context.Context, id uuid.UUID) (*models.Tenant, error) {
	return m.tenants[id], nil
}

func (m *mockTenantStore) GetTenantByCode(ctx context.Context, code string) (*models.Tenant, error) {
	for _, t := range m.tenants {
		if t.Code != nil && *t.Code == code {
			return t, nil
		}
	}
	return nil, nil
}

func (m *mockTenantStore) ListTenants(ctx context.Context, limit int, offset int) ([]*models.Tenant, int, error) {
	var result []*models.Tenant
	for _, t := range m.tenants {
		result = append(result, t)
	}
	return result, len(result), nil
}

func (m *mockTenantStore) UpdateTenant(ctx context.Context, id uuid.UUID, req models.TenantUpdateRequest) (*models.Tenant, error) {
	t, ok := m.tenants[id]
	if !ok {
		return nil, errors.New("tenant not found")
	}

	if req.Name != nil {
		t.Name = *req.Name
	}
	if req.Region != nil {
		t.Region = req.Region
	}
	if req.Plan != nil {
		t.Plan = *req.Plan
	}
	if req.MaxRequests != nil {
		t.MaxRequests = req.MaxRequests
	}
	if req.WindowSeconds != nil {
		t.WindowSeconds = req.WindowSeconds
	}
	if req.IsSuspended != nil {
		t.IsSuspended = *req.IsSuspended
	}
	t.UpdatedAt = time.Now()

	return t, nil
}

func (m *mockTenantStore) DeleteTenant(ctx context.Context, id uuid.UUID) error {
	if _, ok := m.tenants[id]; !ok {
		return errors.New("tenant not found")
	}
	delete(m.tenants, id)
	return nil
}

func (m *mockTenantStore) ValidateTenantIDs(ctx context.Context, ids []uuid.UUID) error {
	for _, id := range ids {
		if _, ok := m.tenants[id]; !ok {
			return errors.New("one or more tenant_ids are invalid")
		}
	}
	return nil
}

func (m *mockTenantStore) SuspendTenant(ctx context.Context, id uuid.UUID) error {
	t, ok := m.tenants[id]
	if !ok {
		return errors.New("tenant not found")
	}
	t.IsSuspended = true
	return nil
}

func (m *mockTenantStore) UnsuspendTenant(ctx context.Context, id uuid.UUID) error {
	t, ok := m.tenants[id]
	if !ok {
		return errors.New("tenant not found")
	}
	t.IsSuspended = false
	return nil
}

// ============================================================================
// Admin Tenant Handler Tests
// ============================================================================

func TestAdminTenantHandler_CreateTenant_Valid(t *testing.T) {
	store := newMockTenantStore()
	handler := NewAdminTenantHandler(store)

	creatorID := uuid.NewString()

	body := map[string]interface{}{
		"name":   "Test Tenant",
		"code":   "test-tenant",
		"region": "us-east-1",
		"plan":   "pro",
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest("POST", "/admin/tenants", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")

	ctx := security.WithAuthInfo(req.Context(), security.AuthInfo{
		UserID:    creatorID,
		Roles:     []string{"GLOBAL_OPS"},
		TenantIDs: []string{uuid.NewString()},
	})
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.CreateTenant(rr, req)

	require.Equal(t, http.StatusCreated, rr.Code)

	var resp map[string]interface{}
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&resp))
	require.NotNil(t, resp["tenant"])
}

func TestAdminTenantHandler_CreateTenant_MissingName(t *testing.T) {
	store := newMockTenantStore()
	handler := NewAdminTenantHandler(store)

	creatorID := uuid.NewString()

	body := map[string]interface{}{
		"plan": "pro",
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest("POST", "/admin/tenants", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")

	ctx := security.WithAuthInfo(req.Context(), security.AuthInfo{
		UserID:    creatorID,
		Roles:     []string{"GLOBAL_OPS"},
		TenantIDs: []string{uuid.NewString()},
	})
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.CreateTenant(rr, req)

	require.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestAdminTenantHandler_CreateTenant_InvalidPlan(t *testing.T) {
	store := newMockTenantStore()
	handler := NewAdminTenantHandler(store)

	creatorID := uuid.NewString()

	body := map[string]interface{}{
		"name": "Test Tenant",
		"plan": "invalid-plan",
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest("POST", "/admin/tenants", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")

	ctx := security.WithAuthInfo(req.Context(), security.AuthInfo{
		UserID:    creatorID,
		Roles:     []string{"GLOBAL_OPS"},
		TenantIDs: []string{uuid.NewString()},
	})
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.CreateTenant(rr, req)

	require.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestAdminTenantHandler_GetTenant_Valid(t *testing.T) {
	store := newMockTenantStore()
	tenantID := uuid.New()

	tenant := &models.Tenant{
		ID:          tenantID,
		Name:        "Test Tenant",
		Plan:        "pro",
		IsSuspended: false,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	store.tenants[tenantID] = tenant

	handler := NewAdminTenantHandler(store)
	creatorID := uuid.NewString()

	req := httptest.NewRequest("GET", "/admin/tenants/"+tenantID.String(), nil)

	ctx := security.WithAuthInfo(req.Context(), security.AuthInfo{
		UserID:    creatorID,
		Roles:     []string{"GLOBAL_OPS"},
		TenantIDs: []string{uuid.NewString()},
	})
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	rr = httptest.NewRecorder()
	handler.GetTenant(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
}

func TestAdminTenantHandler_UpdateTenant_Valid(t *testing.T) {
	store := newMockTenantStore()
	tenantID := uuid.New()

	tenant := &models.Tenant{
		ID:          tenantID,
		Name:        "Test Tenant",
		Plan:        "free",
		IsSuspended: false,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	store.tenants[tenantID] = tenant

	handler := NewAdminTenantHandler(store)
	creatorID := uuid.NewString()

	newName := "Updated Tenant"
	body := map[string]interface{}{
		"name": newName,
		"plan": "pro",
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest("PATCH", "/admin/tenants/"+tenantID.String(), bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")

	ctx := security.WithAuthInfo(req.Context(), security.AuthInfo{
		UserID:    creatorID,
		Roles:     []string{"GLOBAL_OPS"},
		TenantIDs: []string{uuid.NewString()},
	})
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.UpdateTenant(rr, req)
	require.Equal(t, http.StatusOK, rr.Code)
}

func TestAdminTenantHandler_MissingAuth(t *testing.T) {
	store := newMockTenantStore()
	handler := NewAdminTenantHandler(store)

	body := map[string]interface{}{
		"name": "Test Tenant",
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest("POST", "/admin/tenants", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	// No auth context

	rr := httptest.NewRecorder()
	handler.CreateTenant(rr, req)

	require.Equal(t, http.StatusUnauthorized, rr.Code)
}

func TestAdminTenantHandler_WrongRole(t *testing.T) {
	store := newMockTenantStore()
	handler := NewAdminTenantHandler(store)

	creatorID := uuid.NewString()

	body := map[string]interface{}{
		"name": "Test Tenant",
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest("POST", "/admin/tenants", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")

	ctx := security.WithAuthInfo(req.Context(), security.AuthInfo{
		UserID:    creatorID,
		Roles:     []string{"USER"},
		TenantIDs: []string{uuid.NewString()},
	})
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.CreateTenant(rr, req)

	require.Equal(t, http.StatusUnauthorized, rr.Code)
}
