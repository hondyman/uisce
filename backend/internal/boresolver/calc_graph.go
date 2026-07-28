package boresolver

import (
	"fmt"
)

// CalcNode represents a semantic term in the calculation execution plan
type CalcNode struct {
	TermKey        string
	IsBaseField    bool
	PhysicalColumn string // Populated if IsBaseField = true
	Formula        string // e.g., "${gross_return} - ${management_fee}"
	Dependencies   []string
	Depth          int // Execution layer index
}

// CalcGraph manages the topological resolution of derived metrics
type CalcGraph struct {
	Nodes map[string]*CalcNode
}

func NewCalcGraph() *CalcGraph {
	return &CalcGraph{
		Nodes: make(map[string]*CalcNode),
	}
}

func (g *CalcGraph) AddNode(node *CalcNode) {
	g.Nodes[node.TermKey] = node
}

// Resolve Execution Layers via Topological Sort
func (g *CalcGraph) ResolveExecutionLayers() ([][]*CalcNode, error) {
	visited := make(map[string]bool)
	visiting := make(map[string]bool)
	var sorted []*CalcNode

	var visit func(nodeKey string) error
	visit = func(nodeKey string) error {
		if visiting[nodeKey] {
			return fmt.Errorf("FATAL GOVERNANCE ERROR: Circular dependency detected involving %s", nodeKey)
		}
		if visited[nodeKey] {
			return nil
		}

		visiting[nodeKey] = true
		node, exists := g.Nodes[nodeKey]
		if !exists {
			return fmt.Errorf("missing dependency in graph: %s", nodeKey)
		}

		for _, depKey := range node.Dependencies {
			if err := visit(depKey); err != nil {
				return err
			}
		}

		visiting[nodeKey] = false
		visited[nodeKey] = true
		sorted = append(sorted, node)
		return nil
	}

	// Visit all nodes
	for key := range g.Nodes {
		if !visited[key] {
			if err := visit(key); err != nil {
				return nil, err
			}
		}
	}

	// Group by depth
	return g.groupNodesByDepth(sorted), nil
}

// Calculates the execution depth layer for CTE generation
func (g *CalcGraph) groupNodesByDepth(sorted []*CalcNode) [][]*CalcNode {
	for _, node := range sorted {
		if node.IsBaseField {
			node.Depth = 0
		} else {
			maxDepDepth := -1
			for _, depKey := range node.Dependencies {
				if depNode, exists := g.Nodes[depKey]; exists {
					if depNode.Depth > maxDepDepth {
						maxDepDepth = depNode.Depth
					}
				}
			}
			node.Depth = maxDepDepth + 1
		}
	}

	var maxDepth int
	for _, node := range sorted {
		if node.Depth > maxDepth {
			maxDepth = node.Depth
		}
	}

	layers := make([][]*CalcNode, maxDepth+1)
	for _, node := range sorted {
		layers[node.Depth] = append(layers[node.Depth], node)
	}

	return layers
}
