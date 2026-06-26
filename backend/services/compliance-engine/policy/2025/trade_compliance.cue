package compliance

import "time"

// --- 1. Semantic Field Definitions (The "Workday" Business Objects) ---
#Amount: >0 & <1_000_000_000 // Global rule: Positive, max 1B
#Currency: "USD" | "EUR" | "GBP" | "JPY"
#TradeDate: string & time.Format("2006-01-02") // ISO 8601 validation

// --- 2. The Business Object: Trade ---
#Trade: {
	id:          string
	tradeDate:   #TradeDate
	amount:      #Amount
	currency:    #Currency
	securityId:  string
	
	// "Type of Order" Logic
	orderType:   "LIMIT" | "MARKET" | "STOP"
	
	// Dynamic Logic: If it is a LIMIT order, 'limitPrice' is REQUIRED
	if orderType == "LIMIT" {
		limitPrice: >0
	}
}

// --- 3. The Compliance Policy (Pre-Trade) ---
// This acts as the "Gatekeeper"
#PreTradeCheck: #Trade & {
	// Rule: High Value Trade Checks
	if amount > 1_000_000 {
		// High value trades must be Limit orders only for safety
		orderType: "LIMIT"
	}
	
	// Rule: Currency validation - ensure we support the currency
	currency: "USD" | "EUR" | "GBP" | "JPY"
	
	// Rule: Trade date cannot be in the future
	// In practice, this would be a temporal check against current time
}

// --- 4. Post-Trade Compliance (Asynchronous Deep Checks) ---
#PostTradeCheck: #Trade & {
	// Example: Large exposure check
	if amount > 5_000_000 {
		// Require extra manual approval field
		approvalStatus: "PENDING_REVIEW" | "APPROVED"
	}
	
	// Example: Counterparty limits (would require external data enrichment)
	// This is where you'd check wash trades, position limits, etc.
}
