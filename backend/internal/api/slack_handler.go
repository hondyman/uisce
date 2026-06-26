package api

import (
	"database/sql"
	"encoding/json"
	"net/http"

	"github.com/hondyman/semlayer/backend/internal/notifications"
)

type SlackHandler struct {
	db          *sql.DB
	slackClient *notifications.SlackClient
}

func NewSlackHandler(db *sql.DB, sc *notifications.SlackClient) *SlackHandler {
	return &SlackHandler{db: db, slackClient: sc}
}

// GET /api/slack/install
func (h *SlackHandler) InstallSlack(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "https://slack.com/oauth/v2/authorize?client_id=FAKE&scope=chat:write", http.StatusTemporaryRedirect)
}

// POST /api/slack/callback
func (h *SlackHandler) SlackCallback(w http.ResponseWriter, r *http.Request) {
	// Exchange code for token...
	w.Write([]byte("Slack integration successful (Mock)"))
}

// POST /api/slack/interactive
func (h *SlackHandler) HandleSlackInteraction(w http.ResponseWriter, r *http.Request) {
	// Verify sig...
	// Parse payload...

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(200)
	json.NewEncoder(w).Encode(map[string]string{
		"response_type": "in_channel",
		"text":          "Action recorded (Mock)",
	})
}
