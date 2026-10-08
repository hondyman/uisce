package tenantplatform

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
)

const (
	testInstanceID = "22222222-2222-2222-2222-222222222222"
	testTenantCode = "acme"
)

type recorder struct {
	events []string
}

type fakeDB struct {
	rec  *recorder
	err  error
	host string
	port int
	db   string
}

func (f *fakeDB) Ping(_ context.Context, host string, port int, database string) error {
	f.rec.events = append(f.rec.events, "db")
	f.host, f.port, f.db = host, port, database
	return f.err
}

type fakeIdentity struct {
	rec *recorder
	err error
	got string
}

func (f *fakeIdentity) CheckIssuer(_ context.Context, issuer string) error {
	f.rec.events = append(f.rec.events, "identity")
	f.got = issuer
	return f.err
}

type fakeSecrets struct {
	rec     *recorder
	missing map[string]bool
	seen    []string
}

func (f *fakeSecrets) Exists(_ context.Context, path string) (bool, error) {
	f.rec.events = append(f.rec.events, "secrets")
	f.seen = append(f.seen, path)
	return !f.missing[path], nil
}

type fakeStore struct {
	rec        *recorder
	persistErr error
	markErr    error
}

func (f *fakeStore) PersistEndpoints(_ context.Context, _ string, _ Endpoints) error {
	f.rec.events = append(f.rec.events, "persist")
	return f.persistErr
}

func (f *fakeStore) MarkRunning(_ context.Context, _ string) error {
	f.rec.events = append(f.rec.events, "running")
	return f.markErr
}

type harness struct {
	rec *recorder
	db  *fakeDB
	id  *fakeIdentity
	sec *fakeSecrets
	st  *fakeStore
}

func newHarness() harness {
	rec := &recorder{}
	return harness{
		rec: rec,
		db:  &fakeDB{rec: rec},
		id:  &fakeIdentity{rec: rec},
		sec: &fakeSecrets{rec: rec, missing: map[string]bool{}},
		st:  &fakeStore{rec: rec},
	}
}

func (h harness) activities() *Activities {
	return &Activities{DB: h.db, Identity: h.id, Secrets: h.sec, Store: h.st}
}

func validInput() Input {
	return Input{
		InstanceID:  testInstanceID,
		TenantCode:  testTenantCode,
		Environment: "dev",
		Endpoints: Endpoints{
			PostgresHost: "pg.us-east-1.internal",
			PostgresPort: 5432,
			DatabaseName: "tenant_acme",
			Issuer:       "https://keycloak.example.internal/realms/acme",
		},
	}
}

func TestStartPlatformSuccessOrdering(t *testing.T) {
	h := newHarness()
	res, err := h.activities().StartPlatform(context.Background(), validInput())
	if err != nil {
		t.Fatalf("StartPlatform: %v", err)
	}
	wantEvents := []string{"db", "identity", "secrets", "secrets", "persist", "running"}
	if !reflect.DeepEqual(h.rec.events, wantEvents) {
		t.Fatalf("events = %v, want %v", h.rec.events, wantEvents)
	}
	if !reflect.DeepEqual(res.ChecksPassed, []string{CheckDatabase, CheckIdentity, CheckSecrets}) {
		t.Fatalf("checks passed = %v", res.ChecksPassed)
	}
}

func TestDatabaseCheckUsesSagaEndpointExactly(t *testing.T) {
	h := newHarness()
	in := validInput()
	if _, err := h.activities().StartPlatform(context.Background(), in); err != nil {
		t.Fatalf("StartPlatform: %v", err)
	}
	if h.db.host != in.Endpoints.PostgresHost || h.db.port != in.Endpoints.PostgresPort || h.db.db != in.Endpoints.DatabaseName {
		t.Fatalf("DB probe used %s:%d/%s, want the saga endpoint", h.db.host, h.db.port, h.db.db)
	}
	if h.id.got != in.Endpoints.Issuer {
		t.Fatalf("identity probe checked %q", h.id.got)
	}
}

func TestSecretsCheckAssertsLayoutPathsOnly(t *testing.T) {
	h := newHarness()
	if _, err := h.activities().StartPlatform(context.Background(), validInput()); err != nil {
		t.Fatalf("StartPlatform: %v", err)
	}
	want := []string{"tenants/acme/identity", "tenants/acme/dev"}
	if !reflect.DeepEqual(h.sec.seen, want) {
		t.Fatalf("secret paths checked = %v, want %v", h.sec.seen, want)
	}
}

func TestFailedChecksNameEveryFailureAndStopBeforePersist(t *testing.T) {
	h := newHarness()
	h.db.err = errors.New("connection refused to 10.0.0.9 password=s3cret")
	h.id.err = errors.New("issuer mismatch")
	h.sec.missing["tenants/acme/dev"] = true
	_, err := h.activities().StartPlatform(context.Background(), validInput())
	if err == nil {
		t.Fatal("StartPlatform succeeded with failed checks")
	}
	for _, name := range []string{CheckDatabase, CheckIdentity, CheckSecrets} {
		if !strings.Contains(err.Error(), name) {
			t.Errorf("error %q does not name the %q check", err, name)
		}
	}
	if strings.Contains(err.Error(), "s3cret") || strings.Contains(err.Error(), "10.0.0.9") {
		t.Fatalf("check failure leaked probe detail: %v", err)
	}
	for _, e := range h.rec.events {
		if e == "persist" || e == "running" {
			t.Fatalf("instance persisted or marked running after failed checks: %v", h.rec.events)
		}
	}
}

func TestPersistFailureLeavesInstanceNotRunning(t *testing.T) {
	h := newHarness()
	h.st.persistErr = errors.New("write failed for tenant acme")
	_, err := h.activities().StartPlatform(context.Background(), validInput())
	if err == nil {
		t.Fatal("persist failure not reported")
	}
	if strings.Contains(err.Error(), "acme") {
		t.Fatalf("persist error echoed tenant detail: %v", err)
	}
	for _, e := range h.rec.events {
		if e == "running" {
			t.Fatal("marked running after a failed persist")
		}
	}
}

func TestMarkRunningFailureIsReported(t *testing.T) {
	h := newHarness()
	h.st.markErr = errors.New("db down")
	if _, err := h.activities().StartPlatform(context.Background(), validInput()); err == nil {
		t.Fatal("mark-running failure not reported")
	}
}

func TestRejectsBadInputBeforeAnyProbe(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*Input)
	}{
		{"instance ID not a UUID", func(i *Input) { i.InstanceID = "not-a-uuid" }},
		{"tenant code injection", func(i *Input) { i.TenantCode = "x; drop" }},
		{"environment unknown", func(i *Input) { i.Environment = "staging" }},
		{"host with semicolon", func(i *Input) { i.Endpoints.PostgresHost = "pg;evil" }},
		{"host empty", func(i *Input) { i.Endpoints.PostgresHost = "" }},
		{"port zero", func(i *Input) { i.Endpoints.PostgresPort = 0 }},
		{"port too large", func(i *Input) { i.Endpoints.PostgresPort = 70000 }},
		{"database name quote", func(i *Input) { i.Endpoints.DatabaseName = `a"b` }},
		{"database name overlong", func(i *Input) { i.Endpoints.DatabaseName = "d" + strings.Repeat("x", 63) }},
		{"issuer javascript", func(i *Input) { i.Endpoints.Issuer = "javascript:alert(1)" }},
		{"issuer no host", func(i *Input) { i.Endpoints.Issuer = "https://" }},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h := newHarness()
			in := validInput()
			tc.mutate(&in)
			if _, err := h.activities().StartPlatform(context.Background(), in); err == nil {
				t.Fatal("accepted invalid input")
			}
			if len(h.rec.events) != 0 {
				t.Fatalf("probes or store called for invalid input: %v", h.rec.events)
			}
		})
	}
}

func TestWithoutDependenciesFailsClosed(t *testing.T) {
	if _, err := (&Activities{}).StartPlatform(context.Background(), validInput()); err == nil {
		t.Fatal("ran without dependencies")
	}
}

func TestHTTPIdentityProbe(t *testing.T) {
	var issuer string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/realms/good/.well-known/openid-configuration":
			json.NewEncoder(w).Encode(map[string]string{"issuer": issuer})
		case "/realms/wrong/.well-known/openid-configuration":
			json.NewEncoder(w).Encode(map[string]string{"issuer": "https://other.example/realms/wrong"})
		case "/realms/broken/.well-known/openid-configuration":
			http.Error(w, "boom", http.StatusInternalServerError)
		case "/realms/junk/.well-known/openid-configuration":
			w.Write([]byte("<html>not json</html>"))
		case "/realms/huge/.well-known/openid-configuration":
			w.Write([]byte(`{"issuer":"` + strings.Repeat("x", maxDiscoveryBytes+10) + `"}`))
		case "/realms/redirect/.well-known/openid-configuration":
			http.Redirect(w, r, "https://evil.example/", http.StatusFound)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	probe := NewHTTPIdentityProbe()
	issuer = server.URL + "/realms/good"
	if err := probe.CheckIssuer(context.Background(), issuer); err != nil {
		t.Fatalf("matching issuer rejected: %v", err)
	}

	cases := map[string]string{
		"wrong realm":      server.URL + "/realms/wrong",
		"server error":     server.URL + "/realms/broken",
		"not json":         server.URL + "/realms/junk",
		"oversize body":    server.URL + "/realms/huge",
		"redirect refused": server.URL + "/realms/redirect",
		"missing realm":    server.URL + "/realms/absent",
	}
	for name, iss := range cases {
		if err := probe.CheckIssuer(context.Background(), iss); err == nil {
			t.Errorf("%s: accepted", name)
		}
	}
	if err := probe.CheckIssuer(context.Background(), "not a url"); err == nil {
		t.Error("accepted a malformed issuer")
	}
}
