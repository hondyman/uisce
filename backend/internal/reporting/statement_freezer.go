package reporting

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type StatementFreezerService struct {
	db        *sqlx.DB
	secretKey []byte
}

func NewStatementFreezerService(db *sqlx.DB, hmacSecret string) *StatementFreezerService {
	if hmacSecret == "" {
		hmacSecret = "uisce-sec-17a4-default-hmac-key-production"
	}
	return &StatementFreezerService{
		db:        db,
		secretKey: []byte(hmacSecret),
	}
}

// FreezeAndSealStatement writes the immutable audit passport and seals the artifact
func (s *StatementFreezerService) FreezeAndSealStatement(
	ctx context.Context,
	tenantID uuid.UUID,
	statementID string,
	effectiveDate time.Time,
	signerIdentity string,
	rawArrowBatch []byte,
	astPlanJSON []byte,
	pdfBinary []byte,
	objectStoreURI string,
) (*StatementAuditPassport, error) {
	if tenantID == uuid.Nil {
		return nil, fmt.Errorf("Rule 7 violation: tenant_id cannot be nil")
	}

	// 1. Compute Cryptographic SHA-256 Digests
	dataHash := sha256.Sum256(rawArrowBatch)
	astHash := sha256.Sum256(astPlanJSON)
	pdfHash := sha256.Sum256(pdfBinary)

	dataHashHex := hex.EncodeToString(dataHash[:])
	astHashHex := hex.EncodeToString(astHash[:])
	pdfHashHex := hex.EncodeToString(pdfHash[:])

	// 2. Fetch Previous Passport Hash in Chain for SEC Rule 17a-4 Immutability
	var prevHash string
	if s.db != nil {
		_ = s.db.GetContext(ctx, &prevHash, `
			SELECT merkle_passport_hash 
			FROM reporting.statement_snapshot_vault
			WHERE tenant_id = $1 
			ORDER BY sealed_at DESC 
			LIMIT 1;
		`, tenantID)
	}
	if prevHash == "" {
		prevHash = "0000000000000000000000000000000000000000000000000000000000000000"
	}

	now := time.Now().UTC()

	// 3. Generate Merkle HMAC-SHA256 Passport
	mac := hmac.New(sha256.New, s.secretKey)
	macPayload := fmt.Sprintf("%s|%s|%s|%s|%s|%s|%s",
		tenantID.String(),
		statementID,
		dataHashHex,
		astHashHex,
		pdfHashHex,
		effectiveDate.Format(time.RFC3339),
		prevHash,
	)
	mac.Write([]byte(macPayload))
	passportHashHex := hex.EncodeToString(mac.Sum(nil))

	passport := &StatementAuditPassport{
		PassportID:           uuid.New(),
		TenantID:             tenantID,
		StatementID:          statementID,
		EffectiveDate:        effectiveDate,
		KnowledgeDate:        now,
		DataVectorSHA256:     dataHashHex,
		ASTPlanChecksum:      astHashHex,
		PDFArtifactSHA256:    pdfHashHex,
		MerklePassportHash:   passportHashHex,
		PreviousPassportHash: prevHash,
		SignerIdentity:       signerIdentity,
		ObjectStoreURI:       objectStoreURI,
		Status:               SnapshotStatusSealed,
		SealedAt:             now,
	}

	// 4. Persist to Relational Vault Ledger
	if s.db != nil {
		query := `
			INSERT INTO reporting.statement_snapshot_vault (
				passport_id, tenant_id, statement_id, effective_date, knowledge_date,
				data_vector_sha256, ast_plan_checksum, pdf_artifact_sha256,
				merkle_passport_hash, previous_passport_hash, signer_identity,
				object_store_uri, status, sealed_at
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14);
		`
		_, err := s.db.ExecContext(ctx, query,
			passport.PassportID, passport.TenantID, passport.StatementID,
			passport.EffectiveDate, passport.KnowledgeDate,
			passport.DataVectorSHA256, passport.ASTPlanChecksum, passport.PDFArtifactSHA256,
			passport.MerklePassportHash, passport.PreviousPassportHash, passport.SignerIdentity,
			passport.ObjectStoreURI, passport.Status, passport.SealedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed persisting statement audit passport: %w", err)
		}
	}

	return passport, nil
}

// VerifyStatementIntegrity checks if an on-disk or stored PDF binary has been tampered with
func (s *StatementFreezerService) VerifyStatementIntegrity(
	passport StatementAuditPassport,
	pdfBinary []byte,
) (bool, string) {
	currentPDFHash := sha256.Sum256(pdfBinary)
	currentPDFHashHex := hex.EncodeToString(currentPDFHash[:])

	if currentPDFHashHex != passport.PDFArtifactSHA256 {
		return false, fmt.Sprintf("TAMPER DETECTED: Binary hash %s does not match passport %s", currentPDFHashHex, passport.PDFArtifactSHA256)
	}

	// Recompute Merkle Hash
	mac := hmac.New(sha256.New, s.secretKey)
	macPayload := fmt.Sprintf("%s|%s|%s|%s|%s|%s|%s",
		passport.TenantID.String(),
		passport.StatementID,
		passport.DataVectorSHA256,
		passport.ASTPlanChecksum,
		passport.PDFArtifactSHA256,
		passport.EffectiveDate.Format(time.RFC3339),
		passport.PreviousPassportHash,
	)
	mac.Write([]byte(macPayload))
	recomputedPassport := hex.EncodeToString(mac.Sum(nil))

	if recomputedPassport != passport.MerklePassportHash {
		return false, "TAMPER DETECTED: Merkle chain signature validation failure"
	}

	return true, "PASSPORT_VERIFIED_AUTHENTIC"
}
