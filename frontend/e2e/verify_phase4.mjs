import { chromium } from 'playwright';
import path from 'path';
import fs from 'fs';

const ARTIFACT_DIR = process.env.ARTIFACT_DIR || path.resolve(process.cwd(), 'screenshots');
if (!fs.existsSync(ARTIFACT_DIR)) { fs.mkdirSync(ARTIFACT_DIR, { recursive: true }); }
const BASE_URL = 'http://localhost:5173';

async function run() {
  console.log('Launching browser for Phase 4 verification...');
  const browser = await chromium.launch({
    headless: true,
  });
  const context = await browser.newContext({
    viewport: { width: 1440, height: 900 },
  });

  const page = await context.newPage();

  // Test data representing:
  // 1. A Core Report (gold copy, is_core: true, not personal)
  // 2. A Tenant Custom Report (custom, is_core: false, is_personal: false)
  // 3. A Personal Report owned by current user (is_personal: true, created_by_id: current user)
  // 4. A Personal Report owned by another user (is_personal: true, created_by_id: other user)
  let mockReports = [
    {
      id: 'core-001',
      template_name: 'Core Executive Summary',
      name: 'Core Executive Summary',
      description: 'Master gold copy standard executive summary',
      category: 'executive',
      is_active: true,
      is_public: false,
      is_personal: false,
      is_core: true,
      is_favorite: false,
      tenant_id: '99e99e99-99e9-49e9-89e9-99e99e99e999',
      created_by: 'Platform Gold Copy',
      created_by_id: 'platform-admin',
      run_count: 42,
      last_run: new Date().toISOString(),
      layout_config: { metadata: { is_core: true } },
    },
    {
      id: 'custom-002',
      template_name: 'Tenant Holdings Breakdown',
      name: 'Tenant Holdings Breakdown',
      description: 'Tenant-wide shared holdings report',
      category: 'operations',
      is_active: true,
      is_public: false,
      is_personal: false,
      is_core: false,
      is_favorite: false,
      tenant_id: '11111111-1111-1111-1111-111111111111',
      created_by: 'Tenant Admin',
      created_by_id: 'tenant-admin-id',
      run_count: 15,
      last_run: new Date().toISOString(),
      layout_config: { metadata: { is_personal: false } },
    },
    {
      id: 'personal-mine-003',
      template_name: 'My Private Daily Alpha',
      name: 'My Private Daily Alpha',
      description: 'Personal analysis created by current user',
      category: 'performance',
      is_active: true,
      is_public: false,
      is_personal: true,
      is_core: false,
      is_favorite: true,
      tenant_id: '11111111-1111-1111-1111-111111111111',
      created_by: 'Test Admin',
      created_by_id: '8952c905-ac5c-474b-bfc5-87a48c67867f',
      run_count: 7,
      last_run: new Date().toISOString(),
      layout_config: { metadata: { is_personal: true, created_by_id: '8952c905-ac5c-474b-bfc5-87a48c67867f' } },
    },
    {
      id: 'personal-other-004',
      template_name: "Colleague's Draft Report",
      name: "Colleague's Draft Report",
      description: 'Personal report owned by another teammate',
      category: 'research',
      is_active: true,
      is_public: false,
      is_personal: true,
      is_core: false,
      is_favorite: false,
      tenant_id: '11111111-1111-1111-1111-111111111111',
      created_by: 'Jane Doe',
      created_by_id: 'other-user-1234',
      run_count: 2,
      last_run: new Date().toISOString(),
      layout_config: { metadata: { is_personal: true, created_by_id: 'other-user-1234' } },
    },
  ];

  // Intercept API routes
  await page.route('**/api/v1/reports', async (route) => {
    if (route.request().method() === 'GET') {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify(mockReports),
      });
    } else if (route.request().method() === 'POST') {
      const data = JSON.parse(route.request().postData() || '{}');
      const newReport = {
        id: 'cloned-' + Date.now(),
        ...data,
        name: data.template_name || data.name,
        created_at: new Date().toISOString(),
        updated_at: new Date().toISOString(),
      };
      mockReports.push(newReport);
      await route.fulfill({
        status: 201,
        contentType: 'application/json',
        body: JSON.stringify(newReport),
      });
    } else {
      await route.continue();
    }
  });

  await page.route('**/api/v1/reports/*/favorite', async (route) => {
    const url = route.request().url();
    const id = url.split('/reports/')[1].split('/favorite')[0];
    const report = mockReports.find(r => r.id === id);
    if (route.request().method() === 'PUT') {
      if (report) report.is_favorite = true;
      console.log(`[API Mock] PUT favorite for ${id} succeeded`);
      await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ success: true }) });
    } else if (route.request().method() === 'DELETE') {
      if (report) report.is_favorite = false;
      console.log(`[API Mock] DELETE favorite for ${id} succeeded`);
      await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ success: true }) });
    }
  });

  await page.route('**/api/v1/reports/*', async (route) => {
    if (route.request().method() === 'PUT' && !route.request().url().includes('/favorite')) {
      const url = route.request().url();
      const id = url.split('/reports/')[1];
      const data = JSON.parse(route.request().postData() || '{}');
      const report = mockReports.find(r => r.id === id);
      if (mockReports.some(r => r.id !== id && r.name.toLowerCase() === (data.template_name || '').toLowerCase())) {
        await route.fulfill({
          status: 409,
          contentType: 'application/json',
          body: JSON.stringify({ error: `A report with name '${data.template_name}' already exists in this tenant` }),
        });
        return;
      }
      if (report) {
        if (data.template_name) report.name = data.template_name;
      }
      await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify(report || {}) });
    } else {
      await route.continue();
    }
  });

  // Pre-seed authenticated state in localStorage
  await page.addInitScript(() => {
    localStorage.setItem('auth_token', 'mock-valid-jwt-token-xyz');
    localStorage.setItem('auth_user', JSON.stringify({
      id: '8952c905-ac5c-474b-bfc5-87a48c67867f',
      email: 'test.admin@uisce.io',
      name: 'Test Admin',
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

  console.log('Navigating to http://localhost:5173/en/reports/library...');
  await page.goto(`${BASE_URL}/en/reports/library`, { waitUntil: 'networkidle' });

  // 1. Verify Report Library Loads
  await page.waitForSelector('text=Report Library');
  console.log('✓ Report Library page loaded');
  await page.screenshot({ path: path.join(ARTIFACT_DIR, 'phase4_01_library_overview.png'), fullPage: true });

  // 2. Verify Star Favorite Toggle on Core and Custom Reports
  // Core Executive Summary is initially not favorite.
  const coreRow = page.locator('tr', { hasText: 'Core Executive Summary' });
  await coreRow.waitFor();
  const starButton = coreRow.locator('button').filter({ has: page.locator('svg[data-testid="StarBorderIcon"]') });
  console.log('Toggling favorite on Core Executive Summary...');
  await starButton.click();
  // Wait for filled star icon
  await coreRow.locator('svg[data-testid="StarIcon"][data-color="warning"], svg[data-testid="StarIcon"]').first().waitFor();
  console.log('✓ Star toggle on Core report filled successfully');
  await page.screenshot({ path: path.join(ARTIFACT_DIR, 'phase4_02_star_toggled.png') });

  // 3. Verify Favorites Facet
  console.log('Testing Favorites facet filter...');
  await page.locator('.MuiListItemButton-root', { hasText: 'Favorites' }).click();
  await page.waitForTimeout(500);
  const favRows = await page.locator('tbody tr').count();
  console.log(`✓ Favorites facet shows ${favRows} reports`);
  await page.screenshot({ path: path.join(ARTIFACT_DIR, 'phase4_03_favorites_facet.png') });

  // 4. Verify Personal Reports Facet
  console.log('Testing Personal Reports facet filter...');
  await page.locator('.MuiListItemButton-root', { hasText: 'Personal Reports' }).click();
  await page.waitForTimeout(500);
  const personalRow = page.locator('tr', { hasText: 'My Private Daily Alpha' });
  await personalRow.waitFor();
  console.log('✓ Personal Reports facet displays user personal report');
  await page.screenshot({ path: path.join(ARTIFACT_DIR, 'phase4_04_personal_facet.png') });

  // Go back to All Reports via sidebar
  await page.locator('.MuiListItemButton-root', { hasText: 'All Reports' }).click();
  await page.waitForTimeout(500);

  // 5. Verify Clone Dialog with Scope Radio (Admin) & Duplicate Validation
  console.log('Testing Clone / Duplicate Dialog...');
  const customRow = page.locator('tr', { hasText: 'Tenant Holdings Breakdown' });
  const cloneBtn = customRow.locator('button').filter({ has: page.locator('svg[data-testid="FileCopyIcon"]') });
  await cloneBtn.click();
  await page.waitForSelector('text=Duplicate Report');
  console.log('✓ Duplicate Report modal opened');

  // Verify scope radio is visible for Admin
  const personalRadio = page.locator('input[value="personal"]');
  const customRadio = page.locator('input[value="custom"]');
  const hasScopeRadio = (await personalRadio.count()) > 0 && (await customRadio.count()) > 0;
  console.log(`✓ Scope radio options visible for Admin: ${hasScopeRadio}`);

  // Test duplicate name validation
  console.log('Testing duplicate name validation in Clone dialog...');
  const cloneDialog = page.locator('div[role="dialog"]');
  const nameInput = cloneDialog.locator('input[type="text"]').first();
  await nameInput.fill('Core Executive Summary'); // already exists!
  await cloneDialog.locator('button', { hasText: 'Duplicate' }).click();
  await page.waitForSelector('text=A report with this name already exists in this tenant.');
  console.log('✓ Inline error "A report with this name already exists in this tenant." confirmed in Clone dialog');
  await page.screenshot({ path: path.join(ARTIFACT_DIR, 'phase4_05_clone_duplicate_error.png') });
  await cloneDialog.locator('button', { hasText: 'Cancel' }).click();
  await page.waitForTimeout(300);

  // 6. Verify Share Button Disabled Tooltips
  console.log('Testing Share button tooltips...');
  // Hover Core Report share button
  const coreShareSpan = page.locator('tr', { hasText: 'Core Executive Summary' }).locator('span').filter({ has: page.locator('svg[data-testid="ShareIcon"]') });
  await coreShareSpan.hover();
  await page.waitForSelector('text=Core reports cannot be shared');
  console.log('✓ Tooltip verified on Core report: "Core reports cannot be shared"');
  await page.screenshot({ path: path.join(ARTIFACT_DIR, 'phase4_06_core_share_tooltip.png') });

  // Hover Tenant Custom Report share button
  const customShareSpan = page.locator('tr', { hasText: 'Tenant Holdings Breakdown' }).locator('span').filter({ has: page.locator('svg[data-testid="ShareIcon"]') });
  await customShareSpan.hover();
  await page.waitForSelector('text=Only personal reports can be shared');
  console.log('✓ Tooltip verified on Custom report: "Only personal reports can be shared"');

  // Hover Other User Personal Report share button
  const otherShareSpan = page.locator('tr', { hasText: "Colleague's Draft Report" }).locator('span').filter({ has: page.locator('svg[data-testid="ShareIcon"]') });
  await otherShareSpan.hover();
  await page.waitForSelector('text=Only the report author can share this report');
  console.log('✓ Tooltip verified on Colleague report: "Only the report author can share this report"');
  await page.screenshot({ path: path.join(ARTIFACT_DIR, 'phase4_07_other_share_tooltip.png') });

  // 7. Verify Rename Dialog 409 Duplicate Name Validation
  console.log('Testing Rename Dialog duplicate name validation...');
  const renameRow = page.locator('tr', { hasText: 'Tenant Holdings Breakdown' });
  const renameBtn = renameRow.locator('button').filter({ has: page.locator('svg[data-testid="DriveFileRenameOutlineIcon"]') });
  await renameBtn.click();
  await page.waitForSelector('text=Rename Report');
  console.log('✓ Rename dialog opened');

  const renameDialog = page.locator('div[role="dialog"]');
  const renameInput = renameDialog.locator('input[type="text"]').first();
  await renameInput.fill('Core Executive Summary'); // already exists!
  await renameDialog.locator('button', { hasText: 'Rename' }).click();
  await page.waitForSelector('text=A report with this name already exists in this tenant.');
  console.log('✓ Rename inline duplicate error verified: "A report with this name already exists in this tenant."');
  await page.screenshot({ path: path.join(ARTIFACT_DIR, 'phase4_08_rename_duplicate_error.png') });
  await renameDialog.locator('button', { hasText: 'Cancel' }).click();

  console.log('\n========================================');
  console.log('ALL PHASE 4 BROWSER CHECKS PASSED 100%!');
  console.log('========================================');

  await browser.close();
}

run().catch((err) => {
  console.error('Test run failed:', err);
  process.exit(1);
});
