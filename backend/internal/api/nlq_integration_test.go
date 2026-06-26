package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/hondyman/semlayer/backend/internal/services"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"
)

func TestNLQFeedback(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	sqlxDB := sqlx.NewDb(db, "sqlmock")
	feedbackSvc := services.NewFeedbackService(sqlxDB)

	server := &Server{
		FeedbackService: feedbackSvc,
	}

	t.Run("Submit Feedback Success", func(t *testing.T) {
		mock.ExpectExec("INSERT INTO nlq_feedback").
			WithArgs(sqlmock.AnyArg(), "tenant-1", "user-1", 5, "Great!", sqlmock.AnyArg()).
			WillReturnResult(sqlmock.NewResult(1, 1))

		reqBody := map[string]interface{}{
			"query_id":  "00000000-0000-0000-0000-000000000001",
			"tenant_id": "tenant-1",
			"user_id":   "user-1",
			"rating":    5,
			"comment":   "Great!",
		}
		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("POST", "/nlq/feedback", bytes.NewBuffer(body))
		w := httptest.NewRecorder()

		server.handleNLQFeedback(w, req)

		require.Equal(t, http.StatusOK, w.Code)
	})
}


