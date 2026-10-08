import React, { useCallback, useEffect, useState } from 'react';
import {
  BookOpen,
  Search,
  RefreshCw,
  Shield,
  Layers,
  CheckCircle2,
  FileCode,
  Clock,
  ChevronRight,
  X,
  Sliders,
  Copy,
  Check,
  Database,
  Terminal,
  Activity,
  Award,
} from 'lucide-react';

export interface CoreRuleSummary {
  id: string;
  rule_code: string;
  name: string;
  description: string;
  rule_phase: 'PRE_TRADE' | 'POST_TRADE' | 'BOTH' | string;
  severity: 'HARD_BLOCK' | 'SOFT_WARNING' | 'APPROVAL_REQUIRED' | string;
  priority: number;
  jurisdictions: string[];
  citation: string;
  current_version: number;
  content_hash: string;
  library_status: 'ACTIVE' | 'PROVISIONAL' | 'DEPRECATED' | 'STALE_REGULATION' | string;
  effective_from: string;
  effective_to?: string;
  ruleset_codes: string[];
  domain: string;
}

export interface RuleVersionSummary {
  version: number;
  effective_from: string;
  effective_to?: string;
  citation: string;
  content_hash: string;
  created_by: string;
  created_at: string;
}

export interface ScenarioTestCase {
  RuleCode: string;
  Code: string;
  Description: string;
  Expected: {
    Status: string;
    MustContain?: string;
  };
}

export interface RuleDetails extends CoreRuleSummary {
  ast_condition: Record<string, any>;
  parameter_thresholds: Record<string, string>;
  version_history: RuleVersionSummary[];
  sample_test_cases?: ScenarioTestCase[];
}

export interface RulesetSummary {
  ruleset_code: string;
  name: string;
  description: string;
  plan_tier: string;
  total_rules: number;
  rule_ids: string[];
}

export const RuleLibraryExplorer: React.FC = () => {
  const [rules, setRules] = useState<CoreRuleSummary[]>([]);
  const [rulesets, setRulesets] = useState<RulesetSummary[]>([]);
  const [loading, setLoading] = useState(true);
  const [selectedRule, setSelectedRule] = useState<RuleDetails | null>(null);
  const [loadingDetails, setLoadingDetails] = useState(false);
  const [copiedHash, setCopiedHash] = useState<string | null>(null);

  // Filters
  const [searchQuery, setSearchQuery] = useState('');
  const [selectedDomain, setSelectedDomain] = useState<string>('ALL');
  const [selectedRuleset, setSelectedRuleset] = useState<string>('ALL');
  const [selectedPhase, setSelectedPhase] = useState<string>('ALL');
  const [selectedSeverity, setSelectedSeverity] = useState<string>('ALL');
  const [selectedJurisdiction, setSelectedJurisdiction] = useState<string>('ALL');

  const fetchRules = useCallback(async () => {
    setLoading(true);
    try {
      const params = new URLSearchParams();
      if (searchQuery) params.set('search_query', searchQuery);
      if (selectedDomain !== 'ALL') params.set('domain', selectedDomain);
      if (selectedRuleset !== 'ALL') params.set('ruleset_code', selectedRuleset);
      if (selectedPhase !== 'ALL') params.set('rule_phase', selectedPhase);
      if (selectedSeverity !== 'ALL') params.set('severity', selectedSeverity);
      if (selectedJurisdiction !== 'ALL') params.set('jurisdiction', selectedJurisdiction);

      const [rulesRes, rulesetsRes] = await Promise.all([
        fetch(`/api/compliance/library/rules?${params.toString()}`),
        fetch('/api/compliance/library/rulesets'),
      ]);

      if (rulesRes.ok) {
        const data = await rulesRes.json();
        setRules(data.data || []);
      }
      if (rulesetsRes.ok) {
        const data = await rulesetsRes.json();
        setRulesets(data.data || []);
      }
    } catch (err) {
      console.error('Failed to fetch rules library', err);
    } finally {
      setLoading(false);
    }
  }, [searchQuery, selectedDomain, selectedRuleset, selectedPhase, selectedSeverity, selectedJurisdiction]);

  useEffect(() => {
    fetchRules();
  }, [fetchRules]);

  const fetchRuleDetails = async (id: string) => {
    setLoadingDetails(true);
    try {
      const res = await fetch(`/api/compliance/library/rules/${id}`);
      if (res.ok) {
        const data = await res.json();
        setSelectedRule(data);
      }
    } catch (err) {
      console.error('Failed to fetch rule details', err);
    } finally {
      setLoadingDetails(false);
    }
  };

  const copyToClipboard = (text: string) => {
    navigator.clipboard.writeText(text);
    setCopiedHash(text);
    setTimeout(() => setCopiedHash(null), 2000);
  };

  const domains = Array.from(new Set(rules.map((r) => r.domain))).filter(Boolean);

  const getSeverityBadge = (severity: string) => {
    switch (severity) {
      case 'HARD_BLOCK':
        return (
          <span className="inline-flex items-center px-2 py-0.5 rounded text-xs font-medium bg-red-950/60 text-red-400 border border-red-800/60">
            HARD BLOCK
          </span>
        );
      case 'SOFT_WARNING':
        return (
          <span className="inline-flex items-center px-2 py-0.5 rounded text-xs font-medium bg-amber-950/60 text-amber-400 border border-amber-800/60">
            SOFT WARNING
          </span>
        );
      case 'APPROVAL_REQUIRED':
        return (
          <span className="inline-flex items-center px-2 py-0.5 rounded text-xs font-medium bg-purple-950/60 text-purple-400 border border-purple-800/60">
            APPROVAL REQ
          </span>
        );
      default:
        return (
          <span className="inline-flex items-center px-2 py-0.5 rounded text-xs font-medium bg-slate-800 text-slate-300">
            {severity}
          </span>
        );
    }
  };

  const getPhaseBadge = (phase: string) => {
    switch (phase) {
      case 'PRE_TRADE':
        return (
          <span className="inline-flex items-center px-2 py-0.5 rounded text-xs font-medium bg-emerald-950/60 text-emerald-400 border border-emerald-800/60">
            PRE-TRADE
          </span>
        );
      case 'POST_TRADE':
        return (
          <span className="inline-flex items-center px-2 py-0.5 rounded text-xs font-medium bg-cyan-950/60 text-cyan-400 border border-cyan-800/60">
            POST-TRADE
          </span>
        );
      case 'BOTH':
        return (
          <span className="inline-flex items-center px-2 py-0.5 rounded text-xs font-medium bg-indigo-950/60 text-indigo-400 border border-indigo-800/60">
            PRE & POST
          </span>
        );
      default:
        return (
          <span className="inline-flex items-center px-2 py-0.5 rounded text-xs font-medium bg-slate-800 text-slate-400">
            {phase}
          </span>
        );
    }
  };

  const getStatusBadge = (status: string) => {
    switch (status) {
      case 'ACTIVE':
        return (
          <span className="inline-flex items-center px-2 py-0.5 rounded text-xs font-medium bg-emerald-950/80 text-emerald-300 border border-emerald-700/80">
            <CheckCircle2 className="w-3 h-3 mr-1" />
            ACTIVE
          </span>
        );
      case 'PROVISIONAL':
        return (
          <span className="inline-flex items-center px-2 py-0.5 rounded text-xs font-medium bg-blue-950/80 text-blue-300 border border-blue-700/80">
            PROVISIONAL
          </span>
        );
      case 'DEPRECATED':
        return (
          <span className="inline-flex items-center px-2 py-0.5 rounded text-xs font-medium bg-zinc-900 text-zinc-500 border border-zinc-800">
            DEPRECATED
          </span>
        );
      default:
        return null;
    }
  };

  return (
    <div className="flex flex-col h-full bg-slate-950 text-slate-100 font-mono text-sm">
      {/* Header */}
      <div className="flex flex-col gap-3 p-4 border-b border-slate-800 bg-slate-900/60">
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-3">
            <div className="p-2 bg-emerald-950/60 border border-emerald-800/60 rounded-lg text-emerald-400">
              <BookOpen className="w-6 h-6" />
            </div>
            <div>
              <h1 className="text-lg font-bold text-slate-100 tracking-tight flex items-center gap-2">
                Core Compliance Rule Library
                <span className="text-xs font-semibold px-2 py-0.5 bg-emerald-950 text-emerald-400 border border-emerald-800 rounded">
                  Gold Copy v1.0
                </span>
              </h1>
              <p className="text-xs text-slate-400">
                Deterministic regulatory library with cryptographic RFC 8785 AST and parameter immutability
              </p>
            </div>
          </div>
          <button
            onClick={fetchRules}
            disabled={loading}
            className="flex items-center gap-2 px-3 py-1.5 bg-slate-800 hover:bg-slate-700 active:bg-slate-600 rounded border border-slate-700 text-slate-200 transition-colors text-xs"
          >
            <RefreshCw className={`w-3.5 h-3.5 ${loading ? 'animate-spin text-emerald-400' : ''}`} />
            Refresh Catalog
          </button>
        </div>

        {/* Metrics Row */}
        <div className="grid grid-cols-4 gap-3">
          <div className="p-3 bg-slate-900/80 border border-slate-800 rounded flex items-center justify-between">
            <div>
              <div className="text-xs text-slate-400 uppercase tracking-wider">Total Core Rules</div>
              <div className="text-xl font-bold text-slate-100">{rules.length}</div>
            </div>
            <Shield className="w-6 h-6 text-emerald-500/60" />
          </div>
          <div className="p-3 bg-slate-900/80 border border-slate-800 rounded flex items-center justify-between">
            <div>
              <div className="text-xs text-slate-400 uppercase tracking-wider">Licensable Rulesets</div>
              <div className="text-xl font-bold text-slate-100">{rulesets.length}</div>
            </div>
            <Layers className="w-6 h-6 text-cyan-500/60" />
          </div>
          <div className="p-3 bg-slate-900/80 border border-slate-800 rounded flex items-center justify-between">
            <div>
              <div className="text-xs text-slate-400 uppercase tracking-wider">Regulatory Domains</div>
              <div className="text-xl font-bold text-slate-100">{domains.length}</div>
            </div>
            <Database className="w-6 h-6 text-indigo-500/60" />
          </div>
          <div className="p-3 bg-slate-900/80 border border-slate-800 rounded flex items-center justify-between">
            <div>
              <div className="text-xs text-slate-400 uppercase tracking-wider">Active Standard</div>
              <div className="text-xl font-bold text-emerald-400">
                {rules.filter((r) => r.library_status === 'ACTIVE').length} / {rules.length}
              </div>
            </div>
            <Award className="w-6 h-6 text-amber-500/60" />
          </div>
        </div>

        {/* Filter Toolbar */}
        <div className="flex flex-wrap items-center gap-2 pt-2 border-t border-slate-800/80">
          <div className="relative flex-1 min-w-[200px]">
            <Search className="w-4 h-4 absolute left-2.5 top-2.5 text-slate-500" />
            <input
              type="text"
              placeholder="Search by rule code, name, citation, or legal reference..."
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
              className="w-full pl-9 pr-3 py-1.5 bg-slate-900 border border-slate-800 rounded text-xs text-slate-200 placeholder-slate-500 focus:outline-none focus:border-emerald-600"
            />
          </div>

          <select
            value={selectedDomain}
            onChange={(e) => setSelectedDomain(e.target.value)}
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
            value={selectedRuleset}
            onChange={(e) => setSelectedRuleset(e.target.value)}
            className="px-2.5 py-1.5 bg-slate-900 border border-slate-800 rounded text-xs text-slate-300 focus:outline-none focus:border-emerald-600"
          >
            <option value="ALL">All Rulesets</option>
            {rulesets.map((rs) => (
              <option key={rs.ruleset_code} value={rs.ruleset_code}>
                {rs.ruleset_code} ({rs.total_rules} rules)
              </option>
            ))}
          </select>

          <select
            value={selectedPhase}
            onChange={(e) => setSelectedPhase(e.target.value)}
            className="px-2.5 py-1.5 bg-slate-900 border border-slate-800 rounded text-xs text-slate-300 focus:outline-none focus:border-emerald-600"
          >
            <option value="ALL">All Phases</option>
            <option value="PRE_TRADE">Pre-Trade</option>
            <option value="POST_TRADE">Post-Trade</option>
            <option value="BOTH">Pre & Post</option>
          </select>

          <select
            value={selectedSeverity}
            onChange={(e) => setSelectedSeverity(e.target.value)}
            className="px-2.5 py-1.5 bg-slate-900 border border-slate-800 rounded text-xs text-slate-300 focus:outline-none focus:border-emerald-600"
          >
            <option value="ALL">All Severities</option>
            <option value="HARD_BLOCK">Hard Block</option>
            <option value="SOFT_WARNING">Soft Warning</option>
            <option value="APPROVAL_REQUIRED">Approval Required</option>
          </select>

          <select
            value={selectedJurisdiction}
            onChange={(e) => setSelectedJurisdiction(e.target.value)}
            className="px-2.5 py-1.5 bg-slate-900 border border-slate-800 rounded text-xs text-slate-300 focus:outline-none focus:border-emerald-600"
          >
            <option value="ALL">All Jurisdictions</option>
            <option value="US">United States (SEC/FINRA/IRS)</option>
            <option value="EU">European Union (ESMA/UCITS/MiFID)</option>
            <option value="UK">United Kingdom (FCA)</option>
            <option value="GLOBAL">Global / Multi-Jurisdictional</option>
          </select>
        </div>
      </div>

      {/* Main Content: Table View */}
      <div className="flex-1 overflow-auto p-4">
        {loading ? (
          <div className="flex items-center justify-center h-64 text-slate-500">
            <RefreshCw className="w-6 h-6 animate-spin mr-2 text-emerald-400" />
            Loading compliance library catalog...
          </div>
        ) : rules.length === 0 ? (
          <div className="flex flex-col items-center justify-center h-64 text-slate-500">
            <Search className="w-8 h-8 mb-2 text-slate-600" />
            <p>No compliance rules match the current filters.</p>
          </div>
        ) : (
          <div className="border border-slate-800 rounded-lg overflow-hidden bg-slate-900/40">
            <table className="w-full text-left border-collapse">
              <thead>
                <tr className="border-b border-slate-800 bg-slate-900/90 text-slate-400 text-xs font-semibold">
                  <th className="py-2.5 px-3">Rule Code & Name</th>
                  <th className="py-2.5 px-3">Domain</th>
                  <th className="py-2.5 px-3">Phase</th>
                  <th className="py-2.5 px-3">Severity</th>
                  <th className="py-2.5 px-3">Jurisdictions</th>
                  <th className="py-2.5 px-3">Version & Hash</th>
                  <th className="py-2.5 px-3">Status</th>
                  <th className="py-2.5 px-3 text-right">Actions</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-slate-800/60 text-xs">
                {rules.map((rule) => (
                  <tr
                    key={rule.id}
                    onClick={() => fetchRuleDetails(rule.id)}
                    className="hover:bg-slate-800/40 cursor-pointer transition-colors group"
                  >
                    <td className="py-2.5 px-3">
                      <div className="font-bold text-slate-100 group-hover:text-emerald-400 transition-colors">
                        {rule.rule_code}
                      </div>
                      <div className="text-slate-400 text-[11px] truncate max-w-[280px]">{rule.name}</div>
                    </td>
                    <td className="py-2.5 px-3">
                      <span className="text-slate-300">{rule.domain}</span>
                    </td>
                    <td className="py-2.5 px-3">{getPhaseBadge(rule.rule_phase)}</td>
                    <td className="py-2.5 px-3">{getSeverityBadge(rule.severity)}</td>
                    <td className="py-2.5 px-3">
                      <div className="flex flex-wrap gap-1">
                        {rule.jurisdictions.map((j) => (
                          <span
                            key={j}
                            className="px-1.5 py-0.5 bg-slate-800 border border-slate-700 rounded text-[10px] text-slate-300"
                          >
                            {j}
                          </span>
                        ))}
                      </div>
                    </td>
                    <td className="py-2.5 px-3">
                      <div className="flex items-center gap-1.5">
                        <span className="font-semibold text-slate-300">v{rule.current_version}</span>
                        <span className="text-slate-500 font-mono text-[10px]" title={rule.content_hash}>
                          ({rule.content_hash.substring(0, 8)}…)
                        </span>
                      </div>
                    </td>
                    <td className="py-2.5 px-3">{getStatusBadge(rule.library_status)}</td>
                    <td className="py-2.5 px-3 text-right">
                      <button
                        onClick={(e) => {
                          e.stopPropagation();
                          fetchRuleDetails(rule.id);
                        }}
                        className="p-1.5 hover:bg-slate-700 rounded text-slate-400 hover:text-slate-100 transition-colors"
                        title="View Complete Rule Definition & AST"
                      >
                        <ChevronRight className="w-4 h-4" />
                      </button>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </div>

      {/* Detail Slide-over / Modal */}
      {selectedRule && (
        <div className="fixed inset-y-0 right-0 w-full max-w-2xl bg-slate-900 border-l border-slate-800 shadow-2xl z-50 flex flex-col overflow-hidden animate-in slide-in-from-right duration-200">
          {/* Drawer Header */}
          <div className="p-4 border-b border-slate-800 flex items-center justify-between bg-slate-950">
            <div>
              <div className="flex items-center gap-2">
                <h2 className="text-base font-bold text-slate-100">{selectedRule.rule_code}</h2>
                {getStatusBadge(selectedRule.library_status)}
                {getPhaseBadge(selectedRule.rule_phase)}
                {getSeverityBadge(selectedRule.severity)}
              </div>
              <p className="text-xs text-slate-400 mt-0.5">{selectedRule.name}</p>
            </div>
            <button
              onClick={() => setSelectedRule(null)}
              className="p-1.5 hover:bg-slate-800 rounded text-slate-400 hover:text-slate-200 transition-colors"
            >
              <X className="w-5 h-5" />
            </button>
          </div>

          {/* Drawer Body */}
          <div className="flex-1 overflow-auto p-4 space-y-5">
            {loadingDetails ? (
              <div className="flex items-center justify-center h-48 text-slate-500">
                <RefreshCw className="w-6 h-6 animate-spin mr-2 text-emerald-400" />
                Loading detailed AST and version history...
              </div>
            ) : (
              <>
                {/* Description & Legal Citation */}
                <div className="p-3 bg-slate-950 border border-slate-800 rounded space-y-2">
                  <div className="text-xs font-semibold text-slate-300 flex items-center gap-1.5">
                    <FileCode className="w-3.5 h-3.5 text-emerald-400" />
                    Regulatory Citation & Authority
                  </div>
                  <div className="text-xs text-emerald-300/90 font-mono bg-emerald-950/40 p-2 rounded border border-emerald-800/40">
                    {selectedRule.citation || 'Custom Institutional Rule / House Policy'}
                  </div>
                  {selectedRule.description && (
                    <p className="text-xs text-slate-400 pt-1 leading-relaxed">{selectedRule.description}</p>
                  )}
                </div>

                {/* Cryptographic Content Anchor */}
                <div className="p-3 bg-slate-950 border border-slate-800 rounded space-y-2">
                  <div className="text-xs font-semibold text-slate-300 flex items-center justify-between">
                    <span className="flex items-center gap-1.5">
                      <Shield className="w-3.5 h-3.5 text-cyan-400" />
                      RFC 8785 Canonical Content Digest (SHA-256)
                    </span>
                    <button
                      onClick={() => copyToClipboard(selectedRule.content_hash)}
                      className="flex items-center gap-1 text-[11px] text-slate-400 hover:text-cyan-400 transition-colors"
                    >
                      {copiedHash === selectedRule.content_hash ? (
                        <>
                          <Check className="w-3 h-3 text-emerald-400" />
                          Copied!
                        </>
                      ) : (
                        <>
                          <Copy className="w-3 h-3" />
                          Copy Hash
                        </>
                      )}
                    </button>
                  </div>
                  <div className="text-xs font-mono text-cyan-300/90 bg-cyan-950/30 p-2 rounded border border-cyan-800/40 break-all select-all">
                    {selectedRule.content_hash}
                  </div>
                </div>

                {/* Parameter Thresholds */}
                <div className="p-3 bg-slate-950 border border-slate-800 rounded space-y-2">
                  <div className="text-xs font-semibold text-slate-300 flex items-center gap-1.5">
                    <Sliders className="w-3.5 h-3.5 text-amber-400" />
                    Default Parameter Thresholds
                  </div>
                  {Object.keys(selectedRule.parameter_thresholds || {}).length === 0 ? (
                    <div className="text-xs text-slate-500 italic">No configurable parameter thresholds (Hard-coded AST predicate)</div>
                  ) : (
                    <div className="grid grid-cols-2 gap-2">
                      {Object.entries(selectedRule.parameter_thresholds).map(([param, val]) => (
                        <div key={param} className="p-2 bg-slate-900 border border-slate-800 rounded">
                          <div className="text-[11px] text-slate-400">{param}</div>
                          <div className="text-xs font-bold text-amber-300 font-mono mt-0.5">{val}</div>
                        </div>
                      ))}
                    </div>
                  )}
                </div>

                {/* AST Condition Definition */}
                <div className="p-3 bg-slate-950 border border-slate-800 rounded space-y-2">
                  <div className="text-xs font-semibold text-slate-300 flex items-center gap-1.5">
                    <Terminal className="w-3.5 h-3.5 text-indigo-400" />
                    Compiled Rule AST Condition
                  </div>
                  <pre className="p-3 bg-slate-900 border border-slate-800 rounded text-xs text-indigo-200 overflow-x-auto max-h-56 leading-relaxed font-mono">
                    {JSON.stringify(selectedRule.ast_condition, null, 2)}
                  </pre>
                </div>

                {/* Scenario Test Cases */}
                {selectedRule.sample_test_cases && selectedRule.sample_test_cases.length > 0 && (
                  <div className="p-3 bg-slate-950 border border-slate-800 rounded space-y-2">
                    <div className="text-xs font-semibold text-slate-300 flex items-center gap-1.5">
                      <Activity className="w-3.5 h-3.5 text-emerald-400" />
                      Deterministic Scenario Test Vectors ({selectedRule.sample_test_cases.length})
                    </div>
                    <div className="space-y-2">
                      {selectedRule.sample_test_cases.map((sc) => (
                        <div key={sc.Code} className="p-2.5 bg-slate-900 border border-slate-800 rounded text-xs space-y-1">
                          <div className="flex items-center justify-between">
                            <span className="font-bold text-slate-200">{sc.Code}</span>
                            <span
                              className={`px-1.5 py-0.5 rounded text-[10px] font-semibold ${
                                sc.Expected.Status === 'PASSED'
                                  ? 'bg-emerald-950 text-emerald-300 border border-emerald-800'
                                  : 'bg-red-950 text-red-300 border border-red-800'
                              }`}
                            >
                              Expected: {sc.Expected.Status}
                            </span>
                          </div>
                          <p className="text-slate-400 text-[11px]">{sc.Description}</p>
                          {sc.Expected.MustContain && (
                            <div className="text-[10px] text-slate-500 font-mono">
                              Assertion must contain: <code className="text-slate-300">"{sc.Expected.MustContain}"</code>
                            </div>
                          )}
                        </div>
                      ))}
                    </div>
                  </div>
                )}

                {/* Version History Timeline */}
                <div className="p-3 bg-slate-950 border border-slate-800 rounded space-y-2">
                  <div className="text-xs font-semibold text-slate-300 flex items-center gap-1.5">
                    <Clock className="w-3.5 h-3.5 text-slate-400" />
                    Snapshot Version Timeline
                  </div>
                  <div className="space-y-2">
                    {selectedRule.version_history?.map((vh) => (
                      <div
                        key={vh.version}
                        className={`p-2.5 bg-slate-900 border rounded text-xs space-y-1 ${
                          vh.version === selectedRule.current_version
                            ? 'border-emerald-700/80 bg-emerald-950/20'
                            : 'border-slate-800'
                        }`}
                      >
                        <div className="flex items-center justify-between">
                          <span className="font-bold text-slate-200 flex items-center gap-1.5">
                            Version {vh.version}
                            {vh.version === selectedRule.current_version && (
                              <span className="text-[10px] bg-emerald-950 text-emerald-300 border border-emerald-800 px-1 py-0.2 rounded">
                                CURRENT
                              </span>
                            )}
                          </span>
                          <span className="text-slate-500 text-[11px]">
                            {new Date(vh.effective_from).toLocaleDateString()}
                          </span>
                        </div>
                        <div className="text-[11px] text-slate-400">
                          Citation: <span className="text-slate-300">{vh.citation}</span>
                        </div>
                        <div className="text-[10px] text-slate-500 font-mono truncate">Hash: {vh.content_hash}</div>
                      </div>
                    ))}
                  </div>
                </div>
              </>
            )}
          </div>
        </div>
      )}
    </div>
  );
};
