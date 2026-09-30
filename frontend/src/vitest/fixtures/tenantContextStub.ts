/**
 * Shared stub for `contexts/TenantContext`.
 *
 * `useTenant` is a loud, deliberate failure when no AccessProvider is mounted: it
 * throws 'useTenant must be used within an AccessProvider or TenantProvider'. Several
 * component tests render their subject bare, relying on an older `try { useTenant() }
 * catch { return { tenant: null, ... } }` wrapper that silently degraded the throw.
 * That wrapper is gone, so those tests must supply the seam themselves.
 *
 * IMPORTANT — what this stub does and does not prove:
 *   This is an EXPLICIT STUB. A test that passes through here is evidence about the
 *   component's rendering, NOT evidence that `useTenant` behaves correctly. Do not
 *   cite these tests as contract coverage for the tenant context.
 *
 * Note the distinction this encodes, which matters when reading a failure:
 *   - provider MISSING  -> `useTenant` throws (decision A, deliberately loud)
 *   - provider PRESENT but no tenant selected -> `tenant: null`, soft path
 * The shape below models the second case, which is the realistic runtime state for a
 * user who has not picked a tenant yet. See `useTenantProviderContract.test.tsx` for
 * the two behaviours asserted against each other.
 */

export interface StubTenantContext {
  tenant: unknown | null;
  product: unknown | null;
  datasource: { id?: string } | null;
  setSelection: (...args: unknown[]) => void;
  clearSelection: () => void;
  isSelected: boolean;
}

/** The default: provider present, no tenant selected yet. */
export const emptyTenantContext: StubTenantContext = {
  tenant: null,
  product: null,
  datasource: null,
  setSelection: () => {},
  clearSelection: () => {},
  isSelected: false,
};

/** Override individual fields without restating the whole shape. */
export const stubTenantContext = (overrides: Partial<StubTenantContext> = {}): StubTenantContext => ({
  ...emptyTenantContext,
  ...overrides,
});

/**
 * Factory for `vi.mock('../../contexts/TenantContext', tenantContextMockFactory)`.
 *
 * `vi.mock` replaces the whole module, so this returns every runtime export the
 * module has — `useTenant`, `TenantProvider`, and `TENANT_STORAGE_KEYS`. Anything
 * omitted would be `undefined` for any module in the graph that imports it.
 * `TenantProvider` is a pass-through because tests that already wrap their subject in
 * it should keep working without an edit.
 */
export const tenantContextMockFactory = (overrides: Partial<StubTenantContext> = {}) => ({
  useTenant: () => stubTenantContext(overrides),
  TenantProvider: ({ children }: { children?: unknown }) => children,
  TENANT_STORAGE_KEYS: {
    TENANT: 'selected_tenant',
    PRODUCT: 'selected_product',
    DATASOURCE: 'selected_datasource',
  },
});
