package query

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/hondyman/semlayer/backend/internal/domain"
)

// NLQueryRequest represents a natural language query request
type NLQueryRequest struct {
	Text           string            `json:"text"`
	UserID         string            `json:"user_id"`
	TenantID       string            `json:"tenant_id"`
	Datasource     string            `json:"datasource"`
	ConversationID string            `json:"conversation_id,omitempty"` // For multi-turn conversations
	Context        map[string]string `json:"context,omitempty"`
}

// NLQueryResponse represents the response from NL query processing
type NLQueryResponse struct {
	OriginalText    string          `json:"original_text"`
	ParsedIntent    *ParsedIntent   `json:"parsed_intent"`
	GeneratedQuery  *GeneratedQuery `json:"generated_query"`
	GovernanceDiff  *GovernanceDiff `json:"governance_diff"`
	ComplianceNotes []string        `json:"compliance_notes"`
	Warnings        []string        `json:"warnings"`
	QueryID         string          `json:"query_id"`
	Timestamp       time.Time       `json:"timestamp"`
}

// ParsedIntent represents the extracted intent from natural language
type ParsedIntent struct {
	Metrics     []string          `json:"metrics"`
	Dimensions  []string          `json:"dimensions"`
	Filters     []IntentFilter    `json:"filters"`
	TimeRange   *TimeRange        `json:"time_range,omitempty"`
	Aggregation string            `json:"aggregation,omitempty"`
	Confidence  float64           `json:"confidence"`
	RawEntities map[string]string `json:"raw_entities"`
}

// IntentFilter represents a filter extracted from NL
type IntentFilter struct {
	Field    string   `json:"field"`
	Operator string   `json:"operator"`
	Values   []string `json:"values"`
}

// TimeRange represents a time range extracted from NL
type TimeRange struct {
	Start string `json:"start"`
	End   string `json:"end"`
	Label string `json:"label"`
}

// GeneratedQuery represents the generated compliant query
type GeneratedQuery struct {
	SQL         string        `json:"sql"`
	SemanticSQL string        `json:"semantic_sql"`
	Measures    []string      `json:"measures"`
	Dimensions  []string      `json:"dimensions"`
	Filters     []QueryFilter `json:"filters"`
	OrderBy     []OrderBySpec `json:"order_by,omitempty"`
}

// QueryFilter represents a filter in the generated query
type QueryFilter struct {
	Field    string `json:"field"`
	Operator string `json:"operator"`
	Value    string `json:"value"`
}

// OrderBySpec represents an ORDER BY clause
type OrderBySpec struct {
	Field string `json:"field"`
	Dir   string `json:"dir"`
}

// GovernanceDiff shows what governance rules were applied
type GovernanceDiff struct {
	BlockedMetrics    []string        `json:"blocked_metrics,omitempty"`
	BlockedDimensions []string        `json:"blocked_dimensions,omitempty"`
	AddedFilters      []QueryFilter   `json:"added_filters,omitempty"`
	RemovedFilters    []QueryFilter   `json:"removed_filters,omitempty"`
	AppliedPolicies   []AppliedPolicy `json:"applied_policies,omitempty"`
}

// AppliedPolicy shows which policy was applied and why
type AppliedPolicy struct {
	PolicyID string `json:"policy_id"`
	RuleID   string `json:"rule_id"`
	Action   string `json:"action"`
	Reason   string `json:"reason"`
}

// NLQueryEngine handles natural language to query generation
type NLQueryEngine struct {
	schemaProvider     domain.SchemaProvider
	governanceProvider *GovernanceContextProvider
	generationEngine   *GenerationEngine
	parser             *IntentParser
	conversationMgr    *ConversationManager
	dialogueMgr        *DialogueManager
}

// NewNLQueryEngine creates a new NL query engine
func NewNLQueryEngine(schemaProvider domain.SchemaProvider, governanceProvider *GovernanceContextProvider) *NLQueryEngine {
	return &NLQueryEngine{
		schemaProvider:     schemaProvider,
		governanceProvider: governanceProvider,
		generationEngine:   NewGenerationEngine(),
		parser:             NewIntentParser(),
		conversationMgr:    NewConversationManager(),
		dialogueMgr:        NewDialogueManager(NewConversationManager(), schemaProvider, governanceProvider),
	}
}

// ProcessNLQuery processes a natural language query request
func (nle *NLQueryEngine) ProcessNLQuery(ctx context.Context, req *NLQueryRequest) (*NLQueryResponse, error) {
	queryID := generateQueryID()

	// Step 1: Parse intent from natural language
	intent, err := nle.parser.ParseIntent(req.Text)
	if err != nil {
		return nil, fmt.Errorf("failed to parse intent: %w", err)
	}

	// Step 2: Enhance intent with conversation context if available
	if req.ConversationID != "" {
		intent = nle.conversationMgr.EnhanceIntentWithContext(req.ConversationID, intent)
	}

	// Step 3: Get governance context
	govCtx, err := nle.governanceProvider.GetContext(ctx, req.UserID, req.TenantID, req.Datasource)
	if err != nil {
		return nil, fmt.Errorf("failed to get governance context: %w", err)
	}

	// Step 4: Generate initial query skeleton
	initialQuery, err := nle.generationEngine.GenerateQuerySkeleton(intent, govCtx)
	if err != nil {
		return nil, fmt.Errorf("failed to generate query skeleton: %w", err)
	}

	// Step 5: Apply governance compliance
	compliantQuery, govDiff, err := nle.applyGovernanceCompliance(initialQuery, govCtx)
	if err != nil {
		return nil, fmt.Errorf("failed to apply governance compliance: %w", err)
	}

	// Step 6: Generate final SQL
	finalSQL, err := nle.generationEngine.GenerateSQL(compliantQuery, govCtx)
	if err != nil {
		return nil, fmt.Errorf("failed to generate SQL: %w", err)
	}

	// Step 7: Generate compliance notes and warnings
	complianceNotes := nle.generateComplianceNotes(govDiff)
	warnings := nle.generateWarnings(intent, govDiff)

	response := &NLQueryResponse{
		OriginalText: req.Text,
		ParsedIntent: intent,
		GeneratedQuery: &GeneratedQuery{
			SQL:         finalSQL,
			SemanticSQL: compliantQuery.SemanticSQL,
			Measures:    compliantQuery.Measures,
			Dimensions:  compliantQuery.Dimensions,
			Filters:     compliantQuery.Filters,
			OrderBy:     compliantQuery.OrderBy,
		},
		GovernanceDiff:  govDiff,
		ComplianceNotes: complianceNotes,
		Warnings:        warnings,
		QueryID:         queryID,
		Timestamp:       time.Now(),
	}

	// Step 8: Add query to conversation if conversation ID is provided
	if req.ConversationID != "" {
		err = nle.conversationMgr.AddQueryToConversation(
			req.ConversationID,
			req.Text,
			intent,
			finalSQL,
			true, // Assume success for now
		)
		if err != nil {
			// Log error but don't fail the response
			response.Warnings = append(response.Warnings, fmt.Sprintf("Failed to update conversation: %v", err))
		}
	}

	return response, nil
}

// SimulateNLQuery simulates the NL query generation without executing
func (nle *NLQueryEngine) SimulateNLQuery(ctx context.Context, req *NLQueryRequest) (*NLQueryResponse, error) {
	// Same as ProcessNLQuery but marks as simulation
	response, err := nle.ProcessNLQuery(ctx, req)
	if err != nil {
		return nil, err
	}

	// Add simulation note
	response.ComplianceNotes = append(response.ComplianceNotes, "This is a simulation - query not executed")

	return response, nil
}

// applyGovernanceCompliance applies governance rules to the generated query
func (nle *NLQueryEngine) applyGovernanceCompliance(query *QuerySkeleton, govCtx *GovernanceContext) (*QuerySkeleton, *GovernanceDiff, error) {
	govDiff := &GovernanceDiff{
		BlockedMetrics:    []string{},
		BlockedDimensions: []string{},
		AddedFilters:      []QueryFilter{},
		RemovedFilters:    []QueryFilter{},
		AppliedPolicies:   []AppliedPolicy{},
	}

	// Filter out blocked metrics
	allowedMetrics := []string{}
	for _, metric := range query.Measures {
		if nle.isMetricAllowed(metric, govCtx) {
			allowedMetrics = append(allowedMetrics, metric)
		} else {
			govDiff.BlockedMetrics = append(govDiff.BlockedMetrics, metric)
		}
	}
	query.Measures = allowedMetrics

	// Filter out blocked dimensions
	allowedDimensions := []string{}
	for _, dim := range query.Dimensions {
		if nle.isDimensionAllowed(dim, govCtx) {
			allowedDimensions = append(allowedDimensions, dim)
		} else {
			govDiff.BlockedDimensions = append(govDiff.BlockedDimensions, dim)
		}
	}
	query.Dimensions = allowedDimensions

	// Add required row-level filters
	for _, filter := range govCtx.RequiredFilters {
		query.Filters = append(query.Filters, filter)
		govDiff.AddedFilters = append(govDiff.AddedFilters, filter)
	}

	// Record applied policies
	for _, policy := range govCtx.AppliedPolicies {
		govDiff.AppliedPolicies = append(govDiff.AppliedPolicies, AppliedPolicy{
			PolicyID: policy.ID,
			RuleID:   policy.RuleID,
			Action:   policy.Action,
			Reason:   policy.Reason,
		})
	}

	return query, govDiff, nil
}

// isMetricAllowed checks if a metric is allowed by governance
func (nle *NLQueryEngine) isMetricAllowed(metric string, govCtx *GovernanceContext) bool {
	for _, allowed := range govCtx.AllowedMetrics {
		if allowed == metric {
			return true
		}
	}
	return false
}

// isDimensionAllowed checks if a dimension is allowed by governance
func (nle *NLQueryEngine) isDimensionAllowed(dimension string, govCtx *GovernanceContext) bool {
	for _, allowed := range govCtx.AllowedDimensions {
		if allowed == dimension {
			return true
		}
	}
	return false
}

// generateComplianceNotes generates human-readable compliance notes
func (nle *NLQueryEngine) generateComplianceNotes(govDiff *GovernanceDiff) []string {
	notes := []string{}

	if len(govDiff.BlockedMetrics) > 0 {
		notes = append(notes, fmt.Sprintf("Blocked %d metric(s) due to access restrictions: %s",
			len(govDiff.BlockedMetrics), strings.Join(govDiff.BlockedMetrics, ", ")))
	}

	if len(govDiff.BlockedDimensions) > 0 {
		notes = append(notes, fmt.Sprintf("Blocked %d dimension(s) due to access restrictions: %s",
			len(govDiff.BlockedDimensions), strings.Join(govDiff.BlockedDimensions, ", ")))
	}

	if len(govDiff.AddedFilters) > 0 {
		notes = append(notes, fmt.Sprintf("Added %d automatic filter(s) for data security",
			len(govDiff.AddedFilters)))
	}

	if len(govDiff.AppliedPolicies) > 0 {
		notes = append(notes, fmt.Sprintf("Applied %d governance policy rule(s)",
			len(govDiff.AppliedPolicies)))
	}

	return notes
}

// generateWarnings generates warnings for the user
func (nle *NLQueryEngine) generateWarnings(intent *ParsedIntent, govDiff *GovernanceDiff) []string {
	warnings := []string{}

	// Warn if many metrics were blocked
	if len(govDiff.BlockedMetrics) > 0 && len(govDiff.BlockedMetrics) > len(intent.Metrics)/2 {
		warnings = append(warnings, "Many requested metrics were blocked. Consider requesting access or using alternative metrics.")
	}

	// Warn about ambiguous terms
	if intent.Confidence < 0.7 {
		warnings = append(warnings, "Query interpretation confidence is low. Please review the generated query.")
	}

	return warnings
}

// StartConversation starts a new conversation for multi-turn queries
func (nle *NLQueryEngine) StartConversation(userID, tenantID, datasource string) *ConversationContext {
	return nle.conversationMgr.StartConversation(userID, tenantID, datasource)
}

// GetConversation retrieves an existing conversation
func (nle *NLQueryEngine) GetConversation(conversationID string) (*ConversationContext, error) {
	return nle.conversationMgr.GetConversation(conversationID)
}

// GetDialogueManager returns the dialogue manager for conversational query refinement
func (nle *NLQueryEngine) GetDialogueManager() *DialogueManager {
	return nle.dialogueMgr
}

// CleanupExpiredConversations removes expired conversations
func (nle *NLQueryEngine) CleanupExpiredConversations() int {
	return nle.conversationMgr.CleanupExpiredConversations()
}

// generateQueryID generates a unique query ID
func generateQueryID() string {
	return fmt.Sprintf("nlq_%d", time.Now().UnixNano())
}
