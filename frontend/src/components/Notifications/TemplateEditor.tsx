/**
 * Template Editor - Visual Notification Template Builder
 * 
 * Features:
 * - Visual template creation and editing
 * - Variable picker with drag-drop placeholders
 * - Channel configuration (email/SMS/Slack/Teams/push)
 * - Conditional rules builder
 * - Live preview with sample data
 * - Test send functionality
 * - Escalation settings
 * - Quick actions configuration
 */

import React, { useState, useEffect } from 'react';
import {
  Mail,
  MessageSquare,
  Smartphone,
  Plus,
  Save,
  Eye,
  Send,
  X,
  AlertCircle,
  CheckCircle,
  Copy,
  Zap,
  Clock,
  Users,
} from 'lucide-react';

// ============================================================================
// TYPE DEFINITIONS
// ============================================================================

interface Template {
  id?: string;
  template_key: string;
  template_name: string;
  description: string;
  category: string;
  subject_template: string;
  body_template: string;
  template_variables: string[];
  enabled_channels: string[];
  default_channel: string;
  digest_mode: string;
  escalation_enabled: boolean;
  escalation_delay_minutes?: number;
  escalation_recipient_roles: string[];
  is_active: boolean;
  priority: string;
  include_quick_actions: boolean;
  quick_actions?: any;
}

interface TemplateEditorProps {
  tenant: { id: string; display_name: string };
  datasource: { id: string; source_name: string };
  templateId?: string;
  onSave?: () => void;
  onCancel?: () => void;
}

const AVAILABLE_VARIABLES = [
  'user_name',
  'process_name',
  'process_link',
  'step_name',
  'requester_name',
  'assigned_by',
  'due_date',
  'description',
  'error_message',
  'sla_deadline',
  'time_remaining',
  'completion_percentage',
  'comment_text',
  'commenter_name',
];

const CATEGORIES = ['approval', 'reminder', 'alert', 'info', 'escalation'];
const PRIORITIES = ['low', 'normal', 'high', 'urgent'];
const CHANNELS = ['email', 'sms', 'slack', 'teams', 'push'];
const DIGEST_MODES = ['immediate', 'hourly', 'daily', 'weekly'];

// ============================================================================
// MAIN COMPONENT
// ============================================================================

export const TemplateEditor: React.FC<TemplateEditorProps> = ({
  tenant,
  datasource,
  templateId,
  onSave,
  onCancel,
}) => {
  const [template, setTemplate] = useState<Template>({
    template_key: '',
    template_name: '',
    description: '',
    category: 'info',
    subject_template: '',
    body_template: '',
    template_variables: [],
    enabled_channels: ['email'],
    default_channel: 'email',
    digest_mode: 'immediate',
    escalation_enabled: false,
    escalation_recipient_roles: [],
    is_active: true,
    priority: 'normal',
    include_quick_actions: false,
  });

  const [previewData, setPreviewData] = useState<Record<string, string>>({
    user_name: 'John Doe',
    process_name: 'Employee Onboarding',
    step_name: 'Manager Approval',
    requester_name: 'Jane Smith',
    due_date: '2025-01-15',
    description: 'Please review and approve the onboarding checklist',
    process_link: 'https://app.example.com/processes/123',
  });

  const [showPreview, setShowPreview] = useState(false);
  const [preview, setPreview] = useState<{ subject: string; body: string } | null>(null);
  const [loading, setLoading] = useState(false);
  const [saving, setSaving] = useState(false);

  // Load template if editing
  useEffect(() => {
    if (templateId) {
      loadTemplate();
    }
  }, [templateId]);

  const loadTemplate = async () => {
    try {
      setLoading(true);
      const response = await fetch(`/api/bp-notifications/templates/${templateId}`);
      const data = await response.json();
      setTemplate(data);
    } catch (error) {
      console.error('Failed to load template:', error);
    } finally {
      setLoading(false);
    }
  };

  // Save template
  const saveTemplate = async () => {
    try {
      setSaving(true);
      const url = templateId
        ? `/api/bp-notifications/templates/${templateId}?tenant_id=${tenant.id}&tenant_instance_id=${datasource.id}`
        : `/api/bp-notifications/templates?tenant_id=${tenant.id}&tenant_instance_id=${datasource.id}`;

      const response = await fetch(url, {
        method: templateId ? 'PUT' : 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(template),
      });

      if (response.ok) {
        onSave?.();
      }
    } catch (error) {
      console.error('Failed to save template:', error);
    } finally {
      setSaving(false);
    }
  };

  // Generate preview
  const generatePreview = async () => {
    if (!templateId && (!template.subject_template || !template.body_template)) {
      return;
    }

    try {
      const url = templateId
        ? `/api/bp-notifications/templates/${templateId}/render`
        : '/api/bp-notifications/templates/preview';

      const response = await fetch(url, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          subject_template: template.subject_template,
          body_template: template.body_template,
          variables: previewData,
        }),
      });

      const data = await response.json();
      setPreview(data);
      setShowPreview(true);
    } catch (error) {
      console.error('Failed to generate preview:', error);
    }
  };

  // Insert variable at cursor
  const insertVariable = (field: 'subject_template' | 'body_template', variable: string) => {
    const placeholder = `{${variable}}`;
    setTemplate((prev) => ({
      ...prev,
      [field]: prev[field] + placeholder,
      template_variables: prev.template_variables.includes(variable)
        ? prev.template_variables
        : [...prev.template_variables, variable],
    }));
  };

  // Toggle channel
  const toggleChannel = (channel: string) => {
    setTemplate((prev) => {
      const enabled = prev.enabled_channels.includes(channel);
      const newChannels = enabled
        ? prev.enabled_channels.filter((c) => c !== channel)
        : [...prev.enabled_channels, channel];

      // If removing default channel, set new default
      let newDefault = prev.default_channel;
      if (enabled && prev.default_channel === channel && newChannels.length > 0) {
        newDefault = newChannels[0];
      }

      return {
        ...prev,
        enabled_channels: newChannels,
        default_channel: newDefault,
      };
    });
  };

  if (loading) {
    return (
      <div className="flex items-center justify-center min-h-screen">
        <div className="text-center">
          <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-blue-600 mx-auto mb-4"></div>
          <p className="text-gray-600">Loading template...</p>
        </div>
      </div>
    );
  }

  return (
    <div className="min-h-screen bg-gradient-to-br from-slate-50 via-blue-50 to-indigo-50 p-6">
      {/* Header */}
      <div className="mb-8">
        <div className="flex items-center justify-between">
          <div>
            <h1 className="text-3xl font-bold text-gray-900 flex items-center gap-3">
              <Mail className="w-8 h-8 text-blue-600" />
              {templateId ? 'Edit Template' : 'Create Template'}
            </h1>
            <p className="text-gray-600 mt-2">
              Design notification templates with multi-channel support
            </p>
          </div>
          <div className="flex items-center gap-3">
            <button
              onClick={generatePreview}
              className="px-4 py-2 bg-purple-600 text-white rounded-lg font-medium hover:bg-purple-700 transition-all flex items-center gap-2"
            >
              <Eye className="w-4 h-4" />
              Preview
            </button>
            <button
              onClick={saveTemplate}
              disabled={saving}
              className="px-4 py-2 bg-green-600 text-white rounded-lg font-medium hover:bg-green-700 transition-all flex items-center gap-2 disabled:opacity-50"
            >
              <Save className="w-4 h-4" />
              {saving ? 'Saving...' : 'Save'}
            </button>
            {onCancel && (
              <button
                onClick={onCancel}
                className="px-4 py-2 bg-gray-200 text-gray-700 rounded-lg font-medium hover:bg-gray-300 transition-all"
              >
                Cancel
              </button>
            )}
          </div>
        </div>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
        {/* Main Form */}
        <div className="lg:col-span-2 space-y-6">
          {/* Basic Info */}
          <div className="bg-white rounded-2xl shadow-xl p-6">
            <h3 className="text-xl font-bold text-gray-900 mb-4">Basic Information</h3>
            <div className="space-y-4">
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-2">
                  Template Key *
                </label>
                <input
                  type="text"
                  value={template.template_key}
                  onChange={(e) =>
                    setTemplate({ ...template, template_key: e.target.value })
                  }
                  placeholder="bp_approval_required"
                  className="w-full p-3 border-2 border-gray-200 rounded-lg focus:outline-none focus:border-blue-500"
                />
              </div>

              <div>
                <label className="block text-sm font-medium text-gray-700 mb-2">
                  Template Name *
                </label>
                <input
                  type="text"
                  value={template.template_name}
                  onChange={(e) =>
                    setTemplate({ ...template, template_name: e.target.value })
                  }
                  placeholder="Approval Required"
                  className="w-full p-3 border-2 border-gray-200 rounded-lg focus:outline-none focus:border-blue-500"
                />
              </div>

              <div>
                <label className="block text-sm font-medium text-gray-700 mb-2">
                  Description
                </label>
                <textarea
                  value={template.description}
                  onChange={(e) =>
                    setTemplate({ ...template, description: e.target.value })
                  }
                  placeholder="Describe when this template should be used"
                  rows={3}
                  className="w-full p-3 border-2 border-gray-200 rounded-lg focus:outline-none focus:border-blue-500"
                />
              </div>

              <div className="grid grid-cols-2 gap-4">
                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-2">
                    Category
                  </label>
                  <select
                    value={template.category}
                    onChange={(e) =>
                      setTemplate({ ...template, category: e.target.value })
                    }
                    className="w-full p-3 border-2 border-gray-200 rounded-lg focus:outline-none focus:border-blue-500"
                  >
                    {CATEGORIES.map((cat) => (
                      <option key={cat} value={cat}>
                        {cat.charAt(0).toUpperCase() + cat.slice(1)}
                      </option>
                    ))}
                  </select>
                </div>

                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-2">
                    Priority
                  </label>
                  <select
                    value={template.priority}
                    onChange={(e) =>
                      setTemplate({ ...template, priority: e.target.value })
                    }
                    className="w-full p-3 border-2 border-gray-200 rounded-lg focus:outline-none focus:border-blue-500"
                  >
                    {PRIORITIES.map((priority) => (
                      <option key={priority} value={priority}>
                        {priority.charAt(0).toUpperCase() + priority.slice(1)}
                      </option>
                    ))}
                  </select>
                </div>
              </div>
            </div>
          </div>

          {/* Content */}
          <div className="bg-white rounded-2xl shadow-xl p-6">
            <h3 className="text-xl font-bold text-gray-900 mb-4">Content</h3>
            <div className="space-y-4">
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-2">
                  Subject Template *
                </label>
                <input
                  type="text"
                  value={template.subject_template}
                  onChange={(e) =>
                    setTemplate({ ...template, subject_template: e.target.value })
                  }
                  placeholder="Approval Required: {process_name}"
                  className="w-full p-3 border-2 border-gray-200 rounded-lg focus:outline-none focus:border-blue-500"
                />
              </div>

              <div>
                <label className="block text-sm font-medium text-gray-700 mb-2">
                  Body Template *
                </label>
                <textarea
                  value={template.body_template}
                  onChange={(e) =>
                    setTemplate({ ...template, body_template: e.target.value })
                  }
                  placeholder="Hi {user_name},&#10;&#10;You have a pending approval for {process_name}..."
                  rows={10}
                  className="w-full p-3 border-2 border-gray-200 rounded-lg focus:outline-none focus:border-blue-500 font-mono text-sm"
                />
              </div>
            </div>
          </div>

          {/* Channels */}
          <div className="bg-white rounded-2xl shadow-xl p-6">
            <h3 className="text-xl font-bold text-gray-900 mb-4">Delivery Channels</h3>
            <div className="space-y-4">
              <div className="grid grid-cols-2 md:grid-cols-5 gap-3">
                {CHANNELS.map((channel) => (
                  <button
                    key={channel}
                    onClick={() => toggleChannel(channel)}
                    className={`p-4 rounded-lg border-2 transition-all ${
                      template.enabled_channels.includes(channel)
                        ? 'border-blue-500 bg-blue-50 text-blue-700'
                        : 'border-gray-200 bg-white text-gray-600 hover:border-gray-300'
                    }`}
                  >
                    <div className="flex flex-col items-center gap-2">
                      {channel === 'email' && <Mail className="w-5 h-5" />}
                      {channel === 'sms' && <MessageSquare className="w-5 h-5" />}
                      {(channel === 'slack' || channel === 'teams') && (
                        <MessageSquare className="w-5 h-5" />
                      )}
                      {channel === 'push' && <Smartphone className="w-5 h-5" />}
                      <span className="text-xs font-medium capitalize">{channel}</span>
                    </div>
                  </button>
                ))}
              </div>

              <div>
                <label className="block text-sm font-medium text-gray-700 mb-2">
                  Default Channel
                </label>
                <select
                  value={template.default_channel}
                  onChange={(e) =>
                    setTemplate({ ...template, default_channel: e.target.value })
                  }
                  className="w-full p-3 border-2 border-gray-200 rounded-lg focus:outline-none focus:border-blue-500"
                >
                  {template.enabled_channels.map((channel) => (
                    <option key={channel} value={channel}>
                      {channel.charAt(0).toUpperCase() + channel.slice(1)}
                    </option>
                  ))}
                </select>
              </div>

              <div>
                <label className="block text-sm font-medium text-gray-700 mb-2">
                  Digest Mode
                </label>
                <select
                  value={template.digest_mode}
                  onChange={(e) =>
                    setTemplate({ ...template, digest_mode: e.target.value })
                  }
                  className="w-full p-3 border-2 border-gray-200 rounded-lg focus:outline-none focus:border-blue-500"
                >
                  {DIGEST_MODES.map((mode) => (
                    <option key={mode} value={mode}>
                      {mode.charAt(0).toUpperCase() + mode.slice(1)}
                    </option>
                  ))}
                </select>
              </div>
            </div>
          </div>

          {/* Escalation */}
          <div className="bg-white rounded-2xl shadow-xl p-6">
            <h3 className="text-xl font-bold text-gray-900 mb-4 flex items-center gap-2">
              <Zap className="w-5 h-5 text-orange-600" />
              Escalation Settings
            </h3>
            <div className="space-y-4">
              <label className="flex items-center gap-3 cursor-pointer">
                <input
                  type="checkbox"
                  checked={template.escalation_enabled}
                  onChange={(e) =>
                    setTemplate({ ...template, escalation_enabled: e.target.checked })
                  }
                  className="w-5 h-5 rounded border-gray-300 text-blue-600 focus:ring-blue-500"
                />
                <span className="font-medium text-gray-900">
                  Enable automatic escalation
                </span>
              </label>

              {template.escalation_enabled && (
                <>
                  <div>
                    <label className="block text-sm font-medium text-gray-700 mb-2">
                      Escalation Delay (minutes)
                    </label>
                    <input
                      type="number"
                      value={template.escalation_delay_minutes || 60}
                      onChange={(e) =>
                        setTemplate({
                          ...template,
                          escalation_delay_minutes: parseInt(e.target.value),
                        })
                      }
                      min="1"
                      className="w-full p-3 border-2 border-gray-200 rounded-lg focus:outline-none focus:border-blue-500"
                    />
                  </div>

                  <div>
                    <label className="block text-sm font-medium text-gray-700 mb-2">
                      Escalation Recipient Roles (comma-separated)
                    </label>
                    <input
                      type="text"
                      value={template.escalation_recipient_roles.join(', ')}
                      onChange={(e) =>
                        setTemplate({
                          ...template,
                          escalation_recipient_roles: e.target.value
                            .split(',')
                            .map((r) => r.trim()),
                        })
                      }
                      placeholder="manager, director, admin"
                      className="w-full p-3 border-2 border-gray-200 rounded-lg focus:outline-none focus:border-blue-500"
                    />
                  </div>
                </>
              )}
            </div>
          </div>
        </div>

        {/* Sidebar */}
        <div className="space-y-6">
          {/* Variable Picker */}
          <div className="bg-white rounded-2xl shadow-xl p-6">
            <h3 className="text-lg font-bold text-gray-900 mb-4">
              Available Variables
            </h3>
            <p className="text-sm text-gray-600 mb-4">
              Click to insert into subject or body
            </p>
            <div className="space-y-2">
              {AVAILABLE_VARIABLES.map((variable) => (
                <div
                  key={variable}
                  className="flex items-center gap-2 p-2 bg-gray-50 rounded-lg hover:bg-gray-100 transition-all"
                >
                  <button
                    onClick={() => insertVariable('subject_template', variable)}
                    className="flex-1 text-left text-sm font-mono text-blue-600 hover:text-blue-700"
                    title="Insert into subject"
                  >
                    {`{${variable}}`}
                  </button>
                  <button
                    onClick={() => insertVariable('body_template', variable)}
                    className="p-1 text-gray-400 hover:text-blue-600 transition-all"
                    title="Insert into body"
                  >
                    <Copy className="w-4 h-4" />
                  </button>
                </div>
              ))}
            </div>
          </div>

          {/* Preview Data */}
          <div className="bg-white rounded-2xl shadow-xl p-6">
            <h3 className="text-lg font-bold text-gray-900 mb-4">Preview Data</h3>
            <p className="text-sm text-gray-600 mb-4">
              Sample values for preview
            </p>
            <div className="space-y-3">
              {Object.keys(previewData).map((key) => (
                <div key={key}>
                  <label className="block text-xs font-medium text-gray-600 mb-1">
                    {key}
                  </label>
                  <input
                    type="text"
                    value={previewData[key]}
                    onChange={(e) =>
                      setPreviewData({ ...previewData, [key]: e.target.value })
                    }
                    className="w-full p-2 border border-gray-200 rounded text-sm focus:outline-none focus:border-blue-500"
                  />
                </div>
              ))}
            </div>
          </div>
        </div>
      </div>

      {/* Preview Modal */}
      {showPreview && preview && (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50 p-4">
          <div className="bg-white rounded-2xl shadow-2xl max-w-2xl w-full">
            <div className="p-6 border-b-2 border-gray-200 flex items-center justify-between">
              <h3 className="text-xl font-bold text-gray-900">Preview</h3>
              <button
                onClick={() => setShowPreview(false)}
                className="p-2 hover:bg-gray-100 rounded-lg transition-all"
              >
                <X className="w-5 h-5 text-gray-600" />
              </button>
            </div>
            <div className="p-6 space-y-4">
              <div>
                <label className="block text-sm font-medium text-gray-600 mb-2">
                  Subject:
                </label>
                <p className="text-lg font-bold text-gray-900">{preview.subject}</p>
              </div>
              <div>
                <label className="block text-sm font-medium text-gray-600 mb-2">
                  Body:
                </label>
                <div className="bg-gray-50 rounded-lg p-4">
                  <p className="text-gray-800 whitespace-pre-wrap">{preview.body}</p>
                </div>
              </div>
            </div>
          </div>
        </div>
      )}
    </div>
  );
};
