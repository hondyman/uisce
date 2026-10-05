/**
 * PR7 / CUBE-3.3 — one-shot legacy `dataBindings.primary.cube` name → QuerySubject.
 * Fail closed when the name cannot be resolved against cube_definition.
 * No runtime name-only dual-write: callers persist the returned subject pin.
 */

import type { ContractVersionPin, CubeSubject, QuerySubject } from './types';
import { cubeSubject, isCubeSubject } from './types';

/** Minimal cube row needed to resolve a legacy name. */
export type LegacyCubeLookup = {
  id: string;
  name: string;
  contractVersion: number;
};

export type LegacyCubeMigrateOk = {
  ok: true;
  subject: CubeSubject;
  /** Matched cube row when resolved from a name (absent when subject already pinned). */
  cube?: LegacyCubeLookup;
  /** True when input was a legacy name string that resolved. */
  migrated: boolean;
};

export type LegacyCubeMigrateErr = {
  ok: false;
  reason: string;
};

export type LegacyCubeMigrateResult = LegacyCubeMigrateOk | LegacyCubeMigrateErr;

/** Loose binding shape from report_definitions / layout_config / metadata. */
export type LegacyCubeBinding = {
  cube?: string | null;
  subject?: QuerySubject | null;
  measures?: unknown;
  dimensions?: unknown;
  filters?: unknown;
  [key: string]: unknown;
};

function normalizeName(name: string): string {
  return name.trim();
}

/**
 * Resolve a legacy cube **name** (e.g. `oms.account`) to a pinned QuerySubject.
 * Exact name match against the provided cube list (case-sensitive, trimmed).
 */
export function migrateLegacyCubeName(
  legacyName: string | null | undefined,
  cubes: LegacyCubeLookup[],
): LegacyCubeMigrateResult {
  const name = typeof legacyName === 'string' ? normalizeName(legacyName) : '';
  if (!name) {
    return { ok: false, reason: 'legacy cube name is blank' };
  }
  const matches = cubes.filter((c) => normalizeName(c.name) === name);
  if (matches.length === 0) {
    return {
      ok: false,
      reason: `unresolved legacy cube name "${name}" — no cube_definition row with that name`,
    };
  }
  if (matches.length > 1) {
    return {
      ok: false,
      reason: `ambiguous legacy cube name "${name}" — ${matches.length} cube_definition rows`,
    };
  }
  const cube = matches[0];
  if (!cube.id?.trim()) {
    return { ok: false, reason: `cube "${name}" is missing id` };
  }
  if (typeof cube.contractVersion !== 'number' || cube.contractVersion <= 0) {
    return {
      ok: false,
      reason: `cube "${name}" has invalid contractVersion ${String(cube.contractVersion)}`,
    };
  }
  return {
    ok: true,
    subject: cubeSubject(cube.id, cube.contractVersion),
    cube,
    migrated: true,
  };
}

/**
 * Accept an already-pinned cube subject, or migrate a legacy name once.
 * Non-cube subjects pass through only when explicitly provided as `subject`
 * with kind business_object (reports that never used cube names).
 */
export function resolveReportCubeBinding(
  binding: LegacyCubeBinding | null | undefined,
  cubes: LegacyCubeLookup[],
): LegacyCubeMigrateResult | { ok: true; subject: QuerySubject; migrated: false } {
  if (!binding) {
    return { ok: false, reason: 'binding is missing' };
  }

  if (isCubeSubject(binding.subject)) {
    if (!binding.subject.cubeId?.trim()) {
      return { ok: false, reason: 'subject.cubeId is blank' };
    }
    const pin = binding.subject.contractVersion;
    if (pin !== 'latest' && (typeof pin !== 'number' || pin <= 0)) {
      return {
        ok: false,
        reason: `subject.contractVersion is invalid: ${String(pin)}`,
      };
    }
    return { ok: true, subject: binding.subject, migrated: false };
  }

  if (binding.subject?.kind === 'business_object') {
    return { ok: true, subject: binding.subject, migrated: false };
  }

  const legacy = typeof binding.cube === 'string' ? binding.cube : '';
  if (legacy.trim()) {
    return migrateLegacyCubeName(legacy, cubes);
  }

  return {
    ok: false,
    reason: 'binding has neither subject pin nor legacy cube name',
  };
}

/**
 * Rewrite `dataBindings.primary` in place: drop legacy `cube` name, set `subject`.
 * Returns a new object; does not mutate `bindings`.
 */
export function rewriteDataBindingsWithSubject(
  dataBindings: Record<string, LegacyCubeBinding> | null | undefined,
  subject: CubeSubject,
  primaryKey = 'primary',
): Record<string, LegacyCubeBinding> {
  const next: Record<string, LegacyCubeBinding> = { ...(dataBindings || {}) };
  const primary = { ...(next[primaryKey] || {}) };
  delete primary.cube;
  primary.subject = subject;
  next[primaryKey] = primary;
  return next;
}

/** Extract legacy name from a binding when present (for scans / diagnostics). */
export function extractLegacyCubeName(
  binding: LegacyCubeBinding | null | undefined,
): string | null {
  if (!binding || typeof binding.cube !== 'string') return null;
  const name = normalizeName(binding.cube);
  return name || null;
}

/** True when contractVersion is a publishable numeric pin. */
export function isPinnedContractVersion(
  v: ContractVersionPin | null | undefined,
): v is number {
  return typeof v === 'number' && v > 0;
}
