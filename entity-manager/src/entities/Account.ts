import { Entity, ValidationResult } from './Entity.js';

/**
 * Asset types supported by the system
 */
export enum AssetType {
  EQUITY = 'equity',
  BOND = 'bond',
  MUTUAL_FUND = 'mutual_fund',
  ETF = 'etf',
  CRYPTO = 'crypto',
  REAL_ESTATE = 'real_estate',
  PRIVATE_EQUITY = 'private_equity',
  HEDGE_FUND = 'hedge_fund',
  DERIVATIVE = 'derivative',
  SHORT_SELLING = 'short_selling',
  ALTERNATIVE = 'alternative'
}

/**
 * Account status enumeration
 */
export enum AccountStatus {
  PENDING = 'pending',
  ACTIVE = 'active',
  SUSPENDED = 'suspended',
  CLOSED = 'closed',
  FROZEN = 'frozen'
}

/**
 * Approval route definition
 */
export interface ApprovalRoute {
  level: number;
  approvers: string[]; // User IDs
  threshold: number; // Amount threshold for this level
  requiredCount: number; // Number of approvals needed from this level
  timeoutMinutes: number;
}

/**
 * Compliance rule definition
 */
export interface ComplianceRule {
  id: string;
  name: string;
  description: string;
  category: string;
  severity: 'low' | 'medium' | 'high' | 'critical';
}

/**
 * Abstract Account class providing polymorphic behavior
 * Different account types implement different rules and constraints
 */
export abstract class Account extends Entity {
  public readonly accountNumber: string;
  public readonly name: string;
  public readonly ownerId: string; // Client ID
  public readonly custodianId: string;
  public status: AccountStatus;
  public readonly accountType: string; // 'personal', 'ira', 'trust'

  constructor(
    id: string,
    tenantId: string,
    datasourceId: string,
    accountNumber: string,
    name: string,
    ownerId: string,
    custodianId: string,
    accountType: string,
    status: AccountStatus = AccountStatus.PENDING,
    createdAt?: Date,
    updatedAt?: Date
  ) {
    super(id, tenantId, datasourceId, createdAt, updatedAt);
    this.accountNumber = accountNumber;
    this.name = name;
    this.ownerId = ownerId;
    this.custodianId = custodianId;
    this.accountType = accountType;
    this.status = status;
  }

  /**
   * Get the entity type
   */
  getEntityType(): string {
    return 'account';
  }

  /**
   * Get maximum concentration limit as a percentage (0.0 to 1.0)
   * Must be implemented by concrete account types
   */
  abstract getMaxConcentration(): number;

  /**
   * Check if account can hold a specific asset type
   * Must be implemented by concrete account types
   */
  abstract canHoldAsset(assetType: AssetType): boolean;

  /**
   * Get approval chain for a given transaction amount
   * Must be implemented by concrete account types
   */
  abstract getApprovalChain(amount: number): ApprovalRoute[];

  /**
   * Get compliance rules applicable to this account type
   * Must be implemented by concrete account types
   */
  abstract getComplianceRules(): ComplianceRule[];

  /**
   * Validate the account entity
   */
  async validate(): Promise<ValidationResult> {
    const errors: string[] = [];
    const warnings: string[] = [];

    // Required field validation
    if (!this.accountNumber.trim()) errors.push('Account number is required');
    if (!this.name.trim()) errors.push('Account name is required');
    if (!this.ownerId.trim()) errors.push('Owner ID is required');
    if (!this.custodianId.trim()) errors.push('Custodian ID is required');

    // Account number format validation (basic)
    if (this.accountNumber.length < 3) {
      errors.push('Account number must be at least 3 characters');
    }

    // Status validation
    if (!Object.values(AccountStatus).includes(this.status)) {
      errors.push('Invalid account status');
    }

    return {
      isValid: errors.length === 0,
      errors,
      warnings
    };
  }

  /**
   * Check if account is active
   */
  isActive(): boolean {
    return this.status === AccountStatus.ACTIVE;
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