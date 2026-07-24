package onboarding

import (
	"context"
	"fmt"
	"math/rand"
	"time"
)

// RunKYC simulates a KYC check
func RunKYC(ctx context.Context, clientID string) (*KYCResult, error) {
	// Mock Logic: Randomly flag some users
	rand.Seed(time.Now().UnixNano())
	r := rand.Intn(100)

	if r < 10 {
		return &KYCResult{Status: "REJECTED", Reason: "Sanctions List Match"}, nil
	} else if r < 30 {
		return &KYCResult{Status: "FLAGGED", Reason: "PEP Match"}, nil
	}

	return &KYCResult{Status: "APPROVED"}, nil
}

// SendRejectionEmail simulates sending an email
func SendRejectionEmail(ctx context.Context, clientID string) error {
	fmt.Printf("📧 Sending Rejection Email to %s\n", clientID)
	return nil
}

// GenerateAndSendDocuSign simulates DocuSign integration
func GenerateAndSendDocuSign(ctx context.Context, clientID string) (string, error) {
	envelopeID := fmt.Sprintf("env-%d", time.Now().Unix())
	fmt.Printf("📝 Generated DocuSign Envelope: %s for %s\n", envelopeID, clientID)
	return envelopeID, nil
}

// OpenCustodianAccount simulates account provisioning
func OpenCustodianAccount(ctx context.Context, clientID string) error {
	fmt.Printf("🏦 Provisioning Custodian Account for %s\n", clientID)
	return nil
}
