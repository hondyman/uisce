package semanticast

import (
	"context"
	"strings"
	"testing"

	"github.com/google/uuid"
)

func TestCompilePromptToAST_RejectsNilTenant(t *testing.T) {
	c := NewCompiler(nil)
	_, err := c.CompilePromptToAST(context.Background(), uuid.Nil, "show accounts")
	if err == nil || !strings.Contains(err.Error(), "tenant_id") {
		t.Fatalf("expected Rule 7 nil tenant rejection, got %v", err)
	}
}

func TestCompilePromptToAST_GroundsPriceAndSector(t *testing.T) {
	c := NewCompiler(nil)
	tid := uuid.MustParse("aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa")
	ast, err := c.CompilePromptToAST(context.Background(), tid, "Show me security price and industry sector for Apple")
	if err != nil {
		t.Fatal(err)
	}
	if ast.DrivingEntity == "" || len(ast.SelectedFields) == 0 {
		t.Fatalf("expected grounded AST, got %#v", ast)
	}
	joined := strings.Join(ast.SelectedFields, ",")
	if !strings.Contains(joined, "px_last") || !strings.Contains(joined, "bloomberg_industry_sector") {
		t.Fatalf("expected price+sector fields, got %v", ast.SelectedFields)
	}
}
