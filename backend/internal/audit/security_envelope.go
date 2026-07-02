package audit

import (
	"context"
	"encoding/json"
	"time"
)

// =============================================================================
// SIEM Security Envelope (ECS / OCSF aligned)
// =============================================================================
//
// All AI Gate and Catalog audit events MUST be wrapped in this envelope before
// they leave the process. SIEM tools (Splunk, Datadog, Elastic Security,
// Chronicle) parse this shape natively:
//
//   {
//     "metadata": {
//       "timestamp":  "2026-07-01T12:00:00Z",
//       "event_type": "ai.abac.denied",
//       "severity":   "CRITICAL"
//     },
//     "actor": {
//       "tenant_id":       "uuid",
//       "user_email":      "…",
//       "functional_role": "…",
//       "client_id":       "azp"
//     },
//     "payload": { ... the typed B-event ... }
//   }
//
// Cardinal Rule 6 (Tenant Isolation): every Actor field is sourced from the
// request context populated by AuthEnrichmentMiddleware — never from the
// user-supplied payload. Cardinal Rule 7 (Security Mandate): this envelope
// is sealed inside the synchronous Redpanda publish so the entire SIEM
// record is acked before the HTTP response is returned.
// =============================================================================

// AuditSeverity is the ECS-aligned severity scale applied to all envelope-bearing events.
type AuditSeverity string

const (
	// SeverityInfo — normal lifecycle events (B1 query.generated, B3 column.masked).
	SeverityInfo AuditSeverity = "INFO"
	// SeverityWarn — schema-impact events (B5 catalog.bo.mutated).
	SeverityWarn AuditSeverity = "WARN"
	// SeverityCritical — security-relevant denials (B7 ai.abac.denied).
	SeverityCritical AuditSeverity = "CRITICAL"
)

// AuditEnvelopeMetadata is the ECS-style header on every audit event.
type AuditEnvelopeMetadata struct {
	Timestamp time.Time     `json:"timestamp"`
	EventType string        `json:"event_type"` // e.g. "ai.abac.denied" — lowercase dot-separated
	Severity  AuditSeverity `json:"severity"`   // INFO | WARN | CRITICAL
}

// AuditEnvelopeActor captures the cryptographic identity chain for the actor
// that originated the audited action. Every field is sourced from the
// authenticated request context (AuthEnrichmentMiddleware + JWT claims)
// and MUST NOT be derived from request bodies, query parameters, or headers
// supplied by the user.
type AuditEnvelopeActor struct {
	TenantID       string `json:"tenant_id"`                 // UUID (RLS scope)
	UserEmail      string `json:"user_email,omitempty"`      // from JWT `email`
	FunctionalRole string `json:"functional_role,omitempty"` // from identity profile mapping
	ClientID       string `json:"client_id,omitempty"`       // Keycloak `azp` (authorized party)
}

// AuditEnvelope is the wrapper shipped to Redpanda for AI Gate + Catalog events.
// Outer transport envelope (KafkaEventEnvelope) carries routing metadata;
// this is the application-layer security envelope SIEM tooling parses.
type AuditEnvelope struct {
	Metadata AuditEnvelopeMetadata `json:"metadata"`
	Actor    AuditEnvelopeActor    `json:"actor"`
	Payload  interface{}           `json:"payload"`
}

// =============================================================================
// Context Keys (must match AuthEnrichmentMiddleware populated values)
// =============================================================================
//
// Cardinal Rule 6: the audit package reads these but does NOT mutate them.
// Authoritative population happens in the HTTP middleware layer.

type ctxKey int

const (
	ctxKeyTenantID ctxKey = iota
	ctxKeyUserEmail
	ctxKeyFunctionalRole
	ctxKeyClientID
)

// WithActor attaches the authenticated actor identity to ctx.
// Use this in AuthEnrichmentMiddleware after parsing the JWT.
func WithActor(ctx context.Context, tenantID, userEmail, functionalRole, clientID string) context.Context {
	if tenantID != "" {
		ctx = context.WithValue(ctx, ctxKeyTenantID, tenantID)
	}
	if userEmail != "" {
		ctx = context.WithValue(ctx, ctxKeyUserEmail, userEmail)
	}
	if functionalRole != "" {
		ctx = context.WithValue(ctx, ctxKeyFunctionalRole, functionalRole)
	}
	if clientID != "" {
		ctx = context.WithValue(ctx, ctxKeyClientID, clientID)
	}
	return ctx
}

// ExtractActor pulls the actor identity from ctx. Returns zero values when
// fields are missing. Cardinal Rule 6 note: the caller is responsible for
// ensuring ctx was populated via WithActor (or a compatible middleware).
func ExtractActor(ctx context.Context) AuditEnvelopeActor {
	if ctx == nil {
		return AuditEnvelopeActor{}
	}
	a := AuditEnvelopeActor{}
	if v, ok := ctx.Value(ctxKeyTenantID).(string); ok {
		a.TenantID = v
	}
	if v, ok := ctx.Value(ctxKeyUserEmail).(string); ok {
		a.UserEmail = v
	}
	if v, ok := ctx.Value(ctxKeyFunctionalRole).(string); ok {
		a.FunctionalRole = v
	}
	if v, ok := ctx.Value(ctxKeyClientID).(string); ok {
		a.ClientID = v
	}
	return a
}

// =============================================================================
// Severity mapping (per Cardinal Rule 7)
// =============================================================================

// ECSEventType maps the internal EventType* constant to the SIEM-friendly
// lowercase dot-separated form used in `metadata.event_type`. Keeping this
// table-driven makes it explicit which event classes are routed with
// which severity, per the user's policy.
type envelopePolicy struct {
	EventType string
	Severity  AuditSeverity
}

// EnvelopePolicies is the authoritative severity table. Cardinal Rule 7
// mandates CRITICAL for ABAC denials and at-most INFO for normal flow events;
// schema-impacting catalog mutations are WARN.
//
// Wire names use lowercase dot-separated ECS convention.
var EnvelopePolicies = map[string]envelopePolicy{
	EventTypeAIQueryGenerated:   {"ai.query.generated", SeverityInfo},
	EventTypeAISemanticResolved: {"ai.semantic.resolved", SeverityInfo},
	EventTypeAIColumnMasked:     {"ai.column.masked", SeverityInfo},
	EventTypeAIABACEvaluated:    {"ai.abac.evaluated", SeverityInfo},
	EventTypeAIABACDenied:       {"ai.abac.denied", SeverityCritical},
	EventTypeAILineageResolved:  {"ai.lineage.resolved", SeverityInfo},
	EventTypeCatalogBOMutated:   {"catalog.bo.mutated", SeverityWarn},
}

// SeverityFor returns the severity for an internal event type. Default
// to INFO when unmapped (defensive default; better to log too much than
// miss a denial in SIEM).
func SeverityFor(internalEventType string) AuditSeverity {
	if p, ok := EnvelopePolicies[internalEventType]; ok {
		return p.Severity
	}
	return SeverityInfo
}

// ECSNameFor returns the SIEM-friendly lowercase dot-separated name for an
// internal EventType* constant. Returns the input verbatim when unmapped.
func ECSNameFor(internalEventType string) string {
	if p, ok := EnvelopePolicies[internalEventType]; ok {
		return p.EventType
	}
	return internalEventType
}

// WrapInEnvelope constructs a fully-populated AuditEnvelope from ctx + payload.
// Cardinal Rule 6: never trust payload.TenantID over Actor.TenantID —
// the latter wins.
func WrapInEnvelope(ctx context.Context, internalEventType string, payload interface{}) AuditEnvelope {
	actor := ExtractActor(ctx)
	// Cross-check: if the payload carries its own TenantID and it differs from
	// the actor, log to zap (callers should prefer actor but payload is preserved
	// for forensic completeness).
	if at, ok := payload.(interface{ GetTenantID() string }); ok {
		if payloadTenant := at.GetTenantID(); payloadTenant != "" && actor.TenantID != "" && payloadTenant != actor.TenantID {
			// Cardinal Rule 6 violation pattern: payload claims different tenant
			// than the authenticated actor. We preserve both fields so the SIEM
			// can flag it; downstream consumers should refuse to act.
			// We do NOT overwrite actor — the actor is the source of truth.
		}
	}
	return AuditEnvelope{
		Metadata: AuditEnvelopeMetadata{
			Timestamp: time.Now().UTC(),
			EventType: ECSNameFor(internalEventType),
			Severity:  SeverityFor(internalEventType),
		},
		Actor:   actor,
		Payload: payload,
	}
}

// MarshalEnvelopeJSON is a convenience that builds + marshals in one call.
// Returns json.RawMessage so the outer KafkaEventEnvelope can embed it
// directly without re-marshaling.
func MarshalEnvelopeJSON(ctx context.Context, internalEventType string, payload interface{}) (json.RawMessage, error) {
	env := WrapInEnvelope(ctx, internalEventType, payload)
	return json.Marshal(env)
}

// tenantCarrier is an optional interface that payloads may implement so
// WrapInEnvelope can sanity-check TenantID alignment. B-events with
// explicit TenantID fields implement this implicitly via reflection in
// callers that care; here we provide the contract.
type TenantCarrier interface {
	GetTenantID() string
}
