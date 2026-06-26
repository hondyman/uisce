import { getTemporalClient } from '../services/temporal.js';
import { publishEvent, getKafkaConsumer } from '../services/kafka.js';
import { EntityManager } from '../services/EntityManager.js';
import { Account, ApprovalRoute } from '../entities/index.js';
import { logger } from '../utils/logger.js';

/**
 * Approval request data
 */
export interface ApprovalRequest {
  id: string;
  accountId: string;
  tradeId: string;
  amount: number;
  description: string;
  requesterId: string;
  tenantId: string;
  datasourceId: string;
  createdAt: Date;
}

/**
 * Approval decision
 */
export interface ApprovalDecision {
  approvalId: string;
  approverId: string;
  decision: 'approved' | 'rejected' | 'escalated';
  comments?: string;
  timestamp: Date;
}

/**
 * Workflow status
 */
export enum WorkflowStatus {
  PENDING = 'pending',
  IN_PROGRESS = 'in_progress',
  APPROVED = 'approved',
  REJECTED = 'rejected',
  ESCALATED = 'escalated',
  TIMEOUT = 'timeout',
  CANCELLED = 'cancelled'
}

/**
 * Approval Workflow Engine - Handles Temporal workflows and messaging
 */
export class ApprovalWorkflowEngine {
  private static instance: ApprovalWorkflowEngine;
  private entityManager: EntityManager;
  // For testing: store last processed event so integration smoke tests can verify event flow
  private lastEvent: any | null = null;
  // Indicates whether workflow event listeners are fully started and stable
  private listenersReady = false;

  private constructor() {
    this.entityManager = EntityManager.getInstance();
  }

  static getInstance(): ApprovalWorkflowEngine {
    if (!ApprovalWorkflowEngine.instance) {
      ApprovalWorkflowEngine.instance = new ApprovalWorkflowEngine();
    }
    return ApprovalWorkflowEngine.instance;
  }

  /**
   * Start approval workflow for a trade
   */
  async startApprovalWorkflow(request: ApprovalRequest): Promise<{
    workflowId: string;
    approvalChain: ApprovalRoute[];
    status: WorkflowStatus;
  }> {
    try {
      // Load account to get approval chain
      const account = await this.entityManager.loadEntity(request.accountId) as Account;
      if (!account) {
        throw new Error(`Account ${request.accountId} not found`);
      }

      // Get approval chain for the amount
      const approvalChain = account.getApprovalChain(request.amount);

      if (approvalChain.length === 0) {
        // No approval required
        return {
          workflowId: '',
          approvalChain: [],
          status: WorkflowStatus.APPROVED
        };
      }

      // Start Temporal workflow
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

      // Publish to RabbitMQ for real-time updates
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

    } catch (error) {
      logger.error('Failed to start approval workflow:', error);
      throw error;
    }
  }

  /**
   * Submit approval decision
   */
  async submitDecision(
    workflowId: string,
    decision: ApprovalDecision
  ): Promise<void> {
    try {
      const temporal = getTemporalClient();

      // Signal the workflow with the decision
      const workflowHandle = temporal.workflow.getHandle(workflowId);
      await workflowHandle.signal('approvalDecision', decision);

      // Publish decision event
      await this.publishWorkflowEvent('decision.submitted', {
        workflowId,
        decision
      });

      logger.info(`Decision submitted for workflow: ${workflowId}`);

    } catch (error) {
      logger.error('Failed to submit decision:', error);
      throw error;
    }
  }

  /**
   * Get workflow status
   */
  async getWorkflowStatus(workflowId: string): Promise<{
    status: WorkflowStatus;
    currentLevel: number;
    decisions: ApprovalDecision[];
    approvalChain: ApprovalRoute[];
  }> {
    try {
      const temporal = getTemporalClient();
      const workflowHandle = temporal.workflow.getHandle(workflowId);

      // Query workflow for current state
      const state: {
        status: WorkflowStatus;
        currentLevel: number;
        decisions: ApprovalDecision[];
        approvalChain: ApprovalRoute[];
      } = await workflowHandle.query('getState');

      return state;

    } catch (error) {
      logger.error('Failed to get workflow status:', error);
      throw error;
    }
  }

  /**
   * Cancel workflow
   */
  async cancelWorkflow(workflowId: string, reason: string): Promise<void> {
    try {
      const temporal = getTemporalClient();
      const workflowHandle = temporal.workflow.getHandle(workflowId);

      await workflowHandle.cancel();

      // Publish cancellation event
      await this.publishWorkflowEvent('workflow.cancelled', {
        workflowId,
        reason
      });

      logger.info(`Workflow cancelled: ${workflowId}`);

    } catch (error) {
      logger.error('Failed to cancel workflow:', error);
      throw error;
    }
  }

  /**
   * Escalate workflow to next level
   */
  async escalateWorkflow(workflowId: string, reason: string): Promise<void> {
    try {
      const temporal = getTemporalClient();
      const workflowHandle = temporal.workflow.getHandle(workflowId);

      await workflowHandle.signal('escalate', { reason });

      // Publish escalation event
      await this.publishWorkflowEvent('workflow.escalated', {
        workflowId,
        reason
      });

      logger.info(`Workflow escalated: ${workflowId}`);

    } catch (error) {
      logger.error('Failed to escalate workflow:', error);
      throw error;
    }
  }

  /**
   * Get pending approvals for user
   */
  async getPendingApprovals(userId: string): Promise<ApprovalRequest[]> {
    try {
      // Query database for pending approvals assigned to user
      // This would typically query a workflow state table
      const pendingApprovals: ApprovalRequest[] = [];

      // For now, return empty array
      return pendingApprovals;

    } catch (error) {
      logger.error('Failed to get pending approvals:', error);
      throw error;
    }
  }

  /**
   * Publish workflow event (Kafka)
   */
  private async publishWorkflowEvent(eventType: string, data: any): Promise<void> {
    try {
      await publishEvent('approval.events', {
        eventType,
        timestamp: new Date().toISOString(),
        data
      });
    } catch (error) {
      logger.warn('Failed to publish workflow event:', error);
      // Don't throw - event publishing shouldn't break the workflow
    }
  }

  /**
   * Setup workflow event listeners (Kafka consumer)
   */
  async setupEventListeners(): Promise<void> {
    try {
      const consumer = getKafkaConsumer();

      await consumer.subscribe({ topic: 'approval.events', fromBeginning: false });

      await consumer.run({
        eachMessage: async ({ message }) => {
          if (message && message.value) {
            try {
              const event = JSON.parse(message.value.toString());
              await this.handleWorkflowEvent(event);
            } catch (error) {
              logger.error('Failed to handle workflow event:', error);
            }
          }
        }
      });

      // Mark listeners as ready once consumer.run() is active
      this.listenersReady = true;
      logger.info('Workflow event listeners setup complete');

    } catch (error) {
      // Ensure flag is false if setup fails
      this.listenersReady = false;
      logger.error('Failed to setup event listeners:', error);
      throw error;
    }
  }

  /**
   * Handle workflow events
   */
  private async handleWorkflowEvent(event: any): Promise<void> {
    // Record last event for smoke tests
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

  // Expose last processed event for testing
  getLastEvent(): any | null {
    return this.lastEvent;
  }

  // For test fallback: record and optionally process the event directly when Kafka is unavailable
  async recordEvent(event: any): Promise<void> {
    this.lastEvent = event;
    try {
      await this.handleWorkflowEvent(event);
    } catch (e) {
      logger.warn('recordEvent: handler failed', e);
    }
  }

  isListenersReady(): boolean {
    return this.listenersReady;
  }

  /**
   * Handle workflow completed
   */
  private async handleWorkflowCompleted(data: any): Promise<void> {
    const { workflowId, finalDecision } = data;

    logger.info(`Workflow completed: ${workflowId}, decision: ${finalDecision}`);

    // Here you would:
    // 1. Update trade status
    // 2. Execute the trade if approved
    // 3. Send notifications
    // 4. Update audit trail
  }

  /**
   * Handle workflow rejected
   */
  private async handleWorkflowRejected(data: any): Promise<void> {
    const { workflowId, rejectionReason } = data;

    logger.info(`Workflow rejected: ${workflowId}, reason: ${rejectionReason}`);

    // Here you would:
    // 1. Update trade status to rejected
    // 2. Send notifications to requester
    // 3. Update audit trail
  }

  /**
   * Handle workflow escalated
   */
  private async handleWorkflowEscalated(data: any): Promise<void> {
    const { workflowId, escalationReason } = data;

    logger.info(`Workflow escalated: ${workflowId}, reason: ${escalationReason}`);

    // Here you would:
    // 1. Notify escalated approvers
    // 2. Update workflow state
    // 3. Send notifications
  }
}