import React from 'react';
import {
  Box,
  Button,
  FormControl,
  IconButton,
  InputLabel,
  MenuItem,
  Paper,
  Select,
  Stack,
  TextField,
  Typography,
  Chip,
  Autocomplete,
  Alert,
} from '@mui/material';
import DeleteOutlineIcon from '@mui/icons-material/DeleteOutline';
import AddIcon from '@mui/icons-material/Add';
import type { DrillThroughTarget } from '../utils/drilldown';
import type { PageVariable } from '../../../pages/page-studio/app/appModel';

interface Props {
  targets: DrillThroughTarget[];
  onChange: (targets: DrillThroughTarget[]) => void;
  outputColumns?: string[];
  pageVariables?: PageVariable[];
}

export function validateDrillTarget(target: DrillThroughTarget): { valid: boolean; message?: string } {
  if (!target.label || !target.label.trim()) {
    return { valid: false, message: 'Action label is required' };
  }
  if (!target.target || !target.target.trim()) {
    return { valid: false, message: 'Target route or key is required' };
  }
  return { valid: true };
}

export default function DrillThroughTargetsEditor({
  targets = [],
  onChange,
  outputColumns = [],
  pageVariables = [],
}: Props) {
  const tokenSuggestions = [
    '{{selection.recordId}}',
    ...outputColumns.map((col) => `{{row.${col}}}`),
    ...pageVariables.map((v) => `{{vars.${v.name}}}`),
  ];

  const handleAdd = () => {
    const newTarget: DrillThroughTarget = {
      type: 'page',
      target: '',
      label: `Action ${targets.length + 1}`,
      contextMapping: {},
    };
    onChange([...targets, newTarget]);
  };

  const handleRemove = (index: number) => {
    onChange(targets.filter((_, i) => i !== index));
  };

  const handleUpdate = (index: number, patch: Partial<DrillThroughTarget>) => {
    onChange(targets.map((t, i) => (i === index ? { ...t, ...patch } : t)));
  };

  const handleAddMapping = (targetIndex: number) => {
    const t = targets[targetIndex];
    const key = `param_${Object.keys(t.contextMapping || {}).length + 1}`;
    handleUpdate(targetIndex, {
      contextMapping: {
        ...(t.contextMapping || {}),
        [key]: outputColumns[0] ? `{{row.${outputColumns[0]}}}` : '{{selection.recordId}}',
      },
    });
  };

  const handleUpdateMapping = (
    targetIndex: number,
    oldKey: string,
    newKey: string,
    val: string
  ) => {
    const t = targets[targetIndex];
    const nextMapping: Record<string, string> = {};
    for (const [k, v] of Object.entries(t.contextMapping || {})) {
      if (k === oldKey) {
        if (newKey) nextMapping[newKey] = val;
      } else {
        nextMapping[k] = v;
      }
    }
    if (!oldKey && newKey) {
      nextMapping[newKey] = val;
    }
    handleUpdate(targetIndex, { contextMapping: nextMapping });
  };

  const handleRemoveMapping = (targetIndex: number, key: string) => {
    const t = targets[targetIndex];
    const nextMapping = { ...(t.contextMapping || {}) };
    delete nextMapping[key];
    handleUpdate(targetIndex, { contextMapping: nextMapping });
  };

  return (
    <Box sx={{ mt: 1 }}>
      <Stack direction="row" justifyContent="space-between" alignItems="center" sx={{ mb: 1 }}>
        <Typography variant="subtitle2" fontWeight={700}>
          Drill-Through Actions ({targets.length})
        </Typography>
        <Button size="small" variant="outlined" startIcon={<AddIcon />} onClick={handleAdd}>
          Add Action
        </Button>
      </Stack>

      {targets.length === 0 ? (
        <Typography variant="caption" color="text.secondary">
          No drill-through targets configured. Elements will perform hierarchical drill-down if configured.
        </Typography>
      ) : (
        <Stack spacing={1.5}>
          {targets.map((target, idx) => {
            const validation = validateDrillTarget(target);
            return (
              <Paper key={idx} variant="outlined" sx={{ p: 1.5 }}>
                <Stack direction="row" spacing={1.5} alignItems="center" sx={{ mb: 1.5 }}>
                  <FormControl size="small" sx={{ width: 130 }}>
                    <InputLabel>Type</InputLabel>
                    <Select
                      value={target.type}
                      label="Type"
                      onChange={(e) => handleUpdate(idx, { type: e.target.value as DrillThroughTarget['type'] })}
                    >
                      <MenuItem value="page">Page</MenuItem>
                      <MenuItem value="modal">Modal</MenuItem>
                      <MenuItem value="query">Query</MenuItem>
                    </Select>
                  </FormControl>

                  <TextField
                    size="small"
                    label="Button / Action Label"
                    value={target.label}
                    error={!target.label.trim()}
                    onChange={(e) => handleUpdate(idx, { label: e.target.value })}
                    sx={{ width: 200 }}
                  />

                  <TextField
                    size="small"
                    label="Target Route / Key"
                    placeholder="/pages/account-detail or AccountModal"
                    value={target.target}
                    error={!target.target.trim()}
                    onChange={(e) => handleUpdate(idx, { target: e.target.value })}
                    fullWidth
                  />

                  <IconButton size="small" color="error" onClick={() => handleRemove(idx)}>
                    <DeleteOutlineIcon fontSize="small" />
                  </IconButton>
                </Stack>

                {!validation.valid && (
                  <Alert severity="error" sx={{ py: 0, px: 1, mb: 1, fontSize: '0.75rem' }}>
                    {validation.message}
                  </Alert>
                )}

                {/* Context Mappings */}
                <Box sx={{ pl: 1, borderLeft: 2, borderColor: 'primary.light', my: 1 }}>
                  <Stack direction="row" justifyContent="space-between" alignItems="center" sx={{ mb: 1 }}>
                    <Typography variant="caption" fontWeight={600} color="text.secondary">
                      Context Mappings (Parameters passed to target)
                    </Typography>
                    <Button size="small" onClick={() => handleAddMapping(idx)} sx={{ fontSize: '0.75rem', py: 0 }}>
                      + Add Field
                    </Button>
                  </Stack>

                  {Object.entries(target.contextMapping || {}).map(([paramKey, tokenVal]) => (
                    <Stack key={paramKey} direction="row" spacing={1} alignItems="center" sx={{ mb: 1 }}>
                      <TextField
                        size="small"
                        label="Target Param"
                        value={paramKey}
                        onChange={(e) => handleUpdateMapping(idx, paramKey, e.target.value, tokenVal)}
                        sx={{ width: 160 }}
                      />
                      <Autocomplete
                        freeSolo
                        size="small"
                        options={tokenSuggestions}
                        value={tokenVal}
                        onChange={(_, v) => handleUpdateMapping(idx, paramKey, paramKey, v || '')}
                        onInputChange={(_, v) => handleUpdateMapping(idx, paramKey, paramKey, v || '')}
                        sx={{ flex: 1 }}
                        renderInput={(params) => (
                          <TextField {...params} label="Value / Token Source" />
                        )}
                      />
                      <IconButton size="small" onClick={() => handleRemoveMapping(idx, paramKey)}>
                        <DeleteOutlineIcon fontSize="small" />
                      </IconButton>
                    </Stack>
                  ))}
                </Box>
              </Paper>
            );
          })}
        </Stack>
      )}
    </Box>
  );
}
