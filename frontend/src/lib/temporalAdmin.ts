import { Connection } from '@temporalio/client';
import proto from '@temporalio/proto';

export async function describeTaskQueue(namespace: string, taskQueueName: string, type: 'workflow' | 'activity' = 'workflow') {
  const connection = await Connection.connect();
  const taskQueueType = type === 'activity'
    ? proto.temporal.api.enums.v1.TaskQueueType.TASK_QUEUE_TYPE_ACTIVITY
    : proto.temporal.api.enums.v1.TaskQueueType.TASK_QUEUE_TYPE_WORKFLOW;

  const res = await connection.workflowService.describeTaskQueue({
    namespace,
    taskQueue: { name: taskQueueName },
    taskQueueType,
    reportPollers: true,
    reportStats: true,
  });

  return res;
}
