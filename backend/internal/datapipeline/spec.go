// Package datapipeline is the config-first data loader/exporter.
//
// A pipeline is a versioned JSON document (Spec) stored in
// data_pipeline_definitions.dag_json. The UI edits it, the API validates it,
// MCP tools read/draft it, and the Temporal interpreter runs it. Nothing is
// generated as code.
//
// File parsing, schema inference and file export are delegated to the
// DataFusion engine (see FileEngine). Business-object reads/writes go through
// the BO records API (see BOClient), never direct SQL.
package datapipeline

import (
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
)

var stagingTableRE = regexp.MustCompile(`^staging\.[a-z_][a-z0-9_]*$`)

// SpecVersion is bumped on breaking spec changes.
const SpecVersion = 1

// Node types.
const (
	NodeFileSource  = "file_source"
	NodeBOSource    = "bo_source"
	NodeValidate    = "validate"
	NodeRuleCheck   = "rule_check"
	NodeMap         = "map"
	NodeBOSink      = "bo_sink"
	NodeStagingSink = "staging_sink"
	NodeFileSink    = "file_sink"
)

// Error policies (data_pipeline_definitions.error_policy).
const (
	ErrorPolicySkipAndLog = "skip_and_log"
	ErrorPolicyFailFast   = "fail_fast"
)

// Spec is the pipeline document.
type Spec struct {
	Version     int    `json:"version"`
	Nodes       []Node `json:"nodes"`
	Edges       []Edge `json:"edges"`
	BatchSize   int    `json:"batch_size,omitempty"`
	ErrorPolicy string `json:"error_policy,omitempty"`
}

// Node is one step. Config is validated per node type (see ConfigSchema).
type Node struct {
	ID       string          `json:"id"`
	Type     string          `json:"type"`
	Label    string          `json:"label,omitempty"`
	Config   json.RawMessage `json:"config"`
	Position *Position       `json:"position,omitempty"` // canvas only
}

type Position struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

type Edge struct {
	From string `json:"from"`
	To   string `json:"to"`
}

// --- per-node configs -------------------------------------------------

// FileSourceConfig reads a file via the DataFusion engine.
type FileSourceConfig struct {
	URI       string `json:"uri"`                 // s3://, file://, or upload:// reference
	Format    string `json:"format"`              // csv | json | parquet
	Delimiter string `json:"delimiter,omitempty"` // csv only; "|" for FactSet
	HasHeader *bool  `json:"has_header,omitempty"`
	// Columns is the file contract. Empty means "infer" (preview/profile only;
	// a run with an empty contract is rejected).
	Columns []Column `json:"columns,omitempty"`
}

// Column is one contract column.
type Column struct {
	Name     string   `json:"name"`
	Type     string   `json:"type"` // string|int|float|decimal|bool|date|timestamp
	Nullable *bool    `json:"nullable,omitempty"`
	Enum     []string `json:"enum,omitempty"`
}

// BOSourceConfig reads records from a business object via the BO API.
type BOSourceConfig struct {
	BOKey   string      `json:"bo_key"`
	Filters []Condition `json:"filters,omitempty"` // all must hold
	Limit   int         `json:"limit,omitempty"`
}

// ValidateConfig checks rows against simple declarative rules.
type ValidateConfig struct {
	Required []string `json:"required,omitempty"`
	Unique   []string `json:"unique,omitempty"` // composite key within a run
}

// RuleCheckConfig runs validation rules from the tenant's rules catalog (own
// and inherited core rules, evaluated on the rule engine) against every row.
// Analysts pick rules by id from a catalog picker; no expression is written
// here. A BLOCK failure rejects the row; a WARN failure keeps it and is
// recorded as a warning.
type RuleCheckConfig struct {
	RuleIDs []string `json:"rule_ids"`
}

// MapConfig maps source fields to target fields.
type MapConfig struct {
	Fields []FieldMap `json:"fields"`
	// DropUnmapped drops columns with no mapping (default true).
	KeepUnmapped bool `json:"keep_unmapped,omitempty"`
}

// FieldMap maps From -> To with an optional transform.
type FieldMap struct {
	From      string            `json:"from"`
	To        string            `json:"to"`
	Transform string            `json:"transform,omitempty"` // trim|upper|lower|to_date|to_number|lookup
	Lookup    map[string]string `json:"lookup,omitempty"`    // for transform=lookup
	Default   *string           `json:"default,omitempty"`
}

// BOSinkConfig writes records to a business object via the BO API.
type BOSinkConfig struct {
	BOKey string `json:"bo_key"`
	// Mode: create (POST each), upsert (match on KeyFields: PUT if found).
	Mode      string   `json:"mode,omitempty"`
	KeyFields []string `json:"key_fields,omitempty"`
	// DryRun reports what would be written without calling the API.
	DryRun bool `json:"dry_run,omitempty"`
}

// StagingSinkConfig bulk-loads rows into a staging.* table (no business
// object involved) using COPY. Every load is tracked by a staging._load_run
// row: (tenant, source_cd, domain, run_ref) is unique, so re-running a
// completed run_ref is a no-op. The target table must have _load_run_id,
// _source_row_num and tenant_id columns (see db/manual_fixes/008_staging_product.sql).
type StagingSinkConfig struct {
	Table    string `json:"table"`     // must be staging.<name>
	SourceCd string `json:"source_cd"` // e.g. FACTSET
	Domain   string `json:"domain"`    // e.g. PRODUCT
	// RunRef defaults to the pipeline run id when empty; set it for
	// file-derived idempotency (e.g. "20260924-01").
	RunRef string `json:"run_ref,omitempty"`
	// Columns maps row field -> staging column. Empty means identity for all
	// fields present in the rows.
	Columns map[string]string `json:"columns,omitempty"`
}

// FileSinkConfig exports rows to a file via the DataFusion engine.
type FileSinkConfig struct {
	URI       string `json:"uri"`
	Format    string `json:"format"` // csv | json | parquet
	Delimiter string `json:"delimiter,omitempty"`
}

// --- validation -------------------------------------------------------

// Validate checks structure: unique ids, known types, edges reference nodes,
// acyclic, at least one source and one sink, node configs parse and are
// complete. It returns every problem, not just the first, so the UI can mark
// all offending nodes.
func (s *Spec) Validate() []error {
	var errs []error
	add := func(f string, a ...any) { errs = append(errs, fmt.Errorf(f, a...)) }

	if s.Version != SpecVersion {
		add("unsupported spec version %d (want %d)", s.Version, SpecVersion)
	}
	if s.ErrorPolicy != "" && s.ErrorPolicy != ErrorPolicySkipAndLog && s.ErrorPolicy != ErrorPolicyFailFast {
		add("unknown error_policy %q", s.ErrorPolicy)
	}

	ids := map[string]*Node{}
	var sources, sinks int
	for i := range s.Nodes {
		n := &s.Nodes[i]
		if n.ID == "" {
			add("node %d: missing id", i)
			continue
		}
		if _, dup := ids[n.ID]; dup {
			add("node %q: duplicate id", n.ID)
			continue
		}
		ids[n.ID] = n
		switch n.Type {
		case NodeFileSource, NodeBOSource:
			sources++
		case NodeBOSink, NodeFileSink, NodeStagingSink:
			sinks++
		case NodeValidate, NodeMap, NodeRuleCheck:
		default:
			add("node %q: unknown type %q", n.ID, n.Type)
			continue
		}
		for _, e := range validateNodeConfig(n) {
			add("node %q: %v", n.ID, e)
		}
	}
	if sources == 0 {
		add("pipeline needs at least one source node")
	}
	if sinks == 0 {
		add("pipeline needs at least one sink node")
	}

	in := map[string]int{}
	out := map[string]int{}
	for _, e := range s.Edges {
		if ids[e.From] == nil {
			add("edge %s->%s: unknown from node", e.From, e.To)
			continue
		}
		if ids[e.To] == nil {
			add("edge %s->%s: unknown to node", e.From, e.To)
			continue
		}
		out[e.From]++
		in[e.To]++
	}
	for id, n := range ids {
		switch n.Type {
		case NodeFileSource, NodeBOSource:
			if in[id] > 0 {
				add("node %q: source cannot have inputs", id)
			}
			if out[id] == 0 {
				add("node %q: source has no outgoing edge", id)
			}
		default:
			if in[id] == 0 {
				add("node %q: no incoming edge", id)
			}
			if in[id] > 1 {
				add("node %q: multiple inputs are not supported", id)
			}
		}
	}
	if len(errs) == 0 {
		if _, err := s.TopoOrder(); err != nil {
			add("%v", err)
		}
	}
	return errs
}

// TopoOrder returns node ids in dependency order (Kahn), deterministic.
func (s *Spec) TopoOrder() ([]string, error) {
	indeg := map[string]int{}
	adj := map[string][]string{}
	for _, n := range s.Nodes {
		indeg[n.ID] = 0
	}
	for _, e := range s.Edges {
		adj[e.From] = append(adj[e.From], e.To)
		indeg[e.To]++
	}
	var ready []string
	for id, d := range indeg {
		if d == 0 {
			ready = append(ready, id)
		}
	}
	sort.Strings(ready)
	var order []string
	for len(ready) > 0 {
		id := ready[0]
		ready = ready[1:]
		order = append(order, id)
		next := adj[id]
		sort.Strings(next)
		for _, t := range next {
			indeg[t]--
			if indeg[t] == 0 {
				ready = append(ready, t)
			}
		}
	}
	if len(order) != len(s.Nodes) {
		return nil, fmt.Errorf("pipeline contains a cycle")
	}
	return order, nil
}

func validateNodeConfig(n *Node) []error {
	var errs []error
	add := func(f string, a ...any) { errs = append(errs, fmt.Errorf(f, a...)) }
	decode := func(v any) bool {
		if len(n.Config) == 0 {
			add("missing config")
			return false
		}
		if err := json.Unmarshal(n.Config, v); err != nil {
			add("invalid config: %v", err)
			return false
		}
		return true
	}
	validFormat := func(f string) bool { return f == "csv" || f == "json" || f == "parquet" }

	switch n.Type {
	case NodeFileSource:
		var c FileSourceConfig
		if !decode(&c) {
			return errs
		}
		if c.URI == "" {
			add("uri is required")
		}
		if !validFormat(c.Format) {
			add("format must be csv, json or parquet")
		}
		for _, col := range c.Columns {
			if col.Name == "" || !validType(col.Type) {
				add("column %q: invalid name or type %q", col.Name, col.Type)
			}
		}
	case NodeBOSource:
		var c BOSourceConfig
		if !decode(&c) {
			return errs
		}
		if c.BOKey == "" {
			add("bo_key is required")
		}
		errs = append(errs, validateConditions(c.Filters)...)
	case NodeValidate:
		var c ValidateConfig
		decode(&c)
	case NodeRuleCheck:
		var c RuleCheckConfig
		if decode(&c) && len(c.RuleIDs) == 0 {
			add("pick at least one rule")
		}
	case NodeMap:
		var c MapConfig
		if !decode(&c) {
			return errs
		}
		if len(c.Fields) == 0 {
			add("at least one field mapping is required")
		}
		for _, f := range c.Fields {
			if f.From == "" || f.To == "" {
				add("field mapping needs from and to")
			}
			switch f.Transform {
			case "", "trim", "upper", "lower", "to_date", "to_number":
			case "lookup":
				if len(f.Lookup) == 0 {
					add("field %q: lookup transform needs a lookup table", f.From)
				}
			default:
				add("field %q: unknown transform %q", f.From, f.Transform)
			}
		}
	case NodeBOSink:
		var c BOSinkConfig
		if !decode(&c) {
			return errs
		}
		if c.BOKey == "" {
			add("bo_key is required")
		}
		switch c.Mode {
		case "", "create":
		case "upsert":
			if len(c.KeyFields) == 0 {
				add("upsert mode requires key_fields")
			}
		default:
			add("mode must be create or upsert")
		}
	case NodeStagingSink:
		var c StagingSinkConfig
		if !decode(&c) {
			return errs
		}
		if !stagingTableRE.MatchString(c.Table) {
			add("table must be staging.<name> (lowercase letters, digits, underscores)")
		}
		if c.SourceCd == "" || c.Domain == "" {
			add("source_cd and domain are required")
		}
	case NodeFileSink:
		var c FileSinkConfig
		if !decode(&c) {
			return errs
		}
		if c.URI == "" {
			add("uri is required")
		}
		if !validFormat(c.Format) {
			add("format must be csv, json or parquet")
		}
	}
	return errs
}

func validType(t string) bool {
	switch t {
	case "string", "int", "float", "decimal", "bool", "date", "timestamp":
		return true
	}
	return false
}
