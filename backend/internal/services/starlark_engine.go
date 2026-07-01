package services

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/hondyman/semlayer/backend/internal/starlib"
	"github.com/jmoiron/sqlx"
	"go.starlark.net/starlark"
	"go.starlark.net/starlarkstruct"
	"go.starlark.net/syntax"
	"go.uber.org/zap"
)

func formatStarlarkError(err error) string {
	if err == nil {
		return ""
	}
	if ee, ok := err.(*starlark.EvalError); ok {
		bt := strings.TrimSpace(ee.Backtrace())
		if bt == "" {
			return ee.Error()
		}
		return ee.Error() + "\n" + bt
	}
	return err.Error()
}

// ============================================================================
// STARLARK EXPRESSION ENGINE
// XpressO-like calculated fields, validations, and condition rules
// ============================================================================

// ValidationResponse is the struct returned by success() and fail()
type ValidationResponse struct {
	Pass    bool   `json:"pass"`
	Message string `json:"message"`
}

// StarlarkEngine provides Workday-style expression evaluation
type StarlarkEngine struct {
	db       *sqlx.DB
	logger   *zap.Logger
	cache    map[string]*starlark.Program // Compiled expression cache
	cacheMu  sync.RWMutex
	maxSteps uint64
}

// NewStarlarkEngine creates a new expression engine
func NewStarlarkEngine(db *sqlx.DB) *StarlarkEngine {
	logger, _ := zap.NewProduction()
	return &StarlarkEngine{
		db:       db,
		logger:   logger,
		cache:    make(map[string]*starlark.Program),
		maxSteps: 1_000_000,
	}
}

func (e *StarlarkEngine) newThread(name string) *starlark.Thread {
	thread := &starlark.Thread{
		Name: name,
		Print: func(_ *starlark.Thread, msg string) {
			e.logger.Debug("Starlark print", zap.String("message", msg))
		},
	}
	if e.maxSteps > 0 {
		thread.SetMaxExecutionSteps(e.maxSteps)
	}
	return thread
}

func predeclaredSignature(predeclared starlark.StringDict) string {
	keys := make([]string, 0, len(predeclared))
	for k := range predeclared {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	h := sha256.New()
	for _, k := range keys {
		_, _ = h.Write([]byte(k))
		_, _ = h.Write([]byte{0})
	}
	return hex.EncodeToString(h.Sum(nil))
}

func (e *StarlarkEngine) getOrCompileProgram(filename string, script string, predeclared starlark.StringDict) (*starlark.Program, error) {
	scriptSum := sha256.Sum256([]byte(script))
	key := hex.EncodeToString(scriptSum[:]) + ":" + predeclaredSignature(predeclared)

	e.cacheMu.RLock()
	prog := e.cache[key]
	e.cacheMu.RUnlock()
	if prog != nil {
		observeStarlarkProgramCacheHit(filename)
		return prog, nil
	}
	observeStarlarkProgramCacheMiss(filename)

	e.cacheMu.Lock()
	defer e.cacheMu.Unlock()
	if prog := e.cache[key]; prog != nil {
		observeStarlarkProgramCacheHit(filename)
		return prog, nil
	}

	compileStart := time.Now()
	_, compiled, err := starlark.SourceProgramOptions(syntax.LegacyFileOptions(), filename, script, predeclared.Has)
	if err != nil {
		observeStarlarkProgramCompile(filename, time.Since(compileStart), "error")
		return nil, err
	}
	observeStarlarkProgramCompile(filename, time.Since(compileStart), "ok")
	e.cache[key] = compiled
	starlarkProgramCacheEntries.Set(float64(len(e.cache)))
	return compiled, nil
}

// ExpressionType defines the type of expression
type ExpressionType string

const (
	ExpressionTypeValidation  ExpressionType = "validation"
	ExpressionTypeCalculation ExpressionType = "calculation"
	ExpressionTypeCondition   ExpressionType = "condition"
)

// Expression represents a stored expression rule
type Expression struct {
	ID               string         `json:"id"`
	TenantID         string         `json:"tenant_id"`
	BusinessObjectID string         `json:"business_object_id,omitempty"`
	FieldKey         string         `json:"field_key,omitempty"`
	RuleType         ExpressionType `json:"rule_type"`
	Name             string         `json:"name"`
	Description      string         `json:"description,omitempty"`
	Script           string         `json:"script"`
	IsActive         bool           `json:"is_active"`
	Version          int            `json:"version"`
}

// StarlarkValidationResult represents the result of a validation expression
type StarlarkValidationResult struct {
	IsValid  bool   `json:"is_valid"`
	Message  string `json:"message,omitempty"`
	Severity string `json:"severity"` // error, warning, info
}

// OkRule represents an ok-style Starlark rule.
//
// Script must set a global `ok = <bool>` and may set `message = <string>`.
type OkRule struct {
	ID     string
	Script string
}

// OkRuleMeta is lightweight execution metadata used to optimize BO-scale runs.
type OkRuleMeta struct {
	// Cost is a relative cost score (lower runs earlier).
	Cost int
	// FailureLikelihood is a relative likelihood in [0,1] (higher runs earlier within same Cost).
	FailureLikelihood float64
	// RequiredFieldPaths are dot-paths that must be present in the input record for this rule.
	// Example: "account.account_type" or "page.aum".
	RequiredFieldPaths []string
}

// OkRuleWithMeta bundles a rule with its execution metadata.
type OkRuleWithMeta struct {
	OkRule
	Meta OkRuleMeta
}

// ============================================================================
// EXPRESSION EVALUATION
// ============================================================================

// EvaluateValidation runs a validation expression and returns pass/fail
func (e *StarlarkEngine) EvaluateValidation(ctx context.Context, script string, data map[string]interface{}) (*StarlarkValidationResult, error) {
	result, err := e.execute(script, data)
	if err != nil {
		return &StarlarkValidationResult{IsValid: false, Message: err.Error(), Severity: "error"}, err
	}

	// Handle tuple result (is_valid, message)
	if tuple, ok := result.(*starlark.Tuple); ok && tuple.Len() >= 2 {
		isValid := bool(tuple.Index(0).Truth())
		message := ""
		if str, ok := tuple.Index(1).(starlark.String); ok {
			message = string(str)
		}
		severity := "error"
		if isValid {
			severity = "info"
		}
		return &StarlarkValidationResult{IsValid: isValid, Message: message, Severity: severity}, nil
	}

	// Handle boolean result
	isValid := bool(result.Truth())
	return &StarlarkValidationResult{IsValid: isValid, Severity: "error"}, nil
}

// EvaluateCalculation runs a calculation expression and returns the result
func (e *StarlarkEngine) EvaluateCalculation(ctx context.Context, script string, data map[string]interface{}) (interface{}, error) {
	result, err := e.execute(script, data)
	if err != nil {
		return nil, err
	}
	return e.starlarkToGo(result), nil
}

// EvaluateCondition runs a condition expression and returns the action
func (e *StarlarkEngine) EvaluateCondition(ctx context.Context, script string, data map[string]interface{}) (string, error) {
	result, err := e.execute(script, data)
	if err != nil {
		return "", err
	}

	if str, ok := result.(starlark.String); ok {
		return string(str), nil
	}

	return fmt.Sprintf("%v", result), nil
}

// EvaluateUserRule runs a user-defined validation rule (def validate(context): ...)
func (e *StarlarkEngine) EvaluateUserRule(ctx context.Context, script string, data map[string]interface{}) (*StarlarkValidationResult, error) {
	// Backward compatibility: if the script defines validate(context), keep the legacy pathway.
	// Otherwise, support Expresso-ish scripts that set `ok = ...` and use `ctx` + helper builtins.
	if !strings.Contains(script, "def validate(") {
		start := time.Now()
		span, _ := startStarlarkRuleSpan(ctx, starlarkRuleIDFromScript(script), "ok")
		res, err := e.evaluateOkRule(ctx, script, data)
		span.end(res, err)
		observeStarlarkRule(starlarkRuleIDFromScript(script), "ok", time.Since(start), classifyStarlarkOutcome(res, err))
		return res, err
	}

	start := time.Now()
	ruleID := starlarkRuleIDFromScript(script)
	span, _ := startStarlarkRuleSpan(ctx, ruleID, "legacy")

	// 1. Create thread
	thread := e.newThread("validation")

	// 2. Build builtins (success, fail, etc.)
	predeclared := e.buildPredeclared(data)
	predeclared["success"] = starlark.NewBuiltin("success", e.builtinSuccess)
	predeclared["fail"] = starlark.NewBuiltin("fail", e.builtinFail)

	// 3. Execute the user script to define 'validate' function (compiled+cached)
	prog, err := e.getOrCompileProgram("rule.star", script, predeclared)
	if err != nil {
		res := &StarlarkValidationResult{IsValid: false, Message: "Script error: " + formatStarlarkError(err), Severity: "error"}
		span.end(res, nil)
		observeStarlarkRule(ruleID, "legacy", time.Since(start), classifyStarlarkOutcome(res, nil))
		return res, nil
	}
	globals, err := prog.Init(thread, predeclared)
	if err != nil {
		res := &StarlarkValidationResult{IsValid: false, Message: "Script error: " + formatStarlarkError(err), Severity: "error"}
		span.end(res, nil)
		observeStarlarkRule(ruleID, "legacy", time.Since(start), classifyStarlarkOutcome(res, nil))
		return res, nil
	}

	// 4. Find 'validate' function
	validateFn, ok := globals["validate"]
	if !ok {
		res := &StarlarkValidationResult{IsValid: false, Message: "Function 'validate' not found in script", Severity: "error"}
		span.end(res, nil)
		observeStarlarkRule(ruleID, "legacy", time.Since(start), classifyStarlarkOutcome(res, nil))
		return res, nil
	}

	// 5. Create context object (already in predeclared["context"], but we need to pass it as arg)
	// We can reuse the one from predeclared if we can cast it, or recreate it.
	// buildPredeclared puts it in as starlarkstruct.
	ctxVal, ok := predeclared["context"]
	if !ok {
		err := fmt.Errorf("context not found")
		span.end(nil, err)
		observeStarlarkRule(ruleID, "legacy", time.Since(start), classifyStarlarkOutcome(nil, err))
		return nil, err
	}

	// 6. Call validate(context)
	result, err := starlark.Call(thread, validateFn, starlark.Tuple{ctxVal}, nil)
	if err != nil {
		res := &StarlarkValidationResult{IsValid: false, Message: "Runtime error: " + formatStarlarkError(err), Severity: "error"}
		span.end(res, nil)
		observeStarlarkRule(ruleID, "legacy", time.Since(start), classifyStarlarkOutcome(res, nil))
		return res, nil
	}

	// 7. Parse result (ValidationResponse struct)
	if s, ok := result.(*starlarkstruct.Struct); ok {
		// We expect fields "pass" (bool) and "message" (string)
		passVal, err := s.Attr("pass")
		if err != nil {
			res := &StarlarkValidationResult{IsValid: false, Message: "Invalid return value from success/fail", Severity: "error"}
			span.end(res, nil)
			observeStarlarkRule(ruleID, "legacy", time.Since(start), classifyStarlarkOutcome(res, nil))
			return res, nil
		}

		msgVal, _ := s.Attr("message")

		isValid := bool(passVal.Truth())
		message := ""
		if str, ok := msgVal.(starlark.String); ok {
			message = string(str)
		}

		severity := "error"
		if isValid {
			severity = "info"
		}
		res := &StarlarkValidationResult{IsValid: isValid, Message: message, Severity: severity}
		span.end(res, nil)
		observeStarlarkRule(ruleID, "legacy", time.Since(start), classifyStarlarkOutcome(res, nil))
		return res, nil
	}

	// Fallback
	res := &StarlarkValidationResult{IsValid: false, Message: fmt.Sprintf("Invalid return type: %s", result.Type()), Severity: "error"}
	span.end(res, nil)
	observeStarlarkRule(ruleID, "legacy", time.Since(start), classifyStarlarkOutcome(res, nil))
	return res, nil
}

// EvaluateUserRuleBatch evaluates the same rule against many records.
//
// For ok-style scripts (no `def validate(`), it compiles once and reuses the compiled program
// across records; each worker uses its own thread and constructs a fresh ctx per record.
//
// For legacy validate-style scripts, it falls back to per-record EvaluateUserRule (still safe, but
// may compile multiple variants depending on the record's predeclared shape).
func (e *StarlarkEngine) EvaluateUserRuleBatch(
	ctx context.Context,
	script string,
	records []map[string]interface{},
	workers int,
) ([]*StarlarkValidationResult, error) {
	if len(records) == 0 {
		return []*StarlarkValidationResult{}, nil
	}
	if workers <= 0 {
		workers = runtime.GOMAXPROCS(0)
		if workers <= 0 {
			workers = 1
		}
	}
	if workers > len(records) {
		workers = len(records)
	}

	// Legacy path: correctness first.
	if strings.Contains(script, "def validate(") {
		results := make([]*StarlarkValidationResult, len(records))
		jobs := make(chan int)
		errCh := make(chan error, 1)
		var wg sync.WaitGroup

		workerFn := func() {
			defer wg.Done()
			for idx := range jobs {
				if ctx.Err() != nil {
					return
				}
				res, err := e.EvaluateUserRule(ctx, script, records[idx])
				if err != nil {
					select {
					case errCh <- err:
					default:
					}
					return
				}
				results[idx] = res
			}
		}

		wg.Add(workers)
		for i := 0; i < workers; i++ {
			go workerFn()
		}

		for i := range records {
			select {
			case <-ctx.Done():
				close(jobs)
				wg.Wait()
				return nil, ctx.Err()
			case err := <-errCh:
				close(jobs)
				wg.Wait()
				return nil, err
			case jobs <- i:
			}
		}
		close(jobs)
		wg.Wait()
		select {
		case err := <-errCh:
			return nil, err
		default:
		}
		return results, nil
	}

	// Ok-style fast path.
	ruleID := starlarkRuleIDFromScript(script)
	dummyCtx := starlib.BuildCtx(nil, nil)
	dummyGlobals := starlark.StringDict{"ctx": dummyCtx}
	for k, v := range starlib.LibWithCtx(dummyCtx) {
		dummyGlobals[k] = v
	}
	prog, err := e.getOrCompileProgram("rule.star", script, dummyGlobals)
	if err != nil {
		return nil, err
	}

	results := make([]*StarlarkValidationResult, len(records))
	jobs := make(chan int)
	errCh := make(chan error, 1)
	var wg sync.WaitGroup

	workerFn := func(workerID int) {
		defer wg.Done()
		thread := e.newThread(fmt.Sprintf("validation_ok_batch_%d", workerID))
		for idx := range jobs {
			if ctx.Err() != nil {
				return
			}
			start := time.Now()
			span, _ := startStarlarkRuleSpan(ctx, ruleID, "ok_batch")
			res, err := e.evaluateOkRuleWithProgram(thread, prog, script, records[idx])
			span.end(res, err)
			observeStarlarkRule(ruleID, "ok_batch", time.Since(start), classifyStarlarkOutcome(res, err))
			if err != nil {
				select {
				case errCh <- err:
				default:
				}
				return
			}
			results[idx] = res
		}
	}

	for i := 0; i < workers; i++ {
		wg.Add(1)
		go workerFn(i + 1)
	}

	for i := range records {
		select {
		case <-ctx.Done():
			close(jobs)
			wg.Wait()
			return nil, ctx.Err()
		case err := <-errCh:
			close(jobs)
			wg.Wait()
			return nil, err
		case jobs <- i:
		}
	}
	close(jobs)
	wg.Wait()
	select {
	case err := <-errCh:
		return nil, err
	default:
	}
	return results, nil
}

// EvaluateOkRuleBundleBatch evaluates many ok-style rules against many records.
//
// It compiles each rule once, then for each record builds ctx+globals once and reuses them
// across all rule evaluations.
//
// If shortCircuit is true, evaluation stops at the first failing rule for that record and the
// remaining results for that record are left as nil.
//
// Results are returned as results[recordIndex][ruleIndex].
func (e *StarlarkEngine) EvaluateOkRuleBundleBatch(
	ctx context.Context,
	rules []OkRule,
	records []map[string]interface{},
	workers int,
	shortCircuit bool,
) ([][]*StarlarkValidationResult, error) {
	withMeta := make([]OkRuleWithMeta, 0, len(rules))
	for _, r := range rules {
		withMeta = append(withMeta, OkRuleWithMeta{OkRule: r})
	}
	return e.EvaluateOkRuleBundleBatchWithMeta(ctx, withMeta, records, workers, shortCircuit)
}

// EvaluateOkRuleBundleBatchWithMeta evaluates many ok-style rules against many records.
//
// It uses rule metadata to:
// - order rules by (Cost asc, FailureLikelihood desc)
// - pre-project each record to only the union of RequiredFieldPaths (reduces ctx build cost)
//
// Results are returned as results[recordIndex][ruleIndex] where ruleIndex matches the input slice.
// When shortCircuit is true, evaluation stops at the first failing rule (in ordered execution),
// and remaining rules for that record are left nil (potentially non-contiguous by ruleIndex).
func (e *StarlarkEngine) EvaluateOkRuleBundleBatchWithMeta(
	ctx context.Context,
	rules []OkRuleWithMeta,
	records []map[string]interface{},
	workers int,
	shortCircuit bool,
) ([][]*StarlarkValidationResult, error) {
	if len(records) == 0 {
		return [][]*StarlarkValidationResult{}, nil
	}
	if len(rules) == 0 {
		out := make([][]*StarlarkValidationResult, len(records))
		for i := range out {
			out[i] = []*StarlarkValidationResult{}
		}
		return out, nil
	}
	if workers <= 0 {
		workers = runtime.GOMAXPROCS(0)
		if workers <= 0 {
			workers = 1
		}
	}
	if workers > len(records) {
		workers = len(records)
	}

	// Determine execution order from metadata.
	order := make([]int, 0, len(rules))
	for i := range rules {
		order = append(order, i)
	}
	sort.Slice(order, func(i, j int) bool {
		a := rules[order[i]].Meta
		b := rules[order[j]].Meta
		if a.Cost != b.Cost {
			return a.Cost < b.Cost
		}
		if a.FailureLikelihood != b.FailureLikelihood {
			return a.FailureLikelihood > b.FailureLikelihood
		}
		return rules[order[i]].ID < rules[order[j]].ID
	})

	// Union required fieldpaths across all rules.
	var requiredPaths []string
	{
		set := make(map[string]struct{})
		for _, r := range rules {
			paths := r.Meta.RequiredFieldPaths
			if len(paths) == 0 {
				derived, err := starlib.ExtractRequiredFieldPaths(r.Script)
				if err == nil {
					paths = derived
				}
			}
			for _, p := range paths {
				pp := strings.TrimSpace(p)
				if pp == "" {
					continue
				}
				set[pp] = struct{}{}
			}
		}
		if len(set) > 0 {
			requiredPaths = make([]string, 0, len(set))
			for p := range set {
				requiredPaths = append(requiredPaths, p)
			}
			sort.Strings(requiredPaths)
		}
	}

	// Compile all rules once using a stable predeclared shape.
	dummyCtx := starlib.BuildCtx(nil, nil)
	dummyGlobals := starlark.StringDict{"ctx": dummyCtx}
	for k, v := range starlib.LibWithCtx(dummyCtx) {
		dummyGlobals[k] = v
	}

	programs := make([]*starlark.Program, len(rules))
	for i, r := range rules {
		prog, err := e.getOrCompileProgram("rule.star", r.Script, dummyGlobals)
		if err != nil {
			return nil, err
		}
		programs[i] = prog
	}

	results := make([][]*StarlarkValidationResult, len(records))
	for i := range results {
		results[i] = make([]*StarlarkValidationResult, len(rules))
	}

	jobs := make(chan int)
	errCh := make(chan error, 1)
	var wg sync.WaitGroup

	workerFn := func(workerID int) {
		defer wg.Done()
		thread := e.newThread(fmt.Sprintf("validation_ok_bundle_%d", workerID))
		for recordIdx := range jobs {
			if ctx.Err() != nil {
				return
			}

			bundleSpan, bundleCtx := startStarlarkBundleSpan(ctx, "ok_bundle", len(rules), shortCircuit)

			record := records[recordIdx]
			if len(requiredPaths) > 0 {
				record = starlib.ProjectRecord(record, requiredPaths)
			}

			page, objects := starlib.SplitDataIntoPageAndObjects(record)
			ctxDict := starlib.BuildCtx(page, objects)
			ctxDict.Freeze() // ensures rules can't mutate ctx and affect subsequent rules

			globals := starlark.StringDict{"ctx": ctxDict}
			for k, v := range starlib.LibWithCtx(ctxDict) {
				globals[k] = v
			}

			for _, ruleIdx := range order {
				prog := programs[ruleIdx]
				start := time.Now()
				rid := rules[ruleIdx].ID
				if strings.TrimSpace(rid) == "" {
					rid = starlarkRuleIDFromScript(rules[ruleIdx].Script)
				}
				span, _ := startStarlarkRuleSpan(bundleCtx, rid, "ok_bundle")
				res, err := e.evaluateOkProgram(thread, prog, globals)
				span.end(res, err)
				observeStarlarkRule(rid, "ok_bundle", time.Since(start), classifyStarlarkOutcome(res, err))
				if err != nil {
					bundleSpan.end(err)
					select {
					case errCh <- err:
					default:
					}
					return
				}
				results[recordIdx][ruleIdx] = res
				if shortCircuit && res != nil && !res.IsValid {
					break
				}
			}
			bundleSpan.end(nil)
		}
	}

	for i := 0; i < workers; i++ {
		wg.Add(1)
		go workerFn(i + 1)
	}

	for i := range records {
		select {
		case <-ctx.Done():
			close(jobs)
			wg.Wait()
			return nil, ctx.Err()
		case err := <-errCh:
			close(jobs)
			wg.Wait()
			return nil, err
		case jobs <- i:
		}
	}
	close(jobs)
	wg.Wait()
	select {
	case err := <-errCh:
		return nil, err
	default:
	}
	return results, nil
}

func (e *StarlarkEngine) evaluateOkProgram(
	thread *starlark.Thread,
	prog *starlark.Program,
	globals starlark.StringDict,
) (*StarlarkValidationResult, error) {
	resultGlobals, err := prog.Init(thread, globals)
	if err != nil {
		return &StarlarkValidationResult{IsValid: false, Message: "Script error: " + formatStarlarkError(err), Severity: "error"}, nil
	}

	v := resultGlobals["ok"]
	if v == nil {
		return &StarlarkValidationResult{IsValid: false, Message: "Rule did not define 'ok'", Severity: "error"}, nil
	}
	b, ok := v.(starlark.Bool)
	if !ok {
		return &StarlarkValidationResult{IsValid: false, Message: fmt.Sprintf("'ok' must be bool, got %s", v.Type()), Severity: "error"}, nil
	}

	msg := ""
	if mv := resultGlobals["message"]; mv != nil {
		if ms, ok := mv.(starlark.String); ok {
			msg = string(ms)
		}
	}

	severity := "error"
	if bool(b) {
		severity = "info"
	}
	return &StarlarkValidationResult{IsValid: bool(b), Message: msg, Severity: severity}, nil
}

func (e *StarlarkEngine) evaluateOkRuleWithProgram(
	thread *starlark.Thread,
	prog *starlark.Program,
	script string,
	data map[string]interface{},
) (*StarlarkValidationResult, error) {
	page, objects := starlib.SplitDataIntoPageAndObjects(data)
	ctxDict := starlib.BuildCtx(page, objects)

	globals := starlark.StringDict{"ctx": ctxDict}
	for k, v := range starlib.LibWithCtx(ctxDict) {
		globals[k] = v
	}

	return e.evaluateOkProgram(thread, prog, globals)
}

func (e *StarlarkEngine) evaluateOkRule(ctx context.Context, script string, data map[string]interface{}) (*StarlarkValidationResult, error) {
	thread := e.newThread("validation_ok")

	page, objects := starlib.SplitDataIntoPageAndObjects(data)
	ctxDict := starlib.BuildCtx(page, objects)

	globals := starlark.StringDict{
		"ctx": ctxDict,
	}
	for k, v := range starlib.LibWithCtx(ctxDict) {
		globals[k] = v
	}

	prog, err := e.getOrCompileProgram("rule.star", script, globals)
	if err != nil {
		return &StarlarkValidationResult{IsValid: false, Message: "Script error: " + formatStarlarkError(err), Severity: "error"}, nil
	}
	return e.evaluateOkRuleWithProgram(thread, prog, script, data)
}

// ============================================================================
// CORE EXECUTION
// ============================================================================

func (e *StarlarkEngine) execute(script string, data map[string]interface{}) (starlark.Value, error) {
	thread := e.newThread("expression")

	// Build predeclared globals with context data
	predeclared := e.buildPredeclared(data)

	// Wrap script in a result assignment
	wrappedScript := fmt.Sprintf(`
%s
`, script)

	// Execute the script (compiled+cached)
	prog, err := e.getOrCompileProgram("expression.star", wrappedScript, predeclared)
	if err != nil {
		e.logger.Error("Starlark compile failed", zap.Error(err))
		return nil, fmt.Errorf("expression error: %w", err)
	}
	globals, err := prog.Init(thread, predeclared)
	if err != nil {
		e.logger.Error("Starlark execution failed", zap.Error(err))
		return nil, fmt.Errorf("expression error: %w", err)
	}

	// Look for 'result' in globals
	if result, ok := globals["result"]; ok {
		return result, nil
	}

	// Return last value or None
	return starlark.None, nil
}

// buildPredeclared creates Starlark globals from Go data
func (e *StarlarkEngine) buildPredeclared(data map[string]interface{}) starlark.StringDict {
	predeclared := starlark.StringDict{
		// Built-in functions
		"abs":    starlark.NewBuiltin("abs", e.builtinAbs),
		"min":    starlark.NewBuiltin("min", e.builtinMin),
		"max":    starlark.NewBuiltin("max", e.builtinMax),
		"round":  starlark.NewBuiltin("round", e.builtinRound),
		"len":    starlark.NewBuiltin("len", e.builtinLen),
		"str":    starlark.NewBuiltin("str", e.builtinStr),
		"int":    starlark.NewBuiltin("int", e.builtinInt),
		"float":  starlark.NewBuiltin("float", e.builtinFloat),
		"bool":   starlark.NewBuiltin("bool", e.builtinBool),
		"struct": starlark.NewBuiltin("struct", starlarkstruct.Make),
	}

	// Add data as 'context' struct
	contextDict := starlark.StringDict{}
	for key, value := range data {
		contextDict[key] = e.goToStarlark(value)
	}
	predeclared["context"] = starlarkstruct.FromStringDict(starlark.String("context"), contextDict)

	// Also add data fields directly for convenience
	for key, value := range data {
		predeclared[key] = e.goToStarlark(value)
	}

	return predeclared
}

// ============================================================================
// TYPE CONVERSION
// ============================================================================

func (e *StarlarkEngine) goToStarlark(v interface{}) starlark.Value {
	switch val := v.(type) {
	case nil:
		return starlark.None
	case bool:
		return starlark.Bool(val)
	case int:
		return starlark.MakeInt(val)
	case int64:
		return starlark.MakeInt64(val)
	case float64:
		return starlark.Float(val)
	case string:
		return starlark.String(val)
	case []interface{}:
		items := make([]starlark.Value, len(val))
		for i, item := range val {
			items[i] = e.goToStarlark(item)
		}
		return starlark.NewList(items)
	case map[string]interface{}:
		dict := starlark.StringDict{}
		for k, v := range val {
			dict[k] = e.goToStarlark(v)
		}
		return starlarkstruct.FromStringDict(starlark.String("data"), dict)
	default:
		// Try JSON marshaling as fallback
		jsonBytes, _ := json.Marshal(val)
		return starlark.String(string(jsonBytes))
	}
}

func (e *StarlarkEngine) starlarkToGo(v starlark.Value) interface{} {
	switch val := v.(type) {
	case starlark.NoneType:
		return nil
	case starlark.Bool:
		return bool(val)
	case starlark.Int:
		i, _ := val.Int64()
		return i
	case starlark.Float:
		return float64(val)
	case starlark.String:
		return string(val)
	case *starlark.List:
		result := make([]interface{}, val.Len())
		for i := 0; i < val.Len(); i++ {
			result[i] = e.starlarkToGo(val.Index(i))
		}
		return result
	case *starlark.Dict:
		result := make(map[string]interface{})
		for _, item := range val.Items() {
			if key, ok := item[0].(starlark.String); ok {
				result[string(key)] = e.starlarkToGo(item[1])
			}
		}
		return result
	case *starlarkstruct.Struct:
		result := make(map[string]interface{})
		for _, name := range val.AttrNames() {
			attr, _ := val.Attr(name)
			result[name] = e.starlarkToGo(attr)
		}
		return result
	default:
		return fmt.Sprintf("%v", v)
	}
}

// ============================================================================
// BUILT-IN FUNCTIONS
// ============================================================================

func (e *StarlarkEngine) builtinAbs(_ *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, _ []starlark.Tuple) (starlark.Value, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("abs: got %d arguments, want 1", len(args))
	}
	switch v := args[0].(type) {
	case starlark.Int:
		i, _ := v.Int64()
		if i < 0 {
			i = -i
		}
		return starlark.MakeInt64(i), nil
	case starlark.Float:
		f := float64(v)
		if f < 0 {
			f = -f
		}
		return starlark.Float(f), nil
	}
	return nil, fmt.Errorf("abs: unsupported type")
}

func (e *StarlarkEngine) builtinMin(_ *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, _ []starlark.Tuple) (starlark.Value, error) {
	if len(args) < 1 {
		return nil, fmt.Errorf("min: requires at least 1 argument")
	}
	minVal := args[0]
	for _, arg := range args[1:] {
		// Simple numeric comparison
		switch a := arg.(type) {
		case starlark.Int:
			if m, ok := minVal.(starlark.Int); ok {
				ai, _ := a.Int64()
				mi, _ := m.Int64()
				if ai < mi {
					minVal = arg
				}
			}
		case starlark.Float:
			if m, ok := minVal.(starlark.Float); ok {
				if float64(a) < float64(m) {
					minVal = arg
				}
			}
		}
	}
	return minVal, nil
}

func (e *StarlarkEngine) builtinMax(_ *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, _ []starlark.Tuple) (starlark.Value, error) {
	if len(args) < 1 {
		return nil, fmt.Errorf("max: requires at least 1 argument")
	}
	maxVal := args[0]
	for _, arg := range args[1:] {
		// Simple numeric comparison
		switch a := arg.(type) {
		case starlark.Int:
			if m, ok := maxVal.(starlark.Int); ok {
				ai, _ := a.Int64()
				mi, _ := m.Int64()
				if ai > mi {
					maxVal = arg
				}
			}
		case starlark.Float:
			if m, ok := maxVal.(starlark.Float); ok {
				if float64(a) > float64(m) {
					maxVal = arg
				}
			}
		}
	}
	return maxVal, nil
}

func (e *StarlarkEngine) builtinRound(_ *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, _ []starlark.Tuple) (starlark.Value, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("round: got %d arguments, want 1", len(args))
	}
	if f, ok := args[0].(starlark.Float); ok {
		return starlark.MakeInt64(int64(float64(f) + 0.5)), nil
	}
	return args[0], nil
}

func (e *StarlarkEngine) builtinLen(_ *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, _ []starlark.Tuple) (starlark.Value, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("len: got %d arguments, want 1", len(args))
	}
	if seq, ok := args[0].(starlark.Sequence); ok {
		return starlark.MakeInt(seq.Len()), nil
	}
	if str, ok := args[0].(starlark.String); ok {
		return starlark.MakeInt(len(str)), nil
	}
	return nil, fmt.Errorf("len: unsupported type")
}

func (e *StarlarkEngine) builtinStr(_ *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, _ []starlark.Tuple) (starlark.Value, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("str: got %d arguments, want 1", len(args))
	}
	return starlark.String(args[0].String()), nil
}

func (e *StarlarkEngine) builtinInt(_ *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, _ []starlark.Tuple) (starlark.Value, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("int: got %d arguments, want 1", len(args))
	}
	switch v := args[0].(type) {
	case starlark.Int:
		return v, nil
	case starlark.Float:
		return starlark.MakeInt64(int64(v)), nil
	case starlark.String:
		var i int64
		fmt.Sscanf(string(v), "%d", &i)
		return starlark.MakeInt64(i), nil
	}
	return starlark.MakeInt(0), nil
}

func (e *StarlarkEngine) builtinFloat(_ *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, _ []starlark.Tuple) (starlark.Value, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("float: got %d arguments, want 1", len(args))
	}
	switch v := args[0].(type) {
	case starlark.Float:
		return v, nil
	case starlark.Int:
		i, _ := v.Int64()
		return starlark.Float(i), nil
	case starlark.String:
		var f float64
		fmt.Sscanf(string(v), "%f", &f)
		return starlark.Float(f), nil
	}
	return starlark.Float(0), nil
}

func (e *StarlarkEngine) builtinBool(_ *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, _ []starlark.Tuple) (starlark.Value, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("bool: got %d arguments, want 1", len(args))
	}
	return starlark.Bool(args[0].Truth()), nil
}

func (e *StarlarkEngine) builtinSuccess(_ *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, _ []starlark.Tuple) (starlark.Value, error) {
	dict := starlark.StringDict{
		"pass":    starlark.Bool(true),
		"message": starlark.String(""),
	}
	return starlarkstruct.FromStringDict(starlark.String("ValidationResponse"), dict), nil
}

func (e *StarlarkEngine) builtinFail(_ *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, _ []starlark.Tuple) (starlark.Value, error) {
	msg := "Validation failed"
	if len(args) > 0 {
		if s, ok := args[0].(starlark.String); ok {
			msg = string(s)
		}
	}
	dict := starlark.StringDict{
		"pass":    starlark.Bool(false),
		"message": starlark.String(msg),
	}
	return starlarkstruct.FromStringDict(starlark.String("ValidationResponse"), dict), nil
}

// ============================================================================
// EXPRESSION STORAGE
// ============================================================================

// GetExpression fetches an expression by ID
func (e *StarlarkEngine) GetExpression(ctx context.Context, id string) (*Expression, error) {
	var r struct {
		ID               string `db:"id"`
		TenantID         string `db:"tenant_id"`
		BusinessObjectID string `db:"business_object_id"`
		FieldKey         string `db:"field_key"`
		RuleType         string `db:"rule_type"`
		Name             string `db:"name"`
		Description      string `db:"description"`
		Script           string `db:"script"`
		IsActive         bool   `db:"is_active"`
		Version          int    `db:"version"`
	}

	err := e.db.GetContext(ctx, &r, `
		SELECT id, COALESCE(tenant_id::text,'') as tenant_id,
		       COALESCE(business_object_id::text,'') as business_object_id,
		       COALESCE(field_key,'') as field_key, rule_type, name,
		       COALESCE(description,'') as description, script,
		       COALESCE(is_active, false) as is_active,
		       COALESCE(version, 1) as version
		FROM expression_rules WHERE id = $1
	`, id)

	if err != nil {
		return nil, fmt.Errorf("expression not found")
	}

	return &Expression{
		ID:               r.ID,
		TenantID:         r.TenantID,
		BusinessObjectID: r.BusinessObjectID,
		FieldKey:         r.FieldKey,
		RuleType:         ExpressionType(r.RuleType),
		Name:             r.Name,
		Description:      r.Description,
		Script:           r.Script,
		IsActive:         r.IsActive,
		Version:          r.Version,
	}, nil
}

// SaveExpression creates or updates an expression
func (e *StarlarkEngine) SaveExpression(ctx context.Context, expr *Expression) (string, error) {
	if expr.ID == "" {
		expr.ID = uuid.New().String()
	}

	_, err := e.db.ExecContext(ctx, `
		INSERT INTO expression_rules (
			id, tenant_id, business_object_id, field_key, rule_type, name,
			description, script, is_active, version, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6,
			$7, $8, $9, $10, NOW(), NOW()
		)
		ON CONFLICT (id) DO UPDATE SET
			script = EXCLUDED.script,
			name = EXCLUDED.name,
			description = EXCLUDED.description,
			is_active = EXCLUDED.is_active,
			version = expression_rules.version + 1,
			updated_at = NOW()
	`, expr.ID, expr.TenantID, nilIfEmpty(expr.BusinessObjectID), nilIfEmpty(expr.FieldKey),
		string(expr.RuleType), expr.Name, nilIfEmpty(expr.Description),
		expr.Script, expr.IsActive, expr.Version+1)

	if err != nil {
		return "", err
	}
	return expr.ID, nil
}

func nilIfEmpty(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}

// GetExpressionsByBO fetches all expressions for a business object
func (e *StarlarkEngine) GetExpressionsByBO(ctx context.Context, boID string) ([]*Expression, error) {
	type row struct {
		ID       string `db:"id"`
		RuleType string `db:"rule_type"`
		Name     string `db:"name"`
		Script   string `db:"script"`
		FieldKey string `db:"field_key"`
	}

	var rows []row
	err := e.db.SelectContext(ctx, &rows, `
		SELECT id, rule_type, name, script, COALESCE(field_key,'') as field_key
		FROM expression_rules
		WHERE business_object_id = $1 AND is_active = true
	`, boID)

	if err != nil {
		return []*Expression{}, nil
	}

	expressions := make([]*Expression, 0, len(rows))
	for _, r := range rows {
		expressions = append(expressions, &Expression{
			ID:       r.ID,
			RuleType: ExpressionType(r.RuleType),
			Name:     r.Name,
			Script:   r.Script,
			FieldKey: r.FieldKey,
		})
	}
	return expressions, nil
}
