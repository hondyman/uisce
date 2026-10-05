import { describe, it, expect } from 'vitest';
import {
  assertCubeSubjectMirror,
  isNumericCubePin,
} from '../../../features/analytical-subject/subjectPin';
import { cubeSubject } from '../../../features/analytical-subject/types';

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
