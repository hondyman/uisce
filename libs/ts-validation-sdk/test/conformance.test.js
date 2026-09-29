const test = require("node:test");
const assert = require("node:assert");
const fs = require("node:fs");
const path = require("node:path");

// Shared conformance harness reading canonical test cases
const conformanceCasesPath = path.resolve(__dirname, "../../../backend/internal/rules/testdata/conformance_cases.json");

test("Fixture Schema & Integrity Check (HTTP-only Client Mode, WASM local-evaluation deferred to v2)", (t) => {
  assert.ok(fs.existsSync(conformanceCasesPath), `Conformance cases must exist at ${conformanceCasesPath}`);

  const raw = fs.readFileSync(conformanceCasesPath, "utf-8");
  const cases = JSON.parse(raw);

  assert.ok(Array.isArray(cases), "Conformance cases must be a JSON array");
  assert.ok(cases.length >= 7, "Conformance cases must contain at least 7 baseline fixtures");

  for (const tc of cases) {
    assert.ok(tc.id, "Every test case must have an id");
    assert.ok(tc.ast, `Test case ${tc.id} must have an ast`);
    assert.ok(typeof tc.expected === "boolean", `Test case ${tc.id} must have a boolean expected`);
  }
});
