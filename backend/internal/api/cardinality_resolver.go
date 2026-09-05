package api

import (
	"database/sql"

	"github.com/hondyman/uisce/backend/internal/cardinality"
)

// CardinalityResolver, JunctionTable and NewCardinalityResolver are aliases
// onto internal/cardinality, which holds the real implementation. That
// package exists as a separate leaf dependency (rather than living here)
// specifically so internal/scanner — which cannot import internal/api
// without an import cycle, since internal/api already transitively depends
// on internal/scanner — can use the same resolver during a live catalog
// scan (see internal/scanner/ansi_scanner.go's use of
// cardinality.NewResolver). Keeping these as aliases here avoids having to
// touch every existing call site in this package.
type CardinalityResolver = cardinality.Resolver
type JunctionTable = cardinality.JunctionTable

// NewCardinalityResolver creates a resolver backed by db.
func NewCardinalityResolver(db *sql.DB) *CardinalityResolver {
	return cardinality.NewResolver(db)
}
