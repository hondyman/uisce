package vocabulary

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// ResolutionAttempts tracks the outcome of each term-resolution strategy,
// per tenant. This exists because resolveViaAlias/resolveViaSynonym and
// resolveViaEmbedding were previously dead in production (querying edge
// types and a table that were never populated/created, respectively) with
// zero signal — see docs/TERM_RESOLUTION_DESIGN.md §0 and §8.1. A dead or
// failing resolution path should show up here, not require a schema audit
// to discover.
//
// outcome is one of: hit | miss | error. Callers upstream of this package
// (e.g. query.IntentParserWithVocabulary) currently treat a miss and an
// error identically — this metric is what preserves the distinction that
// gets lost at the call site.
var ResolutionAttempts = promauto.NewCounterVec(prometheus.CounterOpts{
	Name: "vocabulary_resolution_attempts_total",
	Help: "Term resolution attempts by tenant, strategy, and outcome (hit|miss|error)",
}, []string{"tenant_id", "strategy", "outcome"})

func recordHit(tenantID, strategy string) {
	ResolutionAttempts.WithLabelValues(tenantID, strategy, "hit").Inc()
}

func recordMiss(tenantID, strategy string) {
	ResolutionAttempts.WithLabelValues(tenantID, strategy, "miss").Inc()
}

func recordError(tenantID, strategy string) {
	ResolutionAttempts.WithLabelValues(tenantID, strategy, "error").Inc()
}
