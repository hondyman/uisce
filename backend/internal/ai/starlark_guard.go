package ai

import (
	"fmt"
	"strings"

	"go.starlark.net/starlark"
)

var allowedHelpers = []string{
	"field", "num_field", "bool_field",
	"eq", "ne", "gt", "ge", "lt", "le",
	"contains", "startswith", "endswith",
	"is_blank", "coalesce",
}

// NormalizeStarlarkSnippet wraps raw body code in a function if needed, ensuring indentation.
// If the code already contains a function definition, it returns it as is.
func NormalizeStarlarkSnippet(src string) string {
	if strings.Contains(src, "def ok(") || strings.Contains(src, "def tenant_ok(") {
		return src
	}
	// Indent the source
	lines := strings.Split(src, "\n")
	for i, line := range lines {
		if strings.TrimSpace(line) != "" {
			lines[i] = "\t" + line
		}
	}
	indented := strings.Join(lines, "\n")
	// For compilation by TenantCompiler, we might strictly need the BODY (indented) if using Custom mode?
	// Actually TenantCompiler wraps the body string. So we just need to return the INDENTED body string?
	// No, Validation needs the FULL wrapper.
	// SuggestService should return the BODY (indented) if it's meant to be injected into TenantCompiler?
	// TenantCompiler keeps indentation of the string it injects.
	// So if SuggestService works with TenantCompiler, it should return just the indented lines.
	// BUT ValidateStarlarkSnippet needs the wrapper.

	// Let's make this return the Indented Body if it was raw, or the Full Src if it was def.
	// Wait, if I return just indented body, ValidateStarlarkSnippet needs to re-wrap it?
	return indented
}

func ValidateStarlarkSnippet(src string, predecl starlark.StringDict) error {
	// Quick reject: disallow ctx[...] directly
	if strings.Contains(src, "ctx[") {
		return fmt.Errorf("direct ctx access not allowed")
	}

	thread := &starlark.Thread{Name: "validate_ai"}

	var wrapped string
	if strings.Contains(src, "def ok(") || strings.Contains(src, "def tenant_ok(") {
		wrapped = src
	} else {
		// Assume src is body. We need to indent it for the wrapper.
		// Use NormalizeStarlarkSnippet?
		// If src is unindented body: Normalize returns Indented body.
		indented := NormalizeStarlarkSnippet(src)
		wrapped = fmt.Sprintf(`
def ok(ctx):
%s
`, indented)
	}

	// Ensure predecl has stubs for allowed helpers so validation compiles
	if predecl == nil {
		predecl = starlark.StringDict{}
	}
	for _, helper := range allowedHelpers {
		if _, ok := predecl[helper]; !ok {
			predecl[helper] = starlark.NewBuiltin(helper, func(thread *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
				return starlark.None, nil
			})
		}
	}

	globals, err := starlark.ExecFile(thread, "ai_suggest.star", wrapped, predecl)
	if err != nil {
		return fmt.Errorf("starlark compile error: %w", err)
	}

	// Check for ok or tenant_ok
	hasOk := false
	if v, ok := globals["ok"]; ok {
		if _, isFn := v.(*starlark.Function); isFn {
			hasOk = true
		}
	}
	if !hasOk {
		if v, ok := globals["tenant_ok"]; ok {
			if _, isFn := v.(*starlark.Function); isFn {
				hasOk = true
			}
		}
	}

	if !hasOk {
		return fmt.Errorf("snippet must define ok(ctx) or tenant_ok(ctx)")
	}

	return nil
}
