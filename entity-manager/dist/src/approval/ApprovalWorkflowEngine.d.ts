import { ApprovalRoute } from '../entities/index.js';
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
export interface ApprovalDecision {
    approvalId: string;
    approverId: string;
    decision: 'approved' | 'rejected' | 'escalated';
    comments?: string;
    timestamp: Date;
}
export declare enum WorkflowStatus {
    PENDING = "pending",
    IN_PROGRESS = "in_progress",
    APPROVED = "approved",
    REJECTED = "rejected",
    ESCALATED = "escalated",
    TIMEOUT = "timeout",
    CANCELLED = "cancelled"
}
export declare class ApprovalWorkflowEngine {
    private static instance;
    private entityManager;
    private lastEvent;
    private listenersReady;
    private constructor();
    static getInstance(): ApprovalWorkflowEngine;
    startApprovalWorkflow(request: ApprovalRequest): Promise<{
        workflowId: string;
        approvalChain: ApprovalRoute[];
        status: WorkflowStatus;
    }>;
    submitDecision(workflowId: string, decision: ApprovalDecision): Promise<void>;
    getWorkflowStatus(workflowId: string): Promise<{
        status: WorkflowStatus;
        currentLevel: number;
        decisions: ApprovalDecision[];
        approvalChain: ApprovalRoute[];
    }>;
    cancelWorkflow(workflowId: string, reason: string): Promise<void>;
    escalateWorkflow(workflowId: string, reason: string): Promise<void>;
    getPendingApprovals(userId: string): Promise<ApprovalRequest[]>;
    private publishWorkflowEvent;
    setupEventListeners(): Promise<void>;
    private handleWorkflowEvent;
    getLastEvent(): any | null;
    recordEvent(event: any): Promise<void>;
    isListenersReady(): boolean;
    private handleWorkflowCompleted;
    private handleWorkflowRejected;
    private handleWorkflowEscalated;
}
//# sourceMappingURL=ApprovalWorkflowEngine.d.ts.map