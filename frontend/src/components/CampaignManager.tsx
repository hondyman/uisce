import { useState, useEffect, useCallback } from 'react';
import { devError } from '../utils/devLogger';
import {
  Box,
  Typography,
  Paper,
  Button,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  IconButton,
  Chip,
  Dialog,
  DialogContent,
  DialogActions,
  TextField,
  FormControl,
  InputLabel,
  Select,
  MenuItem,
  Grid,
  Card,
  CardContent,
  CardActions as _CardActions,
  LinearProgress,
  Alert as _Alert,
  Tabs as _Tabs,
  Tab as _Tab,
} from '@mui/material';
import ModalHeader from './ModalHeader';
import {
  Add as AddIcon,
  PlayArrow as PlayIcon,
  Pause as PauseIcon,
  Stop as StopIcon,
  Edit as _EditIcon,
  Analytics as AnalyticsIcon,
} from '@mui/icons-material';
import { useNotificationAPI, NotificationCampaign, CampaignAnalytics } from '../hooks/useNotificationAPI';

// TabPanel helper intentionally removed — not used in this component

export default function CampaignManager() {
  const [campaigns, setCampaigns] = useState<NotificationCampaign[]>([]);
  const [loading, setLoading] = useState(true);
  const [selectedCampaign, setSelectedCampaign] = useState<NotificationCampaign | null>(null);
  const [analytics, setAnalytics] = useState<CampaignAnalytics | null>(null);
  const [createDialogOpen, setCreateDialogOpen] = useState(false);
  const [analyticsDialogOpen, setAnalyticsDialogOpen] = useState(false);
  const [_tabValue, _setTabValue] = useState(0);

  const {
    getActiveCampaigns,
    createCampaign,
    launchCampaign,
    pauseCampaign,
    resumeCampaign,
    stopCampaign,
    getCampaignAnalytics,
  } = useNotificationAPI();

  const loadCampaigns = useCallback(async () => {
    try {
      setLoading(true);
      const data = await getActiveCampaigns();
      setCampaigns(data);
    } catch (error) {
      try { devError('Failed to load campaigns:', error); } catch {}
    } finally {
      setLoading(false);
    }
  }, [getActiveCampaigns]);

  useEffect(() => {
    loadCampaigns();
  }, [loadCampaigns]);

  const handleCreateCampaign = async (campaignData: Partial<NotificationCampaign>) => {
    try {
      const newCampaign = await createCampaign({
        id: '',
        name: campaignData.name || '',
        description: campaignData.description || '',
        type: campaignData.type || 'onboarding',
        status: 'draft',
        target_users: campaignData.target_users || [],
        user_segment: campaignData.user_segment || '',
        steps: campaignData.steps || [],
        created_by: 'current-user', // TODO: Get from auth context
        created_at: new Date().toISOString(),
        updated_at: new Date().toISOString(),
      });
      setCampaigns([...campaigns, newCampaign]);
      setCreateDialogOpen(false);
    } catch (error) {
      try { devError('Failed to create campaign:', error); } catch {}
    }
  };

  const handleCampaignAction = async (campaignId: string, action: 'launch' | 'pause' | 'resume' | 'stop') => {
    try {
      switch (action) {
        case 'launch':
          await launchCampaign(campaignId);
          break;
        case 'pause':
          await pauseCampaign(campaignId);
          break;
        case 'resume':
          await resumeCampaign(campaignId);
          break;
        case 'stop':
          await stopCampaign(campaignId);
          break;
      }
      await loadCampaigns(); // Refresh the list
    } catch (error) {
      try { devError(`Failed to ${action} campaign:`, error); } catch {}
    }
  };

  const handleViewAnalytics = async (campaign: NotificationCampaign) => {
    try {
      const data = await getCampaignAnalytics(campaign.id);
      setAnalytics(data);
      setSelectedCampaign(campaign);
      setAnalyticsDialogOpen(true);
    } catch (error) {
      try { devError('Failed to load campaign analytics:', error); } catch {}
    }
  };

  const getStatusColor = (status: string) => {
    switch (status) {
      case 'active': return 'success';
      case 'paused': return 'warning';
      case 'completed': return 'info';
      case 'draft': return 'default';
      default: return 'error';
    }
  };

  // status icon helper removed — not referenced

  if (loading) {
    return (
      <Box sx={{ p: 3 }}>
        <LinearProgress />
        <Typography sx={{ mt: 2 }}>Loading campaigns...</Typography>
      </Box>
    );
  }

  return (
    <Box sx={{ p: 3 }}>
      <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', mb: 3 }}>
        <Typography variant="h4" component="h1">
          Notification Campaigns
        </Typography>
        <Button
          variant="contained"
          startIcon={<AddIcon />}
          onClick={() => setCreateDialogOpen(true)}
        >
          Create Campaign
        </Button>
      </Box>

      <Paper sx={{ width: '100%', mb: 2 }}>
        <TableContainer>
          <Table>
            <TableHead>
              <TableRow>
                <TableCell>Name</TableCell>
                <TableCell>Type</TableCell>
                <TableCell>Status</TableCell>
                <TableCell>Target Users</TableCell>
                <TableCell>Steps</TableCell>
                <TableCell>Created</TableCell>
                <TableCell>Actions</TableCell>
              </TableRow>
            </TableHead>
            <TableBody>
              {campaigns.map((campaign) => (
                <TableRow key={campaign.id} hover>
                  <TableCell>
                    <Typography variant="subtitle2">{campaign.name}</Typography>
                    <Typography variant="body2" color="text.secondary">
                      {campaign.description}
                    </Typography>
                  </TableCell>
                  <TableCell>
                    <Chip label={campaign.type} size="small" variant="outlined" />
                  </TableCell>
                  <TableCell>
                    <Chip
                      label={campaign.status}
                      size="small"
                      color={getStatusColor(campaign.status)}
                    />
                  </TableCell>
                  <TableCell>
                    {campaign.target_users?.length || 0} users
                    {campaign.user_segment && (
                      <Typography variant="caption" display="block">
                        Segment: {campaign.user_segment}
                      </Typography>
                    )}
                  </TableCell>
                  <TableCell>{campaign.steps?.length || 0} steps</TableCell>
                  <TableCell>
                    {new Date(campaign.created_at).toLocaleDateString()}
                  </TableCell>
                  <TableCell>
                    <Box sx={{ display: 'flex', gap: 1 }}>
                      {campaign.status === 'draft' && (
                        <IconButton
                          size="small"
                          onClick={() => handleCampaignAction(campaign.id, 'launch')}
                          color="success"
                          title="Launch Campaign"
                        >
                          <PlayIcon />
                        </IconButton>
                      )}
                      {campaign.status === 'active' && (
                        <IconButton
                          size="small"
                          onClick={() => handleCampaignAction(campaign.id, 'pause')}
                          color="warning"
                          title="Pause Campaign"
                        >
                          <PauseIcon />
                        </IconButton>
                      )}
                      {campaign.status === 'paused' && (
                        <>
                          <IconButton
                            size="small"
                            onClick={() => handleCampaignAction(campaign.id, 'resume')}
                            color="success"
                            title="Resume Campaign"
                          >
                            <PlayIcon />
                          </IconButton>
                          <IconButton
                            size="small"
                            onClick={() => handleCampaignAction(campaign.id, 'stop')}
                            color="error"
                            title="Stop Campaign"
                          >
                            <StopIcon />
                          </IconButton>
                        </>
                      )}
                      <IconButton
                        size="small"
                        onClick={() => handleViewAnalytics(campaign)}
                        color="info"
                        title="View Analytics"
                      >
                        <AnalyticsIcon />
                      </IconButton>
                    </Box>
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </TableContainer>
      </Paper>

      {/* Create Campaign Dialog */}
      <CreateCampaignDialog
        open={createDialogOpen}
        onClose={() => setCreateDialogOpen(false)}
        onCreate={handleCreateCampaign}
      />

      {/* Analytics Dialog */}
      <AnalyticsDialog
        open={analyticsDialogOpen}
        onClose={() => setAnalyticsDialogOpen(false)}
        campaign={selectedCampaign}
        analytics={analytics}
      />
    </Box>
  );
}

// Create Campaign Dialog Component
interface CreateCampaignDialogProps {
  open: boolean;
  onClose: () => void;
  onCreate: (campaign: Partial<NotificationCampaign>) => void;
}

function CreateCampaignDialog({ open, onClose, onCreate }: CreateCampaignDialogProps) {
  const [formData, setFormData] = useState({
    name: '',
    description: '',
    type: 'onboarding',
    user_segment: '',
  });

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    onCreate(formData);
    setFormData({ name: '', description: '', type: 'onboarding', user_segment: '' });
  };

  return (
    <Dialog open={open} onClose={onClose} maxWidth="md" fullWidth>
      <ModalHeader title="Create Notification Campaign" onClose={onClose} />
      <form onSubmit={handleSubmit}>
        <DialogContent>
          <Grid container spacing={2}>
            <Grid item xs={12}>
              <TextField
                fullWidth
                label="Campaign Name"
                value={formData.name}
                onChange={(e) => setFormData({ ...formData, name: e.target.value })}
                required
              />
            </Grid>
            <Grid item xs={12}>
              <TextField
                fullWidth
                label="Description"
                value={formData.description}
                onChange={(e) => setFormData({ ...formData, description: e.target.value })}
                multiline
                rows={3}
              />
            </Grid>
            <Grid item xs={6}>
              <FormControl fullWidth>
                <InputLabel>Campaign Type</InputLabel>
                <Select
                  value={formData.type}
                  onChange={(e) => setFormData({ ...formData, type: e.target.value })}
                >
                  <MenuItem value="onboarding">Onboarding</MenuItem>
                  <MenuItem value="feature_adoption">Feature Adoption</MenuItem>
                  <MenuItem value="re_engagement">Re-engagement</MenuItem>
                  <MenuItem value="promotional">Promotional</MenuItem>
                </Select>
              </FormControl>
            </Grid>
            <Grid item xs={6}>
              <TextField
                fullWidth
                label="User Segment (optional)"
                value={formData.user_segment}
                onChange={(e) => setFormData({ ...formData, user_segment: e.target.value })}
                placeholder="e.g., premium_users, new_users"
              />
            </Grid>
          </Grid>
        </DialogContent>
        <DialogActions>
          <Button onClick={onClose}>Cancel</Button>
          <Button type="submit" variant="contained">Create Campaign</Button>
        </DialogActions>
      </form>
    </Dialog>
  );
}

// Analytics Dialog Component
interface AnalyticsDialogProps {
  open: boolean;
  onClose: () => void;
  campaign: NotificationCampaign | null;
  analytics: CampaignAnalytics | null;
}

function AnalyticsDialog({ open, onClose, campaign, analytics }: AnalyticsDialogProps) {
  if (!campaign || !analytics) return null;

  return (
    <Dialog open={open} onClose={onClose} maxWidth="md" fullWidth>
      <ModalHeader title={`Campaign Analytics`} subtitle={campaign.name} onClose={onClose} />
      <DialogContent>
        <Grid container spacing={2}>
          <Grid item xs={6}>
            <Card>
              <CardContent>
                <Typography variant="h6" color="primary">
                  {analytics.total_sent}
                </Typography>
                <Typography variant="body2">Total Sent</Typography>
              </CardContent>
            </Card>
          </Grid>
          <Grid item xs={6}>
            <Card>
              <CardContent>
                <Typography variant="h6" color="success.main">
                  {analytics.total_opened}
                </Typography>
                <Typography variant="body2">Total Opened</Typography>
              </CardContent>
            </Card>
          </Grid>
          <Grid item xs={6}>
            <Card>
              <CardContent>
                <Typography variant="h6" color="info.main">
                  {analytics.total_clicked}
                </Typography>
                <Typography variant="body2">Total Clicked</Typography>
              </CardContent>
            </Card>
          </Grid>
          <Grid item xs={6}>
            <Card>
              <CardContent>
                <Typography variant="h6" color="warning.main">
                  {analytics.total_converted}
                </Typography>
                <Typography variant="body2">Total Converted</Typography>
              </CardContent>
            </Card>
          </Grid>
          <Grid item xs={12}>
            <Typography variant="h6" gutterBottom>
              Performance Metrics
            </Typography>
            <Box sx={{ display: 'flex', gap: 2, mb: 2 }}>
              <Chip label={`Open Rate: ${(analytics.open_rate * 100).toFixed(1)}%`} color="success" />
              <Chip label={`Click Rate: ${(analytics.click_rate * 100).toFixed(1)}%`} color="info" />
              <Chip label={`Conversion Rate: ${(analytics.conversion_rate * 100).toFixed(1)}%`} color="warning" />
            </Box>
          </Grid>
          {analytics.step_performance && analytics.step_performance.length > 0 && (
            <Grid item xs={12}>
              <Typography variant="h6" gutterBottom>
                Step Performance
              </Typography>
              <TableContainer>
                <Table size="small">
                  <TableHead>
                    <TableRow>
                      <TableCell>Step</TableCell>
                      <TableCell>Sent</TableCell>
                      <TableCell>Open Rate</TableCell>
                      <TableCell>Click Rate</TableCell>
                      <TableCell>Conversion</TableCell>
                    </TableRow>
                  </TableHead>
                  <TableBody>
                    {analytics.step_performance.map((step) => (
                      <TableRow key={step.step_number}>
                        <TableCell>{step.step_number}</TableCell>
                        <TableCell>{step.sent_count}</TableCell>
                        <TableCell>{(step.open_rate * 100).toFixed(1)}%</TableCell>
                        <TableCell>{(step.click_rate * 100).toFixed(1)}%</TableCell>
                        <TableCell>{(step.conversion_rate * 100).toFixed(1)}%</TableCell>
                      </TableRow>
                    ))}
                  </TableBody>
                </Table>
              </TableContainer>
            </Grid>
          )}
        </Grid>
      </DialogContent>
      <DialogActions>
        <Button onClick={onClose}>Close</Button>
      </DialogActions>
    </Dialog>
  );
}
