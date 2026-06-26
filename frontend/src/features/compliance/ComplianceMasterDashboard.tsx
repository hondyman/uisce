import React, { useState } from 'react';
import { Shield, ShieldAlert, FileSearch, Filter, Play, CheckCircle2, XCircle } from 'lucide-react';

const ComplianceMasterDashboard: React.FC = () => {
  const [activeTab, setActiveTab] = useState<'rules' | 'evaluations' | 'breaches'>('rules');

  const navItems = [
    { id: 'rules', label: 'Compliance Rules', icon: Shield },
    { id: 'evaluations', label: 'Evaluations', icon: FileSearch },
    { id: 'breaches', label: 'Breaches & Exceptions', icon: ShieldAlert },
  ] as const;

  return (
    <div className="flex flex-col h-full bg-slate-900 text-slate-200 p-6 overflow-hidden">
      <div className="flex justify-between items-center mb-6">
        <div>
          <h1 className="text-2xl font-bold text-white flex items-center gap-2">
            <Shield className="w-6 h-6 text-indigo-400" />
            Compliance Master
          </h1>
          <p className="text-sm text-slate-400 mt-1">
            Manage portfolio constraints, run evaluations, and handle breaches
          </p>
        </div>
        <div className="flex gap-3">
          <button className="flex items-center gap-2 px-4 py-2 bg-indigo-600 hover:bg-indigo-700 text-white rounded-md text-sm font-medium transition-colors">
            <Play className="w-4 h-4" />
            Run Sandbox Evaluation
          </button>
        </div>
      </div>

      <div className="flex bg-slate-800/50 p-1 rounded-lg border border-slate-700 w-fit mb-6">
        {navItems.map(({ id, label, icon: Icon }) => (
          <button
            key={id}
            onClick={() => setActiveTab(id)}
            className={`flex items-center gap-2 px-6 py-2.5 rounded-md text-sm font-medium transition-all ${
              activeTab === id
                ? 'bg-slate-700 text-white shadow-sm'
                : 'text-slate-400 hover:text-slate-200 hover:bg-slate-700/50'
            }`}
          >
            <Icon className="w-4 h-4" />
            {label}
          </button>
        ))}
      </div>

      <div className="flex-1 bg-slate-800 rounded-lg border border-slate-700 overflow-hidden flex flex-col">
        {/* Placeholder mock data panels */}
        {activeTab === 'rules' && <RulesPanel />}
        {activeTab === 'evaluations' && <EvaluationsPanel />}
        {activeTab === 'breaches' && <BreachesPanel />}
      </div>
    </div>
  );
};

const RulesPanel = () => (
  <div className="p-6">
    <div className="flex items-center justify-between mb-4">
      <h2 className="text-lg font-semibold text-white">Active Rule Library</h2>
      <button className="flex items-center gap-2 px-3 py-1.5 bg-slate-700 hover:bg-slate-600 rounded text-sm text-white">
        <Filter className="w-4 h-4" /> Filter
      </button>
    </div>
    <div className="overflow-x-auto border border-slate-700 rounded-lg bg-slate-900">
      <table className="w-full text-left text-sm">
        <thead className="bg-slate-800 text-slate-400">
          <tr>
            <th className="px-4 py-3 font-medium">Rule Code</th>
            <th className="px-4 py-3 font-medium">Name</th>
            <th className="px-4 py-3 font-medium">Scope</th>
            <th className="px-4 py-3 font-medium">Severity</th>
            <th className="px-4 py-3 font-medium">Status</th>
          </tr>
        </thead>
        <tbody className="divide-y divide-slate-700/50">
          <tr className="hover:bg-slate-800/50">
            <td className="px-4 py-3 text-indigo-400 font-mono">EQ_CON_01</td>
            <td className="px-4 py-3">Max Equity Concentration 20%</td>
            <td className="px-4 py-3">PORTFOLIO</td>
            <td className="px-4 py-3"><span className="px-2 py-1 bg-red-500/10 text-red-400 rounded-full text-xs">HARD</span></td>
            <td className="px-4 py-3"><span className="text-emerald-400 text-xs font-semibold">ACTIVE</span></td>
          </tr>
          {/* Mock rows */}
        </tbody>
      </table>
    </div>
  </div>
);

const EvaluationsPanel = () => (
  <div className="p-6">
    <div className="flex items-center justify-between mb-4">
      <h2 className="text-lg font-semibold text-white">Recent Evaluations</h2>
    </div>
    <div className="overflow-x-auto border border-slate-700 rounded-lg bg-slate-900">
      <table className="w-full text-left text-sm">
        <thead className="bg-slate-800 text-slate-400">
          <tr>
            <th className="px-4 py-3 font-medium">Date</th>
            <th className="px-4 py-3 font-medium">Portfolio</th>
            <th className="px-4 py-3 font-medium">Rule Evaluated</th>
            <th className="px-4 py-3 font-medium">Result</th>
          </tr>
        </thead>
        <tbody className="divide-y divide-slate-700/50">
          <tr className="hover:bg-slate-800/50">
            <td className="px-4 py-3">2026-02-22</td>
            <td className="px-4 py-3 text-emerald-400">Global Equity Fund</td>
            <td className="px-4 py-3 font-mono text-indigo-400">EQ_CON_01</td>
            <td className="px-4 py-3">
              <span className="flex items-center gap-1 text-emerald-400">
                <CheckCircle2 className="w-4 h-4" /> PASS
              </span>
            </td>
          </tr>
        </tbody>
      </table>
    </div>
  </div>
);

const BreachesPanel = () => (
  <div className="p-6">
    <div className="flex items-center justify-between mb-4">
      <h2 className="text-lg font-semibold text-white">Active Breaches</h2>
    </div>
    <div className="overflow-x-auto border border-slate-700 rounded-lg bg-slate-900">
      <table className="w-full text-left text-sm">
        <thead className="bg-slate-800 text-slate-400">
          <tr>
            <th className="px-4 py-3 font-medium">Portfolio</th>
            <th className="px-4 py-3 font-medium">Rule</th>
            <th className="px-4 py-3 font-medium">Deviation</th>
            <th className="px-4 py-3 font-medium">Status</th>
            <th className="px-4 py-3 font-medium">Action</th>
          </tr>
        </thead>
        <tbody className="divide-y divide-slate-700/50">
          <tr className="hover:bg-slate-800/50">
            <td className="px-4 py-3 text-emerald-400">US Growth Fund</td>
            <td className="px-4 py-3 font-mono text-indigo-400">EQ_CON_01</td>
            <td className="px-4 py-3 text-red-400">+2.5% over limit</td>
            <td className="px-4 py-3"><span className="px-2 py-1 bg-amber-500/10 text-amber-400 rounded-full text-xs">OPEN</span></td>
            <td className="px-4 py-3">
              <button className="text-indigo-400 hover:text-indigo-300 text-xs font-semibold">Resolve</button>
            </td>
          </tr>
        </tbody>
      </table>
    </div>
  </div>
);

export default ComplianceMasterDashboard;
