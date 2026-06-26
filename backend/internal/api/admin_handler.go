package api

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/hondyman/semlayer/backend/internal/services"
	"go.uber.org/zap"
)

type AdminHandler struct {
	qosManager *services.QoSManager
	logger     *zap.Logger
}

func NewAdminHandler(qosManager *services.QoSManager) *AdminHandler {
	logger, _ := zap.NewProduction()
	return &AdminHandler{
		qosManager: qosManager,
		logger:     logger,
	}
}

func (h *AdminHandler) RegisterRoutes(r chi.Router) {
	r.Route("/admin", func(r chi.Router) {
		r.Get("/quotas", h.GetQuotas)
		r.Post("/quotas", h.UpdateQuota)
	})
}

func (h *AdminHandler) GetQuotas(w http.ResponseWriter, r *http.Request) {
	// In a real implementation, we would query the database or the QoSManager's cache.
	// Since QoSManager allows reading quotas (it has a cache), we might need to expose a getter.
	// For now, let's just query the DB directly here or add a method to QoSManager.
	// Given QoSManager is in services, better to add GetAllQuotas there.
	// But to avoid changing service interface if not needed, let's cheat slightly and just return what we know
	// or adding a GetAllQuotas method to QoSManager is cleaner.

	// Let's assume we added GetAllQuotas to QoSManager for this.
	// Actually, I'll implement a direct DB query here for simplicity if QoSManager doesn't expose it,
	// but AdminHandler doesn't have DB access, only QoSManager.
	// So I will assume I can call qosManager.GetAllQuotas() and I will add it to QoSManager next.

	quotas := h.qosManager.GetAllQuotas()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(quotas)
}

type UpdateQuotaRequest struct {
	TenantID string `json:"tenant_id"`
	Resource string `json:"resource"`
	Limit    int64  `json:"limit_value"`
	Window   int    `json:"window_seconds"`
}

func (h *AdminHandler) UpdateQuota(w http.ResponseWriter, r *http.Request) {
	var req UpdateQuotaRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := h.qosManager.UpdateQuota(req.TenantID, req.Resource, req.Limit, req.Window); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}
