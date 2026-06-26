package workflows

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"github.com/hondyman/semlayer/backend/pkg/llm"
)

type RWAActivities struct {
	ConfigService *llm.LLMConfigService
}

type MintTokenOutput struct {
	TokenID         string `json:"tokenId"`
	TransactionHash string `json:"transactionHash"`
	AuditReport     string `json:"auditReport"`
	AuditPassed     bool   `json:"auditPassed"`
	ContractAddress string `json:"contractAddress"`
}

// ActivityMintToken simulates minting an RWA token on a blockchain
// It uses GenAI to "audit" the smart contract parameters before minting.
func (a *RWAActivities) ActivityMintToken(ctx context.Context, config map[string]interface{}, state map[string]interface{}) (*MintTokenOutput, error) {
	assetName, _ := state["assetName"].(string)
	metadata, _ := state["metadata"].(string)

	// Default fallback
	if assetName == "" {
		assetName = "Unknown Asset"
	}

	// 1. GenAI Smart Contract Audit
	auditReport := "Audit Skipped"
	auditPassed := true

	cfg, err := a.ConfigService.Get()
	if err == nil {
		provider := llm.NewGeminiProvider(cfg.APIKey, cfg.Model)
		prompt := fmt.Sprintf(`Perform a security audit for a smart contract minting token "%s".
		Metadata: %s.
		Return a brief 1-sentence summary of potential vulnerabilities. 
		If safe, say "SAFE".`, assetName, metadata)

		resp, err := provider.GenerateResponse(ctx, prompt)
		if err == nil {
			auditReport = resp
			// excessive simple check for demo
			if len(resp) > 200 { // If long explanation, maybe issues?
				// Just a mock logic
			}
		}
	}

	// 2. Mock Minting Process
	// In production: ethClient.Transaction(...)
	time.Sleep(500 * time.Millisecond) // Simulate easy chain latency

	txHash := fmt.Sprintf("0x%x", rand.Int63())
	contractAddr := fmt.Sprintf("0x%x", rand.Int63())

	return &MintTokenOutput{
		TokenID:         fmt.Sprintf("TOKEN-%d", rand.Intn(9999)),
		TransactionHash: txHash,
		AuditReport:     auditReport,
		AuditPassed:     auditPassed,
		ContractAddress: contractAddr,
	}, nil
}

// ActivityPerformKYC simulates an identity check
func (a *RWAActivities) ActivityPerformKYC(ctx context.Context, config map[string]interface{}, state map[string]interface{}) (map[string]interface{}, error) {
	investorID, _ := state["investorId"].(string)

	// Mock KYC Check
	status := "APPROVED"
	if investorID == "BANNED_ENTITY" {
		status = "REJECTED"
	}

	return map[string]interface{}{
		"kycStatus":  status,
		"verifiedAt": time.Now().Format(time.RFC3339),
		"riskLevel":  "LOW",
	}, nil
}

// ActivityDistributeDividends calculates and distributes payments
func (a *RWAActivities) ActivityDistributeDividends(ctx context.Context, config map[string]interface{}, state map[string]interface{}) (map[string]interface{}, error) {
	amount, _ := state["dividendAmount"].(float64)
	holders, _ := state["holderCount"].(float64)

	if holders == 0 {
		holders = 1
	}

	perShare := amount / holders

	return map[string]interface{}{
		"totalDistributed": amount,
		"perShare":         perShare,
		"recipientCount":   int(holders),
		"status":           "COMPLETED",
	}, nil
}
