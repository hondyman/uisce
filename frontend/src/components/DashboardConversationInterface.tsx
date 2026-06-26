import React, { useState, useRef, useEffect } from 'react';
import { devError } from '../utils/devLogger';
import { Send, BarChart3, LineChart, PieChart, Table, CheckCircle, AlertTriangle, XCircle } from 'lucide-react';
import { useAuthFetch } from '../utils/authFetch';

interface DashboardVisual {
  id: string;
  type: string;
  title: string;
  description: string;
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
}

interface ConversationMessage {
  id: string;
  type: 'user' | 'assistant';
  content: string;
  timestamp: string;
}

interface DashboardConversation {
  id: string;
  state: string;
  title: string;
  description: string;
  visuals: DashboardVisual[];
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
  messages: ConversationMessage[];
}

interface DashboardConversationInterfaceProps {
  tenantId: string;
  datasource: string;
  onDashboardCreated?: (dashboard: DashboardConversation) => void;
}

export const DashboardConversationInterface: React.FC<DashboardConversationInterfaceProps> = ({
  tenantId,
  datasource,
  onDashboardCreated
}) => {
  const { authFetch } = useAuthFetch();
  const [conversation, setConversation] = useState<DashboardConversation | null>(null);
  const [message, setMessage] = useState('');
  const [isLoading, setIsLoading] = useState(false);
  const [isStarting, setIsStarting] = useState(false);
  const messagesEndRef = useRef<HTMLDivElement>(null);

  const scrollToBottom = () => {
    messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' });
  };

  useEffect(() => {
    scrollToBottom();
  }, [conversation?.messages]);

  const startConversation = async () => {
    if (!message.trim()) return;

    setIsStarting(true);
    try {
  const response = await authFetch('/api/dashboards/conversation/start', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({
          user_id: 'current-user', // This should come from auth context
          tenant_id: tenantId,
          datasource: datasource,
          message: message.trim(),
        }),
      });

      if (!response.ok) {
        throw new Error('Failed to start conversation');
      }

  const newConversation = (response && (response as any).data !== undefined) ? (response as any).data : await (response as any).json?.();
      setConversation(newConversation);
      setMessage('');
    } catch (error) {
      devError('Error starting conversation:', error);
    } finally {
      setIsStarting(false);
    }
  };

  const sendMessage = async () => {
    if (!message.trim() || !conversation) return;

    setIsLoading(true);
    try {
  const response = await authFetch(`/api/dashboards/conversation/${conversation.id}/message`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({
          message: message.trim(),
        }),
      });

      if (!response.ok) {
        throw new Error('Failed to send message');
      }

  const updatedConversation = (response && (response as any).data !== undefined) ? (response as any).data : await (response as any).json?.();
      setConversation(updatedConversation);
      setMessage('');
    } catch (error) {
      devError('Error sending message:', error);
      // Handle error (show toast, etc.)
    } finally {
      setIsLoading(false);
    }
  };

  const commitDashboard = async () => {
    if (!conversation) return;

    try {
  const response = await authFetch(`/api/dashboards/conversation/${conversation.id}/commit`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({
          title: conversation.title || 'My Dashboard',
          description: conversation.description || 'Dashboard created via conversation',
        }),
      });

      if (!response.ok) {
        throw new Error('Failed to commit dashboard');
      }

  const committedDashboard = (response && (response as any).data !== undefined) ? (response as any).data : await (response as any).json?.();
      onDashboardCreated?.(committedDashboard);
    } catch (error) {
      devError('Error committing dashboard:', error);
      // Handle error (show toast, etc.)
    }
  };

  const getComplianceIcon = (compliance: DashboardVisual['compliance']) => {
    if (compliance.isCompliant) {
      return <CheckCircle className="w-4 h-4 text-green-500" />;
    }
    if (compliance.riskLevel === 'high') {
      return <XCircle className="w-4 h-4 text-red-500" />;
    }
    return <AlertTriangle className="w-4 h-4 text-yellow-500" />;
  };

  const getChartIcon = (type: string) => {
    switch (type) {
      case 'line':
        return <LineChart className="w-4 h-4" />;
      case 'bar':
        return <BarChart3 className="w-4 h-4" />;
      case 'pie':
        return <PieChart className="w-4 h-4" />;
      case 'table':
        return <Table className="w-4 h-4" />;
      default:
        return <BarChart3 className="w-4 h-4" />;
    }
  };

  return (
    <div className="flex flex-col h-full bg-white rounded-lg shadow-lg">
      {/* Header */}
      <div className="flex items-center justify-between p-4 border-b">
        <div className="flex items-center space-x-2">
          <BarChart3 className="w-6 h-6 text-blue-600" />
          <h2 className="text-lg font-semibold text-gray-900">
            {conversation ? conversation.title : 'Dashboard Builder'}
          </h2>
        </div>
        {conversation && (
          <div className="flex items-center space-x-2">
            <div className="flex items-center space-x-1">
              {getComplianceIcon({
                isCompliant: conversation.compliance.overallCompliant,
                riskLevel: conversation.compliance.highRiskCount > 0 ? 'high' : 'low',
                violations: [],
              })}
              <span className="text-sm text-gray-600">
                {conversation.compliance.compliantCount}/{conversation.compliance.visualCount} compliant
              </span>
            </div>
            <button
              onClick={commitDashboard}
              className="px-4 py-2 bg-blue-600 text-white rounded-md hover:bg-blue-700 transition-colors"
            >
              Save Dashboard
            </button>
          </div>
        )}
      </div>

      {/* Conversation Area */}
      <div className="flex-1 flex flex-col min-h-0">
        {/* Messages */}
        <div className="flex-1 overflow-y-auto p-4 space-y-4">
          {conversation?.messages.map((msg) => (
            <div
              key={msg.id}
              className={`flex ${msg.type === 'user' ? 'justify-end' : 'justify-start'}`}
            >
              <div
                className={`max-w-xs lg:max-w-md px-4 py-2 rounded-lg ${
                  msg.type === 'user'
                    ? 'bg-blue-600 text-white'
                    : 'bg-gray-100 text-gray-900'
                }`}
              >
                <p className="text-sm">{msg.content}</p>
                <p className="text-xs opacity-70 mt-1">
                  {new Date(msg.timestamp).toLocaleTimeString()}
                </p>
              </div>
            </div>
          ))}
          <div ref={messagesEndRef} />
        </div>

        {/* Visual Preview */}
        {conversation && conversation.visuals.length > 0 && (
          <div className="border-t p-4 bg-gray-50">
            <h3 className="text-sm font-medium text-gray-900 mb-3">Current Visualizations</h3>
            <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-3">
              {conversation.visuals.map((visual) => (
                <div
                  key={visual.id}
                  className="bg-white p-3 rounded-md border shadow-sm"
                >
                  <div className="flex items-center justify-between mb-2">
                    <div className="flex items-center space-x-2">
                      {getChartIcon(visual.type)}
                      <span className="text-sm font-medium text-gray-900">
                        {visual.title}
                      </span>
                    </div>
                    {getComplianceIcon(visual.compliance)}
                  </div>
                  {visual.compliance.violations.length > 0 && (
                    <div className="mt-2">
                      <p className="text-xs text-red-600">
                        {visual.compliance.violations[0].message}
                      </p>
                      {visual.compliance.violations[0].suggestion && (
                        <p className="text-xs text-blue-600 mt-1">
                          💡 {visual.compliance.violations[0].suggestion}
                        </p>
                      )}
                    </div>
                  )}
                </div>
              ))}
            </div>
          </div>
        )}

        {/* Input Area */}
        <div className="border-t p-4">
          <div className="flex space-x-2">
            <input
              type="text"
              value={message}
              onChange={(e) => setMessage(e.target.value)}
              onKeyPress={(e) => {
                if (e.key === 'Enter' && !isLoading && !isStarting) {
                  conversation ? sendMessage() : startConversation();
                }
              }}
              placeholder={
                conversation
                  ? "Describe what you'd like to add or modify..."
                  : "Describe the dashboard you want to create..."
              }
              className="flex-1 px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
              disabled={isLoading || isStarting}
            />
            <button
              onClick={conversation ? sendMessage : startConversation}
              disabled={!message.trim() || isLoading || isStarting}
              className="px-4 py-2 bg-blue-600 text-white rounded-md hover:bg-blue-700 disabled:bg-gray-400 disabled:cursor-not-allowed transition-colors"
            >
              {isLoading || isStarting ? (
                <div className="w-4 h-4 border-2 border-white border-t-transparent rounded-full animate-spin" />
              ) : (
                <Send className="w-4 h-4" />
              )}
            </button>
          </div>
          {!conversation && (
            <p className="text-xs text-gray-500 mt-2">
              Start by describing what kind of dashboard you want to create, e.g., "Show me sales performance by region over time"
            </p>
          )}
        </div>
      </div>
    </div>
  );
};
