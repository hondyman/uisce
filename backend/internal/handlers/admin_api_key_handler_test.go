package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/hondyman/semlayer/backend/internal/security"
	"github.com/hondyman/semlayer/backend/internal/services"
	"github.com/stretchr/testify/require"
)

// ============================================================================
// Mock API Key Store for Testing
// ============================================================================

type mockAPIKeyStore struct {
	keys map[string]*services.APIKey
}

func newMockAPIKeyStore() *mockAPIKeyStore {
	return &mockAPIKeyStore{
		keys: make(map[string]*services.APIKey),
	}
}

func (m *mockAPIKeyStore) CreateKey(ctx context.Context, req services.APIKeyCreateRequest) (string, *services.APIKey, error) {
	if req.UserID == "" {
		return "", nil, errors.New("user_id is required")
	}
	if len(req.TenantIDs) == 0 && req.TenantID == "" {
		return "", nil, errors.New("at least one tenant is required")
	}

	plainKey := uuid.NewString()

	ak := &services.APIKey{
		Key:       plainKey,
		UserID:    req.UserID,
		TenantID:  req.TenantID,
		TenantIDs: req.TenantIDs,
		Roles:     req.Roles,
		Active:    true,
	}
	m.keys[plainKey] = ak

	return plainKey, ak, nil
}

func (m *mockAPIKeyStore) GetKey(ctx context.Context, plainKey string) (*services.APIKey, error) {
	if ak, ok := m.keys[plainKey]; ok {
		return ak, nil
	}
	return nil, nil
}

func (m *mockAPIKeyStore) FindByKey(ctx context.Context, rawKey string) (*services.APIKey, error) {
	return m.GetKey(ctx, rawKey)
}

// ============================================================================
// Admin API Key Handler Tests
// ============================================================================

func TestAdminAPIKeyHandler_CreateKey_Valid(t *testing.T) {
	store := newMockAPIKeyStore()
	handler := NewAdminAPIKeyHandler(store)

	creatorID := uuid.NewString()
	userID := uuid.NewString()
	tenantID := uuid.NewString()

	// Create a request with valid admin auth context
	body := map[string]interface{}{
		"user_id":     userID,
		"tenant_id":   tenantID,
		"name":        "test-key",
		"roles":       []string{"GLOBAL_OPS"},
		"description": "test API key",
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest("POST", "/admin/api-keys", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-ID", creatorID)

	// Inject auth context with admin role
	ctx := security.WithAuthInfo(req.Context(), security.AuthInfo{
		UserID:    creatorID,
		Roles:     []string{"GLOBAL_OPS"},
		TenantIDs: []string{tenantID},
	})
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.CreateAPIKey(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)

	var resp map[string]string
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&resp))
	require.NotEmpty(t, resp["api_key"])
}

func TestAdminAPIKeyHandler_CreateKey_MissingAuth(t *testing.T) {
	store := newMockAPIKeyStore()
	handler := NewAdminAPIKeyHandler(store)

	userID := uuid.NewString()
	tenantID := uuid.NewString()

	body := map[string]interface{}{
		"user_id":   userID,
		"tenant_id": tenantID,
		"name":      "test-key",
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest("POST", "/admin/api-keys", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	// No auth context injected

	rr := httptest.NewRecorder()
	handler.CreateAPIKey(rr, req)

	require.Equal(t, http.StatusUnauthorized, rr.Code)
}

func TestAdminAPIKeyHandler_CreateKey_WrongRole(t *testing.T) {
	store := newMockAPIKeyStore()
	handler := NewAdminAPIKeyHandler(store)

	creatorID := uuid.NewString()
	userID := uuid.NewString()
	tenantID := uuid.NewString()

	body := map[string]interface{}{
		"user_id":   userID,
		"tenant_id": tenantID,
		"name":      "test-key",
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest("POST", "/admin/api-keys", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")

	// Inject auth context with non-admin role
	ctx := security.WithAuthInfo(req.Context(), security.AuthInfo{
		UserID:    creatorID,
		Roles:     []string{"USER"},
		TenantIDs: []string{tenantID},
	})
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.CreateAPIKey(rr, req)

	require.Equal(t, http.StatusForbidden, rr.Code)
}

func TestAdminAPIKeyHandler_CreateKey_InvalidJSON(t *testing.T) {
	store := newMockAPIKeyStore()
	handler := NewAdminAPIKeyHandler(store)

	creatorID := uuid.NewString()
	tenantID := uuid.NewString()

	req := httptest.NewRequest("POST", "/admin/api-keys", bytes.NewReader([]byte("invalid json")))
	req.Header.Set("Content-Type", "application/json")

	ctx := security.WithAuthInfo(req.Context(), security.AuthInfo{
		UserID:    creatorID,
		Roles:     []string{"GLOBAL_OPS"},
		TenantIDs: []string{tenantID},
	})
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.CreateAPIKey(rr, req)

	require.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestAdminAPIKeyHandler_CreateKey_MissingUserID(t *testing.T) {
	store := newMockAPIKeyStore()
	handler := NewAdminAPIKeyHandler(store)

	creatorID := uuid.NewString()
	tenantID := uuid.NewString()

	// Omit user_id
	body := map[string]interface{}{
		"tenant_id": tenantID,
		"name":      "test-key",
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest("POST", "/admin/api-keys", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")

	ctx := security.WithAuthInfo(req.Context(), security.AuthInfo{
		UserID:    creatorID,
		Roles:     []string{"GLOBAL_OPS"},
		TenantIDs: []string{tenantID},
	})
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.CreateAPIKey(rr, req)

	require.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestAdminAPIKeyHandler_CreateKey_MissingTenant(t *testing.T) {
	store := newMockAPIKeyStore()
	handler := NewAdminAPIKeyHandler(store)

	creatorID := uuid.NewString()
	userID := uuid.NewString()

	// Omit tenant_id and tenant_ids
	body := map[string]interface{}{
		"user_id": userID,
		"name":    "test-key",
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest("POST", "/admin/api-keys", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")

	ctx := security.WithAuthInfo(req.Context(), security.AuthInfo{
		UserID:    creatorID,
		Roles:     []string{"GLOBAL_OPS"},
		TenantIDs: []string{uuid.NewString()},
	})
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.CreateAPIKey(rr, req)

	require.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestAdminAPIKeyHandler_NoStore(t *testing.T) {
	handler := NewAdminAPIKeyHandler(nil)

	creatorID := uuid.NewString()
	tenantID := uuid.NewString()

	req := httptest.NewRequest("POST", "/admin/api-keys", nil)

	ctx := security.WithAuthInfo(req.Context(), security.AuthInfo{
		UserID:    creatorID,
		Roles:     []string{"GLOBAL_OPS"},
		TenantIDs: []string{tenantID},
	})
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.CreateAPIKey(rr, req)

	require.Equal(t, http.StatusInternalServerError, rr.Code)
}

// ============================================================================
// Test Helper Functions
// ============================================================================

// containsString helper for assertions
func containsString(slice []string, str string) bool {
	for _, s := range slice {
		if s == str {
			return true
		}
	}
	return false
}
