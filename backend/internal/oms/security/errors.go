package security

import "errors"

var (
	ErrInvalidSubtype = errors.New("invalid security subtype_code")
	ErrNotFound       = errors.New("security not found")
)
