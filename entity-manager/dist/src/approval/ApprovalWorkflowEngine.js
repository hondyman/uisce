import { getTemporalClient } from '../services/temporal.js';
import { publishEvent, getKafkaConsumer } from '../services/kafka.js';
import { EntityManager } from '../services/EntityManager.js';
import { logger } from '../utils/logger.js';
export var WorkflowStatus;
(function (WorkflowStatus) {
    WorkflowStatus["PENDING"] = "pending";
    WorkflowStatus["IN_PROGRESS"] = "in_progress";
    WorkflowStatus["APPROVED"] = "approved";
    WorkflowStatus["REJECTED"] = "rejected";
    WorkflowStatus["ESCALATED"] = "escalated";
    WorkflowStatus["TIMEOUT"] = "timeout";
    WorkflowStatus["CANCELLED"] = "cancelled";
})(WorkflowStatus || (WorkflowStatus = {}));
export class ApprovalWorkflowEngine {
    static instance;
    entityManager;
    lastEvent = null;
    listenersReady = false;
    constructor() {
        this.entityManager = EntityManager.getInstance();
    }
    static getInstance() {
        if (!ApprovalWorkflowEngine.instance) {
            ApprovalWorkflowEngine.instance = new ApprovalWorkflowEngine();
        }
        return ApprovalWorkflowEngine.instance;
    }
    async startApprovalWorkflow(request) {
        try {
            const account = await this.entityManager.loadEntity(request.accountId);
            if (!account) {
                throw new Error(`Account ${request.accountId} not found`);
            }
            const approvalChain = account.getApprovalChain(request.amount);
            if (approvalChain.length === 0) {
                return {
                    workflowId: '',
                    approvalChain: [],
                    status: WorkflowStatus.APPROVED
                };
            }
            const temporal = getTemporalClient();
            const workflowId = `approval-${request.id}`;
            const workflowHandle = await temporal.workflow.start('ApprovalWorkflow', {
                workflowId,
                args: [{
                        request,
                        approvalChain,
                        currentLevel: 0
                    }],
                taskQueue: 'approval-queue'
            });
            await this.publishWorkflowEvent('workflow.started', {
                workflowId,
                request,
                approvalChain
            });
            logger.info(`Started approval workflow: ${workflowId}`);
            return {
                workflowId,
                approvalChain,
                status: WorkflowStatus.IN_PROGRESS
            };
        }
        catch (error) {
            logger.error('Failed to start approval workflow:', error);
            throw error;
        }
    }
    async submitDecision(workflowId, decision) {
        try {
            const temporal = getTemporalClient();
            const workflowHandle = temporal.workflow.getHandle(workflowId);
            await workflowHandle.signal('approvalDecision', decision);
            await this.publishWorkflowEvent('decision.submitted', {
                workflowId,
                decision
            });
            logger.info(`Decision submitted for workflow: ${workflowId}`);
        }
        catch (error) {
            logger.error('Failed to submit decision:', error);
            throw error;
        }
    }
    async getWorkflowStatus(workflowId) {
        try {
            const temporal = getTemporalClient();
            const workflowHandle = temporal.workflow.getHandle(workflowId);
            const state = await workflowHandle.query('getState');
            return state;
        }
        catch (error) {
            logger.error('Failed to get workflow status:', error);
            throw error;
        }
    }
    async cancelWorkflow(workflowId, reason) {
        try {
            const temporal = getTemporalClient();
            const workflowHandle = temporal.workflow.getHandle(workflowId);
            await workflowHandle.cancel();
            await this.publishWorkflowEvent('workflow.cancelled', {
                workflowId,
                reason
            });
            logger.info(`Workflow cancelled: ${workflowId}`);
        }
        catch (error) {
            logger.error('Failed to cancel workflow:', error);
            throw error;
        }
    }
    async escalateWorkflow(workflowId, reason) {
        try {
            const temporal = getTemporalClient();
            const workflowHandle = temporal.workflow.getHandle(workflowId);
            await workflowHandle.signal('escalate', { reason });
            await this.publishWorkflowEvent('workflow.escalated', {
                workflowId,
                reason
            });
            logger.info(`Workflow escalated: ${workflowId}`);
        }
        catch (error) {
            logger.error('Failed to escalate workflow:', error);
            throw error;
        }
    }
    async getPendingApprovals(userId) {
        try {
            const pendingApprovals = [];
            return pendingApprovals;
        }
        catch (error) {
            logger.error('Failed to get pending approvals:', error);
            throw error;
        }
    }
    async publishWorkflowEvent(eventType, data) {
        try {
            await publishEvent('approval.events', {
                eventType,
                timestamp: new Date().toISOString(),
                data
            });
        }
        catch (error) {
            logger.warn('Failed to publish workflow event:', error);
        }
    }
    async setupEventListeners() {
        try {
            const consumer = getKafkaConsumer();
            await consumer.subscribe({ topic: 'approval.events', fromBeginning: false });
            await consumer.run({
                eachMessage: async ({ message }) => {
                    if (message && message.value) {
                        try {
                            const event = JSON.parse(message.value.toString());
                            await this.handleWorkflowEvent(event);
                        }
                        catch (error) {
                            logger.error('Failed to handle workflow event:', error);
                        }
                    }
                }
            });
            this.listenersReady = true;
            logger.info('Workflow event listeners setup complete');
        }
        catch (error) {
            this.listenersReady = false;
            logger.error('Failed to setup event listeners:', error);
            throw error;
        }
    }
    async handleWorkflowEvent(event) {
        this.lastEvent = event;
        const { eventType, data } = event;
        switch (eventType) {
            case 'workflow.completed':
                await this.handleWorkflowCompleted(data);
                break;
            case 'workflow.rejected':
                await this.handleWorkflowRejected(data);
                break;
            case 'workflow.escalated':
                await this.handleWorkflowEscalated(data);
                break;
            default:
                logger.debug(`Unhandled workflow event: ${eventType}`);
        }
    }
    getLastEvent() {
        return this.lastEvent;
    }
    async recordEvent(event) {
        this.lastEvent = event;
        try {
            await this.handleWorkflowEvent(event);
        }
        catch (e) {
            logger.warn('recordEvent: handler failed', e);
        }
    }
    isListenersReady() {
        return this.listenersReady;
    }
    async handleWorkflowCompleted(data) {
        const { workflowId, finalDecision } = data;
        logger.info(`Workflow completed: ${workflowId}, decision: ${finalDecision}`);
    }
    async handleWorkflowRejected(data) {
        const { workflowId, rejectionReason } = data;
        logger.info(`Workflow rejected: ${workflowId}, reason: ${rejectionReason}`);
    }
    async handleWorkflowEscalated(data) {
        const { workflowId, escalationReason } = data;
        logger.info(`Workflow escalated: ${workflowId}, reason: ${escalationReason}`);
    }
}
//# sourceMappingURL=ApprovalWorkflowEngine.js.map