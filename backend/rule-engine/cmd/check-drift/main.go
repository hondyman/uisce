// check-drift regenerates the ASL schema/types/Monaco config/version info
// and the browser WASM build, then fails if any of it differs from what's
// committed - the CI-facing guard that the codegen pipeline (and the
// internal/rules/vm AST it now reads from) haven't drifted apart.
//
// Every subcommand here is run with an explicit working directory rather
// than trusting the caller's cwd - the whole reason this file needed
// fixing in the first place was that `go generate ./...`, invoked from an
// unexpected cwd, silently wrote generated output to the wrong place
// instead of failing (see cmd/generate-schema and cmd/generate-types,
// which had the same class of bug and are the reason stray
// backend/rule-engine/cmd/generated/ and similar directories ended up
// committed to this repo).
package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

func main() {
	_, thisFile, _, _ := runtime.Caller(0)
	ruleEngineRoot := filepath.Join(filepath.Dir(thisFile), "..", "..")
	repoRoot := filepath.Join(ruleEngineRoot, "..", "..")

	run := func(dir string, name string, args ...string) {
		cmd := exec.Command(name, args...)
		cmd.Dir = dir
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			log.Fatalf("check-drift: %s %v (in %s) failed: %v", name, args, dir, err)
		}
	}

	// Regenerate schema/types/Monaco/version from the current Go source.
	run(ruleEngineRoot, "go", "generate", "./...")

	// Regenerate the browser WASM build from internal/rules/vm - not
	// covered by `go generate` (a separate GOOS/GOARCH build), but just as
	// load-bearing for drift: this is the artifact frontend/public/
	// depends on (see the commit that discovered it wasn't being kept in
	// sync at all).
	wasmOut := filepath.Join(ruleEngineRoot, "generated", "rule_engine.wasm")
	wasmCmd := exec.Command("go", "build", "-o", wasmOut, "./cmd/wasm")
	wasmCmd.Dir = ruleEngineRoot
	wasmCmd.Env = append(os.Environ(), "GOOS=js", "GOARCH=wasm")
	wasmCmd.Stdout = os.Stdout
	wasmCmd.Stderr = os.Stderr
	if err := wasmCmd.Run(); err != nil {
		log.Fatalf("check-drift: wasm build failed: %v", err)
	}

	// Compare against what's committed. The wasm binary is intentionally
	// excluded from the byte-diff (Go's GOOS=js/wasm output isn't
	// reproducible build-to-build - varies even with unchanged source, see
	// the commit that landed this fix) and instead only checked for
	// existence + non-triviality, so this doesn't flag false drift on
	// every run.
	diffCmd := exec.Command("git", "diff", "--exit-code", "--",
		"backend/rule-engine/generated/asl.d.ts",
		"backend/rule-engine/generated/asl.schema.json",
		"backend/rule-engine/generated/asl.monaco.json",
	)
	diffCmd.Dir = repoRoot
	diffCmd.Stdout = os.Stdout
	diffCmd.Stderr = os.Stderr
	if err := diffCmd.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "check-drift: generated schema/types/monaco config is out of date - run `go generate ./...` from backend/rule-engine and commit the result")
		os.Exit(1)
	}

	if info, err := os.Stat(wasmOut); err != nil || info.Size() < 1_000_000 {
		fmt.Fprintf(os.Stderr, "check-drift: rule_engine.wasm build looks broken or missing (%v)\n", err)
		os.Exit(1)
	}

	fmt.Println("check-drift: generated artifacts are up to date")
}
