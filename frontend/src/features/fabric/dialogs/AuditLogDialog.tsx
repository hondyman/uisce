import React from 'react';
import {
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  Button,
  Chip,
  useTheme,
  Box,
  TextField,
  Stack,
  TablePagination,
} from '@mui/material';
import { Search as SearchIcon } from '@mui/icons-material';

export interface AuditLog {
  id: string;
  timestamp: string;
  action: 'created' | 'modified' | 'activated' | 'deactivated' | 'deleted';
  ruleName: string;
  user: string;
  details?: string;
}

interface AuditLogDialogProps {
  open: boolean;
  onClose: () => void;
  logs?: AuditLog[];
}

const AuditLogDialog: React.FC<AuditLogDialogProps> = ({
  open,
  onClose,
  logs = [
    {
      id: '1',
      timestamp: '2024-12-17 10:45:22',
      action: 'created',
      ruleName: 'check_invoice_total_positive',
      user: 'admin@company.com',
    },
    {
      id: '2',
      timestamp: '2024-12-17 09:30:15',
      action: 'modified',
      ruleName: 'validate_vendor_tax_id',
      user: 'john.doe@company.com',
    },
    {
      id: '3',
      timestamp: '2024-12-16 14:22:08',
      action: 'deactivated',
      ruleName: 'cross_reference_po',
      user: 'jane.smith@company.com',
    },
    {
      id: '4',
      timestamp: '2024-12-16 11:10:45',
      action: 'created',
      ruleName: 'check_due_date_future',
      user: 'admin@company.com',
    },
  ],
}) => {
  const theme = useTheme();
  const [searchQuery, setSearchQuery] = React.useState('');
  const [page, setPage] = React.useState(0);
  const [rowsPerPage, setRowsPerPage] = React.useState(10);

  const filteredLogs = logs.filter(log =>
    log.ruleName.toLowerCase().includes(searchQuery.toLowerCase()) ||
    log.user.toLowerCase().includes(searchQuery.toLowerCase()) ||
    log.action.toLowerCase().includes(searchQuery.toLowerCase())
  );

  const paginatedLogs = filteredLogs.slice(page * rowsPerPage, (page + 1) * rowsPerPage);

  const getActionColor = (action: string) => {
    switch (action) {
      case 'created':
        return 'success';
      case 'modified':
        return 'primary';
      case 'activated':
        return 'success';
      case 'deactivated':
        return 'warning';
      case 'deleted':
        return 'error';
      default:
        return 'default';
    }
  };

  const getActionLabel = (action: string) => {
    return action.charAt(0).toUpperCase() + action.slice(1);
  };

  return (
    <Dialog open={open} onClose={onClose} maxWidth="md" fullWidth>
      <DialogTitle>Audit Log - Validation Rules</DialogTitle>
      <DialogContent sx={{ pt: 2 }}>
        <Stack spacing={2}>
          {/* Search */}
          <TextField
            size="small"
            fullWidth
            placeholder="Search by rule name, user, or action..."
            value={searchQuery}
            onChange={(e) => {
              setSearchQuery(e.target.value);
              setPage(0);
            }}
            InputProps={{
              startAdornment: <SearchIcon sx={{ mr: 1, color: 'action.active', fontSize: '20px' }} />
            }}
          />

          {/* Table */}
          <TableContainer sx={{ border: `1px solid ${theme.palette.divider}`, borderRadius: 1 }}>
            <Table size="small">
              <TableHead>
                <TableRow sx={{ bgcolor: theme.palette.mode === 'dark' ? 'grey.800' : 'grey.100' }}>
                  <TableCell sx={{ fontWeight: 700 }}>Timestamp</TableCell>
                  <TableCell sx={{ fontWeight: 700 }}>Action</TableCell>
                  <TableCell sx={{ fontWeight: 700 }}>Rule</TableCell>
                  <TableCell sx={{ fontWeight: 700 }}>User</TableCell>
                </TableRow>
              </TableHead>
              <TableBody>
                {paginatedLogs.map((log) => (
                  <TableRow key={log.id} hover>
                    <TableCell sx={{ fontSize: '0.875rem' }}>{log.timestamp}</TableCell>
                    <TableCell>
                      <Chip
                        label={getActionLabel(log.action)}
                        size="small"
                        color={getActionColor(log.action) as any}
                        variant="filled"
                      />
                    </TableCell>
                    <TableCell sx={{ fontSize: '0.875rem' }}>{log.ruleName}</TableCell>
                    <TableCell sx={{ fontSize: '0.875rem' }}>{log.user}</TableCell>
                  </TableRow>
                ))}
                {paginatedLogs.length === 0 && (
                  <TableRow>
                    <TableCell colSpan={4} align="center" sx={{ py: 3 }}>
                      No audit logs found
                    </TableCell>
                  </TableRow>
                )}
              </TableBody>
            </Table>
          </TableContainer>

          {/* Pagination */}
          {filteredLogs.length > 0 && (
            <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
              <Box sx={{ fontSize: '0.875rem', color: 'text.secondary' }}>
                Showing {filteredLogs.length === 0 ? 0 : page * rowsPerPage + 1} to{' '}
                {Math.min((page + 1) * rowsPerPage, filteredLogs.length)} of {filteredLogs.length} logs
              </Box>
              <TablePagination
                rowsPerPageOptions={[5, 10, 25]}
                component="div"
                count={filteredLogs.length}
                rowsPerPage={rowsPerPage}
                page={page}
                onPageChange={(_, newPage) => setPage(newPage)}
                onRowsPerPageChange={(e) => {
                  setRowsPerPage(parseInt(e.target.value, 10));
                  setPage(0);
                }}
              />
            </Box>
          )}
        </Stack>
      </DialogContent>
      <DialogActions>
        <Button onClick={onClose}>Close</Button>
      </DialogActions>
    </Dialog>
  );
};

export default AuditLogDialog;
