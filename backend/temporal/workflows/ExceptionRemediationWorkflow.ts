import { proxyActivities, defineQuery, sleep } from '@temporalio/workflow';
import type * as activities from '../activities/exceptionRemediationActivities';

/**
 * ExceptionRemediationWorkflow - Temporal workflow for the platform
 * exception hub's auto-fix path.
 *
 * Modeled directly on TimeoutEscalationWorkflow.ts: query for status,
 * sleep-based pacing between attempts, retryPolicy on activities.
 *
 * Flow: Publish() starts this workflow when the tenant/type autofix policy
 * is enabled (see exception_autofix_policy table + ExceptionAggregator.
 * ResolveAutofixPolicy). If policy is disabled, the caller skips straight
 * to notifyHumans instead of starting this workflow at all.
 *
 *   attemptAutoFix(exception)
 *     -> verifyResolved(exception)   // re-runs the ORIGINATING detector's
 *                                     // check, scoped to this instance —
 *                                     // this is what prevents a re-run
 *                                     // from creating a duplicate exception
 *     -> if verified: closeExceptionAsFixed (auto_fixed/closed)
 *     -> if not verified: retry up to maxAttempts, then notifyHumans and
 *        leave the exception open for a human
 */

const { attemptAutoFix, verifyResolved, recordAttempt, closeExceptionAsFixed, notifyHumans } =
  proxyActivities<typeof activities>({
    startToCloseTimeout: '5 minutes',
    retryPolicy: {
      maximumAttempts: 2,
    },
  });

export const queries = {
  getRemediationStatus: defineQuery<{
    status: 'attempting' | 'verifying' | 'fixed' | 'escalated';
    attempt: number;
  }>('getRemediationStatus'),
};

export interface ExceptionRemediationInput {
  exception: activities.RemediationException;
  /** Max auto-fix attempts before falling back to human notification. */
  maxAttempts?: number;
  /** Delay between attempts, Temporal duration string (e.g. "2 minutes"). */
  retryDelay?: string;
}

let currentStatus: {
  status: 'attempting' | 'verifying' | 'fixed' | 'escalated';
  attempt: number;
} = { status: 'attempting', attempt: 0 };

export async function ExceptionRemediationWorkflow(
  input: ExceptionRemediationInput
): Promise<{ resolved: boolean }> {
  const maxAttempts = input.maxAttempts ?? 3;
  const retryDelay = input.retryDelay ?? '2 minutes';
  const { exception } = input;

  for (let attempt = 1; attempt <= maxAttempts; attempt++) {
    currentStatus = { status: 'attempting', attempt };
    console.log(
      `[${exception.id}] Auto-fix attempt ${attempt}/${maxAttempts} for ${exception.type}`
    );

    const fixResult = await attemptAutoFix(exception);
    await recordAttempt(exception, 'autofix_attempted', fixResult.success, false, fixResult.detail);

    currentStatus = { status: 'verifying', attempt };

    // Verify-before-close: never mark an exception fixed on the strength of
    // the fix attempt alone. Only a passing re-run of the originating
    // detector's check closes it.
    const verifyResult = await verifyResolved(exception);
    await recordAttempt(exception, 'verify_checked', fixResult.success, verifyResult.verified, verifyResult.detail);

    if (verifyResult.verified) {
      currentStatus = { status: 'fixed', attempt };
      await closeExceptionAsFixed(exception);
      console.log(`[${exception.id}] Verified fixed after attempt ${attempt}, closed.`);
      return { resolved: true };
    }

    if (attempt < maxAttempts) {
      await sleep(retryDelay);
    }
  }

  // Auto-fix exhausted its attempts without a verified fix: fall back to
  // notifying a human and leave the exception open.
  currentStatus = { status: 'escalated', attempt: maxAttempts };
  await notifyHumans(
    exception,
    `Auto-fix did not verify after ${maxAttempts} attempts`
  );
  console.log(`[${exception.id}] Auto-fix exhausted, escalated to human notification.`);
  return { resolved: false };
}
