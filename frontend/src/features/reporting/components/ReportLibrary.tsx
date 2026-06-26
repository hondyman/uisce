import React, { useState, useMemo } from 'react';
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
  GetApp as ExportIcon,
  Refresh as RefreshIcon,
  ViewList as ListViewIcon,
  ViewModule as GridViewIcon,
  AccessTime as RecentIcon,
  Person as PersonIcon,
  Group as GroupIcon,
  Public as PublicIcon,
} from '@mui/icons-material';
import { formatDistanceToNow } from 'date-fns';
import { devDebug } from '../../../utils/devLogger';
import { useReportTemplates } from '../../../api/reporting';
import { useFolders } from '../../../api/explorer';

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
  last_run?: string;
  run_count: number;
  config: any;
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
  
  // --- Real API Data ---
  const { data: apiReports, isLoading: isLoadingReports } = useReportTemplates();
  const { data: apiFolders, isLoading: isLoadingFolders } = useFolders();
  
  // Transform API data to component interfaces
  const reports = useMemo<SavedReport[]>(() => {
    if (!apiReports) return [];
    return apiReports.map(r => ({
      id: r.id,
      name: r.name,
      description: r.description || '',
      folder_id: (r.metadata as any)?.folder_id || undefined,
      created_by: (r.metadata as any)?.created_by || 'User',
      created_at: r.createdAt || new Date().toISOString(),
      updated_at: r.updatedAt || new Date().toISOString(),
      is_favorite: (r.metadata as any)?.is_favorite || false,
      is_scheduled: (r.metadata as any)?.is_scheduled || false,
      is_shared: (r.metadata as any)?.is_shared || false,
      share_type: (r.metadata as any)?.share_type || 'private',
      last_run: (r.metadata as any)?.last_run,
      run_count: (r.metadata as any)?.run_count || 0,
      config: r.definition || {},
    }));
  }, [apiReports]);

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
  const [filterType, setFilterType] = useState<'all' | 'favorites' | 'recent' | 'shared'>('all');
  const [selectedReport, setSelectedReport] = useState<SavedReport | null>(null);
  const [menuAnchor, setMenuAnchor] = useState<null | HTMLElement>(null);
  const [shareDialogOpen, setShareDialogOpen] = useState(false);
  const [scheduleDialogOpen, setScheduleDialogOpen] = useState(false);
  const [newFolderDialogOpen, setNewFolderDialogOpen] = useState(false);
  const [newFolderName, setNewFolderName] = useState('');
  
  const isLoading = isLoadingReports || isLoadingFolders;



  const handleMenuOpen = (event: React.MouseEvent<HTMLElement>, report: SavedReport) => {
    setSelectedReport(report);
    setMenuAnchor(event.currentTarget);
  };

  const handleMenuClose = () => {
    setMenuAnchor(null);
  };

  const handleToggleFavorite = async (reportId: string) => {
    // TODO: Call API to toggle favorite status
    devDebug('Toggle favorite for:', reportId);
  };

  const handleCreateFolder = async () => {
    if (!newFolderName.trim()) return;
    // TODO: Call API to create folder
    setNewFolderDialogOpen(false);
    setNewFolderName('');
  };

  const handleRunReport = (report: SavedReport) => {
    // Navigate to report builder in preview mode
    navigate(`/reports/${report.id}/edit`);
    handleMenuClose();
  };

  const handleEditReport = (report: SavedReport) => {
    navigate(`/reports/${report.id}/edit`);
    handleMenuClose();
  };

  const handleDuplicateReport = async (report: SavedReport) => {
    // TODO: Call API to duplicate
    handleMenuClose();
  };

  const handleDeleteReport = async (report: SavedReport) => {
    // TODO: Call API to delete
    handleMenuClose();
  };

  const filteredReports = reports.filter(report => {
    // Filter by folder
    if (currentFolder && report.folder_id !== currentFolder) return false;

    // Filter by type
    if (filterType === 'favorites' && !report.is_favorite) return false;
    if (filterType === 'shared' && !report.is_shared) return false;
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
        <Grid item xs={12} md={3}>
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
                onClick={() => setFilterType('favorites')}
              >
                <ListItemIcon><StarIcon /></ListItemIcon>
                <ListItemText primary="Favorites" />
              </ListItemButton>
              <ListItemButton
                selected={filterType === 'recent'}
                onClick={() => setFilterType('recent')}
              >
                <ListItemIcon><RecentIcon /></ListItemIcon>
                <ListItemText primary="Recent" />
              </ListItemButton>
              <ListItemButton
                selected={filterType === 'shared'}
                onClick={() => setFilterType('shared')}
              >
                <ListItemIcon><ShareIcon /></ListItemIcon>
                <ListItemText primary="Shared" />
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
        <Grid item xs={12} md={9}>
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
              <ToggleButton value="list"><ListViewIcon /></ToggleButton>
              <ToggleButton value="grid"><GridViewIcon /></ToggleButton>
            </ToggleButtonGroup>
          </Box>

          {/* Reports List/Grid */}
          {reports.length === 0 ? (
            <Paper sx={{ p: 4, textAlign: 'center' }}>
              <Typography color="text.secondary">No reports found.</Typography>
            </Paper>
          ) : viewMode === 'list' ? (
            <Card>
              <List>
                {filteredReports.map(report => (
                  <React.Fragment key={report.id}>
                    <ListItem
                      secondaryAction={
                        <IconButton onClick={(e) => handleMenuOpen(e, report)}>
                          <MoreIcon />
                        </IconButton>
                      }
                    >
                      <ListItemButton onClick={() => handleRunReport(report)}>
                        <ListItemIcon>
                           {/* Add logic for icon based on chart type if needed */}
                           <ReportIcon />
                        </ListItemIcon>
                        <ListItemText
                          primary={
                            <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                              {report.name}
                              {report.is_favorite && <StarIcon fontSize="small" color="warning" />}
                              <Chip
                                size="small"
                                icon={getShareIcon(report.share_type)}
                                label={report.share_type === 'private' ? 'Private' : report.share_type === 'team' ? 'Team' : 'Public'}
                                variant="outlined"
                              />
                            </Box>
                          }
                          secondary={
                            <Stack direction="row" spacing={2} component="span" sx={{ mt: 0.5 }}>
                              <Typography variant="caption" component="span">
                                {report.run_count} runs
                              </Typography>
                              <Typography variant="caption" component="span">
                                Last run: {report.last_run ? formatDistanceToNow(new Date(report.last_run)) : 'Never'} ago
                              </Typography>
                              <Typography variant="caption" component="span">
                                By {report.created_by}
                              </Typography>
                            </Stack>
                          }
                        />
                      </ListItemButton>
                    </ListItem>
                    <Divider component="li" />
                  </React.Fragment>
                ))}
              </List>
            </Card>
          ) : (
            <Grid container spacing={2}>
              {filteredReports.map(report => (
                <Grid item xs={12} sm={6} md={4} key={report.id}>
                  <Card>
                    <CardContent>
                      <Box sx={{ display: 'flex', justifyContent: 'space-between', mb: 2 }}>
                        <Avatar sx={{ bgcolor: 'primary.light' }}>
                          <ReportIcon />
                        </Avatar>
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
                      <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                        <Chip
                           size="small"
                           icon={getShareIcon(report.share_type)}
                           label={report.share_type || 'Private'}
                        />
                         <IconButton size="small" onClick={(e) => handleMenuOpen(e, report)}>
                           <MoreIcon />
                         </IconButton>
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
        <MenuItem onClick={() => setShareDialogOpen(true)}>
          <ListItemIcon><ShareIcon fontSize="small" /></ListItemIcon>
          Share
        </MenuItem>
        <MenuItem onClick={() => setScheduleDialogOpen(true)}>
          <ListItemIcon><ScheduleIcon fontSize="small" /></ListItemIcon>
          Schedule
        </MenuItem>
        <MenuItem onClick={() => handleEditReport(selectedReport!)}>
          <ListItemIcon><EditIcon fontSize="small" /></ListItemIcon>
          Edit
        </MenuItem>
        <MenuItem onClick={() => handleDuplicateReport(selectedReport!)}>
          <ListItemIcon><DuplicateIcon fontSize="small" /></ListItemIcon>
          Duplicate
        </MenuItem>
        <Divider />
        <MenuItem onClick={() => handleDeleteReport(selectedReport!)} sx={{ color: 'error.main' }}>
          <ListItemIcon><DeleteIcon fontSize="small" color="error" /></ListItemIcon>
          Delete
        </MenuItem>
      </Menu>

      <Dialog open={shareDialogOpen} onClose={() => setShareDialogOpen(false)}>
        <DialogTitle>Share Report</DialogTitle>
        <DialogContent>
          <Typography>Share settings would go here.</Typography>
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setShareDialogOpen(false)}>Cancel</Button>
          <Button variant="contained" onClick={() => setShareDialogOpen(false)}>Save</Button>
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
             autoFocus
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
    </Box>
  );
};