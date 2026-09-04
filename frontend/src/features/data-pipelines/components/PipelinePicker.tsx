import React, { useState, useEffect } from 'react';
import {
  Box,
  FormControl,
  InputLabel,
  Select,
  MenuItem,
  Typography,
  CircularProgress,
  Alert,
  ToggleButton,
  ToggleButtonGroup,
  FormHelperText,
} from '@mui/material';
import axios from '@/utils/axiosClient';

interface CompactPipeline {
  id: string;
  name: string;
  source_table: string;
  sink_label: string;
}

interface PipelinePickerProps {
  value?: string;
  onChange: (pipelineId: string, dispatchMode: 'sync' | 'async') => void;
  error?: string;
}

export const PipelinePicker: React.FC<PipelinePickerProps> = ({
  value,
  onChange,
  error,
}) => {
  const [pipelines, setPipelines] = useState<CompactPipeline[]>([]);
  const [loading, setLoading] = useState(true);
  const [fetchError, setFetchError] = useState<string | null>(null);
  const [dispatchMode, setDispatchMode] = useState<'sync' | 'async'>('sync');

  useEffect(() => {
    axios
      .get('/api/v1/data-pipelines?compact=true')
      .then((res) => {
        setPipelines(res.data ?? []);
        setLoading(false);
      })
      .catch((err) => {
        setFetchError(err?.response?.data?.error ?? 'Failed to load pipelines');
        setLoading(false);
      });
  }, []);

  const handlePipelineChange = (id: string) => {
    onChange(id, dispatchMode);
  };

  const handleDispatchModeChange = (
    _: React.MouseEvent<HTMLElement>,
    mode: 'sync' | 'async' | null
  ) => {
    if (mode !== null) {
      setDispatchMode(mode);
      if (value) {
        onChange(value, mode);
      }
    }
  };

  return (
    <Box>
      {fetchError && (
        <Alert severity="error" sx={{ mb: 1 }}>
          {fetchError}
        </Alert>
      )}

      <FormControl fullWidth size="small" error={!!error} disabled={loading}>
        <InputLabel>Pipeline</InputLabel>
        <Select
          value={value ?? ''}
          label="Pipeline"
          onChange={(e) => handlePipelineChange(e.target.value)}
        >
          {loading && (
            <MenuItem value="">
              <CircularProgress size={16} sx={{ mr: 1 }} />
              Loading pipelines...
            </MenuItem>
          )}
          {!loading && pipelines.length === 0 && (
            <MenuItem value="" disabled>
              No pipelines available
            </MenuItem>
          )}
          {pipelines.map((p) => (
            <MenuItem key={p.id} value={p.id}>
              <Box>
                <Typography variant="body2">{p.name}</Typography>
                <Typography variant="caption" color="text.secondary">
                  {p.source_table || 'no source'} → {p.sink_label || 'no sink'}
                </Typography>
              </Box>
            </MenuItem>
          ))}
        </Select>
        {error && <FormHelperText>{error}</FormHelperText>}
      </FormControl>

      {value && (
        <Box sx={{ mt: 1.5 }}>
          <Typography variant="caption" color="text.secondary" sx={{ mb: 0.5, display: 'block' }}>
            Dispatch mode
          </Typography>
          <ToggleButtonGroup
            value={dispatchMode}
            exclusive
            onChange={handleDispatchModeChange}
            size="small"
          >
            <ToggleButton value="sync">Sync</ToggleButton>
            <ToggleButton value="async">Async</ToggleButton>
          </ToggleButtonGroup>
          <FormHelperText sx={{ mt: 0.5 }}>
            {dispatchMode === 'sync'
              ? 'Pipeline runs inline; a failure blocks the BO write'
              : 'Pipeline queued via outbox; write is not blocked'}
          </FormHelperText>
        </Box>
      )}
    </Box>
  );
};
