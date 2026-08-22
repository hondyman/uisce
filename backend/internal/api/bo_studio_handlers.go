package api

import (
	"encoding/json"
	"net/http"

	"github.com/google/uuid"
	"github.com/hondyman/uisce/backend/internal/boresolver"
	"github.com/jmoiron/sqlx"
)

type BOStudioHandler struct {
	db               *sqlx.DB
	discoveryService *boresolver.BODiscoveryService
	saveService      *boresolver.BOSaveService
}

func NewBOStudioHandler(db *sqlx.DB) *BOStudioHandler {
	return &BOStudioHandler{
		db:               db,
		discoveryService: boresolver.NewBODiscoveryService(db),
		saveService:      boresolver.NewBOSaveService(db),
	}
}

// GetBindingContext auto-discovers PKs, related tables, and eligible terms for a driving table
func (h *BOStudioHandler) GetBindingContext(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	tenantID, err := uuid.Parse(r.Header.Get("X-Tenant-ID"))
	if err != nil {
		http.Error(w, `{"error":"invalid tenant context"}`, http.StatusBadRequest)
		return
	}

	var req boresolver.BindingContextRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request payload"}`, http.StatusBadRequest)
		return
	}
	req.TenantID = tenantID

	resp, err := h.discoveryService.DiscoverBindingContext(r.Context(), req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	_ = json.NewEncoder(w).Encode(resp)
}

// SaveBusinessObject handles the atomic save/publish submission
func (h *BOStudioHandler) SaveBusinessObject(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	tenantID, err := uuid.Parse(r.Header.Get("X-Tenant-ID"))
	if err != nil {
		http.Error(w, `{"error":"invalid tenant context"}`, http.StatusBadRequest)
		return
	}

	var req boresolver.AtomicSaveBORequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request payload"}`, http.StatusBadRequest)
		return
	}
	req.TenantID = tenantID

	boID, err := h.saveService.SaveBusinessObjectAtomic(r.Context(), req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "SUCCESS",
		"boId":   boID,
	})
}
