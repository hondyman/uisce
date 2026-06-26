/**
 * World-Class Enterprise Scheduler - Notification Templates Page
 * Manage multi-channel notification templates with i18n support
 */

import React, { useState, useEffect, useCallback } from 'react';
import { useTranslation } from 'react-i18next';
import { useParams, useNavigate, Link } from 'react-router-dom';
import * as schedulerService from '../services/schedulerService';
import {
  NotificationTemplate,
  NotificationChannel,
  JobStatus,
} from '../../../types/scheduler';
import '../styles/SchedulerDashboard.css';

// ============================================================================
// Notification Templates List Page
// ============================================================================

export function NotificationTemplatesPage() {
  const { t, i18n } = useTranslation();
  const navigate = useNavigate();
  
  const [templates, setTemplates] = useState<NotificationTemplate[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [filterChannel, setFilterChannel] = useState<string>('');
  const [filterEvent, setFilterEvent] = useState<string>('');
  
  // Load templates
  const loadTemplates = useCallback(async () => {
    try {
      setLoading(true);
      const data = await schedulerService.listNotificationTemplates();
      setTemplates(data);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load templates');
    } finally {
      setLoading(false);
    }
  }, []);
  
  useEffect(() => {
    loadTemplates();
  }, [loadTemplates]);
  
  // Filter templates
  const filteredTemplates = templates.filter(template => {
    const matchesChannel = !filterChannel || template.channel === filterChannel;
    const matchesEvent = !filterEvent || template.event_type === filterEvent;
    return matchesChannel && matchesEvent;
  });
  
  // Handle delete
  const handleDelete = async (id: string) => {
    if (!confirm(t('scheduler.confirmDeleteTemplate', 'Are you sure you want to delete this template?'))) {
      return;
    }
    try {
      await schedulerService.deleteNotificationTemplate(id);
      loadTemplates();
    } catch (err) {
      console.error('Failed to delete:', err);
    }
  };
  
  // Handle toggle active
  const handleToggleActive = async (template: NotificationTemplate) => {
    try {
      await schedulerService.updateNotificationTemplate(template.id, {
        ...template,
        is_active: !template.is_active,
      });
      loadTemplates();
    } catch (err) {
      console.error('Failed to toggle:', err);
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
          <h1>🔔 {t('scheduler.notificationTemplates', 'Notification Templates')}</h1>
          <p className="header-subtitle">
            {t('scheduler.notificationTemplatesDesc', 'Configure multi-channel notifications for job events')}
          </p>
        </div>
        <div className="scheduler-header-actions">
          <Link to="/scheduler/notifications/new" className="btn btn-primary">
            ➕ {t('scheduler.createTemplate', 'Create Template')}
          </Link>
        </div>
      </div>
      
      {/* Stats Cards */}
      <div className="stats-row">
        <StatCard
          icon="🔔"
          label={t('scheduler.totalTemplates', 'Total Templates')}
          value={templates.length}
        />
        <StatCard
          icon="✅"
          label={t('scheduler.activeTemplates', 'Active')}
          value={templates.filter(t => t.is_active).length}
        />
        <StatCard
          icon="📧"
          label={t('scheduler.emailTemplates', 'Email')}
          value={templates.filter(t => t.channel === NotificationChannel.EMAIL).length}
        />
        <StatCard
          icon="💬"
          label={t('scheduler.slackTemplates', 'Slack')}
          value={templates.filter(t => t.channel === NotificationChannel.SLACK).length}
        />
      </div>
      
      {/* Filters */}
      <div className="filters-bar">
        <select
          className="filter-select"
          value={filterChannel}
          onChange={e => setFilterChannel(e.target.value)}
          aria-label={t('scheduler.filterByChannel', 'Filter by channel')}
        >
          <option value="">{t('scheduler.allChannels', 'All Channels')}</option>
          <option value={NotificationChannel.EMAIL}>📧 Email</option>
          <option value={NotificationChannel.SLACK}>💬 Slack</option>
          <option value={NotificationChannel.WEBHOOK}>🔗 Webhook</option>
          <option value={NotificationChannel.SMS}>📱 SMS</option>
          <option value={NotificationChannel.TEAMS}>🟦 Teams</option>
          <option value={NotificationChannel.PAGERDUTY}>🚨 PagerDuty</option>
        </select>
        <select
          className="filter-select"
          value={filterEvent}
          onChange={e => setFilterEvent(e.target.value)}
          aria-label={t('scheduler.filterByEvent', 'Filter by event')}
        >
          <option value="">{t('scheduler.allEvents', 'All Events')}</option>
          <option value="job_started">{t('scheduler.event.jobStarted', 'Job Started')}</option>
          <option value="job_completed">{t('scheduler.event.jobCompleted', 'Job Completed')}</option>
          <option value="job_failed">{t('scheduler.event.jobFailed', 'Job Failed')}</option>
          <option value="job_retrying">{t('scheduler.event.jobRetrying', 'Job Retrying')}</option>
          <option value="job_cancelled">{t('scheduler.event.jobCancelled', 'Job Cancelled')}</option>
          <option value="sla_warning">{t('scheduler.event.slaWarning', 'SLA Warning')}</option>
          <option value="sla_breach">{t('scheduler.event.slaBreach', 'SLA Breach')}</option>
        </select>
      </div>
      
      {/* Error State */}
      {error && (
        <div className="error-banner">
          <span>⚠️ {error}</span>
          <button onClick={loadTemplates}>{t('scheduler.retry', 'Retry')}</button>
        </div>
      )}
      
      {/* Templates List */}
      {filteredTemplates.length === 0 ? (
        <div className="empty-state">
          <div className="empty-state-icon">🔔</div>
          <div className="empty-state-text">
            {filterChannel || filterEvent
              ? t('scheduler.noTemplatesMatch', 'No templates match your filters')
              : t('scheduler.noTemplates', 'No notification templates yet')}
          </div>
          {!filterChannel && !filterEvent && (
            <Link to="/scheduler/notifications/new" className="btn btn-primary">
              {t('scheduler.createFirstTemplate', 'Create Your First Template')}
            </Link>
          )}
        </div>
      ) : (
        <div className="templates-grid">
          {filteredTemplates.map(template => (
            <TemplateCard
              key={template.id}
              template={template}
              onEdit={() => navigate(`/scheduler/notifications/${template.id}/edit`)}
              onDelete={() => handleDelete(template.id)}
              onToggleActive={() => handleToggleActive(template)}
              currentLanguage={i18n.language}
              t={t}
            />
          ))}
        </div>
      )}
    </div>
  );
}

// ============================================================================
// Stat Card Component
// ============================================================================

interface StatCardProps {
  icon: string;
  label: string;
  value: number;
}

function StatCard({ icon, label, value }: StatCardProps) {
  return (
    <div className="stat-card">
      <span className="stat-icon">{icon}</span>
      <div className="stat-content">
        <div className="stat-value">{value}</div>
        <div className="stat-label">{label}</div>
      </div>
    </div>
  );
}

// ============================================================================
// Template Card Component
// ============================================================================

interface TemplateCardProps {
  template: NotificationTemplate;
  onEdit: () => void;
  onDelete: () => void;
  onToggleActive: () => void;
  currentLanguage: string;
  t: (key: string, defaultValue: string) => string;
}

function TemplateCard({ template, onEdit, onDelete, onToggleActive, currentLanguage, t }: TemplateCardProps) {
  const channelIcons: Record<NotificationChannel, string> = {
    [NotificationChannel.EMAIL]: '📧',
    [NotificationChannel.SLACK]: '💬',
    [NotificationChannel.WEBHOOK]: '🔗',
    [NotificationChannel.SMS]: '📱',
    [NotificationChannel.TEAMS]: '🟦',
    [NotificationChannel.PAGERDUTY]: '🚨',
  };
  
  const eventLabels: Record<string, string> = {
    job_started: t('scheduler.event.jobStarted', 'Job Started'),
    job_completed: t('scheduler.event.jobCompleted', 'Job Completed'),
    job_failed: t('scheduler.event.jobFailed', 'Job Failed'),
    job_retrying: t('scheduler.event.jobRetrying', 'Job Retrying'),
    job_cancelled: t('scheduler.event.jobCancelled', 'Job Cancelled'),
    sla_warning: t('scheduler.event.slaWarning', 'SLA Warning'),
    sla_breach: t('scheduler.event.slaBreach', 'SLA Breach'),
  };
  
  // Get content for current language
  const localizedContent = template.localized_content?.[currentLanguage] || 
    template.localized_content?.['en'] ||
    { subject: template.subject_template, body: template.body_template };
  
  const supportedLanguages = Object.keys(template.localized_content || {});
  
  return (
    <div className={`dashboard-card template-card ${!template.is_active ? 'inactive' : ''}`}>
      <div className="card-header">
        <div className="template-header-info">
          <span className="channel-icon" title={template.channel}>
            {channelIcons[template.channel]}
          </span>
          <h3>{template.name}</h3>
        </div>
        <div className="card-actions">
          <button
            className={`btn btn-sm btn-ghost ${template.is_active ? 'active' : ''}`}
            onClick={onToggleActive}
            title={template.is_active ? t('scheduler.deactivate', 'Deactivate') : t('scheduler.activate', 'Activate')}
          >
            {template.is_active ? '✅' : '⭕'}
          </button>
          <button className="btn btn-sm btn-ghost" onClick={onEdit} title={t('scheduler.edit', 'Edit')}>
            ✏️
          </button>
          <button className="btn btn-sm btn-ghost" onClick={onDelete} title={t('scheduler.delete', 'Delete')}>
            🗑️
          </button>
        </div>
      </div>
      
      <div className="card-content">
        <div className="template-meta">
          <span className={`badge ${getEventBadgeClass(template.event_type)}`}>
            {eventLabels[template.event_type] || template.event_type}
          </span>
          {!template.is_active && (
            <span className="badge badge-secondary">{t('scheduler.inactive', 'Inactive')}</span>
          )}
        </div>
        
        {/* Preview */}
        <div className="template-preview">
          {localizedContent.subject && (
            <div className="preview-subject">
              <strong>{t('scheduler.subject', 'Subject')}:</strong> {localizedContent.subject}
            </div>
          )}
          <div className="preview-body">
            {truncateText(localizedContent.body, 150)}
          </div>
        </div>
        
        {/* Supported Languages */}
        {supportedLanguages.length > 0 && (
          <div className="template-languages">
            <span className="languages-label">🌐 {t('scheduler.languages', 'Languages')}:</span>
            <div className="language-badges">
              {supportedLanguages.map(lang => (
                <span
                  key={lang}
                  className={`language-badge ${lang === currentLanguage ? 'current' : ''}`}
                >
                  {getLanguageFlag(lang)} {lang.toUpperCase()}
                </span>
              ))}
            </div>
          </div>
        )}
        
        {/* Recipients Preview */}
        {template.recipients && template.recipients.length > 0 && (
          <div className="template-recipients">
            <span className="recipients-label">👥 {template.recipients.length} {t('scheduler.recipients', 'recipients')}</span>
          </div>
        )}
      </div>
    </div>
  );
}

// ============================================================================
// Template Editor Page
// ============================================================================

export function NotificationTemplateEditorPage() {
  const { t, i18n } = useTranslation();
  const navigate = useNavigate();
  const { templateId } = useParams<{ templateId: string }>();
  const isEditing = Boolean(templateId) && templateId !== 'new';
  
  const [loading, setLoading] = useState(isEditing);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [activeTab, setActiveTab] = useState<'general' | 'content' | 'recipients' | 'preview'>('general');
  const [editingLanguage, setEditingLanguage] = useState(i18n.language || 'en');
  
  // Form state
  const [formData, setFormData] = useState<Partial<NotificationTemplate>>({
    name: '',
    channel: NotificationChannel.EMAIL,
    event_type: 'job_completed',
    subject_template: '',
    body_template: '',
    is_active: true,
    recipients: [],
    localized_content: {},
  });
  
  // Load existing template
  useEffect(() => {
    if (isEditing && templateId) {
      setLoading(true);
      schedulerService.getNotificationTemplate(templateId)
        .then(template => {
          setFormData(template);
        })
        .catch(err => {
          setError(err instanceof Error ? err.message : 'Failed to load template');
        })
        .finally(() => {
          setLoading(false);
        });
    }
  }, [templateId, isEditing]);
  
  // Handle form submit
  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    
    if (!formData.name?.trim()) {
      setError(t('scheduler.validation.nameRequired', 'Template name is required'));
      return;
    }
    
    try {
      setSaving(true);
      setError(null);
      
      if (isEditing && templateId) {
        await schedulerService.updateNotificationTemplate(templateId, formData);
      } else {
        await schedulerService.createNotificationTemplate(
          formData as Omit<NotificationTemplate, 'id' | 'created_at' | 'updated_at'>
        );
      }
      
      navigate('/scheduler/notifications');
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to save template');
    } finally {
      setSaving(false);
    }
  };
  
  // Handle localized content change
  const handleLocalizedContentChange = (lang: string, field: 'subject' | 'body', value: string) => {
    setFormData(prev => ({
      ...prev,
      localized_content: {
        ...prev.localized_content,
        [lang]: {
          ...prev.localized_content?.[lang],
          [field]: value,
        },
      },
    }));
  };
  
  // Handle add language
  const handleAddLanguage = (lang: string) => {
    if (!formData.localized_content?.[lang]) {
      setFormData(prev => ({
        ...prev,
        localized_content: {
          ...prev.localized_content,
          [lang]: { subject: '', body: '' },
        },
      }));
      setEditingLanguage(lang);
    }
  };
  
  // Handle remove language
  const handleRemoveLanguage = (lang: string) => {
    const newContent = { ...formData.localized_content };
    delete newContent[lang];
    setFormData(prev => ({ ...prev, localized_content: newContent }));
    if (editingLanguage === lang) {
      setEditingLanguage('en');
    }
  };
  
  // Handle recipients change
  const handleRecipientsChange = (value: string) => {
    const recipients = value.split(',').map(r => r.trim()).filter(Boolean);
    setFormData(prev => ({ ...prev, recipients }));
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
              ? t('scheduler.editTemplate', 'Edit Template')
              : t('scheduler.createTemplate', 'Create Template')}
          </h1>
        </div>
        <div className="scheduler-header-actions">
          <button className="btn btn-secondary" onClick={() => navigate('/scheduler/notifications')}>
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
          className={`tab ${activeTab === 'content' ? 'active' : ''}`}
          onClick={() => setActiveTab('content')}
        >
          ✏️ {t('scheduler.tabs.content', 'Content')}
        </button>
        <button
          className={`tab ${activeTab === 'recipients' ? 'active' : ''}`}
          onClick={() => setActiveTab('recipients')}
        >
          👥 {t('scheduler.tabs.recipients', 'Recipients')}
        </button>
        <button
          className={`tab ${activeTab === 'preview' ? 'active' : ''}`}
          onClick={() => setActiveTab('preview')}
        >
          👁️ {t('scheduler.tabs.preview', 'Preview')}
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
                  placeholder={t('scheduler.placeholder.templateName', 'e.g., Job Failed Alert')}
                  required
                />
              </div>
              
              <div className="form-row">
                <div className="form-group">
                  <label htmlFor="channel">{t('scheduler.fields.channel', 'Channel')} *</label>
                  <select
                    id="channel"
                    className="form-control"
                    value={formData.channel}
                    onChange={e => setFormData(prev => ({ ...prev, channel: e.target.value as NotificationChannel }))}
                    required
                  >
                    <option value={NotificationChannel.EMAIL}>📧 Email</option>
                    <option value={NotificationChannel.SLACK}>💬 Slack</option>
                    <option value={NotificationChannel.WEBHOOK}>🔗 Webhook</option>
                    <option value={NotificationChannel.SMS}>📱 SMS</option>
                    <option value={NotificationChannel.TEAMS}>🟦 Microsoft Teams</option>
                    <option value={NotificationChannel.PAGERDUTY}>🚨 PagerDuty</option>
                  </select>
                </div>
                
                <div className="form-group">
                  <label htmlFor="event_type">{t('scheduler.fields.eventType', 'Event Type')} *</label>
                  <select
                    id="event_type"
                    className="form-control"
                    value={formData.event_type}
                    onChange={e => setFormData(prev => ({ ...prev, event_type: e.target.value }))}
                    required
                  >
                    <option value="job_started">{t('scheduler.event.jobStarted', 'Job Started')}</option>
                    <option value="job_completed">{t('scheduler.event.jobCompleted', 'Job Completed')}</option>
                    <option value="job_failed">{t('scheduler.event.jobFailed', 'Job Failed')}</option>
                    <option value="job_retrying">{t('scheduler.event.jobRetrying', 'Job Retrying')}</option>
                    <option value="job_cancelled">{t('scheduler.event.jobCancelled', 'Job Cancelled')}</option>
                    <option value="sla_warning">{t('scheduler.event.slaWarning', 'SLA Warning')}</option>
                    <option value="sla_breach">{t('scheduler.event.slaBreach', 'SLA Breach')}</option>
                  </select>
                </div>
              </div>
              
              <div className="form-group">
                <label className="checkbox-label">
                  <input
                    type="checkbox"
                    checked={formData.is_active ?? true}
                    onChange={e => setFormData(prev => ({ ...prev, is_active: e.target.checked }))}
                  />
                  {t('scheduler.templateActive', 'Template is active')}
                </label>
              </div>
            </div>
          </div>
        )}
        
        {/* Content Tab */}
        {activeTab === 'content' && (
          <div className="dashboard-card">
            <div className="card-header">
              <h3>{t('scheduler.contentByLanguage', 'Content by Language')}</h3>
              <div className="language-selector">
                <select
                  className="filter-select"
                  value={editingLanguage}
                  onChange={e => setEditingLanguage(e.target.value)}
                  aria-label={t('scheduler.selectLanguage', 'Select language')}
                >
                  {Object.keys(formData.localized_content || { en: {} }).map(lang => (
                    <option key={lang} value={lang}>
                      {getLanguageFlag(lang)} {getLanguageName(lang)}
                    </option>
                  ))}
                </select>
                <div className="add-language-dropdown">
                  <button type="button" className="btn btn-sm btn-secondary">
                    ➕ {t('scheduler.addLanguage', 'Add Language')}
                  </button>
                  <div className="dropdown-menu">
                    {SUPPORTED_LANGUAGES.filter(l => !formData.localized_content?.[l.code]).map(lang => (
                      <button
                        key={lang.code}
                        type="button"
                        onClick={() => handleAddLanguage(lang.code)}
                      >
                        {lang.flag} {lang.name}
                      </button>
                    ))}
                  </div>
                </div>
                {editingLanguage !== 'en' && (
                  <button
                    type="button"
                    className="btn btn-sm btn-ghost btn-danger"
                    onClick={() => handleRemoveLanguage(editingLanguage)}
                  >
                    🗑️
                  </button>
                )}
              </div>
            </div>
            <div className="card-content">
              {formData.channel === NotificationChannel.EMAIL && (
                <div className="form-group">
                  <label htmlFor="subject">{t('scheduler.fields.subject', 'Subject')}</label>
                  <input
                    id="subject"
                    type="text"
                    className="form-control"
                    value={formData.localized_content?.[editingLanguage]?.subject || ''}
                    onChange={e => handleLocalizedContentChange(editingLanguage, 'subject', e.target.value)}
                    placeholder={t('scheduler.placeholder.subject', 'e.g., Job {{job_name}} has {{status}}')}
                  />
                </div>
              )}
              
              <div className="form-group">
                <label htmlFor="body">{t('scheduler.fields.body', 'Body')}</label>
                <textarea
                  id="body"
                  className="form-control code-textarea"
                  rows={12}
                  value={formData.localized_content?.[editingLanguage]?.body || ''}
                  onChange={e => handleLocalizedContentChange(editingLanguage, 'body', e.target.value)}
                  placeholder={t('scheduler.placeholder.body', 'Enter notification content...')}
                />
              </div>
              
              {/* Template Variables Help */}
              <div className="template-variables-help">
                <h4>{t('scheduler.availableVariables', 'Available Variables')}</h4>
                <div className="variables-grid">
                  {TEMPLATE_VARIABLES.map(variable => (
                    <div key={variable.name} className="variable-item">
                      <code>{`{{${variable.name}}}`}</code>
                      <span>{variable.description}</span>
                    </div>
                  ))}
                </div>
              </div>
            </div>
          </div>
        )}
        
        {/* Recipients Tab */}
        {activeTab === 'recipients' && (
          <div className="dashboard-card">
            <div className="card-content">
              <div className="form-group">
                <label htmlFor="recipients">{t('scheduler.fields.recipients', 'Recipients')}</label>
                <textarea
                  id="recipients"
                  className="form-control"
                  rows={5}
                  value={formData.recipients?.join(', ') || ''}
                  onChange={e => handleRecipientsChange(e.target.value)}
                  placeholder={getRecipientsPlaceholder(formData.channel)}
                />
                <p className="form-help">{getRecipientsHelp(formData.channel, t)}</p>
              </div>
              
              {/* Channel-specific configuration */}
              {formData.channel === NotificationChannel.SLACK && (
                <div className="form-group">
                  <label htmlFor="slack_channel">{t('scheduler.fields.slackChannel', 'Slack Channel')}</label>
                  <input
                    id="slack_channel"
                    type="text"
                    className="form-control"
                    placeholder="#alerts or @username"
                  />
                </div>
              )}
              
              {formData.channel === NotificationChannel.WEBHOOK && (
                <div className="form-group">
                  <label htmlFor="webhook_url">{t('scheduler.fields.webhookUrl', 'Webhook URL')}</label>
                  <input
                    id="webhook_url"
                    type="url"
                    className="form-control"
                    placeholder="https://api.example.com/webhook"
                  />
                </div>
              )}
            </div>
          </div>
        )}
        
        {/* Preview Tab */}
        {activeTab === 'preview' && (
          <div className="dashboard-card">
            <div className="card-header">
              <h3>{t('scheduler.preview', 'Preview')}</h3>
              <select
                className="filter-select"
                value={editingLanguage}
                onChange={e => setEditingLanguage(e.target.value)}
                aria-label={t('scheduler.selectLanguage', 'Select preview language')}
              >
                {Object.keys(formData.localized_content || { en: {} }).map(lang => (
                  <option key={lang} value={lang}>
                    {getLanguageFlag(lang)} {getLanguageName(lang)}
                  </option>
                ))}
              </select>
            </div>
            <div className="card-content">
              <NotificationPreview
                template={formData}
                language={editingLanguage}
                channel={formData.channel || NotificationChannel.EMAIL}
              />
            </div>
          </div>
        )}
      </form>
    </div>
  );
}

// ============================================================================
// Notification Preview Component
// ============================================================================

interface NotificationPreviewProps {
  template: Partial<NotificationTemplate>;
  language: string;
  channel: NotificationChannel;
}

function NotificationPreview({ template, language, channel }: NotificationPreviewProps) {
  const sampleData = {
    job_name: 'Daily ETL Pipeline',
    job_id: 'job_abc123',
    execution_id: 'exec_xyz789',
    status: 'FAILED',
    error_message: 'Connection timeout after 30s',
    started_at: new Date().toISOString(),
    completed_at: new Date().toISOString(),
    duration: '2m 34s',
    attempt: 3,
    max_attempts: 3,
  };
  
  const content = template.localized_content?.[language] || { subject: '', body: '' };
  
  // Simple variable replacement for preview
  const replaceVariables = (text: string) => {
    let result = text;
    Object.entries(sampleData).forEach(([key, value]) => {
      result = result.replace(new RegExp(`{{${key}}}`, 'g'), String(value));
    });
    return result;
  };
  
  if (channel === NotificationChannel.EMAIL) {
    return (
      <div className="email-preview">
        <div className="email-header">
          <div className="email-from">
            <strong>From:</strong> scheduler@company.com
          </div>
          <div className="email-to">
            <strong>To:</strong> {template.recipients?.join(', ') || 'team@company.com'}
          </div>
          <div className="email-subject">
            <strong>Subject:</strong> {replaceVariables(content.subject || '')}
          </div>
        </div>
        <div className="email-body">
          <pre>{replaceVariables(content.body || '')}</pre>
        </div>
      </div>
    );
  }
  
  if (channel === NotificationChannel.SLACK) {
    return (
      <div className="slack-preview">
        <div className="slack-message">
          <div className="slack-avatar">🤖</div>
          <div className="slack-content">
            <div className="slack-username">Scheduler Bot</div>
            <div className="slack-text">
              <pre>{replaceVariables(content.body || '')}</pre>
            </div>
          </div>
        </div>
      </div>
    );
  }
  
  return (
    <div className="generic-preview">
      <pre>{replaceVariables(content.body || '')}</pre>
    </div>
  );
}

// ============================================================================
// Constants
// ============================================================================

const SUPPORTED_LANGUAGES = [
  { code: 'en', name: 'English', flag: '🇺🇸' },
  { code: 'es', name: 'Spanish', flag: '🇪🇸' },
  { code: 'fr', name: 'French', flag: '🇫🇷' },
  { code: 'de', name: 'German', flag: '🇩🇪' },
  { code: 'it', name: 'Italian', flag: '🇮🇹' },
  { code: 'pt', name: 'Portuguese', flag: '🇵🇹' },
  { code: 'ja', name: 'Japanese', flag: '🇯🇵' },
  { code: 'zh', name: 'Chinese', flag: '🇨🇳' },
  { code: 'ko', name: 'Korean', flag: '🇰🇷' },
];

const TEMPLATE_VARIABLES = [
  { name: 'job_name', description: 'Name of the job' },
  { name: 'job_id', description: 'Unique job identifier' },
  { name: 'execution_id', description: 'Execution identifier' },
  { name: 'status', description: 'Job status (COMPLETED, FAILED, etc.)' },
  { name: 'error_message', description: 'Error message if failed' },
  { name: 'started_at', description: 'Execution start time' },
  { name: 'completed_at', description: 'Execution end time' },
  { name: 'duration', description: 'Total execution duration' },
  { name: 'attempt', description: 'Current attempt number' },
  { name: 'max_attempts', description: 'Maximum retry attempts' },
];

// ============================================================================
// Utility Functions
// ============================================================================

function getLanguageFlag(code: string): string {
  const lang = SUPPORTED_LANGUAGES.find(l => l.code === code);
  return lang?.flag || '🌐';
}

function getLanguageName(code: string): string {
  const lang = SUPPORTED_LANGUAGES.find(l => l.code === code);
  return lang?.name || code.toUpperCase();
}

function getEventBadgeClass(eventType: string): string {
  const classes: Record<string, string> = {
    job_completed: 'badge-success',
    job_failed: 'badge-danger',
    job_started: 'badge-info',
    job_retrying: 'badge-warning',
    job_cancelled: 'badge-secondary',
    sla_warning: 'badge-warning',
    sla_breach: 'badge-danger',
  };
  return classes[eventType] || 'badge-secondary';
}

function getRecipientsPlaceholder(channel?: NotificationChannel): string {
  switch (channel) {
    case NotificationChannel.EMAIL:
      return 'user@example.com, team@example.com';
    case NotificationChannel.SLACK:
      return '#channel-name, @username';
    case NotificationChannel.SMS:
      return '+1234567890, +0987654321';
    case NotificationChannel.WEBHOOK:
      return 'https://webhook.example.com/notify';
    default:
      return 'Enter recipients...';
  }
}

function getRecipientsHelp(channel: NotificationChannel | undefined, t: (key: string, defaultValue: string) => string): string {
  switch (channel) {
    case NotificationChannel.EMAIL:
      return t('scheduler.help.emailRecipients', 'Enter email addresses separated by commas');
    case NotificationChannel.SLACK:
      return t('scheduler.help.slackRecipients', 'Enter Slack channels (#channel) or users (@user) separated by commas');
    case NotificationChannel.SMS:
      return t('scheduler.help.smsRecipients', 'Enter phone numbers with country code separated by commas');
    default:
      return t('scheduler.help.recipients', 'Enter recipients separated by commas');
  }
}

function truncateText(text: string, maxLength: number): string {
  if (text.length <= maxLength) return text;
  return text.slice(0, maxLength - 3) + '...';
}

export default NotificationTemplatesPage;
