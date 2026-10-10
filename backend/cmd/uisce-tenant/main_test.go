package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/hondyman/uisce/backend/internal/provisioning"
)

// fakeCatalog backs the real provisioning.Describe, so the CLI is tested against the server's
// actual rules and not a copy of them.
type fakeCatalog struct{}

func (fakeCatalog) Regions(context.Context) ([]provisioning.Region, error) {
	return []provisioning.Region{
		{Code: "us-east-1", Name: "US East (N. Virginia)", ClusterHost: "100.84.50.65", ClusterPort: 5432},
		{Code: "eu-west-1", Name: "Europe (Ireland)"},
	}, nil
}
func (fakeCatalog) Products(context.Context) ([]provisioning.Product, error) {
	return []provisioning.Product{{Code: "ORM", Name: "Order and risk management"}}, nil
}
func (fakeCatalog) DatabaseNameTaken(_ context.Context, n string) (bool, error) {
	return n == "taken_orm", nil
}

type fakeServer struct {
	*httptest.Server
	mu         sync.Mutex
	provisions []provisioning.ProvisionTenantRequest
	describes  int
	statuses   []provisioning.ProvisioningStatus // returned in order, the last one repeats
	polled     int
	denyToken  bool
}

func newServer(t *testing.T, statuses ...provisioning.ProvisioningStatus) *fakeServer {
	t.Helper()
	f := &fakeServer{statuses: statuses}
	mux := http.NewServeMux()
	mux.HandleFunc("/token", func(w http.ResponseWriter, r *http.Request) {
		if f.denyToken {
			http.Error(w, `{"error_description":"bad secret"}`, http.StatusUnauthorized)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"access_token": "tok", "expires_in": 300})
	})
	mux.HandleFunc("/api/system/tenants/provision/describe", func(w http.ResponseWriter, r *http.Request) {
		var req provisioning.ProvisionTenantRequest
		_ = json.NewDecoder(r.Body).Decode(&req)
		d, err := provisioning.Describe(r.Context(), req, fakeCatalog{})
		require.NoError(t, err)
		f.mu.Lock()
		f.describes++
		f.mu.Unlock()
		_ = json.NewEncoder(w).Encode(d)
	})
	mux.HandleFunc("/api/system/tenants/provision", func(w http.ResponseWriter, r *http.Request) {
		var req provisioning.ProvisionTenantRequest
		_ = json.NewDecoder(r.Body).Decode(&req)
		d, _ := provisioning.Describe(r.Context(), req, fakeCatalog{})
		if !d.Complete {
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(d)
			return
		}
		f.mu.Lock()
		f.provisions = append(f.provisions, req)
		f.mu.Unlock()
		_ = json.NewEncoder(w).Encode(provisioning.ProvisionTenantResponse{
			WorkflowID: "tenant-provisioning-xyz_i-1234", TenantID: "t-1", InstanceID: "i-1",
			DatabaseName: d.Plan.Databases[0].Database, Status: "provisioning", Plan: d.Plan,
		})
	})
	mux.HandleFunc("/api/system/tenants/-/provision/", func(w http.ResponseWriter, r *http.Request) {
		f.mu.Lock()
		i := f.polled
		if i >= len(f.statuses) {
			i = len(f.statuses) - 1
		}
		f.polled++
		st := f.statuses[i]
		f.mu.Unlock()
		st.WorkflowID = strings.TrimPrefix(r.URL.Path, "/api/system/tenants/-/provision/")
		_ = json.NewEncoder(w).Encode(st)
	})
	f.Server = httptest.NewServer(mux)
	t.Cleanup(f.Close)
	return f
}

func (f *fakeServer) env(k string) string {
	return map[string]string{
		"UISCE_URL": f.URL, "UISCE_TOKEN_URL": f.URL + "/token", "UISCE_CLIENT_ID": "cli", "UISCE_CLIENT_SECRET": "s",
	}[k]
}

func (f *fakeServer) run(args []string, stdin string) (code int, stdout, stderr string) {
	var out, errb strings.Builder
	code = run(context.Background(), append(args, "--poll", "1ms"), f.env, strings.NewReader(stdin), &out, &errb)
	return code, out.String(), errb.String()
}

var (
	running = provisioning.ProvisioningStatus{Status: "provisioning", Step: "CreateTenantDatabaseInRegion"}
	done    = provisioning.ProvisioningStatus{Status: "completed", TenantID: "t-1", DatabaseName: "abc_orm"}
)

func TestCreate_FromFlags(t *testing.T) {
	f := newServer(t, running, running, done)
	code, out, errs := f.run([]string{"create", "--name", "XYZ Investments", "--region", "US East", "--product", "orm", "--label", "ABC", "--yes"}, "")
	require.Equal(t, exitOK, code, errs+out)
	require.Len(t, f.provisions, 1)
	got := f.provisions[0]
	require.Equal(t, "us-east-1", got.Region, "free text is resolved before the request is sent")
	require.Equal(t, []provisioning.ProductRequest{{Product: "orm", Label: "abc"}}, got.Products)
	require.Contains(t, out, "abc_orm")
	require.Contains(t, out, "CreateTenantDatabaseInRegion", "progress is shown")
	require.Contains(t, out, "is active")
}

func TestCreate_PromptsForWhatIsMissing(t *testing.T) {
	f := newServer(t, done)
	// Answers, in the order asked: region (pick 1), product (orm), label, then confirm.
	code, out, errs := f.run([]string{"create", "--name", "XYZ Investments"}, "1\norm\nabc\ny\n")
	require.Equal(t, exitOK, code, errs+out)
	require.Len(t, f.provisions, 1)
	require.Equal(t, "us-east-1", f.provisions[0].Region)
	require.Equal(t, provisioning.ProductRequest{Product: "orm", Label: "abc"}, f.provisions[0].Products[0])
	require.Contains(t, out, "Region", "the region question is shown")
	require.Contains(t, out, "us-east-1 (US East (N. Virginia))", "with the allowed values")
	require.Contains(t, out, "Create this tenant?")
}

func TestCreate_AnInvalidAnswerIsAskedAgain(t *testing.T) {
	f := newServer(t, done)
	code, out, errs := f.run([]string{"create", "--name", "XYZ", "--region", "us-east-1", "--product", "orm", "--label", "a;b"}, "abc\ny\n")
	require.Equal(t, exitOK, code, errs+out)
	require.Contains(t, out, "label must", "the server's reason is shown")
	require.Equal(t, "abc", f.provisions[0].Products[0].Label)
}

func TestCreate_DecliningCreatesNothing(t *testing.T) {
	f := newServer(t, done)
	code, out, _ := f.run([]string{"create", "--name", "XYZ", "--region", "us-east-1", "--product", "orm", "--label", "abc"}, "n\n")
	require.Equal(t, exitOK, code)
	require.Empty(t, f.provisions)
	require.Contains(t, out, "Nothing was created")
}

func TestCreate_NoPromptFailsWithWhatIsMissing(t *testing.T) {
	f := newServer(t, done)
	code, _, errs := f.run([]string{"create", "--name", "XYZ", "--no-prompt", "--yes"}, "")
	require.Equal(t, exitInvalid, code)
	require.Contains(t, errs, "missing  region")
	require.Contains(t, errs, "missing  products")
	require.Empty(t, f.provisions)
}

func TestCreate_NoPromptNeedsYes(t *testing.T) {
	f := newServer(t, done)
	code, _, errs := f.run([]string{"create", "--name", "XYZ", "--region", "us-east-1", "--product", "orm", "--label", "abc", "--no-prompt"}, "")
	require.Equal(t, exitUsage, code)
	require.Contains(t, errs, "--yes")
	require.Empty(t, f.provisions)
}

func TestCreate_InputEndingMidwayIsAUsageError(t *testing.T) {
	f := newServer(t, done)
	code, _, errs := f.run([]string{"create", "--name", "XYZ"}, "1\n")
	require.Equal(t, exitUsage, code)
	require.Contains(t, errs, "input ended")
	require.Empty(t, f.provisions)
}

func TestCreate_ARefusedRegionIsReportedNotRetriedForever(t *testing.T) {
	f := newServer(t, done)
	code, _, errs := f.run([]string{"create", "--name", "XYZ", "--region", "eu-west-1", "--product", "orm", "--label", "abc", "--no-prompt"}, "")
	require.Equal(t, exitInvalid, code)
	require.Contains(t, errs, "no Postgres cluster")
	require.Empty(t, f.provisions)
}

func TestCreate_AFailedRunExitsOneWithTheReason(t *testing.T) {
	f := newServer(t, running, provisioning.ProvisioningStatus{Status: "failed", Error: "saga failed at AssertRegionCluster: boom"})
	code, out, _ := f.run([]string{"create", "--name", "XYZ", "--region", "us-east-1", "--product", "orm", "--label", "abc", "--yes"}, "")
	require.Equal(t, exitFailed, code)
	require.Contains(t, out, "AssertRegionCluster")
}

func TestCreate_NoWaitReturnsOnceStarted(t *testing.T) {
	f := newServer(t, running)
	code, out, _ := f.run([]string{"create", "--name", "XYZ", "--region", "us-east-1", "--product", "orm", "--label", "abc", "--yes", "--no-wait"}, "")
	require.Equal(t, exitOK, code)
	require.Contains(t, out, "Started tenant-provisioning-xyz_i-1234")
	require.Zero(t, f.polled)
}

func TestCreate_TimesOutWhileStillProvisioning(t *testing.T) {
	f := newServer(t, running)
	code, _, errs := f.run([]string{"create", "--name", "XYZ", "--region", "us-east-1", "--product", "orm", "--label", "abc", "--yes", "--timeout", "30ms"}, "")
	require.Equal(t, exitTimeout, code)
	require.Contains(t, errs, "status --workflow")
}

func TestUnauthorizedAndMisconfigured(t *testing.T) {
	f := newServer(t, done)
	f.denyToken = true
	code, _, _ := f.run([]string{"create", "--name", "XYZ", "--region", "us-east-1", "--product", "orm", "--label", "abc", "--yes"}, "")
	require.Equal(t, exitUnauthorized, code)

	var out, errb strings.Builder
	code = run(context.Background(), []string{"create", "--name", "X"}, func(string) string { return "" }, strings.NewReader(""), &out, &errb)
	require.Equal(t, exitUsage, code)
	require.Contains(t, errb.String(), "UISCE_URL")
	require.NotContains(t, errb.String(), "secret", "no secret is echoed")

	code = run(context.Background(), []string{"frobnicate"}, f.env, strings.NewReader(""), &out, &errb)
	require.Equal(t, exitUsage, code)
}

func TestStatus(t *testing.T) {
	f := newServer(t, done)
	code, out, _ := f.run([]string{"status", "--workflow", "tenant-provisioning-xyz_i-1234"}, "")
	require.Equal(t, exitOK, code)
	require.Contains(t, out, "status=completed")
	require.Contains(t, out, "database=abc_orm")
}
