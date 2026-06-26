package api_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/hondyman/semlayer/backend/internal/api"
)

// Test that the full chi router initializes with middleware and that session auth middleware
// will attempt to query the sessions table. We use sqlmock to verify the query is executed.
func TestSetupRouter_WithSessionAuthAndWsToken(t *testing.T) {
	t.Setenv("DISABLE_BACKGROUND_JOBS", "true")

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	mock.MatchExpectationsInOrder(false)

	// Allow background initialization queries from services (BundleService, CollaborationService)
	// These happen during SetupRouter
	mock.ExpectQuery("SELECT .* FROM private_markets_bundles").WillReturnRows(sqlmock.NewRows([]string{"bundle_id"}))
	mock.ExpectQuery("SELECT .* FROM access_control_policies").WillReturnRows(sqlmock.NewRows([]string{"id"}))
	mock.ExpectQuery("INSERT INTO access_control_policies").WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("00000000-0000-0000-0000-000000000000"))
	mock.ExpectQuery("INSERT INTO access_control_policies").WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("00000000-0000-0000-0000-000000000000"))
	mock.ExpectQuery("SELECT .* FROM access_control_policies").WillReturnRows(sqlmock.NewRows([]string{"id"}))

	// Expect a sessions query in session middleware when auth cookie present
	mock.ExpectQuery("SELECT user_id, expires_at, is_active FROM private_markets_sessions").
		WithArgs("token123").
		WillReturnRows(sqlmock.NewRows([]string{"user_id", "expires_at", "is_active"}).
			AddRow("1", time.Now().Add(24*time.Hour), true))

	// The session middleware then fetches user profile
	mock.ExpectQuery("SELECT id, email, name, role, organization, permissions, is_core_admin, is_active, tenant_id FROM public.users").
		WithArgs("1").
		WillReturnRows(sqlmock.NewRows([]string{"id", "email", "name", "role", "organization", "permissions", "is_core_admin", "is_active", "tenant_id"}).
			AddRow("1", "test@unit.com", "Test User", "admin", "uisce", "[]", true, true, nil))

	// Middleware sets session variables for RLS
	mock.ExpectExec("SET LOCAL app.is_global_admin = 'true'").WillReturnResult(sqlmock.NewResult(0, 0))

	// The /api/roles handler queries roles from iam schema
	mock.ExpectQuery("SELECT role_id, tenant_id, role_name, description, is_global_admin, created_at FROM iam.roles").
		WillReturnRows(sqlmock.NewRows([]string{"role_id", "tenant_id", "role_name", "description", "is_global_admin", "created_at"}).
			AddRow("role1", "tenant1", "Admin", "Admin Role", true, time.Now().Format(time.RFC3339)))

	router := api.SetupRouter(db, nil, nil, nil, nil, nil, nil, &mockResolver{}, nil)

	// Start httptest server
	ts := httptest.NewServer(router)
	defer ts.Close()

	// Make a request to an API path which will run through middleware
	req, _ := http.NewRequest("GET", ts.URL+"/api/roles/", nil)
	// Add a cookie to trigger session handling
	req.AddCookie(&http.Cookie{Name: "session_token", Value: "token123"})

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	// We expect either 200 or 404 depending on handler; ensure the DB expectation was met
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sqlmock expectations not met: %v", err)
	}

	// Test WS token endpoint (POST /api/ws/token) - ensure handler responds
	body := strings.NewReader(`{"jobId":"job1","purpose":"profiler","ttl_seconds":60}`)
	resp2, err := http.Post(ts.URL+"/api/ws/token", "application/json", body)
	if err != nil {
		t.Fatalf("ws token request failed: %v", err)
	}
	defer resp2.Body.Close()
	if resp2.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 got %d", resp2.StatusCode)
	}
	var tokenResp map[string]string
	if err := json.NewDecoder(resp2.Body).Decode(&tokenResp); err != nil {
		t.Fatalf("failed decode token resp: %v", err)
	}
	if tokenResp["token"] == "" {
		t.Fatalf("expected token in response")
	}
}
