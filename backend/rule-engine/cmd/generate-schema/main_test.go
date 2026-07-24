package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestGenerateSchemaGoldenFile tests the generate-schema command against a golden file
func TestGenerateSchemaGoldenFile(t *testing.T) {
	// Run the generator
	cmd := exec.Command("go", "run", "main.go")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("generator failed: %v\nOutput: %s", err, string(output))
	}

	// Read generated file
	got, err := os.ReadFile("../../generated/asl.schema.json")
	if err != nil {
		t.Fatalf("failed to read generated file: %v", err)
	}

	// Check if golden file exists
	goldenPath := "testdata/asl.schema.json"
	if _, err := os.Stat(goldenPath); os.IsNotExist(err) {
		// Create golden file if it doesn't exist
		if err := os.WriteFile(goldenPath, got, 0644); err != nil {
			t.Fatalf("failed to create golden file: %v", err)
		}
		t.Logf("Created golden file: %s", goldenPath)
		return
	}

	// Read golden file
	want, err := os.ReadFile(goldenPath)
	if err != nil {
		t.Fatalf("failed to read golden file: %v", err)
	}

	// Compare (normalize line endings)
	gotStr := strings.ReplaceAll(string(got), "\r\n", "\n")
	wantStr := strings.ReplaceAll(string(want), "\r\n", "\n")

	if gotStr != wantStr {
		t.Errorf("generated output does not match golden file")
		t.Errorf("To update golden file, run: cp ../../generated/asl.schema.json testdata/asl.schema.json")

		// Write diff to help debugging
		gotFile := filepath.Join(t.TempDir(), "got.json")
		wantFile := filepath.Join(t.TempDir(), "want.json")
		os.WriteFile(gotFile, []byte(gotStr), 0644)
		os.WriteFile(wantFile, []byte(wantStr), 0644)
		t.Logf("Generated: %s", gotFile)
		t.Logf("Expected: %s", wantFile)
	}
}

// TestGenerateSchemaDeterministic tests that the generator produces deterministic output
func TestGenerateSchemaDeterministic(t *testing.T) {
	// Run generator twice
	cmd := exec.Command("go", "run", "main.go")
	if err := cmd.Run(); err != nil {
		t.Fatalf("first generator run failed: %v", err)
	}

	got1, err := os.ReadFile("../../generated/asl.schema.json")
	if err != nil {
		t.Fatalf("failed to read first generated file: %v", err)
	}

	cmd = exec.Command("go", "run", "main.go")
	if err := cmd.Run(); err != nil {
		t.Fatalf("second generator run failed: %v", err)
	}

	got2, err := os.ReadFile("../../generated/asl.schema.json")
	if err != nil {
		t.Fatalf("failed to read second generated file: %v", err)
	}

	if string(got1) != string(got2) {
		t.Errorf("generator output is not deterministic")
	}
}