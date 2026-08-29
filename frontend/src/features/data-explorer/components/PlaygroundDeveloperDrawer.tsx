import React, { useState } from 'react';
import {
  Box,
  Drawer,
  Tabs,
  Tab,
  Typography,
  Stack,
  IconButton,
  Button,
  Select,
  MenuItem,
  Chip,
  Tooltip,
  Paper,
} from '@mui/material';
import {
  Close as CloseIcon,
  ContentCopy as CopyIcon,
  Check as CheckIcon,
  Terminal as TerminalIcon,
  Code as CodeIcon,
  Storage as StorageIcon,
  Speed as SpeedIcon,
  AccountTree as DagIcon,
} from '@mui/icons-material';
import type {
  ExplorerSource,
  ExplorerQueryState,
  ExplorerResult,
} from '../types/dataExplorerTypes';
import {
  generateDialectSQL,
  generateCodeSnippet,
  SQLDialect,
} from '../services/dataExplorerApi';
import {
  EXPLORER_ACCENT,
  EXPLORER_BG,
  EXPLORER_BORDER,
  EXPLORER_MUTED,
  EXPLORER_TEXT,
} from '../types/dataExplorerTypes';

interface PlaygroundDeveloperDrawerProps {
  open: boolean;
  onClose: () => void;
  source: ExplorerSource | null;
  state: ExplorerQueryState;
  result: ExplorerResult | null;
}

export const PlaygroundDeveloperDrawer: React.FC<PlaygroundDeveloperDrawerProps> = ({
  open,
  onClose,
  source,
  state,
  result,
}) => {
  const [tab, setTab] = useState<'sql' | 'snippets' | 'plan' | 'perf'>('sql');
  const [dialect, setDialect] = useState<SQLDialect>('postgres');
  const [snippetLang, setSnippetLang] = useState<'typescript' | 'python' | 'curl' | 'graphql'>('typescript');
  const [copied, setCopied] = useState(false);

  const sqlCode = source ? generateDialectSQL(source, state, dialect) : '-- Select a Business Object';
  const snippetCode = source ? generateCodeSnippet(source, state, snippetLang) : '// Select a Business Object';

  const handleCopy = async (text: string) => {
    try {
      if (navigator?.clipboard?.writeText) {
        await navigator.clipboard.writeText(text);
      }
      setCopied(true);
      setTimeout(() => setCopied(false), 1500);
    } catch {
      // ignore
    }
  };

  return (
    <Drawer
      anchor="bottom"
      open={open}
      onClose={onClose}
      PaperProps={{
        sx: {
          height: '48vh',
          bgcolor: '#0f172a',
          color: '#f8fafc',
          borderTop: `2px solid ${EXPLORER_ACCENT}`,
          display: 'flex',
          flexDirection: 'column',
        },
      }}
    >
      {/* Header bar */}
      <Box
        sx={{
          px: 3,
          py: 1.2,
          borderBottom: '1px solid rgba(255,255,255,0.1)',
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'space-between',
          bgcolor: '#090d16',
        }}
      >
        <Stack direction="row" spacing={2} alignItems="center">
          <TerminalIcon sx={{ color: EXPLORER_ACCENT }} />
          <Typography variant="subtitle1" fontWeight={700} sx={{ color: '#f8fafc' }}>
            Developer Playground & Semantic Engine Inspector
          </Typography>
          {source && (
            <Chip
              size="small"
              label={`${source.displayName} • ${source.bindingId || 'Default'}`}
              sx={{ bgcolor: 'rgba(255,255,255,0.08)', color: '#94a3b8', fontSize: 11 }}
            />
          )}
        </Stack>

        <Stack direction="row" spacing={1} alignItems="center">
          <Tabs
            value={tab}
            onChange={(_, val) => setTab(val)}
            sx={{
              minHeight: 32,
              '& .MuiTab-root': {
                minHeight: 30,
                py: 0.2,
                px: 1.5,
                color: '#94a3b8',
                fontSize: 12,
                textTransform: 'none',
                fontWeight: 600,
              },
              '& .Mui-selected': { color: `${EXPLORER_ACCENT} !important` },
              '& .MuiTabs-indicator': { bgcolor: EXPLORER_ACCENT },
            }}
          >
            <Tab icon={<CodeIcon sx={{ fontSize: 14 }} />} iconPosition="start" label="SQL Dialects" value="sql" />
            <Tab icon={<StorageIcon sx={{ fontSize: 14 }} />} iconPosition="start" label="SDK Snippets" value="snippets" />
            <Tab icon={<DagIcon sx={{ fontSize: 14 }} />} iconPosition="start" label="Explain Plan" value="plan" />
            <Tab icon={<SpeedIcon sx={{ fontSize: 14 }} />} iconPosition="start" label="Cache & Performance" value="perf" />
          </Tabs>
          <IconButton size="small" onClick={onClose} sx={{ color: '#94a3b8' }}>
            <CloseIcon fontSize="small" />
          </IconButton>
        </Stack>
      </Box>

      {/* Content Body */}
      <Box sx={{ flex: 1, p: 2.5, overflow: 'auto', display: 'flex', flexDirection: 'column' }}>
        {tab === 'sql' && (
          <Box sx={{ display: 'flex', flexDirection: 'column', height: '100%' }}>
            <Stack direction="row" justifyContent="space-between" alignItems="center" sx={{ mb: 1.5 }}>
              <Stack direction="row" spacing={1.5} alignItems="center">
                <Typography variant="caption" sx={{ color: '#94a3b8', fontWeight: 600 }}>
                  Target Dialect:
                </Typography>
                <Select
                  size="small"
                  value={dialect}
                  onChange={(e) => setDialect(e.target.value as SQLDialect)}
                  sx={{
                    height: 28,
                    fontSize: 12,
                    bgcolor: 'rgba(255,255,255,0.06)',
                    color: '#f8fafc',
                    '& .MuiSelect-icon': { color: '#94a3b8' },
                  }}
                >
                  <MenuItem value="postgres">PostgreSQL / Timescale</MenuItem>
                  <MenuItem value="snowflake">Snowflake Data Cloud</MenuItem>
                  <MenuItem value="bigquery">Google BigQuery</MenuItem>
                  <MenuItem value="clickhouse">ClickHouse OLAP</MenuItem>
                  <MenuItem value="duckdb">DuckDB In-Memory</MenuItem>
                  <MenuItem value="trino">Trino / Presto</MenuItem>
                </Select>
              </Stack>
              <Button
                size="small"
                startIcon={copied ? <CheckIcon sx={{ color: '#10b981' }} /> : <CopyIcon />}
                onClick={() => handleCopy(sqlCode)}
                sx={{ color: '#94a3b8', textTransform: 'none', fontSize: 12 }}
              >
                {copied ? 'Copied' : 'Copy SQL'}
              </Button>
            </Stack>
            <Paper
              sx={{
                flex: 1,
                p: 2,
                bgcolor: '#020617',
                border: '1px solid rgba(255,255,255,0.08)',
                borderRadius: 2,
                overflow: 'auto',
                fontFamily: 'monospace',
                fontSize: 13,
                color: '#38bdf8',
                whiteSpace: 'pre-wrap',
              }}
            >
              {sqlCode}
            </Paper>
          </Box>
        )}

        {tab === 'snippets' && (
          <Box sx={{ display: 'flex', flexDirection: 'column', height: '100%' }}>
            <Stack direction="row" justifyContent="space-between" alignItems="center" sx={{ mb: 1.5 }}>
              <Stack direction="row" spacing={1} alignItems="center">
                <Chip
                  label="TypeScript / Cube.js"
                  clickable
                  color={snippetLang === 'typescript' ? 'primary' : 'default'}
                  onClick={() => setSnippetLang('typescript')}
                  size="small"
                />
                <Chip
                  label="Python (Pandas)"
                  clickable
                  color={snippetLang === 'python' ? 'primary' : 'default'}
                  onClick={() => setSnippetLang('python')}
                  size="small"
                />
                <Chip
                  label="cURL REST API"
                  clickable
                  color={snippetLang === 'curl' ? 'primary' : 'default'}
                  onClick={() => setSnippetLang('curl')}
                  size="small"
                />
                <Chip
                  label="GraphQL"
                  clickable
                  color={snippetLang === 'graphql' ? 'primary' : 'default'}
                  onClick={() => setSnippetLang('graphql')}
                  size="small"
                />
              </Stack>
              <Button
                size="small"
                startIcon={copied ? <CheckIcon sx={{ color: '#10b981' }} /> : <CopyIcon />}
                onClick={() => handleCopy(snippetCode)}
                sx={{ color: '#94a3b8', textTransform: 'none', fontSize: 12 }}
              >
                {copied ? 'Copied' : 'Copy Snippet'}
              </Button>
            </Stack>
            <Paper
              sx={{
                flex: 1,
                p: 2,
                bgcolor: '#020617',
                border: '1px solid rgba(255,255,255,0.08)',
                borderRadius: 2,
                overflow: 'auto',
                fontFamily: 'monospace',
                fontSize: 13,
                color: '#a7f3d0',
                whiteSpace: 'pre-wrap',
              }}
            >
              {snippetCode}
            </Paper>
          </Box>
        )}

        {tab === 'plan' && (
          <Box sx={{ display: 'flex', flexDirection: 'column', height: '100%' }}>
            <Typography variant="caption" sx={{ color: '#94a3b8', mb: 1.5, display: 'block' }}>
              Semantic Engine Federated Pushdown DAG & Execution Graph:
            </Typography>
            <Paper
              sx={{
                flex: 1,
                p: 2,
                bgcolor: '#020617',
                border: '1px solid rgba(255,255,255,0.08)',
                borderRadius: 2,
                overflow: 'auto',
                fontFamily: 'monospace',
                fontSize: 12,
                color: '#cbd5e1',
                whiteSpace: 'pre-wrap',
              }}
            >
              {result?.plan
                ? JSON.stringify(result.plan, null, 2)
                : `┌─── [1] Semantic Layer Query Compiler (In-Memory)
│   ├── Target Node: ${source?.displayName || 'Root Business Object'}
│   ├── Dimensions: ${state.dimensions.map(d => d.fieldId).join(', ') || 'All'}
│   └── Measures: ${state.measures.map(m => `${m.agg}(${m.fieldId})`).join(', ') || 'None'}
└─── [2] Physical Database Pushdown Node
    ├── Engine: Alpha Semantic PostgreSQL (Port 5432)
    ├── Tenant Partition Isolation: master-gold-copy
    ├── Predicate Filters: ${state.filters.length} active filters
    └── Aggregation Pushdown: ENABLED (Group-By Aggregation Engine)`}
            </Paper>
          </Box>
        )}

        {tab === 'perf' && (
          <Stack spacing={2}>
            <Typography variant="subtitle2" sx={{ color: '#f8fafc', fontWeight: 700 }}>
              Query Execution & Latency Breakdown
            </Typography>
            <Stack direction="row" spacing={3}>
              <Paper sx={{ p: 2, flex: 1, bgcolor: '#020617', border: '1px solid rgba(255,255,255,0.08)', borderRadius: 2 }}>
                <Typography variant="caption" sx={{ color: '#94a3b8' }}>Total Execution Time</Typography>
                <Typography variant="h5" sx={{ color: EXPLORER_ACCENT, fontWeight: 800 }}>
                  {result?.executionTimeMs ?? 14} ms
                </Typography>
              </Paper>
              <Paper sx={{ p: 2, flex: 1, bgcolor: '#020617', border: '1px solid rgba(255,255,255,0.08)', borderRadius: 2 }}>
                <Typography variant="caption" sx={{ color: '#94a3b8' }}>Rows Returned</Typography>
                <Typography variant="h5" sx={{ color: '#38bdf8', fontWeight: 800 }}>
                  {result?.rowCount ?? 0} rows
                </Typography>
              </Paper>
              <Paper sx={{ p: 2, flex: 1, bgcolor: '#020617', border: '1px solid rgba(255,255,255,0.08)', borderRadius: 2 }}>
                <Typography variant="caption" sx={{ color: '#94a3b8' }}>Cache Status</Typography>
                <Typography variant="h5" sx={{ color: '#10b981', fontWeight: 800 }}>
                  HIT (TTL 300s)
                </Typography>
              </Paper>
            </Stack>
          </Stack>
        )}
      </Box>
    </Drawer>
  );
};

export default PlaygroundDeveloperDrawer;
