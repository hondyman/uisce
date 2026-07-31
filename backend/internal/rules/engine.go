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

	vm "github.com/hondyman/uisce/backend/internal/rules/vm"
)

// RuleEngine evaluates RuleNode ASTs via the VM-backed fast path with
// atomic-pointer state swaps for multi-tenant dynamic rewarming.
//
// Lifecycle:
//
//   1. NewRuleEngine → empty initial state (no rules can compile yet)
//   2. SetInitialState(syms, enums, version) OR Rewarm(rules, version) once at startup
//   3. Evaluate(ctx, ruleID, version, node, input, force) on every request
//   4. External worker calls Rewarm(allRules, version) every ~60s when rules change
//
// Concurrency: Evaluate is lock-free on the hot path; Rewarm builds a
// new state in local memory and atomically swaps it in. In-flight
// requests finish on the old state; new requests use the new state.
type RuleEngine struct {
	// CEL evaluator retained for backwards compatibility with existing
	// callers that pass string expressions.
	env  *cel.Env
	repo RuleRepository

	state   atomic.Pointer[EngineState]
	vm      *vm.VM
	metrics *EngineMetrics

	// Recursive fallback evaluator. Stateless; safe for concurrent use.
	recursive *AdvancedEvaluator
}

// NewRuleEngine constructs an engine with an empty initial state.
// Call SetInitialState or Rewarm before any production traffic.
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
	// Initial empty state. Any Evaluate call before SetInitialState
	// or Rewarm will fall back to the recursive evaluator.
	e.state.Store(&EngineState{
		Syms:     vm.NewSymbolDict(),
		Enums:    vm.NewEnumDict(),
		Cache:    &sync.Map{},
		Version:  0,
		Revision: 0,
	})
	return e
}

// SetInitialState publishes the first schema dictionary. The caller is
// responsible for calling syms.Freeze() / enums.Freeze() beforehand
// (we do not double-freeze; frozen dicts are idempotent).
func (e *RuleEngine) SetInitialState(syms *vm.SymbolDict, enums *vm.EnumDict, version int) {
	newState := &EngineState{
		Syms:     syms,
		Enums:    enums,
		Cache:    &sync.Map{},
		Version:  version,
		Revision: 1,
	}
	oldState := e.state.Swap(newState)
	_ = oldState // first swap, no previous state to GC
}

// Rewarm builds a brand new EngineState from the provided rule corpus
// and atomically swaps it in. All rules in the corpus are pre-compiled
// during the build so the new state's cache is fully populated when the
// swap commits. In-flight requests on the old state complete safely;
// the old state is GC'd shortly after.
//
// Returns the new state's Revision on success.
func (e *RuleEngine) Rewarm(allRules []*RuleNode, version int) (uint64, error) {
	newSyms := vm.NewSymbolDict()
	newEnums := vm.NewEnumDict()

	for _, rule := range allRules {
		extractPathsAndEnums(rule, newSyms, newEnums)
	}
	newSyms.Freeze()
	newEnums.Freeze()

	newCache := &sync.Map{}

	// Pre-compile every rule into the new cache. After the swap, the
	// hot path is fully cached — no lazy compile on first request.
	for _, rule := range allRules {
		key := cacheKeyFor(rule.ID(), version)
		res := CompileVM(rule, newSyms, newEnums)
		newCache.Store(key, &res)
	}

	oldState := e.state.Load()
	newState := &EngineState{
		Syms:     newSyms,
		Enums:    newEnums,
		Cache:    newCache,
		Version:  version,
		Revision: oldState.Revision + 1,
	}
	e.state.Store(newState)

	// Best-effort GC: reclaim the old state immediately so memory
	// pressure doesn't spike during repeated rewarms.
	runtime.GC()

	return newState.Revision, nil
}

// Warmup is a convenience wrapper around Rewarm(version=0) for first-boot.
func (e *RuleEngine) Warmup(rules []*RuleNode) (uint64, error) {
	return e.Rewarm(rules, 0)
}

// cacheKeyFor returns the versioned cache key for a rule.
// Caller controls versioning: pass 0 to bypass versioning (useful
// for ad-hoc rules). Different versions of the same rule produce
// different keys, so updates naturally invalidate the cache.
func cacheKeyFor(ruleID string, version int) string {
	if version == 0 {
		return ruleID
	}
	return fmt.Sprintf("%s@%d", ruleID, version)
}

// Evaluate is the hot path. Reads the current state via atomic.Pointer.Load
// (lock-free, ~1 ns), checks the cache, lazy-compiles on miss, and runs
// the VM. On any failure (compile error, missing field, unsupported
// operator, empty program) it falls back to the recursive evaluator
// and records the reason in EvalTrace.Fallback.
func (e *RuleEngine) Evaluate(
	ctx context.Context,
	ruleID string,
	version int,
	node *RuleNode,
	input map[string]any,
	force bool,
) (bool, *EvalTrace, error) {

	state := e.state.Load()
	trace := &EvalTrace{
		RuleID:   cacheKeyFor(ruleID, version),
		Revision: state.Revision,
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
		// When force=true, do NOT cache the failure — let the caller
		// retry without polluting the cache.
		if !force {
			state.Cache.Store(key, res)
		}
	}

	// Fallback path: Unsupported operator / empty program / etc.
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

// CurrentRevision returns the active EngineState's monotonic revision.
func (e *RuleEngine) CurrentRevision() uint64 {
	return e.state.Load().Revision
}

// CurrentCacheSize returns the number of cached programs in the active state.
func (e *RuleEngine) CurrentCacheSize() int {
	return e.state.Load().CacheSize()
}

// Metrics returns the cumulative engine metrics (survive rewarm).
func (e *RuleEngine) Metrics() *EngineMetrics { return e.metrics }

// MetricsSnapshot returns a value-typed snapshot of the metrics.
func (e *RuleEngine) MetricsSnapshot() EngineMetricsSnapshot {
	return e.metrics.Snapshot()
}

// EvaluateNode is a backward-compatible wrapper that uses the legacy
// signature (no ruleID/version/force). It defaults version=0 and
// ruleID = node.ID(). New callers should use Evaluate directly.
func (e *RuleEngine) EvaluateNode(ctx context.Context, node *RuleNode, input map[string]any) (bool, error) {
	passed, _, err := e.Evaluate(ctx, node.ID(), 0, node, input, false)
	return passed, err
}

// EvaluateCEL evaluates a CEL expression against the provided input.
// (Legacy path retained for callers that pass string expressions.)
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