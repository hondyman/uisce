import React, { useCallback, useEffect, useMemo, useState } from 'react';
import {
  Alert,
  Box,
  Button,
  Chip,
  CircularProgress,
  Divider,
  Stack,
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableRow,
  Typography,
} from '@mui/material';
import { getCubeImpact, previewCubeImpact } from './cubeDefinitionApi';
import type {
  CubeDraft,
  CubeImpactConsumer,
  CubeImpactPreviewAction,
  CubeImpactPreviewReport,
  CubeImpactReport,
} from './types';
import { cubeDraftPayload } from './draft';

export type ImpactPanelProps = {
  cubeId: string;
  /** When set, patch/publish_version previews use this draft surface. */
  draft?: CubeDraft | null;
  readOnly?: boolean;
  /** Optional title override (catalog drawer vs designer tab). */
  title?: string;
};

function severityColor(sev: string): 'default' | 'warning' | 'error' | 'info' | 'success' {
  if (sev === 'blocking') return 'error';
  if (sev === 'warning') return 'warning';
  if (sev === 'info') return 'info';
  return 'default';
}

function ConsumerTable({ consumers }: { consumers: CubeImpactConsumer[] }) {
  if (!consumers.length) {
    return (
      <Alert severity="success" sx={{ mt: 1 }}>
        No consumers found for this cube.
      </Alert>
    );
  }
  return (
    <Table size="small" sx={{ mt: 1 }}>
      <TableHead>
        <TableRow>
          <TableCell>Kind</TableCell>
          <TableCell>Label</TableCell>
          <TableCell>Severity</TableCell>
          <TableCell>Detail</TableCell>
        </TableRow>
      </TableHead>
      <TableBody>
        {consumers.map((c) => (
          <TableRow key={`${c.kind}:${c.id}`}>
            <TableCell>{c.kind}</TableCell>
            <TableCell>
              {c.href ? (
                <Typography
                  component="a"
                  href={c.href}
                  variant="body2"
                  sx={{ color: 'primary.main', textDecoration: 'none' }}
                >
                  {c.label}
                </Typography>
              ) : (
                c.label
              )}
            </TableCell>
            <TableCell>
              <Chip
                size="small"
                label={c.blocking ? 'blocking' : c.severity}
                color={severityColor(c.blocking ? 'blocking' : c.severity)}
                variant="outlined"
              />
            </TableCell>
            <TableCell>
              <Typography variant="caption" color="text.secondary">
                {c.detail || '—'}
              </Typography>
            </TableCell>
          </TableRow>
        ))}
      </TableBody>
    </Table>
  );
}

const ImpactPanel: React.FC<ImpactPanelProps> = ({
  cubeId,
  draft = null,
  readOnly = false,
  title = 'Cube impact',
}) => {
  const id = (cubeId || '').trim();
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [report, setReport] = useState<CubeImpactReport | null>(null);
  const [preview, setPreview] = useState<CubeImpactPreviewReport | null>(null);
  const [previewBusy, setPreviewBusy] = useState(false);

  const load = useCallback(async () => {
    if (!id) {
      setReport(null);
      setPreview(null);
      setError(null);
      return;
    }
    setLoading(true);
    setError(null);
    setPreview(null);
    try {
      const r = await getCubeImpact(id);
      setReport(r);
    } catch (e) {
      setReport(null);
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      setLoading(false);
    }
  }, [id]);

  useEffect(() => {
    void load();
  }, [load]);

  const runPreview = async (action: CubeImpactPreviewAction) => {
    if (!id || readOnly) return;
    setPreviewBusy(true);
    setError(null);
    try {
      const body: { action: CubeImpactPreviewAction; patch?: Record<string, unknown> } = { action };
      if ((action === 'patch' || action === 'publish_version') && draft) {
        body.patch = cubeDraftPayload(draft);
      }
      const r = await previewCubeImpact(id, body);
      setPreview(r);
    } catch (e) {
      setPreview(null);
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      setPreviewBusy(false);
    }
  };

  const composition = report?.composition;
  const summary = preview?.summary ?? report?.summary;
  const consumers = preview?.consumers ?? report?.consumers ?? [];

  const fedSummary = useMemo(() => {
    const fed = composition?.federation;
    const sources = fed?.sources?.length ?? 0;
    const joins = fed?.joins?.length ?? 0;
    if (!sources && !joins) return 'single-BO (no federation)';
    return `${sources} source(s), ${joins} join(s)`;
  }, [composition]);

  if (!id) {
    return <Alert severity="info">Select a cube to assess impact.</Alert>;
  }

  return (
    <Box sx={{ display: 'flex', flexDirection: 'column', gap: 1.5 }}>
      <Stack direction="row" alignItems="center" justifyContent="space-between" flexWrap="wrap" gap={1}>
        <Typography variant="subtitle1" fontWeight={600}>
          {title}
        </Typography>
        <Button size="small" onClick={() => void load()} disabled={loading}>
          Refresh inventory
        </Button>
      </Stack>

      {loading && (
        <Stack direction="row" alignItems="center" gap={1}>
          <CircularProgress size={18} />
          <Typography variant="body2">Loading impact…</Typography>
        </Stack>
      )}
      {error && (
        <Alert severity="error" onClose={() => setError(null)}>
          {error}
        </Alert>
      )}

      {composition && (
        <Box>
          <Typography variant="subtitle2" gutterBottom>
            Composition
          </Typography>
          <Stack direction="row" flexWrap="wrap" gap={0.75}>
            <Chip size="small" label={composition.name} />
            <Chip size="small" variant="outlined" label={`v${composition.contractVersion}`} />
            <Chip size="small" variant="outlined" label={composition.status} />
            <Chip size="small" variant="outlined" label={`BO ${composition.boId}`} />
            <Chip size="small" variant="outlined" label={`${composition.dimensions?.length ?? 0} dims`} />
            <Chip size="small" variant="outlined" label={`${composition.metrics?.length ?? 0} metrics`} />
            <Chip size="small" variant="outlined" label={`${composition.grains?.length ?? 0} grains`} />
            <Chip size="small" variant="outlined" label={fedSummary} />
            <Chip
              size="small"
              variant="outlined"
              label={`${composition.physical?.grains?.length ?? 0} physical`}
            />
          </Stack>
          {!!composition.metrics?.length && (
            <Typography variant="caption" color="text.secondary" display="block" sx={{ mt: 0.5 }}>
              Metrics:{' '}
              {composition.metrics.map((m) => m.name || m.id).join(', ')}
            </Typography>
          )}
          {!!composition.physical?.grains?.length && (
            <Typography variant="caption" color="text.secondary" display="block">
              Physical:{' '}
              {composition.physical.grains
                .map((g) => `${g.lifecycleStatus || '?'}:${g.grainHash || g.nodeName || 'grain'}`)
                .join(' · ')}
            </Typography>
          )}
        </Box>
      )}

      {summary && (
        <Stack direction="row" flexWrap="wrap" gap={0.75}>
          <Chip size="small" label={`${summary.consumerCount} consumers`} />
          <Chip
            size="small"
            color={summary.blockingCount > 0 ? 'error' : 'default'}
            label={`${summary.blockingCount} blocking`}
          />
          <Chip size="small" color="warning" variant="outlined" label={`${summary.warningCount} warnings`} />
          <Chip size="small" variant="outlined" label={`${summary.physicalGrainCount} physical grains`} />
        </Stack>
      )}

      <Divider />

      <Typography variant="subtitle2">Consumers</Typography>
      <ConsumerTable consumers={consumers} />

      {!readOnly && (
        <>
          <Divider />
          <Typography variant="subtitle2">Dry-run preview (cascade Confirm lands in A4)</Typography>
          <Stack direction="row" flexWrap="wrap" gap={1}>
            <Button
              size="small"
              variant="outlined"
              disabled={previewBusy}
              onClick={() => void runPreview('archive')}
            >
              Preview archive
            </Button>
            <Button
              size="small"
              variant="outlined"
              disabled={previewBusy || !draft}
              onClick={() => void runPreview('patch')}
            >
              Preview patch
            </Button>
            <Button
              size="small"
              variant="outlined"
              color="warning"
              disabled={previewBusy || !draft}
              onClick={() => void runPreview('publish_version')}
            >
              Preview publish version
            </Button>
          </Stack>
          {!draft && (
            <Typography variant="caption" color="text.secondary">
              Patch / publish previews need a draft binding (designer). Archive works from inventory alone.
            </Typography>
          )}
        </>
      )}

      {preview && (
        <Alert
          severity={
            preview.changeClass === 'breaking_contract'
              ? 'warning'
              : preview.blockingCount > 0
                ? 'warning'
                : 'info'
          }
        >
          <Typography variant="body2" fontWeight={600}>
            {preview.changeClass}
            {preview.nextVersion ? ` → next v${preview.nextVersion}` : ''}
          </Typography>
          {!!preview.breakReasons?.length && (
            <Typography variant="body2">Break reasons: {preview.breakReasons.join(', ')}</Typography>
          )}
          <Typography variant="body2">
            Blocking: {preview.blockingCount} · Recommended mode: {preview.recommendedMode}
          </Typography>
          <Typography variant="body2">Allowed: {(preview.allowedModes || []).join(', ')}</Typography>
          <Typography variant="caption" display="block" sx={{ mt: 0.5, wordBreak: 'break-all' }}>
            confirmToken minted ({preview.confirmToken?.length || 0} chars) — cascade apply deferred to A4.
          </Typography>
        </Alert>
      )}
    </Box>
  );
};

export default ImpactPanel;
