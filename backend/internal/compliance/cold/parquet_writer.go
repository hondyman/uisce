package cold

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"

	"github.com/parquet-go/parquet-go"
)

// ParquetArchiveSummary encapsulates the cryptographic and physical metadata of an assembled cold archive slice
type ParquetArchiveSummary struct {
	RowCount       int64  `json:"row_count"`
	FileSizeBytes  int64  `json:"file_size_bytes"`
	SHA256Checksum string `json:"sha256_checksum"`
	MerkleRoot     string `json:"merkle_root"`
	StartLSN       int64  `json:"start_lsn"`
	EndLSN         int64  `json:"end_lsn"`
}

// WriteCanonicalParquet writes records into a deterministic Parquet byte stream and calculates Merkle root & SHA-256
func WriteCanonicalParquet(records []CanonicalRecord) ([]byte, *ParquetArchiveSummary, *MerkleTree, error) {
	if len(records) == 0 {
		return nil, nil, nil, fmt.Errorf("cannot write empty records slice")
	}

	// 1. Deterministic sort by IngestLSN ASC, LineageID ASC
	sortedRecords := make([]CanonicalRecord, len(records))
	copy(sortedRecords, records)
	SortEvaluationRecords(sortedRecords)

	// 2. Generate canonical Merkle leaves & build Merkle tree
	leaves := make([][]byte, len(sortedRecords))
	for i, r := range sortedRecords {
		leaf, err := CanonicalEvaluationLeaf(
			r.LineageID, r.TenantID, r.RuleID, int(r.RuleVersion),
			r.ActionTaken, r.IngestLSN,
			json.RawMessage(r.InputParams), json.RawMessage(r.MetricSnapshots),
		)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("compute leaf for record %d: %w", i, err)
		}
		leaves[i] = leaf
	}

	merkleTree, merkleRoot := BuildMerkleTree(leaves)

	// 3. Write Parquet stream deterministically using parquet-go
	var buf bytes.Buffer
	writer := parquet.NewGenericWriter[CanonicalRecord](
		&buf,
		parquet.Compression(&parquet.Snappy),
		parquet.PageBufferSize(64*1024),
	)

	for _, rec := range sortedRecords {
		if _, err := writer.Write([]CanonicalRecord{rec}); err != nil {
			return nil, nil, nil, fmt.Errorf("write parquet record: %w", err)
		}
	}

	if err := writer.Close(); err != nil {
		return nil, nil, nil, fmt.Errorf("close parquet writer: %w", err)
	}

	parquetBytes := buf.Bytes()
	fileChecksum := sha256.Sum256(parquetBytes)
	checksumHex := hex.EncodeToString(fileChecksum[:])

	startLSN := sortedRecords[0].IngestLSN
	endLSN := sortedRecords[len(sortedRecords)-1].IngestLSN

	summary := &ParquetArchiveSummary{
		RowCount:       int64(len(sortedRecords)),
		FileSizeBytes:  int64(len(parquetBytes)),
		SHA256Checksum: checksumHex,
		MerkleRoot:     merkleRoot,
		StartLSN:       startLSN,
		EndLSN:         endLSN,
	}

	return parquetBytes, summary, merkleTree, nil
}

// ReadCanonicalParquet reads all CanonicalRecord rows from a Parquet reader
func ReadCanonicalParquet(r io.ReaderAt, size int64) ([]CanonicalRecord, error) {
	file, err := parquet.OpenFile(r, size)
	if err != nil {
		return nil, fmt.Errorf("open parquet file: %w", err)
	}

	reader := parquet.NewGenericReader[CanonicalRecord](file)
	defer reader.Close()

	records := make([]CanonicalRecord, file.NumRows())
	n, err := reader.Read(records)
	if err != nil && err != io.EOF {
		return nil, fmt.Errorf("read parquet records: %w", err)
	}

	return records[:n], nil
}
