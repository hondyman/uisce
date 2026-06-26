import { EntityManager } from '../services/EntityManager.js';
import { ValidationEngine } from '../validation/ValidationEngine.js';
import { ApprovalWorkflowEngine } from '../approval/ApprovalWorkflowEngine.js';
import { logger } from '../utils/logger.js';
export class UnifiedValidator {
    static instance;
    entityManager;
    validationEngine;
    approvalEngine;
    constructor() {
        this.entityManager = EntityManager.getInstance();
        this.validationEngine = ValidationEngine.getInstance();
        this.approvalEngine = ApprovalWorkflowEngine.getInstance();
    }
    static getInstance() {
        if (!UnifiedValidator.instance) {
            UnifiedValidator.instance = new UnifiedValidator();
        }
        return UnifiedValidator.instance;
    }
    async processTradeRequest(request) {
        const startTime = Date.now();
        try {
            logger.info(`Processing trade request for account ${request.accountId}`);
            const account = await this.entityManager.loadEntity(request.accountId);
            if (!account) {
                return {
                    success: false,
                    error: `Account ${request.accountId} not found`
                };
            }
            if (!account.isActive()) {
                return {
                    success: false,
                    error: `Account ${request.accountId} is not active`
                };
            }
            const validationContext = {
                accountId: request.accountId,
                trade: {
                    ticker: request.trade.ticker,
                    quantity: request.trade.quantity,
                    price: request.trade.price,
                    amount: request.trade.amount,
                    assetType: request.trade.assetType
                },
                portfolio: request.portfolio,
                userId: request.advisorId,
                tenantId: request.tenantId,
                datasourceId: request.datasourceId
            };
            const validationResult = await this.validationEngine.validateTradeRequest(validationContext);
            const accountValidationPassed = await this.performAccountSpecificValidation(account, request);
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
            const approvalRequest = {
                id: `trade-${Date.now()}-${Math.random().toString(36).substr(2, 9)}`,
                accountId: request.accountId,
                tradeId: '',
                amount: request.trade.amount,
                description: `Trade: ${request.trade.quantity} ${request.trade.ticker} @ $${request.trade.price}`,
                requesterId: request.advisorId,
                tenantId: request.tenantId,
                datasourceId: request.datasourceId,
                createdAt: new Date()
            };
            const approvalResult = await this.approvalEngine.startApprovalWorkflow(approvalRequest);
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
        }
        catch (error) {
            const executionTime = Date.now() - startTime;
            logger.error(`Trade request failed after ${executionTime}ms:`, error);
            return {
                success: false,
                error: error instanceof Error ? error.message : 'Unknown error occurred'
            };
        }
    }
    async performAccountSpecificValidation(account, request) {
        try {
            return true;
        }
        catch (error) {
            logger.error('Account-specific validation failed:', error);
            return false;
        }
    }
    async validateAccount(account) {
        return await account.validate();
    }
    async getAccountApprovalChain(accountId, amount) {
        try {
            const account = await this.entityManager.loadEntity(accountId);
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
        }
        catch (error) {
            logger.error('Failed to get approval chain:', error);
            throw error;
        }
    }
    async getAccountComplianceRules(accountId) {
        try {
            const account = await this.entityManager.loadEntity(accountId);
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
        }
        catch (error) {
            logger.error('Failed to get compliance rules:', error);
            throw error;
            throw error;
        }
    }
}
//# sourceMappingURL=UnifiedValidator.js.map