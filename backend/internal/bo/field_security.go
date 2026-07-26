package bo

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"go.uber.org/zap"
)

// FieldClassification defines the security tier for a field.
type FieldClassification string

const (
	ClassPublic     FieldClassification = "PUBLIC"
	ClassMasked     FieldClassification = "MASKED"
	ClassEncrypted  FieldClassification = "ENCRYPTED"
	ClassRestricted FieldClassification = "RESTRICTED"
)

// FieldSecurityConfig is one row from bo.field_security.
type FieldSecurityConfig struct {
	SecurityID      string              `db:"security_id"       json:"security_id"`
	TenantID        string              `db:"tenant_id"         json:"tenant_id"`
	BOKey           string              `db:"bo_key"            json:"bo_key"`
	FieldKey        string              `db:"field_key"         json:"field_key"`
	Classification  FieldClassification `db:"classification"    json:"classification"`
	MaskPattern     *string             `db:"mask_pattern"      json:"mask_pattern,omitempty"`
	VisibleToRoles  []string            `db:"visible_to_roles"  json:"visible_to_roles"`
	MaskForRoles    []string            `db:"mask_for_roles"    json:"mask_for_roles"`
	DenyToRoles     []string            `db:"deny_to_roles"     json:"deny_to_roles"`
	IsCore          bool                `db:"is_core"           json:"is_core"`
	CreatedAt       time.Time           `db:"created_at"        json:"created_at"`
}

// FieldVisibility describes how a single field appears to a specific role.
type FieldVisibility string

const (
	VisibilityFull     FieldVisibility = "FULL"       // original value shown
	VisibilityMasked   FieldVisibility = "MASKED"     // pattern-masked value
	VisibilityHidden   FieldVisibility = "HIDDEN"     // field not included in response
	VisibilityRedacted FieldVisibility = "REDACTED"   // "RESTRICTED" placeholder
)

// FieldMaskResult is the per-field masking decision for a given role.
type FieldMaskResult struct {
	FieldKey    string          `json:"field_key"`
	Visibility  FieldVisibility `json:"visibility"`
	MaskedValue string          `json:"masked_value,omitempty"` // set when MASKED
}

// FieldSecurityMasker applies field-level security rules to BO record responses.
type FieldSecurityMasker struct {
	db  *sql.DB
	log *zap.Logger
}

// NewFieldSecurityMasker constructs a FieldSecurityMasker.
func NewFieldSecurityMasker(db *sql.DB, log *zap.Logger) *FieldSecurityMasker {
	return &FieldSecurityMasker{db: db, log: log}
}

// LoadFieldSecurityConfigs loads all field security configs for a given tenant + BO.
// Rule 7.4: Core + tenant-custom union via ROW_NUMBER() shadowing.
func (fsm *FieldSecurityMasker) LoadFieldSecurityConfigs(ctx context.Context, tenantID, boKey string) ([]FieldSecurityConfig, error) {
	const q = `
	WITH combined AS (
		SELECT *,
		       ROW_NUMBER() OVER (
		           PARTITION BY field_key
		           ORDER BY CASE WHEN is_core = false THEN 0 ELSE 1 END
		       ) AS rn
		FROM bo.field_security
		WHERE bo_key = $2
		  AND (tenant_id = $1::uuid OR is_core = true)
	)
	SELECT security_id, tenant_id, bo_key, field_key, classification,
	       mask_pattern, visible_to_roles, mask_for_roles, deny_to_roles,
	       is_core, created_at
	FROM combined
	WHERE rn = 1
	`
	rows, err := fsm.db.QueryContext(ctx, q, tenantID, boKey)
	if err != nil {
		return nil, fmt.Errorf("field_security: load configs: %w", err)
	}
	defer rows.Close()

	var configs []FieldSecurityConfig
	for rows.Next() {
		var c FieldSecurityConfig
		// pq.StringArray for role slices
		var visibleRoles, maskRoles, denyRoles []string
		if err := rows.Scan(
			&c.SecurityID, &c.TenantID, &c.BOKey, &c.FieldKey, &c.Classification,
			&c.MaskPattern, &visibleRoles, &maskRoles, &denyRoles,
			&c.IsCore, &c.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("field_security: scan: %w", err)
		}
		c.VisibleToRoles = visibleRoles
		c.MaskForRoles = maskRoles
		c.DenyToRoles = denyRoles
		configs = append(configs, c)
	}
	return configs, rows.Err()
}

// ApplyMasks takes a raw BO record (map) and applies field-level security masks
// based on the principal's roles. Returns a sanitised copy of the record.
// Fields with VisibilityHidden are omitted from the output map.
func (fsm *FieldSecurityMasker) ApplyMasks(
	ctx context.Context,
	tenantID, boKey string,
	record map[string]interface{},
	roles []string,
) (map[string]interface{}, []FieldMaskResult, error) {

	configs, err := fsm.LoadFieldSecurityConfigs(ctx, tenantID, boKey)
	if err != nil {
		return nil, nil, err
	}

	// Index configs by field_key for O(1) lookup
	configIndex := make(map[string]*FieldSecurityConfig, len(configs))
	for i := range configs {
		configIndex[configs[i].FieldKey] = &configs[i]
	}

	// Build role set
	roleSet := make(map[string]struct{}, len(roles))
	for _, r := range roles {
		roleSet[r] = struct{}{}
	}

	sanitised := make(map[string]interface{}, len(record))
	var maskResults []FieldMaskResult

	for fieldKey, rawValue := range record {
		cfg, ok := configIndex[fieldKey]
		if !ok {
			// No config means PUBLIC by default
			sanitised[fieldKey] = rawValue
			continue
		}

		visibility := fsm.resolveVisibility(cfg, roleSet)
		switch visibility {
		case VisibilityFull:
			sanitised[fieldKey] = rawValue
		case VisibilityMasked:
			masked := fsm.applyMaskPattern(cfg, rawValue)
			sanitised[fieldKey] = masked
			maskResults = append(maskResults, FieldMaskResult{
				FieldKey:    fieldKey,
				Visibility:  VisibilityMasked,
				MaskedValue: masked,
			})
		case VisibilityRedacted:
			sanitised[fieldKey] = "RESTRICTED"
			maskResults = append(maskResults, FieldMaskResult{
				FieldKey:   fieldKey,
				Visibility: VisibilityRedacted,
			})
		case VisibilityHidden:
			// Omit entirely — field does not appear in the response
			maskResults = append(maskResults, FieldMaskResult{
				FieldKey:   fieldKey,
				Visibility: VisibilityHidden,
			})
		}
	}

	return sanitised, maskResults, nil
}

// resolveVisibility determines how a field is shown to a principal given their roles.
// Priority: DENY > VISIBLE > MASKED
func (fsm *FieldSecurityMasker) resolveVisibility(cfg *FieldSecurityConfig, roleSet map[string]struct{}) FieldVisibility {
	// Deny roles get nothing
	for _, r := range cfg.DenyToRoles {
		if _, ok := roleSet[r]; ok {
			return VisibilityHidden
		}
	}
	// Explicit visible roles see full value
	for _, r := range cfg.VisibleToRoles {
		if _, ok := roleSet[r]; ok {
			return VisibilityFull
		}
	}
	// Mask roles get pattern-masked value
	for _, r := range cfg.MaskForRoles {
		if _, ok := roleSet[r]; ok {
			return VisibilityMasked
		}
	}
	// Classification fallback when no role list matches
	switch cfg.Classification {
	case ClassPublic:
		return VisibilityFull
	case ClassMasked:
		return VisibilityMasked
	case ClassEncrypted:
		return VisibilityRedacted
	case ClassRestricted:
		return VisibilityHidden
	}
	return VisibilityFull
}

// applyMaskPattern applies the configured mask pattern to a raw value.
// Pattern chars: '#' = digit shown, '*' = replaced with '*', else kept literal.
func (fsm *FieldSecurityMasker) applyMaskPattern(cfg *FieldSecurityConfig, rawValue interface{}) string {
	raw := fmt.Sprintf("%v", rawValue)
	if cfg.MaskPattern == nil || *cfg.MaskPattern == "" {
		// Default: mask all characters except last 4
		if len(raw) <= 4 {
			return strings.Repeat("*", len(raw))
		}
		return strings.Repeat("*", len(raw)-4) + raw[len(raw)-4:]
	}
	// Apply pattern: '#' passes through digit, '*' masks
	pattern := *cfg.MaskPattern
	result := make([]byte, 0, len(pattern))
	ri := 0
	for pi := 0; pi < len(pattern) && ri < len(raw); pi++ {
		switch pattern[pi] {
		case '#':
			result = append(result, raw[ri])
			ri++
		case '*':
			result = append(result, '*')
			ri++
		default:
			result = append(result, pattern[pi])
		}
	}
	return string(result)
}

// UpsertFieldSecurity creates or updates a field security config.
func (fsm *FieldSecurityMasker) UpsertFieldSecurity(ctx context.Context, tenantID string, cfg *FieldSecurityConfig) error {
	cfg.TenantID = tenantID
	const q = `
	INSERT INTO bo.field_security
	    (security_id, tenant_id, bo_key, field_key, classification,
	     mask_pattern, visible_to_roles, mask_for_roles, deny_to_roles,
	     is_core, created_at)
	VALUES
	    (COALESCE(NULLIF($1,'')::uuid, gen_random_uuid()), $2::uuid, $3, $4, $5,
	     $6, $7, $8, $9, $10, NOW())
	ON CONFLICT (tenant_id, bo_key, field_key) DO UPDATE SET
	    classification   = EXCLUDED.classification,
	    mask_pattern     = EXCLUDED.mask_pattern,
	    visible_to_roles = EXCLUDED.visible_to_roles,
	    mask_for_roles   = EXCLUDED.mask_for_roles,
	    deny_to_roles    = EXCLUDED.deny_to_roles,
	    is_core          = EXCLUDED.is_core
	RETURNING security_id
	`
	return fsm.db.QueryRowContext(ctx, q,
		cfg.SecurityID, tenantID, cfg.BOKey, cfg.FieldKey, string(cfg.Classification),
		cfg.MaskPattern, cfg.VisibleToRoles, cfg.MaskForRoles, cfg.DenyToRoles, cfg.IsCore,
	).Scan(&cfg.SecurityID)
}
