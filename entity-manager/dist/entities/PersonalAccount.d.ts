import { Account, AssetType, ApprovalRoute, ComplianceRule, AccountStatus } from './Account.js';
export declare enum RiskTolerance {
    LOW = "low",
    MEDIUM = "medium",
    HIGH = "high",
    AGGRESSIVE = "aggressive"
}
export declare enum InvestmentObjective {
    INCOME = "income",
    GROWTH = "growth",
    BALANCED = "balanced",
    PRESERVATION = "preservation"
}
export declare class PersonalAccount extends Account {
    riskTolerance: RiskTolerance;
    investmentObjective: InvestmentObjective;
    netWorth?: number;
    constructor(id: string, tenantId: string, datasourceId: string, accountNumber: string, name: string, ownerId: string, custodianId: string, riskTolerance?: RiskTolerance, investmentObjective?: InvestmentObjective, status?: AccountStatus, netWorth?: number, createdAt?: Date, updatedAt?: Date);
    getMaxConcentration(): number;
    canHoldAsset(assetType: AssetType): boolean;
    getApprovalChain(amount: number): ApprovalRoute[];
    getComplianceRules(): ComplianceRule[];
    toJSON(): Record<string, any>;
}
//# sourceMappingURL=PersonalAccount.d.ts.map