import React, { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import {
  ShieldAlert,
  ShieldCheck,
  AlertTriangle,
  Clock,
  Search,
  RefreshCw,
  Eye,
  CheckCircle2,
  XCircle,
  ExternalLink,
  Copy,
  Hash,
  Activity,
  Layers,
  FileText,
  Sliders,
  Radio,
  RadioTower,
  ChevronRight,
  X,
  Sparkles,
} from 'lucide-react';
import { useFdc3 } from '../../services/fdc3/useFdc3';

export interface EvaluationEventRecord {
  id: string;
  lineage_id: string;
  tenant_id: string;
  order_id?: string;
  rule_id: string;
  rule_code: string;
  rule_name: string;
  rule_version: number;
  rule_phase: string;
  severity: string;
  passed: boolean;
  action_taken: 'APPROVED' | 'BLOCKED' | 'WARNED' | 'APPROVAL_PENDING' | 'BYPASSED' | string;
  latency_micros: number;
  rule_content_hash: string;
  evaluation_hash: string;
  input_params: any;
  metric_snapshots: any;
  evaluated_at: string;
  created_at: string;
}

export interface RuleSnapshotInfo {
  rule_id: string;
  version: number;
  content_hash: string;
  resolved_ast: any;
  parameter_thresholds: any;
  citation: string;
}

export interface EvaluatedMetricItem {
  metric_path: string;
  observed_value: any;
  threshold_value?: any;
  operator?: string;
  breached: boolean;
  margin?: string;
}

export interface IntegrityProof {
  content_hash_matches: boolean;
  stored_content_hash: string;
  recomputed_content_hash: string;
  evaluation_hash: string;
  authority: string;
}

export interface DecisionEvidenceBundle {
  evaluation: EvaluationEventRecord;
  rule_snapshot: RuleSnapshotInfo;
  metrics: EvaluatedMetricItem[];
  natural_language_explanation: string;
  integrity_proof: IntegrityProof;
}

export const ComplianceDecisionBlotter: React.FC = () => {
  const [evaluations, setEvaluations] = useState<EvaluationEventRecord[]>([]);
  const [isLoading, setIsLoading] = useState<boolean>(true);
  const [error, setError] = useState<string | null>(null);

  // Filters
  const [searchQuery, setSearchQuery] = useState<string>('');
  const [statusFilter, setStatusFilter] = useState<string>('ALL');
  const [phaseFilter, setPhaseFilter] = useState<string>('ALL');
  const [isLiveWs, setIsLiveWs] = useState<boolean>(true);
  const [wsConnected, setWsConnected] = useState<boolean>(false);

  // Selected Evidence Bundle for Drawer / Modal
  const [selectedLineageId, setSelectedLineageId] = useState<string | null>(null);
  const [evidenceBundle, setEvidenceBundle] = useState<DecisionEvidenceBundle | null>(null);
  const [isEvidenceLoading, setIsEvidenceLoading] = useState<boolean>(false);
  const [evidenceError, setEvidenceError] = useState<string | null>(null);

  // FDC3 Context Integration
  const { activeChannel, broadcast, raiseIntent } = useFdc3();
  const wsRef = useRef<WebSocket | null>(null);

  // Fetch Evaluations from REST API
  const fetchEvaluations = useCallback(async () => {
    setIsLoading(true);
    setError(null);
    try {
      const params = new URLSearchParams();
      params.set('page_size', '100');
      if (statusFilter !== 'ALL') {
        params.set('action_taken', statusFilter);
      }
      if (searchQuery.trim()) {
        params.set('rule_code', searchQuery.trim());
      }

      const tenantId = localStorage.getItem('tenant_id') || '99e99e99-99e9-49e9-89e9-99e99e99e999';
      const token = localStorage.getItem('AUTH_TOKEN') || localStorage.getItem('auth_token') || '';

      const resp = await fetch(`/api/compliance/evaluations?${params.toString()}`, {
        headers: {
          'Authorization': token ? `Bearer ${token}` : '',
          'X-Tenant-ID': tenantId,
        },
      });

      if (!resp.ok) {
        throw new Error(`Server returned ${resp.status} ${resp.statusText}`);
      }

      const json = await resp.json();
      setEvaluations(json.data || []);
    } catch (err: any) {
      console.error('[ComplianceBlotter] Failed to fetch evaluations:', err);
      setError(err?.message || 'Failed to load compliance evaluations');
    } finally {
      setIsLoading(false);
    }
  }, [statusFilter, searchQuery]);

  // Load Evidence Bundle by Lineage ID
  const loadEvidenceBundle = useCallback(async (lineageId: string) => {
    setSelectedLineageId(lineageId);
    setIsEvidenceLoading(true);
    setEvidenceError(null);
    setEvidenceBundle(null);

    try {
      const tenantId = localStorage.getItem('tenant_id') || '99e99e99-99e9-49e9-89e9-99e99e99e999';
      const token = localStorage.getItem('AUTH_TOKEN') || localStorage.getItem('auth_token') || '';

      const resp = await fetch(`/api/compliance/evaluations/${lineageId}`, {
        headers: {
          'Authorization': token ? `Bearer ${token}` : '',
          'X-Tenant-ID': tenantId,
        },
      });

      if (!resp.ok) {
        throw new Error(`Failed to load evidence: ${resp.statusText}`);
      }

      const bundle: DecisionEvidenceBundle = await resp.json();
      setEvidenceBundle(bundle);

      // Auto-broadcast FDC3 context on selection
      if (bundle.evaluation) {
        broadcast({
          type: 'fdc3.order',
          id: {
            orderId: bundle.evaluation.order_id || bundle.evaluation.id,
            lineageId: bundle.evaluation.lineage_id,
            ruleCode: bundle.evaluation.rule_code,
          },
          name: `${bundle.evaluation.rule_code} Compliance Evaluation`,
        });
      }
    } catch (err: any) {
      console.error('[ComplianceBlotter] Failed to load evidence bundle:', err);
      setEvidenceError(err?.message || 'Failed to retrieve decision evidence');
    } finally {
      setIsEvidenceLoading(false);
    }
  }, [broadcast]);

  // Initial Fetch & WebSocket setup
  useEffect(() => {
    fetchEvaluations();
  }, [fetchEvaluations]);

  // Track newest evaluated timestamp for reconnect gap-filling
  const lastEvaluatedAtRef = useRef<string | null>(null);

  // Gap-fill missed evaluations during disconnection
  const performGapFill = useCallback(async (sinceIso: string) => {
    try {
      const tenantId = localStorage.getItem('tenant_id') || '99e99e99-99e9-49e9-89e9-99e99e99e999';
      const token = localStorage.getItem('AUTH_TOKEN') || localStorage.getItem('auth_token') || '';

      const resp = await fetch(`/api/compliance/evaluations?from=${encodeURIComponent(sinceIso)}&page_size=100`, {
        headers: {
          'Authorization': token ? `Bearer ${token}` : '',
          'X-Tenant-ID': tenantId,
        },
      });

      if (resp.ok) {
        const json = await resp.json();
        if (json.data && json.data.length > 0) {
          setEvaluations((prev) => {
            const existingIds = new Set(prev.map((e) => e.id));
            const fresh = json.data.filter((e: EvaluationEventRecord) => !existingIds.has(e.id));
            if (fresh.length > 0) {
              const combined = [...fresh, ...prev].slice(0, 200);
              if (combined.length > 0 && combined[0].evaluated_at) {
                lastEvaluatedAtRef.current = combined[0].evaluated_at;
              }
              return combined;
            }
            return prev;
          });
        }
      }
    } catch (err) {
      console.warn('[ComplianceBlotter] Gap-fill fetch notice:', err);
    }
  }, []);

  // Update newest timestamp when evaluations list updates
  useEffect(() => {
    if (evaluations.length > 0 && evaluations[0].evaluated_at) {
      lastEvaluatedAtRef.current = evaluations[0].evaluated_at;
    }
  }, [evaluations]);

  // Live WebSocket Connection with Auto-Reconnect and Gap-Fill
  useEffect(() => {
    if (!isLiveWs) {
      if (wsRef.current) {
        wsRef.current.close();
        wsRef.current = null;
      }
      setWsConnected(false);
      return;
    }

    let isCancelled = false;
    let reconnectTimer: NodeJS.Timeout | null = null;
    let ws: WebSocket;

    const connect = () => {
      if (isCancelled) return;

      const tenantId = localStorage.getItem('tenant_id') || '99e99e99-99e9-49e9-89e9-99e99e99e999';
      const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
      const wsUrl = `${protocol}//${window.location.host}/api/compliance/evaluations/ws?tenant_id=${tenantId}`;

      try {
        ws = new WebSocket(wsUrl);
        wsRef.current = ws;

        ws.onopen = () => {
          if (isCancelled) return;
          setWsConnected(true);

          // On reconnect: gap-fill from last seen timestamp
          if (lastEvaluatedAtRef.current) {
            performGapFill(lastEvaluatedAtRef.current);
          }
        };

        ws.onmessage = (event) => {
          try {
            const payload = JSON.parse(event.data);
            if (payload.type === 'COMPLIANCE_EVALUATION_EVENT' && payload.data) {
              const newEv: EvaluationEventRecord = payload.data;
              setEvaluations((prev) => {
                if (prev.some((e) => e.id === newEv.id)) return prev;
                if (newEv.evaluated_at) {
                  lastEvaluatedAtRef.current = newEv.evaluated_at;
                }
                return [newEv, ...prev.slice(0, 199)];
              });
            }
          } catch (e) {
            // ignore heartbeat/non-json
          }
        };

        ws.onclose = () => {
          if (isCancelled) return;
          setWsConnected(false);
          // Schedule auto-reconnect in 3s
          reconnectTimer = setTimeout(connect, 3000);
        };

        ws.onerror = () => {
          if (isCancelled) return;
          setWsConnected(false);
        };
      } catch (err) {
        console.warn('[ComplianceBlotter] WebSocket connection error:', err);
        reconnectTimer = setTimeout(connect, 3000);
      }
    };

    connect();

    return () => {
      isCancelled = true;
      if (reconnectTimer) clearTimeout(reconnectTimer);
      if (ws) ws.close();
    };
  }, [isLiveWs, performGapFill]);

  // Filtered rows
  const filteredEvaluations = useMemo(() => {
    return evaluations.filter((ev) => {
      if (statusFilter !== 'ALL' && ev.action_taken !== statusFilter) {
        return false;
      }
      if (phaseFilter !== 'ALL' && ev.rule_phase !== phaseFilter) {
        return false;
      }
      if (searchQuery.trim()) {
        const q = searchQuery.toLowerCase();
        const codeMatch = ev.rule_code?.toLowerCase().includes(q);
        const nameMatch = ev.rule_name?.toLowerCase().includes(q);
        const orderMatch = ev.order_id?.toLowerCase().includes(q);
        const lineageMatch = ev.lineage_id?.toLowerCase().includes(q);
        return codeMatch || nameMatch || orderMatch || lineageMatch;
      }
      return true;
    });
  }, [evaluations, statusFilter, phaseFilter, searchQuery]);

  const copyToClipboard = (text: string) => {
    navigator.clipboard?.writeText(text);
  };

  const getStatusBadge = (action: string) => {
    switch (action) {
      case 'APPROVED':
        return (
          <span className="inline-flex items-center gap-1.5 px-2.5 py-0.5 rounded-full text-xs font-semibold bg-emerald-950/70 text-emerald-400 border border-emerald-800/80">
            <CheckCircle2 className="w-3.5 h-3.5" />
            PASSED
          </span>
        );
      case 'BLOCKED':
        return (
          <span className="inline-flex items-center gap-1.5 px-2.5 py-0.5 rounded-full text-xs font-semibold bg-rose-950/80 text-rose-400 border border-rose-800/80">
            <ShieldAlert className="w-3.5 h-3.5" />
            BLOCKED
          </span>
        );
      case 'WARNED':
        return (
          <span className="inline-flex items-center gap-1.5 px-2.5 py-0.5 rounded-full text-xs font-semibold bg-amber-950/80 text-amber-400 border border-amber-800/80">
            <AlertTriangle className="w-3.5 h-3.5" />
            WARNING
          </span>
        );
      case 'APPROVAL_PENDING':
        return (
          <span className="inline-flex items-center gap-1.5 px-2.5 py-0.5 rounded-full text-xs font-semibold bg-purple-950/80 text-purple-300 border border-purple-800/80">
            <Clock className="w-3.5 h-3.5" />
            PENDING
          </span>
        );
      default:
        return (
          <span className="inline-flex items-center gap-1.5 px-2.5 py-0.5 rounded-full text-xs font-semibold bg-slate-800 text-slate-300 border border-slate-700">
            {action}
          </span>
        );
    }
  };

  return (
    <div className="h-full w-full flex flex-col bg-[#050d1a] text-slate-100 font-sans select-text">
      {/* Top Header & Controls */}
      <div className="bg-[#091428] border-b border-slate-800 px-4 py-3 flex flex-wrap items-center justify-between gap-3 shrink-0">
        <div className="flex items-center gap-3">
          <div className="p-2 rounded-lg bg-indigo-950/60 border border-indigo-700/50 text-indigo-400">
            <ShieldCheck className="w-5 h-5" />
          </div>
          <div>
            <h1 className="text-sm font-bold tracking-wide text-white flex items-center gap-2">
              Pre-Trade Compliance Blotter
              <span className="text-[11px] font-medium px-2 py-0.5 rounded bg-slate-800 text-slate-400 border border-slate-700">
                Cold-Tier Merkle Bound
              </span>
            </h1>
            <p className="text-xs text-slate-400">
              Deterministic real-time decision inspection & RFC 8785 canonical hash provenance
            </p>
          </div>
        </div>

        {/* Right side controls: Live WS indicator + FDC3 status + Refresh */}
        <div className="flex items-center gap-3">
          {/* Live stream pill */}
          <button
            onClick={() => setIsLiveWs(!isLiveWs)}
            className={`flex items-center gap-2 px-3 py-1 rounded-md text-xs font-medium border transition-colors ${
              wsConnected
                ? 'bg-emerald-950/50 text-emerald-300 border-emerald-700/60'
                : isLiveWs
                ? 'bg-amber-950/50 text-amber-300 border-amber-700/60'
                : 'bg-slate-800/80 text-slate-400 border-slate-700'
            }`}
            title="Toggle real-time WebSocket decision feed"
          >
            <RadioTower className={`w-3.5 h-3.5 ${wsConnected ? 'animate-pulse text-emerald-400' : ''}`} />
            {wsConnected ? 'Live Stream Active' : isLiveWs ? 'Connecting Stream...' : 'Stream Paused'}
          </button>

          {/* Active FDC3 Channel */}
          <div className="flex items-center gap-1.5 px-2.5 py-1 rounded bg-slate-900 border border-slate-800 text-xs text-slate-400">
            <span>FDC3 Mesh:</span>
            <strong className="text-sky-400 capitalize">{activeChannel}</strong>
          </div>

          {/* Refresh button */}
          <button
            onClick={fetchEvaluations}
            disabled={isLoading}
            className="flex items-center gap-1.5 px-3 py-1 rounded bg-indigo-600/30 hover:bg-indigo-600/50 text-indigo-200 border border-indigo-500/40 text-xs font-semibold transition-colors disabled:opacity-50 cursor-pointer"
          >
            <RefreshCw className={`w-3.5 h-3.5 ${isLoading ? 'animate-spin' : ''}`} />
            Refresh
          </button>
        </div>
      </div>

      {/* Filter Toolbar */}
      <div className="bg-[#071120] border-b border-slate-800/80 px-4 py-2 flex flex-wrap items-center justify-between gap-3 text-xs shrink-0">
        <div className="flex items-center gap-3 flex-1 min-w-[280px]">
          {/* Search box */}
          <div className="relative flex-1 max-w-sm">
            <Search className="w-3.5 h-3.5 absolute left-2.5 top-1/2 -translate-y-1/2 text-slate-500" />
            <input
              type="text"
              placeholder="Search by rule code, name, or order ID..."
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
              className="w-full bg-[#0d1c33] border border-slate-700/80 rounded pl-8 pr-3 py-1.5 text-xs text-white placeholder-slate-500 focus:outline-none focus:border-indigo-500"
            />
            {searchQuery && (
              <button
                onClick={() => setSearchQuery('')}
                className="absolute right-2 top-1/2 -translate-y-1/2 text-slate-400 hover:text-white"
              >
                <X className="w-3 h-3" />
              </button>
            )}
          </div>

          {/* Status filter buttons */}
          <div className="flex items-center bg-[#0d1c33] border border-slate-700/80 rounded p-0.5">
            {['ALL', 'BLOCKED', 'WARNED', 'APPROVED', 'APPROVAL_PENDING'].map((status) => (
              <button
                key={status}
                onClick={() => setStatusFilter(status)}
                className={`px-2.5 py-1 rounded text-xs font-medium transition-colors ${
                  statusFilter === status
                    ? 'bg-indigo-600 text-white shadow-sm'
                    : 'text-slate-400 hover:text-slate-200'
                }`}
              >
                {status === 'ALL' ? 'All Decisions' : status}
              </button>
            ))}
          </div>
        </div>

        {/* Total rows counter */}
        <div className="text-slate-400 text-xs">
          Showing <span className="text-white font-semibold">{filteredEvaluations.length}</span> evaluations
        </div>
      </div>

      {/* Main Table + Drawer Content Area */}
      <div className="flex-1 flex overflow-hidden relative">
        {/* Blotter Grid Table */}
        <div className="flex-1 overflow-auto">
          {isLoading && evaluations.length === 0 ? (
            <div className="flex flex-col items-center justify-center h-64 text-slate-400 gap-3">
              <RefreshCw className="w-6 h-6 animate-spin text-indigo-400" />
              <p className="text-xs">Loading compliance decisions...</p>
            </div>
          ) : error ? (
            <div className="flex flex-col items-center justify-center h-64 text-rose-400 gap-2">
              <AlertTriangle className="w-8 h-8" />
              <p className="text-xs">{error}</p>
              <button
                onClick={fetchEvaluations}
                className="mt-2 px-3 py-1 rounded bg-slate-800 text-slate-200 text-xs hover:bg-slate-700"
              >
                Retry
              </button>
            </div>
          ) : filteredEvaluations.length === 0 ? (
            <div className="flex flex-col items-center justify-center h-64 text-slate-500 gap-2">
              <ShieldCheck className="w-8 h-8 text-slate-600" />
              <p className="text-xs">No compliance evaluations match current filter criteria.</p>
            </div>
          ) : (
            <table className="w-full border-collapse text-left text-xs">
              <thead className="bg-[#09152a] text-slate-400 uppercase font-semibold text-[11px] sticky top-0 border-b border-slate-800 z-10">
                <tr>
                  <th className="py-2.5 px-3">Status</th>
                  <th className="py-2.5 px-3">Evaluated At</th>
                  <th className="py-2.5 px-3">Order / Reference</th>
                  <th className="py-2.5 px-3">Rule Definition</th>
                  <th className="py-2.5 px-3">Phase / Severity</th>
                  <th className="py-2.5 px-3 text-right">Latency</th>
                  <th className="py-2.5 px-3">Content Hash</th>
                  <th className="py-2.5 px-3 text-center">Action</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-slate-800/60 font-mono">
                {filteredEvaluations.map((ev) => {
                  const isSelected = selectedLineageId === ev.lineage_id;
                  return (
                    <tr
                      key={ev.id}
                      onClick={() => loadEvidenceBundle(ev.lineage_id)}
                      className={`cursor-pointer transition-colors ${
                        isSelected
                          ? 'bg-indigo-950/40 border-l-2 border-indigo-500'
                          : ev.action_taken === 'BLOCKED'
                          ? 'hover:bg-rose-950/20 bg-rose-950/5'
                          : ev.action_taken === 'WARNED'
                          ? 'hover:bg-amber-950/20'
                          : 'hover:bg-slate-800/40'
                      }`}
                    >
                      <td className="py-2.5 px-3 font-sans">
                        {getStatusBadge(ev.action_taken)}
                      </td>
                      <td className="py-2.5 px-3 text-slate-300 whitespace-nowrap">
                        <div title={ev.evaluated_at}>
                          {new Date(ev.evaluated_at).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit', second: '2-digit' })}
                        </div>
                      </td>
                      <td className="py-2.5 px-3 text-sky-400 font-medium">
                        {ev.order_id ? (
                          <div className="flex items-center gap-1" title={ev.order_id}>
                            <span>{ev.order_id.substring(0, 8)}...</span>
                          </div>
                        ) : (
                          <span className="text-slate-500">N/A</span>
                        )}
                      </td>
                      <td className="py-2.5 px-3 font-sans">
                        <div className="flex items-center gap-2">
                          <span className="font-semibold text-white font-mono">{ev.rule_code}</span>
                          <span className="text-[11px] text-slate-400 truncate max-w-[200px]" title={ev.rule_name}>
                            {ev.rule_name}
                          </span>
                          <span className="text-[10px] px-1.5 py-0.2 rounded bg-slate-800 text-slate-300 border border-slate-700 font-mono">
                            v{ev.rule_version}
                          </span>
                        </div>
                      </td>
                      <td className="py-2.5 px-3 text-slate-300 font-sans">
                        <div className="flex items-center gap-1.5 text-[11px]">
                          <span className="text-slate-400">{ev.rule_phase}</span>
                          <span className="text-slate-600">•</span>
                          <span className={`${
                            ev.severity === 'HARD_BLOCK' ? 'text-rose-400' : 'text-amber-400'
                          }`}>
                            {ev.severity}
                          </span>
                        </div>
                      </td>
                      <td className="py-2.5 px-3 text-right text-emerald-400">
                        {ev.latency_micros}µs
                      </td>
                      <td className="py-2.5 px-3 text-slate-400 text-[11px]">
                        <div className="flex items-center gap-1.5" title={`RFC 8785 Canonical Hash: ${ev.rule_content_hash}`}>
                          <Hash className="w-3 h-3 text-slate-500" />
                          <span>{ev.rule_content_hash ? `${ev.rule_content_hash.substring(0, 6)}...${ev.rule_content_hash.substring(ev.rule_content_hash.length - 4)}` : 'N/A'}</span>
                        </div>
                      </td>
                      <td className="py-2.5 px-3 text-center">
                        <button
                          onClick={(e) => {
                            e.stopPropagation();
                            loadEvidenceBundle(ev.lineage_id);
                          }}
                          className="px-2 py-1 rounded bg-slate-800 hover:bg-indigo-600 text-slate-300 hover:text-white transition-colors text-[11px] font-sans flex items-center gap-1 mx-auto"
                          title="Inspect complete cryptographic decision evidence"
                        >
                          <Eye className="w-3 h-3" />
                          Inspect
                        </button>
                      </td>
                    </tr>
                  );
                })}
              </tbody>
            </table>
          )}
        </div>

        {/* Right Slide-Over Evidence Inspector Drawer */}
        {selectedLineageId && (
          <div className="w-[480px] bg-[#071324] border-l border-slate-800 flex flex-col h-full shadow-2xl z-20 overflow-hidden shrink-0">
            {/* Drawer Header */}
            <div className="bg-[#0a182f] px-4 py-3 border-b border-slate-800 flex items-center justify-between shrink-0">
              <div className="flex items-center gap-2">
                <FileText className="w-4 h-4 text-indigo-400" />
                <h3 className="text-sm font-bold text-white">Decision Evidence Inspector</h3>
              </div>
              <button
                onClick={() => setSelectedLineageId(null)}
                className="text-slate-400 hover:text-white p-1 rounded hover:bg-slate-800"
              >
                <X className="w-4 h-4" />
              </button>
            </div>

            {/* Drawer Body */}
            <div className="flex-1 overflow-y-auto p-4 space-y-4">
              {isEvidenceLoading ? (
                <div className="flex flex-col items-center justify-center h-48 text-slate-400 gap-3">
                  <RefreshCw className="w-6 h-6 animate-spin text-indigo-400" />
                  <p className="text-xs">Fetching cryptographic evidence bundle...</p>
                </div>
              ) : evidenceError ? (
                <div className="p-3 bg-rose-950/40 border border-rose-800 rounded text-rose-300 text-xs">
                  {evidenceError}
                </div>
              ) : evidenceBundle ? (
                <>
                  {/* Status Banner */}
                  <div className={`p-3 rounded-lg border flex items-center justify-between ${
                    evidenceBundle.evaluation.action_taken === 'BLOCKED'
                      ? 'bg-rose-950/30 border-rose-800/80 text-rose-300'
                      : evidenceBundle.evaluation.action_taken === 'WARNED'
                      ? 'bg-amber-950/30 border-amber-800/80 text-amber-300'
                      : 'bg-emerald-950/30 border-emerald-800/80 text-emerald-300'
                  }`}>
                    <div className="flex items-center gap-2">
                      {getStatusBadge(evidenceBundle.evaluation.action_taken)}
                      <span className="text-xs font-semibold">
                        {evidenceBundle.evaluation.rule_name}
                      </span>
                    </div>
                    <span className="text-xs font-mono text-slate-400">
                      v{evidenceBundle.evaluation.rule_version}
                    </span>
                  </div>

                  {/* Natural-Language Explanation Card */}
                  <div className="bg-[#0b1b36] border border-indigo-900/60 rounded-lg p-3.5 shadow-sm space-y-2">
                    <div className="flex items-center gap-2 text-indigo-300 font-semibold text-xs">
                      <Sparkles className="w-4 h-4 text-indigo-400" />
                      <h4>Generated Decision Explanation</h4>
                    </div>
                    <p className="text-xs text-slate-300 leading-relaxed font-sans">
                      {evidenceBundle.natural_language_explanation}
                    </p>
                  </div>

                  {/* Evaluated Metrics vs Constraints */}
                  <div className="bg-[#09152a] border border-slate-800 rounded-lg p-3 space-y-2">
                    <div className="flex items-center gap-2 text-slate-300 font-semibold text-xs border-b border-slate-800 pb-2">
                      <Sliders className="w-3.5 h-3.5 text-sky-400" />
                      <h4>Evaluated Metrics & Limits</h4>
                    </div>
                    {evidenceBundle.metrics && evidenceBundle.metrics.length > 0 ? (
                      <div className="space-y-2 pt-1">
                        {evidenceBundle.metrics.map((m, idx) => (
                          <div
                            key={idx}
                            className={`p-2.5 rounded border text-xs font-mono flex flex-col gap-1 ${
                              m.breached
                                ? 'bg-rose-950/40 border-rose-800 text-rose-200'
                                : 'bg-slate-900/80 border-slate-800 text-slate-300'
                            }`}
                          >
                            <div className="flex items-center justify-between font-semibold">
                              <span className="text-sky-300">{m.metric_path}</span>
                              <span className={m.breached ? 'text-rose-400' : 'text-emerald-400'}>
                                {m.breached ? 'BREACHED' : 'SATISFIED'}
                              </span>
                            </div>
                            <div className="flex items-center justify-between text-[11px] text-slate-400">
                              <span>Observed: <strong className="text-white">{String(m.observed_value)}</strong></span>
                              {m.threshold_value !== undefined && (
                                <span>Threshold: <strong className="text-white">{String(m.threshold_value)}</strong></span>
                              )}
                            </div>
                            {m.margin && (
                              <div className="text-[10px] text-slate-400 pt-0.5 border-t border-slate-800/80">
                                Margin: {m.margin}
                              </div>
                            )}
                          </div>
                        ))}
                      </div>
                    ) : (
                      <p className="text-xs text-slate-500 py-1">No numerical scalar metrics recorded.</p>
                    )}
                  </div>

                  {/* Regulatory Authority & Legal Citation */}
                  {evidenceBundle.rule_snapshot.citation && (
                    <div className="bg-[#09152a] border border-slate-800 rounded-lg p-3 space-y-1.5">
                      <h4 className="text-xs font-semibold text-slate-400">Legal Citation & Regulatory Authority</h4>
                      <p className="text-xs text-slate-200 font-sans italic bg-slate-900/60 p-2 rounded border border-slate-800">
                        "{evidenceBundle.rule_snapshot.citation}"
                      </p>
                    </div>
                  )}

                  {/* Cryptographic Integrity & Lineage Proof */}
                  <div className="bg-[#09152a] border border-slate-800 rounded-lg p-3 space-y-2 text-xs">
                    <div className="flex items-center justify-between border-b border-slate-800 pb-2">
                      <div className="flex items-center gap-1.5 text-slate-300 font-semibold">
                        <Hash className="w-3.5 h-3.5 text-emerald-400" />
                        <h4>Cryptographic Provenance</h4>
                      </div>
                      <span className="text-[10px] px-2 py-0.5 rounded bg-emerald-950 text-emerald-300 border border-emerald-800 font-semibold">
                        {evidenceBundle.integrity_proof.content_hash_matches ? 'Verified 100%' : 'Hash Tampered'}
                      </span>
                    </div>

                    <div className="space-y-1.5 font-mono text-[11px]">
                      <div>
                        <span className="text-slate-500 block">Lineage UUID:</span>
                        <div className="flex items-center justify-between text-slate-300 bg-slate-900 px-2 py-1 rounded border border-slate-800">
                          <span className="truncate">{evidenceBundle.evaluation.lineage_id}</span>
                          <button
                            onClick={() => copyToClipboard(evidenceBundle.evaluation.lineage_id)}
                            className="text-slate-400 hover:text-white ml-1"
                            title="Copy Lineage ID"
                          >
                            <Copy className="w-3 h-3" />
                          </button>
                        </div>
                      </div>

                      <div>
                        <span className="text-slate-500 block">Rule Content Hash (RFC 8785):</span>
                        <div className="flex items-center justify-between text-slate-300 bg-slate-900 px-2 py-1 rounded border border-slate-800">
                          <span className="truncate">{evidenceBundle.rule_snapshot.content_hash}</span>
                          <button
                            onClick={() => copyToClipboard(evidenceBundle.rule_snapshot.content_hash)}
                            className="text-slate-400 hover:text-white ml-1"
                            title="Copy Content Hash"
                          >
                            <Copy className="w-3 h-3" />
                          </button>
                        </div>
                      </div>

                      <div>
                        <span className="text-slate-500 block">Decision Evaluation Hash:</span>
                        <div className="flex items-center justify-between text-slate-300 bg-slate-900 px-2 py-1 rounded border border-slate-800">
                          <span className="truncate">{evidenceBundle.evaluation.evaluation_hash}</span>
                          <button
                            onClick={() => copyToClipboard(evidenceBundle.evaluation.evaluation_hash)}
                            className="text-slate-400 hover:text-white ml-1"
                            title="Copy Evaluation Hash"
                          >
                            <Copy className="w-3 h-3" />
                          </button>
                        </div>
                      </div>
                    </div>
                  </div>

                  {/* FDC3 Cross-Display Broadcast Button */}
                  <div className="pt-2">
                    <button
                      onClick={() => {
                        broadcast({
                          type: 'fdc3.order',
                          id: {
                            orderId: evidenceBundle.evaluation.order_id || evidenceBundle.evaluation.id,
                            lineageId: evidenceBundle.evaluation.lineage_id,
                            ruleCode: evidenceBundle.evaluation.rule_code,
                          },
                          name: `${evidenceBundle.evaluation.rule_code} Pre-Trade Decision`,
                        });
                      }}
                      className="w-full py-2 px-3 rounded bg-indigo-600 hover:bg-indigo-500 text-white font-semibold text-xs transition-colors flex items-center justify-center gap-2 shadow cursor-pointer"
                    >
                      <Radio className="w-4 h-4" />
                      Broadcast Order to Linked Displays (FDC3)
                    </button>
                  </div>
                </>
              ) : null}
            </div>
          </div>
        )}
      </div>
    </div>
  );
};

export default ComplianceDecisionBlotter;
