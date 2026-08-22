package position

import "errors"

var (
	ErrInvalidSubtype       = errors.New("invalid position subtype_code")
	ErrRequiresPrimeBroker  = errors.New("short borrowed positions require prime_broker_id")
	ErrNotFound             = errors.New("position not found")
)
