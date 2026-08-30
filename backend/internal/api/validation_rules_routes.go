package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"

	"github.com/hondyman/uisce/backend/internal/boresolver"
	"github.com/hondyman/uisce/backend/internal/handlers"
	"github.com/hondyman/uisce/backend/internal/rulefabric"
	"github.com/hondyman/uisce/backend/internal/security"
	"github.com/hondyman/uisce/backend/internal/services"
)

type validationRulesHandler struct {
	securityDeps  handlers.SecurityContextDeps
	db            *sql.DB
	cueEngine     *services.CueEngine // retained on the struct for signature compatibility; no longer invoked - see evaluateValidationRuleCEL
	ruleEvaluator *rulefabric.RuleEvaluator
	boService     interface{}
	resolver      security.DatasourceResolver
}

// ValidationRule represents a validation rule for data quality and business logic
type ValidationRule struct {
	ID              string                 `json:"id"`
	TenantID        string                 `json:"tenant_id"`
	DatasourceID    string                 `json:"datasource_id,omitempty"` // NEW: Datasource scope
	RuleName        string                 `json:"rule_name"`
	RuleType        string                 `json:"rule_type"` // field_format, cardinality, uniqueness, referential_integrity, business_logic
	Description     string                 `json:"description"`
	TargetEntity    string                 `json:"target_entity"`               // Legacy: single entity name
	TargetEntityID  string                 `json:"target_entity_id,omitempty"`  // NEW: Entity UUID
	TargetEntities  pq.StringArray         `json:"target_entities"`             // Legacy: Multi-entity by name
	TargetEntityIDs pq.StringArray         `json:"target_entity_ids,omitempty"` // NEW: Multi-entity by UUID
	ConditionJSON   map[string]interface{} `json:"condition_json"`
	ScriptContent   string                 `json:"script_content,omitempty"`

	Severity  string    `json:"severity"` // error, warning, info
	IsActive  bool      `json:"is_active"`
	IsCore    bool      `json:"is_core"`
	CreatedBy *string   `json:"created_by,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// ValidationRuleRequest represents the request payload for creating/updating a rule
type ValidationRuleRequest struct {
	// Backend snake_case fields
	RuleName        string                 `json:"rule_name"`
	RuleType        string                 `json:"rule_type"`
	Description     string                 `json:"description"`
	TargetEntity    string                 `json:"target_entity"`     // Legacy: entity name
	TargetEntityID  string                 `json:"target_entity_id"`  // NEW: Entity UUID
	TargetEntities  pq.StringArray         `json:"target_entities"`   // Legacy: Multi-entity by name
	TargetEntityIDs pq.StringArray         `json:"target_entity_ids"` // NEW: Multi-entity by UUID
	ConditionJSON   map[string]interface{} `json:"condition_json"`
	ScriptContent   string                 `json:"script_content"`
	Severity        string                 `json:"severity"` // error, warning, info
	IsActive        *bool                  `json:"is_active"`
	IsCore          interface{}            `json:"is_core"`       // Changed from string to interface{} to handle bool/string
	DatasourceID    string                 `json:"datasource_id"` // NEW: Datasource scope

	// Frontend camelCase fields (for compatibility with existing frontend)
	ID                 string                 `json:"id"`
	Name               string                 `json:"name"`
	RuleTypeCC         string                 `json:"ruleType"`
	ScopeArray         pq.StringArray         `json:"scope"`
	EffectiveFrom      string                 `json:"effectiveFrom"`
	EffectiveTo        string                 `json:"effectiveTo"`
	Frequency          string                 `json:"frequency"`
	EvaluationOrder    *int                   `json:"evaluationOrder"`
	OverrideConditions map[string]interface{} `json:"overrideConditions"`
	RequiredAuthority  string                 `json:"requiredAuthority"`
	Parameters         map[string]interface{} `json:"parameters"`
	TenantID           string                 `json:"tenantId"`
	DatasourceIDCC     string                 `json:"datasourceId"` // camelCase variant
}

// ValidationRuleExecutionResult represents the result of executing a validation rule
type ValidationRuleExecutionResult struct {
	RuleID    string    `json:"rule_id"`
	RuleName  string    `json:"rule_name"`
	RuleType  string    `json:"rule_type"`
	Severity  string    `json:"severity"`
	Status    string    `json:"status"` // pass, fail
	Message   string    `json:"message"`
	Timestamp time.Time `json:"timestamp"`
}

func errString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

func RegisterValidationRulesRoutes(r chi.Router, db *sql.DB, cueEngine *services.CueEngine, boService interface{}, resolver security.DatasourceResolver) {
	// Validation rule execution has moved off CUE/Starlark onto RuleFabric's
	// CEL evaluator (see evaluateValidationRuleCEL). NewRuleEvaluator only
	// needs *sqlx.DB for its own rules/rule_logic query methods, none of
	// which this package calls - only EvaluateCELBoolean is used here.
	ruleEvaluator, err := rulefabric.NewRuleEvaluator(sqlx.NewDb(db, "postgres"))
	if err != nil {
		panic(fmt.Sprintf("failed to initialize RuleFabric evaluator for validation rules: %v", err))
	}
	h := &validationRulesHandler{
		securityDeps:  handlers.SecurityContextDeps{Resolver: resolver},
		db:            db,
		cueEngine:     cueEngine,
		ruleEvaluator: ruleEvaluator,
		boService:     boService,
		resolver:      resolver,
	}

	r.Get("/validation-rules", h.handleListValidationRules())

	r.Post("/validation-rules", h.handleCreateValidationRule())
	r.Get("/validation-rules/{id}", h.handleGetValidationRule())
	r.Patch("/validation-rules/{id}", h.handleUpdateValidationRule())
	r.Delete("/validation-rules/{id}", h.handleDeleteValidationRule())

	// Core templates (tenant-agnostic)
	r.Get("/validation-rule-cores", handleListValidationRuleCores(db))
	r.Post("/validation-rule-cores", handleCreateValidationRuleCore(db))
	r.Get("/validation-rule-cores/{id}", handleGetValidationRuleCore(db))
	r.Get("/validation-rule-cores/{id}/impact", handleGetValidationRuleCoreImpact(db))

	// Execution and testing endpoints
	r.Post("/validation-rules/{id}/execute", h.handleExecuteValidationRule())
	r.Post("/validation-rules/execute-binding", h.handleExecuteValidationRuleBinding())
	// r.Post("/validation-rules/execute-batch", handleExecuteValidationRulesBatch(db, cueEngine)) // Batch disabled for now if relies on Starlark logic mismatch
	r.Get("/validation-rules/{id}/audit", h.handleGetValidationRuleAudit())

	// Simulation with instance data
	r.Post("/validation-rules/{id}/simulate-with-instance", h.handleSimulateValidationRuleWithInstance())

	// Schema Generation (IntelliSense)
	r.Get("/validation-rules/schema", h.handleGetValidationRuleSchema())
}

func (h *validationRulesHandler) handleGetValidationRuleSchema() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		secCtx, _, err := handlers.SecurityContextFromRequest(r, "", "", h.securityDeps)
		if err != nil {
			writeJSONError(w, http.StatusUnauthorized, "Security context initialization failed", "auth_error", map[string]string{"error": err.Error()})
			return
		}

		tenantID := secCtx.TenantID
		datasourceID := secCtx.DatasourceID
		boID := r.URL.Query().Get("bo_id")
		locale := r.URL.Query().Get("locale")

		if boID == "" {
			writeJSONError(w, http.StatusBadRequest, "bo_id is required", "missing_params", "")
			return
		}

		// Resolve non-UUID bo_id (name/technical name/entity_key) to the actual BO UUID scoped by tenant and datasource
		if _, err := uuid.Parse(boID); err != nil {
			lookup := `
				SELECT id
				FROM business_objects
				WHERE tenant_id = $1
				  AND ($2::text IS NULL OR datasource_id = $2)
				  AND (id::text = $3 OR name ILIKE $3 OR display_name ILIKE $3 OR technical_name ILIKE $3 OR entity_key ILIKE $3)
				LIMIT 1
			`
			var resolved sql.NullString
			err := h.db.QueryRowContext(r.Context(), lookup, tenantID, nullString(datasourceID), boID).Scan(&resolved)
			if err != nil || !resolved.Valid {
				writeJSONError(w, http.StatusBadRequest, "business object not found for provided bo_id", "bo_not_found", errString(err))
				return
			}
			boID = resolved.String
		}

		// Check invalid DB driver or just assume postgres? Application uses postgres.
		sx := sqlx.NewDb(h.db, "postgres")
		gen := services.NewCueSchemaGenerator(sx)

		schemaStr, err := gen.GenerateSchemaStringPublic(r.Context(), tenantID, boID, locale)
		if err != nil {
			fmt.Fprintf(os.Stderr, "[SCHEMA-ERROR] Failed to generate schema for BO %s: %v\n", boID, err)
			writeJSONError(w, http.StatusInternalServerError, "Failed to generate schema", "schema_error", err.Error())
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"schema": schemaStr,
		})
	}
}

type tenantRuleExecRow struct {
	rule               ValidationRule
	coreRuleID         sql.NullString
	inheritMode        sql.NullString
	coreVersionPin     sql.NullInt32
	extensionScript    sql.NullString // legacy CUE/Starlark extension script, no longer executed
	extensionCondition []byte         // extension_condition_json: the "extend" mode's additional condition, ANDed with the core rule
}

func fetchTenantRuleForExecution(ctx context.Context, db *sql.DB, id string, tenantID string, datasourceID string) (*tenantRuleExecRow, error) {
	var out tenantRuleExecRow
	var conditionJSON []byte
	// script_content no longer exists on catalog_validation_rules (dropped by
	// migrations/20260206_120100_drop_starlark_column.sql) - condition_json
	// is the only condition source now, evaluated via RuleFabric/CEL.
	q := `
		SELECT id, tenant_id, rule_name, rule_type, condition_json, severity,
		       core_rule_id, inherit_mode, core_version_pin, extension_script_content, extension_condition_json
		FROM catalog_validation_rules
		WHERE id = $1 AND tenant_id = $2 AND datasource_id = $3 AND is_active = true
	`
	err := db.QueryRowContext(ctx, q, id, tenantID, datasourceID).Scan(
		&out.rule.ID,
		&out.rule.TenantID,
		&out.rule.RuleName,
		&out.rule.RuleType,
		&conditionJSON,
		&out.rule.Severity,
		&out.coreRuleID,
		&out.inheritMode,
		&out.coreVersionPin,
		&out.extensionScript,
		&out.extensionCondition,
	)
	if err != nil {
		return nil, err
	}
	if len(conditionJSON) > 0 {
		if err := json.Unmarshal(conditionJSON, &out.rule.ConditionJSON); err != nil {
			return nil, fmt.Errorf("invalid condition json: %w", err)
		}
	}
	return &out, nil
}

type coreRuleResolved struct {
	RuleKey       string
	Version       int
	ConditionJSON json.RawMessage
}

// resolveCoreRuleScript resolves a core rule template's condition_json for a
// given (pinned or latest-active) version. Despite the name (kept to avoid
// touching its two call sites' signatures), it no longer resolves a
// script_content - CUE/Starlark execution has been retired in favor of
// evaluating condition_json (a tree or CEL expression) via RuleFabric.
func resolveCoreRuleScript(ctx context.Context, db *sql.DB, coreRuleID string, coreVersionPin *int) (*coreRuleResolved, error) {
	var ruleKey string
	var baseVersion int
	err := db.QueryRowContext(ctx, `SELECT rule_key, version FROM public.catalog_validation_rule_cores WHERE id = $1`, coreRuleID).Scan(&ruleKey, &baseVersion)
	if err != nil {
		return nil, err
	}

	if coreVersionPin != nil {
		var condition []byte
		var version int
		err := db.QueryRowContext(ctx, `
			SELECT version, condition_json
			FROM public.catalog_validation_rule_cores
			WHERE rule_key = $1 AND version = $2
		`, ruleKey, *coreVersionPin).Scan(&version, &condition)
		if err != nil {
			return nil, err
		}
		if len(condition) == 0 {
			return nil, errors.New("core rule has no condition_json")
		}
		return &coreRuleResolved{RuleKey: ruleKey, Version: version, ConditionJSON: condition}, nil
	}

	var condition []byte
	var version int
	err = db.QueryRowContext(ctx, `
		SELECT version, condition_json
		FROM public.catalog_validation_rule_cores
		WHERE rule_key = $1 AND status = 'active'
		ORDER BY version DESC
		LIMIT 1
	`, ruleKey).Scan(&version, &condition)
	if err != nil {
		// Fall back to the referenced version if no active exists.
		var c2 []byte
		var v2 int
		err2 := db.QueryRowContext(ctx, `SELECT version, condition_json FROM public.catalog_validation_rule_cores WHERE id = $1`, coreRuleID).Scan(&v2, &c2)
		if err2 != nil {
			return nil, err
		}
		if len(c2) == 0 {
			return nil, errors.New("core rule has no condition_json")
		}
		return &coreRuleResolved{RuleKey: ruleKey, Version: v2, ConditionJSON: c2}, nil
	}
	if len(condition) == 0 {
		return nil, errors.New("core rule has no condition_json")
	}
	return &coreRuleResolved{RuleKey: ruleKey, Version: version, ConditionJSON: condition}, nil
}

// evaluateValidationRuleCEL resolves the condition for a tenant validation
// rule (custom condition_json, or inherited/extended from a core template)
// and evaluates it via RuleFabric's CEL evaluator - the replacement for the
// retired CUE/Starlark execution path. Returns (isValid, message, err) where
// err is a system error (a compile/eval failure); isValid=false with a nil
// err is an ordinary rule failure, including "no condition configured".
func (h *validationRulesHandler) evaluateValidationRuleCEL(ctx context.Context, row *tenantRuleExecRow, record map[string]interface{}) (bool, string, error) {
	mode := "custom"
	if row.inheritMode.Valid {
		if m := strings.ToLower(strings.TrimSpace(row.inheritMode.String)); m != "" {
			mode = m
		}
	}

	tenantCondition, err := json.Marshal(row.rule.ConditionJSON)
	if err != nil {
		return false, "", fmt.Errorf("failed to marshal tenant condition: %w", err)
	}

	var conditionSrc json.RawMessage
	switch mode {
	case "inherit", "extend":
		if !row.coreRuleID.Valid {
			return false, "inherit_mode requires core_rule_id", nil
		}
		var pin *int
		if row.coreVersionPin.Valid {
			p := int(row.coreVersionPin.Int32)
			pin = &p
		}
		core, err := resolveCoreRuleScript(ctx, h.db, row.coreRuleID.String, pin)
		if err != nil {
			return false, fmt.Sprintf("failed to resolve core rule: %v", err), nil
		}
		if mode == "extend" && !isEmptyConditionJSON(row.extensionCondition) {
			combined, err := combineConditionsAND(core.ConditionJSON, row.extensionCondition)
			if err != nil {
				return false, fmt.Sprintf("failed to combine core and extension conditions: %v", err), nil
			}
			conditionSrc = combined
		} else {
			conditionSrc = core.ConditionJSON
		}
	default:
		conditionSrc = tenantCondition
	}

	if isEmptyConditionJSON(conditionSrc) {
		return false, "No condition configured for this rule (CUE/Starlark removed - author a condition in the visual rule builder)", nil
	}

	normalized, unsupported, err := rulefabric.NormalizeConditionJSONToCEL(conditionSrc)
	if err != nil {
		return false, "", fmt.Errorf("failed to compile condition: %w", err)
	}

	var celShape struct {
		Expression string `json:"expression"`
	}
	if err := json.Unmarshal(normalized, &celShape); err != nil {
		return false, "", fmt.Errorf("failed to parse compiled condition: %w", err)
	}

	valid, err := h.ruleEvaluator.EvaluateCELBoolean(celShape.Expression, record, nil, nil)
	if err != nil {
		return false, "", fmt.Errorf("CEL evaluation error: %w", err)
	}

	message := ""
	if !valid {
		message = "Condition not satisfied"
	}
	if len(unsupported) > 0 {
		message += " (warning: some conditions could not be fully translated - " + strings.Join(unsupported, "; ") + ")"
	}
	return valid, message, nil
}

func isEmptyConditionJSON(raw json.RawMessage) bool {
	if len(raw) == 0 {
		return true
	}
	var m map[string]interface{}
	if err := json.Unmarshal(raw, &m); err != nil {
		return true
	}
	return len(m) == 0
}

// combineConditionsAND normalizes two condition_json blobs (each may be a
// tree or already {"type":"cel",...}) to CEL and ANDs the resulting
// expressions - used for "extend" mode, where a tenant's own condition_json
// is an additional check layered on top of an inherited core rule.
func combineConditionsAND(a, b json.RawMessage) (json.RawMessage, error) {
	na, _, err := rulefabric.NormalizeConditionJSONToCEL(a)
	if err != nil {
		return nil, err
	}
	nb, _, err := rulefabric.NormalizeConditionJSONToCEL(b)
	if err != nil {
		return nil, err
	}
	var ea, eb struct {
		Expression string `json:"expression"`
	}
	if err := json.Unmarshal(na, &ea); err != nil {
		return nil, err
	}
	if err := json.Unmarshal(nb, &eb); err != nil {
		return nil, err
	}
	return json.Marshal(map[string]interface{}{
		"type":       "cel",
		"expression": fmt.Sprintf("(%s) && (%s)", ea.Expression, eb.Expression),
	})
}

// handleListValidationRules retrieves all validation rules for a tenant with facets and pagination
func (h *validationRulesHandler) handleListValidationRules() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		secCtx, _, err := handlers.SecurityContextFromRequest(r, "", "", h.securityDeps)
		if err != nil {
			writeJSONError(w, http.StatusUnauthorized, "Security context initialization failed", "auth_error", map[string]string{"error": err.Error()})
			return
		}

		tenantID := secCtx.TenantID
		datasourceID := secCtx.DatasourceID

		if datasourceID == "" {
			datasourceID = "none"
		}

		// Pagination parameters
		page := 1
		if p := r.URL.Query().Get("page"); p != "" {
			if parsed, err := strconv.Atoi(p); err == nil && parsed > 0 {
				page = parsed
			}
		}
		limit := 20
		if l := r.URL.Query().Get("limit"); l != "" {
			if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 && parsed <= 100 {
				limit = parsed
			}
		}

		offset := (page - 1) * limit

		// Optional filters - handle multiple values
		searchQuery := r.URL.Query().Get("search")
		ruleTypes := r.URL.Query()["rule_type"]
		severities := r.URL.Query()["severity"]
		targetEntities := r.URL.Query()["target_entity"]
		targetEntityIDs := r.URL.Query()["target_entity_id"]

		// NEW: Also check for "entities" parameter (sent by EntityDetailsPage by name)
		if len(targetEntities) == 0 {
			entitiesParam := r.URL.Query().Get("entities")
			if entitiesParam != "" {
				targetEntities = []string{entitiesParam}
			}
		}

		// NEW: Also check for "entity_ids" parameter (sent by EntityDetailsPage by UUID) - PREFERRED
		if len(targetEntityIDs) == 0 {
			entityIDsParam := r.URL.Query().Get("entity_ids")
			if entityIDsParam != "" {
				// Split by comma if multiple IDs are provided
				rawIDs := strings.Split(entityIDsParam, ",")
				for _, id := range rawIDs {
					trimmed := strings.TrimSpace(id)
					if _, err := uuid.Parse(trimmed); err == nil {
						targetEntityIDs = append(targetEntityIDs, trimmed)
					}
				}
			}
		}

		scopes := r.URL.Query()["scope"]
		ruleTypesFilter := r.URL.Query()["type"]
		isActive := r.URL.Query().Get("is_active")

		// NEW: Conflict detection filters
		conflictEntity := r.URL.Query().Get("entity")
		conflictField := r.URL.Query().Get("field")

		// Build WHERE clause
		whereClause := "WHERE tenant_id = $1"
		args := []interface{}{tenantID}
		argNum := 2

		if datasourceID != "" && datasourceID != "none" {
			whereClause += fmt.Sprintf(" AND datasource_id = $%d", argNum)
			args = append(args, datasourceID)
			argNum++
		}

		// Handle entity and field filtering for conflict detection
		if conflictEntity != "" && conflictField != "" {
			// For conflict detection, find rules targeting the same entity and field
			whereClause += ` AND target_entity = $` + fmt.Sprintf("%d", argNum)
			args = append(args, conflictEntity)
			argNum++

			// For field matching, check condition_json structure for field references
			// This is a simple approach - in production, might need more sophisticated parsing
			whereClause += ` AND (condition_json::text ILIKE $` + fmt.Sprintf("%d", argNum) + ` OR description ILIKE $` + fmt.Sprintf("%d", argNum+1) + `)`
			fieldPattern := "%" + conflictField + "%"
			args = append(args, fieldPattern, fieldPattern)
			argNum += 2
		}

		// NEW: Handle entity UUID filtering (PREFERRED over entity name filtering)
		if len(targetEntityIDs) > 0 {
			placeholders := make([]string, len(targetEntityIDs))
			for i, id := range targetEntityIDs {
				placeholders[i] = "$" + fmt.Sprintf("%d", argNum)
				args = append(args, strings.TrimSpace(id))
				argNum++
			}
			// Check if any of the selected entity UUIDs match the rule's target entity UUIDs
			// Using && operator for array overlap, with fallback to text array for backward compatibility
			whereClause += ` AND (ARRAY[` + strings.Join(placeholders, ",") + `]::uuid[] && COALESCE(target_entity_ids, ARRAY[]::uuid[]) OR COALESCE(target_entity_ids, ARRAY[]::uuid[]) = ARRAY[]::uuid[])`

		} else if len(targetEntities) > 0 {
			// Fallback: Handle entity NAME filtering with case-insensitivity support for legacy data
			var expandedEntities []string
			seen := make(map[string]bool)

			for _, e := range targetEntities {
				val := strings.TrimSpace(e)
				if val == "" {
					continue
				}

				// Add original
				if !seen[val] {
					expandedEntities = append(expandedEntities, val)
					seen[val] = true
				}

				// Add lowercase
				low := strings.ToLower(val)
				if !seen[low] {
					expandedEntities = append(expandedEntities, low)
					seen[low] = true
				}

				// Add uppercase
				up := strings.ToUpper(val)
				if !seen[up] {
					expandedEntities = append(expandedEntities, up)
					seen[up] = true
				}
			}

			placeholders := make([]string, len(expandedEntities))
			for i, e := range expandedEntities {
				placeholders[i] = "$" + fmt.Sprintf("%d", argNum)
				args = append(args, e)
				argNum++
			}
			// Check if any of the selected entities (or their case variants) match the rule's target entities
			// Using && operator for array overlap
			whereClause += ` AND ARRAY[` + strings.Join(placeholders, ",") + `]::text[] && COALESCE(target_entities, ARRAY[target_entity])`
		}

		if len(ruleTypes) > 0 {
			placeholders := make([]string, len(ruleTypes))
			for i, rt := range ruleTypes {
				placeholders[i] = "$" + fmt.Sprintf("%d", argNum)
				args = append(args, rt)
				argNum++
			}
			whereClause += ` AND rule_type = ANY(ARRAY[` + strings.Join(placeholders, ",") + `])`
		}

		if len(severities) > 0 {
			placeholders := make([]string, len(severities))
			for i, sv := range severities {
				placeholders[i] = "$" + fmt.Sprintf("%d", argNum)
				args = append(args, sv)
				argNum++
			}
			whereClause += ` AND severity = ANY(ARRAY[` + strings.Join(placeholders, ",") + `])`
		}

		if len(scopes) > 0 {
			// Handle global vs specific scope filtering
			globalIncluded := false
			specificIncluded := false
			for _, scope := range scopes {
				switch scope {
				case "global":
					globalIncluded = true
				case "specific":
					specificIncluded = true
				}
			}

			scopeConditions := []string{}
			if globalIncluded && !specificIncluded {
				scopeConditions = append(scopeConditions, `'global' = ANY(COALESCE(target_entities, ARRAY['global']))`)
			} else if !globalIncluded && specificIncluded {
				scopeConditions = append(scopeConditions, `'global' != ALL(COALESCE(target_entities, ARRAY['global']))`)
			} else {
				// no filter needed
			}

			if len(scopeConditions) > 0 {
				whereClause += ` AND (` + strings.Join(scopeConditions, " OR ") + `)`
			}
		}

		if len(ruleTypesFilter) > 0 {
			coreIncluded := false
			customIncluded := false
			for _, rt := range ruleTypesFilter {
				switch rt {
				case "core":
					coreIncluded = true
				case "custom":
					customIncluded = true
				}
			}

			if coreIncluded && !customIncluded {
				whereClause += ` AND is_core = true`
			} else if !coreIncluded && customIncluded {
				whereClause += ` AND is_core = false`
			}
		}

		if searchQuery != "" {
			whereClause += ` AND (rule_name ILIKE $` + fmt.Sprintf("%d", argNum) + ` OR description ILIKE $` + fmt.Sprintf("%d", argNum+1) + `)`
			searchPattern := "%" + searchQuery + "%"
			args = append(args, searchPattern, searchPattern)
			argNum += 2
		}

		switch isActive {
		case "true":
			whereClause += ` AND is_active = true`
		case "false":
			whereClause += ` AND is_active = false`
		}

		// Get total count
		countQuery := "SELECT COUNT(*) as count FROM catalog_validation_rules " + whereClause
		var totalCount int
		// Use all filter args for the count query
		err = h.db.QueryRow(countQuery, args...).Scan(&totalCount)
		if err != nil {
			// Include the query and arguments in the error details temporarily to aid debugging
			details := fmt.Sprintf("%s | query=%s | args=%v", err.Error(), countQuery, args)
			writeJSONError(w, http.StatusInternalServerError, "Failed to count rules", "query_error", details)
			return
		}

		// Save filter args for facet queries before appending limit/offset
		filterArgs := make([]interface{}, len(args))
		copy(filterArgs, args)

		// Get paginated rules
		query := `
			SELECT id, tenant_id, datasource_id, rule_name, rule_type, description, target_entity,
			       target_entity_id, target_entities, target_entity_ids, condition_json,
			       severity, COALESCE(is_active, true), COALESCE(is_core, false), created_by, created_at, updated_at
			FROM catalog_validation_rules
			` + whereClause + `
			ORDER BY rule_name
			LIMIT $` + fmt.Sprintf("%d", argNum) + ` OFFSET $` + fmt.Sprintf("%d", argNum+1)

		args = append(args, limit, offset)

		rows, err := h.db.Query(query, args...)
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, "Failed to query validation rules", "query_error", err.Error())
			return
		}
		defer rows.Close()

		var rules []ValidationRule
		for rows.Next() {
			var rule ValidationRule
			var conditionJSON []byte
			var targetEntities pq.StringArray
			var targetEntityIDs pq.StringArray
			var createdBy sql.NullString
			var datasourceID sql.NullString
			var targetEntityID sql.NullString
			err := rows.Scan(
				&rule.ID, &rule.TenantID, &datasourceID, &rule.RuleName, &rule.RuleType,
				&rule.Description, &rule.TargetEntity, &targetEntityID, &targetEntities, &targetEntityIDs,
				&conditionJSON, &rule.Severity, &rule.IsActive, &rule.IsCore, &createdBy, &rule.CreatedAt, &rule.UpdatedAt,
			)
			if err != nil {
				writeJSONError(w, http.StatusInternalServerError, "Failed to scan rule", "scan_error", err.Error())
				return
			}

			if err := json.Unmarshal(conditionJSON, &rule.ConditionJSON); err != nil {
				rule.ConditionJSON = make(map[string]interface{})
			}

			rule.TargetEntities = targetEntities
			rule.TargetEntityIDs = targetEntityIDs

			if createdBy.Valid {
				rule.CreatedBy = &createdBy.String
			}

			if datasourceID.Valid {
				rule.DatasourceID = datasourceID.String
			}

			if targetEntityID.Valid {
				rule.TargetEntityID = targetEntityID.String
			}

			rules = append(rules, rule)
		}

		if rules == nil {
			rules = []ValidationRule{}
		}

		// Get facet counts - calculate from UNFILTERED data (only tenant_id filter)
		facets := map[string][]map[string]interface{}{}

		// Base where clause for facets (only tenant and datasource filter)
		facetWhereClause := "WHERE tenant_id = $1 AND datasource_id = $2"
		facetArgs := []interface{}{tenantID, datasourceID}

		// Rule type facets from all rules
		ruleTypeFacets := []map[string]interface{}{}
		ruleTypeQuery := `
			SELECT DISTINCT rule_type, COUNT(*) as count 
			FROM catalog_validation_rules 
			` + facetWhereClause + `
			GROUP BY rule_type 
			ORDER BY count DESC
		`
		ruleTypeRows, err := h.db.Query(ruleTypeQuery, facetArgs...)
		if err == nil {
			defer ruleTypeRows.Close()
			for ruleTypeRows.Next() {
				var ruleType string
				var count int
				if err := ruleTypeRows.Scan(&ruleType, &count); err == nil {
					ruleTypeFacets = append(ruleTypeFacets, map[string]interface{}{
						"value": ruleType,
						"count": count,
					})
				}
			}
		}
		facets["rule_types"] = ruleTypeFacets

		// Severity facets from all rules
		severityFacets := []map[string]interface{}{}
		severityQuery := `
			SELECT DISTINCT severity, COUNT(*) as count 
			FROM catalog_validation_rules 
			` + facetWhereClause + `
			GROUP BY severity 
			ORDER BY count DESC
		`
		severityRows, err := h.db.Query(severityQuery, facetArgs...)
		if err == nil {
			defer severityRows.Close()
			for severityRows.Next() {
				var severity string
				var count int
				if err := severityRows.Scan(&severity, &count); err == nil {
					severityFacets = append(severityFacets, map[string]interface{}{
						"value": severity,
						"count": count,
					})
				}
			}
		}
		facets["severities"] = severityFacets

		// Target entity facets from all rules
		entityFacets := []map[string]interface{}{}
		entityQuery := `
			SELECT DISTINCT target_entity, COUNT(*) as count 
			FROM catalog_validation_rules 
			` + facetWhereClause + `
			GROUP BY target_entity 
			ORDER BY count DESC
		`
		entityRows, err := h.db.Query(entityQuery, facetArgs...)
		if err == nil {
			defer entityRows.Close()
			for entityRows.Next() {
				var entity string
				var count int
				if err := entityRows.Scan(&entity, &count); err == nil {
					entityFacets = append(entityFacets, map[string]interface{}{
						"value": entity,
						"count": count,
					})
				}
			}
		}
		facets["entities"] = entityFacets

		hasMore := (offset + limit) < totalCount

		// Build response
		response := map[string]interface{}{
			"rules":     rules,
			"total":     totalCount,
			"page":      page,
			"limit":     limit,
			"has_more":  hasMore,
			"facets":    facets,
			"timestamp": time.Now().UTC(),
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}
}

// handleGetValidationRule retrieves a single validation rule by ID
func (h *validationRulesHandler) handleGetValidationRule() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		secCtx, _, err := handlers.SecurityContextFromRequest(r, "", "", h.securityDeps)
		if err != nil {
			writeJSONError(w, http.StatusUnauthorized, "Security context initialization failed", "auth_error", map[string]string{"error": err.Error()})
			return
		}

		var rule ValidationRule
		var conditionJSON []byte
		var createdBy sql.NullString
		var datasourceID sql.NullString
		var targetEntityID sql.NullString
		var targetEntities pq.StringArray
		var targetEntityIDs pq.StringArray

		err = h.db.QueryRow(`
			SELECT id, tenant_id, datasource_id, rule_name, rule_type, description, target_entity,
			       target_entity_id, target_entities, target_entity_ids, condition_json,
			       severity, is_active, is_core, created_by, created_at, updated_at
			FROM catalog_validation_rules
			WHERE id = $1 AND tenant_id = $2
		`, id, secCtx.TenantID).Scan(
			&rule.ID, &rule.TenantID, &datasourceID, &rule.RuleName, &rule.RuleType, &rule.Description, &rule.TargetEntity,
			&targetEntityID, &targetEntities, &targetEntityIDs, &conditionJSON,
			&rule.Severity, &rule.IsActive, &rule.IsCore, &createdBy, &rule.CreatedAt, &rule.UpdatedAt,
		)

		if err == sql.ErrNoRows {
			writeJSONError(w, http.StatusNotFound, "Validation rule not found", "not_found", "")
			return
		}
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, "Failed to query rule", "query_error", err.Error())
			return
		}

		if err := json.Unmarshal(conditionJSON, &rule.ConditionJSON); err != nil {
			rule.ConditionJSON = make(map[string]interface{})
		}

		if createdBy.Valid {
			rule.CreatedBy = &createdBy.String
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(rule)
	}
}

// handleCreateValidationRule creates a new validation rule
func (h *validationRulesHandler) handleCreateValidationRule() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		secCtx, _, err := handlers.SecurityContextFromRequest(r, "", "", h.securityDeps)
		if err != nil {
			writeJSONError(w, http.StatusUnauthorized, "Security context initialization failed", "auth_error", map[string]string{"error": err.Error()})
			return
		}

		tenantID := secCtx.TenantID
		datasourceID := secCtx.DatasourceID

		var req ValidationRuleRequest
		bodyBytes, readErr := io.ReadAll(r.Body)
		if readErr != nil {
			writeJSONError(w, http.StatusBadRequest, "Invalid request body", "decode_error", readErr.Error())
			return
		}
		// fmt.Fprintf(os.Stderr, "[POST-DEBUG] Raw body: %s\n", string(bodyBytes))
		unmarshalErr := json.Unmarshal(bodyBytes, &req)

		// Safe IsCore parsing
		isCoreValue := false
		if req.IsCore != nil {
			switch v := req.IsCore.(type) {
			case bool:
				isCoreValue = v
			case string:
				isCoreValue = v == "true" || v == "True" || v == "1"
			case float64:
				isCoreValue = v == 1
			}
		}
		// fmt.Fprintf(os.Stderr, "[DEBUG] isCore parsed to: %v\n", isCoreValue)

		if unmarshalErr != nil {
			fmt.Fprintf(os.Stderr, "[DEBUG] JSON unmarshal failed: %v\n", unmarshalErr)
			// Attempt to tolerate non-JSON payloads sent by clients (form-encoded or "key:val" style)
			// Try url-encoded form first
			parsed := false
			if vals, perr := url.ParseQuery(string(bodyBytes)); perr == nil && len(vals) > 0 {
				if req.Parameters == nil {
					req.Parameters = map[string]interface{}{}
				}
				if v := vals.Get("name"); v != "" {
					req.Name = v
				}
				if v := vals.Get("ruleType"); v != "" {
					req.RuleTypeCC = v
				}
				// scope may be repeated
				if scopes, ok := vals["scope"]; ok && len(scopes) > 0 {
					req.ScopeArray = pq.StringArray(scopes)
				}
				// Copy other keys into Parameters for best-effort
				for k, vs := range vals {
					// Handle keys in the form "key:value" that ParseQuery treats as a single key
					if strings.Contains(k, ":") && len(vs) == 0 {
						kv := strings.SplitN(k, ":", 2)
						kk := strings.TrimSpace(kv[0])
						vv := strings.TrimSpace(kv[1])
						switch kk {
						case "name":
							req.Name = vv
						case "ruleType":
							req.RuleTypeCC = vv
						case "scope":
							req.ScopeArray = pq.StringArray{vv}
						default:
							req.Parameters[kk] = vv
						}
						continue
					}
					if k == "name" || k == "ruleType" || k == "scope" {
						continue
					}
					if len(vs) == 1 {
						req.Parameters[k] = vs[0]
					} else {
						arr := make([]string, len(vs))
						copy(arr, vs)
						req.Parameters[k] = arr
					}
				}
				fmt.Fprintf(os.Stderr, "[POST-DEBUG] Parsed urlencoded body: %v\n", vals)
				parsed = true
			}

			if !parsed {
				// Fallback: parse simple "key:val" or newline-delimited key:value pairs
				s := string(bodyBytes)
				if req.Parameters == nil {
					req.Parameters = map[string]interface{}{}
				}
				parts := strings.FieldsFunc(s, func(r rune) bool { return r == '&' || r == '\n' || r == '\r' || r == ' ' })
				for _, part := range parts {
					if strings.Contains(part, ":") {
						kv := strings.SplitN(part, ":", 2)
						k := strings.TrimSpace(kv[0])
						v := strings.TrimSpace(kv[1])
						switch k {
						case "name":
							req.Name = v
						case "ruleType":
							req.RuleTypeCC = v
						case "scope":
							req.ScopeArray = append(req.ScopeArray, v)
						default:
							req.Parameters[k] = v
						}
					}
				}
				fmt.Fprintf(os.Stderr, "[POST-DEBUG] Parsed colon-delimited body -> name=%s ruleType=%s scope=%v params=%v\n", req.Name, req.RuleTypeCC, req.ScopeArray, req.Parameters)
				// If still nothing parsed, return original JSON error
				if req.Name == "" && req.RuleTypeCC == "" && len(req.Parameters) == 0 {
					writeJSONError(w, http.StatusBadRequest, "Invalid request body", "decode_error", unmarshalErr.Error())
					return
				}
			}
		} else {
			fmt.Fprintf(os.Stderr, "[DEBUG] JSON unmarshal succeeded, req.IsCore=%v\n", req.IsCore)
		}

		// Normalize frontend camelCase fields to backend snake_case
		if req.RuleName == "" && req.Name != "" {
			req.RuleName = req.Name
		}
		if req.RuleType == "" && req.RuleTypeCC != "" {
			req.RuleType = req.RuleTypeCC
		}
		if req.TargetEntity == "" && len(req.ScopeArray) > 0 {
			// Use first scope entry as target_entity if not provided
			req.TargetEntity = req.ScopeArray[0]
		}

		// Generate condition_json from parameters if not provided
		if len(req.ConditionJSON) == 0 && len(req.Parameters) > 0 {
			req.ConditionJSON = req.Parameters
		}

		// Validate required fields
		if req.RuleName == "" || req.RuleType == "" || req.TargetEntity == "" {
			writeJSONError(w, http.StatusBadRequest, "Missing required fields", "validation_error", fmt.Sprintf("Missing: ruleName=%s, ruleType=%s, targetEntity=%s, conditionJSON=%v", req.RuleName, req.RuleType, req.TargetEntity, req.ConditionJSON))
			return
		}

		// Validate rule type
		// Normalize rule_type aliases and casing (accept some frontend variants)
		rt := strings.TrimSpace(req.RuleType)
		rt = strings.ToLower(rt)
		switch rt {
		case "validation":
			rt = "business_logic"
		case "businesslogic":
			rt = "business_logic"
		case "fieldformat":
			rt = "field_format"
		case "referentialintegrity":
			rt = "referential_integrity"

		}
		req.RuleType = rt

		validTypes := map[string]bool{
			"field_format": true, "cardinality": true, "uniqueness": true,
			"referential_integrity": true, "business_logic": true, "cue": true,
		}
		if !validTypes[req.RuleType] {
			writeJSONError(w, http.StatusBadRequest, "Invalid rule_type", "validation_error", "rule_type must be one of: field_format, cardinality, uniqueness, referential_integrity, business_logic, cue")
			return
		}

		// Validate severity
		if req.Severity == "" {
			req.Severity = "error"
		}
		validSeverities := map[string]bool{"error": true, "warning": true, "info": true}
		if !validSeverities[req.Severity] {
			writeJSONError(w, http.StatusBadRequest, "Invalid severity", "validation_error", "severity must be one of: error, warning, info")
			return
		}

		isActive := true
		if req.IsActive != nil {
			isActive = *req.IsActive
		}

		isCore := isCoreValue
		fmt.Fprintf(os.Stderr, "[DEBUG] Final isCore at create: %v\n", isCore)

		id := uuid.New().String()
		conditionJSON := []byte("{}")
		if len(req.ConditionJSON) > 0 {
			conditionJSON, _ = json.Marshal(req.ConditionJSON)
		}

		// Set up target_entities array
		targetEntities := req.TargetEntities
		if len(targetEntities) == 0 {
			// If no target_entities provided, default to 'global' or use legacy single entity
			if req.TargetEntity != "" {
				targetEntities = pq.StringArray{req.TargetEntity}
			} else {
				targetEntities = pq.StringArray{"global"}
			}
		}

		var createdBy sql.NullString
		now := time.Now()

		var retrievedTargetEntities pq.StringArray
		var createdAt, updatedAt time.Time

		err = h.db.QueryRow(`
			INSERT INTO catalog_validation_rules (
				id, tenant_id, datasource_id, rule_name, rule_type, description, target_entity,
				target_entities, condition_json, severity, is_active, is_core, created_at, updated_at
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
			RETURNING id, tenant_id, datasource_id, rule_name, rule_type, description, target_entity,
			          target_entities, condition_json, severity, is_active, is_core, created_by, created_at, updated_at
		`, id, tenantID, datasourceID, req.RuleName, req.RuleType, req.Description, req.TargetEntity,
			targetEntities, conditionJSON, req.Severity, isActive, isCore, now, now).Scan(
			&id, &tenantID, &datasourceID, &req.RuleName, &req.RuleType, &req.Description, &req.TargetEntity,
			&retrievedTargetEntities, &conditionJSON, &req.Severity, &isActive, &isCore, &createdBy, &createdAt, &updatedAt,
		)

		if err != nil {
			// Check for duplicate key error (Postgres 23505)
			// Explicit check for *pq.Error or text match if wrapped
			isDuplicate := false
			if pqErr, ok := err.(*pq.Error); ok && pqErr.Code == "23505" {
				isDuplicate = true
			} else if strings.Contains(err.Error(), "23505") || strings.Contains(err.Error(), "duplicate key") {
				isDuplicate = true
			}

			if isDuplicate {
				// Lookup the conflicting rule to provide better feedback
				var conflictEntity string
				var conflictID string
				_ = h.db.QueryRow(`
					SELECT target_entity, id 
					FROM catalog_validation_rules 
					WHERE tenant_id = $1 AND datasource_id = $2 AND rule_name = $3
				`, tenantID, datasourceID, req.RuleName).Scan(&conflictEntity, &conflictID)

				detailMsg := ""
				if conflictEntity != "" {
					detailMsg = fmt.Sprintf("Rule name '%s' is already used by a rule on entity '%s' (ID: %s)", req.RuleName, conflictEntity, conflictID)
				}

				writeJSONError(w, http.StatusConflict, "Rule name already exists for this tenant", "duplicate_rule", detailMsg)
				return
			}
			writeJSONError(w, http.StatusInternalServerError, "Failed to create validation rule", "create_error", err.Error())
			return
		}

		var createdByPtr *string
		if createdBy.Valid {
			createdByPtr = &createdBy.String
		}

		var condObj map[string]interface{}
		json.Unmarshal(conditionJSON, &condObj)

		rule := ValidationRule{
			ID:             id,
			TenantID:       tenantID,
			DatasourceID:   datasourceID,
			RuleName:       req.RuleName,
			RuleType:       req.RuleType,
			Description:    req.Description,
			TargetEntity:   req.TargetEntity,
			TargetEntities: retrievedTargetEntities,
			ConditionJSON:  condObj,
			Severity:       req.Severity,
			IsActive:       isActive,
			IsCore:         isCore,
			CreatedBy:      createdByPtr,
			CreatedAt:      createdAt,
			UpdatedAt:      updatedAt,
			ScriptContent:  req.ScriptContent,
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(rule)
	}
}

// handleUpdateValidationRule updates an existing validation rule
func (h *validationRulesHandler) handleUpdateValidationRule() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		secCtx, _, err := handlers.SecurityContextFromRequest(r, "", "", h.securityDeps)
		if err != nil {
			writeJSONError(w, http.StatusUnauthorized, "Security context initialization failed", "auth_error", map[string]string{"error": err.Error()})
			return
		}

		var req ValidationRuleRequest
		bodyBytes, readErr := io.ReadAll(r.Body)
		if readErr != nil {
			writeJSONError(w, http.StatusBadRequest, "Invalid request body", "decode_error", readErr.Error())
			return
		}
		fmt.Fprintf(os.Stderr, "[PATCH-DEBUG] Raw body: %s\n", string(bodyBytes))
		if err := json.Unmarshal(bodyBytes, &req); err != nil {
			writeJSONError(w, http.StatusBadRequest, "Invalid request body", "decode_error", err.Error())
			return
		}

		// Normalize frontend camelCase fields to backend snake_case (same as POST handler)
		if req.RuleName == "" && req.Name != "" {
			req.RuleName = req.Name
		}
		if req.RuleType == "" && req.RuleTypeCC != "" {
			req.RuleType = req.RuleTypeCC
		}
		if req.TargetEntity == "" && len(req.ScopeArray) > 0 {
			req.TargetEntity = req.ScopeArray[0]
		}
		if len(req.ConditionJSON) == 0 && len(req.Parameters) > 0 {
			req.ConditionJSON = req.Parameters
		}

		// Normalize rule_type aliases and casing (accept some frontend variants)
		rt := strings.TrimSpace(req.RuleType)
		rt = strings.ToLower(rt)
		switch rt {
		case "validation":
			rt = "business_logic"
		case "businesslogic":
			rt = "business_logic"
		case "fieldformat":
			rt = "field_format"
		case "referentialintegrity":
			rt = "referential_integrity"
		}
		req.RuleType = rt

		// Get current rule for audit
		var oldConditionJSON []byte
		err = h.db.QueryRow(`
			SELECT condition_json FROM catalog_validation_rules
			WHERE id = $1 AND tenant_id = $2
		`, id, secCtx.TenantID).Scan(&oldConditionJSON)

		if err == sql.ErrNoRows {
			writeJSONError(w, http.StatusNotFound, "Validation rule not found", "not_found", "")
			return
		}
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, "Failed to query rule", "query_error", err.Error())
			return
		}

		// Update the rule
		conditionJSON, _ := json.Marshal(req.ConditionJSON)
		isActive := true
		if req.IsActive != nil {
			isActive = *req.IsActive
		}

		isCore := req.IsCore == "true"

		// Set up target_entities array
		targetEntities := req.TargetEntities
		if len(targetEntities) == 0 {
			// If no target_entities provided, use legacy single entity or keep existing
			if req.TargetEntity != "" {
				targetEntities = pq.StringArray{req.TargetEntity}
			} else {
				targetEntities = pq.StringArray{"global"}
			}
		}

		var updatedRule ValidationRule
		var createdBy sql.NullString
		var retrievedTargetEntities pq.StringArray
		err = h.db.QueryRow(`
			UPDATE catalog_validation_rules
			SET rule_name = $1, rule_type = $2, description = $3, target_entity = $4,
			    target_entities = $5, condition_json = $6, severity = $7, is_active = $8, is_core = $9, updated_at = CURRENT_TIMESTAMP
			WHERE id = $10 AND tenant_id = $11
			RETURNING id, tenant_id, rule_name, rule_type, description, target_entity,
			          target_entities, condition_json, severity, is_active, is_core, created_by, created_at, updated_at
		`, req.RuleName, req.RuleType, req.Description, req.TargetEntity,
			targetEntities, conditionJSON, req.Severity, isActive, isCore, id, secCtx.TenantID).Scan(
			&updatedRule.ID, &updatedRule.TenantID, &updatedRule.RuleName, &updatedRule.RuleType,
			&updatedRule.Description, &updatedRule.TargetEntity, &retrievedTargetEntities, &conditionJSON,
			&updatedRule.Severity, &updatedRule.IsActive, &updatedRule.IsCore, &createdBy, &updatedRule.CreatedAt, &updatedRule.UpdatedAt,
		)

		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, "Failed to update validation rule", "update_error", err.Error())
			return
		}

		if err := json.Unmarshal(conditionJSON, &updatedRule.ConditionJSON); err != nil {
			updatedRule.ConditionJSON = make(map[string]interface{})
		}

		updatedRule.TargetEntities = retrievedTargetEntities

		if createdBy.Valid {
			updatedRule.CreatedBy = &createdBy.String
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(updatedRule)
	}
}

// handleDeleteValidationRule deletes a validation rule
func (h *validationRulesHandler) handleDeleteValidationRule() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")

		secCtx, _, err := handlers.SecurityContextFromRequest(r, "", "", h.securityDeps)
		if err != nil {
			writeJSONError(w, http.StatusUnauthorized, "Security context initialization failed", "auth_error", map[string]string{"error": err.Error()})
			return
		}

		result, err := h.db.Exec(`
			DELETE FROM catalog_validation_rules
			WHERE id = $1 AND tenant_id = $2
		`, id, secCtx.TenantID)

		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, "Failed to delete validation rule", "delete_error", err.Error())
			return
		}

		rowsAffected, err := result.RowsAffected()
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, "Failed to verify deletion", "delete_error", err.Error())
			return
		}

		if rowsAffected == 0 {
			writeJSONError(w, http.StatusNotFound, "Validation rule not found", "not_found", "")
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNoContent)
	}
}

// handleExecuteValidationRule executes a validation rule against a provided record.
//
// Supports tenant rule inheritance from core templates:
// - inherit: evaluate the pinned/latest core template script
// - extend: evaluate core; if pass, evaluate extension script
// - override/custom: evaluate tenant script_content
// handleExecuteValidationRule executes a validation rule against a provided record.
func (h *validationRulesHandler) handleExecuteValidationRule() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		secCtx, _, err := handlers.SecurityContextFromRequest(r, "", "", h.securityDeps)
		if err != nil {
			writeJSONError(w, http.StatusUnauthorized, "Security context initialization failed", "auth_error", map[string]string{"error": err.Error()})
			return
		}

		tenantID := secCtx.TenantID
		datasourceID := secCtx.DatasourceID

		var req struct {
			Record map[string]interface{} `json:"record"`
			Data   map[string]interface{} `json:"data"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSONError(w, http.StatusBadRequest, "Invalid request body", "decode_error", err.Error())
			return
		}
		record := req.Record
		if record == nil {
			record = req.Data
		}
		if record == nil {
			writeJSONError(w, http.StatusBadRequest, "record is required", "missing_record", "")
			return
		}

		row, err := fetchTenantRuleForExecution(r.Context(), h.db, id, tenantID, datasourceID)
		if err == sql.ErrNoRows {
			writeJSONError(w, http.StatusNotFound, "Validation rule not found or is inactive", "not_found", "")
			return
		}
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, "Failed to query rule", "query_error", err.Error())
			return
		}

		// Enforce CUE or Business Logic (rule_type naming kept for backward
		// compatibility; both now evaluate condition_json via RuleFabric/CEL,
		// not a CUE script)
		if row.rule.RuleType != "cue" && row.rule.RuleType != "business_logic" {
			writeJSONError(w, http.StatusBadRequest, "Unsupported rule type", "unsupported_type", "")
			return
		}

		valid, message, err := h.evaluateValidationRuleCEL(r.Context(), row, record)
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, "Evaluation failed", "eval_error", err.Error())
			return
		}

		result := ValidationRuleExecutionResult{
			RuleID:    row.rule.ID,
			RuleName:  row.rule.RuleName,
			RuleType:  row.rule.RuleType,
			Severity:  row.rule.Severity,
			Status:    "fail",
			Message:   message,
			Timestamp: time.Now(),
		}
		if valid {
			result.Status = "pass"
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(result)
	}
}

// handleExecuteValidationRulesBatch execution removed as it relied on Starlark logic mismatch/complexity.
// If needed, implement with CUE support.

// handleGetValidationRuleAudit retrieves audit history for a validation rule
func (h *validationRulesHandler) handleGetValidationRuleAudit() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		secCtx, _, err := handlers.SecurityContextFromRequest(r, "", "", h.securityDeps)
		if err != nil {
			writeJSONError(w, http.StatusUnauthorized, "Security context initialization failed", "auth_error", map[string]string{"error": err.Error()})
			return
		}

		tenantID := secCtx.TenantID

		rows, err := h.db.Query(`
			SELECT id, rule_id, tenant_id, action, old_values, new_values, changed_by, changed_at
			FROM catalog_validation_rules_audit
			WHERE rule_id = $1 AND tenant_id = $2
			ORDER BY changed_at DESC
		`, id, tenantID)

		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, "Failed to query audit records", "query_error", err.Error())
			return
		}
		defer rows.Close()

		type AuditRecord struct {
			ID        string                 `json:"id"`
			RuleID    string                 `json:"rule_id"`
			TenantID  string                 `json:"tenant_id"`
			Action    string                 `json:"action"`
			OldValues map[string]interface{} `json:"old_values,omitempty"`
			NewValues map[string]interface{} `json:"new_values,omitempty"`
			ChangedBy *string                `json:"changed_by,omitempty"`
			ChangedAt time.Time              `json:"changed_at"`
		}

		var records []AuditRecord
		for rows.Next() {
			var record AuditRecord
			var oldValuesJSON, newValuesJSON []byte
			var changedBy sql.NullString
			err := rows.Scan(
				&record.ID, &record.RuleID, &record.TenantID, &record.Action,
				&oldValuesJSON, &newValuesJSON, &changedBy, &record.ChangedAt,
			)
			if err != nil {
				continue
			}

			if len(oldValuesJSON) > 0 {
				json.Unmarshal(oldValuesJSON, &record.OldValues)
			}
			if len(newValuesJSON) > 0 {
				json.Unmarshal(newValuesJSON, &record.NewValues)
			}
			if changedBy.Valid {
				record.ChangedBy = &changedBy.String
			}

			records = append(records, record)
		}

		if records == nil {
			records = []AuditRecord{}
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(records)
	}
}

// handleSimulateValidationRule executes a script against provided data for live preview

// handleSimulateValidationRuleWithInstance executes a validation rule against a business object instance
func (h *validationRulesHandler) handleSimulateValidationRuleWithInstance() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ruleID := chi.URLParam(r, "id")
		secCtx, _, err := handlers.SecurityContextFromRequest(r, "", "", h.securityDeps)
		if err != nil {
			writeJSONError(w, http.StatusUnauthorized, "Security context initialization failed", "auth_error", map[string]string{"error": err.Error()})
			return
		}

		tenantID := secCtx.TenantID
		datasourceID := secCtx.DatasourceID

		var req struct {
			InstanceID string `json:"instance_id"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSONError(w, http.StatusBadRequest, "Invalid request body", "decode_error", err.Error())
			return
		}

		if req.InstanceID == "" {
			writeJSONError(w, http.StatusBadRequest, "instance_id is required", "missing_instance_id", "")
			return
		}

		// Try to use the services.BusinessObjectService if available
		if boService, ok := h.boService.(*services.BusinessObjectService); ok {
			// Get instance data formatted for validation
			record, err := boService.GetInstanceForValidation(r.Context(), tenantID, req.InstanceID)
			if err != nil {
				if err.Error() == "instance not found" {
					writeJSONError(w, http.StatusNotFound, "Business object instance not found", "not_found", err.Error())
				} else {
					writeJSONError(w, http.StatusInternalServerError, "Failed to retrieve instance", "retrieval_error", err.Error())
				}
				return
			}

			// Get the validation rule
			row, err := fetchTenantRuleForExecution(r.Context(), h.db, ruleID, tenantID, datasourceID)
			if err == sql.ErrNoRows {
				writeJSONError(w, http.StatusNotFound, "Validation rule not found or is inactive", "not_found", "")
				return
			}
			if err != nil {
				writeJSONError(w, http.StatusInternalServerError, "Failed to query rule", "query_error", err.Error())
				return
			}

			valid, message, evalErr := h.evaluateValidationRuleCEL(r.Context(), row, record)
			res := &services.CueValidationResult{IsValid: valid, Message: message, Severity: "error"}
			if evalErr != nil {
				res = &services.CueValidationResult{IsValid: false, Message: evalErr.Error(), Severity: "error"}
			}

			result := map[string]interface{}{
				"rule_id":     row.rule.ID,
				"rule_name":   row.rule.RuleName,
				"rule_type":   row.rule.RuleType,
				"severity":    row.rule.Severity,
				"status":      "fail",
				"message":     "",
				"timestamp":   time.Now(),
				"instance_id": req.InstanceID,
				"data_used":   record,
			}
			if res != nil {
				if res.IsValid {
					result["status"] = "pass"
				}
				result["message"] = res.Message
			}

			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(result)
			return
		}

		// Fallback: service type not supported
		writeJSONError(w, http.StatusInternalServerError, "Business object service not available", "service_error", "")
	}
}

type sqlDBBORepository struct {
	db *sql.DB
}

func (s *sqlDBBORepository) GetBODefinition(boID string) (*boresolver.BODefinition, error) {
	var drivingTable string
	var datasourceID string
	err := s.db.QueryRow(`SELECT COALESCE(driving_table, technical_name, name), datasource_id::text FROM public.business_objects WHERE id::text = $1`, boID).Scan(&drivingTable, &datasourceID)
	if err != nil {
		drivingTable = "target_entity"
	}

	rows, err := s.db.Query(`SELECT id::text, field_name, display_name, physical_column FROM public.bo_fields WHERE business_object_id::text = $1`, boID)
	var fields []boresolver.BOField
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var f boresolver.BOField
			var pCol sql.NullString
			if err := rows.Scan(&f.ID, &f.Name, &f.DisplayName, &pCol); err == nil {
				if pCol.Valid {
					f.PhysicalColumn = pCol.String
				} else {
					f.PhysicalColumn = f.Name
				}
				fields = append(fields, f)
			}
		}
	}

	return &boresolver.BODefinition{
		ID:           boID,
		DrivingTable: drivingTable,
		DatasourceID: datasourceID,
		Fields:       fields,
	}, nil
}

func (s *sqlDBBORepository) GetBOByTechnicalName(technicalName, tenantID, datasourceID string) (*boresolver.BODefinition, error) {
	var boID string
	err := s.db.QueryRow(`SELECT id::text FROM public.business_objects WHERE (technical_name = $1 OR name = $1) AND (tenant_id::text = $2 OR tenant_id IS NULL) LIMIT 1`, technicalName, tenantID).Scan(&boID)
	if err != nil {
		return nil, err
	}
	return s.GetBODefinition(boID)
}

func (h *validationRulesHandler) handleExecuteValidationRuleBinding() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		secCtx, _, err := handlers.SecurityContextFromRequest(r, "", "", h.securityDeps)
		if err != nil {
			writeJSONError(w, http.StatusUnauthorized, "Security context initialization failed", "auth_error", map[string]string{"error": err.Error()})
			return
		}

		var req boresolver.ValidationRuleCompilationRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSONError(w, http.StatusBadRequest, "Invalid request body", "invalid_json", err.Error())
			return
		}

		if req.TenantID == "" {
			req.TenantID = secCtx.TenantID
		}

		// Rule 1.3 Defense: Safe UUID validation
		if req.BusinessObjectID != "" {
			if _, err := uuid.Parse(req.BusinessObjectID); err != nil {
				writeJSONError(w, http.StatusBadRequest, "Invalid businessObjectId format", "invalid_uuid", err.Error())
				return
			}
		} else {
			writeJSONError(w, http.StatusBadRequest, "businessObjectId is required", "missing_param", nil)
			return
		}

		if req.TenantID != "" {
			if _, err := uuid.Parse(req.TenantID); err != nil {
				writeJSONError(w, http.StatusBadRequest, "Invalid tenantId format", "invalid_uuid", err.Error())
				return
			}
		}

		repo := &sqlDBBORepository{db: h.db}
		generator, err := boresolver.NewBOSQLGenerator(repo, "postgres")
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, "Failed to initialize SQL generator", "generator_error", err.Error())
			return
		}

		compiled, err := generator.CompileValidationRuleSQL(req)
		if err != nil {
			writeJSONError(w, http.StatusBadRequest, fmt.Sprintf("Failed to compile rule into binding SQL: %v", err), "compile_error", err.Error())
			return
		}

		startTime := time.Now()

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"status":          "compiled",
			"compiled_sql":    compiled.SQL,
			"args":            compiled.Args,
			"physical_column": compiled.PhysicalColumn,
			"execution_time":  time.Since(startTime).String(),
		})
	}
}

