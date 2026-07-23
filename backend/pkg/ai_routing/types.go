package ai_routing

import (
	"time"
)

// RoutingRequest represents the input for routing decision
type RoutingRequest struct {
	WorkflowID        string                 `json:"workflow_id"`
	TenantID          string                 `json:"tenant_id"`
	DatasourceID      string                 `json:"datasource_id"`
	Data              map[string]interface{} `json:"data"`
	Context           RoutingContext         `json:"context"`
	AvailableBranches []Branch               `json:"available_branches"`
}

// RoutingContext provides contextual information for routing decisions
type RoutingContext struct {
	UserID           string            `json:"user_id"`
	SessionHistory   []HistoricalEvent `json:"session_history"`
	TimeOfDay        time.Time         `json:"time_of_day"`
	SystemLoad       SystemLoadMetrics `json:"system_load"`
	BusinessPriority string            `json:"business_priority"`
}

// RoutingDecision represents the result of the routing decision
type RoutingDecision struct {
	SelectedBranchID  string             `json:"selected_branch_id"`
	Confidence        float64            `json:"confidence"`
	Reasoning         []string           `json:"reasoning"`
	AlternativePaths  []AlternativePath  `json:"alternative_paths"`
	ModelScores       map[string]float64 `json:"model_scores"`
	ExecutionStrategy string             `json:"execution_strategy"` // immediate|delayed|conditional
	Timestamp         time.Time          `json:"timestamp"`
	DecisionID        string             `json:"decision_id"`
}

// Branch represents a possible workflow branch
type Branch struct {
	ID           string                 `json:"id"`
	Name         string                 `json:"name"`
	Description  string                 `json:"description"`
	Capacity     int                    `json:"capacity"`
	CurrentLoad  int                    `json:"current_load"`
	AvgDuration  float64                `json:"avg_duration"`
	SuccessRate  float64                `json:"success_rate"`
	SLA          float64                `json:"sla"`
	Specialties  []string               `json:"specialties"`
	Requirements map[string]interface{} `json:"requirements"`
}

// HistoricalEvent represents a past routing event
type HistoricalEvent struct {
	EventID      string    `json:"event_id"`
	BranchID     string    `json:"branch_id"`
	Success      bool      `json:"success"`
	Duration     float64   `json:"duration"`
	Timestamp    time.Time `json:"timestamp"`
	Satisfaction float64   `json:"satisfaction"`
	Cost         float64   `json:"cost"`
}

// SystemLoadMetrics captures current system state
type SystemLoadMetrics struct {
	CPUUsage       float64            `json:"cpu_usage"`
	MemoryUsage    float64            `json:"memory_usage"`
	QueueDepths    map[string]int     `json:"queue_depths"`
	AvgResponseMs  map[string]float64 `json:"avg_response_ms"`
	ActiveSessions int                `json:"active_sessions"`
}

// AlternativePath represents an alternative routing option
type AlternativePath struct {
	BranchID      string  `json:"branch_id"`
	BranchName    string  `json:"branch_name"`
	Score         float64 `json:"score"`
	Ranking       int     `json:"ranking"`
	Justification string  `json:"justification"`
}

// Features extracted for ML models
type Features struct {
	CustomerTier         string
	OrderAmount          float64
	CustomerLTV          float64
	HistoricalOrderCount int
	AvgOrderValue        float64
	DaysSinceLastOrder   int
	RiskScore            float64
	CustomerPattern      string
	Timestamp            time.Time
	OrderCount           int
	ReturnRate           float64
	ChurnProbability     float64
}

// MLFeatureVector for ML model prediction
type MLFeatureVector struct {
	OrderAmount           float64 `json:"order_amount"`
	CustomerLTV           float64 `json:"customer_ltv"`
	HistoricalOrderCount  int     `json:"historical_order_count"`
	AvgOrderValue         float64 `json:"avg_order_value"`
	DaysSinceLastOrder    int     `json:"days_since_last_order"`
	RiskScore             float64 `json:"risk_score"`
	CustomerTier_VIP      int     `json:"customer_tier_vip"`
	CustomerTier_Standard int     `json:"customer_tier_standard"`
	PaymentMethod_Card    int     `json:"payment_method_card"`
	PaymentMethod_Wire    int     `json:"payment_method_wire"`
	HourOfDay             int     `json:"hour_of_day"`
	DayOfWeek             int     `json:"day_of_week"`
	IsWeekend             int     `json:"is_weekend"`
	IsBusinessHours       int     `json:"is_business_hours"`
	CurrentQueueDepth     int     `json:"current_queue_depth"`
	SystemLoad            float64 `json:"system_load"`
	SeasonalFactor        float64 `json:"seasonal_factor"`
}

// PredictionResult from ML model
type PredictionResult struct {
	BranchID             string             `json:"branch_id"`
	PredictedSuccessRate float64            `json:"predicted_success_rate"`
	EstimatedDuration    float64            `json:"estimated_duration"`
	Confidence           float64            `json:"confidence"`
	FeatureImportance    map[string]float64 `json:"feature_importance"`
}

// WorkflowOutcome tracks the result of a routed workflow
type WorkflowOutcome struct {
	WorkflowID                string    `json:"workflow_id"`
	RoutingDecisionID         string    `json:"routing_decision_id"`
	BranchID                  string    `json:"branch_id"`
	Success                   bool      `json:"success"`
	CompletionTime            float64   `json:"completion_time"`
	ExpectedTime              float64   `json:"expected_time"`
	CustomerSatisfactionScore float64   `json:"customer_satisfaction_score"`
	FirstTimeResolution       bool      `json:"first_time_resolution"`
	CostIncurred              float64   `json:"cost_incurred"`
	ErrorCount                int       `json:"error_count"`
	StateFeatures             string    `json:"state_features"`
	Timestamp                 time.Time `json:"timestamp"`
	ProcessedForTraining      bool      `json:"processed_for_training"`
}

// RoutingMetrics for dashboard and monitoring
type RoutingMetrics struct {
	OverallAccuracy        float64        `json:"overall_accuracy"`
	AvgDecisionTimeMs      float64        `json:"avg_decision_time_ms"`
	ModelAgreementRate     float64        `json:"model_agreement_rate"`
	WorkflowsRoutedToday   int            `json:"workflows_routed_today"`
	BranchDistribution     []BranchMetric `json:"branch_distribution"`
	ModelPerformance       []ModelMetric  `json:"model_performance"`
	RLEpisodes             int            `json:"rl_episodes"`
	RLEpsilon              float64        `json:"rl_epsilon"`
	RLAvgQValue            float64        `json:"rl_avg_q_value"`
	RLLastReward           float64        `json:"rl_last_reward"`
	PredictiveModelVersion string         `json:"predictive_model_version"`
	LastRetrainTime        time.Time      `json:"last_retrain_time"`
}

// BranchMetric for performance tracking
type BranchMetric struct {
	Name          string  `json:"name"`
	Value         int     `json:"value"`
	SuccessRate   float64 `json:"success_rate"`
	AvgDuration   float64 `json:"avg_duration"`
	CurrentLoad   int     `json:"current_load"`
	CapacityUsage float64 `json:"capacity_usage"`
}

// ModelMetric for ML model performance
type ModelMetric struct {
	Model       string    `json:"model"`
	Accuracy    float64   `json:"accuracy"`
	AvgLatency  float64   `json:"avg_latency"`
	F1Score     float64   `json:"f1_score"`
	Precision   float64   `json:"precision"`
	Recall      float64   `json:"recall"`
	LastUpdated time.Time `json:"last_updated"`
}

// RLState represents a state in the RL Q-table
type RLState struct {
	CustomerTier      string
	OrderAmountBucket string
	TimeOfDay         string
	DayOfWeek         string
	HistoricalPattern string
	RiskScore         string
}

// RLDecision represents an RL decision with metadata
type RLDecision struct {
	BranchID      string  `json:"branch_id"`
	QValue        float64 `json:"q_value"`
	Epsilon       float64 `json:"epsilon"`
	EpisodeCount  int     `json:"episode_count"`
	IsExploration bool    `json:"is_exploration"`
}

// ModelResult aggregates results from a single AI model
type ModelResult struct {
	ModelName      string             `json:"model_name"`
	BranchID       string             `json:"branch_id"`
	Score          float64            `json:"score"`
	Confidence     float64            `json:"confidence"`
	FeatureWeights map[string]float64 `json:"feature_weights"`
	Explanation    string             `json:"explanation"`
	LatencyMs      float64            `json:"latency_ms"`
}
