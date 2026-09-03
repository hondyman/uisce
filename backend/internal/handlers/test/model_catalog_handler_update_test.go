package handlers_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"
	"time"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/hondyman/uisce/backend/internal/handlers"
	"github.com/hondyman/uisce/backend/internal/security"
)

// testTenantID matches the tenant_id every test below drives through the
// ?tenant_id= query param — SecurityContextFromRequest resolves
// secCtx.TenantID from the resolver's response, not the raw query string,
// so the resolver must echo it back for the handler's SQL args to match
// what the test's sqlmock expectations assert.
const testTenantID = "11111111-1111-1111-1111-111111111111"

// echoDatasourceResolver satisfies handlers.SecurityContextFromRequest's
// non-nil Resolver requirement, echoing back whatever datasourceID it's
// asked to resolve under testTenantID — sufficient for UpdateModel, which
// only needs secCtx.TenantID/DatasourceID to round-trip the query params
// the test supplied.
type echoDatasourceResolver struct{}

func (echoDatasourceResolver) Resolve(ctx context.Context, datasourceID string) (*security.ResolvedDatasource, error) {
	return &security.ResolvedDatasource{TenantID: testTenantID, DatasourceID: datasourceID}, nil
}

func (echoDatasourceResolver) ResolveBindingDatasource(ctx context.Context, tenantID, alphaProductID, alphaDatasourceID string) (string, error) {
	return "", nil
}

func testSecurityDeps() handlers.SecurityContextDeps {
	return handlers.SecurityContextDeps{Resolver: echoDatasourceResolver{}}
}

// withTestAuth injects AuthInfo the way AuthContextMiddleware would in
// production, so SecurityContextFromRequest's auth check passes.
func withTestAuth(req *http.Request) *http.Request {
	authInfo := security.AuthInfo{
		UserID:    "test-user",
		Roles:     []string{"admin"},
		TenantIDs: []string{testTenantID},
	}
	return req.WithContext(security.WithAuthInfo(req.Context(), authInfo))
}

// We assert UpdateModel sets published_at on publish and clears it on draft by driving the HTTP handler.
func TestUpdateModel_StatusPublishAndDraft(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	defer db.Close()

	h := handlers.NewModelCatalogHandler(db, testSecurityDeps())
	r := chi.NewRouter()
	h.RegisterRoutes(r)

	tenantID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	datasourceID := uuid.MustParse("22222222-2222-2222-2222-222222222222")
	modelID := uuid.MustParse("2f1f1f1f-2f1f-2f1f-2f1f-2f1f2f1f2f1f")

	// Expect UPDATE for publish
	mock.ExpectQuery(regexp.QuoteMeta("UPDATE fabric_defn")).
		WithArgs("published", sqlmock.AnyArg(), sqlmock.AnyArg(), modelID, tenantID, datasourceID).
		WillReturnRows(sqlmock.NewRows([]string{"id", "tenant_id", "tenant_datasource_id", "model_key", "version", "status", "is_current", "title", "description", "source_config", "resolved_config", "created_by", "created_at", "published_at", "checksum_sha256", "updated_at"}).
			AddRow(modelID.String(), tenantID.String(), datasourceID.String(), "model_key", 1, "published", true, nil, nil, []byte(`{}`), []byte(`{}`), uuid.New().String(), time.Now(), time.Now(), []byte{0x01}, time.Now()))

	// Drive request to publish (handler reads tenant_id and datasource_id from query string, model_id from path)
	body := map[string]any{"status": "published"}
	buf, _ := json.Marshal(body)
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPatch, "/models/"+modelID.String()+"?tenant_id="+tenantID.String()+"&datasource_id="+datasourceID.String(), bytes.NewReader(buf))
	req.Header.Set("Content-Type", "application/json")
	// Set chi route param for model_id
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("model_id", modelID.String())
	req = withTestAuth(req)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	r.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("publish: expected 200, got %d: %s", rr.Code, rr.Body.String())
	}

	// Expect UPDATE for draft (published_at NULL)
	mock.ExpectQuery(regexp.QuoteMeta("UPDATE fabric_defn")).
		WithArgs("draft", sqlmock.AnyArg(), modelID, tenantID, datasourceID).
		WillReturnRows(sqlmock.NewRows([]string{"id", "tenant_id", "tenant_datasource_id", "model_key", "version", "status", "is_current", "title", "description", "source_config", "resolved_config", "created_by", "created_at", "published_at", "checksum_sha256", "updated_at"}).
			AddRow(modelID.String(), tenantID.String(), datasourceID.String(), "model_key", 1, "draft", true, nil, nil, []byte(`{}`), []byte(`{}`), uuid.New().String(), time.Now(), nil, []byte{0x01}, time.Now()))

	// Drive request to draft
	body2 := map[string]any{"status": "draft"}
	buf2, _ := json.Marshal(body2)
	rr2 := httptest.NewRecorder()
	req2 := httptest.NewRequest(http.MethodPatch, "/models/"+modelID.String()+"?tenant_id="+tenantID.String()+"&datasource_id="+datasourceID.String(), bytes.NewReader(buf2))
	req2.Header.Set("Content-Type", "application/json")
	rctx2 := chi.NewRouteContext()
	rctx2.URLParams.Add("model_id", modelID.String())
	req2 = withTestAuth(req2)
	req2 = req2.WithContext(context.WithValue(req2.Context(), chi.RouteCtxKey, rctx2))
	r.ServeHTTP(rr2, req2)
	if rr2.Code != http.StatusOK {
		t.Fatalf("draft: expected 200, got %d: %s", rr2.Code, rr2.Body.String())
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

// We assert UpdateModel computes checksum_sha256 when resolved_config is supplied by driving the handler.
func TestUpdateModel_ChecksumOnResolvedConfig(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	defer db.Close()

	h := handlers.NewModelCatalogHandler(db, testSecurityDeps())
	r := chi.NewRouter()
	h.RegisterRoutes(r)

	tenantID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	datasourceID := uuid.MustParse("22222222-2222-2222-2222-222222222222")
	modelID := uuid.MustParse("2f1f1f1f-2f1f-2f1f-2f1f-2f1f2f1f2f1f")

	// Expect the update to include checksum_sha256 set
	mock.ExpectQuery(regexp.QuoteMeta("UPDATE fabric_defn")).
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), modelID, tenantID, datasourceID).
		WillReturnRows(sqlmock.NewRows([]string{"id", "tenant_id", "tenant_datasource_id", "model_key", "version", "status", "is_current", "title", "description", "source_config", "resolved_config", "created_by", "created_at", "published_at", "checksum_sha256", "updated_at"}).
			AddRow(modelID.String(), tenantID.String(), datasourceID.String(), "model_key", 1, "draft", true, nil, nil, []byte(`{}`), []byte(`{"a":1}`), uuid.New().String(), time.Now(), nil, []byte{0xAA}, time.Now()))

	// Drive request
	body := map[string]any{"resolved_config": map[string]any{"a": 1}}
	buf, _ := json.Marshal(body)
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPatch, "/models/"+modelID.String()+"?tenant_id="+tenantID.String()+"&datasource_id="+datasourceID.String(), bytes.NewReader(buf))
	req.Header.Set("Content-Type", "application/json")
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("model_id", modelID.String())
	req = withTestAuth(req)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	r.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}
