package ai_routing

import (
	"log"
	"math"
	"math/rand"
)

// RLRoutingAgent implements Q-learning for adaptive routing
type RLRoutingAgent struct {
	qTable         map[string]map[string]float64
	learningRate   float64
	discountFactor float64
	epsilon        float64
	epsilonDecay   float64
	minEpsilon     float64
	episodeCount   int
	maxQValue      float64
}

// NewRLRoutingAgent creates a new reinforcement learning agent
func NewRLRoutingAgent() *RLRoutingAgent {
	return &RLRoutingAgent{
		qTable:         make(map[string]map[string]float64),
		learningRate:   0.1,
		discountFactor: 0.9,
		epsilon:        1.0,
		epsilonDecay:   0.995,
		minEpsilon:     0.01,
		episodeCount:   0,
		maxQValue:      0.0,
	}
}

// SelectAction uses epsilon-greedy policy
func (agent *RLRoutingAgent) SelectAction(features Features, availableBranches []Branch) RLDecision {
	state := agent.encodeState(features)

	// Initialize Q-values for new state
	if _, exists := agent.qTable[state]; !exists {
		agent.qTable[state] = make(map[string]float64)
		for _, branch := range availableBranches {
			// Optimistic initialization
			agent.qTable[state][branch.ID] = branch.SuccessRate * 10.0
		}
	}

	isExploration := false
	var selectedBranch string

	if rand.Float64() < agent.epsilon {
		// Explore: random action
		if len(availableBranches) > 0 {
			selectedBranch = availableBranches[rand.Intn(len(availableBranches))].ID
		}
		isExploration = true
	} else {
		// Exploit: best known action
		selectedBranch = agent.getBestAction(state, availableBranches)
	}

	qValue := agent.qTable[state][selectedBranch]

	return RLDecision{
		BranchID:      selectedBranch,
		QValue:        qValue,
		Epsilon:       agent.epsilon,
		EpisodeCount:  agent.episodeCount,
		IsExploration: isExploration,
	}
}

// UpdateQValue implements Q-learning update rule
func (agent *RLRoutingAgent) UpdateQValue(state string, branchID string, reward float64, nextState string, availableBranches []Branch) {
	// Get current Q-value
	if _, exists := agent.qTable[state]; !exists {
		agent.qTable[state] = make(map[string]float64)
	}

	currentQ := agent.qTable[state][branchID]

	// Get max Q-value for next state
	maxNextQ := agent.getMaxQValue(nextState, availableBranches)

	// Q-learning update: Q(s,a) = Q(s,a) + α[r + γ·max(Q(s',a')) - Q(s,a)]
	newQ := currentQ + agent.learningRate*(reward+agent.discountFactor*maxNextQ-currentQ)

	agent.qTable[state][branchID] = newQ

	if newQ > agent.maxQValue {
		agent.maxQValue = newQ
	}

	// Decay epsilon
	agent.epsilon = math.Max(agent.minEpsilon, agent.epsilon*agent.epsilonDecay)
	agent.episodeCount++

	log.Printf("RL Update: state=%s, branch=%s, newQ=%.3f, reward=%.2f, epsilon=%.4f",
		state, branchID, newQ, reward, agent.epsilon)
}

// CalculateReward computes reward from workflow outcome
func (agent *RLRoutingAgent) CalculateReward(outcome WorkflowOutcome) float64 {
	reward := 0.0

	// Time-based rewards (faster = better)
	if outcome.CompletionTime > 0 && outcome.ExpectedTime > 0 {
		if outcome.CompletionTime < outcome.ExpectedTime {
			reward += 10.0 * (1.0 - outcome.CompletionTime/outcome.ExpectedTime)
		} else {
			reward -= 5.0 * (outcome.CompletionTime/outcome.ExpectedTime - 1.0)
		}
	}

	// Outcome-based rewards
	if outcome.Success {
		reward += 20.0
	} else {
		reward -= 20.0
	}

	// Customer satisfaction rewards
	if outcome.CustomerSatisfactionScore > 0 {
		reward += 15.0 * outcome.CustomerSatisfactionScore
	} else if outcome.CustomerSatisfactionScore < 0 {
		reward -= 10.0 * math.Abs(outcome.CustomerSatisfactionScore)
	}

	// First-time resolution bonus
	if outcome.FirstTimeResolution {
		reward += 10.0
	}

	// Cost penalties
	reward -= outcome.CostIncurred * 0.01

	// Error penalties
	if outcome.ErrorCount > 0 {
		reward -= float64(outcome.ErrorCount) * 5.0
	}

	return reward
}

func (agent *RLRoutingAgent) getBestAction(state string, branches []Branch) string {
	maxQ := math.Inf(-1)
	var bestBranch string

	for _, branch := range branches {
		if q, exists := agent.qTable[state][branch.ID]; exists && q > maxQ {
			maxQ = q
			bestBranch = branch.ID
		}
	}

	if bestBranch == "" && len(branches) > 0 {
		bestBranch = branches[0].ID
	}

	return bestBranch
}

func (agent *RLRoutingAgent) getMaxQValue(state string, branches []Branch) float64 {
	maxQ := 0.0

	if qValues, exists := agent.qTable[state]; exists {
		for _, branch := range branches {
			if q := qValues[branch.ID]; q > maxQ {
				maxQ = q
			}
		}
	}

	return maxQ
}

// encodeState converts features to a state key
func (agent *RLRoutingAgent) encodeState(features Features) string {
	state := RLState{
		CustomerTier:      extractCustomerTier(features.OrderAmount),
		OrderAmountBucket: agent.bucketizeAmount(features.OrderAmount),
		TimeOfDay:         agent.getTimeOfDayCategory(features.Timestamp),
		DayOfWeek:         features.Timestamp.Weekday().String(),
		HistoricalPattern: features.CustomerPattern,
		RiskScore:         agent.bucketizeRiskScore(features.RiskScore),
	}

	return state.CustomerTier + "|" + state.OrderAmountBucket + "|" + state.TimeOfDay + "|" +
		state.DayOfWeek + "|" + state.HistoricalPattern + "|" + state.RiskScore
}

func (agent *RLRoutingAgent) bucketizeAmount(amount float64) string {
	if amount < 100 {
		return "low"
	} else if amount < 500 {
		return "medium"
	} else if amount < 5000 {
		return "high"
	}
	return "very_high"
}

func (agent *RLRoutingAgent) bucketizeRiskScore(risk float64) string {
	if risk < 0.3 {
		return "low_risk"
	} else if risk < 0.6 {
		return "medium_risk"
	}
	return "high_risk"
}

func (agent *RLRoutingAgent) getTimeOfDayCategory(t interface{}) string {
	// Handle different time representations
	switch v := t.(type) {
	case interface{ Hour() int }:
		hour := v.Hour()
		if hour >= 6 && hour < 12 {
			return "morning"
		} else if hour >= 12 && hour < 17 {
			return "afternoon"
		} else if hour >= 17 && hour < 21 {
			return "evening"
		}
		return "night"
	default:
		return "afternoon" // default
	}
}

// GetMetrics returns current agent metrics
func (agent *RLRoutingAgent) GetMetrics() map[string]interface{} {
	return map[string]interface{}{
		"episodes":    agent.episodeCount,
		"epsilon":     agent.epsilon,
		"max_q_value": agent.maxQValue,
		"table_size":  len(agent.qTable),
	}
}

func extractCustomerTier(amount float64) string {
	if amount > 10000 {
		return "vip"
	}
	return "standard"
}
