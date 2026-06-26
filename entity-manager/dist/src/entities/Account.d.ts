import { Entity, ValidationResult } from './Entity.js';
export declare enum AssetType {
    EQUITY = "equity",
    BOND = "bond",
    MUTUAL_FUND = "mutual_fund",
    ETF = "etf",
    CRYPTO = "crypto",
    REAL_ESTATE = "real_estate",
    PRIVATE_EQUITY = "private_equity",
    HEDGE_FUND = "hedge_fund",
    DERIVATIVE = "derivative",
    SHORT_SELLING = "short_selling",
    ALTERNATIVE = "alternative"
}
export declare enum AccountStatus {
    PENDING = "pending",
    ACTIVE = "active",
    SUSPENDED = "suspended",
    CLOSED = "closed",
    FROZEN = "frozen"
}
export interface ApprovalRoute {
    level: number;
    approvers: string[];
    threshold: number;
    requiredCount: number;
    timeoutMinutes: number;
}
export interface ComplianceRule {
    id: string;
    name: string;
    description: string;
    category: string;
    severity: 'low' | 'medium' | 'high' | 'critical';
}
export declare abstract class Account extends Entity {
    readonly accountNumber: string;
    readonly name: string;
    readonly ownerId: string;
    readonly custodianId: string;
    status: AccountStatus;
    readonly accountType: string;
    constructor(id: string, tenantId: string, datasourceId: string, accountNumber: string, name: string, ownerId: string, custodianId: string, accountType: string, status?: AccountStatus, createdAt?: Date, updatedAt?: Date);
    getEntityType(): string;
    abstract getMaxConcentration(): number;
    abstract canHoldAsset(assetType: AssetType): boolean;
    abstract getApprovalChain(amount: number): ApprovalRoute[];
    abstract getComplianceRules(): ComplianceRule[];
    validate(): Promise<ValidationResult>;
    isActive(): boolean;
    toJSON(): Record<string, any>;
}
//# sourceMappingURL=Account.d.ts.map