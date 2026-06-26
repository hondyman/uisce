package services

import (
	"context"
	"strings"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

func starlarkTracer() trace.Tracer {
	return otel.Tracer("semlayer/starlark")
}

type starlarkBundleSpan struct {
	span  trace.Span
	start time.Time
}

func startStarlarkBundleSpan(ctx context.Context, mode string, rulesCount int, shortCircuit bool) (*starlarkBundleSpan, context.Context) {
	attrs := []attribute.KeyValue{
		attribute.String("starlark.mode", mode),
		attribute.Int("starlark.rules_total", rulesCount),
		attribute.Bool("starlark.short_circuit", shortCircuit),
	}
	ctx2, span := starlarkTracer().Start(ctx, "starlark.bundle", trace.WithAttributes(attrs...))
	return &starlarkBundleSpan{span: span, start: time.Now()}, ctx2
}

func (s *starlarkBundleSpan) end(err error) {
	if s == nil {
		return
	}
	s.span.SetAttributes(attribute.Float64("starlark.duration_ms", float64(time.Since(s.start).Microseconds())/1000.0))
	if err != nil {
		s.span.RecordError(err)
		s.span.SetStatus(codes.Error, "bundle_error")
	} else {
		s.span.SetStatus(codes.Ok, "")
	}
	s.span.End()
}

type starlarkRuleSpan struct {
	span    trace.Span
	start   time.Time
	ruleID  string
	mode    string
	ended   bool
	ctxUsed context.Context
}

func startStarlarkRuleSpan(ctx context.Context, ruleID, mode string) (*starlarkRuleSpan, context.Context) {
	normID := normalizeRuleID(ruleID)
	attrs := []attribute.KeyValue{
		attribute.String("starlark.rule_id", normID),
		attribute.String("starlark.mode", mode),
	}
	ctx2, span := starlarkTracer().Start(ctx, "starlark.rule", trace.WithAttributes(attrs...))
	return &starlarkRuleSpan{span: span, start: time.Now(), ruleID: normID, mode: mode, ctxUsed: ctx2}, ctx2
}

func (s *starlarkRuleSpan) end(res *StarlarkValidationResult, err error) {
	if s == nil || s.ended {
		return
	}
	s.ended = true

	outcome := classifyStarlarkOutcome(res, err)
	s.span.SetAttributes(
		attribute.String("starlark.outcome", outcome),
		attribute.Float64("starlark.duration_ms", float64(time.Since(s.start).Microseconds())/1000.0),
	)

	if err != nil {
		s.span.RecordError(err)
		s.span.SetStatus(codes.Error, "starlark_error")
		s.span.End()
		return
	}

	if res == nil {
		s.span.SetStatus(codes.Error, "nil_result")
		s.span.End()
		return
	}

	if outcome == "error" {
		// Avoid attaching res.Message (can contain user data). Only classify.
		msg := strings.ToLower(strings.TrimSpace(res.Message))
		errClass := "unknown"
		switch {
		case strings.HasPrefix(msg, "script error"):
			errClass = "script"
		case strings.HasPrefix(msg, "runtime error"):
			errClass = "runtime"
		case strings.Contains(msg, "did not define"):
			errClass = "missing_required_global"
		case strings.Contains(msg, "not found"):
			errClass = "missing_function"
		}
		s.span.SetAttributes(attribute.String("starlark.error_class", errClass))
		s.span.SetStatus(codes.Error, errClass)
		s.span.End()
		return
	}

	s.span.SetStatus(codes.Ok, "")
	s.span.End()
}
