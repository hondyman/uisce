package datapipeline

import (
	"encoding/json"
	"fmt"
)

// legacyReactFlowDef mirrors the shape historically produced by the old
// ReactFlow-based Uisce Flow canvas (formerly handlers.ReactFlowDef in
// internal/handlers/pipelines_handler.go, now retired).
type legacyReactFlowDef struct {
	Nodes []struct {
		ID       string                 `json:"id"`
		Type     string                 `json:"type"`
		Data     map[string]interface{} `json:"data"`
		Position struct {
			X float64 `json:"x"`
			Y float64 `json:"y"`
		} `json:"position"`
	} `json:"nodes"`
	Edges []struct {
		ID     string `json:"id"`
		Source string `json:"source"`
		Target string `json:"target"`
		Label  string `json:"label,omitempty"`
	} `json:"edges"`
}

// ConvertLegacyPipelineJSON converts the legacy ReactFlow {nodes,edges} JSON
// payload (as previously stored in the `pipelines.pipeline_json` column and
// consumed by the retired internal/handlers.PipelineHandler) into a
// datapipeline.PipelineDAG.
//
// Node "type" inference reuses the label-based heuristic that used to live
// in handlers.inferTypeFromLabel: nodes whose label mentions "Validation"
// become "validator" nodes, "Notify"/"Approver" map to a "workflow_caller"
// transform, "External" maps to an "api_caller" transform, and everything
// else defaults to a plain "transform" step tagged with a legacyType hint so
// it round-trips through the DAG engine without silently dropping data.
func ConvertLegacyPipelineJSON(raw json.RawMessage) (PipelineDAG, error) {
	var rf legacyReactFlowDef
	if err := json.Unmarshal(raw, &rf); err != nil {
		return PipelineDAG{}, fmt.Errorf("invalid legacy pipeline JSON: %w", err)
	}
	if len(rf.Nodes) == 0 {
		return PipelineDAG{}, fmt.Errorf("legacy pipeline has no nodes")
	}

	dag := PipelineDAG{}

	for _, n := range rf.Nodes {
		label, _ := n.Data["label"].(string)

		legacyType := ""
		if v, ok := n.Data["filterType"].(string); ok && v != "" {
			legacyType = v
		} else {
			legacyType = inferLegacyTypeFromLabel(label)
		}

		config := map[string]interface{}{}
		if c, ok := n.Data["config"].(map[string]interface{}); ok {
			for k, v := range c {
				config[k] = v
			}
		}
		for k, v := range n.Data {
			if k != "config" && k != "label" {
				config[k] = v
			}
		}
		config["legacyType"] = legacyType

		nodeType := "transform"
		switch legacyType {
		case "validate":
			nodeType = "validator"
		case "integrate":
			nodeType = "transform"
			if _, ok := config["transformType"]; !ok {
				config["transformType"] = "api_caller"
			}
		case "notify", "approve":
			nodeType = "transform"
			if _, ok := config["transformType"]; !ok {
				config["transformType"] = "workflow_caller"
			}
		}
		// Explicit legacy node type wins over label inference when present.
		switch n.Type {
		case "source", "reader":
			nodeType = "source"
		case "loader", "writer", "sink":
			nodeType = "loader"
		}

		pn := PipelineNode{
			ID:      n.ID,
			Type:    nodeType,
			SubType: legacyType,
			Label:   label,
			Config:  config,
		}
		pn.Position.X = n.Position.X
		pn.Position.Y = n.Position.Y
		dag.Nodes = append(dag.Nodes, pn)
	}

	for _, e := range rf.Edges {
		id := e.ID
		if id == "" {
			id = fmt.Sprintf("%s-%s", e.Source, e.Target)
		}
		dag.Edges = append(dag.Edges, PipelineEdge{
			ID:     id,
			Source: e.Source,
			Target: e.Target,
			Label:  e.Label,
		})
	}

	return dag, nil
}

// inferLegacyTypeFromLabel reproduces the heuristic previously in
// handlers.inferTypeFromLabel for legacy records that never had an explicit
// filterType/legacyType stored.
func inferLegacyTypeFromLabel(label string) string {
	switch {
	case containsFold(label, "Validation"):
		return "validate"
	case containsFold(label, "Approver"):
		return "approve"
	case containsFold(label, "Notify"):
		return "notify"
	case containsFold(label, "External"):
		return "integrate"
	default:
		return "generic"
	}
}

func containsFold(s, substr string) bool {
	if substr == "" {
		return true
	}
	sl, subl := len(s), len(substr)
	if subl > sl {
		return false
	}
	for i := 0; i <= sl-subl; i++ {
		if equalFold(s[i:i+subl], substr) {
			return true
		}
	}
	return false
}

func equalFold(a, b string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := 0; i < len(a); i++ {
		ca, cb := a[i], b[i]
		if 'A' <= ca && ca <= 'Z' {
			ca += 'a' - 'A'
		}
		if 'A' <= cb && cb <= 'Z' {
			cb += 'a' - 'A'
		}
		if ca != cb {
			return false
		}
	}
	return true
}
