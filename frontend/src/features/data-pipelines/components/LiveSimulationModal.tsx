import React, { useState } from 'react';
import {
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  Button,
  Box,
  Typography,
  TextField,
  Alert,
  CircularProgress,
  Chip,
  Tabs,
  Tab,
  useTheme,
} from '@mui/material';
import { Play, Sparkles, AlertCircle } from 'lucide-react';
import axios from '@/utils/axiosClient';
import { PipelineDefinition, PipelineExecutionRun } from '../types/pipeline';

interface LiveSimulationModalProps {
  open: boolean;
  onClose: () => void;
  pipeline: PipelineDefinition;
  onSimulationComplete: (run: PipelineExecutionRun) => void;
}

export const LiveSimulationModal: React.FC<LiveSimulationModalProps> = ({
  open,
  onClose,
  pipeline,
  onSimulationComplete,
}) => {
  const theme = useTheme();
  const isDark = theme.palette.mode === 'dark';

  const [sampleData, setSampleData] = useState<string>(
    JSON.stringify(
      [
        {
          ext_acc_num: 'ACC-INST-9021',
          inst_name: 'Vanguard Global Growth',
          order_ref: 'TRD-88912',
          sym: 'AAPL',
          qty: 5000,
          px: 228.5,
          side: 'BUY',
          stype: 'institutional',
          otype: 'block_parent',
        },
        {
          ext_acc_num: 'ACC-SMA-3341',
          inst_name: 'BlackRock Horizon Strategy',
          order_ref: 'TRD-88913',
          sym: 'NVDA',
          qty: 2500,
          px: 132.1,
          side: 'BUY',
          stype: 'sma',
          otype: 'dma_execution',
        },
      ],
      null,
      2
    )
  );

  const [simulating, setSimulating] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [runResult, setRunResult] = useState<PipelineExecutionRun | null>(null);
  const [activeTab, setActiveTab] = useState(0);

  const handleRunSimulation = async () => {
    try {
      setSimulating(true);
      setError(null);
      let parsedSample = [];
      try {
        parsedSample = JSON.parse(sampleData);
      } catch (err) {
        setError('Invalid sample JSON format');
        setSimulating(false);
        return;
      }

      const res = await axios.post(`/api/v1/data-pipelines/${pipeline.id || 'sim'}/simulate`, {
        pipeline: pipeline,
        sample_data: parsedSample,
      });

      setRunResult(res.data);
      onSimulationComplete(res.data);
      setActiveTab(1); // Switch to output tab
    } catch (err: any) {
      setError(err.response?.data?.error || err.message || 'Simulation failed');
    } finally {
      setSimulating(false);
    }
  };

  return (
    <Dialog
      open={open}
      onClose={onClose}
      maxWidth="md"
      fullWidth
      PaperProps={{
        sx: {
          backgroundColor: theme.palette.background.paper,
          backgroundImage: 'none',
          border: `1px solid ${theme.palette.divider}`,
        },
      }}
    >
      <DialogTitle sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', borderBottom: `1px solid ${theme.palette.divider}`, pb: 1.5 }}>
        <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
          <Sparkles size={20} color="#3b82f6" />
          <Typography variant="h6" sx={{ fontWeight: 800, fontSize: '1rem', color: theme.palette.text.primary }}>
            Dry-Run Pipeline Simulation
          </Typography>
        </Box>
        <Chip label="Zero-Side-Effect Mode" size="small" color="primary" sx={{ fontWeight: 700 }} />
      </DialogTitle>

      <DialogContent sx={{ p: 2.5 }}>
        <Typography variant="body2" sx={{ color: theme.palette.text.secondary, mb: 2 }}>
          Execute the visual pipeline in an isolated in-memory sandbox. Verifies STI subtype allowlists, column mappings, and catalog graph synthesizers with zero database side-effects.
        </Typography>

        <Tabs value={activeTab} onChange={(_, v) => setActiveTab(v)} sx={{ borderBottom: `1px solid ${theme.palette.divider}`, mb: 2 }}>
          <Tab label="Sample Input Payload" sx={{ fontWeight: 700, fontSize: '0.85rem' }} />
          <Tab label="Transformed Output & Metrics" disabled={!runResult} sx={{ fontWeight: 700, fontSize: '0.85rem' }} />
        </Tabs>

        {activeTab === 0 && (
          <Box>
            <TextField
              multiline
              rows={10}
              fullWidth
              value={sampleData}
              onChange={(e) => setSampleData(e.target.value)}
              sx={{
                fontFamily: 'monospace',
                fontSize: '0.75rem',
                '& .MuiInputBase-input': {
                  fontFamily: 'monospace',
                  backgroundColor: isDark ? 'rgba(0,0,0,0.25)' : '#f8fafc',
                },
              }}
            />
          </Box>
        )}

        {activeTab === 1 && runResult && (
          <Box>
            <Box sx={{ display: 'flex', gap: 2, mb: 2 }}>
              <Chip
                label={`Status: ${runResult.status.toUpperCase()}`}
                color={runResult.status === 'simulated' ? 'success' : 'error'}
                sx={{ fontWeight: 700 }}
              />
              <Chip
                label={`Throughput: ${Math.round(runResult.peak_throughput_rows_sec).toLocaleString()} rows/sec`}
                variant="outlined"
                sx={{ fontWeight: 700 }}
              />
              <Chip
                label={`Records Out: ${runResult.total_records_out}`}
                variant="outlined"
                sx={{ fontWeight: 700 }}
              />
            </Box>

            <Typography variant="subtitle2" sx={{ fontWeight: 700, mb: 0.5, color: theme.palette.text.primary }}>
              Sample Output Records:
            </Typography>
            <Box
              sx={{
                p: 2,
                backgroundColor: isDark ? 'rgba(0,0,0,0.3)' : '#f8fafc',
                borderRadius: '8px',
                border: `1px solid ${theme.palette.divider}`,
                maxHeight: 250,
                overflowY: 'auto',
              }}
            >
              <pre style={{ margin: 0, fontSize: '0.75rem', fontFamily: 'monospace', color: theme.palette.text.primary }}>
                {JSON.stringify(runResult.sample_output, null, 2)}
              </pre>
            </Box>
          </Box>
        )}

        {error && (
          <Alert severity="error" icon={<AlertCircle size={16} />} sx={{ mt: 2 }}>
            {error}
          </Alert>
        )}
      </DialogContent>

      <DialogActions sx={{ p: 2, borderTop: `1px solid ${theme.palette.divider}` }}>
        <Button onClick={onClose} variant="outlined">
          Close
        </Button>
        <Button
          variant="contained"
          color="primary"
          startIcon={simulating ? <CircularProgress size={16} color="inherit" /> : <Play size={16} />}
          onClick={handleRunSimulation}
          disabled={simulating}
        >
          {simulating ? 'Simulating Pipeline...' : 'Run Simulation'}
        </Button>
      </DialogActions>
    </Dialog>
  );
};
