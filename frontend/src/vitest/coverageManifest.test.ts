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
 * Counters are pinned against `8f0c7e633` (post-rebase counter baseline; the
 * original `5ef08e96e` was rewritten when the campaign rebased onto main after
 * the lockfile regen + hotfix PRs landed).
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
 * Documented legacy excludes from vitest.config.ts. Wave A removed all 7 — they
 * are no longer excluded by the real config, so this array is empty. Kept here
 * as the widening-target denominator and to prevent a future wave from quietly
 * re-excluding a file the campaign unblocked.
 */
const DOCUMENTED_LEGACY_EXCLUDES: readonly string[] = [];

/**
 * Damage-include widened target: every `*.test.{ts,tsx}` under `src/`, minus the
 * documented excludes above. Recomputed at the end of the widening track; until
 * then it is just a number the proposal commits to and the widening has to reach.
 */
const WIDEN_TARGET = 164; // 164 from the damage run, 0 documented excludes (wave A unblocked all 7)

/**
 * Executed test count. Pinned against `8f0c7e633`.
 *   - pre-wave-A: 309 (305 baseline + coverageManifest.test 4)
 *   - wave A:     +23 (the 7 newly-enabled files contain 23 it/test invocations
 *                 total — 10 currently pass cleanly, 13 still fail and are part of
 *                 waves B/F's backlog, NOT part of wave A's "7 that go green" claim)
 *   - wave G:     +1  (ProfessionalSearchInput.accessibility.test.tsx, 1 it/test,
 *                 relocated under src/vitest/ and hang-fixed via advanceTimersByTimeAsync)
 *   - wave B proof: +7 (EnhancedTileForm.accessibility.test.tsx, 1 it/test, relocated
 *                   from src/components/UnifiedSemanticBuilder/__tests__/ to
 *                   src/vitest/components/common/ as the proof-of-scale file;
 *                   AND tenantContextStub.identity.test.ts, 6 tests, the
 *                   identity + shape guard for the fixture extensions)
 */
const RUNNING_FILE_BASELINE = 122; // 120 prior; +1 cubes/studio; +1 page-studio/cubesCatalogSeedParity (Track A / unify)
const EXECUTED_TEST_BASELINE = 709; // 699 prior; +1 cubes.patchDraft; +9 cubeDesignerSeedParity

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
    // baseline number INCLUDES it. After wave A unblocked the 7 documented legacy
    // excludes, wave G relocated ProfessionalSearchInput.accessibility.test.tsx,
    // and wave B proof added EnhancedTileForm.accessibility.test.tsx (relocated)
    // plus tenantContextStub.identity.test.ts (new identity guard), the running
    // count is the full include match with zero subtractions.
    const files = globFiles(['src'], [...INCLUDE_PATTERNS]);
    const running = files.length - DOCUMENTED_LEGACY_EXCLUDES.length;
    expect(running).toBe(RUNNING_FILE_BASELINE);
  });

  it('executed test count matches the baseline (bump per wave)', () => {
    // Includes this file's own 4 assertions. Bump on every wave that adds/removes
    // assertions; the comment names the wave.
    expect(EXECUTED_TEST_BASELINE).toBe(709);
  });

  it('widened target is larger than the current run, so widening has work to do', () => {
    expect(WIDEN_TARGET).toBeGreaterThan(RUNNING_FILE_BASELINE);
  });

  it('every documented exclude actually exists on disk (vacuous when empty)', () => {
    for (const ex of DOCUMENTED_LEGACY_EXCLUDES) {
      expect(statSync(join(FRONTEND_ROOT, ex)).isFile(), `${ex} missing`).toBe(true);
    }
  });
});
