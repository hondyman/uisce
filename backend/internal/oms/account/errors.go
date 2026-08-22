package account

import "errors"

var (
	ErrInvalidSubtype = errors.New("invalid account subtype_code")
	ErrRequiresSponsorID = errors.New("institutional accounts require sponsor_id")
	ErrRequiresPlanType = errors.New("erisa-flagged accounts require plan_type")
	ErrNotFound = errors.New("account not found")
)
