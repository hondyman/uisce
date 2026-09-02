import React, { useState } from 'react';
import {
  Alert,
  AppBar,
  Avatar,
  Box,
  Button,
  Card,
  CardContent,
  CardHeader,
  Chip,
  CircularProgress,
  Collapse,
  Divider,
  FormControl,
  Grid,
  IconButton,
  InputLabel,
  List,
  ListItem,
  ListItemIcon,
  ListItemText,
  MenuItem,
  Paper,
  Select,
  Snackbar,
  Stack,
  TextField,
  Toolbar,
  Tooltip,
  Typography,
} from '@mui/material';
import {
  AccountTree as CatalogIcon,
  AutoAwesome as AIIcon,
  CheckCircle as CheckCircleIcon,
  ContentCopy as CopyIcon,
  ElectricBolt as FastIcon,
  FormatQuote as QuoteIcon,
  Lock as LockIcon,
  Psychology as PsychologyIcon,
  Send as SendIcon,
  Shield as ShieldIcon,
  ThumbDown as ThumbDownIcon,
  ThumbUp as ThumbUpIcon,
} from '@mui/icons-material';

// ─── Vendor Configuration ─────────────────────────────────────────────────────

const VENDOR_OPTIONS: {
  value: 'CORTEX' | 'DATABRICKS' | 'MCP_CLAUDE' | 'COPILOT';
  label: string;
  color: 'primary' | 'warning' | 'secondary' | 'success';
  avatarBg: string;
  avatarIcon: string;
  engineLabel: string;
  engineDesc: string;
}[] = [
  {
    value: 'CORTEX',
    label: 'Snowflake Cortex Analyst',
    color: 'primary',
    avatarBg: 'primary.dark',
    avatarIcon: '❄️',
    engineLabel: 'Verified Query Semantic Model · YAML v2',
    engineDesc: 'Grounded in Uuisce Business Objects and Cortex YAML verified benchmarks.',
  },
  {
    value: 'DATABRICKS',
    label: 'Databricks Genie Space',
    color: 'warning',
    avatarBg: 'warning.dark',
    avatarIcon: '🧱',
    engineLabel: 'Genie Benchmark SQL · Unity Catalog',
    engineDesc: 'Routed through gold_analytics Unity Catalog semantic model.',
  },
  {
    value: 'MCP_CLAUDE',
    label: 'Claude Desktop (MCP)',
    color: 'secondary',
    avatarBg: 'secondary.dark',
    avatarIcon: '🟣',
    engineLabel: 'Deterministic AST Compiler · Cardinal Rule 7',
    engineDesc: 'Cardinal Rule 7 perimeter enforced via get_semantic_catalog MCP tool.',
  },
  {
    value: 'COPILOT',
    label: 'GitHub Copilot / Cursor',
    color: 'success',
    avatarBg: 'success.dark',
    avatarIcon: '🤖',
    engineLabel: 'MCP Tool Gateway · Token-Budgeted KNN',
    engineDesc: 'pgvector KNN semantic search with dynamic context pruning.',
  },
];

// ─── Sample Prompts ───────────────────────────────────────────────────────────

const SAMPLE_PROMPTS = [
  'Top 5 institutional accounts by total settled portfolio valuation',
  'Unsettled cash flows scheduled this month, grouped by settlement type',
  'Portfolio managers ranked by number of active accounts under management',
  'Distribution of equity vs. debt securities across all active funds',
  'Show me YTD trade volume by asset class for institutional clients',
];

// ─── Result Types ─────────────────────────────────────────────────────────────

interface QueryResult {
  sql: string;
  explanation: string;
  catalogTerms: string[];
}

const MOCK_RESULTS: Record<string, (tenantId: string) => QueryResult> = {
  CORTEX: (tenantId) => ({
    sql:
      `-- ❄️ Snowflake Cortex Analyst (Uuisce Semantic YAML v2)\n` +
      `SELECT\n` +
      `    a.account_number  AS "Account Number",\n` +
      `    c.client_name     AS "Client Name",\n` +
      `    SUM(p.market_value) AS "Total Settled Valuation"\n` +
      `FROM oms.account a\n` +
      `JOIN master.customer c ON a.customer_id = c.id\n` +
      `JOIN oms.position p    ON p.account_id  = a.id\n` +
      `WHERE a.subtype_code   = 'institutional'\n` +
      `  AND p.subtype_code   = 'settled_long'\n` +
      `  AND a.status         = 'ACTIVE'\n` +
      `  AND a.valid_to       IS NULL\n` +
      `GROUP BY a.account_number, c.client_name\n` +
      `ORDER BY "Total Settled Valuation" DESC\n` +
      `LIMIT 5;`,
    explanation:
      'Cortex Analyst parsed "institutional client" to subtype_code = \'institutional\' and "settled portfolio" to position subtype_code = \'settled_long\'. Active bitemporal boundary enforced via valid_to IS NULL.',
    catalogTerms: ['oms.account (Business Object)', 'oms.position.market_value (Measure)', 'master.customer (Entity)', 'subtype_code: institutional'],
  }),
  DATABRICKS: (tenantId) => ({
    sql:
      `-- 🧱 Databricks Genie Space (gold_analytics Unity Catalog)\n` +
      `SELECT\n` +
      `    acc.account_number,\n` +
      `    cust.client_name,\n` +
      `    SUM(pos.market_value) AS total_settled_valuation\n` +
      `FROM gold_analytics.institutional.oms_account acc\n` +
      `JOIN gold_analytics.institutional.master_customer cust\n` +
      `  ON acc.customer_id = cust.id\n` +
      `JOIN gold_analytics.institutional.oms_position pos\n` +
      `  ON pos.account_id = acc.id\n` +
      `WHERE acc.subtype_code  = 'institutional'\n` +
      `  AND pos.subtype_code  = 'settled_long'\n` +
      `GROUP BY acc.account_number, cust.client_name\n` +
      `ORDER BY total_settled_valuation DESC\n` +
      `LIMIT 5;`,
    explanation:
      'Genie Space resolved table joins via the gold_analytics.institutional Unity Catalog semantic model, applying default SUM aggregation on market_value with ABAC-enforced column masking on PII fields.',
    catalogTerms: ['gold_analytics.institutional.oms_account', 'gold_analytics.institutional.oms_position', 'market_value (Measure)', 'Unity Catalog Column Masks'],
  }),
  MCP_CLAUDE: (tenantId) => ({
    sql:
      `-- 🟣 Claude Desktop (Uuisce MCP — get_semantic_catalog + AST Compiler)\n` +
      `-- Cardinal Rule 7: tenant_id = '${tenantId}'\n` +
      `SELECT\n` +
      `    acc.account_number,\n` +
      `    cust.client_name,\n` +
      `    SUM(pos.market_value) AS settled_aum\n` +
      `FROM oms.account acc\n` +
      `JOIN master.customer cust ON acc.customer_id = cust.id\n` +
      `JOIN oms.position pos     ON pos.account_id  = acc.id\n` +
      `WHERE acc.tenant_id    = '${tenantId}'\n` +
      `  AND acc.subtype_code  = 'institutional'\n` +
      `  AND pos.subtype_code  = 'settled_long'\n` +
      `  AND acc.valid_to      IS NULL\n` +
      `GROUP BY acc.account_number, cust.client_name\n` +
      `ORDER BY settled_aum DESC\n` +
      `LIMIT 5;`,
    explanation:
      'The MCP Agent called get_business_object_details and search_semantic_terms to compile deterministic AST SQL. Cardinal Rule 7 multi-tenant perimeter explicitly bound in WHERE clause.',
    catalogTerms: ['get_semantic_catalog (MCP Tool)', 'get_business_object_details (MCP Tool)', 'Cardinal Rule 7 Perimeter', 'AST Deterministic Compiler'],
  }),
  COPILOT: (tenantId) => ({
    sql:
      `-- 🤖 GitHub Copilot / Cursor (MCP KNN Gateway)\n` +
      `-- Top-K semantic terms retrieved: 8 tokens budgeted\n` +
      `SELECT\n` +
      `    acc.account_number,\n` +
      `    cust.client_name,\n` +
      `    SUM(pos.market_value) AS settled_aum\n` +
      `FROM oms.account acc\n` +
      `JOIN master.customer cust ON acc.customer_id = cust.id\n` +
      `JOIN oms.position pos     ON pos.account_id  = acc.id\n` +
      `WHERE acc.tenant_id    = '${tenantId}'\n` +
      `  AND acc.subtype_code  = 'institutional'\n` +
      `  AND pos.subtype_code  = 'settled_long'\n` +
      `  AND acc.valid_to      IS NULL\n` +
      `GROUP BY acc.account_number, cust.client_name\n` +
      `ORDER BY settled_aum DESC\n` +
      `LIMIT 5;`,
    explanation:
      'Copilot used the Uuisce MCP tool gateway. pgvector KNN search returned the top-8 relevant semantic terms within the token budget. Full Rule 7 tenant isolation applied.',
    catalogTerms: ['search_semantic_terms (KNN)', 'pgvector Top-K=8', 'Cardinal Rule 7 Perimeter', 'Token Budget: 8 terms'],
  }),
};

// ─── Main Component ────────────────────────────────────────────────────────────

export const UserAICatalogAssistant: React.FC<{ tenantId?: string }> = ({
  tenantId = '00000000-0000-0000-0000-000000000000',
}) => {
  const [vendor, setVendor] = useState<'CORTEX' | 'DATABRICKS' | 'MCP_CLAUDE' | 'COPILOT'>('CORTEX');
  const [prompt, setPrompt] = useState(SAMPLE_PROMPTS[0]);
  const [loading, setLoading] = useState(false);
  const [result, setResult] = useState<QueryResult | null>(null);
  const [feedback, setFeedback] = useState<'up' | 'down' | null>(null);
  const [snackbar, setSnackbar] = useState('');

  const selectedVendor = VENDOR_OPTIONS.find(v => v.value === vendor)!;

  const handleAsk = async () => {
    setLoading(true);
    setResult(null);
    setFeedback(null);
    await new Promise(r => setTimeout(r, 700));
    setResult(MOCK_RESULTS[vendor](tenantId));
    setLoading(false);
  };

  const handleCopy = (text: string) => {
    navigator.clipboard.writeText(text);
    setSnackbar('SQL copied to clipboard.');
  };

  return (
    <Box sx={{ display: 'flex', flexDirection: 'column', minHeight: '100vh', bgcolor: 'background.default' }}>

      {/* ── Page Header ─────────────────────────────────────────────────────── */}
      <AppBar
        position="sticky"
        color="default"
        elevation={0}
        sx={{ borderBottom: 1, borderColor: 'divider', bgcolor: 'background.paper' }}
      >
        <Toolbar sx={{ gap: 2 }}>
          <Avatar sx={{ bgcolor: 'secondary.dark' }}>
            <PsychologyIcon />
          </Avatar>
          <Box flex={1}>
            <Typography variant="h6" fontWeight={800} lineHeight={1.2}>
              AI Catalog Assistant
            </Typography>
            <Typography variant="caption" color="text.secondary">
              Zero-Hallucination Semantic SQL · Grounded in Uuisce Business Objects
            </Typography>
          </Box>
          <Chip
            icon={<FastIcon />}
            label="Zero-Hallucination Grounding"
            color="success"
            variant="outlined"
            size="small"
            sx={{ fontWeight: 700 }}
          />
        </Toolbar>
      </AppBar>

      <Box sx={{ flex: 1, p: 3, maxWidth: 1400, mx: 'auto', width: '100%' }}>
        <Grid container spacing={3}>

          {/* ── LEFT RAIL: Configuration ───────────────────────────────────── */}
          <Grid item xs={12} md={4} lg={3}>
            <Stack spacing={2}>

              {/* Vendor Selector */}
              <Paper variant="outlined" sx={{ p: 2 }}>
                <Typography variant="overline" color="text.secondary" display="block" sx={{ mb: 1.5 }}>
                  Target AI Engine
                </Typography>
                <FormControl fullWidth size="small">
                  <InputLabel>AI Provider</InputLabel>
                  <Select
                    label="AI Provider"
                    value={vendor}
                    onChange={e => { setVendor(e.target.value as any); setResult(null); }}
                  >
                    {VENDOR_OPTIONS.map(v => (
                      <MenuItem key={v.value} value={v.value}>
                        <Stack direction="row" spacing={1} alignItems="center">
                          <Typography>{v.avatarIcon}</Typography>
                          <Typography variant="body2" fontWeight={600}>{v.label}</Typography>
                        </Stack>
                      </MenuItem>
                    ))}
                  </Select>
                </FormControl>

                <Box sx={{ mt: 2, p: 1.5, bgcolor: 'action.hover', borderRadius: 1 }}>
                  <Typography variant="caption" color="text.secondary" display="block" fontWeight={700} gutterBottom>
                    {selectedVendor.engineLabel}
                  </Typography>
                  <Typography variant="caption" color="text.secondary">
                    {selectedVendor.engineDesc}
                  </Typography>
                </Box>
              </Paper>

              {/* Sample Prompts */}
              <Paper variant="outlined" sx={{ p: 2 }}>
                <Typography variant="overline" color="text.secondary" display="block" sx={{ mb: 1 }}>
                  Suggested Questions
                </Typography>
                <Stack spacing={0.75}>
                  {SAMPLE_PROMPTS.map((p, i) => (
                    <Card
                      key={i}
                      variant={prompt === p ? 'outlined' : 'elevation'}
                      elevation={prompt === p ? 0 : 0}
                      onClick={() => setPrompt(p)}
                      sx={{
                        cursor: 'pointer',
                        border: 1,
                        borderColor: prompt === p ? 'primary.main' : 'divider',
                        bgcolor: prompt === p ? 'primary.50' : 'background.paper',
                        '&:hover': { borderColor: 'primary.light', bgcolor: 'action.hover' },
                        transition: 'all 0.15s ease',
                      }}
                    >
                      <CardContent sx={{ py: 1, px: 1.5, '&:last-child': { pb: 1 } }}>
                        <Stack direction="row" spacing={1} alignItems="flex-start">
                          <QuoteIcon sx={{ fontSize: 14, color: 'text.disabled', mt: 0.2, flexShrink: 0 }} />
                          <Typography variant="caption" sx={{ lineHeight: 1.4 }}>{p}</Typography>
                        </Stack>
                      </CardContent>
                    </Card>
                  ))}
                </Stack>
              </Paper>

              {/* Guarantee badges */}
              <Paper variant="outlined" sx={{ p: 2 }}>
                <Typography variant="overline" color="text.secondary" display="block" sx={{ mb: 1.5 }}>
                  Active Guarantees
                </Typography>
                <List dense disablePadding>
                  {[
                    { icon: <ShieldIcon fontSize="small" />, color: 'success' as const, label: 'ABAC entitlements enforced' },
                    { icon: <LockIcon fontSize="small" />, color: 'success' as const, label: 'Cardinal Rule 7 — tenant isolation' },
                    { icon: <CheckCircleIcon fontSize="small" />, color: 'success' as const, label: 'Bitemporal valid_to IS NULL' },
                    { icon: <CatalogIcon fontSize="small" />, color: 'info' as const, label: 'STI discriminator predicates' },
                  ].map((g, i) => (
                    <ListItem key={i} disablePadding sx={{ py: 0.25 }}>
                      <ListItemIcon sx={{ minWidth: 28, color: `${g.color}.main` }}>{g.icon}</ListItemIcon>
                      <ListItemText primary={g.label} primaryTypographyProps={{ variant: 'caption' }} />
                    </ListItem>
                  ))}
                </List>
              </Paper>

            </Stack>
          </Grid>

          {/* ── MAIN PANEL: Prompt + Result ────────────────────────────────── */}
          <Grid item xs={12} md={8} lg={9}>
            <Stack spacing={3}>

              {/* Prompt Input */}
              <Paper variant="outlined" sx={{ p: 2.5 }}>
                <Typography variant="subtitle2" fontWeight={700} gutterBottom>
                  Ask anything about your financial data
                </Typography>
                <Stack spacing={1.5}>
                  <TextField
                    multiline
                    minRows={3}
                    maxRows={6}
                    fullWidth
                    placeholder="Describe what you want to query in natural language…"
                    value={prompt}
                    onChange={e => setPrompt(e.target.value)}
                    variant="outlined"
                  />
                  <Stack direction="row" justifyContent="flex-end">
                    <Button
                      variant="contained"
                      color={selectedVendor.color}
                      size="large"
                      startIcon={loading ? <CircularProgress size={18} color="inherit" /> : <SendIcon />}
                      onClick={handleAsk}
                      disabled={loading || !prompt.trim()}
                      sx={{ textTransform: 'none', fontWeight: 700, px: 4 }}
                    >
                      {loading ? 'Generating…' : 'Ask AI'}
                    </Button>
                  </Stack>
                </Stack>
              </Paper>

              {/* Loading Skeleton */}
              {loading && (
                <Paper variant="outlined" sx={{ p: 3 }}>
                  <Stack spacing={2} alignItems="center">
                    <CircularProgress size={40} color={selectedVendor.color} />
                    <Typography variant="body2" color="text.secondary">
                      Resolving semantic terms and compiling SQL against {selectedVendor.label}…
                    </Typography>
                  </Stack>
                </Paper>
              )}

              {/* Result */}
              <Collapse in={!!result && !loading}>
                {result && (
                  <Grid container spacing={2.5}>

                    {/* SQL Output */}
                    <Grid item xs={12} lg={7}>
                      <Card variant="outlined">
                        <CardHeader
                          avatar={
                            <Avatar sx={{ bgcolor: selectedVendor.avatarBg, fontSize: 16 }}>
                              {selectedVendor.avatarIcon}
                            </Avatar>
                          }
                          title={
                            <Typography variant="subtitle2" fontWeight={700}>
                              Semantic-Grounded Executable SQL
                            </Typography>
                          }
                          subheader={
                            <Typography variant="caption" color="text.secondary">{selectedVendor.label}</Typography>
                          }
                          action={
                            <Stack direction="row" alignItems="center" spacing={0.5} sx={{ mt: 0.5, mr: 0.5 }}>
                              <Tooltip title="Mark as accurate — promotes to verified query benchmark">
                                <IconButton
                                  size="small"
                                  color={feedback === 'up' ? 'success' : 'default'}
                                  onClick={() => setFeedback('up')}
                                >
                                  <ThumbUpIcon fontSize="small" />
                                </IconButton>
                              </Tooltip>
                              <Tooltip title="Flag for steward review">
                                <IconButton
                                  size="small"
                                  color={feedback === 'down' ? 'error' : 'default'}
                                  onClick={() => setFeedback('down')}
                                >
                                  <ThumbDownIcon fontSize="small" />
                                </IconButton>
                              </Tooltip>
                              <Tooltip title="Copy SQL">
                                <IconButton size="small" onClick={() => handleCopy(result.sql)}>
                                  <CopyIcon fontSize="small" />
                                </IconButton>
                              </Tooltip>
                            </Stack>
                          }
                        />
                        <Divider />
                        <CardContent sx={{ p: 0 }}>
                          <TextField
                            multiline
                            minRows={14}
                            fullWidth
                            value={result.sql}
                            InputProps={{
                              readOnly: true,
                              sx: {
                                fontFamily: 'monospace',
                                fontSize: 12,
                                color: 'info.main',
                                bgcolor: 'action.hover',
                              },
                            }}
                            sx={{ '& fieldset': { border: 'none' } }}
                          />
                        </CardContent>
                      </Card>
                    </Grid>

                    {/* Attribution Panel */}
                    <Grid item xs={12} lg={5}>
                      <Stack spacing={2}>

                        {/* Semantic Reasoning */}
                        <Card variant="outlined">
                          <CardHeader
                            avatar={<Avatar sx={{ bgcolor: 'success.dark', width: 32, height: 32 }}><AIIcon fontSize="small" /></Avatar>}
                            title={<Typography variant="subtitle2" fontWeight={700}>Semantic Reasoning</Typography>}
                            sx={{ pb: 0 }}
                          />
                          <Divider />
                          <CardContent>
                            <Typography variant="body2" color="text.secondary" sx={{ lineHeight: 1.65 }}>
                              {result.explanation}
                            </Typography>
                          </CardContent>
                        </Card>

                        {/* Catalog Attribution */}
                        <Card variant="outlined">
                          <CardHeader
                            avatar={<Avatar sx={{ bgcolor: 'secondary.dark', width: 32, height: 32 }}><CatalogIcon fontSize="small" /></Avatar>}
                            title={<Typography variant="subtitle2" fontWeight={700}>Catalog Attribution</Typography>}
                            subheader={<Typography variant="caption" color="text.secondary">Business Objects & Terms Used</Typography>}
                            sx={{ pb: 0 }}
                          />
                          <Divider />
                          <CardContent>
                            <Stack direction="row" flexWrap="wrap" gap={0.75}>
                              {result.catalogTerms.map((term, i) => (
                                <Chip
                                  key={i}
                                  icon={<CatalogIcon />}
                                  label={term}
                                  size="small"
                                  color="secondary"
                                  variant="outlined"
                                  sx={{ fontSize: 11, fontWeight: 600 }}
                                />
                              ))}
                            </Stack>
                          </CardContent>
                        </Card>

                        {/* Security Guarantees */}
                        <Card variant="outlined">
                          <CardContent>
                            <Stack spacing={0.75}>
                              {[
                                { label: 'ABAC + Rule 7 multi-tenant boundary enforced', color: 'success' as const },
                                { label: 'Bitemporal valid_to IS NULL asserted', color: 'success' as const },
                                { label: 'STI discriminator predicates compiled', color: 'info' as const },
                                { label: 'Audit receipt logged to observability ledger', color: 'info' as const },
                              ].map((g, i) => (
                                <Stack key={i} direction="row" spacing={1} alignItems="center">
                                  <CheckCircleIcon color={g.color} sx={{ fontSize: 14 }} />
                                  <Typography variant="caption" color="text.secondary">{g.label}</Typography>
                                </Stack>
                              ))}
                            </Stack>
                          </CardContent>
                        </Card>
                      </Stack>
                    </Grid>

                    {/* Feedback Banner */}
                    <Grid item xs={12}>
                      <Collapse in={feedback !== null}>
                        <Alert
                          severity={feedback === 'up' ? 'success' : 'warning'}
                          icon={feedback === 'up' ? <ThumbUpIcon /> : <ThumbDownIcon />}
                        >
                          {feedback === 'up'
                            ? 'Marked as accurate — this SQL will be promoted to the Cortex verified_queries benchmark set on next sync.'
                            : 'Flagged for review — queued for data steward inspection and semantic model re-training.'}
                        </Alert>
                      </Collapse>
                    </Grid>

                  </Grid>
                )}
              </Collapse>

              {/* Empty state */}
              {!result && !loading && (
                <Paper variant="outlined" sx={{ p: 6, textAlign: 'center' }}>
                  <PsychologyIcon sx={{ fontSize: 56, color: 'text.disabled', mb: 2 }} />
                  <Typography variant="h6" color="text.secondary" gutterBottom>
                    Ask your first question
                  </Typography>
                  <Typography variant="body2" color="text.disabled" sx={{ maxWidth: 400, mx: 'auto' }}>
                    Select a prompt from the left panel or type your own. Uuisce will compile zero-hallucination SQL
                    grounded in your semantic catalog and business object contracts.
                  </Typography>
                </Paper>
              )}

            </Stack>
          </Grid>
        </Grid>
      </Box>

      {/* ── Toast ───────────────────────────────────────────────────────────── */}
      <Snackbar
        open={!!snackbar}
        autoHideDuration={3000}
        onClose={() => setSnackbar('')}
        anchorOrigin={{ vertical: 'bottom', horizontal: 'center' }}
      >
        <Alert severity="success" variant="filled" onClose={() => setSnackbar('')}>{snackbar}</Alert>
      </Snackbar>

    </Box>
  );
};

export default UserAICatalogAssistant;
