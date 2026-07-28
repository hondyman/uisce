package resilience

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestExecuteWithCircuitBreaker_PrimarySuccess(t *testing.T) {
	primary := func(ctx context.Context) (interface{}, error) {
		return "primary_data", nil
	}
	fallback := func(ctx context.Context) (interface{}, error) {
		return "fallback_data", nil
	}

	res, err := ExecuteWithCircuitBreaker(context.Background(), primary, fallback)
	assert.NoError(t, err)
	assert.Equal(t, "primary_data", res)
}

func TestExecuteWithCircuitBreaker_FallbackTriggered(t *testing.T) {
	primary := func(ctx context.Context) (interface{}, error) {
		return nil, errors.New("StarRocks cluster timeout")
	}
	fallback := func(ctx context.Context) (interface{}, error) {
		return "cached_redis_data", nil
	}

	res, err := ExecuteWithCircuitBreaker(context.Background(), primary, fallback)
	assert.NoError(t, err)
	assert.Equal(t, "cached_redis_data", res)
}

func TestExecuteWithCircuitBreaker_TimeoutTriggered(t *testing.T) {
	primary := func(ctx context.Context) (interface{}, error) {
		time.Sleep(3 * time.Second)
		return "slow_data", nil
	}
	fallback := func(ctx context.Context) (interface{}, error) {
		return "fallback_cached_data", nil
	}

	res, err := ExecuteWithCircuitBreaker(context.Background(), primary, fallback)
	assert.NoError(t, err)
	assert.Equal(t, "fallback_cached_data", res)
}
