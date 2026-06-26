/**
 * AdvancedScheduling.tsx
 * 
 * Sophisticated rule scheduling and trigger configuration:
 * - Cron-based scheduling with visual builder
 * - Event-driven triggers (webhooks, database changes, file uploads)
 * - Dependency chains between rules
 * - Time-window constraints
 * - Parallel/sequential execution modes
 * - Retry policies and dead-letter queues
 */

import React, { useState, useCallback } from 'react';
import {
  Calendar,
  Clock,
  Zap,
  GitBranch,
  Play,
  Pause,
  Trash2,
  ChevronRight,
  ChevronDown,
  AlertCircle,
  Check,
  Settings,
  Database,
  Webhook,
  FileUp,
  Timer,
  Repeat,
  Link2,
  ArrowRight
} from 'lucide-react';

// ============================================================================
// Types
// ============================================================================

interface ScheduleConfig {
  id: string;
  type: 'cron' | 'interval' | 'event' | 'dependency';
  name: string;
  enabled: boolean;
  
  // Cron scheduling
  cronExpression?: string;
  cronTimezone?: string;
  
  // Interval scheduling
  intervalMinutes?: number;
  
  // Event triggers
  eventTriggers?: EventTrigger[];
  
  // Dependency triggers
  dependsOn?: DependencyConfig[];
  
  // Time window constraints
  timeWindow?: TimeWindowConfig;
  
  // Execution settings
  executionMode: 'sequential' | 'parallel';
  maxConcurrent: number;
  timeout: number;
  retryPolicy: RetryPolicy;
  
  // Metadata
  lastRun?: Date;
  nextRun?: Date;
  runCount: number;
  failCount: number;
}

interface EventTrigger {
  id: string;
  type: 'webhook' | 'database_change' | 'file_upload' | 'message_queue' | 'api_call';
  name: string;
  config: Record<string, unknown>;
  filter?: string; // CEL expression for filtering events
  enabled: boolean;
}

interface DependencyConfig {
  ruleId: string;
  ruleName: string;
  waitForSuccess: boolean;
  maxWaitMinutes: number;
}

interface TimeWindowConfig {
  enabled: boolean;
  startTime: string; // HH:mm
  endTime: string;   // HH:mm
  daysOfWeek: number[]; // 0-6, Sunday = 0
  timezone: string;
  skipOutsideWindow: boolean;
}

interface RetryPolicy {
  maxRetries: number;
  backoffType: 'fixed' | 'exponential' | 'linear';
  initialDelaySeconds: number;
  maxDelaySeconds: number;
  deadLetterQueue?: string;
}

interface AdvancedSchedulingProps {
  ruleId: string;
  ruleName: string;
  initialConfig?: ScheduleConfig;
  availableRules: Array<{ id: string; name: string }>;
  onChange: (config: ScheduleConfig) => void;
}

// ============================================================================
// Constants
// ============================================================================

const DAYS_OF_WEEK = [
  { value: 0, label: 'Sun' },
  { value: 1, label: 'Mon' },
  { value: 2, label: 'Tue' },
  { value: 3, label: 'Wed' },
  { value: 4, label: 'Thu' },
  { value: 5, label: 'Fri' },
  { value: 6, label: 'Sat' }
];

const COMMON_CRON_PRESETS = [
  { label: 'Every minute', value: '* * * * *' },
  { label: 'Every 5 minutes', value: '*/5 * * * *' },
  { label: 'Every 15 minutes', value: '*/15 * * * *' },
  { label: 'Every hour', value: '0 * * * *' },
  { label: 'Daily at midnight', value: '0 0 * * *' },
  { label: 'Daily at 6am', value: '0 6 * * *' },
  { label: 'Weekly on Monday', value: '0 0 * * 1' },
  { label: 'First of month', value: '0 0 1 * *' },
  { label: 'Weekdays at 9am', value: '0 9 * * 1-5' }
];

const TIMEZONES = [
  'America/New_York',
  'America/Chicago',
  'America/Denver',
  'America/Los_Angeles',
  'Europe/London',
  'Europe/Paris',
  'Asia/Tokyo',
  'Asia/Shanghai',
  'Australia/Sydney',
  'UTC'
];

// ============================================================================
// Helper Functions
// ============================================================================

const generateId = () => `schedule-${Date.now()}-${Math.random().toString(36).substr(2, 9)}`;

const describeCron = (cron: string): string => {
  const parts = cron.split(' ');
  if (parts.length !== 5) return 'Invalid cron expression';
  
  const preset = COMMON_CRON_PRESETS.find(p => p.value === cron);
  if (preset) return preset.label;
  
  return 'Custom schedule';
};

const getDefaultConfig = (ruleId: string, ruleName: string): ScheduleConfig => ({
  id: generateId(),
  type: 'cron',
  name: `${ruleName} Schedule`,
  enabled: true,
  cronExpression: '0 * * * *',
  cronTimezone: 'UTC',
  executionMode: 'sequential',
  maxConcurrent: 1,
  timeout: 300,
  retryPolicy: {
    maxRetries: 3,
    backoffType: 'exponential',
    initialDelaySeconds: 30,
    maxDelaySeconds: 300
  },
  runCount: 0,
  failCount: 0
});

// ============================================================================
// Components
// ============================================================================

const CronBuilder: React.FC<{
  value: string;
  onChange: (value: string) => void;
  timezone: string;
  onTimezoneChange: (tz: string) => void;
}> = ({ value, onChange, timezone, onTimezoneChange }) => {
  const [showAdvanced, setShowAdvanced] = useState(false);
  
  return (
    <div className="space-y-3">
      <div>
        <label className="block text-sm font-medium text-gray-700 mb-2">
          Schedule Preset
        </label>
        <div className="flex flex-wrap gap-2">
          {COMMON_CRON_PRESETS.map(preset => (
            <button
              key={preset.value}
              onClick={() => onChange(preset.value)}
              className={`px-3 py-1.5 text-sm rounded border transition-colors ${
                value === preset.value
                  ? 'bg-blue-600 text-white border-blue-600'
                  : 'bg-white text-gray-700 border-gray-300 hover:border-blue-400'
              }`}
            >
              {preset.label}
            </button>
          ))}
        </div>
      </div>
      
      <div className="flex items-center gap-2">
        <button
          onClick={() => setShowAdvanced(!showAdvanced)}
          className="text-sm text-blue-600 hover:text-blue-700 flex items-center gap-1"
        >
          {showAdvanced ? <ChevronDown size={14} /> : <ChevronRight size={14} />}
          Advanced (Custom Cron)
        </button>
      </div>
      
      {showAdvanced && (
        <div className="bg-gray-50 rounded p-3 space-y-3">
          <div>
            <label className="block text-xs font-medium text-gray-500 mb-1">
              Cron Expression
            </label>
            <input
              type="text"
              value={value}
              onChange={(e) => onChange(e.target.value)}
              placeholder="* * * * *"
              className="w-full px-3 py-2 border rounded font-mono text-sm"
            />
            <p className="text-xs text-gray-500 mt-1">
              Format: minute hour day-of-month month day-of-week
            </p>
          </div>
          
          <div className="text-sm">
            <span className="text-gray-500">Schedule: </span>
            <span className="font-medium">{describeCron(value)}</span>
          </div>
        </div>
      )}
      
      <div>
        <label className="block text-sm font-medium text-gray-700 mb-1">
          Timezone
        </label>
        <select
          value={timezone}
          onChange={(e) => onTimezoneChange(e.target.value)}
          className="w-full px-3 py-2 border rounded text-sm"
          title="Select timezone"
          aria-label="Select timezone"
        >
          {TIMEZONES.map(tz => (
            <option key={tz} value={tz}>{tz}</option>
          ))}
        </select>
      </div>
    </div>
  );
};

const EventTriggerBuilder: React.FC<{
  triggers: EventTrigger[];
  onChange: (triggers: EventTrigger[]) => void;
}> = ({ triggers, onChange }) => {
  const addTrigger = useCallback((type: EventTrigger['type']) => {
    const newTrigger: EventTrigger = {
      id: generateId(),
      type,
      name: `${type.replace('_', ' ')} trigger`,
      config: {},
      enabled: true
    };
    onChange([...triggers, newTrigger]);
  }, [triggers, onChange]);

  const removeTrigger = useCallback((id: string) => {
    onChange(triggers.filter(t => t.id !== id));
  }, [triggers, onChange]);

  const updateTrigger = useCallback((id: string, updates: Partial<EventTrigger>) => {
    onChange(triggers.map(t => t.id === id ? { ...t, ...updates } : t));
  }, [triggers, onChange]);

  const getTypeIcon = (type: EventTrigger['type']) => {
    switch (type) {
      case 'webhook': return <Webhook size={16} />;
      case 'database_change': return <Database size={16} />;
      case 'file_upload': return <FileUp size={16} />;
      case 'message_queue': return <Zap size={16} />;
      case 'api_call': return <ArrowRight size={16} />;
    }
  };

  return (
    <div className="space-y-3">
      <div className="flex items-center gap-2">
        <span className="text-sm font-medium text-gray-700">Event Triggers</span>
        <div className="flex-1" />
        <div className="flex gap-1">
          {(['webhook', 'database_change', 'file_upload', 'message_queue', 'api_call'] as const).map(type => (
            <button
              key={type}
              onClick={() => addTrigger(type)}
              className="p-1.5 text-gray-400 hover:text-gray-600 hover:bg-gray-100 rounded"
              title={`Add ${type.replace('_', ' ')} trigger`}
              aria-label={`Add ${type.replace('_', ' ')} trigger`}
            >
              {getTypeIcon(type)}
            </button>
          ))}
        </div>
      </div>
      
      {triggers.length === 0 ? (
        <div className="text-center py-6 text-gray-500 text-sm bg-gray-50 rounded border border-dashed">
          No event triggers configured. Click an icon above to add one.
        </div>
      ) : (
        <div className="space-y-2">
          {triggers.map(trigger => (
            <div
              key={trigger.id}
              className={`border rounded p-3 ${trigger.enabled ? 'bg-white' : 'bg-gray-50 opacity-60'}`}
            >
              <div className="flex items-center gap-3">
                <div className={`p-2 rounded ${trigger.enabled ? 'bg-blue-100 text-blue-600' : 'bg-gray-200 text-gray-500'}`}>
                  {getTypeIcon(trigger.type)}
                </div>
                <div className="flex-1">
                  <input
                    type="text"
                    value={trigger.name}
                    onChange={(e) => updateTrigger(trigger.id, { name: e.target.value })}
                    className="font-medium text-gray-900 bg-transparent border-none p-0 w-full"
                    title="Trigger name"
                  />
                  <span className="text-xs text-gray-500 capitalize">
                    {trigger.type.replace('_', ' ')}
                  </span>
                </div>
                <button
                  onClick={() => updateTrigger(trigger.id, { enabled: !trigger.enabled })}
                  className={`p-1 rounded ${trigger.enabled ? 'text-green-600' : 'text-gray-400'}`}
                  title={trigger.enabled ? 'Disable trigger' : 'Enable trigger'}
                >
                  {trigger.enabled ? <Check size={16} /> : <Pause size={16} />}
                </button>
                <button
                  onClick={() => removeTrigger(trigger.id)}
                  className="p-1 text-gray-400 hover:text-red-500 rounded"
                  title="Remove trigger"
                  aria-label="Remove trigger"
                >
                  <Trash2 size={16} />
                </button>
              </div>
              
              {trigger.type === 'webhook' && (
                <div className="mt-3 pt-3 border-t">
                  <label className="block text-xs text-gray-500 mb-1">Webhook URL</label>
                  <input
                    type="text"
                    value={(trigger.config.url as string) || ''}
                    onChange={(e) => updateTrigger(trigger.id, { config: { ...trigger.config, url: e.target.value } })}
                    placeholder="https://..."
                    className="w-full px-2 py-1 border rounded text-sm"
                  />
                </div>
              )}
              
              {trigger.type === 'database_change' && (
                <div className="mt-3 pt-3 border-t grid grid-cols-2 gap-2">
                  <div>
                    <label className="block text-xs text-gray-500 mb-1">Table</label>
                    <input
                      type="text"
                      value={(trigger.config.table as string) || ''}
                      onChange={(e) => updateTrigger(trigger.id, { config: { ...trigger.config, table: e.target.value } })}
                      placeholder="table_name"
                      className="w-full px-2 py-1 border rounded text-sm"
                    />
                  </div>
                  <div>
                    <label className="block text-xs text-gray-500 mb-1">Operations</label>
                    <select
                      value={(trigger.config.operation as string) || 'all'}
                      onChange={(e) => updateTrigger(trigger.id, { config: { ...trigger.config, operation: e.target.value } })}
                      className="w-full px-2 py-1 border rounded text-sm"
                      title="Select operation"
                      aria-label="Select operation"
                    >
                      <option value="all">All Changes</option>
                      <option value="insert">INSERT Only</option>
                      <option value="update">UPDATE Only</option>
                      <option value="delete">DELETE Only</option>
                    </select>
                  </div>
                </div>
              )}
            </div>
          ))}
        </div>
      )}
    </div>
  );
};

const TimeWindowBuilder: React.FC<{
  config: TimeWindowConfig | undefined;
  onChange: (config: TimeWindowConfig) => void;
}> = ({ config, onChange }) => {
  const defaultConfig: TimeWindowConfig = {
    enabled: false,
    startTime: '09:00',
    endTime: '17:00',
    daysOfWeek: [1, 2, 3, 4, 5],
    timezone: 'America/New_York',
    skipOutsideWindow: true
  };
  
  const current = config || defaultConfig;
  
  const toggleDay = useCallback((day: number) => {
    const newDays = current.daysOfWeek.includes(day)
      ? current.daysOfWeek.filter(d => d !== day)
      : [...current.daysOfWeek, day].sort();
    onChange({ ...current, daysOfWeek: newDays });
  }, [current, onChange]);

  return (
    <div className="space-y-3">
      <label className="flex items-center gap-2">
        <input
          type="checkbox"
          checked={current.enabled}
          onChange={(e) => onChange({ ...current, enabled: e.target.checked })}
          className="rounded"
        />
        <span className="text-sm font-medium text-gray-700">
          Enable Time Window Constraint
        </span>
      </label>
      
      {current.enabled && (
        <div className="pl-6 space-y-3">
          <div className="flex items-center gap-3">
            <div>
              <label className="block text-xs text-gray-500 mb-1">Start Time</label>
              <input
                type="time"
                value={current.startTime}
                onChange={(e) => onChange({ ...current, startTime: e.target.value })}
                className="px-2 py-1 border rounded text-sm"
                title="Start time"
              />
            </div>
            <span className="mt-5 text-gray-400">to</span>
            <div>
              <label className="block text-xs text-gray-500 mb-1">End Time</label>
              <input
                type="time"
                value={current.endTime}
                onChange={(e) => onChange({ ...current, endTime: e.target.value })}
                className="px-2 py-1 border rounded text-sm"
                title="End time"
              />
            </div>
          </div>
          
          <div>
            <label className="block text-xs text-gray-500 mb-2">Days of Week</label>
            <div className="flex gap-1">
              {DAYS_OF_WEEK.map(day => (
                <button
                  key={day.value}
                  onClick={() => toggleDay(day.value)}
                  className={`w-10 h-10 rounded text-sm font-medium transition-colors ${
                    current.daysOfWeek.includes(day.value)
                      ? 'bg-blue-600 text-white'
                      : 'bg-gray-100 text-gray-600 hover:bg-gray-200'
                  }`}
                >
                  {day.label}
                </button>
              ))}
            </div>
          </div>
          
          <div>
            <label className="block text-xs text-gray-500 mb-1">Timezone</label>
            <select
              value={current.timezone}
              onChange={(e) => onChange({ ...current, timezone: e.target.value })}
              className="px-2 py-1 border rounded text-sm"
              title="Select timezone"
              aria-label="Select timezone"
            >
              {TIMEZONES.map(tz => (
                <option key={tz} value={tz}>{tz}</option>
              ))}
            </select>
          </div>
          
          <label className="flex items-center gap-2">
            <input
              type="checkbox"
              checked={current.skipOutsideWindow}
              onChange={(e) => onChange({ ...current, skipOutsideWindow: e.target.checked })}
              className="rounded"
            />
            <span className="text-sm text-gray-600">
              Skip executions outside time window (don't queue)
            </span>
          </label>
        </div>
      )}
    </div>
  );
};

const RetryPolicyBuilder: React.FC<{
  policy: RetryPolicy;
  onChange: (policy: RetryPolicy) => void;
}> = ({ policy, onChange }) => (
  <div className="space-y-3">
    <h4 className="text-sm font-medium text-gray-700 flex items-center gap-2">
      <Repeat size={16} />
      Retry Policy
    </h4>
    
    <div className="grid grid-cols-2 gap-3">
      <div>
        <label className="block text-xs text-gray-500 mb-1">Max Retries</label>
        <input
          type="number"
          value={policy.maxRetries}
          onChange={(e) => onChange({ ...policy, maxRetries: parseInt(e.target.value) || 0 })}
          min={0}
          max={10}
          className="w-full px-2 py-1 border rounded text-sm"
          title="Maximum number of retries"
        />
      </div>
      <div>
        <label className="block text-xs text-gray-500 mb-1">Backoff Type</label>
        <select
          value={policy.backoffType}
          onChange={(e) => onChange({ ...policy, backoffType: e.target.value as RetryPolicy['backoffType'] })}
          className="w-full px-2 py-1 border rounded text-sm"
          title="Backoff type"
          aria-label="Backoff type"
        >
          <option value="fixed">Fixed Delay</option>
          <option value="linear">Linear Backoff</option>
          <option value="exponential">Exponential Backoff</option>
        </select>
      </div>
      <div>
        <label className="block text-xs text-gray-500 mb-1">Initial Delay (sec)</label>
        <input
          type="number"
          value={policy.initialDelaySeconds}
          onChange={(e) => onChange({ ...policy, initialDelaySeconds: parseInt(e.target.value) || 10 })}
          min={1}
          className="w-full px-2 py-1 border rounded text-sm"
          title="Initial delay in seconds"
        />
      </div>
      <div>
        <label className="block text-xs text-gray-500 mb-1">Max Delay (sec)</label>
        <input
          type="number"
          value={policy.maxDelaySeconds}
          onChange={(e) => onChange({ ...policy, maxDelaySeconds: parseInt(e.target.value) || 300 })}
          min={1}
          className="w-full px-2 py-1 border rounded text-sm"
          title="Maximum delay in seconds"
        />
      </div>
    </div>
  </div>
);

// ============================================================================
// Main Component
// ============================================================================

export const AdvancedScheduling: React.FC<AdvancedSchedulingProps> = ({
  ruleId,
  ruleName,
  initialConfig,
  availableRules,
  onChange
}) => {
  const [config, setConfig] = useState<ScheduleConfig>(
    initialConfig || getDefaultConfig(ruleId, ruleName)
  );
  const [expandedSection, setExpandedSection] = useState<string | null>('schedule');

  const handleChange = useCallback((updates: Partial<ScheduleConfig>) => {
    const newConfig = { ...config, ...updates };
    setConfig(newConfig);
    onChange(newConfig);
  }, [config, onChange]);

  const toggleSection = useCallback((section: string) => {
    setExpandedSection(prev => prev === section ? null : section);
  }, []);

  return (
    <div className="bg-white rounded-lg border shadow-sm">
      {/* Header */}
      <div className="flex items-center justify-between p-4 border-b">
        <div className="flex items-center gap-3">
          <div className="p-2 bg-purple-100 rounded-lg">
            <Calendar size={20} className="text-purple-600" />
          </div>
          <div>
            <h3 className="font-semibold text-gray-900">Schedule & Triggers</h3>
            <p className="text-xs text-gray-500">Configure when and how this rule executes</p>
          </div>
        </div>
        
        <div className="flex items-center gap-2">
          <button
            onClick={() => handleChange({ enabled: !config.enabled })}
            className={`flex items-center gap-2 px-3 py-1.5 rounded text-sm font-medium transition-colors ${
              config.enabled
                ? 'bg-green-100 text-green-700'
                : 'bg-gray-100 text-gray-600'
            }`}
          >
            {config.enabled ? <Play size={14} /> : <Pause size={14} />}
            {config.enabled ? 'Active' : 'Paused'}
          </button>
        </div>
      </div>
      
      {/* Schedule Type Selector */}
      <div className="p-4 border-b bg-gray-50">
        <label className="block text-sm font-medium text-gray-700 mb-2">
          Trigger Type
        </label>
        <div className="flex gap-2">
          {[
            { value: 'cron', label: 'Scheduled', icon: Calendar },
            { value: 'interval', label: 'Interval', icon: Timer },
            { value: 'event', label: 'Event-Driven', icon: Zap },
            { value: 'dependency', label: 'After Rule', icon: GitBranch }
          ].map(type => (
            <button
              key={type.value}
              onClick={() => handleChange({ type: type.value as ScheduleConfig['type'] })}
              className={`flex items-center gap-2 px-4 py-2 rounded border transition-colors ${
                config.type === type.value
                  ? 'bg-blue-600 text-white border-blue-600'
                  : 'bg-white text-gray-700 border-gray-300 hover:border-blue-400'
              }`}
            >
              <type.icon size={16} />
              {type.label}
            </button>
          ))}
        </div>
      </div>
      
      {/* Schedule Configuration */}
      <div className="divide-y">
        {/* Primary Schedule Section */}
        <div>
          <button
            onClick={() => toggleSection('schedule')}
            className="w-full flex items-center justify-between p-4 hover:bg-gray-50 transition-colors"
          >
            <span className="font-medium text-gray-900 flex items-center gap-2">
              <Clock size={16} />
              Schedule Configuration
            </span>
            {expandedSection === 'schedule' ? <ChevronDown size={18} /> : <ChevronRight size={18} />}
          </button>
          
          {expandedSection === 'schedule' && (
            <div className="px-4 pb-4">
              {config.type === 'cron' && (
                <CronBuilder
                  value={config.cronExpression || '0 * * * *'}
                  onChange={(value) => handleChange({ cronExpression: value })}
                  timezone={config.cronTimezone || 'UTC'}
                  onTimezoneChange={(tz) => handleChange({ cronTimezone: tz })}
                />
              )}
              
              {config.type === 'interval' && (
                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-2">
                    Run every
                  </label>
                  <div className="flex items-center gap-2">
                    <input
                      type="number"
                      value={config.intervalMinutes || 60}
                      onChange={(e) => handleChange({ intervalMinutes: parseInt(e.target.value) || 60 })}
                      min={1}
                      className="w-24 px-3 py-2 border rounded text-sm"
                      title="Interval in minutes"
                    />
                    <span className="text-gray-600">minutes</span>
                  </div>
                </div>
              )}
              
              {config.type === 'event' && (
                <EventTriggerBuilder
                  triggers={config.eventTriggers || []}
                  onChange={(triggers) => handleChange({ eventTriggers: triggers })}
                />
              )}
              
              {config.type === 'dependency' && (
                <div className="space-y-3">
                  <label className="block text-sm font-medium text-gray-700">
                    Run after these rules complete
                  </label>
                  
                  {(config.dependsOn || []).map((dep, i) => (
                    <div key={dep.ruleId} className="flex items-center gap-2 bg-gray-50 rounded p-2">
                      <Link2 size={14} className="text-gray-400" />
                      <span className="flex-1 text-sm">{dep.ruleName}</span>
                      <label className="flex items-center gap-1 text-xs">
                        <input
                          type="checkbox"
                          checked={dep.waitForSuccess}
                          onChange={(e) => {
                            const newDeps = [...(config.dependsOn || [])];
                            newDeps[i] = { ...dep, waitForSuccess: e.target.checked };
                            handleChange({ dependsOn: newDeps });
                          }}
                          className="rounded"
                        />
                        Wait for success
                      </label>
                      <button
                        onClick={() => {
                          handleChange({ dependsOn: (config.dependsOn || []).filter((_, j) => j !== i) });
                        }}
                        className="p-1 text-gray-400 hover:text-red-500"
                        title="Remove dependency"
                        aria-label="Remove dependency"
                      >
                        <Trash2 size={14} />
                      </button>
                    </div>
                  ))}
                  
                  <select
                    value=""
                    onChange={(e) => {
                      if (e.target.value) {
                        const rule = availableRules.find(r => r.id === e.target.value);
                        if (rule) {
                          handleChange({
                            dependsOn: [...(config.dependsOn || []), {
                              ruleId: rule.id,
                              ruleName: rule.name,
                              waitForSuccess: true,
                              maxWaitMinutes: 60
                            }]
                          });
                        }
                      }
                    }}
                    className="w-full px-3 py-2 border rounded text-sm"
                    title="Add dependency"
                    aria-label="Add dependency"
                  >
                    <option value="">+ Add dependency...</option>
                    {availableRules
                      .filter(r => r.id !== ruleId && !(config.dependsOn || []).some(d => d.ruleId === r.id))
                      .map(rule => (
                        <option key={rule.id} value={rule.id}>{rule.name}</option>
                      ))
                    }
                  </select>
                </div>
              )}
            </div>
          )}
        </div>
        
        {/* Time Window Section */}
        <div>
          <button
            onClick={() => toggleSection('timeWindow')}
            className="w-full flex items-center justify-between p-4 hover:bg-gray-50 transition-colors"
          >
            <span className="font-medium text-gray-900 flex items-center gap-2">
              <Timer size={16} />
              Time Window
              {config.timeWindow?.enabled && (
                <span className="text-xs px-2 py-0.5 bg-blue-100 text-blue-700 rounded">Active</span>
              )}
            </span>
            {expandedSection === 'timeWindow' ? <ChevronDown size={18} /> : <ChevronRight size={18} />}
          </button>
          
          {expandedSection === 'timeWindow' && (
            <div className="px-4 pb-4">
              <TimeWindowBuilder
                config={config.timeWindow}
                onChange={(tw) => handleChange({ timeWindow: tw })}
              />
            </div>
          )}
        </div>
        
        {/* Execution Settings Section */}
        <div>
          <button
            onClick={() => toggleSection('execution')}
            className="w-full flex items-center justify-between p-4 hover:bg-gray-50 transition-colors"
          >
            <span className="font-medium text-gray-900 flex items-center gap-2">
              <Settings size={16} />
              Execution Settings
            </span>
            {expandedSection === 'execution' ? <ChevronDown size={18} /> : <ChevronRight size={18} />}
          </button>
          
          {expandedSection === 'execution' && (
            <div className="px-4 pb-4 space-y-4">
              <div className="grid grid-cols-3 gap-4">
                <div>
                  <label className="block text-xs text-gray-500 mb-1">Execution Mode</label>
                  <select
                    value={config.executionMode}
                    onChange={(e) => handleChange({ executionMode: e.target.value as 'sequential' | 'parallel' })}
                    className="w-full px-2 py-1 border rounded text-sm"
                    title="Execution mode"
                    aria-label="Execution mode"
                  >
                    <option value="sequential">Sequential</option>
                    <option value="parallel">Parallel</option>
                  </select>
                </div>
                <div>
                  <label className="block text-xs text-gray-500 mb-1">Max Concurrent</label>
                  <input
                    type="number"
                    value={config.maxConcurrent}
                    onChange={(e) => handleChange({ maxConcurrent: parseInt(e.target.value) || 1 })}
                    min={1}
                    max={100}
                    className="w-full px-2 py-1 border rounded text-sm"
                    title="Maximum concurrent executions"
                    disabled={config.executionMode === 'sequential'}
                  />
                </div>
                <div>
                  <label className="block text-xs text-gray-500 mb-1">Timeout (seconds)</label>
                  <input
                    type="number"
                    value={config.timeout}
                    onChange={(e) => handleChange({ timeout: parseInt(e.target.value) || 300 })}
                    min={1}
                    className="w-full px-2 py-1 border rounded text-sm"
                    title="Timeout in seconds"
                  />
                </div>
              </div>
              
              <RetryPolicyBuilder
                policy={config.retryPolicy}
                onChange={(policy) => handleChange({ retryPolicy: policy })}
              />
            </div>
          )}
        </div>
      </div>
      
      {/* Status Footer */}
      {(config.lastRun || config.runCount > 0) && (
        <div className="flex items-center justify-between p-4 border-t bg-gray-50 text-sm">
          <div className="flex items-center gap-4">
            {config.lastRun && (
              <span className="text-gray-500">
                Last run: {new Date(config.lastRun).toLocaleString()}
              </span>
            )}
            <span className="text-gray-500">
              Total runs: {config.runCount}
            </span>
            {config.failCount > 0 && (
              <span className="text-red-600 flex items-center gap-1">
                <AlertCircle size={14} />
                {config.failCount} failures
              </span>
            )}
          </div>
          {config.nextRun && (
            <span className="text-blue-600">
              Next run: {new Date(config.nextRun).toLocaleString()}
            </span>
          )}
        </div>
      )}
    </div>
  );
};

export default AdvancedScheduling;
