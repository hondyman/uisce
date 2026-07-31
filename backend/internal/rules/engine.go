package rules

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/checker/decls"
	"golang.org/x/sync/singleflight"

	vm "github.com/hondyman/uisce/backend/internal/rules/vm"
)

// RuleEngine evaluates RuleNode ASTs via the VM-backed fast path with
// atomic-pointer state swaps for multi-tenant dynamic rewarming.
//
// Architecture: Two-Level Execution Model
//
//   Level 1 — Core (platform-wide):  coreState atomic.Pointer[EngineState]
//     Holds gold-copy rules and standard fields. Serves all tenants.
//
//   Level 2 — Custom (tenant-isolated): tenantStates sync.Map
//     Per-tenant states for custom fields and custom rules. A tenant
//     state is built by merging Core rules + Tenant rules so it is
//     fully self-contained. When Tenant A adds a custom field, only
//     Tenant A's state is rebuilt; Tenant B and Core are untouched.
//
// Lifecycle:
//
//   1. NewRuleEngine → empty core state
//   2. RewarmCore(allCoreRules, version) once at platform startup
//   3. RewarmTenant(tenantID, allTenantRules, version) when a tenant
//      adds a custom field or custom rule
//   4. UpdateRule / DeleteRule for O(1) rule logic changes
//   5. Evaluate(ctx, tenantID, ruleID, version, node, input, force)
//
// Concurrency: Evaluate is lock-free. getState() resolves the correct
// state via atomic.Pointer.Load (core) or sync.Map.Load (tenant) — both
// are wait-free reads. RewarmCore/RewarmTenant build a new state in local
// memory and atomically swap it in.
type RuleEngine struct {
	env  *cel.Env
	repo RuleRepository

	coreState atomic.Pointer[EngineState]

	// tenantStates holds isolated states for tenants with custom rules/fields.
	// Key: tenantID (string). Value: *EngineState.
	// If a tenant has no custom state, evaluation falls back to coreState.
	tenantStates sync.Map

	vm          *vm.VM
	metrics     *EngineMetrics
	recursive   *AdvancedEvaluator
	rewarmGroup singleflight.Group
}

func NewRuleEngine(repo RuleRepository) *RuleEngine {
	env, _ := cel.NewEnv(
		cel.Declarations(
			decls.NewVar("input", decls.NewMapType(decls.String, decls.Dyn)),
		),
	)

	e := &RuleEngine{
		env:       env,
		repo:      repo,
		vm:        vm.NewVM(),
		metrics:   &EngineMetrics{},
		recursive: NewAdvancedEvaluator(),
	}
	e.coreState.Store(&EngineState{
		Syms:     vm.NewSymbolDict(),
		Enums:    vm.NewEnumDict(),
		Cache:    &sync.Map{},
		Version:  0,
		Revision: 0,
	})
	return e
}

// getState resolves the correct state for a tenant. If tenantID is non-empty
// and a tenant-specific state exists, it is returned; otherwise the core state
// is returned. The returned state is always safe for concurrent reads.
// It also updates the LastUsedUnixNano timestamp for TTL eviction tracking.
func (e *RuleEngine) getState(tenantID string) *EngineState {
	var state *EngineState
	if tenantID != "" {
		if ts, ok := e.tenantStates.Load(tenantID); ok {
			state = ts.(*EngineState)
		}
	}
	if state == nil {
		state = e.coreState.Load()
	}
	atomic.StoreInt64(&state.LastUsedUnixNano, time.Now().UnixNano())
	return state
}

// Evaluate is the hot path. It resolves the correct state (Core vs Tenant)
// and executes the rule. The tenantID parameter selects the execution level:
//   - "" (empty)    → core state (platform-wide gold copy rules)
//   - "tenant-id"   → tenant-specific state (custom rules + inherited core)
func (e *RuleEngine) Evaluate(
	ctx context.Context,
	tenantID string,
	ruleID string,
	version int,
	node *RuleNode,
	input map[string]any,
	force bool,
) (bool, *EvalTrace, error) {

	state := e.getState(tenantID)
	trace := &EvalTrace{
		RuleID:   cacheKeyFor(ruleID, version),
		Revision: state.Revision,
		IsTenant: tenantID != "" && state != e.coreState.Load(),
		TenantID: tenantID,
	}

	if node == nil {
		e.metrics.fallbacks.Add(1)
		trace.Fallback = "nil rule node"
		return false, trace, fmt.Errorf("nil rule node")
	}

	key := trace.RuleID
	var res *vm.CompileResult

	if !force {
		if cached, ok := state.Cache.Load(key); ok {
			res = cached.(*vm.CompileResult)
			e.metrics.cacheHits.Add(1)
		}
	}

	if res == nil {
		e.metrics.cacheMisses.Add(1)
		newRes := CompileVM(node, state.Syms, state.Enums)
		res = &newRes
		if !force {
			state.Cache.Store(key, res)
		}
	}

	if res.Unsupported != nil || len(res.Program.Insts) == 0 {
		e.metrics.fallbacks.Add(1)
		if res.Unsupported != nil {
			e.metrics.compileErrors.Add(1)
			trace.Fallback = res.Unsupported.Error()
		} else {
			trace.Fallback = "empty compiled program"
		}
		passed, err := e.recursive.Evaluate(*node, input)
		return passed, trace, err
	}

	e.metrics.vmPathCount.Add(1)
	trace.UsedVM = true

	rec := vm.Project(input, state.Syms, state.Enums)
	defer vm.PutFastRecord(rec)

	stack := vm.GetStack()
	defer vm.PutStack(stack)

	passed := e.vm.Run(res.Program, rec, stack)
	return passed, trace, nil
}

// buildState is the internal O(N) helper that compiles a rule corpus,
// freezes dictionaries, and returns a new EngineState ready to be published.
func buildState(rules []*RuleNode, version int, revision uint64) *EngineState {
	newSyms := vm.NewSymbolDict()
	newEnums := vm.NewEnumDict()

	for _, rule := range rules {
		extractPathsAndEnums(rule, newSyms, newEnums)
	}
	newSyms.Freeze()
	newEnums.Freeze()

	newCache := &sync.Map{}
	for _, rule := range rules {
		key := cacheKeyFor(rule.ID(), version)
		res := CompileVM(rule, newSyms, newEnums)
		newCache.Store(key, &res)
	}

	return &EngineState{
		Syms:     newSyms,
		Enums:    newEnums,
		Cache:    newCache,
		Version:  version,
		Revision: revision,
	}
}

// RewarmCore rebuilds the global core state and atomically swaps it in.
// Called ONLY during platform upgrades when a new core field is introduced
// or a gold-copy rule is added/changed. All tenant states are invalidated
// so they pick up the new core schema on their next evaluation.
// Uses singleflight to deduplicate concurrent calls.
func (e *RuleEngine) RewarmCore(allCoreRules []*RuleNode, version int) (uint64, error) {
	result, err, _ := e.rewarmGroup.Do("rewarm:core", func() (any, error) {
		oldState := e.coreState.Load()
		if version <= oldState.Version {
			return oldState.Revision, nil
		}
		newState := buildState(allCoreRules, version, oldState.Revision+1)
		e.coreState.Store(newState)

		var keysToDelete []string
		e.tenantStates.Range(func(key, _ any) bool {
			keysToDelete = append(keysToDelete, key.(string))
			return true
		})
		for _, k := range keysToDelete {
			e.tenantStates.Delete(k)
		}

		runtime.GC()
		return newState.Revision, nil
	})
	if err != nil {
		return 0, err
	}
	return result.(uint64), nil
}

// RewarmTenant rebuilds a specific tenant's state and stores it.
// allTenantRules must include BOTH the current core rules (inherited) and
// the tenant's custom rules so the tenant state is fully self-contained.
// When a tenant adds a custom field, only that tenant's state is rebuilt.
// Uses singleflight to deduplicate concurrent calls for the same tenantID.
// Skips rebuild if incoming version <= current cached version.
func (e *RuleEngine) RewarmTenant(tenantID string, allTenantRules []*RuleNode, version int) (uint64, error) {
	result, err, _ := e.rewarmGroup.Do("rewarm:tenant:"+tenantID, func() (any, error) {
		if oldState, ok := e.tenantStates.Load(tenantID); ok {
			if version <= oldState.(*EngineState).Version {
				return oldState.(*EngineState).Revision, nil
			}
		}
		revision := uint64(1)
		if oldState, ok := e.tenantStates.Load(tenantID); ok {
			revision = oldState.(*EngineState).Revision + 1
		}
		newState := buildState(allTenantRules, version, revision)
		e.tenantStates.Store(tenantID, newState)
		return newState.Revision, nil
	})
	if err != nil {
		return 0, err
	}
	return result.(uint64), nil
}

// InvalidateAllTenantStates removes all tenant-specific states.
// Used after a core schema change so tenants pick up the new core fields
// on their next full RewarmTenant. Existing in-flight evaluations
// complete safely on the old tenant states.
func (e *RuleEngine) InvalidateAllTenantStates() {
	e.tenantStates.Range(func(key, _ any) bool {
		e.tenantStates.Delete(key)
		return true
	})
}

// StartEvictor launches a background goroutine that evicts idle tenant states.
// It runs until the provided context is cancelled. The evictor removes any
// tenant state that has not been accessed within the given TTL duration,
// checked every interval. This prevents memory buildup from inactive tenants.
// Callers should invoke this once per engine instance.
func (e *RuleEngine) StartEvictor(ctx context.Context, ttl time.Duration, interval time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				e.EvictIdleStates(time.Now(), ttl)
			}
		}
	}()
}

// EvictIdleStates removes tenant states that have not been accessed within ttl.
// It returns the number of states evicted. Safe to call concurrently.
func (e *RuleEngine) EvictIdleStates(now time.Time, ttl time.Duration) int {
	evicted := 0
	e.tenantStates.Range(func(key, value any) bool {
		state := value.(*EngineState)
		lastUsed := time.Unix(0, atomic.LoadInt64(&state.LastUsedUnixNano))
		if now.Sub(lastUsed) > ttl {
			e.tenantStates.Delete(key)
			evicted++
		}
		return true
	})
	return evicted
}

// TenantCount returns the number of active tenant states currently held.
// Useful for monitoring and capacity planning.
func (e *RuleEngine) TenantCount() int {
	count := 0
	e.tenantStates.Range(func(_, _ any) bool {
		count++
		return true
	})
	return count
}

// UpdateRule incrementally compiles and caches a single rule in the
// correct state (Core or Tenant) without rebuilding anything else.
// O(1) — the hot path for CDC-driven rule logic updates.
func (e *RuleEngine) UpdateRule(tenantID string, ruleID string, version int, node *RuleNode) {
	state := e.getState(tenantID)
	key := cacheKeyFor(ruleID, version)
	res := CompileVM(node, state.Syms, state.Enums)
	state.Cache.Store(key, &res)
}

// DeleteRule removes a rule from the correct state (Core or Tenant).
func (e *RuleEngine) DeleteRule(tenantID string, ruleID string, version int) {
	state := e.getState(tenantID)
	key := cacheKeyFor(ruleID, version)
	state.Cache.Delete(key)
}

// HasField checks if a field path is registered in the correct state.
// Used by the CDC worker to decide between UpdateRule (field exists)
// and RewarmTenant (new field, need to rebuild tenant state).
func (e *RuleEngine) HasField(tenantID string, fieldPath string) bool {
	state := e.getState(tenantID)
	_, ok := state.Syms.Resolve(fieldPath)
	return ok
}

// CurrentRevision returns the core state's monotonic revision.
func (e *RuleEngine) CurrentRevision() uint64 {
	return e.coreState.Load().Revision
}

// CurrentCacheSize returns the number of cached programs in the core state.
func (e *RuleEngine) CurrentCacheSize() int {
	return e.coreState.Load().CacheSize()
}

// Metrics returns the cumulative engine metrics.
func (e *RuleEngine) Metrics() *EngineMetrics { return e.metrics }

// MetricsSnapshot returns a value-typed snapshot of the metrics.
func (e *RuleEngine) MetricsSnapshot() EngineMetricsSnapshot {
	return e.metrics.Snapshot()
}

// EvaluateNode is a backward-compatible wrapper that evaluates against the
// core state (tenantID="").
func (e *RuleEngine) EvaluateNode(ctx context.Context, node *RuleNode, input map[string]any) (bool, error) {
	passed, _, err := e.Evaluate(ctx, "", node.ID(), 0, node, input, false)
	return passed, err
}

// EvaluateCEL evaluates a CEL expression against the provided input.
func (e *RuleEngine) EvaluateCEL(ctx context.Context, expression string, input map[string]interface{}) (bool, error) {
	if e.env == nil {
		return false, fmt.Errorf("engine not initialized")
	}
	ast, issues := e.env.Compile(expression)
	if issues != nil && issues.Err() != nil {
		return false, fmt.Errorf("compile error: %w", issues.Err())
	}

	prg, err := e.env.Program(ast)
	if err != nil {
		return false, fmt.Errorf("program creation error: %w", err)
	}

	out, _, err := prg.Eval(map[string]interface{}{
		"input": input,
	})
	if err != nil {
		return false, fmt.Errorf("evaluation error: %w", err)
	}

	result, ok := out.Value().(bool)
	if !ok {
		return false, fmt.Errorf("expression did not return a boolean")
	}

	return result, nil
}

// EvaluateValue evaluates a CEL expression and returns the raw value.
func (e *RuleEngine) EvaluateValue(ctx context.Context, expression string, input map[string]interface{}) (interface{}, error) {
	if e.env == nil {
		return nil, fmt.Errorf("engine not initialized")
	}
	ast, issues := e.env.Compile(expression)
	if issues != nil && issues.Err() != nil {
		return nil, fmt.Errorf("compile error: %w", issues.Err())
	}

	prg, err := e.env.Program(ast)
	if err != nil {
		return nil, fmt.Errorf("program creation error: %w", err)
	}

	out, _, err := prg.Eval(map[string]interface{}{
		"input": input,
	})
	if err != nil {
		return nil, fmt.Errorf("evaluation error: %w", err)
	}

	return out.Value(), nil
}

func (e *RuleEngine) EvaluateTenantRule(ctx context.Context, rule *TenantValidationRule, boCtx map[string]map[string]interface{}) (bool, error) {
	return true, nil
}

func (e *RuleEngine) EvaluateExpr(ctx context.Context, expr string, boCtx map[string]map[string]interface{}) (bool, error) {
	flatInput := make(map[string]interface{})
	for k, v := range boCtx {
		flatInput[k] = v
	}
	return e.EvaluateCEL(ctx, expr, flatInput)
}

func (e *RuleEngine) EvaluateDurationExpr(ctx context.Context, expr string, boCtx map[string]map[string]interface{}) (int, error) {
	flatInput := make(map[string]interface{})
	for k, v := range boCtx {
		flatInput[k] = v
	}
	val, err := e.EvaluateValue(ctx, expr, flatInput)
	if err != nil {
		return 0, err
	}
	switch v := val.(type) {
	case int:
		return v, nil
	case int64:
		return int(v), nil
	case float64:
		return int(v), nil
	default:
		return 0, fmt.Errorf("expression returned non-numeric type: %T", val)
	}
}

type ConditionEvalTrace struct {
	Expression    string                 `json:"expr"`
	Input         map[string]interface{} `json:"input"`
	RuleMatched   bool                   `json:"matched"`
	ExecutionTime time.Duration          `json:"executionMs"`
	Explanation   string                 `json:"explanation"`
}

func (e *RuleEngine) EvaluateExprDebug(ctx context.Context, expr string, boCtx map[string]map[string]interface{}) (*ConditionEvalTrace, error) {
	return nil, fmt.Errorf("legacy Starlark evaluation is removed")
}

// cacheKeyFor returns the versioned cache key for a rule.
func cacheKeyFor(ruleID string, version int) string {
	if version == 0 {
		return ruleID
	}
	return fmt.Sprintf("%s@%d", ruleID, version)
}

// extractPathsAndEnums walks a RuleNode AST and interns every field
// path and string literal into the dictionaries.
func extractPathsAndEnums(node *RuleNode, syms *vm.SymbolDict, enums *vm.EnumDict) {
	if node == nil {
		return
	}
	if node.Type == NodeTypeGroup && node.Group != nil {
		for i := range node.Group.Conditions {
			extractPathsAndEnums(&node.Group.Conditions[i], syms, enums)
		}
		return
	}
	if node.Type == NodeTypeCondition && node.Condition != nil {
		cond := node.Condition
		path := cond.FieldPath
		if path == "" {
			path = cond.Field
		}
		if path != "" {
			syms.Intern(path)
		}
		if s, ok := cond.Value.(string); ok {
			enums.Intern(s)
		}
		if arr, ok := cond.Value.([]any); ok {
			for _, v := range arr {
				if s, ok := v.(string); ok {
					enums.Intern(s)
				}
			}
		}
	}
}
