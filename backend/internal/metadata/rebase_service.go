package metadata

import (
	"fmt"
)

// Conflict represents a merge conflict requiring manual resolution
type Conflict struct {
	Diff        Diff   `json:"diff"`
	Reason      string `json:"reason"`
	TenantID    string `json:"tenant_id"`
	OverlayPath string `json:"overlay_path"`
}

// RebaseResult contains the outcome of a rebase operation
type RebaseResult struct {
	RebasedBO *BusinessObject `json:"rebased_bo,omitempty"`
	Conflicts []Conflict      `json:"conflicts"`
	Success   bool            `json:"success"`
}

// RebaseService handles merging core updates into tenant overlays
type RebaseService struct {
	diffEngine *DiffEngine
}

func NewRebaseService() *RebaseService {
	return &RebaseService{
		diffEngine: NewDiffEngine(),
	}
}

// RebaseBusinessObject merges new core changes into a tenant overlay
func (s *RebaseService) RebaseBusinessObject(oldCore, newCore, tenantOverlay BusinessObject) RebaseResult {
	// 1. Calculate Diff between Old Core and New Core
	diffs := s.diffEngine.DiffBusinessObjects(oldCore, newCore)

	rebasedOverlay := tenantOverlay
	var conflicts []Conflict

	// 2. Apply Diffs to Overlay
	for _, diff := range diffs {
		switch diff.Type {
		case ChangeTypeAttributeAdded:
			// Auto-merge: Add new attribute to overlay
			newAttr := diff.NewValue.(BOAttribute)
			
			// Check if tenant already defined an attribute with this name
			exists := false
			for _, a := range rebasedOverlay.Attributes {
				if a.Name == newAttr.Name {
					exists = true
					break
				}
			}

			if !exists {
				rebasedOverlay.Attributes = append(rebasedOverlay.Attributes, newAttr)
				fmt.Printf("Auto-merged attribute '%s' into tenant %s overlay\n", newAttr.Name, tenantOverlay.Meta.TenantID)
			} else {
				// Tenant already has this attribute - potential conflict if types differ
				// For now, we assume tenant definition wins (Overlay Precedence)
				fmt.Printf("Skipped attribute '%s' (Tenant override exists)\n", newAttr.Name)
			}

		case ChangeTypeAttributeRemoved:
			// Check if tenant uses this attribute
			// In a real system, we'd check usage in Views, Formulas, etc.
			// Here we just flag it as a conflict if it was in the overlay (inherited)
			
			// If the attribute was purely inherited from core, we remove it.
			// If the tenant had overridden it, we keep it but flag a warning.
			
			// Simplified logic: Flag conflict if it's a breaking change
			conflicts = append(conflicts, Conflict{
				Diff:        diff,
				Reason:      "Core attribute removed. Verify tenant usage.",
				TenantID:    tenantOverlay.Meta.TenantID,
				OverlayPath: diff.Path,
			})

		case ChangeTypeAttributeChanged:
			// If tenant overrides this attribute, we ignore core change (Tenant Wins)
			// If tenant inherits, we apply the change
			
			// attrName := diff.Path // simplified parsing
			// Check if tenant overrides
			// ... (logic to check override status)
			
			// For safety, flag breaking changes as conflicts
			if diff.IsBreaking {
				conflicts = append(conflicts, Conflict{
					Diff:        diff,
					Reason:      "Breaking change in core attribute. Verify compatibility.",
					TenantID:    tenantOverlay.Meta.TenantID,
					OverlayPath: diff.Path,
				})
			} else {
				// Apply safe change (e.g. description update)
				// ...
			}
		}
	}

	return RebaseResult{
		RebasedBO: &rebasedOverlay,
		Conflicts: conflicts,
		Success:   len(conflicts) == 0,
	}
}
