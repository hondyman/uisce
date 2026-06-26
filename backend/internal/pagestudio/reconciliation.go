package pagestudio

import (
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
)

// ReconciliationService handles the logic for tenant upgrades
type ReconciliationService struct {
	// dependencies like repo
}

// ComputeUpgradeImpact compares old and new core versions with a tenant overlay
func (s *ReconciliationService) ComputeUpgradeImpact(
	oldCore, newCore *CorePage,
	overlay *PageOverlay,
) (*UpgradeImpact, error) {
	impact := &UpgradeImpact{
		ID:             uuid.New(),
		CorePageID:     newCore.ID,
		CoreOldVersion: oldCore.Version,
		CoreNewVersion: newCore.Version,
		TenantID:       overlay.TenantID,
		OverlayPageID:  overlay.ID,
		Status:         UpgradeStatusPending,
	}

	conflicts := []ConflictItem{}
	inherited := []ChangeItem{}

	// 1. Analyze Components
	var oldComp, newComp map[string]interface{}
	var ovComp map[string]interface{}

	json.Unmarshal(oldCore.Components, &oldComp)
	json.Unmarshal(newCore.Components, &newComp)

	var overlayData struct {
		Components map[string]struct {
			Props map[string]interface{} `json:"props"`
		} `json:"components"`
	}
	json.Unmarshal(overlay.Overrides, &overlayData)
	ovComp = make(map[string]interface{})
	for k, v := range overlayData.Components {
		ovComp[k] = v.Props
	}

	for id, nc := range newComp {
		oc, ok := oldComp[id]
		if !ok {
			impact.NewCoreComponents = append(impact.NewCoreComponents, id)
			continue
		}

		// Check if core changed this component
		if !jsonEqual(oc, nc) {
			// Core changed it. Does tenant override it?
			if tc, overridden := ovComp[id]; overridden {
				conflicts = append(conflicts, ConflictItem{
					Type:           "component",
					ComponentID:    &id,
					CoreBefore:     oc,
					CoreAfter:      nc,
					TenantOverride: tc,
				})
			} else {
				inherited = append(inherited, ChangeItem{
					Type:        "component",
					ComponentID: &id,
					Before:      oc,
					After:       nc,
				})
			}
		}
	}

	for id := range oldComp {
		if _, ok := newComp[id]; !ok {
			impact.RemovedCoreComponents = append(impact.RemovedCoreComponents, id)
		}
	}

	confBytes, _ := json.Marshal(conflicts)
	inhBytes, _ := json.Marshal(inherited)
	impact.Conflicts = confBytes
	impact.InheritedChanges = inhBytes

	impact.Summary = fmt.Sprintf("Core update %d -> %d: %d conflicts, %d inherited changes.",
		oldCore.Version, newCore.Version, len(conflicts), len(inherited))

	return impact, nil
}

func jsonEqual(a, b interface{}) bool {
	aj, _ := json.Marshal(a)
	bj, _ := json.Marshal(b)
	return string(aj) == string(bj)
}
