import { Account, AssetType, ApprovalRoute, ComplianceRule, AccountStatus } from './Account.js';
export declare enum TrustType {
    REVOCABLE = "revocable",
    IRREVOCABLE = "irrevocable",
    CHARITABLE = "charitable",
    GRANTOR = "grantor"
}
export interface Beneficiary {
    id: string;
    name: string;
    relationship: string;
    percentage: number;
    ssn?: string;
}
export declare class TrustAccount extends Account {
    trustType: TrustType;
    trusteeId: string;
    beneficiaries: Beneficiary[];
    constructor(id: string, tenantId: string, datasourceId: string, accountNumber: string, name: string, ownerId: string, custodianId: string, trustType: TrustType | undefined, trusteeId: string, beneficiaries?: Beneficiary[], status?: AccountStatus, createdAt?: Date, updatedAt?: Date);
    getMaxConcentration(): number;
    canHoldAsset(assetType: AssetType): boolean;
    getApprovalChain(amount: number): ApprovalRoute[];
    getComplianceRules(): ComplianceRule[];
    isRevocable(): boolean;
    addBeneficiary(beneficiary: Beneficiary): void;
    removeBeneficiary(beneficiaryId: string): void;
    getTotalBeneficiaryAllocation(): number;
    toJSON(): Record<string, any>;
}
//# sourceMappingURL=TrustAccount.d.ts.map