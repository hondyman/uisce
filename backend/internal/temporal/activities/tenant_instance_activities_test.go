package activities

import (
	"context"
	"strings"
	"testing"

	"github.com/hondyman/uisce/backend/internal/provisioning"
	"go.uber.org/zap"
)

// Hostile names must be rejected before any connection is attempted. The
// activities are built without a DB, and DATABASE_URL points nowhere, so a
// missing validation would surface as a connection error, not "invalid".
func hostileNames() []string {
	return []string{
		"",
		"x; DROP DATABASE alpha;--",
		`x"; DROP DATABASE alpha;--`,
		"x'y",
		"a b",
		"tenant_é",
		"-h",
		"dbname=x host=evil",
		"a" + strings.Repeat("b", 80),
	}
}

func newTestActivities() *TenantProvisioningActivities {
	return &TenantProvisioningActivities{Logger: zap.NewNop().Sugar()}
}

func TestCreateTenantDatabaseRejectsHostileNames(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://u:p@127.0.0.1:1/x?sslmode=disable")
	a := newTestActivities()
	for _, n := range hostileNames() {
		err := a.CreateTenantDatabase(context.Background(), n)
		if err == nil || !strings.Contains(err.Error(), "invalid database name") {
			t.Errorf("CreateTenantDatabase(%q) err=%v, want invalid database name", n, err)
		}
	}
}

func TestRollbackCreateTenantDatabaseRejectsHostileNames(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://u:p@127.0.0.1:1/x?sslmode=disable")
	a := newTestActivities()
	for _, n := range hostileNames() {
		err := a.RollbackCreateTenantDatabase(context.Background(), n)
		if err == nil || !strings.Contains(err.Error(), "invalid database name") {
			t.Errorf("RollbackCreateTenantDatabase(%q) err=%v, want invalid database name", n, err)
		}
	}
}

func TestCloneSchemaFromGoldCopyRejectsHostileNames(t *testing.T) {
	a := newTestActivities()
	for _, n := range hostileNames() {
		for _, in := range []provisioning.CloneSchemaInput{
			{SourceDatabase: n, TargetDatabase: "tenant_ok"},
			{SourceDatabase: "alpha", TargetDatabase: n},
		} {
			err := a.CloneSchemaFromGoldCopy(context.Background(), in)
			if err == nil || !strings.Contains(err.Error(), "invalid database name") {
				t.Errorf("CloneSchemaFromGoldCopy(%+v) err=%v, want invalid database name", in, err)
			}
		}
	}
}
