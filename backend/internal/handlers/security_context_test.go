package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hondyman/uisce/backend/internal/msgcat"
	"github.com/hondyman/uisce/backend/internal/security"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockDatasourceResolver implements security.DatasourceResolver for testing
type MockDatasourceResolver struct {
	resolveFunc func(ctx context.Context, datasourceID string) (*security.ResolvedDatasource, error)
}

func (m *MockDatasourceResolver) Resolve(ctx context.Context, datasourceID string) (*security.ResolvedDatasource, error) {
	if m.resolveFunc != nil {
		return m.resolveFunc(ctx, datasourceID)
	}
	// Default: return valid resolved datasource with no region restrictions
	return &security.ResolvedDatasource{
		TenantID:       "tenant-123",
		InstanceID:     "instance-456",
		ProductID:      "product-789",
		DatasourceID:   datasourceID,
		AllowedRegions: []string{}, // Empty = allow any region
	}, nil
}

func TestSecurityContextFromRequest_ValidHeaders(t *testing.T) {
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("X-Datasource-Id", "ds-123")
	req.Header.Set("X-Region", "us-east-1")

	// Inject AuthInfo into context (simulating AuthContextMiddleware)
	authInfo := security.AuthInfo{
		UserID:    "user-456",
		Roles:     []string{"admin"},
		TenantIDs: []string{"tenant-123"},
	}
	ctx := security.WithAuthInfo(req.Context(), authInfo)
	req = req.WithContext(ctx)

	deps := SecurityContextDeps{
		Resolver: &MockDatasourceResolver{},
	}

	secCtx, newCtx, err := SecurityContextFromRequest(req, "", "", deps)

	require.NoError(t, err)
	assert.NotNil(t, secCtx)
	assert.NotNil(t, newCtx)
	assert.Equal(t, "user-456", secCtx.UserID)
	assert.Equal(t, "tenant-123", secCtx.TenantID)
	assert.Equal(t, "ds-123", secCtx.DatasourceID)
	assert.Equal(t, "us-east-1", secCtx.Region)
}

func TestSecurityContextFromRequest_FallbackHeaders(t *testing.T) {
	tests := []struct {
		name             string
		datasourceHeader string
		datasourceValue  string
		regionHeader     string
		regionValue      string
	}{
		{
			name:             "X-Tenant-Datasource-ID",
			datasourceHeader: "X-Tenant-Datasource-ID",
			datasourceValue:  "ds-456",
			regionHeader:     "X-Region",
			regionValue:      "us-west-2",
		},
		{
			name:             "X-Tenant-Instance-ID",
			datasourceHeader: "X-Tenant-Instance-ID",
			datasourceValue:  "ds-789",
			regionHeader:     "X-Region",
			regionValue:      "us-east-1",
		},
		{
			name:             "X-Tenant-Region",
			datasourceHeader: "X-Datasource-Id",
			datasourceValue:  "ds-111",
			regionHeader:     "X-Tenant-Region",
			regionValue:      "us-west-2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/test", nil)
			req.Header.Set(tt.datasourceHeader, tt.datasourceValue)
			req.Header.Set(tt.regionHeader, tt.regionValue)

			authInfo := security.AuthInfo{
				UserID:    "user-123",
				Roles:     []string{"user"},
				TenantIDs: []string{"tenant-123"},
			}
			ctx := security.WithAuthInfo(req.Context(), authInfo)
			req = req.WithContext(ctx)

			deps := SecurityContextDeps{
				Resolver: &MockDatasourceResolver{},
			}

			secCtx, _, err := SecurityContextFromRequest(req, "", "", deps)

			require.NoError(t, err)
			assert.Equal(t, tt.datasourceValue, secCtx.DatasourceID)
			assert.Equal(t, tt.regionValue, secCtx.Region)
		})
	}
}

func TestSecurityContextFromRequest_BodyParametersPriority(t *testing.T) {
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("X-Datasource-Id", "header-ds")
	req.Header.Set("X-Region", "header-region")

	authInfo := security.AuthInfo{
		UserID:    "user-123",
		Roles:     []string{"user"},
		TenantIDs: []string{"tenant-123"},
	}
	ctx := security.WithAuthInfo(req.Context(), authInfo)
	req = req.WithContext(ctx)

	deps := SecurityContextDeps{
		Resolver: &MockDatasourceResolver{},
	}

	// Body parameters should take priority over headers
	secCtx, _, err := SecurityContextFromRequest(req, "body-ds", "body-region", deps)

	require.NoError(t, err)
	assert.Equal(t, "body-ds", secCtx.DatasourceID)
	assert.Equal(t, "body-region", secCtx.Region)
}

func TestSecurityContextFromRequest_MissingDatasource(t *testing.T) {
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("X-Region", "us-east-1")

	authInfo := security.AuthInfo{
		UserID:    "user-123",
		Roles:     []string{"user"},
		TenantIDs: []string{"tenant-123"},
	}
	ctx := security.WithAuthInfo(req.Context(), authInfo)
	req = req.WithContext(ctx)

	deps := SecurityContextDeps{
		Resolver: &MockDatasourceResolver{},
	}

	_, _, err := SecurityContextFromRequest(req, "", "", deps)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "datasource_id is required")
	assert.Contains(t, err.Error(), "X-Datasource-Id")
}

func TestSecurityContextFromRequest_MissingRegion(t *testing.T) {
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("X-Datasource-Id", "ds-123")

	authInfo := security.AuthInfo{
		UserID:    "user-123",
		Roles:     []string{"user"},
		TenantIDs: []string{"tenant-123"},
	}
	ctx := security.WithAuthInfo(req.Context(), authInfo)
	req = req.WithContext(ctx)

	deps := SecurityContextDeps{
		Resolver: &MockDatasourceResolver{},
	}

	_, _, err := SecurityContextFromRequest(req, "", "", deps)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "region is required")
	assert.Contains(t, err.Error(), "X-Region")
}

func TestSecurityContextFromRequest_MissingAuthContext(t *testing.T) {
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("X-Datasource-Id", "ds-123")
	req.Header.Set("X-Region", "us-east-1")

	// No AuthInfo in context

	deps := SecurityContextDeps{
		Resolver: &MockDatasourceResolver{},
	}

	_, _, err := SecurityContextFromRequest(req, "", "", deps)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "authentication required")
	assert.Contains(t, err.Error(), "JWT token")
}

func TestSecurityContextFromRequest_NoTenants(t *testing.T) {
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("X-Datasource-Id", "ds-123")
	req.Header.Set("X-Region", "us-east-1")

	authInfo := security.AuthInfo{
		UserID:    "user-123",
		Roles:     []string{"user"},
		TenantIDs: []string{}, // Empty tenant list
	}
	ctx := security.WithAuthInfo(req.Context(), authInfo)
	req = req.WithContext(ctx)

	deps := SecurityContextDeps{
		Resolver: &MockDatasourceResolver{},
	}

	_, _, err := SecurityContextFromRequest(req, "", "", deps)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "no tenants assigned")
	assert.Contains(t, err.Error(), "tenant_id or tenant_ids claim")
}

func TestSecurityContextFromRequest_UnauthorizedDatasource(t *testing.T) {
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("X-Datasource-Id", "ds-123")
	req.Header.Set("X-Region", "us-east-1")

	// User has access to tenant-456, but datasource belongs to tenant-123
	authInfo := security.AuthInfo{
		UserID:    "user-123",
		Roles:     []string{"user"},
		TenantIDs: []string{"tenant-456"},
	}
	ctx := security.WithAuthInfo(req.Context(), authInfo)
	req = req.WithContext(ctx)

	deps := SecurityContextDeps{
		Resolver: &MockDatasourceResolver{
			resolveFunc: func(ctx context.Context, datasourceID string) (*security.ResolvedDatasource, error) {
				return &security.ResolvedDatasource{
					TenantID:       "tenant-123", // Different tenant
					InstanceID:     "instance-456",
					ProductID:      "product-789",
					DatasourceID:   datasourceID,
					AllowedRegions: []string{"us-east-1"},
				}, nil
			},
		},
	}

	_, _, err := SecurityContextFromRequest(req, "", "", deps)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "datasource not found")
}

func TestSecurityContextFromRequest_InvalidRegion(t *testing.T) {
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("X-Datasource-Id", "ds-123")
	req.Header.Set("X-Region", "eu-west-1") // Not in allowed regions

	authInfo := security.AuthInfo{
		UserID:    "user-123",
		Roles:     []string{"user"},
		TenantIDs: []string{"tenant-123"},
	}
	ctx := security.WithAuthInfo(req.Context(), authInfo)
	req = req.WithContext(ctx)

	deps := SecurityContextDeps{
		Resolver: &MockDatasourceResolver{
			resolveFunc: func(ctx context.Context, datasourceID string) (*security.ResolvedDatasource, error) {
				return &security.ResolvedDatasource{
					TenantID:       "tenant-123",
					InstanceID:     "instance-456",
					ProductID:      "product-789",
					DatasourceID:   datasourceID,
					AllowedRegions: []string{"us-east-1", "us-west-2"}, // eu-west-1 not allowed
				}, nil
			},
		},
	}

	_, _, err := SecurityContextFromRequest(req, "", "", deps)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "region")
	assert.Contains(t, err.Error(), "not configured")
}

func TestSecurityContextFromRequest_NilResolver(t *testing.T) {
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("X-Datasource-Id", "ds-123")
	req.Header.Set("X-Region", "us-east-1")

	authInfo := security.AuthInfo{
		UserID:    "user-123",
		Roles:     []string{"user"},
		TenantIDs: []string{"tenant-123"},
	}
	ctx := security.WithAuthInfo(req.Context(), authInfo)
	req = req.WithContext(ctx)

	deps := SecurityContextDeps{
		Resolver: nil, // No resolver configured
	}

	_, _, err := SecurityContextFromRequest(req, "", "", deps)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "datasource resolver not configured")
	assert.Contains(t, err.Error(), "internal error")
}

// securityContextFailure runs SecurityContextFromRequest for a caller of
// tenant-own and writes its mapped catalog error, returning the response.
func securityContextFailure(t *testing.T, auth *security.AuthInfo, resolve func(context.Context, string) (*security.ResolvedDatasource, error)) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest("GET", "/api/schedules/", nil)
	req.Header.Set("X-Tenant-Datasource-ID", "11111111-1111-1111-1111-111111111111")
	req.Header.Set("X-Tenant-Region", "us-east-1")
	if auth != nil {
		req = req.WithContext(security.WithAuthInfo(req.Context(), *auth))
	}
	_, _, err := SecurityContextFromRequest(req, "", "", SecurityContextDeps{Resolver: &MockDatasourceResolver{resolveFunc: resolve}})
	require.Error(t, err)
	rec := httptest.NewRecorder()
	var cat *msgcat.Catalog // no catalog wired: renders code and status only
	cat.WriteError(rec, req, "", SecurityContextError(err))
	return rec
}

func decodeErrorBody(t *testing.T, rec *httptest.ResponseRecorder) msgcat.ErrorBody {
	t.Helper()
	var body msgcat.ErrorBody
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&body))
	return body
}

func TestSecurityContextError_UnknownDatasourceIsForbiddenNotUnauthenticated(t *testing.T) {
	rec := securityContextFailure(t,
		&security.AuthInfo{UserID: "u", TenantIDs: []string{"tenant-own"}},
		func(context.Context, string) (*security.ResolvedDatasource, error) {
			return nil, fmt.Errorf("%w: %w", security.ErrDatasourceNotAvailable, sql.ErrNoRows)
		})
	assert.Equal(t, http.StatusForbidden, rec.Code)
	body := decodeErrorBody(t, rec)
	assert.Equal(t, "1-16", body.Code)
	assert.NotContains(t, body.Error, "no rows", "the cause is logged, never sent")
}

func TestSecurityContextError_OtherTenantsDatasourceIsForbidden(t *testing.T) {
	rec := securityContextFailure(t,
		&security.AuthInfo{UserID: "u", TenantIDs: []string{"tenant-own"}},
		func(_ context.Context, id string) (*security.ResolvedDatasource, error) {
			return &security.ResolvedDatasource{TenantID: "tenant-other", InstanceID: "i", ProductID: "p", DatasourceID: id}, nil
		})
	assert.Equal(t, http.StatusForbidden, rec.Code)
	assert.Equal(t, "1-16", decodeErrorBody(t, rec).Code)
}

func TestSecurityContextError_MissingAuthIsStillUnauthenticated(t *testing.T) {
	rec := securityContextFailure(t, nil, nil)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	assert.Equal(t, "1-3", decodeErrorBody(t, rec).Code)
}

func TestSecurityContextError_ResolverOutageIsNotDatasourceUnavailable(t *testing.T) {
	err := SecurityContextError(errors.New("resolving datasource: connection refused"))
	assert.Equal(t, "1-3", err.Code())
}
