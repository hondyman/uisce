export interface EscalationStep {
    id: string;
    stepNumber: number;        // 1st escalation, 2nd, etc.
    delayAfterPreviousExpr: string; // Starlark: def delay_seconds(ctx):
    targetActorRole: string;   // who gets escalated to
    notificationTemplate?: string;
    condition?: any;           // when to escalate (optional)
}

export interface ApprovalNodeData {
    label: string;
    stepKey: string;
    approvalChain: {
        // Using loose type for now to match partials, but ideally strongly typed
        rules: any[];
        fallbackRole?: string;
    };
    escalations?: EscalationStep[]; // cascade of escalations
    slaExpr?: string;               // overall SLA before final escalation
    activityName?: string;
    signalName?: string;
    conditionExpr?: string;
    delayExpr?: string;
    routingRules?: any;
}
