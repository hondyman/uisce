import { registerOperations, type OperationDef } from '../../studio-core/operations/registry';
import type { FormFieldSpec, RowFieldSpec } from '../../pages/page-studio/app/appModel';
import { masteringApi } from '../mastering/api';
import {
  type BOSchemaField, type Column, type ColumnType, type Condition, type FieldMap, type NodeConfigs, type NodeKind, type NodeType,
  type PreviewResult, type RunRecord, type Spec, type SpecNode, type TargetField, boTarget, pipelinesApi, platformApi,
} from './api';
import { downstreamSink, fieldsIn, missingRequired, newNodeId, type SourceFieldLookup } from './fields';

/**
 * The pipeline editor's operations (the Data pipeline editor core page is
 * built in Page Studio: blueprints/dataPipelines.ts). The page holds the
 * spec in a variable and draws it with the Canvas widget; everything a step
 * *means* is decided here - what a new step is and where it chains, what a
 * step's settings form holds (fieldsFrom) and how the form maps back onto
 * its config, which fields reach a step, what a preview or a run says.
 * Operations over the spec are pure and return the next spec, so the page
 * stays declarative.
 */

const s = (p: Record<string, unknown>, k: string) => (p[k] === undefined || p[k] === null ? '' : String(p[k]));
const EMPTY_SPEC: Spec = { version: 1, nodes: [], edges: [], error_policy: 'skip_and_log' };
const specOf = (v: unknown): Spec => {
  const x = (v && typeof v === 'object' ? v : {}) as Partial<Spec>;
  return { ...EMPTY_SPEC, ...x, nodes: x.nodes ?? [], edges: x.edges ?? [] };
};

// --- steps -----------------------------------------------------------------------

/** Each step kind's look on the canvas and in the palette. */
const LOOK: Record<NodeKind, { icon: string; category: 'source' | 'step' | 'destination' }> = {
  file_source: { icon: 'file', category: 'source' },
  bo_source: { icon: 'business', category: 'source' },
  validate: { icon: 'review', category: 'step' },
  rule_check: { icon: 'gavel', category: 'step' },
  map: { icon: 'swap', category: 'step' },
  bo_sink: { icon: 'business', category: 'destination' },
  staging_sink: { icon: 'table', category: 'destination' },
  file_sink: { icon: 'export', category: 'destination' },
  iceberg_sink: { icon: 'storage', category: 'destination' },
  master: { icon: 'hub', category: 'destination' },
};
const lookOf = (k: string) => LOOK[k as NodeKind] ?? { icon: 'help', category: 'step' as const };
export const categoryOf = (k: string) => lookOf(k).category;

export function defaultConfig(kind: NodeKind): NodeConfigs[NodeKind] {
  switch (kind) {
    case 'file_source': return { uri: '', format: 'csv', has_header: true, columns: [] };
    case 'bo_source': return { bo_key: '', filters: [] };
    case 'validate': return { required: [], unique: [] };
    case 'rule_check': return { rule_ids: [] };
    case 'map': return { fields: [] };
    case 'bo_sink': return { bo_key: '', mode: 'create' };
    case 'staging_sink': return { table: '', source_cd: '', domain: '' };
    case 'file_sink': return { uri: '', format: 'csv' };
    case 'iceberg_sink': return { namespace: 'default', table: '', partition_by: [], format: 'parquet' };
    case 'master': return { entity: '' };
  }
}

/** One line: what a configured step does. */
export function summarize(n: SpecNode): string {
  const c = n.config as Record<string, any>; // eslint-disable-line @typescript-eslint/no-explicit-any
  switch (n.type) {
    case 'file_source': return c.uri ? `${c.uri} · ${(c.columns ?? []).length} columns` : '';
    case 'bo_source': return c.bo_key ? `${c.bo_key}${c.filters?.length ? ` · ${c.filters.length} filter(s)` : ''}` : '';
    case 'validate': return [c.required?.length && `${c.required.length} required`, c.unique?.length && `unique on ${c.unique.join('+')}`].filter(Boolean).join(' · ');
    case 'rule_check': return c.rule_ids?.length ? `${c.rule_ids.length} rule(s)${c.bo_key ? ` of ${c.bo_key}` : ''}` : '';
    case 'map': return c.fields?.length ? `${c.fields.length} field(s) mapped` : '';
    case 'bo_sink': return c.bo_key ? `${c.mode === 'upsert' ? 'update or create' : 'create'} ${c.bo_key}${c.dry_run ? ' (rehearsal)' : ''}` : '';
    case 'staging_sink': return c.table ? `${c.table}${c.source_cd ? ` · ${c.source_cd}/${c.domain}` : ''}` : '';
    case 'file_sink': return c.uri ? `${c.uri} (${c.format})` : '';
    case 'iceberg_sink': return c.table ? `${c.namespace || 'default'}.${c.table} (iceberg)` : '';
    case 'master': return c.entity ? `master into ${c.entity} golden records` : '';
  }
  return '';
}

function guessFormat(uri: string): 'csv' | 'json' | 'parquet' {
  const u = uri.toLowerCase();
  if (/\.(json|ndjson|jsonl)$/.test(u)) return 'json';
  if (/\.(parquet|pq)$/.test(u)) return 'parquet';
  return 'csv';
}

// Lookups a step's form needs, cached briefly: the form is reshaped on every edit.
const cache = new Map<string, { at: number; value: Promise<unknown> }>();
function cached<T>(key: string, ms: number, fn: () => Promise<T>): Promise<T> {
  const hit = cache.get(key);
  if (hit && Date.now() - hit.at < ms) return hit.value as Promise<T>;
  const value = fn().catch((e) => { cache.delete(key); throw e; });
  cache.set(key, { at: Date.now(), value });
  return value;
}
const MIN5 = 5 * 60_000;
const bos = () => cached('bos', MIN5, platformApi.businessObjects);
const boFields = (k: string) => (k ? cached(`bo:${k}`, MIN5, async () => (await platformApi.boSchema(k)).fields ?? []).catch(() => [] as BOSchemaField[]) : Promise.resolve([] as BOSchemaField[]));
const stagingTables = () => cached('staging', MIN5, pipelinesApi.stagingTables);
const files = () => cached('files', 30_000, () => pipelinesApi.files());
const rulesOf = (k: string) => (k ? cached(`rules:${k}`, 60_000, () => platformApi.rules(k)) : Promise.resolve([]));
const profiles = () => cached('profiles', MIN5, masteringApi.profiles);

/** Business-object schemas for every BO the spec reads or writes. */
async function schemasFor(spec: Spec): Promise<Record<string, BOSchemaField[]>> {
  const keys = [...new Set(spec.nodes.filter((n) => n.type === 'bo_source' || n.type === 'bo_sink')
    .map((n) => (n.config as { bo_key?: string }).bo_key).filter(Boolean) as string[])];
  return Object.fromEntries(await Promise.all(keys.map(async (k) => [k, await boFields(k)] as const)));
}
const lookupFrom = (schemas: Record<string, BOSchemaField[]>): SourceFieldLookup => (n) => {
  if (n.type !== 'bo_source') return undefined;
  return schemas[(n.config as { bo_key: string }).bo_key]?.map((x) => ({ name: boTarget(x).name, type: (x.type as Column['type']) ?? 'string' }));
};

// --- a step's settings as a form, and back ------------------------------------------

type Form = Record<string, unknown>;
const COLUMN_TYPES: ColumnType[] = ['string', 'int', 'float', 'decimal', 'bool', 'date', 'timestamp'];
const TRANSFORMS = [
  { v: '', l: 'as is' }, { v: 'trim', l: 'trim spaces' }, { v: 'upper', l: 'UPPER CASE' }, { v: 'lower', l: 'lower case' },
  { v: 'to_date', l: 'to date' }, { v: 'to_number', l: 'to number' }, { v: 'lookup', l: 'lookup table' },
];
// The rule engine's operators that can filter at the source (vm.CompileConditionSQL).
const FILTER_OPS: { v: string; l: string; arity: 0 | 1 | 2 | 'list' }[] = [
  { v: 'equals', l: 'equals', arity: 1 }, { v: 'not_equals', l: 'does not equal', arity: 1 },
  { v: 'greater_than', l: '>', arity: 1 }, { v: 'greater_equal', l: '≥', arity: 1 },
  { v: 'less_than', l: '<', arity: 1 }, { v: 'less_equal', l: '≤', arity: 1 },
  { v: 'between', l: 'between', arity: 2 }, { v: 'in', l: 'is one of', arity: 'list' }, { v: 'not_in', l: 'is not one of', arity: 'list' },
  { v: 'contains', l: 'contains', arity: 1 }, { v: 'starts_with', l: 'starts with', arity: 1 }, { v: 'ends_with', l: 'ends with', arity: 1 },
  { v: 'before', l: 'before (date)', arity: 1 }, { v: 'after', l: 'after (date)', arity: 1 },
  { v: 'is_null', l: 'is empty', arity: 0 }, { v: 'is_not_null', l: 'has a value', arity: 0 },
  { v: 'is_true', l: 'is true', arity: 0 }, { v: 'is_false', l: 'is false', arity: 0 },
];
const arityOf = (op: string) => FILTER_OPS.find((o) => o.v === op)?.arity ?? 1;
const byArity = (a: 0 | 1 | 2 | 'list') => FILTER_OPS.filter((o) => o.arity === a).map((o) => o.v);

const parseLookup = (text: string): Record<string, string> => {
  const out: Record<string, string> = {};
  for (const part of text.split(',')) {
    const [k, ...v] = part.split('=');
    if (k?.trim()) out[k.trim()] = v.join('=').trim();
  }
  return out;
};
const lookupText = (l?: Record<string, string>) => Object.entries(l ?? {}).map(([k, v]) => `${k}=${v}`).join(', ');

/** A step's config as the values its form edits (some shapes differ: staging columns are shown column -> field). */
export function toForm(n: SpecNode, info?: StepInfo): Form {
  const c = n.config as Record<string, unknown>;
  const base: Form = { label: n.label ?? '' };
  switch (n.type) {
    case 'file_source': {
      const sample = info?.id === n.id ? info.sample?.[0] : undefined;
      return {
        ...base, ...c, delimiter: c.delimiter ?? ',',
        columns: ((c.columns as Column[] | undefined) ?? []).map((col) => ({
          ...col, required: col.nullable === false, sample: sample ? `e.g. ${String(sample[col.name] ?? '—')}` : '',
        })),
      };
    }
    case 'bo_source':
      return {
        ...base, ...c,
        filters: ((c.filters as Condition[] | undefined) ?? []).map((f) => {
          const a = arityOf(f.operator);
          const v = f.value;
          return {
            field: f.field, operator: f.operator,
            value: a === 1 ? (v ?? '') : '', from: a === 2 && Array.isArray(v) ? (v[0] ?? '') : '', to: a === 2 && Array.isArray(v) ? (v[1] ?? '') : '',
            values: a === 'list' && Array.isArray(v) ? v : [],
          };
        }),
      };
    case 'map':
      return { ...base, ...c, fields: ((c.fields as FieldMap[] | undefined) ?? []).map((f) => ({ ...f, transform: f.transform ?? '', lookup_text: lookupText(f.lookup) })) };
    case 'staging_sink':
      // Stored field -> column; edited per column.
      return { ...base, ...c, columns: Object.fromEntries(Object.entries((c.columns as Record<string, string> | undefined) ?? {}).map(([from, col]) => [col, from])) };
    case 'file_sink':
      return { ...base, ...c, delimiter: c.delimiter ?? ',' };
    default:
      return { ...base, ...c };
  }
}

/** The form's values back onto the step (with the rules a change of file, format or table brings). */
export function fromForm(n: SpecNode, f: Form): SpecNode {
  const { label, ...v } = f;
  const old = n.config as Record<string, unknown>;
  let config: Record<string, unknown>;
  switch (n.type) {
    case 'file_source': {
      const uri = String(v.uri ?? '');
      const cols = ((v.columns as Record<string, unknown>[] | undefined) ?? []).map(({ required, sample: _s, ...col }) => {
        const out = { ...col, nullable: required ? false : undefined } as Record<string, unknown>;
        if (out.nullable === undefined) delete out.nullable;
        return out as unknown as Column;
      });
      // A different file starts over (format guessed, columns re-read); a new version of the same keeps its contract.
      config = uri !== String(old.uri ?? '')
        ? { ...old, ...v, uri, format: guessFormat(uri), columns: [] }
        : { ...old, ...v, columns: cols };
      break;
    }
    case 'bo_source': {
      const filters = ((v.filters as Record<string, unknown>[] | undefined) ?? []).map((r) => {
        const op = String(r.operator || 'equals');
        const a = arityOf(op);
        const value = a === 0 ? undefined : a === 2 ? [r.from ?? '', r.to ?? ''] : a === 'list' ? (r.values ?? []) : (r.value ?? '');
        return { field: String(r.field ?? ''), operator: op, ...(value === undefined ? {} : { value }) } as Condition;
      });
      const limit = v.limit === '' || v.limit === null || v.limit === undefined ? undefined : Number(v.limit);
      config = { ...old, ...v, filters, limit };
      if (limit === undefined) delete config.limit;
      break;
    }
    case 'map':
      config = {
        ...old, ...v,
        fields: ((v.fields as Record<string, unknown>[] | undefined) ?? []).map(({ lookup_text, transform, ...r }) => {
          const out: Record<string, unknown> = { ...r, from: r.from ?? '', to: r.to ?? '' };
          if (transform) out.transform = transform; else delete out.transform;
          if (transform === 'lookup') out.lookup = parseLookup(String(lookup_text ?? '')); else delete out.lookup;
          return out as unknown as FieldMap;
        }),
      };
      break;
    case 'staging_sink':
      config = {
        ...old, ...v,
        source_cd: String(v.source_cd ?? '').toUpperCase(), domain: String(v.domain ?? '').toUpperCase(),
        run_ref: v.run_ref ? v.run_ref : undefined,
        columns: Object.fromEntries(Object.entries((v.columns as Record<string, string> | undefined) ?? {}).filter(([, from]) => !!from).map(([col, from]) => [from, col])),
      };
      if (!config.run_ref) delete config.run_ref;
      break;
    case 'file_sink': {
      const uri = String(v.uri ?? '');
      config = { ...old, ...v, format: uri !== String(old.uri ?? '') ? guessFormat(uri) : v.format };
      break;
    }
    default:
      config = { ...old, ...v };
  }
  return { ...n, label: String(label ?? ''), config: config as unknown as NodeConfigs[NodeKind] };
}

/** What a step shows beside its form: the file read, the last suggestion. */
interface StepInfo { id: string; row_count?: number; sample?: Record<string, unknown>[]; note?: string }

const note = (name: string, label: string, severity?: FormFieldSpec['severity']): FormFieldSpec => ({ name, kind: 'note', label, severity, wide: true });
const cond = (field: string, operator: string, value?: unknown) => ({ type: 'condition' as const, field, operator, value });
const opts = (list: string[]) => list.map((x) => ({ value: x, label: x }));
const set = (name: string, value: unknown) => ({ kind: 'setVariable' as const, name, value });
/** After a step operation: the spec, the form and the page marked unsaved. */
const applied = [set('spec', '{{result.spec}}'), set('stepDraft', '{{result.form}}'), set('dirty', true)];

async function stepFields(spec: Spec, n: SpecNode, info?: StepInfo): Promise<FormFieldSpec[]> {
  const schemas = await schemasFor(spec);
  const input = fieldsIn(spec, n.id, lookupFrom(schemas));
  const inputOpts = opts(input.map((c) => c.name));
  const c = n.config as Record<string, unknown>;
  const mine = info?.id === n.id ? info : undefined;
  const boOptions = async () => (await bos().catch(() => [])).map((b) => ({ value: b.name, label: b.display_name || b.name, caption: b.description }));
  const head: FormFieldSpec[] = [
    { name: 'label', kind: 'text', label: 'Step name', wide: true },
    { name: '_remove', kind: 'button', label: 'Remove this step', icon: 'delete', buttonVariant: 'text',
      onClick: [{ kind: 'runOperation', operation: 'dataPipelines.removeStep', params: { spec: '{{vars.spec}}', id: n.id },
        onSuccess: [set('spec', '{{result.spec}}'), set('selected', null), set('dirty', true)] }] },
  ];
  switch (n.type) {
    case 'file_source': {
      const list = await files().catch(() => null);
      return [...head,
        note('hint', 'Pick an uploaded file (or upload one), then press Read file to see its columns. Every row is checked against the columns below.'),
        { name: 'uri', kind: 'select', label: 'File', required: true, options: opts((list ?? []).map((f) => f.path)), helperText: list ? undefined : 'Could not list files' },
        { name: '_upload', kind: 'upload', label: 'Upload a file', uploadOperation: 'dataPipelines.upload',
          onClick: [{ kind: 'runOperation', operation: 'dataPipelines.setStep', params: { spec: '{{vars.spec}}', id: n.id, patch: { uri: '{{result.uri}}' } }, onSuccess: applied }] },
        { name: 'format', kind: 'select', label: 'Format', required: true, options: opts(['csv', 'json', 'parquet']) },
        { name: 'delimiter', kind: 'select', label: 'Separator', required: true, visibleWhen: cond('form.format', 'equals', 'csv'),
          options: [{ value: ',', label: 'comma ,' }, { value: '|', label: 'pipe |' }, { value: ';', label: 'semicolon ;' }, { value: 'tab', label: 'tab' }] },
        { name: 'has_header', kind: 'switch', label: 'Header row', visibleWhen: cond('form.format', 'equals', 'csv') },
        { name: '_read', kind: 'button', label: 'Read file', icon: 'preview', buttonVariant: 'contained', readOnlyWhen: cond('form.uri', 'is_empty'),
          onClick: [{ kind: 'runOperation', operation: 'dataPipelines.readFile', params: { spec: '{{vars.spec}}', id: n.id },
            onSuccess: [...applied, set('stepInfo', '{{result.info}}')] }] },
        ...(mine?.row_count !== undefined ? [note('rows', `${mine.row_count.toLocaleString()} rows in this file.`, 'info')] : []),
        ...((c.columns as Column[] | undefined)?.length ? [{
          name: 'columns', kind: 'rows' as const, label: `Columns (${(c.columns as Column[]).length})`, fixedRows: true, wide: true,
          rowFields: [
            { name: 'name', label: 'Column', readOnly: true, caption: '{{row.sample}}', flex: 2 },
            { name: 'type', label: 'Type', kind: 'select', options: opts(COLUMN_TYPES) },
            { name: 'required', label: 'Required', kind: 'switch' },
          ] satisfies RowFieldSpec[],
        }] : []),
      ];
    }
    case 'bo_source': {
      const fields = (await boFields(String(c.bo_key ?? ''))).map(boTarget);
      return [...head,
        note('hint', "Read records from a business object. Only your tenant's records are read."),
        { name: 'bo_key', kind: 'select', label: 'Business object', required: true, options: await boOptions() },
        {
          name: 'filters', kind: 'rows', label: 'Only records where…', addLabel: 'Add condition', wide: true, resetOn: ['bo_key'],
          visibleWhen: cond('form.bo_key', 'is_not_empty'),
          rowFields: [
            { name: 'field', label: 'Field', kind: 'select', options: fields.map((t) => ({ value: t.name, label: t.label ?? t.name })) },
            { name: 'operator', label: 'Condition', kind: 'select', default: 'equals', options: FILTER_OPS.map((o) => ({ value: o.v, label: o.l })) },
            { name: 'value', label: 'value', visibleWhen: cond('row.operator', 'in', byArity(1)) },
            { name: 'from', label: 'from', visibleWhen: cond('row.operator', 'equals', 'between') },
            { name: 'to', label: 'to', visibleWhen: cond('row.operator', 'equals', 'between') },
            { name: 'values', label: 'values', kind: 'chips', placeholder: 'type, then Enter', visibleWhen: cond('row.operator', 'in', byArity('list')) },
          ] satisfies RowFieldSpec[],
        },
        { name: 'limit', kind: 'number', label: 'Read at most (optional)', step: 1 },
      ];
    }
    case 'validate':
      return [...head,
        note('hint', 'Rows missing a required value, or repeating a key already seen in this run, are rejected with the reason.'),
        { name: 'required', kind: 'chips', label: 'Must have a value', options: inputOpts, helperText: ' ' },
        { name: 'unique', kind: 'chips', label: 'Must be unique together', options: inputOpts, helperText: ' ' },
        ...(input.length === 0 ? [note('connect', 'Connect this step to a source to pick fields.', 'info')] : []),
      ];
    case 'rule_check': {
      const rules = (await rulesOf(String(c.bo_key ?? '')).catch(() => [])).filter((r) => r.is_active);
      return [...head,
        note('hint', 'Apply validation rules from the rules catalog - the same rules the business object enforces. Block rules reject the row; Warn rules keep it and record a warning.'),
        { name: 'bo_key', kind: 'select', label: 'Rules of business object', options: await boOptions() },
        ...(c.bo_key && rules.length === 0 ? [note('none', 'This business object has no active rules yet.', 'info')] : []),
        {
          name: 'rule_ids', kind: 'checklist', label: '', wide: true,
          options: rules.map((r) => ({
            value: r.id, label: r.name, caption: r.description,
            badges: [
              { label: r.severity === 'BLOCK' ? 'Block' : 'Warn', color: r.severity === 'BLOCK' ? 'error' as const : 'warning' as const, variant: 'outlined' as const },
              ...(r.origin === 'core' ? [{ label: 'core' }] : []),
            ],
          })),
        },
        { name: '_create', kind: 'button', label: 'Create a rule', buttonVariant: 'text', onClick: [{ kind: 'navigate', to: '/core/validation-rules/editor' }] },
      ];
    }
    case 'map': {
      const sink = downstreamSink(spec, n.id);
      const t = await targetsFor(sink, schemas);
      const rows = (c.fields as FieldMap[] | undefined) ?? [];
      const missing = t.targets ? missingRequired(rows.map((r) => r.to), t.targets) : [];
      return [...head,
        note('hint', `Choose which incoming field fills each ${t.label ? t.label : 'output'} field.`),
        ...(t.targets ? [{
          name: '_suggest', kind: 'button' as const, label: 'Suggest mappings', icon: 'assistant', readOnly: input.length === 0 || undefined,
          onClick: [{ kind: 'runOperation' as const, operation: 'dataPipelines.suggestMap', params: { spec: '{{vars.spec}}', id: n.id },
            onSuccess: [...applied, set('stepInfo', '{{result.info}}')] }],
        }] : []),
        ...(mine?.note ? [note('suggested', mine.note, 'info')] : []),
        ...(missing.length ? [note('missing', `Required and not mapped yet: ${missing.map((m) => m.label || m.name).join(', ')}`, 'warning')] : []),
        {
          name: 'fields', kind: 'rows', label: '', addLabel: 'Add mapping', wide: true,
          rowFields: [
            { name: 'from', label: 'From', kind: 'select', options: inputOpts },
            t.targets
              ? { name: 'to', label: 'To', kind: 'select', options: t.targets.map((x) => ({ value: x.name, label: `${x.label || x.name}${x.required ? ' *' : ''}` })) }
              : { name: 'to', label: 'To' },
            { name: 'transform', label: 'Transform', kind: 'select', options: TRANSFORMS.map((x) => ({ value: x.v, label: x.l })) },
            { name: 'lookup_text', label: 'Lookup', placeholder: 'A=Alpha, B=Beta', visibleWhen: cond('row.transform', 'equals', 'lookup') },
          ] satisfies RowFieldSpec[],
        },
        { name: 'keep_unmapped', kind: 'switch', label: "Also pass through fields I didn't map" },
      ];
    }
    case 'bo_sink': {
      const targets = (await boFields(String(c.bo_key ?? ''))).map(boTarget);
      const unknown = targets.length ? input.filter((f) => !targets.some((x) => x.name === f.name)) : [];
      const missing = missingRequired(input.map((f) => f.name), targets);
      return [...head,
        note('hint', "Write each row as a business object record. Every record goes through the object's validation rules; rejected records are reported with the rule that stopped them."),
        { name: 'bo_key', kind: 'select', label: 'Business object', required: true, options: await boOptions() },
        { name: 'mode', kind: 'select', label: 'When the record already exists', required: true,
          options: [{ value: 'create', label: 'always create a new record' }, { value: 'upsert', label: 'update it (match on key fields)' }] },
        { name: 'key_fields', kind: 'chips', label: 'Match records on', options: opts(targets.map((x) => x.name)), resetOn: ['bo_key'],
          visibleWhen: cond('form.mode', 'equals', 'upsert'), helperText: ' ' },
        { name: 'dry_run', kind: 'switch', label: 'Rehearsal: check every record against the rules, but save nothing' },
        ...(unknown.length ? [note('unknown', `Not fields of this object (add a Map step): ${unknown.map((u) => u.name).join(', ')}`, 'warning')] : []),
        ...(missing.length ? [note('missing', `Required fields no step provides: ${missing.map((m) => m.label || m.name).join(', ')}`, 'warning')] : []),
      ];
    }
    case 'staging_sink': {
      let tables: { table: string; columns: TargetField[] }[] = [];
      let error = '';
      try { tables = await stagingTables(); } catch (e) { error = (e as Error).message; }
      const table = tables.find((x) => x.table === c.table);
      return [...head,
        note('hint', 'Bulk-load rows into a staging table, as one tracked load. Re-running the same load reference does nothing; a failed load is cleared and retried.'),
        ...(error ? [note('tables', error, 'warning')] : []),
        { name: 'table', kind: 'select', label: 'Staging table', required: true, options: opts(tables.map((x) => x.table)) },
        { name: 'source_cd', kind: 'text', label: 'Source system', helperText: 'e.g. FACTSET' },
        { name: 'domain', kind: 'text', label: 'Domain', helperText: 'e.g. PRODUCT' },
        { name: 'run_ref', kind: 'text', label: 'Load reference (optional)', helperText: 'e.g. the file date. Blank: every run is a new load.' },
        ...(table ? [{
          name: 'columns', kind: 'map' as const, label: '', wide: true, resetOn: ['table'], options: inputOpts,
          map: {
            rows: table.columns.map((col) => ({ key: col.name, label: `${col.name}${col.required ? ' *' : ''}` })),
            placeholder: 'not loaded', groups: [{ id: '', title: 'Which field fills each column' }],
          },
        }] : []),
      ];
    }
    case 'file_sink':
      return [...head,
        note('hint', 'Export the rows to a file. It appears only when the whole run succeeds.'),
        { name: 'uri', kind: 'text', label: 'File name', helperText: 'e.g. exports/funds.csv' },
        { name: 'format', kind: 'select', label: 'Format', required: true, options: opts(['csv', 'json', 'parquet']) },
        { name: 'delimiter', kind: 'select', label: 'Separator', required: true, visibleWhen: cond('form.format', 'equals', 'csv'),
          options: [{ value: ',', label: 'comma ,' }, { value: '|', label: 'pipe |' }, { value: ';', label: 'semicolon ;' }] },
      ];
    case 'iceberg_sink':
      return [...head,
        note('hint', 'Store rows directly as columnar Parquet in the Iceberg data lakehouse (S3/MinIO cold storage). Provides a cheap historical raw archive queryable via StarRocks.'),
        { name: 'namespace', kind: 'text', label: 'Namespace / Catalog DB', helperText: 'e.g. default' },
        { name: 'table', kind: 'text', label: 'Table name', helperText: 'e.g. factset_security_raw' },
        ...(input.length ? [{ name: 'partition_by', kind: 'chips' as const, label: 'Partition columns', options: inputOpts, helperText: 'e.g. as_of_date' }] : []),
        note('format', 'Format: Parquet (Iceberg REST)'),
      ];
    case 'master': {
      const list = await profiles().then((r) => r.profiles).catch(() => null);
      return [...head,
        note('hint', 'After the staging load commits, master it: resolve each record to its golden record, survive, run the controls and publish - one mastering run, linked to this pipeline run. Connect this step after a "Load staging table" step; the table must be bound to the entity\'s business object (Staging bindings).'),
        { name: 'entity', kind: 'select', label: 'Entity', required: true, options: (list ?? []).map((p) => ({ value: p.entity_cd.toLowerCase(), label: p.display_name })),
          helperText: list && list.length === 0 ? 'No mastered entities are set up' : undefined },
      ];
    }
  }
  return head;
}

async function targetsFor(sink: SpecNode | undefined, schemas: Record<string, BOSchemaField[]>): Promise<{ targets?: TargetField[]; label?: string }> {
  if (!sink) return {};
  if (sink.type === 'bo_sink') {
    const k = (sink.config as { bo_key: string }).bo_key;
    return { targets: (schemas[k] ?? await boFields(k)).map(boTarget), label: k };
  }
  if (sink.type === 'staging_sink') {
    const t = (sink.config as { table: string }).table;
    const tables = await stagingTables().catch(() => []);
    return { targets: tables.find((x) => x.table === t)?.columns, label: t };
  }
  return {};
}

/** Replace one step and return the spec with its form. */
function withStep(spec: Spec, n: SpecNode, info?: StepInfo) {
  const next = { ...spec, nodes: spec.nodes.map((x) => (x.id === n.id ? n : x)) };
  return { spec: next, form: toForm(n, info) };
}
const stepOf = (spec: Spec, id: string) => {
  const n = spec.nodes.find((x) => x.id === id);
  if (!n) throw new Error(`No step ${id}`);
  return n;
};

// --- preview and runs -----------------------------------------------------------------

const fmt = (v: unknown) => (v === null || v === undefined ? '—' : typeof v === 'object' ? JSON.stringify(v) : String(v));
const RUN_COLOR: Record<RunRecord['status'], string> = { queued: 'default', running: 'info', completed: 'success', completed_with_errors: 'warning', failed: 'error' };

/** A preview result shaped for its tab, for the step picked (or all). */
function previewView(p: PreviewResult | null, pick: string, spec: Spec) {
  const label = (id?: string) => spec.nodes.find((n) => n.id === id)?.label || id || '';
  if (!p) return { state: 'none' };
  if (p.error && !p.summary) return { state: 'error', error: p.error };
  const sm = p.summary!;
  const last = sm.Nodes[sm.Nodes.length - 1]?.NodeID;
  const rows = sm.samples?.[pick || last || ''] ?? [];
  const cols = [...new Set(rows.flatMap((r) => Object.keys(r.Data)))];
  return {
    state: 'done', error: p.error ?? '',
    totals: [
      { label: `${sm.RecordsIn} read` }, { label: `${sm.RecordsOut} would be written`, color: 'success' },
      ...(sm.Errors > 0 ? [{ label: `${sm.Errors} rejected`, color: 'error' }] : []),
    ],
    steps: [{ value: '', label: 'All steps' }, ...sm.Nodes.map((n) => ({ value: n.NodeID, label: `${label(n.NodeID)}: ${n.Out}` }))],
    rejects: p.rejects.filter((r) => !pick || r.node_id === pick).slice(0, 50).map((r, i) => ({
      id: String(i), row: r.row, step: label(r.node_id), reason: r.reason, tone: r.kind === 'error' ? 'error' : 'warning',
    })),
    sample_caption: rows.length ? `Sample rows ${pick ? `leaving "${label(pick)}"` : 'that would be written'}` : '',
    columns: cols.map((c) => ({ code: c })),
    rows: rows.map((r) => ({ id: String(r.Num), num: r.Num, cells: Object.fromEntries(cols.map((c) => [c, fmt(r.Data[c])])) })),
  };
}

const operations: OperationDef[] = [
  {
    id: 'dataPipelines.load', domain: 'dp', kind: 'query', label: 'A pipeline to edit',
    description: '{name, spec, is_new} - "new" (or no id) starts a blank pipeline.',
    params: [{ name: 'id', type: 'string' }],
    fields: [{ name: 'name' }, { name: 'spec', type: 'object' }, { name: 'is_new', type: 'boolean' }],
    run: async (p) => {
      const id = s(p, 'id');
      if (!id || id === 'new') return { name: 'Untitled pipeline', spec: EMPTY_SPEC, is_new: true };
      const d = await pipelinesApi.get(id);
      return { name: d.name, spec: specOf(d.spec), is_new: false };
    },
  },
  {
    id: 'dataPipelines.stepKinds', domain: 'dp', kind: 'query', label: 'Pipeline steps to add',
    description: 'The palette: type, label, category (source / step / destination), icon, available, description (with why not).',
    params: [],
    fields: [{ name: 'type' }, { name: 'label' }, { name: 'category' }, { name: 'icon' }, { name: 'available', type: 'boolean' }, { name: 'description' }],
    run: async () => (await pipelinesApi.nodeTypes()).map((t: NodeType) => ({
      ...t, icon: lookOf(t.type).icon, description: t.available ? t.description : `${t.description} - unavailable: ${t.unavailable_reason}`,
    })),
  },
  {
    id: 'dataPipelines.addStep', domain: 'dp', kind: 'mutation', label: 'Add a step', invalidates: [],
    description: 'A new step of item.type with its default settings, after the selected step when that makes sense. result: spec, id.',
    params: [{ name: 'item', type: 'object', required: true }, { name: 'spec', type: 'object', required: true }, { name: 'selected', type: 'string' }],
    run: async (p) => {
      const spec = specOf(p.spec);
      const t = p.item as NodeType;
      const sel = spec.nodes.find((n) => n.id === s(p, 'selected'));
      const id = newNodeId(t.type, spec);
      const position = sel?.position ? { x: sel.position.x + 280, y: sel.position.y } : { x: 60 + spec.nodes.length * 60, y: 80 + spec.nodes.length * 40 };
      const edges = [...spec.edges];
      if (sel && lookOf(sel.type).category !== 'destination' && t.category !== 'source') edges.push({ from: sel.id, to: id });
      return { spec: { ...spec, nodes: [...spec.nodes, { id, type: t.type, label: t.label, config: defaultConfig(t.type), position }], edges }, id };
    },
  },
  {
    id: 'dataPipelines.removeStep', domain: 'dp', kind: 'mutation', label: 'Remove a step', invalidates: [],
    params: [{ name: 'spec', type: 'object', required: true }, { name: 'id', type: 'string', required: true }],
    run: async (p) => {
      const spec = specOf(p.spec);
      const id = s(p, 'id');
      return { spec: { ...spec, nodes: spec.nodes.filter((x) => x.id !== id), edges: spec.edges.filter((e) => e.from !== id && e.to !== id) } };
    },
  },
  {
    id: 'dataPipelines.check', domain: 'dp', kind: 'query', label: 'Problems in a pipeline',
    description: 'Validates the spec: issues (message, step, node_id), count, valid, status_label / status_color for the toolbar chip, empty_text.',
    params: [{ name: 'spec', type: 'object' }, { name: 'is_new', type: 'boolean' }],
    fields: [{ name: 'issues', type: 'array' }, { name: 'valid', type: 'boolean' }, { name: 'status_label' }, { name: 'status_color' }],
    run: async (p) => {
      const spec = specOf(p.spec);
      const empty = spec.nodes.length === 0;
      const issues = empty ? [] : (await pipelinesApi.validate(spec)).issues;
      const label = (id?: string) => spec.nodes.find((n) => n.id === id)?.label || id || 'Pipeline';
      return {
        issues: issues.map((i, k) => ({ id: String(k), node_id: i.node_id ?? '', message: i.message, step: i.node_id ? label(i.node_id) : 'Pipeline' })),
        count: issues.length, valid: !empty && issues.length === 0,
        status_label: empty ? 'Empty' : issues.length === 0 ? 'Ready to run' : `${issues.length} problem${issues.length === 1 ? '' : 's'}`,
        status_color: empty ? 'default' : issues.length === 0 ? 'success' : 'warning',
        empty_text: empty ? 'Add a source to begin.' : 'No problems - preview it, then run it.',
        preview_tip: !empty && issues.length === 0 ? 'Run on the first 100 rows. Nothing is saved or written.' : 'Fix the problems first',
        run_tip: p.is_new === true || p.is_new === 'true' ? 'Save first' : !empty && issues.length === 0 ? 'Run the whole pipeline' : 'Fix the problems first',
        schedule_tip: p.is_new === true || p.is_new === 'true' ? 'Save first' : 'Run this pipeline on a schedule',
      };
    },
  },
  {
    id: 'dataPipelines.nodeView', domain: 'dp', kind: 'query', label: 'How each step shows on the canvas',
    description: 'Per step id: title, summary, icon, category, issues (messages), chips (in / out / rejected from the last preview).',
    params: [{ name: 'spec', type: 'object' }, { name: 'kinds', type: 'object' }, { name: 'issues', type: 'object' }, { name: 'preview', type: 'object' }],
    run: async (p) => {
      const spec = specOf(p.spec);
      const kinds = (Array.isArray(p.kinds) ? p.kinds : []) as NodeType[];
      const issues = (Array.isArray(p.issues) ? p.issues : []) as { node_id: string; message: string }[];
      const stats = Object.fromEntries(((p.preview as PreviewResult | null)?.summary?.Nodes ?? []).map((x) => [x.NodeID, x]));
      return Object.fromEntries(spec.nodes.map((n) => {
        const st = stats[n.id];
        return [n.id, {
          title: n.label || kinds.find((k) => k.type === n.type)?.label || n.type, summary: summarize(n), ...lookOf(n.type),
          issues: issues.filter((i) => i.node_id === n.id).map((i) => i.message),
          chips: st ? [
            { label: `in ${st.In}` }, { label: `out ${st.Out}`, color: 'success', variant: 'outlined' },
            ...(st.Errors > 0 ? [{ label: `${st.Errors} rejected`, color: 'error' }] : []),
            ...(st.Warnings > 0 ? [{ label: `${st.Warnings} warned`, color: 'warning' }] : []),
          ] : [],
        }];
      }));
    },
  },
  {
    id: 'dataPipelines.stepForm', domain: 'dp', kind: 'query', label: 'A step\'s settings form',
    description: 'id, type_label, fields (the form, shaped by step kind and what reaches the step), initial (the step as form values), issues.',
    params: [{ name: 'spec', type: 'object', required: true }, { name: 'id', type: 'string', required: true }, { name: 'info', type: 'object' },
      { name: 'kinds', type: 'object' }, { name: 'issues', type: 'object' }],
    run: async (p) => {
      const spec = specOf(p.spec);
      const n = spec.nodes.find((x) => x.id === s(p, 'id'));
      if (!n) return null;
      const info = (p.info ?? undefined) as StepInfo | undefined;
      const kinds = (Array.isArray(p.kinds) ? p.kinds : []) as NodeType[];
      const issues = ((Array.isArray(p.issues) ? p.issues : []) as { node_id: string; message: string }[]).filter((i) => i.node_id === n.id);
      return {
        id: n.id, type_label: kinds.find((k) => k.type === n.type)?.label ?? n.type,
        fields: [...issues.map((i, k) => note(`issue${k}`, i.message, 'error')), ...await stepFields(spec, n, info)],
        initial: toForm(n, info),
      };
    },
  },
  {
    id: 'dataPipelines.applyStep', domain: 'dp', kind: 'mutation', label: 'Apply a step\'s form', invalidates: [],
    description: 'The form\'s values onto the step. result: spec, form (after the rules a change of file or table brings).',
    params: [{ name: 'spec', type: 'object', required: true }, { name: 'id', type: 'string', required: true }, { name: 'form', type: 'object', required: true },
      { name: 'info', type: 'object' }],
    run: async (p) => {
      const spec = specOf(p.spec);
      return withStep(spec, fromForm(stepOf(spec, s(p, 'id')), (p.form ?? {}) as Form), (p.info ?? undefined) as StepInfo | undefined);
    },
  },
  {
    id: 'dataPipelines.setStep', domain: 'dp', kind: 'mutation', label: 'Change some of a step\'s settings', invalidates: [],
    params: [{ name: 'spec', type: 'object', required: true }, { name: 'id', type: 'string', required: true }, { name: 'patch', type: 'object', required: true }],
    run: async (p) => {
      const spec = specOf(p.spec);
      const n = stepOf(spec, s(p, 'id'));
      cache.delete('files'); // a new upload shows in the file list
      return withStep(spec, fromForm(n, { ...toForm(n), ...(p.patch as Form) }));
    },
  },
  {
    id: 'dataPipelines.upload', domain: 'dp', kind: 'mutation', label: 'Upload a file', invalidates: [],
    params: [{ name: 'file', type: 'object', required: true }],
    run: (p) => pipelinesApi.upload(p.file as File),
  },
  {
    id: 'dataPipelines.readFile', domain: 'dp', kind: 'mutation', label: 'Read a file source\'s columns', invalidates: [],
    description: 'Profiles the file: keeps columns the analyst tightened, adds newly seen ones. result: spec, form, info (row count, sample).',
    params: [{ name: 'spec', type: 'object', required: true }, { name: 'id', type: 'string', required: true }],
    run: async (p) => {
      const spec = specOf(p.spec);
      const n = stepOf(spec, s(p, 'id')) as SpecNode<'file_source'>;
      const cfg = n.config;
      const r = await pipelinesApi.profile({ uri: cfg.uri, format: cfg.format, delimiter: cfg.delimiter, has_header: cfg.has_header ?? true, count_rows: true });
      const known = new Map((cfg.columns ?? []).map((col) => [col.name, col]));
      const next = { ...n, config: { ...cfg, format: r.format as 'csv', columns: r.columns.map((col) => known.get(col.name) ?? col) } };
      const info: StepInfo = { id: n.id, row_count: r.row_count, sample: r.sample };
      return { ...withStep(spec, next, info), info };
    },
  },
  {
    id: 'dataPipelines.suggestMap', domain: 'dp', kind: 'mutation', label: 'Suggest a map step\'s mappings', invalidates: [],
    description: 'Adds confident suggestions for incoming fields not yet mapped. result: spec, form, info.note.',
    params: [{ name: 'spec', type: 'object', required: true }, { name: 'id', type: 'string', required: true }],
    run: async (p) => {
      const spec = specOf(p.spec);
      const n = stepOf(spec, s(p, 'id')) as SpecNode<'map'>;
      const schemas = await schemasFor(spec);
      const t = await targetsFor(downstreamSink(spec, n.id), schemas);
      const rows = n.config.fields ?? [];
      const sug = await pipelinesApi.suggestMapping(fieldsIn(spec, n.id, lookupFrom(schemas)), t.targets ?? []);
      const mapped = new Set(rows.map((r) => r.from));
      const add = sug.filter((x) => !mapped.has(x.from) && x.confidence >= 0.5).map((x) => ({ from: x.from, to: x.to, transform: x.transform || undefined }));
      const next = { ...n, config: { ...n.config, fields: [...rows, ...add] } };
      const info: StepInfo = { id: n.id, note: add.length ? `Added ${add.length} suggested mappings - check them before running.` : 'No new matches found.' };
      return { ...withStep(spec, next), info };
    },
  },
  {
    id: 'dataPipelines.save', domain: 'dp', kind: 'mutation', label: 'Save a pipeline',
    description: 'Creates it when new. result: id, created.',
    params: [{ name: 'id', type: 'string' }, { name: 'name', type: 'string', required: true }, { name: 'spec', type: 'object', required: true }],
    run: async (p) => {
      const id = s(p, 'id');
      const body = { name: s(p, 'name'), spec: specOf(p.spec) };
      const d = !id || id === 'new' ? await pipelinesApi.create(body) : await pipelinesApi.update(id, body);
      return { id: d.id, created: !id || id === 'new' };
    },
  },
  {
    id: 'dataPipelines.preview', domain: 'dp', kind: 'mutation', label: 'Preview a pipeline (nothing written)', invalidates: [],
    description: 'Runs the first 100 rows. result: the preview (an error is kept in it, not thrown).',
    params: [{ name: 'spec', type: 'object', required: true }],
    run: async (p) => {
      try { return await pipelinesApi.preview(specOf(p.spec), 100); } catch (e) { return { rejects: [], error: (e as Error).message }; }
    },
  },
  {
    id: 'dataPipelines.previewView', domain: 'dp', kind: 'query', label: 'A preview, shaped for its tab',
    description: 'state (none / error / done), totals (chips), steps (to pick one), rejects, sample_caption, columns + rows (cells by column).',
    params: [{ name: 'preview', type: 'object' }, { name: 'step', type: 'string' }, { name: 'spec', type: 'object' }],
    run: async (p) => previewView((p.preview ?? null) as PreviewResult | null, s(p, 'step'), specOf(p.spec)),
  },
  {
    id: 'dataPipelines.startRun', domain: 'dp', kind: 'mutation', label: 'Run a pipeline',
    description: 'Saves first when there are unsaved changes. result: run_id.',
    params: [{ name: 'id', type: 'string', required: true }, { name: 'name', type: 'string' }, { name: 'spec', type: 'object' }, { name: 'dirty', type: 'boolean' }],
    run: async (p) => {
      const id = s(p, 'id');
      if (p.dirty === true || p.dirty === 'true') await pipelinesApi.update(id, { name: s(p, 'name'), spec: specOf(p.spec) });
      return pipelinesApi.startRun(id);
    },
  },
  {
    id: 'dataPipelines.runs', domain: 'dp', kind: 'query', label: 'A pipeline\'s runs',
    description: 'Newest first: status (status_label, status_color), started, counts. active_done: the followed run has finished (done_text says how).',
    params: [{ name: 'id', type: 'string', required: true }, { name: 'active', type: 'string' }],
    run: async (p) => {
      const id = s(p, 'id');
      if (!id || id === 'new') return { runs: [], active_done: false };
      const runs = await pipelinesApi.runs(id);
      const a = runs.find((r) => r.id === s(p, 'active'));
      const done = !!a && a.status !== 'queued' && a.status !== 'running';
      return {
        runs: runs.map((r) => ({
          ...r, status_label: r.status.replace(/_/g, ' '), status_color: RUN_COLOR[r.status], started: new Date(r.start_time).toLocaleString(),
          counts: `${r.records_in} read · ${r.records_out} written · ${r.errors} rejected`,
        })),
        active_done: done,
        done_text: done ? (a!.status === 'failed' ? 'Run failed - see Runs' : `Run finished: ${a!.records_out} written, ${a!.errors} rejected`) : '',
      };
    },
  },
  {
    id: 'dataPipelines.runDetail', domain: 'dp', kind: 'query', label: 'One run in detail',
    description: 'steps (in / out / rejected / time / error), mastering (one line per mastering run it started), errors (sample lines).',
    params: [{ name: 'run', type: 'string', required: true }, { name: 'spec', type: 'object' }],
    run: async (p) => {
      const spec = specOf(p.spec);
      const label = (id?: string) => spec.nodes.find((n) => n.id === id)?.label || id || '';
      const d = await pipelinesApi.run(s(p, 'run'));
      const lang = window.location.pathname.split('/')[1] || 'en';
      return {
        steps: (d.steps ?? []).map((x) => ({
          id: x.NodeID, step: label(x.NodeID) || x.Label, in: x.In, out: x.Out, rejected: x.Errors, time: `${(x.Duration / 1e9).toFixed(1)}s`, error: x.Err,
        })),
        mastering: (d.outputs?.mastering ?? []).map((m) => ({
          id: m.run_id, label: `${label(m.node_id) || 'Master'}: ${m.status.toLowerCase()}`,
          color: m.status === 'FAILED' ? 'error' : m.status === 'PARTIAL' ? 'warning' : 'success',
          text: `${m.records} records · ${m.published} published · ${m.held_for_review} held · ${m.exceptions} exceptions${m.replayed ? ' (already mastered)' : ''}`,
          link_label: `Open ${m.entity} mastering`, href: `/${lang}/data/mastering`,
        })),
        errors: d.errors_sample.slice(0, 100).map((e, i) => ({
          id: String(i), text: e.run_error ?? `Row ${e.row} · ${label(e.node_id)} · ${e.reason}`, tone: e.run_error || e.kind === 'error' ? 'error' : 'warning',
        })),
      };
    },
  },
];

registerOperations(operations);
