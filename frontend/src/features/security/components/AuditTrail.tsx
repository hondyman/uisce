import React from 'react';
import {
  Box,
  Paper,
  Stack,
  Typography,
  Chip,
  Divider,
} from '@mui/material';
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

const TimelineDot: React.FC<{ color: string; children: React.ReactNode }> = ({ color, children }) => (
  <Box
    sx={{
      width: 32,
      height: 32,
      borderRadius: '50%',
      display: 'flex',
      alignItems: 'center',
      justifyContent: 'center',
      bgcolor: `${color}.main`,
      color: 'white',
      flexShrink: 0,
    }}
  >
    {children}
  </Box>
);

const TimelineConnector: React.FC = () => (
  <Box
    sx={{
      width: 2,
      flex: 1,
      bgcolor: 'divider',
      minHeight: 24,
      mx: 'auto',
    }}
  />
);

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
        <Box sx={{ display: 'flex', flexDirection: 'column', gap: 0 }}>
          {entries.map((entry, index) => (
            <Box key={entry.id} sx={{ display: 'flex', gap: 2 }}>
              <Box sx={{ display: 'flex', flexDirection: 'column', alignItems: 'center', flexShrink: 0 }}>
                <TimelineDot color={actionColors[entry.action]}>
                  {actionIcons[entry.action]}
                </TimelineDot>
                {index < entries.length - 1 && <TimelineConnector />}
              </Box>

              <Box sx={{ flex: 1, pb: index < entries.length - 1 ? 3 : 0 }}>
                <Box sx={{ display: 'flex', gap: 2, mb: 1 }}>
                  <Typography variant="caption" color="text.secondary" sx={{ flex: 0.3 }}>
                    {formatDate(entry.timestamp)}
                  </Typography>
                  <Typography variant="caption" sx={{ fontWeight: 600 }}>
                    {entry.user}
                  </Typography>
                </Box>

                <Paper elevation={2} sx={{ p: 2 }}>
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
              </Box>
            </Box>
          ))}
        </Box>
      )}
    </Box>
  );
};

export default AuditTrail;
