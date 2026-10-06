import React, { useCallback, useEffect, useState } from 'react';
import {
  FileText,
  AlertTriangle,
  CheckCircle2,
  XCircle,
  Clock,
  Search,
  RefreshCw,
  Eye,
  Sliders,
  Sparkles,
  GitCommit,
  Check,
  Send,
  X,
  Layers,
  ArrowRight,
} from 'lucide-react';

export interface RegulatoryCase {
  id: string;
  case_code: string;
  source: string;
  source_reference?: string;
  title: string;
  description: string;
  affected_rule_ids: string[];
  classification?: string;
  triage_notes?: string;
  triaged_by?: string;
  triaged_at?: string;
  status: 'INTAKED' | 'TRIAGED' | 'UNDER_REVIEW' | 'APPROVED_FOR_PUBLISH' | 'PUBLISHED' | 'CLOSED_NO_IMPACT' | 'REJECTED' | 'ESCALATED' | string;
  is_escalated: boolean;
  escalation_count: number;
  due_at: string;
  created_by: string;
  created_at: string;
  updated_at: string;
}

export interface StewardReviewView {
  case_id: string;
  case_code: string;
  title: string;
  classification: string;
  rules_review: Array<{
    rule_id: string;
    rule_code: string;
    rule_name: string;
    current_version: number;
    proposed_version: number;
    current_ast: any;
    proposed_ast: any;
    ast_has_changed: boolean;
    current_thresholds: any;
    proposed_thresholds: any;
    thresholds_have_changed: boolean;
    threshold_diff_summary: string;
    current_citation: string;
    proposed_citation: string;
    proposed_content_hash: string;
    corpus_passed: boolean;
    corpus_run_count: number;
    is_approved: boolean;
    approved_by?: string;
    approved_at?: string;
  }>;
  overall_status: string;
  can_publish: boolean;
}

export const RegulatoryChangeQueue: React.FC = () => {
  const [cases, setCases] = useState<RegulatoryCase[]>([]);
  const [loading, setLoading] = useState(false);
  const [statusFilter, setStatusFilter] = useState<string>('ALL');
  const [searchQuery, setSearchQuery] = useState('');
  const [selectedCase, setSelectedCase] = useState<RegulatoryCase | null>(null);
  const [reviewView, setReviewView] = useState<StewardReviewView | null>(null);
  const [reviewLoading, setReviewLoading] = useState(false);
  const [actionError, setActionError] = useState<string | null>(null);
  const [approverNotes, setApproverNotes] = useState('');

  // Triage state
  const [triageModalOpen, setTriageModalOpen] = useState(false);
  const [triageClassification, setTriageClassification] = useState('PARAMETER_CHANGE');
  const [triageNotes, setTriageNotes] = useState('');

  const fetchCases = useCallback(async () => {
    setLoading(true);
    try {
      const tenantId = localStorage.getItem('tenant_id') || '99e99e99-99e9-49e9-89e9-99e99e99e999';
      const token = localStorage.getItem('AUTH_TOKEN') || localStorage.getItem('auth_token') || '';
      const url = statusFilter === 'UNADDRESSED'
        ? '/api/compliance/regulatory/cases/unaddressed'
        : `/api/compliance/regulatory/cases${statusFilter !== 'ALL' ? `?status=${statusFilter}` : ''}`;

      const res = await fetch(url, {
        headers: {
          'Authorization': token ? `Bearer ${token}` : '',
          'X-Tenant-ID': tenantId,
        },
      });
      if (res.ok) {
        const json = await res.json();
        setCases(json.data || []);
      }
    } catch (err) {
      console.error('Failed to fetch regulatory cases', err);
    } finally {
      setLoading(false);
    }
  }, [statusFilter]);

  useEffect(() => {
    fetchCases();
  }, [fetchCases]);

  const loadStewardReview = async (c: RegulatoryCase) => {
    setSelectedCase(c);
    setReviewLoading(true);
    setActionError(null);
    try {
      const token = localStorage.getItem('AUTH_TOKEN') || localStorage.getItem('auth_token') || '';
      const res = await fetch(`/api/compliance/regulatory/cases/${c.id}/steward-review`, {
        headers: {
          'Authorization': token ? `Bearer ${token}` : '',
          'X-Tenant-ID': c.id,
        },
      });
      if (res.ok) {
        const data = await res.json();
        setReviewView(data);
      } else {
        setReviewView(null);
      }
    } catch (err) {
      console.error('Failed to load steward review view', err);
    } finally {
      setReviewLoading(false);
    }
  };

  const handleTriageSubmit = async () => {
    if (!selectedCase) return;
    try {
      const token = localStorage.getItem('AUTH_TOKEN') || localStorage.getItem('auth_token') || '';
      const res = await fetch(`/api/compliance/regulatory/cases/${selectedCase.id}/triage`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'Authorization': token ? `Bearer ${token}` : '',
        },
        body: JSON.stringify({
          classification: triageClassification,
          triage_notes: triageNotes,
          affected_rule_ids: selectedCase.affected_rule_ids,
        }),
      });
      if (res.ok) {
        setTriageModalOpen(false);
        fetchCases();
        if (selectedCase) {
          loadStewardReview(selectedCase);
        }
      } else {
        const errTxt = await res.text();
        setActionError(`Triage failed: ${errTxt}`);
      }
    } catch (err: any) {
      setActionError(`Triage error: ${err.message}`);
    }
  };

  const handleApprove = async () => {
    if (!selectedCase) return;
    try {
      const token = localStorage.getItem('AUTH_TOKEN') || localStorage.getItem('auth_token') || '';
      const res = await fetch(`/api/compliance/regulatory/cases/${selectedCase.id}/approve`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'Authorization': token ? `Bearer ${token}` : '',
        },
        body: JSON.stringify({
          approver_notes: approverNotes,
        }),
      });
      if (res.ok) {
        fetchCases();
        loadStewardReview(selectedCase);
      } else {
        const errTxt = await res.text();
        setActionError(`Approval failed: ${errTxt}`);
      }
    } catch (err: any) {
      setActionError(`Approval error: ${err.message}`);
    }
  };

  const handlePublish = async () => {
    if (!selectedCase || !reviewView) return;
    try {
      const token = localStorage.getItem('AUTH_TOKEN') || localStorage.getItem('auth_token') || '';
      const drafts = reviewView.rules_review.map(r => ({
        rule_id: r.rule_id,
        new_ast: r.proposed_ast,
        new_thresholds: r.proposed_thresholds,
        new_citation: r.proposed_citation,
        effective_from: new Date().toISOString(),
      }));

      const res = await fetch(`/api/compliance/regulatory/cases/${selectedCase.id}/publish`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'Authorization': token ? `Bearer ${token}` : '',
        },
        body: JSON.stringify({
          drafts,
        }),
      });
      if (res.ok) {
        fetchCases();
        loadStewardReview(selectedCase);
      } else {
        const errTxt = await res.text();
        setActionError(`Publish failed: ${errTxt}`);
      }
    } catch (err: any) {
      setActionError(`Publish error: ${err.message}`);
    }
  };

  const filteredCases = cases.filter(c => {
    if (!searchQuery) return true;
    const q = searchQuery.toLowerCase();
    return (
      c.case_code.toLowerCase().includes(q) ||
      c.title.toLowerCase().includes(q) ||
      c.description.toLowerCase().includes(q) ||
      (c.source_reference && c.source_reference.toLowerCase().includes(q))
    );
  });

  const getStatusBadge = (status: string) => {
    switch (status) {
      case 'APPROVED_FOR_PUBLISH':
      case 'PUBLISHED':
        return (
          <span className="inline-flex items-center gap-1.5 px-2.5 py-1 rounded-full text-xs font-semibold bg-emerald-500/10 text-emerald-400 border border-emerald-500/30">
            <CheckCircle2 className="w-3.5 h-3.5" />
            {status}
          </span>
        );
      case 'UNDER_REVIEW':
      case 'TRIAGED':
        return (
          <span className="inline-flex items-center gap-1.5 px-2.5 py-1 rounded-full text-xs font-semibold bg-sky-500/10 text-sky-400 border border-sky-500/30">
            <Sliders className="w-3.5 h-3.5" />
            {status}
          </span>
        );
      case 'ESCALATED':
        return (
          <span className="inline-flex items-center gap-1.5 px-2.5 py-1 rounded-full text-xs font-semibold bg-rose-500/10 text-rose-400 border border-rose-500/30 animate-pulse">
            <AlertTriangle className="w-3.5 h-3.5" />
            ESCALATED
          </span>
        );
      case 'INTAKED':
      default:
        return (
          <span className="inline-flex items-center gap-1.5 px-2.5 py-1 rounded-full text-xs font-semibold bg-amber-500/10 text-amber-400 border border-amber-500/30">
            <Clock className="w-3.5 h-3.5" />
            {status}
          </span>
        );
    }
  };

  return (
    <div className="flex h-full w-full bg-slate-950 text-slate-100 overflow-hidden font-sans">
      {/* Main Cases Operational Queue */}
      <div className="flex-1 flex flex-col min-w-0 border-r border-slate-800/80">
        {/* Header & Controls */}
        <div className="p-4 border-b border-slate-800 bg-slate-900/60 flex flex-col gap-3">
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-3">
              <div className="p-2 bg-indigo-500/10 border border-indigo-500/30 rounded-lg text-indigo-400">
                <FileText className="w-5 h-5" />
              </div>
              <div>
                <h1 className="text-base font-bold text-white tracking-wide flex items-center gap-2">
                  Regulatory Change Management Queue
                  <span className="text-xs px-2 py-0.5 rounded-full bg-indigo-900/60 border border-indigo-700/50 text-indigo-300">
                    SLA v_unaddressed
                  </span>
                </h1>
                <p className="text-xs text-slate-400">
                  Steward intake, AST impact analysis, threshold diff review, and release publishing
                </p>
              </div>
            </div>

            <button
              onClick={fetchCases}
              disabled={loading}
              className="px-3 py-1.5 rounded-lg bg-slate-800 hover:bg-slate-700 text-xs text-slate-200 border border-slate-700 flex items-center gap-1.5 transition"
            >
              <RefreshCw className={`w-3.5 h-3.5 ${loading ? 'animate-spin text-indigo-400' : ''}`} />
              Refresh
            </button>
          </div>

          {/* Filter Bar */}
          <div className="flex items-center justify-between gap-3">
            <div className="relative flex-1 max-w-md">
              <Search className="w-4 h-4 text-slate-400 absolute left-3 top-1/2 -translate-y-1/2" />
              <input
                type="text"
                placeholder="Search case code, title, regulator citation..."
                value={searchQuery}
                onChange={e => setSearchQuery(e.target.value)}
                className="w-full bg-slate-900 border border-slate-700/80 rounded-lg pl-9 pr-3 py-1.5 text-xs text-slate-200 placeholder-slate-500 focus:outline-none focus:border-indigo-500"
              />
            </div>

            <div className="flex items-center gap-1.5 bg-slate-900/80 p-1 rounded-lg border border-slate-800 text-xs">
              {['ALL', 'UNADDRESSED', 'INTAKED', 'TRIAGED', 'APPROVED_FOR_PUBLISH', 'PUBLISHED'].map(tab => (
                <button
                  key={tab}
                  onClick={() => setStatusFilter(tab)}
                  className={`px-2.5 py-1 rounded-md transition font-medium ${
                    statusFilter === tab
                      ? 'bg-indigo-600 text-white shadow-sm'
                      : 'text-slate-400 hover:text-slate-200'
                  }`}
                >
                  {tab.replace('_', ' ')}
                </button>
              ))}
            </div>
          </div>
        </div>

        {/* Case Table */}
        <div className="flex-1 overflow-auto">
          <table className="w-full text-left text-xs border-collapse">
            <thead className="sticky top-0 bg-slate-900/95 backdrop-blur z-10 border-b border-slate-800 text-slate-400 font-semibold uppercase tracking-wider">
              <tr>
                <th className="py-2.5 px-4">Case Code</th>
                <th className="py-2.5 px-4">Source</th>
                <th className="py-2.5 px-4">Title & Description</th>
                <th className="py-2.5 px-4">Classification</th>
                <th className="py-2.5 px-4">Status</th>
                <th className="py-2.5 px-4">Due (SLA)</th>
                <th className="py-2.5 px-4 text-right">Actions</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-slate-800/60">
              {filteredCases.map(c => {
                const isSelected = selectedCase?.id === c.id;
                const dueDate = new Date(c.due_at);
                const isOverdue = dueDate.getTime() < Date.now();

                return (
                  <tr
                    key={c.id}
                    onClick={() => loadStewardReview(c)}
                    className={`cursor-pointer transition hover:bg-slate-900/80 ${
                      isSelected ? 'bg-indigo-950/30 border-l-2 border-indigo-500' : ''
                    }`}
                  >
                    <td className="py-3 px-4 font-mono font-semibold text-slate-200">
                      {c.case_code}
                    </td>
                    <td className="py-3 px-4">
                      <span className="px-2 py-0.5 rounded bg-slate-800 text-slate-300 font-mono text-[11px]">
                        {c.source}
                      </span>
                      {c.source_reference && (
                        <div className="text-[10px] text-slate-500 mt-0.5 truncate max-w-[120px]">
                          {c.source_reference}
                        </div>
                      )}
                    </td>
                    <td className="py-3 px-4 max-w-xs">
                      <div className="font-semibold text-slate-200 truncate">{c.title}</div>
                      <div className="text-slate-400 text-[11px] truncate">{c.description}</div>
                    </td>
                    <td className="py-3 px-4">
                      {c.classification ? (
                        <span className="px-2 py-0.5 rounded text-[11px] bg-slate-800 text-indigo-300 border border-indigo-900/50">
                          {c.classification}
                        </span>
                      ) : (
                        <span className="text-slate-500 italic text-[11px]">Untriaged</span>
                      )}
                    </td>
                    <td className="py-3 px-4">{getStatusBadge(c.status)}</td>
                    <td className="py-3 px-4 whitespace-nowrap">
                      <div className={`flex items-center gap-1 font-mono text-[11px] ${
                        isOverdue ? 'text-rose-400 font-bold' : 'text-slate-400'
                      }`}>
                        <Clock className="w-3 h-3" />
                        {dueDate.toLocaleDateString()}
                      </div>
                    </td>
                    <td className="py-3 px-4 text-right">
                      <button
                        onClick={(e) => {
                          e.stopPropagation();
                          loadStewardReview(c);
                        }}
                        className="p-1.5 rounded hover:bg-slate-800 text-slate-300 hover:text-indigo-400 transition"
                      >
                        <Eye className="w-4 h-4" />
                      </button>
                    </td>
                  </tr>
                );
              })}

              {filteredCases.length === 0 && !loading && (
                <tr>
                  <td colSpan={7} className="py-12 text-center text-slate-500">
                    No regulatory cases matching current filters.
                  </td>
                </tr>
              )}
            </tbody>
          </table>
        </div>
      </div>

      {/* Steward Review & Diff Inspector Panel */}
      {selectedCase && (
        <div className="w-[540px] flex flex-col bg-slate-900/90 border-l border-slate-800 min-w-0">
          <div className="p-4 border-b border-slate-800 flex items-center justify-between bg-slate-900">
            <div className="flex items-center gap-2">
              <Sparkles className="w-4 h-4 text-indigo-400" />
              <h2 className="text-sm font-bold text-white font-mono">
                Steward Review: {selectedCase.case_code}
              </h2>
            </div>
            <button
              onClick={() => setSelectedCase(null)}
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

            {/* Case Summary */}
            <div className="p-3 bg-slate-950/70 border border-slate-800 rounded-lg space-y-2">
              <div className="text-xs font-semibold text-slate-300">{selectedCase.title}</div>
              <p className="text-xs text-slate-400 leading-relaxed">{selectedCase.description}</p>
              <div className="flex items-center justify-between pt-2 border-t border-slate-800 text-[11px] text-slate-400">
                <span>Status: <strong className="text-slate-200">{selectedCase.status}</strong></span>
                <span>Source: <strong className="text-slate-200">{selectedCase.source}</strong></span>
              </div>
            </div>

            {/* Actions Toolbar */}
            <div className="flex items-center gap-2">
              {selectedCase.status === 'INTAKED' && (
                <button
                  onClick={() => setTriageModalOpen(true)}
                  className="flex-1 py-2 px-3 rounded-lg bg-indigo-600 hover:bg-indigo-500 text-white font-semibold text-xs flex items-center justify-center gap-1.5 shadow"
                >
                  <Sliders className="w-3.5 h-3.5" />
                  Triage Case
                </button>
              )}

              {reviewView?.rules_review && reviewView.rules_review.some(r => !r.is_approved) && (
                <button
                  onClick={handleApprove}
                  className="flex-1 py-2 px-3 rounded-lg bg-emerald-600 hover:bg-emerald-500 text-white font-semibold text-xs flex items-center justify-center gap-1.5 shadow"
                >
                  <Check className="w-3.5 h-3.5" />
                  Approve Draft Rules
                </button>
              )}

              {reviewView?.can_publish && (
                <button
                  onClick={handlePublish}
                  className="flex-1 py-2 px-3 rounded-lg bg-blue-600 hover:bg-blue-500 text-white font-semibold text-xs flex items-center justify-center gap-1.5 shadow"
                >
                  <Send className="w-3.5 h-3.5" />
                  Publish Release
                </button>
              )}
            </div>

            {/* Rule Diffs & Thresholds Breakdown */}
            {reviewLoading ? (
              <div className="py-12 text-center text-slate-500 text-xs">
                Loading AST and threshold diffs...
              </div>
            ) : reviewView?.rules_review && reviewView.rules_review.length > 0 ? (
              <div className="space-y-4">
                <h3 className="text-xs font-bold uppercase tracking-wider text-slate-400 flex items-center gap-1.5">
                  <GitCommit className="w-3.5 h-3.5 text-indigo-400" />
                  Rule Diffs ({reviewView.rules_review.length})
                </h3>

                {reviewView.rules_review.map(r => (
                  <div key={r.rule_id} className="p-3 bg-slate-950/80 border border-slate-800 rounded-lg space-y-3">
                    <div className="flex items-center justify-between">
                      <div className="font-mono font-bold text-xs text-indigo-300">
                        {r.rule_code} (v{r.current_version} → v{r.proposed_version})
                      </div>
                      <span className={`text-[10px] px-2 py-0.5 rounded font-semibold ${
                        r.is_approved ? 'bg-emerald-500/20 text-emerald-300' : 'bg-amber-500/20 text-amber-300'
                      }`}>
                        {r.is_approved ? 'APPROVED' : 'PENDING APPROVAL'}
                      </span>
                    </div>

                    {/* Threshold Diff Summary */}
                    {r.threshold_diff_summary && (
                      <div className="p-2.5 bg-indigo-950/40 border border-indigo-800/40 rounded text-[11px] text-indigo-200">
                        <div className="font-semibold text-indigo-400 mb-1">Parameter Changes:</div>
                        {r.threshold_diff_summary}
                      </div>
                    )}

                    {/* Citations */}
                    <div className="text-[11px] text-slate-400 space-y-1">
                      <div className="font-semibold text-slate-300">Citation:</div>
                      <div className="p-2 bg-slate-900 rounded font-serif italic text-slate-300">
                        "{r.proposed_citation || r.current_citation}"
                      </div>
                    </div>

                    {/* Hash Fingerprint */}
                    <div className="p-2 bg-slate-900 rounded font-mono text-[10px] text-slate-400 flex items-center justify-between">
                      <span>Proposed Content Hash:</span>
                      <span className="text-slate-200 truncate max-w-[200px]">{r.proposed_content_hash}</span>
                    </div>
                  </div>
                ))}
              </div>
            ) : (
              <div className="p-6 text-center text-slate-500 text-xs border border-dashed border-slate-800 rounded-lg">
                No draft rules attached to this case. Click "Triage Case" to attach rules.
              </div>
            )}
          </div>
        </div>
      )}

      {/* Triage Modal */}
      {triageModalOpen && (
        <div className="fixed inset-0 bg-black/70 backdrop-blur-sm z-50 flex items-center justify-center p-4">
          <div className="bg-slate-900 border border-slate-800 rounded-xl max-w-md w-full p-6 space-y-4 shadow-2xl">
            <h3 className="text-base font-bold text-white">Triage Regulatory Case</h3>
            <p className="text-xs text-slate-400">
              Assign classification and triage notes for case {selectedCase?.case_code}
            </p>

            <div className="space-y-3 text-xs">
              <div>
                <label className="block text-slate-300 font-semibold mb-1">Classification</label>
                <select
                  value={triageClassification}
                  onChange={e => setTriageClassification(e.target.value)}
                  className="w-full bg-slate-950 border border-slate-700 rounded-lg p-2 text-slate-200 focus:outline-none focus:border-indigo-500"
                >
                  <option value="PARAMETER_CHANGE">PARAMETER_CHANGE (Thresholds / Limits only)</option>
                  <option value="SEMANTIC_CHANGE">SEMANTIC_CHANGE (AST Structural alteration)</option>
                  <option value="EDITORIAL">EDITORIAL (Citation / Clarification)</option>
                  <option value="NEW_RULE_REQUIRED">NEW_RULE_REQUIRED (Create brand new rule)</option>
                  <option value="NO_IMPACT">NO_IMPACT (Close with no change)</option>
                </select>
              </div>

              <div>
                <label className="block text-slate-300 font-semibold mb-1">Triage Notes</label>
                <textarea
                  rows={3}
                  value={triageNotes}
                  onChange={e => setTriageNotes(e.target.value)}
                  placeholder="Explain rationale for classification and regulatory impact..."
                  className="w-full bg-slate-950 border border-slate-700 rounded-lg p-2 text-slate-200 focus:outline-none focus:border-indigo-500"
                />
              </div>
            </div>

            <div className="flex items-center justify-end gap-2 pt-2">
              <button
                onClick={() => setTriageModalOpen(false)}
                className="px-4 py-2 rounded-lg bg-slate-800 hover:bg-slate-700 text-xs text-slate-300"
              >
                Cancel
              </button>
              <button
                onClick={handleTriageSubmit}
                className="px-4 py-2 rounded-lg bg-indigo-600 hover:bg-indigo-500 text-xs text-white font-semibold"
              >
                Submit Triage
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
};
