package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/hondyman/semlayer/backend/pkg/workflows"
)

// MockInstanceProvider emulates the BusinessObjectService
type MockInstanceProvider struct {
	Instances map[string]map[string]interface{}
}

func (m *MockInstanceProvider) GetInstanceForValidation(ctx context.Context, tenantID, instanceID string) (map[string]interface{}, error) {
	if val, ok := m.Instances[instanceID]; ok {
		return val, nil
	}
	return nil, fmt.Errorf("instance not found")
}

func main() {
	fmt.Println("==================================================")
	fmt.Println("   TITAN NORTHWINDS CUSTOMER BP DEMO")
	fmt.Println("==================================================")

	// 1. Setup Mock Data (The "Northwinds Database")
	// We simulate a customer with an address that triggers the "Germany" rule
	customerData := map[string]interface{}{
		"customer_id":  "ALFKI",
		"contact_name": "Maria Anders",
		"billing_address": map[string]interface{}{
			"country": "Germany",
			"city":    "Berlin",
		},
	}
	fmt.Printf("[Data] Input Customer: ID=%s, Country=%s\n",
		customerData["customer_id"],
		customerData["billing_address"].(map[string]interface{})["country"])

	// 2. Define the Graph (The "Smart Branching" BP)
	// Start -> Branch (Check Country) -> Germany Node OR USA Node OR Default
	dsl := workflows.WorkflowDefinition{
		Name:        "Northwinds Customer Onboarding",
		StartNodeID: "node_start",
		GlobalState: customerData,
		Nodes: map[string]workflows.WorkflowNode{
			"node_start": {
				ID:         "node_start",
				Type:       "start", // Treated as pass-through
				Name:       "Start Onboarding",
				NextNodeID: stringPtr("node_branch_country"),
			},
			"node_branch_country": {
				ID:   "node_branch_country",
				Type: "BRANCH",
				Name: "Check Country Rules",
				Branches: []workflows.BranchOption{
					{
						Condition:    "billing_address.country == 'Germany'",
						TargetNodeID: "node_germany_compliance",
					},
					{
						Condition:    "billing_address.country == 'USA'",
						TargetNodeID: "node_usa_credit_check",
					},
				},
				// If no match, we could define a default path here or handle as error
				// For this simple demo, we assume match or error
			},
			"node_germany_compliance": {
				ID:         "node_germany_compliance",
				Type:       "ACTIVITY",
				Name:       "Works Council Notification",
				Config:     map[string]interface{}{"activityName": "ActivitySendNotification", "template": "works_council_de"},
				NextNodeID: stringPtr("node_standard_welcome"),
			},
			"node_usa_credit_check": {
				ID:         "node_usa_credit_check",
				Type:       "ACTIVITY",
				Name:       "Perform Credit Check",
				Config:     map[string]interface{}{"activityName": "ActivityCreditCheck"},
				NextNodeID: stringPtr("node_standard_welcome"),
			},
			"node_standard_welcome": {
				ID:     "node_standard_welcome",
				Type:   "ACTIVITY",
				Name:   "Send Welcome Email",
				Config: map[string]interface{}{"activityName": "ActivitySendNotification", "template": "welcome_generic"},
				// No NextNodeID means END
			},
		},
	}

	// 3. Execute the Workflow
	fmt.Println("[Engine] Starting Interpreter Workflow...")
	start := time.Now()

	// We use a mock context/worker setup here for the demo script
	// real execution uses Temporal, but we can call the core logic directly if we allow it,
	// OR use the simulation runner.
	// To keep this demo self-contained and "instant" without needing 3 terminal windows,
	// we will run a Simplified Local Interpreter here that mimics the Temporal behavior
	// just enough to show the branching.

	result, err := runLocalInterpreter(dsl)
	if err != nil {
		log.Fatalf("Workflow failed: %v", err)
	}

	duration := time.Since(start)
	fmt.Println("--------------------------------------------------")
	fmt.Printf("[Result] Status: %s\n", result.Status)
	fmt.Printf("[Metric] Duration: %v\n", duration)
	fmt.Println("==================================================")
}

// runLocalInterpreter is a simplified version of pkg/workflows/dynamic_bp_workflow.go
// specifically for this CLI demo to avoid spinning up a full Temporal cluster connection.
func runLocalInterpreter(dsl workflows.WorkflowDefinition) (*workflows.WorkflowResult, error) {
	currentState := dsl.GlobalState
	currentNodeID := dsl.StartNodeID

	for {
		node, exists := dsl.Nodes[currentNodeID]
		if !exists {
			return nil, fmt.Errorf("node not found: %s", currentNodeID)
		}

		fmt.Printf(" -> Executing Node: [%s] (%s)\n", node.Name, node.Type)

		switch node.Type {
		case "start":
			if node.NextNodeID != nil {
				currentNodeID = *node.NextNodeID
			}

		case "ACTIVITY":
			// Simulate activity execution
			fmt.Printf("    [Action] Running Activity: %v\n", node.Config["activityName"])
			if node.NextNodeID != nil {
				currentNodeID = *node.NextNodeID
			} else {
				// End of flow
				return &workflows.WorkflowResult{Status: "completed", FinalState: currentState}, nil
			}

		case "BRANCH":
			// Simple property evaluator
			nextID := ""
			for _, b := range node.Branches {
				fmt.Printf("    [Logic] Checking: %s... ", b.Condition)
				match := eval(b.Condition, currentState)
				if match {
					fmt.Println("MATCH!")
					nextID = b.TargetNodeID
					break
				} else {
					fmt.Println("False")
				}
			}
			if nextID == "" {
				return nil, fmt.Errorf("no matching branch for node %s", node.ID)
			}
			currentNodeID = nextID
		}
	}
}

// Simple evaluator for the demo
func eval(cond string, data map[string]interface{}) bool {
	// Hardcoded for demo simplicity
	if cond == "billing_address.country == 'Germany'" {
		addr, ok := data["billing_address"].(map[string]interface{})
		return ok && addr["country"] == "Germany"
	}
	if cond == "billing_address.country == 'USA'" {
		addr, ok := data["billing_address"].(map[string]interface{})
		return ok && addr["country"] == "USA"
	}
	return false
}

func stringPtr(s string) *string {
	return &s
}
