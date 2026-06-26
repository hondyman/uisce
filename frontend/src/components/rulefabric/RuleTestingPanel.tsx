/**
 * RuleTestingPanel.tsx
 * 
 * Comprehensive rule testing and simulation component providing:
 * - Live preview with sample data
 * - Historical impact analysis
 * - Batch simulation across datasets
 * - Performance benchmarking
 * - Debug mode with step-by-step evaluation
 */

import React, { useState, useCallback, useEffect } from 'react';
import {
  Play,
  RotateCcw,
  Database,
  FileJson,
  Upload,
  Download,
  Clock,
  Check,
  X,
  AlertTriangle,
  Info,
  ChevronRight,
  ChevronDown,
  Zap,
  History,
  BarChart2,
  Bug,
  Layers,
  Target,
  Filter,
  Settings,
  RefreshCw
} from 'lucide-react';

// Types
interface RuleDefinition {
  id?: string;
  name: string;
  category: string;
  description?: string;
  conditions: ConditionGroup;
  actions: RuleAction[];
  cel_expression?: string;
}

interface ConditionGroup {
  operator: 'AND' | 'OR';
  conditions: (Condition | ConditionGroup)[];
}

interface Condition {
  field: string;
  operator: string;
  value: unknown;
  entity?: string;
}

interface RuleAction {
  type: string;
  config: Record<string, unknown>;
}

interface TestRecord {
  id: string;
  data: Record<string, unknown>;
  source?: string;
}

interface EvaluationResult {
  recordId: string;
  matched: boolean;
  conditionResults: ConditionEvalResult[];
  executionTimeMs: number;
  triggeredActions: string[];
  debugTrace?: DebugStep[];
}

interface ConditionEvalResult {
  condition: string;
  result: boolean;
  actualValue: unknown;
  expectedValue: unknown;
}

interface DebugStep {
  step: number;
  expression: string;
  result: unknown;
  timeMs: number;
}

interface SimulationSummary {
  totalRecords: number;
  matchedRecords: number;
  matchRate: number;
  avgExecutionTimeMs: number;
  minExecutionTimeMs: number;
  maxExecutionTimeMs: number;
  p95ExecutionTimeMs: number;
  actionCounts: Record<string, number>;
  errorCount: number;
}

interface HistoricalImpact {
  period: string;
  recordsAnalyzed: number;
  wouldMatch: number;
  matchRate: number;
  topMatchingEntities: { id: string; name: string; matchCount: number }[];
}

interface RuleTestingPanelProps {
  rule: RuleDefinition;
  tenantId: string;
  datasourceId: string;
  onClose?: () => void;
}

// Sample data generator
const generateSampleData = (count: number): TestRecord[] => {
  const names = ['Acme Corp', 'Globex Inc', 'Initech', 'Umbrella Corp', 'Wayne Enterprises'];
  const statuses = ['active', 'pending', 'inactive', 'suspended'];
  const categories = ['enterprise', 'mid-market', 'smb', 'startup'];
  
  return Array.from({ length: count }, (_, i) => ({
    id: `sample-${i + 1}`,
    data: {
      id: `entity-${1000 + i}`,
      name: names[i % names.length],
      status: statuses[i % statuses.length],
      category: categories[i % categories.length],
      amount: Math.round(Math.random() * 100000),
      created_at: new Date(Date.now() - Math.random() * 365 * 24 * 60 * 60 * 1000).toISOString(),
      email: i % 3 === 0 ? null : `contact${i}@example.com`,
      risk_score: Math.round(Math.random() * 100),
      transaction_count: Math.floor(Math.random() * 500)
    },
    source: 'generated'
  }));
};

// Simulated evaluation (would call backend API in production)
const evaluateRule = async (
  _rule: RuleDefinition,
  records: TestRecord[],
  debugMode: boolean
): Promise<EvaluationResult[]> => {
  await new Promise(resolve => setTimeout(resolve, 500 + Math.random() * 500));
  
  return records.map(record => {
    // Simulate evaluation based on rule conditions
    const matched = Math.random() > 0.6;
    const executionTime = 0.5 + Math.random() * 2;
    
    const result: EvaluationResult = {
      recordId: record.id,
      matched,
      conditionResults: [
        {
          condition: 'amount > 10000',
          result: (record.data.amount as number) > 10000,
          actualValue: record.data.amount,
          expectedValue: 10000
        },
        {
          condition: 'status == "active"',
          result: record.data.status === 'active',
          actualValue: record.data.status,
          expectedValue: 'active'
        }
      ],
      executionTimeMs: executionTime,
      triggeredActions: matched ? ['flag', 'notify'] : []
    };
    
    if (debugMode) {
      result.debugTrace = [
        { step: 1, expression: 'record.amount', result: record.data.amount, timeMs: 0.1 },
        { step: 2, expression: 'amount > 10000', result: (record.data.amount as number) > 10000, timeMs: 0.2 },
        { step: 3, expression: 'record.status', result: record.data.status, timeMs: 0.1 },
        { step: 4, expression: 'status == "active"', result: record.data.status === 'active', timeMs: 0.15 }
      ];
    }
    
    return result;
  });
};

// Historical analysis (simulated)
const analyzeHistoricalImpact = async (
  _rule: RuleDefinition,
  _tenantId: string,
  _datasourceId: string
): Promise<HistoricalImpact[]> => {
  await new Promise(resolve => setTimeout(resolve, 1000));
  
  return [
    {
      period: 'Last 7 days',
      recordsAnalyzed: 15420,
      wouldMatch: 2341,
      matchRate: 15.2,
      topMatchingEntities: [
        { id: 'e1', name: 'Acme Corp', matchCount: 234 },
        { id: 'e2', name: 'Globex Inc', matchCount: 189 },
        { id: 'e3', name: 'Initech', matchCount: 156 }
      ]
    },
    {
      period: 'Last 30 days',
      recordsAnalyzed: 64230,
      wouldMatch: 9876,
      matchRate: 15.4,
      topMatchingEntities: [
        { id: 'e1', name: 'Acme Corp', matchCount: 1023 },
        { id: 'e2', name: 'Globex Inc', matchCount: 876 },
        { id: 'e4', name: 'Wayne Enterprises', matchCount: 654 }
      ]
    },
    {
      period: 'Last 90 days',
      recordsAnalyzed: 198450,
      wouldMatch: 31245,
      matchRate: 15.7,
      topMatchingEntities: [
        { id: 'e1', name: 'Acme Corp', matchCount: 3245 },
        { id: 'e2', name: 'Globex Inc', matchCount: 2876 },
        { id: 'e5', name: 'Umbrella Corp', matchCount: 2341 }
      ]
    }
  ];
};

export const RuleTestingPanel: React.FC<RuleTestingPanelProps> = ({
  rule,
  tenantId,
  datasourceId,
  onClose
}) => {
  const [activeTab, setActiveTab] = useState<'live' | 'batch' | 'historical' | 'performance'>('live');
  const [testRecords, setTestRecords] = useState<TestRecord[]>([]);
  const [results, setResults] = useState<EvaluationResult[]>([]);
  const [isRunning, setIsRunning] = useState(false);
  const [debugMode, setDebugMode] = useState(false);
  const [selectedRecord, setSelectedRecord] = useState<string | null>(null);
  const [historicalData, setHistoricalData] = useState<HistoricalImpact[]>([]);
  const [isLoadingHistory, setIsLoadingHistory] = useState(false);
  const [customJson, setCustomJson] = useState('');
  const [jsonError, setJsonError] = useState<string | null>(null);
  const [showSettings, setShowSettings] = useState(false);
  const [settings, setSettings] = useState({
    sampleSize: 10,
    timeout: 5000,
    parallelExecution: true
  });

  // Generate sample data on mount
  useEffect(() => {
    setTestRecords(generateSampleData(settings.sampleSize));
  }, [settings.sampleSize]);

  // Load historical analysis
  useEffect(() => {
    if (activeTab === 'historical' && historicalData.length === 0) {
      loadHistoricalAnalysis();
    }
  }, [activeTab, historicalData.length]);

  const loadHistoricalAnalysis = async () => {
    setIsLoadingHistory(true);
    try {
      const data = await analyzeHistoricalImpact(rule, tenantId, datasourceId);
      setHistoricalData(data);
    } finally {
      setIsLoadingHistory(false);
    }
  };

  const runEvaluation = useCallback(async () => {
    if (testRecords.length === 0) return;
    
    setIsRunning(true);
    setResults([]);
    
    try {
      const evalResults = await evaluateRule(rule, testRecords, debugMode);
      setResults(evalResults);
    } finally {
      setIsRunning(false);
    }
  }, [rule, testRecords, debugMode]);

  const handleJsonImport = useCallback(() => {
    try {
      const parsed = JSON.parse(customJson);
      const records: TestRecord[] = Array.isArray(parsed) 
        ? parsed.map((item, i) => ({ id: `import-${i}`, data: item, source: 'import' }))
        : [{ id: 'import-0', data: parsed, source: 'import' }];
      setTestRecords(records);
      setJsonError(null);
      setCustomJson('');
    } catch (e) {
      setJsonError('Invalid JSON format');
    }
  }, [customJson]);

  const exportResults = useCallback(() => {
    const exportData = {
      rule: rule.name,
      timestamp: new Date().toISOString(),
      summary: calculateSummary(results),
      results: results
    };
    
    const blob = new Blob([JSON.stringify(exportData, null, 2)], { type: 'application/json' });
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = `rule-test-${rule.name}-${Date.now()}.json`;
    a.click();
    URL.revokeObjectURL(url);
  }, [rule, results]);

  const calculateSummary = (evalResults: EvaluationResult[]): SimulationSummary | null => {
    if (evalResults.length === 0) return null;
    
    const matched = evalResults.filter(r => r.matched);
    const times = evalResults.map(r => r.executionTimeMs).sort((a, b) => a - b);
    const actionCounts: Record<string, number> = {};
    
    evalResults.forEach(r => {
      r.triggeredActions.forEach(action => {
        actionCounts[action] = (actionCounts[action] || 0) + 1;
      });
    });
    
    return {
      totalRecords: evalResults.length,
      matchedRecords: matched.length,
      matchRate: (matched.length / evalResults.length) * 100,
      avgExecutionTimeMs: times.reduce((a, b) => a + b, 0) / times.length,
      minExecutionTimeMs: times[0],
      maxExecutionTimeMs: times[times.length - 1],
      p95ExecutionTimeMs: times[Math.floor(times.length * 0.95)],
      actionCounts,
      errorCount: 0
    };
  };

  const summary = calculateSummary(results);

  const renderSummaryCards = () => {
    if (!summary) return null;
    
    return (
      <div className="grid grid-cols-2 md:grid-cols-4 gap-3 mb-4">
        <div className="bg-white rounded-lg border p-3">
          <div className="flex items-center gap-2 text-gray-500 text-xs mb-1">
            <Database size={14} />
            Records Tested
          </div>
          <div className="text-2xl font-semibold">{summary.totalRecords}</div>
        </div>
        <div className="bg-white rounded-lg border p-3">
          <div className="flex items-center gap-2 text-gray-500 text-xs mb-1">
            <Target size={14} />
            Match Rate
          </div>
          <div className="text-2xl font-semibold">{summary.matchRate.toFixed(1)}%</div>
          <div className="text-xs text-gray-500">
            {summary.matchedRecords} of {summary.totalRecords}
          </div>
        </div>
        <div className="bg-white rounded-lg border p-3">
          <div className="flex items-center gap-2 text-gray-500 text-xs mb-1">
            <Clock size={14} />
            Avg Time
          </div>
          <div className="text-2xl font-semibold">{summary.avgExecutionTimeMs.toFixed(2)}ms</div>
          <div className="text-xs text-gray-500">
            P95: {summary.p95ExecutionTimeMs.toFixed(2)}ms
          </div>
        </div>
        <div className="bg-white rounded-lg border p-3">
          <div className="flex items-center gap-2 text-gray-500 text-xs mb-1">
            <Zap size={14} />
            Actions Triggered
          </div>
          <div className="text-2xl font-semibold">
            {Object.values(summary.actionCounts).reduce((a, b) => a + b, 0)}
          </div>
          <div className="text-xs text-gray-500">
            {Object.keys(summary.actionCounts).length} types
          </div>
        </div>
      </div>
    );
  };

  const renderResultRow = (result: EvaluationResult, record: TestRecord) => {
    const isSelected = selectedRecord === result.recordId;
    
    return (
      <div key={result.recordId} className="border rounded-lg mb-2 overflow-hidden">
        <div
          className={`flex items-center justify-between p-3 cursor-pointer transition-colors ${
            isSelected ? 'bg-gray-100' : 'bg-white hover:bg-gray-50'
          }`}
          onClick={() => setSelectedRecord(isSelected ? null : result.recordId)}
          role="button"
          tabIndex={0}
          onKeyDown={(e) => e.key === 'Enter' && setSelectedRecord(isSelected ? null : result.recordId)}
        >
          <div className="flex items-center gap-3">
            {result.matched ? (
              <div className="w-8 h-8 rounded-full bg-green-100 flex items-center justify-center">
                <Check size={16} className="text-green-600" />
              </div>
            ) : (
              <div className="w-8 h-8 rounded-full bg-gray-100 flex items-center justify-center">
                <X size={16} className="text-gray-400" />
              </div>
            )}
            <div>
              <div className="font-medium text-sm">
                {(record.data.name as string) || record.id}
              </div>
              <div className="text-xs text-gray-500">
                ID: {record.data.id as string || record.id}
              </div>
            </div>
          </div>
          
          <div className="flex items-center gap-4">
            <div className="text-right">
              <div className="text-xs text-gray-500">Execution Time</div>
              <div className="text-sm font-medium">{result.executionTimeMs.toFixed(2)}ms</div>
            </div>
            {result.matched && (
              <div className="flex gap-1">
                {result.triggeredActions.map(action => (
                  <span 
                    key={action}
                    className="text-xs px-2 py-0.5 bg-purple-100 text-purple-700 rounded"
                  >
                    {action}
                  </span>
                ))}
              </div>
            )}
            {isSelected ? <ChevronDown size={18} /> : <ChevronRight size={18} />}
          </div>
        </div>
        
        {isSelected && (
          <div className="border-t bg-gray-50 p-4">
            {/* Record Data */}
            <div className="mb-4">
              <h4 className="text-xs font-medium text-gray-500 mb-2 flex items-center gap-1">
                <FileJson size={12} />
                Record Data
              </h4>
              <pre className="text-xs bg-white border rounded p-2 overflow-x-auto">
                {JSON.stringify(record.data, null, 2)}
              </pre>
            </div>
            
            {/* Condition Results */}
            <div className="mb-4">
              <h4 className="text-xs font-medium text-gray-500 mb-2 flex items-center gap-1">
                <Filter size={12} />
                Condition Evaluation
              </h4>
              <div className="space-y-2">
                {result.conditionResults.map((cond, i) => (
                  <div 
                    key={i}
                    className={`flex items-center justify-between p-2 rounded border ${
                      cond.result ? 'bg-green-50 border-green-200' : 'bg-gray-50 border-gray-200'
                    }`}
                  >
                    <code className="text-xs">{cond.condition}</code>
                    <div className="flex items-center gap-3">
                      <span className="text-xs text-gray-500">
                        actual: <code>{JSON.stringify(cond.actualValue)}</code>
                      </span>
                      {cond.result ? (
                        <Check size={14} className="text-green-600" />
                      ) : (
                        <X size={14} className="text-gray-400" />
                      )}
                    </div>
                  </div>
                ))}
              </div>
            </div>
            
            {/* Debug Trace */}
            {debugMode && result.debugTrace && (
              <div>
                <h4 className="text-xs font-medium text-gray-500 mb-2 flex items-center gap-1">
                  <Bug size={12} />
                  Debug Trace
                </h4>
                <div className="bg-white border rounded overflow-hidden">
                  <table className="w-full text-xs">
                    <thead className="bg-gray-50">
                      <tr>
                        <th className="px-3 py-2 text-left">Step</th>
                        <th className="px-3 py-2 text-left">Expression</th>
                        <th className="px-3 py-2 text-left">Result</th>
                        <th className="px-3 py-2 text-right">Time</th>
                      </tr>
                    </thead>
                    <tbody>
                      {result.debugTrace.map(step => (
                        <tr key={step.step} className="border-t">
                          <td className="px-3 py-2 font-mono">{step.step}</td>
                          <td className="px-3 py-2 font-mono">{step.expression}</td>
                          <td className="px-3 py-2">
                            <code className="px-1 py-0.5 bg-gray-100 rounded">
                              {JSON.stringify(step.result)}
                            </code>
                          </td>
                          <td className="px-3 py-2 text-right text-gray-500">
                            {step.timeMs.toFixed(2)}ms
                          </td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
              </div>
            )}
          </div>
        )}
      </div>
    );
  };

  return (
    <div className="flex flex-col h-full bg-gray-50">
      {/* Header */}
      <div className="flex items-center justify-between p-4 border-b bg-white">
        <div className="flex items-center gap-3">
          <div className="p-2 bg-blue-100 rounded-lg">
            <Play size={18} className="text-blue-600" />
          </div>
          <div>
            <h3 className="font-semibold text-gray-900">Rule Testing</h3>
            <p className="text-xs text-gray-500">Testing: {rule.name}</p>
          </div>
        </div>
        <div className="flex items-center gap-2">
          <button
            onClick={() => setShowSettings(!showSettings)}
            className={`p-2 rounded transition-colors ${
              showSettings ? 'bg-gray-200 text-gray-700' : 'text-gray-400 hover:text-gray-600 hover:bg-gray-100'
            }`}
            title="Test settings"
            aria-label="Test settings"
          >
            <Settings size={18} />
          </button>
          {onClose && (
            <button
              onClick={onClose}
              className="p-2 text-gray-400 hover:text-gray-600 rounded hover:bg-gray-100"
              title="Close testing panel"
              aria-label="Close testing panel"
            >
              <X size={18} />
            </button>
          )}
        </div>
      </div>
      
      {/* Settings Panel */}
      {showSettings && (
        <div className="border-b bg-white p-4">
          <div className="grid grid-cols-3 gap-4">
            <div>
              <label className="block text-xs font-medium text-gray-500 mb-1">
                Sample Size
              </label>
              <input
                type="number"
                value={settings.sampleSize}
                onChange={(e) => setSettings(s => ({ ...s, sampleSize: parseInt(e.target.value) || 10 }))}
                className="w-full px-3 py-2 border rounded text-sm"
                min={1}
                max={1000}
                title="Sample size"
                aria-label="Sample size"
              />
            </div>
            <div>
              <label className="block text-xs font-medium text-gray-500 mb-1">
                Timeout (ms)
              </label>
              <input
                type="number"
                value={settings.timeout}
                onChange={(e) => setSettings(s => ({ ...s, timeout: parseInt(e.target.value) || 5000 }))}
                className="w-full px-3 py-2 border rounded text-sm"
                min={1000}
                max={60000}
                title="Timeout in milliseconds"
                aria-label="Timeout in milliseconds"
              />
            </div>
            <div>
              <label className="block text-xs font-medium text-gray-500 mb-1">
                Debug Mode
              </label>
              <button
                onClick={() => setDebugMode(!debugMode)}
                className={`w-full px-3 py-2 border rounded text-sm flex items-center justify-center gap-2 transition-colors ${
                  debugMode ? 'bg-purple-100 border-purple-300 text-purple-700' : 'bg-white text-gray-700'
                }`}
              >
                <Bug size={14} />
                {debugMode ? 'Enabled' : 'Disabled'}
              </button>
            </div>
          </div>
        </div>
      )}
      
      {/* Tabs */}
      <div className="flex border-b bg-white">
        {[
          { id: 'live', label: 'Live Test', icon: Play },
          { id: 'batch', label: 'Batch Import', icon: Upload },
          { id: 'historical', label: 'Historical Impact', icon: History },
          { id: 'performance', label: 'Performance', icon: BarChart2 }
        ].map(tab => (
          <button
            key={tab.id}
            onClick={() => setActiveTab(tab.id as typeof activeTab)}
            className={`flex-1 flex items-center justify-center gap-2 px-4 py-3 text-sm font-medium transition-colors ${
              activeTab === tab.id
                ? 'text-blue-600 border-b-2 border-blue-600'
                : 'text-gray-500 hover:text-gray-700'
            }`}
          >
            <tab.icon size={16} />
            <span>{tab.label}</span>
          </button>
        ))}
      </div>
      
      {/* Content */}
      <div className="flex-1 overflow-hidden">
        {activeTab === 'live' && (
          <div className="flex flex-col h-full">
            {/* Controls */}
            <div className="flex items-center justify-between p-4 border-b bg-white">
              <div className="flex items-center gap-2">
                <button
                  onClick={runEvaluation}
                  disabled={isRunning || testRecords.length === 0}
                  className="flex items-center gap-2 px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
                >
                  {isRunning ? (
                    <>
                      <RefreshCw size={16} className="animate-spin" />
                      Running...
                    </>
                  ) : (
                    <>
                      <Play size={16} />
                      Run Test
                    </>
                  )}
                </button>
                <button
                  onClick={() => setTestRecords(generateSampleData(settings.sampleSize))}
                  className="flex items-center gap-2 px-4 py-2 border rounded-lg hover:bg-gray-50 transition-colors"
                  title="Generate new sample data"
                >
                  <RotateCcw size={16} />
                  Regenerate
                </button>
              </div>
              
              <div className="flex items-center gap-2">
                <span className="text-sm text-gray-500">
                  {testRecords.length} records loaded
                </span>
                {results.length > 0 && (
                  <button
                    onClick={exportResults}
                    className="flex items-center gap-1 px-3 py-1.5 text-sm border rounded hover:bg-gray-50 transition-colors"
                    title="Export results"
                  >
                    <Download size={14} />
                    Export
                  </button>
                )}
              </div>
            </div>
            
            {/* Results */}
            <div className="flex-1 overflow-y-auto p-4">
              {renderSummaryCards()}
              
              {results.length > 0 ? (
                <div>
                  <h4 className="text-sm font-medium text-gray-700 mb-3 flex items-center gap-2">
                    <Layers size={16} />
                    Results ({results.length})
                  </h4>
                  {results.map((result) => {
                    const record = testRecords.find(r => r.id === result.recordId);
                    return record ? renderResultRow(result, record) : null;
                  })}
                </div>
              ) : (
                <div className="text-center py-12">
                  <Play size={48} className="mx-auto text-gray-300 mb-4" />
                  <h4 className="font-medium text-gray-600 mb-2">Ready to Test</h4>
                  <p className="text-sm text-gray-500">
                    Click "Run Test" to evaluate your rule against {testRecords.length} sample records
                  </p>
                </div>
              )}
            </div>
          </div>
        )}
        
        {activeTab === 'batch' && (
          <div className="p-4">
            <div className="bg-white rounded-lg border p-6">
              <h4 className="font-medium text-gray-900 mb-4 flex items-center gap-2">
                <FileJson size={18} />
                Import Test Data
              </h4>
              
              <div className="space-y-4">
                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-2">
                    Paste JSON data (array of objects or single object)
                  </label>
                  <textarea
                    value={customJson}
                    onChange={(e) => {
                      setCustomJson(e.target.value);
                      setJsonError(null);
                    }}
                    className={`w-full h-48 px-3 py-2 border rounded-lg font-mono text-sm ${
                      jsonError ? 'border-red-300 focus:ring-red-500' : 'focus:ring-blue-500'
                    }`}
                    placeholder='[{"id": "1", "name": "Test", "amount": 50000}, ...]'
                  />
                  {jsonError && (
                    <p className="mt-1 text-sm text-red-600 flex items-center gap-1">
                      <AlertTriangle size={14} />
                      {jsonError}
                    </p>
                  )}
                </div>
                
                <div className="flex items-center gap-4">
                  <button
                    onClick={handleJsonImport}
                    disabled={!customJson.trim()}
                    className="flex items-center gap-2 px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 disabled:opacity-50 disabled:cursor-not-allowed"
                  >
                    <Upload size={16} />
                    Import Data
                  </button>
                  
                  <div className="flex items-center gap-2 text-sm text-gray-500">
                    <Info size={14} />
                    Currently loaded: {testRecords.length} records
                  </div>
                </div>
              </div>
              
              <div className="mt-6 pt-6 border-t">
                <h5 className="text-sm font-medium text-gray-700 mb-3">Quick Templates</h5>
                <div className="flex flex-wrap gap-2">
                  {[
                    { label: 'Transaction', data: { id: '1', amount: 75000, status: 'pending', type: 'wire' } },
                    { label: 'Customer', data: { id: '1', name: 'Test Corp', risk_score: 85, category: 'enterprise' } },
                    { label: 'Trade', data: { id: '1', symbol: 'AAPL', quantity: 100, price: 150.00, side: 'buy' } }
                  ].map(template => (
                    <button
                      key={template.label}
                      onClick={() => setCustomJson(JSON.stringify([template.data], null, 2))}
                      className="px-3 py-1.5 text-sm border rounded hover:bg-gray-50 transition-colors"
                    >
                      {template.label}
                    </button>
                  ))}
                </div>
              </div>
            </div>
          </div>
        )}
        
        {activeTab === 'historical' && (
          <div className="p-4">
            {isLoadingHistory ? (
              <div className="flex items-center justify-center py-12">
                <RefreshCw size={24} className="animate-spin text-blue-500" />
                <span className="ml-2 text-gray-600">Analyzing historical data...</span>
              </div>
            ) : (
              <div className="space-y-4">
                <div className="bg-yellow-50 border border-yellow-200 rounded-lg p-4 flex items-start gap-3">
                  <AlertTriangle size={18} className="text-yellow-600 flex-shrink-0 mt-0.5" />
                  <div>
                    <h4 className="font-medium text-yellow-800">Historical Impact Analysis</h4>
                    <p className="text-sm text-yellow-700">
                      This analysis shows how the rule would have performed against historical data. 
                      Results are estimates and may vary in production.
                    </p>
                  </div>
                </div>
                
                {historicalData.map((period, i) => (
                  <div key={i} className="bg-white rounded-lg border p-4">
                    <div className="flex items-center justify-between mb-4">
                      <h4 className="font-medium text-gray-900">{period.period}</h4>
                      <span className="text-sm text-gray-500">
                        {period.recordsAnalyzed.toLocaleString()} records analyzed
                      </span>
                    </div>
                    
                    <div className="grid grid-cols-3 gap-4 mb-4">
                      <div className="bg-gray-50 rounded p-3">
                        <div className="text-xs text-gray-500">Would Match</div>
                        <div className="text-xl font-semibold text-green-600">
                          {period.wouldMatch.toLocaleString()}
                        </div>
                      </div>
                      <div className="bg-gray-50 rounded p-3">
                        <div className="text-xs text-gray-500">Match Rate</div>
                        <div className="text-xl font-semibold">
                          {period.matchRate.toFixed(1)}%
                        </div>
                      </div>
                      <div className="bg-gray-50 rounded p-3">
                        <div className="text-xs text-gray-500">Top Entity</div>
                        <div className="text-sm font-medium truncate">
                          {period.topMatchingEntities[0]?.name || 'N/A'}
                        </div>
                      </div>
                    </div>
                    
                    <div>
                      <h5 className="text-xs font-medium text-gray-500 mb-2">Top Matching Entities</h5>
                      <div className="space-y-1">
                        {period.topMatchingEntities.map((entity, j) => (
                          <div key={j} className="flex items-center justify-between text-sm">
                            <span className="text-gray-700">{entity.name}</span>
                            <span className="text-gray-500">{entity.matchCount.toLocaleString()} matches</span>
                          </div>
                        ))}
                      </div>
                    </div>
                  </div>
                ))}
                
                <button
                  onClick={loadHistoricalAnalysis}
                  className="w-full py-2 text-sm text-blue-600 hover:text-blue-700 flex items-center justify-center gap-2"
                >
                  <RefreshCw size={14} />
                  Refresh Analysis
                </button>
              </div>
            )}
          </div>
        )}
        
        {activeTab === 'performance' && (
          <div className="p-4">
            {summary ? (
              <div className="space-y-4">
                <div className="bg-white rounded-lg border p-4">
                  <h4 className="font-medium text-gray-900 mb-4 flex items-center gap-2">
                    <BarChart2 size={18} />
                    Performance Metrics
                  </h4>
                  
                  <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
                    <div className="text-center p-3 bg-gray-50 rounded">
                      <div className="text-xs text-gray-500">Min Time</div>
                      <div className="text-lg font-semibold">{summary.minExecutionTimeMs.toFixed(2)}ms</div>
                    </div>
                    <div className="text-center p-3 bg-gray-50 rounded">
                      <div className="text-xs text-gray-500">Avg Time</div>
                      <div className="text-lg font-semibold">{summary.avgExecutionTimeMs.toFixed(2)}ms</div>
                    </div>
                    <div className="text-center p-3 bg-gray-50 rounded">
                      <div className="text-xs text-gray-500">P95 Time</div>
                      <div className="text-lg font-semibold">{summary.p95ExecutionTimeMs.toFixed(2)}ms</div>
                    </div>
                    <div className="text-center p-3 bg-gray-50 rounded">
                      <div className="text-xs text-gray-500">Max Time</div>
                      <div className="text-lg font-semibold">{summary.maxExecutionTimeMs.toFixed(2)}ms</div>
                    </div>
                  </div>
                </div>
                
                <div className="bg-white rounded-lg border p-4">
                  <h4 className="font-medium text-gray-900 mb-4 flex items-center gap-2">
                    <Zap size={18} />
                    Actions Triggered
                  </h4>
                  
                  {Object.keys(summary.actionCounts).length > 0 ? (
                    <div className="space-y-2">
                      {Object.entries(summary.actionCounts).map(([action, count]) => {
                        const widthPercent = Math.round((count / summary.totalRecords) * 100);
                        // Use predefined width classes to avoid inline styles
                        const widthClass = widthPercent >= 100 ? 'w-full' :
                          widthPercent >= 75 ? 'w-3/4' :
                          widthPercent >= 50 ? 'w-1/2' :
                          widthPercent >= 25 ? 'w-1/4' :
                          widthPercent >= 10 ? 'w-[10%]' : 'w-1';
                        return (
                          <div key={action} className="flex items-center justify-between">
                            <span className="text-sm text-gray-700">{action}</span>
                            <div className="flex items-center gap-2">
                              <div className="w-32 h-2 bg-gray-200 rounded-full overflow-hidden" title={`${widthPercent}% of records`}>
                                <div className={`h-full bg-purple-500 rounded-full ${widthClass}`} />
                              </div>
                              <span className="text-sm text-gray-500 w-12 text-right">{count}</span>
                            </div>
                          </div>
                        );
                      })}
                    </div>
                  ) : (
                    <p className="text-sm text-gray-500">No actions triggered during test</p>
                  )}
                </div>
                
                <div className="bg-blue-50 border border-blue-200 rounded-lg p-4">
                  <h4 className="font-medium text-blue-800 mb-2 flex items-center gap-2">
                    <Info size={16} />
                    Performance Recommendations
                  </h4>
                  <ul className="text-sm text-blue-700 space-y-1">
                    {summary.avgExecutionTimeMs > 5 && (
                      <li>• Consider adding indexes on frequently queried fields</li>
                    )}
                    {summary.p95ExecutionTimeMs > summary.avgExecutionTimeMs * 2 && (
                      <li>• High variance in execution time - review complex conditions</li>
                    )}
                    <li>• Enable batch mode for processing large datasets</li>
                  </ul>
                </div>
              </div>
            ) : (
              <div className="text-center py-12">
                <BarChart2 size={48} className="mx-auto text-gray-300 mb-4" />
                <h4 className="font-medium text-gray-600 mb-2">No Performance Data</h4>
                <p className="text-sm text-gray-500">
                  Run a test first to see performance metrics
                </p>
              </div>
            )}
          </div>
        )}
      </div>
    </div>
  );
};

export default RuleTestingPanel;
