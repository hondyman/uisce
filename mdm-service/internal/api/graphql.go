package api

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/hondyman/semlayer/mdm-service/internal/domain"
)

// ============================================================================
// GraphQL Types (Schema)
// ============================================================================

// HolidayScheduleGQL is the GraphQL representation
type HolidayScheduleGQL struct {
	ID              string  `json:"id"`
	CalendarDate    string  `json:"calendar_date"`
	IsBusinessDay   bool    `json:"is_business_day"`
	RegionCode      string  `json:"region_code"`
	ExchangeCode    *string `json:"exchange_code"`
	HolidayName     *string `json:"holiday_name"`
	SourceType      string  `json:"source_type"`
	ConfidenceScore int     `json:"confidence_score"`
}

// LineageRecordGQL represents lineage in GraphQL
type LineageRecordGQL struct {
	ID               string  `json:"id"`
	SemanticTerm     string  `json:"semantic_term"`
	PreviousValue    *string `json:"previous_value"`
	WinningValue     string  `json:"winning_value"`
	WinningSourceID  *string `json:"winning_source_id"`
	RuleApplied      string  `json:"rule_applied"`
	ExecutionTime    string  `json:"execution_time"`
	ConflictDetected bool    `json:"conflict_detected"`
}

// ConflictRecordGQL represents conflicts
type ConflictRecordGQL struct {
	ID             string `json:"id"`
	GoldenRecordID string `json:"golden_record_id"`
	ConflictType   string `json:"conflict_type"`
	Severity       string `json:"severity"`
	Status         string `json:"status"`
	CreatedAt      string `json:"created_at"`
}

// ============================================================================
// Query Resolvers
// ============================================================================

// QueryResolver holds query resolution methods
type QueryResolver struct {
	handler *Handler
}

// NewQueryResolver creates a new query resolver
func NewQueryResolver(handler *Handler) *QueryResolver {
	return &QueryResolver{handler: handler}
}

// GetGoldenCalendarArgs represents arguments for GetGoldenCalendar query
type GetGoldenCalendarArgs struct {
	TenantID  string
	StartDate string // YYYY-MM-DD format
	EndDate   string // YYYY-MM-DD format
	Region    string
	Exchange  *string
}

// GetGoldenCalendar resolves getGoldenCalendar(start: Date!, end: Date!, region: String!, exchange: String)
func (r *QueryResolver) GetGoldenCalendar(ctx context.Context, args GetGoldenCalendarArgs) ([]HolidayScheduleGQL, error) {
	tenantID, err := uuid.Parse(args.TenantID)
	if err != nil {
		return nil, err
	}

	startDate, err := time.Parse("2006-01-02", args.StartDate)
	if err != nil {
		return nil, err
	}

	endDate, err := time.Parse("2006-01-02", args.EndDate)
	if err != nil {
		return nil, err
	}

	req := &domain.GetGoldenCalendarRequest{
		StartDate: startDate,
		EndDate:   endDate,
		Region:    args.Region,
		Exchange:  args.Exchange,
	}

	response, err := r.handler.svc.GetGoldenCalendar(ctx, tenantID, req)
	if err != nil {
		return nil, err
	}

	// Convert to GraphQL types
	var results []HolidayScheduleGQL
	for _, record := range response.Records {
		results = append(results, HolidayScheduleGQL{
			ID:              record.ID.String(),
			CalendarDate:    record.CalendarDate.Format("2006-01-02"),
			IsBusinessDay:   record.IsBusinessDay,
			RegionCode:      record.RegionCode,
			ExchangeCode:    record.ExchangeCode,
			HolidayName:     record.HolidayName,
			SourceType:      record.SourceType,
			ConfidenceScore: record.ConfidenceScore,
		})
	}

	return results, nil
}

// GetCalendarLineageArgs represents arguments for GetCalendarLineage query
type GetCalendarLineageArgs struct {
	TenantID       string
	GoldenRecordID string
}

// GetCalendarLineage resolves getCalendarLineage(golden_id: UUID!)
func (r *QueryResolver) GetCalendarLineage(ctx context.Context, args GetCalendarLineageArgs) ([]LineageRecordGQL, error) {
	tenantID, err := uuid.Parse(args.TenantID)
	if err != nil {
		return nil, err
	}

	goldenID, err := uuid.Parse(args.GoldenRecordID)
	if err != nil {
		return nil, err
	}

	lineage, err := r.handler.svc.GetLineageForRecord(ctx, tenantID, goldenID)
	if err != nil {
		return nil, err
	}

	// Convert to GraphQL types
	var results []LineageRecordGQL
	for _, record := range lineage.History {
		results = append(results, LineageRecordGQL{
			ID:               record.ID.String(),
			SemanticTerm:     record.SemanticTerm,
			PreviousValue:    record.PreviousValue,
			WinningValue:     record.WinningValue,
			WinningSourceID:  (*string)(record.WinningSourceID),
			RuleApplied:      record.RuleApplied,
			ExecutionTime:    record.ExecutionTime.Format("2006-01-02T15:04:05Z07:00"),
			ConflictDetected: record.ConflictDetected,
		})
	}

	return results, nil
}

// GetOpenConflictsArgs represents arguments for GetOpenConflicts query
type GetOpenConflictsArgs struct {
	TenantID string
	Region   *string
}

// GetOpenConflicts resolves getOpenConflicts
func (r *QueryResolver) GetOpenConflicts(ctx context.Context, args GetOpenConflictsArgs) ([]ConflictRecordGQL, error) {
	tenantID, err := uuid.Parse(args.TenantID)
	if err != nil {
		return nil, err
	}

	conflicts, err := r.handler.svc.GetRepo().GetOpenConflicts(ctx, tenantID)
	if err != nil {
		return nil, err
	}

	// Convert to GraphQL types
	var results []ConflictRecordGQL
	for _, conflict := range conflicts {
		results = append(results, ConflictRecordGQL{
			ID:             conflict.ID.String(),
			GoldenRecordID: conflict.GoldenRecordID.String(),
			ConflictType:   conflict.ConflictType,
			Severity:       conflict.Severity,
			Status:         conflict.Status,
			CreatedAt:      conflict.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		})
	}

	return results, nil
}

// GetHealthCheckArgs represents arguments for GetHealthCheck query
type GetHealthCheckArgs struct {
	TenantID string
}

// GetHealthCheck resolves getHealthCheck query
func (r *QueryResolver) GetHealthCheck(ctx context.Context, args GetHealthCheckArgs) (interface{}, error) {
	tenantID, err := uuid.Parse(args.TenantID)
	if err != nil {
		return nil, err
	}

	return r.handler.svc.GetHealthMetrics(ctx, tenantID)
}

// ============================================================================
// GraphQL Schema Definition (SDL)
// ============================================================================

// GraphQLSchema returns the GraphQL schema as a string
func GraphQLSchema() string {
	return `
scalar Date
scalar DateTime
scalar UUID

# Represents a trusted calendar entry (gold copy)
type HolidaySchedule {
  id: UUID!
  calendar_date: Date!
  is_business_day: Boolean!
  region_code: String!
  exchange_code: String
  holiday_name: String
  source_type: String!
  confidence_score: Int!
}

# Represents the lineage/audit trail for a calendar entry
type LineageRecord {
  id: UUID!
  semantic_term: String!
  previous_value: String
  winning_value: String!
  winning_source_id: UUID
  rule_applied: String!
  execution_time: DateTime!
  conflict_detected: Boolean!
}

# Represents a data quality conflict flagged for stewardship
type ConflictRecord {
  id: UUID!
  golden_record_id: UUID!
  conflict_type: String!
  severity: String! # low, medium, high, critical
  status: String! # open, in_review, resolved, rejected
  created_at: DateTime!
}

# Health metrics for monitoring
type HealthMetrics {
  tenant_id: UUID!
  coverage_percentage: Float!
  conflict_count: Int!
  high_confidence_percentage: Float!
  days_since_last_official_feed: Int!
  status: String! # healthy, warning, critical
}

type Query {
  # Get the golden calendar for a date range
  getGoldenCalendar(
    start_date: Date!, 
    end_date: Date!, 
    region: String!, 
    exchange: String
  ): [HolidaySchedule!]!

  # Get the lineage/audit trail for a calendar entry
  getCalendarLineage(
    golden_id: UUID!
  ): [LineageRecord!]!

  # Get open conflicts for stewardship review
  getOpenConflicts(
    region: String
  ): [ConflictRecord!]!

  # Get health metrics
  getHealthCheck: HealthMetrics!

  # Check if a specific date is a business day
  isBusinessDay(
    date: Date!, 
    region: String!, 
    exchange: String
  ): Boolean!
}

type Mutation {
  # Ingest calendar data from a source
  ingestCalendarData(
    source_system: String!,
    data: [CalendarInput!]!,
    is_official: Boolean
  ): IngestResult!

  # Resolve a conflict
  resolveConflict(
    conflict_id: UUID!,
    winning_value: Boolean!,
    notes: String
  ): ConflictRecord!
}

input CalendarInput {
  date: Date!
  region: String!
  exchange: String
  is_business_day: Boolean!
  holiday_name: String
}

type IngestResult {
  golden_record_ids: [UUID!]!
  count: Int!
  ingested_at: DateTime!
}
`
}
