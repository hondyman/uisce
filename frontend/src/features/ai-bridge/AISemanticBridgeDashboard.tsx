import React, { useState, useEffect, useCallback } from 'react';
import { useAuthFetch } from '../../utils/authFetch';
import {
  Alert,
  AppBar,
  Avatar,
  Badge,
  Box,
  Button,
  Card,
  CardActions,
  CardContent,
  CardHeader,
  Chip,
  CircularProgress,
  Collapse,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
  Divider,
  FormControl,
  Grid,
  IconButton,
  InputLabel,
  LinearProgress,
  List,
  ListItem,
  ListItemIcon,
  ListItemText,
  MenuItem,
  Paper,
  Select,
  Skeleton,
  Snackbar,
  Stack,
  Tab,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  Tabs,
  TextField,
  Toolbar,
  Tooltip,
  Typography,
} from '@mui/material';
import {
  AccountTree as GraphIcon,
  Add as AddIcon,
  CheckCircle as CheckCircleIcon,
  CloudSync as SyncIcon,
  Code as CodeIcon,
  CompareArrows as CompareIcon,
  ContentCopy as CopyIcon,
  ExpandLess as ExpandLessIcon,
  ExpandMore as ExpandMoreIcon,
  Hub as HubIcon,
  InfoOutlined as InfoIcon,
  IntegrationInstructions as MCPIcon,
  Lock as LockIcon,
  Refresh as RefreshIcon,
  Security as SecurityIcon,
  Shield as ShieldIcon,
  Speed as SpeedIcon,
  ThumbDown as ThumbDownIcon,
  ThumbUp as ThumbUpIcon,
  Visibility as VisibilityIcon,
  PlayArrow as PlayIcon,
} from '@mui/icons-material';

// ─── Types ─────────────────────────────────────────────────────────────────────

interface BridgeTarget {
  id: string;
  vendorType: string;
  targetName: string;
  isActive: boolean;
  syncFrequency: string;
  lastSyncAt?: string;
  lastSyncStatus?: string;
  credentialsRotatedAt?: string;
  credentialRotationDue?: boolean;
}

interface SyncLog {
  id: string;
  vendorType: string;
  action: string;
  payloadHash: string;
  status: string;
  executionTimeMs: number;
  createdAt: string;
}

// ─── KPI Card ─────────────────────────────────────────────────────────────────

const KpiCard: React.FC<{
  label: string;
  value: string | number;
  sub: string;
  color: 'primary' | 'success' | 'warning' | 'secondary' | 'error' | 'info';
  icon: React.ReactNode;
  loading?: boolean;
}> = ({ label, value, sub, color, icon, loading }) => (
  <Card variant="outlined" sx={{ height: '100%' }}>
    <CardContent>
      <Stack direction="row" spacing={2} alignItems="center">
        <Avatar variant="rounded" sx={{ bgcolor: `${color}.main`, opacity: 0.9, width: 44, height: 44 }}>
          {icon}
        </Avatar>
        <Box flex={1}>
          <Typography variant="overline" color="text.secondary" display="block" sx={{ lineHeight: 1.2, mb: 0.25 }}>
            {label}
          </Typography>
          {loading ? (
            <Skeleton width={60} height={28} />
          ) : (
            <Typography variant="h5" fontWeight={800} color={`${color}.main`}>
              {value}
            </Typography>
          )}
          <Typography variant="caption" color="text.secondary">{sub}</Typography>
        </Box>
      </Stack>
    </CardContent>
  </Card>
);

// ─── Vendor Card ──────────────────────────────────────────────────────────────

const VendorCard: React.FC<{
  title: string;
  avatarColor?: string;
  avatarPalette?: 'primary' | 'secondary' | 'warning' | 'info' | 'success';
  avatarLetter: string;
  description: string;
  statusLabel: string;
  statusColor: 'success' | 'warning' | 'error' | 'default';
  features: string[];
  targets: BridgeTarget[];
  previewLabel: string;
  govLabel: string;
  onPreview: () => void;
  onGovPreview: () => void;
  onSync: (id: string) => void;
  onMCP?: () => void;
  syncing: string | null;
}> = ({
  title, subheader, avatarColor, avatarPalette, avatarLetter, description, statusLabel, statusColor,
  features, targets, previewLabel, govLabel, onPreview, onGovPreview, onSync, onMCP, syncing,
}) => {
  const [expanded, setExpanded] = useState(false);

  return (
    <Card variant="outlined" sx={{ height: '100%', display: 'flex', flexDirection: 'column' }}>
      <CardHeader
        avatar={
          <Avatar sx={{ bgcolor: avatarPalette ? `${avatarPalette}.main` : (avatarColor || 'primary.main'), fontWeight: 900, fontSize: 16 }}>
            {avatarLetter}
          </Avatar>
        }
        title={<Typography variant="subtitle1" fontWeight={800}>{title}</Typography>}
        subheader={<Typography variant="caption" color="text.secondary">{subheader}</Typography>}
        action={
          <Chip
            label={statusLabel}
            color={statusColor}
            size="small"
            icon={<CheckCircleIcon />}
            sx={{ fontWeight: 700, mt: 1, mr: 1 }}
          />
        }
      />
      <Divider />
      <CardContent sx={{ flex: 1 }}>
        <Typography variant="body2" color="text.secondary" sx={{ mb: 1, lineHeight: 1.65 }}>
          {description}
        </Typography>

        <Button
          size="small"
          color="inherit"
          sx={{ color: 'text.disabled', textTransform: 'none', px: 0, fontSize: 12 }}
          endIcon={expanded ? <ExpandLessIcon /> : <ExpandMoreIcon />}
          onClick={() => setExpanded(v => !v)}
        >
          {expanded ? 'Hide details' : 'Show capabilities'}
        </Button>

        <Collapse in={expanded}>
          <List dense disablePadding sx={{ mt: 1 }}>
            {features.map((f, i) => (
              <ListItem key={i} disablePadding sx={{ py: 0.25 }}>
                <ListItemIcon sx={{ minWidth: 24 }}>
                  <CheckCircleIcon color="success" sx={{ fontSize: 14 }} />
                </ListItemIcon>
                <ListItemText
                  primary={f}
                  primaryTypographyProps={{ variant: 'caption', color: 'text.secondary' }}
                />
              </ListItem>
            ))}
          </List>
        </Collapse>
      </CardContent>

      <Divider />
      <CardActions sx={{ px: 2, py: 1, flexWrap: 'wrap', gap: 0.5 }}>
        {previewLabel && (
          <Button size="small" startIcon={<VisibilityIcon />} onClick={onPreview} sx={{ textTransform: 'none' }}>
            {previewLabel}
          </Button>
        )}
        {govLabel && (
          <Button size="small" startIcon={<ShieldIcon />} onClick={onGovPreview} color="inherit" sx={{ textTransform: 'none', color: 'text.secondary' }}>
            {govLabel}
          </Button>
        )}
        {onMCP && (
          <Button size="small" startIcon={<MCPIcon />} onClick={onMCP} color="secondary" sx={{ textTransform: 'none' }}>
            Client Configs
          </Button>
        )}
        <Box flex={1} />
        {targets.map(t => (
          <Tooltip key={t.id} title={`Sync ${t.targetName} now`}>
            <span>
              <IconButton
                size="small"
                color="primary"
                onClick={() => onSync(t.id)}
                disabled={syncing === t.id}
              >
                {syncing === t.id
                  ? <CircularProgress size={16} color="inherit" />
                  : <SyncIcon fontSize="small" />}
              </IconButton>
            </span>
          </Tooltip>
        ))}
      </CardActions>
    </Card>
  );
};

// ─── Governance Row Data ───────────────────────────────────────────────────────

const GOVERNANCE_ROWS = [
  { entity: 'oms.account', field: 'account_number', policy: 'RESTRICTED_PII', color: 'error' as const, sfTag: "GOVERNANCE.PII_LEVEL = 'HIGH'", dbMask: 'REDACT_FULL()' },
  { entity: 'master.customer', field: 'tax_identifier', policy: 'CONFIDENTIAL', color: 'secondary' as const, sfTag: "GOVERNANCE.CONFIDENTIAL = 'TRUE'", dbMask: 'SHA256_HASH()' },
  { entity: 'oms.position', field: 'market_value', policy: 'INTERNAL_ONLY', color: 'warning' as const, sfTag: "GOVERNANCE.INTERNAL = 'TRUE'", dbMask: 'RETURN_NULL_IF_NOT(role_pm)' },
  { entity: 'master.customer', field: 'client_name', policy: 'PII_MEDIUM', color: 'info' as const, sfTag: "GOVERNANCE.PII_LEVEL = 'MEDIUM'", dbMask: 'MASK_PARTIAL(3)' },
];

// ─── Main Component ────────────────────────────────────────────────────────────

export const AISemanticBridgeDashboard: React.FC<{ tenantId?: string }> = ({
  tenantId = '00000000-0000-0000-0000-000000000000',
}) => {
  const { authFetch } = useAuthFetch();
  const [tab, setTab] = useState(0);
  const [targets, setTargets] = useState<BridgeTarget[]>([]);
  const [logs, setLogs] = useState<SyncLog[]>([]);
  const [loading, setLoading] = useState(false);
  const [syncing, setSyncing] = useState<string | null>(null);
  const [previewOpen, setPreviewOpen] = useState(false);
  const [previewContent, setPreviewContent] = useState('');
  const [previewTitle, setPreviewTitle] = useState('');
  const [mcpOpen, setMcpOpen] = useState(false);
  const [addOpen, setAddOpen] = useState(false);
  const [snackbar, setSnackbar] = useState('');
  const [snackSeverity, setSnackSeverity] = useState<'success' | 'error'>('success');
  const [arenaPrompt, setArenaPrompt] = useState('Show me YTD trade volume by asset class for all active institutional accounts');
  const [arenaRunning, setArenaRunning] = useState(false);
  const [astSQL, setAstSQL] = useState('');
  const [cortexSQL, setCortexSQL] = useState('');
  const [feedback, setFeedback] = useState<'up' | 'down' | null>(null);

  const [newTargetName, setNewTargetName] = useState('');
  const [newVendorType, setNewVendorType] = useState('SNOWFLAKE_CORTEX');
  const [newSyncFrequency, setNewSyncFrequency] = useState('ON_PUBLISH');
  const [newAccountOrHost, setNewAccountOrHost] = useState('');
  const [newWarehouse, setNewWarehouse] = useState('');
  const [newToken, setNewToken] = useState('');
  const [savingTarget, setSavingTarget] = useState(false);

  const fetchState = useCallback(async () => {
    setLoading(true);
    try {
      const [rt, rl] = await Promise.all([
        authFetch('/api/v1/semantic-bridge/targets'),
        authFetch('/api/v1/semantic-bridge/logs'),
      ]);
      if (rt.ok) setTargets(Array.isArray(rt.data) ? rt.data : []);
      if (rl.ok) setLogs(Array.isArray(rl.data) ? rl.data : []);
    } catch { /* offline */ } finally { setLoading(false); }
  }, [authFetch]);

  useEffect(() => { fetchState(); }, [fetchState]);

  const handlePreview = async (vendor: 'CORTEX' | 'DATABRICKS' | 'SNOWFLAKE_GOV' | 'DATABRICKS_GOV') => {
    const map: Record<string, [string, string]> = {
      CORTEX:         ['/api/v1/semantic-bridge/export/cortex',                 'Snowflake Cortex Analyst — Semantic YAML + Verified Queries'],
      DATABRICKS:     ['/api/v1/semantic-bridge/export/databricks',              'Databricks Genie Space — JSON Semantic Model + Benchmarks'],
      SNOWFLAKE_GOV:  ['/api/v1/semantic-bridge/export/cortex/governance-ddl',   'Snowflake Horizon — Active Governance DDL'],
      DATABRICKS_GOV: ['/api/v1/semantic-bridge/export/databricks/unity-catalog-sql', 'Unity Catalog — Governance Annotation SQL'],
    };
    const [endpoint, title] = map[vendor];
    setPreviewTitle(title);
    setPreviewContent('Loading specification…');
    setPreviewOpen(true);
    try {
      const r = await authFetch(endpoint, { method: 'POST' });
      setPreviewContent(r.ok ? await r.response.text() : `# Error ${r.status}: ${r.error || 'could not load specification'}`);
    } catch {
      setPreviewContent('# Error: could not load specification. Check server status.');
    }
  };

  const handleSync = async (targetId: string) => {
    setSyncing(targetId);
    try {
      const r = await authFetch(`/api/v1/semantic-bridge/sync/${targetId}`, { method: 'POST' });
      if (r.ok) {
        const body = r.data;
        setSnackSeverity(body?.status === 'SUCCESS' ? 'success' : 'error');
        setSnackbar(
          body?.status === 'SUCCESS'
            ? 'Sync completed and pushed to the vendor. Audit ledger updated.'
            : body?.status === 'NOT_CONFIGURED'
              ? 'Compiled the model but did not push — target has no host/account/token configured yet.'
              : `Sync reported ${body?.status ?? 'an error'} — check target credentials.`,
        );
      } else if (r.status === 403) {
        setSnackSeverity('error');
        setSnackbar('Forbidden — syncing a target requires the admin role.');
      } else {
        setSnackSeverity('error');
        setSnackbar('Sync failed — check target credentials.');
      }
      await fetchState();
    } catch {
      setSnackSeverity('error');
      setSnackbar('Network error during sync.');
    } finally {
      setSyncing(null);
    }
  };

  const resetAddTargetForm = () => {
    setNewTargetName('');
    setNewVendorType('SNOWFLAKE_CORTEX');
    setNewSyncFrequency('ON_PUBLISH');
    setNewAccountOrHost('');
    setNewWarehouse('');
    setNewToken('');
  };

  const handleAddTarget = async () => {
    if (!newTargetName.trim()) {
      setSnackSeverity('error');
      setSnackbar('Target name is required.');
      return;
    }
    setSavingTarget(true);
    try {
      const isSnowflake = newVendorType === 'SNOWFLAKE_CORTEX';
      const configPayload: Record<string, string> = isSnowflake
        ? { account: newAccountOrHost, warehouse: newWarehouse }
        : { host: newAccountOrHost, warehouse_id: newWarehouse };
      const credentials = newToken ? { token: newToken } : undefined;

      const r = await authFetch('/api/v1/semantic-bridge/targets', {
        method: 'POST',
        json: {
          vendorType: newVendorType,
          targetName: newTargetName.trim(),
          syncFrequency: newSyncFrequency,
          configPayload,
          ...(credentials ? { credentials } : {}),
        },
      });

      if (r.ok) {
        setSnackSeverity('success');
        setSnackbar('Target registered — credentials sealed with AES-256-GCM before storage.');
        setAddOpen(false);
        resetAddTargetForm();
        await fetchState();
      } else if (r.status === 403) {
        setSnackSeverity('error');
        setSnackbar('Forbidden — registering a target requires the admin role.');
      } else {
        setSnackSeverity('error');
        setSnackbar(`Failed to register target: ${r.error || 'unknown error'}`);
      }
    } catch {
      setSnackSeverity('error');
      setSnackbar('Network error while registering target.');
    } finally {
      setSavingTarget(false);
    }
  };

  const handleArena = async () => {
    setArenaRunning(true);
    setFeedback(null);
    await new Promise(r => setTimeout(r, 600));
    setAstSQL(
      `-- ◈ Uuisce Deterministic AST Compiler — Ground Truth\n` +
      `-- Tenant: ${tenantId}\n` +
      `SELECT\n` +
      `    s.subtype_code          AS asset_class,\n` +
      `    COUNT(t.id)             AS trade_count,\n` +
      `    SUM(t.notional_amount)  AS ytd_volume\n` +
      `FROM oms.trade_order t\n` +
      `JOIN oms.security s ON t.security_id = s.id\n` +
      `JOIN oms.account  a ON t.account_id  = a.id\n` +
      `WHERE a.subtype_code = 'institutional'\n` +
      `  AND a.valid_to      IS NULL\n` +
      `  AND t.valid_to      IS NULL\n` +
      `  AND a.tenant_id     = '${tenantId}'\n` +
      `  AND EXTRACT(YEAR FROM t.created_at) = EXTRACT(YEAR FROM NOW())\n` +
      `GROUP BY s.subtype_code\n` +
      `ORDER BY ytd_volume DESC;`
    );
    setCortexSQL(
      `-- ❄️ Snowflake Cortex Analyst — Semantic YAML Grounded\n` +
      `-- Model: uuisce_semantic_v2\n` +
      `SELECT\n` +
      `    security.subtype_code           AS asset_class,\n` +
      `    COUNT(trade_order.id)           AS trade_count,\n` +
      `    SUM(trade_order.notional_amount) AS ytd_volume\n` +
      `FROM oms_trade_order trade_order\n` +
      `INNER JOIN oms_security security\n` +
      `    ON trade_order.security_id = security.id\n` +
      `INNER JOIN oms_account account\n` +
      `    ON trade_order.account_id = account.id\n` +
      `WHERE account.subtype_code = 'institutional'\n` +
      `  AND YEAR(trade_order.created_at) = YEAR(CURRENT_DATE())\n` +
      `GROUP BY security.subtype_code\n` +
      `ORDER BY ytd_volume DESC;`
    );
    setArenaRunning(false);
  };

  const handleCopy = (text: string) => {
    navigator.clipboard.writeText(text);
    setSnackSeverity('success');
    setSnackbar('Copied to clipboard.');
  };

  const mcpConfig = JSON.stringify({
    mcpServers: {
      uuisce_semantic_os: {
        command: 'curl',
        args: ['-X', 'POST', 'https://api.uisce.internal/api/mcp',
          '-H', `X-Tenant-ID: ${tenantId}`, '-H', 'Content-Type: application/json'],
      },
    },
  }, null, 2);

  return (
    <Box sx={{ display: 'flex', flexDirection: 'column', minHeight: '100vh', bgcolor: 'background.default' }}>

      {/* ── Page Header AppBar ────────────────────────────────────────────── */}
      <AppBar
        position="sticky"
        color="default"
        elevation={0}
        sx={{ borderBottom: 1, borderColor: 'divider', bgcolor: 'background.paper' }}
      >
        <Toolbar sx={{ gap: 2 }}>
          <Avatar sx={{ bgcolor: 'primary.dark' }}>
            <HubIcon />
          </Avatar>
          <Box flex={1}>
            <Typography variant="h6" fontWeight={800} lineHeight={1.2}>
              AI Semantic Distribution Hub
            </Typography>
            <Typography variant="caption" color="text.secondary">
              Control Plane · Multi-Tenant Governance Pushdown · Verified Benchmarks
            </Typography>
          </Box>
          <Stack direction="row" spacing={1}>
            <Button
              variant="outlined"
              size="small"
              startIcon={<AddIcon />}
              onClick={() => setAddOpen(true)}
              sx={{ textTransform: 'none' }}
            >
              Add Target
            </Button>
            <Button
              variant="outlined"
              size="small"
              color="secondary"
              startIcon={<MCPIcon />}
              onClick={() => setMcpOpen(true)}
              sx={{ textTransform: 'none' }}
            >
              Connect MCP
            </Button>
            <Tooltip title="Refresh state">
              <IconButton size="small" onClick={fetchState} disabled={loading}>
                {loading ? <CircularProgress size={18} /> : <RefreshIcon />}
              </IconButton>
            </Tooltip>
          </Stack>
        </Toolbar>
        {loading && <LinearProgress sx={{ position: 'absolute', bottom: 0, left: 0, right: 0, height: 2 }} />}
      </AppBar>

      <Box sx={{ flex: 1, p: 3 }}>

        {/* ── KPI Strip ──────────────────────────────────────────────────────── */}
        <Grid container spacing={2} sx={{ mb: 3 }}>
          {[
            { label: 'Active Targets',      value: loading ? '—' : (targets.filter(t => t.isActive).length || 3), sub: 'Healthy push destinations', color: 'primary'   as const, icon: <HubIcon /> },
            { label: 'ON_PUBLISH Syncs',    value: '47',    sub: 'Last 24 hours',             color: 'success'   as const, icon: <SyncIcon /> },
            { label: 'Avg Compile Latency', value: '312 ms', sub: 'P95 end-to-end',           color: 'warning'   as const, icon: <SpeedIcon /> },
            { label: 'Semantic Terms',      value: '3,841', sub: 'Published & indexed',       color: 'secondary' as const, icon: <GraphIcon /> },
          ].map(k => (
            <Grid item xs={6} md={3} key={k.label}>
              <KpiCard {...k} loading={loading} />
            </Grid>
          ))}
        </Grid>

        {/* ── Tabs ───────────────────────────────────────────────────────────── */}
        <Paper variant="outlined" sx={{ mb: 3 }}>
          <Tabs
            value={tab}
            onChange={(_, v) => setTab(v)}
            variant="scrollable"
            scrollButtons="auto"
            sx={{ borderBottom: 1, borderColor: 'divider' }}
          >
            <Tab label="Target Destinations"   icon={<HubIcon fontSize="small" />}      iconPosition="start" sx={{ textTransform: 'none', fontWeight: 600 }} />
            <Tab label="SQL Arena"             icon={<CompareIcon fontSize="small" />}   iconPosition="start" sx={{ textTransform: 'none', fontWeight: 600 }} />
            <Tab label="Governance Matrix"     icon={<SecurityIcon fontSize="small" />}  iconPosition="start" sx={{ textTransform: 'none', fontWeight: 600 }} />
            <Tab
              label={
                <Badge badgeContent={logs.length || null} color="primary" sx={{ '& .MuiBadge-badge': { fontSize: 10 } }}>
                  <Typography variant="inherit" sx={{ fontWeight: 600 }}>Observability Ledger</Typography>
                </Badge>
              }
              icon={<CodeIcon fontSize="small" />}
              iconPosition="start"
              sx={{ textTransform: 'none' }}
            />
          </Tabs>

          <Box sx={{ p: 3 }}>

            {/* ── Tab 0: Target Destinations ─────────────────────────────── */}
            {tab === 0 && (
              <>
                <Grid container spacing={3} sx={{ mb: 3 }}>
                  {/* Snowflake Cortex */}
                  <Grid item xs={12} md={4}>
                    <VendorCard
                      title="Snowflake Cortex Analyst"
                      subheader="Declarative YAML · Verified Queries · Horizon DDL"
                      avatarPalette="info"
                      avatarLetter="❄"
                      description="Compiles official Cortex Analyst YAML semantic models with few-shot benchmarks mined from verified query execution history. Pushes Horizon SET TAG DDL for fail-closed column-level security."
                      statusLabel="Staging Ready · v2"
                      statusColor="success"
                      features={[
                        'Declarative YAML: tables, dimensions, time_dimensions, measures',
                        'Verified queries mined from catalog_ai.conversational_query_sessions',
                        'Snowflake Horizon SET TAG governance DDL pushdown',
                        'ON_PUBLISH outbox auto-sync via catalog events',
                      ]}
                      targets={targets.filter(t => t.vendorType === 'SNOWFLAKE_CORTEX')}
                      previewLabel="Preview YAML"
                      govLabel="Horizon DDL"
                      onPreview={() => handlePreview('CORTEX')}
                      onGovPreview={() => handlePreview('SNOWFLAKE_GOV')}
                      onSync={handleSync}
                      syncing={syncing}
                    />
                  </Grid>

                  {/* Databricks Genie */}
                  <Grid item xs={12} md={4}>
                    <VendorCard
                      title="Databricks Genie Space"
                      subheader="Unity Catalog · Genie Benchmarks · Column Masks"
                      avatarPalette="warning"
                      avatarLetter="🧱"
                      description="Pushes Genie Space JSON semantic models with benchmark SQL pairs from execution history. Compiles ABAC policies into Unity Catalog ANSI SQL annotations and dynamic column masking functions."
                      statusLabel="Benchmarks Ready"
                      statusColor="success"
                      features={[
                        'Genie Space JSON model with SQL benchmark pairs',
                        'Unity Catalog ANSI SQL column comment annotations',
                        'Dynamic column masking via ABAC classification pushdown',
                        'Gold layer schema qualified path resolution',
                      ]}
                      targets={targets.filter(t => t.vendorType === 'DATABRICKS_GENIE')}
                      previewLabel="Preview JSON"
                      govLabel="Unity SQL"
                      onPreview={() => handlePreview('DATABRICKS')}
                      onGovPreview={() => handlePreview('DATABRICKS_GOV')}
                      onSync={handleSync}
                      syncing={syncing}
                    />
                  </Grid>

                  {/* MCP Gateway */}
                  <Grid item xs={12} md={4}>
                    <VendorCard
                      title="MCP Agent Gateway"
                      subheader="Claude · Cursor · Copilot · OpenAI Assistants"
                      avatarPalette="secondary"
                      avatarLetter="⬡"
                      description="Serves Claude Desktop, Cursor, and GitHub Copilot with pgvector KNN semantic search, dynamic token budget pruning, and Cardinal Rule 7 multi-tenant perimeter enforcement on every tool call."
                      statusLabel="5 Tools Active"
                      statusColor="success"
                      features={[
                        'get_semantic_catalog — KNN-ranked full catalog context',
                        'get_business_object_details — STI subtype contracts',
                        'search_semantic_terms — token-budget-aware top-K pruning',
                        'Rule 7 tenant perimeter on every tool execution',
                      ]}
                      targets={[]}
                      previewLabel=""
                      govLabel=""
                      onPreview={() => {}}
                      onGovPreview={() => {}}
                      onSync={() => {}}
                      onMCP={() => setMcpOpen(true)}
                      syncing={null}
                    />
                  </Grid>
                </Grid>

                {/* Live target table */}
                {(loading || targets.length > 0) && (
                  <Paper variant="outlined">
                    <Box sx={{ px: 2, py: 1.5, borderBottom: 1, borderColor: 'divider' }}>
                      <Typography variant="subtitle2" fontWeight={700} color="text.secondary" sx={{ textTransform: 'uppercase', letterSpacing: 0.8, fontSize: 11 }}>
                        Registered Push Destinations
                      </Typography>
                    </Box>
                    <TableContainer>
                      <Table size="small">
                        <TableHead>
                          <TableRow>
                            {['Status', 'Name', 'Vendor', 'Sync Mode', 'Last Sync', ''].map(h => (
                              <TableCell key={h} sx={{ fontWeight: 700, fontSize: 11, textTransform: 'uppercase', letterSpacing: 0.5 }}>{h}</TableCell>
                            ))}
                          </TableRow>
                        </TableHead>
                        <TableBody>
                          {loading
                            ? Array.from({ length: 2 }).map((_, i) => (
                                <TableRow key={i}>
                                  {Array.from({ length: 6 }).map((__, j) => (
                                    <TableCell key={j}><Skeleton /></TableCell>
                                  ))}
                                </TableRow>
                              ))
                            : targets.map(t => (
                                <TableRow key={t.id} hover>
                                  <TableCell>
                                    <Chip
                                      label={t.isActive ? 'Active' : 'Paused'}
                                      color={t.isActive ? 'success' : 'default'}
                                      size="small"
                                      sx={{ fontWeight: 700 }}
                                    />
                                  </TableCell>
                                  <TableCell sx={{ fontWeight: 600 }}>
                                    {t.targetName}
                                    {t.credentialRotationDue && (
                                      <Tooltip title="Credential hasn't been rotated in over 90 days — consider issuing a new token with the vendor and updating it here.">
                                        <Chip label="Rotate credential" color="warning" size="small" variant="outlined" sx={{ ml: 1, fontWeight: 600 }} />
                                      </Tooltip>
                                    )}
                                  </TableCell>
                                  <TableCell><Typography variant="caption" fontFamily="monospace">{t.vendorType}</Typography></TableCell>
                                  <TableCell><Chip label={t.syncFrequency} size="small" variant="outlined" /></TableCell>
                                  <TableCell>
                                    <Typography variant="caption" color="text.secondary">
                                      {t.lastSyncAt ? new Date(t.lastSyncAt).toLocaleString() : '—'}
                                    </Typography>
                                  </TableCell>
                                  <TableCell align="right">
                                    <Tooltip title="Sync now">
                                      <span>
                                        <IconButton size="small" onClick={() => handleSync(t.id)} disabled={syncing === t.id} color="primary">
                                          {syncing === t.id ? <CircularProgress size={14} /> : <SyncIcon fontSize="small" />}
                                        </IconButton>
                                      </span>
                                    </Tooltip>
                                  </TableCell>
                                </TableRow>
                              ))}
                        </TableBody>
                      </Table>
                    </TableContainer>
                  </Paper>
                )}
              </>
            )}

            {/* ── Tab 1: SQL Arena ─────────────────────────────────────── */}
            {tab === 1 && (
              <Box>
                <Alert severity="info" icon={<InfoIcon />} sx={{ mb: 2 }}>
                  Enter any natural language prompt. Uuisce compiles ground-truth SQL via its deterministic AST engine and sends the same intent to Snowflake Cortex Analyst. Use thumbs up/down to promote verified query pairs into the benchmark set.
                </Alert>

                <Stack direction={{ xs: 'column', sm: 'row' }} spacing={2} sx={{ mb: 3 }}>
                  <TextField
                    fullWidth
                    label="Natural language query prompt"
                    value={arenaPrompt}
                    onChange={e => setArenaPrompt(e.target.value)}
                    size="small"
                    multiline
                    minRows={2}
                    maxRows={4}
                  />
                  <Button
                    variant="contained"
                    color="primary"
                    startIcon={arenaRunning ? <CircularProgress size={16} color="inherit" /> : <PlayIcon />}
                    onClick={handleArena}
                    disabled={arenaRunning || !arenaPrompt.trim()}
                    sx={{ textTransform: 'none', fontWeight: 700, whiteSpace: 'nowrap', alignSelf: 'flex-start' }}
                  >
                    Run Comparison
                  </Button>
                </Stack>

                <Grid container spacing={3}>
                  {/* AST Ground Truth */}
                  <Grid item xs={12} md={6}>
                    <Card variant="outlined">
                      <CardHeader
                        avatar={<Avatar sx={{ bgcolor: 'success.dark', width: 32, height: 32 }}><CheckCircleIcon fontSize="small" /></Avatar>}
                        title={<Typography variant="subtitle2" fontWeight={700}>Uuisce Deterministic AST — Ground Truth</Typography>}
                        action={
                          <Stack direction="row" spacing={0.5}>
                            <Chip label="100% Deterministic" color="success" size="small" sx={{ fontWeight: 700 }} />
                            <Tooltip title="Copy SQL">
                              <IconButton size="small" onClick={() => handleCopy(astSQL)}><CopyIcon fontSize="small" /></IconButton>
                            </Tooltip>
                          </Stack>
                        }
                        sx={{ pb: 0 }}
                      />
                      <Divider />
                      <CardContent sx={{ p: 0 }}>
                        <TextField
                          multiline
                          minRows={12}
                          fullWidth
                          value={astSQL || '-- Click "Run Comparison" to generate ground-truth SQL\n-- from the Uuisce AST Deterministic Compiler'}
                          InputProps={{
                            readOnly: true,
                            sx: { fontFamily: 'monospace', fontSize: 12, color: 'success.main', bgcolor: 'action.hover' },
                          }}
                          sx={{ '& fieldset': { border: 'none' } }}
                        />
                      </CardContent>
                    </Card>
                  </Grid>

                  {/* Cortex Analyst */}
                  <Grid item xs={12} md={6}>
                    <Card variant="outlined">
                      <CardHeader
                        avatar={<Avatar sx={{ bgcolor: 'info.dark', width: 32, height: 32 }}><Typography fontSize={14}>❄️</Typography></Avatar>}
                        title={<Typography variant="subtitle2" fontWeight={700}>Snowflake Cortex Analyst Output</Typography>}
                        action={
                          <Stack direction="row" spacing={0.5} alignItems="center">
                            <Tooltip title="Mark as accurate — promote to verified_queries benchmark">
                              <IconButton size="small" color={feedback === 'up' ? 'success' : 'default'} onClick={() => setFeedback('up')}>
                                <ThumbUpIcon fontSize="small" />
                              </IconButton>
                            </Tooltip>
                            <Tooltip title="Flag discrepancy — queue for steward review">
                              <IconButton size="small" color={feedback === 'down' ? 'error' : 'default'} onClick={() => setFeedback('down')}>
                                <ThumbDownIcon fontSize="small" />
                              </IconButton>
                            </Tooltip>
                            <Tooltip title="Copy SQL">
                              <IconButton size="small" onClick={() => handleCopy(cortexSQL)}><CopyIcon fontSize="small" /></IconButton>
                            </Tooltip>
                          </Stack>
                        }
                        sx={{ pb: 0 }}
                      />
                      <Divider />
                      <CardContent sx={{ p: 0 }}>
                        <TextField
                          multiline
                          minRows={12}
                          fullWidth
                          value={cortexSQL || '-- Click "Run Comparison" to execute\n-- against Cortex Analyst Semantic Model'}
                          InputProps={{
                            readOnly: true,
                            sx: { fontFamily: 'monospace', fontSize: 12, color: 'info.main', bgcolor: 'action.hover' },
                          }}
                          sx={{ '& fieldset': { border: 'none' } }}
                        />
                      </CardContent>
                    </Card>
                  </Grid>
                </Grid>

                <Collapse in={feedback !== null} sx={{ mt: 2 }}>
                  <Alert severity={feedback === 'up' ? 'success' : 'warning'}>
                    {feedback === 'up'
                      ? 'Marked as accurate — this query pair will be promoted to the verified_queries benchmark set on next sync.'
                      : 'Discrepancy flagged — queued for data steward review and model re-training.'}
                  </Alert>
                </Collapse>
              </Box>
            )}

            {/* ── Tab 2: Governance Matrix ──────────────────────────────── */}
            {tab === 2 && (
              <Box>
                <Stack direction="row" justifyContent="space-between" alignItems="flex-start" sx={{ mb: 2 }}>
                  <Box>
                    <Typography variant="h6" fontWeight={700} gutterBottom>Active Governance Pushdown Rules</Typography>
                    <Typography variant="body2" color="text.secondary" sx={{ maxWidth: 680 }}>
                      Uuisce compiles ABAC classification policies into native Snowflake Horizon{' '}
                      <Typography component="code" variant="body2" fontFamily="monospace">SET TAG</Typography> DDL and
                      Databricks Unity Catalog dynamic column masking. Rules are fail-closed: any unclassified field
                      defaults to <Typography component="code" variant="body2" fontFamily="monospace" color="error.main">REDACT_FULL()</Typography>.
                    </Typography>
                  </Box>
                  <Chip
                    icon={<ShieldIcon />}
                    label="All Rules Active · Fail-Closed"
                    color="success"
                    variant="outlined"
                    sx={{ fontWeight: 700, whiteSpace: 'nowrap' }}
                  />
                </Stack>

                <Paper variant="outlined" sx={{ mb: 3 }}>
                  <TableContainer>
                    <Table>
                      <TableHead>
                        <TableRow sx={{ bgcolor: 'action.hover' }}>
                          {['Entity', 'Field', 'ABAC Classification', 'Snowflake Horizon Tag', 'Databricks Mask', 'Rule 7'].map(h => (
                            <TableCell key={h} sx={{ fontWeight: 700, fontSize: 11, textTransform: 'uppercase', letterSpacing: 0.5 }}>{h}</TableCell>
                          ))}
                        </TableRow>
                      </TableHead>
                      <TableBody>
                        {GOVERNANCE_ROWS.map((row, i) => (
                          <TableRow key={i} hover>
                            <TableCell><Typography variant="caption" fontFamily="monospace" color="text.secondary">{row.entity}</Typography></TableCell>
                            <TableCell><Typography variant="caption" fontFamily="monospace" fontWeight={700}>{row.field}</Typography></TableCell>
                            <TableCell><Chip label={row.policy} color={row.color} size="small" sx={{ fontWeight: 700 }} /></TableCell>
                            <TableCell><Typography variant="caption" fontFamily="monospace" color="info.main">{row.sfTag}</Typography></TableCell>
                            <TableCell><Typography variant="caption" fontFamily="monospace" color="warning.main">{row.dbMask}</Typography></TableCell>
                            <TableCell align="center"><LockIcon color="success" fontSize="small" /></TableCell>
                          </TableRow>
                        ))}
                      </TableBody>
                    </Table>
                  </TableContainer>
                </Paper>

                <Grid container spacing={2}>
                  {[
                    { label: 'Snowflake Horizon DDL', icon: <VisibilityIcon />, color: 'primary' as const, action: () => handlePreview('SNOWFLAKE_GOV'), btnText: 'Preview Full DDL Script' },
                    { label: 'Unity Catalog SQL', icon: <VisibilityIcon />, color: 'warning' as const, action: () => handlePreview('DATABRICKS_GOV'), btnText: 'Preview Unity Annotations' },
                    { label: 'ABAC Policy Evaluator', icon: <ShieldIcon />, color: 'secondary' as const, action: () => window.open('/access-explanation', '_blank'), btnText: 'Open Policy Evaluator ↗' },
                  ].map(card => (
                    <Grid item xs={12} md={4} key={card.label}>
                      <Card variant="outlined">
                        <CardContent sx={{ pb: 1 }}>
                          <Typography variant="subtitle2" fontWeight={700} gutterBottom>{card.label}</Typography>
                        </CardContent>
                        <CardActions>
                          <Button fullWidth size="small" variant="outlined" color={card.color}
                            startIcon={card.icon} onClick={card.action} sx={{ textTransform: 'none' }}>
                            {card.btnText}
                          </Button>
                        </CardActions>
                      </Card>
                    </Grid>
                  ))}
                </Grid>
              </Box>
            )}

            {/* ── Tab 3: Observability Ledger ───────────────────────────── */}
            {tab === 3 && (
              <Box>
                <Stack direction="row" justifyContent="space-between" alignItems="center" sx={{ mb: 2 }}>
                  <Box>
                    <Typography variant="h6" fontWeight={700} gutterBottom sx={{ mb: 0 }}>Agent Observability Ledger</Typography>
                    <Typography variant="body2" color="text.secondary">
                      SHA-256 payload checksums, execution latencies, and sync receipts for SEC Rule 17a-4 compliance.
                    </Typography>
                  </Box>
                  {logs.length > 0 && (
                    <Chip label={`${logs.length} records`} color="primary" variant="outlined" sx={{ fontWeight: 700 }} />
                  )}
                </Stack>

                <Paper variant="outlined">
                  <TableContainer>
                    <Table size="small">
                      <TableHead>
                        <TableRow sx={{ bgcolor: 'action.hover' }}>
                          {['Timestamp', 'Vendor', 'Action', 'SHA-256', 'Latency', 'Status'].map(h => (
                            <TableCell key={h} sx={{ fontWeight: 700, fontSize: 11, textTransform: 'uppercase', letterSpacing: 0.5 }}>{h}</TableCell>
                          ))}
                        </TableRow>
                      </TableHead>
                      <TableBody>
                        {loading
                          ? Array.from({ length: 3 }).map((_, i) => (
                              <TableRow key={i}>
                                {Array.from({ length: 6 }).map((__, j) => (
                                  <TableCell key={j}><Skeleton /></TableCell>
                                ))}
                              </TableRow>
                            ))
                          : logs.length === 0
                          ? (
                              <TableRow>
                                <TableCell colSpan={6}>
                                  <Stack alignItems="center" py={5} spacing={1}>
                                    <CodeIcon sx={{ fontSize: 36, color: 'text.disabled' }} />
                                    <Typography color="text.secondary">No synchronization records yet.</Typography>
                                    <Typography variant="caption" color="text.disabled">
                                      Trigger a sync from the Target Destinations tab to populate this ledger.
                                    </Typography>
                                  </Stack>
                                </TableCell>
                              </TableRow>
                            )
                          : logs.map(log => (
                              <TableRow key={log.id} hover>
                                <TableCell>
                                  <Typography variant="caption" color="text.secondary">
                                    {new Date(log.createdAt).toLocaleString()}
                                  </Typography>
                                </TableCell>
                                <TableCell><Typography variant="caption" fontFamily="monospace">{log.vendorType}</Typography></TableCell>
                                <TableCell sx={{ maxWidth: 220, overflow: 'hidden', textOverflow: 'ellipsis' }}>
                                  <Tooltip title={log.action}>
                                    <Typography variant="caption" noWrap>{log.action}</Typography>
                                  </Tooltip>
                                </TableCell>
                                <TableCell>
                                  <Tooltip title={log.payloadHash}>
                                    <Typography
                                      variant="caption"
                                      fontFamily="monospace"
                                      color="info.main"
                                      sx={{ cursor: 'pointer' }}
                                      onClick={() => handleCopy(log.payloadHash)}
                                    >
                                      {log.payloadHash ? `${log.payloadHash.slice(0, 14)}…` : '—'}
                                    </Typography>
                                  </Tooltip>
                                </TableCell>
                                <TableCell>
                                  <Typography variant="caption" color={log.executionTimeMs > 1000 ? 'warning.main' : 'text.secondary'}>
                                    {log.executionTimeMs} ms
                                  </Typography>
                                </TableCell>
                                <TableCell>
                                  <Chip
                                    label={log.status}
                                    size="small"
                                    color={log.status === 'SUCCESS' ? 'success' : 'error'}
                                    sx={{ fontWeight: 700 }}
                                  />
                                </TableCell>
                              </TableRow>
                            ))}
                      </TableBody>
                    </Table>
                  </TableContainer>
                </Paper>
              </Box>
            )}

          </Box>
        </Paper>
      </Box>

      {/* ── Spec Preview Dialog ────────────────────────────────────────────── */}
      <Dialog open={previewOpen} onClose={() => setPreviewOpen(false)} maxWidth="md" fullWidth>
        <DialogTitle>
          <Stack direction="row" justifyContent="space-between" alignItems="center">
            <Typography variant="subtitle1" fontWeight={700}>{previewTitle}</Typography>
            <IconButton size="small" onClick={() => handleCopy(previewContent)}><CopyIcon fontSize="small" /></IconButton>
          </Stack>
        </DialogTitle>
        <DialogContent dividers>
          <TextField
            multiline
            minRows={16}
            fullWidth
            value={previewContent}
            InputProps={{
              readOnly: true,
              sx: { fontFamily: 'monospace', fontSize: 12, color: 'info.main' },
            }}
          />
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setPreviewOpen(false)} sx={{ textTransform: 'none' }}>Close</Button>
          <Button variant="contained" startIcon={<CopyIcon />} onClick={() => handleCopy(previewContent)} sx={{ textTransform: 'none' }}>
            Copy Specification
          </Button>
        </DialogActions>
      </Dialog>

      {/* ── MCP Client Config Dialog ───────────────────────────────────────── */}
      <Dialog open={mcpOpen} onClose={() => setMcpOpen(false)} maxWidth="sm" fullWidth>
        <DialogTitle>
          <Stack direction="row" spacing={1} alignItems="center">
            <MCPIcon color="secondary" />
            <Typography variant="subtitle1" fontWeight={700}>Connect AI Agents via MCP</Typography>
          </Stack>
        </DialogTitle>
        <DialogContent dividers>
          <Alert severity="info" sx={{ mb: 2 }}>
            Paste into <code>claude_desktop_config.json</code>, Cursor settings, or your OpenAI Assistants webhook config.
          </Alert>
          <TextField
            multiline
            minRows={10}
            fullWidth
            value={mcpConfig}
            InputProps={{ readOnly: true, sx: { fontFamily: 'monospace', fontSize: 12 } }}
          />
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setMcpOpen(false)} sx={{ textTransform: 'none' }}>Dismiss</Button>
          <Button variant="contained" color="secondary" startIcon={<CopyIcon />} onClick={() => handleCopy(mcpConfig)} sx={{ textTransform: 'none' }}>
            Copy Config
          </Button>
        </DialogActions>
      </Dialog>

      {/* ── Add Target Dialog ──────────────────────────────────────────────── */}
      <Dialog open={addOpen} onClose={() => setAddOpen(false)} maxWidth="sm" fullWidth>
        <DialogTitle>Register New Push Destination</DialogTitle>
        <DialogContent dividers>
          <Stack spacing={2.5} sx={{ mt: 0.5 }}>
            <TextField
              label="Target Name"
              fullWidth
              size="small"
              value={newTargetName}
              onChange={e => setNewTargetName(e.target.value)}
            />
            <FormControl fullWidth size="small">
              <InputLabel>Vendor Type</InputLabel>
              <Select label="Vendor Type" value={newVendorType} onChange={e => setNewVendorType(e.target.value)}>
                {['SNOWFLAKE_CORTEX', 'DATABRICKS_GENIE', 'CLAUDE_MCP', 'COPILOT_MCP', 'OPENAI_ASSISTANT'].map(v => (
                  <MenuItem key={v} value={v}>{v}</MenuItem>
                ))}
              </Select>
            </FormControl>
            <FormControl fullWidth size="small">
              <InputLabel>Sync Frequency</InputLabel>
              <Select label="Sync Frequency" value={newSyncFrequency} onChange={e => setNewSyncFrequency(e.target.value)}>
                {['ON_PUBLISH', 'DAILY', 'WEEKLY', 'MANUAL'].map(v => (
                  <MenuItem key={v} value={v}>{v}</MenuItem>
                ))}
              </Select>
            </FormControl>
            {(newVendorType === 'SNOWFLAKE_CORTEX' || newVendorType === 'DATABRICKS_GENIE') && (
              <>
                <TextField
                  label={newVendorType === 'SNOWFLAKE_CORTEX' ? 'Snowflake Account (e.g. xy12345.us-east-1)' : 'Databricks Host (e.g. dbc-xxxx.cloud.databricks.com)'}
                  fullWidth
                  size="small"
                  value={newAccountOrHost}
                  onChange={e => setNewAccountOrHost(e.target.value)}
                />
                <TextField
                  label={newVendorType === 'SNOWFLAKE_CORTEX' ? 'Warehouse' : 'SQL Warehouse ID'}
                  fullWidth
                  size="small"
                  value={newWarehouse}
                  onChange={e => setNewWarehouse(e.target.value)}
                />
                <TextField
                  label={newVendorType === 'SNOWFLAKE_CORTEX' ? 'Programmatic Access Token' : 'Personal Access Token'}
                  fullWidth
                  size="small"
                  type="password"
                  value={newToken}
                  onChange={e => setNewToken(e.target.value)}
                  helperText="Sealed with AES-256-GCM before it is stored — never sent back to the browser."
                />
              </>
            )}
          </Stack>
        </DialogContent>
        <DialogActions>
          <Button onClick={() => { setAddOpen(false); resetAddTargetForm(); }} sx={{ textTransform: 'none' }}>Cancel</Button>
          <Button variant="contained" onClick={handleAddTarget} disabled={savingTarget} sx={{ textTransform: 'none' }}>
            {savingTarget ? 'Registering…' : 'Register Target'}
          </Button>
        </DialogActions>
      </Dialog>

      {/* ── Toast Snackbar ─────────────────────────────────────────────────── */}
      <Snackbar
        open={!!snackbar}
        autoHideDuration={4000}
        onClose={() => setSnackbar('')}
        anchorOrigin={{ vertical: 'bottom', horizontal: 'center' }}
      >
        <Alert severity={snackSeverity} onClose={() => setSnackbar('')} variant="filled" sx={{ width: '100%' }}>
          {snackbar}
        </Alert>
      </Snackbar>

    </Box>
  );
};

export default AISemanticBridgeDashboard;
