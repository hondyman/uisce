import React, { useState } from 'react';
import {
  Box,
  Typography,
  TextField,
  InputAdornment,
  Chip,
  Button,
  Paper,
  Tooltip,
  IconButton
} from '@mui/material';
import { useDraggable } from '@dnd-kit/core';
import SearchIcon from '@mui/icons-material/Search';
import AddIcon from '@mui/icons-material/Add';
import TableChartIcon from '@mui/icons-material/TableChart';
import DragIndicatorIcon from '@mui/icons-material/DragIndicator';
import { dedupeFields } from '../../utils/dedupeFields';

export interface BOField {
  name: string;
  technicalName?: string;
  label?: string;
  dataType?: string;
  type?: string;
  description?: string;
  isCore?: boolean;
}

interface DraggableBOFieldItemProps {
  field: BOField;
  onAdd: (field: BOField) => void;
}

const DraggableBOFieldItem: React.FC<DraggableBOFieldItemProps> = ({ field, onAdd }) => {
  const { attributes, listeners, setNodeRef, isDragging } = useDraggable({
    id: `bofield_${field.name}_${field.isCore ? 'core' : 'custom'}`,
    data: {
      isBOField: true,
      field,
    },
  });

  const getBadgeColor = (type?: string) => {
    const t = (type || '').toLowerCase();
    if (['number', 'integer', 'float', 'decimal', 'currency'].includes(t)) return 'success';
    if (['date', 'datetime', 'timestamp'].includes(t)) return 'warning';
    if (['boolean', 'bool'].includes(t)) return 'secondary';
    return 'info';
  };

  return (
    <Paper
      ref={setNodeRef}
      elevation={0}
      sx={{
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'space-between',
        p: 1,
        mb: 0.75,
        bgcolor: 'rgba(255, 255, 255, 0.03)',
        border: '1px solid rgba(255, 255, 255, 0.08)',
        borderRadius: 1.5,
        cursor: 'grab',
        opacity: isDragging ? 0.4 : 1,
        '&:hover': {
          bgcolor: 'rgba(99, 102, 241, 0.08)',
          borderColor: 'rgba(99, 102, 241, 0.3)',
          '& .add-btn': { opacity: 1 },
        },
      }}
      {...attributes}
      {...listeners}
    >
      <Box sx={{ display: 'flex', alignItems: 'center', gap: 1, minWidth: 0 }}>
        <DragIndicatorIcon sx={{ fontSize: 16, color: 'text.secondary' }} />
        <Box sx={{ minWidth: 0 }}>
          <Typography variant="body2" sx={{ fontSize: '0.75rem', fontWeight: 600, noWrap: true }}>
            {field.label || field.name}
          </Typography>
          <Typography variant="caption" color="text.secondary" sx={{ display: 'block', fontSize: '0.65rem' }}>
            {field.name}
          </Typography>
        </Box>
      </Box>

      <Box sx={{ display: 'flex', alignItems: 'center', gap: 0.5 }}>
        <Chip
          size="small"
          label={field.dataType || field.type || 'string'}
          color={getBadgeColor(field.dataType || field.type) as any}
          variant="outlined"
          sx={{ height: 18, fontSize: '0.6rem', textTransform: 'uppercase' }}
        />
        <Tooltip title="Add to Canvas">
          <IconButton
            className="add-btn"
            size="small"
            onClick={(e) => {
              e.stopPropagation();
              onAdd(field);
            }}
            sx={{ opacity: 0.6, p: 0.25, '&:hover': { color: 'primary.main' } }}
          >
            <AddIcon sx={{ fontSize: 14 }} />
          </IconButton>
        </Tooltip>
      </Box>
    </Paper>
  );
};

interface BOFieldsPaletteProps {
  selectedBO?: any;
  relatedBOs?: any[];
  onAddFieldToCanvas: (field: BOField) => void;
  onAddAllAsTable: (fields: BOField[]) => void;
}

export const BOFieldsPalette: React.FC<BOFieldsPaletteProps> = ({
  selectedBO,
  onAddFieldToCanvas,
  onAddAllAsTable,
}) => {
  const [searchTerm, setSearchTerm] = useState('');

  if (!selectedBO) {
    return (
      <Box sx={{ p: 2, textAlign: 'center', color: 'text.secondary' }}>
        <Typography variant="caption">Select a Business Object in the top bar to view its palette fields.</Typography>
      </Box>
    );
  }

  const coreFields: BOField[] = (selectedBO.coreFields || []).map((f: any) => ({
    name: f.name || f.technicalName,
    label: f.displayName || f.name,
    dataType: f.dataType || 'string',
    isCore: true,
  }));

  const customFields: BOField[] = (selectedBO.customFields || []).map((f: any) => ({
    name: f.name || f.technicalName,
    label: f.displayName || f.name,
    dataType: f.dataType || 'string',
    isCore: false,
  }));

  const allFields = dedupeFields([...coreFields, ...customFields]);

  const filteredFields = allFields.filter(
    (f) =>
      (f.label || f.name).toLowerCase().includes(searchTerm.toLowerCase()) ||
      f.name.toLowerCase().includes(searchTerm.toLowerCase())
  );

  return (
    <Box sx={{ display: 'flex', flexDirection: 'column', height: '100%' }}>
      {/* Quick Search */}
      <Box sx={{ p: 1.5, pb: 1 }}>
        <TextField
          size="small"
          placeholder="Search fields..."
          value={searchTerm}
          onChange={(e) => setSearchTerm(e.target.value)}
          fullWidth
          InputProps={{
            startAdornment: (
              <InputAdornment position="start">
                <SearchIcon sx={{ fontSize: 16, color: 'text.secondary' }} />
              </InputAdornment>
            ),
          }}
          sx={{ '& .MuiInputBase-input': { fontSize: '0.75rem', py: 0.5 } }}
        />
      </Box>

      {/* Add All Button */}
      <Box sx={{ px: 1.5, pb: 1 }}>
        <Button
          fullWidth
          variant="outlined"
          size="small"
          startIcon={<TableChartIcon sx={{ fontSize: 16 }} />}
          onClick={() => onAddAllAsTable(allFields)}
          disabled={allFields.length === 0}
          sx={{ textTransform: 'none', fontSize: '0.72rem', py: 0.5 }}
        >
          Add All as Table Grid ({allFields.length})
        </Button>
      </Box>

      {/* Fields List */}
      <Box sx={{ flex: 1, overflowY: 'auto', px: 1.5 }}>
        {filteredFields.map((field) => (
          <DraggableBOFieldItem
            key={`${field.name}_${field.isCore ? 'core' : 'custom'}`}
            field={field}
            onAdd={onAddFieldToCanvas}
          />
        ))}

        {filteredFields.length === 0 && (
          <Typography variant="caption" color="text.secondary" sx={{ display: 'block', textAlign: 'center', py: 2 }}>
            No fields match "{searchTerm}"
          </Typography>
        )}
      </Box>
    </Box>
  );
};

export default BOFieldsPalette;
