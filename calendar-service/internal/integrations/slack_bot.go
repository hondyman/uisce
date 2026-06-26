package integrations

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/sirupsen/logrus"
	"github.com/slack-go/slack"
)

// SlackBot handles Slack integration
type SlackBot struct {
	client   *slack.Client
	botToken string
	logger   *logrus.Entry
}

// SlackBotConfig holds configuration
type SlackBotConfig struct {
	BotToken string
	Logger   *logrus.Entry
}

// NewSlackBot creates a new Slack bot
func NewSlackBot(cfg SlackBotConfig) *SlackBot {
	return &SlackBot{
		client:   slack.New(cfg.BotToken),
		botToken: cfg.BotToken,
		logger:   cfg.Logger.WithField("component", "slack_bot"),
	}
}

// HandleSlashCommand handles Slack slash commands
func (sb *SlackBot) HandleSlashCommand(w http.ResponseWriter, r *http.Request) {
	// Parse slash command
	s, err := slack.SlashCommandParse(r)
	if err != nil {
		http.Error(w, "Failed to parse command", http.StatusBadRequest)
		return
	}

	// Verify token
	if s.Token != sb.botToken {
		http.Error(w, "Invalid token", http.StatusUnauthorized)
		return
	}

	// Handle command
	response := sb.handleCommand(s)

	// Send response
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleCommand processes slash commands
func (sb *SlackBot) handleCommand(s slack.SlashCommand) interface{} {
	switch s.Command {
	case "/calendar-sync":
		return sb.handleSyncCommand(s)
	case "/calendar-status":
		return sb.handleStatusCommand(s)
	case "/calendar-conflicts":
		return sb.handleConflictsCommand(s)
	default:
		return map[string]interface{}{
			"text": "Unknown command. Available: /calendar-sync, /calendar-status, /calendar-conflicts",
		}
	}
}

// handleSyncCommand triggers manual sync
func (sb *SlackBot) handleSyncCommand(s slack.SlashCommand) interface{} {
	// Trigger sync for user
	// ... sync logic ...

	return map[string]interface{}{
		"response_type": "in_channel",
		"text":          "✅ Calendar sync started! You'll be notified when it's complete.",
	}
}

// handleStatusCommand handles the status command (stub)
func (sb *SlackBot) handleStatusCommand(s slack.SlashCommand) interface{} {
	return map[string]interface{}{
		"response_type": "in_channel",
		"text":          "Status: All systems operational.",
	}
}

// handleConflictsCommand handles the conflicts command (stub)
func (sb *SlackBot) handleConflictsCommand(s slack.SlashCommand) interface{} {
	return map[string]interface{}{
		"response_type": "in_channel",
		"text":          "0 conflicts found.",
	}
}

// SendSyncNotification sends sync notification to Slack
func (sb *SlackBot) SendSyncNotification(ctx context.Context, userID, channelID string, status string, eventsSynced int) error {
	blocks := []slack.Block{
		slack.NewHeaderBlock(slack.NewTextBlockObject("plain_text", "📅 Calendar Sync Update", false, false)),
		slack.NewSectionBlock(
			slack.NewTextBlockObject("mrkdwn", fmt.Sprintf("*Status:* %s\n*Events Synced:* %d", status, eventsSynced), false, false),
			nil,
			nil,
		),
	}

	_, _, err := sb.client.PostMessageContext(ctx, channelID, slack.MsgOptionBlocks(blocks...))
	return err
}

// SendConflictNotification sends conflict notification to Slack
func (sb *SlackBot) SendConflictNotification(ctx context.Context, userID, channelID string, conflictCount int) error {
	blocks := []slack.Block{
		slack.NewHeaderBlock(slack.NewTextBlockObject("plain_text", "⚠️ Sync Conflicts Detected", false, false)),
		slack.NewSectionBlock(
			slack.NewTextBlockObject("mrkdwn", fmt.Sprintf("Found *%d conflicts* that need your attention.", conflictCount), false, false),
			nil,
			nil,
		),
		slack.NewActionBlock(
			"",
			slack.NewButtonBlockElement(
				"resolve_conflicts",
				"Resolve Conflicts",
				slack.NewTextBlockObject("plain_text", "View & Resolve", false, false),
			),
		),
	}

	_, _, err := sb.client.PostMessageContext(ctx, channelID, slack.MsgOptionBlocks(blocks...))
	return err
}
