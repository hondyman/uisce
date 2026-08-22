package reporting

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestCalendarEvaluator_NilCalendar(t *testing.T) {
	evaluator := NewCalendarEvaluator(nil)
	targetDate := time.Date(2026, 8, 24, 8, 0, 0, 0, time.UTC) // Monday

	allowed, effDate, err := evaluator.IsExecutionAllowedOnDate(
		context.Background(),
		uuid.New(),
		uuid.Nil,
		targetDate,
		"SKIP",
	)

	assert.NoError(t, err)
	assert.True(t, allowed)
	assert.Equal(t, targetDate, effDate)
}

func TestSanitizeIdentifier(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"client_id", "client_id"},
		{"client_id; DROP TABLE users;", "client_id DROP TABLE users"},
		{"account_id--comment", "account_idcomment"},
		{"'portfolio_id'", "portfolio_id"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := sanitizeIdentifier(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}
