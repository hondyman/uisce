package charts

import (
	"bytes"
	"compress/gzip"
	"fmt"
)

// compressData compresses a byte slice using gzip
func compressData(data []byte) ([]byte, error) {
	var b bytes.Buffer
	gw := gzip.NewWriter(&b)
	if _, err := gw.Write(data); err != nil {
		return nil, fmt.Errorf("failed to write compressed data: %w", err)
	}
	if err := gw.Close(); err != nil {
		return nil, fmt.Errorf("failed to close gzip writer: %w", err)
	}
	return b.Bytes(), nil
}

// decompressData decompresses a gzipped byte slice
func decompressData(compressedData []byte) ([]byte, error) {
	gr, err := gzip.NewReader(bytes.NewReader(compressedData))
	if err != nil {
		return nil, fmt.Errorf("failed to create gzip reader: %w", err)
	}
	defer gr.Close()

	var b bytes.Buffer
	if _, err := b.ReadFrom(gr); err != nil {
		return nil, fmt.Errorf("failed to read decompressed data: %w", err)
	}
	return b.Bytes(), nil
}
