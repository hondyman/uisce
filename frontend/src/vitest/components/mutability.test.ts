import { describe, expect, it } from 'vitest';
import { getFieldTierTag } from '@/types/mutability';
import type {
  ResolvedCapabilities,
  ResolvedField,
} from '@/types/mutability';

const baseField: ResolvedField = {
  semanticTermKey: 'k',
  displayLabel: 'K',
  hydrationState: 'RESOLVED',
  isEditable: true,
};

const baseCap: ResolvedCapabilities = {
  mutabilityMode: 'DIRECT_OLTP_SQL',
  temporalStrategy: 'NONE',
  allowDirectCrudFormButtons: true,
  activeRouteBackendId: 'pg_postgres_oltp',
};

describe('getFieldTierTag', () => {
  it('returns "oltp" for pg backend', () => {
    expect(getFieldTierTag(baseField, baseCap)).toBe('oltp');
  });

  it('returns "hot" for starrocks backend', () => {
    const cap: ResolvedCapabilities = {
      ...baseCap,
      activeRouteBackendId: 'sr_starrocks_hot',
    };
    expect(getFieldTierTag(baseField, cap)).toBe('hot');
  });

  it('returns "cold" for iceberg backend', () => {
    const cap: ResolvedCapabilities = {
      ...baseCap,
      activeRouteBackendId: 'ib_iceberg_coldstore',
    };
    expect(getFieldTierTag(baseField, cap)).toBe('cold');
  });

  it('prefers the field-level bindingBackendId over capabilities', () => {
    const field: ResolvedField = {
      ...baseField,
      bindingBackendId: 'sr_starrocks_hot',
    };
    const cap: ResolvedCapabilities = {
      ...baseCap,
      activeRouteBackendId: 'pg_postgres_oltp',
    };
    expect(getFieldTierTag(field, cap)).toBe('hot');
  });

  it('returns tier even when hydrationState is UNBOUND_FALLBACK_NULL', () => {
    expect(
      getFieldTierTag(
        { ...baseField, hydrationState: 'UNBOUND_FALLBACK_NULL', bindingBackendId: 'sr_starrocks_hot' },
        baseCap,
      ),
    ).toBe('hot');
  });
});