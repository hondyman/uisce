import React, { useState, useEffect } from 'react';
import { useTenant } from '../../../context/TenantContext';

interface CubeSettings {
  // Connection Settings
  cubeApiUrl: string;
  cubeApiSecret: string;
  
  // Query Settings
  queryTimeout: number;
  maxRowsPerQuery: number;
  enableQueryCaching: boolean;
  defaultCacheTtl: number;
  
  // Pre-Aggregation Settings
  preAggScheduleEnabled: boolean;
  preAggRefreshInterval: string;
  preAggPartitionGranularity: string;
  preAggConcurrency: number;
  
  // Security Settings
  jwtAudience: string;
  jwtIssuer: string;
  enableRowLevelSecurity: boolean;
  rlsContextKey: string;
  
  // Performance Settings
  maxCompilerCacheSize: number;
  scheduledRefreshTimezone: string;
  enableDebugLogs: boolean;
}

export function CubeSettingsPage() {
  const { tenant, datasource } = useTenant();
  const [settings, setSettings] = useState<CubeSettings>({
    cubeApiUrl: 'http://localhost:4000',
    cubeApiSecret: '••••••••••••••••',
    queryTimeout: 120,
    maxRowsPerQuery: 50000,
    enableQueryCaching: true,
    defaultCacheTtl: 300,
    preAggScheduleEnabled: true,
    preAggRefreshInterval: '1 hour',
    preAggPartitionGranularity: 'month',
    preAggConcurrency: 4,
    jwtAudience: 'cube-api',
    jwtIssuer: 'https://auth.example.com',
    enableRowLevelSecurity: true,
    rlsContextKey: 'tenant_id',
    maxCompilerCacheSize: 1000,
    scheduledRefreshTimezone: 'America/New_York',
    enableDebugLogs: false,
  });
  const [saving, setSaving] = useState(false);
  const [activeSection, setActiveSection] = useState<string>('connection');

  if (!tenant || !datasource) {
    return (
      <div className="p-8">
        <div className="bg-yellow-50 border border-yellow-200 rounded-lg p-6 text-center">
          <h2 className="text-lg font-semibold text-yellow-800">Select a Tenant</h2>
          <p className="text-yellow-700 mt-2">
            Please select a tenant and datasource to configure Cube settings.
          </p>
        </div>
      </div>
    );
  }

  const sections = [
    { id: 'connection', label: 'Connection', icon: LinkIcon },
    { id: 'query', label: 'Query Settings', icon: QueryIcon },
    { id: 'preagg', label: 'Pre-Aggregations', icon: LayersIcon },
    { id: 'security', label: 'Security', icon: ShieldIcon },
    { id: 'performance', label: 'Performance', icon: SpeedIcon },
  ];

  const handleSave = async () => {
    setSaving(true);
    try {
      // Simulate API call
      await new Promise((resolve) => setTimeout(resolve, 1000));
      // Would call real API here
    } catch (err) {
      console.error('Failed to save settings:', err);
    } finally {
      setSaving(false);
    }
  };

  return (
    <div className="p-8">
      {/* Header */}
      <div className="flex items-center justify-between mb-8">
        <div>
          <h1 className="text-2xl font-bold text-gray-900">Cube Settings</h1>
          <p className="text-gray-500 mt-1">
            Configure Cube.js semantic layer for {tenant.display_name}
          </p>
        </div>
        <button
          onClick={handleSave}
          disabled={saving}
          className="px-4 py-2 bg-indigo-600 text-white rounded-lg hover:bg-indigo-700 transition-colors disabled:opacity-50 flex items-center gap-2"
        >
          {saving ? (
            <>
              <LoadingIcon className="w-5 h-5 animate-spin" />
              Saving...
            </>
          ) : (
            <>
              <SaveIcon className="w-5 h-5" />
              Save Changes
            </>
          )}
        </button>
      </div>

      <div className="flex gap-8">
        {/* Sidebar Navigation */}
        <nav className="w-48 space-y-1">
          {sections.map((section) => {
            const Icon = section.icon;
            return (
              <button
                key={section.id}
                onClick={() => setActiveSection(section.id)}
                className={`w-full flex items-center gap-3 px-3 py-2 text-sm rounded-lg transition-colors ${
                  activeSection === section.id
                    ? 'bg-indigo-50 text-indigo-700'
                    : 'text-gray-600 hover:bg-gray-100'
                }`}
              >
                <Icon className="w-5 h-5" />
                {section.label}
              </button>
            );
          })}
        </nav>

        {/* Settings Panel */}
        <div className="flex-1 bg-white rounded-xl border border-gray-200 p-6">
          {activeSection === 'connection' && (
            <ConnectionSettings settings={settings} onChange={setSettings} />
          )}
          {activeSection === 'query' && (
            <QuerySettings settings={settings} onChange={setSettings} />
          )}
          {activeSection === 'preagg' && (
            <PreAggSettings settings={settings} onChange={setSettings} />
          )}
          {activeSection === 'security' && (
            <SecuritySettings settings={settings} onChange={setSettings} />
          )}
          {activeSection === 'performance' && (
            <PerformanceSettings settings={settings} onChange={setSettings} />
          )}
        </div>
      </div>
    </div>
  );
}

interface SettingsPanelProps {
  settings: CubeSettings;
  onChange: (settings: CubeSettings) => void;
}

function ConnectionSettings({ settings, onChange }: SettingsPanelProps) {
  return (
    <div className="space-y-6">
      <div>
        <h2 className="text-lg font-semibold text-gray-900 mb-4">Connection Settings</h2>
        <p className="text-gray-500 text-sm mb-6">
          Configure how your application connects to the Cube.js API.
        </p>
      </div>

      <FormField
        label="Cube API URL"
        description="The base URL of your Cube.js API endpoint"
      >
        <input
          type="text"
          value={settings.cubeApiUrl}
          onChange={(e) => onChange({ ...settings, cubeApiUrl: e.target.value })}
          className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-indigo-500"
          placeholder="http://localhost:4000"
        />
      </FormField>

      <FormField
        label="API Secret"
        description="Secret key for signing JWT tokens"
      >
        <div className="flex gap-2">
          <input
            type="password"
            value={settings.cubeApiSecret}
            onChange={(e) => onChange({ ...settings, cubeApiSecret: e.target.value })}
            className="flex-1 px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-indigo-500"
          />
          <button className="px-3 py-2 border border-gray-300 rounded-lg hover:bg-gray-50 transition-colors">
            Regenerate
          </button>
        </div>
      </FormField>

      <div className="pt-4 border-t border-gray-200">
        <button className="px-4 py-2 border border-gray-300 text-gray-700 rounded-lg hover:bg-gray-50 transition-colors">
          Test Connection
        </button>
      </div>
    </div>
  );
}

function QuerySettings({ settings, onChange }: SettingsPanelProps) {
  return (
    <div className="space-y-6">
      <div>
        <h2 className="text-lg font-semibold text-gray-900 mb-4">Query Settings</h2>
        <p className="text-gray-500 text-sm mb-6">
          Configure query execution limits and caching behavior.
        </p>
      </div>

      <FormField
        label="Query Timeout (seconds)"
        description="Maximum time allowed for query execution"
      >
        <input
          type="number"
          value={settings.queryTimeout}
          onChange={(e) => onChange({ ...settings, queryTimeout: parseInt(e.target.value) })}
          className="w-32 px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-indigo-500"
        />
      </FormField>

      <FormField
        label="Max Rows per Query"
        description="Limit the number of rows returned per query"
      >
        <input
          type="number"
          value={settings.maxRowsPerQuery}
          onChange={(e) => onChange({ ...settings, maxRowsPerQuery: parseInt(e.target.value) })}
          className="w-32 px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-indigo-500"
        />
      </FormField>

      <FormField
        label="Query Caching"
        description="Enable in-memory caching of query results"
      >
        <Toggle
          checked={settings.enableQueryCaching}
          onChange={(checked) => onChange({ ...settings, enableQueryCaching: checked })}
        />
      </FormField>

      {settings.enableQueryCaching && (
        <FormField
          label="Default Cache TTL (seconds)"
          description="Time-to-live for cached query results"
        >
          <input
            type="number"
            value={settings.defaultCacheTtl}
            onChange={(e) => onChange({ ...settings, defaultCacheTtl: parseInt(e.target.value) })}
            className="w-32 px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-indigo-500"
          />
        </FormField>
      )}
    </div>
  );
}

function PreAggSettings({ settings, onChange }: SettingsPanelProps) {
  return (
    <div className="space-y-6">
      <div>
        <h2 className="text-lg font-semibold text-gray-900 mb-4">Pre-Aggregation Settings</h2>
        <p className="text-gray-500 text-sm mb-6">
          Configure automatic pre-aggregation refresh and partitioning.
        </p>
      </div>

      <FormField
        label="Scheduled Refresh"
        description="Enable automatic pre-aggregation refresh"
      >
        <Toggle
          checked={settings.preAggScheduleEnabled}
          onChange={(checked) => onChange({ ...settings, preAggScheduleEnabled: checked })}
        />
      </FormField>

      {settings.preAggScheduleEnabled && (
        <>
          <FormField
            label="Refresh Interval"
            description="How often to check for pre-aggregation updates"
          >
            <select
              value={settings.preAggRefreshInterval}
              onChange={(e) => onChange({ ...settings, preAggRefreshInterval: e.target.value })}
              className="px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-indigo-500"
              aria-label="Refresh interval"
            >
              <option value="5 minutes">Every 5 minutes</option>
              <option value="15 minutes">Every 15 minutes</option>
              <option value="30 minutes">Every 30 minutes</option>
              <option value="1 hour">Every hour</option>
              <option value="6 hours">Every 6 hours</option>
              <option value="1 day">Daily</option>
            </select>
          </FormField>

          <FormField
            label="Partition Granularity"
            description="Default time-based partitioning for pre-aggregations"
          >
            <select
              value={settings.preAggPartitionGranularity}
              onChange={(e) => onChange({ ...settings, preAggPartitionGranularity: e.target.value })}
              className="px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-indigo-500"
              aria-label="Partition granularity"
            >
              <option value="day">Day</option>
              <option value="week">Week</option>
              <option value="month">Month</option>
              <option value="quarter">Quarter</option>
              <option value="year">Year</option>
            </select>
          </FormField>

          <FormField
            label="Refresh Concurrency"
            description="Number of concurrent pre-aggregation builds"
          >
            <input
              type="number"
              value={settings.preAggConcurrency}
              onChange={(e) => onChange({ ...settings, preAggConcurrency: parseInt(e.target.value) })}
              min={1}
              max={16}
              className="w-20 px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-indigo-500"
            />
          </FormField>
        </>
      )}
    </div>
  );
}

function SecuritySettings({ settings, onChange }: SettingsPanelProps) {
  return (
    <div className="space-y-6">
      <div>
        <h2 className="text-lg font-semibold text-gray-900 mb-4">Security Settings</h2>
        <p className="text-gray-500 text-sm mb-6">
          Configure JWT authentication and row-level security.
        </p>
      </div>

      <FormField
        label="JWT Audience"
        description="Expected audience claim in JWT tokens"
      >
        <input
          type="text"
          value={settings.jwtAudience}
          onChange={(e) => onChange({ ...settings, jwtAudience: e.target.value })}
          className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-indigo-500"
        />
      </FormField>

      <FormField
        label="JWT Issuer"
        description="Expected issuer claim in JWT tokens"
      >
        <input
          type="text"
          value={settings.jwtIssuer}
          onChange={(e) => onChange({ ...settings, jwtIssuer: e.target.value })}
          className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-indigo-500"
        />
      </FormField>

      <FormField
        label="Row-Level Security"
        description="Enable automatic tenant filtering on queries"
      >
        <Toggle
          checked={settings.enableRowLevelSecurity}
          onChange={(checked) => onChange({ ...settings, enableRowLevelSecurity: checked })}
        />
      </FormField>

      {settings.enableRowLevelSecurity && (
        <FormField
          label="RLS Context Key"
          description="Security context key used for row filtering"
        >
          <input
            type="text"
            value={settings.rlsContextKey}
            onChange={(e) => onChange({ ...settings, rlsContextKey: e.target.value })}
            className="w-48 px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-indigo-500"
          />
        </FormField>
      )}
    </div>
  );
}

function PerformanceSettings({ settings, onChange }: SettingsPanelProps) {
  return (
    <div className="space-y-6">
      <div>
        <h2 className="text-lg font-semibold text-gray-900 mb-4">Performance Settings</h2>
        <p className="text-gray-500 text-sm mb-6">
          Fine-tune Cube.js performance characteristics.
        </p>
      </div>

      <FormField
        label="Compiler Cache Size"
        description="Maximum number of compiled queries to cache"
      >
        <input
          type="number"
          value={settings.maxCompilerCacheSize}
          onChange={(e) => onChange({ ...settings, maxCompilerCacheSize: parseInt(e.target.value) })}
          className="w-32 px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-indigo-500"
        />
      </FormField>

      <FormField
        label="Scheduled Refresh Timezone"
        description="Timezone for scheduled pre-aggregation refresh"
      >
        <select
          value={settings.scheduledRefreshTimezone}
          onChange={(e) => onChange({ ...settings, scheduledRefreshTimezone: e.target.value })}
          className="px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-indigo-500"
          aria-label="Timezone"
        >
          <option value="UTC">UTC</option>
          <option value="America/New_York">America/New_York</option>
          <option value="America/Los_Angeles">America/Los_Angeles</option>
          <option value="Europe/London">Europe/London</option>
          <option value="Asia/Tokyo">Asia/Tokyo</option>
        </select>
      </FormField>

      <FormField
        label="Debug Logging"
        description="Enable verbose debug logs (not recommended for production)"
      >
        <Toggle
          checked={settings.enableDebugLogs}
          onChange={(checked) => onChange({ ...settings, enableDebugLogs: checked })}
        />
      </FormField>
    </div>
  );
}

interface FormFieldProps {
  label: string;
  description?: string;
  children: React.ReactNode;
}

function FormField({ label, description, children }: FormFieldProps) {
  return (
    <div>
      <label className="block text-sm font-medium text-gray-900 mb-1">{label}</label>
      {description && <p className="text-xs text-gray-500 mb-2">{description}</p>}
      {children}
    </div>
  );
}

function Toggle({ checked, onChange }: { checked: boolean; onChange: (checked: boolean) => void }) {
  return (
    <button
      type="button"
      role="switch"
      aria-checked={checked}
      onClick={() => onChange(!checked)}
      className={`relative inline-flex h-6 w-11 items-center rounded-full transition-colors ${
        checked ? 'bg-indigo-600' : 'bg-gray-200'
      }`}
    >
      <span
        className={`inline-block h-4 w-4 transform rounded-full bg-white transition-transform ${
          checked ? 'translate-x-6' : 'translate-x-1'
        }`}
      />
    </button>
  );
}

// Icons
function LinkIcon({ className }: { className?: string }) {
  return (
    <svg className={className} fill="none" viewBox="0 0 24 24" stroke="currentColor">
      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M13.828 10.172a4 4 0 00-5.656 0l-4 4a4 4 0 105.656 5.656l1.102-1.101m-.758-4.899a4 4 0 005.656 0l4-4a4 4 0 00-5.656-5.656l-1.1 1.1" />
    </svg>
  );
}

function QueryIcon({ className }: { className?: string }) {
  return (
    <svg className={className} fill="none" viewBox="0 0 24 24" stroke="currentColor">
      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M8.228 9c.549-1.165 2.03-2 3.772-2 2.21 0 4 1.343 4 3 0 1.4-1.278 2.575-3.006 2.907-.542.104-.994.54-.994 1.093m0 3h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
    </svg>
  );
}

function LayersIcon({ className }: { className?: string }) {
  return (
    <svg className={className} fill="none" viewBox="0 0 24 24" stroke="currentColor">
      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 11H5m14 0a2 2 0 012 2v6a2 2 0 01-2 2H5a2 2 0 01-2-2v-6a2 2 0 012-2m14 0V9a2 2 0 00-2-2M5 11V9a2 2 0 012-2m0 0V5a2 2 0 012-2h6a2 2 0 012 2v2M7 7h10" />
    </svg>
  );
}

function ShieldIcon({ className }: { className?: string }) {
  return (
    <svg className={className} fill="none" viewBox="0 0 24 24" stroke="currentColor">
      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 12l2 2 4-4m5.618-4.016A11.955 11.955 0 0112 2.944a11.955 11.955 0 01-8.618 3.04A12.02 12.02 0 003 9c0 5.591 3.824 10.29 9 11.622 5.176-1.332 9-6.03 9-11.622 0-1.042-.133-2.052-.382-3.016z" />
    </svg>
  );
}

function SpeedIcon({ className }: { className?: string }) {
  return (
    <svg className={className} fill="none" viewBox="0 0 24 24" stroke="currentColor">
      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M13 10V3L4 14h7v7l9-11h-7z" />
    </svg>
  );
}

function SaveIcon({ className }: { className?: string }) {
  return (
    <svg className={className} fill="none" viewBox="0 0 24 24" stroke="currentColor">
      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M8 7H5a2 2 0 00-2 2v9a2 2 0 002 2h14a2 2 0 002-2V9a2 2 0 00-2-2h-3m-1 4l-3 3m0 0l-3-3m3 3V4" />
    </svg>
  );
}

function LoadingIcon({ className }: { className?: string }) {
  return (
    <svg className={className} fill="none" viewBox="0 0 24 24" stroke="currentColor">
      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
    </svg>
  );
}
