package chat_history

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type SessionRecord struct {
	ID              uuid.UUID       `json:"id"`
	TenantID        uuid.UUID       `json:"tenant_id"`
	ConversationID  uuid.UUID       `json:"conversation_id"`
	AgentID         string          `json:"agent_id"`
	AgentVersion    *string         `json:"agent_version,omitempty"`
	UserID          string          `json:"user_id"`
	UserEmail       *string         `json:"user_email,omitempty"`
	ViewType        string          `json:"view_type"`
	Embedded        bool            `json:"embedded"`
	EmbedSurface    *string         `json:"embed_surface,omitempty"`
	StartedAt       time.Time       `json:"started_at"`
	EndedAt         *time.Time      `json:"ended_at,omitempty"`
	MessageCount    int             `json:"message_count"`
	FeedbackScore   *int16          `json:"feedback_score,omitempty"`
	FeedbackComment *string         `json:"feedback_comment,omitempty"`
	Metadata        json.RawMessage `json:"metadata"`
	TraceID         *string         `json:"trace_id,omitempty"`
	FirstMessage    *string         `json:"first_message,omitempty"`
	LastMessage     *string         `json:"last_message,omitempty"`
	PreviousHash    *string         `json:"previous_hash,omitempty"`
	CurrentHash     *string         `json:"current_hash,omitempty"`
	CreatedAt       time.Time       `json:"created_at"`
	UpdatedAt       time.Time       `json:"updated_at"`
}

type MessageRecord struct {
	ID          uuid.UUID       `json:"id"`
	SessionID   uuid.UUID       `json:"session_id"`
	TenantID    uuid.UUID       `json:"tenant_id"`
	Seq         int             `json:"seq"`
	Role        string          `json:"role"`
	Content     string          `json:"content"`
	ContentJSON json.RawMessage `json:"content_json,omitempty"`
	ToolCalls   json.RawMessage `json:"tool_calls,omitempty"`
	ChartSpec   json.RawMessage `json:"chart_spec,omitempty"`
	TraceSteps  json.RawMessage `json:"trace_steps,omitempty"`
	LatencyMs   *int            `json:"latency_ms,omitempty"`
	TokenIn     *int            `json:"token_in,omitempty"`
	TokenOut    *int            `json:"token_out,omitempty"`
	TraceID     *string         `json:"trace_id,omitempty"`
	SpanID      *string         `json:"span_id,omitempty"`
	Error       *string         `json:"error,omitempty"`
	CreatedAt   time.Time       `json:"created_at"`
}

type TraceStep struct {
	Name       string          `json:"name"`
	DurationMs int             `json:"duration_ms"`
	Inputs     json.RawMessage `json:"inputs,omitempty"`
	Outputs    json.RawMessage `json:"outputs,omitempty"`
}

type ListFilters struct {
	TenantID   *uuid.UUID
	AllTenants bool
	AgentID    *string
	ViewType   *string
	Embedded   *bool
	Feedback   *string
	Search     *string
	FromDate   *time.Time
	ToDate     *time.Time
	Limit      int
	Offset     int
	SkipCount  bool
}

type SessionDetail struct {
	Session  SessionRecord   `json:"session"`
	Messages []MessageRecord `json:"messages"`
}