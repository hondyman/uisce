package profiler

import (
	"github.com/hondyman/semlayer/backend/internal/profiler/helpers"
)

// retain_unused_symbols.go
// This file intentionally references exported symbols from the helpers package
// to prevent staticcheck U1000 (unused symbol) warnings for symbols that are
// intentionally kept for examples or external wiring. Keep this file minimal
// and safe — it only creates harmless references at init time.
func init() {
	// Reference function values and types so staticcheck treats them as used.
	_ = helpers.FormatValueForBloom
	_ = helpers.ComputeProfile
	_ = helpers.CreateBloomFilter
	_ = helpers.InferPatterns
	var _ helpers.ColumnProfile
}
