package wealth.washsale

default allow_harvest = true

# Inputs:
# proposal.trades: array of { side, symbol, qty }
# positions: recent trades/positions with buy dates per symbol
# rules.wash_sale_days: int (usually 30)

deny_reasons[msg] {
  some i
  trade := input.proposal.trades[i]
  trade.side == "SELL"
  
  # Check if we bought this symbol recently
  recent_buy := recent_buys[trade.symbol]
  
  # In a real implementation, we'd do precise date math here.
  # For this demo, recent_buys is pre-filtered or contains timestamp logic.
  
  msg := sprintf("wash_sale_risk:%s", [trade.symbol])
}

# Set of symbols bought recently (within wash sale window)
recent_buys[sym] {
  some j
  pos := input.positions[j]
  pos.side == "BUY"
  
  # Placeholder for date logic: check if pos.timestamp is within window
  # For demo, we assume input.positions ONLY contains recent buys
  sym := pos.symbol
}

allow_harvest {
  count(deny_reasons) == 0
}
