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
