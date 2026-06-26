package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/hondyman/semlayer/backend/pkg/workflows"

	"github.com/google/uuid"
	"go.temporal.io/sdk/client"
)

// Local structs for parsing the high-level JSON
type BPDefinition struct {
	ID      string   `json:"id"`
	Name    string   `json:"name"`
	Version string   `json:"version"`
	Steps   []BPStep `json:"steps"`
}

type BPStep struct {
	ID         string                 `json:"id"`
	Name       string                 `json:"name"`
	Type       string                 `json:"type"`
	Category   string                 `json:"category"`
	Config     map[string]interface{} `json:"config"`
	NextStepID string                 `json:"next_step_id"`
}

func main() {
	workflowPath := flag.String("workflow", "", "Path to workflow JSON definition")
	flag.Parse()

	if *workflowPath == "" {
		log.Fatal("Please provide -workflow <path>")
	}

	// 1. Read Workflow Definition
	data, err := os.ReadFile(*workflowPath)
	if err != nil {
		log.Fatalf("Failed to read workflow file: %v", err)
	}

	var bpDef BPDefinition
	if err := json.Unmarshal(data, &bpDef); err != nil {
		log.Fatalf("Failed to parse workflow JSON: %v", err)
	}

	// Unmarshal config to ensure it's a map
	// The JSON parser does this, but we carefully map it to WorkflowNode

	// 2. Convert to Interpreter DSL (WorkflowDefinition)
	dsl := workflows.WorkflowDefinition{
		Name: bpDef.Name,
		GlobalState: map[string]interface{}{
			"initiator":  "admin@example.com",
			"tenant_id":  "tenant-123",
			"bp_id":      bpDef.ID,
			"bp_version": bpDef.Version,
		},
		Nodes:       make(map[string]workflows.WorkflowNode),
		StartNodeID: bpDef.Steps[0].ID, // Assume first step is start for now
	}

	for _, step := range bpDef.Steps {
		node := workflows.WorkflowNode{
			ID:     step.ID,
			Type:   step.Type, // Map Step Type directly to Node Type (interpreter handles strict types)
			Name:   step.Name,
			Config: step.Config,
		}

		// If NextStepID is present and not "end", set it
		if step.NextStepID != "" && step.NextStepID != "end" {
			nextID := step.NextStepID
			node.NextNodeID = &nextID
		} else if step.NextStepID == "end" {
			// Ensure there's an end node or just let it finish
			// For interpreter, if NextNodeID is nil, it stops.
		}

		dsl.Nodes[step.ID] = node
	}

	// Add explicit END node if referenced or implied?
	// The interpreter stops when NextNodeID is nil.
	// Our model_change_request.json has an "end" step with type "Completion".
	// We should make sure that step is processing correctly.

	// 3. Connect to Temporal
	c, err := client.Dial(client.Options{
		HostPort:  os.Getenv("TEMPORAL_ADDRESS"),
		Namespace: os.Getenv("TEMPORAL_NAMESPACE"),
	})
	if err != nil {
		// Fallback to defaults
		c, err = client.Dial(client.Options{})
		if err != nil {
			log.Fatalf("Unable to create client: %v", err)
		}
	}
	defer c.Close()

	// 4. Start Workflow
	// Use formatted ID for uniqueness
	workflowID := fmt.Sprintf("bp-%s-%s", bpDef.ID, uuid.New().String()[:8])
	options := client.StartWorkflowOptions{
		ID:        workflowID,
		TaskQueue: workflows.BPTaskQueue,
	}

	// InterpreterWorkflow signature: func(ctx, dsl)
	we, err := c.ExecuteWorkflow(context.Background(), options, workflows.InterpreterWorkflow, dsl)
	if err != nil {
		log.Fatalf("Unable to execute workflow: %v", err)
	}

	log.Printf("Started workflow '%s'", bpDef.Name)
	log.Printf("WorkflowID: %s", we.GetID())
	log.Printf("RunID: %s", we.GetRunID())
}
