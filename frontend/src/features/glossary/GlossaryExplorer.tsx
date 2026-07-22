import * as React from 'react';
import { useState, useMemo, useEffect, useCallback } from 'react';
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
import { useTenant } from '../../contexts/TenantContext';
import { useAccess } from '../../contexts/AccessContext';
import { useApiQuery } from '../../hooks/useApiQuery';
import apiClient from '../../utils/apiClient';
import GenericCatalogNodePicker from '../../components/common/GenericCatalogNodePicker';
import EditSemanticTermDialog from '../../components/EditSemanticTermDialog';
import { IconButton, Tooltip } from '@mui/material';
import { 
  Edit as EditIcon, 
  Delete as DeleteIcon, 
  Add as AddIcon, 
  AutoFixHigh as AutoFixIcon 
} from '@mui/icons-material';
import { useDeleteTerm } from '../../api/glossary';
import { RelationshipList } from '../../components/glossary/RelationshipList';

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

// ERD Custom Node
const ErdTableNode: React.FC<{ data: any }> = ({ data }) => (
  <div style={{
    background: C.panel, border: `1px solid ${data.selected ? C.accent : C.border}`,
    borderRadius: 10, minWidth: 180, overflow: 'hidden',
    boxShadow: data.selected ? C.accentGlow : '0 4px 24px rgba(0,0,0,0.5)',
    transition: 'all 0.2s ease',
  }}>
    <Handle type="target" position={Position.Left} style={{ background: C.accent, width: 8, height: 8, border: 'none' }} />
    <div style={{ background: data.selected ? C.accent : '#1E2130', padding: '8px 12px', borderBottom: `1px solid ${C.border}` }}>
      <div style={{ color: C.text, fontWeight: 700, fontSize: 12, fontFamily: 'monospace' }}>
        {data.icon || '📌'} {data.label}
      </div>
      <div style={{ color: C.textMuted, fontSize: 10, marginTop: 2 }}>{data.subtitle}</div>
    </div>
    <Handle type="source" position={Position.Right} style={{ background: C.accent, width: 8, height: 8, border: 'none' }} />
  </div>
);

const nodeTypes = { erdTable: ErdTableNode };

const parsePath = (path: string) => {
  if (!path) return { datasource: 'N/A', schema: 'N/A', table: 'N/A', column: 'N/A' };
  
  // Normalize slash paths to dot paths (e.g. /public/employees/address -> public.employees.address)
  let normalized = path;
  if (normalized.includes('/')) {
    normalized = normalized.split('/').filter(Boolean).join('.');
  }
  
  const parts = normalized.split('.');
  if (parts.length === 4) {
    return { datasource: parts[0], schema: parts[1], table: parts[2], column: parts[3] };
  } else if (parts.length === 3) {
    return { datasource: 'N/A', schema: parts[0], table: parts[1], column: parts[2] };
  } else if (parts.length === 2) {
    return { datasource: 'N/A', schema: 'N/A', table: parts[0], column: parts[1] };
  }
  return { datasource: 'N/A', schema: 'N/A', table: 'N/A', column: path };
};

export default function GlossaryExplorer() {
  const { tenant } = useTenant();
  const { currentTenant, isPlatformOperator, accessLevel } = useAccess();
  const tenantId = currentTenant?.id ?? tenant?.id;

  const isWriter = isPlatformOperator || accessLevel === 'tenant_admin' || accessLevel === 'platform_operator';
  const canCreate = isWriter;
  const canUpdate = isWriter;
  const canDelete = isWriter;

  const deleteTermMutation = useDeleteTerm();

  const handleDeleteTerm = async (term: any) => {
    if (!window.confirm(`Are you sure you want to delete the term "${term.node_name}"? This action cannot be undone.`)) {
      return;
    }
    try {
      await deleteTermMutation.mutateAsync(term.id);
      setSearchParams({});
    } catch (e: any) {
      console.error(e);
      // Collect dependencies from any available source
      let deps: any[] = [];
      if (e?.code === 'BO_DEPENDENCIES_BLOCK_DELETION' || e?.error === 'BO_DEPENDENCIES_BLOCK_DELETION') {
        deps = e?.dependencies ?? [];
      }
      if (deps.length === 0) {
        deps = e?.dependencyReport?.dependencies ?? e?.dependencies ?? [];
      }
      // If we have dependency info, show detailed message
      if (deps.length > 0) {
        const boNames = [...new Set(deps.map((d: any) => d.bo_name))].join(', ');
        alert(
          `Cannot delete "${term.node_name}":\n\n` +
          `This term is linked to ${deps.length} BO field(s) in:\n${boNames || 'Unknown BOs'}\n\n` +
          `Details:\n${deps.map((d: any) => `- ${d.bo_name}: ${d.ref_detail || d.bo_key}`).join('\n')}\n\n` +
          `Please unlink these fields from the Business Object before deleting the term.`
        );
        return;
      }
      // No dependency info available - show generic error, avoiding double-prefix
      let msg = e?.message || String(e);
      // Strip common prefixes to avoid "Failed to delete term: Failed to delete term"
      if (msg.startsWith('Failed to delete term: ')) {
        msg = msg.substring('Failed to delete term: '.length);
      } else if (msg === 'Failed to delete term' || msg === 'Failed to delete term: Failed to delete term') {
        msg = 'Delete operation failed. Check console for details.';
      }
      alert(`Failed to delete term: ${msg}`);
    }
  };

  const [searchParams, setSearchParams] = useSearchParams();
  const selectedId = searchParams.get('id');

  const [searchTerm, setSearchTerm] = useState('');
  const [activeTab, setActiveTab] = useState<'properties' | 'technical' | 'relationships' | 'lineage'>('properties');
  const [isSemOpen, setIsSemOpen] = useState(true);
  const [isBusOpen, setIsBusOpen] = useState(true);

  // Modals
  const [isCreateSemModalOpen, setIsCreateSemModalOpen] = useState(false);
  const [isCreateBusModalOpen, setIsCreateBusModalOpen] = useState(false);
  const [isGenModalOpen, setIsGenModalOpen] = useState(false);
  const [isEditSemModalOpen, setIsEditSemModalOpen] = useState(false);

  // Forms
  const [semName, setSemName] = useState('');
  const [semDesc, setSemDesc] = useState('');
  const [busName, setBusName] = useState('');
  const [busDesc, setBusDesc] = useState('');
  const [busLinkSem, setBusLinkSem] = useState('');

  // Fetching
  const { data: semTermsRaw, isLoading: semLoading, refetch: refetchSem } = useApiQuery<any[]>(
    `/api/glossary/semantic-terms?tenant_id=${tenantId}`,
    { enabled: !!tenantId }
  );

  const { data: busTermsRaw, isLoading: busLoading, refetch: refetchBus } = useApiQuery<any[]>(
    `/api/glossary/business-terms?tenant_id=${tenantId}`,
    { enabled: !!tenantId }
  );

  const { data: nodeGraph, isLoading: graphLoading, refetch: refetchGraph } = useApiQuery<any>(
    `/api/glossary/node-graph?node_id=${selectedId}&tenant_id=${tenantId}`,
    { enabled: !!tenantId && !!selectedId }
  );

  // Technical assets — dedicated endpoint with proper edge direction
  const { data: technicalAssetsRaw, isLoading: taLoading, refetch: refetchTa } = useApiQuery<any>(
    `/api/glossary/technical-assets?node_id=${selectedId}&tenant_id=${tenantId}`,
    { enabled: !!tenantId && !!selectedId }
  );

  // Technical assets UI state
  const [taSearch, setTaSearch] = useState('');
  const [taDsFilter, setTaDsFilter] = useState<string>('ALL');
  const [isAddMappingOpen, setIsAddMappingOpen] = useState(false);
  const [mappingColSearch, setMappingColSearch] = useState('');
  const [selectedColIds, setSelectedColIds] = useState<string[]>([]);

  const { data: allColumns, isLoading: columnsLoading } = useApiQuery<any[]>(
    `/api/rest/catalog-nodes?node_type_id=a64c1011-16e8-4ddf-b447-363bf8e15c9a&tenant_id=${tenantId}&limit=10000`,
    { enabled: !!tenantId && (isGenModalOpen || isAddMappingOpen) }
  );

  const semTerms = useMemo(() => {
    const list = Array.isArray(semTermsRaw) ? semTermsRaw : (semTermsRaw as any)?.data ?? [];
    const sorted = [...list].sort((a, b) => (a.node_name || '').localeCompare(b.node_name || ''));
    if (!searchTerm) return sorted;
    return sorted.filter((t: any) => t.node_name?.toLowerCase().includes(searchTerm.toLowerCase()));
  }, [semTermsRaw, searchTerm]);

  const busTerms = useMemo(() => {
    const list = Array.isArray(busTermsRaw) ? busTermsRaw : (busTermsRaw as any)?.data ?? [];
    const sorted = [...list].sort((a, b) => (a.node_name || '').localeCompare(b.node_name || ''));
    if (!searchTerm) return sorted;
    return sorted.filter((t: any) => t.node_name?.toLowerCase().includes(searchTerm.toLowerCase()));
  }, [busTermsRaw, searchTerm]);

  const selectedTerm = useMemo(() => {
    if (!selectedId) return null;
    const semList = Array.isArray(semTermsRaw) ? semTermsRaw : (semTermsRaw as any)?.data ?? [];
    const busList = Array.isArray(busTermsRaw) ? busTermsRaw : (busTermsRaw as any)?.data ?? [];
    let found = semList.find((t: any) => t.id === selectedId);
    if (found) return { ...found, _kind: 'semantic' };
    found = busList.find((t: any) => t.id === selectedId);
    if (found) return { ...found, _kind: 'business' };
    return null;
  }, [selectedId, semTermsRaw, busTermsRaw]);

  // Mutations
  const createSemantic = async () => {
    if (!tenantId || !semName) return;
    try {
      await apiClient(`/api/glossary/terms?tenant_id=${tenantId}`, {
        method: 'POST',
        body: JSON.stringify({
          node_name: semName,
          description: semDesc,
          catalog_type: 'semantic_term'
        })
      });
      setIsCreateSemModalOpen(false);
      setSemName('');
      setSemDesc('');
      refetchSem();
    } catch (e) {
      console.error(e);
      alert('Error creating semantic term');
    }
  };

  const createBusiness = async () => {
    if (!tenantId || !busName) return;
    try {
      const res = await apiClient(`/api/glossary/terms?tenant_id=${tenantId}`, {
        method: 'POST',
        body: JSON.stringify({
          node_name: busName,
          description: busDesc,
          catalog_type: 'business_term'
        })
      });
      if (busLinkSem && res?.id) {
        await apiClient(`/api/glossary/edges?tenant_id=${tenantId}`, {
          method: 'POST',
          body: JSON.stringify({
            subject_node_id: busLinkSem,
            object_node_id: res.id,
            edge_type_id: '3be9d6ae-1598-4628-a3dd-b606921a9193',
            tenant_id: tenantId
          })
        });
      }
      setIsCreateBusModalOpen(false);
      setBusName('');
      setBusDesc('');
      setBusLinkSem('');
      refetchBus();
    } catch (e) {
      console.error(e);
      alert('Error creating business term');
    }
  };

  const generateColumns = async (selectedGroups: any[]) => {
    if (!tenantId || selectedGroups.length === 0) return;
    try {
      for (const group of selectedGroups) {
        await apiClient(`/api/glossary/generate-semantic-terms?tenant_id=${tenantId}`, {
          method: 'POST',
          body: JSON.stringify({
            name: group.suggestedName,
            column_ids: group.columns.map((c: any) => c.id)
          })
        });
      }
      setIsGenModalOpen(false);
      refetchSem();
    } catch (e) {
      console.error(e);
      alert('Error generating semantic terms');
    }
  };

  // Derived technical assets with search + datasource filter
  const technicalAssets: any[] = useMemo(() => {
    const raw = (technicalAssetsRaw as any)?.data ?? [];
    return raw.filter((a: any) => {
      const matchSearch = !taSearch ||
        a.node_name?.toLowerCase().includes(taSearch.toLowerCase()) ||
        a.qualified_path?.toLowerCase().includes(taSearch.toLowerCase());
      const matchDs = taDsFilter === 'ALL' || a.datasource === taDsFilter || a.datasource_id === taDsFilter;
      return matchSearch && matchDs;
    });
  }, [technicalAssetsRaw, taSearch, taDsFilter]);

  // Unique datasource facets
  const datasourceFacets: { label: string; value: string }[] = useMemo(() => {
    const raw = (technicalAssetsRaw as any)?.data ?? [];
    const seen = new Set<string>();
    const facets: { label: string; value: string }[] = [{ label: 'All', value: 'ALL' }];
    raw.forEach((a: any) => {
      const key = a.datasource || a.datasource_id || 'Unknown';
      if (!seen.has(key)) {
        seen.add(key);
        facets.push({ label: a.datasource || a.datasource_id || 'Unknown', value: key });
      }
    });
    return facets;
  }, [technicalAssetsRaw]);

  // Remove a technical asset mapping (delete has_context edge)
  const removeMapping = async (edgeId: string) => {
    if (!window.confirm('Remove this column mapping from the semantic term?')) return;
    try {
      await apiClient(`/api/glossary/technical-assets/${edgeId}?tenant_id=${tenantId}`, { method: 'DELETE' });
      refetchTa();
    } catch (e) {
      console.error(e);
      alert('Failed to remove mapping');
    }
  };

  // Add technical asset mapping(s) (create has_context edge)
  const addMappings = async (columnNodeIds: string[]) => {
    if (!selectedId || columnNodeIds.length === 0) return;
    try {
      await apiClient(`/api/glossary/technical-assets?tenant_id=${tenantId}`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          semantic_term_id: selectedId,
          column_node_ids: columnNodeIds,
        }),
      });
      refetchTa();
      setIsAddMappingOpen(false);
      setMappingColSearch('');
      setSelectedColIds([]);
    } catch (e) {
      console.error(e);
      alert('Failed to add mappings');
    }
  };

  const toggleColumnSelection = (id: string) => {
    setSelectedColIds(prev => 
      prev.includes(id) ? prev.filter(x => x !== id) : [...prev, id]
    );
  };

  // Lineage nodes/edges
  const [nodes, setNodes, onNodesChange] = useNodesState([]);
  const [edges, setEdges, onEdgesChange] = useEdgesState([]);

  useEffect(() => {
    if (activeTab !== 'lineage' || !nodeGraph || !selectedTerm) return;
    const flowNodes: any[] = [];
    const flowEdges: any[] = [];
    
    const centerX = 400;
    const centerY = 300;
    
    // Add selected node
    flowNodes.push({
      id: selectedTerm.id,
      type: 'erdTable',
      position: { x: centerX, y: centerY },
      data: {
        label: selectedTerm.node_name,
        subtitle: selectedTerm.catalog_type_name || (selectedTerm._kind === 'semantic' ? 'Semantic Term' : 'Business Term'),
        selected: true,
        icon: selectedTerm._kind === 'semantic' ? '🧠' : '💼'
      }
    });

    const allNodes: any[] = [
      ...(nodeGraph.node ? [nodeGraph.node] : []),
      ...(nodeGraph.connected_nodes ?? nodeGraph.nodes ?? []),
    ];
    const relatedNodes = allNodes.filter((n: any) => n.id !== selectedTerm.id);
    const radius = 250;
    relatedNodes.forEach((n: any, idx: number) => {
      const angle = (idx / relatedNodes.length) * Math.PI * 2;
      flowNodes.push({
        id: n.id,
        type: 'erdTable',
        position: { x: centerX + Math.cos(angle) * radius, y: centerY + Math.sin(angle) * radius },
        data: {
          label: n.node_name || n.id,
          subtitle: n.qualified_path || n.catalog_type_name,
          selected: false,
          icon: '📄'
        }
      });
    });

    nodeGraph.edges.forEach((e: any) => {
      flowEdges.push({
        id: e.id,
        source: e.source_node_id,
        target: e.target_node_id,
        animated: false,
        label: e.relationship_type || e.edge_type_name || '',
        labelStyle: { fill: '#fff', fontWeight: 600, fontSize: 10 },
        labelBgStyle: { fill: C.panel, fillOpacity: 0.9 },
        labelBgPadding: [4, 2],
        labelBgBorderRadius: 4,
        style: { stroke: C.accent, strokeWidth: 2 },
        markerEnd: { type: MarkerType.ArrowClosed, color: C.accent }
      });
    });

    setNodes(flowNodes);
    setEdges(flowEdges);
  }, [activeTab, nodeGraph, selectedTerm]);

  // Wizard Generation State
  const [genGroups, setGenGroups] = useState<any[]>([]);
  const [selectedGenGroups, setSelectedGenGroups] = useState<Set<string>>(new Set());

  useEffect(() => {
    if (!isGenModalOpen || !allColumns) return;
    const colList = Array.isArray(allColumns) ? allColumns : (allColumns as any)?.data ?? [];
    if (colList.length === 0) return;
    const groups: Record<string, any[]> = {};
    colList.forEach((c: any) => {
      if (!c.qualified_path) return;
      const parts = c.qualified_path.split('.');
      const colName = parts[parts.length - 1];
      const baseName = colName.toLowerCase().replace(/_/g, ' ').replace(/\b\w/g, (l: string) => l.toUpperCase());
      if (!groups[baseName]) groups[baseName] = [];
      groups[baseName].push(c);
    });

    setGenGroups(Object.entries(groups).map(([name, cols]) => ({ suggestedName: name, columns: cols })));
    setSelectedGenGroups(new Set());
  }, [allColumns, isGenModalOpen]);

  const inputStyle = {
    background: '#1E2130', border: `1px solid ${C.border}`, color: C.text,
    padding: '8px 12px', borderRadius: 6, width: '100%', marginBottom: 12, outline: 'none'
  };

  return (
    <div style={{ display: 'flex', height: '100vh', width: '100%', background: C.bg, color: C.text, fontFamily: 'system-ui, sans-serif' }}>
      
      {/* Modals */}
      {isCreateSemModalOpen && (
        <div style={{ position: 'fixed', inset: 0, background: 'rgba(0,0,0,0.7)', zIndex: 100, display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
          <div style={{ background: C.panel, padding: 24, borderRadius: 12, width: 400, border: `1px solid ${C.border}` }}>
            <h2 style={{ margin: '0 0 16px 0' }}>New Semantic Term</h2>
            <input placeholder="Name" style={inputStyle} value={semName} onChange={e => setSemName(e.target.value)} />
            <textarea placeholder="Description" style={{ ...inputStyle, height: 100 }} value={semDesc} onChange={e => setSemDesc(e.target.value)} />
            <div style={{ display: 'flex', gap: 12, justifyContent: 'flex-end', marginTop: 12 }}>
              <button onClick={() => setIsCreateSemModalOpen(false)} style={{ background: 'transparent', color: C.text, border: 'none', cursor: 'pointer' }}>Cancel</button>
              <button onClick={createSemantic} style={{ background: C.accent, color: '#fff', border: 'none', padding: '6px 16px', borderRadius: 6, cursor: 'pointer' }}>Save</button>
            </div>
          </div>
        </div>
      )}

      {isCreateBusModalOpen && (
        <div style={{ position: 'fixed', inset: 0, background: 'rgba(0,0,0,0.7)', zIndex: 100, display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
          <div style={{ background: C.panel, padding: 24, borderRadius: 12, width: 400, border: `1px solid ${C.border}` }}>
            <h2 style={{ margin: '0 0 16px 0' }}>New Business Term</h2>
            <input placeholder="Name" style={inputStyle} value={busName} onChange={e => setBusName(e.target.value)} />
            <textarea placeholder="Description" style={{ ...inputStyle, height: 100 }} value={busDesc} onChange={e => setBusDesc(e.target.value)} />
            <select style={inputStyle} value={busLinkSem} onChange={e => setBusLinkSem(e.target.value)}>
              <option value="">Link to Semantic Term (Optional)</option>
              {(Array.isArray(semTermsRaw) ? semTermsRaw : (semTermsRaw as any)?.data ?? []).map((t: any) => (
                <option key={t.id} value={t.id}>{t.node_name}</option>
              ))}
            </select>
            <div style={{ display: 'flex', gap: 12, justifyContent: 'flex-end', marginTop: 12 }}>
              <button onClick={() => setIsCreateBusModalOpen(false)} style={{ background: 'transparent', color: C.text, border: 'none', cursor: 'pointer' }}>Cancel</button>
              <button onClick={createBusiness} style={{ background: C.accent, color: '#fff', border: 'none', padding: '6px 16px', borderRadius: 6, cursor: 'pointer' }}>Save</button>
            </div>
          </div>
        </div>
      )}

      {isGenModalOpen && (
        <div style={{ position: 'fixed', inset: 0, background: 'rgba(0,0,0,0.7)', zIndex: 100, display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
          <div style={{ background: C.panel, padding: 24, borderRadius: 12, width: 800, maxHeight: '80vh', overflow: 'auto', border: `1px solid ${C.border}` }}>
            <h2 style={{ margin: '0 0 16px 0' }}>Generate Semantic Terms from Columns</h2>
            {columnsLoading ? <Spinner /> : (
              <table style={{ width: '100%', textAlign: 'left', borderCollapse: 'collapse' }}>
                <thead>
                  <tr style={{ borderBottom: `1px solid ${C.border}` }}>
                    <th style={{ padding: '8px' }}><input type="checkbox" onChange={e => {
                      if (e.target.checked) setSelectedGenGroups(new Set(genGroups.map(g => g.suggestedName)));
                      else setSelectedGenGroups(new Set());
                    }} /></th>
                    <th style={{ padding: '8px' }}>Suggested Name</th>
                    <th style={{ padding: '8px' }}>Columns Count</th>
                  </tr>
                </thead>
                <tbody>
                  {genGroups.map((g, i) => (
                    <tr key={i} style={{ borderBottom: `1px solid ${C.border}` }}>
                      <td style={{ padding: '8px' }}>
                        <input type="checkbox" checked={selectedGenGroups.has(g.suggestedName)} onChange={e => {
                          const next = new Set(selectedGenGroups);
                          if (e.target.checked) next.add(g.suggestedName); else next.delete(g.suggestedName);
                          setSelectedGenGroups(next);
                        }} />
                      </td>
                      <td style={{ padding: '8px' }}>
                        <input style={{ ...inputStyle, marginBottom: 0, width: 'auto' }} value={g.suggestedName} onChange={e => {
                          const next = [...genGroups];
                          next[i].suggestedName = e.target.value;
                          setGenGroups(next);
                        }} />
                      </td>
                      <td style={{ padding: '8px' }}>{g.columns.length}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            )}
            <div style={{ display: 'flex', gap: 12, justifyContent: 'flex-end', marginTop: 24 }}>
              <button onClick={() => setIsGenModalOpen(false)} style={{ background: 'transparent', color: C.text, border: 'none', cursor: 'pointer' }}>Cancel</button>
              <button onClick={() => generateColumns(genGroups.filter(g => selectedGenGroups.has(g.suggestedName)))} style={{ background: C.accent, color: '#fff', border: 'none', padding: '6px 16px', borderRadius: 6, cursor: 'pointer' }}>Create Selected</button>
            </div>
          </div>
        </div>
      )}

      {/* Sidebar */}
      <div style={{ width: 280, minWidth: 280, background: C.sidebar, borderRight: `1px solid ${C.border}`, display: 'flex', flexDirection: 'column' }}>
        <div style={{ padding: 16, borderBottom: `1px solid ${C.border}` }}>
          <input 
            placeholder="Search terms..." 
            value={searchTerm} 
            onChange={e => setSearchTerm(e.target.value)}
            style={{ width: '100%', background: '#1E2130', border: `1px solid ${C.border}`, borderRadius: 6, padding: '8px 12px', color: C.text, outline: 'none' }} 
          />
        </div>
        
        <div style={{ flex: 1, overflowY: 'auto' }}>
          <div style={{ padding: '12px 16px', borderBottom: `1px solid ${C.border}` }}>
            <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', cursor: 'pointer' }} onClick={() => setIsSemOpen(!isSemOpen)}>
              <span style={{ fontWeight: 600, fontSize: 13, textTransform: 'uppercase', color: C.textMuted }}>Semantic Terms</span>
              <Badge label={String(semTerms.length)} color={C.textMuted} />
            </div>
            {isSemOpen && (
              <div style={{ marginTop: 8 }}>
                {semLoading && <Spinner />}
                {!semLoading && semTerms.length === 0 && <div style={{ fontSize: 12, color: C.textMuted }}>No terms found.</div>}
                {semTerms.map((t: any) => (
                  <div key={t.id} onClick={() => setSearchParams({ id: t.id })} style={{
                    padding: '6px 8px', borderRadius: 4, cursor: 'pointer', fontSize: 13,
                    background: selectedId === t.id ? C.accentDim : 'transparent',
                    color: selectedId === t.id ? C.accent : C.text,
                    borderLeft: `2px solid ${selectedId === t.id ? C.accent : 'transparent'}`
                  }}>
                    🧠 {t.node_name}
                  </div>
                ))}
              </div>
            )}
          </div>

          <div style={{ padding: '12px 16px' }}>
            <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', cursor: 'pointer' }} onClick={() => setIsBusOpen(!isBusOpen)}>
              <span style={{ fontWeight: 600, fontSize: 13, textTransform: 'uppercase', color: C.textMuted }}>Business Terms</span>
              <Badge label={String(busTerms.length)} color={C.textMuted} />
            </div>
            {isBusOpen && (
              <div style={{ marginTop: 8 }}>
                {busLoading && <Spinner />}
                {!busLoading && busTerms.length === 0 && <div style={{ fontSize: 12, color: C.textMuted }}>No terms found.</div>}
                {busTerms.map((t: any) => (
                  <div key={t.id} onClick={() => setSearchParams({ id: t.id })} style={{
                    padding: '6px 8px', borderRadius: 4, cursor: 'pointer', fontSize: 13,
                    background: selectedId === t.id ? C.accentDim : 'transparent',
                    color: selectedId === t.id ? C.accent : C.text,
                    borderLeft: `2px solid ${selectedId === t.id ? C.accent : 'transparent'}`
                  }}>
                    💼 {t.node_name}
                  </div>
                ))}
              </div>
            )}
          </div>
        </div>
      </div>

      {/* Main Panel */}
      <div style={{ flex: 1, display: 'flex', flexDirection: 'column', background: C.bg }}>
        
        {/* Topbar */}
        <div style={{ height: 60, borderBottom: `1px solid ${C.border}`, display: 'flex', alignItems: 'center', padding: '0 24px', justifyContent: 'space-between' }}>
          <h1 style={{ margin: 0, fontSize: 18, fontWeight: 600 }}>Glossary Explorer</h1>
          {canCreate && (
            <div style={{ display: 'flex', gap: 12 }}>
              <button 
                onClick={() => setIsGenModalOpen(true)} 
                style={{ display: 'flex', alignItems: 'center', gap: 6, background: '#1E2130', border: `1px solid ${C.border}`, color: C.text, padding: '6px 12px', borderRadius: 6, cursor: 'pointer', fontSize: 13 }}
              >
                <AutoFixIcon sx={{ fontSize: 16 }} /> Generate from Columns
              </button>
              <button 
                onClick={() => setIsCreateSemModalOpen(true)} 
                style={{ display: 'flex', alignItems: 'center', gap: 6, background: C.accentDim, border: `1px solid ${C.accent}`, color: C.accent, padding: '6px 12px', borderRadius: 6, cursor: 'pointer', fontSize: 13 }}
              >
                <AddIcon sx={{ fontSize: 16 }} /> Semantic Term
              </button>
              <button 
                onClick={() => setIsCreateBusModalOpen(true)} 
                style={{ display: 'flex', alignItems: 'center', gap: 6, background: C.accentDim, border: `1px solid ${C.accent}`, color: C.accent, padding: '6px 12px', borderRadius: 6, cursor: 'pointer', fontSize: 13 }}
              >
                <AddIcon sx={{ fontSize: 16 }} /> Business Term
              </button>
            </div>
          )}
        </div>

        {/* Detail View */}
        {!selectedTerm ? (
          <Empty icon="📖" title="Select a term" subtitle="Choose a semantic or business term from the sidebar to view details." />
        ) : (
          <div style={{ flex: 1, display: 'flex', flexDirection: 'column' }}>
            <div style={{ padding: 24, borderBottom: `1px solid ${C.border}`, background: C.panel }}>
              <div style={{ display: 'flex', alignItems: 'center', gap: 12, marginBottom: 8 }}>
                <span style={{ fontSize: 24 }}>{selectedTerm._kind === 'semantic' ? '🧠' : '💼'}</span>
                <h2 style={{ margin: 0, fontSize: 20 }}>{selectedTerm.node_name}</h2>
                <Badge label={selectedTerm._kind === 'semantic' ? 'Semantic' : 'Business'} color={C.accent} />
                <div style={{ marginLeft: 'auto', display: 'flex', gap: 8, alignItems: 'center' }}>
                  {canUpdate && (
                    <Tooltip title="Edit Term">
                      <IconButton 
                        onClick={() => setIsEditSemModalOpen(true)}
                        size="small"
                        sx={{ color: C.text, '&:hover': { color: C.accent, background: 'rgba(255,255,255,0.05)' } }}
                      >
                        <EditIcon fontSize="small" />
                      </IconButton>
                    </Tooltip>
                  )}
                  {canDelete && (
                    <Tooltip title="Delete Term">
                      <IconButton 
                        onClick={() => handleDeleteTerm(selectedTerm)}
                        size="small"
                        sx={{ color: C.danger, '&:hover': { color: '#FF6B6B', background: 'rgba(239,68,68,0.1)' } }}
                      >
                        <DeleteIcon fontSize="small" />
                      </IconButton>
                    </Tooltip>
                  )}
                </div>
              </div>
              <p style={{ margin: 0, color: C.textMuted }}>{selectedTerm.description || 'No description provided.'}</p>
            </div>

            <div style={{ display: 'flex', padding: '0 24px', borderBottom: `1px solid ${C.border}`, gap: 12, background: C.panel }}>
              <div onClick={() => setActiveTab('properties')} style={{ padding: '12px 0', borderBottom: `2px solid ${activeTab === 'properties' ? C.accent : 'transparent'}`, color: activeTab === 'properties' ? C.accent : C.textMuted, cursor: 'pointer', fontWeight: 500, fontSize: 14 }}>Properties</div>
              {selectedTerm._kind === 'semantic' && (
                <div onClick={() => setActiveTab('technical')} style={{ padding: '12px 0', borderBottom: `2px solid ${activeTab === 'technical' ? C.accent : 'transparent'}`, color: activeTab === 'technical' ? C.accent : C.textMuted, cursor: 'pointer', fontWeight: 500, fontSize: 14 }}>Technical Assets</div>
              )}
              <div onClick={() => setActiveTab('relationships')} style={{ padding: '12px 0', borderBottom: `2px solid ${activeTab === 'relationships' ? C.accent : 'transparent'}`, color: activeTab === 'relationships' ? C.accent : C.textMuted, cursor: 'pointer', fontWeight: 500, fontSize: 14 }}>Relationships</div>
              <div onClick={() => setActiveTab('lineage')} style={{ padding: '12px 0', borderBottom: `2px solid ${activeTab === 'lineage' ? C.accent : 'transparent'}`, color: activeTab === 'lineage' ? C.accent : C.textMuted, cursor: 'pointer', fontWeight: 500, fontSize: 14 }}>Lineage</div>
            </div>

            <div style={{ flex: 1, overflowY: 'auto', position: 'relative' }}>
              {activeTab === 'properties' && (
                <div style={{ padding: 24 }}>
                  <table style={{ width: '100%', borderCollapse: 'collapse', fontSize: 14 }}>
                    <tbody>
                      <tr style={{ borderBottom: `1px solid ${C.border}` }}>
                        <td style={{ padding: '12px 0', color: C.textMuted, width: 200 }}>Name</td>
                        <td style={{ padding: '12px 0' }}>{selectedTerm.node_name}</td>
                      </tr>
                      <tr style={{ borderBottom: `1px solid ${C.border}` }}>
                        <td style={{ padding: '12px 0', color: C.textMuted }}>Qualified Path</td>
                        <td style={{ padding: '12px 0', fontFamily: 'monospace' }}>{selectedTerm.qualified_path || 'N/A'}</td>
                      </tr>
                      <tr style={{ borderBottom: `1px solid ${C.border}` }}>
                        <td style={{ padding: '12px 0', color: C.textMuted }}>Type</td>
                        <td style={{ padding: '12px 0' }}><Badge label={selectedTerm.catalog_type_name || (selectedTerm._kind === 'semantic' ? 'Semantic Term' : 'Business Term')} color={C.teal} /></td>
                      </tr>
                      <tr style={{ borderBottom: `1px solid ${C.border}` }}>
                        <td style={{ padding: '12px 0', color: C.textMuted }}>Node ID</td>
                        <td style={{ padding: '12px 0', fontFamily: 'monospace', color: C.textMuted }}>{selectedTerm.id}</td>
                      </tr>
                      <tr style={{ borderBottom: `1px solid ${C.border}` }}>
                        <td style={{ padding: '12px 0', color: C.textMuted }}>Created At</td>
                        <td style={{ padding: '12px 0' }}>{new Date(selectedTerm.created_at).toLocaleString()}</td>
                      </tr>
                    </tbody>
                  </table>
                </div>
              )}

              {activeTab === 'technical' && (
                <div style={{ padding: 24, display: 'flex', flexDirection: 'column', gap: 16 }}>
                  {/* Toolbar */}
                  <div style={{ display: 'flex', alignItems: 'center', gap: 10, flexWrap: 'wrap' }}>
                    <div style={{ position: 'relative', flex: 1, minWidth: 200 }}>
                      <span style={{ position: 'absolute', left: 10, top: '50%', transform: 'translateY(-50%)', color: C.textMuted, pointerEvents: 'none' }}>🔍</span>
                      <input
                        placeholder="Search columns or tables…"
                        value={taSearch}
                        onChange={e => setTaSearch(e.target.value)}
                        style={{
                          width: '100%', padding: '8px 12px 8px 32px',
                          background: '#1E2130', border: `1px solid ${C.border}`,
                          borderRadius: 8, color: C.text, fontSize: 13, outline: 'none', boxSizing: 'border-box',
                        }}
                      />
                    </div>
                    {/* Datasource facets */}
                    <div style={{ display: 'flex', gap: 6, flexWrap: 'wrap' }}>
                      {datasourceFacets.map((f: any) => (
                        <button
                          key={f.value}
                          onClick={() => setTaDsFilter(f.value)}
                          style={{
                            padding: '6px 12px', fontSize: 12, borderRadius: 6, cursor: 'pointer',
                            border: `1px solid ${taDsFilter === f.value ? C.accent : C.border}`,
                            background: taDsFilter === f.value ? C.accentDim : 'transparent',
                            color: taDsFilter === f.value ? C.accent : C.textMuted,
                            fontWeight: taDsFilter === f.value ? 700 : 400,
                          }}
                        >{f.label}</button>
                      ))}
                    </div>
                    {selectedTerm?._kind === 'semantic' && (
                      <button
                        onClick={() => setIsAddMappingOpen(true)}
                        style={{
                          display: 'flex', alignItems: 'center', gap: 6, padding: '8px 16px',
                          background: C.accent, border: 'none', borderRadius: 8,
                          color: '#fff', fontSize: 13, fontWeight: 600, cursor: 'pointer',
                          boxShadow: C.accentGlow, whiteSpace: 'nowrap',
                        }}
                      >＋ Map Column</button>
                    )}
                  </div>

                  {/* Count */}
                  <div style={{ fontSize: 12, color: C.textMuted }}>
                    {taLoading ? 'Loading mappings…' : `${technicalAssets.length} column mapping${technicalAssets.length !== 1 ? 's' : ''}`}
                    {(technicalAssetsRaw as any)?.total > technicalAssets.length && ` (filtered from ${(technicalAssetsRaw as any).total})`}
                  </div>

                  {/* Grid */}
                  {taLoading ? <Spinner /> : technicalAssets.length === 0 ? (
                    <Empty icon="🔌" title="No Technical Assets" subtitle={
                      taSearch || taDsFilter !== 'ALL'
                        ? 'No mappings match your search or filter.'
                        : 'This semantic term has no column mappings yet. Click "Map Column" to add one.'
                    } />
                  ) : (
                    <div style={{ background: C.panel, borderRadius: 10, border: `1px solid ${C.border}`, overflow: 'hidden' }}>
                      <table style={{ width: '100%', borderCollapse: 'collapse', textAlign: 'left' }}>
                        <thead>
                          <tr style={{ borderBottom: `2px solid ${C.border}`, background: 'rgba(255,255,255,0.03)' }}>
                            <th style={{ padding: '12px 16px', fontSize: 11, color: C.textMuted, fontWeight: 600, textTransform: 'uppercase', letterSpacing: '0.06em' }}>Datasource</th>
                            <th style={{ padding: '12px 16px', fontSize: 11, color: C.textMuted, fontWeight: 600, textTransform: 'uppercase', letterSpacing: '0.06em' }}>Schema · Table</th>
                            <th style={{ padding: '12px 16px', fontSize: 11, color: C.textMuted, fontWeight: 600, textTransform: 'uppercase', letterSpacing: '0.06em' }}>Column</th>
                            <th style={{ padding: '12px 16px', fontSize: 11, color: C.textMuted, fontWeight: 600, textTransform: 'uppercase', letterSpacing: '0.06em' }}>Data Type</th>
                            <th style={{ padding: '12px 16px', fontSize: 11, color: C.textMuted, fontWeight: 600, textTransform: 'uppercase', letterSpacing: '0.06em' }}>Source</th>
                            <th style={{ padding: '12px 16px', fontSize: 11, color: C.textMuted, fontWeight: 600, textTransform: 'uppercase', letterSpacing: '0.06em' }}></th>
                          </tr>
                        </thead>
                        <tbody>
                          {technicalAssets.map((a: any, i: number) => {
                            const parsed = parsePath(a.qualified_path);
                            const dsLabel = a.datasource || a.datasource_id || 'Unknown';
                            const tableLabel = [parsed.schema !== 'N/A' ? parsed.schema : null, parsed.table !== 'N/A' ? parsed.table : null].filter(Boolean).join(' · ');
                            const dataType = (a.properties as any)?.data_type;
                            return (
                              <tr
                                key={a.edge_id ?? i}
                                style={{ borderBottom: `1px solid ${C.border}`, transition: 'background 0.15s' }}
                                onMouseEnter={e => (e.currentTarget.style.background = 'rgba(99,102,241,0.05)')}
                                onMouseLeave={e => (e.currentTarget.style.background = 'transparent')}
                              >
                                <td style={{ padding: '14px 16px' }}>
                                  <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
                                    <span style={{ fontSize: 18 }}>{a.datasource_type === 'snowflake' ? '❄️' : '🐘'}</span>
                                    <div>
                                      <div style={{ fontSize: 13, fontWeight: 600, color: C.text, fontFamily: 'monospace' }}>{dsLabel}</div>
                                      {a.datasource_host && <div style={{ fontSize: 11, color: C.textMuted }}>{a.datasource_host}</div>}
                                    </div>
                                  </div>
                                </td>
                                <td style={{ padding: '14px 16px', fontFamily: 'monospace', fontSize: 13, color: C.textMuted }}>{tableLabel || '—'}</td>
                                <td style={{ padding: '14px 16px' }}>
                                  <span style={{ fontFamily: 'monospace', fontSize: 13, fontWeight: 700, color: C.text }}>{a.node_name}</span>
                                </td>
                                <td style={{ padding: '14px 16px' }}>
                                  {dataType ? <Badge label={dataType} color={typeColor(dataType)} /> : <span style={{ color: C.textMuted, fontSize: 12 }}>—</span>}
                                </td>
                                <td style={{ padding: '14px 16px' }}>
                                  <Badge label={a.is_core ? 'CORE' : 'CUSTOM'} color={a.is_core ? C.teal : '#ED6C02'} />
                                </td>
                                <td style={{ padding: '14px 16px', textAlign: 'right' }}>
                                  <button
                                    onClick={() => removeMapping(a.edge_id)}
                                    style={{
                                      background: 'transparent', border: `1px solid ${C.border}`,
                                      borderRadius: 6, color: C.danger, cursor: 'pointer',
                                      padding: '4px 10px', fontSize: 12,
                                    }}
                                  >✕ Remove</button>
                                </td>
                              </tr>
                            );
                          })}
                        </tbody>
                      </table>
                    </div>
                  )}

                  {/* Reusable Generic Mappings Selector */}
                  <GenericCatalogNodePicker
                    isOpen={isAddMappingOpen}
                    onClose={() => setIsAddMappingOpen(false)}
                    title="Map Columns"
                    subtitle={`Select one or more physical columns to link to ${selectedTerm?.node_name}`}
                    nodeTypeId="a64c1011-16e8-4ddf-b447-363bf8e15c9a" // Column node type
                    subjectNodeId={selectedId || ''}
                    excludeEdgeTypeId="97d82101-2b84-47a6-9ec0-f930fe389c3c" // has_context
                    tenantId={tenantId || ''}
                    confirmText="Save Mappings"
                    onConfirm={addMappings}
                  />
                </div>
              )}

              {activeTab === 'relationships' && (
                <div style={{ height: '100%', overflow: 'auto' }}>
                  {graphLoading ? <Spinner /> : (!nodeGraph?.edges || nodeGraph.edges.length === 0) ? (
                    <Empty icon="🔗" title="No Relationships" />
                  ) : (
                    <RelationshipList
                      edges={nodeGraph.edges}
                      nodes={[nodeGraph.node, ...(nodeGraph.connected_nodes ?? nodeGraph.nodes ?? [])]}
                      selectedNodeId={selectedId}
                      darkMode={true}
                      onDeleted={() => { refetchGraph(); }}
                    />
                  )}
                </div>
              )}

              {activeTab === 'lineage' && (
                <div style={{ width: '100%', height: '100%' }}>
                  <ReactFlow
                    nodes={nodes}
                    edges={edges}
                    onNodesChange={onNodesChange}
                    onEdgesChange={onEdgesChange}
                    nodeTypes={nodeTypes}
                    fitView
                    attributionPosition="bottom-right"
                  >
                    <Background variant={BackgroundVariant.Dots} gap={24} size={2} color={C.border} />
                    <Controls />
                    <MiniMap
                      nodeStrokeColor={C.border}
                      nodeColor={C.panel}
                      maskColor="rgba(0,0,0,0.5)"
                      style={{ background: C.bg }}
                    />
                  </ReactFlow>
                </div>
              )}

            </div>
          </div>
        )}
      </div>

      <EditSemanticTermDialog
        open={isEditSemModalOpen}
        onClose={() => setIsEditSemModalOpen(false)}
        term={{
          id: selectedTerm?.id,
          name: selectedTerm?.node_name,
          description: selectedTerm?.description,
          properties: selectedTerm?.properties,
          tenant_id: tenantId,
          tenant_datasource_id: selectedTerm?.tenant_datasource_id,
          catalog_type_name: selectedTerm?.catalog_type_name,
        }}
        onSave={() => {
          refetchSem();
          refetchBus();
        }}
      />
    </div>
  );
}
