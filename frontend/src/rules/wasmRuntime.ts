// Load wasm_exec.js dynamically since it can't be imported from public
function loadWasmExec(): Promise<void> {
  return new Promise((resolve, reject) => {
    if ((window as any).Go) {
      resolve();
      return;
    }

    const script = document.createElement('script');
    script.src = '/wasm_exec.js';
    script.onload = () => resolve();
    script.onerror = () => reject(new Error('Failed to load wasm_exec.js'));
    document.head.appendChild(script);
  });
}

let wasmReady: Promise<void> | null = null;

function initWasm(): Promise<void> {
  if (wasmReady) return wasmReady;

  wasmReady = loadWasmExec().then(() => {
    // @ts-ignore
    const go = new (window as any).Go();
    return WebAssembly.instantiateStreaming(
      fetch("/rule_engine.wasm"),
      go.importObject
    ).then((result) => {
      go.run(result.instance);
    });
  });

  return wasmReady;
}

export async function evaluateRuleWasm(rule: unknown, ctx: unknown): Promise<boolean> {
  await initWasm();

  if (!(window as any).evaluateRule) {
    throw new Error('WASM runtime not initialized');
  }

  const ruleJson = JSON.stringify(rule);
  const ctxJson = JSON.stringify(ctx);

  const res = (window as any).evaluateRule(ruleJson, ctxJson);
  if (res && typeof res === "object" && "error" in res) {
    throw new Error(res.error);
  }
  return !!res.result;
}

// A position-bearing parse error, surfaced as-is (not just a message) so
// an editor can underline the exact character - the same *vm.ParseError
// the Go side returns, round-tripped through the WASM boundary rather
// than re-derived in JS.
export class ExpressionParseError extends Error {
  pos: number;
  constructor(message: string, pos: number) {
    super(message);
    this.name = 'ExpressionParseError';
    this.pos = pos;
  }
}

function throwIfWasmError(res: any): void {
  if (res && typeof res === 'object' && 'error' in res) {
    if (typeof res.pos === 'number') {
      throw new ExpressionParseError(res.error, res.pos);
    }
    throw new Error(res.error);
  }
}

// parseExpressionWasm validates expression text without evaluating it -
// live syntax checking as the user types, via vm.ParseExpression
// (internal/rules/vm/parser.go) cross-compiled into this same wasm
// build, not a second, JS-side grammar that could drift from it.
export async function parseExpressionWasm(text: string): Promise<unknown> {
  await initWasm();
  if (!(window as any).parseExpression) {
    throw new Error('WASM runtime not initialized');
  }
  const res = (window as any).parseExpression(text);
  throwIfWasmError(res);
  return res.ast;
}

export interface EvaluateExpressionResult {
  result: number | boolean;
  resultType: 'number' | 'boolean';
}

// evaluateExpressionTextWasm parses and evaluates expression text in one
// call, against a data context - resultType tells the caller whether it
// got a calc-term-shaped numeric answer or a rule-shaped boolean one
// (the text alone doesn't say which until it's actually evaluated - see
// evaluateExpressionText's own comment in cmd/wasm/main.go).
export async function evaluateExpressionTextWasm(
  text: string,
  ctx: unknown
): Promise<EvaluateExpressionResult> {
  await initWasm();
  if (!(window as any).evaluateExpressionText) {
    throw new Error('WASM runtime not initialized');
  }
  const res = (window as any).evaluateExpressionText(text, JSON.stringify(ctx));
  throwIfWasmError(res);
  return { result: res.result, resultType: res.resultType };
}

// compileExpressionTextWasm parses expression text and compiles it to
// SQL client-side, resolving each field reference through a caller-
// supplied {fieldPath: columnExpr} map - a live "does this pushdown?"
// preview using the exact same vm.CompileToSQL the backend uses for real
// DDL generation.
export async function compileExpressionTextWasm(
  text: string,
  fieldMap: Record<string, string>
): Promise<string> {
  await initWasm();
  if (!(window as any).compileExpressionText) {
    throw new Error('WASM runtime not initialized');
  }
  const res = (window as any).compileExpressionText(text, JSON.stringify(fieldMap));
  throwIfWasmError(res);
  return res.sql;
}