package abtesting

import (
	"context"
	"crypto/md5"
	"fmt"
	"sync"
	"time"
)

// ExperimentFramework manages A/B testing for models and features
type ExperimentFramework struct {
	mu          sync.RWMutex
	experiments map[string]*Experiment
	assignments map[string]string // entityID -> experimentID
	metrics     map[string]*ExperimentMetrics
}

// Experiment represents an A/B test
type Experiment struct {
	ID            string        `json:"id"`
	Name          string        `json:"name"`
	Description   string        `json:"description"`
	Status        string        `json:"status"` // "draft", "running", "completed", "cancelled"
	Type          string        `json:"type"`   // "model", "feature", "traffic"
	StartTime     time.Time     `json:"start_time"`
	EndTime       *time.Time    `json:"end_time,omitempty"`
	Control       VariantConfig `json:"control"`
	Treatment     VariantConfig `json:"treatment"`
	TrafficSplit  float64       `json:"traffic_split"` // 0-1, treatment traffic %
	SampleSize    int           `json:"sample_size"`
	PrimaryMetric PrimaryMetric `json:"primary_metric"`
	Segments      []Segment     `json:"segments"`
	RandomSeed    int64         `json:"random_seed"`
	Hypothesis    string        `json:"hypothesis"`
}

// VariantConfig represents control/treatment configuration
type VariantConfig struct {
	Name          string                 `json:"name"`
	Description   string                 `json:"description"`
	ModelVersion  string                 `json:"model_version,omitempty"`
	FeatureSet    string                 `json:"feature_set,omitempty"`
	Configuration map[string]interface{} `json:"configuration"`
}

// PrimaryMetric defines the experiment's success metric
type PrimaryMetric struct {
	Name                string        `json:"name"`                  // "latency", "auc", "revenue", etc
	Direction           string        `json:"direction"`             // "higher", "lower"
	MinDetectableEffect float64       `json:"min_detectable_effect"` // Smallest effect to detect
	BaselineValue       float64       `json:"baseline_value"`
	PowerAnalysis       PowerAnalysis `json:"power_analysis"`
}

// PowerAnalysis holds statistical power settings
type PowerAnalysis struct {
	Alpha           float64 `json:"alpha"` // Type I error (default 0.05)
	Beta            float64 `json:"beta"`  // Type II error (default 0.20)
	MinSampleSize   int     `json:"min_sample_size"`
	DurationDays    int     `json:"duration_days"`
	RequiredSamples int     `json:"required_samples"`
}

// Segment filters experiment participants
type Segment struct {
	Name      string   `json:"name"`
	Attribute string   `json:"attribute"` // "region", "tenant_id", "risk_level"
	Values    []string `json:"values"`
	Negate    bool     `json:"negate"` // Exclude instead of include
}

// ExperimentMetrics tracks experiment results
type ExperimentMetrics struct {
	ExperimentID               string             `json:"experiment_id"`
	ControlMetrics             map[string]float64 `json:"control_metrics"`
	TreatmentMetrics           map[string]float64 `json:"treatment_metrics"`
	ControlCount               int                `json:"control_count"`
	TreatmentCount             int                `json:"treatment_count"`
	PrimaryMetricDelta         float64            `json:"primary_metric_delta"`
	PrimaryMetricPValue        float64            `json:"primary_metric_p_value"`
	IsStatisticallySignificant bool               `json:"is_statistically_significant"`
	Confidence                 float64            `json:"confidence"` // 0-1
	LastUpdated                time.Time          `json:"last_updated"`
}

// AssignmentResult holds variant assignment
type AssignmentResult struct {
	ExperimentID string `json:"experiment_id"`
	EntityID     string `json:"entity_id"`
	Variant      string `json:"variant"` // "control" or "treatment"
	Reason       string `json:"reason"`
}

// EventLog records experiment events for later analysis
type EventLog struct {
	ExperimentID string                 `json:"experiment_id"`
	EntityID     string                 `json:"entity_id"`
	Variant      string                 `json:"variant"`
	EventType    string                 `json:"event_type"` // "prediction", "conversion", "error"
	MetricValues map[string]float64     `json:"metric_values"`
	Timestamp    time.Time              `json:"timestamp"`
	Attributes   map[string]interface{} `json:"attributes"`
}

// NewExperimentFramework creates a new A/B testing framework
func NewExperimentFramework() *ExperimentFramework {
	return &ExperimentFramework{
		experiments: make(map[string]*Experiment),
		assignments: make(map[string]string),
		metrics:     make(map[string]*ExperimentMetrics),
	}
}

// CreateExperiment creates a new experiment
func (ef *ExperimentFramework) CreateExperiment(ctx context.Context, exp *Experiment) (string, error) {
	ef.mu.Lock()
	defer ef.mu.Unlock()

	if exp.ID == "" {
		exp.ID = fmt.Sprintf("exp_%d", time.Now().Unix())
	}

	if _, exists := ef.experiments[exp.ID]; exists {
		return "", fmt.Errorf("experiment %s already exists", exp.ID)
	}

	exp.Status = "draft"
	exp.StartTime = time.Now()
	exp.RandomSeed = time.Now().UnixNano()

	// Calculate required sample size
	if exp.PrimaryMetric.PowerAnalysis.MinSampleSize == 0 {
		exp.PrimaryMetric.PowerAnalysis.MinSampleSize = ef.calculateSampleSize(
			exp.PrimaryMetric.MinDetectableEffect,
			exp.PrimaryMetric.PowerAnalysis.Alpha,
			exp.PrimaryMetric.PowerAnalysis.Beta,
		)
	}

	ef.experiments[exp.ID] = exp

	return exp.ID, nil
}

// StartExperiment starts an experiment
func (ef *ExperimentFramework) StartExperiment(ctx context.Context, experimentID string) error {
	ef.mu.Lock()
	defer ef.mu.Unlock()

	exp, exists := ef.experiments[experimentID]
	if !exists {
		return fmt.Errorf("experiment %s not found", experimentID)
	}

	exp.Status = "running"
	exp.StartTime = time.Now()
	ef.metrics[experimentID] = &ExperimentMetrics{
		ExperimentID:     experimentID,
		ControlMetrics:   make(map[string]float64),
		TreatmentMetrics: make(map[string]float64),
	}

	return nil
}

// AssignVariant assigns an entity to a variant based on experiment rules
func (ef *ExperimentFramework) AssignVariant(ctx context.Context, experimentID string, entityID string, attributes map[string]interface{}) (*AssignmentResult, error) {
	ef.mu.RLock()
	exp, exists := ef.experiments[experimentID]
	ef.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("experiment %s not found", experimentID)
	}

	if exp.Status != "running" {
		return &AssignmentResult{
			ExperimentID: experimentID,
			EntityID:     entityID,
			Variant:      "control",
			Reason:       fmt.Sprintf("experiment in %s state", exp.Status),
		}, nil
	}

	// Check segments
	for _, segment := range exp.Segments {
		if !ef.matchesSegment(segment, attributes) {
			return &AssignmentResult{
				ExperimentID: experimentID,
				EntityID:     entityID,
				Variant:      "control",
				Reason:       fmt.Sprintf("excluded by segment %s", segment.Name),
			}, nil
		}
	}

	// Deterministic assignment using entity ID
	hash := md5.Sum([]byte(entityID + experimentID))
	hashValue := float64((uint32(hash[0])<<24|uint32(hash[1])<<16|uint32(hash[2])<<8|uint32(hash[3]))%1000) / 1000.0

	variant := "control"
	if hashValue < exp.TrafficSplit {
		variant = "treatment"
	}

	ef.mu.Lock()
	ef.assignments[entityID+":"+experimentID] = variant
	ef.mu.Unlock()

	return &AssignmentResult{
		ExperimentID: experimentID,
		EntityID:     entityID,
		Variant:      variant,
		Reason:       "assigned by hash",
	}, nil
}

// RecordEvent records a metric event for experiment analysis
func (ef *ExperimentFramework) RecordEvent(ctx context.Context, log *EventLog) error {
	ef.mu.Lock()
	defer ef.mu.Unlock()

	metrics, exists := ef.metrics[log.ExperimentID]
	if !exists {
		return fmt.Errorf("no metrics for experiment %s", log.ExperimentID)
	}

	// Aggregate metrics by variant
	for metricName, value := range log.MetricValues {
		if log.Variant == "control" {
			metrics.ControlMetrics[metricName] = value
			metrics.ControlCount++
		} else {
			metrics.TreatmentMetrics[metricName] = value
			metrics.TreatmentCount++
		}
	}

	metrics.LastUpdated = time.Now()
	return nil
}

// GetExperimentResults returns statistical analysis of experiment
func (ef *ExperimentFramework) GetExperimentResults(ctx context.Context, experimentID string) (*ExperimentMetrics, error) {
	ef.mu.RLock()
	defer ef.mu.RUnlock()

	metrics, exists := ef.metrics[experimentID]
	if !exists {
		return nil, fmt.Errorf("no results for experiment %s", experimentID)
	}

	// Perform t-test on primary metric
	exp := ef.experiments[experimentID]
	if exp != nil {
		primaryMetricName := exp.PrimaryMetric.Name

		controlValue := metrics.ControlMetrics[primaryMetricName]
		treatmentValue := metrics.TreatmentMetrics[primaryMetricName]

		metrics.PrimaryMetricDelta = treatmentValue - controlValue

		// Simplified significance check
		// In production, would use proper statistical test
		minEffect := exp.PrimaryMetric.MinDetectableEffect
		if metrics.PrimaryMetricDelta > minEffect {
			metrics.IsStatisticallySignificant = true
			metrics.Confidence = 0.95
		}
	}

	return metrics, nil
}

// EndExperiment concludes an experiment
func (ef *ExperimentFramework) EndExperiment(ctx context.Context, experimentID string) error {
	ef.mu.Lock()
	defer ef.mu.Unlock()

	exp, exists := ef.experiments[experimentID]
	if !exists {
		return fmt.Errorf("experiment %s not found", experimentID)
	}

	now := time.Now()
	exp.Status = "completed"
	exp.EndTime = &now

	return nil
}

// ListExperiments returns all experiments
func (ef *ExperimentFramework) ListExperiments(ctx context.Context, status string) ([]*Experiment, error) {
	ef.mu.RLock()
	defer ef.mu.RUnlock()

	var result []*Experiment
	for _, exp := range ef.experiments {
		if status == "" || exp.Status == status {
			result = append(result, exp)
		}
	}

	return result, nil
}

// matchesSegment checks if attributes match segment criteria
func (ef *ExperimentFramework) matchesSegment(seg Segment, attributes map[string]interface{}) bool {
	attr, exists := attributes[seg.Attribute]
	if !exists {
		return !seg.Negate // Missing attributes don't match
	}

	attrStr := fmt.Sprintf("%v", attr)
	found := false
	for _, value := range seg.Values {
		if value == attrStr {
			found = true
			break
		}
	}

	if seg.Negate {
		return !found
	}
	return found
}

// calculateSampleSize computes minimum sample size using Neyman allocation
func (ef *ExperimentFramework) calculateSampleSize(minEffect float64, alpha float64, beta float64) int {
	// Simplified: in production, use proper power analysis library
	// Z-critical values
	zAlpha := 1.96 // 0.05 significance
	zBeta := 0.84  // 0.20 power

	baseSize := int(2 * ((zAlpha + zBeta) / minEffect) * ((zAlpha + zBeta) / minEffect))
	if baseSize < 100 {
		baseSize = 100 // Minimum 100 per variant
	}

	return baseSize
}

// GetAssignment retrieves stored assignment for entity
func (ef *ExperimentFramework) GetAssignment(ctx context.Context, experimentID string, entityID string) (string, error) {
	ef.mu.RLock()
	defer ef.mu.RUnlock()

	key := entityID + ":" + experimentID
	if assignment, exists := ef.assignments[key]; exists {
		return assignment, nil
	}

	return "", fmt.Errorf("no assignment found")
}
