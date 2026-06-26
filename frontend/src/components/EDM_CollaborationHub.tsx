import React, { useState } from 'react';
import {
  Box,
  Card,
  CardContent,
  CardHeader,
  Button,
  Avatar,
  TextField,
  Chip,
  Grid,
  Paper,
  Typography,
  Divider,
  IconButton,
  Menu,
  MenuItem,
  Stepper,
  Step,
  StepLabel,
  Alert,
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
} from '@mui/material';
import SendIcon from '@mui/icons-material/Send';
import MoreVertIcon from '@mui/icons-material/MoreVert';
import CheckCircleIcon from '@mui/icons-material/CheckCircle';
import PendingIcon from '@mui/icons-material/Pending';
import CancelIcon from '@mui/icons-material/Cancel';
import ThumbUpIcon from '@mui/icons-material/ThumbUp';
import ReplyIcon from '@mui/icons-material/Reply';

interface Participant {
  id: string;
  name: string;
  role: string;
  avatar?: string;
}

interface Comment {
  id: string;
  author: Participant;
  content: string;
  timestamp: string;
  replies?: Comment[];
  status?: 'approved' | 'pending' | 'rejected';
}

interface Workflow {
  id: string;
  name: string;
  status: 'completed' | 'in-progress' | 'pending';
  timestamp: string;
}

const mockParticipants: Participant[] = [
  {
    id: 'p-1',
    name: 'Sarah Chen',
    role: 'Data Steward',
    avatar: 'SC',
  },
  {
    id: 'p-2',
    name: 'James Martinez',
    role: 'Compliance Officer',
    avatar: 'JM',
  },
  {
    id: 'p-3',
    name: 'You',
    role: 'Portfolio Manager',
    avatar: 'ME',
  },
];

const mockWorkflow: Workflow[] = [
  { id: 'w-1', name: 'Draft Created', status: 'completed', timestamp: '2026-02-20 14:30' },
  { id: 'w-2', name: 'Simulation Passed', status: 'completed', timestamp: '2026-02-20 15:12' },
  { id: 'w-3', name: 'Manager Approval', status: 'completed', timestamp: '2026-02-20 16:20' },
  { id: 'w-4', name: 'Compliance Review', status: 'in-progress', timestamp: 'In Progress' },
];

const mockComments: Comment[] = [
  {
    id: 'c-1',
    author: mockParticipants[0],
    content:
      'I\'ve reviewed the simulation results. Everything looks compliant with the new Tier 1 standards. @James_Compliance, can you give the final sign-off?',
    timestamp: '2026-02-20 16:25',
    status: 'approved',
  },
  {
    id: 'c-2',
    author: mockParticipants[2],
    content: 'Checking the exposure limits now. Will update in a minute.',
    timestamp: '2026-02-20 16:32',
    status: 'pending',
  },
  {
    id: 'c-3',
    author: mockParticipants[1],
    content:
      'The override meets all compliance requirements. Confidence scores are within acceptable ranges. Recommended for approval.',
    timestamp: '2026-02-20 16:45',
    status: 'approved',
  },
];

export const CollaborationHub: React.FC = () => {
  const [comments, setComments] = useState(mockComments);
  const [newMessage, setNewMessage] = useState('');
  const [anchorEl, setAnchorEl] = useState<null | HTMLElement>(null);
  const [openApprovalDialog, setOpenApprovalDialog] = useState(false);
  const [selectedComment, setSelectedComment] = useState<Comment | null>(null);

  const handleMenuOpen = (event: React.MouseEvent<HTMLElement>, comment: Comment) => {
    setAnchorEl(event.currentTarget);
    setSelectedComment(comment);
  };

  const handleMenuClose = () => {
    setAnchorEl(null);
  };

  const handleSendMessage = () => {
    if (newMessage.trim()) {
      const newComment: Comment = {
        id: `c-${Date.now()}`,
        author: mockParticipants[2],
        content: newMessage,
        timestamp: new Date().toLocaleString(),
        status: 'pending',
      };
      setComments([...comments, newComment]);
      setNewMessage('');
    }
  };

  const getStatusIcon = (status?: string) => {
    switch (status) {
      case 'approved':
        return <CheckCircleIcon sx={{ color: 'success.main', fontSize: 20 }} />;
      case 'rejected':
        return <CancelIcon sx={{ color: 'error.main', fontSize: 20 }} />;
      default:
        return <PendingIcon sx={{ color: 'warning.main', fontSize: 20 }} />;
    }
  };

  const getStatusColor = (status?: string): 'success' | 'warning' | 'error' | 'default' => {
    switch (status) {
      case 'approved':
        return 'success';
      case 'rejected':
        return 'error';
      default:
        return 'warning';
    }
  };

  return (
    <Box sx={{ p: 3 }}>
      <Grid container spacing={3}>
        {/* Left Panel: Details & Workflow */}
        <Grid item xs={12} lg={7}>
          <Card sx={{ mb: 3 }}>
            <CardHeader
              title="Override Request: OR-789"
              subheader="US Region Holiday Scheduling - High Priority"
              action={
                <Chip label="Pending Approval" color="warning" icon={<PendingIcon />} />
              }
            />
            <CardContent>
              <Grid container spacing={2} sx={{ mb: 3 }}>
                <Grid item xs={12} sm={6}>
                  <Box>
                    <Typography variant="caption" color="textSecondary">
                      Requester
                    </Typography>
                    <Box sx={{ display: 'flex', alignItems: 'center', gap: 1, mt: 1 }}>
                      <Avatar sx={{ width: 32, height: 32, bgcolor: '#137fec' }}>
                        JD
                      </Avatar>
                      <Box>
                        <Typography variant="body2" sx={{ fontWeight: 600 }}>
                          John Doe
                        </Typography>
                        <Typography variant="caption" color="textSecondary">
                          Portfolio Manager
                        </Typography>
                      </Box>
                    </Box>
                  </Box>
                </Grid>
                <Grid item xs={12} sm={6}>
                  <Box>
                    <Typography variant="caption" color="textSecondary">
                      Submitted
                    </Typography>
                    <Typography variant="body2" sx={{ fontWeight: 600, mt: 1 }}>
                      Oct 24, 2:45 PM
                    </Typography>
                    <Typography variant="caption" color="textSecondary">
                      EDM-USR-9921
                    </Typography>
                  </Box>
                </Grid>
              </Grid>

              <Divider sx={{ my: 2 }} />

              {/* Workflow Stepper */}
              <Box sx={{ mb: 3 }}>
                <Typography variant="subtitle2" sx={{ fontWeight: 700, mb: 2 }}>
                  Approval Workflow
                </Typography>
                <Stepper activeStep={3} orientation="vertical">
                  {mockWorkflow.map((step, idx) => (
                    <Step key={step.id} completed={step.status === 'completed'}>
                      <StepLabel
                        StepIconComponent={() => getStatusIcon()}
                        sx={{
                          '& .MuiStepLabel-label': {
                            fontSize: '0.95rem',
                            fontWeight: step.status === 'in-progress' ? 600 : 500,
                            color:
                              step.status === 'in-progress'
                                ? '#137fec'
                                : 'rgb(107 114 128)',
                          },
                        }}
                      >
                        <Box>
                          <Typography variant="body2" sx={{ fontWeight: 600 }}>
                            {step.name}
                          </Typography>
                          <Typography variant="caption" color="textSecondary">
                            {step.timestamp}
                          </Typography>
                        </Box>
                      </StepLabel>
                    </Step>
                  ))}
                </Stepper>
              </Box>

              {/* Comparison Table */}
              <Box>
                <Typography variant="subtitle2" sx={{ fontWeight: 700, mb: 2 }}>
                  Override Comparison
                </Typography>
                <Paper variant="outlined" sx={{ overflow: 'hidden' }}>
                  <Box
                    sx={{
                      display: 'grid',
                      gridTemplateColumns: '1fr 1fr',
                      bgcolor: '#f3f4f6',
                      borderBottom: '1px solid #e5e7eb',
                    }}
                  >
                    <Box sx={{ p: 2, fontWeight: 700, fontSize: '0.875rem', color: '#6b7280' }}>
                      Original Value
                    </Box>
                    <Box sx={{ p: 2, fontWeight: 700, fontSize: '0.875rem', color: '#137fec' }}>
                      Proposed Override
                    </Box>
                  </Box>
                  <Box
                    sx={{
                      display: 'grid',
                      gridTemplateColumns: '1fr 1fr',
                      borderBottom: '1px solid #e5e7eb',
                      '&:hover': { bgcolor: '#f9fafb' },
                    }}
                  >
                    <Box sx={{ p: 2 }}>
                      <Typography variant="caption" color="textSecondary">
                        Market Cap Rating
                      </Typography>
                      <Typography
                        variant="body2"
                        sx={{
                          mt: 0.5,
                          textDecoration: 'line-through',
                          color: '#dc2626',
                        }}
                      >
                        Standard High-Cap
                      </Typography>
                    </Box>
                    <Box sx={{ p: 2 }}>
                      <Typography variant="caption" color="textSecondary">
                        &nbsp;
                      </Typography>
                      <Typography
                        variant="body2"
                        sx={{
                          mt: 0.5,
                          fontWeight: 600,
                          color: '#137fec',
                        }}
                      >
                        Premium Enterprise
                      </Typography>
                    </Box>
                  </Box>
                  <Box
                    sx={{
                      display: 'grid',
                      gridTemplateColumns: '1fr 1fr',
                    }}
                  >
                    <Box sx={{ p: 2 }}>
                      <Typography variant="caption" color="textSecondary">
                        Risk Weight
                      </Typography>
                      <Typography
                        variant="body2"
                        sx={{
                          mt: 0.5,
                          textDecoration: 'line-through',
                          color: '#dc2626',
                        }}
                      >
                        0.024
                      </Typography>
                    </Box>
                    <Box sx={{ p: 2 }}>
                      <Typography variant="caption" color="textSecondary">
                        &nbsp;
                      </Typography>
                      <Typography
                        variant="body2"
                        sx={{
                          mt: 0.5,
                          fontWeight: 600,
                          color: '#137fec',
                        }}
                      >
                        0.018
                      </Typography>
                    </Box>
                  </Box>
                </Paper>
              </Box>
            </CardContent>
          </Card>

          {/* Action Buttons */}
          <Box sx={{ display: 'flex', gap: 2 }}>
            <Button
              variant="contained"
              color="success"
              fullWidth
              startIcon={<CheckCircleIcon />}
              onClick={() => setOpenApprovalDialog(true)}
            >
              Approve
            </Button>
            <Button variant="outlined" color="warning" fullWidth>
              Request Changes
            </Button>
            <Button variant="outlined" color="error" fullWidth startIcon={<CancelIcon />}>
              Reject
            </Button>
          </Box>
        </Grid>

        {/* Right Panel: Collaboration Thread */}
        <Grid item xs={12} lg={5}>
          <Card sx={{ height: '100%', display: 'flex', flexDirection: 'column' }}>
            <CardHeader
              title="Collaboration Thread"
              subheader={`${mockParticipants.length} participants`}
            />
            <CardContent
              sx={{
                flex: 1,
                overflowY: 'auto',
                display: 'flex',
                flexDirection: 'column',
                gap: 2,
              }}
            >
              {/* Participant List */}
              <Box>
                <Typography variant="caption" color="textSecondary" sx={{ fontWeight: 600, mb: 1, display: 'block' }}>
                  PARTICIPANTS
                </Typography>
                <Box sx={{ display: 'flex', gap: 1, flexWrap: 'wrap' }}>
                  {mockParticipants.map((p) => (
                    <Chip
                      key={p.id}
                      avatar={<Avatar sx={{ bgcolor: '#137fec' }}>{p.avatar}</Avatar>}
                      label={`${p.name} (${p.role})`}
                      variant="outlined"
                    />
                  ))}
                </Box>
              </Box>

              <Divider />

              {/* Comments */}
              <Box sx={{ flex: 1 }}>
                {comments.map((comment, idx) => (
                  <Box key={comment.id} sx={{ mb: 2 }}>
                    <Box sx={{ display: 'flex', gap: 2, mb: 1 }}>
                      <Avatar sx={{ bgcolor: '#137fec', width: 32, height: 32 }}>
                        {comment.author.avatar}
                      </Avatar>
                      <Box sx={{ flex: 1 }}>
                        <Box sx={{ display: 'flex', alignItems: 'center', gap: 1, mb: 0.5 }}>
                          <Typography variant="subtitle2" sx={{ fontWeight: 600 }}>
                            {comment.author.name}
                          </Typography>
                          <Typography variant="caption" color="textSecondary">
                            {comment.timestamp}
                          </Typography>
                          {comment.status && getStatusIcon(comment.status)}
                        </Box>
                        <Paper
                          sx={{
                            p: 1.5,
                            bgcolor: comment.author.id === 'p-3' ? '#137fec' : '#f3f4f6',
                            color: comment.author.id === 'p-3' ? 'white' : 'inherit',
                          }}
                        >
                          <Typography variant="body2">{comment.content}</Typography>
                        </Paper>
                        <Box sx={{ display: 'flex', gap: 1, mt: 0.5 }}>
                          <Button size="small" startIcon={<ReplyIcon />}>
                            Reply
                          </Button>
                          <Button size="small" startIcon={<ThumbUpIcon />}>
                            Like
                          </Button>
                        </Box>
                      </Box>
                      <IconButton
                        size="small"
                        onClick={(e) => handleMenuOpen(e, comment)}
                      >
                        <MoreVertIcon fontSize="small" />
                      </IconButton>
                    </Box>
                  </Box>
                ))}
              </Box>
            </CardContent>

            <Divider />

            {/* Message Input */}
            <Box sx={{ p: 2 }}>
              <Box sx={{ display: 'flex', gap: 1 }}>
                <TextField
                  multiline
                  rows={2}
                  placeholder="Type your comment..."
                  value={newMessage}
                  onChange={(e) => setNewMessage(e.target.value)}
                  fullWidth
                  size="small"
                />
                <IconButton
                  color="primary"
                  onClick={handleSendMessage}
                  sx={{ alignSelf: 'flex-end' }}
                >
                  <SendIcon />
                </IconButton>
              </Box>
              <Typography variant="caption" color="textSecondary" sx={{ mt: 1, display: 'block' }}>
                💡 Use @name to mention team members
              </Typography>
            </Box>
          </Card>
        </Grid>
      </Grid>

      {/* Menu */}
      <Menu
        anchorEl={anchorEl}
        open={Boolean(anchorEl)}
        onClose={handleMenuClose}
      >
        <MenuItem onClick={handleMenuClose}>Edit</MenuItem>
        <MenuItem onClick={handleMenuClose}>Pin</MenuItem>
        <MenuItem onClick={handleMenuClose}>Delete</MenuItem>
      </Menu>

      {/* Approval Dialog */}
      <Dialog open={openApprovalDialog} onClose={() => setOpenApprovalDialog(false)} maxWidth="sm" fullWidth>
        <DialogTitle>Approve Override Request</DialogTitle>
        <DialogContent sx={{ pt: 2 }}>
          <Alert severity="info" sx={{ mb: 2 }}>
            You are approving override OR-789. This action will advance the request to the next approval stage.
          </Alert>
          <TextField
            label="Approval Comments (Optional)"
            multiline
            rows={4}
            fullWidth
            placeholder="Add any additional notes or conditions..."
          />
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setOpenApprovalDialog(false)}>Cancel</Button>
          <Button
            variant="contained"
            color="success"
            onClick={() => setOpenApprovalDialog(false)}
          >
            Confirm Approval
          </Button>
        </DialogActions>
      </Dialog>
    </Box>
  );
};
