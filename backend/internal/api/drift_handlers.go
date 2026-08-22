package api

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/hondyman/uisce/backend/internal/drift"
	"github.com/jmoiron/sqlx"
)

type DriftHandler struct {
	db           *sqlx.DB
	patchService *drift.PatchService
}

func NewDriftHandler(db *sqlx.DB, patchService *drift.PatchService) *DriftHandler {
	return &DriftHandler{
		db:           db,
		patchService: patchService,
	}
}

// GetPendingProposals returns all pending schema drift remediation proposals
func (h *DriftHandler) GetPendingProposals(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	tenantID, err := uuid.Parse(r.Header.Get("X-Tenant-ID"))
	if err != nil {
		http.Error(w, `{"error":"invalid tenant context"}`, http.StatusBadRequest)
		return
	}

	query := `
		SELECT p.proposal_id AS "proposalId", p.bo_id AS "boId", bo.bo_name AS "boName",
		       p.field_id AS "fieldId", p.field_name AS "fieldName",
		       COALESCE(cn_old.node_name, 'UNKNOWN') AS "currentColumn",
		       p.proposed_column_name AS "proposedColumn",
		       p.confidence_score AS "confidenceScore",
		       p.matching_strategy AS "matchingStrategy",
		       p.affected_reports_count AS "affectedReportsCount",
		       p.remediation_rationale AS "remediationRationale",
		       p.status
		FROM catalog_drift.schema_drift_proposals p
		JOIN public.business_objects bo ON bo.id = p.bo_id
		LEFT JOIN public.catalog_node cn_old ON cn_old.node_id = p.current_source_node_id
		WHERE p.tenant_id = $1 AND p.status = 'PENDING'
		ORDER BY p.confidence_score DESC;
	`
	var results []map[string]interface{}
	rows, err := h.db.QueryxContext(r.Context(), query, tenantID)
	if err != nil {
		http.Error(w, `{"error":"failed querying drift proposals"}`, http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	for rows.Next() {
		item := make(map[string]interface{})
		if err := rows.MapScan(item); err == nil {
			results = append(results, item)
		}
	}

	if results == nil {
		results = []map[string]interface{}{}
	}
	_ = json.NewEncoder(w).Encode(results)
}

// ApplyProposal executes atomic 1-click hot-swap patch
func (h *DriftHandler) ApplyProposal(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	tenantID, err := uuid.Parse(r.Header.Get("X-Tenant-ID"))
	if err != nil {
		http.Error(w, `{"error":"invalid tenant context"}`, http.StatusBadRequest)
		return
	}

	proposalID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, `{"error":"invalid proposal ID"}`, http.StatusBadRequest)
		return
	}

	err = h.patchService.ApplyHotSwapPatch(r.Context(), tenantID, proposalID, "DATA_STEWARD_WEB")
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error":"%s"}`, err.Error()), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]string{
		"status":  "APPLIED",
		"message": "Field binding hot-swap applied successfully",
	})
}
