package starlib

import (
	"testing"

	"go.starlark.net/starlark"
)

func TestHelpBuiltin_ReturnsCatalog(t *testing.T) {
	v, err := builtinHelp(nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	d, ok := v.(*starlark.Dict)
	if !ok {
		t.Fatalf("expected dict, got %T", v)
	}

	for _, key := range []string{
		"to_number(x)",
		"field(bo, key, default=None)",
		"date_before(a, b) / date_after(a, b)",
	} {
		_, found, _ := d.Get(starlark.String(key))
		if !found {
			t.Fatalf("expected key %q in help catalog", key)
		}
	}
}
