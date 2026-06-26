export declare abstract class Entity {
    readonly id: string;
    readonly createdAt: Date;
    updatedAt: Date;
    readonly tenantId: string;
    readonly datasourceId: string;
    constructor(id: string, tenantId: string, datasourceId: string, createdAt?: Date, updatedAt?: Date);
    abstract validate(): Promise<ValidationResult>;
    abstract getEntityType(): string;
    abstract toJSON(): Record<string, any>;
    protected markAsUpdated(): void;
}
export interface ValidationResult {
    isValid: boolean;
    errors: string[];
    warnings: string[];
}
export declare enum KycStatus {
    NOT_STARTED = "not_started",
    PENDING = "pending",
    IN_REVIEW = "in_review",
    APPROVED = "approved",
    REJECTED = "rejected",
    EXPIRED = "expired"
}
export declare enum SanctionsStatus {
    CLEAR = "clear",
    PENDING = "pending",
    FLAGGED = "flagged",
    BLOCKED = "blocked"
}
export declare enum PepStatus {
    NOT_CHECKED = "not_checked",
    CLEAR = "clear",
    PEP = "pep",
    RELATIVE = "relative",
    CLOSE_ASSOCIATE = "close_associate"
}
//# sourceMappingURL=Entity.d.ts.map