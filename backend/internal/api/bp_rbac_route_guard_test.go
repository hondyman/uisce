package api_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"net/http"
	"net/http/httptest"
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// The guard for the RBAC routes. It reads the route table from the source, so a route
// added later is checked without anyone remembering to add it to a test. Two rules:
//
//  1. Every handler on the RBAC router verifies the caller's identity and tenant.
//  2. A handler whose route names a resource id also checks that the resource belongs
//     to the tenant, before it reads or writes anything.
//
// Per-handler checks rot when a new handler forgets them; these tests do not forget.

var rbacSourceFiles = []string{"bp_rbac_handlers.go", "bp_rbac_users_handler.go"}

// Each route registered by RBACHandlers.RegisterRoutes: its method, path and handler name.
type rbacRoute struct {
	Method, Path, Handler string
}

var routeCallNames = map[string]string{"Get": http.MethodGet, "Post": http.MethodPost, "Put": http.MethodPut, "Delete": http.MethodDelete}

func parseRBACSource(t *testing.T) (routes []rbacRoute, handlers map[string]*ast.FuncDecl) {
	t.Helper()
	fset := token.NewFileSet()
	handlers = map[string]*ast.FuncDecl{}
	for _, name := range rbacSourceFiles {
		src, err := os.ReadFile(name)
		require.NoError(t, err)
		f, err := parser.ParseFile(fset, name, src, 0)
		require.NoError(t, err)
		for _, decl := range f.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Recv == nil || len(fn.Recv.List) != 1 {
				continue
			}
			if recv, ok := fn.Recv.List[0].Type.(*ast.StarExpr); ok && isIdent(recv.X, "RBACHandlers") {
				handlers[fn.Name.Name] = fn
				if fn.Name.Name == "RegisterRoutes" {
					routes = append(routes, routesOf(fn)...)
				}
			}
		}
	}
	require.NotEmpty(t, routes, "no routes found in the RBAC router")
	return routes, handlers
}

func isIdent(e ast.Expr, name string) bool {
	id, ok := e.(*ast.Ident)
	return ok && id.Name == name
}

func routesOf(register *ast.FuncDecl) []rbacRoute {
	var out []rbacRoute
	ast.Inspect(register.Body, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok || len(call.Args) != 2 {
			return true
		}
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok {
			return true
		}
		method, ok := routeCallNames[sel.Sel.Name]
		if !ok {
			return true
		}
		path, ok := call.Args[0].(*ast.BasicLit)
		if !ok {
			return true
		}
		handler, ok := call.Args[1].(*ast.SelectorExpr)
		if !ok || !isIdent(handler.X, "h") {
			return true
		}
		out = append(out, rbacRoute{Method: method, Path: strings.Trim(path.Value, `"`), Handler: handler.Sel.Name})
		return true
	})
	return out
}

// bodyCalls reports which functions a handler body calls, by selector name.
func bodyCalls(fn *ast.FuncDecl) map[string]bool {
	called := map[string]bool{}
	ast.Inspect(fn.Body, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		switch f := call.Fun.(type) {
		case *ast.SelectorExpr:
			called[f.Sel.Name] = true
		case *ast.Ident:
			called[f.Name] = true
		}
		return true
	})
	return called
}

var idParam = regexp.MustCompile(`\{(roleId|teamId|delegationId|userId)\}`)
var anyParam = regexp.MustCompile(`\{[A-Za-z]+\}`)

var scopeChecks = []string{"authorizeRole", "authorizeTeam", "authorizeDelegation", "authorizeUser", "authorizeOwned"}

func TestRBACRoutes_EveryHandlerChecksIdentityAndTenant(t *testing.T) {
	routes, handlers := parseRBACSource(t)
	for _, rt := range routes {
		fn, ok := handlers[rt.Handler]
		require.True(t, ok, "handler %s for %s %s is not found in the RBAC sources", rt.Handler, rt.Method, rt.Path)
		calls := bodyCalls(fn)
		require.True(t, calls["SecurityContextFromRequest"],
			"%s %s (%s) does not verify the caller's identity and tenant", rt.Method, rt.Path, rt.Handler)
		if idParam.MatchString(rt.Path) {
			found := false
			for _, check := range scopeChecks {
				found = found || calls[check]
			}
			require.True(t, found,
				"%s %s (%s) takes a resource id but does not check that the resource belongs to the tenant", rt.Method, rt.Path, rt.Handler)
		}
	}
}

// Every route refuses an anonymous caller with 401, before it touches the database.
// A route that queries first would hit the database with no expectation set and fail here.
func TestRBACRoutes_AnonymousCallerIsUnauthorizedEverywhere(t *testing.T) {
	routes, _ := parseRBACSource(t)
	for _, rt := range routes {
		path := anyParam.ReplaceAllString(rt.Path, scopeRole)
		t.Run(rt.Method+" "+rt.Path, func(t *testing.T) {
			mock, r := roleScopeRouter(t)
			req := httptest.NewRequest(rt.Method, "/rbac"+path, strings.NewReader(`{}`))
			req.Header.Set("X-Region", "us-east-1")
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)
			require.Equal(t, http.StatusUnauthorized, w.Code, w.Body.String())
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}
