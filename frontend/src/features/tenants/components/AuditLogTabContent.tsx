import React, { useState, useEffect } from 'react';
import {
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  Paper,
  Button,
  TextField,
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  Chip,
  Box,
  Typography,
  Avatar,
  IconButton,
  Alert,
  CircularProgress,
  TableSortLabel,
} from '@mui/material';
import {
  Download,
  FilterList,
  Search,
  Info,
  Refresh,
} from '@mui/icons-material';
import { LocalizationProvider } from '@mui/x-date-pickers/LocalizationProvider';
import { AdapterDayjs } from '@mui/x-date-pickers/AdapterDayjs';
import { DatePicker } from '@mui/x-date-pickers/DatePicker';
import dayjs, { Dayjs } from 'dayjs';

export interface AuditLogEntry {
  id: string;
  timestamp: string;
  user: {
    name: string;
    email: string;
    initials: string;
    color: string;
  };
  action: string;
  resource: string;
  resourceType: string;
  details: string;
}

interface AuditLogTabContentProps {
  tenantId: string;
  datasourceId: string;
  onViewDetails?: (entry: AuditLogEntry) => void;
  onExport?: () => void;
}

const getActionColor = (action: string) => {
  switch (action?.toLowerCase()) {
    case 'create':
      return 'success';
    case 'update':
      return 'info';
    case 'delete':
      return 'error';
    case 'backup':
      return 'default';
    case 'security':
      return 'warning';
    default:
      return 'default';
  }
};

const getActionIcon = (action: string) => {
  switch (action?.toLowerCase()) {
    case 'create':
      return '➕';
    case 'update':
      return '✏️';
    case 'delete':
      return '🗑️';
    case 'backup':
      return '💾';
    case 'security':
      return '🔒';
    default:
      return '📝';
  }
};

const getActionLabel = (action: string) => {
  switch (action?.toLowerCase()) {
    case 'create':
      return 'Create';
    case 'update':
      return 'Update';
    case 'delete':
      return 'Delete';
    case 'backup':
      return 'Backup';
    case 'security':
      return 'Security';
    default:
      return action || 'Other';
  }
};

export const AuditLogTabContent: React.FC<AuditLogTabContentProps> = ({
  tenantId,
  datasourceId,
  onViewDetails,
  onExport,
}) => {
  const [searchQuery, setSearchQuery] = useState('');
  const [selectedEntry, setSelectedEntry] = useState<AuditLogEntry | null>(null);
  const [entries, setEntries] = useState<AuditLogEntry[]>([]);
  const [totalEntries, setTotalEntries] = useState(0);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  
  // Date Filters
  const [startDate, setStartDate] = useState<Dayjs | null>(null);
  const [endDate, setEndDate] = useState<Dayjs | null>(null);

  // State for Lazy Loading
  const [offset, setOffset] = useState(0);
  const [hasMore, setHasMore] = useState(true);
  const [sortBy, setSortBy] = useState('timestamp');
  const [sortOrder, setSortOrder] = useState<'asc' | 'desc'>('desc');
  const LIMIT = 20;

  // Ref for observer
  const observerTarget = React.useRef<HTMLDivElement>(null);

  // Reset function for manual refresh or filter change
  const resetAndFetch = () => {
    setEntries([]);
    setOffset(0);
    setHasMore(true);
    // The useEffect dependent on offset will trigger the fetch when offset becomes 0
    // But if offset is already 0, we need to force fetch. 
    // Easier strategy: explicitly call fetch with reset flag or just reset entries and let effect handle it?
    // Let's use a explicit fetch function that accumulates.
  };

  // Fetch audit log entries
  const fetchAuditLogs = React.useCallback(async (currentOffset: number, isRefresh: boolean = false) => {
    if (!tenantId || !datasourceId) return;

    setLoading(true);
    setError(null);
    try {
      let url = `/api/audit-logs?tenantId=${encodeURIComponent(tenantId)}&datasourceId=${encodeURIComponent(datasourceId)}&limit=${LIMIT}&offset=${currentOffset}&sortBy=${sortBy}&sortOrder=${sortOrder ? sortOrder.toUpperCase() : 'DESC'}`;
      
      if (startDate) {
          url += `&startDate=${encodeURIComponent(startDate.toISOString())}`;
      }
      if (endDate) {
          url += `&endDate=${encodeURIComponent(endDate.toISOString())}`;
      }

      const response = await fetch(url);
      if (!response.ok) {
        throw new Error(`HTTP error! status: ${response.status}`);
      }
      const data = await response.json();
      
      // Transform API response
      const transformedEntries: AuditLogEntry[] = (data.entries || []).map((entry: any) => ({
        ...entry,
        details: typeof entry.details === 'object' ? JSON.stringify(entry.details) : entry.details,
        user: {
          name: entry.userName || 'Unknown',
          email: entry.userEmail || 'N/A',
          initials: (entry.userName || 'U').charAt(0).toUpperCase(),
          color: '#' + Math.floor(Math.random()*16777215).toString(16).padStart(6, '0'),
        }
      }));
      
      setEntries(prev => isRefresh ? transformedEntries : [...prev, ...transformedEntries]);
      setTotalEntries(data.total || 0);
      setHasMore(transformedEntries.length === LIMIT);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to fetch audit logs');
    } finally {
      setLoading(false);
    }
  }, [tenantId, datasourceId, startDate, endDate, sortBy, sortOrder]);

  // Initial load and filter changes
  useEffect(() => {
    setEntries([]);
    setOffset(0);
    setHasMore(true);
    fetchAuditLogs(0, true);
  }, [tenantId, datasourceId, startDate, endDate, sortBy, sortOrder, fetchAuditLogs]);

  // Infinite Scroll Observer
  useEffect(() => {
    const observer = new IntersectionObserver(
      entries => {
        if (entries[0].isIntersecting && hasMore && !loading) {
          const newOffset = offset + LIMIT;
          setOffset(newOffset);
          fetchAuditLogs(newOffset, false);
        }
      },
      { threshold: 0.1 }
    );

    if (observerTarget.current) {
      observer.observe(observerTarget.current);
    }

    return () => {
      if (observerTarget.current) {
        observer.unobserve(observerTarget.current);
      }
    };
  }, [hasMore, loading, offset, fetchAuditLogs]); // Depend on offset to trigger next page

  const handleRefresh = () => {
    setEntries([]);
    setOffset(0);
    setHasMore(true);
    fetchAuditLogs(0, true);
  };

  const handleSort = (property: string) => {
    const isAsc = sortBy === property && sortOrder === 'asc';
    setSortOrder(isAsc ? 'desc' : 'asc');
    setSortBy(property);
    // useEffect will trigger fetch
  };

  const filteredEntries = entries.filter(
    (entry: AuditLogEntry) =>
      entry.user.name.toLowerCase().includes(searchQuery.toLowerCase()) ||
      entry.resource.toLowerCase().includes(searchQuery.toLowerCase()) ||
      (entry.details || '').toLowerCase().includes(searchQuery.toLowerCase())
  );

  const handleViewDetails = (entry: AuditLogEntry) => {
    setSelectedEntry(entry);
    if (onViewDetails) {
      onViewDetails(entry);
    }
  };

  if (!tenantId || !datasourceId) {
    return (
      <Alert severity="warning">
        Please select a tenant and datasource to view audit logs
      </Alert>
    );
  }

  if (error) {
    return (
      <Alert severity="error">
        Error loading audit logs: {error}
      </Alert>
    );
  }

  return (
    <Box sx={{ display: 'flex', flexDirection: 'column', gap: 3, height: '100%' }}>
      {/* Header */}
      <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
        <Typography variant="h6" sx={{ fontWeight: 'bold' }}>
          Activity Log
        </Typography>
        <Button
          startIcon={<Refresh />}
          onClick={handleRefresh}
          disabled={loading}
          size="small"
        >
          Refresh
        </Button>
      </Box>

      {/* Search and Filter Controls */}
      <Box sx={{ display: 'flex', gap: 2, flexWrap: 'wrap', alignItems: 'center' }}>
        <TextField
          placeholder="Search by user, action, or resource..."
          value={searchQuery}
          onChange={(e) => setSearchQuery(e.target.value)}
          size="small"
          variant="outlined"
          InputProps={{
            startAdornment: <Search sx={{ mr: 1, color: 'action.active' }} />,
          }}
          sx={{ minWidth: 280 }}
        />
        
        <LocalizationProvider dateAdapter={AdapterDayjs}>
            <DatePicker 
                label="Start Date"
                value={startDate}
                onChange={(newValue: any) => setStartDate(newValue)}
                slotProps={{ textField: { size: 'small', sx: { width: 150 } } }}
            />
             <DatePicker 
                label="End Date"
                value={endDate}
                onChange={(newValue: any) => setEndDate(newValue)}
                slotProps={{ textField: { size: 'small', sx: { width: 150 } } }}
            />
        </LocalizationProvider>

        <Button
          variant="outlined"
          startIcon={<FilterList />}
          size="small"
        >
          Filter
        </Button>
        <Button
          variant="outlined"
          startIcon={<Download />}
          size="small"
          onClick={onExport}
          sx={{ ml: 'auto' }}
        >
          Export
        </Button>
      </Box>

      {/* Audit Log Table - Lazy Loading Container */}
      <TableContainer component={Paper} variant="outlined" sx={{ maxHeight: 'calc(100vh - 300px)', overflowY: 'auto' }}>
        <Table stickyHeader>
          <TableHead sx={{ backgroundColor: '#f5f5f5' }}>
            <TableRow>
              <TableCell sx={{ fontWeight: 'bold' }}>
                <TableSortLabel
                  active={sortBy === 'timestamp'}
                  direction={sortBy === 'timestamp' ? sortOrder : 'asc'}
                  onClick={() => handleSort('timestamp')}
                >
                  Timestamp
                </TableSortLabel>
              </TableCell>
              <TableCell sx={{ fontWeight: 'bold' }}>
                <TableSortLabel
                  active={sortBy === 'user_name'}
                  direction={sortBy === 'user_name' ? sortOrder : 'asc'}
                  onClick={() => handleSort('user_name')}
                >
                  User
                </TableSortLabel>
              </TableCell>
              <TableCell sx={{ fontWeight: 'bold' }}>
                <TableSortLabel
                  active={sortBy === 'action'}
                  direction={sortBy === 'action' ? sortOrder : 'asc'}
                  onClick={() => handleSort('action')}
                >
                  Action
                </TableSortLabel>
              </TableCell>
              <TableCell sx={{ fontWeight: 'bold' }}>
                <TableSortLabel
                  active={sortBy === 'resource'}
                  direction={sortBy === 'resource' ? sortOrder : 'asc'}
                  onClick={() => handleSort('resource')}
                >
                  Resource
                </TableSortLabel>
              </TableCell>
              <TableCell sx={{ fontWeight: 'bold' }}>Details</TableCell>
              <TableCell align="right" sx={{ fontWeight: 'bold' }}>
                Actions
              </TableCell>
            </TableRow>
          </TableHead>
          <TableBody>
            {entries.length === 0 && !loading ? (
              <TableRow>
                <TableCell colSpan={6} align="center" sx={{ py: 4 }}>
                  <Typography color="textSecondary">No audit entries found</Typography>
                </TableCell>
              </TableRow>
            ) : (
              filteredEntries.map((entry) => (
                <TableRow key={entry.id} hover>
                  <TableCell>
                    <Box>
                      <Typography variant="body2" sx={{ fontWeight: 'bold' }}>
                        {entry.timestamp.split(',')[0]}
                      </Typography>
                      <Typography variant="caption" sx={{ color: '#666' }}>
                        {entry.timestamp.split(',').slice(1).join(',')}
                      </Typography>
                    </Box>
                  </TableCell>
                  <TableCell>
                    <Box sx={{ display: 'flex', alignItems: 'center', gap: 2 }}>
                      <Avatar
                        sx={{
                          width: 32,
                          height: 32,
                          backgroundColor: entry.user.color,
                          fontSize: '0.75rem',
                          fontWeight: 'bold',
                        }}
                      >
                        {entry.user.initials}
                      </Avatar>
                      <Box>
                        <Typography variant="body2" sx={{ fontWeight: 'bold' }}>
                          {entry.user.name}
                        </Typography>
                        <Typography variant="caption" sx={{ color: '#666' }}>
                          {entry.user.email}
                        </Typography>
                      </Box>
                    </Box>
                  </TableCell>
                  <TableCell>
                    <Chip
                      label={getActionLabel(entry.action)}
                      color={getActionColor(entry.action) as any}
                      size="small"
                      variant="outlined"
                      icon={<Typography component="span">{getActionIcon(entry.action)}</Typography>}
                    />
                  </TableCell>
                  <TableCell>
                    <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                      <Typography variant="body2">{entry.resource}</Typography>
                      <Typography variant="caption" sx={{ color: '#999' }}>
                        ({entry.resourceType})
                      </Typography>
                    </Box>
                  </TableCell>
                  <TableCell>
                    <Typography variant="body2" sx={{ maxWidth: 300 }}>
                      {entry.details}
                    </Typography>
                  </TableCell>
                  <TableCell align="right">
                    <IconButton
                      size="small"
                      onClick={() => handleViewDetails(entry)}
                      title="View Details"
                    >
                      <Info fontSize="small" />
                    </IconButton>
                  </TableCell>
                </TableRow>
              ))
            )}
            {/* Loading Indicator / Sentinel */}
            <TableRow>
              <TableCell colSpan={6} sx={{ p: 0, border: 0 }}>
                 <Box ref={observerTarget} sx={{ display: 'flex', justifyContent: 'center', py: 2, visibility: hasMore ? 'visible' : 'hidden' }}>
                    {loading && <CircularProgress size={24} />}
                 </Box>
              </TableCell>
            </TableRow>
          </TableBody>
        </Table>
      </TableContainer>

      {/* Details Dialog */}
      <Dialog open={selectedEntry !== null} onClose={() => setSelectedEntry(null)} maxWidth="sm">
        <DialogTitle>Audit Entry Details</DialogTitle>
        <DialogContent dividers>
          {selectedEntry && (
            <Box sx={{ display: 'flex', flexDirection: 'column', gap: 2, py: 2 }}>
              <Box>
                <Typography variant="caption" color="textSecondary">
                  Timestamp
                </Typography>
                <Typography variant="body2">{selectedEntry.timestamp}</Typography>
              </Box>
              <Box>
                <Typography variant="caption" color="textSecondary">
                  User
                </Typography>
                <Typography variant="body2">
                  {selectedEntry.user.name} ({selectedEntry.user.email})
                </Typography>
              </Box>
              <Box>
                <Typography variant="caption" color="textSecondary">
                  Action
                </Typography>
                <Chip label={getActionLabel(selectedEntry.action)} size="small" />
              </Box>
              <Box>
                <Typography variant="caption" color="textSecondary">
                  Resource
                </Typography>
                <Typography variant="body2">
                  {selectedEntry.resource} ({selectedEntry.resourceType})
                </Typography>
              </Box>
              <Box>
                <Typography variant="caption" color="textSecondary">
                  Details
                </Typography>
                <Typography variant="body2">{selectedEntry.details}</Typography>
              </Box>
            </Box>
          )}
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setSelectedEntry(null)}>Close</Button>
        </DialogActions>
      </Dialog>
    </Box>
  );
};
