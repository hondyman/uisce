package starlib

import (
	"fmt"
	"strings"

	"go.starlark.net/starlark"
)

func Lib() starlark.StringDict {
	return starlark.StringDict{
		"help":        starlark.NewBuiltin("help", builtinHelp),
		"coalesce":    starlark.NewBuiltin("coalesce", builtinCoalesce),
		"to_string":   starlark.NewBuiltin("to_string", builtinToString),
		"to_number":   starlark.NewBuiltin("to_number", builtinToNumber),
		"to_bool":     starlark.NewBuiltin("to_bool", builtinToBool),
		"eq":          starlark.NewBuiltin("eq", builtinEq),
		"ne":          starlark.NewBuiltin("ne", builtinNe),
		"gt":          starlark.NewBuiltin("gt", builtinGt),
		"ge":          starlark.NewBuiltin("ge", builtinGe),
		"lt":          starlark.NewBuiltin("lt", builtinLt),
		"le":          starlark.NewBuiltin("le", builtinLe),
		"contains":    starlark.NewBuiltin("contains", builtinContains),
		"startswith":  starlark.NewBuiltin("startswith", builtinStartsWith),
		"endswith":    starlark.NewBuiltin("endswith", builtinEndsWith),
		"lower":       starlark.NewBuiltin("lower", builtinLower),
		"upper":       starlark.NewBuiltin("upper", builtinUpper),
		"trim":        starlark.NewBuiltin("trim", builtinTrim),
		"is_blank":    starlark.NewBuiltin("is_blank", builtinIsBlank),
		"regex_match": starlark.NewBuiltin("regex_match", builtinRegexMatch),
		"date_before": starlark.NewBuiltin("date_before", builtinDateBefore),
		"date_after":  starlark.NewBuiltin("date_after", builtinDateAfter),
		"today":       starlark.NewBuiltin("today", builtinToday),
		"hours":       starlark.NewBuiltin("hours", builtinHours),
		"days":        starlark.NewBuiltin("days", builtinDays),
		"minutes":     starlark.NewBuiltin("minutes", builtinMinutes),
	}
}

func LibWithCtx(ctxDict *starlark.Dict) starlark.StringDict {
	lib := Lib()
	lib["ctx"] = ctxDict
	lib["get"] = starlark.NewBuiltin("get", makeGetBuiltin(ctxDict))
	lib["exists"] = starlark.NewBuiltin("exists", makeExistsBuiltin(ctxDict))
	lib["get_path"] = starlark.NewBuiltin("get_path", makeGetPathBuiltin(ctxDict))
	lib["exists_path"] = starlark.NewBuiltin("exists_path", makeExistsPathBuiltin(ctxDict))
	lib["field"] = starlark.NewBuiltin("field", makeFieldBuiltin(ctxDict))
	lib["num_field"] = starlark.NewBuiltin("num_field", makeNumFieldBuiltin(ctxDict))
	lib["bool_field"] = starlark.NewBuiltin("bool_field", makeBoolFieldBuiltin(ctxDict))
	lib["F"] = starlark.NewBuiltin("F", makeFBuiltin(ctxDict))
	return lib
}

func resolveObject(ctxDict *starlark.Dict, obj starlark.Value) (*starlark.Dict, bool) {
	if d, ok := obj.(*starlark.Dict); ok {
		return d, true
	}
	if s, ok := starlark.AsString(obj); ok {
		v, found, _ := ctxDict.Get(starlark.String(s))
		if !found {
			return nil, false
		}
		d, ok := v.(*starlark.Dict)
		if !ok {
			return nil, false
		}
		return d, true
	}
	return nil, false
}

// get(object, field, default=None)
func makeGetBuiltin(ctxDict *starlark.Dict) func(*starlark.Thread, *starlark.Builtin, starlark.Tuple, []starlark.Tuple) (starlark.Value, error) {
	return func(_ *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
		var obj, field, def starlark.Value
		def = starlark.None
		if err := starlark.UnpackArgs(b.Name(), args, kwargs,
			"object", &obj,
			"field", &field,
			"default?", &def,
		); err != nil {
			return nil, err
		}
		fieldStr, ok := starlark.AsString(field)
		if !ok {
			return nil, fmt.Errorf("field must be string")
		}
		d, ok := resolveObject(ctxDict, obj)
		if !ok || d == nil {
			return def, nil
		}
		v, found, _ := d.Get(starlark.String(fieldStr))
		if !found {
			return def, nil
		}
		return v, nil
	}
}

// exists(object, field) -> bool
func makeExistsBuiltin(ctxDict *starlark.Dict) func(*starlark.Thread, *starlark.Builtin, starlark.Tuple, []starlark.Tuple) (starlark.Value, error) {
	return func(_ *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
		var obj, field starlark.Value
		if err := starlark.UnpackArgs(b.Name(), args, kwargs,
			"object", &obj,
			"field", &field,
		); err != nil {
			return nil, err
		}
		fieldStr, ok := starlark.AsString(field)
		if !ok {
			return starlark.Bool(false), nil
		}
		d, ok := resolveObject(ctxDict, obj)
		if !ok || d == nil {
			return starlark.Bool(false), nil
		}
		_, found, _ := d.Get(starlark.String(fieldStr))
		return starlark.Bool(found), nil
	}
}

// coalesce(v1, v2, ..., vn): first non-None, and for strings, first non-empty.
func builtinCoalesce(_ *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, _ []starlark.Tuple) (starlark.Value, error) {
	if len(args) == 0 {
		return starlark.None, nil
	}
	for _, v := range args {
		if v == starlark.None {
			continue
		}
		if s, ok := starlark.AsString(v); ok {
			if s == "" {
				continue
			}
		}
		return v, nil
	}
	return args[len(args)-1], nil
}

func builtinToString(_ *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, _ []starlark.Tuple) (starlark.Value, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("to_string: want 1 arg")
	}
	return starlark.String(args[0].String()), nil
}

func builtinToNumber(_ *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, _ []starlark.Tuple) (starlark.Value, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("to_number: want 1 arg")
	}
	v := args[0]
	switch t := v.(type) {
	case starlark.Int:
		return t, nil
	case starlark.Float:
		return t, nil
	case starlark.String:
		var f float64
		_, err := fmt.Sscan(string(t), &f)
		if err != nil {
			return nil, fmt.Errorf("to_number: cannot parse %q", t)
		}
		return starlark.Float(f), nil
	default:
		return nil, fmt.Errorf("to_number: unsupported type %s", v.Type())
	}
}

func builtinToBool(_ *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, _ []starlark.Tuple) (starlark.Value, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("to_bool: want 1 arg")
	}
	v := args[0]
	switch t := v.(type) {
	case starlark.NoneType:
		return starlark.Bool(false), nil
	case starlark.Bool:
		return t, nil
	case starlark.String:
		return starlark.Bool(len(strings.TrimSpace(string(t))) > 0), nil
	case starlark.Int:
		zero := starlark.MakeInt(0)
		cmp, err := t.Cmp(zero, 0)
		if err != nil {
			return starlark.Bool(true), nil
		}
		return starlark.Bool(cmp != 0), nil
	case starlark.Float:
		return starlark.Bool(float64(t) != 0), nil
	default:
		return starlark.Bool(true), nil
	}
}
