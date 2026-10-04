package api

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"

	dbpkg "github.com/hondyman/uisce/backend/internal/db"
	"github.com/hondyman/uisce/backend/internal/profiler"
	"github.com/hondyman/uisce/backend/internal/security"
	"github.com/hondyman/uisce/backend/internal/sourceconn"
)

// The profile endpoint opens a tenant's SOURCE database, and only through the source connector
// (ADR-030). These tests pin what it does and, as important, what it no longer does: it used to fall
// back to profiling the control-plane alpha database when the tenant had no datasource of its own, and
// to log.Fatal inside the request handler when ALPHA_DB_URL was missing.

type recSources struct {
	mu     sync.Mutex
	tenant string
	id     string
	policy sourceconn.Policy
	calls  int
	err    error
}

func (r *recSources) Source(_ context.Context, tenantID, datasourceID string, p sourceconn.Policy) (sourceconn.Source, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.calls++
	r.tenant, r.id, r.policy = tenantID, datasourceID, p
	if r.err != nil {
		return sourceconn.Source{}, r.err
	}
	return sourceconn.Source{ID: datasourceID, TenantID: tenantID, Config: []byte(`{}`)}, nil
}

type fixedCreds struct{ out string }

func (f fixedCreds) Hydrate(context.Context, sourceconn.Source) ([]byte, error) {
	return []byte(f.out), nil
}

func profileServer(t *testing.T, reg sourceconn.Registry, creds string) *Server {
	t.Helper()
	c, err := sourceconn.New(sourceconn.Config{Registry: reg, Credentials: fixedCreds{creds}, CallerTenant: dbpkg.GetTenantIDFromCtx,
		MaxPools: 2, MaxConnsPerPool: 1, IdleTTL: time.Minute, DialTimeout: time.Second})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(c.Close)
	return &Server{Validate: validator.New(), Sources: c}
}

func startProfileAs(s *Server, tenant, datasource, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest("POST", "/api/profiler/profile", strings.NewReader(body))
	if tenant != "" {
		req = req.WithContext(security.WithAuthInfo(req.Context(), security.AuthInfo{UserID: "u", TenantIDs: []string{tenant}, ActiveTenantID: tenant}))
	}
	if datasource != "" {
		req.Header.Set("X-Tenant-Datasource-ID", datasource)
	}
	w := httptest.NewRecorder()
	s.startProfile(w, req)
	return w
}

func jobCount(s *Server) int {
	n := 0
	s.ProfileJobs.Range(func(_, _ any) bool { n++; return true })
	return n
}

func TestStartProfile_ADatasourceTheTenantDoesNotOwnIsNotFoundAndNothingElseIsProfiled(t *testing.T) {
	// The old fallback would have called log.Fatal here (no ALPHA_DB_URL) or profiled alpha.
	t.Setenv("ALPHA_DB_URL", "")
	for name, e := range map[string]error{"unknown": sourceconn.ErrNotFound, "not this tenant's": sourceconn.ErrNotAllowed} {
		reg := &recSources{err: e}
		s := profileServer(t, reg, `{"host":"127.0.0.1","port":1,"database":"d","username":"u"}`)
		w := startProfileAs(s, "t-1", "ds-9", `{"node_ids":["n1"]}`)
		if w.Code != http.StatusNotFound {
			t.Fatalf("%s: want 404, got %d: %s", name, w.Code, w.Body.String())
		}
		if jobCount(s) != 0 {
			t.Fatalf("%s: no job may be queued", name)
		}
		if reg.tenant != "t-1" || reg.id != "ds-9" || reg.policy != sourceconn.OwnerOnly {
			t.Fatalf("%s: the lookup must be for the authenticated tenant, the requested id and OwnerOnly: %+v", name, reg)
		}
	}
}

func TestStartProfile_AbsentAndNotYoursLookTheSame(t *testing.T) {
	bodies := map[string]string{}
	for name, e := range map[string]error{"absent": sourceconn.ErrNotFound, "not yours": sourceconn.ErrNotAllowed} {
		s := profileServer(t, &recSources{err: e}, `{}`)
		w := startProfileAs(s, "t-1", "ds-9", `{"node_ids":["n1"]}`)
		bodies[name] = w.Body.String()
	}
	if bodies["absent"] != bodies["not yours"] {
		t.Fatalf("the response must not reveal whether another tenant's datasource exists: %q vs %q", bodies["absent"], bodies["not yours"])
	}
}

func TestStartProfile_NoTenantAndNoConnectorFailClosed(t *testing.T) {
	s := profileServer(t, &recSources{}, `{}`)
	if w := startProfileAs(s, "", "ds-1", `{"node_ids":["n1"]}`); w.Code != http.StatusUnauthorized {
		t.Fatalf("no tenant: want 401, got %d", w.Code)
	}
	s.Sources = nil
	if w := startProfileAs(s, "t-1", "ds-1", `{"node_ids":["n1"]}`); w.Code != http.StatusServiceUnavailable {
		t.Fatalf("no connector: want 503, got %d", w.Code)
	}
	if jobCount(s) != 0 {
		t.Fatal("no job may be queued")
	}
}

func TestStartProfile_AnUnreachableSourceIsAGatewayErrorNotAFallback(t *testing.T) {
	s := profileServer(t, &recSources{}, `{"host":"127.0.0.1","port":1,"database":"d","username":"u"}`)
	w := startProfileAs(s, "t-1", "ds-1", `{"node_ids":["n1"]}`)
	if w.Code != http.StatusBadGateway {
		t.Fatalf("want 502, got %d: %s", w.Code, w.Body.String())
	}
	if strings.Contains(w.Body.String(), "127.0.0.1") || strings.Contains(w.Body.String(), "password") {
		t.Fatalf("the response must not echo connection details: %s", w.Body.String())
	}
	if jobCount(s) != 0 {
		t.Fatal("no job may be queued for an unreachable source")
	}
}

// A job whose datasource has gone away fails; it never touches another database.
func TestRunProfile_ASourceThatCannotBeOpenedFailsTheJob(t *testing.T) {
	t.Setenv("SEMLAYER_TEST_SKIP_ALPHA_POOL", "1")
	s := profileServer(t, &recSources{err: errors.New("datasource deactivated")}, `{}`)
	called := false
	orig := profiler.ProfileTablesFunc
	profiler.ProfileTablesFunc = func(context.Context, *zap.Logger, *pgxpool.Pool, string, string, *pgxpool.Pool, string, []string, int, float64, int, profiler.ProgressFunc) error {
		called = true
		return nil
	}
	t.Cleanup(func() { profiler.ProfileTablesFunc = orig })
	id := generateJobID()
	s.ProfileJobs.Store(id, &ProfileJob{ID: id, Status: "pending", CreatedAt: time.Now(), Req: ProfileRequest{TenantID: "t-1", DatasourceID: "ds-1", Schema: "public", Tables: []string{"t"}}})
	s.runProfile(id)
	v, _ := s.ProfileJobs.Load(id)
	job := v.(*ProfileJob)
	if job.Status != "failed" || !strings.Contains(job.Error, "datasource is not available") {
		t.Fatalf("job must fail closed, got %q / %q", job.Status, job.Error)
	}
	if called {
		t.Fatal("the profiler must not run without a source connection")
	}
}

// api.go must not grow the fallback back.
func TestProfilerHandlerHasNoLegacyDSNLookupOrAlphaFallback(t *testing.T) {
	src, err := os.ReadFile("api.go")
	if err != nil {
		t.Fatal(err)
	}
	body := string(src)
	i := strings.Index(body, "func (s *Server) startProfile(")
	j := strings.Index(body, "func (s *Server) getProfileStatus(")
	if i < 0 || j < i {
		t.Fatal("could not locate startProfile")
	}
	code := func(block string) string { // comments may explain history; code may not use it
		var out []string
		for _, line := range strings.Split(block, "\n") {
			if !strings.HasPrefix(strings.TrimSpace(line), "//") {
				out = append(out, line)
			}
		}
		return strings.Join(out, "\n")
	}
	start := code(body[i:j])
	for _, banned := range []string{"tenant_datasources", "connection_string", "ALPHA_DB_URL", "log.Fatal", "req.DataSource"} {
		if strings.Contains(start, banned) {
			t.Errorf("startProfile must not use %q: it opens tenant data only through the source connector", banned)
		}
	}
	k := strings.Index(body, "func (s *Server) runProfile(")
	run := body[k:]
	if e := strings.Index(run, "\nfunc "); e > 0 {
		run = run[:e+0]
	}
	if strings.Contains(code(run), "log.Fatal") {
		t.Error("runProfile must not log.Fatal: a missing setting fails the job, it does not kill the server")
	}
}
