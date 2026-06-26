import { test, expect } from '@playwright/test';

test.setTimeout(120000);

test('Schema Explorer shows business terms and ERD', async ({ page }) => {
  // Open the app
  await page.goto('http://localhost:5173/', { waitUntil: 'domcontentloaded' });
  await page.waitForSelector('#root', { timeout: 30000 });
  // Click the Schema Explorer menu item
  await page.click('text=Schema Explorer');
  // Wait for datasource selector
  await page.waitForSelector('button', { timeout: 30000 });
  // Click the first datasource button
  await page.click('button', { timeout: 10000 });
  // Wait for TabbedModal to load
  await page.waitForSelector('.tabbed-modal-container', { timeout: 30000 });

  // Assert Business Terms tab and its content
  await page.click('text=Business Terms');
  await page.waitForSelector('.term-name', { timeout: 10000 });
  const firstTerm = await page.textContent('.term-name');
  expect(firstTerm).toBeTruthy();

  // Assert ERD tab renders
  await page.click('text=ERD Diagram');
  await page.waitForSelector('.diagram-tab', { timeout: 10000 });
  const diagramExists = await page.$('.diagram-tab');
  expect(diagramExists).not.toBeNull();
});
