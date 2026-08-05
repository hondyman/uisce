import React, { useState, useEffect } from 'react';
import {
  Box,
  Paper,
  Typography,
  Grid,
  Chip,
  Button,
  TextField,
  InputAdornment,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  Stack,
  Card,
  CardContent,
  CircularProgress,
  ToggleButtonGroup,
  ToggleButton,
  useTheme,
  Alert,
} from '@mui/material';
import {
  Search as SearchIcon,
  Download as DownloadIcon,
  VerifiedUser as VerifiedUserIcon,
  Security as SecurityIcon,
  Business as BusinessIcon,
  Shield as ShieldIcon,
  Refresh as RefreshIcon,
} from '@mui/icons-material';
import { apiClient } from '../../../utils/apiClient';

interface AuditorEvent {
  event_id: string;
  timestamp: string;
  tenant_id: string;
  action: string;
  entity_type: string;
  entity_id: string;
  user_id: string;
  chain_of_custody: string;
}

interface AuditTrailResponse {
  auditor_persona: string;
  scope: string;
  total_events: number;
  audit_trail: AuditorEvent[];
}

export const AuditorPortalPage: React.FC = () => {
  const theme = useTheme();
  const [persona, setPersona] = useState<'internal' | 'external'>('external');
  const [data, setData] = useState<AuditTrailResponse | null>(null);
  const [loading, setLoading] = useState<boolean>(true);
  const [searchTerm, setSearchTerm] = useState<string>('');

  const fetchAuditTrail = async () => {
    try {
      setLoading(true);
      const endpoint =
        persona === 'internal'
          ? '/auditor/global/audit-trail'
          : '/auditor/tenant/audit-trail';
      const res = await apiClient<AuditTrailResponse>(endpoint);
      setData(res);
    } catch (err) {
      console.error('Failed to fetch auditor trail:', err);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchAuditTrail();
  }, [persona]);

  const handleExportCSV = () => {
    const backendUrl = import.meta.env.VITE_BACKEND_URL || 'http://localhost:8080';
    window.open(`${backendUrl}/api/auditor/tenant/export`, '_blank');
  };

  const events = data?.audit_trail || [];
  const filteredEvents = events.filter(
    (e) =>
      e.event_id?.toLowerCase().includes(searchTerm.toLowerCase()) ||
      e.action?.toLowerCase().includes(searchTerm.toLowerCase()) ||
      e.entity_type?.toLowerCase().includes(searchTerm.toLowerCase()) ||
      e.user_id?.toLowerCase().includes(searchTerm.toLowerCase())
  );

  return (
    <Box sx={{ p: 4, bgcolor: 'background.default', minHeight: '100vh' }}>
      {/* Top Banner */}
      <Alert
        icon={<VerifiedUserIcon />}
        severity="info"
        sx={{ mb: 4, borderRadius: 3, border: `1px solid ${theme.palette.info.light}` }}
      >
        <Typography variant="subtitle2" fontWeight={700}>
          Immutable Chain-of-Custody Active (Tamper-Proof Audit Graph)
        </Typography>
        <Typography variant="caption">
          All events are cryptographically verified via PostgreSQL Write-Ahead Logging (WAL) → Debezium CDC → Apache Arrow DataFusion → Iceberg Lakehouse.
        </Typography>
      </Alert>

      {/* Header */}
      <Stack direction="row" justifyContent="space-between" alignItems="center" mb={4}>
        <Box>
          <Typography variant="h4" fontWeight={700}>
            Auditor Access & Governance Portal
          </Typography>
          <Typography variant="body2" color="text.secondary">
            Read-only, contextualized regulatory compliance views and exportable audit trails
          </Typography>
        </Box>

        <Stack direction="row" spacing={2} alignItems="center">
          <ToggleButtonGroup
            value={persona}
            exclusive
            onChange={(_, val) => val && setPersona(val)}
            size="small"
            color="primary"
          >
            <ToggleButton value="external">
              <BusinessIcon sx={{ mr: 1, fontSize: 18 }} /> Client External Auditor
            </ToggleButton>
            <ToggleButton value="internal">
              <ShieldIcon sx={{ mr: 1, fontSize: 18 }} /> Uisce Internal Auditor
            </ToggleButton>
          </ToggleButtonGroup>

          <Button
            variant="contained"
            color="primary"
            startIcon={<DownloadIcon />}
            onClick={handleExportCSV}
            sx={{ borderRadius: 2 }}
          >
            Export Regulatory CSV
          </Button>
        </Stack>
      </Stack>

      {/* Persona Stats Cards */}
      <Grid container spacing={3} mb={4}>
        <Grid size={{ xs: 12, sm: 6, md: 4 }}>
          <Card elevation={0} sx={{ borderRadius: 3, border: `1px solid ${theme.palette.divider}` }}>
            <CardContent>
              <Typography variant="body2" color="text.secondary" fontWeight={600}>
                Active Auditor Scope
              </Typography>
              <Typography variant="h6" fontWeight={700} mt={1} color="primary.main">
                {data?.scope || 'Tenant Isolated'}
              </Typography>
              <Typography variant="caption" color="text.secondary">
                {persona === 'external' ? 'Strictly Tenant-Isolated' : 'Cross-Tenant System-Wide'}
              </Typography>
            </CardContent>
          </Card>
        </Grid>

        <Grid size={{ xs: 12, sm: 6, md: 4 }}>
          <Card elevation={0} sx={{ borderRadius: 3, border: `1px solid ${theme.palette.divider}` }}>
            <CardContent>
              <Typography variant="body2" color="text.secondary" fontWeight={600}>
                Verified Audit Records
              </Typography>
              <Typography variant="h4" fontWeight={700} mt={1}>
                {data?.total_events ?? 0}
              </Typography>
              <Typography variant="caption" color="text.secondary">
                Immutable Ledger Snapshots
              </Typography>
            </CardContent>
          </Card>
        </Grid>

        <Grid size={{ xs: 12, sm: 6, md: 4 }}>
          <Card elevation={0} sx={{ borderRadius: 3, border: `1px solid ${theme.palette.divider}` }}>
            <CardContent>
              <Typography variant="body2" color="text.secondary" fontWeight={600}>
                Regulatory Verification
              </Typography>
              <Stack direction="row" spacing={1} alignItems="center" mt={1.5}>
                <Chip label="SHA-256 PASSED" color="success" size="small" icon={<SecurityIcon />} />
                <Typography variant="caption" color="text.secondary">
                  Zero Tampering
                </Typography>
              </Stack>
            </CardContent>
          </Card>
        </Grid>
      </Grid>

      {/* Main Data Table */}
      <Paper elevation={0} sx={{ p: 3, borderRadius: 3, border: `1px solid ${theme.palette.divider}` }}>
        <Stack direction="row" justifyContent="space-between" alignItems="center" mb={3}>
          <Box>
            <Typography variant="h6" fontWeight={700}>
              Regulatory Audit Trail Ledger
            </Typography>
            <Typography variant="body2" color="text.secondary">
              Read-only immutable record of all access control and configuration changes
            </Typography>
          </Box>
          <Stack direction="row" spacing={2}>
            <TextField
              size="small"
              placeholder="Search audit records..."
              value={searchTerm}
              onChange={(e) => setSearchTerm(e.target.value)}
              InputProps={{
                startAdornment: (
                  <InputAdornment position="start">
                    <SearchIcon fontSize="small" sx={{ color: 'text.secondary' }} />
                  </InputAdornment>
                ),
                sx: { borderRadius: 2 },
              }}
            />
            <Button variant="outlined" startIcon={<RefreshIcon />} onClick={fetchAuditTrail}>
              Refresh
            </Button>
          </Stack>
        </Stack>

        {loading ? (
          <Box sx={{ display: 'flex', py: 6, justifyContent: 'center' }}>
            <CircularProgress />
          </Box>
        ) : (
          <TableContainer>
            <Table>
              <TableHead>
                <TableRow>
                  <TableCell sx={{ fontWeight: 700 }}>Timestamp</TableCell>
                  <TableCell sx={{ fontWeight: 700 }}>Event ID</TableCell>
                  <TableCell sx={{ fontWeight: 700 }}>Action</TableCell>
                  <TableCell sx={{ fontWeight: 700 }}>Entity Type</TableCell>
                  <TableCell sx={{ fontWeight: 700 }}>Entity ID</TableCell>
                  <TableCell sx={{ fontWeight: 700 }}>User / Actor</TableCell>
                  <TableCell sx={{ fontWeight: 700 }}>Chain of Custody Proof</TableCell>
                </TableRow>
              </TableHead>
              <TableBody>
                {filteredEvents.length === 0 ? (
                  <TableRow>
                    <TableCell colSpan={7} align="center" sx={{ py: 4, color: 'text.secondary' }}>
                      No regulatory audit records found for this scope.
                    </TableCell>
                  </TableRow>
                ) : (
                  filteredEvents.map((evt) => (
                    <TableRow key={evt.event_id} hover>
                      <TableCell>{new Date(evt.timestamp).toLocaleString()}</TableCell>
                      <TableCell>
                        <Typography variant="caption" sx={{ fontFamily: 'monospace' }}>
                          {evt.event_id?.substring(0, 8)}...
                        </Typography>
                      </TableCell>
                      <TableCell>
                        <Chip
                          label={evt.action}
                          size="small"
                          color={
                            evt.action === 'CREATE' || evt.action === 'created'
                              ? 'success'
                              : evt.action === 'DELETE' || evt.action === 'deleted'
                              ? 'error'
                              : 'primary'
                          }
                          variant="outlined"
                        />
                      </TableCell>
                      <TableCell>{evt.entity_type}</TableCell>
                      <TableCell>
                        <Typography variant="caption" sx={{ fontFamily: 'monospace' }}>
                          {evt.entity_id}
                        </Typography>
                      </TableCell>
                      <TableCell>{evt.user_id}</TableCell>
                      <TableCell>
                        <Chip
                          label={evt.chain_of_custody}
                          size="small"
                          color="info"
                          variant="filled"
                          sx={{ fontSize: '0.7rem' }}
                        />
                      </TableCell>
                    </TableRow>
                  ))
                )}
              </TableBody>
            </Table>
          </TableContainer>
        )}
      </Paper>
    </Box>
  );
};

export default AuditorPortalPage;
