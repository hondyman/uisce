// wasm/engine.go
package wasm

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/hondyman/semlayer/backend/internal/audit"
	"github.com/hondyman/semlayer/backend/internal/validation"
	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
	"github.com/tetratelabs/wazero/imports/wasi_snapshot_preview1"
)

// wazeroEngine implements the Engine interface using wazero runtime
type wazeroEngine struct {
	runtime wazero.Runtime
	module  api.Module
	mu      sync.Mutex
	audit   audit.Logger
	val     *validation.Validator
}

// NewWazeroEngine creates a new WASM engine instance
func NewWazeroEngine(ctx context.Context, wasmBytes []byte, auditLogger audit.Logger, validator *validation.Validator) (Engine, error) {
	r := wazero.NewRuntime(ctx)
	wasi_snapshot_preview1.MustInstantiate(ctx, r)

	// Instantiate the WASM module
	mod, err := r.Instantiate(ctx, wasmBytes)
	if err != nil {
		return nil, fmt.Errorf("instantiate wasm module: %w", err)
	}

	// Verify required exports exist
	requiredExports := []string{
		"alloc",
		"EvaluateComplianceRule",
		"ComputeFactorModel",
		"ComputeVaR",
		"EvaluateScenario",
	}
	for _, name := range requiredExports {
		if mod.ExportedFunction(name) == nil {
			return nil, fmt.Errorf("missing required export: %s", name)
		}
	}

	return &wazeroEngine{
		runtime: r,
		module:  mod,
		audit:   auditLogger,
		val:     validator,
	}, nil
}

func (e *wazeroEngine) Close(ctx context.Context) error {
	return e.runtime.Close(ctx)
}

// callWASM executes a WASM function with JSON inputs/outputs
func (e *wazeroEngine) callWASM(
	ctx context.Context,
	fnName string,
	inputs ...[]byte,
) ([]byte, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	start := time.Now()

	fn := e.module.ExportedFunction(fnName)
	if fn == nil {
		return nil, fmt.Errorf("function %s not found", fnName)
	}

	// Get alloc function for memory management
	alloc := e.module.ExportedFunction("alloc")
	if alloc == nil {
		return nil, fmt.Errorf("alloc function not found")
	}

	// Allocate memory and write inputs
	var args []uint64
	for _, in := range inputs {
		res, err := alloc.Call(ctx, uint64(len(in)))
		if err != nil {
			return nil, fmt.Errorf("alloc memory: %w", err)
		}
		ptr := res[0]
		mem := e.module.Memory()
		if !mem.Write(uint32(ptr), in) {
			return nil, fmt.Errorf("write input to memory failed")
		}
		args = append(args, ptr, uint64(len(in)))
	}

	// Call the function
	out, err := fn.Call(ctx, args...)
	if err != nil {
		return nil, fmt.Errorf("call %s: %w", fnName, err)
	}

	// Read output from WASM memory
	// WASM signature typically returns a single uint64 with packed ptr/len
	if len(out) < 1 {
		return nil, fmt.Errorf("unexpected return format from WASM")
	}
	packed := out[0]
	outPtr := uint32(packed >> 32)
	outLen := uint32(packed)

	mem := e.module.Memory()
	buf, ok := mem.Read(outPtr, outLen)
	if !ok {
		return nil, fmt.Errorf("read output from memory failed")
	}

	// Copy to avoid memory reuse issues
	result := append([]byte(nil), buf...)

	// Audit execution (Whitepaper §9)
	if e.audit != nil {
		tenantID := "SYSTEM"
		if tid, ok := ctx.Value("tenant_id").(string); ok {
			tenantID = tid
		}
		e.audit.Log(ctx, audit.WASMExecution{
			Function:    fnName,
			InputSize:   len(inputs),
			OutputSize:  len(result),
			ExecutionMs: time.Since(start).Milliseconds(),
			TenantID:    tenantID,
		})
	}

	return result, nil
}

// EvaluateComplianceRule implements the Engine interface
func (e *wazeroEngine) EvaluateComplianceRule(
	ctx context.Context,
	rule RuleConfig,
	portfolioCtx ComplianceContext,
) (*ComplianceEvaluationResult, error) {

	// 1. JSON Schema Validation
	// Decode interface logic required for generic jsonschema payload handling
	var payload interface{}
	tempJSON, _ := json.Marshal(portfolioCtx)
	json.Unmarshal(tempJSON, &payload)

	if e.val != nil {
		if err := e.val.Validate("compliance_context.schema.json", payload); err != nil {
			return nil, fmt.Errorf("compliance validation failed: %w", err)
		}
	}
	ruleJSON, err := json.Marshal(rule)
	if err != nil {
		return nil, fmt.Errorf("marshal rule config: %w", err)
	}
	ctxJSON, err := json.Marshal(portfolioCtx)
	if err != nil {
		return nil, fmt.Errorf("marshal compliance context: %w", err)
	}

	resultJSON, err := e.callWASM(ctx, "EvaluateComplianceRule", ruleJSON, ctxJSON)
	if err != nil {
		return nil, fmt.Errorf("WASM execution failed: %w", err)
	}

	var result ComplianceEvaluationResult
	if err := json.Unmarshal(resultJSON, &result); err != nil {
		return nil, fmt.Errorf("unmarshal result: %w", err)
	}

	return &result, nil
}

// ComputeFactorModel implements the Engine interface
func (e *wazeroEngine) ComputeFactorModel(
	ctx context.Context,
	factorCtx FactorModelContext,
) (*FactorModelResult, error) {

	var payload interface{}
	tempJSON, _ := json.Marshal(factorCtx)
	json.Unmarshal(tempJSON, &payload)

	if e.val != nil {
		if err := e.val.Validate("factor_model_context.schema.json", payload); err != nil {
			return nil, fmt.Errorf("factor model validation failed: %w", err)
		}
	}
	ctxJSON, err := json.Marshal(factorCtx)
	if err != nil {
		return nil, fmt.Errorf("marshal factor context: %w", err)
	}

	resultJSON, err := e.callWASM(ctx, "ComputeFactorModel", ctxJSON)
	if err != nil {
		return nil, fmt.Errorf("WASM execution failed: %w", err)
	}

	var result FactorModelResult
	if err := json.Unmarshal(resultJSON, &result); err != nil {
		return nil, fmt.Errorf("unmarshal result: %w", err)
	}

	return &result, nil
}

// ComputeVaR implements the Engine interface
func (e *wazeroEngine) ComputeVaR(
	ctx context.Context,
	varCtx VaRContext,
) (*VaRResult, error) {

	var payload interface{}
	tempJSON, _ := json.Marshal(varCtx)
	json.Unmarshal(tempJSON, &payload)

	if e.val != nil {
		if err := e.val.Validate("var_context.schema.json", payload); err != nil {
			return nil, fmt.Errorf("var validation failed: %w", err)
		}
	}
	ctxJSON, err := json.Marshal(varCtx)
	if err != nil {
		return nil, fmt.Errorf("marshal VaR context: %w", err)
	}

	resultJSON, err := e.callWASM(ctx, "ComputeVaR", ctxJSON)
	if err != nil {
		return nil, fmt.Errorf("WASM execution failed: %w", err)
	}

	var result VaRResult
	if err := json.Unmarshal(resultJSON, &result); err != nil {
		return nil, fmt.Errorf("unmarshal result: %w", err)
	}

	return &result, nil
}

// EvaluateScenario implements the Engine interface
func (e *wazeroEngine) EvaluateScenario(
	ctx context.Context,
	scenarioCtx ScenarioContext,
) (*ScenarioResult, error) {

	var payload interface{}
	tempJSON, _ := json.Marshal(scenarioCtx)
	json.Unmarshal(tempJSON, &payload)

	if e.val != nil {
		if err := e.val.Validate("scenario_context.schema.json", payload); err != nil {
			return nil, fmt.Errorf("scenario validation failed: %w", err)
		}
	}
	ctxJSON, err := json.Marshal(scenarioCtx)
	if err != nil {
		return nil, fmt.Errorf("marshal scenario context: %w", err)
	}

	resultJSON, err := e.callWASM(ctx, "EvaluateScenario", ctxJSON)
	if err != nil {
		return nil, fmt.Errorf("WASM execution failed: %w", err)
	}

	var result ScenarioResult
	if err := json.Unmarshal(resultJSON, &result); err != nil {
		return nil, fmt.Errorf("unmarshal result: %w", err)
	}

	return &result, nil
}
