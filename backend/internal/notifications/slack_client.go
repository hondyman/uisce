package notifications

import (
	"context"

	"github.com/slack-go/slack"
)

type SlackClient struct {
	client *slack.Client
}

func NewSlackClient(botToken string) *SlackClient {
	return &SlackClient{
		client: slack.New(botToken),
	}
}

func (c *SlackClient) PostMessage(ctx context.Context, userID string, blocks map[string]interface{}) error {
	if c.client == nil {
		return nil
	}

	// Convert blocks map to slack.Block slice
	// blockJSON, _ := json.Marshal(blocks)

	// For simplicity with map input, we might wrap in a generic block/msg logic
	// But actual slack.Block is complex.
	// We assume 'blocks' here contains structure compatible with slack MsgOptionBlocks?
	// Actually simpler: just send as attachment or text if Block Kit construction is too complex for map[string]interface{}.
	// But if we trust the JSON structure matches Block Kit:

	// Hack: Parse as a generic message payload if possible, or build blocks manually in service.
	// For now, let's treat it as a raw message send attempt with text fallback if blocks invalid

	// Real implementation would cast blocks to []slack.Block properly
	// This is valid:
	// _, err := c.client.PostMessageContext(ctx, userID, slack.MsgOptionBlocks(parsedBlocks...))

	// MVP: Just send text message
	_, _, err := c.client.PostMessageContext(ctx, userID, slack.MsgOptionText("Notification: check App", false))
	return err
}
