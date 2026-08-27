import { useState, useEffect, useCallback } from 'react';
import { getClaimLifecycleSnapshot } from './api';
import { ClaimLifecycleSnapshot, ClaimLifecycleEvent } from './types';
import ClaimsTable from './ClaimsTable';
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
import Button from '@mui/material/Button';
import Chip from '@mui/material/Chip';

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

const LifecycleTimeline = ({ events }: { events: ClaimLifecycleEvent[] }) => {
  const theme = useTheme();

  const getStatusColor = (eventType: string) => {
    if (eventType.includes('created')) return 'success';
    if (eventType.includes('expired') || eventType.includes('revoked')) return 'error';
    if (eventType.includes('renewal')) return 'warning';
    return 'default';
  };

  return (
    <Box sx={{ mb: 3 }}>
      <Typography variant="h6" sx={{ mb: 2 }}>
        Recent Claim Events
      </Typography>
      <TableContainer component={Paper}>
        <Table>
          <TableHead>
            <TableRow>
              <TableCell>Timestamp</TableCell>
              <TableCell>Event</TableCell>
              <TableCell>Actor</TableCell>
              <TableCell>Notes</TableCell>
            </TableRow>
          </TableHead>
          <TableBody>
            {events.map(event => (
              <TableRow key={event.id}>
                <TableCell>{new Date(event.timestamp).toLocaleString()}</TableCell>
                <TableCell>
                  <Chip
                    label={event.event_type.replace(/_/g, ' ')}
                    color={getStatusColor(event.event_type) as 'success' | 'error' | 'warning' | 'default'}
                    size="small"
                  />
                </TableCell>
                <TableCell>{event.actor_user_id}</TableCell>
                <TableCell>{event.notes || '-'}</TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      </TableContainer>
    </Box>
  );
};

export default function ClaimLifecycleDashboard() {
  const [snapshot, setSnapshot] = useState<ClaimLifecycleSnapshot | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [filterStatus, setFilterStatus] = useState<string>('active');

  const fetchSnapshot = useCallback(async () => {
    try {
      setLoading(true);
      const data = await getClaimLifecycleSnapshot();
      setSnapshot(data);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to fetch lifecycle data');
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    fetchSnapshot();
  }, [fetchSnapshot]);

  if (loading) return <Box sx={{ p: 4 }}>Loading claim lifecycle dashboard...</Box>;
  if (error) return <Box sx={{ p: 4, color: 'error.main' }}>Error: {error}</Box>;
  if (!snapshot) return <Box sx={{ p: 4 }}>No claim data available.</Box>;

  const filterButtons = [
    { id: 'active', label: 'Active' },
    { id: 'expiring', label: 'Expiring' },
    { id: 'renewal_requested', label: 'Pending Renewal' },
    { id: 'expired', label: 'Expired' },
    { id: 'revoked', label: 'Revoked' },
  ];

  return (
    <Box sx={{ p: 3 }}>
      <Typography variant="h5" sx={{ mb: 3, fontWeight: 600 }}>
        Claim Lifecycle Overview
      </Typography>

      <Box
        sx={{
          display: 'grid',
          gridTemplateColumns: { xs: '1fr', sm: 'repeat(2, 1fr)', md: 'repeat(5, 1fr)' },
          gap: 2,
          mb: 3,
        }}
      >
        <MetricCard title="Active Claims" value={snapshot.active_count} />
        <MetricCard title="Expiring Soon" value={snapshot.expiring_soon_count} warning={snapshot.expiring_soon_count > 0} />
        <MetricCard title="Renewal Requests" value={snapshot.renewal_requested_count} />
        <MetricCard title="Expired Claims" value={snapshot.expired_count} />
        <MetricCard title="Revoked Claims" value={snapshot.revoked_count} />
      </Box>

      <LifecycleTimeline events={snapshot.recent_events} />

      <Box sx={{ mt: 3 }}>
        <Typography variant="h6" sx={{ mb: 2 }}>
          Claims List
        </Typography>
        <Box sx={{ display: 'flex', gap: 1, mb: 2, flexWrap: 'wrap' }}>
          {filterButtons.map(btn => (
            <Button
              key={btn.id}
              variant={filterStatus === btn.id ? 'contained' : 'outlined'}
              size="small"
              onClick={() => setFilterStatus(btn.id)}
            >
              {btn.label}
            </Button>
          ))}
        </Box>
        <ClaimsTable statusFilter={filterStatus} />
      </Box>
    </Box>
  );
}
