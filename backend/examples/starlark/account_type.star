# Example rule: validate account type.
#
# Run:
#   cd backend
#   go run ./cmd/starlarktest -file ./examples/starlark/account_type.star
#
# Notes:
# - Top-level maps in the testcase input become business objects in ctx.
#   Here, `account` becomes available via `field("account", "account_type")`.

ok = eq(field("account", "account_type"), "ADVISORY")
message = "Account must be ADVISORY"

# {"name":"advisory","record":{"account":{"account_type":"ADVISORY"}},"expect":true}
# {"name":"brokerage","record":{"account":{"account_type":"BROKERAGE"}},"expect":false}
