package api

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/hondyman/semlayer/backend/internal/investment/alts"
	"github.com/hondyman/semlayer/libs/jwt-middleware"
)

type AltsHandler struct {
	service *alts.Service
}

func NewAltsHandler(service *alts.Service) *AltsHandler {
	return &AltsHandler{service: service}
}

func (h *AltsHandler) RegisterRoutes(r chi.Router) {
	r.Route("/investment/alts", func(r chi.Router) {
		r.Post("/", h.CreateAsset)
		r.Get("/", h.ListAssets)
		r.Route("/{assetID}", func(r chi.Router) {
			r.Get("/", h.GetAsset)
			r.Post("/valuations", h.RecordValuation)
			r.Get("/valuations", h.GetValuationHistory)
			r.Get("/nav", h.GetDailyNAV)
		})
	})
}

func (h *AltsHandler) CreateAsset(w http.ResponseWriter, r *http.Request) {
	var asset alts.AlternativeAsset
	if err := json.NewDecoder(r.Body).Decode(&asset); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	
	// Basic validation (tenant_id should ideally come from context/token)
	if asset.TenantID == uuid.Nil {
		http.Error(w, "tenant_id is required", http.StatusBadRequest)
		return
	}

	if err := h.service.CreateAsset(r.Context(), &asset); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(asset)
}

func (h *AltsHandler) ListAssets(w http.ResponseWriter, r *http.Request) {
	tenantIDStr := jwtmiddleware.GetClaimsFromContext(r).TenantID // Or query param
	if tenantIDStr == "" {
		// Fallback for demo/testing
		tenantIDStr = r.URL.Query().Get("tenant_id")
	}
	
	if tenantIDStr == "" {
		http.Error(w, "tenant_id is required", http.StatusBadRequest)
		return
	}
	
	tenantID, err := uuid.Parse(tenantIDStr)
	if err != nil {
		http.Error(w, "invalid tenant_id", http.StatusBadRequest)
		return
	}

	assetType := r.URL.Query().Get("type")
	assets, err := h.service.ListAssets(r.Context(), tenantID, assetType)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(assets)
}

func (h *AltsHandler) GetAsset(w http.ResponseWriter, r *http.Request) {
	assetID, err := uuid.Parse(chi.URLParam(r, "assetID"))
	if err != nil {
		http.Error(w, "invalid assetID", http.StatusBadRequest)
		return
	}

	asset, err := h.service.GetAsset(r.Context(), assetID)
	if err != nil {
		http.Error(w, "Asset not found", http.StatusNotFound)
		return
	}

	json.NewEncoder(w).Encode(asset)
}

func (h *AltsHandler) RecordValuation(w http.ResponseWriter, r *http.Request) {
	assetID, err := uuid.Parse(chi.URLParam(r, "assetID"))
	if err != nil {
		http.Error(w, "invalid assetID", http.StatusBadRequest)
		return
	}

	var event alts.ValuationEvent
	if err := json.NewDecoder(r.Body).Decode(&event); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	event.AssetID = assetID

	if err := h.service.RecordValuation(r.Context(), &event); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(event)
}

func (h *AltsHandler) GetValuationHistory(w http.ResponseWriter, r *http.Request) {
	assetID, err := uuid.Parse(chi.URLParam(r, "assetID"))
	if err != nil {
		http.Error(w, "invalid assetID", http.StatusBadRequest)
		return
	}

	events, err := h.service.GetValuationHistory(r.Context(), assetID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(events)
}

func (h *AltsHandler) GetDailyNAV(w http.ResponseWriter, r *http.Request) {
	assetID, err := uuid.Parse(chi.URLParam(r, "assetID"))
	if err != nil {
		http.Error(w, "invalid assetID", http.StatusBadRequest)
		return
	}

	startStr := r.URL.Query().Get("start_date")
	endStr := r.URL.Query().Get("end_date")

	if startStr == "" || endStr == "" {
		http.Error(w, "start_date and end_date are required", http.StatusBadRequest)
		return
	}

	start, err := time.Parse("2006-01-02", startStr)
	if err != nil {
		http.Error(w, "invalid start_date format (YYYY-MM-DD)", http.StatusBadRequest)
		return
	}
	end, err := time.Parse("2006-01-02", endStr)
	if err != nil {
		http.Error(w, "invalid end_date format (YYYY-MM-DD)", http.StatusBadRequest)
		return
	}

	navs, err := h.service.GetDailyNAV(r.Context(), assetID, start, end)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(navs)
}
