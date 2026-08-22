import { describe, it, expect } from 'vitest';
import { dedupeFields } from '../../utils/dedupeFields';

describe('dedupeFields', () => {
  it('returns empty array for empty input', () => {
    expect(dedupeFields([])).toEqual([]);
  });

  it('returns same array when no duplicates', () => {
    const input = [
      { key: 'account_id', name: 'Account ID' },
      { key: 'account_name', name: 'Account Name' },
      { id: 'uuid-1', name: 'Some Field' },
    ];
    expect(dedupeFields(input)).toEqual(input);
  });

  it('dedupes by key when present', () => {
    const input = [
      { key: 'account_id', name: 'Account ID' },
      { key: 'account_id', name: 'Account ID (duplicate key)' },
      { key: 'account_name', name: 'Account Name' },
    ];
    const result = dedupeFields(input);
    expect(result).toHaveLength(2);
    expect(result.map((f) => f.key)).toEqual(['account_id', 'account_name']);
  });

  it('falls back to technicalName when key is absent', () => {
    const input = [
      { technicalName: 'account_id', name: 'Account ID' },
      { technicalName: 'account_id', name: 'Account ID (duplicate)' },
      { technicalName: 'account_name', name: 'Account Name' },
    ];
    const result = dedupeFields(input);
    expect(result).toHaveLength(2);
    expect(result.map((f) => f.technicalName)).toEqual(['account_id', 'account_name']);
  });

  it('falls back to id when key and technicalName are absent', () => {
    const input = [
      { id: 'uuid-1', name: 'Field One' },
      { id: 'uuid-1', name: 'Field One (duplicate id)' },
      { id: 'uuid-2', name: 'Field Two' },
    ];
    const result = dedupeFields(input);
    expect(result).toHaveLength(2);
    expect(result.map((f) => f.id)).toEqual(['uuid-1', 'uuid-2']);
  });

  it('skips fields with no key, technicalName, or id', () => {
    const input = [
      { key: 'account_id', name: 'Account ID' },
      { name: 'No stable identifier' },
      { id: 'uuid-1', name: 'Field With ID' },
    ];
    const result = dedupeFields(input);
    expect(result).toHaveLength(2);
    expect(result.map((f) => f.key || f.id)).toEqual(['account_id', 'uuid-1']);
  });

  it('skips all fields when none have stable identifiers', () => {
    const input = [
      { name: 'Just a name' },
      { displayName: 'Just a displayName' },
    ];
    expect(dedupeFields(input)).toEqual([]);
  });

  it('prefers key over technicalName over id (key wins when all present)', () => {
    const input = [
      { key: 'primary_key', technicalName: 'shared_tech', id: 'shared-id', name: 'A' },
      { key: 'different_key', technicalName: 'shared_tech', id: 'shared-id', name: 'B' },
    ];
    const result = dedupeFields(input);
    expect(result).toHaveLength(2);
    expect(result[0].key).toBe('primary_key');
    expect(result[1].key).toBe('different_key');
  });

  it('dedupes across different identifier types (key vs id)', () => {
    const input = [
      { key: 'account_id', name: 'By Key' },
      { id: 'account_id', name: 'By ID (same identifier, different field)' },
    ];
    const result = dedupeFields(input);
    expect(result).toHaveLength(1);
    expect(result[0].key).toBe('account_id');
  });

  it('preserves first-seen occurrence order', () => {
    const input = [
      { key: 'z_field', name: 'Z' },
      { key: 'a_field', name: 'A' },
      { key: 'm_field', name: 'M' },
      { key: 'a_field', name: 'A Duplicate' },
      { key: 'z_field', name: 'Z Duplicate' },
    ];
    const result = dedupeFields(input);
    expect(result).toHaveLength(3);
    expect(result.map((f) => f.key)).toEqual(['z_field', 'a_field', 'm_field']);
  });

  it('works with generic T type', () => {
    interface RawField { key?: string; technicalName?: string; id?: string; label: string }
    const input: RawField[] = [
      { key: 'x', label: 'X' },
      { key: 'x', label: 'X Dup' },
      { technicalName: 'y', label: 'Y' },
      { technicalName: 'y', label: 'Y Dup' },
    ];
    const result = dedupeFields(input);
    expect(result).toHaveLength(2);
  });
});
