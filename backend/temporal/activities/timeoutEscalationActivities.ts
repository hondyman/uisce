/**
 * Timeout Escalation Activities
 * 
 * Handle SLA violations and automated escalation actions.
 * Each activity is idempotent and integrates with the BP Designer trigger system.
 */

/**
 * Escalate to manager with specified action
 */
export async function escalateToManager(
  bpId: string,
  stepName: string,
  options: {
    action: 'notify' | 'escalate';
    escalate_to?: string;
    managerId?: string;
    reason?: string;
  }
): Promise<void> {
  console.log(
    `[ACTIVITY] Escalating BP ${bpId} step ${stepName} with action: ${options.action}`
  );

  const response = await fetch(`/api/business-processes/${bpId}/escalate`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({
      step_name: stepName,
      action: options.action,
      escalate_to: options.escalate_to,
      manager_id: options.managerId,
      reason: options.reason || 'SLA timeout',
    }),
  });

  if (!response.ok) {
    throw new Error(`Escalation to manager failed: ${response.statusText}`);
  }

  console.log(
    `[ACTIVITY] Manager escalation complete for BP ${bpId} step ${stepName}`
  );
}

/**
 * Notify director of escalation
 */
export async function notifyDirector(
  bpId: string,
  stepName: string,
  options: {
    severity: string;
    reason: string;
    clientName?: string;
  }
): Promise<void> {
  console.log(
    `[ACTIVITY] Notifying director of escalation for BP ${bpId} step ${stepName}`
  );

  const response = await fetch(`/api/escalations/director`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({
      business_process_id: bpId,
      step_name: stepName,
      severity: options.severity,
      reason: options.reason,
      client_name: options.clientName,
    }),
  });

  if (!response.ok) {
    throw new Error(`Director notification failed: ${response.statusText}`);
  }

  console.log(`[ACTIVITY] Director notified for BP ${bpId}`);
}

/**
 * Auto-approve a step after timeout
 */
export async function autoApproveStep(
  bpId: string,
  stepName: string,
  options: { reason: string }
): Promise<void> {
  console.log(
    `[ACTIVITY] Auto-approving step ${stepName} for BP ${bpId}...`
  );

  const response = await fetch(
    `/api/business-processes/${bpId}/steps/${stepName}/approve`,
    {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        auto_approved: true,
        reason: options.reason,
        approved_by: 'system',
      }),
    }
  );

  if (!response.ok) {
    throw new Error(`Auto-approval failed: ${response.statusText}`);
  }

  console.log(`[ACTIVITY] Step ${stepName} auto-approved for BP ${bpId}`);
}

/**
 * Auto-reject a step after timeout
 */
export async function autoRejectStep(
  bpId: string,
  stepName: string,
  options: { reason: string }
): Promise<void> {
  console.log(
    `[ACTIVITY] Auto-rejecting step ${stepName} for BP ${bpId}...`
  );

  const response = await fetch(
    `/api/business-processes/${bpId}/steps/${stepName}/reject`,
    {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        auto_rejected: true,
        reason: options.reason,
        rejected_by: 'system',
      }),
    }
  );

  if (!response.ok) {
    throw new Error(`Auto-rejection failed: ${response.statusText}`);
  }

  console.log(`[ACTIVITY] Step ${stepName} auto-rejected for BP ${bpId}`);
}

/**
 * Log escalation event to audit trail
 */
export async function logEscalationEvent(
  bpId: string,
  stepName: string,
  eventData: {
    action: string;
    escalationPath: string;
    timestamp: number;
    reason: string;
  }
): Promise<void> {
  console.log(
    `[ACTIVITY] Logging escalation event for BP ${bpId} step ${stepName}`
  );

  const response = await fetch(`/api/audit-logs`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({
      business_process_id: bpId,
      step_name: stepName,
      event_type: 'escalation',
      action: eventData.action,
      escalation_path: eventData.escalationPath,
      timestamp: eventData.timestamp,
      reason: eventData.reason,
    }),
  });

  if (!response.ok) {
    throw new Error(`Audit logging failed: ${response.statusText}`);
  }

  console.log(
    `[ACTIVITY] Escalation event logged for BP ${bpId} step ${stepName}`
  );
}
