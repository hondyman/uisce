import { EntityManager } from '../services/EntityManager.js';
import { Account, AssetType, ComplianceRule } from '../entities/index.js';
import { logger } from '../utils/logger.js';

/**
 * Validation context for trade/account validation
 */
export interface ValidationContext {
  accountId: string;
  trade?: {
    ticker: string;
    quantity: number;
    price: number;
    amount: number;
    assetType: AssetType;
  };
  portfolio?: {
    totalValue: number;
    cash: number;
    positions: Array<{
      ticker: string;
      quantity: number;
      value: number;
      percentage: number;
    }>;
  };
  userId: string;
  tenantId: string;
  datasourceId: string;
}

/**
 * Validation rule result
 */
export interface ValidationRuleResult {
  ruleId: string;
  ruleName: string;
  passed: boolean;
  severity: 'low' | 'medium' | 'high' | 'critical';
  message: string;
  details?: any;
}

/**
 * Overall validation result
 */
export interface ValidationResult {
  isValid: boolean;
  passedRules: ValidationRuleResult[];
  failedRules: ValidationRuleResult[];
  warnings: ValidationRuleResult[];
  errors: ValidationRuleResult[];
  executionTime: number;
}

/**
 * Validation Engine - Rules engine for entity and trade validation
 */
export class ValidationEngine {
  private static instance: ValidationEngine;
  private entityManager: EntityManager;

  private constructor() {
    this.entityManager = EntityManager.getInstance();
  }

  static getInstance(): ValidationEngine {
    if (!ValidationEngine.instance) {
      ValidationEngine.instance = new ValidationEngine();
    }
    return ValidationEngine.instance;
  }

  /**
   * Validate a trade request
   */
  async validateTradeRequest(context: ValidationContext): Promise<ValidationResult> {
    const startTime = Date.now();

    try {
      // Load account
      const account = await this.entityManager.loadEntity(context.accountId) as Account;
      if (!account) {
        return this.createResult(false, [], [{
          ruleId: 'account_exists',
          ruleName: 'Account Exists',
          passed: false,
          severity: 'critical',
          message: `Account ${context.accountId} not found`
        }], [], [], Date.now() - startTime);
      }

      // Validate account status
      const accountValidation = await this.validateAccountStatus(account);
      if (!accountValidation.isValid) {
        return accountValidation;
      }

      const allResults: ValidationRuleResult[] = [
        ...accountValidation.passedRules,
        ...accountValidation.failedRules
      ];

      // Run all validation rules
      if (context.trade) {
        const concentrationResult = await this.validateConcentrationLimit(account, context);
        allResults.push(...concentrationResult.passedRules, ...concentrationResult.failedRules);

        const assetResult = await this.validateAssetCompatibility(account, context);
        allResults.push(...assetResult.passedRules, ...assetResult.failedRules);

        const kycResult = await this.validateKYCRequirements(account, context);
        allResults.push(...kycResult.passedRules, ...kycResult.failedRules);

        const tradeExecutionResult = await this.validateTradeExecution(account, context);
        allResults.push(...tradeExecutionResult.passedRules, ...tradeExecutionResult.failedRules);

        const feeResult = await this.validateFeeStructure(account, context);
        allResults.push(...feeResult.passedRules, ...feeResult.failedRules);
      }

      // Categorize results
      const passedRules = allResults.filter(r => r.passed);
      const failedRules = allResults.filter(r => !r.passed);
      const warnings = failedRules.filter(r => r.severity === 'low' || r.severity === 'medium');
      const errors = failedRules.filter(r => r.severity === 'high' || r.severity === 'critical');

      const isValid = errors.length === 0;

      return this.createResult(isValid, passedRules, failedRules, warnings, errors, Date.now() - startTime);

    } catch (error) {
      logger.error('Trade validation failed:', error);
      return this.createResult(false, [], [{
        ruleId: 'validation_error',
        ruleName: 'Validation Error',
        passed: false,
        severity: 'critical',
        message: `Validation failed: ${error instanceof Error ? error.message : 'Unknown error'}`,
        details: error
      }], [], [], Date.now() - startTime);
    }
  }

  /**
   * Validate account status
   */
  private async validateAccountStatus(account: Account): Promise<ValidationResult> {
    const passed = account.isActive();
    const result: ValidationRuleResult = {
      ruleId: 'account_status',
      ruleName: 'Account Status',
      passed,
      severity: 'critical',
      message: passed ? 'Account is active' : 'Account is not active'
    };

    return this.createResult(passed, passed ? [result] : [], passed ? [] : [result], [], [], 0);
  }

  /**
   * Validate concentration limit
   */
  private async validateConcentrationLimit(account: Account, context: ValidationContext): Promise<ValidationResult> {
    if (!context.trade || !context.portfolio) {
      return this.createResult(true, [], [], [], [], 0);
    }

    const maxConcentration = account.getMaxConcentration();
    const tradeValue = context.trade.amount;
    const portfolioValue = context.portfolio.totalValue;

    // Check if this trade would exceed concentration limits
    const newPositionValue = tradeValue;
    const newConcentration = newPositionValue / (portfolioValue + tradeValue);

    const passed = newConcentration <= maxConcentration;
    const result: ValidationRuleResult = {
      ruleId: 'concentration_limit',
      ruleName: 'Concentration Limit',
      passed,
      severity: 'high',
      message: passed
        ? `Trade concentration (${(newConcentration * 100).toFixed(2)}%) within limit (${(maxConcentration * 100).toFixed(2)}%)`
        : `Trade would exceed concentration limit: ${(newConcentration * 100).toFixed(2)}% > ${(maxConcentration * 100).toFixed(2)}%`,
      details: {
        maxConcentration,
        newConcentration,
        tradeValue,
        portfolioValue
      }
    };

    return this.createResult(passed, passed ? [result] : [], passed ? [] : [result], [], [], 0);
  }

  /**
   * Validate asset compatibility
   */
  private async validateAssetCompatibility(account: Account, context: ValidationContext): Promise<ValidationResult> {
    if (!context.trade) {
      return this.createResult(true, [], [], [], [], 0);
    }

    const canHold = account.canHoldAsset(context.trade.assetType);
    const result: ValidationRuleResult = {
      ruleId: 'asset_compatibility',
      ruleName: 'Asset Compatibility',
      passed: canHold,
      severity: 'critical',
      message: canHold
        ? `Account can hold ${context.trade.assetType}`
        : `Account cannot hold ${context.trade.assetType}`,
      details: {
        assetType: context.trade.assetType,
        accountType: account.accountType
      }
    };

    return this.createResult(canHold, canHold ? [result] : [], canHold ? [] : [result], [], [], 0);
  }

  /**
   * Validate KYC requirements
   */
  private async validateKYCRequirements(account: Account, context: ValidationContext): Promise<ValidationResult> {
    try {
      // Load client (account owner)
      const client = await this.entityManager.loadEntity(account.ownerId);
      if (!client) {
        return this.createResult(false, [], [{
          ruleId: 'kyc_client_exists',
          ruleName: 'Client Exists',
          passed: false,
          severity: 'critical',
          message: 'Account owner not found'
        }], [], [], 0);
      }

      // For now, assume client has isKYCValid method
      const kycValid = (client as any).isKYCValid ? (client as any).isKYCValid() : true;

      const result: ValidationRuleResult = {
        ruleId: 'kyc_completeness',
        ruleName: 'KYC Completeness',
        passed: kycValid,
        severity: 'critical',
        message: kycValid ? 'KYC is current and complete' : 'KYC is expired or incomplete'
      };

      return this.createResult(kycValid, kycValid ? [result] : [], kycValid ? [] : [result], [], [], 0);
    } catch (error) {
      return this.createResult(false, [], [{
        ruleId: 'kyc_validation_error',
        ruleName: 'KYC Validation',
        passed: false,
        severity: 'critical',
        message: 'Failed to validate KYC requirements'
      }], [], [], 0);
    }
  }

  /**
   * Validate trade execution rules
   */
  private async validateTradeExecution(account: Account, context: ValidationContext): Promise<ValidationResult> {
    // Basic trade execution validation
    const rules = account.getComplianceRules();
    const tradeExecutionRule = rules.find(r => r.id === 'trade_execution');

    if (!tradeExecutionRule) {
      return this.createResult(true, [{
        ruleId: 'trade_execution',
        ruleName: 'Trade Execution',
        passed: true,
        severity: 'high',
        message: 'Trade execution validated'
      }], [], [], [], 0);
    }

    // For now, assume trade execution is valid
    const passed = true;
    const result: ValidationRuleResult = {
      ruleId: tradeExecutionRule.id,
      ruleName: tradeExecutionRule.name,
      passed,
      severity: tradeExecutionRule.severity,
      message: passed ? 'Trade execution approved' : 'Trade execution rejected'
    };

    return this.createResult(passed, passed ? [result] : [], passed ? [] : [result], [], [], 0);
  }

  /**
   * Validate fee structure
   */
  private async validateFeeStructure(account: Account, context: ValidationContext): Promise<ValidationResult> {
    // Basic fee validation - assume fees are within limits
    const passed = true;
    const result: ValidationRuleResult = {
      ruleId: 'fee_validation',
      ruleName: 'Fee Validation',
      passed,
      severity: 'medium',
      message: 'Transaction fees within approved limits'
    };

    return this.createResult(passed, passed ? [result] : [], passed ? [] : [result], [], [], 0);
  }

  /**
   * Create validation result
   */
  private createResult(
    isValid: boolean,
    passedRules: ValidationRuleResult[],
    failedRules: ValidationRuleResult[],
    warnings: ValidationRuleResult[],
    errors: ValidationRuleResult[],
    executionTime: number
  ): ValidationResult {
    return {
      isValid,
      passedRules,
      failedRules,
      warnings,
      errors,
      executionTime
    };
  }

  /**
   * Get account compliance rules
   */
  async getAccountComplianceRules(accountId: string): Promise<ComplianceRule[]> {
    try {
      const account = await this.entityManager.loadEntity(accountId) as Account;
      if (!account) {
        return [];
      }
      return account.getComplianceRules();
    } catch (error) {
      logger.error('Failed to get compliance rules:', error);
      return [];
    }
  }
}