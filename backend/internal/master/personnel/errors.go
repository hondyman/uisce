package personnel

import "errors"

var (
	ErrInvalidSubtype = errors.New("invalid personnel subtype_code")
	ErrNotFound       = errors.New("personnel not found")
)
