package api

import (
	"context"
	"strings"
)

// abbreviationLookup abstracts the abbreviation lookup needed by deriveTermNamesPreview.
type abbreviationLookup interface {
	GetAllAbbreviations(ctx context.Context) ([]map[string]string, error)
}

// deriveTermNamesPreview is the deterministic-only path used by
// POST /api/glossary/preview-semantic-terms. It tokenizes, expands
// abbreviations via the abbreviation table, applies the address-line
// contextual rule, and assembles PascalCase/titleCase. It MUST NOT
// call any LLM and MUST NOT persist anything to the abbreviation table.
func deriveTermNamesPreview(ctx context.Context, abbrevSvc abbreviationLookup, tenantID, rawName, tableSchemaContext string) derivedTermNames {
	tokens := tokenizeColumnName(rawName)
	if len(tokens) == 0 {
		return derivedTermNames{}
	}

	resolved := tokens
	if abbrevSvc != nil {
		svcCtx := context.WithValue(ctx, "tenant_id", tenantID)
		abbrevs, err := abbrevSvc.GetAllAbbreviations(svcCtx)
		if err == nil && len(abbrevs) > 0 {
			abbrMap := make(map[string]string, len(abbrevs))
			for _, a := range abbrevs {
				abbrMap[strings.ToUpper(a["abbreviation"])] = a["full_word"]
			}

			resolved = make([]string, len(tokens))
			for i, tok := range tokens {
				upper := strings.ToUpper(tok)
				if full, ok := abbrMap[upper]; ok {
					resolved[i] = full
				} else {
					resolved[i] = tok
				}
			}
		}
	}

	result := deriveTermNamesDeterministic(resolved, rawName, tableSchemaContext)

	if result.GetSource() == "pascal" && !sameStringSlice(tokens, resolved) {
		result.source = "abbrev_map"
	}

	return result
}

// deriveTermNamesPreviewCandidates returns the ranked candidate list for a column,
// applying abbreviation expansion but no LLM. Used by PreviewSemanticTerms to
// implement rejection filtering.
func deriveTermNamesPreviewCandidates(ctx context.Context, abbrevSvc abbreviationLookup, tenantID, rawName, tableSchemaContext string) []CandidateTerm {
	tokens := tokenizeColumnName(rawName)
	if len(tokens) == 0 {
		return nil
	}

	resolved := tokens
	if abbrevSvc != nil {
		svcCtx := context.WithValue(ctx, "tenant_id", tenantID)
		abbrevs, err := abbrevSvc.GetAllAbbreviations(svcCtx)
		if err == nil && len(abbrevs) > 0 {
			abbrMap := make(map[string]string, len(abbrevs))
			for _, a := range abbrevs {
				abbrMap[strings.ToUpper(a["abbreviation"])] = a["full_word"]
			}

			resolved = make([]string, len(tokens))
			for i, tok := range tokens {
				upper := strings.ToUpper(tok)
				if full, ok := abbrMap[upper]; ok {
					resolved[i] = full
				} else {
					resolved[i] = tok
				}
			}
		}
	}

	return deriveTermNamesCandidates(resolved, rawName, tableSchemaContext)
}

// rejectionSet is a flat set of rejected (datasourceID, qualifiedPath, rejectedName) triples
// for efficient O(1) lookup during preview filtering.
type rejectionSet map[string]struct{} // key = datasourceID + "\x00" + qualifiedPath + "\x00" + rejectedName

// makeRejectionKey constructs the lookup key used by rejectionSet.
func makeRejectionKey(datasourceID, qualifiedPath, rejectedName string) string {
	return datasourceID + "\x00" + qualifiedPath + "\x00" + rejectedName
}

// pickFirstNonRejected returns the first candidate not in the rejection set,
// or the naive PascalCase fallback if all candidates are rejected.
// The fallback always exists as the last candidate from deriveTermNamesCandidates.
// datasourceID is used to scope the rejection lookup to the correct datasource.
func pickFirstNonRejected(candidates []CandidateTerm, rejections rejectionSet, datasourceID, qualifiedPath string) (name string, source string) {
	for _, c := range candidates {
		key := makeRejectionKey(datasourceID, qualifiedPath, c.Name)
		if _, rejected := rejections[key]; !rejected {
			return c.Name, c.Source
		}
	}
	// All rejected: fall back to naive PascalCase (always last candidate)
	if len(candidates) > 0 {
		last := candidates[len(candidates)-1]
		return last.Name, "pascal"
	}
	return "", "pascal"
}

func sameStringSlice(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
