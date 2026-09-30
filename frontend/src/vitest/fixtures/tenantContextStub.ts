/**
 * Shared stub for `contexts/TenantContext`.
 *
 * `useTenant` is a loud, deliberate failure when no AccessProvider is mounted: it
 * throws 'useTenant must be used within an AccessProvider or TenantProvider'. Several
 * component tests render their subject bare, relying on an older `try { useTenant() }
 * catch { return { tenant: null, ... } }` wrapper that silently degraded the throw.
 * That wrapper is gone, so those tests must supply the seam themselves.
 *
 * ---------------------------------------------------------------------------
 * FIXTURE LAW — two rules, learned the hard way. Copy these; don't rediscover them.
 * ---------------------------------------------------------------------------
 *
 * 1. Values a mock exposes as OBJECT are built ONCE at module level, never per call.
 *    Anything shaped like `scope`, `accessibleTenants`, or `tenants` is a frozen
 *    module-level constant. A per-call literal is a fresh reference on every render,
 *    so any effect depending on it re-runs forever. A test written that way reports
 *    green while measuring a render loop rather than the behaviour it names.
 *
 * 2. Every test that uses this stub reports a discrimination note: what was changed or
 *    removed to confirm the assertion can actually fail. "The suite is green" is
 *    inventory, not evidence. An assertion that has never been seen failing is
 *    indistinguishable from one that cannot fail.
 *
 * ---------------------------------------------------------------------------
 * What this stub does and does not prove
 * ---------------------------------------------------------------------------
 * This is an EXPLICIT STUB. A test passing through here is evidence about the
 * component's rendering, NOT evidence that `useTenant` behaves correctly. Do not cite
 * these tests as contract coverage for the tenant context. For that, see
 * `useTenantContract.test.tsx`.
 *
 * Note the distinction this encodes, which matters when reading a failure:
 *   - provider MISSING  -> `useTenant` throws (decision A, deliberately loud)
 *   - provider PRESENT but no tenant selected -> `tenant: null`, soft path
 * The shape below models the second case: the realistic runtime state for a user who
 * has not picked a tenant yet.
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
export const emptyTenantContext: StubTenantContext = Object.freeze({
  tenant: null,
  product: null,
  datasource: null,
  setSelection: () => {},
  clearSelection: () => {},
  isSelected: false,
});

/** Override individual fields without restating the whole shape. */
export const stubTenantContext = (overrides: Partial<StubTenantContext> = {}): StubTenantContext =>
  Object.freeze({ ...emptyTenantContext, ...overrides });

/**
 * Applies the stub on top of the real module.
 *
 * Usage — note the factory is written inline because `vi.mock` is hoisted above
 * imports, so the fixture must be pulled in dynamically rather than referenced as a
 * binding:
 *
 *   vi.mock('../../../contexts/TenantContext', async (importOriginal) => {
 *     const { applyTenantContextStub } = await import('.../fixtures/tenantContextStub');
 *     return applyTenantContextStub(importOriginal);
 *   });
 *
 * The real module is spread in so constants such as TENANT_STORAGE_KEYS come from the
 * real source and cannot drift from it.
 *
 * `TenantProvider` is deliberately NOT the real one: the real provider calls
 * `useAccess`, which is exactly the seam these tests are stubbing around. It stays a
 * pass-through, so a test that already wraps its subject in it needs no edit.
 */
export const applyTenantContextStub = async (
  importOriginal: () => Promise<Record<string, unknown>>,
  overrides: Partial<StubTenantContext> = {},
): Promise<Record<string, unknown>> => {
  const actual = await importOriginal();
  return {
    ...actual,
    useTenant: () => stubTenantContext(overrides),
    TenantProvider: ({ children }: { children?: unknown }) => children,
  };
};
