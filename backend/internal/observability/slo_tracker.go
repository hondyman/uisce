package observability

import (
	"fmt"
	"sync"
	"time"
)

// SLOTarget represents a Service Level Objective target
type SLOTarget struct {
	Name               string        // e.g., "Availability", "Latency"
	TargetPercentage   float64       // e.g., 99.9 for 99.9%
	MeasurementWindow  time.Duration // e.g., 30 days
	ErrorBudgetMinutes float64       // Available minutes for errors
}

// SLIMetric represents a Service Level Indicator metric
type SLIMetric struct {
	Name              string    // Metric name
	Timestamp         time.Time // When measured
	SuccessCount      int64     // Successful operations
	FailureCount      int64     // Failed operations
	TotalCount        int64     // Total operations
	CurrentPercentage float64   // Current SLI percentage
}

// ErrorBudget represents the error budget for a service
type ErrorBudget struct {
	ServiceName        string  // Service identifier
	SLOName            string  // SLO target name
	TotalBudgetMinutes float64 // Total available minutes
	UsedBudgetMinutes  float64 // Used minutes
	RemainingMinutes   float64 // Remaining minutes
	BudgetPercentage   float64 // Percentage of budget used (0-100)
	AlertThreshold     float64 // Alert when usage exceeds this % (e.g., 75)
	Status             string  // "healthy", "warning", "critical"
	UpdatedAt          time.Time
}

// SLOTracker manages SLO/SLI tracking and error budgets
type SLOTracker struct {
	serviceName      string
	sloTargets       map[string]*SLOTarget
	sliMetrics       map[string][]*SLIMetric
	errorBudgets     map[string]*ErrorBudget
	mu               sync.RWMutex
	measurementStart time.Time
	alertThreshold   float64 // Default alert threshold (e.g., 75%)
}

// NewSLOTracker creates a new SLO tracker instance
func NewSLOTracker(serviceName string, alertThreshold float64) *SLOTracker {
	return &SLOTracker{
		serviceName:      serviceName,
		sloTargets:       make(map[string]*SLOTarget),
		sliMetrics:       make(map[string][]*SLIMetric),
		errorBudgets:     make(map[string]*ErrorBudget),
		measurementStart: time.Now(),
		alertThreshold:   alertThreshold,
	}
}

// DefineSLO defines a new SLO target
func (st *SLOTracker) DefineSLO(name string, targetPercentage float64, measurementWindow time.Duration) error {
	if targetPercentage < 0 || targetPercentage > 100 {
		return fmt.Errorf("invalid target percentage: %f", targetPercentage)
	}

	st.mu.Lock()
	defer st.mu.Unlock()

	// Calculate error budget in minutes
	windowMinutes := measurementWindow.Minutes()
	errorMinutes := windowMinutes * ((100 - targetPercentage) / 100)

	slo := &SLOTarget{
		Name:               name,
		TargetPercentage:   targetPercentage,
		MeasurementWindow:  measurementWindow,
		ErrorBudgetMinutes: errorMinutes,
	}

	st.sloTargets[name] = slo
	st.sliMetrics[name] = make([]*SLIMetric, 0)

	// Initialize error budget
	st.errorBudgets[name] = &ErrorBudget{
		ServiceName:        st.serviceName,
		SLOName:            name,
		TotalBudgetMinutes: errorMinutes,
		UsedBudgetMinutes:  0,
		RemainingMinutes:   errorMinutes,
		BudgetPercentage:   0,
		AlertThreshold:     st.alertThreshold,
		Status:             "healthy",
		UpdatedAt:          time.Now(),
	}

	return nil
}

// RecordSLI records a Service Level Indicator measurement
func (st *SLOTracker) RecordSLI(sloName string, successCount, failureCount int64) error {
	st.mu.Lock()
	defer st.mu.Unlock()

	slo, exists := st.sloTargets[sloName]
	if !exists {
		return fmt.Errorf("SLO not found: %s", sloName)
	}

	totalCount := successCount + failureCount
	if totalCount == 0 {
		return fmt.Errorf("no operations recorded")
	}

	currentPercentage := (float64(successCount) / float64(totalCount)) * 100

	metric := &SLIMetric{
		Name:              sloName,
		Timestamp:         time.Now(),
		SuccessCount:      successCount,
		FailureCount:      failureCount,
		TotalCount:        totalCount,
		CurrentPercentage: currentPercentage,
	}

	st.sliMetrics[sloName] = append(st.sliMetrics[sloName], metric)

	// Update error budget
	st.updateErrorBudget(sloName, slo, failureCount)

	return nil
}

// updateErrorBudget updates the error budget based on recorded failures
func (st *SLOTracker) updateErrorBudget(sloName string, slo *SLOTarget, failureCount int64) {
	budget, exists := st.errorBudgets[sloName]
	if !exists {
		return
	}

	// Convert failure count to minutes (assuming failures counted over last minute)
	failureMinutes := float64(failureCount) / 60.0
	budget.UsedBudgetMinutes += failureMinutes
	budget.RemainingMinutes = budget.TotalBudgetMinutes - budget.UsedBudgetMinutes

	// Ensure remaining doesn't go negative
	if budget.RemainingMinutes < 0 {
		budget.RemainingMinutes = 0
	}

	// Calculate percentage
	if budget.TotalBudgetMinutes > 0 {
		budget.BudgetPercentage = (budget.UsedBudgetMinutes / budget.TotalBudgetMinutes) * 100
	}

	// Update status
	if budget.BudgetPercentage >= 100 {
		budget.Status = "critical"
	} else if budget.BudgetPercentage >= budget.AlertThreshold {
		budget.Status = "warning"
	} else {
		budget.Status = "healthy"
	}

	budget.UpdatedAt = time.Now()
}

// GetSLI retrieves the latest SLI metric for an SLO
func (st *SLOTracker) GetSLI(sloName string) (*SLIMetric, error) {
	st.mu.RLock()
	defer st.mu.RUnlock()

	metrics, exists := st.sliMetrics[sloName]
	if !exists || len(metrics) == 0 {
		return nil, fmt.Errorf("no SLI metrics found for: %s", sloName)
	}

	return metrics[len(metrics)-1], nil
}

// GetErrorBudget retrieves the error budget for an SLO
func (st *SLOTracker) GetErrorBudget(sloName string) (*ErrorBudget, error) {
	st.mu.RLock()
	defer st.mu.RUnlock()

	budget, exists := st.errorBudgets[sloName]
	if !exists {
		return nil, fmt.Errorf("error budget not found for: %s", sloName)
	}

	return budget, nil
}

// GetAllErrorBudgets retrieves all error budgets
func (st *SLOTracker) GetAllErrorBudgets() map[string]*ErrorBudget {
	st.mu.RLock()
	defer st.mu.RUnlock()

	budgets := make(map[string]*ErrorBudget)
	for name, budget := range st.errorBudgets {
		budgets[name] = budget
	}
	return budgets
}

// CalculateAverageSLI calculates the average SLI over a time period
func (st *SLOTracker) CalculateAverageSLI(sloName string, lookbackDuration time.Duration) (float64, error) {
	st.mu.RLock()
	defer st.mu.RUnlock()

	metrics, exists := st.sliMetrics[sloName]
	if !exists || len(metrics) == 0 {
		return 0, fmt.Errorf("no SLI metrics found for: %s", sloName)
	}

	cutoffTime := time.Now().Add(-lookbackDuration)
	var relevantMetrics []*SLIMetric
	for _, m := range metrics {
		if m.Timestamp.After(cutoffTime) {
			relevantMetrics = append(relevantMetrics, m)
		}
	}

	if len(relevantMetrics) == 0 {
		return 0, fmt.Errorf("no metrics within lookback period")
	}

	var totalSuccess, totalFailure int64
	for _, m := range relevantMetrics {
		totalSuccess += m.SuccessCount
		totalFailure += m.FailureCount
	}

	totalCount := totalSuccess + totalFailure
	if totalCount == 0 {
		return 0, fmt.Errorf("no operations in lookback period")
	}

	return (float64(totalSuccess) / float64(totalCount)) * 100, nil
}

// CheckSLOCompliance checks if SLO is being met
func (st *SLOTracker) CheckSLOCompliance(sloName string) (bool, float64, error) {
	st.mu.RLock()
	defer st.mu.RUnlock()

	slo, exists := st.sloTargets[sloName]
	if !exists {
		return false, 0, fmt.Errorf("SLO not found: %s", sloName)
	}

	metrics, exists := st.sliMetrics[sloName]
	if !exists || len(metrics) == 0 {
		return false, 0, fmt.Errorf("no SLI metrics found for: %s", sloName)
	}

	latestMetric := metrics[len(metrics)-1]
	isCompliant := latestMetric.CurrentPercentage >= slo.TargetPercentage

	return isCompliant, latestMetric.CurrentPercentage, nil
}

// GetAlertStatus returns alert status based on error budget
func (st *SLOTracker) GetAlertStatus(sloName string) (string, error) {
	st.mu.RLock()
	defer st.mu.RUnlock()

	budget, exists := st.errorBudgets[sloName]
	if !exists {
		return "", fmt.Errorf("error budget not found for: %s", sloName)
	}

	return budget.Status, nil
}

// ExportSLOMetrics exports all SLO metrics in Prometheus format
func (st *SLOTracker) ExportSLOMetrics() string {
	st.mu.RLock()
	defer st.mu.RUnlock()

	output := ""

	// SLO Target metrics
	output += "# TYPE slo_target_percentage gauge\n"
	output += "# HELP slo_target_percentage Target SLO percentage\n"
	for name, slo := range st.sloTargets {
		output += fmt.Sprintf("slo_target_percentage{service=\"%s\",slo=\"%s\"} %f\n",
			st.serviceName, name, slo.TargetPercentage)
	}

	output += "\n"

	// SLI Current metrics
	output += "# TYPE sli_current_percentage gauge\n"
	output += "# HELP sli_current_percentage Current SLI percentage\n"
	for name, metrics := range st.sliMetrics {
		if len(metrics) > 0 {
			latest := metrics[len(metrics)-1]
			output += fmt.Sprintf("sli_current_percentage{service=\"%s\",slo=\"%s\"} %f\n",
				st.serviceName, name, latest.CurrentPercentage)
		}
	}

	output += "\n"

	// Error budget metrics
	output += "# TYPE error_budget_remaining_minutes gauge\n"
	output += "# HELP error_budget_remaining_minutes Remaining error budget in minutes\n"
	for name, budget := range st.errorBudgets {
		output += fmt.Sprintf("error_budget_remaining_minutes{service=\"%s\",slo=\"%s\"} %f\n",
			st.serviceName, name, budget.RemainingMinutes)
	}

	output += "\n"

	output += "# TYPE error_budget_percentage gauge\n"
	output += "# HELP error_budget_percentage Error budget used percentage (0-100)\n"
	for name, budget := range st.errorBudgets {
		output += fmt.Sprintf("error_budget_percentage{service=\"%s\",slo=\"%s\"} %f\n",
			st.serviceName, name, budget.BudgetPercentage)
	}

	output += "\n"

	// Budget status
	output += "# TYPE error_budget_status gauge\n"
	output += "# HELP error_budget_status Error budget status (0=healthy, 1=warning, 2=critical)\n"
	for name, budget := range st.errorBudgets {
		statusValue := 0.0
		if budget.Status == "warning" {
			statusValue = 1.0
		} else if budget.Status == "critical" {
			statusValue = 2.0
		}
		output += fmt.Sprintf("error_budget_status{service=\"%s\",slo=\"%s\",status=\"%s\"} %f\n",
			st.serviceName, name, budget.Status, statusValue)
	}

	return output
}

// ResetErrorBudget resets the error budget for a specific SLO (typically at window boundary)
func (st *SLOTracker) ResetErrorBudget(sloName string) error {
	st.mu.Lock()
	defer st.mu.Unlock()

	slo, exists := st.sloTargets[sloName]
	if !exists {
		return fmt.Errorf("SLO not found: %s", sloName)
	}

	budget, exists := st.errorBudgets[sloName]
	if !exists {
		return fmt.Errorf("error budget not found: %s", sloName)
	}

	budget.UsedBudgetMinutes = 0
	budget.RemainingMinutes = slo.ErrorBudgetMinutes
	budget.BudgetPercentage = 0
	budget.Status = "healthy"
	budget.UpdatedAt = time.Now()

	return nil
}
