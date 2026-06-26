import express from 'express';
import { EntityManager } from '../services/EntityManager.js';
import { UnifiedValidator } from '../services/UnifiedValidator.js';
import { PersonalAccount, IRAAccount, TrustAccount, RiskTolerance, InvestmentObjective, IRAType, TrustType } from '../entities/index.js';
import { logger } from '../utils/logger.js';

const router = express.Router();
const entityManager = EntityManager.getInstance();
const validator = UnifiedValidator.getInstance();

// Create sample accounts for demo
router.post('/create-sample-accounts', async (req, res) => {
  try {
    const tenantId = 'demo-tenant';
    const datasourceId = 'demo-datasource';

    // Create Personal Account
    const personalAccount = new PersonalAccount(
      'demo-personal-1',
      tenantId,
      datasourceId,
      'PA-001',
      'Demo Personal Account',
      'demo-client-1',
      'demo-custodian-1',
      'high' as any,
      'growth' as any,
      'active' as any,
      1000000
    );

    // Create IRA Account
    const iraAccount = new IRAAccount(
      'demo-ira-1',
      tenantId,
      datasourceId,
      'IRA-001',
      'Demo Traditional IRA',
      'demo-client-1',
      'demo-custodian-1',
      'traditional' as any,
      45,
      7000,
      'active' as any
    );

    // Create Trust Account
    const trustAccount = new TrustAccount(
      'demo-trust-1',
      tenantId,
      datasourceId,
      'TRUST-001',
      'Demo Revocable Trust',
      'demo-client-1',
      'demo-custodian-1',
      'revocable' as any,
      'demo-trustee-1',
      [
        { id: 'ben-1', name: 'Beneficiary 1', relationship: 'child', percentage: 50 },
        { id: 'ben-2', name: 'Beneficiary 2', relationship: 'spouse', percentage: 50 }
      ],
      'active' as any
    );

    // Save accounts
    await entityManager.saveEntity(personalAccount);
    await entityManager.saveEntity(iraAccount);
    await entityManager.saveEntity(trustAccount);

    res.json({
      success: true,
      message: 'Sample accounts created successfully',
      accounts: [
        personalAccount.toJSON(),
        iraAccount.toJSON(),
        trustAccount.toJSON()
      ]
    });

  } catch (error) {
    logger.error('Failed to create sample accounts:', error);
    res.status(500).json({
      error: 'Failed to create sample accounts',
      message: error instanceof Error ? error.message : 'Unknown error'
    });
  }
});

// Get account policies (compliance rules and approval chains)
router.get('/account-policies', async (req, res) => {
  try {
    const accounts = [
      { id: 'demo-personal-1', name: 'Personal Account' },
      { id: 'demo-ira-1', name: 'IRA Account' },
      { id: 'demo-trust-1', name: 'Trust Account' }
    ];

    const policies: Array<{
      accountId: string;
      accountName: string;
      complianceRules: any[];
      approvalChains: {
        lowAmount: any;
        highAmount: any;
      };
    }> = [];

    for (const account of accounts) {
      try {
        const complianceRules = await validator.getAccountComplianceRules(account.id);
        const approvalChainLow = await validator.getAccountApprovalChain(account.id, 10000);
        const approvalChainHigh = await validator.getAccountApprovalChain(account.id, 100000);

        policies.push({
          accountId: account.id,
          accountName: account.name,
          complianceRules,
          approvalChains: {
            lowAmount: approvalChainLow,
            highAmount: approvalChainHigh
          }
        });
      } catch (error) {
        logger.warn(`Failed to get policies for account ${account.id}:`, error);
      }
    }

    res.json({
      success: true,
      policies
    });

  } catch (error) {
    logger.error('Failed to get account policies:', error);
    res.status(500).json({
      error: 'Failed to get account policies',
      message: error instanceof Error ? error.message : 'Unknown error'
    });
  }
});

// Validate demo trade
router.post('/validate-trade', async (req, res) => {
  try {
    const demoRequest = {
      accountId: 'demo-personal-1',
      trade: {
        ticker: 'AAPL',
        quantity: 100,
        price: 200,
        assetType: 'equity',
        amount: 20000
      },
      portfolio: {
        totalValue: 500000,
        cash: 100000,
        positions: [
          { ticker: 'MSFT', quantity: 50, value: 15000, percentage: 0.03 },
          { ticker: 'GOOGL', quantity: 20, value: 30000, percentage: 0.06 }
        ]
      },
      advisorId: 'demo-advisor-1',
      tenantId: 'demo-tenant',
      datasourceId: 'demo-datasource'
    };

    const result = await validator.processTradeRequest(demoRequest);

    res.json({
      success: result.success,
      validation: result.validationResults,
      approvalRequired: !!result.workflowId,
      approvalChain: result.approvalChain,
      complianceRules: result.complianceRules,
      error: result.error
    });

  } catch (error) {
    logger.error('Demo trade validation failed:', error);
    res.status(500).json({
      error: 'Demo trade validation failed',
      message: error instanceof Error ? error.message : 'Unknown error'
    });
  }
});

export { router as demoRoutes };