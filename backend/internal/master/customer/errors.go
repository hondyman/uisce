package customer

import "errors"

var (
	ErrInvalidSubtype = errors.New("invalid customer subtype_code")
	ErrNotFound = errors.New("customer not found")
)
