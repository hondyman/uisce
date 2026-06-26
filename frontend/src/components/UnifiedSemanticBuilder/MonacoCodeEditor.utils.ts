import yaml from 'js-yaml';

// Local window/monaco narrow to avoid repeated `(window as any).monaco` casts
const win = (window as unknown) as { monaco?: any };

// Helper: apply an AST-based edit by parsing the document, applying an updater,
// and returning a full-document replacement edit. This preserves valid JSON/YAML
// structure and produces nicely formatted output rather than piecemeal text
// insertion which can easily break commas/indentation.
export const buildAstReplacement = (model: any, language: string, updater: (obj: any) => void) => {
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
      range: new win.monaco.Range(1, 1, lineCount, endCol),
      text: out,
    };
  } catch (e) {
    // Fallback: no AST edit possible
    return null;
  }
};

export const buildMinimalJsonInsert = (model: any, key: string, valueText: string) => {
  try {
    const txt = model.getValue();
    const lastBrace = txt.lastIndexOf('}');
    if (lastBrace === -1) return null;
    const before = txt.slice(0, lastBrace).trimEnd();
    const needComma = before.endsWith('{') ? false : before.endsWith(',') ? false : true;
    const insertText = (needComma ? ',\n  ' : '\n  ') + `"${key}": ${valueText}` + '\n';
    const lineCount = model.getLineCount();
    const endCol = model.getLineMaxColumn(lineCount);
    return {
      range: new win.monaco.Range(lineCount, 1, lineCount, endCol),
      text: insertText + '\n}',
    };
  } catch (_) { return null; }
};

export const buildMinimalYamlInsert = (model: any, key: string, valueText: string) => {
  try {
    const lc = model.getLineCount();
    const ec = model.getLineMaxColumn(lc);
    const insertText = `\n${key}: ${valueText}\n`;
  return { range: new win.monaco.Range(lc, ec, lc, ec), text: insertText };
  } catch (_) { return null; }
};

export const buildJsoncPathInsert = (model: any, path: Array<string | number>, key: string, valueText: string) => {
  try {
    // eslint-disable-next-line @typescript-eslint/no-var-requires
    const { modify, applyEdits } = require('jsonc-parser');
    const text = model.getValue();
    const edits = modify(text, [...path, key], JSON.parse(valueText), { formattingOptions: { insertSpaces: true, tabSize: 2 } });
    if (!edits || edits.length === 0) return null;
    const first = edits[0];
    const startOffset = first.offset;
    const endOffset = first.offset + first.length;
    const toPos = (offset: number) => {
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
  const range = new win.monaco.Range(s.lineNumber, s.column, e.lineNumber, e.column);
    return { range, text: newText };
  } catch (e) {
    return null;
  }
};

export const buildRecastJsonInsert = (model: any, path: Array<string | number>, key: string, valueText: string) => {
  try {
    // eslint-disable-next-line @typescript-eslint/no-var-requires
    const recast = require('recast');
    const text = model.getValue();
  const ast = recast.parse(text, { parser: require('recast/parsers/typescript') });
  void ast;
    if (!path || path.length === 0) {
      const obj = JSON.parse(text || '{}');
      obj[key] = JSON.parse(valueText);
      const out = JSON.stringify(obj, null, 2);
      const lc = model.getLineCount();
      const ec = model.getLineMaxColumn(lc);
    return { range: new win.monaco.Range(1, 1, lc, ec), text: out };
    }
    return null;
  } catch (e) {
    return null;
  }
};

export const computeQuickFixActions = (markers: any[], modelValue: string, language: string) => {
  void modelValue;
  const actions: any[] = [];
  for (const marker of markers || []) {
    const srcMarker = marker;
    const code = (marker as any).code || (marker as any).meta || undefined;
    const msg = String((marker as any).message || '').toLowerCase();

    if (code === 'MISSING_DATASOURCE' || code === 'ERR_DATASOURCE_MISSING' || /missing_datasource/i.test(String(code || ''))) {
      actions.push({
        title: 'Insert tenant_instance_id (AST)',
        kind: 'quickfix',
        marker: srcMarker,
        insert: { key: 'tenant_instance_id', valueText: '"<YOUR_DATASOURCE>"' },
  updater: (_model: any) => ({ kind: 'ast', apply: (m: any) => buildAstReplacement(m, language || 'json', (obj: any) => { obj.tenant_instance_id = obj.tenant_instance_id || '<YOUR_DATASOURCE>'; }) }),
      });
      continue;
    }

    if (code === 'MISSING_JOIN' || /missing_join/i.test(String(code || ''))) {
      actions.push({
        title: 'Scaffold join (AST)',
        kind: 'quickfix',
        marker: srcMarker,
        insert: { path: [], key: 'joins', valueText: '[ { "name": "<other_cube>", "sql_on": "${CUBE}.id = ${other_cube}.id" } ]' },
  updater: (_model: any) => ({ kind: 'ast', apply: (m: any) => buildAstReplacement(m, language || 'json', (obj: any) => { obj.joins = obj.joins || [{ name: '<other_cube>', sql_on: '${CUBE}.id = ${other_cube}.id' }]; }) }),
      });
      continue;
    }

    if (code === 'INVALID_MEASURE' || /invalid_measure/i.test(String(code || '')) || /invalid_measure/.test(msg)) {
      actions.push({ title: 'Rename measure (suggestion)', kind: 'quickfix', updater: null });
      continue;
    }

    if (code === 'MISSING_PRE_AGG' || /pre_?aggregation_missing/i.test(String(code || '')) || /pre[_ -]?aggregation/.test(msg)) {
      actions.push({
        title: 'Create pre_aggregation (AST)',
        kind: 'quickfix',
        marker: srcMarker,
        insert: { path: [], key: 'pre_aggregations', valueText: '[ { "name": "<agg_name>", "type": "rollup", "time_dimension": "<time>", "dimensions": [] } ]' },
  updater: (_model: any) => ({ kind: 'ast', apply: (m: any) => buildAstReplacement(m, language || 'json', (obj: any) => { obj.pre_aggregations = obj.pre_aggregations || [{ name: '<agg_name>', type: 'rollup', time_dimension: '<time>', dimensions: [] }]; }) }),
      });
      continue;
    }

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

export const convertActionsToMonacoEdits = (actions: any[], model: any, monaco: any, language: string) => {
  const out: any[] = [];
  for (const act of actions || []) {
    try {
      if (act.insert && typeof act.insert === 'object') {
        try {
          let replacement: any = null;
          if ((language || '').toLowerCase() === 'json') {
            replacement = buildJsoncPathInsert(model, [], act.insert.key, act.insert.valueText) || buildMinimalJsonInsert(model, act.insert.key, act.insert.valueText) || buildRecastJsonInsert(model, [], act.insert.key, act.insert.valueText);
          } else {
            replacement = buildMinimalYamlInsert(model, act.insert.key, act.insert.valueText);
          }
          if (replacement && replacement.range && typeof replacement.text === 'string') {
            out.push({ title: act.title, edit: { edits: [{ resource: model.uri, edit: { range: replacement.range, text: replacement.text } }] }, kind: act.kind, diagnostics: act.marker ? [act.marker] : [] });
            continue;
          }
        } catch (_) {
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
      const marker = act.marker;
      let range = null;
      if (marker && typeof marker.startLineNumber === 'number') {
        range = new monaco.Range(marker.startLineNumber, marker.startColumn || 1, marker.endLineNumber || marker.startLineNumber, marker.endColumn || (marker.startColumn || 1));
      } else {
        const lc = model.getLineCount();
        const ec = model.getLineMaxColumn(lc);
        range = new monaco.Range(lc, ec, lc, ec);
      }
      const text = act.rawText || act.replacementText || '';
      out.push({ title: act.title, edit: { edits: [{ resource: model.uri, edit: { range, text } }] }, kind: act.kind, diagnostics: marker ? [marker] : [] });
    } catch (_) {
    }
  }
  return out;
};
