package dashboard

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// Service provides dashboard operations
type Service interface {
	// Widget management
	GetWidgets(ctx context.Context, clientID uuid.UUID) ([]*DashboardWidget, error)
	UpdateWidgetLayout(ctx context.Context, clientID uuid.UUID, widgets []WidgetUpdate) error
	CreateWidget(ctx context.Context, clientID uuid.UUID, widgetType WidgetType, position int, size WidgetSize) (*DashboardWidget, error)
	DeleteWidget(ctx context.Context, widgetID uuid.UUID) error

	// Goals
	CreateGoal(ctx context.Context, input CreateGoalInput) (*ClientGoal, error)
	GetGoal(ctx context.Context, goalID uuid.UUID) (*ClientGoal, error)
	UpdateGoalProgress(ctx context.Context, goalID uuid.UUID, currentProgress float64) error
	ListClientGoals(ctx context.Context, clientID uuid.UUID) ([]*ClientGoal, error)
	CalculateGoalProjection(ctx context.Context, goalID uuid.UUID) (*GoalProgress, error)

	// Dashboard data
	GetDashboardSummary(ctx context.Context, clientID uuid.UUID) (*DashboardSummary, error)
	GetPortfolioSummary(ctx context.Context, clientID uuid.UUID) (*PortfolioSummary, error)
}

// HasuraClient defines the interface for Hasura GraphQL operations
type HasuraClient interface {
	Query(query string, variables map[string]interface{}) (map[string]interface{}, error)
	Mutate(mutation string, variables map[string]interface{}) (map[string]interface{}, error)
}

type service struct {
	db     *sqlx.DB
	hasura HasuraClient
}

func NewService(db *sqlx.DB) Service {
	return &service{db: db}
}

func NewServiceWithHasura(db *sqlx.DB, hasura HasuraClient) Service {
	return &service{db: db, hasura: hasura}
}

type WidgetUpdate struct {
	WidgetID  uuid.UUID  `json:"widget_id"`
	Position  int        `json:"position"`
	Size      WidgetSize `json:"size"`
	IsVisible bool       `json:"is_visible"`
}

type CreateGoalInput struct {
	ClientID            uuid.UUID
	GoalType            GoalType
	GoalName            string
	Description         string
	TargetAmount        float64
	TargetDate          time.Time
	MonthlyContribution float64
	AssumedReturnRate   float64
}

// GetWidgets retrieves all dashboard widgets for a client
// TODO: Implement Hasura GraphQL query
// SQL fallback: SelectContext with ORDER BY position
func (s *service) GetWidgets(ctx context.Context, clientID uuid.UUID) ([]*DashboardWidget, error) {
	var widgets []*DashboardWidget
	query := `
		SELECT * FROM dashboard_widgets
		WHERE client_id = $1
		ORDER BY position
	`
	err := s.db.SelectContext(ctx, &widgets, query, clientID)
	return widgets, err
}

func (s *service) UpdateWidgetLayout(ctx context.Context, clientID uuid.UUID, widgets []WidgetUpdate) error {
	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	for _, w := range widgets {
		if err := s.updateWidgetLayoutSingle(ctx, tx, w, clientID); err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (s *service) CreateWidget(ctx context.Context, clientID uuid.UUID, widgetType WidgetType, position int, size WidgetSize) (*DashboardWidget, error) {
	widget := &DashboardWidget{
		WidgetID:   uuid.New(),
		ClientID:   clientID,
		WidgetType: widgetType,
		Position:   position,
		Size:       size,
		Config:     []byte("{}"),
		IsVisible:  true,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	if err := s.createWidgetRecord(ctx, widget); err != nil {
		return nil, err
	}

	return widget, nil
}

func (s *service) DeleteWidget(ctx context.Context, widgetID uuid.UUID) error {
	return s.deleteWidgetRecord(ctx, widgetID)
}

func (s *service) CreateGoal(ctx context.Context, input CreateGoalInput) (*ClientGoal, error) {
	goal := &ClientGoal{
		GoalID:              uuid.New(),
		ClientID:            input.ClientID,
		GoalType:            input.GoalType,
		GoalName:            input.GoalName,
		Description:         &input.Description,
		TargetAmount:        input.TargetAmount,
		TargetDate:          input.TargetDate,
		CurrentProgress:     0,
		MonthlyContribution: &input.MonthlyContribution,
		AssumedReturnRate:   &input.AssumedReturnRate,
		Status:              GoalActive,
		CreatedAt:           time.Now(),
		UpdatedAt:           time.Now(),
	}

	// Calculate projection
	projection := s.calculateProjection(input.TargetAmount, 0, input.MonthlyContribution, input.AssumedReturnRate, input.TargetDate)
	goal.ProjectedCompletionDate = &projection.CompletionDate
	goal.ConfidenceLevel = &projection.Confidence

	if err := s.createGoalRecord(ctx, goal); err != nil {
		return nil, err
	}

	return goal, nil
}

// GetGoal retrieves a single client goal by ID
// TODO: Implement Hasura GraphQL query
// SQL fallback: GetContext SELECT * by goal_id
func (s *service) GetGoal(ctx context.Context, goalID uuid.UUID) (*ClientGoal, error) {
	var goal ClientGoal
	query := `SELECT * FROM client_goals WHERE goal_id = $1`
	err := s.db.GetContext(ctx, &goal, query, goalID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("goal not found: %s", goalID)
		}
		return nil, err
	}
	return &goal, nil
}

func (s *service) UpdateGoalProgress(ctx context.Context, goalID uuid.UUID, currentProgress float64) error {
	return s.updateGoalProgressRecord(ctx, goalID, currentProgress)
}

// ListClientGoals retrieves all active goals for a client
// TODO: Implement Hasura GraphQL query
// SQL fallback: SelectContext with status filter and ORDER BY target_date
func (s *service) ListClientGoals(ctx context.Context, clientID uuid.UUID) ([]*ClientGoal, error) {
	var goals []*ClientGoal
	query := `
		SELECT * FROM client_goals
		WHERE client_id = $1 AND status = $2
		ORDER BY target_date
	`
	err := s.db.SelectContext(ctx, &goals, query, clientID, GoalActive)
	return goals, err
}

func (s *service) CalculateGoalProjection(ctx context.Context, goalID uuid.UUID) (*GoalProgress, error) {
	goal, err := s.GetGoal(ctx, goalID)
	if err != nil {
		return nil, err
	}

	monthsRemaining := int(time.Until(goal.TargetDate).Hours() / 24 / 30)
	progressPct := (goal.CurrentProgress / goal.TargetAmount) * 100

	// Calculate monthly savings needed
	remaining := goal.TargetAmount - goal.CurrentProgress
	monthlyNeed := remaining / float64(monthsRemaining)

	onTrack := false
	if goal.MonthlyContribution != nil && *goal.MonthlyContribution >= monthlyNeed {
		onTrack = true
	}

	return &GoalProgress{
		GoalID:             goal.GoalID,
		GoalName:           goal.GoalName,
		TargetAmount:       goal.TargetAmount,
		CurrentAmount:      goal.CurrentProgress,
		ProgressPercentage: progressPct,
		MonthsRemaining:    monthsRemaining,
		OnTrack:            onTrack,
		MonthlyNeedSavings: monthlyNeed,
	}, nil
}

// GetDashboardSummary retrieves aggregated dashboard metrics
// TODO: Implement Hasura GraphQL query
// SQL fallback: GetContext from client_dashboard_summary view/table
func (s *service) GetDashboardSummary(ctx context.Context, clientID uuid.UUID) (*DashboardSummary, error) {
	var summary DashboardSummary
	query := `SELECT * FROM client_dashboard_summary WHERE client_id = $1`
	err := s.db.GetContext(ctx, &summary, query, clientID)
	return &summary, err
}

// GetPortfolioSummary calculates portfolio aggregations
// TODO: Implement Hasura GraphQL query with aggregations
// SQL fallback: SUM aggregation from portfolio_holdings
func (s *service) GetPortfolioSummary(ctx context.Context, clientID uuid.UUID) (*PortfolioSummary, error) {
	// Query portfolio holdings and calculate summary
	var totalValue float64
	query := `SELECT COALESCE(SUM(market_value), 0) FROM portfolio_holdings WHERE client_id = $1`
	err := s.db.GetContext(ctx, &totalValue, query, clientID)
	if err != nil {
		return nil, err
	}

	// Simplified - would calculate actual day change, YTD, etc.
	summary := &PortfolioSummary{
		TotalValue:       totalValue,
		DayChange:        totalValue * 0.005, // Example: 0.5% gain
		DayChangePercent: 0.5,
		YTDReturn:        12.5,
		Allocation: map[string]float64{
			"Stocks": 60.0,
			"Bonds":  30.0,
			"Cash":   10.0,
		},
	}

	return summary, nil
}

// Helper function to calculate goal projection
func (s *service) calculateProjection(targetAmount, currentAmount, monthlyContribution, returnRate float64, targetDate time.Time) struct {
	CompletionDate time.Time
	Confidence     float64
} {
	// Simplified future value calculation
	months := time.Until(targetDate).Hours() / 24 / 30
	monthlyRate := returnRate / 12

	// Future value of current amount
	fvCurrent := currentAmount * (1 + monthlyRate) * months

	// Future value of monthly contributions (annuity)
	fvContributions := monthlyContribution * (((1+monthlyRate)*months - 1) / monthlyRate)

	projectedTotal := fvCurrent + fvContributions

	// Calculate confidence based on how close we'll get
	confidence := projectedTotal / targetAmount
	if confidence > 1.0 {
		confidence = 0.95 // Cap at 95% confidence
	}

	// Estimate completion date
	completionDate := targetDate
	if projectedTotal < targetAmount {
		// Will complete after target date
		additionalMonths := (targetAmount - projectedTotal) / monthlyContribution
		completionDate = targetDate.AddDate(0, int(additionalMonths), 0)
		confidence = 0.7 // Lower confidence if behind schedule
	}

	return struct {
		CompletionDate time.Time
		Confidence     float64
	}{
		CompletionDate: completionDate,
		Confidence:     confidence,
	}
}

// ============================================================================
// HASURA-FIRST HELPERS
// ============================================================================

// updateWidgetLayoutSingle updates a single widget's layout
// Hasura-first with SQL fallback
func (s *service) updateWidgetLayoutSingle(ctx context.Context, tx *sqlx.Tx, w WidgetUpdate, clientID uuid.UUID) error {
	if s.hasura != nil && tx == nil {
		mutation := `
			mutation UpdateWidgetLayout($widgetID: uuid!, $position: Int!, $size: String!, $isVisible: Boolean!) {
				update_dashboard_widgets_by_pk(
					pk_columns: {widget_id: $widgetID}
					_set: {
						position: $position
						size: $size
						is_visible: $isVisible
						updated_at: "now()"
					}
				) {
					widget_id
				}
			}
		`

		variables := map[string]interface{}{
			"widgetID":  w.WidgetID,
			"position":  w.Position,
			"size":      w.Size,
			"isVisible": w.IsVisible,
		}

		_, err := s.hasura.Mutate(mutation, variables)
		if err == nil {
			return nil
		}
		// Fall through to SQL on Hasura error
	}

	// SQL fallback (use transaction if provided)
	query := `
		UPDATE dashboard_widgets
		SET position = $1, size = $2, is_visible = $3, updated_at = NOW()
		WHERE widget_id = $4 AND client_id = $5
	`
	var err error
	if tx != nil {
		_, err = tx.ExecContext(ctx, query, w.Position, w.Size, w.IsVisible, w.WidgetID, clientID)
	} else {
		_, err = s.db.ExecContext(ctx, query, w.Position, w.Size, w.IsVisible, w.WidgetID, clientID)
	}
	return err
}

// createWidgetRecord inserts a new dashboard widget
// TODO: Already has Hasura GraphQL mutation implemented
// SQL fallback: NamedExec INSERT for 9 widget fields with JSONB config
func (s *service) createWidgetRecord(ctx context.Context, widget *DashboardWidget) error {
	if s.hasura != nil {
		mutation := `
			mutation CreateWidget(
				$widgetID: uuid!
				$clientID: uuid!
				$widgetType: String!
				$position: Int!
				$size: String!
				$config: jsonb!
				$isVisible: Boolean!
				$createdAt: timestamptz!
				$updatedAt: timestamptz!
			) {
				insert_dashboard_widgets_one(object: {
					widget_id: $widgetID
					client_id: $clientID
					widget_type: $widgetType
					position: $position
					size: $size
					config: $config
					is_visible: $isVisible
					created_at: $createdAt
					updated_at: $updatedAt
				}) {
					widget_id
				}
			}
		`

		var configJSON interface{}
		if err := json.Unmarshal(widget.Config, &configJSON); err != nil {
			configJSON = "{}"
		}

		variables := map[string]interface{}{
			"widgetID":   widget.WidgetID,
			"clientID":   widget.ClientID,
			"widgetType": widget.WidgetType,
			"position":   widget.Position,
			"size":       widget.Size,
			"config":     configJSON,
			"isVisible":  widget.IsVisible,
			"createdAt":  widget.CreatedAt,
			"updatedAt":  widget.UpdatedAt,
		}

		_, err := s.hasura.Mutate(mutation, variables)
		if err == nil {
			return nil
		}
		// Fall through to SQL on Hasura error
	}

	// SQL fallback
	query := `
		INSERT INTO dashboard_widgets (
			widget_id, client_id, widget_type, position, size, config, is_visible, created_at, updated_at
		) VALUES (
			:widget_id, :client_id, :widget_type, :position, :size, :config, :is_visible, :created_at, :updated_at
		)
	`
	_, err := s.db.NamedExecContext(ctx, query, widget)
	return err
}

// deleteWidgetRecord deletes a dashboard widget
// TODO: Already has Hasura GraphQL mutation implemented
// SQL fallback: DELETE by widget_id
func (s *service) deleteWidgetRecord(ctx context.Context, widgetID uuid.UUID) error {
	if s.hasura != nil {
		mutation := `
			mutation DeleteWidget($widgetID: uuid!) {
				delete_dashboard_widgets_by_pk(widget_id: $widgetID) {
					widget_id
				}
			}
		`

		variables := map[string]interface{}{
			"widgetID": widgetID,
		}

		_, err := s.hasura.Mutate(mutation, variables)
		if err == nil {
			return nil
		}
		// Fall through to SQL on Hasura error
	}

	// SQL fallback
	query := `DELETE FROM dashboard_widgets WHERE widget_id = $1`
	_, err := s.db.ExecContext(ctx, query, widgetID)
	return err
}

// createGoalRecord inserts a new client goal
// TODO: Already has Hasura GraphQL mutation implemented
// SQL fallback: NamedExec INSERT for 15 goal fields with projections
func (s *service) createGoalRecord(ctx context.Context, goal *ClientGoal) error {
	if s.hasura != nil {
		mutation := `
			mutation CreateGoal(
				$goalID: uuid!
				$clientID: uuid!
				$goalType: String!
				$goalName: String!
				$description: String
				$targetAmount: numeric!
				$targetDate: timestamptz!
				$currentProgress: numeric!
				$monthlyContribution: numeric
				$assumedReturnRate: numeric
				$projectedCompletionDate: timestamptz
				$confidenceLevel: numeric
				$status: String!
				$createdAt: timestamptz!
				$updatedAt: timestamptz!
			) {
				insert_client_goals_one(object: {
					goal_id: $goalID
					client_id: $clientID
					goal_type: $goalType
					goal_name: $goalName
					description: $description
					target_amount: $targetAmount
					target_date: $targetDate
					current_progress: $currentProgress
					monthly_contribution: $monthlyContribution
					assumed_return_rate: $assumedReturnRate
					projected_completion_date: $projectedCompletionDate
					confidence_level: $confidenceLevel
					status: $status
					created_at: $createdAt
					updated_at: $updatedAt
				}) {
					goal_id
				}
			}
		`

		variables := map[string]interface{}{
			"goalID":                  goal.GoalID,
			"clientID":                goal.ClientID,
			"goalType":                goal.GoalType,
			"goalName":                goal.GoalName,
			"description":             goal.Description,
			"targetAmount":            goal.TargetAmount,
			"targetDate":              goal.TargetDate,
			"currentProgress":         goal.CurrentProgress,
			"monthlyContribution":     goal.MonthlyContribution,
			"assumedReturnRate":       goal.AssumedReturnRate,
			"projectedCompletionDate": goal.ProjectedCompletionDate,
			"confidenceLevel":         goal.ConfidenceLevel,
			"status":                  goal.Status,
			"createdAt":               goal.CreatedAt,
			"updatedAt":               goal.UpdatedAt,
		}

		_, err := s.hasura.Mutate(mutation, variables)
		if err == nil {
			return nil
		}
		// Fall through to SQL on Hasura error
	}

	// SQL fallback
	query := `
		INSERT INTO client_goals (
			goal_id, client_id, goal_type, goal_name, description, target_amount, target_date,
			current_progress, monthly_contribution, assumed_return_rate, projected_completion_date,
			confidence_level, status, created_at, updated_at
		) VALUES (
			:goal_id, :client_id, :goal_type, :goal_name, :description, :target_amount, :target_date,
			:current_progress, :monthly_contribution, :assumed_return_rate, :projected_completion_date,
			:confidence_level, :status, :created_at, :updated_at
		)
	`
	_, err := s.db.NamedExecContext(ctx, query, goal)
	return err
}

// updateGoalProgressRecord updates a goal's progress
// TODO: Already has Hasura GraphQL mutation implemented
// SQL fallback: UPDATE current_progress with NOW()
func (s *service) updateGoalProgressRecord(ctx context.Context, goalID uuid.UUID, currentProgress float64) error {
	if s.hasura != nil {
		mutation := `
			mutation UpdateGoalProgress($goalID: uuid!, $progress: numeric!) {
				update_client_goals_by_pk(
					pk_columns: {goal_id: $goalID}
					_set: {
						current_progress: $progress
						updated_at: "now()"
					}
				) {
					goal_id
				}
			}
		`

		variables := map[string]interface{}{
			"goalID":   goalID,
			"progress": currentProgress,
		}

		_, err := s.hasura.Mutate(mutation, variables)
		if err == nil {
			return nil
		}
		// Fall through to SQL on Hasura error
	}

	// SQL fallback
	query := `
		UPDATE client_goals
		SET current_progress = $1, updated_at = NOW()
		WHERE goal_id = $2
	`
	_, err := s.db.ExecContext(ctx, query, currentProgress, goalID)
	return err
}
