import { beforeAll, describe, expect, it } from 'vitest';
import { evaluateCondition } from '../../pages/page-studio/app/conditions';
import type { ConditionNode } from '../../pages/page-studio/app/appModel';
import { loadRuleEngine } from './loadRuleEngine';

/**
 * Page conditions run on the real rule engine (public/rule_engine.wasm, built
 * from backend/internal/rules/vm) - no stub - so these prove the shapes the
 * studio writes are ones the one engine actually answers the way pages need.
 */
beforeAll(loadRuleEngine, 30000);

const c = (field: string, operator: string, value?: unknown): ConditionNode => ({ type: 'condition', field, operator, value });

describe('page conditions on the rule engine', () => {
  it('record-only tabs show while the profile is loading and for records, not for time series', async () => {
    // Not `series not_equals true`: the engine answers null != true as false.
    const RECORD: ConditionNode = { type: 'group', operator: 'OR', conditions: [c('queries.profile.data', 'is_empty'), c('queries.profile.data.series', 'is_false')] };
    expect(await evaluateCondition(RECORD, { queries: { profile: { data: undefined } } })).toBe(true);
    expect(await evaluateCondition(RECORD, { queries: { profile: { data: { series: false } } } })).toBe(true);
    expect(await evaluateCondition(RECORD, { queries: { profile: { data: { series: true } } } })).toBe(false);
  });

  it('time-series tabs show only once the profile says series', async () => {
    const SERIES = c('queries.profile.data.series', 'is_true');
    expect(await evaluateCondition(SERIES, { queries: { profile: { data: undefined } } })).toBe(false);
    expect(await evaluateCondition(SERIES, { queries: { profile: { data: { series: true } } } })).toBe(true);
  });

  it('row states gate row buttons', async () => {
    const canVote = c('row.state', 'equals', 'can_vote');
    expect(await evaluateCondition(canVote, { row: { state: 'can_vote' } })).toBe(true);
    expect(await evaluateCondition(canVote, { row: { state: 'mine' } })).toBe(false);
  });

  it('empty / not empty treat absent values as empty', async () => {
    expect(await evaluateCondition(c('vars.lastRun', 'is_not_empty'), { vars: { lastRun: null } })).toBe(false);
    expect(await evaluateCondition(c('vars.lastRun', 'is_not_empty'), { vars: { lastRun: { status: 'COMPLETED' } } })).toBe(true);
    expect(await evaluateCondition(c('row.golden_id', 'is_empty'), { row: {} })).toBe(true);
    expect(await evaluateCondition(c('queries.profile.data', 'is_empty'), { queries: {} })).toBe(true);
  });

  it('OR groups: the exceptions actions column shows for open filters', async () => {
    const open: ConditionNode = { type: 'group', operator: 'OR', conditions: [c('vars.exceptionStatus', 'is_empty'), c('vars.exceptionStatus', 'in', ['OPEN', 'IN_REVIEW'])] };
    expect(await evaluateCondition(open, { vars: { exceptionStatus: '' } })).toBe(true);
    expect(await evaluateCondition(open, { vars: { exceptionStatus: 'IN_REVIEW' } })).toBe(true);
    expect(await evaluateCondition(open, { vars: { exceptionStatus: 'RESOLVED' } })).toBe(false);
  });

  it('counts compare as numbers', async () => {
    const none = c('queries.profiles.data.length', 'equals', 0);
    expect(await evaluateCondition(none, { queries: { profiles: { data: [] } } })).toBe(true);
    expect(await evaluateCondition(none, { queries: { profiles: { data: [{}] } } })).toBe(false);
    expect(await evaluateCondition(none, { queries: { profiles: {} } })).toBe(false);
  });
});
