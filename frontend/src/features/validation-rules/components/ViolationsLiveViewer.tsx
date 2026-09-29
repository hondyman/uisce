import React, { useState, useEffect } from 'react';
import {
  Box,
  Card,
  CardContent,
  Typography,
  Stack,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  Paper,
  Chip,
  IconButton,
  Button,
  CircularProgress,
  Alert,
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  FormControl,
  InputLabel,
  Select,
  MenuItem,
} from '@mui/material';
import RefreshIcon from '@mui/icons-material/Refresh';
import VisibilityIcon from '@mui/icons-material/Visibility';
import { validationRulesApi, type ViolationRecordItem } from '../api';

interface ViolationsLiveViewerProps {
  initialBOKeys?: string[];
  limit?: number;
}

export const ViolationsLiveViewer: React.FC<ViolationsLiveViewerProps> = ({
  initialBOKeys = ['order', 'execution', 'security', 'account', 'party'],
  limit = 100,
}) => {
  const [selectedBO, setSelectedBO] = useState<string>('');
  const [violations, setViolations] = useState<ViolationRecordItem[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [selectedContext, setSelectedContext] = useState<any | null>(null);

  const loadViolations = async () => {
    setLoading(true);
    setError(null);
    try {
      const data = await validationRulesApi.listViolations(selectedBO || undefined, limit);
      setViolations(data || []);
    } catch (err: any) {
      setError(err.message || 'Failed to load violations');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    loadViolations();
  }, [selectedBO]);

  return (
    <Card variant="outlined" sx={{ borderRadius: 2 }}>
      <CardContent>
        <Stack direction={{ xs: 'column', sm: 'row' }} spacing={2} justifyContent="space-between" alignItems="center" sx={{ mb: 2 }}>
          <Box>
            <Typography variant="h6" fontWeight="bold">
              Centralized Validation Violations Log
            </Typography>
            <Typography variant="body2" color="text.secondary">
              Authoritative queryable sink on alpha.public.validation_rule_violations capturing all API and pipeline blocks.
            </Typography>
          </Box>
          <Stack direction="row" spacing={2} alignItems="center">
            <FormControl size="small" sx={{ minWidth: 160 }}>
              <InputLabel>Filter by BO</InputLabel>
              <Select
                value={selectedBO}
                label="Filter by BO"
                onChange={(e) => setSelectedBO(e.target.value)}
              >
                <MenuItem value="">All Business Objects</MenuItem>
                {initialBOKeys.map((bo) => (
                  <MenuItem key={bo} value={bo}>{bo}</MenuItem>
                ))}
              </Select>
            </FormControl>
            <Button
              variant="outlined"
              startIcon={<RefreshIcon />}
              onClick={loadViolations}
              disabled={loading}
            >
              Refresh
            </Button>
          </Stack>
        </Stack>

        {error && <Alert severity="error" sx={{ mb: 2 }}>{error}</Alert>}

        {loading ? (
          <Box sx={{ display: 'flex', justifyContent: 'center', p: 4 }}>
            <CircularProgress />
          </Box>
        ) : (
          <TableContainer component={Paper} variant="outlined" sx={{ maxHeight: 440 }}>
            <Table size="small" stickyHeader>
              <TableHead>
                <TableRow>
                  <TableCell>Timestamp</TableCell>
                  <TableCell>BO Key</TableCell>
                  <TableCell>Rule Name</TableCell>
                  <TableCell>Severity</TableCell>
                  <TableCell>Record ID</TableCell>
                  <TableCell>Violation Message</TableCell>
                  <TableCell align="center">Blocked</TableCell>
                  <TableCell align="center">Context</TableCell>
                </TableRow>
              </TableHead>
              <TableBody>
                {violations.length === 0 ? (
                  <TableRow>
                    <TableCell colSpan={8} align="center" sx={{ py: 3, color: 'text.secondary' }}>
                      No rule violations recorded.
                    </TableCell>
                  </TableRow>
                ) : (
                  violations.map((v) => (
                    <TableRow key={v.id} hover>
                      <TableCell sx={{ whiteSpace: 'nowrap', fontSize: '0.8rem' }}>
                        {v.created_at ? new Date(v.created_at).toLocaleString() : '-'}
                      </TableCell>
                      <TableCell>
                        <Chip label={v.bo_key} size="small" variant="outlined" />
                      </TableCell>
                      <TableCell fontWeight="medium">{v.rule_name}</TableCell>
                      <TableCell>
                        <Chip
                          label={v.severity}
                          size="small"
                          color={v.severity === 'BLOCK' ? 'error' : 'warning'}
                        />
                      </TableCell>
                      <TableCell sx={{ fontFamily: 'monospace', fontSize: '0.8rem' }}>
                        {v.record_id || '-'}
                      </TableCell>
                      <TableCell sx={{ maxWidth: 300 }}>{v.message}</TableCell>
                      <TableCell align="center">
                        {v.write_blocked ? (
                          <Chip label="BLOCKED" color="error" size="small" />
                        ) : (
                          <Chip label="LOGGED" size="small" />
                        )}
                      </TableCell>
                      <TableCell align="center">
                        <IconButton
                          size="small"
                          onClick={() => setSelectedContext(v.context)}
                          title="View payload context"
                        >
                          <VisibilityIcon fontSize="small" />
                        </IconButton>
                      </TableCell>
                    </TableRow>
                  ))
                )}
              </TableBody>
            </Table>
          </TableContainer>
        )}

        {/* Context Preview Modal */}
        <Dialog
          open={Boolean(selectedContext)}
          onClose={() => setSelectedContext(null)}
          maxWidth="sm"
          fullWidth
        >
          <DialogTitle>Violation Payload Context</DialogTitle>
          <DialogContent dividers>
            <Paper variant="outlined" sx={{ p: 2, bgcolor: '#1e293b', color: '#f8fafc', overflow: 'auto' }}>
              <pre style={{ margin: 0, fontSize: '0.85rem' }}>
                {selectedContext ? JSON.stringify(selectedContext, null, 2) : '{}'}
              </pre>
            </Paper>
          </DialogContent>
          <DialogActions>
            <Button onClick={() => setSelectedContext(null)}>Close</Button>
          </DialogActions>
        </Dialog>
      </CardContent>
    </Card>
  );
};
