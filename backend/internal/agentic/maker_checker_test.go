package agentic

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMakerCheckerService_SubmitAndReviewProposal(t *testing.T) {
	svc := NewMakerCheckerService(nil)

	// Submit proposal
	ticketID, err := svc.SubmitAgentProposal(context.Background(), ProposalRequest{
		TenantID:   "core",
		AgentID:    "RebalanceAgent-v1",
		TargetBOID: "customers",
		ActionType: "BULK_REBALANCE",
		Payload:    json.RawMessage(`{"portfolio_id": "PORT-991", "rebalance_orders": [{"symbol": "AAPL", "qty": 500}]}`),
	})

	assert.NoError(t, err)
	assert.NotEmpty(t, ticketID)

	// Approve ticket
	errApprove := svc.ApproveTicket(context.Background(), ticketID, "pm_john_doe")
	assert.NoError(t, errApprove)

	// Reject ticket
	errReject := svc.RejectTicket(context.Background(), ticketID, "pm_john_doe", "Exceeds risk budget")
	assert.NoError(t, errReject)
}

func TestMakerCheckerService_FourEyesPrinciple(t *testing.T) {
	svc := NewMakerCheckerService(nil)

	tickets, err := svc.ListTickets(context.Background(), "core")
	assert.NoError(t, err)
	assert.NotEmpty(t, tickets)
	assert.True(t, tickets[0].CreatedByAI)
}
