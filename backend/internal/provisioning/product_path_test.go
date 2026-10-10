package provisioning

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/require"
	enumspb "go.temporal.io/api/enums/v1"
	"go.temporal.io/api/serviceerror"
	workflowpb "go.temporal.io/api/workflow/v1"
	"go.temporal.io/api/workflowservice/v1"
	"go.uber.org/zap"
)

// describingTemporal answers DescribeWorkflowExecution as a test says, so the idempotency paths
// can be driven.
type describingTemporal struct {
	*fakeTemporal
	describe func(id string) (*workflowservice.DescribeWorkflowExecutionResponse, error)
}

func (d *describingTemporal) DescribeWorkflowExecution(_ context.Context, id, _ string) (*workflowservice.DescribeWorkflowExecutionResponse, error) {
	d.described = append(d.described, id)
	return d.describe(id)
}

func notStarted(string) (*workflowservice.DescribeWorkflowExecutionResponse, error) {
	return nil, serviceerror.NewNotFound("no such workflow")
}

func inState(s enumspb.WorkflowExecutionStatus) func(string) (*workflowservice.DescribeWorkflowExecutionResponse, error) {
	return func(string) (*workflowservice.DescribeWorkflowExecutionResponse, error) {
		return &workflowservice.DescribeWorkflowExecutionResponse{
			WorkflowExecutionInfo: &workflowpb.WorkflowExecutionInfo{Status: s},
		}, nil
	}
}

func newProductRig(t *testing.T, describe func(string) (*workflowservice.DescribeWorkflowExecutionResponse, error)) (*rig, *describingTemporal) {
	t.Helper()
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	require.NoError(t, err)
	t.Cleanup(func() { db.Close() })
	tc := &describingTemporal{fakeTemporal: &fakeTemporal{}, describe: describe}
	h := NewProvisioningHandler(tc, db, zap.NewNop().Sugar())
	h.catalog = testCatalog()
	r := chi.NewRouter()
	h.RegisterAdminRoutes(r)
	return &rig{h: h, tc: tc.fakeTemporal, mock: mock, r: r}, tc
}

func (g *rig) expectTenantFreeAndGold() {
	g.mock.ExpectQuery(`SELECT count\(\*\) FROM public.tenants`).WillReturnRows(sqlmock.NewRows([]string{"n"}).AddRow(0))
	g.mock.ExpectQuery(`FROM public.tenants t`).WillReturnRows(
		sqlmock.NewRows([]string{"a", "b", "c"}).AddRow("gold-t", "gold-i", "crims"))
}

const productBody = `{"tenant_name":"XYZ Investments","region":"US East","products":[{"product":"ORM","label":"ABC"}]}`

func TestProductPath_CreatesTheTenantFromTheParameters(t *testing.T) {
	g, _ := newProductRig(t, notStarted)
	g.expectTenantFreeAndGold()

	w := g.do("POST", provPath, productBody, globalAdmin)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	require.Len(t, g.tc.started, 1)

	in := g.tc.inputs[0]
	require.Equal(t, "abc_orm", in.DatabaseName, "label ABC and product ORM name the database")
	require.Equal(t, "orm", in.App)
	require.True(t, in.StructureFromGoldCopy, "the structure is compiled from the gold copy's scan")
	require.Equal(t, []string{"orm"}, in.ProductCodes)
	require.Equal(t, "us-east-1", in.Region, "free text 'US East' is resolved to the configured region")
	require.Equal(t, "100.84.50.65", in.ClusterHost)
	require.Equal(t, 5432, in.ClusterPort)
	require.True(t, in.Seed)
	require.Equal(t, "xyz_i", in.TenantCode)
	require.Equal(t, "admin-1", in.RequesterID)
	require.Equal(t, "crims", in.GoldCopyDatabase)
	require.Equal(t, enumspb.WORKFLOW_ID_REUSE_POLICY_ALLOW_DUPLICATE_FAILED_ONLY, g.tc.started[0].WorkflowIDReusePolicy)

	var resp ProvisionTenantResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.Equal(t, g.tc.started[0].ID, resp.WorkflowID)
	require.Equal(t, in.TenantID, resp.TenantID, "the response names the ids the run uses")
	require.Equal(t, "abc_orm", resp.Plan.Databases[0].Database)
	require.NoError(t, g.mock.ExpectationsWereMet())
}

func TestProductPath_TheSameRequestFindsTheRunningRun(t *testing.T) {
	g, _ := newProductRig(t, inState(enumspb.WORKFLOW_EXECUTION_STATUS_RUNNING))

	w := g.do("POST", provPath, productBody, globalAdmin)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	require.Empty(t, g.tc.started, "a repeated request must not start a second run")
	require.NoError(t, g.mock.ExpectationsWereMet(), "and must not query anything else")

	var resp ProvisionTenantResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.Equal(t, g.tc.described[0], resp.WorkflowID)
}

func TestProductPath_ARequestThatAlreadyCompletedIsAConflict(t *testing.T) {
	g, _ := newProductRig(t, inState(enumspb.WORKFLOW_EXECUTION_STATUS_COMPLETED))
	w := g.do("POST", provPath, productBody, globalAdmin)
	require.Equal(t, http.StatusConflict, w.Code, w.Body.String())
	require.Empty(t, g.tc.started)
}

func TestProductPath_ARequestThatFailedMayBeTriedAgain(t *testing.T) {
	g, _ := newProductRig(t, inState(enumspb.WORKFLOW_EXECUTION_STATUS_FAILED))
	g.expectTenantFreeAndGold()
	w := g.do("POST", provPath, productBody, globalAdmin)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	require.Len(t, g.tc.started, 1)
}

func TestProductPath_ALookupFailureIsNotTreatedAsNotStarted(t *testing.T) {
	g, _ := newProductRig(t, func(string) (*workflowservice.DescribeWorkflowExecutionResponse, error) {
		return nil, context.DeadlineExceeded
	})
	w := g.do("POST", provPath, productBody, globalAdmin)
	require.Equal(t, http.StatusBadGateway, w.Code, "an unknown state must fail closed, not start a second run")
	require.Empty(t, g.tc.started)
}

func TestProductPath_AnExistingTenantIsAConflict(t *testing.T) {
	g, _ := newProductRig(t, notStarted)
	g.mock.ExpectQuery(`SELECT count\(\*\) FROM public.tenants`).WillReturnRows(sqlmock.NewRows([]string{"n"}).AddRow(1))
	w := g.do("POST", provPath, productBody, globalAdmin)
	require.Equal(t, http.StatusConflict, w.Code)
	require.Empty(t, g.tc.started)
}

func TestProductPath_ADatabaseErrorFailsClosed(t *testing.T) {
	g, _ := newProductRig(t, notStarted)
	g.mock.ExpectQuery(`SELECT count\(\*\) FROM public.tenants`).WillReturnError(context.DeadlineExceeded)
	w := g.do("POST", provPath, productBody, globalAdmin)
	require.Equal(t, http.StatusInternalServerError, w.Code)
	require.Empty(t, g.tc.started, "an unreadable tenant table must not read as 'no such tenant'")
}

func TestProductPath_IncompleteRequestsAreRefusedWithWhatIsMissing(t *testing.T) {
	cases := map[string]string{
		"no label":      `{"tenant_name":"XYZ","region":"us-east-1","products":[{"product":"orm"}]}`,
		"no region":     `{"tenant_name":"XYZ","region":"","products":[{"product":"orm","label":"abc"}],"instance_name":"p"}`,
		"region only":   `{"tenant_name":"XYZ","region":"us-east-1"}`,
		"unknown":       `{"tenant_name":"XYZ","region":"mars","products":[{"product":"orm","label":"abc"}]}`,
		"no cluster":    `{"tenant_name":"XYZ","region":"eu-west-1","products":[{"product":"orm","label":"abc"}]}`,
		"taken":         `{"tenant_name":"XYZ","region":"us-east-1","products":[{"product":"orm","label":"taken"}]}`,
		"unsafe label":  `{"tenant_name":"XYZ","region":"us-east-1","products":[{"product":"orm","label":"x\"; DROP DATABASE postgres; --"}]}`,
		"two products":  `{"tenant_name":"XYZ","region":"us-east-1","products":[{"product":"orm","label":"a"},{"product":"mdm","label":"a"}]}`,
		"unknown prod":  `{"tenant_name":"XYZ","region":"us-east-1","products":[{"product":"nope","label":"abc"}]}`,
		"unsafe code":   `{"tenant_name":"XYZ","tenant_code":"X;Y","region":"us-east-1","products":[{"product":"orm","label":"abc"}]}`,
		"app with prod": `{"tenant_name":"XYZ","app":"orm","region":"us-east-1","products":[{"product":"orm","label":"abc"}]}`,
	}
	for name, body := range cases {
		g, tc := newProductRig(t, notStarted)
		w := g.do("POST", provPath, body, globalAdmin)
		require.Equal(t, http.StatusBadRequest, w.Code, "%s: %s", name, w.Body.String())
		require.Empty(t, g.tc.started, name)
		require.Empty(t, tc.described, "%s must be refused before Temporal is asked anything", name)
		require.NoError(t, g.mock.ExpectationsWereMet(), "%s must not query the tenant tables", name)
	}
}

func TestProductPath_OnlyAGlobalAdmin(t *testing.T) {
	for name, who := range map[string]string{"anonymous": "", "ordinary": "user"} {
		g, _ := newProductRig(t, notStarted)
		auth := ordinary
		if who == "" {
			auth = nil
		}
		for _, path := range []string{provPath, provPath + "/describe"} {
			w := g.do("POST", path, productBody, auth)
			require.Contains(t, []int{http.StatusUnauthorized, http.StatusForbidden}, w.Code, "%s %s", name, path)
		}
		require.Empty(t, g.tc.started)
	}
}

func TestDescribeEndpoint_ReportsWhatIsMissingAndThePlan(t *testing.T) {
	g, _ := newProductRig(t, notStarted)

	w := g.do("POST", provPath+"/describe", `{"tenant_name":"XYZ Investments"}`, globalAdmin)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	var d DescribeResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &d))
	require.False(t, d.Complete)
	require.Contains(t, fields(d.Missing), "region")
	require.Contains(t, fields(d.Missing), "products")

	w = g.do("POST", provPath+"/describe", productBody, globalAdmin)
	require.Equal(t, http.StatusOK, w.Code)
	d = DescribeResponse{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &d))
	require.True(t, d.Complete, w.Body.String())
	require.Equal(t, "abc_orm", d.Plan.Databases[0].Database)
	require.Equal(t, "us-east-1", d.Normalized.Region, "a client can send the normalized request back as it is")
	require.Empty(t, g.tc.started, "describe creates nothing")
	require.NoError(t, g.mock.ExpectationsWereMet(), "describe reads only the catalog")
}
