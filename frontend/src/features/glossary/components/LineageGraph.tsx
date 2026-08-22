import * as React from 'react';
import { useState, useMemo, useEffect, useCallback } from 'react';
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

// ─────────────────────────────────────────────
// Theme tokens
// ─────────────────────────────────────────────
export const C = {
  bg: '#0A0C10',
  sidebar: '#0F1117',
  panel: '#13161E',
  border: 'rgba(255,255,255,0.07)',
  borderStrong: 'rgba(255,255,255,0.12)',
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

// ─────────────────────────────────────────────
// Shared utility
// ─────────────────────────────────────────────
export interface ParsedPath {
  datasource: string;
  schema: string;
  table: string;
  column: string;
}

export function parsePath(path: string): ParsedPath {
  if (!path) return { datasource: 'N/A', schema: 'N/A', table: 'N/A', column: 'N/A' };
  // API endpoints have a different shape
  if (path.startsWith('api_endpoint/')) {
    const slashParts = path.replace(/^api_endpoint\//, '').split('/');
    return { datasource: 'REST API', schema: slashParts[0] || 'N/A', table: slashParts[1] || 'N/A', column: slashParts[2] || 'N/A' };
  }
  const cleanPath = path.startsWith('/') ? path.substring(1) : path;
  const dotParts = cleanPath.split('.');
  if (dotParts.length >= 4) {
    return { datasource: dotParts[0], schema: dotParts[1], table: dotParts[2], column: dotParts[3] };
  }
  const slashParts = cleanPath.split('/');
  if (slashParts.length >= 4) {
    return { datasource: slashParts[0], schema: slashParts[1], table: slashParts[2], column: slashParts[3] };
  }
  if (slashParts.length === 3) {
    return { datasource: slashParts[0], schema: 'public', table: slashParts[1], column: slashParts[2] };
  }
  if (slashParts.length === 2) {
    return { datasource: slashParts[0], schema: 'public', table: slashParts[0], column: slashParts[1] };
  }
  return { datasource: 'Default Datasource', schema: 'public', table: slashParts[0] || 'table', column: slashParts[1] || path };
}

// ─────────────────────────────────────────────
// Node components
// ─────────────────────────────────────────────

export const resolveNodeType = (data: any): 'business_object' | 'business_term' | 'semantic_term' | 'calculation' | 'table' | 'column' | 'api_endpoint' | 'rule' => {
  const path = (data.path || data.qualified_path || '').toLowerCase();
  const rawType = (data.nodeType || data.catalogTypeName || data.catalog_type_name || data.catalog_type || data.node_type || '').toLowerCase();

  if (rawType.includes('business_object') || rawType === 'bo' || path.startsWith('business_object/') || path.startsWith('/business_object/')) {
    return 'business_object';
  }
  if (rawType.includes('business_term') || path.startsWith('business_term/') || path.startsWith('/business_term/')) {
    return 'business_term';
  }
  if (rawType.includes('semantic') || path.startsWith('semantic.') || path.startsWith('semantic_term/') || path.startsWith('/semantic_term/')) {
    return 'semantic_term';
  }
  if (rawType.includes('calculated') || !!data.isCalc || !!data.formula) {
    return 'calculation';
  }
  if (rawType === 'column' || rawType === 'database_column' || path.startsWith('column/')) {
    return 'column';
  }
  if (rawType === 'table' || rawType === 'database_table' || path.startsWith('table/')) {
    return 'table';
  }
  if (rawType.includes('api_endpoint') || path.startsWith('api_endpoint/') || path.startsWith('/api_endpoint/')) {
    return 'api_endpoint';
  }
  if (rawType.includes('rule')) {
    return 'rule';
  }
  return 'business_term';
};

// 1. Upstream Business Term / Business Object / Calculation Node
const BusinessTermNode: React.FC<{ data: any }> = ({ data }) => {
  const [localExpanded, setLocalExpanded] = useState(false);
  const isExpanded = data.isExpanded !== undefined ? data.isExpanded : localExpanded;

  const nodeCategory = resolveNodeType(data);
  const isBO = nodeCategory === 'business_object';
  const isBusTerm = nodeCategory === 'business_term';
  const isSemantic = nodeCategory === 'semantic_term';
  const isCalc = nodeCategory === 'calculation';
  const isTable = nodeCategory === 'table';
  const isColumn = nodeCategory === 'column';
  const isApi = nodeCategory === 'api_endpoint';
  
  const icon = isBO ? '🏢' : isBusTerm ? '💼' : isSemantic ? '🧠' : isCalc ? '🧮' : isTable ? '📊' : isColumn ? '📋' : isApi ? '🌐' : '💼';
  const labelText = isBO ? 'Business Object' : isBusTerm ? 'Business Term' : isSemantic ? 'Semantic Term' : isCalc ? 'Calculation Dependency' : isTable ? 'Database Table' : isColumn ? 'Database Column' : isApi ? 'API Endpoint' : (data.catalogTypeName || 'Business Term');
  const color = isBO ? '#A855F7' : isBusTerm ? C.teal : isSemantic ? C.accent : isCalc ? '#38BDF8' : isTable ? C.blue : isColumn ? C.blue : isApi ? '#EC4899' : C.teal;

  const props = typeof data.properties === 'string'
    ? (() => { try { return JSON.parse(data.properties); } catch { return {}; } })()
    : (data.properties || {});

  const hasExtra = isBO || !!data.formula || !!props.formula || Object.keys(props).length > 0 || (data.description && data.description.length > 50);

  return (
    <div
      style={{
        background: C.panel,
        border: `1px solid ${color}66`,
        borderRadius: 10,
        minWidth: 240,
        maxWidth: isExpanded ? 340 : 300,
        overflow: 'hidden',
        boxShadow: isBO ? '0 0 20px rgba(168,85,247,0.2)' : `0 4px 20px rgba(0,0,0,0.4)`,
        transition: 'all 0.15s ease',
      }}
      onMouseEnter={(e) => (e.currentTarget.style.borderColor = color, e.currentTarget.style.transform = 'translateY(-2px)')}
      onMouseLeave={(e) => (e.currentTarget.style.borderColor = `${color}66`, e.currentTarget.style.transform = 'translateY(0)')}
    >
      <Handle
        type="target"
        position={Position.Left}
        style={{ background: color, width: 8, height: 8, border: `2px solid ${C.panel}` }}
      />
      <div style={{
        background: `${color}18`,
        padding: '8px 12px',
        borderBottom: `1px solid ${C.border}`,
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'space-between',
      }}>
        <div style={{ display: 'flex', alignItems: 'center', gap: 6 }}>
          <span style={{ fontSize: 14 }}>{icon}</span>
          <span style={{ color: color, fontWeight: 700, fontSize: 11, letterSpacing: '0.04em', textTransform: 'uppercase' }}>
            {labelText}
          </span>
        </div>
        <div style={{ display: 'flex', alignItems: 'center', gap: 6 }}>
          <button
            className="nodrag nopan"
            title="Inspect Node Details"
            onClick={(e) => {
              e.stopPropagation();
              if (data.onInspectNode) data.onInspectNode(data);
            }}
            style={{
              background: 'rgba(255,255,255,0.08)',
              border: `1px solid ${C.border}`,
              color: C.text,
              borderRadius: 4,
              padding: '2px 5px',
              fontSize: 11,
              cursor: 'pointer',
              display: 'flex',
              alignItems: 'center',
            }}
          >
            ℹ️
          </button>
          <button
            className="nodrag nopan"
            title="Make this node the focal point"
            onClick={(e) => {
              e.stopPropagation();
              if (data.onSelectNode) data.onSelectNode(data.rawId || data.id);
            }}
            style={{
              fontSize: 10,
              color: color,
              fontWeight: 700,
              background: `${color}22`,
              border: `1px solid ${color}44`,
              padding: '2px 6px',
              borderRadius: 4,
              cursor: 'pointer',
            }}
          >
            Focus 🔍
          </button>
        </div>
      </div>
      <div
        style={{ padding: '10px 12px', cursor: 'pointer' }}
        onClick={() => data.onInspectNode && data.onInspectNode(data)}
      >
        <div style={{ color: C.text, fontWeight: 700, fontSize: 13, wordBreak: 'break-word' }}>
          {data.label}
        </div>
        {/* Description intentionally hidden in lineage nodes — visible only on
            the details tab and in the list/summary page (with hover). */}
        {data.formula && (
          <div style={{ marginTop: 6, padding: '3px 6px', background: 'rgba(56,189,248,0.1)', borderRadius: 4, border: '1px solid rgba(56,189,248,0.2)', fontSize: 10, fontFamily: 'monospace', color: '#38BDF8' }}>
            {data.formula}
          </div>
        )}
        {data.path && (
          <div style={{ color: C.textMuted, fontSize: 10, fontFamily: 'monospace', marginTop: 6, background: 'rgba(255,255,255,0.03)', padding: '2px 4px', borderRadius: 4, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: isExpanded ? 'normal' : 'nowrap' }}>
            {data.path}
          </div>
        )}

        {isExpanded && (
          <div style={{ marginTop: 8, paddingTop: 8, borderTop: `1px solid ${C.border}`, display: 'flex', flexDirection: 'column', gap: 6 }}>
            {isBO && (
              <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', fontSize: 11 }}>
                <span style={{ color: C.textMuted }}>Category:</span>
                <span style={{ color: '#C084FC', fontWeight: 600 }}>Core Business Object</span>
              </div>
            )}
            {Object.keys(props).length > 0 && (
              <div style={{ fontSize: 10, background: 'rgba(0,0,0,0.2)', padding: '4px 6px', borderRadius: 4, maxHeight: 120, overflowY: 'auto' }}>
                {Object.entries(props).slice(0, 5).map(([k, v]) => (
                  <div key={k} style={{ display: 'flex', justifyContent: 'space-between', gap: 8, padding: '1px 0' }}>
                    <span style={{ color: C.textMuted, fontFamily: 'monospace' }}>{k}:</span>
                    <span style={{ color: C.text, fontFamily: 'monospace', overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
                      {typeof v === 'object' ? JSON.stringify(v) : String(v)}
                    </span>
                  </div>
                ))}
              </div>
            )}
          </div>
        )}

        {hasExtra && (
          <button
            className="nodrag nopan"
            onClick={(e) => {
              e.stopPropagation();
              if (data.onToggleExpand) {
                data.onToggleExpand(data.rawId || data.id);
              } else {
                setLocalExpanded(!isExpanded);
              }
            }}
            style={{
              background: isExpanded ? 'rgba(255,255,255,0.08)' : `${color}18`,
              border: `1px solid ${isExpanded ? C.border : color}44`,
              color: isExpanded ? C.text : color,
              padding: '3px 8px',
              borderRadius: 4,
              fontSize: 10,
              fontWeight: 700,
              cursor: 'pointer',
              width: '100%',
              marginTop: 8,
            }}
          >
            {isExpanded ? '▲ Collapse' : '▼ Expand'}
          </button>
        )}
      </div>
      <Handle
        type="source"
        position={Position.Right}
        style={{ background: color, width: 8, height: 8, border: `2px solid ${C.panel}` }}
      />
    </div>
  );
};

// 2. Focal Term Node (Center)
const FocalTermNode: React.FC<{ data: any }> = ({ data }) => {
  const isCalc = !!data.formula || data.termType === 'calculated' || data.termType === 'preaggregated';
  const isBO = data.isBO || data.catalogTypeName === 'business_object' || (data.path || '').startsWith('business_object/') || (data.path || '').startsWith('/business_object/');
  const color = isBO ? '#A855F7' : isCalc ? '#38BDF8' : C.accent;
  const icon = isBO ? '🏢' : isCalc ? '🧮' : '🧠';
  const title = isBO ? 'Business Object (Focal)' : isCalc ? 'Calculated Term (Focal)' : (data.focalLabel || 'Semantic Term (Focal)');

  return (
    <div style={{
      background: C.panel,
      border: `2px solid ${color}`,
      borderRadius: 12,
      minWidth: 280,
      maxWidth: 340,
      overflow: 'hidden',
      boxShadow: isBO ? '0 0 24px rgba(168,85,247,0.25)' : isCalc ? '0 0 24px rgba(56,189,248,0.25)' : C.accentGlow,
      transition: 'all 0.2s ease',
    }}>
      <Handle
        type="target"
        position={Position.Left}
        style={{ background: color, width: 9, height: 9, border: `2px solid ${C.panel}` }}
      />
      <div style={{
        background: isBO ? 'rgba(168,85,247,0.15)' : isCalc ? 'rgba(56,189,248,0.15)' : C.accentDim,
        padding: '10px 14px',
        borderBottom: `1px solid ${C.border}`,
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'space-between',
      }}>
        <div style={{ display: 'flex', alignItems: 'center', gap: 6 }}>
          <span style={{ fontSize: 16 }}>{icon}</span>
          <span style={{ color: color, fontWeight: 800, fontSize: 11, letterSpacing: '0.05em', textTransform: 'uppercase' }}>
            {title}
          </span>
        </div>
        <div style={{ display: 'flex', alignItems: 'center', gap: 6 }}>
          <button
            className="nodrag nopan"
            title="Inspect Details"
            onClick={(e) => {
              e.stopPropagation();
              if (data.onInspectNode) data.onInspectNode(data);
            }}
            style={{
              background: 'rgba(255,255,255,0.08)',
              border: `1px solid ${C.border}`,
              color: C.text,
              borderRadius: 4,
              padding: '2px 5px',
              fontSize: 11,
              cursor: 'pointer',
              display: 'flex',
              alignItems: 'center',
            }}
          >
            ℹ️
          </button>
        </div>
      </div>
      <div
        style={{ padding: '12px 14px', cursor: 'pointer' }}
        onClick={() => data.onInspectNode && data.onInspectNode(data)}
      >
        <div style={{ color: '#fff', fontWeight: 800, fontSize: 15, wordBreak: 'break-word' }}>
          {data.label}
        </div>
        {/* Description hidden in lineage nodes — visible only on details tab. */}
        {data.formula && (
          <div style={{
            marginTop: 8, padding: '6px 8px', background: 'rgba(0,0,0,0.3)',
            borderRadius: 6, border: '1px solid rgba(56,189,248,0.3)',
            fontFamily: 'monospace', fontSize: 11, color: '#38BDF8'
          }}>
            fx: {data.formula}
          </div>
        )}
        {data.path && (
          <div style={{ color: C.textMuted, fontSize: 11, fontFamily: 'monospace', marginTop: 8, background: 'rgba(255,255,255,0.04)', padding: '4px 6px', borderRadius: 4 }}>
            {data.path}
          </div>
        )}
      </div>
      <Handle
        type="source"
        position={Position.Right}
        style={{ background: isCalc ? '#38BDF8' : C.accent, width: 9, height: 9, border: `2px solid ${C.panel}` }}
      />
    </div>
  );
};

// 3. Datasource Node (Expandable)
interface DatasourceNodeData {
  datasourceName: string;
  datasourceType?: string;
  datasourceHost?: string;
  totalColumns: number;
  totalTables: number;
  schemas: Record<string, any>;
  isExpanded: boolean;
  onToggleExpand: (name: string) => void;
  onInspectNode: (data: any) => void;
}

const DatasourceNode: React.FC<{ data: DatasourceNodeData }> = ({ data }) => {
  const isExpanded = !!data.isExpanded;
  const dsType = (data.datasourceType || '').toLowerCase();
  const dsName = (data.datasourceName || '').toLowerCase();
  const isApi = dsType.includes('api') || dsName.includes('api') || dsName.includes('salesforce') || dsName.includes('servicenow');
  const dsIcon = isApi ? '🌐' : dsType.includes('snowflake') ? '❄️' : dsType.includes('postgres') ? '🐘' : '🗄️';

  return (
    <div style={{
      background: C.panel,
      border: `1px solid ${isExpanded ? (isApi ? C.accent : C.blue) : 'rgba(96,165,250,0.3)'}`,
      borderRadius: 10,
      minWidth: isExpanded ? 340 : 250,
      maxWidth: 420,
      overflow: 'hidden',
      boxShadow: isExpanded ? `0 0 20px ${isApi ? 'rgba(99,102,241,0.25)' : 'rgba(96,165,250,0.25)'}` : '0 4px 20px rgba(0,0,0,0.4)',
      transition: 'all 0.2s ease',
    }}>
      <Handle
        type="target"
        position={Position.Left}
        style={{ background: isApi ? C.accent : C.blue, width: 8, height: 8, border: `2px solid ${C.panel}` }}
      />
      <div style={{
        background: isExpanded ? (isApi ? 'rgba(99,102,241,0.15)' : 'rgba(96,165,250,0.15)') : 'rgba(96,165,250,0.08)',
        padding: '10px 12px',
        borderBottom: `1px solid ${C.border}`,
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'space-between',
        gap: 8,
      }}>
        <div style={{ display: 'flex', alignItems: 'center', gap: 8, overflow: 'hidden' }}>
          <span style={{ fontSize: 18 }}>{dsIcon}</span>
          <div>
            <div style={{ color: C.text, fontWeight: 700, fontSize: 13, fontFamily: 'monospace', overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
              {data.datasourceName}
            </div>
            <div style={{ color: C.textMuted, fontSize: 10, marginTop: 1 }}>
              {data.totalColumns} {isApi ? 'field' : 'column'}{data.totalColumns !== 1 ? 's' : ''} · {data.totalTables} {isApi ? 'endpoint' : 'table'}{data.totalTables !== 1 ? 's' : ''}
            </div>
          </div>
        </div>
        <div style={{ display: 'flex', alignItems: 'center', gap: 6 }}>
          <button
            className="nodrag nopan"
            title="Inspect Datasource Details"
            onClick={(e) => {
              e.stopPropagation();
              if (data.onInspectNode) data.onInspectNode(data);
            }}
            style={{
              background: 'rgba(255,255,255,0.08)',
              border: `1px solid ${C.border}`,
              color: C.text,
              borderRadius: 4,
              padding: '2px 5px',
              fontSize: 11,
              cursor: 'pointer',
              display: 'flex',
              alignItems: 'center',
            }}
          >
            ℹ️
          </button>
          <button
            className="nodrag nopan"
            onClick={(e) => {
              e.stopPropagation();
              if (data.onToggleExpand) data.onToggleExpand(data.datasourceName);
            }}
            style={{
              background: isExpanded ? 'rgba(255,255,255,0.1)' : C.accentDim,
              border: `1px solid ${isExpanded ? C.border : C.accent}`,
              color: isExpanded ? C.text : C.accent,
              padding: '4px 8px',
              borderRadius: 6,
              fontSize: 11,
              fontWeight: 700,
              cursor: 'pointer',
              whiteSpace: 'nowrap',
              display: 'flex',
              alignItems: 'center',
              gap: 4,
            }}
          >
            {isExpanded ? '▲ Collapse' : '▼ Expand'}
          </button>
        </div>
      </div>
      {isExpanded && (
        <div className="nodrag nopan" style={{ padding: '10px 12px', maxHeight: 360, overflowY: 'auto', display: 'flex', flexDirection: 'column', gap: 10 }}>
          {Object.entries(data.schemas || {}).map(([schemaName, schema]: [string, any]) => (
            <div key={schemaName} style={{ background: 'rgba(0,0,0,0.2)', borderRadius: 6, padding: '8px 10px', border: `1px solid ${C.border}` }}>
              <div style={{ display: 'flex', alignItems: 'center', gap: 6, marginBottom: 6 }}>
                <span style={{ fontSize: 12 }}>{isApi ? '📦' : '📁'}</span>
                <span style={{ fontSize: 11, fontWeight: 700, color: C.teal, fontFamily: 'monospace' }}>
                  {schemaName}
                </span>
                <span style={{ fontSize: 10, color: C.textMuted }}>{isApi ? 'resource' : 'schema'}</span>
              </div>
              <div style={{ display: 'flex', flexDirection: 'column', gap: 6, paddingLeft: 12, borderLeft: `1px dashed rgba(255,255,255,0.1)` }}>
                {Object.entries(schema.tables || {}).map(([tableName, table]: [string, any]) => (
                  <div key={tableName} style={{ background: 'rgba(255,255,255,0.02)', borderRadius: 4, padding: '6px 8px' }}>
                    <div style={{ display: 'flex', alignItems: 'center', gap: 6, marginBottom: 4 }}>
                      <span style={{ fontSize: 12 }}>📊</span>
                      <span style={{ fontSize: 11, fontWeight: 700, color: C.blue, fontFamily: 'monospace' }}>
                        {tableName}
                      </span>
                    </div>
                    <div style={{ display: 'flex', flexDirection: 'column', gap: 2, paddingLeft: 12 }}>
                      {table.columns.map((col: any) => (
                        <div key={col.id} style={{ fontSize: 10, color: C.text, fontFamily: 'monospace', display: 'flex', justifyContent: 'space-between', padding: '2px 4px' }}>
                          <span>• {col.name}</span>
                          {col.dataType && <span style={{ color: C.textMuted }}>{col.dataType}</span>}
                        </div>
                      ))}
                    </div>
                  </div>
                ))}
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  );
};

// 4. Grouped Business Object Node
const GroupedBONode: React.FC<{ data: any }> = ({ data }) => {
  const isExpanded = !!data.isExpanded;
  const color = '#A855F7';
  return (
    <div style={{
      background: C.panel,
      border: `1px solid ${color}66`,
      borderRadius: 10,
      minWidth: 260,
      maxWidth: 320,
      overflow: 'hidden',
      boxShadow: '0 4px 20px rgba(0,0,0,0.4)',
    }}>
      <div style={{
        background: `${color}18`,
        padding: '8px 12px',
        borderBottom: `1px solid ${C.border}`,
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'space-between',
      }}>
        <div style={{ display: 'flex', alignItems: 'center', gap: 6 }}>
          <span style={{ fontSize: 14 }}>🏢</span>
          <span style={{ color: color, fontWeight: 700, fontSize: 11, letterSpacing: '0.04em', textTransform: 'uppercase' }}>
            Business Objects Group
          </span>
        </div>
        <span style={{ fontSize: 11, color: C.text, fontWeight: 700 }}>{data.items?.length || 0}</span>
      </div>
      <div style={{ padding: '10px 12px' }}>
        {isExpanded && data.items && (
          <div style={{ display: 'flex', flexDirection: 'column', gap: 6, maxHeight: 280, overflowY: 'auto' }}>
            {data.items.map((bo: any) => (
              <div
                key={bo.id}
                onClick={() => data.onSelectNode && data.onSelectNode(bo.id)}
                style={{
                  background: 'rgba(168,85,247,0.08)',
                  border: `1px solid ${color}44`,
                  borderRadius: 4,
                  padding: '6px 8px',
                  cursor: 'pointer',
                  fontSize: 12,
                  color: C.text,
                }}
              >
                <div style={{ fontWeight: 600 }}>{bo.node_name || bo.label}</div>
                {/* Description hidden in lineage BO list — visible on details tab. */}
              </div>
            ))}
          </div>
        )}
        <button
          className="nodrag nopan"
          onClick={(e) => { e.stopPropagation(); data.onToggleExpand && data.onToggleExpand(); }}
          style={{
            background: isExpanded ? 'rgba(255,255,255,0.1)' : C.accentDim,
            border: `1px solid ${isExpanded ? C.border : C.accent}`,
            color: isExpanded ? C.text : C.accent,
            padding: '4px 8px',
            borderRadius: 6,
            fontSize: 11,
            fontWeight: 700,
            cursor: 'pointer',
            width: '100%',
            marginTop: 8,
          }}
        >
          {isExpanded ? '▲ Collapse' : `▼ Expand (${data.items?.length || 0} BOs)`}
        </button>
      </div>
      <Handle
        type="source"
        position={Position.Right}
        style={{ background: color, width: 8, height: 8, border: `2px solid ${C.panel}` }}
      />
    </div>
  );
};

const nodeTypes = {
  upstream: BusinessTermNode,
  focalTerm: FocalTermNode,
  datasource: DatasourceNode,
  groupedBO: GroupedBONode,
};

// ─────────────────────────────────────────────
// Inspector Drawer
// ─────────────────────────────────────────────
interface DrawerState {
  open: boolean;
  type: 'node' | 'edge';
  data: any;
}

const InspectorDrawer: React.FC<{ drawer: DrawerState; onClose: () => void; onNavigate?: (id: string) => void }> = ({ drawer, onClose, onNavigate }) => {
  if (!drawer.open) return null;
  return (
    <div style={{
      position: 'absolute',
      top: 0,
      right: 0,
      bottom: 0,
      width: 360,
      maxWidth: '90%',
      background: C.panel,
      borderLeft: `1px solid ${C.border}`,
      boxShadow: '-8px 0 30px rgba(0,0,0,0.6)',
      zIndex: 20,
      display: 'flex',
      flexDirection: 'column',
      animation: 'slideIn 0.2s ease-out',
    }}>
      <div style={{
        padding: '16px 20px',
        borderBottom: `1px solid ${C.border}`,
        background: 'rgba(255,255,255,0.03)',
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'space-between',
      }}>
        <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
          <span style={{ fontSize: 16 }}>{drawer.type === 'node' ? 'ℹ️' : '🔗'}</span>
          <span style={{ fontWeight: 800, fontSize: 13, textTransform: 'uppercase', letterSpacing: '0.05em', color: C.accent }}>
            {drawer.type === 'node' ? 'Node Inspector' : 'Edge Attribute Inspector'}
          </span>
        </div>
        <button
          onClick={onClose}
          style={{
            background: 'transparent',
            border: 'none',
            color: C.textMuted,
            cursor: 'pointer',
            fontSize: 16,
            padding: '4px 8px',
            borderRadius: 4,
          }}
        >
          ✕
        </button>
      </div>
      <div style={{ flex: 1, padding: '20px', overflowY: 'auto', display: 'flex', flexDirection: 'column', gap: 16 }}>
        {drawer.type === 'node' ? (
          <>
            <div>
              <div style={{ fontSize: 11, fontWeight: 700, color: C.textMuted, textTransform: 'uppercase', marginBottom: 4 }}>
                Node Name
              </div>
              <div style={{ fontSize: 16, fontWeight: 800, color: '#fff' }}>
                {drawer.data?.node_name || drawer.data?.label || drawer.data?.name}
              </div>
            </div>
            {drawer.data?.catalog_type_name && (
              <div>
                <div style={{ fontSize: 11, fontWeight: 700, color: C.textMuted, textTransform: 'uppercase', marginBottom: 4 }}>
                  Type
                </div>
                <div style={{ fontSize: 13, color: C.text }}>{drawer.data.catalog_type_name}</div>
              </div>
            )}
            {drawer.data?.description && (
              <div>
                <div style={{ fontSize: 11, fontWeight: 700, color: C.textMuted, textTransform: 'uppercase', marginBottom: 4 }}>
                  Description
                </div>
                <div style={{ fontSize: 13, color: C.text, lineHeight: 1.5, background: 'rgba(255,255,255,0.02)', padding: '8px 12px', borderRadius: 6, border: `1px solid ${C.border}` }}>
                  {drawer.data.description}
                </div>
              </div>
            )}
            {drawer.data?.formula && (
              <div>
                <div style={{ fontSize: 11, fontWeight: 700, color: '#38BDF8', textTransform: 'uppercase', marginBottom: 4 }}>
                  Calculation Formula
                </div>
                <div style={{ fontSize: 12, fontFamily: 'monospace', color: '#38BDF8', background: 'rgba(56,189,248,0.1)', padding: '8px 12px', borderRadius: 6, border: '1px solid rgba(56,189,248,0.3)' }}>
                  {drawer.data.formula}
                </div>
              </div>
            )}
            {drawer.data?.qualified_path && (
              <div>
                <div style={{ fontSize: 11, fontWeight: 700, color: C.textMuted, textTransform: 'uppercase', marginBottom: 4 }}>
                  Qualified Path
                </div>
                <div style={{ fontSize: 11, fontFamily: 'monospace', color: C.textMuted, background: 'rgba(0,0,0,0.3)', padding: '6px 10px', borderRadius: 6 }}>
                  {drawer.data.qualified_path}
                </div>
              </div>
            )}
            <div>
              <div style={{ fontSize: 11, fontWeight: 700, color: C.textMuted, textTransform: 'uppercase', marginBottom: 6 }}>
                Node Attributes
              </div>
              <div style={{ background: 'rgba(0,0,0,0.3)', borderRadius: 6, padding: '10px 12px', fontSize: 11, fontFamily: 'monospace', color: C.textMuted, overflowX: 'auto', maxHeight: 200 }}>
                <pre style={{ margin: 0 }}>
                  {JSON.stringify(drawer.data?.properties || drawer.data, null, 2)}
                </pre>
              </div>
            </div>
            {drawer.data?.id && onNavigate && (
              <button
                onClick={() => {
                  onNavigate(drawer.data.rawId || drawer.data.id);
                  onClose();
                }}
                style={{
                  marginTop: 10,
                  background: C.accent,
                  color: '#fff',
                  border: 'none',
                  padding: '10px 16px',
                  borderRadius: 6,
                  fontWeight: 700,
                  fontSize: 13,
                  cursor: 'pointer',
                }}
              >
                Make This Node Focal Point 🔍
              </button>
            )}
          </>
        ) : (
          <>
            <div>
              <div style={{ fontSize: 11, fontWeight: 700, color: C.textMuted, textTransform: 'uppercase', marginBottom: 4 }}>
                Relationship / Predicate
              </div>
              <div style={{ fontSize: 16, fontWeight: 800, color: C.accent, fontFamily: 'monospace' }}>
                {drawer.data?.predicate || drawer.data?.relationship_type || drawer.data?.label || 'relationship'}
              </div>
            </div>
            <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 12 }}>
              <div>
                <div style={{ fontSize: 11, fontWeight: 700, color: C.textMuted, textTransform: 'uppercase', marginBottom: 4 }}>
                  Source
                </div>
                <div style={{ fontSize: 12, fontWeight: 600, color: '#fff' }}>
                  {drawer.data?.source_name || drawer.data?.source || 'Source'}
                </div>
              </div>
              <div>
                <div style={{ fontSize: 11, fontWeight: 700, color: C.textMuted, textTransform: 'uppercase', marginBottom: 4 }}>
                  Target
                </div>
                <div style={{ fontSize: 12, fontWeight: 600, color: '#fff' }}>
                  {drawer.data?.target_name || drawer.data?.target || 'Target'}
                </div>
              </div>
            </div>
            <div>
              <div style={{ fontSize: 11, fontWeight: 700, color: C.textMuted, textTransform: 'uppercase', marginBottom: 6 }}>
                Edge Attributes
              </div>
              <div style={{ background: 'rgba(0,0,0,0.3)', borderRadius: 6, padding: '10px 12px', fontSize: 11, fontFamily: 'monospace', color: C.textMuted, overflowX: 'auto', maxHeight: 220 }}>
                <pre style={{ margin: 0 }}>
                  {JSON.stringify(drawer.data, null, 2)}
                </pre>
              </div>
            </div>
          </>
        )}
      </div>
    </div>
  );
};

// ─────────────────────────────────────────────
// LineageGraph Component
// ─────────────────────────────────────────────
export interface LineageGraphProps {
  // Data
  focalTerm: any;
  upstreamNodes: any[];          // Business Objects, Calculations, related business terms
  downstreamNodes: any[];        // Connected semantic terms (BT lineage) or physical columns/datasources
  edges: any[];                  // Edges between focal and upstream/downstream
  // Optional downstream datasources (for full hierarchy)
  datasources?: Array<{
    name: string;
    type?: string;
    host?: string;
    totalColumns: number;
    totalTables: number;
    schemas: Record<string, any>;
  }>;
  // Callbacks
  onNavigate?: (id: string) => void;
  // Config
  focalLabel?: string;            // e.g., 'Business Term (Focal)' or 'Semantic Term (Focal)'
  showBOsByDefault?: boolean;
  showCalcsByDefault?: boolean;
  showDatasourcesByDefault?: boolean;
  showDatasourceLayer?: boolean;  // Show the datasources layer toggle (semantic terms only)
  height?: number | string;
  emptyMessage?: string;
}

export const LineageGraph: React.FC<LineageGraphProps> = ({
  focalTerm,
  upstreamNodes,
  downstreamNodes,
  edges,
  datasources,
  onNavigate,
  focalLabel = 'Semantic Term (Focal)',
  showBOsByDefault = true,
  showCalcsByDefault = true,
  showDatasourcesByDefault = true,
  showDatasourceLayer = true,
  height = '100%',
  emptyMessage = 'No upstream or downstream nodes for this term.',
}) => {
  const [nodes, setNodes, onNodesChange] = useNodesState([]);
  const [edgesList, setEdgesList, onEdgesChange] = useEdgesState([]);
  const [showBOs, setShowBOs] = useState(showBOsByDefault);
  const [showCalcs, setShowCalcs] = useState(showCalcsByDefault);
  const [showDatasources, setShowDatasources] = useState(showDatasourcesByDefault);
  const [expandedDs, setExpandedDs] = useState<Record<string, boolean>>({});
  const [isFullscreen, setIsFullscreen] = useState(false);
  const [drawer, setDrawer] = useState<DrawerState>({ open: false, type: 'node', data: null });

  const toggleDsExpand = useCallback((dsName: string) => {
    setExpandedDs((prev) => ({ ...prev, [dsName]: !prev[dsName] }));
  }, []);

  const expandAllDs = useCallback(() => {
    const next: Record<string, boolean> = {};
    (datasources || []).forEach((ds) => { next[ds.name] = true; });
    setExpandedDs(next);
  }, [datasources]);

  const collapseAllDs = useCallback(() => {
    setExpandedDs({});
  }, []);

  const handleNavigate = useCallback((id: string) => {
    if (onNavigate) onNavigate(id);
  }, [onNavigate]);

  // Build graph
  useEffect(() => {
    if (!focalTerm) {
      setNodes([]);
      setEdgesList([]);
      return;
    }

    const flowNodes: any[] = [];
    const flowEdges: any[] = [];

    const centerX = 500;
    const centerY = 280;

    // 1. Center focal node
    const props = typeof focalTerm.properties === 'string'
      ? (() => { try { return JSON.parse(focalTerm.properties); } catch { return {}; } })()
      : (focalTerm.properties || {});

    flowNodes.push({
      id: focalTerm.id,
      type: 'focalTerm',
      position: { x: centerX, y: centerY },
      data: {
        label: focalTerm.node_name,
        description: focalTerm.description,
        path: focalTerm.qualified_path,
        formula: props.formula || props.expression,
        termType: props.term_type,
        focalLabel,
        isBO: focalLabel?.includes('Business Object'),
        onInspectNode: (data: any) => setDrawer({ open: true, type: 'node', data: { ...focalTerm, ...data, properties: props } }),
      },
    });

    // 2. Upstream nodes (left column)
    const filteredUpstream = (upstreamNodes || []).filter((bt: any) => {
      const cat = resolveNodeType(bt);
      const isBO = cat === 'business_object';
      const isCalc = cat === 'calculation';
      if (isBO && !showBOs) return false;
      if (isCalc && !showCalcs) return false;
      if (!isBO && !isCalc && !showBOs) return false;
      return true;
    });

    let currentUpstreamY = Math.max(30, centerY - 140);
    filteredUpstream.forEach((bt: any) => {
      const cat = resolveNodeType(bt);
      const isBO = cat === 'business_object';
      const isCalc = cat === 'calculation';
      const isBus = cat === 'business_term';
      const color = isBO ? '#A855F7' : isCalc ? '#38BDF8' : isBus ? C.teal : C.accent;
      const relLabel = bt.relLabel || (isBO ? 'member_of' : isCalc ? 'depends_on' : isBus ? 'HAS_BUSINESS_TERM' : 'describes');

      flowNodes.push({
        id: `up-${bt.id}`,
        type: 'upstream',
        position: { x: 70, y: currentUpstreamY },
        data: {
          rawId: bt.id,
          id: bt.id,
          label: bt.node_name || bt.label,
          description: bt.description,
          path: bt.qualified_path,
          formula: bt.formula || (bt.properties && bt.properties.formula),
          nodeType: bt.node_type || bt.catalog_type_name || 'business_term',
          catalogTypeName: bt.catalog_type_name || (isBO ? 'Business Object' : isCalc ? 'Calculated Term' : 'Business Term'),
          isCalc,
          onSelectNode: handleNavigate,
          onInspectNode: (data: any) => setDrawer({ open: true, type: 'node', data: { ...bt, ...data } }),
        },
      });

      flowEdges.push({
        id: `edge-up-${bt.id}-${focalTerm.id}`,
        source: `up-${bt.id}`,
        target: focalTerm.id,
        type: 'smoothstep',
        animated: true,
        label: relLabel,
        labelStyle: { fill: color, fontWeight: 700, fontSize: 10, cursor: 'pointer' },
        labelBgStyle: { fill: C.panel, fillOpacity: 0.95 },
        labelBgPadding: [4, 2],
        labelBgBorderRadius: 4,
        style: { stroke: color, strokeWidth: 2, cursor: 'pointer' },
        markerEnd: { type: MarkerType.ArrowClosed, color: color },
        data: {
          predicate: relLabel,
          source_name: bt.node_name || bt.label,
          target_name: focalTerm.node_name,
        },
      });

      currentUpstreamY += 150;
    });

    // 3. Downstream nodes (right column)
    let currentDownstreamY = Math.max(40, centerY - (downstreamNodes.length * 90));
    (downstreamNodes || []).forEach((dn: any) => {
      const nodeId = `dn-${dn.id}`;
      flowNodes.push({
        id: nodeId,
        type: 'upstream', // Reuse the upstream node renderer (works for any term)
        position: { x: 920, y: currentDownstreamY },
        data: {
          rawId: dn.id,
          id: dn.id,
          label: dn.node_name || dn.label,
          description: dn.description,
          path: dn.qualified_path,
          nodeType: dn.node_type || dn.catalog_type_name || 'semantic_term',
          catalogTypeName: dn.catalogTypeName || dn.catalog_type_name || 'Semantic Term',
          onSelectNode: handleNavigate,
          onInspectNode: (data: any) => setDrawer({ open: true, type: 'node', data: { ...dn, ...data } }),
        },
      });

      flowEdges.push({
        id: `edge-${focalTerm.id}-${nodeId}`,
        source: focalTerm.id,
        target: nodeId,
        type: 'smoothstep',
        animated: true,
        label: dn.relLabel || 'maps to',
        labelStyle: { fill: C.accent, fontWeight: 700, fontSize: 10, cursor: 'pointer' },
        labelBgStyle: { fill: C.panel, fillOpacity: 0.95 },
        labelBgPadding: [4, 2],
        labelBgBorderRadius: 4,
        style: { stroke: C.accent, strokeWidth: 2, cursor: 'pointer' },
        markerEnd: { type: MarkerType.ArrowClosed, color: C.accent },
        data: {
          predicate: dn.relLabel || 'maps to',
          source_name: focalTerm.node_name,
          target_name: dn.node_name || dn.label,
        },
      });

      currentDownstreamY += 150;
    });

    // 4. Datasources (right column, beyond downstream)
    if (showDatasourceLayer && showDatasources && datasources && datasources.length > 0) {
      let currentDsY = Math.max(40, centerY - (datasources.length * 100));
      datasources.forEach((ds) => {
        const isExp = !!expandedDs[ds.name];
        const dsNodeId = `ds-${ds.name}`;
        flowNodes.push({
          id: dsNodeId,
          type: 'datasource',
          position: { x: 1300, y: currentDsY },
          data: {
            datasourceName: ds.name,
            datasourceType: ds.type,
            datasourceHost: ds.host,
            totalColumns: ds.totalColumns,
            totalTables: ds.totalTables,
            schemas: ds.schemas,
            isExpanded: isExp,
            onToggleExpand: toggleDsExpand,
            onInspectNode: (data: any) => setDrawer({ open: true, type: 'node', data }),
          },
        });

        // Connect from focal to datasource (or from last downstream)
        const bridge = downstreamNodes.length > 0 ? `dn-${downstreamNodes[downstreamNodes.length - 1].id}` : focalTerm.id;
        flowEdges.push({
          id: `edge-${bridge}-${dsNodeId}`,
          source: bridge,
          target: dsNodeId,
          type: 'smoothstep',
          animated: true,
          label: 'realized_by',
          labelStyle: { fill: C.blue, fontWeight: 700, fontSize: 10, cursor: 'pointer' },
          labelBgStyle: { fill: C.panel, fillOpacity: 0.95 },
          labelBgPadding: [4, 2],
          labelBgBorderRadius: 4,
          style: { stroke: C.blue, strokeWidth: 1.5, cursor: 'pointer' },
          markerEnd: { type: MarkerType.ArrowClosed, color: C.blue },
          data: {
            predicate: 'realized_by',
            source_name: ds.name,
          },
        });

        currentDsY += 200;
      });
    }

    // 5. Custom edges — caller's `style` and `markerEnd` win if provided so
    // different edge types can be visually distinguished (e.g. realized_by edges
    // from semantic term → physical column).
    (edges || []).forEach((e: any) => {
      flowEdges.push({
        id: e.id || `edge-custom-${Math.random()}`,
        source: e.source,
        target: e.target,
        type: e.type || 'smoothstep',
        animated: e.animated !== false,
        label: e.label,
        labelStyle: e.labelStyle || { fill: C.text, fontWeight: 600, fontSize: 10, cursor: 'pointer' },
        labelBgStyle: e.labelBgStyle || { fill: C.panel, fillOpacity: 0.95 },
        labelBgPadding: e.labelBgPadding || [4, 2],
        labelBgBorderRadius: e.labelBgBorderRadius || 4,
        style: e.style || { stroke: C.borderStrong, strokeWidth: 1.5, cursor: 'pointer' },
        markerEnd: e.markerEnd || { type: MarkerType.ArrowClosed, color: C.borderStrong },
        data: e.data || e,
      });
    });

    setNodes(flowNodes);
    setEdgesList(flowEdges);
  }, [focalTerm, upstreamNodes, downstreamNodes, edges, datasources, showBOs, showCalcs, showDatasources, expandedDs, focalLabel, toggleDsExpand, handleNavigate]);

  const containerStyle: React.CSSProperties = {
    width: '100%',
    height: isFullscreen ? '100vh' : height,
    position: isFullscreen ? 'fixed' : 'relative',
    top: isFullscreen ? 0 : 'auto',
    left: isFullscreen ? 0 : 'auto',
    zIndex: isFullscreen ? 1000 : 'auto',
    background: C.bg,
    display: 'flex',
    flexDirection: 'column',
  };

  if (!focalTerm) {
    return (
      <div style={{ padding: 64, textAlign: 'center', color: C.textMuted, fontSize: 14 }}>
        Select a term to view its lineage.
      </div>
    );
  }

  if ((upstreamNodes || []).length === 0 && (downstreamNodes || []).length === 0 && (!datasources || datasources.length === 0)) {
    return (
      <div style={{ padding: 64, textAlign: 'center', color: C.textMuted, fontSize: 14 }}>
        {emptyMessage}
      </div>
    );
  }

  return (
    <div style={containerStyle}>
      <style>{`@keyframes slideIn { from { transform: translateX(100%); opacity: 0; } to { transform: translateX(0); opacity: 1; } }`}</style>
      {/* Toolbar */}
      <div style={{
        padding: '10px 20px',
        background: 'rgba(255,255,255,0.02)',
        borderBottom: `1px solid ${C.border}`,
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'space-between',
        gap: 16,
        flexWrap: 'wrap',
      }}>
        <div style={{ display: 'flex', alignItems: 'center', gap: 12, fontSize: 12, flexWrap: 'wrap' }}>
          <span style={{ fontSize: 11, fontWeight: 700, color: C.textMuted, textTransform: 'uppercase', letterSpacing: '0.04em' }}>
            Layers:
          </span>
          <button
            onClick={() => setShowBOs(!showBOs)}
            style={{
              background: showBOs ? 'rgba(168,85,247,0.15)' : 'transparent',
              border: `1px solid ${showBOs ? '#A855F7' : C.border}`,
              color: showBOs ? '#C084FC' : C.textMuted,
              padding: '4px 10px',
              borderRadius: 6,
              fontSize: 12,
              fontWeight: 600,
              cursor: 'pointer',
              display: 'flex',
              alignItems: 'center',
              gap: 5,
              transition: 'all 0.15s ease',
            }}
          >
            <span>🏢</span> Business Objects ({showBOs ? 'ON' : 'OFF'})
          </button>
          <button
            onClick={() => setShowCalcs(!showCalcs)}
            style={{
              background: showCalcs ? 'rgba(56,189,248,0.15)' : 'transparent',
              border: `1px solid ${showCalcs ? '#38BDF8' : C.border}`,
              color: showCalcs ? '#38BDF8' : C.textMuted,
              padding: '4px 10px',
              borderRadius: 6,
              fontSize: 12,
              fontWeight: 600,
              cursor: 'pointer',
              display: 'flex',
              alignItems: 'center',
              gap: 5,
            }}
          >
            <span>🧮</span> Dependencies ({showCalcs ? 'ON' : 'OFF'})
          </button>
          {showDatasourceLayer && (
            <button
              onClick={() => setShowDatasources(!showDatasources)}
              style={{
                background: showDatasources ? 'rgba(96,165,250,0.15)' : 'transparent',
                border: `1px solid ${showDatasources ? C.blue : C.border}`,
                color: showDatasources ? C.blue : C.textMuted,
                padding: '4px 10px',
                borderRadius: 6,
                fontSize: 12,
                fontWeight: 600,
                cursor: 'pointer',
                display: 'flex',
                alignItems: 'center',
                gap: 5,
              }}
            >
              <span>🗄️</span> Datasources ({showDatasources ? 'ON' : 'OFF'})
            </button>
          )}
        </div>
        <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
          {showDatasourceLayer && datasources && datasources.length > 0 && showDatasources && (
            <>
              <button
                onClick={expandAllDs}
                style={{
                  background: C.accentDim,
                  border: `1px solid ${C.accent}`,
                  color: C.accent,
                  padding: '4px 10px',
                  borderRadius: 6,
                  fontSize: 12,
                  fontWeight: 600,
                  cursor: 'pointer',
                }}
              >
                ➕ Expand All Tables
              </button>
              <button
                onClick={collapseAllDs}
                style={{
                  background: 'transparent',
                  border: `1px solid ${C.border}`,
                  color: C.textMuted,
                  padding: '4px 10px',
                  borderRadius: 6,
                  fontSize: 12,
                  cursor: 'pointer',
                }}
              >
                ➖ Collapse All
              </button>
            </>
          )}
          <button
            onClick={() => setIsFullscreen(!isFullscreen)}
            style={{
              background: isFullscreen ? C.accent : 'transparent',
              border: `1px solid ${isFullscreen ? C.accent : C.border}`,
              color: isFullscreen ? '#fff' : C.text,
              padding: '4px 10px',
              borderRadius: 6,
              fontSize: 12,
              fontWeight: 600,
              cursor: 'pointer',
              display: 'flex',
              alignItems: 'center',
              gap: 4,
              transition: 'all 0.15s ease',
            }}
          >
            {isFullscreen ? '⤓ Exit Fullscreen' : '⤢ Fullscreen'}
          </button>
        </div>
      </div>
      {/* Canvas */}
      <div style={{ flex: 1, position: 'relative', overflow: 'hidden' }}>
        <ReactFlow
          nodes={nodes}
          edges={edgesList}
          onNodesChange={onNodesChange}
          onEdgesChange={onEdgesChange}
          nodeTypes={nodeTypes}
          fitView
          attributionPosition="bottom-right"
          onNodeClick={(_event, node) => {
            setDrawer({
              open: true,
              type: 'node',
              data: node.data || node,
            });
          }}
          onEdgeClick={(_event, edge) => {
            setDrawer({
              open: true,
              type: 'edge',
              data: edge.data || {
                predicate: edge.label,
                id: edge.id,
                source: edge.source,
                target: edge.target,
              },
            });
          }}
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
        <InspectorDrawer
          drawer={drawer}
          onClose={() => setDrawer({ open: false, type: 'node', data: null })}
          onNavigate={onNavigate}
        />
      </div>
    </div>
  );
};

export default LineageGraph;
