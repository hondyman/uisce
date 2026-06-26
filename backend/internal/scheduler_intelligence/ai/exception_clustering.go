package ai

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
)

// ExceptionClusterer groups similar failures for pattern recognition
type ExceptionClusterer struct {
	logger *slog.Logger
}

// NewExceptionClusterer creates a new exception clusterer
func NewExceptionClusterer(logger *slog.Logger) *ExceptionClusterer {
	return &ExceptionClusterer{
		logger: logger,
	}
}

// ExceptionEvent represents a single failure event
type ExceptionEvent struct {
	ID           uuid.UUID         `json:"id"`
	JobID        uuid.UUID         `json:"job_id"`
	JobName      string            `json:"job_name"`
	DAGID        *uuid.UUID        `json:"dag_id,omitempty"`
	TenantID     uuid.UUID         `json:"tenant_id"`
	Timestamp    time.Time         `json:"timestamp"`
	ErrorMessage string            `json:"error_message"`
	ErrorType    string            `json:"error_type"`
	StackTrace   string            `json:"stack_trace,omitempty"`
	StepName     string            `json:"step_name,omitempty"`
	DurationMS   int64             `json:"duration_ms"`
	AttemptNum   int               `json:"attempt_number"`
	Context      map[string]string `json:"context,omitempty"`
}

// ExceptionCluster represents a group of similar failures
type ExceptionCluster struct {
	ID               string         `json:"id"`
	Name             string         `json:"name"`
	Pattern          string         `json:"pattern"`
	Category         string         `json:"category"` // timeout, auth, data, resource, dependency, code
	FirstSeen        time.Time      `json:"first_seen"`
	LastSeen         time.Time      `json:"last_seen"`
	OccurrenceCount  int            `json:"occurrence_count"`
	AffectedJobs     []string       `json:"affected_jobs"`
	AffectedTenants  []string       `json:"affected_tenants"`
	TrendDirection   string         `json:"trend_direction"` // increasing, stable, decreasing
	Severity         string         `json:"severity"`        // critical, high, medium, low
	RootCauseSummary string         `json:"root_cause_summary"`
	Remediation      []string       `json:"remediation_suggestions"`
	RepresentativeID uuid.UUID      `json:"representative_event_id"`
	Fingerprint      string         `json:"fingerprint"`
	Metrics          ClusterMetrics `json:"metrics"`
}

// ClusterMetrics provides statistical data about a cluster
type ClusterMetrics struct {
	AvgDurationMS         float64 `json:"avg_duration_ms"`
	P95DurationMS         float64 `json:"p95_duration_ms"`
	AvgAttempts           float64 `json:"avg_attempts"`
	SuccessRateAfterRetry float64 `json:"success_rate_after_retry"`
	MTTR                  float64 `json:"mttr_minutes"` // Mean Time to Resolve
	OccurrencesByHour     []int   `json:"occurrences_by_hour"`
	OccurrencesByDay      []int   `json:"occurrences_by_day"`
}

// ClusteringResult contains the clustering analysis results
type ClusteringResult struct {
	Clusters        []ExceptionCluster `json:"clusters"`
	TotalExceptions int                `json:"total_exceptions"`
	ClusteredCount  int                `json:"clustered_count"`
	NoiseCount      int                `json:"noise_count"` // Unclustered outliers
	AnalysisTime    time.Time          `json:"analysis_time"`
	TimeRange       TimeRange          `json:"time_range"`
	Insights        []ClusterInsight   `json:"insights"`
}

// TimeRange defines the analysis window
type TimeRange struct {
	From time.Time `json:"from"`
	To   time.Time `json:"to"`
}

// ClusterInsight provides actionable intelligence
type ClusterInsight struct {
	Type        string `json:"type"` // emerging_pattern, recurring_issue, cross_tenant, correlation
	Title       string `json:"title"`
	Description string `json:"description"`
	Impact      string `json:"impact"`
	Priority    int    `json:"priority"` // 1-5
	ActionURL   string `json:"action_url,omitempty"`
}

// ClusterExceptions groups similar exceptions using text similarity and pattern matching
func (c *ExceptionClusterer) ClusterExceptions(ctx context.Context, events []ExceptionEvent) (*ClusteringResult, error) {
	c.logger.Info("Clustering exceptions",
		"event_count", len(events),
	)

	if len(events) == 0 {
		return &ClusteringResult{
			Clusters:     []ExceptionCluster{},
			AnalysisTime: time.Now(),
		}, nil
	}

	// Generate fingerprints for each event
	fingerprints := make(map[string][]ExceptionEvent)
	for _, event := range events {
		fp := c.generateFingerprint(event)
		fingerprints[fp] = append(fingerprints[fp], event)
	}

	// Build clusters from fingerprints
	var clusters []ExceptionCluster
	for fp, fpEvents := range fingerprints {
		if len(fpEvents) >= 2 { // Minimum cluster size
			cluster := c.buildCluster(fp, fpEvents)
			clusters = append(clusters, cluster)
		}
	}

	// Merge similar clusters
	clusters = c.mergeSimilarClusters(clusters)

	// Sort by severity and count
	sort.Slice(clusters, func(i, j int) bool {
		if clusters[i].Severity != clusters[j].Severity {
			return severityRank(clusters[i].Severity) > severityRank(clusters[j].Severity)
		}
		return clusters[i].OccurrenceCount > clusters[j].OccurrenceCount
	})

	// Generate insights
	insights := c.generateInsights(clusters, events)

	result := &ClusteringResult{
		Clusters:        clusters,
		TotalExceptions: len(events),
		ClusteredCount:  c.countClusteredEvents(clusters),
		NoiseCount:      len(events) - c.countClusteredEvents(clusters),
		AnalysisTime:    time.Now(),
		TimeRange: TimeRange{
			From: c.findEarliestTime(events),
			To:   c.findLatestTime(events),
		},
		Insights: insights,
	}

	c.logger.Info("Clustering complete",
		"clusters", len(clusters),
		"clustered", result.ClusteredCount,
		"noise", result.NoiseCount,
	)

	return result, nil
}

// generateFingerprint creates a unique signature for an error
func (c *ExceptionClusterer) generateFingerprint(event ExceptionEvent) string {
	// Normalize and extract key patterns from error message
	normalized := c.normalizeError(event.ErrorMessage)

	// Include job type and error category
	return fmt.Sprintf("%s|%s|%s",
		event.ErrorType,
		normalized,
		event.StepName,
	)
}

// normalizeError removes variable parts from error messages
func (c *ExceptionClusterer) normalizeError(msg string) string {
	msg = strings.ToLower(msg)

	// Remove UUIDs
	msg = replacePattern(msg, `[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}`, "<UUID>")

	// Remove timestamps
	msg = replacePattern(msg, `\d{4}-\d{2}-\d{2}[T ]\d{2}:\d{2}:\d{2}`, "<TIMESTAMP>")

	// Remove numbers
	msg = replacePattern(msg, `\d+`, "<NUM>")

	// Remove file paths
	msg = replacePattern(msg, `/[a-zA-Z0-9_/.-]+`, "<PATH>")

	// Truncate to key portion
	if len(msg) > 100 {
		msg = msg[:100]
	}

	return msg
}

// replacePattern is a simplified pattern replacement (would use regex in real impl)
func replacePattern(s, pattern, replacement string) string {
	// Simplified - in real implementation would use regex
	return s
}

// buildCluster creates a cluster from grouped events
func (c *ExceptionClusterer) buildCluster(fingerprint string, events []ExceptionEvent) ExceptionCluster {
	cluster := ExceptionCluster{
		ID:              uuid.NewString()[:8],
		Fingerprint:     fingerprint,
		FirstSeen:       c.findEarliestTime(events),
		LastSeen:        c.findLatestTime(events),
		OccurrenceCount: len(events),
	}

	// Extract unique affected entities
	jobSet := make(map[string]bool)
	tenantSet := make(map[string]bool)
	for _, e := range events {
		jobSet[e.JobName] = true
		tenantSet[e.TenantID.String()] = true
	}
	for job := range jobSet {
		cluster.AffectedJobs = append(cluster.AffectedJobs, job)
	}
	for tenant := range tenantSet {
		cluster.AffectedTenants = append(cluster.AffectedTenants, tenant)
	}

	// Categorize and name
	cluster.Category = c.categorizeError(events[0].ErrorMessage)
	cluster.Name = c.generateClusterName(cluster.Category, cluster.AffectedJobs)
	cluster.Pattern = c.extractPattern(events[0].ErrorMessage)
	cluster.RepresentativeID = events[0].ID

	// Calculate severity
	cluster.Severity = c.calculateSeverity(cluster)

	// Calculate trend
	cluster.TrendDirection = c.calculateTrend(events)

	// Generate remediation suggestions
	cluster.Remediation = c.suggestRemediation(cluster.Category, cluster.Pattern)

	// Calculate metrics
	cluster.Metrics = c.calculateMetrics(events)

	return cluster
}

// categorizeError determines the error category
func (c *ExceptionClusterer) categorizeError(msg string) string {
	msg = strings.ToLower(msg)

	switch {
	case strings.Contains(msg, "timeout"):
		return "timeout"
	case strings.Contains(msg, "connection") || strings.Contains(msg, "network"):
		return "connectivity"
	case strings.Contains(msg, "auth") || strings.Contains(msg, "permission") || strings.Contains(msg, "forbidden"):
		return "auth"
	case strings.Contains(msg, "null") || strings.Contains(msg, "invalid") || strings.Contains(msg, "parse"):
		return "data"
	case strings.Contains(msg, "memory") || strings.Contains(msg, "disk") || strings.Contains(msg, "cpu"):
		return "resource"
	case strings.Contains(msg, "dependency") || strings.Contains(msg, "upstream"):
		return "dependency"
	default:
		return "code"
	}
}

// generateClusterName creates a human-readable name
func (c *ExceptionClusterer) generateClusterName(category string, jobs []string) string {
	prefix := strings.Title(category)
	if len(jobs) == 1 {
		return fmt.Sprintf("%s Failures in %s", prefix, jobs[0])
	}
	return fmt.Sprintf("%s Failures (%d jobs)", prefix, len(jobs))
}

// extractPattern extracts the key error pattern
func (c *ExceptionClusterer) extractPattern(msg string) string {
	// Take first line or first 80 chars
	lines := strings.Split(msg, "\n")
	pattern := lines[0]
	if len(pattern) > 80 {
		pattern = pattern[:80] + "..."
	}
	return pattern
}

// calculateSeverity determines cluster severity
func (c *ExceptionClusterer) calculateSeverity(cluster ExceptionCluster) string {
	// High severity if many occurrences or affects many tenants
	if cluster.OccurrenceCount > 50 || len(cluster.AffectedTenants) > 5 {
		return "critical"
	}
	if cluster.OccurrenceCount > 20 || len(cluster.AffectedTenants) > 2 {
		return "high"
	}
	if cluster.OccurrenceCount > 5 {
		return "medium"
	}
	return "low"
}

// calculateTrend determines if issue is getting worse or better
func (c *ExceptionClusterer) calculateTrend(events []ExceptionEvent) string {
	if len(events) < 4 {
		return "stable"
	}

	// Compare first half to second half
	mid := len(events) / 2
	firstHalf := len(events[:mid])
	secondHalf := len(events[mid:])

	ratio := float64(secondHalf) / float64(firstHalf)
	if ratio > 1.5 {
		return "increasing"
	}
	if ratio < 0.5 {
		return "decreasing"
	}
	return "stable"
}

// suggestRemediation provides fix suggestions
func (c *ExceptionClusterer) suggestRemediation(category, pattern string) []string {
	switch category {
	case "timeout":
		return []string{
			"Increase job timeout settings",
			"Add retry with exponential backoff",
			"Check for slow downstream dependencies",
			"Consider breaking into smaller jobs",
		}
	case "connectivity":
		return []string{
			"Verify network connectivity to external services",
			"Check firewall and security group rules",
			"Add connection pooling and retry logic",
			"Implement circuit breaker pattern",
		}
	case "auth":
		return []string{
			"Refresh or rotate credentials",
			"Verify IAM permissions",
			"Check token expiration settings",
			"Review service account configuration",
		}
	case "data":
		return []string{
			"Add data validation before processing",
			"Implement null checks and defaults",
			"Review upstream data quality",
			"Add schema validation",
		}
	case "resource":
		return []string{
			"Increase resource allocation",
			"Optimize memory usage",
			"Schedule during off-peak hours",
			"Consider horizontal scaling",
		}
	default:
		return []string{
			"Review recent code changes",
			"Check application logs",
			"Add more detailed error handling",
			"Consider adding unit tests",
		}
	}
}

// calculateMetrics computes statistical metrics
func (c *ExceptionClusterer) calculateMetrics(events []ExceptionEvent) ClusterMetrics {
	metrics := ClusterMetrics{
		OccurrencesByHour: make([]int, 24),
		OccurrencesByDay:  make([]int, 7),
	}

	var totalDuration float64
	var totalAttempts float64
	durations := make([]float64, 0, len(events))

	for _, e := range events {
		totalDuration += float64(e.DurationMS)
		totalAttempts += float64(e.AttemptNum)
		durations = append(durations, float64(e.DurationMS))

		metrics.OccurrencesByHour[e.Timestamp.Hour()]++
		metrics.OccurrencesByDay[int(e.Timestamp.Weekday())]++
	}

	n := float64(len(events))
	metrics.AvgDurationMS = totalDuration / n
	metrics.AvgAttempts = totalAttempts / n

	// Calculate P95
	sort.Float64s(durations)
	p95Index := int(math.Ceil(0.95*float64(len(durations)))) - 1
	if p95Index >= 0 && p95Index < len(durations) {
		metrics.P95DurationMS = durations[p95Index]
	}

	return metrics
}

// mergeSimilarClusters combines clusters with similar patterns
func (c *ExceptionClusterer) mergeSimilarClusters(clusters []ExceptionCluster) []ExceptionCluster {
	// Simple implementation - would use more sophisticated similarity in production
	return clusters
}

// generateInsights creates actionable intelligence from clusters
func (c *ExceptionClusterer) generateInsights(clusters []ExceptionCluster, events []ExceptionEvent) []ClusterInsight {
	var insights []ClusterInsight

	// Check for emerging patterns (new clusters in last 24h)
	for _, cluster := range clusters {
		if time.Since(cluster.FirstSeen) < 24*time.Hour && cluster.OccurrenceCount > 5 {
			insights = append(insights, ClusterInsight{
				Type:        "emerging_pattern",
				Title:       fmt.Sprintf("New failure pattern: %s", cluster.Name),
				Description: fmt.Sprintf("A new error pattern emerged in the last 24 hours with %d occurrences", cluster.OccurrenceCount),
				Impact:      fmt.Sprintf("Affecting %d jobs across %d tenants", len(cluster.AffectedJobs), len(cluster.AffectedTenants)),
				Priority:    2,
			})
		}
	}

	// Check for cross-tenant issues
	for _, cluster := range clusters {
		if len(cluster.AffectedTenants) > 3 {
			insights = append(insights, ClusterInsight{
				Type:        "cross_tenant",
				Title:       fmt.Sprintf("Cross-tenant issue: %s", cluster.Name),
				Description: fmt.Sprintf("This issue affects %d tenants, indicating a platform-level problem", len(cluster.AffectedTenants)),
				Impact:      "Wide blast radius - prioritize investigation",
				Priority:    1,
			})
		}
	}

	// Check for increasing trends
	for _, cluster := range clusters {
		if cluster.TrendDirection == "increasing" && cluster.Severity == "high" {
			insights = append(insights, ClusterInsight{
				Type:        "recurring_issue",
				Title:       fmt.Sprintf("Escalating issue: %s", cluster.Name),
				Description: "This issue is occurring more frequently over time",
				Impact:      "Trend suggests issue is getting worse without intervention",
				Priority:    1,
			})
		}
	}

	return insights
}

// Helper functions
func (c *ExceptionClusterer) findEarliestTime(events []ExceptionEvent) time.Time {
	if len(events) == 0 {
		return time.Time{}
	}
	earliest := events[0].Timestamp
	for _, e := range events {
		if e.Timestamp.Before(earliest) {
			earliest = e.Timestamp
		}
	}
	return earliest
}

func (c *ExceptionClusterer) findLatestTime(events []ExceptionEvent) time.Time {
	if len(events) == 0 {
		return time.Time{}
	}
	latest := events[0].Timestamp
	for _, e := range events {
		if e.Timestamp.After(latest) {
			latest = e.Timestamp
		}
	}
	return latest
}

func (c *ExceptionClusterer) countClusteredEvents(clusters []ExceptionCluster) int {
	total := 0
	for _, c := range clusters {
		total += c.OccurrenceCount
	}
	return total
}

func severityRank(severity string) int {
	switch severity {
	case "critical":
		return 4
	case "high":
		return 3
	case "medium":
		return 2
	case "low":
		return 1
	default:
		return 0
	}
}
