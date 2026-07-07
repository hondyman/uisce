/**
 * AutoFormRenderer — Self-building field picker.
 *
 * Consumes the Meta-API BO schema and renders the field picker dynamically.
 * Adding a semantic term to the graph automatically surfaces it here after the
 * cache TTL expires or the schema cache is invalidated.
 */

import React from 'react';
import {
  Box,
  List,
  ListItemButton,
  ListItemIcon,
  ListItemText,
  Tooltip,
  Typography,
  IconButton,
} from '@mui/material';
import AddIcon from '@mui/icons-material/Add';
import FilterIcon from '@mui/icons-material/FilterList';
import Numbers from '@mui/icons-material/Numbers';
import Abc from '@mui/icons-material/Abc';
import CalendarToday from '@mui/icons-material/CalendarToday';
import CheckCircle from '@mui/icons-material/CheckCircle';
import type { BOSchema, BOSchemaField } from '../types/queryDef';

interface Props {
  schema: BOSchema;
  onAddField: (field: BOSchemaField) => void;
  onAddFilter: (field: BOSchemaField) => void;
  isInQuery: (fieldId: string) => boolean;
}

const FieldIcon: React.FC<{ type?: string }> = ({ type }) => {
  const t = (type || '').toLowerCase();
  if (['integer', 'decimal', 'number', 'float', 'double'].includes(t)) {
    return <Numbers fontSize="small" sx={{ color: '#4caf50' }} />;
  }
  if (['date', 'datetime', 'timestamp', 'time'].includes(t)) {
    return <CalendarToday fontSize="small" sx={{ color: '#ff9800' }} />;
  }
  return <Abc fontSize="small" sx={{ color: '#2196f3' }} />;
};

const roleColor = (type?: string): string => {
  const t = (type || '').toLowerCase();
  if (t === 'measure' || t === 'calculated') return '#9c27b0';
  return '#2196f3';
};

export const AutoFormRenderer: React.FC<Props> = ({
  schema,
  onAddField,
  onAddFilter,
  isInQuery,
}) => {
  const grouped = React.useMemo(() => {
    const dimensions = schema.fields.filter(
      (f) => !['measure', 'calculated'].includes((f.type || '').toLowerCase())
    );
    const measures = schema.fields.filter(
      (f) => ['measure', 'calculated'].includes((f.type || '').toLowerCase())
    );
    return { dimensions, measures };
  }, [schema]);

  const renderField = (field: BOSchemaField) => {
    const selected = isInQuery(field.id);
    return (
      <ListItemButton
        key={field.id}
        selected={selected}
        sx={{ borderRadius: 1, mb: 0.5 }}
      >
        <ListItemIcon sx={{ minWidth: 32 }}>
          <FieldIcon type={field.type} />
        </ListItemIcon>
        <ListItemText
          primary={
            <Box sx={{ display: 'flex', alignItems: 'center' }}>
              {field.displayName || field.name}
              <Box
                component="span"
                sx={{
                  fontSize: '0.65rem',
                  textTransform: 'uppercase',
                  fontWeight: 700,
                  color: roleColor(field.type),
                  ml: 1,
                }}
              >
                {field.type || 'DIMENSION'}
              </Box>
            </Box>
          }
          secondary={field.physicalColumn}
          primaryTypographyProps={{ variant: 'body2' }}
          secondaryTypographyProps={{ variant: 'caption', sx: { fontSize: '0.65rem' } }}
        />
        <Tooltip title="Add to query">
          <IconButton
            size="small"
            onClick={() => onAddField(field)}
            disabled={selected}
          >
            <AddIcon fontSize="small" />
          </IconButton>
        </Tooltip>
        <Tooltip title="Add as filter">
          <IconButton size="small" onClick={() => onAddFilter(field)}>
            <FilterIcon fontSize="small" />
          </IconButton>
        </Tooltip>
        {selected && <CheckCircle fontSize="small" color="primary" sx={{ ml: 0.5 }} />}
      </ListItemButton>
    );
  };

  return (
    <Box>
      <Typography variant="overline" color="text.secondary">
        Schema: {schema.drivingTable}
      </Typography>

      {grouped.dimensions.length > 0 && (
        <>
          <Typography
            variant="caption"
            sx={{
              px: 2,
              pt: 1,
              pb: 0.5,
              display: 'block',
              fontWeight: 700,
              color: 'text.secondary',
              textTransform: 'uppercase',
            }}
          >
            Dimensions
          </Typography>
          <List dense sx={{ px: 1 }}>
            {grouped.dimensions.map(renderField)}
          </List>
        </>
      )}

      {grouped.measures.length > 0 && (
        <>
          <Typography
            variant="caption"
            sx={{
              px: 2,
              pt: 1,
              pb: 0.5,
              display: 'block',
              fontWeight: 700,
              color: 'text.secondary',
              textTransform: 'uppercase',
            }}
          >
            Measures
          </Typography>
          <List dense sx={{ px: 1 }}>
            {grouped.measures.map(renderField)}
          </List>
        </>
      )}

      {schema.fields.length === 0 && (
        <Box sx={{ p: 2, textAlign: 'center', color: 'text.secondary' }}>
          <Typography variant="body2">No visible fields in this schema.</Typography>
        </Box>
      )}
    </Box>
  );
};

export default AutoFormRenderer;
