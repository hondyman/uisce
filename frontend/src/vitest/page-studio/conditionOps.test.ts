import { beforeAll, describe, expect, it } from 'vitest';
import { loadRuleEngine } from './loadRuleEngine';
import { evaluateRuleWasm } from '../../rules/wasmRuntime';
import type { ConditionNode } from '../../pages/page-studio/app/appModel';
import {
  OPERATORS, coerceValue, describeCondition, fieldTypeOf, operatorsFor, shapeProblems, simplify, type Leaf,
} from '../../pages/page-studio/app/conditionOps';

/**
 * The operator catalogue the condition builder offers, proved against the
 * real rule engine: every operator exists there and answers its sample the
 * way the catalogue says; the helpers that pick, type, validate and describe
 * conditions behave.
 */

beforeAll(loadRuleEngine, 30000);

describe('the catalogue matches the engine', () => {
  for (const op of OPERATORS) {
    it(`${op.value} answers its sample`, async () => {
      const node = { type: 'condition', field: 'x', operator: op.value, value: op.sample.value, secondValue: op.sample.secondValue };
      await expect(evaluateRuleWasm(node, { x: op.sample.field })).resolves.toBe(op.sample.result);
    });
  }
  it('an operator the engine lacks is an error, not a silent false', async () => {
    await expect(evaluateRuleWasm({ type: 'condition', field: 'x', operator: 'sounds_like', value: 'a' }, { x: 'a' })).rejects.toBeTruthy();
  });
});

describe('field kinds and the operators that fit', () => {
  it('reads a field\'s kind from what it holds', () => {
    expect(['a', 1, true, [1], { a: 1 }, null, '2026-09-30'].map(fieldTypeOf)).toEqual(['text', 'number', 'yesno', 'list', 'object', 'unknown', 'date']);
  });
  it('offers number comparisons for numbers, truth tests for yes/no, everything while unknown', () => {
    const names = (t: Parameters<typeof operatorsFor>[0]) => operatorsFor(t).map((o) => o.value);
    expect(names('number')).toContain('between');
    expect(names('number')).not.toContain('starts_with');
    expect(names('yesno')).toEqual(['equals', 'not_equals', 'is_empty', 'is_not_empty', 'is_null', 'is_not_null', 'is_true', 'is_false']);
    expect(names('unknown')).toHaveLength(OPERATORS.length);
    expect(names('date')).toContain('before');
  });
});

describe('values are typed as the field holds them', () => {
  it('keeps text as text, reads numbers and flags where the field is one', () => {
    expect(coerceValue('007', 'one', 'text')).toBe('007');
    expect(coerceValue('7', 'one', 'number')).toBe(7);
    expect(coerceValue('true', 'one', 'yesno')).toBe(true);
    expect(coerceValue('a, b ,', 'list', 'text')).toEqual(['a', 'b']);
    expect(coerceValue('3', 'range', 'unknown')).toBe(3);
    expect(coerceValue('7', 'one', 'unknown')).toBe(7);
  });
});

describe('what is wrong with a condition', () => {
  const leaf = (o: Partial<Leaf>): Leaf => ({ type: 'condition', field: 'vars.x', operator: 'equals', ...o });
  const msgs = (l: Leaf) => shapeProblems(l).map((p) => `${p.severity}: ${p.message}`);
  it('wants a field, a known operator and the value the operator takes', () => {
    expect(msgs(leaf({ field: '' }))).toEqual(['error: Choose what to check.', 'error: "equals" needs a value.']);
    expect(msgs(leaf({ operator: 'sounds_like' }))).toEqual(['error: The rule engine has no operator "sounds_like".']);
    expect(msgs(leaf({ operator: 'in', value: [] }))).toEqual(['error: "is one of" needs at least one value.']);
    expect(msgs(leaf({ operator: 'between', value: 1 }))).toEqual(['error: "is between" needs a low and a high number.']);
    expect(msgs(leaf({ operator: 'between', value: 9, secondValue: 1 }))).toEqual(['warning: The low end is above the high end, so nothing can match.']);
    expect(msgs(leaf({ operator: 'length_equals', value: 'abc' }))).toEqual(['error: "has length" needs a number.']);
    expect(msgs(leaf({ operator: 'matches_regex', value: '(' }))).toEqual(['error: That pattern is not a valid regular expression.']);
  });
  it('is quiet when it is complete', () => {
    expect(msgs(leaf({ operator: 'is_true' }))).toEqual([]);
    expect(msgs(leaf({ operator: 'between', value: 1, secondValue: 9 }))).toEqual([]);
    expect(msgs(leaf({ value: 'x' }))).toEqual([]);
  });
});

describe('in words, and tidied', () => {
  const a = { type: 'condition', field: 'vars.tab', operator: 'equals', value: 'runs' } as const;
  const b = { type: 'condition', field: 'queries.q.data.count', operator: 'greater_than', value: 0 } as const;
  const c = { type: 'condition', field: 'vars.open', operator: 'is_true' } as const;
  it('reads a nested condition as a sentence', () => {
    expect(describeCondition(undefined)).toBe('always');
    expect(describeCondition({ type: 'group', operator: 'AND', conditions: [a, { type: 'group', operator: 'OR', conditions: [b, c] }] }))
      .toBe('vars.tab equals runs and (queries.q.data.count is more than 0 or vars.open is true)');
    expect(describeCondition({ type: 'condition', field: 'row.n', operator: 'between', value: 1, secondValue: 9 })).toBe('row.n is between 1 and 9');
  });
  it('turns a group of one into that condition and an empty group into none', () => {
    expect(simplify({ type: 'group', operator: 'AND', conditions: [a] })).toEqual(a);
    expect(simplify({ type: 'group', operator: 'OR', conditions: [] })).toBeUndefined();
    const two: ConditionNode = { type: 'group', operator: 'AND', conditions: [a, c] };
    expect(simplify(two)).toBe(two);
  });
});
