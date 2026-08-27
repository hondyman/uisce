package ai

import (
	"context"
	"strings"
	"testing"

	"github.com/google/uuid"
)

func TestGrammarCompiler(t *testing.T) {
	compiler := NewGrammarCompiler(nil)

	grammar, err := compiler.CompileTenantASTGrammar(context.Background(), uuid.New())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(grammar, "root ::= ObjectPayload") {
		t.Errorf("grammar missing root rule")
	}

	if !strings.Contains(grammar, "AggOp ::=") {
		t.Errorf("grammar missing AggOp rule")
	}
}
