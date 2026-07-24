package api

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/lib/pq"
)

// CoreValidationRule is a tenant-agnostic, versioned template.
type CoreValidationRule struct {
	ID              string                 `json:"id"`
	RuleKey         string                 `json:"rule_key"`
	Version         int                    `json:"version"`
	RuleName        string                 `json:"rule_name"`
	RuleType        string                 `json:"rule_type"`
	Description     string                 `json:"description,omitempty"`
	TargetEntity    string                 `json:"target_entity,omitempty"`
	TargetEntityID  string                 `json:"target_entity_id,omitempty"`
	TargetEntities  pq.StringArray         `json:"target_entities,omitempty"`
	TargetEntityIDs pq.StringArray         `json:"target_entity_ids,omitempty"`
	ConditionJSON   map[string]interface{} `json:"condition_json,omitempty"`
	ScriptContent   string                 `json:"script_content,omitempty"`
	Severity        string                 `json:"severity"`
	Status          string                 `json:"status"` // draft|active|deprecated
	IsCoreLocked    bool                   `json:"is_core_locked"`
	CreatedBy       *string                `json:"created_by,omitempty"`
	CreatedAt       time.Time              `json:"created_at"`
	UpdatedAt       time.Time              `json:"updated_at"`
}

type coreValidationRuleRequest struct {
	RuleKey         string                 `json:"rule_key"`
	Version         *int                   `json:"version,omitempty"`
	RuleName        string                 `json:"rule_name"`
	RuleType        string                 `json:"rule_type"`
	Description     string                 `json:"description"`
	TargetEntity    string                 `json:"target_entity"`
	TargetEntityID  string                 `json:"target_entity_id"`
	TargetEntities  pq.StringArray         `json:"target_entities"`
	TargetEntityIDs pq.StringArray         `json:"target_entity_ids"`
	ConditionJSON   map[string]interface{} `json:"condition_json"`
	ScriptContent   string                 `json:"script_content"`
	Severity        string                 `json:"severity"`
	Status          string                 `json:"status"`
	IsCoreLocked    *bool                  `json:"is_core_locked"`
	CreatedBy       string                 `json:"created_by"`
}

func handleListValidationRuleCores(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ruleKey := strings.TrimSpace(r.URL.Query().Get("rule_key"))
		status := strings.TrimSpace(r.URL.Query().Get("status"))
		if status == "" {
			status = "active"
		}

		q := `
			SELECT id, rule_key, version, rule_name, rule_type, description,
			       target_entity, target_entity_id, target_entities, target_entity_ids,
			       condition_json, script_content, severity, status, is_core_locked,
			       created_by, created_at, updated_at
			FROM public.catalog_validation_rule_cores
			WHERE ($1 = '' OR rule_key = $1)
			  AND ($2 = '' OR status = $2)
			ORDER BY rule_key, version DESC
		`

		rows, err := db.Query(q, ruleKey, status)
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, "Failed to query core rules", "query_error", err.Error())
			return
		}
		defer rows.Close()

		out := make([]CoreValidationRule, 0)
		for rows.Next() {
			var cr CoreValidationRule
			var condBytes []byte
			var script sql.NullString
			var createdBy sql.NullString
			var targetEntity sql.NullString
			var targetEntityID sql.NullString
			var targetEntities pq.StringArray
			var targetEntityIDs pq.StringArray

			if err := rows.Scan(
				&cr.ID, &cr.RuleKey, &cr.Version, &cr.RuleName, &cr.RuleType, &cr.Description,
				&targetEntity, &targetEntityID, &targetEntities, &targetEntityIDs,
				&condBytes, &script, &cr.Severity, &cr.Status, &cr.IsCoreLocked,
				&createdBy, &cr.CreatedAt, &cr.UpdatedAt,
			); err != nil {
				writeJSONError(w, http.StatusInternalServerError, "Failed to scan core rule", "scan_error", err.Error())
				return
			}

			if targetEntity.Valid {
				cr.TargetEntity = targetEntity.String
			}
			if targetEntityID.Valid {
				cr.TargetEntityID = targetEntityID.String
			}
			cr.TargetEntities = targetEntities
			cr.TargetEntityIDs = targetEntityIDs
			if script.Valid {
				cr.ScriptContent = script.String
			}
			if createdBy.Valid {
				cr.CreatedBy = &createdBy.String
			}
			if len(condBytes) > 0 {
				var m map[string]interface{}
				if err := json.Unmarshal(condBytes, &m); err == nil {
					cr.ConditionJSON = m
				}
			}

			out = append(out, cr)
		}

		if err := rows.Err(); err != nil {
			writeJSONError(w, http.StatusInternalServerError, "Failed to read core rules", "query_error", err.Error())
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"rules": out,
			"total": len(out),
		})
	}
}

func handleGetValidationRuleCore(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		if _, err := uuid.Parse(id); err != nil {
			writeJSONError(w, http.StatusBadRequest, "Invalid core rule id", "validation_error", err.Error())
			return
		}

		q := `
			SELECT id, rule_key, version, rule_name, rule_type, description,
			       target_entity, target_entity_id, target_entities, target_entity_ids,
			       condition_json, script_content, severity, status, is_core_locked,
			       created_by, created_at, updated_at
			FROM public.catalog_validation_rule_cores
			WHERE id = $1
		`

		var cr CoreValidationRule
		var condBytes []byte
		var script sql.NullString
		var createdBy sql.NullString
		var targetEntity sql.NullString
		var targetEntityID sql.NullString
		var targetEntities pq.StringArray
		var targetEntityIDs pq.StringArray

		err := db.QueryRow(q, id).Scan(
			&cr.ID, &cr.RuleKey, &cr.Version, &cr.RuleName, &cr.RuleType, &cr.Description,
			&targetEntity, &targetEntityID, &targetEntities, &targetEntityIDs,
			&condBytes, &script, &cr.Severity, &cr.Status, &cr.IsCoreLocked,
			&createdBy, &cr.CreatedAt, &cr.UpdatedAt,
		)
		if err == sql.ErrNoRows {
			writeJSONError(w, http.StatusNotFound, "Core rule not found", "not_found", "")
			return
		}
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, "Failed to query core rule", "query_error", err.Error())
			return
		}

		if targetEntity.Valid {
			cr.TargetEntity = targetEntity.String
		}
		if targetEntityID.Valid {
			cr.TargetEntityID = targetEntityID.String
		}
		cr.TargetEntities = targetEntities
		cr.TargetEntityIDs = targetEntityIDs
		if script.Valid {
			cr.ScriptContent = script.String
		}
		if createdBy.Valid {
			cr.CreatedBy = &createdBy.String
		}
		if len(condBytes) > 0 {
			var m map[string]interface{}
			if err := json.Unmarshal(condBytes, &m); err == nil {
				cr.ConditionJSON = m
			}
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(cr)
	}
}

func handleCreateValidationRuleCore(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req coreValidationRuleRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSONError(w, http.StatusBadRequest, "Invalid request body", "decode_error", err.Error())
			return
		}

		req.RuleKey = strings.TrimSpace(req.RuleKey)
		if req.RuleKey == "" {
			writeJSONError(w, http.StatusBadRequest, "rule_key is required", "validation_error", "")
			return
		}
		req.RuleName = strings.TrimSpace(req.RuleName)
		if req.RuleName == "" {
			writeJSONError(w, http.StatusBadRequest, "rule_name is required", "validation_error", "")
			return
		}
		req.RuleType = strings.TrimSpace(req.RuleType)
		if req.RuleType == "" {
			writeJSONError(w, http.StatusBadRequest, "rule_type is required", "validation_error", "")
			return
		}

		status := strings.TrimSpace(req.Status)
		if status == "" {
			status = "active"
		}

		severity := strings.TrimSpace(req.Severity)
		if severity == "" {
			severity = "error"
		}

		isLocked := false
		if req.IsCoreLocked != nil {
			isLocked = *req.IsCoreLocked
		}

		var createdBy interface{} = nil
		if strings.TrimSpace(req.CreatedBy) != "" {
			if _, err := uuid.Parse(req.CreatedBy); err == nil {
				createdBy = req.CreatedBy
			}
		}

		condBytes, _ := json.Marshal(req.ConditionJSON)

		q := `
			INSERT INTO public.catalog_validation_rule_cores (
				rule_key, version, rule_name, rule_type, description,
				target_entity, target_entity_id, target_entities, target_entity_ids,
				condition_json, script_content, severity, status, is_core_locked, created_by,
				created_at, updated_at
			)
			VALUES (
				$1,
				COALESCE($2, (SELECT COALESCE(MAX(version), 0) + 1 FROM public.catalog_validation_rule_cores WHERE rule_key = $1)),
				$3, $4, $5,
				NULLIF($6, ''), NULLIF($7, '')::uuid, $8::text[], $9::uuid[],
				$10::jsonb, NULLIF($11, ''), $12, $13, $14, $15,
				NOW(), NOW()
			)
			RETURNING id, rule_key, version, rule_name, rule_type, description,
			          target_entity, target_entity_id, target_entities, target_entity_ids,
			          condition_json, script_content, severity, status, is_core_locked,
			          created_by, created_at, updated_at
		`

		var cr CoreValidationRule
		var outCond []byte
		var outScript sql.NullString
		var outCreatedBy sql.NullString
		var targetEntity sql.NullString
		var targetEntityID sql.NullString
		var targetEntities pq.StringArray
		var targetEntityIDs pq.StringArray

		err := db.QueryRow(
			q,
			req.RuleKey,
			req.Version,
			req.RuleName,
			req.RuleType,
			req.Description,
			req.TargetEntity,
			req.TargetEntityID,
			pq.Array(req.TargetEntities),
			pq.Array(req.TargetEntityIDs),
			string(condBytes),
			req.ScriptContent,
			severity,
			status,
			isLocked,
			createdBy,
		).Scan(
			&cr.ID, &cr.RuleKey, &cr.Version, &cr.RuleName, &cr.RuleType, &cr.Description,
			&targetEntity, &targetEntityID, &targetEntities, &targetEntityIDs,
			&outCond, &outScript, &cr.Severity, &cr.Status, &cr.IsCoreLocked,
			&outCreatedBy, &cr.CreatedAt, &cr.UpdatedAt,
		)
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, "Failed to create core rule", "insert_error", err.Error())
			return
		}

		if targetEntity.Valid {
			cr.TargetEntity = targetEntity.String
		}
		if targetEntityID.Valid {
			cr.TargetEntityID = targetEntityID.String
		}
		cr.TargetEntities = targetEntities
		cr.TargetEntityIDs = targetEntityIDs
		if outScript.Valid {
			cr.ScriptContent = outScript.String
		}
		if outCreatedBy.Valid {
			cr.CreatedBy = &outCreatedBy.String
		}
		if len(outCond) > 0 {
			var m map[string]interface{}
			if err := json.Unmarshal(outCond, &m); err == nil {
				cr.ConditionJSON = m
			}
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(cr)
	}
}

func handleGetValidationRuleCoreImpact(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		if _, err := uuid.Parse(id); err != nil {
			writeJSONError(w, http.StatusBadRequest, "Invalid core rule id", "validation_error", err.Error())
			return
		}

		q := `
			SELECT tenant_id, datasource_id, inherit_mode, COUNT(*) as cnt
			FROM public.catalog_validation_rules
			WHERE core_rule_id = $1
			GROUP BY tenant_id, datasource_id, inherit_mode
			ORDER BY tenant_id, datasource_id, inherit_mode
		`
		rows, err := db.Query(q, id)
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, "Failed to query impact", "query_error", err.Error())
			return
		}
		defer rows.Close()

		type impactRow struct {
			TenantID     string `json:"tenant_id"`
			DatasourceID string `json:"datasource_id"`
			InheritMode  string `json:"inherit_mode"`
			Count        int    `json:"count"`
		}

		items := make([]impactRow, 0)
		totals := map[string]int{}
		grand := 0
		for rows.Next() {
			var it impactRow
			if err := rows.Scan(&it.TenantID, &it.DatasourceID, &it.InheritMode, &it.Count); err != nil {
				writeJSONError(w, http.StatusInternalServerError, "Failed to scan impact", "scan_error", err.Error())
				return
			}
			items = append(items, it)
			totals[it.InheritMode] += it.Count
			grand += it.Count
		}
		if err := rows.Err(); err != nil {
			writeJSONError(w, http.StatusInternalServerError, "Failed to read impact", "query_error", err.Error())
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"core_rule_id": id,
			"total":        grand,
			"totals":       totals,
			"items":        items,
		})
	}
}
