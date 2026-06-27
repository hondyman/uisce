import React, { useState, useRef, useEffect } from 'react';
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
  CircularProgress
} from '@mui/material';
import { Send as SendIcon, Sparkles as SparklesIcon } from '@mui/icons-material';
import { ComparisonChart } from '../genui/components/ComparisonChart';
import { ComplianceDisclaimer } from '../genui/components/ComplianceDisclaimer';
import { ImpactAnalysisCard } from '../genui/components/ImpactAnalysisCard';

interface Message {
  id: string;
  role: 'user' | 'assistant';
  content: string;
  display?: React.ReactNode;
  timestamp: Date;
}

export default function GenUIChatPage() {
  const [messages, setMessages] = useState<Message[]>([]);
  const [inputValue, setInputValue] = useState('');
  const [loading, setLoading] = useState(false);
  const messagesEndRef = useRef<HTMLDivElement>(null);
  const theme = useTheme();

  const scrollToBottom = () => {
    messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' });
  };

  useEffect(() => {
    scrollToBottom();
  }, [messages]);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!inputValue.trim() || loading) return;

    const userQuery = inputValue;
    setInputValue('');
    setLoading(true);

    const userMessage: Message = {
      id: `user-${Date.now()}`,
      role: 'user',
      content: userQuery,
      timestamp: new Date(),
    };

    setMessages(prev => [...prev, userMessage]);

    // Simulate AI response logic
    setTimeout(() => {
      let displayComponent: React.ReactNode | undefined;
      let textResponse = '';

      const queryLower = userQuery.toLowerCase();

      if (queryLower.includes('compare') || queryLower.includes('s&p 500') || queryLower.includes('tech')) {
        displayComponent = (
          <ComparisonChart 
            metric="Tech Exposure" 
            benchmark="S&P 500" 
            period="YTD" 
          />
        );
        textResponse = "Here is the comparison chart for your Tech Exposure vs the S&P 500 index YTD.";
      } else if (queryLower.includes('futures') || queryLower.includes('risk') || queryLower.includes('option')) {
        displayComponent = (
          <ComplianceDisclaimer 
            topic="Futures & Derivatives Trading" 
            severity="warning" 
          />
        );
        textResponse = "I have flagged this topic with the required compliance notice for Futures and regulated derivatives.";
      } else if (queryLower.includes('rate hikes') || queryLower.includes('real estate') || queryLower.includes('impact')) {
        displayComponent = (
          <ImpactAnalysisCard 
            headline="Fed signals further rate hikes amid persistent inflation" 
            affectedSector="Real Estate & Financials" 
            impactScore={82} 
          />
        );
        textResponse = "Here is the projected impact analysis of the latest interest rate hikes on interest-sensitive sectors.";
      } else {
        textResponse = `I received your query: "${userQuery}". You can try asking things like:
- "Compare Tech vs S&P 500"
- "What are the risks of Futures Trading?"
- "Analyze impact of rate hikes on Real Estate"`;
      }

      const assistantMessage: Message = {
        id: `assistant-${Date.now()}`,
        role: 'assistant',
        content: textResponse,
        display: displayComponent,
        timestamp: new Date(),
      };

      setMessages(prev => [...prev, assistantMessage]);
      setLoading(false);
    }, 1500);
  };

  return (
    <Box sx={{ 
      display: 'flex', 
      flexDirection: 'column', 
      height: 'calc(100vh - 80px)', 
      backgroundColor: theme.palette.mode === 'dark' ? 'background.default' : 'grey.50' 
    }}>
      {/* Header */}
      <Box sx={{ 
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
        zIndex: 10
      }}>
        <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
          <Avatar sx={{ bgcolor: theme.palette.primary.main, width: 32, height: 32 }}>
            <SparklesIcon sx={{ fontSize: 18 }} />
          </Avatar>
          <Typography variant="h6" sx={{ fontWeight: 'bold' }}>
            WealthStream OS Console
          </Typography>
        </Box>
        <Box sx={{ display: 'flex', alignItems: 'center', gap: 0.5, color: 'text.secondary', fontSize: '0.875rem' }}>
          <SparklesIcon sx={{ fontSize: 14, color: theme.palette.primary.main }} />
          Powered by Gemini Pro
        </Box>
      </Box>

      {/* Chat History */}
      <Box sx={{ flex: 1, overflowY: 'auto', p: 3, display: 'flex', flexDirection: 'column', gap: 3 }}>
        {messages.length === 0 ? (
          <Box sx={{ 
            height: '100%', 
            display: 'flex', 
            flexDirection: 'column', 
            alignItems: 'center', 
            justifyContent: 'center', 
            textAlign: 'center', 
            color: 'text.secondary',
            gap: 2
          }}>
            <Avatar sx={{ bgcolor: 'action.hover', width: 64, height: 64, fontSize: 32 }}>
              ✨
            </Avatar>
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
                { label: "Compare Tech vs S&P 500", query: "Compare my Tech exposure to S&P 500 YTD" },
                { label: "Risks of Futures Trading", query: "What are the risks of Futures Trading?" },
                { label: "Impact of Rate Hikes", query: "Analyze impact of rate hikes on Real Estate" }
              ].map((btn, index) => (
                <Grid item xs={12} md={4} key={index}>
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
                      '&:hover': {
                        borderColor: 'primary.main',
                        color: 'primary.main',
                        bgcolor: 'action.hover'
                      }
                    }}
                  >
                    "{btn.label}"
                  </Button>
                </Grid>
              ))}
            </Grid>
          </Box>
        ) : (
          messages.map((message) => (
            <Box 
              key={message.id} 
              sx={{ 
                display: 'flex', 
                justifyContent: message.role === 'user' ? 'flex-end' : 'flex-start' 
              }}
            >
              <Box sx={{ maxWidth: '75%', width: message.role === 'user' ? 'auto' : '100%' }}>
                {message.role === 'user' ? (
                  <Paper sx={{ 
                    p: 2, 
                    borderRadius: '16px 16px 0px 16px', 
                    bgcolor: 'primary.main', 
                    color: 'primary.contrastText',
                    boxShadow: 1
                  }}>
                    <Typography variant="body2">{message.content}</Typography>
                  </Paper>
                ) : (
                  <Box sx={{ display: 'flex', flexDirection: 'column', gap: 1 }}>
                    <Paper sx={{ 
                      p: 2, 
                      borderRadius: '16px 16px 16px 0px', 
                      border: '1px solid',
                      borderColor: 'divider',
                      boxShadow: 1,
                      whiteSpace: 'pre-wrap'
                    }}>
                      <Typography variant="body2">{message.content}</Typography>
                    </Paper>
                    {message.display}
                  </Box>
                )}
              </Box>
            </Box>
          ))
        )}
        
        {loading && (
          <Box sx={{ display: 'flex', justifyContent: 'flex-start' }}>
            <Paper sx={{ 
              p: 2, 
              borderRadius: '16px 16px 16px 0px', 
              border: '1px solid',
              borderColor: 'divider',
              boxShadow: 1,
              display: 'flex',
              alignItems: 'center',
              gap: 1
            }}>
              <CircularProgress size={16} thickness={5} />
              <Typography variant="caption" color="text.secondary">Thinking...</Typography>
            </Paper>
          </Box>
        )}
        <div ref={messagesEndRef} />
      </Box>

      {/* Input Form */}
      <Box sx={{ 
        p: 2, 
        backgroundColor: 'background.paper', 
        borderTop: '1px solid', 
        borderColor: 'divider' 
      }}>
        <Box 
          component="form" 
          onSubmit={handleSubmit} 
          sx={{ 
            maxWidth: 800, 
            mx: 'auto', 
            position: 'relative', 
            display: 'flex', 
            alignItems: 'center' 
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
              '& .MuiOutlinedInput-root': {
                borderRadius: 3,
                pr: 6
              }
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
              '&:hover': {
                backgroundColor: 'primary.dark'
              },
              '&.Mui-disabled': {
                backgroundColor: 'action.disabledBackground',
                color: 'action.disabled'
              }
            }}
          >
            <SendIcon sx={{ fontSize: 18 }} />
          </IconButton>
        </Box>
      </Box>
    </Box>
  );
}

  );
}
