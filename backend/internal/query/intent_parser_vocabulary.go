package query

import (
	"context"
	"strings"

	"github.com/hondyman/uisce/backend/internal/ai/vocabulary"
)

type IntentParserWithVocabulary struct {
	baseParser *IntentParser
	vocab      *vocabulary.Resolver
}

func NewIntentParserWithVocabulary(base *IntentParser, vocab *vocabulary.Resolver) *IntentParserWithVocabulary {
	return &IntentParserWithVocabulary{
		baseParser: base,
		vocab:      vocab,
	}
}

func (ip *IntentParserWithVocabulary) ParseIntentWithTenant(ctx context.Context, tenantID, text string) (*ParsedIntent, error) {
	intent, err := ip.baseParser.ParseIntent(text)
	if err != nil {
		return nil, err
	}

	if ip.vocab == nil || ip.vocab.DB() == nil {
		return intent, nil
	}

	rawTokens := extractTokens(text)
	var resolvedTerms []string
	var resolvedFilters []IntentFilter

	for _, raw := range rawTokens {
		cleaned := strings.Trim(cleanedToken(raw), ".,?!'\"():;")
		if cleaned == "" {
			continue
		}

		canonicalTerms, err := ip.vocab.ResolveTerm(ctx, tenantID, cleaned)
		if err == nil && len(canonicalTerms) > 0 {
			for _, term := range canonicalTerms {
				if term.TermName != "" {
					resolvedTerms = append(resolvedTerms, term.TermName)
				}
				if term.SemanticTerm != nil && *term.SemanticTerm != "" {
					resolvedTerms = append(resolvedTerms, *term.SemanticTerm)
				}
				if term.MatchedVia != "" && term.MatchedToken != "" {
					resolvedFilters = append(resolvedFilters, IntentFilter{
						Field:    term.MatchedToken,
						Operator: "SYNONYM_OF",
						Values:  []string{term.TermName},
					})
				}
			}
		}
	}

	if len(resolvedTerms) > 0 {
		unique := removeDuplicates(resolvedTerms)
		if intent.RawEntities == nil {
			intent.RawEntities = make(map[string]string)
		}
		intent.RawEntities["resolved_graph_terms"] = strings.Join(unique, ", ")
		if intent.Confidence < 0.95 {
			intent.Confidence = 0.95
		}
		for _, f := range resolvedFilters {
			found := false
			for _, ef := range intent.Filters {
				if ef.Field == f.Field {
					found = true
					break
				}
			}
			if !found {
				intent.Filters = append(intent.Filters, f)
			}
		}
	}

	return intent, nil
}

func extractTokens(text string) []string {
	text = strings.ToLower(text)
	raw := strings.Fields(text)
	var tokens []string
	for _, t := range raw {
		cleaned := strings.Trim(t, ".,?!'\"():;")
		if len(cleaned) > 1 {
			tokens = append(tokens, cleaned)
		}
	}
	return tokens
}

func cleanedToken(s string) string {
	s = strings.TrimPrefix(s, "(")
	s = strings.TrimPrefix(s, ")")
	return strings.Trim(s, ".,?!'\"():;")
}
