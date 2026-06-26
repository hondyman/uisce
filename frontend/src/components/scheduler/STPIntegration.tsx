/**
 * STPIntegration.tsx
 * 
 * Straight-Through Processing (STP) Integration:
 * - Trade lifecycle automation with real-time status tracking
 * - Settlement workflow monitoring and exception handling
 * - Multi-custodian integration status dashboard
 * - Automated reconciliation and fail resolution
 */

import React, { useState, useEffect, useMemo } from 'react';
import {
  Activity,
  CheckCircle,
  Clock,
  AlertTriangle,
  RefreshCw,
  Zap,
  Settings,
  Filter,
  Search,
  TrendingUp,
  BarChart3,
  Link2,
  Database
} from 'lucide-react';

// ============================================================================
// Types
// ============================================================================

interface Trade {
  id: string;
  orderId: string;
  accountId: string;
  accountName: string;
  symbol: string;
  side: 'BUY' | 'SELL';
  quantity: number;
  price: number;
  currency: string;
  custodian: string;
  status: TradeStatus;
  settledDate?: Date;
  tradeDate: Date;
  expectedSettleDate: Date;
  lifecycle: LifecycleStage[];
  exceptions: TradeException[];
  stpRate: number;
}

type TradeStatus = 
  | 'PENDING'
  | 'CONFIRMED'
  | 'MATCHED'
  | 'SETTLING'
  | 'SETTLED'
  | 'FAILED'
  | 'CANCELLED';

interface LifecycleStage {
  stage: string;
  status: 'COMPLETED' | 'IN_PROGRESS' | 'PENDING' | 'FAILED';
  timestamp?: Date;
  details?: string;
}

interface TradeException {
  id: string;
  type: ExceptionType;
  severity: 'LOW' | 'MEDIUM' | 'HIGH' | 'CRITICAL';
  message: string;
  timestamp: Date;
  resolved: boolean;
  resolution?: string;
}

type ExceptionType = 
  | 'QUANTITY_MISMATCH'
  | 'PRICE_VARIANCE'
  | 'SETTLEMENT_INSTRUCTION_MISSING'
  | 'CUSTODY_FAIL'
  | 'COUNTERPARTY_REJECT'
  | 'COMPLIANCE_BLOCK'
  | 'FUNDING_ISSUE';

interface CustodianStatus {
  id: string;
  name: string;
  status: 'CONNECTED' | 'DEGRADED' | 'DISCONNECTED';
  lastSync: Date;
  pendingTrades: number;
  settledToday: number;
  failRate: number;
  avgSettleTime: number;
}

interface STPMetrics {
  totalTrades: number;
  stpRate: number;
  manualInterventions: number;
  avgCycleTime: number;
  pendingSettlements: number;
  failedTrades: number;
  settledToday: number;
}

// ============================================================================
// Constants
// ============================================================================

const STATUS_CONFIG: Record<TradeStatus, { color: string; bgColor: string; label: string }> = {
  PENDING: { color: 'text-yellow-600', bgColor: 'bg-yellow-100', label: 'Pending' },
  CONFIRMED: { color: 'text-blue-600', bgColor: 'bg-blue-100', label: 'Confirmed' },
  MATCHED: { color: 'text-purple-600', bgColor: 'bg-purple-100', label: 'Matched' },
  SETTLING: { color: 'text-indigo-600', bgColor: 'bg-indigo-100', label: 'Settling' },
  SETTLED: { color: 'text-green-600', bgColor: 'bg-green-100', label: 'Settled' },
  FAILED: { color: 'text-red-600', bgColor: 'bg-red-100', label: 'Failed' },
  CANCELLED: { color: 'text-gray-600', bgColor: 'bg-gray-100', label: 'Cancelled' }
};

const LIFECYCLE_STAGES = [
  'ORDER_RECEIVED',
  'COMPLIANCE_CHECK',
  'BROKER_SENT',
  'EXECUTION',
  'ALLOCATION',
  'CONFIRMATION',
  'AFFIRMATION',
  'SETTLEMENT_INSTRUCTION',
  'SETTLEMENT'
];

const EXCEPTION_COLORS: Record<string, string> = {
  LOW: 'bg-blue-100 text-blue-800 border-blue-200',
  MEDIUM: 'bg-yellow-100 text-yellow-800 border-yellow-200',
  HIGH: 'bg-orange-100 text-orange-800 border-orange-200',
  CRITICAL: 'bg-red-100 text-red-800 border-red-200'
};

// ============================================================================
// Mock Data
// ============================================================================

const MOCK_CUSTODIANS: CustodianStatus[] = [
  { id: 'cust1', name: 'Charles Schwab', status: 'CONNECTED', lastSync: new Date(), pendingTrades: 45, settledToday: 234, failRate: 0.8, avgSettleTime: 1.2 },
  { id: 'cust2', name: 'Fidelity', status: 'CONNECTED', lastSync: new Date(), pendingTrades: 32, settledToday: 189, failRate: 0.5, avgSettleTime: 1.1 },
  { id: 'cust3', name: 'Pershing', status: 'DEGRADED', lastSync: new Date(Date.now() - 15 * 60 * 1000), pendingTrades: 67, settledToday: 156, failRate: 2.1, avgSettleTime: 1.8 },
  { id: 'cust4', name: 'TD Ameritrade', status: 'CONNECTED', lastSync: new Date(), pendingTrades: 28, settledToday: 145, failRate: 0.6, avgSettleTime: 1.3 }
];

const generateMockTrades = (): Trade[] => {
  const symbols = ['AAPL', 'GOOGL', 'MSFT', 'AMZN', 'META', 'TSLA', 'NVDA', 'BRK.B'];
  const custodians = ['Charles Schwab', 'Fidelity', 'Pershing', 'TD Ameritrade'];
  const statuses: TradeStatus[] = ['PENDING', 'CONFIRMED', 'MATCHED', 'SETTLING', 'SETTLED', 'FAILED'];
  
  return Array.from({ length: 25 }, (_, i) => {
    const status = statuses[Math.floor(Math.random() * statuses.length)];
    const tradeDate = new Date(Date.now() - Math.random() * 3 * 24 * 60 * 60 * 1000);
    const expectedSettleDate = new Date(tradeDate.getTime() + 2 * 24 * 60 * 60 * 1000);
    
    const lifecycle: LifecycleStage[] = LIFECYCLE_STAGES.map((stage, _idx) => {
      const stageIdx = LIFECYCLE_STAGES.indexOf(stage);
      const completedStages = status === 'SETTLED' ? 9 : 
        status === 'SETTLING' ? 7 :
        status === 'MATCHED' ? 5 :
        status === 'CONFIRMED' ? 3 :
        status === 'FAILED' ? Math.floor(Math.random() * 6) :
        1;
      
      return {
        stage,
        status: stageIdx < completedStages ? 'COMPLETED' :
          stageIdx === completedStages ? (status === 'FAILED' ? 'FAILED' : 'IN_PROGRESS') :
          'PENDING',
        timestamp: stageIdx < completedStages ? new Date(tradeDate.getTime() + stageIdx * 2 * 60 * 60 * 1000) : undefined
      };
    });

    const exceptions: TradeException[] = status === 'FAILED' ? [{
      id: `exc-${i}`,
      type: ['QUANTITY_MISMATCH', 'PRICE_VARIANCE', 'CUSTODY_FAIL'][Math.floor(Math.random() * 3)] as ExceptionType,
      severity: 'HIGH',
      message: 'Settlement instruction mismatch detected',
      timestamp: new Date(),
      resolved: false
    }] : [];

    return {
      id: `trade-${i + 1}`,
      orderId: `ORD-${100000 + i}`,
      accountId: `ACC-${10000 + Math.floor(Math.random() * 1000)}`,
      accountName: `Client Account ${Math.floor(Math.random() * 100)}`,
      symbol: symbols[Math.floor(Math.random() * symbols.length)],
      side: Math.random() > 0.5 ? 'BUY' : 'SELL',
      quantity: Math.floor(Math.random() * 1000) + 100,
      price: Math.random() * 500 + 50,
      currency: 'USD',
      custodian: custodians[Math.floor(Math.random() * custodians.length)],
      status,
      tradeDate,
      expectedSettleDate,
      settledDate: status === 'SETTLED' ? new Date() : undefined,
      lifecycle,
      exceptions,
      stpRate: Math.random() * 100
    };
  });
};

// ============================================================================
// Helper Components
// ============================================================================

const STATUS_BAR_COLORS: Record<TradeStatus, string> = {
  PENDING: 'bg-yellow-500',
  CONFIRMED: 'bg-blue-500',
  MATCHED: 'bg-purple-500',
  SETTLING: 'bg-indigo-500',
  SETTLED: 'bg-green-500',
  FAILED: 'bg-red-500',
  CANCELLED: 'bg-gray-500'
};

const StatusBar: React.FC<{ percentage: number; status: TradeStatus }> = ({ percentage, status }) => {
  const colorClass = STATUS_BAR_COLORS[status];
  const widthClass = percentage >= 100 ? 'w-full' : 
    percentage >= 90 ? 'w-[90%]' : 
    percentage >= 80 ? 'w-[80%]' : 
    percentage >= 70 ? 'w-[70%]' : 
    percentage >= 60 ? 'w-[60%]' : 
    percentage >= 50 ? 'w-[50%]' : 
    percentage >= 40 ? 'w-[40%]' : 
    percentage >= 30 ? 'w-[30%]' : 
    percentage >= 20 ? 'w-[20%]' : 
    percentage >= 10 ? 'w-[10%]' : 
    percentage >= 5 ? 'w-[5%]' : 'w-0';
  return <div className={`h-full rounded-full ${colorClass} ${widthClass}`} />;
};

const StatusIndicator: React.FC<{ status: 'CONNECTED' | 'DEGRADED' | 'DISCONNECTED' }> = ({ status }) => {
  const colors = {
    CONNECTED: 'bg-green-500',
    DEGRADED: 'bg-yellow-500',
    DISCONNECTED: 'bg-red-500'
  };
  return (
    <div className="flex items-center gap-2">
      <span className={`w-2 h-2 rounded-full ${colors[status]} ${status === 'DEGRADED' ? 'animate-pulse' : ''}`} />
      <span className={`text-xs ${status === 'CONNECTED' ? 'text-green-600' : status === 'DEGRADED' ? 'text-yellow-600' : 'text-red-600'}`}>
        {status}
      </span>
    </div>
  );
};

const LifecycleProgress: React.FC<{ stages: LifecycleStage[] }> = ({ stages }) => {
  return (
    <div className="flex items-center gap-1">
      {stages.map((stage, idx) => (
        <React.Fragment key={stage.stage}>
          <div
            className={`w-6 h-6 rounded-full flex items-center justify-center text-xs font-medium ${
              stage.status === 'COMPLETED' ? 'bg-green-500 text-white' :
              stage.status === 'IN_PROGRESS' ? 'bg-blue-500 text-white animate-pulse' :
              stage.status === 'FAILED' ? 'bg-red-500 text-white' :
              'bg-gray-200 text-gray-500'
            }`}
            title={stage.stage.replace(/_/g, ' ')}
          >
            {stage.status === 'COMPLETED' ? '✓' : 
             stage.status === 'FAILED' ? '✕' : 
             idx + 1}
          </div>
          {idx < stages.length - 1 && (
            <div className={`w-4 h-0.5 ${
              stage.status === 'COMPLETED' ? 'bg-green-500' :
              stage.status === 'FAILED' ? 'bg-red-500' :
              'bg-gray-200'
            }`} />
          )}
        </React.Fragment>
      ))}
    </div>
  );
};

// ============================================================================
// Main Component
// ============================================================================

interface STPIntegrationProps {
  tenantId?: string;
  datasourceId?: string;
}

export const STPIntegration: React.FC<STPIntegrationProps> = ({
  tenantId: _tenantId,
  datasourceId: _datasourceId
}) => {
  // State
  const [trades, setTrades] = useState<Trade[]>([]);
  const [custodians] = useState<CustodianStatus[]>(MOCK_CUSTODIANS);
  const [selectedTrade, setSelectedTrade] = useState<string | null>(null);
  const [activeTab, setActiveTab] = useState<'trades' | 'custodians' | 'exceptions' | 'metrics'>('trades');
  const [filterStatus, setFilterStatus] = useState<string>('ALL');
  const [filterCustodian, setFilterCustodian] = useState<string>('ALL');
  const [searchQuery, setSearchQuery] = useState('');
  const [isRefreshing, setIsRefreshing] = useState(false);

  // Load mock data
  useEffect(() => {
    setTrades(generateMockTrades());
  }, []);

  // Derived state
  const metrics: STPMetrics = useMemo(() => ({
    totalTrades: trades.length,
    stpRate: trades.filter(t => t.exceptions.length === 0 && t.status !== 'FAILED').length / Math.max(trades.length, 1) * 100,
    manualInterventions: trades.filter(t => t.exceptions.length > 0).length,
    avgCycleTime: 1.4,
    pendingSettlements: trades.filter(t => ['PENDING', 'CONFIRMED', 'MATCHED', 'SETTLING'].includes(t.status)).length,
    failedTrades: trades.filter(t => t.status === 'FAILED').length,
    settledToday: trades.filter(t => t.status === 'SETTLED').length
  }), [trades]);

  const filteredTrades = useMemo(() => {
    return trades.filter(trade => {
      if (filterStatus !== 'ALL' && trade.status !== filterStatus) return false;
      if (filterCustodian !== 'ALL' && trade.custodian !== filterCustodian) return false;
      if (searchQuery && !trade.symbol.toLowerCase().includes(searchQuery.toLowerCase()) && 
          !trade.orderId.toLowerCase().includes(searchQuery.toLowerCase()) &&
          !trade.accountName.toLowerCase().includes(searchQuery.toLowerCase())) return false;
      return true;
    });
  }, [trades, filterStatus, filterCustodian, searchQuery]);

  const exceptions = useMemo(() => {
    return trades.flatMap(t => t.exceptions.map(e => ({ ...e, tradeId: t.id, symbol: t.symbol, orderId: t.orderId })));
  }, [trades]);

  // Refresh trades
  const handleRefresh = async () => {
    setIsRefreshing(true);
    await new Promise(resolve => setTimeout(resolve, 1000));
    setTrades(generateMockTrades());
    setIsRefreshing(false);
  };

  // Render trades tab
  const renderTrades = () => (
    <div className="space-y-4">
      {/* Filters */}
      <div className="flex items-center gap-4">
        <div className="relative flex-1 max-w-md">
          <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-gray-400" />
          <input
            type="text"
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
            placeholder="Search by symbol, order ID, or account..."
            className="w-full pl-10 pr-4 py-2 border rounded-lg text-sm"
          />
        </div>
        <div className="flex items-center gap-2">
          <Filter className="w-4 h-4 text-gray-500" />
          <select
            value={filterStatus}
            onChange={(e) => setFilterStatus(e.target.value)}
            className="border rounded-lg px-3 py-2 text-sm"
            title="Filter by trade status"
          >
            <option value="ALL">All Status</option>
            {Object.entries(STATUS_CONFIG).map(([key, config]) => (
              <option key={key} value={key}>{config.label}</option>
            ))}
          </select>
          <select
            value={filterCustodian}
            onChange={(e) => setFilterCustodian(e.target.value)}
            className="border rounded-lg px-3 py-2 text-sm"
            title="Filter by custodian"
          >
            <option value="ALL">All Custodians</option>
            {custodians.map(c => (
              <option key={c.id} value={c.name}>{c.name}</option>
            ))}
          </select>
        </div>
        <button
          onClick={handleRefresh}
          disabled={isRefreshing}
          className="flex items-center gap-2 px-3 py-2 border rounded-lg hover:bg-gray-50 disabled:opacity-50"
        >
          <RefreshCw className={`w-4 h-4 ${isRefreshing ? 'animate-spin' : ''}`} />
          Refresh
        </button>
      </div>

      {/* Trade list */}
      <div className="bg-white rounded-lg border overflow-hidden">
        <table className="w-full">
          <thead className="bg-gray-50 border-b">
            <tr>
              <th className="text-left px-4 py-3 text-xs font-medium text-gray-500">Order</th>
              <th className="text-left px-4 py-3 text-xs font-medium text-gray-500">Symbol</th>
              <th className="text-left px-4 py-3 text-xs font-medium text-gray-500">Side</th>
              <th className="text-right px-4 py-3 text-xs font-medium text-gray-500">Quantity</th>
              <th className="text-right px-4 py-3 text-xs font-medium text-gray-500">Price</th>
              <th className="text-left px-4 py-3 text-xs font-medium text-gray-500">Custodian</th>
              <th className="text-left px-4 py-3 text-xs font-medium text-gray-500">Status</th>
              <th className="text-left px-4 py-3 text-xs font-medium text-gray-500">Lifecycle</th>
            </tr>
          </thead>
          <tbody className="divide-y">
            {filteredTrades.map(trade => (
              <React.Fragment key={trade.id}>
                <tr 
                  className={`hover:bg-gray-50 cursor-pointer ${selectedTrade === trade.id ? 'bg-blue-50' : ''}`}
                  onClick={() => setSelectedTrade(selectedTrade === trade.id ? null : trade.id)}
                >
                  <td className="px-4 py-3">
                    <div className="text-sm font-medium">{trade.orderId}</div>
                    <div className="text-xs text-gray-500">{trade.accountName}</div>
                  </td>
                  <td className="px-4 py-3 font-medium">{trade.symbol}</td>
                  <td className="px-4 py-3">
                    <span className={`px-2 py-0.5 rounded text-xs font-medium ${
                      trade.side === 'BUY' ? 'bg-green-100 text-green-800' : 'bg-red-100 text-red-800'
                    }`}>
                      {trade.side}
                    </span>
                  </td>
                  <td className="px-4 py-3 text-right text-sm">{trade.quantity.toLocaleString()}</td>
                  <td className="px-4 py-3 text-right text-sm">${trade.price.toFixed(2)}</td>
                  <td className="px-4 py-3 text-sm">{trade.custodian}</td>
                  <td className="px-4 py-3">
                    <span className={`px-2 py-0.5 rounded text-xs ${STATUS_CONFIG[trade.status].bgColor} ${STATUS_CONFIG[trade.status].color}`}>
                      {STATUS_CONFIG[trade.status].label}
                    </span>
                  </td>
                  <td className="px-4 py-3">
                    <LifecycleProgress stages={trade.lifecycle} />
                  </td>
                </tr>
                {selectedTrade === trade.id && (
                  <tr>
                    <td colSpan={8} className="bg-gray-50 px-4 py-4">
                      <div className="grid grid-cols-3 gap-6">
                        <div>
                          <h4 className="text-xs font-medium text-gray-500 mb-2">Trade Details</h4>
                          <div className="space-y-1 text-sm">
                            <div className="flex justify-between">
                              <span className="text-gray-500">Trade Date:</span>
                              <span>{trade.tradeDate.toLocaleDateString()}</span>
                            </div>
                            <div className="flex justify-between">
                              <span className="text-gray-500">Settle Date:</span>
                              <span>{trade.expectedSettleDate.toLocaleDateString()}</span>
                            </div>
                            <div className="flex justify-between">
                              <span className="text-gray-500">Total Value:</span>
                              <span>${(trade.quantity * trade.price).toLocaleString(undefined, { maximumFractionDigits: 2 })}</span>
                            </div>
                          </div>
                        </div>
                        <div>
                          <h4 className="text-xs font-medium text-gray-500 mb-2">Lifecycle Stages</h4>
                          <div className="space-y-1 text-xs">
                            {trade.lifecycle.filter(s => s.status !== 'PENDING').slice(0, 5).map(stage => (
                              <div key={stage.stage} className="flex items-center justify-between">
                                <span className="text-gray-600">{stage.stage.replace(/_/g, ' ')}</span>
                                <span className={stage.status === 'COMPLETED' ? 'text-green-600' : stage.status === 'FAILED' ? 'text-red-600' : 'text-blue-600'}>
                                  {stage.status === 'COMPLETED' ? '✓' : stage.status === 'FAILED' ? '✕' : '...'}
                                </span>
                              </div>
                            ))}
                          </div>
                        </div>
                        <div>
                          <h4 className="text-xs font-medium text-gray-500 mb-2">Exceptions</h4>
                          {trade.exceptions.length > 0 ? (
                            <div className="space-y-2">
                              {trade.exceptions.map(exc => (
                                <div key={exc.id} className={`p-2 rounded border text-xs ${EXCEPTION_COLORS[exc.severity]}`}>
                                  <div className="font-medium">{exc.type.replace(/_/g, ' ')}</div>
                                  <div className="text-gray-600 mt-1">{exc.message}</div>
                                </div>
                              ))}
                            </div>
                          ) : (
                            <span className="text-sm text-green-600">No exceptions</span>
                          )}
                        </div>
                      </div>
                    </td>
                  </tr>
                )}
              </React.Fragment>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  );

  // Render custodians tab
  const renderCustodians = () => (
    <div className="grid grid-cols-2 gap-6">
      {custodians.map(custodian => (
        <div key={custodian.id} className="bg-white rounded-lg border p-6">
          <div className="flex items-center justify-between mb-4">
            <div className="flex items-center gap-3">
              <div className="w-10 h-10 bg-gradient-to-br from-blue-500 to-purple-600 rounded-lg flex items-center justify-center">
                <Database className="w-5 h-5 text-white" />
              </div>
              <div>
                <h3 className="font-semibold">{custodian.name}</h3>
                <StatusIndicator status={custodian.status} />
              </div>
            </div>
            <span className="text-xs text-gray-500">
              Last sync: {custodian.lastSync.toLocaleTimeString()}
            </span>
          </div>

          <div className="grid grid-cols-2 gap-4">
            <div className="bg-gray-50 rounded-lg p-3">
              <div className="text-xs text-gray-500">Pending Trades</div>
              <div className="text-xl font-bold">{custodian.pendingTrades}</div>
            </div>
            <div className="bg-green-50 rounded-lg p-3">
              <div className="text-xs text-gray-500">Settled Today</div>
              <div className="text-xl font-bold text-green-600">{custodian.settledToday}</div>
            </div>
            <div className="bg-red-50 rounded-lg p-3">
              <div className="text-xs text-gray-500">Fail Rate</div>
              <div className="text-xl font-bold text-red-600">{custodian.failRate}%</div>
            </div>
            <div className="bg-blue-50 rounded-lg p-3">
              <div className="text-xs text-gray-500">Avg Settle Time</div>
              <div className="text-xl font-bold text-blue-600">{custodian.avgSettleTime}d</div>
            </div>
          </div>

          <div className="mt-4 flex gap-2">
            <button className="flex-1 flex items-center justify-center gap-2 px-3 py-2 border rounded-lg text-sm hover:bg-gray-50">
              <Link2 className="w-4 h-4" />
              Test Connection
            </button>
            <button className="flex-1 flex items-center justify-center gap-2 px-3 py-2 border rounded-lg text-sm hover:bg-gray-50">
              <RefreshCw className="w-4 h-4" />
              Force Sync
            </button>
          </div>
        </div>
      ))}
    </div>
  );

  // Render exceptions tab
  const renderExceptions = () => (
    <div className="space-y-4">
      {exceptions.length === 0 ? (
        <div className="text-center py-12 text-gray-500">
          <CheckCircle className="w-12 h-12 mx-auto mb-4 text-green-300" />
          <p>No active exceptions</p>
          <p className="text-sm">All trades are processing normally</p>
        </div>
      ) : (
        <div className="space-y-3">
          {exceptions.map(exc => (
            <div
              key={exc.id}
              className={`rounded-lg border p-4 ${EXCEPTION_COLORS[exc.severity]}`}
            >
              <div className="flex items-start justify-between">
                <div className="flex items-start gap-3">
                  <AlertTriangle className={`w-5 h-5 ${
                    exc.severity === 'CRITICAL' ? 'text-red-600' :
                    exc.severity === 'HIGH' ? 'text-orange-600' :
                    exc.severity === 'MEDIUM' ? 'text-yellow-600' :
                    'text-blue-600'
                  }`} />
                  <div>
                    <div className="font-medium">{exc.type.replace(/_/g, ' ')}</div>
                    <div className="text-sm mt-1">{exc.message}</div>
                    <div className="flex items-center gap-4 mt-2 text-xs text-gray-500">
                      <span>Trade: {'orderId' in exc ? (exc as unknown as { orderId: string }).orderId : 'N/A'}</span>
                      <span>Symbol: {'symbol' in exc ? (exc as unknown as { symbol: string }).symbol : 'N/A'}</span>
                      <span>{exc.timestamp.toLocaleTimeString()}</span>
                    </div>
                  </div>
                </div>
                <div className="flex items-center gap-2">
                  <span className={`px-2 py-0.5 rounded text-xs ${
                    exc.severity === 'CRITICAL' ? 'bg-red-200' :
                    exc.severity === 'HIGH' ? 'bg-orange-200' :
                    exc.severity === 'MEDIUM' ? 'bg-yellow-200' :
                    'bg-blue-200'
                  }`}>
                    {exc.severity}
                  </span>
                  <button className="px-3 py-1 bg-white border rounded text-sm hover:bg-gray-50">
                    Resolve
                  </button>
                </div>
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  );

  // Render metrics tab
  const renderMetrics = () => (
    <div className="space-y-6">
      {/* Key metrics */}
      <div className="grid grid-cols-4 gap-4">
        <div className="bg-white rounded-lg border p-6">
          <div className="flex items-center justify-between">
            <div>
              <div className="text-xs text-gray-500">STP Rate</div>
              <div className="text-3xl font-bold text-green-600">{metrics.stpRate.toFixed(1)}%</div>
            </div>
            <TrendingUp className="w-8 h-8 text-green-200" />
          </div>
          <div className="mt-2 text-xs text-gray-500">
            {metrics.totalTrades - metrics.manualInterventions} of {metrics.totalTrades} trades
          </div>
        </div>
        <div className="bg-white rounded-lg border p-6">
          <div className="flex items-center justify-between">
            <div>
              <div className="text-xs text-gray-500">Avg Cycle Time</div>
              <div className="text-3xl font-bold text-blue-600">{metrics.avgCycleTime}d</div>
            </div>
            <Clock className="w-8 h-8 text-blue-200" />
          </div>
          <div className="mt-2 text-xs text-gray-500">
            Order to settlement
          </div>
        </div>
        <div className="bg-white rounded-lg border p-6">
          <div className="flex items-center justify-between">
            <div>
              <div className="text-xs text-gray-500">Manual Interventions</div>
              <div className="text-3xl font-bold text-orange-600">{metrics.manualInterventions}</div>
            </div>
            <AlertTriangle className="w-8 h-8 text-orange-200" />
          </div>
          <div className="mt-2 text-xs text-gray-500">
            Requiring attention
          </div>
        </div>
        <div className="bg-white rounded-lg border p-6">
          <div className="flex items-center justify-between">
            <div>
              <div className="text-xs text-gray-500">Settled Today</div>
              <div className="text-3xl font-bold text-purple-600">{metrics.settledToday}</div>
            </div>
            <CheckCircle className="w-8 h-8 text-purple-200" />
          </div>
          <div className="mt-2 text-xs text-gray-500">
            Successfully completed
          </div>
        </div>
      </div>

      {/* Status breakdown */}
      <div className="bg-white rounded-lg border p-6">
        <h3 className="font-semibold mb-4">Trade Status Distribution</h3>
        <div className="flex gap-4">
          {Object.entries(STATUS_CONFIG).map(([status, config]) => {
            const count = trades.filter(t => t.status === status).length;
            const percentage = (count / Math.max(trades.length, 1)) * 100;
            return (
              <div key={status} className="flex-1">
                <div className="flex items-center justify-between mb-1">
                  <span className={`text-xs ${config.color}`}>{config.label}</span>
                  <span className="text-xs font-medium">{count}</span>
                </div>
                <div className="h-2 bg-gray-200 rounded-full overflow-hidden">
                  <StatusBar percentage={percentage} status={status as TradeStatus} />
                </div>
              </div>
            );
          })}
        </div>
      </div>

      {/* Custodian performance */}
      <div className="bg-white rounded-lg border p-6">
        <h3 className="font-semibold mb-4">Custodian Performance</h3>
        <table className="w-full">
          <thead>
            <tr className="text-xs text-gray-500">
              <th className="text-left pb-3">Custodian</th>
              <th className="text-right pb-3">Pending</th>
              <th className="text-right pb-3">Settled</th>
              <th className="text-right pb-3">Fail Rate</th>
              <th className="text-right pb-3">Avg Time</th>
            </tr>
          </thead>
          <tbody className="divide-y">
            {custodians.map(c => (
              <tr key={c.id}>
                <td className="py-3">
                  <div className="flex items-center gap-2">
                    <StatusIndicator status={c.status} />
                    <span className="font-medium">{c.name}</span>
                  </div>
                </td>
                <td className="py-3 text-right">{c.pendingTrades}</td>
                <td className="py-3 text-right text-green-600">{c.settledToday}</td>
                <td className="py-3 text-right">
                  <span className={c.failRate > 1 ? 'text-red-600' : 'text-green-600'}>
                    {c.failRate}%
                  </span>
                </td>
                <td className="py-3 text-right">{c.avgSettleTime}d</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  );

  return (
    <div className="h-full flex flex-col bg-gray-50">
      {/* Header */}
      <div className="bg-white border-b px-6 py-4">
        <div className="flex items-center justify-between">
          <div>
            <h1 className="text-xl font-semibold flex items-center gap-2">
              <Zap className="w-6 h-6 text-blue-600" />
              Straight-Through Processing
            </h1>
            <p className="text-sm text-gray-500 mt-1">
              Trade lifecycle automation and settlement monitoring
            </p>
          </div>
          <div className="flex items-center gap-3">
            <div className={`flex items-center gap-2 px-3 py-1.5 rounded-lg ${
              metrics.stpRate >= 95 ? 'bg-green-100 text-green-700' :
              metrics.stpRate >= 90 ? 'bg-yellow-100 text-yellow-700' :
              'bg-red-100 text-red-700'
            }`}>
              <Activity className="w-4 h-4" />
              <span className="text-sm font-medium">STP Rate: {metrics.stpRate.toFixed(1)}%</span>
            </div>
            <button className="flex items-center gap-2 px-3 py-1.5 border rounded-lg hover:bg-gray-50">
              <Settings className="w-4 h-4" />
              Settings
            </button>
          </div>
        </div>

        {/* Quick stats */}
        <div className="grid grid-cols-5 gap-4 mt-4">
          <div className="bg-gray-50 rounded-lg p-3">
            <div className="text-xs text-gray-500">Total Trades</div>
            <div className="text-xl font-bold">{metrics.totalTrades}</div>
          </div>
          <div className="bg-blue-50 rounded-lg p-3">
            <div className="text-xs text-blue-600">Pending Settlement</div>
            <div className="text-xl font-bold text-blue-900">{metrics.pendingSettlements}</div>
          </div>
          <div className="bg-green-50 rounded-lg p-3">
            <div className="text-xs text-green-600">Settled Today</div>
            <div className="text-xl font-bold text-green-900">{metrics.settledToday}</div>
          </div>
          <div className="bg-red-50 rounded-lg p-3">
            <div className="text-xs text-red-600">Failed</div>
            <div className="text-xl font-bold text-red-900">{metrics.failedTrades}</div>
          </div>
          <div className="bg-orange-50 rounded-lg p-3">
            <div className="text-xs text-orange-600">Exceptions</div>
            <div className="text-xl font-bold text-orange-900">{exceptions.length}</div>
          </div>
        </div>
      </div>

      {/* Tabs */}
      <div className="bg-white border-b px-6">
        <div className="flex gap-6">
          {[
            { id: 'trades' as const, label: 'Trade Lifecycle', icon: Activity },
            { id: 'custodians' as const, label: 'Custodian Status', icon: Database },
            { id: 'exceptions' as const, label: 'Exceptions', icon: AlertTriangle, count: exceptions.length },
            { id: 'metrics' as const, label: 'Metrics', icon: BarChart3 }
          ].map(tab => (
            <button
              key={tab.id}
              onClick={() => setActiveTab(tab.id)}
              className={`flex items-center gap-2 px-1 py-3 border-b-2 transition-colors ${
                activeTab === tab.id 
                  ? 'border-blue-500 text-blue-600' 
                  : 'border-transparent text-gray-500 hover:text-gray-700'
              }`}
            >
              <tab.icon className="w-4 h-4" />
              {tab.label}
              {tab.count !== undefined && tab.count > 0 && (
                <span className={`px-1.5 py-0.5 rounded text-xs ${
                  activeTab === tab.id ? 'bg-blue-100' : 'bg-red-100 text-red-700'
                }`}>
                  {tab.count}
                </span>
              )}
            </button>
          ))}
        </div>
      </div>

      {/* Content */}
      <div className="flex-1 overflow-auto p-6">
        {activeTab === 'trades' && renderTrades()}
        {activeTab === 'custodians' && renderCustodians()}
        {activeTab === 'exceptions' && renderExceptions()}
        {activeTab === 'metrics' && renderMetrics()}
      </div>
    </div>
  );
};

export default STPIntegration;
