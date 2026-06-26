import React, { useEffect, useState } from 'react';
import {
  Dialog,
  DialogTitle,
  DialogContent,
  Tabs,
  Tab,
  Box,
  IconButton,
  Typography,
  CircularProgress,
  Alert,
} from '@mui/material';
import CloseIcon from '@mui/icons-material/Close';
import ContentCopyIcon from '@mui/icons-material/ContentCopy';

interface PreAggSQL {
  iceberg_sql: string;
  starrocks_mv_sql: string;
}

interface PreAggSQLModalProps {
  open: boolean;
  onClose: () => void;
  preAggId: string;
  preAggName?: string;
}

export const PreAggSQLModal: React.FC<PreAggSQLModalProps> = ({
  open,
  onClose,
  preAggId,
  preAggName,
}) => {
  const [tab, setTab] = useState<'iceberg' | 'starrocks'>('iceberg');
  const [sql, setSql] = useState<PreAggSQL | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [copied, setCopied] = useState(false);

  useEffect(() => {
    if (!open || !preAggId) return;

    setLoading(true);
    setError(null);
    
    fetch(`/api/preaggs/${preAggId}/sql`)
      .then((r) => {
        if (!r.ok) throw new Error('Failed to fetch SQL');
        return r.json();
      })
      .then((data: PreAggSQL) => {
        setSql(data);
        setLoading(false);
      })
      .catch((err) => {
        setError(err.message);
        setLoading(false);
      });
  }, [open, preAggId]);

  const handleCopy = () => {
    if (!sql) return;
    const text = tab === 'iceberg' ? sql.iceberg_sql : sql.starrocks_mv_sql;
    navigator.clipboard.writeText(text);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  };

  const currentSQL = sql
    ? tab === 'iceberg'
      ? sql.iceberg_sql
      : sql.starrocks_mv_sql
    : '';

  return (
    <Dialog open={open} onClose={onClose} maxWidth="md" fullWidth>
      <DialogTitle
        sx={{
          display: 'flex',
          justifyContent: 'space-between',
          alignItems: 'center',
        }}
      >
        <Typography variant="h6">
          View SQL{preAggName ? `: ${preAggName}` : ''}
        </Typography>
        <IconButton onClick={onClose} size="small">
          <CloseIcon />
        </IconButton>
      </DialogTitle>

      <Box sx={{ borderBottom: 1, borderColor: 'divider', px: 2 }}>
        <Tabs
          value={tab}
          onChange={(_, v) => setTab(v)}
          aria-label="SQL type tabs"
        >
          <Tab label="Iceberg Rollup" value="iceberg" />
          <Tab label="StarRocks MV" value="starrocks" />
        </Tabs>
      </Box>

      <DialogContent sx={{ pt: 2 }}>
        {loading && (
          <Box sx={{ display: 'flex', justifyContent: 'center', py: 4 }}>
            <CircularProgress />
          </Box>
        )}

        {error && (
          <Alert severity="error" sx={{ mb: 2 }}>
            {error}
          </Alert>
        )}

        {!loading && !error && sql && (
          <>
            <Box
              sx={{
                display: 'flex',
                justifyContent: 'flex-end',
                mb: 1,
              }}
            >
              <IconButton
                onClick={handleCopy}
                size="small"
                title={copied ? 'Copied!' : 'Copy to clipboard'}
              >
                <ContentCopyIcon fontSize="small" />
              </IconButton>
              {copied && (
                <Typography
                  variant="caption"
                  sx={{ ml: 1, alignSelf: 'center', color: 'success.main' }}
                >
                  Copied!
                </Typography>
              )}
            </Box>
            <Box
              component="pre"
              sx={{
                maxHeight: 400,
                overflow: 'auto',
                fontFamily: 'monospace',
                fontSize: '0.875rem',
                backgroundColor: 'grey.900',
                color: 'grey.100',
                p: 2,
                borderRadius: 1,
                whiteSpace: 'pre-wrap',
                wordBreak: 'break-word',
              }}
            >
              {currentSQL}
            </Box>
          </>
        )}
      </DialogContent>
    </Dialog>
  );
};

export default PreAggSQLModal;
