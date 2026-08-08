package ai_routing

import (
	"context"
	"database/sql"
	"encoding/json"
	"log"
	"time"

	"github.com/hondyman/uisce/libs/db/queries"
)

// FeedbackCollector manages the feedback loop for continuous learning
type FeedbackCollector struct {
	router             *IntelligentRouter
	rlAgent            *RLRoutingAgent
	db                 *sql.DB
	retrainingInterval time.Duration
	batchSize          int
}

// NewFeedbackCollector creates a new feedback collector
func NewFeedbackCollector(
	router *IntelligentRouter,
	rlAgent *RLRoutingAgent,
	db *sql.DB,
) *FeedbackCollector {
	return &FeedbackCollector{
		router:             router,
		rlAgent:            rlAgent,
		db:                 db,
		retrainingInterval: 1 * time.Hour,
		batchSize:          100,
	}
}

// StartFeedbackLoop begins continuous learning from outcomes
func (fc *FeedbackCollector) StartFeedbackLoop(ctx context.Context) {
	ticker := time.NewTicker(fc.retrainingInterval)
	defer ticker.Stop()

	log.Println("Starting AI routing feedback loop")

	for {
		select {
		case <-ctx.Done():
			log.Println("Feedback loop stopped")
			return
		case <-ticker.C:
			if err := fc.processOutcomes(ctx); err != nil {
				log.Printf("Error processing outcomes: %v", err)
			}
		}
	}
}

// processOutcomes fetches completed workflows and updates RL agent
func (fc *FeedbackCollector) processOutcomes(ctx context.Context) error {
	// TODO(hasura-migration): Replace SQL query with Hasura GraphQL query
	// Example GraphQL query:
	// query GetUnprocessedOutcomes($limit: Int!, $since: timestamptz!) {
	//   workflow_outcomes(
	//     where: {
	//       processed_for_training: {_eq: false},
	//       completed_at: {_gte: $since}
	//     },
	//     limit: $limit
	//   ) {
	//     workflow_id
	//     routing_decision_id
	//     branch_id
	//     success
	//     completion_time_minutes
	//     expected_time_minutes
	//     customer_satisfaction_score
	//     first_time_resolution
	//     cost_incurred
	//     error_count
	//     state_features
	//   }
	// }
	rows, err := fc.db.QueryContext(ctx, queries.SelectUnprocessedOutcomes, fc.batchSize)

	if err != nil {
		return err
	}
	defer rows.Close()

	updateCount := 0

	for rows.Next() {
		var outcome WorkflowOutcome
		var stateFeatures sql.NullString

		err := rows.Scan(
			&outcome.WorkflowID, &outcome.RoutingDecisionID, &outcome.BranchID,
			&outcome.Success, &outcome.CompletionTime, &outcome.ExpectedTime,
			&outcome.CustomerSatisfactionScore, &outcome.FirstTimeResolution,
			&outcome.CostIncurred, &outcome.ErrorCount, &stateFeatures,
		)

		if err != nil {
			log.Printf("Error scanning outcome: %v", err)
			continue
		}

		// Calculate reward
		reward := fc.rlAgent.CalculateReward(outcome)

		// Update RL agent
		nextState := stateFeatures.String
		if stateFeatures.Valid && nextState != "" {
			fc.rlAgent.UpdateQValue(
				stateFeatures.String,
				outcome.BranchID,
				reward,
				"", // next state unknown in offline learning
				nil,
			)
		}

		// Mark as processed
		// TODO(hasura-migration): Replace SQL UPDATE with Hasura GraphQL mutation
		// Example GraphQL mutation:
		// mutation MarkOutcomeProcessed($workflowId: String!, $reward: Float!) {
		//   update_workflow_outcomes(
		//     where: {workflow_id: {_eq: $workflowId}},
		//     _set: {
		//       processed_for_training: true,
		//       rl_reward: $reward,
		//       updated_at: "now()"
		//     }
		//   ) {
		//     affected_rows
		// }
		// }
		_, err = fc.db.ExecContext(ctx, queries.UpdateOutcomeProcessed, reward, outcome.WorkflowID)

		if err != nil {
			log.Printf("Error updating outcome: %v", err)
			continue
		}

		updateCount++

		log.Printf("RL training update: workflow=%s, branch=%s, reward=%.2f",
			outcome.WorkflowID, outcome.BranchID, reward)
	}

	log.Printf("Processed %d outcomes for RL training", updateCount)
	return rows.Err()
}

// StoreRoutingDecision persists a routing decision
func (fc *FeedbackCollector) StoreRoutingDecision(ctx context.Context, decision *RoutingDecision, req RoutingRequest) error {
	modelScoresJSON, _ := json.Marshal(decision.ModelScores)
	reasoningJSON, _ := json.Marshal(decision.Reasoning)

	// TODO(hasura-migration): Replace SQL INSERT with Hasura GraphQL mutation
	// Example GraphQL mutation:
	// mutation StoreRoutingDecision($object: routing_decisions_insert_input!) {
	//   insert_routing_decisions_one(object: $object) {
	//     decision_id
	//     workflow_id
	//   }
	// }
	// Variables: {
	//   "object": {
	//     "decision_id": "<uuid>",
	//     "workflow_id": "<uuid>",
	//     "tenant_id": "<uuid>",
	//     "datasource_id": "<uuid>",
	//     "selected_branch_id": "<uuid>",
	//     "confidence": 0.95,
	//     "reasoning": {...},
	//     "model_scores": {...},
	//     "execution_strategy": "ensemble",
	//     "created_at": "now()"
	// }
	// }
	_, err := fc.db.ExecContext(ctx, queries.InsertRoutingDecision,
		decision.DecisionID, req.WorkflowID, req.TenantID, req.DatasourceID,
		decision.SelectedBranchID, decision.Confidence,
		string(reasoningJSON), string(modelScoresJSON),
		decision.ExecutionStrategy,
	)

	return err
}

// StoreWorkflowOutcome persists workflow completion data
func (fc *FeedbackCollector) StoreWorkflowOutcome(ctx context.Context, outcome WorkflowOutcome) error {
	// TODO(hasura-migration): Replace SQL INSERT with Hasura GraphQL mutation
	// Example GraphQL mutation:
	// mutation StoreWorkflowOutcome($object: workflow_outcomes_insert_input!) {
	//   insert_workflow_outcomes_one(object: $object) {
	//     workflow_id
	//     routing_decision_id
	// }
	// }
	_, err := fc.db.ExecContext(ctx, queries.InsertWorkflowOutcome,
		outcome.WorkflowID, outcome.RoutingDecisionID, outcome.BranchID,
		outcome.Success, outcome.CompletionTime, outcome.ExpectedTime,
		outcome.CustomerSatisfactionScore, outcome.FirstTimeResolution,
		outcome.CostIncurred, outcome.ErrorCount, outcome.StateFeatures,
	)

	return err
}

// GetDecisionHistory retrieves routing decisions for a workflow
func (fc *FeedbackCollector) GetDecisionHistory(ctx context.Context, workflowID string, limit int) ([]RoutingDecision, error) {
	// TODO(hasura-migration): Replace SQL query with Hasura GraphQL query
	// Example GraphQL query:
	// query GetDecisionHistory($workflowId: String!, $limit: Int!) {
	//   routing_decisions(
	//     where: {workflow_id: {_eq: $workflowId}},
	//     order_by: {created_at: desc},
	//     limit: $limit
	//   ) {
	//     decision_id
	//     selected_branch_id
	//     confidence
	//     reasoning
	//     model_scores
	//     created_at
	// }
	// }
	rows, err := fc.db.QueryContext(ctx, queries.GetDecisionHistory, workflowID, limit)

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var decisions []RoutingDecision

	for rows.Next() {
		var decision RoutingDecision
		var reasoningJSON, scoresJSON []byte

		err := rows.Scan(
			&decision.DecisionID, &decision.SelectedBranchID, &decision.Confidence,
			&reasoningJSON, &scoresJSON, &decision.Timestamp,
		)

		if err != nil {
			log.Printf("Error scanning decision: %v", err)
			continue
		}

		json.Unmarshal(reasoningJSON, &decision.Reasoning)
		json.Unmarshal(scoresJSON, &decision.ModelScores)

		decisions = append(decisions, decision)
	}

	return decisions, rows.Err()
}

// GetOutcomeStats returns performance statistics
func (fc *FeedbackCollector) GetOutcomeStats(ctx context.Context, tenantID string, hoursBack int) (map[string]interface{}, error) {
	var totalCount int
	var successCount int
	var avgSatisfaction float64
	var avgCost float64

	row := fc.db.QueryRowContext(ctx, queries.GetOutcomeStats, tenantID, hoursBack)

	err := row.Scan(&totalCount, &successCount, &avgSatisfaction, &avgCost)
	if err != nil {
		return nil, err
	}

	successRate := 0.0
	if totalCount > 0 {
		successRate = float64(successCount) / float64(totalCount)
	}

	return map[string]interface{}{
		"total_workflows":  totalCount,
		"successful":       successCount,
		"success_rate":     successRate,
		"avg_satisfaction": avgSatisfaction,
		"avg_cost":         avgCost,
		"hours_analyzed":   hoursBack,
	}, nil
}

// GetBranchPerformance returns per-branch metrics
func (fc *FeedbackCollector) GetBranchPerformance(ctx context.Context, tenantID string) ([]BranchMetric, error) {
	rows, err := fc.db.QueryContext(ctx, `
		SELECT 
			branch_id,
			COUNT(*) as total_routed,
			COUNT(CASE WHEN success THEN 1 END) as successful,
			AVG(completion_time_minutes) as avg_duration
		FROM workflow_outcomes
		WHERE tenant_id = $1
		  AND created_at >= NOW() - INTERVAL '24 hours'
		GROUP BY branch_id
		ORDER BY total_routed DESC
	`, tenantID)

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var metrics []BranchMetric

	for rows.Next() {
		var metric BranchMetric
		var totalRouted int
		var successful int

		err := rows.Scan(&metric.Name, &totalRouted, &successful, &metric.AvgDuration)
		if err != nil {
			log.Printf("Error scanning branch metric: %v", err)
			continue
		}

		metric.Value = totalRouted
		metric.SuccessRate = float64(successful) / float64(totalRouted)

		metrics = append(metrics, metric)
	}

	return metrics, rows.Err()
}
