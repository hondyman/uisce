import { defineConfig, devices } from '@playwright/test';

/**
 * Config for the A11y Ratchet workflow (.github/workflows/a11y-ratchet.yml).
 *
 * The default playwright.config.ts only matches `phase3-*.spec.ts` and defines
 * seven browser projects, so `playwright test e2e/a11y/...` found no tests
 * there (and would have needed browsers the job does not install). The a11y
 * specs get their own config: just e2e/a11y, Chromium only.
 */
const BASE_URL = process.env.BASE_URL ?? 'http://localhost:5173';

export default defineConfig({
  testDir: './e2e/a11y',
  testMatch: '*.spec.ts',
  forbidOnly: !!process.env.CI,
  retries: process.env.CI ? 1 : 0,
  workers: process.env.CI ? 1 : undefined,
  reporter: [['line']],
  use: {
    baseURL: BASE_URL,
    trace: 'on-first-retry',
  },
  projects: [{ name: 'chromium', use: { ...devices['Desktop Chrome'] } }],
  webServer: {
    command: 'npm run dev',
    url: BASE_URL,
    // The workflow starts the dev server itself; only launch one when nothing
    // is listening (local runs).
    reuseExistingServer: true,
    timeout: 120 * 1000,
  },
});
