import express from 'express';
import { UnifiedValidator } from '../services/UnifiedValidator.js';
import { logger } from '../utils/logger.js';
const router = express.Router();
const validator = UnifiedValidator.getInstance();
router.post('/validate', async (req, res) => {
    try {
        const request = req.body;
        if (!request.accountId || !request.trade || !request.portfolio) {
            return res.status(400).json({
                error: 'Missing required fields: accountId, trade, portfolio'
            });
        }
        const result = await validator.processTradeRequest(request);
        if (result.success) {
            return res.json({
                valid: true,
                approvalRequired: !!result.workflowId,
                approvalChain: result.approvalChain,
                complianceRules: result.complianceRules,
                validationResults: result.validationResults
            });
        }
        else {
            return res.status(400).json({
                valid: false,
                error: result.error,
                validationResults: result.validationResults
            });
        }
    }
    catch (error) {
        logger.error('Trade validation failed:', error);
        return res.status(500).json({
            error: 'Trade validation failed',
            message: error instanceof Error ? error.message : 'Unknown error'
        });
    }
});
router.post('/execute', async (req, res) => {
    try {
        const request = req.body;
        if (!request.accountId || !request.trade || !request.portfolio || !request.advisorId) {
            return res.status(400).json({
                error: 'Missing required fields: accountId, trade, portfolio, advisorId'
            });
        }
        const result = await validator.processTradeRequest(request);
        if (result.success) {
            return res.json({
                success: true,
                workflowId: result.workflowId,
                approvalChain: result.approvalChain,
                complianceRules: result.complianceRules,
                validationResults: result.validationResults,
                message: result.workflowId
                    ? 'Trade submitted for approval'
                    : 'Trade executed successfully'
            });
        }
        else {
            return res.status(400).json({
                success: false,
                error: result.error,
                validationResults: result.validationResults
            });
        }
    }
    catch (error) {
        logger.error('Trade execution failed:', error);
        return res.status(500).json({
            error: 'Trade execution failed',
            message: error instanceof Error ? error.message : 'Unknown error'
        });
    }
});
export { router as tradeRoutes };
//# sourceMappingURL=trades.js.map