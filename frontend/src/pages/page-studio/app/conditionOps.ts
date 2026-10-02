import type { ConditionNode } from './appModel';

/**
 * What the rule engine's operators mean for a page condition: which kinds of
 * value each fits, what it needs to be given, and how to put a condition
 * into words. Pure - the condition builder (UI) and the page checker both
 * read it, so what the designer offers and what publish accepts agree.
 * The operators and their value shapes mirror backend/internal/rules/vm
 * (condition_operators.go): between is an inclusive [low, high]; in /
 * contains_* take a list; length_* a number; date operators read dates.
 */

export type FieldType = 'text' | 'number' | 'yesno' | 'list' | 'object' | 'date' | 'unknown';
/** none: nothing; one: a value; number: a number; list: several; range: low and high. */
export type ValueKind = 'none' | 'one' | 'number' | 'list' | 'range';

export interface OperatorInfo {
  value: string;
  label: string;
  kind: ValueKind;
  /** Field types it fits; 'any' fits all. */
  fits: FieldType[] | 'any';
  /** What a sample of the field holds / what the operator is given / what the engine answers - proves the catalogue against the engine in tests. */
  sample: { field: unknown; value?: unknown; secondValue?: unknown; result: boolean };
}

/** Date samples are "now" so the relative operators (today, this week...) hold whenever the test runs. */
const NOW = new Date().toISOString();

export const OPERATORS: OperatorInfo[] = [
  { value: 'equals', label: 'equals', kind: 'one', fits: 'any', sample: { field: 'a', value: 'a', result: true } },
  { value: 'not_equals', label: 'does not equal', kind: 'one', fits: 'any', sample: { field: 'a', value: 'b', result: true } },
  { value: 'in', label: 'is one of', kind: 'list', fits: ['text', 'number'], sample: { field: 'a', value: ['a', 'b'], result: true } },
  { value: 'not_in', label: 'is none of', kind: 'list', fits: ['text', 'number'], sample: { field: 'c', value: ['a', 'b'], result: true } },
  { value: 'contains', label: 'contains', kind: 'one', fits: ['text', 'list'], sample: { field: 'hello', value: 'ell', result: true } },
  { value: 'not_contains', label: 'does not contain', kind: 'one', fits: ['text', 'list'], sample: { field: 'hello', value: 'xyz', result: true } },
  { value: 'starts_with', label: 'starts with', kind: 'one', fits: ['text'], sample: { field: 'hello', value: 'he', result: true } },
  { value: 'ends_with', label: 'ends with', kind: 'one', fits: ['text'], sample: { field: 'hello', value: 'lo', result: true } },
  { value: 'matches_regex', label: 'matches the pattern', kind: 'one', fits: ['text'], sample: { field: 'abc123', value: '^[a-z]+\\d+$', result: true } },
  { value: 'greater_than', label: 'is more than', kind: 'one', fits: ['number'], sample: { field: 5, value: 3, result: true } },
  { value: 'greater_equal', label: 'is at least', kind: 'one', fits: ['number'], sample: { field: 5, value: 5, result: true } },
  { value: 'less_than', label: 'is less than', kind: 'one', fits: ['number'], sample: { field: 3, value: 5, result: true } },
  { value: 'less_equal', label: 'is at most', kind: 'one', fits: ['number'], sample: { field: 5, value: 5, result: true } },
  { value: 'between', label: 'is between', kind: 'range', fits: ['number'], sample: { field: 5, value: 1, secondValue: 9, result: true } },
  { value: 'not_between', label: 'is outside', kind: 'range', fits: ['number'], sample: { field: 12, value: 1, secondValue: 9, result: true } },
  { value: 'is_positive', label: 'is positive', kind: 'none', fits: ['number'], sample: { field: 5, result: true } },
  { value: 'is_negative', label: 'is negative', kind: 'none', fits: ['number'], sample: { field: -5, result: true } },
  { value: 'is_zero', label: 'is zero', kind: 'none', fits: ['number'], sample: { field: 0, result: true } },
  { value: 'is_empty', label: 'is empty', kind: 'none', fits: 'any', sample: { field: '', result: true } },
  { value: 'is_not_empty', label: 'is not empty', kind: 'none', fits: 'any', sample: { field: 'x', result: true } },
  { value: 'is_null', label: 'is not set', kind: 'none', fits: 'any', sample: { field: null, result: true } },
  { value: 'is_not_null', label: 'is set', kind: 'none', fits: 'any', sample: { field: 'x', result: true } },
  { value: 'is_true', label: 'is true', kind: 'none', fits: ['yesno'], sample: { field: true, result: true } },
  { value: 'is_false', label: 'is false', kind: 'none', fits: ['yesno'], sample: { field: false, result: true } },
  { value: 'length_equals', label: 'has length', kind: 'number', fits: ['text', 'list'], sample: { field: 'abc', value: 3, result: true } },
  { value: 'length_greater', label: 'is longer than', kind: 'number', fits: ['text', 'list'], sample: { field: 'abcd', value: 3, result: true } },
  { value: 'length_less', label: 'is shorter than', kind: 'number', fits: ['text', 'list'], sample: { field: 'ab', value: 3, result: true } },
  { value: 'contains_any', label: 'contains any of', kind: 'list', fits: ['list', 'text'], sample: { field: ['a', 'b'], value: ['b', 'z'], result: true } },
  { value: 'contains_all', label: 'contains all of', kind: 'list', fits: ['list', 'text'], sample: { field: ['a', 'b'], value: ['a', 'b'], result: true } },
  { value: 'before', label: 'is before', kind: 'one', fits: ['date'], sample: { field: '2026-01-01', value: '2026-06-01', result: true } },
  { value: 'after', label: 'is after', kind: 'one', fits: ['date'], sample: { field: '2026-06-01', value: '2026-01-01', result: true } },
  { value: 'on_or_before', label: 'is on or before', kind: 'one', fits: ['date'], sample: { field: '2026-01-01', value: '2026-01-01', result: true } },
  { value: 'is_today', label: 'is today', kind: 'none', fits: ['date'], sample: { field: NOW, result: true } },
  { value: 'is_this_week', label: 'is this week', kind: 'none', fits: ['date'], sample: { field: NOW, result: true } },
  { value: 'is_this_month', label: 'is this month', kind: 'none', fits: ['date'], sample: { field: NOW, result: true } },
  { value: 'in_last_n_days', label: 'is within the last (days)', kind: 'number', fits: ['date'], sample: { field: NOW, value: 7, result: true } },
];

const BY_NAME = new Map(OPERATORS.map((o) => [o.value, o]));
export const operatorInfo = (name: string) => BY_NAME.get(name);

const DATE = /^\d{4}-\d{2}-\d{2}([T ].*)?$/;

/** The kind of a field, read from a value it holds (empty or missing = unknown: offer everything). */
export function fieldTypeOf(v: unknown): FieldType {
  if (v === null || v === undefined) return 'unknown';
  if (Array.isArray(v)) return 'list';
  if (typeof v === 'object') return 'object';
  if (typeof v === 'boolean') return 'yesno';
  if (typeof v === 'number') return 'number';
  if (typeof v === 'string') return DATE.test(v) ? 'date' : 'text';
  return 'unknown';
}

export const TYPE_LABEL: Record<FieldType, string> = {
  text: 'text', number: 'number', yesno: 'yes/no', list: 'list', object: 'object', date: 'date', unknown: 'not loaded yet',
};

/** The operators that fit a field kind; an unknown kind fits all of them. */
export function operatorsFor(type: FieldType): OperatorInfo[] {
  if (type === 'unknown') return OPERATORS;
  return OPERATORS.filter((o) => o.fits === 'any' || o.fits.includes(type));
}

export const fits = (op: OperatorInfo, type: FieldType) => type === 'unknown' || op.fits === 'any' || op.fits.includes(type);

export type Leaf = Extract<ConditionNode, { type: 'condition' }>;
export type Group = Extract<ConditionNode, { type: 'group' }>;

/** A value typed in a box, as the field it is compared with holds values. */
export function coerceValue(raw: string, kind: ValueKind, type: FieldType): unknown {
  if (kind === 'list') return raw.split(',').map((x) => x.trim()).filter(Boolean);
  const numeric = raw.trim() !== '' && !Number.isNaN(Number(raw));
  if (kind === 'number' || kind === 'range') return numeric ? Number(raw) : raw;
  if (type === 'text' || type === 'date') return raw;
  if (type === 'number') return numeric ? Number(raw) : raw;
  if (type === 'yesno') return raw === 'true' ? true : raw === 'false' ? false : raw;
  // Not known yet: read it as most people mean it.
  if (raw === 'true') return true;
  if (raw === 'false') return false;
  return numeric ? Number(raw) : raw;
}

export const showValue = (v: unknown) => (Array.isArray(v) ? v.join(', ') : v === undefined || v === null ? '' : String(v));

// --- what is wrong with a condition ----------------------------------------------------------

export interface ConditionProblem {
  severity: 'error' | 'warning';
  message: string;
}

const blank = (v: unknown) => v === undefined || v === null || v === '' || (Array.isArray(v) && v.length === 0);

/** What is wrong with a condition by itself: no field, an operator the engine lacks, a missing or malformed value. */
export function shapeProblems(leaf: Leaf): ConditionProblem[] {
  const out: ConditionProblem[] = [];
  if (!leaf.field) out.push({ severity: 'error', message: 'Choose what to check.' });
  const op = operatorInfo(leaf.operator);
  if (!op) {
    out.push({ severity: 'error', message: `The rule engine has no operator "${leaf.operator}".` });
    return out;
  }
  if (op.kind === 'one' && blank(leaf.value)) out.push({ severity: 'error', message: `"${op.label}" needs a value.` });
  if (op.kind === 'list' && !(Array.isArray(leaf.value) ? leaf.value.length : !blank(leaf.value))) out.push({ severity: 'error', message: `"${op.label}" needs at least one value.` });
  if (op.kind === 'number' && (blank(leaf.value) || Number.isNaN(Number(leaf.value)))) out.push({ severity: 'error', message: `"${op.label}" needs a number.` });
  if (op.kind === 'range') {
    const [lo, hi] = [leaf.value, leaf.secondValue];
    if (blank(lo) || blank(hi)) out.push({ severity: 'error', message: `"${op.label}" needs a low and a high number.` });
    else if (Number.isNaN(Number(lo)) || Number.isNaN(Number(hi))) out.push({ severity: 'error', message: 'Both ends of the range must be numbers.' });
    else if (Number(lo) > Number(hi)) out.push({ severity: 'warning', message: 'The low end is above the high end, so nothing can match.' });
  }
  if (op.value === 'matches_regex' && typeof leaf.value === 'string') {
    try { new RegExp(leaf.value); } catch { out.push({ severity: 'error', message: 'That pattern is not a valid regular expression.' }); }
  }
  return out;
}

// --- in words ---------------------------------------------------------------------------------

const valueWords = (leaf: Leaf, op?: OperatorInfo) => {
  if (!op || op.kind === 'none') return '';
  if (op.kind === 'range') return `${showValue(leaf.value)} and ${showValue(leaf.secondValue)}`;
  if (op.kind === 'list') return `(${showValue(leaf.value)})`;
  return showValue(leaf.value);
};

/** "vars.tab is not empty and (queries.x.data.count is more than 0 or vars.open is true)". */
export function describeCondition(node: ConditionNode | undefined, top = true): string {
  if (!node) return 'always';
  if (node.type === 'condition') {
    const op = operatorInfo(node.operator);
    return [node.field || '…', op?.label ?? node.operator, valueWords(node, op)].filter(Boolean).join(' ');
  }
  const parts = node.conditions.map((c) => describeCondition(c, false));
  if (parts.length === 0) return 'always';
  const joined = parts.join(node.operator === 'AND' ? ' and ' : ' or ');
  return top || parts.length === 1 ? joined : `(${joined})`;
}

/** A group of one at the top is just that condition; an empty group is no condition. */
export function simplify(node: ConditionNode | undefined): ConditionNode | undefined {
  if (!node || node.type === 'condition') return node;
  if (node.conditions.length === 0) return undefined;
  if (node.conditions.length === 1 && node.conditions[0].type === 'condition') return node.conditions[0];
  return node;
}

/** Every condition inside a node, depth first. */
export function leavesOf(node: ConditionNode | undefined): Leaf[] {
  if (!node) return [];
  return node.type === 'condition' ? [node] : node.conditions.flatMap((c) => leavesOf(c));
}
