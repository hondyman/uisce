export type ExternalTask = {
    id: string;
    system: 'Salesforce' | 'ServiceNow' | 'Jira';
    action: 'create' | 'update' | 'close' | 'comment';
    externalId: string | null;
    status: 'created' | 'in_progress' | 'resolved' | 'failed';
    llmDecision?: {
        system: string;
        reason: string;
        metadata?: Record<string, any>;
    };
    createdAt: string;
    updatedAt: string;
};

export type ReportInfo = {
    reportId: string;
    reportUrl: string;
    generatedAt: string;
};
