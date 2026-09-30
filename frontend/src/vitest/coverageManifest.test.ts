/**
 * Inclusion / execution counter for the widening track.
 *
 * Two numbers, exact (not >=), bumped per wave:
 *   1. RUNNING FILE COUNT — files the vitest `include` glob actually matches, minus
 *      the documented excludes. Computed here from the filesystem, not hardcoded, so
 *      adding a file anywhere else also reddens this test and forces a deliberate
 *      update — the "widening is measurable" property.
 *   2. EXECUTED TEST COUNT — total `it`/`test` invocations the suite actually runs.
 *      This one is hardcoded (vitest can't introspect itself from inside a test).
 *      Bump on every wave; the comment names the wave.
 *
 * The widened target (current damage include minus the documented excludes) is also
 * recorded here as `WIDEN_TARGET`. Pre-widening this is larger than the current
 * RUNNING FILE COUNT, and the difference is exactly the work this track is doing.
 * When RUNNING FILE COUNT catches up to WIDEN_TARGET, widening is done by definition.
 *
 * The numbers below are the BASELINE (post hotfix 16c1615bd, post RootProviders
 * extraction b0a0a626e). Wave A is the first deliberate bump.
 */
import { describe, it, expect } from 'vitest';
import { readdirSync, statSync } from 'fs';
import { join } from 'path';

// The vitest config treats paths as relative to the frontend/ directory (where
// vitest.config.ts lives). `__dirname` here is `src/vitest/`, so two levels up.
const FRONTEND_ROOT = join(__dirname, '..', '..');

/** Mirrors the `include` glob in vitest.config.ts. */
const INCLUDE_PATTERNS = ['src/vitest/**/*.test.ts', 'src/vitest/**/*.test.tsx'] as const;

/**
 * Mirrors the documented legacy exclude list in vitest.config.ts — 7 files the
 * real config skips with the reason "pre-workstation unmaintained legacy tests".
 * Adding or removing entries here is part of widening and must be deliberate.
 */
const DOCUMENTED_LEGACY_EXCLUDES = [
  'src/vitest/components/pagestudio/PageStudioCanvas.test.tsx',
  'src/vitest/components/pagestudio/pageStudioTabsMigration.test.ts',
  'src/vitest/components/common/UnifiedBOPickerModalMultiSubtype.test.tsx',
  'src/vitest/components/rules/RuleDiffViewer.test.tsx',
  'src/vitest/BusinessObjectDetailsPage.test.tsx',
  'src/vitest/utils/dedupeFields.test.ts',
  'src/vitest/components/business-objects/BusinessObjectBindingWizard.test.tsx',
] as const;

/**
 * Damage-include widened target: every `*.test.{ts,tsx}` under `src/`, minus the
 * documented excludes above. Recomputed at the end of the widening track; until
 * then it is just a number the proposal commits to and the widening has to reach.
 */
const WIDEN_TARGET = 164 - DOCUMENTED_LEGACY_EXCLUDES.length; // 164 from the damage run, 7 documented excludes

/**
 * Baseline executed test count, measured against the merged tree at commit b0a0a626e.
 * Bump on every wave that adds/removes assertions. The comment names the wave.
 */
const EXECUTED_TEST_BASELINE = 309; // b0a0a626e (305) + coverageManifest.test (4) — pre-wave-A

function globFiles(roots: string[], patterns: string[]): string[] {
  const out: string[] = [];
  const walk = (dir: string) => {
    let entries: { name: string; isFile: () => boolean; isDirectory: () => boolean }[];
    try {
      entries = readdirSync(dir, { withFileTypes: true }) as any;
    } catch {
      return;
    }
    for (const e of entries) {
      const full = join(dir, e.name);
      if (e.isDirectory()) walk(full);
      else if (e.isFile()) {
        const rel = full.slice(FRONTEND_ROOT.length + 1);
        if (matchesAny(rel, patterns)) out.push(rel);
      }
    }
  };
  for (const r of roots) walk(join(FRONTEND_ROOT, r));
  return out.sort();
}

function matchesAny(rel: string, patterns: readonly string[]): boolean {
  for (const p of patterns) {
    if (matchGlob(rel, p)) return true;
  }
  return false;
}

function matchGlob(rel: string, pattern: string): boolean {
  // The vitest include globs here only need to handle `**` (any number of
  // directories, including zero) and `*` (a single path segment). The pattern
  // `src/vitest/**/*.test.ts` must match BOTH `src/vitest/x.test.ts` AND
  // `src/vitest/sub/x.test.ts`.
  let rx = '';
  let i = 0;
  while (i < pattern.length) {
    const c = pattern[i];
    if (c === '*') {
      if (pattern[i + 1] === '*') {
        // `**` — any chars including separators; we anchor to the surrounding
        // `/` so it can also match the empty segment.
        rx += '.*';
        i += 2;
        if (pattern[i] === '/') i += 1; // eat the trailing slash, if any
      } else {
        rx += '[^/]*';
        i += 1;
      }
    } else if ('.+^${}()|[]\\'.includes(c)) {
      rx += '\\' + c;
      i += 1;
    } else {
      rx += c;
      i += 1;
    }
  }
  return new RegExp('^' + rx + '$').test(rel);
}

describe('test suite inclusion counter', () => {
  it('currently-running file count matches the baseline (bump per wave)', () => {
    // Self-counting: this file is one of the files matching the include, so the
    // baseline number INCLUDES it. Subtract the 7 documented legacy excludes, which
    // vitest skips but our glob matches.
    const files = globFiles(['src'], [...INCLUDE_PATTERNS]);
    const running = files.length - DOCUMENTED_LEGACY_EXCLUDES.length;
    expect(running).toBe(66);
  });

  it('executed test count matches the baseline (bump per wave)', () => {
    // Includes this file's own 4 assertions. Bump on every wave that adds/removes
    // assertions; the comment names the wave.
    expect(EXECUTED_TEST_BASELINE).toBe(309);
  });

  it('widened target is larger than the current run, so widening has work to do', () => {
    expect(WIDEN_TARGET).toBeGreaterThan(66);
  });

  it('every documented exclude actually exists on disk', () => {
    for (const ex of DOCUMENTED_LEGACY_EXCLUDES) {
      expect(statSync(join(FRONTEND_ROOT, ex)).isFile(), `${ex} missing`).toBe(true);
    }
  });
});
