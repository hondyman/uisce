import React, { useState } from 'react';
import { Box, Typography, TextField, Button, Paper, Grid } from '@mui/material';

interface StreamingConfig {
  topicName: string;
  windowType: 'TUMBLE' | 'HOP' | 'SESSION';
  windowInterval: string;
  watermarkDelay: string;
}

export const StreamingBindingPanel: React.FC = () => {
  const [config, setConfig] = useState<StreamingConfig>({
    topicName: 'redpanda.market_ticks.live',
    windowType: 'TUMBLE',
    windowInterval: '5 MINUTE',
    watermarkDelay: '5 SECOND',
  });

  // Generate a live preview of the Flink SQL the compiler will create
  const generatePreviewSQL = () => {
    return `SELECT window_start, window_end, \n  JSON_VALUE(payload, '$.trade_amount') AS trade_amount\nFROM TABLE(\n  ${config.windowType}(TABLE ${config.topicName}, DESCRIPTOR(proctime), INTERVAL '${config.windowInterval}')\n)\nWHERE JSON_VALUE(payload, '$.tenant_id') = '<INJECTED_AT_RUNTIME>'\nGROUP BY window_start, window_end, trade_amount`;
  };

  return (
    <Paper sx={{ bgcolor: '#fff', border: '1px solid', borderColor: 'divider', borderRadius: 2, p: 3, boxShadow: 1, maxWidth: 900 }}>
      <Box sx={{ display: 'flex', alignItems: 'center', gap: 1.5, mb: 3, borderBottom: '1px solid', borderColor: 'divider', pb: 2 }}>
        <Box sx={{ width: 32, height: 32, borderRadius: '50%', bgcolor: 'warning.light', display: 'flex', alignItems: 'center', justifyContent: 'center', color: 'warning.main' }}>
          ⚡
        </Box>
        <Box>
          <Typography variant="h6" sx={{ fontWeight: 700, color: 'text.primary' }}>CEP Streaming Binding</Typography>
          <Typography variant="caption" sx={{ color: 'text.secondary' }}>Bind this Business Object to a continuous Redpanda/Flink stream.</Typography>
        </Box>
      </Box>

      <Grid container spacing={3}>
        <Grid size={{ xs: 12, md: 6 }}>
          <Box sx={{ display: 'flex', flexDirection: 'column', gap: 2 }}>
            <Box>
              <Typography variant="caption" sx={{ fontWeight: 600, color: 'text.secondary', textTransform: 'uppercase', display: 'block', mb: 0.5 }}>Source Topic</Typography>
              <TextField
                fullWidth
                size="small"
                value={config.topicName}
                onChange={e => setConfig({...config, topicName: e.target.value})}
                sx={{
                  '& .MuiOutlinedInput-root': {
                    fontFamily: 'monospace',
                    fontSize: '0.875rem',
                  },
                }}
              />
            </Box>

            <Grid container spacing={2}>
              <Grid size={{ xs: 6 }}>
                <Typography variant="caption" sx={{ fontWeight: 600, color: 'text.secondary', textTransform: 'uppercase', display: 'block', mb: 0.5 }}>Window Type</Typography>
                <TextField
                  select
                  fullWidth
                  size="small"
                  value={config.windowType}
                  onChange={e => setConfig({...config, windowType: e.target.value as any})}
                  SelectProps={{ native: true }}
                >
                  <option value="TUMBLE">Tumble (Fixed)</option>
                  <option value="HOP">Hop (Sliding)</option>
                  <option value="SESSION">Session</option>
                </TextField>
              </Grid>
              <Grid size={{ xs: 6 }}>
                <Typography variant="caption" sx={{ fontWeight: 600, color: 'text.secondary', textTransform: 'uppercase', display: 'block', mb: 0.5 }}>Interval</Typography>
                <TextField
                  fullWidth
                  size="small"
                  value={config.windowInterval}
                  onChange={e => setConfig({...config, windowInterval: e.target.value})}
                  placeholder="e.g., 5 MINUTE"
                  sx={{
                    '& .MuiOutlinedInput-root': {
                      fontFamily: 'monospace',
                      fontSize: '0.875rem',
                    },
                  }}
                />
              </Grid>
            </Grid>

            <Box>
              <Typography variant="caption" sx={{ fontWeight: 600, color: 'text.secondary', textTransform: 'uppercase', display: 'block', mb: 0.5 }}>Watermark Lateness</Typography>
              <TextField
                fullWidth
                size="small"
                value={config.watermarkDelay}
                onChange={e => setConfig({...config, watermarkDelay: e.target.value})}
                sx={{
                  '& .MuiOutlinedInput-root': {
                    fontFamily: 'monospace',
                    fontSize: '0.875rem',
                  },
                }}
              />
            </Box>
            
            <Button variant="contained" sx={{ bgcolor: '#111827', '&:hover': { bgcolor: '#1f2937' }, fontWeight: 700, borderRadius: 1, py: 1 }}>
              Save Streaming Binding
            </Button>
          </Box>
        </Grid>

        <Grid size={{ xs: 12, md: 6 }}>
          <Paper sx={{ bgcolor: '#111827', borderRadius: 1, p: 2, height: '100%', display: 'flex', flexDirection: 'column' }}>
            <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', mb: 1 }}>
              <Typography variant="caption" sx={{ fontWeight: 600, color: '#9ca3af', textTransform: 'uppercase', letterSpacing: '0.05em' }}>Generated Flink SQL Preview</Typography>
              <Box sx={{ display: 'flex', height: 8, width: 8, position: 'relative' }}>
                <Box sx={{ position: 'absolute', inset: 0, borderRadius: '50%', bgcolor: '#4ade80', animation: 'ping 1s cubic-bezier(0, 0, 0.2, 1) infinite', opacity: 0.75 }} />
                <Box sx={{ position: 'relative', borderRadius: '50%', bgcolor: '#22c55e', width: 8, height: 8 }} />
              </Box>
            </Box>
            <Box component="pre" sx={{ color: '#4ade80', fontFamily: 'monospace', fontSize: '0.75rem', overflowX: 'auto', flex: 1, mt: 1 }}>
              {generatePreviewSQL()}
            </Box>
            <Box sx={{ mt: 2, pt: 1.5, borderTop: '1px solid #374151', color: '#6b7280', fontSize: '0.6875rem' }}>
              * Tenant scoping and ABAC policies are automatically injected at compilation time.
            </Box>
          </Paper>
        </Grid>
      </Grid>
    </Paper>
  );
};

export default StreamingBindingPanel;
