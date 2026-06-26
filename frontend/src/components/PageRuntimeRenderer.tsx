import React, { useState, useEffect } from 'react';
import { useViewDefinition, ViewComponent } from '../api/uiMetadata';
import {
  Box,
  Typography,
  TextField,
  Button,
  Select,
  MenuItem,
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableRow,
  Paper,
  FormControl,
  InputLabel,
  CircularProgress,
  Alert
} from '@mui/material';

interface PageRuntimeRendererProps {
  viewId: string; // UUID or Name of the view definition
  dataContext: Record<string, any>; // Data from the workflow context
  onSubmit: (formData: Record<string, any>) => void;
}

export const PageRuntimeRenderer: React.FC<PageRuntimeRendererProps> = ({ 
  viewId, 
  dataContext, 
  onSubmit 
}) => {
  const { data: viewDef, isLoading, error } = useViewDefinition(viewId);
  const [formData, setFormData] = useState<Record<string, any>>({});

  // Initialize form state when view definition loads
  useEffect(() => {
    if (viewDef) {
      const initial: Record<string, any> = {};
      viewDef.components.forEach(comp => {
        if (comp.componentType !== 'Button' && comp.componentType !== 'ReadOnlyText' && comp.componentType !== 'Table') {
          // Pre-fill from dataContext if available, else empty
          initial[comp.dataKey] = dataContext[comp.dataKey] || '';
        }
      });
      setFormData(initial);
    }
  }, [viewDef, dataContext]);

  const handleChange = (key: string, value: any) => {
    setFormData(prev => ({ ...prev, [key]: value }));
  };

  const renderComponent = (comp: ViewComponent) => {
    const value = formData[comp.dataKey] ?? dataContext[comp.dataKey];
    
    switch (comp.componentType) {
      case 'ReadOnlyText':
        const style = comp.properties?.style === 'heading' ? 'h5' : 'body1';
        const isCurrency = comp.properties?.format === 'currency';
        const displayValue = isCurrency ? 
          new Intl.NumberFormat('en-US', { style: 'currency', currency: 'USD' }).format(Number(value || 0)) : 
          String(value || '');

        return (
          <Box key={comp.id} sx={{ mb: 2 }}>
            <Typography variant="subtitle2" color="textSecondary">{comp.label}</Typography>
            <Typography variant={style}>{displayValue}</Typography>
          </Box>
        );

      case 'Table':
        const columns = comp.properties?.columns || [];
        const rows = (Array.isArray(value) ? value : []) as any[];

        return (
          <Box key={comp.id} sx={{ mb: 2 }}>
            <Typography variant="h6" sx={{ mb: 1 }}>{comp.label}</Typography>
            <Paper variant="outlined">
              <Table size="small">
                <TableHead>
                  <TableRow>
                    {columns.map((col: any, idx: number) => (
                      <TableCell key={idx}>{col.header}</TableCell>
                    ))}
                  </TableRow>
                </TableHead>
                <TableBody>
                  {rows.map((row, rIdx) => (
                    <TableRow key={rIdx}>
                      {columns.map((col: any, cIdx: number) => (
                        <TableCell key={cIdx}>{row[col.key]}</TableCell>
                      ))}
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            </Paper>
          </Box>
        );

      case 'Select':
        const options = comp.properties?.options || [];
        return (
          <FormControl key={comp.id} fullWidth sx={{ mb: 2 }}>
            <InputLabel>{comp.label}</InputLabel>
            <Select
              value={value || ''}
              label={comp.label}
              onChange={(e) => handleChange(comp.dataKey, e.target.value)}
            >
              {options.map((opt: any) => (
                <MenuItem key={opt.value} value={opt.value}>{opt.label}</MenuItem>
              ))}
            </Select>
          </FormControl>
        );

      case 'TextArea':
      case 'TextField':
        return (
          <TextField
            key={comp.id}
            fullWidth
            multiline={comp.componentType === 'TextArea'}
            rows={comp.componentType === 'TextArea' ? 4 : 1}
            label={comp.label}
            placeholder={comp.properties?.placeholder}
            value={value || ''}
            onChange={(e) => handleChange(comp.dataKey, e.target.value)}
            sx={{ mb: 2 }}
          />
        );

      case 'Button':
        const isPrimary = comp.properties?.isPrimary;
        return (
          <Button
            key={comp.id}
            variant={isPrimary ? 'contained' : 'outlined'}
            color={isPrimary ? 'primary' : 'inherit'}
            onClick={() => onSubmit(formData)}
            sx={{ mt: 2 }}
          >
            {comp.label}
          </Button>
        );

      default:
        return <Alert severity="warning" key={comp.id}>Unknown component type: {comp.componentType}</Alert>;
    }
  };

  if (isLoading) return <CircularProgress />;
  if (error) return <Alert severity="error">Error loading view definition</Alert>;
  if (!viewDef) return <Alert severity="info">No view definition found</Alert>;

  return (
    <Box sx={{ p: 3, maxWidth: 800, mx: 'auto' }}>
      <Typography variant="h4" sx={{ mb: 3 }}>{viewDef.title}</Typography>
      {viewDef.components
        .sort((a, b) => a.order - b.order)
        .map(comp => renderComponent(comp))}
    </Box>
  );
};
