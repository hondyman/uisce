// Command verify_mdm_e2e checks, against the real database and as a regular tenant, that the MDM
// rule chain works end to end:
//
//	gold-copy rules are visible to a tenant (code + catalog_node RLS policy)
//	no other gold-copy catalog data leaks (the policy is narrow)
//	every rule's semantic terms resolve to columns (a term that does not is a rule_error at write time)
//	the stored rule ASTs behave exactly like the tested catalog (accept/reject examples)
//	a binding-scoped rule can find the inherited gold-copy binding (business_object_binding policy)
//	core rules cannot be shadowed; with -write, a custom rule is created, seen only by its tenant, cleaned up
//
// It connects with DATABASE_URL and sets the tenant context for the session, so the RLS checks are only
// meaningful if that role is NOT a superuser and does not bypass RLS; the command says so if it is.
//
//	DATABASE_URL=... go run ./cmd/verify_mdm_e2e [-tenant <uuid>] [-write]
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/hondyman/uisce/backend/internal/analytics"
	"github.com/hondyman/uisce/backend/internal/mdmrules"
	"github.com/hondyman/uisce/backend/internal/models"
	vm "github.com/hondyman/uisce/backend/internal/rules/vm"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

var failed, passed, skipped int

func ok(format string, a ...any)   { passed++; fmt.Printf("  PASS  "+format+"\n", a...) }
func bad(format string, a ...any)  { failed++; fmt.Printf("  FAIL  "+format+"\n", a...) }
func skip(format string, a ...any) { skipped++; fmt.Printf("  SKIP  "+format+"\n", a...) }
func note(format string, a ...any) { fmt.Printf("  NOTE  "+format+"\n", a...) }
func head(s string)                { fmt.Printf("\n== %s\n", s) }

func main() {
	tenantFlag := flag.String("tenant", "", "regular tenant to act as (default: the first non-gold-copy tenant)")
	otherFlag := flag.String("other", "", "a second regular tenant, to check custom-rule isolation (public.tenants shows a tenant only its own row under RLS)")
	write := flag.Bool("write", false, "also create a temporary custom rule as the tenant (and remove it)")
	flag.Parse()
	ctx := context.Background()

	db, err := sqlx.Connect("postgres", os.Getenv("DATABASE_URL"))
	if err != nil {
		fmt.Println("connect:", err)
		os.Exit(2)
	}
	defer db.Close()
	db.SetMaxOpenConns(1) // the tenant GUC is per-connection

	var gold string
	tenant := *tenantFlag
	if tenant == "" {
		if err := db.GetContext(ctx, &tenant, `SELECT id::text FROM public.tenants WHERE gold_copy IS NOT TRUE ORDER BY id LIMIT 1`); err != nil {
			fmt.Println("no regular tenant to act as; pass -tenant:", err)
			os.Exit(2)
		}
	}
	if _, err := db.ExecContext(ctx, `SELECT set_config('uisce.current_tenant', $1, false), set_config('app.current_tenant', $1, false), set_config('hasura.tenant_id', $1, false)`, tenant); err != nil {
		fmt.Println("set tenant context:", err)
		os.Exit(2)
	}
	// public.tenants is under RLS (a tenant sees only its own row); the SECURITY DEFINER function returns just the gold id.
	if err := db.GetContext(ctx, &gold, `SELECT COALESCE(public.uisce_gold_copy_tenant_id()::text, '')`); err != nil || gold == "" {
		gold = "99e99e99-99e9-49e9-89e9-99e99e99e999"
		fmt.Printf("FINDING: uisce_gold_copy_tenant_id() gave no gold-copy tenant (%v); assuming %s (migration 20261024_006 not applied?)\n", err, gold)
	}
	if tenant == gold {
		fmt.Println("-tenant must not be the gold-copy tenant")
		os.Exit(2)
	}
	// The backend applies uisce.gold_tenant for every tenant transaction (db.WithTenantGoldTransaction); the
	// gold-aware policies on business_object_fields and catalog_edge read it. Mirror that here.
	if _, err := db.ExecContext(ctx, `SELECT set_config('uisce.gold_tenant', $1, false)`, gold); err != nil {
		fmt.Println("set gold tenant context:", err)
		os.Exit(2)
	}
	fmt.Printf("acting as regular tenant %s (gold-copy tenant %s)\n", tenant, gold)

	head("0. role")
	var role struct {
		Name    string `db:"rolname"`
		Super   bool   `db:"rolsuper"`
		Bypass  bool   `db:"rolbypassrls"`
		TypeSel bool   `db:"typesel"`
	}
	if err := db.GetContext(ctx, &role, `SELECT rolname, rolsuper, rolbypassrls, has_table_privilege(current_user, 'public.catalog_node_type', 'SELECT') AS typesel FROM pg_roles WHERE rolname = current_user`); err != nil {
		bad("read role: %v", err)
	}
	rlsMeaningful := !role.Super && !role.Bypass
	if rlsMeaningful {
		ok("connected as %q, which is subject to RLS", role.Name)
	} else {
		skip("connected as %q (superuser=%v, bypassrls=%v): RLS is NOT enforced for it, so the visibility/leak checks below prove nothing about policies. Use the application's role.", role.Name, role.Super, role.Bypass)
	}
	if role.TypeSel {
		ok("role can SELECT catalog_node_type (the gold-copy rule policy reads it as the caller)")
	} else {
		bad("role cannot SELECT catalog_node_type: with the catalog_node_read_gold_copy_rules policy in place EVERY catalog_node read would fail")
	}

	svc := analytics.NewValidationRuleService(db)
	catalog := mdmrules.Catalog()
	byBO := map[string][]mdmrules.Rule{}
	for _, r := range catalog {
		byBO[r.BO] = append(byBO[r.BO], r)
	}

	head("1. the tenant sees the gold-copy rules, marked core")
	missing, wrongOrigin := 0, 0
	listed := map[string]models.ValidationRuleDescriptor{}
	for bo := range byBO {
		list, err := svc.ListByBO(ctx, tenant, bo, "")
		if err != nil {
			bad("list rules for %s: %v", bo, err)
			continue
		}
		for _, d := range list {
			listed[d.Name] = d
		}
	}
	for _, r := range catalog {
		d, found := listed[r.Name]
		switch {
		case !found:
			missing++
		case d.Origin != models.ValidationRuleOriginCore:
			wrongOrigin++
		}
	}
	switch {
	case missing == 0 && wrongOrigin == 0:
		ok("all %d catalog rules are visible to the tenant and marked core", len(catalog))
	case missing == len(catalog):
		bad("the tenant sees none of the %d core rules. Either they are not seeded (run ./cmd/seed_mdm_rules -apply) or the catalog_node policy does not let this role read them", len(catalog))
	default:
		bad("%d of %d core rules are not visible to the tenant, %d visible but not marked core", missing, len(catalog), wrongOrigin)
	}

	head("2. what else of the gold-copy tenant can the tenant read")
	goldReadable := 0
	if !rlsMeaningful {
		skip("needs a role subject to RLS")
	} else {
		var structure int
		if err := db.GetContext(ctx, &goldReadable, `
			SELECT count(*) FROM public.catalog_node n JOIN public.catalog_node_type nt ON nt.id = n.node_type_id
			WHERE n.tenant_id = $1::uuid AND nt.catalog_type_name NOT IN ('validation_rule', 'table', 'column')`, gold); err != nil {
			bad("query: %v", err)
		} else if err := db.GetContext(ctx, &structure, `
			SELECT count(*) FROM public.catalog_node n JOIN public.catalog_node_type nt ON nt.id = n.node_type_id
			WHERE n.tenant_id = $1::uuid AND nt.catalog_type_name IN ('table', 'column')`, gold); err != nil {
			bad("query: %v", err)
		} else if goldReadable == 0 {
			ok("the tenant reads only validation rules and table/column structure (%d nodes) from the gold-copy tenant (policies catalog_node_read_gold_copy_rules and _structure)", structure)
		} else {
			bad("the tenant can read %d gold-copy catalog nodes beyond validation rules and table/column structure (terms, business terms, API nodes...): some policy is wider than intended", goldReadable)
		}
	}

	head("3. every rule's semantic terms resolve to a column (an unresolved term is a rule_error at write time)")
	unresolved, unresolvedBOs := 0, 0
	for bo, rules := range byBO {
		var b struct {
			ID     string `db:"id"`
			Driver string `db:"driver_table_name"`
		}
		if err := db.GetContext(ctx, &b, `SELECT id::text AS id, COALESCE(driver_table_name,'') AS driver_table_name FROM public.business_objects WHERE tenant_id = $1::uuid AND bo_key = $2`, gold, bo); err != nil {
			bad("BO %s: not found in the gold-copy tenant (%v)", bo, err)
			unresolved++
			continue
		}
		fields, err := analytics.ResolveSemanticFieldMap(ctx, db, b.ID, b.Driver)
		if err != nil {
			bad("BO %s: resolving terms failed: %v", bo, err)
			unresolved++
			continue
		}
		var lost []string
		for _, r := range rules {
			for _, term := range termsOf(r.AST) {
				if _, has := fields[term]; !has {
					lost = append(lost, term)
				}
			}
		}
		if len(lost) > 0 {
			unresolved += len(lost)
			unresolvedBOs++
			show := lost
			if len(show) > 4 {
				show = show[:4]
			}
			bad("%s (%d of its %d mapped terms are visible): %d rule term reference(s) do not resolve, e.g. %s", bo, len(fields), len(fields)+len(lost), len(lost), strings.Join(show, ", "))
		}
	}
	if unresolved == 0 {
		ok("all rule terms resolve to columns")
	} else if unresolvedBOs > 3 {
		fmt.Printf("        -> %d BOs fail together, which points at visibility rather than at individual mappings: a regular tenant is probably unable to read the gold-copy column nodes / MAPS_TO targets it needs to resolve inherited terms (the evaluation path resolves them as the writing tenant). If only one or two BOs fail, the mapping for those terms is missing.\n", unresolvedBOs)
	}

	head("4. the stored rules behave like the tested catalog")
	ev := vm.NewAdvancedEvaluator()
	drift := 0
	for _, r := range catalog {
		d, found := listed[r.Name]
		if !found {
			continue
		}
		var node vm.RuleNode
		if err := json.Unmarshal(d.RuleAST, &node); err != nil {
			bad("%s: stored rule_ast does not parse: %v", r.Name, err)
			drift++
			continue
		}
		for _, c := range r.Cases {
			got, err := ev.Evaluate(node, c.Record)
			if err != nil || got != c.Pass {
				bad("%s / %s: stored rule gives (%v, %v), want %v", r.Name, c.Name, got, err, c.Pass)
				drift++
			}
		}
		if !d.IsActive {
			bad("%s is switched off (inactive rules are skipped by the evaluator)", r.Name)
			drift++
		}
	}
	if drift == 0 {
		ok("every stored rule accepts and rejects its examples and is active")
	}

	head("5. a binding-scoped rule finds the inherited gold-copy binding")
	var issuerBO, driver string
	if err := db.GetContext(ctx, &issuerBO, `SELECT id::text FROM public.business_objects WHERE tenant_id = $1::uuid AND bo_key = 'issuer'`, gold); err != nil {
		skip("issuer BO not found: %v", err)
	} else {
		_ = db.GetContext(ctx, &driver, `SELECT COALESCE(driver_table_name,'') FROM public.business_objects WHERE id = $1::uuid`, issuerBO)
		ab, err := analytics.ResolveActiveBinding(ctx, db, tenant, issuerBO, driver)
		switch {
		case err != nil:
			bad("resolve active binding: %v", err)
		case ab == nil:
			bad("no active binding found for the inherited issuer BO (%s). Scoped rules would be reported as rule_errors. Either the binding is missing or the business_object_binding gold-copy policy is not letting this role read it", driver)
		default:
			ok("issuer resolves to binding %s on %s", ab.ID, ab.DrivingPath)
			d := listed["mdm.issuer.sourced_from_a_system"]
			applies, _ := analytics.RuleScopeApplies(d.BindingIDs, ab.ID)
			if applies {
				ok("the scoped rule mdm.issuer.sourced_from_a_system applies to that binding")
			} else {
				bad("the scoped rule's binding_ids %v do not include the resolved binding %s", d.BindingIDs, ab.ID)
			}
		}
	}

	head("6. core rules cannot be shadowed by a tenant")
	if len(catalog) > 0 {
		r := catalog[0]
		_, err := svc.UpsertValidationRule(ctx, models.UpsertValidationRuleRequest{TenantID: tenant, BOName: r.BO, Name: r.Name, Severity: "WARN", Timing: "pre_write", Domain: "mdm",
			RuleAST: json.RawMessage(`{"type":"condition","field":"TenantId","fieldPath":"TenantId","operator":"is_not_null","valueType":""}`)})
		if err != nil && strings.Contains(err.Error(), "core rule") {
			ok("a tenant rule reusing the core name %q is rejected", r.Name)
		} else if err == nil {
			bad("a tenant rule reusing the core name %q was ACCEPTED, which means the tenant cannot see the core rule to compare against (see step 1)", r.Name)
			// undo what this check just wrote, so a failing run leaves nothing behind
			db.ExecContext(ctx, `DELETE FROM public.catalog_edge WHERE target_node_id IN (SELECT id FROM public.catalog_node WHERE tenant_id = $1::uuid AND node_name = $2)`, tenant, r.Name)
			db.ExecContext(ctx, `DELETE FROM public.catalog_node WHERE tenant_id = $1::uuid AND node_name = $2`, tenant, r.Name)
			fmt.Println("        (the rule this check created has been removed again)")
		} else {
			bad("unexpected error shadowing a core rule: %v", err)
		}
	}

	head("7. tenant custom rules (needs -write)")
	if !*write {
		skip("re-run with -write to create a temporary custom rule as the tenant, check isolation, and remove it")
	} else {
		name := "verify.mdm.e2e.temporary_custom_rule"
		_, err := svc.UpsertValidationRule(ctx, models.UpsertValidationRuleRequest{TenantID: tenant, BOName: "party", Name: name, Description: "temporary rule from verify_mdm_e2e", Severity: "WARN", Timing: "pre_write", Domain: "mdm",
			RuleAST: json.RawMessage(`{"type":"condition","field":"Segment","fieldPath":"Segment","operator":"is_not_null","valueType":""}`)})
		if err != nil {
			bad("create a custom rule on the inherited party BO: %v", err)
		} else {
			defer func() {
				if _, err := db.ExecContext(ctx, `DELETE FROM public.catalog_edge WHERE target_node_id IN (SELECT id FROM public.catalog_node WHERE tenant_id = $1::uuid AND node_name = $2)`, tenant, name); err != nil {
					fmt.Println("cleanup edge:", err)
				}
				if _, err := db.ExecContext(ctx, `DELETE FROM public.catalog_node WHERE tenant_id = $1::uuid AND node_name = $2`, tenant, name); err != nil {
					fmt.Println("cleanup node:", err)
				} else {
					fmt.Println("\n(temporary custom rule removed)")
				}
			}()
			list, _ := svc.ListByBO(ctx, tenant, "party", "")
			var core, custom int
			for _, d := range list {
				switch d.Origin {
				case models.ValidationRuleOriginCore:
					core++
				default:
					if d.Name == name {
						custom++
					}
				}
			}
			if core > 0 && custom == 1 {
				ok("the tenant sees %d core rule(s) and its own custom rule together on party", core)
			} else {
				bad("tenant view of party rules: core=%d, own custom=%d", core, custom)
			}
			other := *otherFlag
			var lookupErr error
			if other == "" {
				lookupErr = db.GetContext(ctx, &other, `SELECT id::text FROM public.tenants WHERE gold_copy IS NOT TRUE AND id <> $1::uuid ORDER BY id LIMIT 1`, tenant)
			}
			if lookupErr != nil || other == "" {
				skip("no second regular tenant to check isolation against (pass -other; public.tenants shows a tenant only its own row under RLS)")
			} else if rlsMeaningful {
				if _, err := db.ExecContext(ctx, `SELECT set_config('uisce.current_tenant', $1, false), set_config('app.current_tenant', $1, false), set_config('hasura.tenant_id', $1, false)`, other); err == nil {
					l2, _ := svc.ListByBO(ctx, other, "party", "")
					leaked := false
					for _, d := range l2 {
						if d.Name == name {
							leaked = true
						}
					}
					if leaked {
						bad("another tenant (%s) can see this tenant's custom rule", other)
					} else {
						ok("another tenant cannot see this tenant's custom rule")
					}
					db.ExecContext(ctx, `SELECT set_config('uisce.current_tenant', $1, false), set_config('app.current_tenant', $1, false), set_config('hasura.tenant_id', $1, false)`, tenant)
				}
			} else {
				skip("isolation between tenants needs a role subject to RLS")
			}
		}
	}

	fmt.Printf("\n%d passed, %d failed, %d skipped\n", passed, failed, skipped)
	if failed > 0 {
		os.Exit(1)
	}
}

// termsOf lists the semantic terms a rule references.
func termsOf(n vm.RuleNode) []string {
	seen := map[string]bool{}
	var out []string
	add := func(t string) {
		if !seen[t] {
			seen[t] = true
			out = append(out, t)
		}
	}
	var walkExpr func(e vm.ExprNode)
	walkExpr = func(e vm.ExprNode) {
		switch x := e.(type) {
		case *vm.BinaryExpr:
			walkExpr(x.Left)
			walkExpr(x.Right)
		case *vm.FieldRef:
			add(x.Path)
		case *vm.FuncCall:
			for _, a := range x.Args {
				walkExpr(a)
			}
		}
	}
	var walk func(n vm.RuleNode)
	walk = func(n vm.RuleNode) {
		switch n.Type {
		case vm.NodeTypeGroup:
			for _, c := range n.Group.Conditions {
				walk(c)
			}
		case vm.NodeTypeCondition:
			add(n.Condition.Field)
		case vm.NodeTypeExpression:
			walkExpr(n.Expression.Root)
		}
	}
	walk(n)
	return out
}
