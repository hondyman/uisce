import React, { useState, useCallback } from 'react';
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
  InputLabel,
  Tabs,
  Tab,
  Switch,
  FormControlLabel,
  Divider,
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
import VisibilityIcon from '@mui/icons-material/Visibility';
import VisibilityOffIcon from '@mui/icons-material/VisibilityOff';
import { ColumnConfig, AggregateFunction, createDefaultColumnConfig } from './tableColumnModel';

interface SortableColumnItemProps {
  column: ColumnConfig;
  isSelected: boolean;
  onSelect: () => void;
  onUpdate: (id: string, updates: Partial<ColumnConfig>) => void;
  onDelete: (id: string) => void;
}

const SortableColumnItem: React.FC<SortableColumnItemProps> = ({
  column,
  isSelected,
  onSelect,
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
      onClick={onSelect}
      sx={{
        display: 'flex',
        alignItems: 'center',
        gap: 0.5,
        p: 0.75,
        mb: 0.5,
        cursor: 'pointer',
        bgcolor: isSelected ? 'rgba(0, 212, 255, 0.08)' : 'rgba(255, 255, 255, 0.02)',
        border: `1px solid ${isSelected ? 'rgba(0, 212, 255, 0.35)' : 'rgba(255, 255, 255, 0.06)'}`,
        borderRadius: 1,
        transition: 'all 0.12s',
        '&:hover': {
          bgcolor: isSelected ? 'rgba(0, 212, 255, 0.12)' : 'rgba(255, 255, 255, 0.04)',
          borderColor: isSelected ? 'rgba(0, 212, 255, 0.5)' : 'rgba(255, 255, 255, 0.1)',
        },
      }}
    >
      <Box {...attributes} {...listeners} sx={{ cursor: 'grab', display: 'flex', alignItems: 'center', flexShrink: 0 }}>
        <DragIndicatorIcon sx={{ fontSize: 16, color: 'text.secondary' }} />
      </Box>

      <Box sx={{ flex: 1, minWidth: 0 }}>
        <Typography
          variant="caption"
          sx={{
            fontSize: '0.72rem',
            fontWeight: isSelected ? 700 : 500,
            color: column.visible ? 'text.primary' : 'text.disabled',
            display: 'block',
            whiteSpace: 'nowrap',
            overflow: 'hidden',
            textOverflow: 'ellipsis',
          }}
        >
          {column.headerText || column.field}
        </Typography>
        <Typography variant="caption" color="text.disabled" sx={{ fontSize: '0.6rem', display: 'block' }}>
          {column.field}{column.aggregate?.enabled ? ` · ${column.aggregate.function}` : ''}
        </Typography>
      </Box>

      <Tooltip title={column.visible ? 'Hide column' : 'Show column'}>
        <IconButton
          size="small"
          onClick={(e) => { e.stopPropagation(); onUpdate(column.id, { visible: !column.visible }); }}
          sx={{ p: 0.3 }}
        >
          {column.visible
            ? <VisibilityIcon sx={{ fontSize: 14, color: 'text.secondary' }} />
            : <VisibilityOffIcon sx={{ fontSize: 14, color: 'text.disabled' }} />}
        </IconButton>
      </Tooltip>

      <Tooltip title="Remove column">
        <IconButton
          size="small"
          onClick={(e) => { e.stopPropagation(); onDelete(column.id); }}
          color="error"
          sx={{ p: 0.3 }}
        >
          <DeleteOutlineIcon sx={{ fontSize: 14 }} />
        </IconButton>
      </Tooltip>
    </Paper>
  );
};

interface ColumnDetailEditorProps {
  column: ColumnConfig;
  onUpdate: (id: string, updates: Partial<ColumnConfig>) => void;
}

const ColumnDetailEditor: React.FC<ColumnDetailEditorProps> = ({ column, onUpdate }) => {
  const [tab, setTab] = useState(0);

  const update = (updates: Partial<ColumnConfig>) => onUpdate(column.id, updates);

  const handleAggregateToggle = useCallback((_: React.SyntheticEvent, v: boolean) => {
    const currentFn = column.aggregate?.function;
    const resolvedFn: AggregateFunction = (typeof currentFn === 'string') ? currentFn : 'SUM';
    const resolvedScope: 'column' | 'group' | 'report' = (column.aggregate?.scope as any) || 'column';
    update({ aggregate: { ...column.aggregate, enabled: v, function: resolvedFn, scope: resolvedScope } });
  }, [column.aggregate, update]);

  return (
    <Box sx={{ display: 'flex', flexDirection: 'column', height: '100%', overflow: 'hidden' }}>
      <Tabs
        value={tab}
        onChange={(_, v) => setTab(v)}
        sx={{
          minHeight: 32,
          '& .MuiTab-root': { minHeight: 32, py: 0, fontSize: '0.65rem', px: 1 },
          borderBottom: '1px solid rgba(255,255,255,0.08)',
        }}
      >
        <Tab label="General" />
        <Tab label="Style" />
        <Tab label="Align" />
        <Tab label="Format" />
        <Tab label="Aggregate" />
      </Tabs>

      <Box sx={{ flex: 1, overflow: 'auto', p: 1.5 }}>
        {tab === 0 && (
          <Box sx={{ display: 'flex', flexDirection: 'column', gap: 1.5 }}>
            <TextField
              size="small"
              label="Header Text"
              value={column.headerText}
              onChange={e => update({ headerText: e.target.value })}
              fullWidth
              sx={{ '& .MuiInputBase-input': { fontSize: '0.75rem' } }}
            />
            <TextField
              size="small"
              label="Field Key"
              value={column.field}
              disabled
              fullWidth
              sx={{ '& .MuiInputBase-input': { fontSize: '0.75rem' } }}
              helperText="Bound to data source field"
            />
            <Box sx={{ display: 'flex', gap: 1 }}>
              <TextField
                size="small"
                label="Width (px)"
                type="number"
                value={column.widthPx || ''}
                placeholder="Auto"
                onChange={e => update({ widthPx: Number(e.target.value) || 0 })}
                sx={{ '& .MuiInputBase-input': { fontSize: '0.75rem' }, width: 100 }}
              />
              <TextField
                size="small"
                label="Min Width (px)"
                type="number"
                value={column.minWidthPx || ''}
                placeholder="None"
                onChange={e => update({ minWidthPx: Number(e.target.value) || undefined })}
                sx={{ '& .MuiInputBase-input': { fontSize: '0.75rem' }, width: 100 }}
              />
            </Box>
            <FormControlLabel
              control={<Switch size="small" checked={column.visible} onChange={(_, v) => update({ visible: v })} />}
              label={<Typography variant="caption" sx={{ fontSize: '0.7rem' }}>Visible</Typography>}
            />
          </Box>
        )}

        {tab === 1 && (
          <Box sx={{ display: 'flex', flexDirection: 'column', gap: 1.5 }}>
            <Typography variant="caption" fontWeight={700} color="text.secondary" sx={{ fontSize: '0.65rem' }}>
              Header Style
            </Typography>
            <Box sx={{ display: 'flex', gap: 1 }}>
              <FormControl size="small" fullWidth sx={{ '& .MuiInputBase-input': { fontSize: '0.72rem' } }}>
                <InputLabel sx={{ fontSize: '0.65rem' }}>Font</InputLabel>
                <Select value={column.headerStyle?.fontFamily || ''} label="Font"
                  onChange={e => update({ headerStyle: { ...column.headerStyle, fontFamily: String(e.target.value) } })}>
                  {['Calibri','Arial','Segoe UI','Roboto','Times New Roman','Georgia','Courier New','Verdana'].map(f => (
                    <MenuItem key={f} value={f} sx={{ fontSize: '0.72rem' }}>{f}</MenuItem>
                  ))}
                </Select>
              </FormControl>
              <TextField
                size="small" label="Size (pt)" type="number" value={column.headerStyle?.fontSize || 11}
                onChange={e => update({ headerStyle: { ...column.headerStyle, fontSize: Number(e.target.value) } })}
                sx={{ '& .MuiInputBase-input': { fontSize: '0.72rem' }, width: 70 }}
              />
            </Box>
            <Box sx={{ display: 'flex', gap: 0.5 }}>
              {[{ key: 'fontWeight', label: 'B', bold: true }, { key: 'fontStyle', label: 'I', italic: true }].map(({ key, label, bold, italic }) => (
                <Button
                  key={key}
                  size="small"
                  variant={column.headerStyle?.[key as 'fontWeight' | 'fontStyle'] === (bold ? 700 : 'italic') ? 'contained' : 'outlined'}
                  onClick={() => {
                    const current = column.headerStyle?.[key as 'fontWeight' | 'fontStyle'];
                    update({ headerStyle: { ...column.headerStyle, [key]: bold ? (current === 700 ? 400 : 700) : (current === 'italic' ? 'normal' : 'italic') } });
                  }}
                  sx={{ minWidth: 28, p: 0.25, fontSize: '0.7rem', fontWeight: bold ? 700 : 400, fontStyle: italic ? 'italic' : 'normal' }}
                >{label}</Button>
              ))}
            </Box>
            <Divider sx={{ borderColor: 'rgba(255,255,255,0.06)' }} />

            <Typography variant="caption" fontWeight={700} color="text.secondary" sx={{ fontSize: '0.65rem' }}>
              Body Style
            </Typography>
            <Box sx={{ display: 'flex', gap: 1 }}>
              <FormControl size="small" fullWidth sx={{ '& .MuiInputBase-input': { fontSize: '0.72rem' } }}>
                <InputLabel sx={{ fontSize: '0.65rem' }}>Font</InputLabel>
                <Select value={column.bodyStyle?.fontFamily || ''} label="Font"
                  onChange={e => update({ bodyStyle: { ...column.bodyStyle, fontFamily: String(e.target.value) } })}>
                  {['Calibri','Arial','Segoe UI','Roboto','Times New Roman','Georgia','Courier New','Verdana'].map(f => (
                    <MenuItem key={f} value={f} sx={{ fontSize: '0.72rem' }}>{f}</MenuItem>
                  ))}
                </Select>
              </FormControl>
              <TextField
                size="small" label="Size (pt)" type="number" value={column.bodyStyle?.fontSize || 11}
                onChange={e => update({ bodyStyle: { ...column.bodyStyle, fontSize: Number(e.target.value) } })}
                sx={{ '& .MuiInputBase-input': { fontSize: '0.72rem' }, width: 70 }}
              />
            </Box>
            <Box sx={{ display: 'flex', gap: 0.5 }}>
              {[{ key: 'fontWeight', label: 'B', bold: true }, { key: 'fontStyle', label: 'I', italic: true }].map(({ key, label, bold, italic }) => (
                <Button
                  key={key}
                  size="small"
                  variant={column.bodyStyle?.[key as 'fontWeight' | 'fontStyle'] === (bold ? 700 : 'italic') ? 'contained' : 'outlined'}
                  onClick={() => {
                    const current = column.bodyStyle?.[key as 'fontWeight' | 'fontStyle'];
                    update({ bodyStyle: { ...column.bodyStyle, [key]: bold ? (current === 700 ? 400 : 700) : (current === 'italic' ? 'normal' : 'italic') } });
                  }}
                  sx={{ minWidth: 28, p: 0.25, fontSize: '0.7rem', fontWeight: bold ? 700 : 400, fontStyle: italic ? 'italic' : 'normal' }}
                >{label}</Button>
              ))}
            </Box>
            <Box sx={{ display: 'flex', gap: 1 }}>
              <TextField
                size="small" label="Text Color" type="color"
                value={column.bodyStyle?.color || '#E2E8F0'}
                onChange={e => update({ bodyStyle: { ...column.bodyStyle, color: e.target.value } })}
                sx={{ '& .MuiInputBase-input': { p: 0.5, height: 28 }, width: 80 }}
                inputProps={{ style: { padding: '4px 8px' } }}
              />
              <TextField
                size="small" label="Fill Color" type="color"
                value={column.bodyStyle?.backgroundColor || 'transparent'}
                onChange={e => update({ bodyStyle: { ...column.bodyStyle, backgroundColor: e.target.value } })}
                sx={{ '& .MuiInputBase-input': { p: 0.5, height: 28 }, width: 80 }}
                inputProps={{ style: { padding: '4px 8px' } }}
              />
            </Box>
          </Box>
        )}

        {tab === 2 && (
          <Box sx={{ display: 'flex', flexDirection: 'column', gap: 1.5 }}>
            <Typography variant="caption" fontWeight={700} color="text.secondary" sx={{ fontSize: '0.65rem' }}>
              Horizontal Alignment
            </Typography>
            <Box sx={{ display: 'flex', gap: 0.5 }}>
              {(['left', 'center', 'right'] as const).map(align => (
                <Button
                  key={align}
                  size="small"
                  variant={column.align === align ? 'contained' : 'outlined'}
                  onClick={() => update({ align })}
                  sx={{ flex: 1, textTransform: 'capitalize', fontSize: '0.7rem', py: 0.5 }}
                >
                  {align}
                </Button>
              ))}
            </Box>

            <Typography variant="caption" fontWeight={700} color="text.secondary" sx={{ fontSize: '0.65rem' }}>
              Vertical Alignment
            </Typography>
            <Box sx={{ display: 'flex', gap: 0.5 }}>
              {(['top', 'middle', 'bottom'] as const).map(va => (
                <Button
                  key={va}
                  size="small"
                  variant={column.verticalAlign === va ? 'contained' : 'outlined'}
                  onClick={() => update({ verticalAlign: va })}
                  sx={{ flex: 1, textTransform: 'capitalize', fontSize: '0.7rem', py: 0.5 }}
                >
                  {va}
                </Button>
              ))}
            </Box>

            <FormControlLabel
              control={<Switch size="small" checked={column.wrap} onChange={(_, v) => update({ wrap: v })} />}
              label={<Typography variant="caption" sx={{ fontSize: '0.7rem' }}>Wrap text</Typography>}
            />
          </Box>
        )}

        {tab === 3 && (
          <Box sx={{ display: 'flex', flexDirection: 'column', gap: 1.5 }}>
            <FormControl size="small" fullWidth sx={{ '& .MuiInputBase-input': { fontSize: '0.75rem' } }}>
              <InputLabel sx={{ fontSize: '0.7rem' }}>Format Type</InputLabel>
              <Select value={column.formatType} label="Format Type"
                onChange={e => update({ formatType: e.target.value as ColumnConfig['formatType'] })}>
                {['Auto','Currency','Percent','Decimal','Integer','Date','Text','Custom'].map(t => (
                  <MenuItem key={t} value={t} sx={{ fontSize: '0.72rem' }}>{t}</MenuItem>
                ))}
              </Select>
            </FormControl>
            {column.formatType === 'Custom' && (
              <TextField
                size="small"
                label="Format Mask"
                value={column.formatMask}
                onChange={e => update({ formatMask: e.target.value })}
                placeholder="e.g. #,##0.00;[Red]-#,##0.00"
                fullWidth
                sx={{ '& .MuiInputBase-input': { fontSize: '0.72rem', fontFamily: 'monospace' } }}
                helperText="Excel-style format string"
              />
            )}
            <Box sx={{ display: 'flex', gap: 1 }}>
              <TextField size="small" label="Prefix" value={column.formatPrefix}
                onChange={e => update({ formatPrefix: e.target.value })} placeholder="$"
                sx={{ '& .MuiInputBase-input': { fontSize: '0.72rem' }, flex: 1 }} />
              <TextField size="small" label="Suffix" value={column.formatSuffix}
                onChange={e => update({ formatSuffix: e.target.value })} placeholder="%"
                sx={{ '& .MuiInputBase-input': { fontSize: '0.72rem' }, flex: 1 }} />
            </Box>
            {column.formatMask && (
              <Paper sx={{ p: 1, bgcolor: 'rgba(0,0,0,0.15)', borderRadius: 1, border: '1px solid rgba(255,255,255,0.06)' }}>
                <Typography variant="caption" color="text.secondary" sx={{ fontSize: '0.6rem', display: 'block' }}>Preview:</Typography>
                <Typography variant="caption" sx={{ fontSize: '0.72rem', fontFamily: 'monospace', color: 'primary.main' }}>
                  1234.567 → {column.formatMask}
                </Typography>
              </Paper>
            )}
          </Box>
        )}

        {tab === 4 && (
          <Box sx={{ display: 'flex', flexDirection: 'column', gap: 1.5 }}>
            <FormControlLabel
              control={<Switch size="small" checked={column.aggregate?.enabled || false} onChange={handleAggregateToggle} />}
              label={<Typography variant="caption" sx={{ fontSize: '0.7rem' }}>Enable aggregation</Typography>}
            />
            {column.aggregate?.enabled && (
              <>
                <FormControl size="small" fullWidth sx={{ '& .MuiInputBase-input': { fontSize: '0.75rem' } }}>
                  <InputLabel sx={{ fontSize: '0.7rem' }}>Function</InputLabel>
                  <Select
                    value={typeof column.aggregate?.function === 'string' ? column.aggregate.function : 'custom'}
                    label="Function"
                    onChange={e => {
                      const fn = e.target.value;
                      if (fn === 'custom') {
                        update({ aggregate: { ...column.aggregate!, function: { customExpression: '' } } });
                      } else {
                        update({ aggregate: { ...column.aggregate!, function: fn as 'SUM' | 'AVG' | 'COUNT' | 'MIN' | 'MAX' | 'MEDIAN' } });
                      }
                    }}
                  >
                    {['SUM','AVG','COUNT','MIN','MAX','MEDIAN'].map(f => (
                      <MenuItem key={f} value={f} sx={{ fontSize: '0.72rem' }}>{f}</MenuItem>
                    ))}
                    <MenuItem value="custom" sx={{ fontSize: '0.72rem', fontStyle: 'italic' }}>Custom Expression…</MenuItem>
                  </Select>
                </FormControl>

                {typeof column.aggregate?.function === 'object' && 'customExpression' in column.aggregate?.function && (
                  <TextField
                    size="small"
                    label="Custom Expression"
                    value={column.aggregate.function.customExpression}
                    onChange={e => update({
                      aggregate: { ...column.aggregate!, function: { customExpression: e.target.value } }
                    })}
                    placeholder="e.g. SUM([field]) * 0.1"
                    fullWidth
                    multiline
                    minRows={2}
                    sx={{ '& .MuiInputBase-input': { fontSize: '0.72rem', fontFamily: 'monospace' } }}
                    helperText="Use [field] to reference columns"
                  />
                )}

                <FormControl size="small" fullWidth sx={{ '& .MuiInputBase-input': { fontSize: '0.75rem' } }}>
                  <InputLabel sx={{ fontSize: '0.7rem' }}>Scope</InputLabel>
                  <Select
                    value={column.aggregate?.scope || 'column'}
                    label="Scope"
                    onChange={e => update({ aggregate: { ...column.aggregate!, scope: e.target.value as 'column' | 'group' | 'report' } })}
                  >
                    <MenuItem value="column" sx={{ fontSize: '0.72rem' }}>Column (within group)</MenuItem>
                    <MenuItem value="group" sx={{ fontSize: '0.72rem' }}>Group</MenuItem>
                    <MenuItem value="report" sx={{ fontSize: '0.72rem' }}>Report (Grand Total)</MenuItem>
                  </Select>
                </FormControl>
              </>
            )}
          </Box>
        )}
      </Box>
    </Box>
  );
};

interface TableColumnEditorProps {
  columns: ColumnConfig[];
  onChange: (columns: ColumnConfig[]) => void;
  availableFields?: Array<{ name: string; type: string; label?: string }>;
}

export const TableColumnEditor: React.FC<TableColumnEditorProps> = ({
  columns: rawColumns,
  onChange,
  availableFields = [],
}) => {
  const [selectedId, setSelectedId] = useState<string | null>(null);
  const [addField, setAddField] = useState('');

  const sensors = useSensors(
    useSensor(PointerSensor, { activationConstraint: { distance: 5 } }),
    useSensor(KeyboardSensor, { coordinateGetter: sortableKeyboardCoordinates })
  );

  const handleDragEnd = useCallback((event: DragEndEvent) => {
    const { active, over } = event;
    if (over && active.id !== over.id) {
      const oldIndex = rawColumns.findIndex((c) => c.id === active.id);
      const newIndex = rawColumns.findIndex((c) => c.id === over.id);
      onChange(arrayMove(rawColumns, oldIndex, newIndex));
    }
  }, [rawColumns, onChange]);

  const handleUpdate = useCallback((id: string, updates: Partial<ColumnConfig>) => {
    onChange(rawColumns.map((c) => (c.id === id ? { ...c, ...updates } : c)));
  }, [rawColumns, onChange]);

  const handleDelete = useCallback((id: string) => {
    onChange(rawColumns.filter((c) => c.id !== id));
    if (selectedId === id) setSelectedId(null);
  }, [rawColumns, onChange, selectedId]);

  const handleAddColumn = useCallback(() => {
    if (!addField) return;
    const existing = rawColumns.find(c => c.field === addField);
    if (existing) { setAddField(''); return; }
    const newCol = createDefaultColumnConfig(`col_${addField}_${Date.now()}`, addField);
    const fieldDef = availableFields.find(f => f.name === addField);
    if (fieldDef?.label) {
      newCol.headerText = fieldDef.label;
      if (['number','int','float','decimal','currency','money'].includes(fieldDef.type?.toLowerCase() || '')) {
        newCol.formatType = 'Decimal';
      }
    }
    onChange([...rawColumns, newCol]);
    setSelectedId(newCol.id);
    setAddField('');
  }, [addField, rawColumns, onChange, availableFields]);

  const selectedColumn = rawColumns.find(c => c.id === selectedId) || null;

  const usedFields = new Set(rawColumns.map(c => c.field));

  return (
    <Box sx={{ display: 'flex', gap: 0, height: 340, overflow: 'hidden' }}>
      {/* Left pane: column list */}
      <Box sx={{ width: 220, flexShrink: 0, display: 'flex', flexDirection: 'column', overflow: 'hidden', borderRight: '1px solid rgba(255,255,255,0.06)' }}>
        <Box sx={{ p: 1, flex: 1, overflow: 'auto' }}>
          <DndContext sensors={sensors} collisionDetection={closestCenter} onDragEnd={handleDragEnd}>
            <SortableContext items={rawColumns.map(c => c.id)} strategy={verticalListSortingStrategy}>
              {rawColumns.map(col => (
                <SortableColumnItem
                  key={col.id}
                  column={col}
                  isSelected={selectedId === col.id}
                  onSelect={() => setSelectedId(col.id)}
                  onUpdate={handleUpdate}
                  onDelete={handleDelete}
                />
              ))}
              {rawColumns.length === 0 && (
                <Typography variant="caption" color="text.disabled" sx={{ fontStyle: 'italic', display: 'block', p: 1, fontSize: '0.7rem' }}>
                  No columns. Add fields below.
                </Typography>
              )}
            </SortableContext>
          </DndContext>
        </Box>

        {/* Add field */}
        <Box sx={{ p: 1, borderTop: '1px solid rgba(255,255,255,0.06)' }}>
          <Box sx={{ display: 'flex', gap: 0.5 }}>
            <FormControl size="small" fullWidth>
              <InputLabel sx={{ fontSize: '0.68rem' }}>Field</InputLabel>
              <Select
                value={addField}
                label="Field"
                onChange={e => setAddField(e.target.value)}
                sx={{ '& .MuiInputBase-input': { fontSize: '0.72rem' }, height: 30 }}
              >
                {availableFields.filter(f => !usedFields.has(f.name)).map(f => (
                  <MenuItem key={f.name} value={f.name} sx={{ fontSize: '0.72rem' }}>
                    {f.label || f.name}
                  </MenuItem>
                ))}
              </Select>
            </FormControl>
            <Button
              variant="outlined"
              size="small"
              onClick={handleAddColumn}
              disabled={!addField}
              startIcon={<AddIcon sx={{ fontSize: 14 }} />}
              sx={{ height: 30, minWidth: 28, p: 0.5, textTransform: 'none', fontSize: '0.7rem' }}
            />
          </Box>
        </Box>
      </Box>

      {/* Right pane: column detail */}
      <Box sx={{ flex: 1, overflow: 'hidden', display: 'flex', flexDirection: 'column' }}>
        {selectedColumn ? (
          <ColumnDetailEditor column={selectedColumn} onUpdate={handleUpdate} />
        ) : (
          <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'center', flex: 1, p: 2 }}>
            <Typography variant="caption" color="text.disabled" sx={{ fontStyle: 'italic', fontSize: '0.72rem', textAlign: 'center' }}>
              Select a column to edit its properties
            </Typography>
          </Box>
        )}
      </Box>
    </Box>
  );
};

export default TableColumnEditor;
