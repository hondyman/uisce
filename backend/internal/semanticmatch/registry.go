package semanticmatch

import (
	"fmt"
	"math"
	"strings"
)

// TermRegistry holds the canonical vocabulary plus lookup structures.
type TermRegistry struct {
	Terms []*SemanticTerm
	byID  map[string]*SemanticTerm

	byAlias       map[string][]string // NormalizeKey(alias) -> term IDs
	glossarySeeds map[string]string   // lower(table) + \x00 + colkey -> term ID (curated, table-scoped)
	sameNameSeeds map[string]string   // colkey -> term ID (glossary "same-name rule")

	idf map[string]float64
}

func NewTermRegistry() *TermRegistry {
	return &TermRegistry{
		byID:          map[string]*SemanticTerm{},
		byAlias:       map[string][]string{},
		glossarySeeds: map[string]string{},
		sameNameSeeds: map[string]string{},
		idf:           map[string]float64{},
	}
}

// AddTerm registers a term. tables is the list of source tables for curated
// glossary entries (empty for Bloomberg fields).
func (r *TermRegistry) AddTerm(t *SemanticTerm, tables []string) {
	if t.ID == "" {
		t.ID = fmt.Sprintf("%s:%s", t.Source, NormalizeKey(t.Mnemonic+t.Name))
	}
	if _, exists := r.byID[t.ID]; exists {
		return
	}
	r.Terms = append(r.Terms, t)
	r.byID[t.ID] = t

	for _, a := range append([]string{t.Mnemonic, t.Name}, t.Aliases...) {
		if k := NormalizeKey(a); k != "" {
			r.byAlias[k] = appendUnique(r.byAlias[k], t.ID)
		}
	}
	if t.Source == "glossary" {
		// Curated seeds: exact (table, column) provenance.
		for _, tbl := range tables {
			for _, a := range t.Aliases {
				r.glossarySeeds[strings.ToLower(tbl)+"\x00"+NormalizeKey(a)] = t.ID
			}
		}
		// Same-name rule: glossary ruling says the term carries to any table
		// where the column name appears (e.g. create_date, comments, city).
		for _, a := range t.Aliases {
			if k := NormalizeKey(a); k != "" {
				if _, exists := r.sameNameSeeds[k]; !exists {
					r.sameNameSeeds[k] = t.ID
				}
			}
		}
	}
}

// SeedLookup returns the curated glossary match for a column, if any.
func (r *TermRegistry) SeedLookup(col ColumnMeta) (termID string, exactTable bool) {
	key := NormalizeKey(col.Column)
	if key == "" {
		return "", false
	}
	for _, tbl := range []string{col.Table, col.Schema + "." + col.Table} {
		if id, ok := r.glossarySeeds[strings.ToLower(tbl)+"\x00"+key]; ok {
			return id, true
		}
	}
	if id, ok := r.sameNameSeeds[key]; ok {
		return id, false
	}
	return "", false
}

// BuildIDF computes token document frequencies across the whole vocabulary.
// It also warms per-term token caches so parallel scoring is race-free.
// MUST be called after all AddTerm calls and before NewPipeline.
func (r *TermRegistry) BuildIDF() {
	n := float64(len(r.Terms))
	if n == 0 {
		return
	}
	df := map[string]int{}
	for _, t := range r.Terms {
		for tok := range r.termTokenSet(t) {
			df[tok]++
		}
	}
	for tok, d := range df {
		r.idf[tok] = math.Log(1 + n/float64(1+d))
	}
}

// IDF returns the informativeness of a token. Tokens unseen in the corpus
// (rare, distinctive) get maximum weight.
func (r *TermRegistry) IDF(tok string) float64 {
	if w, ok := r.idf[tok]; ok {
		return w
	}
	return math.Log(1 + float64(len(r.Terms)))
}

func (r *TermRegistry) termTokenSet(t *SemanticTerm) map[string]bool {
	if t.tokCache != nil {
		return t.tokCache
	}
	set := map[string]bool{}
	for _, s := range append([]string{t.Name, t.Mnemonic}, t.Aliases...) {
		if s == "" {
			continue
		}
		for _, tok := range ExpandTokens(Tokenize(s)) {
			set[tok] = true
		}
	}
	t.tokCache = set
	return set
}

func appendUnique(ids []string, id string) []string {
	for _, x := range ids {
		if x == id {
			return ids
		}
	}
	return append(ids, id)
}

// SuggestTermName builds a human-readable term name from a column name,
// e.g. "cash_rsrv_pct" -> "Cash Reserve Percentage".
func SuggestTermName(col string) string {
	var b strings.Builder
	first := true
	for _, t := range ExpandTokens(Tokenize(col)) {
		if t == "" || isAllDigits(t) {
			continue
		}
		if !first {
			b.WriteByte(' ')
		}
		first = false
		b.WriteString(strings.ToUpper(t[:1]) + t[1:])
	}
	return b.String()
}

func isAllDigits(s string) bool {
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return len(s) > 0
}
