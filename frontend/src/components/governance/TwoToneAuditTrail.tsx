import React, { useState } from 'react';
import {
  Box,
  Typography,
  Paper,
  Chip,
  Button,
  TextField,
  Stack,
} from '@mui/material';
import HistoryIcon from '@mui/icons-material/History';
import SmartToyIcon from '@mui/icons-material/SmartToy';
import PersonIcon from '@mui/icons-material/Person';
import SendIcon from '@mui/icons-material/Send';

export interface TwoToneCommentItem {
  commentId: string;
  authorType: 'SYSTEM' | 'USER';
  authorIdentity: string;
  actionTaken: string;
  commentText: string;
  colorTone: 'RED' | 'BLUE';
  createdAt: string;
}

interface TwoToneAuditTrailProps {
  entityTitle?: string;
  comments?: TwoToneCommentItem[];
  onAddComment?: (text: string) => Promise<void>;
}

export const TwoToneAuditTrail: React.FC<TwoToneAuditTrailProps> = ({
  entityTitle = 'Concentration Violation: Tech Sector > 20%',
  comments = [
    {
      commentId: 'c1',
      authorType: 'SYSTEM',
      authorIdentity: 'SYSTEM',
      actionTaken: 'Status set to New (Violation Detected)',
      commentText: 'RESPONSE: Status set to "New" (Violation Detected)',
      colorTone: 'RED',
      createdAt: '2026-08-23 09:15:02',
    },
    {
      commentId: 'c2',
      authorType: 'USER',
      authorIdentity: 'TRADER_JANE',
      actionTaken: 'Status set to Closed - Corrected',
      commentText: 'Closed with client consent via email authorization',
      colorTone: 'BLUE',
      createdAt: '2026-08-23 09:48:37',
    },
    {
      commentId: 'c3',
      authorType: 'SYSTEM',
      authorIdentity: 'SYSTEM',
      actionTaken: 'ALERT REOPENED',
      commentText: 'ALERT REOPENED: Market value drift (14.2%) > 10% tolerance',
      colorTone: 'RED',
      createdAt: '2026-08-23 11:30:15',
    },
  ],
  onAddComment,
}) => {
  const [newComment, setNewComment] = useState('');
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [errorMsg, setErrorMsg] = useState<string | null>(null);

  const handleSubmit = async () => {
    if (!newComment.trim()) {
      setErrorMsg('Comment is required by test-specific closing privileges; operation failed.');
      return;
    }
    setErrorMsg(null);
    setIsSubmitting(true);
    try {
      if (onAddComment) {
        await onAddComment(newComment);
      }
      setNewComment('');
    } finally {
      setIsSubmitting(false);
    }
  };

  return (
    <Box sx={{ width: '100%', bgcolor: '#050D1A', color: '#fff', borderRadius: 2, border: '1px solid #1E293B', overflow: 'hidden', fontFamily: 'sans-serif' }}>
      
      {/* Header Bar */}
      <Box sx={{ p: 2, bgcolor: '#071526', borderBottom: '1px solid #1E293B', display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
        <Box sx={{ display: 'flex', alignItems: 'center', gap: 1.5 }}>
          <HistoryIcon sx={{ color: '#00D4FF', fontSize: 22 }} />
          <Box>
            <Typography variant="subtitle2" fontWeight="700" sx={{ letterSpacing: 0.5 }}>
              Two-Tone Institutional Audit Trail
            </Typography>
            <Typography variant="caption" sx={{ color: '#64748B' }}>
              Target: <span style={{ color: '#E2E8F0' }}>{entityTitle}</span>
            </Typography>
          </Box>
        </Box>

        <Stack direction="row" spacing={1}>
          <Chip size="small" label="System Logs (Red)" sx={{ bgcolor: 'rgba(239, 68, 68, 0.15)', color: '#EF4444', fontWeight: 700, fontSize: '10px' }} />
          <Chip size="small" label="User Justifications (Blue)" sx={{ bgcolor: 'rgba(59, 130, 246, 0.15)', color: '#3B82F6', fontWeight: 700, fontSize: '10px' }} />
        </Stack>
      </Box>

      {/* Comment History Frame */}
      <Box sx={{ p: 2.5, maxHeight: 360, overflowY: 'auto' }}>
        <Stack spacing={2}>
          {comments.map((c) => {
            const isSystem = c.authorType === 'SYSTEM';
            return (
              <Paper
                key={c.commentId}
                sx={{
                  p: 1.5,
                  bgcolor: '#071526',
                  borderLeft: '4px solid',
                  borderColor: isSystem ? '#EF4444' : '#3B82F6',
                  borderRadius: 1,
                }}
              >
                <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', mb: 0.5 }}>
                  <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                    {isSystem ? <SmartToyIcon sx={{ fontSize: 16, color: '#EF4444' }} /> : <PersonIcon sx={{ fontSize: 16, color: '#3B82F6' }} />}
                    <Typography variant="caption" fontWeight="700" sx={{ color: isSystem ? '#EF4444' : '#3B82F6' }}>
                      {c.authorIdentity}
                    </Typography>
                    <Typography variant="caption" sx={{ color: '#64748B', fontSize: '10px' }}>
                      &bull; {c.actionTaken}
                    </Typography>
                  </Box>
                  <Typography variant="caption" sx={{ color: '#475569', fontSize: '10px', fontFamily: 'monospace' }}>
                    {c.createdAt}
                  </Typography>
                </Box>

                <Typography
                  variant="body2"
                  sx={{
                    color: isSystem ? '#FCA5A5' : '#93C5FD',
                    fontFamily: isSystem ? 'monospace' : 'inherit',
                    fontSize: '11px',
                    pl: 3,
                  }}
                >
                  {c.commentText}
                </Typography>
              </Paper>
            );
          })}
        </Stack>
      </Box>

      {/* Mandatory Human Input Box */}
      <Box sx={{ p: 2, bgcolor: '#0B1E36', borderTop: '1px solid #1E293B' }}>
        {errorMsg && (
          <Typography variant="caption" sx={{ color: '#EF4444', display: 'block', mb: 1, fontWeight: 600 }}>
            {errorMsg}
          </Typography>
        )}
        <Box sx={{ display: 'flex', gap: 1.5 }}>
          <TextField
            size="small"
            fullWidth
            placeholder="Enter mandatory compliance rationale before overriding or closing alert..."
            value={newComment}
            onChange={(e) => setNewComment(e.target.value)}
            sx={{ bgcolor: '#050D1A', input: { color: '#fff', fontSize: '11px' } }}
          />
          <Button
            variant="contained"
            size="small"
            endIcon={<SendIcon />}
            onClick={handleSubmit}
            disabled={isSubmitting}
            sx={{ bgcolor: '#3B82F6', textTransform: 'none', fontWeight: 700, fontSize: '11px', px: 2.5 }}
          >
            Post Note
          </Button>
        </Box>
      </Box>

    </Box>
  );
};

export default TwoToneAuditTrail;
