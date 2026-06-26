package wealth

import (
	"context"
	"database/sql"

	"fmt"
	"sort"
	"time"

	"github.com/hondyman/semlayer/backend/internal/logging"
	"github.com/hondyman/semlayer/backend/internal/metadata"
)

// CurationService is responsible for generating personalized feeds for clients
type CurationService struct {
	db        *sql.DB
	boService *metadata.BusinessObjectService
}

// NewCurationService creates a new CurationService
func NewCurationService(db *sql.DB, boService *metadata.BusinessObjectService) *CurationService {
	return &CurationService{
		db:        db,
		boService: boService,
	}
}

// FeedItem represents a curated item in the client's feed
type FeedItem struct {
	ID            string                 `json:"id"`
	TemplateID    string                 `json:"template_id"`
	Title         string                 `json:"title"`
	Body          string                 `json:"body"`
	ImageURL      string                 `json:"image_url,omitempty"`
	ActionLabel   string                 `json:"action_label,omitempty"`
	ActionURL     string                 `json:"action_url,omitempty"`
	PriorityScore float64                `json:"priority_score"`
	GeneratedAt   time.Time              `json:"generated_at"`
	Context       map[string]interface{} `json:"context,omitempty"`
}

// GenerateFeed generates a ranked list of feed items for a specific client
func (s *CurationService) GenerateFeed(ctx context.Context, tenantID, clientID string) ([]FeedItem, error) {
	// 1. Fetch Client Profile (Extended)
	// In a real implementation, we would use the BusinessObjectService to fetch the instance
	// For now, we'll simulate fetching the profile
	profile, err := s.fetchClientProfile(ctx, tenantID, clientID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch client profile: %v", err)
	}

	// 2. Fetch Active Feed Card Templates from Metadata
	// We would query meta_objects where type = 'bo_feed_card'
	// For now, we'll use a placeholder list of templates
	templates := s.getFeedTemplates()

	// 3. Evaluate Eligibility & Rank Cards
	var feed []FeedItem
	for _, tmpl := range templates {
		if s.isEligible(profile, tmpl) {
			score := s.calculatePriority(profile, tmpl)
			item := FeedItem{
				ID:            fmt.Sprintf("feed_%s_%s_%d", clientID, tmpl.ID, time.Now().Unix()),
				TemplateID:    tmpl.ID,
				Title:         tmpl.Title,
				Body:          tmpl.Body, // In reality, we'd hydrate templates with client data
				PriorityScore: score,
				GeneratedAt:   time.Now(),
				Context:       map[string]interface{}{"risk_score": profile.RiskTolerance},
			}
			feed = append(feed, item)
		}
	}

	// 4. Sort by Priority
	sort.Slice(feed, func(i, j int) bool {
		return feed[i].PriorityScore > feed[j].PriorityScore
	})

	// 5. Log Curation Event (UAR)
	logging.GetLogger().Sugar().Infof("Generated feed for client %s with %d items", clientID, len(feed))

	return feed, nil
}

// Helper structs and methods

type CurationClientProfile struct {
	ClientID      string
	RiskTolerance int
	ESGFocus      []string
	BehaviorTags  []string
}

type FeedTemplate struct {
	ID       string
	Title    string
	Body     string
	MinRisk  int
	MaxRisk  int
	Tags     []string
	BasePrio float64
}

func (s *CurationService) fetchClientProfile(ctx context.Context, tenantID, clientID string) (*CurationClientProfile, error) {
	// Placeholder: Fetch from DB or BO Service
	return &CurationClientProfile{
		ClientID:      clientID,
		RiskTolerance: 60, // Moderate
		ESGFocus:      []string{"Climate"},
		BehaviorTags:  []string{"active_trader"},
	}, nil
}

func (s *CurationService) getFeedTemplates() []FeedTemplate {
	// Placeholder: These would come from meta_objects
	return []FeedTemplate{
		{
			ID:       "tmpl_market_update",
			Title:    "Market Update",
			Body:     "Markets are up today driven by tech sector.",
			MinRisk:  0,
			MaxRisk:  100,
			BasePrio: 1.0,
		},
		{
			ID:       "tmpl_risk_alert",
			Title:    "Portfolio Risk Alert",
			Body:     "Your portfolio risk is higher than your target.",
			MinRisk:  0,
			MaxRisk:  50, // Only for conservative/moderate who drifted high
			BasePrio: 2.0,
		},
		{
			ID:       "tmpl_esg_impact",
			Title:    "ESG Impact Report",
			Body:     "See how your portfolio aligns with your climate goals.",
			MinRisk:  0,
			MaxRisk:  100,
			Tags:     []string{"Climate"},
			BasePrio: 1.5,
		},
	}
}

func (s *CurationService) isEligible(profile *CurationClientProfile, tmpl FeedTemplate) bool {
	// Check Risk
	// Note: Logic is simplified. Real logic would check if profile.RiskTolerance is within range
	// or if the card is relevant to their specific situation.

	// Check Tags
	if len(tmpl.Tags) > 0 {
		matched := false
		for _, tag := range tmpl.Tags {
			for _, pTag := range profile.ESGFocus {
				if tag == pTag {
					matched = true
					break
				}
			}
		}
		if !matched {
			return false
		}
	}
	return true
}

func (s *CurationService) calculatePriority(profile *CurationClientProfile, tmpl FeedTemplate) float64 {
	score := tmpl.BasePrio
	// Boost score if behavior tags match
	for _, tag := range profile.BehaviorTags {
		if tag == "active_trader" && tmpl.ID == "tmpl_market_update" {
			score += 0.5
		}
	}
	return score
}
