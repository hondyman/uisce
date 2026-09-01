import React, { useState, useMemo, useEffect, useCallback } from 'react';
import { useNavigate } from 'react-router-dom';
import {
  Box,
  Card,
  CardContent,
  Typography,
  Button,
  Grid,
  List,
  ListItem,
  ListItemText,
  ListItemIcon,
  ListItemButton,
  IconButton,
  TextField,
  InputAdornment,
  Chip,
  Menu,
  MenuItem,
  Dialog,
  DialogTitle,
  DialogContent,
  DialogContentText,
  DialogActions,
  Breadcrumbs,
  Link,
  Paper,
  Divider,
  Avatar,
  Tooltip,
  Badge,
  Stack,
  ToggleButton,
  ToggleButtonGroup,
  CircularProgress,
  Tabs,
  Tab,
  Snackbar,
  Alert,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  AlertTitle,
} from '@mui/material';
import {
  Add as AddIcon,
  Folder as FolderIcon,
  FolderOpen as FolderOpenIcon,
  Storage as QueryIcon,
  Search as SearchIcon,
  MoreVert as MoreIcon,
  Star as StarIcon,
  StarBorder as StarBorderIcon,
  Share as ShareIcon,
  Edit as EditIcon,
  Delete as DeleteIcon,
  FileCopy as DuplicateIcon,
  Refresh as RefreshIcon,
  ViewList as ListViewIcon,
  ViewModule as GridViewIcon,
  AccessTime as RecentIcon,
  Person as PersonIcon,
  Group as GroupIcon,
  Public as PublicIcon,
  VerifiedUser as CoreIcon,
  DriveFileMove as MoveIcon,
  PlayArrow as PlayIcon,
  Close as CloseIcon,
  AutoAwesome as SparklesIcon,
  FilterList as FilterIcon,
  TableChart as TableIcon,
} from '@mui/icons-material';
import { formatDistanceToNow } from 'date-fns';
import { useTenant } from '../../../contexts/TenantContext';
import type { SavedExplorerQuery, ExplorerQueryState } from '../types/dataExplorerTypes';
import { fetchBusinessObjects } from '../services/dataExplorerApi';
import type { BusinessObjectSummary } from '../types/dataExplorerTypes';

export interface QueryFolder {
  id: string;
  name: string;
  description?: string;
  parent_id?: string;
  is_core: boolean;
  created_by: string;
  query_count: number;
  created_at?: string;
}

export interface LibraryQueryItem extends SavedExplorerQuery {
  description?: string;
  folder_id?: string;
  is_favorite?: boolean;
  is_shared?: boolean;
  share_type?: 'private' | 'shared_by_me' | 'shared_with_me' | 'team' | 'public';
  shared_by?: string;
  is_core?: boolean;
  run_count?: number;
  last_run?: string;
  tags?: string[];
}

const STORAGE_KEY_CUSTOM_FOLDERS = 'uisce_data_explorer_folders_v1';
const STORAGE_KEY_SAVED_QUERIES = 'uisce_data_explorer_queries_v1';

const INITIAL_CORE_FOLDERS: QueryFolder[] = [
  {
    id: 'folder-core-oms',
    name: 'Order Management & Trading',
    description: 'Gold copy standard queries for Accounts, Positions, Securities, and Trade Orders',
    is_core: true,
    created_by: 'Master Tenant',
    query_count: 3,
    created_at: new Date('2026-01-01').toISOString(),
  },
  {
    id: 'folder-core-altinv',
    name: 'Alternative Investments',
    description: 'Private equity, venture capital, hedge fund commitments and NAVs',
    is_core: true,
    created_by: 'Master Tenant',
    query_count: 2,
    created_at: new Date('2026-01-01').toISOString(),
  },
  {
    id: 'folder-core-master',
    name: 'Master Directory & Sales',
    description: 'Client, Vendor, Personnel, and Fee Ledger queries',
    is_core: true,
    created_by: 'Master Tenant',
    query_count: 2,
    created_at: new Date('2026-01-01').toISOString(),
  },
];

const INITIAL_SEED_QUERIES: LibraryQueryItem[] = [
  {
    id: 'query-core-001',
    name: 'Institutional Accounts by Sponsor',
    description: 'Aggregated account valuation and mandate types grouped by sponsor ID.',
    sourceKind: 'business_object',
    sourceId: 'oms.account',
    folder_id: 'folder-core-oms',
    is_core: true,
    is_favorite: true,
    is_shared: true,
    share_type: 'shared_by_me',
    run_count: 54,
    last_run: new Date(Date.now() - 3600000 * 2).toISOString(),
    tags: ['OMS', 'Accounts', 'Sponsor'],
    queryState: {
      sourceId: 'oms.account',
      bindingId: 'default-binding',
      dimensions: [{ fieldId: 'sponsor_id' }, { fieldId: 'mandate_type' }],
      measures: [{ fieldId: 'aum_basis_amount', agg: 'SUM' }],
      timeDimensions: [],
      calculations: [],
      parameters: [],
      filters: [{ fieldId: 'subtype_code', operator: 'equals', values: ['institutional'] }],
      sorts: [{ fieldId: 'aum_basis_amount', direction: 'desc' }],
      limit: 1000,
    },
    createdBy: 'Master Tenant',
    createdAt: new Date(Date.now() - 86400000 * 20).toISOString(),
    updatedAt: new Date(Date.now() - 86400000 * 2).toISOString(),
  },
  {
    id: 'query-core-002',
    name: 'Daily Settled Long Positions & Exposure',
    description: 'Market value and quantity for settled long positions across prime brokers.',
    sourceKind: 'business_object',
    sourceId: 'oms.position',
    folder_id: 'folder-core-oms',
    is_core: true,
    is_favorite: false,
    is_shared: false,
    share_type: 'private',
    run_count: 31,
    last_run: new Date(Date.now() - 3600000 * 8).toISOString(),
    tags: ['Positions', 'Settled', 'Exposure'],
    queryState: {
      sourceId: 'oms.position',
      bindingId: 'default-binding',
      dimensions: [{ fieldId: 'security_id' }, { fieldId: 'custodian_id' }],
      measures: [{ fieldId: 'quantity', agg: 'SUM' }, { fieldId: 'market_value', agg: 'SUM' }],
      timeDimensions: [],
      calculations: [],
      parameters: [],
      filters: [{ fieldId: 'subtype_code', operator: 'equals', values: ['settled_long'] }],
      sorts: [{ fieldId: 'market_value', direction: 'desc' }],
      limit: 500,
    },
    createdBy: 'Master Tenant',
    createdAt: new Date(Date.now() - 86400000 * 15).toISOString(),
    updatedAt: new Date(Date.now() - 86400000 * 3).toISOString(),
  },
  {
    id: 'query-core-003',
    name: 'Private Equity Capital Calls & Commitments',
    description: 'Total commitment amount and unfunded capital by fund manager and vintage year.',
    sourceKind: 'business_object',
    sourceId: 'altinv.alternative_investment',
    folder_id: 'folder-core-altinv',
    is_core: true,
    is_favorite: true,
    is_shared: true,
    share_type: 'public',
    run_count: 42,
    last_run: new Date(Date.now() - 3600000 * 24).toISOString(),
    tags: ['PrivateEquity', 'CapitalCalls', 'NAV'],
    queryState: {
      sourceId: 'altinv.alternative_investment',
      bindingId: 'default-binding',
      dimensions: [{ fieldId: 'fund_manager' }, { fieldId: 'vintage_year' }],
      measures: [{ fieldId: 'total_commitment', agg: 'SUM' }, { fieldId: 'unfunded_commitment', agg: 'SUM' }],
      timeDimensions: [],
      calculations: [],
      parameters: [],
      filters: [{ fieldId: 'subtype_code', operator: 'equals', values: ['private_equity'] }],
      sorts: [{ fieldId: 'vintage_year', direction: 'desc' }],
      limit: 1000,
    },
    createdBy: 'Master Tenant',
    createdAt: new Date(Date.now() - 86400000 * 30).toISOString(),
    updatedAt: new Date(Date.now() - 86400000 * 5).toISOString(),
  },
];

interface QueryLibraryProps {
  onOpenQuery?: (query: LibraryQueryItem) => void;
  onCreateNew?: () => void;
}

export const QueryLibrary: React.FC<QueryLibraryProps> = ({ onOpenQuery, onCreateNew }) => {
  const navigate = useNavigate();
  const { currentTenant } = useTenant();

  const [activeTab, setActiveTab] = useState<'all' | 'core' | 'custom' | 'favorites' | 'shared'>('all');
  const [selectedFolderId, setSelectedFolderId] = useState<string | null>(null);
  const [searchQuery, setSearchQuery] = useState('');
  const [viewMode, setViewMode] = useState<'grid' | 'list'>('grid');

  // Persistence
  const [folders, setFolders] = useState<QueryFolder[]>(() => {
    const saved = localStorage.getItem(STORAGE_KEY_CUSTOM_FOLDERS);
    return saved ? JSON.parse(saved) : INITIAL_CORE_FOLDERS;
  });

  const [queries, setQueries] = useState<LibraryQueryItem[]>(() => {
    const saved = localStorage.getItem(STORAGE_KEY_SAVED_QUERIES);
    return saved ? JSON.parse(saved) : INITIAL_SEED_QUERIES;
  });

  const [businessObjects, setBusinessObjects] = useState<BusinessObjectSummary[]>([]);

  useEffect(() => {
    fetchBusinessObjects().then(setBusinessObjects).catch(() => {});
  }, []);

  const saveQueries = useCallback((updated: LibraryQueryItem[]) => {
    setQueries(updated);
    localStorage.setItem(STORAGE_KEY_SAVED_QUERIES, JSON.stringify(updated));
  }, []);

  const saveFolders = useCallback((updated: QueryFolder[]) => {
    setFolders(updated);
    localStorage.setItem(STORAGE_KEY_CUSTOM_FOLDERS, JSON.stringify(updated));
  }, []);

  // Menu action state
  const [menuAnchorEl, setMenuAnchorEl] = useState<null | HTMLElement>(null);
  const [selectedQuery, setSelectedQuery] = useState<LibraryQueryItem | null>(null);

  // Modals
  const [isFolderDialogOpen, setIsFolderDialogOpen] = useState(false);
  const [folderName, setFolderName] = useState('');
  const [folderDesc, setFolderDesc] = useState('');
  const [moveDialogOpen, setMoveDialogOpen] = useState(false);
  const [targetFolderId, setTargetFolderId] = useState<string>('');

  // Toast
  const [snackbar, setSnackbar] = useState<{ open: boolean; message: string; severity: 'success' | 'info' | 'error' }>({
    open: false,
    message: '',
    severity: 'success',
  });

  const handleOpenPlayground = (query?: LibraryQueryItem) => {
    if (onOpenQuery && query) {
      onOpenQuery(query);
    } else if (onCreateNew) {
      onCreateNew();
    } else {
      navigate('/data-explorer/builder', { state: { query } });
    }
  };

  const handleToggleFavorite = (queryId: string, e: React.MouseEvent) => {
    e.stopPropagation();
    const updated = queries.map((q) => (q.id === queryId ? { ...q, is_favorite: !q.is_favorite } : q));
    saveQueries(updated);
    setSnackbar({ open: true, message: 'Favorites updated', severity: 'success' });
  };

  const handleDuplicate = (query: LibraryQueryItem) => {
    const duplicated: LibraryQueryItem = {
      ...query,
      id: `query-custom-${Date.now()}`,
      name: `${query.name} (Copy)`,
      is_core: false,
      createdBy: currentTenant?.tenant_name || 'Current User',
      createdAt: new Date().toISOString(),
      updatedAt: new Date().toISOString(),
      run_count: 0,
      last_run: undefined,
    };
    saveQueries([duplicated, ...queries]);
    setSnackbar({ open: true, message: `Duplicated "${query.name}"`, severity: 'success' });
    setMenuAnchorEl(null);
  };

  const handleDelete = (queryId: string) => {
    const updated = queries.filter((q) => q.id !== queryId);
    saveQueries(updated);
    setSnackbar({ open: true, message: 'Query deleted', severity: 'info' });
    setMenuAnchorEl(null);
  };

  const handleCreateFolder = () => {
    if (!folderName.trim()) return;
    const newFolder: QueryFolder = {
      id: `folder-custom-${Date.now()}`,
      name: folderName.trim(),
      description: folderDesc.trim(),
      is_core: false,
      created_by: currentTenant?.tenant_name || 'Current User',
      query_count: 0,
      created_at: new Date().toISOString(),
    };
    saveFolders([...folders, newFolder]);
    setFolderName('');
    setFolderDesc('');
    setIsFolderDialogOpen(false);
    setSnackbar({ open: true, message: `Folder "${newFolder.name}" created`, severity: 'success' });
  };

  const handleMoveQuery = () => {
    if (!selectedQuery) return;
    const updated = queries.map((q) =>
      q.id === selectedQuery.id ? { ...q, folder_id: targetFolderId || undefined } : q
    );
    saveQueries(updated);
    setMoveDialogOpen(false);
    setMenuAnchorEl(null);
    setSnackbar({ open: true, message: 'Query moved successfully', severity: 'success' });
  };

  // Filtered queries
  const filteredQueries = useMemo(() => {
    return queries.filter((q) => {
      // Tab filter
      if (activeTab === 'core' && !q.is_core) return false;
      if (activeTab === 'custom' && q.is_core) return false;
      if (activeTab === 'favorites' && !q.is_favorite) return false;
      if (activeTab === 'shared' && !q.is_shared) return false;

      // Folder filter
      if (selectedFolderId && q.folder_id !== selectedFolderId) return false;

      // Search query
      if (searchQuery.trim()) {
        const term = searchQuery.toLowerCase();
        const matchesName = q.name.toLowerCase().includes(term);
        const matchesDesc = q.description?.toLowerCase().includes(term);
        const matchesTags = q.tags?.some((t) => t.toLowerCase().includes(term));
        const matchesSource = q.sourceId.toLowerCase().includes(term);
        if (!matchesName && !matchesDesc && !matchesTags && !matchesSource) return false;
      }

      return true;
    });
  }, [queries, activeTab, selectedFolderId, searchQuery]);

  return (
    <Box sx={{ display: 'flex', height: '100%', bgcolor: 'background.default' }}>
      {/* Left Sidebar: Folders & Filters */}
      <Paper
        square
        elevation={0}
        sx={{
          width: 280,
          borderRight: 1,
          borderColor: 'divider',
          display: 'flex',
          flexDirection: 'column',
          bgcolor: 'background.paper',
        }}
      >
        <Box sx={{ p: 2, pb: 1, display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
          <Typography variant="subtitle2" sx={{ fontWeight: 700, textTransform: 'uppercase', letterSpacing: 0.5 }}>
            Query Folders
          </Typography>
          <Tooltip title="Create Folder">
            <IconButton size="small" onClick={() => setIsFolderDialogOpen(true)}>
              <AddIcon fontSize="small" />
            </IconButton>
          </Tooltip>
        </Box>

        <List dense sx={{ px: 1, flex: 1, overflowY: 'auto' }}>
          <ListItemButton
            selected={selectedFolderId === null}
            onClick={() => setSelectedFolderId(null)}
            sx={{ borderRadius: 1.5, mb: 0.5 }}
          >
            <ListItemIcon sx={{ minWidth: 32 }}>
              <FolderOpenIcon fontSize="small" color={selectedFolderId === null ? 'primary' : 'inherit'} />
            </ListItemIcon>
            <ListItemText primary="All Queries" />
            <Chip size="small" label={queries.length} variant="outlined" sx={{ height: 20, fontSize: 11 }} />
          </ListItemButton>

          <Divider sx={{ my: 1 }} />

          <Typography variant="caption" sx={{ px: 1.5, py: 0.5, color: 'text.secondary', fontWeight: 600 }}>
            CORE FOLDERS
          </Typography>
          {folders
            .filter((f) => f.is_core)
            .map((folder) => {
              const count = queries.filter((q) => q.folder_id === folder.id).length;
              return (
                <ListItemButton
                  key={folder.id}
                  selected={selectedFolderId === folder.id}
                  onClick={() => setSelectedFolderId(folder.id)}
                  sx={{ borderRadius: 1.5, mb: 0.5 }}
                >
                  <ListItemIcon sx={{ minWidth: 32 }}>
                    <FolderIcon fontSize="small" sx={{ color: 'primary.main' }} />
                  </ListItemIcon>
                  <ListItemText
                    primary={folder.name}
                    primaryTypographyProps={{ variant: 'body2', noWrap: true }}
                  />
                  <Chip size="small" label={count} sx={{ height: 18, fontSize: 10 }} />
                </ListItemButton>
              );
            })}

          <Divider sx={{ my: 1 }} />

          <Typography variant="caption" sx={{ px: 1.5, py: 0.5, color: 'text.secondary', fontWeight: 600 }}>
            CUSTOM FOLDERS
          </Typography>
          {folders
            .filter((f) => !f.is_core)
            .map((folder) => {
              const count = queries.filter((q) => q.folder_id === folder.id).length;
              return (
                <ListItemButton
                  key={folder.id}
                  selected={selectedFolderId === folder.id}
                  onClick={() => setSelectedFolderId(folder.id)}
                  sx={{ borderRadius: 1.5, mb: 0.5 }}
                >
                  <ListItemIcon sx={{ minWidth: 32 }}>
                    <FolderIcon fontSize="small" sx={{ color: 'warning.main' }} />
                  </ListItemIcon>
                  <ListItemText
                    primary={folder.name}
                    primaryTypographyProps={{ variant: 'body2', noWrap: true }}
                  />
                  <Chip size="small" label={count} sx={{ height: 18, fontSize: 10 }} />
                </ListItemButton>
              );
            })}
        </List>
      </Paper>

      {/* Main Content Pane */}
      <Box sx={{ flex: 1, display: 'flex', flexDirection: 'column', overflow: 'hidden' }}>
        {/* Top Header Bar */}
        <Box
          sx={{
            p: 2.5,
            borderBottom: 1,
            borderColor: 'divider',
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'space-between',
            bgcolor: 'background.paper',
          }}
        >
          <Box>
            <Stack direction="row" alignItems="center" spacing={1}>
              <Typography variant="h5" sx={{ fontWeight: 700 }}>
                Data Explorer
              </Typography>
              <Chip
                icon={<CoreIcon sx={{ fontSize: '14px !important' }} />}
                label={currentTenant?.gold_copy ? 'Master Tenant (Core)' : `${currentTenant?.tenant_name || 'Client'} (Custom)`}
                size="small"
                color={currentTenant?.gold_copy ? 'primary' : 'secondary'}
                variant="outlined"
              />
            </Stack>
            <Typography variant="body2" color="text.secondary" sx={{ mt: 0.5 }}>
              Cube Playground self-service data queries, semantic modeling, and live result inspection.
            </Typography>
          </Box>

          <Stack direction="row" spacing={1.5} alignItems="center">
            <Button
              variant="outlined"
              color="inherit"
              startIcon={<SparklesIcon sx={{ color: '#0D9488' }} />}
              onClick={() => navigate('/ai-explorer')}
              sx={{ textTransform: 'none', borderRadius: 2 }}
            >
              Ask AI Co-Pilot
            </Button>
            <Button
              variant="contained"
              color="primary"
              startIcon={<AddIcon />}
              onClick={() => handleOpenPlayground()}
              sx={{ textTransform: 'none', fontWeight: 600, borderRadius: 2 }}
            >
              New Query
            </Button>
          </Stack>
        </Box>

        {/* Filter Toolbar & Search */}
        <Box
          sx={{
            px: 2.5,
            py: 1.5,
            borderBottom: 1,
            borderColor: 'divider',
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'space-between',
            gap: 2,
            bgcolor: 'background.paper',
          }}
        >
          <Tabs
            value={activeTab}
            onChange={(_, val) => setActiveTab(val)}
            textColor="primary"
            indicatorColor="primary"
            sx={{ minHeight: 36, '& .MuiTab-root': { minHeight: 36, py: 0.5, textTransform: 'none', fontWeight: 600 } }}
          >
            <Tab label="All Queries" value="all" />
            <Tab label="Core (Gold Copy)" value="core" />
            <Tab label="Custom" value="custom" />
            <Tab label="Favorites" value="favorites" />
            <Tab label="Shared With Me" value="shared" />
          </Tabs>

          <Stack direction="row" spacing={1.5} alignItems="center">
            <TextField
              size="small"
              placeholder="Search queries, tags, or sources..."
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
              InputProps={{
                startAdornment: (
                  <InputAdornment position="start">
                    <SearchIcon fontSize="small" sx={{ color: 'text.secondary' }} />
                  </InputAdornment>
                ),
                sx: { width: 280, borderRadius: 2, bgcolor: 'background.default' },
              }}
            />

            <ToggleButtonGroup
              size="small"
              value={viewMode}
              exclusive
              onChange={(_, val) => val && setViewMode(val)}
            >
              <ToggleButton value="grid">
                <GridViewIcon fontSize="small" />
              </ToggleButton>
              <ToggleButton value="list">
                <ListViewIcon fontSize="small" />
              </ToggleButton>
            </ToggleButtonGroup>
          </Stack>
        </Box>

        {/* Query Items Container */}
        <Box sx={{ flex: 1, p: 3, overflowY: 'auto' }}>
          {filteredQueries.length === 0 ? (
            <Box
              sx={{
                p: 6,
                textAlign: 'center',
                bgcolor: 'background.paper',
                borderRadius: 3,
                border: '1px dashed',
                borderColor: 'divider',
              }}
            >
              <QueryIcon sx={{ fontSize: 48, color: 'text.disabled', mb: 1.5 }} />
              <Typography variant="h6" color="text.secondary">
                No Queries Found
              </Typography>
              <Typography variant="body2" color="text.disabled" sx={{ mb: 2 }}>
                {searchQuery
                  ? 'Try adjusting your search criteria or filter tags.'
                  : 'Get started by creating your first self-service query.'}
              </Typography>
              <Button
                variant="contained"
                startIcon={<AddIcon />}
                onClick={() => handleOpenPlayground()}
                sx={{ textTransform: 'none', borderRadius: 2 }}
              >
                Create New Query
              </Button>
            </Box>
          ) : viewMode === 'grid' ? (
            <Grid container spacing={2.5}>
              {filteredQueries.map((query) => (
                <Grid    key={query.id} size={{ xs: 12, sm: 6, md: 4 }}>
                  <Card
                    elevation={0}
                    sx={{
                      borderRadius: 2.5,
                      border: 1,
                      borderColor: 'divider',
                      transition: 'all 0.2s ease-in-out',
                      cursor: 'pointer',
                      display: 'flex',
                      flexDirection: 'column',
                      height: '100%',
                      '&:hover': {
                        borderColor: 'primary.main',
                        boxShadow: '0 4px 20px rgba(0,0,0,0.06)',
                        transform: 'translateY(-2px)',
                      },
                    }}
                    onClick={() => handleOpenPlayground(query)}
                  >
                    <CardContent sx={{ p: 2.5, flex: 1, display: 'flex', flexDirection: 'column' }}>
                      <Box sx={{ display: 'flex', alignItems: 'flex-start', justifyContent: 'space-between', mb: 1 }}>
                        <Stack direction="row" spacing={1} alignItems="center">
                          <Avatar
                            sx={{
                              width: 32,
                              height: 32,
                              bgcolor: query.is_core ? 'primary.50' : 'warning.50',
                              color: query.is_core ? 'primary.main' : 'warning.main',
                              fontSize: 14,
                            }}
                          >
                            <TableIcon fontSize="small" />
                          </Avatar>
                          <Box>
                            <Typography variant="subtitle1" sx={{ fontWeight: 700, lineHeight: 1.2 }}>
                              {query.name}
                            </Typography>
                            <Typography variant="caption" color="text.secondary">
                              Source: {query.sourceId}
                            </Typography>
                          </Box>
                        </Stack>

                        <Stack direction="row" spacing={0.5}>
                          <IconButton
                            size="small"
                            onClick={(e) => handleToggleFavorite(query.id, e)}
                            color={query.is_favorite ? 'warning' : 'default'}
                          >
                            {query.is_favorite ? <StarIcon fontSize="small" /> : <StarBorderIcon fontSize="small" />}
                          </IconButton>
                          <IconButton
                            size="small"
                            onClick={(e) => {
                              e.stopPropagation();
                              setSelectedQuery(query);
                              setMenuAnchorEl(e.currentTarget);
                            }}
                          >
                            <MoreIcon fontSize="small" />
                          </IconButton>
                        </Stack>
                      </Box>

                      <Typography
                        variant="body2"
                        color="text.secondary"
                        sx={{
                          mb: 2,
                          flex: 1,
                          display: '-webkit-box',
                          WebkitLineClamp: 2,
                          WebkitBoxOrient: 'vertical',
                          overflow: 'hidden',
                        }}
                      >
                        {query.description || 'Self-service semantic query.'}
                      </Typography>

                      {/* Dimensions & Measures badges */}
                      <Stack direction="row" spacing={0.8} sx={{ mb: 2, flexWrap: 'wrap', gap: 0.5 }}>
                        <Chip
                          size="small"
                          label={`${query.queryState.dimensions.length} dims`}
                          variant="outlined"
                          sx={{ fontSize: 11, height: 22 }}
                        />
                        <Chip
                          size="small"
                          label={`${query.queryState.measures.length} measures`}
                          variant="outlined"
                          sx={{ fontSize: 11, height: 22 }}
                        />
                        {query.is_core && (
                          <Chip
                            size="small"
                            label="Core"
                            color="primary"
                            sx={{ fontSize: 11, height: 22 }}
                          />
                        )}
                        {query.tags?.map((t) => (
                          <Chip
                            key={t}
                            size="small"
                            label={t}
                            sx={{ fontSize: 11, height: 22, bgcolor: 'background.default' }}
                          />
                        ))}
                      </Stack>

                      <Divider sx={{ my: 1 }} />

                      <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', pt: 0.5 }}>
                        <Typography variant="caption" color="text.disabled">
                          {query.last_run
                            ? `Ran ${formatDistanceToNow(new Date(query.last_run))} ago`
                            : `Created by ${query.createdBy || 'User'}`}
                        </Typography>
                        <Button
                          size="small"
                          variant="text"
                          startIcon={<PlayIcon fontSize="small" />}
                          onClick={(e) => {
                            e.stopPropagation();
                            handleOpenPlayground(query);
                          }}
                          sx={{ textTransform: 'none', fontWeight: 600 }}
                        >
                          Run
                        </Button>
                      </Box>
                    </CardContent>
                  </Card>
                </Grid>
              ))}
            </Grid>
          ) : (
            <TableContainer component={Paper} elevation={0} sx={{ border: 1, borderColor: 'divider', borderRadius: 2 }}>
              <Table>
                <TableHead sx={{ bgcolor: 'background.default' }}>
                  <TableRow>
                    <TableCell width={40} />
                    <TableCell sx={{ fontWeight: 700 }}>Query Name</TableCell>
                    <TableCell sx={{ fontWeight: 700 }}>Source BO</TableCell>
                    <TableCell sx={{ fontWeight: 700 }}>Dimensions / Measures</TableCell>
                    <TableCell sx={{ fontWeight: 700 }}>Scope</TableCell>
                    <TableCell sx={{ fontWeight: 700 }}>Last Run</TableCell>
                    <TableCell align="right" sx={{ fontWeight: 700 }}>Actions</TableCell>
                  </TableRow>
                </TableHead>
                <TableBody>
                  {filteredQueries.map((query) => (
                    <TableRow
                      key={query.id}
                      hover
                      sx={{ cursor: 'pointer' }}
                      onClick={() => handleOpenPlayground(query)}
                    >
                      <TableCell onClick={(e) => handleToggleFavorite(query.id, e)}>
                        <IconButton size="small" color={query.is_favorite ? 'warning' : 'default'}>
                          {query.is_favorite ? <StarIcon fontSize="small" /> : <StarBorderIcon fontSize="small" />}
                        </IconButton>
                      </TableCell>
                      <TableCell>
                        <Typography variant="subtitle2" sx={{ fontWeight: 700 }}>
                          {query.name}
                        </Typography>
                        <Typography variant="caption" color="text.secondary" noWrap sx={{ maxWidth: 300, display: 'block' }}>
                          {query.description}
                        </Typography>
                      </TableCell>
                      <TableCell>
                        <Chip size="small" label={query.sourceId} variant="outlined" />
                      </TableCell>
                      <TableCell>
                        <Typography variant="body2">
                          {query.queryState.dimensions.length} dims · {query.queryState.measures.length} measures
                        </Typography>
                      </TableCell>
                      <TableCell>
                        <Chip
                          size="small"
                          label={query.is_core ? 'Core' : 'Custom'}
                          color={query.is_core ? 'primary' : 'default'}
                        />
                      </TableCell>
                      <TableCell>
                        <Typography variant="caption" color="text.secondary">
                          {query.last_run ? formatDistanceToNow(new Date(query.last_run)) + ' ago' : 'Never'}
                        </Typography>
                      </TableCell>
                      <TableCell align="right">
                        <Stack direction="row" spacing={1} justifyContent="flex-end">
                          <Button
                            size="small"
                            variant="contained"
                            startIcon={<PlayIcon fontSize="small" />}
                            onClick={(e) => {
                              e.stopPropagation();
                              handleOpenPlayground(query);
                            }}
                            sx={{ textTransform: 'none' }}
                          >
                            Run
                          </Button>
                          <IconButton
                            size="small"
                            onClick={(e) => {
                              e.stopPropagation();
                              setSelectedQuery(query);
                              setMenuAnchorEl(e.currentTarget);
                            }}
                          >
                            <MoreIcon fontSize="small" />
                          </IconButton>
                        </Stack>
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            </TableContainer>
          )}
        </Box>
      </Box>

      {/* Query Actions Context Menu */}
      <Menu
        anchorEl={menuAnchorEl}
        open={Boolean(menuAnchorEl)}
        onClose={() => setMenuAnchorEl(null)}
      >
        <MenuItem
          onClick={() => {
            if (selectedQuery) handleOpenPlayground(selectedQuery);
            setMenuAnchorEl(null);
          }}
        >
          <ListItemIcon><EditIcon fontSize="small" /></ListItemIcon>
          <ListItemText>Edit in Playground</ListItemText>
        </MenuItem>
        <MenuItem
          onClick={() => {
            if (selectedQuery) handleDuplicate(selectedQuery);
          }}
        >
          <ListItemIcon><DuplicateIcon fontSize="small" /></ListItemIcon>
          <ListItemText>Duplicate Query</ListItemText>
        </MenuItem>
        <MenuItem
          onClick={() => {
            setTargetFolderId(selectedQuery?.folder_id || '');
            setMoveDialogOpen(true);
          }}
        >
          <ListItemIcon><MoveIcon fontSize="small" /></ListItemIcon>
          <ListItemText>Move to Folder</ListItemText>
        </MenuItem>
        <Divider />
        <MenuItem
          onClick={() => {
            if (selectedQuery) handleDelete(selectedQuery.id);
          }}
          sx={{ color: 'error.main' }}
        >
          <ListItemIcon sx={{ color: 'error.main' }}><DeleteIcon fontSize="small" /></ListItemIcon>
          <ListItemText>Delete Query</ListItemText>
        </MenuItem>
      </Menu>

      {/* Create Folder Dialog */}
      <Dialog open={isFolderDialogOpen} onClose={() => setIsFolderDialogOpen(false)} maxWidth="xs" fullWidth>
        <DialogTitle>Create Query Folder</DialogTitle>
        <DialogContent>
          <TextField
            autoFocus
            fullWidth
            label="Folder Name"
            value={folderName}
            onChange={(e) => setFolderName(e.target.value)}
            sx={{ mt: 1, mb: 2 }}
          />
          <TextField
            fullWidth
            label="Description (Optional)"
            multiline
            rows={2}
            value={folderDesc}
            onChange={(e) => setFolderDesc(e.target.value)}
          />
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setIsFolderDialogOpen(false)}>Cancel</Button>
          <Button variant="contained" onClick={handleCreateFolder} disabled={!folderName.trim()}>
            Create
          </Button>
        </DialogActions>
      </Dialog>

      {/* Move to Folder Dialog */}
      <Dialog open={moveDialogOpen} onClose={() => setMoveDialogOpen(false)} maxWidth="xs" fullWidth>
        <DialogTitle>Move Query to Folder</DialogTitle>
        <DialogContent>
          <DialogContentText sx={{ mb: 2 }}>
            Choose the folder for <strong>{selectedQuery?.name}</strong>:
          </DialogContentText>
          <List dense>
            <ListItemButton
              selected={targetFolderId === ''}
              onClick={() => setTargetFolderId('')}
            >
              <ListItemIcon><FolderOpenIcon fontSize="small" /></ListItemIcon>
              <ListItemText primary="Root (No Folder)" />
            </ListItemButton>
            {folders.map((f) => (
              <ListItemButton
                key={f.id}
                selected={targetFolderId === f.id}
                onClick={() => setTargetFolderId(f.id)}
              >
                <ListItemIcon><FolderIcon fontSize="small" color={f.is_core ? 'primary' : 'warning'} /></ListItemIcon>
                <ListItemText primary={f.name} secondary={f.is_core ? 'Core' : 'Custom'} />
              </ListItemButton>
            ))}
          </List>
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setMoveDialogOpen(false)}>Cancel</Button>
          <Button variant="contained" onClick={handleMoveQuery}>
            Move
          </Button>
        </DialogActions>
      </Dialog>

      {/* Feedback Toast */}
      <Snackbar
        open={snackbar.open}
        autoHideDuration={3500}
        onClose={() => setSnackbar((prev) => ({ ...prev, open: false }))}
      >
        <Alert severity={snackbar.severity} sx={{ width: '100%' }}>
          {snackbar.message}
        </Alert>
      </Snackbar>
    </Box>
  );
};
