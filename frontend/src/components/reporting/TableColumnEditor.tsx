import React, { useState } from 'react';
import {
  Box,
  Typography,
  IconButton,
  TextField,
  Button,
  Paper,
  Tooltip,
  Select,
  MenuItem,
  FormControl,
  InputLabel
} from '@mui/material';
import {
  DndContext,
  closestCenter,
  KeyboardSensor,
  PointerSensor,
  useSensor,
  useSensors,
  DragEndEvent
} from '@dnd-kit/core';
import {
  arrayMove,
  SortableContext,
  sortableKeyboardCoordinates,
  verticalListSortingStrategy,
  useSortable
} from '@dnd-kit/sortable';
import { CSS } from '@dnd-kit/utilities';
import DragIndicatorIcon from '@mui/icons-material/DragIndicator';
import DeleteOutlineIcon from '@mui/icons-material/DeleteOutline';
import AddIcon from '@mui/icons-material/Add';

export interface ColumnConfig {
  id: string;
  field: string;
  headerName: string;
  width?: number;
  align?: 'left' | 'center' | 'right';
}

interface SortableColumnItemProps {
  column: ColumnConfig;
  onUpdate: (id: string, updates: Partial<ColumnConfig>) => void;
  onDelete: (id: string) => void;
}

const SortableColumnItem: React.FC<SortableColumnItemProps> = ({
  column,
  onUpdate,
  onDelete,
}) => {
  const {
    attributes,
    listeners,
    setNodeRef,
    transform,
    transition,
    isDragging,
  } = useSortable({ id: column.id });

  const style = {
    transform: CSS.Transform.toString(transform),
    transition,
    opacity: isDragging ? 0.5 : 1,
    zIndex: isDragging ? 1000 : 1,
  };

  return (
    <Paper
      ref={setNodeRef}
      style={style}
      elevation={0}
      sx={{
        display: 'flex',
        alignItems: 'center',
        gap: 1,
        p: 1,
        mb: 0.75,
        bgcolor: 'rgba(255, 255, 255, 0.03)',
        border: '1px solid rgba(255, 255, 255, 0.08)',
        borderRadius: 1.5,
      }}
    >
      <Box {...attributes} {...listeners} sx={{ cursor: 'grab', display: 'flex', alignItems: 'center' }}>
        <DragIndicatorIcon sx={{ fontSize: 18, color: 'text.secondary' }} />
      </Box>

      <Box sx={{ flex: 1, minWidth: 0 }}>
        <TextField
          size="small"
          label="Header Label"
          value={column.headerName}
          onChange={(e) => onUpdate(column.id, { headerName: e.target.value })}
          sx={{ '& .MuiInputBase-input': { fontSize: '0.75rem', py: 0.5 } }}
          fullWidth
        />
        <Typography variant="caption" color="text.secondary" sx={{ display: 'block', mt: 0.25, fontSize: '0.65rem' }}>
          Field: <code>{column.field}</code>
        </Typography>
      </Box>

      <TextField
        size="small"
        type="number"
        label="Width"
        value={column.width || ''}
        placeholder="Auto"
        onChange={(e) => onUpdate(column.id, { width: Number(e.target.value) || undefined })}
        sx={{ width: 65, '& .MuiInputBase-input': { fontSize: '0.75rem', py: 0.5 } }}
      />

      <Tooltip title="Remove column">
        <IconButton size="small" onClick={() => onDelete(column.id)} color="error">
          <DeleteOutlineIcon sx={{ fontSize: 16 }} />
        </IconButton>
      </Tooltip>
    </Paper>
  );
};

interface TableColumnEditorProps {
  columns: ColumnConfig[] | string[];
  onChange: (columns: ColumnConfig[]) => void;
  availableFields?: Array<{ name: string; type: string; label?: string }>;
}

export const TableColumnEditor: React.FC<TableColumnEditorProps> = ({
  columns: rawColumns,
  onChange,
  availableFields = [],
}) => {
  // Normalize string[] or ColumnConfig[] to ColumnConfig[]
  const normalizedColumns: ColumnConfig[] = React.useMemo(() => {
    return (rawColumns || []).map((col, idx) => {
      if (typeof col === 'string') {
        return {
          id: `col_${col}_${idx}`,
          field: col,
          headerName: col.charAt(0).toUpperCase() + col.slice(1).replace(/_/g, ' '),
        };
      }
      return col;
    });
  }, [rawColumns]);

  const [selectedFieldToAdd, setSelectedFieldToAdd] = useState('');

  const sensors = useSensors(
    useSensor(PointerSensor, { activationConstraint: { distance: 5 } }),
    useSensor(KeyboardSensor, { coordinateGetter: sortableKeyboardCoordinates })
  );

  const handleDragEnd = (event: DragEndEvent) => {
    const { active, over } = event;
    if (over && active.id !== over.id) {
      const oldIndex = normalizedColumns.findIndex((c) => c.id === active.id);
      const newIndex = normalizedColumns.findIndex((c) => c.id === over.id);
      const newCols = arrayMove(normalizedColumns, oldIndex, newIndex);
      onChange(newCols);
    }
  };

  const handleUpdate = (id: string, updates: Partial<ColumnConfig>) => {
    const newCols = normalizedColumns.map((c) => (c.id === id ? { ...c, ...updates } : c));
    onChange(newCols);
  };

  const handleDelete = (id: string) => {
    const newCols = normalizedColumns.filter((c) => c.id !== id);
    onChange(newCols);
  };

  const handleAddColumn = () => {
    if (!selectedFieldToAdd) return;
    const newCol: ColumnConfig = {
      id: `col_${selectedFieldToAdd}_${Date.now()}`,
      field: selectedFieldToAdd,
      headerName: selectedFieldToAdd.charAt(0).toUpperCase() + selectedFieldToAdd.slice(1).replace(/_/g, ' '),
    };
    onChange([...normalizedColumns, newCol]);
    setSelectedFieldToAdd('');
  };

  return (
    <Box sx={{ display: 'flex', flexDirection: 'column', gap: 1 }}>
      <Typography variant="caption" fontWeight="600" color="text.secondary">
        Columns & Reordering (Drag to Reorder)
      </Typography>

      <DndContext sensors={sensors} collisionDetection={closestCenter} onDragEnd={handleDragEnd}>
        <SortableContext items={normalizedColumns.map((c) => c.id)} strategy={verticalListSortingStrategy}>
          <Box sx={{ maxHeight: 220, overflowY: 'auto', pr: 0.5 }}>
            {normalizedColumns.map((col) => (
              <SortableColumnItem
                key={col.id}
                column={col}
                onUpdate={handleUpdate}
                onDelete={handleDelete}
              />
            ))}
            {normalizedColumns.length === 0 && (
              <Typography variant="caption" color="text.secondary" sx={{ fontStyle: 'italic', display: 'block', p: 1 }}>
                No columns selected. Add fields below.
              </Typography>
            )}
          </Box>
        </SortableContext>
      </DndContext>

      {/* Add column selector */}
      <Box sx={{ display: 'flex', gap: 1, mt: 1, alignItems: 'center' }}>
        <FormControl fullWidth size="small">
          <InputLabel sx={{ fontSize: '0.75rem' }}>Add Field Column</InputLabel>
          <Select
            size="small"
            value={selectedFieldToAdd}
            label="Add Field Column"
            onChange={(e) => setSelectedFieldToAdd(e.target.value)}
            sx={{ fontSize: '0.75rem', height: 32 }}
          >
            {availableFields.map((f) => (
              <MenuItem key={f.name} value={f.name} sx={{ fontSize: '0.8rem' }}>
                {f.label || f.name} ({f.type})
              </MenuItem>
            ))}
          </Select>
        </FormControl>
        <Button
          variant="outlined"
          size="small"
          onClick={handleAddColumn}
          disabled={!selectedFieldToAdd}
          startIcon={<AddIcon sx={{ fontSize: 16 }} />}
          sx={{ height: 32, textTransform: 'none', fontSize: '0.75rem', whiteSpace: 'nowrap' }}
        >
          Add
        </Button>
      </Box>
    </Box>
  );
};

export default TableColumnEditor;
