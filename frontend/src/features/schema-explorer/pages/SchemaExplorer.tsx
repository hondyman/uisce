import React, { useState, useMemo, useEffect, useCallback } from 'react';
import { useSearchParams } from 'react-router-dom';
import ReactFlow, {
  Background,
  Controls,
  MiniMap,
  MarkerType,
  Handle,
  Position,
  useNodesState,
  useEdgesState,
  BackgroundVariant,
} from 'reactflow';
import 'reactflow/dist/style.css';
import { useTenant } from '../../../contexts/TenantContext';
import { useApiQuery } from '../../../hooks/useApiQuery';
import useBlockableNavigate from '../../../components/RouteBlocker/useBlockableNavigate';
import apiClient from '../../../utils/apiClient';

// ─────────────────────────────────────────────
// Types
// ─────────────────────────────────────────────
interface CatalogNode {
  id: string;
  node_name: string;
  qualified_path?: string;
  node_type_id: string;
  parent_id?: string;
  tenant_tenant_instance_id?: string;
  properties?: {
    data_type?: string;
    is_nullable?: boolean;
    is_primary_key?: boolean;
    is_foreign_key?: boolean;
    description?: string;
    row_count?: number;
    [key: string]: any;
  };
}

interface CatalogEdge {
  id: string;
  source_node_id: string;
  target_node_id: string;
  edge_type_id?: string;
  relationship_type?: string;
  properties?: {
    label?: string;
    cardinality?: string;
    columns?: Array<{
      source_column: string;
      target_column: string;
      source_column_id?: string;
      target_column_id?: string;
      ordinal_position?: number;
    }>;
    source_table?: string;
    target_table?: string;
    on_delete?: string;
    on_update?: string;
    [key: string]: any;
  };
}

interface CatalogNodeType {
  id: string;
  catalog_type_name: string;
  description?: string;
}

interface TableNode extends CatalogNode {
  schema: string;
  columns: CatalogNode[];
}

const TABLE_TYPE = '49a50271-ae58-4d3e-ae1c-2f5b89d89192';
const COLUMN_TYPE = 'a64c1011-16e8-4ddf-b447-363bf8e15c9a';

// ─────────────────────────────────────────────
// Theme tokens
// ─────────────────────────────────────────────
const C = {
  bg: '#0A0C12',
  sidebar: '#0F1117',
  panel: '#13161E',
  border: 'rgba(255,255,255,0.07)',
  accent: '#6366F1',
  accentDim: 'rgba(99,102,241,0.15)',
  accentGlow: '0 0 20px rgba(99,102,241,0.4)',
  text: '#E2E8F0',
  textMuted: '#8892A4',
  success: '#10B981',
  warning: '#F59E0B',
  danger: '#EF4444',
  purple: '#A78BFA',
  teal: '#2DD4BF',
  blue: '#60A5FA',
  orange: '#FB923C',
  green: '#4ADE80',
};

const TYPE_COLOR: Record<string, string> = {
  uuid: C.purple,
  varchar: C.blue,
  character: C.blue,
  char: C.blue,
  text: C.blue,
  int: C.green,
  integer: C.green,
  bigint: C.green,
  smallint: C.green,
  numeric: C.green,
  float: C.green,
  double: C.green,
  decimal: C.green,
  bool: C.orange,
  boolean: C.orange,
  timestamp: C.teal,
  date: C.teal,
  time: C.teal,
  jsonb: C.warning,
  json: C.warning,
};

function typeColor(dt: string): string {
  const lower = dt.toLowerCase();
  for (const key of Object.keys(TYPE_COLOR)) {
    if (lower.startsWith(key)) return TYPE_COLOR[key];
  }
  return C.textMuted;
}

// ─────────────────────────────────────────────
// Sub-components
// ─────────────────────────────────────────────

const Badge: React.FC<{ label: string; color: string; bg?: string }> = ({ label, color, bg }) => (
  <span style={{
    display: 'inline-flex', alignItems: 'center', padding: '1px 7px',
    borderRadius: 9999, fontSize: 10, fontWeight: 700, letterSpacing: '0.04em',
    color, background: bg ?? `${color}22`, border: `1px solid ${color}44`,
    fontFamily: 'monospace', textTransform: 'uppercase',
  }}>
    {label}
  </span>
);

const Pill: React.FC<{ children: React.ReactNode; active?: boolean; onClick: () => void }> = ({ children, active, onClick }) => (
  <button onClick={onClick} style={{
    display: 'flex', alignItems: 'center', gap: 6, padding: '6px 14px',
    background: active ? C.accentDim : 'transparent',
    border: `1px solid ${active ? C.accent : C.border}`,
    borderRadius: 8, cursor: 'pointer', color: active ? C.accent : C.textMuted,
    fontSize: 13, fontWeight: 600, transition: 'all 0.2s ease',
    boxShadow: active ? C.accentGlow : 'none',
    whiteSpace: 'nowrap',
  }}>
    {children}
  </button>
);

const Spinner: React.FC = () => (
  <div style={{ display: 'flex', justifyContent: 'center', alignItems: 'center', padding: 48 }}>
    <div style={{
      width: 36, height: 36, border: `3px solid ${C.border}`,
      borderTop: `3px solid ${C.accent}`, borderRadius: '50%',
      animation: 'spin 0.8s linear infinite',
    }} />
    <style>{`@keyframes spin { to { transform: rotate(360deg); } }`}</style>
  </div>
);

const Empty: React.FC<{ icon: string; title: string; subtitle?: string }> = ({ icon, title, subtitle }) => (
  <div style={{ display: 'flex', flexDirection: 'column', alignItems: 'center', justifyContent: 'center', padding: 64, gap: 12, textAlign: 'center' }}>
    <div style={{ fontSize: 48, opacity: 0.3 }}>{icon}</div>
    <div style={{ color: C.text, fontWeight: 600, fontSize: 15 }}>{title}</div>
    {subtitle && <div style={{ color: C.textMuted, fontSize: 13, maxWidth: 280 }}>{subtitle}</div>}
  </div>
);

// ─────────────────────────────────────────────
// ERD Custom Node
// ─────────────────────────────────────────────
const ErdTableNode: React.FC<{ data: any }> = ({ data }) => (
  <div style={{
    background: C.panel, border: `1px solid ${data.selected ? C.accent : C.border}`,
    borderRadius: 10, minWidth: 180, overflow: 'hidden',
    boxShadow: data.selected ? C.accentGlow : '0 4px 24px rgba(0,0,0,0.5)',
    transition: 'all 0.2s ease',
  }}>
    <Handle type="target" position={Position.Top} style={{ background: C.accent, width: 8, height: 8, border: 'none' }} />
    <div style={{ background: data.selected ? C.accent : '#1E2130', padding: '8px 12px', borderBottom: `1px solid ${C.border}` }}>
      <div style={{ color: C.text, fontWeight: 700, fontSize: 12, fontFamily: 'monospace' }}>
        📋 {data.label}
      </div>
      <div style={{ color: C.textMuted, fontSize: 10, marginTop: 2 }}>{data.schema}</div>
    </div>
    <div style={{ padding: '4px 0' }} className="erd-node-columns lineage-node-columns">
      {(data.columns || []).slice(0, 8).map((col: any, i: number) => (
        <div key={i} style={{
          display: 'flex', alignItems: 'center', gap: 6, padding: '3px 12px',
          borderBottom: `1px solid ${C.border}22`,
        }}>
          {col.isPrimaryKey && <span style={{ color: C.warning, fontSize: 9 }}>🔑</span>}
          {col.isForeignKey && !col.isPrimaryKey && <span style={{ color: C.blue, fontSize: 9 }}>🔗</span>}
          {!col.isPrimaryKey && !col.isForeignKey && <span style={{ fontSize: 9, opacity: 0.3 }}>·</span>}
          <span style={{ color: C.text, fontSize: 11, flex: 1, fontFamily: 'monospace' }}>{col.name}</span>
          <span style={{ color: typeColor(col.type || ''), fontSize: 10, fontFamily: 'monospace' }}>{col.type}</span>
        </div>
      ))}
      {(data.columns || []).length > 8 && (
        <div style={{ padding: '3px 12px', color: C.textMuted, fontSize: 10, textAlign: 'center' }}>
          +{data.columns.length - 8} more columns
        </div>
      )}
    </div>
    <Handle type="source" position={Position.Bottom} style={{ background: C.accent, width: 8, height: 8, border: 'none' }} />
  </div>
);

const nodeTypes = { erdTable: ErdTableNode, lineageTable: ErdTableNode };

// ─────────────────────────────────────────────
// Tabs
// ─────────────────────────────────────────────
type Tab = 'properties' | 'columns' | 'relationships' | 'erd' | 'lineage';

// ─────────────────────────────────────────────
// Relationship dialog
// ─────────────────────────────────────────────
const RelationshipDialog: React.FC<{
  open: boolean;
  tables: TableNode[];
  existing?: CatalogEdge | null;
  onClose: () => void;
  onSave: (edge: Partial<CatalogEdge>) => void;
}> = ({ open, tables, existing, onClose, onSave }) => {
  const [src, setSrc] = useState(existing?.source_node_id ?? '');
  const [tgt, setTgt] = useState(existing?.target_node_id ?? '');
  const [label, setLabel] = useState(existing?.properties?.label ?? '');

  useEffect(() => {
    setSrc(existing?.source_node_id ?? '');
    setTgt(existing?.target_node_id ?? '');
    setLabel(existing?.properties?.label ?? '');
  }, [existing, open]);

  if (!open) return null;

  const inp: React.CSSProperties = {
    width: '100%', background: C.bg, border: `1px solid ${C.border}`,
    borderRadius: 8, padding: '10px 14px', color: C.text, fontSize: 14,
    outline: 'none', boxSizing: 'border-box',
  };

  return (
    <div style={{
      position: 'fixed', inset: 0, background: 'rgba(0,0,0,0.7)',
      display: 'flex', alignItems: 'center', justifyContent: 'center', zIndex: 9999,
      backdropFilter: 'blur(4px)',
    }} onClick={onClose}>
      <div onClick={e => e.stopPropagation()} style={{
        background: C.sidebar, border: `1px solid ${C.border}`,
        borderRadius: 16, padding: 32, width: 480,
        boxShadow: '0 32px 64px rgba(0,0,0,0.6)',
      }}>
        <h3 style={{ color: C.text, margin: '0 0 24px', fontSize: 18, fontWeight: 700 }}>
          {existing ? 'Edit Relationship' : 'Add Relationship'}
        </h3>
        <div style={{ display: 'flex', flexDirection: 'column', gap: 16 }}>
          <div>
            <label style={{ color: C.textMuted, fontSize: 12, fontWeight: 600, display: 'block', marginBottom: 6 }}>SOURCE TABLE</label>
            <select value={src} onChange={e => setSrc(e.target.value)} style={{ ...inp, appearance: 'auto' }}>
              <option value="">Select source table…</option>
              {tables.map(t => <option key={t.id} value={t.id}>{t.qualified_path || t.node_name}</option>)}
            </select>
          </div>
          <div>
            <label style={{ color: C.textMuted, fontSize: 12, fontWeight: 600, display: 'block', marginBottom: 6 }}>TARGET TABLE</label>
            <select value={tgt} onChange={e => setTgt(e.target.value)} style={{ ...inp, appearance: 'auto' }}>
              <option value="">Select target table…</option>
              {tables.map(t => <option key={t.id} value={t.id}>{t.qualified_path || t.node_name}</option>)}
            </select>
          </div>
          <div>
            <label style={{ color: C.textMuted, fontSize: 12, fontWeight: 600, display: 'block', marginBottom: 6 }}>LABEL</label>
            <input value={label} onChange={e => setLabel(e.target.value)} placeholder="e.g. FK, references, owns…" style={inp} />
          </div>
        </div>
        <div style={{ display: 'flex', gap: 12, marginTop: 28, justifyContent: 'flex-end' }}>
          <button onClick={onClose} style={{
            padding: '10px 20px', borderRadius: 8, border: `1px solid ${C.border}`,
            background: 'transparent', color: C.textMuted, cursor: 'pointer', fontSize: 14,
          }}>Cancel</button>
          <button
            onClick={() => {
              if (!src || !tgt) return;
              onSave({ source_node_id: src, target_node_id: tgt, properties: { label } });
            }}
            style={{
              padding: '10px 24px', borderRadius: 8, border: 'none',
              background: C.accent, color: '#fff', cursor: 'pointer', fontSize: 14, fontWeight: 700,
              boxShadow: C.accentGlow,
            }}
          >
            {existing ? 'Save Changes' : 'Add Relationship'}
          </button>
        </div>
      </div>
    </div>
  );
};

// ─────────────────────────────────────────────
// Main SchemaExplorerPage
// ─────────────────────────────────────────────
const SchemaExplorerPage: React.FC = () => {
  const [searchParams] = useSearchParams();
  const navigate = useBlockableNavigate();
  const { tenant: scopedTenant, datasource: contextDatasource } = useTenant();

  const datasourceId = searchParams.get('instanceId') || searchParams.get('datasource') || contextDatasource?.id || '';
  // tenantId available for future use
  const _tenantId = searchParams.get('tenantId') || scopedTenant?.id || '';

  // ── Data fetching ──────────────────────────
  const nodesUrl = datasourceId ? `api/rest/catalog-nodes?limit=10000&datasource_id=${encodeURIComponent(datasourceId)}` : `api/rest/catalog-nodes?limit=10000`;
  const edgesUrl = datasourceId ? `api/rest/catalog-edges?limit=10000&datasource_id=${encodeURIComponent(datasourceId)}` : `api/rest/catalog-edges?limit=10000`;

  const { loading: nodesLoading, data: rawNodes, refetch: refetchNodes } = useApiQuery<CatalogNode[]>(nodesUrl);
  const { loading: edgesLoading, data: rawEdges, refetch: refetchEdges } = useApiQuery<CatalogEdge[]>(edgesUrl);
  const { data: rawNodeTypes } = useApiQuery<CatalogNodeType[]>(
    `api/rest/catalog-node-types?limit=1000`
  );
  const nodeTypeMap = useMemo(() => {
    const m = new Map<string, CatalogNodeType>();
    (rawNodeTypes ?? []).forEach(nt => m.set(nt.id, nt));
    return m;
  }, [rawNodeTypes]);

  const loading = nodesLoading || edgesLoading;

  // ── Process nodes into tables ──────────────
  const { tables, tableMap } = useMemo(() => {
    if (!rawNodes) return { tables: [] as TableNode[], tableMap: new Map<string, TableNode>() };

    // 1. Deduplicate raw nodes strictly by UUID id
    const uniqueNodeMap = new Map<string, CatalogNode>();
    rawNodes.forEach(n => {
      if (n.id && !uniqueNodeMap.has(n.id)) {
        uniqueNodeMap.set(n.id, n);
      }
    });
    const uniqueNodes = Array.from(uniqueNodeMap.values());

    // 2. Resolve table and column node_type_ids (dynamically from catalog-node-types if available, fallback to constants)
    const tableTypeIds = new Set<string>([TABLE_TYPE]);
    const columnTypeIds = new Set<string>([COLUMN_TYPE]);

    nodeTypeMap.forEach((nt, id) => {
      const typeName = (nt.catalog_type_name || '').toLowerCase();
      if (typeName === 'table' || typeName === 'database_table' || typeName === 'physical_table' || typeName === 'iceberg_table') {
        tableTypeIds.add(id);
      } else if (typeName === 'column' || typeName === 'database_column' || typeName === 'iceberg_column') {
        columnTypeIds.add(id);
      }
    });

    const tableNodes = uniqueNodes.filter(n => tableTypeIds.has(n.node_type_id));
    const columnNodes = uniqueNodes.filter(n => columnTypeIds.has(n.node_type_id));

    // 3. Map columns to tables strictly by UUID references (parent_id, parent_node_id, properties.table_id, properties.parent_id)
    const processedTables: TableNode[] = [];
    const seenTableIds = new Set<string>();

    tableNodes.forEach(t => {
      if (seenTableIds.has(t.id)) return;
      seenTableIds.add(t.id);

      const cols = columnNodes.filter(c => {
        if (c.parent_id === t.id) return true;
        if (c.properties?.parent_id === t.id || c.properties?.table_id === t.id) return true;
        if (t.qualified_path && c.qualified_path && c.qualified_path.startsWith(t.qualified_path + '/')) return true;
        return false;
      });

      // Deduplicate columns strictly by column node UUID id
      const uniqueCols: CatalogNode[] = [];
      const seenColIds = new Set<string>();
      cols.forEach(c => {
        if (!seenColIds.has(c.id)) {
          seenColIds.add(c.id);
          uniqueCols.push(c);
        }
      });

      const schema = t.properties?.schema || 
                     (t.qualified_path?.startsWith('/') ? t.qualified_path.split('/')[1] : null) || 
                     (t.qualified_path?.includes('.') ? t.qualified_path.split('.')[0] : null) || 
                     'public';
      processedTables.push({ ...t, schema, columns: uniqueCols });
    });

    const mp = new Map<string, TableNode>();
    processedTables.forEach(t => mp.set(t.id, t));
    return { tables: processedTables, tableMap: mp };
  }, [rawNodes, nodeTypeMap]);

  const allNodeMap = useMemo(() => {
    const m = new Map<string, CatalogNode>();
    (rawNodes ?? []).forEach(n => { if (n.id) m.set(n.id, n); });
    return m;
  }, [rawNodes]);

  const edges: CatalogEdge[] = useMemo(() => rawEdges ?? [], [rawEdges]);

  // ── Schema grouping ────────────────────────
  const schemaGroups = useMemo(() => {
    const groups: Record<string, TableNode[]> = {};
    tables.forEach(t => {
      if (!groups[t.schema]) groups[t.schema] = [];
      groups[t.schema].push(t);
    });
    // sort schemas, sort tables within each
    Object.keys(groups).forEach(s => groups[s].sort((a, b) => a.node_name.localeCompare(b.node_name)));
    return groups;
  }, [tables]);

  // ── Local UI state ─────────────────────────
  const [search, setSearch] = useState('');
  const [expandedSchemas, setExpandedSchemas] = useState<Set<string>>(new Set());
  const [selectedTableId, setSelectedTableId] = useState<string | null>(null);
  const [activeTab, setActiveTab] = useState<Tab>('columns');
  const [dialogOpen, setDialogOpen] = useState(false);
  const [editingEdge, setEditingEdge] = useState<CatalogEdge | null>(null);
  const [localEdges, setLocalEdges] = useState<CatalogEdge[]>([]);
  const [_savingEdge, setSavingEdge] = useState(false);
  const [deletingEdgeId, setDeletingEdgeId] = useState<string | null>(null);
  const [showErdColumns, setShowErdColumns] = useState(true);
  const [showLineageColumns, setShowLineageColumns] = useState(true);

  // Sync remote edges to local
  useEffect(() => { setLocalEdges(edges); }, [edges]);

  // Auto-expand all schemas when first loaded
  useEffect(() => {
    if (Object.keys(schemaGroups).length > 0 && expandedSchemas.size === 0) {
      setExpandedSchemas(new Set(Object.keys(schemaGroups)));
    }
  }, [schemaGroups]);

  // Auto-select first table
  useEffect(() => {
    if (tables.length > 0 && !selectedTableId) {
      setSelectedTableId(tables[0].id);
    }
  }, [tables]);

  const selectedTable = selectedTableId ? tableMap.get(selectedTableId) : null;

  // ── Filtered tree ──────────────────────────
  const filteredGroups = useMemo(() => {
    if (!search.trim()) return schemaGroups;
    const q = search.toLowerCase();
    const result: Record<string, TableNode[]> = {};
    Object.entries(schemaGroups).forEach(([schema, tbls]) => {
      const matched = tbls.filter(t =>
        t.node_name.toLowerCase().includes(q) || schema.toLowerCase().includes(q)
      );
      if (matched.length > 0) result[schema] = matched;
    });
    return result;
  }, [schemaGroups, search]);

  // Auto-expand schemas that match search
  useEffect(() => {
    if (search.trim()) {
      setExpandedSchemas(new Set(Object.keys(filteredGroups)));
    }
  }, [filteredGroups, search]);

  const toggleSchema = useCallback((s: string) => {
    setExpandedSchemas(prev => {
      const next = new Set(prev);
      next.has(s) ? next.delete(s) : next.add(s);
      return next;
    });
  }, []);

  // ── ERD nodes/edges ────────────────────────
  const [flowNodes, setFlowNodes, onFlowNodesChange] = useNodesState([]);
  const [flowEdges, setFlowEdges, onFlowEdgesChange] = useEdgesState([]);

  useEffect(() => {
    if (tables.length === 0) return;
    const COLS = 4;
    const W = 220, H_BASE = 48, H_ROW = 26, GAP_X = 80, GAP_Y = 60;
    const newNodes = tables.map((t, i) => ({
      id: t.id,
      type: 'erdTable',
      position: {
        x: (i % COLS) * (W + GAP_X),
        y: Math.floor(i / COLS) * (H_BASE + t.columns.length * H_ROW + GAP_Y),
      },
      data: {
        label: t.node_name,
        schema: t.schema,
        selected: t.id === selectedTableId,
        columns: t.columns.map(c => ({
          name: c.node_name,
          type: c.properties?.data_type || '?',
          isPrimaryKey: c.properties?.is_primary_key,
          isForeignKey: c.properties?.is_foreign_key,
        })),
      },
    }));
    const resolveTableId = (id: string): string => {
      if (tableMap.has(id)) return id;
      const nodeObj = allNodeMap.get(id);
      if (nodeObj?.parent_id && tableMap.has(nodeObj.parent_id)) return nodeObj.parent_id;
      for (const t of tables) {
        if (t.columns.some(c => c.id === id)) return t.id;
      }
      return id;
    };

    const newEdges = localEdges
      .map(e => {
        const src = resolveTableId(e.source_node_id);
        const tgt = resolveTableId(e.target_node_id);
        if (!tableMap.has(src) || !tableMap.has(tgt)) return null;
        return {
          id: e.id,
          source: src,
          target: tgt,
          label: e.relationship_type || e.properties?.edge_type_name || e.properties?.label,
          type: 'smoothstep',
          animated: false,
          style: { stroke: C.accent, strokeWidth: 2 },
          markerEnd: { type: MarkerType.ArrowClosed, color: C.accent },
        };
      })
      .filter(Boolean);

    setFlowNodes(newNodes as any);
    setFlowEdges(newEdges as any);
  }, [tables, localEdges, selectedTableId, tableMap, allNodeMap]);

  // Update selected highlight in ERD nodes
  useEffect(() => {
    setFlowNodes(nds => nds.map(n => ({ ...n, data: { ...n.data, selected: n.id === selectedTableId } })));
  }, [selectedTableId]);

  // ── Relationship CRUD ──────────────────────
  const handleSaveRelationship = useCallback(async (edge: Partial<CatalogEdge>) => {
    setSavingEdge(true);
    try {
      if (editingEdge) {
        const updated = await apiClient<CatalogEdge>(`api/glossary/edges/${editingEdge.id}`, {
          method: 'PUT',
          body: JSON.stringify(edge),
          headers: { 'Content-Type': 'application/json' },
        });
        setLocalEdges(prev => prev.map(e => e.id === editingEdge.id ? updated : e));
      } else {
        const created = await apiClient<CatalogEdge>('api/glossary/edges', {
          method: 'POST',
          body: JSON.stringify({ ...edge, datasource_id: datasourceId }),
          headers: { 'Content-Type': 'application/json' },
        });
        setLocalEdges(prev => [...prev, created]);
      }
      setDialogOpen(false);
      setEditingEdge(null);
    } catch (err) {
      console.error('Failed to save relationship:', err);
    } finally {
      setSavingEdge(false);
    }
  }, [editingEdge, datasourceId]);

  const handleDeleteRelationship = useCallback(async (id: string) => {
    setDeletingEdgeId(id);
    try {
      await apiClient(`api/glossary/edges/${id}`, { method: 'DELETE' });
      setLocalEdges(prev => prev.filter(e => e.id !== id));
    } catch (err) {
      console.error('Failed to delete relationship:', err);
    } finally {
      setDeletingEdgeId(null);
    }
  }, []);



  // ── Tab stats ──────────────────────────────
  const tableRelEdges = useMemo(() => {
    if (!selectedTable) return [];
    const tableAndColIds = new Set<string>([
      selectedTable.id,
      ...selectedTable.columns.map(c => c.id)
    ]);
    const resolveTableId = (id: string): string => {
      if (tableMap.has(id)) return id;
      const nodeObj = allNodeMap.get(id);
      if (nodeObj?.parent_id && tableMap.has(nodeObj.parent_id)) return nodeObj.parent_id;
      for (const t of tables) {
        if (t.columns.some(c => c.id === id)) return t.id;
      }
      return id;
    };

    const isMatch = (e: CatalogEdge) => {
      if (tableAndColIds.has(e.source_node_id) || tableAndColIds.has(e.target_node_id)) return true;
      if (resolveTableId(e.source_node_id) === selectedTable.id || resolveTableId(e.target_node_id) === selectedTable.id) return true;
      if (e.properties?.source_table === selectedTable.node_name || e.properties?.target_table === selectedTable.node_name) return true;
      return false;
    };

    return localEdges.filter(isMatch);
  }, [localEdges, selectedTable, tableMap, allNodeMap, tables]);

  const relCount = tableRelEdges.length;
  const colCount = selectedTable?.columns.length ?? 0;

  // ── Lineage nodes ──────────────────────────
  const [lineageNodes, setLineageNodes, onLineageNodesChange] = useNodesState([]);
  const [lineageEdges, setLineageEdges, onLineageEdgesChange] = useEdgesState([]);

  useEffect(() => {
    if (!selectedTable || localEdges.length === 0) {
      setLineageNodes([]);
      setLineageEdges([]);
      return;
    }
    const tableAndColIds = new Set<string>([
      selectedTable.id,
      ...selectedTable.columns.map(c => c.id)
    ]);
    const relevant = tableRelEdges;
    // Helper to resolve a node ID (column or table) to its parent table ID
    const resolveTableId = (id: string): string => {
      if (tableMap.has(id)) return id;
      const nodeObj = allNodeMap.get(id);
      if (nodeObj?.parent_id && tableMap.has(nodeObj.parent_id)) return nodeObj.parent_id;
      // Fallback: check if any table owns this column ID
      for (const t of tables) {
        if (t.columns.some(c => c.id === id)) return t.id;
      }
      return id;
    };

    const ids = new Set<string>([selectedTable.id]);
    relevant.forEach(e => {
      ids.add(resolveTableId(e.source_node_id));
      ids.add(resolveTableId(e.target_node_id));
    });
    const nodeArr = Array.from(ids);
    const lNodes = nodeArr.map((id, i) => {
      const nodeObj = allNodeMap.get(id);
      const tableNode = tableMap.get(id) || tables.find(t => t.id === id);
      const isSelf = id === selectedTable.id;
      const typeObj = nodeObj ? nodeTypeMap.get(nodeObj.node_type_id) : undefined;
      const typeLabel = typeObj?.catalog_type_name ? ` (${typeObj.catalog_type_name})` : '';
      return {
        id,
        type: 'lineageTable',
        position: { x: isSelf ? 300 : (i % 2 === 0 ? 0 : 600), y: isSelf ? 120 : (Math.floor(i / 2) * 120) },
        style: {
          background: isSelf ? C.accent : C.panel,
          border: `1px solid ${isSelf ? C.accent : C.border}`,
          borderRadius: 10, color: C.text, padding: '8px 16px', fontSize: 13, fontWeight: 700,
          boxShadow: isSelf ? C.accentGlow : 'none',
        },
        data: {
          label: (nodeObj?.node_name ?? id) + typeLabel,
          selected: isSelf,
          schema: tableNode?.schema,
          columns: (tableNode?.columns || []).map((c: any) => ({
            name: c.node_name,
            type: c.properties?.data_type || '?',
            isPrimaryKey: c.properties?.is_primary_key,
            isForeignKey: c.properties?.is_foreign_key,
          })),
        },
      };
    });
    const lEdges = relevant.map(e => ({
      id: e.id,
      source: resolveTableId(e.source_node_id),
      target: resolveTableId(e.target_node_id),
      type: 'smoothstep',
      animated: false,
      label: e.relationship_type || e.properties?.edge_type_name || e.properties?.label,
      style: { stroke: C.accent, strokeWidth: 2 },
      markerEnd: { type: MarkerType.ArrowClosed, color: C.accent },
    }));
    setLineageNodes(lNodes as any);
    setLineageEdges(lEdges as any);
  }, [selectedTable, localEdges, allNodeMap, nodeTypeMap, tableMap, tables]);

  // ─────────────────────────────────────────────
  // Render
  // ─────────────────────────────────────────────
  if (!datasourceId) {
    return (
      <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'center', height: '100vh', background: C.bg }}>
        <div style={{ textAlign: 'center', color: C.textMuted }}>
          <div style={{ fontSize: 48, marginBottom: 16 }}>🔌</div>
          <div style={{ fontSize: 18, fontWeight: 700, color: C.text, marginBottom: 8 }}>No datasource selected</div>
          <div style={{ fontSize: 14 }}>Provide an <code>instanceId</code> query parameter to load a schema.</div>
        </div>
      </div>
    );
  }

  return (
    <div style={{ display: 'flex', flexDirection: 'column', height: '100vh', background: C.bg, fontFamily: "'Inter', 'Roboto', system-ui, sans-serif", color: C.text }}>
      <style>{`
        * { box-sizing: border-box; }
        ::-webkit-scrollbar { width: 6px; height: 6px; }
        ::-webkit-scrollbar-track { background: transparent; }
        ::-webkit-scrollbar-thumb { background: rgba(255,255,255,0.1); border-radius: 3px; }
        ::-webkit-scrollbar-thumb:hover { background: rgba(255,255,255,0.2); }
        @keyframes fadein { from { opacity: 0; transform: translateY(4px); } to { opacity: 1; transform: none; } }
        .tree-item { transition: background 0.15s ease; border-radius: 6px; }
        .tree-item:hover { background: rgba(255,255,255,0.05) !important; }
        .tree-item.selected { background: rgba(99,102,241,0.15) !important; border-left: 2px solid #6366F1 !important; }
        .react-flow__node { transition: box-shadow 0.2s ease; }
        input, select { color-scheme: dark; }
        .hide-columns .erd-node-columns { display: none !important; }
        .hide-columns .lineage-node-columns { display: none !important; }
        .hide-columns .react-flow__node { min-height: auto !important; }
        .hide-columns .react-flow__node .erd-node-columns { display: none !important; }
        .hide-columns .react-flow__node .lineage-node-columns { display: none !important; }
      `}</style>

      {/* ── Top Bar ── */}
      <div style={{
        display: 'flex', alignItems: 'center', gap: 16, padding: '12px 24px',
        background: C.sidebar, borderBottom: `1px solid ${C.border}`,
        flexShrink: 0,
      }}>
        <button onClick={() => navigate(-1)} style={{
          background: 'transparent', border: `1px solid ${C.border}`, borderRadius: 8,
          color: C.textMuted, cursor: 'pointer', padding: '6px 12px', fontSize: 13,
          display: 'flex', alignItems: 'center', gap: 6, transition: 'all 0.2s',
        }}>← Back</button>
        <div style={{ width: 1, height: 24, background: C.border }} />
        <div style={{ display: 'flex', flexDirection: 'column' }}>
          <span style={{ color: C.text, fontWeight: 700, fontSize: 16, letterSpacing: '-0.02em' }}>Schema Explorer</span>
          <span style={{ color: C.textMuted, fontSize: 11, fontFamily: 'monospace' }}>{datasourceId}</span>
        </div>
        <div style={{ flex: 1 }} />
        {!loading && (
          <>
            <Badge label={`${tables.length} tables`} color={C.accent} />
            <Badge label={`${Object.keys(schemaGroups).length} schemas`} color={C.teal} />
            <Badge label={`${relCount} relationships`} color={C.purple} />
          </>
        )}
        <button onClick={() => { refetchNodes(); refetchEdges(); }} style={{
          background: 'transparent', border: `1px solid ${C.border}`, borderRadius: 8,
          color: C.textMuted, cursor: 'pointer', padding: '6px 12px', fontSize: 13,
          transition: 'all 0.2s',
        }}>↺ Refresh</button>
      </div>

      {/* ── Main content ── */}
      <div style={{ display: 'flex', flex: 1, overflow: 'hidden' }}>

        {/* ── Left Sidebar: Schema Tree ── */}
        <div style={{
          width: 280, flexShrink: 0, background: C.sidebar, borderRight: `1px solid ${C.border}`,
          display: 'flex', flexDirection: 'column', overflow: 'hidden',
        }}>
          {/* Search */}
          <div style={{ padding: '14px 16px', borderBottom: `1px solid ${C.border}` }}>
            <div style={{ position: 'relative' }}>
              <span style={{ position: 'absolute', left: 12, top: '50%', transform: 'translateY(-50%)', color: C.textMuted, fontSize: 14 }}>🔍</span>
              <input
                value={search}
                onChange={e => setSearch(e.target.value)}
                placeholder="Search tables…"
                style={{
                  width: '100%', background: C.bg, border: `1px solid ${C.border}`,
                  borderRadius: 8, padding: '8px 12px 8px 34px', color: C.text,
                  fontSize: 13, outline: 'none',
                }}
              />
            </div>
          </div>

          {/* Tree */}
          <div style={{ flex: 1, overflowY: 'auto', padding: '8px 8px' }}>
            {loading ? (
              <div style={{ padding: 16 }}>
                {[...Array(6)].map((_, i) => (
                  <div key={i} style={{
                    height: 32, background: 'rgba(255,255,255,0.04)', borderRadius: 6,
                    marginBottom: 6, animation: 'fadein 0.5s ease',
                    animationDelay: `${i * 0.05}s`, animationFillMode: 'both',
                  }} />
                ))}
              </div>
            ) : Object.keys(filteredGroups).length === 0 ? (
              <div style={{ padding: 24, textAlign: 'center', color: C.textMuted, fontSize: 13 }}>
                {search ? 'No tables match your search.' : 'No tables found.'}
              </div>
            ) : (
              Object.entries(filteredGroups).sort(([a], [b]) => a.localeCompare(b)).map(([schema, tbls]) => (
                <div key={schema} style={{ marginBottom: 4 }}>
                  {/* Schema header */}
                  <div
                    onClick={() => toggleSchema(schema)}
                    style={{
                      display: 'flex', alignItems: 'center', gap: 8, padding: '6px 8px',
                      cursor: 'pointer', borderRadius: 6, marginBottom: 2,
                      transition: 'background 0.15s',
                    }}
                    className="tree-item"
                  >
                    <span style={{ color: C.accent, fontSize: 11, transition: 'transform 0.2s', transform: expandedSchemas.has(schema) ? 'rotate(90deg)' : 'none', display: 'inline-block' }}>▶</span>
                    <span style={{ fontSize: 11, fontWeight: 700, color: C.textMuted, textTransform: 'uppercase', letterSpacing: '0.06em', flex: 1 }}>
                      🗂 {schema}
                    </span>
                    <span style={{
                      fontSize: 10, fontWeight: 700, color: C.accent, background: C.accentDim,
                      borderRadius: 9999, padding: '1px 7px', fontFamily: 'monospace',
                    }}>{tbls.length}</span>
                  </div>

                  {/* Tables */}
                  {expandedSchemas.has(schema) && (
                    <div style={{ paddingLeft: 16 }}>
                      {tbls.map(t => {
                        const isSelected = t.id === selectedTableId;
                        return (
                          <div
                            key={t.id}
                            onClick={() => { setSelectedTableId(t.id); setActiveTab('columns'); }}
                            className={`tree-item${isSelected ? ' selected' : ''}`}
                            style={{
                              display: 'flex', alignItems: 'center', gap: 8, padding: '5px 10px',
                              cursor: 'pointer', marginBottom: 1,
                              borderLeft: isSelected ? `2px solid ${C.accent}` : '2px solid transparent',
                              animation: 'fadein 0.2s ease',
                            }}
                          >
                            <span style={{ fontSize: 11 }}>📋</span>
                            <span style={{
                              flex: 1, fontSize: 13, fontWeight: isSelected ? 600 : 400,
                              color: isSelected ? C.accent : C.text,
                              overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap',
                              fontFamily: 'monospace',
                            }}>{t.node_name}</span>
                            <span style={{ fontSize: 10, color: C.textMuted, flexShrink: 0 }}>{t.columns.length}</span>
                          </div>
                        );
                      })}
                    </div>
                  )}
                </div>
              ))
            )}
          </div>
        </div>

        {/* ── Right Panel ── */}
        <div style={{ flex: 1, display: 'flex', flexDirection: 'column', overflow: 'hidden', background: C.bg }}>

          {/* Tab bar */}
          <div style={{
            display: 'flex', gap: 8, padding: '12px 20px',
            borderBottom: `1px solid ${C.border}`, background: C.panel, flexShrink: 0,
            overflowX: 'auto',
          }}>
            {([
              { id: 'properties', icon: '📊', label: 'Properties' },
              { id: 'columns', icon: '📐', label: `Columns${selectedTable ? ` (${colCount})` : ''}` },
              { id: 'relationships', icon: '🔗', label: selectedTable ? `Relationships (${relCount})` : 'Relationships' },
              { id: 'erd', icon: '🗺', label: 'ERD Diagram' },
              { id: 'lineage', icon: '🔀', label: 'Lineage' },
            ] as { id: Tab; icon: string; label: string }[]).map(tab => (
              <Pill key={tab.id} active={activeTab === tab.id} onClick={() => setActiveTab(tab.id)}>
                <span>{tab.icon}</span>
                <span>{tab.label}</span>
              </Pill>
            ))}
          </div>

          {/* Tab content */}
          <div style={{ flex: 1, overflow: 'auto', animation: 'fadein 0.2s ease' }}>

            {/* Properties Tab */}
            {activeTab === 'properties' && (
              <div style={{ padding: 28, maxWidth: 720 }}>
                {!selectedTable ? (
                  <Empty icon="📊" title="Select a table" subtitle="Click any table in the left panel to view its properties." />
                ) : (
                  <div style={{ animation: 'fadein 0.2s ease' }}>
                    <div style={{ display: 'flex', alignItems: 'flex-start', gap: 16, marginBottom: 28 }}>
                      <div style={{
                        width: 56, height: 56, background: C.accentDim, borderRadius: 14,
                        display: 'flex', alignItems: 'center', justifyContent: 'center', fontSize: 28, flexShrink: 0,
                      }}>📋</div>
                      <div>
                        <h2 style={{ margin: 0, fontSize: 24, fontWeight: 800, color: C.text, fontFamily: 'monospace' }}>
                          {selectedTable.node_name}
                        </h2>
                        <div style={{ color: C.textMuted, fontSize: 13, marginTop: 4 }}>{selectedTable.qualified_path}</div>
                      </div>
                    </div>
                    <div style={{
                      background: C.panel, border: `1px solid ${C.border}`, borderRadius: 14,
                      overflow: 'hidden',
                    }}>
                      {[
                        ['Table Name', selectedTable.node_name],
                        ['Schema', selectedTable.schema],
                        ['Qualified Path', selectedTable.qualified_path || '—'],
                        ['Node ID', selectedTable.id],
                        ['Column Count', String(selectedTable.columns.length)],
                        ['Primary Keys', String(selectedTable.columns.filter(c => c.properties?.is_primary_key).length)],
                        ['Foreign Keys', String(selectedTable.columns.filter(c => c.properties?.is_foreign_key).length)],
                        ['Row Count', selectedTable.properties?.row_count ? String(selectedTable.properties.row_count) : '—'],
                        ['Description', selectedTable.properties?.description || '—'],
                      ].map(([k, v], i) => (
                        <div key={k} style={{
                          display: 'flex', alignItems: 'flex-start', padding: '14px 20px',
                          borderBottom: i < 8 ? `1px solid ${C.border}` : 'none',
                          background: i % 2 === 0 ? 'transparent' : 'rgba(255,255,255,0.01)',
                        }}>
                          <span style={{ color: C.textMuted, fontSize: 12, fontWeight: 700, textTransform: 'uppercase', letterSpacing: '0.05em', minWidth: 160, paddingTop: 2 }}>{k}</span>
                          <span style={{ color: C.text, fontSize: 14, fontFamily: 'monospace', wordBreak: 'break-all' }}>{v}</span>
                        </div>
                      ))}
                    </div>
                  </div>
                )}
              </div>
            )}

            {/* Columns Tab */}
            {activeTab === 'columns' && (
              <div style={{ padding: 24 }}>
                {!selectedTable ? (
                  <Empty icon="📐" title="Select a table" subtitle="Choose a table from the left panel to browse its columns." />
                ) : selectedTable.columns.length === 0 ? (
                  <Empty icon="📭" title="No columns found" subtitle="This table has no column metadata available." />
                ) : (
                  <div style={{ animation: 'fadein 0.2s ease' }}>
                    <div style={{ display: 'flex', alignItems: 'center', gap: 12, marginBottom: 20 }}>
                      <h2 style={{ margin: 0, fontSize: 18, fontWeight: 700, color: C.text, fontFamily: 'monospace' }}>
                        {selectedTable.node_name}
                      </h2>
                      <Badge label={`${selectedTable.columns.length} columns`} color={C.accent} />
                      <Badge label={`${selectedTable.columns.filter(c => c.properties?.is_primary_key).length} PK`} color={C.warning} />
                      <Badge label={`${selectedTable.columns.filter(c => c.properties?.is_foreign_key).length} FK`} color={C.blue} />
                    </div>
                    <div style={{
                      background: C.panel, border: `1px solid ${C.border}`, borderRadius: 14, overflow: 'hidden',
                    }}>
                      {/* Header */}
                      <div style={{
                        display: 'grid', gridTemplateColumns: '2fr 1.5fr 60px 60px 60px',
                        padding: '10px 20px', borderBottom: `1px solid ${C.border}`,
                        background: 'rgba(255,255,255,0.02)',
                      }}>
                        {['Column Name', 'Data Type', 'PK', 'FK', 'Nullable'].map(h => (
                          <span key={h} style={{ color: C.textMuted, fontSize: 11, fontWeight: 700, textTransform: 'uppercase', letterSpacing: '0.05em' }}>{h}</span>
                        ))}
                      </div>
                      {/* Rows */}
                      {selectedTable.columns.map((col, i) => {
                        const dt = col.properties?.data_type || '?';
                        const isPK = col.properties?.is_primary_key;
                        const isFK = col.properties?.is_foreign_key;
                        const nullable = col.properties?.is_nullable !== false;
                        return (
                          <div key={col.id} style={{
                            display: 'grid', gridTemplateColumns: '2fr 1.5fr 60px 60px 60px',
                            padding: '11px 20px',
                            borderBottom: i < selectedTable.columns.length - 1 ? `1px solid ${C.border}22` : 'none',
                            background: i % 2 === 0 ? 'transparent' : 'rgba(255,255,255,0.01)',
                            transition: 'background 0.15s',
                          }}>
                            <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
                              {isPK && <span title="Primary Key" style={{ fontSize: 12 }}>🔑</span>}
                              {isFK && !isPK && <span title="Foreign Key" style={{ fontSize: 12 }}>🔗</span>}
                              <span style={{ color: C.text, fontSize: 13, fontFamily: 'monospace', fontWeight: isPK ? 700 : 400 }}>
                                {col.node_name}
                              </span>
                            </div>
                            <div>
                              <span style={{
                                color: typeColor(dt), fontSize: 12, fontFamily: 'monospace',
                                background: `${typeColor(dt)}18`, padding: '2px 8px', borderRadius: 6,
                                border: `1px solid ${typeColor(dt)}33`,
                              }}>{dt}</span>
                            </div>
                            <div style={{ display: 'flex', alignItems: 'center' }}>
                              {isPK ? <span style={{ color: C.warning, fontSize: 14 }}>✓</span> : <span style={{ color: C.border }}>—</span>}
                            </div>
                            <div style={{ display: 'flex', alignItems: 'center' }}>
                              {isFK ? <span style={{ color: C.blue, fontSize: 14 }}>✓</span> : <span style={{ color: C.border }}>—</span>}
                            </div>
                            <div style={{ display: 'flex', alignItems: 'center' }}>
                              {nullable ? <span style={{ color: C.textMuted, fontSize: 14 }}>✓</span> : <span style={{ color: C.danger, fontSize: 14 }}>✗</span>}
                            </div>
                          </div>
                        );
                      })}
                    </div>
                  </div>
                )}
              </div>
            )}

            {/* Relationships Tab */}
            {activeTab === 'relationships' && (
              <div style={{ padding: 24 }}>
                {!selectedTable ? (
                  <Empty icon="🔗" title="Select a table" subtitle="Choose a table from the left panel to view its relationships." />
                ) : (
                  <>
                    {/* Header */}
                    <div style={{ display: 'flex', alignItems: 'center', gap: 16, marginBottom: 8 }}>
                      <h2 style={{ margin: 0, fontSize: 18, fontWeight: 700, color: C.text, fontFamily: 'monospace' }}>
                        {selectedTable.node_name}
                      </h2>
                      <Badge label={`${tableRelEdges.length} relationship${tableRelEdges.length !== 1 ? 's' : ''}`} color={C.purple} />
                      {(() => {
                        const nt = nodeTypeMap.get(selectedTable.node_type_id);
                        return nt ? <Badge label={nt.catalog_type_name} color={C.teal} /> : null;
                      })()}
                      <div style={{ flex: 1 }} />
                      <button
                        onClick={() => { setEditingEdge(null); setDialogOpen(true); }}
                        style={{
                          display: 'flex', alignItems: 'center', gap: 8,
                          padding: '8px 16px', borderRadius: 10, border: 'none',
                          background: C.accent, color: '#fff', cursor: 'pointer',
                          fontSize: 13, fontWeight: 700, boxShadow: C.accentGlow,
                        }}
                      >+ Add Relationship</button>
                    </div>
                    <div style={{ color: C.textMuted, fontSize: 12, marginBottom: 20, fontFamily: 'monospace' }}>
                      {selectedTable.qualified_path || `${selectedTable.schema}.${selectedTable.node_name}`}
                    </div>

                    {tableRelEdges.length === 0 ? (
                      <Empty icon="🔗" title="No relationships" subtitle={`${selectedTable.node_name} has no known foreign key or join relationships.`} />
                    ) : (
                      <div style={{ display: 'flex', flexDirection: 'column', gap: 12 }}>
                        {tableRelEdges.map((edge) => {
                          const isSrc = edge.source_node_id === selectedTable.id || selectedTable.columns.some(c => c.id === edge.source_node_id);
                          const otherNodeId = isSrc ? edge.target_node_id : edge.source_node_id;
                          const otherNodeObj = allNodeMap.get(otherNodeId);
                          const otherTable = tableMap.get(otherNodeId);
                          const otherNodeType = otherNodeObj ? nodeTypeMap.get(otherNodeObj.node_type_id) : undefined;
                          const thisNodeType = nodeTypeMap.get(selectedTable.node_type_id);
                          const isDeleting = deletingEdgeId === edge.id;
                          const relType = edge.relationship_type || edge.properties?.edge_type_name || 'relationship';
                          const cardinality = edge.properties?.cardinality;

                          const thisQPath = selectedTable.qualified_path || `${selectedTable.schema}.${selectedTable.node_name}`;
                          const otherQPath = otherNodeObj?.qualified_path || (otherTable ? `${otherTable.schema}.${otherTable.node_name}` : otherNodeId.slice(0, 16));
                          const otherName = otherNodeObj?.node_name ?? otherTable?.node_name ?? otherNodeId.slice(0, 8);

                          const relColor = relType === 'foreign_key' ? C.blue
                            : relType === 'PRIMARY_KEY' ? C.warning
                            : relType === 'table_relationship' ? C.purple
                            : C.textMuted;

                          return (
                            <div key={edge.id} style={{
                              background: C.panel, border: `1px solid ${C.border}`,
                              borderRadius: 14, overflow: 'hidden',
                              transition: 'all 0.2s', animation: 'fadein 0.2s ease',
                              opacity: isDeleting ? 0.5 : 1,
                              borderLeft: `3px solid ${relColor}`,
                            }}>
                              {/* Relationship header row */}
                              <div style={{
                                display: 'flex', alignItems: 'center', gap: 12,
                                padding: '14px 20px',
                              }}>
                                {/* Source table + node type */}
                                <div style={{ display: 'flex', flexDirection: 'column', gap: 4 }}>
                                  <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
                                    <span style={{
                                      color: isSrc ? C.accent : C.text,
                                      fontFamily: 'monospace', fontWeight: 700, fontSize: 14,
                                    }}>
                                      📋 {isSrc ? selectedTable.node_name : otherName}
                                    </span>
                                    {(isSrc ? thisNodeType : otherNodeType) && (
                                      <Badge label={(isSrc ? thisNodeType : otherNodeType)!.catalog_type_name} color={C.teal} />
                                    )}
                                  </div>
                                  <span style={{ color: C.textMuted, fontSize: 11, fontFamily: 'monospace' }}>
                                    {isSrc ? thisQPath : otherQPath}
                                  </span>
                                </div>

                                {/* Arrow + cardinality */}
                                <div style={{ display: 'flex', flexDirection: 'column', alignItems: 'center', gap: 4, flex: 1 }}>
                                  <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
                                    {cardinality && <Badge label={cardinality} color={C.orange} />}
                                  </div>
                                  <span style={{ color: relColor, fontSize: 20 }}>
                                    {isSrc ? '→' : '←'}
                                  </span>
                                </div>

                                {/* Target table + node type */}
                                <div style={{ display: 'flex', flexDirection: 'column', gap: 4, alignItems: 'flex-end' }}>
                                  <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
                                    {(isSrc ? otherNodeType : thisNodeType) && (
                                      <Badge label={(isSrc ? otherNodeType : thisNodeType)!.catalog_type_name} color={C.teal} />
                                    )}
                                    <span style={{
                                      color: !isSrc ? C.accent : C.text,
                                      fontFamily: 'monospace', fontWeight: 700, fontSize: 14,
                                    }}>
                                      📋 {isSrc ? otherName : selectedTable.node_name}
                                    </span>
                                  </div>
                                  <span style={{ color: C.textMuted, fontSize: 11, fontFamily: 'monospace' }}>
                                    {isSrc ? otherQPath : thisQPath}
                                  </span>
                                </div>

                                {/* Actions */}
                                <div style={{ display: 'flex', gap: 8, marginLeft: 8 }}>
                                  <button
                                    onClick={() => { setEditingEdge(edge); setDialogOpen(true); }}
                                    style={{
                                      padding: '5px 12px', borderRadius: 7, border: `1px solid ${C.border}`,
                                      background: 'transparent', color: C.textMuted, cursor: 'pointer', fontSize: 12,
                                    }}
                                  >✏️</button>
                                  <button
                                    onClick={() => handleDeleteRelationship(edge.id)}
                                    disabled={isDeleting}
                                    style={{
                                      padding: '5px 12px', borderRadius: 7, border: `1px solid ${C.danger}44`,
                                      background: 'rgba(239,68,68,0.08)', color: C.danger, cursor: 'pointer', fontSize: 12,
                                    }}
                                  >{isDeleting ? '…' : '🗑'}</button>
                                </div>
                              </div>
                            </div>
                          );
                        })}
                      </div>
                    )}
                  </>
                )}
              </div>
            )}

            {/* ERD Tab */}
            {activeTab === 'erd' && (
              <div style={{ height: '100%', position: 'relative' }}>
                {loading ? <Spinner /> : tables.length === 0 ? (
                  <Empty icon="🗺" title="No tables to display" subtitle="Load a datasource with tables to see the ERD diagram." />
                ) : (
                  <>
                    <div style={{ position: 'absolute', top: 12, right: 12, zIndex: 10 }}>
                      <button
                        onClick={() => setShowErdColumns(!showErdColumns)}
                        style={{
                          padding: '6px 12px', borderRadius: 8, border: `1px solid ${C.border}`,
                          background: showErdColumns ? C.accent : C.panel, color: showErdColumns ? '#fff' : C.text,
                          cursor: 'pointer', fontSize: 12, fontWeight: 600, display: 'flex', alignItems: 'center', gap: 6,
                        }}
                      >
                        {showErdColumns ? '🙈 Hide Columns' : '👁 Show Columns'}
                      </button>
                    </div>
                    <ReactFlow
                      nodes={flowNodes}
                      edges={flowEdges}
                      onNodesChange={onFlowNodesChange}
                      onEdgesChange={onFlowEdgesChange}
                      onNodeClick={(_, node) => { setSelectedTableId(node.id); setActiveTab('columns'); }}
                      nodeTypes={nodeTypes}
                      fitView
                      style={{ background: C.bg }}
                      proOptions={{ hideAttribution: true }}
                      className={showErdColumns ? '' : 'hide-columns'}
                    >
                      <Background variant={BackgroundVariant.Dots} color={C.border} gap={24} size={1} />
                      <Controls style={{ background: C.panel, border: `1px solid ${C.border}`, borderRadius: 10 }} />
                      <MiniMap
                        nodeColor={() => C.accent}
                        style={{ background: C.panel, border: `1px solid ${C.border}`, borderRadius: 10 }}
                      />
                    </ReactFlow>
                  </>
                )}
              </div>
            )}

            {/* Lineage Tab */}
            {activeTab === 'lineage' && (
              <div style={{ height: '100%', position: 'relative' }}>
                {!selectedTable ? (
                  <Empty icon="🔀" title="Select a table" subtitle="Pick a table from the left panel to view its upstream and downstream lineage." />
                ) : lineageNodes.length === 0 ? (
                  <Empty icon="🌐" title="No lineage data" subtitle={`${selectedTable.node_name} has no known relationships to visualize as lineage.`} />
                ) : (
                  <>
                    <div style={{ position: 'absolute', top: 12, right: 12, zIndex: 10 }}>
                      <button
                        onClick={() => setShowLineageColumns(!showLineageColumns)}
                        style={{
                          padding: '6px 12px', borderRadius: 8, border: `1px solid ${C.border}`,
                          background: showLineageColumns ? C.accent : C.panel, color: showLineageColumns ? '#fff' : C.text,
                          cursor: 'pointer', fontSize: 12, fontWeight: 600, display: 'flex', alignItems: 'center', gap: 6,
                        }}
                      >
                        {showLineageColumns ? '🙈 Hide Columns' : '👁 Show Columns'}
                      </button>
                    </div>
                    <ReactFlow
                      nodes={lineageNodes}
                      edges={lineageEdges}
                      onNodesChange={onLineageNodesChange}
                      onEdgesChange={onLineageEdgesChange}
                      onNodeClick={(_, node) => { setSelectedTableId(node.id); }}
                      fitView
                      style={{ background: C.bg }}
                      proOptions={{ hideAttribution: true }}
                      className={showLineageColumns ? '' : 'hide-columns'}
                    >
                      <Background variant={BackgroundVariant.Lines} color={C.border} gap={32} />
                      <Controls style={{ background: C.panel, border: `1px solid ${C.border}`, borderRadius: 10 }} />
                    </ReactFlow>
                  </>
                )}
              </div>
            )}

          </div>
        </div>
      </div>

      {/* Relationship Dialog */}
      <RelationshipDialog
        open={dialogOpen}
        tables={tables}
        existing={editingEdge}
        onClose={() => { setDialogOpen(false); setEditingEdge(null); }}
        onSave={handleSaveRelationship}
      />
    </div>
  );
};

export default SchemaExplorerPage;
