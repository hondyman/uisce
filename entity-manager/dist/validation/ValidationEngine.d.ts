import { AssetType, ComplianceRule } from '../entities/index.js';
export interface ValidationContext {
    accountId: string;
    trade?: {
        ticker: string;
        quantity: number;
        price: number;
        amount: number;
        assetType: AssetType;
    };
    portfolio?: {
        totalValue: number;
        cash: number;
        positions: Array<{
            ticker: string;
            quantity: number;
            value: number;
            percentage: number;
        }>;
    };
    userId: string;
    tenantId: string;
    datasourceId: string;
}
export interface ValidationRuleResult {
    ruleId: string;
    ruleName: string;
    passed: boolean;
    severity: 'low' | 'medium' | 'high' | 'critical';
    message: string;
    details?: any;
}
export interface ValidationResult {
    isValid: boolean;
    passedRules: ValidationRuleResult[];
    failedRules: ValidationRuleResult[];
    warnings: ValidationRuleResult[];
    errors: ValidationRuleResult[];
    executionTime: number;
}
export declare class ValidationEngine {
    private static instance;
    private entityManager;
    private constructor();
    static getInstance(): ValidationEngine;
    validateTradeRequest(context: ValidationContext): Promise<ValidationResult>;
    private validateAccountStatus;
    private validateConcentrationLimit;
    private validateAssetCompatibility;
    private validateKYCRequirements;
    private validateTradeExecution;
    private validateFeeStructure;
    private createResult;
    getAccountComplianceRules(accountId: string): Promise<ComplianceRule[]>;
}
//# sourceMappingURL=ValidationEngine.d.ts.map