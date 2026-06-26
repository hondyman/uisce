import React, { useState } from 'react';
import { gql, useSubscription } from '@apollo/client';
import './ScenarioAnalysisPro.css';

const PORTFOLIOS_SUBSCRIPTION = gql`
  subscription {
    portfolios {
      id
      aum
      sharpe
      risk
      status
      assetAllocation {
        asset
        percentage
      }
    }
  }
`;

interface Portfolio {
  id: string;
  aum: number;
  sharpe: number;
  risk: number;
  status: string;
  assetAllocation: Array<{ asset: string; percentage: number }>;
}

interface AnalysisResult {
  baseCase: {
    aum: number;
    sharpe: number;
    risk: number;
    status: string;
    assetAllocation: Array<{ asset: string; percentage: number }>;
  };
  scenarioCase: {
    aum: number;
    aumChange: number;
    sharpe: number;
    sharpeChange: number;
    risk: number;
    riskChange: number;
    status: string;
    assetAllocation: Array<{ asset: string; percentage: number }>;
  };
  comparison: {
    aumDifference: number;
    sharpeDifference: number;
    riskDifference: number;
  };
}

interface AnalysisHistoryItem {
  scenario: string;
  date: string;
  result: AnalysisResult;
}

const ScenarioAnalysisPro: React.FC = () => {
  const { data: portfolioData } = useSubscription(PORTFOLIOS_SUBSCRIPTION);
  const [selectedPortfolio, setSelectedPortfolio] = useState<string>('');
  const [selectedScenario, setSelectedScenario] = useState<string>('');
  const [analysisResult, setAnalysisResult] = useState<AnalysisResult | null>(null);
  const [loadingAnalysis, setLoadingAnalysis] = useState(false);
  const [analysisHistory, setAnalysisHistory] = useState<AnalysisHistoryItem[]>([]);

  const scenarios = [
    'Market Crash (-20%)',
    'Interest Rate Hike (+2%)',
    'High Inflation (+5%)',
    'Tech Bubble Burst (-30% on tech stocks)',
    'Geopolitical Crisis',
  ];

  const handleRunAnalysis = async () => {
    if (!selectedPortfolio || !selectedScenario) return;
    setLoadingAnalysis(true);
    try {
      const resp = await fetch(`/api/portfolio/${selectedPortfolio}/scenario`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ scenario: selectedScenario }),
      });
      if (!resp.ok) throw new Error('Failed');
      const result = await resp.json();
      setAnalysisResult(result);
      setAnalysisHistory((h) => [
        { scenario: selectedScenario, date: new Date().toLocaleString(), result },
        ...h,
      ]);
    } catch (e) {
      console.error(e);
    } finally {
      setLoadingAnalysis(false);
    }
  };

  const renderGauge = (value: number, max = 100, color = '#00875A') => {
    const percentage = Math.max(0, Math.min(100, (value / max) * 100));
    const radius = 45;
    const circumference = 2 * Math.PI * radius;
    const offset = circumference - (circumference * percentage) / 100;

    return (
      <svg viewBox="0 0 100 100" width={96} height={96}>
        <circle cx={50} cy={50} r={radius} fill="none" stroke="#eee" strokeWidth={10} />
        <circle
          cx={50}
          cy={50}
          r={radius}
          fill="none"
          stroke={color}
          strokeWidth={10}
          strokeDasharray={circumference}
          strokeDashoffset={offset}
          strokeLinecap="round"
          className="gauge-circle"
        />
      </svg>
    );
  };

  const ar = analysisResult as AnalysisResult;

  return (
    <div className="flex h-screen bg-gradient-to-br from-slate-50 to-slate-100 dark:from-slate-900 dark:to-slate-800">
      <div className="w-1/3 bg-white dark:bg-slate-800 border-r border-slate-200 dark:border-slate-700 overflow-y-auto">
        <div className="p-8">
          <h1 className="text-3xl font-bold text-slate-900 dark:text-white mb-2">Scenario Analysis</h1>
          <p className="text-slate-500 dark:text-slate-400 text-sm mb-8">Analyze portfolio performance under various market conditions</p>

          <div className="bg-slate-50 dark:bg-slate-700 rounded-xl p-6 mb-6 border border-slate-200 dark:border-slate-600">
            <div className="mb-6">
              <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">Select Portfolio</label>
              <select
                title="Select a portfolio"
                value={selectedPortfolio}
                onChange={(e) => setSelectedPortfolio(e.target.value)}
                className="w-full px-4 py-3 rounded-lg border"
              >
                <option value="">Select a portfolio</option>
                {portfolioData?.portfolios?.map((p: Portfolio) => (
                  <option key={p.id} value={p.id}>
                    {p.id} | ${(p.aum / 1e6).toFixed(1)}M AUM
                  </option>
                ))}
              </select>
            </div>

            <div className="mb-6">
              <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">Select Scenario</label>
              <select title="Select a scenario" value={selectedScenario} onChange={(e) => setSelectedScenario(e.target.value)} className="w-full px-4 py-3 rounded-lg border">
                <option value="">Select a scenario</option>
                {scenarios.map((s) => (
                  <option key={s} value={s}>
                    {s}
                  </option>
                ))}
                <option disabled>───────────────────</option>
                <option value="custom">Create Custom Scenario...</option>
                <option value="ai">AI Scenario Proposal...</option>
              </select>
            </div>

            <button
              onClick={handleRunAnalysis}
              disabled={!selectedPortfolio || !selectedScenario || loadingAnalysis}
              className="w-full py-3 bg-blue-600 hover:bg-blue-700 disabled:bg-slate-300 text-white font-semibold rounded-lg transition-colors"
            >
              {loadingAnalysis ? 'Running Analysis...' : 'Run Analysis'}
            </button>
          </div>

          <div className="bg-slate-50 dark:bg-slate-700 rounded-xl p-6 border">
            <h3 className="text-lg font-semibold text-slate-900 dark:text-white mb-4">Analysis History</h3>
            <div className="space-y-2 max-h-64 overflow-y-auto">
              {analysisHistory.length === 0 ? (
                <p className="text-sm text-slate-500 dark:text-slate-400">No analyses yet</p>
              ) : (
                analysisHistory.map((item, idx) => (
                  <button
                    key={idx}
                    onClick={() => {
                      setAnalysisResult(item.result);
                      setSelectedScenario(item.scenario);
                    }}
                    className={`w-full text-left p-3 rounded-lg transition-colors ${analysisResult === item.result ? 'bg-blue-100' : 'hover:bg-slate-100'}`}
                  >
                    <p className="font-medium text-slate-900 dark:text-white text-sm">{item.scenario}</p>
                    <p className="text-xs text-slate-500 dark:text-slate-400">{item.date}</p>
                  </button>
                ))
              )}
            </div>
          </div>
        </div>
      </div>

      <div className="w-2/3 overflow-y-auto p-8">
        <h2 className="text-2xl font-bold text-slate-900 dark:text-white mb-2">
          Analysis Results: <span className="text-blue-600">{selectedScenario || 'Select a scenario'}</span>
        </h2>

        {loadingAnalysis && (
          <div className="flex items-center justify-center h-96">
            <div className="text-center">
              <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-blue-600 mx-auto mb-4"></div>
              <p className="text-slate-600 dark:text-slate-300">Running analysis...</p>
            </div>
          </div>
        )}

        {!loadingAnalysis && analysisResult && (
          <div className="space-y-6">
            <div className="grid grid-cols-2 gap-6">
              <div className="bg-white dark:bg-slate-800 rounded-xl p-6 border shadow-sm">
                <h3 className="text-xl font-semibold">Base Case</h3>
                <p className="text-3xl font-bold mt-3">${(ar.baseCase.aum / 1e6).toFixed(2)}M</p>
                <div className="flex gap-6 mt-4">
                  <div className="flex flex-col items-center">{renderGauge(ar.baseCase.sharpe, 3)}<div className="mt-2 text-sm font-bold">{ar.baseCase.sharpe.toFixed(2)}</div></div>
                  <div className="flex flex-col items-center">{renderGauge(ar.baseCase.risk, 100, '#FFAB00')}<div className="mt-2 text-sm font-bold">{ar.baseCase.risk.toFixed(0)}</div></div>
                </div>
              </div>

              <div className="bg-white dark:bg-slate-800 rounded-xl p-6 border shadow-sm">
                <h3 className="text-xl font-semibold">Scenario Case</h3>
                <p className="text-3xl font-bold mt-3">${(ar.scenarioCase.aum / 1e6).toFixed(2)}M</p>
                <div className="flex gap-6 mt-4">
                  <div className="flex flex-col items-center">{renderGauge(ar.scenarioCase.sharpe, 3, '#DE350B')}<div className="mt-2 text-sm font-bold">{ar.scenarioCase.sharpe.toFixed(2)}</div></div>
                  <div className="flex flex-col items-center">{renderGauge(ar.scenarioCase.risk, 100, '#DE350B')}<div className="mt-2 text-sm font-bold">{ar.scenarioCase.risk.toFixed(0)}</div></div>
                </div>
              </div>
            </div>

            <div className="bg-white dark:bg-slate-800 rounded-xl p-6 border shadow-sm">
              <h3 className="text-xl font-semibold">Comparison</h3>
              <div className="grid grid-cols-3 gap-6 mt-4">
                <div className="text-center">
                  <div className="text-2xl font-bold">${Math.abs((ar.comparison.aumDifference ?? 0) / 1e6).toFixed(2)}M</div>
                  <div className={`text-sm ${ (ar.comparison.aumDifference ?? 0) < 0 ? 'text-red-600' : 'text-green-600'}`}>{(ar.comparison.aumDifference ?? 0) < 0 ? '↓' : '↑'} {Math.abs(((ar.comparison.aumDifference ?? 0) / (ar.baseCase.aum ?? 1)) * 100).toFixed(2)}%</div>
                </div>
                <div className="text-center">
                  <div className="text-2xl font-bold">{ar.comparison.sharpeDifference.toFixed(2)}</div>
                  <div className={`text-sm ${ar.comparison.sharpeDifference < 0 ? 'text-red-600' : 'text-green-600'}`}>{ar.comparison.sharpeDifference > 0 ? '+' : ''}{ar.comparison.sharpeDifference.toFixed(2)}</div>
                </div>
                <div className="text-center">
                  <div className="text-2xl font-bold">{ar.comparison.riskDifference.toFixed(0)}</div>
                  <div className={`text-sm ${ar.comparison.riskDifference > 0 ? 'text-red-600' : 'text-green-600'}`}>{ar.comparison.riskDifference > 0 ? '+' : ''}{ar.comparison.riskDifference.toFixed(0)}</div>
                </div>
              </div>
            </div>
          </div>
        )}

        {!loadingAnalysis && !analysisResult && (
          <div className="flex items-center justify-center h-96">
            <p className="text-slate-500 dark:text-slate-400 text-lg">Select a portfolio and scenario, then click "Run Analysis" to begin</p>
          </div>
        )}
      </div>
    </div>
  );
};

export default ScenarioAnalysisPro;


