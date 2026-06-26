import { useState, useEffect, useCallback } from 'react';
import { devError } from '../utils/devLogger';
import {
  Box,
  Typography,
  Paper,
  Grid,
  Card,
  CardContent,
  CardActions,
  Button,
  IconButton,
  Dialog,
  DialogContent,
  DialogActions,
  // TextField removed: previously unused in this component
  Chip,
  Menu,
  MenuItem,
  Fab,
  CircularProgress,
  Alert,
} from '@mui/material';
import ModalHeader from './ModalHeader';
import {
  Add as AddIcon,
  Edit as EditIcon,
  Delete as DeleteIcon,
  MoreVert as MoreVertIcon,
  Share as ShareIcon,
  ContentCopy as DuplicateIcon,
  Public as PublicIcon,
  Lock as PrivateIcon,
} from '@mui/icons-material';
import { CustomDashboardBuilder, Dashboard } from './CustomDashboardBuilder';
import { useDashboardService } from '../hooks/useDashboardService';

interface DashboardManagerProps {
  userId: string;
  onDashboardSelect?: (dashboard: Dashboard) => void;
}

export const DashboardManager: React.FC<DashboardManagerProps> = ({
  userId,
  onDashboardSelect,
}) => {
  const [dashboards, setDashboards] = useState<Dashboard[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [builderOpen, setBuilderOpen] = useState(false);
  const [editingDashboard, setEditingDashboard] = useState<Dashboard | null>(null);
  const [deleteDialogOpen, setDeleteDialogOpen] = useState(false);
  const [dashboardToDelete, setDashboardToDelete] = useState<Dashboard | null>(null);
  const [menuAnchor, setMenuAnchor] = useState<null | HTMLElement>(null);
  const [selectedDashboard, setSelectedDashboard] = useState<Dashboard | null>(null);

  const {
    getDashboards,
    saveDashboard,
    deleteDashboard,
    duplicateDashboard,
  } = useDashboardService();

  const loadDashboards = useCallback(async () => {
    try {
      setLoading(true);
      const data = await getDashboards(userId);
      setDashboards(data);
      setError(null);
    } catch (err) {
      setError('Failed to load dashboards');
      try { devError('Error loading dashboards:', err); } catch {}
    } finally {
      setLoading(false);
    }
  }, [getDashboards, userId]);

  useEffect(() => {
    loadDashboards();
  }, [loadDashboards]);

  const handleCreateDashboard = () => {
    setEditingDashboard(null);
    setBuilderOpen(true);
  };

  const handleEditDashboard = (dashboard: Dashboard) => {
    setEditingDashboard(dashboard);
    setBuilderOpen(true);
  };

  const handleSaveDashboard = async (dashboard: Dashboard) => {
    try {
      await saveDashboard(dashboard);
      await loadDashboards();
      setBuilderOpen(false);
      setEditingDashboard(null);
    } catch (err) {
      try { devError('Error saving dashboard:', err); } catch {}
      setError('Failed to save dashboard');
    }
  };

  const handleDeleteDashboard = async () => {
    if (!dashboardToDelete) return;

    try {
      await deleteDashboard(dashboardToDelete.id);
      await loadDashboards();
      setDeleteDialogOpen(false);
      setDashboardToDelete(null);
    } catch (err) {
      try { devError('Error deleting dashboard:', err); } catch {}
      setError('Failed to delete dashboard');
    }
  };

  const handleDuplicateDashboard = async (dashboard: Dashboard) => {
    try {
      const newName = `${dashboard.name} (Copy)`;
      await duplicateDashboard(dashboard.id, newName);
      await loadDashboards();
    } catch (err) {
      try { devError('Error duplicating dashboard:', err); } catch {}
      setError('Failed to duplicate dashboard');
    }
  };

  const handleMenuOpen = (event: React.MouseEvent<HTMLElement>, dashboard: Dashboard) => {
    setMenuAnchor(event.currentTarget);
    setSelectedDashboard(dashboard);
  };

  const handleMenuClose = () => {
    setMenuAnchor(null);
    setSelectedDashboard(null);
  };

  const handleShareDashboard = (dashboard: Dashboard) => {
    // Implement sharing logic
    const shareUrl = `${window.location.origin}/dashboard/${dashboard.id}`;
    navigator.clipboard.writeText(shareUrl);
    handleMenuClose();
  };

  if (loading) {
    return (
      <Box sx={{ display: 'flex', justifyContent: 'center', p: 4 }}>
        <CircularProgress />
      </Box>
    );
  }

  return (
    <Box sx={{ p: 3 }}>
      <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', mb: 3 }}>
        <Typography variant="h4">
          My Dashboards
        </Typography>
        <Button
          variant="contained"
          startIcon={<AddIcon />}
          onClick={handleCreateDashboard}
        >
          Create Dashboard
        </Button>
      </Box>

      {error && (
        <Alert severity="error" sx={{ mb: 3 }}>
          {error}
        </Alert>
      )}

      {dashboards.length === 0 ? (
        <Paper sx={{ p: 4, textAlign: 'center' }}>
          <Typography variant="h6" gutterBottom>
            No dashboards yet
          </Typography>
          <Typography variant="body2" color="text.secondary" sx={{ mb: 3 }}>
            Create your first dashboard to get started with data visualization
          </Typography>
          <Button
            variant="contained"
            startIcon={<AddIcon />}
            onClick={handleCreateDashboard}
          >
            Create Your First Dashboard
          </Button>
        </Paper>
      ) : (
        <Grid container spacing={3}>
          {dashboards.map((dashboard) => (
            <Grid item xs={12} sm={6} md={4} key={dashboard.id}>
              <Card sx={{ height: '100%', display: 'flex', flexDirection: 'column' }}>
                <CardContent sx={{ flex: 1 }}>
                  <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start', mb: 2 }}>
                    <Typography variant="h6" sx={{ flex: 1, mr: 1 }}>
                      {dashboard.name}
                    </Typography>
                    <IconButton
                      size="small"
                      onClick={(e) => handleMenuOpen(e, dashboard)}
                    >
                      <MoreVertIcon />
                    </IconButton>
                  </Box>

                  {dashboard.description && (
                    <Typography variant="body2" color="text.secondary" sx={{ mb: 2 }}>
                      {dashboard.description}
                    </Typography>
                  )}

                  <Box sx={{ display: 'flex', gap: 1, mb: 2 }}>
                    <Chip
                      size="small"
                      label={`${dashboard.widgets.length} widgets`}
                      variant="outlined"
                    />
                    <Chip
                      size="small"
                      label={dashboard.layout}
                      variant="outlined"
                    />
                    {dashboard.isPublic ? (
                      <Chip
                        size="small"
                        icon={<PublicIcon />}
                        label="Public"
                        color="success"
                        variant="outlined"
                      />
                    ) : (
                      <Chip
                        size="small"
                        icon={<PrivateIcon />}
                        label="Private"
                        variant="outlined"
                      />
                    )}
                  </Box>

                  <Typography variant="caption" color="text.secondary">
                    Updated {new Date(dashboard.updatedAt).toLocaleDateString()}
                  </Typography>
                </CardContent>

                <CardActions sx={{ justifyContent: 'space-between' }}>
                  <Button
                    size="small"
                    onClick={() => onDashboardSelect?.(dashboard)}
                  >
                    View
                  </Button>
                  <Box>
                    <IconButton
                      size="small"
                      onClick={() => handleEditDashboard(dashboard)}
                    >
                      <EditIcon />
                    </IconButton>
                    <IconButton
                      size="small"
                      onClick={() => handleDuplicateDashboard(dashboard)}
                    >
                      <DuplicateIcon />
                    </IconButton>
                  </Box>
                </CardActions>
              </Card>
            </Grid>
          ))}
        </Grid>
      )}

      {/* Floating Action Button */}
      <Fab
        color="primary"
        sx={{ position: 'fixed', bottom: 16, right: 16 }}
        onClick={handleCreateDashboard}
      >
        <AddIcon />
      </Fab>

      {/* Dashboard Builder Dialog */}
      <Dialog
        open={builderOpen}
        onClose={() => setBuilderOpen(false)}
        maxWidth="xl"
        fullWidth
        fullScreen
      >
        <CustomDashboardBuilder
          initialDashboard={editingDashboard || undefined}
          availableWidgets={[]} // This would be populated with actual widget types
          onSave={handleSaveDashboard}
          onCancel={() => {
            setBuilderOpen(false);
            setEditingDashboard(null);
          }}
          userId={userId}
        />
      </Dialog>

      {/* Delete Confirmation Dialog */}
      <Dialog open={deleteDialogOpen} onClose={() => setDeleteDialogOpen(false)}>
  <ModalHeader title="Delete Dashboard" onClose={() => setDeleteDialogOpen(false)} />
        <DialogContent>
          <Typography>
            Are you sure you want to delete "{dashboardToDelete?.name}"?
            This action cannot be undone.
          </Typography>
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setDeleteDialogOpen(false)}>Cancel</Button>
          <Button onClick={handleDeleteDashboard} color="error" variant="contained">
            Delete
          </Button>
        </DialogActions>
      </Dialog>

      {/* Context Menu */}
      <Menu
        anchorEl={menuAnchor}
        open={Boolean(menuAnchor)}
        onClose={handleMenuClose}
      >
        <MenuItem onClick={() => selectedDashboard && handleEditDashboard(selectedDashboard)}>
          <EditIcon sx={{ mr: 1 }} />
          Edit
        </MenuItem>
        <MenuItem onClick={() => selectedDashboard && handleDuplicateDashboard(selectedDashboard)}>
          <DuplicateIcon sx={{ mr: 1 }} />
          Duplicate
        </MenuItem>
        <MenuItem onClick={() => selectedDashboard && handleShareDashboard(selectedDashboard)}>
          <ShareIcon sx={{ mr: 1 }} />
          Share
        </MenuItem>
        <MenuItem
          onClick={() => {
            setDashboardToDelete(selectedDashboard);
            setDeleteDialogOpen(true);
            handleMenuClose();
          }}
          sx={{ color: 'error.main' }}
        >
          <DeleteIcon sx={{ mr: 1 }} />
          Delete
        </MenuItem>
      </Menu>
    </Box>
  );
};
