/**
 * User Notification Preferences - Personal Notification Settings
 * 
 * Features:
 * - Email/SMS/Slack/Teams/Push channel toggles
 * - Digest mode configuration (immediate/hourly/daily/weekly)
 * - Do Not Disturb scheduling
 * - Priority filtering
 * - Connection testing for external channels
 */

import React, { useState, useEffect } from 'react';
import {
  Bell,
  Mail,
  MessageSquare,
  Smartphone,
  Clock,
  Moon,
  Filter,
  Save,
  CheckCircle,
  AlertCircle,
  RefreshCw,
} from 'lucide-react';

// ============================================================================
// TYPE DEFINITIONS
// ============================================================================

interface UserPreferences {
  id?: string;
  user_id: string;
  tenant_id: string;
  tenant_instance_id: string;
  
  // Email preferences
  email_enabled: boolean;
  email_address: string;
  
  // SMS preferences
  sms_enabled: boolean;
  phone_number: string;
  
  // Slack preferences
  slack_enabled: boolean;
  slack_user_id: string;
  slack_webhook_url: string;
  
  // Teams preferences
  teams_enabled: boolean;
  teams_user_id: string;
  teams_webhook_url: string;
  
  // Push preferences
  push_enabled: boolean;
  push_token: string;
  
  // Digest preferences
  digest_mode: string;
  digest_time: string;
  digest_days: string[];
  include_summary: boolean;
  include_full_details: boolean;
  
  // Do Not Disturb
  dnd_enabled: boolean;
  dnd_start_time: string;
  dnd_end_time: string;
  
  // Priority filtering
  min_priority: string;
  
  is_active: boolean;
}

interface UserPreferencesProps {
  tenant: { id: string; display_name: string };
  datasource: { id: string; source_name: string };
  userId: string;
}

const DIGEST_MODES = ['immediate', 'hourly', 'daily', 'weekly'];
const PRIORITIES = ['low', 'normal', 'high', 'urgent'];
const WEEKDAYS = ['Monday', 'Tuesday', 'Wednesday', 'Thursday', 'Friday', 'Saturday', 'Sunday'];

// ============================================================================
// MAIN COMPONENT
// ============================================================================

export const UserPreferences: React.FC<UserPreferencesProps> = ({
  tenant,
  datasource,
  userId,
}) => {
  const [preferences, setPreferences] = useState<UserPreferences>({
    user_id: userId,
    tenant_id: tenant.id,
    tenant_instance_id: datasource.id,
    email_enabled: true,
    email_address: '',
    sms_enabled: false,
    phone_number: '',
    slack_enabled: false,
    slack_user_id: '',
    slack_webhook_url: '',
    teams_enabled: false,
    teams_user_id: '',
    teams_webhook_url: '',
    push_enabled: false,
    push_token: '',
    digest_mode: 'immediate',
    digest_time: '09:00',
    digest_days: ['Monday', 'Tuesday', 'Wednesday', 'Thursday', 'Friday'],
    include_summary: true,
    include_full_details: false,
    dnd_enabled: false,
    dnd_start_time: '22:00',
    dnd_end_time: '08:00',
    min_priority: 'low',
    is_active: true,
  });

  const [loading, setLoading] = useState(false);
  const [saving, setSaving] = useState(false);
  const [testingSlack, setTestingSlack] = useState(false);
  const [testingTeams, setTestingTeams] = useState(false);
  const [successMessage, setSuccessMessage] = useState('');
  const [errorMessage, setErrorMessage] = useState('');

  // Load preferences on mount
  useEffect(() => {
    loadPreferences();
  }, [tenant.id, datasource.id, userId]);

  const loadPreferences = async () => {
    try {
      setLoading(true);
      const response = await fetch(
        `/api/bp-notifications/preferences?tenant_id=${tenant.id}&tenant_instance_id=${datasource.id}&user_id=${userId}`
      );

      if (response.ok) {
        const data = await response.json();
        if (data) {
          setPreferences(data);
        }
      }
    } catch (error) {
      console.error('Failed to load preferences:', error);
      setErrorMessage('Failed to load preferences');
    } finally {
      setLoading(false);
    }
  };

  const savePreferences = async () => {
    try {
      setSaving(true);
      setSuccessMessage('');
      setErrorMessage('');

      const response = await fetch(
        `/api/bp-notifications/preferences?tenant_id=${tenant.id}&tenant_instance_id=${datasource.id}`,
        {
          method: 'PUT',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify(preferences),
        }
      );

      if (response.ok) {
        setSuccessMessage('Preferences saved successfully!');
        setTimeout(() => setSuccessMessage(''), 3000);
      } else {
        setErrorMessage('Failed to save preferences');
      }
    } catch (error) {
      console.error('Failed to save preferences:', error);
      setErrorMessage('Failed to save preferences');
    } finally {
      setSaving(false);
    }
  };

  const testSlackConnection = async () => {
    setTestingSlack(true);
    setSuccessMessage('');
    setErrorMessage('');

    try {
      const response = await fetch('/api/bp-notifications/test/slack', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          webhook_url: preferences.slack_webhook_url,
          user_id: preferences.slack_user_id,
        }),
      });

      if (response.ok) {
        setSuccessMessage('Slack connection successful! Check your Slack for a test message.');
      } else {
        setErrorMessage('Slack connection failed. Check your credentials.');
      }
    } catch (error) {
      setErrorMessage('Failed to test Slack connection');
    } finally {
      setTestingSlack(false);
    }
  };

  const testTeamsConnection = async () => {
    setTestingTeams(true);
    setSuccessMessage('');
    setErrorMessage('');

    try {
      const response = await fetch('/api/bp-notifications/test/teams', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          webhook_url: preferences.teams_webhook_url,
          user_id: preferences.teams_user_id,
        }),
      });

      if (response.ok) {
        setSuccessMessage('Teams connection successful! Check your Teams for a test message.');
      } else {
        setErrorMessage('Teams connection failed. Check your credentials.');
      }
    } catch (error) {
      setErrorMessage('Failed to test Teams connection');
    } finally {
      setTestingTeams(false);
    }
  };

  const resetToDefaults = () => {
    if (confirm('Reset all preferences to default values?')) {
      setPreferences({
        ...preferences,
        email_enabled: true,
        sms_enabled: false,
        slack_enabled: false,
        teams_enabled: false,
        push_enabled: false,
        digest_mode: 'immediate',
        digest_time: '09:00',
        digest_days: ['Monday', 'Tuesday', 'Wednesday', 'Thursday', 'Friday'],
        include_summary: true,
        include_full_details: false,
        dnd_enabled: false,
        dnd_start_time: '22:00',
        dnd_end_time: '08:00',
        min_priority: 'low',
      });
    }
  };

  const toggleWeekday = (day: string) => {
    setPreferences((prev) => ({
      ...prev,
      digest_days: prev.digest_days.includes(day)
        ? prev.digest_days.filter((d) => d !== day)
        : [...prev.digest_days, day],
    }));
  };

  if (loading) {
    return (
      <div className="flex items-center justify-center min-h-screen">
        <div className="text-center">
          <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-blue-600 mx-auto mb-4"></div>
          <p className="text-gray-600">Loading preferences...</p>
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
              <Bell className="w-8 h-8 text-blue-600" />
              Notification Preferences
            </h1>
            <p className="text-gray-600 mt-2">
              Customize how and when you receive notifications
            </p>
          </div>
          <div className="flex items-center gap-3">
            <button
              onClick={resetToDefaults}
              className="px-4 py-2 bg-gray-200 text-gray-700 rounded-lg font-medium hover:bg-gray-300 transition-all flex items-center gap-2"
            >
              <RefreshCw className="w-4 h-4" />
              Reset
            </button>
            <button
              onClick={savePreferences}
              disabled={saving}
              className="px-4 py-2 bg-blue-600 text-white rounded-lg font-medium hover:bg-blue-700 transition-all flex items-center gap-2 disabled:opacity-50"
            >
              <Save className="w-4 h-4" />
              {saving ? 'Saving...' : 'Save Preferences'}
            </button>
          </div>
        </div>

        {/* Success/Error Messages */}
        {successMessage && (
          <div className="mt-4 p-4 bg-green-100 border-2 border-green-300 rounded-lg flex items-center gap-3">
            <CheckCircle className="w-5 h-5 text-green-600" />
            <p className="text-green-800 font-medium">{successMessage}</p>
          </div>
        )}
        {errorMessage && (
          <div className="mt-4 p-4 bg-red-100 border-2 border-red-300 rounded-lg flex items-center gap-3">
            <AlertCircle className="w-5 h-5 text-red-600" />
            <p className="text-red-800 font-medium">{errorMessage}</p>
          </div>
        )}
      </div>

      <div className="max-w-4xl mx-auto space-y-6">
        {/* Email Settings */}
        <div className="bg-white rounded-2xl shadow-xl p-6">
          <div className="flex items-center gap-3 mb-4">
            <Mail className="w-6 h-6 text-blue-600" />
            <h3 className="text-xl font-bold text-gray-900">Email Notifications</h3>
          </div>
          <div className="space-y-4">
            <label className="flex items-center gap-3 cursor-pointer">
              <input
                type="checkbox"
                checked={preferences.email_enabled}
                onChange={(e) =>
                  setPreferences({ ...preferences, email_enabled: e.target.checked })
                }
                className="w-5 h-5 rounded border-gray-300 text-blue-600 focus:ring-blue-500"
              />
              <span className="font-medium text-gray-900">Enable email notifications</span>
            </label>

            {preferences.email_enabled && (
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-2">
                  Email Address
                </label>
                <input
                  type="email"
                  value={preferences.email_address}
                  onChange={(e) =>
                    setPreferences({ ...preferences, email_address: e.target.value })
                  }
                  placeholder="your.email@example.com"
                  className="w-full p-3 border-2 border-gray-200 rounded-lg focus:outline-none focus:border-blue-500"
                />
              </div>
            )}
          </div>
        </div>

        {/* SMS Settings */}
        <div className="bg-white rounded-2xl shadow-xl p-6">
          <div className="flex items-center gap-3 mb-4">
            <MessageSquare className="w-6 h-6 text-green-600" />
            <h3 className="text-xl font-bold text-gray-900">SMS Notifications</h3>
          </div>
          <div className="space-y-4">
            <label className="flex items-center gap-3 cursor-pointer">
              <input
                type="checkbox"
                checked={preferences.sms_enabled}
                onChange={(e) =>
                  setPreferences({ ...preferences, sms_enabled: e.target.checked })
                }
                className="w-5 h-5 rounded border-gray-300 text-blue-600 focus:ring-blue-500"
              />
              <span className="font-medium text-gray-900">Enable SMS notifications</span>
            </label>

            {preferences.sms_enabled && (
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-2">
                  Phone Number
                </label>
                <input
                  type="tel"
                  value={preferences.phone_number}
                  onChange={(e) =>
                    setPreferences({ ...preferences, phone_number: e.target.value })
                  }
                  placeholder="+1 (555) 123-4567"
                  className="w-full p-3 border-2 border-gray-200 rounded-lg focus:outline-none focus:border-blue-500"
                />
              </div>
            )}
          </div>
        </div>

        {/* Slack Settings */}
        <div className="bg-white rounded-2xl shadow-xl p-6">
          <div className="flex items-center gap-3 mb-4">
            <MessageSquare className="w-6 h-6 text-purple-600" />
            <h3 className="text-xl font-bold text-gray-900">Slack Notifications</h3>
          </div>
          <div className="space-y-4">
            <label className="flex items-center gap-3 cursor-pointer">
              <input
                type="checkbox"
                checked={preferences.slack_enabled}
                onChange={(e) =>
                  setPreferences({ ...preferences, slack_enabled: e.target.checked })
                }
                className="w-5 h-5 rounded border-gray-300 text-blue-600 focus:ring-blue-500"
              />
              <span className="font-medium text-gray-900">Enable Slack notifications</span>
            </label>

            {preferences.slack_enabled && (
              <>
                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-2">
                    Slack User ID
                  </label>
                  <input
                    type="text"
                    value={preferences.slack_user_id}
                    onChange={(e) =>
                      setPreferences({ ...preferences, slack_user_id: e.target.value })
                    }
                    placeholder="U01234ABCDE"
                    className="w-full p-3 border-2 border-gray-200 rounded-lg focus:outline-none focus:border-blue-500"
                  />
                </div>

                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-2">
                    Slack Webhook URL
                  </label>
                  <input
                    type="url"
                    value={preferences.slack_webhook_url}
                    onChange={(e) =>
                      setPreferences({ ...preferences, slack_webhook_url: e.target.value })
                    }
                    placeholder="https://hooks.slack.com/services/..."
                    className="w-full p-3 border-2 border-gray-200 rounded-lg focus:outline-none focus:border-blue-500"
                  />
                </div>

                <button
                  onClick={testSlackConnection}
                  disabled={testingSlack}
                  className="px-4 py-2 bg-purple-600 text-white rounded-lg font-medium hover:bg-purple-700 transition-all disabled:opacity-50"
                >
                  {testingSlack ? 'Testing...' : 'Test Slack Connection'}
                </button>
              </>
            )}
          </div>
        </div>

        {/* Teams Settings */}
        <div className="bg-white rounded-2xl shadow-xl p-6">
          <div className="flex items-center gap-3 mb-4">
            <MessageSquare className="w-6 h-6 text-blue-700" />
            <h3 className="text-xl font-bold text-gray-900">Microsoft Teams Notifications</h3>
          </div>
          <div className="space-y-4">
            <label className="flex items-center gap-3 cursor-pointer">
              <input
                type="checkbox"
                checked={preferences.teams_enabled}
                onChange={(e) =>
                  setPreferences({ ...preferences, teams_enabled: e.target.checked })
                }
                className="w-5 h-5 rounded border-gray-300 text-blue-600 focus:ring-blue-500"
              />
              <span className="font-medium text-gray-900">Enable Teams notifications</span>
            </label>

            {preferences.teams_enabled && (
              <>
                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-2">
                    Teams User ID
                  </label>
                  <input
                    type="text"
                    value={preferences.teams_user_id}
                    onChange={(e) =>
                      setPreferences({ ...preferences, teams_user_id: e.target.value })
                    }
                    placeholder="user@company.com"
                    className="w-full p-3 border-2 border-gray-200 rounded-lg focus:outline-none focus:border-blue-500"
                  />
                </div>

                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-2">
                    Teams Webhook URL
                  </label>
                  <input
                    type="url"
                    value={preferences.teams_webhook_url}
                    onChange={(e) =>
                      setPreferences({ ...preferences, teams_webhook_url: e.target.value })
                    }
                    placeholder="https://outlook.office.com/webhook/..."
                    className="w-full p-3 border-2 border-gray-200 rounded-lg focus:outline-none focus:border-blue-500"
                  />
                </div>

                <button
                  onClick={testTeamsConnection}
                  disabled={testingTeams}
                  className="px-4 py-2 bg-blue-700 text-white rounded-lg font-medium hover:bg-blue-800 transition-all disabled:opacity-50"
                >
                  {testingTeams ? 'Testing...' : 'Test Teams Connection'}
                </button>
              </>
            )}
          </div>
        </div>

        {/* Push Settings */}
        <div className="bg-white rounded-2xl shadow-xl p-6">
          <div className="flex items-center gap-3 mb-4">
            <Smartphone className="w-6 h-6 text-orange-600" />
            <h3 className="text-xl font-bold text-gray-900">Push Notifications</h3>
          </div>
          <div className="space-y-4">
            <label className="flex items-center gap-3 cursor-pointer">
              <input
                type="checkbox"
                checked={preferences.push_enabled}
                onChange={(e) =>
                  setPreferences({ ...preferences, push_enabled: e.target.checked })
                }
                className="w-5 h-5 rounded border-gray-300 text-blue-600 focus:ring-blue-500"
              />
              <span className="font-medium text-gray-900">Enable push notifications</span>
            </label>

            {preferences.push_enabled && (
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-2">
                  Push Token (Auto-generated)
                </label>
                <input
                  type="text"
                  value={preferences.push_token}
                  readOnly
                  className="w-full p-3 border-2 border-gray-200 rounded-lg bg-gray-50 text-gray-500"
                />
              </div>
            )}
          </div>
        </div>

        {/* Digest Settings */}
        <div className="bg-white rounded-2xl shadow-xl p-6">
          <div className="flex items-center gap-3 mb-4">
            <Clock className="w-6 h-6 text-indigo-600" />
            <h3 className="text-xl font-bold text-gray-900">Digest Settings</h3>
          </div>
          <div className="space-y-4">
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-2">
                Digest Mode
              </label>
              <select
                value={preferences.digest_mode}
                onChange={(e) =>
                  setPreferences({ ...preferences, digest_mode: e.target.value })
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

            {(preferences.digest_mode === 'daily' || preferences.digest_mode === 'weekly') && (
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-2">
                  Delivery Time
                </label>
                <input
                  type="time"
                  value={preferences.digest_time}
                  onChange={(e) =>
                    setPreferences({ ...preferences, digest_time: e.target.value })
                  }
                  className="w-full p-3 border-2 border-gray-200 rounded-lg focus:outline-none focus:border-blue-500"
                />
              </div>
            )}

            {preferences.digest_mode === 'weekly' && (
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-2">
                  Delivery Days
                </label>
                <div className="grid grid-cols-2 md:grid-cols-4 gap-2">
                  {WEEKDAYS.map((day) => (
                    <label
                      key={day}
                      className="flex items-center gap-2 cursor-pointer p-2 rounded-lg hover:bg-gray-50"
                    >
                      <input
                        type="checkbox"
                        checked={preferences.digest_days.includes(day)}
                        onChange={() => toggleWeekday(day)}
                        className="w-4 h-4 rounded border-gray-300 text-blue-600 focus:ring-blue-500"
                      />
                      <span className="text-sm text-gray-700">{day}</span>
                    </label>
                  ))}
                </div>
              </div>
            )}

            <label className="flex items-center gap-3 cursor-pointer">
              <input
                type="checkbox"
                checked={preferences.include_summary}
                onChange={(e) =>
                  setPreferences({ ...preferences, include_summary: e.target.checked })
                }
                className="w-5 h-5 rounded border-gray-300 text-blue-600 focus:ring-blue-500"
              />
              <span className="font-medium text-gray-900">Include summary</span>
            </label>

            <label className="flex items-center gap-3 cursor-pointer">
              <input
                type="checkbox"
                checked={preferences.include_full_details}
                onChange={(e) =>
                  setPreferences({
                    ...preferences,
                    include_full_details: e.target.checked,
                  })
                }
                className="w-5 h-5 rounded border-gray-300 text-blue-600 focus:ring-blue-500"
              />
              <span className="font-medium text-gray-900">Include full details</span>
            </label>
          </div>
        </div>

        {/* Do Not Disturb */}
        <div className="bg-white rounded-2xl shadow-xl p-6">
          <div className="flex items-center gap-3 mb-4">
            <Moon className="w-6 h-6 text-indigo-600" />
            <h3 className="text-xl font-bold text-gray-900">Do Not Disturb</h3>
          </div>
          <div className="space-y-4">
            <label className="flex items-center gap-3 cursor-pointer">
              <input
                type="checkbox"
                checked={preferences.dnd_enabled}
                onChange={(e) =>
                  setPreferences({ ...preferences, dnd_enabled: e.target.checked })
                }
                className="w-5 h-5 rounded border-gray-300 text-blue-600 focus:ring-blue-500"
              />
              <span className="font-medium text-gray-900">Enable Do Not Disturb</span>
            </label>

            {preferences.dnd_enabled && (
              <div className="grid grid-cols-2 gap-4">
                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-2">
                    Start Time
                  </label>
                  <input
                    type="time"
                    value={preferences.dnd_start_time}
                    onChange={(e) =>
                      setPreferences({ ...preferences, dnd_start_time: e.target.value })
                    }
                    className="w-full p-3 border-2 border-gray-200 rounded-lg focus:outline-none focus:border-blue-500"
                  />
                </div>

                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-2">
                    End Time
                  </label>
                  <input
                    type="time"
                    value={preferences.dnd_end_time}
                    onChange={(e) =>
                      setPreferences({ ...preferences, dnd_end_time: e.target.value })
                    }
                    className="w-full p-3 border-2 border-gray-200 rounded-lg focus:outline-none focus:border-blue-500"
                  />
                </div>
              </div>
            )}
          </div>
        </div>

        {/* Priority Filter */}
        <div className="bg-white rounded-2xl shadow-xl p-6">
          <div className="flex items-center gap-3 mb-4">
            <Filter className="w-6 h-6 text-red-600" />
            <h3 className="text-xl font-bold text-gray-900">Priority Filter</h3>
          </div>
          <div className="space-y-4">
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-2">
                Minimum Priority (only send if notification is at or above this level)
              </label>
              <select
                value={preferences.min_priority}
                onChange={(e) =>
                  setPreferences({ ...preferences, min_priority: e.target.value })
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
    </div>
  );
};
