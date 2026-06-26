package analytics

import (
	"encoding/json"
	"reflect"

	"github.com/hondyman/semlayer/backend/internal/models"
)

type BODiffService struct{}

func NewBODiffService() *BODiffService {
	return &BODiffService{}
}

// DiffNodes performs a 3-way diff: Postgres (Existing), Iceberg (Snapshot), Incoming (Bundle)
func (s *BODiffService) DiffNodes(postgres, iceberg, incoming *models.CatalogNodeExport) (*models.NodeDiff, error) {
	diff := &models.NodeDiff{
		NodeType: incoming.NodeTypeID,
		NodeName: incoming.NodeName,
		Status:   models.DiffExistsSame, // Default, updated below
		Existing: &models.NodeSnapshot{
			Properties: postgres.Properties,
			Config:     postgres.Config,
		},
		Incoming: &models.NodeSnapshot{
			Properties: incoming.Properties,
			Config:     incoming.Config,
		},
	}

	if iceberg != nil {
		diff.Iceberg = &models.NodeSnapshot{
			Properties: iceberg.Properties,
			Config:     iceberg.Config,
		}
	} else {
		// If no iceberg snapshot, treat as 2-way diff (Iceberg = empty or nil)
		// Or maybe Iceberg equals Postgres if we assume no history?
		// For robustness, let's treat nil/empty as distinct.
	}

	var icebergProps, icebergConfig json.RawMessage
	if iceberg != nil {
		icebergProps = iceberg.Properties
		icebergConfig = iceberg.Config
	}

	// 1. Compare Properties (3-Way)
	propDiff, err := DiffThreeWay(postgres.Properties, icebergProps, incoming.Properties)
	if err != nil {
		return nil, err
	}

	// 2. Compare Config (3-Way)
	configDiff, err := DiffThreeWay(postgres.Config, icebergConfig, incoming.Config)
	if err != nil {
		return nil, err
	}

	if !isThreeWayDiffEmpty(propDiff) || !isThreeWayDiffEmpty(configDiff) {
		diff.Status = models.DiffExistsDifferent
		diff.Diff = &models.NodePropertyDiff{
			Properties: propDiff,
			Config:     configDiff,
		}

		// Detect Conflicts explicitly
		if len(propDiff.Conflicts) > 0 || len(configDiff.Conflicts) > 0 {
			diff.Status = models.DiffConflict
		}
	}

	return diff, nil
}

// DiffEdges - Keeping 2-way for now as Edges in 3-way is complex (Graph Diff)
// For now, simpler implementation: Edges are usually "Replaced" or "Merged".
// We can enhance this later.
func (s *BODiffService) DiffEdges(existingEdges, incomingEdges []models.EdgeExport) []models.EdgeDiff {
	var diffs []models.EdgeDiff
	existingMap := make(map[string]models.EdgeExport)
	for _, e := range existingEdges {
		existingMap[fmtEdgeKey(e)] = e
	}

	for _, inc := range incomingEdges {
		key := fmtEdgeKey(inc)
		if _, exists := existingMap[key]; exists {
			diffs = append(diffs, models.EdgeDiff{
				EdgeType: inc.EdgeType,
				Source:   models.EdgeEndRef{Type: inc.SourceType, Name: inc.SourceName},
				Target:   models.EdgeEndRef{Type: inc.TargetType, Name: inc.TargetName},
				Status:   models.DiffExistsSame,
			})
			delete(existingMap, key)
		} else {
			diffs = append(diffs, models.EdgeDiff{
				EdgeType: inc.EdgeType,
				Source:   models.EdgeEndRef{Type: inc.SourceType, Name: inc.SourceName},
				Target:   models.EdgeEndRef{Type: inc.TargetType, Name: inc.TargetName},
				Status:   models.DiffMissing, // New
			})
		}
	}
	return diffs
}

// --- 3-Way Diff Algorithm ---

func DiffThreeWay(postgresJSON, icebergJSON, incomingJSON json.RawMessage) (models.ThreeWayDiff, error) {
	pg := make(map[string]interface{})
	ic := make(map[string]interface{})
	in := make(map[string]interface{})

	if len(postgresJSON) > 0 {
		_ = json.Unmarshal(postgresJSON, &pg)
	}
	if len(icebergJSON) > 0 {
		_ = json.Unmarshal(icebergJSON, &ic)
	}
	if len(incomingJSON) > 0 {
		_ = json.Unmarshal(incomingJSON, &in)
	}

	diff := models.ThreeWayDiff{
		Added:     make(map[string]interface{}),
		Removed:   make(map[string]interface{}),
		Changed:   make(map[string]models.FieldChange),
		Conflicts: make(map[string]models.ConflictDetail),
	}

	keys := unionKeys(pg, ic, in)

	for _, key := range keys {
		oldVal := ic[key]  // Iceberg
		currVal := pg[key] // Postgres
		newVal := in[key]  // Incoming

		// Check Existence State
		existsOld := oldVal != nil
		existsCurr := currVal != nil
		existsNew := newVal != nil

		// Case 1: Field Added in Incoming (Nil in Old/Curr, Present in New)
		// Note: Also covers if Added in Postgres AND Incoming (Merge/Same?)
		// Let's stick to logic:

		if !existsOld && !existsCurr && existsNew {
			diff.Added[key] = newVal
			continue
		}

		// Case 2: Field Removed in Incoming (Present in Old/Curr, Nil in New)
		if existsOld && existsCurr && !existsNew {
			// Check if Postgres also removed it? No, currVal is present.
			// If Postgres removed it, currVal would be nil.
			diff.Removed[key] = currVal
			continue
		}

		// If all nil, skip
		if !existsOld && !existsCurr && !existsNew {
			continue
		}

		// Case 3: Safe Forward Change (Postgres unchanged from Iceberg, Incoming has new value)
		// curr == old AND new != old
		// BUT: we handle nil/existence above mostly. DeepEqual handles nil too.

		isCurrUnchanged := deepEqualJSON(currVal, oldVal)
		isIncomingChanged := !deepEqualJSON(newVal, oldVal)

		if isCurrUnchanged && isIncomingChanged {
			// Apply New
			// Could be an Add (if old was nil) or Change
			if !existsOld && existsNew {
				diff.Added[key] = newVal
			} else if existsOld && !existsNew {
				diff.Removed[key] = oldVal // Should match Case 2
			} else {
				diff.Changed[key] = models.FieldChange{From: currVal, To: newVal}
			}
			continue
		}

		// Case 4: Local Change Only (Incoming matches Iceberg, Postgres has changed)
		// curr != old AND new == old
		if !isCurrUnchanged && !isIncomingChanged {
			// Keep Postgres, ignore "stale" incoming
			continue
		}

		// Case 5: Conflict
		// curr != old AND new != old AND curr != new
		if !isCurrUnchanged && isIncomingChanged && !deepEqualJSON(currVal, newVal) {
			diff.Conflicts[key] = models.ConflictDetail{
				Postgres: currVal,
				Iceberg:  oldVal,
				Incoming: newVal,
			}
			continue
		}
	}

	return diff, nil
}

// Schema/Helpers

func unionKeys(m1, m2, m3 map[string]interface{}) []string {
	keys := make(map[string]bool)
	for k := range m1 {
		keys[k] = true
	}
	for k := range m2 {
		keys[k] = true
	}
	for k := range m3 {
		keys[k] = true
	}

	list := make([]string, 0, len(keys))
	for k := range keys {
		list = append(list, k)
	}
	return list
}

func deepEqualJSON(a, b interface{}) bool {
	// Simple reflection for now. Can be optimized.
	// user's implementation:
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}

	switch aTyped := a.(type) {
	case map[string]interface{}:
		bTyped, ok := b.(map[string]interface{})
		if !ok {
			return false
		}
		if len(aTyped) != len(bTyped) {
			return false
		}
		return reflect.DeepEqual(aTyped, bTyped)
	case []interface{}:
		bTyped, ok := b.([]interface{})
		if !ok {
			return false
		}
		if len(aTyped) != len(bTyped) {
			return false
		}
		return reflect.DeepEqual(aTyped, bTyped)
	default:
		return reflect.DeepEqual(a, b)
	}
}

func isThreeWayDiffEmpty(d models.ThreeWayDiff) bool {
	return len(d.Added) == 0 && len(d.Removed) == 0 && len(d.Changed) == 0 && len(d.Conflicts) == 0
}

func fmtEdgeKey(e models.EdgeExport) string {
	return e.EdgeType + "|" + e.SourceType + ":" + e.SourceName + "->" + e.TargetType + ":" + e.TargetName
}
