import React, { useState, useMemo, useCallback, useEffect } from 'react';
import {
  Box, Typography, Button, IconButton, TextField, InputBase, Tooltip, Chip, Tabs, Tab, Paper,
  Table, TableBody, TableCell, TableContainer, TableHead, TableRow, Divider, Snackbar, Alert,
  ToggleButtonGroup, ToggleButton, CircularProgress,
} from '@mui/material';
import {
  Search as SearchIcon, AutoAwesome as AutoAwesomeIcon, SyncAlt as SyncAltIcon,
  ViewList as ViewListIcon, Schema as SchemaIcon, Add as AddIcon, Verified as VerifiedIcon,
  Edit as EditIcon, History as HistoryIcon, Code as CodeIcon, MenuBook as MenuBookIcon,
  Hub as HubIcon, AltRoute as AltRouteIcon, Terminal as TerminalIcon,
  PlayArrow as PlayArrowIcon, OpenInFull as OpenInFullIcon, AddLink as AddLinkIcon,
  Apartment as ApartmentIcon, Key as KeyIcon, Functions as FunctionsIcon,
  Percent as PercentIcon, Label as LabelIcon, CalendarToday as CalendarTodayIcon,
  ShowChart as ShowChartIcon, MoreHoriz as MoreHorizIcon, ChevronRight as ChevronRightIcon,
} from '@mui/icons-material';
import AnalyticalShell from '../../components/analytical/AnalyticalShell';
import apiClient from '../../utils/apiClient';
import { useTenant } from '../../contexts/TenantContext';
import { devError, devLog } from '../../utils/devLogger';

interface BackendBO {
  id: string;
  key: string;
  name: string;
  displayName: string;
  description: string;
  category: string;
  isCore: boolean;
  instanceCount: number;
  driverTableName: string;
  bindings: Array<{ binding_id: string; backend_type: string; driving_node_name: string; is_default: boolean }>;
  coreFields: Array<{ id: string; name: string; technicalName: string; semanticTermId: string; type: string }>;
  customFields: Array<{ id: string; name: string; technicalName: string; semanticTermId: string; type: string }>;
}

interface BackendField {
  id: string;
  name: string;
  technicalName: string;
  semanticTermId: string;
  type: string;
}

interface DataInspectorResult {
  columns: string[];
  rows: Record<string, unknown>[];
  total: number;
  executionTimeMs: number;
  driverTable: string;
}

interface SemanticEntity {
  id: string;
  name: string;
  domain: string;
  description: string;
  tier: 'Gold' | 'Silver' | 'Bronze';
  isVerified: boolean;
  isCritical: boolean;
  physicalBinding: string;
  slaFreshness: string;
  custodian: string;
  storageEngine: string;
  totalFields: number;
  dimCount: number;
  measureCount: number;
  calcCount: number;
  downstreamQueries: number;
  downstreamCalls24h: string;
  activeDashboards: number;
  cacheHitRate: string;
  p95Latency: string;
}

interface SemanticField {
  id: string;
  key: string;
  type: 'TEXT' | 'NUM' | 'DATE' | 'BOOL' | 'CURR' | 'RATE' | 'CAT' | 'COUNT';
  description: string;
  formula: string;
  verified: boolean;
  isMeasure?: boolean;
}

interface JoinCard {
  id: string;
  cardinality: 'N:1' | '1:N' | '1:1';
  target: string;
  condition: string;
  meta: string;
  color: 'primary' | 'tertiary' | 'secondary';
}

interface DomainGroup {
  id: string;
  name: string;
  entities: { id: string; name: string; tier: 'Gold' | 'Silver' | 'Bronze' }[];
}

const DEFAULT_ENTITY: SemanticEntity = {
  id: '',
  name: 'Select a business object',
  domain: '',
  description: 'Loading...',
  tier: 'Gold',
  isVerified: false,
  isCritical: false,
  physicalBinding: '',
  slaFreshness: '',
  custodian: '',
  storageEngine: '',
  totalFields: 0,
  dimCount: 0,
  measureCount: 0,
  calcCount: 0,
  downstreamQueries: 0,
  downstreamCalls24h: '',
  activeDashboards: 0,
  cacheHitRate: '',
  p95Latency: '',
};

const DEFAULT_FIELDS: SemanticField[] = [];

const DEFAULT_JOINS: JoinCard[] = [
  { id: 'j1', cardinality: 'N:1', target: 'corp.issuer_entity',  condition: 'sec.issuer_id = iss.id',     meta: 'Latency: 0.1ms • Enforced Index', color: 'primary' },
  { id: 'j2', cardinality: '1:N', target: 'book.fund_holdings',  condition: 'sec.isin_code = hold.isin',   meta: 'Partition Pruned • Colocated',   color: 'tertiary' },
  { id: 'j3', cardinality: 'N:1', target: 'mkt.mic_venues',      condition: 'sec.primary_mic = mic.code',  meta: 'Broadcast Join • Cached Memory', color: 'secondary' },
];

const DEFAULT_DOMAINS: DomainGroup[] = [
  {
    id: 'equities',
    name: 'Equities & Derivatives',
    entities: [
      { id: 'sec-master',          name: 'Security Master',          tier: 'Gold' },
      { id: 'option-greeks',       name: 'Option Greeks Surface',    tier: 'Silver' },
      { id: 'corp-actions',        name: 'Corporate Actions Feed',   tier: 'Bronze' },
      { id: 'div-forecast',        name: 'Dividend Forecast Matrix', tier: 'Silver' },
    ],
  },
  {
    id: 'portfolio',
    name: 'Portfolio & Positions',
    entities: [
      { id: 'daily-val',  name: 'Daily Valuation Snap',    tier: 'Gold' },
      { id: 'alloc',      name: 'Fund Capital Allocation', tier: 'Gold' },
      { id: 'bench',      name: 'Benchmark Composition',   tier: 'Silver' },
    ],
  },
  {
    id: 'txns',
    name: 'Transactions & Trades',
    entities: [
      { id: 'exec-blotter', name: 'Executed Order Blotter',  tier: 'Gold' },
      { id: 'clear',       name: 'Clearing & Settlement Log', tier: 'Silver' },
    ],
  },
  {
    id: 'risk',
    name: 'Risk & Compliance',
    entities: [
      { id: 'var',     name: 'Value at Risk (Parametric)', tier: 'Gold' },
      { id: 'screen',  name: 'Sanctions & AML Screening',   tier: 'Bronze' },
    ],
  },
];

const TYPE_COLORS: Record<SemanticField['type'], { bg: string; color: string }> = {
  TEXT:  { bg: 'var(--mui-bg-elevated)', color: 'var(--mui-secondary-fixed, #dce1fa)' },
  NUM:   { bg: 'rgba(251, 146, 60, 0.18)', color: 'var(--mui-warning-main)' },
  DATE:  { bg: 'rgba(147, 204, 255, 0.18)', color: 'var(--mui-secondary-main)' },
  BOOL:  { bg: 'var(--mui-bg-elevated)', color: 'var(--mui-text-secondary)' },
  CURR:  { bg: 'rgba(251, 146, 60, 0.18)', color: 'var(--mui-warning-main)' },
  RATE:  { bg: 'var(--mui-bg-elevated)', color: 'var(--mui-tertiary)' },
  CAT:   { bg: 'var(--mui-bg-elevated)', color: 'var(--mui-text-secondary)' },
  COUNT: { bg: 'rgba(251, 146, 60, 0.18)', color: 'var(--mui-warning-main)' },
};

const TYPE_ICONS: Record<SemanticField['type'], React.ReactNode> = {
  TEXT:  <LabelIcon sx={{ fontSize: 14, color: 'var(--mui-text-secondary)' }} />,
  NUM:   <FunctionsIcon sx={{ fontSize: 14, color: 'var(--mui-warning-main)' }} />,
  DATE:  <CalendarTodayIcon sx={{ fontSize: 14, color: 'var(--mui-text-secondary)' }} />,
  BOOL:  <LabelIcon sx={{ fontSize: 14, color: 'var(--mui-text-secondary)' }} />,
  CURR:  <FunctionsIcon sx={{ fontSize: 14, color: 'var(--mui-warning-main)' }} />,
  RATE:  <PercentIcon sx={{ fontSize: 14, color: 'var(--mui-tertiary)' }} />,
  CAT:   <LabelIcon sx={{ fontSize: 14, color: 'var(--mui-text-secondary)' }} />,
  COUNT: <FunctionsIcon sx={{ fontSize: 14, color: 'var(--mui-warning-main)' }} />,
};

function mapBackendType(t: string): SemanticField['type'] {
  const upper = (t || '').toUpperCase();
  if (upper.includes('INT') || upper.includes('DECIMAL') || upper.includes('NUMERIC') || upper.includes('FLOAT') || upper.includes('DOUBLE') || upper.includes('REAL')) return 'NUM';
  if (upper.includes('BOOL')) return 'BOOL';
  if (upper.includes('DATE') || upper.includes('TIME') || upper.includes('TIMESTAMP')) return 'DATE';
  return 'TEXT';
}

const SemanticCatalogDetailPage: React.FC = () => {
  const { tenant, datasource } = useTenant();
  const [entity, setEntity] = useState<SemanticEntity>(DEFAULT_ENTITY);
  const [fields, setFields] = useState<SemanticField[]>(DEFAULT_FIELDS);
  const [joins, setJoins] = useState<JoinCard[]>(DEFAULT_JOINS);
  const [domains, setDomains] = useState<DomainGroup[]>([]);
  const [loadedBOs, setLoadedBOs] = useState<BackendBO[]>([]);
  const [activeEntityId, setActiveEntityId] = useState<string>('');
  const [activeTab, setActiveTab] = useState(0);
  const [viewMode, setViewMode] = useState<'list' | 'graph'>('list');
  const [searchTerm, setSearchTerm] = useState('');
  const [fieldFilter, setFieldFilter] = useState('');
  const [domainFilter, setDomainFilter] = useState('');
  const [loading, setLoading] = useState(false);
  const [inspectorLoading, setInspectorLoading] = useState(false);
  const [inspectorResult, setInspectorResult] = useState<DataInspectorResult | null>(null);
  const [snack, setSnack] = useState<{ open: boolean; msg: string; severity?: 'success' | 'info' | 'error' }>({ open: false, msg: '' });

  useEffect(() => {
    const loadBOs = async () => {
      try {
        const bos = await apiClient<BackendBO[]>('/business-objects?format=array');
        setLoadedBOs(bos);
        if (Array.isArray(bos) && bos.length > 0) {
          const grouped: Record<string, DomainGroup> = {};
          for (const bo of bos) {
            const cat = bo.category || 'Uncategorized';
            if (!grouped[cat]) {
              grouped[cat] = { id: cat.toLowerCase().replace(/\s+/g, '-'), name: cat, entities: [] };
            }
            const tier: 'Gold' | 'Silver' | 'Bronze' = bo.isCore ? 'Gold' : 'Silver';
            grouped[cat].entities.push({ id: bo.id, name: bo.displayName || bo.name, tier });
          }
          const domainList = Object.values(grouped);
          setDomains(domainList);
          if (domainList.length > 0 && domainList[0].entities.length > 0) {
            handleSelectEntity(domainList[0].entities[0].id);
          }
        }
      } catch (err) {
        devError('[SemanticCatalog] Failed to load BOs', err);
        setDomains(DEFAULT_DOMAINS);
      }
    };
    void loadBOs();
  }, []);

  const showSnack = useCallback((msg: string, severity: 'success' | 'info' | 'error' = 'info') => {
    setSnack({ open: true, msg, severity });
  }, []);

  const visibleFields = useMemo(() => {
    const q = fieldFilter.toLowerCase();
    if (!q) return fields;
    return fields.filter((f) => f.key.toLowerCase().includes(q) || f.description.toLowerCase().includes(q));
  }, [fields, fieldFilter]);

  const visibleDomains = useMemo(() => {
    const q = domainFilter.toLowerCase();
    if (!q) return domains;
    return domains.map((d) => ({
      ...d,
      entities: d.entities.filter((e) => e.name.toLowerCase().includes(q)),
    })).filter((d) => d.entities.length > 0);
  }, [domains, domainFilter]);

  const handleSelectEntity = useCallback(async (id: string) => {
    setActiveEntityId(id);
    setInspectorResult(null);
    setLoading(true);
    try {
      const [boRes, fieldsRes] = await Promise.allSettled([
        apiClient<BackendBO>(`/business-objects/${encodeURIComponent(id)}`),
        apiClient<BackendField[]>(`/business-objects/${encodeURIComponent(id)}/fields`),
      ]);

      if (boRes.status === 'fulfilled' && boRes.value) {
        const bo = boRes.value;
        const defaultBinding = bo.bindings?.find((b: { is_default: boolean }) => b.is_default) ?? bo.bindings?.[0];
        setEntity({
          id: bo.id,
          name: bo.displayName || bo.name,
          domain: bo.category || '',
          description: bo.description || '',
          tier: bo.isCore ? 'Gold' : 'Silver',
          isVerified: bo.isCore,
          isCritical: bo.instanceCount > 0,
          physicalBinding: bo.driverTableName || defaultBinding?.driving_node_name || '',
          slaFreshness: bo.instanceCount > 0 ? `Synced — ${bo.instanceCount} records` : 'No data',
          custodian: 'backend-managed',
          storageEngine: defaultBinding?.backend_type || 'StarRocks OLAP',
          totalFields: (bo.coreFields?.length ?? 0) + (bo.customFields?.length ?? 0),
          dimCount: bo.coreFields?.length ?? 0,
          measureCount: bo.customFields?.length ?? 0,
          calcCount: 0,
          downstreamQueries: 0,
          downstreamCalls24h: '',
          activeDashboards: 0,
          cacheHitRate: '',
          p95Latency: '',
        });
      }

      if (fieldsRes.status === 'fulfilled' && fieldsRes.value) {
        const backendFields: BackendField[] = fieldsRes.value;
        setFields(backendFields.map((f) => ({
          id: f.id,
          key: f.technicalName || f.name,
          type: mapBackendType(f.type),
          description: '',
          formula: f.semanticTermId ? `Linked to term ${f.semanticTermId}` : '',
          verified: !!f.semanticTermId,
          isMeasure: false,
        })));
      }

      showSnack(`Loaded ${id}`, 'success');
    } catch (err) {
      devError('[SemanticCatalog] Failed to load BO', err);
      showSnack(`Failed to load ${id} — using cached data`, 'error');
    } finally {
      setLoading(false);
    }
  }, [showSnack]);

  const handleExecuteTester = useCallback(async () => {
    if (!activeEntityId) return;
    setInspectorLoading(true);
    try {
      const result = await apiClient<DataInspectorResult>(
        `/business-objects/${encodeURIComponent(activeEntityId)}/data?limit=4`
      );
      setInspectorResult(result);
      showSnack(`Executed tester — ${result.rows.length} rows from ${result.driverTable} (${result.executionTimeMs}ms)`, 'success');
    } catch (err) {
      devError('[SemanticCatalog] Execute tester failed', err);
      showSnack(`Query failed: ${err instanceof Error ? err.message : String(err)}`, 'error');
    } finally {
      setInspectorLoading(false);
    }
  }, [activeEntityId, showSnack]);

  const handleAddMeasure = useCallback(() => {
    const newField: SemanticField = {
      id: `f-${Date.now()}`,
      key: `new_measure_${fields.length + 1}`,
      type: 'NUM',
      description: 'Newly drafted measure — describe its semantic intent.',
      formula: 'SUM(<expr>)',
      verified: false,
      isMeasure: true,
    };
    setFields((prev) => [...prev, newField]);
    showSnack('New measure drafted — pending review', 'success');
  }, [fields.length, showSnack]);

  const handleAddJoinTarget = useCallback(() => {
    const newJoin: JoinCard = {
      id: `j-${Date.now()}`,
      cardinality: 'N:1',
      target: 'catalog.new_entity',
      condition: 'sec.<id> = new.id',
      meta: 'Drafted — needs index',
      color: 'primary',
    };
    setJoins((prev) => [...prev, newJoin]);
    showSnack('New join target drafted', 'success');
  }, [showSnack]);

  return (
    <AnalyticalShell>
      <Box sx={{ display: 'flex', flexDirection: 'column', flex: 1, minHeight: 0, bgcolor: 'var(--mui-bg-default)' }}>
        {/* Command Bar / Sub-header */}
        <Paper sx={{ p: 1.5, display: 'flex', flexDirection: { xs: 'column', xl: 'row' }, alignItems: { xl: 'center' }, justifyContent: 'space-between', gap: 1.5, mx: 2, mt: 2, borderRadius: '4px' }}>
          <Box sx={{ display: 'flex', flexWrap: 'wrap', alignItems: 'center', gap: 1.5, flex: 1, minWidth: 0 }}>
            {/* Search */}
            <Box sx={{ position: 'relative', flex: 1, minWidth: 280, maxWidth: 520 }}>
              <SearchIcon sx={{ position: 'absolute', left: 12, top: '50%', transform: 'translateY(-50%)', color: 'var(--mui-text-secondary)', fontSize: 18, pointerEvents: 'none' }} />
              <InputBase
                value={searchTerm}
                onChange={(e) => setSearchTerm(e.target.value)}
                placeholder="Search semantic entities, models, measures, calculated dimensions, tags..."
                fullWidth
                sx={{
                  bgcolor: 'var(--mui-bg-subtle)',
                  color: 'var(--mui-text-primary)',
                  fontFamily: 'var(--font-body-md)',
                  fontSize: '0.8125rem',
                  pl: 4.5,
                  pr: 6,
                  py: 1,
                  borderRadius: '4px',
                  '& input': { padding: 0 },
                }}
              />
              <Box sx={{ position: 'absolute', right: 8, top: '50%', transform: 'translateY(-50%)', display: 'flex', alignItems: 'center', gap: 0.5 }}>
                <Box component="kbd" sx={{ fontFamily: 'var(--font-mono-label)', fontSize: '0.625rem', color: 'var(--mui-text-secondary)', bgcolor: 'var(--mui-bg-elevated)', px: 0.75, py: 0.25, borderRadius: '2px' }}>
                  ⌘K
                </Box>
              </Box>
            </Box>
            {/* Domain filter */}
            <Button
              startIcon={<SchemaIcon sx={{ fontSize: 18, color: 'var(--mui-primary-light)' }} />}
              endIcon={<Box component="span" sx={{ fontFamily: 'var(--font-mono-label)', fontSize: '0.625rem', color: 'var(--mui-text-secondary)', bgcolor: 'var(--mui-bg-elevated)', px: 0.75, py: 0.1, borderRadius: '2px' }}>5</Box>}
              sx={{
                bgcolor: 'var(--mui-bg-subtle)',
                color: 'var(--mui-text-primary)',
                fontFamily: 'var(--font-body-md)',
                fontSize: '0.8125rem',
                px: 1.5,
                py: 1,
                textTransform: 'none',
                '&:hover': { bgcolor: 'var(--mui-bg-elevated)' },
              }}
            >
              All Domains
            </Button>
            {/* Engine filter */}
            <Button
              startIcon={<SchemaIcon sx={{ fontSize: 18, color: 'var(--mui-tertiary)' }} />}
              endIcon={
                <Box sx={{ display: 'flex', alignItems: 'center', gap: 0.5 }}>
                  <Box sx={{ width: 6, height: 6, borderRadius: '50%', bgcolor: 'var(--mui-primary-light)', animation: 'pulse 2s infinite' }} />
                </Box>
              }
              sx={{
                bgcolor: 'var(--mui-bg-subtle)',
                color: 'var(--mui-text-primary)',
                fontFamily: 'var(--font-body-md)',
                fontSize: '0.8125rem',
                px: 1.5,
                py: 1,
                textTransform: 'none',
                '&:hover': { bgcolor: 'var(--mui-bg-elevated)' },
              }}
            >
              Engine: StarRocks OLAP
            </Button>
          </Box>
          <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
            <Button
              startIcon={<AutoAwesomeIcon sx={{ fontSize: 18, color: 'var(--mui-tertiary)' }} />}
              sx={{
                bgcolor: 'var(--mui-bg-subtle)',
                color: 'var(--mui-tertiary)',
                fontFamily: 'var(--font-mono-label)',
                fontSize: '0.6875rem',
                px: 1.5,
                py: 1,
                textTransform: 'uppercase',
                letterSpacing: '0.05em',
                '&:hover': { bgcolor: 'var(--mui-bg-elevated)' },
              }}
            >
              Embeddings & Synonyms
            </Button>
            <Button
              startIcon={<SyncAltIcon sx={{ fontSize: 18 }} />}
              sx={{
                bgcolor: 'var(--mui-bg-subtle)',
                color: 'var(--mui-text-primary)',
                fontFamily: 'var(--font-body-md)',
                fontSize: '0.8125rem',
                px: 1.5,
                py: 1,
                textTransform: 'none',
                '&:hover': { bgcolor: 'var(--mui-bg-elevated)' },
              }}
            >
              Import
            </Button>
            <ToggleButtonGroup
              size="small"
              exclusive
              value={viewMode}
              onChange={(_, v) => v && setViewMode(v)}
              sx={{ bgcolor: 'var(--mui-bg-lowest)', p: 0.5, borderRadius: '4px' }}
            >
              <ToggleButton value="list" sx={{ border: 0, color: 'var(--mui-text-secondary)', '&.Mui-selected': { bgcolor: 'var(--mui-bg-elevated)', color: 'var(--mui-primary-light)' }, fontFamily: 'var(--font-mono-label)', fontSize: '0.625rem', textTransform: 'uppercase', px: 1.5, py: 0.75 }}>
                <ViewListIcon sx={{ fontSize: 14, mr: 0.5 }} />
                List
              </ToggleButton>
              <ToggleButton value="graph" sx={{ border: 0, color: 'var(--mui-text-secondary)', '&.Mui-selected': { bgcolor: 'var(--mui-bg-elevated)', color: 'var(--mui-primary-light)' }, fontFamily: 'var(--font-mono-label)', fontSize: '0.625rem', textTransform: 'uppercase', px: 1.5, py: 0.75 }}>
                <SchemaIcon sx={{ fontSize: 14, mr: 0.5 }} />
                Graph
              </ToggleButton>
            </ToggleButtonGroup>
            <Button
              startIcon={<AddIcon sx={{ fontSize: 18 }} />}
              onClick={handleAddMeasure}
              sx={{
                bgcolor: 'var(--mui-primary-main)',
                color: 'var(--mui-primary-contrastText)',
                fontFamily: 'var(--font-headline-sm)',
                fontSize: '0.8125rem',
                px: 1.75,
                py: 1,
                textTransform: 'none',
                boxShadow: 'none',
                '&:hover': { bgcolor: 'var(--mui-primary-light)', boxShadow: 'none' },
              }}
            >
              New Model
            </Button>
          </Box>
        </Paper>

        {/* Main grid */}
        <Box sx={{ display: 'grid', gridTemplateColumns: { xs: '1fr', lg: '290px 1fr' }, gap: 1, p: 2, flex: 1, minHeight: 0, alignItems: 'start' }}>
          {/* LEFT: Domain tree */}
          <Paper sx={{ display: 'flex', flexDirection: 'column', gap: 1, p: 1.25 }}>
            <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', px: 1, py: 0.5 }}>
              <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                <Box component="span" sx={{ display: 'flex', alignItems: 'center', color: 'var(--mui-text-secondary)' }}>
                  <svg width="16" height="16" viewBox="0 0 24 24" fill="currentColor"><path d="M22 11V3h-7v3H9V3H2v8h7V8h2v10h4v3h7v-8h-7v3h-2V8h2v3z"/></svg>
                </Box>
                <Typography sx={{ fontFamily: 'var(--font-data-label)', fontSize: '0.6875rem', color: 'var(--mui-text-secondary)', textTransform: 'uppercase', letterSpacing: '0.05em' }}>
                  Business Domains
                </Typography>
              </Box>
              <Box sx={{ fontFamily: 'var(--font-mono-label)', fontSize: '0.625rem', color: 'var(--mui-text-secondary)', bgcolor: 'var(--mui-bg-elevated)', px: 0.75, py: 0.25, borderRadius: '2px' }}>
                {visibleDomains.length} Groups
              </Box>
            </Box>
            <InputBase
              value={domainFilter}
              onChange={(e) => setDomainFilter(e.target.value)}
              placeholder="Filter tree..."
              fullWidth
              sx={{
                bgcolor: 'var(--mui-bg-subtle)',
                color: 'var(--mui-text-primary)',
                fontFamily: 'var(--font-mono-label)',
                fontSize: '0.75rem',
                px: 1.25,
                py: 0.75,
                borderRadius: '3px',
                '& input': { padding: 0 },
              }}
            />
            <Box sx={{ display: 'flex', flexDirection: 'column', gap: 0.5 }}>
              {visibleDomains.map((d) => (
                <DomainGroupComponent
                  key={d.id}
                  group={d}
                  activeEntityId={activeEntityId}
                  onSelect={handleSelectEntity}
                />
              ))}
            </Box>
            <Box sx={{ mt: 'auto', pt: 1.5, bgcolor: 'var(--mui-bg-subtle)', p: 1.25, borderRadius: '4px', display: 'flex', flexDirection: 'column', gap: 0.75 }}>
              <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
                <Typography sx={{ fontFamily: 'var(--font-data-label)', fontSize: '0.625rem', color: 'var(--mui-text-secondary)', textTransform: 'uppercase', letterSpacing: '0.05em' }}>
                  Catalog Sync SLA
                </Typography>
                <Typography className="font-mono" sx={{ fontSize: '0.625rem', color: 'var(--mui-primary-light)', fontWeight: 600 }}>
                  99.99% OK
                </Typography>
              </Box>
              <Box sx={{ width: '100%', bgcolor: 'var(--mui-bg-elevated)', height: 6, borderRadius: '3px', overflow: 'hidden' }}>
                <Box sx={{ bgcolor: 'var(--mui-primary-light)', height: '100%', width: '96%', borderRadius: '3px' }} />
              </Box>
              <Typography className="font-mono" sx={{ fontSize: '0.625rem', color: 'var(--mui-text-secondary)' }}>
                Cluster Hash: #d4a9-starrocks-prod02
              </Typography>
            </Box>
          </Paper>

          {/* CENTER/RIGHT: Active Entity */}
          <Box sx={{ display: 'flex', flexDirection: 'column', gap: 1, minWidth: 0 }}>
            {/* Entity Header */}
            <Paper sx={{ p: 2, position: 'relative', overflow: 'hidden' }}>
              <Box sx={{ position: 'absolute', right: -64, top: -64, width: 256, height: 256, borderRadius: '50%', bgcolor: 'rgba(107, 216, 203, 0.05)', filter: 'blur(48px)', pointerEvents: 'none' }} />
              <Box sx={{ position: 'relative', display: 'flex', alignItems: { xs: 'flex-start', md: 'center' }, justifyContent: 'space-between', gap: 1.5, flexWrap: 'wrap' }}>
                <Box sx={{ display: 'flex', alignItems: 'flex-start', gap: 1.25 }}>
                  <Box sx={{ width: 44, height: 44, borderRadius: '6px', bgcolor: 'var(--mui-bg-elevated)', display: 'flex', alignItems: 'center', justifyContent: 'center', flexShrink: 0 }}>
                    <SchemaIcon sx={{ color: 'var(--mui-primary-light)', fontSize: 26 }} />
                  </Box>
                  <Box sx={{ display: 'flex', flexDirection: 'column', gap: 0.5 }}>
                    <Box sx={{ display: 'flex', alignItems: 'center', gap: 1, flexWrap: 'wrap' }}>
                      <Typography sx={{ fontFamily: 'var(--font-headline-md)', fontSize: '1.5rem', color: 'var(--mui-text-primary)', fontWeight: 600, letterSpacing: '-0.01em' }}>
                        {entity.name}
                      </Typography>
                      <Chip
                        size="small"
                        icon={<VerifiedIcon sx={{ fontSize: '14px !important' }} />}
                        label="Verified Gold"
                        sx={{ bgcolor: 'var(--mui-bg-elevated)', color: 'var(--mui-primary-light)', fontFamily: 'var(--font-mono-label)', fontSize: '0.6875rem', height: 22 }}
                      />
                      <Chip
                        size="small"
                        label="Tier 1 Critical"
                        sx={{ bgcolor: '#93000a', color: '#ffdad6', fontFamily: 'var(--font-data-label)', fontSize: '0.625rem', textTransform: 'uppercase', letterSpacing: '0.05em', height: 22 }}
                      />
                    </Box>
                    <Typography sx={{ fontFamily: 'var(--font-body-md)', fontSize: '0.8125rem', color: 'var(--mui-text-secondary)', maxWidth: 820 }}>
                      {entity.description}
                    </Typography>
                  </Box>
                </Box>
                <Box sx={{ display: 'flex', alignItems: 'center', gap: 0.5 }}>
                  <Tooltip title="Export Model DDL">
                    <IconButton size="small" onClick={() => showSnack('Exported model DDL', 'success')} sx={{ color: 'var(--mui-text-secondary)', bgcolor: 'var(--mui-bg-subtle)' }}>
                      <CodeIcon sx={{ fontSize: 18 }} />
                    </IconButton>
                  </Tooltip>
                  <Tooltip title="Edit Metadata">
                    <IconButton size="small" onClick={() => showSnack('Editing metadata', 'info')} sx={{ color: 'var(--mui-text-secondary)', bgcolor: 'var(--mui-bg-subtle)' }}>
                      <EditIcon sx={{ fontSize: 18 }} />
                    </IconButton>
                  </Tooltip>
                  <Button
                    startIcon={<HistoryIcon sx={{ fontSize: 16, color: 'var(--mui-secondary-main)' }} />}
                    sx={{ bgcolor: 'var(--mui-bg-subtle)', color: 'var(--mui-text-primary)', fontFamily: 'var(--font-mono-label)', fontSize: '0.75rem', px: 1.25, py: 0.75, textTransform: 'none', '&:hover': { bgcolor: 'var(--mui-bg-elevated)' } }}
                  >
                    v2.4.1
                  </Button>
                </Box>
              </Box>

              {/* Physical Binding & Operational Context Bar */}
              <Box sx={{ display: 'grid', gridTemplateColumns: { xs: '1fr 1fr', md: 'repeat(4, 1fr)' }, gap: 1, pt: 1.5 }}>
                <ContextCell label="Physical Binding" value={entity.physicalBinding} valueColor="var(--mui-primary-light)" />
                <ContextCell label="SLA Freshness" value={<Box sx={{ display: 'flex', alignItems: 'center', gap: 0.75 }}><Box sx={{ width: 6, height: 6, borderRadius: '50%', bgcolor: 'var(--mui-primary-light)', animation: 'pulse 2s infinite' }} /><Box component="span">{entity.slaFreshness}</Box></Box>} />
                <ContextCell label="Data Custodian" value={`@${entity.custodian.replace('@', '')}`} valueColor="var(--mui-secondary-main)" />
                <ContextCell label="Storage Engine" value={entity.storageEngine} valueColor="var(--mui-tertiary)" />
              </Box>

              {/* Metrics Ribbon */}
              <Box sx={{ display: 'grid', gridTemplateColumns: { xs: 'repeat(2, 1fr)', sm: 'repeat(4, 1fr)' }, gap: 1, pt: 1.5 }}>
                <MetricCard label="Total Fields" value={entity.totalFields} subtitle={`${entity.dimCount} Dim · ${entity.measureCount} Meas · ${entity.calcCount} Calc`} icon={<FunctionsIcon sx={{ fontSize: 20, color: 'var(--mui-primary-light)' }} />} />
                <MetricCard label="Downstream Queries" value={entity.downstreamQueries} subtitle={entity.downstreamCalls24h} icon={<TerminalIcon sx={{ fontSize: 20, color: 'var(--mui-tertiary)' }} />} valueColor="var(--mui-tertiary)" />
                <MetricCard label="Active Dashboards" value={entity.activeDashboards} subtitle="Prod Executive Deck" icon={<SchemaIcon sx={{ fontSize: 20, color: '#afb4cc' }} />} valueColor="#afb4cc" />
                <MetricCard label="Cache Hit Rate" value={entity.cacheHitRate} subtitle={entity.p95Latency} icon={<svg width="20" height="20" viewBox="0 0 24 24" fill="#6bd8cb"><path d="M11 21h-1l1-7H7.5c-.58 0-.57-.32-.38-.66.19-.34.05-.08.07-.12C8.48 10.94 10.42 7.54 13 3h1l-1 7h3.5c.49 0 .56.33.47.51l-.07.15C12.96 17.55 11 21 11 21z"/></svg>} valueColor="var(--mui-primary-light)" />
              </Box>
            </Paper>

            {/* Tabs */}
            <Paper sx={{ px: 1.5, py: 0.5, display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
              <Tabs value={activeTab} onChange={(_, v) => setActiveTab(v)} sx={{ minHeight: 40, '& .MuiTab-root': { minHeight: 40, py: 0.5, px: 1.5 } }}>
                <Tab
                  label={
                    <Box sx={{ display: 'flex', alignItems: 'center', gap: 0.75 }}>
                      <MenuBookIcon sx={{ fontSize: 16 }} />
                      <span>Fields &amp; Metrics Dictionary</span>
                      <Box sx={{ fontFamily: 'var(--font-mono-label)', fontSize: '0.6875rem', bgcolor: 'rgba(107, 216, 203, 0.2)', color: 'var(--mui-primary-light)', px: 0.75, borderRadius: '2px' }}>{entity.totalFields}</Box>
                    </Box>
                  }
                />
                <Tab
                  label={
                    <Box sx={{ display: 'flex', alignItems: 'center', gap: 0.75 }}>
                      <HubIcon sx={{ fontSize: 16 }} />
                      <span>Joins &amp; Relationship Graph</span>
                      <Box sx={{ fontFamily: 'var(--font-mono-label)', fontSize: '0.6875rem', bgcolor: 'var(--mui-bg-elevated)', color: 'var(--mui-text-secondary)', px: 0.75, borderRadius: '2px' }}>{joins.length}</Box>
                    </Box>
                  }
                />
                <Tab
                  label={
                    <Box sx={{ display: 'flex', alignItems: 'center', gap: 0.75 }}>
                      <AltRouteIcon sx={{ fontSize: 16 }} />
                      <span>Lineage &amp; Downstream Consumers</span>
                      <Box sx={{ fontFamily: 'var(--font-mono-label)', fontSize: '0.6875rem', bgcolor: 'var(--mui-bg-elevated)', color: 'var(--mui-text-secondary)', px: 0.75, borderRadius: '2px' }}>16</Box>
                    </Box>
                  }
                />
              </Tabs>
              <Box sx={{ display: { xs: 'none', sm: 'flex' }, alignItems: 'center', gap: 0.5, bgcolor: 'var(--mui-bg-subtle)', px: 1, py: 0.5, borderRadius: '3px' }}>
                <SearchIcon sx={{ fontSize: 14, color: 'var(--mui-text-secondary)' }} />
                <InputBase
                  value={fieldFilter}
                  onChange={(e) => setFieldFilter(e.target.value)}
                  placeholder="Filter fields..."
                  sx={{ fontFamily: 'var(--font-mono-label)', fontSize: '0.75rem', color: 'var(--mui-text-primary)', width: 140, '& input': { padding: 0 } }}
                />
              </Box>
            </Paper>

            {/* Tab content */}
            {activeTab === 0 && (
              <Paper sx={{ p: 2, display: 'flex', flexDirection: 'column', gap: 1.5 }}>
                <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
                  <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                    <Typography sx={{ fontFamily: 'var(--font-headline-sm)', fontSize: '1.125rem', color: 'var(--mui-text-primary)' }}>
                      Semantic Definitions
                    </Typography>
                    <Typography className="font-mono" sx={{ fontSize: '0.75rem', color: 'var(--mui-text-secondary)' }}>
                      Showing {visibleFields.length} priority fields of {entity.totalFields}
                    </Typography>
                  </Box>
                  <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                    <Button
                      sx={{ fontFamily: 'var(--font-mono-label)', fontSize: '0.6875rem', color: 'var(--mui-text-secondary)', bgcolor: 'var(--mui-bg-subtle)', px: 1, py: 0.5, textTransform: 'none', '&:hover': { bgcolor: 'var(--mui-bg-elevated)' } }}
                      startIcon={<Box component="span" sx={{ fontSize: '0.625rem' }}>Sort: Type</Box>}
                    >
                      Type
                    </Button>
                    <Button
                      onClick={handleAddMeasure}
                      startIcon={<AddIcon sx={{ fontSize: 14 }} />}
                      sx={{ fontFamily: 'var(--font-mono-label)', fontSize: '0.6875rem', color: 'var(--mui-primary-light)', bgcolor: 'var(--mui-bg-subtle)', border: '1px solid rgba(107, 216, 203, 0.3)', px: 1, py: 0.5, textTransform: 'none', '&:hover': { bgcolor: 'var(--mui-bg-elevated)' } }}
                    >
                      Add Measure
                    </Button>
                  </Box>
                </Box>
                <TableContainer sx={{ borderRadius: '4px' }}>
                  <Table size="small">
                    <TableHead>
                      <TableRow sx={{ bgcolor: 'var(--mui-bg-elevated)' }}>
                        <TableCell sx={{ color: 'var(--mui-text-secondary)', fontFamily: 'var(--font-data-label)', fontSize: '0.625rem', textTransform: 'uppercase', letterSpacing: '0.05em', borderBottom: '1px solid var(--mui-border)' }}>Field / Key</TableCell>
                        <TableCell sx={{ color: 'var(--mui-text-secondary)', fontFamily: 'var(--font-data-label)', fontSize: '0.625rem', textTransform: 'uppercase', letterSpacing: '0.05em', borderBottom: '1px solid var(--mui-border)', width: 80 }}>Type</TableCell>
                        <TableCell sx={{ color: 'var(--mui-text-secondary)', fontFamily: 'var(--font-data-label)', fontSize: '0.625rem', textTransform: 'uppercase', letterSpacing: '0.05em', borderBottom: '1px solid var(--mui-border)' }}>Logical Description</TableCell>
                        <TableCell sx={{ color: 'var(--mui-text-secondary)', fontFamily: 'var(--font-data-label)', fontSize: '0.625rem', textTransform: 'uppercase', letterSpacing: '0.05em', borderBottom: '1px solid var(--mui-border)', width: 280 }}>SQL Formula / Expression</TableCell>
                        <TableCell sx={{ color: 'var(--mui-text-secondary)', fontFamily: 'var(--font-data-label)', fontSize: '0.625rem', textTransform: 'uppercase', letterSpacing: '0.05em', borderBottom: '1px solid var(--mui-border)', width: 90, textAlign: 'center' }}>Verified</TableCell>
                        <TableCell sx={{ color: 'var(--mui-text-secondary)', fontFamily: 'var(--font-data-label)', fontSize: '0.625rem', textTransform: 'uppercase', letterSpacing: '0.05em', borderBottom: '1px solid var(--mui-border)', width: 70, textAlign: 'right' }}>Actions</TableCell>
                      </TableRow>
                    </TableHead>
                    <TableBody>
                      {visibleFields.map((f, i) => (
                        <TableRow key={f.id} hover sx={{ '&:nth-of-type(odd)': { bgcolor: 'rgba(16, 32, 52, 0.5)' } }}>
                          <TableCell sx={{ borderBottom: '1px solid var(--mui-border)', fontFamily: 'var(--font-mono-label)', fontSize: '0.75rem', color: 'var(--mui-text-primary)', fontWeight: 600 }}>
                            <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                              <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'center', width: 22, height: 22, borderRadius: '4px', bgcolor: 'rgba(56, 189, 248, 0.1)' }}>
                                {f.type === 'TEXT' ? <KeyIcon sx={{ fontSize: 14, color: 'var(--mui-primary-light)' }} /> :
                                 f.type === 'CURR' || f.type === 'COUNT' ? <FunctionsIcon sx={{ fontSize: 14, color: 'var(--mui-tertiary)' }} /> :
                                 f.type === 'RATE' ? <PercentIcon sx={{ fontSize: 14, color: 'var(--mui-tertiary)' }} /> :
                                 f.type === 'DATE' ? <CalendarTodayIcon sx={{ fontSize: 14, color: 'var(--mui-text-secondary)' }} /> :
                                 f.type === 'CAT' ? <LabelIcon sx={{ fontSize: 14, color: 'var(--mui-text-secondary)' }} /> :
                                 <ShowChartIcon sx={{ fontSize: 14, color: 'var(--mui-tertiary)' }} />}
                              </Box>
                              {f.key}
                            </Box>
                          </TableCell>
                          <TableCell sx={{ borderBottom: '1px solid var(--mui-border)' }}>
                            <Box sx={{ fontFamily: 'var(--font-mono-label)', fontSize: '0.6875rem', bgcolor: TYPE_COLORS[f.type].bg, color: TYPE_COLORS[f.type].color, px: 0.75, py: 0.25, borderRadius: '2px', display: 'inline-block', fontWeight: 700 }}>
                              {f.type}
                            </Box>
                          </TableCell>
                          <TableCell sx={{ borderBottom: '1px solid var(--mui-border)', fontFamily: 'var(--font-body-md)', fontSize: '0.8125rem', color: 'var(--mui-text-secondary)', maxWidth: 320 }}>
                            <Box sx={{ overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>{f.description}</Box>
                          </TableCell>
                          <TableCell sx={{ borderBottom: '1px solid var(--mui-border)' }}>
                            <Box sx={{ fontFamily: 'var(--font-mono-label)', fontSize: '0.75rem', bgcolor: 'var(--mui-bg-elevated)', px: 1, py: 0.5, borderRadius: '2px', color: 'var(--mui-text-secondary)', display: 'inline-block', overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap', maxWidth: '100%' }}>
                              {f.formula}
                            </Box>
                          </TableCell>
                          <TableCell sx={{ borderBottom: '1px solid var(--mui-border)', textAlign: 'center' }}>
                            {f.verified ? <VerifiedIcon sx={{ fontSize: 18, color: 'var(--mui-primary-light)' }} /> : <Box sx={{ display: 'inline-block', width: 18, height: 18, borderRadius: '50%', border: '1.5px solid var(--mui-text-secondary)' }} />}
                          </TableCell>
                          <TableCell sx={{ borderBottom: '1px solid var(--mui-border)', textAlign: 'right' }}>
                            <IconButton size="small" sx={{ color: 'var(--mui-text-secondary)' }} onClick={() => showSnack(`Actions for ${f.key}`, 'info')}>
                              <MoreHorizIcon sx={{ fontSize: 16 }} />
                            </IconButton>
                          </TableCell>
                        </TableRow>
                      ))}
                    </TableBody>
                  </Table>
                </TableContainer>
              </Paper>
            )}

            {activeTab === 1 && (
              <Paper sx={{ p: 2, display: 'flex', flexDirection: 'column', gap: 1.5 }}>
                <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
                  <Box sx={{ display: 'flex', flexDirection: 'column' }}>
                    <Typography sx={{ fontFamily: 'var(--font-headline-sm)', fontSize: '1.125rem', color: 'var(--mui-text-primary)' }}>
                      Relationship Graph Topology
                    </Typography>
                    <Typography sx={{ fontFamily: 'var(--font-body-md)', fontSize: '0.8125rem', color: 'var(--mui-text-secondary)' }}>
                      Configured foreign constraints &amp; 1-to-N join cardinalities for optimal engine planning.
                    </Typography>
                  </Box>
                  <Button
                    onClick={handleAddJoinTarget}
                    startIcon={<AddLinkIcon sx={{ fontSize: 16 }} />}
                    sx={{ bgcolor: 'var(--mui-bg-subtle)', color: 'var(--mui-primary-light)', border: '1px solid rgba(107, 216, 203, 0.3)', fontFamily: 'var(--font-mono-label)', fontSize: '0.6875rem', px: 1.25, py: 0.75, textTransform: 'none', '&:hover': { bgcolor: 'var(--mui-bg-elevated)' } }}
                  >
                    Add Join Target
                  </Button>
                </Box>
                <Box sx={{ display: 'grid', gridTemplateColumns: { xs: '1fr', md: 'repeat(3, 1fr)' }, gap: 1.5 }}>
                  {joins.map((j) => (
                    <Paper key={j.id} sx={{ p: 1.5, display: 'flex', flexDirection: 'column', gap: 1, bgcolor: 'var(--mui-bg-subtle)', border: '1px solid var(--mui-border)' }}>
                      <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
                        <Box sx={{ fontFamily: 'var(--font-mono-label)', fontSize: '0.625rem', bgcolor: 'var(--mui-bg-elevated)', px: 0.75, py: 0.25, borderRadius: '2px', color: j.color === 'primary' ? 'var(--mui-primary-light)' : j.color === 'tertiary' ? 'var(--mui-tertiary)' : 'var(--mui-secondary-main)', fontWeight: 700 }}>
                          {j.cardinality === 'N:1' ? 'MANY-TO-ONE' : j.cardinality === '1:N' ? 'ONE-TO-MANY' : 'ONE-TO-ONE'} ({j.cardinality})
                        </Box>
                        <HubIcon sx={{ fontSize: 18, color: j.color === 'primary' ? 'var(--mui-primary-light)' : j.color === 'tertiary' ? 'var(--mui-tertiary)' : 'var(--mui-secondary-main)' }} />
                      </Box>
                      <Box>
                        <Typography sx={{ fontFamily: 'var(--font-data-label)', fontSize: '0.625rem', color: 'var(--mui-text-secondary)', textTransform: 'uppercase', letterSpacing: '0.05em' }}>
                          Target Entity
                        </Typography>
                        <Typography sx={{ fontFamily: 'var(--font-headline-sm)', fontSize: '0.95rem', color: 'var(--mui-text-primary)', fontWeight: 600 }}>
                          {j.target}
                        </Typography>
                      </Box>
                      <Box sx={{ bgcolor: 'var(--mui-bg-elevated)', p: 0.75, borderRadius: '3px', display: 'flex', flexDirection: 'column', gap: 0.25 }}>
                        <Typography sx={{ fontFamily: 'var(--font-data-label)', fontSize: '0.625rem', color: 'var(--mui-text-secondary)', textTransform: 'uppercase', letterSpacing: '0.05em' }}>
                          Foreign Condition
                        </Typography>
                        <Box component="code" sx={{ fontFamily: 'var(--font-mono-label)', fontSize: '0.75rem', color: 'var(--mui-text-primary)' }}>
                          {j.condition}
                        </Box>
                      </Box>
                      <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', fontFamily: 'var(--font-mono-label)', fontSize: '0.6875rem', color: 'var(--mui-text-secondary)' }}>
                        <span>{j.meta}</span>
                        <span style={{ color: 'var(--mui-primary-light)' }}>OK</span>
                      </Box>
                    </Paper>
                  ))}
                </Box>
              </Paper>
            )}

            {activeTab === 2 && (
              <Paper sx={{ p: 2, display: 'flex', flexDirection: 'column', gap: 1.5 }}>
                <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
                  <Box sx={{ display: 'flex', flexDirection: 'column' }}>
                    <Typography sx={{ fontFamily: 'var(--font-headline-sm)', fontSize: '1.125rem', color: 'var(--mui-text-primary)' }}>
                      Upstream &amp; Downstream Lineage
                    </Typography>
                    <Typography sx={{ fontFamily: 'var(--font-body-md)', fontSize: '0.8125rem', color: 'var(--mui-text-secondary)' }}>
                      Trace source CDC feeds down to financial models and regulatory reporting engines.
                    </Typography>
                  </Box>
                  <Box sx={{ fontFamily: 'var(--font-mono-label)', fontSize: '0.75rem', color: 'var(--mui-tertiary)', bgcolor: 'var(--mui-bg-subtle)', px: 1.25, py: 0.5, borderRadius: '3px' }}>
                    Graph Depth: 4 Layers
                  </Box>
                </Box>
                <Box sx={{ bgcolor: 'var(--mui-bg-subtle)', borderRadius: '4px', p: 2, display: 'flex', alignItems: 'center', justifyContent: 'center', overflowX: 'auto' }}>
                  <svg fill="none" style={{ width: '100%', maxWidth: 680, height: 160 }} viewBox="0 0 680 160" xmlns="http://www.w3.org/2000/svg">
                    <rect fill="#102034" height="48" rx="6" width="160" x="20" y="20" />
                    <text fill="#d3e4fe" fontFamily="Inter" fontSize="12" fontWeight="600" x="35" y="42">Bloomberg B-PIPE</text>
                    <text fill="#879391" fontFamily="JetBrains Mono" fontSize="10" x="35" y="56">Kafka Topic · Raw Ingest</text>
                    <rect fill="#102034" height="48" rx="6" width="160" x="20" y="90" />
                    <text fill="#d3e4fe" fontFamily="Inter" fontSize="12" fontWeight="600" x="35" y="112">Refinitiv ESG Fixes</text>
                    <text fill="#879391" fontFamily="JetBrains Mono" fontSize="10" x="35" y="126">S3 Parquet · 04:00 UTC</text>
                    <rect fill="#1b2b3f" height="56" rx="6" stroke="#6bd8cb" strokeWidth="1.5" width="170" x="260" y="52" />
                    <text fill="#6bd8cb" fontFamily="Inter" fontSize="13" fontWeight="700" x="275" y="76">Security Master</text>
                    <text fill="#d3e4fe" fontFamily="JetBrains Mono" fontSize="10" x="275" y="94">StarRocks OLAP · Verified</text>
                    <rect fill="#102034" height="48" rx="6" width="160" x="500" y="20" />
                    <text fill="#d3e4fe" fontFamily="Inter" fontSize="12" fontWeight="600" x="515" y="42">Risk VaR Simulation</text>
                    <text fill="#879391" fontFamily="JetBrains Mono" fontSize="10" x="515" y="56">C++ Pricing Engine</text>
                    <rect fill="#102034" height="48" rx="6" width="160" x="500" y="90" />
                    <text fill="#d3e4fe" fontFamily="Inter" fontSize="12" fontWeight="600" x="515" y="112">Exec P&amp;L Portal</text>
                    <text fill="#879391" fontFamily="JetBrains Mono" fontSize="10" x="515" y="126">4 Active Superset Dashboards</text>
                    <path d="M 180 44 C 220 44, 220 70, 260 70" stroke="#6bd8cb" strokeDasharray="3 3" strokeWidth="2" />
                    <path d="M 180 114 C 220 114, 220 90, 260 90" stroke="#6bd8cb" strokeWidth="2" />
                    <path d="M 430 70 C 465 70, 465 44, 500 44" stroke="#93ccff" strokeWidth="2" />
                    <path d="M 430 90 C 465 90, 465 114, 500 114" stroke="#93ccff" strokeWidth="2" />
                  </svg>
                </Box>
              </Paper>
            )}

            {/* Data Inspector */}
            <Paper sx={{ p: 2, display: 'flex', flexDirection: 'column', gap: 1 }}>
              <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
                <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                  <TerminalIcon sx={{ fontSize: 20, color: 'var(--mui-primary-light)' }} />
                  <Typography sx={{ fontFamily: 'var(--font-headline-sm)', fontSize: '1.125rem', color: 'var(--mui-text-primary)' }}>
                    Data Inspector &amp; Quick Tester
                  </Typography>
                  <Box sx={{ fontFamily: 'var(--font-mono-label)', fontSize: '0.625rem', color: 'var(--mui-text-secondary)', bgcolor: 'var(--mui-bg-subtle)', px: 0.75, py: 0.25, borderRadius: '2px' }}>
                    LIMIT 4 ROWS
                  </Box>
                </Box>
                <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                  <Button
                    onClick={handleExecuteTester}
                    startIcon={<PlayArrowIcon sx={{ fontSize: 14 }} />}
                    sx={{ bgcolor: 'var(--mui-primary-main)', color: 'var(--mui-primary-contrastText)', fontFamily: 'var(--font-mono-label)', fontSize: '0.6875rem', px: 1.5, py: 0.5, textTransform: 'none', boxShadow: 'none', '&:hover': { bgcolor: 'var(--mui-primary-light)', boxShadow: 'none' } }}
                  >
                    Execute Tester
                  </Button>
                  <Tooltip title="Maximize">
                    <IconButton size="small" sx={{ color: 'var(--mui-text-secondary)', bgcolor: 'var(--mui-bg-subtle)' }}>
                      <OpenInFullIcon sx={{ fontSize: 16 }} />
                    </IconButton>
                  </Tooltip>
                </Box>
              </Box>
              <Box sx={{ overflowX: 'auto', bgcolor: 'var(--mui-bg-lowest)', borderRadius: '4px', p: 1 }}>
                {inspectorLoading ? (
                  <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'center', py: 4, gap: 1.5 }}>
                    <CircularProgress size={18} sx={{ color: 'var(--mui-primary-light)' }} />
                    <Typography sx={{ fontFamily: 'var(--font-mono-label)', fontSize: '0.75rem', color: 'var(--mui-text-secondary)' }}>
                      Executing query...
                    </Typography>
                  </Box>
                ) : inspectorResult ? (
                  inspectorResult.rows.length === 0 ? (
                    <Box sx={{ p: 3, textAlign: 'center' }}>
                      <Typography sx={{ fontFamily: 'var(--font-mono-label)', fontSize: '0.75rem', color: 'var(--mui-text-secondary)' }}>
                        No rows returned — the bound table may be empty
                      </Typography>
                    </Box>
                  ) : (
                    <Table size="small">
                      <TableHead>
                        <TableRow>
                          {inspectorResult.columns.map((c) => (
                            <TableCell key={c} sx={{ color: 'var(--mui-text-secondary)', fontFamily: 'var(--font-mono-label)', fontSize: '0.625rem', textTransform: 'uppercase', letterSpacing: '0.05em', borderBottom: 'none', py: 1 }}>
                              {c}
                            </TableCell>
                          ))}
                        </TableRow>
                      </TableHead>
                      <TableBody>
                        {inspectorResult.rows.map((row, i) => (
                          <TableRow key={i} hover>
                            {inspectorResult.columns.map((col) => (
                              <TableCell
                                key={col}
                                sx={{
                                  fontFamily: 'var(--font-mono-label)',
                                  fontSize: '0.75rem',
                                  py: 0.75,
                                  borderBottom: '1px solid rgba(255,255,255,0.04)',
                                  color: 'var(--mui-text-primary)',
                                }}
                              >
                                {String(row[col] ?? '')}
                              </TableCell>
                            ))}
                          </TableRow>
                        ))}
                      </TableBody>
                    </Table>
                  )
                ) : (
                  <Box sx={{ p: 3, textAlign: 'center' }}>
                    <Typography sx={{ fontFamily: 'var(--font-mono-label)', fontSize: '0.75rem', color: 'var(--mui-text-secondary)' }}>
                      Click "Execute Tester" to run a live query against the bound table
                    </Typography>
                  </Box>
                )}
              </Box>
            </Paper>
          </Box>
        </Box>

        <Snackbar open={snack.open} autoHideDuration={3500} onClose={() => setSnack({ ...snack, open: false })} anchorOrigin={{ vertical: 'bottom', horizontal: 'center' }}>
          <Alert severity={snack.severity ?? 'info'} sx={{ width: '100%' }}>
            {snack.msg}
          </Alert>
        </Snackbar>
      </Box>
    </AnalyticalShell>
  );
};

const ContextCell: React.FC<{ label: string; value: React.ReactNode; valueColor?: string }> = ({ label, value, valueColor }) => (
  <Box sx={{ bgcolor: 'var(--mui-bg-subtle)', px: 1.25, py: 1, borderRadius: '4px', display: 'flex', flexDirection: 'column' }}>
    <Typography sx={{ fontFamily: 'var(--font-data-label)', fontSize: '0.625rem', color: 'var(--mui-text-secondary)', textTransform: 'uppercase', letterSpacing: '0.05em' }}>
      {label}
    </Typography>
    <Box sx={{ fontFamily: 'var(--font-mono-label)', fontSize: '0.8125rem', color: valueColor ?? 'var(--mui-text-primary)', fontWeight: 600, mt: 0.5, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
      {value}
    </Box>
  </Box>
);

const MetricCard: React.FC<{ label: string; value: string | number; subtitle: string; icon: React.ReactNode; valueColor?: string }> = ({ label, value, subtitle, icon, valueColor }) => (
  <Box sx={{ bgcolor: 'rgba(27, 43, 63, 0.6)', p: 1.25, borderRadius: '4px', display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
    <Box>
      <Typography sx={{ fontFamily: 'var(--font-data-label)', fontSize: '0.625rem', color: 'var(--mui-text-secondary)', textTransform: 'uppercase', letterSpacing: '0.05em' }}>
        {label}
      </Typography>
      <Typography sx={{ fontFamily: 'var(--font-display-lg)', fontSize: '1.75rem', color: valueColor ?? 'var(--mui-text-primary)', fontWeight: 700, lineHeight: 1.1, mt: 0.5 }}>
        {value}
      </Typography>
      <Typography className="font-mono" sx={{ fontSize: '0.6875rem', color: 'var(--mui-text-secondary)', mt: 0.25 }}>
        {subtitle}
      </Typography>
    </Box>
    <Box sx={{ width: 36, height: 36, borderRadius: '6px', bgcolor: 'var(--mui-bg-elevated)', display: 'flex', alignItems: 'center', justifyContent: 'center', color: valueColor ?? 'var(--mui-primary-light)' }}>
      {icon}
    </Box>
  </Box>
);

const DomainGroupComponent: React.FC<{ group: DomainGroup; activeEntityId: string; onSelect: (id: string) => void }> = ({ group, activeEntityId, onSelect }) => {
  const [open, setOpen] = useState(true);
  return (
    <Box>
      <Box
        onClick={() => setOpen((o) => !o)}
        sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', px: 1, py: 0.75, borderRadius: '3px', cursor: 'pointer', userSelect: 'none', '&:hover': { bgcolor: 'var(--mui-bg-subtle)' } }}
      >
        <Box sx={{ display: 'flex', alignItems: 'center', gap: 0.75 }}>
          <Box sx={{ display: 'flex', alignItems: 'center', color: 'var(--mui-text-secondary)', transition: 'transform 0.2s', transform: open ? 'rotate(90deg)' : 'rotate(0deg)' }}>
            <ChevronRightIcon sx={{ fontSize: 18 }} />
          </Box>
          <Typography sx={{ fontFamily: 'var(--font-body-md)', fontSize: '0.8125rem', fontWeight: 600, color: 'var(--mui-text-primary)' }}>
            {group.name}
          </Typography>
        </Box>
        <Box sx={{ fontFamily: 'var(--font-mono-label)', fontSize: '0.625rem', color: 'var(--mui-text-secondary)', bgcolor: 'var(--mui-bg-subtle)', px: 0.75, py: 0.1, borderRadius: '2px' }}>
          {group.entities.length}
        </Box>
      </Box>
      {open && (
        <Box sx={{ pl: 3.5, pr: 0.5, pt: 0.5, display: 'flex', flexDirection: 'column', gap: 0.25 }}>
          {group.entities.map((e) => {
            const isActive = e.id === activeEntityId;
            return (
              <Box
                key={e.id}
                role="button"
                tabIndex={0}
                onClick={() => onSelect(e.id)}
                onKeyDown={(ev) => { if (ev.key === 'Enter' || ev.key === ' ') onSelect(e.id); }}
                sx={{
                  display: 'flex',
                  alignItems: 'center',
                  justifyContent: 'space-between',
                  px: 1,
                  py: 0.5,
                  borderRadius: '3px',
                  cursor: 'pointer',
                  bgcolor: isActive ? 'var(--mui-bg-elevated)' : 'transparent',
                  color: isActive ? 'var(--mui-primary-light)' : 'var(--mui-text-primary)',
                  fontFamily: 'var(--font-body-md)',
                  fontSize: '0.8125rem',
                  fontWeight: isActive ? 600 : 400,
                  '&:hover': { bgcolor: 'var(--mui-bg-subtle)' },
                }}
              >
                <Box sx={{ display: 'flex', alignItems: 'center', gap: 1, minWidth: 0 }}>
                  <Box sx={{ width: 6, height: 6, borderRadius: '50%', bgcolor: 'var(--mui-primary-light)' }} />
                  <Box sx={{ overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>{e.name}</Box>
                </Box>
                <Box sx={{ fontFamily: 'var(--font-mono-label)', fontSize: '0.5625rem', color: e.tier === 'Gold' ? 'var(--mui-primary-light)' : 'var(--mui-text-secondary)', bgcolor: e.tier === 'Gold' ? 'var(--mui-bg-lowest)' : 'transparent', px: 0.5, py: 0.1, borderRadius: '2px', textTransform: 'uppercase' }}>
                  {e.tier}
                </Box>
              </Box>
            );
          })}
        </Box>
      )}
    </Box>
  );
};

export default SemanticCatalogDetailPage;
