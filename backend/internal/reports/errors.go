package reports

import "errors"

// ErrConflict is returned when a report name violates tenant uniqueness constraints.
var ErrConflict = errors.New("report conflict")

// ErrNotFound is returned when a requested report template or favorite target does not exist.
var ErrNotFound = errors.New("report not found")

// ErrCycleDetected is returned when a folder move would introduce a circular hierarchy.
var ErrCycleDetected = errors.New("folder cycle detected")

// ErrDepthLimitExceeded is returned when a folder hierarchy exceeds the maximum depth limit.
var ErrDepthLimitExceeded = errors.New("folder depth limit exceeded")

