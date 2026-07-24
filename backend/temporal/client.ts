import { Connection, Client } from '@temporalio/client';
import { getEnv } from '../../internal/pkg/env/getEnv';

/**
 * Temporal Client
 * Used to start workflows and send signals from the API layer
 */

let client: Client | null = null;

/**
 * Initialize Temporal client connection
 */
export async function initializeTemporalClient(): Promise<Client> {
  if (client) {
    return client;
  }

  const connection = await Connection.connect({
    address: getEnv('TEMPORAL_SERVER_ADDRESS', 'VITE_TEMPORAL_SERVER_ADDRESS', 'localhost:7233'),
  });

  client = new Client({
    connection,
    namespace: getEnv('TEMPORAL_NAMESPACE', 'VITE_TEMPORAL_NAMESPACE', 'default'),
  });

  console.log('Temporal client initialized');
  return client;
}

/**
 * Get the Temporal client instance
 */
export async function getTemporalClient(): Promise<Client> {
  if (!client) {
    return initializeTemporalClient();
  }
  return client;
}

/**
 * Start client onboarding workflow
 */
export async function startClientOnboardingWorkflow(
  clientId: string,
  clientData: {
    name: string;
    email: string;
    managerId: string;
    amlProvider?: string;
  }
) {
  const temporalClient = await getTemporalClient();

  const workflowId = `client-onboarding-${clientId}-${Date.now()}`;

  const handle = await temporalClient.workflow.start(
    'ClientOnboardingWorkflow',
    {
      args: [
        {
          clientId,
          clientName: clientData.name,
          clientEmail: clientData.email,
          managerId: clientData.managerId,
          amlProvider: clientData.amlProvider || 'default',
        },
      ],
      taskQueue: getEnv('TEMPORAL_TASK_QUEUE', 'VITE_TEMPORAL_TASK_QUEUE', 'default'),
      workflowId,
    }
  );

  console.log(
    `Started ClientOnboardingWorkflow ${workflowId} for client ${clientId}`
  );

  return {
    workflowId,
    handle,
  };
}

/**
 * Signal a client onboarding workflow for approval
 */
export async function approveClientOnboarding(
  workflowId: string,
  managerId: string
) {
  const temporalClient = await getTemporalClient();
  const handle = temporalClient.workflow.getHandle(workflowId);

  await handle.signal('approve', managerId);

  console.log(`Sent approval signal to workflow ${workflowId}`);
}

/**
 * Signal a client onboarding workflow for rejection
 */
export async function rejectClientOnboarding(
  workflowId: string,
  managerId: string,
  reason: string
) {
  const temporalClient = await getTemporalClient();
  const handle = temporalClient.workflow.getHandle(workflowId);

  await handle.signal('reject', { managerId, reason });

  console.log(`Sent rejection signal to workflow ${workflowId}`);
}

/**
 * Query client onboarding workflow status
 */
export async function getClientOnboardingStatus(workflowId: string) {
  const temporalClient = await getTemporalClient();
  const handle = temporalClient.workflow.getHandle(workflowId);

  const status = await handle.query('getStatus');
  const step = await handle.query('getStep');
  const approvalDetails = await handle.query('getApprovalDetails');

  return {
    status,
    step,
    approvalDetails,
  };
}

/**
 * Start timeout escalation workflow
 */
export async function startTimeoutEscalationWorkflow(
  businessProcessId: string,
  stepName: string,
  escalationData: {
    timeoutHours: number;
    escalationAction: 'notify' | 'escalate' | 'auto_approve' | 'auto_reject';
    managerId?: string;
    reason?: string;
  }
) {
  const temporalClient = await getTemporalClient();

  const workflowId = `timeout-escalation-${businessProcessId}-${stepName}-${Date.now()}`;

  const handle = await temporalClient.workflow.start(
    'TimeoutEscalationWorkflow',
    {
      args: [
        {
          businessProcessId,
          stepName,
          timeoutHours: escalationData.timeoutHours,
          escalationAction: escalationData.escalationAction,
          managerId: escalationData.managerId,
          reason: escalationData.reason || 'SLA timeout',
        },
      ],
      taskQueue: getEnv('TEMPORAL_TASK_QUEUE', 'VITE_TEMPORAL_TASK_QUEUE', 'default'),
      workflowId,
      executionTimeout: {
        seconds: escalationData.timeoutHours * 3600 + 300, // Timeout + 5 min buffer
      },
    }
  );

  console.log(
    `Started TimeoutEscalationWorkflow ${workflowId} for BP ${businessProcessId} step ${stepName}`
  );

  return {
    workflowId,
    handle,
  };
}

/**
 * Query timeout escalation workflow status
 */
export async function getTimeoutEscalationStatus(workflowId: string) {
  const temporalClient = await getTemporalClient();
  const handle = temporalClient.workflow.getHandle(workflowId);

  const status = await handle.query('getEscalationStatus');

  return status;
}

/**
 * Get workflow execution history
 */
export async function getWorkflowHistory(workflowId: string) {
  const temporalClient = await getTemporalClient();
  const handle = temporalClient.workflow.getHandle(workflowId);

  const execution = await handle.describe();

  return {
    status: execution.status,
    startTime: execution.startTime,
    closeTime: execution.closeTime,
    closeStatus: execution.closeStatus,
    closeReason: execution.closeReason,
  };
}

/**
 * Terminate workflow with reason
 */
export async function terminateWorkflow(
  workflowId: string,
  reason: string
): Promise<void> {
  const temporalClient = await getTemporalClient();
  const handle = temporalClient.workflow.getHandle(workflowId);

  await handle.terminate(reason);

  console.log(`Terminated workflow ${workflowId}: ${reason}`);
}

/**
 * Close the Temporal client connection
 */
export async function closeTemporalClient(): Promise<void> {
  if (client) {
    await client.connection.close();
    client = null;
    console.log('Temporal client connection closed');
  }
}
