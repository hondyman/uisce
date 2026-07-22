import * as React from 'react';
import { useState, useMemo } from 'react';
import { useApiQuery } from '../../hooks/useApiQuery';

interface GenericCatalogNodePickerProps {
  isOpen: boolean;
  onClose: () => void;
  title: string;
  subtitle?: string;
  nodeTypeId: string;
  subjectNodeId: string;
  excludeEdgeTypeId?: string;
  excludeRelationshipType?: string;
  tenantId: string;
  confirmText?: string;
  onConfirm: (selectedNodeIds: string[]) => void | Promise<void>;
}

const C = {
  panel: '#13161E',
  border: 'rgba(255,255,255,0.07)',
  accent: '#6366F1',
  accentDim: 'rgba(99,102,241,0.15)',
  accentGlow: '0 0 20px rgba(99,102,241,0.4)',
  text: '#E2E8F0',
  textMuted: '#8892A4',
};

const Spinner: React.FC = () => (
  <div style={{ display: 'flex', justifyContent: 'center', padding: '24px 0' }}>
    <div style={{
      width: 24, height: 24,
      border: `2px solid ${C.accentDim}`,
      borderTop: `2px solid ${C.accent}`,
      borderRadius: '50%',
      animation: 'spin 0.8s linear infinite',
    }} />
    <style>{`
      @keyframes spin {
        0% { transform: rotate(0deg); }
        100% { transform: rotate(360deg); }
      }
    `}</style>
  </div>
);

export default function GenericCatalogNodePicker({
  isOpen,
  onClose,
  title,
  subtitle,
  nodeTypeId,
  subjectNodeId,
  excludeEdgeTypeId,
  excludeRelationshipType,
  tenantId,
  confirmText = 'Save Mappings',
  onConfirm,
}: GenericCatalogNodePickerProps) {
  const [search, setSearch] = useState('');
  const [selectedIds, setSelectedIds] = useState<string[]>([]);

  // Build the REST query URL with exclude query params for backend-driven duplicate prevention
  const queryUrl = useMemo(() => {
    let url = `/api/rest/catalog-nodes?node_type_id=${nodeTypeId}&tenant_id=${tenantId}&limit=1000`;
    if (subjectNodeId) {
      url += `&exclude_connected_to_subject_id=${subjectNodeId}`;
      if (excludeEdgeTypeId) {
        url += `&exclude_connection_edge_type_id=${excludeEdgeTypeId}`;
      }
      if (excludeRelationshipType) {
        url += `&exclude_connection_relationship_type=${excludeRelationshipType}`;
      }
    }
    return url;
  }, [nodeTypeId, subjectNodeId, excludeEdgeTypeId, excludeRelationshipType, tenantId]);

  const { data: nodesRaw, loading } = useApiQuery<any[]>(queryUrl, {
    skip: !isOpen || !tenantId,
    dependencies: [queryUrl, isOpen],
  });

  const nodes = useMemo(() => {
    const list = Array.isArray(nodesRaw) ? nodesRaw : (nodesRaw as any)?.data ?? [];
    return list.filter((n: any) =>
      !search ||
      n.node_name?.toLowerCase().includes(search.toLowerCase()) ||
      n.qualified_path?.toLowerCase().includes(search.toLowerCase())
    );
  }, [nodesRaw, search]);

  const toggleSelect = (id: string) => {
    setSelectedIds(prev =>
      prev.includes(id) ? prev.filter(x => x !== id) : [...prev, id]
    );
  };

  const toggleAll = () => {
    if (selectedIds.length === nodes.length) {
      setSelectedIds([]);
    } else {
      setSelectedIds(nodes.map((n: any) => n.id));
    }
  };

  const handleConfirm = async () => {
    if (selectedIds.length === 0) return;
    await onConfirm(selectedIds);
    setSelectedIds([]);
    setSearch('');
  };

  if (!isOpen) return null;

  return (
    <div style={{
      position: 'fixed',
      inset: 0,
      background: 'rgba(0,0,0,0.75)',
      zIndex: 1000,
      display: 'flex',
      alignItems: 'center',
      justifyContent: 'center',
    }}>
      <div style={{
        background: C.panel,
        borderRadius: 14,
        width: 540,
        maxHeight: '80vh',
        display: 'flex',
        flexDirection: 'column',
        border: `1px solid ${C.border}`,
        boxShadow: '0 24px 80px rgba(0,0,0,0.6)',
      }}>
        {/* Header */}
        <div style={{
          padding: '20px 24px',
          borderBottom: `1px solid ${C.border}`,
          display: 'flex',
          justifyContent: 'space-between',
          alignItems: 'center',
        }}>
          <div>
            <h3 style={{ margin: 0, fontSize: 16, fontWeight: 700, color: C.text }}>{title}</h3>
            {subtitle && <p style={{ margin: '4px 0 0', fontSize: 12, color: C.textMuted }}>{subtitle}</p>}
          </div>
          <button
            onClick={() => { onClose(); setSelectedIds([]); setSearch(''); }}
            style={{ background: 'transparent', border: 'none', color: C.textMuted, cursor: 'pointer', fontSize: 20 }}
          >
            ✕
          </button>
        </div>

        {/* Search & Actions */}
        <div style={{
          padding: '16px 24px',
          borderBottom: `1px solid ${C.border}`,
          display: 'flex',
          gap: 12,
          alignItems: 'center',
        }}>
          <input
            autoFocus
            placeholder="Search nodes by name or path…"
            value={search}
            onChange={e => setSearch(e.target.value)}
            style={{
              flex: 1,
              padding: '10px 14px',
              background: '#1E2130',
              border: `1px solid ${C.border}`,
              borderRadius: 8,
              color: C.text,
              fontSize: 13,
              outline: 'none',
              boxSizing: 'border-box',
            }}
          />
          {nodes.length > 0 && (
            <button
              onClick={toggleAll}
              style={{
                background: 'transparent',
                border: `1px solid ${C.border}`,
                color: C.textMuted,
                borderRadius: 8,
                padding: '8px 12px',
                fontSize: 12,
                cursor: 'pointer',
                whiteSpace: 'nowrap',
              }}
            >
              {selectedIds.length === nodes.length ? 'Clear All' : 'Select All'}
            </button>
          )}
        </div>

        {/* List Content */}
        <div style={{ overflowY: 'auto', flex: 1 }}>
          {loading ? (
            <Spinner />
          ) : nodes.length === 0 ? (
            <div style={{ padding: '40px 24px', color: C.textMuted, textAlign: 'center', fontSize: 13 }}>
              No available nodes to map
            </div>
          ) : (
            nodes.map((n: any) => {
              const isSelected = selectedIds.includes(n.id);
              return (
                <div
                  key={n.id}
                  onClick={() => toggleSelect(n.id)}
                  style={{
                    padding: '12px 24px',
                    cursor: 'pointer',
                    borderBottom: `1px solid ${C.border}`,
                    display: 'flex',
                    justifyContent: 'space-between',
                    alignItems: 'center',
                    background: isSelected ? 'rgba(99,102,241,0.06)' : 'transparent',
                    transition: 'background 0.15s',
                  }}
                  onMouseEnter={e => { if (!isSelected) e.currentTarget.style.background = 'rgba(255,255,255,0.02)'; }}
                  onMouseLeave={e => { if (!isSelected) e.currentTarget.style.background = 'transparent'; }}
                >
                  <div style={{ display: 'flex', alignItems: 'center', gap: 12 }}>
                    <input
                      type="checkbox"
                      checked={isSelected}
                      readOnly
                      style={{ cursor: 'pointer', accentColor: C.accent }}
                    />
                    <div>
                      <div style={{ fontSize: 13, fontWeight: 600, fontFamily: 'monospace', color: C.text }}>
                        {n.node_name}
                      </div>
                      {n.qualified_path && (
                        <div style={{ fontSize: 11, color: C.textMuted, marginTop: 2 }}>
                          {n.qualified_path}
                        </div>
                      )}
                    </div>
                  </div>
                </div>
              );
            })
          )}
        </div>

        {/* Footer actions */}
        <div style={{
          padding: '16px 24px',
          borderTop: `1px solid ${C.border}`,
          display: 'flex',
          justifyContent: 'flex-end',
          gap: 12,
        }}>
          <button
            onClick={() => { onClose(); setSelectedIds([]); setSearch(''); }}
            style={{
              background: 'transparent',
              color: C.textMuted,
              border: `1px solid ${C.border}`,
              borderRadius: 8,
              padding: '8px 16px',
              cursor: 'pointer',
              fontSize: 13,
            }}
          >
            Cancel
          </button>
          <button
            disabled={selectedIds.length === 0}
            onClick={handleConfirm}
            style={{
              background: selectedIds.length === 0 ? C.border : C.accent,
              color: '#fff',
              border: 'none',
              borderRadius: 8,
              padding: '8px 20px',
              cursor: selectedIds.length === 0 ? 'not-allowed' : 'pointer',
              fontSize: 13,
              fontWeight: 600,
              boxShadow: selectedIds.length === 0 ? 'none' : C.accentGlow,
            }}
          >
            {confirmText} ({selectedIds.length})
          </button>
        </div>
      </div>
    </div>
  );
}
