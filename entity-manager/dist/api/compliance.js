import express from 'express';
import { UnifiedValidator } from '../services/UnifiedValidator.js';
import { EntityManager } from '../services/EntityManager.js';
import { Account } from '../entities/Account.js';
import { logger } from '../utils/logger.js';
const router = express.Router();
const validator = UnifiedValidator.getInstance();
const entityManager = EntityManager.getInstance();
router.get('/validate-all', async (req, res) => {
    try {
        const accountIds = ['demo-personal-1', 'demo-ira-1', 'demo-trust-1'];
        const results = [];
        for (const accountId of accountIds) {
            try {
                const account = await entityManager.loadEntity(accountId);
                if (account && account instanceof Account) {
                    const complianceRules = await validator.getAccountComplianceRules(accountId);
                    const isValid = account.isActive();
                    results.push({
                        accountId,
                        accountName: account.name,
                        accountType: account.accountType,
                        isValid,
                        complianceRules,
                        status: account.status
                    });
                }
            }
            catch (error) {
                logger.warn(`Failed to validate account ${accountId}:`, error);
                results.push({
                    accountId,
                    error: 'Failed to validate account'
                });
            }
        }
        const allValid = results.every(r => r.isValid);
        res.json({
            success: true,
            overallCompliance: allValid,
            totalAccounts: results.length,
            compliantAccounts: results.filter(r => r.isValid).length,
            results
        });
    }
    catch (error) {
        logger.error('Failed to validate all accounts:', error);
        res.status(500).json({
            error: 'Failed to validate accounts',
            message: error instanceof Error ? error.message : 'Unknown error'
        });
    }
});
router.get('/account/:accountId', async (req, res) => {
    try {
        const { accountId } = req.params;
        const account = await entityManager.loadEntity(accountId);
        if (!account) {
            return res.status(404).json({
                error: 'Account not found'
            });
        }
        if (!(account instanceof Account)) {
            return res.status(400).json({
                error: 'Entity is not an account'
            });
        }
        const complianceRules = await validator.getAccountComplianceRules(accountId);
        return res.json({
            success: true,
            accountId,
            accountName: account.name,
            accountType: account.accountType,
            status: account.status,
            isActive: account.isActive(),
            complianceRules
        });
    }
    catch (error) {
        logger.error('Failed to get account compliance:', error);
        return res.status(500).json({
            error: 'Failed to get account compliance',
            message: error instanceof Error ? error.message : 'Unknown error'
        });
    }
});
export { router as complianceRoutes };
//# sourceMappingURL=compliance.js.map