package healing

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/hondyman/semlayer/backend/internal/pagestudio"
	"github.com/hondyman/semlayer/backend/internal/semantic_intelligence/drift"
)

type HealingEngine struct {
	strategies []Strategy
}

func NewHealingEngine() *HealingEngine {
	return &HealingEngine{
		strategies: []Strategy{
			&FieldRenameStrategy{},
		},
	}
}

// GenerateProposal attempts to fix all drift events in a page
func (h *HealingEngine) GenerateProposal(ctx context.Context, page *pagestudio.CorePage, report *drift.DriftReport) (*HealProposal, error) {
	// Deep copy page to avoid mutating original during trial
	pageCopy := *page
	// Copy slices/maps if needed (Components, DataBindings are json.RawMessage so effectively byte slices)
	// For simple modification we assume json.Unmarshal -> Modify -> json.Marshal pattern

	proposal := &HealProposal{
		PageID:  page.ID.String(),
		Changes: make([]string, 0),
	}

	for _, event := range report.Events {
		healed := false
		for _, strategy := range h.strategies {
			if strategy.CanHeal(event) {
				changes, err := strategy.Heal(ctx, &pageCopy, event)
				if err == nil && len(changes) > 0 {
					proposal.Changes = append(proposal.Changes, changes...)
					healed = true
					break // One fix per event
				}
			}
		}
		if !healed {
			proposal.Changes = append(proposal.Changes, fmt.Sprintf("⚠️ Could not auto-heal: %s", event.Description))
		}
	}

	proposal.HealedPage = pageCopy
	if len(proposal.Changes) > 0 {
		proposal.Description = fmt.Sprintf("Proposed fixes for %d drift events", len(report.Events))
	} else {
		proposal.Description = "No fixes possible or needed"
	}

	return proposal, nil
}

// --- Strategies ---

type FieldRenameStrategy struct{}

func (s *FieldRenameStrategy) Name() string { return "FieldRename" }

func (s *FieldRenameStrategy) CanHeal(event drift.DriftEvent) bool {
	return event.Type == drift.DriftTypeFieldRemoved || event.Type == drift.DriftTypeEndpointMissing
}

func (s *FieldRenameStrategy) Heal(ctx context.Context, page *pagestudio.CorePage, event drift.DriftEvent) ([]string, error) {
	// 1. Identify what to replace
	// Mock logic: If field "market_value" is missing, replace with "market_value_usd"

	targetField := event.ItemName
	// In real world, we'd look up renaming history or use Similarity matching
	suggestedField := ""
	if targetField == "market_value" {
		suggestedField = "market_value_usd"
	}

	if suggestedField == "" {
		return nil, nil // Cannot heal
	}

	// 2. Perform replacement in Components JSON
	// This is effectively a text find-and-replace in the JSON blob for MVP
	// A proper implementation would parse the component tree
	compJSON := string(page.Components)
	newCompJSON := strings.ReplaceAll(compJSON, fmt.Sprintf(`"field": "%s"`, targetField), fmt.Sprintf(`"field": "%s"`, suggestedField))
	newCompJSON = strings.ReplaceAll(newCompJSON, fmt.Sprintf(`"dataKey": "%s"`, targetField), fmt.Sprintf(`"dataKey": "%s"`, suggestedField))

	if compJSON != newCompJSON {
		page.Components = json.RawMessage(newCompJSON)
		return []string{fmt.Sprintf("Renamed field '%s' to '%s' in components", targetField, suggestedField)}, nil
	}

	return nil, nil
}
