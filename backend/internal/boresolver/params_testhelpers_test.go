package boresolver

import "testing"

// finalizeParams turns the sentinel-bearing SQL a low-level compile call
// (CompileFilterPredicate, nextParam) returns into this generator's real
// placeholders, exactly as GenerateSQL's final pass does. Tests that call
// those functions directly assert on the finalized text and args.
func finalizeParams(t *testing.T, g *BOSQLGenerator, ctx *GenerationContext, sql string) (string, []interface{}) {
	t.Helper()
	finalSQL, finalArgs, err := renumberParams(sql, g.Dialect, ctx.Args, ensureParamNonce(ctx))
	if err != nil {
		t.Fatalf("renumberParams: %v", err)
	}
	return finalSQL, finalArgs
}
