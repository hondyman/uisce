package datapipeline

import "fmt"

// topologicalOrder returns nodes in topological order following the
// edges in `dag`. Nodes with no incoming edges appear first, in their
// JSON-array order (stable). Nodes with edges appear in dependency
// order.
//
// Returns an error if the DAG contains:
//   - a cycle,
//   - an edge whose source or target is not in dag.Nodes,
//   - two nodes sharing the same ID.
//
// Orphan nodes (no edges either way) appear in the order they were
// declared in dag.Nodes — Kahn's algorithm is stable on the initial
// queue because we seed it from dag.Nodes in array order.
//
// Phase 2 acceptance gate: TestEngine_DiamondDAG_ExecutesInEdgeOrder
// (in engine_test.go) exercises a→{b,c}→d and asserts the visited
// order respects the edges. The diamond's exact middle order (b then
// c, or c then b) depends on Kahn's queue but both are correct.
func topologicalOrder(dag PipelineDAG) ([]PipelineNode, error) {
	if len(dag.Nodes) == 0 {
		return nil, nil
	}

	inDeg := make(map[string]int, len(dag.Nodes))
	out := make(map[string][]string, len(dag.Nodes))
	id2node := make(map[string]PipelineNode, len(dag.Nodes))

	for _, n := range dag.Nodes {
		if _, exists := id2node[n.ID]; exists {
			return nil, fmt.Errorf("pipeline DAG has duplicate node id %q", n.ID)
		}
		inDeg[n.ID] = 0
		out[n.ID] = nil
		id2node[n.ID] = n
	}

	for _, e := range dag.Edges {
		src, dst := e.Source, e.Target
		if _, ok := id2node[src]; !ok {
			return nil, fmt.Errorf("edge %s references unknown source node %q", e.ID, src)
		}
		if _, ok := id2node[dst]; !ok {
			return nil, fmt.Errorf("edge %s references unknown target node %q", e.ID, dst)
		}
		out[src] = append(out[src], dst)
		inDeg[dst]++
	}

	// Seed the queue from the original node order so the result is
	// deterministic for orphan subgraphs.
	queue := make([]string, 0, len(dag.Nodes))
	for _, n := range dag.Nodes {
		if inDeg[n.ID] == 0 {
			queue = append(queue, n.ID)
		}
	}

	ordered := make([]PipelineNode, 0, len(dag.Nodes))
	for len(queue) > 0 {
		id := queue[0]
		queue = queue[1:]
		ordered = append(ordered, id2node[id])
		for _, succ := range out[id] {
			inDeg[succ]--
			if inDeg[succ] == 0 {
				queue = append(queue, succ)
			}
		}
	}

	if len(ordered) != len(dag.Nodes) {
		return nil, fmt.Errorf("pipeline DAG has a cycle: ordered %d of %d nodes",
			len(ordered), len(dag.Nodes))
	}
	return ordered, nil
}
