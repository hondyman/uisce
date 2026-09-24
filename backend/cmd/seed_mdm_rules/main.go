// Command seed_mdm_rules writes the tier 1/2 MDM validation rules (internal/mdmrules) into the gold-copy
// tenant, through analytics.ValidationRuleService.UpsertValidationRule so binding-scope validation and
// duplicate detection apply.
//
// Dry run by default: it prints what it would write and skips nothing silently. Pass -apply to write.
// Idempotent: a rule is keyed by name, so re-running updates in place.
//
//	DATABASE_URL=... go run ./cmd/seed_mdm_rules            # plan
//	DATABASE_URL=... go run ./cmd/seed_mdm_rules -apply     # write
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/hondyman/uisce/backend/internal/analytics"
	"github.com/hondyman/uisce/backend/internal/mdmrules"
	"github.com/hondyman/uisce/backend/internal/models"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

func main() {
	apply := flag.Bool("apply", false, "write the rules (default: dry run)")
	flag.Parse()

	db, err := sqlx.Connect("postgres", os.Getenv("DATABASE_URL"))
	if err != nil {
		log.Fatalf("connect: %v", err)
	}
	defer db.Close()
	db.SetMaxOpenConns(1) // the tenant GUC below is per-connection
	ctx := context.Background()

	var tenant string
	if err := db.GetContext(ctx, &tenant, `SELECT id::text FROM public.tenants WHERE gold_copy = true LIMIT 1`); err != nil {
		log.Fatalf("no gold-copy tenant: %v", err)
	}
	// Rules are authored in the gold-copy tenant, like the BOs; regular tenants inherit read-only.
	if _, err := db.ExecContext(ctx, `SELECT set_config('app.current_tenant', $1, false), set_config('uisce.current_tenant', $1, false)`, tenant); err != nil {
		log.Fatalf("set tenant context: %v", err)
	}
	svc := analytics.NewValidationRuleService(db)

	var wrote, skipped, failed int
	for _, r := range mdmrules.Catalog() {
		var boID string
		if err := db.GetContext(ctx, &boID, `SELECT id::text FROM public.business_objects WHERE tenant_id = $1::uuid AND bo_key = $2`, tenant, r.BO); err != nil {
			fmt.Printf("SKIP  %-58s BO %q not found in the gold-copy tenant\n", r.Name, r.BO)
			skipped++
			continue
		}
		var bindings []string
		scope := "all bindings"
		if r.Scope == mdmrules.ScopeCurrentBindings {
			if err := db.SelectContext(ctx, &bindings, `SELECT bo_binding_id::text FROM public.business_object_binding WHERE tenant_id = $1::uuid AND bo_id = $2::uuid AND is_active ORDER BY bo_binding_id`, tenant, boID); err != nil || len(bindings) == 0 {
				fmt.Printf("SKIP  %-58s scoped rule but %q has no active binding\n", r.Name, r.BO)
				skipped++
				continue
			}
			scope = fmt.Sprintf("%d binding(s)", len(bindings))
		}
		ast, err := json.Marshal(r.AST)
		if err != nil {
			log.Fatalf("%s: marshal: %v", r.Name, err)
		}
		verb := "PLAN "
		if *apply {
			verb = "WRITE"
			if _, err := svc.UpsertValidationRule(ctx, models.UpsertValidationRuleRequest{
				TenantID: tenant, BOName: r.BO, Name: r.Name, Description: r.Description,
				Severity: r.Severity, Timing: r.Timing, Category: r.Category,
				Domain: models.ValidationRuleDomainMDM, RuleAST: ast, BindingIDs: bindings,
			}); err != nil {
				fmt.Printf("FAIL  %-58s %v\n", r.Name, err)
				failed++
				continue
			}
		}
		fmt.Printf("%s %-58s %-5s %s\n", verb, r.Name, r.Severity, scope)
		wrote++
	}
	fmt.Printf("\n%d %s, %d skipped, %d failed (tenant %s)\n", wrote, map[bool]string{true: "written", false: "planned"}[*apply], skipped, failed, tenant)
	if failed > 0 {
		os.Exit(1)
	}
}
