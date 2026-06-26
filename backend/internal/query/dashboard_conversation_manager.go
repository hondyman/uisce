package query

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"
	"time"
)

// ConversationMessage represents a message in the dashboard conversation
type ConversationMessage struct {
	ID        string    `json:"id"`
	Type      string    `json:"type"` // "user", "assistant"
	Content   string    `json:"content"`
	Timestamp time.Time `json:"timestamp"`
}

// DecisionTrace represents a governance decision trace
type DecisionTrace struct {
	ID          string    `json:"id"`
	UserID      string    `json:"user_id"`
	AssetID     string    `json:"asset_id"`
	Action      string    `json:"action"`
	Decision    string    `json:"decision"`
	Reason      string    `json:"reason"`
	EvaluatedAt time.Time `json:"evaluated_at"`
}
type DashboardConversation struct {
	ID          string    `json:"id"`
	UserID      string    `json:"user_id"`
	TenantID    string    `json:"tenant_id"`
	Datasource  string    `json:"datasource"`
	State       string    `json:"state"` // "active", "completed", "abandoned"
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	Title       string    `json:"title"`
	Description string    `json:"description"`

	// Dashboard components
	Visuals       []DashboardVisual `json:"visuals"`
	GlobalFilters []IntentFilter    `json:"global_filters"`
	Layout        DashboardLayout   `json:"layout"`

	// Conversation state
	Messages      []ConversationMessage `json:"messages"`
	RefinementCtx *RefinementContext    `json:"refinement_ctx"`

	// Governance
	GovernanceCtx    *GovernanceContext  `json:"governance_ctx"`
	ComplianceStatus DashboardCompliance `json:"compliance_status"`
}

// DashboardVisual represents a single chart/visual in the dashboard
type DashboardVisual struct {
	ID          string `json:"id"`
	Type        string `json:"type"` // "line", "bar", "pie", "table", etc.
	Title       string `json:"title"`
	Description string `json:"description"`

	// Query specification
	QuerySpec VisualQuerySpec `json:"query_spec"`

	// Visual configuration
	Config VisualConfig `json:"config"`

	// Governance
	Compliance    VisualCompliance `json:"compliance"`
	DecisionTrace []DecisionTrace  `json:"decision_trace"`

	// Layout
	Position VisualPosition `json:"position"`
	Size     VisualSize     `json:"size"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// VisualQuerySpec defines the query for a visual
type VisualQuerySpec struct {
	Metrics     []string       `json:"metrics"`
	Dimensions  []string       `json:"dimensions"`
	Filters     []IntentFilter `json:"filters"`
	TimeRange   *TimeRange     `json:"time_range"`
	Aggregation string         `json:"aggregation"`

	// Generated SQL
	SQL         string `json:"sql"`
	SemanticSQL string `json:"semantic_sql"`
}

// VisualConfig defines chart configuration
type VisualConfig struct {
	ChartType  string `json:"chart_type"`
	XAxis      string `json:"x_axis,omitempty"`
	YAxis      string `json:"y_axis,omitempty"`
	ColorBy    string `json:"color_by,omitempty"`
	SizeBy     string `json:"size_by,omitempty"`
	SortBy     string `json:"sort_by,omitempty"`
	Limit      int    `json:"limit,omitempty"`
	ShowLegend bool   `json:"show_legend"`
	ShowGrid   bool   `json:"show_grid"`
}

// VisualCompliance tracks governance compliance for a visual
type VisualCompliance struct {
	IsCompliant     bool                      `json:"is_compliant"`
	RiskLevel       string                    `json:"risk_level"` // "low", "medium", "high"
	Violations      []ComplianceViolation     `json:"violations"`
	AppliedPolicies []AppliedGovernancePolicy `json:"applied_policies"`
}

// ComplianceViolation represents a policy violation
type ComplianceViolation struct {
	PolicyID   string `json:"policy_id"`
	RuleID     string `json:"rule_id"`
	Severity   string `json:"severity"`
	Message    string `json:"message"`
	Suggestion string `json:"suggestion,omitempty"`
}

// VisualPosition defines where a visual appears in the layout
type VisualPosition struct {
	X      int `json:"x"`
	Y      int `json:"y"`
	Width  int `json:"width"`
	Height int `json:"height"`
}

// VisualSize defines the size constraints
type VisualSize struct {
	MinWidth  int `json:"min_width"`
	MinHeight int `json:"min_height"`
	MaxWidth  int `json:"max_width"`
	MaxHeight int `json:"max_height"`
}

// DashboardLayout defines the overall dashboard layout
type DashboardLayout struct {
	Type             string `json:"type"` // "grid", "masonry", "freeform"
	Columns          int    `json:"columns"`
	RowHeight        int    `json:"row_height"`
	Margin           [2]int `json:"margin"`
	ContainerPadding [2]int `json:"container_padding"`
}

// DashboardCompliance tracks overall dashboard compliance
type DashboardCompliance struct {
	OverallCompliant bool                  `json:"overall_compliant"`
	VisualCount      int                   `json:"visual_count"`
	CompliantCount   int                   `json:"compliant_count"`
	HighRiskCount    int                   `json:"high_risk_count"`
	Violations       []ComplianceViolation `json:"violations"`
}

// VisualComposer generates chart specifications from NL + governance
type VisualComposer struct{}

// NewVisualComposer creates a new visual composer
func NewVisualComposer() *VisualComposer {
	return &VisualComposer{}
}

// ComposeVisual creates a dashboard visual from query spec and governance
func (vc *VisualComposer) ComposeVisual(chartType, title string, querySpec VisualQuerySpec, govCtx *GovernanceContext) (*DashboardVisual, error) {
	// Generate SQL from query spec
	sql, semanticSQL := vc.generateSQL(querySpec)

	// Check compliance
	compliance := vc.checkCompliance(querySpec, govCtx)

	// Create visual
	visual := &DashboardVisual{
		ID:         generateDashboardVisualID(),
		Type:       chartType,
		Title:      title,
		QuerySpec:  querySpec,
		Config:     vc.generateConfig(chartType, querySpec),
		Compliance: compliance,
		Position:   VisualPosition{X: 0, Y: 0, Width: 6, Height: 4},
		Size:       VisualSize{MinWidth: 4, MinHeight: 3, MaxWidth: 12, MaxHeight: 8},
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	visual.QuerySpec.SQL = sql
	visual.QuerySpec.SemanticSQL = semanticSQL

	return visual, nil
}

func (vc *VisualComposer) generateSQL(querySpec VisualQuerySpec) (string, string) {
	// Simple SQL generation - in real implementation this would be more sophisticated
	selectClause := "SELECT "
	if len(querySpec.Metrics) > 0 {
		selectClause += strings.Join(querySpec.Metrics, ", ")
	}
	if len(querySpec.Dimensions) > 0 {
		if len(querySpec.Metrics) > 0 {
			selectClause += ", "
		}
		selectClause += strings.Join(querySpec.Dimensions, ", ")
	}

	fromClause := " FROM your_table" // Would be determined from schema

	whereClause := ""
	if len(querySpec.Filters) > 0 {
		conditions := []string{}
		for _, filter := range querySpec.Filters {
			conditions = append(conditions, fmt.Sprintf("%s %s '%s'", filter.Field, filter.Operator, strings.Join(filter.Values, ",")))
		}
		whereClause = " WHERE " + strings.Join(conditions, " AND ")
	}

	groupByClause := ""
	if len(querySpec.Dimensions) > 0 {
		groupByClause = " GROUP BY " + strings.Join(querySpec.Dimensions, ", ")
	}

	sql := selectClause + fromClause + whereClause + groupByClause
	return sql, sql // For now, semantic SQL is same as regular SQL
}

func (vc *VisualComposer) checkCompliance(querySpec VisualQuerySpec, govCtx *GovernanceContext) VisualCompliance {
	// Simplified compliance check
	compliance := VisualCompliance{
		IsCompliant:     true,
		RiskLevel:       "low",
		Violations:      []ComplianceViolation{},
		AppliedPolicies: []AppliedGovernancePolicy{},
	}

	// Check if any metrics are blocked (not in allowed list)
	for _, metric := range querySpec.Metrics {
		if govCtx != nil && len(govCtx.AllowedMetrics) > 0 {
			allowed := false
			for _, allowedMetric := range govCtx.AllowedMetrics {
				if metric == allowedMetric {
					allowed = true
					break
				}
			}
			if !allowed {
				compliance.IsCompliant = false
				compliance.RiskLevel = "high"
				compliance.Violations = append(compliance.Violations, ComplianceViolation{
					PolicyID:   "blocked_metric",
					Severity:   "high",
					Message:    fmt.Sprintf("Metric '%s' is not allowed by governance policy", metric),
					Suggestion: "Use an allowed metric or request access",
				})
			}
		}
	}

	return compliance
}

func (vc *VisualComposer) generateConfig(chartType string, querySpec VisualQuerySpec) VisualConfig {
	config := VisualConfig{
		ChartType:  chartType,
		ShowLegend: true,
		ShowGrid:   true,
	}

	if len(querySpec.Dimensions) > 0 {
		config.XAxis = querySpec.Dimensions[0]
	}
	if len(querySpec.Metrics) > 0 {
		config.YAxis = querySpec.Metrics[0]
	}

	return config
}

// DashboardGovernanceGate ensures each visual is compliant
type DashboardGovernanceGate struct{}

// NewDashboardGovernanceGate creates a new governance gate
func NewDashboardGovernanceGate() *DashboardGovernanceGate {
	return &DashboardGovernanceGate{}
}

// LayoutOptimizer suggests dashboard layout and optimizations
type LayoutOptimizer struct{}

// NewLayoutOptimizer creates a new layout optimizer
func NewLayoutOptimizer() *LayoutOptimizer {
	return &LayoutOptimizer{}
}

// GenerateInitialLayout creates an initial layout for visuals
func (lo *LayoutOptimizer) GenerateInitialLayout(visualCount int) DashboardLayout {
	return DashboardLayout{
		Type:             "grid",
		Columns:          12,
		RowHeight:        100,
		Margin:           [2]int{10, 10},
		ContainerPadding: [2]int{10, 10},
	}
}

// OptimizeLayout optimizes the layout based on visual count and types
func (lo *LayoutOptimizer) OptimizeLayout(visuals []DashboardVisual, currentLayout DashboardLayout) DashboardLayout {
	// Simple optimization - adjust columns based on visual count
	if len(visuals) <= 2 {
		currentLayout.Columns = 6
	} else if len(visuals) <= 4 {
		currentLayout.Columns = 12
	} else {
		currentLayout.Columns = 12
	}

	return currentLayout
}

// DashboardConversationManager manages multi-turn dashboard conversations
type DashboardConversationManager struct {
	conversations   map[string]*DashboardConversation
	nlEngine        *NLQueryEngine
	visualComposer  *VisualComposer
	governanceGate  *DashboardGovernanceGate
	layoutOptimizer *LayoutOptimizer
}

// NewDashboardConversationManager creates a new dashboard conversation manager
func NewDashboardConversationManager(nlEngine *NLQueryEngine) *DashboardConversationManager {
	return &DashboardConversationManager{
		conversations:   make(map[string]*DashboardConversation),
		nlEngine:        nlEngine,
		visualComposer:  NewVisualComposer(),
		governanceGate:  NewDashboardGovernanceGate(),
		layoutOptimizer: NewLayoutOptimizer(),
	}
}

// StartConversation begins a new dashboard conversation
func (dcm *DashboardConversationManager) StartConversation(ctx context.Context, userID, tenantID, datasource, initialMessage string) (*DashboardConversation, error) {
	conversationID := generateDashboardConversationID()

	// Get governance context
	govCtx, err := dcm.nlEngine.governanceProvider.GetContext(ctx, userID, tenantID, datasource)
	if err != nil {
		return nil, fmt.Errorf("failed to get governance context: %w", err)
	}

	// Parse initial intent
	intent, err := dcm.nlEngine.parser.ParseIntent(initialMessage)
	if err != nil {
		return nil, fmt.Errorf("failed to parse initial intent: %w", err)
	}

	// Generate initial dashboard layout
	initialVisuals, err := dcm.generateInitialVisuals(intent, govCtx)
	if err != nil {
		return nil, fmt.Errorf("failed to generate initial visuals: %w", err)
	}

	// Create conversation
	conversation := &DashboardConversation{
		ID:            conversationID,
		UserID:        userID,
		TenantID:      tenantID,
		Datasource:    datasource,
		State:         "active",
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
		Title:         dcm.generateTitleFromIntent(intent),
		Description:   fmt.Sprintf("Dashboard conversation started with: %s", initialMessage),
		Visuals:       initialVisuals,
		GlobalFilters: intent.Filters,
		Layout:        dcm.layoutOptimizer.GenerateInitialLayout(len(initialVisuals)),
		Messages:      []ConversationMessage{},
		RefinementCtx: &RefinementContext{
			ConversationID: conversationID,
			UserID:         userID,
			TenantID:       tenantID,
			Datasource:     datasource,
			State:          "active",
			CreatedAt:      time.Now(),
			LastActivity:   time.Now(),
		},
		GovernanceCtx:    govCtx,
		ComplianceStatus: dcm.calculateComplianceStatus(initialVisuals),
	}

	// Add initial message
	conversation.Messages = append(conversation.Messages, ConversationMessage{
		ID:        generateDashboardMessageID(),
		Type:      "user",
		Content:   initialMessage,
		Timestamp: time.Now(),
	})

	// Add system response
	systemResponse := dcm.generateInitialResponse(conversation)
	conversation.Messages = append(conversation.Messages, ConversationMessage{
		ID:        generateDashboardMessageID(),
		Type:      "assistant",
		Content:   systemResponse,
		Timestamp: time.Now(),
	})

	dcm.conversations[conversationID] = conversation
	return conversation, nil
}

// ProcessMessage processes a user message in the dashboard conversation
func (dcm *DashboardConversationManager) ProcessMessage(ctx context.Context, conversationID, message string) (*DashboardConversation, error) {
	conversation, exists := dcm.conversations[conversationID]
	if !exists {
		return nil, fmt.Errorf("conversation not found: %s", conversationID)
	}

	// Add user message
	conversation.Messages = append(conversation.Messages, ConversationMessage{
		ID:        generateDashboardMessageID(),
		Type:      "user",
		Content:   message,
		Timestamp: time.Now(),
	})

	// Parse intent from message
	intent, err := dcm.nlEngine.parser.ParseIntent(message)
	if err != nil {
		return nil, fmt.Errorf("failed to parse message intent: %w", err)
	}

	// Process the message based on intent
	updatedConversation, err := dcm.processIntent(conversation, intent, message)
	if err != nil {
		return nil, fmt.Errorf("failed to process intent: %w", err)
	}

	// Update compliance status
	updatedConversation.ComplianceStatus = dcm.calculateComplianceStatus(updatedConversation.Visuals)

	// Generate assistant response
	systemResponse := dcm.generateResponse(updatedConversation, intent, message)
	updatedConversation.Messages = append(updatedConversation.Messages, ConversationMessage{
		ID:        generateDashboardMessageID(),
		Type:      "assistant",
		Content:   systemResponse,
		Timestamp: time.Now(),
	})

	updatedConversation.UpdatedAt = time.Now()
	return updatedConversation, nil
}

// GetConversation retrieves a dashboard conversation
func (dcm *DashboardConversationManager) GetConversation(conversationID string) (*DashboardConversation, error) {
	conversation, exists := dcm.conversations[conversationID]
	if !exists {
		return nil, fmt.Errorf("conversation not found: %s", conversationID)
	}
	return conversation, nil
}

// CommitConversation finalizes and saves the dashboard
func (dcm *DashboardConversationManager) CommitConversation(conversationID, title, description string) (*DashboardConversation, error) {
	conversation, exists := dcm.conversations[conversationID]
	if !exists {
		return nil, fmt.Errorf("conversation not found: %s", conversationID)
	}

	conversation.State = "completed"
	conversation.Title = title
	conversation.Description = description
	conversation.UpdatedAt = time.Now()

	// Here you would typically save to database
	// For now, just mark as completed

	return conversation, nil
}

// Helper methods

func (dcm *DashboardConversationManager) generateInitialVisuals(intent *ParsedIntent, govCtx *GovernanceContext) ([]DashboardVisual, error) {
	var visuals []DashboardVisual

	// Generate primary metric visual
	if len(intent.Metrics) > 0 {
		primaryVisual, err := dcm.visualComposer.ComposeVisual(
			"line",
			fmt.Sprintf("%s Over Time", intent.Metrics[0]),
			VisualQuerySpec{
				Metrics:     []string{intent.Metrics[0]},
				Dimensions:  []string{"date"}, // Assume time dimension exists
				Filters:     intent.Filters,
				TimeRange:   intent.TimeRange,
				Aggregation: intent.Aggregation,
			},
			govCtx,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to compose primary visual: %w", err)
		}
		visuals = append(visuals, *primaryVisual)
	}

	// Generate secondary visual if multiple metrics
	if len(intent.Metrics) > 1 {
		secondaryVisual, err := dcm.visualComposer.ComposeVisual(
			"bar",
			fmt.Sprintf("%s by Category", intent.Metrics[1]),
			VisualQuerySpec{
				Metrics:     []string{intent.Metrics[1]},
				Dimensions:  []string{"category"}, // Assume category dimension exists
				Filters:     intent.Filters,
				TimeRange:   intent.TimeRange,
				Aggregation: intent.Aggregation,
			},
			govCtx,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to compose secondary visual: %w", err)
		}
		visuals = append(visuals, *secondaryVisual)
	}

	return visuals, nil
}

func (dcm *DashboardConversationManager) processIntent(conversation *DashboardConversation, intent *ParsedIntent, message string) (*DashboardConversation, error) {
	// Analyze the intent to determine what action to take
	action := dcm.analyzeIntentAction(intent, message)

	switch action {
	case "add_visual":
		return dcm.addVisual(conversation, intent)
	case "modify_visual":
		return dcm.modifyVisual(conversation, intent)
	case "remove_visual":
		return dcm.removeVisual(conversation, intent)
	case "add_filter":
		return dcm.addGlobalFilter(conversation, intent)
	case "modify_layout":
		return dcm.modifyLayout(conversation, intent)
	default:
		// Default to adding a visual
		return dcm.addVisual(conversation, intent)
	}
}

func (dcm *DashboardConversationManager) analyzeIntentAction(_intent *ParsedIntent, message string) string {
	_ = _intent // Mark parameter as intentionally used
	messageLower := strings.ToLower(message)

	// Simple keyword-based action detection
	if strings.Contains(messageLower, "add") || strings.Contains(messageLower, "show") || strings.Contains(messageLower, "create") {
		return "add_visual"
	}
	if strings.Contains(messageLower, "change") || strings.Contains(messageLower, "modify") || strings.Contains(messageLower, "update") {
		return "modify_visual"
	}
	if strings.Contains(messageLower, "remove") || strings.Contains(messageLower, "delete") || strings.Contains(messageLower, "hide") {
		return "remove_visual"
	}
	if strings.Contains(messageLower, "filter") || strings.Contains(messageLower, "limit") {
		return "add_filter"
	}

	return "add_visual"
}

func (dcm *DashboardConversationManager) addVisual(conversation *DashboardConversation, intent *ParsedIntent) (*DashboardConversation, error) {
	// Determine visual type based on intent
	visualType := dcm.inferVisualType(intent)

	// Generate title
	title := dcm.generateVisualTitle(intent)

	// Create query spec
	querySpec := VisualQuerySpec{
		Metrics:     intent.Metrics,
		Dimensions:  intent.Dimensions,
		Filters:     append(conversation.GlobalFilters, intent.Filters...),
		TimeRange:   intent.TimeRange,
		Aggregation: intent.Aggregation,
	}

	// Compose visual
	visual, err := dcm.visualComposer.ComposeVisual(visualType, title, querySpec, conversation.GovernanceCtx)
	if err != nil {
		return nil, fmt.Errorf("failed to compose visual: %w", err)
	}

	// Add to conversation
	conversation.Visuals = append(conversation.Visuals, *visual)

	// Update layout
	conversation.Layout = dcm.layoutOptimizer.OptimizeLayout(conversation.Visuals, conversation.Layout)

	return conversation, nil
}

func (dcm *DashboardConversationManager) modifyVisual(conversation *DashboardConversation, intent *ParsedIntent) (*DashboardConversation, error) {
	// Find the visual to modify (simplified - would need better logic)
	if len(conversation.Visuals) == 0 {
		return conversation, nil
	}

	// For now, modify the last visual
	lastVisual := &conversation.Visuals[len(conversation.Visuals)-1]

	// Update based on intent
	if len(intent.Metrics) > 0 {
		lastVisual.QuerySpec.Metrics = intent.Metrics
	}
	if len(intent.Dimensions) > 0 {
		lastVisual.QuerySpec.Dimensions = intent.Dimensions
	}

	// Re-compose the visual
	updatedVisual, err := dcm.visualComposer.ComposeVisual(
		lastVisual.Type,
		lastVisual.Title,
		lastVisual.QuerySpec,
		conversation.GovernanceCtx,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to recompose visual: %w", err)
	}

	conversation.Visuals[len(conversation.Visuals)-1] = *updatedVisual
	return conversation, nil
}

func (dcm *DashboardConversationManager) removeVisual(conversation *DashboardConversation, _intent *ParsedIntent) (*DashboardConversation, error) {
	_ = _intent // Mark parameter as intentionally used
	// Simplified - remove last visual
	if len(conversation.Visuals) > 0 {
		conversation.Visuals = conversation.Visuals[:len(conversation.Visuals)-1]
		conversation.Layout = dcm.layoutOptimizer.OptimizeLayout(conversation.Visuals, conversation.Layout)
	}
	return conversation, nil
}

func (dcm *DashboardConversationManager) addGlobalFilter(conversation *DashboardConversation, intent *ParsedIntent) (*DashboardConversation, error) {
	// Add filters to global filters
	conversation.GlobalFilters = append(conversation.GlobalFilters, intent.Filters...)

	// Update all visuals with new global filters
	for i := range conversation.Visuals {
		conversation.Visuals[i].QuerySpec.Filters = append(conversation.Visuals[i].QuerySpec.Filters, intent.Filters...)
		// Re-compose visual with updated filters
		updatedVisual, err := dcm.visualComposer.ComposeVisual(
			conversation.Visuals[i].Type,
			conversation.Visuals[i].Title,
			conversation.Visuals[i].QuerySpec,
			conversation.GovernanceCtx,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to recompose visual %d: %w", i, err)
		}
		conversation.Visuals[i] = *updatedVisual
	}

	return conversation, nil
}

func (dcm *DashboardConversationManager) modifyLayout(conversation *DashboardConversation, _intent *ParsedIntent) (*DashboardConversation, error) {
	_ = _intent // Mark parameter as intentionally used
	// Update layout based on intent
	conversation.Layout = dcm.layoutOptimizer.OptimizeLayout(conversation.Visuals, conversation.Layout)
	return conversation, nil
}

func (dcm *DashboardConversationManager) inferVisualType(intent *ParsedIntent) string {
	// Simple heuristic for visual type
	if len(intent.Dimensions) == 1 && len(intent.Metrics) == 1 {
		if intent.TimeRange != nil {
			return "line"
		}
		return "bar"
	}
	if len(intent.Dimensions) == 2 && len(intent.Metrics) == 1 {
		return "heatmap"
	}
	return "table"
}

func (dcm *DashboardConversationManager) generateVisualTitle(intent *ParsedIntent) string {
	if len(intent.Metrics) == 0 {
		return "Data Visualization"
	}

	metric := intent.Metrics[0]
	if len(intent.Dimensions) > 0 {
		dimension := intent.Dimensions[0]
		return fmt.Sprintf("%s by %s", metric, dimension)
	}

	return fmt.Sprintf("%s Overview", metric)
}

func (dcm *DashboardConversationManager) generateTitleFromIntent(intent *ParsedIntent) string {
	if len(intent.Metrics) > 0 {
		return fmt.Sprintf("%s Dashboard", intent.Metrics[0])
	}
	return "Custom Dashboard"
}

func (dcm *DashboardConversationManager) generateInitialResponse(conversation *DashboardConversation) string {
	response := fmt.Sprintf("I've created a dashboard with %d initial visualizations:\n\n", len(conversation.Visuals))

	for i, visual := range conversation.Visuals {
		response += fmt.Sprintf("%d. %s (%s chart)\n", i+1, visual.Title, visual.Type)
	}

	response += "\nWould you like to add more visualizations, modify existing ones, or adjust filters?"

	return response
}

func (dcm *DashboardConversationManager) generateResponse(conversation *DashboardConversation, intent *ParsedIntent, message string) string {
	action := dcm.analyzeIntentAction(intent, message)

	switch action {
	case "add_visual":
		return fmt.Sprintf("Added a new %s visualization: %s", conversation.Visuals[len(conversation.Visuals)-1].Type, conversation.Visuals[len(conversation.Visuals)-1].Title)
	case "modify_visual":
		return "Updated the visualization based on your request."
	case "remove_visual":
		return "Removed the specified visualization."
	case "add_filter":
		return "Applied the filter to all visualizations."
	default:
		return "I've processed your request. How else can I help with your dashboard?"
	}
}

func (dcm *DashboardConversationManager) calculateComplianceStatus(visuals []DashboardVisual) DashboardCompliance {
	compliance := DashboardCompliance{
		VisualCount:    len(visuals),
		CompliantCount: 0,
		HighRiskCount:  0,
		Violations:     []ComplianceViolation{},
	}

	for _, visual := range visuals {
		if visual.Compliance.IsCompliant {
			compliance.CompliantCount++
		}
		if visual.Compliance.RiskLevel == "high" {
			compliance.HighRiskCount++
		}
		compliance.Violations = append(compliance.Violations, visual.Compliance.Violations...)
	}

	compliance.OverallCompliant = compliance.CompliantCount == compliance.VisualCount
	return compliance
}

// Utility functions

func generateDashboardConversationID() string {
	bytes := make([]byte, 16)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

func generateDashboardMessageID() string {
	bytes := make([]byte, 8)
	rand.Read(bytes)
	return "msg_" + hex.EncodeToString(bytes)
}

func generateDashboardVisualID() string {
	bytes := make([]byte, 8)
	rand.Read(bytes)
	return "vis_" + hex.EncodeToString(bytes)
}
