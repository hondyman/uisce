import { describe, it, expect } from 'vitest';
import { mapFieldToControlType } from '../../../components/pagestudio/pageStudioTypes';

describe('mapFieldToControlType', () => {
  it('maps boolean-ish types to switch', () => {
    expect(mapFieldToControlType('boolean')).toBe('switch');
    expect(mapFieldToControlType('bool')).toBe('switch');
  });

  it('maps datetime/timestamp to datetime', () => {
    expect(mapFieldToControlType('timestamp')).toBe('datetime');
    expect(mapFieldToControlType('datetime')).toBe('datetime');
  });

  it('maps date to date', () => {
    expect(mapFieldToControlType('date')).toBe('date');
  });

  it('maps numeric-ish types to number', () => {
    expect(mapFieldToControlType('decimal')).toBe('number');
    expect(mapFieldToControlType('currency')).toBe('number');
    expect(mapFieldToControlType('int')).toBe('number');
  });

  it('maps code/lookup/status/type/enum to select', () => {
    expect(mapFieldToControlType('code')).toBe('select');
    expect(mapFieldToControlType('lookup')).toBe('select');
    expect(mapFieldToControlType('status')).toBe('select');
    expect(mapFieldToControlType('enum')).toBe('select');
  });

  it('defaults strings/unknown/missing to text', () => {
    expect(mapFieldToControlType('string')).toBe('text');
    expect(mapFieldToControlType('id')).toBe('text');
    expect(mapFieldToControlType(undefined)).toBe('text');
    expect(mapFieldToControlType(null)).toBe('text');
  });
});
