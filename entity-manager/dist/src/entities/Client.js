import { Entity, KycStatus, SanctionsStatus, PepStatus } from './Entity.js';
export class Client extends Entity {
    firstName;
    lastName;
    email;
    dateOfBirth;
    ssn;
    kycStatus;
    sanctionsStatus;
    pepStatus;
    kycExpiresAt;
    kycLastCheckedAt;
    isAccreditedInvestor;
    riskTolerance;
    netWorth;
    annualIncome;
    constructor(id, tenantId, datasourceId, firstName, lastName, email, dateOfBirth, options = {}) {
        super(id, tenantId, datasourceId, options.createdAt, options.updatedAt);
        this.firstName = firstName;
        this.lastName = lastName;
        this.email = email;
        this.dateOfBirth = dateOfBirth;
        this.ssn = options.ssn;
        this.kycStatus = options.kycStatus || KycStatus.NOT_STARTED;
        this.sanctionsStatus = options.sanctionsStatus || SanctionsStatus.PENDING;
        this.pepStatus = options.pepStatus || PepStatus.NOT_CHECKED;
        this.kycExpiresAt = options.kycExpiresAt;
        this.kycLastCheckedAt = options.kycLastCheckedAt;
        this.isAccreditedInvestor = options.isAccreditedInvestor || false;
        this.riskTolerance = options.riskTolerance;
        this.netWorth = options.netWorth;
        this.annualIncome = options.annualIncome;
    }
    getEntityType() {
        return 'client';
    }
    async validate() {
        const errors = [];
        const warnings = [];
        if (!this.firstName.trim())
            errors.push('First name is required');
        if (!this.lastName.trim())
            errors.push('Last name is required');
        if (!this.email.trim())
            errors.push('Email is required');
        if (!this.dateOfBirth)
            errors.push('Date of birth is required');
        const emailRegex = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;
        if (this.email && !emailRegex.test(this.email)) {
            errors.push('Invalid email format');
        }
        const age = this.calculateAge();
        if (age < 18) {
            errors.push('Client must be at least 18 years old');
        }
        if (this.kycStatus === KycStatus.EXPIRED) {
            errors.push('KYC has expired and must be renewed');
        }
        if (this.kycExpiresAt && this.kycExpiresAt < new Date()) {
            warnings.push('KYC is expiring soon');
        }
        if (this.sanctionsStatus === SanctionsStatus.BLOCKED) {
            errors.push('Client is blocked due to sanctions');
        }
        if (this.pepStatus === PepStatus.PEP || this.pepStatus === PepStatus.CLOSE_ASSOCIATE) {
            warnings.push('Client is a politically exposed person - additional due diligence required');
        }
        return {
            isValid: errors.length === 0,
            errors,
            warnings
        };
    }
    isKYCValid() {
        if (this.kycStatus !== KycStatus.APPROVED) {
            return false;
        }
        if (this.kycExpiresAt && this.kycExpiresAt < new Date()) {
            return false;
        }
        return true;
    }
    isAccreditedInvestorCheck() {
        if (this.netWorth && this.netWorth >= 1000000) {
            return true;
        }
        if (this.annualIncome && this.annualIncome >= 200000) {
            return true;
        }
        return this.isAccreditedInvestor;
    }
    calculateAge() {
        const today = new Date();
        let age = today.getFullYear() - this.dateOfBirth.getFullYear();
        const monthDiff = today.getMonth() - this.dateOfBirth.getMonth();
        if (monthDiff < 0 || (monthDiff === 0 && today.getDate() < this.dateOfBirth.getDate())) {
            age--;
        }
        return age;
    }
    getFullName() {
        return `${this.firstName} ${this.lastName}`;
    }
    toJSON() {
        return {
            id: this.id,
            entityType: this.getEntityType(),
            tenantId: this.tenantId,
            datasourceId: this.datasourceId,
            firstName: this.firstName,
            lastName: this.lastName,
            email: this.email,
            dateOfBirth: this.dateOfBirth.toISOString(),
            ssn: this.ssn ? '***-**-****' : undefined,
            kycStatus: this.kycStatus,
            sanctionsStatus: this.sanctionsStatus,
            pepStatus: this.pepStatus,
            kycExpiresAt: this.kycExpiresAt?.toISOString(),
            kycLastCheckedAt: this.kycLastCheckedAt?.toISOString(),
            isAccreditedInvestor: this.isAccreditedInvestor,
            riskTolerance: this.riskTolerance,
            netWorth: this.netWorth,
            annualIncome: this.annualIncome,
            createdAt: this.createdAt.toISOString(),
            updatedAt: this.updatedAt.toISOString()
        };
    }
}
//# sourceMappingURL=Client.js.map