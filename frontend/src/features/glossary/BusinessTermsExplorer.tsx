import * as React from 'react';
import { useState, useMemo, useEffect } from 'react';
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
} from 'reactflow';
import 'reactflow/dist/style.css';
import { useAccess } from '../../contexts/AccessContext';
import { readCachedSelection } from '../../utils/tenantScope';
import { useQuery } from '@tanstack/react-query';
import apiClient from '../../utils/apiClient';
import { IconButton, Tooltip, useTheme } from '@mui/material';
import { 
  Edit as EditIcon, 
  Delete as DeleteIcon, 
  Add as AddIcon, 
  Schema as SchemaIcon,
  Storage as StorageIcon,
  Timeline as TimelineIcon,
  Search as SearchIcon,
  AccountTree as AccountTreeIcon
} from '@mui/icons-material';
import { useDeleteTerm, useUpdateTerm, useCreateTerm } from '../../api/glossary';
import { useNodeTypes } from '../../api/nodeTypes';
import { usePropertyLookupMaps } from '../../hooks/usePropertyLookupMaps';
import { CoreIcon, CustomIcon } from '../../components/common/CoreCustomIcons';
import { LineageGraph } from './components/LineageGraph';
import { RelationshipExplorer } from './components/RelationshipExplorer';

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

const Pill: React.FC<{ children: React.ReactNode; active?: boolean; onClick: () => void; accentColor?: string; border?: string; textMuted?: string }> = ({ 
  children, active, onClick, accentColor = '#6366F1', border = 'rgba(255,255,255,0.07)', textMuted = '#8892A4' 
}) => (
  <button onClick={onClick} style={{
    display: 'flex', alignItems: 'center', gap: 6, padding: '6px 14px',
    background: active ? `${accentColor}18` : 'transparent',
    border: `1px solid ${active ? accentColor : border}`,
    borderRadius: 8, cursor: 'pointer', color: active ? accentColor : textMuted,
    fontSize: 13, fontWeight: 600, transition: 'all 0.2s ease',
    boxShadow: active ? `0 0 12px ${accentColor}44` : 'none',
    whiteSpace: 'nowrap',
  }}>
    {children}
  </button>
);

const Spinner: React.FC<{ border?: string; accent?: string }> = ({ border = 'rgba(255,255,255,0.1)', accent = '#6366F1' }) => (
  <div style={{ display: 'flex', justifyContent: 'center', alignItems: 'center', padding: 48 }}>
    <div style={{
      width: 36, height: 36, border: `3px solid ${border}`,
      borderTop: `3px solid ${accent}`, borderRadius: '50%',
      animation: 'spin 0.8s linear infinite',
    }} />
    <style>{`@keyframes spin { to { transform: rotate(360deg); } }`}</style>
  </div>
);

const Empty: React.FC<{ icon: string; title: string; subtitle?: string; text?: string; textMuted?: string }> = ({ icon, title, subtitle, text = '#E2E8F0', textMuted = '#8892A4' }) => (
  <div style={{ display: 'flex', flexDirection: 'column', alignItems: 'center', justifyContent: 'center', padding: 64, gap: 12, textAlign: 'center' }}>
    <div style={{ fontSize: 48, opacity: 0.3 }}>{icon}</div>
    <div style={{ color: text, fontWeight: 600, fontSize: 15 }}>{title}</div>
    {subtitle && <div style={{ color: textMuted, fontSize: 13, maxWidth: 320 }}>{subtitle}</div>}
  </div>
);

// ─────────────────────────────────────────────
// Lineage Custom Node Types
// ─────────────────────────────────────────────

// 1. Business Term Node (Center Focus)
const MainBusinessTermNode: React.FC<{ data: any }> = ({ data }) => (
  <div style={{
    background: data.panelBg || '#13161E',
    border: `2px solid #2DD4BF`,
    borderRadius: 12,
    minWidth: 260,
    maxWidth: 320,
    overflow: 'hidden',
    boxShadow: '0 0 20px rgba(45,212,191,0.3)',
    transition: 'all 0.2s ease',
  }}>
    <div style={{
      background: 'rgba(45,212,191,0.12)',
      padding: '10px 14px',
      borderBottom: `1px solid ${data.border || 'rgba(255,255,255,0.07)'}`,
      display: 'flex',
      alignItems: 'center',
      justifyContent: 'space-between',
    }}>
      <div style={{ display: 'flex', alignItems: 'center', gap: 6 }}>
        <span style={{ fontSize: 16 }}>💼</span>
        <span style={{ color: '#2DD4BF', fontWeight: 800, fontSize: 11, letterSpacing: '0.05em', textTransform: 'uppercase' }}>
          Business Term
        </span>
      </div>
      {data.isCore ? <CoreIcon fontSize="small" /> : <CustomIcon fontSize="small" />}
    </div>
    <div style={{ padding: '12px 14px' }}>
      <div style={{ color: data.textColor || '#fff', fontWeight: 800, fontSize: 15, wordBreak: 'break-word' }}>
        {data.label}
      </div>
      {data.description && (
        <div style={{ color: data.textMuted || '#8892A4', fontSize: 12, marginTop: 6, lineHeight: 1.4 }}>
          {data.description}
        </div>
      )}
    </div>
    <Handle
      type="source"
      position={Position.Right}
      style={{ background: '#2DD4BF', width: 9, height: 9, border: `2px solid ${data.panelBg || '#13161E'}` }}
    />
  </div>
);

// 2. Semantic Term Node (Downstream)
const DownstreamSemanticTermNode: React.FC<{ data: any }> = ({ data }) => (
  <div style={{
    background: data.panelBg || '#13161E',
    border: `1px solid rgba(99,102,241,0.4)`,
    borderRadius: 10,
    minWidth: 220,
    maxWidth: 280,
    overflow: 'hidden',
    boxShadow: '0 4px 20px rgba(0,0,0,0.4)',
    transition: 'all 0.2s ease',
  }}>
    <Handle
      type="target"
      position={Position.Left}
      style={{ background: '#6366F1', width: 8, height: 8, border: `2px solid ${data.panelBg || '#13161E'}` }}
    />
    <div style={{
      background: 'rgba(99,102,241,0.12)',
      padding: '8px 12px',
      borderBottom: `1px solid ${data.border || 'rgba(255,255,255,0.07)'}`,
      display: 'flex',
      alignItems: 'center',
      justifyContent: 'space-between',
    }}>
      <div style={{ display: 'flex', alignItems: 'center', gap: 6 }}>
        <span style={{ fontSize: 14 }}>🧠</span>
        <span style={{ color: '#818CF8', fontWeight: 700, fontSize: 11, letterSpacing: '0.04em', textTransform: 'uppercase' }}>
          Semantic Term
        </span>
      </div>
      <Badge label={data.mapped ? 'Mapped' : 'Unmapped'} color={data.mapped ? '#10B981' : '#8892A4'} />
    </div>
    <div style={{ padding: '10px 12px' }}>
      <div style={{ color: data.textColor || '#fff', fontWeight: 700, fontSize: 13, wordBreak: 'break-word' }}>
        {data.label}
      </div>
      {data.description && (
        <div style={{ color: data.textMuted || '#8892A4', fontSize: 11, marginTop: 4, lineHeight: 1.4, maxHeight: 44, overflow: 'hidden', textOverflow: 'ellipsis' }}>
          {data.description}
        </div>
      )}
    </div>
    <Handle
      type="source"
      position={Position.Right}
      style={{ background: '#6366F1', width: 8, height: 8, border: `2px solid ${data.panelBg || '#13161E'}` }}
    />
  </div>
);

// 3. Datasource Node (Columns / Tables downstream of Semantic Terms)
const DatasourceNode: React.FC<{ data: any }> = ({ data }) => (
  <div style={{
    background: data.panelBg || '#13161E',
    border: `1px solid rgba(96,165,250,0.4)`,
    borderRadius: 10,
    minWidth: 200,
    maxWidth: 260,
    overflow: 'hidden',
    boxShadow: '0 4px 16px rgba(0,0,0,0.3)',
    transition: 'all 0.2s ease',
  }}>
    <Handle
      type="target"
      position={Position.Left}
      style={{ background: '#60A5FA', width: 8, height: 8, border: `2px solid ${data.panelBg || '#13161E'}` }}
    />
    <div style={{
      background: 'rgba(96,165,250,0.12)',
      padding: '8px 12px',
      borderBottom: `1px solid ${data.border || 'rgba(255,255,255,0.07)'}`,
      display: 'flex',
      alignItems: 'center',
      justifyContent: 'space-between',
    }}>
      <div style={{ display: 'flex', alignItems: 'center', gap: 6 }}>
        <span style={{ fontSize: 14 }}>🗄️</span>
        <span style={{ color: '#60A5FA', fontWeight: 700, fontSize: 11, letterSpacing: '0.04em', textTransform: 'uppercase' }}>
          Physical Column
        </span>
      </div>
    </div>
    <div style={{ padding: '10px 12px' }}>
      <div style={{ color: data.textColor || '#fff', fontWeight: 700, fontSize: 13, wordBreak: 'break-word', fontFamily: 'monospace' }}>
        {data.label}
      </div>
      {data.path && (
        <div style={{ color: data.textMuted || '#8892A4', fontSize: 10, marginTop: 4, fontFamily: 'monospace', wordBreak: 'break-all' }}>
          {data.path}
        </div>
      )}
    </div>
  </div>
);

const nodeTypes = {
  mainBusinessTerm: MainBusinessTermNode,
  downstreamSemanticTerm: DownstreamSemanticTermNode,
  datasource: DatasourceNode,
};

export default function BusinessTermsExplorer() {
  const theme = useTheme();
  const isDark = theme.palette.mode === 'dark';

  const C = useMemo(() => ({
    bg: isDark ? '#0A0C10' : '#F8FAFC',
    sidebar: isDark ? '#0F1117' : '#F1F5F9',
    panel: isDark ? '#13161E' : '#FFFFFF',
    panelHover: isDark ? '#1A1E2A' : '#F1F5F9',
    border: isDark ? 'rgba(255,255,255,0.07)' : 'rgba(0,0,0,0.08)',
    borderStrong: isDark ? 'rgba(255,255,255,0.12)' : 'rgba(0,0,0,0.14)',
    accent: '#2DD4BF', // Teal accent for Business Terms
    accentDim: isDark ? 'rgba(45,212,191,0.15)' : 'rgba(45,212,191,0.08)',
    accentGlow: '0 0 20px rgba(45,212,191,0.4)',
    text: isDark ? '#E2E8F0' : '#0F172A',
    textMuted: isDark ? '#8892A4' : '#64748B',
    success: '#10B981',
    warning: '#F59E0B',
    danger: '#EF4444',
    purple: '#A78BFA',
    teal: '#2DD4BF',
    blue: '#60A5FA',
    orange: '#FB923C',
  }), [isDark]);

  const { currentTenant, isPlatformOperator, accessLevel } = useAccess();
  const cachedSelection = readCachedSelection();
  const tenantId = currentTenant?.id ?? cachedSelection.tenant?.id;

  const isWriter = isPlatformOperator || accessLevel === 'tenant_admin' || accessLevel === 'platform_operator';
  const canCreate = isWriter;
  const canUpdate = isWriter;
  const canDelete = isWriter;

  const [searchParams, setSearchParams] = useSearchParams();
  const selectedId = searchParams.get('id');

  const [searchTerm, setSearchTerm] = useState('');
  const [activeTab, setActiveTab] = useState<'details' | 'relationships' | 'lineage'>('details');

  const handleTabChange = (tab: 'details' | 'relationships' | 'lineage') => {
    setActiveTab(tab);
    setSearchParams((prev) => {
      const next = new URLSearchParams(prev);
      next.set('tab', tab);
      return next;
    });
  };

  // Modals
  const [isCreateBusModalOpen, setIsCreateBusModalOpen] = useState(false);
  const [isEditModalOpen, setIsEditModalOpen] = useState(false);
  const [busName, setBusName] = useState('');
  const [busDesc, setBusDesc] = useState('');
  const [busLinkSem, setBusLinkSem] = useState('');
  const [editName, setEditName] = useState('');
  const [editDesc, setEditDesc] = useState('');

  // 1. Fetch Business Terms
  const {
    data: busTermsRaw,
    isLoading: busLoading,
    refetch: refetchBus,
  } = useQuery<any[]>({
    queryKey: ['business-terms-explorer', tenantId],
    queryFn: async () => {
      if (!tenantId) return [];
      const res = await apiClient<any[]>(`/api/catalog/nodes?catalog_type=business_term&tenant_id=${tenantId}`);
      return Array.isArray(res) ? res : ((res as any)?.data ?? []);
    },
    enabled: !!tenantId,
  });

  // 2. Fetch Semantic Terms (for linking & relationships)
  const {
    data: semTermsRaw,
    isLoading: _semLoading,
  } = useQuery<any[]>({
    queryKey: ['semantic-terms-for-bus-explorer', tenantId],
    queryFn: async () => {
      if (!tenantId) return [];
      const res = await apiClient<any[]>(`/api/glossary/semantic-terms?tenant_id=${tenantId}`);
      return Array.isArray(res) ? res : ((res as any)?.data ?? []);
    },
    enabled: !!tenantId,
  });

  // 3. (Inline glossary-edges fetch removed in PR 4 — the new
  //    <RelationshipExplorer /> centralizes both the data fetch and the
  //    rendering. Edges are re-fetched on-demand by RelationshipExplorer
  //    via useEntityRelationships(), and the relationship list auto-refreshes
  //    via its onMutated callback.)

  // 4. (Technical assets fetch removed in PR 4 — the Cognitive Studio tab
  //    and the relationships tab now own their own data needs via the
  //    shared hooks in useEntityRelationships.)

  const busTerms = useMemo(() => {
    const list = Array.isArray(busTermsRaw) ? busTermsRaw : [];
    return list.filter((t: any) =>
      !searchTerm ||
      t.node_name?.toLowerCase().includes(searchTerm.toLowerCase()) ||
      t.description?.toLowerCase().includes(searchTerm.toLowerCase())
    );
  }, [busTermsRaw, searchTerm]);

  const semTerms = useMemo(() => {
    return Array.isArray(semTermsRaw) ? semTermsRaw : [];
  }, [semTermsRaw]);

  // Selected term object
  const selectedTerm = useMemo(() => {
    if (!selectedId) return null;
    return busTerms.find((t: any) => t.id === selectedId) || null;
  }, [selectedId, busTerms]);

  // Auto-select first term if none selected
  useEffect(() => {
    if (!selectedId && busTerms.length > 0) {
      setSearchParams({ id: busTerms[0].id });
    }
  }, [selectedId, busTerms, setSearchParams]);

  // Node type & Lookup Map resolution
  const { data: nodeTypesList } = useNodeTypes({ tenantId });
  const busNodeType = useMemo(() => {
    if (!nodeTypesList) return null;
    return (nodeTypesList as any[]).find((nt: any) => {
      const name = String(nt.catalog_type_name || '').toLowerCase();
      return name === 'business_term' || name.includes('business_term');
    }) || null;
  }, [nodeTypesList]);

  const _lookupMaps = usePropertyLookupMaps(busNodeType, selectedTerm?.properties);

  // Delete Term
  const deleteTermMutation = useDeleteTerm();
  const handleDeleteTerm = async (term: any) => {
    if (!window.confirm(`Are you sure you want to delete the business term "${term.node_name}"?`)) {
      return;
    }
    try {
      await deleteTermMutation.mutateAsync(term.id);
      setSearchParams({});
      await refetchBus();
    } catch (e: any) {
      console.error(e);
      alert('Failed to delete term: ' + (e?.message || 'Unknown error'));
    }
  };

  // Create Business Term
  const createBusiness = async () => {
    if (!tenantId || !busName.trim()) return;
    try {
      const res = await apiClient<any>(`/api/catalog/nodes?tenant_id=${tenantId}`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          node_name: busName.trim(),
          description: busDesc.trim(),
          catalog_type: 'business_term',
          tenant_id: tenantId,
        })
      });

      if (busLinkSem && res?.id) {
        await apiClient(`/api/glossary/edges?tenant_id=${tenantId}`, {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({
            subject_node_id: res.id,
            object_node_id: busLinkSem,
            edge_type_name: 'business_term_to_semantic_term',
            tenant_id: tenantId
          })
        });
      }

      setIsCreateBusModalOpen(false);
      setBusName('');
      setBusDesc('');
      setBusLinkSem('');
      await refetchBus();
      if (res?.id) setSearchParams({ id: res.id });
    } catch (e) {
      console.error(e);
      alert('Error creating business term');
    }
  };

  // Edit Business Term
  const handleEditOpen = (term: any) => {
    setEditName(term.node_name || '');
    setEditDesc(term.description || '');
    setIsEditModalOpen(true);
  };

  const handleEditSave = async () => {
    if (!selectedTerm || !tenantId) return;
    try {
      await apiClient(`/api/catalog/nodes/${selectedTerm.id}?tenant_id=${tenantId}`, {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          ...selectedTerm,
          node_name: editName.trim(),
          description: editDesc.trim(),
        })
      });
      setIsEditModalOpen(false);
      await refetchBus();
    } catch (e) {
      console.error(e);
      alert('Error updating business term');
    }
  };

  // Connected Semantic Terms for Selected Business Term
  // PR 4: connectedSemanticTerms + connectedCount useMemos removed —
  // the new <RelationshipExplorer /> shell owns its own edge filtering
  // via useEntityRelationships() and renders the unified relationship
  // grid (semantic terms + technical assets) inside the Relationships tab.

  // Statistics (topbar summary — total + active)
  const totalCount = busTerms.length;
  const activeCount = busTerms.filter((t: any) => t.is_active !== false).length;

  // PR 4: the ReactFlow node/edge state + the giant useEffect that
  // built the upstream/downstream graph are removed. The shared
  // <RelationshipExplorer /> shell handles its own graph state via
  // useEntityRelationships, and the LineageGraph in the lineage tab
  // manages the ReactFlow lifecycle.

  const inputStyle: React.CSSProperties = {
    width: '100%',
    background: isDark ? '#0A0C10' : '#F8FAFC',
    border: `1px solid ${C.border}`,
    borderRadius: 6,
    padding: '8px 12px',
    color: C.text,
    outline: 'none',
    boxSizing: 'border-box',
    marginBottom: 12,
    fontSize: 13,
  };

  return (
    <div style={{ display: 'flex', minHeight: 'calc(100vh - 64px)', background: C.bg, color: C.text, overflow: 'hidden' }}>
      
      {/* Create Modal */}
      {isCreateBusModalOpen && (
        <div style={{ position: 'fixed', inset: 0, background: 'rgba(0,0,0,0.7)', zIndex: 100, display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
          <div style={{ background: C.panel, padding: 24, borderRadius: 12, width: 440, border: `1px solid ${C.border}` }}>
            <h2 style={{ margin: '0 0 16px 0', fontSize: 18, fontWeight: 700, color: C.text }}>New Business Term</h2>
            <input 
              placeholder="Term Name (e.g. Total Revenue)" 
              style={inputStyle} 
              value={busName} 
              onChange={e => setBusName(e.target.value)} 
            />
            <textarea 
              placeholder="Business Definition & Context..." 
              style={{ ...inputStyle, height: 100, resize: 'vertical' }} 
              value={busDesc} 
              onChange={e => setBusDesc(e.target.value)} 
            />
            <select 
              style={inputStyle} 
              value={busLinkSem} 
              onChange={e => setBusLinkSem(e.target.value)}
            >
              <option value="">Link to Semantic Term (Optional)</option>
              {semTerms.map((t: any) => (
                <option key={t.id} value={t.id}>{t.node_name}</option>
              ))}
            </select>
            <div style={{ display: 'flex', gap: 12, justifyContent: 'flex-end', marginTop: 16 }}>
              <button 
                onClick={() => setIsCreateBusModalOpen(false)} 
                style={{ background: 'transparent', color: C.textMuted, border: 'none', cursor: 'pointer', padding: '6px 12px' }}
              >
                Cancel
              </button>
              <button 
                onClick={createBusiness} 
                style={{ background: C.accent, color: '#0F172A', fontWeight: 700, border: 'none', padding: '8px 18px', borderRadius: 6, cursor: 'pointer' }}
              >
                Create Term
              </button>
            </div>
          </div>
        </div>
      )}

      {/* Edit Modal */}
      {isEditModalOpen && (
        <div style={{ position: 'fixed', inset: 0, background: 'rgba(0,0,0,0.7)', zIndex: 100, display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
          <div style={{ background: C.panel, padding: 24, borderRadius: 12, width: 440, border: `1px solid ${C.border}` }}>
            <h2 style={{ margin: '0 0 16px 0', fontSize: 18, fontWeight: 700, color: C.text }}>Edit Business Term</h2>
            <input 
              placeholder="Term Name" 
              style={inputStyle} 
              value={editName} 
              onChange={e => setEditName(e.target.value)} 
            />
            <textarea 
              placeholder="Description" 
              style={{ ...inputStyle, height: 100, resize: 'vertical' }} 
              value={editDesc} 
              onChange={e => setEditDesc(e.target.value)} 
            />
            <div style={{ display: 'flex', gap: 12, justifyContent: 'flex-end', marginTop: 16 }}>
              <button 
                onClick={() => setIsEditModalOpen(false)} 
                style={{ background: 'transparent', color: C.textMuted, border: 'none', cursor: 'pointer', padding: '6px 12px' }}
              >
                Cancel
              </button>
              <button 
                onClick={handleEditSave} 
                style={{ background: C.accent, color: '#0F172A', fontWeight: 700, border: 'none', padding: '8px 18px', borderRadius: 6, cursor: 'pointer' }}
              >
                Save Changes
              </button>
            </div>
          </div>
        </div>
      )}

      {/* Sidebar */}
      <div style={{ width: 300, minWidth: 300, background: C.sidebar, borderRight: `1px solid ${C.border}`, display: 'flex', flexDirection: 'column' }}>
        <div style={{ padding: 14, borderBottom: `1px solid ${C.border}` }}>
          <div style={{ position: 'relative' }}>
            <SearchIcon sx={{ position: 'absolute', left: 10, top: 10, fontSize: 18, color: C.textMuted }} />
            <input 
              placeholder="Search business terms..." 
              value={searchTerm} 
              onChange={e => setSearchTerm(e.target.value)}
              style={{ 
                width: '100%', 
                background: isDark ? '#141722' : '#FFFFFF', 
                border: `1px solid ${C.border}`, 
                borderRadius: 6, 
                padding: '8px 12px 8px 34px', 
                color: C.text, 
                outline: 'none',
                boxSizing: 'border-box',
                fontSize: 13
              }} 
            />
          </div>
        </div>
        
        <div style={{ flex: 1, overflowY: 'auto' }}>
          <div style={{ padding: '12px 14px' }}>
            <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 8 }}>
              <span style={{ fontWeight: 700, fontSize: 11, letterSpacing: '0.04em', textTransform: 'uppercase', color: C.textMuted }}>
                Business Terms
              </span>
              <Badge label={String(busTerms.length)} color={C.teal} />
            </div>

            {busLoading && <Spinner accent={C.accent} />}
            {!busLoading && busTerms.length === 0 && (
              <div style={{ fontSize: 12, color: C.textMuted, padding: '16px 0', textAlign: 'center' }}>
                No business terms found.
              </div>
            )}
            
            {busTerms.map((t: any) => {
              const isSelected = selectedId === t.id;
              return (
                <div 
                  key={t.id} 
                  onClick={() => setSearchParams({ id: t.id })} 
                  style={{
                    padding: '8px 10px', 
                    borderRadius: 6, 
                    cursor: 'pointer', 
                    fontSize: 13,
                    fontWeight: isSelected ? 700 : 500,
                    marginBottom: 3,
                    background: isSelected ? C.accentDim : 'transparent',
                    color: isSelected ? C.teal : C.text,
                    borderLeft: `3px solid ${isSelected ? C.teal : 'transparent'}`,
                    display: 'flex',
                    alignItems: 'center',
                    justifyContent: 'space-between',
                    transition: 'all 0.15s ease'
                  }}
                >
                  <div style={{ display: 'flex', alignItems: 'center', gap: 6, overflow: 'hidden' }}>
                    <span style={{ fontSize: 13 }}>💼</span>
                    <span style={{ overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
                      {t.node_name}
                    </span>
                  </div>
                  {t.type && (
                    <span style={{ fontSize: 9, opacity: 0.6, textTransform: 'uppercase', fontFamily: 'monospace' }}>
                      {t.type}
                    </span>
                  )}
                </div>
              );
            })}
          </div>
        </div>
      </div>

      {/* Main Panel */}
      <div style={{ flex: 1, display: 'flex', flexDirection: 'column', background: C.bg, overflow: 'hidden' }}>
        
        {/* Topbar */}
        <div style={{ height: 60, borderBottom: `1px solid ${C.border}`, display: 'flex', alignItems: 'center', padding: '0 24px', justifyContent: 'space-between', background: C.panel }}>
          <div style={{ display: 'flex', alignItems: 'center', gap: 16 }}>
            <h1 style={{ margin: 0, fontSize: 17, fontWeight: 700, letterSpacing: '-0.01em', color: C.text }}>
              Business Terms Glossary
            </h1>
            
            {/* Outlined Summary Badges */}
            <div style={{ display: 'flex', gap: 8 }}>
              <span style={{
                display: 'inline-flex', alignItems: 'center', padding: '3px 9px',
                borderRadius: 9999, fontSize: 11, fontWeight: 700, letterSpacing: '0.04em',
                color: C.teal, background: isDark ? 'rgba(45,212,191,0.12)' : 'rgba(45,212,191,0.08)',
                border: `1px solid ${C.teal}44`, fontFamily: 'monospace', textTransform: 'uppercase',
              }}>
                {totalCount} Terms
              </span>
              <span style={{
                display: 'inline-flex', alignItems: 'center', padding: '3px 9px',
                borderRadius: 9999, fontSize: 11, fontWeight: 700, letterSpacing: '0.04em',
                color: C.success, background: isDark ? 'rgba(16,185,129,0.12)' : 'rgba(16,185,129,0.08)',
                border: `1px solid ${C.success}44`, fontFamily: 'monospace', textTransform: 'uppercase',
              }}>
                {activeCount} Active
              </span>
            </div>
          </div>

          {canCreate && (
            <div style={{ display: 'flex', gap: 10 }}>
              <button 
                onClick={() => setIsCreateBusModalOpen(true)} 
                style={{ 
                  display: 'flex', alignItems: 'center', gap: 6, 
                  background: C.accent, color: '#0F172A', 
                  border: 'none', padding: '6px 14px', borderRadius: 6, 
                  cursor: 'pointer', fontSize: 13, fontWeight: 700,
                  boxShadow: isDark ? C.accentGlow : 'none'
                }}
              >
                <AddIcon sx={{ fontSize: 16 }} /> Create Business Term
              </button>
            </div>
          )}
        </div>

        {/* Detail View */}
        {!selectedTerm ? (
          <Empty icon="💼" title="Select a Business Term" subtitle="Choose a business term from the sidebar to inspect lineage, metadata, and connected semantic terms." text={C.text} textMuted={C.textMuted} />
        ) : (
          <div style={{ flex: 1, display: 'flex', flexDirection: 'column', overflow: 'hidden' }}>
            {/* Header section for selected term */}
            <div style={{ padding: '16px 24px', borderBottom: `1px solid ${C.border}`, background: C.panel }}>
              <div style={{ display: 'flex', alignItems: 'center', gap: 12, marginBottom: 8 }}>
                <span style={{ fontSize: 22 }}>💼</span>
                <h2 style={{ margin: 0, fontSize: 19, fontWeight: 700, color: C.text }}>{selectedTerm.node_name}</h2>
                {selectedTerm.type === 'core' ? <CoreIcon fontSize="small" /> : <CustomIcon fontSize="small" />}
                {selectedTerm.is_active !== false && (
                  <Badge label="Active" color={C.success} />
                )}
                <div style={{ marginLeft: 'auto', display: 'flex', gap: 6, alignItems: 'center' }}>
                  {canUpdate && (
                    <Tooltip title="Edit Business Term">
                      <IconButton 
                        size="small" 
                        onClick={() => handleEditOpen(selectedTerm)}
                        sx={{ color: C.textMuted, '&:hover': { color: C.teal, bgcolor: C.accentDim } }}
                      >
                        <EditIcon fontSize="small" />
                      </IconButton>
                    </Tooltip>
                  )}
                  {canDelete && (
                    <Tooltip title="Delete Business Term">
                      <IconButton 
                        size="small" 
                        onClick={() => handleDeleteTerm(selectedTerm)}
                        sx={{ color: C.textMuted, '&:hover': { color: C.danger, bgcolor: 'rgba(239,68,68,0.15)' } }}
                      >
                        <DeleteIcon fontSize="small" />
                      </IconButton>
                    </Tooltip>
                  )}
                </div>
              </div>
              {selectedTerm.description && (
                <div style={{ color: C.textMuted, fontSize: 13, lineHeight: 1.5, maxWidth: 900 }}>
                  {selectedTerm.description}
                </div>
              )}
            </div>

            {/* Navigation Tabs */}
            <div style={{ display: 'flex', gap: 8, padding: '10px 24px', borderBottom: `1px solid ${C.border}`, background: isDark ? '#0A0C10' : '#F1F5F9' }}>
              <Pill active={activeTab === 'details'} onClick={() => handleTabChange('details')} accentColor={C.teal} border={C.border} textMuted={C.textMuted}>
                <SchemaIcon sx={{ fontSize: 16 }} /> Details
              </Pill>
              <Pill active={activeTab === 'relationships'} onClick={() => handleTabChange('relationships')} accentColor={C.teal} border={C.border} textMuted={C.textMuted}>
                <AccountTreeIcon sx={{ fontSize: 16 }} /> Relationships
              </Pill>
              <Pill active={activeTab === 'lineage'} onClick={() => handleTabChange('lineage')} accentColor={C.teal} border={C.border} textMuted={C.textMuted}>
                <TimelineIcon sx={{ fontSize: 16 }} /> Lineage
              </Pill>
            </div>

            {/* Tab Body */}
            <div style={{ flex: 1, overflow: 'auto', position: 'relative' }}>
              {activeTab === 'lineage' && selectedTerm && (
                <LineageGraph
                  focalTerm={selectedTerm}
                  focalLabel="Business Term (Focal)"
                  upstreamNodes={[]}
                  downstreamNodes={[]}
                  edges={[]}
                  showDatasourceLayer={false}
                  height={480}
                  emptyMessage="No lineage data available."
                />
              )}

              {activeTab === 'details' && (
                <div style={{ padding: 24, maxWidth: 900 }}>
                  <div style={{ background: C.panel, border: `1px solid ${C.border}`, borderRadius: 10, padding: 20, marginBottom: 20 }}>
                    <h3 style={{ margin: '0 0 14px 0', fontSize: 15, fontWeight: 700, color: C.text }}>Definition</h3>
                    <p style={{ color: C.text, fontSize: 14, lineHeight: 1.6, margin: 0 }}>
                      {selectedTerm.description || 'No description provided.'}
                    </p>
                  </div>

                  <div style={{ background: C.panel, border: `1px solid ${C.border}`, borderRadius: 10, padding: 20 }}>
                    <h3 style={{ margin: '0 0 14px 0', fontSize: 15, fontWeight: 700, color: C.text }}>System Metadata</h3>
                    <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 14, fontSize: 13 }}>
                      <div>
                        <span style={{ color: C.textMuted }}>Term ID: </span>
                        <span style={{ fontFamily: 'monospace', color: C.text }}>{selectedTerm.id}</span>
                      </div>
                      <div>
                        <span style={{ color: C.textMuted }}>Type: </span>
                        <span style={{ color: C.text }}>{selectedTerm.type || 'business_term'}</span>
                      </div>
                      <div>
                        <span style={{ color: C.textMuted }}>Qualified Path: </span>
                        <span style={{ fontFamily: 'monospace', color: C.text }}>{selectedTerm.qualified_path || 'N/A'}</span>
                      </div>
                      <div>
                        <span style={{ color: C.textMuted }}>Created At: </span>
                        <span style={{ color: C.text }}>{selectedTerm.created_at ? new Date(selectedTerm.created_at).toLocaleDateString() : 'N/A'}</span>
                      </div>
                    </div>
                  </div>
                </div>
              )}


              {activeTab === 'relationships' && selectedTerm && (
                <div style={{ flex: 1, overflowY: 'auto', background: C.bg }}>
                  <RelationshipExplorer
                    entityType="business_term"
                    entityId={selectedTerm.id}
                    focalNode={selectedTerm}
                    onNavigate={(id) => setSearchParams({ id })}
                    onMutated={() => {
                      void refetchBus();
                    }}
                  />
                </div>
              )}
            </div>
          </div>
        )}
      </div>
    </div>
  );
}
