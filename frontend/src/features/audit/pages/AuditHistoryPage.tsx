import React, { useState, useEffect } from 'react';
import {
  Box,
  Container,
  Paper,
  Typography,
  Grid,
  Card,
  CardContent,
  FormControl,
  InputLabel,
  Select,
  MenuItem,
  TextField,
  Button,
  Chip,
  IconButton,
  Tooltip,
  CircularProgress,
  Alert,
  Tabs,
  Tab,
  Stack,
  Divider,
  Breadcrumbs,
  Link,
} from '@mui/material';
import {
  History as HistoryIcon,
  Restore as RestoreIcon,
  CompareArrows as CompareIcon,
  FilterList as FilterIcon,
  Refresh as RefreshIcon,
  Timeline as TimelineIcon,
  ViewList as ViewListIcon,
  ArrowBack as ArrowBackIcon,
} from '@mui/icons-material';
import { auditApi, EntitySnapshot, HistoryFilters } from '../../../api/auditApi';
import { format, parseISO } from 'date-fns';
import EntityTimelineView from '../components/EntityTimelineView';
import EntityHistoryTable from '../components/EntityHistoryTable';
import EntityDiffViewer from '../components/EntityDiffViewer';
import RestoreDialog from '../components/RestoreDialog';
import RecentChangesTable from '../components/RecentChangesTable';

type EntityType = 'tenant' | 'instance' | 'connection' | 'product' | 'all';
type ViewMode = 'timeline' | 'table';
type PageMode = 'list' | 'details';

const AuditHistoryPage: React.FC = () => {
  // State
  const [pageMode, setPageMode] = useState<PageMode>('list');
  const [entityType, setEntityType] = useState<EntityType>('all');
  const [entityId, setEntityId] = useState('');
  const [entityName, setEntityName] = useState('');
  const [viewMode, setViewMode] = useState<ViewMode>('timeline');
  const [history, setHistory] = useState<EntitySnapshot[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [selectedVersions, setSelectedVersions] = useState<EntitySnapshot[]>([]);
  const [restoreDialogOpen, setRestoreDialogOpen] = useState(false);
  const [selectedVersion, setSelectedVersion] = useState<EntitySnapshot | null>(null);

  // Filters
  const [filters, setFilters] = useState<HistoryFilters>({
    limit: 50,
    includeDeleted: false,
  });
  
  // Default to last 7 days
  const getDefaultDateFrom = () => {
    const date = new Date();
    date.setDate(date.getDate() - 7);
    return date.toISOString().slice(0, 16); // Format for datetime-local input
  };
  
  const getDefaultDateTo = () => {
    return new Date().toISOString().slice(0, 16); // Format for datetime-local input
  };
  
  const [dateFrom, setDateFrom] = useState(getDefaultDateFrom());
  const [dateTo, setDateTo] = useState(getDefaultDateTo());

  // Load history
  const loadHistory = async () => {
    if (!entityId) {
      setError('Please enter an entity ID');
      return;
    }

    setLoading(true);
    setError(null);

    try {
      const filterParams: HistoryFilters = {
        ...filters,
        from: dateFrom ? new Date(dateFrom).toISOString() : undefined,
        to: dateTo ? new Date(dateTo).toISOString() : undefined,
      };

      const response = await auditApi.getEntityHistory(entityType, entityId, filterParams);
      setHistory(response.history);
    } catch (err: any) {
      setError(err.response?.data?.message || 'Failed to load history');
    } finally {
      setLoading(false);
    }
  };

  // Handle version selection for comparison
  const handleVersionSelect = (version: EntitySnapshot) => {
    if (selectedVersions.find((v) => v.version_id === version.version_id)) {
      setSelectedVersions(selectedVersions.filter((v) => v.version_id !== version.version_id));
    } else if (selectedVersions.length < 2) {
      setSelectedVersions([...selectedVersions, version]);
    } else {
      // Replace oldest selection
      setSelectedVersions([selectedVersions[1], version]);
    }
  };

  // Handle restore
  const handleRestoreClick = (version: EntitySnapshot) => {
    setSelectedVersion(version);
    setRestoreDialogOpen(true);
  };

  const handleRestoreConfirm = async (reason: string) => {
    if (!selectedVersion) return;

    try {
      await auditApi.restoreEntity(entityType as any, entityId, {
        restoreToTime: selectedVersion.system_from,
        reason,
      });
      setRestoreDialogOpen(false);
      loadHistory(); // Reload to show the restore event
    } catch (err: any) {
      setError(err.response?.data?.message || 'Failed to restore entity');
    }
  };

  // Handle entity selection from list
  const handleEntitySelect = (type: string, id: string, name?: string) => {
    setEntityType(type as EntityType);
    setEntityId(id);
    setEntityName(name || '');
    setPageMode('details');
    
    // Auto-load history for selected entity
    setTimeout(() => {
      const filterParams: HistoryFilters = {
        limit: 50,
        includeDeleted: false,
        from: dateFrom ? new Date(dateFrom).toISOString() : undefined,
        to: dateTo ? new Date(dateTo).toISOString() : undefined,
      };
      
      setLoading(true);
      setError(null);
      
      auditApi.getEntityHistory(type, id, filterParams)
        .then(response => {
          setHistory(response.history);
          setLoading(false);
        })
        .catch(err => {
          setError(err.response?.data?.message || 'Failed to load history');
          setLoading(false);
        });
    }, 100);
  };

  // Handle back to list
  const handleBackToList = () => {
    setPageMode('list');
    setHistory([]);
    setSelectedVersions([]);
    setEntityId('');
    setEntityName('');
  };

  return (
    <Container maxWidth="xl" sx={{ py: 4 }}>
      {/* Header */}
      <Box sx={{ mb: 4 }}>
        {/* Breadcrumbs */}
        {pageMode === 'details' && (
          <Breadcrumbs sx={{ mb: 2 }}>
            <Link
              component="button"
              variant="body2"
              onClick={handleBackToList}
              sx={{ display: 'flex', alignItems: 'center', gap: 0.5, cursor: 'pointer' }}
            >
              <ArrowBackIcon fontSize="small" />
              Recent Changes
            </Link>
            <Typography variant="body2" color="text.primary">
              {entityName || entityId}
            </Typography>
          </Breadcrumbs>
        )}

        <Stack direction="row" alignItems="center" spacing={2} sx={{ mb: 2 }}>
          <HistoryIcon sx={{ fontSize: 40, color: 'primary.main' }} />
          <Box>
            <Typography variant="h4" sx={{ fontWeight: 'bold' }}>
              Audit History
            </Typography>
            <Typography variant="body2" color="text.secondary">
              {pageMode === 'list'
                ? 'Browse entities with recent changes'
                : 'View and restore historical entity states'}
            </Typography>
          </Box>
        </Stack>
      </Box>

      {/* Filters */}
      <Paper sx={{ p: 3, mb: 3 }}>
        <Stack direction="row" alignItems="center" spacing={2} sx={{ mb: 3 }}>
          <FilterIcon />
          <Typography variant="h6">Filters</Typography>
        </Stack>

        <Grid container spacing={2}>
          <Grid item xs={12} md={pageMode === 'list' ? 4 : 3}>
            <FormControl fullWidth>
              <InputLabel>Entity Type</InputLabel>
              <Select
                value={entityType}
                label="Entity Type"
                onChange={(e) => setEntityType(e.target.value as EntityType)}
              >
                {pageMode === 'list' && <MenuItem value="all">All Types</MenuItem>}
                <MenuItem value="tenant">Tenant</MenuItem>
                <MenuItem value="instance">Instance</MenuItem>
                <MenuItem value="connection">Connection</MenuItem>
                <MenuItem value="product">Product</MenuItem>
              </Select>
            </FormControl>
          </Grid>

          {pageMode === 'details' && (
            <Grid item xs={12} md={3}>
              <TextField
                fullWidth
                label="Entity ID"
                value={entityId}
                onChange={(e) => setEntityId(e.target.value)}
                placeholder="Enter UUID"
                disabled
              />
            </Grid>
          )}

          <Grid item xs={12} md={pageMode === 'list' ? 3 : 2}>
            <TextField
              fullWidth
              label="From Date"
              type="datetime-local"
              value={dateFrom}
              onChange={(e) => setDateFrom(e.target.value)}
              InputLabelProps={{ shrink: true }}
            />
          </Grid>

          <Grid item xs={12} md={pageMode === 'list' ? 3 : 2}>
            <TextField
              fullWidth
              label="To Date"
              type="datetime-local"
              value={dateTo}
              onChange={(e) => setDateTo(e.target.value)}
              InputLabelProps={{ shrink: true }}
            />
          </Grid>

          {pageMode === 'details' && (
            <Grid item xs={12} md={2}>
              <Stack direction="row" spacing={1} sx={{ height: '100%' }}>
                <Button
                  fullWidth
                  variant="contained"
                  onClick={loadHistory}
                  disabled={loading || !entityId}
                  startIcon={loading ? <CircularProgress size={20} /> : <HistoryIcon />}
                >
                  Load History
                </Button>
                <Tooltip title="Refresh">
                  <IconButton onClick={loadHistory} disabled={loading || !entityId}>
                    <RefreshIcon />
                  </IconButton>
                </Tooltip>
              </Stack>
            </Grid>
          )}
        </Grid>
      </Paper>

      {/* Error Alert */}
      {error && (
        <Alert severity="error" sx={{ mb: 3 }} onClose={() => setError(null)}>
          {error}
        </Alert>
      )}

      {/* List View - Recent Changes */}
      {pageMode === 'list' && (
        <RecentChangesTable
          dateFrom={dateFrom}
          dateTo={dateTo}
          entityType={entityType === 'all' ? '' : entityType}
          onEntitySelect={handleEntitySelect}
        />
      )}

      {/* Details View - Entity History */}
      {pageMode === 'details' && history.length > 0 && (
        <>
          {/* View Mode Selector & Stats */}
          <Paper sx={{ p: 2, mb: 3 }}>
            <Stack direction="row" justifyContent="space-between" alignItems="center">
              <Stack direction="row" spacing={2} alignItems="center">
                <Chip
                  label={`${history.length} versions`}
                  color="primary"
                  variant="outlined"
                />
                <Chip
                  label={`Current: ${history.find((h) => h.is_current)?.change_type || 'N/A'}`}
                  color="success"
                />
                {selectedVersions.length > 0 && (
                  <Chip
                    label={`${selectedVersions.length} selected for comparison`}
                    color="secondary"
                    onDelete={() => setSelectedVersions([])}
                  />
                )}
              </Stack>

              <Tabs value={viewMode} onChange={(_, v) => setViewMode(v)}>
                <Tab
                  value="timeline"
                  icon={<TimelineIcon />}
                  label="Timeline"
                  iconPosition="start"
                />
                <Tab value="table" icon={<ViewListIcon />} label="Table" iconPosition="start" />
              </Tabs>
            </Stack>
          </Paper>

          {/* Timeline or Table View */}
          {viewMode === 'timeline' ? (
            <EntityTimelineView
              history={history}
              onVersionSelect={handleVersionSelect}
              onRestoreClick={handleRestoreClick}
              selectedVersions={selectedVersions}
            />
          ) : (
            <EntityHistoryTable
              history={history}
              onVersionSelect={handleVersionSelect}
              onRestoreClick={handleRestoreClick}
              selectedVersions={selectedVersions}
            />
          )}

          {/* Diff Viewer */}
          {selectedVersions.length === 2 && (
            <Box sx={{ mt: 3 }}>
              <EntityDiffViewer
                leftVersion={selectedVersions[0]}
                rightVersion={selectedVersions[1]}
              />
            </Box>
          )}
        </>
      )}

      {/* Empty State */}
      {!loading && history.length === 0 && entityId && (
        <Paper sx={{ p: 6, textAlign: 'center' }}>
          <HistoryIcon sx={{ fontSize: 80, color: 'text.disabled', mb: 2 }} />
          <Typography variant="h6" color="text.secondary">
            No history found for this entity
          </Typography>
          <Typography variant="body2" color="text.secondary">
            Try adjusting your filters or check the entity ID
          </Typography>
        </Paper>
      )}

      {/* Restore Dialog */}
      <RestoreDialog
        open={restoreDialogOpen}
        version={selectedVersion}
        onClose={() => setRestoreDialogOpen(false)}
        onConfirm={handleRestoreConfirm}
      />
    </Container>
  );
};

export default AuditHistoryPage;
