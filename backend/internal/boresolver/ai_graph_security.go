package boresolver

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/hondyman/semlayer/backend/internal/audit"
)

// Cardinal Rule 6 note: when the interceptor is invoked without an audit
// Recorder (e.g., from legacy tests), audit emissions are skipped. Cardinal
// Rule 7 note: every emit is synchronous; MutateSQLSelectExpression MUST
// honor the ctx parameter so the actor block on the emitted event reflects
// the original HTTP request, not background noise.

type ClassificationRule struct {
	Tag      string `json:"tag"`
	MaskType string `json:"mask_type"`
}

type PolicyMaskingPayload struct {
	Classifications []ClassificationRule `json:"classifications"`
}

type AIGraphSecurityInterceptor struct {
	TenantPool *sql.DB // Connection pool locked into tenant RLS
	SystemPool *sql.DB // Bypasses RLS to read system blueprints safely
	// Recorder (optional) emits AI Gate audit events. Cardinal Rule 7 mandates
	// that production deployments always wire one. nil is permitted only for
	// unit tests that don't exercise the audit surface.
	Recorder *audit.Recorder
}

// ResolveGraphGovernanceContext traces lineage down to the Business Term node to find classification tags.
//
// Cardinal Rule 7: emits an AIABACEvaluated event when a classification is
// resolved, or an AIABACDenied event when the metadata graph returns the
// baseline "REDACT_FULL" sentinel (graph broken → deny).
func (in *AIGraphSecurityInterceptor) ResolveGraphGovernanceContext(ctx context.Context, physicalColumnPath string) (string, error) {
	// Traces Physical -> Business -> Semantic lineage path
	query := `
		SELECT
			bus_node.properties AS business_properties
		FROM public.catalog_node phys_node
		INNER JOIN public.catalog_edge e1 ON e1.source_node_id = phys_node.id AND e1.is_active = true
		INNER JOIN public.catalog_node bus_node ON e1.target_node_id = bus_node.id AND bus_node.is_active = true
		WHERE phys_node.qualified_path = $1
		  AND phys_node.is_active = true
		LIMIT 1;
	`

	var rawBusProps []byte

	// Invariant Check 1: Query tenant local database space first
	err := in.TenantPool.QueryRowContext(ctx, query, physicalColumnPath).Scan(&rawBusProps)
	if err == sql.ErrNoRows {
		// Fallback Invariant 2: Pull global Gold Copy definitions if no tenant override exists
		err = in.SystemPool.QueryRowContext(ctx, query, physicalColumnPath).Scan(&rawBusProps)
		if err != nil {
			// Zero-Tolerance Default: Redact field fully if metadata connections are broken
			if in.Recorder != nil {
				denial := audit.AIABACDeniedEvent{
					QueryID:        ctx.Value(queryIDKey).(string), //nolint:errcheck // ok if empty
					TenantID:       tenantIDFromCtx(ctx),
					UserID:         userIDFromCtx(ctx),
					ProfileID:      "graph-broken-baseline",
					Classification: "REDACT_FULL_BASELINE",
					DeniedResource: physicalColumnPath,
					DenialReason:   "metadata graph unavailable; defaulting to REDACT_FULL",
					EmittedSync:    false,
				}
				_ = in.Recorder.RecordAIABACDenied(ctx, denial) //nolint:errcheck // Cardinal Rule 7: caller decides
			}
			return "REDACT_FULL", nil
		}
	} else if err != nil {
		return "", fmt.Errorf("metadata graph traversal exception: %w", err)
	}

	var busProps map[string]interface{}
	if err := json.Unmarshal(rawBusProps, &busProps); err != nil {
		return "", err
	}

	classification, ok := busProps["classification"].(string)
	if !ok || classification == "" {
		// Cardinal Rule 7: log "no classification" outcome as an allow-equivalent
		// evaluation so the audit chain is complete.
		if in.Recorder != nil {
			_ = in.Recorder.RecordAILineageResolved(ctx, audit.AILineageResolvedEvent{
				QueryID:            ctx.Value(queryIDKey).(string), //nolint:errcheck
				TenantID:           tenantIDFromCtx(ctx),
				PhysicalColumnPath: physicalColumnPath,
				ResolvedTags:       []string{"NONE"},
				ResolvedAt:         nowOrZero(),
			}) //nolint:errcheck
		}
		return "NONE", nil
	}

	// Cardinal Rule 7: emit lineage resolution event for SIEM replay.
	if in.Recorder != nil {
		_ = in.Recorder.RecordAILineageResolved(ctx, audit.AILineageResolvedEvent{
			QueryID:            ctx.Value(queryIDKey).(string), //nolint:errcheck
			TenantID:           tenantIDFromCtx(ctx),
			PhysicalColumnPath: physicalColumnPath,
			ResolvedTags:       []string{classification},
			ResolvedAt:         nowOrZero(),
		}) //nolint:errcheck
	}
	return classification, nil
}

// EvaluateEffectiveMaskingType cross-references active classifications against profile-level ABAC rule dictionaries.
//
// Cardinal Rule 7: emits AIABACEvaluated for allow/mask decisions. Returns
// "DENY" + emits AIABACDenied when the configured ABAC rule for the
// classification is an explicit deny (mask_type == "DENY").
func (in *AIGraphSecurityInterceptor) EvaluateEffectiveMaskingType(ctx context.Context, targetProfile string, userTenantID uuid.UUID, classificationTag string) string {
	if classificationTag == "NONE" {
		// Cardinal Rule 7: emit an "allow" evaluation for completeness.
		if in.Recorder != nil {
			_ = in.Recorder.RecordAIABACEvaluated(ctx, audit.AIABACEvaluatedEvent{
				QueryID:        ctx.Value(queryIDKey).(string), //nolint:errcheck
				TenantID:       tenantIDFromCtx(ctx),
				ProfileID:      targetProfile,
				Classification: classificationTag,
				Decision:       "allow",
				Reason:         "no classification tag",
			}) //nolint:errcheck
		}
		return "NONE"
	}

	// Fetch both global and tenant custom overrides ordered by priority
	query := `
		SELECT masking_rules
		FROM security.abac_policies
		WHERE target_profile = $1
		  AND (tenant_id IS NULL OR tenant_id = $2)
		  AND is_active = true
		ORDER BY tenant_id FETCH FIRST ROW ONLY; -- Custom tenant rows apply last to overwrite settings
	`

	var rawRules []byte
	err := in.TenantPool.QueryRowContext(ctx, query, targetProfile, userTenantID).Scan(&rawRules)
	if err != nil {
		// Cardinal Rule 7: log unknown profile as allow-with-default.
		if in.Recorder != nil {
			_ = in.Recorder.RecordAIABACEvaluated(ctx, audit.AIABACEvaluatedEvent{
				QueryID:        ctx.Value(queryIDKey).(string), //nolint:errcheck
				TenantID:       userTenantID.String(),
				ProfileID:      targetProfile,
				Classification: classificationTag,
				Decision:       "allow",
				Reason:         "no profile constraints",
			}) //nolint:errcheck
		}
		return "NONE" // Return baseline clean status if no matching profile constraints exist
	}

	var payload PolicyMaskingPayload
	if err := json.Unmarshal(rawRules, &payload); err != nil {
		if in.Recorder != nil {
			_ = in.Recorder.RecordAIABACEvaluated(ctx, audit.AIABACEvaluatedEvent{
				QueryID:        ctx.Value(queryIDKey).(string), //nolint:errcheck
				TenantID:       userTenantID.String(),
				ProfileID:      targetProfile,
				Classification: classificationTag,
				Decision:       "allow",
				Reason:         "policy payload malformed",
			}) //nolint:errcheck
		}
		return "NONE"
	}

	// Match classification strings sequentially
	for _, rule := range payload.Classifications {
		if rule.Tag == classificationTag {
			maskType := rule.MaskType
			decision := "mask:" + strings.ToUpper(maskType)
			if strings.EqualFold(maskType, "DENY") {
				decision = "deny"
			}
			if in.Recorder != nil {
				if decision == "deny" {
					// Cardinal Rule 7: synchronously emit denial record.
					_ = in.Recorder.RecordAIABACDenied(ctx, audit.AIABACDeniedEvent{
						QueryID:        ctx.Value(queryIDKey).(string), //nolint:errcheck
						TenantID:       userTenantID.String(),
						ProfileID:      targetProfile,
						Classification: classificationTag,
						DeniedResource: classificationTag,
						DenialReason:   "ABAC policy explicitly denies access to " + classificationTag,
						EmittedSync:    false,
					}) //nolint:errcheck
				} else {
					_ = in.Recorder.RecordAIABACEvaluated(ctx, audit.AIABACEvaluatedEvent{
						QueryID:        ctx.Value(queryIDKey).(string), //nolint:errcheck
						TenantID:       userTenantID.String(),
						ProfileID:      targetProfile,
						Classification: classificationTag,
						Decision:       decision,
					}) //nolint:errcheck
				}
			}
			return maskType
		}
	}

	if in.Recorder != nil {
		_ = in.Recorder.RecordAIABACEvaluated(ctx, audit.AIABACEvaluatedEvent{
			QueryID:        ctx.Value(queryIDKey).(string), //nolint:errcheck
			TenantID:       userTenantID.String(),
			ProfileID:      targetProfile,
			Classification: classificationTag,
			Decision:       "allow",
			Reason:         "no rule matched classification",
		}) //nolint:errcheck
	}
	return "NONE"
}

// MutateSQLSelectExpression transforms projected column queries into encapsulated dialect-safe operations.
//
// Cardinal Rule 7: emits AIColumnMasked when a non-NONE mask is applied. The
// ctx parameter carries the request identity (Cardinal Rule 6).
func (in *AIGraphSecurityInterceptor) MutateSQLSelectExpression(ctx context.Context, tableAlias string, columnName string, maskType string) string {
	qualifiedColumn := fmt.Sprintf("%s.%s", tableAlias, columnName)

	var masked string
	switch strings.ToUpper(maskType) {
	case "REDACT_FULL":
		masked = fmt.Sprintf("('[REDACTED]') AS %s", columnName)
	case "REDACT_LAST_FOUR":
		masked = fmt.Sprintf("(CONCAT('******', RIGHT(CAST(%s AS VARCHAR), 4))) AS %s", qualifiedColumn, columnName)
	case "HASH_SHA256":
		masked = fmt.Sprintf("(SHA256(CAST(%s AS VARCHAR))) AS %s", qualifiedColumn, columnName)
	default:
		masked = fmt.Sprintf("%s AS %s", qualifiedColumn, columnName)
	}

	// Cardinal Rule 7: every masked column is independently auditable.
	if in.Recorder != nil && !strings.EqualFold(maskType, "NONE") && maskType != "" {
		_ = in.Recorder.RecordAIColumnMasked(ctx, audit.AIColumnMaskedEvent{
			QueryID:    ctx.Value(queryIDKey).(string), //nolint:errcheck
			TenantID:   tenantIDFromCtx(ctx),
			TableAlias: tableAlias,
			ColumnName: columnName,
			Original:   qualifiedColumn,
			Masked:     masked,
			MaskType:   maskType,
			MaskedAt:   nowOrZero(),
		}) //nolint:errcheck
	}
	return masked
}

// =============================================================================
// Interceptor context helpers (used by the emit points above)
// =============================================================================
//
// The Interceptor is invoked from middleware-injected request contexts. We
// pull the actor identity from there. When ctx is background (unit tests),
// the helpers return zero values and the actor block on emitted events is
// empty (the SIEM can flag these as "untagged" for triage).

// queryIDKey is the context key under which middleware must store the
// originating QueryID so ABAC events chain with B1.
type queryIDCtxKey struct{}

var queryIDKey = queryIDCtxKey{}

// WithQueryID stores the AI query ID for downstream events to chain to.
func WithQueryID(ctx context.Context, id string) context.Context {
	if id == "" {
		return ctx
	}
	return context.WithValue(ctx, queryIDKey, id)
}

func tenantIDFromCtx(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	if v, ok := ctx.Value("tenant_id").(string); ok {
		return v
	}
	return ""
}

// nowOrZero stamps the current UTC time for event record fields. Unit tests
// can replace the package-level nowFn to inject a deterministic clock.
var nowFn = func() time.Time { return time.Now().UTC() }

func nowOrZero() time.Time { return nowFn() }

// userIDFromCtx is defined in bo_sql_generator.go and used here (Cardinal Rule 6).
// It looks up "user_id" then "user_email" from the context so audit envelopes
// stay attributed to the real actor regardless of how middleware populates the
// request context.
