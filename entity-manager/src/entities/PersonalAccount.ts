import { Account, AssetType, ApprovalRoute, ComplianceRule, AccountStatus } from './Account.js';

/**
 * Risk tolerance levels for personal accounts
 */
export enum RiskTolerance {
  LOW = 'low',
  MEDIUM = 'medium',
  HIGH = 'high',
  AGGRESSIVE = 'aggressive'
}

/**
 * Investment objectives for personal accounts
 */
export enum InvestmentObjective {
  INCOME = 'income',
  GROWTH = 'growth',
  BALANCED = 'balanced',
  PRESERVATION = 'preservation'
}

/**
 * Personal Account - High flexibility account type
 * Allows most asset types based on risk tolerance
 */
export class PersonalAccount extends Account {
  public riskTolerance: RiskTolerance;
  public investmentObjective: InvestmentObjective;
  public netWorth?: number;

  constructor(
    id: string,
    tenantId: string,
    datasourceId: string,
    accountNumber: string,
    name: string,
    ownerId: string,
    custodianId: string,
    riskTolerance: RiskTolerance = RiskTolerance.MEDIUM,
    investmentObjective: InvestmentObjective = InvestmentObjective.BALANCED,
    status: AccountStatus = AccountStatus.PENDING,
    netWorth?: number,
    createdAt?: Date,
    updatedAt?: Date
  ) {
    super(id, tenantId, datasourceId, accountNumber, name, ownerId, custodianId, 'personal', status, createdAt, updatedAt);
    this.riskTolerance = riskTolerance;
    this.investmentObjective = investmentObjective;
    this.netWorth = netWorth;
  }

  /**
   * Get maximum concentration limit based on risk tolerance
   * Returns as decimal (0.20 = 20%)
   */
  getMaxConcentration(): number {
    switch (this.riskTolerance) {
      case RiskTolerance.LOW:
        return 0.20; // 20%
      case RiskTolerance.MEDIUM:
        return 0.30; // 30%
      case RiskTolerance.HIGH:
        return 0.40; // 40%
      case RiskTolerance.AGGRESSIVE:
        return 0.50; // 50%
      default:
        return 0.25; // 25% default
    }
  }

  /**
   * Check if account can hold specific asset type based on risk tolerance
   */
  canHoldAsset(assetType: AssetType): boolean {
    // Most assets are allowed for personal accounts
    const restrictedAssets = new Set<AssetType>();

    // Low risk: No crypto, alternatives, or derivatives
    if (this.riskTolerance === RiskTolerance.LOW) {
      restrictedAssets.add(AssetType.CRYPTO);
      restrictedAssets.add(AssetType.ALTERNATIVE);
      restrictedAssets.add(AssetType.DERIVATIVE);
      restrictedAssets.add(AssetType.SHORT_SELLING);
    }

    // Medium risk: No alternatives or derivatives
    else if (this.riskTolerance === RiskTolerance.MEDIUM) {
      restrictedAssets.add(AssetType.ALTERNATIVE);
      restrictedAssets.add(AssetType.DERIVATIVE);
      restrictedAssets.add(AssetType.SHORT_SELLING);
    }

    // High/Aggressive: Only restrict the most speculative
    else if (this.riskTolerance === RiskTolerance.HIGH || this.riskTolerance === RiskTolerance.AGGRESSIVE) {
      // Allow most assets, only restrict the most exotic
      restrictedAssets.add(AssetType.PRIVATE_EQUITY); // Might require special licensing
    }

    return !restrictedAssets.has(assetType);
  }

  /**
   * Get approval chain based on transaction amount
   */
  getApprovalChain(amount: number): ApprovalRoute[] {
    const routes: ApprovalRoute[] = [];

    // Small amounts: Advisor only
    if (amount <= 50000) {
      routes.push({
        level: 1,
        approvers: ['advisor'],
        threshold: 50000,
        requiredCount: 1,
        timeoutMinutes: 60
      });
    }
    // Medium amounts: Advisor + Regional Director
    else if (amount <= 250000) {
      routes.push(
        {
          level: 1,
          approvers: ['advisor'],
          threshold: 50000,
          requiredCount: 1,
          timeoutMinutes: 30
        },
        {
          level: 2,
          approvers: ['regional_director'],
          threshold: 250000,
          requiredCount: 1,
          timeoutMinutes: 120
        }
      );
    }
    // Large amounts: Full chain
    else {
      routes.push(
        {
          level: 1,
          approvers: ['advisor'],
          threshold: 50000,
          requiredCount: 1,
          timeoutMinutes: 30
        },
        {
          level: 2,
          approvers: ['regional_director'],
          threshold: 250000,
          requiredCount: 1,
          timeoutMinutes: 120
        },
        {
          level: 3,
          approvers: ['compliance_officer'],
          threshold: 1000000,
          requiredCount: 1,
          timeoutMinutes: 240
        }
      );
    }

    return routes;
  }

  /**
   * Get compliance rules for personal accounts
   */
  getComplianceRules(): ComplianceRule[] {
    return [
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
        id: 'risk_tolerance_alignment',
        name: 'Risk Tolerance Alignment',
        description: `Investments must align with ${this.riskTolerance} risk tolerance`,
        category: 'suitability',
        severity: 'high'
      }
    ];
  }

  /**
   * Convert to JSON with personal account specific fields
   */
  toJSON(): Record<string, any> {
    return {
      ...super.toJSON(),
      riskTolerance: this.riskTolerance,
      investmentObjective: this.investmentObjective,
      netWorth: this.netWorth
    };
  }
}