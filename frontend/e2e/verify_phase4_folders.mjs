import { chromium } from 'playwright';
import path from 'path';
import fs from 'fs';

const ARTIFACT_DIR = process.env.ARTIFACT_DIR || '/Users/eganpj/.gemini/antigravity/brain/07b374e7-ed95-48ac-90a7-6b1b317e4c47';
const LOCAL_SCREENSHOT_DIR = path.resolve(process.cwd(), 'screenshots');

if (!fs.existsSync(ARTIFACT_DIR)) { fs.mkdirSync(ARTIFACT_DIR, { recursive: true }); }
if (!fs.existsSync(LOCAL_SCREENSHOT_DIR)) { fs.mkdirSync(LOCAL_SCREENSHOT_DIR, { recursive: true }); }

const BASE_URL = 'http://localhost:5173';

async function saveScreenshot(page, filename) {
  const localPath = path.join(LOCAL_SCREENSHOT_DIR, filename);
  const artifactPath = path.join(ARTIFACT_DIR, filename);
  await page.screenshot({ path: localPath });
  try {
    fs.copyFileSync(localPath, artifactPath);
  } catch (e) {
    console.warn(`Could not copy screenshot to ${artifactPath}:`, e.message);
  }
}

async function run() {
  console.log('Launching browser for Phase 4 Folders verification...');
  const browser = await chromium.launch({
    headless: true,
  });
  const context = await browser.newContext({
    viewport: { width: 1440, height: 900 },
  });

  const page = await context.newPage();

  page.on('console', msg => {
    const text = msg.text();
    if (text.includes('[apiClient]') || text.includes('[Mock Folders]')) {
      console.log('  [Browser]', text);
    }
  });
  page.on('pageerror', err => console.error('  [Browser Error]', err.message));

  // Test data representing:
  // 1. Core report
  // 2. Custom report (Tenant Holdings Breakdown)
  // 3. Personal report (My Private Daily Alpha)
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
  ];

  // Folder state in-memory mock
  let mockFolders = [];
  // Map: folderId -> Set of template_ids
  const folderItems = new Map();

  // Intercept Reports API routes
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

  await page.route('**/api/v1/reports/*/favorite', async (route) => {
    const url = route.request().url();
    const id = url.split('/reports/')[1].split('/favorite')[0];
    const report = mockReports.find(r => r.id === id);
    if (route.request().method() === 'PUT') {
      if (report) report.is_favorite = true;
      await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ success: true }) });
    } else if (route.request().method() === 'DELETE') {
      if (report) report.is_favorite = false;
      await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ success: true }) });
    }
  });

  // Intercept Folder API routes
  await page.route('**/api/v1/reports/folders**', async (route) => {
    const req = route.request();
    const url = req.url();
    const method = req.method();

    // 1. Folder Items endpoints: /api/v1/reports/folders/:id/items or /api/v1/reports/folders/:id/items/:templateId
    if (url.includes('/items')) {
      const parts = url.split('/reports/folders/')[1].split('/items');
      const folderId = parts[0];
      const subPath = parts[1] || '';

      if (method === 'GET') {
        const items = folderItems.get(folderId) || new Set();
        console.log(`[Mock Folders] GET items for ${folderId}:`, Array.from(items));
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify(Array.from(items)),
        });
        return;
      }

      if (method === 'POST') {
        const data = JSON.parse(req.postData() || '{}');
        const templateId = data.template_id || data.report_id;
        let items = folderItems.get(folderId);
        if (!items) {
          items = new Set();
          folderItems.set(folderId, items);
        }
        items.add(templateId);
        console.log(`[Mock Folders] ADD item ${templateId} to folder ${folderId}`);
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({ folder_id: folderId, template_id: templateId }),
        });
        return;
      }

      if (method === 'DELETE') {
        const templateId = subPath.replace('/', '').split('?')[0];
        const items = folderItems.get(folderId);
        if (items) {
          items.delete(templateId);
        }
        console.log(`[Mock Folders] REMOVE item ${templateId} from folder ${folderId}`);
        await route.fulfill({
          status: 204,
        });
        return;
      }
    }

    // 2. Collection root: /api/v1/reports/folders (GET or POST)
    const urlPath = url.split('?')[0];
    if (urlPath.endsWith('/api/v1/reports/folders')) {
      if (method === 'GET') {
        const list = mockFolders.map(f => {
          const items = folderItems.get(f.id) || new Set();
          return {
            ...f,
            item_count: items.size,
            report_count: items.size,
          };
        });
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify(list),
        });
        return;
      }

      if (method === 'POST') {
        const data = JSON.parse(req.postData() || '{}');
        const name = (data.name || '').trim();
        const parent_id = data.parent_id || null;

        // Duplicate sibling check (409 Conflict)
        const isDuplicate = mockFolders.some(f =>
          (f.parent_id || null) === parent_id &&
          f.name.toLowerCase() === name.toLowerCase()
        );

        if (isDuplicate) {
          console.log(`[Mock Folders] 409 Conflict: duplicate sibling folder "${name}" under parent ${parent_id}`);
          await route.fulfill({
            status: 409,
            contentType: 'application/json',
            body: JSON.stringify({ error: `A folder with name '${name}' already exists in this location.` }),
          });
          return;
        }

        const newFolder = {
          id: 'folder-' + Date.now() + '-' + Math.floor(Math.random() * 1000),
          name,
          parent_id,
          tenant_id: '11111111-1111-1111-1111-111111111111',
          user_id: '8952c905-ac5c-474b-bfc5-87a48c67867f',
          created_at: new Date().toISOString(),
          updated_at: new Date().toISOString(),
          item_count: 0,
          report_count: 0,
        };
        mockFolders.push(newFolder);
        folderItems.set(newFolder.id, new Set());
        console.log(`[Mock Folders] Created folder "${name}" (${newFolder.id})`);
        await route.fulfill({
          status: 201,
          contentType: 'application/json',
          body: JSON.stringify(newFolder),
        });
        return;
      }
    }

    // 3. Folder instance: /api/v1/reports/folders/:id (PUT or DELETE)
    const folderId = url.split('/reports/folders/')[1].split('?')[0];

    if (method === 'PUT') {
      const data = JSON.parse(req.postData() || '{}');
      const target = mockFolders.find(f => f.id === folderId);
      if (!target) {
        await route.fulfill({ status: 404, contentType: 'application/json', body: JSON.stringify({ error: 'Not found' }) });
        return;
      }
      const name = (data.name || '').trim();
      const isDuplicate = mockFolders.some(f =>
        f.id !== folderId &&
        (f.parent_id || null) === (target.parent_id || null) &&
        f.name.toLowerCase() === name.toLowerCase()
      );
      if (isDuplicate) {
        console.log(`[Mock Folders] 409 Conflict on rename: folder "${name}" already exists`);
        await route.fulfill({
          status: 409,
          contentType: 'application/json',
          body: JSON.stringify({ error: `A folder with name '${name}' already exists in this location.` }),
        });
        return;
      }
      target.name = name;
      console.log(`[Mock Folders] Renamed folder ${folderId} to "${name}"`);
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify(target),
      });
      return;
    }

    if (method === 'DELETE') {
      const toDelete = new Set([folderId]);
      let added = true;
      while (added) {
        added = false;
        for (const f of mockFolders) {
          if (f.parent_id && toDelete.has(f.parent_id) && !toDelete.has(f.id)) {
            toDelete.add(f.id);
            added = true;
          }
        }
      }
      mockFolders = mockFolders.filter(f => !toDelete.has(f.id));
      for (const id of toDelete) {
        folderItems.delete(id);
      }
      console.log(`[Mock Folders] Deleted folder ${folderId} and descendants:`, Array.from(toDelete));
      await route.fulfill({ status: 204 });
      return;
    }

    await route.continue();
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

  // ==========================================
  // Step A: Root & Nested Folder Creation
  // ==========================================
  console.log('\n--- Step A: Creating Root Folder "Q3 Board Deck" ---');
  const newFolderBtn = page.locator('button', { hasText: 'New Folder' });
  await newFolderBtn.click();
  await page.waitForSelector('div[role="dialog"]:has-text("New Folder")');

  const dialog = page.locator('div[role="dialog"]');
  const nameInput = dialog.locator('input[type="text"]').first();
  await nameInput.fill('Q3 Board Deck');
  await dialog.locator('button', { hasText: 'Create' }).click();

  // Wait for dialog to close and folder to appear in sidebar
  await page.waitForSelector('.MuiListItem-root:has-text("Q3 Board Deck")');
  console.log('✓ Root folder "Q3 Board Deck" created successfully in sidebar tree');

  console.log('--- Step A2: Creating Subfolder "Financials" under "Q3 Board Deck" ---');
  const rootItem = page.locator('.MuiListItem-root', { hasText: 'Q3 Board Deck' });
  await rootItem.hover();
  const addSubfolderBtn = rootItem.locator('button[aria-label="Add subfolder to Q3 Board Deck"]');
  await addSubfolderBtn.click();

  await page.waitForSelector('div[role="dialog"]:has-text("New Subfolder in \\"Q3 Board Deck\\"")');
  const subDialog = page.locator('div[role="dialog"]');
  const subNameInput = subDialog.locator('input[type="text"]').first();
  await subNameInput.fill('Financials');
  await subDialog.locator('button', { hasText: 'Create' }).click();

  // Wait for subfolder to appear
  await page.waitForSelector('.MuiListItem-root:has-text("Financials")');
  console.log('✓ Subfolder "Financials" created and nested under "Q3 Board Deck"');

  await saveScreenshot(page, 'phase4_01_folders_created.png');

  // ==========================================
  // Step B: Client-Side Folder Search
  // ==========================================
  console.log('\n--- Step B: Testing Client-Side Folder Search ---');
  const filterInput = page.locator('input[placeholder="Filter folders..."]');
  await filterInput.fill('Finan');
  await page.waitForTimeout(300);

  // Assert "Financials" is displayed and ancestor "Q3 Board Deck" is auto-expanded
  const financialsVisible = await page.locator('.MuiListItem-root:has-text("Financials")').isVisible();
  const rootVisible = await page.locator('.MuiListItem-root:has-text("Q3 Board Deck")').isVisible();
  if (!financialsVisible || !rootVisible) {
    throw new Error('Folder search failed to retain ancestor and matching child folder');
  }
  console.log('✓ Folder search for "Finan" correctly filters tree and keeps ancestor expanded');
  await saveScreenshot(page, 'phase4_02_folder_search.png');

  // Clear search filter
  await page.locator('button[aria-label="Clear folder search"]').click();
  await page.waitForTimeout(300);

  // ==========================================
  // Step C: Inline 409 Conflict Error
  // ==========================================
  console.log('\n--- Step C: Testing 409 Conflict on Duplicate Sibling ---');
  await rootItem.hover();
  const addSubfolderDuplicateBtn = rootItem.locator('button[aria-label="Add subfolder to Q3 Board Deck"]');
  await addSubfolderDuplicateBtn.click();

  await page.waitForSelector('div[role="dialog"]:has-text("New Subfolder in \\"Q3 Board Deck\\"")');
  const dupDialog = page.locator('div[role="dialog"]');
  const dupNameInput = dupDialog.locator('input[type="text"]').first();
  await dupNameInput.fill('Financials'); // Already exists as child of Q3 Board Deck!
  await dupDialog.locator('button', { hasText: 'Create' }).click();

  // Assert inline error alert
  await page.waitForSelector('text=A folder with this name already exists in this location.');
  console.log('✓ Inline 409 error displayed: "A folder with this name already exists in this location."');
  await saveScreenshot(page, 'phase4_03_folder_409_conflict.png');

  // Cancel dialog
  await dupDialog.locator('button', { hasText: 'Cancel' }).click();
  await page.waitForTimeout(300);

  // ==========================================
  // Step D: Report Filing & Count Badges
  // ==========================================
  console.log('\n--- Step D: Filing Report into Folder "Financials" ---');
  // Find report "Tenant Holdings Breakdown" in the table
  const customRow = page.locator('tr', { hasText: 'Tenant Holdings Breakdown' });
  await customRow.waitFor();
  const moreBtn = customRow.locator('button[aria-label="More options"]');
  await moreBtn.click();

  // Click "Add to Folder"
  await page.locator('.MuiMenuItem-root', { hasText: 'Add to Folder' }).click();
  await page.waitForSelector('div[role="dialog"]:has-text("Add \\"Tenant Holdings Breakdown\\" to Folder")');

  const filingDialog = page.locator('div[role="dialog"]');
  // Select radio for "Financials"
  await filingDialog.locator('label', { hasText: 'Financials' }).click();
  await filingDialog.locator('button', { hasText: 'Add' }).click();

  // Assert count badge on "Financials" updates to 1
  const financialsItem = page.locator('.MuiListItem-root', { hasText: 'Financials' });
  await page.waitForSelector('.MuiListItem-root:has-text("Financials") .MuiChip-label:has-text("1")');
  console.log('✓ Report successfully filed; "Financials" count badge updated to 1');
  await saveScreenshot(page, 'phase4_04_report_filed_count_badge.png');

  // ==========================================
  // Step E: Folder Selection & Filtering
  // ==========================================
  console.log('\n--- Step E: Clicking Folder "Financials" to Filter Table ---');
  await financialsItem.locator('.MuiListItemButton-root').click();

  // Assert Active Folder Filter Banner appears
  await page.waitForSelector('text=Folder: Financials');
  await page.waitForSelector('text=1 report');
  console.log('✓ Active folder banner displayed: "Folder: Financials (1 report)"');

  // Assert reports table contains only 1 report: "Tenant Holdings Breakdown"
  const rowCount = await page.locator('tbody tr').count();
  const hasTenantHoldings = await page.locator('tbody tr', { hasText: 'Tenant Holdings Breakdown' }).isVisible();
  const hasCore = await page.locator('tbody tr', { hasText: 'Core Executive Summary' }).isVisible();
  if (rowCount !== 1 || !hasTenantHoldings || hasCore) {
    throw new Error(`Folder filter mismatch: expected 1 row ("Tenant Holdings Breakdown"), got ${rowCount}`);
  }
  console.log('✓ Reports table correctly filtered by active folder items');
  await saveScreenshot(page, 'phase4_05_folder_filter_active.png');

  // ==========================================
  // Step F: Facet-vs-Folder Decoupling Assertion
  // ==========================================
  console.log('\n--- Step F: Testing Facet-vs-Folder Decoupling Rule ---');
  // Clicking ANY facet (Favorites, All Reports, Personal, etc.) must immediately reset currentFolder to null
  const favFacet = page.locator('.MuiListItemButton-root', { hasText: 'Favorites' });
  await favFacet.click();
  await page.waitForTimeout(400);

  // Assert banner is gone
  const bannerCount = await page.locator('text=Folder: Financials').count();
  if (bannerCount !== 0) {
    throw new Error('Facet-vs-Folder decoupling violated: active folder banner still visible after clicking facet');
  }

  // Assert table now shows library-wide favorites ("My Private Daily Alpha")
  const favVisible = await page.locator('tbody tr', { hasText: 'My Private Daily Alpha' }).isVisible();
  if (!favVisible) {
    throw new Error('Expected favorite report "My Private Daily Alpha" to be visible under Favorites facet');
  }
  console.log('✓ Facet-vs-Folder decoupling confirmed: clicking facet reset folder selection and displays library favorites');
  await saveScreenshot(page, 'phase4_06_facet_decoupling_reset.png');

  // ==========================================
  // Step G: Move Semantics
  // ==========================================
  console.log('\n--- Step G: Moving Report from "Financials" to "Q3 Board Deck" ---');
  // Re-select "Financials"
  await financialsItem.locator('.MuiListItemButton-root').click();
  await page.waitForSelector('text=Folder: Financials');

  const rowInFolder = page.locator('tbody tr', { hasText: 'Tenant Holdings Breakdown' });
  const rowMoreBtn = rowInFolder.locator('button[aria-label="More options"]');
  await rowMoreBtn.click();

  // Click "Move to Folder"
  await page.locator('.MuiMenuItem-root', { hasText: 'Move to Folder' }).click();
  await page.waitForSelector('div[role="dialog"]:has-text("Move \\"Tenant Holdings Breakdown\\" to Folder")');

  const moveDialog = page.locator('div[role="dialog"]');
  // Select "Q3 Board Deck"
  await moveDialog.locator('label', { hasText: 'Q3 Board Deck' }).click();
  await moveDialog.locator('button', { hasText: 'Move' }).click();

  // Verify count in "Financials" decrements to 0
  await page.waitForSelector('.MuiListItem-root:has-text("Financials") .MuiChip-label:has-text("0")');
  // Verify count in "Q3 Board Deck" increments to 1
  await page.waitForSelector('.MuiListItem-root:has-text("Q3 Board Deck") .MuiChip-label:has-text("1")');
  console.log('✓ Move semantics verified: source count decremented to 0, target count incremented to 1');

  // Switch to "Q3 Board Deck" folder and verify report is present
  await rootItem.locator('.MuiListItemButton-root').first().click();
  await page.waitForSelector('text=Folder: Q3 Board Deck');
  await page.waitForSelector('tbody tr:has-text("Tenant Holdings Breakdown")');
  console.log('✓ Report visible in target folder "Q3 Board Deck"');
  await saveScreenshot(page, 'phase4_07_report_moved.png');

  // ==========================================
  // Step H: Folder Deletion & Retention (Self-Cleaning)
  // ==========================================
  console.log('\n--- Step H: Deleting Folder "Financials" & Root "Q3 Board Deck" (Self-Cleaning) ---');
  await financialsItem.hover();
  const deleteBtn = financialsItem.locator('button[aria-label="Delete folder Financials"]');
  await deleteBtn.click();

  await page.waitForSelector('div[role="dialog"]:has-text("Delete Folder \\"Financials\\"")');
  const delDialog = page.locator('div[role="dialog"]');
  await delDialog.locator('button', { hasText: 'Delete' }).click();

  // Wait for "Financials" to disappear from the tree
  await page.waitForSelector('.MuiListItem-root:has-text("Financials")', { state: 'detached' });
  console.log('✓ Subfolder "Financials" deleted from tree');

  // Self-cleaning: Also delete root folder "Q3 Board Deck" so no state is left behind in alpha
  await rootItem.hover();
  const deleteRootBtn = rootItem.locator('button[aria-label="Delete folder Q3 Board Deck"]');
  await deleteRootBtn.click();

  await page.waitForSelector('div[role="dialog"]:has-text("Delete Folder \\"Q3 Board Deck\\"")');
  const delRootDialog = page.locator('div[role="dialog"]');
  await delRootDialog.locator('button', { hasText: 'Delete' }).click();

  // Wait for "Q3 Board Deck" to disappear from tree
  await page.waitForSelector('.MuiListItem-root:has-text("Q3 Board Deck")', { state: 'detached' });
  console.log('✓ Root folder "Q3 Board Deck" deleted from tree (full self-cleaning complete)');

  // Return to All Reports and verify report is retained
  await page.locator('.MuiListItemButton-root', { hasText: 'All Reports' }).click();
  await page.waitForTimeout(300);
  await page.waitForSelector('tbody tr:has-text("Tenant Holdings Breakdown")');
  console.log('✓ Underlying report retained in library after folder deletion');
  await saveScreenshot(page, 'phase4_08_folder_deleted.png');

  console.log('\n========================================================');
  console.log('ALL PHASE 4 PLAYWRIGHT E2E FOLDER CHECKS PASSED 100%!');
  console.log('========================================================\n');

  await browser.close();
}

run().catch((err) => {
  console.error('Playwright E2E Test run failed:', err);
  process.exit(1);
});
