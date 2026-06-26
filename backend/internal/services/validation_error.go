package services

// FieldError represents a validation error on a specific field path.
type FieldError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

// ValidationError captures one or more field level issues coming from the service layer.
type ValidationError struct {
	Message string       `json:"message"`
	Errors  []FieldError `json:"errors"`
}

// Error implements the error interface.
func (e *ValidationError) Error() string {
	if e == nil {
		return ""
	}
	if e.Message != "" {
		return e.Message
	}
	return "validation failed"
}

// NewValidationError constructs a new ValidationError with a default message when none is provided.
func NewValidationError(message string, errors []FieldError) *ValidationError {
	if message == "" {
		message = "validation failed"
	}
	return &ValidationError{Message: message, Errors: errors}
}
