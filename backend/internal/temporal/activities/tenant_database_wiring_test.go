package activities_test

import (
	"context"
	"fmt"
	"os"
	"reflect"
	"runtime"
	"sort"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/hondyman/uisce/backend/internal/provisioning"
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

// Unset is a legitimate no-op. Set and malformed must STOP the run, with the offending value in the message: a gate that is
// switched off by a typo is not a gate.
func TestConfigureFromEnv_ASetButMalformedRoleGroupRefusesEveryTenantDatabaseStep(t *testing.T) {
	admin := activities.TenantDatabaseAdmin{Host: "h", Port: 5432, User: "u", Password: "p"}
	in := provisioning.TenantDatabaseInput{TenantID: "11111111-2222-3333-4444-555555555555", App: "orm", DatabaseName: "tenant_acme",
		TemplateDatasourceID: "441f62c9-aad1-481d-9aab-62943fa11cd3"}
	steps := map[string]func(a *activities.TenantProvisioningActivities) error{
		"PlanTenantStructure": func(a *activities.TenantProvisioningActivities) error {
			_, err := a.PlanTenantStructure(context.Background(), in)
			return err
		},
		"BindTenantDatabase": func(a *activities.TenantProvisioningActivities) error {
			_, err := a.BindTenantDatabase(context.Background(), in)
			return err
		},
		"ProvisionTenantDatabaseAccess": func(a *activities.TenantProvisioningActivities) error {
			return a.ProvisionTenantDatabaseAccess(context.Background(), in)
		},
		"ApplyTenantStructure": func(a *activities.TenantProvisioningActivities) error {
			x := in
			x.StructureHash = "h"
			_, err := a.ApplyTenantStructure(context.Background(), x)
			return err
		},
	}
	for name, bad := range map[string]string{
		"upper case": "Ivy_Apps", "an injected name": `x; DROP ROLE postgres`, "a quoted name": `"apps"`,
		"too long": strings.Repeat("a", 64), "a dash": "ivy-apps", "a leading digit": "1apps",
	} {
		t.Run(name, func(t *testing.T) {
			t.Setenv("TENANT_DB_ROLE_GROUP", bad)
			a := &activities.TenantProvisioningActivities{Logger: zap.NewNop().Sugar(), TenantDB: admin}
			a.ConfigureTenantDatabaseFromEnv()
			require.Equal(t, bad, a.RoleGroup, "the value is kept, not dropped: dropping it would turn the gate off")
			for step, run := range steps {
				err := run(a)
				require.Error(t, err, step)
				require.True(t, isNonRetryableOf(err, "TenantDatabaseNotConfigured"), "%s: %v", step, err)
				require.ErrorContains(t, err, fmt.Sprintf("%q", bad), "%s must name the offending value (quoted, so it is unambiguous in a log)", step)
				require.ErrorContains(t, err, "TENANT_DB_ROLE_GROUP", step)
			}
		})
	}
}

func TestConfigureFromEnv_AValidOrUnsetRoleGroupIsAccepted(t *testing.T) {
	for name, tc := range map[string]string{"unset": "", "a plain identifier": "ivy_tenant_apps", "with digits": "ivy_apps_2"} {
		t.Run(name, func(t *testing.T) {
			t.Setenv("TENANT_DB_ROLE_GROUP", tc)
			a := &activities.TenantProvisioningActivities{Logger: zap.NewNop().Sugar(), TenantDB: activities.TenantDatabaseAdmin{Host: "h", Port: 5432, User: "u", Password: "p"}}
			a.ConfigureTenantDatabaseFromEnv()
			require.Equal(t, tc, a.RoleGroup)
			// no refusal on account of the group: the step fails later, for other reasons, or succeeds
			_, err := a.BindTenantDatabase(context.Background(), provisioning.TenantDatabaseInput{TenantID: "11111111-2222-3333-4444-555555555555", App: "orm", DatabaseName: "Bad Name"})
			require.Error(t, err)
			require.NotContains(t, err.Error(), "TENANT_DB_ROLE_GROUP")
		})
	}
}
