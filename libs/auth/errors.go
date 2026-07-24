package auth

import "errors"

var (
	ErrUnauthorized       = errors.New("unauthorized")
	ErrForbidden          = errors.New("forbidden")
	ErrTokenInvalid       = errors.New("invalid token")
	ErrTokenExpired       = errors.New("token has expired")
	ErrInvalidSignature   = errors.New("invalid signature")
	ErrMissingToken       = errors.New("missing token")
	ErrTenantMismatch     = errors.New("tenant mismatch")
	ErrInsufficientAccess = errors.New("insufficient access")
)
