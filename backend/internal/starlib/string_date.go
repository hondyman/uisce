package starlib

import (
	"fmt"
	"strings"
	"time"

	"go.starlark.net/starlark"
)

func builtinContains(_ *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, _ []starlark.Tuple) (starlark.Value, error) {
	if len(args) != 2 {
		return nil, fmt.Errorf("%s: want 2 args", b.Name())
	}
	h, ok1 := starlark.AsString(args[0])
	n, ok2 := starlark.AsString(args[1])
	if !ok1 || !ok2 {
		return starlark.Bool(false), nil
	}
	return starlark.Bool(strings.Contains(h, n)), nil
}

func builtinStartsWith(_ *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, _ []starlark.Tuple) (starlark.Value, error) {
	if len(args) != 2 {
		return nil, fmt.Errorf("%s: want 2 args", b.Name())
	}
	h, ok1 := starlark.AsString(args[0])
	n, ok2 := starlark.AsString(args[1])
	if !ok1 || !ok2 {
		return starlark.Bool(false), nil
	}
	return starlark.Bool(strings.HasPrefix(h, n)), nil
}

func builtinEndsWith(_ *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, _ []starlark.Tuple) (starlark.Value, error) {
	if len(args) != 2 {
		return nil, fmt.Errorf("%s: want 2 args", b.Name())
	}
	h, ok1 := starlark.AsString(args[0])
	n, ok2 := starlark.AsString(args[1])
	if !ok1 || !ok2 {
		return starlark.Bool(false), nil
	}
	return starlark.Bool(strings.HasSuffix(h, n)), nil
}

func builtinIsBlank(_ *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, _ []starlark.Tuple) (starlark.Value, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("%s: want 1 arg", b.Name())
	}
	if args[0] == starlark.None {
		return starlark.Bool(true), nil
	}
	s, ok := starlark.AsString(args[0])
	if !ok {
		return starlark.Bool(false), nil
	}
	return starlark.Bool(strings.TrimSpace(s) == ""), nil
}

func builtinDateBefore(_ *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, _ []starlark.Tuple) (starlark.Value, error) {
	if len(args) != 2 {
		return nil, fmt.Errorf("%s: want 2 args", b.Name())
	}
	s1, ok1 := starlark.AsString(args[0])
	s2, ok2 := starlark.AsString(args[1])
	if !ok1 || !ok2 {
		return starlark.Bool(false), nil
	}
	d1, err1 := time.Parse(dateLayout, s1)
	d2, err2 := time.Parse(dateLayout, s2)
	if err1 != nil || err2 != nil {
		return starlark.Bool(false), nil
	}
	return starlark.Bool(d1.Before(d2)), nil
}

func builtinDateAfter(_ *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, _ []starlark.Tuple) (starlark.Value, error) {
	if len(args) != 2 {
		return nil, fmt.Errorf("%s: want 2 args", b.Name())
	}
	s1, ok1 := starlark.AsString(args[0])
	s2, ok2 := starlark.AsString(args[1])
	if !ok1 || !ok2 {
		return starlark.Bool(false), nil
	}
	d1, err1 := time.Parse(dateLayout, s1)
	d2, err2 := time.Parse(dateLayout, s2)
	if err1 != nil || err2 != nil {
		return starlark.Bool(false), nil
	}
	return starlark.Bool(d1.After(d2)), nil
}

func builtinToday(_ *starlark.Thread, _ *starlark.Builtin, _ starlark.Tuple, _ []starlark.Tuple) (starlark.Value, error) {
	return starlark.String(time.Now().Format(dateLayout)), nil
}
