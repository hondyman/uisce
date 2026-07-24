import { Client, Connection } from '@temporalio/client';
import { temporal } from '@temporalio/proto';

let _connection: Connection | null = null;
let _client: Client | null = null;

async function getConnectionClient() {
  if (_client && _connection) return { connection: _connection, client: _client };
  _connection = await Connection.connect();
  _client = new Client({ connection: _connection });
  return { connection: _connection, client: _client };
}

export async function signal(workflowId: string, runId?: string, signalName = 'unblock', payload?: unknown) {
  const { client } = await getConnectionClient();
  const h = client.workflow.getHandle(workflowId, runId);
  return await h.signal(signalName as any, payload);
}

export async function update(workflowId: string, runId?: string, updateName = 'changePriority', payload?: unknown) {
  const { client } = await getConnectionClient();
  const h = client.workflow.getHandle(workflowId, runId);
  // `update` is an experimental API on some client versions; cast to any to be tolerant of different SDK typings.
  return await (h as any).update(updateName as any, payload);
}

export async function cancel(workflowId: string, runId?: string) {
  const { client } = await getConnectionClient();
  const h = client.workflow.getHandle(workflowId, runId);
  return await h.cancel();
}

export async function terminate(workflowId: string, runId?: string, reason = 'terminated by ops') {
  const { client } = await getConnectionClient();
  const h = client.workflow.getHandle(workflowId, runId);
  return await h.terminate(reason);
}

export async function resetToLastWorkflowTask(workflowId: string, runId?: string, workflowTaskFinishEventId?: number) {
  const { connection, client } = await getConnectionClient();
  // Fetch history
  const historyResp = await connection.workflowService.getWorkflowExecutionHistory({
    namespace: client.options.namespace ?? 'default',
    execution: { workflowId, runId },
  });
  const events = historyResp.history?.events ?? [];
  // If caller passed an explicit event id, use it; otherwise find the last WorkflowTaskCompleted event id
  const lastWorkflowTaskCompleted = events.slice().reverse().find((e: any) => e.workflowTaskCompletedEventAttributes);
  const chosenEventId = workflowTaskFinishEventId ?? (lastWorkflowTaskCompleted as any)?.eventId ?? 3;

  const req = temporal.api.workflowservice.v1.ResetWorkflowExecutionRequest.create({
    namespace: client.options.namespace ?? 'default',
    workflowExecution: { workflowId, runId },
    resetReapplyType: temporal.api.enums.v1.ResetReapplyType.RESET_REAPPLY_TYPE_NONE,
    // typings for the protobuf Long may differ between environments; cast to any to allow numeric ids.
    workflowTaskFinishEventId: chosenEventId as any,
  });
  return await connection.workflowService.resetWorkflowExecution(req);
}

export async function stackTrace(workflowId: string, runId?: string) {
  const { client } = await getConnectionClient();
  const h = client.workflow.getHandle(workflowId, runId);
  return await h.query('__stack_trace' as any);
}

export async function describeTaskQueue(taskQueue: string, taskQueueType: 'workflow' | 'activity' = 'workflow') {
  const { connection, client } = await getConnectionClient();
  return await connection.workflowService.describeTaskQueue({
    namespace: client.options.namespace ?? 'default',
    taskQueue: { name: taskQueue },
    taskQueueType: taskQueueType === 'activity'
      ? temporal.api.enums.v1.TaskQueueType.TASK_QUEUE_TYPE_ACTIVITY
      : temporal.api.enums.v1.TaskQueueType.TASK_QUEUE_TYPE_WORKFLOW,
    reportPollers: true,
    reportStats: true,
  });
}

export async function describe(workflowId: string, runId?: string) {
  const { connection, client } = await getConnectionClient();
  return await connection.workflowService.describeWorkflowExecution({
    namespace: client.options.namespace ?? 'default',
    execution: { workflowId, runId },
  });
}

export async function childExecutions(workflowId: string, runId?: string) {
  const { connection, client } = await getConnectionClient();
  const res = await connection.workflowService.getWorkflowExecutionHistory({
    namespace: client.options.namespace ?? 'default',
    execution: { workflowId, runId },
  });
  const events = res.history?.events ?? [];
  return events
    .filter((e: any) =>
      e.startChildWorkflowExecutionInitiatedEventAttributes ||
      e.childWorkflowExecutionStartedEventAttributes)
    .map((e: any) => ({
      type: e.eventType,
      child: (e.childWorkflowExecutionStartedEventAttributes as any)?.workflowExecution,
      initiated: (e.startChildWorkflowExecutionInitiatedEventAttributes as any)?.workflowId,
    }));
}

export async function closeConnection() {
  if (_connection) {
    await _connection.close();
    _connection = null;
    _client = null;
  }
}
