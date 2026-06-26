package rules

import (
	"fmt"

	"go.starlark.net/starlark"
)

type CoreCompiler struct {
	cache       *CoreFnCache
	predeclared starlark.StringDict // your helpers: field(), num_field(), etc.
}

func NewCoreCompiler(cache *CoreFnCache, predecl starlark.StringDict) *CoreCompiler {
	return &CoreCompiler{cache: cache, predeclared: predecl}
}

func (cc *CoreCompiler) CompileAndCache(core CoreValidationRule) (*starlark.Function, error) {
	key := CoreFnKey{CoreRuleID: core.CoreRuleID, Version: core.Version}
	if fn, ok := cc.cache.Get(key); ok {
		return fn, nil
	}

	thread := &starlark.Thread{Name: "compile_core"}
	globals, err := starlark.ExecFile(thread, core.ModuleName, core.ConditionSrc, cc.predeclared)
	if err != nil {
		return nil, fmt.Errorf("compile core rule %s v%d: %w", core.RuleKey, core.Version, err)
	}

	entry, ok := globals[core.Entrypoint]
	if !ok {
		return nil, fmt.Errorf("entrypoint %q not found in module %s", core.Entrypoint, core.ModuleName)
	}
	fn, ok := entry.(*starlark.Function)
	if !ok {
		return nil, fmt.Errorf("entrypoint %q is not a function", core.Entrypoint)
	}

	cc.cache.Set(key, fn)
	return fn, nil
}
