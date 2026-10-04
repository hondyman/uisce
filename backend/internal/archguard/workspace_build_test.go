package archguard

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

// This guard keeps the per-module workspace build honest.
//
// The check it guards does not exist by accident. `go build ./...` at the
// repository root exits 0 while compiling 19 packages, because every module in
// go.work carries its own go.mod and Go excludes nested modules from a parent
// module's `./...`. Zero packages come from backend/, calc-engine/, apps/*,
// portfolio-management/ or rebalancing/*. Two of those modules did not compile
// — portfolio-management/backend since 2026-08-05, rebalancing/worker since the
// same commit — and the root build was green the entire time.
//
// That is the same failure shape as the `backend` path filter guarded in
// workflow_path_filter_test.go: a gate whose defect is not a wrong entry but a
// missing one, so it cannot be reviewed by reading it. The fix is the same
// too. This test does not check that the filter's entries are correct. It
// re-derives, from go.work, the set of module directories a workspace build
// must cover, and fails if the workflow does not cover them or does not derive
// them at run time.
//
// Two things are asserted, and the second is the one that would have prevented
// the original mistake:
//
//  1. the `workspace` path filter covers every directory go.work lists, so a
//     module added later cannot be silently excluded from the trigger;
//  2. the build step derives its module list from go.work at run time
//     (`go work edit`), rather than carrying a hardcoded list or -- the trap --
//     a single `go build ./...` at the root that compiles almost nothing.

const workspaceJob = "workspace-modules"

// goWorkModuleDirs returns every directory listed in go.work's `use` block,
// normalised to a repo-relative path without a leading "./".
//
// It deliberately does not skip a directory that lacks a go.mod. Such an entry
// would make the workflow's build loop warn-and-continue past it, so it is
// exactly the case worth failing on here rather than filtering out.
func goWorkModuleDirs(t *testing.T, root string) []string {
	t.Helper()

	raw, err := os.ReadFile(filepath.Join(root, "go.work"))
	if err != nil {
		t.Fatalf("cannot read go.work: %v", err)
	}
	// "use (\n\t./backend\n)" and a bare "use ./backend" are both legal.
	useRe := regexp.MustCompile(`(?m)^\s*(?:use\s+)?(?:\./)?([A-Za-z0-9_./-]+)\s*$`)

	var dirs []string
	seen := map[string]bool{}
	for _, m := range useRe.FindAllStringSubmatch(string(raw), -1) {
		dir := m[1]
		if dir == "" || seen[dir] {
			continue
		}
		// A bare "." is the root module.
		if dir == "." {
			dir = "."
		}
		seen[dir] = true
		dirs = append(dirs, dir)
	}
	if len(dirs) == 0 {
		t.Fatalf("found no module directories in go.work. Either go.work is empty " +
			"or its format changed; refusing to conclude anything about coverage.")
	}
	return dirs
}

// cicdWorkspaceFilter returns the `workspace` filter list from the `changes`
// job, and the run script of the workspace build step.
//
// Both are read from the workflow rather than hardcoded here, so the test keeps
// working when either is moved. It fails the test rather than returning an
// empty value if the structure is not what it expects: an empty result would
// make every module look uncovered, turning a structural change into a
// confusing assertion failure instead of "the workflow moved".
func cicdWorkspaceFilterAndScript(t *testing.T, root string) (filter []string, script string) {
	t.Helper()

	raw, err := os.ReadFile(filepath.Join(root, cicdWorkflow))
	if err != nil {
		t.Fatalf("cannot read %s: %v", cicdWorkflow, err)
	}

	var wf struct {
		Jobs map[string]struct {
			Steps []struct {
				ID   string                   `yaml:"id"`
				Run  string                   `yaml:"run"`
				With struct{ Filters string } `yaml:"with"`
			} `yaml:"steps"`
		} `yaml:"jobs"`
	}
	if err := yaml.Unmarshal(raw, &wf); err != nil {
		t.Fatalf("cannot parse %s: %v", cicdWorkflow, err)
	}

	changes, ok := wf.Jobs["changes"]
	if !ok {
		t.Fatalf("no `changes` job in %s; the workspace filter is defined there", cicdWorkflow)
	}
	var filters string
	for _, s := range changes.Steps {
		if s.ID == "filter" && s.With.Filters != "" {
			filters = s.With.Filters
		}
	}
	if filters == "" {
		t.Fatalf("the `changes` job in %s has no `filter` step with a `filters:` block", cicdWorkflow)
	}
	var parsed struct {
		Workspace []string `yaml:"workspace"`
	}
	if err := yaml.Unmarshal([]byte(filters), &parsed); err != nil {
		t.Fatalf("cannot parse the embedded filters block: %v", err)
	}
	if len(parsed.Workspace) == 0 {
		t.Fatalf("the `workspace` filter in %s is empty. An empty filter would skip the "+
			"per-module build on every change, which is the failure this guard exists "+
			"to prevent.", cicdWorkflow)
	}

	job, ok := wf.Jobs[workspaceJob]
	if !ok {
		t.Fatalf("no `%s` job in %s. The per-module build this test guards is defined "+
			"there; if it was renamed, update workspaceJob and this test together.",
			workspaceJob, cicdWorkflow)
	}
	for _, s := range job.Steps {
		if strings.Contains(s.Run, "go build") {
			script = s.Run
		}
	}
	if script == "" {
		t.Fatalf("no step in the `%s` job of %s runs `go build`", workspaceJob, cicdWorkflow)
	}
	return parsed.Workspace, script
}

// TestWorkspaceBuildFilterCoversEveryGoWorkModule proves the trigger cannot
// silently exclude a module.
//
// A module directory that no filter entry matches never triggers the build, so
// it can be broken for as long as the filter is wrong and the pipeline stays
// green. This is the missing-entry shape, not a wrong-entry one, so it cannot
// be found by reading the filter.
func TestWorkspaceBuildFilterCoversEveryGoWorkModule(t *testing.T) {
	root := repoRoot(t)
	dirs := goWorkModuleDirs(t, root)
	filter, _ := cicdWorkspaceFilterAndScript(t, root)

	var uncovered []string
	for _, dir := range dirs {
		if dir == "." {
			// The root module's files are covered by the go.work/go.sum/
			// **/*.go entries; there is no directory prefix to match.
			continue
		}
		covered := false
		for _, pattern := range filter {
			if strings.HasSuffix(pattern, "/**") && strings.HasPrefix(dir+"/", strings.TrimSuffix(pattern, "/**")+"/") {
				covered = true
				break
			}
			if strings.Contains(pattern, "*") && matchesGlob(pattern, dir) {
				covered = true
				break
			}
			if pattern == dir {
				covered = true
				break
			}
		}
		if !covered {
			uncovered = append(uncovered, dir)
		}
	}

	if len(uncovered) > 0 {
		t.Fatalf("go.work lists %d module(s) the `workspace` path filter does not cover: %v\n"+
			"A module no filter entry matches never triggers the per-module build, so it can "+
			"stay broken while the pipeline reports green. Add an entry for each directory, or a "+
			"`**/*.go` style entry that covers it.", len(uncovered), uncovered)
	}
	t.Logf("workspace filter covers all %d go.work modules", len(dirs))
}

// TestWorkspaceBuildDerivesModulesFromGoWork is the assertion that would have
// prevented the original mistake.
//
// `go build ./...` at the repository root exits 0 and compiles almost nothing,
// because nested modules are excluded from a parent module's `./...`. A job
// that hardcodes a module list, or that runs a single root build and calls it a
// workspace build, is the same trap wearing a different hat: correct-looking,
// permanently green, and measuring nothing.
func TestWorkspaceBuildDerivesModulesFromGoWork(t *testing.T) {
	root := repoRoot(t)
	_, script := cicdWorkspaceFilterAndScript(t, root)

	if !strings.Contains(script, "go work edit") {
		t.Errorf("the `%s` job does not derive its module list from go.work at run time.\n"+
			"A hardcoded list goes stale the moment a module is added, and a single "+
			"`go build ./...` at the root compiles only the root module (Go excludes "+
			"nested modules), reporting success while the workspace is broken.\n"+
			"Derive the list with `go work edit -json | jq -r '.Use[].DiskPath'` and "+
			"build each directory in turn.\n\ngot:\n%s", workspaceJob, script)
	}

	// A root-level `go build ./...` is the specific thing that must never be
	// mistaken for a workspace build. Allow it only when it is inside a loop
	// that has changed directory.
	if strings.Contains(script, "go build ./...") && !strings.Contains(script, "cd ") {
		t.Errorf("the `%s` job runs `go build ./...` without changing directory. Run from "+
			"the repository root that compiles only the root module.", workspaceJob)
	}
}

// TestWorkspaceBuildFailsOnZeroModules is a guard on the guard.
//
// The build step's own safety net treats "built 0 modules" as an error rather
// than success, because a broken derivation that reports a clean pass is worse
// than a build that reports nothing.
//
// The assertion is deliberately narrow: it requires a comparison of the module
// count against zero, not merely the presence of the word "count" or of
// "exit 1". An earlier version of this test checked both of those and passed
// with the zero-check deleted, because the job's separate "any module failed"
// branch also contains `exit 1` and the count is still printed. A guard whose
// subject can be removed without it noticing is decoration.
func TestWorkspaceBuildFailsOnZeroModules(t *testing.T) {
	root := repoRoot(t)
	_, script := cicdWorkspaceFilterAndScript(t, root)

	countComparedToZero := regexp.MustCompile(`\$\{?count\}?"?\s*-(?:eq|le|lt)\s*0`)
	if !countComparedToZero.MatchString(script) {
		t.Errorf("the `%s` job never compares the number of built modules against zero.\n"+
			"An empty or mis-parsed go.work would then produce a clean pass over nothing, "+
			"which is the failure this whole check exists to prevent. The step should "+
			"fail when the count is 0, not only when some module fails to build.\n\n"+
			"got:\n%s", workspaceJob, script)
	}
}

// matchesGlob implements the subset of glob semantics dorny/paths-filter
// applies to the forms this repository writes, namely `**/*.go` and `**.mod`.
//
// An unrecognised form returns false: for a coverage check the safe direction
// to be wrong in is "not covered", because that produces a failing test rather
// than a passing one.
func matchesGlob(pattern, path string) bool {
	if pattern == "**/*.go" {
		return strings.HasSuffix(path, ".go")
	}
	if strings.HasPrefix(pattern, "**/") {
		return strings.HasSuffix(path, strings.TrimPrefix(pattern, "**/"))
	}
	return false
}
