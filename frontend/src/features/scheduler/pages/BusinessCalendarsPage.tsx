/**
 * World-Class Enterprise Scheduler - Business Calendars Page
 * Manage business calendars with holidays, working hours, and time zones
 */

import React, { useState, useEffect, useCallback } from 'react';
import { useTranslation } from 'react-i18next';
import { useParams, useNavigate, Link } from 'react-router-dom';
import * as schedulerService from '../services/schedulerService';
import {
  BusinessCalendar,
  Holiday,
  WorkingHours,
} from '../../../types/scheduler';
import '../styles/SchedulerDashboard.css';

// ============================================================================
// Calendar List Page
// ============================================================================

export function BusinessCalendarsPage() {
  const { t } = useTranslation();
  const navigate = useNavigate();
  
  const [calendars, setCalendars] = useState<BusinessCalendar[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [searchTerm, setSearchTerm] = useState('');
  const [filterTimezone, setFilterTimezone] = useState('');
  
  // Load calendars
  const loadCalendars = useCallback(async () => {
    try {
      setLoading(true);
      const data = await schedulerService.listCalendars();
      setCalendars(data);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load calendars');
    } finally {
      setLoading(false);
    }
  }, []);
  
  useEffect(() => {
    loadCalendars();
  }, [loadCalendars]);
  
  // Filter calendars
  const filteredCalendars = calendars.filter(cal => {
    const matchesSearch = !searchTerm || 
      cal.name.toLowerCase().includes(searchTerm.toLowerCase()) ||
      cal.description?.toLowerCase().includes(searchTerm.toLowerCase());
    const matchesTimezone = !filterTimezone || cal.timezone === filterTimezone;
    return matchesSearch && matchesTimezone;
  });
  
  // Get unique timezones
  const timezones = Array.from(new Set(calendars.map(c => c.timezone))).sort();
  
  // Handle delete
  const handleDelete = async (id: string) => {
    if (!confirm(t('scheduler.confirmDeleteCalendar', 'Are you sure you want to delete this calendar?'))) {
      return;
    }
    try {
      await schedulerService.deleteCalendar(id);
      loadCalendars();
    } catch (err) {
      console.error('Failed to delete:', err);
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
        <div>
          <h1>📅 {t('scheduler.businessCalendars', 'Business Calendars')}</h1>
          <p className="header-subtitle">
            {t('scheduler.businessCalendarsDesc', 'Manage business calendars, holidays, and working hours')}
          </p>
        </div>
        <div className="scheduler-header-actions">
          <Link to="/scheduler/calendars/new" className="btn btn-primary">
            ➕ {t('scheduler.createCalendar', 'Create Calendar')}
          </Link>
        </div>
      </div>
      
      {/* Filters */}
      <div className="filters-bar">
        <div className="search-box">
          <span className="search-icon">🔍</span>
          <input
            type="text"
            placeholder={t('scheduler.searchCalendars', 'Search calendars...')}
            value={searchTerm}
            onChange={e => setSearchTerm(e.target.value)}
            className="search-input"
          />
        </div>
        <select
          className="filter-select"
          value={filterTimezone}
          onChange={e => setFilterTimezone(e.target.value)}
          aria-label={t('scheduler.filterByTimezone', 'Filter by timezone')}
        >
          <option value="">{t('scheduler.allTimezones', 'All Timezones')}</option>
          {timezones.map(tz => (
            <option key={tz} value={tz}>{tz}</option>
          ))}
        </select>
      </div>
      
      {/* Error State */}
      {error && (
        <div className="error-banner">
          <span>⚠️ {error}</span>
          <button onClick={loadCalendars}>{t('scheduler.retry', 'Retry')}</button>
        </div>
      )}
      
      {/* Calendar Grid */}
      {filteredCalendars.length === 0 ? (
        <div className="empty-state">
          <div className="empty-state-icon">📅</div>
          <div className="empty-state-text">
            {searchTerm || filterTimezone
              ? t('scheduler.noCalendarsMatch', 'No calendars match your filters')
              : t('scheduler.noCalendars', 'No business calendars yet')}
          </div>
          {!searchTerm && !filterTimezone && (
            <Link to="/scheduler/calendars/new" className="btn btn-primary">
              {t('scheduler.createFirstCalendar', 'Create Your First Calendar')}
            </Link>
          )}
        </div>
      ) : (
        <div className="calendar-grid">
          {filteredCalendars.map(calendar => (
            <CalendarCard
              key={calendar.id}
              calendar={calendar}
              onEdit={() => navigate(`/scheduler/calendars/${calendar.id}/edit`)}
              onDelete={() => handleDelete(calendar.id)}
              t={t}
            />
          ))}
        </div>
      )}
    </div>
  );
}

// ============================================================================
// Calendar Card Component
// ============================================================================

interface CalendarCardProps {
  calendar: BusinessCalendar;
  onEdit: () => void;
  onDelete: () => void;
  t: (key: string, defaultValue: string) => string;
}

function CalendarCard({ calendar, onEdit, onDelete, t }: CalendarCardProps) {
  const upcomingHolidays = calendar.holidays
    .filter(h => new Date(h.date) >= new Date())
    .sort((a, b) => new Date(a.date).getTime() - new Date(b.date).getTime())
    .slice(0, 3);
  
  return (
    <div className="dashboard-card calendar-card">
      <div className="card-header">
        <div className="calendar-header-info">
          <h3>{calendar.name}</h3>
          {calendar.is_default && (
            <span className="badge badge-primary">{t('scheduler.default', 'Default')}</span>
          )}
        </div>
        <div className="card-actions">
          <button className="btn btn-sm btn-ghost" onClick={onEdit} title={t('scheduler.edit', 'Edit')}>
            ✏️
          </button>
          <button className="btn btn-sm btn-ghost" onClick={onDelete} title={t('scheduler.delete', 'Delete')}>
            🗑️
          </button>
        </div>
      </div>
      
      <div className="card-content">
        {calendar.description && (
          <p className="calendar-description">{calendar.description}</p>
        )}
        
        <div className="calendar-info">
          <div className="info-row">
            <span className="info-label">🌍 {t('scheduler.timezone', 'Timezone')}:</span>
            <span className="info-value">{calendar.timezone}</span>
          </div>
          <div className="info-row">
            <span className="info-label">🎉 {t('scheduler.holidays', 'Holidays')}:</span>
            <span className="info-value">{calendar.holidays.length}</span>
          </div>
        </div>
        
        {/* Working Hours Summary */}
        <div className="working-hours-summary">
          <h4>⏰ {t('scheduler.workingHours', 'Working Hours')}</h4>
          <div className="hours-grid">
            {['mon', 'tue', 'wed', 'thu', 'fri', 'sat', 'sun'].map(day => {
              const hours = calendar.working_hours[day as keyof WorkingHours];
              const isWorking = hours && hours.start && hours.end;
              return (
                <div
                  key={day}
                  className={`day-indicator ${isWorking ? 'working' : 'off'}`}
                  title={isWorking ? `${hours.start} - ${hours.end}` : t('scheduler.dayOff', 'Day off')}
                >
                  {day.charAt(0).toUpperCase()}
                </div>
              );
            })}
          </div>
        </div>
        
        {/* Upcoming Holidays */}
        {upcomingHolidays.length > 0 && (
          <div className="upcoming-holidays">
            <h4>📌 {t('scheduler.upcomingHolidays', 'Upcoming Holidays')}</h4>
            <ul className="holiday-list">
              {upcomingHolidays.map((holiday, index) => (
                <li key={index} className="holiday-item">
                  <span className="holiday-date">
                    {new Date(holiday.date).toLocaleDateString()}
                  </span>
                  <span className="holiday-name">{holiday.name}</span>
                </li>
              ))}
            </ul>
          </div>
        )}
      </div>
    </div>
  );
}

// ============================================================================
// Calendar Editor Page
// ============================================================================

export function CalendarEditorPage() {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const { calendarId } = useParams<{ calendarId: string }>();
  const isEditing = Boolean(calendarId) && calendarId !== 'new';
  
  const [loading, setLoading] = useState(isEditing);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [activeTab, setActiveTab] = useState<'general' | 'hours' | 'holidays'>('general');
  
  // Form state
  const [formData, setFormData] = useState<Partial<BusinessCalendar>>({
    name: '',
    description: '',
    timezone: Intl.DateTimeFormat().resolvedOptions().timeZone,
    is_default: false,
    working_hours: {
      mon: { start: '09:00', end: '17:00' },
      tue: { start: '09:00', end: '17:00' },
      wed: { start: '09:00', end: '17:00' },
      thu: { start: '09:00', end: '17:00' },
      fri: { start: '09:00', end: '17:00' },
      sat: { start: '', end: '' },
      sun: { start: '', end: '' },
    },
    holidays: [],
  });
  
  // Load existing calendar
  useEffect(() => {
    if (isEditing && calendarId) {
      setLoading(true);
      schedulerService.getCalendar(calendarId)
        .then(calendar => {
          setFormData(calendar);
        })
        .catch(err => {
          setError(err instanceof Error ? err.message : 'Failed to load calendar');
        })
        .finally(() => {
          setLoading(false);
        });
    }
  }, [calendarId, isEditing]);
  
  // Handle form submit
  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    
    if (!formData.name?.trim()) {
      setError(t('scheduler.validation.nameRequired', 'Calendar name is required'));
      return;
    }
    
    try {
      setSaving(true);
      setError(null);
      
      if (isEditing && calendarId) {
        await schedulerService.updateCalendar(calendarId, formData);
      } else {
        await schedulerService.createCalendar(formData as Omit<BusinessCalendar, 'id' | 'created_at' | 'updated_at'>);
      }
      
      navigate('/scheduler/calendars');
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to save calendar');
    } finally {
      setSaving(false);
    }
  };
  
  // Handle working hours change
  const handleHoursChange = (day: keyof WorkingHours, field: 'start' | 'end', value: string) => {
    setFormData(prev => ({
      ...prev,
      working_hours: {
        ...prev.working_hours!,
        [day]: {
          ...prev.working_hours![day],
          [field]: value,
        },
      },
    }));
  };
  
  // Handle holiday add
  const handleAddHoliday = () => {
    setFormData(prev => ({
      ...prev,
      holidays: [
        ...(prev.holidays || []),
        { id: crypto.randomUUID(), name: '', date: '', recurring: false },
      ],
    }));
  };
  
  // Handle holiday change
  const handleHolidayChange = (index: number, field: keyof Holiday, value: string | boolean) => {
    setFormData(prev => ({
      ...prev,
      holidays: prev.holidays?.map((h, i) => 
        i === index ? { ...h, [field]: value } : h
      ),
    }));
  };
  
  // Handle holiday remove
  const handleRemoveHoliday = (index: number) => {
    setFormData(prev => ({
      ...prev,
      holidays: prev.holidays?.filter((_, i) => i !== index),
    }));
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
        <div>
          <h1>
            {isEditing
              ? t('scheduler.editCalendar', 'Edit Calendar')
              : t('scheduler.createCalendar', 'Create Calendar')}
          </h1>
          <p className="header-subtitle">
            {t('scheduler.calendarEditorDesc', 'Configure business calendar settings')}
          </p>
        </div>
        <div className="scheduler-header-actions">
          <button className="btn btn-secondary" onClick={() => navigate('/scheduler/calendars')}>
            {t('common.cancel', 'Cancel')}
          </button>
          <button className="btn btn-primary" onClick={handleSubmit} disabled={saving}>
            {saving ? t('common.saving', 'Saving...') : t('common.save', 'Save')}
          </button>
        </div>
      </div>
      
      {/* Error */}
      {error && (
        <div className="error-banner">
          <span>⚠️ {error}</span>
          <button onClick={() => setError(null)}>✕</button>
        </div>
      )}
      
      {/* Tabs */}
      <div className="editor-tabs">
        <button
          className={`tab ${activeTab === 'general' ? 'active' : ''}`}
          onClick={() => setActiveTab('general')}
        >
          📋 {t('scheduler.tabs.general', 'General')}
        </button>
        <button
          className={`tab ${activeTab === 'hours' ? 'active' : ''}`}
          onClick={() => setActiveTab('hours')}
        >
          ⏰ {t('scheduler.tabs.workingHours', 'Working Hours')}
        </button>
        <button
          className={`tab ${activeTab === 'holidays' ? 'active' : ''}`}
          onClick={() => setActiveTab('holidays')}
        >
          🎉 {t('scheduler.tabs.holidays', 'Holidays')}
          {formData.holidays && formData.holidays.length > 0 && (
            <span className="tab-badge">{formData.holidays.length}</span>
          )}
        </button>
      </div>
      
      {/* Tab Content */}
      <form onSubmit={handleSubmit}>
        {/* General Tab */}
        {activeTab === 'general' && (
          <div className="dashboard-card">
            <div className="card-content">
              <div className="form-group">
                <label htmlFor="name">{t('scheduler.fields.name', 'Name')} *</label>
                <input
                  id="name"
                  type="text"
                  className="form-control"
                  value={formData.name || ''}
                  onChange={e => setFormData(prev => ({ ...prev, name: e.target.value }))}
                  placeholder={t('scheduler.placeholder.calendarName', 'e.g., US Business Calendar')}
                  required
                />
              </div>
              
              <div className="form-group">
                <label htmlFor="description">{t('scheduler.fields.description', 'Description')}</label>
                <textarea
                  id="description"
                  className="form-control"
                  rows={3}
                  value={formData.description || ''}
                  onChange={e => setFormData(prev => ({ ...prev, description: e.target.value }))}
                  placeholder={t('scheduler.placeholder.calendarDescription', 'Describe this calendar...')}
                />
              </div>
              
              <div className="form-group">
                <label htmlFor="timezone">{t('scheduler.fields.timezone', 'Timezone')} *</label>
                <select
                  id="timezone"
                  className="form-control"
                  value={formData.timezone || ''}
                  onChange={e => setFormData(prev => ({ ...prev, timezone: e.target.value }))}
                  required
                >
                  {TIMEZONES.map(tz => (
                    <option key={tz} value={tz}>{tz}</option>
                  ))}
                </select>
              </div>
              
              <div className="form-group">
                <label className="checkbox-label">
                  <input
                    type="checkbox"
                    checked={formData.is_default || false}
                    onChange={e => setFormData(prev => ({ ...prev, is_default: e.target.checked }))}
                  />
                  {t('scheduler.setAsDefault', 'Set as default calendar')}
                </label>
                <p className="form-help">
                  {t('scheduler.defaultCalendarHelp', 'The default calendar will be used when no calendar is specified for a job.')}
                </p>
              </div>
            </div>
          </div>
        )}
        
        {/* Working Hours Tab */}
        {activeTab === 'hours' && (
          <div className="dashboard-card">
            <div className="card-header">
              <h3>{t('scheduler.workingHours', 'Working Hours')}</h3>
              <p className="header-description">
                {t('scheduler.workingHoursDesc', 'Define the business hours for each day of the week. Leave empty for non-working days.')}
              </p>
            </div>
            <div className="card-content">
              <table className="working-hours-table">
                <thead>
                  <tr>
                    <th>{t('scheduler.day', 'Day')}</th>
                    <th>{t('scheduler.start', 'Start')}</th>
                    <th>{t('scheduler.end', 'End')}</th>
                    <th>{t('scheduler.status', 'Status')}</th>
                  </tr>
                </thead>
                <tbody>
                  {DAYS_OF_WEEK.map(({ key, label }) => {
                    const hours = formData.working_hours?.[key as keyof WorkingHours];
                    const isWorking = hours?.start && hours?.end;
                    
                    return (
                      <tr key={key} className={isWorking ? 'working-day' : 'non-working-day'}>
                        <td className="day-cell">
                          <strong>{t(`scheduler.days.${key}`, label)}</strong>
                        </td>
                        <td>
                          <input
                            type="time"
                            className="form-control time-input"
                            aria-label={`${t(`scheduler.startForDay`, 'Start for day')} ${label}`}
                            value={hours?.start || ''}
                            onChange={e => handleHoursChange(key as keyof WorkingHours, 'start', e.target.value)}
                          />
                        </td>
                        <td>
                          <input
                            type="time"
                            className="form-control time-input"
                            aria-label={`${t(`scheduler.endForDay`, 'End for day')} ${label}`}
                            value={hours?.end || ''}
                            onChange={e => handleHoursChange(key as keyof WorkingHours, 'end', e.target.value)}
                          />
                        </td>
                        <td>
                          {isWorking ? (
                            <span className="badge badge-success">{t('scheduler.working', 'Working')}</span>
                          ) : (
                            <span className="badge badge-secondary">{t('scheduler.dayOff', 'Day Off')}</span>
                          )}
                        </td>
                      </tr>
                    );
                  })}
                </tbody>
              </table>
              
              {/* Quick Actions */}
              <div className="quick-actions">
                <button
                  type="button"
                  className="btn btn-sm btn-secondary"
                  onClick={() => {
                    setFormData(prev => ({
                      ...prev,
                      working_hours: {
                        mon: { start: '09:00', end: '17:00' },
                        tue: { start: '09:00', end: '17:00' },
                        wed: { start: '09:00', end: '17:00' },
                        thu: { start: '09:00', end: '17:00' },
                        fri: { start: '09:00', end: '17:00' },
                        sat: { start: '', end: '' },
                        sun: { start: '', end: '' },
                      },
                    }));
                  }}
                >
                  {t('scheduler.setStandard9to5', 'Set Standard 9-5')}
                </button>
                <button
                  type="button"
                  className="btn btn-sm btn-secondary"
                  onClick={() => {
                    setFormData(prev => ({
                      ...prev,
                      working_hours: {
                        mon: { start: '00:00', end: '23:59' },
                        tue: { start: '00:00', end: '23:59' },
                        wed: { start: '00:00', end: '23:59' },
                        thu: { start: '00:00', end: '23:59' },
                        fri: { start: '00:00', end: '23:59' },
                        sat: { start: '00:00', end: '23:59' },
                        sun: { start: '00:00', end: '23:59' },
                      },
                    }));
                  }}
                >
                  {t('scheduler.set24x7', 'Set 24/7')}
                </button>
              </div>
            </div>
          </div>
        )}
        
        {/* Holidays Tab */}
        {activeTab === 'holidays' && (
          <div className="dashboard-card">
            <div className="card-header">
              <h3>{t('scheduler.holidays', 'Holidays')}</h3>
              <button
                type="button"
                className="btn btn-sm btn-primary"
                onClick={handleAddHoliday}
              >
                ➕ {t('scheduler.addHoliday', 'Add Holiday')}
              </button>
            </div>
            <div className="card-content">
              {!formData.holidays || formData.holidays.length === 0 ? (
                <div className="empty-state empty-state-small">
                  <div className="empty-state-icon">🎉</div>
                  <div className="empty-state-text">
                    {t('scheduler.noHolidays', 'No holidays configured')}
                  </div>
                  <button type="button" className="btn btn-primary btn-sm" onClick={handleAddHoliday}>
                    {t('scheduler.addFirstHoliday', 'Add First Holiday')}
                  </button>
                </div>
              ) : (
                <table className="data-table holidays-table">
                  <thead>
                    <tr>
                      <th>{t('scheduler.fields.name', 'Name')}</th>
                      <th>{t('scheduler.fields.date', 'Date')}</th>
                      <th>{t('scheduler.fields.recurring', 'Recurring')}</th>
                      <th className="actions-cell">{t('scheduler.actions', 'Actions')}</th>
                    </tr>
                  </thead>
                  <tbody>
                    {formData.holidays.map((holiday, index) => (
                      <tr key={index}>
                        <td>
                          <input
                            type="text"
                            className="form-control"
                            value={holiday.name}
                            onChange={e => handleHolidayChange(index, 'name', e.target.value)}
                            placeholder={t('scheduler.placeholder.holidayName', 'e.g., Christmas')}
                          />
                        </td>
                        <td>
                          <input
                            type="date"
                            className="form-control"
                            value={holiday.date}
                            onChange={e => handleHolidayChange(index, 'date', e.target.value)}
                          />
                        </td>
                        <td>
                          <label className="checkbox-label">
                            <input
                              type="checkbox"
                              checked={holiday.recurring || false}
                              onChange={e => handleHolidayChange(index, 'recurring', e.target.checked)}
                            />
                            {t('scheduler.yearly', 'Yearly')}
                          </label>
                        </td>
                        <td className="actions-cell">
                          <button
                            type="button"
                            className="btn btn-sm btn-ghost btn-danger"
                            onClick={() => handleRemoveHoliday(index)}
                            title={t('scheduler.remove', 'Remove')}
                          >
                            🗑️
                          </button>
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              )}
              
              {/* Import Options */}
              <div className="import-section">
                <h4>{t('scheduler.importHolidays', 'Import Holidays')}</h4>
                <div className="import-buttons">
                  <button
                    type="button"
                    className="btn btn-sm btn-secondary"
                    onClick={() => importUSHolidays(setFormData)}
                  >
                    🇺🇸 {t('scheduler.importUS', 'US Federal Holidays')}
                  </button>
                  <button
                    type="button"
                    className="btn btn-sm btn-secondary"
                    onClick={() => importUKHolidays(setFormData)}
                  >
                    🇬🇧 {t('scheduler.importUK', 'UK Bank Holidays')}
                  </button>
                </div>
              </div>
            </div>
          </div>
        )}
      </form>
    </div>
  );
}

// ============================================================================
// Constants
// ============================================================================

const DAYS_OF_WEEK = [
  { key: 'mon', label: 'Monday' },
  { key: 'tue', label: 'Tuesday' },
  { key: 'wed', label: 'Wednesday' },
  { key: 'thu', label: 'Thursday' },
  { key: 'fri', label: 'Friday' },
  { key: 'sat', label: 'Saturday' },
  { key: 'sun', label: 'Sunday' },
];

const TIMEZONES = [
  'UTC',
  'America/New_York',
  'America/Chicago',
  'America/Denver',
  'America/Los_Angeles',
  'America/Anchorage',
  'Pacific/Honolulu',
  'Europe/London',
  'Europe/Paris',
  'Europe/Berlin',
  'Europe/Moscow',
  'Asia/Tokyo',
  'Asia/Shanghai',
  'Asia/Singapore',
  'Asia/Dubai',
  'Asia/Kolkata',
  'Australia/Sydney',
  'Australia/Perth',
  'Pacific/Auckland',
];

// ============================================================================
// Holiday Import Functions
// ============================================================================

function importUSHolidays(setFormData: React.Dispatch<React.SetStateAction<Partial<BusinessCalendar>>>) {
  const currentYear = new Date().getFullYear();
  const usHolidays: Holiday[] = [
    { id: crypto.randomUUID(), name: "New Year's Day", date: `${currentYear}-01-01`, recurring: true },
    { id: crypto.randomUUID(), name: "Martin Luther King Jr. Day", date: `${currentYear}-01-15`, recurring: true },
    { id: crypto.randomUUID(), name: "Presidents' Day", date: `${currentYear}-02-19`, recurring: true },
    { id: crypto.randomUUID(), name: "Memorial Day", date: `${currentYear}-05-27`, recurring: true },
    { id: crypto.randomUUID(), name: "Juneteenth", date: `${currentYear}-06-19`, recurring: true },
    { id: crypto.randomUUID(), name: "Independence Day", date: `${currentYear}-07-04`, recurring: true },
    { id: crypto.randomUUID(), name: "Labor Day", date: `${currentYear}-09-02`, recurring: true },
    { id: crypto.randomUUID(), name: "Columbus Day", date: `${currentYear}-10-14`, recurring: true },
    { id: crypto.randomUUID(), name: "Veterans Day", date: `${currentYear}-11-11`, recurring: true },
    { id: crypto.randomUUID(), name: "Thanksgiving Day", date: `${currentYear}-11-28`, recurring: true },
    { id: crypto.randomUUID(), name: "Christmas Day", date: `${currentYear}-12-25`, recurring: true },
  ];
  
  setFormData(prev => ({
    ...prev,
    holidays: [...(prev.holidays || []), ...usHolidays],
  }));
}

function importUKHolidays(setFormData: React.Dispatch<React.SetStateAction<Partial<BusinessCalendar>>>) {
  const currentYear = new Date().getFullYear();
  const ukHolidays: Holiday[] = [
    { id: crypto.randomUUID(), name: "New Year's Day", date: `${currentYear}-01-01`, recurring: true },
    { id: crypto.randomUUID(), name: "Good Friday", date: `${currentYear}-03-29`, recurring: true },
    { id: crypto.randomUUID(), name: "Easter Monday", date: `${currentYear}-04-01`, recurring: true },
    { id: crypto.randomUUID(), name: "Early May Bank Holiday", date: `${currentYear}-05-06`, recurring: true },
    { id: crypto.randomUUID(), name: "Spring Bank Holiday", date: `${currentYear}-05-27`, recurring: true },
    { id: crypto.randomUUID(), name: "Summer Bank Holiday", date: `${currentYear}-08-26`, recurring: true },
    { id: crypto.randomUUID(), name: "Christmas Day", date: `${currentYear}-12-25`, recurring: true },
    { id: crypto.randomUUID(), name: "Boxing Day", date: `${currentYear}-12-26`, recurring: true },
  ];
  
  setFormData(prev => ({
    ...prev,
    holidays: [...(prev.holidays || []), ...ukHolidays],
  }));
}

export default BusinessCalendarsPage;
