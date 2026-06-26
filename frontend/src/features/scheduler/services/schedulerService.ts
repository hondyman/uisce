/**
 * World-Class Enterprise Scheduler - API Service
 * Comprehensive API client for all scheduler operations
 */

import {
  Job,
  JobExecution,
  ExecutionLog,
  Schedule,
  JobDependency,
  JobChain,
  BusinessCalendar,
  Holiday,
  NotificationRule,
  NotificationTemplate,
  NotificationHistory,
  AuditLog,
  ComplianceReport,
  SchedulerDashboardMetrics,
  PaginatedResponse,
  JobListFilters,
  ExecutionListFilters,
  CreateJobRequest,
  UpdateJobRequest,
  TriggerJobRequest,
  ResubmitExecutionRequest,
  JobStatus,
} from '../../../types/scheduler';

// ============================================================================
// Utilities
// ============================================================================

type TenantHeaders = Record<string, string>;

function getTenantHeaders(): TenantHeaders {
  try {
    const t = window.localStorage.getItem('selected_tenant');
    const d = window.localStorage.getItem('selected_datasource');
    const tenant = t ? JSON.parse(t) : null;
    const datasource = d ? JSON.parse(d) : null;
    const headers: TenantHeaders = {};
    if (tenant?.id) headers['X-Tenant-ID'] = tenant.id;
    if (datasource?.id) headers['X-Tenant-Datasource-ID'] = datasource.id;
    return headers;
  } catch {
    return {};
  }
}

function getTenantQueryParams(): string {
  try {
    const t = window.localStorage.getItem('selected_tenant');
    const d = window.localStorage.getItem('selected_datasource');
    const tenant = t ? JSON.parse(t) : null;
    const datasource = d ? JSON.parse(d) : null;
    const params = new URLSearchParams();
    if (tenant?.id) params.set('tenant_id', tenant.id);
    if (datasource?.id) params.set('tenant_instance_id', datasource.id);
    return params.toString();
  } catch {
    return '';
  }
}

async function handleResponse<T>(response: Response): Promise<T> {
  if (!response.ok) {
    const errorText = await response.text();
    let errorMessage: string;
    try {
      const errorJson = JSON.parse(errorText);
      errorMessage = errorJson.error || errorJson.message || errorText;
    } catch {
      errorMessage = errorText;
    }
    throw new Error(`API Error (${response.status}): ${errorMessage}`);
  }
  return response.json();
}

function buildUrl(path: string, params?: Record<string, string | number | boolean | undefined>): string {
  const tenantParams = getTenantQueryParams();
  const url = new URL(path, window.location.origin);

  if (tenantParams) {
    const existingParams = new URLSearchParams(tenantParams);
    existingParams.forEach((value, key) => url.searchParams.set(key, value));
  }

  if (params) {
    Object.entries(params).forEach(([key, value]) => {
      if (value !== undefined) {
        url.searchParams.set(key, String(value));
      }
    });
  }

  return url.toString();
}

// ============================================================================
// Jobs API
// ============================================================================

export async function listJobs(
  filters?: JobListFilters,
  page = 1,
  pageSize = 20
): Promise<PaginatedResponse<Job>> {
  const params: Record<string, string | number | boolean | undefined> = {
    page,
    page_size: pageSize,
  };

  if (filters) {
    if (filters.status?.length) params.status = filters.status.join(',');
    if (filters.priority?.length) params.priority = filters.priority.join(',');
    if (filters.job_type?.length) params.job_type = filters.job_type.join(',');
    if (filters.owner_id) params.owner_id = filters.owner_id;
    if (filters.team_id) params.team_id = filters.team_id;
    if (filters.tags?.length) params.tags = filters.tags.join(',');
    if (filters.enabled !== undefined) params.enabled = filters.enabled;
    if (filters.search) params.search = filters.search;
    if (filters.created_after) params.created_after = filters.created_after;
    if (filters.created_before) params.created_before = filters.created_before;
  }

  const response = await fetch(buildUrl('/api/scheduler/jobs', params), {
    headers: { 'Content-Type': 'application/json', ...getTenantHeaders() },
  });

  return handleResponse<PaginatedResponse<Job>>(response);
}

export async function getJob(jobId: string): Promise<Job> {
  const response = await fetch(buildUrl(`/api/scheduler/jobs/${jobId}`), {
    headers: { 'Content-Type': 'application/json', ...getTenantHeaders() },
  });
  return handleResponse<Job>(response);
}

export async function createJob(job: CreateJobRequest): Promise<Job> {
  const response = await fetch(buildUrl('/api/scheduler/jobs'), {
    method: 'POST',
    headers: { 'Content-Type': 'application/json', ...getTenantHeaders() },
    body: JSON.stringify(job),
  });
  return handleResponse<Job>(response);
}

export async function updateJob(jobId: string, updates: UpdateJobRequest): Promise<Job> {
  const response = await fetch(buildUrl(`/api/scheduler/jobs/${jobId}`), {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json', ...getTenantHeaders() },
    body: JSON.stringify(updates),
  });
  return handleResponse<Job>(response);
}

export async function deleteJob(jobId: string): Promise<void> {
  const response = await fetch(buildUrl(`/api/scheduler/jobs/${jobId}`), {
    method: 'DELETE',
    headers: { 'Content-Type': 'application/json', ...getTenantHeaders() },
  });
  if (!response.ok) {
    throw new Error(`Failed to delete job: ${response.status}`);
  }
}

export async function triggerJob(jobId: string, request?: TriggerJobRequest): Promise<JobExecution> {
  const response = await fetch(buildUrl(`/api/scheduler/jobs/${jobId}/trigger`), {
    method: 'POST',
    headers: { 'Content-Type': 'application/json', ...getTenantHeaders() },
    body: JSON.stringify(request || {}),
  });
  return handleResponse<JobExecution>(response);
}

export async function pauseJob(jobId: string): Promise<Job> {
  const response = await fetch(buildUrl(`/api/scheduler/jobs/${jobId}/pause`), {
    method: 'POST',
    headers: { 'Content-Type': 'application/json', ...getTenantHeaders() },
  });
  return handleResponse<Job>(response);
}

export async function resumeJob(jobId: string): Promise<Job> {
  const response = await fetch(buildUrl(`/api/scheduler/jobs/${jobId}/resume`), {
    method: 'POST',
    headers: { 'Content-Type': 'application/json', ...getTenantHeaders() },
  });
  return handleResponse<Job>(response);
}

export async function cloneJob(jobId: string, newName: string): Promise<Job> {
  const response = await fetch(buildUrl(`/api/scheduler/jobs/${jobId}/clone`), {
    method: 'POST',
    headers: { 'Content-Type': 'application/json', ...getTenantHeaders() },
    body: JSON.stringify({ name: newName }),
  });
  return handleResponse<Job>(response);
}

// ============================================================================
// Executions API
// ============================================================================

export async function listExecutions(
  filters?: ExecutionListFilters,
  page = 1,
  pageSize = 20
): Promise<PaginatedResponse<JobExecution>> {
  const params: Record<string, string | number | boolean | undefined> = {
    page,
    page_size: pageSize,
  };

  if (filters) {
    if (filters.job_id) params.job_id = filters.job_id;
    if (filters.status?.length) params.status = filters.status.join(',');
    if (filters.triggered_by) params.triggered_by = filters.triggered_by;
    if (filters.started_after) params.started_after = filters.started_after;
    if (filters.started_before) params.started_before = filters.started_before;
    if (filters.has_errors !== undefined) params.has_errors = filters.has_errors;
  }

  const response = await fetch(buildUrl('/api/scheduler/executions', params), {
    headers: { 'Content-Type': 'application/json', ...getTenantHeaders() },
  });

  return handleResponse<PaginatedResponse<JobExecution>>(response);
}

/**
 * Compatible export for older components expecting 'executions' field
 */
export async function listAllExecutions(
  filters?: any,
  page = 1,
  limit = 25
): Promise<{ executions: JobExecution[]; total: number }> {
  const response = await listExecutions(filters, page, limit);
  return {
    executions: response.data,
    total: response.total,
  };
}

export async function getExecution(executionId: string): Promise<JobExecution> {
  const response = await fetch(buildUrl(`/api/scheduler/executions/${executionId}`), {
    headers: { 'Content-Type': 'application/json', ...getTenantHeaders() },
  });
  return handleResponse<JobExecution>(response);
}

export async function getExecutionLogs(
  executionId: string,
  level?: string
): Promise<ExecutionLog[]> {
  const params: Record<string, string | undefined> = {};
  if (level) params.level = level;

  const response = await fetch(buildUrl(`/api/scheduler/executions/${executionId}/logs`, params), {
    headers: { 'Content-Type': 'application/json', ...getTenantHeaders() },
  });
  return handleResponse<ExecutionLog[]>(response);
}

export async function cancelExecution(executionId: string, reason?: string): Promise<JobExecution> {
  const response = await fetch(buildUrl(`/api/scheduler/executions/${executionId}/cancel`), {
    method: 'POST',
    headers: { 'Content-Type': 'application/json', ...getTenantHeaders() },
    body: JSON.stringify({ reason }),
  });
  return handleResponse<JobExecution>(response);
}

export async function resubmitExecution(
  executionId: string,
  request?: ResubmitExecutionRequest
): Promise<JobExecution> {
  const response = await fetch(buildUrl(`/api/scheduler/executions/${executionId}/resubmit`), {
    method: 'POST',
    headers: { 'Content-Type': 'application/json', ...getTenantHeaders() },
    body: JSON.stringify(request || {}),
  });
  return handleResponse<JobExecution>(response);
}

// ============================================================================
// Schedules API
// ============================================================================

export async function listSchedules(page = 1, pageSize = 20): Promise<PaginatedResponse<Schedule>> {
  const response = await fetch(buildUrl('/api/scheduler/schedules', { page, page_size: pageSize }), {
    headers: { 'Content-Type': 'application/json', ...getTenantHeaders() },
  });
  return handleResponse<PaginatedResponse<Schedule>>(response);
}

export async function getSchedule(scheduleId: string): Promise<Schedule> {
  const response = await fetch(buildUrl(`/api/scheduler/schedules/${scheduleId}`), {
    headers: { 'Content-Type': 'application/json', ...getTenantHeaders() },
  });
  return handleResponse<Schedule>(response);
}

export async function createSchedule(schedule: Partial<Schedule>): Promise<Schedule> {
  const response = await fetch(buildUrl('/api/scheduler/schedules'), {
    method: 'POST',
    headers: { 'Content-Type': 'application/json', ...getTenantHeaders() },
    body: JSON.stringify(schedule),
  });
  return handleResponse<Schedule>(response);
}

export async function updateSchedule(scheduleId: string, updates: Partial<Schedule>): Promise<Schedule> {
  const response = await fetch(buildUrl(`/api/scheduler/schedules/${scheduleId}`), {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json', ...getTenantHeaders() },
    body: JSON.stringify(updates),
  });
  return handleResponse<Schedule>(response);
}

export async function deleteSchedule(scheduleId: string): Promise<void> {
  const response = await fetch(buildUrl(`/api/scheduler/schedules/${scheduleId}`), {
    method: 'DELETE',
    headers: { 'Content-Type': 'application/json', ...getTenantHeaders() },
  });
  if (!response.ok) {
    throw new Error(`Failed to delete schedule: ${response.status}`);
  }
}

export async function getNextRuns(scheduleId: string, count = 10): Promise<string[]> {
  const response = await fetch(buildUrl(`/api/scheduler/schedules/${scheduleId}/next-runs`, { count }), {
    headers: { 'Content-Type': 'application/json', ...getTenantHeaders() },
  });
  return handleResponse<string[]>(response);
}

// ============================================================================
// Dependencies API
// ============================================================================

export async function getJobDependencies(jobId: string): Promise<JobDependency[]> {
  const response = await fetch(buildUrl(`/api/scheduler/jobs/${jobId}/dependencies`), {
    headers: { 'Content-Type': 'application/json', ...getTenantHeaders() },
  });
  return handleResponse<JobDependency[]>(response);
}

export async function addJobDependency(
  jobId: string,
  dependency: Partial<JobDependency>
): Promise<JobDependency> {
  const response = await fetch(buildUrl(`/api/scheduler/jobs/${jobId}/dependencies`), {
    method: 'POST',
    headers: { 'Content-Type': 'application/json', ...getTenantHeaders() },
    body: JSON.stringify(dependency),
  });
  return handleResponse<JobDependency>(response);
}

export async function removeJobDependency(jobId: string, dependencyId: string): Promise<void> {
  const response = await fetch(buildUrl(`/api/scheduler/jobs/${jobId}/dependencies/${dependencyId}`), {
    method: 'DELETE',
    headers: { 'Content-Type': 'application/json', ...getTenantHeaders() },
  });
  if (!response.ok) {
    throw new Error(`Failed to remove dependency: ${response.status}`);
  }
}

export async function getDependencyGraph(jobId: string): Promise<{
  nodes: Array<{ id: string; name: string; status?: JobStatus }>;
  edges: Array<{ source: string; target: string; type: string }>;
}> {
  const response = await fetch(buildUrl(`/api/scheduler/jobs/${jobId}/dependency-graph`), {
    headers: { 'Content-Type': 'application/json', ...getTenantHeaders() },
  });
  return handleResponse(response);
}

// ============================================================================
// Job Chains API
// ============================================================================

export async function listJobChains(page = 1, pageSize = 20): Promise<PaginatedResponse<JobChain>> {
  const response = await fetch(buildUrl('/api/scheduler/chains', { page, page_size: pageSize }), {
    headers: { 'Content-Type': 'application/json', ...getTenantHeaders() },
  });
  return handleResponse<PaginatedResponse<JobChain>>(response);
}

export async function getJobChain(chainId: string): Promise<JobChain> {
  const response = await fetch(buildUrl(`/api/scheduler/chains/${chainId}`), {
    headers: { 'Content-Type': 'application/json', ...getTenantHeaders() },
  });
  return handleResponse<JobChain>(response);
}

export async function createJobChain(chain: Partial<JobChain>): Promise<JobChain> {
  const response = await fetch(buildUrl('/api/scheduler/chains'), {
    method: 'POST',
    headers: { 'Content-Type': 'application/json', ...getTenantHeaders() },
    body: JSON.stringify(chain),
  });
  return handleResponse<JobChain>(response);
}

export async function updateJobChain(chainId: string, updates: Partial<JobChain>): Promise<JobChain> {
  const response = await fetch(buildUrl(`/api/scheduler/chains/${chainId}`), {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json', ...getTenantHeaders() },
    body: JSON.stringify(updates),
  });
  return handleResponse<JobChain>(response);
}

export async function deleteJobChain(chainId: string): Promise<void> {
  const response = await fetch(buildUrl(`/api/scheduler/chains/${chainId}`), {
    method: 'DELETE',
    headers: { 'Content-Type': 'application/json', ...getTenantHeaders() },
  });
  if (!response.ok) {
    throw new Error(`Failed to delete chain: ${response.status}`);
  }
}

export async function triggerJobChain(chainId: string): Promise<JobExecution[]> {
  const response = await fetch(buildUrl(`/api/scheduler/chains/${chainId}/trigger`), {
    method: 'POST',
    headers: { 'Content-Type': 'application/json', ...getTenantHeaders() },
  });
  return handleResponse<JobExecution[]>(response);
}

// ============================================================================
// Calendars API
// ============================================================================

export async function listCalendars(): Promise<BusinessCalendar[]> {
  const response = await fetch(buildUrl('/api/scheduler/calendars'), {
    headers: { 'Content-Type': 'application/json', ...getTenantHeaders() },
  });
  return handleResponse<BusinessCalendar[]>(response);
}

export async function getCalendar(calendarId: string): Promise<BusinessCalendar> {
  const response = await fetch(buildUrl(`/api/scheduler/calendars/${calendarId}`), {
    headers: { 'Content-Type': 'application/json', ...getTenantHeaders() },
  });
  return handleResponse<BusinessCalendar>(response);
}

export async function createCalendar(calendar: Partial<BusinessCalendar>): Promise<BusinessCalendar> {
  const response = await fetch(buildUrl('/api/scheduler/calendars'), {
    method: 'POST',
    headers: { 'Content-Type': 'application/json', ...getTenantHeaders() },
    body: JSON.stringify(calendar),
  });
  return handleResponse<BusinessCalendar>(response);
}

export async function updateCalendar(
  calendarId: string,
  updates: Partial<BusinessCalendar>
): Promise<BusinessCalendar> {
  const response = await fetch(buildUrl(`/api/scheduler/calendars/${calendarId}`), {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json', ...getTenantHeaders() },
    body: JSON.stringify(updates),
  });
  return handleResponse<BusinessCalendar>(response);
}

export async function deleteCalendar(calendarId: string): Promise<void> {
  const response = await fetch(buildUrl(`/api/scheduler/calendars/${calendarId}`), {
    method: 'DELETE',
    headers: { 'Content-Type': 'application/json', ...getTenantHeaders() },
  });
  if (!response.ok) {
    throw new Error(`Failed to delete calendar: ${response.status}`);
  }
}

export async function addHoliday(calendarId: string, holiday: Partial<Holiday>): Promise<Holiday> {
  const response = await fetch(buildUrl(`/api/scheduler/calendars/${calendarId}/holidays`), {
    method: 'POST',
    headers: { 'Content-Type': 'application/json', ...getTenantHeaders() },
    body: JSON.stringify(holiday),
  });
  return handleResponse<Holiday>(response);
}

export async function removeHoliday(calendarId: string, holidayId: string): Promise<void> {
  const response = await fetch(buildUrl(`/api/scheduler/calendars/${calendarId}/holidays/${holidayId}`), {
    method: 'DELETE',
    headers: { 'Content-Type': 'application/json', ...getTenantHeaders() },
  });
  if (!response.ok) {
    throw new Error(`Failed to remove holiday: ${response.status}`);
  }
}

export async function isBusinessDay(calendarId: string, date: string): Promise<boolean> {
  const response = await fetch(buildUrl(`/api/scheduler/calendars/${calendarId}/is-business-day`, { date }), {
    headers: { 'Content-Type': 'application/json', ...getTenantHeaders() },
  });
  const result = await handleResponse<{ is_business_day: boolean }>(response);
  return result.is_business_day;
}

export async function getNextBusinessDay(calendarId: string, date: string): Promise<string> {
  const response = await fetch(buildUrl(`/api/scheduler/calendars/${calendarId}/next-business-day`, { date }), {
    headers: { 'Content-Type': 'application/json', ...getTenantHeaders() },
  });
  const result = await handleResponse<{ date: string }>(response);
  return result.date;
}

// ============================================================================
// Notifications API
// ============================================================================

export async function listNotificationRules(jobId?: string): Promise<NotificationRule[]> {
  const params: Record<string, string | undefined> = {};
  if (jobId) params.job_id = jobId;

  const response = await fetch(buildUrl('/api/scheduler/notifications/rules', params), {
    headers: { 'Content-Type': 'application/json', ...getTenantHeaders() },
  });
  return handleResponse<NotificationRule[]>(response);
}

export async function createNotificationRule(rule: Partial<NotificationRule>): Promise<NotificationRule> {
  const response = await fetch(buildUrl('/api/scheduler/notifications/rules'), {
    method: 'POST',
    headers: { 'Content-Type': 'application/json', ...getTenantHeaders() },
    body: JSON.stringify(rule),
  });
  return handleResponse<NotificationRule>(response);
}

export async function updateNotificationRule(
  ruleId: string,
  updates: Partial<NotificationRule>
): Promise<NotificationRule> {
  const response = await fetch(buildUrl(`/api/scheduler/notifications/rules/${ruleId}`), {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json', ...getTenantHeaders() },
    body: JSON.stringify(updates),
  });
  return handleResponse<NotificationRule>(response);
}

export async function deleteNotificationRule(ruleId: string): Promise<void> {
  const response = await fetch(buildUrl(`/api/scheduler/notifications/rules/${ruleId}`), {
    method: 'DELETE',
    headers: { 'Content-Type': 'application/json', ...getTenantHeaders() },
  });
  if (!response.ok) {
    throw new Error(`Failed to delete notification rule: ${response.status}`);
  }
}

export async function listNotificationTemplates(): Promise<NotificationTemplate[]> {
  const response = await fetch(buildUrl('/api/scheduler/notifications/templates'), {
    headers: { 'Content-Type': 'application/json', ...getTenantHeaders() },
  });
  return handleResponse<NotificationTemplate[]>(response);
}

export async function getNotificationHistory(
  filters?: { job_id?: string; execution_id?: string; channel?: string },
  page = 1,
  pageSize = 50
): Promise<PaginatedResponse<NotificationHistory>> {
  const params: Record<string, string | number | undefined> = { page, page_size: pageSize };
  if (filters?.job_id) params.job_id = filters.job_id;
  if (filters?.execution_id) params.execution_id = filters.execution_id;
  if (filters?.channel) params.channel = filters.channel;

  const response = await fetch(buildUrl('/api/scheduler/notifications/history', params), {
    headers: { 'Content-Type': 'application/json', ...getTenantHeaders() },
  });
  return handleResponse<PaginatedResponse<NotificationHistory>>(response);
}

export async function testNotificationRule(ruleId: string): Promise<{ success: boolean; message?: string }> {
  const response = await fetch(buildUrl(`/api/scheduler/notifications/rules/${ruleId}/test`), {
    method: 'POST',
    headers: { 'Content-Type': 'application/json', ...getTenantHeaders() },
  });
  return handleResponse(response);
}

export async function getNotificationTemplate(id: string): Promise<NotificationTemplate> {
  const response = await fetch(buildUrl(`/api/scheduler/notifications/templates/${id}`), {
    headers: { 'Content-Type': 'application/json', ...getTenantHeaders() },
  });
  return handleResponse<NotificationTemplate>(response);
}

export async function createNotificationTemplate(template: Partial<NotificationTemplate>): Promise<NotificationTemplate> {
  const response = await fetch(buildUrl('/api/scheduler/notifications/templates'), {
    method: 'POST',
    headers: { 'Content-Type': 'application/json', ...getTenantHeaders() },
    body: JSON.stringify(template),
  });
  return handleResponse<NotificationTemplate>(response);
}

export async function updateNotificationTemplate(id: string, updates: Partial<NotificationTemplate>): Promise<NotificationTemplate> {
  const response = await fetch(buildUrl(`/api/scheduler/notifications/templates/${id}`), {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json', ...getTenantHeaders() },
    body: JSON.stringify(updates),
  });
  return handleResponse<NotificationTemplate>(response);
}

export async function deleteNotificationTemplate(id: string): Promise<void> {
  const response = await fetch(buildUrl(`/api/scheduler/notifications/templates/${id}`), {
    method: 'DELETE',
    headers: { 'Content-Type': 'application/json', ...getTenantHeaders() },
  });
  if (!response.ok) {
    throw new Error(`Failed to delete template: ${response.status}`);
  }
}

// ============================================================================
// Audit API
// ============================================================================

export async function listAuditLogs(
  filters?: {
    action?: string;
    resource_type?: string;
    resource_id?: string;
    user_id?: string;
    start_date?: string;
    end_date?: string;
  },
  page = 1,
  pageSize = 50
): Promise<PaginatedResponse<AuditLog>> {
  const params: Record<string, string | number | undefined> = { page, page_size: pageSize };
  if (filters) {
    Object.entries(filters).forEach(([key, value]) => {
      if (value) params[key] = value;
    });
  }

  const response = await fetch(buildUrl('/api/scheduler/audit', params), {
    headers: { 'Content-Type': 'application/json', ...getTenantHeaders() },
  });
  return handleResponse<PaginatedResponse<AuditLog>>(response);
}

export async function getAuditLog(auditId: string): Promise<AuditLog> {
  const response = await fetch(buildUrl(`/api/scheduler/audit/${auditId}`), {
    headers: { 'Content-Type': 'application/json', ...getTenantHeaders() },
  });
  return handleResponse<AuditLog>(response);
}

export const getAuditLogs = listAuditLogs;

// ============================================================================
// Compliance Reports API
// ============================================================================

export async function generateComplianceReport(
  reportType: string,
  startDate: string,
  endDate: string,
  filters?: { job_ids?: string[]; job_types?: string[]; statuses?: JobStatus[] }
): Promise<ComplianceReport> {
  const response = await fetch(buildUrl('/api/scheduler/compliance/reports'), {
    method: 'POST',
    headers: { 'Content-Type': 'application/json', ...getTenantHeaders() },
    body: JSON.stringify({
      report_type: reportType,
      start_date: startDate,
      end_date: endDate,
      ...filters,
    }),
  });
  return handleResponse<ComplianceReport>(response);
}

export async function listComplianceReports(page = 1, pageSize = 20): Promise<PaginatedResponse<ComplianceReport>> {
  const response = await fetch(buildUrl('/api/scheduler/compliance/reports', { page, page_size: pageSize }), {
    headers: { 'Content-Type': 'application/json', ...getTenantHeaders() },
  });
  return handleResponse<PaginatedResponse<ComplianceReport>>(response);
}

export async function getComplianceReport(reportId: string): Promise<ComplianceReport> {
  const response = await fetch(buildUrl(`/api/scheduler/compliance/reports/${reportId}`), {
    headers: { 'Content-Type': 'application/json', ...getTenantHeaders() },
  });
  return handleResponse<ComplianceReport>(response);
}

export async function exportComplianceReport(reportId: string, format: 'csv' | 'pdf' | 'excel'): Promise<Blob> {
  const response = await fetch(buildUrl(`/api/scheduler/compliance/reports/${reportId}/export`, { format }), {
    headers: { ...getTenantHeaders() },
  });
  if (!response.ok) {
    throw new Error(`Failed to export report: ${response.status}`);
  }
  return response.blob();
}

// ============================================================================
// Dashboard API
// ============================================================================

export async function getDashboardMetrics(): Promise<SchedulerDashboardMetrics> {
  const response = await fetch(buildUrl('/api/scheduler/dashboard/metrics'), {
    headers: { 'Content-Type': 'application/json', ...getTenantHeaders() },
  });
  return handleResponse<SchedulerDashboardMetrics>(response);
}

export async function getExecutionTimeline(
  startDate: string,
  endDate: string,
  jobIds?: string[]
): Promise<Array<{ date: string; successful: number; failed: number; total: number }>> {
  const params: Record<string, string | undefined> = {
    start_date: startDate,
    end_date: endDate,
  };
  if (jobIds?.length) params.job_ids = jobIds.join(',');

  const response = await fetch(buildUrl('/api/scheduler/dashboard/timeline', params), {
    headers: { 'Content-Type': 'application/json', ...getTenantHeaders() },
  });
  return handleResponse(response);
}

export async function getJobTypeDistribution(): Promise<Array<{ type: string; count: number }>> {
  const response = await fetch(buildUrl('/api/scheduler/dashboard/distribution'), {
    headers: { 'Content-Type': 'application/json', ...getTenantHeaders() },
  });
  return handleResponse(response);
}

// ============================================================================
// WebSocket for Real-time Updates
// ============================================================================

export function subscribeToExecutionUpdates(
  executionId: string,
  onUpdate: (execution: JobExecution) => void,
  onLog: (log: ExecutionLog) => void,
  onError: (error: Error) => void
): () => void {
  const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
  const tenantParams = getTenantQueryParams();
  const wsUrl = `${protocol}//${window.location.host}/api/scheduler/executions/${executionId}/ws?${tenantParams}`;

  const ws = new WebSocket(wsUrl);

  ws.onmessage = (event) => {
    try {
      const message = JSON.parse(event.data);
      if (message.type === 'execution_update') {
        onUpdate(message.data);
      } else if (message.type === 'log') {
        onLog(message.data);
      }
    } catch (err) {
      console.error('Failed to parse WebSocket message:', err);
    }
  };

  ws.onerror = () => {
    onError(new Error('WebSocket connection error'));
  };

  ws.onclose = () => {
    // WebSocket connection closed - cleanup handled by return function
  };

  // Return cleanup function
  return () => {
    if (ws.readyState === WebSocket.OPEN) {
      ws.close();
    }
  };
}

export function subscribeToJobUpdates(
  onUpdate: (job: Job) => void,
  onExecution: (execution: JobExecution) => void
): () => void {
  const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
  const tenantParams = getTenantQueryParams();
  const wsUrl = `${protocol}//${window.location.host}/api/scheduler/ws?${tenantParams}`;

  const ws = new WebSocket(wsUrl);

  ws.onmessage = (event) => {
    try {
      const message = JSON.parse(event.data);
      if (message.type === 'job_update') {
        onUpdate(message.data);
      } else if (message.type === 'execution_update') {
        onExecution(message.data);
      }
    } catch (err) {
      console.error('Failed to parse WebSocket message:', err);
    }
  };

  return () => {
    if (ws.readyState === WebSocket.OPEN) {
      ws.close();
    }
  };
}
