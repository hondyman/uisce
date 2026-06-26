package rules

import (
	"context"
	"fmt"
)

// WarmCoreCache pre-compiles and caches all active core rules.
// Should be called on service startup and after core rule updates.
func WarmCoreCache(ctx context.Context, compiler *CoreCompiler, coreRepo CoreRuleRepository) error {
	cores, err := coreRepo.ListActiveCoreRules(ctx)
	if err != nil {
		return fmt.Errorf("failed to list active core rules: %w", err)
	}

	for _, core := range cores {
		// CompileAndCache checks the cache first, so this is safe to call repeatedly
		if _, err := compiler.CompileAndCache(core); err != nil {
			// Log error but continue warming others?
			// For now, we return the error to signal startup failure or partial success
			return fmt.Errorf("failed to compile core rule %s v%d: %w", core.RuleKey, core.Version, err)
		}
	}
	return nil
}
