package services

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/hondyman/semlayer/backend/internal/optimizer"
	"github.com/hondyman/semlayer/backend/internal/temporal/activities"
)

// EvidenceBundle contains all evidence for regulatory compliance.
type EvidenceBundle struct {
	BundleID        string                             `json:"bundle_id"`
	Proposal        optimizer.Plan                     `json:"proposal"`
	WorkflowHistory []WorkflowEvent                    `json:"workflow_history"`
	Snapshots       SnapshotRefs                       `json:"snapshots"`
	AdvisorDecision optimizer.AdvisorDecision          `json:"advisor_decision"`
	SagaExecution   *activities.ExecuteTradeSagaOutput `json:"saga_execution,omitempty"`
	Signatures      BundleSignatures                   `json:"signatures"`
	PolicyVersions  PolicyVersions                     `json:"policy_versions"`
	GeneratedAt     time.Time                          `json:"generated_at"`
}

// WorkflowEvent represents a step in the workflow history.
type WorkflowEvent struct {
	Event     string    `json:"event"`
	Timestamp time.Time `json:"timestamp"`
	Details   string    `json:"details,omitempty"`
}

// SnapshotRefs contains references to data snapshots used.
type SnapshotRefs struct {
	Positions      string             `json:"positions"`
	FactorUniverse string             `json:"factor_universe"`
	TaxRules       string             `json:"tax_rules"`
	MonteCarlo     MonteCarloSnapshot `json:"monte_carlo"`
}

// MonteCarloSnapshot contains simulation parameters for reproducibility.
type MonteCarloSnapshot struct {
	Seed int64 `json:"seed"`
	Runs int   `json:"runs"`
}

// BundleSignatures contains cryptographic signatures.
type BundleSignatures struct {
	BundleHash   string `json:"bundle_hash"`
	KMSSignature string `json:"kms_signature,omitempty"`
}

// PolicyVersions tracks which policy versions were used.
type PolicyVersions struct {
	Rego string `json:"rego"`
	CEL  string `json:"cel"`
}

// UARRecord is a User Action Record for compliance tracking.
type UARRecord struct {
	ID          string                 `json:"id"`
	TenantID    string                 `json:"tenant_id"`
	PortfolioID string                 `json:"portfolio_id"`
	ProposalID  string                 `json:"proposal_id"`
	Event       string                 `json:"event"`
	Data        map[string]interface{} `json:"data"`
	CreatedAt   time.Time              `json:"created_at"`
}

// UARReader reads UAR records.
type UARReader interface {
	ReadByPortfolio(ctx context.Context, tenantID, portfolioID string) ([]UARRecord, error)
	ReadByProposal(ctx context.Context, tenantID, proposalID string) ([]UARRecord, error)
}

// KMSClient signs bundles.
type KMSClient interface {
	Sign(ctx context.Context, data []byte) (string, error)
}

// EvidenceBundleBuilder assembles compliance evidence bundles.
// Named differently to avoid conflict with stub in stubs.go
type EvidenceBundleBuilder struct {
	uarReader UARReader
	kmsClient KMSClient
}

// NewEvidenceBundleBuilder creates a new builder.
func NewEvidenceBundleBuilder(uarReader UARReader, kmsClient KMSClient) *EvidenceBundleBuilder {
	return &EvidenceBundleBuilder{
		uarReader: uarReader,
		kmsClient: kmsClient,
	}
}

// AssembleBundle creates a complete evidence bundle for a proposal.
func (s *EvidenceBundleBuilder) AssembleBundle(ctx context.Context, tenantID, portfolioID, proposalID string) (*EvidenceBundle, error) {
	// Read all UAR records for this proposal
	records, err := s.uarReader.ReadByProposal(ctx, tenantID, proposalID)
	if err != nil {
		return nil, err
	}

	bundle := &EvidenceBundle{
		BundleID:    "evidence_" + uuid.New().String()[:8],
		GeneratedAt: time.Now().UTC(),
	}

	// Build workflow history from UAR records
	for _, record := range records {
		bundle.WorkflowHistory = append(bundle.WorkflowHistory, WorkflowEvent{
			Event:     record.Event,
			Timestamp: record.CreatedAt,
			Details:   extractDetails(record.Data),
		})

		// Extract specific data based on event type
		switch record.Event {
		case "ProposalGenerated":
			if plan, ok := record.Data["plan"].(map[string]interface{}); ok {
				planJSON, _ := json.Marshal(plan)
				json.Unmarshal(planJSON, &bundle.Proposal)
			}
		case "AdvisorDecision":
			if decision, ok := record.Data["decision"].(map[string]interface{}); ok {
				decisionJSON, _ := json.Marshal(decision)
				json.Unmarshal(decisionJSON, &bundle.AdvisorDecision)
			}
		case "SagaCompleted":
			if saga, ok := record.Data["saga"].(map[string]interface{}); ok {
				sagaJSON, _ := json.Marshal(saga)
				var sagaOutput activities.ExecuteTradeSagaOutput
				json.Unmarshal(sagaJSON, &sagaOutput)
				bundle.SagaExecution = &sagaOutput
			}
		}
	}

	return bundle, nil
}

// extractDetails converts data map to string for display.
func extractDetails(data map[string]interface{}) string {
	if data == nil {
		return ""
	}
	bytes, _ := json.Marshal(data)
	return string(bytes)
}

// SignBundle adds cryptographic signatures to the bundle.
func (s *EvidenceBundleBuilder) SignBundle(ctx context.Context, bundle *EvidenceBundle) error {
	// Compute hash of bundle content
	bundleJSON, err := json.Marshal(bundle)
	if err != nil {
		return err
	}

	hash := sha256.Sum256(bundleJSON)
	bundle.Signatures.BundleHash = hex.EncodeToString(hash[:])

	// Sign with KMS if available
	if s.kmsClient != nil {
		signature, err := s.kmsClient.Sign(ctx, hash[:])
		if err != nil {
			// Log but don't fail - KMS might not be configured
			return nil
		}
		bundle.Signatures.KMSSignature = signature
	}

	return nil
}

// ExportJSON exports the bundle as JSON.
func (s *EvidenceBundleBuilder) ExportJSON(bundle *EvidenceBundle) ([]byte, error) {
	return json.MarshalIndent(bundle, "", "  ")
}

// ExportPDFPlaceholder returns a placeholder for PDF export.
// In production, this would use a PDF generation library.
func (s *EvidenceBundleBuilder) ExportPDFPlaceholder(bundle *EvidenceBundle) ([]byte, error) {
	// Return JSON wrapped in a comment indicating PDF would be generated
	content := map[string]interface{}{
		"note":   "PDF generation placeholder - integrate wkhtmltopdf or similar",
		"bundle": bundle,
	}
	return json.MarshalIndent(content, "", "  ")
}
