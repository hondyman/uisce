package reporting

import (
	"github.com/google/uuid"
)

type AggregationFunction string

const (
	AggSum         AggregationFunction = "SUM"
	AggAvg         AggregationFunction = "AVG"
	AggCount       AggregationFunction = "COUNT"
	AggMin         AggregationFunction = "MIN"
	AggMax         AggregationFunction = "MAX"
	AggWeightedAvg AggregationFunction = "WEIGHTED_AVG"
)

type RollupFieldDef struct {
	FieldKey       string              `json:"fieldKey"`
	ResultKey      string              `json:"resultKey"`
	Function       AggregationFunction `json:"function"`
	WeightFieldKey string              `json:"weightFieldKey,omitempty"` // For WEIGHTED_AVG
	FormatMask     string              `json:"formatMask"`
}

type GroupLevelDef struct {
	LevelIndex    int              `json:"levelIndex"`
	GroupFieldKey string           `json:"groupFieldKey"` // e.g., "asset_class"
	DisplayName   string           `json:"displayName"`
	PageBreak     bool             `json:"pageBreak"`
	KeepTogether  bool             `json:"keepTogether"`
	ShowHeader    bool             `json:"showHeader"`
	ShowFooter    bool             `json:"showFooter"`
	Rollups       []RollupFieldDef `json:"rollups"`
}

type GroupHierarchySpec struct {
	Levels           []GroupLevelDef `json:"levels"`
	SortAscending    bool            `json:"sortAscending"`
	DefaultCollapsed bool            `json:"defaultCollapsed"`
}

type GroupNode struct {
	NodeID       string                   `json:"nodeId"`
	LevelIndex   int                      `json:"levelIndex"`
	GroupKey     string                   `json:"groupKey"`   // Field name (e.g., "asset_class")
	GroupValue   string                   `json:"groupValue"` // Resolved value (e.g., "Equities")
	ParentNodeID *string                  `json:"parentNodeId,omitempty"`
	Depth        int                      `json:"depth"`
	ItemCount    int                      `json:"itemCount"`
	Aggregations map[string]float64       `json:"aggregations"`
	Children     []*GroupNode             `json:"children,omitempty"`
	LeafRecords  []map[string]interface{} `json:"leafRecords,omitempty"`
	IsCollapsed  bool                     `json:"isCollapsed"`
}

type HierarchicalDataset struct {
	TenantID     uuid.UUID          `json:"tenantId"`
	RootNodes    []*GroupNode       `json:"rootNodes"`
	GrandTotals  map[string]float64 `json:"grandTotals"`
	TotalRecords int                `json:"totalRecords"`
}
