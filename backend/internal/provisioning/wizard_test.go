package provisioning

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/require"

	"github.com/hondyman/uisce/backend/internal/security"
)

type fakeStarter struct {
	order  *[]string
	err    error
	inputs []WizardInput
}

func (f *fakeStarter) StartWizard(_ context.Context, in WizardInput) (string, error) {
	*f.order = append(*f.order, "start")
	f.inputs = append(f.inputs, in)
	if f.err != nil {
		return "", f.err
	}
	return "wf-" + in.TenantCode, nil
}

type fakeWizSecrets struct {
	order  *[]string
	err    error
	paths  []string
	values []map[string]string
}

func (f *fakeWizSecrets) PutMap(_ context.Context, key string, values map[string]string) error {
	*f.order = append(*f.order, "secrets")
	f.paths = append(f.paths, key)
	f.values = append(f.values, values)
	return f.err
}

type knownRegions map[string]bool

func (k knownRegions) Known(code string) bool { return k[code] }

type wizRig struct {
	h     *WizardHandler
	st    *fakeStarter
	sec   *fakeWizSecrets
	mock  sqlmock.Sqlmock
	order []string
	r     http.Handler
}

func newWizRig(t *testing.T) *wizRig {
	t.Helper()
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	require.NoError(t, err)
	t.Cleanup(func() { db.Close() })
	g := &wizRig{mock: mock}
	g.st = &fakeStarter{order: &g.order}
	g.sec = &fakeWizSecrets{order: &g.order}
	g.h = &WizardHandler{DB: db, Starter: g.st, Regions: knownRegions{"us-east-1": true}, Secrets: g.sec}
	r := chi.NewRouter()
	g.h.RegisterWizardRoutes(r)
	g.r = r
	return g
}

// expectTenantFree sets the uniqueness query to find no match.
func (g *wizRig) expectTenantFree() {
	g.mock.ExpectQuery(`SELECT count\(\*\) FROM public.tenants`).WillReturnRows(sqlmock.NewRows([]string{"n"}).AddRow(0))
}

func (g *wizRig) do(body string, auth *security.AuthInfo) *httptest.ResponseRecorder {
	req := httptest.NewRequest("POST", "/system/tenants/wizard", strings.NewReader(body))
	if auth != nil {
		req = req.WithContext(security.WithAuthInfo(req.Context(), *auth))
	}
	w := httptest.NewRecorder()
	g.r.ServeHTTP(w, req)
	return w
}

const validWizard = `{"tenant_code":"acme","tenant_name":"Acme Corp","environment":"dev","region":"us-east-1"}`

// --- auth contract ---

func TestWizardRequiresAnAuthenticatedGlobalAdmin(t *testing.T) {
	g := newWizRig(t)
	require.Equal(t, http.StatusUnauthorized, g.do(validWizard, nil).Code)
	require.Equal(t, http.StatusForbidden, g.do(validWizard, ordinary).Code)
	require.Empty(t, g.order, "nothing may happen for an unauthorized caller")
}

// --- success path and ordering ---

func TestWizardStoresSecretsBeforeStartingTheWorkflow(t *testing.T) {
	g := newWizRig(t)
	g.expectTenantFree()
	body := `{"tenant_code":"acme","tenant_name":"Acme Corp","environment":"dev","region":"us-east-1",
		"allowlist":["203.0.113.10","10.0.0.0/8"],
		"ldap":{"host":"ldap.corp.example.internal","port":636,"base_dn":"dc=corp,dc=example","bind_dn":"cn=svc,dc=corp,dc=example","bind_password":"bind-secret-value"}}`
	w := g.do(body, globalAdmin)
	require.Equal(t, http.StatusAccepted, w.Code, w.Body.String())
	require.Equal(t, []string{"secrets", "start"}, g.order, "the password must be stored before the workflow starts")
	require.Equal(t, "tenants/acme/identity", g.sec.paths[0])
	require.Equal(t, "bind-secret-value", g.sec.values[0]["ldap_bind_password"])
}

// The workflow input and the response must never carry the password.
func TestWizardNeverPutsThePasswordInTheWorkflowInputOrResponse(t *testing.T) {
	g := newWizRig(t)
	g.expectTenantFree()
	body := `{"tenant_code":"acme","tenant_name":"Acme Corp","environment":"dev","region":"us-east-1",
		"ldap":{"host":"ldap.corp.example.internal","port":636,"base_dn":"dc=corp,dc=example","bind_dn":"cn=svc,dc=corp,dc=example","bind_password":"bind-secret-value"}}`
	w := g.do(body, globalAdmin)
	require.Equal(t, http.StatusAccepted, w.Code)
	raw, _ := json.Marshal(g.st.inputs[0])
	require.NotContains(t, string(raw), "bind-secret-value")
	require.NotContains(t, w.Body.String(), "bind-secret-value")
}

func TestWizardGeneratesTheTenantIdentifiersServerSide(t *testing.T) {
	g := newWizRig(t)
	g.expectTenantFree()
	body := `{"tenant_code":"acme","tenant_name":"Acme Corp","environment":"dev","region":"us-east-1","tenant_id":"11111111-1111-1111-1111-111111111111"}`
	w := g.do(body, globalAdmin)
	require.Equal(t, http.StatusBadRequest, w.Code, "a client-supplied tenant_id must be refused, not trusted")
}

// --- hostile and invalid input, refused before any side effect ---

func TestWizardRefusesInvalidInputBeforeAnySideEffect(t *testing.T) {
	cases := map[string]string{
		"injection code":      `{"tenant_code":"x; DROP DATABASE alpha;--","tenant_name":"A","environment":"dev","region":"us-east-1"}`,
		"uppercase code":      `{"tenant_code":"Acme","tenant_name":"A","environment":"dev","region":"us-east-1"}`,
		"leading digit code":  `{"tenant_code":"1acme","tenant_name":"A","environment":"dev","region":"us-east-1"}`,
		"empty name":          `{"tenant_code":"acme","tenant_name":"","environment":"dev","region":"us-east-1"}`,
		"overlong name":       `{"tenant_code":"acme","tenant_name":"` + strings.Repeat("n", 201) + `","environment":"dev","region":"us-east-1"}`,
		"unknown env":         `{"tenant_code":"acme","tenant_name":"A","environment":"staging","region":"us-east-1"}`,
		"env traversal":       `{"tenant_code":"acme","tenant_name":"A","environment":"../prod","region":"us-east-1"}`,
		"unknown region":      `{"tenant_code":"acme","tenant_name":"A","environment":"dev","region":"eu-west-1"}`,
		"region injection":    `{"tenant_code":"acme","tenant_name":"A","environment":"dev","region":"us-east-1; drop"}`,
		"wildcard allowlist":  `{"tenant_code":"acme","tenant_name":"A","environment":"dev","region":"us-east-1","allowlist":["*"]}`,
		"bad cidr":            `{"tenant_code":"acme","tenant_name":"A","environment":"dev","region":"us-east-1","allowlist":["10.0.0.0/33"]}`,
		"allowlist injection": `{"tenant_code":"acme","tenant_name":"A","environment":"dev","region":"us-east-1","allowlist":["10.0.0.1; drop"]}`,
		"unknown field":       `{"tenant_code":"acme","tenant_name":"A","environment":"dev","region":"us-east-1","role":"admin"}`,
		"ldap host injection": `{"tenant_code":"acme","tenant_name":"A","environment":"dev","region":"us-east-1","ldap":{"host":"ldap;evil","port":636,"base_dn":"dc=a","bind_dn":"cn=b","bind_password":"p"}}`,
		"ldap port zero":      `{"tenant_code":"acme","tenant_name":"A","environment":"dev","region":"us-east-1","ldap":{"host":"ldap.x","port":0,"base_dn":"dc=a","bind_dn":"cn=b","bind_password":"p"}}`,
		"ldap dn quote":       `{"tenant_code":"acme","tenant_name":"A","environment":"dev","region":"us-east-1","ldap":{"host":"ldap.x","port":636,"base_dn":"dc=\"a\"","bind_dn":"cn=b","bind_password":"p"}}`,
		"ldap empty password": `{"tenant_code":"acme","tenant_name":"A","environment":"dev","region":"us-east-1","ldap":{"host":"ldap.x","port":636,"base_dn":"dc=a","bind_dn":"cn=b","bind_password":""}}`,
	}
	for name, body := range cases {
		t.Run(name, func(t *testing.T) {
			g := newWizRig(t)
			w := g.do(body, globalAdmin)
			require.Equal(t, http.StatusBadRequest, w.Code, w.Body.String())
			require.Empty(t, g.order, "a secret was written or a workflow started for invalid input")
		})
	}
}

func TestWizardRefusesAnOversizedBody(t *testing.T) {
	g := newWizRig(t)
	w := g.do(`{"tenant_code":"acme","tenant_name":"`+strings.Repeat("n", wizardMaxBody)+`"}`, globalAdmin)
	require.Equal(t, http.StatusBadRequest, w.Code)
	require.Empty(t, g.order)
}

func TestWizardFailsClosedWithoutARegionRegistry(t *testing.T) {
	g := newWizRig(t)
	g.h.Regions = nil
	require.Equal(t, http.StatusServiceUnavailable, g.do(validWizard, globalAdmin).Code)
	require.Empty(t, g.order)
}

// --- existing tenants and database errors ---

func TestWizardRefusesAnExistingTenant(t *testing.T) {
	g := newWizRig(t)
	g.mock.ExpectQuery(`SELECT count\(\*\) FROM public.tenants`).WillReturnRows(sqlmock.NewRows([]string{"n"}).AddRow(1))
	require.Equal(t, http.StatusConflict, g.do(validWizard, globalAdmin).Code)
	require.Empty(t, g.order)
}

// The uniqueness check fails closed. A database error is not "no such tenant".
func TestWizardUniquenessCheckFailsClosed(t *testing.T) {
	g := newWizRig(t)
	g.mock.ExpectQuery(`SELECT count\(\*\) FROM public.tenants`).WillReturnError(errors.New("connection reset"))
	w := g.do(validWizard, globalAdmin)
	require.Equal(t, http.StatusInternalServerError, w.Code)
	require.Empty(t, g.order, "a failed check must not proceed to provisioning")
}

// --- failures after validation ---

func TestWizardStopsWhenTheSecretStoreFails(t *testing.T) {
	g := newWizRig(t)
	g.expectTenantFree()
	g.sec.err = errors.New("infisical 503 with the password")
	body := `{"tenant_code":"acme","tenant_name":"Acme Corp","environment":"dev","region":"us-east-1",
		"ldap":{"host":"ldap.x","port":636,"base_dn":"dc=a","bind_dn":"cn=b","bind_password":"bind-secret-value"}}`
	w := g.do(body, globalAdmin)
	require.Equal(t, http.StatusInternalServerError, w.Code)
	require.NotContains(t, w.Body.String(), "bind-secret-value")
	require.NotContains(t, g.order, "start", "the workflow started without its credential")
}

func TestWizardReportsStartFailureWithoutDetail(t *testing.T) {
	g := newWizRig(t)
	g.expectTenantFree()
	g.st.err = errors.New("temporal refused: detail")
	w := g.do(validWizard, globalAdmin)
	require.Equal(t, http.StatusInternalServerError, w.Code)
	require.NotContains(t, w.Body.String(), "detail")
}

func TestWizardAcceptedResponseCarriesIdentifiersAndNoSecrets(t *testing.T) {
	g := newWizRig(t)
	g.expectTenantFree()
	w := g.do(validWizard, globalAdmin)
	require.Equal(t, http.StatusAccepted, w.Code)
	var resp WizardResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.NotEmpty(t, resp.TenantID)
	require.NotEmpty(t, resp.InstanceID)
	require.Equal(t, "provisioning", resp.Status)
}
