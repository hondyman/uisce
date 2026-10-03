package archguard

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"
)

// This guard enforces ADR-030: a tenant's own database is reached only through
// internal/tenantdb, which resolves the datasource, checks the caller's tenant, takes its
// credential from the secrets store, and asserts the connection landed on the right database.
//
// It cannot tell by itself which opener serves a tenant datasource, so it inventories EVERY call
// that opens a database connection in non-test code and requires each file to be classified.
// A new opener, or more openers in a classified file, fails CI until someone decides what it
// is. The inventory is the useful part: it is the list the tenantdb migration works through.

type openerKind string

const (
	// kindTenantDB is the router itself: the one place allowed to open a tenant database.
	kindTenantDB openerKind = "tenantdb"
	// kindControlPlane opens alpha or another platform database from the process's own
	// configuration (DATABASE_URL and friends), never from a tenant's datasource row.
	kindControlPlane openerKind = "control-plane"
	// kindWarehouse opens StarRocks (MySQL protocol).
	kindWarehouse openerKind = "warehouse"
	// kindProvisioning connects to the cluster as an administrator to create or drop tenant
	// databases and roles; it is the provisioning saga, not data access.
	kindProvisioning openerKind = "provisioning-admin"
	// kindTool is a command-line tool or test support that is not served: seeds, one-off
	// verifiers, demos, migration commands.
	kindTool openerKind = "tool"
	// kindTenantDatasource opens a tenant's datasource, or the database an app keeps its data
	// in, by some path other than tenantdb. Each needs an Until: it has to move behind
	// tenantdb (or a source-connector sibling for non-Postgres sources), or be retired.
	kindTenantDatasource openerKind = "tenant-datasource"
)

type opener struct {
	Kind   openerKind
	Max    int    // most opener calls this file may contain
	Reason string // why it is what it is
	Until  string // required for kindTenantDatasource: the slice that removes it
}

// openerCalls are the functions that open a connection, by import path and function name.
// sqlx.NewDb and sql.OpenDB wrap or adopt an existing handle; NewDb is not an opener, OpenDB is
// (it takes a connector that dials).
var openerCalls = map[string]map[string]bool{
	"database/sql":                    {"Open": true, "OpenDB": true},
	"github.com/jmoiron/sqlx":         {"Open": true, "Connect": true, "MustConnect": true, "MustOpen": true, "ConnectContext": true},
	"github.com/jackc/pgx/v5":         {"Connect": true, "ConnectConfig": true},
	"github.com/jackc/pgx/v5/pgxpool": {"New": true, "NewWithConfig": true},
	"github.com/jackc/pgx/v5/stdlib":  {"OpenDB": true},
	"gorm.io/gorm":                    {"Open": true},
}

func countOpeners(t *testing.T, root string) map[string]int {
	t.Helper()
	counts := map[string]int{}
	fset := token.NewFileSet()
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			switch d.Name() {
			case "vendor", "node_modules", ".git", "testdata":
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		f, perr := parser.ParseFile(fset, path, nil, 0)
		if perr != nil {
			return nil // a file that does not parse is the compiler's problem, not this guard's
		}
		local := map[string]map[string]bool{} // local import name -> opener functions
		for _, imp := range f.Imports {
			ip, _ := strconv.Unquote(imp.Path.Value)
			fns, ok := openerCalls[ip]
			if !ok {
				continue
			}
			name := filepath.Base(ip)
			if ip == "github.com/jackc/pgx/v5" {
				name = "pgx"
			}
			if imp.Name != nil {
				name = imp.Name.Name
			}
			local[name] = fns
		}
		if len(local) == 0 {
			return nil
		}
		n := 0
		ast.Inspect(f, func(node ast.Node) bool {
			call, ok := node.(*ast.CallExpr)
			if !ok {
				return true
			}
			sel, ok := call.Fun.(*ast.SelectorExpr)
			if !ok {
				return true
			}
			if id, ok := sel.X.(*ast.Ident); ok && local[id.Name][sel.Sel.Name] {
				n++
			}
			return true
		})
		if n > 0 {
			rel, _ := filepath.Rel(root, path)
			counts[filepath.ToSlash(rel)] = n
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return counts
}

func TestEveryDatabaseOpenerIsClassified(t *testing.T) {
	counts := countOpeners(t, "../..")

	var problems []string
	for file, n := range counts {
		o, ok := openerInventory[file]
		if !ok {
			problems = append(problems, fmt.Sprintf(
				"%s opens %d database connection(s) and is not classified. If it serves a tenant's datasource, "+
					"use internal/tenantdb. Otherwise add it to openerInventory with its kind and a reason.", file, n))
			continue
		}
		if n > o.Max {
			problems = append(problems, fmt.Sprintf("%s now opens %d connections; its classification allows %d (%s)", file, n, o.Max, o.Kind))
		}
	}
	for file, o := range openerInventory {
		if counts[file] == 0 {
			if o.Until != "" {
				t.Logf("stale inventory entry (file no longer opens a connection; remove it): %s", file)
			} else {
				problems = append(problems, fmt.Sprintf("%s is in openerInventory but no longer opens a connection; remove the entry", file))
			}
		}
	}
	sort.Strings(problems)
	if len(problems) > 0 {
		t.Fatalf("database opener inventory is out of date (ADR-030):\n  %s", strings.Join(problems, "\n  "))
	}
}

func TestInventoryEntriesAreHonest(t *testing.T) {
	for file, o := range openerInventory {
		if o.Reason == "" {
			t.Errorf("%s: every classification needs a reason", file)
		}
		if o.Max <= 0 {
			t.Errorf("%s: Max must be positive", file)
		}
		switch o.Kind {
		case kindTenantDatasource:
			if o.Until == "" {
				t.Errorf("%s: a tenant-datasource opener must name the slice that moves it behind tenantdb", file)
			}
		case kindTenantDB, kindControlPlane, kindWarehouse, kindProvisioning, kindTool:
			if o.Until != "" {
				t.Errorf("%s: only tenant-datasource entries carry an Until", file)
			}
		default:
			t.Errorf("%s: unknown kind %q", file, o.Kind)
		}
		if _, err := os.Stat(filepath.Join("../..", filepath.FromSlash(file))); err != nil {
			t.Errorf("%s: file does not exist", file)
		}
	}
}

// Only the router may open a tenant database for data access, so exactly one entry is the
// router and it lives in internal/tenantdb.
func TestOnlyTenantdbIsTheRouter(t *testing.T) {
	for file, o := range openerInventory {
		if o.Kind == kindTenantDB && !strings.HasPrefix(file, "internal/tenantdb/") {
			t.Errorf("%s is classified as the tenant router but is outside internal/tenantdb", file)
		}
		if strings.HasPrefix(file, "internal/tenantdb/") && o.Kind != kindTenantDB {
			t.Errorf("%s is in internal/tenantdb and must be classified as the router", file)
		}
	}
}
