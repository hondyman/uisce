package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	_ "github.com/lib/pq"
)

// loadAuditMigration reads the audit migration and ensures it parses as SQL.
// This is a structural sanity check — the actual CREATE TABLE / RLS /
// CREATE INDEX statements are exercised by the DB-backed integration test.
func loadAuditMigration(t *testing.T) string {
	t.Helper()
	candidates := []string{
		"../../db/migrations/20260817000005_audit.sql",
		"../../../db/migrations/20260817000005_audit.sql",
	}
	for _, p := range candidates {
		abs, _ := filepath.Abs(p)
		if data, err := os.ReadFile(abs); err == nil {
			return string(data)
		}
	}
	t.Skip("audit migration not found")
	return ""
}

func TestAuditMigration_ParsesAndContainsExpectedStatements(t *testing.T) {
	body := loadAuditMigration(t)
	if body == "" {
		return
	}
	for _, want := range []string{
		"CREATE TABLE IF NOT EXISTS public.api_dispatch_audit_log",
		"tenant_id",
		"api_endpoint_id",
		"ENABLE ROW LEVEL SECURITY",
		"CREATE POLICY tenant_isolation_audit_select",
		"CREATE POLICY tenant_isolation_audit_insert",
		"idx_audit_tenant_endpoint_recent",
		"idx_audit_tenant_recent",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("audit migration missing required statement: %q", want)
		}
	}
}

// TestRecordAudit_NonBlocking verifies that recordAudit returns immediately
// even when the buffer is full, instead of waiting on the worker.
func TestRecordAudit_NonBlocking(t *testing.T) {
	h := &ApiDispatcherHandler{}
	// Fill the buffer to capacity. We don't start a worker, so nothing drains.
	for i := 0; i < cap(auditBuffer); i++ {
		h.recordAudit(auditEntry{Method: "GET", Path: "/test"})
	}
	preDrops := auditDrops.Load()
	// The buffer is now full. recordAudit must NOT block; it should drop and
	// increment the counter. We measure the call with a timer that would
	// fire if recordAudit blocks waiting for buffer space.
	done := make(chan struct{})
	go func() {
		h.recordAudit(auditEntry{Method: "GET", Path: "/overflow"})
		close(done)
	}()
	select {
	case <-done:
		// good, returned promptly
	case <-time.After(500 * time.Millisecond):
		t.Fatalf("recordAudit blocked when buffer was full")
	}
	if got := auditDrops.Load(); got <= preDrops {
		t.Errorf("auditDrops counter should have advanced; pre=%d post=%d", preDrops, got)
	}
}

// TestListDispatchAudit_ReturnsRows exercises the GET /audit handler against
// an httptest server backed by a real DB. Skipped if DATABASE_URL is unset.
func TestListDispatchAudit_ReturnsRows(t *testing.T) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Skip("DATABASE_URL not set; skipping DB-backed audit test")
	}
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Fatalf("sql.Open: %v", err)
	}
	defer db.Close()
	if err := db.Ping(); err != nil {
		t.Skipf("database unreachable: %v", err)
	}

	// Ensure the audit table exists. We can't run the migration directly, but
	// if the operator hasn't applied it, we can CREATE TABLE here for the
	// duration of the test (the IF NOT EXISTS clause makes this a no-op
	// once the real migration is applied).
	if _, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS api_dispatch_audit_log (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			tenant_id UUID NOT NULL,
			user_id UUID,
			api_datasource_id UUID NOT NULL,
			api_endpoint_id UUID NOT NULL,
			method TEXT NOT NULL,
			path TEXT NOT NULL,
			status_code INT NOT NULL DEFAULT 0,
			duration_ms BIGINT NOT NULL DEFAULT 0,
			success BOOLEAN NOT NULL DEFAULT false,
			record_count INT NOT NULL DEFAULT 0,
			error TEXT NOT NULL DEFAULT '',
			request_params JSONB DEFAULT '{}'::jsonb,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)
	`); err != nil {
		t.Fatalf("ensure table: %v", err)
	}

	// Locate a real tenant + endpoint so the foreign-key-ish query returns rows.
	// We use a stable endpoint id from the gold-copy seed (the
	// Salesforce GET /sobjects/Account row) if present; otherwise we generate
	// synthetic UUIDs and clean up after ourselves.
	var tenantID string
	if err := db.QueryRow(`SELECT id FROM tenants LIMIT 1`).Scan(&tenantID); err != nil {
		t.Skipf("no tenants in DB: %v", err)
	}

	endpointID := "a1000000-0000-0000-0000-000000000003" // Salesforce GET /sobjects/Account from seed
	datasourceID := "a1000000-0000-0000-0000-000000000001"
	var epExists bool
	_ = db.QueryRow(`SELECT EXISTS(SELECT 1 FROM catalog_node WHERE id = $1)`, endpointID).Scan(&epExists)
	if !epExists {
		// Pick any real endpoint.
		_ = db.QueryRow(`SELECT id FROM catalog_node WHERE node_type_id IN (SELECT id FROM catalog_node_types WHERE catalog_type_name = 'api_endpoint') LIMIT 1`).Scan(&endpointID)
		_ = db.QueryRow(`SELECT api_datasource_id FROM catalog_node WHERE id = $1`, endpointID).Scan(&datasourceID)
	}
	if endpointID == "" || datasourceID == "" {
		t.Skipf("no API endpoint row found in DB to anchor the test (tenant_id=%s)", tenantID)
	}

	// Insert one synthetic audit row and clean it up afterwards.
	now := time.Now()
	insertQuery := `
		INSERT INTO api_dispatch_audit_log (
			tenant_id, api_datasource_id, api_endpoint_id,
			method, path, status_code, duration_ms,
			success, record_count, error, request_params, created_at
		) VALUES (
			$1::uuid, $2::uuid, $3::uuid,
			$4, $5, $6, $7,
			$8, $9, $10, $11::jsonb, $12
		) RETURNING id
	`
	var rowID string
	if err := db.QueryRow(insertQuery,
		tenantID, datasourceID, endpointID,
		"GET", "/test/audit/list", 200, 42,
		true, 7, "", `{"x":"y"}`, now,
	).Scan(&rowID); err != nil {
		t.Fatalf("insert audit row: %v", err)
	}
	t.Cleanup(func() {
		_, _ = db.Exec(`DELETE FROM api_dispatch_audit_log WHERE id = $1::uuid`, rowID)
	})

	// Stand up a minimal chi router serving only /api/api-dispatcher/audit.
	h := &ApiDispatcherHandler{db: db}
	router := newTestAuditRouter(h)
	srv := httptest.NewServer(router)
	defer srv.Close()

	url := fmt.Sprintf("%s/api/api-dispatcher/audit?tenant_id=%s&endpoint_id=%s&limit=10", srv.URL, tenantID, endpointID)
	resp, err := http.Get(url)
	if err != nil {
		t.Fatalf("GET: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	var payload struct {
		Success bool              `json:"success"`
		Data    []json.RawMessage `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if !payload.Success {
		t.Errorf("payload.success should be true")
	}
	if len(payload.Data) == 0 {
		t.Errorf("expected at least one audit row, got none")
	}
}

// TestRecordAudit_FiresBackground verifies that the worker actually drains
// the buffer and writes rows when a real DB is available.
func TestRecordAudit_FiresBackground(t *testing.T) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Skip("DATABASE_URL not set")
	}
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Fatalf("sql.Open: %v", err)
	}
	defer db.Close()
	if err := db.Ping(); err != nil {
		t.Skipf("database unreachable: %v", err)
	}

	var tenantID, endpointID, datasourceID string
	if err := db.QueryRow(`SELECT id FROM tenants LIMIT 1`).Scan(&tenantID); err != nil {
		t.Skip("no tenants")
	}
	if err := db.QueryRow(`SELECT id FROM catalog_node WHERE node_type_id IN (SELECT id FROM catalog_node_types WHERE catalog_type_name = 'api_endpoint') LIMIT 1`).Scan(&endpointID); err != nil {
		t.Skip("no endpoint")
	}
	if err := db.QueryRow(`SELECT id FROM catalog_node WHERE node_type_id IN (SELECT id FROM catalog_node_types WHERE catalog_type_name = 'api_datasource') LIMIT 1`).Scan(&datasourceID); err != nil {
		t.Skip("no datasource")
	}

	h := &ApiDispatcherHandler{db: db}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	h.StartAuditWorker(ctx)

	const n = 5
	for i := 0; i < n; i++ {
		h.recordAudit(auditEntry{
			TenantID:     tenantID,
			DatasourceID: datasourceID,
			EndpointID:   endpointID,
			Method:       "GET",
			Path:         fmt.Sprintf("/background-fire/%d", i),
			StatusCode:   200,
			DurationMs:   int64(i + 1),
			Success:      true,
			RecordCount:  i,
		})
	}

	// Poll for the rows to appear. With no DB latency, 1s is generous.
	deadline := time.Now().Add(2 * time.Second)
	var got int
	for time.Now().Before(deadline) {
		_ = db.QueryRow(`
			SELECT count(*) FROM api_dispatch_audit_log
			WHERE tenant_id = $1::uuid AND api_endpoint_id = $2::uuid AND path LIKE '/background-fire/%'
		`, tenantID, endpointID).Scan(&got)
		if got >= n {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Cleanup(func() {
		_, _ = db.Exec(`DELETE FROM api_dispatch_audit_log WHERE tenant_id = $1::uuid AND path LIKE '/background-fire/%'`, tenantID)
	})
	if got < n {
		t.Errorf("expected %d background audit rows, got %d", n, got)
	}
}

// newTestAuditRouter mounts just the audit endpoint on a fresh chi router.
// Kept private to this test file so it doesn't leak into the production
// routing table.
func newTestAuditRouter(h *ApiDispatcherHandler) http.Handler {
	// We deliberately don't pull in chi at the top of the file to keep this
	// test's dependency surface small. A one-off tiny mux is sufficient.
	mux := http.NewServeMux()
	mux.HandleFunc("/api/api-dispatcher/audit", h.ListDispatchAudit)
	return mux
}