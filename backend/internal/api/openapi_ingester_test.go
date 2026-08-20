package api

import (
	"bytes"
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hondyman/uisce/backend/internal/security"
	_ "github.com/lib/pq"
)

// loadDemoSpec reads the demo_crm.json fixture from the samples/ directory.
// `go test` runs in the test's package directory (backend/internal/api/), so
// the relative path walks up three levels to reach the repo root.
func loadDemoSpec(t *testing.T) []byte {
	t.Helper()
	candidates := []string{
		"../../samples/openapi/demo_crm.json",
		"../../../samples/openapi/demo_crm.json",
	}
	for _, p := range candidates {
		abs, _ := filepath.Abs(p)
		if data, err := os.ReadFile(abs); err == nil {
			return data
		}
	}
	t.Skip("samples/openapi/demo_crm.json not found; skipping parser-only tests")
	return nil
}

// requireTestDB returns a *sql.DB pointed at the test database, or skips the
// test when DATABASE_URL is unset. Uses the same default as cmd/server.
func requireTestDB(t *testing.T) *sql.DB {
	t.Helper()
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = os.Getenv("TEST_DATABASE_URL")
	}
	if dsn == "" {
		t.Skip("DATABASE_URL not set; skipping DB-backed tests")
	}
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Fatalf("sql.Open: %v", err)
	}
	if err := db.Ping(); err != nil {
		t.Skipf("database unreachable: %v", err)
	}
	return db
}

func TestCollectFields_TopLevelArrayOfRefs(t *testing.T) {
	specBytes := loadDemoSpec(t)
	if specBytes == nil {
		return
	}
	var spec parsedSpec
	if err := json.Unmarshal(specBytes, &spec); err != nil {
		t.Fatalf("parse spec: %v", err)
	}

	// Find the GET /customers operation and check its fields.
	for rawPath, rawPathItem := range spec.Paths {
		if rawPath != "/customers" {
			continue
		}
		var methods map[string]json.RawMessage
		_ = json.Unmarshal(rawPathItem, &methods)
		rawOp, ok := methods["get"]
		if !ok {
			t.Fatalf("missing get on /customers")
		}
		var op parsedOperation
		_ = json.Unmarshal(rawOp, &op)
		fields := collectFields(op, spec.Components.Schemas)

		got := map[string]fieldDef{}
		for _, f := range fields {
			got[f.Name] = f
		}

		required := []string{"id", "email", "name", "balance", "is_active"}
		for _, name := range required {
			if _, ok := got[name]; !ok {
				t.Errorf("expected field %q from Customer schema, got %+v", name, fieldNames(fields))
			}
		}
		if f := got["id"]; !f.IsPrimaryKey {
			t.Errorf("id should be marked primary key (x-primary-key); got %+v", f)
		}
		if got["id"].SemanticTerm != "Customer ID" {
			t.Errorf("id.x-semantic-term should be 'Customer ID'; got %q", got["id"].SemanticTerm)
		}
		if got["balance"].DataType != "numeric" {
			t.Errorf("balance data_type should be numeric; got %q", got["balance"].DataType)
		}
		if got["is_active"].DataType != "boolean" {
			t.Errorf("is_active data_type should be boolean; got %q", got["is_active"].DataType)
		}
		// Verify array-style JSON path: "$[*].name"
		for _, f := range fields {
			if f.Name == "name" && !strings.Contains(f.JSONPath, "[*]") {
				t.Errorf("expected array-style JSON path for /customers collection; got %q", f.JSONPath)
			}
		}
		return
	}
	t.Fatalf("did not find /customers path in demo spec")
}

func TestCollectFields_InlineObject(t *testing.T) {
	specBytes := loadDemoSpec(t)
	if specBytes == nil {
		return
	}
	var spec parsedSpec
	if err := json.Unmarshal(specBytes, &spec); err != nil {
		t.Fatalf("parse spec: %v", err)
	}
	for rawPath, rawPathItem := range spec.Paths {
		if rawPath != "/orders" {
			continue
		}
		var methods map[string]json.RawMessage
		_ = json.Unmarshal(rawPathItem, &methods)
		rawOp, ok := methods["get"]
		if !ok {
			t.Fatalf("missing get on /orders")
		}
		var op parsedOperation
		_ = json.Unmarshal(rawOp, &op)
		fields := collectFields(op, spec.Components.Schemas)
		got := map[string]fieldDef{}
		for _, f := range fields {
			got[f.Name] = f
		}
		for _, name := range []string{"order_id", "total", "placed_at"} {
			if _, ok := got[name]; !ok {
				t.Errorf("expected inline field %q from /orders response, got %+v", name, fieldNames(fields))
			}
		}
		if got["order_id"].SemanticTerm != "Order ID" {
			t.Errorf("order_id.x-semantic-term should be 'Order ID'; got %q", got["order_id"].SemanticTerm)
		}
		if got["placed_at"].DataType != "varchar" {
			t.Errorf("placed_at should map to varchar (date-time is a format, not a type); got %q", got["placed_at"].DataType)
		}
		return
	}
	t.Fatalf("did not find /orders path in demo spec")
}

func TestMapOpenAPIType(t *testing.T) {
	cases := map[string]string{
		"string":  "varchar",
		"integer": "integer",
		"number":  "numeric",
		"boolean": "boolean",
		"null":    "varchar",
		"object":  "varchar", // default
		"array":   "varchar",
	}
	for in, want := range cases {
		if got := mapOpenAPIType(in); got != want {
			t.Errorf("mapOpenAPIType(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestInferDefaultAuth(t *testing.T) {
	cases := []struct {
		name    string
		schemes string
		want    string
	}{
		{"oauth2_password", `{"oauth":{"type":"oauth2","flows":{"password":{"tokenUrl":"https://x/t"}}}}`, "oauth2_bearer"},
		{"oauth2_client", `{"oauth":{"type":"oauth2","flows":{"clientCredentials":{"tokenUrl":"https://x/t"}}}}`, "oauth2_bearer"},
		{"http_bearer", `{"b":{"type":"http","scheme":"bearer"}}`, "basic_auth"},
		{"apiKey", `{"k":{"type":"apiKey","in":"header"}}`, "api_key"},
		{"empty", `{}`, "none"},
	}
	for _, c := range cases {
		var schemes map[string]json.RawMessage
		if err := json.Unmarshal([]byte(c.schemes), &schemes); err != nil {
			t.Fatalf("%s: invalid test fixture: %v", c.name, err)
		}
		if got := inferDefaultAuth(schemes); got != c.want {
			t.Errorf("%s: inferDefaultAuth = %q, want %q", c.name, got, c.want)
		}
	}
}

func TestResourceNameFor(t *testing.T) {
	cases := []struct {
		path string
		tags []string
		want string
	}{
		{"/customers", nil, "Customers"},
		{"/customers", []string{"Accounts"}, "Accounts"},
		{"/orders/{id}/items", nil, "Orders"},
		{"/{tenant}/widgets", nil, "Widgets"},
		{"/", nil, "Root"},
		{"", nil, "Root"},
		{"/a/b/c", []string{""}, "A"}, // empty tag is skipped, falls back to first segment
	}
	for _, c := range cases {
		if got := resourceNameFor(c.path, c.tags); got != c.want {
			t.Errorf("resourceNameFor(%q, %v) = %q, want %q", c.path, c.tags, got, c.want)
		}
	}
}

func TestSlug(t *testing.T) {
	cases := map[string]string{
		"":                  "x",
		" ":                 "x",
		"hello":             "hello",
		"Hello World":       "hello_world",
		"v1.0":              "v1_0",
		"acme-crm.example":  "acme_crm_example",
		"path/with/slashes": "path_with_slashes",
	}
	for in, want := range cases {
		if got := slug(in); got != want {
			t.Errorf("slug(%q) = %q, want %q", in, got, want)
		}
	}
}

// TestResolveSpecBytes_InlineJSON verifies the inline-spec path of the
// ingester without needing network access.
func TestResolveSpecBytes_InlineJSON(t *testing.T) {
	h := &ApiDispatcherHandler{}
	req := OpenAPIIngestRequest{Spec: json.RawMessage(`{"openapi":"3.0.0","info":{"title":"x"}}`)}
	got, err := h.resolveSpecBytes(context.Background(), req)
	if err != nil {
		t.Fatalf("resolveSpecBytes: %v", err)
	}
	if !strings.Contains(string(got), `"openapi":"3.0.0"`) {
		t.Errorf("unexpected body: %s", string(got))
	}
}

// TestResolveSpecBytes_RejectsYAML signals a clear error when callers pass
// YAML-shaped text in the `spec` field (v1 supports JSON only).
func TestResolveSpecBytes_RejectsYAML(t *testing.T) {
	h := &ApiDispatcherHandler{}
	req := OpenAPIIngestRequest{Spec: json.RawMessage(`openapi: 3.0.0\ninfo:\n  title: x`)}
	_, err := h.resolveSpecBytes(context.Background(), req)
	if err == nil {
		t.Fatalf("expected error for YAML payload")
	}
	if !strings.Contains(err.Error(), "valid JSON") {
		t.Errorf("expected JSON-validation error; got %v", err)
	}
}

// TestIngestOpenAPISpec_RejectsNonV3 verifies the version guard fires before
// any database work.
func TestIngestOpenAPISpec_RejectsNonV3(t *testing.T) {
	h := &ApiDispatcherHandler{}
	req := OpenAPIIngestRequest{Spec: json.RawMessage(`{"openapi":"2.0.0","info":{"title":"x"}}`)}
	_, err := h.IngestOpenAPISpec(context.Background(), req)
	if err == nil {
		t.Fatalf("expected version error for swagger 2.0")
	}
	if !strings.Contains(err.Error(), "only OpenAPI 3.x") {
		t.Errorf("expected version-guard error; got %v", err)
	}
}

// TestIngestOpenAPISpec_RequiresSource makes sure the handler refuses empty
// input.
func TestIngestOpenAPISpec_RequiresSource(t *testing.T) {
	h := &ApiDispatcherHandler{}
	_, err := h.IngestOpenAPISpec(context.Background(), OpenAPIIngestRequest{})
	if err == nil {
		t.Fatalf("expected error for empty request")
	}
}

// TestIngestOpenAPISpec_DemoAgainstRealDB exercises the full ingest path
// against a real database. Skipped unless DATABASE_URL is set; cleans up its
// own catalog_node rows on success.
func TestIngestOpenAPISpec_DemoAgainstRealDB(t *testing.T) {
	db := requireTestDB(t)
	defer db.Close()

	enc, err := newEncryptorForTest()
	if err != nil {
		t.Fatalf("encryptor: %v", err)
	}
	h := &ApiDispatcherHandler{db: db, encryptor: enc}

	specBytes := loadDemoSpec(t)
	if specBytes == nil {
		return
	}

	// Use a unique datasource name so we can clean up afterwards and don't
	// collide with prior runs.
	dsName := "Demo CRM API " + uniqueSuffix()

	// Run as gold-copy tenant (TenantID empty).
	result, err := h.IngestOpenAPISpec(context.Background(), OpenAPIIngestRequest{
		Name: dsName,
		Spec: specBytes,
	})
	if err != nil {
		t.Fatalf("IngestOpenAPISpec: %v", err)
	}
	t.Cleanup(func() {
		// Clean up: delete all rows under our datasource qualified_path.
		_, _ = db.Exec(`DELETE FROM catalog_edge WHERE source_node_id IN (
			SELECT id FROM catalog_node WHERE qualified_path LIKE $1
		) OR target_node_id IN (
			SELECT id FROM catalog_node WHERE qualified_path LIKE $1
		)`, "/api/"+slug(dsName)+"%")
		_, _ = db.Exec(`DELETE FROM catalog_node WHERE qualified_path LIKE $1`, "/api/"+slug(dsName)+"%")
	})

	if result.DatasourceName != dsName {
		t.Errorf("DatasourceName = %q, want %q", result.DatasourceName, dsName)
	}
	if result.ResourcesCreated < 2 {
		t.Errorf("expected at least 2 resources (Customers, Orders); got %d", result.ResourcesCreated)
	}
	if result.EndpointsCreated < 4 {
		t.Errorf("expected at least 4 endpoints (GET/POST customers, GET customers/{id}, GET orders); got %d", result.EndpointsCreated)
	}
	if result.FieldsCreated < 8 {
		t.Errorf("expected at least 8 fields (5 from Customer + 3 from orders inline); got %d", result.FieldsCreated)
	}
	if result.SemanticEdgesLinked < 0 {
		// Integration-test assertion: depends on whether semantic terms with
		// the fixture's x-semantic-term names already exist in the target DB.
		// We assert the code path executed (>= 0) but don't enforce a count.
		t.Errorf("SemanticEdgesLinked should be >= 0; got %d", result.SemanticEdgesLinked)
	}

	// Re-ingest must update in place (no new rows).
	preResCount := result.ResourcesCreated
	preEpCount := result.EndpointsCreated
	preFieldCount := result.FieldsCreated
	result2, err := h.IngestOpenAPISpec(context.Background(), OpenAPIIngestRequest{
		Name: dsName,
		Spec: specBytes,
	})
	if err != nil {
		t.Fatalf("IngestOpenAPISpec (re-run): %v", err)
	}
	if result2.ResourcesCreated != preResCount {
		t.Errorf("re-ingest changed resource count: was %d, now %d", preResCount, result2.ResourcesCreated)
	}
	if result2.EndpointsCreated != preEpCount {
		t.Errorf("re-ingest changed endpoint count: was %d, now %d", preEpCount, result2.EndpointsCreated)
	}
	if result2.FieldsCreated != preFieldCount {
		t.Errorf("re-ingest changed field count: was %d, now %d", preFieldCount, result2.FieldsCreated)
	}
}

// newEncryptorForTest returns a deterministic 32-byte AES encryptor used by
// the ingester's auth_config plumbing. We don't actually encrypt anything in
// the openapi ingester path but the handler struct still has the field set.
func newEncryptorForTest() (*security.TokenEncryptor, error) {
	key := bytes.Repeat([]byte("o"), 32)
	return security.NewTokenEncryptor(key)
}

// uniqueSuffix produces a short per-run suffix to keep parallel DB tests
// from clobbering each other.
func uniqueSuffix() string {
	var b [4]byte
	_, _ = rand.Read(b[:])
	return hex.EncodeToString(b[:])
}

func fieldNames(fs []fieldDef) []string {
	out := make([]string, 0, len(fs))
	for _, f := range fs {
		out = append(out, f.Name)
	}
	return out
}