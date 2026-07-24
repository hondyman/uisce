package genui

import (
	"context"
	"fmt"
	"time"

	"go.temporal.io/sdk/client"
)

// ApprovalItem represents a pending approval in the inbox
type ApprovalItem struct {
	WorkflowID   string                 `json:"workflow_id"`
	RunID        string                 `json:"run_id"`
	TenantID     string                 `json:"tenant_id"`
	PortfolioID  string                 `json:"portfolio_id"`
	ProposalID   string                 `json:"proposal_id"`
	Stage        string                 `json:"stage"`
	SLA          string                 `json:"sla"`
	SLAStatus    string                 `json:"sla_status"` // ok, warning, overdue
	Risk         string                 `json:"risk"`
	RiskColor    string                 `json:"risk_color"`
	Proposal     map[string]interface{} `json:"proposal"`
	CreatedAt    time.Time              `json:"created_at"`
	DriftReport  map[string]interface{} `json:"drift_report"`
	PolicyResult map[string]interface{} `json:"policy_result"`
}

// ApprovalsService handles approval-related operations
type ApprovalsService struct {
	temporalClient client.Client
}

// NewApprovalsService creates a new approvals service
func NewApprovalsService(temporalClient client.Client) *ApprovalsService {
	return &ApprovalsService{
		temporalClient: temporalClient,
	}
}

// GetPendingApprovals retrieves all workflows waiting for approval signals
func (s *ApprovalsService) GetPendingApprovals(ctx context.Context, tenantID string) ([]ApprovalItem, error) {
	// In a real implementation, this would query Temporal's workflow search API
	// For now, return mock data that matches our demo workflow
	
	// TODO: Implement actual Temporal query:
	// query := fmt.Sprintf("WorkflowType = 'RebalanceWorkflow' AND ExecutionStatus = 'Running' AND TenantID = '%s'", tenantID)
	// resp, err := s.temporalClient.ListWorkflow(ctx, &workflowservice.ListWorkflowExecutionsRequest{
	// 	Query: query,
	// })

	approvals := []ApprovalItem{
		{
			WorkflowID:  "rebalancer-demo-123456",
			RunID:       "run-abc-123",
			TenantID:    tenantID,
			PortfolioID: "demo_portfolio",
			ProposalID:  "proposal-001",
			Stage:       "Advisor Approval",
			SLA:         "Due in: 1h 45m",
			SLAStatus:   "warning",
			Risk:        "Medium",
			RiskColor:   "orange",
			CreatedAt:   time.Now().Add(-15 * time.Minute),
			Proposal: map[string]interface{}{
				"id": "proposal-001",
				"trades": []map[string]interface{}{
					{"side": "SELL", "symbol": "IVV", "qty": 50},
					{"side": "BUY", "symbol": "VXUS", "qty": 75},
				},
				"explanation": "Rebalancing to target allocation: reduce US equity overweight, increase international exposure",
				"confidence":  0.85,
			},
			DriftReport: map[string]interface{}{
				"has_drift": true,
				"drift_pct": 6.2,
				"positions": []map[string]interface{}{
					{"asset_class": "US_EQUITY", "current_weight": 45.0, "target_weight": 40.0, "drift": 5.0},
					{"asset_class": "INTL_EQUITY", "current_weight": 18.0, "target_weight": 25.0, "drift": 7.0},
				},
			},
			PolicyResult: map[string]interface{}{
				"ok":      true,
				"reasons": []string{},
			},
		},
	}

	return approvals, nil
}

// SendApprovalSignal sends an approval/rejection signal to a workflow
func (s *ApprovalsService) SendApprovalSignal(ctx context.Context, workflowID, runID string, approved bool, rationale string) error {
	signal := map[string]interface{}{
		"approved":   approved,
		"advisor_id": "advisor-001", // TODO: Get from JWT claims
		"rationale":  rationale,
		"time":       time.Now(),
	}

	err := s.temporalClient.SignalWorkflow(ctx, workflowID, runID, "AdvisorApproval", signal)
	if err != nil {
		return fmt.Errorf("failed to send approval signal: %w", err)
	}

	return nil
}

// GetApprovalDetail retrieves detailed information about a specific approval
func (s *ApprovalsService) GetApprovalDetail(ctx context.Context, workflowID string) (*ApprovalItem, error) {
	// In production, query workflow state via Temporal query API
	// For now, return mock data
	approvals, err := s.GetPendingApprovals(ctx, "demo_tenant")
	if err != nil {
		return nil, err
	}

	for _, approval := range approvals {
		if approval.WorkflowID == workflowID {
			return &approval, nil
		}
	}

	return nil, fmt.Errorf("approval not found: %s", workflowID)
}
