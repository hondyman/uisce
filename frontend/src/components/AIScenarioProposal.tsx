import React, { useState, useEffect } from 'react';

interface AIProposedScenario {
  id: string;
  title: string;
  description: string;
  confidence: number;
  impact: 'High' | 'Medium' | 'Low';
  category: string;
  marketSnapshot?: string;
}

interface MarketData {
  sp500: number;
  sp500Change: number;
  vix: number;
  vixChange: number;
  treasuryYield: number;
  treasuryYieldChange: number;
}

const AIScenarioProposal: React.FC<{
  isOpen: boolean;
  onClose: () => void;
  onSelectScenario: (scenario: AIProposedScenario) => void;
}> = ({ isOpen, onClose, onSelectScenario }) => {
  const [scenarios, setScenarios] = useState<AIProposedScenario[]>([]);
  const [marketData, setMarketData] = useState<MarketData | null>(null);
  const [loading, setLoading] = useState(false);
  const [selectedScenario, setSelectedScenario] = useState<AIProposedScenario | null>(null);

  useEffect(() => {
    if (isOpen) {
      fetchAIScenarios();
    }
  }, [isOpen]);

  const fetchAIScenarios = async () => {
    setLoading(true);
    try {
      // Fetch from backend API
      const response = await fetch('/api/ai/scenario-proposals');
      const data = await response.json();
      setScenarios(data.scenarios || []);
      setMarketData(data.marketData || null);
    } catch (error) {
      console.error('Failed to fetch AI scenarios:', error);
      // Fallback mock data
      setScenarios(mockScenarios);
      setMarketData(mockMarketData);
    } finally {
      setLoading(false);
    }
  };

  const handleRefresh = () => {
    fetchAIScenarios();
  };

  if (!isOpen) return null;

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/60 backdrop-blur-sm p-4">
      <div className="w-full max-w-4xl bg-white dark:bg-slate-800 rounded-xl shadow-2xl flex flex-col max-h-[90vh] border border-slate-200 dark:border-slate-700">
        {/* Header */}
        <div className="flex items-center justify-between border-b border-slate-200 dark:border-slate-700 p-6 flex-shrink-0">
          <div>
            <h2 className="text-2xl font-bold text-slate-900 dark:text-white">AI Proposed Scenarios</h2>
            <p className="text-sm text-slate-600 dark:text-slate-400 mt-1">
              Leverage AI to identify and propose new scenarios based on current and historical market data
            </p>
          </div>
          <button
            onClick={onClose}
            title="Close"
            className="text-slate-500 dark:text-slate-400 hover:text-slate-700 dark:hover:text-slate-200"
          >
            <svg className="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
            </svg>
          </button>
        </div>

        {/* Market Snapshot */}
        {marketData && (
          <div className="p-6 border-b border-slate-200 dark:border-slate-700 bg-slate-50 dark:bg-slate-700/50">
            <h3 className="text-lg font-semibold text-slate-900 dark:text-white mb-4">AI-Powered Market Snapshot</h3>
            <p className="text-sm text-slate-600 dark:text-slate-400 mb-4">
              Based on current market sentiment and key economic drivers, our AI analysis indicates a cautious outlook.
              Increased volatility is expected in the tech sector due to recent regulatory announcements, while the
              energy sector shows strong upward momentum.
            </p>
            <div className="grid grid-cols-3 gap-4">
              <div className="bg-white dark:bg-slate-800 rounded-lg p-4 border border-slate-200 dark:border-slate-600">
                <p className="text-sm text-slate-600 dark:text-slate-400 mb-1">S&P 500</p>
                <p className="text-2xl font-bold text-slate-900 dark:text-white">{marketData.sp500.toLocaleString()}</p>
                <p className={`text-sm font-medium mt-1 ${marketData.sp500Change > 0 ? 'text-green-600' : 'text-red-600'}`}>
                  {marketData.sp500Change > 0 ? '+' : ''}{marketData.sp500Change.toFixed(2)}%
                </p>
              </div>
              <div className="bg-white dark:bg-slate-800 rounded-lg p-4 border border-slate-200 dark:border-slate-600">
                <p className="text-sm text-slate-600 dark:text-slate-400 mb-1">VIX</p>
                <p className="text-2xl font-bold text-slate-900 dark:text-white">{marketData.vix.toFixed(2)}</p>
                <p className={`text-sm font-medium mt-1 ${marketData.vixChange < 0 ? 'text-green-600' : 'text-red-600'}`}>
                  {marketData.vixChange > 0 ? '+' : ''}{marketData.vixChange.toFixed(2)}%
                </p>
              </div>
              <div className="bg-white dark:bg-slate-800 rounded-lg p-4 border border-slate-200 dark:border-slate-600">
                <p className="text-sm text-slate-600 dark:text-slate-400 mb-1">10-Yr Treasury Yield</p>
                <p className="text-2xl font-bold text-slate-900 dark:text-white">{marketData.treasuryYield.toFixed(2)}%</p>
                <p className={`text-sm font-medium mt-1 ${marketData.treasuryYieldChange > 0 ? 'text-red-600' : 'text-green-600'}`}>
                  {marketData.treasuryYieldChange > 0 ? '+' : ''}{marketData.treasuryYieldChange.toFixed(2)}%
                </p>
              </div>
            </div>
          </div>
        )}

        {/* Content */}
        <div className="flex-1 overflow-y-auto p-6">
          {loading ? (
            <div className="flex items-center justify-center h-64">
              <div className="text-center">
                <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-blue-600 mx-auto mb-4"></div>
                <p className="text-slate-600 dark:text-slate-400">Analyzing market data and generating scenarios...</p>
              </div>
            </div>
          ) : (
            <div className="space-y-4">
              {scenarios.map((scenario) => (
                <div
                  key={scenario.id}
                  className="p-4 rounded-lg border border-slate-200 dark:border-slate-600 bg-white dark:bg-slate-700 hover:border-blue-400 dark:hover:border-blue-500 transition-colors"
                >
                  <div className="flex justify-between items-start gap-4">
                    <div className="flex-1">
                      <h4 className="font-semibold text-slate-900 dark:text-white text-lg mb-2">{scenario.title}</h4>
                      <div className="flex gap-2 mb-3">
                        <span className="text-xs font-semibold px-2.5 py-1 bg-blue-100 dark:bg-blue-600/30 text-blue-700 dark:text-blue-300 rounded-full">
                          {scenario.category}
                        </span>
                        <span
                          className={`text-xs font-semibold px-2.5 py-1 rounded-full ${
                            scenario.impact === 'High'
                              ? 'bg-red-100 dark:bg-red-600/30 text-red-700 dark:text-red-300'
                              : scenario.impact === 'Medium'
                                ? 'bg-yellow-100 dark:bg-yellow-600/30 text-yellow-700 dark:text-yellow-300'
                                : 'bg-green-100 dark:bg-green-600/30 text-green-700 dark:text-green-300'
                          }`}
                        >
                          {scenario.impact} Impact
                        </span>
                      </div>
                      <p className="text-sm text-slate-600 dark:text-slate-300 leading-relaxed">
                        {scenario.description}
                      </p>
                    </div>
                    <div className="flex items-center gap-2 flex-shrink-0">
                      <div className="text-right">
                        <p className="text-xs text-slate-500 dark:text-slate-400">Confidence</p>
                        <p className="text-lg font-bold text-slate-900 dark:text-white">{scenario.confidence}%</p>
                      </div>
                      <div className="flex items-center gap-1">
                        <svg
                          className="w-5 h-5 text-blue-600"
                          fill="currentColor"
                          viewBox="0 0 20 20"
                        >
                          <path d="M2 11a1 1 0 011-1h2.101a7 7 0 01.05-3.452 1 1 0 01.894-.553h.994a1 1 0 01.894.553 7 7 0 01.05 3.452h2.101a1 1 0 011 1v5a2 2 0 01-2 2H4a2 2 0 01-2-2v-5z" />
                        </svg>
                      </div>
                    </div>
                  </div>
                  <div className="flex gap-3 mt-4 pt-4 border-t border-slate-200 dark:border-slate-600">
                    <button
                      onClick={() => onSelectScenario(scenario)}
                      className="flex-1 px-4 py-2 bg-blue-600 hover:bg-blue-700 text-white text-sm font-semibold rounded-lg transition-colors flex items-center justify-center gap-2"
                    >
                      <svg className="w-4 h-4" fill="currentColor" viewBox="0 0 20 20">
                        <path d="M5.5 13a3.5 3.5 0 01-.369-6.98 4 4 0 117.753-1 4.5 4.5 0 11-4.384 5.98z" />
                      </svg>
                      Run Analysis
                    </button>
                    <button
                      onClick={() => setSelectedScenario(scenario)}
                      className="flex-1 px-4 py-2 bg-slate-200 dark:bg-slate-600 hover:bg-slate-300 dark:hover:bg-slate-500 text-slate-900 dark:text-white text-sm font-semibold rounded-lg transition-colors flex items-center justify-center gap-2"
                    >
                      <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
                      </svg>
                      View Details
                    </button>
                  </div>
                </div>
              ))}
            </div>
          )}
        </div>

        {/* Footer */}
        <div className="flex-shrink-0 border-t border-slate-200 dark:border-slate-700 p-6 flex justify-between items-center bg-slate-50 dark:bg-slate-700/50">
          <button
            onClick={handleRefresh}
            className="px-4 py-2 flex items-center gap-2 bg-slate-200 dark:bg-slate-600 hover:bg-slate-300 dark:hover:bg-slate-500 text-slate-900 dark:text-white text-sm font-semibold rounded-lg transition-colors"
          >
            <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
            </svg>
            Refresh
          </button>
          <button
            onClick={onClose}
            className="px-4 py-2 text-slate-600 dark:text-slate-400 hover:text-slate-900 dark:hover:text-white text-sm font-semibold"
          >
            Cancel
          </button>
        </div>
      </div>

      {/* Details Modal */}
      {selectedScenario && (
        <ScenarioDetailsModal
          scenario={selectedScenario}
          onClose={() => setSelectedScenario(null)}
          onUse={() => {
            onSelectScenario(selectedScenario);
            setSelectedScenario(null);
            onClose();
          }}
        />
      )}
    </div>
  );
};

interface ScenarioDetailsModalProps {
  scenario: AIProposedScenario;
  onClose: () => void;
  onUse: () => void;
}

const ScenarioDetailsModal: React.FC<ScenarioDetailsModalProps> = ({ scenario, onClose, onUse }) => {
  return (
    <div className="fixed inset-0 z-60 flex items-center justify-center bg-black/60 backdrop-blur-sm p-4">
      <div className="w-full max-w-2xl bg-white dark:bg-slate-800 rounded-xl shadow-2xl overflow-hidden border border-slate-200 dark:border-slate-700">
        {/* Header */}
        <div className="flex items-center justify-between border-b border-slate-200 dark:border-slate-700 p-6">
          <h3 className="text-xl font-bold text-slate-900 dark:text-white">
            Scenario Details: {scenario.title}
          </h3>
          <button
            onClick={onClose}
            title="Close"
            className="text-slate-500 dark:text-slate-400 hover:text-slate-700 dark:hover:text-slate-200"
          >
            <svg className="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
            </svg>
          </button>
        </div>

        {/* Content */}
        <div className="overflow-y-auto p-6 max-h-[calc(90vh-200px)] space-y-8">
          {/* AI Rationale */}
          <div>
            <h4 className="text-lg font-bold text-slate-900 dark:text-white mb-3">AI Rationale</h4>
            <p className="text-sm text-slate-600 dark:text-slate-300 leading-relaxed mb-4">
              {scenario.description}
            </p>
            <div className="grid grid-cols-1 gap-3 border-t border-slate-200 dark:border-slate-600 pt-4">
              <div className="grid grid-cols-[150px_1fr] gap-4">
                <p className="text-sm text-slate-600 dark:text-slate-400">Key Driver 1</p>
                <p className="text-sm text-slate-900 dark:text-white">
                  Market momentum driven by positive economic indicators
                </p>
              </div>
              <div className="grid grid-cols-[150px_1fr] gap-4">
                <p className="text-sm text-slate-600 dark:text-slate-400">Key Driver 2</p>
                <p className="text-sm text-slate-900 dark:text-white">
                  Sector rotation towards high-growth technology stocks
                </p>
              </div>
              <div className="grid grid-cols-[150px_1fr] gap-4">
                <p className="text-sm text-slate-600 dark:text-slate-400">Key Driver 3</p>
                <p className="text-sm text-slate-900 dark:text-white">
                  Fed policy signaling lower interest rates in near term
                </p>
              </div>
            </div>
          </div>

          {/* Projected Impact */}
          <div>
            <h4 className="text-lg font-bold text-slate-900 dark:text-white mb-3">Projected Impact</h4>
            <div className="grid grid-cols-3 gap-4">
              <div className="flex items-start gap-3 rounded-lg bg-blue-50 dark:bg-blue-600/10 p-4 border border-blue-200 dark:border-blue-700">
                <div className="flex h-10 w-10 items-center justify-center rounded-full bg-blue-600/20 text-blue-600">
                  <svg className="w-5 h-5" fill="currentColor" viewBox="0 0 20 20">
                    <path d="M2 11a1 1 0 011-1h2.101a7 7 0 01.05-3.452 1 1 0 01.894-.553h.994a1 1 0 01.894.553 7 7 0 01.05 3.452h2.101a1 1 0 011 1v5a2 2 0 01-2 2H4a2 2 0 01-2-2v-5z" />
                  </svg>
                </div>
                <div>
                  <p className="text-xs text-slate-600 dark:text-slate-400">Projected Alpha</p>
                  <p className="text-xl font-bold text-slate-900 dark:text-white">+1.5%</p>
                </div>
              </div>
              <div className="flex items-start gap-3 rounded-lg bg-blue-50 dark:bg-blue-600/10 p-4 border border-blue-200 dark:border-blue-700">
                <div className="flex h-10 w-10 items-center justify-center rounded-full bg-blue-600/20 text-blue-600">
                  <svg className="w-5 h-5" fill="currentColor" viewBox="0 0 20 20">
                    <path d="M3 4a1 1 0 011-1h12a1 1 0 011 1v2a1 1 0 01-1 1H4a1 1 0 01-1-1V4zm0 4a1 1 0 011-1h12a1 1 0 011 1v2a1 1 0 01-1 1H4a1 1 0 01-1-1V8zm0 4a1 1 0 011-1h12a1 1 0 011 1v2a1 1 0 01-1 1H4a1 1 0 01-1-1v-2z" />
                  </svg>
                </div>
                <div>
                  <p className="text-xs text-slate-600 dark:text-slate-400">Risk Profile</p>
                  <p className="text-xl font-bold text-slate-900 dark:text-white">Moderate</p>
                </div>
              </div>
              <div className="flex items-start gap-3 rounded-lg bg-blue-50 dark:bg-blue-600/10 p-4 border border-blue-200 dark:border-blue-700">
                <div className="flex h-10 w-10 items-center justify-center rounded-full bg-blue-600/20 text-blue-600">
                  <svg className="w-5 h-5" fill="currentColor" viewBox="0 0 20 20">
                    <path d="M2 11a1 1 0 011-1h2.101a7 7 0 01.05-3.452 1 1 0 01.894-.553h.994a1 1 0 01.894.553 7 7 0 01.05 3.452h2.101a1 1 0 011 1v5a2 2 0 01-2 2H4a2 2 0 01-2-2v-5z" />
                  </svg>
                </div>
                <div>
                  <p className="text-xs text-slate-600 dark:text-slate-400">Key Sectors</p>
                  <p className="text-xl font-bold text-slate-900 dark:text-white">Tech, Finance</p>
                </div>
              </div>
            </div>
          </div>
        </div>

        {/* Footer */}
        <div className="flex-shrink-0 border-t border-slate-200 dark:border-slate-700 p-6 flex justify-end gap-3 bg-slate-50 dark:bg-slate-700/50">
          <button
            onClick={onClose}
            className="px-4 py-2 bg-slate-200 dark:bg-slate-600 hover:bg-slate-300 dark:hover:bg-slate-500 text-slate-900 dark:text-white text-sm font-semibold rounded-lg transition-colors"
          >
            Close
          </button>
          <button
            onClick={onUse}
            className="px-4 py-2 bg-blue-600 hover:bg-blue-700 text-white text-sm font-semibold rounded-lg transition-colors"
          >
            Use this Scenario
          </button>
        </div>
      </div>
    </div>
  );
};

// Mock data for testing
const mockScenarios: AIProposedScenario[] = [
  {
    id: '1',
    title: 'Impending Interest Rate Hike',
    description: 'Anticipated central bank action to curb inflation could slow economic growth and negatively impact equity valuations.',
    confidence: 92,
    impact: 'High',
    category: 'Macro',
  },
  {
    id: '2',
    title: 'Geopolitical Tensions in EMEA',
    description: 'Escalating conflicts may disrupt supply chains and impact oil prices, leading to increased market volatility.',
    confidence: 78,
    impact: 'Medium',
    category: 'Geopolitical',
  },
  {
    id: '3',
    title: 'Consumer Spending Slowdown',
    description: 'Analysis of retail sales data suggests a potential contraction in discretionary spending over the next 6 months.',
    confidence: 65,
    impact: 'Low',
    category: 'Economic',
  },
];

const mockMarketData: MarketData = {
  sp500: 4510.5,
  sp500Change: 0.5,
  vix: 15.8,
  vixChange: -1.2,
  treasuryYield: 4.25,
  treasuryYieldChange: 0.02,
};

export default AIScenarioProposal;
