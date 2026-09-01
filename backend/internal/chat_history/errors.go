package chat_history

import "errors"

var (
	ErrSessionNotFound    = errors.New("chat session not found")
	ErrMessageNotFound    = errors.New("chat message not found")
	ErrInvalidRole        = errors.New("invalid message role; must be user, assistant, system, or tool")
	ErrInvalidViewType    = errors.New("invalid view type; must be end_user or admin")
	ErrUnauthorizedTenant = errors.New("tenant mismatch or cross-tenant access denied")
	ErrForbidden          = errors.New("caller lacks authorization to perform this operation")
	ErrInvalidFeedback    = errors.New("feedback score must be 1 (up) or -1 (down)")
)