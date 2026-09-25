package datapipeline

import (
	"sort"
	"strings"
)

// TargetField is a column/field on the target (a business object or a staging
// table) that the analyst can map to.
type TargetField struct {
	Name     string `json:"name"`
	Label    string `json:"label,omitempty"`
	Type     string `json:"type,omitempty"`
	Required bool   `json:"required,omitempty"`
}

// Suggestion is a proposed source->target match. Confidence is 0..1; the UI
// auto-applies >= 0.9 and shows the rest as "did you mean".
type Suggestion struct {
	From       string  `json:"from"`
	To         string  `json:"to"`
	Confidence float64 `json:"confidence"`
	Transform  string  `json:"transform,omitempty"`
	Reason     string  `json:"reason"`
}

// SuggestMapping proposes source->target matches so the analyst starts from a
// filled-in mapping rather than a blank one. It is deterministic and local
// (no LLM): normalised-name equality, then token overlap, then edit distance;
// a compatible-type bonus breaks ties; each target is used at most once.
func SuggestMapping(src []Column, tgt []TargetField) []Suggestion {
	type cand struct {
		s   Suggestion
		key float64
	}
	var cands []cand
	for _, c := range src {
		for _, t := range tgt {
			score, why := nameScore(c.Name, t.Name, t.Label)
			if score < 0.5 {
				continue
			}
			if typesCompatible(c.Type, t.Type) {
				score = min1(score + 0.05)
			} else if c.Type != "" && t.Type != "" {
				score -= 0.15
			}
			if score < 0.5 {
				continue
			}
			sg := Suggestion{From: c.Name, To: t.Name, Confidence: round2(score), Reason: why}
			sg.Transform = suggestTransform(c.Type, t.Type)
			cands = append(cands, cand{sg, score})
		}
	}
	sort.SliceStable(cands, func(i, j int) bool {
		if cands[i].key != cands[j].key {
			return cands[i].key > cands[j].key
		}
		return cands[i].s.From < cands[j].s.From
	})
	usedSrc, usedTgt := map[string]bool{}, map[string]bool{}
	var out []Suggestion
	for _, c := range cands {
		if usedSrc[c.s.From] || usedTgt[c.s.To] {
			continue
		}
		usedSrc[c.s.From], usedTgt[c.s.To] = true, true
		out = append(out, c.s)
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].From < out[j].From })
	return out
}

func nameScore(src, tgt, label string) (float64, string) {
	ns, nt := norm(src), norm(tgt)
	if ns == nt {
		return 1, "same name"
	}
	if label != "" && ns == norm(label) {
		return 0.97, "matches display name"
	}
	a, b := tokens(src), tokens(tgt)
	if len(a) > 0 && len(b) > 0 {
		inter := 0
		set := map[string]bool{}
		for _, t := range b {
			set[t] = true
		}
		for _, t := range a {
			if set[t] {
				inter++
			}
		}
		union := len(a) + len(b) - inter
		if j := float64(inter) / float64(union); j >= 0.5 {
			return 0.6 + 0.35*j, "similar words"
		}
	}
	if strings.Contains(ns, nt) || strings.Contains(nt, ns) {
		if len(ns) >= 4 && len(nt) >= 4 {
			return 0.7, "one name contains the other"
		}
	}
	if d := lev(ns, nt); len(ns) >= 4 && len(nt) >= 4 {
		m := max(len(ns), len(nt))
		if r := 1 - float64(d)/float64(m); r >= 0.8 {
			return 0.55 + 0.3*(r-0.8)/0.2, "spelled similarly"
		}
	}
	return 0, ""
}

func suggestTransform(src, tgt string) string {
	switch {
	case src == "string" && (tgt == "date" || tgt == "timestamp"):
		return "to_date"
	case src == "string" && (tgt == "int" || tgt == "float" || tgt == "decimal"):
		return "to_number"
	case src == "string" && tgt == "string":
		return "trim"
	}
	return ""
}

func typesCompatible(a, b string) bool {
	if a == "" || b == "" || a == b {
		return true
	}
	num := func(t string) bool { return t == "int" || t == "float" || t == "decimal" }
	if num(a) && num(b) {
		return true
	}
	// any source type can land in text; text can be parsed into others
	return a == "string" || b == "string"
}

func norm(s string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(s) {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func tokens(s string) []string {
	var out []string
	var cur strings.Builder
	flush := func() {
		if cur.Len() > 0 {
			out = append(out, strings.ToLower(cur.String()))
			cur.Reset()
		}
	}
	prevLower := false
	for _, r := range s {
		switch {
		case r >= 'A' && r <= 'Z':
			if prevLower {
				flush()
			}
			cur.WriteRune(r)
			prevLower = false
		case (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9'):
			cur.WriteRune(r)
			prevLower = true
		default:
			flush()
			prevLower = false
		}
	}
	flush()
	return out
}

func lev(a, b string) int {
	ra, rb := []rune(a), []rune(b)
	prev := make([]int, len(rb)+1)
	for j := range prev {
		prev[j] = j
	}
	for i := 1; i <= len(ra); i++ {
		cur := make([]int, len(rb)+1)
		cur[0] = i
		for j := 1; j <= len(rb); j++ {
			cost := 1
			if ra[i-1] == rb[j-1] {
				cost = 0
			}
			cur[j] = min(prev[j]+1, cur[j-1]+1, prev[j-1]+cost)
		}
		prev = cur
	}
	return prev[len(rb)]
}

func min1(f float64) float64 {
	if f > 1 {
		return 1
	}
	return f
}

func round2(f float64) float64 { return float64(int(f*100+0.5)) / 100 }
