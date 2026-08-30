package semanticmatch

import (
	"math"
	"sort"
)

// LexIndex is a token inverted index over the registry, used to pull a
// manageable candidate set before any LLM call.
type LexIndex struct {
	postings map[string][]int
	terms    []*SemanticTerm
	reg      *TermRegistry
}

func BuildLexIndex(reg *TermRegistry) *LexIndex {
	ix := &LexIndex{postings: map[string][]int{}, terms: reg.Terms, reg: reg}
	for i, t := range reg.Terms {
		for tok := range reg.termTokenSet(t) {
			ix.postings[tok] = append(ix.postings[tok], i)
		}
	}
	return ix
}

// Search returns up to k candidates ranked by lexical score. Exact-alias
// hits are injected regardless of index ranking.
func (ix *LexIndex) Search(col ColumnMeta, k int) []Candidate {
	colToks := tokenSet(ExpandTokens(Tokenize(col.Column)))
	acc := map[int]float64{}
	for tok := range colToks {
		w := ix.reg.IDF(tok)
		for _, i := range ix.postings[tok] {
			acc[i] += w
		}
	}
	type hit struct {
		idx int
		s   float64
	}
	hits := make([]hit, 0, len(acc))
	for i, s := range acc {
		hits = append(hits, hit{i, s})
	}
	sort.Slice(hits, func(a, b int) bool { return hits[a].s > hits[b].s })
	if len(hits) > 4*k { // oversample, then rescore precisely
		hits = hits[:4*k]
	}

	seen := map[string]bool{}
	out := make([]Candidate, 0, k)
	for _, h := range hits {
		t := ix.terms[h.idx]
		if seen[t.ID] {
			continue
		}
		seen[t.ID] = true
		out = append(out, Candidate{Term: t, Lexical: ix.reg.LexicalScore(col, t)})
	}
	for _, id := range ix.reg.byAlias[NormalizeKey(col.Column)] {
		if !seen[id] {
			if t := ix.reg.byID[id]; t != nil {
				seen[id] = true
				out = append(out, Candidate{Term: t, Lexical: 1.0})
			}
		}
	}
	sort.Slice(out, func(a, b int) bool { return out[a].Lexical > out[b].Lexical })
	if len(out) > k {
		out = out[:k]
	}
	return out
}

// ---- Vector index (brute-force cosine; fine up to ~100k vectors) ----

type VectorIndex struct {
	IDs  []string
	Vecs [][]float32
}

type VecHit struct {
	ID    string
	Score float64
}

func NewVectorIndex(ids []string, vecs [][]float32) *VectorIndex {
	return &VectorIndex{IDs: ids, Vecs: vecs}
}

func (v *VectorIndex) TopK(q []float32, k int) []VecHit {
	hits := make([]VecHit, 0, len(v.IDs))
	for i, id := range v.IDs {
		if s := cosineSim(q, v.Vecs[i]); s > 0.25 { // prune obvious noise
			hits = append(hits, VecHit{ID: id, Score: s})
		}
	}
	sort.Slice(hits, func(a, b int) bool { return hits[a].Score > hits[b].Score })
	if len(hits) > k {
		hits = hits[:k]
	}
	return hits
}

func cosineSim(a, b []float32) float64 {
	n := len(a)
	if len(b) < n {
		n = len(b)
	}
	var dot, na, nb float64
	for i := 0; i < n; i++ {
		dot += float64(a[i]) * float64(b[i])
		na += float64(a[i]) * float64(a[i])
		nb += float64(b[i]) * float64(b[i])
	}
	if na == 0 || nb == 0 {
		return 0
	}
	return dot / math.Sqrt(na*nb)
}
