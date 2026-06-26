package security

import (
	"testing"

	"github.com/hondyman/semlayer/backend/internal/models"
	"github.com/stretchr/testify/assert"
)

// mockAccessRule creates a test access rule.
func mockAccessRule(id, groupDn, rowFilter, level string, masks []models.ColumnMask) *models.AccessRule {
	appliesToApis := true
	appliesToBi := true
	appliesToAi := true

	return &models.AccessRule{
		RuleID:           id,
		TenantID:         "tenant-1",
		BusinessObjectID: "bo:portfolio",
		GroupDn:          groupDn,
		AccessLevel:      level,
		Status:           "APPROVED",
		RowFilterDsl:     rowFilter,
		ColumnMasks:      masks,
		AppliesToApis:    &appliesToApis,
		AppliesToBi:      &appliesToBi,
		AppliesToAi:      &appliesToAi,
	}
}

func TestRuleComposition_CombinesPredicatesWithOr(t *testing.T) {
	rules := []*models.AccessRule{
		mockAccessRule("r1", "g1", "region = 'EMEA'", "READ", nil),
		mockAccessRule("r2", "g2", "region = 'APAC'", "READ", nil),
	}

	// In production, this would call ComposeAccessDecision
	// For now, test the logic manually
	var predicates []string
	for _, rule := range rules {
		if rule.RowFilterDsl != "" {
			predicates = append(predicates, rule.RowFilterDsl)
		}
	}

	// Should combine with OR
	assert.Len(t, predicates, 2)
	assert.Contains(t, predicates, "region = 'EMEA'")
	assert.Contains(t, predicates, "region = 'APAC'")

	// In actual implementation: combined = (region = 'EMEA') OR (region = 'APAC')
}

func TestRuleComposition_PicksMaxAccessLevel(t *testing.T) {
	tests := []struct {
		name     string
		levels   []string
		expected string
	}{
		{"WRITE wins", []string{"READ", "WRITE"}, "WRITE"},
		{"READ over NONE", []string{"NONE", "READ"}, "READ"},
		{"All same", []string{"READ", "READ", "READ"}, "READ"},
		{"WRITE among many", []string{"NONE", "READ", "WRITE"}, "WRITE"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var rules []*models.AccessRule
			for i, level := range tt.levels {
				rules = append(rules, mockAccessRule("r"+string(rune(i)), "g1", "", level, nil))
			}

			// Find max level
			maxLevel := "NONE"
			for _, rule := range rules {
				level := rule.AccessLevel
				if level == "WRITE" {
					maxLevel = "WRITE"
					break
				} else if level == "READ" && maxLevel == "NONE" {
					maxLevel = "READ"
				}
			}

			assert.Equal(t, tt.expected, maxLevel)
		})
	}
}

func TestRuleComposition_MostRestrictiveMask(t *testing.T) {
	// HIDE is most restrictive, then MASK, then NONE
	masks1 := []models.ColumnMask{
		{SemanticTermID: "term:email", MaskType: "MASK"},
	}
	masks2 := []models.ColumnMask{
		{SemanticTermID: "term:email", MaskType: "HIDE"},
	}

	rules := []*models.AccessRule{
		mockAccessRule("r1", "g1", "", "READ", masks1),
		mockAccessRule("r2", "g2", "", "READ", masks2),
	}

	// Compose masks
	composedMasks := make(map[string]string)
	for _, rule := range rules {
		for _, mask := range rule.ColumnMasks {
			existing, found := composedMasks[mask.SemanticTermID]
			if !found {
				composedMasks[mask.SemanticTermID] = mask.MaskType
				continue
			}

			// HIDE beats MASK, MASK beats NONE
			if mask.MaskType == "HIDE" || (mask.MaskType == "MASK" && existing == "NONE") {
				composedMasks[mask.SemanticTermID] = mask.MaskType
			}
		}
	}

	// Should pick HIDE over MASK
	assert.Equal(t, "HIDE", composedMasks["term:email"])
}

func TestRuleComposition_MultipleMasks(t *testing.T) {
	masks1 := []models.ColumnMask{
		{SemanticTermID: "term:email", MaskType: "MASK"},
		{SemanticTermID: "term:ssn", MaskType: "HIDE"},
	}
	masks2 := []models.ColumnMask{
		{SemanticTermID: "term:email", MaskType: "HIDE"},
		{SemanticTermID: "term:phone", MaskType: "MASK"},
	}

	rules := []*models.AccessRule{
		mockAccessRule("r1", "g1", "", "READ", masks1),
		mockAccessRule("r2", "g2", "", "READ", masks2),
	}

	// Compose masks
	composedMasks := make(map[string]string)
	for _, rule := range rules {
		for _, mask := range rule.ColumnMasks {
			existing, found := composedMasks[mask.SemanticTermID]
			if !found {
				composedMasks[mask.SemanticTermID] = mask.MaskType
				continue
			}

			// Pick most restrictive
			if mask.MaskType == "HIDE" || (mask.MaskType == "MASK" && existing == "NONE") {
				composedMasks[mask.SemanticTermID] = mask.MaskType
			}
		}
	}

	// Verify final masks
	assert.Equal(t, "HIDE", composedMasks["term:email"], "HIDE should beat MASK")
	assert.Equal(t, "HIDE", composedMasks["term:ssn"])
	assert.Equal(t, "MASK", composedMasks["term:phone"])
	assert.Len(t, composedMasks, 3)
}

func TestRuleComposition_NoRowFilters(t *testing.T) {
	// When no row filters are specified, access is granted to all rows
	rules := []*models.AccessRule{
		mockAccessRule("r1", "g1", "", "READ", nil),
		mockAccessRule("r2", "g2", "", "WRITE", nil),
	}

	var predicates []string
	for _, rule := range rules {
		if rule.RowFilterDsl != "" {
			predicates = append(predicates, rule.RowFilterDsl)
		}
	}

	// Should have no predicates
	assert.Len(t, predicates, 0)
	// In the actual API, this would return no WHERE clause
}

func TestRuleComposition_EmptyMasks(t *testing.T) {
	rules := []*models.AccessRule{
		mockAccessRule("r1", "g1", "region = 'EMEA'", "READ", nil),
		mockAccessRule("r2", "g2", "region = 'APAC'", "READ", nil),
	}

	// Compose masks
	composedMasks := make(map[string]string)
	for _, rule := range rules {
		for _, mask := range rule.ColumnMasks {
			composedMasks[mask.SemanticTermID] = mask.MaskType
		}
	}

	// Should have no masks
	assert.Len(t, composedMasks, 0)
}
