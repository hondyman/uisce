/**
 * Exception Remediation Activities
 *
 * Backs ExceptionRemediationWorkflow. Each activity is idempotent and talks
 * to the platform-intelligence exceptions API (see
 * backend/internal/handlers/platform_intelligence_handler.go and
 * backend/internal/platform_intelligence/exceptions/aggregator.go).
 *
 * Modeled directly on timeoutEscalationActivities.ts's shape: small,
 * fetch-based activities, no shared engine.
 */

export interface RemediationException {
  id: string;
  tenantId: string;
  type: string;
  severity: string;
  source: string;
  description: string;
}

/**
 * attemptAutoFix runs a type-specific remediation action for the exception.
 * The fix itself is intentionally thin per-type dispatch (e.g. re-trigger a
 * stale pre-agg refresh) rather than a generic remediation engine.
 */
export async function attemptAutoFix(
  exception: RemediationException
): Promise<{ success: boolean; detail: string }> {
  console.log(
    `[ACTIVITY] Attempting auto-fix for exception ${exception.id} (${exception.type})`
  );

  let endpoint: string | null = null;
  let body: Record<string, unknown> = { exception_id: exception.id };

  switch (exception.type) {
    case 'preagg_inconsistency':
      // Re-trigger the pre-agg refresh workflow for the affected target.
      endpoint = '/api/aso/preagg/refresh';
      body = { ...body, target_source: exception.source };
      break;
    case 'semantic_drift':
      // Force a fresh semantic snapshot so downstream consumers reconcile.
      endpoint = '/api/semantic/snapshot';
      body = { ...body, target_source: exception.source };
      break;
    case 'slo_breach':
      // Re-run the scheduler job the SLO breach was reported against.
      endpoint = '/api/scheduler/jobs/rerun';
      body = { ...body, target_source: exception.source };
      break;
    default:
      return {
        success: false,
        detail: `No auto-fix action registered for exception type "${exception.type}"`,
      };
  }

  try {
    const response = await fetch(endpoint, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(body),
    });
    if (!response.ok) {
      return { success: false, detail: `Auto-fix request failed: ${response.statusText}` };
    }
    return { success: true, detail: `Auto-fix action dispatched to ${endpoint}` };
  } catch (err) {
    return {
      success: false,
      detail: `Auto-fix action threw: ${err instanceof Error ? err.message : String(err)}`,
    };
  }
}

/**
 * verifyResolved re-runs the *originating detector's check* scoped to this
 * exception instance. This is what makes "a re-run doesn't generate
 * duplicate errors, closes if fixed" true: verification must actually
 * re-check reality, not just trust the fix attempt succeeded.
 */
export async function verifyResolved(
  exception: RemediationException
): Promise<{ verified: boolean; detail: string }> {
  console.log(
    `[ACTIVITY] Verifying resolution for exception ${exception.id} (${exception.type})`
  );

  let endpoint: string | null = null;
  switch (exception.type) {
    case 'preagg_inconsistency':
      endpoint = '/api/aso/preagg/verify';
      break;
    case 'semantic_drift':
      endpoint = '/api/semantic/verify-drift';
      break;
    case 'slo_breach':
      endpoint = '/api/scheduler/jobs/verify-health';
      break;
    default:
      return {
        verified: false,
        detail: `No verify check registered for exception type "${exception.type}"`,
      };
  }

  try {
    const response = await fetch(
      `${endpoint}?source=${encodeURIComponent(exception.source)}`
    );
    if (!response.ok) {
      return { verified: false, detail: `Verify check failed: ${response.statusText}` };
    }
    const result = (await response.json()) as { resolved?: boolean };
    return {
      verified: Boolean(result.resolved),
      detail: result.resolved ? 'Verify check confirmed resolution' : 'Verify check still failing',
    };
  } catch (err) {
    return {
      verified: false,
      detail: `Verify check threw: ${err instanceof Error ? err.message : String(err)}`,
    };
  }
}

/**
 * recordAttempt appends an autofix attempt to the exception record via
 * POST /exceptions/{id}/rerun-equivalent write path (AppendAutofixAttempt).
 */
export async function recordAttempt(
  exception: RemediationException,
  action: string,
  success: boolean,
  verified: boolean,
  detail: string
): Promise<void> {
  await fetch(`/api/platform-intelligence/exceptions/${exception.id}/rerun`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ action, success, verified, detail }),
  });
}

/**
 * closeExceptionAsFixed marks the exception auto_fixed/closed after
 * verifyResolved has actually confirmed the fix — never before.
 */
export async function closeExceptionAsFixed(
  exception: RemediationException
): Promise<void> {
  await fetch(`/api/platform-intelligence/exceptions/${exception.id}/close`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ status: 'auto_fixed', closed_by_ai: true }),
  });
}

/**
 * notifyHumans falls back to the existing enterprise notification service
 * when auto-fix is disabled for this exception type, or auto-fix attempts
 * did not verify after the retry policy is exhausted.
 */
export async function notifyHumans(
  exception: RemediationException,
  reason: string
): Promise<void> {
  console.log(
    `[ACTIVITY] Notifying humans about exception ${exception.id}: ${reason}`
  );
  await fetch('/api/notifications/send-from-template', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({
      template_code: 'EXCEPTION_NEEDS_ATTENTION',
      tenant_id: exception.tenantId,
      data: {
        exception_id: exception.id,
        type: exception.type,
        severity: exception.severity,
        source: exception.source,
        description: exception.description,
        reason,
      },
    }),
  });
}
