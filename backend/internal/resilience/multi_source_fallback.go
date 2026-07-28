package resilience

import (
	"context"
	"errors"
	"time"
)

type QueryExecutorFunc func(ctx context.Context) (interface{}, error)

// ExecuteWithCircuitBreaker wraps primary high-performance analytical queries with tight timeouts and automatic fallback to semantic caches
func ExecuteWithCircuitBreaker(ctx context.Context, primaryExec QueryExecutorFunc, fallbackExec QueryExecutorFunc) (interface{}, error) {
	execCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	type execRes struct {
		data interface{}
		err  error
	}

	ch := make(chan execRes, 1)
	go func() {
		res, err := primaryExec(execCtx)
		ch <- execRes{data: res, err: err}
	}()

	select {
	case res := <-ch:
		if res.err == nil {
			return res.data, nil
		}
	case <-execCtx.Done():
		// Primary timeout reached
	}

	// Primary analytical storage failed or timed out; trigger graceful fallback
	if fallbackExec != nil {
		cachedResult, fallbackErr := fallbackExec(ctx)
		if fallbackErr == nil {
			return cachedResult, nil
		}
	}
	return nil, errors.New("primary storage failed and fallback unavailable")
}
