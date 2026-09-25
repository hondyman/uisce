import React, { useCallback, useEffect, useMemo, useState } from 'react';
import { useNavigate, useParams } from 'react-router-dom';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import ReactFlow, {
  Background, Connection, Controls, Edge as RFEdge, MiniMap, Node as RFNode, NodeChange, applyNodeChanges,
  EdgeChange, MarkerType,
} from 'reactflow';
import 'reactflow/dist/style.css';
import {
  Alert, Badge, Box, Button, Chip, CircularProgress, Divider, IconButton, LinearProgress, List, ListItemButton,
  ListItemIcon, ListItemText, ListSubheader, Paper, Snackbar, Stack, Tab, Table, TableBody, TableCell, TableHead,
  TableRow, Tabs, TextField, Tooltip, Typography, useMediaQuery, useTheme,
} from '@mui/material';
import ArrowBackIcon from '@mui/icons-material/ArrowBack';
import SaveIcon from '@mui/icons-material/Save';
import PlayArrowIcon from '@mui/icons-material/PlayArrow';
import VisibilityIcon from '@mui/icons-material/Visibility';
import AutoAwesomeIcon from '@mui/icons-material/AutoAwesome';
import CloseIcon from '@mui/icons-material/Close';
import {
  BOSchemaField, Column, NodeConfigs, NodeKind, NodeType, pipelinesApi, PreviewResult, RunRecord, Spec, SpecNode, TargetField, boTarget,
  platformApi,
} from './api';
import { downstreamSink, fieldsIn, newNodeId, SourceFieldLookup } from './fields';
import { categoryColor, NODE_META, PipelineNode, PipelineNodeData } from './PipelineNode';
import { NodeConfigPanel } from './NodeConfigPanel';
import { AssistantPanel } from './AssistantPanel';
import { ScheduleDialog } from './ScheduleDialog';
import { useFillHeight } from './useFillHeight';
import ScheduleIcon from '@mui/icons-material/Schedule';

const nodeTypes = { pipeline: PipelineNode };

const EMPTY_SPEC: Spec = { version: 1, nodes: [], edges: [], error_policy: 'skip_and_log' };

export function defaultConfig(kind: NodeKind): NodeConfigs[NodeKind] {
  switch (kind) {
    case 'file_source': return { uri: '', format: 'csv', has_header: true, columns: [] };
    case 'bo_source': return { bo_key: '', filters: [] };
    case 'validate': return { required: [], unique: [] };
    case 'rule_check': return { rule_ids: [] };
    case 'map': return { fields: [] };
    case 'bo_sink': return { bo_key: '', mode: 'create' };
    case 'staging_sink': return { table: '', source_cd: '', domain: '' };
    case 'file_sink': return { uri: '', format: 'csv' };
  }
}

/** One-line description of what a configured step does. */
export function summarize(n: SpecNode): string {
  const c = n.config as Record<string, any>;
  switch (n.type) {
    case 'file_source': return c.uri ? `${c.uri} · ${(c.columns ?? []).length} columns` : '';
    case 'bo_source': return c.bo_key ? `${c.bo_key}${c.filters?.length ? ` · ${c.filters.length} filter(s)` : ''}` : '';
    case 'validate': return [c.required?.length && `${c.required.length} required`, c.unique?.length && `unique on ${c.unique.join('+')}`].filter(Boolean).join(' · ');
    case 'rule_check': return c.rule_ids?.length ? `${c.rule_ids.length} rule(s)${c.bo_key ? ` of ${c.bo_key}` : ''}` : '';
    case 'map': return c.fields?.length ? `${c.fields.length} field(s) mapped` : '';
    case 'bo_sink': return c.bo_key ? `${c.mode === 'upsert' ? 'update or create' : 'create'} ${c.bo_key}${c.dry_run ? ' (rehearsal)' : ''}` : '';
    case 'staging_sink': return c.table ? `${c.table}${c.source_cd ? ` · ${c.source_cd}/${c.domain}` : ''}` : '';
    case 'file_sink': return c.uri ? `${c.uri} (${c.format})` : '';
  }
  return '';
}

export default function PipelineEditorPage() {
  const { id } = useParams<{ id: string }>();
  const isNew = !id || id === 'new';
  const navigate = useNavigate();
  const qc = useQueryClient();

  const def = useQuery({ queryKey: ['dp', id], enabled: !isNew, queryFn: () => pipelinesApi.get(id!) });
  const palette = useQuery({ queryKey: ['dp-node-types'], queryFn: pipelinesApi.nodeTypes, staleTime: 60_000 });

  const [name, setName] = useState('Untitled pipeline');
  const [spec, setSpec] = useState<Spec>(EMPTY_SPEC);
  const [dirty, setDirty] = useState(false);
  const [selected, setSelected] = useState<string | null>(null);
  const [issues, setIssues] = useState<{ node_id?: string; message: string }[]>([]);
  const [tab, setTab] = useState<'problems' | 'preview' | 'runs'>('problems');
  const [preview, setPreview] = useState<PreviewResult | null>(null);
  const [previewNode, setPreviewNode] = useState<string | null>(null);
  const [toast, setToast] = useState<string | null>(null);
  const [assistantOpen, setAssistantOpen] = useState(false);
  const [scheduleOpen, setScheduleOpen] = useState(false);
  const schedule = useQuery({ queryKey: ['dp-schedule', id], enabled: !isNew, queryFn: () => pipelinesApi.schedule(id!) });
  const theme = useTheme();
  const compact = useMediaQuery(theme.breakpoints.down('lg')); // keep room for the canvas

  useEffect(() => {
    if (def.data) {
      setName(def.data.name);
      setSpec({ ...EMPTY_SPEC, ...def.data.spec, nodes: def.data.spec.nodes ?? [], edges: def.data.spec.edges ?? [] });
      setDirty(false);
    }
  }, [def.data]);

  const update = useCallback((next: Spec) => { setSpec(next); setDirty(true); setPreview(null); }, []);

  // Live validation (debounced).
  useEffect(() => {
    if (spec.nodes.length === 0) { setIssues([]); return; }
    const t = setTimeout(() => {
      pipelinesApi.validate(spec).then(r => setIssues(r.issues)).catch(() => undefined);
    }, 400);
    return () => clearTimeout(t);
  }, [spec]);

  // Field lookup for BO sources: their schema.
  const boKeys = useMemo(() => [...new Set(spec.nodes.filter(n => n.type === 'bo_source' || n.type === 'bo_sink')
    .map(n => (n.config as { bo_key?: string }).bo_key).filter(Boolean) as string[])], [spec.nodes]);
  const schemas = useQuery({
    queryKey: ['dp-bo-schemas', boKeys],
    enabled: boKeys.length > 0,
    queryFn: async (): Promise<Record<string, BOSchemaField[]>> => Object.fromEntries(await Promise.all(boKeys.map(async k => {
      try { return [k, (await platformApi.boSchema(k)).fields ?? []] as const; } catch { return [k, [] as BOSchemaField[]] as const; }
    }))),
  });
  const staging = useQuery({ queryKey: ['dp-staging'], queryFn: pipelinesApi.stagingTables, retry: false });

  const lookup: SourceFieldLookup = useCallback((n: SpecNode) => {
    if (n.type !== 'bo_source') return undefined;
    const f = schemas.data?.[(n.config as { bo_key: string }).bo_key];
    return f?.map(x => ({ name: boTarget(x).name, type: (x.type as Column['type']) ?? 'string' }));
  }, [schemas.data]);

  const targetsFor = useCallback((sink?: SpecNode): { targets?: TargetField[]; label?: string } => {
    if (!sink) return {};
    if (sink.type === 'bo_sink') {
      const k = (sink.config as { bo_key: string }).bo_key;
      return { targets: schemas.data?.[k]?.map(boTarget), label: k };
    }
    if (sink.type === 'staging_sink') {
      const t = (sink.config as { table: string }).table;
      return { targets: staging.data?.find(s => s.table === t)?.columns, label: t };
    }
    return {};
  }, [schemas.data, staging.data]);

  // --- React Flow projection ------------------------------------------------

  const issuesByNode = useMemo(() => {
    const m: Record<string, string[]> = {};
    for (const i of issues) if (i.node_id) (m[i.node_id] ??= []).push(i.message);
    return m;
  }, [issues]);

  const statsByNode = useMemo(() => Object.fromEntries((preview?.summary?.Nodes ?? []).map(s => [s.NodeID, s])), [preview]);
  const typeLabel = useCallback((k: NodeKind) => palette.data?.find(p => p.type === k)?.label ?? k, [palette.data]);

  const rfNodes: RFNode<PipelineNodeData>[] = useMemo(() => spec.nodes.map((n, i) => ({
    id: n.id,
    type: 'pipeline',
    position: n.position ?? { x: 60 + i * 280, y: 120 },
    selected: n.id === selected,
    data: {
      kind: n.type, title: n.label || typeLabel(n.type), summary: summarize(n),
      issues: issuesByNode[n.id] ?? [], stats: statsByNode[n.id],
    },
  })), [spec.nodes, selected, issuesByNode, statsByNode, typeLabel]);

  const rfEdges: RFEdge[] = useMemo(() => spec.edges.map(e => ({
    id: `${e.from}->${e.to}`, source: e.from, target: e.to, markerEnd: { type: MarkerType.ArrowClosed },
    animated: !!preview,
  })), [spec.edges, preview]);

  const onNodesChange = useCallback((changes: NodeChange[]) => {
    const moved = applyNodeChanges(changes, rfNodes);
    const removed = changes.filter(c => c.type === 'remove').map(c => (c as { id: string }).id);
    const pos = new Map(moved.map(n => [n.id, n.position]));
    if (removed.length) {
      update({
        ...spec,
        nodes: spec.nodes.filter(n => !removed.includes(n.id)),
        edges: spec.edges.filter(e => !removed.includes(e.from) && !removed.includes(e.to)),
      });
      return;
    }
    if (changes.some(c => c.type === 'position')) {
      setSpec(s => ({ ...s, nodes: s.nodes.map(n => ({ ...n, position: pos.get(n.id) ?? n.position })) }));
      if (changes.some(c => c.type === 'position' && !(c as { dragging?: boolean }).dragging)) setDirty(true);
    }
  }, [rfNodes, spec, update]);

  const onEdgesChange = useCallback((changes: EdgeChange[]) => {
    const removed = new Set(changes.filter(c => c.type === 'remove').map(c => (c as { id: string }).id));
    if (removed.size) update({ ...spec, edges: spec.edges.filter(e => !removed.has(`${e.from}->${e.to}`)) });
  }, [spec, update]);

  const onConnect = useCallback((c: Connection) => {
    if (!c.source || !c.target || c.source === c.target) return;
    // One input per step: a new connection replaces the old one.
    update({ ...spec, edges: [...spec.edges.filter(e => e.to !== c.target), { from: c.source, to: c.target }] });
  }, [spec, update]);

  const addNode = (t: NodeType) => {
    const sel = spec.nodes.find(n => n.id === selected);
    const newId = newNodeId(t.type, spec);
    const pos = sel?.position ? { x: sel.position.x + 280, y: sel.position.y } : { x: 60 + spec.nodes.length * 60, y: 80 + spec.nodes.length * 40 };
    const node: SpecNode = { id: newId, type: t.type, label: t.label, config: defaultConfig(t.type), position: pos };
    const edges = [...spec.edges];
    // Chain after the selected step when that makes sense.
    if (sel && NODE_META[sel.type].category !== 'destination' && t.category !== 'source') edges.push({ from: sel.id, to: newId });
    update({ ...spec, nodes: [...spec.nodes, node], edges });
    setSelected(newId);
  };

  // --- actions -----------------------------------------------------------------

  const save = useMutation({
    mutationFn: async () => (isNew ? pipelinesApi.create({ name, spec }) : pipelinesApi.update(id!, { name, spec })),
    onSuccess: d => {
      setDirty(false);
      qc.invalidateQueries({ queryKey: ['dp-list'] });
      setToast('Saved');
      if (isNew) navigate(`/data/pipelines/${d.id}`, { replace: true });
    },
    onError: e => setToast(`Save failed: ${(e as Error).message}`),
  });

  const runPreview = useMutation({
    mutationFn: () => pipelinesApi.preview(spec, 100),
    onSuccess: r => {
      setPreview(r); setTab('preview');
      setPreviewNode(null); // all steps: every reject, final rows
    },
    onError: e => { setPreview({ rejects: [], error: (e as Error).message }); setTab('preview'); },
  });

  const [activeRun, setActiveRun] = useState<string | null>(null);
  const runs = useQuery({ queryKey: ['dp-runs', id], enabled: !isNew, queryFn: () => pipelinesApi.runs(id!), refetchInterval: activeRun ? 2000 : false });
  useEffect(() => {
    const r = runs.data?.find(x => x.id === activeRun);
    if (r && r.status !== 'queued' && r.status !== 'running') {
      setToast(r.status === 'failed' ? 'Run failed - see Runs' : `Run finished: ${r.records_out} written, ${r.errors} rejected`);
      setActiveRun(null);
    }
  }, [runs.data, activeRun]);

  const startRun = useMutation({
    mutationFn: async () => { if (dirty) await save.mutateAsync(); return pipelinesApi.startRun(id!); },
    onSuccess: r => { setActiveRun(r.run_id); setTab('runs'); runs.refetch(); },
    onError: e => setToast(`Could not start: ${(e as Error).message}`),
  });

  const selectedNode = spec.nodes.find(n => n.id === selected);
  const valid = issues.length === 0 && spec.nodes.length > 0;

  const [rootRef, fillHeight] = useFillHeight();

  if (!isNew && def.isLoading) return <LinearProgress />;
  if (!isNew && def.isError) return <Alert severity="error" sx={{ m: 2 }}>Could not load this pipeline.</Alert>;

  return (
    // Own themed surface (the shell's canvas is dark whatever the MUI mode),
    // sized to the space below the app header.
    <Box ref={rootRef} sx={{ display: 'flex', flexDirection: 'column', height: fillHeight ?? 'calc(100vh - 64px)', bgcolor: 'background.default', color: 'text.primary' }}>
      {/* Toolbar */}
      <Stack direction="row" alignItems="center" useFlexGap flexWrap="wrap" sx={{ gap: 1, px: 2, py: 1, borderBottom: 1, borderColor: 'divider' }}>
        <IconButton onClick={() => navigate('/data/pipelines')} aria-label="back"><ArrowBackIcon /></IconButton>
        <TextField variant="standard" value={name} onChange={e => { setName(e.target.value); setDirty(true); }}
          inputProps={{ 'aria-label': 'pipeline name', style: { fontSize: 20, fontWeight: 600 } }} sx={{ minWidth: 220, flex: '1 1 260px', maxWidth: 480 }} />
        {dirty && <Chip size="small" label="unsaved" />}
        <Chip size="small" color={valid ? 'success' : spec.nodes.length ? 'warning' : 'default'}
          label={!spec.nodes.length ? 'Empty' : valid ? 'Ready to run' : `${issues.length} problem${issues.length === 1 ? '' : 's'}`}
          onClick={() => setTab('problems')} />
        <Box sx={{ flex: '1 1 0' }} />
        <Button startIcon={<AutoAwesomeIcon />} onClick={() => setAssistantOpen(o => !o)} variant={assistantOpen ? 'contained' : 'outlined'} color="secondary">
          Assistant
        </Button>
        <Tooltip title={valid ? 'Run on the first 100 rows. Nothing is saved or written.' : 'Fix the problems first'}>
          <span>
            <Button startIcon={runPreview.isPending ? <CircularProgress size={16} /> : <VisibilityIcon />} disabled={!valid || runPreview.isPending} onClick={() => runPreview.mutate()}>
              Preview
            </Button>
          </span>
        </Tooltip>
        <Button startIcon={<SaveIcon />} disabled={!dirty || save.isPending} onClick={() => save.mutate()}>Save</Button>
        <Tooltip title={isNew ? 'Save first' : 'Run this pipeline on a schedule'}>
          <span>
            <Button startIcon={<ScheduleIcon />} disabled={isNew} onClick={() => setScheduleOpen(true)}
              color={schedule.data?.schedule?.enabled ? 'success' : 'primary'}>
              {schedule.data?.schedule?.enabled ? 'Scheduled' : 'Schedule'}
            </Button>
          </span>
        </Tooltip>
        <Tooltip title={isNew ? 'Save first' : !valid ? 'Fix the problems first' : 'Run the whole pipeline'}>
          <span>
            <Button variant="contained" startIcon={<PlayArrowIcon />} disabled={isNew || !valid || startRun.isPending || !!activeRun} onClick={() => startRun.mutate()}>
              {activeRun ? 'Running…' : 'Run'}
            </Button>
          </span>
        </Tooltip>
      </Stack>
      {activeRun && <LinearProgress />}

      <Box sx={{ flex: 1, display: 'flex', minHeight: 0 }}>
        {/* Palette */}
        <Paper square elevation={0} sx={{ width: compact ? 56 : 230, flexShrink: 0, borderRight: 1, borderColor: 'divider', overflowY: 'auto' }}>
          {(['source', 'step', 'destination'] as const).map(cat => (
            <List key={cat} dense subheader={compact ? <Divider /> : <ListSubheader>{{ source: 'Read from', step: 'Check & shape', destination: 'Write to' }[cat]}</ListSubheader>}>
              {(palette.data ?? []).filter(p => p.category === cat).map(p => (
                <Tooltip key={p.type} placement="right" title={`${compact ? `${p.label}: ` : ''}${p.available ? p.description : `${p.description} - unavailable: ${p.unavailable_reason}`}`}>
                  <span>
                    <ListItemButton disabled={!p.available} onClick={() => addNode(p)}>
                      <ListItemIcon sx={{ minWidth: 32, color: categoryColor(theme, p.category) }}>{NODE_META[p.type].icon}</ListItemIcon>
                      {!compact && <ListItemText primary={p.label} />}
                    </ListItemButton>
                  </span>
                </Tooltip>
              ))}
            </List>
          ))}
          {palette.isError && <Alert severity="error" sx={{ m: 1 }}>Could not load steps.</Alert>}
        </Paper>

        {/* Canvas + bottom panel */}
        <Box sx={{ flex: 1, display: 'flex', flexDirection: 'column', minWidth: 280 }}>
          <Box sx={{ flex: 1, position: 'relative' }}>
            {spec.nodes.length === 0 && (
              <Stack spacing={1} alignItems="center" sx={{ position: 'absolute', inset: 0, justifyContent: 'center', zIndex: 1, pointerEvents: 'none' }}>
                <Typography variant="h6" color="text.secondary">Start with a source on the left</Typography>
                <Typography color="text.secondary">…or open the Assistant and describe what you want to load.</Typography>
              </Stack>
            )}
            <ReactFlow
              nodes={rfNodes} edges={rfEdges} nodeTypes={nodeTypes}
              onNodesChange={onNodesChange} onEdgesChange={onEdgesChange} onConnect={onConnect}
              onNodeClick={(_, n) => setSelected(n.id)} onPaneClick={() => setSelected(null)}
              fitView deleteKeyCode={['Backspace', 'Delete']}
            >
              <Background /><Controls />{!compact && <MiniMap pannable zoomable />}
            </ReactFlow>
          </Box>
          <BottomPanel
            tab={tab} setTab={setTab} issues={issues} spec={spec} setSelected={setSelected}
            preview={preview} previewNode={previewNode} setPreviewNode={setPreviewNode}
            runs={runs.data ?? []} runsLoading={runs.isLoading} isNew={isNew}
          />
        </Box>

        {/* Config drawer */}
        {selectedNode && (
          <Paper square elevation={0} sx={{ width: compact ? 340 : 420, flexShrink: 0, borderLeft: 1, borderColor: 'divider', overflowY: 'auto' }}>
            <Stack direction="row" alignItems="center" sx={{ px: 2, pt: 1 }}>
              <Typography variant="overline" sx={{ flex: 1 }}>{typeLabel(selectedNode.type)}</Typography>
              <IconButton size="small" onClick={() => setSelected(null)}><CloseIcon fontSize="small" /></IconButton>
            </Stack>
            {(issuesByNode[selectedNode.id] ?? []).map(m => <Alert key={m} severity="error" sx={{ mx: 2, mb: 1 }}>{m}</Alert>)}
            <NodeConfigPanel
              key={selectedNode.id}
              node={selectedNode}
              inputFields={fieldsIn(spec, selectedNode.id, lookup)}
              {...(selectedNode.type === 'map' ? (() => { const t = targetsFor(downstreamSink(spec, selectedNode.id)); return { mapTargets: t.targets, mapTargetLabel: t.label }; })() : {})}
              onChange={n => update({ ...spec, nodes: spec.nodes.map(x => (x.id === n.id ? n : x)) })}
              onDelete={() => {
                update({ ...spec, nodes: spec.nodes.filter(x => x.id !== selectedNode.id), edges: spec.edges.filter(e => e.from !== selectedNode.id && e.to !== selectedNode.id) });
                setSelected(null);
              }}
            />
          </Paper>
        )}

        {assistantOpen && (
          <AssistantPanel
            spec={spec} selectedNodeId={selected} preview={preview}
            onApply={(next, focus) => { update(next); if (focus) setSelected(focus); }}
            onClose={() => setAssistantOpen(false)}
          />
        )}
      </Box>
      {!isNew && <ScheduleDialog pipelineId={id!} open={scheduleOpen} onClose={() => setScheduleOpen(false)} />}
      <Snackbar open={!!toast} autoHideDuration={4000} onClose={() => setToast(null)} message={toast} />
    </Box>
  );
}

function BottomPanel(p: {
  tab: 'problems' | 'preview' | 'runs'; setTab: (t: 'problems' | 'preview' | 'runs') => void;
  issues: { node_id?: string; message: string }[]; spec: Spec; setSelected: (id: string) => void;
  preview: PreviewResult | null; previewNode: string | null; setPreviewNode: (id: string | null) => void;
  runs: RunRecord[]; runsLoading: boolean; isNew: boolean;
}) {
  const label = (id?: string) => p.spec.nodes.find(n => n.id === id)?.label || id;
  return (
    <Paper square elevation={0} sx={{ height: 280, borderTop: 1, borderColor: 'divider', display: 'flex', flexDirection: 'column' }}>
      <Tabs value={p.tab} onChange={(_, v) => p.setTab(v)} sx={{ minHeight: 36, '& .MuiTab-root': { minHeight: 36 } }}>
        <Tab value="problems" label={<Badge color="error" badgeContent={p.issues.length}>Problems&nbsp;&nbsp;</Badge>} />
        <Tab value="preview" label="Preview" />
        <Tab value="runs" label="Runs" disabled={p.isNew} />
      </Tabs>
      <Divider />
      <Box sx={{ flex: 1, overflow: 'auto', p: 1 }}>
        {p.tab === 'problems' && (p.issues.length === 0
          ? <Typography color="text.secondary" sx={{ p: 1 }}>{p.spec.nodes.length ? 'No problems - preview it, then run it.' : 'Add a source to begin.'}</Typography>
          : <List dense>{p.issues.map((i, k) => (
            <ListItemButton key={k} onClick={() => i.node_id && p.setSelected(i.node_id)}>
              <ListItemText primary={i.message} secondary={i.node_id ? label(i.node_id) : 'Pipeline'} />
            </ListItemButton>))}</List>)}
        {p.tab === 'preview' && <PreviewView {...p} label={label} />}
        {p.tab === 'runs' && <RunsView runs={p.runs} loading={p.runsLoading} label={label} />}
      </Box>
    </Paper>
  );
}

function PreviewView({ preview, previewNode, setPreviewNode, label }: {
  preview: PreviewResult | null; previewNode: string | null; setPreviewNode: (id: string | null) => void; label: (id?: string) => string | undefined;
}) {
  if (!preview) return <Typography color="text.secondary" sx={{ p: 1 }}>Press Preview to run the pipeline on the first 100 rows without writing anything.</Typography>;
  if (preview.error && !preview.summary) return <Alert severity="error">{preview.error}</Alert>;
  const s = preview.summary!;
  const lastNode = s.Nodes[s.Nodes.length - 1]?.NodeID;
  const rows = s.samples?.[previewNode ?? lastNode ?? ''] ?? [];
  const cols = [...new Set(rows.flatMap(r => Object.keys(r.Data)))];
  const rejects = preview.rejects.filter(r => !previewNode || r.node_id === previewNode);
  return (
    <Stack spacing={1}>
      {preview.error && <Alert severity="error">{preview.error}</Alert>}
      <Stack direction="row" spacing={1} flexWrap="wrap">
        <Chip label={`${s.RecordsIn} read`} /><Chip color="success" label={`${s.RecordsOut} would be written`} />
        {s.Errors > 0 && <Chip color="error" label={`${s.Errors} rejected`} />}
        <Chip variant={previewNode === null ? 'filled' : 'outlined'} color="primary" label="All steps" onClick={() => setPreviewNode(null)} />
        {s.Nodes.map(n => (
          <Chip key={n.NodeID} variant={n.NodeID === previewNode ? 'filled' : 'outlined'} color="primary"
            label={`${label(n.NodeID)}: ${n.Out}`} onClick={() => setPreviewNode(n.NodeID)} />
        ))}
      </Stack>
      {rejects.length > 0 && (
        <Table size="small">
          <TableHead><TableRow><TableCell>Row</TableCell><TableCell>Step</TableCell><TableCell>Why</TableCell></TableRow></TableHead>
          <TableBody>{rejects.slice(0, 50).map((r, i) => (
            <TableRow key={i}>
              <TableCell>{r.row}</TableCell><TableCell>{label(r.node_id)}</TableCell>
              <TableCell><Typography variant="body2" color={r.kind === 'error' ? 'error' : 'warning.main'}>{r.reason}</Typography></TableCell>
            </TableRow>))}</TableBody>
        </Table>
      )}
      {rows.length > 0 && (
        <Typography variant="caption" color="text.secondary">
          Sample rows {previewNode ? `leaving "${label(previewNode)}"` : 'that would be written'}
        </Typography>
      )}
      {rows.length > 0 && (
        <Table size="small">
          <TableHead><TableRow><TableCell>#</TableCell>{cols.map(c => <TableCell key={c}>{c}</TableCell>)}</TableRow></TableHead>
          <TableBody>{rows.map(r => (
            <TableRow key={r.Num}><TableCell>{r.Num}</TableCell>{cols.map(c => <TableCell key={c}>{fmt(r.Data[c])}</TableCell>)}</TableRow>))}</TableBody>
        </Table>
      )}
    </Stack>
  );
}

function fmt(v: unknown): string {
  if (v === null || v === undefined) return '—';
  return typeof v === 'object' ? JSON.stringify(v) : String(v);
}

const STATUS_COLOR: Record<RunRecord['status'], 'default' | 'info' | 'success' | 'warning' | 'error'> = {
  queued: 'default', running: 'info', completed: 'success', completed_with_errors: 'warning', failed: 'error',
};

function RunsView({ runs, loading, label }: { runs: RunRecord[]; loading: boolean; label: (id?: string) => string | undefined }) {
  const [open, setOpen] = useState<string | null>(null);
  const detail = useQuery({ queryKey: ['dp-run', open], enabled: !!open, queryFn: () => pipelinesApi.run(open!) });
  if (loading) return <LinearProgress />;
  if (!runs.length) return <Typography color="text.secondary" sx={{ p: 1 }}>No runs yet.</Typography>;
  return (
    <Stack direction="row" spacing={1} sx={{ height: '100%' }}>
      <List dense sx={{ width: 320, overflow: 'auto' }}>
        {runs.map(r => (
          <ListItemButton key={r.id} selected={open === r.id} onClick={() => setOpen(r.id)}>
            <ListItemText
              primary={<Stack direction="row" spacing={1} alignItems="center">
                <Chip size="small" color={STATUS_COLOR[r.status]} label={r.status.replace(/_/g, ' ')} />
                <span>{new Date(r.start_time).toLocaleString()}</span></Stack>}
              secondary={`${r.records_in} read · ${r.records_out} written · ${r.errors} rejected`}
            />
          </ListItemButton>
        ))}
      </List>
      <Box sx={{ flex: 1, overflow: 'auto' }}>
        {detail.data && (
          <Stack spacing={1}>
            <Table size="small">
              <TableHead><TableRow><TableCell>Step</TableCell><TableCell>In</TableCell><TableCell>Out</TableCell><TableCell>Rejected</TableCell><TableCell>Time</TableCell><TableCell /></TableRow></TableHead>
              <TableBody>{(detail.data.steps ?? []).map(s => (
                <TableRow key={s.NodeID}>
                  <TableCell>{label(s.NodeID) ?? s.Label}</TableCell><TableCell>{s.In}</TableCell><TableCell>{s.Out}</TableCell>
                  <TableCell>{s.Errors}</TableCell><TableCell>{(s.Duration / 1e9).toFixed(1)}s</TableCell>
                  <TableCell>{s.Err && <Typography variant="caption" color="error">{s.Err}</Typography>}</TableCell>
                </TableRow>))}</TableBody>
            </Table>
            {detail.data.errors_sample.slice(0, 100).map((e, i) => (
              <Typography key={i} variant="body2" color={e.run_error || e.kind === 'error' ? 'error' : 'warning.main'}>
                {e.run_error ?? `Row ${e.row} · ${label(e.node_id)} · ${e.reason}`}
              </Typography>
            ))}
          </Stack>
        )}
      </Box>
    </Stack>
  );
}
