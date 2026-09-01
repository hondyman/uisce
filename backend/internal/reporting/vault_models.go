package reporting

import (
	"time"

	"github.com/google/uuid"
)

type SnapshotStatus string

const (
	SnapshotStatusSealed             SnapshotStatus = "SEALED"
	SnapshotStatusVerified           SnapshotStatus = "VERIFIED"
	SnapshotStatusRestatementPending SnapshotStatus = "RESTATEMENT_PENDING"
	SnapshotStatusSuperseded         SnapshotStatus = "SUPERSEDED"
)

type StatementAuditPassport struct {
	PassportID           uuid.UUID      `json:"passportId"`
	TenantID             uuid.UUID      `json:"tenantId"`
	StatementID          string         `json:"statementId"`
	EffectiveDate        time.Time      `json:"effectiveDate"`       // Te
	KnowledgeDate        time.Time      `json:"knowledgeDate"`       // Tk
	DataVectorSHA256     string         `json:"dataVectorSha256"`
	ASTPlanChecksum      string         `json:"astPlanChecksum"`
	PDFArtifactSHA256    string         `json:"pdfArtifactSha256"`
	MerklePassportHash   string         `json:"merklePassportHash"`
	PreviousPassportHash string         `json:"previousPassportHash"`
	SignerIdentity       string         `json:"signerIdentity"`
	ObjectStoreURI       string         `json:"objectStoreUri"`
	Status               SnapshotStatus `json:"status"`
	SealedAt             time.Time      `json:"sealedAt"`
}

type StatementRestatementDelta struct {
	CorrectionID     uuid.UUID `json:"correctionId"`
	StatementID      string    `json:"statementId"`
	PositionID       string    `json:"positionId"`
	FieldKey         string    `json:"fieldKey"`
	OriginalValue    float64   `json:"originalValue"`
	RestatedValue    float64   `json:"restatedValue"`
	VarianceAmount   float64   `json:"varianceAmount"`
	CorrectionReason string    `json:"correctionReason"`
	DiscoveredAt     time.Time `json:"discoveredAt"`
}

type StatementComparisonReport struct {
	StatementID        string                      `json:"statementId"`
	TenantID           uuid.UUID                   `json:"tenantId"`
	OriginalPassport   StatementAuditPassport      `json:"originalPassport"`
	IsIntegrityValid   bool                        `json:"isIntegrityValid"`
	HasRestatements    bool                        `json:"hasRestatements"`
	TotalOriginalNAV   float64                     `json:"totalOriginalNav"`
	TotalRestatedNAV   float64                     `json:"totalRestatedNav"`
	NetNAVVariance     float64                     `json:"netNavVariance"`
	DivergenceBasisPts float64                     `json:"divergenceBasisPts"`
	Deltas             []StatementRestatementDelta `json:"deltas"`
}
