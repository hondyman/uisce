package settlement

import "errors"

var (
	ErrInvalidSubtype = errors.New("invalid settlement subtype_code")
	ErrNotFound = errors.New("settlement not found")
)
