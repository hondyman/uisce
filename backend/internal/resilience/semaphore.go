package resilience

import (
	"sync"
)

// Semaphore implements a semaphore for limiting concurrent operations
type Semaphore struct {
	sem chan struct{}
	mu  sync.Mutex
}

// NewSemaphore creates a new semaphore with the given capacity
func NewSemaphore(capacity int64) *Semaphore {
	return &Semaphore{
		sem: make(chan struct{}, capacity),
	}
}

// Acquire acquires a permit from the semaphore, blocking if none available
func (s *Semaphore) Acquire() {
	s.sem <- struct{}{}
}

// TryAcquire attempts to acquire a permit without blocking
// Returns true if permit was acquired, false otherwise
func (s *Semaphore) TryAcquire() bool {
	select {
	case s.sem <- struct{}{}:
		return true
	default:
		return false
	}
}

// Release releases a permit back to the semaphore
func (s *Semaphore) Release() {
	select {
	case <-s.sem:
	default:
		panic("semaphore release without acquire")
	}
}

// CurrentPermits returns the number of available permits
func (s *Semaphore) CurrentPermits() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.sem)
}
