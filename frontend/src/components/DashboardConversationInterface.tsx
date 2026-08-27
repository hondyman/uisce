import React, { useState, useRef, useEffect } from 'react';
import { devError } from '../utils/devLogger';
import { Send, BarChart3, LineChart, PieChart, Table, CheckCircle, AlertTriangle, XCircle, Save } from 'lucide-react';
import { useAuthFetch } from '../utils/authFetch';
import { Box, Button, Paper, Typography, TextField, IconButton, useTheme, alpha } from '@mui/material';

interface DashboardVisual {
  id: string;
  type: string;
  title: string;
  description: string;
  querySpec?: {
    metrics: string[];
    dimensions: string[];
    sql?: string;
  };
  config?: {
    chartType: string;
    xAxis?: string;
    yAxis?: string;
    colorBy?: string;
    showLegend?: boolean;
    showGrid?: boolean;
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

interface CommitResponse {
  conversation_id: string;
  state: string;
  title: string;
  description: string;
  dashboard_id: string;
  visual_count: number;
}

interface CreateDashboardResponse {
  dashboard_id: string;
  visual_id: string;
  dashboard_name: string;
}

interface DashboardConversationInterfaceProps {
  tenantId: string;
  datasource: string;
  onDashboardCreated?: (dashboard: { id: string; title: string; dashboard_id: string }) => void;
}

export const DashboardConversationInterface: React.FC<DashboardConversationInterfaceProps> = ({
  tenantId,
  datasource,
  onDashboardCreated
}) => {
  const theme = useTheme();
  const { authFetch } = useAuthFetch();
  const [conversation, setConversation] = useState<DashboardConversation | null>(null);
  const [message, setMessage] = useState('');
  const [isLoading, setIsLoading] = useState(false);
  const [isStarting, setIsStarting] = useState(false);
  const [savingVisualId, setSavingVisualId] = useState<string | null>(null);
  const [savedVisuals, setSavedVisuals] = useState<Set<string>>(new Set());
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
      const response = await authFetch('/api/nl/conversations/start', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({
          user_id: 'current-user',
          tenant_id: tenantId,
          datasource: datasource,
          message: message.trim(),
        }),
      });

      if (!response.ok) {
        throw new Error('Failed to start conversation');
      }

      const data = await response.json();
      setConversation(data);
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
      const response = await authFetch(`/api/nl/conversations/${conversation.id}/message`, {
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

      const data = await response.json();
      setConversation(data);
      setMessage('');
    } catch (error) {
      devError('Error sending message:', error);
    } finally {
      setIsLoading(false);
    }
  };

  const commitDashboard = async () => {
    if (!conversation) return;

    try {
      const response = await authFetch(`/api/nl/conversations/${conversation.id}/commit`, {
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

      const data: CommitResponse = await response.json();
      onDashboardCreated?.({
        id: conversation.id,
        title: data.title,
        dashboard_id: data.dashboard_id,
      });
    } catch (error) {
      devError('Error committing dashboard:', error);
    }
  };

  const saveVisualAsDashboard = async (visualId: string) => {
    if (!conversation) return;

    setSavingVisualId(visualId);
    try {
      const response = await authFetch(
        `/api/nl/conversations/${conversation.id}/visuals/${visualId}/create-dashboard`,
        {
          method: 'POST',
          headers: {
            'Content-Type': 'application/json',
          },
          body: JSON.stringify({}),
        }
      );

      if (!response.ok) {
        throw new Error('Failed to save visual');
      }

      const data: CreateDashboardResponse = await response.json();
      setSavedVisuals(prev => new Set(prev).add(visualId));
      onDashboardCreated?.({
        id: visualId,
        title: data.dashboard_name,
        dashboard_id: data.dashboard_id,
      });
    } catch (error) {
      devError('Error saving visual:', error);
    } finally {
      setSavingVisualId(null);
    }
  };

  const getComplianceIcon = (compliance: DashboardVisual['compliance']) => {
    if (compliance.isCompliant) {
      return <CheckCircle sx={{ width: 16, height: 16, color: 'success.main' }} />;
    }
    if (compliance.riskLevel === 'high') {
      return <XCircle sx={{ width: 16, height: 16, color: 'error.main' }} />;
    }
    return <AlertTriangle sx={{ width: 16, height: 16, color: 'warning.main' }} />;
  };

  const getChartIcon = (type: string) => {
    switch (type) {
      case 'line':
        return <LineChart sx={{ width: 16, height: 16 }} />;
      case 'bar':
        return <BarChart3 sx={{ width: 16, height: 16 }} />;
      case 'pie':
        return <PieChart sx={{ width: 16, height: 16 }} />;
      case 'table':
        return <Table sx={{ width: 16, height: 16 }} />;
      default:
        return <BarChart3 sx={{ width: 16, height: 16 }} />;
    }
  };

  const isDark = theme.palette.mode === 'dark';

  return (
    <Paper sx={{ 
      display: 'flex', 
      flexDirection: 'column', 
      height: '100%', 
      bgcolor: 'background.paper',
      borderRadius: 2,
      overflow: 'hidden'
    }}>
      <Box sx={{ 
        display: 'flex', 
        alignItems: 'center', 
        justifyContent: 'space-between', 
        p: 2, 
        borderBottom: 1, 
        borderColor: 'divider',
        bgcolor: isDark ? 'rgba(255,255,255,0.02)' : 'rgba(0,0,0,0.02)'
      }}>
        <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
          <BarChart3 sx={{ color: 'primary.main', fontSize: 24 }} />
          <Typography variant="h6" sx={{ fontWeight: 600 }}>
            {conversation ? conversation.title : 'Dashboard Builder'}
          </Typography>
        </Box>
        {conversation && (
          <Box sx={{ display: 'flex', alignItems: 'center', gap: 2 }}>
            <Box sx={{ display: 'flex', alignItems: 'center', gap: 0.5 }}>
              {getComplianceIcon({
                isCompliant: conversation.compliance.overallCompliant,
                riskLevel: conversation.compliance.highRiskCount > 0 ? 'high' : 'low',
                violations: [],
              })}
              <Typography variant="body2" color="text.secondary">
                {conversation.compliance.compliantCount}/{conversation.compliance.visualCount} compliant
              </Typography>
            </Box>
            <Button 
              variant="contained" 
              onClick={commitDashboard}
              sx={{ textTransform: 'none' }}
            >
              Save Dashboard
            </Button>
          </Box>
        )}
      </Box>

      <Box sx={{ flex: 1, display: 'flex', flexDirection: 'column', overflow: 'hidden' }}>
        <Box sx={{ flex: 1, overflowY: 'auto', p: 2, display: 'flex', flexDirection: 'column', gap: 2 }}>
          {conversation?.messages.map((msg) => (
            <Box 
              key={msg.id} 
              sx={{ 
                display: 'flex', 
                justifyContent: msg.type === 'user' ? 'flex-end' : 'flex-start' 
              }}
            >
              <Box
                sx={{
                  maxWidth: { xs: '100%', lg: 400 },
                  px: 2,
                  py: 1,
                  borderRadius: 2,
                  bgcolor: msg.type === 'user' ? 'primary.main' : isDark ? 'grey.900' : 'grey.100',
                  color: msg.type === 'user' ? 'white' : 'text.primary',
                }}
              >
                <Typography variant="body2">{msg.content}</Typography>
                <Typography variant="caption" sx={{ opacity: 0.7, display: 'block', mt: 0.5 }}>
                  {new Date(msg.timestamp).toLocaleTimeString()}
                </Typography>
              </Box>
            </Box>
          ))}
          <div ref={messagesEndRef} />
        </Box>

        {conversation && conversation.visuals.length > 0 && (
          <Box sx={{ borderTop: 1, borderColor: 'divider', p: 2, bgcolor: isDark ? 'rgba(255,255,255,0.02)' : 'grey.50' }}>
            <Typography variant="subtitle2" sx={{ mb: 1.5, fontWeight: 600 }}>
              Current Visualizations
            </Typography>
            <Box sx={{ display: 'grid', gridTemplateColumns: { xs: '1fr', md: 'repeat(2, 1fr)', lg: 'repeat(3, 1fr)' }, gap: 2 }}>
              {conversation.visuals.map((visual) => (
                <Paper 
                  key={visual.id} 
                  sx={{ p: 1.5, border: 1, borderColor: 'divider' }}
                >
                  <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', mb: 1 }}>
                    <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                      {getChartIcon(visual.type)}
                      <Typography variant="body2" sx={{ fontWeight: 500 }}>
                        {visual.title}
                      </Typography>
                    </Box>
                    {getComplianceIcon(visual.compliance)}
                  </Box>
                  {visual.compliance.violations.length > 0 && (
                    <Box sx={{ mt: 1 }}>
                      <Typography variant="caption" color="error.main">
                        {visual.compliance.violations[0].message}
                      </Typography>
                      {visual.compliance.violations[0].suggestion && (
                        <Typography variant="caption" color="info.main" sx={{ display: 'block', mt: 0.5 }}>
                          {visual.compliance.violations[0].suggestion}
                        </Typography>
                      )}
                    </Box>
                  )}
                  <Button
                    size="small"
                    variant={savedVisuals.has(visual.id) ? "outlined" : "contained"}
                    color="success"
                    disabled={savingVisualId === visual.id || savedVisuals.has(visual.id)}
                    onClick={() => saveVisualAsDashboard(visual.id)}
                    startIcon={<Save sx={{ fontSize: 14 }} />}
                    sx={{ 
                      mt: 1.5, 
                      width: '100%',
                      textTransform: 'none',
                      bgcolor: savedVisuals.has(visual.id) ? 'transparent' : undefined,
                      '&:hover': savedVisuals.has(visual.id) ? {
                        bgcolor: alpha(theme.palette.success.main, 0.1)
                      } : undefined
                    }}
                  >
                    {savingVisualId === visual.id
                      ? 'Saving...'
                      : savedVisuals.has(visual.id)
                      ? 'Saved'
                      : 'Save as Dashboard'}
                  </Button>
                </Paper>
              ))}
            </Box>
          </Box>
        )}

        <Box sx={{ borderTop: 1, borderColor: 'divider', p: 2 }}>
          <Box sx={{ display: 'flex', gap: 1 }}>
            <TextField
              fullWidth
              size="small"
              placeholder={
                conversation
                  ? "Describe what you'd like to add or modify..."
                  : "Describe the dashboard you want to create..."
              }
              value={message}
              onChange={(e) => setMessage(e.target.value)}
              onKeyPress={(e) => {
                if (e.key === 'Enter' && !isLoading && !isStarting) {
                  conversation ? sendMessage() : startConversation();
                }
              }}
              disabled={isLoading || isStarting}
              sx={{
                '& .MuiOutlinedInput-root': {
                  bgcolor: isDark ? 'rgba(255,255,255,0.05)' : 'white',
                }
              }}
            />
            <IconButton
              color="primary"
              onClick={conversation ? sendMessage : startConversation}
              disabled={!message.trim() || isLoading || isStarting}
              sx={{
                bgcolor: 'primary.main',
                color: 'white',
                '&:hover': { bgcolor: 'primary.dark' },
                '&:disabled': { bgcolor: 'grey.400' }
              }}
            >
              {isLoading || isStarting ? (
                <Box sx={{ 
                  width: 20, 
                  height: 20, 
                  border: '2px solid white', 
                  borderTopColor: 'transparent', 
                  borderRadius: '50%', 
                  animation: 'spin 1s linear infinite',
                  '@keyframes spin': {
                    '0%': { transform: 'rotate(0deg)' },
                    '100%': { transform: 'rotate(360deg)' }
                  }
                }} />
              ) : (
                <Send sx={{ fontSize: 18 }} />
              )}
            </IconButton>
          </Box>
          {!conversation && (
            <Typography variant="caption" color="text.secondary" sx={{ mt: 1, display: 'block' }}>
              Start by describing what kind of dashboard you want to create, e.g., "Show me sales performance by region over time"
            </Typography>
          )}
        </Box>
      </Box>
    </Paper>
  );
};
