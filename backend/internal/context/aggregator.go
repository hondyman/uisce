package context

import (
	"time"
)

// ClientContext represents the aggregated state of a client for decision making
type ClientContext struct {
	ClientID       string
	TenantID       string
	Profile        ClientProfile
	Portfolio      PortfolioSummary
	RecentSignals  []Signal
	PendingActions []Action
	Compliance     ComplianceStatus
}

type ClientProfile struct {
	Name      string
	TaxStatus string // e.g., "taxable", "tax_deferred"
	RiskScore int
}

type PortfolioSummary struct {
	TotalValue        float64
	UnrealizedLoss    float64
	UnrealizedLossPct float64
	DriftPct          float64
	CashBalance       float64
}

type Signal struct {
	ID        string
	Type      string // "LOGIN", "MARKET_DROP", "DIVIDEND"
	Timestamp time.Time
	Payload   map[string]interface{}
}

type Action struct {
	ID          string
	Type        string
	Description string
}

type ComplianceStatus struct {
	IsRestricted bool
	Flags        []string
}

// ContextAggregator defines the interface for building client context
type ContextAggregator interface {
	GetContext(tenantID, clientID string) (*ClientContext, error)
}
