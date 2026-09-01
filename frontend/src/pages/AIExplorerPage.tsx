import React, { useState, useEffect, useRef, useMemo, useCallback } from 'react';
import {
  Box,
  Paper,
  Typography,
  TextField,
  IconButton,
  Button,
  Avatar,
  Stack,
  Chip,
  CircularProgress,
  Tooltip,
  Divider,
  List,
  ListItem,
  ListItemText,
  ListItemIcon,
  ListItemButton,
  Menu,
  MenuItem,
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  Card,
  CardContent,
  CardActionArea,
  Badge,
  useTheme,
  Alert,
  Grid,
} from '@mui/material';
import {
  AutoAwesome as SparklesIcon,
  Send as SendIcon,
  Add as AddIcon,
  Folder as FolderIcon,
  FolderOpen as FolderOpenIcon,
  History as HistoryIcon,
  Star as StarIcon,
  StarBorder as StarBorderIcon,
  MoreVert as MoreIcon,
  ContentCopy as CopyIcon,
  PlayArrow as PlayIcon,
  DataObject as CodeIcon,
  Storage as StorageIcon,
  ThumbUp as ThumbUpIcon,
  ThumbDown as ThumbDownIcon,
  Delete as DeleteIcon,
  TrendingUp as TrendingUpIcon,
  BarChart as BarChartIcon,
  AccountBalance as BankIcon,
  Search as SearchIcon,
} from '@mui/icons-material';
import { CoreIcon } from '../components/common/CoreCustomIcons';
import { useNavigate } from 'react-router-dom';
import { useTenant } from '../contexts/TenantContext';
import { fetchBusinessObjects, loadExplorerSource } from '../features/data-explorer/services/dataExplorerApi';
import type { BusinessObjectSummary, ExplorerSource } from '../features/data-explorer/types/dataExplorerTypes';
import { formatDistanceToNow } from 'date-fns';

interface AIChatMessage {
  id: string;
  role: 'user' | 'assistant';
  content: string;
  timestamp: string;
  sourceBo?: string;
  generatedSql?: string;
  suggestedDimensions?: string[];
  suggestedMeasures?: string[];
  confidence?: number;
}

interface AIChatSession {
  id: string;
  title: string;
  folder_id?: string;
  is_pinned?: boolean;
  created_at: string;
  updated_at: string;
  messages: AIChatMessage[];
}

const STORAGE_KEY_AI_SESSIONS = 'uisce_ai_explorer_sessions_v1';

const INITIAL_SESSIONS: AIChatSession[] = [
  {
    id: 'session-001',
    title: 'AUM Distribution & Sponsor Mandates',
    is_pinned: true,
    created_at: new Date(Date.now() - 3600000 * 5).toISOString(),
    updated_at: new Date(Date.now() - 3600000 * 2).toISOString(),
    messages: [
      {
        id: 'msg-1',
        role: 'user',
        content: 'Show me total valuation grouped by client name and sponsor for institutional accounts.',
        timestamp: new Date(Date.now() - 3600000 * 5).toISOString(),
      },
      {
        id: 'msg-2',
        role: 'assistant',
        content: `I've analyzed the semantic ontology for **Institutional Accounts** (\`oms.account/institutional\`). 
Here are the relevant dimensions and measures:
- **Dimensions**: \`client_name\`, \`sponsor_id\`, \`mandate_type\`
- **Measures**: \`SUM(aum_basis_amount)\`, \`COUNT(account_id)\`

Here is the pushdown query generated against the unified semantic catalog:`,
        timestamp: new Date(Date.now() - 3600000 * 4).toISOString(),
        sourceBo: 'oms.account',
        generatedSql: `SELECT 
    acc.client_name,
    acc.sponsor_id,
    acc.mandate_type,
    SUM(acc.aum_basis_amount) AS total_aum,
    COUNT(acc.account_id) AS account_count
FROM oms.account acc
WHERE acc.subtype_code = 'institutional'
  AND acc.valid_to IS NULL
GROUP BY acc.client_name, acc.sponsor_id, acc.mandate_type
ORDER BY total_aum DESC;`,
        suggestedDimensions: ['client_name', 'sponsor_id', 'mandate_type'],
        suggestedMeasures: ['aum_basis_amount'],
        confidence: 0.94,
      },
    ],
  },
];

export const AIExplorerPage: React.FC = () => {
  const navigate = useNavigate();
  const theme = useTheme();
  const { currentTenant } = useTenant();

  const [sessions, setSessions] = useState<AIChatSession[]>(() => {
    const saved = localStorage.getItem(STORAGE_KEY_AI_SESSIONS);
    return saved ? JSON.parse(saved) : INITIAL_SESSIONS;
  });

  const [activeSessionId, setActiveSessionId] = useState<string>(() => {
    return sessions.length > 0 ? sessions[0].id : '';
  });

  const [inputVal, setInputVal] = useState('');
  const [isLoading, setIsLoading] = useState(false);
  const [businessObjects, setBusinessObjects] = useState<BusinessObjectSummary[]>([]);
  const [selectedBo, setSelectedBo] = useState<string>('oms.account');
  const [searchFilter, setSearchFilter] = useState('');

  const messagesEndRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    fetchBusinessObjects().then(setBusinessObjects).catch(() => {});
  }, []);

  const saveSessions = (updated: AIChatSession[]) => {
    setSessions(updated);
    localStorage.setItem(STORAGE_KEY_AI_SESSIONS, JSON.stringify(updated));
  };

  const activeSession = useMemo(() => {
    return sessions.find((s) => s.id === activeSessionId) || sessions[0];
  }, [sessions, activeSessionId]);

  useEffect(() => {
    messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' });
  }, [activeSession?.messages, isLoading]);

  const handleNewSession = () => {
    const newSession: AIChatSession = {
      id: `session-${Date.now()}`,
      title: 'New Co-Pilot Chat',
      created_at: new Date().toISOString(),
      updated_at: new Date().toISOString(),
      messages: [],
    };
    saveSessions([newSession, ...sessions]);
    setActiveSessionId(newSession.id);
  };

  const handleTogglePin = (sessionId: string, e: React.MouseEvent) => {
    e.stopPropagation();
    const updated = sessions.map((s) => (s.id === sessionId ? { ...s, is_pinned: !s.is_pinned } : s));
    saveSessions(updated);
  };

  const handleDeleteSession = (sessionId: string, e: React.MouseEvent) => {
    e.stopPropagation();
    const updated = sessions.filter((s) => s.id !== sessionId);
    saveSessions(updated);
    if (activeSessionId === sessionId && updated.length > 0) {
      setActiveSessionId(updated[0].id);
    }
  };

  const handleSendPrompt = async (textToSend?: string) => {
    const query = (textToSend || inputVal).trim();
    if (!query || isLoading) return;

    setInputVal('');
    setIsLoading(true);

    const userMsg: AIChatMessage = {
      id: `msg-${Date.now()}`,
      role: 'user',
      content: query,
      timestamp: new Date().toISOString(),
    };

    let targetSession = activeSession;
    if (!targetSession) {
      targetSession = {
        id: `session-${Date.now()}`,
        title: query.slice(0, 40),
        created_at: new Date().toISOString(),
        updated_at: new Date().toISOString(),
        messages: [],
      };
    }

    const updatedWithUser = {
      ...targetSession,
      title: targetSession.messages.length === 0 ? query.slice(0, 40) : targetSession.title,
      updated_at: new Date().toISOString(),
      messages: [...targetSession.messages, userMsg],
    };

    const currentSessions = sessions.some((s) => s.id === targetSession.id)
      ? sessions.map((s) => (s.id === targetSession.id ? updatedWithUser : s))
      : [updatedWithUser, ...sessions];

    saveSessions(currentSessions);
    setActiveSessionId(targetSession.id);

    // Simulate AI synthesis over semantic graph / business objects
    setTimeout(() => {
      const isAlt = query.toLowerCase().includes('alt') || query.toLowerCase().includes('private');
      const isTrade = query.toLowerCase().includes('trade') || query.toLowerCase().includes('order');

      let boId = selectedBo || 'oms.account';
      let boName = 'Accounts';
      let sql = '';

      if (isAlt) {
        boId = 'altinv.alternative_investment';
        boName = 'Alternative Investments';
        sql = `SELECT \n    ai.fund_manager,\n    ai.vintage_year,\n    SUM(ai.total_commitment) AS total_committed,\n    SUM(ai.unfunded_commitment) AS unfunded_nav\nFROM altinv.alternative_investment ai\nWHERE ai.subtype_code = 'private_equity'\n  AND ai.valid_to IS NULL\nGROUP BY ai.fund_manager, ai.vintage_year\nORDER BY total_committed DESC;`;
      } else if (isTrade) {
        boId = 'oms.trade_order';
        boName = 'Trade Orders';
        sql = `SELECT \n    t.asset_class,\n    t.currency,\n    SUM(t.order_quantity) AS volume,\n    COUNT(t.order_id) AS trade_count\nFROM oms.trade_order t\nWHERE t.valid_to IS NULL\nGROUP BY t.asset_class, t.currency\nORDER BY volume DESC;`;
      } else {
        sql = `SELECT \n    acc.client_name,\n    acc.sponsor_id,\n    SUM(acc.aum_basis_amount) AS total_aum\nFROM oms.account acc\nWHERE acc.valid_to IS NULL\nGROUP BY acc.client_name, acc.sponsor_id\nORDER BY total_aum DESC;`;
      }

      const assistantMsg: AIChatMessage = {
        id: `msg-${Date.now() + 1}`,
        role: 'assistant',
        content: `Based on your request, I identified **${boName}** (\`${boId}\`) as the primary Business Object and mapped the required dimensions and metrics through the Uuisce Semantic Layer.`,
        timestamp: new Date().toISOString(),
        sourceBo: boId,
        generatedSql: sql,
        suggestedDimensions: ['client_name', 'sponsor_id'],
        suggestedMeasures: ['aum_basis_amount'],
        confidence: 0.92,
      };

      const finalSession = {
        ...updatedWithUser,
        updated_at: new Date().toISOString(),
        messages: [...updatedWithUser.messages, assistantMsg],
      };

      const finalSessions = currentSessions.map((s) => (s.id === finalSession.id ? finalSession : s));
      saveSessions(finalSessions);
      setIsLoading(false);
    }, 1000);
  };

  const handleOpenInPlayground = (msg: AIChatMessage) => {
    navigate('/data-explorer/builder', {
      state: {
        query: {
          name: `${msg.sourceBo} AI Query`,
          sourceId: msg.sourceBo || 'oms.account',
          queryState: {
            sourceId: msg.sourceBo || 'oms.account',
            bindingId: 'default-binding',
            dimensions: msg.suggestedDimensions?.map((d) => ({ fieldId: d })) || [],
            measures: msg.suggestedMeasures?.map((m) => ({ fieldId: m, agg: 'SUM' })) || [],
            timeDimensions: [],
            calculations: [],
            parameters: [],
            filters: [],
            sorts: [],
            limit: 1000,
          },
        },
      },
    });
  };

  return (
    <Box sx={{ display: 'flex', height: 'calc(100vh - 64px)', bgcolor: 'background.default' }}>
      {/* Left Co-Pilot Sidebar: Sessions & Topics */}
      <Paper
        square
        elevation={0}
        sx={{
          width: 300,
          borderRight: 1,
          borderColor: 'divider',
          display: 'flex',
          flexDirection: 'column',
          bgcolor: 'background.paper',
        }}
      >
        <Box sx={{ p: 2, borderBottom: 1, borderColor: 'divider' }}>
          <Button
            fullWidth
            variant="contained"
            color="primary"
            startIcon={<AddIcon />}
            onClick={handleNewSession}
            sx={{ borderRadius: 2, textTransform: 'none', fontWeight: 600 }}
          >
            New Conversation
          </Button>
        </Box>

        <Box sx={{ px: 2, py: 1.5, borderBottom: 1, borderColor: 'divider' }}>
          <TextField
            size="small"
            fullWidth
            placeholder="Search conversations..."
            value={searchFilter}
            onChange={(e) => setSearchFilter(e.target.value)}
            InputProps={{
              startAdornment: <SearchIcon fontSize="small" sx={{ mr: 1, color: 'text.secondary' }} />,
              sx: { borderRadius: 2, bgcolor: 'background.default', fontSize: 13 },
            }}
          />
        </Box>

        <List dense sx={{ flex: 1, overflowY: 'auto', px: 1, py: 1 }}>
          {sessions
            .filter((s) => !searchFilter || s.title.toLowerCase().includes(searchFilter.toLowerCase()))
            .map((session) => (
              <ListItemButton
                key={session.id}
                selected={session.id === activeSessionId}
                onClick={() => setActiveSessionId(session.id)}
                sx={{ borderRadius: 1.5, mb: 0.5 }}
              >
                <ListItemIcon sx={{ minWidth: 32 }}>
                  <SparklesIcon fontSize="small" sx={{ color: session.id === activeSessionId ? 'primary.main' : 'text.secondary' }} />
                </ListItemIcon>
                <ListItemText
                  primary={session.title}
                  secondary={formatDistanceToNow(new Date(session.updated_at)) + ' ago'}
                  primaryTypographyProps={{ variant: 'body2', noWrap: true, fontWeight: session.id === activeSessionId ? 700 : 500 }}
                  secondaryTypographyProps={{ variant: 'caption', fontSize: 11 }}
                />
                <IconButton
                  size="small"
                  onClick={(e) => handleTogglePin(session.id, e)}
                  color={session.is_pinned ? 'warning' : 'default'}
                >
                  {session.is_pinned ? <StarIcon sx={{ fontSize: 16 }} /> : <StarBorderIcon sx={{ fontSize: 16 }} />}
                </IconButton>
                <IconButton size="small" onClick={(e) => handleDeleteSession(session.id, e)}>
                  <DeleteIcon sx={{ fontSize: 16 }} />
                </IconButton>
              </ListItemButton>
            ))}
        </List>
      </Paper>

      {/* Main Co-Pilot Canvas */}
      <Box sx={{ flex: 1, display: 'flex', flexDirection: 'column', overflow: 'hidden' }}>
        {/* Top Header Banner */}
        <Box
          sx={{
            p: 2,
            borderBottom: 1,
            borderColor: 'divider',
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'space-between',
            bgcolor: 'background.paper',
          }}
        >
          <Stack direction="row" spacing={1.5} alignItems="center">
            <Avatar sx={{ bgcolor: 'primary.50', color: 'primary.main', width: 36, height: 36 }}>
              <SparklesIcon fontSize="small" />
            </Avatar>
            <Box>
              <Stack direction="row" spacing={1} alignItems="center">
                <Typography variant="h6" sx={{ fontWeight: 700 }}>
                  AI Explorer Co-Pilot
                </Typography>
                <Chip
                  icon={<CoreIcon sx={{ fontSize: '13px !important' }} />}
                  label="Gold Copy Semantic Mesh"
                  size="small"
                  color="primary"
                  variant="outlined"
                  sx={{ height: 22, fontSize: 11 }}
                />
              </Stack>
              <Typography variant="caption" color="text.secondary">
                Conversational data exploration with automatic business object resolution and SQL synthesis.
              </Typography>
            </Box>
          </Stack>

          <Button
            variant="outlined"
            size="small"
            startIcon={<StorageIcon />}
            onClick={() => navigate('/data-explorer')}
            sx={{ textTransform: 'none', borderRadius: 2 }}
          >
            Open Data Explorer
          </Button>
        </Box>

        {/* Message Thread */}
        <Box sx={{ flex: 1, p: 3, overflowY: 'auto', display: 'flex', flexDirection: 'column', gap: 2.5 }}>
          {(!activeSession || activeSession.messages.length === 0) ? (
            <Box sx={{ maxWidth: 760, mx: 'auto', my: 'auto', textAlign: 'center' }}>
              <Avatar sx={{ width: 56, height: 56, bgcolor: 'primary.50', color: 'primary.main', mx: 'auto', mb: 2 }}>
                <SparklesIcon fontSize="medium" />
              </Avatar>
              <Typography variant="h5" sx={{ fontWeight: 700, mb: 1 }}>
                How can I assist your data exploration today?
              </Typography>
              <Typography variant="body2" color="text.secondary" sx={{ mb: 3 }}>
                Ask questions in natural language. Co-Pilot resolves Business Objects, attests calculations, and generates SQL queries.
              </Typography>

              <Grid container spacing={2}>
                {[
                  {
                    title: 'Institutional Accounts',
                    desc: 'Show total valuation by client name for active accounts',
                    prompt: 'Show total valuation by client name for active institutional accounts',
                    icon: <BankIcon fontSize="small" color="primary" />,
                  },
                  {
                    title: 'Trading Execution',
                    desc: 'Analyze trading volume by asset class and currency',
                    prompt: 'Analyze trade execution volume broken down by asset class and currency',
                    icon: <BarChartIcon fontSize="small" color="secondary" />,
                  },
                  {
                    title: 'PE Commitments',
                    desc: 'List private equity commitments grouped by vintage year',
                    prompt: 'List private equity commitments grouped by vintage year and fund manager',
                    icon: <TrendingUpIcon fontSize="small" color="warning" />,
                  },
                ].map((card, i) => (
                  <Grid   key={i} size={{ xs: 12, sm: 4 }}>
                    <Card
                      elevation={0}
                      sx={{
                        border: 1,
                        borderColor: 'divider',
                        borderRadius: 2.5,
                        textAlign: 'left',
                        cursor: 'pointer',
                        '&:hover': { borderColor: 'primary.main', bgcolor: 'action.hover' },
                      }}
                      onClick={() => handleSendPrompt(card.prompt)}
                    >
                      <CardContent sx={{ p: 2 }}>
                        <Box sx={{ display: 'flex', alignItems: 'center', gap: 1, mb: 1 }}>
                          {card.icon}
                          <Typography variant="subtitle2" sx={{ fontWeight: 700 }}>
                            {card.title}
                          </Typography>
                        </Box>
                        <Typography variant="caption" color="text.secondary">
                          {card.desc}
                        </Typography>
                      </CardContent>
                    </Card>
                  </Grid>
                ))}
              </Grid>
            </Box>
          ) : (
            activeSession.messages.map((msg) => (
              <Box
                key={msg.id}
                sx={{
                  display: 'flex',
                  flexDirection: 'column',
                  alignItems: msg.role === 'user' ? 'flex-end' : 'flex-start',
                }}
              >
                <Paper
                  elevation={0}
                  sx={{
                    p: 2.5,
                    borderRadius: 3,
                    maxWidth: msg.role === 'user' ? '70%' : '85%',
                    bgcolor: msg.role === 'user' ? 'primary.main' : 'background.paper',
                    color: msg.role === 'user' ? 'primary.contrastText' : 'text.primary',
                    border: msg.role === 'assistant' ? 1 : 0,
                    borderColor: 'divider',
                  }}
                >
                  <Typography variant="body1" sx={{ whiteSpace: 'pre-line', lineHeight: 1.6 }}>
                    {msg.content}
                  </Typography>

                  {/* Assistant SQL & Action Card */}
                  {msg.generatedSql && (
                    <Box sx={{ mt: 2, pt: 2, borderTop: 1, borderColor: 'divider' }}>
                      <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', mb: 1 }}>
                        <Stack direction="row" spacing={1} alignItems="center">
                          <CodeIcon fontSize="small" sx={{ color: 'text.secondary' }} />
                          <Typography variant="caption" sx={{ fontWeight: 700 }}>
                            Synthesized SQL Pushdown
                          </Typography>
                        </Stack>
                        <Button
                          size="small"
                          variant="contained"
                          color="primary"
                          startIcon={<PlayIcon fontSize="small" />}
                          onClick={() => handleOpenInPlayground(msg)}
                          sx={{ textTransform: 'none', borderRadius: 1.5, fontSize: 12 }}
                        >
                          Open in Query Playground
                        </Button>
                      </Box>
                      <Paper
                        elevation={0}
                        sx={{
                          p: 1.5,
                          borderRadius: 2,
                          bgcolor: theme.palette.mode === 'dark' ? '#0d1117' : '#f6f8fa',
                          border: 1,
                          borderColor: 'divider',
                          fontFamily: 'monospace',
                          fontSize: 12,
                          overflowX: 'auto',
                        }}
                      >
                        <pre style={{ margin: 0 }}>{msg.generatedSql}</pre>
                      </Paper>
                    </Box>
                  )}
                </Paper>
              </Box>
            ))
          )}
          {isLoading && (
            <Box sx={{ display: 'flex', alignItems: 'center', gap: 1.5, p: 2 }}>
              <CircularProgress size={18} color="primary" />
              <Typography variant="body2" color="text.secondary">
                Co-Pilot is introspecting semantic catalog & synthesizing queries...
              </Typography>
            </Box>
          )}
          <div ref={messagesEndRef} />
        </Box>

        {/* Bottom Co-Pilot Prompt Bar */}
        <Box sx={{ p: 2, borderTop: 1, borderColor: 'divider', bgcolor: 'background.paper' }}>
          <Paper
            elevation={0}
            sx={{
              p: 1,
              display: 'flex',
              alignItems: 'center',
              borderRadius: 3,
              border: 1,
              borderColor: 'divider',
              bgcolor: 'background.default',
            }}
          >
            <TextField
              fullWidth
              variant="standard"
              placeholder="Ask anything or request queries (e.g. 'Show total valuation by client for active accounts in 2026')..."
              value={inputVal}
              onChange={(e) => setInputVal(e.target.value)}
              onKeyDown={(e) => {
                if (e.key === 'Enter' && !e.shiftKey) {
                  e.preventDefault();
                  handleSendPrompt();
                }
              }}
              disabled={isLoading}
              InputProps={{
                disableUnderline: true,
                sx: { px: 1.5, fontSize: 14 },
              }}
            />
            <IconButton
              color="primary"
              onClick={() => handleSendPrompt()}
              disabled={isLoading || !inputVal.trim()}
              sx={{ bgcolor: 'primary.main', color: 'primary.contrastText', '&:hover': { bgcolor: 'primary.dark' } }}
            >
              <SendIcon fontSize="small" />
            </IconButton>
          </Paper>
        </Box>
      </Box>
    </Box>
  );
};

export default AIExplorerPage;
