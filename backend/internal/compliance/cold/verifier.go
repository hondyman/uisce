package cold

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"
)

// VerificationReport contains the audit certificate results of an independent cryptographic verification run
type VerificationReport struct {
	TenantID        string    `json:"tenant_id"`
	S3Bucket        string    `json:"s3_bucket"`
	S3Key           string    `json:"s3_key"`
	StartLSN        int64     `json:"start_lsn"`
	EndLSN          int64     `json:"end_lsn"`
	RecordCount     int64     `json:"record_count"`
	FileSizeBytes   int64     `json:"file_size_bytes"`
	SHA256Checksum  string    `json:"sha256_checksum"`
	ManifestRoot    string    `json:"manifest_root"`
	ComputedRoot    string    `json:"computed_root"`
	RootMatch       bool      `json:"root_match"`
	InclusionProofs bool      `json:"inclusion_proofs_verified"`
	VerifiedAt      time.Time `json:"verified_at"`
	AuditStatus     string    `json:"audit_status"`
}

// ArchiveVerifier provides pure cryptographic verification of sealed Parquet slices against Merkle roots
type ArchiveVerifier struct{}

// NewArchiveVerifier creates a new ArchiveVerifier
func NewArchiveVerifier() *ArchiveVerifier {
	return &ArchiveVerifier{}
}

// VerifyParquetSlice independently reads raw Parquet bytes and recomputes the Merkle Root from scratch
func (v *ArchiveVerifier) VerifyParquetSlice(ctx context.Context, parquetBytes []byte, manifestRoot string) (*VerificationReport, error) {
	if len(parquetBytes) == 0 {
		return nil, fmt.Errorf("cannot verify empty parquet bytes")
	}

	checksum := sha256.Sum256(parquetBytes)
	checksumHex := hex.EncodeToString(checksum[:])

	// 1. Read records from raw Parquet
	records, err := ReadCanonicalParquet(bytes.NewReader(parquetBytes), int64(len(parquetBytes)))
	if err != nil {
		return nil, fmt.Errorf("read canonical parquet bytes: %w", err)
	}

	if len(records) == 0 {
		return nil, fmt.Errorf("parquet file contains 0 records")
	}

	// 2. Sort records deterministically (IngestLSN ASC, LineageID ASC)
	SortEvaluationRecords(records)

	// 3. Compute leaves
	leaves := make([][]byte, len(records))
	for i, r := range records {
		leaf, err := CanonicalEvaluationLeaf(
			r.LineageID, r.TenantID, r.RuleID, int(r.RuleVersion), r.RuleContentHash,
			r.ActionTaken, r.IngestLSN,
			json.RawMessage(r.InputParams), json.RawMessage(r.MetricSnapshots),
		)
		if err != nil {
			return nil, fmt.Errorf("compute leaf for record %d (%s): %w", i, r.LineageID, err)
		}
		leaves[i] = leaf
	}

	// 4. Build Merkle tree and extract root
	tree, computedRoot := BuildMerkleTree(leaves)

	rootMatches := computedRoot == manifestRoot

	// 5. Verify sample inclusion proofs
	proofsValid := true
	sampleIndices := []int{0, len(records) / 2, len(records) - 1}
	for _, idx := range sampleIndices {
		proof, err := tree.GenerateProof(idx)
		if err != nil || !VerifyInclusionProof(tree.Leaves[idx], proof, computedRoot) {
			proofsValid = false
			break
		}
	}

	status := "VERIFIED_TAMPER_EVIDENT_MATCH"
	if !rootMatches || !proofsValid {
		status = "CRYPTOGRAPHIC_MISMATCH_TAMPERED"
	}

	tenantID := ""
	if len(records) > 0 {
		tenantID = records[0].TenantID
	}

	return &VerificationReport{
		TenantID:        tenantID,
		StartLSN:        records[0].IngestLSN,
		EndLSN:          records[len(records)-1].IngestLSN,
		RecordCount:     int64(len(records)),
		FileSizeBytes:   int64(len(parquetBytes)),
		SHA256Checksum:  checksumHex,
		ManifestRoot:    manifestRoot,
		ComputedRoot:    computedRoot,
		RootMatch:       rootMatches,
		InclusionProofs: proofsValid,
		VerifiedAt:      time.Now().UTC(),
		AuditStatus:     status,
	}, nil
}

// VerifyRuleRegistrySlice independently reads companion rule registry Parquet bytes and recomputes the Merkle Root
func (v *ArchiveVerifier) VerifyRuleRegistrySlice(ctx context.Context, parquetBytes []byte, manifestRoot string) (*VerificationReport, error) {
	if len(parquetBytes) == 0 {
		return nil, fmt.Errorf("cannot verify empty rule registry parquet bytes")
	}

	checksum := sha256.Sum256(parquetBytes)
	checksumHex := hex.EncodeToString(checksum[:])

	// 1. Read records from raw Parquet
	records, err := ReadCanonicalRuleRegistryParquet(bytes.NewReader(parquetBytes), int64(len(parquetBytes)))
	if err != nil {
		return nil, fmt.Errorf("read canonical rule registry parquet bytes: %w", err)
	}

	if len(records) == 0 {
		return nil, fmt.Errorf("rule registry parquet file contains 0 records")
	}

	// 2. Sort records deterministically (RuleID ASC, Version ASC)
	SortRuleRegistryRecords(records)

	// 3. Compute leaves
	leaves := make([][]byte, len(records))
	for i, r := range records {
		leaf := CanonicalRuleLeaf(r.RuleID, r.Version, r.ContentHash, r.CompiledBytecodeHash)
		leaves[i] = leaf
	}

	// 4. Build Merkle tree and extract root
	tree, computedRoot := BuildMerkleTree(leaves)

	rootMatches := computedRoot == manifestRoot

	// 5. Verify sample inclusion proofs
	proofsValid := true
	sampleIndices := []int{0, len(records) / 2, len(records) - 1}
	for _, idx := range sampleIndices {
		proof, err := tree.GenerateProof(idx)
		if err != nil || !VerifyInclusionProof(tree.Leaves[idx], proof, computedRoot) {
			proofsValid = false
			break
		}
	}

	status := "VERIFIED_TAMPER_EVIDENT_MATCH"
	if !rootMatches || !proofsValid {
		status = "CRYPTOGRAPHIC_MISMATCH_TAMPERED"
	}

	tenantID := ""
	if len(records) > 0 {
		tenantID = records[0].TenantID
	}

	return &VerificationReport{
		TenantID:        tenantID,
		RecordCount:     int64(len(records)),
		FileSizeBytes:   int64(len(parquetBytes)),
		SHA256Checksum:  checksumHex,
		ManifestRoot:    manifestRoot,
		ComputedRoot:    computedRoot,
		RootMatch:       rootMatches,
		InclusionProofs: proofsValid,
		VerifiedAt:      time.Now().UTC(),
		AuditStatus:     status,
	}, nil
}
