import { describe, it, expect } from 'vitest';
import { isToMany } from '../../../components/pagestudio/pageStudioTypes';

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
