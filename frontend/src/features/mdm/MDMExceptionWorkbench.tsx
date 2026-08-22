import React, { useState, useEffect } from 'react';
import {
  AlertTriangle,
  CheckCircle2,
  ShieldAlert,
  Bot,
  ArrowRight,
  TrendingUp,
  Clock,
  Sparkles,
  RefreshCw,
  Sliders,
  Send,
  Building2,
  Layers,
  Database,
  ExternalLink,
} from 'lucide-react';

export interface CompetingVendorValue {
  vendor: string;
  value: any;
  confidence: number;
  timestamp?: string;
  isStale?: boolean;
}

export interface MDMExceptionItem {
  exceptionId: string;
  tenantId: string;
  domainKey: string; // SECURITY, PRICING, CORP_ACTION, ISSUER, FUND, ACCOUNT
  masterEntitySid: string;
  entityName?: string;
  fieldName: string;
  anomalyType: string; // PRICE_TOLERANCE_BREACH, CHECKSUM_FAILURE, UNRESOLVED_XREF, STALE_FEED
  status: 'OPEN' | 'IN_REVIEW' | 'RESOLVED' | 'OVERRIDDEN';
  competingValues: CompetingVendorValue[];
  maxDeviationPct: number;
  createdAt: string;
  aiDiagnosis?: {
    recommendation: string;
    winningVendor: string;
    suggestedValue: any;
    confidenceScore: number;
    rationale: string;
  };
}

export const MDMExceptionWorkbench: React.FC<{ tenantId: string }> = ({ tenantId }) => {
  const [exceptions, setExceptions] = useState<MDMExceptionItem[]>([]);
  const [selectedException, setSelectedException] = useState<MDMExceptionItem | null>(null);
  const [selectedDomain, setSelectedDomain] = useState<string>('ALL');
  const [selectedAnomaly, setSelectedAnomaly] = useState<string>('ALL');
  const [overrideReason, setOverrideReason] = useState<string>('');
  const [customValue, setCustomValue] = useState<string>('');
  const [isSubmitting, setIsSubmitting] = useState<boolean>(false);
  const [notification, setNotification] = useState<string | null>(null);

  useEffect(() => {
    fetchExceptions();
  }, [tenantId, selectedDomain, selectedAnomaly]);

  const fetchExceptions = async () => {
    try {
      const queryParams = new URLSearchParams({
        domain: selectedDomain,
        anomaly: selectedAnomaly,
      });
      const res = await fetch(`/api/v1/mdm/exceptions?${queryParams.toString()}`, {
        headers: { 'X-Tenant-ID': tenantId },
      });
      if (res.ok) {
        const data = await res.json();
        setExceptions(data);
        if (data.length > 0 && !selectedException) {
          setSelectedException(data[0]);
        }
      }
    } catch (err) {
      console.error('Failed fetching MDM exceptions:', err);
    }
  };

  const handleApplyOverride = async (vendor: string, overrideVal: any, reason: string) => {
    if (!selectedException) return;
    setIsSubmitting(true);

    try {
      const payload = {
        exceptionId: selectedException.exceptionId,
        masterEntitySid: selectedException.masterEntitySid,
        domainKey: selectedException.domainKey,
        fieldName: selectedException.fieldName,
        chosenVendor: vendor,
        overrideValue: overrideVal,
        overrideReason: reason || 'Manual Data Steward Override',
        signalTemporalWorkflow: true,
      };

      const res = await fetch(`/api/v1/mdm/exceptions/${selectedException.exceptionId}/override`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'X-Tenant-ID': tenantId,
        },
        body: JSON.stringify(payload),
      });

      if (res.ok) {
        setNotification(`Override applied for ${selectedException.masterEntitySid}. Temporal workflow signaled.`);
        setTimeout(() => setNotification(null), 4000);
        await fetchExceptions();
      }
    } finally {
      setIsSubmitting(false);
      setOverrideReason('');
      setCustomValue('');
    }
  };

  const filteredExceptions = exceptions.filter((item) => {
    const matchDomain = selectedDomain === 'ALL' || item.domainKey === selectedDomain;
    const matchAnomaly = selectedAnomaly === 'ALL' || item.anomalyType === selectedAnomaly;
    return matchDomain && matchAnomaly;
  });

  return (
    <div className="flex flex-col h-full bg-[#030914] text-slate-100 border border-slate-800 rounded-xl overflow-hidden font-sans">
      {/* Top Banner / Triage HUD */}
      <div className="flex items-center justify-between px-6 py-4 bg-[#071526] border-b border-slate-800">
        <div>
          <h2 className="text-base font-bold text-slate-100 tracking-wide flex items-center gap-2">
            <ShieldAlert className="w-5 h-5 text-amber-400" />
            MDM Data Stewardship & Exception Workbench
          </h2>
          <p className="text-xs text-slate-400 mt-0.5">
            Resolve price tolerance breaches, vendor feed collisions, and checksum anomalies across 8 master domains.
          </p>
        </div>

        {/* Global Filters */}
        <div className="flex items-center gap-3">
          <div className="flex items-center gap-2 bg-slate-900 border border-slate-800 rounded-lg px-2.5 py-1.5 text-xs">
            <Layers className="w-3.5 h-3.5 text-cyan-400" />
            <select
              value={selectedDomain}
              onChange={(e) => setSelectedDomain(e.target.value)}
              className="bg-transparent text-slate-200 outline-none cursor-pointer"
            >
              <option value="ALL">All Master Domains</option>
              <option value="PRICING">Pricing & Curves</option>
              <option value="SECURITY">Security Master</option>
              <option value="CORP_ACTION">Corporate Actions</option>
              <option value="ISSUER">Legal Entity & Issuer</option>
              <option value="FUND">Fund & Vehicle Master</option>
            </select>
          </div>

          <div className="flex items-center gap-2 bg-slate-900 border border-slate-800 rounded-lg px-2.5 py-1.5 text-xs">
            <AlertTriangle className="w-3.5 h-3.5 text-amber-400" />
            <select
              value={selectedAnomaly}
              onChange={(e) => setSelectedAnomaly(e.target.value)}
              className="bg-transparent text-slate-200 outline-none cursor-pointer"
            >
              <option value="ALL">All Anomaly Types</option>
              <option value="PRICE_TOLERANCE_BREACH">Price Breach (&gt;10%)</option>
              <option value="CHECKSUM_FAILURE">Checksum Failure</option>
              <option value="UNRESOLVED_XREF">Unresolved XREF</option>
              <option value="STALE_FEED">Stale Feed</option>
            </select>
          </div>

          <button
            onClick={fetchExceptions}
            className="p-2 bg-slate-900 hover:bg-slate-800 border border-slate-800 rounded-lg text-slate-300 transition"
            title="Refresh Exceptions"
          >
            <RefreshCw className="w-3.5 h-3.5" />
          </button>
        </div>
      </div>

      {notification && (
        <div className="bg-emerald-500/20 border-b border-emerald-500/40 px-6 py-2 text-xs text-emerald-300 flex items-center gap-2">
          <CheckCircle2 className="w-4 h-4 text-emerald-400" />
          {notification}
        </div>
      )}

      {/* Main Split Layout */}
      <div className="grid grid-cols-12 flex-1 overflow-hidden">
        {/* Left Column: Exceptions Queue List (4 Columns) */}
        <div className="col-span-4 border-r border-slate-800 overflow-y-auto bg-[#050e1d]/60">
          <div className="p-3 bg-slate-900/40 border-b border-slate-800/80 text-[11px] font-semibold text-slate-400 uppercase tracking-wider flex justify-between">
            <span>Open Break Queue</span>
            <span className="text-amber-400 font-mono">{filteredExceptions.length} Items</span>
          </div>

          <div className="divide-y divide-slate-800/60">
            {filteredExceptions.map((item) => {
              const isSelected = selectedException?.exceptionId === item.exceptionId;
              return (
                <div
                  key={item.exceptionId}
                  onClick={() => setSelectedException(item)}
                  className={`p-4 cursor-pointer transition-all ${
                    isSelected
                      ? 'bg-cyan-500/10 border-l-4 border-cyan-400'
                      : 'hover:bg-slate-900/40'
                  }`}
                >
                  <div className="flex items-start justify-between">
                    <div>
                      <span className="text-xs font-mono font-bold text-slate-200 block">
                        {item.masterEntitySid}
                      </span>
                      <span className="text-[11px] text-slate-400 mt-0.5 block">
                        {item.entityName || item.fieldName}
                      </span>
                    </div>
                    <span
                      className={`px-2 py-0.5 rounded-full text-[10px] font-bold ${
                        (item.maxDeviationPct || 0) > 10
                          ? 'bg-red-500/20 text-red-400 border border-red-500/30'
                          : 'bg-amber-500/20 text-amber-400 border border-amber-500/30'
                      }`}
                    >
                      Δ {(item.maxDeviationPct || 0).toFixed(1)}%
                    </span>
                  </div>

                  <div className="flex items-center gap-2 mt-3 text-[10px] text-slate-400">
                    <span className="px-1.5 py-0.5 bg-slate-800 rounded font-semibold text-slate-300">
                      {item.domainKey}
                    </span>
                    <span className="font-mono">{item.fieldName}</span>
                    <span className="ml-auto flex items-center gap-1 text-slate-400">
                      <Clock className="w-3 h-3" />
                      {new Date(item.createdAt).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })}
                    </span>
                  </div>
                </div>
              );
            })}

            {filteredExceptions.length === 0 && (
              <div className="p-12 text-center text-slate-400 text-xs">
                No active breaks found in the selected domain.
              </div>
            )}
          </div>
        </div>

        {/* Right Column: Multi-Vendor Inspection & Action Pane (8 Columns) */}
        {selectedException ? (
          <div className="col-span-8 overflow-y-auto p-6 space-y-6 bg-[#030914]">
            {/* Entity Header Card */}
            <div className="p-4 bg-slate-900/60 rounded-xl border border-slate-800 flex items-center justify-between">
              <div>
                <div className="flex items-center gap-2">
                  <span className="text-xs font-bold px-2 py-0.5 rounded bg-cyan-500/20 text-cyan-400 border border-cyan-500/30">
                    {selectedException.domainKey}
                  </span>
                  <h3 className="text-base font-bold text-slate-100 font-mono">
                    {selectedException.masterEntitySid}
                  </h3>
                </div>
                <p className="text-xs text-slate-400 mt-1">
                  Evaluating field: <span className="text-slate-200 font-semibold">{selectedException.fieldName}</span>
                  {' '} | Anomaly: <span className="text-amber-400 font-semibold">{selectedException.anomalyType}</span>
                </p>
              </div>

              <div className="text-right">
                <span className="text-[10px] text-slate-400 block uppercase font-semibold">Max Deviation</span>
                <span className="text-xl font-bold font-mono text-red-400">
                  +{(selectedException.maxDeviationPct || 0).toFixed(2)}%
                </span>
              </div>
            </div>

            {/* AI Data Steward Copilot Recommendation Card */}
            {selectedException.aiDiagnosis && (
              <div className="p-4 bg-gradient-to-r from-purple-950/30 via-slate-900/60 to-purple-950/20 border border-purple-500/40 rounded-xl space-y-3">
                <div className="flex items-center justify-between">
                  <span className="text-xs font-bold text-purple-300 uppercase tracking-wider flex items-center gap-1.5">
                    <Bot className="w-4 h-4 text-purple-400" />
                    MCP AI Data Steward Copilot Recommendation
                  </span>
                  <span className="px-2 py-0.5 bg-purple-500/20 border border-purple-500/40 text-purple-300 text-[10px] rounded-full font-bold">
                    Confidence: {(selectedException.aiDiagnosis.confidenceScore * 100).toFixed(0)}%
                  </span>
                </div>

                <p className="text-xs text-slate-200 leading-relaxed font-sans">
                  {selectedException.aiDiagnosis.rationale}
                </p>

                <div className="flex items-center justify-between pt-2 border-t border-purple-500/20">
                  <span className="text-xs text-purple-200">
                    Suggested Action: <strong className="text-white font-mono">{selectedException.aiDiagnosis.recommendation}</strong>
                  </span>
                  <button
                    onClick={() =>
                      handleApplyOverride(
                        selectedException.aiDiagnosis!.winningVendor,
                        selectedException.aiDiagnosis!.suggestedValue,
                        `AI Recommendation: ${selectedException.aiDiagnosis!.rationale}`
                      )
                    }
                    disabled={isSubmitting}
                    className="px-3.5 py-1.5 bg-purple-600 hover:bg-purple-500 text-white text-xs font-bold rounded-lg shadow transition flex items-center gap-1.5 disabled:opacity-50"
                  >
                    <Sparkles className="w-3.5 h-3.5" />
                    1-Click Apply AI Fix
                  </button>
                </div>
              </div>
            )}

            {/* Side-by-Side Competing Vendor Feed Grid */}
            <div className="space-y-3">
              <h4 className="text-xs font-bold text-slate-300 uppercase tracking-wider flex items-center gap-2">
                <Database className="w-4 h-4 text-cyan-400" />
                Competing Multi-Source Vendor Payloads
              </h4>

              <div className="grid grid-cols-3 gap-4">
                {(selectedException.competingValues || []).map((feed) => {
                  const isWinningCandidate = selectedException.aiDiagnosis?.winningVendor === feed.vendor;
                  return (
                    <div
                      key={feed.vendor}
                      className={`p-4 rounded-xl border transition-all flex flex-col justify-between ${
                        isWinningCandidate
                          ? 'bg-slate-900/80 border-cyan-500/60 shadow-lg shadow-cyan-950/30'
                          : 'bg-slate-900/40 border-slate-800'
                      }`}
                    >
                      <div>
                        <div className="flex items-center justify-between pb-2 border-b border-slate-800">
                          <span className="text-xs font-bold text-slate-100 uppercase tracking-wide">
                            {feed.vendor}
                          </span>
                          <span className="text-[10px] text-cyan-400 font-mono">
                            {((feed.confidence || 0.9) * 100).toFixed(0)}% Trust
                          </span>
                        </div>

                        <div className="my-4 text-center">
                          <span className="text-[10px] text-slate-400 block uppercase font-medium">Reported Value</span>
                          <span className="text-2xl font-bold font-mono text-slate-100">
                            {typeof feed.value === 'number' ? `$${feed.value.toFixed(2)}` : String(feed.value)}
                          </span>
                        </div>
                      </div>

                      <button
                        onClick={() =>
                          handleApplyOverride(
                            feed.vendor,
                            feed.value,
                            `Manual Steward selection: Accepted ${feed.vendor} feed`
                          )
                        }
                        disabled={isSubmitting}
                        className={`w-full py-2 text-xs font-bold rounded-lg transition flex items-center justify-center gap-1.5 ${
                          isWinningCandidate
                            ? 'bg-[#F5A623] hover:bg-amber-400 text-slate-950'
                            : 'bg-slate-800 hover:bg-slate-700 text-slate-200'
                        }`}
                      >
                        <CheckCircle2 className="w-3.5 h-3.5" />
                        Accept {feed.vendor}
                      </button>
                    </div>
                  );
                })}
              </div>
            </div>

            {/* Manual Custom Override & Reason Form */}
            <div className="p-4 bg-slate-900/50 rounded-xl border border-slate-800 space-y-4">
              <h4 className="text-xs font-bold text-slate-300 uppercase tracking-wider flex items-center gap-2">
                <Sliders className="w-4 h-4 text-amber-400" />
                Manual Data Steward Override & Audit Reason
              </h4>

              <div className="grid grid-cols-2 gap-4">
                <div>
                  <label className="text-xs font-semibold text-slate-300 mb-1.5 block">Custom Field Value</label>
                  <input
                    type="text"
                    value={customValue}
                    onChange={(e) => setCustomValue(e.target.value)}
                    placeholder="Enter explicit golden value..."
                    className="w-full bg-slate-950 border border-slate-800 rounded-lg px-3 py-2 text-xs text-slate-100 font-mono"
                  />
                </div>
                <div>
                  <label className="text-xs font-semibold text-slate-300 mb-1.5 block">Steward Override Rationale</label>
                  <input
                    type="text"
                    value={overrideReason}
                    onChange={(e) => setOverrideReason(e.target.value)}
                    placeholder="e.g. Verified with trading desk, IDC feed shifted decimal"
                    className="w-full bg-slate-950 border border-slate-800 rounded-lg px-3 py-2 text-xs text-slate-100"
                  />
                </div>
              </div>

              <div className="flex justify-end pt-2">
                <button
                  onClick={() => handleApplyOverride('MANUAL_STEWARD', customValue, overrideReason)}
                  disabled={isSubmitting || !customValue || !overrideReason}
                  className="px-4 py-2 bg-gradient-to-r from-amber-500 to-[#F5A623] text-slate-950 font-bold rounded-lg shadow hover:opacity-95 text-xs flex items-center gap-1.5 disabled:opacity-40 transition"
                >
                  <Send className="w-3.5 h-3.5" />
                  Commit Manual Override & Signal Workflow
                </button>
              </div>
            </div>
          </div>
        ) : (
          <div className="col-span-8 flex items-center justify-center text-slate-400 text-xs bg-[#030914]">
            Select an open exception from the left queue to review vendor feeds and apply break overrides.
          </div>
        )}
      </div>
    </div>
  );
};
