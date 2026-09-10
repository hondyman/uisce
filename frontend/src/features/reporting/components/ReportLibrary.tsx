import React, { useState, useMemo, useEffect } from 'react';
import { useNavigate } from 'react-router-dom';
import {
  Box,
  Card,
  CardContent,
  Typography,
  Button,
  Grid,
  List,
  ListItemButton,
  ListItemIcon,
  ListItemText,
  IconButton,
  TextField,
  InputAdornment,
  Chip,
  Menu,
  MenuItem,
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  Breadcrumbs,
  Link,
  Paper,
  Divider,
  Avatar,
  Stack,
  ToggleButton,
  ToggleButtonGroup,
  CircularProgress,
  Alert,
  Tooltip,
  Snackbar,
  FormControlLabel,
  Switch,
  RadioGroup,
  Radio,
  FormControl,
  FormLabel,
  Autocomplete,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
} from '@mui/material';
import {
  Add as AddIcon,
  Folder as FolderIcon,
  FolderOpen as FolderOpenIcon,
  Description as ReportIcon,
  Search as SearchIcon,
  MoreVert as MoreIcon,
  Star as StarIcon,
  StarBorder as StarBorderIcon,
  Share as ShareIcon,
  Schedule as ScheduleIcon,
  Edit as EditIcon,
  Delete as DeleteIcon,
  FileCopy as DuplicateIcon,
  Refresh as RefreshIcon,
  ViewList as ListViewIcon,
  ViewModule as GridViewIcon,
  AccessTime as RecentIcon,
  Person as PersonIcon,
  PersonOutline as PersonOutlineIcon,
  Group as GroupIcon,
  Public as PublicIcon,
  PlayArrow as PlayArrowIcon,
  DriveFileRenameOutline as RenameIcon,
  Lock as LockIcon,
} from '@mui/icons-material';
import { formatDistanceToNow } from 'date-fns';
import {
  useReportTemplates,
  useDeleteReportTemplate,
  useCreateReportTemplate,
  useUpdateReportTemplate,
  useSetReportFavorite,
  useRemoveReportFavorite,
} from '../../../api/reporting';
import { useFolders } from '../../../api/explorer';
import { resolveGoldCopyTenantId, getCachedGoldCopyId } from '../../../utils/goldCopy';
import {
  CoreIcon,
  CustomIcon,
} from '../../../components/common/CoreCustomIcons';
import { useAccess } from '../../../contexts/AccessContext';
import { useAuth } from '../../../contexts/AuthContext';

// ============================================================================
// REPORT LIBRARY
// Enterprise-grade report management with folders, sharing, and scheduling
// ============================================================================

interface SavedReport {
  id: string;
  name: string;
  description?: string;
  folder_id?: string;
  created_by: string;
  created_at: string;
  updated_at: string;
  is_favorite: boolean;
  is_scheduled: boolean;
  is_shared: boolean;
  share_type?: 'private' | 'team' | 'public';
  shared_with?: string[];
  last_run?: string;
  run_count: number;
  config: any;
  tenant_id?: string;
  category?: string;
  metadata?: any;
  is_core?: boolean;
  is_personal?: boolean;
  created_by_id?: string;
}

interface Folder {
  id: string;
  name: string;
  parent_id?: string;
  created_by: string;
  report_count: number;
}

export const ReportLibrary: React.FC = () => {
  const navigate = useNavigate();
  const { currentTenant, accessibleTenants } = useAccess();
  const { user, isAdmin } = useAuth();
  
  // Identify the gold copy master tenant
  const goldCopyTenant = useMemo(() => {
    return (
      accessibleTenants.find(t => t.gold_copy) ||
      (currentTenant?.gold_copy ? currentTenant : null)
    );
  }, [accessibleTenants, currentTenant]);

  // Track gold copy tenant ID
  const [goldCopyId, setGoldCopyId] = useState<string | null>(
    goldCopyTenant?.id || getCachedGoldCopyId()
  );
  useEffect(() => {
    resolveGoldCopyTenantId().then(id => {
      if (id) setGoldCopyId(id);
    });
  }, []);

  // --- Real API Data ---
  const { data: apiReports, isLoading: isLoadingReports } = useReportTemplates();
  const { data: apiFolders, isLoading: isLoadingFolders } = useFolders();
  const deleteReportMutation = useDeleteReportTemplate();
  const createReportMutation = useCreateReportTemplate();
  const updateReportMutation = useUpdateReportTemplate();
  const setFavoriteMutation = useSetReportFavorite();
  const removeFavoriteMutation = useRemoveReportFavorite();
  
  // Transform API data to component interfaces
  const reports = useMemo<SavedReport[]>(() => {
    if (!apiReports) return [];
    const isGoldCopyScope = Boolean(currentTenant?.gold_copy);
    const goldId = goldCopyTenant?.id || goldCopyId;

    return apiReports.map(r => {
      const meta = (r.metadata as any) || (r.definition as any)?.metadata || (r as any).layout_config?.metadata || {};
      const tId = (r as any).tenant_id;
      const isCore = Boolean(
        (r as any).is_core === true ||
        (r as any).isCore === true ||
        (r as any).gold_copy === true ||
        meta.is_core === true ||
        meta.isCore === true ||
        isGoldCopyScope ||
        (tId && goldId && tId === goldId) ||
        tId === '00000000-0000-0000-0000-000000000000' ||
        !tId
      );
      const isFav = Boolean((r as any).is_favorite);
      const isShared = Boolean(meta.is_shared ?? (r as any).is_public ?? (r as any).is_shared);
      const shareType = (meta.share_type as 'private' | 'team' | 'public') || (isShared ? 'public' : 'private');
      const sharedWith = Array.isArray(meta.shared_with) ? meta.shared_with : [];
      const isPersonal = Boolean((r as any).is_personal ?? meta.is_personal);
      const createdById = (r as any).created_by_id || meta.created_by_id || undefined;

      return {
        id: r.id,
        name: r.name,
        description: r.description || '',
        folder_id: meta.folder_id || undefined,
        created_by: meta.created_by || (r as any).created_by || 'User',
        created_at: r.createdAt || new Date().toISOString(),
        updated_at: r.updatedAt || new Date().toISOString(),
        is_favorite: isFav,
        is_scheduled: meta.is_scheduled || false,
        is_shared: isShared,
        share_type: shareType,
        shared_with: sharedWith,
        last_run: meta.last_run,
        run_count: meta.run_count || 0,
        config: r.definition || (r as any).layout_config || {},
        tenant_id: tId,
        category: (r as any).category,
        metadata: meta,
        is_core: isCore,
        is_personal: isPersonal,
        created_by_id: createdById,
      };
    });
  }, [apiReports, currentTenant, goldCopyTenant, goldCopyId]);

  const folders = useMemo<Folder[]>(() => {
    if (!apiFolders) return [];
    return apiFolders.map(f => ({
      id: f.id,
      name: f.name,
      parent_id: f.parentId || undefined,
      created_by: 'User',
      report_count: f.items ? f.items.filter(i => i.itemType === 'workbook').length : 0,
    }));
  }, [apiFolders]);
  
  const [currentFolder, setCurrentFolder] = useState<string | null>(null);
  const [searchQuery, setSearchQuery] = useState('');
  const [viewMode, setViewMode] = useState<'list' | 'grid'>('list');
  const [filterType, setFilterType] = useState<'all' | 'favorites' | 'recent' | 'shared' | 'core' | 'custom' | 'personal'>('all');
  const [selectedReport, setSelectedReport] = useState<SavedReport | null>(null);
  const [menuAnchor, setMenuAnchor] = useState<null | HTMLElement>(null);
  const [shareDialogOpen, setShareDialogOpen] = useState(false);
  const [scheduleDialogOpen, setScheduleDialogOpen] = useState(false);
  const [newFolderDialogOpen, setNewFolderDialogOpen] = useState(false);
  const [newFolderName, setNewFolderName] = useState('');
  const [deleteDialogOpen, setDeleteDialogOpen] = useState(false);
  const [reportToDelete, setReportToDelete] = useState<SavedReport | null>(null);
  const [deleteError, setDeleteError] = useState<string | null>(null);

  // Rename state
  const [renameDialogOpen, setRenameDialogOpen] = useState(false);
  const [reportToRename, setReportToRename] = useState<SavedReport | null>(null);
  const [renameValue, setRenameValue] = useState('');
  const [renameError, setRenameError] = useState<string | null>(null);

  // Clone/Duplicate state
  const [cloneDialogOpen, setCloneDialogOpen] = useState(false);
  const [reportToClone, setReportToClone] = useState<SavedReport | null>(null);
  const [cloneName, setCloneName] = useState('');
  const [cloneIsPersonal, setCloneIsPersonal] = useState<boolean>(true);
  const [cloneError, setCloneError] = useState<string | null>(null);

  // Helper for determining if sharing is permitted on this report
  const getShareDisabledReason = (report: SavedReport): string | null => {
    if (report.is_core) {
      return 'Core reports cannot be shared';
    }
    if (!report.is_personal) {
      return 'Only personal reports can be shared';
    }
    if (report.created_by_id && user?.id && report.created_by_id !== user.id) {
      return 'Only the report author can share this report';
    }
    return null;
  };

  // Share state
  const [shareTargetReport, setShareTargetReport] = useState<SavedReport | null>(null);
  const [shareIsShared, setShareIsShared] = useState(false);
  const [shareType, setShareType] = useState<'public' | 'team'>('public');
  const [sharedWith, setSharedWith] = useState<string[]>([]);
  const [shareError, setShareError] = useState<string | null>(null);

  // Feedback notifications
  const [snackbar, setSnackbar] = useState<{ open: boolean; message: string; severity: 'success' | 'error' | 'info' }>({
    open: false,
    message: '',
    severity: 'info',
  });
  
  const isLoading = isLoadingReports || isLoadingFolders;

  const handleMenuOpen = (event: React.MouseEvent<HTMLElement>, report: SavedReport) => {
    setSelectedReport(report);
    setMenuAnchor(event.currentTarget);
  };

  const handleMenuClose = () => {
    setMenuAnchor(null);
  };

  const handleToggleFavorite = async (reportId: string) => {
    const report = reports.find(r => r.id === reportId);
    if (!report) return;
    const nextFavorite = !report.is_favorite;
    try {
      if (nextFavorite) {
        await setFavoriteMutation.mutateAsync(report.id);
      } else {
        await removeFavoriteMutation.mutateAsync(report.id);
      }
      setSnackbar({
        open: true,
        message: nextFavorite ? `Added "${report.name}" to favorites` : `Removed "${report.name}" from favorites`,
        severity: 'success',
      });
    } catch (err: any) {
      setSnackbar({
        open: true,
        message: `Failed to update favorite: ${err?.message || 'Unknown error'}`,
        severity: 'error',
      });
    }
  };

  const handleOpenShareDialog = (report: SavedReport) => {
    setShareTargetReport(report);
    setShareIsShared(report.is_shared);
    setShareType(report.share_type === 'team' ? 'team' : 'public');
    setSharedWith(report.shared_with || []);
    setShareError(null);
    setShareDialogOpen(true);
    handleMenuClose();
  };

  const handleSaveShare = async () => {
    if (!shareTargetReport) return;
    try {
      setShareError(null);
      const nextMetadata = {
        ...(shareTargetReport.metadata || {}),
        is_shared: shareIsShared,
        share_type: shareIsShared ? shareType : 'private',
        shared_with: shareIsShared ? sharedWith : [],
      };

      await updateReportMutation.mutateAsync({
        id: shareTargetReport.id,
        payload: {
          id: shareTargetReport.id,
          name: shareTargetReport.name,
          template_name: shareTargetReport.name,
          tenant_id: shareTargetReport.tenant_id || '00000000-0000-0000-0000-000000000000',
          description: shareTargetReport.description || '',
          category: shareTargetReport.category || 'general',
          is_active: true,
          is_public: shareIsShared && shareType === 'public',
          layout_config: {
            ...(shareTargetReport.config || {}),
            metadata: nextMetadata,
          },
          definition: {
            ...(shareTargetReport.config || {}),
            metadata: nextMetadata,
          },
          parameter_schema: shareTargetReport.config?.parameters || {},
          metadata: nextMetadata,
        },
      });
      setShareDialogOpen(false);
      setSnackbar({
        open: true,
        message: shareIsShared
          ? `Report "${shareTargetReport.name}" is now shared`
          : `Report "${shareTargetReport.name}" is now private`,
        severity: 'success',
      });
    } catch (err: any) {
      setShareError(err?.message || 'Failed to update share settings.');
    }
  };

  const handleCreateFolder = async () => {
    if (!newFolderName.trim()) return;
    setNewFolderDialogOpen(false);
    setNewFolderName('');
  };

  const handleRunReport = (report: SavedReport) => {
    navigate(`/reports/${report.id}/edit`);
    handleMenuClose();
  };

  const handleEditReport = (report: SavedReport) => {
    navigate(`/reports/${report.id}/edit`);
    handleMenuClose();
  };

  const handleRenameClick = (report: SavedReport) => {
    setReportToRename(report);
    setRenameValue(report.name);
    setRenameError(null);
    setRenameDialogOpen(true);
    handleMenuClose();
  };

  const handleConfirmRename = async () => {
    if (!reportToRename || !renameValue.trim()) return;
    const newName = renameValue.trim();
    try {
      setRenameError(null);
      await updateReportMutation.mutateAsync({
        id: reportToRename.id,
        payload: {
          id: reportToRename.id,
          name: newName,
          template_name: newName,
          tenant_id: reportToRename.tenant_id || '00000000-0000-0000-0000-000000000000',
          description: reportToRename.description || '',
          category: reportToRename.category || 'general',
          is_active: true,
          layout_config: reportToRename.config || {},
          definition: reportToRename.config || {},
          parameter_schema: reportToRename.config?.parameters || {},
          metadata: reportToRename.metadata || {},
        },
      });
      setRenameDialogOpen(false);
      setReportToRename(null);
      setSnackbar({ open: true, message: `Report renamed to "${newName}"`, severity: 'success' });
    } catch (err: any) {
      if (err?.message?.includes('409') || err?.message?.toLowerCase().includes('already exists') || err?.message?.toLowerCase().includes('conflict')) {
        setRenameError('A report with this name already exists in this tenant.');
      } else {
        setRenameError(err?.message || 'Failed to rename report.');
      }
    }
  };

  const handleOpenCloneDialog = (report: SavedReport) => {
    setReportToClone(report);
    const baseCopyName = `${report.name} (Copy)`;
    let copyName = baseCopyName;
    let counter = 1;
    while (reports.some(r => r.name.toLowerCase() === copyName.toLowerCase())) {
      copyName = `${baseCopyName} ${++counter}`;
    }
    setCloneName(copyName);
    const isGoldCopy = Boolean(currentTenant?.gold_copy);
    setCloneIsPersonal(isGoldCopy ? false : true);
    setCloneError(null);
    setCloneDialogOpen(true);
    handleMenuClose();
  };

  const handleConfirmClone = async () => {
    if (!reportToClone || !cloneName.trim()) return;
    const isGoldCopy = Boolean(currentTenant?.gold_copy);
    const isCallerAdmin = typeof isAdmin === 'function' ? isAdmin() : Boolean(isAdmin);
    const targetIsPersonal = isGoldCopy ? false : (isCallerAdmin ? cloneIsPersonal : true);

    const trimmedName = cloneName.trim();
    if (reports.some(r => r.name.trim().toLowerCase() === trimmedName.toLowerCase())) {
      setCloneError('A report with this name already exists in this tenant.');
      return;
    }

    try {
      setCloneError(null);
      await createReportMutation.mutateAsync({
        name: trimmedName,
        template_name: trimmedName,
        tenant_id: currentTenant?.id || reportToClone.tenant_id || goldCopyId || '00000000-0000-0000-0000-000000000000',
        description: reportToClone.description || '',
        definition: reportToClone.config || {},
        layout_config: reportToClone.config || {},
        parameter_schema: reportToClone.config?.parameters || {},
        category: (reportToClone as any).category || 'general',
        is_active: true,
        is_personal: targetIsPersonal,
        metadata: {
          folder_id: reportToClone.folder_id,
          created_by: user?.name || user?.email || 'User',
          created_by_id: user?.id,
          is_shared: false,
          share_type: 'private',
          shared_with: [],
        },
      });
      setCloneDialogOpen(false);
      setReportToClone(null);
      setSnackbar({ open: true, message: `Report duplicated as "${trimmedName}"`, severity: 'success' });
    } catch (err: any) {
      if (err?.message?.includes('409') || err?.message?.toLowerCase().includes('already exists') || err?.message?.toLowerCase().includes('conflict')) {
        setCloneError('A report with this name already exists in this tenant.');
      } else {
        setCloneError(err?.message || 'Failed to duplicate report.');
      }
    }
  };

  const handleDeleteClick = (report: SavedReport) => {
    setReportToDelete(report);
    setDeleteError(null);
    setDeleteDialogOpen(true);
    handleMenuClose();
  };

  const handleConfirmDelete = async () => {
    if (!reportToDelete) return;
    try {
      setDeleteError(null);
      const deletedName = reportToDelete.name;
      await deleteReportMutation.mutateAsync(reportToDelete.id);
      setDeleteDialogOpen(false);
      setReportToDelete(null);
      setSnackbar({ open: true, message: `Report "${deletedName}" deleted successfully`, severity: 'success' });
    } catch (err: any) {
      setDeleteError(err?.message || 'Failed to delete report.');
    }
  };

  const filteredReports = reports.filter(report => {
    // Filter by folder only in the general "all" view; specific facets show across all folders
    if (filterType === 'all' && currentFolder && report.folder_id !== currentFolder) return false;

    // Filter by type
    if (filterType === 'favorites' && !report.is_favorite) return false;
    if (filterType === 'shared' && !report.is_shared) return false;
    if (filterType === 'core' && !report.is_core) return false;
    if (filterType === 'custom' && (report.is_core || report.is_personal)) return false;
    if (filterType === 'personal' && (!report.is_personal || (report.created_by_id && user?.id && report.created_by_id !== user.id))) return false;
    if (filterType === 'recent' && report.last_run) {
      const daysSinceRun = (Date.now() - new Date(report.last_run).getTime()) / (1000 * 60 * 60 * 24);
      if (daysSinceRun > 7) return false;
    }

    // Search filter
    if (searchQuery && !report.name.toLowerCase().includes(searchQuery.toLowerCase())) {
      return false;
    }

    return true;
  });

  const getShareIcon = (shareType?: string) => {
    switch (shareType) {
      case 'public':
        return <PublicIcon fontSize="small" />;
      case 'team':
        return <GroupIcon fontSize="small" />;
      default:
        return <PersonIcon fontSize="small" />;
    }
  };

  if (isLoading) {
    return (
      <Box sx={{ display: 'flex', justifyContent: 'center', alignItems: 'center', height: '50vh' }}>
        <CircularProgress />
      </Box>
    );
  }

  return (
    <Box sx={{ p: 3 }}>
      {/* Header */}
      <Box sx={{ display: 'flex', justifyContent: 'space-between', mb: 3 }}>
        <Box>
          <Typography variant="h4">Report Library</Typography>
          <Breadcrumbs sx={{ mt: 1 }}>
            <Link
              component="button"
              variant="body2"
              onClick={() => setCurrentFolder(null)}
              sx={{ cursor: 'pointer' }}
            >
              All Reports
            </Link>
            {currentFolder && (
              <Typography variant="body2" color="text.primary">
                {folders.find(f => f.id === currentFolder)?.name}
              </Typography>
            )}
          </Breadcrumbs>
        </Box>
        <Box sx={{ display: 'flex', gap: 1 }}>
          <Button
            variant="outlined"
            startIcon={<FolderIcon />}
            onClick={() => setNewFolderDialogOpen(true)}
          >
            New Folder
          </Button>
          <Button
            variant="contained"
            startIcon={<AddIcon />}
            onClick={() => navigate('/reports/builder')}
          >
            New Report
          </Button>
        </Box>
      </Box>

      <Grid container spacing={3}>
        {/* Sidebar */}
        <Grid size={{ 'xs': 12, 'md': 3 }}>
          <Card>
            <List>
              <ListItemButton
                selected={filterType === 'all' && !currentFolder}
                onClick={() => {
                  setFilterType('all');
                  setCurrentFolder(null);
                }}
              >
                <ListItemIcon><ReportIcon /></ListItemIcon>
                <ListItemText primary="All Reports" />
              </ListItemButton>
              <ListItemButton
                selected={filterType === 'favorites'}
                onClick={() => {
                  setFilterType('favorites');
                  setCurrentFolder(null);
                }}
              >
                <ListItemIcon><StarIcon /></ListItemIcon>
                <ListItemText primary="Favorites" />
              </ListItemButton>
              <ListItemButton
                selected={filterType === 'recent'}
                onClick={() => {
                  setFilterType('recent');
                  setCurrentFolder(null);
                }}
              >
                <ListItemIcon><RecentIcon /></ListItemIcon>
                <ListItemText primary="Recent" />
              </ListItemButton>
              <ListItemButton
                selected={filterType === 'shared'}
                onClick={() => {
                  setFilterType('shared');
                  setCurrentFolder(null);
                }}
              >
                <ListItemIcon><ShareIcon /></ListItemIcon>
                <ListItemText primary="Shared" />
              </ListItemButton>
              <ListItemButton
                selected={filterType === 'personal'}
                onClick={() => {
                  setFilterType('personal');
                  setCurrentFolder(null);
                }}
              >
                <ListItemIcon><PersonOutlineIcon fontSize="small" /></ListItemIcon>
                <ListItemText primary="Personal Reports" />
              </ListItemButton>
              <ListItemButton
                selected={filterType === 'custom'}
                onClick={() => {
                  setFilterType('custom');
                  setCurrentFolder(null);
                }}
              >
                <ListItemIcon><CustomIcon fontSize="small" /></ListItemIcon>
                <ListItemText primary="Custom Reports" />
              </ListItemButton>
              <ListItemButton
                selected={filterType === 'core'}
                onClick={() => {
                  setFilterType('core');
                  setCurrentFolder(null);
                }}
              >
                <ListItemIcon><CoreIcon fontSize="small" /></ListItemIcon>
                <ListItemText primary="Core Reports" />
              </ListItemButton>
            </List>
            <Divider />
            <List subheader={<Typography variant="overline" sx={{ px: 2 }}>Folders</Typography>}>
              {folders.map(folder => (
                <ListItemButton
                  key={folder.id}
                  selected={currentFolder === folder.id}
                  onClick={() => setCurrentFolder(folder.id)}
                >
                  <ListItemIcon>
                    {currentFolder === folder.id ? <FolderOpenIcon color="primary" /> : <FolderIcon />}
                  </ListItemIcon>
                  <ListItemText
                    primary={folder.name}
                    secondary={`${folder.report_count} reports`}
                  />
                </ListItemButton>
              ))}
            </List>
          </Card>
        </Grid>

        {/* Main Content */}
        <Grid size={{ 'xs': 12, 'md': 9 }}>
          {/* Controls */}
          <Box sx={{ display: 'flex', justifyContent: 'space-between', mb: 2 }}>
            <TextField
              placeholder="Search reports..."
              size="small"
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
              InputProps={{
                startAdornment: <InputAdornment position="start"><SearchIcon /></InputAdornment>,
              }}
              sx={{ width: 300 }}
            />
            <ToggleButtonGroup
              size="small"
              value={viewMode}
              exclusive
              onChange={(_, v) => v && setViewMode(v)}
            >
              <ToggleButton value="list" aria-label="List view"><ListViewIcon /></ToggleButton>
              <ToggleButton value="grid" aria-label="Grid view"><GridViewIcon /></ToggleButton>
            </ToggleButtonGroup>
          </Box>

          {/* Reports List/Grid */}
          {reports.length === 0 ? (
            <Paper sx={{ p: 4, textAlign: 'center' }}>
              <Typography color="text.secondary">No reports found.</Typography>
            </Paper>
          ) : viewMode === 'list' ? (
            <TableContainer component={Card}>
              <Table size="small" aria-label="reports table">
                <TableHead sx={{ backgroundColor: 'action.hover' }}>
                  <TableRow>
                    <TableCell sx={{ width: 80, fontWeight: 600 }}>Type</TableCell>
                    <TableCell sx={{ minWidth: 220, fontWeight: 600 }}>Report Name</TableCell>
                    <TableCell sx={{ minWidth: 120, fontWeight: 600 }}>Sharing</TableCell>
                    <TableCell sx={{ width: 90, fontWeight: 600 }}>Runs</TableCell>
                    <TableCell sx={{ width: 140, fontWeight: 600 }}>Last Run</TableCell>
                    <TableCell sx={{ width: 130, fontWeight: 600 }}>Created By</TableCell>
                    <TableCell align="right" sx={{ width: 220, fontWeight: 600 }}>Actions</TableCell>
                  </TableRow>
                </TableHead>
                <TableBody>
                  {filteredReports.map(report => (
                    <TableRow
                      key={report.id}
                      hover
                      sx={{ cursor: 'pointer' }}
                      onClick={() => handleRunReport(report)}
                    >
                      {/* Column 1: Type (Core / Custom) */}
                      <TableCell sx={{ width: 70, py: 1, textAlign: 'center' }} onClick={(e) => e.stopPropagation()}>
                        <Tooltip title={report.is_core ? 'Core Report' : 'Custom Report'}>
                          <Box sx={{ display: 'inline-flex', alignItems: 'center', justifyContent: 'center' }}>
                            {report.is_core ? (
                              <CoreIcon fontSize="medium" />
                            ) : (
                              <CustomIcon fontSize="medium" />
                            )}
                          </Box>
                        </Tooltip>
                      </TableCell>

                      {/* Column 2: Name */}
                      <TableCell sx={{ py: 1 }}>
                        <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                          <Typography variant="body2" sx={{ fontWeight: 600, color: 'text.primary' }}>
                            {report.name}
                          </Typography>
                          <IconButton
                            size="small"
                            onClick={(e) => {
                              e.stopPropagation();
                              handleToggleFavorite(report.id);
                            }}
                          >
                            {report.is_favorite ? (
                              <StarIcon fontSize="small" color="warning" />
                            ) : (
                              <StarBorderIcon fontSize="small" />
                            )}
                          </IconButton>
                        </Box>
                        {report.description && (
                          <Typography variant="caption" color="text.secondary" noWrap sx={{ display: 'block', maxWidth: 350 }}>
                            {report.description}
                          </Typography>
                        )}
                      </TableCell>

                      {/* Column 3: Sharing */}
                      <TableCell sx={{ py: 1 }}>
                        {report.is_shared ? (
                          <Chip
                            size="small"
                            icon={getShareIcon(report.share_type)}
                            label={report.share_type === 'team' ? 'Team' : 'Public'}
                            variant="outlined"
                            color="info"
                          />
                        ) : (
                          <Typography variant="caption" color="text.secondary">
                            Private
                          </Typography>
                        )}
                      </TableCell>

                      {/* Column 4: Runs */}
                      <TableCell sx={{ py: 1 }}>
                        <Typography variant="body2">{report.run_count}</Typography>
                      </TableCell>

                      {/* Column 5: Last Run */}
                      <TableCell sx={{ py: 1 }}>
                        <Typography variant="caption" color="text.secondary">
                          {report.last_run ? `${formatDistanceToNow(new Date(report.last_run))} ago` : 'Never'}
                        </Typography>
                      </TableCell>

                      {/* Column 6: Owner */}
                      <TableCell sx={{ py: 1 }}>
                        <Typography variant="caption">{report.created_by}</Typography>
                      </TableCell>

                      {/* Column 7: Actions */}
                      <TableCell align="right" sx={{ py: 1 }} onClick={(e) => e.stopPropagation()}>
                        <Stack direction="row" spacing={0.5} justifyContent="flex-end" alignItems="center">
                          <Tooltip title="Run Report">
                            <IconButton size="small" color="primary" onClick={() => handleRunReport(report)}>
                              <PlayArrowIcon fontSize="small" />
                            </IconButton>
                          </Tooltip>
                          <Tooltip title="Edit Report">
                            <IconButton size="small" onClick={() => handleEditReport(report)}>
                              <EditIcon fontSize="small" />
                            </IconButton>
                          </Tooltip>
                          <Tooltip title={report.is_core ? "Core reports cannot be renamed" : "Rename Report"}>
                            <span>
                              <IconButton
                                size="small"
                                onClick={() => handleRenameClick(report)}
                                disabled={report.is_core}
                              >
                                <RenameIcon fontSize="small" />
                              </IconButton>
                            </span>
                          </Tooltip>
                          {(() => {
                            const shareDisabledReason = getShareDisabledReason(report);
                            return (
                              <Tooltip title={shareDisabledReason || (report.is_shared ? 'Manage Sharing' : 'Share Report')}>
                                <span>
                                  <IconButton
                                    size="small"
                                    color={report.is_shared ? 'primary' : 'default'}
                                    onClick={() => handleOpenShareDialog(report)}
                                    disabled={Boolean(shareDisabledReason)}
                                  >
                                    <ShareIcon fontSize="small" />
                                  </IconButton>
                                </span>
                              </Tooltip>
                            );
                          })()}
                          <Tooltip title="Duplicate Report">
                            <span>
                              <IconButton
                                size="small"
                                onClick={() => handleOpenCloneDialog(report)}
                                disabled={createReportMutation.isPending}
                              >
                                <DuplicateIcon fontSize="small" />
                              </IconButton>
                            </span>
                          </Tooltip>
                          <Tooltip title={report.is_core ? "Core reports cannot be deleted" : "Delete Report"}>
                            <span>
                              <IconButton
                                size="small"
                                color="error"
                                onClick={() => handleDeleteClick(report)}
                                disabled={report.is_core}
                              >
                                <DeleteIcon fontSize="small" />
                              </IconButton>
                            </span>
                          </Tooltip>
                          <Tooltip title="More options">
                            <IconButton size="small" onClick={(e) => handleMenuOpen(e, report)}>
                              <MoreIcon fontSize="small" />
                            </IconButton>
                          </Tooltip>
                        </Stack>
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            </TableContainer>
          ) : (
            <Grid container spacing={2}>
              {filteredReports.map(report => (
                <Grid size={{ 'xs': 12, 'sm': 6, 'md': 4 }} key={report.id}>
                  <Card>
                    <CardContent>
                      <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', mb: 2 }}>
                        <Box sx={{ display: 'flex', alignItems: 'center' }}>
                          <Tooltip title={report.is_core ? 'Core Report' : 'Custom Report'}>
                            <Avatar sx={{ bgcolor: report.is_core ? 'info.light' : 'secondary.light', width: 36, height: 36 }}>
                              {report.is_core ? (
                                <CoreIcon fontSize="small" sx={{ color: '#fff' }} />
                              ) : (
                                <CustomIcon fontSize="small" sx={{ color: '#fff' }} />
                              )}
                            </Avatar>
                          </Tooltip>
                        </Box>
                        <IconButton
                          size="small"
                          onClick={() => handleToggleFavorite(report.id)}
                        >
                          {report.is_favorite ? <StarIcon color="warning" /> : <StarBorderIcon />}
                        </IconButton>
                      </Box>
                      <Typography variant="h6" noWrap gutterBottom>
                        {report.name}
                      </Typography>
                      <Typography variant="body2" color="text.secondary" sx={{ mb: 2, height: 40, overflow: 'hidden' }}>
                        {report.description || 'No description available'}
                      </Typography>
                      <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', mt: 2, pt: 1, borderTop: 1, borderColor: 'divider' }}>
                        <Box sx={{ display: 'flex', alignItems: 'center', gap: 0.5 }}>
                          {report.is_shared && (
                            <Chip
                              size="small"
                              icon={getShareIcon(report.share_type)}
                              label={report.share_type === 'team' ? 'Team' : 'Public'}
                              color="info"
                              variant="outlined"
                            />
                          )}
                        </Box>
                        <Stack direction="row" spacing={0.5} alignItems="center">
                          <Tooltip title="Run Report">
                            <IconButton size="small" color="primary" onClick={() => handleRunReport(report)}>
                              <PlayArrowIcon fontSize="small" />
                            </IconButton>
                          </Tooltip>
                          <Tooltip title="Edit Report">
                            <IconButton size="small" onClick={() => handleEditReport(report)}>
                              <EditIcon fontSize="small" />
                            </IconButton>
                          </Tooltip>
                          <Tooltip title={report.is_core ? "Core reports cannot be renamed" : "Rename Report"}>
                            <span>
                              <IconButton
                                size="small"
                                onClick={() => handleRenameClick(report)}
                                disabled={report.is_core}
                              >
                                <RenameIcon fontSize="small" />
                              </IconButton>
                            </span>
                          </Tooltip>
                          {(() => {
                            const shareDisabledReason = getShareDisabledReason(report);
                            return (
                              <Tooltip title={shareDisabledReason || (report.is_shared ? 'Manage Sharing' : 'Share Report')}>
                                <span>
                                  <IconButton
                                    size="small"
                                    color={report.is_shared ? 'primary' : 'default'}
                                    onClick={() => handleOpenShareDialog(report)}
                                    disabled={Boolean(shareDisabledReason)}
                                  >
                                    <ShareIcon fontSize="small" />
                                  </IconButton>
                                </span>
                              </Tooltip>
                            );
                          })()}
                          <Tooltip title="Duplicate Report">
                            <span>
                              <IconButton
                                size="small"
                                onClick={() => handleOpenCloneDialog(report)}
                                disabled={createReportMutation.isPending}
                              >
                                <DuplicateIcon fontSize="small" />
                              </IconButton>
                            </span>
                          </Tooltip>
                          <Tooltip title={report.is_core ? "Core reports cannot be deleted" : "Delete Report"}>
                            <span>
                              <IconButton
                                size="small"
                                color="error"
                                onClick={() => handleDeleteClick(report)}
                                disabled={report.is_core}
                              >
                                <DeleteIcon fontSize="small" />
                              </IconButton>
                            </span>
                          </Tooltip>
                          <Tooltip title="More options">
                            <IconButton size="small" onClick={(e) => handleMenuOpen(e, report)}>
                              <MoreIcon fontSize="small" />
                            </IconButton>
                          </Tooltip>
                        </Stack>
                      </Box>
                    </CardContent>
                  </Card>
                </Grid>
              ))}
            </Grid>
          )}
        </Grid>
      </Grid>

      {/* Menu & Dialogs */}
      <Menu
        anchorEl={menuAnchor}
        open={Boolean(menuAnchor)}
        onClose={handleMenuClose}
      >
        <MenuItem onClick={() => handleRunReport(selectedReport!)}>
          <ListItemIcon><RefreshIcon fontSize="small" /></ListItemIcon>
          Run Report
        </MenuItem>
        <MenuItem onClick={() => handleToggleFavorite(selectedReport!.id)}>
          <ListItemIcon>
            {selectedReport?.is_favorite ? <StarBorderIcon fontSize="small" /> : <StarIcon fontSize="small" />}
          </ListItemIcon>
          {selectedReport?.is_favorite ? 'Remove from Favorites' : 'Add to Favorites'}
        </MenuItem>
        <Divider />
        {(() => {
          if (!selectedReport) return null;
          const shareDisabledReason = getShareDisabledReason(selectedReport);
          const item = (
            <MenuItem
              disabled={Boolean(shareDisabledReason)}
              onClick={() => handleOpenShareDialog(selectedReport)}
            >
              <ListItemIcon><ShareIcon fontSize="small" /></ListItemIcon>
              Share
            </MenuItem>
          );
          if (shareDisabledReason) {
            return (
              <Tooltip title={shareDisabledReason} placement="left">
                <div>{item}</div>
              </Tooltip>
            );
          }
          return item;
        })()}
        <MenuItem onClick={() => setScheduleDialogOpen(true)}>
          <ListItemIcon><ScheduleIcon fontSize="small" /></ListItemIcon>
          Schedule
        </MenuItem>
        <MenuItem onClick={() => handleEditReport(selectedReport!)}>
          <ListItemIcon><EditIcon fontSize="small" /></ListItemIcon>
          Edit
        </MenuItem>
        <MenuItem
          disabled={selectedReport?.is_core}
          onClick={() => handleRenameClick(selectedReport!)}
        >
          <ListItemIcon><RenameIcon fontSize="small" /></ListItemIcon>
          Rename
        </MenuItem>
        <MenuItem onClick={() => handleOpenCloneDialog(selectedReport!)}>
          <ListItemIcon><DuplicateIcon fontSize="small" /></ListItemIcon>
          Duplicate
        </MenuItem>
        <Divider />
        <MenuItem
          disabled={selectedReport?.is_core}
          onClick={() => handleDeleteClick(selectedReport!)}
          sx={{ color: selectedReport?.is_core ? 'text.disabled' : 'error.main' }}
        >
          <ListItemIcon><DeleteIcon fontSize="small" color={selectedReport?.is_core ? 'disabled' : 'error'} /></ListItemIcon>
          Delete
        </MenuItem>
      </Menu>

      {/* Delete Confirmation Dialog */}
      <Dialog
        open={deleteDialogOpen}
        onClose={() => !deleteReportMutation.isPending && setDeleteDialogOpen(false)}
      >
        <DialogTitle>Delete Report</DialogTitle>
        <DialogContent>
          {deleteError && (
            <Alert severity="error" sx={{ mb: 2 }}>
              {deleteError}
            </Alert>
          )}
          <Typography>
            Are you sure you want to delete &quot;{reportToDelete?.name}&quot;? This action cannot be undone.
          </Typography>
        </DialogContent>
        <DialogActions>
          <Button
            onClick={() => setDeleteDialogOpen(false)}
            disabled={deleteReportMutation.isPending}
          >
            Cancel
          </Button>
          <Button
            variant="contained"
            color="error"
            onClick={handleConfirmDelete}
            disabled={deleteReportMutation.isPending}
            startIcon={deleteReportMutation.isPending ? <CircularProgress size={16} color="inherit" /> : <DeleteIcon />}
          >
            {deleteReportMutation.isPending ? 'Deleting...' : 'Delete'}
          </Button>
        </DialogActions>
      </Dialog>

      {/* Interactive Share Report Dialog */}
      <Dialog
        open={shareDialogOpen}
        onClose={() => !updateReportMutation.isPending && setShareDialogOpen(false)}
        maxWidth="sm"
        fullWidth
      >
        <DialogTitle sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
          <ShareIcon color="primary" />
          Share Report &quot;{shareTargetReport?.name}&quot;
        </DialogTitle>
        <DialogContent dividers>
          {shareError && (
            <Alert severity="error" sx={{ mb: 2 }}>
              {shareError}
            </Alert>
          )}
          <Stack spacing={2.5}>
            <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
              <Box>
                <Typography variant="subtitle1" fontWeight={600}>
                  Enable Report Sharing
                </Typography>
                <Typography variant="body2" color="text.secondary">
                  Allow other users and team members to access this report.
                </Typography>
              </Box>
              <FormControlLabel
                control={
                  <Switch
                    checked={shareIsShared}
                    onChange={(e) => setShareIsShared(e.target.checked)}
                    color="primary"
                  />
                }
                label={shareIsShared ? 'Shared' : 'Private'}
              />
            </Box>

            {shareIsShared && (
              <>
                <Divider />
                <FormControl component="fieldset">
                  <FormLabel component="legend" sx={{ fontWeight: 600, mb: 0.5 }}>
                    Who can access
                  </FormLabel>
                  <RadioGroup
                    value={shareType}
                    onChange={(e) => setShareType(e.target.value as 'public' | 'team')}
                  >
                    <FormControlLabel
                      value="public"
                      control={<Radio size="small" />}
                      label={
                        <Box>
                          <Typography variant="body2" fontWeight={500}>
                            Organization / Public
                          </Typography>
                          <Typography variant="caption" color="text.secondary">
                            Anyone in your organization can view and execute this report.
                          </Typography>
                        </Box>
                      }
                      sx={{ mb: 1 }}
                    />
                    <FormControlLabel
                      value="team"
                      control={<Radio size="small" />}
                      label={
                        <Box>
                          <Typography variant="body2" fontWeight={500}>
                            Specific People &amp; Groups
                          </Typography>
                          <Typography variant="caption" color="text.secondary">
                            Only designated users and groups listed below can access.
                          </Typography>
                        </Box>
                      }
                    />
                  </RadioGroup>
                </FormControl>

                {shareType === 'team' && (
                  <Box>
                    <Typography variant="caption" color="text.secondary" sx={{ display: 'block', mb: 1 }}>
                      Add colleagues or team groups (e.g. Finance, Risk, Operations, john.doe@company.com):
                    </Typography>
                    <Autocomplete
                      multiple
                      freeSolo
                      options={[
                        'Finance Team',
                        'Risk Management',
                        'Operations',
                        'Executive Board',
                        'Portfolio Managers',
                        'Compliance',
                        'admin@uisce.io',
                      ]}
                      value={sharedWith}
                      onChange={(_, newValue) => setSharedWith(newValue)}
                      renderTags={(value: readonly string[], getTagProps) =>
                        value.map((option: string, index: number) => {
                          const tagProps = getTagProps({ index });
                          return (
                            <Chip
                              {...tagProps}
                              key={option}
                              variant="outlined"
                              label={option}
                              size="small"
                              icon={option.includes('@') ? <PersonIcon fontSize="small" /> : <GroupIcon fontSize="small" />}
                            />
                          );
                        })
                      }
                      renderInput={(params) => (
                        <TextField
                          {...params}
                          variant="outlined"
                          label="People or Groups"
                          placeholder="Type name or group and press Enter"
                          size="small"
                        />
                      )}
                    />
                  </Box>
                )}
              </>
            )}
          </Stack>
        </DialogContent>
        <DialogActions>
          <Button
            onClick={() => setShareDialogOpen(false)}
            disabled={updateReportMutation.isPending}
          >
            Cancel
          </Button>
          <Button
            variant="contained"
            onClick={handleSaveShare}
            disabled={updateReportMutation.isPending}
            startIcon={updateReportMutation.isPending ? <CircularProgress size={16} color="inherit" /> : <ShareIcon />}
          >
            {updateReportMutation.isPending ? 'Saving...' : 'Save Settings'}
          </Button>
        </DialogActions>
      </Dialog>

      <Dialog open={scheduleDialogOpen} onClose={() => setScheduleDialogOpen(false)}>
        <DialogTitle>Schedule Report</DialogTitle>
        <DialogContent>
          <Typography>Schedule settings would go here.</Typography>
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setScheduleDialogOpen(false)}>Cancel</Button>
          <Button variant="contained" onClick={() => setScheduleDialogOpen(false)}>Save</Button>
        </DialogActions>
      </Dialog>

      <Dialog open={newFolderDialogOpen} onClose={() => setNewFolderDialogOpen(false)}>
        <DialogTitle>New Folder</DialogTitle>
        <DialogContent>
           <TextField
             margin="dense"
             label="Folder Name"
             fullWidth
             variant="outlined"
             value={newFolderName}
             onChange={(e) => setNewFolderName(e.target.value)}
           />
        </DialogContent>
        <DialogActions>
           <Button onClick={() => setNewFolderDialogOpen(false)}>Cancel</Button>
            <Button variant="contained" onClick={handleCreateFolder}>Create</Button>
        </DialogActions>
      </Dialog>

      {/* Rename Dialog */}
      <Dialog
        open={renameDialogOpen}
        onClose={() => !updateReportMutation.isPending && setRenameDialogOpen(false)}
        maxWidth="xs"
        fullWidth
      >
        <DialogTitle>Rename Report</DialogTitle>
        <DialogContent>
          {renameError && (
            <Alert severity="error" sx={{ mb: 2 }}>
              {renameError}
            </Alert>
          )}
          <TextField
            margin="dense"
            label="Report Name"
            fullWidth
            variant="outlined"
            value={renameValue}
            onChange={(e) => setRenameValue(e.target.value)}
            disabled={updateReportMutation.isPending}
            onKeyDown={(e) => {
              if (e.key === 'Enter' && renameValue.trim() && !updateReportMutation.isPending) {
                handleConfirmRename();
              }
            }}
          />
        </DialogContent>
        <DialogActions>
          <Button
            onClick={() => setRenameDialogOpen(false)}
            disabled={updateReportMutation.isPending}
          >
            Cancel
          </Button>
          <Button
            variant="contained"
            onClick={handleConfirmRename}
            disabled={updateReportMutation.isPending || !renameValue.trim()}
            startIcon={updateReportMutation.isPending ? <CircularProgress size={16} color="inherit" /> : <RenameIcon />}
          >
            {updateReportMutation.isPending ? 'Saving...' : 'Rename'}
          </Button>
        </DialogActions>
      </Dialog>

      {/* Duplicate / Clone Report Dialog */}
      <Dialog
        open={cloneDialogOpen}
        onClose={() => !createReportMutation.isPending && setCloneDialogOpen(false)}
        maxWidth="xs"
        fullWidth
      >
        <DialogTitle>Duplicate Report</DialogTitle>
        <DialogContent>
          {cloneError && (
            <Alert severity="error" sx={{ mb: 2 }}>
              {cloneError}
            </Alert>
          )}
          <TextField
            margin="dense"
            label="Report Name"
            fullWidth
            variant="outlined"
            value={cloneName}
            onChange={(e) => {
              setCloneName(e.target.value);
              setCloneError(null);
            }}
            disabled={createReportMutation.isPending}
            onKeyDown={(e) => {
              if (e.key === 'Enter' && cloneName.trim() && !createReportMutation.isPending) {
                handleConfirmClone();
              }
            }}
            sx={{ mb: 2 }}
          />
          {Boolean(currentTenant?.gold_copy) ? (
            <Alert severity="info" sx={{ mt: 1 }}>
              Duplicating in Master Gold Copy tenant creates a Core report template.
            </Alert>
          ) : (typeof isAdmin === 'function' ? isAdmin() : Boolean(isAdmin)) ? (
            <FormControl component="fieldset" sx={{ mt: 1.5, display: 'block' }}>
              <FormLabel component="legend" sx={{ fontSize: '0.875rem', fontWeight: 600, mb: 0.5 }}>
                Report Visibility
              </FormLabel>
              <RadioGroup
                value={cloneIsPersonal ? 'personal' : 'custom'}
                onChange={(e) => setCloneIsPersonal(e.target.value === 'personal')}
              >
                <FormControlLabel
                  value="personal"
                  control={<Radio size="small" />}
                  label={
                    <Box>
                      <Typography variant="body2" fontWeight={500}>
                        Personal Report
                      </Typography>
                      <Typography variant="caption" color="text.secondary">
                        Visible only to you until shared.
                      </Typography>
                    </Box>
                  }
                  sx={{ mb: 1 }}
                />
                <FormControlLabel
                  value="custom"
                  control={<Radio size="small" />}
                  label={
                    <Box>
                      <Typography variant="body2" fontWeight={500}>
                        Tenant Custom Report
                      </Typography>
                      <Typography variant="caption" color="text.secondary">
                        Visible to all users in this tenant.
                      </Typography>
                    </Box>
                  }
                />
              </RadioGroup>
            </FormControl>
          ) : (
            <Typography variant="caption" color="text.secondary" sx={{ display: 'block', mt: 1 }}>
              This report will be duplicated as your personal report.
            </Typography>
          )}
        </DialogContent>
        <DialogActions>
          <Button
            onClick={() => setCloneDialogOpen(false)}
            disabled={createReportMutation.isPending}
          >
            Cancel
          </Button>
          <Button
            variant="contained"
            onClick={handleConfirmClone}
            disabled={createReportMutation.isPending || !cloneName.trim()}
            startIcon={createReportMutation.isPending ? <CircularProgress size={16} color="inherit" /> : <DuplicateIcon />}
          >
            {createReportMutation.isPending ? 'Duplicating...' : 'Duplicate'}
          </Button>
        </DialogActions>
      </Dialog>

      {/* Notification Snackbar */}
      <Snackbar
        open={snackbar.open}
        autoHideDuration={4000}
        onClose={() => setSnackbar(prev => ({ ...prev, open: false }))}
        anchorOrigin={{ vertical: 'bottom', horizontal: 'right' }}
      >
        <Alert
          onClose={() => setSnackbar(prev => ({ ...prev, open: false }))}
          severity={snackbar.severity}
          sx={{ width: '100%' }}
        >
          {snackbar.message}
        </Alert>
      </Snackbar>
    </Box>
  );
};