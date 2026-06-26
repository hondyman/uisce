package semlayer.governance.trades

import future.keywords.if
import future.keywords.in
import future.keywords.contains

default allow = false

# Allow if no deny reasons
allow if count(deny) == 0

# 1. Block high-value trades without specific approval flag
deny contains msg if {
	input.amount >= 1000000
	not input.compliance_approved
	msg := "High-value trade (>1M) requires pre-approval"
}

# 2. Block restricted securities
restricted_symbols := {"RSTR", "BAD", "LOCKED"}
deny contains msg if {
	input.symbol in restricted_symbols
	msg := sprintf("Symbol %v is on the restricted list", [input.symbol])
}

# 3. Exposure limit check (Mock logic: simplistic exposure check)
# In reality, this would likely query an external data source or be injected via 'data'
deny contains msg if {
	input.exposure > 5000000
	msg := "Portfolio exposure limit exceeded (5M)"
}
