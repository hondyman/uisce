package archguard

import (
	"fmt"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

// This guard keeps the `backend` path filter in .github/workflows/ci-cd.yml honest.
//
// The filter exists to skip the heavy backend jobs on a pull request that cannot affect them.
// Its failure mode is silent and one-directional: a path the filter forgets does not fail the
// build, it stops the build from running, and the pull request reports green. So the filter
// cannot be reviewed by reading it -- every entry that is present looks correct, and the defect
// is the entry nobody thought of.
//
// This test does not check the entries that are present. It re-derives, from go.work and the
// import statements under backend/ and cmd/, the set of Go modules a backend build actually
// depends on, and fails if the filter does not cover them. A module added to go.work later is
// therefore caught here rather than in a pull request that quietly stops building.
//
// It was written because the first version of the filter omitted calc-engine/** while three
// files under backend/ imported it:
//
//	internal/calc-engine/worker/init.go
//	internal/api/calc-engine_handlers.go
//	internal/analytics/semantic_calculation_service.go
//
// build-backend runs `go build ./...` in backend/, so a pull request touching only calc-engine/
// would have skipped the build that would have caught a broken API. The test suite itself was
// not affected: the shards run `go test` with working-directory backend, so they only ever
// covered the backend module.

// cicdWorkflow is the repo-relative path of the workflow under guard.
const cicdWorkflow = ".github/workflows/ci-cd.yml"

// repoRoot walks up from the working directory until it finds the directory that
// holds both go.work and the workflow, so the test works whether it is invoked
// from the repo root or from backend/ (which is how the shards invoke it).
func repoRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("cannot determine working directory: %v", err)
	}
	for {
		goWork := filepath.Join(dir, "go.work")
		wf := filepath.Join(dir, cicdWorkflow)
		if fileExists(goWork) && fileExists(wf) {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatalf("could not find a directory containing both go.work and %s "+
				"walking up from the working directory; last directory tried was %s",
				cicdWorkflow, dir)
		}
		dir = parent
	}
}

func fileExists(p string) bool {
	st, err := os.Stat(p)
	return err == nil && !st.IsDir()
}

// backendFilterPaths extracts the `backend` filter list from the `changes` job
// in ci-cd.yml. The filter is a YAML document embedded in a step's `with:`
// block, so this parses the workflow and then parses that string.
//
// It fails the test rather than returning an empty list if the structure is not
// what it expects: an empty list would make every module look uncovered and
// turn a structural change into a confusing assertion failure. Better to say
// the workflow shape moved.
func backendFilterPaths(t *testing.T, root string) []string {
	t.Helper()

	raw, err := os.ReadFile(filepath.Join(root, cicdWorkflow))
	if err != nil {
		t.Fatalf("cannot read %s: %v", cicdWorkflow, err)
	}

	// Only the parts of the workflow this test needs.
	var wf struct {
		Jobs map[string]struct {
			Steps []struct {
				ID   string `yaml:"id"`
				With struct {
					Filters string `yaml:"filters"`
				} `yaml:"with"`
			} `yaml:"steps"`
		} `yaml:"jobs"`
	}
	if err := yaml.Unmarshal(raw, &wf); err != nil {
		t.Fatalf("cannot parse %s: %v", cicdWorkflow, err)
	}

	changes, ok := wf.Jobs["changes"]
	if !ok {
		t.Fatalf("no `changes` job in %s. The path filter this test guards is "+
			"defined there; if it was renamed or moved, update cicdWorkflow and "+
			"this test together.", cicdWorkflow)
	}

	var filters string
	for _, s := range changes.Steps {
		if s.ID == "filter" && s.With.Filters != "" {
			filters = s.With.Filters
		}
	}
	if filters == "" {
		t.Fatalf("the `changes` job in %s has no step with id `filter` carrying a "+
			"`filters:` block. This test reads the backend filter from there.",
			cicdWorkflow)
	}

	var parsed struct {
		Backend []string `yaml:"backend"`
	}
	if err := yaml.Unmarshal([]byte(filters), &parsed); err != nil {
		t.Fatalf("cannot parse the embedded filters block: %v", err)
	}
	if len(parsed.Backend) == 0 {
		t.Fatalf("the `backend` filter in %s is empty. An empty filter would skip "+
			"every backend job on every pull request, including backend changes.",
			cicdWorkflow)
	}
	return parsed.Backend
}

// workspaceModules maps each directory listed in go.work's `use` block to the
// module path declared in that directory's go.mod. Directories without a
// readable go.mod are skipped rather than guessed at.
func workspaceModules(t *testing.T, root string) map[string]string {
	t.Helper()

	raw, err := os.ReadFile(filepath.Join(root, "go.work"))
	if err != nil {
		t.Fatalf("cannot read go.work: %v", err)
	}
	// `use` entries look like "\t./backend" or "use ./backend".
	useRe := regexp.MustCompile(`(?m)^\s*(?:use\s+)?\./(\S+)\s*$`)
	moduleRe := regexp.MustCompile(`(?m)^module\s+(\S+)\s*$`)

	out := map[string]string{}
	for _, m := range useRe.FindAllStringSubmatch(string(raw), -1) {
		dir := m[1]
		gomod, err := os.ReadFile(filepath.Join(root, dir, "go.mod"))
		if err != nil {
			continue
		}
		if mod := moduleRe.FindSubmatch(gomod); mod != nil {
			out[string(mod[1])] = dir
		}
	}
	if len(out) == 0 {
		t.Fatalf("found no modules in go.work. Either go.work is empty or its " +
			"format changed; refusing to conclude anything about coverage.")
	}
	return out
}

// filterCovers reports whether a `paths` filter entry matches a repo-relative
// file path, under the semantics dorny/paths-filter uses for the forms this
// repository actually writes:
//
//	"backend/**"   matches anything under backend/
//	"go.work"      matches exactly that file
//
// Anything it does not recognise returns false. For a coverage check the safe
// direction to be wrong in is "not covered", because that produces a failing
// test rather than a passing one. A pattern form that appears later and is
// mishandled here will therefore surface as a false failure, not a false pass.
func filterCovers(pattern, path string) bool {
	if strings.HasSuffix(pattern, "/**") {
		return strings.HasPrefix(path, strings.TrimSuffix(pattern, "/**")+"/")
	}
	if strings.Contains(pattern, "*") {
		// Unrecognised glob form: report uncovered rather than assume.
		return false
	}
	return pattern == path
}

// importedWorkspaceModules walks the Go sources under roots and returns the
// set of workspace module import paths they reference, mapped to the
// directories those modules live in.
func importedWorkspaceModules(t *testing.T, root string, modules map[string]string) map[string]string {
	t.Helper()

	// Longest module path first, so an import of a nested module is not
	// attributed to a shorter module that happens to share a prefix.
	paths := make([]string, 0, len(modules))
	for p := range modules {
		paths = append(paths, p)
	}
	sort.Slice(paths, func(i, j int) bool {
		if len(paths[i]) != len(paths[j]) {
			return len(paths[i]) > len(paths[j])
		}
		return paths[i] < paths[j]
	})

	found := map[string]string{}
	for _, rel := range []string{"backend", "cmd"} {
		base := filepath.Join(root, rel)
		if !isDir(base) {
			continue
		}
		// filepath.Walk uses Lstat, so a symlinked root is reported as a
		// non-directory and never descended. Resolve it first. This repo has
		// many worktrees, so a symlinked checkout is a realistic shape.
		if resolved, err := filepath.EvalSymlinks(base); err == nil {
			base = resolved
		}
		err := filepath.Walk(base, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if info.IsDir() {
				if info.Name() == "testdata" || info.Name() == "vendor" ||
					strings.HasPrefix(info.Name(), ".") {
					return filepath.SkipDir
				}
				return nil
			}
			if !strings.HasSuffix(path, ".go") {
				return nil
			}
			f, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.ImportsOnly)
			if err != nil {
				// Unparseable source is not this test's business; the build
				// gate will report it.
				return nil
			}
			for _, imp := range f.Imports {
				v := strings.Trim(imp.Path.Value, `"`)
				for _, mp := range paths {
					if v == mp || strings.HasPrefix(v, mp+"/") {
						found[mp] = modules[mp]
						break
					}
				}
			}
			return nil
		})
		if err != nil {
			t.Fatalf("walking %s: %v", rel, err)
		}
	}
	return found
}

func isDir(p string) bool {
	st, err := os.Stat(p)
	return err == nil && st.IsDir()
}

// TestBackendPathFilterCoversEveryImportedModule asserts that every Go module
// the backend build depends on is inside the backend path filter.
func TestBackendPathFilterCoversEveryImportedModule(t *testing.T) {
	root := repoRoot(t)
	filter := backendFilterPaths(t, root)
	modules := workspaceModules(t, root)
	imported := importedWorkspaceModules(t, root, modules)

	if len(imported) == 0 {
		t.Fatalf("found no workspace module imports under backend/ or cmd/. " +
			"The walk found nothing, so this test would pass while checking " +
			"nothing -- refusing to report success on an empty result.")
	}

	// Probe each module with a file that exists inside it, so the question
	// asked of the filter is the one the filter actually answers.
	var uncovered []string
	var checked []string
	for mod, dir := range imported {
		probe := dir + "/go.mod"
		ok := false
		for _, p := range filter {
			if filterCovers(p, probe) {
				ok = true
				break
			}
		}
		if ok {
			checked = append(checked, fmt.Sprintf("  covered   %-50s (%s/)", mod, dir))
		} else {
			uncovered = append(uncovered, fmt.Sprintf("  UNCOVERED %-50s (%s/)", mod, dir))
		}
	}
	sort.Strings(checked)
	sort.Strings(uncovered)

	t.Logf("backend filter: %v", filter)
	for _, l := range checked {
		t.Log(l)
	}
	for _, l := range uncovered {
		t.Log(l)
	}

	if len(uncovered) > 0 {
		t.Errorf("the backend path filter in %s does not cover %d of the %d Go "+
			"module(s) a backend build depends on. A pull request changing only "+
			"one of these would skip build-backend and the backend test shards "+
			"and still report green.\n\n%s\n\nAdd the listed directories to the "+
			"`backend` filter, e.g. 'calc-engine/**'.\n\nWiden the filter to fix "+
			"this; never narrow it to make the test pass.",
			cicdWorkflow, len(uncovered), len(imported), strings.Join(uncovered, "\n"))
	}
}
