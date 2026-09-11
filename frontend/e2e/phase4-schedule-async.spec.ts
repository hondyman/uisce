/**
 * Phase 4 E2E: Report Schedule Async Contract
 *
 * Tests the frontend polling state machine end-to-end using Playwright's
 * page.route() to mock the backend contract (no live backend needed).
 *
 * Backend contract (Phase 3, commit 81ff74506f):
 *   POST /api/v1/reports/{id}/schedules/{sid}/run
 *     → 202 Accepted  { status: "pending", execution_id, workflow_id }
 *     → 503 Service Unavailable { execution_id, error }
 *   GET  /api/v1/reports/executions/{id}  (polled repeatedly)
 *     → 200 { status: "pending" | "running" | "completed" | "failed", ... }
 *
 * Attribution:
 *   Live-backend coverage of the 202/503/polling contract is held by
 *   Go integration tests in backend/internal/api/report_handlers_test.go
 *   (81ff74506f). This spec tests the UI transition rendering only,
 *   using network-layer mocks via page.route(). Not claimed as live-backend
 *   security or correctness coverage.
 */

import { test, expect, Page } from '@playwright/test';

const BASE_URL = process.env.PLAYWRIGHT_BASE_URL ?? 'http://localhost:3000';

// ---------------------------------------------------------------------------
// Shared helpers
// ---------------------------------------------------------------------------

/** Sequence of execution records returned by GET /api/v1/reports/executions/{id} */
const EXECUTION_SEQUENCE = [
  { id: 'exec-ph4', tenant_id: 'tenant-001', template_id: 'tmpl-1', report_key: 'valuation', status: 'pending', created_at: '2026-09-11T10:00:00Z', is_personal: false },
  { id: 'exec-ph4', tenant_id: 'tenant-001', template_id: 'tmpl-1', report_key: 'valuation', status: 'running', created_at: '2026-09-11T10:00:00Z', is_personal: false },
  { id: 'exec-ph4', tenant_id: 'tenant-001', template_id: 'tmpl-1', report_key: 'valuation', status: 'completed', output_url: 'https://storage.example.com/out.pdf', rows_processed: 1420, execution_time_ms: 3200, created_at: '2026-09-11T10:00:00Z', completed_at: '2026-09-11T10:00:03Z', is_personal: false },
];

const EXECUTION_FAILED_SEQUENCE = [
  { id: 'exec-ph4-err', tenant_id: 'tenant-001', template_id: 'tmpl-1', report_key: 'valuation', status: 'pending', created_at: '2026-09-11T10:00:00Z', is_personal: false },
  { id: 'exec-ph4-err', tenant_id: 'tenant-001', template_id: 'tmpl-1', report_key: 'valuation', status: 'failed', error_message: 'Template rendering error: missing field portfolio_id', created_at: '2026-09-11T10:00:00Z', is_personal: false },
];

let pollCallCount = 0;

function mockTriggerOk(page: Page) {
  return page.route('/api/v1/reports/report-001/schedules/sched-001/run', (route) => {
    route.fulfill({
      status: 202,
      contentType: 'application/json',
      body: JSON.stringify({ status: 'pending', execution_id: 'exec-ph4', workflow_id: 'wf-ph4' }),
    });
  });
}

function mockTrigger503(page: Page) {
  return page.route('/api/v1/reports/report-001/schedules/sched-001/run', (route) => {
    route.fulfill({
      status: 503,
      contentType: 'application/json',
      body: JSON.stringify({ execution_id: 'exec-dispatch-err', error: 'Temporal cluster unavailable' }),
    });
  });
}

function mockPollingSequence(page: Page, sequence: Record<string, unknown>[]) {
  pollCallCount = 0;
  return page.route('/api/v1/reports/executions/exec-ph4', (route) => {
    const idx = Math.min(pollCallCount, sequence.length - 1);
    pollCallCount++;
    route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify(sequence[idx]),
    });
  });
}

function mockPollingFailedSequence(page: Page, sequence: Record<string, unknown>[]) {
  pollCallCount = 0;
  return page.route('/api/v1/reports/executions/exec-ph4-err', (route) => {
    const idx = Math.min(pollCallCount, sequence.length - 1);
    pollCallCount++;
    route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify(sequence[idx]),
    });
  });
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

test.describe('Phase 4: Report Schedule — Async 202 Contract', () => {
  test.beforeEach(async ({ page }) => {
    // Intercept schedule list — needed for the dialog to show the Run Now button
    await page.route('/api/v1/reports/report-001/schedules', (route) => {
      route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify([{ id: 'sched-001', schedule_name: 'Daily Valuation' }]),
      });
    });
  });

  test('202 → chip transitions pending → running → completed; output shown', async ({ page }) => {
    await mockTriggerOk(page);
    await mockPollingSequence(page, EXECUTION_SEQUENCE);

    await page.goto(`${BASE_URL}/reporting`);
    await page.click('button:has-text("Schedule")');

    const runButton = page.getByRole('button', { name: /run now/i });
    await expect(runButton).toBeVisible();
    await runButton.click();

    // Pending chip from 202 body
    await expect(page.getByText('Pending')).toBeVisible({ timeout: 3000 });

    // Running chip after first poll
    await expect(page.getByText('Running')).toBeVisible({ timeout: 6000 });

    // Completed chip after second poll
    await expect(page.getByText('Completed')).toBeVisible({ timeout: 10000 });

    // Output section appears
    await expect(page.getByText('Download output')).toBeVisible({ timeout: 3000 });
    await expect(page.getByText('1,420')).toBeVisible(); // rows_processed
    await expect(page.getByText('3.2s')).toBeVisible(); // execution_time_ms
  });

  test('202 → terminal failed renders error_message', async ({ page }) => {
    await mockTriggerOk(page);
    await mockPollingFailedSequence(page, EXECUTION_FAILED_SEQUENCE);

    await page.goto(`${BASE_URL}/reporting`);
    await page.click('button:has-text("Schedule")');

    const runButton = page.getByRole('button', { name: /run now/i });
    await runButton.click();

    await expect(page.getByText('Pending')).toBeVisible({ timeout: 3000 });
    await expect(page.getByText('Failed')).toBeVisible({ timeout: 8000 });
    await expect(page.getByText(/Template rendering error/i)).toBeVisible();
  });

  test('503 dispatch_failed renders alert with execution_id', async ({ page }) => {
    await mockTrigger503(page);

    await page.goto(`${BASE_URL}/reporting`);
    await page.click('button:has-text("Schedule")');

    const runButton = page.getByRole('button', { name: /run now/i });
    await runButton.click();

    await expect(page.getByText('Dispatch failed')).toBeVisible({ timeout: 3000 });
    await expect(page.getByText(/Temporal cluster unavailable/i)).toBeVisible();
    await expect(page.getByText(/exec-dispatch-err/i)).toBeVisible();
  });
});
