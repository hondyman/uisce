package semanticmatch

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

const verifySystemPrompt = `You are a senior data-governance analyst matching physical database columns from investment-management systems to canonical business terms (an internal glossary plus Bloomberg Data License fields).

For every column you receive, pick the single candidate that represents the same business concept, or NONE.

Principles:
1. Match meaning, not wording. "deliv_loc_num" = "Delivery Location Number". A shared generic word (id, code, date, amount, name, type) is not evidence by itself.
2. Use the table for context. ts_order_broker.amount is the broker's amount for the order, not the order total.
3. Types must be plausible. A TIMESTAMP column cannot be "Last Price"; a CHAR(1) Y/N flag can match a Boolean term; a 3-character VARCHAR suits an ISO currency code.
4. Definitions are authoritative when available; prefer the term whose definition explains the column's purpose in its table.
5. Glossary terms are curated internally; when a glossary term and a Bloomberg field fit equally well, prefer the glossary term.
6. Calibrate confidence: 0.95-1.00 definitive; 0.85-0.94 strong; 0.65-0.84 plausible but uncertain; below 0.65 return NONE.
7. Only use term IDs from the candidate list. Never invent IDs.

Respond with JSON only: one match object per column, including NONE rows (term_id = "NONE").`

var verifyResponseSchema = map[string]any{
	"type": "OBJECT",
	"properties": map[string]any{
		"matches": map[string]any{
			"type": "ARRAY",
			"items": map[string]any{
				"type": "OBJECT",
				"properties": map[string]any{
					"column_id":  map[string]any{"type": "STRING"},
					"term_id":    map[string]any{"type": "STRING"},
					"confidence": map[string]any{"type": "NUMBER"},
					"reasoning":  map[string]any{"type": "STRING"},
				},
				"required": []string{"column_id", "term_id", "confidence", "reasoning"},
			},
		},
	},
	"required": []string{"matches"},
}

type llmMatch struct {
	ColumnID   string  `json:"column_id"`
	TermID     string  `json:"term_id"`
	Confidence float64 `json:"confidence"`
	Reasoning  string  `json:"reasoning"`
}

type llmResponse struct {
	Matches []llmMatch `json:"matches"`
}

type LLMVerifier struct {
	Client *GeminiClient
}

type VerifyItem struct {
	Col   ColumnMeta
	Cands []Candidate
}

// Verify sends one batch of columns (each with its candidate list) to Gemini
// and returns per-column decisions keyed by positional ID ("c0", "c1", ...).
func (v *LLMVerifier) Verify(ctx context.Context, items []VerifyItem) (map[string]llmMatch, error) {
	if len(items) == 0 {
		return map[string]llmMatch{}, nil
	}
	raw, err := v.Client.GenerateJSON(ctx, verifySystemPrompt, buildVerifyPrompt(items), verifyResponseSchema)
	if err != nil {
		return nil, err
	}
	var resp llmResponse
	if err := json.Unmarshal([]byte(raw), &resp); err != nil {
		return nil, fmt.Errorf("parse llm response: %w", err)
	}
	out := map[string]llmMatch{}
	for _, m := range resp.Matches {
		out[m.ColumnID] = m
	}
	for i := range items {
		id := fmt.Sprintf("c%d", i)
		if _, ok := out[id]; !ok {
			out[id] = llmMatch{ColumnID: id, TermID: "NONE", Confidence: 0, Reasoning: "missing from LLM response"}
		}
	}
	return out, nil
}

var textCleaner = strings.NewReplacer(
	"\n", " ", "\r", " ", "\t", " ",
	"<br>", " ", "<BR>", " ", "<br/>", " ", "</br>", " ", "<BR/>", " ",
)

func buildVerifyPrompt(items []VerifyItem) string {
	var b strings.Builder
	b.WriteString("Match each COLUMN to at most one candidate term. ")
	b.WriteString("Return one JSON match object per column (term_id \"NONE\" when no candidate is right).\n\n")
	for i, it := range items {
		fmt.Fprintf(&b, "### COLUMN c%d\n", i)
		fmt.Fprintf(&b, "table: %s\ncolumn: %s\n", it.Col.Table, it.Col.Column)
		fmt.Fprintf(&b, "column (expanded): %s\n", strings.Join(ExpandTokens(Tokenize(it.Col.Column)), " "))
		fmt.Fprintf(&b, "type: %s\n", strings.TrimSpace(it.Col.DataType))
		if it.Col.Description != "" {
			fmt.Fprintf(&b, "column description: %s\n", truncate(textCleaner.Replace(it.Col.Description), 300))
		}
		if len(it.Col.SampleValues) > 0 {
			fmt.Fprintf(&b, "sample values: %s\n", truncate(strings.Join(it.Col.SampleValues, ", "), 200))
		}
		b.WriteString("candidates:\n")
		for _, c := range it.Cands {
			t := c.Term
			fmt.Fprintf(&b, "  - %s | %s | %q | type=%s | domain=%s | %s\n",
				t.ID, t.Mnemonic, t.Name, t.RawType, t.Domain,
				truncate(textCleaner.Replace(t.Description), 220))
		}
		b.WriteString("\n")
	}
	return b.String()
}

func truncate(s string, n int) string {
	s = strings.TrimSpace(s)
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}

// TermEmbeddingText renders a term for embedding.
func TermEmbeddingText(t *SemanticTerm) string {
	var b strings.Builder
	if t.Mnemonic != "" {
		b.WriteString(t.Mnemonic + " — ")
	}
	b.WriteString(t.Name)
	if t.Domain != "" {
		b.WriteString(" (" + t.Domain + ")")
	}
	if t.Description != "" {
		b.WriteString(". " + t.Description)
	}
	return truncate(textCleaner.Replace(b.String()), 1024)
}

// ColumnEmbeddingText renders a column for embedding.
func ColumnEmbeddingText(c ColumnMeta) string {
	var b strings.Builder
	b.WriteString(c.Table + "." + c.Column)
	b.WriteString(" (" + strings.Join(ExpandTokens(Tokenize(c.Column)), " ") + ")")
	if c.Description != "" {
		b.WriteString(". " + c.Description)
	}
	return truncate(textCleaner.Replace(b.String()), 512)
}
