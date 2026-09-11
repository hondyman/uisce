package reports

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// ReportSchedule represents a recurring report generation schedule linked to a template.
type ReportSchedule struct {
	ID                  uuid.UUID  `json:"id" db:"id"`
	TenantID            uuid.UUID  `json:"tenant_id" db:"tenant_id"`
	ReportDefinitionID  uuid.UUID  `json:"report_definition_id" db:"report_definition_id"`
	OwnerID             string     `json:"owner_id" db:"owner_id"`
	ScheduleName        string     `json:"schedule_name" db:"schedule_name"`
	CronExpression      string     `json:"cron_expression" db:"cron_expression"`
	Region              string     `json:"region" db:"region"`
	CalendarID          *uuid.UUID `json:"calendar_id,omitempty" db:"calendar_id"`
	StartOfDayTime      string     `json:"start_of_day_time" db:"start_of_day_time"`
	UnscheduledBehavior string     `json:"unscheduled_behavior" db:"unscheduled_behavior"`
	BusinessDayOffset   int        `json:"business_day_offset" db:"business_day_offset"`
	BurstDimension      string     `json:"burst_dimension" db:"burst_dimension"`
	ExportFormat        string     `json:"export_format" db:"export_format"`
	NotificationChannels map[string]interface{} `json:"notification_channels" db:"notification_channels"`
	IsActive            bool       `json:"is_active" db:"is_active"`
	DeletedAt           *time.Time `json:"deleted_at,omitempty" db:"deleted_at"`
	LastRunAt           *time.Time `json:"last_run_at,omitempty" db:"last_run_at"`
	NextRunAt           *time.Time `json:"next_run_at,omitempty" db:"next_run_at"`
	CreatedAt           time.Time  `json:"created_at" db:"created_at"`
}

// CreateScheduleInput contains the fields required to create a new report schedule.
type CreateScheduleInput struct {
	TemplateID          uuid.UUID
	ScheduleName        string
	CronExpression      string
	Region              string
	CalendarID          *uuid.UUID
	StartOfDayTime      string
	UnscheduledBehavior string
	BusinessDayOffset   int
	BurstDimension      string
	ExportFormat        string
	NotifyInApp         bool
	NotifyEmail         bool
}

// ScheduleExecutionResult describes the output of a triggered schedule run.
type ScheduleExecutionResult struct {
	ExecutionID uuid.UUID `json:"execution_id"`
	OutputURL   string    `json:"output_url"`
	Status      string    `json:"status"`
	RequestedBy string    `json:"requested_by"`
	TenantID    uuid.UUID `json:"tenant_id"`
	TemplateID  uuid.UUID `json:"template_id"`
}

// ReportExecutor executes a report template and returns execution details.
// Injectable interface to decouple repository tests from live rendering services.
type ReportExecutor interface {
	ExecuteReport(ctx context.Context, template *ReportTemplate, params map[string]interface{}) (*ScheduleExecutionResult, error)
}
