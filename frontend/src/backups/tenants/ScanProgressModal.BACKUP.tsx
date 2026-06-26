import React, { useEffect, useState } from 'react';
import { 
  Dialog, DialogTitle, DialogContent, DialogActions, Button, 
  CircularProgress, Typography, Box, Alert, Chip, 
  List, ListItem, ListItemText, ListItemIcon,
  LinearProgress
} from '@mui/material';
import CheckCircleIcon from '@mui/icons-material/CheckCircle';
import ErrorIcon from '@mui/icons-material/Error';
import AddCircleIcon from '@mui/icons-material/AddCircle';
import RemoveCircleIcon from '@mui/icons-material/RemoveCircle';
import UpdateIcon from '@mui/icons-material/Update';

export interface ScanResultDetail {
  tenant_instance_id: string;
  name: string;
  success: boolean;
  error?: string;
  added?: number;
  updated?: number;
  removed?: number;
}

export interface ScanResult {
  status: string;
  message: string;
  results: ScanResultDetail[];
}

interface ScanProgress {
  phase: string;
  percent: number;
  current_item: string;
  total: number;
  completed: number;
  message: string;
}

interface ScanProgressModalProps {
  open: boolean;
  onClose: () => void;
  loading: boolean;
  result: ScanResult | null;
  error?: Error;
  datasourceId?: string; // Required for SSE streaming
  useStreaming?: boolean; // Enable SSE streaming mode
}

export default function ScanProgressModal({ 
  open, 
  onClose, 
  loading, 
  result, 
  error, 
  datasourceId,
  useStreaming = true 
}: ScanProgressModalProps) {
  const [progress, setProgress] = useState<ScanProgress | null>(null);
  const [streamError, setStreamError] = useState<string | null>(null);
  const [isStreaming, setIsStreaming] = useState(false);

  useEffect(() => {
    if (!open || !datasourceId || !useStreaming) return;

    console.log('[SSE] Opening connection for datasource:', datasourceId);
    setProgress(null);
    setStreamError(null);
    setIsStreaming(true);

    // Use backend URL directly for SSE to bypass potential proxy issues
    const backendUrl = import.meta.env.VITE_BACKEND_URL || 'http://localhost:8080';
    const sseUrl = `${backendUrl}/api/catalog/scan/stream?datasource_id=${datasourceId}`;
    console.log('[SSE] Connecting to:', sseUrl);
    
    const eventSource = new EventSource(sseUrl);

    eventSource.onopen = () => {
      console.log('[SSE] Connection opened');
    };

    eventSource.onmessage = (event) => {
      console.log('[SSE] Message received:', event.data);
      try {
        const data = JSON.parse(event.data) as ScanProgress;
        setProgress(data);

        if (data.phase === 'complete' || data.phase === 'error') {
          console.log('[SSE] Scan complete, closing connection');
          eventSource.close();
          setIsStreaming(false);
        }
      } catch (e) {
        console.error('[SSE] Failed to parse message:', e);
      }
    };

    eventSource.onerror = (e) => {
      console.error('[SSE] Connection error:', e);
      setStreamError('Connection to scan service lost');
      eventSource.close();
      setIsStreaming(false);
    };

    return () => {
      console.log('[SSE] Cleaning up connection');
      eventSource.close();
      setIsStreaming(false);
    };
  }, [open, datasourceId, useStreaming]);

  const getPhaseLabel = (phase: string): string => {
    switch (phase) {
      case 'starting': return 'Initializing...';
      case 'fetching': return 'Fetching configuration...';
      case 'preparing': return 'Loading mappings...';
      case 'connecting': return 'Connecting to database...';
      case 'scanning': return 'Scanning metadata...';
      case 'storing': return 'Storing results...';
      case 'complete': return 'Complete!';
      case 'error': return 'Error';
      default: return phase;
    }
  };

  return (
    <Dialog open={open} onClose={loading || isStreaming ? undefined : onClose} maxWidth="md" fullWidth>
      <DialogTitle>
        Datasource Scan
        {(loading || isStreaming) && <Typography variant="caption" sx={{ ml: 2 }}>Scanning in progress...</Typography>}
      </DialogTitle>
      <DialogContent>
        {/* Waiting for SSE connection */}
        {useStreaming && isStreaming && !progress && (
          <Box sx={{ display: 'flex', flexDirection: 'column', alignItems: 'center', py: 4 }}>
            <CircularProgress size={48} sx={{ mb: 2 }} />
            <Typography>Connecting to scan service...</Typography>
            <Typography variant="body2" color="text.secondary">Establishing real-time progress stream</Typography>
          </Box>
        )}

        {/* Streaming Progress Mode */}
        {useStreaming && isStreaming && progress && (
          <Box sx={{ py: 3 }}>
            <Box sx={{ display: 'flex', alignItems: 'center', mb: 2 }}>
              <Box sx={{ width: '100%', mr: 1 }}>
                <LinearProgress 
                  variant="determinate" 
                  value={progress.percent} 
                  sx={{ height: 10, borderRadius: 5 }}
                />
              </Box>
              <Box sx={{ minWidth: 50 }}>
                <Typography variant="body2" color="text.secondary">
                  {Math.round(progress.percent)}%
                </Typography>
              </Box>
            </Box>
            
            <Typography variant="h6" sx={{ mb: 1 }}>
              {getPhaseLabel(progress.phase)}
            </Typography>
            
            <Typography variant="body1" color="text.secondary" sx={{ mb: 1 }}>
              {progress.message}
            </Typography>
            
            {progress.current_item && (
              <Typography variant="body2" color="primary.main">
                Current: {progress.current_item}
              </Typography>
            )}
            
            {progress.total > 0 && (
              <Typography variant="caption" color="text.secondary">
                {progress.completed} of {progress.total} items
              </Typography>
            )}
          </Box>
        )}

        {/* Legacy Loading Mode */}
        {!useStreaming && loading && (
          <Box sx={{ display: 'flex', flexDirection: 'column', alignItems: 'center', py: 4 }}>
            <CircularProgress size={48} sx={{ mb: 2 }} />
            <Typography>Scanning datasource metadata...</Typography>
            <Typography variant="body2" color="text.secondary">This may take a minute depending on the database size.</Typography>
          </Box>
        )}

        {/* Stream Error */}
        {streamError && (
          <Alert severity="error" sx={{ mt: 2 }}>
            {streamError}
          </Alert>
        )}

        {/* Regular Error */}
        {error && (
          <Alert severity="error" sx={{ mt: 2 }}>
            Scan failed: {error.message}
          </Alert>
        )}

        {/* Results */}
        {!loading && !isStreaming && result && (
          <Box sx={{ mt: 2 }}>
            <Alert severity={result.status === 'success' ? 'success' : result.status === 'partial' ? 'warning' : 'error'} sx={{ mb: 3 }}>
              {result.message}
            </Alert>

            <Typography variant="subtitle1" gutterBottom>Scan Details:</Typography>
            <List>
              {result.results.map((res) => (
                <React.Fragment key={res.tenant_instance_id}>
                  <ListItem alignItems="flex-start" sx={{ bgcolor: 'background.paper', mb: 1, borderRadius: 1, border: 1, borderColor: 'divider' }}>
                    <ListItemIcon sx={{ minWidth: 40, mt: 1 }}>
                      {res.success ? <CheckCircleIcon color="success" /> : <ErrorIcon color="error" />}
                    </ListItemIcon>
                    <ListItemText
                      primary={
                        <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                          <Typography variant="subtitle2">{res.name}</Typography>
                          {!res.success && <Chip label="Failed" color="error" size="small" />}
                        </Box>
                      }
                      secondary={
                        <Box sx={{ mt: 1 }}>
                          {res.error ? (
                            <Typography variant="body2" color="error">{res.error}</Typography>
                          ) : (
                            <Box sx={{ display: 'flex', gap: 2 }}>
                              <Box sx={{ display: 'flex', alignItems: 'center', gap: 0.5 }}>
                                <AddCircleIcon color="success" fontSize="small" />
                                <Typography variant="body2">+{res.added || 0} Added</Typography>
                              </Box>
                              <Box sx={{ display: 'flex', alignItems: 'center', gap: 0.5 }}>
                                <UpdateIcon color="warning" fontSize="small" />
                                <Typography variant="body2">~{res.updated || 0} Updated</Typography>
                              </Box>
                              <Box sx={{ display: 'flex', alignItems: 'center', gap: 0.5 }}>
                                <RemoveCircleIcon color="error" fontSize="small" />
                                <Typography variant="body2">-{res.removed || 0} Removed</Typography>
                              </Box>
                            </Box>
                          )}
                        </Box>
                      }
                    />
                  </ListItem>
                </React.Fragment>
              ))}
            </List>
          </Box>
        )}

        {/* Streaming Complete State */}
        {useStreaming && !isStreaming && progress?.phase === 'complete' && !result && (
          <Alert severity="success" sx={{ mt: 2 }}>
            {progress.message}
          </Alert>
        )}
      </DialogContent>
      <DialogActions>
        <Button onClick={onClose} disabled={loading || isStreaming}>
          Close
        </Button>
      </DialogActions>
    </Dialog>
  );
}
