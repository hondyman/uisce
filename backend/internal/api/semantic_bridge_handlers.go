package api

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/hondyman/uisce/backend/internal/security"
	"github.com/hondyman/uisce/backend/internal/semantic_bridge"
	jwtmiddleware "github.com/hondyman/uisce/libs/jwt-middleware"
	"github.com/jmoiron/sqlx"
)

// requireAdminRole is the role gate for actions that write config/credentials
// or push to an external vendor — as opposed to read-only preview/listing
// endpoints, which only need a valid authenticated caller.
const requireAdminRole = "admin"

type SemanticBridgeHandler struct {
	db            *sqlx.DB
	cortexExp     *semantic_bridge.CortexExporter
	databricksExp *semantic_bridge.DatabricksExporter
	vault         *semantic_bridge.CredentialVault
	ledger        *semantic_bridge.Ledger
	databricksPsh *semantic_bridge.DatabricksPusher
	snowflakePsh  *semantic_bridge.SnowflakePusher
}

// NewSemanticBridgeHandler wires the AI bridge handlers to a real credential
// vault and tamper-evident ledger. It fails closed (returns an error instead
// of a handler that silently stores plaintext credentials) unless
// API_TOKEN_ENCRYPTION_KEY is set, or API_TOKEN_ENCRYPTION_KEY_DEV_FALLBACK=true
// for local development.
func NewSemanticBridgeHandler(db *sqlx.DB) (*SemanticBridgeHandler, error) {
	vault, err := semantic_bridge.NewCredentialVault()
	if err != nil {
		return nil, err
	}
	hmacKey, err := security.LoadKeyFromEnv(semantic_bridge.CredentialVaultKeyEnv, semantic_bridge.CredentialVaultDevFallbackEnv)
	if err != nil {
		return nil, err
	}
	return &SemanticBridgeHandler{
		db:            db,
		cortexExp:     semantic_bridge.NewCortexExporter(db),
		databricksExp: semantic_bridge.NewDatabricksExporter(db),
		vault:         vault,
		ledger:        semantic_bridge.NewLedger(db, hmacKey),
		databricksPsh: semantic_bridge.NewDatabricksPusher(),
		snowflakePsh:  semantic_bridge.NewSnowflakePusher(),
	}, nil
}

// RegisterRoutes mounts real JWT authentication on the whole semantic-bridge
// route group (not just an X-Tenant-ID header, which any caller could set to
// impersonate any tenant), plus an admin-role gate on the two actions that
// write config/credentials or push to an external vendor.
func (h *SemanticBridgeHandler) RegisterRoutes(r chi.Router) {
	r.Route("/api/v1/semantic-bridge", func(r chi.Router) {
		r.Use(jwtmiddleware.ChiMiddleware())

		r.Get("/targets", h.ListTargets)
		r.Post("/targets", requireAdmin(h.CreateOrUpdateTarget))
		r.Post("/export/cortex", h.PreviewCortexYAML)
		r.Post("/export/databricks", h.PreviewDatabricksGenie)
		r.Post("/export/cortex/governance-ddl", h.PreviewSnowflakeGovernanceDDL)
		r.Post("/export/databricks/unity-catalog-sql", h.PreviewDatabricksUnityCatalogSQL)
		r.Post("/sync/{targetId}", requireAdmin(h.TriggerSync))
		r.Get("/logs", h.GetSyncLogs)
		r.Get("/logs/verify", h.VerifyLedger)
	})
}

func requireAdmin(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		jwtmiddleware.RequireRole(requireAdminRole, next).ServeHTTP(w, r)
	}
}

// verifiedTenantIDFromClaims derives the tenant to operate on from the verified JWT
// claims set by jwtmiddleware.ChiMiddleware — not from the client-supplied
// X-Tenant-ID header, which any caller could set to any value to act as a
// different tenant. ChiMiddleware runs first and rejects unauthenticated
// requests with 401, so claims is never nil here.
func verifiedTenantIDFromClaims(r *http.Request) (uuid.UUID, error) {
	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		return uuid.Nil, fmt.Errorf("no authenticated claims on request")
	}
	return uuid.Parse(claims.TenantID)
}

// VerifyLedger walks the tenant's hash-chained sync log and confirms no row
// has been tampered with or removed since it was written.
func (h *SemanticBridgeHandler) VerifyLedger(w http.ResponseWriter, r *http.Request) {
	tenantID, err := verifiedTenantIDFromClaims(r)
	if err != nil {
		http.Error(w, "Invalid or missing tenant claim", http.StatusBadRequest)
		return
	}

	brokenAt, err := h.ledger.Verify(r.Context(), tenantID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if brokenAt == uuid.Nil {
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"intact": true})
	} else {
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"intact": false, "firstBrokenLogId": brokenAt})
	}
}

func (h *SemanticBridgeHandler) ListTargets(w http.ResponseWriter, r *http.Request) {
	tenantID, err := verifiedTenantIDFromClaims(r)
	if err != nil {
		http.Error(w, "Invalid or missing tenant claim", http.StatusBadRequest)
		return
	}

	var targets []semantic_bridge.BridgeTarget
	query := `SELECT id, tenant_id, vendor_type, target_name, is_active, config_payload,
	                 sync_frequency, credentials_rotated_at, last_sync_at, last_sync_status, last_sync_error, created_at, updated_at
	          FROM catalog_ai.ai_bridge_targets WHERE tenant_id = $1 ORDER BY created_at DESC`

	if err := h.db.SelectContext(r.Context(), &targets, query, tenantID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	type targetView struct {
		semantic_bridge.BridgeTarget
		CredentialRotationDue bool `json:"credentialRotationDue"`
	}
	views := make([]targetView, len(targets))
	for i, t := range targets {
		views[i] = targetView{BridgeTarget: t, CredentialRotationDue: t.CredentialRotationDue()}
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(views)
}

func (h *SemanticBridgeHandler) PreviewCortexYAML(w http.ResponseWriter, r *http.Request) {
	tenantID, err := verifiedTenantIDFromClaims(r)
	if err != nil {
		http.Error(w, "Invalid or missing tenant claim", http.StatusBadRequest)
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
	tenantID, err := verifiedTenantIDFromClaims(r)
	if err != nil {
		http.Error(w, "Invalid or missing tenant claim", http.StatusBadRequest)
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
	tenantID, err := verifiedTenantIDFromClaims(r)
	if err != nil {
		http.Error(w, "Invalid or missing tenant claim", http.StatusBadRequest)
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
	tenantID, err := verifiedTenantIDFromClaims(r)
	if err != nil {
		http.Error(w, "Invalid or missing tenant claim", http.StatusBadRequest)
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
	tenantID, err := verifiedTenantIDFromClaims(r)
	if err != nil {
		http.Error(w, "Invalid or missing tenant claim", http.StatusBadRequest)
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
	status, httpStatus, responseBody, payload := h.pushTarget(r.Context(), tenantID, &target)

	logID, logErr := h.ledger.Append(r.Context(), tenantID, &targetID, string(target.VendorType), "STAGE_PUSH",
		payload, status, httpStatus, responseBody, int(time.Since(startTime).Milliseconds()))
	if logErr != nil {
		http.Error(w, "sync ran but failed to write audit ledger: "+logErr.Error(), http.StatusInternalServerError)
		return
	}

	// Update target status
	_, _ = h.db.ExecContext(r.Context(), `
		UPDATE catalog_ai.ai_bridge_targets
		SET last_sync_at = NOW(), last_sync_status = $1, last_sync_error = $2, updated_at = NOW()
		WHERE id = $3`, status, responseBody, targetID)

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"status":          status,
		"httpStatus":      httpStatus,
		"logId":           logID,
		"executionTimeMs": time.Since(startTime).Milliseconds(),
	})
}

// pushTarget compiles the target's semantic model and, if the target has
// real credentials configured, pushes it over the network to the vendor.
// Without credentials it still compiles the artifact (useful for
// preview/dry-run) but reports NOT_CONFIGURED rather than faking a 200.
func (h *SemanticBridgeHandler) pushTarget(ctx context.Context, tenantID uuid.UUID, target *semantic_bridge.BridgeTarget) (status string, httpStatus int, responseBody string, payload []byte) {
	creds := h.vault.Open(target.CredentialsVaulted)

	switch target.VendorType {
	case semantic_bridge.VendorSnowflakeCortex:
		var err error
		payload, err = h.cortexExp.CompileFullCortexModel(ctx, tenantID)
		if err != nil {
			return "ERROR", 0, err.Error(), payload
		}
		ddl, err := h.cortexExp.GenerateSnowflakeGovernanceDDL(ctx, tenantID)
		if err != nil {
			return "ERROR", 0, err.Error(), payload
		}
		statements := make([]string, 0, len(ddl))
		for _, d := range ddl {
			statements = append(statements, d.SQL)
		}
		if len(statements) == 0 {
			return "SUCCESS", 0, "compiled model only — no governance-tagged columns to push", payload
		}
		res, err := h.snowflakePsh.Push(ctx, target.ConfigPayload, creds, statements)
		if err != nil {
			return "ERROR", 0, err.Error(), payload
		}
		if !res.Success {
			if res.HTTPStatus == 0 {
				return "NOT_CONFIGURED", 0, res.ResponseBody, payload
			}
			return "ERROR", res.HTTPStatus, res.ResponseBody, payload
		}
		return "SUCCESS", res.HTTPStatus, res.ResponseBody, payload

	case semantic_bridge.VendorDatabricksGenie:
		var err error
		payload, err = h.databricksExp.CompileGenieModel(ctx, tenantID)
		if err != nil {
			return "ERROR", 0, err.Error(), payload
		}
		ddlScript, err := h.databricksExp.GenerateUnityCatalogSQL(ctx, tenantID)
		if err != nil {
			return "ERROR", 0, err.Error(), payload
		}
		res, err := h.databricksPsh.Push(ctx, target.ConfigPayload, creds, ddlScript)
		if err != nil {
			return "ERROR", 0, err.Error(), payload
		}
		if !res.Success {
			if res.HTTPStatus == 0 {
				return "NOT_CONFIGURED", 0, res.ResponseBody, payload
			}
			return "ERROR", res.HTTPStatus, res.ResponseBody, payload
		}
		return "SUCCESS", res.HTTPStatus, res.ResponseBody, payload

	default:
		return "UNSUPPORTED_VENDOR", 0, "no push implementation for vendor " + string(target.VendorType), nil
	}
}

func (h *SemanticBridgeHandler) GetSyncLogs(w http.ResponseWriter, r *http.Request) {
	tenantID, err := verifiedTenantIDFromClaims(r)
	if err != nil {
		http.Error(w, "Invalid or missing tenant claim", http.StatusBadRequest)
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
	tenantID, err := verifiedTenantIDFromClaims(r)
	if err != nil {
		http.Error(w, "Invalid or missing tenant claim", http.StatusBadRequest)
		return
	}

	var req struct {
		VendorType    string                 `json:"vendorType"`
		TargetName    string                 `json:"targetName"`
		ConfigPayload map[string]interface{} `json:"configPayload"`
		SyncFrequency string                 `json:"syncFrequency"`
		// Credentials holds plaintext secrets (e.g. {"token": "dapi..."}) —
		// never persisted as-is. Sealed with the AES-256-GCM vault before
		// storage. Omit this field on updates that don't rotate a secret;
		// existing sealed credentials are left untouched in that case.
		Credentials map[string]string `json:"credentials"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	cfgJSON, _ := json.Marshal(req.ConfigPayload)

	var credsJSON []byte
	if len(req.Credentials) > 0 {
		sealed, err := h.vault.Seal(req.Credentials)
		if err != nil {
			http.Error(w, "failed to seal credentials: "+err.Error(), http.StatusInternalServerError)
			return
		}
		credsJSON, _ = json.Marshal(sealed)
	}

	isRotation := credsJSON != nil

	query := `
		INSERT INTO catalog_ai.ai_bridge_targets (
			tenant_id, vendor_type, target_name, config_payload, sync_frequency, credentials_vaulted, credentials_rotated_at
		) VALUES ($1, $2, $3, $4, $5, COALESCE($6, '{}'::jsonb), CASE WHEN $6 IS NULL THEN NULL ELSE NOW() END)
		ON CONFLICT (tenant_id, vendor_type, target_name) DO UPDATE
		SET config_payload = EXCLUDED.config_payload,
		    sync_frequency = EXCLUDED.sync_frequency,
		    credentials_vaulted = CASE WHEN $6 IS NULL THEN catalog_ai.ai_bridge_targets.credentials_vaulted ELSE EXCLUDED.credentials_vaulted END,
		    credentials_rotated_at = CASE WHEN $6 IS NULL THEN catalog_ai.ai_bridge_targets.credentials_rotated_at ELSE NOW() END,
		    updated_at = NOW()
		RETURNING id;`

	var id uuid.UUID
	var credsArg interface{}
	if credsJSON != nil {
		credsArg = credsJSON
	}
	err = h.db.QueryRowContext(r.Context(), query, tenantID, req.VendorType, req.TargetName, cfgJSON, req.SyncFrequency, credsArg).Scan(&id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Credential rotation is a security-sensitive event worth its own
	// tamper-evident audit trail entry — never the secret itself, just the
	// fact that it changed, by whom (via the ledger's tenant scoping), and
	// when.
	if isRotation {
		if _, err := h.ledger.Append(r.Context(), tenantID, &id, req.VendorType, "CREDENTIALS_ROTATED",
			[]byte(fmt.Sprintf("target=%s vendor=%s keys=%d", req.TargetName, req.VendorType, len(req.Credentials))),
			"SUCCESS", 0, "", 0); err != nil {
			log.Printf("[SemanticBridge] failed to audit-log credential rotation for target %s: %v", id, err)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{"id": id, "status": "SAVED", "credentialsRotated": isRotation})
}
