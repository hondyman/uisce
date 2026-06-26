package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/hondyman/semlayer/backend/internal/billing"
)

// InvoiceHandlers provides HTTP handlers for the tenant invoice API.
type InvoiceHandlers struct {
	svc *billing.InvoiceService
}

// NewInvoiceHandlers creates new invoice HTTP handlers.
func NewInvoiceHandlers(svc *billing.InvoiceService) *InvoiceHandlers {
	return &InvoiceHandlers{svc: svc}
}

// RegisterRoutes mounts all invoice routes on the chi router.
func (h *InvoiceHandlers) RegisterRoutes(r chi.Router) {
	r.Route("/invoices", func(r chi.Router) {
		// ── Generate Invoice ────────────────────────────
		// POST /api/invoices/generate
		// Body: { "tenantId": "...", "month": "2026-01" }
		r.Post("/generate", h.GenerateInvoice)

		// ── List Invoices (admin: all, or by tenant) ────
		// GET /api/invoices?tenantId=tenant-a
		r.Get("/", h.ListInvoices)

		// ── Get Single Invoice ──────────────────────────
		// GET /api/invoices/{invoiceId}
		r.Get("/{invoiceId}", h.GetInvoice)

		// ── Issue Invoice (DRAFT → ISSUED) ──────────────
		// POST /api/invoices/{invoiceId}/issue
		r.Post("/{invoiceId}/issue", h.IssueInvoice)

		// ── Mark Paid (ISSUED → PAID) ───────────────────
		// POST /api/invoices/{invoiceId}/pay
		r.Post("/{invoiceId}/pay", h.MarkPaid)

		// ── Void Invoice ────────────────────────────────
		// POST /api/invoices/{invoiceId}/void
		// Body: { "reason": "..." }
		r.Post("/{invoiceId}/void", h.VoidInvoice)

		// ── Add Credit ──────────────────────────────────
		// POST /api/invoices/credits
		// Body: BillingCredit
		r.Post("/credits", h.AddCredit)
	})
}

// ─── Generate Invoice ───────────────────────────────────────────

type generateInvoiceRequest struct {
	TenantID string `json:"tenantId"`
	Month    string `json:"month"` // "2026-01" or "current"
}

func (h *InvoiceHandlers) GenerateInvoice(w http.ResponseWriter, r *http.Request) {
	var req generateInvoiceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if req.TenantID == "" {
		jsonError(w, "tenantId is required", http.StatusBadRequest)
		return
	}
	if req.Month == "" {
		req.Month = "current"
	}

	invoice, err := h.svc.GenerateMonthlyInvoice(r.Context(), req.TenantID, req.Month)
	if err != nil {
		jsonError(w, fmt.Sprintf("failed to generate invoice: %v", err), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	jsonOK(w, invoice)
}

// ─── List Invoices ──────────────────────────────────────────────

func (h *InvoiceHandlers) ListInvoices(w http.ResponseWriter, r *http.Request) {
	tenantID := r.URL.Query().Get("tenantId")

	var summaries []billing.InvoiceSummary
	if tenantID != "" {
		summaries = h.svc.ListTenantInvoices(tenantID)
	} else {
		summaries = h.svc.ListAllInvoices()
	}

	jsonOK(w, summaries)
}

// ─── Get Invoice Detail ─────────────────────────────────────────

func (h *InvoiceHandlers) GetInvoice(w http.ResponseWriter, r *http.Request) {
	invoiceID := chi.URLParam(r, "invoiceId")
	if invoiceID == "" {
		jsonError(w, "invoiceId is required", http.StatusBadRequest)
		return
	}

	invoice, err := h.svc.GetInvoice(invoiceID)
	if err != nil {
		jsonError(w, err.Error(), http.StatusNotFound)
		return
	}

	jsonOK(w, invoice)
}

// ─── Issue Invoice ──────────────────────────────────────────────

func (h *InvoiceHandlers) IssueInvoice(w http.ResponseWriter, r *http.Request) {
	invoiceID := chi.URLParam(r, "invoiceId")
	if invoiceID == "" {
		jsonError(w, "invoiceId is required", http.StatusBadRequest)
		return
	}

	invoice, err := h.svc.IssueInvoice(invoiceID)
	if err != nil {
		jsonError(w, err.Error(), http.StatusBadRequest)
		return
	}

	jsonOK(w, invoice)
}

// ─── Mark Paid ──────────────────────────────────────────────────

func (h *InvoiceHandlers) MarkPaid(w http.ResponseWriter, r *http.Request) {
	invoiceID := chi.URLParam(r, "invoiceId")
	if invoiceID == "" {
		jsonError(w, "invoiceId is required", http.StatusBadRequest)
		return
	}

	invoice, err := h.svc.MarkPaid(invoiceID)
	if err != nil {
		jsonError(w, err.Error(), http.StatusBadRequest)
		return
	}

	jsonOK(w, invoice)
}

// ─── Void Invoice ───────────────────────────────────────────────

type voidInvoiceRequest struct {
	Reason string `json:"reason"`
}

func (h *InvoiceHandlers) VoidInvoice(w http.ResponseWriter, r *http.Request) {
	invoiceID := chi.URLParam(r, "invoiceId")
	if invoiceID == "" {
		jsonError(w, "invoiceId is required", http.StatusBadRequest)
		return
	}

	var req voidInvoiceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if req.Reason == "" {
		jsonError(w, "reason is required", http.StatusBadRequest)
		return
	}

	invoice, err := h.svc.VoidInvoice(invoiceID, req.Reason)
	if err != nil {
		jsonError(w, err.Error(), http.StatusBadRequest)
		return
	}

	jsonOK(w, invoice)
}

// ─── Add Credit ─────────────────────────────────────────────────

type addCreditRequest struct {
	TenantID  string  `json:"tenantId"`
	AmountUSD float64 `json:"amountUSD"`
	Reason    string  `json:"reason"`
	ExpiresIn int     `json:"expiresInDays"` // days until expiry
}

func (h *InvoiceHandlers) AddCredit(w http.ResponseWriter, r *http.Request) {
	var req addCreditRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if req.TenantID == "" || req.AmountUSD <= 0 || req.Reason == "" {
		jsonError(w, "tenantId, amountUSD (>0), and reason are required", http.StatusBadRequest)
		return
	}

	if req.ExpiresIn <= 0 {
		req.ExpiresIn = 365 // default: 1 year
	}

	now := time.Now().UTC()
	credit := billing.BillingCredit{
		ID:        fmt.Sprintf("crd-%d", now.UnixNano()),
		TenantID:  req.TenantID,
		AmountUSD: req.AmountUSD,
		Reason:    req.Reason,
		ExpiresAt: now.AddDate(0, 0, req.ExpiresIn),
		CreatedAt: now,
		CreatedBy: "api", // would come from auth context
	}

	h.svc.AddCredit(credit)

	w.WriteHeader(http.StatusCreated)
	jsonOK(w, credit)
}
