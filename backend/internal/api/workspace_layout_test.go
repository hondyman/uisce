package api

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	jwtmiddleware "github.com/hondyman/uisce/libs/jwt-middleware"
)

func withAuthClaims(r *http.Request, userID, tenantID string) *http.Request {
	claims := &jwtmiddleware.JWTClaims{
		UserID:   userID,
		TenantID: tenantID,
	}
	ctx := context.WithValue(r.Context(), jwtmiddleware.ClaimsContextKey, claims)
	return r.WithContext(ctx)
}

func TestWorkspaceLayout_Unauthorized(t *testing.T) {
	db, _, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	handler := NewWorkspaceLayoutHandler(db)

	// Unauthenticated GET
	req := httptest.NewRequest(http.MethodGet, "/api/user/preferences/workspace-layout", nil)
	rec := httptest.NewRecorder()
	handler.GetLayout(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 Unauthorized, got %d", rec.Code)
	}

	// Unauthenticated POST
	body := bytes.NewBufferString(`{"layout_data":{"dockviewLayout":{}}}`)
	reqPost := httptest.NewRequest(http.MethodPost, "/api/user/preferences/workspace-layout", body)
	recPost := httptest.NewRecorder()
	handler.SaveLayout(recPost, reqPost)

	if recPost.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 Unauthorized, got %d", recPost.Code)
	}
}

func TestWorkspaceLayout_NotFound(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	handler := NewWorkspaceLayoutHandler(db)
	tenantID := uuid.New().String()
	userID := "usr_trader_123"

	mock.ExpectQuery(`SELECT (.+) FROM public\.user_workspace_layouts WHERE tenant_id = \$1 AND user_id = \$2 AND profile_name = \$3`).
		WithArgs(tenantID, userID, "default").
		WillReturnError(sql.ErrNoRows)

	req := httptest.NewRequest(http.MethodGet, "/api/user/preferences/workspace-layout", nil)
	req = withAuthClaims(req, userID, tenantID)
	rec := httptest.NewRecorder()

	handler.GetLayout(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404 Not Found, got %d body=%s", rec.Code, rec.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet mock expectations: %v", err)
	}
}

func TestWorkspaceLayout_PayloadTooLarge(t *testing.T) {
	db, _, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	handler := NewWorkspaceLayoutHandler(db)
	tenantID := uuid.New().String()
	userID := "usr_trader_123"

	// Create payload > 1MB (1.1MB)
	bigJSON := `{"layout_data":{"blob":"` + strings.Repeat("x", 1100000) + `"}}`
	req := httptest.NewRequest(http.MethodPost, "/api/user/preferences/workspace-layout", strings.NewReader(bigJSON))
	req = withAuthClaims(req, userID, tenantID)
	rec := httptest.NewRecorder()

	handler.SaveLayout(rec, req)

	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("expected 413 Payload Too Large, got %d", rec.Code)
	}
}

func TestWorkspaceLayout_SaveAndGet_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	handler := NewWorkspaceLayoutHandler(db)
	tenantID := uuid.New().String()
	userID := "usr_trader_999"
	layoutID := uuid.New().String()
	now := time.Now()

	sampleLayout := `{"dockviewLayout":{"type":"grid"},"detachedWindows":[]}`

	// 1. Test SaveLayout (UPSERT)
	saveRows := sqlmock.NewRows([]string{
		"id", "tenant_id", "user_id", "profile_name", "schema_version", "layout_data", "created_at", "updated_at",
	}).AddRow(layoutID, tenantID, userID, "default", "v1", []byte(sampleLayout), now, now)

	mock.ExpectQuery(`INSERT INTO public\.user_workspace_layouts`).
		WithArgs(tenantID, userID, "default", "v1", []byte(sampleLayout)).
		WillReturnRows(saveRows)

	reqSave := httptest.NewRequest(http.MethodPost, "/api/user/preferences/workspace-layout", bytes.NewBufferString(`{
		"profile_name": "default",
		"schema_version": "v1",
		"layout_data": {"dockviewLayout":{"type":"grid"},"detachedWindows":[]}
	}`))
	reqSave = withAuthClaims(reqSave, userID, tenantID)
	recSave := httptest.NewRecorder()

	handler.SaveLayout(recSave, reqSave)

	if recSave.Code != http.StatusOK {
		t.Fatalf("expected 200 OK on save, got %d body=%s", recSave.Code, recSave.Body.String())
	}

	var savedResp WorkspaceLayoutResponse
	if err := json.Unmarshal(recSave.Body.Bytes(), &savedResp); err != nil {
		t.Fatalf("failed to parse save response: %v", err)
	}
	if savedResp.UserID != userID || savedResp.TenantID != tenantID || savedResp.ProfileName != "default" {
		t.Fatalf("unexpected saved response: %+v", savedResp)
	}

	// 2. Test GetLayout
	getRows := sqlmock.NewRows([]string{
		"id", "tenant_id", "user_id", "profile_name", "schema_version", "layout_data", "created_at", "updated_at",
	}).AddRow(layoutID, tenantID, userID, "default", "v1", []byte(sampleLayout), now, now)

	mock.ExpectQuery(`SELECT (.+) FROM public\.user_workspace_layouts WHERE tenant_id = \$1 AND user_id = \$2 AND profile_name = \$3`).
		WithArgs(tenantID, userID, "default").
		WillReturnRows(getRows)

	reqGet := httptest.NewRequest(http.MethodGet, "/api/user/preferences/workspace-layout?profile=default", nil)
	reqGet = withAuthClaims(reqGet, userID, tenantID)
	recGet := httptest.NewRecorder()

	handler.GetLayout(recGet, reqGet)

	if recGet.Code != http.StatusOK {
		t.Fatalf("expected 200 OK on get, got %d body=%s", recGet.Code, recGet.Body.String())
	}

	var getResp WorkspaceLayoutResponse
	if err := json.Unmarshal(recGet.Body.Bytes(), &getResp); err != nil {
		t.Fatalf("failed to parse get response: %v", err)
	}
	if string(getResp.LayoutData) != sampleLayout {
		t.Fatalf("expected layout_data %s, got %s", sampleLayout, string(getResp.LayoutData))
	}

	// 3. Test ListProfiles
	listRows := sqlmock.NewRows([]string{
		"id", "profile_name", "schema_version", "updated_at",
	}).AddRow(layoutID, "default", "v1", now)

	mock.ExpectQuery(`SELECT id, profile_name, schema_version, updated_at FROM public\.user_workspace_layouts WHERE tenant_id = \$1 AND user_id = \$2`).
		WithArgs(tenantID, userID).
		WillReturnRows(listRows)

	reqList := httptest.NewRequest(http.MethodGet, "/api/user/preferences/workspace-layout/profiles", nil)
	reqList = withAuthClaims(reqList, userID, tenantID)
	recList := httptest.NewRecorder()

	handler.ListProfiles(recList, reqList)

	if recList.Code != http.StatusOK {
		t.Fatalf("expected 200 OK on list, got %d body=%s", recList.Code, recList.Body.String())
	}

	var profiles []WorkspaceLayoutProfileSummary
	if err := json.Unmarshal(recList.Body.Bytes(), &profiles); err != nil {
		t.Fatalf("failed to parse profiles: %v", err)
	}
	if len(profiles) != 1 || profiles[0].ProfileName != "default" {
		t.Fatalf("unexpected profiles: %+v", profiles)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet mock expectations: %v", err)
	}
}

func TestWorkspaceLayout_TenantAndUserIsolation(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	handler := NewWorkspaceLayoutHandler(db)

	tenantA := uuid.New().String()
	tenantB := uuid.New().String()
	userA := "usr_alice_1"
	userB := "usr_bob_2"
	profileName := "equities_desk"

	// 1. Cross-User Isolation (Same Tenant):
	// User B attempts to access User A's layout in the same tenant.
	// The DB query strictly includes user_id = $2, so User A's row is not returned.
	mock.ExpectQuery(`SELECT (.+) FROM public\.user_workspace_layouts WHERE tenant_id = \$1 AND user_id = \$2 AND profile_name = \$3`).
		WithArgs(tenantA, userB, profileName).
		WillReturnError(sql.ErrNoRows)

	reqUserB := httptest.NewRequest(http.MethodGet, "/api/user/preferences/workspace-layout?profile="+profileName, nil)
	reqUserB = withAuthClaims(reqUserB, userB, tenantA)
	recUserB := httptest.NewRecorder()

	handler.GetLayout(recUserB, reqUserB)

	if recUserB.Code != http.StatusNotFound {
		t.Fatalf("expected 404 Not Found for cross-user access, got %d", recUserB.Code)
	}

	// 2. Cross-Tenant Isolation (Same User ID across different tenants):
	// User A in Tenant B attempts to access the profile that exists in Tenant A.
	// The DB query strictly enforces tenant_id = $1, returning no rows.
	mock.ExpectQuery(`SELECT (.+) FROM public\.user_workspace_layouts WHERE tenant_id = \$1 AND user_id = \$2 AND profile_name = \$3`).
		WithArgs(tenantB, userA, profileName).
		WillReturnError(sql.ErrNoRows)

	reqTenantB := httptest.NewRequest(http.MethodGet, "/api/user/preferences/workspace-layout?profile="+profileName, nil)
	reqTenantB = withAuthClaims(reqTenantB, userA, tenantB)
	recTenantB := httptest.NewRecorder()

	handler.GetLayout(recTenantB, reqTenantB)

	if recTenantB.Code != http.StatusNotFound {
		t.Fatalf("expected 404 Not Found for cross-tenant access, got %d", recTenantB.Code)
	}

	// 3. Cross-Tenant Save Isolation:
	// Verify that saving a layout for User A in Tenant A binds tenantA and userA to the INSERT/UPSERT.
	sampleLayout := `{"dockviewLayout":{}}`
	mock.ExpectQuery(`INSERT INTO public\.user_workspace_layouts`).
		WithArgs(tenantA, userA, profileName, "v1", []byte(sampleLayout)).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "tenant_id", "user_id", "profile_name", "schema_version", "layout_data", "created_at", "updated_at",
		}).AddRow(uuid.New().String(), tenantA, userA, profileName, "v1", []byte(sampleLayout), time.Now(), time.Now()))

	reqSave := httptest.NewRequest(http.MethodPost, "/api/user/preferences/workspace-layout", bytes.NewBufferString(`{
		"profile_name": "`+profileName+`",
		"schema_version": "v1",
		"layout_data": {"dockviewLayout":{}}
	}`))
	reqSave = withAuthClaims(reqSave, userA, tenantA)
	recSave := httptest.NewRecorder()

	handler.SaveLayout(recSave, reqSave)

	if recSave.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", recSave.Code)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sqlmock expectations: %v", err)
	}
}

