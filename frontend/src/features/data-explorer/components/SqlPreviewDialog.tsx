import React, { useState } from 'react';
import {
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  Button,
  Box,
  Stack,
  Typography,
  IconButton,
  Tooltip,
  Chip,
} from '@mui/material';
import {
  ContentCopy as CopyIcon,
  Check as CheckIcon,
  Code as CodeIcon,
  Download as DownloadIcon,
} from '@mui/icons-material';
import {
  EXPLORER_ACCENT,
  EXPLORER_BG,
  EXPLORER_BORDER,
  EXPLORER_MUTED,
  EXPLORER_TEXT,
} from '../types/dataExplorerTypes';

interface SqlPreviewDialogProps {
  open: boolean;
  onClose: () => void;
  sql: string;
  dialect?: string;
  warningCount?: number;
}

export const SqlPreviewDialog: React.FC<SqlPreviewDialogProps> = ({
  open,
  onClose,
  sql,
  dialect,
  warningCount,
}) => {
  const [copied, setCopied] = useState(false);

  const handleCopy = async () => {
    try {
      if (navigator?.clipboard?.writeText) {
        await navigator.clipboard.writeText(sql);
      }
      setCopied(true);
      setTimeout(() => setCopied(false), 1500);
    } catch {
      // ignore
    }
  };

  const handleDownload = () => {
    const blob = new Blob([sql], { type: 'text/plain' });
    const url = window.URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = 'data-explorer-query.sql';
    a.click();
    window.URL.revokeObjectURL(url);
  };

  return (
    <Dialog open={open} onClose={onClose} maxWidth="md" fullWidth>
      <DialogTitle sx={{ p: 0 }}>
        <Stack
          direction="row"
          justifyContent="space-between"
          alignItems="center"
          sx={{
            px: 3,
            py: 2,
            borderBottom: `1px solid ${EXPLORER_BORDER}`,
          }}
        >
          <Stack direction="row" spacing={1.5} alignItems="center">
            <Box
              sx={{
                width: 36,
                height: 36,
                borderRadius: 2,
                bgcolor: EXPLORER_ACCENT,
                color: EXPLORER_TEXT,
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'center',
              }}
            >
              <CodeIcon sx={{ color: EXPLORER_TEXT }} />
            </Box>
            <Box>
              <Typography variant="subtitle1" fontWeight={700} sx={{ color: EXPLORER_TEXT }}>
                Generated SQL
              </Typography>
              <Stack direction="row" spacing={1} alignItems="center">
                {dialect && (
                  <Chip
                    size="small"
                    label={dialect.toUpperCase()}
                    sx={{
                      height: 18,
                      fontSize: 10,
                      fontWeight: 700,
                      bgcolor: EXPLORER_BG,
                      color: EXPLORER_TEXT,
                    }}
                  />
                )}
                {typeof warningCount === 'number' && warningCount > 0 && (
                  <Chip
                    size="small"
                    label={`${warningCount} warning${warningCount > 1 ? 's' : ''}`}
                    sx={{
                      height: 18,
                      fontSize: 10,
                      fontWeight: 700,
                      bgcolor: '#fef3c7',
                      color: '#92400e',
                    }}
                  />
                )}
              </Stack>
            </Box>
          </Stack>
          <Stack direction="row" spacing={0.5}>
            <Tooltip title="Copy SQL">
              <IconButton onClick={handleCopy} sx={{ color: EXPLORER_MUTED }}>
                {copied ? <CheckIcon sx={{ color: '#10b981' }} /> : <CopyIcon />}
              </IconButton>
            </Tooltip>
            <Tooltip title="Download .sql file">
              <IconButton onClick={handleDownload} sx={{ color: EXPLORER_MUTED }}>
                <DownloadIcon />
              </IconButton>
            </Tooltip>
          </Stack>
        </Stack>
      </DialogTitle>
      <DialogContent sx={{ p: 0, bgcolor: '#0F172A' }}>
        <Box
          component="pre"
          sx={{
            m: 0,
            p: 3,
            color: '#E2E8F0',
            fontFamily: 'monospace',
            fontSize: 13,
            whiteSpace: 'pre-wrap',
            wordBreak: 'break-word',
            minHeight: 200,
            maxHeight: '60vh',
            overflow: 'auto',
          }}
        >
          {sql || '-- Select dimensions or measures to generate SQL.'}
        </Box>
      </DialogContent>
      <DialogActions sx={{ px: 3, py: 2, borderTop: `1px solid ${EXPLORER_BORDER}` }}>
        <Button onClick={onClose} sx={{ textTransform: 'none', color: EXPLORER_MUTED }}>
          Close
        </Button>
      </DialogActions>
    </Dialog>
  );
};

export default SqlPreviewDialog;
