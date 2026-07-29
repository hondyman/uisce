package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/golang-jwt/jwt/v5"
	"github.com/hondyman/uisce/backend/internal/governance/contracts"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testJWTSecret = "test-contract-gateway-secret-2024"

func init() {
	os.Setenv("JWT_SECRET", testJWTSecret)
}

func createTestToken(tenantID, userID string) string {
	claims := jwt.MapClaims{
		"user_id":    userID,
		"tenant_id":  tenantID,
		"email":      userID + "@test.com",
		"tenant_ids": []string{tenantID},
		"roles":      []string{"admin"},
		"is_active":  true,
		"exp":        time.Now().Add(time.Hour).Unix(),
		"iat":        time.Now().Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, _ := token.SignedString([]byte(testJWTSecret))
	return tokenString
}

func TestDataContractHandler_Validate_Unauthorized(t *testing.T) {
	handler := NewDataContractHandler(contracts.NewGatekeeper(nil, nil))
	r := chi.NewRouter()
	handler.RegisterRoutes(r)

	body := `{"tenant_id": "tenant-1", "datasource_id": "ds-1", "proposed_diffs": []}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/governance/contracts/validate", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestDataContractHandler_Validate_EmptyBody(t *testing.T) {
	handler := NewDataContractHandler(contracts.NewGatekeeper(nil, nil))
	r := chi.NewRouter()
	handler.RegisterRoutes(r)

	token := createTestToken("tenant-1", "user-1")
	req := httptest.NewRequest(http.MethodPost, "/api/v1/governance/contracts/validate", bytes.NewBufferString(""))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestDataContractHandler_Validate_InvalidJSON(t *testing.T) {
	handler := NewDataContractHandler(contracts.NewGatekeeper(nil, nil))
	r := chi.NewRouter()
	handler.RegisterRoutes(r)

	token := createTestToken("tenant-1", "user-1")
	req := httptest.NewRequest(http.MethodPost, "/api/v1/governance/contracts/validate", bytes.NewBufferString("not json"))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestDataContractHandler_Validate_TenantMismatch(t *testing.T) {
	handler := NewDataContractHandler(contracts.NewGatekeeper(nil, nil))
	r := chi.NewRouter()
	handler.RegisterRoutes(r)

	token := createTestToken("tenant-1", "user-1")
	body := `{"tenant_id": "different-tenant", "datasource_id": "ds-1", "proposed_diffs": []}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/governance/contracts/validate", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)

	var resp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, "tenant_mismatch", resp["code"])
}

func TestDataContractHandler_Validate_EmptyDiffs_Safe(t *testing.T) {
	handler := NewDataContractHandler(contracts.NewGatekeeper(nil, nil))
	r := chi.NewRouter()
	handler.RegisterRoutes(r)

	token := createTestToken("tenant-1", "user-1")
	body := `{"tenant_id": "tenant-1", "datasource_id": "ds-1", "proposed_diffs": []}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/governance/contracts/validate", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp contracts.ContractValidationResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.True(t, resp.AllSafe)
	assert.False(t, resp.HasCritical)
	assert.Equal(t, 0, resp.ViolationsCount)
}

func TestDataContractHandler_Validate_SafeColumnChange(t *testing.T) {
	handler := NewDataContractHandler(contracts.NewGatekeeper(nil, nil))
	r := chi.NewRouter()
	handler.RegisterRoutes(r)

	token := createTestToken("tenant-1", "user-1")
	body := `{
		"tenant_id": "tenant-1",
		"datasource_id": "ds-1",
		"proposed_diffs": [{
			"table_name": "orders",
			"datasource_id": "ds-1",
			"columns": [{
				"column_name": "price",
				"change_kind": "TYPE_CHANGED",
				"old_type": "INTEGER",
				"new_type": "BIGINT"
			}]
		}]
	}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/governance/contracts/validate", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp contracts.ContractValidationResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.True(t, resp.AllSafe)
	assert.False(t, resp.HasCritical)
}

func TestDataContractHandler_Validate_CriticalViolation_Returns422(t *testing.T) {
	handler := NewDataContractHandler(contracts.NewGatekeeper(nil, nil))
	r := chi.NewRouter()
	handler.RegisterRoutes(r)

	token := createTestToken("tenant-1", "user-1")
	body := `{
		"tenant_id": "tenant-1",
		"datasource_id": "ds-1",
		"proposed_diffs": [{
			"table_name": "orders",
			"datasource_id": "ds-1",
			"columns": [{
				"column_name": "customer_id",
				"change_kind": "RENAMED"
			}]
		}]
	}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/governance/contracts/validate", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)

	var resp contracts.ContractValidationResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.False(t, resp.AllSafe)
	assert.True(t, resp.HasCritical)
}

func TestDataContractHandler_Validate_NonNullToNull_ReturnsCritical(t *testing.T) {
	handler := NewDataContractHandler(contracts.NewGatekeeper(nil, nil))
	r := chi.NewRouter()
	handler.RegisterRoutes(r)

	token := createTestToken("tenant-1", "user-1")
	body := `{
		"tenant_id": "tenant-1",
		"datasource_id": "ds-1",
		"proposed_diffs": [{
			"table_name": "accounts",
			"datasource_id": "ds-1",
			"columns": [{
				"column_name": "balance",
				"change_kind": "NULLABILITY_CHANGED",
				"old_nullable": false,
				"new_nullable": true
			}]
		}]
	}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/governance/contracts/validate", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
}

func TestDataContractHandler_Validate_ColumnRename_IsCritical(t *testing.T) {
	handler := NewDataContractHandler(contracts.NewGatekeeper(nil, nil))
	r := chi.NewRouter()
	handler.RegisterRoutes(r)

	token := createTestToken("tenant-1", "user-1")
	body := `{
		"tenant_id": "tenant-1",
		"datasource_id": "ds-1",
		"proposed_diffs": [{
			"table_name": "customers",
			"datasource_id": "ds-1",
			"columns": [{
				"column_name": "old_name",
				"change_kind": "RENAMED"
			}]
		}]
	}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/governance/contracts/validate", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
}

func TestDataContractHandler_Validate_HeaderTenantOverridesClaims(t *testing.T) {
	handler := NewDataContractHandler(contracts.NewGatekeeper(nil, nil))
	r := chi.NewRouter()
	handler.RegisterRoutes(r)

	token := createTestToken("tenant-1", "user-1")
	body := `{"tenant_id": "tenant-1", "datasource_id": "ds-1", "proposed_diffs": []}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/governance/contracts/validate", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("X-Tenant-ID", "tenant-1")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestDataContractHandler_ListViolations_NoAuth(t *testing.T) {
	handler := NewDataContractHandler(contracts.NewGatekeeper(nil, nil))
	r := chi.NewRouter()
	handler.RegisterRoutes(r)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/governance/contracts/violations", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestDataContractHandler_ListViolations_WithNilDB_ReturnsEmpty(t *testing.T) {
	handler := NewDataContractHandler(contracts.NewGatekeeper(nil, nil))
	r := chi.NewRouter()
	handler.RegisterRoutes(r)

	token := createTestToken("tenant-1", "user-1")
	req := httptest.NewRequest(http.MethodGet, "/api/v1/governance/contracts/violations", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("X-Tenant-ID", "tenant-1")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, float64(0), resp["count"])
}

func TestDataContractHandler_ApproveViolation_NoAuth(t *testing.T) {
	handler := NewDataContractHandler(contracts.NewGatekeeper(nil, nil))
	r := chi.NewRouter()
	handler.RegisterRoutes(r)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/governance/contracts/violations/viol-1/approve", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestDataContractHandler_ApproveViolation_MissingID(t *testing.T) {
	handler := NewDataContractHandler(contracts.NewGatekeeper(nil, nil))
	r := chi.NewRouter()
	handler.RegisterRoutes(r)

	token := createTestToken("tenant-1", "user-1")
	req := httptest.NewRequest(http.MethodPost, "/api/v1/governance/contracts/violations//approve", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestDataContractHandler_RejectViolation_NoAuth(t *testing.T) {
	handler := NewDataContractHandler(contracts.NewGatekeeper(nil, nil))
	r := chi.NewRouter()
	handler.RegisterRoutes(r)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/governance/contracts/violations/viol-1/reject", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestDataContractHandler_RejectViolation_WithBody(t *testing.T) {
	handler := NewDataContractHandler(contracts.NewGatekeeper(nil, nil))
	r := chi.NewRouter()
	handler.RegisterRoutes(r)

	token := createTestToken("tenant-1", "user-1")
	body := `{"reason": "schema change rejected by data steward"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/governance/contracts/violations/viol-1/reject", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, "blocked", resp["status"])
	assert.Equal(t, "viol-1", resp["violation_id"])
}

func TestDataContractHandler_Validate_RequiresColumnsField(t *testing.T) {
	handler := NewDataContractHandler(contracts.NewGatekeeper(nil, nil))
	r := chi.NewRouter()
	handler.RegisterRoutes(r)

	token := createTestToken("tenant-1", "user-1")
	body := `{"tenant_id": "tenant-1", "datasource_id": "ds-1", "proposed_diffs": [{"table_name": "orders", "datasource_id": "ds-1"}]}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/governance/contracts/validate", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestExtractValidatedTenantID(t *testing.T) {
	tests := []struct {
		name          string
		authHeader    string
		tenantHeader  string
		expectedCode int
	}{
		{
			name:          "valid token with matching header",
			authHeader:    "Bearer " + createTestToken("tenant-1", "user-1"),
			tenantHeader:  "tenant-1",
			expectedCode:  http.StatusOK,
		},
		{
			name:          "no auth header",
			tenantHeader:  "tenant-1",
			expectedCode:  http.StatusUnauthorized,
		},
		{
			name:          "tenant mismatch",
			authHeader:    "Bearer " + createTestToken("tenant-1", "user-1"),
			tenantHeader:  "other-tenant",
			expectedCode:  http.StatusForbidden,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := NewDataContractHandler(contracts.NewGatekeeper(nil, nil))
			r := chi.NewRouter()
			handler.RegisterRoutes(r)

			body := `{"tenant_id": "tenant-1", "datasource_id": "ds-1", "proposed_diffs": []}`
			req := httptest.NewRequest(http.MethodPost, "/api/v1/governance/contracts/validate", bytes.NewBufferString(body))
			req.Header.Set("Content-Type", "application/json")
			if tt.authHeader != "" {
				req.Header.Set("Authorization", tt.authHeader)
			}
			if tt.tenantHeader != "" {
				req.Header.Set("X-Tenant-ID", tt.tenantHeader)
			}
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedCode, w.Code)
		})
	}
}
