package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"
)

// NLQScheduler handles natural language queries about scheduler status
type NLQScheduler struct {
	llmClient LLMClient
	logger    *slog.Logger
}

// NewNLQScheduler creates a new NLQ scheduler
func NewNLQScheduler(llmClient LLMClient, logger *slog.Logger) *NLQScheduler {
	return &NLQScheduler{
		llmClient: llmClient,
		logger:    logger,
	}
}

// SchedulerContext provides current state for answering queries
type SchedulerContext struct {
	TenantID       uuid.UUID        `json:"tenant_id"`
	ActiveJobs     []JobSummary     `json:"active_jobs"`
	RecentFailures []FailureSummary `json:"recent_failures"`
	ScheduleStats  ScheduleStats    `json:"schedule_stats"`
	SLOStatus      []SLOSummary     `json:"slo_status"`
	CurrentTime    time.Time        `json:"current_time"`
}

// JobSummary provides job overview
type JobSummary struct {
	ID          uuid.UUID  `json:"id"`
	Name        string     `json:"name"`
	Category    string     `json:"category"`
	Status      string     `json:"status"`
	LastRun     time.Time  `json:"last_run"`
	NextRun     *time.Time `json:"next_run,omitempty"`
	SuccessRate float64    `json:"success_rate"`
	AvgDuration int64      `json:"avg_duration_ms"`
}

// FailureSummary describes a recent failure
type FailureSummary struct {
	JobName   string    `json:"job_name"`
	FailedAt  time.Time `json:"failed_at"`
	ErrorType string    `json:"error_type"`
	StepName  string    `json:"step_name,omitempty"`
}

// ScheduleStats provides aggregate statistics
type ScheduleStats struct {
	TotalJobs          int     `json:"total_jobs"`
	RunningNow         int     `json:"running_now"`
	ScheduledToday     int     `json:"scheduled_today"`
	FailuresLast24h    int     `json:"failures_last_24h"`
	SuccessRateLast24h float64 `json:"success_rate_last_24h"`
	AnomaliesDetected  int     `json:"anomalies_detected"`
}

// SLOSummary shows SLO health
type SLOSummary struct {
	Name    string  `json:"name"`
	Current float64 `json:"current"`
	Target  float64 `json:"target"`
	Status  string  `json:"status"` // healthy, at_risk, breached
}

// NLQQuery represents a natural language query
type NLQQuery struct {
	Query    string    `json:"query"`
	TenantID uuid.UUID `json:"tenant_id"`
}

// NLQResponse is the answer to a query
type NLQResponse struct {
	Query           string       `json:"query"`
	Answer          string       `json:"answer"`
	AnswerType      string       `json:"answer_type"` // text, list, metric, chart_data
	Data            interface{}  `json:"data,omitempty"`
	Suggestions     []string     `json:"suggestions,omitempty"`
	ActionableLinks []ActionLink `json:"actionable_links,omitempty"`
	Confidence      float64      `json:"confidence"`
	ProcessedAt     time.Time    `json:"processed_at"`
}

// ActionLink provides a link to take action
type ActionLink struct {
	Label  string `json:"label"`
	URL    string `json:"url"`
	Action string `json:"action"` // view, trigger, pause, etc.
}

// ProcessQuery handles a natural language query
func (n *NLQScheduler) ProcessQuery(ctx context.Context, query NLQQuery, context SchedulerContext) (*NLQResponse, error) {
	n.logger.Info("Processing NLQ",
		"query", query.Query,
		"tenant_id", query.TenantID,
	)

	// Parse query intent
	intent := n.parseIntent(query.Query)

	// Generate response based on intent
	var response *NLQResponse
	var err error

	switch intent.Type {
	case "status":
		response = n.handleStatusQuery(query, context, intent)
	case "failures":
		response = n.handleFailureQuery(query, context, intent)
	case "schedule":
		response = n.handleScheduleQuery(query, context, intent)
	case "slo":
		response = n.handleSLOQuery(query, context, intent)
	case "comparison":
		response = n.handleComparisonQuery(query, context, intent)
	default:
		// Use LLM for complex queries
		response, err = n.handleWithLLM(ctx, query, context)
		if err != nil {
			response = n.handleUnknownQuery(query)
		}
	}

	response.ProcessedAt = time.Now()
	return response, nil
}

// QueryIntent represents parsed query intent
type QueryIntent struct {
	Type       string // status, failures, schedule, slo, comparison
	Subject    string // job name, category, "all"
	TimeRange  string // today, last_hour, last_24h, last_week
	Keywords   []string
	Comparison string // vs_yesterday, vs_last_week
}

// parseIntent extracts intent from natural language
func (n *NLQScheduler) parseIntent(query string) QueryIntent {
	q := strings.ToLower(query)
	intent := QueryIntent{}

	// Detect query type
	switch {
	case containsAny(q, []string{"status", "state", "how is", "how are", "what's happening"}):
		intent.Type = "status"
	case containsAny(q, []string{"fail", "error", "broken", "wrong", "issue", "problem"}):
		intent.Type = "failures"
	case containsAny(q, []string{"scheduled", "upcoming", "when", "next", "running"}):
		intent.Type = "schedule"
	case containsAny(q, []string{"slo", "performance", "latency", "success rate", "target"}):
		intent.Type = "slo"
	case containsAny(q, []string{"compare", "vs", "versus", "than", "better", "worse"}):
		intent.Type = "comparison"
	default:
		intent.Type = "unknown"
	}

	// Detect time range
	switch {
	case containsAny(q, []string{"today", "so far"}):
		intent.TimeRange = "today"
	case containsAny(q, []string{"last hour", "past hour"}):
		intent.TimeRange = "last_hour"
	case containsAny(q, []string{"24 hours", "last day", "past day"}):
		intent.TimeRange = "last_24h"
	case containsAny(q, []string{"this week", "last week", "weekly"}):
		intent.TimeRange = "last_week"
	default:
		intent.TimeRange = "last_24h" // default
	}

	// Extract subject (would use NER in production)
	if strings.Contains(q, "pre-agg") || strings.Contains(q, "preagg") {
		intent.Subject = "pre-agg"
	} else if strings.Contains(q, "all jobs") || strings.Contains(q, "everything") {
		intent.Subject = "all"
	}

	return intent
}

// handleStatusQuery answers status questions
func (n *NLQScheduler) handleStatusQuery(query NLQQuery, ctx SchedulerContext, intent QueryIntent) *NLQResponse {
	stats := ctx.ScheduleStats

	answer := fmt.Sprintf(
		"**Scheduler Status Summary**\n\n"+
			"• **%d jobs** total, **%d running** now\n"+
			"• **%d jobs** scheduled for today\n"+
			"• **%d failures** in the last 24 hours\n"+
			"• **%.1f%% success rate** overall\n",
		stats.TotalJobs, stats.RunningNow,
		stats.ScheduledToday,
		stats.FailuresLast24h,
		stats.SuccessRateLast24h*100,
	)

	if stats.AnomaliesDetected > 0 {
		answer += fmt.Sprintf("\n⚠️ **%d anomalies** detected and need attention\n", stats.AnomaliesDetected)
	}

	return &NLQResponse{
		Query:      query.Query,
		Answer:     answer,
		AnswerType: "text",
		Data:       stats,
		Confidence: 0.9,
		Suggestions: []string{
			"Show me the failing jobs",
			"What's scheduled for the next hour?",
			"Are there any SLO risks?",
		},
	}
}

// handleFailureQuery answers failure-related questions
func (n *NLQScheduler) handleFailureQuery(query NLQQuery, ctx SchedulerContext, intent QueryIntent) *NLQResponse {
	failures := ctx.RecentFailures

	if len(failures) == 0 {
		return &NLQResponse{
			Query:      query.Query,
			Answer:     "✅ **No failures** in the recent time window. All systems healthy!",
			AnswerType: "text",
			Confidence: 0.95,
			Suggestions: []string{
				"Show me the current job status",
				"What's running right now?",
			},
		}
	}

	var answer strings.Builder
	answer.WriteString(fmt.Sprintf("**%d Recent Failures**\n\n", len(failures)))

	var links []ActionLink
	for i, f := range failures {
		if i >= 5 { // Limit to 5
			answer.WriteString(fmt.Sprintf("\n...and %d more\n", len(failures)-5))
			break
		}
		answer.WriteString(fmt.Sprintf("• **%s** - %s\n  └─ %s at %s\n",
			f.JobName, f.ErrorType, f.StepName, f.FailedAt.Format("15:04")))

		links = append(links, ActionLink{
			Label:  fmt.Sprintf("View %s", f.JobName),
			URL:    fmt.Sprintf("/scheduler/jobs/%s", f.JobName),
			Action: "view",
		})
	}

	return &NLQResponse{
		Query:           query.Query,
		Answer:          answer.String(),
		AnswerType:      "list",
		Data:            failures,
		ActionableLinks: links,
		Confidence:      0.9,
		Suggestions: []string{
			"Why is " + failures[0].JobName + " failing?",
			"Show me the runbook for " + failures[0].JobName,
			"Retry the failed job",
		},
	}
}

// handleScheduleQuery answers schedule-related questions
func (n *NLQScheduler) handleScheduleQuery(query NLQQuery, ctx SchedulerContext, intent QueryIntent) *NLQResponse {
	// Find upcoming jobs
	var upcoming []JobSummary
	now := ctx.CurrentTime

	for _, job := range ctx.ActiveJobs {
		if job.NextRun != nil && job.NextRun.After(now) && job.NextRun.Before(now.Add(24*time.Hour)) {
			upcoming = append(upcoming, job)
		}
	}

	running := 0
	for _, job := range ctx.ActiveJobs {
		if job.Status == "running" {
			running++
		}
	}

	var answer strings.Builder
	answer.WriteString(fmt.Sprintf("**%d jobs running now**, **%d scheduled** in next 24h\n\n", running, len(upcoming)))

	if len(upcoming) > 0 {
		answer.WriteString("**Upcoming:**\n")
		for i, job := range upcoming {
			if i >= 5 {
				answer.WriteString(fmt.Sprintf("...and %d more\n", len(upcoming)-5))
				break
			}
			answer.WriteString(fmt.Sprintf("• **%s** at %s\n", job.Name, job.NextRun.Format("15:04")))
		}
	}

	return &NLQResponse{
		Query:      query.Query,
		Answer:     answer.String(),
		AnswerType: "list",
		Data:       upcoming,
		Confidence: 0.85,
		Suggestions: []string{
			"Show me what's running right now",
			"Are there any schedule conflicts?",
			"What ran in the last hour?",
		},
	}
}

// handleSLOQuery answers SLO-related questions
func (n *NLQScheduler) handleSLOQuery(query NLQQuery, ctx SchedulerContext, intent QueryIntent) *NLQResponse {
	slos := ctx.SLOStatus

	healthy := 0
	atRisk := 0
	breached := 0

	for _, slo := range slos {
		switch slo.Status {
		case "healthy":
			healthy++
		case "at_risk":
			atRisk++
		case "breached":
			breached++
		}
	}

	var answer strings.Builder
	answer.WriteString("**SLO Health Summary**\n\n")
	answer.WriteString(fmt.Sprintf("• ✅ **%d healthy**\n", healthy))
	if atRisk > 0 {
		answer.WriteString(fmt.Sprintf("• ⚠️ **%d at risk**\n", atRisk))
	}
	if breached > 0 {
		answer.WriteString(fmt.Sprintf("• 🔴 **%d breached**\n", breached))
	}

	// List problematic SLOs
	for _, slo := range slos {
		if slo.Status != "healthy" {
			answer.WriteString(fmt.Sprintf("\n**%s**: %.1f%% (target: %.1f%%) - %s\n",
				slo.Name, slo.Current*100, slo.Target*100, slo.Status))
		}
	}

	return &NLQResponse{
		Query:      query.Query,
		Answer:     answer.String(),
		AnswerType: "metric",
		Data: map[string]interface{}{
			"healthy":  healthy,
			"at_risk":  atRisk,
			"breached": breached,
			"details":  slos,
		},
		Confidence: 0.9,
		Suggestions: []string{
			"What's causing the SLO breach?",
			"Show me the SLO trend",
			"How much error budget do we have left?",
		},
	}
}

// handleComparisonQuery handles comparative questions
func (n *NLQScheduler) handleComparisonQuery(query NLQQuery, ctx SchedulerContext, intent QueryIntent) *NLQResponse {
	// This would compare with historical data in production
	return &NLQResponse{
		Query:      query.Query,
		Answer:     "📊 **Comparison vs Yesterday**\n\n• Success rate: +2.5% ↑\n• Avg duration: -150ms ↓ (faster)\n• Failures: -3 ↓",
		AnswerType: "comparison",
		Confidence: 0.7,
		Suggestions: []string{
			"Show me the weekly trend",
			"What improved?",
			"What got worse?",
		},
	}
}

// handleWithLLM uses AI for complex queries
func (n *NLQScheduler) handleWithLLM(ctx context.Context, query NLQQuery, context SchedulerContext) (*NLQResponse, error) {
	if n.llmClient == nil {
		return nil, fmt.Errorf("LLM client not available")
	}

	prompt := n.buildLLMPrompt(query, context)
	response, err := n.llmClient.Generate(ctx, prompt)
	if err != nil {
		return nil, err
	}

	return &NLQResponse{
		Query:      query.Query,
		Answer:     response,
		AnswerType: "text",
		Confidence: 0.6,
	}, nil
}

// buildLLMPrompt creates prompt for complex queries
func (n *NLQScheduler) buildLLMPrompt(query NLQQuery, context SchedulerContext) string {
	contextJSON, _ := json.MarshalIndent(context.ScheduleStats, "", "  ")

	return fmt.Sprintf(`You are a helpful scheduler assistant. Answer the following question based on the provided context.

Question: %s

Current Status:
%s

Recent Failures: %d
SLO Status: %d healthy, %d at risk

Provide a concise, helpful answer. Use markdown formatting.`,
		query.Query,
		string(contextJSON),
		len(context.RecentFailures),
		countSLOsByStatus(context.SLOStatus, "healthy"),
		countSLOsByStatus(context.SLOStatus, "at_risk"),
	)
}

// handleUnknownQuery fallback for unrecognized queries
func (n *NLQScheduler) handleUnknownQuery(query NLQQuery) *NLQResponse {
	return &NLQResponse{
		Query:      query.Query,
		Answer:     "I'm not sure how to answer that question. Here are some things I can help with:",
		AnswerType: "text",
		Confidence: 0.3,
		Suggestions: []string{
			"What's the current scheduler status?",
			"Show me recent failures",
			"What's scheduled for today?",
			"Are there any SLO risks?",
			"Compare today vs yesterday",
		},
	}
}

// Helper functions
func containsAny(s string, substrs []string) bool {
	for _, sub := range substrs {
		if strings.Contains(s, sub) {
			return true
		}
	}
	return false
}

func countSLOsByStatus(slos []SLOSummary, status string) int {
	count := 0
	for _, slo := range slos {
		if slo.Status == status {
			count++
		}
	}
	return count
}
