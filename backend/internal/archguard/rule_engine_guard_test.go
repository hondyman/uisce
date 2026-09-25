package archguard

import (
	"bufio"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"
)

// forbiddenEngineModules are libraries that embed a second rule, expression,
// scripting or policy engine. internal/rules/vm (AST + VM, built to
// rule_engine.wasm for the browser) is the only engine; new evaluation needs
// belong there as vm functions, not as another runtime.
var forbiddenEngineModules = []string{
	"github.com/google/cel-go",
	"go.starlark.net",
	"cuelang.org/go",
	"github.com/tetratelabs/wazero",
	"github.com/wasmerio/wasmer-go",
	"github.com/bytecodealliance/wasmtime-go",
	"github.com/open-policy-agent/opa",
	"github.com/expr-lang/expr",
	"github.com/antonmedv/expr",
	"github.com/Knetic/govaluate",
	"github.com/PaesslerAG/gval",
	"github.com/hashicorp/go-bexpr",
	"github.com/robertkrimen/otto",
	"github.com/dop251/goja",
	"github.com/yuin/gopher-lua",
	"github.com/traefik/yaegi",
	"github.com/hyperjumptech/grule-rule-engine",
	"github.com/casbin/casbin",
	"github.com/diegoholiveira/jsonlogic",
}

// allowance is a known, tracked exception. Until names the plan slice or PR
// that removes it. An allowance whose file no longer imports an engine is
// stale: logged while Until is set (the removing PR may merge in any order),
// and a failure once Until is empty - permanent exceptions must stay honest.
type allowance struct {
	Reason string
	Until  string
}

// engineImportAllowlist is keyed by backend-relative file path. It is empty:
// every other engine (cel-go, Starlark, wazero, CUE, OPA) has been retired.
// Adding a file here requires a signed amendment to
// docs/rule-engine-centralization-plan.md.
var engineImportAllowlist = map[string]allowance{}

// TestSingleRuleEngine fails when a backend Go file imports a forbidden
// engine library without an allowance.
func TestSingleRuleEngine(t *testing.T) {
	root := backendRoot(t)
	found := map[string][]string{} // rel file -> engine modules imported

	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			switch d.Name() {
			case "vendor", "node_modules", "testdata", ".git":
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") {
			return nil
		}
		f, perr := parser.ParseFile(token.NewFileSet(), path, nil, parser.ImportsOnly)
		if perr != nil {
			return nil // unparsable files are the compiler's problem, not this guard's
		}
		rel, _ := filepath.Rel(root, path)
		rel = filepath.ToSlash(rel)
		for _, imp := range f.Imports {
			p, _ := strconv.Unquote(imp.Path.Value)
			if m := engineModule(p); m != "" && !contains(found[rel], m) {
				found[rel] = append(found[rel], m)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	var violations []string
	for rel, mods := range found {
		if _, ok := engineImportAllowlist[rel]; !ok {
			violations = append(violations, rel+" imports "+strings.Join(mods, ", "))
		}
	}
	sort.Strings(violations)
	for _, v := range violations {
		t.Errorf("second rule engine: %s - internal/rules/vm is the only engine (docs/rule-engine-centralization-plan.md)", v)
	}

	for rel, a := range engineImportAllowlist {
		if _, still := found[rel]; still {
			continue
		}
		if a.Until != "" {
			t.Logf("allowance for %s is stale (removed by %s) - delete it from engineImportAllowlist", rel, a.Until)
		} else {
			t.Errorf("permanent allowance for %s no longer matches any import - delete it", rel)
		}
	}
}

// TestNoNewEngineModulesInGoMod fails when go.mod requires a forbidden engine
// module that no allowed file imports - a new engine arriving as a dependency
// before any code uses it.
func TestNoNewEngineModulesInGoMod(t *testing.T) {
	root := backendRoot(t)
	allowedMods := map[string]bool{}
	for rel := range engineImportAllowlist {
		f, err := parser.ParseFile(token.NewFileSet(), filepath.Join(root, rel), nil, parser.ImportsOnly)
		if err != nil {
			continue // file already deleted by the removing PR
		}
		for _, imp := range f.Imports {
			p, _ := strconv.Unquote(imp.Path.Value)
			if m := engineModule(p); m != "" {
				allowedMods[m] = true
			}
		}
	}

	fh, err := os.Open(filepath.Join(root, "go.mod"))
	if err != nil {
		t.Fatal(err)
	}
	defer fh.Close()
	sc := bufio.NewScanner(fh)
	for sc.Scan() {
		fields := strings.Fields(strings.TrimPrefix(strings.TrimSpace(sc.Text()), "require "))
		if len(fields) == 0 {
			continue
		}
		for _, m := range forbiddenEngineModules {
			if fields[0] == m || strings.HasPrefix(fields[0], m+"/") {
				if !allowedMods[m] {
					t.Errorf("go.mod requires %s, a second rule engine with no allowed importer - internal/rules/vm is the only engine", fields[0])
				}
			}
		}
	}
	if err := sc.Err(); err != nil {
		t.Fatal(err)
	}
}

// engineModule returns the forbidden module an import path belongs to, or "".
func engineModule(importPath string) string {
	for _, m := range forbiddenEngineModules {
		if importPath == m || strings.HasPrefix(importPath, m+"/") {
			return m
		}
	}
	return ""
}

func contains(xs []string, x string) bool {
	for _, y := range xs {
		if y == x {
			return true
		}
	}
	return false
}

// backendRoot is the directory holding the backend go.mod.
func backendRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for {
		b, err := os.ReadFile(filepath.Join(dir, "go.mod"))
		if err == nil && strings.Contains(string(b), "module github.com/hondyman/uisce/backend\n") {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("backend go.mod not found above " + dir)
		}
		dir = parent
	}
}
