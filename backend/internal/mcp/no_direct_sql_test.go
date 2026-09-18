package mcp

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// transitionalDirectSQLFiles are MCP sources still allowed to call db.Query/
// Select/Get/Exec against business tables. Shrink this list to empty as Phase SL
// completes — empty + green is the invariant's standing enforcement, not a
// one-time grep receipt.
//
// audit.go is intentionally absent: audit writes to catalog_mdm_ai are an
// explicit out-of-scope ledger write, not business-table SQL owned by a tool.
// transitionalDirectSQLFiles must stay empty after SL commit 5/5 (CRIMS unify).
// Non-empty means a tool file re-acquired business SQL — fail the invariant.
var transitionalDirectSQLFiles = map[string]string{}

// sqlCallSuffixes are database/sql and sqlx query methods. Bare "Get" is
// omitted — chi.Router.Get collides and is not SQL.
var sqlCallSuffixes = []string{
	"Query", "QueryContext", "Queryx", "QueryxContext",
	"QueryRow", "QueryRowContext", "QueryRowx", "QueryRowxContext",
	"Select", "SelectContext", "GetContext",
	"Exec", "ExecContext",
}

// TestNoDirectSQLInTools fails when a non-excepted MCP tool source makes a
// database/sql or sqlx query call. Exception list must shrink to zero.
func TestNoDirectSQLInTools(t *testing.T) {
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}

	fset := token.NewFileSet()
	var offenders []string
	seenExcepted := map[string]bool{}

	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		isToolSurface := name == "tool_handler.go" ||
			(strings.HasPrefix(name, "tools_") && strings.HasSuffix(name, ".go"))
		if !isToolSurface {
			continue
		}

		src, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			t.Fatal(err)
		}
		file, err := parser.ParseFile(fset, name, src, 0)
		if err != nil {
			t.Fatalf("parse %s: %v", name, err)
		}

		hits := collectSQLCallLines(fset, file)
		if len(hits) == 0 {
			if _, ok := transitionalDirectSQLFiles[name]; ok {
				t.Errorf("%s is on the transitional exception list but has no SQL calls — remove it from transitionalDirectSQLFiles", name)
			}
			continue
		}
		reason, excepted := transitionalDirectSQLFiles[name]
		if excepted {
			seenExcepted[name] = true
			t.Logf("transitional SQL still present in %s (%s): %d call site(s)", name, reason, len(hits))
			continue
		}
		for _, h := range hits {
			offenders = append(offenders, name+":"+h)
		}
	}

	for name := range transitionalDirectSQLFiles {
		if !seenExcepted[name] {
			t.Errorf("transitionalDirectSQLFiles lists %s but file missing or unreadable in package dir", name)
		}
	}

	if len(offenders) > 0 {
		t.Fatalf("MCP tool sources must not own business SQL (Phase SL invariant). Offenders:\n  %s\nAdd to transitionalDirectSQLFiles only as a temporary, shrinking exception.",
			strings.Join(offenders, "\n  "))
	}

	remaining := len(transitionalDirectSQLFiles)
	t.Logf("TestNoDirectSQLInTools: %d transitional exception file(s) remaining (goal: 0)", remaining)
	if remaining != 0 {
		t.Fatalf("SL invariant incomplete: transitionalDirectSQLFiles length=%d (must be 0 after CRIMS unify)", remaining)
	}
}

func collectSQLCallLines(fset *token.FileSet, file *ast.File) []string {
	var hits []string
	ast.Inspect(file, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok {
			return true
		}
		name := sel.Sel.Name
		for _, suf := range sqlCallSuffixes {
			if name == suf {
				pos := fset.Position(call.Pos())
				hits = append(hits, formatHit(pos.Line, name))
				break
			}
		}
		return true
	})
	return hits
}

func formatHit(line int, method string) string {
	return fmt.Sprintf("L%d .%s", line, method)
}
