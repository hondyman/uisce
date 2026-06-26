import React, { useMemo } from 'react';
import {
  Paper,
  Typography,
  Box,
  Grid,
  Chip,
  Stack,
  Divider,
} from '@mui/material';
import {
  CompareArrows as CompareIcon,
  ArrowForward as ArrowIcon,
} from '@mui/icons-material';
import { EntitySnapshot } from '../../../api/auditApi';
import { format, parseISO } from 'date-fns';

interface Props {
  leftVersion: EntitySnapshot;
  rightVersion: EntitySnapshot;
}

const EntityDiffViewer: React.FC<Props> = ({ leftVersion, rightVersion }) => {
  // Calculate differences
  const { added, removed, modified } = useMemo(() => {
    const leftData = leftVersion.entity_data;
    const rightData = rightVersion.entity_data;

    const added: string[] = [];
    const removed: string[] = [];
    const modified: string[] = [];

    // Find added and modified keys
    Object.keys(rightData).forEach((key) => {
      if (!(key in leftData)) {
        added.push(key);
      } else if (JSON.stringify(leftData[key]) !== JSON.stringify(rightData[key])) {
        modified.push(key);
      }
    });

    // Find removed keys
    Object.keys(leftData).forEach((key) => {
      if (!(key in rightData)) {
        removed.push(key);
      }
    });

    return { added, removed, modified };
  }, [leftVersion, rightVersion]);

  const leftJson = JSON.stringify(leftVersion.entity_data, null, 2);
  const rightJson = JSON.stringify(rightVersion.entity_data, null, 2);

  return (
    <Paper sx={{ p: 3 }}>
      <Stack direction="row" alignItems="center" spacing={2} sx={{ mb: 3 }}>
        <CompareIcon />
        <Typography variant="h6">Version Comparison</Typography>
      </Stack>

      {/* Version Info */}
      <Grid container spacing={2} sx={{ mb: 3 }}>
        <Grid item xs={12} md={6}>
          <Paper variant="outlined" sx={{ p: 2, bgcolor: 'error.50' }}>
            <Typography variant="subtitle2" color="error" gutterBottom>
              OLD VERSION
            </Typography>
            <Typography variant="body2" color="text.secondary">
              <strong>Change:</strong> {leftVersion.change_type}
            </Typography>
            <Typography variant="body2" color="text.secondary">
              <strong>Time:</strong> {format(parseISO(leftVersion.system_from), 'PPpp')}
            </Typography>
            <Typography variant="body2" color="text.secondary">
              <strong>By:</strong> {leftVersion.changed_by}
            </Typography>
          </Paper>
        </Grid>

        <Grid item xs={12} md={6}>
          <Paper variant="outlined" sx={{ p: 2, bgcolor: 'success.50' }}>
            <Typography variant="subtitle2" color="success.dark" gutterBottom>
              NEW VERSION
            </Typography>
            <Typography variant="body2" color="text.secondary">
              <strong>Change:</strong> {rightVersion.change_type}
            </Typography>
            <Typography variant="body2" color="text.secondary">
              <strong>Time:</strong> {format(parseISO(rightVersion.system_from), 'PPpp')}
            </Typography>
            <Typography variant="body2" color="text.secondary">
              <strong>By:</strong> {rightVersion.changed_by}
            </Typography>
          </Paper>
        </Grid>
      </Grid>

      {/* Change Summary */}
      <Stack direction="row" spacing={1} sx={{ mb: 2 }}>
        {added.length > 0 && (
          <Chip label={`${added.length} added`} color="success" size="small" />
        )}
        {modified.length > 0 && (
          <Chip label={`${modified.length} modified`} color="info" size="small" />
        )}
        {removed.length > 0 && (
          <Chip label={`${removed.length} removed`} color="error" size="small" />
        )}
      </Stack>

      <Divider sx={{ my: 2 }} />

      {/* Side-by-Side JSON View */}
      <Grid container spacing={2} sx={{ mb: 3 }}>
        <Grid item xs={12} md={6}>
          <Typography variant="subtitle2" gutterBottom>
            Old Version
          </Typography>
          <Paper
            variant="outlined"
            sx={{
              p: 2,
              bgcolor: 'grey.50',
              maxHeight: 400,
              overflow: 'auto',
              fontFamily: 'monospace',
              fontSize: '0.875rem',
            }}
          >
            <pre style={{ margin: 0 }}>{leftJson}</pre>
          </Paper>
        </Grid>

        <Grid item xs={12} md={6}>
          <Typography variant="subtitle2" gutterBottom>
            New Version
          </Typography>
          <Paper
            variant="outlined"
            sx={{
              p: 2,
              bgcolor: 'grey.50',
              maxHeight: 400,
              overflow: 'auto',
              fontFamily: 'monospace',
              fontSize: '0.875rem',
            }}
          >
            <pre style={{ margin: 0 }}>{rightJson}</pre>
          </Paper>
        </Grid>
      </Grid>

      {/* Field-by-Field Changes */}
      {(added.length > 0 || modified.length > 0 || removed.length > 0) && (
        <Box sx={{ mt: 3 }}>
          <Typography variant="subtitle2" gutterBottom>
            Field Changes
          </Typography>

          {added.map((key) => (
            <Box key={key} sx={{ mb: 1, p: 1, bgcolor: 'success.50', borderRadius: 1 }}>
              <Typography variant="body2" color="success.dark">
                <strong>+ {key}:</strong> {JSON.stringify(rightVersion.entity_data[key])}
              </Typography>
            </Box>
          ))}

          {modified.map((key) => (
            <Box key={key} sx={{ mb: 1, p: 1, bgcolor: 'info.50', borderRadius: 1 }}>
              <Stack direction="row" alignItems="center" spacing={1}>
                <Typography variant="body2" color="text.secondary">
                  <strong>{key}:</strong> {JSON.stringify(leftVersion.entity_data[key])}
                </Typography>
                <ArrowIcon fontSize="small" />
                <Typography variant="body2" color="info.dark">
                  {JSON.stringify(rightVersion.entity_data[key])}
                </Typography>
              </Stack>
            </Box>
          ))}

          {removed.map((key) => (
            <Box key={key} sx={{ mb: 1, p: 1, bgcolor: 'error.50', borderRadius: 1 }}>
              <Typography variant="body2" color="error.dark">
                <strong>- {key}:</strong> {JSON.stringify(leftVersion.entity_data[key])}
              </Typography>
            </Box>
          ))}
        </Box>
      )}
    </Paper>
  );
};

export default EntityDiffViewer;
