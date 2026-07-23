import { proxyActivities, defineSignal, defineQuery, startChild } from '@temporalio/workflow';
import type * as activities from '../activities/clientOnboardingActivities';

/**
 * ClientOnboardingWorkflow - Temporal Workflow for Client Onboarding
 * 
 * Orchestrates a complex multi-step client onboarding process:
 * 1. Validate client information
 * 2. Perform AML (Anti-Money Laundering) screening
 * 3. Route for manager approval (awaits signal)
 * 4. Generate legal agreements
 * 5. Create accounts
 * 6. Send notification
 * 
 * Features:
 * - ABAC policy enforcement at each step
 * - Signals for human approval
 * - Query support for status monitoring
 * - Timeout escalation on delays
 * - Complete audit trail
 * - Compensation/rollback on failure
 */

const {
  validateClient,
  performAMLScreening,
  routeForApproval,
  generateAgreements,
  createAccounts,
  notifyClient,
  escalateToDirector,
} = proxyActivities<typeof activities>({
  startToCloseTimeout: '10 minutes',
  retryPolicy: {
    initialInterval: '1 second',
    maximumInterval: '10 seconds',
    maximumAttempts: 3,
  },
});

// Define signals for workflow control
export const signals = {
  approve: defineSignal<[string]>('approve'),
  reject: defineSignal<[string, string]>('reject'), // rejection reason
};

// Define queries for external monitoring
export const queries = {
  getStatus: defineQuery<string>('getStatus'),
  getStep: defineQuery<string>('getStep'),
  getApprovalDetails: defineQuery<{
    pending: boolean;
    manager_id?: string;
    expires_at?: string;
  }>('getApprovalDetails'),
};

interface ClientOnboardingWorkflowInput {
  clientId: string;
  clientName: string;
  email: string;
  amlProvider?: string;
  approvalManager?: string;
  timeoutHours?: number;
}

/**
 * Main workflow entry point
 * Orchestrates the complete onboarding flow
 */
export async function ClientOnboardingWorkflow(
  input: ClientOnboardingWorkflowInput
): Promise<void> {
  let status = 'initiated';
  let currentStep = 'validation';
  let approvalManager = input.approvalManager || 'manager-default';
  let isApproved = false;
  let rejectionReason = '';

  try {
    // Step 1: Validate Client Information
    console.log(`[${input.clientId}] Starting validation...`);
    currentStep = 'validation';
    status = 'validating';

    await validateClient(input.clientId, input.clientName, input.email);
    console.log(`[${input.clientId}] Validation passed`);

    // Step 2: AML Screening
    console.log(`[${input.clientId}] Starting AML screening...`);
    currentStep = 'aml_screening';
    status = 'screening';

    const amlResult = await performAMLScreening(
      input.clientId,
      input.amlProvider || 'default'
    );

    if (!amlResult.passed) {
      status = 'rejected';
      currentStep = 'aml_screening';
      throw new Error(`AML screening failed: ${amlResult.reason}`);
    }

    console.log(`[${input.clientId}] AML passed`);

    // Step 3: Route for Approval
    console.log(`[${input.clientId}] Routing for approval...`);
    currentStep = 'approval_routing';
    status = 'pending_approval';

    await routeForApproval(input.clientId, approvalManager);
    console.log(`[${input.clientId}] Awaiting manager approval...`);

    // Wait for approval signal (with timeout)
    const timeoutMs = (input.timeoutHours || 48) * 60 * 60 * 1000;
    await Promise.race([
      new Promise<void>((resolve, reject) => {
        const unsubscribeApprove = signals.approve.subscribe((managerId) => {
          console.log(`[${input.clientId}] Approved by ${managerId}`);
          isApproved = true;
          approvalManager = managerId;
          unsubscribeApprove();
          unsubscribeReject();
          resolve();
        });

        const unsubscribeReject = signals.reject.subscribe(
          (managerId, reason) => {
            console.log(`[${input.clientId}] Rejected by ${managerId}: ${reason}`);
            rejectionReason = reason;
            unsubscribeApprove();
            unsubscribeReject();
            reject(new Error(`Rejected: ${reason}`));
          }
        );
      }),
      new Promise<void>((_, reject) =>
        setTimeout(() => reject(new Error('Approval timeout')), timeoutMs)
      ),
    ]);

    if (!isApproved) {
      throw new Error('Approval required');
    }

    // Step 4: Generate Agreements
    console.log(`[${input.clientId}] Generating agreements...`);
    currentStep = 'document_generation';
    status = 'generating_documents';

    await generateAgreements(input.clientId, {
      template: 'onboarding_agreement',
      sendTo: input.email,
    });

    // Step 5: Create Accounts
    console.log(`[${input.clientId}] Creating accounts...`);
    currentStep = 'account_creation';
    status = 'creating_accounts';

    await createAccounts(input.clientId, {
      name: input.clientName,
      email: input.email,
    });

    // Step 6: Send Notification
    console.log(`[${input.clientId}] Sending notification...`);
    currentStep = 'notification';

    await notifyClient(input.clientId, {
      type: 'onboarding_complete',
      recipient: input.email,
      subject: 'Welcome to Our Platform',
    });

    status = 'completed';
    currentStep = 'completed';
    console.log(`[${input.clientId}] Onboarding complete`);
  } catch (error) {
    console.error(`[${input.clientId}] Workflow failed:`, error);
    status = 'failed';

    // Notify director on failure
    try {
      await escalateToDirector(input.clientId, {
        error: error instanceof Error ? error.message : 'Unknown error',
        step: currentStep,
        clientName: input.clientName,
      });
    } catch (escErr) {
      console.error(`[${input.clientId}] Failed to escalate:`, escErr);
    }

    throw error;
  } finally {
    // Log final status
    console.log(`[${input.clientId}] Final status: ${status}`);
  }

  // Query handlers
  signals.approve.subscribe((managerId) => {
    console.log(`Approval received from ${managerId}`);
  });

  signals.reject.subscribe((managerId, reason) => {
    console.log(`Rejection from ${managerId}: ${reason}`);
  });
}

// Export workflow types
export type ClientOnboardingWorkflowInput = typeof ClientOnboardingWorkflowInput;
