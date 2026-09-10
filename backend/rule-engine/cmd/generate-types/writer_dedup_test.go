package main

import (
	"go/ast"
	"go/parser"
	"go/token"
	"testing"
)

// TestASLTypeGenerator_DoesNotOwnMonacoSchemaOrVersion pins the fix for a
// real nondeterminism bug: this package used to define its own
// generateJSONSchema/generateMonacoMetadata/generateVersionInfo methods,
// producing genuinely different content for asl.schema.json,
// asl.monaco.json, and version.json than the canonical writers
// (cmd/generate-schema, cmd/generate-monaco, cmd/generate-version) -
// whichever generator ran last in `go generate ./...` silently won, and
// running them concurrently (e.g. under `go test ./...`, which runs each
// package's tests as a separate process) raced on the same file -
// TestGenerateSchemaDeterministic flaked exactly this way, from
// generate-types' copy racing with generate-schema's, before this fix.
// There is now exactly one writer per file; this asserts none of the
// three methods are defined on ASLTypeGenerator in this package's own
// source any more.
//
// Uses go/parser directly on main.go rather than reflect: reflect.Type's
// method introspection only sees exported methods (verified empirically -
// a deliberately reintroduced unexported generateVersionInfo was invisible
// to reflect.Type.MethodByName), so it can never observe any of these
// forbidden methods, unexported by convention. Source scanning is the only
// reliable way to assert their absence - and it's the same technique this
// package already applies to internal/rules/vm et al., turned on itself.
func TestASLTypeGenerator_DoesNotOwnMonacoSchemaOrVersion(t *testing.T) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "main.go", nil, 0)
	if err != nil {
		t.Fatalf("failed to parse main.go: %v", err)
	}

	forbidden := map[string]bool{
		"generateJSONSchema":     true,
		"generateMonacoMetadata": true,
		"generateVersionInfo":    true,
	}

	ast.Inspect(file, func(n ast.Node) bool {
		fn, ok := n.(*ast.FuncDecl)
		if !ok || fn.Recv == nil || len(fn.Recv.List) == 0 {
			return true
		}
		if forbidden[fn.Name.Name] {
			t.Errorf("method %s reappeared on ASLTypeGenerator - this duplicates cmd/generate-schema, "+
				"cmd/generate-monaco, or cmd/generate-version and reopens the writer race that produced "+
				"silently nondeterministic asl.schema.json/asl.monaco.json/version.json content", fn.Name.Name)
		}
		return true
	})
}
