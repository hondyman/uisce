import type { BOField } from '../reporting/BOFieldsPalette';
import type { DynamicFieldUIConfig } from '../pagedesigner/DynamicPropertyTypes';
import { isToMany as sharedIsToMany, type Cardinality } from '../../types/cardinality';

export interface ContainerStyle {
  backgroundColor?: string;
  paddingTop?: number;
  paddingRight?: number;
  paddingBottom?: number;
  paddingLeft?: number;
  borderWidth?: number;
  borderStyle?: 'none' | 'solid' | 'dashed' | 'dotted';
  borderColor?: string;
}

export type ControlType = 'text' | 'number' | 'switch' | 'date' | 'datetime' | 'select';

export interface PageParameter {
  key: string;
  displayName: string;
  dataType: 'string' | 'number' | 'date' | 'boolean' | 'list';
  defaultValue?: unknown;
}

export function mapFieldToControlType(dataType: string | undefined | null): ControlType {
  const t = (dataType || '').toLowerCase();
  if (['bool', 'boolean'].some((k) => t.includes(k))) return 'switch';
  if (['datetime', 'timestamp'].some((k) => t.includes(k))) return 'datetime';
  if (t.includes('date')) return 'date';
  if (['number', 'int', 'float', 'double', 'decimal', 'numeric', 'currency', 'money'].some((k) => t.includes(k))) return 'number';
  if (['code', 'lookup', 'status', 'type', 'enum'].some((k) => t.includes(k))) return 'select';
  return 'text';
}

// Mirrors backend/internal/metadata/businessobject_service.go's RelationshipResult JSON shape.
export interface RelationshipResult {
  id: string;
  relKey: string;
  direction: 'outbound' | 'inbound' | string;
  relatedObjectName: string;
  relatedBoKey: string;
  targetObjectId: string;
  relationshipType: string;
  cardinality: Cardinality | string; // older/unnormalized values may still appear; see normalizeCardinality
  description?: string;
  joinCondition?: string;
  sourceDriverTable?: string;
  targetDriverTable?: string;
}

export const isToMany = sharedIsToMany;

export interface CanvasWidgetBase {
  id: string;
}

export interface FieldWidget extends CanvasWidgetBase {
  type: 'field';
  fieldKey: string;
  label: string;
  dataType: string;
  controlType: ControlType;
  subtypeKey: string | null;
  isCore?: boolean;
  presentation?: DynamicFieldUIConfig;
  gridSpan?: { xs?: number; md?: number; lg: number };
}

export type ContainerWidgetType = 'section' | 'row' | 'grid' | 'chart' | 'kpi';

export interface ContainerWidget extends CanvasWidgetBase {
  type: ContainerWidgetType;
  /** 'column' (default) stacks children vertically; 'row' places them side-by-side. */
  flow?: 'column' | 'row';
  title?: string;
  children: CanvasWidget[];
  boKey?: string;
  subtypeKey?: string | null;
  presentation?: DynamicFieldUIConfig;
  subscribedParams?: string[];
  gridSpan?: { xs?: number; md?: number; lg: number };
  collapsed?: boolean;
  containerStyle?: ContainerStyle;
}

export interface RelatedObjectWidget extends CanvasWidgetBase {
  type: 'relatedObject';
  title: string;
  relationshipId: string;
  relKey: string;
  targetBoId: string;
  targetBoKey: string;
  cardinality: string;
  displayColumns: string[];
}

export type CanvasWidgetType = 'field' | ContainerWidgetType | 'relatedObject';
export type CanvasWidget = FieldWidget | ContainerWidget | RelatedObjectWidget;

export interface PageTab {
  id: string;
  title: string;
  canvas: CanvasWidget[];
}

export interface PageStudioLayoutSpec {
  version: 3;
  pageKey: string;
  title: string;
  description?: string;
  rootBoKey: string;
  rootBoId: string;
  selectedSubtypeKeys: string[];
  declaredParameters: PageParameter[];
  tabs: PageTab[];
}

// The pre-tabs shape this feature originally shipped with; kept only as a migration input type.
export interface PageStudioLayoutSpecV2 {
  version: 2;
  pageKey: string;
  title: string;
  description?: string;
  rootBoKey: string;
  rootBoId: string;
  selectedSubtypeKeys: string[];
  canvas: CanvasWidget[];
}

let widgetIdCounter = 0;
export function newWidgetId(prefix: string): string {
  widgetIdCounter += 1;
  return `${prefix}_${Date.now().toString(36)}_${widgetIdCounter}`;
}

export function newTab(title = 'New Tab'): PageTab {
  return { id: newWidgetId('tab'), title, canvas: [] };
}

// Migrates a loaded layout_spec (v2 single-canvas, or already-v3) into the current v3 shape.
// Anything else (missing/unrecognized version) throws so callers can show the existing
// "predates rebuild" unsupported-legacy notice, same as before tabs existed.
export function migrateLayoutSpecToV3(
  input: PageStudioLayoutSpecV2 | PageStudioLayoutSpec
): PageStudioLayoutSpec {
  if (input.version === 3) {
    return { ...input, declaredParameters: input.declaredParameters ?? [] };
  }
  if ((input as PageStudioLayoutSpecV2).version === 2) {
    const v2 = input as PageStudioLayoutSpecV2;
    return {
      version: 3,
      pageKey: v2.pageKey,
      title: v2.title,
      description: v2.description,
      rootBoKey: v2.rootBoKey,
      rootBoId: v2.rootBoId,
      selectedSubtypeKeys: v2.selectedSubtypeKeys,
      declaredParameters: [],
      tabs: [{ id: newWidgetId('tab'), title: 'Details', canvas: migrateCanvasWidgets(v2.canvas) }],
    };
  }
  throw new Error(`Unsupported page layout_spec version: ${(input as any)?.version}`);
}

function migrateCanvasWidgets(widgets: CanvasWidget[]): CanvasWidget[] {
  return widgets.map((w) => {
    const base = {
      ...w,
      ...(('gridSpan' in w && (w as any).gridSpan) ? {} : { gridSpan: { xs: 12, md: 6, lg: 6 } }),
      ...(w.type === 'row' ? { flow: 'row' as const } : w.type === 'section' ? { flow: 'column' as const } : {}),
    };
    if ('children' in w && Array.isArray((w as any).children)) {
      return { ...base, children: migrateCanvasWidgets((w as any).children) };
    }
    return base;
  });
}

export function addTab(tabs: PageTab[], title?: string): PageTab[] {
  return [...tabs, newTab(title)];
}

export function renameTab(tabs: PageTab[], tabId: string, title: string): PageTab[] {
  return tabs.map((t) => (t.id === tabId ? { ...t, title } : t));
}

export function removeTab(tabs: PageTab[], tabId: string): PageTab[] {
  if (tabs.length <= 1) return tabs;
  return tabs.filter((t) => t.id !== tabId);
}

export function moveTab(tabs: PageTab[], index: number, direction: -1 | 1): PageTab[] {
  const target = index + direction;
  if (target < 0 || target >= tabs.length) return tabs;
  const next = [...tabs];
  [next[index], next[target]] = [next[target], next[index]];
  return next;
}

export function fieldToWidget(field: BOField): FieldWidget {
  return {
    id: newWidgetId('field'),
    type: 'field',
    fieldKey: field.technicalName || field.name,
    label: field.label || field.name,
    dataType: field.dataType || field.type || 'string',
    controlType: mapFieldToControlType(field.dataType || field.type),
    subtypeKey: field._subtypeKey ?? null,
    isCore: field.isCore,
  };
}

export function newContainerWidget(type: ContainerWidgetType, title?: string): ContainerWidget {
  return {
    id: newWidgetId(type),
    type,
    title: title || defaultTitleFor(type),
    children: [],
  };
}

export function newRelatedObjectWidget(
  rel: Pick<RelationshipResult, 'id' | 'relKey' | 'cardinality'> & { relatedObjectName?: string },
  targetBoKey: string,
  targetBoId: string,
  displayColumns: string[]
): RelatedObjectWidget {
  return {
    id: newWidgetId('relatedObject'),
    type: 'relatedObject',
    title: rel.relatedObjectName || targetBoKey,
    relationshipId: rel.id,
    relKey: rel.relKey,
    targetBoId,
    targetBoKey,
    cardinality: rel.cardinality,
    displayColumns,
  };
}

function defaultTitleFor(type: ContainerWidgetType): string {
  switch (type) {
    case 'section': return 'New Section';
    case 'row': return 'New Row';
    case 'grid': return 'Data Grid';
    case 'chart': return 'Analytics Chart';
    case 'kpi': return 'KPI Tile';
    default: return 'Widget';
  }
}

function isContainer(widget: CanvasWidget): widget is ContainerWidget {
  return widget.type !== 'field' && widget.type !== 'relatedObject';
}

// path is a list of indices identifying a widget by walking `.children` from the root `canvas` array.
export function findContainerAndIndex(
  canvas: CanvasWidget[],
  path: number[]
): { siblings: CanvasWidget[]; index: number } | null {
  if (path.length === 0) return null;
  let siblings = canvas;
  for (let depth = 0; depth < path.length - 1; depth++) {
    const idx = path[depth];
    const widget = siblings[idx];
    if (!widget || !isContainer(widget)) return null;
    siblings = widget.children;
  }
  const index = path[path.length - 1];
  if (index < 0 || index >= siblings.length) return null;
  return { siblings, index };
}

function cloneTree(canvas: CanvasWidget[]): CanvasWidget[] {
  return canvas.map((w) => (isContainer(w) ? { ...w, children: cloneTree(w.children) } : { ...w }));
}

export function addFieldToContainer(
  canvas: CanvasWidget[],
  containerId: string | null,
  field: BOField
): CanvasWidget[] {
  const widget = fieldToWidget(field);
  const next = cloneTree(canvas);
  if (!containerId) {
    next.push(widget);
    return next;
  }
  const target = findContainerById(next, containerId);
  if (target) {
    target.children.push(widget);
  } else {
    next.push(widget);
  }
  return next;
}

export function addWidget(
  canvas: CanvasWidget[],
  containerId: string | null,
  widgetType: ContainerWidgetType
): CanvasWidget[] {
  const next = cloneTree(canvas);
  const widget: CanvasWidget = newContainerWidget(widgetType);
  if (!containerId) {
    next.push(widget);
    return next;
  }
  const target = findContainerById(next, containerId);
  if (target) {
    target.children.push(widget);
  } else {
    next.push(widget);
  }
  return next;
}

export function addRelatedObjectWidget(
  canvas: CanvasWidget[],
  containerId: string | null,
  widget: RelatedObjectWidget
): CanvasWidget[] {
  const next = cloneTree(canvas);
  if (!containerId) {
    next.push(widget);
    return next;
  }
  const target = findContainerById(next, containerId);
  if (target) {
    target.children.push(widget);
  } else {
    next.push(widget);
  }
  return next;
}

function findContainerById(canvas: CanvasWidget[], id: string): ContainerWidget | null {
  for (const widget of canvas) {
    if (widget.id === id && isContainer(widget)) return widget;
    if (isContainer(widget)) {
      const found = findContainerById(widget.children, id);
      if (found) return found;
    }
  }
  return null;
}

export function moveItem(canvas: CanvasWidget[], path: number[], direction: -1 | 1): CanvasWidget[] {
  const next = cloneTree(canvas);
  const located = findContainerAndIndex(next, path);
  if (!located) return canvas;
  const { siblings, index } = located;
  const target = index + direction;
  if (target < 0 || target >= siblings.length) return canvas;
  [siblings[index], siblings[target]] = [siblings[target], siblings[index]];
  return next;
}

export function removeItem(canvas: CanvasWidget[], path: number[]): CanvasWidget[] {
  const next = cloneTree(canvas);
  const located = findContainerAndIndex(next, path);
  if (!located) return canvas;
  const { siblings, index } = located;
  siblings.splice(index, 1);
  return next;
}

export function findWidgetByPath(
  canvas: CanvasWidget[],
  path: number[]
): CanvasWidget | null {
  if (path.length === 0) return null;
  let current: CanvasWidget[] = canvas;
  for (let depth = 0; depth < path.length - 1; depth++) {
    const idx = path[depth];
    const widget = current[idx];
    if (!widget || !isContainer(widget)) return null;
    current = widget.children;
  }
  const lastIdx = path[path.length - 1];
  if (lastIdx < 0 || lastIdx >= current.length) return null;
  return current[lastIdx];
}

export function updateWidgetByPath(
  canvas: CanvasWidget[],
  path: number[],
  updater: (widget: CanvasWidget) => CanvasWidget
): CanvasWidget[] {
  const next = cloneTree(canvas);
  const located = findContainerAndIndex(next, path);
  if (!located) return canvas;
  const { siblings, index } = located;
  siblings[index] = updater(siblings[index]);
  return next;
}
