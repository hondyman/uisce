package metadata

import (
	"fmt"
)

// ChangeType defines the nature of a metadata change
type ChangeType string

const (
	ChangeTypeAttributeAdded   ChangeType = "AttributeAdded"
	ChangeTypeAttributeRemoved ChangeType = "AttributeRemoved"
	ChangeTypeAttributeChanged ChangeType = "AttributeChanged"
	ChangeTypeTransitionAdded  ChangeType = "TransitionAdded"
	ChangeTypeTransitionRemoved ChangeType = "TransitionRemoved"
)

// Diff represents a single change between two metadata versions
type Diff struct {
	Type        ChangeType      `json:"type"`
	Path        string          `json:"path"` // e.g., "attributes.risk_rating"
	OldValue    interface{}     `json:"old_value,omitempty"`
	NewValue    interface{}     `json:"new_value,omitempty"`
	IsBreaking  bool            `json:"is_breaking"`
	Description string          `json:"description"`
}

// DiffEngine compares metadata objects
type DiffEngine struct{}

func NewDiffEngine() *DiffEngine {
	return &DiffEngine{}
}

// DiffBusinessObjects compares two versions of a Business Object
func (e *DiffEngine) DiffBusinessObjects(oldBO, newBO BusinessObject) []Diff {
	var diffs []Diff

	// 1. Compare Attributes
	oldAttrs := make(map[string]BOAttribute)
	for _, a := range oldBO.Attributes {
		oldAttrs[a.Name] = a
	}

	newAttrs := make(map[string]BOAttribute)
	for _, a := range newBO.Attributes {
		newAttrs[a.Name] = a
	}

	// Check for Added or Changed
	for name, newAttr := range newAttrs {
		if oldAttr, exists := oldAttrs[name]; exists {
			// Attribute exists in both, check for changes
			if oldAttr.Type != newAttr.Type {
				diffs = append(diffs, Diff{
					Type:        ChangeTypeAttributeChanged,
					Path:        fmt.Sprintf("attributes.%s.type", name),
					OldValue:    oldAttr.Type,
					NewValue:    newAttr.Type,
					IsBreaking:  true, // Type change is breaking
					Description: fmt.Sprintf("Attribute '%s' type changed from %s to %s", name, oldAttr.Type, newAttr.Type),
				})
			}
			// Check other fields like Validation, Required, etc.
			if oldAttr.Required != newAttr.Required {
				diffs = append(diffs, Diff{
					Type:        ChangeTypeAttributeChanged,
					Path:        fmt.Sprintf("attributes.%s.required", name),
					OldValue:    oldAttr.Required,
					NewValue:    newAttr.Required,
					IsBreaking:  newAttr.Required, // Making optional->required is breaking
					Description: fmt.Sprintf("Attribute '%s' required status changed", name),
				})
			}
		} else {
			// Attribute Added
			diffs = append(diffs, Diff{
				Type:        ChangeTypeAttributeAdded,
				Path:        fmt.Sprintf("attributes.%s", name),
				NewValue:    newAttr,
				IsBreaking:  newAttr.Required, // Adding required attr is breaking for existing data
				Description: fmt.Sprintf("Attribute '%s' added", name),
			})
		}
	}

	// Check for Removed
	for name, oldAttr := range oldAttrs {
		if _, exists := newAttrs[name]; !exists {
			diffs = append(diffs, Diff{
				Type:        ChangeTypeAttributeRemoved,
				Path:        fmt.Sprintf("attributes.%s", name),
				OldValue:    oldAttr,
				IsBreaking:  true, // Removing attribute is breaking
				Description: fmt.Sprintf("Attribute '%s' removed", name),
			})
		}
	}

	return diffs
}

// DiffBusinessProcesses compares two versions of a Business Process
func (e *DiffEngine) DiffBusinessProcesses(oldBP, newBP BusinessProcess) []Diff {
	var diffs []Diff

	// Compare Transitions
	oldTrans := make(map[string]Transition)
	for _, t := range oldBP.Transitions {
		key := fmt.Sprintf("%s->%s", t.From, t.To)
		oldTrans[key] = t
	}

	newTrans := make(map[string]Transition)
	for _, t := range newBP.Transitions {
		key := fmt.Sprintf("%s->%s", t.From, t.To)
		newTrans[key] = t
	}

	// Added Transitions
	for key, newT := range newTrans {
		if _, exists := oldTrans[key]; !exists {
			diffs = append(diffs, Diff{
				Type:        ChangeTypeTransitionAdded,
				Path:        fmt.Sprintf("transitions[%s]", key),
				NewValue:    newT,
				IsBreaking:  false, // Adding transition is usually safe
				Description: fmt.Sprintf("Transition '%s' added", key),
			})
		}
	}

	// Removed Transitions
	for key, oldT := range oldTrans {
		if _, exists := newTrans[key]; !exists {
			diffs = append(diffs, Diff{
				Type:        ChangeTypeTransitionRemoved,
				Path:        fmt.Sprintf("transitions[%s]", key),
				OldValue:    oldT,
				IsBreaking:  true, // Removing transition breaks existing workflows
				Description: fmt.Sprintf("Transition '%s' removed", key),
			})
		}
	}

	return diffs
}
