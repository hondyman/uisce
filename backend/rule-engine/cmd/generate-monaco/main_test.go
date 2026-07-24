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

	// Change to temp directory and run main
	oldWd, _ := os.Getwd()
	os.Chdir(tempDir)
	defer os.Chdir(oldWd)

	// Create the generated directory
	if err := os.MkdirAll("generated", 0o755); err != nil {
		t.Fatalf("Failed to create generated dir: %v", err)
	}

	// Run the generator (may fail due to missing packages, but should still create valid JSON)
	main()

	// Check if output file exists
	outputFile := filepath.Join("generated", "asl.monaco.json")
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
