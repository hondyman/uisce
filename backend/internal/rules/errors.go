package rules

import "fmt"

type EvalErrorKind string

const (
	EvalAuthoring EvalErrorKind = "authoring" // bad script: syntax, wrong return type, disallowed helper
	EvalData      EvalErrorKind = "data"      // input-specific: missing field, bad value
	EvalInfra     EvalErrorKind = "infra"     // DB/network/timeouts building ctx
)

type EvalError struct {
	Kind EvalErrorKind
	Msg  string
	Err  error // Underlying error if any
}

func (e *EvalError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %s (cause: %v)", e.Kind, e.Msg, e.Err)
	}
	return string(e.Kind) + ": " + e.Msg
}

func (e *EvalError) Unwrap() error {
	return e.Err
}
