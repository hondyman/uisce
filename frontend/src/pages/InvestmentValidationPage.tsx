import React, { useState, useEffect } from 'react';
import { useNotification } from '../hooks/useNotification';
import {
  CheckCircle,
  AlertCircle,
  XCircle,
  Clock,
  Plus,
  Play,
  Settings,
  TrendingUp,
  DollarSign,
  Shield,
  Zap,
} from 'lucide-react';
import { useTenant } from '../contexts/TenantContext';
import InvestmentValidationEngine, {
  ValidationContext,
  ValidationExecutionResult,
  RuleSeverity,
  getComplianceStatus,
  formatResultMessage,
  groupResultsBySeverity,
  getSeverityIcon,
} from '../services/validationEngine';
import {
  getRuleTypeMetadata,
  formatPercentage,
  formatCurrency,
  RULE_TYPES,
  ACCOUNT_TYPES,
} from '../lib/validationConstants';
import { devLog } from '../utils/devLogger';

/**
 * Investment Validation Page
 * Main UI for running and managing wealth management validations
 */
export const InvestmentValidationPage: React.FC = () => {
  const { tenant, datasource } = useTenant();

  const [engine, setEngine] = useState<InvestmentValidationEngine | null>(null);
  const [loading, setLoading] = useState(false);
  const [validating, setValidating] = useState(false);
  const notification = useNotification();

  const [selectedAccount, setSelectedAccount] = useState('');
  const [selectedAccountType, setSelectedAccountType] = useState('INDIVIDUAL_ACCOUNT');
  const [executionResult, setExecutionResult] = useState<ValidationExecutionResult | null>(null);
  const [history, setHistory] = useState<ValidationExecutionResult[]>([]);

  const [sampleData, setSampleData] = useState({
    portfolioValue: 1000000,
    concentrationPositions: [
      { ticker: 'AAPL', value: 350000, percent: 35 },
      { ticker: 'MSFT', value: 250000, percent: 25 },
      { ticker: 'VTSAX', value: 400000, percent: 40 },
    ],
    cash: 50000,
  });

  const [expandedRule, setExpandedRule] = useState<string | null>(null);

  // Initialize validation engine
  useEffect(() => {
    if (tenant && datasource) {
      const newEngine = new InvestmentValidationEngine(tenant.id, datasource.id);
      setEngine(newEngine);
      devLog('Validation engine initialized', { tenantId: tenant.id, datasourceId: datasource.id });
    }
  }, [tenant, datasource]);

  // Load validation history
  useEffect(() => {
    if (engine && selectedAccount) {
      loadHistory();
    }
  }, [engine, selectedAccount]);

  const loadHistory = async () => {
    if (!engine || !selectedAccount) return;
    setLoading(true);
    try {
      const thirtyDaysAgo = new Date();
      thirtyDaysAgo.setDate(thirtyDaysAgo.getDate() - 30);
      
      const hist = await engine.getValidationHistory(selectedAccount, thirtyDaysAgo);
      setHistory(hist);
    } catch (error) {
      console.error('Failed to load validation history:', error);
    } finally {
      setLoading(false);
    }
  };

  const runValidation = async () => {
    if (!engine || !selectedAccount) {
      const notification = useNotification();
      notification.error('Please select an account');
      return;
    }

    setValidating(true);
    try {
      const context: ValidationContext = {
        accountId: selectedAccount,
        accountType: selectedAccountType,
        clientId: `client-${selectedAccount}`,
        timestamp: new Date(),
        tenantId: tenant?.id || '',
        datasourceId: datasource?.id || '',
        portfolioData: {
          totalValue: sampleData.portfolioValue,
          cash: sampleData.cash,
          positions: sampleData.concentrationPositions.map((p) => ({
            ticker: p.ticker,
            marketValue: p.value,
            assetType: 'EQUITY',
            costBasis: p.value * 0.95,
          })),
        },
        clientProfile: {
          fullName: 'John Doe',
          dateOfBirth: new Date('1975-05-15'),
          riskTolerance: 'MODERATE',
          investmentObjective: 'GROWTH',
          netWorth: sampleData.portfolioValue * 2.5,
          accreditedInvestorStatus: true,
          pepStatus: 'CLEAR',
        },
        transactionData: {
          type: 'REBALANCE',
          amount: 50000,
          feePercentage: 0.005,
        },
      };

      const result = await engine.executeValidations(context);
      setExecutionResult(result);
      
      // Reload history
      await loadHistory();
    } catch (error) {
      console.error('Validation execution failed:', error);
      const notification = useNotification();
      notification.error('Validation execution failed: ' + (error instanceof Error ? error.message : 'Unknown error'));
    } finally {
      setValidating(false);
    }
  };

  if (!tenant || !datasource) {
    return (
      <div className="p-8 bg-gradient-to-br from-blue-50 to-blue-50/50 dark:from-blue-950/20 dark:to-blue-950/10 rounded-lg">
        <div className="flex items-center gap-4">
          <AlertCircle className="w-6 h-6 text-blue-600 dark:text-blue-400" />
          <p className="text-blue-900 dark:text-blue-300">
            Please select a tenant and datasource to use the validation engine.
          </p>
        </div>
      </div>
    );
  }

  const complianceStatus = executionResult ? getComplianceStatus(executionResult) : null;

  return (
    <div className="space-y-8">
      {/* Header */}
      <div className="space-y-2">
        <h1 className="text-4xl font-bold bg-gradient-to-r from-slate-900 via-slate-800 to-slate-900 dark:from-white dark:via-slate-100 dark:to-white bg-clip-text text-transparent">
          Investment Validation Engine
        </h1>
        <p className="text-lg text-slate-600 dark:text-slate-400">
          Run wealth management validation rules on portfolio accounts to ensure compliance
        </p>
      </div>

      {/* Controls */}
      <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
        {/* Account Selection */}
        <div className="bg-white dark:bg-slate-900 rounded-lg p-6 border border-slate-200 dark:border-slate-800">
          <label className="block text-sm font-semibold text-slate-900 dark:text-white mb-3">
            Select Account
          </label>
          <input
            type="text"
            placeholder="e.g., ACC-001"
            value={selectedAccount}
            onChange={(e) => setSelectedAccount(e.target.value)}
            className="w-full px-4 py-2 rounded-lg border border-slate-300 dark:border-slate-600 dark:bg-slate-800 dark:text-white"
          />
        </div>

        {/* Account Type */}
        <div className="bg-white dark:bg-slate-900 rounded-lg p-6 border border-slate-200 dark:border-slate-800">
          <label htmlFor="accountType" className="block text-sm font-semibold text-slate-900 dark:text-white mb-3">
            Account Type
          </label>
          <select
            id="accountType"
            value={selectedAccountType}
            onChange={(e) => setSelectedAccountType(e.target.value)}
            className="w-full px-4 py-2 rounded-lg border border-slate-300 dark:border-slate-600 dark:bg-slate-800 dark:text-white"
          >
            {Object.values(ACCOUNT_TYPES).map((type) => (
              <option key={type.id} value={type.id}>
                {type.label}
              </option>
            ))}
          </select>
        </div>

        {/* Run Validation Button */}
        <div className="bg-white dark:bg-slate-900 rounded-lg p-6 border border-slate-200 dark:border-slate-800 flex items-end">
          <button
            onClick={runValidation}
            disabled={validating || !selectedAccount}
            className="w-full px-6 py-2 bg-blue-600 hover:bg-blue-700 disabled:bg-slate-400 text-white font-semibold rounded-lg flex items-center justify-center gap-2 transition"
          >
            {validating ? (
              <>
                <Clock className="w-4 h-4 animate-spin" />
                Validating...
              </>
            ) : (
              <>
                <Play className="w-4 h-4" />
                Run Validation
              </>
            )}
          </button>
        </div>
      </div>

      {/* Portfolio Summary */}
      <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
        <div className="bg-white dark:bg-slate-900 rounded-lg p-4 border border-slate-200 dark:border-slate-800">
          <div className="flex items-center gap-3 mb-2">
            <DollarSign className="w-4 h-4 text-green-600 dark:text-green-400" />
            <span className="text-sm text-slate-600 dark:text-slate-400">Portfolio Value</span>
          </div>
          <p className="text-2xl font-bold text-slate-900 dark:text-white">
            {formatCurrency(sampleData.portfolioValue)}
          </p>
        </div>

        <div className="bg-white dark:bg-slate-900 rounded-lg p-4 border border-slate-200 dark:border-slate-800">
          <div className="flex items-center gap-3 mb-2">
            <TrendingUp className="w-4 h-4 text-blue-600 dark:text-blue-400" />
            <span className="text-sm text-slate-600 dark:text-slate-400">Positions</span>
          </div>
          <p className="text-2xl font-bold text-slate-900 dark:text-white">
            {sampleData.concentrationPositions.length}
          </p>
        </div>

        <div className="bg-white dark:bg-slate-900 rounded-lg p-4 border border-slate-200 dark:border-slate-800">
          <div className="flex items-center gap-3 mb-2">
            <Shield className="w-4 h-4 text-purple-600 dark:text-purple-400" />
            <span className="text-sm text-slate-600 dark:text-slate-400">Cash Available</span>
          </div>
          <p className="text-2xl font-bold text-slate-900 dark:text-white">
            {formatCurrency(sampleData.cash)}
          </p>
        </div>

        <div className="bg-white dark:bg-slate-900 rounded-lg p-4 border border-slate-200 dark:border-slate-800">
          <div className="flex items-center gap-3 mb-2">
            <Zap className="w-4 h-4 text-yellow-600 dark:text-yellow-400" />
            <span className="text-sm text-slate-600 dark:text-slate-400">Max Concentration</span>
          </div>
          <p className="text-2xl font-bold text-slate-900 dark:text-white">
            {formatPercentage(Math.max(...sampleData.concentrationPositions.map((p) => p.percent / 100)))}
          </p>
        </div>
      </div>

      {/* Validation Results */}
      {executionResult && (
        <div className="space-y-6">
          {/* Status Banner */}
          <div
            className={`rounded-lg p-6 border-2 ${
              complianceStatus
                ? `border-${complianceStatus.status === 'pass' ? 'green' : complianceStatus.status === 'warn' ? 'yellow' : 'red'}-200 dark:border-${complianceStatus.status === 'pass' ? 'green' : complianceStatus.status === 'warn' ? 'yellow' : 'red'}-800 ${complianceStatus.color}`
                : ''
            }`}
          >
            <div className="flex items-start gap-4">
              {complianceStatus?.status === 'pass' && (
                <CheckCircle className="w-6 h-6 text-green-600 dark:text-green-400 flex-shrink-0 mt-1" />
              )}
              {complianceStatus?.status === 'warn' && (
                <AlertCircle className="w-6 h-6 text-yellow-600 dark:text-yellow-400 flex-shrink-0 mt-1" />
              )}
              {complianceStatus?.status === 'fail' && (
                <XCircle className="w-6 h-6 text-red-600 dark:text-red-400 flex-shrink-0 mt-1" />
              )}
              <div className="flex-1">
                <h3 className="text-lg font-bold mb-1">
                  {complianceStatus?.label}
                </h3>
                <p className="text-sm opacity-90">
                  Validation completed in {executionResult.executionTimeMs}ms with {executionResult.results.length} rules evaluated
                </p>
              </div>
            </div>
          </div>

          {/* Results Summary */}
          <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
            <div className="bg-red-50 dark:bg-red-950/20 rounded-lg p-4 border border-red-200 dark:border-red-800">
              <div className="text-sm text-red-900 dark:text-red-300 mb-2">Blocked Rules</div>
              <p className="text-3xl font-bold text-red-600 dark:text-red-400">
                {executionResult.blockedRules.length}
              </p>
            </div>

            <div className="bg-yellow-50 dark:bg-yellow-950/20 rounded-lg p-4 border border-yellow-200 dark:border-yellow-800">
              <div className="text-sm text-yellow-900 dark:text-yellow-300 mb-2">Warnings</div>
              <p className="text-3xl font-bold text-yellow-600 dark:text-yellow-400">
                {executionResult.warningRules.length}
              </p>
            </div>

            <div className="bg-blue-50 dark:bg-blue-950/20 rounded-lg p-4 border border-blue-200 dark:border-blue-800">
              <div className="text-sm text-blue-900 dark:text-blue-300 mb-2">Info Messages</div>
              <p className="text-3xl font-bold text-blue-600 dark:text-blue-400">
                {executionResult.infoRules.length}
              </p>
            </div>

            <div className="bg-green-50 dark:bg-green-950/20 rounded-lg p-4 border border-green-200 dark:border-green-800">
              <div className="text-sm text-green-900 dark:text-green-300 mb-2">Passed</div>
              <p className="text-3xl font-bold text-green-600 dark:text-green-400">
                {executionResult.results.filter((r) => r.passed).length}
              </p>
            </div>
          </div>

          {/* Blocked Rules */}
          {executionResult.blockedRules.length > 0 && (
            <div className="bg-white dark:bg-slate-900 rounded-lg border border-red-200 dark:border-red-800 overflow-hidden">
              <div className="bg-red-50 dark:bg-red-950/20 px-6 py-4 border-b border-red-200 dark:border-red-800">
                <h3 className="font-bold text-red-900 dark:text-red-300 flex items-center gap-2">
                  <XCircle className="w-5 h-5" />
                  Blocked Rules ({executionResult.blockedRules.length})
                </h3>
              </div>
              <div className="divide-y divide-slate-200 dark:divide-slate-800">
                {executionResult.blockedRules.map((rule) => (
                  <div
                    key={rule.ruleId}
                    className="p-4 cursor-pointer hover:bg-slate-50 dark:hover:bg-slate-800/50 transition"
                    onClick={() =>
                      setExpandedRule(expandedRule === rule.ruleId ? null : rule.ruleId)
                    }
                  >
                    <div className="flex items-start gap-3">
                      <span className="text-2xl">{getSeverityIcon(rule.severity)}</span>
                      <div className="flex-1">
                        <p className="font-semibold text-slate-900 dark:text-white">
                          {rule.ruleName}
                        </p>
                        <p className="text-sm text-slate-600 dark:text-slate-400 mt-1">
                          {formatResultMessage(rule)}
                        </p>
                        {expandedRule === rule.ruleId && rule.details && (
                          <div className="mt-3 p-3 bg-slate-100 dark:bg-slate-800 rounded text-sm">
                            <pre className="text-slate-700 dark:text-slate-300 overflow-auto max-h-40">
                              {JSON.stringify(rule.details, null, 2)}
                            </pre>
                          </div>
                        )}
                      </div>
                    </div>
                  </div>
                ))}
              </div>
            </div>
          )}

          {/* Warning Rules */}
          {executionResult.warningRules.length > 0 && (
            <div className="bg-white dark:bg-slate-900 rounded-lg border border-yellow-200 dark:border-yellow-800 overflow-hidden">
              <div className="bg-yellow-50 dark:bg-yellow-950/20 px-6 py-4 border-b border-yellow-200 dark:border-yellow-800">
                <h3 className="font-bold text-yellow-900 dark:text-yellow-300 flex items-center gap-2">
                  <AlertCircle className="w-5 h-5" />
                  Warnings ({executionResult.warningRules.length})
                </h3>
              </div>
              <div className="divide-y divide-slate-200 dark:divide-slate-800">
                {executionResult.warningRules.map((rule) => (
                  <div key={rule.ruleId} className="p-4">
                    <div className="flex items-start gap-3">
                      <span className="text-2xl">{getSeverityIcon(rule.severity)}</span>
                      <div className="flex-1">
                        <p className="font-semibold text-slate-900 dark:text-white">
                          {rule.ruleName}
                        </p>
                        <p className="text-sm text-slate-600 dark:text-slate-400 mt-1">
                          {formatResultMessage(rule)}
                        </p>
                      </div>
                    </div>
                  </div>
                ))}
              </div>
            </div>
          )}

          {/* All Results Table */}
          <div className="bg-white dark:bg-slate-900 rounded-lg border border-slate-200 dark:border-slate-800 overflow-hidden">
            <div className="px-6 py-4 border-b border-slate-200 dark:border-slate-800">
              <h3 className="font-bold text-slate-900 dark:text-white">
                All Validation Results ({executionResult.results.length})
              </h3>
            </div>
            <div className="overflow-x-auto">
              <table className="w-full">
                <thead className="bg-slate-50 dark:bg-slate-800/50 border-b border-slate-200 dark:border-slate-800">
                  <tr>
                    <th className="px-6 py-3 text-left text-sm font-semibold text-slate-900 dark:text-white">
                      Rule Name
                    </th>
                    <th className="px-6 py-3 text-left text-sm font-semibold text-slate-900 dark:text-white">
                      Status
                    </th>
                    <th className="px-6 py-3 text-left text-sm font-semibold text-slate-900 dark:text-white">
                      Severity
                    </th>
                    <th className="px-6 py-3 text-left text-sm font-semibold text-slate-900 dark:text-white">
                      Message
                    </th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-slate-200 dark:divide-slate-800">
                  {executionResult.results.map((result) => (
                    <tr
                      key={result.ruleId}
                      className="hover:bg-slate-50 dark:hover:bg-slate-800/50 transition"
                    >
                      <td className="px-6 py-4 text-sm font-medium text-slate-900 dark:text-white">
                        {result.ruleName}
                      </td>
                      <td className="px-6 py-4 text-sm">
                        {result.passed ? (
                          <span className="text-green-600 dark:text-green-400 font-semibold">
                            ✓ Passed
                          </span>
                        ) : (
                          <span className="text-red-600 dark:text-red-400 font-semibold">
                            ✗ Failed
                          </span>
                        )}
                      </td>
                      <td className="px-6 py-4 text-sm">
                        <span>{getSeverityIcon(result.severity)} {result.severity}</span>
                      </td>
                      <td className="px-6 py-4 text-sm text-slate-600 dark:text-slate-400">
                        {result.message.substring(0, 60)}...
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          </div>
        </div>
      )}

      {/* History */}
      {history.length > 0 && (
        <div className="bg-white dark:bg-slate-900 rounded-lg border border-slate-200 dark:border-slate-800 overflow-hidden">
          <div className="px-6 py-4 border-b border-slate-200 dark:border-slate-800">
            <h3 className="font-bold text-slate-900 dark:text-white">
              Validation History (Last 30 Days)
            </h3>
          </div>
          <div className="overflow-x-auto">
            <table className="w-full">
              <thead className="bg-slate-50 dark:bg-slate-800/50 border-b border-slate-200 dark:border-slate-800">
                <tr>
                  <th className="px-6 py-3 text-left text-sm font-semibold text-slate-900 dark:text-white">
                    Date
                  </th>
                  <th className="px-6 py-3 text-left text-sm font-semibold text-slate-900 dark:text-white">
                    Status
                  </th>
                  <th className="px-6 py-3 text-left text-sm font-semibold text-slate-900 dark:text-white">
                    Issues
                  </th>
                  <th className="px-6 py-3 text-left text-sm font-semibold text-slate-900 dark:text-white">
                    Time
                  </th>
                </tr>
              </thead>
              <tbody className="divide-y divide-slate-200 dark:divide-slate-800">
                {history.slice(0, 10).map((item, idx) => (
                  <tr key={idx} className="hover:bg-slate-50 dark:hover:bg-slate-800/50 transition">
                    <td className="px-6 py-4 text-sm text-slate-900 dark:text-white">
                      {new Date(item.timestamp).toLocaleString()}
                    </td>
                    <td className="px-6 py-4 text-sm">
                      {item.passed ? (
                        <span className="text-green-600 dark:text-green-400">✓ Pass</span>
                      ) : (
                        <span className="text-red-600 dark:text-red-400">✗ Fail</span>
                      )}
                    </td>
                    <td className="px-6 py-4 text-sm">
                      <span className="text-red-600 dark:text-red-400 font-semibold">
                        {item.blockedRules.length}
                      </span>
                      {' blocked, '}
                      <span className="text-yellow-600 dark:text-yellow-400 font-semibold">
                        {item.warningRules.length}
                      </span>
                      {' warnings'}
                    </td>
                    <td className="px-6 py-4 text-sm text-slate-600 dark:text-slate-400">
                      {item.executionTimeMs}ms
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
      )}
    </div>
  );
};

export default InvestmentValidationPage;
