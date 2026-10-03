package provisioning

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/require"
	"go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/sdk/client"
	"go.uber.org/zap"

	"github.com/hondyman/uisce/backend/internal/security"
)

type fakeRun struct {
	client.WorkflowRun
	id string
}

func (r fakeRun) GetID() string    { return r.id }
func (r fakeRun) GetRunID() string { return "run-1" }

// fakeTemporal records what the handler starts; every other method panics via the nil embed.
type fakeTemporal struct {
	client.Client
	mu        sync.Mutex
	started   []client.StartWorkflowOptions
	workflow  []interface{}
	inputs    []ProvisioningWorkflowInput
	described []string
}

func (f *fakeTemporal) ExecuteWorkflow(_ context.Context, o client.StartWorkflowOptions, wf interface{}, args ...interface{}) (client.WorkflowRun, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.started = append(f.started, o)
	f.workflow = append(f.workflow, wf)
	f.inputs = append(f.inputs, args[0].(ProvisioningWorkflowInput))
	return fakeRun{id: o.ID}, nil
}

func (f *fakeTemporal) DescribeWorkflowExecution(_ context.Context, id, _ string) (*workflowservice.DescribeWorkflowExecutionResponse, error) {
	f.described = append(f.described, id)
	return nil, context.DeadlineExceeded
}

var (
	globalAdmin = &security.AuthInfo{UserID: "admin-1", Roles: []string{"global_admin"}, IsGlobalAdmin: true}
	ordinary    = &security.AuthInfo{UserID: "user-1", Roles: []string{"user"}}
)

type rig struct {
	h    *ProvisioningHandler
	tc   *fakeTemporal
	mock sqlmock.Sqlmock
	r    http.Handler
}

func newRig(t *testing.T) *rig {
	t.Helper()
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	require.NoError(t, err)
	t.Cleanup(func() { db.Close() })
	tc := &fakeTemporal{}
	h := NewProvisioningHandler(tc, db, zap.NewNop().Sugar())
	r := chi.NewRouter()
	h.RegisterAdminRoutes(r)
	return &rig{h: h, tc: tc, mock: mock, r: r}
}

func (g *rig) expectFree(goldDB string) {
	g.mock.ExpectQuery(`SELECT code FROM public.tenants`).WillReturnRows(sqlmock.NewRows([]string{"code"}))
	g.mock.ExpectQuery(`SELECT name FROM public.tenants`).WillReturnRows(sqlmock.NewRows([]string{"name"}))
	g.mock.ExpectQuery(`FROM public.tenants t`).WillReturnRows(
		sqlmock.NewRows([]string{"id", "id", "db"}).AddRow("gold-t", "gold-i", goldDB))
}

func (g *rig) do(method, path, body string, auth *security.AuthInfo) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	if auth != nil {
		req = req.WithContext(security.WithAuthInfo(req.Context(), *auth))
	}
	w := httptest.NewRecorder()
	g.r.ServeHTTP(w, req)
	return w
}

const provPath = "/system/tenants/provision"

func TestProvision_OnlyAGlobalAdminMayCreateATenant(t *testing.T) {
	for name, who := range map[string]*security.AuthInfo{"anonymous": nil, "ordinary user": ordinary} {
		g := newRig(t)
		w := g.do("POST", provPath, `{"tenant_name":"Acme","instance_name":"prod"}`, who)
		require.Contains(t, []int{http.StatusUnauthorized, http.StatusForbidden}, w.Code, name)
		require.Empty(t, g.tc.started, name)
		require.NoError(t, g.mock.ExpectationsWereMet(), "%s must not touch the database", name)
	}
}

func TestProvision_AppAndQueueReachTheWorkflow(t *testing.T) {
	g := newRig(t)
	g.expectFree("gold_copy_db")
	w := g.do("POST", provPath,
		`{"tenant_name":"Acme Corp","instance_name":"prod","tenant_code":"acme","app":"orm","requester_id":"spoofed"}`, globalAdmin)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	require.Len(t, g.tc.started, 1)

	in := g.tc.inputs[0]
	require.Equal(t, "orm", in.App)
	require.Equal(t, "tenant_acme", in.DatabaseName)
	require.Equal(t, "gold_copy_db", in.GoldCopyDatabase)
	require.Equal(t, "admin-1", in.RequesterID, "the requester is the authenticated caller, never the request body")
	require.Equal(t, "bp_queue", g.tc.started[0].TaskQueue, "the queue cmd/worker polls")
	require.Equal(t, "TenantInstanceProvisioningWorkflowFn", g.tc.workflow[0])
	require.Equal(t, provisioningExecutionTimeout, g.tc.started[0].WorkflowExecutionTimeout)
	require.NoError(t, g.mock.ExpectationsWereMet())
}

func TestProvision_WithoutAnAppNothingChanges(t *testing.T) {
	g := newRig(t)
	g.expectFree("gold_copy_db")
	w := g.do("POST", provPath, `{"tenant_name":"Acme","instance_name":"prod","tenant_code":"acme"}`, globalAdmin)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	require.Empty(t, g.tc.inputs[0].App)
}

// Everything below must be refused before the database is read or a workflow is started.
func TestProvision_RefusesUnsafeInputBeforeDoingAnything(t *testing.T) {
	cases := map[string]string{
		"app uppercase":       `{"tenant_name":"A","instance_name":"p","app":"Orm"}`,
		"app path":            `{"tenant_name":"A","instance_name":"p","app":"../orm"}`,
		"app sql":             `{"tenant_name":"A","instance_name":"p","app":"orm; drop"}`,
		"code with quote":     `{"tenant_name":"A","instance_name":"p","tenant_code":"x\"; DROP DATABASE postgres; --"}`,
		"code uppercase":      `{"tenant_name":"A","instance_name":"p","tenant_code":"Acme"}`,
		"code leading digit":  `{"tenant_name":"A","instance_name":"p","tenant_code":"1acme"}`,
		"code too long":       `{"tenant_name":"A","instance_name":"p","tenant_code":"` + strings.Repeat("a", 60) + `"}`,
		"code with a space":   `{"tenant_name":"A","instance_name":"p","tenant_code":"ac me"}`,
		"missing tenant name": `{"instance_name":"p"}`,
		"missing instance":    `{"tenant_name":"A"}`,
		"not json":            `nope`,
	}
	for name, body := range cases {
		g := newRig(t)
		w := g.do("POST", provPath, body, globalAdmin)
		require.Equal(t, http.StatusBadRequest, w.Code, name)
		require.Empty(t, g.tc.started, name)
		require.NoError(t, g.mock.ExpectationsWereMet(), "%s must not query the database", name)
	}
}

func TestProvision_ADerivedCodeThatIsNotAnIdentifierIsRefused(t *testing.T) {
	g := newRig(t)
	g.mock.ExpectQuery(`SELECT code FROM public.tenants`).WillReturnRows(sqlmock.NewRows([]string{"code"}))
	g.mock.ExpectQuery(`SELECT name FROM public.tenants`).WillReturnRows(sqlmock.NewRows([]string{"name"}))
	g.mock.ExpectQuery(`FROM public.tenants t`).WillReturnRows(sqlmock.NewRows([]string{"a", "b", "c"}).AddRow("g", "i", "gold_copy_db"))
	w := g.do("POST", provPath, `{"tenant_name":"1st Bank","instance_name":"prod"}`, globalAdmin)
	require.Equal(t, http.StatusBadRequest, w.Code, w.Body.String())
	require.Empty(t, g.tc.started)
}

func TestProvision_NeverClonesTheControlPlaneAsTheGoldCopy(t *testing.T) {
	for _, bad := range []string{"", "alpha"} {
		g := newRig(t)
		g.expectFree(bad)
		w := g.do("POST", provPath, `{"tenant_name":"Acme","instance_name":"prod","tenant_code":"acme"}`, globalAdmin)
		require.Equal(t, http.StatusInternalServerError, w.Code, "gold copy database %q", bad)
		require.Empty(t, g.tc.started, "a gold copy with database %q must not start a run", bad)
	}
}

func TestProvision_WithoutTemporalIsUnavailable(t *testing.T) {
	db, _, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()
	r := chi.NewRouter()
	NewProvisioningHandler(nil, db, zap.NewNop().Sugar()).RegisterAdminRoutes(r)
	req := httptest.NewRequest("POST", provPath, strings.NewReader(`{"tenant_name":"A","instance_name":"p"}`))
	req = req.WithContext(security.WithAuthInfo(req.Context(), *globalAdmin))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusServiceUnavailable, w.Code)
}

func TestStatus_AdminOnlyAndOnlyForProvisioningRuns(t *testing.T) {
	path := "/system/tenants/11111111-2222-3333-4444-555555555555/provision/"

	g := newRig(t)
	w := g.do("GET", path+"tenant-provisioning-acme-1234", "", ordinary)
	require.Equal(t, http.StatusForbidden, w.Code)
	require.Empty(t, g.tc.described)

	w = g.do("GET", path+"some-other-workflow", "", globalAdmin)
	require.Equal(t, http.StatusBadRequest, w.Code, "a status call must not describe arbitrary workflows")
	require.Empty(t, g.tc.described)

	w = g.do("GET", path+"tenant-provisioning-acme-1234", "", globalAdmin)
	require.Equal(t, http.StatusInternalServerError, w.Code, "the fake describe fails; what matters is that it was asked")
	require.Equal(t, []string{"tenant-provisioning-acme-1234"}, g.tc.described)
}

func TestAdminRoutesDoNotExposeReprovisioningAnExistingInstance(t *testing.T) {
	g := newRig(t)
	w := g.do("POST", "/v1/tenants/t1/instances/i1/provision", "", globalAdmin)
	require.Equal(t, http.StatusNotFound, w.Code)
	w = g.do("POST", "/system/tenants/t1/instances/i1/provision", "", globalAdmin)
	require.NotEqual(t, http.StatusOK, w.Code)
}

func TestTheProvisioningHandlerIsMountedAdminOnly(t *testing.T) {
	src, err := os.ReadFile("../api/api.go")
	require.NoError(t, err)
	body := string(src)
	require.Contains(t, body, "NewProvisioningHandler(", "the handler must be mounted")
	require.Contains(t, body, "RegisterAdminRoutes(r)", "and only through the admin routes")
	for _, line := range strings.Split(body, "\n") {
		if strings.Contains(line, "NewProvisioningHandler(") {
			require.NotContains(t, line, ".RegisterRoutes(", "the unrestricted route set must stay unmounted")
		}
	}
}
