import React from 'react';
import {
  Box,
  Paper,
  Stack,
  Typography,
  Chip,
  Divider,
  List,
  ListItem,
  ListItemText,
  ListItemIcon,
} from '@mui/material';
import {
  Timeline,
  TimelineItem,
  TimelineSeparator,
  TimelineConnector,
  TimelineContent,
  TimelineDot,
  TimelineOppositeContent,
} from '@mui/lab';
import {
  Edit as EditIcon,
  Add as AddIcon,
  Delete as DeleteIcon,
  CheckCircle as ApproveIcon,
} from '@mui/icons-material';

interface AuditEntry {
  id: string;
  timestamp: string;
  user: string;
  action: 'created' | 'updated' | 'deleted' | 'approved' | 'rejected';
  changes?: Array<{ field: string; oldValue: any; newValue: any }>;
  comment?: string;
}

interface AuditTrailProps {
  ruleId: string;
  entries: AuditEntry[];
}

const actionIcons = {
  created: <AddIcon />,
  updated: <EditIcon />,
  deleted: <DeleteIcon />,
  approved: <ApproveIcon />,
  rejected: <DeleteIcon />,
};

const actionColors = {
  created: 'success',
  updated: 'primary',
  deleted: 'error',
  approved: 'success',
  rejected: 'warning',
} as const;

export const AuditTrail: React.FC<AuditTrailProps> = ({ ruleId, entries }) => {
  const formatDate = (timestamp: string) => {
    const date = new Date(timestamp);
    return date.toLocaleString();
  };

  const formatValue = (value: any) => {
    if (typeof value === 'object') {
      return JSON.stringify(value);
    }
    return String(value);
  };

  return (
    <Box>
      <Typography variant="h6" sx={{ fontWeight: 600, mb: 3 }}>
        Change History
      </Typography>

      {entries.length === 0 ? (
        <Paper elevation={0} sx={{ p: 3, bgcolor: 'grey.50', textAlign: 'center' }}>
          <Typography variant="body2" color="text.secondary">
            No change history available
          </Typography>
        </Paper>
      ) : (
        <Timeline position="right">
          {entries.map((entry, index) => (
            <TimelineItem key={entry.id}>
              <TimelineOppositeContent color="text.secondary" sx={{ flex: 0.3 }}>
                <Typography variant="caption">{formatDate(entry.timestamp)}</Typography>
                <Typography variant="caption" sx={{ display: 'block', fontWeight: 600 }}>
                  {entry.user}
                </Typography>
              </TimelineOppositeContent>

              <TimelineSeparator>
                <TimelineDot color={actionColors[entry.action]}>
                  {actionIcons[entry.action]}
                </TimelineDot>
                {index < entries.length - 1 && <TimelineConnector />}
              </TimelineSeparator>

              <TimelineContent>
                <Paper elevation={2} sx={{ p: 2, mb: 2 }}>
                  <Stack spacing={2}>
                    <Stack direction="row" spacing={1} alignItems="center">
                      <Chip
                        label={entry.action.toUpperCase()}
                        size="small"
                        color={actionColors[entry.action]}
                      />
                    </Stack>

                    {entry.comment && (
                      <Typography variant="body2" color="text.secondary">
                        {entry.comment}
                      </Typography>
                    )}

                    {entry.changes && entry.changes.length > 0 && (
                      <Box>
                        <Divider sx={{ my: 1 }} />
                        <Typography variant="caption" color="text.secondary" sx={{ display: 'block', mb: 1 }}>
                          Changes:
                        </Typography>
                        <Stack spacing={1}>
                          {entry.changes.map((change, idx) => (
                            <Paper key={idx} elevation={0} sx={{ p: 1, bgcolor: 'grey.50' }}>
                              <Typography variant="caption" sx={{ fontWeight: 600 }}>
                                {change.field}
                              </Typography>
                              <Stack direction="row" spacing={1} alignItems="center" sx={{ mt: 0.5 }}>
                                <Chip
                                  label={formatValue(change.oldValue) || '(empty)'}
                                  size="small"
                                  variant="outlined"
                                  color="error"
                                />
                                <Typography variant="caption">→</Typography>
                                <Chip
                                  label={formatValue(change.newValue) || '(empty)'}
                                  size="small"
                                  variant="outlined"
                                  color="success"
                                />
                              </Stack>
                            </Paper>
                          ))}
                        </Stack>
                      </Box>
                    )}
                  </Stack>
                </Paper>
              </TimelineContent>
            </TimelineItem>
          ))}
        </Timeline>
      )}
    </Box>
  );
};

export default AuditTrail;
