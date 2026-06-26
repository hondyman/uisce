import { useEffect, useMemo, useState } from 'react';
import { useSearchParams } from 'react-router-dom';
import BlockableLink from '../../../components/RouteBlocker/BlockableLink';
import {
  Alert,
  Box,
  Breadcrumbs,
  // Button not used
  Card,
  CardContent,
  Chip,
  CircularProgress,
  Divider,
  Grid,
  Link as MUILink,
  Stack,
  Tab,
  Tabs,
  Typography,
  List,
  ListItem,
  ListItemText,
} from '@mui/material';
import { fetchDiff } from '../api';
import BrokenRefsFixer from '../components/BrokenRefsFixer';
import PreviewRunner from '../components/PreviewRunner';
import getErrorMessage from '../../../utils/errors';

type DiffReport = {
  from: string;
  to: string;
  cubes: {
    added: string[];
    removed: string[];
    changed: Record<string, string>;
    pk_fk_diffs: string[];
    join_path_diffs: string[];
  };
  views: {
    join_path_changes: string[];
    include_impacts: string[];
    folder_diffs: string[];
  };
  governance: {
    pii_changes: string[];
    access_policy_diffs: string[];
    tenant_isolation_checks: string[];
  };
  pre_aggs: {
    rebuild_plan: string[];
    estimated_cost: string;
    estimated_time: string;
  };
};

function KeyValueList({ data }: { data: Record<string, string> }) {
  const entries = Object.entries(data);
  if (!entries.length) return <Typography variant="body2" color="text.secondary">No changes</Typography>;
  return (
    <List dense>
      {entries.map(([k, v]) => (
        <ListItem key={k} disableGutters>
          <ListItemText primaryTypographyProps={{ sx: { fontFamily: 'monospace' } }} primary={`${k}: ${v}`} />
        </ListItem>
      ))}
    </List>
  );
}

function SimpleList({ items }: { items: string[] }) {
  if (!items?.length) return <Typography variant="body2" color="text.secondary">No changes</Typography>;
  return (
    <List dense>
      {items.map((it, i) => (
        <ListItem key={`${it}-${i}`} disableGutters>
          <ListItemText primaryTypographyProps={{ sx: { fontFamily: 'monospace' } }} primary={it} />
        </ListItem>
      ))}
    </List>
  );
}

export default function DiffViewerPage() {
  const [search] = useSearchParams();
  const from = search.get('from') || 'active';
  const to = search.get('to') || '';
  const [tab, setTab] = useState(0);
  const [diff, setDiff] = useState<DiffReport | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (!to) return;
    setLoading(true);
    setError(null);
    (async () => {
      try {
        const data = await fetchDiff(from, to);
        setDiff(data as DiffReport);
      } catch (e: unknown) {
        setError(getErrorMessage(e, 'Failed to load diff'));
      } finally {
        setLoading(false);
      }
    })();
  }, [from, to]);

  const counts = useMemo(() => {
    if (!diff) return { added: 0, removed: 0, changed: 0 };
    return {
      added: diff.cubes.added.length,
      removed: diff.cubes.removed.length,
      changed: Object.keys(diff.cubes.changed || {}).length,
    };
  }, [diff]);

  return (
    <Box sx={{ p: 3 }}>
      <Breadcrumbs sx={{ mb: 2 }}>
        <MUILink component={BlockableLink} to="/">Home</MUILink>
        <MUILink component={BlockableLink} to="/upgrade">Upgrade Center</MUILink>
        <Typography>Diff</Typography>
      </Breadcrumbs>
      <Stack direction="row" alignItems="center" spacing={1} sx={{ mb: 2 }}>
        <Typography variant="h5">Upgrade Diff</Typography>
        <Chip label={`${from} → ${to}`} size="small" />
      </Stack>

      {loading && <CircularProgress />}
      {error && <Alert severity="error">{error}</Alert>}
      {diff && (
        <Grid container spacing={2}>
          <Grid item xs={12} md={8}>
            <Card variant="outlined" sx={{ mb: 2 }}>
              <CardContent>
                <Stack direction="row" spacing={1} sx={{ mb: 1 }}>
                  <Chip color="success" label={`Cubes added: ${counts.added}`} size="small" />
                  <Chip color="warning" label={`Cubes changed: ${counts.changed}`} size="small" />
                  <Chip color="error" label={`Cubes removed: ${counts.removed}`} size="small" />
                </Stack>
                <Tabs value={tab} onChange={(_, v) => setTab(v)}>
                  <Tab label="Cubes" />
                  <Tab label="Views" />
                  <Tab label="Governance" />
                  <Tab label="Pre‑aggs" />
                </Tabs>
                <Divider sx={{ mb: 2 }} />
                {tab === 0 && (
                  <Grid container spacing={2}>
                    <Grid item xs={12} md={6}>
                      <Typography variant="subtitle2">Added</Typography>
                      <SimpleList items={diff.cubes.added} />
                    </Grid>
                    <Grid item xs={12} md={6}>
                      <Typography variant="subtitle2">Removed</Typography>
                      <SimpleList items={diff.cubes.removed} />
                    </Grid>
                    <Grid item xs={12}>
                      <Typography variant="subtitle2">Changed</Typography>
                      <KeyValueList data={diff.cubes.changed} />
                    </Grid>
                    <Grid item xs={12} md={6}>
                      <Typography variant="subtitle2">PK/FK diffs</Typography>
                      <SimpleList items={diff.cubes.pk_fk_diffs} />
                    </Grid>
                    <Grid item xs={12} md={6}>
                      <Typography variant="subtitle2">Join path diffs</Typography>
                      <SimpleList items={diff.cubes.join_path_diffs} />
                    </Grid>
                  </Grid>
                )}
                {tab === 1 && (
                  <Grid container spacing={2}>
                    <Grid item xs={12}>
                      <Typography variant="subtitle2">Join path changes</Typography>
                      <SimpleList items={diff.views.join_path_changes} />
                    </Grid>
                    <Grid item xs={12}>
                      <Typography variant="subtitle2">Include impacts</Typography>
                      <SimpleList items={diff.views.include_impacts} />
                    </Grid>
                    <Grid item xs={12}>
                      <Typography variant="subtitle2">Folder diffs</Typography>
                      <SimpleList items={diff.views.folder_diffs} />
                    </Grid>
                  </Grid>
                )}
                {tab === 2 && (
                  <Grid container spacing={2}>
                    <Grid item xs={12} md={6}>
                      <Typography variant="subtitle2">PII changes</Typography>
                      <SimpleList items={diff.governance.pii_changes} />
                    </Grid>
                    <Grid item xs={12} md={6}>
                      <Typography variant="subtitle2">Access policy diffs</Typography>
                      <SimpleList items={diff.governance.access_policy_diffs} />
                    </Grid>
                    <Grid item xs={12}>
                      <Typography variant="subtitle2">Tenant isolation checks</Typography>
                      <SimpleList items={diff.governance.tenant_isolation_checks} />
                    </Grid>
                  </Grid>
                )}
                {tab === 3 && (
                  <Grid container spacing={2}>
                    <Grid item xs={12}>
                      <Typography variant="subtitle2">Rebuild plan</Typography>
                      <SimpleList items={diff.pre_aggs.rebuild_plan} />
                    </Grid>
                    <Grid item xs={12}>
                      <Typography variant="subtitle2">Estimates</Typography>
                      <Stack direction="row" spacing={2}>
                        <Chip label={`Cost: ${diff.pre_aggs.estimated_cost}`} size="small" />
                        <Chip label={`Time: ${diff.pre_aggs.estimated_time}`} size="small" />
                      </Stack>
                    </Grid>
                  </Grid>
                )}
              </CardContent>
            </Card>
          </Grid>
          <Grid item xs={12} md={4}>
            <Card variant="outlined" sx={{ mb: 2 }}>
              <CardContent>
                <Typography variant="h6" sx={{ mb: 1 }}>Broken refs & fixes</Typography>
                <BrokenRefsFixer version={diff.to} />
              </CardContent>
            </Card>
            <Card variant="outlined">
              <CardContent>
                <Typography variant="h6" sx={{ mb: 1 }}>Preview runner</Typography>
                <PreviewRunner fromVersion={diff.from} toVersion={diff.to} />
              </CardContent>
            </Card>
          </Grid>
        </Grid>
      )}

      {!to && (
        <Alert severity="info">No target version selected. Open from Versions page.</Alert>
      )}
    </Box>
  );
}
