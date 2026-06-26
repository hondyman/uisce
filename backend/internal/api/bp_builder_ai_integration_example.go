// BP Builder + AI Routing Integration Example
// File: backend/internal/api/bp_builder_ai_integration_example.go

package api

import (
	"context"
	"encoding/json"
	"log"

	"github.com/hondyman/semlayer/backend/pkg/ai_routing"
)

// BPBuilderAIIntegration shows how to integrate AI routing with BP Builder
type BPBuilderAIIntegration struct {
	router            *ai_routing.IntelligentRouter
	feedbackCollector *ai_routing.FeedbackCollector
}

// ExecuteProcessWithAIRouting runs a business process with AI-driven branching
func (bpi *BPBuilderAIIntegration) ExecuteProcessWithAIRouting(
	ctx context.Context,
	workflowID string,
	tenantID string,
	datasourceID string,
	processData map[string]interface{},
) (string, error) {

	// 1. Fetch business process definition from BP Builder
	process, err := bpi.getBusinessProcess(ctx, tenantID, processData["entity"].(string))
	if err != nil {
		return "", err
	}

	log.Printf("Executing process: %s for workflow: %s", process.ProcessName, workflowID)

	// 2. Iterate through steps until hitting a condition
	for _, step := range process.Steps {
		log.Printf("Executing step: %s (type: %s)", step.StepName, step.StepType)

		// If step is AI-routed condition
		if step.ConditionLogic != nil && step.ConditionLogic.Condition == "ai_route" {

			// 3. Map BP branches to AI routing branches
			routingBranches := bpi.mapBPBranchesToRoutingBranches(
				[]string{}, // Would be populated from next steps
				process.Entity,
			)

			// 4. Create routing request
			routingReq := ai_routing.RoutingRequest{
				WorkflowID:        workflowID,
				TenantID:          tenantID,
				DatasourceID:      datasourceID,
				Data:              processData,
				AvailableBranches: routingBranches,
				Context: ai_routing.RoutingContext{
					UserID:           processData["user_id"].(string),
					BusinessPriority: process.Description,
				},
			}

			// 5. Get AI routing decision
			decision, err := bpi.router.Route(ctx, routingReq)
			if err != nil {
				log.Printf("AI routing error: %v, falling back to first branch", err)
				decision = &ai_routing.RoutingDecision{
					SelectedBranchID: routingBranches[0].ID,
					Confidence:       0.5,
				}
			}

			log.Printf("AI routing decision: branch=%s, confidence=%.2f",
				decision.SelectedBranchID, decision.Confidence)

			// Store decision
			bpi.feedbackCollector.StoreRoutingDecision(ctx, decision, routingReq)

			// 6. Route to selected branch (execute corresponding step group)
			selectedStep := bpi.findStepByBranchID(process.Steps, decision.SelectedBranchID)
			if selectedStep != nil {
				// Execute branch-specific steps
				err := bpi.executeStepGroup(ctx, workflowID, *selectedStep, processData)
				if err != nil {
					log.Printf("Step group error: %v", err)
				}
			}

			// 7. Wait for workflow to complete
			outcome, err := bpi.waitForWorkflowCompletion(ctx, workflowID)
			if err == nil {
				// Store outcome for RL training
				outcomeData := ai_routing.WorkflowOutcome{
					WorkflowID:                workflowID,
					RoutingDecisionID:         decision.DecisionID,
					BranchID:                  decision.SelectedBranchID,
					Success:                   outcome.Success,
					CompletionTime:            outcome.Duration,
					CustomerSatisfactionScore: outcome.Satisfaction,
					FirstTimeResolution:       outcome.FirstTimeResolution,
					CostIncurred:              outcome.Cost,
				}

				bpi.feedbackCollector.StoreWorkflowOutcome(ctx, outcomeData)
				log.Printf("Outcome recorded: success=%v, satisfaction=%.2f",
					outcome.Success, outcome.Satisfaction)
			}

			return decision.SelectedBranchID, nil
		}

		// Regular non-AI step
		err := bpi.executeStep(ctx, workflowID, step, processData)
		if err != nil {
			log.Printf("Step failed: %v", err)
			return "", err
		}
	}

	return "", nil
}

// Example: Real-time Business Logic

// Example 1: Credit Application Processing
// "Route credit applications based on risk, amount, customer history"
func (bpi *BPBuilderAIIntegration) CreditApplicationExample() {
	example := map[string]interface{}{
		"business_process": map[string]interface{}{
			"name": "Credit Application",
			"steps": []map[string]interface{}{
				{"name": "Validate Application", "type": "validate"},
				{
					"name": "AI Route by Risk",
					"type": "condition",
					"ai_routing": map[string]interface{}{
						"enabled": true,
						"branches": []string{
							"fast_approve",    // RL learns: VIP + low risk = fast
							"manual_review",   // RL learns: medium risk = review
							"escalation",      // RL learns: high risk = escalation
							"fraud_detection", // RL learns: fraud signals = detection
						},
						"models": map[string]float64{
							"predictive":    0.4, // "Will this application succeed?"
							"rl":            0.3, // "What worked best historically?"
							"sentiment":     0.2, // "What's the customer's intent?"
							"load_balancer": 0.1, // "Which team has capacity?"
						},
					},
				},
				{"name": "Process Decision", "type": "data_entry"},
				{"name": "Final Approval", "type": "approve"},
				{"name": "Notify Customer", "type": "notify"},
			},
		},
		"expected_outcomes": map[string]interface{}{
			"before_ai": map[string]interface{}{
				"accuracy":     "72%",
				"latency":      "8 hours",
				"cost":         "$125 per application",
				"satisfaction": "62%",
			},
			"after_ai_training": map[string]interface{}{
				"accuracy":     "91%",                 // RL learns optimal routing
				"latency":      "15 minutes",          // Load balancer optimizes
				"cost":         "$45 per application", // Fewer manual reviews
				"satisfaction": "88%",                 // Faster decisions
			},
		},
	}

	b, _ := json.MarshalIndent(example, "", "  ")
	log.Printf("Example: %s", string(b))
}

// Example 2: Customer Support Ticket Routing
// "Route support tickets to optimal team based on issue + sentiment"
func (bpi *BPBuilderAIIntegration) CustomerSupportExample() {
	example := map[string]interface{}{
		"routing_logic": map[string]interface{}{
			"step": "Route to Support Team",
			"models_used": map[string]string{
				"sentiment_analysis": "Angry customer? → escalation",
				"predictive":         "Will technical team solve it? → tech_team",
				"rl_agent":           "What team solved this before? → best_team",
				"load_balancer":      "Who has capacity right now? → available_team",
			},
		},
		"outcomes": []map[string]interface{}{
			{
				"case":       "VIP + Negative Sentiment + Technical Issue",
				"ai_routing": "priority_escalation_team",
				"rl_learns":  "VIP + negative → escalation (reward: +20 satisfaction)",
			},
			{
				"case":       "New Customer + Product Question",
				"ai_routing": "onboarding_team",
				"rl_learns":  "New + question → onboarding (reward: +15 satisfaction)",
			},
			{
				"case":       "Regular + Technical Bug Report",
				"ai_routing": "technical_team",
				"rl_learns":  "Regular + bug → tech (reward: +8 satisfaction)",
			},
		},
	}

	b, _ := json.MarshalIndent(example, "", "  ")
	log.Printf("Example: %s", string(b))
}

// Example 3: Claim Processing with Dynamic Routing
// "Route insurance claims based on amount, complexity, fraud risk"
func (bpi *BPBuilderAIIntegration) ClaimProcessingExample() {
	example := map[string]interface{}{
		"workflow": "Insurance Claim Processing",
		"ai_steps": map[string]interface{}{
			"step_1": map[string]interface{}{
				"name": "Initial Validation",
				"type": "validate",
			},
			"step_2": map[string]interface{}{
				"name": "AI Route by Complexity & Risk",
				"type": "ai_condition",
				"available_branches": []string{
					"auto_approve",     // Simple, low risk, high frequency
					"standard_process", // Normal claims
					"special_handling", // Complex or high value
					"fraud_review",     // High fraud score
					"manual_review",    // Edge cases
				},
				"model_contributions": map[string]interface{}{
					"predictive": "35% - Will approval succeed?",
					"rl":         "30% - What path worked best?",
					"sentiment":  "20% - Customer urgency?",
					"load":       "15% - Team availability?",
				},
			},
		},
		"learning_outcomes": []map[string]interface{}{
			{
				"scenario":  "$500 homeowner claim, low risk, 100th similar claim",
				"rl_learns": "auto_approve path: reward +25 (fast + low cost)",
			},
			{
				"scenario":  "$50K commercial claim, high value, first time",
				"rl_learns": "special_handling path: reward +15 (thorough review)",
			},
			{
				"scenario":  "$5K claim, high fraud indicators",
				"rl_learns": "fraud_review path: reward +20 (catches fraud)",
			},
		},
	}

	b, _ := json.MarshalIndent(example, "", "  ")
	log.Printf("Example: %s", string(b))
}

// Integration API for BP Builder
// This would be called from the BP Editor when saving a process with AI conditions

// SaveBusinessProcessWithAIRouting persists a BP with AI conditions
func (bpi *BPBuilderAIIntegration) SaveBusinessProcessWithAIRouting(ctx context.Context,
	bp *BusinessProcess,
) error {
	// Validate AI routing steps
	for _, step := range bp.Steps {
		if step.ConditionLogic != nil && step.ConditionLogic.Condition == "ai_route" {
			// Pre-flight checks
			_ = ctx
			if len(bp.Steps) < 2 {
				return ErrBusinessError("AI routing step needs at least 2 branches")
			}

			log.Printf("Valid AI routing step: %s", step.StepName)
		}
	}

	// Save normally
	return nil
}

// Helper functions (stubs)

func (bpi *BPBuilderAIIntegration) getBusinessProcess(_ context.Context, _ string, _ string,
) (*BusinessProcess, error) {
	// Implementation: fetch from database
	return &BusinessProcess{}, nil
}

func (bpi *BPBuilderAIIntegration) mapBPBranchesToRoutingBranches(
	branchIDs []string,
	_ string,
) []ai_routing.Branch {
	// Implementation: convert BP branches to AI routing format
	branches := make([]ai_routing.Branch, len(branchIDs))
	for i, id := range branchIDs {
		branches[i] = ai_routing.Branch{
			ID:          id,
			Name:        id,
			Capacity:    100,
			CurrentLoad: 0,
		}
	}
	return branches
}

func (bpi *BPBuilderAIIntegration) findStepByBranchID(_ []BPStep, _ string) *BPStep {
	// Implementation: find step matching branch
	return nil
}

func (bpi *BPBuilderAIIntegration) executeStep(
	ctx context.Context,
	workflowID string,
	_ BPStep,
	_ map[string]interface{},
) error {
	_ = ctx
	_ = workflowID
	return nil
}

func (bpi *BPBuilderAIIntegration) executeStepGroup(
	_ context.Context, _ string, _ BPStep, _ map[string]interface{},
) error {
	return nil
}

type WorkflowOutcomeData struct {
	Success             bool
	Duration            float64
	Satisfaction        float64
	FirstTimeResolution bool
	Cost                float64
}

func (bpi *BPBuilderAIIntegration) waitForWorkflowCompletion(
	_ context.Context, _ string,
) (*WorkflowOutcomeData, error) {
	return &WorkflowOutcomeData{}, nil
}

// Error helper
func ErrBusinessError(msg string) error {
	return nil
}
