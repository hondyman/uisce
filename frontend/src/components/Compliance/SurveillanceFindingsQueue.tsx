import React, { useCallback, useEffect, useState } from 'react';
import {
  ShieldAlert,
  AlertTriangle,
  CheckCircle2,
  Clock,
  Search,
  RefreshCw,
  Eye,
  Sliders,
  X,
  History,
  Activity,
  ArrowRight,
  UserCheck,
  Check,
  AlertOctagon,
} from 'lucide-react';

export interface SurveillanceFinding {
  id: string;
  tenant_id: string;
  detector_type: string;
  severity: 'LOW' | 'MEDIUM' | 'HIGH' | 'CRITICAL';
  status: 'OPEN' | 'IN_REVIEW' | 'ESCALATED' | 'DISMISSED' | 'REMEDIATED' | 'CLOSED';
  dedup_key: string;
  title: string;
  description: string;
  entity_id?: string;
  entity_type: string;
  metadata: any;
  detected_at: string;
  activity_window_start: string;
  activity_window_end: string;
  assigned_to?: string;
  resolution_notes?: string;
  resolved_by?: string;
  resolved_at?: string;
  created_at: string;
  updated_at: string;
}

export interface SurveillanceEvent {
  id: string;
  finding_id: string;
  tenant_id: string;
  event_type: string;
  actor: string;
  payload: any;
  created_at: string;
}

export const SurveillanceFindingsQueue: React.FC = () => {
  const [findings, setFindings] = useState<SurveillanceFinding[]>([]);
  const [loading, setLoading] = useState(false);
  const [detectorFilter, setDetectorFilter] = useState('ALL');
  const [severityFilter, setSeverityFilter] = useState('HIGH_AND_CRITICAL');
  const [statusFilter, setStatusFilter] = useState('UNADDRESSED');
  const [searchQuery, setSearchQuery] = useState('');

  const [selectedFinding, setSelectedFinding] = useState<SurveillanceFinding | null>(null);
  const [findingEvents, setFindingEvents] = useState<SurveillanceEvent[]>([]);
  const [detailLoading, setDetailLoading] = useState(false);

  // Status transition modal
  const [transitionModalOpen, setTransitionModalOpen] = useState(false);
  const [targetStatus, setTargetStatus] = useState<string>('IN_REVIEW');
  const [resolutionNotes, setResolutionNotes] = useState('');
  const [assignedTo, setAssignedTo] = useState('');
  const [actionError, setActionError] = useState<string | null>(null);

  const fetchFindings = useCallback(async () => {
    setLoading(true);
    try {
      const tenantId = localStorage.getItem('tenant_id') || '99e99e99-99e9-49e9-89e9-99e99e99e999';
      const token = localStorage.getItem('AUTH_TOKEN') || localStorage.getItem('auth_token') || '';

      let url = '/api/compliance/surveillance/findings';
      if (statusFilter === 'UNADDRESSED') {
        url += '/unaddressed';
      } else if (statusFilter !== 'ALL') {
        url += `?status=${statusFilter}`;
      }

      const res = await fetch(url, {
        headers: {
          'Authorization': token ? `Bearer ${token}` : '',
          'X-Tenant-ID': tenantId,
        },
      });
      if (res.ok) {
        const json = await res.json();
        setFindings(json.data || []);
      }
    } catch (err) {
      console.error('Failed to fetch surveillance findings', err);
    } finally {
      setLoading(false);
    }
  }, [statusFilter]);

  useEffect(() => {
    fetchFindings();
  }, [fetchFindings]);

  const loadFindingDetails = async (f: SurveillanceFinding) => {
    setSelectedFinding(f);
    setDetailLoading(true);
    setActionError(null);
    try {
      const tenantId = localStorage.getItem('tenant_id') || '99e99e99-99e9-49e9-89e9-99e99e99e999';
      const token = localStorage.getItem('AUTH_TOKEN') || localStorage.getItem('auth_token') || '';
      const res = await fetch(`/api/compliance/surveillance/findings/${f.id}`, {
        headers: {
          'Authorization': token ? `Bearer ${token}` : '',
          'X-Tenant-ID': tenantId,
        },
      });
      if (res.ok) {
        const json = await res.json();
        setSelectedFinding(json.finding);
        setFindingEvents(json.events || []);
      }
    } catch (err) {
      console.error('Failed to load finding details', err);
    } finally {
      setDetailLoading(false);
    }
  };

  const handleStatusTransition = async () => {
    if (!selectedFinding) return;
    try {
      const tenantId = localStorage.getItem('tenant_id') || '99e99e99-99e9-49e9-89e9-99e99e99e999';
      const token = localStorage.getItem('AUTH_TOKEN') || localStorage.getItem('auth_token') || '';

      const res = await fetch(`/api/compliance/surveillance/findings/${selectedFinding.id}/status`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'Authorization': token ? `Bearer ${token}` : '',
          'X-Tenant-ID': tenantId,
        },
        body: JSON.stringify({
          status: targetStatus,
          resolution_notes: resolutionNotes,
          assigned_to: assignedTo,
        }),
      });

      if (res.ok) {
        setTransitionModalOpen(false);
        setResolutionNotes('');
        fetchFindings();
        loadFindingDetails(selectedFinding);
      } else {
        const errTxt = await res.text();
        setActionError(`Transition failed: ${errTxt}`);
      }
    } catch (err: any) {
      setActionError(`Transition error: ${err.message}`);
    }
  };

  const filteredFindings = findings.filter(f => {
    if (detectorFilter !== 'ALL' && f.detector_type !== detectorFilter) return false;
    if (severityFilter === 'HIGH_AND_CRITICAL') {
      if (f.severity !== 'HIGH' && f.severity !== 'CRITICAL') return false;
    } else if (severityFilter !== 'ALL' && f.severity !== severityFilter) {
      return false;
    }
    if (!searchQuery) return true;
    const q = searchQuery.toLowerCase();
    return (
      f.title.toLowerCase().includes(q) ||
      f.description.toLowerCase().includes(q) ||
      f.dedup_key.toLowerCase().includes(q) ||
      (f.entity_id && f.entity_id.toLowerCase().includes(q))
    );
  });

  const getSeverityBadge = (sev: string) => {
    switch (sev) {
      case 'CRITICAL':
        return (
          <span className="inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-[11px] font-bold bg-rose-500/20 text-rose-400 border border-rose-500/40 animate-pulse">
            <AlertOctagon className="w-3 h-3" />
            CRITICAL
          </span>
        );
      case 'HIGH':
        return (
          <span className="inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-[11px] font-semibold bg-orange-500/20 text-orange-400 border border-orange-500/40">
            <AlertTriangle className="w-3 h-3" />
            HIGH
          </span>
        );
      case 'MEDIUM':
        return (
          <span className="inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-[11px] font-semibold bg-amber-500/20 text-amber-300 border border-amber-500/30">
            MEDIUM
          </span>
        );
      case 'LOW':
      default:
        return (
          <span className="inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-[11px] font-medium bg-slate-800 text-slate-300">
            LOW
          </span>
        );
    }
  };

  return (
    <div className="flex h-full w-full bg-slate-950 text-slate-100 overflow-hidden font-sans">
      {/* Main Findings Queue */}
      <div className="flex-1 flex flex-col min-w-0 border-r border-slate-800/80">
        {/* Header */}
        <div className="p-4 border-b border-slate-800 bg-slate-900/60 flex flex-col gap-3">
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-3">
              <div className="p-2 bg-rose-500/10 border border-rose-500/30 rounded-lg text-rose-400">
                <ShieldAlert className="w-5 h-5" />
              </div>
              <div>
                <h1 className="text-base font-bold text-white tracking-wide flex items-center gap-2">
                  Post-Trade Streaming Surveillance Findings
                  <span className="text-xs px-2 py-0.5 rounded-full bg-rose-950/60 border border-rose-700/50 text-rose-300">
                    CDC Real-Time
                  </span>
                </h1>
                <p className="text-xs text-slate-400">
                  Wash sales (IRS 1091), pro-rata allocation fairness, and cross-account conflict monitoring
                </p>
              </div>
            </div>

            <button
              onClick={fetchFindings}
              disabled={loading}
              className="px-3 py-1.5 rounded-lg bg-slate-800 hover:bg-slate-700 text-xs text-slate-200 border border-slate-700 flex items-center gap-1.5 transition"
            >
              <RefreshCw className={`w-3.5 h-3.5 ${loading ? 'animate-spin text-indigo-400' : ''}`} />
              Refresh
            </button>
          </div>

          {/* Filters */}
          <div className="flex items-center justify-between gap-3">
            <div className="relative flex-1 max-w-md">
              <Search className="w-4 h-4 text-slate-400 absolute left-3 top-1/2 -translate-y-1/2" />
              <input
                type="text"
                placeholder="Search title, dedup key, account/security ID..."
                value={searchQuery}
                onChange={e => setSearchQuery(e.target.value)}
                className="w-full bg-slate-900 border border-slate-700/80 rounded-lg pl-9 pr-3 py-1.5 text-xs text-slate-200 placeholder-slate-500 focus:outline-none focus:border-rose-500"
              />
            </div>

            <div className="flex items-center gap-2">
              <select
                value={detectorFilter}
                onChange={e => setDetectorFilter(e.target.value)}
                className="bg-slate-900 border border-slate-700/80 rounded-lg px-2.5 py-1.5 text-xs text-slate-200 focus:outline-none focus:border-rose-500"
              >
                <option value="ALL">All Detectors</option>
                <option value="WASH_SALE">Wash Sale (1091)</option>
                <option value="PRO_RATA_ALLOCATION_FAIRNESS">Allocation Fairness</option>
                <option value="CROSS_ACCOUNT_CONFLICT">Cross-Account Conflict</option>
              </select>

              <select
                value={severityFilter}
                onChange={e => setSeverityFilter(e.target.value)}
                className="bg-slate-900 border border-slate-700/80 rounded-lg px-2.5 py-1.5 text-xs text-slate-200 focus:outline-none focus:border-rose-500"
              >
                <option value="HIGH_AND_CRITICAL">High & Critical (Default)</option>
                <option value="CRITICAL">Critical Only</option>
                <option value="HIGH">High Only</option>
                <option value="MEDIUM">Medium</option>
                <option value="LOW">Low</option>
                <option value="ALL">All Severities</option>
              </select>

              <select
                value={statusFilter}
                onChange={e => setStatusFilter(e.target.value)}
                className="bg-slate-900 border border-slate-700/80 rounded-lg px-2.5 py-1.5 text-xs text-slate-200 focus:outline-none focus:border-rose-500"
              >
                <option value="UNADDRESSED">Unaddressed Only</option>
                <option value="ALL">All Statuses</option>
                <option value="OPEN">OPEN</option>
                <option value="IN_REVIEW">IN_REVIEW</option>
                <option value="ESCALATED">ESCALATED</option>
                <option value="REMEDIATED">REMEDIATED</option>
                <option value="DISMISSED">DISMISSED</option>
              </select>
            </div>
          </div>
        </div>

        {/* Findings Table */}
        <div className="flex-1 overflow-auto">
          <table className="w-full text-left text-xs border-collapse">
            <thead className="sticky top-0 bg-slate-900/95 backdrop-blur z-10 border-b border-slate-800 text-slate-400 font-semibold uppercase tracking-wider">
              <tr>
                <th className="py-2.5 px-4">Severity</th>
                <th className="py-2.5 px-4">Detector</th>
                <th className="py-2.5 px-4">Finding & Summary</th>
                <th className="py-2.5 px-4">Activity Window</th>
                <th className="py-2.5 px-4">Status</th>
                <th className="py-2.5 px-4">Detected At</th>
                <th className="py-2.5 px-4 text-right">Actions</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-slate-800/60">
              {filteredFindings.map(f => {
                const isSelected = selectedFinding?.id === f.id;
                const actStart = new Date(f.activity_window_start).toLocaleDateString();
                const actEnd = new Date(f.activity_window_end).toLocaleDateString();
                const detectedAt = new Date(f.detected_at).toLocaleTimeString();

                return (
                  <tr
                    key={f.id}
                    onClick={() => loadFindingDetails(f)}
                    className={`cursor-pointer transition hover:bg-slate-900/80 ${
                      isSelected ? 'bg-rose-950/20 border-l-2 border-rose-500' : ''
                    }`}
                  >
                    <td className="py-3 px-4">{getSeverityBadge(f.severity)}</td>
                    <td className="py-3 px-4">
                      <span className="px-2 py-0.5 rounded bg-slate-800 text-slate-300 font-mono text-[11px]">
                        {f.detector_type}
                      </span>
                    </td>
                    <td className="py-3 px-4 max-w-xs">
                      <div className="font-semibold text-slate-200 truncate">{f.title}</div>
                      <div className="text-slate-400 text-[11px] truncate">{f.description}</div>
                    </td>
                    <td className="py-3 px-4 font-mono text-[11px] text-slate-400 whitespace-nowrap">
                      {actStart} → {actEnd}
                    </td>
                    <td className="py-3 px-4">
                      <span className="px-2 py-0.5 rounded text-[11px] font-semibold bg-slate-800 text-slate-300 border border-slate-700">
                        {f.status}
                      </span>
                    </td>
                    <td className="py-3 px-4 font-mono text-slate-400 text-[11px] whitespace-nowrap">
                      {detectedAt}
                    </td>
                    <td className="py-3 px-4 text-right">
                      <button
                        onClick={(e) => {
                          e.stopPropagation();
                          loadFindingDetails(f);
                        }}
                        className="p-1.5 rounded hover:bg-slate-800 text-slate-300 hover:text-rose-400 transition"
                      >
                        <Eye className="w-4 h-4" />
                      </button>
                    </td>
                  </tr>
                );
              })}

              {filteredFindings.length === 0 && !loading && (
                <tr>
                  <td colSpan={7} className="py-12 text-center text-slate-500">
                    No post-trade surveillance findings matching current filters.
                  </td>
                </tr>
              )}
            </tbody>
          </table>
        </div>
      </div>

      {/* Finding Detail & Audit Trail Inspector */}
      {selectedFinding && (
        <div className="w-[500px] flex flex-col bg-slate-900/90 border-l border-slate-800 min-w-0">
          <div className="p-4 border-b border-slate-800 flex items-center justify-between bg-slate-900">
            <div className="flex items-center gap-2">
              <ShieldAlert className="w-4 h-4 text-rose-400" />
              <h2 className="text-sm font-bold text-white truncate">
                {selectedFinding.title}
              </h2>
            </div>
            <button
              onClick={() => setSelectedFinding(null)}
              className="p-1 rounded hover:bg-slate-800 text-slate-400 hover:text-slate-200"
            >
              <X className="w-4 h-4" />
            </button>
          </div>

          <div className="flex-1 overflow-auto p-4 space-y-4">
            {actionError && (
              <div className="p-3 bg-rose-500/10 border border-rose-500/30 rounded-lg text-rose-300 text-xs flex items-center gap-2">
                <AlertTriangle className="w-4 h-4 shrink-0" />
                <span>{actionError}</span>
              </div>
            )}

            {/* Severity & Status */}
            <div className="p-3 bg-slate-950/70 border border-slate-800 rounded-lg space-y-2">
              <div className="flex items-center justify-between">
                {getSeverityBadge(selectedFinding.severity)}
                <span className="text-xs font-mono text-slate-400">
                  Status: <strong className="text-slate-200">{selectedFinding.status}</strong>
                </span>
              </div>
              <p className="text-xs text-slate-300 leading-relaxed">{selectedFinding.description}</p>
            </div>

            {/* Temporal Activity Window Card */}
            <div className="p-3 bg-slate-950/70 border border-slate-800 rounded-lg space-y-2">
              <h4 className="text-[11px] font-bold uppercase tracking-wider text-slate-400 flex items-center gap-1.5">
                <Clock className="w-3.5 h-3.5 text-rose-400" />
                Temporal Semantics & Window
              </h4>
              <div className="grid grid-cols-2 gap-2 text-xs">
                <div className="p-2 bg-slate-900 rounded">
                  <div className="text-[10px] text-slate-500">Activity Window Start</div>
                  <div className="font-mono text-slate-200 text-[11px]">
                    {new Date(selectedFinding.activity_window_start).toLocaleString()}
                  </div>
                </div>
                <div className="p-2 bg-slate-900 rounded">
                  <div className="text-[10px] text-slate-500">Activity Window End</div>
                  <div className="font-mono text-slate-200 text-[11px]">
                    {new Date(selectedFinding.activity_window_end).toLocaleString()}
                  </div>
                </div>
              </div>
              <div className="text-[11px] text-slate-400 pt-1">
                Detected at: <span className="font-mono text-slate-300">{new Date(selectedFinding.detected_at).toLocaleString()}</span>
              </div>
            </div>

            {/* Dedup & Metadata */}
            <div className="p-3 bg-slate-950/70 border border-slate-800 rounded-lg space-y-2">
              <h4 className="text-[11px] font-bold uppercase tracking-wider text-slate-400">
                Breach Metadata
              </h4>
              <div className="p-2 bg-slate-900 rounded font-mono text-[10px] text-slate-400 break-all">
                <div><strong>Dedup Key:</strong> {selectedFinding.dedup_key}</div>
                {selectedFinding.entity_id && (
                  <div><strong>Entity ID:</strong> {selectedFinding.entity_id} ({selectedFinding.entity_type})</div>
                )}
              </div>
              <pre className="p-2 bg-slate-900 rounded text-[11px] text-indigo-300 overflow-auto max-h-32">
                {JSON.stringify(selectedFinding.metadata, null, 2)}
              </pre>
            </div>

            {/* Action Bar */}
            <div className="flex items-center gap-2">
              <button
                onClick={() => {
                  setTargetStatus('IN_REVIEW');
                  setTransitionModalOpen(true);
                }}
                className="flex-1 py-2 px-3 rounded-lg bg-sky-600 hover:bg-sky-500 text-white font-semibold text-xs flex items-center justify-center gap-1.5 shadow"
              >
                <Sliders className="w-3.5 h-3.5" />
                Review Finding
              </button>
              <button
                onClick={() => {
                  setTargetStatus('ESCALATED');
                  setTransitionModalOpen(true);
                }}
                className="flex-1 py-2 px-3 rounded-lg bg-rose-600 hover:bg-rose-500 text-white font-semibold text-xs flex items-center justify-center gap-1.5 shadow"
              >
                <AlertTriangle className="w-3.5 h-3.5" />
                Escalate
              </button>
              <button
                onClick={() => {
                  setTargetStatus('DISMISSED');
                  setTransitionModalOpen(true);
                }}
                className="flex-1 py-2 px-3 rounded-lg bg-slate-800 hover:bg-slate-700 text-slate-200 font-semibold text-xs flex items-center justify-center gap-1.5 shadow border border-slate-700"
              >
                <Check className="w-3.5 h-3.5" />
                Dismiss
              </button>
            </div>

            {/* Audit Event Timeline */}
            <div className="space-y-3 pt-2 border-t border-slate-800">
              <h4 className="text-[11px] font-bold uppercase tracking-wider text-slate-400 flex items-center gap-1.5">
                <History className="w-3.5 h-3.5 text-indigo-400" />
                Append-Only Event Ledger
              </h4>
              <div className="space-y-2">
                {findingEvents.map(ev => (
                  <div key={ev.id} className="p-2.5 bg-slate-950/80 border border-slate-800 rounded-lg text-xs space-y-1">
                    <div className="flex items-center justify-between">
                      <span className="font-semibold text-indigo-300 font-mono text-[11px]">{ev.event_type}</span>
                      <span className="text-[10px] text-slate-500 font-mono">
                        {new Date(ev.created_at).toLocaleTimeString()}
                      </span>
                    </div>
                    <div className="text-[11px] text-slate-400">Actor: <span className="text-slate-300">{ev.actor}</span></div>
                  </div>
                ))}
              </div>
            </div>
          </div>
        </div>
      )}

      {/* Transition Modal */}
      {transitionModalOpen && (
        <div className="fixed inset-0 bg-black/70 backdrop-blur-sm z-50 flex items-center justify-center p-4">
          <div className="bg-slate-900 border border-slate-800 rounded-xl max-w-md w-full p-6 space-y-4 shadow-2xl">
            <h3 className="text-base font-bold text-white">Update Finding Status</h3>
            <p className="text-xs text-slate-400">
              Transition finding to <strong>{targetStatus}</strong>
            </p>

            <div className="space-y-3 text-xs">
              <div>
                <label className="block text-slate-300 font-semibold mb-1">Target Status</label>
                <select
                  value={targetStatus}
                  onChange={e => setTargetStatus(e.target.value)}
                  className="w-full bg-slate-950 border border-slate-700 rounded-lg p-2 text-slate-200 focus:outline-none focus:border-rose-500"
                >
                  <option value="IN_REVIEW">IN_REVIEW</option>
                  <option value="ESCALATED">ESCALATED</option>
                  <option value="REMEDIATED">REMEDIATED (Position closed / adjusted)</option>
                  <option value="DISMISSED">DISMISSED (False positive / approved)</option>
                  <option value="CLOSED">CLOSED</option>
                </select>
              </div>

              <div>
                <label className="block text-slate-300 font-semibold mb-1">
                  Resolution Notes {(targetStatus === 'DISMISSED' || targetStatus === 'REMEDIATED' || targetStatus === 'CLOSED') && <span className="text-rose-400">*</span>}
                </label>
                <textarea
                  rows={3}
                  value={resolutionNotes}
                  onChange={e => setResolutionNotes(e.target.value)}
                  placeholder="Document review rationale, trader explanations, or remediation steps..."
                  className="w-full bg-slate-950 border border-slate-700 rounded-lg p-2 text-slate-200 focus:outline-none focus:border-rose-500"
                />
              </div>

              <div>
                <label className="block text-slate-300 font-semibold mb-1">Assign To</label>
                <input
                  type="text"
                  value={assignedTo}
                  onChange={e => setAssignedTo(e.target.value)}
                  placeholder="compliance_officer@firm.com"
                  className="w-full bg-slate-950 border border-slate-700 rounded-lg p-2 text-slate-200 focus:outline-none focus:border-rose-500"
                />
              </div>
            </div>

            <div className="flex items-center justify-end gap-2 pt-2">
              <button
                onClick={() => setTransitionModalOpen(false)}
                className="px-4 py-2 rounded-lg bg-slate-800 hover:bg-slate-700 text-xs text-slate-300"
              >
                Cancel
              </button>
              <button
                onClick={handleStatusTransition}
                className="px-4 py-2 rounded-lg bg-rose-600 hover:bg-rose-500 text-xs text-white font-semibold"
              >
                Confirm Transition
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
};
