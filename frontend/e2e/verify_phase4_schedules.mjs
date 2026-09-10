/**
 * Mocked API-Contract E2E Verification: Scheduled Reports & Subscriptions
 * 
 * Scope & Limitations:
 * - This test verifies the frontend UI workflow, dialog state, inline error handling (409 conflict),
 *   trigger refresh, and response rendering using deterministic mock route intercepts.
 * - Server-side security invariants (two-sided template-owner execution, tenant isolation,
 *   fail-closed auth, and synthetic database markers) are verified against the live database
 *   by the integration test suite (TestTriggerRun_twoSidedIdentity, TestScheduleModel_LiveAuditTrail,
 *   and TestReportScheduleAPI_AuthAndSecurity), NOT by this browser mock test.
 */

import { chromium } from 'playwright';
import path from 'path';
import fs from 'fs';

const LOCAL_SCREENSHOT_DIR = path.resolve(process.cwd(), 'screenshots');
const ARTIFACT_DIR = process.env.ARTIFACT_DIR || LOCAL_SCREENSHOT_DIR;

if (!fs.existsSync(LOCAL_SCREENSHOT_DIR)) { fs.mkdirSync(LOCAL_SCREENSHOT_DIR, { recursive: true }); }
if (!fs.existsSync(ARTIFACT_DIR)) { fs.mkdirSync(ARTIFACT_DIR, { recursive: true }); }

const BASE_URL = 'http://localhost:5173';

async function saveScreenshot(page, filename) {
  const localPath = path.join(LOCAL_SCREENSHOT_DIR, filename);
  await page.screenshot({ path: localPath });
  if (ARTIFACT_DIR !== LOCAL_SCREENSHOT_DIR) {
    try {
      const artifactPath = path.join(ARTIFACT_DIR, filename);
      fs.copyFileSync(localPath, artifactPath);
    } catch (e) {
      console.warn(`Could not copy screenshot to ARTIFACT_DIR:`, e.message);
    }
  }
}

async function run() {
  console.log('=== Phase 4 Mocked Contract E2E: Scheduled Reports & Subscriptions ===');
  const browser = await chromium.launch({
    headless: true,
  });
  const context = await browser.newContext({
    viewport: { width: 1440, height: 900 },
  });

  const page = await context.newPage();

  page.on('console', msg => {
    const text = msg.text();
    if (text.includes('[apiClient]') || text.includes('[Mock Schedules]')) {
      console.log('  [Browser]', text);
    }
  });
  page.on('pageerror', err => console.error('  [Browser Error]', err.message));

  // Fixture: Dedicated Test Report Template
  const TEST_TEMPLATE = {
    id: 'e2e-template-4444-8888-cccc-dddddddddddd',
    template_name: 'E2E Scheduled Valuation Report',
    name: 'E2E Scheduled Valuation Report',
    description: 'Mocked template for Phase 4 schedule UI workflow testing',
    category: 'operations',
    is_active: true,
    is_public: false,
    is_personal: false,
    is_core: false,
    is_favorite: false,
    tenant_id: '11111111-1111-1111-1111-111111111111',
    created_by: 'Portfolio Admin',
    created_by_id: 'template-owner-uuid-9999',
    run_count: 5,
    last_run: new Date().toISOString(),
    layout_config: { metadata: { is_personal: false, created_by_id: 'template-owner-uuid-9999' } },
  };

  let mockReports = [TEST_TEMPLATE];

  // In-memory schedules state for the test template
  let mockSchedules = [];
  let mockBatches = [];

  // 1. Intercept Reports API
  await page.route('**/api/v1/reports', async (route) => {
    if (route.request().method() === 'GET') {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify(mockReports),
      });
    } else {
      await route.continue();
    }
  });

  // 2. Intercept Option B Schedule endpoints: /api/v1/reports/{id}/schedules
  await page.route('**/api/v1/reports/*/schedules**', async (route) => {
    const req = route.request();
    const url = req.url();
    const method = req.method();

    // POST .../schedules/{sid}/run
    if (url.includes('/run') && method === 'POST') {
      const parts = url.split('/schedules/')[1].split('/run')[0];
      const sid = parts;
      console.log(`[Mock Schedules] Trigger run for schedule ${sid}`);
      
      const executionResult = {
        execution_id: 'exec-uuid-7777-8888-9999',
        output_url: 's3://report-artifacts-eu-west-1/tenant-11111111-1111-1111-1111-111111111111/e2e.pdf',
        status: 'synthetic',
        requested_by: TEST_TEMPLATE.created_by_id,
        tenant_id: TEST_TEMPLATE.tenant_id,
        template_id: TEST_TEMPLATE.id,
      };

      // Add a batch so the execution ledger updates
      mockBatches.unshift({
        id: 'batch-synthetic-001',
        schedule_id: sid,
        effective_date: new Date().toISOString().split('T')[0],
        total_clients: 12,
        successful_renders: 12,
        status: 'COMPLETED',
        started_at: new Date().toISOString(),
      });

      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify(executionResult),
      });
      return;
    }

    // DELETE .../schedules/{sid} -> soft-delete (returns 204)
    if (method === 'DELETE') {
      const sid = url.split('/schedules/')[1].split('?')[0];
      console.log(`[Mock Schedules] DELETE schedule ${sid}`);
      mockSchedules = mockSchedules.filter(s => s.id !== sid);
      await route.fulfill({ status: 204 });
      return;
    }

    // GET .../schedules (list for template)
    if (method === 'GET' && !url.includes('/run')) {
      console.log(`[Mock Schedules] GET list for template, count=${mockSchedules.length}`);
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify(mockSchedules),
      });
      return;
    }

    // POST .../schedules (create schedule)
    if (method === 'POST') {
      const body = JSON.parse(req.postData() || '{}');
      console.log(`[Mock Schedules] POST create schedule: "${body.schedule_name}"`);

      // Inline 409 Conflict check: name uniqueness per template
      const isDuplicate = mockSchedules.some(
        s => s.schedule_name.toLowerCase() === (body.schedule_name || '').toLowerCase()
      );
      if (isDuplicate) {
        console.log(`[Mock Schedules] 409 Conflict: schedule name "${body.schedule_name}" already exists`);
        await route.fulfill({
          status: 409,
          contentType: 'text/plain',
          body: 'Schedule name already exists for this report template',
        });
        return;
      }

      const newSched = {
        id: 'sched-' + Date.now(),
        tenant_id: TEST_TEMPLATE.tenant_id,
        template_id: TEST_TEMPLATE.id,
        schedule_name: body.schedule_name,
        cron_expression: body.cron_expression,
        region: body.region || 'us-west',
        calendar_id: null,
        start_of_day_time: '00:00:00',
        unscheduled_behavior: body.unscheduled_behavior || 'RUN_PREVIOUS_BUS_DAY',
        business_day_offset: body.business_day_offset || -1,
        burst_dimension: body.burst_dimension || 'client_id',
        export_format: body.export_format || 'PDF',
        is_active: true,
        created_at: new Date().toISOString(),
      };
      mockSchedules.push(newSched);
      await route.fulfill({
        status: 201,
        contentType: 'application/json',
        body: JSON.stringify(newSched),
      });
      return;
    }

    await route.continue();
  });

  // 3. Intercept batches subroute
  await page.route('**/api/reports/schedules/*/batches', async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify(mockBatches),
    });
  });

  // Pre-seed authenticated state in localStorage (admin caller with tenant context)
  await page.addInitScript(() => {
    localStorage.setItem('auth_token', 'mock-valid-jwt-token-xyz');
    localStorage.setItem('auth_user', JSON.stringify({
      id: 'calling-admin-uuid-1111',
      email: 'admin.caller@uisce.io',
      name: 'Admin Caller',
      role: 'admin',
      organization: 'Uisce Org',
      permissions: [],
      is_active: true,
      roles: ['admin'],
      is_admin: true,
      is_global_admin: true,
    }));
    localStorage.setItem('tenant_context', JSON.stringify({
      tenantId: '11111111-1111-1111-1111-111111111111',
      tenantName: 'Client Alpha Tenant',
      gold_copy: false,
    }));
  });

  try {
    console.log('\n--- Step 1: Navigating to Report Library ---');
    await page.goto(`${BASE_URL}/en/reports/library`, { waitUntil: 'networkidle' });
    await page.waitForSelector('text=Report Library');
    console.log('✓ Report Library page loaded successfully');

    // --- Step 2: Open Context Menu and Click Schedule ---
    console.log('\n--- Step 2: Opening Schedule Dialog on Test Report ---');
    const reportRow = page.locator('tr', { hasText: 'E2E Scheduled Valuation Report' });
    await reportRow.waitFor();
    const moreBtn = reportRow.locator('button[aria-label="More options"]');
    await moreBtn.click();

    // Click "Schedule" in context menu
    const scheduleMenuItem = page.locator('.MuiMenuItem-root', { hasText: 'Schedule' });
    await scheduleMenuItem.waitFor();
    await scheduleMenuItem.click();

    // --- Step 3: Assert Schedule Dialog Renders with Full Bursting Controls ---
    console.log('\n--- Step 3: Verifying Schedule Dialog Form Controls ---');
    await page.waitForSelector('div[role="dialog"]:has-text("Schedule Report — E2E Scheduled Valuation Report")');
    const dialog = page.locator('div[role="dialog"]');

    // Assert form fields are present in dialog
    await dialog.locator('label:has-text("Schedule Name")').waitFor();
    await dialog.locator('label:has-text("Cron Expression")').waitFor();
    await dialog.locator('label:has-text("Execution Region")').waitFor();
    await dialog.locator('label:has-text("Exchange Master Calendar")').waitFor();
    await dialog.locator('text=Client Partitioning & File Export').waitFor();
    await dialog.locator('text=Tenant-Isolated Mesh').waitFor();
    console.log('✓ Schedule dialog loaded with all expected form controls and mesh badges');
    await saveScreenshot(page, 'phase4_01_schedule_dialog_rendered.png');

    // --- Step 4: Save Active Schedule ---
    console.log('\n--- Step 4: Saving New Schedule "Daily Portfolio Valuation" ---');
    const nameInput = dialog.locator('input[type="text"]').first();
    await nameInput.fill('Daily Portfolio Valuation');

    const cronInput = dialog.locator('input[value="0 8 * * 1-5"]');
    await cronInput.waitFor();

    const saveBtn = dialog.locator('button', { hasText: 'Save Active Schedule' });
    await saveBtn.click();

    // Verify confirmation message
    await page.waitForSelector('text=Schedule registered and activated successfully!');
    console.log('✓ Schedule saved; confirmation banner displayed in dialog');
    await saveScreenshot(page, 'phase4_02_schedule_saved_active.png');

    // --- Step 5: Test 409 Conflict on Duplicate Name ---
    console.log('\n--- Step 5: Testing 409 Conflict on Duplicate Schedule Name ---');
    // Click Save again with the same name "Daily Portfolio Valuation"
    await saveBtn.click();

    // Verify error message is surfaced inline
    await page.waitForSelector('text=Schedule name already exists for this report template');
    console.log('✓ Inline 409 conflict message correctly surfaced in dialog UI: "Schedule name already exists for this report template"');
    await saveScreenshot(page, 'phase4_03_duplicate_schedule_409_conflict.png');

    // --- Step 6: Trigger Run and Verify UI Response Rendering ---
    console.log('\n--- Step 6: Triggering "Run Burst Now" & Verifying UI Response Handling ---');
    
    // Set up response listener for the run endpoint
    let capturedRunResponse = null;
    const runResponsePromise = page.waitForResponse(response => 
      response.url().includes('/run') && response.request().method() === 'POST'
    );

    const runNowBtn = dialog.locator('button', { hasText: 'Run Burst Now' });
    await runNowBtn.click();

    const runResponse = await runResponsePromise;
    capturedRunResponse = await runResponse.json();

    console.log('  [Captured Run Response]', capturedRunResponse);

    // Assert UI received the expected contract response shape:
    if (!capturedRunResponse.execution_id || capturedRunResponse.status !== 'synthetic') {
      throw new Error(`Unexpected run response shape: ${JSON.stringify(capturedRunResponse)}`);
    }
    console.log('✓ UI received valid execution payload (execution_id present, status="synthetic")');

    // Verify execution banner in UI
    await page.waitForSelector(`text=Run started successfully! Execution ID: ${capturedRunResponse.execution_id}`);
    console.log(`✓ Execution ID surfaced in UI: ${capturedRunResponse.execution_id}`);

    // Verify recent batches table rendered the updated batch
    await page.waitForSelector('text=Recent Burst Batches & Artifact Ledger');
    await page.waitForSelector('text=batch-sy...');
    console.log('✓ Batches ledger updated and rendered in UI');
    await saveScreenshot(page, 'phase4_04_run_triggered_synthetic.png');

    // --- Step 7: Self-Cleaning via DELETE Endpoint ---
    console.log('\n--- Step 7: Self-Cleaning via Mock DELETE Endpoint ---');
    const sidToDelete = mockSchedules[0]?.id;
    if (sidToDelete) {
      const deleteRes = await page.evaluate(async (sid) => {
        const res = await fetch(`/api/v1/reports/e2e-template-4444-8888-cccc-dddddddddddd/schedules/${sid}`, {
          method: 'DELETE',
        });
        return res.status;
      }, sidToDelete);

      if (deleteRes === 204) {
        console.log(`✓ Cleanup DELETE returned 204 No Content for schedule ${sidToDelete}`);
      } else {
        console.warn(`Cleanup DELETE returned status ${deleteRes}`);
      }
    }

    // Close dialog
    const closeBtn = dialog.locator('button', { hasText: 'Close' });
    await closeBtn.click();
    await page.waitForTimeout(300);
    await saveScreenshot(page, 'phase4_05_clean_state_closed.png');

    console.log('\n=== All Phase 4 Mocked Contract E2E Verification Steps Passed! ===\n');
  } catch (err) {
    console.error('Phase 4 Schedule Contract E2E Verification FAILED:', err);
    await saveScreenshot(page, 'phase4_failure.png');
    throw err;
  } finally {
    await browser.close();
  }
}

run().catch(err => {
  console.error(err);
  process.exit(1);
});
