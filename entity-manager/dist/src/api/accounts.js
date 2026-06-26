import express from 'express';
import { EntityManager } from '../services/EntityManager.js';
import { UnifiedValidator } from '../services/UnifiedValidator.js';
import { PersonalAccount, IRAAccount, TrustAccount } from '../entities/index.js';
import { logger } from '../utils/logger.js';
const router = express.Router();
const entityManager = EntityManager.getInstance();
const validator = UnifiedValidator.getInstance();
router.post('/personal', async (req, res) => {
    try {
        const { id, tenantId, datasourceId, accountNumber, name, ownerId, custodianId, riskTolerance, investmentObjective, netWorth } = req.body;
        const account = new PersonalAccount(id, tenantId, datasourceId, accountNumber, name, ownerId, custodianId, riskTolerance, investmentObjective, 'pending', netWorth);
        const validation = await validator.validateAccount(account);
        if (!validation.isValid) {
            return res.status(400).json({
                error: 'Validation failed',
                details: validation
            });
        }
        await entityManager.saveEntity(account);
        return res.status(201).json({
            success: true,
            account: account.toJSON()
        });
    }
    catch (error) {
        logger.error('Failed to create personal account:', error);
        return res.status(500).json({
            error: 'Failed to create account',
            message: error instanceof Error ? error.message : 'Unknown error'
        });
    }
});
router.post('/ira', async (req, res) => {
    try {
        const { id, tenantId, datasourceId, accountNumber, name, ownerId, custodianId, iraType, ownerAge, contributionLimit } = req.body;
        const account = new IRAAccount(id, tenantId, datasourceId, accountNumber, name, ownerId, custodianId, iraType, ownerAge, contributionLimit);
        const validation = await validator.validateAccount(account);
        if (!validation.isValid) {
            return res.status(400).json({
                error: 'Validation failed',
                details: validation
            });
        }
        await entityManager.saveEntity(account);
        return res.status(201).json({
            success: true,
            account: account.toJSON()
        });
    }
    catch (error) {
        logger.error('Failed to create IRA account:', error);
        return res.status(500).json({
            error: 'Failed to create account',
            message: error instanceof Error ? error.message : 'Unknown error'
        });
    }
});
router.post('/trust', async (req, res) => {
    try {
        const { id, tenantId, datasourceId, accountNumber, name, ownerId, custodianId, trustType, trusteeId, beneficiaries } = req.body;
        const account = new TrustAccount(id, tenantId, datasourceId, accountNumber, name, ownerId, custodianId, trustType, trusteeId, beneficiaries || []);
        const validation = await validator.validateAccount(account);
        if (!validation.isValid) {
            return res.status(400).json({
                error: 'Validation failed',
                details: validation
            });
        }
        await entityManager.saveEntity(account);
        return res.status(201).json({
            success: true,
            account: account.toJSON()
        });
    }
    catch (error) {
        logger.error('Failed to create trust account:', error);
        return res.status(500).json({
            error: 'Failed to create account',
            message: error instanceof Error ? error.message : 'Unknown error'
        });
    }
});
router.get('/:id', async (req, res) => {
    try {
        const { id } = req.params;
        const account = await entityManager.loadEntity(id);
        if (!account) {
            return res.status(404).json({
                error: 'Account not found'
            });
        }
        return res.json({
            success: true,
            account: account.toJSON()
        });
    }
    catch (error) {
        logger.error('Failed to get account:', error);
        return res.status(500).json({
            error: 'Failed to get account',
            message: error instanceof Error ? error.message : 'Unknown error'
        });
    }
});
router.get('/:id/compliance', async (req, res) => {
    try {
        const { id } = req.params;
        const rules = await validator.getAccountComplianceRules(id);
        res.json({
            success: true,
            rules
        });
    }
    catch (error) {
        logger.error('Failed to get compliance rules:', error);
        res.status(500).json({
            error: 'Failed to get compliance rules',
            message: error instanceof Error ? error.message : 'Unknown error'
        });
    }
});
router.get('/:id/approval-chain', async (req, res) => {
    try {
        const { id } = req.params;
        const { amount } = req.query;
        if (!amount || isNaN(Number(amount))) {
            return res.status(400).json({
                error: 'Amount parameter is required and must be a number'
            });
        }
        const chain = await validator.getAccountApprovalChain(id, Number(amount));
        return res.json({
            success: true,
            approvalChain: chain
        });
    }
    catch (error) {
        logger.error('Failed to get approval chain:', error);
        return res.status(500).json({
            error: 'Failed to get approval chain',
            message: error instanceof Error ? error.message : 'Unknown error'
        });
    }
});
router.put('/:id', async (req, res) => {
    try {
        const { id } = req.params;
        const account = await entityManager.loadEntity(id);
        if (!account) {
            return res.status(404).json({
                error: 'Account not found'
            });
        }
        Object.assign(account, req.body);
        account.markAsUpdated();
        const validation = await validator.validateAccount(account);
        if (!validation.isValid) {
            return res.status(400).json({
                error: 'Validation failed',
                details: validation
            });
        }
        await entityManager.saveEntity(account);
        return res.json({
            success: true,
            account: account.toJSON()
        });
    }
    catch (error) {
        logger.error('Failed to update account:', error);
        return res.status(500).json({
            error: 'Failed to update account',
            message: error instanceof Error ? error.message : 'Unknown error'
        });
    }
});
router.delete('/:id', async (req, res) => {
    try {
        const { id } = req.params;
        await entityManager.deleteEntity(id);
        return res.json({
            success: true,
            message: 'Account deleted successfully'
        });
    }
    catch (error) {
        logger.error('Failed to delete account:', error);
        return res.status(500).json({
            error: 'Failed to delete account',
            message: error instanceof Error ? error.message : 'Unknown error'
        });
    }
});
export { router as accountRoutes };
//# sourceMappingURL=accounts.js.map