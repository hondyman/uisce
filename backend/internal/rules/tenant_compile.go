package rules

import (
	"context"
	"fmt"

	"go.starlark.net/starlark"
)

type TenantCompiler struct {
	coreCompiler *CoreCompiler
	tenantCache  *TenantFnCache
	predeclared  starlark.StringDict // same helpers as core
	coreRepo     CoreRuleRepository
}

func NewTenantCompiler(
	coreCompiler *CoreCompiler,
	tenantCache *TenantFnCache,
	predecl starlark.StringDict,
	coreRepo CoreRuleRepository,
) *TenantCompiler {
	return &TenantCompiler{
		coreCompiler: coreCompiler,
		tenantCache:  tenantCache,
		predeclared:  predecl,
		coreRepo:     coreRepo,
	}
}

// BuildCombinedModule returns a module with ok(ctx) for EXTEND.
func BuildCombinedModule(core CoreValidationRule, tenant TenantValidationRule) string {
	return fmt.Sprintf(`
%s

def tenant_ok(ctx):
%s

def ok(ctx):
    return core_ok(ctx) and tenant_ok(ctx)
`, core.ConditionSrc, tenant.ConditionSrc)
}

// BuildRawModule returns a module with ok(ctx) for pure/custom rules.
func BuildRawModule(tenant TenantValidationRule) string {
	return fmt.Sprintf(`
def ok(ctx):
%s
`, tenant.ConditionSrc)
}

func (tc *TenantCompiler) CompileTenantFn(
	tenantRule TenantValidationRule,
) (*starlark.Function, error) {
	coreVer := 0
	if tenantRule.CreatedFromVers != nil {
		coreVer = *tenantRule.CreatedFromVers
	}
	key := TenantFnKey{
		TenantID: tenantRule.TenantID,
		RuleID:   tenantRule.RuleID,
		CoreVer:  coreVer,
	}
	if fn, ok := tc.tenantCache.Get(key); ok {
		return fn, nil
	}

	var moduleSrc string
	var moduleName string

	switch tenantRule.InheritMode {
	case Custom:
		moduleSrc = BuildRawModule(tenantRule)
		moduleName = fmt.Sprintf("tenant.%s.%s.custom", tenantRule.TenantID, tenantRule.RuleID)

	case Override:
		moduleSrc = BuildRawModule(tenantRule)
		moduleName = fmt.Sprintf("tenant.%s.%s.override", tenantRule.TenantID, tenantRule.RuleID)

	case Extend:
		if tenantRule.CoreRuleID == nil {
			return nil, fmt.Errorf("extend mode but no core_rule_id")
		}
		core, err := tc.coreRepo.GetCoreRuleByID(context.TODO(), *tenantRule.CoreRuleID)
		if err != nil {
			return nil, err
		}
		moduleSrc = BuildCombinedModule(*core, tenantRule)
		moduleName = fmt.Sprintf("tenant.%s.%s.extend.v%d", tenantRule.TenantID, tenantRule.RuleID, core.Version)

	case Inherit:
		// Should not compile tenant function; use compiled core instead.
		return nil, fmt.Errorf("CompileTenantFn called for inherit mode")

	default:
		return nil, fmt.Errorf("unknown inherit mode %s", tenantRule.InheritMode)
	}

	thread := &starlark.Thread{Name: "compile_tenant"}
	globals, err := starlark.ExecFile(thread, moduleName, moduleSrc, tc.predeclared)
	if err != nil {
		return nil, fmt.Errorf("compile tenant rule %s: %w", tenantRule.RuleID, err)
	}

	entry, ok := globals["ok"]
	if !ok {
		return nil, fmt.Errorf("entrypoint ok not found in tenant module %s", moduleName)
	}
	fn, ok := entry.(*starlark.Function)
	if !ok {
		return nil, fmt.Errorf("entrypoint ok is not a function")
	}

	tc.tenantCache.Set(key, fn)
	return fn, nil
}
