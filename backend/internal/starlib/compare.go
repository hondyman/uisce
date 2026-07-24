package starlib

import (
	"fmt"

	"go.starlark.net/starlark"
)

func toFloat(v starlark.Value) (float64, bool) {
	switch t := v.(type) {
	case starlark.Int:
		i, ok := t.Int64()
		if !ok {
			return 0, false
		}
		return float64(i), true
	case starlark.Float:
		return float64(t), true
	default:
		return 0, false
	}
}

func cmpValues(a, b starlark.Value) (int, error) {
	if fa, ok := toFloat(a); ok {
		fb, ok2 := toFloat(b)
		if !ok2 {
			return 0, fmt.Errorf("type mismatch %s vs %s", a.Type(), b.Type())
		}
		switch {
		case fa < fb:
			return -1, nil
		case fa > fb:
			return 1, nil
		default:
			return 0, nil
		}
	}
	sa, ok1 := starlark.AsString(a)
	sb, ok2 := starlark.AsString(b)
	if ok1 && ok2 {
		switch {
		case sa < sb:
			return -1, nil
		case sa > sb:
			return 1, nil
		default:
			return 0, nil
		}
	}
	if eq, err := starlark.Equal(a, b); err == nil && eq {
		return 0, nil
	}
	return 0, fmt.Errorf("unsupported comparison %s vs %s", a.Type(), b.Type())
}

func binaryCmp(b *starlark.Builtin, args starlark.Tuple, f func(c int) bool) (starlark.Value, error) {
	var left, right starlark.Value
	if err := starlark.UnpackArgs(b.Name(), args, nil, "left", &left, "right", &right); err != nil {
		return nil, err
	}
	c, err := cmpValues(left, right)
	if err != nil {
		return nil, err
	}
	return starlark.Bool(f(c)), nil
}

func builtinEq(_ *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, _ []starlark.Tuple) (starlark.Value, error) {
	return binaryCmp(b, args, func(c int) bool { return c == 0 })
}

func builtinNe(_ *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, _ []starlark.Tuple) (starlark.Value, error) {
	return binaryCmp(b, args, func(c int) bool { return c != 0 })
}

func builtinGt(_ *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, _ []starlark.Tuple) (starlark.Value, error) {
	return binaryCmp(b, args, func(c int) bool { return c > 0 })
}

func builtinGe(_ *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, _ []starlark.Tuple) (starlark.Value, error) {
	return binaryCmp(b, args, func(c int) bool { return c >= 0 })
}

func builtinLt(_ *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, _ []starlark.Tuple) (starlark.Value, error) {
	return binaryCmp(b, args, func(c int) bool { return c < 0 })
}

func builtinLe(_ *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, _ []starlark.Tuple) (starlark.Value, error) {
	return binaryCmp(b, args, func(c int) bool { return c <= 0 })
}
