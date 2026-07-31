import React, { useState } from 'react';
import { Box, TextField, Button, Switch, FormControlLabel, Typography, Paper } from '@mui/material';
import HistoryIcon from '@mui/icons-material/History';
import { usePageContextStore } from '../../store/usePageContextStore';

export const LookbackToolbar: React.FC = () => {
  const [enabled, setEnabled] = useState(false);
  const [timestamp, setTimestamp] = useState<string>(
    new Date().toISOString().slice(0, 16)
  );

  const setContextValue = usePageContextStore((state) => state.setContextValue);
  const clearContextValue = usePageContextStore((state) => state.clearContextValue);

  const handleToggle = (event: React.ChangeEvent<HTMLInputElement>) => {
    const isChecked = event.target.checked;
    setEnabled(isChecked);
    if (isChecked) {
      setContextValue('lookback', { enabled: true, as_of_timestamp: new Date(timestamp).toISOString() });
    } else {
      clearContextValue('lookback');
    }
  };

  const handleApply = () => {
    if (enabled) {
      setContextValue('lookback', { enabled: true, as_of_timestamp: new Date(timestamp).toISOString() });
    }
  };

  return (
    <Paper
      elevation={1}
      sx={{
        p: 1.5,
        mb: 2,
        display: 'flex',
        alignItems: 'center',
        gap: 3,
        bgcolor: enabled ? 'rgba(234, 179, 8, 0.12)' : '#1e293b',
        border: enabled ? '1px solid #facc15' : '1px solid #334155',
        color: '#f8fafc',
      }}
    >
      <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
        <HistoryIcon sx={{ color: enabled ? '#facc15' : '#94a3b8' }} />
        <Typography variant="subtitle2" fontWeight="bold">
          Point-in-Time Time-Travel
        </Typography>
      </Box>

      <FormControlLabel
        control={<Switch checked={enabled} onChange={handleToggle} color="warning" />}
        label={enabled ? 'Historical Mode Active' : 'Live Data Mode'}
        sx={{ color: enabled ? '#facc15' : '#94a3b8' }}
      />

      {enabled && (
        <Box sx={{ display: 'flex', alignItems: 'center', gap: 2 }}>
          <TextField
            label="As Of Timestamp"
            type="datetime-local"
            size="small"
            value={timestamp}
            onChange={(e) => setTimestamp(e.target.value)}
            InputLabelProps={{ shrink: true }}
            sx={{ bgcolor: '#0f172a', input: { color: '#fff' } }}
          />
          <Button variant="contained" size="small" color="warning" onClick={handleApply}>
            Apply Time-Travel
          </Button>
        </Box>
      )}
    </Paper>
  );
};
