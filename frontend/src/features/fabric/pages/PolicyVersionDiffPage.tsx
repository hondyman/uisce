import React, { useState, Suspense } from 'react';
import { useParams, Link as RouterLink } from 'react-router-dom';
import { useQuery, useMutation } from '@tanstack/react-query';
import {
  Box,
  Typography,
  Paper,
  Grid,
  List,
  ListItemButton,
  ListItemText,
  CircularProgress,
  Alert,
  Breadcrumbs,
  Link,
  Divider,
  Button,
  FormControlLabel,
  Switch,
  ToggleButtonGroup,
  ToggleButton,
} from '@mui/material';
import { LocalizationProvider } from '@mui/x-date-pickers/LocalizationProvider';
import { AdapterDateFns } from '@mui/x-date-pickers/AdapterDateFns';
import { DatePicker } from '@mui/x-date-pickers/DatePicker';
import { createPatch } from 'diff';
import { parseDiff, Diff, Hunk } from 'react-diff-view';
import 'react-diff-view/style/index.css';
import { format, subMonths } from 'date-fns';
import yaml from 'js-yaml';
// Lazy-load heavier visualization/table components to keep this page's initial bundle small
const ImpactSummary = React.lazy(() => import('../components/ImpactSummary'));
const ChangeHeatmap = React.lazy(() => import('../components/ChangeHeatmap'));
const DifferenceOnlyTimeline = React.lazy(() => import('../components/DifferenceOnlyTimeline'));
const DecisionReplayChart = React.lazy(() => import('../components/DecisionReplayChart'));
const DecisionDiffTable = React.lazy(() => import('../components/DecisionDiffTable'));
import { useDrillDown } from '../../../contexts/DrillDownContext';
import { apiFetch } from '../../../lib/apiClient';

interface PolicyVersion {
  id: string;
  version: number;
  author: string;
  created_at: string;
  change_summary?: string | null;
}

const PolicyVersionList: React.FC<{
  policyId: string;
  title: string;
  selectedVersionId: string | null;
  onSelectVersion: (id: string) => void;
}> = ({ policyId, title, selectedVersionId, onSelectVersion }) => {
  const { data, isLoading, error } = useQuery({
    queryKey: ['policy-versions', policyId],
    queryFn: async () => {
      const res = await apiFetch(`/api/rest/policy-versions?policy_id=${encodeURIComponent(policyId)}&order_by=version desc`);
      if (!res.ok) throw new Error(await res.text());
      return res.json();
    },
  });

  return (
    <Box>
      <Typography variant="h6" gutterBottom>
        {title}
      </Typography>
      {isLoading && <CircularProgress size={24} />}
      {error && <Alert severity="error">Failed to load versions.</Alert>}
      <Paper variant="outlined">
        <List dense>
          {(data || []).map((v: PolicyVersion) => (
            <ListItemButton key={v.id} selected={selectedVersionId === v.id} onClick={() => onSelectVersion(v.id)}>
              <ListItemText
                primary={`v${v.version} - ${v.change_summary || 'Initial Version'}`}
                secondary={`${format(new Date(v.created_at), 'yyyy-MM-dd')} by ${
                  v.author || 'system'
                }`}
              />
            </ListItemButton>
          ))}
        </List>
      </Paper>
    </Box>
  );
};

const PolicyDiffView: React.FC<{ versionAId: string; versionBId: string }> = ({ versionAId, versionBId }) => {
  const { data, isLoading, error } = useQuery({
    queryKey: ['policy-version-specs', versionAId, versionBId],
    queryFn: async () => {
      const [resA, resB] = await Promise.all([
        apiFetch(`/api/rest/policy-versions/${versionAId}`),
        apiFetch(`/api/rest/policy-versions/${versionBId}`),
      ]);
      if (!resA.ok) throw new Error(await resA.text());
      if (!resB.ok) throw new Error(await resB.text());
      return Promise.all([resA.json(), resB.json()]);
    },
    enabled: !!versionAId && !!versionBId,
  });

  if (isLoading) return <CircularProgress />;
  if (error) return <Alert severity="error">Failed to load policy specs for diffing.</Alert>;
  if (!data || data.length < 2) {
    return <Alert severity="info">Select two versions to compare.</Alert>;
  }

  const versionA = data[0];
  const versionB = data[1];

  if (!versionA || !versionB) {
    return <Alert severity="warning">Could not find one of the selected versions.</Alert>;
  }

  // Ensure A is the older version for the diff viewer
  const [oldVersion, newVersion] = versionA.version < versionB.version ? [versionA, versionB] : [versionB, versionA];

  const oldSpec = yaml.dump(oldVersion.spec);
  const newSpec = yaml.dump(newVersion.spec);

  // Generate a diff patch to be rendered by the new viewer
  const patch = createPatch(`v${oldVersion.version} vs v${newVersion.version}`, oldSpec, newSpec);
  const [file] = parseDiff(patch, { nearbySequences: 'zip' });

  return (
    <Box sx={{ '& .diff-line-old': { backgroundColor: 'rgba(255, 0, 0, 0.1)' }, '& .diff-line-new': { backgroundColor: 'rgba(0, 255, 0, 0.1)' }, '& pre': { m: 0, whiteSpace: 'pre-wrap', wordBreak: 'break-all' } }}>
      <Box sx={{ display: 'flex', justifyContent: 'space-around', mb: 1, p: 1, backgroundColor: 'action.hover' }}>
        <Typography variant="subtitle2">Version {oldVersion.version}</Typography>
        <Typography variant="subtitle2">Version {newVersion.version}</Typography>
      </Box>
      <Diff viewType="split" diffType={file.type} hunks={file.hunks || []}>
        {(hunks: any[]) => hunks.map(hunk => <Hunk key={hunk.content} hunk={hunk} />)}
      </Diff>
    </Box>
  );
};

const PolicyVersionDiffPage: React.FC = () => {
  const { policyId } = useParams<{ policyId: string }>();
  const [versionA, setVersionA] = useState<string | null>(null);
  const [versionB, setVersionB] = useState<string | null>(null);
  const [fromDate, setFromDate] = useState<Date | null>(subMonths(new Date(), 1));
  const [toDate, setToDate] = useState<Date | null>(new Date());
  const [showDiffOnly, setShowDiffOnly] = useState(false);
  const { showDrillDown } = useDrillDown();
  const [viewMode, setViewMode] = useState<'timeline' | 'heatmap'>('timeline');

  const { data: versionsData } = useQuery({
    queryKey: ['policy-versions', policyId],
    queryFn: async () => {
      const res = await apiFetch(`/api/rest/policy-versions?policy_id=${encodeURIComponent(policyId || '')}&order_by=version desc`);
      if (!res.ok) throw new Error(await res.text());
      return res.json();
    },
    enabled: !!policyId,
  });

  const { data: compareData, isPending: compareLoading, error: compareError, mutate: runCompare } = useMutation({
    mutationFn: async ({ policyId, versionA, versionB, fromDate, toDate, differencesOnly }: {
      policyId: string; versionA: number; versionB: number; fromDate: string; toDate: string; differencesOnly: boolean
    }) => {
      const res = await apiFetch('/api/rest/compare-policy-versions', {
        method: 'POST',
        body: JSON.stringify({
          policy_id: policyId,
          version_a: versionA,
          version_b: versionB,
          from_date: fromDate,
          to_date: toDate,
          differences_only: differencesOnly,
        }),
      });
      if (!res.ok) throw new Error(await res.text());
      return res.json();
    },
  });

  const handleCompare = () => {
    if (!policyId || !versionA || !versionB || !fromDate || !toDate || !versionsData) {
      return;
    }

    const findVersion = (id: string) => versionsData.find((v: PolicyVersion) => v.id === id)?.version;
    const versionANum = findVersion(versionA);
    const versionBNum = findVersion(versionB);

    if (versionANum !== undefined && versionBNum !== undefined) {
      runCompare({
        policyId,
        versionA: versionANum,
        versionB: versionBNum,
        fromDate: format(fromDate, 'yyyy-MM-dd'),
        toDate: format(toDate, 'yyyy-MM-dd'),
        differencesOnly: showDiffOnly,
      });
    }
  };

  if (!policyId) {
    return <Alert severity="error">No Policy ID provided.</Alert>;
  }

  return (
    <Box sx={{ p: 3, height: '100%', display: 'flex', flexDirection: 'column' }}>
      <Breadcrumbs aria-label="breadcrumb" sx={{ mb: 2 }}>
        <Link component={RouterLink} underline="hover" color="inherit" to="/fabric/policies">
          Policy Management
        </Link>
        <Typography color="text.primary">Version History: {policyId}</Typography>
      </Breadcrumbs>

      <Grid container spacing={3} sx={{ flexGrow: 1 }}>
        <Grid size={{ 'xs': 12, 'md': 3 }}>
          <Box sx={{ display: 'flex', flexDirection: 'column', gap: 3 }}>
            <PolicyVersionList
              policyId={policyId}
              title="Compare Version (A)"
              selectedVersionId={versionA}
              onSelectVersion={setVersionA}
            />
            <PolicyVersionList
              policyId={policyId}
              title="With Version (B)"
              selectedVersionId={versionB}
              onSelectVersion={setVersionB}
            />
            <Box sx={{ mt: 2 }}>
              <Typography variant="h6" gutterBottom>
                Date Range
              </Typography>
              <LocalizationProvider dateAdapter={AdapterDateFns}>
                <Box sx={{ display: 'flex', flexDirection: 'column', gap: 2 }}>
                  <DatePicker label="From Date" value={fromDate} onChange={(v) => setFromDate(v instanceof Date ? v : (v?.toDate ? v.toDate() : null))} />
                  <DatePicker label="To Date" value={toDate} onChange={(v) => setToDate(v instanceof Date ? v : (v?.toDate ? v.toDate() : null))} />
                </Box>
              </LocalizationProvider>
              <FormControlLabel
                control={<Switch checked={showDiffOnly} onChange={(e) => setShowDiffOnly(e.target.checked)} />}
                label="Show Differences Only"
                sx={{ mt: 2 }}
              />
              <ToggleButtonGroup
                value={viewMode}
                exclusive
                onChange={(_e, newMode) => newMode && setViewMode(newMode)}
                aria-label="view mode"
                size="small"
                sx={{ mt: 2 }}
              >
                <ToggleButton value="timeline" aria-label="timeline view">Timeline</ToggleButton>
                <ToggleButton value="heatmap" aria-label="heatmap view">Heatmap</ToggleButton>
              </ToggleButtonGroup>
              <Button onClick={handleCompare} disabled={!versionA || !versionB || compareLoading} sx={{ mt: 2 }} variant="contained">
                {compareLoading ? <CircularProgress size={24} /> : 'Compare Impact'}
              </Button>
            </Box>
          </Box>
        </Grid>
        <Grid size={{ 'xs': 12, 'md': 9 }}>
          <Paper sx={{ flexGrow: 1, p: 2, overflow: 'hidden', display: 'flex', flexDirection: 'column' }}>
            <Typography variant="h6" gutterBottom>
              Side-by-Side Comparison
            </Typography>
            <Divider sx={{ mb: 2 }} />
            <Box sx={{ flexGrow: 1, overflow: 'auto', '& .react-diff-viewer': { fontFamily: 'monospace' } }}>
              {versionA && versionB ? (
                <PolicyDiffView versionAId={versionA} versionBId={versionB} />
              ) : (
                <Box
                  sx={{
                    display: 'flex',
                    alignItems: 'center',
                    justifyContent: 'center',
                    height: '100%',
                    color: 'text.secondary',
                  }}
                >
                  <Typography>Select two versions from the lists to see the difference.</Typography>
                </Box>
              )}
              {compareError && <Alert severity="error" sx={{ mt: 2 }}>{compareError?.message}</Alert>}
              {compareData && (
                <Suspense fallback={<div>Loading visualizations...</div>}>
                  <Box sx={{ mt: 3 }}>
                    <ImpactSummary summary={compareData.summary} />
                    <Box sx={{ mt: 3, p: 2, height: '500px', border: '1px solid', borderColor: 'divider', borderRadius: 1 }}>
                      {viewMode === 'heatmap' ? (
                        <>
                          <Typography variant="h6" gutterBottom>Decision Change Heatmap</Typography>
                          <ChangeHeatmap
                            timeline={compareData.timeline}
                            bucketSize="week"
                            onCellClick={(bucket, changeType) => {
                              const versionANum = versionsData?.find((v: PolicyVersion) => v.id === versionA)?.version;
                              const versionBNum = versionsData?.find((v: PolicyVersion) => v.id === versionB)?.version;
                              showDrillDown('policy_compare', {
                                policyId,
                                versionA: versionANum,
                                versionB: versionBNum,
                                bucket,
                                changeType,
                                fromDate: format(fromDate!, 'yyyy-MM-dd'),
                                toDate: format(toDate!, 'yyyy-MM-dd'),
                                bucketSize: 'week',
                              });
                            }}
                          />
                        </>
                      ) : showDiffOnly ? (
                        <>
                          <Typography variant="h6" gutterBottom>Decision Change Timeline</Typography>
                          <DifferenceOnlyTimeline diffs={compareData.timeline} />
                        </>
                      ) : (
                        <>
                          <Typography variant="h6" gutterBottom>Decision Replay Timeline</Typography>
                          <DecisionReplayChart timeline={compareData.timeline} />
                        </>
                      )}
                    </Box>
                    <DecisionDiffTable diffs={compareData.timeline.filter((t: any) => t.decision_a !== t.decision_b)} />
                  </Box>
                </Suspense>
              )}
            </Box>
          </Paper>
        </Grid>
      </Grid>
    </Box>
  );
};

export default PolicyVersionDiffPage;