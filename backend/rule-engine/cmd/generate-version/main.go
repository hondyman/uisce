package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"
)

const (
	outputVersion = "version.json"
)

// outputDir is resolved from this file's own location, not the process
// cwd - see the matching comment in cmd/generate-schema/main.go.
var outputDir = func() string {
	_, thisFile, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(thisFile), "..", "..", "generated")
}()

type VersionInfo struct {
	SchemaVersion   string `json:"schemaVersion"`
	GeneratedAt     string `json:"generatedAt"`
	Commit          string `json:"commit"`
	CompatibleSince string `json:"compatibleSince"`
}

func main() {
	if err := generateVersionInfo(outputDir); err != nil {
		log.Fatal(err)
	}
}

// generateVersionInfo writes version.json into dir. Takes an explicit
// directory (rather than reading the package-level outputDir global
// itself) so tests can point it at a temp dir directly - outputDir is now
// an absolute path fixed at package init via runtime.Caller (see above),
// so the old os.Chdir()-based test technique for redirecting a "relative"
// outputDir no longer applies, and isn't worth preserving: real dependency
// injection is simpler than routing through cwd and a global.
func generateVersionInfo(dir string) error {
	commit := gitCommit()
	now := time.Now().UTC().Format(time.RFC3339)

	info := VersionInfo{
		SchemaVersion:   "1.0.0",
		GeneratedAt:     now,
		Commit:          commit,
		CompatibleSince: "1.0.0",
	}

	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("failed to create output dir: %w", err)
	}

	outPath := filepath.Join(dir, outputVersion)
	data, err := json.MarshalIndent(info, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal version: %w", err)
	}

	if err := os.WriteFile(outPath, data, 0o644); err != nil {
		return fmt.Errorf("failed to write %s: %w", outPath, err)
	}
	return nil
}

func gitCommit() string {
	cmd := exec.Command("git", "rev-parse", "HEAD")
	out, err := cmd.Output()
	if err != nil {
		return "unknown"
	}
	return string(out[:len(out)-1])
}
