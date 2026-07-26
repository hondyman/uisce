// Package services - Starlark rule engine facade shim.
//
// This file re-exports the StarlarkEngine and rule types from the new
// internal/starlark domain package so existing services-package consumers
// (cmd/starlarktest/main.go, workflows/starlark_workflows.go) can gradually
// migrate to direct internal/starlark imports without breaking the monolith.
//
// Cardinal Rule 3 (no cycles): This shim only depends on internal/starlark,
// which itself does NOT depend on internal/services. Zero back-coupling.
package services

import (
	"github.com/hondyman/uisce/backend/internal/starlark"
	hasuraclient "github.com/hondyman/uisce/libs/hasura-client"
)

// ============================================================================
// STARLARK DOMAIN TYPE ALIASES
// ============================================================================

type (
	StarlarkEngine          = starlark.StarlarkEngine
	StarlarkValidationResult = starlark.StarlarkValidationResult
	ValidationResponse      = starlark.ValidationResponse
	Expression              = starlark.Expression
	ExpressionType          = starlark.ExpressionType
	OkRule                  = starlark.OkRule
	OkRuleMeta              = starlark.OkRuleMeta
	OkRuleWithMeta          = starlark.OkRuleWithMeta
)

// ============================================================================
// STARLARK CONSTRUCTOR WRAPPER
// ============================================================================

// NewStarlarkEngine delegates to internal/starlark.NewStarlarkEngine.
// Accepts *hasuraclient.HasuraClient (nil-safe).
func NewStarlarkEngine(hasuraClient *hasuraclient.HasuraClient) *starlark.StarlarkEngine {
	return starlark.NewStarlarkEngine(hasuraClient)
}

// formatStarlarkError delegates to internal/starlark.formatStarlarkError.
func formatStarlarkError(err error) string {
	return starlark.FormatStarlarkError(err)
}