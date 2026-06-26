import { Entity } from './Entity.js';
export var AssetType;
(function (AssetType) {
    AssetType["EQUITY"] = "equity";
    AssetType["BOND"] = "bond";
    AssetType["MUTUAL_FUND"] = "mutual_fund";
    AssetType["ETF"] = "etf";
    AssetType["CRYPTO"] = "crypto";
    AssetType["REAL_ESTATE"] = "real_estate";
    AssetType["PRIVATE_EQUITY"] = "private_equity";
    AssetType["HEDGE_FUND"] = "hedge_fund";
    AssetType["DERIVATIVE"] = "derivative";
    AssetType["SHORT_SELLING"] = "short_selling";
    AssetType["ALTERNATIVE"] = "alternative";
})(AssetType || (AssetType = {}));
export var AccountStatus;
(function (AccountStatus) {
    AccountStatus["PENDING"] = "pending";
    AccountStatus["ACTIVE"] = "active";
    AccountStatus["SUSPENDED"] = "suspended";
    AccountStatus["CLOSED"] = "closed";
    AccountStatus["FROZEN"] = "frozen";
})(AccountStatus || (AccountStatus = {}));
export class Account extends Entity {
    accountNumber;
    name;
    ownerId;
    custodianId;
    status;
    accountType;
    constructor(id, tenantId, datasourceId, accountNumber, name, ownerId, custodianId, accountType, status = AccountStatus.PENDING, createdAt, updatedAt) {
        super(id, tenantId, datasourceId, createdAt, updatedAt);
        this.accountNumber = accountNumber;
        this.name = name;
        this.ownerId = ownerId;
        this.custodianId = custodianId;
        this.accountType = accountType;
        this.status = status;
    }
    getEntityType() {
        return 'account';
    }
    async validate() {
        const errors = [];
        const warnings = [];
        if (!this.accountNumber.trim())
            errors.push('Account number is required');
        if (!this.name.trim())
            errors.push('Account name is required');
        if (!this.ownerId.trim())
            errors.push('Owner ID is required');
        if (!this.custodianId.trim())
            errors.push('Custodian ID is required');
        if (this.accountNumber.length < 3) {
            errors.push('Account number must be at least 3 characters');
        }
        if (!Object.values(AccountStatus).includes(this.status)) {
            errors.push('Invalid account status');
        }
        return {
            isValid: errors.length === 0,
            errors,
            warnings
        };
    }
    isActive() {
        return this.status === AccountStatus.ACTIVE;
    }
    toJSON() {
        return {
            id: this.id,
            entityType: this.getEntityType(),
            tenantId: this.tenantId,
            datasourceId: this.datasourceId,
            accountNumber: this.accountNumber,
            name: this.name,
            ownerId: this.ownerId,
            custodianId: this.custodianId,
            accountType: this.accountType,
            status: this.status,
            maxConcentration: this.getMaxConcentration(),
            complianceRules: this.getComplianceRules(),
            createdAt: this.createdAt.toISOString(),
            updatedAt: this.updatedAt.toISOString()
        };
    }
}
//# sourceMappingURL=Account.js.map