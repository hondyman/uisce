package billing

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"
	"time"
)

// ─── Invoice Status Lifecycle ───────────────────────────────────
// Draft → Issued → Paid
//                → Overdue
//                → Void (manual cancel)

// InvoiceStatus represents the current state of an invoice.
type InvoiceStatus string

const (
	InvoiceStatusDraft   InvoiceStatus = "DRAFT"
	InvoiceStatusIssued  InvoiceStatus = "ISSUED"
	InvoiceStatusPaid    InvoiceStatus = "PAID"
	InvoiceStatusOverdue InvoiceStatus = "OVERDUE"
	InvoiceStatusVoid    InvoiceStatus = "VOID"
)

// ─── Enhanced Invoice Types ─────────────────────────────────────

// Invoice represents a complete monthly invoice for a tenant.
type Invoice struct {
	// Identity
	InvoiceID     string        `json:"invoiceId"`
	InvoiceNumber string        `json:"invoiceNumber"` // human-readable, e.g. INV-2026-01-0001
	TenantID      string        `json:"tenantId"`
	Status        InvoiceStatus `json:"status"`

	// Period
	PeriodStart time.Time `json:"periodStart"`
	PeriodEnd   time.Time `json:"periodEnd"`
	PeriodLabel string    `json:"periodLabel"` // "2026-01"

	// Costs
	LineItems   []DetailedLineItem `json:"lineItems"`
	SubtotalUSD float64            `json:"subtotalUSD"`
	CreditsUSD  float64            `json:"creditsUSD"`  // credits applied (negative)
	DiscountUSD float64            `json:"discountUSD"` // volumetric / promo discount
	TaxUSD      float64            `json:"taxUSD"`      // sales tax if applicable
	TotalDueUSD float64            `json:"totalDueUSD"`

	// Applied credits
	AppliedCredits []AppliedCredit `json:"appliedCredits,omitempty"`

	// Metadata
	IssuedAt  *time.Time `json:"issuedAt,omitempty"`
	DueAt     *time.Time `json:"dueAt,omitempty"`
	PaidAt    *time.Time `json:"paidAt,omitempty"`
	CreatedAt time.Time  `json:"createdAt"`
	CostModel CostModel  `json:"costModel"` // snapshot of cost model used

	// Notes
	Notes string `json:"notes,omitempty"`
}

// DetailedLineItem is a single cost line with usage and unit context.
type DetailedLineItem struct {
	Type        string  `json:"type"`        // compute, storage, events, overage, slo_breach
	Description string  `json:"description"` // human-readable description
	Quantity    float64 `json:"quantity"`    // usage quantity (ms, GB, events)
	UnitLabel   string  `json:"unitLabel"`   // "ms", "GB-month", "events"
	UnitPrice   float64 `json:"unitPrice"`   // price per unit
	AmountUSD   float64 `json:"amountUSD"`   // quantity * unitPrice
}

// AppliedCredit tracks a specific credit applied to an invoice.
type AppliedCredit struct {
	CreditID  string  `json:"creditId"`
	Reason    string  `json:"reason"`
	AmountUSD float64 `json:"amountUSD"`
}

// InvoiceSummary is a lightweight item for listing invoices.
type InvoiceSummary struct {
	InvoiceID     string        `json:"invoiceId"`
	InvoiceNumber string        `json:"invoiceNumber"`
	TenantID      string        `json:"tenantId"`
	PeriodLabel   string        `json:"periodLabel"`
	Status        InvoiceStatus `json:"status"`
	TotalDueUSD   float64       `json:"totalDueUSD"`
	IssuedAt      *time.Time    `json:"issuedAt,omitempty"`
	DueAt         *time.Time    `json:"dueAt,omitempty"`
}

// ─── Invoice Store (in-memory, swap for DB later) ───────────────

// InvoiceStore persists invoices. The in-memory implementation is
// a placeholder; swap with a database-backed store in production.
type InvoiceStore struct {
	invoices map[string]*Invoice        // invoiceID → Invoice
	credits  map[string][]BillingCredit // tenantID → credits
	sequence int
}

// NewInvoiceStore creates an empty in-memory invoice store.
func NewInvoiceStore() *InvoiceStore {
	return &InvoiceStore{
		invoices: make(map[string]*Invoice),
		credits:  make(map[string][]BillingCredit),
		sequence: 0,
	}
}

// Save persists an invoice.
func (s *InvoiceStore) Save(inv *Invoice) {
	s.invoices[inv.InvoiceID] = inv
}

// Get retrieves an invoice by ID.
func (s *InvoiceStore) Get(invoiceID string) (*Invoice, bool) {
	inv, ok := s.invoices[invoiceID]
	return inv, ok
}

// ListByTenant returns all invoices for a tenant, sorted newest-first.
func (s *InvoiceStore) ListByTenant(tenantID string) []*Invoice {
	var result []*Invoice
	for _, inv := range s.invoices {
		if inv.TenantID == tenantID {
			result = append(result, inv)
		}
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].PeriodEnd.After(result[j].PeriodEnd)
	})
	return result
}

// ListAll returns all invoices, sorted newest-first.
func (s *InvoiceStore) ListAll() []*Invoice {
	var result []*Invoice
	for _, inv := range s.invoices {
		result = append(result, inv)
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].PeriodEnd.After(result[j].PeriodEnd)
	})
	return result
}

// NextSequence returns an incrementing sequence number for invoice numbers.
func (s *InvoiceStore) NextSequence() int {
	s.sequence++
	return s.sequence
}

// AddCredit adds a billing credit for a tenant.
func (s *InvoiceStore) AddCredit(credit BillingCredit) {
	s.credits[credit.TenantID] = append(s.credits[credit.TenantID], credit)
}

// GetCredits returns all active (unexpired) credits for a tenant.
func (s *InvoiceStore) GetCredits(tenantID string) []BillingCredit {
	now := time.Now()
	var active []BillingCredit
	for _, c := range s.credits[tenantID] {
		if c.ExpiresAt.After(now) && c.AmountUSD > 0 {
			active = append(active, c)
		}
	}
	return active
}

// ConsumeCredit reduces a credit's balance (in-place mutation).
func (s *InvoiceStore) ConsumeCredit(creditID, tenantID string, amount float64) {
	for i, c := range s.credits[tenantID] {
		if c.ID == creditID {
			s.credits[tenantID][i].AmountUSD -= amount
			if s.credits[tenantID][i].AmountUSD < 0 {
				s.credits[tenantID][i].AmountUSD = 0
			}
			return
		}
	}
}

// ─── Invoice Service ────────────────────────────────────────────

// InvoiceService handles invoice lifecycle: generation, credit
// application, status transitions, and listing.
type InvoiceService struct {
	billing *PlatformBillingService
	store   *InvoiceStore
}

// NewInvoiceService creates a new invoice service.
func NewInvoiceService(billing *PlatformBillingService, store *InvoiceStore) *InvoiceService {
	return &InvoiceService{billing: billing, store: store}
}

// GenerateMonthlyInvoice creates a DRAFT invoice for the given tenant
// and month. The month format is "YYYY-MM" or "current" for the
// current calendar month.
func (svc *InvoiceService) GenerateMonthlyInvoice(ctx context.Context, tenantID, month string) (*Invoice, error) {
	// Parse period
	periodStart, periodEnd, periodLabel := parsePeriod(month)

	// Collect usage from Prometheus
	billingResp, err := svc.billing.GetTenantBilling(ctx, tenantID, "30d")
	if err != nil {
		return nil, fmt.Errorf("invoice: get billing: %w", err)
	}

	// Build detailed line items with unit context
	cost := billingResp.EstimatedCost
	usage := billingResp.Usage
	costModel := svc.billing.cost

	var lineItems []DetailedLineItem

	// Compute
	if cost.ComputeUSD > 0 {
		lineItems = append(lineItems, DetailedLineItem{
			Type:        "compute",
			Description: "Compute time (commit processing)",
			Quantity:    usage.ComputeMs.Total,
			UnitLabel:   "ms",
			UnitPrice:   costModel.CostPerComputeMs,
			AmountUSD:   cost.ComputeUSD,
		})
	}

	// Storage
	if cost.StorageUSD > 0 {
		storageGB := float64(usage.Storage.TotalBytes) / (1024 * 1024 * 1024)
		lineItems = append(lineItems, DetailedLineItem{
			Type:        "storage",
			Description: "Iceberg table storage",
			Quantity:    round2(storageGB),
			UnitLabel:   "GB-month",
			UnitPrice:   costModel.CostPerGBMonth,
			AmountUSD:   cost.StorageUSD,
		})
	}

	// Events
	if cost.EventsUSD > 0 {
		lineItems = append(lineItems, DetailedLineItem{
			Type:        "events",
			Description: "Events published (commits)",
			Quantity:    float64(usage.EventsPublished),
			UnitLabel:   "events",
			UnitPrice:   costModel.CostPerEvent,
			AmountUSD:   cost.EventsUSD,
		})
	}

	// Overage
	if cost.OverageUSD > 0 {
		lineItems = append(lineItems, DetailedLineItem{
			Type:        "overage",
			Description: "Quota overage (events above plan limit)",
			Quantity:    0, // would come from quota tracking
			UnitLabel:   "events",
			UnitPrice:   costModel.CostPerEventOverage,
			AmountUSD:   cost.OverageUSD,
		})
	}

	// SLO breach
	if cost.SLOBreachUSD > 0 {
		lineItems = append(lineItems, DetailedLineItem{
			Type:        "slo_breach",
			Description: "SLO latency threshold breach penalty",
			Quantity:    0,
			UnitLabel:   "ms over SLO",
			UnitPrice:   costModel.CostPerMsOverSLO,
			AmountUSD:   cost.SLOBreachUSD,
		})
	}

	// Calculate subtotal
	subtotal := 0.0
	for _, li := range lineItems {
		subtotal += li.AmountUSD
	}

	// Apply credits
	credits := svc.store.GetCredits(tenantID)
	creditsApplied := 0.0
	var appliedCredits []AppliedCredit
	remaining := subtotal

	for _, c := range credits {
		if remaining <= 0 {
			break
		}
		apply := c.AmountUSD
		if apply > remaining {
			apply = remaining
		}
		creditsApplied += apply
		remaining -= apply
		appliedCredits = append(appliedCredits, AppliedCredit{
			CreditID:  c.ID,
			Reason:    c.Reason,
			AmountUSD: round2(apply),
		})
		svc.store.ConsumeCredit(c.ID, tenantID, apply)
	}

	// Volumetric discount (>$100 subtotal → 5% off)
	discount := 0.0
	if subtotal > 100 {
		discount = round2(subtotal * 0.05)
	}

	totalDue := round2(subtotal - creditsApplied - discount)
	if totalDue < 0 {
		totalDue = 0
	}

	// Generate invoice ID and number
	seq := svc.store.NextSequence()
	invoiceID := generateInvoiceID(tenantID, periodLabel, seq)
	invoiceNumber := fmt.Sprintf("INV-%s-%04d", strings.ReplaceAll(periodLabel, "-", ""), seq)

	now := time.Now().UTC()

	invoice := &Invoice{
		InvoiceID:      invoiceID,
		InvoiceNumber:  invoiceNumber,
		TenantID:       tenantID,
		Status:         InvoiceStatusDraft,
		PeriodStart:    periodStart,
		PeriodEnd:      periodEnd,
		PeriodLabel:    periodLabel,
		LineItems:      lineItems,
		SubtotalUSD:    round2(subtotal),
		CreditsUSD:     round2(creditsApplied),
		DiscountUSD:    discount,
		TaxUSD:         0, // no tax by default
		TotalDueUSD:    totalDue,
		AppliedCredits: appliedCredits,
		CreatedAt:      now,
		CostModel:      costModel,
	}

	svc.store.Save(invoice)
	return invoice, nil
}

// IssueInvoice transitions a DRAFT invoice to ISSUED and sets
// the issue date and 30-day due date.
func (svc *InvoiceService) IssueInvoice(invoiceID string) (*Invoice, error) {
	inv, ok := svc.store.Get(invoiceID)
	if !ok {
		return nil, fmt.Errorf("invoice %s not found", invoiceID)
	}
	if inv.Status != InvoiceStatusDraft {
		return nil, fmt.Errorf("invoice %s is %s, expected DRAFT", invoiceID, inv.Status)
	}

	now := time.Now().UTC()
	due := now.AddDate(0, 0, 30) // net-30
	inv.Status = InvoiceStatusIssued
	inv.IssuedAt = &now
	inv.DueAt = &due

	svc.store.Save(inv)
	return inv, nil
}

// MarkPaid transitions an ISSUED invoice to PAID.
func (svc *InvoiceService) MarkPaid(invoiceID string) (*Invoice, error) {
	inv, ok := svc.store.Get(invoiceID)
	if !ok {
		return nil, fmt.Errorf("invoice %s not found", invoiceID)
	}
	if inv.Status != InvoiceStatusIssued && inv.Status != InvoiceStatusOverdue {
		return nil, fmt.Errorf("invoice %s is %s, expected ISSUED or OVERDUE", invoiceID, inv.Status)
	}

	now := time.Now().UTC()
	inv.Status = InvoiceStatusPaid
	inv.PaidAt = &now

	svc.store.Save(inv)
	return inv, nil
}

// VoidInvoice marks an invoice as VOID (cancelled).
func (svc *InvoiceService) VoidInvoice(invoiceID, reason string) (*Invoice, error) {
	inv, ok := svc.store.Get(invoiceID)
	if !ok {
		return nil, fmt.Errorf("invoice %s not found", invoiceID)
	}
	if inv.Status == InvoiceStatusPaid {
		return nil, fmt.Errorf("cannot void a paid invoice")
	}

	inv.Status = InvoiceStatusVoid
	inv.Notes = "Voided: " + reason

	svc.store.Save(inv)
	return inv, nil
}

// GetInvoice retrieves a single invoice by ID.
func (svc *InvoiceService) GetInvoice(invoiceID string) (*Invoice, error) {
	inv, ok := svc.store.Get(invoiceID)
	if !ok {
		return nil, fmt.Errorf("invoice %s not found", invoiceID)
	}
	return inv, nil
}

// ListTenantInvoices returns all invoices for a tenant.
func (svc *InvoiceService) ListTenantInvoices(tenantID string) []InvoiceSummary {
	invoices := svc.store.ListByTenant(tenantID)
	return toSummaries(invoices)
}

// ListAllInvoices returns all invoices across all tenants (admin view).
func (svc *InvoiceService) ListAllInvoices() []InvoiceSummary {
	invoices := svc.store.ListAll()
	return toSummaries(invoices)
}

// AddCredit adds a billing credit to a tenant's account.
func (svc *InvoiceService) AddCredit(credit BillingCredit) {
	svc.store.AddCredit(credit)
}

// ─── Helpers ────────────────────────────────────────────────────

func toSummaries(invoices []*Invoice) []InvoiceSummary {
	summaries := make([]InvoiceSummary, 0, len(invoices))
	for _, inv := range invoices {
		summaries = append(summaries, InvoiceSummary{
			InvoiceID:     inv.InvoiceID,
			InvoiceNumber: inv.InvoiceNumber,
			TenantID:      inv.TenantID,
			PeriodLabel:   inv.PeriodLabel,
			Status:        inv.Status,
			TotalDueUSD:   inv.TotalDueUSD,
			IssuedAt:      inv.IssuedAt,
			DueAt:         inv.DueAt,
		})
	}
	return summaries
}

func parsePeriod(month string) (start, end time.Time, label string) {
	now := time.Now().UTC()
	if month == "" || month == "current" {
		start = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
		end = start.AddDate(0, 1, -1)
		label = start.Format("2006-01")
		return
	}

	t, err := time.Parse("2006-01", month)
	if err != nil {
		// Fallback to current month
		start = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
		end = start.AddDate(0, 1, -1)
		label = start.Format("2006-01")
		return
	}

	start = t
	end = t.AddDate(0, 1, -1)
	label = month
	return
}

func generateInvoiceID(tenantID, period string, seq int) string {
	raw := fmt.Sprintf("%s-%s-%d-%d", tenantID, period, seq, time.Now().UnixNano())
	hash := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(hash[:16]) // 32-char hex ID
}
