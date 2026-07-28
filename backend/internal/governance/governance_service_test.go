package governance_test

import (
	"context"
	"testing"

	"github.com/hondyman/uisce/backend/internal/governance"
	"github.com/stretchr/testify/assert"
)

func TestMakerCheckerService_FourEyesPrinciple(t *testing.T) {
	svc := governance.NewMakerCheckerService()
	ctx := context.Background()

	makerID := "user-maker-123"
	checkerID := "user-checker-456"
	tenantID := "tenant-789"

	// 1. Propose change as Maker
	req, err := svc.ProposeChange(ctx, tenantID, makerID, "Update Fee Rule", "Change HWM formula", `{"hwm_threshold": 0.15}`)
	assert.NoError(t, err)
	assert.Equal(t, governance.StatusPendingApproval, req.Status)

	// 2. Maker attempts self-approval -> must FAIL (Four-Eyes Principle)
	_, err = svc.ApproveChange(ctx, req.ID, makerID)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "FOUR-EYES VIOLATION")

	// 3. Independent Checker approves -> SUCCESS
	approvedReq, err := svc.ApproveChange(ctx, req.ID, checkerID)
	assert.NoError(t, err)
	assert.Equal(t, governance.StatusActive, approvedReq.Status)
	assert.Equal(t, checkerID, approvedReq.CheckerUserID)
}

func TestMakerCheckerService_SubmitAndReviewProposal(t *testing.T) {
	svc := governance.NewMakerCheckerService()

	makerID := "user-maker-123"
	checkerID := "user-checker-456"
	tenantID := "tenant-789"

	// 1. Missing tenant_id -> Rule 7 FATAL
	_, err := svc.SubmitProposal(context.Background(), "bo-100", []byte(`{"rate": 0.02}`), "Update rate")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Rule 7 Violation")

	// 2. Submit with valid context
	ctxMaker := context.WithValue(context.Background(), "tenant_id", tenantID)
	ctxMaker = context.WithValue(ctxMaker, "user_id", makerID)

	proposal, err := svc.SubmitProposal(ctxMaker, "bo-100", []byte(`{"rate": 0.02}`), "Update rate")
	assert.NoError(t, err)
	assert.Equal(t, governance.StatusPendingApproval, proposal.Status)

	// 3. Maker tries to review own proposal -> FATAL GOVERNANCE VIOLATION
	err = svc.ReviewProposal(ctxMaker, proposal.ProposalID, true)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "GOVERNANCE VIOLATION")

	// 4. Independent Checker approves -> SUCCESS
	ctxChecker := context.WithValue(context.Background(), "tenant_id", tenantID)
	ctxChecker = context.WithValue(ctxChecker, "user_id", checkerID)

	err = svc.ReviewProposal(ctxChecker, proposal.ProposalID, true)
	assert.NoError(t, err)
}

