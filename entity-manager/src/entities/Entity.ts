/**
 * Abstract base class for all entities in the system
 * Provides common functionality and enforces type safety
 */
export abstract class Entity {
  public readonly id: string;
  public readonly createdAt: Date;
  public updatedAt: Date;
  public readonly tenantId: string;
  public readonly datasourceId: string;

  constructor(
    id: string,
    tenantId: string,
    datasourceId: string,
    createdAt?: Date,
    updatedAt?: Date
  ) {
    this.id = id;
    this.tenantId = tenantId;
    this.datasourceId = datasourceId;
    this.createdAt = createdAt || new Date();
    this.updatedAt = updatedAt || new Date();
  }

  /**
   * Validate the entity's current state
   * Must be implemented by concrete classes
   */
  abstract validate(): Promise<ValidationResult>;

  /**
   * Get the entity type for polymorphic behavior
   */
  abstract getEntityType(): string;

  /**
   * Convert entity to plain object for serialization
   */
  abstract toJSON(): Record<string, any>;

  /**
   * Update the entity's timestamp
   */
  protected markAsUpdated(): void {
    this.updatedAt = new Date();
  }
}

/**
 * Validation result interface
 */
export interface ValidationResult {
  isValid: boolean;
  errors: string[];
  warnings: string[];
}

/**
 * KYC Status enumeration
 */
export enum KycStatus {
  NOT_STARTED = 'not_started',
  PENDING = 'pending',
  IN_REVIEW = 'in_review',
  APPROVED = 'approved',
  REJECTED = 'rejected',
  EXPIRED = 'expired'
}

/**
 * Sanctions status enumeration
 */
export enum SanctionsStatus {
  CLEAR = 'clear',
  PENDING = 'pending',
  FLAGGED = 'flagged',
  BLOCKED = 'blocked'
}

/**
 * PEP (Politically Exposed Person) status
 */
export enum PepStatus {
  NOT_CHECKED = 'not_checked',
  CLEAR = 'clear',
  PEP = 'pep',
  RELATIVE = 'relative',
  CLOSE_ASSOCIATE = 'close_associate'
}