package feed

import (
	"context"
	"sort"

	clientContext "github.com/hondyman/semlayer/backend/internal/context"
	"github.com/hondyman/semlayer/backend/internal/feed/rules"
)

type Curator struct {
	contextAggregator clientContext.ContextAggregator
	ruleEngine        rules.RuleEngine
}

func NewCurator(aggregator clientContext.ContextAggregator) (*Curator, error) {
	engine, err := rules.NewRuleEngine()
	if err != nil {
		return nil, err
	}
	return &Curator{
		contextAggregator: aggregator,
		ruleEngine:        *engine,
	}, nil
}

func (c *Curator) GenerateFeed(ctx context.Context, tenantID, clientID string) ([]FeedItem, error) {
	// 1. Get Client Context
	clientCtx, err := c.contextAggregator.GetContext(tenantID, clientID)
	if err != nil {
		return nil, err
	}

	// 2. Load Rules
	cardRules, err := c.ruleEngine.LoadRules()
	if err != nil {
		return nil, err
	}

	// 3. Hardcoded card templates (metadata will come from DB/YAML in future)
	templates := map[string]CardTemplate{
		"welcome_message": {
			ID:              "welcome_message",
			Type:            "insight",
			PriorityBase:    100,
			ContentTemplate: "Welcome to WealthStream. Your portfolio is being monitored.",
		},
		"tax_loss_harvest": {
			ID:              "tax_loss_harvest",
			Type:            "action",
			PriorityBase:    80,
			ContentTemplate: "You have harvestable tax losses. Review this opportunity to offset gains.",
			ActionWorkflow:  "wf_tax_loss_execution",
		},
		"portfolio_drift": {
			ID:              "portfolio_drift",
			Type:            "action",
			PriorityBase:    70,
			ContentTemplate: "Your portfolio has drifted from target allocation. Consider rebalancing.",
			ActionWorkflow:  "wf_rebalance_execution",
		},
	}

	var feed []FeedItem

	// 4. Evaluate each rule and create feed items for eligible cards
	for _, rule := range cardRules {
		result := c.ruleEngine.Evaluate(rule, clientCtx)
		if !result.Eligible {
			continue
		}

		tmpl, exists := templates[rule.CardID]
		if !exists {
			continue
		}

		item := FeedItem{
			CardID:           tmpl.ID,
			Title:            getCardTitle(tmpl.ID),
			Content:          tmpl.ContentTemplate,
			Type:             tmpl.Type,
			Score:            result.RankScore,
			ActionWorkflowID: tmpl.ActionWorkflow,
		}

		if tmpl.Type == "action" {
			item.ActionLabel = getActionLabel(tmpl.ID)
		}

		feed = append(feed, item)
	}

	// 5. Sort by Score
	sort.Slice(feed, func(i, j int) bool {
		return feed[i].Score > feed[j].Score
	})

	return feed, nil
}

func getCardTitle(cardID string) string {
	titles := map[string]string{
		"welcome_message":    "Welcome",
		"tax_loss_harvest":   "Tax Optimization",
		"portfolio_drift":    "Rebalancing Alert",
	}
	if title, ok := titles[cardID]; ok {
		return title
	}
	return "Insight"
}

func getActionLabel(cardID string) string {
	labels := map[string]string{
		"tax_loss_harvest": "Review Harvest",
		"portfolio_drift":  "Review Rebalance",
	}
	if label, ok := labels[cardID]; ok {
		return label
	}
	return "Take Action"
}

