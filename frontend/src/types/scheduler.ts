/**
 * World-Class Enterprise Scheduler - Type Definitions
 * Comprehensive types for job scheduling, dependencies, calendars, and notifications
 */

// ============================================================================
// Core Enums
// ============================================================================

export enum JobStatus {
  PENDING = 'pending',
  QUEUED = 'queued',
  RUNNING = 'running',
  COMPLETED = 'completed',
  FAILED = 'failed',
  CANCELLED = 'cancelled',
  PAUSED = 'paused',
  WAITING_DEPENDENCY = 'waiting_dependency',
  WAITING_CALENDAR = 'waiting_calendar',
  RETRYING = 'retrying',
  SKIPPED = 'skipped',
}

export enum JobPriority {
  CRITICAL = 'critical',
  HIGH = 'high',
  MEDIUM = 'medium',
  LOW = 'low',
}

export enum DependencyType {
  FINISH_TO_START = 'finish_to_start',     // B starts when A finishes
  START_TO_START = 'start_to_start',       // B starts when A starts
  FINISH_TO_FINISH = 'finish_to_finish',   // B finishes when A finishes
  START_TO_FINISH = 'start_to_finish',     // B finishes when A starts
}

export enum ScheduleType {
  ONCE = 'once',
  RECURRING = 'recurring',
  CRON = 'cron',
  EVENT_DRIVEN = 'event_driven',
  CALENDAR_BASED = 'calendar_based',
}

export enum RecurrencePattern {
  DAILY = 'daily',
  WEEKLY = 'weekly',
  BIWEEKLY = 'biweekly',
  MONTHLY = 'monthly',
  QUARTERLY = 'quarterly',
  YEARLY = 'yearly',
  CUSTOM = 'custom',
}

export enum DayOfWeek {
  MONDAY = 'monday',
  TUESDAY = 'tuesday',
  WEDNESDAY = 'wednesday',
  THURSDAY = 'thursday',
  FRIDAY = 'friday',
  SATURDAY = 'saturday',
  SUNDAY = 'sunday',
}

export enum NotificationChannel {
  EMAIL = 'email',
  SMS = 'sms',
  SLACK = 'slack',
  TEAMS = 'teams',
  WEBHOOK = 'webhook',
  IN_APP = 'in_app',
  PAGERDUTY = 'pagerduty',
}

export enum NotificationTrigger {
  ON_START = 'on_start',
  ON_COMPLETION = 'on_completion',
  ON_FAILURE = 'on_failure',
  ON_RETRY = 'on_retry',
  ON_TIMEOUT = 'on_timeout',
  ON_SLA_BREACH = 'on_sla_breach',
  ON_DEPENDENCY_FAILURE = 'on_dependency_failure',
  ON_APPROVAL_REQUIRED = 'on_approval_required',
}

export enum AuditAction {
  JOB_CREATED = 'job_created',
  JOB_UPDATED = 'job_updated',
  JOB_DELETED = 'job_deleted',
  JOB_STARTED = 'job_started',
  JOB_COMPLETED = 'job_completed',
  JOB_FAILED = 'job_failed',
  JOB_CANCELLED = 'job_cancelled',
  JOB_RESUBMITTED = 'job_resubmitted',
  JOB_PAUSED = 'job_paused',
  JOB_RESUMED = 'job_resumed',
  SCHEDULE_CREATED = 'schedule_created',
  SCHEDULE_UPDATED = 'schedule_updated',
  SCHEDULE_DELETED = 'schedule_deleted',
  CALENDAR_UPDATED = 'calendar_updated',
  DEPENDENCY_ADDED = 'dependency_added',
  DEPENDENCY_REMOVED = 'dependency_removed',
  NOTIFICATION_SENT = 'notification_sent',
  SLA_BREACH = 'sla_breach',
  APPROVAL_REQUESTED = 'approval_requested',
  APPROVAL_GRANTED = 'approval_granted',
  APPROVAL_DENIED = 'approval_denied',
}

// ============================================================================
// Core Models
// ============================================================================

export interface Job {
  id: string;
  name: string;
  description?: string;
  tenant_id: string;
  tenant_instance_id?: string;

  // Job configuration
  job_type: string;
  job_category?: string;
  payload: Record<string, unknown>;
  parameters?: JobParameter[];

  // Execution settings
  priority: JobPriority;
  timeout_seconds?: number;
  max_retries: number;
  retry_delay_seconds: number;
  retry_backoff_multiplier?: number;

  // Scheduling
  schedule?: Schedule;
  schedule_id?: string;

  // Dependencies
  dependencies?: JobDependency[];

  // Notifications
  notifications?: NotificationRule[];

  // SLA & Compliance
  sla_deadline_minutes?: number;
  requires_approval?: boolean;
  approval_roles?: string[];
  compliance_tags?: string[];

  // Metadata
  owner_id?: string;
  owner_email?: string;
  team_id?: string;
  tags?: string[];
  metadata?: Record<string, unknown>;

  // State
  enabled: boolean;
  created_at: string;
  updated_at: string;
  created_by: string;
  updated_by?: string;
  version: number;
}

export interface JobParameter {
  name: string;
  type: 'string' | 'number' | 'boolean' | 'date' | 'datetime' | 'json' | 'file';
  required: boolean;
  default_value?: unknown;
  description?: string;
  validation_regex?: string;
  allowed_values?: unknown[];
}

export interface JobExecution {
  id: string;
  job_id: string;
  job_name: string;
  run_number: number;

  // Execution state
  status: JobStatus;
  progress_percent?: number;
  current_step?: string;
  total_steps?: number;
  completed_steps?: number;

  // Timing
  scheduled_at?: string;
  queued_at?: string;
  started_at?: string;
  completed_at?: string;
  duration_ms?: number;

  // Results
  output?: Record<string, unknown>;
  error_message?: string;
  error_code?: string;
  error_stack?: string;

  // Retry information
  attempt_number: number;
  max_attempts: number;
  next_retry_at?: string;

  // Context
  triggered_by: 'schedule' | 'manual' | 'api' | 'dependency' | 'event' | 'resubmit';
  triggered_by_user_id?: string;
  parent_execution_id?: string;
  correlation_id?: string;

  // Resources
  worker_id?: string;
  worker_host?: string;
  memory_used_mb?: number;
  cpu_seconds?: number;

  // Audit
  logs?: ExecutionLog[];

  created_at: string;
  updated_at: string;
}

export interface ExecutionLog {
  id: string;
  execution_id: string;
  timestamp: string;
  level: 'debug' | 'info' | 'warn' | 'error';
  message: string;
  step?: string;
  metadata?: Record<string, unknown>;
}

export interface Schedule {
  id: string;
  name: string;
  description?: string;
  tenant_id: string;

  // Schedule type
  schedule_type: ScheduleType;

  // One-time schedule
  run_at?: string;

  // Recurring schedule
  recurrence?: RecurrenceConfig;

  // Cron expression
  cron_expression?: string;
  cron_timezone?: string;

  // Event-driven
  event_trigger?: EventTrigger;

  // Calendar-based
  calendar_id?: string;
  calendar_offset_days?: number;
  calendar_offset_direction?: 'before' | 'after';

  // Constraints
  valid_from?: string;
  valid_until?: string;
  skip_holidays?: boolean;
  skip_weekends?: boolean;
  business_days_only?: boolean;

  // State
  enabled: boolean;
  next_run_at?: string;
  last_run_at?: string;

  created_at: string;
  updated_at: string;
}

export interface RecurrenceConfig {
  pattern: RecurrencePattern;
  interval: number;  // every N days/weeks/months

  // Weekly options
  days_of_week?: DayOfWeek[];

  // Monthly options
  day_of_month?: number;
  week_of_month?: number;  // 1-4 or -1 for last

  // Time of day
  time_of_day?: string;  // HH:MM format
  timezone?: string;

  // End conditions
  end_after_occurrences?: number;
  end_date?: string;
}

export interface EventTrigger {
  event_type: string;
  event_source?: string;
  filter_expression?: string;  // JSON path or expression
  debounce_seconds?: number;
}

// ============================================================================
// Dependencies
// ============================================================================

export interface JobDependency {
  id: string;
  job_id: string;
  depends_on_job_id: string;
  depends_on_job_name?: string;

  dependency_type: DependencyType;

  // Conditions
  required_status?: JobStatus[];  // default: [COMPLETED]
  condition_expression?: string;

  // Lag/Lead time
  lag_minutes?: number;  // delay after dependency condition is met

  // Handling
  on_dependency_failure?: 'fail' | 'skip' | 'continue' | 'wait';
  timeout_minutes?: number;

  created_at: string;
}

export interface JobChain {
  id: string;
  name: string;
  description?: string;
  tenant_id: string;

  // Jobs in the chain
  jobs: ChainJob[];

  // Global settings
  stop_on_failure?: boolean;
  parallel_execution?: boolean;
  max_parallel?: number;

  // Schedule (optional - can be triggered manually)
  schedule_id?: string;

  enabled: boolean;
  created_at: string;
  updated_at: string;
}

export interface ChainJob {
  job_id: string;
  job_name?: string;
  order: number;

  // Parallel group - jobs with same group run in parallel
  parallel_group?: number;

  // Override job settings for this chain
  parameter_overrides?: Record<string, unknown>;
  timeout_override?: number;

  // Conditions
  run_condition?: string;  // expression to evaluate
  skip_condition?: string;
}

// ============================================================================
// Calendars
// ============================================================================

export interface WorkingHours {
  mon: { start: string; end: string };
  tue: { start: string; end: string };
  wed: { start: string; end: string };
  thu: { start: string; end: string };
  fri: { start: string; end: string };
  sat: { start: string; end: string };
  sun: { start: string; end: string };
}

export interface BusinessCalendar {
  id: string;
  name: string;
  description?: string;
  tenant_id: string;

  // Base calendar
  timezone: string;
  country_code?: string;
  region_code?: string;

  // Working days
  working_days: DayOfWeek[];
  working_hours_start?: string;  // HH:MM
  working_hours_end?: string;
  working_hours: WorkingHours;  // Extended working hours by day

  // Holidays
  holidays: Holiday[];

  // Custom non-working days
  custom_non_working_days?: CustomNonWorkingDay[];

  // Special working days (override holidays)
  special_working_days?: string[];  // ISO dates

  // Inheritance
  parent_calendar_id?: string;

  // Default flag
  is_default?: boolean;

  enabled: boolean;
  created_at: string;
  updated_at: string;
}

export interface Holiday {
  id: string;
  name: string;
  date: string;  // ISO date
  recurring: boolean;
  recurring_month?: number;
  recurring_day?: number;
  observed_rule?: 'exact' | 'nearest_weekday' | 'following_monday';
}

export interface CustomNonWorkingDay {
  date: string;
  reason: string;
  recurring?: boolean;
}

// ============================================================================
// Notifications
// ============================================================================

export interface NotificationRule {
  id: string;
  job_id?: string;
  job_chain_id?: string;
  tenant_id: string;

  name: string;
  description?: string;

  // Triggers
  triggers: NotificationTrigger[];

  // Recipients
  recipients: NotificationRecipient[];

  // Template
  template_id?: string;
  custom_subject?: string;
  custom_body?: string;

  // Conditions
  condition_expression?: string;

  // Rate limiting
  cooldown_minutes?: number;
  max_notifications_per_hour?: number;

  enabled: boolean;
  created_at: string;
  updated_at: string;
}

export interface NotificationRecipient {
  channel: NotificationChannel;
  address: string;  // email, phone, webhook URL, etc.

  // Channel-specific settings
  settings?: Record<string, unknown>;

  // Localization
  language?: string;
  timezone?: string;
}

export interface NotificationTemplate {
  id: string;
  name: string;
  tenant_id: string;

  // Content by language
  content: Record<string, NotificationContent>;  // keyed by language code

  // Supported channels
  channels: NotificationChannel[];

  created_at: string;
  updated_at: string;
}

export interface NotificationContent {
  subject: string;
  body: string;
  html_body?: string;

  // Channel-specific content
  slack_blocks?: unknown;
  teams_card?: unknown;
}

export interface NotificationHistory {
  id: string;
  notification_rule_id: string;
  execution_id?: string;
  job_id?: string;

  trigger: NotificationTrigger;
  channel: NotificationChannel;
  recipient: string;

  subject?: string;
  body?: string;

  status: 'pending' | 'sent' | 'delivered' | 'failed' | 'bounced';
  error_message?: string;

  sent_at?: string;
  delivered_at?: string;

  created_at: string;
}

// ============================================================================
// Audit & Compliance
// ============================================================================

export interface AuditLog {
  id: string;
  tenant_id: string;

  // Action details
  action: AuditAction;
  resource_type: 'job' | 'execution' | 'schedule' | 'calendar' | 'notification' | 'dependency' | 'chain';
  resource_id: string;
  resource_name?: string;

  // Actor
  user_id?: string;
  user_email?: string;
  user_name?: string;
  service_account?: string;
  ip_address?: string;
  user_agent?: string;

  // Changes
  old_value?: Record<string, unknown>;
  new_value?: Record<string, unknown>;
  change_summary?: string;

  // Context
  correlation_id?: string;
  request_id?: string;
  session_id?: string;

  // Compliance
  compliance_flags?: string[];
  data_classification?: string;
  retention_days?: number;

  timestamp: string;
}

export interface ComplianceReport {
  id: string;
  tenant_id: string;

  report_type: 'execution_summary' | 'sla_compliance' | 'failure_analysis' | 'audit_trail' | 'access_log';
  report_name: string;

  // Date range
  start_date: string;
  end_date: string;

  // Filters
  job_ids?: string[];
  job_types?: string[];
  statuses?: JobStatus[];

  // Generated report
  generated_at: string;
  generated_by: string;

  summary: ComplianceReportSummary;
  details?: unknown;

  // Export
  export_format?: 'json' | 'csv' | 'pdf' | 'excel';
  export_url?: string;
}

export interface ComplianceReportSummary {
  total_executions: number;
  successful_executions: number;
  failed_executions: number;

  sla_compliance_rate: number;
  average_duration_ms: number;

  top_failures?: { job_name: string; failure_count: number; }[];
  sla_breaches?: { job_name: string; breach_count: number; }[];
}

// ============================================================================
// Dashboard & Metrics
// ============================================================================

export interface SchedulerDashboardMetrics {
  // Overview
  total_jobs: number;
  active_jobs: number;
  paused_jobs: number;

  // Today's executions
  executions_today: number;
  successful_today: number;
  failed_today: number;
  running_now: number;
  queued_now: number;

  // Performance
  average_duration_ms: number;
  p95_duration_ms: number;
  success_rate_7d: number;

  // SLA
  sla_compliance_rate: number;
  sla_breaches_today: number;

  // Upcoming
  next_scheduled_jobs: UpcomingJob[];

  // Recent failures
  recent_failures: RecentFailure[];

  // Resource utilization
  worker_count: number;
  active_workers: number;
  queue_depth: number;
}

export interface UpcomingJob {
  job_id: string;
  job_name: string;
  scheduled_at: string;
  schedule_name?: string;
}

export interface RecentFailure {
  execution_id: string;
  job_id: string;
  job_name: string;
  failed_at: string;
  error_message?: string;
  attempt_number: number;
}

// ============================================================================
// API Request/Response Types
// ============================================================================

export interface PaginatedResponse<T> {
  data: T[];
  total: number;
  page: number;
  page_size: number;
  total_pages: number;
}

export interface JobListFilters {
  status?: JobStatus[];
  priority?: JobPriority[];
  job_type?: string[];
  owner_id?: string;
  team_id?: string;
  tags?: string[];
  enabled?: boolean;
  search?: string;
  created_after?: string;
  created_before?: string;
}

export interface ExecutionListFilters {
  job_id?: string;
  status?: JobStatus[];
  triggered_by?: string;
  started_after?: string;
  started_before?: string;
  has_errors?: boolean;
}

export interface CreateJobRequest {
  name: string;
  description?: string;
  job_type: string;
  job_category?: string;
  payload: Record<string, unknown>;
  parameters?: JobParameter[];
  priority?: JobPriority;
  timeout_seconds?: number;
  max_retries?: number;
  retry_delay_seconds?: number;
  schedule?: Partial<Schedule>;
  dependencies?: Partial<JobDependency>[];
  notifications?: Partial<NotificationRule>[];
  sla_deadline_minutes?: number;
  requires_approval?: boolean;
  approval_roles?: string[];
  tags?: string[];
  metadata?: Record<string, unknown>;
}

export interface UpdateJobRequest extends Partial<CreateJobRequest> {
  enabled?: boolean;
}

export interface TriggerJobRequest {
  parameters?: Record<string, unknown>;
  priority_override?: JobPriority;
  skip_dependencies?: boolean;
  correlation_id?: string;
}

export interface ResubmitExecutionRequest {
  use_same_parameters?: boolean;
  parameter_overrides?: Record<string, unknown>;
  skip_failed_steps?: boolean;
}
