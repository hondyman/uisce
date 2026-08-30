package semanticmatch

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"sort"
	"strings"
	"sync"
	"time"
)

type Config struct {
	AutoLinkThreshold    float64 // fused score >= this -> auto_linked (default 0.90)
	ReviewThreshold      float64 // fused score >= this -> pending_review (default 0.65)
	LexicalAutoThreshold float64 // lexical-only auto accept, no LLM call (default 0.90)
	LexicalMargin        float64 // required gap over 2nd candidate (default 0.10)
	TopK                 int     // candidates per column (default 12)
	LLMBatchSize         int     // columns per Gemini request (default 8)
	EmbedModel           string  // "" disables embeddings, e.g. "text-embedding-004"
	Workers              int     // parallelism for local scoring
}

func DefaultConfig() Config {
	return Config{
		AutoLinkThreshold:    0.90,
		ReviewThreshold:      0.65,
		LexicalAutoThreshold: 0.90,
		LexicalMargin:        0.10,
		TopK:                 12,
		LLMBatchSize:         8,
		EmbedModel:           "",
		Workers:              8,
	}
}

type Pipeline struct {
	Cfg      Config
	Registry *TermRegistry
	Lex      *LexIndex
	Vectors  *VectorIndex // optional; improves recall for definition-level matches
	LLM      *LLMVerifier // optional; when nil the pipeline runs lexical-only
	Log      *slog.Logger
}

func NewPipeline(cfg Config, reg *TermRegistry, llm *LLMVerifier) *Pipeline {
	reg.BuildIDF() // idempotent: warms per-term token caches so parallel scoring is race-free
	if cfg.Workers <= 0 {
		cfg.Workers = 8
	}
	if cfg.TopK <= 0 {
		cfg.TopK = 12
	}
	if cfg.LLMBatchSize <= 0 {
		cfg.LLMBatchSize = 8
	}
	return &Pipeline{Cfg: cfg, Registry: reg, Lex: BuildLexIndex(reg), LLM: llm, Log: slog.Default()}
}

func (p *Pipeline) Run(ctx context.Context, cols []ColumnMeta) ([]MatchOutcome, error) {
	outcomes := make([]MatchOutcome, len(cols))
	pending := []int{}

	// Phase 0 — curated table-scoped glossary seeds (deterministic, conf 1.0).
	for i, col := range cols {
		if id, exact := p.Registry.SeedLookup(col); id != "" && exact {
			t := p.Registry.byID[id]
			outcomes[i] = p.decide(col, t, 1.0, MethodSeed, "curated glossary mapping (table-scoped)")
			continue
		}
		pending = append(pending, i)
	}
	p.Log.Info("phase 0 done", "seeded", len(cols)-len(pending), "pending", len(pending))

	// Phase 1 — candidate retrieval + lexical scoring (parallel).
	type work struct {
		idx   int
		cands []Candidate
	}
	works := make([]work, len(pending))
	var wg sync.WaitGroup
	sem := make(chan struct{}, p.Cfg.Workers)
	for j, i := range pending {
		wg.Add(1)
		go func(j, i int) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			works[j] = work{idx: i, cands: p.Lex.Search(cols[i], p.Cfg.TopK)}
		}(j, i)
	}
	wg.Wait()

	// Phase 1b — merge embedding candidates (one batched embed call for all columns).
	if p.Vectors != nil && p.LLM != nil && p.Cfg.EmbedModel != "" {
		texts := make([]string, len(works))
		for j, w := range works {
			texts[j] = ColumnEmbeddingText(cols[w.idx])
		}
		vecs, err := p.LLM.Client.EmbedBatch(ctx, p.Cfg.EmbedModel, "RETRIEVAL_QUERY", texts)
		if err != nil {
			p.Log.Warn("embedding failed; continuing with lexical retrieval only", "err", err)
		} else {
			for j := range works {
				byID := map[string]float64{}
				for _, h := range p.Vectors.TopK(vecs[j], 20) {
					byID[h.ID] = h.Score
				}
				for _, c := range works[j].cands {
					delete(byID, c.Term.ID)
				}
				merged := works[j].cands
				for id, s := range byID {
					t := p.Registry.byID[id]
					if t == nil {
						continue
					}
					merged = append(merged, Candidate{
						Term:    t,
						Lexical: p.Registry.LexicalScore(cols[works[j].idx], t),
						Embed:   normalizeCos(s),
					})
				}
				sort.Slice(merged, func(a, b int) bool {
					return merged[a].Lexical+0.25*merged[a].Embed > merged[b].Lexical+0.25*merged[b].Embed
				})
				if len(merged) > p.Cfg.TopK {
					merged = merged[:p.Cfg.TopK]
				}
				works[j].cands = merged
			}
		}
	}

	// Phase 2 — lexical auto-accept, cheap rejects; everything else goes to the LLM.
	needLLM := []work{}
	for _, w := range works {
		col := cols[w.idx]
		if len(w.cands) == 0 {
			outcomes[w.idx] = p.noMatch(col, "no candidates retrieved")
			continue
		}
		top := w.cands[0]
		margin := 0.0
		if len(w.cands) > 1 {
			margin = top.Lexical - w.cands[1].Lexical
		}
		if p.LLM == nil {
			outcomes[w.idx] = p.decide(col, top.Term, math.Min(top.Lexical, 0.97),
				MethodLexical, fmt.Sprintf("lexical-only mode: score %.2f", top.Lexical))
			continue
		}
		if top.Lexical >= p.Cfg.LexicalAutoThreshold && margin >= p.Cfg.LexicalMargin {
			outcomes[w.idx] = p.decide(col, top.Term, math.Min(top.Lexical, 0.97), MethodLexical,
				fmt.Sprintf("lexical %.2f, margin %.2f (LLM bypassed)", top.Lexical, margin))
			continue
		}
		if top.Lexical < 0.15 && top.Embed < 0.2 {
			outcomes[w.idx] = p.noMatch(col, "no plausible candidates")
			continue
		}
		needLLM = append(needLLM, w)
	}
	p.Log.Info("phase 2 done", "lexical_resolved", len(works)-len(needLLM), "to_llm", len(needLLM))

	// Phase 3 — Gemini adjudication in batches.
	for start := 0; start < len(needLLM); start += p.Cfg.LLMBatchSize {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		end := min(start+p.Cfg.LLMBatchSize, len(needLLM))
		batch := needLLM[start:end]

		items := make([]VerifyItem, len(batch))
		for j, w := range batch {
			items[j] = VerifyItem{Col: cols[w.idx], Cands: w.cands}
		}
		res, err := p.LLM.Verify(ctx, items)
		if err != nil {
			p.Log.Error("llm verify failed; falling back to lexical decision", "err", err)
			for _, w := range batch {
				outcomes[w.idx] = p.decide(cols[w.idx], w.cands[0].Term,
					math.Min(w.cands[0].Lexical, 0.85), MethodLexical, "llm unavailable; lexical fallback")
			}
			continue
		}
		for j, w := range batch {
			col := cols[w.idx]
			m := res[fmt.Sprintf("c%d", j)]

			candByID := map[string]Candidate{}
			for _, c := range w.cands {
				candByID[c.Term.ID] = c
			}
			if c, ok := candByID[m.TermID]; ok && m.TermID != "NONE" {
				outcomes[w.idx] = p.decide(col, c.Term, fuse(c, m.Confidence), MethodLLM, m.Reasoning)
				continue
			}
			// LLM said NONE (or hallucinated an ID): keep strong lexical for review.
			top := w.cands[0]
			if top.Lexical >= 0.85 {
				outcomes[w.idx] = p.decide(col, top.Term, 0.70, MethodLexical,
					"LLM rejected; strong lexical kept for review: "+m.Reasoning)
			} else {
				outcomes[w.idx] = p.noMatch(col, "no match (LLM): "+m.Reasoning)
			}
		}
	}

	// Defensive: every column gets an outcome.
	for i := range outcomes {
		if outcomes[i].Status == "" {
			outcomes[i] = p.noMatch(cols[i], "unresolved")
		}
	}

	var auto, review, none int
	for _, o := range outcomes {
		switch o.Status {
		case StatusAutoLinked:
			auto++
		case StatusReview:
			review++
		default:
			none++
		}
	}
	p.Log.Info("matching complete",
		"columns", len(cols), "auto_linked", auto, "pending_review", review, "no_match", none)
	return outcomes, nil
}

type RankedCandidate struct {
	Term       *SemanticTerm `json:"term"`
	Lexical    float64       `json:"lexical"`
	Embedding  float64       `json:"embedding"`
	Confidence float64       `json:"confidence"`
	Reasoning  string        `json:"reasoning,omitempty"`
}

// Suggest matches a single column on demand and returns ranked candidates.
// It never writes edges — the wizard lets a human confirm.
func (p *Pipeline) Suggest(ctx context.Context, col ColumnMeta) ([]RankedCandidate, MatchOutcome, error) {
	if id, exact := p.Registry.SeedLookup(col); id != "" && exact {
		t := p.Registry.byID[id]
		oc := p.decide(col, t, 1.0, MethodSeed, "curated glossary mapping (table-scoped)")
		return []RankedCandidate{{Term: t, Confidence: 1.0, Reasoning: oc.Reasoning}}, oc, nil
	}

	cands := p.Lex.Search(col, p.Cfg.TopK)
	if len(cands) == 0 {
		return nil, p.noMatch(col, "no candidates retrieved"), nil
	}

	var outcome MatchOutcome
	if p.LLM == nil {
		outcome = p.decide(col, cands[0].Term, min(cands[0].Lexical, 0.97), MethodLexical, "lexical-only mode")
	} else if res, err := p.LLM.Verify(ctx, []VerifyItem{{Col: col, Cands: cands}}); err != nil {
		outcome = p.decide(col, cands[0].Term, min(cands[0].Lexical, 0.85), MethodLexical, "llm unavailable; lexical fallback")
	} else if m := res["c0"]; m.TermID != "NONE" && m.TermID != "" {
		outcome = p.noMatch(col, "no match (LLM): "+m.Reasoning)
		for _, c := range cands {
			if c.Term.ID == m.TermID {
				outcome = p.decide(col, c.Term, fuse(c, m.Confidence), MethodLLM, m.Reasoning)
				break
			}
		}
	} else {
		outcome = p.noMatch(col, "no match (LLM): "+res["c0"].Reasoning)
	}

	ranked := make([]RankedCandidate, len(cands))
	for i, c := range cands {
		rc := RankedCandidate{Term: c.Term, Confidence: c.Lexical, Lexical: c.Lexical, Embedding: c.Embed}
		if outcome.Term != nil && outcome.Term.ID == c.Term.ID {
			rc.Confidence, rc.Reasoning = outcome.Confidence, outcome.Reasoning
		}
		ranked[i] = rc
	}
	return ranked, outcome, nil
}

// fuse combines lexical + LLM confidence with an anti-hallucination guard:
// if the LLM picks a term that lexical retrieval barely surfaced, cap the
// result below the auto-link threshold so a human sees it.
func fuse(c Candidate, llmConf float64) float64 {
	combined := 0.30*c.Lexical + 0.70*llmConf
	if c.Lexical < 0.05 && c.Embed < 0.35 {
		combined = math.Min(combined, 0.65)
	}
	return math.Min(math.Max(combined, 0), 0.97)
}

// normalizeCos maps text-embedding cosine similarity (~0.2 noise, ~0.55+ strong)
// to a 0..1 scale.
func normalizeCos(cos float64) float64 {
	return math.Min(math.Max((cos-0.30)/0.40, 0), 1)
}

func (p *Pipeline) decide(col ColumnMeta, t *SemanticTerm, conf float64, method, reasoning string) MatchOutcome {
	status := StatusReview
	switch {
	case conf >= p.Cfg.AutoLinkThreshold:
		status = StatusAutoLinked
	case conf < p.Cfg.ReviewThreshold:
		status = StatusNoMatch
	}
	o := MatchOutcome{
		Column: col, Term: t,
		Confidence: math.Round(conf*1000) / 1000,
		Method:     method, Status: status,
		Reasoning: reasoning, DecidedAt: time.Now(),
	}
	if status == StatusNoMatch {
		o.SuggestedNewTerm = SuggestTermName(col.Column)
	}
	return o
}

func (p *Pipeline) noMatch(col ColumnMeta, reason string) MatchOutcome {
	return MatchOutcome{
		Column: col, Confidence: 0, Method: MethodNone, Status: StatusNoMatch,
		Reasoning: reason, SuggestedNewTerm: SuggestTermName(col.Column), DecidedAt: time.Now(),
	}
}

// Evaluate scores outcomes against curated expectations keyed by "table.column"
// -> expected term Name. Use Semantic_Definitions.csv rows as a holdout set to
// tune thresholds and prompt rules.
func Evaluate(outcomes []MatchOutcome, expected map[string]string) (correct, total int, misses []string) {
	for _, o := range outcomes {
		want, ok := expected[o.Column.Table+"."+o.Column.Column]
		if !ok {
			continue
		}
		total++
		got := ""
		if o.Term != nil {
			got = o.Term.Name
		}
		if strings.EqualFold(got, want) {
			correct++
		} else {
			misses = append(misses, fmt.Sprintf("%s.%s: want %q got %q (%.2f %s)",
				o.Column.Table, o.Column.Column, want, got, o.Confidence, o.Method))
		}
	}
	return
}
