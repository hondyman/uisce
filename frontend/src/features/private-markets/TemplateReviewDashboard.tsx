import React, { useState, useEffect, useCallback } from 'react';
import {
  Box,
  Typography,
  Paper,
  Grid,
  Card as _Card,
  CardContent as _CardContent,
  Button,
  Chip,
  Table as _Table,
  TableBody as _TableBody,
  TableCell as _TableCell,
  TableContainer as _TableContainer,
  TableHead as _TableHead,
  TableRow as _TableRow,
  Dialog,
  DialogContent,
  DialogActions,
  List,
  Tabs,
  Tab,
  Accordion,
  AccordionSummary,
  AccordionDetails,
  TextField,
  FormControl,
  InputLabel,
  Select,
  MenuItem,
  SelectChangeEvent,
  Alert,
  CircularProgress,
  // ...existing code...
  ListItem,
  ListItemText,
} from '@mui/material';
  // Divider not used
import ModalHeader from '../../components/ModalHeader';
import {
  ExpandMore,
  CheckCircle,
  Pending,
  Error as ErrorIcon,
  Visibility,
  ThumbUp,
  Comment,
  History as _History
} from '@mui/icons-material';

interface Template {
  id: string;
  name: string;
  domain: string;
  category: string;
  subcategory: string;
  version: string;
  status: 'draft' | 'reviewed' | 'golden';
  owner: string;
  description: string;
  tags: string[];
  governance: {
    status: string;
    steward_group: string;
    schema_hash: string;
    sla: {
      refresh_frequency: string;
      max_latency: string;
    };
  };
  created_at: string;
  updated_at: string;
  review_comments: ReviewComment[];
}

interface ReviewComment {
  id: string;
  author: string;
  comment: string;
  type: 'comment' | 'approval' | 'rejection' | 'change_request';
  created_at: string;
}

interface TemplateReviewDashboardProps {
  domain?: string;
}

export const TemplateReviewDashboard: React.FC<TemplateReviewDashboardProps> = ({
  domain = 'private_markets'
}) => {
  const [templates, setTemplates] = useState<Template[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [selectedTemplate, setSelectedTemplate] = useState<Template | null>(null);
  const [reviewDialogOpen, setReviewDialogOpen] = useState(false);
  const [commentDialogOpen, setCommentDialogOpen] = useState(false);
  const [activeTab, setActiveTab] = useState(0);
  const [filterStatus, setFilterStatus] = useState<string>('all');
  const [filterCategory, setFilterCategory] = useState<string>('all');
  const [newComment, setNewComment] = useState('');
  const [commentType, setCommentType] = useState<'comment' | 'approval' | 'rejection' | 'change_request'>('comment');

  const loadTemplates = useCallback(async () => {
    try {
      setLoading(true);
      const response = await fetch(`/api/templates?domain=${domain}`);
      if (!response.ok) {
        throw new Error('Failed to load templates');
      }
      const data = await response.json();
      setTemplates(data);
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : 'Failed to load templates');
    } finally {
      setLoading(false);
    }
  }, [domain]);

  useEffect(() => {
    loadTemplates();
  }, [loadTemplates]);

  

  const handleStatusFilterChange = (event: SelectChangeEvent) => {
    setFilterStatus(event.target.value);
  };

  const handleCategoryFilterChange = (event: SelectChangeEvent) => {
    setFilterCategory(event.target.value);
  };

  const handleTabChange = (_event: React.SyntheticEvent, newValue: number) => {
    setActiveTab(newValue);
  };

  const getStatusColor = (status: string) => {
    switch (status) {
      case 'draft': return 'warning';
      case 'reviewed': return 'info';
      case 'golden': return 'success';
      default: return 'default';
    }
  };

  const getStatusIcon = (status: string) => {
    switch (status) {
  case 'draft': return <Pending />;
  case 'reviewed': return <CheckCircle />;
  case 'golden': return <CheckCircle />;
  default: return <ErrorIcon />;
    }
  };

  const filteredTemplates = templates.filter(template => {
    const statusMatch = filterStatus === 'all' || template.status === filterStatus;
    const categoryMatch = filterCategory === 'all' || template.category === filterCategory;
    return statusMatch && categoryMatch;
  });

  const handlePromoteTemplate = async (templateId: string, newStatus: 'reviewed' | 'golden') => {
    try {
      const response = await fetch(`/api/templates/${templateId}/promote`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({ status: newStatus }),
      });

      if (!response.ok) {
        throw new Error('Failed to promote template');
      }

      // Refresh templates
      await loadTemplates();
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : 'Failed to promote template');
    }
  };

  const handleAddComment = async () => {
    if (!selectedTemplate || !newComment.trim()) return;

    try {
      const response = await fetch(`/api/templates/${selectedTemplate.id}/comments`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({
          comment: newComment,
          type: commentType,
        }),
      });

      if (!response.ok) {
        throw new Error('Failed to add comment');
      }

      setNewComment('');
      setCommentDialogOpen(false);
      await loadTemplates();
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : 'Failed to add comment');
    }
  };

  const categories = [...new Set(templates.map(t => t.category))];

  if (loading) {
    return (
      <Box display="flex" justifyContent="center" alignItems="center" minHeight="400px">
        <CircularProgress />
      </Box>
    );
  }

  if (error) {
    return (
      <Alert severity="error" sx={{ m: 2 }}>
        {error}
      </Alert>
    );
  }

  return (
    <Box sx={{ flexGrow: 1, p: 3 }}>
      <Typography variant="h4" component="h1" gutterBottom>
        Template Review Dashboard - {domain.replace('_', ' ').toUpperCase()}
      </Typography>

      {/* Filters */}
      <Paper sx={{ p: 2, mb: 3 }}>
        <Grid container spacing={2} alignItems="center">
          <Grid item xs={12} md={3}>
            <FormControl fullWidth>
              <InputLabel>Status</InputLabel>
              <Select value={filterStatus} label="Status" onChange={handleStatusFilterChange}>
                <MenuItem value="all">All Status</MenuItem>
                <MenuItem value="draft">Draft</MenuItem>
                <MenuItem value="reviewed">Reviewed</MenuItem>
                <MenuItem value="golden">Golden</MenuItem>
              </Select>
            </FormControl>
          </Grid>
          <Grid item xs={12} md={3}>
            <FormControl fullWidth>
              <InputLabel>Category</InputLabel>
              <Select value={filterCategory} label="Category" onChange={handleCategoryFilterChange}>
                <MenuItem value="all">All Categories</MenuItem>
                {categories.map(category => (
                  <MenuItem key={category} value={category}>
                    {category}
                  </MenuItem>
                ))}
              </Select>
            </FormControl>
          </Grid>
          <Grid item xs={12} md={6}>
            <Typography variant="body2" color="text.secondary">
              Showing {filteredTemplates.length} of {templates.length} templates
            </Typography>
          </Grid>
        </Grid>
      </Paper>

      {/* Main Content */}
      <Grid container spacing={3}>
        {/* Template List */}
        <Grid item xs={12} lg={4}>
          <Paper sx={{ p: 2, maxHeight: 600, overflow: 'auto' }}>
            <Typography variant="h6" gutterBottom>
              Templates ({filteredTemplates.length})
            </Typography>
            <List>
              {filteredTemplates.map((template) => (
                <ListItem
                  key={template.id}
                  button
                  onClick={() => setSelectedTemplate(template)}
                  selected={selectedTemplate?.id === template.id}
                >
                  <ListItemText
                    primary={
                      <Box display="flex" alignItems="center" gap={1}>
                        {getStatusIcon(template.status)}
                        <Typography variant="subtitle2">
                          {template.name}
                        </Typography>
                      </Box>
                    }
                    secondary={
                      <Box>
                        <Typography variant="caption" display="block">
                          {template.category} • v{template.version}
                        </Typography>
                        <Chip
                          label={template.status}
                          size="small"
                          color={getStatusColor(template.status) as any}
                          sx={{ mt: 0.5 }}
                        />
                      </Box>
                    }
                  />
                </ListItem>
              ))}
            </List>
          </Paper>
        </Grid>

        {/* Template Details */}
        <Grid item xs={12} lg={8}>
          {selectedTemplate ? (
            <Paper sx={{ p: 3 }}>
              <Box display="flex" justifyContent="space-between" alignItems="flex-start" mb={2}>
                <Box>
                  <Typography variant="h5" gutterBottom>
                    {selectedTemplate.name}
                  </Typography>
                  <Typography variant="body2" color="text.secondary" gutterBottom>
                    {selectedTemplate.description}
                  </Typography>
                </Box>
                <Box display="flex" gap={1}>
                  <Button
                    variant="outlined"
                    startIcon={<Visibility />}
                    onClick={() => setReviewDialogOpen(true)}
                  >
                    Review
                  </Button>
                  <Button
                    variant="outlined"
                    startIcon={<Comment />}
                    onClick={() => setCommentDialogOpen(true)}
                  >
                    Comment
                  </Button>
                  {selectedTemplate.status === 'draft' && (
                    <Button
                      variant="contained"
                      startIcon={<ThumbUp />}
                      onClick={() => handlePromoteTemplate(selectedTemplate.id, 'reviewed')}
                    >
                      Approve
                    </Button>
                  )}
                  {selectedTemplate.status === 'reviewed' && (
                    <Button
                      variant="contained"
                      startIcon={<CheckCircle />}
                      onClick={() => handlePromoteTemplate(selectedTemplate.id, 'golden')}
                    >
                      Make Golden
                    </Button>
                  )}
                </Box>
              </Box>

              <Tabs value={activeTab} onChange={handleTabChange} sx={{ mb: 2 }}>
                <Tab label="Details" />
                <Tab label="Governance" />
                <Tab label="Comments" />
                <Tab label="History" />
              </Tabs>

              {activeTab === 0 && (
                <Box>
                  <Grid container spacing={2}>
                    <Grid item xs={12} md={6}>
                      <Typography variant="subtitle2">Domain</Typography>
                      <Typography>{selectedTemplate.domain}</Typography>
                    </Grid>
                    <Grid item xs={12} md={6}>
                      <Typography variant="subtitle2">Category</Typography>
                      <Typography>{selectedTemplate.category}</Typography>
                    </Grid>
                    <Grid item xs={12} md={6}>
                      <Typography variant="subtitle2">Version</Typography>
                      <Typography>{selectedTemplate.version}</Typography>
                    </Grid>
                    <Grid item xs={12} md={6}>
                      <Typography variant="subtitle2">Owner</Typography>
                      <Typography>{selectedTemplate.owner}</Typography>
                    </Grid>
                    <Grid item xs={12}>
                      <Typography variant="subtitle2">Tags</Typography>
                      <Box display="flex" gap={1} flexWrap="wrap">
                        {selectedTemplate.tags.map(tag => (
                          <Chip key={tag} label={tag} size="small" variant="outlined" />
                        ))}
                      </Box>
                    </Grid>
                  </Grid>
                </Box>
              )}

              {activeTab === 1 && (
                <Box>
                  <Typography variant="h6" gutterBottom>
                    Governance Information
                  </Typography>
                  <Grid container spacing={2}>
                    <Grid item xs={12} md={6}>
                      <Typography variant="subtitle2">Status</Typography>
                      <Chip
                        label={selectedTemplate.governance.status}
                        color={getStatusColor(selectedTemplate.status) as any}
                        icon={getStatusIcon(selectedTemplate.status)}
                      />
                    </Grid>
                    <Grid item xs={12} md={6}>
                      <Typography variant="subtitle2">Steward Group</Typography>
                      <Typography>{selectedTemplate.governance.steward_group}</Typography>
                    </Grid>
                    <Grid item xs={12} md={6}>
                      <Typography variant="subtitle2">Refresh Frequency</Typography>
                      <Typography>{selectedTemplate.governance.sla.refresh_frequency}</Typography>
                    </Grid>
                    <Grid item xs={12} md={6}>
                      <Typography variant="subtitle2">Max Latency</Typography>
                      <Typography>{selectedTemplate.governance.sla.max_latency}</Typography>
                    </Grid>
                  </Grid>
                </Box>
              )}

              {activeTab === 2 && (
                <Box>
                  <Typography variant="h6" gutterBottom>
                    Review Comments ({selectedTemplate.review_comments.length})
                  </Typography>
                  {selectedTemplate.review_comments.length === 0 ? (
                    <Typography color="text.secondary">No comments yet</Typography>
                  ) : (
                    <List>
                      {selectedTemplate.review_comments.map((comment) => (
                        <ListItem key={comment.id}>
                          <ListItemText
                            primary={
                              <Box display="flex" alignItems="center" gap={1}>
                                <Typography variant="subtitle2">{comment.author}</Typography>
                                <Chip
                                  label={comment.type}
                                  size="small"
                                  color={
                                    comment.type === 'approval' ? 'success' :
                                    comment.type === 'rejection' ? 'error' :
                                    comment.type === 'change_request' ? 'warning' : 'default'
                                  }
                                />
                              </Box>
                            }
                            secondary={
                              <Box>
                                <Typography variant="body2">{comment.comment}</Typography>
                                <Typography variant="caption" color="text.secondary">
                                  {new Date(comment.created_at).toLocaleString()}
                                </Typography>
                              </Box>
                            }
                          />
                        </ListItem>
                      ))}
                    </List>
                  )}
                </Box>
              )}

              {activeTab === 3 && (
                <Box>
                  <Typography variant="h6" gutterBottom>
                    Version History
                  </Typography>
                  <Typography color="text.secondary">
                    Version history tracking would be implemented here
                  </Typography>
                </Box>
              )}
            </Paper>
          ) : (
            <Paper sx={{ p: 3, textAlign: 'center' }}>
              <Typography variant="h6" color="text.secondary">
                Select a template to view details
              </Typography>
            </Paper>
          )}
        </Grid>
      </Grid>

      {/* Review Dialog */}
      <Dialog open={reviewDialogOpen} onClose={() => setReviewDialogOpen(false)} maxWidth="md" fullWidth>
        <ModalHeader title="Template Review" onClose={() => setReviewDialogOpen(false)} />
        <DialogContent>
          {selectedTemplate && (
            <Box>
              <Typography variant="h6" gutterBottom>
                {selectedTemplate.name}
              </Typography>
              <Typography variant="body2" color="text.secondary" paragraph>
                {selectedTemplate.description}
              </Typography>

              <Accordion>
                <AccordionSummary expandIcon={<ExpandMore />}>
                  <Typography>Template JSON</Typography>
                </AccordionSummary>
                <AccordionDetails>
                  <Box sx={{ fontSize: '12px', overflow: 'auto', fontFamily: 'monospace' }}>
                    {JSON.stringify(selectedTemplate, null, 2)}
                  </Box>
                </AccordionDetails>
              </Accordion>
            </Box>
          )}
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setReviewDialogOpen(false)}>Close</Button>
        </DialogActions>
      </Dialog>

      {/* Comment Dialog */}
      <Dialog open={commentDialogOpen} onClose={() => setCommentDialogOpen(false)} maxWidth="sm" fullWidth>
        <ModalHeader title="Add Comment" onClose={() => setCommentDialogOpen(false)} />
        <DialogContent>
          <FormControl fullWidth sx={{ mt: 2, mb: 2 }}>
            <InputLabel>Comment Type</InputLabel>
            <Select
              value={commentType}
              label="Comment Type"
              onChange={(e) => setCommentType(e.target.value as any)}
            >
              <MenuItem value="comment">General Comment</MenuItem>
              <MenuItem value="approval">Approval</MenuItem>
              <MenuItem value="rejection">Rejection</MenuItem>
              <MenuItem value="change_request">Change Request</MenuItem>
            </Select>
          </FormControl>
          <TextField
            fullWidth
            multiline
            rows={4}
            label="Comment"
            value={newComment}
            onChange={(e) => setNewComment(e.target.value)}
          />
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setCommentDialogOpen(false)}>Cancel</Button>
          <Button onClick={handleAddComment} variant="contained">
            Add Comment
          </Button>
        </DialogActions>
      </Dialog>
    </Box>
  );
};
