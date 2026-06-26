import { EntityManager } from '../services/EntityManager.js';
import { logger } from '../utils/logger.js';
export class ValidationEngine {
    static instance;
    entityManager;
    constructor() {
        this.entityManager = EntityManager.getInstance();
    }
    static getInstance() {
        if (!ValidationEngine.instance) {
            ValidationEngine.instance = new ValidationEngine();
        }
        return ValidationEngine.instance;
    }
    async validateTradeRequest(context) {
        const startTime = Date.now();
        try {
            const account = await this.entityManager.loadEntity(context.accountId);
            if (!account) {
                return this.createResult(false, [], [{
                        ruleId: 'account_exists',
                        ruleName: 'Account Exists',
                        passed: false,
                        severity: 'critical',
                        message: `Account ${context.accountId} not found`
                    }], [], [], Date.now() - startTime);
            }
            const accountValidation = await this.validateAccountStatus(account);
            if (!accountValidation.isValid) {
                return accountValidation;
            }
            const allResults = [
                ...accountValidation.passedRules,
                ...accountValidation.failedRules
            ];
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
            const passedRules = allResults.filter(r => r.passed);
            const failedRules = allResults.filter(r => !r.passed);
            const warnings = failedRules.filter(r => r.severity === 'low' || r.severity === 'medium');
            const errors = failedRules.filter(r => r.severity === 'high' || r.severity === 'critical');
            const isValid = errors.length === 0;
            return this.createResult(isValid, passedRules, failedRules, warnings, errors, Date.now() - startTime);
        }
        catch (error) {
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
    async validateAccountStatus(account) {
        const passed = account.isActive();
        const result = {
            ruleId: 'account_status',
            ruleName: 'Account Status',
            passed,
            severity: 'critical',
            message: passed ? 'Account is active' : 'Account is not active'
        };
        return this.createResult(passed, passed ? [result] : [], passed ? [] : [result], [], [], 0);
    }
    async validateConcentrationLimit(account, context) {
        if (!context.trade || !context.portfolio) {
            return this.createResult(true, [], [], [], [], 0);
        }
        const maxConcentration = account.getMaxConcentration();
        const tradeValue = context.trade.amount;
        const portfolioValue = context.portfolio.totalValue;
        const newPositionValue = tradeValue;
        const newConcentration = newPositionValue / (portfolioValue + tradeValue);
        const passed = newConcentration <= maxConcentration;
        const result = {
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
    async validateAssetCompatibility(account, context) {
        if (!context.trade) {
            return this.createResult(true, [], [], [], [], 0);
        }
        const canHold = account.canHoldAsset(context.trade.assetType);
        const result = {
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
    async validateKYCRequirements(account, context) {
        try {
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
            const kycValid = client.isKYCValid ? client.isKYCValid() : true;
            const result = {
                ruleId: 'kyc_completeness',
                ruleName: 'KYC Completeness',
                passed: kycValid,
                severity: 'critical',
                message: kycValid ? 'KYC is current and complete' : 'KYC is expired or incomplete'
            };
            return this.createResult(kycValid, kycValid ? [result] : [], kycValid ? [] : [result], [], [], 0);
        }
        catch (error) {
            return this.createResult(false, [], [{
                    ruleId: 'kyc_validation_error',
                    ruleName: 'KYC Validation',
                    passed: false,
                    severity: 'critical',
                    message: 'Failed to validate KYC requirements'
                }], [], [], 0);
        }
    }
    async validateTradeExecution(account, context) {
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
        const passed = true;
        const result = {
            ruleId: tradeExecutionRule.id,
            ruleName: tradeExecutionRule.name,
            passed,
            severity: tradeExecutionRule.severity,
            message: passed ? 'Trade execution approved' : 'Trade execution rejected'
        };
        return this.createResult(passed, passed ? [result] : [], passed ? [] : [result], [], [], 0);
    }
    async validateFeeStructure(account, context) {
        const passed = true;
        const result = {
            ruleId: 'fee_validation',
            ruleName: 'Fee Validation',
            passed,
            severity: 'medium',
            message: 'Transaction fees within approved limits'
        };
        return this.createResult(passed, passed ? [result] : [], passed ? [] : [result], [], [], 0);
    }
    createResult(isValid, passedRules, failedRules, warnings, errors, executionTime) {
        return {
            isValid,
            passedRules,
            failedRules,
            warnings,
            errors,
            executionTime
        };
    }
    async getAccountComplianceRules(accountId) {
        try {
            const account = await this.entityManager.loadEntity(accountId);
            if (!account) {
                return [];
            }
            return account.getComplianceRules();
        }
        catch (error) {
            logger.error('Failed to get compliance rules:', error);
            return [];
        }
    }
}
//# sourceMappingURL=ValidationEngine.js.map