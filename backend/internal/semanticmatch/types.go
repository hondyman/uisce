package semanticmatch

import (
	"strings"
	"time"
)

// ColumnMeta describes a physical column discovered by a metadata scan.
type ColumnMeta struct {
	TenantID     string   `json:"tenant_id"`
	Database     string   `json:"database"`
	Schema       string   `json:"schema"`
	Table        string   `json:"table"`
	Column       string   `json:"column"`
	DataType     string   `json:"data_type"` // raw DDL fragment, e.g. "NUMERIC(18,4) NULL"
	Nullable     bool     `json:"nullable"`
	Description  string   `json:"description,omitempty"` // source DB comment, if any
	SampleValues []string `json:"sample_values,omitempty"`
}

func (c ColumnMeta) QualName() string {
	var scope []string
	for _, p := range []string{c.Database, c.Schema, c.Table} {
		if p != "" {
			scope = append(scope, p)
		}
	}
	if len(scope) == 0 {
		return c.Column
	}
	return strings.Join(scope, ".") + "." + c.Column
}

// SemanticTerm is one canonical vocabulary entry (glossary term or Bloomberg field).
type SemanticTerm struct {
	ID          string   `json:"id"`                 // e.g. "bb:PX_LAST", "glossary:57"
	Source      string   `json:"source"`             // "glossary" | "bloomberg"
	Mnemonic    string   `json:"mnemonic,omitempty"`
	Name        string   `json:"name"`               // human business term, e.g. "Cash Account Number"
	Aliases     []string `json:"aliases,omitempty"`  // physical column names mapped to this term
	Description string   `json:"description,omitempty"`
	Domain      string   `json:"domain,omitempty"`   // glossary domain or BB category
	RawType     string   `json:"raw_type,omitempty"`

	tokCache map[string]bool // warmed by BuildIDF / BuildLexIndex
}

// Candidate is a retrieved term with component scores for one column.
type Candidate struct {
	Term    *SemanticTerm
	Lexical float64
	Embed   float64 // normalized cosine, 0 when embeddings disabled
}

// Decision statuses and provenance methods.
const (
	StatusAutoLinked = "auto_linked"
	StatusReview     = "pending_review"
	StatusNoMatch    = "no_match"

	MethodExact   = "exact_alias"
	MethodSeed    = "glossary_seed"
	MethodLexical = "lexical"
	MethodLLM     = "llm_gemini"
	MethodNone    = "none"
)

// MatchOutcome is the final decision for one column.
type MatchOutcome struct {
	Column           ColumnMeta    `json:"column"`
	Term             *SemanticTerm `json:"term,omitempty"`
	Confidence       float64       `json:"confidence"`
	Method           string        `json:"method"`
	Status           string        `json:"status"`
	Reasoning        string        `json:"reasoning,omitempty"`
	SuggestedNewTerm string        `json:"suggested_new_term,omitempty"`
	DecidedAt        time.Time     `json:"decided_at"`
}
