//go:build !no_parquet

package iceberg

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"time"

	"github.com/segmentio/parquet-go"
)

// StorageBackend interface allows mocking S3/Local filesystem interactions
type StorageBackend interface {
	ReadFile(ctx context.Context, path string) ([]byte, error)
	WriteFile(ctx context.Context, path string, data []byte) error
}

// TaxLotWriter handles the Copy-on-Write logic for Iceberg tables
type TaxLotWriter struct {
	storage StorageBackend
}

func NewTaxLotWriter(storage StorageBackend) *TaxLotWriter {
	return &TaxLotWriter{storage: storage}
}

// UpdateLot performs a Copy-on-Write update on a specific Parquet file.
// It reads the file, applies the mutation function to matching lots, and writes a NEW file.
// It returns the path of the new file and its metrics (needed for the Iceberg Manifest).
func (w *TaxLotWriter) UpdateLot(ctx context.Context, sourcePath string, mutation func(*TaxLot) bool) (string, *FileMetrics, error) {
	// 1. Read source file
	data, err := w.storage.ReadFile(ctx, sourcePath)
	if err != nil {
		return "", nil, fmt.Errorf("failed to read source file: %w", err)
	}

	reader := bytes.NewReader(data)
	parquetReader := parquet.NewGenericReader[TaxLot](reader)

	// 2. Prepare writer for new file
	var buf bytes.Buffer
	writer := parquet.NewWriter(&buf)

	// 3. Iterate and apply mutation
	rows := make([]TaxLot, 100) // Batch size
	metrics := &FileMetrics{
		RecordCount: 0,
		LowerBounds: make(map[string]interface{}),
		UpperBounds: make(map[string]interface{}),
	}

	for {
		n, err := parquetReader.Read(rows)
		if n > 0 {
			for i := 0; i < n; i++ {
				lot := rows[i]

				// Apply mutation if needed
				// The mutation function returns true if the row was modified
				_ = mutation(&lot)

				// Write to new file
				if err := writer.Write(&lot); err != nil {
					return "", nil, fmt.Errorf("failed to write row: %w", err)
				}

				// Update metrics (simplified for MVP)
				metrics.RecordCount++
			}
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", nil, fmt.Errorf("failed to read parquet rows: %w", err)
		}
	}

	if err := writer.Close(); err != nil {
		return "", nil, fmt.Errorf("failed to close parquet writer: %w", err)
	}

	// 4. Write new file to storage
	// Generate a new unique path (e.g., using UUID or timestamp)
	newPath := fmt.Sprintf("data/tax_lots/%d_%s.parquet", time.Now().UnixNano(), "updated")
	if err := w.storage.WriteFile(ctx, newPath, buf.Bytes()); err != nil {
		return "", nil, fmt.Errorf("failed to write new file: %w", err)
	}

	metrics.FileSize = int64(buf.Len())

	return newPath, metrics, nil
}

// FileMetrics stores metadata required for the Iceberg Manifest
type FileMetrics struct {
	RecordCount int64
	FileSize    int64
	LowerBounds map[string]interface{}
	UpperBounds map[string]interface{}
}
