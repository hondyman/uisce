import React, { useState, useEffect } from 'react';
import {
  Box,
  Typography,
  Paper,
  Stack,
  Grid,
  Chip,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  Button,
  Card,
  CardContent,
  CardHeader,
  Divider,
  CircularProgress,
  Alert,
  Tabs,
  Tab,
  Tooltip,
} from '@mui/material';
import {
  Storage as StorageIcon,
  CheckCircle as ResolvedIcon,
  Warning as UnresolvedIcon,
  AutoAwesome as AIIcon,
  Code as CodeIcon,
  AccountTree as GraphIcon,
  Hub as TierIcon,
  CloudQueue as CloudIcon,
  ContentCopy as CopyIcon,
} from '@mui/icons-material';
import { fetchAPI } from '../../../../api';
import { useNotification } from '../../../../hooks/useNotification';

const MAX_FIELDS_DISPLAY = 10;

interface BindingsTabProps {
  bindings: any[];
  businessObject?: any;
}

export function BindingsTab({ bindings, businessObject }: BindingsTabProps) {
  const notification = useNotification();
  const [subTab, setSubTab] = useState(0);

  const [loadingScope, setLoadingScope] = useState(false);
  const [scopeData, setScopeData] = useState<any>(null);

  const [multiBackend, setMultiBackend] = useState<any>(null);
  const [artifacts, setArtifacts] = useState<any>(null);
  const [loadingArtifacts, setLoadingArtifacts] = useState(false);

  const boId = businessObject?.id || businessObject?.key;

  useEffect(() => {
    if (!boId) return;

    const loadData = async () => {
      setLoadingScope(true);
      try {
        const [scopeResp, multiResp, artResp] = await Promise.all([
          fetchAPI<any>(`/business-objects/${boId}/scope`).catch(() => null),
          fetchAPI<any>(`/business-objects/${boId}/multi-backend`).catch(() => null),
          fetchAPI<any>(`/business-objects/${boId}/artifacts`).catch(() => null),
        ]);
        if (scopeResp) setScopeData(scopeResp);
        if (multiResp) setMultiBackend(multiResp);
        if (artResp) setArtifacts(artResp);
      } catch (err) {
        console.error('Failed to load binding details', err);
      } finally {
        setLoadingScope(false);
      }
    };

    loadData();
  }, [boId]);

  const copyToClipboard = (text: string, label: string) => {
    navigator.clipboard.writeText(text);
    notification.success(`Copied ${label} to clipboard!`);
  };

  return (
    <Box sx={{ p: 3 }}>
      {/* Header Bar */}
      <Stack direction="row" justifyContent="space-between" alignItems="center" sx={{ mb: 3 }}>
        <Box>
          <Stack direction="row" spacing={1.5} alignItems="center">
            <StorageIcon color="primary" />
            <Typography variant="h6" sx={{ fontWeight: 700 }}>
              Polymorphic Storage Bindings & Dynamic Scope
            </Typography>
          </Stack>
          <Typography variant="body2" color="text.secondary" sx={{ mt: 0.5 }}>
            Manage physical database mappings across Tier 1 (Postgres OLTP), Tier 2 (StarRocks OLAP), Tier 3 (Iceberg Deep History), and auto-discovered scope fences.
          </Typography>
        </Box>
        {scopeData && (
          <Chip
            icon={scopeData.isPublishReady ? <ResolvedIcon /> : <UnresolvedIcon />}
            label={scopeData.isPublishReady ? 'Scope Gate: Resolved' : 'Scope Gate: Blocked'}
            color={scopeData.isPublishReady ? 'success' : 'warning'}
            sx={{ fontWeight: 700 }}
          />
        )}
      </Stack>

      <Tabs value={subTab} onChange={(_, val) => setSubTab(val)} sx={{ mb: 3, borderBottom: 1, borderColor: 'divider' }}>
        <Tab label="Multi-Tier Storage Planes" icon={<TierIcon />} iconPosition="start" />
        <Tab label="Dynamic Scope & Auto-Discovery" icon={<GraphIcon />} iconPosition="start" />
        <Tab label="Zero-Code Artifacts (OpenAPI)" icon={<CodeIcon />} iconPosition="start" />
      </Tabs>

      {/* SUBTAB 0: Multi-Tier Storage Planes */}
      {subTab === 0 && (
        <Stack spacing={3}>
          {multiBackend && (
            <Card variant="outlined" sx={{ bgcolor: 'action.hover' }}>
              <CardContent sx={{ p: 2 }}>
                <Grid container spacing={2} alignItems="center">
                  <Grid size={{ xs: 12, md: 8 }}>
                    <Typography variant="subtitle2" sx={{ fontWeight: 700 }}>
                      Hot / Cold Watermark Seam
                    </Typography>
                    <Typography variant="caption" color="text.secondary">
                      Queries automatically route to PostgreSQL / StarRocks for live operational dates, and seam into Apache Iceberg for archival queries ($Date &lt; W_t$).
                    </Typography>
                  </Grid>
                  <Grid size={{ xs: 12, md: 4 }} sx={{ textAlign: { md: 'right' } }}>
                    <Chip
                      label={`Watermark: ${new Date(multiBackend.watermarkDate || Date.now()).toLocaleDateString()}`}
                      color="primary"
                      variant="outlined"
                      sx={{ fontWeight: 700 }}
                    />
                  </Grid>
                </Grid>
              </CardContent>
            </Card>
          )}

          <Grid container spacing={2}>
            {(multiBackend?.bindings || [
              {
                storageTier: 'TIER_1_POSTGRES',
                backendName: 'PostgreSQL (Control Plane / OLTP)',
                physicalTarget: `public.${businessObject?.driverTableName || 'driver_table'}`,
                requirement: 'REQUIRED',
                coveragePercentage: 100,
              },
              {
                storageTier: 'TIER_2_STARROCKS',
                backendName: 'StarRocks (Hot Analytical Data Plane)',
                physicalTarget: `olap.${businessObject?.driverTableName || 'driver_table'}_hot`,
                requirement: 'OPTIONAL',
                coveragePercentage: 90,
              },
              {
                storageTier: 'TIER_3_ICEBERG',
                backendName: 'Apache Iceberg (Cold Historical Archival)',
                physicalTarget: `iceberg.catalog.${businessObject?.driverTableName || 'driver_table'}_historical`,
                requirement: 'OPTIONAL',
                coveragePercentage: 100,
              },
            ]).map((b: any, idx: number) => (
              <Grid size={{ xs: 12, md: 4 }} key={idx}>
                <Card variant="outlined" sx={{ height: '100%' }}>
                  <CardHeader
                    avatar={<CloudIcon color="primary" />}
                    title={<Typography variant="subtitle2" sx={{ fontWeight: 700 }}>{b.backendName}</Typography>}
                    subheader={<Typography variant="caption" sx={{ fontFamily: 'monospace' }}>{b.physicalTarget}</Typography>}
                  />
                  <Divider />
                  <CardContent sx={{ p: 2 }}>
                    <Stack spacing={1}>
                      <Stack direction="row" justifyContent="space-between">
                        <Typography variant="caption" color="text.secondary">Requirement</Typography>
                        <Chip label={b.requirement} size="small" variant="outlined" sx={{ fontSize: '0.65rem', height: 20 }} />
                      </Stack>
                      <Stack direction="row" justifyContent="space-between">
                        <Typography variant="caption" color="text.secondary">Field Coverage</Typography>
                        <Typography variant="caption" sx={{ fontWeight: 700 }}>{b.coveragePercentage}%</Typography>
                      </Stack>
                    </Stack>
                  </CardContent>
                </Card>
              </Grid>
            ))}
          </Grid>
        </Stack>
      )}

      {/* SUBTAB 1: Dynamic Scope & Auto-Discovery */}
      {subTab === 1 && (
        <Stack spacing={3}>
          {scopeData && (
            <Grid container spacing={2}>
              <Grid size={{ xs: 6, sm: 3 }}>
                <Paper variant="outlined" sx={{ p: 1.5, textAlign: 'center' }}>
                  <Typography variant="caption" color="text.secondary" sx={{ fontWeight: 700 }}>DIRECT TERMS</Typography>
                  <Typography variant="h5" sx={{ fontWeight: 800, color: 'primary.main', my: 0.5 }}>{scopeData.directCount}</Typography>
                  <Typography variant="caption" color="text.secondary">Auto-mapped via columns</Typography>
                </Paper>
              </Grid>
              <Grid size={{ xs: 6, sm: 3 }}>
                <Paper variant="outlined" sx={{ p: 1.5, textAlign: 'center' }}>
                  <Typography variant="caption" color="text.secondary" sx={{ fontWeight: 700 }}>RELATED TERMS</Typography>
                  <Typography variant="h5" sx={{ fontWeight: 800, color: 'info.main', my: 0.5 }}>{scopeData.relatedCount}</Typography>
                  <Typography variant="caption" color="text.secondary">Traversed via foreign keys</Typography>
                </Paper>
              </Grid>
              <Grid size={{ xs: 6, sm: 3 }}>
                <Paper variant="outlined" sx={{ p: 1.5, textAlign: 'center' }}>
                  <Typography variant="caption" color="text.secondary" sx={{ fontWeight: 700 }}>CALCULATED TERMS</Typography>
                  <Typography variant="h5" sx={{ fontWeight: 800, color: 'secondary.main', my: 0.5 }}>{scopeData.calculatedCount}</Typography>
                  <Typography variant="caption" color="text.secondary">USES_INPUT dependency tree</Typography>
                </Paper>
              </Grid>
              <Grid size={{ xs: 6, sm: 3 }}>
                <Paper variant="outlined" sx={{ p: 1.5, textAlign: 'center' }}>
                  <Typography variant="caption" color="text.secondary" sx={{ fontWeight: 700 }}>MANUAL TERMS</Typography>
                  <Typography variant="h5" sx={{ fontWeight: 800, my: 0.5 }}>{scopeData.manualCount}</Typography>
                  <Typography variant="caption" color="text.secondary">Explicit user injections</Typography>
                </Paper>
              </Grid>
            </Grid>
          )}

          {/* Scope Fields Table */}
          <TableContainer component={Paper} variant="outlined">
            <Table size="small">
              <TableHead>
                <TableRow sx={{ bgcolor: 'action.hover' }}>
                  <TableCell sx={{ fontWeight: 700 }}>Field Name</TableCell>
                  <TableCell sx={{ fontWeight: 700 }}>Eligibility Level</TableCell>
                  <TableCell sx={{ fontWeight: 700 }}>Resolution Path</TableCell>
                  <TableCell sx={{ fontWeight: 700 }}>Physical Column</TableCell>
                  <TableCell sx={{ fontWeight: 700 }}>Status</TableCell>
                </TableRow>
              </TableHead>
              <TableBody>
                {(scopeData?.eligibleFields || []).map((ef: any, idx: number) => (
                  <TableRow key={idx} hover>
                    <TableCell sx={{ fontWeight: 600 }}>{ef.displayName || ef.fieldName}</TableCell>
                    <TableCell>
                      <Chip
                        label={ef.eligibilityLevel}
                        size="small"
                        color={ef.eligibilityLevel === 'DIRECT' ? 'primary' : ef.eligibilityLevel === 'RELATED' ? 'info' : ef.eligibilityLevel === 'CALCULATED' ? 'secondary' : 'default'}
                        sx={{ fontSize: '0.65rem', height: 20 }}
                      />
                    </TableCell>
                    <TableCell sx={{ fontFamily: 'monospace', fontSize: '0.75rem', color: 'text.secondary' }}>
                      {ef.resolutionPath}
                    </TableCell>
                    <TableCell sx={{ fontFamily: 'monospace', fontSize: '0.75rem' }}>
                      {ef.physicalColumn}
                    </TableCell>
                    <TableCell>
                      <Chip
                        label={ef.resolutionStatus}
                        size="small"
                        color={ef.resolutionStatus === 'RESOLVED' ? 'success' : 'error'}
                        variant="outlined"
                        sx={{ fontSize: '0.65rem', height: 20 }}
                      />
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </TableContainer>
        </Stack>
      )}

      {/* SUBTAB 2: Zero-Code Generated Artifacts */}
      {subTab === 2 && (
        <Stack spacing={3}>
          {artifacts && (
            <>
              {/* OpenAPI 3.0 */}
              <Card variant="outlined">
                <CardHeader
                  title={<Typography variant="subtitle2" sx={{ fontWeight: 700 }}>OpenAPI 3.0 REST Specification</Typography>}
                  subheader={`Endpoint: ${artifacts.restEndpointUrl}`}
                  action={
                    <Button
                      size="small"
                      startIcon={<CopyIcon />}
                      onClick={() => copyToClipboard(artifacts.openApiSpecJson, 'OpenAPI Spec')}
                    >
                      Copy JSON
                    </Button>
                  }
                />
                <Divider />
                <CardContent sx={{ p: 2, bgcolor: 'action.hover' }}>
                  <Typography variant="body2" component="pre" sx={{ fontFamily: 'monospace', fontSize: '0.75rem', m: 0, overflowX: 'auto' }}>
                    {artifacts.openApiSpecJson}
                  </Typography>
                </CardContent>
              </Card>

              {/* StarRocks Materialized View */}
              <Card variant="outlined">
                <CardHeader
                  title={<Typography variant="subtitle2" sx={{ fontWeight: 700 }}>StarRocks Materialized View DDL</Typography>}
                  action={
                    <Button
                      size="small"
                      startIcon={<CopyIcon />}
                      onClick={() => copyToClipboard(artifacts.starRocksMvDdl, 'StarRocks DDL')}
                    >
                      Copy SQL
                    </Button>
                  }
                />
                <Divider />
                <CardContent sx={{ p: 2, bgcolor: 'action.hover' }}>
                  <Typography variant="body2" component="pre" sx={{ fontFamily: 'monospace', fontSize: '0.75rem', m: 0, overflowX: 'auto' }}>
                    {artifacts.starRocksMvDdl}
                  </Typography>
                </CardContent>
              </Card>
            </>
          )}
        </Stack>
      )}
    </Box>
  );
}
