import { describe, it, expect } from 'vitest';
import { isToMany } from '../../../components/pagestudio/pageStudioTypes';
import { isToMany as sharedIsToMany, normalizeCardinality } from '../../../types/cardinality';

describe('isToMany', () => {
  it('treats 1:M, M:M, and the legacy 1:N convention as to-many', () => {
    expect(isToMany('1:M')).toBe(true);
    expect(isToMany('M:M')).toBe(true);
    expect(isToMany('1:N')).toBe(true);
    expect(isToMany('1:m')).toBe(true); // case-insensitive
  });

  it('treats 1:1 and M:1 as to-one', () => {
    expect(isToMany('1:1')).toBe(false);
    expect(isToMany('M:1')).toBe(false);
  });

  it('defaults unrecognized or missing values to to-one', () => {
    expect(isToMany('')).toBe(false);
    expect(isToMany(undefined)).toBe(false);
    expect(isToMany(null)).toBe(false);
    expect(isToMany('MANY')).toBe(false);
  });
});

describe('shared cardinality module', () => {
  it('page designer isToMany is the shared implementation', () => {
    expect(isToMany).toBe(sharedIsToMany);
  });

  it('normalizes backend canonical (ONE_TO_ONE-style) values', () => {
    expect(normalizeCardinality('ONE_TO_ONE')).toBe('1:1');
    expect(normalizeCardinality('ONE_TO_MANY')).toBe('1:M');
    expect(normalizeCardinality('MANY_TO_ONE')).toBe('M:1');
    expect(normalizeCardinality('MANY_TO_MANY')).toBe('M:M');
  });

  it('correctly identifies the 1:1 and M:M cases the old backend heuristic never produced', () => {
    expect(sharedIsToMany('1:1')).toBe(false);
    expect(sharedIsToMany('ONE_TO_ONE')).toBe(false);
    expect(sharedIsToMany('M:M')).toBe(true);
    expect(sharedIsToMany('MANY_TO_MANY')).toBe(true);
    expect(sharedIsToMany('N:M')).toBe(true);
  });
});
