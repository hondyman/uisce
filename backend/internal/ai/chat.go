package ai

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	"github.com/hondyman/uisce/backend/internal/chat_history"
)

// ChatQuery represents a user's natural language question
type ChatQuery struct {
	Text           string    `json:"text"`
	TenantID       uuid.UUID `json:"tenant_id"`
	ConversationID uuid.UUID `json:"conversation_id"`
	UserID         string    `json:"user_id"`
	UserEmail      string    `json:"user_email"`
	ViewType       string    `json:"view_type"`
	Embedded       bool      `json:"embedded"`
	EmbedSurface   string    `json:"embed_surface"`
	AgentID        string    `json:"agent_id"`
}

// ChatResponse represents the AI's answer with semantic grounding
type ChatResponse struct {
	Response         string                   `json:"response"`
	SemanticEntities []map[string]interface{} `json:"semantic_entities"`
	Confidence       int                      `json:"confidence"`
	Explainability   map[string]interface{}   `json:"explainability"`
	ChartSpec        json.RawMessage          `json:"chart_spec,omitempty"`
	TraceSteps       []chat_history.TraceStep `json:"trace_steps,omitempty"`
	ToolCalls        json.RawMessage          `json:"tool_calls,omitempty"`
	LatencyMs        int                      `json:"latency_ms"`
	TokenIn          int                      `json:"token_in"`
	TokenOut         int                      `json:"token_out"`
	TraceID          string                   `json:"trace_id"`
	SpanID           string                   `json:"span_id"`
}

// ChatEngine processes natural language queries within the semantic graph
type ChatEngine struct {
	db          *sql.DB
	llmProvider interface{}
	historySvc  *chat_history.Service
	tracer      trace.Tracer
}

// NewChatEngine creates a new semantic chat engine. If historySvc is nil,
// session/message persistence is silently skipped (callers can use the
// engine without chat history).
func NewChatEngine(db *sql.DB, llmProvider interface{}, historySvc *chat_history.Service) *ChatEngine {
	return &ChatEngine{
		db:          db,
		llmProvider: llmProvider,
		historySvc:  historySvc,
		tracer:      otel.Tracer("uisce.ai.chat"),
	}
}

// ProcessQuery handles natural language understanding and graph traversal.
// It persists the conversation to ai_telemetry.chat_session + chat_message
// when historySvc is wired, capturing trace_id/span_id and trace_steps.
func (e *ChatEngine) ProcessQuery(ctx context.Context, query ChatQuery) (*ChatResponse, error) {
	ctx, span := e.tracer.Start(ctx, "chat.process_query",
		trace.WithAttributes(
			attribute.String("tenant.id", query.TenantID.String()),
			attribute.String("conversation.id", query.ConversationID.String()),
			attribute.String("agent.id", defaultAgentID(query.AgentID)),
		),
	)
	defer span.End()

	spanCtx := span.SpanContext()
	traceID := ""
	spanID := ""
	if spanCtx.IsValid() {
		traceID = spanCtx.TraceID().String()
		spanID = spanCtx.SpanID().String()
	}

	sessionID := uuid.Nil
	if e.historySvc != nil && query.ConversationID != uuid.Nil && query.TenantID != uuid.Nil {
		var err error
		sessionID, err = e.historySvc.EnsureSession(
			ctx, query.TenantID, query.ConversationID,
			defaultAgentID(query.AgentID),
			query.UserID, nullableString(query.UserEmail),
			normalizeViewType(query.ViewType),
			query.Embedded, nullableString(query.EmbedSurface),
		)
		if err != nil {
			span.RecordError(err)
			return nil, fmt.Errorf("ensure chat session: %w", err)
		}

		if traceID != "" {
			if err := e.historySvc.StampSessionTrace(ctx, query.TenantID, sessionID, traceID); err != nil {
				span.RecordError(err)
			}
		}

		if _, err := e.historySvc.AppendMessageWithTrace(ctx, query.TenantID, sessionID,
			chat_history.MessageRecord{
				Role:    "user",
				Content: query.Text,
			}, traceID, spanID); err != nil {
			span.RecordError(err)
		}
	}

	resp := &ChatResponse{
		Response:   "Searching the semantic graph for your query...",
		Confidence: 90,
		TraceID:    traceID,
		SpanID:     spanID,
	}

	if e.historySvc != nil && sessionID != uuid.Nil {
		var chartSpec, toolCalls json.RawMessage
		if len(resp.ChartSpec) > 0 {
			chartSpec = resp.ChartSpec
		}
		if len(resp.ToolCalls) > 0 {
			toolCalls = resp.ToolCalls
		}
		var steps json.RawMessage
		if len(resp.TraceSteps) > 0 {
			if b, err := json.Marshal(resp.TraceSteps); err == nil {
				steps = b
			}
		}
		latency := resp.LatencyMs
		tokenIn := resp.TokenIn
		tokenOut := resp.TokenOut
		if _, err := e.historySvc.AppendMessageWithTrace(ctx, query.TenantID, sessionID,
			chat_history.MessageRecord{
				Role:       "assistant",
				Content:    resp.Response,
				ChartSpec:  chartSpec,
				ToolCalls:  toolCalls,
				TraceSteps:  steps,
				LatencyMs:  &latency,
				TokenIn:    &tokenIn,
				TokenOut:   &tokenOut,
			}, traceID, spanID); err != nil {
			span.RecordError(err)
		}
	}

	return resp, nil
}

func defaultAgentID(provided string) string {
	if provided == "" {
		return "default-semantic-analyst"
	}
	return provided
}

func normalizeViewType(v string) string {
	if v == "admin" {
		return "admin"
	}
	return "end_user"
}

func nullableString(s string) *string {
	if s == "" {
		return nil
	}
	out := s
	return &out
}