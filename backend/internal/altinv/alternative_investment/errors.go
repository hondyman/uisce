package alternative_investment

import "errors"

var (
	ErrInvalidSubtype = errors.New("invalid alternative investment subtype_code")
	ErrNotFound       = errors.New("alternative investment not found")
)