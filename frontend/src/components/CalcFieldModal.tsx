import React, { useState, useEffect } from 'react';
import {
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  Button,
  TextField,
  Box,
  Typography,
  IconButton,
  Stack,
  Alert,
  CircularProgress,
  Paper,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  FormControlLabel,
  Switch,
  MenuItem,
  Tooltip,
} from '@mui/material';
import CloseIcon from '@mui/icons-material/Close';
import PlayArrowIcon from '@mui/icons-material/PlayArrow';
import AutoAwesomeIcon from '@mui/icons-material/AutoAwesome';
import Editor from '@monaco-editor/react';
import { createCalcField, previewCalcField, type CalcField } from '../api/calcFields';
import { useNotification } from '../hooks/useNotification';

interface CalcFieldModalProps {
  isOpen: boolean;
  onClose: () => void;
  objectId: string;
  onSaved: () => void;
  initialData?: Partial<CalcField>;
}

export const CalcFieldModal = ({
  isOpen,
  onClose,
  objectId,
  onSaved,
  initialData,
}: CalcFieldModalProps) => {
  const notification = useNotification();
  const [loading, setLoading] = useState(false);
  const [previewLoading, setPreviewLoading] = useState(false);
  const [previewData, setPreviewData] = useState<{ columns: string[]; rows: any[][] } | null>(null);
  const [aiLoading, setAiLoading] = useState(false);

  const [formData, setFormData] = useState<CalcField>({
    object_id: objectId,
    name: '',
    sql_expr: '',
    data_type: 'number',
    is_measure: true,
    realtime: false,
  });

  useEffect(() => {
    if (initialData) {
      setFormData(prev => ({ ...prev, ...initialData }));
    } else {
      setFormData({
        object_id: objectId,
        name: '',
        sql_expr: '',
        data_type: 'number',
        is_measure: true,
        realtime: false,
      });
    }
  }, [initialData, objectId, isOpen]);

  const handleSave = async () => {
    if (!formData.name || !formData.sql_expr) {
      notification.error('Name and SQL Expression are required');
      return;
    }

    setLoading(true);
    try {
      await createCalcField(formData);
      notification.success('Calculated field saved successfully');
      onSaved();
      onClose();
    } catch (err) {
      notification.error(err instanceof Error ? err.message : 'Failed to save calculated field');
    } finally {
      setLoading(false);
    }
  };

  const handlePreview = async () => {
    if (!formData.sql_expr) {
      notification.error('SQL Expression is required for preview');
      return;
    }

    setPreviewLoading(true);
    try {
      const resp = await previewCalcField({
        object_id: objectId,
        sql_expr: formData.sql_expr,
        limit: 5,
      });
      setPreviewData(resp);
    } catch (err) {
      notification.error(err instanceof Error ? err.message : 'Preview failed');
    } finally {
      setPreviewLoading(false);
    }
  };

  const handleAiSuggest = async () => {
    setAiLoading(true);
    try {
      // Logic for AI Suggest
      // Call /api/ai/suggest or similar
      const prompt = `Suggest a SQL expression for a calculated field named "${formData.name}" for a business object with ID ${objectId}.`;
      const resp = await fetch('/api/ai/proxy/completions', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          prompt: `You are a SQL expert. Provide ONLY the SQL expression for: ${prompt}. Example: COUNT(DISTINCT user_id)`,
        }),
      });
      const data = await resp.json();
      if (data.text) {
        setFormData(prev => ({ ...prev, sql_expr: data.text.trim() }));
      }
    } catch (err) {
      notification.error('AI suggestion failed');
    } finally {
      setAiLoading(false);
    }
  };

  return (
    <Dialog open={isOpen} onClose={onClose} maxWidth="md" fullWidth>
      <DialogTitle>
        <Stack direction="row" justifyContent="space-between" alignItems="center">
          <Typography variant="h6">
            {initialData?.id ? 'Edit Calculated Field' : 'Add Calculated Field'}
          </Typography>
          <IconButton onClick={onClose} size="small">
            <CloseIcon />
          </IconButton>
        </Stack>
      </DialogTitle>

      <DialogContent>
        <Stack spacing={3} sx={{ mt: 1 }}>
          <TextField
            label="Field Name"
            fullWidth
            value={formData.name}
            onChange={e => setFormData(prev => ({ ...prev, name: e.target.value }))}
            placeholder="e.g. total_weighted_market_value"
          />

          <Box>
            <Stack direction="row" justifyContent="space-between" alignItems="center" sx={{ mb: 1 }}>
              <Typography variant="subtitle2" color="text.secondary">
                SQL Expression
              </Typography>
              <Button
                size="small"
                startIcon={aiLoading ? <CircularProgress size={16} /> : <AutoAwesomeIcon />}
                onClick={handleAiSuggest}
                disabled={aiLoading}
              >
                AI Suggest
              </Button>
            </Stack>
            <Paper variant="outlined" sx={{ height: 200, overflow: 'hidden' }}>
              <Editor
                height="100%"
                defaultLanguage="sql"
                theme="vs-dark"
                value={formData.sql_expr}
                onChange={val => setFormData(prev => ({ ...prev, sql_expr: val || '' }))}
                options={{
                  minimap: { enabled: false },
                  fontSize: 14,
                  lineNumbers: 'on',
                  scrollBeyondLastLine: false,
                }}
              />
            </Paper>
          </Box>

          <Stack direction="row" spacing={2}>
            <TextField
              select
              label="Data Type"
              sx={{ flex: 1 }}
              value={formData.data_type}
              onChange={e => setFormData(prev => ({ ...prev, data_type: e.target.value }))}
            >
              <MenuItem value="number">Number</MenuItem>
              <MenuItem value="string">String</MenuItem>
              <MenuItem value="boolean">Boolean</MenuItem>
              <MenuItem value="date">Date</MenuItem>
            </TextField>

            <FormControlLabel
              control={
                <Switch
                  checked={formData.is_measure}
                  onChange={e => setFormData(prev => ({ ...prev, is_measure: e.target.checked }))}
                />
              }
              label="Is Measure"
            />

            <FormControlLabel
              control={
                <Switch
                  checked={formData.realtime}
                  onChange={e => setFormData(prev => ({ ...prev, realtime: e.target.checked }))}
                />
              }
              label="Real-time"
            />
          </Stack>

          <Box>
            <Button
              variant="outlined"
              startIcon={previewLoading ? <CircularProgress size={16} /> : <PlayArrowIcon />}
              onClick={handlePreview}
              disabled={previewLoading}
              sx={{ mb: 1 }}
            >
              Run Preview
            </Button>

            {previewData && (
              <TableContainer component={Paper} variant="outlined" sx={{ maxHeight: 200 }}>
                <Table size="small" stickyHeader>
                  <TableHead>
                    <TableRow>
                      {previewData.columns.map(col => (
                        <TableCell key={col} sx={{ fontWeight: 'bold' }}>
                          {col}
                        </TableCell>
                      ))}
                    </TableRow>
                  </TableHead>
                  <TableBody>
                    {previewData.rows.map((row, i) => (
                      <TableRow key={i}>
                        {row.map((cell, j) => (
                          <TableCell key={j}>{String(cell)}</TableCell>
                        ))}
                      </TableRow>
                    ))}
                  </TableBody>
                </Table>
              </TableContainer>
            )}
          </Box>
        </Stack>
      </DialogContent>

      <DialogActions sx={{ p: 3 }}>
        <Button onClick={onClose}>Cancel</Button>
        <Button
          variant="contained"
          onClick={handleSave}
          disabled={loading}
          startIcon={loading && <CircularProgress size={16} />}
        >
          {initialData?.id ? 'Update Field' : 'Create Field'}
        </Button>
      </DialogActions>
    </Dialog>
  );
};
