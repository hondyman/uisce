package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hondyman/uisce/backend/internal/handlers"
	"github.com/hondyman/uisce/backend/internal/security"
)

// noTenantResolver stands in for the datasource resolver in the no-datasource
// BuildContext branch, which is what listTeams actually hits when no
// X-Datasource-Id header is sent — the exact shape of the request that
// originally produced the 22P02 crash (empty/non-UUID tenant reaching SQL).
type noTenantResolver struct{}

func (noTenantResolver) Resolve(ctx context.Context, datasourceID string) (*security.ResolvedDatasource, error) {
	return &security.ResolvedDatasource{TenantID: "irrelevant", DatasourceID: datasourceID}, nil
}

func (noTenantResolver) ResolveBindingDatasource(ctx context.Context, tenantID, alphaProductID, alphaDatasourceID string) (string, error) {
	return "", nil
}

// TestListTeams_NoTenantContext proves the handler-level guard, not just the
// SecurityContextFromRequest unit tests: a global admin hitting /rbac/teams
// with no datasource header and no assigned tenant (a real shape — admins
// can hold zero tenant claims) must get a clean 400, not the raw Postgres
// 22P02 the original bug produced.
func TestListTeams_NoTenantContext(t *testing.T) {
	h := &RBACHandlers{
		securityDeps: handlers.SecurityContextDeps{Resolver: noTenantResolver{}},
	}

	req := httptest.NewRequest(http.MethodGet, "/rbac/teams", nil)
	authInfo := security.AuthInfo{
		UserID:    "admin-1",
		Roles:     []string{"global_admin"},
		TenantIDs: []string{}, // no tenant claim at all — plausible for a global admin
	}
	req = req.WithContext(security.WithAuthInfo(req.Context(), authInfo))

	rec := httptest.NewRecorder()
	h.listTeams(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 (tenant context required), got %d: %s", rec.Code, rec.Body.String())
	}
}
