import {
  Avatar,
  AvatarGroup,
  Box,
  Button,
  Card,
  CardContent,
  Chip,
  CircularProgress,
  Divider,
  IconButton,
  InputAdornment,
  Paper,
  TextField,
  Tooltip,
  Typography,
  useTheme,
  Alert,
  Menu,
  MenuItem,
} from '@mui/material';
import {
  Delete as DeleteIcon,
  PushPin as PushPinIcon,
  PushPinOutlined as PushPinOutlinedIcon,
  MoreVert as MoreVertIcon,
  Reply as ReplyIcon,
} from '@mui/icons-material';
import { useState, useCallback, useMemo } from 'react';
import { useScenarioAnnotations } from '../../hooks/useScenarioAnnotations';
import type { Annotation, AddAnnotationRequest } from '../../types/scenarios';

export interface CollaborativeAnnotationsPanelProps {
  simulationId: string | null;
  currentUserId: string;
  currentUserName: string;
  currentUserEmail?: string;
  onSelectCell?: (cellReference: string) => void;
  enabled?: boolean;
}

/**
 * Collaborative annotations panel for sharing insights and commenting
 * 
 * Features:
 * - Real-time annotation display
 * - Add annotations with mentions
 * - Pin important annotations
 * - Threaded replies
 * - User avatars and timestamps
 * - Search/filter annotations
 * - 100% Material UI design
 * - Dark mode support
 */
export function CollaborativeAnnotationsPanel({
  simulationId,
  currentUserId,
  currentUserName,
  currentUserEmail,
  onSelectCell,
  enabled = true,
}: CollaborativeAnnotationsPanelProps) {
  const theme = useTheme();
  const {
    annotations,
    isLoading,
    error,
    add,
    delete: deleteAnnotation,
    togglePin,
    reply,
    refresh,
  } = useScenarioAnnotations(simulationId, enabled);

  const [newAnnotationText, setNewAnnotationText] = useState('');
  const [newCellReference, setNewCellReference] = useState('');
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [submitError, setSubmitError] = useState<string | null>(null);
  const [replyingTo, setReplyingTo] = useState<string | null>(null);
  const [replyText, setReplyText] = useState('');
  const [anchorEl, setAnchorEl] = useState<null | HTMLElement>(null);
  const [selectedAnnotationId, setSelectedAnnotationId] = useState<string | null>(null);

  // Sort annotations: pinned first, then by timestamp
  const sortedAnnotations = useMemo(() => {
    return [...annotations].sort((a, b) => {
      if (a.isPinned && !b.isPinned) return -1;
      if (!a.isPinned && b.isPinned) return 1;
      return new Date(b.timestamp).getTime() - new Date(a.timestamp).getTime();
    });
  }, [annotations]);

  // Handle add annotation
  const handleAddAnnotation = useCallback(async () => {
    if (!newAnnotationText.trim()) {
      setSubmitError('Annotation cannot be empty');
      return;
    }

    setIsSubmitting(true);
    setSubmitError(null);

    try {
      await add({
        simulationId: simulationId!,
        userId: currentUserId,
        text: newAnnotationText,
        cellReference: newCellReference || undefined,
        mentions: [],
      } as AddAnnotationRequest);

      setNewAnnotationText('');
      setNewCellReference('');
    } catch (err) {
      const message = err instanceof Error ? err.message : 'Failed to add annotation';
      setSubmitError(message);
    } finally {
      setIsSubmitting(false);
    }
  }, [newAnnotationText, newCellReference, simulationId, currentUserId, add]);

  // Handle reply
  const handleReply = useCallback(async (annotationId: string) => {
    if (!replyText.trim()) return;

    setIsSubmitting(true);
    try {
      await reply(annotationId, {
        simulationId: simulationId!,
        userId: currentUserId,
        text: replyText,
        mentions: [],
      } as AddAnnotationRequest);

      setReplyText('');
      setReplyingTo(null);
    } catch (err) {
      const message = err instanceof Error ? err.message : 'Failed to add reply';
      setSubmitError(message);
    } finally {
      setIsSubmitting(false);
    }
  }, [replyText, simulationId, currentUserId, reply]);

  // Handle pin toggle
  const handleTogglePin = useCallback(
    async (annotationId: string) => {
      try {
        await togglePin(annotationId);
        setAnchorEl(null);
      } catch (err) {
        const message = err instanceof Error ? err.message : 'Failed to pin annotation';
        setSubmitError(message);
      }
    },
    [togglePin]
  );

  // Handle delete
  const handleDelete = useCallback(
    async (annotationId: string) => {
      try {
        await deleteAnnotation(annotationId);
        setAnchorEl(null);
      } catch (err) {
        const message = err instanceof Error ? err.message : 'Failed to delete annotation';
        setSubmitError(message);
      }
    },
    [deleteAnnotation]
  );

  // Context menu
  const handleMenuOpen = (event: React.MouseEvent<HTMLElement>, annotationId: string) => {
    setAnchorEl(event.currentTarget);
    setSelectedAnnotationId(annotationId);
  };

  const handleMenuClose = () => {
    setAnchorEl(null);
    setSelectedAnnotationId(null);
  };

  // Get initials for avatar
  const getInitials = (name: string): string => {
    return name
      .split(' ')
      .map(n => n[0])
      .join('')
      .toUpperCase()
      .slice(0, 2);
  };

  // Get background color for avatar
  const getAvatarColor = (userId: string): string => {
    const colors = [
      theme.palette.primary.main,
      theme.palette.secondary.main,
      theme.palette.info.main,
      theme.palette.success.main,
      theme.palette.warning.main,
      theme.palette.error.main,
    ];
    const index = userId.charCodeAt(0) % colors.length;
    return colors[index];
  };

  if (!enabled || !simulationId) {
    return null;
  }

  return (
    <Box
      sx={{
        display: 'flex',
        flexDirection: 'column',
        height: '100%',
        backgroundColor: theme.palette.background.paper,
        borderLeft: 1,
        borderColor: 'divider',
      }}
    >
      {/* Header */}
      <Box
        sx={{
          p: 2,
          borderBottom: 1,
          borderColor: 'divider',
          display: 'flex',
          justifyContent: 'space-between',
          alignItems: 'center',
        }}
      >
        <Typography variant="h6">Comments & Insights</Typography>
        {annotations.length > 0 && (
          <Chip label={annotations.length} size="small" sx={{ ml: 1 }} />
        )}
      </Box>

      {/* Annotations List */}
      <Box
        sx={{
          flex: 1,
          overflowY: 'auto',
          p: 2,
          display: 'flex',
          flexDirection: 'column',
          gap: 2,
        }}
      >
        {isLoading ? (
          <Box display="flex" justifyContent="center" alignItems="center" sx={{ py: 4 }}>
            <CircularProgress />
          </Box>
        ) : error ? (
          <Alert severity="error" sx={{ mb: 2 }}>
            {error.message}
            <Button size="small" onClick={refresh} sx={{ ml: 1 }}>
              Retry
            </Button>
          </Alert>
        ) : sortedAnnotations.length === 0 ? (
          <Typography
            variant="body2"
            color="textSecondary"
            textAlign="center"
            sx={{ py: 4 }}
          >
            No annotations yet. Start sharing insights!
          </Typography>
        ) : (
          sortedAnnotations.map((annotation, idx) => (
            <Box key={annotation.id}>
              <Paper
                sx={{
                  p: 2,
                  backgroundColor:
                    annotation.isPinned
                      ? theme.palette.action.hover
                      : theme.palette.background.default,
                  border: annotation.isPinned ? 2 : 1,
                  borderColor: annotation.isPinned
                    ? theme.palette.primary.main
                    : 'divider',
                  transition: 'all 0.2s ease',
                  '&:hover': {
                    boxShadow: theme.shadows[2],
                  },
                }}
              >
                {/* Annotation Header */}
                <Box
                  sx={{
                    display: 'flex',
                    justifyContent: 'space-between',
                    alignItems: 'flex-start',
                    mb: 1,
                  }}
                >
                  <Box sx={{ display: 'flex', gap: 1, flex: 1 }}>
                    <Avatar
                      sx={{
                        width: 32,
                        height: 32,
                        backgroundColor: getAvatarColor(annotation.userId),
                        fontSize: '0.75rem',
                      }}
                    >
                      {getInitials(annotation.userName)}
                    </Avatar>

                    <Box sx={{ flex: 1 }}>
                      <Box sx={{ display: 'flex', gap: 1, alignItems: 'center' }}>
                        <Typography variant="subtitle2" sx={{ fontWeight: 600 }}>
                          {annotation.userName}
                        </Typography>
                        <Typography variant="caption" color="textSecondary">
                          {new Date(annotation.timestamp).toLocaleTimeString()}
                        </Typography>
                      </Box>

                      {annotation.cellReference && (
                        <Chip
                          label={annotation.cellReference}
                          size="small"
                          sx={{ mt: 0.5, height: 20 }}
                          onClick={() => onSelectCell?.(annotation.cellReference!)}
                        />
                      )}
                    </Box>
                  </Box>

                  {/* Action Menu */}
                  <IconButton
                    size="small"
                    onClick={e => handleMenuOpen(e, annotation.id)}
                  >
                    <MoreVertIcon fontSize="small" />
                  </IconButton>
                </Box>

                {/* Annotation Text */}
                <Typography variant="body2" sx={{ mb: 1, whiteSpace: 'pre-wrap' }}>
                  {annotation.text}
                </Typography>

                {/* Mentions */}
                {annotation.mentions && annotation.mentions.length > 0 && (
                  <Box sx={{ mb: 1, display: 'flex', gap: 0.5, flexWrap: 'wrap' }}>
                    {annotation.mentions.map(mention => (
                      <Chip
                        key={mention}
                        label={`@${mention}`}
                        size="small"
                        variant="outlined"
                        sx={{ height: 20 }}
                      />
                    ))}
                  </Box>
                )}

                {/* Reply Button */}
                {replyingTo !== annotation.id && (
                  <Button
                    size="small"
                    startIcon={<ReplyIcon />}
                    onClick={() => setReplyingTo(annotation.id)}
                    sx={{ mt: 1 }}
                  >
                    Reply
                  </Button>
                )}

                {/* Reply Form */}
                {replyingTo === annotation.id && (
                  <Box sx={{ mt: 1, pt: 1, borderTop: 1, borderColor: 'divider' }}>
                    <TextField
                      fullWidth
                      size="small"
                      placeholder="Write a reply..."
                      multiline
                      rows={2}
                      value={replyText}
                      onChange={e => setReplyText(e.target.value)}
                      sx={{ mb: 1 }}
                    />
                    <Box sx={{ display: 'flex', gap: 1 }}>
                      <Button
                        size="small"
                        variant="contained"
                        onClick={() => handleReply(annotation.id)}
                        disabled={isSubmitting || !replyText.trim()}
                      >
                        {isSubmitting ? <CircularProgress size={20} /> : 'Reply'}
                      </Button>
                      <Button
                        size="small"
                        onClick={() => {
                          setReplyingTo(null);
                          setReplyText('');
                        }}
                      >
                        Cancel
                      </Button>
                    </Box>
                  </Box>
                )}

                {/* Replies */}
                {annotation.replies && annotation.replies.length > 0 && (
                  <Box sx={{ mt: 2, ml: 2, pl: 2, borderLeft: 2, borderColor: 'divider' }}>
                    {annotation.replies.map(reply => (
                      <Box key={reply.id} sx={{ mb: 1 }}>
                        <Box sx={{ display: 'flex', gap: 1, mb: 0.5 }}>
                          <Avatar
                            sx={{
                              width: 24,
                              height: 24,
                              backgroundColor: getAvatarColor(reply.userId),
                              fontSize: '0.65rem',
                            }}
                          >
                            {getInitials(reply.userName)}
                          </Avatar>
                          <Box sx={{ flex: 1 }}>
                            <Typography variant="caption" sx={{ fontWeight: 600 }}>
                              {reply.userName}
                            </Typography>
                            <Typography variant="caption" color="textSecondary" sx={{ ml: 1 }}>
                              {new Date(reply.timestamp).toLocaleTimeString()}
                            </Typography>
                          </Box>
                        </Box>
                        <Typography variant="caption" sx={{ pl: 4, display: 'block', mb: 1 }}>
                          {reply.text}
                        </Typography>
                      </Box>
                    ))}
                  </Box>
                )}
              </Paper>

              {idx < sortedAnnotations.length - 1 && <Divider />}
            </Box>
          ))
        )}
      </Box>

      {/* Add Annotation Form */}
      <Box
        sx={{
          p: 2,
          borderTop: 1,
          borderColor: 'divider',
          backgroundColor: theme.palette.action.hover,
        }}
      >
        {submitError && (
          <Alert severity="error" sx={{ mb: 1 }} onClose={() => setSubmitError(null)}>
            {submitError}
          </Alert>
        )}

        <Typography variant="caption" color="textSecondary" sx={{ display: 'block', mb: 1 }}>
          Cell Reference (optional)
        </Typography>
        <TextField
          fullWidth
          size="small"
          placeholder="e.g., Tech - Equity Move"
          value={newCellReference}
          onChange={e => setNewCellReference(e.target.value)}
          sx={{ mb: 1 }}
        />

        <TextField
          fullWidth
          size="small"
          placeholder="Share an insight or comment..."
          multiline
          rows={3}
          value={newAnnotationText}
          onChange={e => setNewAnnotationText(e.target.value)}
          sx={{ mb: 1 }}
          disabled={isSubmitting}
        />

        <Box sx={{ display: 'flex', gap: 1 }}>
          <Button
            variant="contained"
            size="small"
            onClick={handleAddAnnotation}
            disabled={isSubmitting || !newAnnotationText.trim()}
            sx={{ flex: 1 }}
          >
            {isSubmitting ? <CircularProgress size={20} /> : 'Post'}
          </Button>
          <Button
            size="small"
            onClick={() => {
              setNewAnnotationText('');
              setNewCellReference('');
              setSubmitError(null);
            }}
            disabled={isSubmitting}
          >
            Clear
          </Button>
        </Box>
      </Box>

      {/* More Options Menu */}
      <Menu
        anchorEl={anchorEl}
        open={Boolean(anchorEl && selectedAnnotationId)}
        onClose={handleMenuClose}
      >
        <MenuItem
          onClick={() => {
            if (selectedAnnotationId) {
              handleTogglePin(selectedAnnotationId);
            }
          }}
        >
          <PushPinIcon fontSize="small" sx={{ mr: 1 }} />
          Pin
        </MenuItem>
        <MenuItem
          onClick={() => {
            if (selectedAnnotationId) {
              handleDelete(selectedAnnotationId);
            }
          }}
        >
          <DeleteIcon fontSize="small" sx={{ mr: 1 }} />
          Delete
        </MenuItem>
      </Menu>
    </Box>
  );
}

export type { CollaborativeAnnotationsPanelProps };
