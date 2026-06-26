import { Account } from '../entities/index.js';
export interface TradeExecutionRequest {
    accountId: string;
    trade: {
        ticker: string;
        quantity: number;
        price: number;
        assetType: string;
        amount: number;
    };
    portfolio: {
        totalValue: number;
        cash: number;
        positions: Array<{
            ticker: string;
            quantity: number;
            value: number;
            percentage: number;
        }>;
    };
    advisorId: string;
    tenantId: string;
    datasourceId: string;
}
export interface TradeExecutionResponse {
    success: boolean;
    workflowId?: string;
    approvalChain?: Array<{
        level: number;
        approvers: string[];
        threshold: number;
        requiredCount: number;
        timeoutMinutes: number;
    }>;
    complianceRules?: Array<{
        id: string;
        name: string;
        description: string;
        category: string;
        severity: string;
    }>;
    validationResults?: {
        isValid: boolean;
        passedRules: Array<{
            ruleId: string;
            ruleName: string;
            severity: string;
            message: string;
        }>;
        failedRules: Array<{
            ruleId: string;
            ruleName: string;
            severity: string;
            message: string;
        }>;
        warnings: Array<{
            ruleId: string;
            ruleName: string;
            severity: string;
            message: string;
        }>;
        errors: Array<{
            ruleId: string;
            ruleName: string;
            severity: string;
            message: string;
        }>;
    };
    error?: string;
}
export declare class UnifiedValidator {
    private static instance;
    private entityManager;
    private validationEngine;
    private approvalEngine;
    private constructor();
    static getInstance(): UnifiedValidator;
    processTradeRequest(request: TradeExecutionRequest): Promise<TradeExecutionResponse>;
    private performAccountSpecificValidation;
    validateAccount(account: Account): Promise<{
        isValid: boolean;
        errors: string[];
        warnings: string[];
    }>;
    getAccountApprovalChain(accountId: string, amount: number): Promise<Array<{
        level: number;
        approvers: string[];
        threshold: number;
        requiredCount: number;
        timeoutMinutes: number;
    }>>;
    getAccountComplianceRules(accountId: string): Promise<Array<{
        id: string;
        name: string;
        description: string;
        category: string;
        severity: string;
    }>>;
}
//# sourceMappingURL=UnifiedValidator.d.ts.map