package activities

import (
	"context"
	"strings"
	"testing"

	"github.com/hondyman/uisce/backend/internal/provisioning"
	"go.uber.org/zap"
)

// Hostile database names must be rejected before any process or connection is
// started. These activities are built without a DB or pg_dump, so a missing
// check would surface as a connection or exec error rather than "safe identifier".
var hostileDatabaseNames = []string{
	"",
	"x; DROP DATABASE alpha;--",
	`x"; DROP DATABASE alpha;--`,
	"x'y",
	"a b",
	"tenant_é",
	"-h",
	"dbname=x host=evil",
	"a\nb",
	"a\x00b",
	"d" + strings.Repeat("x", 63),
}

func newIdentifierTestActivities() *TenantProvisioningActivities {
	return &TenantProvisioningActivities{Logger: zap.NewNop().Sugar()}
}

func TestRollbackCreateTenantDatabaseRejectsHostileNames(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://u:p@127.0.0.1:1/x?sslmode=disable")
	a := newIdentifierTestActivities()
	for _, name := range hostileDatabaseNames {
		err := a.RollbackCreateTenantDatabase(context.Background(), name)
		if err == nil || !strings.Contains(err.Error(), "safe identifier") {
			t.Errorf("RollbackCreateTenantDatabase(%q) err=%v, want safe identifier rejection", name, err)
		}
	}
}

func TestCloneSchemaFromGoldCopyRejectsHostileNames(t *testing.T) {
	a := newIdentifierTestActivities()
	for _, name := range hostileDatabaseNames {
		for _, in := range []provisioning.CloneSchemaInput{
			{SourceDatabase: name, TargetDatabase: "tenant_ok"},
			{SourceDatabase: "alpha", TargetDatabase: name},
		} {
			err := a.CloneSchemaFromGoldCopy(context.Background(), in)
			if err == nil || !strings.Contains(err.Error(), "safe identifier") {
				t.Errorf("CloneSchemaFromGoldCopy(%+v) err=%v, want safe identifier rejection", in, err)
			}
		}
	}
}
