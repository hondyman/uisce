import express from 'express';
import { logger } from '../utils/logger.js';

// Import route handlers (will be created)
import { accountRoutes } from './accounts.js';
import { tradeRoutes } from './trades.js';
import { approvalRoutes } from './approvals.js';
import { demoRoutes } from './demo.js';
import { complianceRoutes } from './compliance.js';
import { internalRoutes } from './internal.js';

export function setupRoutes(app: express.Application): void {
  // API versioning
  const apiRouter = express.Router();

  // Mount route handlers
  apiRouter.use('/accounts', accountRoutes);
  apiRouter.use('/trades', tradeRoutes);
  apiRouter.use('/approvals', approvalRoutes);
  apiRouter.use('/demo', demoRoutes);
  apiRouter.use('/compliance', complianceRoutes);
  apiRouter.use('/internal', internalRoutes);

  // API info endpoint
  apiRouter.get('/', (req, res) => {
    res.json({
      name: 'Entity Manager API',
      version: '1.0.0',
      description: 'Production-grade entity management system',
      endpoints: {
        accounts: '/api/accounts',
        trades: '/api/trades',
        approvals: '/api/approvals',
        demo: '/api/demo',
        compliance: '/api/compliance',
        internal: '/api/internal'
      }
    });
  });

  // Mount API router
  app.use('/api', apiRouter);

  logger.info('✅ API routes configured');
}