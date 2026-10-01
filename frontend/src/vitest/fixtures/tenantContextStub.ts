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
 *
 * The default scenario is "provider present, no tenant selected". Tests that assert a
 * tenant-scoped render path must override explicitly (pass an `overrides` arg) — this
 * mirrors the `useAccess` guard against the vacuous-green class.
 */

import type { Tenant, Product, DataSource } from '../../types';
import type { TenantContextType } from '../../contexts/TenantContext';

/**
 * The pass-through `TenantProvider` this module installs. It is NOT the real provider
 * (the real one calls `useAccess`, the seam under test), so its props type is declared
 * locally and structurally — it only has to accept and pass through `children`. The
 * compile-time shape guard below deliberately does NOT compare this against the real
 * module's props type, because the two are intentionally different implementations.
 */
type TenantProviderPropsShape = { children?: unknown };

/**
 * Mirror of `TenantContextType` from the real module. The field types use the real
 * `Tenant` / `Product` / `DataSource` shapes (rather than `unknown`) so the compile-time
 * shape guard at the bottom of this file is a real check. Tests can still override the
 * value fields with their own shapes via the `Partial<StubTenantContext>` argument; the
 * typecheck guards the SHAPE, not the value identity.
 */
export interface StubTenantContext {
  tenant: Tenant | null;
  product: Product | null;
  datasource: DataSource | null;
  setSelection: (tenant: Tenant, product: Product, datasource: DataSource) => void;
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
 * Throws the exact string the real `useTenant` throws when no provider is mounted.
 * Lives at module scope so the throw message is constant — tests can match it
 * literally without reconstructing the string.
 */
export const NO_PROVIDER_ERROR =
  'useTenant must be used within an AccessProvider or TenantProvider';

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
 * real source and cannot drift from it. An identity test in
 * `tenantContextStub.identity.test.tsx` enforces that drift.
 *
 * `TenantProvider` is deliberately NOT the real one: the real provider calls
 * `useAccess`, which is exactly the seam these tests are stubbing around. It stays a
 * pass-through, so a test that already wraps its subject in it needs no edit.
 *
 * Two scenarios, controlled by `__noProvider`:
 *   - `__noProvider: false` (default): `useTenant` returns the stub context (provider
 *     present, no tenant selected). This is the realistic runtime state for a user
 *     who has not picked a tenant yet.
 *   - `__noProvider: true`: `useTenant` throws `NO_PROVIDER_ERROR`, matching the real
 *     `useTenant`'s loud-failure contract. Tests that exercise the no-provider branch
 *     use this mode and assert the throw.
 *
 * The `__noProvider` key is stripped from the overrides object before applying it to
 * the stub context, so it never leaks into a tenant-shaped object.
 */
export interface ApplyTenantStubOverrides extends Partial<StubTenantContext> {
  __noProvider?: boolean;
}

export const applyTenantContextStub = async (
  importOriginal: () => Promise<Record<string, unknown>>,
  overrides: ApplyTenantStubOverrides = {},
): Promise<Record<string, unknown>> => {
  const actual = await importOriginal();
  const { __noProvider, ...tenantOverrides } = overrides;
  const hasOverrides = Object.keys(tenantOverrides).length > 0;
  return {
    ...actual,
    useTenant: __noProvider
      ? () => {
          throw new Error(NO_PROVIDER_ERROR);
        }
      : hasOverrides
        ? () => stubTenantContext(tenantOverrides)
        : () => emptyTenantContext,
    TenantProvider: ({ children }: TenantProviderPropsShape) => children,
  };
};

/**
 * ============================================================================
 * COMPILE-TIME SHAPE GUARD — exact, bidirectional, no escape hatch
 * ============================================================================
 *
 * This guard lives HERE, in the fixture module, rather than in a test file. That
 * placement is the point: TypeScript checks this module as soon as anything imports
 * it, so every consumer of the stub is covered, not just whichever test happens to
 * assert on the shape. A guard in one test file would be a property of that file.
 *
 * WHY NOT THE OBVIOUS THREE (all three were tried and all three are decorative):
 *
 *   1. `const c: TenantContextType = emptyTenantContext as any`
 *      `as any` is assignable to every type. This line can never fail. It is a
 *      comment with type syntax, and it is exactly what this guard replaces.
 *
 *   2. `const c: TenantContextType = emptyTenantContext`
 *      Directional — catches fields MISSING from the stub, but not fields EXTRA
 *      on it. A stub carrying a field the real module does not have still passes,
 *      which is the drift that actually bites: a test sets `canAccess` on the stub,
 *      the component reads it, the component ships, the real context has no such
 *      field, and production silently reads `undefined`.
 *
 *   3. `const c = {...emptyTenantContext} satisfies TenantContextType`
 *      The excess-property check only fires on a FRESH object literal. The spread
 *      produces a fresh literal, so this one is closer — but `satisfies` is still
 *      one-directional, and it would not catch the stub TYPE drifting away from the
 *      real type, only the default VALUE drifting away from the real type.
 *
 * `Exact<A, B>` below compares in BOTH directions, so it fails on a field added,
 * a field removed, and a field whose type changed. It is a pure type-level
 * computation: `true` is assignable to it only when the two types are mutually
 * assignable, so any drift makes the `const` below a TS2322.
 *
 * Discrimination — verified by deliberately perturbing each side and confirming
 * `npx tsc --noEmit` fails, then reverting (see PR #241 for the transcripts):
 *
 *   - add `__drift: string` to `StubTenantContext`   -> TS2322 here (extra field)
 *   - remove `isSelected` from `TenantContextType`   -> TS2322 here (missing field)
 *   - widen `isSelected` to `boolean | undefined`    -> TS2322 here (type change)
 *   - revert any of the three                        -> clean
 *
 * The two types are compared structurally. `Exact` is written with the `[T]`
 * tuple wrapper so the conditional distributes nothing and `never`/`any` edge
 * cases cannot silently produce `true`.
 */
type Exact<A, B> = [A] extends [B] ? ([B] extends [A] ? true : false) : false;

/**
 * `true` only when `StubTenantContext` and the real `TenantContextType` are mutually
 * assignable — i.e. identical in every field name and field type, in both directions.
 */
export type TenantContextShapeIsExact = Exact<StubTenantContext, TenantContextType>;

/**
 * The assertion. Deliberately NOT `as any`, NOT `as unknown as`, NOT a `void` of a
 * cast value: the cast forms are what made the previous guard decorative. This is a
 * plain annotated assignment, so a `false` from `Exact` is a hard compile error
 * pointing at this exact line.
 */
// eslint-disable-next-line @typescript-eslint/no-unused-vars
const _shapeIsExact: TenantContextShapeIsExact = true;

/**
 * The default value is checked against the STUB type (not the real type) so that a
 * missing key in `emptyTenantContext` is also a compile error. A stub type that
 * declares `isSelected` but a default object that forgets it is still drift — the
 * test would render with `undefined` and report green.
 */
type DefaultIsComplete = Exact<StubTenantContext, typeof emptyTenantContext>;
export type DefaultIsCompleteIsExact = DefaultIsComplete;
// eslint-disable-next-line @typescript-eslint/no-unused-vars
const _defaultIsComplete: DefaultIsComplete = true;
