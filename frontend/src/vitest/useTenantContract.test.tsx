/**
 * Pins the two halves of the `useTenant` contract against each other, so they stay
 * legible instead of one quietly eroding the other.
 *
 *   1. provider MISSING            -> throws. Deliberate and loud (see 1d77f82c8).
 *   2. provider PRESENT, no tenant -> soft: returns nulls, no throw. This is the
 *                                     normal state for a user who has not selected a
 *                                     tenant yet, and it must NOT throw.
 *
 * Only the leaf source is stubbed (`AccessContext`), so `TenantProvider` and
 * `useTenant` themselves are the real implementations under test.
 */
import { describe, it, expect, beforeEach, vi } from 'vitest';
import { render } from '@testing-library/react';
import React from 'react';

/** undefined = no AccessProvider mounted. */
let accessValue: any = undefined;

vi.mock('../contexts/AccessContext', () => ({
  useAccess: () => {
    if (accessValue === undefined) throw new Error('useAccess must be used within an AccessProvider');
    return accessValue;
  },
  // The real hook returns the context default (undefined) when no provider is mounted.
  useAccessOptional: () => accessValue,
}));

const { useTenant, TenantProvider } = await import('../contexts/TenantContext');

const NO_TENANT_SELECTED = {
  currentTenant: null,
  currentProduct: null,
  currentDatasource: null,
  setSelection: () => {},
  clearScope: () => {},
  isSelected: false,
};

/**
 * Minimal hook probe. `@testing-library/react` in this repo does not export
 * `renderHook` in its type surface, so the hook is captured by a one-line component
 * instead of depending on a version-specific helper.
 */
let captured: any;
const Probe: React.FC<{ wrapper?: (node: React.ReactElement) => React.ReactElement }> = ({ wrapper }) => {
  captured = useTenant();
  const node = <div />;
  return wrapper ? wrapper(node) : node;
};

const renderProbe = (wrapper?: (node: React.ReactElement) => React.ReactElement) => {
  captured = undefined;
  render(<Probe wrapper={wrapper} />);
  return captured;
};

describe('useTenant contract', () => {
  beforeEach(() => {
    accessValue = undefined;
    captured = undefined;
  });

  it('throws when no AccessProvider and no TenantProvider are mounted', () => {
    // React logs uncaught render errors to console.error even when the throw is
    // expected and asserted. Silence it for this test only, so the expected failure
    // does not look like a real one in CI output.
    const consoleError = vi.spyOn(console, 'error').mockImplementation(() => {});
    try {
      // Provider MISSING is a loud failure on purpose. If this ever stops throwing, the
      // loud-failure decision has been silently reverted somewhere.
      expect(() => renderProbe()).toThrow(
        /useTenant must be used within an AccessProvider or TenantProvider/,
      );
    } finally {
      consoleError.mockRestore();
    }
  });

  it('returns a soft null tenant when the provider is present but no tenant is selected', () => {
    accessValue = NO_TENANT_SELECTED;

    // Provider PRESENT with nothing selected must NOT throw. This is the realistic
    // "user has not picked a tenant yet" path, distinct from the case above.
    const value = renderProbe((node) => <TenantProvider>{node}</TenantProvider>);

    expect(value.tenant).toBeNull();
    expect(value.datasource).toBeNull();
    expect(value.isSelected).toBe(false);
    expect(typeof value.setSelection).toBe('function');
    expect(typeof value.clearSelection).toBe('function');
  });

  it('passes the selected tenant through when one is set', () => {
    const tenant = { id: 't-1', display_name: 'Acme' };
    accessValue = { ...NO_TENANT_SELECTED, currentTenant: tenant, isSelected: true };

    const value = renderProbe((node) => <TenantProvider>{node}</TenantProvider>);

    expect(value.tenant).toBe(tenant);
    expect(value.isSelected).toBe(true);
  });
});
