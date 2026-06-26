import { Context } from '@temporalio/activity';

/**
 * Client Onboarding Activities
 * 
 * These are the individual tasks that get executed as part of the workflow.
 * Each activity is idempotent and can be safely retried.
 */

/**
 * Validate client information against business rules
 */
export async function validateClient(
  clientId: string,
  clientName: string,
  email: string
): Promise<void> {
  console.log(`[ACTIVITY] Validating client ${clientId}...`);

  // Call backend validation endpoint
  const response = await fetch(`/api/clients/${clientId}/validate`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ name: clientName, email }),
  });

  if (!response.ok) {
    throw new Error(`Validation failed: ${response.statusText}`);
  }

  const result = await response.json();
  if (!result.valid) {
    throw new Error(`Validation error: ${result.reason}`);
  }

  console.log(`[ACTIVITY] Client ${clientId} validated`);
}

/**
 * Perform AML (Anti-Money Laundering) screening
 */
export async function performAMLScreening(
  clientId: string,
  amlProvider: string = 'default'
): Promise<{ passed: boolean; reason?: string }> {
  console.log(`[ACTIVITY] Performing AML screening for client ${clientId}...`);

  const response = await fetch(`/api/aml/screen`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ client_id: clientId, provider: amlProvider }),
  });

  if (!response.ok) {
    throw new Error(`AML screening failed: ${response.statusText}`);
  }

  const result = await response.json();
  console.log(
    `[ACTIVITY] AML screening ${result.passed ? 'passed' : 'failed'} for ${clientId}`
  );

  return result;
}

/**
 * Route the approval request to the appropriate manager
 */
export async function routeForApproval(
  clientId: string,
  managerId: string
): Promise<void> {
  console.log(`[ACTIVITY] Routing client ${clientId} for approval...`);

  const response = await fetch(`/api/approvals`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({
      client_id: clientId,
      type: 'onboarding',
      manager_id: managerId,
    }),
  });

  if (!response.ok) {
    throw new Error(`Routing failed: ${response.statusText}`);
  }

  console.log(`[ACTIVITY] Approval routed for client ${clientId}`);
}

/**
 * Generate legal agreements and documents
 */
export async function generateAgreements(
  clientId: string,
  options: { template: string; sendTo: string }
): Promise<void> {
  console.log(`[ACTIVITY] Generating agreements for client ${clientId}...`);

  const response = await fetch(`/api/documents/generate`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({
      client_id: clientId,
      template: options.template,
      send_to: options.sendTo,
    }),
  });

  if (!response.ok) {
    throw new Error(`Document generation failed: ${response.statusText}`);
  }

  console.log(`[ACTIVITY] Agreements generated for client ${clientId}`);
}

/**
 * Create accounts for the newly onboarded client
 */
export async function createAccounts(
  clientId: string,
  options: { name: string; email: string }
): Promise<void> {
  console.log(`[ACTIVITY] Creating accounts for client ${clientId}...`);

  const response = await fetch(`/api/accounts`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({
      client_id: clientId,
      name: options.name,
      email: options.email,
    }),
  });

  if (!response.ok) {
    throw new Error(`Account creation failed: ${response.statusText}`);
  }

  console.log(`[ACTIVITY] Accounts created for client ${clientId}`);
}

/**
 * Send notification to client
 */
export async function notifyClient(
  clientId: string,
  options: { type: string; recipient: string; subject: string }
): Promise<void> {
  console.log(
    `[ACTIVITY] Sending ${options.type} notification to ${options.recipient}...`
  );

  const response = await fetch(`/api/notifications`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({
      client_id: clientId,
      type: options.type,
      recipient: options.recipient,
      subject: options.subject,
    }),
  });

  if (!response.ok) {
    throw new Error(`Notification failed: ${response.statusText}`);
  }

  console.log(`[ACTIVITY] Notification sent to ${options.recipient}`);
}

/**
 * Escalate issue to director
 */
export async function escalateToDirector(
  clientId: string,
  options: { error: string; step: string; clientName: string }
): Promise<void> {
  console.log(
    `[ACTIVITY] Escalating client ${clientId} issue to director...`
  );

  const response = await fetch(`/api/escalations`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({
      client_id: clientId,
      error: options.error,
      step: options.step,
      client_name: options.clientName,
      severity: 'high',
    }),
  });

  if (!response.ok) {
    throw new Error(`Escalation failed: ${response.statusText}`);
  }

  console.log(`[ACTIVITY] Issue escalated to director for client ${clientId}`);
}
