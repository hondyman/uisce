package iceberg

import (
	"context"
	"os"
	"path/filepath"
)

// LocalStorage implements StorageBackend for local filesystem
type LocalStorage struct {
	BaseDir string
}

func NewLocalStorage(baseDir string) *LocalStorage {
	return &LocalStorage{BaseDir: baseDir}
}

func (s *LocalStorage) ReadFile(ctx context.Context, path string) ([]byte, error) {
	fullPath := filepath.Join(s.BaseDir, path)
	return os.ReadFile(fullPath)
}

func (s *LocalStorage) WriteFile(ctx context.Context, path string, data []byte) error {
	fullPath := filepath.Join(s.BaseDir, path)
	
	// Ensure directory exists
	dir := filepath.Dir(fullPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	return os.WriteFile(fullPath, data, 0644)
}
