package api

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/hondyman/semlayer/backend/internal/wealth"

	"github.com/go-chi/chi/v5"
	"github.com/shopspring/decimal"
)

// WealthTransferHandlers contains handlers for wealth transfer endpoints
type WealthTransferHandlers struct {
	familyOfficeService *wealth.FamilyOfficeService
	taxCalcService      *wealth.TaxCalculationService
	giftHistoryService  *wealth.GiftHistoryService
	trustEntityService  *wealth.TrustEntityService
}

// NewWealthTransferHandlers creates wealth transfer handlers
func NewWealthTransferHandlers(
	familyOfficeService *wealth.FamilyOfficeService,
	taxCalcService *wealth.TaxCalculationService,
	giftHistoryService *wealth.GiftHistoryService,
	trustEntityService *wealth.TrustEntityService,
) *WealthTransferHandlers {
	return &WealthTransferHandlers{
		familyOfficeService: familyOfficeService,
		taxCalcService:      taxCalcService,
		giftHistoryService:  giftHistoryService,
		trustEntityService:  trustEntityService,
	}
}

// RegisterRoutes registers all wealth transfer routes
func (h *WealthTransferHandlers) RegisterRoutes(r chi.Router) {
	r.Route("/api/wealth-transfer", func(r chi.Router) {
		// Family Office routes
		r.Post("/families", h.CreateFamilyOffice)
		r.Get("/families", h.ListFamilyOffices)
		r.Get("/families/{familyID}", h.GetFamilyOffice)
		r.Get("/families/{familyID}/tree", h.GetFamilyTree)
		r.Get("/families/{familyID}/profile", h.GetFamilyProfile)

		// Family Member routes
		r.Post("/families/{familyID}/members", h.AddFamilyMember)
		r.Get("/families/{familyID}/members", h.GetFamilyMembers)

		// Tax Calculation routes
		r.Post("/tax/estate/federal", h.CalculateFederalEstateTax)
		r.Post("/tax/estate/state", h.CalculateStateTax)
		r.Post("/tax/estate/combined", h.CalculateCombinedEstateTax)
		r.Post("/tax/gift", h.CalculateGiftTax)
		r.Post("/tax/gst", h.CalculateGSTTax)

		// Gift History routes
		r.Post("/gifts", h.RecordGift)
		r.Get("/families/{familyID}/gifts", h.GetGiftHistory)
		r.Get("/families/{familyID}/gifts/pending-form-709", h.GetPendingForm709Filings)
		r.Post("/gifts/{giftID}/mark-filed", h.MarkForm709Filed)
		r.Get("/families/{familyID}/members/{memberID}/exemptions", h.GetExemptionSummary)

		// Trust Entity routes
		r.Post("/trusts", h.CreateTrust)
		r.Get("/families/{familyID}/trusts", h.ListTrusts)
		r.Get("/trusts/{entityID}", h.GetTrust)
		r.Get("/trusts/{entityID}/compliance", h.ValidateTrustCompliance)
		r.Get("/trusts/{entityID}/value", h.CalculateTrustValue)
		r.Post("/trusts/{entityID}/tax-filing", h.UpdateTaxFilingStatus)
		r.Post("/trusts/{entityID}/terminate", h.TerminateTrust)
	})
}

//==============================================================================
// FAMILY OFFICE HANDLERS
// ==============================================================================

func (h *WealthTransferHandlers) CreateFamilyOffice(w http.ResponseWriter, r *http.Request) {
	var input wealth.CreateFamilyOfficeInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	family, err := h.familyOfficeService.CreateFamilyOffice(r.Context(), input)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(family)
}

func (h *WealthTransferHandlers) GetFamilyOffice(w http.ResponseWriter, r *http.Request) {
	familyID := chi.URLParam(r, "familyID")

	family, err := h.familyOfficeService.GetFamilyOffice(r.Context(), familyID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(family)
}

func (h *WealthTransferHandlers) ListFamilyOffices(w http.ResponseWriter, r *http.Request) {
	tenantID := r.URL.Query().Get("tenant_id")
	if tenantID == "" {
		http.Error(w, "tenant_id required", http.StatusBadRequest)
		return
	}

	families, err := h.familyOfficeService.ListFamilyOffices(r.Context(), tenantID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(families)
}

func (h *WealthTransferHandlers) GetFamilyTree(w http.ResponseWriter, r *http.Request) {
	familyID := chi.URLParam(r, "familyID")

	tree, err := h.familyOfficeService.GetFamilyTree(r.Context(), familyID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(tree)
}

func (h *WealthTransferHandlers) GetFamilyProfile(w http.ResponseWriter, r *http.Request) {
	familyID := chi.URLParam(r, "familyID")

	profile, err := h.familyOfficeService.GetFamilyProfile(r.Context(), familyID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(profile)
}

func (h *WealthTransferHandlers) AddFamilyMember(w http.ResponseWriter, r *http.Request) {
	familyID := chi.URLParam(r, "familyID")

	var input wealth.AddFamilyMemberInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	input.FamilyID = familyID

	member, err := h.familyOfficeService.AddFamilyMember(r.Context(), input)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(member)
}

func (h *WealthTransferHandlers) GetFamilyMembers(w http.ResponseWriter, r *http.Request) {
	familyID := chi.URLParam(r, "familyID")

	members, err := h.familyOfficeService.GetFamilyMembers(r.Context(), familyID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(members)
}

// ==============================================================================
// TAX CALCULATION HANDLERS
// ==============================================================================

type FederalEstateTaxRequest struct {
	GrossEstate        decimal.Decimal `json:"gross_estate"`
	PriorExemptionUsed decimal.Decimal `json:"prior_exemption_used"`
	Year               *int            `json:"year,omitempty"`
}

func (h *WealthTransferHandlers) CalculateFederalEstateTax(w http.ResponseWriter, r *http.Request) {
	var req FederalEstateTaxRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	result, err := h.taxCalcService.CalculateFederalEstateTax(r.Context(), req.GrossEstate, req.PriorExemptionUsed, req.Year)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

type StateTaxRequest struct {
	StateCode   string          `json:"state_code"`
	GrossEstate decimal.Decimal `json:"gross_estate"`
	Year        *int            `json:"year,omitempty"`
}

func (h *WealthTransferHandlers) CalculateStateTax(w http.ResponseWriter, r *http.Request) {
	var req StateTaxRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	result, err := h.taxCalcService.CalculateStateTax(r.Context(), req.StateCode, req.GrossEstate, req.Year)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

type CombinedEstateTaxRequest struct {
	StateCode                 string          `json:"state_code"`
	GrossEstate               decimal.Decimal `json:"gross_estate"`
	PriorFederalExemptionUsed decimal.Decimal `json:"prior_federal_exemption_used"`
	Year                      *int            `json:"year,omitempty"`
}

func (h *WealthTransferHandlers) CalculateCombinedEstateTax(w http.ResponseWriter, r *http.Request) {
	var req CombinedEstateTaxRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	result, err := h.taxCalcService.CalculateCombinedEstateTax(
		r.Context(),
		req.StateCode,
		req.GrossEstate,
		req.PriorFederalExemptionUsed,
		req.Year,
	)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

type GiftTaxRequest struct {
	GiftValue                   decimal.Decimal `json:"gift_value"`
	AnnualExclusionUsedThisYear decimal.Decimal `json:"annual_exclusion_used_this_year"`
	LifetimeExemptionUsedPrior  decimal.Decimal `json:"lifetime_exemption_used_prior"`
	SpousalSplit                bool            `json:"spousal_split"`
	Year                        *int            `json:"year,omitempty"`
}

func (h *WealthTransferHandlers) CalculateGiftTax(w http.ResponseWriter, r *http.Request) {
	var req GiftTaxRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	result, err := h.taxCalcService.CalculateGiftTax(
		r.Context(),
		req.GiftValue,
		req.AnnualExclusionUsedThisYear,
		req.LifetimeExemptionUsedPrior,
		req.SpousalSplit,
		req.Year,
	)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

type GSTTaxRequest struct {
	TransferValue         decimal.Decimal `json:"transfer_value"`
	GSTExemptionUsedPrior decimal.Decimal `json:"gst_exemption_used_prior"`
	GenerationsSkipped    int             `json:"generations_skipped"`
	Year                  *int            `json:"year,omitempty"`
}

func (h *WealthTransferHandlers) CalculateGSTTax(w http.ResponseWriter, r *http.Request) {
	var req GSTTaxRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	result, err := h.taxCalcService.CalculateGSTTax(
		r.Context(),
		req.TransferValue,
		req.GSTExemptionUsedPrior,
		req.GenerationsSkipped,
		req.Year,
	)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// ==============================================================================
// GIFT HISTORY HANDLERS
// ==============================================================================

func (h *WealthTransferHandlers) RecordGift(w http.ResponseWriter, r *http.Request) {
	var input wealth.RecordGiftInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	gift, err := h.giftHistoryService.RecordGift(r.Context(), input)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(gift)
}

func (h *WealthTransferHandlers) GetGiftHistory(w http.ResponseWriter, r *http.Request) {
	familyID := chi.URLParam(r, "familyID")

	gifts, err := h.giftHistoryService.GetGiftHistory(r.Context(), familyID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(gifts)
}

func (h *WealthTransferHandlers) GetPendingForm709Filings(w http.ResponseWriter, r *http.Request) {
	familyID := chi.URLParam(r, "familyID")

	gifts, err := h.giftHistoryService.GetPendingForm709Filings(r.Context(), familyID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(gifts)
}

type MarkForm709FiledRequest struct {
	FilingDate time.Time `json:"filing_date"`
	DocumentID *string   `json:"document_id,omitempty"`
}

func (h *WealthTransferHandlers) MarkForm709Filed(w http.ResponseWriter, r *http.Request) {
	giftID := chi.URLParam(r, "giftID")

	var req MarkForm709FiledRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	err := h.giftHistoryService.MarkForm709Filed(r.Context(), giftID, req.FilingDate, req.DocumentID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *WealthTransferHandlers) GetExemptionSummary(w http.ResponseWriter, r *http.Request) {
	familyID := chi.URLParam(r, "familyID")
	memberID := chi.URLParam(r, "memberID")

	summary, err := h.giftHistoryService.GetExemptionSummary(r.Context(), familyID, memberID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(summary)
}

// ==============================================================================
// TRUST ENTITY HANDLERS
// ==============================================================================

func (h *WealthTransferHandlers) CreateTrust(w http.ResponseWriter, r *http.Request) {
	var input wealth.CreateTrustInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	trust, err := h.trustEntityService.CreateTrust(r.Context(), input)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(trust)
}

func (h *WealthTransferHandlers) GetTrust(w http.ResponseWriter, r *http.Request) {
	entityID := chi.URLParam(r, "entityID")

	trust, err := h.trustEntityService.GetTrust(r.Context(), entityID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(trust)
}

func (h *WealthTransferHandlers) ListTrusts(w http.ResponseWriter, r *http.Request) {
	familyID := chi.URLParam(r, "familyID")

	trusts, err := h.trustEntityService.ListTrusts(r.Context(), familyID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(trusts)
}

func (h *WealthTransferHandlers) ValidateTrustCompliance(w http.ResponseWriter, r *http.Request) {
	entityID := chi.URLParam(r, "entityID")

	issues, err := h.trustEntityService.ValidateTrustCompliance(r.Context(), entityID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"entity_id": entityID,
		"issues":    issues,
		"compliant": len(issues) == 0,
	})
}

func (h *WealthTransferHandlers) CalculateTrustValue(w http.ResponseWriter, r *http.Request) {
	entityID := chi.URLParam(r, "entityID")

	value, err := h.trustEntityService.CalculateTrustValue(r.Context(), entityID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"entity_id":   entityID,
		"total_value": value,
	})
}

type UpdateTaxFilingRequest struct {
	FilingDate time.Time `json:"filing_date"`
}

func (h *WealthTransferHandlers) UpdateTaxFilingStatus(w http.ResponseWriter, r *http.Request) {
	entityID := chi.URLParam(r, "entityID")

	var req UpdateTaxFilingRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	err := h.trustEntityService.UpdateTaxFilingStatus(r.Context(), entityID, req.FilingDate)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

type TerminateTrustRequest struct {
	TerminationDate time.Time `json:"termination_date"`
	Reason          string    `json:"reason"`
}

func (h *WealthTransferHandlers) TerminateTrust(w http.ResponseWriter, r *http.Request) {
	entityID := chi.URLParam(r, "entityID")

	var req TerminateTrustRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	err := h.trustEntityService.TerminateTrust(r.Context(), entityID, req.TerminationDate, req.Reason)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
