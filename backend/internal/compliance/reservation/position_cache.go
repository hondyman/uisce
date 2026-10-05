package reservation

import (
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// PositionState holds the in-memory settled and reserved quantities for a specific asset
type PositionState struct {
	TenantID            uuid.UUID       `json:"tenantId"`
	AccountID           uuid.UUID       `json:"accountId"`
	SecurityID          uuid.UUID       `json:"securityId"`
	SettledQuantity     decimal.Decimal `json:"settledQuantity"`
	SettledMarketValue  decimal.Decimal `json:"settledMarketValue"`
	ReservedBuyValue    decimal.Decimal `json:"reservedBuyValue"`
	ReservedSellValue   decimal.Decimal `json:"reservedSellValue"`
	ReservedBuyQuantity decimal.Decimal `json:"reservedBuyQuantity"`
	LastUpdated         time.Time       `json:"lastUpdated"`
}

// TotalProjectedValue returns settled value + pending in-flight buy reservations
func (p *PositionState) TotalProjectedValue() decimal.Decimal {
	return p.SettledMarketValue.Add(p.ReservedBuyValue)
}

// PositionCache is a thread-safe, lock-free process-local position store
type PositionCache struct {
	mu        sync.RWMutex
	positions map[string]*PositionState // Key: tenantID:accountID:securityID
	accountNAV map[string]decimal.Decimal // Key: tenantID:accountID -> NAV
}

// NewPositionCache creates a new position cache
func NewPositionCache() *PositionCache {
	return &PositionCache{
		positions:  make(map[string]*PositionState),
		accountNAV: make(map[string]decimal.Decimal),
	}
}

func cacheKey(tenantID, accountID, securityID uuid.UUID) string {
	return tenantID.String() + ":" + accountID.String() + ":" + securityID.String()
}

func navKey(tenantID, accountID uuid.UUID) string {
	return tenantID.String() + ":" + accountID.String()
}

// SetAccountNAV updates the current portfolio NAV
func (c *PositionCache) SetAccountNAV(tenantID, accountID uuid.UUID, nav decimal.Decimal) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.accountNAV[navKey(tenantID, accountID)] = nav
}

// GetAccountNAV retrieves the cached NAV
func (c *PositionCache) GetAccountNAV(tenantID, accountID uuid.UUID) decimal.Decimal {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.accountNAV[navKey(tenantID, accountID)]
}

// SetSettledPosition updates or sets the settled base position
func (c *PositionCache) SetSettledPosition(tenantID, accountID, securityID uuid.UUID, qty, marketVal decimal.Decimal) {
	c.mu.Lock()
	defer c.mu.Unlock()

	k := cacheKey(tenantID, accountID, securityID)
	pos, exists := c.positions[k]
	if !exists {
		pos = &PositionState{
			TenantID:   tenantID,
			AccountID:  accountID,
			SecurityID: securityID,
		}
		c.positions[k] = pos
	}

	pos.SettledQuantity = qty
	pos.SettledMarketValue = marketVal
	pos.LastUpdated = time.Now().UTC()
}

// GetPosition retrieves current position state including active reservations
func (c *PositionCache) GetPosition(tenantID, accountID, securityID uuid.UUID) PositionState {
	c.mu.RLock()
	defer c.mu.RUnlock()

	k := cacheKey(tenantID, accountID, securityID)
	pos, exists := c.positions[k]
	if !exists {
		return PositionState{
			TenantID:   tenantID,
			AccountID:  accountID,
			SecurityID: securityID,
		}
	}
	return *pos
}
