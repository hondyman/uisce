package vm

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"testing"

	_ "github.com/lib/pq"
)

type ConformanceCase struct {
	Name          string          `json:"name"`
	RuleAST       json.RawMessage `json:"rule_ast"`
	Input         map[string]any  `json:"input"`
	ExpectedValid *bool           `json:"expected_valid,omitempty"`
	ExpectedError bool            `json:"expected_error,omitempty"`
}

// TestSQLConformanceParity runs conformance_cases.json through CompileToSQL
// against a real Postgres, proving the compiled predicate agrees with the
// in-memory evaluator on every case. Requires UISCE_TEST_DSN.
//
// Expected-error cases are skipped: in SQL, an unresolvable column is a
// COMPILE-time error (stronger than Go's runtime error) — a documented
// capability-matrix divergence, not a parity failure.
func TestSQLConformanceParity(t *testing.T) {
	dsn := os.Getenv("UISCE_TEST_DSN")
	if dsn == "" {
		t.Skip("UISCE_TEST_DSN not set")
	}
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	raw, err := os.ReadFile("../testdata/conformance_cases.json")
	if err != nil {
		// try relative path if run from vm package or root
		raw, err = os.ReadFile("testdata/conformance_cases.json")
		if err != nil {
			t.Fatal(err)
		}
	}
	var suite struct {
		Cases []ConformanceCase `json:"cases"`
	}
	if err := json.Unmarshal(raw, &suite); err != nil {
		t.Fatal(err)
	}

	for _, tc := range suite.Cases {
		tc := tc
		t.Run(tc.Name, func(t *testing.T) {
			if tc.ExpectedError {
				t.Skip("error cases are compile-time in SQL (capability matrix)")
			}
			var node RuleNode
			if err := json.Unmarshal(tc.RuleAST, &node); err != nil {
				t.Fatalf("fixture unparseable: %v", err)
			}

			// Build a temp table from the case's input keys (missing key -> NULL col).
			cols := make([]string, 0, len(tc.Input))
			for k, v := range tc.Input {
				switch v.(type) {
				case string:
					cols = append(cols, fmt.Sprintf(`"%s" text`, k))
				default:
					cols = append(cols, fmt.Sprintf(`"%s" numeric`, k))
				}
			}
			sort.Strings(cols)

			tblName := fmt.Sprintf("cf_%d", os.Getpid())
			_, _ = db.Exec(fmt.Sprintf(`DROP TABLE IF EXISTS %s`, tblName))
			// tenant_id is always present (Layer 2); input fields may be absent.
			ddl := fmt.Sprintf(`CREATE TEMP TABLE %s (tenant_id text%s)`, tblName, func() string {
				if len(cols) > 0 {
					return ", " + strings.Join(cols, ", ")
				}
				return ""
			}())
			if _, err := db.Exec(ddl); err != nil {
				t.Fatalf("ddl: %v", err)
			}
			defer func() {
				_, _ = db.Exec(fmt.Sprintf(`DROP TABLE IF EXISTS %s`, tblName))
			}()

			names, ph, args := []string{"tenant_id"}, []string{"$1"}, []any{"t"}
			i := 1
			for k, v := range tc.Input {
				i++
				names = append(names, fmt.Sprintf(`"%s"`, k))
				ph = append(ph, fmt.Sprintf("$%d", i))
				args = append(args, v)
			}
			if _, err := db.Exec(
				fmt.Sprintf(`INSERT INTO %s (%s) VALUES (%s)`, tblName, strings.Join(names, ","), strings.Join(ph, ",")),
				args...); err != nil {
				t.Fatalf("insert: %v", err)
			}

			b := &ParamBinder{}
			if got := b.Bind("t"); got != "$1" {
				t.Fatalf("tenant must bind to $1, got %s", got)
			}
			resolve := func(field string) (string, error) {
				return `"` + field + `"`, nil
			}
			passSQL, err := CompileToSQL(node, resolve, b)
			if err != nil {
				t.Fatalf("compile: %v", err)
			}

			var passCount int
			err = db.QueryRow(fmt.Sprintf(
				`SELECT COUNT(*) FROM %s WHERE tenant_id = $1 AND COALESCE((%s), FALSE)`, tblName, passSQL),
				b.Args()...).Scan(&passCount)
			if err != nil {
				t.Fatalf("execute: %v (sql=%s)", err, passSQL)
			}
			gotValid := passCount == 1
			if gotValid != *tc.ExpectedValid {
				t.Fatalf("SQL engine disagrees with fixture: got valid=%v, want %v (sql=%s)", gotValid, *tc.ExpectedValid, passSQL)
			}
		})
	}
}
