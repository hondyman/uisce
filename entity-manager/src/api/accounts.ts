import express from 'express';
import { EntityManager } from '../services/EntityManager.js';
import { UnifiedValidator } from '../services/UnifiedValidator.js';
import { PersonalAccount, IRAAccount, TrustAccount, RiskTolerance, InvestmentObjective, IRAType, TrustType } from '../entities/index.js';
import type { AccountStatus } from '../entities/index.js';
import { logger } from '../utils/logger.js';
import { getClaims } from '../../libs/jwt-middleware-node.js';

const router = express.Router();
const entityManager = EntityManager.getInstance();
const validator = UnifiedValidator.getInstance();

// Create Personal Account
router.post('/personal', async (req, res) => {
  try {
    // Get JWT claims for tenant isolation
    const claims = getClaims(req);
    if (!claims) {
      return res.status(401).json({ error: 'Unauthorized' });
    }

    const {
      id,
      datasourceId,
      accountNumber,
      name,
      ownerId,
      custodianId,
      riskTolerance,
      investmentObjective,
      netWorth
    } = req.body;

    // Use tenant_id from JWT claims, not from request body
    const tenantId = claims.tenant_id;

    const account = new PersonalAccount(
      id,
      tenantId,
      datasourceId,
      accountNumber,
      name,
      ownerId,
      custodianId,
      riskTolerance as RiskTolerance,
      investmentObjective as InvestmentObjective,
      'pending' as any,
      netWorth
    );

    // Validate account
    const validation = await validator.validateAccount(account);
    if (!validation.isValid) {
      return res.status(400).json({
        error: 'Validation failed',
        details: validation
      });
    }

    // Save account
    await entityManager.saveEntity(account);

    return res.status(201).json({
      success: true,
      account: account.toJSON()
    });

  } catch (error) {
    logger.error('Failed to create personal account:', error);
    return res.status(500).json({
      error: 'Failed to create account',
      message: error instanceof Error ? error.message : 'Unknown error'
    });
  }
});

// Create IRA Account
router.post('/ira', async (req, res) => {
  try {
    // Get JWT claims for tenant isolation
    const claims = getClaims(req);
    if (!claims) {
      return res.status(401).json({ error: 'Unauthorized' });
    }

    const {
      id,
      datasourceId,
      accountNumber,
      name,
      ownerId,
      custodianId,
      iraType,
      ownerAge,
      contributionLimit
    } = req.body;

    // Use tenant_id from JWT claims, not from request body
    const tenantId = claims.tenant_id;

    const account = new IRAAccount(
      id,
      tenantId,
      datasourceId,
      accountNumber,
      name,
      ownerId,
      custodianId,
      iraType as IRAType,
      ownerAge,
      contributionLimit
    );

    // Validate account
    const validation = await validator.validateAccount(account);
    if (!validation.isValid) {
      return res.status(400).json({
        error: 'Validation failed',
        details: validation
      });
    }

    // Save account
    await entityManager.saveEntity(account);

    return res.status(201).json({
      success: true,
      account: account.toJSON()
    });

  } catch (error) {
    logger.error('Failed to create IRA account:', error);
    return res.status(500).json({
      error: 'Failed to create account',
      message: error instanceof Error ? error.message : 'Unknown error'
    });
  }
});

// Create Trust Account
router.post('/trust', async (req, res) => {
  try {
    // Get JWT claims for tenant isolation
    const claims = getClaims(req);
    if (!claims) {
      return res.status(401).json({ error: 'Unauthorized' });
    }

    const {
      id,
      datasourceId,
      accountNumber,
      name,
      ownerId,
      custodianId,
      trustType,
      trusteeId,
      beneficiaries
    } = req.body;

    // Use tenant_id from JWT claims, not from request body
    const tenantId = claims.tenant_id;

    const account = new TrustAccount(
      id,
      tenantId,
      datasourceId,
      accountNumber,
      name,
      ownerId,
      custodianId,
      trustType as TrustType,
      trusteeId,
      beneficiaries || []
    );

    // Validate account
    const validation = await validator.validateAccount(account);
    if (!validation.isValid) {
      return res.status(400).json({
        error: 'Validation failed',
        details: validation
      });
    }

    // Save account
    await entityManager.saveEntity(account);

    return res.status(201).json({
      success: true,
      account: account.toJSON()
    });

  } catch (error) {
    logger.error('Failed to create trust account:', error);
    return res.status(500).json({
      error: 'Failed to create account',
      message: error instanceof Error ? error.message : 'Unknown error'
    });
  }
});

// Get account by ID
router.get('/:id', async (req, res) => {
  try {
    // Get JWT claims for tenant isolation
    const claims = getClaims(req);
    if (!claims) {
      return res.status(401).json({ error: 'Unauthorized' });
    }

    const { id } = req.params;
    const account = await entityManager.loadEntity(id);

    // Ensure user can only access their own tenant's data
    if (account && account.tenantId !== claims.tenant_id) {
      return res.status(403).json({
        error: 'Forbidden',
        message: 'You do not have access to this account'
      });
    }    if (!account) {
      return res.status(404).json({
        error: 'Account not found'
      });
    }

    return res.json({
      success: true,
      account: account.toJSON()
    });

  } catch (error) {
    logger.error('Failed to get account:', error);
    return res.status(500).json({
      error: 'Failed to get account',
      message: error instanceof Error ? error.message : 'Unknown error'
    });
  }
});

// Get account compliance rules
router.get('/:id/compliance', async (req, res) => {
  try {
    // Get JWT claims for tenant isolation
    const claims = getClaims(req);
    if (!claims) {
      return res.status(401).json({ error: 'Unauthorized' });
    }

    const { id } = req.params;
    const rules = await validator.getAccountComplianceRules(id);

    res.json({
      success: true,
      rules
    });

  } catch (error) {
    logger.error('Failed to get compliance rules:', error);
    res.status(500).json({
      error: 'Failed to get compliance rules',
      message: error instanceof Error ? error.message : 'Unknown error'
    });
  }
});

// Get account approval chain for amount
router.get('/:id/approval-chain', async (req, res) => {
  try {
    // Get JWT claims for tenant isolation
    const claims = getClaims(req);
    if (!claims) {
      return res.status(401).json({ error: 'Unauthorized' });
    }

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

  } catch (error) {
    logger.error('Failed to get approval chain:', error);
    return res.status(500).json({
      error: 'Failed to get approval chain',
      message: error instanceof Error ? error.message : 'Unknown error'
    });
  }
});

// Update account
router.put('/:id', async (req, res) => {
  try {
    // Get JWT claims for tenant isolation
    const claims = getClaims(req);
    if (!claims) {
      return res.status(401).json({ error: 'Unauthorized' });
    }

    const { id } = req.params;
    const account = await entityManager.loadEntity(id) as any;

    if (!account) {
      return res.status(404).json({
        error: 'Account not found'
      });
    }

    // Ensure user can only access their own tenant's data
    if (account.tenantId !== claims.tenant_id) {
      return res.status(403).json({
        error: 'Forbidden',
        message: 'You do not have access to this account'
      });
    }

    // Update fields (simplified - would need more robust updating logic)
    Object.assign(account, req.body);
    account.markAsUpdated();

    // Validate updated account
    const validation = await validator.validateAccount(account);
    if (!validation.isValid) {
      return res.status(400).json({
        error: 'Validation failed',
        details: validation
      });
    }

    // Save updated account
    await entityManager.saveEntity(account);

    return res.json({
      success: true,
      account: account.toJSON()
    });

  } catch (error) {
    logger.error('Failed to update account:', error);
    return res.status(500).json({
      error: 'Failed to update account',
      message: error instanceof Error ? error.message : 'Unknown error'
    });
  }
});

// Delete account
router.delete('/:id', async (req, res) => {
  try {
    // Get JWT claims for tenant isolation
    const claims = getClaims(req);
    if (!claims) {
      return res.status(401).json({ error: 'Unauthorized' });
    }

    const { id } = req.params;
    const account = await entityManager.loadEntity(id) as any;

    if (!account) {
      return res.status(404).json({
        error: 'Account not found'
      });
    }

    // Ensure user can only access their own tenant's data
    if (account.tenantId !== claims.tenant_id) {
      return res.status(403).json({
        error: 'Forbidden',
        message: 'You do not have access to this account'
      });
    }

    await entityManager.deleteEntity(id);

    return res.json({
      success: true,
      message: 'Account deleted successfully'
    });

  } catch (error) {
    logger.error('Failed to delete account:', error);
    return res.status(500).json({
      error: 'Failed to delete account',
      message: error instanceof Error ? error.message : 'Unknown error'
    });
  }
});

export { router as accountRoutes };