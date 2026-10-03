package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/hondyman/uisce/backend/internal/lakehouse/registry"
	"github.com/hondyman/uisce/backend/internal/security"
	"github.com/stretchr/testify/require"
)

type fakeLakehouseRegistry struct {
	cfg        *registry.Config
	err        error // returned by every call when set
	setCalls   []int
	setResult  *registry.Config // what SetRetention returns, when set
	recorded   []string
	broken     *int64
	listArgs   [3]any
	lastTenant uuid.UUID
}

func (f *fakeLakehouseRegistry) Get(_ context.Context, id uuid.UUID) (*registry.Config, error) {
	f.lastTenant = id
	if f.err != nil {
		return nil, f.err
	}
	c := *f.cfg
	c.TenantID = id.String()
	return &c, nil
}

func (f *fakeLakehouseRegistry) SetRetention(_ context.Context, id uuid.UUID, days int, _ registry.Actor) (*registry.Config, error) {
	f.setCalls = append(f.setCalls, days)
	if f.err != nil {
		return nil, f.err
	}
	if f.setResult != nil {
		c := *f.setResult
		c.TenantID = id.String()
		return &c, nil
	}
	d := days
	return &registry.Config{TenantID: id.String(), Configured: true, AuditRetentionDays: &d, LifecycleState: "provisioning"}, nil
}

func (f *fakeLakehouseRegistry) List(_ context.Context, q string, limit, offset int) ([]registry.Config, int, error) {
	f.listArgs = [3]any{q, limit, offset}
	if f.err != nil {
		return nil, 0, f.err
	}
	return []registry.Config{*f.cfg}, 1, nil
}

func (f *fakeLakehouseRegistry) Audit(context.Context, uuid.UUID, int) ([]registry.AuditEntry, error) {
	return nil, f.err
}

func (f *fakeLakehouseRegistry) VerifyAudit(context.Context, uuid.UUID) (*int64, error) {
	return f.broken, f.err
}

func (f *fakeLakehouseRegistry) Record(_ context.Context, _ uuid.UUID, _ registry.Actor, action string, _, _ any) error {
	f.recorded = append(f.recorded, action)
	return f.err
}

type fakeProvisioner struct {
	started int
	err     error
	syncs   int
	syncErr error
	copies  int
	copyErr error
}

func (p *fakeProvisioner) StartAuditCopy(_ context.Context, id uuid.UUID, _ registry.Actor) (string, error) {
	p.copies++
	if p.copyErr != nil {
		return "", p.copyErr
	}
	return "lakehouse-audit-copy-" + id.String(), nil
}

func (p *fakeProvisioner) StartProvision(_ context.Context, id uuid.UUID, _ registry.Actor) (string, error) {
	p.started++
	if p.err != nil {
		return "", p.err
	}
	return "lakehouse-" + id.String(), nil
}

func (p *fakeProvisioner) StartRetentionSync(_ context.Context, id uuid.UUID, _ registry.Actor) (string, error) {
	p.syncs++
	if p.syncErr != nil {
		return "", p.syncErr
	}
	return "lakehouse-retention-" + id.String(), nil
}

var testTenant = uuid.MustParse("11111111-2222-3333-4444-555555555555")

func newLakehouseRouter(reg lakehouseRegistry, prov LakehouseProvisioner) http.Handler {
	r := chi.NewRouter()
	(&SystemLakehouseHandler{reg: reg, prov: prov}).RegisterRoutes(r)
	return r
}

func call(h http.Handler, method, path, body string, auth *security.AuthInfo) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	if auth != nil {
		req = req.WithContext(security.WithAuthInfo(req.Context(), *auth))
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	return w
}

var (
	globalAdmin = &security.AuthInfo{UserID: "admin-1", Roles: []string{"global_admin"}, IsGlobalAdmin: true}
	ordinary    = &security.AuthInfo{UserID: "user-1", Roles: []string{"user"}, TenantIDs: []string{testTenant.String()}, ActiveTenantID: testTenant.String()}
)

func lakehousePath(suffix string) string {
	return "/system/tenants/" + testTenant.String() + "/lakehouse" + suffix
}

func errCode(t *testing.T, w *httptest.ResponseRecorder) string {
	t.Helper()
	var b map[string]string
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &b))
	return b["code"]
}

func days(n int) *int { return &n }

// Every route refuses an anonymous caller, and refuses a tenant's own user even for
// their own tenant: this is a platform area.
func TestSystemLakehouse_RequiresGlobalAdmin(t *testing.T) {
	reg := &fakeLakehouseRegistry{cfg: &registry.Config{}}
	h := newLakehouseRouter(reg, &fakeProvisioner{})
	routes := []struct{ method, path, body string }{
		{"GET", "/system/lakehouses", ""},
		{"GET", lakehousePath(""), ""},
		{"PUT", lakehousePath(""), `{"audit_retention_days":365}`},
		{"GET", lakehousePath("/audit"), ""},
		{"POST", lakehousePath("/provision"), ""},
		{"POST", lakehousePath("/retention/sync"), ""},
	}
	for _, rt := range routes {
		t.Run(rt.method+" "+rt.path, func(t *testing.T) {
			require.Equal(t, http.StatusUnauthorized, call(h, rt.method, rt.path, rt.body, nil).Code, "anonymous")
			w := call(h, rt.method, rt.path, rt.body, ordinary)
			require.Equal(t, http.StatusForbidden, w.Code, "non-admin")
			require.Equal(t, "forbidden", errCode(t, w))
		})
	}
	require.Empty(t, reg.setCalls, "a refused caller must not reach the store")
	require.Empty(t, reg.recorded)
}

func TestSystemLakehouse_TenantIDMustBeAUUID(t *testing.T) {
	h := newLakehouseRouter(&fakeLakehouseRegistry{cfg: &registry.Config{}}, nil)
	for _, bad := range []string{"not-a-uuid", "00000000-0000-0000-0000-000000000000", "1"} {
		w := call(h, "GET", "/system/tenants/"+bad+"/lakehouse", "", globalAdmin)
		require.Equal(t, http.StatusBadRequest, w.Code, bad)
		require.Equal(t, "invalid_tenant_id", errCode(t, w))
	}
}

func TestSystemLakehouse_Put(t *testing.T) {
	t.Run("sets retention for the tenant in the path", func(t *testing.T) {
		reg := &fakeLakehouseRegistry{cfg: &registry.Config{}}
		w := call(newLakehouseRouter(reg, nil), "PUT", lakehousePath(""), `{"audit_retention_days":2555}`, globalAdmin)
		require.Equal(t, http.StatusOK, w.Code)
		require.Equal(t, []int{2555}, reg.setCalls)
		var got registry.Config
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &got))
		require.Equal(t, testTenant.String(), got.TenantID)
		require.Equal(t, 2555, *got.AuditRetentionDays)
	})

	bad := map[string]string{
		"missing field":          `{}`,
		"null":                   `{"audit_retention_days":null}`,
		"not a number":           `{"audit_retention_days":"a lot"}`,
		"not json":               `nope`,
		"empty":                  ``,
		"client names a bucket":  `{"audit_retention_days":365,"bucket":"someone-elses"}`,
		"client names warehouse": `{"audit_retention_days":365,"warehouse_name":"x"}`,
		"fractional":             `{"audit_retention_days":1.5}`,
	}
	for name, body := range bad {
		t.Run("rejects "+name, func(t *testing.T) {
			reg := &fakeLakehouseRegistry{cfg: &registry.Config{}}
			w := call(newLakehouseRouter(reg, nil), "PUT", lakehousePath(""), body, globalAdmin)
			require.Equal(t, http.StatusBadRequest, w.Code)
			require.Empty(t, reg.setCalls, "an invalid body must not reach the store")
		})
	}

	t.Run("missing retention says there is no default", func(t *testing.T) {
		w := call(newLakehouseRouter(&fakeLakehouseRegistry{cfg: &registry.Config{}}, nil), "PUT", lakehousePath(""), `{}`, globalAdmin)
		require.Equal(t, "retention_required", errCode(t, w))
	})

	errCases := []struct {
		name   string
		err    error
		status int
		code   string
	}{
		{"out of range", registry.ErrInvalidRetention, http.StatusBadRequest, "invalid_retention"},
		{"lowering", registry.ErrRetentionLowered, http.StatusConflict, "retention_lowered"},
		{"unknown tenant", registry.ErrTenantNotFound, http.StatusNotFound, "tenant_not_found"},
		{"database failure", errors.New("pq: connection refused to 10.0.0.5"), http.StatusInternalServerError, "internal_error"},
	}
	for _, c := range errCases {
		t.Run("maps "+c.name, func(t *testing.T) {
			w := call(newLakehouseRouter(&fakeLakehouseRegistry{cfg: &registry.Config{}, err: c.err}, nil),
				"PUT", lakehousePath(""), `{"audit_retention_days":365}`, globalAdmin)
			require.Equal(t, c.status, w.Code)
			require.Equal(t, c.code, errCode(t, w))
			require.NotContains(t, w.Body.String(), "10.0.0.5", "internal detail must not reach the client")
		})
	}
}

func TestSystemLakehouse_GetAndList(t *testing.T) {
	reg := &fakeLakehouseRegistry{cfg: &registry.Config{LifecycleState: "unconfigured"}}
	h := newLakehouseRouter(reg, nil)

	w := call(h, "GET", lakehousePath(""), "", globalAdmin)
	require.Equal(t, http.StatusOK, w.Code)
	require.Equal(t, testTenant, reg.lastTenant)
	require.Equal(t, "no-store", w.Header().Get("Cache-Control"))

	w = call(h, "GET", "/system/lakehouses?q=acme&limit=500&offset=10", "", globalAdmin)
	require.Equal(t, http.StatusOK, w.Code)
	require.Equal(t, [3]any{"acme", 500, 10}, reg.listArgs, "paging is passed through; the store clamps it")
	var body struct {
		Items  []registry.Config `json:"items"`
		Total  int               `json:"total"`
		Limit  int               `json:"limit"`
		Offset int               `json:"offset"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	require.Len(t, body.Items, 1)
	require.Equal(t, 50, body.Limit, "an out-of-range limit is reported as the clamped value")
	require.Equal(t, 10, body.Offset)
}

func TestSystemLakehouse_Audit(t *testing.T) {
	reg := &fakeLakehouseRegistry{cfg: &registry.Config{}}
	h := newLakehouseRouter(reg, nil)

	w := call(h, "GET", lakehousePath("/audit"), "", globalAdmin)
	require.Equal(t, http.StatusOK, w.Code)
	var ok map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &ok))
	require.Equal(t, true, ok["chain_intact"])
	require.Equal(t, []any{}, ok["entries"], "no entries is [], not null")
	require.NotContains(t, ok, "first_broken_id")

	id := int64(7)
	reg.broken = &id
	w = call(h, "GET", lakehousePath("/audit"), "", globalAdmin)
	var broken map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &broken))
	require.Equal(t, false, broken["chain_intact"])
	require.EqualValues(t, 7, broken["first_broken_id"])
}

func TestSystemLakehouse_Provision(t *testing.T) {
	configured := func(provisioned bool) *registry.Config {
		return &registry.Config{Configured: true, AuditRetentionDays: days(2555), Provisioned: provisioned, LifecycleState: "provisioning"}
	}

	t.Run("requires retention first", func(t *testing.T) {
		for name, cfg := range map[string]*registry.Config{
			"never configured":      {LifecycleState: "unconfigured"},
			"row without retention": {Configured: true, LifecycleState: "provisioning"},
		} {
			prov := &fakeProvisioner{}
			reg := &fakeLakehouseRegistry{cfg: cfg}
			w := call(newLakehouseRouter(reg, prov), "POST", lakehousePath("/provision"), "", globalAdmin)
			require.Equal(t, http.StatusConflict, w.Code, name)
			require.Equal(t, "retention_required", errCode(t, w), name)
			require.Zero(t, prov.started, name)
			require.Empty(t, reg.recorded, name)
		}
	})

	t.Run("refuses an already provisioned tenant", func(t *testing.T) {
		prov := &fakeProvisioner{}
		w := call(newLakehouseRouter(&fakeLakehouseRegistry{cfg: configured(true)}, prov), "POST", lakehousePath("/provision"), "", globalAdmin)
		require.Equal(t, http.StatusConflict, w.Code)
		require.Equal(t, "already_provisioned", errCode(t, w))
		require.Zero(t, prov.started)
	})

	t.Run("503 when no provisioner is configured, and nothing is audited", func(t *testing.T) {
		reg := &fakeLakehouseRegistry{cfg: configured(false)}
		w := call(newLakehouseRouter(reg, nil), "POST", lakehousePath("/provision"), "", globalAdmin)
		require.Equal(t, http.StatusServiceUnavailable, w.Code)
		require.Equal(t, "provisioner_unavailable", errCode(t, w))
		require.Empty(t, reg.recorded, "a request that cannot be served is not recorded as requested")
	})

	t.Run("202 starts the workflow and audits the request first", func(t *testing.T) {
		prov := &fakeProvisioner{}
		reg := &fakeLakehouseRegistry{cfg: configured(false)}
		w := call(newLakehouseRouter(reg, prov), "POST", lakehousePath("/provision"), "", globalAdmin)
		require.Equal(t, http.StatusAccepted, w.Code)
		require.Equal(t, 1, prov.started)
		require.Equal(t, []string{"provision_requested"}, reg.recorded)
		var b map[string]string
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &b))
		require.Equal(t, "lakehouse-"+testTenant.String(), b["workflow_id"])
	})

	t.Run("in-progress is a 409, other failures a 500", func(t *testing.T) {
		w := call(newLakehouseRouter(&fakeLakehouseRegistry{cfg: configured(false)}, &fakeProvisioner{err: ErrProvisionInProgress}),
			"POST", lakehousePath("/provision"), "", globalAdmin)
		require.Equal(t, http.StatusConflict, w.Code)
		require.Equal(t, "provision_in_progress", errCode(t, w))

		w = call(newLakehouseRouter(&fakeLakehouseRegistry{cfg: configured(false)}, &fakeProvisioner{err: errors.New("temporal down")}),
			"POST", lakehousePath("/provision"), "", globalAdmin)
		require.Equal(t, http.StatusInternalServerError, w.Code)
		require.NotContains(t, w.Body.String(), "temporal down")
	})
}

func pendingCfg() *registry.Config {
	d, applied := 3650, 365
	return &registry.Config{Configured: true, Provisioned: true, AuditRetentionDays: &d, RetentionAppliedDays: &applied,
		RetentionPending: true, LifecycleState: "active"}
}

// Raising the retention of a provisioned tenant starts raising its bucket, but that is best
// effort: the new retention is already saved and audited, so a failure to start must not fail
// the request.
func TestSystemLakehouse_PutStartsARetentionSyncOnlyWhenTheBucketIsBehind(t *testing.T) {
	t.Run("provisioned and behind: a sync starts", func(t *testing.T) {
		reg := &fakeLakehouseRegistry{cfg: &registry.Config{}, setResult: pendingCfg()}
		prov := &fakeProvisioner{}
		w := call(newLakehouseRouter(reg, prov), "PUT", lakehousePath(""), `{"audit_retention_days":3650}`, globalAdmin)
		require.Equal(t, http.StatusOK, w.Code)
		require.Equal(t, 1, prov.syncs)
	})

	t.Run("not provisioned, or already current: nothing starts", func(t *testing.T) {
		for name, cfg := range map[string]*registry.Config{
			"not provisioned": {Configured: true, LifecycleState: "provisioning"},
			"already current": func() *registry.Config { c := pendingCfg(); c.RetentionPending = false; return c }(),
		} {
			prov := &fakeProvisioner{}
			reg := &fakeLakehouseRegistry{cfg: &registry.Config{}, setResult: cfg}
			w := call(newLakehouseRouter(reg, prov), "PUT", lakehousePath(""), `{"audit_retention_days":3650}`, globalAdmin)
			require.Equal(t, http.StatusOK, w.Code, name)
			require.Zero(t, prov.syncs, name)
		}
	})

	t.Run("no provisioner configured: the PUT still succeeds", func(t *testing.T) {
		reg := &fakeLakehouseRegistry{cfg: &registry.Config{}, setResult: pendingCfg()}
		w := call(newLakehouseRouter(reg, nil), "PUT", lakehousePath(""), `{"audit_retention_days":3650}`, globalAdmin)
		require.Equal(t, http.StatusOK, w.Code)
		var got registry.Config
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &got))
		require.True(t, got.RetentionPending, "and the gap is reported")
	})

	t.Run("a failure to start never fails the request", func(t *testing.T) {
		for name, e := range map[string]error{"temporal down": errors.New("temporal down"), "already running": ErrRetentionSyncInProgress} {
			reg := &fakeLakehouseRegistry{cfg: &registry.Config{}, setResult: pendingCfg()}
			prov := &fakeProvisioner{syncErr: e}
			w := call(newLakehouseRouter(reg, prov), "PUT", lakehousePath(""), `{"audit_retention_days":3650}`, globalAdmin)
			require.Equal(t, http.StatusOK, w.Code, name)
			require.Equal(t, []int{3650}, reg.setCalls, name+": the retention was saved regardless")
		}
	})
}

func TestSystemLakehouse_RetentionSyncEndpoint(t *testing.T) {
	path := lakehousePath("/retention/sync")

	t.Run("202 starts it for a provisioned tenant whose bucket is behind", func(t *testing.T) {
		prov := &fakeProvisioner{}
		w := call(newLakehouseRouter(&fakeLakehouseRegistry{cfg: pendingCfg()}, prov), "POST", path, "", globalAdmin)
		require.Equal(t, http.StatusAccepted, w.Code)
		require.Equal(t, 1, prov.syncs)
		var b map[string]string
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &b))
		require.Equal(t, "lakehouse-retention-"+testTenant.String(), b["workflow_id"])
	})

	t.Run("409 when there is no bucket, or the bucket is already current", func(t *testing.T) {
		notProv := &registry.Config{Configured: true, LifecycleState: "provisioning"}
		current := pendingCfg()
		current.RetentionPending = false
		for name, c := range map[string]struct {
			cfg  *registry.Config
			code string
		}{"not provisioned": {notProv, "not_provisioned"}, "already current": {current, "retention_current"}} {
			prov := &fakeProvisioner{}
			w := call(newLakehouseRouter(&fakeLakehouseRegistry{cfg: c.cfg}, prov), "POST", path, "", globalAdmin)
			require.Equal(t, http.StatusConflict, w.Code, name)
			require.Equal(t, c.code, errCode(t, w), name)
			require.Zero(t, prov.syncs, name)
		}
	})

	t.Run("503 without a provisioner, 409 while one is running, 500 on other failures", func(t *testing.T) {
		w := call(newLakehouseRouter(&fakeLakehouseRegistry{cfg: pendingCfg()}, nil), "POST", path, "", globalAdmin)
		require.Equal(t, http.StatusServiceUnavailable, w.Code)

		w = call(newLakehouseRouter(&fakeLakehouseRegistry{cfg: pendingCfg()}, &fakeProvisioner{syncErr: ErrRetentionSyncInProgress}), "POST", path, "", globalAdmin)
		require.Equal(t, http.StatusConflict, w.Code)
		require.Equal(t, "sync_in_progress", errCode(t, w))

		w = call(newLakehouseRouter(&fakeLakehouseRegistry{cfg: pendingCfg()}, &fakeProvisioner{syncErr: errors.New("temporal down")}), "POST", path, "", globalAdmin)
		require.Equal(t, http.StatusInternalServerError, w.Code)
		require.NotContains(t, w.Body.String(), "temporal down")
	})
}

func TestSystemLakehouse_AuditCopyEndpoint(t *testing.T) {
	path := lakehousePath("/audit/copy")
	provisioned := &registry.Config{Configured: true, Provisioned: true, LifecycleState: "active"}

	t.Run("202 starts the copy through the provisioner", func(t *testing.T) {
		prov := &fakeProvisioner{}
		w := call(newLakehouseRouter(&fakeLakehouseRegistry{cfg: provisioned}, prov), "POST", path, "", globalAdmin)
		require.Equal(t, http.StatusAccepted, w.Code)
		require.Equal(t, 1, prov.copies)
		var b map[string]string
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &b))
		require.Equal(t, "lakehouse-audit-copy-"+testTenant.String(), b["workflow_id"])
	})

	t.Run("409 when the tenant has no warehouse, and nothing starts", func(t *testing.T) {
		prov := &fakeProvisioner{}
		notProv := &registry.Config{Configured: true, LifecycleState: "provisioning"}
		w := call(newLakehouseRouter(&fakeLakehouseRegistry{cfg: notProv}, prov), "POST", path, "", globalAdmin)
		require.Equal(t, http.StatusConflict, w.Code)
		require.Equal(t, "not_provisioned", errCode(t, w))
		require.Zero(t, prov.copies)
	})

	t.Run("409 while one is running; 500 on other failures; 503 without a provisioner", func(t *testing.T) {
		w := call(newLakehouseRouter(&fakeLakehouseRegistry{cfg: provisioned}, &fakeProvisioner{copyErr: ErrAuditCopyInProgress}), "POST", path, "", globalAdmin)
		require.Equal(t, http.StatusConflict, w.Code)
		require.Equal(t, "copy_in_progress", errCode(t, w))

		w = call(newLakehouseRouter(&fakeLakehouseRegistry{cfg: provisioned}, &fakeProvisioner{copyErr: errors.New("temporal down")}), "POST", path, "", globalAdmin)
		require.GreaterOrEqual(t, w.Code, 500)

		w = call(newLakehouseRouter(&fakeLakehouseRegistry{cfg: provisioned}, nil), "POST", path, "", globalAdmin)
		require.Equal(t, http.StatusServiceUnavailable, w.Code)
	})

	t.Run("only a global admin may start it", func(t *testing.T) {
		prov := &fakeProvisioner{}
		for _, who := range []*security.AuthInfo{ordinary, nil} {
			w := call(newLakehouseRouter(&fakeLakehouseRegistry{cfg: provisioned}, prov), "POST", path, "", who)
			require.Contains(t, []int{http.StatusForbidden, http.StatusUnauthorized}, w.Code)
		}
		require.Zero(t, prov.copies)
	})
}
