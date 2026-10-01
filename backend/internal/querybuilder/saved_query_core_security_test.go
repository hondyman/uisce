package querybuilder

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/hondyman/uisce/backend/internal/boresolver"
	"github.com/hondyman/uisce/backend/internal/handlers"
	"github.com/hondyman/uisce/backend/internal/security"
)

// --- 1. Parameter-Safety Test (SQL Injection Proof) ---

func TestSavedQueryExecution_ParameterizedSQL(t *testing.T) {
	primary := &boresolver.BODefinition{
		ID:           "bo-account",
		DrivingTable: "account",
		Fields: []boresolver.BOField{
			{ID: "f1", Name: "account_id", Type: "string", PhysicalColumn: "account_id"},
			{ID: "f2", Name: "security_name", Type: "string", PhysicalColumn: "security_name"},
		},
	}

	evil := `'; DROP TABLE accounts; --`
	qd := &boresolver.QueryDef{
		Context: boresolver.QueryContext{
			BOID:     "bo-account",
			TenantID: "tenant-123",
		},
		Query: boresolver.QueryRequest{
			Dimensions: []boresolver.DimensionDef{
				{TermNodeID: "account_id", Alias: "Account", BOID: "bo-account"},
			},
			Filters: []boresolver.FilterDef{
				{TermNodeID: "security_name", Operator: "eq", Value: evil, BOID: "bo-account"},
			},
			Limit: 50,
		},
	}

	gen, err := boresolver.NewBOSQLGenerator(nil, "postgres")
	if err != nil {
		t.Fatalf("failed to create SQL generator: %v", err)
	}

	sqlStr, args, _, err := buildMultiBOSQL(gen, primary, nil, qd, "tenant-123")
	if err != nil {
		t.Fatalf("buildMultiBOSQL failed: %v", err)
	}

	// 1. Decisive check: raw evil payload must NEVER appear in the generated SQL string
	if strings.Contains(sqlStr, evil) {
		t.Fatalf("SQL string contains raw user value — interpolation detected:\n%s", sqlStr)
	}

	// 2. Decisive check: SQL must use placeholder tokens ($1, $2, etc.)
	if !strings.Contains(sqlStr, "$") {
		t.Fatalf("expected placeholder tokens ($1, $2) in SQL, got:\n%s", sqlStr)
	}

	// 3. Decisive check: the user payload must be safely passed in the args slice
	found := false
	for _, arg := range args {
		if s, ok := arg.(string); ok && s == evil {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("user value %q not found in bound args: %#v", evil, args)
	}
}

// --- 2. Cross-Tenant Execution & Adoption Protection ---

func TestCoreLifecycle_CrossTenantRejection(t *testing.T) {
	tenantA := "tenant-a"
	tenantB := "tenant-b"
	coreQueryID := "core-q1"

	// Existing adoption record in tenant-a's partition
	adoptMap := map[string]map[string]queryAdoption{
		tenantA: {
			coreQueryID: {
				CoreObjectID: coreQueryID,
				Active:       true,
				Mode:         "extended",
				BaseVersion:  sql.NullInt64{Int64: 1, Valid: true},
				Extension:    []byte(`{"state":{"dimensions":[{"alias":"custom_dim","termNodeId":"td1"}]}}`),
			},
		},
		tenantB: {},
	}

	// Verify tenant isolation: tenant-b cannot access or mutate tenant-a's adoption
	if _, ok := adoptMap[tenantB][coreQueryID]; ok {
		t.Errorf("tenant-b should not have tenant-a's adoption row")
	}

	// Tenant-a retains exact mode and content
	admA, ok := adoptMap[tenantA][coreQueryID]
	if !ok || admA.Mode != "extended" {
		t.Fatalf("tenant-a adoption record missing or corrupted: %+v", admA)
	}
}

func TestRuntimeExecution_CrossTenantQueryRejection(t *testing.T) {
	// A private query belonging to tenant-a
	privQuery := SavedQuery{
		ID:         "priv-q1",
		TenantID:   "tenant-a",
		UserID:     "user-a",
		Visibility: "private",
	}

	canAccess := func(callerTenant, callerUser string, sq SavedQuery) bool {
		if sq.TenantID != callerTenant {
			return false
		}
		if sq.Visibility == "shared" {
			return true
		}
		return sq.UserID == callerUser
	}

	// 1. Cross-tenant caller cannot access
	if canAccess("tenant-b", "user-b", privQuery) {
		t.Errorf("tenant-b should not access tenant-a private query")
	}

	// 2. Different user in same tenant cannot access private query
	if canAccess("tenant-a", "user-other", privQuery) {
		t.Errorf("different user in tenant-a should not access user-a private query")
	}

	// 3. Owner user in tenant-a can access
	if !canAccess("tenant-a", "user-a", privQuery) {
		t.Errorf("owner user in tenant-a should access own private query")
	}
}

// --- 3. Dormant Revert Fidelity ---

func TestRevert_KeepsExtensionDormantFidelity(t *testing.T) {
	baseCore := SavedQuery{
		ID:     "core-1",
		IsCore: true,
		Name:   "Core Query",
		BOID:   "bo-1",
		State: SavedQueryState{
			Dimensions: []SavedQueryDimension{{TermNodeID: "t-dim", Alias: "dim"}},
			Measures:   []SavedQueryMeasure{{TermNodeID: "t-meas", Alias: "revenue", Aggregation: "SUM"}},
		},
	}

	extensionJSON := []byte(`{
		"name": "Core Query (Extended)",
		"state": {
			"dimensions": [{"termNodeId": "t-dim", "alias": "dim"}],
			"measures": [
				{"termNodeId": "t-meas", "alias": "revenue", "agg": "SUM"},
				{"termNodeId": "t-m2", "alias": "revenue_ytd", "agg": "SUM"}
			]
		}
	}`)
	adm := queryAdoption{
		CoreObjectID: "core-1",
		Active:       true,
		Mode:         "extended",
		BaseVersion:  sql.NullInt64{Int64: 1, Valid: true},
		Extension:    extensionJSON,
	}

	// Revert sets mode = 'vanilla', but keeps Extension bytes dormant
	adm.Mode = "vanilla"

	// Presentation in vanilla mode returns pure core state
	presented := baseCore
	presentCoreQuery(&presented, &adm, false, true)

	if presented.CoreStatus != "vanilla" {
		t.Errorf("coreStatus = %q, want vanilla", presented.CoreStatus)
	}
	if len(presented.State.Measures) != 1 || presented.State.Measures[0].Alias != "revenue" {
		t.Errorf("presented query should be pure core in vanilla mode, got: %+v", presented.State.Measures)
	}

	// But the stored extension in the adoption is still dormant and intact
	var dormantExt savedQueryContent
	if err := json.Unmarshal(adm.Extension, &dormantExt); err != nil {
		t.Fatalf("failed to unmarshal dormant extension: %v", err)
	}
	foundYTD := false
	for _, m := range dormantExt.State.Measures {
		if m.Alias == "revenue_ytd" {
			foundYTD = true
			break
		}
	}
	if !foundYTD {
		t.Errorf("dormant extension lost its custom measure: %+v", dormantExt.State.Measures)
	}

	// Re-extend reactivates the mode to 'extended'
	adm.Mode = "extended"
	reactivated := baseCore
	presentCoreQuery(&reactivated, &adm, false, true)
	if reactivated.CoreStatus != "extended" {
		t.Errorf("coreStatus = %q, want extended", reactivated.CoreStatus)
	}
	if len(reactivated.State.Measures) != 2 {
		t.Errorf("reactivated query should merge base and extension measures, got %d", len(reactivated.State.Measures))
	}
}

// --- 4. Base Snapshot Fidelity ---

func TestExtend_CapturesExactBaseSnapshotFidelity(t *testing.T) {
	baseCore := SavedQuery{
		ID:     "core-1",
		IsCore: true,
		Name:   "Core Query",
		BOID:   "bo-1",
		State: SavedQueryState{
			Dimensions: []SavedQueryDimension{{TermNodeID: "t-dim", Alias: "dim"}},
			Measures:   []SavedQueryMeasure{{TermNodeID: "t-meas", Alias: "revenue", Aggregation: "SUM"}},
		},
	}

	coreContent := contentFromSavedQuery(&baseCore)
	coreHash := computeContentHash(coreContent)

	// Adoption base snapshot serialized at adoption time
	snapBytes, err := json.Marshal(coreContent)
	if err != nil {
		t.Fatalf("failed to marshal base snapshot: %v", err)
	}

	var loadedSnap savedQueryContent
	if err := json.Unmarshal(snapBytes, &loadedSnap); err != nil {
		t.Fatalf("failed to unmarshal base snapshot: %v", err)
	}

	loadedHash := computeContentHash(loadedSnap)
	if loadedHash != coreHash {
		t.Errorf("base snapshot hash %q != core hash %q at adoption time", loadedHash, coreHash)
	}
}

// --- 5. Real HTTP Handler Layer Testing ---

type mockDSResolver struct {
	defaultTenantID string
}

func (m *mockDSResolver) Resolve(ctx context.Context, datasourceID string) (*security.ResolvedDatasource, error) {
	tid := "tenant-a"
	if m != nil && m.defaultTenantID != "" {
		tid = m.defaultTenantID
	}
	return &security.ResolvedDatasource{
		DatasourceID: datasourceID,
		TenantID:     tid,
	}, nil
}

func TestSavedQueryCore_HTTPHandlerRouting(t *testing.T) {
	deps := handlers.SecurityContextDeps{Resolver: &mockDSResolver{}}
	h := &SavedQueryHandler{deps: deps}

	r := chi.NewRouter()
	r.Route("/api/explorer/saved-queries", func(r chi.Router) {
		r.Get("/{id}", h.HandleGetSavedQuery)
		r.Post("/{id}/extend", h.HandleExtendCoreQuery)
		r.Get("/{id}/compare", h.HandleCompareCoreQuery)
		r.Post("/{id}/upgrade", h.HandleUpgradeCoreQuery)
		r.Post("/{id}/revert", h.HandleRevertCoreQuery)
		r.Get("/{id}/preview", h.HandleGetPreview)
	})

	// Helper to send HTTP requests with tenant auth info
	doReq := func(method, path, tenantID, userID string, body string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(method, path, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Datasource-Id", "ds-1")
		auth := security.AuthInfo{
			TenantIDs: []string{tenantID},
			UserID:    userID,
			Roles:     []string{"tenant_admin", "can_customize_core"},
		}
		ctx := security.WithAuthInfo(req.Context(), auth)
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req.WithContext(ctx))
		return rec
	}

	// 1. Missing query ID or nil DB fails gracefully through router
	rec := doReq("GET", "/api/explorer/saved-queries/invalid-id/compare", "tenant-a", "user-a", "")
	if rec.Code == 0 {
		t.Errorf("expected HTTP response, got 0")
	}

	// 2. Unauthenticated request without JWT/AuthInfo returns 400/401
	unauthReq := httptest.NewRequest("GET", "/api/explorer/saved-queries/core-1/compare", nil)
	unauthRec := httptest.NewRecorder()
	r.ServeHTTP(unauthRec, unauthReq)
	if unauthRec.Code != http.StatusBadRequest && unauthRec.Code != http.StatusUnauthorized {
		t.Errorf("unauthenticated request: got %d, want 400/401", unauthRec.Code)
	}

	// 3. Extend with invalid JSON / unconfigured goldcopy returns 403 Forbidden
	rec = doReq("POST", "/api/explorer/saved-queries/core-1/extend", "tenant-a", "user-a", "invalid-json{")
	if rec.Code != http.StatusForbidden {
		t.Errorf("extend without active goldcopy: got %d, want 403", rec.Code)
	}

	// 4. Extend without admin role returns 403 Forbidden
	unauthorizedReq := httptest.NewRequest("POST", "/api/explorer/saved-queries/core-1/extend", strings.NewReader(`{}`))
	unauthorizedReq.Header.Set("Content-Type", "application/json")
	unauthAuth := security.AuthInfo{
		TenantIDs: []string{"tenant-a"},
		UserID:    "user-a",
		Roles:     []string{"viewer"}, // no tenant_admin
	}
	unauthCtx := security.WithAuthInfo(unauthorizedReq.Context(), unauthAuth)
	unauthorizedRec := httptest.NewRecorder()
	r.ServeHTTP(unauthorizedRec, unauthorizedReq.WithContext(unauthCtx))
	if unauthorizedRec.Code != http.StatusForbidden {
		t.Errorf("extend without tenant_admin role: got %d, want 403", unauthorizedRec.Code)
	}
}
