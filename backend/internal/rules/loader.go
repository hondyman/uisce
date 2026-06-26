package rules

import (
	"fmt"

	"go.starlark.net/starlark"

	"github.com/hondyman/semlayer/backend/internal/starlib"
)

// ModuleLoader implements starlark.Thread.Load for whitelisted modules.
type ModuleLoader struct {
	// Allowed maps module name -> source code
	Allowed map[string]string
}

// NewModuleLoader creates a loader with the given allowed modules.
func NewModuleLoader(allowed map[string]string) *ModuleLoader {
	return &ModuleLoader{
		Allowed: allowed,
	}
}

// Load is the function to be assigned to thread.Load.
func (ml *ModuleLoader) Load(thread *starlark.Thread, module string) (starlark.StringDict, error) {
	src, ok := ml.Allowed[module]
	if !ok {
		return nil, fmt.Errorf("module %q not allowed", module)
	}

	// Use a restricted environment for the module
	// Note: Modules effectively run in their own scope but can be loaded by others.
	// We use the same baseline restricted lib for them.
	// Note: We don't inject 'ctx' here because modules are usually pure libraries.
	// Context is passed to functions (def my_func(ctx): ...)
	predecl := starlib.Lib() // Base library without context binding acts as the "Universe" for modules

	// Execute the module
	// We cache execution if needed, but for now strict re-execution ensures isolation per Load call in a thread
	// (Starlark caches Load results within a Thread automatically)
	globals, err := starlark.ExecFile(thread, module+".star", src, predecl)
	if err != nil {
		return nil, err
	}

	return globals, nil
}
