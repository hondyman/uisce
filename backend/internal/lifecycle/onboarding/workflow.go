package onboarding

import (
	"time"

	"go.temporal.io/sdk/workflow"
)

// OnboardingRequest is the input to the workflow
type OnboardingRequest struct {
	ClientID  string
	RiskScore int
}

// KYCResult represents the outcome of the KYC check
type KYCResult struct {
	Status string // "APPROVED", "FLAGGED", "REJECTED"
	Reason string
}

// OnboardingWorkflow orchestrates the client onboarding process
func OnboardingWorkflow(ctx workflow.Context, req OnboardingRequest) error {
	ao := workflow.ActivityOptions{
		StartToCloseTimeout: time.Minute * 5,
	}
	ctx = workflow.WithActivityOptions(ctx, ao)
	logger := workflow.GetLogger(ctx)

	logger.Info("Starting Onboarding Workflow", "ClientID", req.ClientID)

	// Step 1: KYC/AML Check (Automated)
	var kycResult KYCResult
	err := workflow.ExecuteActivity(ctx, RunKYC, req.ClientID).Get(ctx, &kycResult)
	if err != nil {
		return err
	}

	// Step 2: Decision Logic (Auto-Approve vs Manual Review)
	if kycResult.Status == "FLAGGED" {
		logger.Info("KYC Flagged, waiting for compliance review", "Reason", kycResult.Reason)
		
		// Wait for human signal
		var approvalSignal string
		signalChan := workflow.GetSignalChannel(ctx, "ComplianceResponse")
		
		// Wait indefinitely (or add a timer for SLA)
		signalChan.Receive(ctx, &approvalSignal)
		
		if approvalSignal == "REJECT" {
			logger.Info("Compliance Rejected Application")
			return workflow.ExecuteActivity(ctx, SendRejectionEmail, req.ClientID).Get(ctx, nil)
		}
		logger.Info("Compliance Approved Application")
	} else if kycResult.Status == "REJECTED" {
		return workflow.ExecuteActivity(ctx, SendRejectionEmail, req.ClientID).Get(ctx, nil)
	}

	// Step 3: Document Generation
	var envelopeID string
	err = workflow.ExecuteActivity(ctx, GenerateAndSendDocuSign, req.ClientID).Get(ctx, &envelopeID)
	if err != nil {
		return err
	}

	// Step 4: Wait for Signature (Webhook Pattern)
	logger.Info("Waiting for DocuSign signature", "EnvelopeID", envelopeID)
	var docStatus string
	docSignalChan := workflow.GetSignalChannel(ctx, "DocuSignUpdate")
	
	for docStatus != "COMPLETED" {
		docSignalChan.Receive(ctx, &docStatus)
		if docStatus == "DECLINED" {
			logger.Warn("Client declined document signing")
			// Logic to resend or cancel could go here
			return nil 
		}
	}

	// Step 5: Provision Account (Custodian API)
	err = workflow.ExecuteActivity(ctx, OpenCustodianAccount, req.ClientID).Get(ctx, nil)
	if err != nil {
		return err
	}

	logger.Info("Onboarding Complete", "ClientID", req.ClientID)
	return nil
}
