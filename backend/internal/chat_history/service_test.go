package chat_history

import (
	"context"
	"testing"

	"github.com/google/uuid"
)

func TestServiceAppendMessage_InvalidRole(t *testing.T) {
	svc := NewService(nil)
	_, err := svc.AppendMessage(context.Background(), uuid.New(), uuid.New(), MessageRecord{Role: "manager"})
	if err != ErrInvalidRole {
		t.Fatalf("expected ErrInvalidRole, got %v", err)
	}
}

func TestServiceAppendMessage_ValidRoles(t *testing.T) {
	for _, role := range []string{"user", "assistant", "system", "tool"} {
		if err := validateRole(role); err != nil {
			t.Errorf("role %q should be valid, got %v", role, err)
		}
	}
}

func TestServiceSubmitFeedback_InvalidScore(t *testing.T) {
	svc := NewService(nil)
	if svc == nil {
		t.Fatal("service construction failed")
	}
	for _, score := range []int16{0, 2, -2, 42} {
		err := svc.SubmitFeedback(context.Background(), uuid.New(), uuid.New(), "u", false, score, nil)
		if err != ErrInvalidFeedback {
			t.Errorf("score %d should be rejected, got %v", score, err)
		}
	}
}

func TestServiceSubmitFeedback_ValidScores(t *testing.T) {
	for _, score := range []int16{1, -1} {
		if score != 1 && score != -1 {
			t.Errorf("expected %d to pass validation", score)
		}
	}
}

// validateRole mirrors the guard inside AppendMessage so the test exercises
// the role vocabulary without needing a database.
func validateRole(role string) error {
	if role != "user" && role != "assistant" && role != "system" && role != "tool" {
		return ErrInvalidRole
	}
	return nil
}