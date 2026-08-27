import { useState, useEffect } from 'react';
import { useTheme } from '@mui/material/styles';
import Box from '@mui/material/Box';
import Typography from '@mui/material/Typography';
import TextField from '@mui/material/TextField';
import Paper from '@mui/material/Paper';

interface ContextPanelProps {
  kernel: any;
}

export function ContextPanel({kernel }: ContextPanelProps) {
  const theme = useTheme();
  const [context, setContext] = useState<any>(kernel.state.context);

  const isDark = theme.palette.mode === 'dark';

  const update = (key: any, value: any) => {
    const newCtx: any = { ...context, [key]: value }
    setContext(newCtx)
    kernel.state.context = newCtx
    kernel.events.dispatch("contextChanged", newCtx)
    kernel.services.persistence.save(kernel)
  }

  return (
    <Paper sx={{ p: 2, height: '100%', display: 'flex', flexDirection: 'column', overflow: 'hidden' }}>
      <Typography variant="h6" sx={{ mb: 2, fontWeight: 600 }}>
        Context
      </Typography>
      <Box sx={{ display: 'flex', flexDirection: 'column', gap: 2, overflow: 'auto' }}>
        {Object.entries(context).map(([k, v]: [any, any]) => (
          <Box key={k}>
            <Typography variant="caption" sx={{ fontWeight: 500, mb: 0.5, display: 'block' }}>
              {k}
            </Typography>
            <TextField
              fullWidth
              size="small"
              variant="outlined"
              value={v}
              onChange={(e: any) => update(k, e.target.value)}
            />
          </Box>
        ))}
      </Box>
    </Paper>
  )
}
