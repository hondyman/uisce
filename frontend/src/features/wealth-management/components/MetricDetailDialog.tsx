// no React hooks used directly here
import {
  Dialog,
  DialogContent,
  DialogActions,
  Button,
  Typography,
  Box,
  Chip,
  Grid,
  Paper,
  List,
  ListItem,
  ListItemText,
  ListItemIcon,
  Alert,
} from '@mui/material';
import ModalHeader from '../../../components/ModalHeader';
import {
  TrendingUp as TrendingUpIcon,
  Assessment as AssessmentIcon,
  AccountBalance as AccountBalanceIcon,
  PieChart as PieChartIcon,
  Timeline as TimelineIcon,
  ShowChart as ShowChartIcon,
  Calculate as CalculateIcon,
  Schedule as ScheduleIcon,
  CheckCircle as CheckCircleIcon,
  Info as InfoIcon,
  Code as CodeIcon,
  // Close icon not used here
} from '@mui/icons-material';

interface WealthManagementMetric {
  node_id: string;
  category: string;
  description: string;
  governance_status: 'golden' | 'draft';
  formula_type: string;
  formula?: string;
  arguments?: Record<string, string>;
  audience: string[];
  tags: string[];
  created_at?: string;
  updated_at?: string;
}

interface MetricDetailDialogProps {
  open: boolean;
  onClose: () => void;
  metric: WealthManagementMetric | null;
}

const MetricDetailDialog: React.FC<MetricDetailDialogProps> = ({
  open,
  onClose,
  metric
}) => {
  if (!metric) return null;

  const getCategoryIcon = (category: string) => {
    switch (category) {
      case 'performance':
        return <TrendingUpIcon color="primary" />;
      case 'risk':
      case 'risk_adjusted_performance':
        return <AssessmentIcon color="warning" />;
      case 'composition':
        return <PieChartIcon color="success" />;
      case 'income':
        return <AccountBalanceIcon color="success" />;
      case 'client_kpi':
        return <TimelineIcon color="info" />;
      case 'business_efficiency':
        return <ShowChartIcon color="secondary" />;
      default:
        return <CalculateIcon />;
    }
  };

  const getGovernanceIcon = (status: string) => {
    return status === 'golden' ?
      <CheckCircleIcon color="success" /> :
      <InfoIcon color="warning" />;
  };

  const getGovernanceDescription = (status: string) => {
    return status === 'golden'
      ? 'This metric has been reviewed and approved for production use. It follows governance standards and is actively monitored.'
      : 'This metric is in draft status and requires governance review before production use.';
  };

  return (
    <Dialog
      open={open}
      onClose={onClose}
      maxWidth="md"
      fullWidth
      PaperProps={{
        sx: { minHeight: '600px' }
      }}
    >
      <ModalHeader title={<Box display="flex" alignItems="center">{getCategoryIcon(metric.category)}<Typography variant="h6" sx={{ ml: 1 }}>{metric.node_id.replace(/_/g, ' ').replace(/\b\w/g, l => l.toUpperCase())}</Typography></Box>} onClose={onClose} />

      <DialogContent dividers>
        <Grid container spacing={3}>
          {/* Basic Information */}
          <Grid item xs={12}>
            <Typography variant="body1" paragraph>
              {metric.description}
            </Typography>
          </Grid>

          {/* Status and Category */}
          <Grid item xs={12} md={6}>
            <Paper sx={{ p: 2 }}>
              <Typography variant="h6" gutterBottom>
                Status & Category
              </Typography>
              <Box display="flex" alignItems="center" mb={2}>
                {getGovernanceIcon(metric.governance_status)}
                <Chip
                  label={metric.governance_status.toUpperCase()}
                  color={metric.governance_status === 'golden' ? 'success' : 'warning'}
                  sx={{ ml: 1 }}
                />
              </Box>
              <Typography variant="body2" color="text.secondary" paragraph>
                {getGovernanceDescription(metric.governance_status)}
              </Typography>
              <Chip
                label={metric.category.replace(/_/g, ' ').toUpperCase()}
                color="primary"
                variant="outlined"
              />
            </Paper>
          </Grid>

          {/* Technical Details */}
          <Grid item xs={12} md={6}>
            <Paper sx={{ p: 2 }}>
              <Typography variant="h6" gutterBottom>
                Technical Details
              </Typography>
              <List dense>
                <ListItem>
                  <ListItemIcon>
                    <CodeIcon />
                  </ListItemIcon>
                  <ListItemText
                    primary="Formula Type"
                    secondary={metric.formula_type}
                  />
                </ListItem>
                <ListItem>
                  <ListItemIcon>
                    <ScheduleIcon />
                  </ListItemIcon>
                  <ListItemText
                    primary="Auto-Refresh"
                    secondary={metric.governance_status === 'golden' ? 'Enabled' : 'Pending Approval'}
                  />
                </ListItem>
              </List>
            </Paper>
          </Grid>

          {/* Formula (if available) */}
          {metric.formula && (
            <Grid item xs={12}>
              <Paper sx={{ p: 2 }}>
                <Typography variant="h6" gutterBottom>
                  Calculation Formula
                </Typography>
                <Box
                  sx={{
                    bgcolor: 'grey.100',
                    p: 2,
                    borderRadius: 1,
                    fontFamily: 'monospace',
                    fontSize: '0.875rem'
                  }}
                >
                  {metric.formula}
                </Box>
                {metric.arguments && Object.keys(metric.arguments).length > 0 && (
                  <Box mt={2}>
                    <Typography variant="subtitle2" gutterBottom>
                      Arguments:
                    </Typography>
                    <List dense>
                      {Object.entries(metric.arguments).map(([key, value]) => (
                        <ListItem key={key}>
                          <ListItemText
                            primary={key}
                            secondary={value}
                          />
                        </ListItem>
                      ))}
                    </List>
                  </Box>
                )}
              </Paper>
            </Grid>
          )}

          {/* Audience and Tags */}
          <Grid item xs={12} md={6}>
            <Paper sx={{ p: 2 }}>
              <Typography variant="h6" gutterBottom>
                Target Audience
              </Typography>
              <Box>
                {metric.audience.map((audience) => (
                  <Chip
                    key={audience}
                    label={audience.charAt(0).toUpperCase() + audience.slice(1)}
                    size="small"
                    color="primary"
                    sx={{ mr: 1, mb: 1 }}
                  />
                ))}
              </Box>
            </Paper>
          </Grid>

          <Grid item xs={12} md={6}>
            <Paper sx={{ p: 2 }}>
              <Typography variant="h6" gutterBottom>
                Tags
              </Typography>
              <Box>
                {metric.tags.map((tag) => (
                  <Chip
                    key={tag}
                    label={tag.replace(/_/g, ' ')}
                    size="small"
                    variant="outlined"
                    sx={{ mr: 1, mb: 1 }}
                  />
                ))}
              </Box>
            </Paper>
          </Grid>

          {/* Usage Information */}
          <Grid item xs={12}>
            <Alert severity="info">
              <Typography variant="body2">
                <strong>Usage:</strong> This metric is available in the semantic layer and can be used in dashboards,
                reports, and analytical queries. {metric.governance_status === 'golden'
                  ? 'It is automatically refreshed according to the configured schedule.'
                  : 'It requires governance approval before automated refresh can be enabled.'}
              </Typography>
            </Alert>
          </Grid>
        </Grid>
      </DialogContent>

      <DialogActions>
        <Button onClick={onClose}>Close</Button>
        <Button
          variant="contained"
          color="primary"
          disabled={metric.governance_status !== 'golden'}
        >
          Add to Dashboard
        </Button>
      </DialogActions>
    </Dialog>
  );
};

export default MetricDetailDialog;
