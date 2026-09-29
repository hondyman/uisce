const test = require("node:test");
const assert = require("node:assert");
const { ValidationClient } = require("../dist/index.js");

test("ValidationClient initialization and parameter contracts", (t) => {
  const client = new ValidationClient({
    baseUrl: "https://api.uisce.io",
    tenantId: "tenant-ts-999",
    authToken: "jwt-token-abc",
  });

  assert.ok(client, "Client instance created");
  assert.strictEqual(typeof client.evaluateRecord, "function");
  assert.strictEqual(typeof client.evaluateBatch, "function");
  assert.strictEqual(typeof client.verifyChain, "function");
});
