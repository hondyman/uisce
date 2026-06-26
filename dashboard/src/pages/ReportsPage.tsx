import React, { useEffect, useState } from 'react'
import { Layout, Card, Button, MetricCard, Spinner, ErrorMessage } from '../components/Layout'
import { apiClient } from '../api/client'

interface Report {
  id: string
  name: string
  type: 'sla' | 'health' | 'predictions' | 'cost'
  status: 'completed' | 'pending' | 'failed'
  created_at: string
  created_by: string
  period: string
  file_url?: string
  download_count?: number
}

export function ReportsPage() {
  const tenantId = localStorage.getItem('tenant_id') || 'default'
  const [reports, setReports] = useState<Report[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [selectedType, setSelectedType] = useState<'all' | 'sla' | 'health' | 'predictions' | 'cost'>('all')
  const [showScheduleModal, setShowScheduleModal] = useState(false)

  const fetchReports = async () => {
    try {
      setLoading(true)
      setError(null)
      
      // Mock data - in real implementation, this would come from the API
      const mockReports: Report[] = [
        {
          id: 'rpt-001',
          name: 'Weekly SLA Summary',
          type: 'sla',
          status: 'completed',
          created_at: new Date(Date.now() - 7 * 24 * 3600000).toISOString(),
          created_by: 'system',
          period: 'Last 7 days',
          file_url: '/reports/sla-summary-2024-01.pdf',
          download_count: 45
        },
        {
          id: 'rpt-002',
          name: 'Chain Health Report - January',
          type: 'health',
          status: 'completed',
          created_at: new Date(Date.now() - 1 * 24 * 3600000).toISOString(),
          created_by: 'system',
          period: 'January 2024',
          file_url: '/reports/health-jan-2024.pdf',
          download_count: 23
        },
        {
          id: 'rpt-003',
          name: 'Failure Predictions',
          type: 'predictions',
          status: 'pending',
          created_at: new Date(Date.now() - 2 * 3600000).toISOString(),
          created_by: 'ml-engine',
          period: 'Current',
          download_count: 0
        },
        {
          id: 'rpt-004',
          name: 'Monthly Cost Analysis',
          type: 'cost',
          status: 'completed',
          created_at: new Date(Date.now() - 30 * 24 * 3600000).toISOString(),
          created_by: 'analytics',
          period: 'December 2023',
          file_url: '/reports/cost-dec-2023.pdf',
          download_count: 12
        }
      ]

      setReports(mockReports)
    } catch (err) {
      console.error('Failed to fetch reports:', err)
      setError(err instanceof Error ? err.message : 'Failed to load reports')
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    fetchReports()
  }, [])

  const filteredReports = reports.filter(r => selectedType === 'all' || r.type === selectedType)

  const stats = {
    total: reports.length,
    completed: reports.filter(r => r.status === 'completed').length,
    pending: reports.filter(r => r.status === 'pending').length,
    avgDownloads: reports.filter(r => r.status === 'completed').length > 0
      ? (reports.filter(r => r.status === 'completed').reduce((sum, r) => sum + (r.download_count || 0), 0) / 
         reports.filter(r => r.status === 'completed').length).toFixed(1)
      : '0'
  }

  if (loading) return <Spinner />

  return (
    <Layout sidebar={<ReportsSidebar />} header={<ReportsHeader />}>
      <div className="space-y-8">
        {/* Statistics */}
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6">
          <MetricCard
            label="Total Reports"
            value={stats.total}
            change={25}
            trend="up"
            color="info"
          />
          <MetricCard
            label="Completed"
            value={stats.completed}
            change={10}
            trend="up"
            color="success"
          />
          <MetricCard
            label="In Progress"
            value={stats.pending}
            change={0}
            trend="neutral"
            color="warning"
          />
          <MetricCard
            label="Avg Downloads"
            value={stats.avgDownloads}
            change={15}
            trend="up"
            color="info"
          />
        </div>

        {/* Actions */}
        <div className="flex gap-3">
          <Button variant="primary" onClick={() => setShowScheduleModal(true)}>
            + Schedule Report
          </Button>
          <Button variant="secondary" onClick={fetchReports}>
            Refresh
          </Button>
        </div>

        {/* Filter Tabs */}
        <div className="flex gap-2 border-b border-slate-200">
          {(['all', 'sla', 'health', 'predictions', 'cost'] as const).map(type => (
            <button
              key={type}
              onClick={() => setSelectedType(type)}
              className={`px-4 py-3 font-medium text-sm border-b-2 transition-colors ${
                selectedType === type
                  ? 'border-blue-600 text-blue-600'
                  : 'border-transparent text-slate-600 hover:text-slate-900'
              }`}
            >
              {type.charAt(0).toUpperCase() + type.slice(1)}
            </button>
          ))}
        </div>

        {/* Reports List */}
        <div className="space-y-3">
          {filteredReports.length > 0 ? (
            filteredReports.map(report => (
              <ReportCard key={report.id} report={report} />
            ))
          ) : (
            <Card className="text-center py-12">
              <p className="text-slate-500">No reports found for this filter</p>
            </Card>
          )}
        </div>
      </div>

      {/* Schedule Modal */}
      {showScheduleModal && (
        <ScheduleModal onClose={() => setShowScheduleModal(false)} onSubmit={async (data) => {
          console.log('Schedule report:', data)
          setShowScheduleModal(false)
          fetchReports()
        }} />
      )}
    </Layout>
  )
}

function ReportCard({ report }: { report: Report }) {
  const age = new Date(report.created_at)
  const ageString = getRelativeTime(age)

  const statusColor = {
    completed: 'bg-emerald-100 text-emerald-700',
    pending: 'bg-amber-100 text-amber-700',
    failed: 'bg-red-100 text-red-700'
  }

  const typeIcon = {
    sla: '📊',
    health: '🏥',
    predictions: '🤖',
    cost: '💰'
  }

  return (
    <Card className="flex items-center justify-between p-4 hover:shadow-md transition-shadow">
      <div className="flex-1">
        <div className="flex items-center gap-3 mb-2">
          <span className="text-2xl">{typeIcon[report.type]}</span>
          <div>
            <h3 className="font-medium text-slate-900">{report.name}</h3>
            <p className="text-xs text-slate-600">
              {report.period} • {ageString} • by {report.created_by}
            </p>
          </div>
        </div>
      </div>

      <div className="flex items-center gap-4">
        <div className="text-right">
          <span className={`inline-block px-3 py-1 rounded-full text-xs font-medium ${statusColor[report.status]}`}>
            {report.status.charAt(0).toUpperCase() + report.status.slice(1)}
          </span>
          {report.download_count !== undefined && (
            <p className="text-xs text-slate-600 mt-1">↓ {report.download_count} downloads</p>
          )}
        </div>

        {report.status === 'completed' && report.file_url && (
          <a
            href={report.file_url}
            download
            className="px-4 py-2 bg-blue-600 text-white rounded-lg text-sm font-medium hover:bg-blue-700 transition-colors"
          >
            Download
          </a>
        )}
        {report.status === 'pending' && (
          <div className="w-6 h-6 border-2 border-blue-600 border-t-transparent rounded-full animate-spin"></div>
        )}
      </div>
    </Card>
  )
}

interface ScheduleModalProps {
  onClose: () => void
  onSubmit: (data: { type: string; frequency: string; name: string }) => Promise<void>
}

function ScheduleModal({ onClose, onSubmit }: ScheduleModalProps) {
  const [formData, setFormData] = useState({ type: 'sla', frequency: 'weekly', name: '' })
  const [loading, setLoading] = useState(false)

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setLoading(true)
    try {
      await onSubmit(formData)
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
      <Card className="w-full max-w-md space-y-4">
        <h2 className="text-lg font-bold text-slate-900">Schedule Report</h2>

        <form onSubmit={handleSubmit} className="space-y-4">
          <div>
            <label className="block text-sm font-medium text-slate-700 mb-2">Report Name</label>
            <input
              type="text"
              value={formData.name}
              onChange={(e) => setFormData({ ...formData, name: e.target.value })}
              placeholder="e.g., Weekly SLA Review"
              className="w-full px-3 py-2 border border-slate-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500"
              required
            />
          </div>

          <div>
            <label className="block text-sm font-medium text-slate-700 mb-2">Report Type</label>
            <select
              value={formData.type}
              onChange={(e) => setFormData({ ...formData, type: e.target.value })}
              className="w-full px-3 py-2 border border-slate-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500"
            >
              <option value="sla">SLA Compliance</option>
              <option value="health">Chain Health</option>
              <option value="predictions">Failure Predictions</option>
              <option value="cost">Cost Analysis</option>
            </select>
          </div>

          <div>
            <label className="block text-sm font-medium text-slate-700 mb-2">Frequency</label>
            <select
              value={formData.frequency}
              onChange={(e) => setFormData({ ...formData, frequency: e.target.value })}
              className="w-full px-3 py-2 border border-slate-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500"
            >
              <option value="daily">Daily</option>
              <option value="weekly">Weekly</option>
              <option value="monthly">Monthly</option>
            </select>
          </div>

          <div className="flex gap-3 pt-4">
            <Button variant="secondary" onClick={onClose}>
              Cancel
            </Button>
            <Button variant="primary" type="submit" disabled={loading}>
              {loading ? 'Scheduling...' : 'Schedule'}
            </Button>
          </div>
        </form>
      </Card>
    </div>
  )
}

function getRelativeTime(date: Date): string {
  const seconds = Math.floor((Date.now() - date.getTime()) / 1000)
  if (seconds < 60) return 'Just now'
  if (seconds < 3600) return `${Math.floor(seconds / 60)}m ago`
  if (seconds < 86400) return `${Math.floor(seconds / 3600)}h ago`
  return `${Math.floor(seconds / 86400)}d ago`
}

function ReportsSidebar() {
  return (
    <nav className="p-6 space-y-4">
      <a href="/" className="block px-4 py-2 rounded-lg hover:bg-brand-light">
        Dashboard
      </a>
      <a href="/chains" className="block px-4 py-2 rounded-lg hover:bg-brand-light">
        Chains
      </a>
      <a href="/feed" className="block px-4 py-2 rounded-lg hover:bg-brand-light">
        Live Feed
      </a>
      <a href="/reports" className="block px-4 py-2 rounded-lg bg-blue-600 text-white">
        Reports
      </a>
    </nav>
  )
}

function ReportsHeader() {
  return (
    <div className="px-8 py-4 flex items-center justify-between border-b border-slate-200">
      <h1 className="text-2xl font-bold text-slate-900">Analytics Reports</h1>
    </div>
  )
}
