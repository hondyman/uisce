import React, { useEffect, useState } from 'react';
import {
  Alert,
  Box,
  Typography,
  Button,
  Stack,
  Chip,
  Drawer,
  IconButton,
  Divider,
  List,
  ListItem,
  ListItemIcon,
  ListItemText,
} from '@mui/material';
import {
  Warning as WarningIcon,
  Error as ErrorIcon,
  CheckCircle as CheckCircleIcon,
  Info as InfoIcon,
  Close as CloseIcon,
  DataObject as DataObjectIcon,
  Calculate as CalculateIcon,
} from '@mui/icons-material';
import { format } from 'date-fns';

// ============================================================================
// Types
// ============================================================================

interface BOStatus {
  status: string;
  reason: string;
  pending_terms: string[];
  pending_calculations: string[];
  pending_dependencies: DependencyIssue[];
  validation_errors: ValidationError[];
  diff_required: boolean;
  import_pending: boolean;
  last_modified: string;
  modified_by: string;
  version: string;
  is_published: boolean;
  can_publish: boolean;
}

interface DependencyIssue {
  type: string;
  id: string;
  name: string;
  status: string;
  blocks_publish: boolean;
}

interface ValidationError {
  field: string;
  message: string;
  severity: string;
}

interface BOPendingBannerProps {
  boId: string;
  onTabChange?: (tabIndex: number) => void;
  onPublish?: () => void;
  onRefresh?: () => void;
}

// ============================================================================
// Component
// ============================================================================

export const BOPendingBanner: React.FC<BOPendingBannerProps> = ({ 
  boId, 
  onTabChange,
  onPublish,
  onRefresh,
}) => {
  const [status, setStatus] = useState<BOStatus | null>(null);
  const [detailsOpen, setDetailsOpen] = useState(false);
  const [loading, setLoading] = useState(true);
  const [publishing, setPublishing] = useState(false);

  const fetchStatus = () => {
    if (!boId) return;

    fetch(`/api/bo/${boId}/status`)
      .then(res => res.json())
      .then(data => {
        setStatus(data);
        setLoading(false);
      })
      .catch(err => {
        console.error('Failed to fetch BO status:', err);
        setLoading(false);
      });
  };

  useEffect(() => {
    fetchStatus();
  }, [boId]);

  if (loading || !status || status.status === 'published') {
    return null; // No banner for published BOs
  }

  const getColor = (): 'error' | 'warning' | 'info' | 'success' => {
    switch (status.status) {
      case 'error': return 'error';
      case 'pending_import':
      case 'pending_diff_resolution':
      case 'pending_dependencies': return 'warning';
      case 'pending_publish': return 'info';
      case 'draft':
      case 'pending_review':
      default: return 'warning';
    }
  };

  const getIcon = () => {
    switch (status.status) {
      case 'error': return <ErrorIcon />;
      case 'pending_publish': return <CheckCircleIcon />;
      default: return <WarningIcon />;
    }
  };

  const getStatusLabel = (): string => {
    const labels: Record<string, string> = {
      'draft': 'Draft',
      'pending_review': 'Pending Review',
      'pending_dependencies': 'Pending Dependencies',
      'pending_import': 'Pending Import',
      'pending_diff_resolution': 'Pending Diff Resolution',
      'pending_publish': 'Ready to Publish',
      'error': 'Validation Errors',
    };
    return labels[status.status] || 'Unknown Status';
  };

  const getActions = () => {
    const actions = [];

    if (status.pending_terms && status.pending_terms.length > 0) {
      actions.push(
        <Button
          key="review-terms"
          variant="outlined"
          size="small"
          onClick={() => onTabChange?.(1)} // Tab 1 = Terms
        >
          Review Terms
        </Button>
      );
    }

    if (status.pending_calculations && status.pending_calculations.length > 0) {
      actions.push(
        <Button
          key="review-calcs"
          variant="outlined"
          size="small"
          onClick={() => {
            // Navigate to calculations tab if it exists
            // For now, open details drawer
            setDetailsOpen(true);
          }}
        >
          Review Calculations
        </Button>
      );
    }

    if (status.diff_required) {
      actions.push(
        <Button
          key="view-diff"
          variant="outlined"
          size="small"
          onClick={() => {
            // TODO: Open diff viewer modal
            setDetailsOpen(true);
          }}
        >
          View Diff
        </Button>
      );
    }

    if (status.can_publish) {
      actions.push(
        <Button
          key="publish"
          variant="contained"
          size="small"
          color="primary"
          disabled={publishing}
          onClick={async () => {
            setPublishing(true);
            try {
              const response = await fetch(`/api/bo/${boId}/publish`, {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
              });
              
              if (response.ok) {
                // Refresh status
                fetchStatus();
                // Notify parent
                onPublish?.();
                onRefresh?.();
              } else {
                console.error('Failed to publish BO');
              }
            } catch (err) {
              console.error('Error publishing BO:', err);
            } finally {
              setPublishing(false);
            }
          }}
        >
          {publishing ? 'Publishing...' : 'Publish Now'}
        </Button>
      );
    }

    actions.push(
      <Button
        key="details"
        variant="text"
        size="small"
        onClick={() => setDetailsOpen(true)}
      >
        View Details
      </Button>
    );

    return actions;
  };

  const formatDate = (dateStr: string) => {
    try {
      return format(new Date(dateStr), 'MMM d, yyyy');
    } catch {
      return dateStr;
    }
  };

  return (
    <>
      <Alert
        severity={getColor()}
        icon={getIcon()}
        sx={{
          mb: 2,
          borderRadius: 2,
          '& .MuiAlert-message': {
            width: '100%',
          },
        }}
      >
        <Box>
          <Typography variant="subtitle1" fontWeight="bold">
            {getStatusLabel()}: {status.reason}
          </Typography>

          <Typography variant="caption" color="text.secondary" sx={{ display: 'block', mt: 0.5 }}>
            Last modified by {status.modified_by || 'Unknown'} • {formatDate(status.last_modified)} • Version {status.version}
          </Typography>

          <Stack direction="row" spacing={1} sx={{ mt: 1.5 }} flexWrap="wrap">
            {getActions()}
          </Stack>
        </Box>
      </Alert>

      <BOStatusDetailsDrawer
        open={detailsOpen}
        onClose={() => setDetailsOpen(false)}
        status={status}
      />
    </>
  );
};

// ============================================================================
// Details Drawer
// ============================================================================

const BOStatusDetailsDrawer: React.FC<{
  open: boolean;
  onClose: () => void;
  status: BOStatus;
}> = ({ open, onClose, status }) => {
  return (
    <Drawer anchor="right" open={open} onClose={onClose}>
      <Box sx={{ width: 450, p: 3 }}>
        <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', mb: 2 }}>
          <Typography variant="h6">BO Status Details</Typography>
          <IconButton onClick={onClose} size="small">
            <CloseIcon />
          </IconButton>
        </Box>

        <Divider sx={{ mb: 2 }} />

        {/* Pending Terms */}
        {status.pending_terms && status.pending_terms.length > 0 && (
          <Box sx={{ mb: 3 }}>
            <Typography variant="subtitle2" gutterBottom>
              Pending Terms ({status.pending_terms.length})
            </Typography>
            <List dense>
              {status.pending_terms.map(term => (
                <ListItem key={term}>
                  <ListItemIcon>
                    <DataObjectIcon fontSize="small" />
                  </ListItemIcon>
                  <ListItemText primary={term} />
                </ListItem>
              ))}
            </List>
          </Box>
        )}

        {/* Pending Calculations */}
        {status.pending_calculations && status.pending_calculations.length > 0 && (
          <Box sx={{ mb: 3 }}>
            <Typography variant="subtitle2" gutterBottom>
              Pending Calculations ({status.pending_calculations.length})
            </Typography>
            <List dense>
              {status.pending_calculations.map(calc => (
                <ListItem key={calc}>
                  <ListItemIcon>
                    <CalculateIcon fontSize="small" />
                  </ListItemIcon>
                  <ListItemText primary={calc} />
                </ListItem>
              ))}
            </List>
          </Box>
        )}

        {/* Validation Errors */}
        {status.validation_errors && status.validation_errors.length > 0 && (
          <Box sx={{ mb: 3 }}>
            <Typography variant="subtitle2" gutterBottom>
              Validation Errors ({status.validation_errors.length})
            </Typography>
            <List dense>
              {status.validation_errors.map((error, idx) => (
                <ListItem key={idx}>
                  <ListItemIcon>
                    <ErrorIcon fontSize="small" color="error" />
                  </ListItemIcon>
                  <ListItemText
                    primary={error.field}
                    secondary={error.message}
                  />
                </ListItem>
              ))}
            </List>
          </Box>
        )}

        {/* Pending Dependencies */}
        {status.pending_dependencies && status.pending_dependencies.length > 0 && (
          <Box sx={{ mb: 3 }}>
            <Typography variant="subtitle2" gutterBottom>
              Pending Dependencies ({status.pending_dependencies.length})
            </Typography>
            <List dense>
              {status.pending_dependencies.map(dep => (
                <ListItem key={dep.id}>
                  <ListItemText
                    primary={dep.name}
                    secondary={`${dep.type} • ${dep.status}`}
                  />
                  {dep.blocks_publish && (
                    <Chip label="Blocks Publish" size="small" color="error" />
                  )}
                </ListItem>
              ))}
            </List>
          </Box>
        )}

        {/* Status Summary */}
        <Box sx={{ mt: 3, p: 2, bgcolor: 'grey.100', borderRadius: 1 }}>
          <Typography variant="caption" color="text.secondary">
            Status Summary
          </Typography>
          <Typography variant="body2" sx={{ mt: 0.5 }}>
            {status.reason}
          </Typography>
          <Stack direction="row" spacing={1} sx={{ mt: 1 }}>
            {status.can_publish ? (
              <Chip label="Can Publish" size="small" color="success" />
            ) : (
              <Chip label="Cannot Publish" size="small" color="error" />
            )}
            {status.is_published && (
              <Chip label="Published" size="small" color="info" />
            )}
          </Stack>
        </Box>
      </Box>
    </Drawer>
  );
};

export default BOPendingBanner;
