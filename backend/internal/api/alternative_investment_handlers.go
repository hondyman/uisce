package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/hondyman/semlayer/backend/internal/audit"
	"github.com/hondyman/semlayer/backend/internal/services"
	"github.com/hondyman/semlayer/backend/internal/types"
	"github.com/hondyman/semlayer/backend/internal/webhooks"
)

// AlternativeInvestmentHandlers handles HTTP requests for alternative investments
type AlternativeInvestmentHandlers struct {
	altInvService *services.AlternativeInvestmentService
	perfService   *services.PerformanceService
	docService    *services.DocumentProcessingService
	auditSvc      *audit.Service
	secMgr        *services.SecurityManager
	webhookSvc    *webhooks.Service
}

// NewAlternativeInvestmentHandlers creates a new handler
func NewAlternativeInvestmentHandlers(
	altInvService *services.AlternativeInvestmentService,
	perfService *services.PerformanceService,
	docService *services.DocumentProcessingService,
	auditSvc *audit.Service,
	secMgr *services.SecurityManager,
	webhookSvc *webhooks.Service,
) *AlternativeInvestmentHandlers {
	return &AlternativeInvestmentHandlers{
		altInvService: altInvService,
		perfService:   perfService,
		docService:    docService,
		auditSvc:      auditSvc,
		secMgr:        secMgr,
		webhookSvc:    webhookSvc,
	}
}

const (
	permAltInvRead  = "altinvestments.read"
	permAltInvWrite = "altinvestments.write"
)

// RegisterRoutes registers all alternative investment routes
func (h *AlternativeInvestmentHandlers) RegisterRoutes(r chi.Router) {
	r.Route("/api/v1/alternative-investments", func(r chi.Router) {
		// Investments CRUD
		r.Post("/", h.CreateInvestment)
		r.Get("/", h.ListInvestments)
		r.Get("/{id}", h.GetInvestment)
		r.Put("/{id}", h.UpdateInvestment)
		r.Delete("/{id}", h.DeleteInvestment)

		// Capital calls
		r.Post("/{id}/capital-calls", h.CreateCapitalCall)
		r.Get("/{id}/capital-calls", h.ListCapitalCalls)
		r.Post("/capital-calls/{callId}/fund", h.FundCapitalCall)

		// Distributions
		r.Post("/{id}/distributions", h.CreateDistribution)
		r.Get("/{id}/distributions", h.ListDistributions)

		// Performance
		r.Get("/{id}/performance", h.GetPerformance)
		r.Post("/{id}/performance/calculate", h.CalculatePerformance)
		r.Get("/{id}/performance/history", h.GetPerformanceHistory)

		// Documents
		r.Post("/{id}/documents", h.UploadDocument)
		r.Get("/{id}/documents", h.ListDocuments)
		r.Post("/documents/{docId}/approve", h.ApproveDocument)
		r.Post("/documents/{docId}/reject", h.RejectDocument)
	})
}

// CreateInvestment creates a new alternative investment
func (h *AlternativeInvestmentHandlers) CreateInvestment(w http.ResponseWriter, r *http.Request) {
	actorID, tenantID, datasourceID, ok := h.authorize(w, r, permAltInvWrite)
	if !ok {
		return
	}

	var inv types.AlternativeInvestment
	if err := json.NewDecoder(r.Body).Decode(&inv); err != nil {
		writeJSONError(w, http.StatusBadRequest, "Invalid request body", "invalid_payload", nil)
		return
	}

	tenantUUID, err := uuid.Parse(tenantID)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "Invalid tenant scope provided", "invalid_tenant_scope", nil)
		return
	}
	inv.TenantID = tenantUUID

	if inv.ClientID == uuid.Nil {
		writeJSONError(w, http.StatusBadRequest, "clientId is required", "missing_client", nil)
		return
	}

	if err := h.altInvService.CreateInvestment(r.Context(), &inv); err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error(), "create_failed", nil)
		return
	}

	h.auditModification(r.Context(), actorID, tenantID, inv.ID.String(), "alternative_investment", "create", nil, inv)
	h.emitWebhookEvent(r.Context(), "altinvestments.investment.created", map[string]interface{}{"investment": inv}, map[string]string{
		"tenant_id":     tenantID,
		"datasource_id": datasourceID,
		"actor_id":      actorID,
	})

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(inv)
}

// ListInvestments retrieves all investments for a client
func (h *AlternativeInvestmentHandlers) ListInvestments(w http.ResponseWriter, r *http.Request) {
	actorID, tenantID, _, ok := h.authorize(w, r, permAltInvRead)
	if !ok {
		return
	}

	clientIDStr := r.URL.Query().Get("client_id")
	if clientIDStr == "" {
		writeJSONError(w, http.StatusBadRequest, "client_id parameter required", "missing_client_id", nil)
		return
	}

	clientID, err := uuid.Parse(clientIDStr)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "Invalid client_id", "invalid_client_id", nil)
		return
	}

	tenantUUID, err := uuid.Parse(tenantID)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "Invalid tenant scope provided", "invalid_tenant_scope", nil)
		return
	}

	investments, err := h.altInvService.GetInvestmentsByClient(r.Context(), tenantUUID, clientID)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error(), "list_failed", nil)
		return
	}

	h.auditAccess(r.Context(), actorID, tenantID, "", "alternative_investment", "list", map[string]interface{}{
		"client_id": clientIDStr,
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(investments)
}

// GetInvestment retrieves a single investment
func (h *AlternativeInvestmentHandlers) GetInvestment(w http.ResponseWriter, r *http.Request) {
	actorID, tenantID, _, ok := h.authorize(w, r, permAltInvRead)
	if !ok {
		return
	}

	investmentID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "Invalid investment ID", "invalid_investment_id", nil)
		return
	}

	tenantUUID, err := uuid.Parse(tenantID)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "Invalid tenant scope provided", "invalid_tenant_scope", nil)
		return
	}

	investment, err := h.altInvService.GetInvestment(r.Context(), tenantUUID, investmentID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeJSONError(w, http.StatusNotFound, "Investment not found", "not_found", nil)
			return
		}
		writeJSONError(w, http.StatusInternalServerError, err.Error(), "lookup_failed", nil)
		return
	}

	h.auditAccess(r.Context(), actorID, tenantID, investmentID.String(), "alternative_investment", "read", nil)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(investment)
}

// UpdateInvestment updates an investment
func (h *AlternativeInvestmentHandlers) UpdateInvestment(w http.ResponseWriter, r *http.Request) {
	actorID, tenantID, datasourceID, ok := h.authorize(w, r, permAltInvWrite)
	if !ok {
		return
	}

	investmentID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "Invalid investment ID", "invalid_investment_id", nil)
		return
	}

	var inv types.AlternativeInvestment
	if err := json.NewDecoder(r.Body).Decode(&inv); err != nil {
		writeJSONError(w, http.StatusBadRequest, "Invalid request body", "invalid_payload", nil)
		return
	}

	inv.ID = investmentID
	tenantUUID, err := uuid.Parse(tenantID)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "Invalid tenant scope provided", "invalid_tenant_scope", nil)
		return
	}
	inv.TenantID = tenantUUID

	previous, err := h.altInvService.GetInvestment(r.Context(), tenantUUID, investmentID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeJSONError(w, http.StatusNotFound, "Investment not found", "not_found", nil)
			return
		}
		writeJSONError(w, http.StatusInternalServerError, err.Error(), "lookup_failed", nil)
		return
	}

	if err := h.altInvService.UpdateInvestment(r.Context(), &inv); err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error(), "update_failed", nil)
		return
	}

	updated, err := h.altInvService.GetInvestment(r.Context(), tenantUUID, investmentID)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error(), "lookup_failed", nil)
		return
	}

	h.auditModification(r.Context(), actorID, tenantID, investmentID.String(), "alternative_investment", "update", previous, updated)
	h.emitWebhookEvent(r.Context(), "altinvestments.investment.updated", map[string]interface{}{"investment": updated}, map[string]string{
		"tenant_id":     tenantID,
		"datasource_id": datasourceID,
		"actor_id":      actorID,
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(updated)
}

// DeleteInvestment soft-deletes an investment
func (h *AlternativeInvestmentHandlers) DeleteInvestment(w http.ResponseWriter, r *http.Request) {
	actorID, tenantID, datasourceID, ok := h.authorize(w, r, permAltInvWrite)
	if !ok {
		return
	}

	investmentID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "Invalid investment ID", "invalid_investment_id", nil)
		return
	}

	tenantUUID, err := uuid.Parse(tenantID)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "Invalid tenant scope provided", "invalid_tenant_scope", nil)
		return
	}

	existing, err := h.altInvService.GetInvestment(r.Context(), tenantUUID, investmentID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeJSONError(w, http.StatusNotFound, "Investment not found", "not_found", nil)
			return
		}
		writeJSONError(w, http.StatusInternalServerError, err.Error(), "lookup_failed", nil)
		return
	}

	if err := h.altInvService.DeleteInvestment(r.Context(), tenantUUID, investmentID); err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error(), "delete_failed", nil)
		return
	}

	h.auditModification(r.Context(), actorID, tenantID, investmentID.String(), "alternative_investment", "delete", existing, nil)
	h.emitWebhookEvent(r.Context(), "altinvestments.investment.deleted", map[string]interface{}{"investment": existing}, map[string]string{
		"tenant_id":     tenantID,
		"datasource_id": datasourceID,
		"actor_id":      actorID,
	})

	w.WriteHeader(http.StatusNoContent)
}

// CreateCapitalCall creates a new capital call
func (h *AlternativeInvestmentHandlers) CreateCapitalCall(w http.ResponseWriter, r *http.Request) {
	actorID, tenantID, datasourceID, ok := h.authorize(w, r, permAltInvWrite)
	if !ok {
		return
	}

	investmentID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "Invalid investment ID", "invalid_investment_id", nil)
		return
	}

	var call types.CapitalCall
	if err := json.NewDecoder(r.Body).Decode(&call); err != nil {
		writeJSONError(w, http.StatusBadRequest, "Invalid request body", "invalid_payload", nil)
		return
	}

	call.InvestmentID = investmentID

	if err := h.altInvService.RecordCapitalCall(r.Context(), &call); err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error(), "create_failed", nil)
		return
	}

	h.auditModification(r.Context(), actorID, tenantID, call.ID.String(), "capital_call", "create", nil, call)
	h.emitWebhookEvent(r.Context(), "altinvestments.capital_call.created", map[string]interface{}{
		"capital_call": call,
	}, map[string]string{
		"tenant_id":     tenantID,
		"datasource_id": datasourceID,
		"actor_id":      actorID,
	})

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(call)
}

// ListCapitalCalls lists all capital calls for an investment
func (h *AlternativeInvestmentHandlers) ListCapitalCalls(w http.ResponseWriter, r *http.Request) {
	actorID, tenantID, _, ok := h.authorize(w, r, permAltInvRead)
	if !ok {
		return
	}

	investmentID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "Invalid investment ID", "invalid_investment_id", nil)
		return
	}

	calls, err := h.altInvService.GetCapitalCallsByInvestment(r.Context(), investmentID)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error(), "list_failed", nil)
		return
	}

	h.auditAccess(r.Context(), actorID, tenantID, investmentID.String(), "capital_call", "list", nil)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(calls)
}

// FundCapitalCall marks a capital call as funded
func (h *AlternativeInvestmentHandlers) FundCapitalCall(w http.ResponseWriter, r *http.Request) {
	actorID, tenantID, datasourceID, ok := h.authorize(w, r, permAltInvWrite)
	if !ok {
		return
	}

	callID, err := uuid.Parse(chi.URLParam(r, "callId"))
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "Invalid call ID", "invalid_call_id", nil)
		return
	}

	var req struct {
		Amount     float64 `json:"amount"`
		FundedDate string  `json:"fundedDate"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "Invalid request body", "invalid_payload", nil)
		return
	}

	fundedDate, err := time.Parse("2006-01-02", req.FundedDate)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "Invalid fundedDate format", "invalid_date", nil)
		return
	}

	if err := h.altInvService.FundCapitalCall(r.Context(), callID, req.Amount, fundedDate); err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error(), "update_failed", nil)
		return
	}

	details := map[string]interface{}{
		"amount":      req.Amount,
		"funded_date": fundedDate.Format(time.RFC3339),
	}
	h.auditModification(r.Context(), actorID, tenantID, callID.String(), "capital_call", "fund", nil, details)
	h.emitWebhookEvent(r.Context(), "altinvestments.capital_call.funded", map[string]interface{}{
		"capital_call_id": callID.String(),
		"amount":          req.Amount,
	}, map[string]string{
		"tenant_id":     tenantID,
		"datasource_id": datasourceID,
		"actor_id":      actorID,
	})

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "funded"})
}

// CreateDistribution creates a new distribution
func (h *AlternativeInvestmentHandlers) CreateDistribution(w http.ResponseWriter, r *http.Request) {
	actorID, tenantID, datasourceID, ok := h.authorize(w, r, permAltInvWrite)
	if !ok {
		return
	}

	investmentID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "Invalid investment ID", "invalid_investment_id", nil)
		return
	}

	var dist types.CapitalDistribution
	if err := json.NewDecoder(r.Body).Decode(&dist); err != nil {
		writeJSONError(w, http.StatusBadRequest, "Invalid request body", "invalid_payload", nil)
		return
	}

	dist.InvestmentID = investmentID

	if err := h.altInvService.RecordDistribution(r.Context(), &dist); err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error(), "create_failed", nil)
		return
	}

	h.auditModification(r.Context(), actorID, tenantID, dist.ID.String(), "capital_distribution", "create", nil, dist)
	h.emitWebhookEvent(r.Context(), "altinvestments.distribution.created", map[string]interface{}{"distribution": dist}, map[string]string{
		"tenant_id":     tenantID,
		"datasource_id": datasourceID,
		"actor_id":      actorID,
	})

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(dist)
}

// ListDistributions lists all distributions for an investment
func (h *AlternativeInvestmentHandlers) ListDistributions(w http.ResponseWriter, r *http.Request) {
	actorID, tenantID, _, ok := h.authorize(w, r, permAltInvRead)
	if !ok {
		return
	}

	investmentID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "Invalid investment ID", "invalid_investment_id", nil)
		return
	}

	distributions, err := h.altInvService.GetDistributionsByInvestment(r.Context(), investmentID)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error(), "list_failed", nil)
		return
	}

	h.auditAccess(r.Context(), actorID, tenantID, investmentID.String(), "capital_distribution", "list", nil)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(distributions)
}

// GetPerformance retrieves current performance metrics
func (h *AlternativeInvestmentHandlers) GetPerformance(w http.ResponseWriter, r *http.Request) {
	actorID, tenantID, _, ok := h.authorize(w, r, permAltInvRead)
	if !ok {
		return
	}

	investmentID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "Invalid investment ID", "invalid_investment_id", nil)
		return
	}

	asOfDateStr := r.URL.Query().Get("as_of_date")
	var asOfDate *time.Time
	if asOfDateStr != "" {
		parsed, err := time.Parse("2006-01-02", asOfDateStr)
		if err != nil {
			writeJSONError(w, http.StatusBadRequest, "Invalid date format", "invalid_date", nil)
			return
		}
		asOfDate = &parsed
	}

	perf, err := h.perfService.GetPerformanceMetrics(r.Context(), investmentID, asOfDate)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error(), "lookup_failed", nil)
		return
	}

	if perf == nil {
		writeJSONError(w, http.StatusNotFound, "No performance metrics found", "not_found", nil)
		return
	}

	h.auditAccess(r.Context(), actorID, tenantID, investmentID.String(), "investment_performance", "read", map[string]interface{}{
		"as_of_date": asOfDateStr,
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(perf)
}

// CalculatePerformance triggers performance calculation
func (h *AlternativeInvestmentHandlers) CalculatePerformance(w http.ResponseWriter, r *http.Request) {
	actorID, tenantID, datasourceID, ok := h.authorize(w, r, permAltInvWrite)
	if !ok {
		return
	}

	investmentID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "Invalid investment ID", "invalid_investment_id", nil)
		return
	}

	asOfDate := time.Now()
	asOfDateStr := r.URL.Query().Get("as_of_date")
	if asOfDateStr != "" {
		asOfDate, err = time.Parse("2006-01-02", asOfDateStr)
		if err != nil {
			writeJSONError(w, http.StatusBadRequest, "Invalid date format", "invalid_date", nil)
			return
		}
	}

	tenantUUID, err := uuid.Parse(tenantID)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "Invalid tenant ID", "invalid_tenant", nil)
		return
	}

	perf, err := h.perfService.CalculateAndSavePerformanceMetrics(r.Context(), tenantUUID, investmentID, asOfDate)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error(), "calculation_failed", nil)
		return
	}

	details := map[string]interface{}{
		"as_of_date": asOfDate.Format(time.RFC3339),
	}
	h.auditModification(r.Context(), actorID, tenantID, investmentID.String(), "investment_performance", "calculate", nil, details)
	h.emitWebhookEvent(r.Context(), "altinvestments.performance.calculated", map[string]interface{}{"performance": perf}, map[string]string{
		"tenant_id":     tenantID,
		"datasource_id": datasourceID,
		"actor_id":      actorID,
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(perf)
}

// GetPerformanceHistory retrieves performance history
func (h *AlternativeInvestmentHandlers) GetPerformanceHistory(w http.ResponseWriter, r *http.Request) {
	actorID, tenantID, _, ok := h.authorize(w, r, permAltInvRead)
	if !ok {
		return
	}

	investmentID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "Invalid investment ID", "invalid_investment_id", nil)
		return
	}

	history, err := h.perfService.GetPerformanceHistory(r.Context(), investmentID, 12) // Last 12 periods
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error(), "lookup_failed", nil)
		return
	}

	h.auditAccess(r.Context(), actorID, tenantID, investmentID.String(), "investment_performance", "history", map[string]interface{}{"periods": 12})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(history)
}

// UploadDocument handles document upload
func (h *AlternativeInvestmentHandlers) UploadDocument(w http.ResponseWriter, r *http.Request) {
	actorID, tenantID, datasourceID, ok := h.authorize(w, r, permAltInvWrite)
	if !ok {
		return
	}

	investmentID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "Invalid investment ID", "invalid_investment_id", nil)
		return
	}

	// Parse multipart form
	if err := r.ParseMultipartForm(32 << 20); err != nil { // 32 MB max
		writeJSONError(w, http.StatusBadRequest, "Failed to parse form", "invalid_form", nil)
		return
	}

	file, handler, err := r.FormFile("file")
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "Failed to read file", "invalid_file", nil)
		return
	}
	defer file.Close()

	documentType := r.FormValue("document_type")
	if documentType == "" {
		writeJSONError(w, http.StatusBadRequest, "document_type required", "missing_document_type", nil)
		return
	}

	// TODO: Save file to storage (S3, local filesystem, etc.)
	filePath := "/path/to/storage/" + handler.Filename
	mimeType := handler.Header.Get("Content-Type")
	var uploadedBy *uuid.UUID
	if parsedActor, err := uuid.Parse(actorID); err == nil {
		uploadedBy = &parsedActor
	}

	doc := &types.AlternativeInvestmentDocument{
		InvestmentID: investmentID,
		DocumentType: documentType,
		FileName:     handler.Filename,
		FilePath:     filePath,
		MimeType:     &mimeType,
		UploadedBy:   uploadedBy,
	}

	if err := h.docService.UploadDocument(r.Context(), doc); err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error(), "upload_failed", nil)
		return
	}

	// Trigger processing asynchronously
	go h.docService.ProcessDocument(context.Background(), doc.ID)

	h.auditModification(r.Context(), actorID, tenantID, doc.ID.String(), "alternative_investment_document", "upload", nil, doc)
	h.emitWebhookEvent(r.Context(), "altinvestments.document.uploaded", map[string]interface{}{"document": doc}, map[string]string{
		"tenant_id":     tenantID,
		"datasource_id": datasourceID,
		"actor_id":      actorID,
	})

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(doc)
}

// ListDocuments lists all documents for an investment
func (h *AlternativeInvestmentHandlers) ListDocuments(w http.ResponseWriter, r *http.Request) {
	actorID, tenantID, _, ok := h.authorize(w, r, permAltInvRead)
	if !ok {
		return
	}

	investmentID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "Invalid investment ID", "invalid_investment_id", nil)
		return
	}

	documents, err := h.docService.ListDocumentsByInvestment(r.Context(), investmentID)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error(), "list_failed", nil)
		return
	}

	h.auditAccess(r.Context(), actorID, tenantID, investmentID.String(), "alternative_investment_document", "list", nil)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(documents)
}

// ApproveDocument approves a reviewed document
func (h *AlternativeInvestmentHandlers) ApproveDocument(w http.ResponseWriter, r *http.Request) {
	actorID, tenantID, datasourceID, ok := h.authorize(w, r, permAltInvWrite)
	if !ok {
		return
	}

	docID, err := uuid.Parse(chi.URLParam(r, "docId"))
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "Invalid document ID", "invalid_document_id", nil)
		return
	}

	var req struct {
		Notes string `json:"notes"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "Invalid request body", "invalid_payload", nil)
		return
	}

	reviewerID, err := uuid.Parse(actorID)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "Authenticated reviewer must be a UUID", "invalid_actor", nil)
		return
	}

	if err := h.docService.ApproveDocument(r.Context(), docID, reviewerID, req.Notes); err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error(), "approve_failed", nil)
		return
	}

	details := map[string]interface{}{"notes": req.Notes}
	h.auditModification(r.Context(), actorID, tenantID, docID.String(), "alternative_investment_document", "approve", nil, details)
	h.emitWebhookEvent(r.Context(), "altinvestments.document.approved", map[string]interface{}{"document_id": docID.String()}, map[string]string{
		"tenant_id":     tenantID,
		"datasource_id": datasourceID,
		"actor_id":      actorID,
	})

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "approved"})
}

// RejectDocument rejects a reviewed document
func (h *AlternativeInvestmentHandlers) RejectDocument(w http.ResponseWriter, r *http.Request) {
	actorID, tenantID, datasourceID, ok := h.authorize(w, r, permAltInvWrite)
	if !ok {
		return
	}

	docID, err := uuid.Parse(chi.URLParam(r, "docId"))
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "Invalid document ID", "invalid_document_id", nil)
		return
	}

	var req struct {
		Notes string `json:"notes"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "Invalid request body", "invalid_payload", nil)
		return
	}

	reviewerID, err := uuid.Parse(actorID)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "Authenticated reviewer must be a UUID", "invalid_actor", nil)
		return
	}

	if err := h.docService.RejectDocument(r.Context(), docID, reviewerID, req.Notes); err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error(), "reject_failed", nil)
		return
	}

	details := map[string]interface{}{"notes": req.Notes}
	h.auditModification(r.Context(), actorID, tenantID, docID.String(), "alternative_investment_document", "reject", nil, details)
	h.emitWebhookEvent(r.Context(), "altinvestments.document.rejected", map[string]interface{}{"document_id": docID.String()}, map[string]string{
		"tenant_id":     tenantID,
		"datasource_id": datasourceID,
		"actor_id":      actorID,
	})

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "rejected"})
}

// Helper methods for shared authorization/audit/webhook functionality.
func (h *AlternativeInvestmentHandlers) authorize(w http.ResponseWriter, r *http.Request, permission string) (string, string, string, bool) {
	return authorizeRequest(w, r, h.secMgr, permission)
}

func (h *AlternativeInvestmentHandlers) auditAccess(ctx context.Context, actorID, tenantID, objectID, objectType, action string, details map[string]interface{}) {
	logAuditAccess(ctx, h.auditSvc, actorID, tenantID, objectID, objectType, action, details)
}

func (h *AlternativeInvestmentHandlers) auditModification(ctx context.Context, actorID, tenantID, objectID, objectType, action string, oldData, newData interface{}) {
	logAuditModification(ctx, h.auditSvc, actorID, tenantID, objectID, objectType, action, oldData, newData)
}

func (h *AlternativeInvestmentHandlers) emitWebhookEvent(ctx context.Context, eventType string, payload map[string]interface{}, attributes map[string]string) {
	dispatchWebhookEvent(ctx, h.webhookSvc, eventType, payload, attributes)
}
