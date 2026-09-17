import React, { useEffect, useMemo, useRef, useState } from 'react';
import { Box, Typography, CircularProgress, Chip } from '@mui/material';
import LockIcon from '@mui/icons-material/Lock';
import DragIndicatorIcon from '@mui/icons-material/DragIndicator';
import TextFieldsIcon from '@mui/icons-material/TextFields';
import NumbersIcon from '@mui/icons-material/Numbers';
import CalendarMonthIcon from '@mui/icons-material/CalendarMonth';
import CheckBoxIcon from '@mui/icons-material/CheckBox';
import ListAltIcon from '@mui/icons-material/ListAlt';
import AddCircleOutlineIcon from '@mui/icons-material/AddCircleOutline';
import { useDndMonitor, type DragEndEvent } from '@dnd-kit/core';
import { SortableContext, useSortable, rectSortingStrategy } from '@dnd-kit/sortable';
import { CSS } from '@dnd-kit/utilities';
import { fetchBOSchema } from '../../features/query-builder/services/queryBuilderApi';
import type { BOSchemaField } from '../../features/query-builder/types/queryDef';

const GRID_COLS = 12;

export interface FieldStyleEntry { fontSize?: 'small' | 'medium' | 'large'; bold?: boolean; color?: string }
export interface FieldLayoutEntry { order: number; colSpan: number; section?: string }
export interface FieldOverrideEntry { label?: string; required?: boolean; hidden?: boolean; style?: FieldStyleEntry }

/** field.name -> layout/override maps, stored on the Form component's own props (fieldLayout/fieldOverrides). */
export interface FormFieldsProps {
  fieldLayout?: Record<string, FieldLayoutEntry>;
  fieldOverrides?: Record<string, FieldOverrideEntry>;
}

interface FormFieldsDesignerProps extends FormFieldsProps {
  /** The placing Form/DetailPanel widget id - prefixes sortable ids so two
   * forms on the same page don't collide inside PageEditor's shared DndContext. */
  componentId: string;
  boId: string;
  tenantId: string;
  selectedFieldName: string | null;
  onSelectField: (fieldName: string | null) => void;
  onLayoutChange: (fieldName: string, entry: Partial<FieldLayoutEntry>) => void;
  onReorder: (orderedFieldNames: string[]) => void;
  /** Brings a deleted (hidden) field back onto the form - see PropertiesPanel.tsx's field delete button. */
  onUnhideField: (fieldName: string) => void;
}

/** Form is the editable record widget; DetailPanel was a palette duplicate of
 * the same selected-record view, so renderer/properties treat them as one. */
export const isFormLikeWidget = (type: string): boolean => type === 'Form' || type === 'DetailPanel';

const fieldSortableId = (componentId: string, fieldName: string) => `form-field:${componentId}:${fieldName}`;

const FONT_SIZE_PX: Record<NonNullable<FieldStyleEntry['fontSize']>, number> = { small: 11, medium: 13, large: 16 };

/** Applies a field's style override to its rendered label - shared by the design tile and BOFormWidget so they match. */
export const labelSxFromStyle = (style?: FieldStyleEntry): Record<string, unknown> => ({
  fontSize: style?.fontSize ? FONT_SIZE_PX[style.fontSize] : undefined,
  fontWeight: style?.bold ? 700 : undefined,
  color: style?.color || undefined,
});

/** Shared with PropertiesPanel.tsx's Data Binding display and "Change field" picker, so a field's type icon is consistent wherever it appears. */
export const TYPE_ICON: Record<string, React.ElementType> = {
  text: TextFieldsIcon,
  string: TextFieldsIcon,
  number: NumbersIcon,
  integer: NumbersIcon,
  decimal: NumbersIcon,
  date: CalendarMonthIcon,
  datetime: CalendarMonthIcon,
  boolean: CheckBoxIcon,
};

/** One field tile - a dedicated grip icon drags it to reorder (the tile body itself only selects, so a click never
 * gets mistaken for a drag), and a visible handle along the right edge resizes it, both fully inside the tile's own
 * bounds so neither ever lands on a neighboring tile. */
/** Resolves a field's data-type icon consistently everywhere one is shown (this tile, PropertiesPanel's Data Binding block and "Change field" picker). */
export const iconForField = (field: Pick<BOSchemaField, 'type' | 'referenceBoId' | 'enumValues'>): React.ElementType =>
  TYPE_ICON[(field.type || '').toLowerCase()] || (field.referenceBoId || (field.enumValues && field.enumValues.length > 0) ? ListAltIcon : TextFieldsIcon);

const FieldTile: React.FC<{
  field: BOSchemaField;
  sortableId: string;
  componentId: string;
  colSpan: number;
  order: number;
  selected: boolean;
  label: string;
  required: boolean;
  style?: FieldStyleEntry;
  onSelect: () => void;
  onResize: (deltaCols: number) => void;
}> = ({ field, sortableId, componentId, colSpan, order, selected, label, required, style, onSelect, onResize }) => {
  const { attributes, listeners, setNodeRef, transform, isDragging } = useSortable({
    id: sortableId,
    data: { kind: 'form-field', componentId, fieldName: field.name },
  });
  const tileRef = useRef<HTMLDivElement | null>(null);
  const [resizing, setResizing] = useState(false);
  const TypeIcon = iconForField(field);

  const handleResizeStart = (e: React.PointerEvent) => {
    e.stopPropagation();
    e.preventDefault();
    (e.target as Element).setPointerCapture?.(e.pointerId);
    setResizing(true);
    const startX = e.clientX;
    const tileWidth = tileRef.current?.getBoundingClientRect().width || 1;
    const colWidth = tileWidth / colSpan;
    let lastDelta = 0;
    const onMove = (moveEvt: PointerEvent) => {
      const deltaPx = moveEvt.clientX - startX;
      const deltaCols = Math.round(deltaPx / colWidth);
      if (deltaCols !== lastDelta) {
        onResize(deltaCols - lastDelta);
        lastDelta = deltaCols;
      }
    };
    const onUp = () => {
      setResizing(false);
      window.removeEventListener('pointermove', onMove);
      window.removeEventListener('pointerup', onUp);
    };
    window.addEventListener('pointermove', onMove);
    window.addEventListener('pointerup', onUp);
  };

  return (
    <Box
      ref={(el: HTMLDivElement) => { setNodeRef(el); tileRef.current = el; }}
      style={{ transform: CSS.Translate.toString(transform), order }}
      onClick={(e) => { e.stopPropagation(); onSelect(); }}
      sx={{
        gridColumn: `span ${colSpan}`,
        position: 'relative',
        border: '1px solid',
        borderColor: selected ? 'primary.main' : 'rgba(0,0,0,0.12)',
        borderRadius: 1,
        bgcolor: selected ? 'rgba(25, 118, 210, 0.06)' : 'background.paper',
        boxShadow: selected ? '0 0 0 1px' : undefined,
        p: 1,
        pr: 1.75,
        cursor: 'pointer',
        opacity: isDragging ? 0.4 : 1,
        userSelect: resizing ? 'none' : undefined,
        '&:hover': { borderColor: 'primary.light' },
      }}
    >
      <Box sx={{ display: 'flex', alignItems: 'flex-start', justifyContent: 'space-between', gap: 0.5 }}>
        <Box sx={{ display: 'flex', alignItems: 'center', gap: 0.5, minWidth: 0 }}>
          <TypeIcon sx={{ fontSize: 13, color: 'text.secondary', flexShrink: 0 }} />
          <Typography variant="caption" fontWeight={600} noWrap sx={labelSxFromStyle(style)}>
            {label}
          </Typography>
          {required && <Typography component="span" variant="caption" color="error.main">*</Typography>}
          {field.referenceBoId && <LockIcon sx={{ fontSize: 11, opacity: 0.5, flexShrink: 0 }} titleAccess="Reference field" />}
        </Box>
        {/* Dedicated drag handle - dnd-kit's listeners live ONLY here, so
            clicking anywhere else on the tile is unambiguously a select,
            never an accidental drag. */}
        <Box
          {...attributes}
          {...listeners}
          onClick={(e) => e.stopPropagation()}
          sx={{ cursor: 'grab', color: 'text.disabled', flexShrink: 0, '&:hover': { color: 'text.secondary' } }}
        >
          <DragIndicatorIcon sx={{ fontSize: 16 }} />
        </Box>
      </Box>
      {/* A grey placeholder bar, not a real input - this is the whole point:
          design mode shows the FIELD exists and its shape, never a value. */}
      <Box sx={{ mt: 0.5, height: 22, borderRadius: 0.5, bgcolor: 'rgba(0,0,0,0.06)' }} />
      <Typography variant="caption" color="text.disabled" sx={{ display: 'block', mt: 0.25, fontFamily: 'monospace', fontSize: 10 }} noWrap>
        {field.physicalColumn || field.name}
      </Typography>
      {/* Resize handle - a real, always-visible grip fully inside the
          tile's own bounds (never overlapping a neighbor), widened well
          past its painted width for an easy hit target. */}
      <Box
        onPointerDown={handleResizeStart}
        sx={{
          position: 'absolute', top: 0, right: 0, bottom: 0, width: 14,
          display: 'flex', alignItems: 'center', justifyContent: 'center',
          cursor: 'col-resize', touchAction: 'none',
          '&:hover .resize-grip, &:active .resize-grip': { bgcolor: 'primary.main', opacity: 1 },
        }}
      >
        <Box className="resize-grip" sx={{ width: 3, height: '60%', borderRadius: 1, bgcolor: 'rgba(0,0,0,0.15)', opacity: 0.7 }} />
      </Box>
    </Box>
  );
};

/**
 * Design-time field editor for a Form widget - lets the author resize
 * (the right-edge grip, snaps to a 12-column grid), reorder (the drag-handle
 * icon), and select individual fields for the Properties panel, without
 * ever fetching or showing a real record's values. BOFormWidget.tsx is the
 * preview/runtime counterpart: it renders the same fieldLayout/
 * fieldOverrides but with real data, only reachable from Preview mode.
 */
const FormFieldsDesigner: React.FC<FormFieldsDesignerProps> = ({
  componentId, boId, tenantId, fieldLayout, fieldOverrides, selectedFieldName, onSelectField, onLayoutChange, onReorder, onUnhideField,
}) => {
  const [schema, setSchema] = useState<{ fields: BOSchemaField[] } | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    if (!boId || !tenantId) return;
    setLoading(true);
    fetchBOSchema(boId, tenantId)
      .then((s) => setSchema(s))
      .catch(() => setSchema(null))
      .finally(() => setLoading(false));
  }, [boId, tenantId]);

  const orderedFields = useMemo(() => {
    if (!schema) return [];
    return [...schema.fields].sort((a, b) => {
      const oa = fieldLayout?.[a.name]?.order ?? 999;
      const ob = fieldLayout?.[b.name]?.order ?? 999;
      return oa - ob;
    });
  }, [schema, fieldLayout]);

  const visibleFields = useMemo(
    () => orderedFields.filter((f) => !fieldOverrides?.[f.name]?.hidden),
    [orderedFields, fieldOverrides],
  );

  // Reorder uses PageEditor's single DndContext (palette + canvas). A nested
  // DndContext here fought that parent and made grip-drags unreliable.
  useDndMonitor({
    onDragEnd(event: DragEndEvent) {
      const { active, over } = event;
      if (!over || active.id === over.id) return;
      const data = active.data.current as { kind?: string; componentId?: string } | undefined;
      if (data?.kind !== 'form-field' || data.componentId !== componentId) return;
      const ids = visibleFields.map((f) => fieldSortableId(componentId, f.name));
      const oldIndex = ids.indexOf(String(active.id));
      const newIndex = ids.indexOf(String(over.id));
      if (oldIndex < 0 || newIndex < 0) return;
      const reordered = [...visibleFields];
      const [moved] = reordered.splice(oldIndex, 1);
      reordered.splice(newIndex, 0, moved);
      onReorder(reordered.map((f) => f.name));
    },
  });

  if (loading) {
    return <Box sx={{ p: 2, display: 'flex', justifyContent: 'center' }}><CircularProgress size={20} /></Box>;
  }
  if (!schema || schema.fields.length === 0) {
    return <Typography variant="caption" color="text.secondary" sx={{ p: 1, display: 'block' }}>No fields on this Business Object.</Typography>;
  }

  return (
    <Box onClick={() => onSelectField(null)}>
      <Chip
        size="small"
        variant="outlined"
        label="Design view - drag ⠿ to reorder, drag the right edge to resize, click a field to edit it"
        sx={{ mb: 1, fontSize: 11, height: 22 }}
      />
      <SortableContext items={visibleFields.map((f) => fieldSortableId(componentId, f.name))} strategy={rectSortingStrategy}>
        <Box sx={{ display: 'grid', gridTemplateColumns: `repeat(${GRID_COLS}, 1fr)`, gap: 1 }}>
          {visibleFields.map((field, i) => {
            const layout = fieldLayout?.[field.name];
            const colSpan = Math.min(GRID_COLS, Math.max(1, layout?.colSpan ?? GRID_COLS));
            return (
              <FieldTile
                key={field.id || field.name}
                field={field}
                sortableId={fieldSortableId(componentId, field.name)}
                componentId={componentId}
                colSpan={colSpan}
                order={layout?.order ?? i}
                selected={selectedFieldName === field.name}
                label={fieldOverrides?.[field.name]?.label || field.displayName || field.name}
                required={fieldOverrides?.[field.name]?.required ?? !!field.required}
                style={fieldOverrides?.[field.name]?.style}
                onSelect={() => onSelectField(field.name)}
                onResize={(deltaCols) => onLayoutChange(field.name, { colSpan: Math.min(GRID_COLS, Math.max(1, colSpan + deltaCols)) })}
              />
            );
          })}
        </Box>
      </SortableContext>
      {(() => {
        const hiddenFields = orderedFields.filter((f) => fieldOverrides?.[f.name]?.hidden);
        if (hiddenFields.length === 0) return null;
        return (
          <Box sx={{ mt: 1.5, pt: 1, borderTop: '1px dashed rgba(0,0,0,0.1)' }} onClick={(e) => e.stopPropagation()}>
            <Typography variant="caption" color="text.secondary" sx={{ display: 'block', mb: 0.5 }}>
              Deleted fields - click to bring one back:
            </Typography>
            <Box sx={{ display: 'flex', flexWrap: 'wrap', gap: 0.5 }}>
              {hiddenFields.map((f) => (
                <Chip
                  key={f.id}
                  size="small"
                  variant="outlined"
                  icon={<AddCircleOutlineIcon sx={{ fontSize: 14 }} />}
                  label={fieldOverrides?.[f.name]?.label || f.displayName || f.name}
                  onClick={() => onUnhideField(f.name)}
                  sx={{ fontSize: 11 }}
                />
              ))}
            </Box>
          </Box>
        );
      })()}
    </Box>
  );
};

export default FormFieldsDesigner;
