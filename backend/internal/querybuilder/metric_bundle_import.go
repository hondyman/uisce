package querybuilder

import (
	"fmt"
	"strings"
)

// ImportMetricBundle is the ingestion boundary for metric bundles, and the
// FIRST consumer of ValidateMetricExpression outside compilation.
//
// That placement is the point. A bundle can arrive by migration, seed, import
// or the 8.3 golden corpus, and none of those paths go through CompileMetric.
// Without validation here, an ambiguous metric could be stored and only
// rejected when it was first executed - at query or deploy time, far from
// whoever authored it. Rejecting at ingestion is where the error is still
// actionable. See ADR-026.
//
// The exporter is ExportMetricBundle; this is its counterpart. There is
// deliberately no parallel format, so a bundle round-trips.
//
// existing may be nil. It carries metrics that already exist and may be
// referenced without being restated in the bundle - a derived metric naming a
// base metric the tenant already has, for instance.
func ImportMetricBundle(bundle *MetricBundle, existing map[string]MetricDefinition) ([]MetricDefinition, error) {
	if bundle == nil {
		return nil, fmt.Errorf("%w: bundle is nil", ErrInvalidMetricBundle)
	}
	if bundle.SchemaVersion != MetricBundleSchemaVersion {
		return nil, fmt.Errorf("%w: unsupported schema version %q, this build reads %q",
			ErrInvalidMetricBundle, bundle.SchemaVersion, MetricBundleSchemaVersion)
	}

	// Combined lookup: what the bundle brings, plus what already exists.
	known := make(map[string]MetricDefinition, len(bundle.Metrics)+len(existing))
	for id, m := range existing {
		known[normalizeMetricID(id)] = m
	}

	// Pass 1: shape. Every metric is validated on its own terms, and duplicate
	// IDs are rejected rather than silently last-wins - a bundle carrying two
	// definitions for one ID is ambiguous in the same way a bare ratio was.
	imported := make([]MetricDefinition, 0, len(bundle.Metrics))
	for _, m := range bundle.Metrics {
		id := normalizeMetricID(m.ID)
		if id == "" {
			return nil, fmt.Errorf("%w: bundle contains a metric with no id", ErrInvalidMetricBundle)
		}
		if err := ValidateMetricExpression(m); err != nil {
			return nil, fmt.Errorf("%w: metric %q is not importable: %v", ErrInvalidMetricBundle, m.ID, err)
		}
		if _, dup := known[id]; dup {
			return nil, fmt.Errorf("%w: metric %q appears twice (in the bundle and in the existing set)", ErrInvalidMetricBundle, m.ID)
		}
		known[id] = m
		imported = append(imported, m)
	}

	// Pass 2: content. The exported content hash is recomputed and must match.
	// A mismatch means the bundle was edited after export - which is exactly
	// what a hand-tuned "golden" fixture does, and exactly what would let the
	// corpus drift away from the semantics it claims to protect. When the hash
	// is absent it is filled in, so a minimal bundle still imports.
	for i := range imported {
		computed := ComputeMetricContentHash(imported[i])
		if imported[i].ContentHash != "" && imported[i].ContentHash != computed {
			return nil, fmt.Errorf("%w: metric %q content hash does not match its content "+
				"(bundle says %s, content computes %s) - the bundle was edited after export, or was "+
				"exported by a compiler whose hashing differs from this one",
				ErrInvalidMetricBundle, imported[i].ID, imported[i].ContentHash, computed)
		}
		imported[i].ContentHash = computed
	}

	// Pass 3: references. A derived metric must be able to name its operands -
	// in the bundle or already present. A dangling base is not a definition
	// anyone can evaluate.
	for _, m := range imported {
		for _, baseID := range m.Expression.BaseMetricIDs {
			if _, ok := known[normalizeMetricID(baseID)]; !ok {
				return nil, fmt.Errorf("%w: metric %q references base metric %q, which is neither in the bundle nor already present",
					ErrInvalidMetricBundle, m.ID, baseID)
			}
		}
	}

	// Pass 4: cycles. A cycle compiles to infinite recursion; catching it at
	// ingestion names the members instead of blowing the stack later.
	if cycle := findMetricCycle(imported, known); len(cycle) > 0 {
		return nil, fmt.Errorf("%w: metric dependency cycle: %s",
			ErrInvalidMetricBundle, strings.Join(cycle, " -> "))
	}

	return imported, nil
}

// findMetricCycle returns the first dependency cycle it finds, as a path, or
// nil. It walks only derived metrics, since only they have base references.
func findMetricCycle(imported []MetricDefinition, known map[string]MetricDefinition) []string {
	const (
		white = 0 // unvisited
		grey  = 1 // on the current path
		black = 2 // fully explored
	)
	state := make(map[string]int, len(known))
	var path []string

	var walk func(id string) []string
	walk = func(id string) []string {
		switch state[id] {
		case grey:
			// Back edge: the cycle starts at the first grey entry.
			for i := range path {
				if path[i] == id {
					cycle := append([]string(nil), path[i:]...)
					return append(cycle, id)
				}
			}
			return []string{id, id}
		case black:
			return nil
		}
		state[id] = grey
		path = append(path, id)
		for _, baseID := range known[id].Expression.BaseMetricIDs {
			next := normalizeMetricID(baseID)
			if _, ok := known[next]; !ok {
				continue // dangling refs are rejected in pass 3
			}
			if cycle := walk(next); cycle != nil {
				return cycle
			}
		}
		path = path[:len(path)-1]
		state[id] = black
		return nil
	}

	for _, m := range imported {
		if cycle := walk(normalizeMetricID(m.ID)); cycle != nil {
			return cycle
		}
	}
	return nil
}

// normalizeMetricID lowercases and trims a metric ID for lookup, matching how
// cube validation and the DDL generator normalize before resolving.
func normalizeMetricID(id string) string {
	return strings.ToLower(strings.TrimSpace(id))
}
