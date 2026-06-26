#!/usr/bin/env node
"use strict";
var __createBinding = (this && this.__createBinding) || (Object.create ? (function(o, m, k, k2) {
    if (k2 === undefined) k2 = k;
    var desc = Object.getOwnPropertyDescriptor(m, k);
    if (!desc || ("get" in desc ? !m.__esModule : desc.writable || desc.configurable)) {
      desc = { enumerable: true, get: function() { return m[k]; } };
    }
    Object.defineProperty(o, k2, desc);
}) : (function(o, m, k, k2) {
    if (k2 === undefined) k2 = k;
    o[k2] = m[k];
}));
var __setModuleDefault = (this && this.__setModuleDefault) || (Object.create ? (function(o, v) {
    Object.defineProperty(o, "default", { enumerable: true, value: v });
}) : function(o, v) {
    o["default"] = v;
});
var __importStar = (this && this.__importStar) || function (mod) {
    if (mod && mod.__esModule) return mod;
    var result = {};
    if (mod != null) for (var k in mod) if (k !== "default" && Object.prototype.hasOwnProperty.call(mod, k)) __createBinding(result, mod, k);
    __setModuleDefault(result, mod);
    return result;
};
var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
Object.defineProperty(exports, "__esModule", { value: true });
const yargs_1 = __importDefault(require("yargs"));
const helpers_1 = require("yargs/helpers");
const ops = __importStar(require("./ops"));
const argv = (0, yargs_1.default)((0, helpers_1.hideBin)(process.argv))
    .command('signal <workflowId>', 'Send a signal', (y) => y
    .positional('workflowId', { type: 'string' })
    .option('runId', { type: 'string' })
    .option('signal', { type: 'string', default: 'unblock' })
    .option('payload', { type: 'string' }), async (args) => {
    const payload = args.payload ? JSON.parse(args.payload) : undefined;
    const r = await ops.signal(args.workflowId, args.runId, args.signal, payload);
    console.log('OK', r);
})
    .command('update <workflowId>', 'Send an update', (y) => y
    .positional('workflowId', { type: 'string' })
    .option('runId', { type: 'string' })
    .option('update', { type: 'string', default: 'changePriority' })
    .option('payload', { type: 'string' }), async (args) => {
    const payload = args.payload ? JSON.parse(args.payload) : undefined;
    const r = await ops.update(args.workflowId, args.runId, args.update, payload);
    console.log('OK', r);
})
    .command('cancel <workflowId>', 'Cancel a workflow', (y) => y
    .positional('workflowId', { type: 'string' })
    .option('runId', { type: 'string' }), async (args) => {
    const r = await ops.cancel(args.workflowId, args.runId);
    console.log('OK', r);
})
    .command('terminate <workflowId>', 'Terminate a workflow', (y) => y
    .positional('workflowId', { type: 'string' })
    .option('runId', { type: 'string' })
    .option('reason', { type: 'string', default: 'terminated by ops' }), async (args) => {
    const r = await ops.terminate(args.workflowId, args.runId, args.reason);
    console.log('OK', r);
})
    .command('reset <workflowId>', 'Reset workflow to last workflow task', (y) => y
    .positional('workflowId', { type: 'string' })
    .option('runId', { type: 'string' })
    .option('eventId', { type: 'number' }), async (args) => {
    const r = await ops.resetToLastWorkflowTask(args.workflowId, args.runId, args.eventId);
    console.log('OK', r);
})
    .command('stack <workflowId>', 'Get stack trace query', (y) => y
    .positional('workflowId', { type: 'string' })
    .option('runId', { type: 'string' }), async (args) => {
    const r = await ops.stackTrace(args.workflowId, args.runId);
    console.log(JSON.stringify(r, null, 2));
})
    .command('describeTaskQueue <name>', 'Describe a task queue', (y) => y
    .positional('name', { type: 'string' })
    .option('type', { type: 'string', choices: ['workflow', 'activity'], default: 'workflow' }), async (args) => {
    const r = await ops.describeTaskQueue(args.name, args.type);
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
