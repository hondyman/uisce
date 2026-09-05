package analytics

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/hondyman/uisce/backend/internal/logging"
)

// duplicateBusinessTermThreshold is the fuzzy-name-similarity score (0..1) above
// which a candidate business term is treated as a duplicate of an existing one
// rather than proposed as new.
const duplicateBusinessTermThreshold = 0.82

// GeneratedBusinessTermInput describes the source column/semantic term a new
// business term is being proposed for.
type GeneratedBusinessTermInput struct {
	SemanticTermID   string
	SemanticTermName string
	Definition       string
	TableName        string
	ColumnName       string
	DataType         string
	SampleValues     []string
}

// GeneratedBusinessTermResult is the outcome of GenerateBusinessTermForColumn:
// either a brand-new business term was created, or an existing one was reused.
type GeneratedBusinessTermResult struct {
	BusinessTermID   string  `json:"business_term_id"`
	TermName         string  `json:"term_name"`
	Definition       string  `json:"definition"`
	Category         string  `json:"category,omitempty"`
	Duplicate        bool    `json:"duplicate"`
	DuplicateOfID    string  `json:"duplicate_of_id,omitempty"`
	DuplicateScore   float64 `json:"duplicate_score,omitempty"`
	LinkedToSemantic bool    `json:"linked_to_semantic_term"`
}

// geminiBusinessTermDraft is the strict JSON shape requested from the LLM.
type geminiBusinessTermDraft struct {
	Name       string `json:"name"`
	Definition string `json:"definition"`
	Category   string `json:"category"`
}

// GenerateBusinessTermForColumn proposes (or reuses) a business term for a
// scanned column/semantic term, using Gemini to draft a candidate name and
// definition, then deduplicating against existing business terms before
// ever writing a new catalog_node. This is the "don't introduce duplicates"
// gate: Gemini only gets to mint a new term when no existing term is close
// enough in name.
func (s *SemanticMappingService) GenerateBusinessTermForColumn(ctx context.Context, tenantID, tenantDatasourceID string, input GeneratedBusinessTermInput) (*GeneratedBusinessTermResult, error) {
	if strings.TrimSpace(input.SemanticTermName) == "" {
		return nil, fmt.Errorf("semantic term name is required")
	}

	existing, err := s.fetchBusinessTerms(ctx, tenantID, tenantDatasourceID)
	if err != nil {
		return nil, fmt.Errorf("failed to load existing business terms for dedup: %w", err)
	}

	draft, err := s.draftBusinessTerm(ctx, input, existing)
	if err != nil {
		return nil, err
	}

	result := &GeneratedBusinessTermResult{
		TermName:   draft.Name,
		Definition: draft.Definition,
		Category:   draft.Category,
	}

	if match, score := findClosestBusinessTerm(draft.Name, existing); match != nil && score >= duplicateBusinessTermThreshold {
		result.BusinessTermID = match.NodeID
		result.TermName = match.TermName
		result.Duplicate = true
		result.DuplicateOfID = match.NodeID
		result.DuplicateScore = score
		logging.GetLogger().Sugar().Infof("Reusing existing business term %q (score=%.2f) for semantic term %q instead of creating duplicate", match.TermName, score, input.SemanticTermName)
	} else {
		termName := toTitleCase(draft.Name)
		props := map[string]interface{}{
			"description": draft.Definition,
			"category":    draft.Category,
			"source":      "gemini_generated",
			"generated_from": map[string]interface{}{
				"semantic_term_id":   input.SemanticTermID,
				"semantic_term_name": input.SemanticTermName,
				"table":              input.TableName,
				"column":             input.ColumnName,
			},
		}
		newID, err := s.CreateBusinessTerm(ctx, tenantID, tenantDatasourceID, termName, props)
		if err != nil {
			return nil, fmt.Errorf("failed to create generated business term: %w", err)
		}
		result.BusinessTermID = newID
		result.TermName = termName
	}

	if input.SemanticTermID != "" {
		if err := s.LinkBusinessTermToSemanticTerm(ctx, tenantID, result.BusinessTermID, input.SemanticTermID); err != nil {
			logging.GetLogger().Sugar().Warnf("business term %s generated but failed to link to semantic term %s: %v", result.BusinessTermID, input.SemanticTermID, err)
		} else {
			result.LinkedToSemantic = true
		}
	}

	return result, nil
}

func (s *SemanticMappingService) draftBusinessTerm(ctx context.Context, input GeneratedBusinessTermInput, existing []SemanticTerm) (*geminiBusinessTermDraft, error) {
	llmProvider, ok := s.llmProvider.(interface {
		GenerateContent(context.Context, string) (string, error)
	})
	if !ok || s.llmProvider == nil {
		// No LLM configured: fall back to a deterministic draft derived from the
		// semantic term itself so the pipeline still functions without Gemini.
		return &geminiBusinessTermDraft{
			Name:       input.SemanticTermName,
			Definition: input.Definition,
			Category:   "",
		}, nil
	}

	existingNames := make([]string, 0, len(existing))
	for _, t := range existing {
		existingNames = append(existingNames, t.TermName)
	}

	prompt := fmt.Sprintf(`You are a financial data steward proposing a business glossary term for a scanned database column.

Semantic term: %s
Semantic term definition: %s
Source table.column: %s.%s
Data type: %s
Sample values: %s

Existing business terms already in the glossary (DO NOT propose a name that duplicates or is a trivial rewording of one of these; if one of these already covers this concept, return that exact existing name instead of inventing a new one):
%s

Return ONLY a JSON object matching this schema, no commentary, no markdown fences:
{"name": "...", "definition": "...", "category": "..."}`,
		input.SemanticTermName,
		input.Definition,
		input.TableName, input.ColumnName,
		input.DataType,
		strings.Join(input.SampleValues, ", "),
		strings.Join(existingNames, ", "),
	)

	raw, err := llmProvider.GenerateContent(ctx, prompt)
	if err != nil {
		return nil, fmt.Errorf("gemini business term generation failed: %w", err)
	}

	cleaned := strings.TrimSpace(raw)
	cleaned = strings.TrimPrefix(cleaned, "```json")
	cleaned = strings.TrimPrefix(cleaned, "```")
	cleaned = strings.TrimSuffix(cleaned, "```")
	cleaned = strings.TrimSpace(cleaned)

	var draft geminiBusinessTermDraft
	if err := json.Unmarshal([]byte(cleaned), &draft); err != nil {
		return nil, fmt.Errorf("failed to parse gemini business term draft: %w (raw: %s)", err, cleaned)
	}
	if strings.TrimSpace(draft.Name) == "" {
		return nil, fmt.Errorf("gemini returned an empty business term name")
	}
	return &draft, nil
}

// findClosestBusinessTerm returns the existing term whose name is most
// similar to candidateName, and its similarity score in [0,1].
func findClosestBusinessTerm(candidateName string, existing []SemanticTerm) (*SemanticTerm, float64) {
	var best *SemanticTerm
	bestScore := 0.0
	for i := range existing {
		score := nameSimilarity(candidateName, existing[i].TermName)
		if score > bestScore {
			bestScore = score
			best = &existing[i]
		}
	}
	return best, bestScore
}

// nameSimilarity blends two signals so abbreviation drift ("Cust Acct No" vs
// "Customer Account Number") scores as highly as a straight typo:
//   - character-level Levenshtein similarity over normalized names
//   - token-level Jaccard similarity after expanding known abbreviations
//     (reusing the same dictionary the column matcher already uses, so
//     abbreviations are defined once, not twice)
//
// The max of the two is used: abbreviation-heavy names win on the token
// score even when their raw character distance is large, while near-typos
// with no abbreviations still win on the character score.
func nameSimilarity(a, b string) float64 {
	na, nb := normalizeTermName(a), normalizeTermName(b)
	if na == "" || nb == "" {
		return 0
	}
	if na == nb {
		return 1.0
	}

	dist := levenshtein(na, nb)
	maxLen := len(na)
	if len(nb) > maxLen {
		maxLen = len(nb)
	}
	charScore := 0.0
	if maxLen > 0 {
		charScore = 1.0 - float64(dist)/float64(maxLen)
	}

	tokenScore := tokenJaccardSimilarity(expandAbbreviatedTokens(a), expandAbbreviatedTokens(b))

	if tokenScore > charScore {
		return tokenScore
	}
	return charScore
}

var abbreviationExpansionMap = initializeAbbreviationMap()

// expandAbbreviatedTokens splits a term name into tokens (snake_case,
// kebab-case, camelCase, and whitespace boundaries) and expands any token
// that is a known abbreviation to its canonical form, so "ACCT" and
// "ACCOUNT" land in the same bucket for Jaccard comparison.
func expandAbbreviatedTokens(name string) []string {
	s := strings.ReplaceAll(name, "_", " ")
	s = strings.ReplaceAll(s, "-", " ")
	camelRegex := regexp.MustCompile("([a-z0-9])([A-Z])")
	s = camelRegex.ReplaceAllString(s, "$1 $2")

	rawTokens := strings.Fields(strings.ToUpper(s))
	tokens := make([]string, 0, len(rawTokens))
	for _, t := range rawTokens {
		if len(t) < 2 {
			continue
		}
		if expansion, ok := abbreviationExpansionMap[t]; ok {
			tokens = append(tokens, expansion)
		} else {
			tokens = append(tokens, t)
		}
	}
	return tokens
}

func tokenJaccardSimilarity(tokens1, tokens2 []string) float64 {
	if len(tokens1) == 0 && len(tokens2) == 0 {
		return 1.0
	}
	if len(tokens1) == 0 || len(tokens2) == 0 {
		return 0.0
	}

	set1 := make(map[string]struct{}, len(tokens1))
	for _, t := range tokens1 {
		set1[t] = struct{}{}
	}
	set2 := make(map[string]struct{}, len(tokens2))
	intersection := 0
	for _, t := range tokens2 {
		set2[t] = struct{}{}
		if _, ok := set1[t]; ok {
			intersection++
		}
	}

	union := len(set1)
	for t := range set2 {
		if _, ok := set1[t]; !ok {
			union++
		}
	}
	if union == 0 {
		return 0
	}
	return float64(intersection) / float64(union)
}

func normalizeTermName(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	var b strings.Builder
	prevSpace := false
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
			prevSpace = false
		default:
			if !prevSpace {
				b.WriteRune(' ')
				prevSpace = true
			}
		}
	}
	return strings.TrimSpace(b.String())
}

func levenshtein(a, b string) int {
	ra, rb := []rune(a), []rune(b)
	m, n := len(ra), len(rb)
	if m == 0 {
		return n
	}
	if n == 0 {
		return m
	}
	prev := make([]int, n+1)
	curr := make([]int, n+1)
	for j := 0; j <= n; j++ {
		prev[j] = j
	}
	for i := 1; i <= m; i++ {
		curr[0] = i
		for j := 1; j <= n; j++ {
			cost := 1
			if ra[i-1] == rb[j-1] {
				cost = 0
			}
			del := prev[j] + 1
			ins := curr[j-1] + 1
			sub := prev[j-1] + cost
			min := del
			if ins < min {
				min = ins
			}
			if sub < min {
				min = sub
			}
			curr[j] = min
		}
		prev, curr = curr, prev
	}
	return prev[n]
}

// businessTermToSemanticTermEdgeType is the edge type name actually seeded in
// the live catalog (verified against the running instance) for connecting a
// business term to the semantic term it represents. The 'has_semantic' name
// referenced by backend/migrations/20260125_fix_has_semantic_edge_type.sql
// was never actually seeded — that migration no-ops when its tenant/node-type
// lookups miss (see its own RAISE NOTICE guard) — and 'has_semantic_context'
// is what's really there, with no source/target node-type constraint, unlike
// 'USES_SEMANTIC_TERM' which is scoped to business_object->semantic_model.
const businessTermToSemanticTermEdgeType = "has_semantic_context"

// LinkBusinessTermToSemanticTerm creates the edge connecting a business term
// to the semantic term it represents, resolving (or creating, if genuinely
// missing for this tenant) businessTermToSemanticTermEdgeType. It is
// idempotent.
func (s *SemanticMappingService) LinkBusinessTermToSemanticTerm(ctx context.Context, tenantID, businessTermID, semanticTermID string) error {
	edgeTypeID, err := resolveOrCreateEdgeTypeID(ctx, s.db, tenantID, businessTermToSemanticTermEdgeType)
	if err != nil {
		return fmt.Errorf("failed to resolve %s edge type: %w", businessTermToSemanticTermEdgeType, err)
	}

	edgeID := uuid.New().String()
	_, err = s.db.ExecContext(ctx, `
		INSERT INTO catalog_edge (id, source_node_id, target_node_id, edge_type_id, tenant_id, properties, created_at)
		SELECT $1, $2, $3, $4, $5, '{}'::jsonb, $6
		WHERE NOT EXISTS (
			SELECT 1 FROM catalog_edge
			WHERE source_node_id = $2 AND target_node_id = $3 AND edge_type_id = $4 AND tenant_id = $5
		)
	`, edgeID, businessTermID, semanticTermID, edgeTypeID, tenantID, time.Now())
	if err != nil {
		return fmt.Errorf("failed to link business term to semantic term: %w", err)
	}
	return nil
}
