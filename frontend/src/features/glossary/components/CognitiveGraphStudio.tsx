import React, { useState, useEffect, useCallback, useMemo, useRef } from 'react';
import { useAccess } from '../../../contexts/AccessContext';
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
  NodeProps,
  Edge,
  Node,
} from 'reactflow';
import 'reactflow/dist/style.css';
import {
  CircularProgress,
  Tooltip,
} from '@mui/material';
import LayersIcon from '@mui/icons-material/Layers';
import StorageIcon from '@mui/icons-material/Storage';
import HubIcon from '@mui/icons-material/Hub';
import TimelineIcon from '@mui/icons-material/Timeline';
import AccountTreeIcon from '@mui/icons-material/AccountTree';
import ArrowBackIcon from '@mui/icons-material/ArrowBack';
import OpenInFullIcon from '@mui/icons-material/OpenInFull';
import apiClient from '../../../utils/apiClient';

export type LensType =
  | 'SEMANTIC_CALCULATION_MESH'
  | 'PHYSICAL_ERD'
  | 'SUBTYPE_AND_PEERS'
  | 'TAXONOMY_HIERARCHY'
  | 'PIPELINE_IMPACT';

interface LensOption {
  type: LensType;
  label: string;
  shortLabel: string;
  icon: React.ReactNode;
  color: string;
  bg: string;
  description: string;
}

const LENS_OPTIONS: LensOption[] = [
  {
    type: 'SUBTYPE_AND_PEERS',
    label: '🧬 Subtypes, Symbology & Peers',
    shortLabel: 'Subtypes & Peers',
    icon: <HubIcon sx={{ fontSize: 16 }} />,
    color: '#A855F7',
    bg: '#A855F718',
    description: 'Specializations (child -> parent), peer symbologies, and semantic differentiation rules.',
  },
  {
    type: 'TAXONOMY_HIERARCHY',
    label: '🏛️ 3-Tier Enterprise Taxonomy',
    shortLabel: '3-Tier Taxonomy',
    icon: <LayersIcon sx={{ fontSize: 16 }} />,
    color: '#10B981',
    bg: '#10B98118',
    description: 'Fixed 3-tier enterprise governance (Domain L1 -> Category L2 -> Classification L3).',
  },
  {
    type: 'SEMANTIC_CALCULATION_MESH',
    label: '🧮 Semantic & Calculation Mesh',
    shortLabel: 'Calculation Mesh',
    icon: <AccountTreeIcon sx={{ fontSize: 16 }} />,
    color: '#F59E0B',
    bg: '#F59E0B18',
    description: 'AST formulas, calculation DAGs, upstream metric inputs, and consuming Business Objects.',
  },
  {
    type: 'PHYSICAL_ERD',
    label: '🗄️ Physical ERD & Storage Binding',
    shortLabel: 'Physical ERD',
    icon: <StorageIcon sx={{ fontSize: 16 }} />,
    color: '#00D4FF',
    bg: '#00D4FF18',
    description: 'Physical tables, column types, primary/foreign keys, and join conditions across engines.',
  },
  {
    type: 'PIPELINE_IMPACT',
    label: '🌊 End-to-End Pipeline & Impact',
    shortLabel: 'Blast Radius & Impact',
    icon: <TimelineIcon sx={{ fontSize: 16 }} />,
    color: '#EF4444',
    bg: '#EF444418',
    description: '4-tier horizontal blast radius: Ingestion (CDC) -> Storage Seam -> Contracts -> APIs/Reports.',
  },
];

// ─────────────────────────────────────────────
// Custom Node Component: Glassmorphic Cognitive Node
// ─────────────────────────────────────────────
const CognitiveNode: React.FC<NodeProps> = ({ data, selected }) => {
  const isFocal = data.is_focal;
  const nodeType = data.node_type || 'business_term';
  const role = data.properties?.role || data.properties?.tier;
  const ast = data.properties?.ast_expression;
  const engine = data.properties?.engine;
  const columns: string[] = data.properties?.columns || [];
  const diffNote = data.properties?.differentiation || data.properties?.description;

  const typeColor = useMemo(() => {
    if (isFocal) return '#38BDF8';
    if (nodeType.includes('L1') || nodeType.includes('L2') || nodeType.includes('L3')) return '#10B981';
    if (nodeType === 'database_table') return '#00D4FF';
    if (nodeType === 'business_object') return '#F59E0B';
    if (nodeType === 'consumer_endpoint') return '#EF4444';
    if (nodeType === 'pipeline_source') return '#EC4899';
    return '#A855F7';
  }, [isFocal, nodeType]);

  return (
    <div
      style={{
        width: 290,
        background: isFocal ? 'rgba(7, 21, 38, 0.95)' : 'rgba(15, 23, 42, 0.88)',
        backdropFilter: 'blur(16px)',
        border: `1.5px solid ${isFocal ? '#38BDF8' : selected ? '#818CF8' : 'rgba(255, 255, 255, 0.1)'}`,
        borderRadius: 12,
        padding: '12px 14px',
        boxShadow: isFocal
          ? '0 0 24px rgba(56, 189, 248, 0.35), 0 8px 32px rgba(0,0,0,0.6)'
          : '0 4px 20px rgba(0,0,0,0.4)',
        color: '#F8FAFC',
        fontFamily: 'Inter, system-ui, sans-serif',
        position: 'relative',
      }}
    >
      <Handle type="target" position={Position.Top} style={{ background: typeColor, width: 8, height: 8 }} />
      <Handle type="target" position={Position.Left} style={{ background: typeColor, width: 8, height: 8 }} />

      {/* Header Bar */}
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 6 }}>
        <div style={{ display: 'flex', alignItems: 'center', gap: 6, maxWidth: 210 }}>
          <span
            style={{
              padding: '2px 6px',
              borderRadius: 4,
              fontSize: 9.5,
              fontWeight: 800,
              letterSpacing: '0.04em',
              textTransform: 'uppercase',
              color: typeColor,
              background: `${typeColor}22`,
              border: `1px solid ${typeColor}44`,
            }}
          >
            {engine ? `🐘 ${engine}` : nodeType.replace(/_/g, ' ')}
          </span>
          {isFocal && (
            <span
              style={{
                padding: '2px 6px',
                borderRadius: 4,
                fontSize: 9.5,
                fontWeight: 800,
                color: '#38BDF8',
                background: 'rgba(56, 189, 248, 0.2)',
                border: '1px solid rgba(56, 189, 248, 0.5)',
              }}
            >
              Focal
            </span>
          )}
        </div>

        {/* Shift Focus / Explode Button */}
        {!isFocal && data.onFocus && (
          <Tooltip title="Shift Focus & Re-anchor Lens on this Node">
            <button
              onClick={(e) => {
                e.stopPropagation();
                data.onFocus();
              }}
              style={{
                background: 'transparent',
                border: '1px solid rgba(255, 255, 255, 0.15)',
                borderRadius: 4,
                color: '#94A3B8',
                cursor: 'pointer',
                padding: '2px 5px',
                display: 'flex',
                alignItems: 'center',
                fontSize: 11,
              }}
            >
              <OpenInFullIcon sx={{ fontSize: 12 }} />
            </button>
          </Tooltip>
        )}
      </div>

      {/* Node Name */}
      <div style={{ fontWeight: 700, fontSize: 13.5, color: '#FFFFFF', marginBottom: 4, wordBreak: 'break-word' }}>
        {data.node_name}
      </div>

      {/* Role / Subtype / AST details */}
      {role && (
        <div style={{ fontSize: 11, color: typeColor, fontWeight: 600, marginBottom: 4 }}>
          {role}
        </div>
      )}

      {ast && (
        <div
          style={{
            padding: '6px 8px',
            background: 'rgba(0, 0, 0, 0.4)',
            borderRadius: 6,
            fontFamily: 'monospace',
            fontSize: 11,
            color: '#FCD34D',
            border: '1px solid rgba(245, 158, 11, 0.25)',
            marginBottom: 6,
          }}
        >
          {ast}
        </div>
      )}

      {columns.length > 0 && (
        <div style={{ display: 'flex', flexDirection: 'column', gap: 2, marginTop: 6, borderTop: '1px solid rgba(255,255,255,0.08)', paddingTop: 6 }}>
          {columns.slice(0, 4).map((c, i) => (
            <div key={i} style={{ fontSize: 10.5, fontFamily: 'monospace', color: '#94A3B8' }}>
              {c}
            </div>
          ))}
        </div>
      )}

      {diffNote && (
        <div style={{ fontSize: 11, color: '#94A3B8', lineHeight: 1.35, marginTop: 4 }}>
          {diffNote}
        </div>
      )}

      <Handle type="source" position={Position.Bottom} style={{ background: typeColor, width: 8, height: 8 }} />
      <Handle type="source" position={Position.Right} style={{ background: typeColor, width: 8, height: 8 }} />
    </div>
  );
};

const nodeTypes = {
  cognitiveNode: CognitiveNode,
};

// ─────────────────────────────────────────────
// Cognitive Studio Layout Engine
// ─────────────────────────────────────────────
function layoutLensNodes(nodes: any[], edges: any[], lensType: LensType): { layoutNodes: Node[]; layoutEdges: Edge[] } {
  const nodeWidth = 300;
  const xSpacing = 360;
  const ySpacing = 190;

  const focal = nodes.find((n) => n.is_focal) || nodes[0];
  const positionedNodes: Node[] = [];

  if (lensType === 'SUBTYPE_AND_PEERS') {
    // Top-down: Supertype Parent at top (0, 0), Subtype Children branching downward in a horizontal row
    const parent = nodes.find((n) => {
      const role = (n.properties?.role || '').toLowerCase();
      return role.includes('supertype') || role.includes('parent') || role.includes('concept') || role.includes('standard');
    }) || (nodes.length > 1 && focal?.is_focal ? focal : nodes[0]);

    const children = nodes.filter((n) => n.id !== parent?.id);

    if (parent) {
      positionedNodes.push({
        id: parent.id,
        type: 'cognitiveNode',
        position: { x: Math.max(50, (children.length * xSpacing) / 2 - nodeWidth / 2), y: 50 },
        data: parent,
      });
    }

    children.forEach((child, idx) => {
      positionedNodes.push({
        id: child.id,
        type: 'cognitiveNode',
        position: { x: 50 + idx * xSpacing, y: 50 + ySpacing },
        data: child,
      });
    });
  } else if (lensType === 'TAXONOMY_HIERARCHY') {
    // Top-down vertical tree: Domain (L1) -> Category (L2) -> Classification (L3) -> Leaf Term
    const sorted = [...nodes].sort((a, b) => {
      const tierScore = (n: any) => {
        if (n.node_type === 'Classification_L1') return 1;
        if (n.node_type === 'Classification_L2') return 2;
        if (n.node_type === 'Classification_L3') return 3;
        if (n.is_focal) return 4;
        return 5;
      };
      return tierScore(a) - tierScore(b);
    });

    sorted.forEach((n, idx) => {
      positionedNodes.push({
        id: n.id,
        type: 'cognitiveNode',
        position: { x: 180, y: 40 + idx * ySpacing },
        data: n,
      });
    });
  } else if (lensType === 'SEMANTIC_CALCULATION_MESH') {
    // Left-to-right DAG: Inputs (col 0) -> Mid terms (col 1) -> Focal (col 2) -> Consuming BO (col 3)
    const bo = nodes.filter((n) => n.node_type === 'business_object');
    const focalList = nodes.filter((n) => n.is_focal);
    const nonFocalNonBo = nodes.filter((n) => !n.is_focal && n.node_type !== 'business_object');

    // Categorize into upstream inputs vs intermediate
    const inputs: any[] = [];
    const mid: any[] = [];

    nonFocalNonBo.forEach((n) => {
      const isInput = (n.properties?.role || '').toLowerCase().includes('input') ||
        (n.properties?.role || '').toLowerCase().includes('source') ||
        (n.properties?.role || '').toLowerCase().includes('multiplier') ||
        n.node_key?.startsWith('base_') || n.node_key?.startsWith('raw_') ||
        edges.some((e) => e.source_id === n.id && (e.edge_type === 'USES_INPUT' || e.edge_type === 'TRANSFORMS_TO'));

      if (isInput && nonFocalNonBo.length > 1) {
        inputs.push(n);
      } else {
        mid.push(n);
      }
    });

    // If all fell into mid, move them to inputs
    if (inputs.length === 0 && mid.length > 0) {
      inputs.push(...mid.splice(0, mid.length));
    }

    inputs.forEach((inp, idx) => {
      positionedNodes.push({
        id: inp.id,
        type: 'cognitiveNode',
        position: { x: 50, y: 50 + idx * 130 },
        data: inp,
      });
    });

    mid.forEach((m, idx) => {
      positionedNodes.push({
        id: m.id,
        type: 'cognitiveNode',
        position: { x: 50 + xSpacing, y: 80 + idx * 140 },
        data: m,
      });
    });

    const focalX = 50 + (inputs.length > 0 ? xSpacing : 0) + (mid.length > 0 ? xSpacing : 0);
    focalList.forEach((f, idx) => {
      positionedNodes.push({
        id: f.id,
        type: 'cognitiveNode',
        position: { x: focalX, y: 100 + idx * 140 },
        data: f,
      });
    });

    const boX = focalX + xSpacing;
    bo.forEach((b, idx) => {
      positionedNodes.push({
        id: b.id,
        type: 'cognitiveNode',
        position: { x: boX, y: 100 + idx * 140 },
        data: b,
      });
    });
  } else if (lensType === 'PHYSICAL_ERD') {
    // Physical ERD: Focal Term (left) -> Bound Tables (right)
    if (focal) {
      positionedNodes.push({
        id: focal.id,
        type: 'cognitiveNode',
        position: { x: 50, y: 120 },
        data: focal,
      });
    }

    const tables = nodes.filter((n) => n.id !== focal?.id);
    tables.forEach((t, idx) => {
      positionedNodes.push({
        id: t.id,
        type: 'cognitiveNode',
        position: { x: 50 + (idx + 1) * xSpacing, y: 60 + (idx % 2) * 50 },
        data: t,
      });
    });
  } else {
    // Horizontal 4-tier pipeline
    const pipelineOrder = (n: any) => {
      if (n.node_type === 'pipeline_source') return 1;
      if (n.node_type === 'database_table') return 2;
      if (n.is_focal) return 3;
      if (n.node_type === 'business_object') return 4;
      if (n.node_type === 'consumer_endpoint') return 5;
      return 6;
    };

    const sorted = [...nodes].sort((a, b) => pipelineOrder(a) - pipelineOrder(b));

    sorted.forEach((n, idx) => {
      positionedNodes.push({
        id: n.id,
        type: 'cognitiveNode',
        position: { x: 50 + idx * xSpacing, y: 120 },
        data: n,
      });
    });
  }

  const formattedEdges: Edge[] = edges.map((e) => {
    const isSpecialization = e.edge_type === 'IS_SPECIALIZATION_OF';
    const isDifferentiated = e.edge_type === 'DIFFERENTIATED_FROM';
    const isTaxonomy = e.edge_type === 'CLASSIFIED_BY' || e.edge_type === 'PARENT_OF';

    let strokeColor = '#818CF8';
    if (isSpecialization) strokeColor = '#F59E0B';
    else if (isDifferentiated) strokeColor = '#EC4899';
    else if (isTaxonomy) strokeColor = '#10B981';

    return {
      id: e.id,
      source: e.source_id,
      target: e.target_id,
      label: e.edge_type,
      animated: isSpecialization || isTaxonomy,
      style: { stroke: strokeColor, strokeWidth: 2 },
      labelStyle: { fill: strokeColor, fontWeight: 700, fontSize: 10.5, fontFamily: 'monospace' },
      labelBgStyle: { fill: '#0A0C10', fillOpacity: 0.9 },
      markerEnd: {
        type: MarkerType.ArrowClosed,
        color: strokeColor,
      },
    };
  });

  return { layoutNodes: positionedNodes, layoutEdges: formattedEdges };
}

// ─────────────────────────────────────────────
// Main CognitiveGraphStudio Component
// ─────────────────────────────────────────────
const DEFAULT_LENS_BY_ENTITY_TYPE: Record<string, LensType> = {
  business_object: 'SEMANTIC_CALCULATION_MESH',
  semantic_term: 'SUBTYPE_AND_PEERS',
  business_term: 'TAXONOMY_HIERARCHY',
  calculation: 'SEMANTIC_CALCULATION_MESH',
  column: 'PHYSICAL_ERD',
  table: 'PHYSICAL_ERD',
  api_endpoint: 'PHYSICAL_ERD',
};

/**
 * Cognitive Studio: adaptive multi-lens graph visualizer for any catalog node.
 *
 * Renders five projection lenses (Calculation Mesh, Physical ERD, Subtypes/Peers,
 * Taxonomy, Pipeline Impact) over a focal catalog entity. Adaptive default lens
 * routes by entityType so the most useful lens opens immediately.
 *
 * Permission gating: destructive lens actions (delete-edge, re-bind) require
 * data_steward, data_engineer, or admin; read-only viewing is allowed for
 * everyone.
 */
export const CognitiveGraphStudio: React.FC<{
  entityId: string;
  entityType: 'business_term' | 'semantic_term' | 'business_object' | string;
  tenantId: string;
  onNavigate?: (id: string, entityType: string) => void;
  height?: number | string;
}> = ({ entityId, entityType, tenantId, onNavigate, height = 540 }) => {
  const { accessLevel, isPlatformOperator } = useAccess();
  const roleAllowsMutate =
    accessLevel === 'tenant_admin' ||
    accessLevel === 'platform_operator' ||
    isPlatformOperator;

  const initialLens: LensType =
    DEFAULT_LENS_BY_ENTITY_TYPE[entityType] ?? 'SUBTYPE_AND_PEERS';

  const [focalNodeId, setFocalNodeId] = useState<string>(entityId);
  const [activeLens, setActiveLens] = useState<LensType>(initialLens);
  const [loading, setLoading] = useState<boolean>(true);
  const [breadcrumb, setBreadcrumb] = useState<string>('');
  const [historyStack, setHistoryStack] = useState<Array<{ id: string; name: string }>>([
    { id: entityId, name: entityId },
  ]);

  const [nodes, setNodes, onNodesChange] = useNodesState([]);
  const [edges, setEdges, onEdgesChange] = useEdgesState([]);

  const historyStackRef = useRef(historyStack);
  historyStackRef.current = historyStack;

  // Auto-sync with prop changes — reset the focal node + history when
  // the parent passes a new entityId. The active lens is preserved so the
  // user keeps their current view across re-mounts of the same entity.
  useEffect(() => {
    if (entityId) {
      setFocalNodeId(entityId);
      setHistoryStack([{ id: entityId, name: entityId }]);
    }
  }, [entityId]);

  const handleShiftFocus = useCallback((nodeId: string, nodeName: string) => {
    setHistoryStack((prev) => [...prev, { id: nodeId, name: nodeName }]);
    setFocalNodeId(nodeId);
    if (onNavigate) onNavigate(nodeId, entityType);
  }, [onNavigate, entityType]);

  const handleBackFocus = useCallback(() => {
    setHistoryStack((prev) => {
      if (prev.length <= 1) return prev;
      const next = prev.slice(0, prev.length - 1);
      const target = next[next.length - 1];
      setFocalNodeId(target.id);
      return next;
    });
  }, []);

  const loadGraphForLens = useCallback(
    async (nodeId: string, lens: LensType) => {
      if (!nodeId) return;
      setLoading(true);
      try {
        const res = await apiClient<any>(`/api/catalog/nodes/${encodeURIComponent(nodeId)}/visualize-lens`, {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({
            tenant_id: tenantId,
            node_name: historyStackRef.current.find((h: { id: string; name: string }) => h.id === nodeId)?.name ?? '',
            lens_type: lens,
            depth: 2,
            include_indirect: false,
          }),
        });

        const data = (res as any)?.data ?? res;
        if (data && Array.isArray(data.nodes)) {
          setBreadcrumb(data.breadcrumb_path || '');
          const enrichedNodes = data.nodes.map((n: any) => ({
            ...n,
            onFocus: () => handleShiftFocus(n.id, n.node_name),
          }));

          const { layoutNodes, layoutEdges } = layoutLensNodes(enrichedNodes, data.edges || [], lens);
          setNodes(layoutNodes);
          setEdges(layoutEdges);
        }
      } catch (err) {
        console.error('[CognitiveGraphStudio] Failed loading lens graph:', err);
      } finally {
        setLoading(false);
      }
    },
    [tenantId, handleShiftFocus, setNodes, setEdges]
  );

  useEffect(() => {
    loadGraphForLens(focalNodeId, activeLens);
  }, [focalNodeId, activeLens, loadGraphForLens]);

  const currentLensInfo = useMemo(() => {
    return LENS_OPTIONS.find((l) => l.type === activeLens) || LENS_OPTIONS[0];
  }, [activeLens]);

  return (
    <div
      style={{
        width: '100%',
        height,
        display: 'flex',
        flexDirection: 'column',
        background: '#050D1A',
        borderRadius: 12,
        overflow: 'hidden',
        border: '1px solid rgba(255, 255, 255, 0.08)',
        position: 'relative',
      }}
    >
      {/* Studio Header Toolbar & Lens Switcher */}
      <div
        style={{
          padding: '10px 16px',
          background: 'rgba(7, 21, 38, 0.95)',
          backdropFilter: 'blur(12px)',
          borderBottom: '1px solid rgba(255, 255, 255, 0.08)',
          display: 'flex',
          justifyContent: 'space-between',
          alignItems: 'center',
          flexWrap: 'wrap',
          gap: 10,
          zIndex: 10,
        }}
      >
        <div style={{ display: 'flex', alignItems: 'center', gap: 10, flexWrap: 'wrap' }}>
          <span style={{ fontSize: 11, fontWeight: 800, color: '#94A3B8', textTransform: 'uppercase', letterSpacing: '0.05em' }}>
            Cognitive Lenses:
          </span>

          <div style={{ display: 'flex', gap: 4, background: 'rgba(15, 23, 42, 0.8)', padding: 3, borderRadius: 8, border: '1px solid rgba(255, 255, 255, 0.06)' }}>
            {LENS_OPTIONS.map((lens) => {
              const active = activeLens === lens.type;
              return (
                <button
                  key={lens.type}
                  onClick={() => setActiveLens(lens.type)}
                  style={{
                    display: 'flex',
                    alignItems: 'center',
                    gap: 6,
                    padding: '5px 10px',
                    borderRadius: 6,
                    fontSize: 11.5,
                    fontWeight: active ? 700 : 500,
                    color: active ? '#FFFFFF' : '#94A3B8',
                    background: active ? lens.color : 'transparent',
                    border: 'none',
                    cursor: 'pointer',
                    transition: 'all 0.18s ease',
                    boxShadow: active ? `0 0 14px ${lens.color}55` : 'none',
                    whiteSpace: 'nowrap',
                  }}
                >
                  {lens.icon}
                  <span>{lens.shortLabel}</span>
                </button>
              );
            })}
          </div>
        </div>

        {/* History / Focus Navigation Stack */}
        <div style={{ display: 'flex', alignItems: 'center', gap: 10 }}>
          {breadcrumb && activeLens === 'TAXONOMY_HIERARCHY' && (
            <span style={{ fontSize: 11, color: '#6EE7B7', fontWeight: 600, fontFamily: 'monospace' }}>
              🏛️ {breadcrumb}
            </span>
          )}

          {historyStack.length > 1 && (
            <button
              onClick={handleBackFocus}
              style={{
                display: 'flex',
                alignItems: 'center',
                gap: 4,
                padding: '4px 10px',
                borderRadius: 6,
                fontSize: 11,
                fontWeight: 600,
                background: 'rgba(30, 41, 59, 0.8)',
                color: '#E2E8F0',
                border: '1px solid rgba(255, 255, 255, 0.15)',
                cursor: 'pointer',
              }}
            >
              <ArrowBackIcon sx={{ fontSize: 13 }} />
              <span>Back ({historyStack[historyStack.length - 2]?.name || 'Previous'})</span>
            </button>
          )}
        </div>
      </div>

      {/* Lens Sub-banner description */}
      <div
        style={{
          padding: '6px 16px',
          background: 'rgba(15, 23, 42, 0.6)',
          borderBottom: '1px solid rgba(255, 255, 255, 0.05)',
          display: 'flex',
          justifyContent: 'space-between',
          alignItems: 'center',
          fontSize: 11,
          color: '#94A3B8',
        }}
      >
        <div>
          <strong style={{ color: currentLensInfo.color }}>{currentLensInfo.label}:</strong> {currentLensInfo.description}
        </div>
        <div style={{ display: 'flex', alignItems: 'center', gap: 6 }}>
          <span>Focal Anchor:</span>
          <span style={{ color: '#38BDF8', fontWeight: 700, fontFamily: 'monospace' }}>
            {historyStack[historyStack.length - 1]?.name || focalNodeId}
          </span>
        </div>
      </div>

      {/* Main Canvas */}
      <div style={{ flex: 1, position: 'relative' }}>
        {loading && (
          <div
            style={{
              position: 'absolute',
              inset: 0,
              display: 'flex',
              flexDirection: 'column',
              alignItems: 'center',
              justifyContent: 'center',
              background: 'rgba(5, 13, 26, 0.75)',
              backdropFilter: 'blur(4px)',
              zIndex: 20,
              gap: 12,
            }}
          >
            <CircularProgress size={28} sx={{ color: currentLensInfo.color }} />
            <span style={{ fontSize: 12, fontWeight: 600, color: '#E2E8F0' }}>
              Projecting {currentLensInfo.label}...
            </span>
          </div>
        )}

        <ReactFlow
          nodes={nodes}
          edges={edges}
          onNodesChange={onNodesChange}
          onEdgesChange={onEdgesChange}
          nodeTypes={nodeTypes}
          fitView
          attributionPosition="bottom-right"
        >
          <Background color="#1E293B" gap={24} size={1.2} variant={BackgroundVariant.Dots} />
          <Controls style={{ background: '#0F172A', border: '1px solid rgba(255, 255, 255, 0.1)', borderRadius: 8 }} />
          <MiniMap
            nodeColor={(n) => (n.data?.is_focal ? '#38BDF8' : '#6366F1')}
            maskColor="rgba(5, 13, 26, 0.85)"
            style={{ background: '#071526', border: '1px solid rgba(255, 255, 255, 0.1)', borderRadius: 8 }}
          />
        </ReactFlow>
      </div>
    </div>
  );
};

export default CognitiveGraphStudio;
