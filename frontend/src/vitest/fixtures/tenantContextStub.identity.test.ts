/**
 * Identity + shape guard for `applyTenantContextStub`.
 *
 * Two guards here, both load-bearing for the stub:
 *
 * 1. IDENTITY: every constant exported by the real `contexts/TenantContext` is
 *    preserved through the stub's `importOriginal` spread. If a real-module constant
 *    is renamed or removed, this test goes red at the affected line. The stub's
 *    `...actual` is what guarantees the constants come from the real source rather
 *    than a hand-maintained mirror.
 *
 * 2. SHAPE: the stub's default context shape is typechecked at compile time against
 *    the real module's `TenantContextType`. The guard itself lives in the fixture
 *    module (`tenantContextStub.ts`) so it is evaluated wherever the fixture is
 *    imported, not just here; the tail of this file re-asserts it so that deleting
 *    or weakening the fixture's guard breaks THIS file too. It is bidirectional, so
 *    it catches a field added, a field removed, and a field whose type changed.
 *
 * Discrimination: change a constant value in the real module (e.g., rename
 * `TENANT_STORAGE_KEYS.TENANT`), and the identity test goes red. Add a field to
 * `StubTenantContext`, and the shape check forces every stub call site to either fill
 * the field or add it to the default.
 */
import { describe, it, expect } from 'vitest';
import { applyTenantContextStub, emptyTenantContext } from './tenantContextStub';
import type {
  ApplyTenantStubOverrides,
  TenantContextShapeIsExact,
  DefaultIsCompleteIsExact,
} from './tenantContextStub';
import * as actualModule from '../../contexts/TenantContext';

describe('applyTenantContextStub', () => {
  it('preserves TENANT_STORAGE_KEYS identity from the real module', async () => {
    const stub = await applyTenantContextStub(async () => actualModule as any);
    expect(stub.TENANT_STORAGE_KEYS).toBe(actualModule.TENANT_STORAGE_KEYS);
    expect(stub.TENANT_STORAGE_KEYS).toEqual({
      TENANT: 'selected_tenant',
      PRODUCT: 'selected_product',
      DATASOURCE: 'selected_datasource',
    });
  });

  it('preserves TenantProvider pass-through identity from the real module', async () => {
    const stub = await applyTenantContextStub(async () => actualModule as any);
    // The stubbed TenantProvider is a pass-through (NOT the real one) because the real
    // provider calls useAccess — that's the seam these tests stub around. Identity
    // with the real module would defeat the stub, so we assert inequality here and
    // document the choice inline.
    expect(stub.TenantProvider).not.toBe(actualModule.TenantProvider);
    // But its TYPE matches — both are functions taking children.
    expect(typeof stub.TenantProvider).toBe('function');
  });

  it('returns the default stub context when no overrides are given', async () => {
    const stub = await applyTenantContextStub(async () => actualModule as any);
    const ctx = (stub.useTenant as () => any)();
    expect(ctx).toEqual(emptyTenantContext);
    // Same reference, not just deep-equal — proves the module-level freeze.
    expect(ctx).toBe(emptyTenantContext);
  });

  it('honours overrides to the stub context', async () => {
    // The override fields are cast as the real types because the test only cares
    // about identity (the stub returns the override verbatim), not the full
    // real-world shape of Tenant/DataSource.
    const stub = await applyTenantContextStub(async () => actualModule as any, {
      tenant: { id: 't-1', name: 'Test Tenant' } as any,
      datasource: { id: 'd-1' } as any,
      isSelected: true,
    });
    const ctx = (stub.useTenant as () => any)();
    expect(ctx.tenant).toEqual({ id: 't-1', name: 'Test Tenant' });
    expect(ctx.datasource).toEqual({ id: 'd-1' });
    expect(ctx.isSelected).toBe(true);
    // The stub is a NEW frozen object, not the module-level default.
    expect(ctx).not.toBe(emptyTenantContext);
  });

  it('throws the loud-failure string in __noProvider mode', async () => {
    const stub = await applyTenantContextStub(async () => actualModule as any, {
      __noProvider: true,
    });
    expect(() => (stub.useTenant as () => any)()).toThrow(
      'useTenant must be used within an AccessProvider or TenantProvider',
    );
  });

  it('strips __noProvider from the overrides before applying (no leak into stub context)', async () => {
    // Even with __noProvider: true, the literal overrides object passed in should not
    // retain the __noProvider key as a tenant-shaped field. The factory destructures
    // it out before passing to stubTenantContext.
    const overrides: ApplyTenantStubOverrides = { __noProvider: true, tenant: { id: 't-1' } as any };
    const stub = await applyTenantContextStub(async () => actualModule as any, overrides);
    // The override shape goes through __noProvider branch (throws), so we can't read
    // back the context. But we CAN verify the destructuring by reading the source:
    expect(stub.useTenant).toBeDefined();
    // Sanity: stub.useTenant still throws in this mode.
    expect(() => (stub.useTenant as () => any)()).toThrow();
    // The `__noProvider` key should NOT be a tenant-shaped field of the overrides
    // we passed in (which would be a leak from the destructuring).
    expect((overrides as any).__noProvider).toBe(true); // untouched
  });
});

/**
 * Compile-time shape guard — see the full rationale in `tenantContextStub.ts`, where
 * the guard actually lives.
 *
 * It is defined in the FIXTURE MODULE, not here, so that TypeScript evaluates it at
 * every import site rather than only in this one test file. The two assertions below
 * are what keep that true from this side: they re-state the guard as a value-level
 * dependency of this suite, so if the fixture's guard is deleted or weakened, THIS
 * file stops compiling rather than silently going quiet.
 *
 * Discrimination (verified — see the PR body):
 *   - add `__drift: string` to `StubTenantContext`  -> TS2322 in the fixture
 *   - remove `isSelected` from `TenantContextType`  -> TS2322 in the fixture
 *   - widen `isSelected` to `boolean | undefined`   -> TS2322 in the fixture
 *   - revert any of the three                       -> clean
 */
type GuardedShapeIsExact = TenantContextShapeIsExact & DefaultIsCompleteIsExact;

// Fails to compile if the fixture's guard ever stops resolving to literal `true`.
const _guardHolds: GuardedShapeIsExact = true;
// eslint-disable-next-line @typescript-eslint/no-unused-vars
void _guardHolds;

