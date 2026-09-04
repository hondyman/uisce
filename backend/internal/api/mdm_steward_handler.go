package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/hondyman/uisce/backend/internal/audit"
	"github.com/jmoiron/sqlx"
	"go.temporal.io/sdk/client"
)

type MDMOverrideRequest struct {
	ExceptionID            uuid.UUID   `json:"exceptionId"`
	MasterEntitySID        string      `json:"masterEntitySid"`
	DomainKey              string      `json:"domainKey"`
	FieldName              string      `json:"fieldName"`
	ChosenVendor           string      `json:"chosenVendor"`
	OverrideValue          interface{} `json:"overrideValue"`
	OverrideReason         string      `json:"overrideReason"`
	SignalTemporalWorkflow bool        `json:"signalTemporalWorkflow"`
}

type MDMStewardHandler struct {
	db             *sqlx.DB
	temporalClient client.Client
	outboxMgr      *audit.TransactionalOutboxManager
}

func NewMDMStewardHandler(db *sqlx.DB, tc client.Client, outbox *audit.TransactionalOutboxManager) *MDMStewardHandler {
	return &MDMStewardHandler{
		db:             db,
		temporalClient: tc,
		outboxMgr:      outbox,
	}
}

func (h *MDMStewardHandler) GetExceptions(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	// getSecureTenantID (helpers.go) validates the X-Tenant-ID header against
	// the caller's JWT-issued tenant list / global-admin status before
	// trusting it; it never trusts a raw header/query param directly.
	tenantIDStr := getSecureTenantID(r)
	tenantID, err := uuid.Parse(tenantIDStr)
	if err != nil {
		http.Error(w, `{"error":"invalid tenant context"}`, http.StatusBadRequest)
		return
	}

	domainKey := r.URL.Query().Get("domain")
	anomalyType := r.URL.Query().Get("anomaly")

	query := `
		SELECT exception_id, tenant_id, domain_key, master_entity_sid, field_name,
		       competing_values, anomaly_type, status, assigned_steward_id,
		       steward_override_value, steward_override_reason, created_at
		FROM mdm.universal_exception_queue
		WHERE tenant_id = $1
	`
	args := []interface{}{tenantID}
	argIdx := 2

	if domainKey != "" && domainKey != "ALL" {
		query += fmt.Sprintf(" AND domain_key = $%d", argIdx)
		args = append(args, domainKey)
		argIdx++
	}
	if anomalyType != "" && anomalyType != "ALL" {
		query += fmt.Sprintf(" AND anomaly_type = $%d", argIdx)
		args = append(args, anomalyType)
		argIdx++
	}
	query += " ORDER BY created_at DESC LIMIT 100"

	type ExceptionRow struct {
		ExceptionID           uuid.UUID       `db:"exception_id" json:"exceptionId"`
		TenantID              uuid.UUID       `db:"tenant_id" json:"tenantId"`
		DomainKey             string          `db:"domain_key" json:"domainKey"`
		MasterEntitySID       string          `db:"master_entity_sid" json:"masterEntitySid"`
		FieldName             string          `db:"field_name" json:"fieldName"`
		CompetingValuesRaw    []byte          `db:"competing_values" json:"-"`
		CompetingValues       json.RawMessage `json:"competingValues"`
		AnomalyType           string          `db:"anomaly_type" json:"anomalyType"`
		Status                string          `db:"status" json:"status"`
		AssignedStewardID     *string         `db:"assigned_steward_id" json:"assignedStewardId,omitempty"`
		StewardOverrideValue  json.RawMessage `db:"steward_override_value" json:"stewardOverrideValue,omitempty"`
		StewardOverrideReason *string         `db:"steward_override_reason" json:"stewardOverrideReason,omitempty"`
		CreatedAt             time.Time       `db:"created_at" json:"createdAt"`
	}

	var rows []ExceptionRow
	if h.db != nil {
		_ = h.db.SelectContext(r.Context(), &rows, query, args...)
	}

	for i := range rows {
		if len(rows[i].CompetingValuesRaw) > 0 {
			rows[i].CompetingValues = json.RawMessage(rows[i].CompetingValuesRaw)
		}
	}

	_ = json.NewEncoder(w).Encode(rows)
}

func (h *MDMStewardHandler) ApplyOverride(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	// getSecureTenantID (helpers.go) validates the X-Tenant-ID header against
	// the caller's JWT-issued tenant list / global-admin status before
	// trusting it; it never trusts a raw header/query param directly.
	tenantIDStr := getSecureTenantID(r)
	tenantID, err := uuid.Parse(tenantIDStr)
	if err != nil {
		http.Error(w, `{"error":"invalid tenant context"}`, http.StatusBadRequest)
		return
	}

	var req MDMOverrideRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid payload"}`, http.StatusBadRequest)
		return
	}

	if h.db != nil {
		tx, err := h.db.BeginTxx(r.Context(), nil)
		if err != nil {
			http.Error(w, `{"error":"transaction start failed"}`, http.StatusInternalServerError)
			return
		}
		defer tx.Rollback()

		// 1. Update Exception Record in Queue
		valJSON, _ := json.Marshal(req.OverrideValue)
		updateQueueSQL := `
			UPDATE mdm.universal_exception_queue
			SET status = 'OVERRIDDEN',
			    steward_override_value = $1,
			    steward_override_reason = $2,
			    resolved_at = NOW()
			WHERE exception_id = $3 AND tenant_id = $4;
		`
		_, err = tx.ExecContext(r.Context(), updateQueueSQL, valJSON, req.OverrideReason, req.ExceptionID, tenantID)
		if err != nil {
			http.Error(w, `{"error":"failed updating exception queue"}`, http.StatusInternalServerError)
			return
		}

		// 2. Stage SEC Rule 17a-4 Tamper-Evident Regulatory Outbox Event
		if h.outboxMgr != nil {
			outboxPayload := map[string]interface{}{
				"exception_id":    req.ExceptionID.String(),
				"entity_sid":      req.MasterEntitySID,
				"domain_key":      req.DomainKey,
				"field_name":      req.FieldName,
				"override_value":  req.OverrideValue,
				"chosen_vendor":   req.ChosenVendor,
				"steward_reason":  req.OverrideReason,
				"resolved_at_utc": time.Now().UTC().Format(time.RFC3339),
			}
			_ = h.outboxMgr.StageOutboxEventAtomic(
				r.Context(), tx, tenantID, req.ExceptionID,
				"MDM_STEWARD", "EXCEPTION_OVERRIDDEN",
				"STEWARD_PORTAL", "DATA_STEWARD", outboxPayload,
			)
		}

		if err := tx.Commit(); err != nil {
			http.Error(w, `{"error":"commit failed"}`, http.StatusInternalServerError)
			return
		}
	}

	// 3. Dispatch Temporal Signal to Resume Ingestion Workflow
	if req.SignalTemporalWorkflow && h.temporalClient != nil {
		workflowID := "mdm-ingest-" + req.MasterEntitySID
		_ = h.temporalClient.SignalWorkflow(
			r.Context(),
			workflowID,
			"",
			"MDM_STEWARD_OVERRIDE_SIGNAL",
			req,
		)
	}

	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]string{
		"status":  "OVERRIDDEN",
		"message": "Override applied and downstream workflow signaled",
	})
}
