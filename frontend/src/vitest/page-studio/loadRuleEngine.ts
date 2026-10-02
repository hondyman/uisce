import fs from 'fs';
import path from 'path';

/**
 * Loads the real rule engine (public/rule_engine.wasm, built from
 * backend/internal/rules/vm) for tests: wasmRuntime.ts fetches it, so fetch
 * serves the file. No stub engine - page conditions are tested on the one
 * engine they run on.
 */
export function loadRuleEngine(): void {
  const g = globalThis as Record<string, unknown>;
  const root = path.resolve(__dirname, '../../..');
  // wasm_exec.js defines globalThis.Go.
  new Function(fs.readFileSync(path.join(root, 'public/wasm_exec.js'), 'utf8'))();
  const bytes = fs.readFileSync(path.join(root, 'public/rule_engine.wasm'));
  const prev = g.fetch as ((i: unknown, init?: unknown) => Promise<Response>) | undefined;
  g.fetch = async (input: unknown, init?: unknown) =>
    String(input).endsWith('rule_engine.wasm') ? new Response(bytes, { headers: { 'Content-Type': 'application/wasm' } }) : prev!(input, init);
}
