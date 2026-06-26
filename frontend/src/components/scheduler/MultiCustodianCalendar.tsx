/**
 * MultiCustodianCalendar.tsx
 * 
 * Multi-Custodian Settlement Calendar Harmonization:
 * - Unified view of settlement calendars across custodians
 * - Holiday and market closure coordination
 * - Settlement date prediction and conflict detection
 * - Cash flow forecasting based on settlement timing
 */

import React, { useState, useMemo } from 'react';
import {
  Calendar,
  Clock,
  AlertTriangle,
  Building2,
  DollarSign,
  ChevronLeft,
  ChevronRight,
  Filter,
  Settings,
  CheckCircle,
  XCircle,
  Info
} from 'lucide-react';

// ============================================================================
// Types
// ============================================================================

interface Custodian {
  id: string;
  name: string;
  shortCode: string;
  color: string;
  settlementCycle: 'T+1' | 'T+2' | 'T+3';
  supportedMarkets: string[];
  timezone: string;
}

interface CalendarEvent {
  id: string;
  date: string;
  type: EventType;
  custodians: string[];
  markets: string[];
  description: string;
  impact: 'LOW' | 'MEDIUM' | 'HIGH';
}

type EventType = 'MARKET_HOLIDAY' | 'CUSTODIAN_CLOSED' | 'EARLY_CLOSE' | 'SETTLEMENT_CUTOFF' | 'SPECIAL_SETTLEMENT';

interface PendingSettlement {
  id: string;
  tradeId: string;
  symbol: string;
  side: 'BUY' | 'SELL';
  quantity: number;
  amount: number;
  custodian: string;
  tradeDate: Date;
  originalSettleDate: Date;
  adjustedSettleDate: Date;
  hasConflict: boolean;
  conflictReason?: string;
}

interface CashFlowForecast {
  date: string;
  inflows: { custodian: string; amount: number }[];
  outflows: { custodian: string; amount: number }[];
  netFlow: number;
}

// ============================================================================
// Constants
// ============================================================================

const CUSTODIANS: Custodian[] = [
  { id: 'schwab', name: 'Charles Schwab', shortCode: 'SCH', color: 'bg-blue-500', settlementCycle: 'T+1', supportedMarkets: ['NYSE', 'NASDAQ', 'AMEX'], timezone: 'America/New_York' },
  { id: 'fidelity', name: 'Fidelity', shortCode: 'FID', color: 'bg-green-500', settlementCycle: 'T+1', supportedMarkets: ['NYSE', 'NASDAQ', 'AMEX', 'OTC'], timezone: 'America/New_York' },
  { id: 'pershing', name: 'Pershing', shortCode: 'PER', color: 'bg-purple-500', settlementCycle: 'T+1', supportedMarkets: ['NYSE', 'NASDAQ', 'AMEX', 'LSE'], timezone: 'America/New_York' },
  { id: 'td', name: 'TD Ameritrade', shortCode: 'TDA', color: 'bg-orange-500', settlementCycle: 'T+1', supportedMarkets: ['NYSE', 'NASDAQ'], timezone: 'America/New_York' }
];

const EVENT_TYPES: Record<EventType, { label: string; color: string; icon: React.FC<{ className?: string }> }> = {
  MARKET_HOLIDAY: { label: 'Market Holiday', color: 'bg-red-100 text-red-800', icon: XCircle },
  CUSTODIAN_CLOSED: { label: 'Custodian Closed', color: 'bg-orange-100 text-orange-800', icon: Building2 },
  EARLY_CLOSE: { label: 'Early Close', color: 'bg-yellow-100 text-yellow-800', icon: Clock },
  SETTLEMENT_CUTOFF: { label: 'Settlement Cutoff', color: 'bg-blue-100 text-blue-800', icon: AlertTriangle },
  SPECIAL_SETTLEMENT: { label: 'Special Settlement', color: 'bg-purple-100 text-purple-800', icon: Info }
};

// Generate mock calendar events
const generateMockEvents = (): CalendarEvent[] => {
  const events: CalendarEvent[] = [];
  const today = new Date();
  
  // Market holidays
  const holidays = [
    { offset: 15, name: 'Presidents Day', markets: ['NYSE', 'NASDAQ'] },
    { offset: 45, name: 'Good Friday', markets: ['NYSE', 'NASDAQ', 'LSE'] },
    { offset: 75, name: 'Memorial Day', markets: ['NYSE', 'NASDAQ'] }
  ];
  
  holidays.forEach((h, idx) => {
    const date = new Date(today.getTime() + h.offset * 24 * 60 * 60 * 1000);
    events.push({
      id: `holiday-${idx}`,
      date: date.toISOString().split('T')[0],
      type: 'MARKET_HOLIDAY',
      custodians: CUSTODIANS.map(c => c.id),
      markets: h.markets,
      description: h.name,
      impact: 'HIGH'
    });
  });

  // Early closes
  for (let i = 0; i < 3; i++) {
    const date = new Date(today.getTime() + (20 + i * 30) * 24 * 60 * 60 * 1000);
    events.push({
      id: `early-${i}`,
      date: date.toISOString().split('T')[0],
      type: 'EARLY_CLOSE',
      custodians: CUSTODIANS.map(c => c.id),
      markets: ['NYSE', 'NASDAQ'],
      description: 'Early market close at 1:00 PM ET',
      impact: 'MEDIUM'
    });
  }

  return events;
};

// Generate mock pending settlements
const generateMockSettlements = (): PendingSettlement[] => {
  const settlements: PendingSettlement[] = [];
  const symbols = ['AAPL', 'GOOGL', 'MSFT', 'AMZN', 'TSLA', 'NVDA'];
  const today = new Date();

  for (let i = 0; i < 15; i++) {
    const tradeDate = new Date(today.getTime() - Math.floor(Math.random() * 3) * 24 * 60 * 60 * 1000);
    const custodian = CUSTODIANS[Math.floor(Math.random() * CUSTODIANS.length)];
    const originalSettleDate = new Date(tradeDate.getTime() + 24 * 60 * 60 * 1000); // T+1
    const hasConflict = Math.random() > 0.8;
    
    settlements.push({
      id: `settle-${i}`,
      tradeId: `TRD-${100000 + i}`,
      symbol: symbols[Math.floor(Math.random() * symbols.length)],
      side: Math.random() > 0.5 ? 'BUY' : 'SELL',
      quantity: Math.floor(Math.random() * 1000) + 100,
      amount: Math.floor(Math.random() * 100000) + 10000,
      custodian: custodian.id,
      tradeDate,
      originalSettleDate,
      adjustedSettleDate: hasConflict 
        ? new Date(originalSettleDate.getTime() + 24 * 60 * 60 * 1000)
        : originalSettleDate,
      hasConflict,
      conflictReason: hasConflict ? 'Market holiday - settlement moved to next business day' : undefined
    });
  }

  return settlements;
};

// ============================================================================
// Helper Functions
// ============================================================================

const getDaysInMonth = (year: number, month: number): number => {
  return new Date(year, month + 1, 0).getDate();
};

const getFirstDayOfMonth = (year: number, month: number): number => {
  return new Date(year, month, 1).getDay();
};

// ============================================================================
// Main Component
// ============================================================================

interface MultiCustodianCalendarProps {
  tenantId?: string;
  datasourceId?: string;
}

export const MultiCustodianCalendar: React.FC<MultiCustodianCalendarProps> = ({
  tenantId: _tenantId,
  datasourceId: _datasourceId
}) => {
  // State
  const [currentDate, setCurrentDate] = useState(new Date());
  const [events] = useState<CalendarEvent[]>(generateMockEvents);
  const [settlements] = useState<PendingSettlement[]>(generateMockSettlements);
  const [selectedDate, setSelectedDate] = useState<string | null>(null);
  const [activeTab, setActiveTab] = useState<'calendar' | 'settlements' | 'cashflow'>('calendar');
  const [filterCustodian, setFilterCustodian] = useState<string>('ALL');

  // Navigation
  const navigateMonth = (direction: number) => {
    const newDate = new Date(currentDate);
    newDate.setMonth(newDate.getMonth() + direction);
    setCurrentDate(newDate);
  };

  // Derived state
  const currentYear = currentDate.getFullYear();
  const currentMonth = currentDate.getMonth();
  const daysInMonth = getDaysInMonth(currentYear, currentMonth);
  const firstDay = getFirstDayOfMonth(currentYear, currentMonth);

  const calendarDays = useMemo(() => {
    const days: { date: string; day: number; isCurrentMonth: boolean; events: CalendarEvent[] }[] = [];
    
    // Previous month days
    const prevMonthDays = getDaysInMonth(currentYear, currentMonth - 1);
    for (let i = firstDay - 1; i >= 0; i--) {
      const day = prevMonthDays - i;
      const date = new Date(currentYear, currentMonth - 1, day);
      days.push({
        date: date.toISOString().split('T')[0],
        day,
        isCurrentMonth: false,
        events: events.filter(e => e.date === date.toISOString().split('T')[0])
      });
    }
    
    // Current month days
    for (let day = 1; day <= daysInMonth; day++) {
      const date = new Date(currentYear, currentMonth, day);
      days.push({
        date: date.toISOString().split('T')[0],
        day,
        isCurrentMonth: true,
        events: events.filter(e => e.date === date.toISOString().split('T')[0])
      });
    }
    
    // Next month days
    const remainingDays = 42 - days.length;
    for (let day = 1; day <= remainingDays; day++) {
      const date = new Date(currentYear, currentMonth + 1, day);
      days.push({
        date: date.toISOString().split('T')[0],
        day,
        isCurrentMonth: false,
        events: events.filter(e => e.date === date.toISOString().split('T')[0])
      });
    }
    
    return days;
  }, [currentYear, currentMonth, daysInMonth, firstDay, events]);

  const filteredSettlements = useMemo(() => {
    return settlements.filter(s => filterCustodian === 'ALL' || s.custodian === filterCustodian);
  }, [settlements, filterCustodian]);

  const cashFlowForecast = useMemo((): CashFlowForecast[] => {
    const forecast: CashFlowForecast[] = [];
    const dateMap = new Map<string, { inflows: { custodian: string; amount: number }[]; outflows: { custodian: string; amount: number }[] }>();
    
    settlements.forEach(s => {
      const dateStr = s.adjustedSettleDate.toISOString().split('T')[0];
      if (!dateMap.has(dateStr)) {
        dateMap.set(dateStr, { inflows: [], outflows: [] });
      }
      const dayData = dateMap.get(dateStr)!;
      
      if (s.side === 'SELL') {
        dayData.inflows.push({ custodian: s.custodian, amount: s.amount });
      } else {
        dayData.outflows.push({ custodian: s.custodian, amount: s.amount });
      }
    });
    
    dateMap.forEach((data, date) => {
      const totalInflow = data.inflows.reduce((sum, i) => sum + i.amount, 0);
      const totalOutflow = data.outflows.reduce((sum, o) => sum + o.amount, 0);
      forecast.push({
        date,
        inflows: data.inflows,
        outflows: data.outflows,
        netFlow: totalInflow - totalOutflow
      });
    });
    
    return forecast.sort((a, b) => new Date(a.date).getTime() - new Date(b.date).getTime());
  }, [settlements]);

  const metrics = useMemo(() => ({
    totalPendingSettlements: settlements.length,
    conflictCount: settlements.filter(s => s.hasConflict).length,
    upcomingEvents: events.filter(e => new Date(e.date) > new Date()).length,
    totalSettlementValue: settlements.reduce((sum, s) => sum + s.amount, 0)
  }), [settlements, events]);

  // Render calendar tab
  const renderCalendar = () => {
    const today = new Date().toISOString().split('T')[0];
    
    return (
      <div className="space-y-4">
        {/* Calendar header */}
        <div className="bg-white rounded-lg border p-4">
          <div className="flex items-center justify-between mb-4">
            <button 
              onClick={() => navigateMonth(-1)}
              className="p-2 hover:bg-gray-100 rounded"
              aria-label="Previous month"
            >
              <ChevronLeft className="w-5 h-5" />
            </button>
            <h2 className="text-lg font-semibold">
              {currentDate.toLocaleDateString('en-US', { month: 'long', year: 'numeric' })}
            </h2>
            <button 
              onClick={() => navigateMonth(1)}
              className="p-2 hover:bg-gray-100 rounded"
              aria-label="Next month"
            >
              <ChevronRight className="w-5 h-5" />
            </button>
          </div>

          {/* Custodian legend */}
          <div className="flex items-center gap-4 mb-4 pb-4 border-b">
            {CUSTODIANS.map(c => (
              <div key={c.id} className="flex items-center gap-2">
                <span className={`w-3 h-3 rounded-full ${c.color}`} />
                <span className="text-sm">{c.shortCode}</span>
              </div>
            ))}
          </div>

          {/* Calendar grid */}
          <div className="grid grid-cols-7 gap-1">
            {['Sun', 'Mon', 'Tue', 'Wed', 'Thu', 'Fri', 'Sat'].map(day => (
              <div key={day} className="text-center text-xs font-medium text-gray-500 py-2">
                {day}
              </div>
            ))}
            {calendarDays.map((day, idx) => (
              <div
                key={idx}
                className={`min-h-24 p-1 border rounded cursor-pointer transition-colors ${
                  !day.isCurrentMonth ? 'bg-gray-50 text-gray-400' :
                  day.date === today ? 'bg-blue-50 border-blue-200' :
                  day.date === selectedDate ? 'bg-purple-50 border-purple-200' :
                  'hover:bg-gray-50'
                }`}
                onClick={() => setSelectedDate(day.date === selectedDate ? null : day.date)}
                onKeyDown={(e) => e.key === 'Enter' && setSelectedDate(day.date === selectedDate ? null : day.date)}
                tabIndex={0}
                role="button"
              >
                <div className="text-sm font-medium mb-1">{day.day}</div>
                {day.events.slice(0, 2).map(event => {
                  const eventConfig = EVENT_TYPES[event.type];
                  return (
                    <div 
                      key={event.id} 
                      className={`text-xs px-1 py-0.5 rounded mb-0.5 truncate ${eventConfig.color}`}
                      title={event.description}
                    >
                      {event.description}
                    </div>
                  );
                })}
                {day.events.length > 2 && (
                  <div className="text-xs text-gray-500">+{day.events.length - 2} more</div>
                )}
              </div>
            ))}
          </div>
        </div>

        {/* Event type legend */}
        <div className="bg-white rounded-lg border p-4">
          <h3 className="text-sm font-medium mb-3">Event Types</h3>
          <div className="flex flex-wrap gap-3">
            {Object.entries(EVENT_TYPES).map(([key, config]) => (
              <div key={key} className={`flex items-center gap-2 px-2 py-1 rounded ${config.color}`}>
                <config.icon className="w-3 h-3" />
                <span className="text-xs">{config.label}</span>
              </div>
            ))}
          </div>
        </div>

        {/* Selected date details */}
        {selectedDate && (
          <div className="bg-white rounded-lg border p-4">
            <h3 className="font-medium mb-3">
              Events for {new Date(selectedDate).toLocaleDateString('en-US', { weekday: 'long', month: 'long', day: 'numeric' })}
            </h3>
            {(() => {
              const dayEvents = events.filter(e => e.date === selectedDate);
              const daySettlements = settlements.filter(s => 
                s.adjustedSettleDate.toISOString().split('T')[0] === selectedDate
              );
              
              if (dayEvents.length === 0 && daySettlements.length === 0) {
                return <p className="text-sm text-gray-500">No events or settlements scheduled</p>;
              }
              
              return (
                <div className="space-y-4">
                  {dayEvents.length > 0 && (
                    <div>
                      <h4 className="text-xs font-medium text-gray-500 mb-2">Calendar Events</h4>
                      <div className="space-y-2">
                        {dayEvents.map(event => {
                          const eventConfig = EVENT_TYPES[event.type];
                          return (
                            <div key={event.id} className={`p-3 rounded ${eventConfig.color}`}>
                              <div className="flex items-center gap-2">
                                <eventConfig.icon className="w-4 h-4" />
                                <span className="font-medium">{event.description}</span>
                              </div>
                              <div className="flex items-center gap-2 mt-1 text-xs">
                                <span>Markets: {event.markets.join(', ')}</span>
                                <span>•</span>
                                <span>Impact: {event.impact}</span>
                              </div>
                            </div>
                          );
                        })}
                      </div>
                    </div>
                  )}
                  {daySettlements.length > 0 && (
                    <div>
                      <h4 className="text-xs font-medium text-gray-500 mb-2">Pending Settlements ({daySettlements.length})</h4>
                      <div className="space-y-1">
                        {daySettlements.slice(0, 5).map(s => (
                          <div key={s.id} className="flex items-center justify-between p-2 bg-gray-50 rounded text-sm">
                            <div className="flex items-center gap-3">
                              <span className={`px-2 py-0.5 rounded text-xs ${s.side === 'BUY' ? 'bg-green-100 text-green-800' : 'bg-red-100 text-red-800'}`}>
                                {s.side}
                              </span>
                              <span className="font-medium">{s.symbol}</span>
                              <span className="text-gray-500">{s.quantity} shares</span>
                            </div>
                            <div className="flex items-center gap-2">
                              <span className="font-medium">${s.amount.toLocaleString()}</span>
                              {s.hasConflict && (
                                <span title={s.conflictReason}>
                                  <AlertTriangle className="w-4 h-4 text-orange-500" />
                                </span>
                              )}
                            </div>
                          </div>
                        ))}
                      </div>
                    </div>
                  )}
                </div>
              );
            })()}
          </div>
        )}
      </div>
    );
  };

  // Render settlements tab
  const renderSettlements = () => (
    <div className="space-y-4">
      {/* Filters */}
      <div className="flex items-center gap-4">
        <div className="flex items-center gap-2">
          <Filter className="w-4 h-4 text-gray-500" />
          <select
            value={filterCustodian}
            onChange={(e) => setFilterCustodian(e.target.value)}
            className="border rounded-lg px-3 py-2 text-sm"
            title="Filter by custodian"
          >
            <option value="ALL">All Custodians</option>
            {CUSTODIANS.map(c => (
              <option key={c.id} value={c.id}>{c.name}</option>
            ))}
          </select>
        </div>
      </div>

      {/* Settlements table */}
      <div className="bg-white rounded-lg border overflow-hidden">
        <table className="w-full">
          <thead className="bg-gray-50 border-b">
            <tr>
              <th className="text-left px-4 py-3 text-xs font-medium text-gray-500">Trade ID</th>
              <th className="text-left px-4 py-3 text-xs font-medium text-gray-500">Symbol</th>
              <th className="text-left px-4 py-3 text-xs font-medium text-gray-500">Side</th>
              <th className="text-right px-4 py-3 text-xs font-medium text-gray-500">Quantity</th>
              <th className="text-right px-4 py-3 text-xs font-medium text-gray-500">Amount</th>
              <th className="text-left px-4 py-3 text-xs font-medium text-gray-500">Custodian</th>
              <th className="text-left px-4 py-3 text-xs font-medium text-gray-500">Trade Date</th>
              <th className="text-left px-4 py-3 text-xs font-medium text-gray-500">Settle Date</th>
              <th className="text-left px-4 py-3 text-xs font-medium text-gray-500">Status</th>
            </tr>
          </thead>
          <tbody className="divide-y">
            {filteredSettlements.map(settlement => {
              const custodian = CUSTODIANS.find(c => c.id === settlement.custodian);
              return (
                <tr key={settlement.id} className="hover:bg-gray-50">
                  <td className="px-4 py-3 text-sm font-medium">{settlement.tradeId}</td>
                  <td className="px-4 py-3 text-sm">{settlement.symbol}</td>
                  <td className="px-4 py-3">
                    <span className={`px-2 py-0.5 rounded text-xs ${
                      settlement.side === 'BUY' ? 'bg-green-100 text-green-800' : 'bg-red-100 text-red-800'
                    }`}>
                      {settlement.side}
                    </span>
                  </td>
                  <td className="px-4 py-3 text-sm text-right">{settlement.quantity.toLocaleString()}</td>
                  <td className="px-4 py-3 text-sm text-right">${settlement.amount.toLocaleString()}</td>
                  <td className="px-4 py-3">
                    <div className="flex items-center gap-2">
                      <span className={`w-2 h-2 rounded-full ${custodian?.color || 'bg-gray-400'}`} />
                      <span className="text-sm">{custodian?.shortCode || settlement.custodian}</span>
                    </div>
                  </td>
                  <td className="px-4 py-3 text-sm">{settlement.tradeDate.toLocaleDateString()}</td>
                  <td className="px-4 py-3 text-sm">
                    {settlement.adjustedSettleDate.toLocaleDateString()}
                    {settlement.hasConflict && (
                      <span className="ml-1 text-xs text-orange-600">(adjusted)</span>
                    )}
                  </td>
                  <td className="px-4 py-3">
                    {settlement.hasConflict ? (
                      <div className="flex items-center gap-1 text-orange-600">
                        <AlertTriangle className="w-4 h-4" />
                        <span className="text-xs">Adjusted</span>
                      </div>
                    ) : (
                      <div className="flex items-center gap-1 text-green-600">
                        <CheckCircle className="w-4 h-4" />
                        <span className="text-xs">On Track</span>
                      </div>
                    )}
                  </td>
                </tr>
              );
            })}
          </tbody>
        </table>
      </div>
    </div>
  );

  // Render cash flow tab
  const renderCashFlow = () => (
    <div className="space-y-6">
      {/* Cash flow summary */}
      <div className="grid grid-cols-3 gap-4">
        <div className="bg-green-50 rounded-lg border border-green-200 p-4">
          <div className="text-xs text-green-600">Total Inflows</div>
          <div className="text-2xl font-bold text-green-700">
            ${cashFlowForecast.reduce((sum, d) => sum + d.inflows.reduce((s, i) => s + i.amount, 0), 0).toLocaleString()}
          </div>
        </div>
        <div className="bg-red-50 rounded-lg border border-red-200 p-4">
          <div className="text-xs text-red-600">Total Outflows</div>
          <div className="text-2xl font-bold text-red-700">
            ${cashFlowForecast.reduce((sum, d) => sum + d.outflows.reduce((s, o) => s + o.amount, 0), 0).toLocaleString()}
          </div>
        </div>
        <div className="bg-blue-50 rounded-lg border border-blue-200 p-4">
          <div className="text-xs text-blue-600">Net Position</div>
          <div className="text-2xl font-bold text-blue-700">
            ${cashFlowForecast.reduce((sum, d) => sum + d.netFlow, 0).toLocaleString()}
          </div>
        </div>
      </div>

      {/* Daily breakdown */}
      <div className="bg-white rounded-lg border overflow-hidden">
        <div className="px-4 py-3 border-b bg-gray-50">
          <h3 className="font-medium">Daily Cash Flow Forecast</h3>
        </div>
        <table className="w-full">
          <thead className="bg-gray-50 border-b">
            <tr>
              <th className="text-left px-4 py-3 text-xs font-medium text-gray-500">Settlement Date</th>
              <th className="text-right px-4 py-3 text-xs font-medium text-gray-500">Inflows</th>
              <th className="text-right px-4 py-3 text-xs font-medium text-gray-500">Outflows</th>
              <th className="text-right px-4 py-3 text-xs font-medium text-gray-500">Net Flow</th>
              <th className="text-left px-4 py-3 text-xs font-medium text-gray-500">Custodian Breakdown</th>
            </tr>
          </thead>
          <tbody className="divide-y">
            {cashFlowForecast.map(day => (
              <tr key={day.date} className="hover:bg-gray-50">
                <td className="px-4 py-3 text-sm font-medium">
                  {new Date(day.date).toLocaleDateString('en-US', { weekday: 'short', month: 'short', day: 'numeric' })}
                </td>
                <td className="px-4 py-3 text-sm text-right text-green-600">
                  +${day.inflows.reduce((sum, i) => sum + i.amount, 0).toLocaleString()}
                </td>
                <td className="px-4 py-3 text-sm text-right text-red-600">
                  -${day.outflows.reduce((sum, o) => sum + o.amount, 0).toLocaleString()}
                </td>
                <td className={`px-4 py-3 text-sm text-right font-medium ${day.netFlow >= 0 ? 'text-green-600' : 'text-red-600'}`}>
                  {day.netFlow >= 0 ? '+' : ''}${day.netFlow.toLocaleString()}
                </td>
                <td className="px-4 py-3">
                  <div className="flex items-center gap-2">
                    {[...new Set([...day.inflows.map(i => i.custodian), ...day.outflows.map(o => o.custodian)])].map(custId => {
                      const custodian = CUSTODIANS.find(c => c.id === custId);
                      return (
                        <span key={custId} className={`w-4 h-4 rounded-full ${custodian?.color || 'bg-gray-400'}`} title={custodian?.name} />
                      );
                    })}
                  </div>
                </td>
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
              <Calendar className="w-6 h-6 text-blue-600" />
              Multi-Custodian Settlement Calendar
            </h1>
            <p className="text-sm text-gray-500 mt-1">
              Unified settlement calendar across all custodians
            </p>
          </div>
          <button className="flex items-center gap-2 px-3 py-1.5 border rounded-lg hover:bg-gray-50">
            <Settings className="w-4 h-4" />
            Settings
          </button>
        </div>

        {/* Stats bar */}
        <div className="grid grid-cols-4 gap-4 mt-4">
          <div className="bg-gray-50 rounded-lg p-3">
            <div className="flex items-center justify-between">
              <span className="text-xs text-gray-500">Pending Settlements</span>
              <Clock className="w-4 h-4 text-gray-400" />
            </div>
            <div className="text-xl font-bold">{metrics.totalPendingSettlements}</div>
          </div>
          <div className="bg-orange-50 rounded-lg p-3">
            <div className="flex items-center justify-between">
              <span className="text-xs text-orange-600">Date Conflicts</span>
              <AlertTriangle className="w-4 h-4 text-orange-400" />
            </div>
            <div className="text-xl font-bold text-orange-700">{metrics.conflictCount}</div>
          </div>
          <div className="bg-blue-50 rounded-lg p-3">
            <div className="flex items-center justify-between">
              <span className="text-xs text-blue-600">Calendar Events</span>
              <Calendar className="w-4 h-4 text-blue-400" />
            </div>
            <div className="text-xl font-bold text-blue-700">{metrics.upcomingEvents}</div>
          </div>
          <div className="bg-green-50 rounded-lg p-3">
            <div className="flex items-center justify-between">
              <span className="text-xs text-green-600">Settlement Value</span>
              <DollarSign className="w-4 h-4 text-green-400" />
            </div>
            <div className="text-xl font-bold text-green-700">${(metrics.totalSettlementValue / 1000000).toFixed(2)}M</div>
          </div>
        </div>
      </div>

      {/* Tabs */}
      <div className="bg-white border-b px-6">
        <div className="flex gap-6">
          {[
            { id: 'calendar' as const, label: 'Calendar View', icon: Calendar },
            { id: 'settlements' as const, label: 'Pending Settlements', icon: Clock },
            { id: 'cashflow' as const, label: 'Cash Flow Forecast', icon: DollarSign }
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
            </button>
          ))}
        </div>
      </div>

      {/* Content */}
      <div className="flex-1 overflow-auto p-6">
        {activeTab === 'calendar' && renderCalendar()}
        {activeTab === 'settlements' && renderSettlements()}
        {activeTab === 'cashflow' && renderCashFlow()}
      </div>
    </div>
  );
};

export default MultiCustodianCalendar;
