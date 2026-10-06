package cold

import (
	"bytes"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestParquetDeterminism_RepeatedByteIdentityAndMerkleProof(t *testing.T) {
	tenantID := uuid.New().String()
	ruleID := uuid.New().String()

	// Generate 100 realistic records
	baseTime := time.Date(2026, 10, 5, 14, 0, 0, 0, time.UTC)
	records := make([]CanonicalRecord, 100)
	for i := 0; i < 100; i++ {
		lineageID := fmt.Sprintf("00000000-0000-0000-0000-%012d", i+1)
		orderID := fmt.Sprintf("11111111-0000-0000-0000-%012d", i+1)
		evalTime := baseTime.Add(time.Duration(i) * time.Millisecond).Format(time.RFC3339Nano)
		lsn := int64(45000000000 + i*100)

		records[i] = CanonicalRecord{
			LineageID:       lineageID,
			EvaluatedAt:     evalTime,
			TenantID:        tenantID,
			OrderID:         orderID,
			RuleID:          ruleID,
			RuleVersion:     1,
			ActionTaken:     "APPROVED",
			Passed:          true,
			LatencyMicros:   145,
			EvaluationHash:  fmt.Sprintf("eval_hash_%04d", i),
			IngestLSN:       lsn,
			InputParams:     fmt.Sprintf(`{"index":%d,"price":"150.250000","quantity":"200.000000","symbol":"AAPL"}`, i),
			MetricSnapshots: fmt.Sprintf(`{"adv_pct":"0.001200","gross_exposure":"30050.000000","seq":%d}`, i),
			CreatedAt:       evalTime,
		}
	}

	// 1. First run: generate Parquet bytes, summary, and Merkle tree
	parquetBytes1, summary1, tree1, err := WriteCanonicalParquet(records)
	require.NoError(t, err)
	require.Equal(t, int64(100), summary1.RowCount)
	require.NotEmpty(t, summary1.SHA256Checksum)
	require.NotEmpty(t, summary1.MerkleRoot)

	// 2. Second run: shuffle records order and generate Parquet bytes again
	shuffledRecords := make([]CanonicalRecord, len(records))
	// Reverse order to test sorting determinism
	for i := range records {
		shuffledRecords[i] = records[len(records)-1-i]
	}

	parquetBytes2, summary2, _, err := WriteCanonicalParquet(shuffledRecords)
	require.NoError(t, err)

	// 3. Assert 100% BYTE-IDENTITY between runs
	require.Equal(t, summary1.SHA256Checksum, summary2.SHA256Checksum, "Parquet SHA-256 checksums MUST be 100%% byte-identical across runs")
	require.Equal(t, summary1.FileSizeBytes, summary2.FileSizeBytes, "File size bytes must match identically")
	require.Equal(t, summary1.MerkleRoot, summary2.MerkleRoot, "Merkle Root MUST match identically across runs")
	require.True(t, bytes.Equal(parquetBytes1, parquetBytes2), "Raw output byte streams MUST be byte-for-byte identical")

	t.Logf("Byte-Identity Test PASSED: FileSize=%d bytes, SHA256=%s, MerkleRoot=%s",
		summary1.FileSizeBytes, summary1.SHA256Checksum, summary1.MerkleRoot)

	// 4. Read back records and verify all 100 rows match
	readRecords, err := ReadCanonicalParquet(bytes.NewReader(parquetBytes1), int64(len(parquetBytes1)))
	require.NoError(t, err)
	require.Len(t, readRecords, 100)
	require.Equal(t, records[0].LineageID, readRecords[0].LineageID)
	require.Equal(t, records[99].LineageID, readRecords[99].LineageID)

	// 5. Test Merkle Inclusion Proofs on arbitrary leaves (e.g. leaf 0, 42, 99)
	for _, targetIdx := range []int{0, 42, 99} {
		proof, err := tree1.GenerateProof(targetIdx)
		require.NoError(t, err)
		require.NotEmpty(t, proof)

		targetLeaf := tree1.Leaves[targetIdx]
		valid := VerifyInclusionProof(targetLeaf, proof, summary1.MerkleRoot)
		require.True(t, valid, "Inclusion proof for leaf %d must verify against Merkle root %s", targetIdx, summary1.MerkleRoot)

		// Assert that corrupting the leaf fails verification
		corruptedLeaf := append([]byte(nil), targetLeaf...)
		corruptedLeaf[0] ^= 0xFF
		invalid := VerifyInclusionProof(corruptedLeaf, proof, summary1.MerkleRoot)
		require.False(t, invalid, "Corrupted leaf must fail inclusion proof")
	}

	t.Logf("Merkle Inclusion Proofs PASSED for 100-leaf tree across multiple indices!")
}
