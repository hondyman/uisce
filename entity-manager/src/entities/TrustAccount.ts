import { Account, AssetType, ApprovalRoute, ComplianceRule, AccountStatus } from './Account.js';

/**
 * Trust account types
 */
export enum TrustType {
  REVOCABLE = 'revocable',
  IRREVOCABLE = 'irrevocable',
  CHARITABLE = 'charitable',
  GRANTOR = 'grantor'
}

/**
 * Beneficiary interface
 */
export interface Beneficiary {
  id: string;
  name: string;
  relationship: string;
  percentage: number;
  ssn?: string;
}

/**
 * Trust Account - Most conservative account type
 * Strict approval chains, fiduciary duty requirements
 */
export class TrustAccount extends Account {
  public trustType: TrustType;
  public trusteeId: string;
  public beneficiaries: Beneficiary[];

  constructor(
    id: string,
    tenantId: string,
    datasourceId: string,
    accountNumber: string,
    name: string,
    ownerId: string,
    custodianId: string,
    trustType: TrustType = TrustType.REVOCABLE,
    trusteeId: string,
    beneficiaries: Beneficiary[] = [],
    status: AccountStatus = AccountStatus.PENDING,
    createdAt?: Date,
    updatedAt?: Date
  ) {
    super(id, tenantId, datasourceId, accountNumber, name, ownerId, custodianId, 'trust', status, createdAt, updatedAt);
    this.trustType = trustType;
    this.trusteeId = trusteeId;
    this.beneficiaries = beneficiaries;
  }

  /**
   * Get maximum concentration limit (15% for trusts)
   */
  getMaxConcentration(): number {
    return 0.15; // 15%
  }

  /**
   * Check if account can hold specific asset type
   * Trusts have the most restrictive asset rules
   */
  canHoldAsset(assetType: AssetType): boolean {
    // Prohibited assets for trusts
    const prohibitedAssets = new Set<AssetType>([
      AssetType.CRYPTO,
      AssetType.DERIVATIVE,
      AssetType.SHORT_SELLING,
      AssetType.HEDGE_FUND,
      AssetType.PRIVATE_EQUITY,
      AssetType.ALTERNATIVE
    ]);

    // Additional restrictions for charitable trusts
    if (this.trustType === TrustType.CHARITABLE) {
      prohibitedAssets.add(AssetType.REAL_ESTATE); // May have special rules
    }

    return !prohibitedAssets.has(assetType);
  }

  /**
   * Get approval chain - strict 3-level approval for trusts
   */
  getApprovalChain(amount: number): ApprovalRoute[] {
    return [
      {
        level: 1,
        approvers: ['advisor'],
        threshold: Number.MAX_SAFE_INTEGER,
        requiredCount: 1,
        timeoutMinutes: 30
      },
      {
        level: 2,
        approvers: ['compliance_officer'],
        threshold: Number.MAX_SAFE_INTEGER,
        requiredCount: 1,
        timeoutMinutes: 120
      },
      {
        level: 3,
        approvers: ['cfo', 'trustee'],
        threshold: Number.MAX_SAFE_INTEGER,
        requiredCount: 1, // At least one from CFO or Trustee
        timeoutMinutes: 480 // 8 hours
      }
    ];
  }

  /**
   * Get compliance rules for trust accounts
   */
  getComplianceRules(): ComplianceRule[] {
    const rules: ComplianceRule[] = [
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
        id: 'fiduciary_duty',
        name: 'Fiduciary Duty',
        description: 'Trustee must act in the best interest of beneficiaries',
        category: 'regulatory',
        severity: 'critical'
      },
      {
        id: 'beneficiary_protection',
        name: 'Beneficiary Protection',
        description: 'All actions must protect beneficiary interests',
        category: 'regulatory',
        severity: 'critical'
      },
      {
        id: 'trust_documentation',
        name: 'Trust Documentation',
        description: 'All trust documents must be current and on file',
        category: 'legal',
        severity: 'critical'
      }
    ];

    // Additional rules for irrevocable trusts
    if (this.trustType === TrustType.IRREVOCABLE) {
      rules.push({
        id: 'irrevocable_restrictions',
        name: 'Irrevocable Restrictions',
        description: 'Trust cannot be modified without beneficiary consent',
        category: 'legal',
        severity: 'critical'
      });
    }

    return rules;
  }

  /**
   * Check if trust is revocable
   */
  isRevocable(): boolean {
    return this.trustType === TrustType.REVOCABLE || this.trustType === TrustType.GRANTOR;
  }

  /**
   * Add a beneficiary to the trust
   */
  addBeneficiary(beneficiary: Beneficiary): void {
    // Validate beneficiary percentage doesn't exceed 100%
    const totalPercentage = this.beneficiaries.reduce((sum, b) => sum + b.percentage, 0) + beneficiary.percentage;
    if (totalPercentage > 100) {
      throw new Error('Total beneficiary percentages cannot exceed 100%');
    }

    this.beneficiaries.push(beneficiary);
    this.markAsUpdated();
  }

  /**
   * Remove a beneficiary from the trust
   */
  removeBeneficiary(beneficiaryId: string): void {
    const index = this.beneficiaries.findIndex(b => b.id === beneficiaryId);
    if (index === -1) {
      throw new Error('Beneficiary not found');
    }

    this.beneficiaries.splice(index, 1);
    this.markAsUpdated();
  }

  /**
   * Get total beneficiary allocation
   */
  getTotalBeneficiaryAllocation(): number {
    return this.beneficiaries.reduce((sum, b) => sum + b.percentage, 0);
  }

  /**
   * Convert to JSON with trust specific fields
   */
  toJSON(): Record<string, any> {
    return {
      ...super.toJSON(),
      trustType: this.trustType,
      trusteeId: this.trusteeId,
      beneficiaries: this.beneficiaries,
      isRevocable: this.isRevocable(),
      totalBeneficiaryAllocation: this.getTotalBeneficiaryAllocation()
    };
  }
}