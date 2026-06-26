import { Entity, ValidationResult, KycStatus, SanctionsStatus, PepStatus } from './Entity.js';
export declare class Client extends Entity {
    readonly firstName: string;
    readonly lastName: string;
    readonly email: string;
    readonly dateOfBirth: Date;
    readonly ssn?: string;
    kycStatus: KycStatus;
    sanctionsStatus: SanctionsStatus;
    pepStatus: PepStatus;
    kycExpiresAt?: Date;
    kycLastCheckedAt?: Date;
    isAccreditedInvestor: boolean;
    riskTolerance?: string;
    netWorth?: number;
    annualIncome?: number;
    constructor(id: string, tenantId: string, datasourceId: string, firstName: string, lastName: string, email: string, dateOfBirth: Date, options?: {
        ssn?: string;
        kycStatus?: KycStatus;
        sanctionsStatus?: SanctionsStatus;
        pepStatus?: PepStatus;
        kycExpiresAt?: Date;
        kycLastCheckedAt?: Date;
        isAccreditedInvestor?: boolean;
        riskTolerance?: string;
        netWorth?: number;
        annualIncome?: number;
        createdAt?: Date;
        updatedAt?: Date;
    });
    getEntityType(): string;
    validate(): Promise<ValidationResult>;
    isKYCValid(): boolean;
    isAccreditedInvestorCheck(): boolean;
    private calculateAge;
    getFullName(): string;
    toJSON(): Record<string, any>;
}
//# sourceMappingURL=Client.d.ts.map