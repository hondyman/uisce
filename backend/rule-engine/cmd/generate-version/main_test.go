package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// TestGenerateVersionInfo tests the version info generation
func TestGenerateVersionInfo(t *testing.T) {
	// Create temporary directory for output
	tempDir, err := os.MkdirTemp("", "version-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Change to temp directory so the relative outputDir works
	oldDir, _ := os.Getwd()
	os.Chdir(tempDir)
	defer os.Chdir(oldDir)

	// Create the generated directory
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		t.Fatalf("Failed to create generated dir: %v", err)
	}

	// Generate version info by calling main()
	main()

	// Check if file was created
	versionFile := filepath.Join(tempDir, outputDir, outputVersion)
	if _, err := os.Stat(versionFile); os.IsNotExist(err) {
		t.Fatalf("Version file was not created: %s", versionFile)
	}

	// Read and parse the generated file
	data, err := os.ReadFile(versionFile)
	if err != nil {
		t.Fatalf("Failed to read version file: %v", err)
	}

	var versionInfo VersionInfo
	if err := json.Unmarshal(data, &versionInfo); err != nil {
		t.Fatalf("Failed to parse version JSON: %v", err)
	}

	// Verify the structure
	if versionInfo.SchemaVersion != "1.0.0" {
		t.Errorf("Expected schema version 1.0.0, got %s", versionInfo.SchemaVersion)
	}

	if versionInfo.CompatibleSince != "1.0.0" {
		t.Errorf("Expected compatible since 1.0.0, got %s", versionInfo.CompatibleSince)
	}

	if versionInfo.GeneratedAt == "" {
		t.Error("GeneratedAt should not be empty")
	}

	// Commit might be "unknown" if git is not available in test
	if versionInfo.Commit == "" {
		t.Error("Commit should not be empty")
	}
}
