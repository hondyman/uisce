import React, { useState, useEffect, useRef, useCallback } from 'react';
import {
  Box,
  Paper,
  Typography,
  TextField,
  Button,
  IconButton,
  Chip,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  Stack,
  Avatar,
  Checkbox,
  CircularProgress,
  useTheme,
  alpha,
} from '@mui/material';
import {
  Add as AddIcon,
  Edit as EditIcon,
  Delete as DeleteIcon,
  Download as DownloadIcon,
  FilterList as FilterListIcon,
  Search as SearchIcon,
} from '@mui/icons-material';
import { useIPWhitelistAPI, useIPWhitelistTable } from '../hooks/useIPWhitelist';
import { useNotification } from '../../../hooks/useNotification';
import { IPWhitelistEntry } from '../types/ipWhitelist';


const IPManagementView: React.FC = () => {
  const theme = useTheme();
  const notification = useNotification();
  const api = useIPWhitelistAPI();
  
  const [entries, setEntries] = useState<IPWhitelistEntry[]>([]);
  const [tenants, setTenants] = useState<any[]>([]);
  const [selectedIds, setSelectedIds] = useState<Set<string>>(new Set());
  const [deleteConfirm, setDeleteConfirm] = useState<IPWhitelistEntry | null>(null);
  const tableRef = useRef<HTMLDivElement>(null);

  const table = useIPWhitelistTable(entries, tenants);

  // Load data on mount
  useEffect(() => {
    const loadData = async () => {
      const [entriesData, tenantsData] = await Promise.all([
        api.fetchAllIPWhitelist(),
        api.fetchTenants(),
      ]);
      setEntries(entriesData);
      setTenants(tenantsData);
    };
    loadData();
  }, []);

  // Infinite scroll handler
  const handleScroll = useCallback(() => {
    if (!tableRef.current) return;
    const element = tableRef.current.querySelector('[role="table"]');
    if (!element) return;
    
    const { scrollHeight, clientHeight, scrollTop } = element;
    if (scrollHeight - scrollTop <= clientHeight + 100) {
      if (table.hasMore && !table.isLoadingMore) {
        table.loadMore();
      }
    }
  }, [table]);

  useEffect(() => {
    const container = tableRef.current;
    if (!container) return;
    container.addEventListener('scroll', handleScroll);
    return () => container.removeEventListener('scroll', handleScroll);
  }, [handleScroll]);

  // Handle select all
  const handleSelectAll = (event: React.ChangeEvent<HTMLInputElement>) => {
    if (event.target.checked) {
      const allIds = new Set(table.visibleEntries.map(e => e.ipAddress));
      setSelectedIds(allIds);
    } else {
      setSelectedIds(new Set());
    }
  };

  // Handle select row
  const handleSelectRow = (ipAddress: string) => {
    const newSelected = new Set(selectedIds);
    if (newSelected.has(ipAddress)) {
      newSelected.delete(ipAddress);
    } else {
      newSelected.add(ipAddress);
    }
    setSelectedIds(newSelected);
  };

  // Handle delete
  const handleDelete = async (entry: IPWhitelistEntry) => {
    const success = await api.removeIPWhitelist(entry.tenantIds?.[0] || '', entry.ipAddress);
    if (success) {
      setEntries(prev => prev.filter(e => e.ipAddress !== entry.ipAddress));
      notification.success(`IP ${entry.ipAddress} removed successfully`);
      setDeleteConfirm(null);
    } else {
      notification.error(`Failed to remove IP ${entry.ipAddress}`);
    }
  };

  // Handle export
  const handleExport = () => {
    const csv = [
      ['Status', 'IP Address', 'Tenant', 'Label', 'Date Added'],
      ...table.visibleEntries.map(e => [
        (e as any).allTenants ? 'Global' : 'Active',
        e.ipAddress,
        e.tenantIds?.map(id => tenants.find(t => t.id === id)?.displayName || id).join('; ') || 'None',
        e.label || '',
        e.createdAt || '',
      ])
    ].map(row => row.map(cell => `"${cell}"`).join(',')).join('\n');
    
    const blob = new Blob([csv], { type: 'text/csv' });
    const url = window.URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = 'ip-whitelist.csv';
    a.click();
    notification.success('IP whitelist exported successfully');
  };

  const getTenantInitials = (tenantId: string): string => {
    const tenant = tenants.find(t => t.id === tenantId);
    if (!tenant) return 'UN';
    const parts = tenant.displayName?.split(' ') || ['U'];
    return (parts[0]?.[0] + (parts[1]?.[0] || '')).toUpperCase();
  };

  const getTenantColor = (index: number): string => {
    const colors = ['#f3f4f6', '#e5e7eb', '#d1d5db', '#9ca3af', '#6b7280'];
    return colors[index % colors.length];
  };

  const allSelected = table.visibleEntries.length > 0 && table.visibleEntries.every(
    e => selectedIds.has(e.ipAddress)
  );

  return (
    <>
      {/* Toolbar */}
      <Stack spacing={3}>
        <Stack
          direction={{ xs: 'column', md: 'row' }}
          spacing={2}
          justifyContent="space-between"
          alignItems={{ xs: 'stretch', md: 'center' }}
          sx={{
            p: 2,
            backgroundColor: alpha(theme.palette.common.white, theme.palette.mode === 'dark' ? 0.05 : 0.5),
            borderRadius: 1.5,
            border: `1px solid ${theme.palette.divider}`,
          }}
        >
          {/* Search */}
          <TextField
            placeholder="Search by IP, CIDR or Label"
            size="small"
            value={table.searchQuery}
            onChange={(e) => table.setSearchQuery(e.target.value)}
            InputProps={{
              startAdornment: <SearchIcon sx={{ mr: 1, color: 'text.secondary', fontSize: 20 }} />,
            }}
            sx={{
              flex: { xs: 1, md: 'none' },
              maxWidth: { md: '320px' },
              '& .MuiOutlinedInput-root': {
                backgroundColor: theme.palette.background.paper,
              }
            }}
          />

          {/* Action Buttons */}
          <Stack direction="row" spacing={1} sx={{ flex: 1, justifyContent: { xs: 'flex-start', md: 'flex-end' } }}>
            <Button
              startIcon={<FilterListIcon />}
              variant="outlined"
              size="small"
              sx={{ textTransform: 'none' }}
            >
              Filter
            </Button>
            <Button
              startIcon={<DownloadIcon />}
              variant="outlined"
              size="small"
              onClick={handleExport}
              sx={{ textTransform: 'none' }}
            >
              Export
            </Button>
            <Button
              startIcon={<AddIcon />}
              variant="contained"
              size="small"
              sx={{ textTransform: 'none' }}
            >
              Add New IP
            </Button>
          </Stack>
        </Stack>

        {/* Table Container */}
        <TableContainer
          ref={tableRef}
          component={Paper}
          sx={{
            maxHeight: '650px',
            overflowY: 'auto',
            backgroundColor: theme.palette.background.paper,
            borderRadius: 1.5,
            border: `1px solid ${theme.palette.divider}`,
            '&::-webkit-scrollbar': {
              height: '6px',
              width: '6px',
            },
            '&::-webkit-scrollbar-track': {
              background: 'transparent',
            },
            '&::-webkit-scrollbar-thumb': {
              backgroundColor: theme.palette.action.disabled,
              borderRadius: '20px',
            },
          }}
        >
          <Table stickyHeader size="small">
            <TableHead>
              <TableRow 
                sx={{ 
                  backgroundColor: theme.palette.mode === 'dark' 
                    ? alpha(theme.palette.common.white, 0.05) 
                    : alpha(theme.palette.common.black, 0.02)
                }}
              >
                <TableCell padding="checkbox">
                  <Checkbox
                    indeterminate={selectedIds.size > 0 && selectedIds.size < table.visibleEntries.length}
                    checked={allSelected}
                    onChange={handleSelectAll}
                  />
                </TableCell>
                <TableCell
                  sx={{ 
                    fontSize: '0.75rem', 
                    fontWeight: 700, 
                    textTransform: 'uppercase', 
                    color: 'text.secondary',
                    letterSpacing: '0.5px'
                  }}
                >
                  Status
                </TableCell>
                <TableCell
                  sx={{ 
                    fontSize: '0.75rem', 
                    fontWeight: 700, 
                    textTransform: 'uppercase', 
                    color: 'text.secondary',
                    letterSpacing: '0.5px'
                  }}
                >
                  IP Address / CIDR
                </TableCell>
                <TableCell
                  sx={{ 
                    fontSize: '0.75rem', 
                    fontWeight: 700, 
                    textTransform: 'uppercase', 
                    color: 'text.secondary',
                    letterSpacing: '0.5px'
                  }}
                >
                  Tenant
                </TableCell>
                <TableCell
                  sx={{ 
                    fontSize: '0.75rem', 
                    fontWeight: 700, 
                    textTransform: 'uppercase', 
                    color: 'text.secondary',
                    letterSpacing: '0.5px'
                  }}
                >
                  Label
                </TableCell>
                <TableCell
                  sx={{ 
                    fontSize: '0.75rem', 
                    fontWeight: 700, 
                    textTransform: 'uppercase', 
                    color: 'text.secondary',
                    letterSpacing: '0.5px'
                  }}
                >
                  Date Added
                </TableCell>
                <TableCell
                  align="right"
                  sx={{ 
                    fontSize: '0.75rem', 
                    fontWeight: 700, 
                    textTransform: 'uppercase', 
                    color: 'text.secondary',
                    letterSpacing: '0.5px'
                  }}
                >
                  Actions
                </TableCell>
              </TableRow>
            </TableHead>
            <TableBody>
              {table.visibleEntries.length === 0 ? (
                <TableRow>
                  <TableCell colSpan={7} align="center" sx={{ py: 4 }}>
                    <Typography color="text.secondary">
                      {table.totalCount === 0 ? 'No IP addresses configured' : 'No matching results'}
                    </Typography>
                  </TableCell>
                </TableRow>
              ) : (
                table.visibleEntries.map((entry, index) => (
                  <TableRow
                    key={`${entry.ipAddress}-${index}`}
                    hover
                    sx={{
                      '&:hover': {
                        backgroundColor: alpha(theme.palette.primary.main, 0.05),
                        '& [data-actions]': {
                          opacity: 1,
                        }
                      },
                      backgroundColor: 'transparent',
                    }}
                  >
                    <TableCell padding="checkbox">
                      <Checkbox
                        checked={selectedIds.has(entry.ipAddress)}
                        onChange={() => handleSelectRow(entry.ipAddress)}
                      />
                    </TableCell>
                    <TableCell>
                      <Chip
                        label={(entry as any).allTenants ? 'Global' : 'Active'}
                        size="small"
                        color={(entry as any).allTenants ? 'info' : 'success'}
                        variant="filled"
                        sx={{ height: 24 }}
                      />
                    </TableCell>
                    <TableCell>
                      <Stack spacing={0.25}>
                        <Typography variant="body2" sx={{ fontFamily: 'monospace', fontWeight: 600 }}>
                          {entry.ipAddress}
                        </Typography>
                        <Typography variant="caption" color="text.secondary">
                          {entry.ipAddress.includes('/') ? '' : '/32'}
                        </Typography>
                      </Stack>
                    </TableCell>
                    <TableCell>
                      {(entry as any).allTenants ? (
                        <Chip label="All Tenants" size="small" variant="outlined" />
                      ) : entry.tenantIds && entry.tenantIds.length > 0 ? (
                        <Stack direction="row" spacing={0.5} flexWrap="wrap">
                          {entry.tenantIds.map((tenantId, tIdx) => {
                            const tenant = tenants.find(t => t.id === tenantId);
                            return (
                              <Stack key={tenantId} direction="row" alignItems="center" spacing={0.5}>
                                <Avatar
                                  sx={{
                                    width: 24,
                                    height: 24,
                                    fontSize: '0.7rem',
                                    fontWeight: 'bold',
                                    backgroundColor: getTenantColor(tIdx),
                                    color: theme.palette.text.primary,
                                  }}
                                >
                                  {getTenantInitials(tenantId)}
                                </Avatar>
                              </Stack>
                            );
                          })}
                        </Stack>
                      ) : (
                        <Typography variant="body2" color="text.secondary">
                          Unassigned
                        </Typography>
                      )}
                    </TableCell>
                    <TableCell>
                      <Typography variant="body2" color="text.secondary">
                        {entry.label || '—'}
                      </Typography>
                    </TableCell>
                    <TableCell>
                      <Typography variant="body2" color="text.secondary">
                        {entry.createdAt ? new Date(entry.createdAt).toLocaleDateString('en-US', { month: 'short', day: 'numeric', year: 'numeric' }) : '—'}
                      </Typography>
                    </TableCell>
                    <TableCell align="right">
                      <Stack 
                        direction="row" 
                        spacing={0.5} 
                        justifyContent="flex-end"
                        data-actions
                        sx={{ 
                          opacity: 0,
                          transition: 'opacity 0.2s'
                        }}
                      >
                        <IconButton
                          size="small"
                          onClick={() => {}}
                          sx={{
                            '&:hover': { backgroundColor: alpha(theme.palette.primary.main, 0.1) },
                          }}
                        >
                          <EditIcon fontSize="small" />
                        </IconButton>
                        <IconButton
                          size="small"
                          onClick={() => setDeleteConfirm(entry)}
                          sx={{
                            color: theme.palette.error.main,
                            '&:hover': { backgroundColor: alpha(theme.palette.error.main, 0.1) },
                          }}
                        >
                          <DeleteIcon fontSize="small" />
                        </IconButton>
                      </Stack>
                    </TableCell>
                  </TableRow>
                ))
              )}
            </TableBody>
          </Table>
        </TableContainer>

        {/* Load More & Info */}
        <Stack direction="row" justifyContent="space-between" alignItems="center" sx={{ px: 2 }}>
          <Typography variant="body2" color="text.secondary">
            Showing <strong>{table.visibleEntries.length}</strong> of <strong>{table.totalCount}</strong> results
          </Typography>
          {table.hasMore && (
            <Button
              onClick={table.loadMore}
              disabled={table.isLoadingMore}
              endIcon={table.isLoadingMore ? <CircularProgress size={16} /> : undefined}
              size="small"
            >
              {table.isLoadingMore ? 'Loading...' : 'Load More'}
            </Button>
          )}
        </Stack>
      </Stack>

      {/* Delete Confirmation Dialog */}
      <Dialog open={!!deleteConfirm} onClose={() => setDeleteConfirm(null)} maxWidth="sm" fullWidth>
        <DialogTitle>Delete IP Address</DialogTitle>
        <DialogContent>
          <Typography variant="body2" sx={{ mt: 2 }}>
            Are you sure you want to remove <strong>{deleteConfirm?.ipAddress}</strong> from the whitelist?
          </Typography>
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setDeleteConfirm(null)}>Cancel</Button>
          <Button
            onClick={() => deleteConfirm && handleDelete(deleteConfirm)}
            color="error"
            variant="contained"
          >
            Delete
          </Button>
        </DialogActions>
      </Dialog>
    </>
  );
};

export default IPManagementView;
