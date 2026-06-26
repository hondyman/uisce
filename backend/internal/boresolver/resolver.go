package boresolver

import (
	"fmt"
)

// PhysicalMapping maps a semantic term to a physical column
type PhysicalMapping struct {
	Table  string
	Column string
}

// JoinStep represents a join required to reach a table
type JoinStep struct {
	FromTable string
	ToTable   string
	Condition string // e.g. "t1.id = t2.ref_id"
	Type      string // "LEFT", "INNER"
}

// Resolver maintains the context for resolving a calculation against a BO
type Resolver struct {
	BOID         string
	DrivingTable string
	TermMappings map[string]PhysicalMapping
	JoinPaths    map[string][]JoinStep
	Dialect      Dialect
}

// NewResolver creates a new resolver context
func NewResolver(boID string, drivingTable string, dialect Dialect) *Resolver {
	return &Resolver{
		BOID:         boID,
		DrivingTable: drivingTable,
		TermMappings: make(map[string]PhysicalMapping),
		JoinPaths:    make(map[string][]JoinStep),
		Dialect:      dialect,
	}
}

// AddMapping adds a term mapping
func (r *Resolver) AddMapping(term string, table string, column string) {
	r.TermMappings[term] = PhysicalMapping{Table: table, Column: column}
}

// AddJoinPath adds a join path for a target table
func (r *Resolver) AddJoinPath(targetTable string, steps []JoinStep) {
	r.JoinPaths[targetTable] = steps
}

// ResolveTerm looks up a term and returns its physical location and necessary joins
func (r *Resolver) ResolveTerm(term string) (PhysicalMapping, []JoinStep, error) {
	// 1. Check direct mappings
	mapping, ok := r.TermMappings[term]
	if !ok {
		// Could handle "table.column" passthrough if semantic layer permits,
		// but strict semantic layer requires mapping.
		return PhysicalMapping{}, nil, fmt.Errorf("unknown semantic term: %s", term)
	}

	// 2. Determine join path to the mapped table
	// If the table is the driving table, no joins needed.
	if mapping.Table == r.DrivingTable {
		return mapping, nil, nil
	}

	// 3. Lookup pre-calculated join path
	joins, ok := r.JoinPaths[mapping.Table]
	if !ok {
		// If no path found and not driving table, checking if it's implicitly joinable?
		// For now, strict error.
		return mapping, nil, fmt.Errorf("no join path found for table: %s", mapping.Table)
	}

	return mapping, joins, nil
}
