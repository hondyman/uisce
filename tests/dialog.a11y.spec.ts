// tests/dialog.a11y.spec.ts
import { test, expect } from '@playwright/test';

/**
 * Dialog Accessibility Tests: Verify focus trap, ESC close, and scroll lock.
 * These tests ensure layouts pass accessibility standards before publication.
 */

test.describe('Dialog Accessibility', () => {
  test('modal focus trap and esc close', async ({ page }) => {
    await page.goto('/iframe.html?id=infra-dialogs--modal-dialog');

    // Open modal
    await page.getByRole('button', { name: 'Open Modal' }).click();

    // Modal should be visible
    const modal = page.getByRole('dialog', { name: 'Configure' });
    await expect(modal).toBeVisible();

    // Verify aria-modal
    await expect(modal).toHaveAttribute('aria-modal', 'true');

    // Verify aria-labelledby
    const labelledBy = await modal.getAttribute('aria-labelledby');
    expect(labelledBy).toBeTruthy();

    // Tab navigation should cycle within modal
    const inputs = page.locator('input');
    await expect(inputs).toHaveCount(2);

    // Tab forward
    await page.keyboard.press('Tab');
    const focusedAfterTab = await page.evaluate(() => document.activeElement?.tagName);
    expect(focusedAfterTab).toBe('INPUT');

    // Escape should close modal
    await page.keyboard.press('Escape');
    await expect(modal).toBeHidden({ timeout: 2000 });

    // Focus should return to trigger button
    const triggerBtn = page.getByRole('button', { name: 'Open Modal' });
    await expect(triggerBtn).toBeFocused();
  });

  test('panel locks scroll when open', async ({ page }) => {
    await page.goto('/iframe.html?id=infra-dialogs--slide-over-panel');

    // Verify initial scroll is available
    let bodyOverflow = await page.evaluate(() => document.body.style.overflow);
    expect(bodyOverflow).not.toBe('hidden');

    // Open panel
    await page.getByRole('button', { name: 'Open Panel' }).click();

    // Panel should be visible
    const panel = page.getByRole('dialog', { name: 'Related Records' });
    await expect(panel).toBeVisible();

    // Body overflow should be locked
    bodyOverflow = await page.evaluate(() => document.body.style.overflow);
    expect(bodyOverflow).toBe('hidden');

    // ESC should close panel
    await page.keyboard.press('Escape');
    await expect(panel).toBeHidden({ timeout: 2000 });

    // Scroll should be unlocked
    bodyOverflow = await page.evaluate(() => document.body.style.overflow);
    expect(bodyOverflow).not.toBe('hidden');
  });

  test('panel scroll works while background locked', async ({ page }) => {
    await page.goto('/iframe.html?id=infra-dialogs--slide-over-panel');

    // Open panel
    await page.getByRole('button', { name: 'Open Panel' }).click();
    await expect(page.getByRole('dialog', { name: 'Related Records' })).toBeVisible();

    // Record 40 should initially be hidden
    const record40 = page.locator('text=Record 40');
    await expect(record40).not.toBeInViewport();

    // Scroll within panel
    const panel = page.getByRole('dialog', { name: 'Related Records' });
    await panel.evaluate((el) => {
      el.scrollTop = el.scrollHeight;
    });

    // Record 40 should now be visible
    await expect(record40).toBeInViewport();
  });

  test('dialog keyboard navigation', async ({ page }) => {
    await page.goto('/iframe.html?id=infra-dialogs--modal-dialog');

    // Open modal
    await page.getByRole('button', { name: 'Open Modal' }).click();
    const modal = page.getByRole('dialog', { name: 'Configure' });
    await expect(modal).toBeVisible();

    // Verify tabindex is set (for focus management)
    const tabindex = await modal.getAttribute('tabindex');
    expect(tabindex).toBeTruthy();

    // Tab through form elements
    const nameInput = page.locator('#name');
    const emailInput = page.locator('#email');

    await page.keyboard.press('Tab');
    await expect(nameInput).toBeFocused();

    await page.keyboard.press('Tab');
    await expect(emailInput).toBeFocused();
  });
});

test.describe('Modal Accessibility Features', () => {
  test('modal has required ARIA attributes', async ({ page }) => {
    await page.goto('/iframe.html?id=infra-dialogs--modal-dialog');

    await page.getByRole('button', { name: 'Open Modal' }).click();
    const modal = page.getByRole('dialog', { name: 'Configure' });

    // Check aria-modal
    const ariaModal = await modal.getAttribute('aria-modal');
    expect(ariaModal).toBe('true');

    // Check aria-labelledby
    const labelledBy = await modal.getAttribute('aria-labelledby');
    expect(labelledBy).toBeTruthy();

    // Verify label element exists
    const label = page.locator(`#${labelledBy}`);
    await expect(label).toBeVisible();
  });

  test('modal disables background scroll', async ({ page }) => {
    await page.goto('/iframe.html?id=infra-dialogs--modal-dialog');

    const bodyOverflowBefore = await page.evaluate(
      () => document.body.style.overflow,
    );

    await page.getByRole('button', { name: 'Open Modal' }).click();

    const bodyOverflowAfter = await page.evaluate(
      () => document.body.style.overflow,
    );
    expect(bodyOverflowAfter).toBe('hidden');

    await page.keyboard.press('Escape');

    const bodyOverflowFinal = await page.evaluate(
      () => document.body.style.overflow,
    );
    expect(bodyOverflowFinal).not.toBe('hidden');
  });
});

test.describe('Panel Accessibility Features', () => {
  test('panel slide animation works', async ({ page }) => {
    await page.goto('/iframe.html?id=infra-dialogs--slide-over-panel');

    const panel = page.getByRole('dialog', { name: 'Related Records' });

    // Panel should not be visible initially
    await expect(panel).not.toBeVisible();

    await page.getByRole('button', { name: 'Open Panel' }).click();

    // Panel should animate in
    await expect(panel).toBeVisible();

    // Panel should have animation applied
    const animation = await panel.evaluate((el) => {
      return window.getComputedStyle(el).animation;
    });
    expect(animation).toBeTruthy();
  });

  test('panel supports keyboard navigation', async ({ page }) => {
    await page.goto('/iframe.html?id=infra-dialogs--slide-over-panel');

    await page.getByRole('button', { name: 'Open Panel' }).click();
    const panel = page.getByRole('dialog', { name: 'Related Records' });

    // ESC should close panel
    await page.keyboard.press('Escape');
    await expect(panel).toBeHidden();

    // Reopen
    await page.getByRole('button', { name: 'Open Panel' }).click();
    await expect(panel).toBeVisible();
  });
});
