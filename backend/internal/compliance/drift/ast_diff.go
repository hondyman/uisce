package drift

import (
	"fmt"
	"reflect"
	"strings"
)

// ASTDiffType represents the category of change in an AST
type ASTDiffType string

const (
	DiffNodeAdded       ASTDiffType = "NODE_ADDED"
	DiffNodeRemoved     ASTDiffType = "NODE_REMOVED"
	DiffThresholdChange ASTDiffType = "THRESHOLD_CHANGED"
	DiffOperatorChange  ASTDiffType = "OPERATOR_CHANGED"
	DiffFieldRefChange  ASTDiffType = "FIELD_REF_CHANGED"
)

// ASTDiffEntry represents a single structural difference between two ASTs
type ASTDiffEntry struct {
	Type        ASTDiffType `json:"type"`
	Path        string      `json:"path"`
	OldValue    interface{} `json:"oldValue,omitempty"`
	NewValue    interface{} `json:"newValue,omitempty"`
	Description string      `json:"description"`
}

// ASTSemanticDiff contains the structured differences and human-readable summary
type ASTSemanticDiff struct {
	HasBreakingChanges bool           `json:"hasBreakingChanges"`
	DiffEntries        []ASTDiffEntry `json:"diffEntries"`
	SummaryText        string         `json:"summaryText"`
}

// CompareAST computes a structure-aware semantic diff between two rule AST maps
func CompareAST(oldAST, newAST map[string]interface{}) *ASTSemanticDiff {
	diff := &ASTSemanticDiff{
		DiffEntries: make([]ASTDiffEntry, 0),
	}

	compareNodes(oldAST, newAST, "root", diff)

	// Generate human-readable summary
	var sb strings.Builder
	if len(diff.DiffEntries) == 0 {
		sb.WriteString("No semantic changes detected between rule versions.")
	} else {
		sb.WriteString(fmt.Sprintf("Detected %d structural change(s):\n", len(diff.DiffEntries)))
		for i, entry := range diff.DiffEntries {
			sb.WriteString(fmt.Sprintf(" %d. [%s] %s: %s\n", i+1, entry.Type, entry.Path, entry.Description))
		}
	}
	diff.SummaryText = sb.String()

	return diff
}

func compareNodes(oldNode, newNode map[string]interface{}, path string, diff *ASTSemanticDiff) {
	if oldNode == nil && newNode == nil {
		return
	}
	if oldNode == nil && newNode != nil {
		diff.DiffEntries = append(diff.DiffEntries, ASTDiffEntry{
			Type:        DiffNodeAdded,
			Path:        path,
			NewValue:    newNode,
			Description: fmt.Sprintf("New condition branch added at %s", path),
		})
		diff.HasBreakingChanges = true
		return
	}
	if oldNode != nil && newNode == nil {
		diff.DiffEntries = append(diff.DiffEntries, ASTDiffEntry{
			Type:        DiffNodeRemoved,
			Path:        path,
			OldValue:    oldNode,
			Description: fmt.Sprintf("Condition branch removed at %s", path),
		})
		diff.HasBreakingChanges = true
		return
	}

	// Compare Operator
	oldOp, _ := oldNode["operator"].(string)
	newOp, _ := newNode["operator"].(string)
	if oldOp != newOp {
		diff.DiffEntries = append(diff.DiffEntries, ASTDiffEntry{
			Type:        DiffOperatorChange,
			Path:        path + ".operator",
			OldValue:    oldOp,
			NewValue:    newOp,
			Description: fmt.Sprintf("Operator changed from %q to %q", oldOp, newOp),
		})
		diff.HasBreakingChanges = true
	}

	// Compare Field Reference / Target Field
	oldField, _ := oldNode["field"].(string)
	newField, _ := newNode["field"].(string)
	if oldField != newField {
		diff.DiffEntries = append(diff.DiffEntries, ASTDiffEntry{
			Type:        DiffFieldRefChange,
			Path:        path + ".field",
			OldValue:    oldField,
			NewValue:    newField,
			Description: fmt.Sprintf("Target field changed from %q to %q", oldField, newField),
		})
		diff.HasBreakingChanges = true
	}

	// Compare Threshold / Value
	oldVal := oldNode["value"]
	newVal := newNode["value"]
	if !reflect.DeepEqual(oldVal, newVal) && oldVal != nil && newVal != nil {
		diff.DiffEntries = append(diff.DiffEntries, ASTDiffEntry{
			Type:        DiffThresholdChange,
			Path:        path + ".value",
			OldValue:    oldVal,
			NewValue:    newVal,
			Description: fmt.Sprintf("Threshold value updated from %v to %v", oldVal, newVal),
		})
	}

	// Recursively compare children / conditions
	oldChildren, _ := oldNode["conditions"].([]interface{})
	newChildren, _ := newNode["conditions"].([]interface{})

	maxLen := len(oldChildren)
	if len(newChildren) > maxLen {
		maxLen = len(newChildren)
	}

	for i := 0; i < maxLen; i++ {
		childPath := fmt.Sprintf("%s.conditions[%d]", path, i)
		var oldChildMap, newChildMap map[string]interface{}
		if i < len(oldChildren) {
			oldChildMap, _ = oldChildren[i].(map[string]interface{})
		}
		if i < len(newChildren) {
			newChildMap, _ = newChildren[i].(map[string]interface{})
		}
		compareNodes(oldChildMap, newChildMap, childPath, diff)
	}
}
