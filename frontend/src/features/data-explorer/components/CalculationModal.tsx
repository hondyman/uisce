import React, { useState, useEffect } from 'react';
import {
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  Button,
  TextField,
  FormControl,
  InputLabel,
  Select,
  MenuItem,
  Stack,
  Typography,
  Box,
  Chip,
  Alert,
} from '@mui/material';
import { Functions as FunctionsIcon } from '@mui/icons-material';
import type {
  CalculationDefinition,
  ExplorerField,
  FieldType,
} from '../types/dataExplorerTypes';
import { EXPLORER_ACCENT, EXPLORER_BORDER, EXPLORER_TEXT } from '../types/dataExplorerTypes';

interface CalculationModalProps {
  open: boolean;
  onClose: () => void;
  availableFields: ExplorerField[];
  initialCalculation?: CalculationDefinition | null;
  onSave: (calculation: CalculationDefinition) => void;
}

export const CalculationModal: React.FC<CalculationModalProps> = ({
  open,
  onClose,
  availableFields,
  initialCalculation,
  onSave,
}) => {
  const [name, setName] = useState('');
  const [displayName, setDisplayName] = useState('');
  const [formula, setFormula] = useState('');
  const [returnType, setReturnType] = useState<FieldType>('number');
  const [format, setFormat] = useState<'currency' | 'percent' | 'number'>('number');
  const [description, setDescription] = useState('');
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (open) {
      if (initialCalculation) {
        setName(initialCalculation.name);
        setDisplayName(initialCalculation.displayName);
        setFormula(initialCalculation.formula);
        setReturnType(initialCalculation.returnType);
        setFormat(initialCalculation.format || 'number');
        setDescription(initialCalculation.description || '');
      } else {
        setName('');
        setDisplayName('');
        setFormula('');
        setReturnType('number');
        setFormat('number');
        setDescription('');
      }
      setError(null);
    }
  }, [open, initialCalculation]);

  const handleAddFieldToFormula = (fieldName: string) => {
    setFormula((prev) => `${prev}[${fieldName}]`);
  };

  const handleAddOperator = (op: string) => {
    setFormula((prev) => `${prev} ${op} `);
  };

  const handleSave = () => {
    if (!displayName.trim()) {
      setError('Calculation display name is required.');
      return;
    }
    if (!formula.trim()) {
      setError('Formula expression cannot be empty.');
      return;
    }

    const cleanName =
      name.trim() ||
      displayName
        .toLowerCase()
        .replace(/[^a-z0-9]+/g, '_')
        .replace(/^_+|_+$/g, '');

    onSave({
      id: initialCalculation?.id || `calc_${Date.now().toString(36)}`,
      name: cleanName,
      displayName: displayName.trim(),
      formula: formula.trim(),
      returnType,
      format,
      description: description.trim(),
    });
    onClose();
  };

  return (
    <Dialog open={open} onClose={onClose} maxWidth="sm" fullWidth>
      <DialogTitle sx={{ display: 'flex', alignItems: 'center', gap: 1, fontWeight: 700 }}>
        <FunctionsIcon sx={{ color: '#0D9488' }} />
        {initialCalculation ? 'Edit Calculation' : 'New Calculated Field'}
      </DialogTitle>

      <DialogContent dividers sx={{ display: 'flex', flexDirection: 'column', gap: 2.5 }}>
        {error && <Alert severity="error">{error}</Alert>}

        <TextField
          label="Display Name"
          placeholder="e.g. Operating Margin %, Average Trade Value"
          value={displayName}
          onChange={(e) => setDisplayName(e.target.value)}
          fullWidth
          size="small"
          required
        />

        <Stack direction="row" spacing={2}>
          <FormControl size="small" fullWidth>
            <InputLabel>Return Type</InputLabel>
            <Select
              value={returnType}
              label="Return Type"
              onChange={(e) => setReturnType(e.target.value as FieldType)}
            >
              <MenuItem value="number">Number</MenuItem>
              <MenuItem value="string">String / Text</MenuItem>
              <MenuItem value="date">Date</MenuItem>
              <MenuItem value="boolean">Boolean</MenuItem>
            </Select>
          </FormControl>

          {returnType === 'number' && (
            <FormControl size="small" fullWidth>
              <InputLabel>Format</InputLabel>
              <Select
                value={format}
                label="Format"
                onChange={(e) => setFormat(e.target.value as 'currency' | 'percent' | 'number')}
              >
                <MenuItem value="number">Standard Number</MenuItem>
                <MenuItem value="currency">Currency ($)</MenuItem>
                <MenuItem value="percent">Percentage (%)</MenuItem>
              </Select>
            </FormControl>
          )}
        </Stack>

        <Box>
          <Typography variant="caption" sx={{ fontWeight: 700, color: '#64748B', mb: 1, display: 'block' }}>
            INSERT FIELD
          </Typography>
          <Box sx={{ display: 'flex', flexWrap: 'wrap', gap: 0.8, maxHeight: 110, overflowY: 'auto', p: 1, bgcolor: '#F8FAFC', borderRadius: 1.5, border: `1px solid ${EXPLORER_BORDER}` }}>
            {availableFields.map((f) => (
              <Chip
                key={f.id}
                label={f.displayName}
                size="small"
                onClick={() => handleAddFieldToFormula(f.name)}
                sx={{
                  bgcolor: '#FFF',
                  border: `1px solid ${EXPLORER_BORDER}`,
                  fontWeight: 600,
                  fontSize: '0.72rem',
                  cursor: 'pointer',
                  '&:hover': { bgcolor: EXPLORER_ACCENT, color: EXPLORER_TEXT },
                }}
              />
            ))}
          </Box>
        </Box>

        <Box>
          <Typography variant="caption" sx={{ fontWeight: 700, color: '#64748B', mb: 1, display: 'block' }}>
            OPERATORS & FUNCTIONS
          </Typography>
          <Stack direction="row" spacing={1} flexWrap="wrap" useFlexGap>
            {['+', '-', '*', '/', '(', ')', 'SUM()', 'AVG()', 'COUNT()', 'MIN()', 'MAX()'].map((op) => (
              <Button
                key={op}
                size="small"
                variant="outlined"
                onClick={() => (op.endsWith('()') ? setFormula((prev) => `${prev}${op.slice(0, -1)}`) : handleAddOperator(op))}
                sx={{
                  minWidth: 40,
                  fontSize: '0.75rem',
                  fontWeight: 700,
                  textTransform: 'none',
                  borderColor: EXPLORER_BORDER,
                }}
              >
                {op}
              </Button>
            ))}
          </Stack>
        </Box>

        <TextField
          label="Formula Expression"
          placeholder="e.g. SUM([revenue]) / SUM([volume]) * 100"
          value={formula}
          onChange={(e) => setFormula(e.target.value)}
          multiline
          rows={3}
          fullWidth
          required
          helperText="Refer to columns using [column_name] format"
        />

        <TextField
          label="Description (Optional)"
          placeholder="Explanation of financial calculation or metric derivation"
          value={description}
          onChange={(e) => setDescription(e.target.value)}
          fullWidth
          size="small"
        />
      </DialogContent>

      <DialogActions sx={{ px: 3, py: 1.5 }}>
        <Button onClick={onClose} sx={{ textTransform: 'none' }}>
          Cancel
        </Button>
        <Button
          onClick={handleSave}
          variant="contained"
          sx={{
            bgcolor: '#0D9488',
            color: '#FFF',
            textTransform: 'none',
            fontWeight: 700,
            '&:hover': { bgcolor: '#0F766E' },
          }}
        >
          Save Calculation
        </Button>
      </DialogActions>
    </Dialog>
  );
};
