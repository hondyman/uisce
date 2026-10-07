package lists

import (
	"time"

	"github.com/google/uuid"
)

// ListType defines the regulatory category of the restricted list.
type ListType string

const (
	ListTypeRestrictedTrading    ListType = "RESTRICTED_TRADING"
	ListTypeWatchlist            ListType = "WATCHLIST"
	ListTypeSanctionsLookthrough ListType = "SANCTIONS_LOOKTHROUGH"
	ListTypeEmployeePreclear     ListType = "EMPLOYEE_PRECLEAR"
)

// MatchType defines how list items are compared against orders and positions.
type MatchType string

const (
	MatchTypeExact            MatchType = "EXACT"
	MatchTypeAlias            MatchType = "ALIAS"
	MatchTypeIdentifierLookup MatchType = "IDENTIFIER_LOOKUP"
)

// RestrictionScope defines what trading activity is constrained.
type RestrictionScope string

const (
	RestrictionScopeAllTrading      RestrictionScope = "ALL_TRADING"
	RestrictionScopeBuysOnly        RestrictionScope = "BUYS_ONLY"
	RestrictionScopeSellsOnly       RestrictionScope = "SELLS_ONLY"
	RestrictionScopeDerivativesOnly RestrictionScope = "DERIVATIVES_ONLY"
)

// RestrictedList represents a versioned collection of trading restrictions.
type RestrictedList struct {
	ID             uuid.UUID `json:"id"`
	TenantID       uuid.UUID `json:"tenant_id"`
	ListCode       string    `json:"list_code"`
	ListName       string    `json:"list_name"`
	ListType       ListType  `json:"list_type"`
	IsActive       bool      `json:"is_active"`
	Version        int       `json:"version"`
	SourceFeedName string    `json:"source_feed_name"`
	LastRefreshedAt time.Time `json:"last_refreshed_at"`
	ItemCount      int       `json:"item_count"`
	ContentHash    string    `json:"content_hash"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// RestrictedListItem represents a single restricted security, issuer, or entity.
type RestrictedListItem struct {
	ID                 uuid.UUID        `json:"id"`
	TenantID           uuid.UUID        `json:"tenant_id"`
	ListID             uuid.UUID        `json:"list_id"`
	SecurityID         string           `json:"security_id,omitempty"`
	IssuerID           string           `json:"issuer_id"`
	IssuerName         string           `json:"issuer_name,omitempty"`
	MatchKey           string           `json:"match_key,omitempty"`
	MatchType          MatchType        `json:"match_type"`
	RestrictionReason  string           `json:"restriction_reason"`
	RestrictionScope   RestrictionScope `json:"restriction_scope"`
	EffectiveFrom      time.Time        `json:"effective_from"`
	EffectiveTo        *time.Time       `json:"effective_to,omitempty"`
	CreatedAt          time.Time        `json:"created_at"`
}

// StaleListAlert represents an operational compliance alert for an un-refreshed list.
type StaleListAlert struct {
	ListID          uuid.UUID `json:"list_id"`
	ListCode        string    `json:"list_code"`
	ListName        string    `json:"list_name"`
	LastRefreshedAt time.Time `json:"last_refreshed_at"`
	AgeHours        float64   `json:"age_hours"`
	Severity        string    `json:"severity"`
	Message         string    `json:"message"`
}
