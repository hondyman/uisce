import { Account, AssetType, AccountStatus } from './Account.js';
export var IRAType;
(function (IRAType) {
    IRAType["TRADITIONAL"] = "traditional";
    IRAType["ROTH"] = "roth";
    IRAType["SEP"] = "sep";
    IRAType["SIMPLE"] = "simple";
    IRAType["ROLLOVER"] = "rollover";
})(IRAType || (IRAType = {}));
export class IRAAccount extends Account {
    iraType;
    ownerAge;
    contributionLimit;
    constructor(id, tenantId, datasourceId, accountNumber, name, ownerId, custodianId, iraType = IRAType.TRADITIONAL, ownerAge, contributionLimit, status = AccountStatus.PENDING, createdAt, updatedAt) {
        super(id, tenantId, datasourceId, accountNumber, name, ownerId, custodianId, 'ira', status, createdAt, updatedAt);
        this.iraType = iraType;
        this.ownerAge = ownerAge;
        this.contributionLimit = contributionLimit;
    }
    getMaxConcentration() {
        return 0.25;
    }
    canHoldAsset(assetType) {
        const prohibitedAssets = new Set([
            AssetType.CRYPTO,
            AssetType.ALTERNATIVE,
            AssetType.DERIVATIVE,
            AssetType.SHORT_SELLING,
            AssetType.HEDGE_FUND,
            AssetType.PRIVATE_EQUITY
        ]);
        return !prohibitedAssets.has(assetType);
    }
    getApprovalChain(amount) {
        return [
            {
                level: 1,
                approvers: ['advisor'],
                threshold: Number.MAX_SAFE_INTEGER,
                requiredCount: 1,
                timeoutMinutes: 60
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
                id: 'ira_eligibility',
                name: 'IRA Eligibility',
                description: 'Account holder must meet IRA contribution and eligibility requirements',
                category: 'regulatory',
                severity: 'critical'
            },
            {
                id: 'prohibited_assets',
                name: 'Prohibited Assets',
                description: 'Crypto, alternatives, derivatives, and short selling are prohibited',
                category: 'regulatory',
                severity: 'critical'
            }
        ];
        if (this.iraType === IRAType.TRADITIONAL && this.ownerAge >= 72) {
            rules.push({
                id: 'required_minimum_distribution',
                name: 'Required Minimum Distribution',
                description: 'Annual RMD must be calculated and distributed',
                category: 'regulatory',
                severity: 'critical'
            });
        }
        return rules;
    }
    canContribute(amount) {
        if (amount > this.contributionLimit) {
            return false;
        }
        if (this.iraType === IRAType.TRADITIONAL) {
            return true;
        }
        else if (this.iraType === IRAType.ROTH) {
            return this.ownerAge < 70.5;
        }
        return true;
    }
    isEligibleForWithdrawal() {
        if (this.iraType === IRAType.ROTH) {
            return true;
        }
        if (this.iraType === IRAType.TRADITIONAL) {
            return this.ownerAge >= 59.5;
        }
        return this.ownerAge >= 59.5;
    }
    toJSON() {
        return {
            ...super.toJSON(),
            iraType: this.iraType,
            ownerAge: this.ownerAge,
            contributionLimit: this.contributionLimit
        };
    }
}
//# sourceMappingURL=IRAAccount.js.map