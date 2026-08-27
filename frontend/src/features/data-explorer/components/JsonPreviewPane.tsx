import React, { useState } from 'react';
import {
  Box,
  Stack,
  Tooltip,
  IconButton,
  Paper,
  Typography,
} from '@mui/material';
import {
  ContentCopy as CopyIcon,
  Check as CheckIcon,
  DataObject as JsonIcon,
} from '@mui/icons-material';
import {
  EXPLORER_BG,
  EXPLORER_BORDER,
  EXPLORER_MUTED,
} from '../types/dataExplorerTypes';
import type { ExplorerResult } from '../types/dataExplorerTypes';

interface JsonPreviewPaneProps {
  result: ExplorerResult | null;
}

export const JsonPreviewPane: React.FC<JsonPreviewPaneProps> = ({ result }) => {
  const [copied, setCopied] = useState(false);

  if (!result) {
    return (
      <Box
        sx={{
          flex: 1,
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'center',
          bgcolor: EXPLORER_BG,
          p: 6,
        }}
      >
        <Typography variant="body2" sx={{ color: EXPLORER_MUTED }}>
          No result yet.
        </Typography>
      </Box>
    );
  }

  const payload = {
    columns: result.columns,
    rows: result.rows,
    rowCount: result.rowCount,
    executionTimeMs: result.executionTimeMs,
    warnings: result.warnings,
  };
  const json = JSON.stringify(payload, null, 2);

  const handleCopy = async () => {
    try {
      if (navigator?.clipboard?.writeText) {
        await navigator.clipboard.writeText(json);
      }
      setCopied(true);
      setTimeout(() => setCopied(false), 1500);
    } catch {
      // ignore
    }
  };

  return (
    <Box sx={{ flex: 1, display: 'flex', flexDirection: 'column', bgcolor: EXPLORER_BG, p: 3 }}>
      <Stack direction="row" justifyContent="space-between" alignItems="center" sx={{ mb: 2 }}>
        <Stack direction="row" spacing={1.5} alignItems="center">
          <JsonIcon fontSize="small" sx={{ color: EXPLORER_MUTED }} />
          <Typography variant="caption" sx={{ color: EXPLORER_MUTED, fontWeight: 700, letterSpacing: 1, textTransform: 'uppercase' }}>
            Raw JSON · {result.rowCount.toLocaleString()} rows · {result.executionTimeMs} ms
          </Typography>
        </Stack>
        <Tooltip title="Copy JSON">
          <IconButton onClick={handleCopy} size="small" sx={{ color: EXPLORER_MUTED }}>
            {copied ? <CheckIcon sx={{ color: '#10b981' }} /> : <CopyIcon />}
          </IconButton>
        </Tooltip>
      </Stack>
      <Paper
        elevation={0}
        sx={{
          flex: 1,
          minHeight: 320,
          overflow: 'auto',
          p: 3,
          borderRadius: 3,
          border: `1px solid ${EXPLORER_BORDER}`,
          bgcolor: '#0F172A',
        }}
      >
        <Box
          component="pre"
          sx={{
            m: 0,
            color: '#E2E8F0',
            fontFamily: 'monospace',
            fontSize: 13,
            whiteSpace: 'pre',
          }}
        >
          {json}
        </Box>
      </Paper>
    </Box>
  );
};

export default JsonPreviewPane;
