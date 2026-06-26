import { test, expect } from '@playwright/test';

test.setTimeout(120000);

test('Workflows mega-menu -> Approval Workflows loads and shows Data Seeding tab', async ({ page }) => {
  // Seed localStorage with selected tenant/datasource (mimic TenantContext cache)
  await page.addInitScript(() => {
    try {
      localStorage.setItem('selected_tenant', JSON.stringify({ id: 'test-tenant', display_name: 'Test Tenant' }));
      localStorage.setItem('selected_datasource', JSON.stringify({ id: 'test-ds', source_name: 'Test Datasource' }));
    } catch (e) {
      // ignore
    }
  });

  // Open the app root
  await page.goto('http://localhost:5173/', { waitUntil: 'domcontentloaded' });
  await page.waitForSelector('#root', { timeout: 30000 });

  // Open the category selector (shows current category label)
  await page.click('button:has-text("Tenants")', { timeout: 5000 }).catch(async () => {
    // If it isn't Tenants by label, click the first category button
    const btns = await page.$$('button');
    if (btns.length) await btns[0].click();
  });

  // Click 'Workflows & Ops' in the category dropdown
  await page.click('text=Workflows & Ops', { timeout: 5000 });

  // Click the 'Workflows' menu button to open its menu
  await page.click('button:has-text("Workflows")', { timeout: 5000 });

  // Click the Approval Workflows menu item
  await page.click('text=Approval Workflows', { timeout: 5000 });

  // Wait for the Approval Workflow header to appear
  await page.waitForSelector('text=Approval Workflow Manager', { timeout: 10000 });

  // Ensure the Data Seeding tab/button is present
  const seedingTab = await page.waitForSelector('text=Data Seeding', { timeout: 5000 });
  expect(seedingTab).not.toBeNull();
});
