import { useState } from 'react';
import { Split } from 'lucide-react';
import { devLog } from '../utils/devLogger';
import { DashboardConversationInterface } from './DashboardConversationInterface';
import { DashboardPreview } from './DashboardPreview';

interface DashboardConversation {
  id: string;
  state: string;
  title: string;
  description: string;
  visuals: Array<{
    id: string;
    type: string;
    title: string;
    description: string;
    querySpec: {
      metrics: string[];
      dimensions: string[];
      sql: string;
    };
    config: {
      chartType: string;
      xAxis?: string;
      yAxis?: string;
      colorBy?: string;
      showLegend: boolean;
      showGrid: boolean;
    };
    compliance: {
      isCompliant: boolean;
      riskLevel: string;
      violations: Array<{
        policyId: string;
        severity: string;
        message: string;
        suggestion?: string;
      }>;
    };
    position: {
      x: number;
      y: number;
      width: number;
      height: number;
    };
  }>;
  layout: {
    type: string;
    columns: number;
    rowHeight: number;
  };
  compliance: {
    overallCompliant: boolean;
    visualCount: number;
    compliantCount: number;
    highRiskCount: number;
  };
}

interface ConversationalDashboardPageProps {
  tenantId: string;
  datasource: string;
}

export const ConversationalDashboardPage: React.FC<ConversationalDashboardPageProps> = ({
  tenantId,
  datasource,
}) => {
  const [currentDashboard, setCurrentDashboard] = useState<DashboardConversation | null>(null);
  const [viewMode, setViewMode] = useState<'split' | 'conversation' | 'preview'>('split');

  const handleDashboardCreated = (dashboard: DashboardConversation) => {
    setCurrentDashboard(dashboard);
    // Here you could navigate to a saved dashboard view or show a success message
    devLog('Dashboard created:', dashboard);
  };

  return (
    <div className="min-h-screen bg-gray-50">
      {/* Header */}
      <div className="bg-white shadow-sm border-b">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
          <div className="flex items-center justify-between h-16">
            <div className="flex items-center space-x-4">
              <h1 className="text-xl font-semibold text-gray-900">
                Conversational Dashboard Builder
              </h1>
              <span className="text-sm text-gray-500">
                Build dashboards through natural conversation
              </span>
            </div>

            {/* View Mode Toggle */}
            <div className="flex items-center space-x-2">
              <button
                onClick={() => setViewMode('split')}
                className={`px-3 py-1 rounded-md text-sm font-medium ${
                  viewMode === 'split'
                    ? 'bg-blue-100 text-blue-700'
                    : 'text-gray-500 hover:text-gray-700'
                }`}
              >
                <Split className="w-4 h-4 inline mr-1" />
                Split
              </button>
              <button
                onClick={() => setViewMode('conversation')}
                className={`px-3 py-1 rounded-md text-sm font-medium ${
                  viewMode === 'conversation'
                    ? 'bg-blue-100 text-blue-700'
                    : 'text-gray-500 hover:text-gray-700'
                }`}
              >
                Chat
              </button>
              <button
                onClick={() => setViewMode('preview')}
                className={`px-3 py-1 rounded-md text-sm font-medium ${
                  viewMode === 'preview'
                    ? 'bg-blue-100 text-blue-700'
                    : 'text-gray-500 hover:text-gray-700'
                }`}
              >
                Preview
              </button>
            </div>
          </div>
        </div>
      </div>

      {/* Main Content */}
      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-6">
        {viewMode === 'split' && (
          <div className="grid grid-cols-1 lg:grid-cols-2 gap-6 h-[calc(100vh-12rem)]">
            {/* Conversation Panel */}
            <div className="h-full">
              <DashboardConversationInterface
                tenantId={tenantId}
                datasource={datasource}
                onDashboardCreated={handleDashboardCreated}
              />
            </div>

            {/* Preview Panel */}
            <div className="h-full">
              {currentDashboard ? (
                <DashboardPreview
                  visuals={currentDashboard.visuals}
                  layout={currentDashboard.layout}
                  className="h-full"
                />
              ) : (
                <div className="h-full bg-white rounded-lg shadow-lg flex items-center justify-center">
                  <div className="text-center text-gray-500">
                    <Split className="w-16 h-16 mx-auto mb-4" />
                    <h3 className="text-lg font-medium mb-2">Dashboard Preview</h3>
                    <p className="text-sm">
                      Start a conversation to see your dashboard preview here
                    </p>
                  </div>
                </div>
              )}
            </div>
          </div>
        )}

        {viewMode === 'conversation' && (
          <div className="max-w-4xl mx-auto h-[calc(100vh-12rem)]">
            <DashboardConversationInterface
              tenantId={tenantId}
              datasource={datasource}
              onDashboardCreated={handleDashboardCreated}
            />
          </div>
        )}

        {viewMode === 'preview' && (
          <div className="h-[calc(100vh-12rem)]">
            {currentDashboard ? (
              <DashboardPreview
                visuals={currentDashboard.visuals}
                layout={currentDashboard.layout}
                className="h-full"
              />
            ) : (
              <div className="h-full bg-white rounded-lg shadow-lg flex items-center justify-center">
                <div className="text-center text-gray-500">
                  <Split className="w-16 h-16 mx-auto mb-4" />
                  <h3 className="text-lg font-medium mb-2">No Dashboard to Preview</h3>
                  <p className="text-sm">
                    Switch to conversation mode to start building a dashboard
                  </p>
                </div>
              </div>
            )}
          </div>
        )}
      </div>

      {/* Footer */}
      <div className="bg-white border-t mt-8">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-4">
          <div className="flex items-center justify-between text-sm text-gray-500">
            <div className="flex items-center space-x-4">
              <span>🔒 All queries are governance-compliant</span>
              <span>💬 Natural language dashboard creation</span>
              <span>📊 Real-time preview and validation</span>
            </div>
            <div className="flex items-center space-x-2">
              <span>Status:</span>
              {currentDashboard ? (
                <span className="text-green-600 font-medium">
                  Active Conversation ({currentDashboard.visuals.length} visuals)
                </span>
              ) : (
                <span className="text-gray-400">Ready to start</span>
              )}
            </div>
          </div>
        </div>
      </div>
    </div>
  );
};
