package activities_test

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/hondyman/uisce/backend/internal/provisioning"
	"github.com/hondyman/uisce/backend/internal/temporal/activities"
)

// The product path's activities refuse before they open a connection: an unsafe name, a missing
// cluster, an unconfigured worker, or a region that runs somewhere this worker is not the
// administrator of. None of these tests needs a database, because each refusal comes first.

func regionActs(host string, port int) *activities.TenantProvisioningActivities {
	return &activities.TenantProvisioningActivities{
		Logger:   zap.NewNop().Sugar(),
		TenantDB: activities.TenantDatabaseAdmin{Host: host, Port: port, User: "admin", Password: "x"},
	}
}

func regionSteps(a *activities.TenantProvisioningActivities) map[string]func(provisioning.RegionDatabaseInput) error {
	ctx := context.Background()
	return map[string]func(provisioning.RegionDatabaseInput) error{
		"AssertRegionCluster":          func(in provisioning.RegionDatabaseInput) error { return a.AssertRegionCluster(ctx, in) },
		"CreateTenantDatabaseInRegion": func(in provisioning.RegionDatabaseInput) error { return a.CreateTenantDatabaseInRegion(ctx, in) },
		"RollbackCreateTenantDatabaseInRegion": func(in provisioning.RegionDatabaseInput) error {
			return a.RollbackCreateTenantDatabaseInRegion(ctx, in)
		},
	}
}

func TestRegionActivities_RefuseUnsafeNames(t *testing.T) {
	a := regionActs("100.84.50.65", 5432)
	for step, run := range regionSteps(a) {
		for _, bad := range []string{`x"; DROP DATABASE postgres; --`, "Abc_Orm", "abc orm", "", "abc;", strings.Repeat("a", 64)} {
			err := run(provisioning.RegionDatabaseInput{Region: "us-east-1", Host: "100.84.50.65", Port: 5432, DatabaseName: bad})
			require.Error(t, err, "%s %q", step, bad)
			require.True(t, isNonRetryableOf(err, "TenantDatabaseInvalidInput"), "%s %q: %v", step, bad, err)
		}
	}
}

func TestRegionActivities_RefuseAnIncompleteCluster(t *testing.T) {
	a := regionActs("100.84.50.65", 5432)
	for step, run := range regionSteps(a) {
		for name, in := range map[string]provisioning.RegionDatabaseInput{
			"no region": {Host: "100.84.50.65", Port: 5432, DatabaseName: "abc_orm"},
			"no host":   {Region: "us-east-1", Port: 5432, DatabaseName: "abc_orm"},
			"no port":   {Region: "us-east-1", Host: "100.84.50.65", DatabaseName: "abc_orm"},
		} {
			err := run(in)
			require.True(t, isNonRetryableOf(err, "TenantDatabaseInvalidInput"), "%s %s: %v", step, name, err)
		}
	}
}

func TestRegionActivities_RefuseAWorkerThatIsNotConfigured(t *testing.T) {
	a := &activities.TenantProvisioningActivities{Logger: zap.NewNop().Sugar()}
	for step, run := range regionSteps(a) {
		err := run(provisioning.RegionDatabaseInput{Region: "us-east-1", Host: "100.84.50.65", Port: 5432, DatabaseName: "abc_orm"})
		require.True(t, isNonRetryableOf(err, "TenantDatabaseNotConfigured"), "%s: %v", step, err)
	}
}

// A region whose cluster is not the one this worker administers must not get a database on the
// worker's cluster: it is the placement the caller asked for that is wrong, not the name.
func TestRegionActivities_RefuseACluster_ThisWorkerDoesNotAdminister(t *testing.T) {
	for name, c := range map[string]struct {
		host string
		port int
	}{
		"another host": {"10.0.0.9", 5432},
		"another port": {"100.84.50.65", 5433},
	} {
		a := regionActs("100.84.50.65", 5432)
		for step, run := range regionSteps(a) {
			err := run(provisioning.RegionDatabaseInput{Region: "eu-west-1", Host: c.host, Port: c.port, DatabaseName: "abc_orm"})
			require.True(t, isNonRetryableOf(err, "TenantDatabaseRegionMismatch"), "%s %s: %v", step, name, err)
			require.ErrorContains(t, err, "no database was created", "%s %s", step, name)
		}
	}
}

func TestSeedTenantDatabase_FailsClosedBeforeConnecting(t *testing.T) {
	ctx := context.Background()
	in := provisioning.TenantDatabaseInput{TenantID: "11111111-2222-3333-4444-555555555555", App: "orm", DatabaseName: "abc_orm"}

	_, err := regionActs("h", 5432).SeedTenantDatabase(ctx, provisioning.TenantDatabaseInput{TenantID: in.TenantID, App: "orm", DatabaseName: `x"; --`})
	require.True(t, isNonRetryableOf(err, "TenantDatabaseInvalidInput"), "%v", err)

	// No control database: there is nowhere to read the reference rows from.
	_, err = regionActs("h", 5432).SeedTenantDatabase(ctx, in)
	require.True(t, isNonRetryableOf(err, "TenantDatabaseNotConfigured"), "%v", err)

	// An app with no seed is not an error.
	rep, err := (&activities.TenantProvisioningActivities{Logger: zap.NewNop().Sugar()}).
		SeedTenantDatabase(ctx, provisioning.TenantDatabaseInput{TenantID: in.TenantID, App: "mdm", DatabaseName: "abc_mdm"})
	require.NoError(t, err)
	require.True(t, rep.Done)
}

// The same steps against a real cluster. Environment (the cluster the saga tests use; only an
// administrator connection is needed, no scratch alpha):
//
//	SAGA_TEST_PG_HOST/PORT/USER  a superuser on a throwaway cluster
//	SAGA_TEST_PG_PASSWORD        its password
func TestRegionActivities_RealCluster_CreateRefuseDropAndForce(t *testing.T) {
	host, user := os.Getenv("SAGA_TEST_PG_HOST"), os.Getenv("SAGA_TEST_PG_USER")
	port, _ := strconv.Atoi(os.Getenv("SAGA_TEST_PG_PORT"))
	password := os.Getenv("SAGA_TEST_PG_PASSWORD")
	if host == "" || user == "" || port == 0 || password == "" {
		t.Skip("SAGA_TEST_PG_* not set")
	}
	ctx := context.Background()
	admin := activities.TenantDatabaseAdmin{Host: host, Port: port, User: user, Password: password}
	acts := &activities.TenantProvisioningActivities{Logger: zap.NewNop().Sugar(), TenantDB: admin}
	name := "regtest_" + strings.ReplaceAll(uuid.NewString()[:8], "-", "")
	in := provisioning.RegionDatabaseInput{Region: "us-east-1", Host: host, Port: port, DatabaseName: name}

	pg, err := admin.Open("postgres")
	require.NoError(t, err)
	defer pg.Close()
	t.Cleanup(func() { _, _ = pg.Exec(fmt.Sprintf(`DROP DATABASE IF EXISTS "%s" WITH (FORCE)`, name)) })
	exists := func() bool {
		var e bool
		require.NoError(t, pg.QueryRow(`SELECT EXISTS (SELECT 1 FROM pg_database WHERE datname = $1)`, name).Scan(&e))
		return e
	}

	// Free name: allowed. Nothing is created by asking.
	require.NoError(t, acts.AssertRegionCluster(ctx, in))
	require.False(t, exists())

	// Created, and created again: a retry after a partial failure finishes the job.
	require.NoError(t, acts.CreateTenantDatabaseInRegion(ctx, in))
	require.NoError(t, acts.CreateTenantDatabaseInRegion(ctx, in))
	require.True(t, exists())

	// Not connectable by a role that has no grant: CONNECT is closed to PUBLIC. A role is made for the
	// check and dropped with it.
	probe := "regprobe_" + name[len("regtest_"):]
	_, err = pg.Exec(fmt.Sprintf(`CREATE ROLE "%s" NOLOGIN`, probe))
	require.NoError(t, err)
	t.Cleanup(func() { _, _ = pg.Exec(fmt.Sprintf(`DROP ROLE IF EXISTS "%s"`, probe)) })
	var canConnect, allowsConn bool
	require.NoError(t, pg.QueryRow(`SELECT has_database_privilege($1, $2, 'CONNECT'), datallowconn FROM pg_database WHERE datname = $2`, probe, name).Scan(&canConnect, &allowsConn))
	require.False(t, canConnect, "a new database must not be connectable by PUBLIC")
	require.True(t, allowsConn, "connections are enabled once the database is closed to PUBLIC")

	// Now the name is taken: a second tenant with the same label is refused, non-retryably.
	err = acts.AssertRegionCluster(ctx, in)
	require.True(t, isNonRetryableOf(err, "TenantDatabaseInvalidInput"), "%v", err)
	require.ErrorContains(t, err, "already exists")

	// Dropped even while something is still connected to it, and dropping again is not an error.
	conn, err := admin.Open(name)
	require.NoError(t, err)
	defer conn.Close()
	require.NoError(t, conn.Ping())
	require.NoError(t, acts.RollbackCreateTenantDatabaseInRegion(ctx, in))
	require.False(t, exists())
	require.NoError(t, acts.RollbackCreateTenantDatabaseInRegion(ctx, in))

	// A cluster that is not this one is refused before any statement runs.
	other := in
	other.Port = port + 1
	require.True(t, isNonRetryableOf(acts.CreateTenantDatabaseInRegion(ctx, other), "TenantDatabaseRegionMismatch"))
	require.False(t, exists())
}
