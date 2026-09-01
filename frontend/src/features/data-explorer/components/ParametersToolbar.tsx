import React, { useState } from 'react';
import {
  Box,
  Paper,
  Typography,
  Stack,
  Chip,
  IconButton,
  Button,
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  TextField,
  FormControl,
  InputLabel,
  Select,
  MenuItem,
  Tooltip,
} from '@mui/material';
import {
  Tune as ParameterIcon,
  Add as AddIcon,
  Edit as EditIcon,
  Delete as DeleteIcon,
} from '@mui/icons-material';
import type { QueryParameter } from '../types/dataExplorerTypes';
import { useExplorerTheme } from '../hooks/useExplorerTheme';

interface ParametersToolbarProps {
  parameters: QueryParameter[];
  onAddParameter: (param: QueryParameter) => void;
  onUpdateParameter: (param: QueryParameter) => void;
  onRemoveParameter: (paramId: string) => void;
  onChangeParamValue: (paramId: string, value: any) => void;
}

export const ParametersToolbar: React.FC<ParametersToolbarProps> = ({
  parameters,
  onAddParameter,
  onUpdateParameter,
  onRemoveParameter,
  onChangeParamValue,
}) => {
  const theme = useExplorerTheme();
  const [modalOpen, setModalOpen] = useState(false);
  const [editingParam, setEditingParam] = useState<QueryParameter | null>(null);
  const [paramName, setParamName] = useState('');
  const [paramDisplayName, setParamDisplayName] = useState('');
  const [paramType, setParamType] = useState<QueryParameter['type']>('string');
  const [paramDefault, setParamDefault] = useState('');
  const [paramDescription, setParamDescription] = useState('');

  const handleOpenAdd = () => {
    setEditingParam(null);
    setParamName('');
    setParamDisplayName('');
    setParamType('string');
    setParamDefault('');
    setParamDescription('');
    setModalOpen(true);
  };

  const handleOpenEdit = (p: QueryParameter) => {
    setEditingParam(p);
    setParamName(p.name);
    setParamDisplayName(p.displayName);
    setParamType(p.type);
    setParamDefault(p.defaultValue ?? '');
    setParamDescription(p.description || '');
    setModalOpen(true);
  };

  const handleSave = () => {
    if (!paramDisplayName.trim()) return;
    const cleanName =
      paramName.trim() ||
      paramDisplayName
        .toLowerCase()
        .replace(/[^a-z0-9]+/g, '_')
        .replace(/^_+|_+$/g, '');

    const saved: QueryParameter = {
      id: editingParam?.id || `param_${Date.now().toString(36)}`,
      name: cleanName,
      displayName: paramDisplayName.trim(),
      type: paramType,
      defaultValue: paramDefault,
      currentValue: editingParam ? editingParam.currentValue ?? paramDefault : paramDefault,
      description: paramDescription.trim(),
    };

    if (editingParam) {
      onUpdateParameter(saved);
    } else {
      onAddParameter(saved);
    }
    setModalOpen(false);
  };

  return (
    <Box
      sx={{
        px: 2,
        py: 0.8,
        bgcolor: theme.background,
        borderBottom: `1px solid ${theme.border}`,
        display: 'flex',
        alignItems: 'center',
        gap: 1.5,
        overflowX: 'auto',
      }}
    >
      <Stack direction="row" alignItems="center" spacing={0.5} sx={{ color: theme.textMuted, flexShrink: 0 }}>
        <ParameterIcon sx={{ fontSize: 16, color: theme.accent }} />
        <Typography variant="caption" sx={{ fontWeight: 700, textTransform: 'uppercase', letterSpacing: 0.5 }}>
          Parameters
        </Typography>
      </Stack>

      {parameters.length === 0 && (
        <Typography variant="caption" sx={{ color: theme.textMuted, fontStyle: 'italic' }}>
          No runtime query parameters configured.
        </Typography>
      )}

      <Stack direction="row" spacing={1} alignItems="center" sx={{ flex: 1, overflowX: 'auto' }}>
        {parameters.map((p) => (
          <Paper
            key={p.id}
            elevation={0}
            sx={{
              display: 'flex',
              alignItems: 'center',
              gap: 1,
              px: 1.2,
              py: 0.3,
              bgcolor: theme.backgroundElevated,
              border: `1px solid ${theme.border}`,
              borderRadius: 1.5,
            }}
          >
            <Typography variant="caption" sx={{ fontWeight: 700, color: theme.text }}>
              @{p.displayName}:
            </Typography>
            <TextField
              size="small"
              variant="standard"
              placeholder={p.defaultValue || 'value'}
              value={p.currentValue ?? ''}
              onChange={(e) => onChangeParamValue(p.id, e.target.value)}
              InputProps={{
                disableUnderline: true,
                sx: { fontSize: '0.75rem', fontWeight: 600, width: 90, color: theme.text },
              }}
            />
            <Tooltip title="Edit parameter">
              <IconButton size="small" onClick={() => handleOpenEdit(p)} sx={{ p: 0.2 }}>
                <EditIcon sx={{ fontSize: 13, color: theme.textMuted }} />
              </IconButton>
            </Tooltip>
            <Tooltip title="Remove parameter">
              <IconButton size="small" onClick={() => onRemoveParameter(p.id)} sx={{ p: 0.2 }}>
                <DeleteIcon sx={{ fontSize: 13, color: theme.error }} />
              </IconButton>
            </Tooltip>
          </Paper>
        ))}
      </Stack>

      <Button
        size="small"
        startIcon={<AddIcon sx={{ fontSize: 14 }} />}
        onClick={handleOpenAdd}
        sx={{
          fontSize: '0.72rem',
          textTransform: 'none',
          color: theme.accent,
          fontWeight: 700,
          flexShrink: 0,
        }}
      >
        Add Parameter
      </Button>

      <Dialog open={modalOpen} onClose={() => setModalOpen(false)} maxWidth="xs" fullWidth>
        <DialogTitle sx={{ fontWeight: 700, fontSize: '1rem', color: theme.text }}>
          {editingParam ? 'Edit Query Parameter' : 'Add Query Parameter'}
        </DialogTitle>
        <DialogContent dividers sx={{ display: 'flex', flexDirection: 'column', gap: 2 }}>
          <TextField
            label="Parameter Label"
            placeholder="e.g. Start Date, Minimum AUM"
            value={paramDisplayName}
            onChange={(e) => setParamDisplayName(e.target.value)}
            fullWidth
            size="small"
            required
          />
          <FormControl size="small" fullWidth>
            <InputLabel>Parameter Type</InputLabel>
            <Select
              value={paramType}
              label="Parameter Type"
              onChange={(e) => setParamType(e.target.value as QueryParameter['type'])}
            >
              <MenuItem value="string">Text / String</MenuItem>
              <MenuItem value="number">Number</MenuItem>
              <MenuItem value="date">Date</MenuItem>
              <MenuItem value="daterange">Date Range</MenuItem>
            </Select>
          </FormControl>
          <TextField
            label="Default Value"
            placeholder="Optional fallback value"
            value={paramDefault}
            onChange={(e) => setParamDefault(e.target.value)}
            fullWidth
            size="small"
          />
          <TextField
            label="Description (Optional)"
            placeholder="Explain usage in queries"
            value={paramDescription}
            onChange={(e) => setParamDescription(e.target.value)}
            fullWidth
            size="small"
          />
        </DialogContent>
        <DialogActions sx={{ px: 3, py: 1.5 }}>
          <Button onClick={() => setModalOpen(false)} sx={{ textTransform: 'none', color: theme.textMuted }}>
            Cancel
          </Button>
          <Button
            onClick={handleSave}
            variant="contained"
            sx={{
              bgcolor: theme.accent,
              color: '#FFF',
              textTransform: 'none',
              fontWeight: 700,
              '&:hover': { bgcolor: theme.accentDark },
            }}
          >
            Save Parameter
          </Button>
        </DialogActions>
      </Dialog>
    </Box>
  );
};
