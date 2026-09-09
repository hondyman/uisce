// Wires Monaco's completion/hover surface to asl.monaco.json's real
// `functions` array (backend/rule-engine/cmd/generate-monaco, derived
// directly from internal/rules/vm.Library - the FunctionSpec registry
// every native/WASM/SQL-pushdown consumer already shares). Before this
// file, nothing in the frontend read asl.monaco.json at all - it existed
// only in backend/rule-engine/generated with zero consumers, which is
// exactly what made autocomplete's "keywords hardcoded" complaint keep
// being true no matter how complete the registry itself got.
//
// asl.monaco.json is manually synced to frontend/public/ (same
// established pattern - and landmine - as rule_engine.wasm: a rebuild on
// the backend doesn't automatically reach the browser, see
// docs/unified-rule-engine-handoff.md's "committed != deployed" note).

import type * as Monaco from 'monaco-editor';

export interface AslFunctionMeta {
  name: string;
  signature: string;
  category: string;
  description: string;
  noClosedForm: boolean;
  pushdown: Record<string, boolean>;
}

interface AslMonacoMetadata {
  functions?: AslFunctionMeta[];
}

// The language id every expression-mode Monaco instance in this app
// should use - exported as a plain string constant (not returned from
// the async registration below) so it's available synchronously for a
// component's `language` prop before registration has necessarily
// finished; Monaco queues tokenization/providers until the language
// exists.
export const UISCE_EXPRESSION_LANGUAGE = 'uisce-expression';

let functionsPromise: Promise<AslFunctionMeta[]> | null = null;

// loadAslFunctions fetches and caches asl.monaco.json's function list -
// safe to call repeatedly (every expression editor instance calls it on
// mount); only the first call hits the network.
export function loadAslFunctions(): Promise<AslFunctionMeta[]> {
  if (!functionsPromise) {
    functionsPromise = fetch('/asl.monaco.json')
      .then((res) => {
        if (!res.ok) throw new Error(`asl.monaco.json fetch failed: ${res.status}`);
        return res.json() as Promise<AslMonacoMetadata>;
      })
      .then((data) => data.functions ?? [])
      .catch((err) => {
        console.error('Failed to load asl.monaco.json - function autocomplete will be empty', err);
        return [];
      });
  }
  return functionsPromise;
}

// capabilityBadge renders a function's pushdown/native status as the
// short label the proof bar asks for: "wasm-only" for a function with no
// closed-form SQL expansion (IRR/XIRR/MIRR-shaped), "pushdown" for one
// that compiles to SQL on at least one dialect (SUM/AVG/SUMPRODUCT-
// shaped). Every registered function is native/WASM-capable by
// construction (FunctionSpec.Native is mandatory - see library.go) so
// there is no third "native-only, not even WASM" case to render.
export function capabilityBadge(fn: AslFunctionMeta): 'wasm-only' | 'pushdown' {
  if (fn.noClosedForm) return 'wasm-only';
  const pushdownable = Object.values(fn.pushdown ?? {}).some(Boolean);
  return pushdownable ? 'pushdown' : 'wasm-only';
}

let languageRegistered = false;

// registerUisceExpressionLanguage registers the language, tokenizer,
// completion provider, and hover provider exactly once per page load
// (Monaco's registries are global, not per-editor-instance - calling
// this from every editor's onMount is intentional and idempotent).
export async function registerUisceExpressionLanguage(monaco: typeof Monaco): Promise<void> {
  if (languageRegistered) return;
  languageRegistered = true;

  monaco.languages.register({ id: UISCE_EXPRESSION_LANGUAGE });

  // A minimal tokenizer - enough for readable syntax coloring (function
  // calls, field references, numbers, operators), not a full grammar;
  // vm.ParseExpression (via the WASM parseExpression export) is the
  // actual authority on what's valid, surfaced separately as inline
  // diagnostics, not through this tokenizer.
  monaco.languages.setMonarchTokensProvider(UISCE_EXPRESSION_LANGUAGE, {
    tokenizer: {
      root: [
        [/[A-Za-z_][A-Za-z0-9_]*(?=\s*\()/, 'keyword'],
        [/[A-Za-z_][A-Za-z0-9_.]*/, 'identifier'],
        [/\d+(\.\d+)?/, 'number'],
        [/[()]/, '@brackets'],
        [/[+\-*/,]/, 'operator'],
        [/==|!=|>=|<=|[<>]/, 'operator.comparison'],
      ],
    },
  });

  const functions = await loadAslFunctions();
  const byName = new Map(functions.map((f) => [f.name.toUpperCase(), f]));

  monaco.languages.registerCompletionItemProvider(UISCE_EXPRESSION_LANGUAGE, {
    triggerCharacters: [],
    provideCompletionItems(model, position) {
      const word = model.getWordUntilPosition(position);
      const range: Monaco.IRange = {
        startLineNumber: position.lineNumber,
        endLineNumber: position.lineNumber,
        startColumn: word.startColumn,
        endColumn: word.endColumn,
      };
      const suggestions: Monaco.languages.CompletionItem[] = functions.map((fn) => ({
        label: fn.name,
        kind: monaco.languages.CompletionItemKind.Function,
        detail: `${fn.signature}  ·  ${capabilityBadge(fn)}`,
        documentation: {
          value: `**${fn.name}**  \`${capabilityBadge(fn)}\`\n\n${fn.description}`,
        },
        insertText: `${fn.name}($1)`,
        insertTextRules: monaco.languages.CompletionItemInsertTextRule.InsertAsSnippet,
        range,
      }));
      return { suggestions };
    },
  });

  monaco.languages.registerHoverProvider(UISCE_EXPRESSION_LANGUAGE, {
    provideHover(model, position) {
      const word = model.getWordAtPosition(position);
      if (!word) return null;
      const fn = byName.get(word.word.toUpperCase());
      if (!fn) return null;
      return {
        range: new monaco.Range(position.lineNumber, word.startColumn, position.lineNumber, word.endColumn),
        contents: [
          { value: `**${fn.name}**  \`${capabilityBadge(fn)}\`` },
          { value: '```\n' + fn.signature + '\n```' },
          { value: fn.description },
        ],
      };
    },
  });
}
