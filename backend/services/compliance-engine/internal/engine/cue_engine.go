package engine

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	"cuelang.org/go/cue/load"

	"github.com/hondyman/semlayer/backend/services/compliance-engine/internal/models"
)

// ValidationEngine handles CUE-based validation
type ValidationEngine struct {
	ctx        *cue.Context
	cache      map[string]cue.Value
	cacheMutex sync.RWMutex
	policyPath string
}

// NewValidationEngine creates a new validation engine
func NewValidationEngine(policyPath string) *ValidationEngine {
	return &ValidationEngine{
		ctx:        cuecontext.New(),
		cache:      make(map[string]cue.Value),
		policyPath: policyPath,
	}
}

// LoadRuleSet loads and compiles CUE rules for a specific version
// Returns a compiled cue.Value that can be used for validation
func (ve *ValidationEngine) LoadRuleSet(versionDate string) (cue.Value, error) {
	// Check cache first
	cacheKey := versionDate
	ve.cacheMutex.RLock()
	if cached, exists := ve.cache[cacheKey]; exists {
		ve.cacheMutex.RUnlock()
		return cached, nil
	}
	ve.cacheMutex.RUnlock()

	// Construct path to CUE file
	policyFile := filepath.Join(ve.policyPath, versionDate, "trade_compliance.cue")

	// Check if file exists
	if _, err := os.Stat(policyFile); os.IsNotExist(err) {
		return cue.Value{}, fmt.Errorf("policy file not found for version %s: %s", versionDate, policyFile)
	}

	// Load the CUE instances
	cfg := &load.Config{
		Package: "compliance",
		Dir:     filepath.Join(ve.policyPath, versionDate),
	}

	instances := load.Instances([]string{"."}, cfg)
	if len(instances) == 0 {
		return cue.Value{}, fmt.Errorf("no CUE instances found for version %s", versionDate)
	}

	if err := instances[0].Err; err != nil {
		return cue.Value{}, fmt.Errorf("error loading CUE for version %s: %w", versionDate, err)
	}

	// Build the value
	val := ve.ctx.BuildInstance(instances[0])
	if err := val.Err(); err != nil {
		return cue.Value{}, fmt.Errorf("error building CUE value for version %s: %w", versionDate, err)
	}

	// Cache the compiled value
	ve.cacheMutex.Lock()
	ve.cache[cacheKey] = val
	ve.cacheMutex.Unlock()

	return val, nil
}

// Validate executes pre-trade compliance check
func (ve *ValidationEngine) Validate(ctx context.Context, trade models.TradeRequest, version string, checkType string) error {
	// Load the policy for this version
	policy, err := ve.LoadRuleSet(version)
	if err != nil {
		return fmt.Errorf("failed to load policy: %w", err)
	}

	// Select the specific check type (#PreTradeCheck or #PostTradeCheck)
	checkPath := fmt.Sprintf("#%sCheck", checkType)
	check := policy.LookupPath(cue.ParsePath(checkPath))
	if !check.Exists() {
		return fmt.Errorf("check type %s not found in version %s", checkPath, version)
	}

	// Unify the trade data with the CUE schema
	// This is where the "magic" happens - CUE validates the data against constraints
	tradeValue := ve.ctx.Encode(trade)
	result := check.Unify(tradeValue)

	// Validate with concrete values required
	if err := result.Validate(cue.Concrete(true)); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	return nil
}

// ClearCache clears the rule cache (useful for hot-reloading in development)
func (ve *ValidationEngine) ClearCache() {
	ve.cacheMutex.Lock()
	defer ve.cacheMutex.Unlock()
	ve.cache = make(map[string]cue.Value)
}
