package starlib

import (
	"sort"
	"strings"

	"go.starlark.net/syntax"
)

// ExtractRequiredFieldPaths performs a best-effort static extraction of BO field paths
// used by Starlark rules.
//
// Currently it detects helper calls of these forms:
// - field("bo", "key") / num_field("bo","key") / bool_field("bo","key")
// - F("bo.key") or F("bo", "key")
//
// It ignores dynamic paths and any calls where arguments are not string literals.
func ExtractRequiredFieldPaths(script string) ([]string, error) {
	f, err := syntax.Parse("rule.star", script, 0)
	if err != nil {
		return nil, err
	}

	set := make(map[string]struct{})
	syntax.Walk(f, func(n syntax.Node) bool {
		call, ok := n.(*syntax.CallExpr)
		if !ok {
			return true
		}
		fn, ok := call.Fn.(*syntax.Ident)
		if !ok {
			return true
		}
		name := fn.Name
		switch name {
		case "field", "num_field", "bool_field":
			if len(call.Args) < 2 {
				return true
			}
			bo, ok1 := stringLiteral(call.Args[0])
			key, ok2 := stringLiteral(call.Args[1])
			if !ok1 || !ok2 {
				return true
			}
			addPath(set, bo+"."+key)
		case "F":
			if len(call.Args) == 1 {
				path, ok := stringLiteral(call.Args[0])
				if !ok {
					return true
				}
				path = strings.TrimSpace(path)
				if path != "" {
					addPath(set, path)
				}
				return true
			}
			if len(call.Args) >= 2 {
				bo, ok1 := stringLiteral(call.Args[0])
				key, ok2 := stringLiteral(call.Args[1])
				if !ok1 || !ok2 {
					return true
				}
				addPath(set, bo+"."+key)
			}
		}
		return true
	})

	if len(set) == 0 {
		return nil, nil
	}
	out := make([]string, 0, len(set))
	for p := range set {
		out = append(out, p)
	}
	sort.Strings(out)
	return out, nil
}

func stringLiteral(expr syntax.Expr) (string, bool) {
	lit, ok := expr.(*syntax.Literal)
	if !ok {
		return "", false
	}
	if lit.Token != syntax.STRING {
		return "", false
	}
	s, ok := lit.Value.(string)
	if !ok {
		return "", false
	}
	return s, true
}

func addPath(set map[string]struct{}, path string) {
	p := strings.TrimSpace(path)
	if p == "" {
		return
	}
	// Basic sanity: require at least one dot.
	if !strings.Contains(p, ".") {
		return
	}
	set[p] = struct{}{}
}
