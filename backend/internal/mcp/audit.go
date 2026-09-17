package mcp

// Audit writer for unified Server tool calls.
//
// Schema verdict (PR C, live DB): only catalog_mdm_ai.mcp_tool_execution_logs
// exists. Path 2's INSERT into catalog_ai.mcp_tool_execution_logs was a
// silent no-op (table absent) — count(*)=0. This writer targets
// catalog_mdm_ai and logs insert failures loudly (tool response still
// succeeds — availability over fail-closed audit).
//
// Prompt policy (text_to_semantic_ast): truncate to AuditPromptMaxRunes
// (512) and store prompt_len alongside. Enough to debug; bounds PII.

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"log"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/hondyman/uisce/backend/internal/security"
	"github.com/jmoiron/sqlx"
)

const (
	auditTable          = "catalog_mdm_ai.mcp_tool_execution_logs"
	AuditPromptMaxRunes = 512
	redactedSentinel    = "<redacted>"
)

var sensitiveArgKeys = map[string]struct{}{
	"password":      {},
	"passwd":        {},
	"secret":        {},
	"token":         {},
	"api_key":       {},
	"apikey":        {},
	"authorization": {},
	"access_token":  {},
	"refresh_token": {},
	"jwt":           {},
	"credential":    {},
	"credentials":   {},
}

type auditRecord struct {
	TenantID   uuid.UUID
	ToolName   string
	Actor      string
	Args       json.RawMessage
	Result     interface{}
	DurationMs int
	Success    bool
	ErrMsg     string
}

func redactArgs(tool string, raw json.RawMessage) json.RawMessage {
	if len(raw) == 0 {
		return json.RawMessage(`{}`)
	}
	var m map[string]interface{}
	if err := json.Unmarshal(raw, &m); err != nil {
		return json.RawMessage(`{"_unparsed":"<redacted>"}`)
	}
	out := redactMap(m)
	if tool == "text_to_semantic_ast" {
		if prompt, ok := out["prompt"].(string); ok {
			out["prompt"] = truncateRunes(prompt, AuditPromptMaxRunes)
			out["prompt_len"] = utf8.RuneCountInString(prompt)
		}
	}
	b, err := json.Marshal(out)
	if err != nil {
		return json.RawMessage(`{}`)
	}
	return b
}

func redactMap(m map[string]interface{}) map[string]interface{} {
	out := make(map[string]interface{}, len(m))
	for k, v := range m {
		lk := strings.ToLower(k)
		if _, sens := sensitiveArgKeys[lk]; sens {
			out[k] = redactedSentinel
			continue
		}
		switch child := v.(type) {
		case map[string]interface{}:
			out[k] = redactMap(child)
		default:
			out[k] = v
		}
	}
	return out
}

func truncateRunes(s string, max int) string {
	if max <= 0 || utf8.RuneCountInString(s) <= max {
		return s
	}
	var b strings.Builder
	n := 0
	for _, r := range s {
		if n >= max {
			break
		}
		b.WriteRune(r)
		n++
	}
	return b.String()
}

func writeToolAudit(ctx context.Context, db *sqlx.DB, rec auditRecord) {
	if db == nil {
		return
	}
	redacted := redactArgs(rec.ToolName, rec.Args)
	outBytes, _ := json.Marshal(rec.Result)
	if !rec.Success && rec.ErrMsg != "" && (len(outBytes) == 0 || string(outBytes) == "null") {
		outBytes, _ = json.Marshal(map[string]string{"error": rec.ErrMsg})
	}
	sum := sha256.Sum256(append(append([]byte{}, redacted...), outBytes...))
	receipt := hex.EncodeToString(sum[:])

	errMsg := interface{}(nil)
	if rec.ErrMsg != "" {
		errMsg = rec.ErrMsg
	}

	_, err := db.ExecContext(ctx, `
		INSERT INTO catalog_mdm_ai.mcp_tool_execution_logs (
			tenant_id, tool_name, request_parameters, response_data,
			execution_duration_ms, success, error_message, merkle_receipt
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`, rec.TenantID, rec.ToolName, redacted, outBytes, rec.DurationMs, rec.Success, errMsg, receipt)
	if err != nil {
		log.Printf("mcp audit insert failed table=%s tool=%s tenant=%s err=%v", auditTable, rec.ToolName, rec.TenantID, err)
	}
}

func (s *Server) auditCall(ctx context.Context, tenantID uuid.UUID, name string, args json.RawMessage, result interface{}, callErr error, started time.Time) {
	actor := "mcp"
	if auth, ok := security.AuthInfoFromContext(ctx); ok && auth.UserID != "" {
		actor = auth.UserID
	}
	rec := auditRecord{
		TenantID:   tenantID,
		ToolName:   name,
		Actor:      actor,
		Args:       args,
		Result:     result,
		DurationMs: int(time.Since(started).Milliseconds()),
		Success:    callErr == nil,
	}
	if callErr != nil {
		rec.ErrMsg = callErr.Error()
	}
	writeToolAudit(ctx, s.db, rec)
}
