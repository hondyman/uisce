import { Account, AssetType, ApprovalRoute, ComplianceRule, AccountStatus } from './Account.js';

/**
 * IRA account types
 */
export enum IRAType {
  TRADITIONAL = 'traditional',
  ROTH = 'roth',
  SEP = 'sep',
  SIMPLE = 'simple',
  ROLLOVER = 'rollover'
}

/**
 * IRA Account - Conservative account type with strict rules
 * Limited asset types, lower concentration limits, simplified approval
 */
export class IRAAccount extends Account {
  public iraType: IRAType;
  public ownerAge: number;
  public contributionLimit: number;

  constructor(
    id: string,
    tenantId: string,
    datasourceId: string,
    accountNumber: string,
    name: string,
    ownerId: string,
    custodianId: string,
    iraType: IRAType = IRAType.TRADITIONAL,
    ownerAge: number,
    contributionLimit: number,
    status: AccountStatus = AccountStatus.PENDING,
    createdAt?: Date,
    updatedAt?: Date
  ) {
    super(id, tenantId, datasourceId, accountNumber, name, ownerId, custodianId, 'ira', status, createdAt, updatedAt);
    this.iraType = iraType;
    this.ownerAge = ownerAge;
    this.contributionLimit = contributionLimit;
  }

  /**
   * Get maximum concentration limit (25% for IRAs)
   */
  getMaxConcentration(): number {
    return 0.25; // 25%
  }

  /**
   * Check if account can hold specific asset type
   * IRAs have strict restrictions
   */
  canHoldAsset(assetType: AssetType): boolean {
    // Prohibited assets for IRAs
    const prohibitedAssets = new Set<AssetType>([
      AssetType.CRYPTO,
      AssetType.ALTERNATIVE,
      AssetType.DERIVATIVE,
      AssetType.SHORT_SELLING,
      AssetType.HEDGE_FUND,
      AssetType.PRIVATE_EQUITY
    ]);

    return !prohibitedAssets.has(assetType);
  }

  /**
   * Get approval chain - simplified for IRAs (advisor only)
   */
  getApprovalChain(amount: number): ApprovalRoute[] {
    return [
      {
        level: 1,
        approvers: ['advisor'],
        threshold: Number.MAX_SAFE_INTEGER, // No amount limits for IRAs
        requiredCount: 1,
        timeoutMinutes: 60
      }
    ];
  }

  /**
   * Get compliance rules for IRA accounts
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
        id: 'ira_eligibility',
        name: 'IRA Eligibility',
        description: 'Account holder must meet IRA contribution and eligibility requirements',
        category: 'regulatory',
        severity: 'critical'
      },
      {
        id: 'prohibited_assets',
        name: 'Prohibited Assets',
        description: 'Crypto, alternatives, derivatives, and short selling are prohibited',
        category: 'regulatory',
        severity: 'critical'
      }
    ];

    // Add RMD rules for traditional IRAs when owner reaches 72
    if (this.iraType === IRAType.TRADITIONAL && this.ownerAge >= 72) {
      rules.push({
        id: 'required_minimum_distribution',
        name: 'Required Minimum Distribution',
        description: 'Annual RMD must be calculated and distributed',
        category: 'regulatory',
        severity: 'critical'
      });
    }

    return rules;
  }

  /**
   * Check if owner can contribute to this IRA
   */
  canContribute(amount: number): boolean {
    // Check contribution limit
    if (amount > this.contributionLimit) {
      return false;
    }

    // Age restrictions
    if (this.iraType === IRAType.TRADITIONAL) {
      // No age limit for contributions to traditional IRAs
      return true;
    } else if (this.iraType === IRAType.ROTH) {
      // Roth IRA contributions not allowed after 70½
      return this.ownerAge < 70.5;
    }

    return true;
  }

  /**
   * Check if owner is eligible for withdrawal
   */
  isEligibleForWithdrawal(): boolean {
    // Roth IRAs allow penalty-free withdrawals of contributions at any time
    if (this.iraType === IRAType.ROTH) {
      return true;
    }

    // Traditional IRAs: penalty-free after 59½
    if (this.iraType === IRAType.TRADITIONAL) {
      return this.ownerAge >= 59.5;
    }

    // SEP and SIMPLE IRAs have their own rules
    return this.ownerAge >= 59.5;
  }

  /**
   * Convert to JSON with IRA specific fields
   */
  toJSON(): Record<string, any> {
    return {
      ...super.toJSON(),
      iraType: this.iraType,
      ownerAge: this.ownerAge,
      contributionLimit: this.contributionLimit
    };
  }
}