import { useEffect, useRef } from 'react';
import yaml from 'js-yaml';
import './MonacoCodeEditor.css';

const escapeRegExp = (s: string) => s.replace(/[.*+?^${}()|[\]\\]/g, '\\$&');

// Helper: apply an AST-based edit by parsing the document, applying an updater,
// and returning a full-document replacement edit. This preserves valid JSON/YAML
// structure and produces nicely formatted output rather than piecemeal text
// insertion which can easily break commas/indentation.
  const buildAstReplacement = (model: any, language: string, updater: (obj: any) => void) => {
  try {
    const text = model.getValue();
    let obj: any = null;
    if (language === 'json') obj = JSON.parse(text || '{}');
    else obj = yaml.load(text || '{}') || {};
    if (!obj || typeof obj !== 'object') obj = {};
    updater(obj);
    const out = language === 'json' ? JSON.stringify(obj, null, 2) : yaml.dump(obj);
    const lineCount = model.getLineCount();
    const endCol = model.getLineMaxColumn(lineCount);
    return {
      range: new (window as any).monaco.Range(1, 1, lineCount, endCol),
      text: out,
    };
  } catch (e) {
    // Fallback: no AST edit possible
    return null;
  }
};

// Build a minimal JSON insertion edit: insert a property at top-level before the final '}'
const buildMinimalJsonInsert = (model: any, key: string, valueText: string) => {
  try {
    const txt = model.getValue();
      const lastBrace = txt.lastIndexOf('}');
      if (lastBrace === -1) return null;
      // determine if we need a comma before insertion
      const before = txt.slice(0, lastBrace).trimEnd();
      const needComma = before.endsWith('{') ? false : before.endsWith(',') ? false : true;
      const insertText = (needComma ? ',\n  ' : '\n  ') + `"${key}": ${valueText}` + '\n';
      const lineCount = model.getLineCount();
      const endCol = model.getLineMaxColumn(lineCount);
      // insert before last brace
      const insertPos = lastBrace + 1; // char index based; but Monaco Range uses lines/cols; fallback to whole-doc replace
      // reference insertPos so it's not flagged as unused in some builds
      void insertPos;
      return {
        // fallback to replacing final brace region: last line start..end
        range: new (window as any).monaco.Range(lineCount, 1, lineCount, endCol),
        text: insertText + '\n}',
      };
  } catch (_) { return null; }
};

// Minimal YAML insert: append at end with correct indentation
const buildMinimalYamlInsert = (model: any, key: string, valueText: string) => {
  try {
    const lc = model.getLineCount();
    const ec = model.getLineMaxColumn(lc);
    const insertText = `\n${key}: ${valueText}\n`;
    return { range: new (window as any).monaco.Range(lc, ec, lc, ec), text: insertText };
  } catch (_) { return null; }
};

// Path-aware JSONC minimal edit using jsonc-parser to insert a property while
// preserving formatting. Falls back to the simple insertion above if parser
// isn't available.
const buildJsoncPathInsert = (model: any, path: Array<string | number>, key: string, valueText: string) => {
  try {
    // dynamic require so tests still run without installing deps
  // eslint-disable-next-line @typescript-eslint/no-var-requires
  const { parse, modify, applyEdits, getLocation } = require('jsonc-parser');
  // reference to avoid TS6133 in strict builds when some helpers are unused
  void parse; void getLocation;
    const text = model.getValue();
    // compute edits to set at path
    const edits = modify(text, [...path, key], JSON.parse(valueText), { formattingOptions: { insertSpaces: true, tabSize: 2 } });
    if (!edits || edits.length === 0) return null;
    // jsonc-parser edits are offset-based; map first edit to monaco range
    const first = edits[0];
    const startOffset = first.offset;
    const endOffset = first.offset + first.length;
    // helper to convert offset to (line,col)
    const toPos = (offset: number) => {
      // naive mapping using model's getPositionAt if available, else compute lines
      if (model.getPositionAt) return model.getPositionAt(offset);
      const textUpTo = text.slice(0, offset);
      const lines = textUpTo.split('\n');
      const line = lines.length;
      const col = lines[lines.length - 1].length + 1;
      return { lineNumber: line, column: col };
    };
    const s = toPos(startOffset);
    const e = toPos(endOffset);
    const newText = applyEdits(text, edits);
    const range = new (window as any).monaco.Range(s.lineNumber, s.column, e.lineNumber, e.column);
    return { range, text: newText };
  } catch (e) {
    return null;
  }
};

// Recast fallback: parse / modify / print using recast when jsonc-parser not available.
const buildRecastJsonInsert = (model: any, path: Array<string | number>, key: string, valueText: string) => {
  try {
    // eslint-disable-next-line @typescript-eslint/no-var-requires
    const recast = require('recast');
    const text = model.getValue();
    const ast = recast.parse(text, { parser: require('recast/parsers/typescript') });
    // naive: add property to top-level object if path is empty
    if (!path || path.length === 0) {
  const b = recast.types.builders;
  const root = ast.program.body[0].expression || ast.program.body[0];
  // reference to avoid unused-local errors in minimal test environments
  void b; void root;
      // This is a simplistic approach; if parsing fails just return null
      // Fallback to full document replacement with JSON stringify
      const obj = JSON.parse(text || '{}');
      obj[key] = JSON.parse(valueText);
      const out = JSON.stringify(obj, null, 2);
      const lc = model.getLineCount();
      const ec = model.getLineMaxColumn(lc);
      return { range: new (window as any).monaco.Range(1, 1, lc, ec), text: out };
    }
    return null;
  } catch (e) {
    return null;
  }
};

// Exported helper used by tests: compute deterministic quick-fix actions from
// marker codes (preferred) or message text (fallback). Returns an array of
// action descriptors with title and an updater function or raw text replacement.
export const computeQuickFixActions = (markers: any[], modelValue: string, language: string) => {
  // reference modelValue in case callers compile without using it directly
  void modelValue;
  const actions: any[] = [];
  for (const marker of markers || []) {
    // attach originating marker so consumers can map edits back to diagnostics
    const srcMarker = marker;
    const code = (marker as any).code || (marker as any).meta || undefined;
    const msg = String((marker as any).message || '').toLowerCase();

    // Deterministic mapping by code when present
    if (code === 'MISSING_DATASOURCE' || code === 'ERR_DATASOURCE_MISSING' || /missing_datasource/i.test(String(code || ''))) {
      // derive path/key from marker.source when available to target nested locations
      const derive = (mk: any) => {
        const source = mk && (mk.source || mk.element_id || mk.elementId || mk.code || undefined);
        if (source && typeof source === 'string' && (source.includes('.') || source.includes('/'))) {
          const parts = source.includes('/') ? source.split('/').filter(Boolean) : source.split('.').filter(Boolean);
          if (parts.length > 1) return { path: parts.slice(0, -1), key: parts[parts.length - 1] };
        }
        return { path: [], key: 'tenant_instance_id' };
      };
      const derived = derive(srcMarker);
      actions.push({
        title: 'Insert tenant_instance_id (AST)',
        kind: 'quickfix',
        marker: srcMarker,
        // minimal insert descriptor for provider to create targeted edit
        insert: { path: derived.path, key: derived.key, valueText: '"<YOUR_DATASOURCE>"' },
        updater: (_model: any) => ({ kind: 'ast', apply: (m: any) => buildAstReplacement(m, language || 'json', (obj: any) => { obj.tenant_instance_id = obj.tenant_instance_id || '<YOUR_DATASOURCE>'; }) }),
      });
      continue;
    }

    if (code === 'MISSING_JOIN' || /missing_join/i.test(String(code || ''))) {
      const source = srcMarker && (srcMarker.source || srcMarker.element_id || srcMarker.elementId || undefined);
      const path = (source && typeof source === 'string' && (source.includes('.') || source.includes('/'))) ? (source.includes('/') ? source.split('/').filter(Boolean).slice(0, -1) : source.split('.').filter(Boolean).slice(0, -1)) : [];
      actions.push({
        title: 'Scaffold join (AST)',
        kind: 'quickfix',
        marker: srcMarker,
        insert: { path, key: 'joins', valueText: '[ { "name": "<other_cube>", "sql_on": "${CUBE}.id = ${other_cube}.id" } ]' },
        updater: (_model: any) => ({ kind: 'ast', apply: (m: any) => buildAstReplacement(m, language || 'json', (obj: any) => { obj.joins = obj.joins || [{ name: '<other_cube>', sql_on: '${CUBE}.id = ${other_cube}.id' }]; }) }),
      });
      continue;
    }

    if (code === 'INVALID_MEASURE' || /invalid_measure/i.test(String(code || '')) || /invalid_measure/.test(msg)) {
      actions.push({ title: 'Rename measure (suggestion)', kind: 'quickfix', updater: null });
      continue;
    }

    if (code === 'MISSING_PRE_AGG' || /pre_?aggregation_missing/i.test(String(code || '')) || /pre[_ -]?aggregation/.test(msg)) {
      const source = srcMarker && (srcMarker.source || srcMarker.element_id || srcMarker.elementId || undefined);
      const path = (source && typeof source === 'string' && (source.includes('.') || source.includes('/'))) ? (source.includes('/') ? source.split('/').filter(Boolean).slice(0, -1) : source.split('.').filter(Boolean).slice(0, -1)) : [];
      actions.push({
        title: 'Create pre_aggregation (AST)',
        kind: 'quickfix',
        marker: srcMarker,
        insert: { path, key: 'pre_aggregations', valueText: '[ { "name": "<agg_name>", "type": "rollup", "time_dimension": "<time>", "dimensions": [] } ]' },
        updater: (_model: any) => ({ kind: 'ast', apply: (m: any) => buildAstReplacement(m, language || 'json', (obj: any) => { obj.pre_aggregations = obj.pre_aggregations || [{ name: '<agg_name>', type: 'rollup', time_dimension: '<time>', dimensions: [] }]; }) }),
      });
      continue;
    }

    // Fallback heuristics (existing behavior) based on message text
    if (/datasource/.test(msg) || /tenant_instance_id/.test(msg)) {
      actions.push({ title: 'Insert tenant_instance_id placeholder', kind: 'quickfix', marker: srcMarker, updater: null, rawText: '"tenant_instance_id": "<YOUR_DATASOURCE>",' });
    } else if (/join/.test(msg) && /missing|undefined|not found/.test(msg)) {
      actions.push({ title: 'Scaffold join placeholder', kind: 'quickfix', marker: srcMarker, updater: null, rawText: '"joins": [\n  { "name": "<other_cube>", "sql_on": "${CUBE}.id = ${other_cube}.id" }\n],' });
    } else if (/(measure|dimension)/.test(msg) && /invalid|unknown|not found/.test(msg)) {
      actions.push({ title: 'Rename to <suggestion>', kind: 'quickfix', marker: srcMarker, updater: null });
    } else if (/pre[_ -]?aggregation/.test(msg) && /missing|not found/.test(msg)) {
      actions.push({ title: 'Create pre_aggregation placeholder', kind: 'quickfix', marker: srcMarker, updater: null, rawText: '\n"pre_aggregations": [\n  { "name": "<agg_name>", "type": "rollup", "time_dimension": "<time>", "dimensions": [] }\n],' });
    }
  }
  return actions;
};

// Convert actions from computeQuickFixActions into Monaco-style CodeAction objects
export const convertActionsToMonacoEdits = (actions: any[], model: any, monaco: any, language: string) => {
  const out: any[] = [];
  for (const act of actions || []) {
    try {
      // Prefer minimal in-place insert when an insert descriptor is present
      if (act.insert && typeof act.insert === 'object') {
        try {
          let replacement: any = null;
          if ((language || '').toLowerCase() === 'json') {
            // prefer insert.path when supplied so we can target nested objects
            const path = Array.isArray(act.insert.path) ? act.insert.path : [];
            // try jsonc path-aware insert first
            replacement = buildJsoncPathInsert(model, path, act.insert.key, act.insert.valueText) || buildMinimalJsonInsert(model, act.insert.key, act.insert.valueText) || buildRecastJsonInsert(model, path, act.insert.key, act.insert.valueText);
          } else {
            replacement = buildMinimalYamlInsert(model, act.insert.key, act.insert.valueText);
          }
          if (replacement && replacement.range && typeof replacement.text === 'string') {
            out.push({ title: act.title, edit: { edits: [{ resource: model.uri, edit: { range: replacement.range, text: replacement.text } }] }, kind: act.kind, diagnostics: act.marker ? [act.marker] : [] });
            continue;
          }
        } catch (_) {
          // fall through to other strategies
        }
      }
      if (act.updater && typeof act.updater === 'function') {
        const upd = act.updater(model);
        if (upd && upd.kind === 'ast' && typeof upd.apply === 'function') {
          const replacement = upd.apply(model);
          if (replacement && replacement.range && typeof replacement.text === 'string') {
            out.push({ title: act.title, edit: { edits: [{ resource: model.uri, edit: { range: replacement.range, text: replacement.text } }] }, kind: act.kind, diagnostics: act.marker ? [act.marker] : [] });
            continue;
          }
        }
      }

      // Fallback: rawText insertion using marker range if available
      const marker = act.marker;
      let range = null;
      if (marker && typeof marker.startLineNumber === 'number') {
        range = new monaco.Range(marker.startLineNumber, marker.startColumn || 1, marker.endLineNumber || marker.startLineNumber, marker.endColumn || (marker.startColumn || 1));
      } else {
        // append at end
        const lc = model.getLineCount();
        const ec = model.getLineMaxColumn(lc);
        range = new monaco.Range(lc, ec, lc, ec);
      }
      const text = act.rawText || act.replacementText || '';
      out.push({ title: act.title, edit: { edits: [{ resource: model.uri, edit: { range, text } }] }, kind: act.kind, diagnostics: marker ? [marker] : [] });
    } catch (_) {
      // ignore conversion error for a single action
    }
  }
  return out;
};

interface Marker {
  startLineNumber: number;
  startColumn: number;
  endLineNumber: number;
  endColumn: number;
  message: string;
  severity?: number;
  // optional backend-provided metadata
  code?: string | number;
  code_id?: string | number;
  element_id?: string;
  elementId?: string;
  source?: string;
}

interface CompletionItemLike {
  label: string;
  insertText: string;
  kind?: number;
}

interface MonacoCodeEditorProps {
  value: string;
  language: 'json' | 'yaml' | 'python' | null | undefined;
  readOnly?: boolean;
  onChange?: (val: string) => void;
  markers?: Marker[];
  onMount?: (api: {
    setMarkers: (markers: Marker[]) => void;
    getValue: () => string;
    revealLine?: (line: number, behavior?: 'center' | 'top') => void;
    revealRange?: (startLine: number, endLine?: number) => void;
    focus?: () => void;
    findAndReveal?: (term: string) => number;
  }) => void;
  dynamicCompletions?: CompletionItemLike[];
  /** optional search term to highlight inside the editor (plain text) */
  highlight?: string;
}

// Lightweight prototype that dynamically loads monaco-editor if available and
// wires a basic model + diagnostics (markers). If monaco isn't present the
// component simply renders a message so callers can fall back.
const MonacoCodeEditor: React.FC<MonacoCodeEditorProps> = ({ value, language, readOnly, onChange, markers, onMount, dynamicCompletions, highlight }) => {
  const containerRef = useRef<HTMLDivElement | null>(null);
  const editorRef = useRef<any>(null);
  const modelRef = useRef<any>(null);
  const monacoRef = useRef<any>(null);
  const decorationsRef = useRef<string[]>([]);
  const disposeRefs = useRef<{ action?: any; jsonCompletion?: any; yamlCompletion?: any; pythonCompletion?: any } | null>(null);
  const dynCompletionsRef = useRef<CompletionItemLike[] | undefined>(undefined);
  dynCompletionsRef.current = dynamicCompletions;

  useEffect(() => {
    let cancelled = false;
    (async () => {
      try {
        // dynamic import of the ESM entrypoint for monaco which Vite can resolve
  const monaco = await import('monaco-editor/esm/vs/editor/editor.api');
  // Ensure JSON and YAML languages are registered for syntax highlighting when running in a browser
  if (typeof window !== 'undefined') {
    try { await import('monaco-editor/esm/vs/language/json/monaco.contribution'); } catch (_) {}
    try { await import('monaco-editor/esm/vs/basic-languages/yaml/yaml.contribution'); } catch (_) {}
    try { await import('monaco-editor/esm/vs/basic-languages/python/python.contribution'); } catch (_) {}
  }
  monacoRef.current = monaco;
        if (cancelled) return;
        if (!containerRef.current) return;
        // Try to register local JSON schema for cube semantic models
        try {
          const schemaUrl = '/src/schema/cube-semantic.json';
          // reference schemaUrl to avoid unused-local errors in some bundlers
          void schemaUrl;
          // fetch schema content via import (static file in repo)
          // When running in tests we can require it directly
          // eslint-disable-next-line @typescript-eslint/no-var-requires
          const schema = require('../../schema/cube-semantic.json');
          if ((monaco as any).languages && (monaco as any).languages.json && (monaco as any).languages.json.jsonDefaults) {
            (monaco as any).languages.json.jsonDefaults.setDiagnosticsOptions({
              validate: true,
              schemas: [
                {
                  uri: 'inmemory://schema/cube-semantic.json',
                  fileMatch: ['*'],
                  schema: schema,
                },
              ],
            });
          }
          // YAML validation via monaco-yaml is not required for syntax highlighting;
          // we rely on basic-languages for highlighting in this setup.
        } catch (_) {}
        const el = containerRef.current;
        const mode = language === 'json' ? 'json' : language === 'yaml' ? 'yaml' : 'python';
        modelRef.current = monaco.editor.createModel(value || '', mode);
        editorRef.current = monaco.editor.create(el, {
          model: modelRef.current,
          readOnly: !!readOnly,
          minimap: { enabled: false },
          automaticLayout: true,
          glyphMargin: true,
        });
        if (onChange) {
          editorRef.current.onDidChangeModelContent(() => {
            try { onChange(modelRef.current.getValue()); } catch (_) {}
          });
        }
        if (onMount) {
          onMount({
            setMarkers: (mk: Marker[]) => {
              try {
                const ms = (mk || []).map((m) => ({
                  startLineNumber: m.startLineNumber,
                  startColumn: m.startColumn,
                  endLineNumber: m.endLineNumber,
                  endColumn: m.endColumn,
                  message: m.message,
                  severity: (monaco as any).MarkerSeverity.Error,
                  // preserve code and element id if provided by backend
                  code: m.code !== undefined && m.code !== null ? String(m.code) : (m.code_id !== undefined && m.code_id !== null ? String(m.code_id) : undefined),
                  source: m.element_id || m.elementId || undefined,
                }));
                // setModelMarkers will expose 'code' and 'source' on returned markers
                monaco.editor.setModelMarkers(modelRef.current, 'semlayer', ms);
              } catch (_) {}
            },
            getValue: () => modelRef.current?.getValue?.() || '',
            revealLine: (line: number, behavior?: 'center' | 'top') => {
              try {
                const ed = editorRef.current;
                if (!ed || typeof line !== 'number' || line <= 0) return;
                if (behavior === 'top' && ed.revealLineNearTop) ed.revealLineNearTop(line);
                else if (ed.revealLineInCenter) ed.revealLineInCenter(line);
                else if (ed.revealLine) ed.revealLine(line);
              } catch (_) {}
            },
      revealRange: (startLine: number, endLine?: number) => {
              try {
                const ed = editorRef.current;
                const mon = monacoRef.current || (window as any).monaco;
                if (!ed || !mon || !startLine) return;
                const range = new mon.Range(startLine, 1, (endLine || startLine), 1);
        // Prefer centering the target line in view
        if (ed.revealRangeInCenter) ed.revealRangeInCenter(range);
        else if (ed.revealRangeNearTop) ed.revealRangeNearTop(range);
        else if (ed.revealRange) ed.revealRange(range);
              } catch (_) {}
            },
            focus: () => { try { editorRef.current?.focus?.(); } catch {} },
            // find text using Monaco model search and reveal the first match
            findAndReveal: (term: string) => {
              try {
                if (!term || !modelRef.current || !editorRef.current) return 0;
                const model = modelRef.current;
                const mon = monacoRef.current || (window as any).monaco;
                if (!mon) return 0;
                // use Monaco's findMatches on the model for accurate ranges (case-insensitive)
                const matches = (model.findMatches && typeof model.findMatches === 'function')
                  ? model.findMatches(term, false /* searchOnlyEditableRange */, true /* isRegex */, false /* matchCase */, null /* wordSeparators */, true /* captureMatches */)
                  : [];
                if (!matches || matches.length === 0) return 0;
                const first = matches[0];
                const range = first.range || first;
                if (editorRef.current.revealRangeInCenter) editorRef.current.revealRangeInCenter(range);
                else if (editorRef.current.revealRangeNearTop) editorRef.current.revealRangeNearTop(range);
                else if (editorRef.current.revealRange) editorRef.current.revealRange(range);
                // place a selection/cursor at the start of the match for visibility
                try { editorRef.current.setSelection(range); editorRef.current.focus(); } catch (_) {}
                return matches.length;
              } catch (_) {
                return 0;
              }
            },
          });
        }
        // Helper: apply an AST-based edit by parsing the document, applying an updater,
        // and returning a full-document replacement edit. This preserves valid JSON/YAML
        // structure and produces nicely formatted output rather than piecemeal text
        // insertion which can easily break commas/indentation.
  const buildAstReplacement = (model: any, language: string, updater: (obj: any) => void) => {
          try {
            const text = model.getValue();
            let obj: any = null;
            if (language === 'json') obj = JSON.parse(text || '{}');
            else obj = yaml.load(text || '{}') || {};
            if (!obj || typeof obj !== 'object') obj = {};
            updater(obj);
            const out = language === 'json' ? JSON.stringify(obj, null, 2) : yaml.dump(obj);
            const lineCount = model.getLineCount();
            const endCol = model.getLineMaxColumn(lineCount);
            const range = new (window as any).monaco.Range(1, 1, lineCount, endCol);
            // reference range to silence unused-local when running tests that don't use replacements
            void range;
            return {
              range,
              text: out,
            };
          } catch (e) {
            // Fallback: no AST edit possible
            return null;
          }
        };

        // (computeQuickFixActions is defined at module scope and used here)

        // register a simple CodeAction provider for quick fixes (e.g., missing datasource)
        try {
          const lang = language === 'json' ? 'json' : 'yaml';
          const provider = (monaco as any).languages.registerCodeActionProvider(lang, {
            provideCodeActions: (model: any, range: any, context: any) => {
              // monaco.editor.getModelMarkers returns markers created on the model
              const monacoMarkers = monaco.editor.getModelMarkers({ resource: model.uri }) || [];
              // context.markers contains the diagnostics passed into the provider by Monaco
              const ctxMarkers = (context && (context.markers || context.diagnostics)) || [];
              // merge: prefer explicit source/element id from provider diagnostics when present
              const merged = monacoMarkers.map((m: any) => {
                // try to find a matching ctx marker by position+message
                const match = ctxMarkers.find((c: any) => {
                  try {
                    return (c.startLineNumber === m.startLineNumber && c.startColumn === m.startColumn && String(c.message || '').trim() === String(m.message || '').trim());
                  } catch (e) { return false; }
                }) || {};
                return { ...m, // monaco marker fields
                  // prefer explicit source from provider diagnostic
                  source: (match && (match.source || match.element_id || match.elementId)) || m.source || match.code || m.code,
                  // preserve code field as string where present
                  code: (match && match.code) || m.code,
                };
              });
              // compute deterministic actions using codes where available and prefer marker.source
              const computed = computeQuickFixActions(merged, model.getValue(), language === 'json' ? 'json' : 'yaml');
              // reference provided range to avoid TS6133 in some monaco builds where it isn't used
              void range;
              // convert to Monaco edits using AST replacements when possible
              const converted = convertActionsToMonacoEdits(computed, model, monaco, language === 'json' ? 'json' : 'yaml');
              return { actions: converted || [], dispose: () => {} };
            }
          });
          // reference function so it's not flagged when provider isn't used in minimal test bundles
          void buildAstReplacement;
          // small set of completions/snippets for semantic layer keys
          try {
              const completionProviderJson = (monaco as any).languages.registerCompletionItemProvider('json', {
              provideCompletionItems: (_model: any, _position: any) => {
                const suggestions = [
                  { label: 'tenant_instance_id', kind: (monaco as any).languages.CompletionItemKind.Property, insertText: '"tenant_instance_id": "${1:<DATASOURCE>}"', insertTextRules: (monaco as any).languages.CompletionItemInsertTextRule.InsertAsSnippet },
                  { label: 'joins', kind: (monaco as any).languages.CompletionItemKind.Property, insertText: '"joins": [\n  { "name": "${1:<other_cube>}", "sql_on": "${CUBE}.id = ${other_cube}.id" }\n]', insertTextRules: (monaco as any).languages.CompletionItemInsertTextRule.InsertAsSnippet },
                  { label: 'pre_aggregations', kind: (monaco as any).languages.CompletionItemKind.Property, insertText: '"pre_aggregations": [\n  { "name": "${1:<agg_name>}", "type": "rollup", "time_dimension": "${2:<time>}", "dimensions": [] }\n]', insertTextRules: (monaco as any).languages.CompletionItemInsertTextRule.InsertAsSnippet },
                ];
                return { suggestions };
              }
            });
              const completionProviderYaml = (monaco as any).languages.registerCompletionItemProvider('yaml', {
              provideCompletionItems: (_model: any, _position: any) => {
                const suggestions = [
                  { label: 'tenant_instance_id', kind: (monaco as any).languages.CompletionItemKind.Property, insertText: 'tenant_instance_id: ${1:<DATASOURCE>}', insertTextRules: (monaco as any).languages.CompletionItemInsertTextRule.InsertAsSnippet },
                  { label: 'joins', kind: (monaco as any).languages.CompletionItemKind.Property, insertText: 'joins:\n  - name: ${1:<other_cube>}\n    sql_on: "${CUBE}.id = ${other_cube}.id"', insertTextRules: (monaco as any).languages.CompletionItemInsertTextRule.InsertAsSnippet },
                  { label: 'pre_aggregations', kind: (monaco as any).languages.CompletionItemKind.Property, insertText: 'pre_aggregations:\n  - name: ${1:<agg_name>}\n    type: rollup\n    time_dimension: ${2:<time>}\n    dimensions: []', insertTextRules: (monaco as any).languages.CompletionItemInsertTextRule.InsertAsSnippet },
                ];
                return { suggestions };
              }
            });
            (editorRef.current as any).__semlayer_completionProviders = [completionProviderJson, completionProviderYaml];
          } catch (_) {}
          // store provider on editorRef so we can dispose later
          (editorRef.current as any).__semlayer_codeActionProvider = provider;
          disposeRefs.current = { ...(disposeRefs.current || {}), action: provider };
        } catch (_) {}
        // register baseline semantic completions plus dynamic ones
        try {
          const mkSuggestions = (lang: 'json'|'yaml') => {
            const base = [
              lang === 'json'
                ? { label: 'tenant_instance_id', kind: (monaco as any).languages.CompletionItemKind.Property, insertText: '"tenant_instance_id": "${1:<DATASOURCE>}"', insertTextRules: (monaco as any).languages.CompletionItemInsertTextRule.InsertAsSnippet }
                : { label: 'tenant_instance_id', kind: (monaco as any).languages.CompletionItemKind.Property, insertText: 'tenant_instance_id: ${1:<DATASOURCE>}', insertTextRules: (monaco as any).languages.CompletionItemInsertTextRule.InsertAsSnippet },
              lang === 'json'
                ? { label: 'joins', kind: (monaco as any).languages.CompletionItemKind.Property, insertText: '"joins": [\n  { "name": "${1:<other_cube>}", "sql_on": "${CUBE}.id = ${other_cube}.id" }\n]', insertTextRules: (monaco as any).languages.CompletionItemInsertTextRule.InsertAsSnippet }
                : { label: 'joins', kind: (monaco as any).languages.CompletionItemKind.Property, insertText: 'joins:\n  - name: ${1:<other_cube>}\n    sql_on: "${CUBE}.id = ${other_cube}.id"', insertTextRules: (monaco as any).languages.CompletionItemInsertTextRule.InsertAsSnippet },
              lang === 'json'
                ? { label: 'pre_aggregations', kind: (monaco as any).languages.CompletionItemKind.Property, insertText: '"pre_aggregations": [\n  { "name": "${1:<agg_name>}", "type": "rollup", "time_dimension": "${2:<time>}", "dimensions": [] }\n]', insertTextRules: (monaco as any).languages.CompletionItemInsertTextRule.InsertAsSnippet }
                : { label: 'pre_aggregations', kind: (monaco as any).languages.CompletionItemKind.Property, insertText: 'pre_aggregations:\n  - name: ${1:<agg_name>}\n    type: rollup\n    time_dimension: ${2:<time>}\n    dimensions: []', insertTextRules: (monaco as any).languages.CompletionItemInsertTextRule.InsertAsSnippet },
              // Dimension skeleton
              lang === 'json'
                ? { label: 'dimension_skeleton', kind: (monaco as any).languages.CompletionItemKind.Snippet, insertText: '\"dimensions\": {\n  \"${1:dim_name}\": { \n    \"sql\": \"${TABLE}.${2:column}\", \n    \"type\": \"${3:string}\", \n    \"description\": \"\" \n  }\n}', insertTextRules: (monaco as any).languages.CompletionItemInsertTextRule.InsertAsSnippet }
                : { label: 'dimension_skeleton', kind: (monaco as any).languages.CompletionItemKind.Snippet, insertText: 'dimensions:\n  ${1:dim_name}:\n    sql: \"${TABLE}.${2:column}\"\n    type: ${3:string}\n    description: \"\"', insertTextRules: (monaco as any).languages.CompletionItemInsertTextRule.InsertAsSnippet },
              // Measure skeleton
              lang === 'json'
                ? { label: 'measure_skeleton', kind: (monaco as any).languages.CompletionItemKind.Snippet, insertText: '\"measures\": {\n  \"${1:measure_name}\": {\n    \"sql\": \"${TABLE}.${2:column}\",\n    \"type\": \"${3:sum}\",\n    \"description\": \"\"\n  }\n}', insertTextRules: (monaco as any).languages.CompletionItemInsertTextRule.InsertAsSnippet }
                : { label: 'measure_skeleton', kind: (monaco as any).languages.CompletionItemKind.Snippet, insertText: 'measures:\n  ${1:measure_name}:\n    sql: \"${TABLE}.${2:column}\"\n    type: ${3:sum}\n    description: \"\"', insertTextRules: (monaco as any).languages.CompletionItemInsertTextRule.InsertAsSnippet },
              // Pre-aggregation originalSql variant
              lang === 'json'
                ? { label: 'pre_aggregation_originalSql', kind: (monaco as any).languages.CompletionItemKind.Snippet, insertText: '\"pre_aggregations\": [\n  { \"name\": \"${1:raw}\", \"type\": \"originalSql\" }\n]', insertTextRules: (monaco as any).languages.CompletionItemInsertTextRule.InsertAsSnippet }
                : { label: 'pre_aggregation_originalSql', kind: (monaco as any).languages.CompletionItemKind.Snippet, insertText: 'pre_aggregations:\n  - name: ${1:raw}\n    type: originalSql', insertTextRules: (monaco as any).languages.CompletionItemInsertTextRule.InsertAsSnippet },
            ];
            const extra = (dynCompletionsRef.current || []).map((c) => ({ label: c.label, insertText: c.insertText, kind: c.kind || (monaco as any).languages.CompletionItemKind.Text }));
            return [...base, ...extra];
          };
          const jsonProv = (monaco as any).languages.registerCompletionItemProvider('json', {
            provideCompletionItems: () => ({ suggestions: mkSuggestions('json') })
          });
          const yamlProv = (monaco as any).languages.registerCompletionItemProvider('yaml', {
            provideCompletionItems: () => ({ suggestions: mkSuggestions('yaml') })
          });
          disposeRefs.current = { ...(disposeRefs.current || {}), jsonCompletion: jsonProv, yamlCompletion: yamlProv };
        } catch (_) {}
      } catch (err) {
        // monaco not available; consumer should fallback to Prism implementation.
      }
    })();

    return () => {
      cancelled = true;
      try {
        if (editorRef.current) editorRef.current.dispose();
        if (modelRef.current) modelRef.current.dispose();
      } catch (_) {}
    };
  }, []);

  // sync value & markers
  useEffect(() => {
    try {
      const monaco = monacoRef.current || (window as any).monaco;
      if (!monaco || !modelRef.current) return;
      const current = modelRef.current.getValue();
      if (value !== undefined && value !== current) modelRef.current.setValue(value);
      if (markers && Array.isArray(markers)) {
        // Convert markers to monaco format
        const m = markers.map((mk) => ({
          startLineNumber: mk.startLineNumber,
          startColumn: mk.startColumn,
          endLineNumber: mk.endLineNumber,
          endColumn: mk.endColumn,
          message: mk.message,
          severity: mk.severity ?? monaco.MarkerSeverity.Error,
        }));
        monaco.editor.setModelMarkers(modelRef.current, 'semlayer', m);
      }
      // switch language dynamically when prop changes
      if (language) {
        const langId = language === 'json' ? 'json' : language === 'yaml' ? 'yaml' : 'python';
        try { monaco.editor.setModelLanguage(modelRef.current, langId); } catch (_) {}
      }
    } catch (_) {}
  }, [value, markers, language]);

  // apply highlight decorations when `highlight` changes
  useEffect(() => {
    try {
      const monaco = monacoRef.current || (window as any).monaco;
      const model = modelRef.current;
      const ed = editorRef.current;
      if (!monaco || !model || !ed) return;

      // clear previous decorations when no highlight provided
      if (!highlight || !String(highlight).trim()) {
        try {
          decorationsRef.current = ed.deltaDecorations(decorationsRef.current || [], []);
        } catch (_) {}
        return;
      }

      const term = String(highlight);
      const regex = new RegExp(escapeRegExp(term), 'gi');
      const text = model.getValue() || '';
      const newDecs: any[] = [];
      let m: RegExpExecArray | null = null;
      while ((m = regex.exec(text)) !== null) {
        const startOffset = m.index;
        const endOffset = m.index + m[0].length;
        const startPos = model.getPositionAt(startOffset);
        const endPos = model.getPositionAt(endOffset);
        newDecs.push({
          range: new monaco.Range(startPos.lineNumber, startPos.column, endPos.lineNumber, endPos.column),
          options: { inlineClassName: 'semlayer-inline-highlight' },
        });
        if (m.index === regex.lastIndex) regex.lastIndex++;
      }

      try {
        decorationsRef.current = ed.deltaDecorations(decorationsRef.current || [], newDecs);
      } catch (_) {}
    } catch (_) {}
  }, [highlight, value]);

  // re-register completion providers when dynamicCompletions change
  useEffect(() => {
    try {
      const monaco = monacoRef.current || (window as any).monaco;
      if (!monaco || !editorRef.current) return;
      // dispose existing and re-register to pick up new dynamic completions
      try { disposeRefs.current?.jsonCompletion?.dispose?.(); } catch {}
      try { disposeRefs.current?.yamlCompletion?.dispose?.(); } catch {}
      const mkSuggestions = (lang: 'json'|'yaml') => {
        const base = [
          lang === 'json'
            ? { label: 'tenant_instance_id', kind: (monaco as any).languages.CompletionItemKind.Property, insertText: '"tenant_instance_id": "${1:<DATASOURCE>}"', insertTextRules: (monaco as any).languages.CompletionItemInsertTextRule.InsertAsSnippet }
            : { label: 'tenant_instance_id', kind: (monaco as any).languages.CompletionItemKind.Property, insertText: 'tenant_instance_id: ${1:<DATASOURCE>}', insertTextRules: (monaco as any).languages.CompletionItemInsertTextRule.InsertAsSnippet },
          lang === 'json'
            ? { label: 'joins', kind: (monaco as any).languages.CompletionItemKind.Property, insertText: '"joins": [\n  { "name": "${1:<other_cube>}", "sql_on": "${CUBE}.id = ${other_cube}.id" }\n]', insertTextRules: (monaco as any).languages.CompletionItemInsertTextRule.InsertAsSnippet }
            : { label: 'joins', kind: (monaco as any).languages.CompletionItemKind.Property, insertText: 'joins:\n  - name: ${1:<other_cube>}\n    sql_on: "${CUBE}.id = ${other_cube}.id"', insertTextRules: (monaco as any).languages.CompletionItemInsertTextRule.InsertAsSnippet },
          lang === 'json'
            ? { label: 'pre_aggregations', kind: (monaco as any).languages.CompletionItemKind.Property, insertText: '"pre_aggregations": [\n  { "name": "${1:<agg_name>}", "type": "rollup", "time_dimension": "${2:<time>}", "dimensions": [] }\n]', insertTextRules: (monaco as any).languages.CompletionItemInsertTextRule.InsertAsSnippet }
            : { label: 'pre_aggregations', kind: (monaco as any).languages.CompletionItemKind.Property, insertText: 'pre_aggregations:\n  - name: ${1:<agg_name>}\n    type: rollup\n    time_dimension: ${2:<time>}\n    dimensions: []', insertTextRules: (monaco as any).languages.CompletionItemInsertTextRule.InsertAsSnippet },
          // Dimension skeleton
          lang === 'json'
            ? { label: 'dimension_skeleton', kind: (monaco as any).languages.CompletionItemKind.Snippet, insertText: '\"dimensions\": {\n  \"${1:dim_name}\": { \n    \"sql\": \"${TABLE}.${2:column}\", \n    \"type\": \"${3:string}\", \n    \"description\": \"\" \n  }\n}', insertTextRules: (monaco as any).languages.CompletionItemInsertTextRule.InsertAsSnippet }
            : { label: 'dimension_skeleton', kind: (monaco as any).languages.CompletionItemKind.Snippet, insertText: 'dimensions:\n  ${1:dim_name}:\n    sql: \"${TABLE}.${2:column}\"\n    type: ${3:string}\n    description: \"\"', insertTextRules: (monaco as any).languages.CompletionItemInsertTextRule.InsertAsSnippet },
          // Measure skeleton
          lang === 'json'
            ? { label: 'measure_skeleton', kind: (monaco as any).languages.CompletionItemKind.Snippet, insertText: '\"measures\": {\n  \"${1:measure_name}\": {\n    \"sql\": \"${TABLE}.${2:column}\",\n    \"type\": \"${3:sum}\",\n    \"description\": \"\"\n  }\n}', insertTextRules: (monaco as any).languages.CompletionItemInsertTextRule.InsertAsSnippet }
            : { label: 'measure_skeleton', kind: (monaco as any).languages.CompletionItemKind.Snippet, insertText: 'measures:\n  ${1:measure_name}:\n    sql: \"${TABLE}.${2:column}\"\n    type: ${3:sum}\n    description: \"\"', insertTextRules: (monaco as any).languages.CompletionItemInsertTextRule.InsertAsSnippet },
          // Pre-aggregation originalSql variant
          lang === 'json'
            ? { label: 'pre_aggregation_originalSql', kind: (monaco as any).languages.CompletionItemKind.Snippet, insertText: '\"pre_aggregations\": [\n  { \"name\": \"${1:raw}\", \"type\": \"originalSql\" }\n]', insertTextRules: (monaco as any).languages.CompletionItemInsertTextRule.InsertAsSnippet }
            : { label: 'pre_aggregation_originalSql', kind: (monaco as any).languages.CompletionItemKind.Snippet, insertText: 'pre_aggregations:\n  - name: ${1:raw}\n    type: originalSql', insertTextRules: (monaco as any).languages.CompletionItemInsertTextRule.InsertAsSnippet },
        ];
        const extra = (dynCompletionsRef.current || []).map((c) => ({ label: c.label, insertText: c.insertText, kind: c.kind || (monaco as any).languages.CompletionItemKind.Text }));
        return [...base, ...extra];
      };
      
      const jsonProv = (monaco as any).languages.registerCompletionItemProvider('json', { provideCompletionItems: () => ({ suggestions: mkSuggestions('json') }) });
      const yamlProv = (monaco as any).languages.registerCompletionItemProvider('yaml', { provideCompletionItems: () => ({ suggestions: mkSuggestions('yaml') }) });
      
      // Generic provider for Python to support Starlark dynamic variables
      const pyProv = (monaco as any).languages.registerCompletionItemProvider('python', {
        provideCompletionItems: () => {
             const extra = (dynCompletionsRef.current || []).map((c) => ({ 
                 label: c.label, 
                 insertText: c.insertText, 
                 kind: c.kind || (monaco as any).languages.CompletionItemKind.Variable 
             }));
             return { suggestions: extra };
        }
      });
      
      disposeRefs.current = { ...(disposeRefs.current || {}), jsonCompletion: jsonProv, yamlCompletion: yamlProv, pythonCompletion: pyProv };
    } catch {}
  }, [dynamicCompletions]);

  return (
    <div className="monaco-container">
      <div ref={containerRef} className="monaco-inner editor-input" role="textbox" aria-label={`${(language || 'code').toString().toUpperCase()} code editor`} />
      {/* If monaco not loaded the container will remain empty; callers should
          provide a fallback editor in that case. */}
    </div>
  );
};

export default MonacoCodeEditor;
