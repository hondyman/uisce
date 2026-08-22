package trade_order

import "errors"

var (
	ErrInvalidSubtype = errors.New("invalid trade_order subtype_code")
	ErrNotFound       = errors.New("trade_order not found")
)
