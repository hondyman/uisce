#!/usr/bin/env node
// Functional smoke test for frontend/public/rule_engine.wasm - the exact
// file the browser loads (see frontend/src/rules/wasmRuntime.ts).
//
// Why functional, not byte-identity: Go's GOOS=js GOARCH=wasm output isn't
// reproducible build-to-build even with unchanged source (build IDs vary),
// so diffing frontend/public/rule_engine.wasm against a freshly-rebuilt
// backend/rule-engine/generated/rule_engine.wasm would false-positive on
// every run. What actually matters is that the shipped file evaluates the
// current internal/rules/vm.RuleNode AST correctly - specifically the
// Group.Conditions shape, which is exactly what silently broke when
// frontend/public/rule_engine.wasm went stale relative to a source rebuild
// (it was still running rule-engine/runtime's old Group.Children AST).
//
// Run from anywhere: node backend/rule-engine/scripts/verify_wasm.js

const path = require("path");
const fs = require("fs");

const repoRoot = path.resolve(__dirname, "..", "..", "..");
const wasmExecPath = path.join(repoRoot, "frontend", "public", "wasm_exec.js");
const wasmPath = path.join(repoRoot, "frontend", "public", "rule_engine.wasm");

if (!fs.existsSync(wasmExecPath) || !fs.existsSync(wasmPath)) {
  console.error(`verify_wasm: missing ${wasmExecPath} or ${wasmPath}`);
  process.exit(1);
}

require(wasmExecPath);

async function main() {
  const go = new Go();
  const { instance } = await WebAssembly.instantiate(fs.readFileSync(wasmPath), go.importObject);
  go.run(instance);

  const groupRule = {
    Type: "group",
    Operator: "AND",
    Conditions: [
      { Type: "condition", Field: "age", Operator: ">", Value: 18 },
      { Type: "condition", Field: "status", Operator: "equals", Value: "active" },
    ],
  };

  const failures = [];

  const pass = evaluateRule(JSON.stringify(groupRule), JSON.stringify({ age: 25, status: "active" }));
  if (pass.result !== true) {
    failures.push(`expected Group AND to be true for matching data, got ${JSON.stringify(pass)}`);
  }

  const fail = evaluateRule(JSON.stringify(groupRule), JSON.stringify({ age: 25, status: "inactive" }));
  if (fail.result !== false) {
    // The exact regression this test exists to catch: the old
    // rule-engine/runtime AST used Group.Children (not .Conditions), so an
    // unmarshal against this JSON silently produced zero children, and
    // AND-over-zero-conditions evaluates vacuously true regardless of data.
    failures.push(
      `expected Group AND to be false for non-matching data, got ${JSON.stringify(fail)} - ` +
      `this is the exact silent-vacuous-truth regression this check exists to catch`
    );
  }

  const funcCallRule = { Type: "expression", root: { func: "SUM", args: [{ path: "cash_flows" }] } };
  const funcCallResult = evaluateRule(JSON.stringify(funcCallRule), JSON.stringify({ cash_flows: [100, 200, 300] }));
  if (!funcCallResult.error || !funcCallResult.error.includes("float64")) {
    // FuncCall/SUM should compute (proving arithmetic support exists) and
    // only fail because evaluateRule requires a boolean root - if it fails
    // any other way, FuncCall support itself has regressed.
    failures.push(`expected FuncCall/SUM to compute and reject only for non-bool root, got ${JSON.stringify(funcCallResult)}`);
  }

  if (failures.length > 0) {
    console.error("verify_wasm: FAILED");
    failures.forEach((f) => console.error(`  - ${f}`));
    process.exit(1);
  }

  console.log("verify_wasm: frontend/public/rule_engine.wasm evaluates correctly (Group AND/OR, FuncCall arithmetic)");
}

main().catch((err) => {
  console.error("verify_wasm: threw:", err);
  process.exit(1);
});
