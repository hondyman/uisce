// lib/analytics.ts

/**
 * logInteraction: Send user interaction event to backend analytics.
 * Uses navigator.sendBeacon for fire-and-forget reliability.
 */
import { devDebug } from '../utils/devLogger';

export function logInteraction(
  eventType: string,
  data: Record<string, unknown> = {},
): void {
  try {
    const payload = {
      eventType,
      ...data,
      ts: Date.now(),
    };
    navigator.sendBeacon(
      '/api/analytics/layout',
      new Blob([JSON.stringify(payload)], { type: 'application/json' }),
    );
  } catch (err) {
    // Silently fail - analytics should never break the UI
    devDebug('[analytics] beacon failed', err);
  }
}

export interface PublishValidationFlags {
  accessibilityOk: boolean;
  performanceOk: boolean;
  customData?: Record<string, unknown>;
}

export interface PublishValidationResponse {
  allowed: boolean;
  reasons?: string[];
  warnings?: string[];
}

/**
 * validateBeforePublish: Call backend governance gate before publishing.
 * Returns validation result or throws error with reasons.
 * 
 * Example:
 * ```tsx
 * const a11yCheck = checkDialogs();
 * const perfCheck = await checkPerformanceBudget();
 * 
 * try {
 *   await validateBeforePublish({
 *     accessibilityOk: a11yCheck.ok,
 *     performanceOk: perfCheck.ok,
 *   });
 *   await publishLayout();
 * } catch (err) {
 *   // Prefer useNotification in UI code instead of window.alert
 *   // const notification = useNotification();
 *   // notification.error(`Publish blocked: ${err.message}`);
 *   devWarn(`Publish blocked: ${err.message}`);
 * }
 * ```
 */
export async function validateBeforePublish(
  flags: PublishValidationFlags,
): Promise<PublishValidationResponse> {
  const res = await fetch('/api/publish/validate', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(flags),
  });

  const data: PublishValidationResponse = await res.json();

  if (!res.ok) {
    const reasons = data.reasons?.join('; ') || 'Validation failed';
    throw new Error(reasons);
  }

  return data;
}

/**
 * createAnalyticsContext: Helper to build consistent event payloads
 */
export function createAnalyticsContext(overrides?: Record<string, unknown>) {
  return {
    userAgent: navigator.userAgent,
    timestamp: new Date().toISOString(),
    ...overrides,
  };
}
