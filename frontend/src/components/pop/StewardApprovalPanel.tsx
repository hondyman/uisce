import { useState, useEffect, useCallback } from 'react';
import type { ReactNode } from 'react';
import {
  Card,
  CardHeader,
  CardContent,
  List,
  ListItem,
  ListItemText,
  ListItemAvatar,
  Button,
  IconButton,
  Modal,
  FormControl,
  InputLabel,
  Select,
  MenuItem,
  Chip,
  Stack,
  Avatar,
  Tooltip,
  Checkbox,
  FormControlLabel,
  TextField,
  Typography,
  Box,
  CircularProgress,
} from '@mui/material';
import {
  CheckCircle as CheckCircleOutlined,
  Cancel as CloseCircleOutlined,
  Flag as FlagOutlined,
  Comment as CommentOutlined,
  Person as UserOutlined,
} from '@mui/icons-material';
import { useSnackbar } from 'notistack';
import { formatDistanceToNow } from 'date-fns';
import { useForm } from 'react-hook-form';



interface PendingApproval {
  node_id: string;
  node_type: string;
  name: string;
  description: string;
  schema_def: string;
  created_at: string;
  review_status: string;
}

interface StewardApprovalPanelProps {
  stewardUser?: string;
  onApprovalAction?: (action: string, nodeId: string, data: any) => void;
}

const StewardApprovalPanel: React.FC<StewardApprovalPanelProps> = ({
  stewardUser = 'system',
  onApprovalAction
}) => {
  const { enqueueSnackbar } = useSnackbar();
  const [pendingApprovals, setPendingApprovals] = useState<PendingApproval[]>([]);
  const [loading, setLoading] = useState(false);
  const [actionModalVisible, setActionModalVisible] = useState(false);
  const [selectedApproval, setSelectedApproval] = useState<PendingApproval | null>(null);
  const [actionType, setActionType] = useState<'approve' | 'reject' | 'flag' | 'comment'>('comment');
  const { register, handleSubmit, formState: { errors }, reset } = useForm();

  const fetchPendingApprovals = useCallback(async () => {
    setLoading(true);
    try {
      const response = await fetch(`/api/steward/approvals?steward=${stewardUser}`);
      const data = await response.json();
      setPendingApprovals(data.pending_approvals);
    } catch (error) {
      enqueueSnackbar('Failed to fetch pending approvals', { variant: 'error' });
    } finally {
      setLoading(false);
    }
  }, [stewardUser]);

  useEffect(() => {
    fetchPendingApprovals();
  }, [fetchPendingApprovals]);

  const handleAction = (approval: PendingApproval, action: 'approve' | 'reject' | 'flag' | 'comment') => {
    setSelectedApproval(approval);
    setActionType(action);
    setActionModalVisible(true);
  };

  const handleActionSubmit = async (values: any) => {
    if (!selectedApproval) return;

    try {
      let endpoint = '';
      const method = 'POST';
      const body: any = {
        steward_user: stewardUser,
        comment: values.comment
      };

      switch (actionType) {
        case 'approve':
          endpoint = `/api/steward/approvals/${selectedApproval.node_id}/approve`;
          body.golden_path = values.golden_path;
          break;
        case 'reject':
          endpoint = `/api/steward/approvals/${selectedApproval.node_id}/reject`;
          break;
        case 'flag':
          endpoint = `/api/steward/approvals/${selectedApproval.node_id}/flag`;
          body.severity = values.severity;
          break;
        case 'comment':
          endpoint = `/api/steward/approvals/${selectedApproval.node_id}/comment`;
          body.action = 'comment';
          break;
      }

      const response = await fetch(endpoint, {
        method,
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify(body),
      });

      if (!response.ok) {
        throw new Error('Action failed');
      }

      enqueueSnackbar(`${actionType.charAt(0).toUpperCase() + actionType.slice(1)} action completed`, { variant: 'success' });
      setActionModalVisible(false);
      reset();
      fetchPendingApprovals();

      if (onApprovalAction) {
        onApprovalAction(actionType, selectedApproval.node_id, body);
      }
    } catch (error) {
      enqueueSnackbar(`Failed to ${actionType} approval`, { variant: 'error' });
    }
  };

  const getStatusColor = (status: string): "warning" | "primary" | "default" => {
    switch (status) {
      case 'draft':
        return 'warning';
      case 'pending_review':
        return 'primary';
      default:
        return 'default';
    }
  };

  const getActionButtons = (approval: PendingApproval) => (
    <Stack direction="row" spacing={1}>
      <Tooltip title="Approve">
        <IconButton
          color="primary"
          size="small"
          onClick={() => handleAction(approval, 'approve')}
        >
          <CheckCircleOutlined />
        </IconButton>
      </Tooltip>
      <Tooltip title="Reject">
        <IconButton
          color="error"
          size="small"
          onClick={() => handleAction(approval, 'reject')}
        >
          <CloseCircleOutlined />
        </IconButton>
      </Tooltip>
      <Tooltip title="Flag for Review">
        <IconButton
          size="small"
          onClick={() => handleAction(approval, 'flag')}
        >
          <FlagOutlined />
        </IconButton>
      </Tooltip>
      <Tooltip title="Add Comment">
        <IconButton
          size="small"
          onClick={() => handleAction(approval, 'comment')}
        >
          <CommentOutlined />
        </IconButton>
      </Tooltip>
    </Stack>
  );

  return (
    <>
      <Card>
        <CardHeader
          title={`Steward Approvals (${pendingApprovals.length})`}
          action={
            <Button size="small" onClick={fetchPendingApprovals}>
              Refresh
            </Button>
          }
        />
        <CardContent>
          {loading ? (
            <Box sx={{ display: 'flex', justifyContent: 'center', my: 2 }}>
              <CircularProgress />
            </Box>
          ) : (
            <List>
              {pendingApprovals.map((approval) => (
                <ListItem
                  key={approval.node_id}
                  secondaryAction={getActionButtons(approval)}
                >
                  <ListItemAvatar>
                    <Avatar>
                      <UserOutlined />
                    </Avatar>
                  </ListItemAvatar>
                  <ListItemText
                    primary={
                      <Stack direction="row" spacing={1} alignItems="center">
                        <Typography>{approval.name}</Typography>
                        <Chip
                          label={approval.review_status.replace('_', ' ')}
                          color={getStatusColor(approval.review_status)}
                          size="small"
                        />
                        <Chip label={approval.node_type} size="small" />
                      </Stack>
                    }
                    secondary={
                      <>
                        <Typography variant="body2">{approval.description}</Typography>
                        <Typography variant="caption">
                          Created {formatDistanceToNow(new Date(approval.created_at), { addSuffix: true })}
                        </Typography>
                      </>
                    }
                  />
                </ListItem>
              ))}
            </List>
          )}
        </CardContent>
      </Card>

      <Modal
        open={actionModalVisible}
        onClose={() => setActionModalVisible(false)}
      >
        <Box sx={{
          position: 'absolute',
          top: '50%',
          left: '50%',
          transform: 'translate(-50%, -50%)',
          width: 600,
          bgcolor: 'background.paper',
          boxShadow: 24,
          p: 4,
        }}>
          <Typography variant="h6" component="h2">
            {`${actionType.charAt(0).toUpperCase() + actionType.slice(1)} Approval`}
          </Typography>
          {selectedApproval && (
            <Box sx={{ my: 2 }}>
              <Typography variant="h6">{selectedApproval.name}</Typography>
              <Typography variant="body2">{selectedApproval.description}</Typography>
              <Chip label={selectedApproval.node_type} size="small" />
            </Box>
          )}
          <form onSubmit={handleSubmit(handleActionSubmit)}>
            <Stack spacing={2}>
              {actionType === 'approve' && (
                <FormControlLabel
                  control={<Checkbox {...register('golden_path')} />}
                  label="Add this metric to the golden path for future reuse"
                />
              )}

              {actionType === 'flag' && (
                <FormControl fullWidth>
                  <InputLabel>Severity</InputLabel>
                  <Select
                    label="Severity"
                    {...register('severity', { required: 'Please select severity' })}
                  >
                    <MenuItem value="low">Low</MenuItem>
                    <MenuItem value="medium">Medium</MenuItem>
                    <MenuItem value="high">High</MenuItem>
                    <MenuItem value="critical">Critical</MenuItem>
                  </Select>
                </FormControl>
              )}

              <TextField
                label={actionType === 'comment' ? 'Comment' : 'Feedback'}
                multiline
                rows={4}
                placeholder={
                  actionType === 'approve' ? 'Optional approval notes...' :
                  actionType === 'reject' ? 'Please provide reason for rejection...' :
                  actionType === 'flag' ? 'Please describe the issue...' :
                  'Add your comment...'
                }
                {...register('comment', { required: actionType !== 'approve' ? 'Please provide feedback' : false })}
                error={!!errors.comment}
                helperText={errors.comment?.message as ReactNode}
              />

              <Stack direction="row" spacing={2}>
                <Button type="submit" variant="contained">
                  {actionType.charAt(0).toUpperCase() + actionType.slice(1)}
                </Button>
                <Button onClick={() => setActionModalVisible(false)}>
                  Cancel
                </Button>
              </Stack>
            </Stack>
          </form>
        </Box>
      </Modal>
    </>
  );
};

export default StewardApprovalPanel;
