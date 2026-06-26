import React, { useState, useEffect } from 'react';
import { TrendingUp, TrendingDown, Target, MessageSquare, Calendar, Bell, Settings, DollarSign, PieChart } from 'lucide-react';
import { Responsive, WidthProvider, Layout } from 'react-grid-layout';
import 'react-grid-layout/css/styles.css';
import 'react-resizable/css/styles.css';

const ResponsiveGridLayout = WidthProvider(Responsive);

interface DashboardWidget {
  widgetId: string;
  widgetType: string;
  position: number;
  size: string;
  config: any;
  isVisible: boolean;
}

interface PortfolioSummary {
  totalValue: number;
  dayChange: number;
  dayChangePercent: number;
  ytdReturn: number;
  allocation: Record<string, number>;
}

interface Goal {
  goalId: string;
  goalName: string;
  targetAmount: number;
  currentAmount: number;
  progressPercentage: number;
  monthsRemaining: number;
  onTrack: boolean;
}

interface DashboardSummary {
  unreadMessages: number;
  unreadNotifications: number;
  pendingActions: number;
  activeGoals: number;
}

export const ClientDashboard: React.FC = () => {
  const [widgets, setWidgets] = useState<DashboardWidget[]>([]);
  const [portfolioSummary, setPortfolioSummary] = useState<PortfolioSummary | null>(null);
  const [goals, setGoals] = useState<Goal[]>([]);
  const [summary, setSummary] = useState<DashboardSummary | null>(null);
  const [isCustomizing, setIsCustomizing] = useState(false);
  const [layouts, setLayouts] = useState<{ [key: string]: Layout[] }>({});

  useEffect(() => {
    fetchDashboardData();
  }, []);

  const fetchDashboardData = async () => {
    try {
      // Fetch widgets configuration
      const widgetsRes = await fetch('/api/dashboard/widgets');
      const widgetsData = await widgetsRes.json();
      setWidgets(widgetsData);

      // Generate grid layout from widgets
      const gridLayouts = generateLayouts(widgetsData);
      setLayouts(gridLayouts);

      // Fetch portfolio summary
      const portfolioRes = await fetch('/api/dashboard/portfolio-summary');
      const portfolioData = await portfolioRes.json();
      setPortfolioSummary(portfolioData);

      // Fetch goals
      const goalsRes = await fetch('/api/dashboard/goals');
      const goalsData = await goalsRes.json();
      setGoals(goalsData);

      // Fetch summary stats
      const summaryRes = await fetch('/api/dashboard/summary');
      const summaryData = await summaryRes.json();
      setSummary(summaryData);
    } catch (error) {
      console.error('Failed to fetch dashboard data:', error);
    }
  };

  const generateLayouts = (widgets: DashboardWidget[]): { [key: string]: Layout[] } => {
    const layout: Layout[] = widgets
      .filter(w => w.isVisible)
      .sort((a, b) => a.position - b.position)
      .map((widget, index) => {
        const sizeMap = {
          SMALL: { w: 3, h: 2 },
          MEDIUM: { w: 6, h: 3 },
          LARGE: { w: 6, h: 4 },
          FULL_WIDTH: { w: 12, h: 3 },
        };

        const size = sizeMap[widget.size as keyof typeof sizeMap] || sizeMap.MEDIUM;

        return {
          i: widget.widgetId,
          x: (index * 6) % 12,
          y: Math.floor(index / 2) * 3,
          ...size,
          minW: 3,
          minH: 2,
        };
      });

    return { lg: layout, md: layout, sm: layout, xs: layout };
  };

  const handleLayoutChange = (layout: Layout[], layouts: { [key: string]: Layout[] }) => {
    if (!isCustomizing) return;

    setLayouts(layouts);

    // Save layout to backend
    const updatedWidgets = widgets.map(widget => {
      const layoutItem = layout.find(l => l.i === widget.widgetId);
      if (!layoutItem) return widget;

      return {
        ...widget,
        position: layoutItem.y * 12 + layoutItem.x,
        size: layoutItem.w === 12 ? 'FULL_WIDTH' : layoutItem.w >= 6 ? 'LARGE' : layoutItem.w >= 4 ? 'MEDIUM' : 'SMALL',
      };
    });

    saveLayout(updatedWidgets);
  };

  const saveLayout = async (updatedWidgets: DashboardWidget[]) => {
    try {
      await fetch('/api/dashboard/widgets/layout', {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ widgets: updatedWidgets }),
      });
    } catch (error) {
      console.error('Failed to save layout:', error);
    }
  };

  const formatCurrency = (value: number) => {
    return new Intl.NumberFormat('en-US', {
      style: 'currency',
      currency: 'USD',
      minimumFractionDigits: 0,
      maximumFractionDigits: 0,
    }).format(value);
  };

  const formatPercent = (value: number) => {
    return `${value >= 0 ? '+' : ''}${value.toFixed(2)}%`;
  };

  return (
    <div className="min-h-screen bg-gradient-to-br from-slate-50 to-blue-50">
      {/* Header */}
      <div className="bg-white shadow-sm border-b">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-4">
          <div className="flex justify-between items-center">
            <div>
              <h1 className="text-3xl font-bold text-gray-900">Welcome back, John</h1>
              <p className="text-gray-600 mt-1">Here's your financial overview</p>
            </div>

            <button
              onClick={() => setIsCustomizing(!isCustomizing)}
              className={`flex items-center gap-2 px-4 py-2 rounded-lg transition-colors ${
                isCustomizing
                  ? 'bg-indigo-600 text-white'
                  : 'bg-gray-100 text-gray-700 hover:bg-gray-200'
              }`}
            >
              <Settings className="w-5 h-5" />
              {isCustomizing ? 'Done Customizing' : 'Customize Dashboard'}
            </button>
          </div>
        </div>
      </div>

      {/* Quick Stats Bar */}
      {summary && (
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-6">
          <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
            <QuickStatCard
              icon={<MessageSquare className="w-6 h-6" />}
              label="Unread Messages"
              value={summary.unreadMessages}
              color="blue"
            />
            <QuickStatCard
              icon={<Bell className="w-6 h-6" />}
              label="Notifications"
              value={summary.unreadNotifications}
              color="purple"
            />
            <QuickStatCard
              icon={<Target className="w-6 h-6" />}
              label="Active Goals"
              value={summary.activeGoals}
              color="green"
            />
            <QuickStatCard
              icon={<TrendingUp className="w-6 h-6" />}
              label="Action Items"
              value={summary.pendingActions}
              color="orange"
            />
          </div>
        </div>
      )}

      {/* Widgets Grid */}
      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 pb-8">
        <ResponsiveGridLayout
          className="layout"
          layouts={layouts}
          breakpoints={{ lg: 1200, md: 996, sm: 768, xs: 480, xxs: 0 }}
          cols={{ lg: 12, md: 10, sm: 6, xs: 4, xxs: 2 }}
          rowHeight={100}
          onLayoutChange={handleLayoutChange}
          isDraggable={isCustomizing}
          isResizable={isCustomizing}
          compactType="vertical"
        >
          {widgets.filter(w => w.isVisible).map(widget => (
            <div key={widget.widgetId} className="widget-container">
              {renderWidget(widget, {
                portfolioSummary,
                goals,
                formatCurrency,
                formatPercent,
              })}
            </div>
          ))}
        </ResponsiveGridLayout>
      </div>
    </div>
  );
};

const renderWidget = (
  widget: DashboardWidget,
  data: {
    portfolioSummary: PortfolioSummary | null;
    goals: Goal[];
    formatCurrency: (v: number) => string;
    formatPercent: (v: number) => string;
  }
) => {
  switch (widget.widgetType) {
    case 'PORTFOLIO_SUMMARY':
      return <PortfolioSummaryWidget {...data} />;
    case 'GOALS_PROGRESS':
      return <GoalsProgressWidget goals={data.goals} formatCurrency={data.formatCurrency} />;
    case 'RECENT_TRANSACTIONS':
      return <RecentTransactionsWidget />;
    case 'MESSAGES_INBOX':
      return <MessagesInboxWidget />;
    case 'UPCOMING_MEETINGS':
      return <UpcomingMeetingsWidget />;
    case 'RECOMMENDED_ACTIONS':
      return <RecommendedActionsWidget />;
    case 'NET_WORTH_TREND':
      return <NetWorthTrendWidget />;
    case 'ASSET_ALLOCATION':
      return <AssetAllocationWidget summary={data.portfolioSummary} />;
    default:
      return <div className="widget-card">Unknown Widget</div>;
  }
};

// Widget Components
const PortfolioSummaryWidget: React.FC<any> = ({ portfolioSummary, formatCurrency, formatPercent }) => {
  if (!portfolioSummary) return <WidgetSkeleton />;

  const isPositive = portfolioSummary.dayChange >= 0;

  return (
    <div className="widget-card bg-gradient-to-br from-indigo-500 to-purple-600 text-white">
      <div className="flex justify-between items-start mb-4">
        <div>
          <p className="text-indigo-100 text-sm font-medium">Total Portfolio Value</p>
          <h2 className="text-4xl font-bold mt-2">{formatCurrency(portfolioSummary.totalValue)}</h2>
        </div>
        <DollarSign className="w-10 h-10 text-indigo-200 opacity-50" />
      </div>

      <div className="flex items-center gap-4 mt-4">
        <div className="flex items-center gap-2">
          {isPositive ? (
            <TrendingUp className="w-5 h-5 text-green-300" />
          ) : (
            <TrendingDown className="w-5 h-5 text-red-300" />
          )}
          <span className={`text-lg font-semibold ${isPositive ? 'text-green-300' : 'text-red-300'}`}>
            {formatCurrency(Math.abs(portfolioSummary.dayChange))}
          </span>
          <span className="text-indigo-100">today</span>
        </div>

        <div className="ml-auto">
          <p className="text-indigo-100 text-sm">YTD Return</p>
          <p className="text-xl font-bold text-green-300">{formatPercent(portfolioSummary.ytdReturn)}</p>
        </div>
      </div>
    </div>
  );
};

const GoalsProgressWidget: React.FC<{ goals: Goal[]; formatCurrency: (v: number) => string }> = ({
  goals,
  formatCurrency,
}) => {
  return (
    <div className="widget-card">
      <div className="flex justify-between items-center mb-4">
        <h3 className="text-lg font-semibold text-gray-900">Financial Goals</h3>
        <Target className="w-6 h-6 text-indigo-600" />
      </div>

      <div className="space-y-4">
        {goals.slice(0, 3).map(goal => (
          <div key={goal.goalId}>
            <div className="flex justify-between items-center mb-2">
              <span className="text-sm font-medium text-gray-700">{goal.goalName}</span>
              <span className="text-sm text-gray-600">{Math.round(goal.progressPercentage)}%</span>
            </div>

            <div className="w-full bg-gray-200 rounded-full h-3 overflow-hidden">
              <div
                className={`h-full rounded-full transition-all duration-500 ${
                  goal.onTrack ? 'bg-gradient-to-r from-green-400 to-green-600' : 'bg-gradient-to-r from-orange-400 to-orange-600'
                }`}
                style={{ width: `${Math.min(goal.progressPercentage, 100)}%` }}
              />
            </div>

            <div className="flex justify-between items-center mt-1">
              <span className="text-xs text-gray-500">
                {formatCurrency(goal.currentAmount)} of {formatCurrency(goal.targetAmount)}
              </span>
              <span className="text-xs text-gray-500">
                {goal.monthsRemaining} months left
                {goal.onTrack ? ' ✓' : ' ⚠️'}
              </span>
            </div>
          </div>
        ))}
      </div>

      <button className="w-full mt-4 text-indigo-600 hover:text-indigo-700 text-sm font-medium">
        View All Goals →
      </button>
    </div>
  );
};

const RecentTransactionsWidget: React.FC = () => {
  return (
    <div className="widget-card">
      <h3 className="text-lg font-semibold text-gray-900 mb-4">Recent Activity</h3>
      <div className="space-y-3">
        {/* Transaction items */}
        <p className="text-sm text-gray-500">No recent transactions</p>
      </div>
    </div>
  );
};

const MessagesInboxWidget: React.FC = () => {
  return (
    <div className="widget-card">
      <div className="flex justify-between items-center mb-4">
        <h3 className="text-lg font-semibold text-gray-900">Messages</h3>
        <MessageSquare className="w-6 h-6 text-indigo-600" />
      </div>
      <p className="text-sm text-gray-500">No unread messages</p>
    </div>
  );
};

const UpcomingMeetingsWidget: React.FC = () => {
  return (
    <div className="widget-card">
      <div className="flex justify-between items-center mb-4">
        <h3 className="text-lg font-semibold text-gray-900">Upcoming Meetings</h3>
        <Calendar className="w-6 h-6 text-indigo-600" />
      </div>
      <p className="text-sm text-gray-500">No upcoming meetings</p>
    </div>
  );
};

const RecommendedActionsWidget: React.FC = () => {
  return (
    <div className="widget-card bg-gradient-to-br from-amber-50 to-orange-100 border-orange-200">
      <h3 className="text-lg font-semibold text-gray-900 mb-4">Recommended Actions</h3>
      <p className="text-sm text-gray-600">No pending actions at this time</p>
    </div>
  );
};

const NetWorthTrendWidget: React.FC = () => {
  return (
    <div className="widget-card">
      <h3 className="text-lg font-semibold text-gray-900 mb-4">Net Worth Trend</h3>
      <div className="h-48 flex items-center justify-center bg-gray-50 rounded">
        <p className="text-sm text-gray-500">Chart visualization</p>
      </div>
    </div>
  );
};

const AssetAllocationWidget: React.FC<{ summary: PortfolioSummary | null }> = ({ summary }) => {
  return (
    <div className="widget-card">
      <div className="flex justify-between items-center mb-4">
        <h3 className="text-lg font-semibold text-gray-900">Asset Allocation</h3>
        <PieChart className="w-6 h-6 text-indigo-600" />
      </div>

      {summary?.allocation && (
        <div className="space-y-2">
          {Object.entries(summary.allocation).map(([asset, percent]) => (
            <div key={asset}>
              <div className="flex justify-between text-sm mb-1">
                <span className="text-gray-700">{asset}</span>
                <span className="font-medium">{percent.toFixed(1)}%</span>
              </div>
              <div className="w-full bg-gray-200 rounded-full h-2">
                <div
                  className="bg-indigo-600 h-2 rounded-full"
                  style={{ width: `${percent}%` }}
                />
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  );
};

// Utility Components
const QuickStatCard: React.FC<{
  icon: React.ReactNode;
  label: string;
  value: number;
  color: string;
}> = ({ icon, label, value, color }) => {
  const colorMap: Record<string, string> = {
    blue: 'from-blue-500 to-blue-600',
    purple: 'from-purple-500 to-purple-600',
    green: 'from-green-500 to-green-600',
    orange: 'from-orange-500 to-orange-600',
  };

  return (
    <div className="bg-white rounded-xl shadow-sm p-4 border border-gray-100">
      <div className="flex items-center gap-3">
        <div className={`p-3 rounded-lg bg-gradient-to-br ${colorMap[color]} text-white`}>
          {icon}
        </div>
        <div>
          <p className="text-2xl font-bold text-gray-900">{value}</p>
          <p className="text-sm text-gray-600">{label}</p>
        </div>
      </div>
    </div>
  );
};

const WidgetSkeleton: React.FC = () => {
  return (
    <div className="widget-card animate-pulse">
      <div className="h-4 bg-gray-200 rounded w-1/4 mb-4"></div>
      <div className="h-8 bg-gray-200 rounded w-3/4 mb-2"></div>
      <div className="h-4 bg-gray-200 rounded w-1/2"></div>
    </div>
  );
};

// Add global styles
const styles = `
  .widget-card {
    background: white;
    border-radius: 1rem;
    padding: 1.5rem;
    height: 100%;
    box-shadow: 0 1px 3px 0 rgb(0 0 0 / 0.1);
    border: 1px solid rgb(243 244 246);
    transition: all 0.2s;
  }

  .widget-card:hover {
    box-shadow: 0 4px 6px -1px rgb(0 0 0 / 0.1);
    transform: translateY(-2px);
  }

  .widget-container {
    touch-action: none;
  }

  .react-grid-item.react-grid-placeholder {
    background: rgb(99 102 241 / 0.2);
    border-radius: 1rem;
    border: 2px dashed rgb(99 102 241);
  }
`;
