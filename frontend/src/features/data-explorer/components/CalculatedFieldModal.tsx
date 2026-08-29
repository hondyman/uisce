import React, { useState } from 'react';
import {
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  Button,
  TextField,
  Typography,
  Stack,
  Box,
  MenuItem,
  Select,
  FormControl,
  InputLabel,
  Chip,
} from '@mui/material';
import { Calculate as CalcIcon } from '@mui/icons-material';
import type {
  ExplorerSource,
  CalculationDefinition,
} from '../types/dataExplorerTypes';
import {
  EXPLORER_ACCENT,
  EXPLORER_BG,
  EXPLORER_BORDER,
  EXPLORER_MUTED,
  EXPLORER_TEXT,
} from '../types/dataExplorerTypes';

interface CalculatedFieldModalProps {
  open: boolean;
  onClose: () => void;
  source: ExplorerSource;
  onSave: (calc: CalculationDefinition) => void;
}

export const CalculatedFieldModal: React.FC<CalculatedFieldModalProps> = ({
  open,
  onClose,
  source,
  onSave,
}) => {
  const [name, setName] = useState('');
  const [formula, setFormula] = useState('');
  const [returnType, setReturnType] = useState<'number' | 'string' | 'boolean' | 'date'>('number');
  const [error, setError] = useState<string | null>(null);

  const handleApplyTemplate = (tpl: string) => {
    setFormula(tpl);
  };

  const handleInsertField = (fieldName: string) => {
    setFormula((prev) => `${prev} ${fieldName}`.trim());
  };

  const handleSubmit = () => {
    if (!name.trim()) {
      setError('Field name is required.');
      return;
    }
    if (!formula.trim()) {
      setError('Formula expression is required.');
      return;
    }

    onSave({
      id: `calc_${Date.now()}`,
      name: name.trim().toLowerCase().replace(/\s+/g, '_'),
      displayName: name.trim(),
      formula: formula.trim(),
      returnType,
    });
    setName('');
    setFormula('');
    setError(null);
    onClose();
  };

  return (
    <Dialog open={open} onClose={onClose} maxWidth="sm" fullWidth>
      <DialogTitle sx={{ borderBottom: `1px solid ${EXPLORER_BORDER}`, px: 3, py: 2 }}>
        <Stack direction="row" spacing={1.5} alignItems="center">
          <CalcIcon sx={{ color: '#8b5cf6' }} />
          <Box>
            <Typography variant="subtitle1" fontWeight={700} sx={{ color: EXPLORER_TEXT }}>
              Create Calculated Field (Looker Expression)
            </Typography>
            <Typography variant="caption" sx={{ color: EXPLORER_MUTED }}>
              Write custom calculations, ratios, and formula expressions.
            </Typography>
          </Box>
        </Stack>
      </DialogTitle>

      <DialogContent sx={{ p: 3 }}>
        <Stack spacing={2.5} sx={{ mt: 1 }}>
          <TextField
            label="Field Display Name"
            size="small"
            fullWidth
            value={name}
            onChange={(e) => setName(e.target.value)}
            placeholder="e.g. Profit Margin %, Net Asset Ratio"
            autoFocus
          />

          <FormControl size="small" fullWidth>
            <InputLabel>Return Data Type</InputLabel>
            <Select
              value={returnType}
              label="Return Data Type"
              onChange={(e) => setReturnType(e.target.value as any)}
            >
              <MenuItem value="number">Numeric (Number / Currency / %)</MenuItem>
              <MenuItem value="string">Text (String)</MenuItem>
              <MenuItem value="boolean">Logical (Boolean)</MenuItem>
              <MenuItem value="date">Date / Timestamp</MenuItem>
            </Select>
          </FormControl>

          {/* Quick Formula Templates */}
          <Box>
            <Typography variant="caption" fontWeight={700} sx={{ color: EXPLORER_MUTED, mb: 1, display: 'block' }}>
              Formula Templates:
            </Typography>
            <Stack direction="row" spacing={1} flexWrap="wrap" useFlexGap>
              <Chip
                label="Ratio: A / B * 100"
                size="small"
                clickable
                onClick={() => handleApplyTemplate('SUM(amount) / SUM(cost) * 100')}
              />
              <Chip
                label="Margin: (A - B) / A"
                size="small"
                clickable
                onClick={() => handleApplyTemplate('(SUM(price) - SUM(cost)) / NULLIF(SUM(price), 0)')}
              />
              <Chip
                label="Running Total: SUM() OVER ()"
                size="small"
                clickable
                onClick={() => handleApplyTemplate('SUM(amount) OVER (ORDER BY open_date)')}
              />
            </Stack>
          </Box>

          {/* Field Helper Chips */}
          <Box>
            <Typography variant="caption" fontWeight={700} sx={{ color: EXPLORER_MUTED, mb: 1, display: 'block' }}>
              Insert Field Reference:
            </Typography>
            <Stack direction="row" spacing={0.8} flexWrap="wrap" useFlexGap sx={{ maxHeight: 100, overflowY: 'auto' }}>
              {source.fields.slice(0, 12).map((f) => (
                <Chip
                  key={f.id}
                  label={f.displayName || f.name}
                  size="small"
                  variant="outlined"
                  clickable
                  onClick={() => handleInsertField(f.technicalName || f.name)}
                />
              ))}
            </Stack>
          </Box>

          <TextField
            label="Calculation Formula (SQL / Looker Expression)"
            size="small"
            fullWidth
            multiline
            rows={3}
            value={formula}
            onChange={(e) => setFormula(e.target.value)}
            placeholder="e.g. SUM(total_valuation) * 0.02"
            error={Boolean(error)}
            helperText={error}
          />
        </Stack>
      </DialogContent>

      <DialogActions sx={{ px: 3, py: 2, borderTop: `1px solid ${EXPLORER_BORDER}` }}>
        <Button onClick={onClose} sx={{ textTransform: 'none', color: EXPLORER_MUTED }}>
          Cancel
        </Button>
        <Button variant="contained" onClick={handleSubmit} sx={{ textTransform: 'none', fontWeight: 700, borderRadius: 2 }}>
          Add Calculated Field
        </Button>
      </DialogActions>
    </Dialog>
  );
};

export default CalculatedFieldModal;
