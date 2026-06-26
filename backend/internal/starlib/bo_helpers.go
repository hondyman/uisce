package starlib

import (
	"fmt"
	"strings"

	"go.starlark.net/starlark"
)

func makeFieldBuiltin(ctxDict *starlark.Dict) func(*starlark.Thread, *starlark.Builtin, starlark.Tuple, []starlark.Tuple) (starlark.Value, error) {
	return func(_ *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
		var boNameVal, keyVal, def starlark.Value
		def = starlark.String("")

		if err := starlark.UnpackArgs(b.Name(), args, kwargs,
			"bo", &boNameVal,
			"key", &keyVal,
			"default?", &def,
		); err != nil {
			return nil, err
		}

		boName, ok := starlark.AsString(boNameVal)
		if !ok {
			return nil, fmt.Errorf("bo must be string")
		}
		key, ok := starlark.AsString(keyVal)
		if !ok {
			return nil, fmt.Errorf("key must be string")
		}

		boVal, found, _ := ctxDict.Get(starlark.String(boName))
		if !found {
			return def, nil
		}
		boDict, ok := boVal.(*starlark.Dict)
		if !ok {
			return def, nil
		}

		v, found, _ := boDict.Get(starlark.String(key))
		if !found {
			return def, nil
		}
		return v, nil
	}
}

func makeNumFieldBuiltin(ctxDict *starlark.Dict) func(*starlark.Thread, *starlark.Builtin, starlark.Tuple, []starlark.Tuple) (starlark.Value, error) {
	return func(t *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
		v, err := makeFieldBuiltin(ctxDict)(t, b, args, kwargs)
		if err != nil {
			return nil, err
		}
		if v == starlark.None {
			return starlark.MakeInt(0), nil
		}
		num, err := builtinToNumber(t, nil, starlark.Tuple{v}, nil)
		if err != nil {
			return starlark.MakeInt(0), nil
		}
		return num, nil
	}
}

func makeBoolFieldBuiltin(ctxDict *starlark.Dict) func(*starlark.Thread, *starlark.Builtin, starlark.Tuple, []starlark.Tuple) (starlark.Value, error) {
	return func(t *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
		v, err := makeFieldBuiltin(ctxDict)(t, b, args, kwargs)
		if err != nil {
			return nil, err
		}
		if v == starlark.None {
			return starlark.Bool(false), nil
		}
		bv, err := builtinToBool(t, nil, starlark.Tuple{v}, nil)
		if err != nil {
			return starlark.Bool(false), nil
		}
		return bv, nil
	}
}

// F("account.account_type") or F("account", "account_type")
func makeFBuiltin(ctxDict *starlark.Dict) func(*starlark.Thread, *starlark.Builtin, starlark.Tuple, []starlark.Tuple) (starlark.Value, error) {
	return func(_ *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
		if len(kwargs) != 0 {
			return nil, fmt.Errorf("%s: kwargs not supported", b.Name())
		}
		if len(args) == 1 {
			path, ok := starlark.AsString(args[0])
			if !ok {
				return starlark.None, nil
			}
			parts := strings.SplitN(path, ".", 2)
			if len(parts) != 2 {
				return starlark.None, nil
			}
			return makeFieldBuiltin(ctxDict)(nil, b, starlark.Tuple{starlark.String(parts[0]), starlark.String(parts[1])}, nil)
		}
		if len(args) == 2 {
			return makeFieldBuiltin(ctxDict)(nil, b, args, nil)
		}
		return nil, fmt.Errorf("%s: want 1 or 2 args", b.Name())
	}
}
