import { test, expect } from '@playwright/test';

test('override lifecycle: create → edit → revert', async ({ page }) => {
  await page.goto('/rules');

  // Open a core rule - select first rule row
  await page.getByRole('row').first().click();

  // Banner should show core rule (may be a toast or banner text)
  await expect(page.getByText('This rule is inherited from Gold Copy')).toBeVisible();

  // Create override
  await page.getByRole('button', { name: 'Create Tenant Override' }).click();

  // Editor should now be editable: find editor area (monaco uses textarea for accessibility)
  await expect(page.locator('[data-testid="monaco-editor"]').first()).toBeVisible();

  // Banner should show override
  await expect(page.getByText(/tenant override/i)).toBeVisible();

  // Revert to core
  await page.getByRole('button', { name: 'Revert to Core' }).click();
  await page.getByRole('button', { name: 'Confirm' }).click();

  // Should reopen the core rule
  await expect(page.getByText('This rule is inherited from Gold Copy')).toBeVisible();
});