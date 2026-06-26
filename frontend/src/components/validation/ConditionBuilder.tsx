import React, { useState, useMemo } from 'react';
import { DndContext, useDraggable, useDroppable, DragOverlay, type DragEndEvent } from '@dnd-kit/core';
import {
  Box,
  Button,
  Card,
  CardContent,
  FormControl,
  Grid,
  InputLabel,
  MenuItem,
  Select,
  TextField,
  Typography,
  IconButton,
  Stack,
} from '@mui/material';
import { makeStyles } from '@mui/styles';
import AddIcon from '@mui/icons-material/Add';
import DeleteIcon from '@mui/icons-material/Delete';

const useStyles = makeStyles({
  conditionBlock: {
    padding: '12px',
    border: '1px solid #e0e0e0',
    borderRadius: '4px',
    backgroundColor: '#fafafa',
    marginBottom: '12px',
  },
  nestedBlock: {
    marginLeft: '20px',
    paddingLeft: '12px',
    borderLeft: '3px solid #1976d2',
  },
  jsonDisplay: {
    margin: 0,
    overflow: 'auto',
  },
});

interface SimpleCondition {
  field: string;
  operator: string;
  value: string;
}

interface ComplexCondition {
  type: 'AND' | 'OR' | 'NOT';
  conditions: (SimpleCondition | ComplexCondition)[];
}

type Condition = SimpleCondition | ComplexCondition;

const OPERATORS = [
  { value: '=', label: 'Equals (=)' },
  { value: '!=', label: 'Not Equals (!=)' },
  { value: '>', label: 'Greater Than (>)' },
  { value: '<', label: 'Less Than (<)' },
  { value: '>=', label: 'Greater or Equal (>=)' },
  { value: '<=', label: 'Less or Equal (<=)' },
  { value: 'contains', label: 'Contains' },
  { value: 'startsWith', label: 'Starts With' },
  { value: 'endsWith', label: 'Ends With' },
  { value: 'in', label: 'In List' },
  { value: 'regex', label: 'Regex Pattern' },
  { value: 'isEmpty', label: 'Is Empty' },
  { value: 'between', label: 'Between' },
];

interface ConditionBuilderProps {
  value: string;
  onChange: (json: string) => void;
}

const ConditionBuilder: React.FC<ConditionBuilderProps> = ({ value, onChange }) => {
  const classes = useStyles();
  const [condition, setCondition] = useState<Condition>(() => {
    try {
      return JSON.parse(value) as Condition;
    } catch {
      return {
        field: '',
        operator: '=',
        value: '',
      } as SimpleCondition;
    }
  });

  const isComplex = (c: Condition): c is ComplexCondition => {
    return 'type' in c && ('AND' === c.type || 'OR' === c.type || 'NOT' === c.type);
  };

  const isSimple = (c: Condition): c is SimpleCondition => {
    return 'field' in c && 'operator' in c;
  };

  const updateAndSync = (newCondition: Condition) => {
    setCondition(newCondition);
    onChange(JSON.stringify(newCondition));
  };

  // DnD handlers: simple palette items
  const paletteFields = useMemo(() => ['age', 'status', 'email', 'salary', 'vip'], []);
  const _paletteOperators = useMemo(() => ['=', '!=', '>', '<', '>=', '<=', 'contains', 'in'], []);

  const DraggableItem: React.FC<{id: string; label: string}> = ({ id, label }) => {
    const {attributes, listeners, setNodeRef, transform} = useDraggable({ id });
    const style = transform ? { transform: `translate3d(${transform.x}px, ${transform.y}px, 0)` } : undefined;
    const nodeRef = setNodeRef as unknown as (el: HTMLElement | null) => void;
    return (
      <Box ref={nodeRef} sx={{ border: '1px solid #ddd', px: 1, py: 0.5, borderRadius: 1, cursor: 'grab', backgroundColor: '#fff' }} {...listeners} {...attributes} style={style}>
        {label}
      </Box>
    );
  };

  const DroppableCanvas: React.FC<React.PropsWithChildren<{onDropField: (field: string) => void}>> = ({ onDropField: _onDropField, children }) => {
    const { isOver, setNodeRef } = useDroppable({ id: 'canvas' });
    const dropRef = setNodeRef as unknown as (el: HTMLElement | null) => void;
    return (
      <Box ref={dropRef} sx={{ minHeight: 80, p: 1, border: '1px dashed #bbb', backgroundColor: isOver ? '#f0f7ff' : '#fafafa' }}>
        {children}
      </Box>
    );
  };

  const handleDragEnd = (event: DragEndEvent) => {
    const { active, over } = event;
    if (!over) return;
    if (over.id === 'canvas' && active && active.id) {
      const field = String(active.id);
      // If simple condition present, set its field value, else add a new simple condition
      if (isSimple(condition)) {
        updateAndSync({ ...condition, field });
      } else if (isComplex(condition)) {
        // Add as a new simple condition into first group
        const newCond: SimpleCondition = { field, operator: '=', value: '' };
        updateAndSync({ ...condition, conditions: [...condition.conditions, newCond] } as ComplexCondition);
      }
    }
  };

  const [overlayContent, setOverlayContent] = useState<string | null>(null);

  const _handleDragStart = (id: string) => {
    setOverlayContent(id);
  };

  const handleDragCancel = () => setOverlayContent(null);

  const handleSimpleConditionChange = (
    field: keyof SimpleCondition,
    value: string
  ) => {
    if (isSimple(condition)) {
      updateAndSync({
        ...condition,
        [field]: value,
      });
    }
  };

  const handleConvertToComplex = (type: 'AND' | 'OR') => {
    updateAndSync({
      type,
      conditions: [condition],
    } as ComplexCondition);
  };

  const SimpleConditionEditor: React.FC<{
    cond: SimpleCondition;
    onUpdate: (field: keyof SimpleCondition, value: string) => void;
  }> = ({ cond, onUpdate }) => (
    <Box className={classes.conditionBlock}>
      <Grid container spacing={1} alignItems="flex-end">
        <Grid item xs={12} sm={4}>
          <TextField
            fullWidth
            label="Field"
            size="small"
            value={cond.field}
            onChange={(e) => onUpdate('field', e.target.value)}
            placeholder="e.g., age, status"
          />
        </Grid>
        <Grid item xs={12} sm={3}>
          <FormControl fullWidth size="small">
            <InputLabel>Operator</InputLabel>
            <Select
              value={cond.operator}
              label="Operator"
              onChange={(e) => onUpdate('operator', e.target.value)}
            >
              {OPERATORS.map((op) => (
                <MenuItem key={op.value} value={op.value}>
                  {op.label}
                </MenuItem>
              ))}
            </Select>
          </FormControl>
        </Grid>
        <Grid item xs={12} sm={5}>
          <TextField
            fullWidth
            label="Value"
            size="small"
            value={cond.value}
            onChange={(e) => onUpdate('value', e.target.value)}
            placeholder="e.g., 25"
          />
        </Grid>
      </Grid>
    </Box>
  );

  const ComplexConditionEditor: React.FC<{
    cond: ComplexCondition;
  }> = ({ cond }) => (
    <Box className={`${classes.conditionBlock} ${classes.nestedBlock}`}>
      <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', mb: 2 }}>
        <Typography variant="subtitle2">
          {cond.type} Block
        </Typography>
        <Button
          size="small"
          variant="outlined"
          startIcon={<AddIcon />}
          onClick={() => {
            updateAndSync({
              ...cond,
              conditions: [
                ...cond.conditions,
                {
                  field: '',
                  operator: '=',
                  value: '',
                } as SimpleCondition,
              ],
            });
          }}
        >
          Add Condition
        </Button>
      </Box>
      {cond.conditions.map((subcond, idx) => (
        <Box key={idx} sx={{ display: 'flex', gap: 1, mb: 1 }}>
          <Box sx={{ flex: 1 }}>
            {isSimple(subcond) ? (
              <SimpleConditionEditor
                cond={subcond}
                onUpdate={(field, val) => {
                  const newConditions = [...cond.conditions];
                  const newSubcond = newConditions[idx] as SimpleCondition;
                  newSubcond[field] = val;
                  updateAndSync({ ...cond, conditions: newConditions });
                }}
              />
            ) : (
              <ComplexConditionEditor cond={subcond} />
            )}
          </Box>
          <IconButton
            size="small"
            onClick={() => {
              updateAndSync({
                ...cond,
                conditions: cond.conditions.filter((_, i) => i !== idx),
              });
            }}
          >
            <DeleteIcon fontSize="small" />
          </IconButton>
        </Box>
      ))}
    </Box>
  );

  return (
    <Card>
      <CardContent>
  <DndContext onDragEnd={(e) => { handleDragEnd(e); handleDragCancel(); }} onDragStart={(e) => { if (e.active && e.active.id) setOverlayContent(String(e.active.id)); }}>
          <Stack spacing={2}>
            <Box sx={{ display: 'flex', gap: 1, mb: 1 }}>
              <Typography variant="subtitle2">Fields:</Typography>
              <Box sx={{ display: 'flex', gap: 1 }}>
                {paletteFields.map((f) => (
                  <DraggableItem key={f} id={f} label={f} />
                ))}
              </Box>
            </Box>

            <DroppableCanvas onDropField={(_f) => { /* handled by DndContext */ }}>
              {/* Canvas: editors are placed here */}
              {isSimple(condition) && (
                <SimpleConditionEditor
                  cond={condition}
                  onUpdate={handleSimpleConditionChange}
                />
              )}
              {isComplex(condition) && <ComplexConditionEditor cond={condition} />}
            </DroppableCanvas>

            <DragOverlay>
              {overlayContent ? (
                <Box sx={{ p: 1, backgroundColor: '#1976d2', color: '#fff', borderRadius: 1 }}>{overlayContent}</Box>
              ) : null}
            </DragOverlay>

            <Box sx={{ display: 'flex', gap: 1 }}>
              {isSimple(condition) && (
                <>
                  <Button size="small" variant="outlined" onClick={() => handleConvertToComplex('AND')}>
                    Convert to AND
                  </Button>
                  <Button size="small" variant="outlined" onClick={() => handleConvertToComplex('OR')}>
                    Convert to OR
                  </Button>
                </>
              )}
            </Box>

            <Box sx={{ mt: 2, p: 1, backgroundColor: '#f5f5f5', borderRadius: 1, fontFamily: 'monospace', fontSize: '0.85rem', wordBreak: 'break-all' }}>
              <Typography variant="caption">Condition JSON:</Typography>
              <pre className={classes.jsonDisplay}>{JSON.stringify(condition, null, 2)}</pre>
            </Box>
          </Stack>
        </DndContext>
      </CardContent>
    </Card>
  );
};

export default ConditionBuilder;
