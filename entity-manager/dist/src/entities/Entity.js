export class Entity {
    id;
    createdAt;
    updatedAt;
    tenantId;
    datasourceId;
    constructor(id, tenantId, datasourceId, createdAt, updatedAt) {
        this.id = id;
        this.tenantId = tenantId;
        this.datasourceId = datasourceId;
        this.createdAt = createdAt || new Date();
        this.updatedAt = updatedAt || new Date();
    }
    markAsUpdated() {
        this.updatedAt = new Date();
    }
}
export var KycStatus;
(function (KycStatus) {
    KycStatus["NOT_STARTED"] = "not_started";
    KycStatus["PENDING"] = "pending";
    KycStatus["IN_REVIEW"] = "in_review";
    KycStatus["APPROVED"] = "approved";
    KycStatus["REJECTED"] = "rejected";
    KycStatus["EXPIRED"] = "expired";
})(KycStatus || (KycStatus = {}));
export var SanctionsStatus;
(function (SanctionsStatus) {
    SanctionsStatus["CLEAR"] = "clear";
    SanctionsStatus["PENDING"] = "pending";
    SanctionsStatus["FLAGGED"] = "flagged";
    SanctionsStatus["BLOCKED"] = "blocked";
})(SanctionsStatus || (SanctionsStatus = {}));
export var PepStatus;
(function (PepStatus) {
    PepStatus["NOT_CHECKED"] = "not_checked";
    PepStatus["CLEAR"] = "clear";
    PepStatus["PEP"] = "pep";
    PepStatus["RELATIVE"] = "relative";
    PepStatus["CLOSE_ASSOCIATE"] = "close_associate";
})(PepStatus || (PepStatus = {}));
//# sourceMappingURL=Entity.js.map