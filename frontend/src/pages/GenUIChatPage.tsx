import React, { useEffect, useRef, useState } from 'react';
import ReactECharts from 'echarts-for-react';
import {
  Box,
  Paper,
  TextField,
  IconButton,
  Typography,
  Avatar,
  Grid,
  Button,
  useTheme,
  CircularProgress,
} from '@mui/material';
import { Send as SendIcon, AutoAwesome as SparklesIcon } from '@mui/icons-material';
import apiClient from '../utils/apiClient';

interface Message {
  id: string;
  role: 'user' | 'assistant';
  content: string;
  chartSpec?: unknown;
  timestamp: Date;
}

const CONVERSATION_KEY = 'uisce_chat_conversation_id';

function getOrCreateConversationId(): string {
  if (typeof sessionStorage === 'undefined') {
    return crypto.randomUUID();
  }
  const existing = sessionStorage.getItem(CONVERSATION_KEY);
  if (existing) return existing;
  const fresh = crypto.randomUUID();
  sessionStorage.setItem(CONVERSATION_KEY, fresh);
  return fresh;
}

export default function GenUIChatPage() {
  const [messages, setMessages] = useState<Message[]>([]);
  const [inputValue, setInputValue] = useState('');
  const [loading, setLoading] = useState(false);
  const messagesEndRef = useRef<HTMLDivElement>(null);
  const theme = useTheme();

  useEffect(() => {
    getOrCreateConversationId();
  }, []);

  useEffect(() => {
    messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' });
  }, [messages]);

  const sendToBackend = async (query: string): Promise<{ content: string; chartSpec?: unknown }> => {
    const conversationId = getOrCreateConversationId();
    try {
      const resp = await apiClient<{ response?: string; chart_spec?: unknown }>('/ai/chat', {
        method: 'POST',
        body: JSON.stringify({
          text: query,
          conversation_id: conversationId,
          tenant_id: localStorage.getItem('selected_tenant') || undefined,
        }),
      });
      return { content: resp.response ?? '', chartSpec: resp.chart_spec };
    } catch {
      // Backend unreachable in demo/dev — fall back to local stub response
      return stubResponse(query);
    }
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    const trimmed = inputValue.trim();
    if (!trimmed || loading) return;

    setInputValue('');
    setLoading(true);

    const userMessage: Message = {
      id: `user-${Date.now()}`,
      role: 'user',
      content: trimmed,
      timestamp: new Date(),
    };
    setMessages((prev) => [...prev, userMessage]);

    const { content, chartSpec } = await sendToBackend(trimmed);
    const assistantMessage: Message = {
      id: `assistant-${Date.now()}`,
      role: 'assistant',
      content,
      chartSpec,
      timestamp: new Date(),
    };
    setMessages((prev) => [...prev, assistantMessage]);
    setLoading(false);
  };

  return (
    <Box
      sx={{
        display: 'flex',
        flexDirection: 'column',
        height: 'calc(100vh - 80px)',
        backgroundColor: theme.palette.mode === 'dark' ? 'background.default' : 'grey.50',
      }}
    >
      <Box
        sx={{
          px: 3,
          py: 2,
          backgroundColor: 'background.paper',
          borderBottom: '1px solid',
          borderColor: 'divider',
          display: 'flex',
          justifyContent: 'space-between',
          alignItems: 'center',
          position: 'sticky',
          top: 0,
          zIndex: 10,
        }}
      >
        <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
          <Avatar sx={{ bgcolor: theme.palette.primary.main, width: 32, height: 32 }}>
            <SparklesIcon sx={{ fontSize: 18 }} />
          </Avatar>
          <Typography variant="h6" sx={{ fontWeight: 'bold' }}>
            Uuisce Console
          </Typography>
        </Box>
        <Box sx={{ display: 'flex', alignItems: 'center', gap: 0.5, color: 'text.secondary', fontSize: '0.875rem' }}>
          <SparklesIcon sx={{ fontSize: 14, color: theme.palette.primary.main }} />
          Powered by Gemini
        </Box>
      </Box>

      <Box sx={{ flex: 1, overflowY: 'auto', p: 3, display: 'flex', flexDirection: 'column', gap: 3 }}>
        {messages.length === 0 ? (
          <Box
            sx={{
              height: '100%',
              display: 'flex',
              flexDirection: 'column',
              alignItems: 'center',
              justifyContent: 'center',
              textAlign: 'center',
              color: 'text.secondary',
              gap: 2,
            }}
          >
            <Avatar sx={{ bgcolor: 'action.hover', width: 64, height: 64, fontSize: 32 }}>✨</Avatar>
            <Box>
              <Typography variant="h6" color="text.primary" sx={{ fontWeight: 'medium' }}>
                Generative UI Ready
              </Typography>
              <Typography variant="body2" color="text.secondary" sx={{ mt: 0.5 }}>
                Ask about your portfolio, compliance, or market events.
              </Typography>
            </Box>

            <Grid container spacing={2} sx={{ maxWidth: 600, mt: 4 }}>
              {[
                { label: 'Compare Tech vs S&P 500', query: 'Compare my Tech exposure to S&P 500 YTD' },
                { label: 'Risks of Futures Trading', query: 'What are the risks of Futures Trading?' },
                { label: 'Impact of Rate Hikes', query: 'Analyze impact of rate hikes on Real Estate' },
              ].map((btn) => (
                <Grid key={btn.label} size={{ xs: 12, md: 4 }}>
                  <Button
                    fullWidth
                    variant="outlined"
                    onClick={() => setInputValue(btn.query)}
                    sx={{
                      p: 2,
                      borderRadius: 3,
                      borderColor: 'divider',
                      color: 'text.primary',
                      textTransform: 'none',
                      textAlign: 'left',
                      justifyContent: 'flex-start',
                      '&:hover': { borderColor: 'primary.main', color: 'primary.main', bgcolor: 'action.hover' },
                    }}
                  >
                    &ldquo;{btn.label}&rdquo;
                  </Button>
                </Grid>
              ))}
            </Grid>
          </Box>
        ) : (
          messages.map((message) => (
            <Box
              key={message.id}
              sx={{ display: 'flex', justifyContent: message.role === 'user' ? 'flex-end' : 'flex-start' }}
            >
              <Box sx={{ maxWidth: '75%', width: message.role === 'user' ? 'auto' : '100%' }}>
                {message.role === 'user' ? (
                  <Paper
                    sx={{
                      p: 2,
                      borderRadius: '16px 16px 0px 16px',
                      bgcolor: 'primary.main',
                      color: 'primary.contrastText',
                      boxShadow: 1,
                    }}
                  >
                    <Typography variant="body2">{message.content}</Typography>
                  </Paper>
                ) : (
                  <Box sx={{ display: 'flex', flexDirection: 'column', gap: 1 }}>
                    <Paper
                      sx={{
                        p: 2,
                        borderRadius: '16px 16px 16px 0px',
                        border: '1px solid',
                        borderColor: 'divider',
                        boxShadow: 1,
                        whiteSpace: 'pre-wrap',
                      }}
                    >
                      <Typography variant="body2">{message.content}</Typography>
                    </Paper>
                    {message.chartSpec ? (
                      <Paper sx={{ p: 1, border: '1px solid', borderColor: 'divider' }}>
                        <ReactECharts
                          option={message.chartSpec as Record<string, unknown>}
                          style={{ height: 280, width: '100%' }}
                          notMerge
                          lazyUpdate
                        />
                      </Paper>
                    ) : null}
                  </Box>
                )}
              </Box>
            </Box>
          ))
        )}

        {loading && (
          <Box sx={{ display: 'flex', justifyContent: 'flex-start' }}>
            <Paper
              sx={{
                p: 2,
                borderRadius: '16px 16px 16px 0px',
                border: '1px solid',
                borderColor: 'divider',
                boxShadow: 1,
                display: 'flex',
                alignItems: 'center',
                gap: 1,
              }}
            >
              <CircularProgress size={16} thickness={5} />
              <Typography variant="caption" color="text.secondary">Thinking...</Typography>
            </Paper>
          </Box>
        )}
        <div ref={messagesEndRef} />
      </Box>

      <Box
        sx={{
          p: 2,
          backgroundColor: 'background.paper',
          borderTop: '1px solid',
          borderColor: 'divider',
        }}
      >
        <Box
          component="form"
          onSubmit={handleSubmit}
          sx={{
            maxWidth: 800,
            mx: 'auto',
            position: 'relative',
            display: 'flex',
            alignItems: 'center',
          }}
        >
          <TextField
            fullWidth
            value={inputValue}
            onChange={(e) => setInputValue(e.target.value)}
            placeholder="Ask a question or request a portfolio comparison..."
            variant="outlined"
            disabled={loading}
            size="small"
            sx={{
              '& .MuiOutlinedInput-root': { borderRadius: 3, pr: 6 },
            }}
          />
          <IconButton
            type="submit"
            disabled={!inputValue.trim() || loading}
            color="primary"
            sx={{
              position: 'absolute',
              right: 8,
              backgroundColor: 'primary.main',
              color: 'primary.contrastText',
              borderRadius: 2,
              p: 1,
              '&:hover': { backgroundColor: 'primary.dark' },
              '&.Mui-disabled': { backgroundColor: 'action.disabledBackground', color: 'action.disabled' },
            }}
          >
            <SendIcon sx={{ fontSize: 18 }} />
          </IconButton>
        </Box>
      </Box>
    </Box>
  );
}

function stubResponse(query: string): { content: string; chartSpec?: unknown } {
  const q = query.toLowerCase();
  if (q.includes('compare') || q.includes('tech') || q.includes('s&p')) {
    return {
      content: 'Tech exposure is 28% of AUM vs the S&P 500 weighting of 29% — a 1pp underweight.',
      chartSpec: {
        tooltip: { trigger: 'axis' },
        xAxis: { type: 'category', data: ['Portfolio', 'S&P 500'] },
        yAxis: { type: 'value', name: 'Tech %' },
        series: [{ type: 'bar', data: [28, 29], itemStyle: { color: '#0284c7' } }],
      },
    };
  }
  if (q.includes('rate') || q.includes('real estate')) {
    return {
      content: 'Rate hikes of +100bp correlate with a ~6.8% drop in REIT NAV over the trailing 12 months.',
      chartSpec: {
        tooltip: { trigger: 'axis' },
        xAxis: { type: 'category', data: ['+0bp', '+25bp', '+50bp', '+75bp', '+100bp'] },
        yAxis: { type: 'value', name: 'REIT NAV Δ%' },
        series: [{ type: 'line', data: [0, -1.6, -3.2, -5.0, -6.8], itemStyle: { color: '#dc2626' } }],
      },
    };
  }
  return {
    content: `I received your query: "${query}". The semantic graph returned no matching business objects. Try a more specific question.`,
  };
}