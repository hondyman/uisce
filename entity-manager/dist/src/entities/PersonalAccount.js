import { Account, AssetType, AccountStatus } from './Account.js';
export var RiskTolerance;
(function (RiskTolerance) {
    RiskTolerance["LOW"] = "low";
    RiskTolerance["MEDIUM"] = "medium";
    RiskTolerance["HIGH"] = "high";
    RiskTolerance["AGGRESSIVE"] = "aggressive";
})(RiskTolerance || (RiskTolerance = {}));
export var InvestmentObjective;
(function (InvestmentObjective) {
    InvestmentObjective["INCOME"] = "income";
    InvestmentObjective["GROWTH"] = "growth";
    InvestmentObjective["BALANCED"] = "balanced";
    InvestmentObjective["PRESERVATION"] = "preservation";
})(InvestmentObjective || (InvestmentObjective = {}));
export class PersonalAccount extends Account {
    riskTolerance;
    investmentObjective;
    netWorth;
    constructor(id, tenantId, datasourceId, accountNumber, name, ownerId, custodianId, riskTolerance = RiskTolerance.MEDIUM, investmentObjective = InvestmentObjective.BALANCED, status = AccountStatus.PENDING, netWorth, createdAt, updatedAt) {
        super(id, tenantId, datasourceId, accountNumber, name, ownerId, custodianId, 'personal', status, createdAt, updatedAt);
        this.riskTolerance = riskTolerance;
        this.investmentObjective = investmentObjective;
        this.netWorth = netWorth;
    }
    getMaxConcentration() {
        switch (this.riskTolerance) {
            case RiskTolerance.LOW:
                return 0.20;
            case RiskTolerance.MEDIUM:
                return 0.30;
            case RiskTolerance.HIGH:
                return 0.40;
            case RiskTolerance.AGGRESSIVE:
                return 0.50;
            default:
                return 0.25;
        }
    }
    canHoldAsset(assetType) {
        const restrictedAssets = new Set();
        if (this.riskTolerance === RiskTolerance.LOW) {
            restrictedAssets.add(AssetType.CRYPTO);
            restrictedAssets.add(AssetType.ALTERNATIVE);
            restrictedAssets.add(AssetType.DERIVATIVE);
            restrictedAssets.add(AssetType.SHORT_SELLING);
        }
        else if (this.riskTolerance === RiskTolerance.MEDIUM) {
            restrictedAssets.add(AssetType.ALTERNATIVE);
            restrictedAssets.add(AssetType.DERIVATIVE);
            restrictedAssets.add(AssetType.SHORT_SELLING);
        }
        else if (this.riskTolerance === RiskTolerance.HIGH || this.riskTolerance === RiskTolerance.AGGRESSIVE) {
            restrictedAssets.add(AssetType.PRIVATE_EQUITY);
        }
        return !restrictedAssets.has(assetType);
    }
    getApprovalChain(amount) {
        const routes = [];
        if (amount <= 50000) {
            routes.push({
                level: 1,
                approvers: ['advisor'],
                threshold: 50000,
                requiredCount: 1,
                timeoutMinutes: 60
            });
        }
        else if (amount <= 250000) {
            routes.push({
                level: 1,
                approvers: ['advisor'],
                threshold: 50000,
                requiredCount: 1,
                timeoutMinutes: 30
            }, {
                level: 2,
                approvers: ['regional_director'],
                threshold: 250000,
                requiredCount: 1,
                timeoutMinutes: 120
            });
        }
        else {
            routes.push({
                level: 1,
                approvers: ['advisor'],
                threshold: 50000,
                requiredCount: 1,
                timeoutMinutes: 30
            }, {
                level: 2,
                approvers: ['regional_director'],
                threshold: 250000,
                requiredCount: 1,
                timeoutMinutes: 120
            }, {
                level: 3,
                approvers: ['compliance_officer'],
                threshold: 1000000,
                requiredCount: 1,
                timeoutMinutes: 240
            });
        }
        return routes;
    }
    getComplianceRules() {
        return [
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
                id: 'risk_tolerance_alignment',
                name: 'Risk Tolerance Alignment',
                description: `Investments must align with ${this.riskTolerance} risk tolerance`,
                category: 'suitability',
                severity: 'high'
            }
        ];
    }
    toJSON() {
        return {
            ...super.toJSON(),
            riskTolerance: this.riskTolerance,
            investmentObjective: this.investmentObjective,
            netWorth: this.netWorth
        };
    }
}
//# sourceMappingURL=PersonalAccount.js.map