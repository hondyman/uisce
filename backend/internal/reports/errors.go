package reports

import "errors"

// ErrConflict is returned when a report name violates tenant uniqueness constraints.
var ErrConflict = errors.New("report conflict")

// ErrNotFound is returned when a requested report template or favorite target does not exist.
var ErrNotFound = errors.New("report not found")
