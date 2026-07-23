package wealth

// NOTE: tax_calc_engine_adapter.go is temporarily disabled due to type conflicts
// It needs to be updated to use TaxBracketDetail instead of TaxBracket
// The existing implementation has incompatible types between CalcEngine result types
// and the wealth package types. This needs to be resolved before re-enabling.

// TODO: Fix the following issues:
// 1. TaxBracket vs TaxBracketDetail type mismatch
// 2. result.Sources ([]string) vs []map[string]interface{} type mismatch
// 3. CalcEngine result types need to align with wealth types
