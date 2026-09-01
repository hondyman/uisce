import React from 'react';
import {
  Box, Typography, Button, IconButton, TextField, FormControl,
  InputLabel, Select, MenuItem, Paper,
} from '@mui/material';
import DeleteOutlineIcon from '@mui/icons-material/DeleteOutline';
import AddIcon from '@mui/icons-material/Add';
import { ConditionalRule, ColorScaleConfig, DataBarConfig, ExpressionConfig } from '../tableColumnModel';

interface ConditionalRuleEditorProps {
  rules: ConditionalRule[];
  onChange: (rules: ConditionalRule[]) => void;
  columnIds: string[];
}

function uid() {
  return `${Date.now()}_${Math.random().toString(36).slice(2, 6)}`;
}

export const ConditionalRuleEditor: React.FC<ConditionalRuleEditorProps> = ({ rules, onChange, columnIds }) => {
  const add = () => {
    const newRule: ConditionalRule = {
      id: uid(),
      name: `Rule ${rules.length + 1}`,
      appliesTo: 'all',
      type: 'colorScale',
      precedence: rules.length,
      config: {
        minColor: '#EF4444',
        midColor: '#F59E0B',
        maxColor: '#10B981',
      } as ColorScaleConfig,
    };
    onChange([...rules, newRule]);
  };

  const update = (id: string, partial: Partial<ConditionalRule>) =>
    onChange(rules.map(r => r.id === id ? { ...r, ...partial } : r));

  const remove = (id: string) => onChange(rules.filter(r => r.id !== id));

  return (
    <Box sx={{ display: 'flex', flexDirection: 'column', gap: 1 }}>
      <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
        <Typography variant="caption" color="text.secondary" sx={{ fontSize: '0.7rem' }}>
          {rules.length} rule(s)
        </Typography>
        <Button size="small" startIcon={<AddIcon sx={{ fontSize: 14 }} />} onClick={add}
          sx={{ textTransform: 'none', fontSize: '0.7rem', py: 0.25 }}>
          Add Rule
        </Button>
      </Box>

      {rules.map((rule) => (
        <Paper key={rule.id} sx={{ p: 1.25, bgcolor: 'rgba(0,0,0,0.15)', border: '1px solid rgba(255,255,255,0.06)', borderRadius: 1 }}>
          <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', mb: 1 }}>
            <TextField size="small" label="Name" value={rule.name}
              onChange={e => update(rule.id, { name: e.target.value })}
              sx={{ '& .MuiInputBase-input': { fontSize: '0.72rem' }, flex: 1, mr: 1 }} />
            <IconButton size="small" onClick={() => remove(rule.id)} color="error" sx={{ p: 0.3 }}>
              <DeleteOutlineIcon sx={{ fontSize: 14 }} />
            </IconButton>
          </Box>

          <Box sx={{ display: 'flex', gap: 1, mb: 1 }}>
            <FormControl size="small" sx={{ flex: 1 }}>
              <InputLabel sx={{ fontSize: '0.65rem' }}>Type</InputLabel>
              <Select value={rule.type} label="Type"
                onChange={e => update(rule.id, { type: e.target.value as ConditionalRule['type'], config: getDefaultConfig(e.target.value as ConditionalRule['type']) })}
                sx={{ '& .MuiInputBase-input': { fontSize: '0.72rem' } }}>
                <MenuItem value="colorScale" sx={{ fontSize: '0.72rem' }}>3-Color Scale</MenuItem>
                <MenuItem value="dataBar" sx={{ fontSize: '0.72rem' }}>Data Bar</MenuItem>
                <MenuItem value="expression" sx={{ fontSize: '0.72rem' }}>Expression</MenuItem>
              </Select>
            </FormControl>
            <FormControl size="small" sx={{ flex: 1 }}>
              <InputLabel sx={{ fontSize: '0.65rem' }}>Applies To</InputLabel>
              <Select value={Array.isArray(rule.appliesTo) ? rule.appliesTo[0] || 'all' : rule.appliesTo} label="Applies To"
                onChange={e => update(rule.id, { appliesTo: e.target.value as any })}
                sx={{ '& .MuiInputBase-input': { fontSize: '0.72rem' } }}>
                <MenuItem value="all" sx={{ fontSize: '0.72rem' }}>All columns</MenuItem>
                {columnIds.map(cid => <MenuItem key={cid} value={cid} sx={{ fontSize: '0.72rem' }}>{cid}</MenuItem>)}
              </Select>
            </FormControl>
          </Box>

          {rule.type === 'colorScale' && (rule.config as ColorScaleConfig) && (
            <Box sx={{ display: 'flex', gap: 0.5 }}>
              <TextField size="small" type="color" value={(rule.config as ColorScaleConfig).minColor}
                onChange={e => update(rule.id, { config: { ...rule.config, minColor: e.target.value } as ColorScaleConfig })}
                sx={{ '& .MuiInputBase-input': { p: 0.5, height: 24 }, flex: 1 }}
                inputProps={{ style: { padding: '2px 4px' } }} />
              <TextField size="small" type="color" value={(rule.config as ColorScaleConfig).midColor}
                onChange={e => update(rule.id, { config: { ...rule.config, midColor: e.target.value } as ColorScaleConfig })}
                sx={{ '& .MuiInputBase-input': { p: 0.5, height: 24 }, flex: 1 }}
                inputProps={{ style: { padding: '2px 4px' } }} />
              <TextField size="small" type="color" value={(rule.config as ColorScaleConfig).maxColor}
                onChange={e => update(rule.id, { config: { ...rule.config, maxColor: e.target.value } as ColorScaleConfig })}
                sx={{ '& .MuiInputBase-input': { p: 0.5, height: 24 }, flex: 1 }}
                inputProps={{ style: { padding: '2px 4px' } }} />
            </Box>
          )}

          {rule.type === 'dataBar' && (rule.config as DataBarConfig) && (
            <Box sx={{ display: 'flex', gap: 1 }}>
              <TextField size="small" type="color" label="Bar color"
                value={(rule.config as DataBarConfig).color}
                onChange={e => update(rule.id, { config: { ...rule.config, color: e.target.value } as DataBarConfig })}
                sx={{ '& .MuiInputBase-input': { p: 0.5, height: 24 }, flex: 1 }}
                inputProps={{ style: { padding: '2px 4px' } }} />
            </Box>
          )}

          {rule.type === 'expression' && (rule.config as ExpressionConfig) && (
            <Box sx={{ display: 'flex', flexDirection: 'column', gap: 0.5 }}>
              <Typography variant="caption" color="text.secondary" sx={{ fontSize: '0.65rem' }}>
                Conditional expression — evaluates per row:
              </Typography>
              <TextField size="small" label="Expression"
                value={(rule.config as ExpressionConfig).expression}
                onChange={e => update(rule.id, { config: { ...rule.config, expression: e.target.value } as ExpressionConfig })}
                placeholder="e.g. [value] < 0"
                fullWidth multiline minRows={2}
                InputProps={{ sx: { fontSize: '0.72rem', fontFamily: 'monospace' } }}
                helperText="e.g. [value] < 0  or  [status] == 'Active'"
              />
            </Box>
          )}
        </Paper>
      ))}

      {rules.length === 0 && (
        <Typography variant="caption" color="text.disabled" sx={{ fontStyle: 'italic', display: 'block', fontSize: '0.7rem', textAlign: 'center', py: 1 }}>
          No conditional formatting rules.
        </Typography>
      )}
    </Box>
  );
};

function getDefaultConfig(type: ConditionalRule['type']): ConditionalRule['config'] {
  if (type === 'colorScale') return { minColor: '#EF4444', midColor: '#F59E0B', maxColor: '#10B981' } as ColorScaleConfig;
  if (type === 'dataBar') return { color: '#3B82F6', showValue: false } as DataBarConfig;
  return { expression: '', style: {} } as ExpressionConfig;
}

export default ConditionalRuleEditor;
