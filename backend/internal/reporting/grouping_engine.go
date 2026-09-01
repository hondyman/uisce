package reporting

import (
	"context"
	"fmt"
	"sort"

	"github.com/google/uuid"
)

type HierarchicalGroupingEngine struct{}

func NewHierarchicalGroupingEngine() *HierarchicalGroupingEngine {
	return &HierarchicalGroupingEngine{}
}

// BuildHierarchy partitions flat row batches into a recursive N-level tree and calculates rollups
func (e *HierarchicalGroupingEngine) BuildHierarchy(
	ctx context.Context,
	tenantID uuid.UUID,
	spec GroupHierarchySpec,
	records []map[string]interface{},
) (*HierarchicalDataset, error) {
	if tenantID == uuid.Nil {
		return nil, fmt.Errorf("Rule 7 violation: tenant_id cannot be nil")
	}

	if len(spec.Levels) == 0 {
		grandTotals := e.computeAggregations(records, e.collectAllRollups(spec.Levels))
		return &HierarchicalDataset{
			TenantID:     tenantID,
			RootNodes:    nil,
			GrandTotals:  grandTotals,
			TotalRecords: len(records),
		}, nil
	}

	// 1. Recursively build group buckets
	rootNodes := e.partitionGroup(records, spec.Levels, 0, nil, spec.DefaultCollapsed)

	// 2. Derive grand totals across all leaf records
	allRollups := e.collectAllRollups(spec.Levels)
	grandTotals := e.computeAggregations(records, allRollups)

	return &HierarchicalDataset{
		TenantID:     tenantID,
		RootNodes:    rootNodes,
		GrandTotals:  grandTotals,
		TotalRecords: len(records),
	}, nil
}

func (e *HierarchicalGroupingEngine) partitionGroup(
	records []map[string]interface{},
	levels []GroupLevelDef,
	currentLevelIdx int,
	parentNodeID *string,
	defaultCollapsed bool,
) []*GroupNode {
	if currentLevelIdx >= len(levels) || len(records) == 0 {
		return nil
	}

	currentLevel := levels[currentLevelIdx]
	groupedBuckets := make(map[string][]map[string]interface{})
	orderedKeys := make([]string, 0)

	for _, rec := range records {
		valRaw := rec[currentLevel.GroupFieldKey]
		valStr := "Unassigned"
		if valRaw != nil {
			valStr = fmt.Sprintf("%v", valRaw)
		}

		if _, exists := groupedBuckets[valStr]; !exists {
			groupedBuckets[valStr] = make([]map[string]interface{}, 0)
			orderedKeys = append(orderedKeys, valStr)
		}
		groupedBuckets[valStr] = append(groupedBuckets[valStr], rec)
	}

	sort.Strings(orderedKeys)

	nodes := make([]*GroupNode, 0, len(orderedKeys))
	for _, key := range orderedKeys {
		bucketRecords := groupedBuckets[key]
		nodeID := fmt.Sprintf("grp_%d_%s", currentLevelIdx, key)
		if parentNodeID != nil {
			nodeID = fmt.Sprintf("%s/%s", *parentNodeID, key)
		}

		aggregations := e.computeAggregations(bucketRecords, currentLevel.Rollups)

		node := &GroupNode{
			NodeID:       nodeID,
			LevelIndex:   currentLevelIdx,
			GroupKey:     currentLevel.GroupFieldKey,
			GroupValue:   key,
			ParentNodeID: parentNodeID,
			Depth:        currentLevelIdx,
			ItemCount:    len(bucketRecords),
			Aggregations: aggregations,
			IsCollapsed:  defaultCollapsed,
		}

		if currentLevelIdx+1 < len(levels) {
			node.Children = e.partitionGroup(bucketRecords, levels, currentLevelIdx+1, &nodeID, defaultCollapsed)
		} else {
			node.LeafRecords = bucketRecords
		}

		nodes = append(nodes, node)
	}

	return nodes
}

func (e *HierarchicalGroupingEngine) computeAggregations(
	records []map[string]interface{},
	rollups []RollupFieldDef,
) map[string]float64 {
	results := make(map[string]float64)

	for _, r := range rollups {
		targetKey := r.ResultKey
		if targetKey == "" {
			targetKey = fmt.Sprintf("%s_%s", r.Function, r.FieldKey)
		}

		switch r.Function {
		case AggSum:
			var sum float64
			for _, rec := range records {
				sum += toFloat(rec[r.FieldKey])
			}
			results[targetKey] = sum

		case AggCount:
			results[targetKey] = float64(len(records))

		case AggMin:
			if len(records) == 0 {
				results[targetKey] = 0
				continue
			}
			minVal := toFloat(records[0][r.FieldKey])
			for _, rec := range records[1:] {
				v := toFloat(rec[r.FieldKey])
				if v < minVal {
					minVal = v
				}
			}
			results[targetKey] = minVal

		case AggMax:
			if len(records) == 0 {
				results[targetKey] = 0
				continue
			}
			maxVal := toFloat(records[0][r.FieldKey])
			for _, rec := range records[1:] {
				v := toFloat(rec[r.FieldKey])
				if v > maxVal {
					maxVal = v
				}
			}
			results[targetKey] = maxVal

		case AggAvg:
			if len(records) == 0 {
				results[targetKey] = 0
				continue
			}
			var sum float64
			for _, rec := range records {
				sum += toFloat(rec[r.FieldKey])
			}
			results[targetKey] = sum / float64(len(records))

		case AggWeightedAvg:
			if len(records) == 0 || r.WeightFieldKey == "" {
				results[targetKey] = 0
				continue
			}
			var weightedSum, totalWeight float64
			for _, rec := range records {
				val := toFloat(rec[r.FieldKey])
				weight := toFloat(rec[r.WeightFieldKey])
				weightedSum += val * weight
				totalWeight += weight
			}
			if totalWeight > 0 {
				results[targetKey] = weightedSum / totalWeight
			} else {
				results[targetKey] = 0
			}
		}
	}

	return results
}

func (e *HierarchicalGroupingEngine) collectAllRollups(levels []GroupLevelDef) []RollupFieldDef {
	seen := make(map[string]bool)
	all := make([]RollupFieldDef, 0)

	for _, lvl := range levels {
		for _, r := range lvl.Rollups {
			key := fmt.Sprintf("%s_%s_%s", r.Function, r.FieldKey, r.WeightFieldKey)
			if !seen[key] {
				seen[key] = true
				all = append(all, r)
			}
		}
	}
	return all
}

func toFloat(val interface{}) float64 {
	if val == nil {
		return 0
	}
	switch v := val.(type) {
	case float64:
		return v
	case float32:
		return float64(v)
	case int:
		return float64(v)
	case int64:
		return float64(v)
	default:
		return 0
	}
}
