package main

import (
	"encoding/json"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

const (
	outputDir     = "generated"
	outputVersion = "version.json"
)

type VersionInfo struct {
	SchemaVersion   string `json:"schemaVersion"`
	GeneratedAt     string `json:"generatedAt"`
	Commit          string `json:"commit"`
	CompatibleSince string `json:"compatibleSince"`
}

func main() {
	commit := gitCommit()
	now := time.Now().UTC().Format(time.RFC3339)

	info := VersionInfo{
		SchemaVersion:   "1.0.0",
		GeneratedAt:     now,
		Commit:          commit,
		CompatibleSince: "1.0.0",
	}

	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		log.Fatalf("failed to create output dir: %v", err)
	}

	outPath := filepath.Join(outputDir, outputVersion)
	data, err := json.MarshalIndent(info, "", "  ")
	if err != nil {
		log.Fatalf("failed to marshal version: %v", err)
	}

	if err := os.WriteFile(outPath, data, 0o644); err != nil {
		log.Fatalf("failed to write %s: %v", outPath, err)
	}
}

func gitCommit() string {
	cmd := exec.Command("git", "rev-parse", "HEAD")
	out, err := cmd.Output()
	if err != nil {
		return "unknown"
	}
	return string(out[:len(out)-1])
}
