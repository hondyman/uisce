package compliance

import "time"

// --- 2021 Historical Rules (Different business logic) ---
#Amount: >0 & <500_000_000 // Lower max in 2021
#Currency: "USD" | "EUR" | "GBP"
#TradeDate: string & time.Format("2006-01-02")

#Trade: {
	id:          string
	tradeDate:   #TradeDate
	amount:      #Amount
	currency:    #Currency
	securityId:  string
	orderType:   "LIMIT" | "MARKET" | "STOP"
	
	if orderType == "LIMIT" {
		limitPrice: >0
	}
}

// 2021 had more lenient pre-trade rules
#PreTradeCheck: #Trade & {
	// In 2021, high value threshold was 2M (vs 1M in 2025)
	if amount > 2_000_000 {
		orderType: "LIMIT"
	}
}

// 2021 post-trade checks
#PostTradeCheck: #Trade & {
	if amount > 10_000_000 {
		approvalStatus: "PENDING_REVIEW" | "APPROVED"
	}
}
