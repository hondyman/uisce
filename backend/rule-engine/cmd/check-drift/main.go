package main

import (
	"os"
	"os/exec"
)

func main() {
	// Run go generate to ensure all generators are up to date
	cmd := exec.Command("go", "generate", "./...")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		os.Exit(1)
	}

	// Check if generated files have changed (schema drift)
	cmd = exec.Command("git", "diff", "--exit-code", "backend/rule-engine/generated")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		// git diff --exit-code returns non-zero if there are changes
		os.Exit(1)
	}
}
