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