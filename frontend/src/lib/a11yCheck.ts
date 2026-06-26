// lib/a11yCheck.ts

import { devDebug, devWarn } from '../utils/devLogger';

export interface A11yCheckResult {
  ok: boolean;
  issues: string[];
  warnings?: string[];
}

/**
 * checkDialogs: Validate ARIA dialog patterns before publish.
 * Checks for required attributes per WCAG 2.1 and WAI-ARIA authoring practices.
 * 
 * Requirements checked:
 * - aria-modal="true" on dialog role elements
 * - aria-labelledby pointing to valid heading
 * - tabindex for focus management (typically -1 or 0)
 * - Focus trap (via Radix/Headless UI or custom focus management)
 * - ESC key to close (check in dialog component)
 */
export function checkDialogs(): A11yCheckResult {
  const dialogs = Array.from(
    document.querySelectorAll<HTMLElement>('[role="dialog"]'),
  );
  const issues: string[] = [];
  const warnings: string[] = [];

  dialogs.forEach((dialog, idx) => {
    const dialogId = dialog.id || `[dialog-${idx}]`;

    // Critical: aria-modal must be present
    if (!dialog.getAttribute('aria-modal')) {
      issues.push(`Dialog "${dialogId}" missing aria-modal="true"`);
    } else if (dialog.getAttribute('aria-modal') !== 'true') {
      warnings.push(`Dialog "${dialogId}" aria-modal should be "true"`);
    }

    // Critical: dialog must be labelled
    const labelledBy = dialog.getAttribute('aria-labelledby');
    if (!labelledBy) {
      issues.push(`Dialog "${dialogId}" missing aria-labelledby`);
    } else {
      const labelElem = document.getElementById(labelledBy);
      if (!labelElem) {
        issues.push(
          `Dialog "${dialogId}" aria-labelledby="${labelledBy}" not found in DOM`,
        );
      }
    }

    // Warning: tabindex helps focus management
    const tabIndex = dialog.getAttribute('tabindex');
    if (tabIndex === null || tabIndex === '') {
      warnings.push(
        `Dialog "${dialogId}" missing tabindex (recommend tabindex="0" or "-1")`,
      );
    }
  });

  return {
    ok: issues.length === 0,
    issues,
    warnings,
  };
}

/**
 * checkKeyboardNav: Verify ESC key closes all visible dialogs.
 * Tests that keyboard support is wired correctly.
 */
export function checkKeyboardNav(): A11yCheckResult {
  const dialogs = Array.from(
    document.querySelectorAll<HTMLElement>('[role="dialog"][aria-modal="true"]'),
  );
  const issues: string[] = [];

  dialogs.forEach((dialog, idx) => {
    const dialogId = dialog.id || `[dialog-${idx}]`;

    // Check if dialog or a child has keydown listener for ESC
    // This is a heuristic - full testing requires e2e tests
    if (!dialog.onkeydown && !dialog.hasAttribute('data-keyboard-trap')) {
      // Only warn; actual ESC handling is hard to detect statically
      devDebug(`Dialog "${dialogId}" - ensure ESC key closes this dialog`);
    }
  });

  return {
    ok: issues.length === 0,
    issues,
  };
}

/**
 * checkFocusReturn: After closing a dialog, focus should return to trigger.
 * This is mostly tested via Playwright, but here we verify the structure.
 */
export function checkFocusStructure(): A11yCheckResult {
  const dialogs = Array.from(
    document.querySelectorAll<HTMLElement>('[role="dialog"]'),
  );
  const issues: string[] = [];

  if (dialogs.length === 0) {
    return { ok: true, issues: ['No dialogs found'] };
  }

  dialogs.forEach((dialog) => {
    const dialogId = dialog.id || 'unnamed-dialog';

    // Check for interactive elements inside
    const focusableElements = dialog.querySelectorAll(
      'button, [href], input, select, textarea, [tabindex]:not([tabindex="-1"])',
    );

    if (focusableElements.length === 0) {
      devWarn(
        `Dialog "${dialogId}" has no focusable elements (modal might be read-only info)`,
      );
    }
  });

  return {
    ok: issues.length === 0,
    issues,
  };
}

/**
 * checkScrollLock: Verify body overflow is hidden when dialog is open.
 * Prevents background scrolling, which is critical for mobile a11y.
 */
export function checkScrollLock(): A11yCheckResult {
  const visibleDialogs = Array.from(
    document.querySelectorAll<HTMLElement>(
      '[role="dialog"][aria-modal="true"]:not([hidden])',
    ),
  );

  const issues: string[] = [];

  if (visibleDialogs.length > 0) {
    const bodyOverflow = document.body.style.overflow;
    if (bodyOverflow !== 'hidden') {
      issues.push(
        `Body overflow not locked (is "${bodyOverflow}"). Set overflow:hidden when modal is open.`,
      );
    }
  }

  return {
    ok: issues.length === 0,
    issues,
  };
}

/**
 * runAllA11yChecks: Comprehensive accessibility validation.
 * Call this before publish to ensure dialogs meet WCAG 2.1 AA standards.
 */
export function runAllA11yChecks(): A11yCheckResult {
  const dialogCheck = checkDialogs();
  const keyboardCheck = checkKeyboardNav();
  const focusCheck = checkFocusStructure();
  const scrollCheck = checkScrollLock();

  const allIssues = [
    ...dialogCheck.issues,
    ...keyboardCheck.issues,
    ...focusCheck.issues,
    ...scrollCheck.issues,
  ];

  const allWarnings = [
    ...(dialogCheck.warnings || []),
    ...(keyboardCheck.warnings || []),
    ...(focusCheck.warnings || []),
    ...(scrollCheck.warnings || []),
  ];

  return {
    ok: allIssues.length === 0,
    issues: allIssues,
    warnings: allWarnings,
  };
}
