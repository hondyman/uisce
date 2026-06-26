import { Account, AssetType, ApprovalRoute, ComplianceRule, AccountStatus } from './Account.js';
export declare enum IRAType {
    TRADITIONAL = "traditional",
    ROTH = "roth",
    SEP = "sep",
    SIMPLE = "simple",
    ROLLOVER = "rollover"
}
export declare class IRAAccount extends Account {
    iraType: IRAType;
    ownerAge: number;
    contributionLimit: number;
    constructor(id: string, tenantId: string, datasourceId: string, accountNumber: string, name: string, ownerId: string, custodianId: string, iraType: IRAType | undefined, ownerAge: number, contributionLimit: number, status?: AccountStatus, createdAt?: Date, updatedAt?: Date);
    getMaxConcentration(): number;
    canHoldAsset(assetType: AssetType): boolean;
    getApprovalChain(amount: number): ApprovalRoute[];
    getComplianceRules(): ComplianceRule[];
    canContribute(amount: number): boolean;
    isEligibleForWithdrawal(): boolean;
    toJSON(): Record<string, any>;
}
//# sourceMappingURL=IRAAccount.d.ts.map