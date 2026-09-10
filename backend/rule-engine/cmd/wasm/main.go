//go:build js && wasm

// Browser-side WASM build of the rule/calc AST evaluator. This exposes the
// exact same evaluator used server-side (internal/rules/vm.AdvancedEvaluator)
// so a rule authored in the Monaco editor evaluates identically in the
// client-side live-preview panel and on the server - one evaluator, two
// build targets, instead of two separate implementations that can drift.
package main

import (
	"encoding/json"
	"syscall/js"

	vm "github.com/hondyman/uisce/backend/internal/rules/vm"
)

func main() {
	js.Global().Set("evaluateRule", js.FuncOf(evaluateRule))
	js.Global().Set("traceRule", js.FuncOf(traceRule))
	js.Global().Set("analyzeRuleHealth", js.FuncOf(analyzeRuleHealth))
	select {} // keep WASM alive
}

func parseRuleAndCtx(args []js.Value) (vm.RuleNode, map[string]interface{}, error) {
	var rule vm.RuleNode
	if err := json.Unmarshal([]byte(args[0].String()), &rule); err != nil {
		return rule, nil, err
	}
	var ctx map[string]interface{}
	if err := json.Unmarshal([]byte(args[1].String()), &ctx); err != nil {
		return rule, nil, err
	}
	return rule, ctx, nil
}

func evaluateRule(this js.Value, args []js.Value) interface{} {
	if len(args) != 2 {
		return map[string]interface{}{"error": "expected 2 arguments: rule, context"}
	}
	rule, ctx, err := parseRuleAndCtx(args)
	if err != nil {
		return map[string]interface{}{"error": err.Error()}
	}

	ae := vm.NewAdvancedEvaluator()
	result, err := ae.Evaluate(rule, ctx)
	if err != nil {
		return map[string]interface{}{"error": err.Error()}
	}
	return map[string]interface{}{"result": result}
}

func traceRule(this js.Value, args []js.Value) interface{} {
	if len(args) != 2 {
		return map[string]interface{}{"error": "expected 2 arguments: rule, context"}
	}
	rule, ctx, err := parseRuleAndCtx(args)
	if err != nil {
		return map[string]interface{}{"error": err.Error()}
	}

	trace := vm.TraceRule(rule, ctx)
	// syscall/js can only marshal nil/bool/numeric/string/[]interface{}/
	// map[string]interface{} across the JS boundary, not arbitrary structs
	// - round-trip through JSON so []TraceStep survives the call.
	traceJSON, err := json.Marshal(trace)
	if err != nil {
		return map[string]interface{}{"error": err.Error()}
	}
	var traceAny interface{}
	_ = json.Unmarshal(traceJSON, &traceAny)
	return map[string]interface{}{"trace": traceAny}
}

func analyzeRuleHealth(this js.Value, args []js.Value) interface{} {
	if len(args) != 1 {
		return map[string]interface{}{"error": "expected 1 argument: rule"}
	}
	var rule vm.RuleNode
	if err := json.Unmarshal([]byte(args[0].String()), &rule); err != nil {
		return map[string]interface{}{"error": "invalid rule JSON"}
	}

	health := vm.AnalyzeRuleHealth(rule)
	healthJSON, err := json.Marshal(health)
	if err != nil {
		return map[string]interface{}{"error": err.Error()}
	}
	var healthAny interface{}
	_ = json.Unmarshal(healthJSON, &healthAny)
	return map[string]interface{}{"health": healthAny}
}
