import { describe, it, expect } from 'vitest';
import {
  assertCubeSubjectMirror,
  isNumericCubePin,
  subjectFromSavedQuery,
  savedQueryBindProps,
} from '../../../features/analytical-subject/subjectPin';
import { businessObjectSubject, cubeSubject } from '../../../features/analytical-subject/types';

describe('isNumericCubePin', () => {
  it('accepts positive numeric contractVersion', () => {
    expect(isNumericCubePin(cubeSubject('c1', 3))).toBe(true);
  });
  it('rejects latest and non-cube', () => {
    expect(isNumericCubePin(cubeSubject('c1', 'latest'))).toBe(false);
    expect(isNumericCubePin({ kind: 'business_object', boId: 'b', bindingId: 'x' })).toBe(false);
  });
});

describe('assertCubeSubjectMirror', () => {
  it('passes when page has no cube mirror', () => {
    expect(assertCubeSubjectMirror(null, { sourceKind: 'business_object', boId: 'bo' }).ok).toBe(true);
  });

  it('passes when mirror matches saved subject', () => {
    const pin = cubeSubject('cube-1', 2);
    const check = assertCubeSubjectMirror(pin, { subject: pin, sourceKind: 'cube', boId: 'cube-1' });
    expect(check).toEqual({ ok: true });
  });

  it('fails on cubeId mismatch', () => {
    const check = assertCubeSubjectMirror(cubeSubject('a', 1), {
      subject: cubeSubject('b', 1),
      sourceKind: 'cube',
      boId: 'b',
    });
    expect(check.ok).toBe(false);
    if (!check.ok) expect(check.reason).toMatch(/cubeId mismatch/);
  });

  it('fails on contractVersion drift (numeric vs latest)', () => {
    const check = assertCubeSubjectMirror(cubeSubject('c1', 1), {
      subject: cubeSubject('c1', 'latest'),
      sourceKind: 'cube',
      boId: 'c1',
    });
    expect(check.ok).toBe(false);
    if (!check.ok) expect(check.reason).toMatch(/contractVersion mismatch/);
  });

  it('fails when page pins cube but saved query is BO', () => {
    const check = assertCubeSubjectMirror(cubeSubject('c1', 1), {
      sourceKind: 'business_object',
      boId: 'bo-1',
    });
    expect(check.ok).toBe(false);
    if (!check.ok) expect(check.reason).toMatch(/not cube-backed/);
  });

  it('synthesizes saved cube from sourceKind when subject omitted', () => {
    const check = assertCubeSubjectMirror(cubeSubject('c1', 'latest'), {
      sourceKind: 'cube',
      boId: 'c1',
    });
    expect(check).toEqual({ ok: true });
  });
});

describe('subjectFromSavedQuery / savedQueryBindProps (PR1b)', () => {
  it('prefers explicit cube subject', () => {
    const pin = cubeSubject('cube-9', 4);
    expect(subjectFromSavedQuery({ subject: pin, sourceKind: 'cube', boId: 'cube-9' })).toEqual(pin);
  });

  it('falls back to state.subject', () => {
    const pin = cubeSubject('cube-2', 'latest');
    expect(subjectFromSavedQuery({ state: { subject: pin }, boId: 'other' })).toEqual(pin);
  });

  it('synthesizes cube from sourceKind + boId', () => {
    expect(subjectFromSavedQuery({ sourceKind: 'cube', boId: 'c-abc' })).toEqual(
      cubeSubject('c-abc', 'latest'),
    );
  });

  it('synthesizes BO subject from boId + bindingId', () => {
    expect(subjectFromSavedQuery({ boId: 'bo-1', bindingId: 'bind-1', relatedBoIds: ['r1'] })).toEqual(
      businessObjectSubject('bo-1', 'bind-1', ['r1']),
    );
  });

  it('bind props always write subject beside savedQueryId', () => {
    const pin = cubeSubject('c1', 2);
    expect(
      savedQueryBindProps({
        id: 'sq-1',
        subject: pin,
        sourceKind: 'cube',
        boId: 'c1',
      }),
    ).toEqual({
      savedQueryId: 'sq-1',
      subject: pin,
      savedQueryParams: undefined,
    });
  });

  it('unbind clears savedQueryId and subject', () => {
    expect(savedQueryBindProps(null)).toEqual({
      savedQueryId: undefined,
      subject: undefined,
      savedQueryParams: undefined,
    });
  });
});
