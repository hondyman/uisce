import { Entity, ValidationResult, KycStatus, SanctionsStatus, PepStatus } from './Entity.js';

/**
 * Client entity representing an individual or organization
 * Extends base Entity with KYC and compliance features
 */
export class Client extends Entity {
  public readonly firstName: string;
  public readonly lastName: string;
  public readonly email: string;
  public readonly dateOfBirth: Date;
  public readonly ssn?: string; // Optional for international clients

  // KYC Fields
  public kycStatus: KycStatus;
  public sanctionsStatus: SanctionsStatus;
  public pepStatus: PepStatus;
  public kycExpiresAt?: Date;
  public kycLastCheckedAt?: Date;

  // Additional compliance fields
  public isAccreditedInvestor: boolean;
  public riskTolerance?: string;
  public netWorth?: number;
  public annualIncome?: number;

  constructor(
    id: string,
    tenantId: string,
    datasourceId: string,
    firstName: string,
    lastName: string,
    email: string,
    dateOfBirth: Date,
    options: {
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
    } = {}
  ) {
    super(id, tenantId, datasourceId, options.createdAt, options.updatedAt);

    this.firstName = firstName;
    this.lastName = lastName;
    this.email = email;
    this.dateOfBirth = dateOfBirth;
    this.ssn = options.ssn;

    // KYC defaults
    this.kycStatus = options.kycStatus || KycStatus.NOT_STARTED;
    this.sanctionsStatus = options.sanctionsStatus || SanctionsStatus.PENDING;
    this.pepStatus = options.pepStatus || PepStatus.NOT_CHECKED;
    this.kycExpiresAt = options.kycExpiresAt;
    this.kycLastCheckedAt = options.kycLastCheckedAt;

    // Additional fields
    this.isAccreditedInvestor = options.isAccreditedInvestor || false;
    this.riskTolerance = options.riskTolerance;
    this.netWorth = options.netWorth;
    this.annualIncome = options.annualIncome;
  }

  /**
   * Get the entity type
   */
  getEntityType(): string {
    return 'client';
  }

  /**
   * Validate the client entity
   */
  async validate(): Promise<ValidationResult> {
    const errors: string[] = [];
    const warnings: string[] = [];

    // Required field validation
    if (!this.firstName.trim()) errors.push('First name is required');
    if (!this.lastName.trim()) errors.push('Last name is required');
    if (!this.email.trim()) errors.push('Email is required');
    if (!this.dateOfBirth) errors.push('Date of birth is required');

    // Email format validation
    const emailRegex = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;
    if (this.email && !emailRegex.test(this.email)) {
      errors.push('Invalid email format');
    }

    // Age validation (must be 18+)
    const age = this.calculateAge();
    if (age < 18) {
      errors.push('Client must be at least 18 years old');
    }

    // KYC validation
    if (this.kycStatus === KycStatus.EXPIRED) {
      errors.push('KYC has expired and must be renewed');
    }

    if (this.kycExpiresAt && this.kycExpiresAt < new Date()) {
      warnings.push('KYC is expiring soon');
    }

    // Sanctions check
    if (this.sanctionsStatus === SanctionsStatus.BLOCKED) {
      errors.push('Client is blocked due to sanctions');
    }

    // PEP check
    if (this.pepStatus === PepStatus.PEP || this.pepStatus === PepStatus.CLOSE_ASSOCIATE) {
      warnings.push('Client is a politically exposed person - additional due diligence required');
    }

    return {
      isValid: errors.length === 0,
      errors,
      warnings
    };
  }

  /**
   * Check if KYC is valid and current
   */
  isKYCValid(): boolean {
    if (this.kycStatus !== KycStatus.APPROVED) {
      return false;
    }

    if (this.kycExpiresAt && this.kycExpiresAt < new Date()) {
      return false;
    }

    return true;
  }

  /**
   * Check if client is accredited investor
   */
  isAccreditedInvestorCheck(): boolean {
    // SEC accredited investor criteria
    if (this.netWorth && this.netWorth >= 1000000) {
      return true;
    }

    if (this.annualIncome && this.annualIncome >= 200000) {
      return true;
    }

    return this.isAccreditedInvestor;
  }

  /**
   * Calculate client's current age
   */
  private calculateAge(): number {
    const today = new Date();
    let age = today.getFullYear() - this.dateOfBirth.getFullYear();
    const monthDiff = today.getMonth() - this.dateOfBirth.getMonth();

    if (monthDiff < 0 || (monthDiff === 0 && today.getDate() < this.dateOfBirth.getDate())) {
      age--;
    }

    return age;
  }

  /**
   * Get full name
   */
  getFullName(): string {
    return `${this.firstName} ${this.lastName}`;
  }

  /**
   * Convert to JSON for serialization
   */
  toJSON(): Record<string, any> {
    return {
      id: this.id,
      entityType: this.getEntityType(),
      tenantId: this.tenantId,
      datasourceId: this.datasourceId,
      firstName: this.firstName,
      lastName: this.lastName,
      email: this.email,
      dateOfBirth: this.dateOfBirth.toISOString(),
      ssn: this.ssn ? '***-**-****' : undefined, // Mask SSN
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