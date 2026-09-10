package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// TestGenerateMonacoMetadata tests the Monaco metadata generation
func TestGenerateMonacoMetadata(t *testing.T) {
	// Create temporary directory for output
	tempDir, err := os.MkdirTemp("", "monaco-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// buildMetadata/writeMetadata take the output dir and package set
	// explicitly - no os.Chdir needed (and none would work here: chdir'ing
	// outside the module, as the old version of this test did, makes
	// packages.Load fail to resolve goPkgPaths at all).
	built := buildMetadata()
	if err := writeMetadata(tempDir, built); err != nil {
		t.Fatalf("writeMetadata failed: %v", err)
	}

	// Check if output file exists
	outputFile := filepath.Join(tempDir, "asl.monaco.json")
	if _, err := os.Stat(outputFile); os.IsNotExist(err) {
		t.Fatalf("Output file %s was not created", outputFile)
	}

	// Read and validate JSON
	data, err := os.ReadFile(outputFile)
	if err != nil {
		t.Fatalf("Failed to read output file: %v", err)
	}

	// Should be valid JSON even if empty
	var meta MonacoMetadata
	if err := json.Unmarshal(data, &meta); err != nil {
		t.Fatalf("Failed to parse JSON output: %v", err)
	}

	// Basic validation that the structure is initialized
	if meta.Snippets == nil {
		t.Error("Expected snippets to be initialized")
	}

	if meta.Enums == nil {
		t.Error("Expected enums to be initialized")
	}
}
