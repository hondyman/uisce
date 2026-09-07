import { defineConfig, devices } from '@playwright/test';

/**
 * Smoke-test configuration — E2E smoke tests for critical paths.
 *
 * Tests live in ./tests/playwright/ and cover:
 *   - Trigger lifecycle (create → toggle → dispatch → last_fired_at)
 *   - Workflow navigation smoke
 *
 * Auth: shared helpers in ./tests/playwright/auth-helper.ts
 *   E2E_JWT           — pre-minted JWT (CI)
 *   E2E_KC_USER+PASS  — Keycloak password grant (local dev)
 *   E2E_KC_ISSUER     — optional Keycloak issuer URL
 *   E2E_TENANT_ID      — optional tenant ID (default: northwind fixture)
 *
 * Backend: expects a running server on BASE_URL (default http://localhost:8080)
 *   Vite dev server proxies /api/* to BASE_URL via VITE_BACKEND_TARGET.
 */
export default defineConfig({
  testDir: './tests/playwright',

  /* Smoke tests are sequential — they share DB state */
  fullyParallel: false,
  forbidOnly: !!process.env.CI,
  retries: 0,
  workers: 1,

  reporter: [
    ['list'],
    ['html', { outputFolder: 'test-results/smoke' }],
    ['json', { outputFile: 'test-results/smoke/results.json' }],
  ],

  use: {
    /* baseURL is the Vite dev server; backend must be on :8080 */
    baseURL: 'http://localhost:5173',
    trace: 'on-first-retry',
    screenshot: 'only-on-failure',
    video: 'retain-on-failure',
  },

  projects: [
    {
      name: 'chromium',
      use: { ...devices['Desktop Chrome'] },
    },
  ],

  /* Start Vite dev server before tests (npm run dev → start-dev.sh) */
  webServer: {
    command: 'npm run dev',
    url: 'http://localhost:5173',
    reuseExistingServer: !process.env.CI,
    timeout: 120 * 1000,
    env: {
      /* Vite proxies /api/* to the backend */
      VITE_BACKEND_TARGET: 'http://localhost:8080',
    },
  },
});
