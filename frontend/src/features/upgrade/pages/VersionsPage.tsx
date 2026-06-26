import { Suspense, lazy } from 'react';
import { devError } from '../../../utils/devLogger';
import { useEffect, useState } from 'react';
import { Box, Breadcrumbs, Button, Card, CardContent, Chip, Grid, Link as MUILink, Stack, Typography, List, ListItem, ListItemText, FormControl, InputLabel, Select, MenuItem, Tooltip, Alert, LinearProgress, Accordion, AccordionSummary, AccordionDetails, Dialog, DialogContent, DialogActions, TextField, Paper, Table, TableBody, TableCell, TableContainer, TableHead, TableRow, IconButton as _IconButton, Badge as _Badge } from '@mui/material';
import ModalHeader from '../../../components/ModalHeader';
import { ExpandMore as ExpandMoreIcon, Refresh as RefreshIcon, Compare as CompareIcon, Build as BuildIcon, QueryStats as QueryStatsIcon, Assessment as AssessmentIcon, History as HistoryIcon, HealthAndSafety as HealthIcon, Schedule as ScheduleIcon, Approval as ApprovalIcon, CheckCircle as CheckCircleIcon, Error as ErrorIcon, Warning as WarningIcon, Info as InfoIcon, PlayArrow as PlayArrowIcon, Archive as ArchiveIcon, Security as _SecurityIcon, Timeline as _TimelineIcon, WarningAmber as WarningAmberIcon } from '@mui/icons-material';
import type { GridColDef, GridRenderCellParams } from '@mui/x-data-grid';
import { Link, useSearchParams } from 'react-router-dom';
import BlockableLink from '../../../components/RouteBlocker/BlockableLink';
import _NotificationsActiveIcon from '@mui/icons-material/NotificationsActive';
import _HealthAndSafetyIcon from '@mui/icons-material/HealthAndSafety';
const LazyDataGrid = lazy(() => import('@mui/x-data-grid').then(m => ({ default: (m as any).DataGrid })));
import { fetchVersions, activateVersion, rollbackVersion, setPreview, startCanary, listNotifications, prepareUpgrade, generateCore, mergeCustom, validateVersion, runShadow, getValidationReport as _getValidationReport, archiveVersion, getSchemaChanges as _getSchemaChanges, getDeprecationMap as _getDeprecationMap, getPreAggRebuild as _getPreAggRebuild, scheduleBatchJob, listBatchJobs, listArtifacts, getSLOMetrics, getSchemaVersion, getUpgradeStatus as _getUpgradeStatus, getUpgradeOverview } from '../api';
import DiffViewer from '../components/DiffViewer';
import ExtensionFixer from '../components/ExtensionFixer';
import GoldenQueryRunner from '../components/GoldenQueryRunner';
import { SchemaVersionDisplay } from '../components/SchemaVersionDisplay';
import { UpgradeOverview } from '../components/UpgradeOverview';
import { SchemaHistory as _SchemaHistory } from '../components/SchemaHistory';
import { VersionsTable } from '../components/VersionsTable';
import * as _UpgradeTypes from '../../types/upgrade-generated';
import type { UpgradeStatusResponse } from '../../types/upgrade';

// Define types inline to avoid import issues
type UpgradeArtifactsData = {
  schema_version: string;
  changelog?: Array<{
    version: string;
    date: string;
    description: string;
  }>;
  report: any;
  aliases: any;
};

type UpgradeStatusResponse = {
  core_version: string;
  status: "pending" | "ready" | "canary" | "active" | "rolled_back";
  warnings: string[];
  blockers: string[];
};

export default function VersionsPage() {
  const [rows, setRows] = useState<any[]>([]);
  const [slo, setSlo] = useState<any>(null);
  const [canary, setCanary] = useState<any>(null);
  const [notifications, setNotifications] = useState<Array<{ id: string; type: string; message: string; severity: string; created_at: string }>>([]);
  const [severityFilter, setSeverityFilter] = useState<'all'|'success'|'info'|'warning'|'error'>('all');
  const [typeFilter, setTypeFilter] = useState<string>('all');
  const [autoRefresh, setAutoRefresh] = useState<boolean>(true);
  const [refreshIntervalSec, setRefreshIntervalSec] = useState<number>(30);
  const [searchParams, setSearchParams] = useSearchParams();

  // New state for lifecycle management
  const [lifecycleStep, setLifecycleStep] = useState<string>('idle');
  const [currentVersion, setCurrentVersion] = useState<string>('');
  const [validationReport, setValidationReport] = useState<any>(null);
  const [_schemaChanges, setSchemaChanges] = useState<any[]>([]);
  const [_deprecationMap, setDeprecationMap] = useState<any>(null);
  const [_preAggRebuild, _setPreAggRebuild] = useState<any>(null);

  // Batch jobs state
  const [batchJobs, setBatchJobs] = useState<any[]>([]);
  const [showBatchJobs, setShowBatchJobs] = useState<boolean>(false);

  // New state for VersionsTable
  const [showInteractiveTable, setShowInteractiveTable] = useState<boolean>(false);

  // Artifacts state
  const [artifacts, setArtifacts] = useState<any[]>([]);
  const [showArtifacts, setShowArtifacts] = useState<boolean>(false);

  // SLO monitoring state
  const [sloMetrics, setSloMetrics] = useState<any[]>([]);
  const [sloAlerts, setSloAlerts] = useState<any[]>([]);
  const [showSLOMonitoring, setShowSLOMonitoring] = useState<boolean>(false);

  // Schema version state
  const [schemaVersion, setSchemaVersion] = useState<string>('');

  // Upgrade overview state
  const [upgradeArtifacts, setUpgradeArtifacts] = useState<UpgradeArtifactsData | null>(null);
  const [upgradeStatus, setUpgradeStatus] = useState<UpgradeStatusResponse | null>(null);
  const [showUpgradeOverview, setShowUpgradeOverview] = useState<boolean>(false);

  // New component states
  const [showDiffViewer, setShowDiffViewer] = useState<boolean>(false);
  const [showExtensionFixer, setShowExtensionFixer] = useState<boolean>(false);
  const [showGoldenQueryRunner, setShowGoldenQueryRunner] = useState<boolean>(false);
  const [selectedVersions, setSelectedVersions] = useState<{from: string, to: string}>({from: '', to: ''});

  // New features state
  const [showAuditTrail, setShowAuditTrail] = useState<boolean>(false);
  const [auditTrail, setAuditTrail] = useState<any[]>([]);
  const [showRollbackConfirm, setShowRollbackConfirm] = useState<boolean>(false);
  const [rollbackTarget, setRollbackTarget] = useState<string>('');
  const [rollbackReason, setRollbackReason] = useState<string>('');
  const [showProgressTracker, setShowProgressTracker] = useState<boolean>(false);
  const [currentOperation, setCurrentOperation] = useState<string>('');
  const [operationProgress, setOperationProgress] = useState<number>(0);
  const [showHealthMonitor, setShowHealthMonitor] = useState<boolean>(false);
  const [healthMetrics, setHealthMetrics] = useState<any>({});
  const [showScheduler, setShowScheduler] = useState<boolean>(false);
  const [scheduledUpgrades, setScheduledUpgrades] = useState<any[]>([]);
  const [showApprovalWorkflow, setShowApprovalWorkflow] = useState<boolean>(false);
  const [pendingApprovals, setPendingApprovals] = useState<any[]>([]);
  const [_activeTab, _setActiveTab] = useState<number>(0);

  // Initialize from URL params and do an initial load
  useEffect(() => {
    // initialize filters from URL
    const sev = (searchParams.get('severity') || 'all').toLowerCase();
    const typ = searchParams.get('type') || 'all';
    if (['all','success','info','warning','error'].includes(sev)) setSeverityFilter(sev as any);
    if (typ) setTypeFilter(typ);
    // initialize refresh controls from URL
    const auto = (searchParams.get('autorefresh') || '1');
    const interval = parseInt(searchParams.get('interval') || '30', 10);
    setAutoRefresh(auto === '1' || auto.toLowerCase?.() === 'true');
    if (!Number.isNaN(interval) && interval > 0) setRefreshIntervalSec(interval);

    (async () => {
      const { versions, slo, canary } = await fetchVersions();
      setRows(versions.map((v, i) => ({ id: i, ...v })));
      setSlo(slo); setCanary(canary);
      try {
        const notes = await listNotifications();
        setNotifications(notes.slice(0, 5));
      } catch (e) {
        // non-blocking - failed to load notifications
      }
      try {
        const { schema_version } = await getSchemaVersion();
        setSchemaVersion(schema_version);
      } catch (e) {
        // non-blocking - failed to load schema version
      }
    })();
  }, [searchParams, setSearchParams]);

  // Persist filters and refresh controls to URL
  useEffect(() => {
    const next = new URLSearchParams(searchParams);
    next.set('severity', severityFilter);
    next.set('type', typeFilter);
    next.set('autorefresh', autoRefresh ? '1' : '0');
    next.set('interval', String(refreshIntervalSec));
    setSearchParams(next, { replace: true });
  }, [severityFilter, typeFilter, autoRefresh, refreshIntervalSec, searchParams, setSearchParams]);

  const refreshNow = async () => {
    try {
      const { slo, canary } = await fetchVersions();
      setSlo(slo); setCanary(canary);
      const notes = await listNotifications();
      setNotifications(notes.slice(0, 5));
    } catch (e) {
      // ignore refresh errors
    }
  };

  // Auto-refresh SLOs and Notifications (and canary)
  useEffect(() => {
    if (!autoRefresh) return;
    const handle = setInterval(async () => {
      try {
        const { slo, canary } = await fetchVersions();
        setSlo(slo); setCanary(canary);
        const notes = await listNotifications();
        setNotifications(notes.slice(0, 5));
      } catch (e) {
        // ignore refresh errors
      }
    }, refreshIntervalSec * 1000);
    return () => clearInterval(handle);
  }, [autoRefresh, refreshIntervalSec]);

  // Lifecycle action handlers
  const handlePrepareUpgrade = async (version: string) => {
    try {
      setLifecycleStep('preparing');
      setCurrentVersion(version);
      const result = await prepareUpgrade(version);
      setSchemaChanges(result.changes);
      setDeprecationMap(result.deprecation_map);
      setLifecycleStep('prepared');
      // Refresh versions
      const { versions } = await fetchVersions();
      setRows(versions.map((v,i)=>({id:i,...v})));
  } catch (error) {
  devError('Failed to prepare upgrade:', error);
      setLifecycleStep('error');
    }
  };

  const handleGenerateCore = async (version: string) => {
    try {
      setLifecycleStep('generating');
      await generateCore(version);
      setLifecycleStep('generated');
      // Refresh versions
      const { versions } = await fetchVersions();
      setRows(versions.map((v,i)=>({id:i,...v})));
    } catch (error) {
      devError('Failed to generate core:', error);
      setLifecycleStep('error');
    }
  };

  const handleMergeCustom = async (version: string) => {
    try {
      setLifecycleStep('merging');
      await mergeCustom(version);
      setLifecycleStep('merged');
      // Refresh versions
      const { versions } = await fetchVersions();
      setRows(versions.map((v,i)=>({id:i,...v})));
    } catch (error) {
      devError('Failed to merge custom:', error);
      setLifecycleStep('error');
    }
  };

  const handleValidate = async (version: string) => {
    try {
      setLifecycleStep('validating');
      const report = await validateVersion(version);
      setValidationReport(report);
      setLifecycleStep('validated');
    } catch (error) {
      devError('Failed to validate:', error);
      setLifecycleStep('error');
    }
  };

  const handleRunShadow = async (version: string) => {
    try {
      setLifecycleStep('shadow_testing');
      // Get some sample queries for shadow testing
      const queries = [
        'SELECT COUNT(*) FROM users',
        'SELECT * FROM orders LIMIT 10',
        'SELECT SUM(amount) FROM transactions WHERE created_at > NOW() - INTERVAL \'1 day\''
      ];
  await runShadow(version, queries);
      setLifecycleStep('shadow_completed');
    } catch (error) {
      devError('Failed to run shadow test:', error);
      setLifecycleStep('error');
    }
  };

  const handleArchive = async (version: string) => {
    try {
      setLifecycleStep('archiving');
      await archiveVersion(version);
      setLifecycleStep('archived');
      // Refresh versions
      const { versions } = await fetchVersions();
      setRows(versions.map((v,i)=>({id:i,...v})));
    } catch (error) {
      devError('Failed to archive:', error);
      setLifecycleStep('error');
    }
  };

  const handleScheduleBatchJob = async (version: string, jobType: string) => {
    try {
      const config = { version, queries: ['SELECT * FROM test_table LIMIT 100'] };
      await scheduleBatchJob(jobType, config);
      // Refresh batch jobs
      const jobs = await listBatchJobs();
      setBatchJobs(jobs);
      setShowBatchJobs(true);
    } catch (error) {
      devError('Failed to schedule batch job:', error);
    }
  };

  const handleLoadUpgradeOverview = async () => {
    try {
      // Use the first available version or default to '1.1.0'
      const targetVersion = rows.length > 0 ? rows[0].version : '1.1.0';
      const overview = await getUpgradeOverview(targetVersion);

      // Transform the response to match our component expectations
      const transformedArtifacts: UpgradeArtifactsData = {
        schema_version: overview.schema_version,
        changelog: overview.changelog,
        report: overview.report,
        aliases: overview.aliases
      };

      setUpgradeArtifacts(transformedArtifacts);
      setUpgradeStatus(overview.status);
      setShowUpgradeOverview(true);
    } catch (error) {
      devError('Failed to load upgrade overview:', error);
    }
  };

  const handleOverviewNavigate = (tool: "diff" | "fixer" | "queries") => {
    if (tool === "diff" && selectedVersions.from && selectedVersions.to) {
      setShowDiffViewer(true);
    } else if (tool === "fixer" && selectedVersions.from && selectedVersions.to) {
      setShowExtensionFixer(true);
    } else if (tool === "queries" && selectedVersions.from && selectedVersions.to) {
      setShowGoldenQueryRunner(true);
    }
    setShowUpgradeOverview(false);
  };

  const handleLoadSLOMetrics = async () => {
    try {
      const { metrics, alerts } = await getSLOMetrics('1h');
      setSloMetrics(metrics);
      setSloAlerts(alerts);
      setShowSLOMonitoring(true);
    } catch (error) {
      devError('Failed to load SLO metrics:', error);
    }
  };

  const handleLoadArtifacts = async () => {
    try {
      const artifactsList = await listArtifacts();
      setArtifacts(artifactsList);
      setShowArtifacts(true);
    } catch (error) {
      devError('Failed to load artifacts:', error);
    }
  };

  const handleOpenDiffViewer = (fromVersion: string, toVersion: string) => {
    setSelectedVersions({ from: fromVersion, to: toVersion });
    setShowDiffViewer(true);
  };

  // New feature handlers
  const handleLoadAuditTrail = async () => {
    try {
      // Mock audit trail data - in real implementation, this would come from backend
      const mockAuditTrail = [
        {
          id: '1',
          timestamp: new Date().toISOString(),
          action: 'ACTIVATE_VERSION',
          version: '2.1.0',
          user: 'admin',
          details: 'Activated version 2.1.0 for all tenants',
          status: 'SUCCESS',
          duration: 45000
        },
        {
          id: '2',
          timestamp: new Date(Date.now() - 3600000).toISOString(),
          action: 'START_CANARY',
          version: '2.1.0',
          user: 'steward',
          details: 'Started canary deployment for 10% of tenants',
          status: 'SUCCESS',
          duration: 120000
        },
        {
          id: '3',
          timestamp: new Date(Date.now() - 7200000).toISOString(),
          action: 'VALIDATE_VERSION',
          version: '2.0.5',
          user: 'system',
          details: 'Automated validation completed',
          status: 'WARNING',
          duration: 30000
        }
      ];
      setAuditTrail(mockAuditTrail);
      setShowAuditTrail(true);
    } catch (error) {
      devError('Failed to load audit trail:', error);
    }
  };

  // rollback confirm handler removed (not used) to satisfy strict unused checks

  const handleRollbackExecute = async () => {
    if (!rollbackTarget || !rollbackReason.trim()) return;

    try {
      setCurrentOperation(`Rolling back to ${rollbackTarget}`);
      setOperationProgress(0);
      setShowProgressTracker(true);

      // Simulate progress updates
      const progressInterval = setInterval(() => {
        setOperationProgress(prev => {
          if (prev >= 90) {
            clearInterval(progressInterval);
            return 90;
          }
          return prev + 10;
        });
      }, 1000);

      await rollbackVersion(rollbackTarget);

      setOperationProgress(100);
      setTimeout(() => {
        setShowProgressTracker(false);
        setShowRollbackConfirm(false);
        setRollbackTarget('');
        setRollbackReason('');
        // Refresh data
        refreshNow();
      }, 1000);

    } catch (error) {
      devError('Rollback failed:', error);
      setShowProgressTracker(false);
    }
  };

  const handleLoadHealthMetrics = async () => {
    try {
      // Mock health metrics - in real implementation, this would come from backend
      const mockMetrics = {
        overall: 'HEALTHY',
        components: {
          database: { status: 'HEALTHY', latency: 12, uptime: 99.9 },
          cache: { status: 'HEALTHY', hitRate: 94.2, uptime: 99.8 },
          api: { status: 'WARNING', latency: 245, errorRate: 2.1 },
          websocket: { status: 'HEALTHY', connections: 15, uptime: 100 }
        },
        alerts: [
          { level: 'WARNING', message: 'API latency above threshold', time: new Date().toISOString() },
          { level: 'INFO', message: 'Cache hit rate improved', time: new Date(Date.now() - 300000).toISOString() }
        ]
      };
      setHealthMetrics(mockMetrics);
      setShowHealthMonitor(true);
    } catch (error) {
      devError('Failed to load health metrics:', error);
    }
  };

  const handleLoadScheduledUpgrades = async () => {
    try {
      // Mock scheduled upgrades - in real implementation, this would come from backend
      const mockScheduled = [
        {
          id: '1',
          version: '2.2.0',
          scheduledTime: new Date(Date.now() + 86400000).toISOString(),
          type: 'CANARY',
          tenants: ['tenant-1', 'tenant-2'],
          status: 'PENDING',
          createdBy: 'admin'
        },
        {
          id: '2',
          version: '2.1.5',
          scheduledTime: new Date(Date.now() + 172800000).toISOString(),
          type: 'FULL_ROLLOUT',
          tenants: ['all'],
          status: 'APPROVED',
          createdBy: 'steward'
        }
      ];
      setScheduledUpgrades(mockScheduled);
      setShowScheduler(true);
    } catch (error) {
      devError('Failed to load scheduled upgrades:', error);
    }
  };

  const handleLoadPendingApprovals = async () => {
    try {
      // Mock pending approvals - in real implementation, this would come from backend
      const mockApprovals = [
        {
          id: '1',
          version: '2.2.0',
          type: 'PRODUCTION_DEPLOYMENT',
          requestedBy: 'developer',
          requestedAt: new Date(Date.now() - 3600000).toISOString(),
          riskLevel: 'HIGH',
          reviewers: ['admin', 'steward'],
          status: 'PENDING'
        }
      ];
      setPendingApprovals(mockApprovals);
      setShowApprovalWorkflow(true);
    } catch (error) {
      devError('Failed to load pending approvals:', error);
    }
  };

  const handleOpenExtensionFixer = (fromVersion: string, toVersion: string) => {
    setSelectedVersions({ from: fromVersion, to: toVersion });
    setShowExtensionFixer(true);
  };

  const handleOpenGoldenQueryRunner = (fromVersion: string, toVersion: string) => {
    setSelectedVersions({ from: fromVersion, to: toVersion });
    setShowGoldenQueryRunner(true);
  };

  const cols: GridColDef[] = [
    { field: 'health', headerName: 'Health', flex: 0.5, sortable: false,
      renderCell: (params: GridRenderCellParams<any, any>) => {
        const warnings = Array.isArray((params.row as any).warnings) ? (params.row as any).warnings : [];
        const sev: 'success'|'warning' = warnings.length > 0 ? 'warning' : 'success';
        return (
          <Stack direction="row" spacing={0.5} alignItems="center">
            {sev === 'success' ? (
              <Tooltip title="No warnings"><CheckCircleIcon fontSize="small" color="success" /></Tooltip>
            ) : (
              <Tooltip title="Has warnings"><WarningAmberIcon fontSize="small" color="warning" /></Tooltip>
            )}
            <Chip size="small" label={sev === 'success' ? 'OK' : 'Warn'} color={sev === 'success' ? 'success' : 'warning'} />
          </Stack>
        );
      }
    },
    { field: 'version', headerName: 'Version', flex: 1 },
    { field: 'schema_hash', headerName: 'Schema Hash', flex: 1 },
    { field: 'status', headerName: 'Status', flex: 0.6, renderCell: (p) => <Chip size="small" label={p.value} color={p.value==='active'?'success':p.value==='canary'?'warning':p.value==='preview'?'info':'default'} /> },
    { field: 'warnings', headerName: 'Warnings', flex: 1.2, renderCell: (p) => (Array.isArray(p.row.warnings) ? p.row.warnings.join('; ') : '') },
    { field: 'created_at', headerName: 'Created', flex: 0.8 },
    { field: 'activated_at', headerName: 'Activated', flex: 0.8 },
    {
      field: 'actions', headerName: 'Actions', flex: 2.5, sortable: false, renderCell: (params) => {
        const v = params.row.version as string;
        const status = params.row.status as string;
        return (
          <Stack direction="row" spacing={1} flexWrap="wrap">
            <Button size="small" component={Link} to={`/upgrade/diff?from=active&to=${encodeURIComponent(v)}`}>Preview</Button>
            <Button size="small" onClick={async ()=>{ await setPreview(v); const { versions } = await fetchVersions(); setRows(versions.map((v,i)=>({id:i,...v}))); }}>Canary Prep</Button>
            <Button size="small" color="warning" onClick={async ()=>{ await startCanary(v, []); const { canary } = await fetchVersions(); setCanary(canary); }}>Start Canary</Button>
            <Button size="small" color="success" onClick={async ()=>{ await activateVersion(v); const { versions } = await fetchVersions(); setRows(versions.map((v,i)=>({id:i,...v}))); }}>Activate</Button>

            {/* New Lifecycle Actions */}
            {status === 'available' && (
              <>
                <Button size="small" variant="outlined" color="info" onClick={() => handlePrepareUpgrade(v)}>Prepare</Button>
                <Button size="small" variant="outlined" color="secondary" onClick={() => handleGenerateCore(v)}>Generate Core</Button>
                <Button size="small" variant="outlined" color="secondary" onClick={() => handleMergeCustom(v)}>Merge Custom</Button>
                <Button size="small" variant="outlined" color="warning" onClick={() => handleValidate(v)}>Validate</Button>
                <Button size="small" variant="outlined" color="error" onClick={() => handleRunShadow(v)}>Shadow Test</Button>
                <Button size="small" variant="outlined" onClick={() => handleArchive(v)}>Archive</Button>
              </>
            )}

            {/* Batch Job Actions */}
            <Button size="small" variant="text" onClick={() => handleScheduleBatchJob(v, 'shadow_test')}>Schedule Shadow</Button>

            {/* New Upgrade Tools */}
            <Button size="small" variant="outlined" startIcon={<CompareIcon />} onClick={() => handleOpenDiffViewer('active', v)}>Diff</Button>
            <Button size="small" variant="outlined" startIcon={<BuildIcon />} onClick={() => handleOpenExtensionFixer('active', v)}>Fix Extensions</Button>
            <Button size="small" variant="outlined" startIcon={<QueryStatsIcon />} onClick={() => handleOpenGoldenQueryRunner('active', v)}>Golden Queries</Button>
          </Stack>
        );
      }
    }
  ];

  return (
    <Box sx={{ p: 3 }}>
  <Breadcrumbs sx={{ mb: 2 }}><MUILink component={BlockableLink} to="/">Home</MUILink><MUILink component={BlockableLink} to="/upgrade">Upgrade Center</MUILink><Typography>Versions</Typography></Breadcrumbs>
      <Typography variant="h4" sx={{ mb: 2 }}>Core Versions</Typography>

      {/* View Toggle */}
      <Box sx={{ mb: 3, display: 'flex', gap: 2, alignItems: 'center' }}>
        <Button
          variant={!showInteractiveTable ? "contained" : "outlined"}
          color="primary"
          onClick={() => setShowInteractiveTable(false)}
        >
          Classic View
        </Button>
        <Button
          variant={showInteractiveTable ? "contained" : "outlined"}
          color="primary"
          onClick={() => setShowInteractiveTable(true)}
        >
          Interactive Table
        </Button>
      </Box>

      {/* Upgrade Overview Button */}
      <Box sx={{ mb: 3, display: 'flex', gap: 2 }}>
        <Button
          variant="contained"
          color="primary"
          onClick={handleLoadUpgradeOverview}
          startIcon={<AssessmentIcon />}
        >
          Upgrade Overview
        </Button>
      </Box>

      {/* Upgrade Overview Modal/Dialog */}
      {showUpgradeOverview && upgradeArtifacts && upgradeStatus && (
        <Box sx={{ mb: 3, p: 3, border: '1px solid #e0e0e0', borderRadius: 2, backgroundColor: 'background.paper' }}>
          <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', mb: 2 }}>
            <Typography variant="h5">Upgrade Overview</Typography>
            <Button
              variant="outlined"
              size="small"
              onClick={() => setShowUpgradeOverview(false)}
            >
              Close
            </Button>
          </Box>
          <UpgradeOverview
            artifacts={upgradeArtifacts}
            status={upgradeStatus}
            onNavigate={handleOverviewNavigate}
          />
        </Box>
      )}

      {/* Schema Version Display */}
      {schemaVersion && (
        <Box sx={{ mb: 3 }}>
          <SchemaVersionDisplay
            artifact={{
              schema_version: schemaVersion,
              report: {
                core_version: schemaVersion,
                previous_version: '',
                generated_at: new Date().toISOString(),
                schema_hash: '',
                summary: {
                  cubes_added: 0,
                  cubes_removed: 0,
                  cubes_changed: 0,
                  views_added: 0,
                  views_removed: 0,
                  views_changed: 0,
                  breaking_changes: 0,
                  warnings: 0,
                },
                cubes: [],
                views: [],
                governance: [],
                pre_aggregations: [],
                warnings: [],
              },
              aliases: {
                core_version: schemaVersion,
                previous_version: '',
                generated_at: new Date().toISOString(),
                aliases: [],
              }
            }}
            backendVersion={schemaVersion}
          />
        </Box>
      )}

      {/* Lifecycle Progress Indicator */}
      {lifecycleStep !== 'idle' && (
        <Card variant="outlined" sx={{ mb: 2 }}>
          <CardContent>
            <Typography variant="h6" sx={{ mb: 2 }}>Upgrade Lifecycle Progress</Typography>
            <Stack spacing={2}>
              <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                <PlayArrowIcon color="info" />
                <Typography variant="body2">Prepare Upgrade</Typography>
                <Chip size="small" label={lifecycleStep === 'preparing' ? 'Running' : lifecycleStep === 'prepared' || ['generated', 'merged', 'validated', 'shadow_completed', 'archived'].includes(lifecycleStep) ? 'Complete' : 'Pending'} 
                      color={lifecycleStep === 'preparing' ? 'warning' : ['prepared', 'generated', 'merged', 'validated', 'shadow_completed', 'archived'].includes(lifecycleStep) ? 'success' : 'default'} />
              </Box>

              <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                <AssessmentIcon color="secondary" />
                <Typography variant="body2">Generate Core Views</Typography>
                <Chip size="small" label={lifecycleStep === 'generating' ? 'Running' : ['generated', 'merged', 'validated', 'shadow_completed', 'archived'].includes(lifecycleStep) ? 'Complete' : 'Pending'} 
                      color={lifecycleStep === 'generating' ? 'warning' : ['generated', 'merged', 'validated', 'shadow_completed', 'archived'].includes(lifecycleStep) ? 'success' : 'default'} />
              </Box>

              <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                <ArchiveIcon color="secondary" />
                <Typography variant="body2">Merge Custom Extensions</Typography>
                <Chip size="small" label={lifecycleStep === 'merging' ? 'Running' : ['merged', 'validated', 'shadow_completed', 'archived'].includes(lifecycleStep) ? 'Complete' : 'Pending'} 
                      color={lifecycleStep === 'merging' ? 'warning' : ['merged', 'validated', 'shadow_completed', 'archived'].includes(lifecycleStep) ? 'success' : 'default'} />
              </Box>

              <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                <CheckCircleIcon color="warning" />
                <Typography variant="body2">Validate & Test</Typography>
                <Chip size="small" label={lifecycleStep === 'validating' ? 'Running' : ['validated', 'shadow_completed', 'archived'].includes(lifecycleStep) ? 'Complete' : 'Pending'} 
                      color={lifecycleStep === 'validating' ? 'warning' : ['validated', 'shadow_completed', 'archived'].includes(lifecycleStep) ? 'success' : 'default'} />
              </Box>

              <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                <RefreshIcon color="error" />
                <Typography variant="body2">Shadow Testing</Typography>
                <Chip size="small" label={lifecycleStep === 'shadow_testing' ? 'Running' : ['shadow_completed', 'archived'].includes(lifecycleStep) ? 'Complete' : 'Pending'} 
                      color={lifecycleStep === 'shadow_testing' ? 'warning' : ['shadow_completed', 'archived'].includes(lifecycleStep) ? 'success' : 'default'} />
              </Box>

              <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                <ArchiveIcon color="success" />
                <Typography variant="body2">Archive & Deploy</Typography>
                <Chip size="small" label={lifecycleStep === 'archiving' ? 'Running' : lifecycleStep === 'archived' ? 'Complete' : 'Pending'} 
                      color={lifecycleStep === 'archiving' ? 'warning' : lifecycleStep === 'archived' ? 'success' : 'default'} />
              </Box>
            </Stack>

            {lifecycleStep === 'error' && (
              <Alert severity="error" sx={{ mt: 2 }}>
                An error occurred during the upgrade process. Please check the logs and try again.
              </Alert>
            )}
          </CardContent>
        </Card>
      )}

      <Grid container spacing={2}>
        <Grid item xs={12} md={8}>
          {showInteractiveTable ? (
            <Card variant="outlined" sx={{ p: 2 }}>
              <Typography variant="h6" sx={{ mb: 2 }}>Interactive Versions Table</Typography>
                <VersionsTable
                onSelectVersion={() => {
                  // Handle version selection - could navigate to details or open modal
                }}
              />
            </Card>
          ) : (
            <Box sx={{ height: 420, width: '100%' }}>
              <Suspense fallback={<Typography variant="body2">Loading table…</Typography>}>
                {
                  <LazyDataGrid {...({ rows, columns: cols, pageSizeOptions: [5, 10, 25], initialState: { pagination: { paginationModel: { pageSize: 10 } } } } as any)} />
                }
              </Suspense>
            </Box>
          )}
        </Grid>
        <Grid item xs={12} md={4}>
          <Card variant="outlined" sx={{ mb: 2 }}>
            <CardContent>
              <Stack direction="row" alignItems="center" justifyContent="space-between" sx={{ mb: 1 }}>
                <Typography variant="h6">SLO Panel</Typography>
                <Stack direction="row" spacing={1}>
                  <Button size="small" component={MUILink as any} href="/api/admin/qos/config" target="_blank" rel="noopener">QoS Config</Button>
                  <Button size="small" component={MUILink as any} href="/api/admin/qos/tenants" target="_blank" rel="noopener">Tenant QoS</Button>
                </Stack>
              </Stack>
              {slo ? (
                <Stack spacing={1}>
                  <Typography variant="body2">Error rate: {(slo.error_rate*100).toFixed(2)}%</Typography>
                  <Typography variant="body2">P95 latency: {slo.p95_latency_ms} ms</Typography>
                  <Typography variant="body2">Shadow diffs: {(slo.shadow_diff_rate*100).toFixed(2)}%</Typography>
                  <Typography variant="body2">Pre‑agg rebuild: {slo.preagg_rebuild_ms} ms</Typography>
                  <Typography variant="body2">Cache hit ratio: {(slo.cache_hit_ratio*100).toFixed(1)}%</Typography>
                </Stack>
              ) : (<Typography variant="body2">Loading...</Typography>)}
            </CardContent>
          </Card>
          <Card variant="outlined" sx={{ mb: 2 }}>
            <CardContent>
              <Stack direction="row" alignItems="center" justifyContent="space-between" sx={{ mb: 1 }}>
                <Typography variant="h6">Recent Upgrade Notifications</Typography>
                <Stack direction="row" spacing={1} alignItems="center">
                  <FormControl size="small" sx={{ minWidth: 110 }}>
                    <InputLabel id="refresh-interval-label">Refresh</InputLabel>
                    <Select labelId="refresh-interval-label" label="Refresh" value={String(refreshIntervalSec)} onChange={(e)=> setRefreshIntervalSec(parseInt(e.target.value as string, 10))}>
                      <MenuItem value={15}>Every 15s</MenuItem>
                      <MenuItem value={30}>Every 30s</MenuItem>
                      <MenuItem value={60}>Every 60s</MenuItem>
                    </Select>
                  </FormControl>
                  <Button size="small" variant="outlined" onClick={()=> setAutoRefresh(v => !v)}>{autoRefresh ? 'Pause' : 'Resume'}</Button>
                  <Button size="small" startIcon={<RefreshIcon />} onClick={refreshNow}>Refresh now</Button>
                  <MUILink component={Link} to="/notifications" underline="hover">View all</MUILink>
                </Stack>
              </Stack>
              <Stack direction={{ xs: 'column', sm: 'row' }} spacing={1} sx={{ mb: 1 }}>
                <FormControl size="small" sx={{ minWidth: 120 }}>
                  <InputLabel id="severity-filter-label">Severity</InputLabel>
                  <Select labelId="severity-filter-label" label="Severity" value={severityFilter} onChange={(e)=>setSeverityFilter(e.target.value as any)}>
                    <MenuItem value="all">All</MenuItem>
                    <MenuItem value="success">Success</MenuItem>
                    <MenuItem value="info">Info</MenuItem>
                    <MenuItem value="warning">Warning</MenuItem>
                    <MenuItem value="error">Error</MenuItem>
                  </Select>
                </FormControl>
                <FormControl size="small" sx={{ minWidth: 140 }}>
                  <InputLabel id="type-filter-label">Type</InputLabel>
                  <Select labelId="type-filter-label" label="Type" value={typeFilter} onChange={(e)=>setTypeFilter(e.target.value)}>
                    <MenuItem value="all">All</MenuItem>
                    {[...new Set(notifications.map(n=>n.type))].map(t => (
                      <MenuItem key={t} value={t}>{t}</MenuItem>
                    ))}
                  </Select>
                </FormControl>
              </Stack>
              {notifications.length === 0 ? (
                <Typography variant="body2" color="text.secondary">No recent notifications</Typography>
              ) : (
                <List dense sx={{ p: 0 }}>
                  {notifications
                      .filter(n => severityFilter === 'all' || n.severity === severityFilter)
                      .filter(n => typeFilter === 'all' || n.type === typeFilter)
                      .map(n => (
                    <ListItem key={n.id} disableGutters sx={{ display: 'block' }}>
                      <Stack direction="row" spacing={1} alignItems="center">
                        <Chip size="small" label={n.severity} color={n.severity === 'success' ? 'success' : n.severity === 'warning' ? 'warning' : n.severity === 'error' ? 'error' : 'info'} />
                        <Chip size="small" variant="outlined" label={n.type} />
                        <Typography variant="body2" sx={{ fontFamily: 'monospace' }}>{new Date(n.created_at).toLocaleString()}</Typography>
                      </Stack>
                      <ListItemText primaryTypographyProps={{ variant: 'body2' }} primary={n.message} />
                    </ListItem>
                  ))}
                </List>
              )}
            </CardContent>
          </Card>
          <Card variant="outlined">
            <CardContent>
              <Typography variant="h6" sx={{ mb: 1 }}>Canary</Typography>
              {canary ? (
                <Stack spacing={1}>
                  <Typography variant="body2">Version: {canary.version}</Typography>
                  <Typography variant="body2">Tenants: {canary.tenants?.join(', ') || '—'}</Typography>
                  <Typography variant="body2">Until: {new Date(canary.until).toLocaleString()}</Typography>
                </Stack>
              ) : (<Typography variant="body2">No active canary</Typography>)}
            </CardContent>
          </Card>

          {/* Lifecycle Status Card */}
          {lifecycleStep !== 'idle' && (
            <Card variant="outlined" sx={{ mt: 2 }}>
              <CardContent>
                <Typography variant="h6" sx={{ mb: 1 }}>Lifecycle Status</Typography>
                <Stack spacing={1}>
                  <Typography variant="body2">Version: {currentVersion}</Typography>
                  <Typography variant="body2">Step: {lifecycleStep}</Typography>
                  {validationReport && (
                    <Typography variant="body2" color={validationReport.passed ? 'success.main' : 'error.main'}>
                      Validation: {validationReport.passed ? 'PASSED' : 'FAILED'}
                    </Typography>
                  )}
                </Stack>
              </CardContent>
            </Card>
          )}

          {/* Batch Jobs Card */}
          <Card variant="outlined" sx={{ mt: 2 }}>
            <CardContent>
              <Stack direction="row" alignItems="center" justifyContent="space-between" sx={{ mb: 1 }}>
                <Typography variant="h6">Batch Jobs</Typography>
                <Button size="small" onClick={async () => { const jobs = await listBatchJobs(); setBatchJobs(jobs); setShowBatchJobs(!showBatchJobs); }}>
                  {showBatchJobs ? 'Hide' : 'Show'}
                </Button>
              </Stack>
              {showBatchJobs && (
                <Stack spacing={1}>
                  {batchJobs.length === 0 ? (
                    <Typography variant="body2" color="text.secondary">No batch jobs</Typography>
                  ) : (
                    batchJobs.slice(0, 3).map(job => (
                      <Stack key={job.job_id} direction="row" justifyContent="space-between" alignItems="center">
                        <Typography variant="body2">{job.job_type}</Typography>
                        <Chip size="small" label={job.status} color={job.status === 'completed' ? 'success' : job.status === 'running' ? 'warning' : 'default'} />
                      </Stack>
                    ))
                  )}
                </Stack>
              )}
            </CardContent>
          </Card>

          {/* Artifacts Card */}
          <Card variant="outlined" sx={{ mt: 2 }}>
            <CardContent>
              <Stack direction="row" alignItems="center" justifyContent="space-between" sx={{ mb: 1 }}>
                <Typography variant="h6">Artifacts</Typography>
                <Button size="small" onClick={() => handleLoadArtifacts()}>Load</Button>
              </Stack>
              {showArtifacts && (
                <Stack spacing={1}>
                  {artifacts.length === 0 ? (
                    <Typography variant="body2" color="text.secondary">No artifacts</Typography>
                  ) : (
                    artifacts.slice(0, 3).map(artifact => (
                      <Stack key={artifact.artifact_id} spacing={0.5}>
                        <Typography variant="body2">{artifact.type}</Typography>
                        <Typography variant="caption" color="text.secondary">
                          {artifact.size_bytes} bytes • {new Date(artifact.created_at).toLocaleDateString()}
                        </Typography>
                      </Stack>
                    ))
                  )}
                </Stack>
              )}
            </CardContent>
          </Card>

          {/* SLO Monitoring Card */}
          <Card variant="outlined" sx={{ mt: 2 }}>
            <CardContent>
              <Stack direction="row" alignItems="center" justifyContent="space-between" sx={{ mb: 1 }}>
                <Typography variant="h6">SLO Monitoring</Typography>
                <Button size="small" onClick={handleLoadSLOMetrics}>Monitor</Button>
              </Stack>
              {showSLOMonitoring && (
                <Stack spacing={1}>
                  {sloAlerts.length > 0 && (
                    <Typography variant="body2" color="warning.main">
                      {sloAlerts.length} active alert{sloAlerts.length !== 1 ? 's' : ''}
                    </Typography>
                  )}
                  {sloMetrics.length > 0 && (
                    <Stack spacing={0.5}>
                      <Typography variant="body2">
                        Error Rate: {(sloMetrics[sloMetrics.length - 1]?.error_rate * 100).toFixed(2)}%
                      </Typography>
                      <Typography variant="body2">
                        P95 Latency: {sloMetrics[sloMetrics.length - 1]?.p95_latency_ms}ms
                      </Typography>
                      <Typography variant="body2">
                        Cache Hit: {(sloMetrics[sloMetrics.length - 1]?.cache_hit_ratio * 100).toFixed(1)}%
                      </Typography>
                    </Stack>
                  )}
                </Stack>
              )}
            </CardContent>
          </Card>

          {/* New Feature Cards */}
          <Card variant="outlined" sx={{ mt: 2 }}>
            <CardContent>
              <Typography variant="h6" sx={{ mb: 1 }}>Workflow Tools</Typography>
              <Stack spacing={1}>
                <Button
                  size="small"
                  startIcon={<HistoryIcon />}
                  onClick={handleLoadAuditTrail}
                  variant="outlined"
                  fullWidth
                >
                  Audit Trail
                </Button>
                <Button
                  size="small"
                  startIcon={<HealthIcon />}
                  onClick={handleLoadHealthMetrics}
                  variant="outlined"
                  fullWidth
                >
                  Health Monitor
                </Button>
                <Button
                  size="small"
                  startIcon={<ScheduleIcon />}
                  onClick={handleLoadScheduledUpgrades}
                  variant="outlined"
                  fullWidth
                >
                  Scheduler
                </Button>
                <Button
                  size="small"
                  startIcon={<ApprovalIcon />}
                  onClick={handleLoadPendingApprovals}
                  variant="outlined"
                  fullWidth
                >
                  Approvals
                </Button>
              </Stack>
            </CardContent>
          </Card>

          {/* Diff Viewer */}
          <Accordion expanded={showDiffViewer} onChange={() => setShowDiffViewer(!showDiffViewer)} sx={{ mt: 2 }}>
            <AccordionSummary expandIcon={<ExpandMoreIcon />}>
              <Stack direction="row" alignItems="center" spacing={1}>
                <CompareIcon />
                <Typography variant="h6">Diff Viewer</Typography>
                {selectedVersions.from && selectedVersions.to && (
                  <Chip size="small" label={`${selectedVersions.from} → ${selectedVersions.to}`} />
                )}
              </Stack>
            </AccordionSummary>
            <AccordionDetails>
              {showDiffViewer && selectedVersions.from && selectedVersions.to && (
                <DiffViewer fromVersion={selectedVersions.from} toVersion={selectedVersions.to} />
              )}
            </AccordionDetails>
          </Accordion>

          {/* Extension Fixer */}
          <Accordion expanded={showExtensionFixer} onChange={() => setShowExtensionFixer(!showExtensionFixer)} sx={{ mt: 1 }}>
            <AccordionSummary expandIcon={<ExpandMoreIcon />}>
              <Stack direction="row" alignItems="center" spacing={1}>
                <BuildIcon />
                <Typography variant="h6">Extension Fixer</Typography>
                {selectedVersions.from && selectedVersions.to && (
                  <Chip size="small" label={`${selectedVersions.from} → ${selectedVersions.to}`} />
                )}
              </Stack>
            </AccordionSummary>
            <AccordionDetails>
              {showExtensionFixer && selectedVersions.from && selectedVersions.to && (
                <ExtensionFixer fromVersion={selectedVersions.from} toVersion={selectedVersions.to} />
              )}
            </AccordionDetails>
          </Accordion>

          {/* Golden Query Runner */}
          <Accordion expanded={showGoldenQueryRunner} onChange={() => setShowGoldenQueryRunner(!showGoldenQueryRunner)} sx={{ mt: 1 }}>
            <AccordionSummary expandIcon={<ExpandMoreIcon />}>
              <Stack direction="row" alignItems="center" spacing={1}>
                <QueryStatsIcon />
                <Typography variant="h6">Golden Query Runner</Typography>
                {selectedVersions.from && selectedVersions.to && (
                  <Chip size="small" label={`${selectedVersions.from} → ${selectedVersions.to}`} />
                )}
              </Stack>
            </AccordionSummary>
            <AccordionDetails>
              {showGoldenQueryRunner && selectedVersions.from && selectedVersions.to && (
                <GoldenQueryRunner fromVersion={selectedVersions.from} toVersion={selectedVersions.to} />
              )}
            </AccordionDetails>
          </Accordion>

          <Box sx={{ mt: 2 }}>
            <Button size="small" color="inherit" onClick={async ()=>{ 
              // For demo purposes, rollback the first active version
              const activeVersion = rows.find(r => r.status === 'active' || r.status === 'canary');
              if (activeVersion) {
                await rollbackVersion(activeVersion.version); 
                const { versions, canary } = await fetchVersions(); 
                setRows(versions.map((v,i)=>({id:i,...v}))); 
                setCanary(canary); 
              }
            }}>Rollback</Button>
          </Box>
        </Grid>
      </Grid>

      {/* New Feature Dialogs */}

      {/* Audit Trail Dialog */}
      <Dialog open={showAuditTrail} onClose={() => setShowAuditTrail(false)} maxWidth="lg" fullWidth>
          <ModalHeader title={(
            <Stack direction="row" alignItems="center" spacing={1}>
              <HistoryIcon />
              <Typography variant="h6">Audit Trail</Typography>
            </Stack>
          )} onClose={() => setShowAuditTrail(false)} />
        <DialogContent>
          <TableContainer component={Paper}>
            <Table>
              <TableHead>
                <TableRow>
                  <TableCell>Timestamp</TableCell>
                  <TableCell>Action</TableCell>
                  <TableCell>Version</TableCell>
                  <TableCell>User</TableCell>
                  <TableCell>Status</TableCell>
                  <TableCell>Duration</TableCell>
                </TableRow>
              </TableHead>
              <TableBody>
                {auditTrail.map((entry) => (
                  <TableRow key={entry.id}>
                    <TableCell>{new Date(entry.timestamp).toLocaleString()}</TableCell>
                    <TableCell>{entry.action}</TableCell>
                    <TableCell>{entry.version}</TableCell>
                    <TableCell>{entry.user}</TableCell>
                    <TableCell>
                      <Chip
                        size="small"
                        label={entry.status}
                        color={entry.status === 'SUCCESS' ? 'success' : entry.status === 'WARNING' ? 'warning' : 'error'}
                      />
                    </TableCell>
                    <TableCell>{entry.duration}ms</TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </TableContainer>
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setShowAuditTrail(false)}>Close</Button>
        </DialogActions>
      </Dialog>

      {/* Rollback Confirmation Dialog */}
      <Dialog open={showRollbackConfirm} onClose={() => setShowRollbackConfirm(false)} maxWidth="sm" fullWidth>
        <ModalHeader title={(
          <Stack direction="row" alignItems="center" spacing={1}>
            <WarningIcon />
            <Typography variant="h6">Confirm Rollback</Typography>
          </Stack>
        )} onClose={() => setShowRollbackConfirm(false)} />
        <DialogContent>
          <Typography variant="body1" sx={{ mb: 2 }}>
            Are you sure you want to rollback to version {rollbackTarget}?
          </Typography>
          <Typography variant="body2" color="text.secondary" sx={{ mb: 2 }}>
            This action will deactivate the current version and activate the selected version for all tenants.
          </Typography>
          <TextField
            fullWidth
            label="Reason for rollback"
            multiline
            rows={3}
            value={rollbackReason}
            onChange={(e) => setRollbackReason(e.target.value)}
            placeholder="Please provide a reason for this rollback..."
            required
          />
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setShowRollbackConfirm(false)}>Cancel</Button>
          <Button
            onClick={handleRollbackExecute}
            variant="contained"
            color="warning"
            disabled={!rollbackReason.trim()}
          >
            Confirm Rollback
          </Button>
        </DialogActions>
      </Dialog>

      {/* Health Monitor Dialog */}
      <Dialog open={showHealthMonitor} onClose={() => setShowHealthMonitor(false)} maxWidth="md" fullWidth>
        <ModalHeader title={(
          <Stack direction="row" alignItems="center" spacing={1}>
            <HealthIcon />
            <Typography variant="h6">Health Monitor</Typography>
          </Stack>
        )} onClose={() => setShowHealthMonitor(false)} />
        <DialogContent>
          <Stack spacing={2}>
            <Box>
              <Typography variant="h6" sx={{ mb: 1 }}>Overall Status</Typography>
              <Chip
                label={healthMetrics.overall}
                color={healthMetrics.overall === 'HEALTHY' ? 'success' : 'warning'}
                size="medium"
              />
            </Box>

            <Box>
              <Typography variant="h6" sx={{ mb: 1 }}>Component Status</Typography>
              <Grid container spacing={2}>
                {Object.entries(healthMetrics.components).map(([component, status]: [string, any]) => (
                  <Grid item xs={12} sm={6} key={component}>
                    <Card variant="outlined">
                      <CardContent>
                        <Typography variant="subtitle1" sx={{ textTransform: 'capitalize' }}>
                          {component}
                        </Typography>
                        <Stack direction="row" alignItems="center" spacing={1} sx={{ mt: 1 }}>
                          <Chip
                            size="small"
                            label={status.status}
                            color={status.status === 'HEALTHY' ? 'success' : 'warning'}
                          />
                          {status.latency && (
                            <Typography variant="body2">Latency: {status.latency}ms</Typography>
                          )}
                          {status.uptime && (
                            <Typography variant="body2">Uptime: {status.uptime}%</Typography>
                          )}
                        </Stack>
                      </CardContent>
                    </Card>
                  </Grid>
                ))}
              </Grid>
            </Box>

            <Box>
              <Typography variant="h6" sx={{ mb: 1 }}>Recent Alerts</Typography>
              <List dense>
                {(healthMetrics.alerts as Array<{ level: string; message: string; time: string }>).map((alert: { level: string; message: string; time: string }, index: number) => (
                  <ListItem key={index}>
                    <ListItemText
                      primary={
                        <Stack direction="row" alignItems="center" spacing={1}>
                          {alert.level === 'WARNING' ? <WarningIcon color="warning" /> :
                           alert.level === 'ERROR' ? <ErrorIcon color="error" /> :
                           <InfoIcon color="info" />}
                          <Typography variant="body2">{alert.message}</Typography>
                        </Stack>
                      }
                      secondary={new Date(alert.time).toLocaleString()}
                    />
                  </ListItem>
                ))}
              </List>
            </Box>
          </Stack>
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setShowHealthMonitor(false)}>Close</Button>
        </DialogActions>
      </Dialog>

      {/* Scheduler Dialog */}
      <Dialog open={showScheduler} onClose={() => setShowScheduler(false)} maxWidth="lg" fullWidth>
        <ModalHeader title={<Stack direction="row" alignItems="center" spacing={1}><ScheduleIcon />Scheduled Upgrades</Stack>} onClose={() => setShowScheduler(false)} />
        <DialogContent>
          <TableContainer component={Paper}>
            <Table>
              <TableHead>
                <TableRow>
                  <TableCell>Version</TableCell>
                  <TableCell>Scheduled Time</TableCell>
                  <TableCell>Type</TableCell>
                  <TableCell>Tenants</TableCell>
                  <TableCell>Status</TableCell>
                  <TableCell>Created By</TableCell>
                  <TableCell>Actions</TableCell>
                </TableRow>
              </TableHead>
              <TableBody>
                {scheduledUpgrades.map((upgrade) => (
                  <TableRow key={upgrade.id}>
                    <TableCell>{upgrade.version}</TableCell>
                    <TableCell>{new Date(upgrade.scheduledTime).toLocaleString()}</TableCell>
                    <TableCell>{upgrade.type}</TableCell>
                    <TableCell>{upgrade.tenants.join(', ')}</TableCell>
                    <TableCell>
                      <Chip
                        size="small"
                        label={upgrade.status}
                        color={upgrade.status === 'APPROVED' ? 'success' : upgrade.status === 'PENDING' ? 'warning' : 'default'}
                      />
                    </TableCell>
                    <TableCell>{upgrade.createdBy}</TableCell>
                    <TableCell>
                      <Stack direction="row" spacing={1}>
                        <Button size="small" variant="outlined">Edit</Button>
                        <Button size="small" variant="outlined" color="error">Cancel</Button>
                      </Stack>
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </TableContainer>
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setShowScheduler(false)}>Close</Button>
          <Button variant="contained" startIcon={<ScheduleIcon />}>Schedule New Upgrade</Button>
        </DialogActions>
      </Dialog>

      {/* Approval Workflow Dialog */}
      <Dialog open={showApprovalWorkflow} onClose={() => setShowApprovalWorkflow(false)} maxWidth="lg" fullWidth>
        <ModalHeader title={<Stack direction="row" alignItems="center" spacing={1}><ApprovalIcon />Pending Approvals</Stack>} onClose={() => setShowApprovalWorkflow(false)} />
        <DialogContent>
          <TableContainer component={Paper}>
            <Table>
              <TableHead>
                <TableRow>
                  <TableCell>Version</TableCell>
                  <TableCell>Type</TableCell>
                  <TableCell>Requested By</TableCell>
                  <TableCell>Requested At</TableCell>
                  <TableCell>Risk Level</TableCell>
                  <TableCell>Reviewers</TableCell>
                  <TableCell>Actions</TableCell>
                </TableRow>
              </TableHead>
              <TableBody>
                {pendingApprovals.map((approval) => (
                  <TableRow key={approval.id}>
                    <TableCell>{approval.version}</TableCell>
                    <TableCell>{approval.type}</TableCell>
                    <TableCell>{approval.requestedBy}</TableCell>
                    <TableCell>{new Date(approval.requestedAt).toLocaleString()}</TableCell>
                    <TableCell>
                      <Chip
                        size="small"
                        label={approval.riskLevel}
                        color={approval.riskLevel === 'HIGH' ? 'error' : approval.riskLevel === 'MEDIUM' ? 'warning' : 'success'}
                      />
                    </TableCell>
                    <TableCell>{approval.reviewers.join(', ')}</TableCell>
                    <TableCell>
                      <Stack direction="row" spacing={1}>
                        <Button size="small" variant="contained" color="success">Approve</Button>
                        <Button size="small" variant="outlined" color="error">Reject</Button>
                      </Stack>
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </TableContainer>
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setShowApprovalWorkflow(false)}>Close</Button>
        </DialogActions>
      </Dialog>

      {/* Progress Tracker Dialog */}
      <Dialog open={showProgressTracker} onClose={() => {}} maxWidth="sm" fullWidth>
        <ModalHeader title={<Typography variant="h6">{currentOperation}</Typography>} onClose={() => {}} />
        <DialogContent>
          <Stack spacing={2}>
            <LinearProgress variant="determinate" value={operationProgress} />
            <Typography variant="body2" color="text.secondary">
              {operationProgress}% complete
            </Typography>
          </Stack>
        </DialogContent>
      </Dialog>
    </Box>
  );
}
