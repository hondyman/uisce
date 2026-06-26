package catalog

import (
	"sort"
	"strings"
)

// CandidateGroup represents a cluster of related columns and semantic terms
type CandidateGroup struct {
	SemanticTerms []SemanticTerm    `json:"semantic_terms"`
	Columns       []TechnicalColumn `json:"columns"`
}

// GroupCandidates clusters columns and semantic terms into candidate concepts
func GroupCandidates(columns []TechnicalColumn, semanticTerms []SemanticTerm) []CandidateGroup {
	usedColIDs := make(map[string]bool) // Key: table.column
	var groups []CandidateGroup

	// 1. Seed groups from semantic terms
	for _, st := range semanticTerms {
		stTokens := tokenize(st.Name)
		group := CandidateGroup{
			SemanticTerms: []SemanticTerm{st},
			Columns:       []TechnicalColumn{},
		}

		for _, col := range columns {
			colID := col.Table + "." + col.Column
			if usedColIDs[colID] {
				continue
			}

			colTokens := tokenize(col.Column)
			// Heuristic: overlap of >= 2 tokens (e.g., client_address vs client_address_line1)
			if overlapCount(stTokens, colTokens) >= 2 {
				group.Columns = append(group.Columns, col)
				usedColIDs[colID] = true
			}
		}

		// Only add if we found related columns or just want to propose based on semantic term alone
		groups = append(groups, group)
	}

	// 2. Remaining columns: group by shared table + prefix
	// Bucket by (table, first_token)
	buckets := make(map[string][]TechnicalColumn)
	for _, col := range columns {
		colID := col.Table + "." + col.Column
		if usedColIDs[colID] {
			continue
		}

		tokens := tokenize(col.Column)
		if len(tokens) == 0 {
			continue
		}

		// Sort tokens to find a deterministic "primary" token, or just use the first split
		// Let's use the first part of split by '_' as a simple prefix
		parts := strings.Split(col.Column, "_")
		if len(parts) > 0 {
			key := col.Table + ":" + parts[0]
			buckets[key] = append(buckets[key], col)
		}
	}

	for _, cols := range buckets {
		// Only suggest groups with sufficient mass (>= 2 columns)
		if len(cols) >= 2 {
			groups = append(groups, CandidateGroup{
				SemanticTerms: []SemanticTerm{},
				Columns:       cols,
			})
		}
	}

	return groups
}

func tokenize(s string) map[string]bool {
	tokens := make(map[string]bool)
	parts := strings.FieldsFunc(s, func(r rune) bool {
		return r == '_' || r == '.' || r == ' '
	})
	for _, p := range parts {
		if len(p) > 2 { // Ignore short/noise tokens
			tokens[strings.ToLower(p)] = true
		}
	}
	return tokens
}

func overlapCount(setA, setB map[string]bool) int {
	count := 0
	for k := range setA {
		if setB[k] {
			count++
		}
	}
	return count
}

// Deterministic helpers if needed
func sortedKeys(m map[string][]TechnicalColumn) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
