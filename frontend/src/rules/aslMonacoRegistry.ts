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

// A semantic term this expression's data context can resolve — sourced from
// GET /validation-rule-nodes/bo-fields (ListSemanticFields), which returns
// the BO's semantic term names (business_object_fields.field_name,
// e.g. "TargetQuantity") paired with their currently-bound physical
// column's data type. These are the vocabulary a rule should be authored
// against, since they're portable across physical binding changes. `entity`
// is the identifier preceding the dot for a dotted field (e.g. "client"
// in "client.risk_score"); absent for flat fields like "ExecQuantity".
export interface AslFieldMeta {
  name: string;
  type: string;
  entity?: string;
  description?: string;
}

// setAslFields updates the live field list every registered completion/
// hover provider reads from - called whenever the authoring page's BO
// selection (and therefore its field list) changes. A plain module-level
// variable, not a React prop threaded into the provider: Monaco's
// providers are registered once per language for the whole page
// (registerUisceExpressionLanguage no-ops after the first call), so the
// provider closure can't capture a fresh `fields` array per BO switch -
// it has to read a mutable source at call time instead.
let currentFields: AslFieldMeta[] = [];

export function setAslFields(fields: AslFieldMeta[]): void {
  currentFields = fields;
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
    // '.' triggers dot-notation completion (a related entity's fields);
    // '(' and ',' re-trigger right where a function's first/next
    // argument goes, since that's exactly where a field reference
    // belongs. Monaco's own word-character triggering (typing "XI",
    // "Exec", ...) fires without being listed here - trigger characters
    // are for characters that AREN'T word constituents.
    triggerCharacters: ['.', '(', ','],
    provideCompletionItems(model, position) {
      const lineUpToCursor = model.getValueInRange({
        startLineNumber: position.lineNumber, startColumn: 1,
        endLineNumber: position.lineNumber, endColumn: position.column,
      });

      // Dot notation: "client." completing to that entity's fields only.
      // Matches a trailing "<ident>." immediately before the cursor
      // (allowing the word already being typed after the dot, which
      // getWordUntilPosition below excludes from the replace range).
      const dotMatch = lineUpToCursor.match(/([A-Za-z_][A-Za-z0-9_]*)\.[A-Za-z0-9_]*$/);
      const entityScope = dotMatch ? dotMatch[1] : null;

      const word = model.getWordUntilPosition(position);
      const range: Monaco.IRange = {
        startLineNumber: position.lineNumber,
        endLineNumber: position.lineNumber,
        startColumn: word.startColumn,
        endColumn: word.endColumn,
      };

      const toFieldItem = (f: AslFieldMeta): Monaco.languages.CompletionItem => ({
        label: { label: f.name, description: f.type },
        kind: monaco.languages.CompletionItemKind.Field,
        detail: 'semantic term',
        documentation: f.description
          ? { value: `**${f.name}**  \`${f.type}\`  semantic term\n\n${f.description}` }
          : { value: `**${f.name}**  \`${f.type}\`  semantic term` },
        insertText: f.name,
        sortText: '0_' + f.name,
        range,
      });

      let fieldSuggestions: Monaco.languages.CompletionItem[] = currentFields
        .filter((f) => (entityScope ? f.entity === entityScope : !f.entity))
        .map(toFieldItem);

      // A dot after an identifier this BO's fields don't recognize as an
      // entity (no cross-entity binding tagged that name) falls back to
      // every field rather than an empty list - more useful than
      // silence, and an empty completion result lets Monaco's own
      // unrelated "Text" suggestion (from whatever last identifier was
      // typed) surface as the only entry, which reads as a wrong answer
      // rather than an honestly-empty one.
      if (entityScope && fieldSuggestions.length === 0) {
        fieldSuggestions = currentFields.map(toFieldItem);
      }

      // Dot-scoped completion only offers fields - a function call
      // doesn't make sense as the right-hand side of a dotted field
      // access.
      if (entityScope) {
        return { suggestions: fieldSuggestions };
      }

      const functionSuggestions: Monaco.languages.CompletionItem[] = functions.map((fn) => ({
        label: fn.name,
        kind: monaco.languages.CompletionItemKind.Function,
        detail: `${fn.signature}  ·  ${capabilityBadge(fn)}`,
        documentation: {
          value: `**${fn.name}**  \`${capabilityBadge(fn)}\`\n\n${fn.description}`,
        },
        insertText: `${fn.name}($1)`,
        insertTextRules: monaco.languages.CompletionItemInsertTextRule.InsertAsSnippet,
        command: { id: 'editor.action.triggerParameterHints', title: 'Trigger Parameter Hints' },
        sortText: '1_' + fn.name,
        range,
      }));

      return { suggestions: [...fieldSuggestions, ...functionSuggestions] };
    },
  });

  monaco.languages.registerHoverProvider(UISCE_EXPRESSION_LANGUAGE, {
    provideHover(model, position) {
      const word = model.getWordAtPosition(position);
      if (!word) return null;
      const range = new monaco.Range(position.lineNumber, word.startColumn, position.lineNumber, word.endColumn);

      const fn = byName.get(word.word.toUpperCase());
      if (fn) {
        return {
          range,
          contents: [
            { value: `**${fn.name}**  \`${capabilityBadge(fn)}\`` },
            { value: '```\n' + fn.signature + '\n```' },
            { value: fn.description },
          ],
        };
      }

      const field = currentFields.find((f) => f.name === word.word);
      if (field) {
        return {
          range,
          contents: [
            { value: `**${field.name}**  \`${field.type}\`` },
            ...(field.description ? [{ value: field.description }] : []),
          ],
        };
      }
      return null;
    },
  });

  // Signature help (parameter hints): shows the active function's full
  // signature while inside its parentheses, bolding the parameter the
  // cursor is currently on - derived by parsing FunctionSpec.Signature's
  // "(a type, b type) -> type" text into individual parameter labels,
  // not a second, hand-maintained parameter list.
  monaco.languages.registerSignatureHelpProvider(UISCE_EXPRESSION_LANGUAGE, {
    signatureHelpTriggerCharacters: ['(', ','],
    signatureHelpRetriggerCharacters: [','],
    provideSignatureHelp(model, position) {
      const lineUpToCursor = model.getValueInRange({
        startLineNumber: position.lineNumber, startColumn: 1,
        endLineNumber: position.lineNumber, endColumn: position.column,
      });
      const call = findEnclosingCall(lineUpToCursor);
      if (!call) return null;
      const fn = byName.get(call.name.toUpperCase());
      if (!fn) return null;

      const params = parseSignatureParams(fn.signature);
      return {
        value: {
          signatures: [{
            label: fn.signature,
            documentation: fn.description,
            parameters: params.map((p) => ({ label: p })),
            activeParameter: Math.min(call.argIndex, Math.max(params.length - 1, 0)),
          }],
          activeSignature: 0,
          activeParameter: Math.min(call.argIndex, Math.max(params.length - 1, 0)),
        },
        dispose() {},
      };
    },
  });
}

// findEnclosingCall walks backward from the cursor through the current
// line's text, tracking paren depth, to find the nearest unclosed
// "funcName(" the cursor sits inside, and which comma-separated argument
// index the cursor is currently in. Returns null outside any call (or
// inside a nested, already-closed one).
function findEnclosingCall(lineUpToCursor: string): { name: string; argIndex: number } | null {
  let depth = 0;
  let argIndex = 0;
  for (let i = lineUpToCursor.length - 1; i >= 0; i--) {
    const c = lineUpToCursor[i];
    if (c === ')') {
      depth++;
    } else if (c === '(') {
      if (depth === 0) {
        const before = lineUpToCursor.slice(0, i);
        const nameMatch = before.match(/([A-Za-z_][A-Za-z0-9_]*)$/);
        if (!nameMatch) return null;
        return { name: nameMatch[1], argIndex };
      }
      depth--;
    } else if (c === ',' && depth === 0) {
      argIndex++;
    }
  }
  return null;
}

// parseSignatureParams extracts individual parameter labels from a
// FunctionSpec.Signature string like "(rate number, cash_flows
// number[]) -> number" -> ["rate number", "cash_flows number[]"] -
// display-only text, parsed from the same signature the hover/
// completion detail already shows rather than a second hand-maintained
// parameter list.
function parseSignatureParams(signature: string): string[] {
  const match = signature.match(/\(([^)]*)\)/);
  if (!match || !match[1].trim()) return [];
  return match[1].split(',').map((p) => p.trim());
}
