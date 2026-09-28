import React, { memo, useCallback, useMemo } from 'react';
import ReactFlow, {
  Background, Connection, Controls, Edge as RFEdge, EdgeChange, Handle, MarkerType, MiniMap, Node as RFNode, NodeChange, NodeProps, Position,
  applyNodeChanges,
} from 'reactflow';
import 'reactflow/dist/style.css';
import {
  Alert, Box, Chip, Divider, List, ListItemButton, ListItemIcon, ListItemText, ListSubheader, Paper, Stack, Tooltip, Typography,
  alpha, useMediaQuery, useTheme,
} from '@mui/material';
import ErrorOutlineIcon from '@mui/icons-material/ErrorOutline';
import type { Action, Binding, ChipColor, ConditionNode, TextSpec } from './appModel';
import { getPath, resolve, text, type Scope } from './bindings';
import { useCondition } from './conditions';
import { useAppRuntime, type QueryState } from './AppRuntime';
import { PageIcon } from './icons';

/**
 * A graph editor (a pipeline's steps, a workflow, a lineage): nodes and edges
 * held in a page variable, drawn with their look from bindings, moved,
 * connected and deleted in place, with an optional palette of things to add.
 * The widget edits the graph's shape; what a new node *is* (its id, its
 * default settings, where it chains) is the domain's - onAdd runs actions
 * (usually an operation that returns the next graph) with {{item}},
 * {{graph}} and {{selected}}.
 */

type GraphNode = Record<string, unknown> & { id: string; position?: { x: number; y: number } };
type Graph = Record<string, unknown>;

export interface CanvasCategory {
  color?: ChipColor;
  /** Whether nodes of this category take an input / give an output (default both). */
  inputs?: boolean;
  outputs?: boolean;
}

export interface CanvasProps {
  /** The page variable holding the graph (an object with a node list and an edge list). */
  variable: string;
  nodesPath?: string;
  edgesPath?: string;
  /** Edge fields naming the two ends (default from / to). */
  edgeFrom?: string;
  edgeTo?: string;
  /** The page variable holding the selected node's id. */
  selectedVariable?: string;
  /** How a node looks; templates see {{node}} and {{extra}} (nodeData for this node). */
  node: {
    title: TextSpec;
    subtitle?: TextSpec;
    /** Shown when the subtitle is empty (e.g. "Click to configure"). */
    placeholder?: TextSpec;
    icon?: Binding;
    category?: Binding;
    /** Problems: a message or a list of them - a red border and a tooltip. */
    errors?: Binding;
    /** Chips under the subtitle: [{label, color, variant}] (e.g. rows in / out from a preview). */
    chips?: Binding;
    width?: number;
  };
  /** Category -> colour and handles. */
  categories?: Record<string, CanvasCategory>;
  /** Extra data per node id (validation issues, preview statistics). */
  nodeData?: Binding;
  /** Edges animate while this holds (e.g. a preview has run). */
  animatedWhen?: ConditionNode;
  /** A node takes one input: connecting a new one replaces the old. */
  singleInput?: boolean;
  /** Things to add, from a query; each click runs onAdd with {{item}}, {{graph}}, {{selected}}. */
  palette?: {
    query: string;
    rowsPath?: string;
    groupBy?: string;
    groups?: { id: string; label: TextSpec }[];
    label: TextSpec;
    description?: TextSpec;
    icon?: Binding;
    category?: Binding;
    disabledWhen?: ConditionNode;
  };
  onAdd?: Action[];
  /** After the graph changes (not while a node is being dragged), with {{graph}}. */
  onChange?: Action[];
  emptyText?: TextSpec;
  emptyHint?: TextSpec;
  /** Pixels, or any CSS height ('100%' fills the region it is in). */
  height?: number | string;
  minimap?: boolean;
}

export const DEFAULT_CANVAS_PROPS: CanvasProps = {
  variable: 'graph', nodesPath: 'nodes', edgesPath: 'edges', selectedVariable: 'selected',
  node: { title: '{{node.label}}', subtitle: '', icon: '', category: '' },
  singleInput: true, height: 520, minimap: true,
};

interface NodeView {
  title: string; subtitle: string; placeholder: string; icon: string; color: string; inputs: boolean; outputs: boolean;
  errors: string[]; chips: { label: string; color?: ChipColor; variant?: 'filled' | 'outlined' }[]; width: number;
}

const StudioNode = memo(({ data, selected }: NodeProps<NodeView>) => {
  const theme = useTheme();
  const color = data.color;
  const bad = data.errors.length > 0;
  return (
    <Box sx={{
      width: data.width, borderRadius: 2, bgcolor: 'background.paper',
      border: `2px solid ${bad ? theme.palette.error.main : selected ? color : alpha(color, 0.35)}`,
      boxShadow: selected ? `0 0 0 3px ${alpha(color, 0.2)}` : 1,
    }}>
      {data.inputs && <Handle type="target" position={Position.Left} />}
      <Stack direction="row" spacing={1} alignItems="center" sx={{ px: 1.25, py: 0.75, bgcolor: alpha(color, 0.1), borderRadius: '6px 6px 0 0' }}>
        {data.icon && <Box sx={{ color, display: 'flex' }}><PageIcon name={data.icon} fontSize="small" /></Box>}
        <Typography variant="subtitle2" noWrap sx={{ flex: 1, fontWeight: 700 }}>{data.title}</Typography>
        {bad && <Tooltip title={data.errors.join(' · ')}><ErrorOutlineIcon fontSize="small" color="error" /></Tooltip>}
      </Stack>
      <Box sx={{ px: 1.25, py: 0.75 }}>
        <Typography variant="caption" color="text.secondary" sx={{ display: 'block' }} noWrap title={data.subtitle}>{data.subtitle || data.placeholder}</Typography>
        {data.chips.length > 0 && (
          <Stack direction="row" spacing={0.5} sx={{ mt: 0.5 }} flexWrap="wrap" useFlexGap>
            {data.chips.map((c, i) => <Chip key={i} size="small" color={c.color ?? 'default'} variant={c.variant ?? 'filled'} label={c.label} />)}
          </Stack>
        )}
      </Box>
      {data.outputs && <Handle type="source" position={Position.Right} />}
    </Box>
  );
});
StudioNode.displayName = 'StudioNode';
const NODE_TYPES = { studio: StudioNode };

/** Where a graph keeps its nodes and edges, and what an edge's ends are called. */
export interface GraphShape { nodesPath: string; edgesPath: string; from: string; to: string }
const listAt = (g: Graph, path: string) => (Array.isArray(getPath(g, path)) ? getPath(g, path) : []) as Record<string, unknown>[];

/** Graph edits, pure: the canvas writes their result back to its variable. */
export const graphEdits = {
  removeNodes(g: Graph, ids: string[], s: GraphShape): Graph {
    const nodes = listAt(g, s.nodesPath).filter((n) => !ids.includes(String(n.id)));
    const edges = listAt(g, s.edgesPath).filter((e) => !ids.includes(String(e[s.from])) && !ids.includes(String(e[s.to])));
    return setPathImmutable(setPathImmutable(g, s.nodesPath, nodes), s.edgesPath, edges);
  },
  removeEdges(g: Graph, ids: string[], s: GraphShape): Graph {
    return setPathImmutable(g, s.edgesPath, listAt(g, s.edgesPath).filter((e) => !ids.includes(`${String(e[s.from])}->${String(e[s.to])}`)));
  },
  /** Connects two nodes; with singleInput the target's old input goes. Self-loops are ignored. */
  connect(g: Graph, source: string, target: string, s: GraphShape, singleInput = false): Graph {
    if (!source || !target || source === target) return g;
    const edges = listAt(g, s.edgesPath);
    const kept = singleInput ? edges.filter((e) => String(e[s.to]) !== target) : edges.filter((e) => !(String(e[s.from]) === source && String(e[s.to]) === target));
    return setPathImmutable(g, s.edgesPath, [...kept, { [s.from]: source, [s.to]: target }]);
  },
  move(g: Graph, positions: Map<string, { x: number; y: number }>, s: GraphShape): Graph {
    return setPathImmutable(g, s.nodesPath, listAt(g, s.nodesPath).map((n) => ({ ...n, position: positions.get(String(n.id)) ?? n.position })));
  },
};

const asList = (v: unknown): string[] => (Array.isArray(v) ? v.map(String).filter(Boolean) : v === undefined || v === null || v === '' ? [] : [String(v)]);

function PaletteItem({ p, item, scope, onAdd, compact, colorOf }: {
  p: NonNullable<CanvasProps['palette']>; item: unknown; scope: Scope; onAdd: (item: unknown) => void; compact: boolean; colorOf: (category: string) => string;
}) {
  const s: Scope = { ...scope, item };
  const disabled = useCondition(p.disabledWhen, s, false) && !!p.disabledWhen;
  const label = text(p.label, s);
  const description = p.description ? text(p.description, s) : '';
  const icon = String(resolve(p.icon ?? '', s) ?? '');
  const category = String(resolve(p.category ?? '', s) ?? '');
  return (
    <Tooltip placement="right" title={`${compact ? `${label}: ` : ''}${description}`}>
      <span>
        <ListItemButton disabled={disabled} onClick={() => onAdd(item)} aria-label={label}>
          <ListItemIcon sx={{ minWidth: 32, color: colorOf(category) }}>{icon ? <PageIcon name={icon} fontSize="small" /> : null}</ListItemIcon>
          {!compact && <ListItemText primary={label} />}
        </ListItemButton>
      </span>
    </Tooltip>
  );
}

export function Canvas({ p, scope }: { p: CanvasProps; scope: Scope }) {
  const theme = useTheme();
  const { setVariable, runActions } = useAppRuntime();
  const compact = useMediaQuery(theme.breakpoints.down('lg'));
  const nodesPath = p.nodesPath || 'nodes';
  const edgesPath = p.edgesPath || 'edges';
  const from = p.edgeFrom || 'from';
  const to = p.edgeTo || 'to';
  const vars = (scope.vars ?? {}) as Record<string, unknown>;
  const graph = (vars[p.variable] && typeof vars[p.variable] === 'object' ? vars[p.variable] : {}) as Graph;
  const nodes = (Array.isArray(getPath(graph, nodesPath)) ? getPath(graph, nodesPath) : []) as GraphNode[];
  const edges = (Array.isArray(getPath(graph, edgesPath)) ? getPath(graph, edgesPath) : []) as Record<string, unknown>[];
  const selected = p.selectedVariable ? (vars[p.selectedVariable] as string | null | undefined) ?? null : null;
  const extra = (p.nodeData !== undefined ? resolve(p.nodeData, scope) : undefined) as Record<string, unknown> | undefined;
  const animated = useCondition(p.animatedWhen, scope, false) && !!p.animatedWhen;

  const colorOf = useCallback((category: string) => {
    const c = p.categories?.[category]?.color ?? 'primary';
    return c === 'default' ? theme.palette.text.secondary : theme.palette[c].main;
  }, [p.categories, theme]);

  const rfNodes: RFNode<NodeView>[] = nodes.map((n, i) => {
    const s: Scope = { ...scope, node: n, extra: extra?.[n.id] };
    const category = String(resolve(p.node.category ?? '', s) ?? '');
    const cat = p.categories?.[category];
    const chips = resolve(p.node.chips ?? [], s);
    return {
      id: n.id, type: 'studio', selected: n.id === selected,
      position: n.position ?? { x: 60 + i * 280, y: 120 },
      data: {
        title: text(p.node.title, s), subtitle: p.node.subtitle ? text(p.node.subtitle, s) : '',
        placeholder: p.node.placeholder ? text(p.node.placeholder, s) : '',
        icon: String(resolve(p.node.icon ?? '', s) ?? ''), color: colorOf(category),
        inputs: cat?.inputs !== false, outputs: cat?.outputs !== false,
        errors: asList(p.node.errors !== undefined ? resolve(p.node.errors, s) : undefined),
        chips: Array.isArray(chips) ? chips as NodeView['chips'] : [], width: p.node.width ?? 230,
      },
    };
  });
  const edgeId = (e: Record<string, unknown>) => `${String(e[from])}->${String(e[to])}`;
  const rfEdges: RFEdge[] = edges.map((e) => ({
    id: edgeId(e), source: String(e[from]), target: String(e[to]), markerEnd: { type: MarkerType.ArrowClosed }, animated,
  }));

  // Every edit writes the whole graph back; onChange runs once the edit is done.
  const commit = (next: Graph, done = true) => {
    setVariable(p.variable, next);
    if (done && p.onChange?.length) void runActions(p.onChange, { graph: next });
  };
  const shape: GraphShape = { nodesPath, edgesPath, from, to };
  const select = (id: string | null) => { if (p.selectedVariable) setVariable(p.selectedVariable, id); };

  const onNodesChange = (changes: NodeChange[]) => {
    const removed = changes.filter((c) => c.type === 'remove').map((c) => (c as { id: string }).id);
    if (removed.length) {
      commit(graphEdits.removeNodes(graph, removed, shape));
      if (selected && removed.includes(selected)) select(null);
      return;
    }
    const moves = changes.filter((c) => c.type === 'position');
    if (moves.length) {
      const pos = new Map(applyNodeChanges(changes, rfNodes).map((n) => [n.id, n.position]));
      const done = moves.some((c) => !(c as { dragging?: boolean }).dragging);
      commit(graphEdits.move(graph, pos, shape), done);
    }
  };
  const onEdgesChange = (changes: EdgeChange[]) => {
    const removed = changes.filter((c) => c.type === 'remove').map((c) => (c as { id: string }).id);
    if (removed.length) commit(graphEdits.removeEdges(graph, removed, shape));
  };
  const onConnect = (c: Connection) => {
    if (!c.source || !c.target || c.source === c.target) return;
    commit(graphEdits.connect(graph, c.source, c.target, shape, !!p.singleInput));
  };

  const pq = p.palette ? (getPath(scope, `queries.${p.palette.query}`) as QueryState | undefined) : undefined;
  const paletteRows = useMemo(() => {
    const raw = p.palette?.rowsPath ? getPath(pq?.data, p.palette.rowsPath) : pq?.data;
    return Array.isArray(raw) ? raw : [];
  }, [pq?.data, p.palette?.rowsPath]);
  const groups = p.palette?.groupBy
    ? (p.palette.groups ?? Array.from(new Set(paletteRows.map((r) => String(getPath(r, p.palette!.groupBy!) ?? '')))).map((id) => ({ id, label: id })))
    : [{ id: '', label: '' }];
  const add = (item: unknown) => void runActions(p.onAdd, { item, graph, selected });

  return (
    <Paper variant="outlined" sx={{ display: 'flex', height: p.height ?? 520, minHeight: 320, overflow: 'hidden' }}>
      {p.palette && (
        <Box sx={{ width: compact ? 56 : 230, flexShrink: 0, borderRight: 1, borderColor: 'divider', overflowY: 'auto' }}>
          {groups.map((g) => (
            <List key={g.id || 'all'} dense
              subheader={g.label ? (compact ? <Divider /> : <ListSubheader>{text(g.label, scope)}</ListSubheader>) : undefined}>
              {paletteRows.filter((r) => !p.palette!.groupBy || String(getPath(r, p.palette!.groupBy) ?? '') === g.id).map((r, i) => (
                <PaletteItem key={String(getPath(r, 'id') ?? getPath(r, 'type') ?? i)} p={p.palette!} item={r} scope={scope} onAdd={add} compact={compact} colorOf={colorOf} />
              ))}
            </List>
          ))}
          {!!pq?.error && <Alert severity="error" sx={{ m: 1 }}>Could not load the palette.</Alert>}
        </Box>
      )}
      <Box sx={{ flex: 1, position: 'relative', minWidth: 240 }}>
        {nodes.length === 0 && (p.emptyText || p.emptyHint) && (
          <Stack spacing={1} alignItems="center" sx={{ position: 'absolute', inset: 0, justifyContent: 'center', zIndex: 1, pointerEvents: 'none', px: 2, textAlign: 'center' }}>
            {p.emptyText && <Typography variant="h6" color="text.secondary">{text(p.emptyText, scope)}</Typography>}
            {p.emptyHint && <Typography color="text.secondary">{text(p.emptyHint, scope)}</Typography>}
          </Stack>
        )}
        <ReactFlow nodes={rfNodes} edges={rfEdges} nodeTypes={NODE_TYPES}
          onNodesChange={onNodesChange} onEdgesChange={onEdgesChange} onConnect={onConnect}
          onNodeClick={(_, n) => select(n.id)} onPaneClick={() => select(null)}
          fitView deleteKeyCode={['Backspace', 'Delete']} proOptions={{ hideAttribution: true }}>
          <Background /><Controls />{p.minimap !== false && !compact && <MiniMap pannable zoomable />}
        </ReactFlow>
      </Box>
    </Paper>
  );
}

/** A copy of obj with the value at a dotted path replaced. */
function setPathImmutable(obj: Graph, path: string, value: unknown): Graph {
  const [head, ...rest] = path.split('.');
  if (rest.length === 0) return { ...obj, [head]: value };
  const child = (obj[head] && typeof obj[head] === 'object' ? obj[head] : {}) as Graph;
  return { ...obj, [head]: setPathImmutable(child, rest.join('.'), value) };
}
