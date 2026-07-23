package observability

import (
	"fmt"
	"sync"
	"time"
)

// BusinessMetric represents a business-level metric
type BusinessMetric struct {
	Name       string                 // Metric name (e.g., "validation_attempts")
	Type       string                 // "counter", "gauge", "histogram"
	Timestamp  time.Time              // When recorded
	Value      float64                // Metric value
	TenantID   string                 // Tenant scope
	Tags       map[string]string      // Additional tags
	Attributes map[string]interface{} // Additional attributes
}

// BusinessMetricsCollector collects and aggregates business metrics
type BusinessMetricsCollector struct {
	serviceName string
	metrics     map[string][]*BusinessMetric
	aggregates  map[string]*BusinessMetricAggregate
	mu          sync.RWMutex
}

// BusinessMetricAggregate represents aggregated metrics over time
type BusinessMetricAggregate struct {
	Name         string
	Count        int64
	Sum          float64
	Min          float64
	Max          float64
	Average      float64
	P50          float64
	P95          float64
	P99          float64
	LastRecorded time.Time
	TenantCounts map[string]int64
}

// NewBusinessMetricsCollector creates a new business metrics collector
func NewBusinessMetricsCollector(serviceName string) *BusinessMetricsCollector {
	return &BusinessMetricsCollector{
		serviceName: serviceName,
		metrics:     make(map[string][]*BusinessMetric),
		aggregates:  make(map[string]*BusinessMetricAggregate),
	}
}

// RecordValidationAttempt records a validation attempt
func (bmc *BusinessMetricsCollector) RecordValidationAttempt(tenantID string, passed bool, durationMs int64) {
	bmc.mu.Lock()
	defer bmc.mu.Unlock()

	metric := &BusinessMetric{
		Name:      "validation_attempts",
		Type:      "counter",
		Timestamp: time.Now(),
		Value:     1,
		TenantID:  tenantID,
		Tags: map[string]string{
			"passed": fmt.Sprintf("%v", passed),
		},
		Attributes: map[string]interface{}{
			"duration_ms": durationMs,
		},
	}

	bmc.recordMetric(metric)

	// Also record pass/fail split
	if passed {
		passMetric := &BusinessMetric{
			Name:      "validation_successes",
			Type:      "counter",
			Timestamp: time.Now(),
			Value:     1,
			TenantID:  tenantID,
		}
		bmc.recordMetric(passMetric)
	} else {
		failMetric := &BusinessMetric{
			Name:      "validation_failures",
			Type:      "counter",
			Timestamp: time.Now(),
			Value:     1,
			TenantID:  tenantID,
		}
		bmc.recordMetric(failMetric)
	}

	// Record duration
	durationMetric := &BusinessMetric{
		Name:      "validation_duration_ms",
		Type:      "histogram",
		Timestamp: time.Now(),
		Value:     float64(durationMs),
		TenantID:  tenantID,
	}
	bmc.recordMetric(durationMetric)
}

// RecordRuleEvaluation records a rule evaluation
func (bmc *BusinessMetricsCollector) RecordRuleEvaluation(tenantID, ruleID string, outcome bool, durationMs int64) {
	bmc.mu.Lock()
	defer bmc.mu.Unlock()

	metric := &BusinessMetric{
		Name:      "rule_evaluations",
		Type:      "counter",
		Timestamp: time.Now(),
		Value:     1,
		TenantID:  tenantID,
		Tags: map[string]string{
			"rule_id": ruleID,
			"outcome": fmt.Sprintf("%v", outcome),
		},
		Attributes: map[string]interface{}{
			"duration_ms": durationMs,
		},
	}

	bmc.recordMetric(metric)

	// Record outcome
	if outcome {
		passMetric := &BusinessMetric{
			Name:      "rule_evaluations_passed",
			Type:      "counter",
			Timestamp: time.Now(),
			Value:     1,
			TenantID:  tenantID,
			Tags: map[string]string{
				"rule_id": ruleID,
			},
		}
		bmc.recordMetric(passMetric)
	} else {
		failMetric := &BusinessMetric{
			Name:      "rule_evaluations_failed",
			Type:      "counter",
			Timestamp: time.Now(),
			Value:     1,
			TenantID:  tenantID,
			Tags: map[string]string{
				"rule_id": ruleID,
			},
		}
		bmc.recordMetric(failMetric)
	}

	// Record duration
	durationMetric := &BusinessMetric{
		Name:      "rule_evaluation_duration_ms",
		Type:      "histogram",
		Timestamp: time.Now(),
		Value:     float64(durationMs),
		TenantID:  tenantID,
	}
	bmc.recordMetric(durationMetric)
}

// RecordNotificationDelivery records notification delivery
func (bmc *BusinessMetricsCollector) RecordNotificationDelivery(tenantID, notificationType string, delivered bool, durationMs int64) {
	bmc.mu.Lock()
	defer bmc.mu.Unlock()

	metric := &BusinessMetric{
		Name:      "notifications_sent",
		Type:      "counter",
		Timestamp: time.Now(),
		Value:     1,
		TenantID:  tenantID,
		Tags: map[string]string{
			"type":      notificationType,
			"delivered": fmt.Sprintf("%v", delivered),
		},
		Attributes: map[string]interface{}{
			"duration_ms": durationMs,
		},
	}

	bmc.recordMetric(metric)

	// Record success/failure
	if delivered {
		successMetric := &BusinessMetric{
			Name:      "notifications_delivered",
			Type:      "counter",
			Timestamp: time.Now(),
			Value:     1,
			TenantID:  tenantID,
			Tags: map[string]string{
				"type": notificationType,
			},
		}
		bmc.recordMetric(successMetric)
	} else {
		failMetric := &BusinessMetric{
			Name:      "notifications_failed",
			Type:      "counter",
			Timestamp: time.Now(),
			Value:     1,
			TenantID:  tenantID,
			Tags: map[string]string{
				"type": notificationType,
			},
		}
		bmc.recordMetric(failMetric)
	}

	// Record delivery time
	durationMetric := &BusinessMetric{
		Name:      "notification_delivery_duration_ms",
		Type:      "histogram",
		Timestamp: time.Now(),
		Value:     float64(durationMs),
		TenantID:  tenantID,
	}
	bmc.recordMetric(durationMetric)
}

// RecordSearchQuery records search query execution
func (bmc *BusinessMetricsCollector) RecordSearchQuery(tenantID string, queryType string, resultCount int, durationMs int64) {
	bmc.mu.Lock()
	defer bmc.mu.Unlock()

	metric := &BusinessMetric{
		Name:      "search_queries",
		Type:      "counter",
		Timestamp: time.Now(),
		Value:     1,
		TenantID:  tenantID,
		Tags: map[string]string{
			"query_type": queryType,
		},
		Attributes: map[string]interface{}{
			"result_count": resultCount,
			"duration_ms":  durationMs,
		},
	}

	bmc.recordMetric(metric)

	// Record result count
	resultMetric := &BusinessMetric{
		Name:      "search_result_count",
		Type:      "gauge",
		Timestamp: time.Now(),
		Value:     float64(resultCount),
		TenantID:  tenantID,
		Tags: map[string]string{
			"query_type": queryType,
		},
	}
	bmc.recordMetric(resultMetric)

	// Record duration
	durationMetric := &BusinessMetric{
		Name:      "search_query_duration_ms",
		Type:      "histogram",
		Timestamp: time.Now(),
		Value:     float64(durationMs),
		TenantID:  tenantID,
	}
	bmc.recordMetric(durationMetric)
}

// RecordPolicyExecution records policy execution
func (bmc *BusinessMetricsCollector) RecordPolicyExecution(tenantID, policyID string, success bool, durationMs int64) {
	bmc.mu.Lock()
	defer bmc.mu.Unlock()

	metric := &BusinessMetric{
		Name:      "policy_executions",
		Type:      "counter",
		Timestamp: time.Now(),
		Value:     1,
		TenantID:  tenantID,
		Tags: map[string]string{
			"policy_id": policyID,
			"success":   fmt.Sprintf("%v", success),
		},
		Attributes: map[string]interface{}{
			"duration_ms": durationMs,
		},
	}

	bmc.recordMetric(metric)

	// Record success/failure
	if success {
		successMetric := &BusinessMetric{
			Name:      "policy_executions_success",
			Type:      "counter",
			Timestamp: time.Now(),
			Value:     1,
			TenantID:  tenantID,
		}
		bmc.recordMetric(successMetric)
	} else {
		failMetric := &BusinessMetric{
			Name:      "policy_executions_failed",
			Type:      "counter",
			Timestamp: time.Now(),
			Value:     1,
			TenantID:  tenantID,
		}
		bmc.recordMetric(failMetric)
	}
}

// recordMetric records a metric internally
func (bmc *BusinessMetricsCollector) recordMetric(metric *BusinessMetric) {
	if _, exists := bmc.metrics[metric.Name]; !exists {
		bmc.metrics[metric.Name] = make([]*BusinessMetric, 0)
	}

	bmc.metrics[metric.Name] = append(bmc.metrics[metric.Name], metric)

	// Update aggregate
	bmc.updateAggregate(metric)
}

// updateAggregate updates the aggregate for a metric
func (bmc *BusinessMetricsCollector) updateAggregate(metric *BusinessMetric) {
	key := metric.Name

	agg, exists := bmc.aggregates[key]
	if !exists {
		agg = &BusinessMetricAggregate{
			Name:         metric.Name,
			TenantCounts: make(map[string]int64),
			Min:          metric.Value,
			Max:          metric.Value,
		}
		bmc.aggregates[key] = agg
	}

	agg.Count++
	agg.Sum += metric.Value
	agg.Average = agg.Sum / float64(agg.Count)
	agg.LastRecorded = metric.Timestamp

	if metric.Value < agg.Min {
		agg.Min = metric.Value
	}
	if metric.Value > agg.Max {
		agg.Max = metric.Value
	}

	if metric.TenantID != "" {
		agg.TenantCounts[metric.TenantID]++
	}
}

// GetMetricCount gets the count of a specific metric
func (bmc *BusinessMetricsCollector) GetMetricCount(metricName string) int64 {
	bmc.mu.RLock()
	defer bmc.mu.RUnlock()

	agg, exists := bmc.aggregates[metricName]
	if !exists {
		return 0
	}

	return agg.Count
}

// GetMetricAggregate gets the aggregate for a metric
func (bmc *BusinessMetricsCollector) GetMetricAggregate(metricName string) *BusinessMetricAggregate {
	bmc.mu.RLock()
	defer bmc.mu.RUnlock()

	agg, exists := bmc.aggregates[metricName]
	if !exists {
		return nil
	}

	return agg
}

// GetAllAggregates gets all aggregates
func (bmc *BusinessMetricsCollector) GetAllAggregates() map[string]*BusinessMetricAggregate {
	bmc.mu.RLock()
	defer bmc.mu.RUnlock()

	result := make(map[string]*BusinessMetricAggregate)
	for name, agg := range bmc.aggregates {
		result[name] = agg
	}

	return result
}

// ExportBusinessMetrics exports business metrics in Prometheus format
func (bmc *BusinessMetricsCollector) ExportBusinessMetrics() string {
	bmc.mu.RLock()
	defer bmc.mu.RUnlock()

	output := ""

	// Export aggregates as Prometheus metrics
	for name, agg := range bmc.aggregates {
		output += fmt.Sprintf("# TYPE business_%s gauge\n", name)
		output += fmt.Sprintf("# HELP business_%s Business metric: %s\n", name, name)

		// Total count
		output += fmt.Sprintf("business_%s_total{service=\"%s\"} %d\n", name, bmc.serviceName, agg.Count)

		// Average
		output += fmt.Sprintf("business_%s_average{service=\"%s\"} %f\n", name, bmc.serviceName, agg.Average)

		// Min/Max
		output += fmt.Sprintf("business_%s_min{service=\"%s\"} %f\n", name, bmc.serviceName, agg.Min)
		output += fmt.Sprintf("business_%s_max{service=\"%s\"} %f\n", name, bmc.serviceName, agg.Max)

		// Per-tenant counts
		for tenantID, count := range agg.TenantCounts {
			output += fmt.Sprintf("business_%s_by_tenant{service=\"%s\",tenant=\"%s\"} %d\n",
				name, bmc.serviceName, tenantID, count)
		}

		output += "\n"
	}

	return output
}

// GetTenantMetrics gets metrics for a specific tenant
func (bmc *BusinessMetricsCollector) GetTenantMetrics(tenantID string) map[string]int64 {
	bmc.mu.RLock()
	defer bmc.mu.RUnlock()

	result := make(map[string]int64)

	for name, agg := range bmc.aggregates {
		if count, exists := agg.TenantCounts[tenantID]; exists {
			result[name] = count
		}
	}

	return result
}
