package vendor

import "errors"

var (
	ErrInvalidSubtype = errors.New("invalid vendor subtype_code")
	ErrNotFound = errors.New("vendor not found")
)
