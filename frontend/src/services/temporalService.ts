import apiClient from '../utils/apiClient';

/* Centralized Temporal-related API service
 * Re-implemented to use centralized apiClient
 */

async function safeJson(res: Response) {
  const text = await res.text()
  try {
    return JSON.parse(text)
  } catch (err) {
    return text
  }
}

export async function listExecutions(limit?: number) {
  const q = limit ? `?limit=${encodeURIComponent(String(limit))}` : ''
  const res = await apiClient(`temporal/executions${q}`)
  if (!res.ok) {
    const body = await safeJson(res)
    throw new Error(`listExecutions failed: ${res.status} ${String(body)}`)
  }
  return res.json()
}

export async function fetchAMQPMetrics() {
  const res = await apiClient(`_debug/amqp-metrics`)
  if (!res.ok) {
    const text = await res.text()
    throw new Error(`AMQP metrics failed: ${res.status} ${text}`)
  }
  return res.json()
}

export async function fetchTriggerEvents() {
  const res = await apiClient(`v1/triggers/events`)
  if (!res.ok) return []
  return res.json()
}

export async function publishTestEvent(payload: any) {
  const res = await apiClient(`_debug/publish-event`, {
    method: 'POST',
    body: JSON.stringify(payload),
  })
  const body = await safeJson(res)
  return { status: res.status, body }
}

export async function saveWorkflow(workflow: any) {
  const res = await apiClient(`temporal/workflows`, {
    method: 'POST',
    body: JSON.stringify(workflow),
  })
  const body = await safeJson(res)
  if (!res.ok) {
    throw new Error(`saveWorkflow failed: ${res.status} ${String(body)}`)
  }
  return body
}

// Placeholder admin actions for workflows (signal/terminate/cancel)
export async function signalWorkflow(workflowId: string, signalName: string, data: any) {
  const res = await apiClient(`temporal/workflows/${encodeURIComponent(workflowId)}/signal`, {
    method: 'POST',
    body: JSON.stringify({ signal: signalName, data }),
  })
  if (!res.ok) {
    const body = await safeJson(res)
    throw new Error(`signalWorkflow failed: ${res.status} ${String(body)}`)
  }
  return res.json()
}

export async function terminateWorkflow(workflowId: string, reason?: string) {
  const res = await apiClient(`temporal/workflows/${encodeURIComponent(workflowId)}/terminate`, {
    method: 'POST',
    body: JSON.stringify({ reason }),
  })
  if (!res.ok) {
    const body = await safeJson(res)
    throw new Error(`terminateWorkflow failed: ${res.status} ${String(body)}`)
  }
  return res.json()
}

export default {
  listExecutions,
  fetchAMQPMetrics,
  fetchTriggerEvents,
  publishTestEvent,
  signalWorkflow,
  terminateWorkflow,
  saveWorkflow,
}
