package chat_history

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/uuid"
)

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) EnsureSession(
	ctx context.Context,
	tenantID, conversationID uuid.UUID,
	agentID, userID string,
	userEmail *string,
	viewType string,
	embedded bool,
	embedSurface *string,
) (uuid.UUID, error) {
	if viewType == "" {
		viewType = "end_user"
	}

	query := `
		INSERT INTO ai_telemetry.chat_session (
			tenant_id, conversation_id, agent_id, user_id, user_email, view_type, embedded, embed_surface, started_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NOW())
		ON CONFLICT (tenant_id, conversation_id) DO UPDATE
		SET updated_at = NOW()
		RETURNING id;
	`
	var sessionID uuid.UUID
	err := r.db.QueryRowContext(ctx, query, tenantID, conversationID, agentID, userID, userEmail, viewType, embedded, embedSurface).Scan(&sessionID)
	return sessionID, err
}

// AppendMessage serializes concurrent writes to the same session via a
// pg_advisory_xact_lock scoped to the session UUID. The CTE then computes the
// next seq under the lock so the UNIQUE (session_id, seq) constraint cannot be
// violated by two concurrent inserts.
func (r *Repository) AppendMessage(ctx context.Context, tenantID, sessionID uuid.UUID, msg MessageRecord) (*MessageRecord, error) {
	if msg.ID == uuid.Nil {
		msg.ID = uuid.New()
	}
	msg.SessionID = sessionID
	msg.TenantID = tenantID

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx,
		`SELECT pg_advisory_xact_lock(hashtext('chat_history:' || $1::text))`,
		sessionID); err != nil {
		return nil, err
	}

	query := `
		WITH next_seq AS (
			SELECT COALESCE(MAX(seq), 0) + 1 AS val
			FROM ai_telemetry.chat_message
			WHERE session_id = $1
		),
		inserted AS (
			INSERT INTO ai_telemetry.chat_message (
				id, session_id, tenant_id, seq, role, content, content_json, tool_calls,
				chart_spec, trace_steps, latency_ms, token_in, token_out, trace_id, span_id, error, created_at
			)
			SELECT
				$2, $1, $3, next_seq.val, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, NOW()
			FROM next_seq
			RETURNING id, session_id, tenant_id, seq, role, content, content_json, tool_calls,
			          chart_spec, trace_steps, latency_ms, token_in, token_out, trace_id, span_id, error, created_at
		),
		updated_session AS (
			UPDATE ai_telemetry.chat_session
			SET message_count = message_count + 1, updated_at = NOW()
			WHERE id = $1 AND tenant_id = $3
		)
		SELECT id, session_id, tenant_id, seq, role, content, content_json, tool_calls,
		       chart_spec, trace_steps, latency_ms, token_in, token_out, trace_id, span_id, error, created_at
		FROM inserted;
	`

	var ret MessageRecord
	var cJSON, toolCalls, chartSpec, traceSteps []byte

	err = tx.QueryRowContext(
		ctx, query,
		sessionID, msg.ID, tenantID, msg.Role, msg.Content,
		msg.ContentJSON, msg.ToolCalls, msg.ChartSpec, msg.TraceSteps,
		msg.LatencyMs, msg.TokenIn, msg.TokenOut, msg.TraceID, msg.SpanID, msg.Error,
	).Scan(
		&ret.ID, &ret.SessionID, &ret.TenantID, &ret.Seq, &ret.Role, &ret.Content,
		&cJSON, &toolCalls, &chartSpec, &traceSteps,
		&ret.LatencyMs, &ret.TokenIn, &ret.TokenOut, &ret.TraceID, &ret.SpanID, &ret.Error, &ret.CreatedAt,
	)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	if len(cJSON) > 0 {
		ret.ContentJSON = json.RawMessage(cJSON)
	}
	if len(toolCalls) > 0 {
		ret.ToolCalls = json.RawMessage(toolCalls)
	}
	if len(chartSpec) > 0 {
		ret.ChartSpec = json.RawMessage(chartSpec)
	}
	if len(traceSteps) > 0 {
		ret.TraceSteps = json.RawMessage(traceSteps)
	}

	return &ret, nil
}

// SetFeedback applies feedback only when the caller is allowed to mutate the
// session row. Per-tenant callers must be the original session creator;
// global admins may set feedback for any session.
func (r *Repository) SetFeedback(ctx context.Context, tenantID, sessionID uuid.UUID, userID string, isGlobalAdmin bool, score int16, comment *string) error {
	args := []interface{}{score, comment, sessionID, tenantID}
	// $1=score, $2=comment, $3=sessionID, $4=tenantID
	where := "WHERE id = $3 AND tenant_id = $4"
	if !isGlobalAdmin {
		where += " AND user_id = $5"
		args = append(args, userID)
	}

	query := fmt.Sprintf(`
		UPDATE ai_telemetry.chat_session
		SET feedback_score = $1, feedback_comment = $2, updated_at = NOW()
		%s;
	`, where)

	res, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return err
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrSessionNotFound
	}
	return nil
}

func (r *Repository) EndSession(ctx context.Context, tenantID, sessionID uuid.UUID) error {
	query := `
		UPDATE ai_telemetry.chat_session
		SET ended_at = NOW(), updated_at = NOW()
		WHERE id = $1 AND tenant_id = $2 AND ended_at IS NULL;
	`
	_, err := r.db.ExecContext(ctx, query, sessionID, tenantID)
	return err
}

func (r *Repository) ListSessions(ctx context.Context, f ListFilters) ([]SessionRecord, int, error) {
	var whereClauses []string
	var args []interface{}
	idx := 1

	if f.TenantID != nil {
		whereClauses = append(whereClauses, fmt.Sprintf("s.tenant_id = $%d", idx))
		args = append(args, *f.TenantID)
		idx++
	}
	if f.AgentID != nil {
		whereClauses = append(whereClauses, fmt.Sprintf("s.agent_id = $%d", idx))
		args = append(args, *f.AgentID)
		idx++
	}
	if f.ViewType != nil {
		whereClauses = append(whereClauses, fmt.Sprintf("s.view_type = $%d", idx))
		args = append(args, *f.ViewType)
		idx++
	}
	if f.Embedded != nil {
		whereClauses = append(whereClauses, fmt.Sprintf("s.embedded = $%d", idx))
		args = append(args, *f.Embedded)
		idx++
	}
	if f.Feedback != nil {
		switch *f.Feedback {
		case "positive":
			whereClauses = append(whereClauses, "s.feedback_score = 1")
		case "negative":
			whereClauses = append(whereClauses, "s.feedback_score = -1")
		case "unrated":
			whereClauses = append(whereClauses, "s.feedback_score IS NULL")
		}
	}
	if f.Search != nil && strings.TrimSpace(*f.Search) != "" {
		whereClauses = append(whereClauses, fmt.Sprintf(`EXISTS (
			SELECT 1 FROM ai_telemetry.chat_message m
			WHERE m.session_id = s.id AND m.tenant_id = s.tenant_id AND m.content ILIKE $%d
		)`, idx))
		args = append(args, "%"+strings.TrimSpace(*f.Search)+"%")
		idx++
	}
	if f.FromDate != nil {
		whereClauses = append(whereClauses, fmt.Sprintf("s.started_at >= $%d", idx))
		args = append(args, *f.FromDate)
		idx++
	}
	if f.ToDate != nil {
		whereClauses = append(whereClauses, fmt.Sprintf("s.started_at <= $%d", idx))
		args = append(args, *f.ToDate)
		idx++
	}

	whereSQL := ""
	if len(whereClauses) > 0 {
		whereSQL = "WHERE " + strings.Join(whereClauses, " AND ")
	}

	total := 0
	if !f.SkipCount {
		countQuery := fmt.Sprintf("SELECT COUNT(*) FROM ai_telemetry.chat_session s %s", whereSQL)
		if err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
			return []SessionRecord{}, 0, err
		}
	}

	if f.Limit <= 0 {
		f.Limit = 50
	}
	limitSQL := fmt.Sprintf(" ORDER BY s.started_at DESC LIMIT %d OFFSET %d", f.Limit, f.Offset)
	selectQuery := fmt.Sprintf(`
		SELECT s.id, s.tenant_id, s.conversation_id, s.agent_id, s.agent_version, s.user_id, s.user_email, s.view_type,
		       s.embedded, s.embed_surface, s.started_at, s.ended_at, s.message_count,
		       s.feedback_score, s.feedback_comment, s.metadata, s.trace_id, s.previous_hash, s.current_hash,
		       s.created_at, s.updated_at,
		       fm.content AS first_message,
		       lm.content AS last_message
		FROM ai_telemetry.chat_session s
		LEFT JOIN LATERAL (
			SELECT content FROM ai_telemetry.chat_message
			WHERE session_id = s.id ORDER BY seq ASC LIMIT 1
		) fm ON true
		LEFT JOIN LATERAL (
			SELECT content FROM ai_telemetry.chat_message
			WHERE session_id = s.id ORDER BY seq DESC LIMIT 1
		) lm ON true
		%s %s;
	`, whereSQL, limitSQL)

	rows, err := r.db.QueryContext(ctx, selectQuery, args...)
	if err != nil {
		return []SessionRecord{}, 0, err
	}
	defer rows.Close()

	sessions := make([]SessionRecord, 0)
	for rows.Next() {
		var s SessionRecord
		var meta []byte
		err := rows.Scan(
			&s.ID, &s.TenantID, &s.ConversationID, &s.AgentID, &s.AgentVersion, &s.UserID, &s.UserEmail, &s.ViewType,
			&s.Embedded, &s.EmbedSurface, &s.StartedAt, &s.EndedAt, &s.MessageCount,
			&s.FeedbackScore, &s.FeedbackComment, &meta, &s.TraceID, &s.PreviousHash, &s.CurrentHash,
			&s.CreatedAt, &s.UpdatedAt,
			&s.FirstMessage, &s.LastMessage,
		)
		if err != nil {
			return []SessionRecord{}, 0, err
		}
		s.Metadata = json.RawMessage(meta)
		sessions = append(sessions, s)
	}

	return sessions, total, nil
}

func (r *Repository) GetSession(ctx context.Context, tenantID, sessionID uuid.UUID, allowCrossTenant bool) (*SessionDetail, error) {
	where := "WHERE s.id = $1"
	args := []interface{}{sessionID}
	if !allowCrossTenant {
		where += " AND s.tenant_id = $2"
		args = append(args, tenantID)
	}

	sessionQuery := fmt.Sprintf(`
		SELECT s.id, s.tenant_id, s.conversation_id, s.agent_id, s.agent_version, s.user_id, s.user_email, s.view_type,
		       s.embedded, s.embed_surface, s.started_at, s.ended_at, s.message_count,
		       s.feedback_score, s.feedback_comment, s.metadata, s.trace_id, s.previous_hash, s.current_hash,
		       s.created_at, s.updated_at
		FROM ai_telemetry.chat_session s %s;
	`, where)

	var s SessionRecord
	var meta []byte
	err := r.db.QueryRowContext(ctx, sessionQuery, args...).Scan(
		&s.ID, &s.TenantID, &s.ConversationID, &s.AgentID, &s.AgentVersion, &s.UserID, &s.UserEmail, &s.ViewType,
		&s.Embedded, &s.EmbedSurface, &s.StartedAt, &s.EndedAt, &s.MessageCount,
		&s.FeedbackScore, &s.FeedbackComment, &meta, &s.TraceID, &s.PreviousHash, &s.CurrentHash,
		&s.CreatedAt, &s.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, ErrSessionNotFound
	} else if err != nil {
		return nil, err
	}
	s.Metadata = json.RawMessage(meta)

	messages, err := r.GetMessages(ctx, s.TenantID, s.ID)
	if err != nil {
		return nil, err
	}

	return &SessionDetail{
		Session:  s,
		Messages: messages,
	}, nil
}

func (r *Repository) GetMessages(ctx context.Context, tenantID, sessionID uuid.UUID) ([]MessageRecord, error) {
	msgQuery := `
		SELECT id, session_id, tenant_id, seq, role, content, content_json, tool_calls,
		       chart_spec, trace_steps, latency_ms, token_in, token_out, trace_id, span_id, error, created_at
		FROM ai_telemetry.chat_message
		WHERE session_id = $1 AND tenant_id = $2
		ORDER BY seq ASC;
	`
	rows, err := r.db.QueryContext(ctx, msgQuery, sessionID, tenantID)
	if err != nil {
		return []MessageRecord{}, err
	}
	defer rows.Close()

	messages := make([]MessageRecord, 0)
	for rows.Next() {
		var m MessageRecord
		var cJSON, toolCalls, chartSpec, traceSteps []byte
		err := rows.Scan(
			&m.ID, &m.SessionID, &m.TenantID, &m.Seq, &m.Role, &m.Content,
			&cJSON, &toolCalls, &chartSpec, &traceSteps,
			&m.LatencyMs, &m.TokenIn, &m.TokenOut, &m.TraceID, &m.SpanID, &m.Error, &m.CreatedAt,
		)
		if err != nil {
			return []MessageRecord{}, err
		}
		if len(cJSON) > 0 {
			m.ContentJSON = json.RawMessage(cJSON)
		}
		if len(toolCalls) > 0 {
			m.ToolCalls = json.RawMessage(toolCalls)
		}
		if len(chartSpec) > 0 {
			m.ChartSpec = json.RawMessage(chartSpec)
		}
		if len(traceSteps) > 0 {
			m.TraceSteps = json.RawMessage(traceSteps)
		}
		messages = append(messages, m)
	}
	return messages, nil
}

// RecordMessageDirect is used by the ChatEngine when it already holds an
// open OTel span context. It captures trace_id/span_id from arguments so
// the call site stays in control of the active span.
func (r *Repository) RecordMessageDirect(ctx context.Context, tenantID, sessionID uuid.UUID, msg MessageRecord, traceID, spanID string) (*MessageRecord, error) {
	if traceID != "" {
		t := traceID
		msg.TraceID = &t
	}
	if spanID != "" {
		s := spanID
		msg.SpanID = &s
	}
	return r.AppendMessage(ctx, tenantID, sessionID, msg)
}

// StampSessionTrace stores the session-level OTel trace id so the chat
// history UI can link directly to the whole-conversation trace.
func (r *Repository) StampSessionTrace(ctx context.Context, tenantID, sessionID uuid.UUID, traceID string) error {
	if traceID == "" {
		return nil
	}
	_, err := r.db.ExecContext(ctx, `
		UPDATE ai_telemetry.chat_session
		SET trace_id = $1, updated_at = NOW()
		WHERE id = $2 AND tenant_id = $3;
	`, traceID, sessionID, tenantID)
	return err
}