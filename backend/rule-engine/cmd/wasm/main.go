//go:build js && wasm

package main

import (
	"encoding/json"
	"syscall/js"

	"github.com/hondyman/semlayer/backend/rule-engine/runtime"
)

func main() {
	js.Global().Set("evaluateRule", js.FuncOf(evaluateRule))
	js.Global().Set("traceRule", js.FuncOf(traceRule))
	js.Global().Set("analyzeRuleHealth", js.FuncOf(analyzeRuleHealth))
	select {} // keep WASM alive
}

func evaluateRule(this js.Value, args []js.Value) interface{} {
	if len(args) != 2 {
		return map[string]interface{}{"error": "expected 2 arguments: rule, context"}
	}

	ruleJSON := args[0].String()
	ctxJSON := args[1].String()

	var rule runtime.RuleNode
	if err := json.Unmarshal([]byte(ruleJSON), &rule); err != nil {
		return map[string]interface{}{"error": "invalid rule JSON"}
	}

	var ctx map[string]interface{}
	if err := json.Unmarshal([]byte(ctxJSON), &ctx); err != nil {
		return map[string]interface{}{"error": "invalid context JSON"}
	}

	result := runtime.EvaluateRule(rule, ctx)
	return map[string]interface{}{"result": result}
}

func traceRule(this js.Value, args []js.Value) interface{} {
	if len(args) != 2 {
		return map[string]interface{}{"error": "expected 2 arguments: rule, context"}
	}

	ruleJSON := args[0].String()
	ctxJSON := args[1].String()

	var rule runtime.RuleNode
	if err := json.Unmarshal([]byte(ruleJSON), &rule); err != nil {
		return map[string]interface{}{"error": "invalid rule JSON"}
	}

	var ctx map[string]interface{}
	if err := json.Unmarshal([]byte(ctxJSON), &ctx); err != nil {
		return map[string]interface{}{"error": "invalid context JSON"}
	}

	trace := runtime.TraceRule(rule, ctx)
	return map[string]interface{}{"trace": trace}
}

func analyzeRuleHealth(this js.Value, args []js.Value) interface{} {
	if len(args) != 1 {
		return map[string]interface{}{"error": "expected 1 argument: rule"}
	}

	ruleJSON := args[0].String()

	var rule runtime.RuleNode
	if err := json.Unmarshal([]byte(ruleJSON), &rule); err != nil {
		return map[string]interface{}{"error": "invalid rule JSON"}
	}

	health := runtime.AnalyzeRuleHealth(rule)
	return map[string]interface{}{"health": health}
}
