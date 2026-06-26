package dashboard

import (
	"time"

	"github.com/google/uuid"
)

// WidgetType represents different dashboard widgets
type WidgetType string

const (
	WidgetPortfolioSummary   WidgetType = "PORTFOLIO_SUMMARY"
	WidgetGoalsProgress      WidgetType = "GOALS_PROGRESS"
	WidgetRecentTransactions WidgetType = "RECENT_TRANSACTIONS"
	WidgetMessagesInbox      WidgetType = "MESSAGES_INBOX"
	WidgetBillingStatus      WidgetType = "BILLING_STATUS"
	WidgetUpcomingMeetings   WidgetType = "UPCOMING_MEETINGS"
	WidgetMarketNews         WidgetType = "MARKET_NEWS"
	WidgetRecommendedActions WidgetType = "RECOMMENDED_ACTIONS"
	WidgetNetWorthTrend      WidgetType = "NET_WORTH_TREND"
	WidgetAssetAllocation    WidgetType = "ASSET_ALLOCATION"
)

// WidgetSize represents widget dimensions
type WidgetSize string

const (
	SizeSmall     WidgetSize = "SMALL"
	SizeMedium    WidgetSize = "MEDIUM"
	SizeLarge     WidgetSize = "LARGE"
	SizeFullWidth WidgetSize = "FULL_WIDTH"
)

// DashboardWidget represents a customizable dashboard component
type DashboardWidget struct {
	WidgetID   uuid.UUID  `json:"widget_id" db:"widget_id"`
	ClientID   uuid.UUID  `json:"client_id" db:"client_id"`
	WidgetType WidgetType `json:"widget_type" db:"widget_type"`
	Position   int        `json:"position" db:"position"`
	Size       WidgetSize `json:"size" db:"size"`
	Config     []byte     `json:"config" db:"config"` // JSONB
	IsVisible  bool       `json:"is_visible" db:"is_visible"`
	CreatedAt  time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at" db:"updated_at"`
}

// GoalType represents different financial goals
type GoalType string

const (
	GoalRetirement    GoalType = "RETIREMENT"
	GoalEducation     GoalType = "EDUCATION"
	GoalHomePurchase  GoalType = "HOME_PURCHASE"
	GoalDebtPayoff    GoalType = "DEBT_PAYOFF"
	GoalLegacy        GoalType = "LEGACY"
	GoalMajorPurchase GoalType = "MAJOR_PURCHASE"
	GoalEmergencyFund GoalType = "EMERGENCY_FUND"
	GoalCustom        GoalType = "CUSTOM"
)

// GoalStatus represents the state of a goal
type GoalStatus string

const (
	GoalActive    GoalStatus = "ACTIVE"
	GoalCompleted GoalStatus = "COMPLETED"
	GoalPaused    GoalStatus = "PAUSED"
	GoalAbandoned GoalStatus = "ABANDONED"
)

// ClientGoal represents a financial goal with tracking
type ClientGoal struct {
	GoalID                  uuid.UUID  `json:"goal_id" db:"goal_id"`
	ClientID                uuid.UUID  `json:"client_id" db:"client_id"`
	GoalType                GoalType   `json:"goal_type" db:"goal_type"`
	GoalName                string     `json:"goal_name" db:"goal_name"`
	Description             *string    `json:"description" db:"description"`
	TargetAmount            float64    `json:"target_amount" db:"target_amount"`
	TargetDate              time.Time  `json:"target_date" db:"target_date"`
	CurrentProgress         float64    `json:"current_progress" db:"current_progress"`
	MonthlyContribution     *float64   `json:"monthly_contribution" db:"monthly_contribution"`
	AssumedReturnRate       *float64   `json:"assumed_return_rate" db:"assumed_return_rate"`
	ProjectedCompletionDate *time.Time `json:"projected_completion_date" db:"projected_completion_date"`
	ConfidenceLevel         *float64   `json:"confidence_level" db:"confidence_level"`
	Status                  GoalStatus `json:"status" db:"status"`
	CompletedAt             *time.Time `json:"completed_at" db:"completed_at"`
	Notes                   *string    `json:"notes" db:"notes"`
	CreatedAt               time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt               time.Time  `json:"updated_at" db:"updated_at"`
}

// PortfolioSummary represents aggregated portfolio data
type PortfolioSummary struct {
	TotalValue       float64            `json:"total_value"`
	DayChange        float64            `json:"day_change"`
	DayChangePercent float64            `json:"day_change_percent"`
	YTDReturn        float64            `json:"ytd_return"`
	Allocation       map[string]float64 `json:"allocation"` // Asset class percentages
}

// GoalProgress represents calculated goal metrics
type GoalProgress struct {
	GoalID             uuid.UUID `json:"goal_id"`
	GoalName           string    `json:"goal_name"`
	TargetAmount       float64   `json:"target_amount"`
	CurrentAmount      float64   `json:"current_amount"`
	ProgressPercentage float64   `json:"progress_percentage"`
	MonthsRemaining    int       `json:"months_remaining"`
	OnTrack            bool      `json:"on_track"`
	MonthlyNeedSavings float64   `json:"monthly_need_savings"`
}

// DashboardSummary represents the full dashboard data
type DashboardSummary struct {
	UnreadMessages      int `json:"unread_messages"`
	UnreadNotifications int `json:"unread_notifications"`
	PendingActions      int `json:"pending_actions"`
	ActiveGoals         int `json:"active_goals"`
}
