package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/go-chi/chi/v5"
	"github.com/hondyman/semlayer/backend/internal/security"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTenantHandler(t *testing.T) (*TenantAccessHandlers, sqlmock.Sqlmock, *chi.Mux) {
	t.Helper()
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { db.Close() })

	handler := NewTenantAccessHandlers(db)
	r := chi.NewRouter()
	handler.RegisterRoutes(r)
	return handler, mock, r
}

func expectTenantByID(mock sqlmock.Sqlmock, tenantID string) {
	rows := sqlmock.NewRows([]string{
		"id", "display_name", "name", "description", "is_active", "gold_copy", "region", "allowed_regions",
	}).AddRow(tenantID, "InvestCo", "investco", nil, true, false, "us-east", []byte("[]"))
	mock.ExpectQuery("SELECT id, COALESCE").
		WithArgs(tenantID).
		WillReturnRows(rows)

	mock.ExpectQuery("SELECT id, COALESCE").
		WithArgs(tenantID).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "display_name", "instance_name", "description", "is_active", "url", "tenant_id",
		}))

	mock.ExpectQuery("SELECT tp.id, tp.version").
		WithArgs(tenantID).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "version", "alpha_product_id", "ap_id", "product_name", "product_code", "ap_is_active",
		}))

	mock.ExpectQuery("SELECT tpd.id, COALESCE").
		WithArgs(tenantID).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "is_active", "source_name", "tenant_product_id", "tenant_instance_id",
		}))
}

func expectAllTenants(mock sqlmock.Sqlmock, tenantIDs []string) {
	rows := sqlmock.NewRows([]string{
		"id", "display_name", "name", "description", "is_active", "gold_copy", "region", "allowed_regions",
	})
	for _, id := range tenantIDs {
		rows.AddRow(id, id, id, nil, true, false, "us-east", []byte("[]"))
	}
	mock.ExpectQuery("SELECT id, COALESCE").
		WillReturnRows(rows)

	mock.ExpectQuery("SELECT id, COALESCE").
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "display_name", "instance_name", "description", "is_active", "url", "tenant_id",
		}))

	mock.ExpectQuery("SELECT tp.id, tp.version").
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "version", "alpha_product_id", "ap_id", "product_name", "product_code", "ap_is_active",
		}))

	mock.ExpectQuery("SELECT tpd.id, COALESCE").
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "is_active", "source_name", "tenant_product_id", "tenant_instance_id",
		}))
}

func TestListAccessibleTenants_ProfessionalServices_LeaseScoped(t *testing.T) {
	_, mock, r := setupTenantHandler(t)
	expectTenantByID(mock, "investco")

	req := httptest.NewRequest(http.MethodGet, "/tenants/accessible", nil)
	req.Header.Set("X-User-ID", "ali.g")
	req.Header.Set("X-User-Role", "professional_services")
	req.Header.Set("X-Tenant-ID", "investco")

	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)

	var tenants []TenantResponse
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &tenants))
	require.Len(t, tenants, 1)
	assert.Equal(t, "investco", tenants[0].ID)
}

func TestListAccessibleTenants_Helpdesk_LeaseScoped(t *testing.T) {
	_, mock, r := setupTenantHandler(t)
	expectTenantByID(mock, "investco")

	req := httptest.NewRequest(http.MethodGet, "/tenants/accessible", nil)
	req.Header.Set("X-User-ID", "support.user")
	req.Header.Set("X-User-Role", "helpdesk")
	req.Header.Set("X-Tenant-ID", "investco")

	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)

	var tenants []TenantResponse
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &tenants))
	require.Len(t, tenants, 1)
	assert.Equal(t, "investco", tenants[0].ID)
}

func TestListAccessibleTenants_GlobalAdmin_SeesAll(t *testing.T) {
	_, mock, r := setupTenantHandler(t)
	expectAllTenants(mock, []string{"investco", "acmecorp", "globex"})

	req := httptest.NewRequest(http.MethodGet, "/tenants/accessible", nil)
	req.Header.Set("X-User-ID", "jim.g")
	req.Header.Set("X-User-Role", "global_admin")

	ctx := security.WithAuthInfo(req.Context(), security.AuthInfo{
		UserID:        "jim.g",
		IsGlobalAdmin: true,
		Roles:         []string{"global_admin"},
	})
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)

	var tenants []TenantResponse
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &tenants))
	assert.Len(t, tenants, 3)
}

func TestListAccessibleTenants_CoreAdmin_SeesAll(t *testing.T) {
	_, mock, r := setupTenantHandler(t)
	expectAllTenants(mock, []string{"investco", "acmecorp"})

	req := httptest.NewRequest(http.MethodGet, "/tenants/accessible", nil)
	req.Header.Set("X-User-ID", "root.ops")
	req.Header.Set("X-User-Role", "core_admin")
	req.Header.Set("X-Is-Core-Admin", "true")

	ctx := security.WithAuthInfo(req.Context(), security.AuthInfo{
		UserID:        "root.ops",
		IsGlobalAdmin: true,
		Roles:         []string{"core_admin"},
	})
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)

	var tenants []TenantResponse
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &tenants))
	assert.Len(t, tenants, 2)
}

func TestListAccessibleTenants_TenantUser_SeesAssigned(t *testing.T) {
	_, mock, r := setupTenantHandler(t)

	// public.user_tenant lookup
	mock.ExpectQuery("SELECT tenant_id FROM public\\.user_tenant WHERE user_id =").
		WithArgs("tenant.user").
		WillReturnRows(sqlmock.NewRows([]string{"tenant_id"}).AddRow("investco"))

	expectTenantByID(mock, "investco")

	req := httptest.NewRequest(http.MethodGet, "/tenants/accessible", nil)
	req.Header.Set("X-User-ID", "tenant.user")

	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)

	var tenants []TenantResponse
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &tenants))
	require.Len(t, tenants, 1)
	assert.Equal(t, "investco", tenants[0].ID)
}

func TestListAccessibleTenants_NoAssignment_Empty(t *testing.T) {
	_, mock, r := setupTenantHandler(t)

	mock.ExpectQuery("SELECT tenant_id FROM public\\.user_tenant WHERE user_id =").
		WithArgs("unassigned.user").
		WillReturnRows(sqlmock.NewRows([]string{"tenant_id"}))

	mock.ExpectQuery("SELECT tenant_id FROM users WHERE id =").
		WithArgs("unassigned.user").
		WillReturnRows(sqlmock.NewRows([]string{"tenant_id"}).AddRow(nil))

	req := httptest.NewRequest(http.MethodGet, "/tenants/accessible", nil)
	req.Header.Set("X-User-ID", "unassigned.user")

	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)

	var tenants []TenantResponse
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &tenants))
	assert.Empty(t, tenants)
}
