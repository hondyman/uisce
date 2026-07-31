import React, { useState } from 'react';
import {
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  Button,
  TextField,
  Typography,
  Box,
  Paper,
  Alert,
  Autocomplete,
  Chip
} from '@mui/material';
import CalculateIcon from '@mui/icons-material/Calculate';
import CheckCircleIcon from '@mui/icons-material/CheckCircle';

interface ExpressionBuilderModalProps {
  open: boolean;
  onClose: () => void;
  boName?: string;
  onSaveExpression: (formula: string, compiledSql: string) => void;
}

const availableFields = [
  'Account.market_value',
  'Account.total_cost',
  'Account.balance',
  'Customer.credit_limit',
  'Customer.total_orders',
];

export const ExpressionBuilderModal: React.FC<ExpressionBuilderModalProps> = ({
  open,
  onClose,
  boName = 'Account',
  onSaveExpression,
}) => {
  const [formula, setFormula] = useState('IF([Account.market_value] > 1000000, [Account.market_value] * 0.01, 0)');
  const [loading, setLoading] = useState(false);
  const [validationResult, setValidationResult] = useState<{
    valid: boolean;
    compiledSql?: string;
    extractedFields?: string[];
    error?: string;
  } | null>(null);

  const handleTestExpression = async () => {
    setLoading(true);
    setValidationResult(null);

    try {
      const res = await fetch('/api/calculation/compile', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ formula, boName }),
      });
      const data = await res.json();
      setValidationResult(data);
    } catch (err: any) {
      setValidationResult({ valid: false, error: err.message || 'Failed to compile formula' });
    } finally {
      setLoading(false);
    }
  };

  const handleInsertToken = (token: string) => {
    setFormula((prev) => `${prev} [${token}]`);
  };

  return (
    <Dialog open={open} onClose={onClose} maxWidth="md" fullWidth>
      <DialogTitle sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
        <CalculateIcon sx={{ color: '#38bdf8' }} />
        Calculated Metric & Formula Builder
      </DialogTitle>
      <DialogContent>
        <Typography variant="body2" color="textSecondary" mb={2}>
          Write Excel-style formulas using fields enclosed in <code>[BusinessObject.field]</code> format.
        </Typography>

        <Box mb={2} display="flex" gap={1} flexWrap="wrap">
          <Typography variant="caption" sx={{ alignSelf: 'center', color: '#94a3b8' }}>
            Quick Token Insert:
          </Typography>
          {availableFields.map((field) => (
            <Chip
              key={field}
              label={`[${field}]`}
              size="small"
              onClick={() => handleInsertToken(field)}
              sx={{ bgcolor: 'rgba(56, 189, 248, 0.1)', color: '#38bdf8', cursor: 'pointer' }}
            />
          ))}
        </Box>

        <TextField
          label="Formula Expression"
          value={formula}
          onChange={(e) => setFormula(e.target.value)}
          multiline
          rows={3}
          fullWidth
          sx={{ fontFamily: 'monospace', mb: 2 }}
        />

        {validationResult && (
          <Box mt={2}>
            {validationResult.valid ? (
              <Alert icon={<CheckCircleIcon />} severity="success">
                <Typography variant="subtitle2" fontWeight="600">Formula Validated!</Typography>
                <Typography variant="caption" display="block">Extracted Fields: {validationResult.extractedFields?.join(', ')}</Typography>
                <Typography variant="caption" display="block" sx={{ fontFamily: 'monospace', color: '#0284c7' }}>
                  SQL Projection: {validationResult.compiledSql}
                </Typography>
              </Alert>
            ) : (
              <Alert severity="error">
                {validationResult.error || 'Expression compilation error'}
              </Alert>
            )}
          </Box>
        )}
      </DialogContent>
      <DialogActions>
        <Button onClick={onClose}>Cancel</Button>
        <Button onClick={handleTestExpression} variant="outlined" disabled={loading}>
          Test Expression
        </Button>
        <Button
          onClick={() => {
            if (validationResult?.valid) {
              onSaveExpression(formula, validationResult.compiledSql || '');
              onClose();
            } else {
              handleTestExpression();
            }
          }}
          variant="contained"
          sx={{ bgcolor: '#0284c7', '&:hover': { bgcolor: '#0369a1' } }}
        >
          Save Metric
        </Button>
      </DialogActions>
    </Dialog>
  );
};
