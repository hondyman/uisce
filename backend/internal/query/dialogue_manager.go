package query

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/hondyman/semlayer/backend/internal/domain"
)

// DialogueState represents the current state of a refinement conversation
type DialogueState string

const (
	StateInitial    DialogueState = "initial"
	StateClarifying DialogueState = "clarifying"
	StateSuggesting DialogueState = "suggesting"
	StateRefining   DialogueState = "refining"
	StateReady      DialogueState = "ready"
	StateError      DialogueState = "error"
)

// RefinementMessage represents a message in the conversation
type RefinementMessage struct {
	ID          string                 `json:"id"`
	Type        string                 `json:"type"` // "user", "system", "clarification", "suggestion"
	Content     string                 `json:"content"`
	Timestamp   time.Time              `json:"timestamp"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	Suggestions []RefinementSuggestion `json:"suggestions,omitempty"`
}

// RefinementSuggestion represents a suggestion for query refinement
type RefinementSuggestion struct {
	ID          string      `json:"id"`
	Type        string      `json:"type"` // "metric", "dimension", "filter", "time_range", "aggregation"
	Description string      `json:"description"`
	Action      string      `json:"action"` // "replace", "add", "remove", "clarify"
	Value       interface{} `json:"value,omitempty"`
	Reason      string      `json:"reason,omitempty"`
	Confidence  float64     `json:"confidence"`
}

// ClarificationQuestion represents a question to clarify user intent
type ClarificationQuestion struct {
	ID       string   `json:"id"`
	Question string   `json:"question"`
	Options  []string `json:"options,omitempty"`
	Field    string   `json:"field"`
	Required bool     `json:"required"`
}

// RefinementContext represents the current context of query refinement
type RefinementContext struct {
	ConversationID    string                  `json:"conversation_id"`
	UserID            string                  `json:"user_id"`
	TenantID          string                  `json:"tenant_id"`
	Datasource        string                  `json:"datasource"`
	State             DialogueState           `json:"state"`
	OriginalQuery     string                  `json:"original_query"`
	CurrentIntent     *ParsedIntent           `json:"current_intent"`
	CurrentQuery      *GeneratedQuery         `json:"current_query"`
	GovernanceCtx     *GovernanceContext      `json:"governance_ctx"`
	Messages          []*RefinementMessage    `json:"messages"`
	PendingQuestions  []ClarificationQuestion `json:"pending_questions"`
	RefinementHistory []*RefinementStep       `json:"refinement_history"`
	LastActivity      time.Time               `json:"last_activity"`
	CreatedAt         time.Time               `json:"created_at"`
}

// RefinementStep represents a step in the refinement process
type RefinementStep struct {
	ID          string      `json:"id"`
	Type        string      `json:"type"`
	Description string      `json:"description"`
	Before      interface{} `json:"before,omitempty"`
	After       interface{} `json:"after,omitempty"`
	Timestamp   time.Time   `json:"timestamp"`
	UserAction  string      `json:"user_action,omitempty"`
}

// DialogueManager manages conversational query refinement
type DialogueManager struct {
	conversationMgr     *ConversationManager
	schemaProvider      domain.SchemaProvider
	governanceProvider  *GovernanceContextProvider
	clarificationEngine *ClarificationEngine
	suggestionEngine    *SuggestionEngine
	refinementEngine    *RefinementEngine
}

// NewDialogueManager creates a new dialogue manager
func NewDialogueManager(
	conversationMgr *ConversationManager,
	schemaProvider domain.SchemaProvider,
	governanceProvider *GovernanceContextProvider,
) *DialogueManager {
	return &DialogueManager{
		conversationMgr:     conversationMgr,
		schemaProvider:      schemaProvider,
		governanceProvider:  governanceProvider,
		clarificationEngine: NewClarificationEngine(),
		suggestionEngine:    NewSuggestionEngine(),
		refinementEngine:    NewRefinementEngine(),
	}
}

// StartRefinement starts a new query refinement conversation
func (dm *DialogueManager) StartRefinement(ctx context.Context, req *NLQueryRequest) (*RefinementContext, error) {
	// Start or get existing conversation
	var conversationID string
	if req.ConversationID != "" {
		// Use existing conversation
		conversationID = req.ConversationID
	} else {
		// Start new conversation
		conversation := dm.conversationMgr.StartConversation(req.UserID, req.TenantID, req.Datasource)
		conversationID = conversation.ConversationID
	}

	// Create refinement context
	refinementCtx := &RefinementContext{
		ConversationID:    conversationID,
		UserID:            req.UserID,
		TenantID:          req.TenantID,
		Datasource:        req.Datasource,
		State:             StateInitial,
		OriginalQuery:     req.Text,
		Messages:          []*RefinementMessage{},
		PendingQuestions:  []ClarificationQuestion{},
		RefinementHistory: []*RefinementStep{},
		LastActivity:      time.Now(),
		CreatedAt:         time.Now(),
	}

	// Add initial user message
	initialMessage := &RefinementMessage{
		ID:        generateMessageID(),
		Type:      "user",
		Content:   req.Text,
		Timestamp: time.Now(),
	}
	refinementCtx.Messages = append(refinementCtx.Messages, initialMessage)

	// Process initial query
	err := dm.processInitialQuery(ctx, refinementCtx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to process initial query: %w", err)
	}

	return refinementCtx, nil
}

// ProcessUserResponse processes a user response in the refinement conversation
func (dm *DialogueManager) ProcessUserResponse(ctx context.Context, conversationID, userResponse string) (*RefinementContext, error) {
	// Get current refinement context (this would be stored in a database/cache in production)
	refinementCtx, err := dm.GetRefinementContext(conversationID)
	if err != nil {
		return nil, fmt.Errorf("failed to get refinement context: %w", err)
	}

	// Add user response message
	userMessage := &RefinementMessage{
		ID:        generateMessageID(),
		Type:      "user",
		Content:   userResponse,
		Timestamp: time.Now(),
	}
	refinementCtx.Messages = append(refinementCtx.Messages, userMessage)

	// Process user response based on current state
	switch refinementCtx.State {
	case StateClarifying:
		err = dm.processClarificationResponse(ctx, refinementCtx, userResponse)
	case StateSuggesting:
		err = dm.processSuggestionResponse(ctx, refinementCtx, userResponse)
	case StateRefining:
		err = dm.processRefinementResponse(ctx, refinementCtx, userResponse)
	default:
		err = dm.processGeneralResponse(ctx, refinementCtx, userResponse)
	}

	if err != nil {
		refinementCtx.State = StateError
		return refinementCtx, err
	}

	refinementCtx.LastActivity = time.Now()
	return refinementCtx, nil
}

// processInitialQuery processes the initial user query
func (dm *DialogueManager) processInitialQuery(ctx context.Context, refinementCtx *RefinementContext, req *NLQueryRequest) error {
	// Parse initial intent
	intent, err := dm.parseIntent(req.Text)
	if err != nil {
		return fmt.Errorf("failed to parse intent: %w", err)
	}
	refinementCtx.CurrentIntent = intent

	// Get governance context
	govCtx, err := dm.governanceProvider.GetContext(ctx, req.UserID, req.TenantID, req.Datasource)
	if err != nil {
		return fmt.Errorf("failed to get governance context: %w", err)
	}
	refinementCtx.GovernanceCtx = govCtx

	// Generate initial query
	query, err := dm.generateInitialQuery(intent, govCtx)
	if err != nil {
		return fmt.Errorf("failed to generate initial query: %w", err)
	}
	refinementCtx.CurrentQuery = query

	// Check for ambiguities and governance issues
	questions := dm.clarificationEngine.DetectAmbiguities(intent, govCtx)
	suggestions := dm.suggestionEngine.GenerateSuggestions(intent, govCtx, query)

	if len(questions) > 0 {
		refinementCtx.State = StateClarifying
		refinementCtx.PendingQuestions = questions
		dm.addSystemMessage(refinementCtx, "clarification", dm.formatClarificationPrompt(questions))
	} else if len(suggestions) > 0 {
		refinementCtx.State = StateSuggesting
		dm.addSystemMessageWithSuggestions(refinementCtx, "suggestion", "Here are some suggestions to improve your query:", suggestions)
	} else {
		refinementCtx.State = StateReady
		dm.addSystemMessage(refinementCtx, "ready", "Your query is ready to execute!")
	}

	return nil
}

// processClarificationResponse processes a response to clarification questions
func (dm *DialogueManager) processClarificationResponse(_ context.Context, refinementCtx *RefinementContext, response string) error {
	// Parse user response and update intent
	updatedIntent, err := dm.clarificationEngine.ProcessClarificationResponse(refinementCtx.CurrentIntent, refinementCtx.PendingQuestions, response)
	if err != nil {
		return fmt.Errorf("failed to process clarification response: %w", err)
	}

	// Record refinement step
	step := &RefinementStep{
		ID:          generateStepID(),
		Type:        "clarification",
		Description: "Applied clarification responses",
		Before:      refinementCtx.CurrentIntent,
		After:       updatedIntent,
		Timestamp:   time.Now(),
		UserAction:  response,
	}
	refinementCtx.RefinementHistory = append(refinementCtx.RefinementHistory, step)

	refinementCtx.CurrentIntent = updatedIntent
	refinementCtx.PendingQuestions = []ClarificationQuestion{}

	// Regenerate query with updated intent
	query, err := dm.refinementEngine.RegenerateQuery(updatedIntent, refinementCtx.GovernanceCtx)
	if err != nil {
		return fmt.Errorf("failed to regenerate query: %w", err)
	}
	refinementCtx.CurrentQuery = query

	// Check for new suggestions
	suggestions := dm.suggestionEngine.GenerateSuggestions(updatedIntent, refinementCtx.GovernanceCtx, query)
	if len(suggestions) > 0 {
		refinementCtx.State = StateSuggesting
		dm.addSystemMessageWithSuggestions(refinementCtx, "suggestion", "Great! Here are some additional suggestions:", suggestions)
	} else {
		refinementCtx.State = StateReady
		dm.addSystemMessage(refinementCtx, "ready", "Perfect! Your query is now ready to execute.")
	}

	return nil
}

// processSuggestionResponse processes a response to suggestions
func (dm *DialogueManager) processSuggestionResponse(_ context.Context, refinementCtx *RefinementContext, response string) error {
	// Parse user response and apply selected suggestions
	updatedIntent, updatedQuery, err := dm.suggestionEngine.ProcessSuggestionResponse(
		refinementCtx.CurrentIntent,
		refinementCtx.CurrentQuery,
		refinementCtx.Messages[len(refinementCtx.Messages)-2].Suggestions, // Get suggestions from previous message
		response,
	)
	if err != nil {
		return fmt.Errorf("failed to process suggestion response: %w", err)
	}

	// Record refinement step
	step := &RefinementStep{
		ID:          generateStepID(),
		Type:        "suggestion",
		Description: "Applied user suggestions",
		Before:      map[string]interface{}{"intent": refinementCtx.CurrentIntent, "query": refinementCtx.CurrentQuery},
		After:       map[string]interface{}{"intent": updatedIntent, "query": updatedQuery},
		Timestamp:   time.Now(),
		UserAction:  response,
	}
	refinementCtx.RefinementHistory = append(refinementCtx.RefinementHistory, step)

	refinementCtx.CurrentIntent = updatedIntent
	refinementCtx.CurrentQuery = updatedQuery

	refinementCtx.State = StateReady
	dm.addSystemMessage(refinementCtx, "ready", "Excellent! Your query has been refined and is ready to execute.")

	return nil
}

// processRefinementResponse processes a general refinement response
func (dm *DialogueManager) processRefinementResponse(_ context.Context, refinementCtx *RefinementContext, response string) error {
	// Parse general refinement request
	updatedIntent, err := dm.parseRefinementRequest(refinementCtx.CurrentIntent, response)
	if err != nil {
		return fmt.Errorf("failed to parse refinement request: %w", err)
	}

	// Regenerate query
	query, err := dm.refinementEngine.RegenerateQuery(updatedIntent, refinementCtx.GovernanceCtx)
	if err != nil {
		return fmt.Errorf("failed to regenerate query: %w", err)
	}

	// Record refinement step
	step := &RefinementStep{
		ID:          generateStepID(),
		Type:        "refinement",
		Description: "Applied general refinement",
		Before:      refinementCtx.CurrentIntent,
		After:       updatedIntent,
		Timestamp:   time.Now(),
		UserAction:  response,
	}
	refinementCtx.RefinementHistory = append(refinementCtx.RefinementHistory, step)

	refinementCtx.CurrentIntent = updatedIntent
	refinementCtx.CurrentQuery = query
	refinementCtx.State = StateReady

	dm.addSystemMessage(refinementCtx, "ready", "Query refined successfully!")

	return nil
}

// processGeneralResponse handles general user responses
func (dm *DialogueManager) processGeneralResponse(ctx context.Context, refinementCtx *RefinementContext, response string) error {
	// Try to interpret as a refinement request
	return dm.processRefinementResponse(ctx, refinementCtx, response)
}

// Helper methods

func (dm *DialogueManager) addSystemMessage(refinementCtx *RefinementContext, msgType, content string) {
	message := &RefinementMessage{
		ID:        generateMessageID(),
		Type:      msgType,
		Content:   content,
		Timestamp: time.Now(),
	}
	refinementCtx.Messages = append(refinementCtx.Messages, message)
}

func (dm *DialogueManager) addSystemMessageWithSuggestions(refinementCtx *RefinementContext, msgType, content string, suggestions []RefinementSuggestion) {
	message := &RefinementMessage{
		ID:          generateMessageID(),
		Type:        msgType,
		Content:     content,
		Timestamp:   time.Now(),
		Suggestions: suggestions,
	}
	refinementCtx.Messages = append(refinementCtx.Messages, message)
}

func (dm *DialogueManager) formatClarificationPrompt(questions []ClarificationQuestion) string {
	var prompt strings.Builder
	prompt.WriteString("I need to clarify a few things to generate the best query for you:\n\n")

	for i, q := range questions {
		prompt.WriteString(fmt.Sprintf("%d. %s\n", i+1, q.Question))
		if len(q.Options) > 0 {
			prompt.WriteString("   Options: " + strings.Join(q.Options, ", ") + "\n")
		}
	}

	prompt.WriteString("\nPlease provide your answers, and I'll refine the query accordingly.")
	return prompt.String()
}

func (dm *DialogueManager) parseIntent(text string) (*ParsedIntent, error) {
	// Use existing intent parser
	parser := NewIntentParser()
	return parser.ParseIntent(text)
}

func (dm *DialogueManager) generateInitialQuery(intent *ParsedIntent, govCtx *GovernanceContext) (*GeneratedQuery, error) {
	// Use existing generation engine
	engine := NewGenerationEngine()
	skeleton, err := engine.GenerateQuerySkeleton(intent, govCtx)
	if err != nil {
		return nil, err
	}

	sql, err := engine.GenerateSQL(skeleton, govCtx)
	if err != nil {
		return nil, err
	}

	return &GeneratedQuery{
		SQL:         sql,
		SemanticSQL: skeleton.SemanticSQL,
		Measures:    skeleton.Measures,
		Dimensions:  skeleton.Dimensions,
		Filters:     skeleton.Filters,
		OrderBy:     skeleton.OrderBy,
	}, nil
}

func (dm *DialogueManager) parseRefinementRequest(currentIntent *ParsedIntent, _ string) (*ParsedIntent, error) {
	// Simple parsing - in production this would be more sophisticated
	// For now, just return the current intent (placeholder implementation)
	return currentIntent, nil
}

// GetRefinementContext retrieves the current refinement context
// In production, this would be stored in a database
func (dm *DialogueManager) GetRefinementContext(conversationID string) (*RefinementContext, error) {
	// Placeholder - in production this would retrieve from database/cache
	return nil, fmt.Errorf("refinement context storage not implemented")
}

// Utility functions

func generateMessageID() string {
	return fmt.Sprintf("msg_%d", time.Now().UnixNano())
}

func generateStepID() string {
	return fmt.Sprintf("step_%d", time.Now().UnixNano())
}
