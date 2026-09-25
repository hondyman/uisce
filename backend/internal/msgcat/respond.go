package msgcat

import (
	"encoding/json"
	"errors"
	"net/http"
	"regexp"

	"github.com/google/uuid"

	"github.com/hondyman/uisce/backend/internal/logging"
)

// ErrorBody is what a client receives for any error. error/code/error_code
// keep the shape of the platform's existing ErrorResponse, so current
// clients that read "error" show the catalog text.
type ErrorBody struct {
	Error         string `json:"error"`      // the rendered message
	Status        int    `json:"code"`       // HTTP status
	Code          string `json:"error_code"` // catalog code, "set-nbr"
	Severity      string `json:"severity"`
	UserAction    string `json:"user_action,omitempty"`
	Language      string `json:"language"`
	CorrelationID string `json:"correlation_id"`
}

var requestIDRE = regexp.MustCompile(`^[A-Za-z0-9._-]{1,64}$`)

// CorrelationID is the request's X-Request-ID when it is well-formed (it is
// client-supplied, so anything else is replaced), else a new one.
func CorrelationID(r *http.Request) string {
	if id := r.Header.Get("X-Request-ID"); requestIDRE.MatchString(id) {
		return id
	}
	return uuid.NewString()
}

// WriteError answers r with err rendered from the catalog in the caller's
// language. A *Error is rendered as itself; anything else becomes the
// internal-error message carrying only the correlation ID. Causes are
// logged against the correlation ID and never sent.
func (c *Catalog) WriteError(w http.ResponseWriter, r *http.Request, tenantID string, err error) {
	ref := CorrelationID(r)
	var me *Error
	if !errors.As(err, &me) {
		logging.GetLogger().Sugar().Errorw("request failed", "correlation_id", ref, "method", r.Method, "path", r.URL.Path, "tenant", tenantID, "error", err)
		me = Internal(ref)
	} else if cause := errors.Unwrap(me); cause != nil {
		logging.GetLogger().Sugar().Warnw("request failed", "correlation_id", ref, "method", r.Method, "path", r.URL.Path, "tenant", tenantID, "code", me.Code(), "error", cause)
	}
	out := c.Render(r.Context(), tenantID, Preferences(r.Header.Get("Accept-Language")), me, ref)
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Request-ID", ref)
	w.Header().Set("Content-Language", out.Language)
	w.WriteHeader(out.Status)
	_ = json.NewEncoder(w).Encode(ErrorBody{
		Error: out.Message, Status: out.Status, Code: out.Code, Severity: out.Severity,
		UserAction: out.UserAction, Language: out.Language, CorrelationID: ref,
	})
}
