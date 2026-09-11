//go:build js && wasm

// Browser-side WASM build of the rule/calc AST evaluator. This exposes the
// exact same evaluator used server-side (internal/rules/vm.AdvancedEvaluator)
// so a rule authored in the Monaco editor evaluates identically in the
// client-side live-preview panel and on the server - one evaluator, two
// build targets, instead of two separate implementations that can drift.
package main

import (
	"encoding/json"
	"fmt"
	"strings"
	"syscall/js"

	vm "github.com/hondyman/uisce/backend/internal/rules/vm"
)

func main() {
	js.Global().Set("evaluateRule", js.FuncOf(evaluateRule))
	js.Global().Set("traceRule", js.FuncOf(traceRule))
	js.Global().Set("analyzeRuleHealth", js.FuncOf(analyzeRuleHealth))
	js.Global().Set("parseExpression", js.FuncOf(parseExpressionJS))
	js.Global().Set("evaluateExpressionText", js.FuncOf(evaluateExpressionText))
	js.Global().Set("compileExpressionText", js.FuncOf(compileExpressionText))
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

// parseExpressionJS exposes vm.ParseExpression for the editor's live
// syntax-checking - parse-only, no evaluation, so a keystroke can be
// validated without a data context. Returns the parsed AST as JSON (so
// the editor can e.g. list the function names referenced, for
// capability-badge display) alongside a position-bearing error on
// failure - the same ParseError the Go side returns, round-tripped
// through JSON rather than re-derived in JS.
func parseExpressionJS(this js.Value, args []js.Value) interface{} {
	if len(args) != 1 {
		return map[string]interface{}{"error": "expected 1 argument: expression text"}
	}
	expr, err := vm.ParseExpression(args[0].String())
	if err != nil {
		return parseErrorResult(err)
	}
	astJSON, err := json.Marshal(expr)
	if err != nil {
		return map[string]interface{}{"error": err.Error()}
	}
	var astAny interface{}
	_ = json.Unmarshal(astJSON, &astAny)
	return map[string]interface{}{"ast": astAny}
}

// evaluateExpressionText parses expression text and evaluates it against
// a data context in one call - the primitive a "Test with sample data"
// editor panel needs. Text authored via ParseExpression's grammar can be
// either a pure calc-term formula ("SUM(qty * price)", numeric) or a
// validation-rule-shaped comparison ("XIRR(cash_flows, dates) > 0.15",
// boolean) - the editor doesn't know which until it tries, so this tries
// AdvancedEvaluator.EvaluateNumeric first (the entry point calc terms
// use) and falls back to the boolean Evaluate only on the specific
// "didn't evaluate to a number" mismatch, not on every error (a real
// evaluation error - an unresolvable field, a division by zero - should
// surface as-is, not be masked by a second, differently-wrong attempt).
func evaluateExpressionText(this js.Value, args []js.Value) interface{} {
	if len(args) != 2 {
		return map[string]interface{}{"error": "expected 2 arguments: expression text, context"}
	}
	expr, err := vm.ParseExpression(args[0].String())
	if err != nil {
		return parseErrorResult(err)
	}
	var ctx map[string]interface{}
	if err := json.Unmarshal([]byte(args[1].String()), &ctx); err != nil {
		return map[string]interface{}{"error": "invalid context JSON: " + err.Error()}
	}
	ae := vm.NewAdvancedEvaluator()
	node := vm.RuleNode{Type: vm.NodeTypeExpression, Expression: expr}

	numResult, numErr := ae.EvaluateNumeric(node, ctx)
	if numErr == nil {
		return map[string]interface{}{"result": numResult, "resultType": "number"}
	}
	if !strings.Contains(numErr.Error(), "did not evaluate to a number") {
		return map[string]interface{}{"error": numErr.Error()}
	}
	boolResult, boolErr := ae.Evaluate(node, ctx)
	if boolErr != nil {
		return map[string]interface{}{"error": boolErr.Error()}
	}
	return map[string]interface{}{"result": boolResult, "resultType": "boolean"}
}

// compileExpressionText parses expression text and compiles it to SQL
// via vm.CompileToSQL, resolving each field reference through a
// caller-supplied {fieldPath: columnExpr} map - a client-side preview
// (e.g. "does this compile, and to what?") that doesn't need a server
// round trip, using the exact same compiler the backend uses for real
// DDL generation (pre_aggregation_service.go). An unresolvable field
// reference is a hard error here too, same fail-loud discipline as
// every other consumer of this compiler.
func compileExpressionText(this js.Value, args []js.Value) interface{} {
	if len(args) != 2 {
		return map[string]interface{}{"error": "expected 2 arguments: expression text, field map"}
	}
	expr, err := vm.ParseExpression(args[0].String())
	if err != nil {
		return parseErrorResult(err)
	}
	var fieldMap map[string]string
	if err := json.Unmarshal([]byte(args[1].String()), &fieldMap); err != nil {
		return map[string]interface{}{"error": "invalid field map JSON: " + err.Error()}
	}
	resolveColumn := func(path string) (string, error) {
		col, ok := fieldMap[path]
		if !ok {
			return "", fmt.Errorf("no column mapping for field %q", path)
		}
		return col, nil
	}
	sql, err := vm.CompileToSQL(expr, resolveColumn)
	if err != nil {
		return map[string]interface{}{"error": err.Error()}
	}
	return map[string]interface{}{"sql": sql}
}

// parseErrorResult surfaces a *vm.ParseError's position alongside its
// message when available, so the editor can underline the exact
// character instead of just showing a toast.
func parseErrorResult(err error) map[string]interface{} {
	if perr, ok := err.(*vm.ParseError); ok {
		return map[string]interface{}{"error": perr.Message, "pos": perr.Pos}
	}
	return map[string]interface{}{"error": err.Error()}
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
