import { EntityManager } from '../services/EntityManager.js';
import { ValidationEngine, ValidationContext, ValidationResult } from '../validation/ValidationEngine.js';
import { ApprovalWorkflowEngine, ApprovalRequest, WorkflowStatus } from '../approval/ApprovalWorkflowEngine.js';
import { Account } from '../entities/index.js';
import { logger } from '../utils/logger.js';

/**
 * Trade execution request
 */
export interface TradeExecutionRequest {
  accountId: string;
  trade: {
    ticker: string;
    quantity: number;
    price: number;
    assetType: string;
    amount: number;
  };
  portfolio: {
    totalValue: number;
    cash: number;
    positions: Array<{
      ticker: string;
      quantity: number;
      value: number;
      percentage: number;
    }>;
  };
  advisorId: string;
  tenantId: string;
  datasourceId: string;
}

/**
 * Trade execution response
 */
export interface TradeExecutionResponse {
  success: boolean;
  workflowId?: string;
  approvalChain?: Array<{
    level: number;
    approvers: string[];
    threshold: number;
    requiredCount: number;
    timeoutMinutes: number;
  }>;
  complianceRules?: Array<{
    id: string;
    name: string;
    description: string;
    category: string;
    severity: string;
  }>;
  validationResults?: {
    isValid: boolean;
    passedRules: Array<{
      ruleId: string;
      ruleName: string;
      severity: string;
      message: string;
    }>;
    failedRules: Array<{
      ruleId: string;
      ruleName: string;
      severity: string;
      message: string;
    }>;
    warnings: Array<{
      ruleId: string;
      ruleName: string;
      severity: string;
      message: string;
    }>;
    errors: Array<{
      ruleId: string;
      ruleName: string;
      severity: string;
      message: string;
    }>;
  };
  error?: string;
}

/**
 * Unified Validator - Orchestrates Entity → Validation → Approval → Execution
 */
export class UnifiedValidator {
  private static instance: UnifiedValidator;
  private entityManager: EntityManager;
  private validationEngine: ValidationEngine;
  private approvalEngine: ApprovalWorkflowEngine;

  private constructor() {
    this.entityManager = EntityManager.getInstance();
    this.validationEngine = ValidationEngine.getInstance();
    this.approvalEngine = ApprovalWorkflowEngine.getInstance();
  }

  static getInstance(): UnifiedValidator {
    if (!UnifiedValidator.instance) {
      UnifiedValidator.instance = new UnifiedValidator();
    }
    return UnifiedValidator.instance;
  }

  /**
   * Process trade execution request
   * Main entry point for the complete workflow
   */
  async processTradeRequest(request: TradeExecutionRequest): Promise<TradeExecutionResponse> {
    const startTime = Date.now();

    try {
      logger.info(`Processing trade request for account ${request.accountId}`);

      // Step 1: Entity Validation
      const account = await this.entityManager.loadEntity(request.accountId) as Account;
      if (!account) {
        return {
          success: false,
          error: `Account ${request.accountId} not found`
        };
      }

      // Step 2: Account Status Check
      if (!account.isActive()) {
        return {
          success: false,
          error: `Account ${request.accountId} is not active`
        };
      }

      // Step 3: Validation Rules Engine
      const validationContext: ValidationContext = {
        accountId: request.accountId,
        trade: {
          ticker: request.trade.ticker,
          quantity: request.trade.quantity,
          price: request.trade.price,
          amount: request.trade.amount,
          assetType: request.trade.assetType as any
        },
        portfolio: request.portfolio,
        userId: request.advisorId,
        tenantId: request.tenantId,
        datasourceId: request.datasourceId
      };

      const validationResult = await this.validationEngine.validateTradeRequest(validationContext);

      // Step 4: Account-Specific Validation
      const accountValidationPassed = await this.performAccountSpecificValidation(account, request);

      // Combine validation results
      const overallValidationPassed = validationResult.isValid && accountValidationPassed;

      if (!overallValidationPassed) {
        return {
          success: false,
          validationResults: {
            isValid: false,
            passedRules: validationResult.passedRules.map(r => ({
              ruleId: r.ruleId,
              ruleName: r.ruleName,
              severity: r.severity,
              message: r.message
            })),
            failedRules: validationResult.failedRules.map(r => ({
              ruleId: r.ruleId,
              ruleName: r.ruleName,
              severity: r.severity,
              message: r.message
            })),
            warnings: validationResult.warnings.map(r => ({
              ruleId: r.ruleId,
              ruleName: r.ruleName,
              severity: r.severity,
              message: r.message
            })),
            errors: validationResult.errors.map(r => ({
              ruleId: r.ruleId,
              ruleName: r.ruleName,
              severity: r.severity,
              message: r.message
            }))
          },
          error: 'Trade validation failed'
        };
      }

      // Step 5: Approval Routing
      const approvalRequest: ApprovalRequest = {
        id: `trade-${Date.now()}-${Math.random().toString(36).substr(2, 9)}`,
        accountId: request.accountId,
        tradeId: '', // Would be generated
        amount: request.trade.amount,
        description: `Trade: ${request.trade.quantity} ${request.trade.ticker} @ $${request.trade.price}`,
        requesterId: request.advisorId,
        tenantId: request.tenantId,
        datasourceId: request.datasourceId,
        createdAt: new Date()
      };

      const approvalResult = await this.approvalEngine.startApprovalWorkflow(approvalRequest);

      // Step 6: Get Compliance Rules
      const complianceRules = await this.validationEngine.getAccountComplianceRules(request.accountId);

      const executionTime = Date.now() - startTime;
      logger.info(`Trade request processed in ${executionTime}ms`);

      return {
        success: true,
        workflowId: approvalResult.workflowId,
        approvalChain: approvalResult.approvalChain.map(route => ({
          level: route.level,
          approvers: route.approvers,
          threshold: route.threshold,
          requiredCount: route.requiredCount,
          timeoutMinutes: route.timeoutMinutes
        })),
        complianceRules: complianceRules.map(rule => ({
          id: rule.id,
          name: rule.name,
          description: rule.description,
          category: rule.category,
          severity: rule.severity
        })),
        validationResults: {
          isValid: true,
          passedRules: validationResult.passedRules.map(r => ({
            ruleId: r.ruleId,
            ruleName: r.ruleName,
            severity: r.severity,
            message: r.message
          })),
          failedRules: [],
          warnings: validationResult.warnings.map(r => ({
            ruleId: r.ruleId,
            ruleName: r.ruleName,
            severity: r.severity,
            message: r.message
          })),
          errors: []
        }
      };

    } catch (error) {
      const executionTime = Date.now() - startTime;
      logger.error(`Trade request failed after ${executionTime}ms:`, error);

      return {
        success: false,
        error: error instanceof Error ? error.message : 'Unknown error occurred'
      };
    }
  }

  /**
   * Perform account-specific validation
   */
  private async performAccountSpecificValidation(
    account: Account,
    request: TradeExecutionRequest
  ): Promise<boolean> {
    try {
      // This would contain account-specific validation logic
      // For example:
      // - IRA: Check contribution limits, withdrawal eligibility
      // - Trust: Check fiduciary duty, beneficiary notifications
      // - Personal: Check concentration risk, risk tolerance alignment

      // For now, return true (validation passed)
      return true;

    } catch (error) {
      logger.error('Account-specific validation failed:', error);
      return false;
    }
  }

  /**
   * Validate account creation/update
   */
  async validateAccount(account: Account): Promise<{
    isValid: boolean;
    errors: string[];
    warnings: string[];
  }> {
    return await account.validate();
  }

  /**
   * Get account approval chain for amount
   */
  async getAccountApprovalChain(accountId: string, amount: number): Promise<Array<{
    level: number;
    approvers: string[];
    threshold: number;
    requiredCount: number;
    timeoutMinutes: number;
  }>> {
    try {
      const account = await this.entityManager.loadEntity(accountId) as Account;
      if (!account) {
        throw new Error(`Account ${accountId} not found`);
      }

      const chain = account.getApprovalChain(amount);
      return chain.map(route => ({
        level: route.level,
        approvers: route.approvers,
        threshold: route.threshold,
        requiredCount: route.requiredCount,
        timeoutMinutes: route.timeoutMinutes
      }));

    } catch (error) {
      logger.error('Failed to get approval chain:', error);
      throw error;
    }
  }

  /**
   * Get account compliance rules
   */
  async getAccountComplianceRules(accountId: string): Promise<Array<{
    id: string;
    name: string;
    description: string;
    category: string;
    severity: string;
  }>> {
    try {
      const account = await this.entityManager.loadEntity(accountId) as Account;
      if (!account) {
        throw new Error(`Account ${accountId} not found`);
      }

      const rules = account.getComplianceRules();
      return rules.map(rule => ({
        id: rule.id,
        name: rule.name,
        description: rule.description,
        category: rule.category,
        severity: rule.severity
      }));

    } catch (error) {
      logger.error('Failed to get compliance rules:', error);
      throw error;
      throw error;
    }
  }
}