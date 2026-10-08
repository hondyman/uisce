import React, { useCallback, useEffect, useState } from 'react';
import {
  Layers,
  Search,
  RefreshCw,
  AlertTriangle,
  Check,
  X,
  Lock,
  RotateCcw,
  Info,
} from 'lucide-react';

export interface TenantRulesetStatus {
  ruleset_code: string;
  name: string;
  description: string;
  plan_tier: string;
  is_licensed: boolean;
  is_active: boolean;
  active_rules: number;
  total_rules: number;
}

export interface TenantActivationItem {
  rule_id: string;
  core_rule_id?: string;
  rule_code: string;
  rule_name: string;
  rule_phase: string;
  severity: string;
  citation: string;
  domain: string;
  jurisdictions: string[];
  ruleset_codes: string[];
  current_core_version: number;
  pinned_core_version: number;
  enabled: boolean;
  inherit_mode: 'inherit' | 'extend' | 'custom';
  drift_status: 'CURRENT' | 'CORE_VERSION_UPDATED' | 'DRIFT_DETECTED' | 'RECONCILED';
  core_thresholds: Record<string, string>;
  tenant_thresholds: Record<string, string>;
  core_content_hash: string;
  tenant_content_hash?: string;
  activated_by?: string;
  activated_at?: string;
}

export interface TenantActivationMatrixData {
  tenant_id: string;
  tenant_name: string;
  plan: string;
  gold_copy: boolean;
  rulesets: TenantRulesetStatus[];
  rules: TenantActivationItem[];
  total_active: number;
  total_rules: number;
  drift_count: number;
}

interface Props {
  tenantId?: string;
}

export const RuleActivationMatrix: React.FC<Props> = ({ tenantId = '00000000-0000-4000-a000-000000000002' }) => {
  const [matrix, setMatrix] = useState<TenantActivationMatrixData | null>(null);
  const [loading, setLoading] = useState(true);
  const [updatingRuleId, setUpdatingRuleId] = useState<string | null>(null);
  const [togglingRuleset, setTogglingRuleset] = useState<string | null>(null);

  // Filters
  const [searchQuery, setSearchQuery] = useState('');
  const [domainFilter, setDomainFilter] = useState('ALL');
  const [rulesetFilter, setRulesetFilter] = useState('ALL');
  const [statusFilter, setStatusFilter] = useState<'ALL' | 'ACTIVE' | 'INACTIVE' | 'DRIFT'>('ALL');

  // Repin Modal state
  const [repinModalRule, setRepinModalRule] = useState<TenantActivationItem | null>(null);
  const [stewardNotes, setStewardNotes] = useState('');
  const [repinning, setRepinning] = useState(false);

  // Parameter editing in extend mode
  const [editingParams, setEditingParams] = useState<Record<string, Record<string, string>>>({});

  const fetchMatrix = useCallback(async () => {
    setLoading(true);
    try {
      const res = await fetch(`/api/compliance/tenants/${tenantId}/activations`);
      if (res.ok) {
        const data = await res.json();
        setMatrix(data);
      }
    } catch (err) {
      console.error('Failed to fetch activation matrix', err);
    } finally {
      setLoading(false);
    }
  }, [tenantId]);

  useEffect(() => {
    fetchMatrix();
  }, [fetchMatrix]);

  const handleToggleRule = async (rule: TenantActivationItem, enabled: boolean) => {
    setUpdatingRuleId(rule.rule_id);
    try {
      const overrides = editingParams[rule.rule_id] || rule.tenant_thresholds || {};
      const res = await fetch(`/api/compliance/tenants/${tenantId}/activations/rules/${rule.rule_id}`, {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          enabled,
          inherit_mode: rule.inherit_mode,
          parameter_overrides: rule.inherit_mode === 'extend' ? overrides : undefined,
          actor_id: 'compliance_officer',
        }),
      });

      if (res.ok) {
        await fetchMatrix();
      }
    } catch (err) {
      console.error('Failed to update rule activation', err);
    } finally {
      setUpdatingRuleId(null);
    }
  };

  const handleModeChange = async (rule: TenantActivationItem, mode: 'inherit' | 'extend' | 'custom') => {
    setUpdatingRuleId(rule.rule_id);
    try {
      const overrides = editingParams[rule.rule_id] || rule.core_thresholds || {};
      const res = await fetch(`/api/compliance/tenants/${tenantId}/activations/rules/${rule.rule_id}`, {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          enabled: rule.enabled,
          inherit_mode: mode,
          parameter_overrides: mode === 'extend' ? overrides : undefined,
          actor_id: 'compliance_officer',
        }),
      });

      if (res.ok) {
        await fetchMatrix();
      }
    } catch (err) {
      console.error('Failed to switch inheritance mode', err);
    } finally {
      setUpdatingRuleId(null);
    }
  };

  const handleSaveParameters = async (rule: TenantActivationItem) => {
    setUpdatingRuleId(rule.rule_id);
    try {
      const overrides = editingParams[rule.rule_id] || rule.tenant_thresholds || {};
      const res = await fetch(`/api/compliance/tenants/${tenantId}/activations/rules/${rule.rule_id}`, {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          enabled: rule.enabled,
          inherit_mode: 'extend',
          parameter_overrides: overrides,
          actor_id: 'compliance_officer',
        }),
      });

      if (res.ok) {
        // Clear editing state for this rule
        const next = { ...editingParams };
        delete next[rule.rule_id];
        setEditingParams(next);
        await fetchMatrix();
      }
    } catch (err) {
      console.error('Failed to save parameter overrides', err);
    } finally {
      setUpdatingRuleId(null);
    }
  };

  const handleToggleRuleset = async (rulesetCode: string, enabled: boolean) => {
    setTogglingRuleset(rulesetCode);
    try {
      const res = await fetch(`/api/compliance/tenants/${tenantId}/activations/rulesets/${rulesetCode}/toggle`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          enabled,
          actor_id: 'compliance_officer',
        }),
      });

      if (res.ok) {
        await fetchMatrix();
      }
    } catch (err) {
      console.error('Failed to toggle ruleset', err);
    } finally {
      setTogglingRuleset(null);
    }
  };

  const handleRepinSubmit = async () => {
    if (!repinModalRule) return;
    setRepinning(true);
    try {
      const res = await fetch(`/api/compliance/tenants/${tenantId}/activations/rules/${repinModalRule.rule_id}/repin`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          target_version: repinModalRule.current_core_version,
          steward_notes: stewardNotes || 'Re-anchored extended rule to latest upstream core release.',
          actor_id: 'compliance_officer',
        }),
      });

      if (res.ok) {
        setRepinModalRule(null);
        setStewardNotes('');
        await fetchMatrix();
      }
    } catch (err) {
      console.error('Failed to repin rule', err);
    } finally {
      setRepinning(false);
    }
  };

  if (loading || !matrix) {
    return (
      <div className="flex items-center justify-center h-full bg-slate-950 text-slate-400 font-mono text-xs">
        <RefreshCw className="w-6 h-6 animate-spin mr-2 text-emerald-400" />
        Loading tenant compliance activation matrix...
      </div>
    );
  }

  const domains = Array.from(new Set(matrix.rules.map((r) => r.domain))).filter(Boolean);

  const filteredRules = matrix.rules.filter((rule) => {
    if (searchQuery) {
      const q = searchQuery.toLowerCase();
      const matchCode = rule.rule_code.toLowerCase().includes(q);
      const matchName = rule.rule_name.toLowerCase().includes(q);
      const matchCitation = rule.citation.toLowerCase().includes(q);
      if (!matchCode && !matchName && !matchCitation) return false;
    }

    if (domainFilter !== 'ALL' && rule.domain !== domainFilter) return false;
    if (rulesetFilter !== 'ALL' && !rule.ruleset_codes.includes(rulesetFilter)) return false;

    if (statusFilter === 'ACTIVE' && !rule.enabled) return false;
    if (statusFilter === 'INACTIVE' && rule.enabled) return false;
    if (statusFilter === 'DRIFT' && rule.drift_status === 'CURRENT') return false;

    return true;
  });

  return (
    <div className="flex flex-col h-full bg-slate-950 text-slate-100 font-mono text-sm">
      {/* Header & Tenant Plan Info */}
      <div className="flex flex-col gap-4 p-4 border-b border-slate-800 bg-slate-900/60">
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-3">
            <div className="p-2 bg-indigo-950/60 border border-indigo-800/60 rounded-lg text-indigo-400">
              <Layers className="w-6 h-6" />
            </div>
            <div>
              <div className="flex items-center gap-2">
                <h1 className="text-lg font-bold text-slate-100 tracking-tight">{matrix.tenant_name}</h1>
                <span className="text-xs font-semibold px-2 py-0.5 bg-indigo-950 text-indigo-300 border border-indigo-800 rounded uppercase">
                  {matrix.plan} Plan
                </span>
                {matrix.gold_copy && (
                  <span className="text-xs font-semibold px-2 py-0.5 bg-amber-950 text-amber-300 border border-amber-800 rounded">
                    Master Gold Copy
                  </span>
                )}
              </div>
              <p className="text-xs text-slate-400">
                Multi-tenant opt-in activation matrix with inheritance control and version-drift reconciliation
              </p>
            </div>
          </div>

          <button
            onClick={fetchMatrix}
            className="flex items-center gap-2 px-3 py-1.5 bg-slate-800 hover:bg-slate-700 rounded border border-slate-700 text-slate-200 text-xs transition-colors"
          >
            <RefreshCw className="w-3.5 h-3.5" />
            Refresh Matrix
          </button>
        </div>

        {/* Drift Alert Banner */}
        {matrix.drift_count > 0 && (
          <div className="p-3 bg-amber-950/40 border border-amber-700/60 rounded-lg flex items-center justify-between animate-pulse">
            <div className="flex items-center gap-2.5 text-amber-300 text-xs">
              <AlertTriangle className="w-4 h-4 text-amber-400" />
              <span>
                <strong>{matrix.drift_count} Extended Rule(s) Outdated:</strong> Upstream Gold-Copy Core rules have advanced to newer versions.
              </span>
            </div>
            <button
              onClick={() => setStatusFilter('DRIFT')}
              className="px-2.5 py-1 bg-amber-900 hover:bg-amber-800 rounded text-amber-100 text-xs font-medium transition-colors"
            >
              Filter Outdated Rules
            </button>
          </div>
        )}

        {/* Ruleset Licensing Cards */}
        <div className="grid grid-cols-3 gap-3">
          {matrix.rulesets.map((rs) => (
            <div
              key={rs.ruleset_code}
              className={`p-3 rounded-lg border flex flex-col justify-between ${
                rs.is_active
                  ? 'bg-emerald-950/20 border-emerald-800/80'
                  : 'bg-slate-900/80 border-slate-800'
              }`}
            >
              <div>
                <div className="flex items-center justify-between mb-1">
                  <span className="font-bold text-xs text-slate-100">{rs.name}</span>
                  {rs.is_licensed ? (
                    <span className="text-[10px] px-1.5 py-0.5 bg-emerald-950 text-emerald-400 border border-emerald-800 rounded">
                      LICENSED
                    </span>
                  ) : (
                    <span className="text-[10px] px-1.5 py-0.5 bg-zinc-900 text-zinc-500 border border-zinc-800 rounded flex items-center gap-1">
                      <Lock className="w-2.5 h-2.5" /> REQUIRES {rs.plan_tier.toUpperCase()}
                    </span>
                  )}
                </div>
                <p className="text-[11px] text-slate-400 line-clamp-2 leading-relaxed mb-3">{rs.description}</p>
              </div>

              <div className="flex items-center justify-between pt-2 border-t border-slate-800/80">
                <div className="text-xs text-slate-300">
                  <span className="font-bold text-emerald-400">{rs.active_rules}</span> / {rs.total_rules} active
                </div>

                <button
                  disabled={!rs.is_licensed || togglingRuleset === rs.ruleset_code}
                  onClick={() => handleToggleRuleset(rs.ruleset_code, !rs.is_active)}
                  className={`px-2.5 py-1 rounded text-xs font-semibold transition-colors flex items-center gap-1.5 ${
                    !rs.is_licensed
                      ? 'bg-slate-800 text-slate-600 cursor-not-allowed'
                      : rs.is_active
                      ? 'bg-red-950/80 hover:bg-red-900 text-red-300 border border-red-800'
                      : 'bg-emerald-950/80 hover:bg-emerald-900 text-emerald-300 border border-emerald-800'
                  }`}
                >
                  {togglingRuleset === rs.ruleset_code ? (
                    <RefreshCw className="w-3 h-3 animate-spin" />
                  ) : rs.is_active ? (
                    'Deactivate Pack'
                  ) : (
                    'Activate All'
                  )}
                </button>
              </div>
            </div>
          ))}
        </div>

        {/* Filters Row */}
        <div className="flex flex-wrap items-center gap-2 pt-2 border-t border-slate-800/80">
          <div className="relative flex-1 min-w-[200px]">
            <Search className="w-4 h-4 absolute left-2.5 top-2.5 text-slate-500" />
            <input
              type="text"
              placeholder="Search tenant rule activation matrix..."
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
              className="w-full pl-9 pr-3 py-1.5 bg-slate-900 border border-slate-800 rounded text-xs text-slate-200 placeholder-slate-500 focus:outline-none focus:border-emerald-600"
            />
          </div>

          <select
            value={domainFilter}
            onChange={(e) => setDomainFilter(e.target.value)}
            className="px-2.5 py-1.5 bg-slate-900 border border-slate-800 rounded text-xs text-slate-300 focus:outline-none focus:border-emerald-600"
          >
            <option value="ALL">All Domains ({domains.length})</option>
            {domains.map((d) => (
              <option key={d} value={d}>
                {d}
              </option>
            ))}
          </select>

          <select
            value={rulesetFilter}
            onChange={(e) => setRulesetFilter(e.target.value)}
            className="px-2.5 py-1.5 bg-slate-900 border border-slate-800 rounded text-xs text-slate-300 focus:outline-none focus:border-emerald-600"
          >
            <option value="ALL">All Rulesets</option>
            {matrix.rulesets.map((rs) => (
              <option key={rs.ruleset_code} value={rs.ruleset_code}>
                {rs.ruleset_code}
              </option>
            ))}
          </select>

          <select
            value={statusFilter}
            onChange={(e) => setStatusFilter(e.target.value as any)}
            className="px-2.5 py-1.5 bg-slate-900 border border-slate-800 rounded text-xs text-slate-300 focus:outline-none focus:border-emerald-600"
          >
            <option value="ALL">All Statuses ({matrix.total_rules})</option>
            <option value="ACTIVE">Active Only ({matrix.total_active})</option>
            <option value="INACTIVE">Inactive Only ({matrix.total_rules - matrix.total_active})</option>
            <option value="DRIFT">Outdated / Drifted ({matrix.drift_count})</option>
          </select>
        </div>
      </div>

      {/* Rules Matrix Table */}
      <div className="flex-1 overflow-auto p-4">
        <div className="border border-slate-800 rounded-lg overflow-hidden bg-slate-900/40">
          <table className="w-full text-left border-collapse">
            <thead>
              <tr className="border-b border-slate-800 bg-slate-900/90 text-slate-400 text-xs font-semibold">
                <th className="py-2.5 px-3 w-12 text-center">Active</th>
                <th className="py-2.5 px-3">Rule Definition</th>
                <th className="py-2.5 px-3">Domain</th>
                <th className="py-2.5 px-3">Inheritance Mode</th>
                <th className="py-2.5 px-3">Version & Drift</th>
                <th className="py-2.5 px-3">Threshold Configuration</th>
                <th className="py-2.5 px-3 text-right">Actions</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-slate-800/60 text-xs">
              {filteredRules.map((rule) => {
                const isExtended = rule.inherit_mode === 'extend';
                const hasDrift = rule.drift_status === 'CORE_VERSION_UPDATED';
                const isEditing = !!editingParams[rule.rule_id];

                return (
                  <tr key={rule.rule_id} className="hover:bg-slate-800/30 transition-colors">
                    {/* Toggle */}
                    <td className="py-3 px-3 text-center">
                      <input
                        type="checkbox"
                        checked={rule.enabled}
                        disabled={updatingRuleId === rule.rule_id}
                        onChange={(e) => handleToggleRule(rule, e.target.checked)}
                        className="w-4 h-4 rounded bg-slate-900 border-slate-700 text-emerald-500 focus:ring-0 focus:ring-offset-0 cursor-pointer accent-emerald-500"
                      />
                    </td>

                    {/* Rule Definition */}
                    <td className="py-3 px-3">
                      <div className="font-bold text-slate-100">{rule.rule_code}</div>
                      <div className="text-slate-400 text-[11px] truncate max-w-[240px]">{rule.rule_name}</div>
                      <div className="text-slate-500 text-[10px] font-mono mt-0.5">{rule.citation}</div>
                    </td>

                    {/* Domain */}
                    <td className="py-3 px-3">
                      <span className="text-slate-300">{rule.domain}</span>
                    </td>

                    {/* Inheritance Mode */}
                    <td className="py-3 px-3">
                      <select
                        value={rule.inherit_mode}
                        disabled={updatingRuleId === rule.rule_id}
                        onChange={(e) => handleModeChange(rule, e.target.value as any)}
                        className={`px-2 py-1 rounded text-xs font-semibold focus:outline-none ${
                          rule.inherit_mode === 'inherit'
                            ? 'bg-slate-800 text-slate-300 border border-slate-700'
                            : 'bg-amber-950/80 text-amber-300 border border-amber-800'
                        }`}
                      >
                        <option value="inherit">inherit (Core Standard)</option>
                        <option value="extend">extend (Custom Limits)</option>
                      </select>
                    </td>

                    {/* Version & Drift */}
                    <td className="py-3 px-3">
                      <div className="space-y-1">
                        <div className="flex items-center gap-1.5">
                          <span className="text-slate-400">Pinned:</span>
                          <span className="font-bold text-slate-200">v{rule.pinned_core_version}</span>
                          <span className="text-slate-500">/ Core: v{rule.current_core_version}</span>
                        </div>
                        {hasDrift ? (
                          <span className="inline-flex items-center px-1.5 py-0.5 rounded text-[10px] font-bold bg-amber-950 text-amber-300 border border-amber-700">
                            <AlertTriangle className="w-3 h-3 mr-1 text-amber-400" />
                            CORE UPGRADED
                          </span>
                        ) : (
                          <span className="inline-flex items-center px-1.5 py-0.5 rounded text-[10px] font-medium bg-emerald-950 text-emerald-400 border border-emerald-800">
                            <Check className="w-2.5 h-2.5 mr-1" />
                            CURRENT
                          </span>
                        )}
                      </div>
                    </td>

                    {/* Thresholds */}
                    <td className="py-3 px-3">
                      {isExtended ? (
                        <div className="space-y-1.5 max-w-[240px]">
                          {Object.entries(rule.core_thresholds || {}).map(([param, defaultVal]) => {
                            const customVal =
                              editingParams[rule.rule_id]?.[param] ?? rule.tenant_thresholds?.[param] ?? defaultVal;

                            return (
                              <div key={param} className="flex items-center justify-between gap-2">
                                <span className="text-[10px] text-slate-400 truncate" title={param}>
                                  {param}:
                                </span>
                                <input
                                  type="text"
                                  value={customVal}
                                  onChange={(e) => {
                                    const next = {
                                      ...editingParams,
                                      [rule.rule_id]: {
                                        ...(editingParams[rule.rule_id] || rule.tenant_thresholds || {}),
                                        [param]: e.target.value,
                                      },
                                    };
                                    setEditingParams(next);
                                  }}
                                  className="w-20 px-1.5 py-0.5 bg-slate-900 border border-amber-800/80 rounded text-[11px] text-amber-300 font-mono focus:outline-none focus:border-amber-500 text-right"
                                />
                              </div>
                            );
                          })}
                          {isEditing && (
                            <button
                              onClick={() => handleSaveParameters(rule)}
                              disabled={updatingRuleId === rule.rule_id}
                              className="w-full px-2 py-0.5 bg-amber-900 hover:bg-amber-800 rounded text-[10px] font-semibold text-amber-100 transition-colors"
                            >
                              Save Overrides
                            </button>
                          )}
                        </div>
                      ) : (
                        <div className="text-[11px] text-slate-400 font-mono">
                          {Object.entries(rule.core_thresholds || {})
                            .map(([k, v]) => `${k}=${v}`)
                            .join(', ') || 'Predicate only'}
                        </div>
                      )}
                    </td>

                    {/* Actions */}
                    <td className="py-3 px-3 text-right">
                      {hasDrift && (
                        <button
                          onClick={() => setRepinModalRule(rule)}
                          className="px-2.5 py-1 bg-amber-900/80 hover:bg-amber-800 rounded text-xs font-semibold text-amber-200 border border-amber-700 transition-colors flex items-center gap-1 ml-auto"
                        >
                          <RotateCcw className="w-3 h-3" />
                          Repin to v{rule.current_core_version}
                        </button>
                      )}
                    </td>
                  </tr>
                );
              })}
            </tbody>
          </table>
        </div>
      </div>

      {/* Repin Confirmation Modal */}
      {repinModalRule && (
        <div className="fixed inset-0 bg-black/70 backdrop-blur-sm z-50 flex items-center justify-center p-4">
          <div className="bg-slate-900 border border-slate-800 rounded-xl max-w-lg w-full p-5 space-y-4 shadow-2xl animate-in zoom-in-95 duration-150">
            <div className="flex items-center justify-between border-b border-slate-800 pb-3">
              <div className="flex items-center gap-2 text-amber-400 font-bold">
                <RotateCcw className="w-5 h-5" />
                <span>Repin Extended Rule & Reconcile Drift</span>
              </div>
              <button
                onClick={() => setRepinModalRule(null)}
                className="p-1 hover:bg-slate-800 rounded text-slate-400 hover:text-slate-200 transition-colors"
              >
                <X className="w-4 h-4" />
              </button>
            </div>

            <div className="space-y-3 text-xs">
              <div className="p-3 bg-slate-950 border border-slate-800 rounded space-y-1.5">
                <div className="font-bold text-slate-100">{repinModalRule.rule_code}</div>
                <div className="text-slate-400">{repinModalRule.rule_name}</div>
                <div className="flex items-center gap-3 pt-1 font-mono text-[11px]">
                  <span className="text-slate-400">Current Pinned: v{repinModalRule.pinned_core_version}</span>
                  <span className="text-emerald-400 font-bold">Target Core: v{repinModalRule.current_core_version}</span>
                </div>
              </div>

              <div className="space-y-1">
                <label htmlFor="steward-notes" className="text-slate-300 font-semibold">Compliance Steward Governance Notes</label>
                <textarea
                  id="steward-notes"
                  rows={3}
                  value={stewardNotes}
                  onChange={(e) => setStewardNotes(e.target.value)}
                  placeholder="State rationale for re-anchoring extended rule to upstream core release..."
                  className="w-full p-2.5 bg-slate-950 border border-slate-800 rounded text-slate-200 placeholder-slate-600 focus:outline-none focus:border-amber-600 font-mono text-xs"
                />
              </div>

              <div className="p-3 bg-emerald-950/30 border border-emerald-800/40 rounded text-[11px] text-emerald-300 flex items-start gap-2">
                <Info className="w-4 h-4 text-emerald-400 mt-0.5 flex-shrink-0" />
                <span>
                  Repinning creates an immutable companion snapshot, updates the AST predicate to Core v{repinModalRule.current_core_version}, preserves custom tenant thresholds, and records a governance audit event.
                </span>
              </div>
            </div>

            <div className="flex items-center justify-end gap-2 pt-3 border-t border-slate-800">
              <button
                onClick={() => setRepinModalRule(null)}
                className="px-3 py-1.5 bg-slate-800 hover:bg-slate-700 rounded text-slate-300 text-xs font-medium transition-colors"
              >
                Cancel
              </button>
              <button
                disabled={repinning}
                onClick={handleRepinSubmit}
                className="px-4 py-1.5 bg-amber-600 hover:bg-amber-500 rounded text-slate-950 font-bold text-xs transition-colors flex items-center gap-1.5 shadow-lg shadow-amber-900/30"
              >
                {repinning ? <RefreshCw className="w-3.5 h-3.5 animate-spin" /> : <Check className="w-3.5 h-3.5" />}
                Confirm & Reconcile Drift
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
};
