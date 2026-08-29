import React, { useRef, useCallback } from 'react';
import { Box, Paper, Stack, Typography, IconButton, TextField, Switch, MenuItem, Select, Grid } from '@mui/material';
import DeleteOutlineIcon from '@mui/icons-material/DeleteOutline';
import DragIndicatorIcon from '@mui/icons-material/DragIndicator';
import TableChartIcon from '@mui/icons-material/TableChart';
import BarChartIcon from '@mui/icons-material/BarChart';
import SpeedIcon from '@mui/icons-material/Speed';
import LinkIcon from '@mui/icons-material/Link';
import { ResizableBox } from 'react-resizable';
import 'react-resizable/css/styles.css';
import {
  DndContext,
  DragEndEvent,
  PointerSensor,
  useSensor,
  useSensors,
} from '@dnd-kit/core';
import {
  SortableContext,
  verticalListSortingStrategy,
  horizontalListSortingStrategy,
  useSortable,
} from '@dnd-kit/sortable';
import { CSS } from '@dnd-kit/utilities';
import type { BOField } from '../reporting/BOFieldsPalette';
import { CoreFieldBadge } from './CoreFieldBadge';
import {
  addFieldToContainer,
  addWidget,
  removeItem,
  type CanvasWidget,
  type ContainerWidgetType,
  type FieldWidget,
  type RelationshipResult,
} from './pageStudioTypes';

export interface PageStudioCanvasProps {
  canvas: CanvasWidget[];
  onChange: (next: CanvasWidget[]) => void;
  onAddRelatedObject?: (containerId: string | null, relationship: RelationshipResult) => void;
  selectedWidgetPath: number[] | null;
  onSelectWidget: (path: number[] | null) => void;
}

interface DragPayload {
  type: 'bofield' | 'bofield_batch' | 'widget' | 'relatedobject';
  field?: BOField;
  fields?: BOField[];
  widgetType?: ContainerWidgetType;
  relationship?: RelationshipResult;
}

function readDragPayload(e: React.DragEvent): DragPayload | null {
  try {
    const raw = e.dataTransfer.getData('application/json');
    if (!raw) return null;
    return JSON.parse(raw);
  } catch {
    return null;
  }
}

function applyDrop(canvas: CanvasWidget[], containerId: string | null, payload: DragPayload): CanvasWidget[] {
  if (payload.type === 'bofield' && payload.field) {
    return addFieldToContainer(canvas, containerId, payload.field);
  }
  if (payload.type === 'bofield_batch' && payload.fields) {
    return payload.fields.reduce((acc, f) => addFieldToContainer(acc, containerId, f), canvas);
  }
  if (payload.type === 'widget' && payload.widgetType) {
    return addWidget(canvas, containerId, payload.widgetType);
  }
  return canvas;
}

const controlPreview = (widget: FieldWidget) => {
  switch (widget.controlType) {
    case 'switch':
      return <Switch size="small" disabled />;
    case 'date':
      return <TextField size="small" type="date" disabled sx={{ width: 160 }} />;
    case 'datetime':
      return <TextField size="small" type="datetime-local" disabled sx={{ width: 200 }} />;
    case 'number':
      return <TextField size="small" type="number" disabled sx={{ width: 140 }} />;
    case 'select':
      return (
        <Select size="small" disabled value="" displayEmpty sx={{ width: 180 }}>
          <MenuItem value="">Select…</MenuItem>
        </Select>
      );
    default:
      return <TextField size="small" disabled sx={{ width: 220 }} />;
  }
};

const containerIcon = (type: ContainerWidgetType) => {
  if (type === 'grid') return <TableChartIcon fontSize="small" />;
  if (type === 'chart') return <BarChartIcon fontSize="small" />;
  if (type === 'kpi') return <SpeedIcon fontSize="small" />;
  return null;
};

const DEFAULT_GRID_SPAN = 6;

function getGridSpan(widget: CanvasWidget): { xs: number; md: number; lg: number } {
  if ('gridSpan' in widget && (widget as any).gridSpan) {
    return (widget as any).gridSpan;
  }
  return { xs: 12, md: DEFAULT_GRID_SPAN, lg: DEFAULT_GRID_SPAN };
}

function updateGridSpan(widget: CanvasWidget, span: number): CanvasWidget {
  return { ...widget, gridSpan: { xs: 12, md: span, lg: span } };
}

function findWidgetByPath(canvas: CanvasWidget[], path: number[]): CanvasWidget | null {
  if (path.length === 0) return null;
  let current: CanvasWidget[] = canvas;
  for (let depth = 0; depth < path.length - 1; depth++) {
    const idx = path[depth];
    const widget = current[idx];
    if (!widget || !('children' in widget)) return null;
    current = (widget as any).children;
  }
  const lastIdx = path[path.length - 1];
  if (lastIdx < 0 || lastIdx >= current.length) return null;
  return current[lastIdx];
}

function updateWidgetByPath(
  canvas: CanvasWidget[],
  path: number[],
  updater: (widget: CanvasWidget) => CanvasWidget
): CanvasWidget[] {
  const next = deepClone(canvas);
  let current: CanvasWidget[] = next;
  for (let depth = 0; depth < path.length - 1; depth++) {
    const idx = path[depth];
    const widget = current[idx];
    if (!widget || !('children' in widget)) return canvas;
    current = (widget as any).children;
  }
  const lastIdx = path[path.length - 1];
  if (lastIdx < 0 || lastIdx >= current.length) return canvas;
  current[lastIdx] = updater(current[lastIdx]);
  return next;
}

function deepClone<T>(obj: T): T {
  return JSON.parse(JSON.stringify(obj));
}

function findContainerPath(canvas: CanvasWidget[], widgetId: string): number[] | null {
  for (let i = 0; i < canvas.length; i++) {
    const w = canvas[i];
    if (w.id === widgetId) return [i];
    if ('children' in w) {
      const childPath = findContainerPath((w as any).children, widgetId);
      if (childPath) return [i, ...childPath];
    }
  }
  return null;
}

function reorderWithinContainer(
  canvas: CanvasWidget[],
  containerPath: number[],
  activeId: string,
  overId: string
): CanvasWidget[] {
  const container = findWidgetByPath(canvas, containerPath);
  if (!container || !('children' in container)) return canvas;
  const children: CanvasWidget[] = (container as any).children;
  const oldIndex = children.findIndex((c) => c.id === activeId);
  const newIndex = children.findIndex((c) => c.id === overId);
  if (oldIndex === -1 || newIndex === -1) return canvas;
  const reordered = [...children];
  const [moved] = reordered.splice(oldIndex, 1);
  reordered.splice(newIndex, 0, moved);
  return updateWidgetByPath(canvas, containerPath, () => ({ ...container, children: reordered }));
}

export const PageStudioCanvas: React.FC<PageStudioCanvasProps> = ({
  canvas,
  onChange,
  onAddRelatedObject,
  selectedWidgetPath,
  onSelectWidget,
}) => {
  const containerRef = useRef<HTMLDivElement>(null);

  const sensors = useSensors(
    useSensor(PointerSensor, { activationConstraint: { distance: 5 } })
  );

  const handleDrop = (containerId: string | null) => (e: React.DragEvent) => {
    e.preventDefault();
    e.stopPropagation();
    const payload = readDragPayload(e);
    if (!payload) return;
    if (payload.type === 'relatedobject' && payload.relationship) {
      onAddRelatedObject?.(containerId, payload.relationship);
      return;
    }
    onChange(applyDrop(canvas, containerId, payload));
  };

  const handleDndDragEnd = useCallback(
    (event: DragEndEvent) => {
      const { active, over } = event;
      if (!over || active.id === over.id) return;
      const activeId = String(active.id);
      const overId = String(over.id);
      const activePath = findContainerPath(canvas, activeId);
      const overPath = findContainerPath(canvas, overId);
      if (!activePath || !overPath) return;
      if (activePath.length !== overPath.length) return;
      if (!activePath.slice(0, -1).every((v, i) => v === overPath[i])) return;
      onChange(reorderWithinContainer(canvas, activePath.slice(0, -1), activeId, overId));
    },
    [canvas, onChange]
  );

  const pathMatches = (a: number[] | null, b: number[] | null): boolean => {
    if (!a || !b) return false;
    if (a.length !== b.length) return false;
    return a.every((v, i) => v === b[i]);
  };

  const selectedSx = (isSelected: boolean) => ({
    border: isSelected ? '2px solid #00D4FF' : '1px solid',
    borderColor: isSelected ? '#00D4FF' : 'rgba(255,255,255,0.12)',
    bgcolor: isSelected ? 'rgba(0,212,255,0.04)' : undefined,
    boxShadow: isSelected ? '0 0 0 1px rgba(0,212,255,0.2)' : undefined,
  });

  const remove = (path: number[]) => () => onChange(removeItem(canvas, path));

  const handleWidgetResize = (path: number[], newSpan: number) => {
    if (!path.length) return;
    const widget = findWidgetByPath(canvas, path);
    if (!widget) return;
    const next = updateWidgetByPath(canvas, path, () => updateGridSpan(widget, newSpan));
    onChange(next);
  };

  const renderSortableWidget = (widget: CanvasWidget, widgetPath: number[]): React.ReactNode => {
    const {
      attributes,
      listeners,
      setNodeRef,
      transform,
      transition,
      isDragging,
    } = useSortable({ id: widget.id });

    const isSelected = pathMatches(selectedWidgetPath, widgetPath);
    const span = getGridSpan(widget);
    const spanLg = span.lg;
    const removeHandler = remove(widgetPath);

    const dragStyle = {
      transform: CSS.Transform.toString(transform),
      transition,
      opacity: isDragging ? 0.4 : 1,
      zIndex: isDragging ? 9999 : 1,
    };

    const paperSx = {
      ...selectedSx(isSelected),
      cursor: 'grab',
      transition: 'border-color 0.15s, box-shadow 0.15s',
      '&:hover .resize-handle': { opacity: 1 },
    };

    const resizeHandle = (
      <Box
        className="resize-handle"
        onMouseDown={(e) => e.stopPropagation()}
        onClick={(e) => e.stopPropagation()}
        sx={{
          position: 'absolute',
          right: 0,
          top: 0,
          bottom: 0,
          width: 8,
          cursor: 'ew-resize',
          opacity: 0,
          bgcolor: '#00D4FF',
          borderRadius: '0 2px 2px 0',
          transition: 'opacity 0.15s',
          zIndex: 1,
          '&:hover': { bgcolor: '#00B4E6' },
        }}
      />
    );

    const deleteButton = (
      <IconButton size="small" onClick={(e) => { e.stopPropagation(); removeHandler(); }} aria-label="remove">
        <DeleteOutlineIcon fontSize="inherit" />
      </IconButton>
    );

    const widgetContent = (
      <Box ref={setNodeRef} style={dragStyle}>
        <ResizableBox
          width={200}
          height={0}
          axis="x"
          resizeHandles={['e']}
          onResize={(_e, _d, delta) => {
            const newSpan = Math.max(1, Math.min(12, spanLg + Math.round(delta.width / 20)));
            handleWidgetResize(widgetPath, newSpan);
          }}
          minConstraints={[60, 0]}
          maxConstraints={[600, 0]}
          enable={{ right: true, left: false, top: false, bottom: false }}
        >
          <Paper
            variant="outlined"
            onClick={() => onSelectWidget(widgetPath)}
            sx={{ p: 1, display: 'flex', alignItems: 'center', justifyContent: 'space-between', gap: 1, ...paperSx }}
          >
            <Stack sx={{ minWidth: 0, flex: 1 }} direction="row" alignItems="center" gap={0.5}>
              <Box {...attributes} {...listeners} sx={{ cursor: 'grab', display: 'flex', alignItems: 'center', color: '#64748B', flexShrink: 0 }}>
                <DragIndicatorIcon fontSize="small" />
              </Box>
              <Typography variant="caption" fontWeight={700} noWrap>{(widget as FieldWidget).label}</Typography>
              <CoreFieldBadge isCore={(widget as FieldWidget).isCore} size={12} />
              {controlPreview(widget as FieldWidget)}
            </Stack>
            <Stack direction="row">
              {deleteButton}
              {isSelected && resizeHandle}
            </Stack>
          </Paper>
        </ResizableBox>
      </Box>
    );

    if (widget.type === 'field') {
      return (
        <Grid key={widget.id} size={{ xs: span.xs, md: span.md, lg: spanLg }} sx={{ position: 'relative' }}>
          {widgetContent}
        </Grid>
      );
    }

    if (widget.type === 'relatedObject') {
      return (
        <Grid key={widget.id} size={{ xs: span.xs, md: span.md, lg: spanLg }} sx={{ position: 'relative' }}>
          <ResizableBox
            width={200}
            height={0}
            axis="x"
            resizeHandles={['e']}
            onResize={(_e, _d, delta) => {
              const newSpan = Math.max(1, Math.min(12, spanLg + Math.round(delta.width / 20)));
              handleWidgetResize(widgetPath, newSpan);
            }}
            minConstraints={[60, 0]}
            maxConstraints={[600, 0]}
            enable={{ right: true, left: false, top: false, bottom: false }}
          >
            <Paper
              variant="outlined"
              onClick={() => onSelectWidget(widgetPath)}
              sx={{ p: 1.5, display: 'flex', alignItems: 'center', justifyContent: 'space-between', gap: 1, ...paperSx }}
            >
              <Stack direction="row" alignItems="center" gap={1}>
                <Box {...attributes} {...listeners} sx={{ cursor: 'grab', display: 'flex', alignItems: 'center', color: '#64748B', flexShrink: 0 }}>
                  <DragIndicatorIcon fontSize="small" />
                </Box>
                <LinkIcon fontSize="small" />
                <Typography variant="body2" fontWeight={700}>{(widget as any).title}</Typography>
                <CoreFieldBadge isCore={false} size={12} />
                <Typography variant="caption" color="text.secondary">
                  {(widget as any).cardinality} → {(widget as any).targetBoKey}
                </Typography>
              </Stack>
              <Stack direction="row">
                {deleteButton}
                {isSelected && resizeHandle}
              </Stack>
            </Paper>
          </ResizableBox>
        </Grid>
      );
    }

    if (widget.type === 'grid' || widget.type === 'chart' || widget.type === 'kpi') {
      return (
        <Grid key={widget.id} size={{ xs: span.xs, md: span.md, lg: spanLg }} sx={{ position: 'relative' }}>
          <ResizableBox
            width={200}
            height={0}
            axis="x"
            resizeHandles={['e']}
            onResize={(_e, _d, delta) => {
              const newSpan = Math.max(1, Math.min(12, spanLg + Math.round(delta.width / 20)));
              handleWidgetResize(widgetPath, newSpan);
            }}
            minConstraints={[60, 0]}
            maxConstraints={[600, 0]}
            enable={{ right: true, left: false, top: false, bottom: false }}
          >
            <Paper
              variant="outlined"
              onClick={() => onSelectWidget(widgetPath)}
              sx={{ p: 1.5, display: 'flex', alignItems: 'center', justifyContent: 'space-between', gap: 1, ...paperSx }}
            >
              <Stack direction="row" alignItems="center" gap={1}>
                <Box {...attributes} {...listeners} sx={{ cursor: 'grab', display: 'flex', alignItems: 'center', color: '#64748B', flexShrink: 0 }}>
                  <DragIndicatorIcon fontSize="small" />
                </Box>
                {containerIcon(widget.type)}
                <Typography variant="body2" fontWeight={700}>{(widget as any).title}</Typography>
                <CoreFieldBadge isCore={true} size={12} />
                <Typography variant="caption" color="text.secondary">
                  {(widget as any).boKey ? `bound to ${(widget as any).boKey}` : 'not yet bound'}
                </Typography>
              </Stack>
              <Stack direction="row">
                {deleteButton}
                {isSelected && resizeHandle}
              </Stack>
            </Paper>
          </ResizableBox>
        </Grid>
      );
    }

    return null;
  };

  const renderContainer = (widget: CanvasWidget, widgetPath: number[]): React.ReactNode => {
    const isSelected = pathMatches(selectedWidgetPath, widgetPath);
    const isRow = (widget as any).flow === 'row' || widget.type === 'row';
    const strategy = isRow ? horizontalListSortingStrategy : verticalListSortingStrategy;
    const children: CanvasWidget[] = (widget as any).children || [];
    const childIds = children.map((c) => c.id);
    const isCollapsed = (widget as any).collapsed;

    return (
      <Grid key={widget.id} size={{ xs: 12 }} sx={{ position: 'relative' }}>
        <Paper
          variant="outlined"
          onClick={() => onSelectWidget(widgetPath)}
          onDragOver={(e) => e.preventDefault()}
          onDrop={handleDrop(widget.id)}
          sx={{ p: 1.5, mb: 1.5, bgcolor: 'rgba(255,255,255,0.02)', ...selectedSx(isSelected) }}
        >
          <Stack direction="row" alignItems="center" justifyContent="space-between" sx={{ mb: 1 }}>
            <Stack direction="row" alignItems="center" gap={0.5}>
              <DragIndicatorIcon fontSize="small" sx={{ color: '#64748B', cursor: 'grab' }} />
              <Typography variant="subtitle2" fontWeight={700}>{(widget as any).title}</Typography>
              <CoreFieldBadge isCore={true} size={12} />
              {(widget as any).collapsed && (
                <Typography variant="caption" color="text.secondary" sx={{ ml: 0.5 }}>(collapsed)</Typography>
              )}
            </Stack>
            <IconButton size="small" onClick={(e) => { e.stopPropagation(); remove(widgetPath)(); }} aria-label="remove">
              <DeleteOutlineIcon fontSize="inherit" />
            </IconButton>
          </Stack>
          {!isCollapsed && children.length > 0 && (
            <Box sx={{ mt: 1 }}>
              <SortableContext items={childIds} strategy={strategy}>
                <Grid container spacing={2} direction={isRow ? 'row' : 'column'}>
                  {children.map((child, idx) => renderSortableWidget(child, [...widgetPath, idx]))}
                </Grid>
              </SortableContext>
            </Box>
          )}
          {!isCollapsed && children.length === 0 && (
            <Typography variant="caption" color="text.secondary" sx={{ p: 1, textAlign: 'center', display: 'block' }}>
              Drop fields or widgets here
            </Typography>
          )}
        </Paper>
      </Grid>
    );
  };

  const renderWidget = (widget: CanvasWidget, widgetPath: number[]): React.ReactNode => {
    if (widget.type === 'field' || widget.type === 'relatedObject' || widget.type === 'grid' || widget.type === 'chart' || widget.type === 'kpi') {
      return renderSortableWidget(widget, widgetPath);
    }
    return renderContainer(widget, widgetPath);
  };

  const rootChildIds = canvas.map((w) => w.id);

  return (
    <DndContext sensors={sensors} onDragEnd={handleDndDragEnd}>
      <Box
        ref={containerRef}
        sx={{ flex: 1, height: '100%', overflowY: 'auto', p: 2 }}
        onDragOver={(e) => e.preventDefault()}
        onDrop={handleDrop(null)}
        onClick={(e) => { if (e.target === e.currentTarget) onSelectWidget(null); }}
        data-testid="page-studio-canvas-root"
      >
        {canvas.length === 0 ? (
          <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'center', height: '100%' }}>
            <Typography variant="body2" color="text.secondary">
              Drag fields from the BO Fields tab or widgets from the Widgets tab onto this canvas.
            </Typography>
          </Box>
        ) : (
          <SortableContext items={rootChildIds} strategy={verticalListSortingStrategy}>
            <Grid container spacing={2}>
              {canvas.map((widget, idx) => renderWidget(widget, [idx]))}
            </Grid>
          </SortableContext>
        )}
      </Box>
    </DndContext>
  );
};

export default PageStudioCanvas;
