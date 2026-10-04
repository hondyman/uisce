package activities_test

import (
	"context"
	"os"
	"reflect"
	"runtime"
	"sort"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/hondyman/uisce/backend/internal/temporal/activities"
)

type recorder struct{ names []string }

func (r *recorder) RegisterActivity(a interface{}) {
	full := runtime.FuncForPC(reflect.ValueOf(a).Pointer()).Name()
	name := full[strings.LastIndex(full, ".")+1:]
	r.names = append(r.names, strings.TrimSuffix(name, "-fm"))
}

// Every activity the compensation-fixed saga calls that is not in the workers' own lists must come
// from RegisterTenantDatabaseActivities. InspectProvisioningState used to be registered nowhere,
// so every run failed at step 3 and rolled the tenant back.
func TestRegisterTenantDatabaseActivities_RegistersWhatTheSagaCalls(t *testing.T) {
	rec := &recorder{}
	(&activities.TenantProvisioningActivities{}).RegisterTenantDatabaseActivities(rec)
	sort.Strings(rec.names)
	require.Equal(t, []string{
		"ActivateTenantDatabase", "ApplyTenantMigrations", "ApplyTenantStructure", "BindTenantDatabase", "InspectProvisioningState",
		"PlanTenantStructure", "ProbeTenantDatabase", "ProvisionTenantDatabaseAccess", "RollbackTenantDatabase",
	}, rec.names)
}

func TestBothWorkersRegisterTheTenantDatabaseActivitiesAndNeverAsBPSafe(t *testing.T) {
	for _, f := range []string{"../../../cmd/worker/main.go", "../worker.go"} {
		src, err := os.ReadFile(f)
		require.NoError(t, err, f)
		body := string(src)
		for _, want := range []string{"RegisterTenantDatabaseActivities(", "ConfigureTenantDatabaseFromEnv()"} {
			if !strings.Contains(body, want) {
				t.Errorf("%s does not contain %q; the provisioning saga would run on this worker without its tenant-database activities", f, want)
			}
		}
		for _, line := range strings.Split(body, "\n") {
			if !strings.Contains(line, "RegisterSafeActivity") {
				continue
			}
			for _, name := range []string{"BindTenantDatabase", "ProvisionTenantDatabaseAccess", "ApplyTenantMigrations",
				"PlanTenantStructure", "ApplyTenantStructure", "ProbeTenantDatabase", "ActivateTenantDatabase", "RollbackTenantDatabase", "InspectProvisioningState"} {
				if !strings.Contains(line, name) {
					continue
				}
				t.Errorf("%s: these activities create roles and credentials and must never be BP-Designer client-safe: %s", f, strings.TrimSpace(line))
			}
		}
	}
}

// CreateTenantDatabase and RollbackCreateTenantDatabase build CREATE/DROP DATABASE from the name.
// Now that a request can reach them they must refuse anything that is not a plain identifier,
// before they open any connection (so no database is needed here).
func TestDatabaseActivitiesRefuseUnsafeNames(t *testing.T) {
	acts := &activities.TenantProvisioningActivities{Logger: zap.NewNop().Sugar()}
	for _, bad := range []string{`x"; DROP DATABASE postgres; --`, "tenant acme", "Tenant_Acme", "", "tenant_acme;", strings.Repeat("a", 64)} {
		err := acts.CreateTenantDatabase(context.Background(), bad)
		require.Error(t, err, bad)
		require.True(t, isNonRetryableOf(err, "TenantDatabaseInvalidInput"), "%q: %v", bad, err)
		err = acts.RollbackCreateTenantDatabase(context.Background(), bad)
		require.Error(t, err, bad)
		require.True(t, isNonRetryableOf(err, "TenantDatabaseInvalidInput"), "%q: %v", bad, err)
	}
}

func TestConfigureFromEnv_TheRoleGroupMustBeAPlainIdentifier(t *testing.T) {
	for name, tc := range map[string]struct{ env, want string }{
		"unset":              {"", ""},
		"a plain identifier": {"ivy_tenant_apps", "ivy_tenant_apps"},
		"upper case":         {"Ivy_Apps", ""},
		"an injected name":   {`x; DROP ROLE postgres`, ""},
		"a quoted name":      {`"apps"`, ""},
		"too long":           {strings.Repeat("a", 64), ""},
	} {
		t.Run(name, func(t *testing.T) {
			t.Setenv("TENANT_DB_ROLE_GROUP", tc.env)
			a := &activities.TenantProvisioningActivities{Logger: zap.NewNop().Sugar()}
			a.ConfigureTenantDatabaseFromEnv()
			require.Equal(t, tc.want, a.RoleGroup, "a group name that is not a plain identifier is ignored, never used")
		})
	}
}
