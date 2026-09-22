package analytics

import (
	"context"
	"database/sql/driver"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/hondyman/uisce/backend/internal/models"
	"github.com/jmoiron/sqlx"
)

const (
	goldT   = "99e99e99-99e9-49e9-89e9-99e99e99e999"
	tenantT = "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa"
)

func TestVisibleTenantsAndOrigin(t *testing.T) {
	if got := visibleTenants(tenantT, goldT); len(got) != 2 || got[0] != tenantT || got[1] != goldT {
		t.Errorf("tenant view = %v; want [tenant gold]", got)
	}
	if got := visibleTenants(goldT, goldT); len(got) != 1 {
		t.Errorf("gold-copy view = %v; want just itself", got)
	}
	if got := visibleTenants(tenantT, ""); len(got) != 1 {
		t.Errorf("no gold-copy tenant: %v; want just the tenant", got)
	}
	if originOf(goldT, goldT) != models.ValidationRuleOriginCore || originOf(tenantT, goldT) != models.ValidationRuleOriginCustom {
		t.Error("gold-copy rules are core, tenant rules are custom")
	}
	if originOf(tenantT, "") != models.ValidationRuleOriginCustom {
		t.Error("with no gold-copy tenant nothing is core")
	}
}

func ruleRow(name, tenant string, active bool) []driver.Value {
	return []driver.Value{"00000000-0000-0000-0000-000000000001", name, "", []byte(`{"bo_name":"party","tenant_id":"` + tenant + `","severity":"BLOCK","timing":"pre_write"}`),
		[]byte(`{"rule_ast":{}}`), active, tenant}
}

func TestListByBO_ReturnsCoreAndCustom_CoreWinsANameCollision(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	svc := NewValidationRuleService(sqlx.NewDb(db, "postgres"))

	expectGold(mock, goldT)
	rows := sqlmock.NewRows([]string{"id", "node_name", "description", "properties", "config", "is_active", "tenant_id"}).
		AddRow(ruleRow("mdm.party.required_terms", goldT, true)...).   // core
		AddRow(ruleRow("tenant.party.extra", tenantT, true)...).       // custom
		AddRow(ruleRow("mdm.party.required_terms", tenantT, true)...). // custom, same name as a core rule
		AddRow(ruleRow("a.retired.rule", goldT, false)...)             // core, switched off: still listed
	mock.ExpectQuery(`FROM catalog_node n`).WillReturnRows(rows)

	got, err := svc.ListByBO(context.Background(), tenantT, "party", "")
	if err != nil {
		t.Fatal(err)
	}
	origin := map[string]string{}
	active := map[string]bool{}
	for _, r := range got {
		origin[r.Name] = r.Origin
		active[r.Name] = r.IsActive
	}
	if len(got) != 3 {
		t.Fatalf("got %d rules %v; want 3 (the same-name custom rule is shadowed by the core one)", len(got), origin)
	}
	if origin["mdm.party.required_terms"] != "core" {
		t.Errorf("same-name rule is %q; core must win", origin["mdm.party.required_terms"])
	}
	if origin["tenant.party.extra"] != "custom" || origin["a.retired.rule"] != "core" {
		t.Errorf("origins = %v", origin)
	}
	if active["a.retired.rule"] {
		t.Error("a switched-off core rule must be reported inactive so the evaluator can skip it")
	}
}

func TestListByBO_GoldCopyTenantSeesOnlyItsOwn(t *testing.T) {
	db, mock, _ := sqlmock.New()
	defer db.Close()
	svc := NewValidationRuleService(sqlx.NewDb(db, "postgres"))
	expectGold(mock, goldT)
	mock.ExpectQuery(`FROM catalog_node n`).
		WithArgs(sqlmock.AnyArg(), "party", "", models.ValidationRuleDomainDefault).
		WillReturnRows(sqlmock.NewRows([]string{"id", "node_name", "description", "properties", "config", "is_active", "tenant_id"}).
			AddRow(ruleRow("mdm.party.required_terms", goldT, true)...))
	got, err := svc.ListByBO(context.Background(), goldT, "party", "")
	if err != nil || len(got) != 1 || got[0].Origin != "core" {
		t.Fatalf("got %+v, %v; want the one core rule", got, err)
	}
}

func TestRejectCoreShadow(t *testing.T) {
	newSvc := func(t *testing.T) (*ValidationRuleService, sqlmock.Sqlmock) {
		db, mock, err := sqlmock.New()
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { db.Close() })
		return NewValidationRuleService(sqlx.NewDb(db, "postgres")), mock
	}
	ctx := context.Background()

	t.Run("a tenant cannot reuse a core rule's name", func(t *testing.T) {
		svc, mock := newSvc(t)
		mock.ExpectQuery(`SELECT count`).WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
		err := svc.rejectCoreShadow(ctx, tenantT, goldT, "party", "mdm.party.required_terms")
		if err == nil || !strings.Contains(err.Error(), "core rule") {
			t.Fatalf("err = %v; want a core-rule rejection", err)
		}
	})
	t.Run("an unused name is fine", func(t *testing.T) {
		svc, mock := newSvc(t)
		mock.ExpectQuery(`SELECT count`).WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
		if err := svc.rejectCoreShadow(ctx, tenantT, goldT, "party", "tenant.party.extra"); err != nil {
			t.Fatal(err)
		}
	})
	t.Run("the gold-copy tenant may edit its own rules", func(t *testing.T) {
		svc, mock := newSvc(t)
		if err := svc.rejectCoreShadow(ctx, goldT, goldT, "party", "mdm.party.required_terms"); err != nil {
			t.Fatal(err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Error("the gold-copy tenant must not trigger a lookup:", err)
		}
	})
	t.Run("no gold-copy tenant means no core rules", func(t *testing.T) {
		svc, _ := newSvc(t)
		if err := svc.rejectCoreShadow(ctx, tenantT, "", "party", "x"); err != nil {
			t.Fatal(err)
		}
	})
}
