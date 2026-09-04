package datapipeline

import "testing"

func TestTopologicalOrder_Linear(t *testing.T) {
	dag := PipelineDAG{
		Nodes: []PipelineNode{
			{ID: "a", Label: "A", Type: "source"},
			{ID: "b", Label: "B", Type: "transform"},
			{ID: "c", Label: "C", Type: "loader"},
		},
		Edges: []PipelineEdge{
			{ID: "e1", Source: "a", Target: "b"},
			{ID: "e2", Source: "b", Target: "c"},
		},
	}
	got, err := topologicalOrder(dag)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 3 {
		t.Fatalf("expected 3 nodes, got %d", len(got))
	}
	if got[0].ID != "a" || got[1].ID != "b" || got[2].ID != "c" {
		t.Errorf("expected [a b c], got [%s %s %s]", got[0].ID, got[1].ID, got[2].ID)
	}
}

func TestTopologicalOrder_Diamond(t *testing.T) {
	dag := PipelineDAG{
		Nodes: []PipelineNode{
			{ID: "a", Label: "A", Type: "source"},
			{ID: "b", Label: "B", Type: "transform"},
			{ID: "c", Label: "C", Type: "transform"},
			{ID: "d", Label: "D", Type: "loader"},
		},
		Edges: []PipelineEdge{
			{ID: "e1", Source: "a", Target: "b"},
			{ID: "e2", Source: "a", Target: "c"},
			{ID: "e3", Source: "b", Target: "d"},
			{ID: "e4", Source: "c", Target: "d"},
		},
	}
	got, err := topologicalOrder(dag)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 4 {
		t.Fatalf("expected 4 nodes, got %d", len(got))
	}
	// a must be first
	if got[0].ID != "a" {
		t.Errorf("expected a first, got %s", got[0].ID)
	}
	// d must be last
	if got[3].ID != "d" {
		t.Errorf("expected d last, got %s", got[3].ID)
	}
	// b and c must both come after a and before d
	bIdx := -1
	cIdx := -1
	for i, n := range got {
		if n.ID == "b" {
			bIdx = i
		}
		if n.ID == "c" {
			cIdx = i
		}
	}
	if bIdx <= 0 || cIdx <= 0 {
		t.Errorf("b and c must both have index > 0; got b=%d c=%d", bIdx, cIdx)
	}
	if bIdx >= 3 || cIdx >= 3 {
		t.Errorf("b and c must both have index < 3; got b=%d c=%d", bIdx, cIdx)
	}
}

func TestTopologicalOrder_CycleErrors(t *testing.T) {
	dag := PipelineDAG{
		Nodes: []PipelineNode{
			{ID: "a", Label: "A", Type: "source"},
			{ID: "b", Label: "B", Type: "transform"},
		},
		Edges: []PipelineEdge{
			{ID: "e1", Source: "a", Target: "b"},
			{ID: "e2", Source: "b", Target: "a"},
		},
	}
	_, err := topologicalOrder(dag)
	if err == nil {
		t.Fatal("expected error for cycle, got nil")
	}
}

func TestTopologicalOrder_OrphanAllowed(t *testing.T) {
	dag := PipelineDAG{
		Nodes: []PipelineNode{
			{ID: "a", Label: "A", Type: "source"},
			{ID: "b", Label: "B", Type: "transform"},
			{ID: "c", Label: "C", Type: "loader"},
		},
		Edges: []PipelineEdge{},
	}
	got, err := topologicalOrder(dag)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 3 {
		t.Fatalf("expected 3 nodes, got %d", len(got))
	}
	if got[0].ID != "a" || got[1].ID != "b" || got[2].ID != "c" {
		t.Errorf("expected [a b c], got [%s %s %s]", got[0].ID, got[1].ID, got[2].ID)
	}
}

func TestTopologicalOrder_DanglingEdgeErrors(t *testing.T) {
	dag := PipelineDAG{
		Nodes: []PipelineNode{
			{ID: "a", Label: "A", Type: "source"},
			{ID: "b", Label: "B", Type: "transform"},
		},
		Edges: []PipelineEdge{
			{ID: "e1", Source: "a", Target: "nonexistent"},
		},
	}
	_, err := topologicalOrder(dag)
	if err == nil {
		t.Fatal("expected error for dangling edge, got nil")
	}
}

func TestTopologicalOrder_DuplicateNodeID(t *testing.T) {
	dag := PipelineDAG{
		Nodes: []PipelineNode{
			{ID: "a", Label: "A", Type: "source"},
			{ID: "a", Label: "A2", Type: "transform"},
		},
		Edges: []PipelineEdge{},
	}
	_, err := topologicalOrder(dag)
	if err == nil {
		t.Fatal("expected error for duplicate node ID, got nil")
	}
}
