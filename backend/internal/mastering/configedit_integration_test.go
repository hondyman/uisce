//go:build integration

package mastering

import (
	"context"
	"os"
	"testing"

	"github.com/hondyman/uisce/backend/internal/analytics"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

// TestConfigMakerChecker runs a tenant's override of the gold copy's source
// hierarchy through maker-checker on a real data plane: proposed (nothing
// changes), refused for the proposer, approved by a second admin (applied),
// seen by the engine, then removed the same way. Leaves nothing behind.
func TestConfigMakerChecker(t *testing.T) {
	alphaDSN, dataDSN := os.Getenv("MASTERING_ALPHA_DSN"), os.Getenv("MASTERING_DATA_DSN")
	if alphaDSN == "" || dataDSN == "" {
		t.Skip("set MASTERING_ALPHA_DSN and MASTERING_DATA_DSN")
	}
	alpha, data := sqlx.MustConnect("postgres", alphaDSN), sqlx.MustConnect("postgres", dataDSN)
	defer alpha.Close()
	defer data.Close()
	platform := PlatformCatalog{DB: alpha}
	e := &Engine{Data: data, Rules: analytics.NewValidationRuleService(alpha), Fields: platform, GoldCopy: platform.GoldCopyTenant}
	ctx := context.Background()
	const tenant = "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa"
	maker := Actor{TenantID: tenant, UserID: "config-test-maker", Name: "Maker", Admin: true}
	checker := Actor{TenantID: tenant, UserID: "config-test-checker", Name: "Checker", Admin: true}
	defer func() {
		_, _ = data.ExecContext(ctx, `DELETE FROM mdm.product_source_priority WHERE tenant_id = $1::uuid`, tenant)
		_, _ = data.ExecContext(ctx, `DELETE FROM mdm.mastering_config_change WHERE tenant_id = $1::uuid AND requested_by = $2`, tenant, maker.UserID)
	}()

	table, err := e.ConfigTable(ctx, tenant, KindSourcePriority, "product")
	if err != nil {
		t.Fatal(err)
	}
	var goldName *ConfigRow
	for i, r := range table.Rows {
		if r.Values["field_group"] == "NAME" && r.Values["source"] == "FACTSET" {
			goldName = &table.Rows[i]
		}
	}
	if goldName == nil || !goldName.Inherited || goldName.Origin != "core" {
		t.Fatalf("expected the gold copy's NAME/FACTSET row, inherited: %+v", table.Rows)
	}

	// A tenant cannot change the gold copy's row...
	if _, err := e.ProposeConfig(ctx, maker, ConfigProposal{Kind: KindSourcePriority, Entity: "product", TargetID: goldName.ID,
		Values: map[string]any{"priority": 1}}); err == nil {
		t.Fatal("changing a gold-copy row must be refused")
	}
	// ...it proposes its own row with the same key: FactSet first for names.
	ch, err := e.ProposeConfig(ctx, maker, ConfigProposal{Kind: KindSourcePriority, Entity: "product",
		Values: map[string]any{"field_group": "NAME", "source": "factset", "priority": 1}, Reason: "FactSet names are cleaner"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := e.DecideConfig(ctx, maker, ch.ID, "approve", ""); err == nil {
		t.Fatal("the proposer must not approve their own change")
	}
	cfg, _ := e.loadConfig(ctx, tenant, "product")
	if cfg.hierarchy["NAME"][0] == "FACTSET" {
		t.Fatal("nothing may change before approval")
	}
	if _, err := e.DecideConfig(ctx, checker, ch.ID, "approve", "ok"); err != nil {
		t.Fatal(err)
	}
	cfg, err = e.loadConfig(ctx, tenant, "product")
	if err != nil {
		t.Fatal(err)
	}
	if got := cfg.hierarchy["NAME"]; len(got) == 0 || got[0] != "FACTSET" {
		t.Fatalf("after approval the engine must rank FactSet first for NAME: %v", got)
	}
	table, _ = e.ConfigTable(ctx, tenant, KindSourcePriority, "product")
	var own *ConfigRow
	for i, r := range table.Rows {
		if r.ID == goldName.ID && !r.Overridden {
			t.Fatal("the gold row must show as overridden")
		}
		if r.Origin == "tenant" {
			own = &table.Rows[i]
		}
	}
	if own == nil {
		t.Fatal("the tenant's own row is missing")
	}

	// Removing it goes through approval too.
	del, err := e.ProposeConfig(ctx, maker, ConfigProposal{Kind: KindSourcePriority, Entity: "product", Action: "delete", TargetID: own.ID})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := e.DecideConfig(ctx, checker, del.ID, "approve", ""); err != nil {
		t.Fatal(err)
	}
	cfg, _ = e.loadConfig(ctx, tenant, "product")
	if cfg.hierarchy["NAME"][0] == "FACTSET" {
		t.Fatalf("after removal the gold copy's order applies again: %v", cfg.hierarchy["NAME"])
	}
	changes, err := e.ConfigChanges(ctx, maker, KindSourcePriority, "product", "")
	if err != nil || len(changes) < 2 || changes[0].Status != "applied" || !changes[0].Mine {
		t.Fatalf("history: %+v %v", changes, err)
	}
}

// TestConfigTablesRead: every table the configuration pages show reads with
// its own columns - product and security match rules, security and price
// hierarchies (scoped differently), and the vendor registry.
func TestConfigTablesRead(t *testing.T) {
	alphaDSN, dataDSN := os.Getenv("MASTERING_ALPHA_DSN"), os.Getenv("MASTERING_DATA_DSN")
	if alphaDSN == "" || dataDSN == "" {
		t.Skip("set MASTERING_ALPHA_DSN and MASTERING_DATA_DSN")
	}
	alpha, data := sqlx.MustConnect("postgres", alphaDSN), sqlx.MustConnect("postgres", dataDSN)
	defer alpha.Close()
	defer data.Close()
	platform := PlatformCatalog{DB: alpha}
	e := &Engine{Data: data, Fields: platform, GoldCopy: platform.GoldCopyTenant}
	ctx := context.Background()
	const tenant = "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa"
	for _, c := range []struct{ kind, entity, mustHave string }{
		{KindMatchRule, "product", "deterministic_keys"},
		{KindMatchRule, "security", "threshold_auto_match"},
		{KindSourcePriority, "security", "asset_class_cd"},
		{KindSourcePriority, "price", "price_type_cd"},
		{KindSourceSystem, "", "display_name"},
	} {
		tb, err := e.ConfigTable(ctx, tenant, c.kind, c.entity)
		if err != nil {
			t.Fatalf("%s %s: %v", c.kind, c.entity, err)
		}
		has := false
		for _, col := range tb.Columns {
			has = has || col.Name == c.mustHave
		}
		if !has || len(tb.Rows) == 0 || !tb.Rows[0].Inherited {
			t.Fatalf("%s %s: columns %+v rows %d", c.kind, c.entity, tb.Columns, len(tb.Rows))
		}
		t.Logf("%-15s %-8s %2d rows, %d columns", c.kind, c.entity, len(tb.Rows), len(tb.Columns))
	}
	if _, err := e.ConfigTable(ctx, tenant, KindMatchRule, "price"); err == nil {
		t.Fatal("match rules do not apply to a time-series entity")
	}
}

// A match rule the engine could not read is refused when proposed, before
// anything is stored.
func TestConfigRejectsUnreadableMatchRule(t *testing.T) {
	alphaDSN, dataDSN := os.Getenv("MASTERING_ALPHA_DSN"), os.Getenv("MASTERING_DATA_DSN")
	if alphaDSN == "" || dataDSN == "" {
		t.Skip("set MASTERING_ALPHA_DSN and MASTERING_DATA_DSN")
	}
	alpha, data := sqlx.MustConnect("postgres", alphaDSN), sqlx.MustConnect("postgres", dataDSN)
	defer alpha.Close()
	defer data.Close()
	platform := PlatformCatalog{DB: alpha}
	e := &Engine{Data: data, Fields: platform, GoldCopy: platform.GoldCopyTenant}
	a := Actor{TenantID: "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa", UserID: "config-test-maker", Admin: true}
	_, err := e.ProposeConfig(context.Background(), a, ConfigProposal{Kind: KindMatchRule, Entity: "product", Values: map[string]any{
		"rule_cd": "BAD", "rule_name": "Bad", "product_type_cd": "ALL", "match_keys": []string{"name"},
		"fuzzy_keys": []map[string]any{{"field": "name; drop table x", "method": "trigram", "weight": 1}},
		"threshold_auto_match": 0.9, "threshold_review": 0.7,
	}})
	if err == nil {
		t.Fatal("an unreadable fuzzy key must be refused")
	}
}
