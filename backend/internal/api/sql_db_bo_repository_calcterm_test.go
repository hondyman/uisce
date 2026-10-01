package api

import (
	"testing"

	"github.com/hondyman/uisce/backend/internal/boresolver"
)

// sqlDBBORepository serves the validation-rules path and does not resolve
// calculated terms. It must satisfy boresolver.BORepository and return no
// expressions, so a referenced calc term fails closed in the generator
// ("no preloaded calc term config") instead of compiling to wrong SQL.
func TestSQLDBBORepository_CalcTermExpressionsFailClosed(t *testing.T) {
	var repo boresolver.BORepository = &sqlDBBORepository{}

	exprs, err := repo.GetCalcTermExpressions([]string{"calc-node-1", "calc-node-2"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(exprs) != 0 {
		t.Fatalf("expected no calc-term expressions from this repository, got %d", len(exprs))
	}
}
