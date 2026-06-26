/**
 * World-Class Enterprise Scheduler - Job Editor Page
 * Create and edit jobs with comprehensive scheduling, dependencies, and notification options
 */

import React, { useState, useEffect, useCallback } from 'react';
import { useTranslation } from 'react-i18next';
import { useNavigate, useParams } from 'react-router-dom';
import * as schedulerService from '../services/schedulerService';
import {
  Job,
  JobPriority,
  ScheduleType,
  RecurrencePattern,
  DayOfWeek,
  DependencyType,
  NotificationTrigger,
  NotificationChannel,
  CreateJobRequest,
  BusinessCalendar,
} from '../../../types/scheduler';
import '../styles/SchedulerDashboard.css';

// ============================================================================
// Job Editor Page
// ============================================================================

export function JobEditorPage() {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const { jobId } = useParams<{ jobId: string }>();
  const isEditing = Boolean(jobId);
  
  const [loading, setLoading] = useState(isEditing);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [activeTab, setActiveTab] = useState<'basic' | 'schedule' | 'dependencies' | 'notifications' | 'advanced'>('basic');
  
  // Job data
  const [name, setName] = useState('');
  const [description, setDescription] = useState('');
  const [jobType, setJobType] = useState('');
  const [jobCategory, setJobCategory] = useState('');
  const [priority, setPriority] = useState<JobPriority>(JobPriority.MEDIUM);
  const [enabled, setEnabled] = useState(true);
  const [payload, setPayload] = useState('{}');
  const [tags, setTags] = useState<string[]>([]);
  const [tagInput, setTagInput] = useState('');
  
  // Schedule data
  const [scheduleType, setScheduleType] = useState<ScheduleType>(ScheduleType.ONCE);
  const [runAt, setRunAt] = useState('');
  const [cronExpression, setCronExpression] = useState('');
  const [cronTimezone, setCronTimezone] = useState('UTC');
  const [recurrencePattern, setRecurrencePattern] = useState<RecurrencePattern>(RecurrencePattern.DAILY);
  const [recurrenceInterval, setRecurrenceInterval] = useState(1);
  const [daysOfWeek, setDaysOfWeek] = useState<DayOfWeek[]>([]);
  const [dayOfMonth, setDayOfMonth] = useState(1);
  const [timeOfDay, setTimeOfDay] = useState('09:00');
  const [validFrom, setValidFrom] = useState('');
  const [validUntil, setValidUntil] = useState('');
  const [skipHolidays, setSkipHolidays] = useState(false);
  const [skipWeekends, setSkipWeekends] = useState(false);
  const [selectedCalendarId, setSelectedCalendarId] = useState('');
  
  // Advanced settings
  const [timeoutSeconds, setTimeoutSeconds] = useState(3600);
  const [maxRetries, setMaxRetries] = useState(3);
  const [retryDelaySeconds, setRetryDelaySeconds] = useState(60);
  const [slaDeadlineMinutes, setSlaDeadlineMinutes] = useState<number | undefined>();
  const [requiresApproval, setRequiresApproval] = useState(false);
  
  // Available calendars
  const [calendars, setCalendars] = useState<BusinessCalendar[]>([]);
  
  // Load existing job if editing
  useEffect(() => {
    const loadJob = async () => {
      if (!jobId) return;
      
      try {
        setLoading(true);
        const job = await schedulerService.getJob(jobId);
        
        // Populate form fields
        setName(job.name);
        setDescription(job.description || '');
        setJobType(job.job_type);
        setJobCategory(job.job_category || '');
        setPriority(job.priority);
        setEnabled(job.enabled);
        setPayload(JSON.stringify(job.payload, null, 2));
        setTags(job.tags || []);
        
        // Schedule
        if (job.schedule) {
          setScheduleType(job.schedule.schedule_type);
          setRunAt(job.schedule.run_at || '');
          setCronExpression(job.schedule.cron_expression || '');
          setCronTimezone(job.schedule.cron_timezone || 'UTC');
          setValidFrom(job.schedule.valid_from || '');
          setValidUntil(job.schedule.valid_until || '');
          setSkipHolidays(job.schedule.skip_holidays || false);
          setSkipWeekends(job.schedule.skip_weekends || false);
          setSelectedCalendarId(job.schedule.calendar_id || '');
          
          if (job.schedule.recurrence) {
            setRecurrencePattern(job.schedule.recurrence.pattern);
            setRecurrenceInterval(job.schedule.recurrence.interval);
            setDaysOfWeek(job.schedule.recurrence.days_of_week || []);
            setDayOfMonth(job.schedule.recurrence.day_of_month || 1);
            setTimeOfDay(job.schedule.recurrence.time_of_day || '09:00');
          }
        }
        
        // Advanced
        setTimeoutSeconds(job.timeout_seconds || 3600);
        setMaxRetries(job.max_retries);
        setRetryDelaySeconds(job.retry_delay_seconds);
        setSlaDeadlineMinutes(job.sla_deadline_minutes);
        setRequiresApproval(job.requires_approval || false);
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Failed to load job');
      } finally {
        setLoading(false);
      }
    };
    
    loadJob();
  }, [jobId]);
  
  // Load calendars
  useEffect(() => {
    const loadCalendars = async () => {
      try {
        const cals = await schedulerService.listCalendars();
        setCalendars(cals);
      } catch (err) {
        console.error('Failed to load calendars:', err);
      }
    };
    loadCalendars();
  }, []);
  
  // Handle save
  const handleSave = async () => {
    try {
      setSaving(true);
      setError(null);
      
      // Validate
      if (!name.trim()) {
        setError(t('scheduler.errors.nameRequired', 'Job name is required'));
        return;
      }
      if (!jobType.trim()) {
        setError(t('scheduler.errors.typeRequired', 'Job type is required'));
        return;
      }
      
      let parsedPayload = {};
      try {
        parsedPayload = JSON.parse(payload);
      } catch {
        setError(t('scheduler.errors.invalidPayload', 'Invalid JSON payload'));
        return;
      }
      
      const request: CreateJobRequest = {
        name: name.trim(),
        description: description.trim() || undefined,
        job_type: jobType.trim(),
        job_category: jobCategory.trim() || undefined,
        priority,
        payload: parsedPayload,
        tags: tags.length > 0 ? tags : undefined,
        timeout_seconds: timeoutSeconds,
        max_retries: maxRetries,
        retry_delay_seconds: retryDelaySeconds,
        sla_deadline_minutes: slaDeadlineMinutes,
        requires_approval: requiresApproval,
      };
      
      // Build schedule
      if (scheduleType !== ScheduleType.EVENT_DRIVEN) {
        const schedule: any = {
          schedule_type: scheduleType,
          skip_holidays: skipHolidays,
          skip_weekends: skipWeekends,
        };
        
        if (validFrom) schedule.valid_from = validFrom;
        if (validUntil) schedule.valid_until = validUntil;
        if (selectedCalendarId) schedule.calendar_id = selectedCalendarId;
        
        if (scheduleType === ScheduleType.ONCE && runAt) {
          schedule.run_at = runAt;
        }
        
        if (scheduleType === ScheduleType.CRON && cronExpression) {
          schedule.cron_expression = cronExpression;
          schedule.cron_timezone = cronTimezone;
        }
        
        if (scheduleType === ScheduleType.RECURRING) {
          schedule.recurrence = {
            pattern: recurrencePattern,
            interval: recurrenceInterval,
            time_of_day: timeOfDay,
            timezone: cronTimezone,
          };
          
          if (recurrencePattern === RecurrencePattern.WEEKLY && daysOfWeek.length > 0) {
            schedule.recurrence.days_of_week = daysOfWeek;
          }
          
          if (recurrencePattern === RecurrencePattern.MONTHLY) {
            schedule.recurrence.day_of_month = dayOfMonth;
          }
        }
        
        request.schedule = schedule;
      }
      
      if (isEditing && jobId) {
        await schedulerService.updateJob(jobId, { ...request, enabled });
      } else {
        await schedulerService.createJob(request);
      }
      
      navigate('/scheduler/jobs');
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to save job');
    } finally {
      setSaving(false);
    }
  };
  
  // Handle tag addition
  const handleAddTag = () => {
    const tag = tagInput.trim();
    if (tag && !tags.includes(tag)) {
      setTags([...tags, tag]);
      setTagInput('');
    }
  };
  
  const handleRemoveTag = (tag: string) => {
    setTags(tags.filter(t => t !== tag));
  };
  
  // Toggle day of week
  const toggleDayOfWeek = (day: DayOfWeek) => {
    if (daysOfWeek.includes(day)) {
      setDaysOfWeek(daysOfWeek.filter(d => d !== day));
    } else {
      setDaysOfWeek([...daysOfWeek, day]);
    }
  };
  
  if (loading) {
    return (
      <div className="scheduler-dashboard">
        <div className="loading-spinner">
          <div className="spinner" />
        </div>
      </div>
    );
  }
  
  return (
    <div className="scheduler-dashboard">
      {/* Header */}
      <div className="scheduler-header">
        <h1>
          {isEditing
            ? t('scheduler.editJob', 'Edit Job')
            : t('scheduler.createJob', 'Create Job')}
        </h1>
        <div className="scheduler-header-actions">
          <button className="btn btn-secondary" onClick={() => navigate('/scheduler/jobs')}>
            {t('scheduler.cancel', 'Cancel')}
          </button>
          <button className="btn btn-primary" onClick={handleSave} disabled={saving}>
            {saving ? t('scheduler.saving', 'Saving...') : t('scheduler.save', 'Save')}
          </button>
        </div>
      </div>
      
      {/* Error */}
      {error && (
        <div className="error-banner" style={{
          background: '#fef2f2',
          border: '1px solid #fecaca',
          borderRadius: 8,
          padding: 12,
          marginBottom: 16,
          color: '#991b1b',
        }}>
          ⚠️ {error}
        </div>
      )}
      
      {/* Tabs */}
      <div className="tabs">
        {(['basic', 'schedule', 'dependencies', 'notifications', 'advanced'] as const).map(tab => (
          <button
            key={tab}
            className={`tab ${activeTab === tab ? 'active' : ''}`}
            onClick={() => setActiveTab(tab)}
          >
            {t(`scheduler.tabs.${tab}`, tab.charAt(0).toUpperCase() + tab.slice(1))}
          </button>
        ))}
      </div>
      
      {/* Tab Content */}
      <div className="dashboard-card">
        <div className="card-content">
          {/* Basic Tab */}
          {activeTab === 'basic' && (
            <BasicInfoTab
              name={name}
              setName={setName}
              description={description}
              setDescription={setDescription}
              jobType={jobType}
              setJobType={setJobType}
              jobCategory={jobCategory}
              setJobCategory={setJobCategory}
              priority={priority}
              setPriority={setPriority}
              enabled={enabled}
              setEnabled={setEnabled}
              payload={payload}
              setPayload={setPayload}
              tags={tags}
              tagInput={tagInput}
              setTagInput={setTagInput}
              handleAddTag={handleAddTag}
              handleRemoveTag={handleRemoveTag}
              t={t}
            />
          )}
          
          {/* Schedule Tab */}
          {activeTab === 'schedule' && (
            <ScheduleTab
              scheduleType={scheduleType}
              setScheduleType={setScheduleType}
              runAt={runAt}
              setRunAt={setRunAt}
              cronExpression={cronExpression}
              setCronExpression={setCronExpression}
              cronTimezone={cronTimezone}
              setCronTimezone={setCronTimezone}
              recurrencePattern={recurrencePattern}
              setRecurrencePattern={setRecurrencePattern}
              recurrenceInterval={recurrenceInterval}
              setRecurrenceInterval={setRecurrenceInterval}
              daysOfWeek={daysOfWeek}
              toggleDayOfWeek={toggleDayOfWeek}
              dayOfMonth={dayOfMonth}
              setDayOfMonth={setDayOfMonth}
              timeOfDay={timeOfDay}
              setTimeOfDay={setTimeOfDay}
              validFrom={validFrom}
              setValidFrom={setValidFrom}
              validUntil={validUntil}
              setValidUntil={setValidUntil}
              skipHolidays={skipHolidays}
              setSkipHolidays={setSkipHolidays}
              skipWeekends={skipWeekends}
              setSkipWeekends={setSkipWeekends}
              calendars={calendars}
              selectedCalendarId={selectedCalendarId}
              setSelectedCalendarId={setSelectedCalendarId}
              t={t}
            />
          )}
          
          {/* Dependencies Tab */}
          {activeTab === 'dependencies' && (
            <DependenciesTab jobId={jobId} t={t} />
          )}
          
          {/* Notifications Tab */}
          {activeTab === 'notifications' && (
            <NotificationsTab jobId={jobId} t={t} />
          )}
          
          {/* Advanced Tab */}
          {activeTab === 'advanced' && (
            <AdvancedTab
              timeoutSeconds={timeoutSeconds}
              setTimeoutSeconds={setTimeoutSeconds}
              maxRetries={maxRetries}
              setMaxRetries={setMaxRetries}
              retryDelaySeconds={retryDelaySeconds}
              setRetryDelaySeconds={setRetryDelaySeconds}
              slaDeadlineMinutes={slaDeadlineMinutes}
              setSlaDeadlineMinutes={setSlaDeadlineMinutes}
              requiresApproval={requiresApproval}
              setRequiresApproval={setRequiresApproval}
              t={t}
            />
          )}
        </div>
      </div>
    </div>
  );
}

// ============================================================================
// Basic Info Tab
// ============================================================================

interface BasicInfoTabProps {
  name: string;
  setName: (v: string) => void;
  description: string;
  setDescription: (v: string) => void;
  jobType: string;
  setJobType: (v: string) => void;
  jobCategory: string;
  setJobCategory: (v: string) => void;
  priority: JobPriority;
  setPriority: (v: JobPriority) => void;
  enabled: boolean;
  setEnabled: (v: boolean) => void;
  payload: string;
  setPayload: (v: string) => void;
  tags: string[];
  tagInput: string;
  setTagInput: (v: string) => void;
  handleAddTag: () => void;
  handleRemoveTag: (tag: string) => void;
  t: any;
}

function BasicInfoTab(props: BasicInfoTabProps) {
  const { t } = props;
  
  const jobTypes = [
    { value: 'etl', label: t('scheduler.jobTypes.etl', 'ETL / Data Pipeline') },
    { value: 'report', label: t('scheduler.jobTypes.report', 'Report Generation') },
    { value: 'export', label: t('scheduler.jobTypes.export', 'Data Export') },
    { value: 'import', label: t('scheduler.jobTypes.import', 'Data Import') },
    { value: 'notification', label: t('scheduler.jobTypes.notification', 'Notification') },
    { value: 'cleanup', label: t('scheduler.jobTypes.cleanup', 'Cleanup / Maintenance') },
    { value: 'backup', label: t('scheduler.jobTypes.backup', 'Backup') },
    { value: 'sync', label: t('scheduler.jobTypes.sync', 'Data Sync') },
    { value: 'calculation', label: t('scheduler.jobTypes.calculation', 'Calculation') },
    { value: 'validation', label: t('scheduler.jobTypes.validation', 'Validation') },
    { value: 'workflow', label: t('scheduler.jobTypes.workflow', 'Business Workflow') },
    { value: 'custom', label: t('scheduler.jobTypes.custom', 'Custom') },
  ];
  
  return (
    <div className="form-grid">
      <FormField label={t('scheduler.fields.name', 'Job Name')} required>
        <input
          type="text"
          className="form-input"
          value={props.name}
          onChange={e => props.setName(e.target.value)}
          placeholder={t('scheduler.placeholders.name', 'Enter job name')}
        />
      </FormField>
      
      <FormField label={t('scheduler.fields.description', 'Description')}>
        <textarea
          className="form-input"
          value={props.description}
          onChange={e => props.setDescription(e.target.value)}
          placeholder={t('scheduler.placeholders.description', 'Describe what this job does')}
          rows={3}
        />
      </FormField>
      
      <FormField label={t('scheduler.fields.type', 'Job Type')} required>
        <select
          className="form-input"
          value={props.jobType}
          onChange={e => props.setJobType(e.target.value)}
          title={t('scheduler.fields.type', 'Job Type')}
        >
          <option value="">{t('scheduler.select', 'Select...')}</option>
          {jobTypes.map(type => (
            <option key={type.value} value={type.value}>{type.label}</option>
          ))}
        </select>
      </FormField>
      
      <FormField label={t('scheduler.fields.category', 'Category')}>
        <input
          type="text"
          className="form-input"
          value={props.jobCategory}
          onChange={e => props.setJobCategory(e.target.value)}
          placeholder={t('scheduler.placeholders.category', 'e.g., Finance, HR, Operations')}
        />
      </FormField>
      
      <FormField label={t('scheduler.fields.priority', 'Priority')}>
        <select
          className="form-input"
          value={props.priority}
          onChange={e => props.setPriority(e.target.value as JobPriority)}
          title={t('scheduler.fields.priority', 'Priority')}
        >
          <option value={JobPriority.CRITICAL}>{t('scheduler.priority.critical', 'Critical')}</option>
          <option value={JobPriority.HIGH}>{t('scheduler.priority.high', 'High')}</option>
          <option value={JobPriority.MEDIUM}>{t('scheduler.priority.medium', 'Medium')}</option>
          <option value={JobPriority.LOW}>{t('scheduler.priority.low', 'Low')}</option>
        </select>
      </FormField>
      
      <FormField label={t('scheduler.fields.enabled', 'Status')}>
        <label className="toggle-label">
          <input
            type="checkbox"
            checked={props.enabled}
            onChange={e => props.setEnabled(e.target.checked)}
          />
          <span>{props.enabled ? t('scheduler.enabled', 'Enabled') : t('scheduler.disabled', 'Disabled')}</span>
        </label>
      </FormField>
      
      <FormField label={t('scheduler.fields.tags', 'Tags')} fullWidth>
        <div className="tags-input-container">
          <div className="tags-list">
            {props.tags.map(tag => (
              <span key={tag} className="tag">
                {tag}
                <button onClick={() => props.handleRemoveTag(tag)}>×</button>
              </span>
            ))}
          </div>
          <div className="tag-input-row">
            <input
              type="text"
              className="form-input"
              value={props.tagInput}
              onChange={e => props.setTagInput(e.target.value)}
              onKeyPress={e => e.key === 'Enter' && (e.preventDefault(), props.handleAddTag())}
              placeholder={t('scheduler.placeholders.tag', 'Add a tag...')}
            />
            <button className="btn btn-secondary" onClick={props.handleAddTag}>
              {t('scheduler.add', 'Add')}
            </button>
          </div>
        </div>
      </FormField>
      
      <FormField label={t('scheduler.fields.payload', 'Job Payload (JSON)')} fullWidth>
        <textarea
          className="form-input code-input"
          value={props.payload}
          onChange={e => props.setPayload(e.target.value)}
          rows={8}
          style={{ fontFamily: 'monospace' }}
        />
      </FormField>
    </div>
  );
}

// ============================================================================
// Schedule Tab
// ============================================================================

interface ScheduleTabProps {
  scheduleType: ScheduleType;
  setScheduleType: (v: ScheduleType) => void;
  runAt: string;
  setRunAt: (v: string) => void;
  cronExpression: string;
  setCronExpression: (v: string) => void;
  cronTimezone: string;
  setCronTimezone: (v: string) => void;
  recurrencePattern: RecurrencePattern;
  setRecurrencePattern: (v: RecurrencePattern) => void;
  recurrenceInterval: number;
  setRecurrenceInterval: (v: number) => void;
  daysOfWeek: DayOfWeek[];
  toggleDayOfWeek: (day: DayOfWeek) => void;
  dayOfMonth: number;
  setDayOfMonth: (v: number) => void;
  timeOfDay: string;
  setTimeOfDay: (v: string) => void;
  validFrom: string;
  setValidFrom: (v: string) => void;
  validUntil: string;
  setValidUntil: (v: string) => void;
  skipHolidays: boolean;
  setSkipHolidays: (v: boolean) => void;
  skipWeekends: boolean;
  setSkipWeekends: (v: boolean) => void;
  calendars: BusinessCalendar[];
  selectedCalendarId: string;
  setSelectedCalendarId: (v: string) => void;
  t: any;
}

function ScheduleTab(props: ScheduleTabProps) {
  const { t } = props;
  
  const allDays: DayOfWeek[] = [
    DayOfWeek.MONDAY,
    DayOfWeek.TUESDAY,
    DayOfWeek.WEDNESDAY,
    DayOfWeek.THURSDAY,
    DayOfWeek.FRIDAY,
    DayOfWeek.SATURDAY,
    DayOfWeek.SUNDAY,
  ];
  
  const timezones = [
    'UTC',
    'America/New_York',
    'America/Chicago',
    'America/Denver',
    'America/Los_Angeles',
    'Europe/London',
    'Europe/Paris',
    'Europe/Berlin',
    'Asia/Tokyo',
    'Asia/Shanghai',
    'Asia/Singapore',
    'Australia/Sydney',
  ];
  
  return (
    <div className="form-grid">
      <FormField label={t('scheduler.fields.scheduleType', 'Schedule Type')} required>
        <select
          className="form-input"
          value={props.scheduleType}
          onChange={e => props.setScheduleType(e.target.value as ScheduleType)}
          title={t('scheduler.fields.scheduleType', 'Schedule Type')}
        >
          <option value={ScheduleType.ONCE}>{t('scheduler.scheduleTypes.once', 'Run Once')}</option>
          <option value={ScheduleType.RECURRING}>{t('scheduler.scheduleTypes.recurring', 'Recurring')}</option>
          <option value={ScheduleType.CRON}>{t('scheduler.scheduleTypes.cron', 'Cron Expression')}</option>
          <option value={ScheduleType.CALENDAR_BASED}>{t('scheduler.scheduleTypes.calendar', 'Calendar Based')}</option>
          <option value={ScheduleType.EVENT_DRIVEN}>{t('scheduler.scheduleTypes.event', 'Event Driven')}</option>
        </select>
      </FormField>
      
      {/* One-time schedule */}
      {props.scheduleType === ScheduleType.ONCE && (
        <FormField label={t('scheduler.fields.runAt', 'Run At')}>
          <input
            type="datetime-local"
            className="form-input"
            value={props.runAt}
            onChange={e => props.setRunAt(e.target.value)}
          />
        </FormField>
      )}
      
      {/* Cron schedule */}
      {props.scheduleType === ScheduleType.CRON && (
        <>
          <FormField label={t('scheduler.fields.cronExpression', 'Cron Expression')}>
            <input
              type="text"
              className="form-input"
              value={props.cronExpression}
              onChange={e => props.setCronExpression(e.target.value)}
              placeholder="0 9 * * MON-FRI"
            />
            <small className="form-help">
              {t('scheduler.cronHelp', 'Format: minute hour day month weekday (e.g., 0 9 * * MON-FRI for 9 AM on weekdays)')}
            </small>
          </FormField>
          
          <FormField label={t('scheduler.fields.timezone', 'Timezone')}>
            <select
              className="form-input"
              value={props.cronTimezone}
              onChange={e => props.setCronTimezone(e.target.value)}
              title={t('scheduler.fields.timezone', 'Timezone')}
            >
              {timezones.map(tz => (
                <option key={tz} value={tz}>{tz}</option>
              ))}
            </select>
          </FormField>
        </>
      )}
      
      {/* Recurring schedule */}
      {props.scheduleType === ScheduleType.RECURRING && (
        <>
          <FormField label={t('scheduler.fields.pattern', 'Recurrence Pattern')}>
            <select
              className="form-input"
              value={props.recurrencePattern}
              onChange={e => props.setRecurrencePattern(e.target.value as RecurrencePattern)}
              title={t('scheduler.fields.pattern', 'Recurrence Pattern')}
            >
              <option value={RecurrencePattern.DAILY}>{t('scheduler.patterns.daily', 'Daily')}</option>
              <option value={RecurrencePattern.WEEKLY}>{t('scheduler.patterns.weekly', 'Weekly')}</option>
              <option value={RecurrencePattern.BIWEEKLY}>{t('scheduler.patterns.biweekly', 'Bi-weekly')}</option>
              <option value={RecurrencePattern.MONTHLY}>{t('scheduler.patterns.monthly', 'Monthly')}</option>
              <option value={RecurrencePattern.QUARTERLY}>{t('scheduler.patterns.quarterly', 'Quarterly')}</option>
              <option value={RecurrencePattern.YEARLY}>{t('scheduler.patterns.yearly', 'Yearly')}</option>
            </select>
          </FormField>
          
          <FormField label={t('scheduler.fields.interval', 'Interval')}>
            <div className="input-with-addon">
              <span>{t('scheduler.every', 'Every')}</span>
              <input
                type="number"
                className="form-input"
                value={props.recurrenceInterval}
                onChange={e => props.setRecurrenceInterval(parseInt(e.target.value) || 1)}
                min={1}
                max={100}
                style={{ width: 80 }}
              />
              <span>{props.recurrencePattern}</span>
            </div>
          </FormField>
          
          {props.recurrencePattern === RecurrencePattern.WEEKLY && (
            <FormField label={t('scheduler.fields.daysOfWeek', 'Days of Week')} fullWidth>
              <div className="days-picker">
                {allDays.map(day => (
                  <button
                    key={day}
                    type="button"
                    className={`day-btn ${props.daysOfWeek.includes(day) ? 'active' : ''}`}
                    onClick={() => props.toggleDayOfWeek(day)}
                  >
                    {day.substring(0, 3)}
                  </button>
                ))}
              </div>
            </FormField>
          )}
          
          {props.recurrencePattern === RecurrencePattern.MONTHLY && (
            <FormField label={t('scheduler.fields.dayOfMonth', 'Day of Month')}>
              <input
                type="number"
                className="form-input"
                value={props.dayOfMonth}
                onChange={e => props.setDayOfMonth(parseInt(e.target.value) || 1)}
                min={1}
                max={31}
              />
            </FormField>
          )}
          
          <FormField label={t('scheduler.fields.timeOfDay', 'Time of Day')}>
            <input
              type="time"
              className="form-input"
              value={props.timeOfDay}
              onChange={e => props.setTimeOfDay(e.target.value)}
            />
          </FormField>
          
          <FormField label={t('scheduler.fields.timezone', 'Timezone')}>
            <select
              className="form-input"
              value={props.cronTimezone}
              onChange={e => props.setCronTimezone(e.target.value)}
              title={t('scheduler.fields.timezone', 'Timezone')}
            >
              {timezones.map(tz => (
                <option key={tz} value={tz}>{tz}</option>
              ))}
            </select>
          </FormField>
        </>
      )}
      
      {/* Calendar based */}
      {props.scheduleType === ScheduleType.CALENDAR_BASED && (
        <FormField label={t('scheduler.fields.calendar', 'Business Calendar')}>
          <select
            className="form-input"
            value={props.selectedCalendarId}
            onChange={e => props.setSelectedCalendarId(e.target.value)}
            title={t('scheduler.fields.calendar', 'Business Calendar')}
          >
            <option value="">{t('scheduler.select', 'Select a calendar...')}</option>
            {props.calendars.map(cal => (
              <option key={cal.id} value={cal.id}>{cal.name}</option>
            ))}
          </select>
        </FormField>
      )}
      
      {/* Validity period */}
      <FormField label={t('scheduler.fields.validFrom', 'Valid From')}>
        <input
          type="date"
          className="form-input"
          value={props.validFrom}
          onChange={e => props.setValidFrom(e.target.value)}
        />
      </FormField>
      
      <FormField label={t('scheduler.fields.validUntil', 'Valid Until')}>
        <input
          type="date"
          className="form-input"
          value={props.validUntil}
          onChange={e => props.setValidUntil(e.target.value)}
        />
      </FormField>
      
      {/* Skip options */}
      <FormField label={t('scheduler.fields.skipOptions', 'Skip Options')} fullWidth>
        <div className="checkbox-group">
          <label className="checkbox-label">
            <input
              type="checkbox"
              checked={props.skipHolidays}
              onChange={e => props.setSkipHolidays(e.target.checked)}
            />
            {t('scheduler.skipHolidays', 'Skip Holidays')}
          </label>
          <label className="checkbox-label">
            <input
              type="checkbox"
              checked={props.skipWeekends}
              onChange={e => props.setSkipWeekends(e.target.checked)}
            />
            {t('scheduler.skipWeekends', 'Skip Weekends')}
          </label>
        </div>
      </FormField>
    </div>
  );
}

// ============================================================================
// Dependencies Tab (Stub)
// ============================================================================

function DependenciesTab({ jobId, t }: { jobId?: string; t: any }) {
  return (
    <div className="dependencies-tab">
      <p className="tab-description">
        {t('scheduler.dependencies.description', 'Configure job dependencies to create complex workflows with finish-to-start, start-to-start, and other dependency types.')}
      </p>
      
      {!jobId ? (
        <div className="empty-state">
          <div className="empty-state-icon">🔗</div>
          <div className="empty-state-text">
            {t('scheduler.dependencies.saveFirst', 'Save the job first to add dependencies')}
          </div>
        </div>
      ) : (
        <DependencyEditor jobId={jobId} t={t} />
      )}
    </div>
  );
}

function DependencyEditor({ jobId, t }: { jobId: string; t: any }) {
  const [dependencies, setDependencies] = useState<any[]>([]);
  const [loading, setLoading] = useState(true);
  const [availableJobs, setAvailableJobs] = useState<Job[]>([]);
  
  useEffect(() => {
    const load = async () => {
      try {
        const [deps, jobs] = await Promise.all([
          schedulerService.getJobDependencies(jobId),
          schedulerService.listJobs({}, 1, 100),
        ]);
        setDependencies(deps);
        setAvailableJobs(jobs.data.filter(j => j.id !== jobId));
      } catch (err) {
        console.error('Failed to load dependencies:', err);
      } finally {
        setLoading(false);
      }
    };
    load();
  }, [jobId]);
  
  if (loading) {
    return <div className="loading-spinner"><div className="spinner" /></div>;
  }
  
  return (
    <div>
      <h4>{t('scheduler.dependencies.current', 'Current Dependencies')}</h4>
      {dependencies.length === 0 ? (
        <p>{t('scheduler.dependencies.none', 'No dependencies configured')}</p>
      ) : (
        <ul className="dependencies-list">
          {dependencies.map(dep => (
            <li key={dep.id}>
              {dep.depends_on_job_name} ({dep.dependency_type})
            </li>
          ))}
        </ul>
      )}
      
      <h4>{t('scheduler.dependencies.add', 'Add Dependency')}</h4>
      <p>{t('scheduler.dependencies.selectJob', 'Select a job to depend on:')}</p>
      {/* Add dependency form would go here */}
    </div>
  );
}

// ============================================================================
// Notifications Tab (Stub)
// ============================================================================

function NotificationsTab({ jobId, t }: { jobId?: string; t: any }) {
  return (
    <div className="notifications-tab">
      <p className="tab-description">
        {t('scheduler.notifications.description', 'Configure multi-channel notifications for job events like start, completion, failure, and SLA breaches.')}
      </p>
      
      {!jobId ? (
        <div className="empty-state">
          <div className="empty-state-icon">🔔</div>
          <div className="empty-state-text">
            {t('scheduler.notifications.saveFirst', 'Save the job first to add notifications')}
          </div>
        </div>
      ) : (
        <div>
          <h4>{t('scheduler.notifications.channels', 'Notification Channels')}</h4>
          <p>{t('scheduler.notifications.channelsHelp', 'Send notifications via Email, SMS, Slack, Teams, or Webhooks')}</p>
          {/* Notification configuration would go here */}
        </div>
      )}
    </div>
  );
}

// ============================================================================
// Advanced Tab
// ============================================================================

interface AdvancedTabProps {
  timeoutSeconds: number;
  setTimeoutSeconds: (v: number) => void;
  maxRetries: number;
  setMaxRetries: (v: number) => void;
  retryDelaySeconds: number;
  setRetryDelaySeconds: (v: number) => void;
  slaDeadlineMinutes?: number;
  setSlaDeadlineMinutes: (v: number | undefined) => void;
  requiresApproval: boolean;
  setRequiresApproval: (v: boolean) => void;
  t: any;
}

function AdvancedTab(props: AdvancedTabProps) {
  const { t } = props;
  
  return (
    <div className="form-grid">
      <FormField label={t('scheduler.fields.timeout', 'Timeout (seconds)')}>
        <input
          type="number"
          className="form-input"
          value={props.timeoutSeconds}
          onChange={e => props.setTimeoutSeconds(parseInt(e.target.value) || 3600)}
          min={60}
          max={86400}
        />
        <small className="form-help">
          {t('scheduler.timeoutHelp', 'Maximum time allowed for job execution')}
        </small>
      </FormField>
      
      <FormField label={t('scheduler.fields.maxRetries', 'Max Retries')}>
        <input
          type="number"
          className="form-input"
          value={props.maxRetries}
          onChange={e => props.setMaxRetries(parseInt(e.target.value) || 0)}
          min={0}
          max={10}
        />
      </FormField>
      
      <FormField label={t('scheduler.fields.retryDelay', 'Retry Delay (seconds)')}>
        <input
          type="number"
          className="form-input"
          value={props.retryDelaySeconds}
          onChange={e => props.setRetryDelaySeconds(parseInt(e.target.value) || 60)}
          min={10}
          max={3600}
        />
      </FormField>
      
      <FormField label={t('scheduler.fields.slaDeadline', 'SLA Deadline (minutes)')}>
        <input
          type="number"
          className="form-input"
          value={props.slaDeadlineMinutes || ''}
          onChange={e => props.setSlaDeadlineMinutes(e.target.value ? parseInt(e.target.value) : undefined)}
          min={1}
          placeholder={t('scheduler.placeholders.optional', 'Optional')}
        />
        <small className="form-help">
          {t('scheduler.slaHelp', 'Alert if job takes longer than this')}
        </small>
      </FormField>
      
      <FormField label={t('scheduler.fields.approval', 'Approval Required')} fullWidth>
        <label className="toggle-label">
          <input
            type="checkbox"
            checked={props.requiresApproval}
            onChange={e => props.setRequiresApproval(e.target.checked)}
          />
          <span>{t('scheduler.requireApproval', 'Require approval before execution')}</span>
        </label>
      </FormField>
    </div>
  );
}

// ============================================================================
// Helper Components
// ============================================================================

function FormField({
  label,
  required,
  fullWidth,
  children,
}: {
  label: string;
  required?: boolean;
  fullWidth?: boolean;
  children: React.ReactNode;
}) {
  return (
    <div className={`form-field ${fullWidth ? 'full-width' : ''}`}>
      <label className="form-label">
        {label}
        {required && <span className="required">*</span>}
      </label>
      {children}
    </div>
  );
}

export default JobEditorPage;
