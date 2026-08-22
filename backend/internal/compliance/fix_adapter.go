package compliance

import (
	"strconv"
	"strings"

	"github.com/google/uuid"
)

// ParseFIXNewOrderSingle converts raw FIX Tag 35=D byte streams directly into TradeTickets
// Tags: 11=ClOrdID, 55=Symbol, 54=Side (1=Buy, 2=Sell), 38=OrderQty, 44=Price, 1=Account
func ParseFIXNewOrderSingle(tenantID uuid.UUID, rawFIXMsg string) (*TradeTicket, error) {
	ticket := &TradeTicket{
		TenantID: tenantID,
	}

	fields := strings.Split(rawFIXMsg, "\x01")
	for _, f := range fields {
		if f == "" {
			continue
		}
		kv := strings.SplitN(f, "=", 2)
		if len(kv) != 2 {
			continue
		}
		tag, val := kv[0], kv[1]

		switch tag {
		case "11": // ClOrdID
			ticket.TicketID = val
		case "1": // Account
			ticket.AccountID = val
		case "55": // Symbol
			ticket.SecurityID = val
		case "54": // Side
			if val == "1" {
				ticket.OrderAction = "BUY"
			} else {
				ticket.OrderAction = "SELL"
			}
		case "38": // OrderQty
			ticket.OrderShares, _ = strconv.ParseFloat(val, 64)
		case "44": // Price
			ticket.OrderPrice, _ = strconv.ParseFloat(val, 64)
		}
	}

	return ticket, nil
}
