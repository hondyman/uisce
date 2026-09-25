package datapipeline

import (
	"path"
	"strings"
)

// NodeType describes one palette entry for the visual editor: what the step
// does in plain language, and whether this environment can run it.
type NodeType struct {
	Type        string `json:"type"`
	Label       string `json:"label"`
	Category    string `json:"category"` // source | step | destination
	Description string `json:"description"`
	Available   bool   `json:"available"`
	Unavailable string `json:"unavailable_reason,omitempty"`
}

// Palette lists the node types in canvas order.
func Palette(d Deps) []NodeType {
	avail := func(ok bool, why string) (bool, string) {
		if ok {
			return true, ""
		}
		return false, why
	}
	files, filesWhy := avail(d.Files != nil, "the file engine is not configured")
	bo, boWhy := avail(d.BO != nil, "business object access is not configured")
	rules, rulesWhy := avail(d.Rules != nil, "the rule engine is not configured")
	staging, stagingWhy := avail(d.StagingDB != nil, "the staging database is not configured")
	return []NodeType{
		{NodeFileSource, "Read a file", "source", "Read a CSV, JSON or Parquet file you uploaded. Define its columns once; every row is checked against them.", files, filesWhy},
		{NodeBOSource, "Read business object", "source", "Read records of a business object, optionally filtered.", bo, boWhy},
		{NodeValidate, "Check required and unique", "step", "Reject rows missing required values, or repeating a key.", true, ""},
		{NodeRuleCheck, "Apply validation rules", "step", "Run rules from the rules catalog. Blocking rules reject the row; warnings are recorded.", rules, rulesWhy},
		{NodeMap, "Map fields", "step", "Rename fields and apply simple transforms (trim, dates, numbers, lookups).", true, ""},
		{NodeBOSink, "Write business object", "destination", "Create or update business object records. Every record goes through the object's rules.", bo, boWhy},
		{NodeStagingSink, "Load staging table", "destination", "Bulk-load rows into a staging table, tracked as a load run (re-running the same run is safe).", staging, stagingWhy},
		{NodeFileSink, "Export a file", "destination", "Write the rows to a CSV, JSON or Parquet file.", files, filesWhy},
	}
}

// GuessFormat picks a file format from its extension (default csv).
func GuessFormat(uri string) string {
	switch strings.ToLower(path.Ext(uri)) {
	case ".json", ".ndjson", ".jsonl":
		return "json"
	case ".parquet", ".pq":
		return "parquet"
	}
	return "csv"
}

// ContractFromProfile proposes a column contract from a file profile. Types
// are the engine's inference; every column starts nullable so the analyst
// tightens rather than loosens.
func ContractFromProfile(p *FileProfile) []Column {
	out := make([]Column, 0, len(p.Columns))
	for _, c := range p.Columns {
		typ := c.Type
		if !validType(typ) {
			typ = "string"
		}
		out = append(out, Column{Name: c.Name, Type: typ})
	}
	return out
}
