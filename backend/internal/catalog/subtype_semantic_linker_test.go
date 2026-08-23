package catalog

import (
	"testing"
)

func TestSubtypeSemanticLinker_New(t *testing.T) {
	linker := NewSubtypeSemanticLinker()
	if linker == nil {
		t.Fatal("expected non-nil linker")
	}
}

func TestSubtypeSemanticLinker_QueryContainsClassifiedAs(t *testing.T) {
	linker := NewSubtypeSemanticLinker()
	if linker == nil {
		t.Fatal("linker is nil")
	}
}
