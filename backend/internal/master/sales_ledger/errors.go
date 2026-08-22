package sales_ledger

import "errors"

var (
	ErrInvalidSubtype = errors.New("invalid sales_ledger subtype_code")
	ErrNotFound       = errors.New("sales_ledger not found")
)
