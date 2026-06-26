import React, { useState, useEffect } from 'react';
import { useTenant } from '../../../context/TenantContext';

interface ScheduledReport {
  id: string;
  name: string;
  description: string;
  cube_name: string;
  query: {
    measures: string[];
    dimensions: string[];
    filters?: any[];
  };
  schedule: {
    frequency: 'daily' | 'weekly' | 'monthly';
    day_of_week?: number;
    day_of_month?: number;
    time: string;
    timezone: string;
  };
  delivery: {
    type: 'email' | 'slack' | 's3' | 'webhook';
    recipients?: string[];
    channel?: string;
    bucket?: string;
    url?: string;
  };
  format: 'csv' | 'xlsx' | 'json' | 'pdf';
  status: 'active' | 'paused' | 'error';
  last_run?: string;
  next_run?: string;
  created_by: string;
  created_at: string;
}

export function CubeReportsPage() {
  const { tenant, datasource } = useTenant();
  const [reports, setReports] = useState<ScheduledReport[]>([]);
  const [loading, setLoading] = useState(true);
  const [showCreateModal, setShowCreateModal] = useState(false);

  useEffect(() => {
    if (!tenant?.id || !datasource?.id) return;
    loadReports();
  }, [tenant?.id, datasource?.id]);

  const loadReports = async () => {
    setLoading(true);
    try {
      // Sample data - would call real API
      const sampleReports: ScheduledReport[] = [
        {
          id: '1',
          name: 'Daily Sales Summary',
          description: 'Daily summary of sales metrics by region',
          cube_name: 'Orders',
          query: {
            measures: ['count', 'total_amount', 'avg_amount'],
            dimensions: ['region', 'status'],
          },
          schedule: {
            frequency: 'daily',
            time: '08:00',
            timezone: 'America/New_York',
          },
          delivery: {
            type: 'email',
            recipients: ['sales@company.com', 'analytics@company.com'],
          },
          format: 'xlsx',
          status: 'active',
          last_run: new Date(Date.now() - 86400000).toISOString(),
          next_run: new Date(Date.now() + 28800000).toISOString(),
          created_by: 'admin@company.com',
          created_at: new Date(Date.now() - 30 * 86400000).toISOString(),
        },
        {
          id: '2',
          name: 'Weekly User Growth',
          description: 'User acquisition and retention metrics',
          cube_name: 'Users',
          query: {
            measures: ['count', 'active_count'],
            dimensions: ['plan', 'country'],
          },
          schedule: {
            frequency: 'weekly',
            day_of_week: 1,
            time: '09:00',
            timezone: 'America/New_York',
          },
          delivery: {
            type: 'slack',
            channel: '#analytics',
          },
          format: 'csv',
          status: 'active',
          last_run: new Date(Date.now() - 7 * 86400000).toISOString(),
          next_run: new Date(Date.now() + 2 * 86400000).toISOString(),
          created_by: 'admin@company.com',
          created_at: new Date(Date.now() - 60 * 86400000).toISOString(),
        },
        {
          id: '3',
          name: 'Monthly Revenue Report',
          description: 'Comprehensive monthly revenue breakdown',
          cube_name: 'Revenue',
          query: {
            measures: ['mrr', 'arr', 'churn_rate'],
            dimensions: ['product', 'customer_segment'],
          },
          schedule: {
            frequency: 'monthly',
            day_of_month: 1,
            time: '07:00',
            timezone: 'UTC',
          },
          delivery: {
            type: 's3',
            bucket: 'analytics-exports',
          },
          format: 'json',
          status: 'paused',
          last_run: new Date(Date.now() - 45 * 86400000).toISOString(),
          created_by: 'finance@company.com',
          created_at: new Date(Date.now() - 90 * 86400000).toISOString(),
        },
      ];
      setReports(sampleReports);
    } catch (err) {
      console.error('Failed to load reports:', err);
    } finally {
      setLoading(false);
    }
  };

  if (!tenant || !datasource) {
    return (
      <div className="p-8">
        <div className="bg-yellow-50 border border-yellow-200 rounded-lg p-6 text-center">
          <h2 className="text-lg font-semibold text-yellow-800">Select a Tenant</h2>
          <p className="text-yellow-700 mt-2">
            Please select a tenant and datasource to manage scheduled reports.
          </p>
        </div>
      </div>
    );
  }

  return (
    <div className="p-8">
      {/* Header */}
      <div className="flex items-center justify-between mb-8">
        <div>
          <h1 className="text-2xl font-bold text-gray-900">Scheduled Reports</h1>
          <p className="text-gray-500 mt-1">
            Automate report generation and delivery
          </p>
        </div>
        <button
          onClick={() => setShowCreateModal(true)}
          className="px-4 py-2 bg-indigo-600 text-white rounded-lg hover:bg-indigo-700 transition-colors flex items-center gap-2"
        >
          <PlusIcon className="w-5 h-5" />
          New Report
        </button>
      </div>

      {/* Premium Feature Banner */}
      <div className="bg-gradient-to-r from-indigo-500 to-purple-600 rounded-xl p-6 mb-8 text-white">
        <div className="flex items-center gap-4">
          <div className="p-3 bg-white/20 rounded-lg">
            <SparklesIcon className="w-8 h-8" />
          </div>
          <div>
            <h3 className="text-lg font-semibold">Premium Feature: Scheduled Reports</h3>
            <p className="text-white/80 mt-1">
              Automatically generate and deliver reports via email, Slack, S3, or webhooks.
              Available in Enterprise tier.
            </p>
          </div>
        </div>
      </div>

      {/* Stats */}
      <div className="grid grid-cols-4 gap-6 mb-8">
        <StatCard
          label="Total Reports"
          value={reports.length}
          icon={ReportIcon}
          color="indigo"
        />
        <StatCard
          label="Active"
          value={reports.filter((r) => r.status === 'active').length}
          icon={CheckIcon}
          color="green"
        />
        <StatCard
          label="Paused"
          value={reports.filter((r) => r.status === 'paused').length}
          icon={PauseIcon}
          color="yellow"
        />
        <StatCard
          label="With Errors"
          value={reports.filter((r) => r.status === 'error').length}
          icon={ErrorIcon}
          color="red"
        />
      </div>

      {/* Reports List */}
      {loading ? (
        <LoadingSkeleton />
      ) : reports.length === 0 ? (
        <EmptyState onCreateClick={() => setShowCreateModal(true)} />
      ) : (
        <div className="space-y-4">
          {reports.map((report) => (
            <ReportCard key={report.id} report={report} />
          ))}
        </div>
      )}

      {/* Create Modal Placeholder */}
      {showCreateModal && (
        <div className="fixed inset-0 z-50 flex items-center justify-center">
          <div className="absolute inset-0 bg-black/40" onClick={() => setShowCreateModal(false)} />
          <div className="relative bg-white rounded-xl shadow-xl w-full max-w-2xl p-6">
            <h2 className="text-xl font-semibold text-gray-900 mb-6">Create Scheduled Report</h2>
            <p className="text-gray-500 mb-6">
              Configure a new automated report with custom queries, schedules, and delivery options.
            </p>
            <div className="bg-gray-50 rounded-lg p-8 text-center">
              <ReportIcon className="w-12 h-12 text-gray-300 mx-auto mb-4" />
              <p className="text-gray-500">Report builder coming soon</p>
              <p className="text-sm text-gray-400">
                Full query builder, schedule configuration, and delivery setup
              </p>
            </div>
            <div className="flex justify-end gap-3 mt-6">
              <button
                onClick={() => setShowCreateModal(false)}
                className="px-4 py-2 border border-gray-300 text-gray-700 rounded-lg hover:bg-gray-50 transition-colors"
              >
                Cancel
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}

function ReportCard({ report }: { report: ScheduledReport }) {
  const statusConfig = {
    active: { label: 'Active', className: 'bg-green-100 text-green-700', icon: CheckIcon },
    paused: { label: 'Paused', className: 'bg-yellow-100 text-yellow-700', icon: PauseIcon },
    error: { label: 'Error', className: 'bg-red-100 text-red-700', icon: ErrorIcon },
  };

  const deliveryIcons = {
    email: EmailIcon,
    slack: SlackIcon,
    s3: S3Icon,
    webhook: WebhookIcon,
  };

  const status = statusConfig[report.status];
  const DeliveryIcon = deliveryIcons[report.delivery.type];

  const frequencyText = {
    daily: `Daily at ${report.schedule.time}`,
    weekly: `Weekly on ${['Sun', 'Mon', 'Tue', 'Wed', 'Thu', 'Fri', 'Sat'][report.schedule.day_of_week || 0]} at ${report.schedule.time}`,
    monthly: `Monthly on day ${report.schedule.day_of_month} at ${report.schedule.time}`,
  };

  return (
    <div className="bg-white rounded-xl border border-gray-200 p-6">
      <div className="flex items-start justify-between mb-4">
        <div className="flex items-start gap-4">
          <div className="p-2 rounded-lg bg-indigo-100 text-indigo-600">
            <ReportIcon className="w-6 h-6" />
          </div>
          <div>
            <div className="flex items-center gap-3">
              <h3 className="font-semibold text-gray-900">{report.name}</h3>
              <span className={`px-2 py-0.5 text-xs rounded ${status.className}`}>
                {status.label}
              </span>
            </div>
            <p className="text-sm text-gray-500 mt-1">{report.description}</p>
          </div>
        </div>
        <div className="flex items-center gap-2">
          <button className="px-3 py-1.5 text-sm border border-gray-300 rounded-lg hover:bg-gray-50 transition-colors">
            Run Now
          </button>
          <button className="px-3 py-1.5 text-sm text-indigo-600 hover:text-indigo-700">
            Edit
          </button>
        </div>
      </div>

      <div className="grid grid-cols-4 gap-6 text-sm">
        {/* Query Info */}
        <div>
          <h4 className="text-xs font-medium text-gray-500 uppercase mb-2">Query</h4>
          <p className="text-gray-900">Cube: {report.cube_name}</p>
          <p className="text-gray-600">
            {report.query.measures.length} measures, {report.query.dimensions.length} dimensions
          </p>
        </div>

        {/* Schedule */}
        <div>
          <h4 className="text-xs font-medium text-gray-500 uppercase mb-2">Schedule</h4>
          <p className="text-gray-900">{frequencyText[report.schedule.frequency]}</p>
          <p className="text-gray-600">{report.schedule.timezone}</p>
        </div>

        {/* Delivery */}
        <div>
          <h4 className="text-xs font-medium text-gray-500 uppercase mb-2">Delivery</h4>
          <div className="flex items-center gap-2">
            <DeliveryIcon className="w-4 h-4 text-gray-400" />
            <span className="text-gray-900 capitalize">{report.delivery.type}</span>
          </div>
          <p className="text-gray-600">{report.format.toUpperCase()}</p>
        </div>

        {/* Timing */}
        <div>
          <h4 className="text-xs font-medium text-gray-500 uppercase mb-2">Status</h4>
          {report.last_run && (
            <p className="text-gray-600">
              Last: {new Date(report.last_run).toLocaleDateString()}
            </p>
          )}
          {report.next_run && (
            <p className="text-gray-900">
              Next: {new Date(report.next_run).toLocaleDateString()}
            </p>
          )}
        </div>
      </div>
    </div>
  );
}

interface StatCardProps {
  label: string;
  value: number;
  icon: React.FC<{ className?: string }>;
  color: 'indigo' | 'green' | 'yellow' | 'red';
}

function StatCard({ label, value, icon: Icon, color }: StatCardProps) {
  const colorClasses = {
    indigo: 'bg-indigo-50 text-indigo-600',
    green: 'bg-green-50 text-green-600',
    yellow: 'bg-yellow-50 text-yellow-600',
    red: 'bg-red-50 text-red-600',
  };

  return (
    <div className="bg-white rounded-xl border border-gray-200 p-6">
      <div className={`p-2 rounded-lg inline-flex ${colorClasses[color]}`}>
        <Icon className="w-5 h-5" />
      </div>
      <p className="text-2xl font-bold text-gray-900 mt-4">{value}</p>
      <p className="text-sm text-gray-500">{label}</p>
    </div>
  );
}

function EmptyState({ onCreateClick }: { onCreateClick: () => void }) {
  return (
    <div className="text-center py-12 bg-white rounded-xl border border-gray-200">
      <ReportIcon className="w-12 h-12 text-gray-300 mx-auto mb-4" />
      <h3 className="text-lg font-medium text-gray-900">No scheduled reports</h3>
      <p className="text-gray-500 mt-1">Create your first automated report</p>
      <button
        onClick={onCreateClick}
        className="mt-4 px-4 py-2 bg-indigo-600 text-white rounded-lg hover:bg-indigo-700 transition-colors"
      >
        Create Report
      </button>
    </div>
  );
}

function LoadingSkeleton() {
  return (
    <div className="space-y-4">
      {[...Array(3)].map((_, i) => (
        <div key={i} className="bg-white rounded-xl border p-6 animate-pulse">
          <div className="flex items-center gap-4 mb-4">
            <div className="w-10 h-10 bg-gray-200 rounded-lg" />
            <div className="h-5 w-48 bg-gray-200 rounded" />
          </div>
          <div className="grid grid-cols-4 gap-6">
            {[...Array(4)].map((_, j) => (
              <div key={j} className="space-y-2">
                <div className="h-3 w-16 bg-gray-100 rounded" />
                <div className="h-4 w-24 bg-gray-200 rounded" />
              </div>
            ))}
          </div>
        </div>
      ))}
    </div>
  );
}

// Icons
function PlusIcon({ className }: { className?: string }) {
  return (
    <svg className={className} fill="none" viewBox="0 0 24 24" stroke="currentColor">
      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 4v16m8-8H4" />
    </svg>
  );
}

function SparklesIcon({ className }: { className?: string }) {
  return (
    <svg className={className} fill="none" viewBox="0 0 24 24" stroke="currentColor">
      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 3v4M3 5h4M6 17v4m-2-2h4m5-16l2.286 6.857L21 12l-5.714 2.143L13 21l-2.286-6.857L5 12l5.714-2.143L13 3z" />
    </svg>
  );
}

function ReportIcon({ className }: { className?: string }) {
  return (
    <svg className={className} fill="none" viewBox="0 0 24 24" stroke="currentColor">
      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 17v-2m3 2v-4m3 4v-6m2 10H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z" />
    </svg>
  );
}

function CheckIcon({ className }: { className?: string }) {
  return (
    <svg className={className} fill="none" viewBox="0 0 24 24" stroke="currentColor">
      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" />
    </svg>
  );
}

function PauseIcon({ className }: { className?: string }) {
  return (
    <svg className={className} fill="none" viewBox="0 0 24 24" stroke="currentColor">
      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M10 9v6m4-6v6m7-3a9 9 0 11-18 0 9 9 0 0118 0z" />
    </svg>
  );
}

function ErrorIcon({ className }: { className?: string }) {
  return (
    <svg className={className} fill="none" viewBox="0 0 24 24" stroke="currentColor">
      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 8v4m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
    </svg>
  );
}

function EmailIcon({ className }: { className?: string }) {
  return (
    <svg className={className} fill="none" viewBox="0 0 24 24" stroke="currentColor">
      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M3 8l7.89 5.26a2 2 0 002.22 0L21 8M5 19h14a2 2 0 002-2V7a2 2 0 00-2-2H5a2 2 0 00-2 2v10a2 2 0 002 2z" />
    </svg>
  );
}

function SlackIcon({ className }: { className?: string }) {
  return (
    <svg className={className} fill="currentColor" viewBox="0 0 24 24">
      <path d="M5.042 15.165a2.528 2.528 0 0 1-2.52 2.523A2.528 2.528 0 0 1 0 15.165a2.527 2.527 0 0 1 2.522-2.52h2.52v2.52zM6.313 15.165a2.527 2.527 0 0 1 2.521-2.52 2.527 2.527 0 0 1 2.521 2.52v6.313A2.528 2.528 0 0 1 8.834 24a2.528 2.528 0 0 1-2.521-2.522v-6.313zM8.834 5.042a2.528 2.528 0 0 1-2.521-2.52A2.528 2.528 0 0 1 8.834 0a2.528 2.528 0 0 1 2.521 2.522v2.52H8.834zM8.834 6.313a2.528 2.528 0 0 1 2.521 2.521 2.528 2.528 0 0 1-2.521 2.521H2.522A2.528 2.528 0 0 1 0 8.834a2.528 2.528 0 0 1 2.522-2.521h6.312zM18.956 8.834a2.528 2.528 0 0 1 2.522-2.521A2.528 2.528 0 0 1 24 8.834a2.528 2.528 0 0 1-2.522 2.521h-2.522V8.834zM17.688 8.834a2.528 2.528 0 0 1-2.523 2.521 2.527 2.527 0 0 1-2.52-2.521V2.522A2.527 2.527 0 0 1 15.165 0a2.528 2.528 0 0 1 2.523 2.522v6.312zM15.165 18.956a2.528 2.528 0 0 1 2.523 2.522A2.528 2.528 0 0 1 15.165 24a2.527 2.527 0 0 1-2.52-2.522v-2.522h2.52zM15.165 17.688a2.527 2.527 0 0 1-2.52-2.523 2.526 2.526 0 0 1 2.52-2.52h6.313A2.527 2.527 0 0 1 24 15.165a2.528 2.528 0 0 1-2.522 2.523h-6.313z" />
    </svg>
  );
}

function S3Icon({ className }: { className?: string }) {
  return (
    <svg className={className} fill="none" viewBox="0 0 24 24" stroke="currentColor">
      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M3 15a4 4 0 004 4h9a5 5 0 10-.1-9.999 5.002 5.002 0 10-9.78 2.096A4.001 4.001 0 003 15z" />
    </svg>
  );
}

function WebhookIcon({ className }: { className?: string }) {
  return (
    <svg className={className} fill="none" viewBox="0 0 24 24" stroke="currentColor">
      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M13.828 10.172a4 4 0 00-5.656 0l-4 4a4 4 0 105.656 5.656l1.102-1.101m-.758-4.899a4 4 0 005.656 0l4-4a4 4 0 00-5.656-5.656l-1.1 1.1" />
    </svg>
  );
}
