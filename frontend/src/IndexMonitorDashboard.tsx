import { useState, useEffect, useCallback } from 'react';
import { getIndexMonitorSnapshot } from './api';
import { IndexMonitorSnapshot, IndexJob, AssetFreshness } from './types';
import MetricBreakdown from './MetricBreakdown';
import { useTheme } from '@mui/material/styles';
import Box from '@mui/material/Box';
import Typography from '@mui/material/Typography';
import Paper from '@mui/material/Paper';
import Table from '@mui/material/Table';
import TableBody from '@mui/material/TableBody';
import TableCell from '@mui/material/TableCell';
import TableContainer from '@mui/material/TableContainer';
import TableHead from '@mui/material/TableHead';
import TableRow from '@mui/material/TableRow';
import Chip from '@mui/material/Chip';
import LinearProgress from '@mui/material/LinearProgress';

const MetricCard = ({ title, value, warning = false }: { title: string; value: number | string; warning?: boolean }) => {
  const theme = useTheme();
  return (
    <Paper
      sx={{
        p: 2,
        textAlign: 'center',
        backgroundColor: warning ? 'warning.light' : 'background.paper',
        border: warning ? '2px solid' : '1px solid',
        borderColor: warning ? 'warning.main' : 'divider',
      }}
    >
      <Typography variant="h4" sx={{ fontWeight: 600, color: warning ? 'warning.dark' : 'text.primary' }}>
        {value}
      </Typography>
      <Typography variant="body2" color="text.secondary">
        {title}
      </Typography>
    </Paper>
  );
};

const ProgressBar = ({ title, percent }: { title: string; percent: number }) => {
  return (
    <Paper sx={{ p: 2, mb: 2 }}>
      <Box sx={{ display: 'flex', justifyContent: 'space-between', mb: 1 }}>
        <Typography variant="body2" fontWeight={500}>{title}</Typography>
        <Typography variant="body2">{percent.toFixed(1)}%</Typography>
      </Box>
      <LinearProgress variant="determinate" value={percent} sx={{ height: 8, borderRadius: 4 }} />
    </Paper>
  );
};

const JobTimeline = ({ jobs }: { jobs: IndexJob[] }) => {
  const getStatusColor = (status: string) => {
    switch (status) {
      case 'completed': return 'success';
      case 'failed': return 'error';
      case 'running': return 'info';
      case 'pending': return 'warning';
      default: return 'default';
    }
  };

  return (
    <Box sx={{ mb: 3 }}>
      <Typography variant="h6" sx={{ mb: 2 }}>
        Recent Indexing Jobs
      </Typography>
      <TableContainer component={Paper}>
        <Table>
          <TableHead>
            <TableRow>
              <TableCell>Status</TableCell>
              <TableCell>Job Type</TableCell>
              <TableCell>Triggered By</TableCell>
              <TableCell>Assets Affected</TableCell>
              <TableCell>Timestamp</TableCell>
            </TableRow>
          </TableHead>
          <TableBody>
            {jobs.map(job => (
              <TableRow key={job.id}>
                <TableCell>
                  <Chip
                    label={job.status}
                    color={getStatusColor(job.status) as 'success' | 'error' | 'info' | 'warning' | 'default'}
                    size="small"
                  />
                </TableCell>
                <TableCell>{job.job_type}</TableCell>
                <TableCell>{job.triggered_by}</TableCell>
                <TableCell>{job.affected_assets > 0 ? job.affected_assets : '-'}</TableCell>
                <TableCell>{new Date(job.started_at).toLocaleString()}</TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      </TableContainer>
    </Box>
  );
};

const StaleAssetList = ({ assets }: { assets: AssetFreshness[] }) => {
  return (
    <Box sx={{ mb: 3 }}>
      <Typography variant="h6" sx={{ mb: 2 }}>
        Stale Assets (Not Indexed in &gt;7 Days)
      </Typography>
      {assets.length === 0 ? (
        <Paper sx={{ p: 2, textAlign: 'center' }}>
          <Typography>No stale assets found. ✅</Typography>
        </Paper>
      ) : (
        <TableContainer component={Paper}>
          <Table>
            <TableHead>
              <TableRow>
                <TableCell>Asset Name</TableCell>
                <TableCell>Type</TableCell>
                <TableCell>Last Indexed</TableCell>
                <TableCell>Certified</TableCell>
              </TableRow>
            </TableHead>
            <TableBody>
              {assets.map(asset => (
                <TableRow key={asset.asset_id}>
                  <TableCell>{asset.asset_name}</TableCell>
                  <TableCell>{asset.asset_type}</TableCell>
                  <TableCell>{new Date(asset.last_indexed_at).toLocaleDateString()}</TableCell>
                  <TableCell>{asset.certified ? '✅' : '❌'}</TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </TableContainer>
      )}
    </Box>
  );
};

export default function IndexMonitorDashboard() {
  const [snapshot, setSnapshot] = useState<IndexMonitorSnapshot | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const fetchSnapshot = useCallback(async () => {
    try {
      setLoading(true);
      const data = await getIndexMonitorSnapshot();
      setSnapshot(data);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to fetch index monitor data');
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    fetchSnapshot();
  }, [fetchSnapshot]);

  if (loading) return <Box sx={{ p: 4 }}>Loading index monitor...</Box>;
  if (error) return <Box sx={{ p: 4, color: 'error.main' }}>Error: {error}</Box>;
  if (!snapshot) return <Box sx={{ p: 4 }}>No index data available.</Box>;

  const timeAgo = (dateString: string) => {
    const date = new Date(dateString);
    const seconds = Math.floor((new Date().getTime() - date.getTime()) / 1000);
    let interval = seconds / 31536000;
    if (interval > 1) return Math.floor(interval) + " years ago";
    interval = seconds / 2592000;
    if (interval > 1) return Math.floor(interval) + " months ago";
    interval = seconds / 86400;
    if (interval > 1) return Math.floor(interval) + " days ago";
    interval = seconds / 3600;
    if (interval > 1) return Math.floor(interval) + " hours ago";
    interval = seconds / 60;
    if (interval > 1) return Math.floor(interval) + " minutes ago";
    return Math.floor(seconds) + " seconds ago";
  };

  return (
    <Box sx={{ p: 3 }}>
      <ProgressBar title="Semantic Health Score" percent={snapshot.semantic_health_score} />
      <MetricBreakdown
        certified={snapshot.certified_coverage}
        claims={snapshot.claim_alignment}
        usage={snapshot.usage_coverage}
        audit={snapshot.audit_completeness}
        risk={snapshot.risk_exposure}
      />
      <Box
        sx={{
          display: 'grid',
          gridTemplateColumns: { xs: '1fr', md: 'repeat(3, 1fr)' },
          gap: 2,
          mb: 3,
        }}
      >
        <MetricCard title="Last Full Refresh" value={timeAgo(snapshot.last_full_refresh)} />
        <MetricCard title="Certified Coverage" value={`${snapshot.certified_coverage.toFixed(1)}%`} />
        <MetricCard title="Unindexed Assets" value={snapshot.unindexed_asset_count} warning={snapshot.unindexed_asset_count > 0} />
      </Box>
      <JobTimeline jobs={snapshot.recent_jobs} />
      <StaleAssetList assets={snapshot.stale_assets} />
    </Box>
  );
}
