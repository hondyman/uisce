package msgcat

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
)

// Severity of a catalog message.
type Severity string

const (
	SeverityMessage Severity = "Message"
	SeverityWarning Severity = "Warning"
	SeverityError   Severity = "Error"
	SeverityFatal   Severity = "Fatal"
)

// ValidSeverity reports whether s is one of the four catalog severities.
func ValidSeverity(s string) bool {
	switch Severity(s) {
	case SeverityMessage, SeverityWarning, SeverityError, SeverityFatal:
		return true
	}
	return false
}

// Well-known sets.
const (
	SetSystem = 1    // platform-wide generic messages
	SetMsgcat = 9100 // the catalog's own messages
)

// Error is a user-facing error: a catalog message plus its parameters. The
// wrapped cause is for logs only; clients receive the rendered catalog text,
// never the cause.
type Error struct {
	Set    int
	Nbr    int
	Params []string
	// Status is the HTTP status to answer with; 0 means 500 for a Fatal
	// message and 400 otherwise.
	Status int
	// Details are structured, client-safe facts (the rules that rejected a
	// write, the fields it left empty) sent alongside the message.
	Details map[string]any
	cause   error
}

// New returns the catalog message set-nbr with its parameters (%1, %2, ...).
func New(set, nbr int, params ...any) *Error {
	e := &Error{Set: set, Nbr: nbr}
	for _, p := range params {
		e.Params = append(e.Params, fmt.Sprint(p))
	}
	return e
}

// WithStatus returns a copy answering with the given HTTP status.
func (e *Error) WithStatus(status int) *Error {
	c := *e
	c.Status = status
	return &c
}

// WithDetail returns a copy carrying one client-safe detail.
func (e *Error) WithDetail(key string, value any) *Error {
	c := *e
	c.Details = map[string]any{}
	for k, v := range e.Details {
		c.Details[k] = v
	}
	c.Details[key] = value
	return &c
}

// Wrap returns a copy carrying the internal cause (logged, never sent).
func (e *Error) Wrap(cause error) *Error {
	c := *e
	c.cause = cause
	return &c
}

// Code is the message's catalog code, "set-nbr".
func (e *Error) Code() string { return strconv.Itoa(e.Set) + "-" + strconv.Itoa(e.Nbr) }

func (e *Error) Error() string {
	s := "msgcat " + e.Code()
	if len(e.Params) > 0 {
		s += " [" + strings.Join(e.Params, ", ") + "]"
	}
	if e.cause != nil {
		s += ": " + e.cause.Error()
	}
	return s
}

func (e *Error) Unwrap() error { return e.cause }

// ParseCode parses "set-nbr".
func ParseCode(code string) (set, nbr int, ok bool) {
	a, b, found := strings.Cut(strings.TrimSpace(code), "-")
	if !found {
		return 0, 0, false
	}
	set, err1 := strconv.Atoi(a)
	nbr, err2 := strconv.Atoi(b)
	if err1 != nil || err2 != nil || set < 1 || nbr < 1 {
		return 0, 0, false
	}
	return set, nbr, true
}

// Generic system messages (set 1).

// Internal is the answer to any error that is not a catalog message: the
// client learns only the reference, which finds the cause in the logs.
func Internal(ref string) *Error {
	return New(SetSystem, 4, ref).WithStatus(http.StatusInternalServerError)
}

// Unauthenticated: no valid credentials.
func Unauthenticated() *Error { return New(SetSystem, 3).WithStatus(http.StatusUnauthorized) }

// InvalidInput: %1 describes what is wrong.
func InvalidInput(what string) *Error { return New(SetSystem, 5, what) }

// MalformedJSON: the request body did not decode.
func MalformedJSON() *Error { return New(SetSystem, 13) }

// NotPermitted: %1 names the operation.
func NotPermitted(what string) *Error {
	return New(SetSystem, 10, what).WithStatus(http.StatusForbidden)
}

// DatasourceNotAvailable: the selected datasource does not exist or is not
// one the caller's tenant may use. The caller is authenticated; the scope
// they asked for is what is wrong.
func DatasourceNotAvailable() *Error {
	return New(SetSystem, 16).WithStatus(http.StatusForbidden)
}
