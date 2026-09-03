package api

import (
	"testing"

	"github.com/hondyman/uisce/backend/internal/graphview"
)

func TestApplyTypePolicy_ExcludesConfiguredNodeType(t *testing.T) {
	graph := &graphview.ResponseGraph{
		Nodes: []graphview.ResponseNode{
			{ID: "n1", Type: "table"},
			{ID: "n2", Type: "internal_audit_flag"},
		},
		Edges: []graphview.ResponseEdge{
			{ID: "n1->n2", Source: "n1", Target: "n2", Type: "belongs_to"},
		},
	}

	config := map[string]interface{}{
		"typePolicy": map[string]interface{}{
			"defaultInclude": true,
			"nodeTypes": map[string]interface{}{
				"internal_audit_flag": "exclude",
			},
		},
	}

	applyTypePolicy(graph, config)

	if len(graph.Nodes) != 1 || graph.Nodes[0].ID != "n1" {
		t.Fatalf("expected only n1 to survive exclusion, got %+v", graph.Nodes)
	}
	if len(graph.Edges) != 0 {
		t.Fatalf("expected edge referencing excluded node to be dropped, got %+v", graph.Edges)
	}
}

func TestApplyTypePolicy_DefaultIncludeFalseRequiresExplicitInclude(t *testing.T) {
	graph := &graphview.ResponseGraph{
		Nodes: []graphview.ResponseNode{
			{ID: "n1", Type: "table"},
			{ID: "n2", Type: "column"},
		},
	}
	config := map[string]interface{}{
		"typePolicy": map[string]interface{}{
			"defaultInclude": false,
			"nodeTypes": map[string]interface{}{
				"table": "include",
			},
		},
	}

	applyTypePolicy(graph, config)

	if len(graph.Nodes) != 1 || graph.Nodes[0].ID != "n1" {
		t.Fatalf("expected only explicitly-included type to survive, got %+v", graph.Nodes)
	}
}

func TestApplyGrouping_CollapsesLargeFanoutIntoCluster(t *testing.T) {
	graph := &graphview.ResponseGraph{
		Nodes: []graphview.ResponseNode{
			{ID: "table1", Type: "table"},
		},
		Edges: []graphview.ResponseEdge{},
	}
	for i := 0; i < 20; i++ {
		colID := "col" + string(rune('a'+i))
		graph.Nodes = append(graph.Nodes, graphview.ResponseNode{ID: colID, Type: "column"})
		graph.Edges = append(graph.Edges, graphview.ResponseEdge{
			ID: "table1->" + colID, Source: "table1", Target: colID, Type: "belongs_to",
		})
	}

	config := map[string]interface{}{
		"grouping": []interface{}{
			map[string]interface{}{
				"childNodeType":     "column",
				"parentRelation":    "belongs_to",
				"clusterLabel":      "Columns",
				"defaultCollapsed":  true,
				"collapseThreshold": 15,
			},
		},
	}

	applyGrouping(graph, config, nil)

	var clusterCount, columnCount int
	for _, n := range graph.Nodes {
		if n.IsCluster {
			clusterCount++
			if len(n.MemberIDs) != 20 {
				t.Fatalf("expected cluster to carry all 20 members, got %d", len(n.MemberIDs))
			}
		}
		if n.Type == "column" {
			columnCount++
		}
	}
	if clusterCount != 1 {
		t.Fatalf("expected exactly one cluster node, got %d", clusterCount)
	}
	if columnCount != 0 {
		t.Fatalf("expected individual column nodes to be folded away, got %d remaining", columnCount)
	}

	// Payload must stay bounded regardless of fan-out size: 1 table + 1 cluster node.
	if len(graph.Nodes) != 2 {
		t.Fatalf("expected payload to bound at 2 nodes (table + cluster), got %d", len(graph.Nodes))
	}
}

func TestApplyGrouping_ExpandedClusterStaysUngrouped(t *testing.T) {
	graph := &graphview.ResponseGraph{
		Nodes: []graphview.ResponseNode{
			{ID: "table1", Type: "table"},
		},
	}
	for i := 0; i < 20; i++ {
		colID := "col" + string(rune('a'+i))
		graph.Nodes = append(graph.Nodes, graphview.ResponseNode{ID: colID, Type: "column"})
		graph.Edges = append(graph.Edges, graphview.ResponseEdge{
			ID: "table1->" + colID, Source: "table1", Target: colID, Type: "belongs_to",
		})
	}

	config := map[string]interface{}{
		"grouping": []interface{}{
			map[string]interface{}{
				"childNodeType":     "column",
				"parentRelation":    "belongs_to",
				"collapseThreshold": 15,
			},
		},
	}

	applyGrouping(graph, config, map[string]bool{"cluster:table1:column": true})

	var columnCount int
	for _, n := range graph.Nodes {
		if n.IsCluster {
			t.Fatalf("expected explicitly expanded cluster to stay ungrouped, got a cluster node")
		}
		if n.Type == "column" {
			columnCount++
		}
	}
	if columnCount != 20 {
		t.Fatalf("expected all 20 columns to render individually, got %d", columnCount)
	}
}

func TestApplyGrouping_LeavesSmallFanoutUngrouped(t *testing.T) {
	graph := &graphview.ResponseGraph{
		Nodes: []graphview.ResponseNode{
			{ID: "table1", Type: "table"},
			{ID: "col1", Type: "column"},
			{ID: "col2", Type: "column"},
		},
		Edges: []graphview.ResponseEdge{
			{ID: "table1->col1", Source: "table1", Target: "col1", Type: "belongs_to"},
			{ID: "table1->col2", Source: "table1", Target: "col2", Type: "belongs_to"},
		},
	}
	config := map[string]interface{}{
		"grouping": []interface{}{
			map[string]interface{}{
				"childNodeType":     "column",
				"parentRelation":    "belongs_to",
				"collapseThreshold": 15,
			},
		},
	}

	applyGrouping(graph, config, nil)

	if len(graph.Nodes) != 3 {
		t.Fatalf("expected small fan-out to stay ungrouped, got %d nodes", len(graph.Nodes))
	}
}
