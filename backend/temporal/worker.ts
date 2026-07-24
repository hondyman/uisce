import { Worker } from '@temporalio/worker';
import * as activities from './activities';
import * as workflows from './workflows';
import { getEnv } from '../../internal/pkg/env/getEnv';

/**
 * Temporal Worker
 * Registers all workflows and activities for execution
 */

export async function createWorker() {
  const worker = await Worker.create({
    // Server connection
    connection: {
      address: getEnv('TEMPORAL_SERVER_ADDRESS', 'VITE_TEMPORAL_SERVER_ADDRESS', 'localhost:7233'),
    },

    // Task queue - workflows go here for processing
    taskQueue: getEnv('TEMPORAL_TASK_QUEUE', 'VITE_TEMPORAL_TASK_QUEUE', 'default'),

    // Register all workflows
    workflowsPath: require.resolve('./workflows'),

    // Register all activities
    activities: {
      // Client onboarding activities
      validateClient: activities.validateClient,
      performAMLScreening: activities.performAMLScreening,
      routeForApproval: activities.routeForApproval,
      generateAgreements: activities.generateAgreements,
      createAccounts: activities.createAccounts,
      notifyClient: activities.notifyClient,
      escalateToDirector: activities.escalateToDirector,

      // Timeout escalation activities
      escalateToManager: activities.escalateToManager,
      notifyDirector: activities.notifyDirector,
      autoApproveStep: activities.autoApproveStep,
      autoRejectStep: activities.autoRejectStep,
      logEscalationEvent: activities.logEscalationEvent,
    },

    // Activity execution settings
    activityMaxAttempts: 3,
    activityRetryPolicy: {
      initialInterval: 1000, // 1 second
      backoffCoefficient: 2,
      maximumInterval: 30000, // 30 seconds
      maximumAttempts: 3,
    },

    // Worker settings
    maxConcurrentActivityTaskExecutors: 10,
    maxConcurrentWorkflowTaskExecutors: 5,
  });

  console.log('Temporal Worker created and configured');
  return worker;
}

/**
 * Start the worker and keep it running
 */
export async function startWorker() {
  const worker = await createWorker();

  // Run indefinitely until signaled to stop
  await worker.run();
}

/**
 * Start worker with graceful shutdown
 */
export async function runWorker() {
  const worker = await createWorker();

  // Handle shutdown signals
  process.on('SIGINT', async () => {
    console.log('Shutting down Temporal Worker...');
    await worker.shutdown();
    process.exit(0);
  });

  process.on('SIGTERM', async () => {
    console.log('Temporal Worker termination requested...');
    await worker.shutdown();
    process.exit(0);
  });

  // Start worker
  console.log('Starting Temporal Worker...');
  await worker.run();
}

// Export for use in other modules
export { Worker };
