import React, { useEffect, useState } from 'react';
import {
  Alert,
  Box,
  Button,
  Chip,
  CircularProgress,
  Paper,
  Stack,
  Tab,
  Tabs,
  Typography,
} from '@mui/material';
import ArrowBackIcon from '@mui/icons-material/ArrowBack';
import SpeedIcon from '@mui/icons-material/Speed';
import { Link as RouterLink, useParams } from 'react-router-dom';
import { useLocale } from '../../../i18n/useLocale';
import { getCube } from '../cubeDefinitionApi';
import type { CubeDefinition } from '../types';

type TabKey = 'overview' | 'dimensions' | 'metrics' | 'grains' | 'federation' | 'materialization' | 'versions';

const CubeDesignerPage: React.FC = () => {
  const locale = useLocale();
  const { cubeId } = useParams<{ cubeId: string }>();
  const isNew = cubeId === 'new' || !cubeId;

  const [tab, setTab] = useState<TabKey>('overview');
  const [cube, setCube] = useState<CubeDefinition | null>(null);
  const [loading, setLoading] = useState(!isNew);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (isNew || !cubeId) {
      setLoading(false);
      setCube(null);
      return;
    }
    let cancelled = false;
    setLoading(true);
    setError(null);
    getCube(cubeId)
      .then((c) => {
        if (!cancelled) setCube(c);
      })
      .catch((e) => {
        if (!cancelled) setError(e instanceof Error ? e.message : String(e));
      })
      .finally(() => {
        if (!cancelled) setLoading(false);
      });
    return () => {
      cancelled = true;
    };
  }, [cubeId, isNew]);

  return (
    <Box sx={{ p: 3, maxWidth: 1100, mx: 'auto' }}>
      <Stack direction="row" spacing={1} mb={2}>
        <Button
          component={RouterLink}
          to={`/${locale}/build/cubes`}
          startIcon={<ArrowBackIcon />}
          size="small"
        >
          Cubes
        </Button>
        <Button
          component={RouterLink}
          to={`/${locale}/fabric/preaggregations`}
          startIcon={<SpeedIcon />}
          size="small"
          variant="outlined"
        >
          Preaggregations ops
        </Button>
      </Stack>

      <Stack direction={{ xs: 'column', sm: 'row' }} justifyContent="space-between" spacing={1} mb={2}>
        <Box>
          <Typography variant="h4" component="h1">
            {isNew ? 'New Cube' : cube?.name || 'Cube Designer'}
          </Typography>
          {!isNew && cube && (
            <Stack direction="row" spacing={1} mt={1} flexWrap="wrap" useFlexGap>
              <Chip label={`v${cube.contractVersion}`} size="small" />
              <Chip label={cube.status} size="small" color={cube.status === 'active' ? 'success' : 'default'} />
              {cube.isCore && <Chip label="Core" size="small" color="secondary" />}
              <Chip label={cube.boId} size="small" variant="outlined" />
            </Stack>
          )}
        </Box>
      </Stack>

      {error && (
        <Alert severity="error" sx={{ mb: 2 }}>
          {error}
        </Alert>
      )}

      {loading ? (
        <Box sx={{ display: 'flex', justifyContent: 'center', py: 6 }}>
          <CircularProgress size={32} />
        </Box>
      ) : (
        <Paper variant="outlined" sx={{ p: 2 }}>
          <Tabs
            value={tab}
            onChange={(_, v) => setTab(v as TabKey)}
            variant="scrollable"
            scrollButtons="auto"
            sx={{ borderBottom: 1, borderColor: 'divider', mb: 2 }}
          >
            <Tab label="Overview" value="overview" />
            <Tab label="Dimensions" value="dimensions" />
            <Tab label="Metrics" value="metrics" />
            <Tab label="Grains" value="grains" />
            <Tab label="Federation" value="federation" />
            <Tab label="Materialization" value="materialization" />
            <Tab label="Versions" value="versions" />
          </Tabs>

          {tab === 'overview' && (
            <Box>
              {isNew ? (
                <Alert severity="info">
                  Cube Designer authoring (pick BO, dimensions, metrics, grains) lands in CUBE-0.4.
                  This shell establishes the route and layout.
                </Alert>
              ) : cube ? (
                <Stack spacing={1}>
                  <Typography variant="body2" color="text.secondary">
                    {cube.description || 'No description'}
                  </Typography>
                  <Typography variant="body2">
                    <strong>Metrics:</strong> {cube.metricIds?.length ?? 0} ·{' '}
                    <strong>Dimensions:</strong> {cube.dimensions?.length ?? 0} ·{' '}
                    <strong>Grains:</strong> {cube.grains?.length ?? 0}
                  </Typography>
                  <Typography variant="caption" color="text.secondary" sx={{ fontFamily: 'monospace' }}>
                    contentHash {cube.contentHash || '—'}
                  </Typography>
                </Stack>
              ) : null}
            </Box>
          )}

          {tab === 'dimensions' && (
            <ShellPlaceholder
              title="Dimensions"
              detail={
                isNew
                  ? 'Ordered dimension surface (term nodes) will be editable in CUBE-0.4.'
                  : `Stored axes: ${(cube?.dimensions || []).map((d) => d.termNodeId).join(', ') || 'none'}`
              }
            />
          )}
          {tab === 'metrics' && (
            <ShellPlaceholder
              title="Metrics"
              detail={
                isNew
                  ? 'Governed metric picker lands in CUBE-0.4.'
                  : `metricIds: ${(cube?.metricIds || []).join(', ') || 'none'}`
              }
            />
          )}
          {tab === 'grains' && (
            <ShellPlaceholder
              title="Grains"
              detail={
                isNew
                  ? 'Materialization grain editor lands in CUBE-0.4.'
                  : `Declared grains: ${cube?.grains?.length ?? 0}`
              }
            />
          )}
          {tab === 'federation' && (
            <ShellPlaceholder
              title="Federation"
              detail="Multi-BO join plan UI lands in CUBE-2.3. Empty federation means single-BO."
            />
          )}
          {tab === 'materialization' && (
            <ShellPlaceholder
              title="Materialization"
              detail={
                cube
                  ? `strategy=${cube.materialization?.strategy || '—'} hot=${cube.materialization?.hotEngine || 'starrocks'} cold=${cube.materialization?.coldEngine || 'iceberg'}`
                  : 'Defaults: StarRocks hot + Iceberg cold (CUBE-0.4).'
              }
            />
          )}
          {tab === 'versions' && (
            <ShellPlaceholder
              title="Contract versions"
              detail={
                cube
                  ? `Current contract_version=${cube.contractVersion}. Breaking publishes use POST /api/cubes/{id}/versions.`
                  : 'Version history appears after the first save.'
              }
            />
          )}
        </Paper>
      )}
    </Box>
  );
};

function ShellPlaceholder({ title, detail }: { title: string; detail: string }) {
  return (
    <Box>
      <Typography variant="h6" gutterBottom>
        {title}
      </Typography>
      <Typography variant="body2" color="text.secondary">
        {detail}
      </Typography>
    </Box>
  );
}

export default CubeDesignerPage;
