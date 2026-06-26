package starlib

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"go.starlark.net/starlark"
)

func builtinHelp(_ *starlark.Thread, _ *starlark.Builtin, _ starlark.Tuple, _ []starlark.Tuple) (starlark.Value, error) {
	// Keep this as a stable, human-oriented catalog for rule authors.
	// Values are plain strings so the result is easy to JSON-ify by callers if desired.
	catalog := map[string]string{
		"ctx":                                       "Injected context dict (ctx.page + ctx.<object>)",
		"ok/message":                                "ok-style rules set ok=<bool> (required) and optional message=\"...\"",
		"get(object, field, default=None)":          "Fetch field from dict or named ctx object",
		"exists(object, field)":                     "True if field exists on dict or named ctx object",
		"get_path(object, path, default=None)":      "Dot-path traversal (supports list indexes like items.0.name)",
		"exists_path(object, path)":                 "True if get_path would find a value",
		"field(bo, key, default=None)":              "BO-aware field accessor (ctx.<bo>[key])",
		"num_field(bo, key, default=None)":          "BO-aware numeric accessor (coerces via to_number)",
		"bool_field(bo, key, default=None)":         "BO-aware boolean accessor (coerces via to_bool)",
		"F(\"bo.key\") / F(bo, key)":                "Shorthand for field()",
		"coalesce(a, b, ...)":                       "First non-None; for strings first non-empty",
		"to_string(x)":                              "Convert to string",
		"to_number(x)":                              "Convert to number (int/float/string)",
		"to_bool(x)":                                "Convert to bool",
		"eq/ne/gt/ge/lt/le(a, b)":                   "Comparison helpers",
		"contains/startswith/endswith(hay, needle)": "String predicate helpers",
		"is_blank(s)":                               "True if None or whitespace",
		"lower/upper/trim(s)":                       "String normalization",
		"regex_match(pattern, s)":                   "Regex match (errors if pattern invalid)",
		"today()":                                   "Current date as YYYY-MM-DD",
		"date_before(a, b) / date_after(a, b)":      "Date comparisons for YYYY-MM-DD",
	}

	d := starlark.NewDict(len(catalog))
	for k, v := range catalog {
		_ = d.SetKey(starlark.String(k), starlark.String(v))
	}
	return d, nil
}

func resolvePath(ctxDict *starlark.Dict, obj starlark.Value, path string) (starlark.Value, bool) {
	path = strings.TrimSpace(path)
	if path == "" {
		return nil, false
	}

	cur, ok := resolveObject(ctxDict, obj)
	if !ok || cur == nil {
		return nil, false
	}

	var current starlark.Value = cur
	for _, seg := range strings.Split(path, ".") {
		seg = strings.TrimSpace(seg)
		if seg == "" {
			return nil, false
		}
		switch node := current.(type) {
		case *starlark.Dict:
			v, found, _ := node.Get(starlark.String(seg))
			if !found {
				return nil, false
			}
			current = v
		case *starlark.List:
			idx, err := strconv.Atoi(seg)
			if err != nil {
				return nil, false
			}
			if idx < 0 || idx >= node.Len() {
				return nil, false
			}
			current = node.Index(idx)
		default:
			return nil, false
		}
	}

	if current == nil {
		return nil, false
	}
	return current, true
}

func builtinLower(_ *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, _ []starlark.Tuple) (starlark.Value, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("%s: want 1 arg", b.Name())
	}
	s, ok := starlark.AsString(args[0])
	if !ok {
		return starlark.String(""), nil
	}
	return starlark.String(strings.ToLower(s)), nil
}

func builtinUpper(_ *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, _ []starlark.Tuple) (starlark.Value, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("%s: want 1 arg", b.Name())
	}
	s, ok := starlark.AsString(args[0])
	if !ok {
		return starlark.String(""), nil
	}
	return starlark.String(strings.ToUpper(s)), nil
}

func builtinTrim(_ *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, _ []starlark.Tuple) (starlark.Value, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("%s: want 1 arg", b.Name())
	}
	s, ok := starlark.AsString(args[0])
	if !ok {
		return starlark.String(""), nil
	}
	return starlark.String(strings.TrimSpace(s)), nil
}

// regex_match(pattern, s) -> bool
// Returns an error if the pattern is invalid.
func builtinRegexMatch(_ *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, _ []starlark.Tuple) (starlark.Value, error) {
	if len(args) != 2 {
		return nil, fmt.Errorf("%s: want 2 args", b.Name())
	}
	pattern, ok1 := starlark.AsString(args[0])
	s, ok2 := starlark.AsString(args[1])
	if !ok1 || !ok2 {
		return starlark.Bool(false), nil
	}
	re, err := regexp.Compile(pattern)
	if err != nil {
		return nil, fmt.Errorf("%s: invalid pattern: %v", b.Name(), err)
	}
	return starlark.Bool(re.MatchString(s)), nil
}

// get_path(object, path, default=None)
// - object: a dict, or a string naming a ctx object (e.g. "account")
// - path: dot-separated (e.g. "sub.field"), with optional numeric segments for list indexes.
func makeGetPathBuiltin(ctxDict *starlark.Dict) func(*starlark.Thread, *starlark.Builtin, starlark.Tuple, []starlark.Tuple) (starlark.Value, error) {
	return func(_ *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
		var obj, pathVal, def starlark.Value
		def = starlark.None
		if err := starlark.UnpackArgs(b.Name(), args, kwargs,
			"object", &obj,
			"path", &pathVal,
			"default?", &def,
		); err != nil {
			return nil, err
		}

		path, ok := starlark.AsString(pathVal)
		if !ok {
			return def, nil
		}
		v, found := resolvePath(ctxDict, obj, path)
		if !found {
			return def, nil
		}
		return v, nil
	}
}

// exists_path(object, path) -> bool
func makeExistsPathBuiltin(ctxDict *starlark.Dict) func(*starlark.Thread, *starlark.Builtin, starlark.Tuple, []starlark.Tuple) (starlark.Value, error) {
	return func(_ *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
		var obj, pathVal starlark.Value
		if err := starlark.UnpackArgs(b.Name(), args, kwargs,
			"object", &obj,
			"path", &pathVal,
		); err != nil {
			return nil, err
		}
		path, ok := starlark.AsString(pathVal)
		if !ok {
			return starlark.Bool(false), nil
		}
		_, found := resolvePath(ctxDict, obj, path)
		return starlark.Bool(found), nil
	}
}

func builtinHours(_ *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, _ []starlark.Tuple) (starlark.Value, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("%s: want 1 arg", b.Name())
	}
	n, err := starlark.AsInt32(args[0])
	if err != nil {
		return nil, fmt.Errorf("%s: expected int", b.Name())
	}
	return starlark.MakeInt(n * 3600), nil
}

func builtinDays(_ *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, _ []starlark.Tuple) (starlark.Value, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("%s: want 1 arg", b.Name())
	}
	n, err := starlark.AsInt32(args[0])
	if err != nil {
		return nil, fmt.Errorf("%s: expected int", b.Name())
	}
	return starlark.MakeInt(n * 86400), nil
}

func builtinMinutes(_ *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, _ []starlark.Tuple) (starlark.Value, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("%s: want 1 arg", b.Name())
	}
	n, err := starlark.AsInt32(args[0])
	if err != nil {
		return nil, fmt.Errorf("%s: expected int", b.Name())
	}
	return starlark.MakeInt(n * 60), nil
}
