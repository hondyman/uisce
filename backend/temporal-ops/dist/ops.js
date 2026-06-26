"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.closeConnection = exports.childExecutions = exports.describe = exports.describeTaskQueue = exports.stackTrace = exports.resetToLastWorkflowTask = exports.terminate = exports.cancel = exports.update = exports.signal = void 0;
const client_1 = require("@temporalio/client");
const proto_1 = require("@temporalio/proto");
let _connection = null;
let _client = null;
async function getConnectionClient() {
    if (_client && _connection)
        return { connection: _connection, client: _client };
    _connection = await client_1.Connection.connect();
    _client = new client_1.Client({ connection: _connection });
    return { connection: _connection, client: _client };
}
async function signal(workflowId, runId, signalName = 'unblock', payload) {
    const { client } = await getConnectionClient();
    const h = client.workflow.getHandle(workflowId, runId);
    return await h.signal(signalName, payload);
}
exports.signal = signal;
async function update(workflowId, runId, updateName = 'changePriority', payload) {
    const { client } = await getConnectionClient();
    const h = client.workflow.getHandle(workflowId, runId);
    // `update` is an experimental API on some client versions; cast to any to be tolerant of different SDK typings.
    return await h.update(updateName, payload);
}
exports.update = update;
async function cancel(workflowId, runId) {
    const { client } = await getConnectionClient();
    const h = client.workflow.getHandle(workflowId, runId);
    return await h.cancel();
}
exports.cancel = cancel;
async function terminate(workflowId, runId, reason = 'terminated by ops') {
    const { client } = await getConnectionClient();
    const h = client.workflow.getHandle(workflowId, runId);
    return await h.terminate(reason);
}
exports.terminate = terminate;
async function resetToLastWorkflowTask(workflowId, runId, workflowTaskFinishEventId) {
    const { connection, client } = await getConnectionClient();
    // Fetch history
    const historyResp = await connection.workflowService.getWorkflowExecutionHistory({
        namespace: client.options.namespace ?? 'default',
        execution: { workflowId, runId },
    });
    const events = historyResp.history?.events ?? [];
    // If caller passed an explicit event id, use it; otherwise find the last WorkflowTaskCompleted event id
    const lastWorkflowTaskCompleted = events.slice().reverse().find((e) => e.workflowTaskCompletedEventAttributes);
    const chosenEventId = workflowTaskFinishEventId ?? lastWorkflowTaskCompleted?.eventId ?? 3;
    const req = proto_1.temporal.api.workflowservice.v1.ResetWorkflowExecutionRequest.create({
        namespace: client.options.namespace ?? 'default',
        workflowExecution: { workflowId, runId },
        resetReapplyType: proto_1.temporal.api.enums.v1.ResetReapplyType.RESET_REAPPLY_TYPE_NONE,
        // typings for the protobuf Long may differ between environments; cast to any to allow numeric ids.
        workflowTaskFinishEventId: chosenEventId,
    });
    return await connection.workflowService.resetWorkflowExecution(req);
}
exports.resetToLastWorkflowTask = resetToLastWorkflowTask;
async function stackTrace(workflowId, runId) {
    const { client } = await getConnectionClient();
    const h = client.workflow.getHandle(workflowId, runId);
    return await h.query('__stack_trace');
}
exports.stackTrace = stackTrace;
async function describeTaskQueue(taskQueue, taskQueueType = 'workflow') {
    const { connection, client } = await getConnectionClient();
    return await connection.workflowService.describeTaskQueue({
        namespace: client.options.namespace ?? 'default',
        taskQueue: { name: taskQueue },
        taskQueueType: taskQueueType === 'activity'
            ? proto_1.temporal.api.enums.v1.TaskQueueType.TASK_QUEUE_TYPE_ACTIVITY
            : proto_1.temporal.api.enums.v1.TaskQueueType.TASK_QUEUE_TYPE_WORKFLOW,
        reportPollers: true,
        reportStats: true,
    });
}
exports.describeTaskQueue = describeTaskQueue;
async function describe(workflowId, runId) {
    const { connection, client } = await getConnectionClient();
    return await connection.workflowService.describeWorkflowExecution({
        namespace: client.options.namespace ?? 'default',
        execution: { workflowId, runId },
    });
}
exports.describe = describe;
async function childExecutions(workflowId, runId) {
    const { connection, client } = await getConnectionClient();
    const res = await connection.workflowService.getWorkflowExecutionHistory({
        namespace: client.options.namespace ?? 'default',
        execution: { workflowId, runId },
    });
    const events = res.history?.events ?? [];
    return events
        .filter((e) => e.startChildWorkflowExecutionInitiatedEventAttributes ||
        e.childWorkflowExecutionStartedEventAttributes)
        .map((e) => ({
        type: e.eventType,
        child: e.childWorkflowExecutionStartedEventAttributes?.workflowExecution,
        initiated: e.startChildWorkflowExecutionInitiatedEventAttributes?.workflowId,
    }));
}
exports.childExecutions = childExecutions;
async function closeConnection() {
    if (_connection) {
        await _connection.close();
        _connection = null;
        _client = null;
    }
}
exports.closeConnection = closeConnection;
