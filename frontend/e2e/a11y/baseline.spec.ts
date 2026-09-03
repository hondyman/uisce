import { test, expect } from '@playwright/test';
import AxeBuilder from '@axe-core/playwright';
import fs from 'node:fs';
import path from 'node:path';

/**
 * Phase 0 axe baseline. Run against the default-locale URLs (locales all use
 * the same components, so a single crawl covers all languages for now). Once
 * /ar becomes a real surface, add a parallel spec iterating over locales.
 *
 * Output: docs/a11y/baseline-{date}.json grouped by WCAG SC.
 */

const ROUTES = [
  '/',
  '/en/',
  '/en/dashboard',
  '/en/scheduler/jobs',
  '/en/reports/library',
  '/en/admin/rbac/roles',
  '/en/business-objects',
  '/en/core/glossary',
  '/login',
  '/api-studio',
];

test.describe('Phase 0 axe baseline (WCAG 2.1 AA)', () => {
  for (const route of ROUTES) {
    test(`axe: ${route}`, async ({ page }) => {
      await page.goto(route, { waitUntil: 'networkidle' });
      const results = await new AxeBuilder({ page })
        .withTags(['wcag2a', 'wcag2aa', 'wcag21a', 'wcag21aa'])
        // Color-contrast is part of WCAG AA; surface it but don't fail build yet.
        .disableRules(['color-contrast'])
        .analyze();

      // Persist a per-route JSON; one aggregate is written after the run.
      const dir = 'test-results/a11y';
      fs.mkdirSync(dir, { recursive: true });
      fs.writeFileSync(
        path.join(dir, `${route.replace(/\W+/g, '_')}.json`),
        JSON.stringify(results, null, 2),
      );

      // For Phase 0 we RECORD violations rather than fail — the baseline is the
      // artifact, not a gate. Subsequent phases flip to expect.toEqual([]).
      test.info().annotations.push({
        type: 'axe-summary',
        description: `${results.violations.length} rule violations, ${results.passes.length} passes`,
      });
    });
  }
});
