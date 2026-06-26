package analytics

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/hondyman/semlayer/backend/internal/logging"
)

// EnrichTermsFromFeedback scans approved feedback to enrich business term properties
// specifically adding synonyms and abbreviations found in approved mappings.
func (s *SemanticMappingService) EnrichTermsFromFeedback(ctx context.Context, tenantID string) (int, error) {
	logger := logging.GetLogger().Sugar()
	logger.Infof("Starting term enrichment from feedback for tenant %s", tenantID)

	// 1. Query approved mappings from term_ai_feedback
	// We want to find terms that have been approved multiple times with different column names
	query := `
		SELECT term_id, features->>'column_name' as column_name
		FROM term_ai_feedback
		WHERE tenant_id = $1 AND action = 'approved' AND term_id IS NOT NULL
	`

	type TermMapping struct {
		TermID     string `db:"term_id"`
		ColumnName string `db:"column_name"`
	}

	var mappings []TermMapping
	err := s.db.SelectContext(ctx, &mappings, query, tenantID)
	if err != nil {
		return 0, fmt.Errorf("failed to query feedback: %w", err)
	}

	// 2. Aggregate synonyms by term
	termSynonyms := make(map[string]map[string]bool)
	for _, m := range mappings {
		if m.ColumnName == "" {
			continue
		}
		if termSynonyms[m.TermID] == nil {
			termSynonyms[m.TermID] = make(map[string]bool)
		}
		// Treat column name as potential synonym/abbreviation
		termSynonyms[m.TermID][strings.ToLower(m.ColumnName)] = true
	}

	updatedCount := 0

	// 3. For each term, update properties
	for termID, synonyms := range termSynonyms {
		// Fetch current term properties
		var propsJSON []byte
		err := s.db.GetContext(ctx, &propsJSON, "SELECT properties FROM businessterm WHERE termid = $1", termID)
		if err != nil {
			logger.Warnf("Failed to fetch term %s: %v", termID, err)
			continue
		}

		var props map[string]interface{}
		if err := json.Unmarshal(propsJSON, &props); err != nil {
			logger.Warnf("Failed to unmarshal properties for term %s: %v", termID, err)
			continue
		}

		if props == nil {
			props = make(map[string]interface{})
		}

		// Get existing synonyms/abbreviations
		existingSynonyms := make(map[string]bool)
		if synRaw, ok := props["synonyms"].([]interface{}); ok {
			for _, s := range synRaw {
				if str, ok := s.(string); ok {
					existingSynonyms[strings.ToLower(str)] = true
				}
			}
		}

		// Merge new synonyms
		changed := false
		var newSynonymList []string

		// Keep existing
		for s := range existingSynonyms {
			newSynonymList = append(newSynonymList, s)
		}

		for syn := range synonyms {
			if !existingSynonyms[syn] {
				newSynonymList = append(newSynonymList, syn)
				changed = true
			}
		}

		if changed {
			props["synonyms"] = newSynonymList

			// Also add to abbreviations if short (<= 5 chars)
			// This is a naive heuristic but fits the requirement
			var abbreviations []string
			if abbrRaw, ok := props["abbreviation"].([]interface{}); ok {
				for _, a := range abbrRaw {
					if str, ok := a.(string); ok {
						abbreviations = append(abbreviations, str)
					}
				}
			} else if abbrStr, ok := props["abbreviation"].(string); ok {
				abbreviations = append(abbreviations, abbrStr)
			}

			abbrChanged := false
			existingAbbr := make(map[string]bool)
			for _, a := range abbreviations {
				existingAbbr[strings.ToLower(a)] = true
			}

			for syn := range synonyms {
				if len(syn) <= 5 && !existingAbbr[syn] {
					abbreviations = append(abbreviations, syn)
					abbrChanged = true
				}
			}

			if abbrChanged {
				props["abbreviation"] = abbreviations
			}

			// Update database
			updatedPropsJSON, _ := json.Marshal(props)
			_, err := s.db.ExecContext(ctx, "UPDATE businessterm SET properties = $1 WHERE termid = $2", updatedPropsJSON, termID)
			if err != nil {
				logger.Errorf("Failed to update term %s: %v", termID, err)
			} else {
				updatedCount++
				logger.Infof("Enriched term %s with %d new synonyms", termID, len(newSynonymList)-len(existingSynonyms))
			}
		}
	}

	return updatedCount, nil
}
