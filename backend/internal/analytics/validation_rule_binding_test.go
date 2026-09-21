package analytics

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

func TestRuleScopeApplies(t *testing.T) {
	cases := []struct {
		name               string
		scope              []string
		active             string
		wantApplies, wantU bool
	}{
		{"unscoped rule applies to any binding", nil, "b1", true, false},
		{"unscoped rule applies with no known binding", nil, "", true, false},
		{"scoped rule applies to a listed binding", []string{"b1", "b2"}, "b2", true, false},
		{"scoped rule is skipped for another binding", []string{"b1"}, "b2", false, false},
		{"scoped rule with unknown binding is undetermined, not skipped", []string{"b1"}, "", false, true},
	}
	for _, c := range cases {
		applies, undetermined := RuleScopeApplies(c.scope, c.active)
		if applies != c.wantApplies || undetermined != c.wantU {
			t.Errorf("%s: got applies=%v undetermined=%v; want %v %v", c.name, applies, undetermined, c.wantApplies, c.wantU)
		}
	}
}

func TestBindingContextRoundTrip(t *testing.T) {
	if got := BindingFromContext(context.Background()); got != "" {
		t.Errorf("empty context returned %q", got)
	}
	if got := BindingFromContext(WithBinding(context.Background(), "b1")); got != "b1" {
		t.Errorf("got %q; want b1", got)
	}
}

func TestDescriptorCarriesBindingScope(t *testing.T) {
	props := json.RawMessage(`{"bo_name":"party","tenant_id":"t","severity":"BLOCK","timing":"pre_write","binding_ids":["b1","b2"]}`)
	d, err := descriptorFromNode(uuid.New(), "r", "", props, json.RawMessage(`{"rule_ast":{}}`), true)
	if err != nil {
		t.Fatal(err)
	}
	if len(d.BindingIDs) != 2 || d.BindingIDs[0] != "b1" {
		t.Errorf("BindingIDs = %v; want [b1 b2]", d.BindingIDs)
	}
	// A rule written before the field existed has no scope and applies everywhere.
	old := json.RawMessage(`{"bo_name":"party","tenant_id":"t","severity":"WARN","timing":"reconcile"}`)
	d, err = descriptorFromNode(uuid.New(), "r", "", old, json.RawMessage(`{"rule_ast":{}}`), true)
	if err != nil {
		t.Fatal(err)
	}
	if len(d.BindingIDs) != 0 {
		t.Errorf("legacy rule got scope %v; want none", d.BindingIDs)
	}
}

func TestValidateBindingScope(t *testing.T) {
	newSvc := func(t *testing.T) (*ValidationRuleService, sqlmock.Sqlmock) {
		db, mock, err := sqlmock.New()
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { db.Close() })
		return NewValidationRuleService(sqlx.NewDb(db, "postgres")), mock
	}
	ctx := context.Background()

	t.Run("empty scope needs no query", func(t *testing.T) {
		svc, mock := newSvc(t)
		if err := svc.validateBindingScope(ctx, "t", "party", nil); err != nil {
			t.Fatal(err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Error(err)
		}
	})
	t.Run("accepts bindings of the BO", func(t *testing.T) {
		svc, mock := newSvc(t)
		mock.ExpectQuery(`FROM public.business_object_binding b`).
			WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("b1").AddRow("b2"))
		if err := svc.validateBindingScope(ctx, "t", "party", []string{"b1", "b2"}); err != nil {
			t.Fatal(err)
		}
	})
	t.Run("rejects a binding that is not on the BO", func(t *testing.T) {
		svc, mock := newSvc(t)
		mock.ExpectQuery(`FROM public.business_object_binding b`).
			WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("b1"))
		err := svc.validateBindingScope(ctx, "t", "party", []string{"b1", "elsewhere"})
		if err == nil || !strings.Contains(err.Error(), "elsewhere") {
			t.Fatalf("err = %v; want rejection naming the foreign binding", err)
		}
	})
	t.Run("rejects duplicates without querying", func(t *testing.T) {
		svc, _ := newSvc(t)
		if err := svc.validateBindingScope(ctx, "t", "party", []string{"b1", "b1"}); err == nil {
			t.Fatal("duplicate binding id accepted")
		}
	})
}

func TestResolveActiveBinding(t *testing.T) {
	newDB := func(t *testing.T) (*sqlx.DB, sqlmock.Sqlmock) {
		db, mock, err := sqlmock.New()
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { db.Close() })
		return sqlx.NewDb(db, "postgres"), mock
	}
	cols := []string{"id", "driving_path"}

	t.Run("inferred from the driving table", func(t *testing.T) {
		db, mock := newDB(t)
		mock.ExpectQuery(`n.qualified_path = \$3`).WithArgs("t", "bo", "/mdm/party").
			WillReturnRows(sqlmock.NewRows(cols).AddRow("b1", "/mdm/party"))
		ab, err := ResolveActiveBinding(context.Background(), db, "t", "bo", "/mdm/party")
		if err != nil || ab == nil || ab.ID != "b1" {
			t.Fatalf("got %+v, %v; want b1", ab, err)
		}
	})
	t.Run("no matching binding is (nil, nil), not an error", func(t *testing.T) {
		db, mock := newDB(t)
		mock.ExpectQuery(`n.qualified_path = \$3`).WillReturnRows(sqlmock.NewRows(cols))
		ab, err := ResolveActiveBinding(context.Background(), db, "t", "bo", "/mdm/party")
		if err != nil || ab != nil {
			t.Fatalf("got %+v, %v; want nil, nil", ab, err)
		}
	})
	t.Run("explicit binding wins and must belong to the BO", func(t *testing.T) {
		db, mock := newDB(t)
		mock.ExpectQuery(`b.id = \$3::uuid`).WithArgs("t", "bo", "b9").
			WillReturnRows(sqlmock.NewRows(cols).AddRow("b9", "/alpha/oms/orders"))
		ab, err := ResolveActiveBinding(WithBinding(context.Background(), "b9"), db, "t", "bo", "/mdm/party")
		if err != nil || ab == nil || ab.DrivingPath != "/alpha/oms/orders" {
			t.Fatalf("got %+v, %v", ab, err)
		}
		db2, mock2 := newDB(t)
		mock2.ExpectQuery(`b.id = \$3::uuid`).WillReturnRows(sqlmock.NewRows(cols))
		if _, err := ResolveActiveBinding(WithBinding(context.Background(), "nope"), db2, "t", "bo", "/mdm/party"); err == nil {
			t.Fatal("explicit binding that is not on the BO was accepted")
		}
	})
}
