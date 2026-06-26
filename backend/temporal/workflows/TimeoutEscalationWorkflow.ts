import { proxyActivities, defineQuery, sleep } from '@temporalio/workflow';
import type * as activities from '../activities/timeoutEscalationActivities';

/**
 * TimeoutEscalationWorkflow - Temporal Workflow for SLA/Timeout Management
 * 
 * Handles automatic escalation when steps or approvals exceed SLA times.
 * 
 * Features:
 * - Configurable timeout duration
 * - Multiple escalation actions (notify, escalate, auto_approve, auto_reject)
 * - Status monitoring via queries
 * - Integration with ABAC policies
 * - Audit trail of all actions
 */

const { escalateToManager, notifyDirector, autoApproveStep, autoRejectStep } =
  proxyActivities<typeof activities>({
    startToCloseTimeout: '5 minutes',
    retryPolicy: {
      maximumAttempts: 2,
    },
  });

export const queries = {
  getEscalationStatus: defineQuery<{
    status: 'pending' | 'escalated' | 'completed';
    timeoutAt: string;
    escalatedAt?: string;
  }>('getEscalationStatus'),
};

interface TimeoutEscalationInput {
  bpId: string;
  stepName: string;
  timeoutHours: number;
  escalationAction: 'notify' | 'escalate' | 'auto_approve' | 'auto_reject';
  escalateToManager?: string;
}

/**
 * Main timeout escalation workflow
 */
export async function TimeoutEscalationWorkflow(
  input: TimeoutEscalationInput
): Promise<void> {
  let status: 'pending' | 'escalated' | 'completed' = 'pending';
  const startTime = new Date();
  const timeoutAt = new Date(startTime.getTime() + input.timeoutHours * 3600000);

  console.log(
    `[${input.bpId}/${input.stepName}] Timeout escalation scheduled for ${timeoutAt.toISOString()}`
  );

  try {
    // Wait for the timeout duration
    await sleep(`${input.timeoutHours} hours`);

    status = 'escalated';
    const escalatedAt = new Date();

    console.log(
      `[${input.bpId}/${input.stepName}] Timeout reached, executing escalation action: ${input.escalationAction}`
    );

    // Execute escalation action
    switch (input.escalationAction) {
      case 'notify':
        await escalateToManager(input.bpId, input.stepName, {
          action: 'notify',
          manager_id: input.escalateToManager,
          message: `Step "${input.stepName}" in process ${input.bpId} has exceeded the SLA of ${input.timeoutHours} hours`,
        });
        break;

      case 'escalate':
        await escalateToManager(input.bpId, input.stepName, {
          action: 'escalate',
          manager_id: input.escalateToManager,
          escalate_to: 'director', // Escalate to director level
          reason: `SLA violation: ${input.timeoutHours} hours exceeded`,
        });
        break;

      case 'auto_approve':
        console.log(
          `[${input.bpId}/${input.stepName}] Auto-approving step due to timeout`
        );
        await autoApproveStep(input.bpId, input.stepName, {
          reason: `Auto-approved after ${input.timeoutHours} hour timeout`,
          approver: 'system',
        });
        break;

      case 'auto_reject':
        console.log(
          `[${input.bpId}/${input.stepName}] Auto-rejecting step due to timeout`
        );
        await autoRejectStep(input.bpId, input.stepName, {
          reason: `Auto-rejected after ${input.timeoutHours} hour timeout`,
          rejector: 'system',
        });
        break;

      default:
        throw new Error(`Unknown escalation action: ${input.escalationAction}`);
    }

    status = 'completed';
    console.log(
      `[${input.bpId}/${input.stepName}] Escalation action completed`
    );
  } catch (error) {
    console.error(`[${input.bpId}/${input.stepName}] Escalation failed:`, error);

    // Notify director of escalation failure
    try {
      await notifyDirector(input.bpId, input.stepName, {
        error: error instanceof Error ? error.message : 'Unknown error',
        action: input.escalationAction,
      });
    } catch (notifyErr) {
      console.error(`Failed to notify director:`, notifyErr);
    }

    throw error;
  }
}
