package main

import (
	"encoding/json"
	"fmt"
	"go/types"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"sort"

	"golang.org/x/tools/go/packages"

	"github.com/hondyman/uisce/backend/internal/rules/vm"
)

const (
	// outputDir is a filesystem path and needs the runtime.Caller-based fix
	// below (this generator previously assumed cwd = rule-engine/, the
	// opposite assumption from generate-schema and generate-types, which
	// is exactly what produced stray cmd/generate-monaco/generated/
	// directories when invoked the other way).
	outputMonaco = "asl.monaco.json"
)

// goPkgPaths are Go import paths, not filesystem paths - packages.Load
// resolves them via the module system (go.mod), so this works from any cwd
// inside the module. internal/rules/vm is the canonical rule/calc AST
// (RuleNode, RuleGroup, RuleCondition, Expression, FuncCall, ...) - without
// it this generator's NodeKinds/Snippets never included any of it, the
// same class of gap fixed for generate-schema/generate-types.
var goPkgPaths = []string{
	"github.com/hondyman/uisce/backend/internal/services",
	"github.com/hondyman/uisce/backend/internal/rules/vm",
}

var outputDir = func() string {
	_, thisFile, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(thisFile), "..", "..", "generated")
}()

type MonacoSnippet struct {
	Label  string `json:"label"`
	Insert string `json:"insert"`
	Detail string `json:"detail"`
}

// MonacoFunction describes one registered rule/calc function for editor
// autocomplete and capability badges - derived directly from
// vm.LibraryEntries() (the same FunctionSpec registry that drives native
// evaluation and SQL pushdown), not redeclared here. Pushdown is keyed by
// dialect name so the editor can badge "pushdownable to StarRocks" vs.
// "native/WASM only" per binding, not just a single yes/no.
type MonacoFunction struct {
	Name         string          `json:"name"`
	Signature    string          `json:"signature"`
	Category     string          `json:"category"`
	Description  string          `json:"description"`
	NoClosedForm bool            `json:"noClosedForm"`
	Pushdown     map[string]bool `json:"pushdown"`
}

type MonacoMetadata struct {
	Keywords  []string            `json:"keywords"`
	Operators []string            `json:"operators"`
	NodeKinds []string            `json:"nodeKinds"`
	Snippets  []MonacoSnippet     `json:"snippets"`
	Enums     map[string][]string `json:"enums"`
	Functions []MonacoFunction    `json:"functions"`
}

// libraryDialects lists every SQL dialect the editor should report a
// pushdown badge for - extend alongside vm.Dialect as new dialects gain
// SQLEmit entries.
var libraryDialects = []vm.Dialect{vm.DialectStarRocks}

// buildFunctionMetadata derives editor-facing function metadata straight
// from the live vm.Library (a real import, not go/types static analysis -
// unlike NodeKinds/Enums below, function capabilities are registry data,
// not Go type declarations, so reading them at generation time just means
// calling the registry directly).
func buildFunctionMetadata() []MonacoFunction {
	entries := vm.LibraryEntries()
	fns := make([]MonacoFunction, 0, len(entries))
	for _, spec := range entries {
		pushdown := make(map[string]bool, len(libraryDialects))
		for _, d := range libraryDialects {
			pushdown[string(d)] = spec.Pushdownable(d)
		}
		fns = append(fns, MonacoFunction{
			Name:         spec.Name,
			Signature:    spec.Signature,
			Category:     spec.Category,
			Description:  spec.Description,
			NoClosedForm: spec.NoClosedForm,
			Pushdown:     pushdown,
		})
	}
	return fns
}

func main() {
	meta := buildMetadata()
	if err := writeMetadata(outputDir, meta); err != nil {
		log.Fatal(err)
	}
}

// buildMetadata loads goPkgPaths and derives Monaco completion metadata
// from their real types (go/types, not go/ast text parsing - so it
// resolves cross-package references, unlike generate-schema/generate-types).
// Pure/no I/O so tests can call it directly instead of running main()
// after os.Chdir()'ing into a temp dir: packages.Load needs a real cwd
// inside the module to resolve goPkgPaths at all (chdir'ing outside it, as
// the old test did, made every load fail with "go.mod file not found").
func buildMetadata() MonacoMetadata {
	cfg := &packages.Config{
		Mode: packages.NeedTypes | packages.NeedTypesInfo | packages.NeedSyntax,
	}

	pkgs, err := packages.Load(cfg, goPkgPaths...)
	if err != nil {
		log.Printf("Warning: failed to load packages: %v", err)
		// Continue with empty metadata
		pkgs = []*packages.Package{}
	}
	if packages.PrintErrors(pkgs) > 0 {
		log.Printf("Warning: errors while loading packages")
		// Continue with empty metadata
		pkgs = []*packages.Package{}
	}

	meta := MonacoMetadata{
		Keywords:  []string{"rule", "when", "then", "and", "or", "not"},
		Operators: []string{},
		NodeKinds: []string{},
		Snippets:  []MonacoSnippet{},
		Enums:     map[string][]string{},
	}

	for _, pkg := range pkgs {
		scope := pkg.Types.Scope()
		names := scope.Names()
		sort.Strings(names)

		for _, name := range names {
			obj := scope.Lookup(name)
			typeName, ok := obj.(*types.TypeName)
			if !ok {
				continue
			}

			named, ok := typeName.Type().(*types.Named)
			if !ok {
				continue
			}

			// Enums
			if isEnum(named) {
				values := collectEnumValues(named)
				sort.Strings(values)
				meta.Enums[name] = values

				if isOperatorEnum(name) {
					meta.Operators = append(meta.Operators, values...)
				}
				continue
			}

			// Structs
			structType, ok := named.Underlying().(*types.Struct)
			if !ok {
				continue
			}

			if hasDiscriminator(structType) {
				meta.NodeKinds = append(meta.NodeKinds, name)
				meta.Snippets = append(meta.Snippets, MonacoSnippet{
					Label:  name,
					Insert: name + " {\n  \n}",
					Detail: "Insert " + name + " node",
				})
			}
		}
	}

	meta.Functions = buildFunctionMetadata()
	for _, fn := range meta.Functions {
		detail := fn.Description
		if fn.Signature != "" {
			detail = fn.Signature + " - " + detail
		}
		meta.Snippets = append(meta.Snippets, MonacoSnippet{
			Label:  fn.Name,
			Insert: fn.Name + "()",
			Detail: detail,
		})
	}

	sort.Strings(meta.Operators)
	sort.Strings(meta.NodeKinds)
	sort.Slice(meta.Snippets, func(i, j int) bool {
		return meta.Snippets[i].Label < meta.Snippets[j].Label
	})

	return meta
}

// writeMetadata writes meta to dir/asl.monaco.json. Takes dir explicitly
// (rather than the package-level outputDir global) for the same reason as
// generate-version's generateVersionInfo: real dependency injection for
// tests instead of routing through cwd and a global.
func writeMetadata(dir string, meta MonacoMetadata) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("failed to create output dir: %w", err)
	}

	outPath := filepath.Join(dir, outputMonaco)
	data, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal monaco metadata: %w", err)
	}

	if err := os.WriteFile(outPath, data, 0o644); err != nil {
		return fmt.Errorf("failed to write %s: %w", outPath, err)
	}
	return nil
}

func isEnum(named *types.Named) bool {
	_, ok := named.Underlying().(*types.Basic)
	return ok
}

func collectEnumValues(named *types.Named) []string {
	pkg := named.Obj().Pkg()
	if pkg == nil {
		return nil
	}

	scope := pkg.Scope()
	names := scope.Names()
	sort.Strings(names)

	var values []string
	for _, name := range names {
		obj := scope.Lookup(name)
		c, ok := obj.(*types.Const)
		if ok && c.Type() == named {
			values = append(values, c.Val().ExactString())
		}
	}
	return values
}

func isOperatorEnum(name string) bool {
	return name == "Operator" || name == "RuleOperator"
}

func hasDiscriminator(s *types.Struct) bool {
	for i := 0; i < s.NumFields(); i++ {
		f := s.Field(i)
		if f.Exported() && (f.Name() == "Type" || f.Name() == "Kind") {
			return true
		}
	}
	return false
}
