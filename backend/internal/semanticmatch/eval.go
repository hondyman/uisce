package semanticmatch

import (
	"context"
	"encoding/csv"
	"fmt"
	"hash/fnv"
	"io"
	"os"
	"regexp"
	"sort"
	"strings"
)

// GlossaryRow is one parsed row of Semantic_Definitions.csv before registration.
type GlossaryRow struct {
	ID           string
	Domain       string
	SourceTables []string // cleaned, real table names only
	Aliases      []string // expanded physical column names
	BusinessTerm string
	DataType     string
	Definition   string
}

var (
	tableNameRe  = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_]*$`)
	columnNameRe = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_]*$`)
	typeFragRe   = regexp.MustCompile(`^(\d+\)?\s*(NULL|NOT NULL)?|NULL|NOT NULL|TEXT|\d+)$`)
)

// LoadGlossaryRows parses the glossary CSV without registering anything.
// It tolerates the export artifacts in Semantic_Definitions.csv:
//   - multi-table cells "trade_order; ts_order_fund" (split on ';')
//   - pseudo-entries like "multiple ts_ tables" or "ts_order (PK) and its
//     child tables" (filtered — no eval column can be generated from them)
//   - types split across cells ("NUMERIC(18" + "9) NULL"), repaired below
//   - grouped ranges and synonym lists, expanded via ParseAliases
func LoadGlossaryRows(path string) ([]GlossaryRow, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	r := csv.NewReader(f)
	r.FieldsPerRecord = -1
	r.LazyQuotes = true
	header, err := r.Read()
	if err != nil {
		return nil, err
	}
	idx := headerIndex(header)

	var rows []GlossaryRow
	for {
		rec, err := r.Read()
		if err == io.EOF {
			return rows, nil
		}
		if err != nil {
			return nil, err
		}
		get := func(name string) string {
			i, ok := idx[name]
			if !ok || i >= len(rec) {
				return ""
			}
			return strings.TrimSpace(rec[i])
		}
		if get("business_term") == "" {
			continue
		}

		row := GlossaryRow{
			ID:           get("id"),
			Domain:       get("domain"),
			BusinessTerm: get("business_term"),
			Definition:   get("definition"),
		}
		if di, ok := idx["data_type"]; ok && di < len(rec) {
			row.DataType = repairType(strings.TrimSpace(rec[di]), rec, di)
		}

		for _, s := range strings.Split(get("source_tables"), ";") {
			if s = strings.TrimSpace(s); tableNameRe.MatchString(s) {
				row.SourceTables = append(row.SourceTables, s)
			}
		}
		for _, a := range ParseAliases(get("column_name")) {
			if columnNameRe.MatchString(a) {
				row.Aliases = append(row.Aliases, a)
			}
		}
		if len(row.Aliases) > 0 && len(row.SourceTables) > 0 {
			rows = append(rows, row)
		}
	}
}

// repairType rejoins type fragments that the export split across cells,
// e.g. ["NUMERIC(18", "9) NULL"] -> "NUMERIC(18 9) NULL". ParseTypeFamily
// only needs the leading token, so this is cosmetic-but-tidy; types that
// are already intact pass through untouched.
func repairType(first string, rec []string, startIdx int) string {
	if first == "" || strings.Contains(first, "NULL") || strings.HasSuffix(first, ")") {
		return first
	}
	parts := []string{first}
	for i := startIdx + 1; i < len(rec) && i <= startIdx+3; i++ {
		cell := strings.TrimSpace(rec[i])
		if cell == "" || !typeFragRe.MatchString(cell) {
			break
		}
		parts = append(parts, cell)
		if strings.Contains(cell, "NULL") || strings.HasSuffix(cell, ")") {
			break
		}
	}
	return strings.Join(parts, " ")
}

// SplitGlossary deterministically partitions rows into seeds and hold-out
// using a hash of the term identity, so the same -holdout-frac always yields
// the identical split regardless of row order or CSV re-export.
func SplitGlossary(rows []GlossaryRow, holdoutFrac float64) (seeds, heldOut []GlossaryRow) {
	for _, row := range rows {
		h := fnv.New32a()
		h.Write([]byte(row.ID + "|" + row.BusinessTerm))
		if float64(h.Sum32()%100) < holdoutFrac*100 {
			heldOut = append(heldOut, row)
		} else {
			seeds = append(seeds, row)
		}
	}
	return
}

// RegisterGlossaryRows adds rows to the registry. With hideAliases=true the
// terms keep name/definition/domain/type but drop their physical aliases —
// simulating a differently-named scanned column and forcing matches through
// lexical scoring + LLM adjudication instead of seed lookup.
func RegisterGlossaryRows(reg *TermRegistry, rows []GlossaryRow, hideAliases bool) {
	for _, row := range rows {
		aliases := row.Aliases
		if hideAliases {
			aliases = nil
		}
		reg.AddTerm(&SemanticTerm{
			ID:          "glossary:" + row.ID,
			Source:      "glossary",
			Name:        row.BusinessTerm,
			Aliases:     aliases,
			Description: row.Definition,
			Domain:      row.Domain,
			RawType:     row.DataType,
		}, row.SourceTables)
	}
}

// ColumnsFromRows materializes one ColumnMeta per (table, alias) pair.
func ColumnsFromRows(rows []GlossaryRow) []ColumnMeta {
	var cols []ColumnMeta
	for _, row := range rows {
		for _, tbl := range row.SourceTables {
			for _, a := range row.Aliases {
				cols = append(cols, ColumnMeta{Table: tbl, Column: a, DataType: row.DataType})
			}
		}
	}
	return cols
}

// ExpectedMap builds the ground truth for Evaluate: "table.column" -> term name.
func ExpectedMap(rows []GlossaryRow) map[string]string {
	m := map[string]string{}
	for _, row := range rows {
		for _, tbl := range row.SourceTables {
			for _, a := range row.Aliases {
				m[tbl+"."+a] = row.BusinessTerm
			}
		}
	}
	return m
}

// EvalSummary is the scored result of a hold-out run.
type EvalSummary struct {
	Total    int
	Correct  int
	Accuracy float64
	ByMethod map[string]int // correct matches by decision path
	ByStatus map[string]int // status distribution over eval columns
	Misses   []string
}

// RunHoldoutEval runs the pipeline over eval columns and scores the outcomes.
func RunHoldoutEval(ctx context.Context, p *Pipeline, cols []ColumnMeta, expected map[string]string) (*EvalSummary, error) {
	outcomes, err := p.Run(ctx, cols)
	if err != nil {
		return nil, err
	}
	s := &EvalSummary{ByMethod: map[string]int{}, ByStatus: map[string]int{}, Misses: []string{}}
	for _, o := range outcomes {
		want, ok := expected[o.Column.Table+"."+o.Column.Column]
		if !ok {
			continue
		}
		s.Total++
		s.ByStatus[o.Status]++
		got := ""
		if o.Term != nil {
			got = o.Term.Name
		}
		if strings.EqualFold(got, want) {
			s.Correct++
			s.ByMethod[o.Method]++
		} else {
			s.Misses = append(s.Misses, fmt.Sprintf(
				"%s.%s\twant=%q\tgot=%q\tconf=%.2f\tmethod=%s\tstatus=%s",
				o.Column.Table, o.Column.Column, want, got, o.Confidence, o.Method, o.Status))
		}
	}
	if s.Total > 0 {
		s.Accuracy = float64(s.Correct) / float64(s.Total)
	}
	sort.Strings(s.Misses)
	return s, nil
}
