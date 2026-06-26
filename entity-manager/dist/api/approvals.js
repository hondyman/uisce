import express from 'express';
import { ApprovalWorkflowEngine } from '../approval/ApprovalWorkflowEngine.js';
import { logger } from '../utils/logger.js';
const router = express.Router();
const approvalEngine = ApprovalWorkflowEngine.getInstance();
router.get('/:workflowId', async (req, res) => {
    try {
        const { workflowId } = req.params;
        const status = await approvalEngine.getWorkflowStatus(workflowId);
        res.json({
            success: true,
            workflowId,
            status
        });
    }
    catch (error) {
        logger.error('Failed to get workflow status:', error);
        res.status(500).json({
            error: 'Failed to get workflow status',
            message: error instanceof Error ? error.message : 'Unknown error'
        });
    }
});
router.post('/:workflowId/decisions', async (req, res) => {
    try {
        const { workflowId } = req.params;
        const { decision, approverId, comments } = req.body;
        if (!decision || !approverId) {
            return res.status(400).json({
                error: 'Missing required fields: decision, approverId'
            });
        }
        const approvalDecision = {
            approvalId: `${workflowId}-${Date.now()}`,
            approverId,
            decision,
            comments,
            timestamp: new Date()
        };
        await approvalEngine.submitDecision(workflowId, approvalDecision);
        return res.json({
            success: true,
            message: `Decision ${decision} submitted for workflow ${workflowId}`
        });
    }
    catch (error) {
        logger.error('Failed to submit decision:', error);
        return res.status(500).json({
            error: 'Failed to submit decision',
            message: error instanceof Error ? error.message : 'Unknown error'
        });
    }
});
router.post('/:workflowId/cancel', async (req, res) => {
    try {
        const { workflowId } = req.params;
        const { reason } = req.body;
        await approvalEngine.cancelWorkflow(workflowId, reason || 'Cancelled by user');
        res.json({
            success: true,
            message: `Workflow ${workflowId} cancelled`
        });
    }
    catch (error) {
        logger.error('Failed to cancel workflow:', error);
        res.status(500).json({
            error: 'Failed to cancel workflow',
            message: error instanceof Error ? error.message : 'Unknown error'
        });
    }
});
router.post('/:workflowId/escalate', async (req, res) => {
    try {
        const { workflowId } = req.params;
        const { reason } = req.body;
        await approvalEngine.escalateWorkflow(workflowId, reason || 'Escalated by user');
        res.json({
            success: true,
            message: `Workflow ${workflowId} escalated`
        });
    }
    catch (error) {
        logger.error('Failed to escalate workflow:', error);
        res.status(500).json({
            error: 'Failed to escalate workflow',
            message: error instanceof Error ? error.message : 'Unknown error'
        });
    }
});
router.get('/pending/:userId', async (req, res) => {
    try {
        const { userId } = req.params;
        const pendingApprovals = await approvalEngine.getPendingApprovals(userId);
        res.json({
            success: true,
            pendingApprovals
        });
    }
    catch (error) {
        logger.error('Failed to get pending approvals:', error);
        res.status(500).json({
            error: 'Failed to get pending approvals',
            message: error instanceof Error ? error.message : 'Unknown error'
        });
    }
});
export { router as approvalRoutes };
//# sourceMappingURL=approvals.js.map