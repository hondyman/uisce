#!/usr/bin/env node
import yargs from 'yargs';
import { hideBin } from 'yargs/helpers';
import * as ops from './ops';

const argv = yargs(hideBin(process.argv))
  .command('signal <workflowId>', 'Send a signal', (y: any) => y
    .positional('workflowId', { type: 'string' })
    .option('runId', { type: 'string' })
    .option('signal', { type: 'string', default: 'unblock' })
    .option('payload', { type: 'string' })
  , async (args: any) => {
    const payload = args.payload ? JSON.parse(args.payload) : undefined;
    const r = await ops.signal(args.workflowId, args.runId, args.signal, payload);
    console.log('OK', r);
  })
  .command('update <workflowId>', 'Send an update', (y: any) => y
    .positional('workflowId', { type: 'string' })
    .option('runId', { type: 'string' })
    .option('update', { type: 'string', default: 'changePriority' })
    .option('payload', { type: 'string' })
  , async (args: any) => {
    const payload = args.payload ? JSON.parse(args.payload) : undefined;
    const r = await ops.update(args.workflowId, args.runId, args.update, payload);
    console.log('OK', r);
  })
  .command('cancel <workflowId>', 'Cancel a workflow', (y: any) => y
    .positional('workflowId', { type: 'string' })
    .option('runId', { type: 'string' })
  , async (args: any) => {
    const r = await ops.cancel(args.workflowId, args.runId);
    console.log('OK', r);
  })
  .command('terminate <workflowId>', 'Terminate a workflow', (y: any) => y
    .positional('workflowId', { type: 'string' })
    .option('runId', { type: 'string' })
    .option('reason', { type: 'string', default: 'terminated by ops' })
  , async (args: any) => {
    const r = await ops.terminate(args.workflowId, args.runId, args.reason);
    console.log('OK', r);
  })
  .command('reset <workflowId>', 'Reset workflow to last workflow task', (y: any) => y
    .positional('workflowId', { type: 'string' })
    .option('runId', { type: 'string' })
    .option('eventId', { type: 'number' })
  , async (args: any) => {
    const r = await ops.resetToLastWorkflowTask(args.workflowId, args.runId, args.eventId);
    console.log('OK', r);
  })
  .command('stack <workflowId>', 'Get stack trace query', (y: any) => y
    .positional('workflowId', { type: 'string' })
    .option('runId', { type: 'string' })
  , async (args: any) => {
    const r = await ops.stackTrace(args.workflowId, args.runId);
    console.log(JSON.stringify(r, null, 2));
  })
  .command('describeTaskQueue <name>', 'Describe a task queue', (y: any) => y
    .positional('name', { type: 'string' })
    .option('type', { type: 'string', choices: ['workflow', 'activity'], default: 'workflow' })
  , async (args: any) => {
    const r = await ops.describeTaskQueue(args.name, args.type as any);
    console.log(JSON.stringify(r, null, 2));
  })
  .demandCommand(1)
  .strict()
  .help()
  .argv;

process.on('SIGINT', async () => {
  await ops.closeConnection();
  process.exit(0);
});
