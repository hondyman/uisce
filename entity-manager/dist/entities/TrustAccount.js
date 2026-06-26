import { Account, AssetType, AccountStatus } from './Account.js';
export var TrustType;
(function (TrustType) {
    TrustType["REVOCABLE"] = "revocable";
    TrustType["IRREVOCABLE"] = "irrevocable";
    TrustType["CHARITABLE"] = "charitable";
    TrustType["GRANTOR"] = "grantor";
})(TrustType || (TrustType = {}));
export class TrustAccount extends Account {
    trustType;
    trusteeId;
    beneficiaries;
    constructor(id, tenantId, datasourceId, accountNumber, name, ownerId, custodianId, trustType = TrustType.REVOCABLE, trusteeId, beneficiaries = [], status = AccountStatus.PENDING, createdAt, updatedAt) {
        super(id, tenantId, datasourceId, accountNumber, name, ownerId, custodianId, 'trust', status, createdAt, updatedAt);
        this.trustType = trustType;
        this.trusteeId = trusteeId;
        this.beneficiaries = beneficiaries;
    }
    getMaxConcentration() {
        return 0.15;
    }
    canHoldAsset(assetType) {
        const prohibitedAssets = new Set([
            AssetType.CRYPTO,
            AssetType.DERIVATIVE,
            AssetType.SHORT_SELLING,
            AssetType.HEDGE_FUND,
            AssetType.PRIVATE_EQUITY,
            AssetType.ALTERNATIVE
        ]);
        if (this.trustType === TrustType.CHARITABLE) {
            prohibitedAssets.add(AssetType.REAL_ESTATE);
        }
        return !prohibitedAssets.has(assetType);
    }
    getApprovalChain(amount) {
        return [
            {
                level: 1,
                approvers: ['advisor'],
                threshold: Number.MAX_SAFE_INTEGER,
                requiredCount: 1,
                timeoutMinutes: 30
            },
            {
                level: 2,
                approvers: ['compliance_officer'],
                threshold: Number.MAX_SAFE_INTEGER,
                requiredCount: 1,
                timeoutMinutes: 120
            },
            {
                level: 3,
                approvers: ['cfo', 'trustee'],
                threshold: Number.MAX_SAFE_INTEGER,
                requiredCount: 1,
                timeoutMinutes: 480
            }
        ];
    }
    getComplianceRules() {
        const rules = [
            {
                id: 'concentration_limit',
                name: 'Concentration Limit',
                description: `Maximum ${this.getMaxConcentration() * 100}% in any single position`,
                category: 'risk',
                severity: 'high'
            },
            {
                id: 'kyc_completeness',
                name: 'KYC Completeness',
                description: 'Client KYC must be current and complete',
                category: 'compliance',
                severity: 'critical'
            },
            {
                id: 'trade_execution',
                name: 'Trade Execution',
                description: 'Trades must be executed through approved channels',
                category: 'operational',
                severity: 'high'
            },
            {
                id: 'fee_validation',
                name: 'Fee Validation',
                description: 'Transaction fees must be within approved limits',
                category: 'financial',
                severity: 'medium'
            },
            {
                id: 'fiduciary_duty',
                name: 'Fiduciary Duty',
                description: 'Trustee must act in the best interest of beneficiaries',
                category: 'regulatory',
                severity: 'critical'
            },
            {
                id: 'beneficiary_protection',
                name: 'Beneficiary Protection',
                description: 'All actions must protect beneficiary interests',
                category: 'regulatory',
                severity: 'critical'
            },
            {
                id: 'trust_documentation',
                name: 'Trust Documentation',
                description: 'All trust documents must be current and on file',
                category: 'legal',
                severity: 'critical'
            }
        ];
        if (this.trustType === TrustType.IRREVOCABLE) {
            rules.push({
                id: 'irrevocable_restrictions',
                name: 'Irrevocable Restrictions',
                description: 'Trust cannot be modified without beneficiary consent',
                category: 'legal',
                severity: 'critical'
            });
        }
        return rules;
    }
    isRevocable() {
        return this.trustType === TrustType.REVOCABLE || this.trustType === TrustType.GRANTOR;
    }
    addBeneficiary(beneficiary) {
        const totalPercentage = this.beneficiaries.reduce((sum, b) => sum + b.percentage, 0) + beneficiary.percentage;
        if (totalPercentage > 100) {
            throw new Error('Total beneficiary percentages cannot exceed 100%');
        }
        this.beneficiaries.push(beneficiary);
        this.markAsUpdated();
    }
    removeBeneficiary(beneficiaryId) {
        const index = this.beneficiaries.findIndex(b => b.id === beneficiaryId);
        if (index === -1) {
            throw new Error('Beneficiary not found');
        }
        this.beneficiaries.splice(index, 1);
        this.markAsUpdated();
    }
    getTotalBeneficiaryAllocation() {
        return this.beneficiaries.reduce((sum, b) => sum + b.percentage, 0);
    }
    toJSON() {
        return {
            ...super.toJSON(),
            trustType: this.trustType,
            trusteeId: this.trusteeId,
            beneficiaries: this.beneficiaries,
            isRevocable: this.isRevocable(),
            totalBeneficiaryAllocation: this.getTotalBeneficiaryAllocation()
        };
    }
}
//# sourceMappingURL=TrustAccount.js.map