package api_test

import (
	"context"
	"net/http"
	"os"

	"github.com/hondyman/uisce/backend/internal/security"
)

func init() {
	if os.Getenv("API_TOKEN_ENCRYPTION_KEY") == "" {
		_ = os.Setenv("API_TOKEN_ENCRYPTION_KEY", "MDEyMzQ1Njc4OWFiY2RlZjAxMjM0NTY3ODlhYmNkZWY=")
	}
}

// withAuth injects a mock AuthInfo into the request context for testing security-aware handlers.
func withAuth(r *http.Request, tenantID string) *http.Request {
	auth := security.AuthInfo{
		UserID:    "user1",
		TenantIDs: []string{tenantID},
	}
	return r.WithContext(security.WithAuthInfo(r.Context(), auth))
}

// withValidHeaders adds required security headers to the request.
func withValidHeaders(r *http.Request, tenantID, datasourceID string) *http.Request {
	r.Header.Set("X-Tenant-ID", tenantID)
	r.Header.Set("X-Datasource-ID", datasourceID)
	r.Header.Set("X-Region", "us-east-1")
	return r
}

// mockResolver implements security.DatasourceResolver for testing
type mockResolver struct{}

func (m *mockResolver) Resolve(ctx context.Context, datasourceID string) (*security.ResolvedDatasource, error) {
	return &security.ResolvedDatasource{
		DatasourceID:   datasourceID,
		TenantID:       "ten", // Match tenantID in tests
		InstanceID:     "inst1",
		ProductID:      "prod1",
		AllowedRegions: []string{"us-east-1"},
	}, nil
}
