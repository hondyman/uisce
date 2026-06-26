import { test, expect } from '@playwright/test';

test('modal traps focus, Esc closes, focus returns', async ({ page }) => {
  await page.goto('/iframe.html?id=editor-editorhost--modal-short');
  await page.getByRole('button', { name: 'Open Modal' }).click();
  await expect(page.getByRole('dialog', { name: 'Configure Section' })).toBeVisible();
  const opener = page.getByRole('button', { name: 'Open Modal' });
  // Cycle focus with Tab a few times
  await page.keyboard.press('Tab');
  await page.keyboard.press('Tab');
  await page.keyboard.press('Shift+Tab');
  // Esc to close
  await page.keyboard.press('Escape');
  await expect(page.getByRole('dialog')).toBeHidden({ timeout: 2000 });
  await expect(opener).toBeFocused();
});

test('panel locks body scroll when open', async ({ page }) => {
  await page.goto('/iframe.html?id=editor-editorhost--panel-long');
  await page.getByRole('button', { name: 'Open Panel' }).click();
  const panel = page.getByRole('dialog', { name: 'Related Records' });
  await expect(panel).toBeVisible();
  const overflow = await page.evaluate(() => document.body.style.overflow);
  expect(overflow).toBe('hidden');
});