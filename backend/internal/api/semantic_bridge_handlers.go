package api

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/hondyman/uisce/backend/internal/semantic_bridge"
)

type SemanticBridgeHandler struct {
	db            *sqlx.DB
	cortexExp     *semantic_bridge.CortexExporter
	databricksExp *semantic_bridge.DatabricksExporter
}

func NewSemanticBridgeHandler(db *sqlx.DB) *SemanticBridgeHandler {
	return &SemanticBridgeHandler{
		db:            db,
		cortexExp:     semantic_bridge.NewCortexExporter(db),
		databricksExp: semantic_bridge.NewDatabricksExporter(db),
	}
}

func (h *SemanticBridgeHandler) RegisterRoutes(r chi.Router) {
	r.Route("/api/v1/semantic-bridge", func(r chi.Router) {
		r.Get("/targets", h.ListTargets)
		r.Post("/targets", h.CreateOrUpdateTarget)
		r.Post("/export/cortex", h.PreviewCortexYAML)
		r.Post("/export/databricks", h.PreviewDatabricksGenie)
		r.Post("/export/cortex/governance-ddl", h.PreviewSnowflakeGovernanceDDL)
		r.Post("/export/databricks/unity-catalog-sql", h.PreviewDatabricksUnityCatalogSQL)
		r.Post("/sync/{targetId}", h.TriggerSync)
		r.Get("/logs", h.GetSyncLogs)
	})
}

func (h *SemanticBridgeHandler) ListTargets(w http.ResponseWriter, r *http.Request) {
	tenantID, err := uuid.Parse(r.Header.Get("X-Tenant-ID"))
	if err != nil {
		http.Error(w, "Invalid X-Tenant-ID header", http.StatusBadRequest)
		return
	}

	var targets []semantic_bridge.BridgeTarget
	query := `SELECT id, tenant_id, vendor_type, target_name, is_active, config_payload, 
	                 sync_frequency, last_sync_at, last_sync_status, last_sync_error, created_at, updated_at
	          FROM catalog_ai.ai_bridge_targets WHERE tenant_id = $1 ORDER BY created_at DESC`

	if err := h.db.SelectContext(r.Context(), &targets, query, tenantID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(targets)
}

func (h *SemanticBridgeHandler) PreviewCortexYAML(w http.ResponseWriter, r *http.Request) {
	tenantID, err := uuid.Parse(r.Header.Get("X-Tenant-ID"))
	if err != nil {
		http.Error(w, "Invalid X-Tenant-ID header", http.StatusBadRequest)
		return
	}

	yamlBytes, err := h.cortexExp.CompileFullCortexModel(r.Context(), tenantID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "text/yaml")
	_, _ = w.Write(yamlBytes)
}

func (h *SemanticBridgeHandler) PreviewDatabricksGenie(w http.ResponseWriter, r *http.Request) {
	tenantID, err := uuid.Parse(r.Header.Get("X-Tenant-ID"))
	if err != nil {
		http.Error(w, "Invalid X-Tenant-ID header", http.StatusBadRequest)
		return
	}

	jsonBytes, err := h.databricksExp.CompileGenieModel(r.Context(), tenantID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(jsonBytes)
}

func (h *SemanticBridgeHandler) PreviewSnowflakeGovernanceDDL(w http.ResponseWriter, r *http.Request) {
	tenantID, err := uuid.Parse(r.Header.Get("X-Tenant-ID"))
	if err != nil {
		http.Error(w, "Invalid X-Tenant-ID header", http.StatusBadRequest)
		return
	}

	ddlStatements, err := h.cortexExp.GenerateSnowflakeGovernanceDDL(r.Context(), tenantID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(ddlStatements)
}

func (h *SemanticBridgeHandler) PreviewDatabricksUnityCatalogSQL(w http.ResponseWriter, r *http.Request) {
	tenantID, err := uuid.Parse(r.Header.Get("X-Tenant-ID"))
	if err != nil {
		http.Error(w, "Invalid X-Tenant-ID header", http.StatusBadRequest)
		return
	}

	sqlScript, err := h.databricksExp.GenerateUnityCatalogSQL(r.Context(), tenantID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "text/plain")
	_, _ = w.Write([]byte(sqlScript))
}

func (h *SemanticBridgeHandler) TriggerSync(w http.ResponseWriter, r *http.Request) {
	tenantID, err := uuid.Parse(r.Header.Get("X-Tenant-ID"))
	if err != nil {
		http.Error(w, "Invalid X-Tenant-ID header", http.StatusBadRequest)
		return
	}
	targetID, err := uuid.Parse(chi.URLParam(r, "targetId"))
	if err != nil {
		http.Error(w, "Invalid targetId format", http.StatusBadRequest)
		return
	}

	var target semantic_bridge.BridgeTarget
	if err := h.db.GetContext(r.Context(), &target, "SELECT * FROM catalog_ai.ai_bridge_targets WHERE id = $1 AND tenant_id = $2", targetID, tenantID); err != nil {
		http.Error(w, "Target not found", http.StatusNotFound)
		return
	}

	startTime := time.Now()
	var payload []byte

	if target.VendorType == semantic_bridge.VendorSnowflakeCortex {
		payload, err = h.cortexExp.CompileFullCortexModel(r.Context(), tenantID)
	} else if target.VendorType == semantic_bridge.VendorDatabricksGenie {
		payload, err = h.databricksExp.CompileGenieModel(r.Context(), tenantID)
	}

	hasher := sha256.New()
	hasher.Write(payload)
	payloadHash := hex.EncodeToString(hasher.Sum(nil))

	status := "SUCCESS"
	var errMsg string
	if err != nil {
		status = "ERROR"
		errMsg = err.Error()
	}

	// Record execution audit log
	_, _ = h.db.ExecContext(r.Context(), `
		INSERT INTO catalog_ai.ai_bridge_sync_logs (
			tenant_id, target_id, vendor_type, action, payload_hash, artifact_payload,
			status, http_status, response_body, execution_time_ms
		) VALUES ($1, $2, $3, 'STAGE_PUSH', $4, $5, $6, $7, $8, $9)`,
		tenantID, targetID, target.VendorType, payloadHash, string(payload),
		status, 200, errMsg, int(time.Since(startTime).Milliseconds()),
	)

	// Update target status
	_, _ = h.db.ExecContext(r.Context(), `
		UPDATE catalog_ai.ai_bridge_targets
		SET last_sync_at = NOW(), last_sync_status = $1, last_sync_error = $2, updated_at = NOW()
		WHERE id = $3`, status, errMsg, targetID)

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"status":          status,
		"payloadHash":     payloadHash,
		"executionTimeMs": time.Since(startTime).Milliseconds(),
	})
}

func (h *SemanticBridgeHandler) GetSyncLogs(w http.ResponseWriter, r *http.Request) {
	tenantID, err := uuid.Parse(r.Header.Get("X-Tenant-ID"))
	if err != nil {
		http.Error(w, "Invalid X-Tenant-ID header", http.StatusBadRequest)
		return
	}

	var logs []semantic_bridge.SyncLog
	query := `SELECT id, tenant_id, target_id, vendor_type, action, payload_hash,
	                 status, http_status, execution_time_ms, created_at
	          FROM catalog_ai.ai_bridge_sync_logs
	          WHERE tenant_id = $1 ORDER BY created_at DESC LIMIT 50`

	if err := h.db.SelectContext(r.Context(), &logs, query, tenantID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(logs)
}

func (h *SemanticBridgeHandler) CreateOrUpdateTarget(w http.ResponseWriter, r *http.Request) {
	tenantID, err := uuid.Parse(r.Header.Get("X-Tenant-ID"))
	if err != nil {
		http.Error(w, "Invalid X-Tenant-ID header", http.StatusBadRequest)
		return
	}

	var req struct {
		VendorType    string                 `json:"vendorType"`
		TargetName    string                 `json:"targetName"`
		ConfigPayload map[string]interface{} `json:"configPayload"`
		SyncFrequency string                 `json:"syncFrequency"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	cfgJSON, _ := json.Marshal(req.ConfigPayload)

	query := `
		INSERT INTO catalog_ai.ai_bridge_targets (
			tenant_id, vendor_type, target_name, config_payload, sync_frequency
		) VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (tenant_id, vendor_type, target_name) DO UPDATE
		SET config_payload = EXCLUDED.config_payload,
		    sync_frequency = EXCLUDED.sync_frequency,
		    updated_at = NOW()
		RETURNING id;`

	var id uuid.UUID
	err = h.db.QueryRowContext(r.Context(), query, tenantID, req.VendorType, req.TargetName, cfgJSON, req.SyncFrequency).Scan(&id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{"id": id, "status": "SAVED"})
}
