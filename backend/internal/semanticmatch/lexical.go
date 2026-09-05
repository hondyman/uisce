package semanticmatch

import "math"

// Trigram dice coefficient over normalized strings — robust to typos and
// partial-word overlap, complements token metrics.
func trigramSim(a, b string) float64 {
	ta := trigramSet(NormalizeKey(a))
	tb := trigramSet(NormalizeKey(b))
	if len(ta) == 0 || len(tb) == 0 {
		return 0
	}
	inter := 0
	for g := range ta {
		if tb[g] {
			inter++
		}
	}
	return 2 * float64(inter) / float64(len(ta)+len(tb))
}

func trigramSet(s string) map[string]bool {
	out := map[string]bool{}
	if len(s) < 3 {
		if s != "" {
			out[s] = true
		}
		return out
	}
	for i := 0; i+3 <= len(s); i++ {
		out[s[i:i+3]] = true
	}
	return out
}

// LexicalScore computes a 0..1 similarity between a column and a term:
//
//	0.50  IDF-weighted token containment (subset direction — handles
//	      "acct_no" ⊂ "Trade Order Account Number", while common tokens
//	      like code/id/date contribute almost nothing)
//	0.15  IDF-weighted Jaccard
//	0.15  trigram similarity (column vs term name and mnemonic)
//	0.20  data-type family compatibility
//
// Exact alias/mnemonic identity short-circuits to 1.0.
func (r *TermRegistry) LexicalScore(col ColumnMeta, t *SemanticTerm) float64 {
	key := NormalizeKey(col.Column)
	if r.aliasExact(key, t) {
		return 1.0
	}
	colToks := tokenSet(ExpandTokens(Tokenize(col.Column)))
	termToks := r.termTokenSet(t)

	cont, jac := r.weightedMetrics(colToks, termToks)
	trig := trigramSim(col.Column, t.Name)
	if t.Mnemonic != "" {
		if m := trigramSim(col.Column, t.Mnemonic); m > trig {
			trig = m
		}
	}
	tc := TypeCompatibility(col.TypeFamily(), t.Family())
	return 0.50*cont + 0.15*jac + 0.15*trig + 0.20*tc
}

func (r *TermRegistry) weightedMetrics(a, b map[string]bool) (containment, jaccard float64) {
	var massA, massB, interMass float64
	for tok := range a {
		w := r.IDF(tok)
		massA += w
		if b[tok] {
			interMass += w
		}
	}
	for tok := range b {
		massB += r.IDF(tok)
	}
	if massA == 0 || massB == 0 {
		return 0, 0
	}
	minMass := math.Min(massA, massB)
	return interMass / minMass, interMass / (massA + massB - interMass)
}

func (r *TermRegistry) aliasExact(colKey string, t *SemanticTerm) bool {
	if colKey == "" {
		return false
	}
	if NormalizeKey(t.Mnemonic) == colKey || NormalizeKey(t.Name) == colKey {
		return true
	}
	for _, a := range t.Aliases {
		if NormalizeKey(a) == colKey {
			return true
		}
	}
	return false
}

func tokenSet(toks []string) map[string]bool {
	m := map[string]bool{}
	for _, t := range toks {
		if t != "" {
			m[t] = true
		}
	}
	return m
}
