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
import { useAccess } from '../../contexts/AccessContext';
import { readCachedSelection } from '../../utils/tenantScope';
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
import { CoreIcon, CustomIcon } from '../../components/common/CoreCustomIcons';

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
// Lineage Custom Node Types
// ─────────────────────────────────────────────

// 1. Business Term Node (Upstream)
// 1. Upstream Node (Business Object, Business Term, or Dependent Calculation)
const BusinessTermNode: React.FC<{ data: any }> = ({ data }) => {
  const typeName = (data.nodeType || data.catalogTypeName || 'business_term').toLowerCase();
  const isBO = typeName.includes('business_object') || typeName.includes('bo');
  const isCalc = typeName.includes('calculated') || (data.path || '').includes('Investment ORM') || (data.path || '').includes('Northwind') || !!data.isCalc;
  const isTable = typeName.includes('table');
  const icon = isBO ? '🏢' : isCalc ? '🧮' : isTable ? '📊' : '💼';
  const labelText = isBO ? 'Business Object' : isCalc ? 'Calculation Dependency' : isTable ? 'Database Table' : (data.catalogTypeName || 'Business Term');
  const color = isBO ? '#A855F7' : isCalc ? '#38BDF8' : isTable ? C.blue : C.teal;

  return (
    <div 
      style={{
        background: C.panel,
        border: `1px solid ${color}66`,
        borderRadius: 10,
        minWidth: 240,
        maxWidth: 300,
        overflow: 'hidden',
        boxShadow: `0 4px 20px rgba(0,0,0,0.4)`,
        transition: 'all 0.15s ease',
      }}
      onMouseEnter={e => (e.currentTarget.style.borderColor = color, e.currentTarget.style.transform = 'translateY(-2px)')}
      onMouseLeave={e => (e.currentTarget.style.borderColor = `${color}66`, e.currentTarget.style.transform = 'translateY(0)')}
    >
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
          {/* Info Icon Button for Node Details Drawer */}
          <button
            className="nodrag nopan"
            title="Inspect Node Details"
            onClick={(e) => {
              e.stopPropagation();
              if (data.onInspectNode) {
                data.onInspectNode(data);
              }
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
            onMouseEnter={e => (e.currentTarget.style.background = `${color}44`)}
            onMouseLeave={e => (e.currentTarget.style.background = 'rgba(255,255,255,0.08)')}
          >
            ℹ️
          </button>
          
          {/* Focus Button */}
          <button
            className="nodrag nopan"
            title="Make this node the focal point"
            onClick={(e) => {
              e.stopPropagation();
              if (data.onSelectNode) {
                data.onSelectNode(data.rawId || data.id);
              }
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
        onClick={() => data.onSelectNode && data.onSelectNode(data.rawId || data.id)}
      >
        <div style={{ color: C.text, fontWeight: 700, fontSize: 13, wordBreak: 'break-word' }}>
          {data.label}
        </div>
        {data.description && (
          <div style={{ color: C.textMuted, fontSize: 11, marginTop: 4, lineHeight: 1.4, maxHeight: 44, overflow: 'hidden', textOverflow: 'ellipsis' }}>
            {data.description}
          </div>
        )}
        {data.formula && (
          <div style={{ marginTop: 6, padding: '3px 6px', background: 'rgba(56,189,248,0.1)', borderRadius: 4, border: '1px solid rgba(56,189,248,0.2)', fontSize: 10, fontFamily: 'monospace', color: '#38BDF8' }}>
            {data.formula}
          </div>
        )}
        {data.path && (
          <div style={{ color: C.textMuted, fontSize: 10, fontFamily: 'monospace', marginTop: 6, background: 'rgba(255,255,255,0.03)', padding: '2px 4px', borderRadius: 4, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
            {data.path}
          </div>
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

// 2. Semantic / Focal Term Node (Center Focus)
const SemanticTermNode: React.FC<{ data: any }> = ({ data }) => {
  const isCalc = !!data.formula || data.termType === 'calculated' || data.termType === 'preaggregated';
  const isBO = data.isBO || data.catalogTypeName === 'business_object' || (data.path || '').startsWith('business_object/') || (data.path || '').startsWith('/business_object/');
  const color = isBO ? '#A855F7' : isCalc ? '#38BDF8' : C.accent;
  const icon = isBO ? '🏢' : isCalc ? '🧮' : '🧠';
  const title = isBO ? 'Business Object (Focal)' : isCalc ? 'Calculated Term (Focal)' : 'Semantic Term (Focal)';

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
          {/* Info Icon Button */}
          <button
            className="nodrag nopan"
            title="Inspect Details"
            onClick={(e) => {
              e.stopPropagation();
              if (data.onInspectNode) {
                data.onInspectNode(data);
              }
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
          <Badge label={isBO ? 'Active BO' : data.mapped ? 'Mapped' : 'Unmapped'} color={isBO ? '#A855F7' : data.mapped ? C.success : C.textMuted} />
        </div>
      </div>
      <div style={{ padding: '12px 14px' }}>
        <div style={{ color: '#fff', fontWeight: 800, fontSize: 15, wordBreak: 'break-word' }}>
          {data.label}
        </div>
        {data.description && (
          <div style={{ color: C.textMuted, fontSize: 12, marginTop: 6, lineHeight: 1.4 }}>
            {data.description}
          </div>
        )}
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

// 3. Expandable Datasource Node (Single parent-child container for Datasource -> Schema -> Table -> Column)
const DatasourceNode: React.FC<{ data: any }> = ({ data }) => {
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
      {/* Header */}
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
          {/* Info Icon Button for Datasource Details */}
          <button
            className="nodrag nopan"
            title="Inspect Datasource Details"
            onClick={(e) => {
              e.stopPropagation();
              if (data.onInspectNode) {
                data.onInspectNode(data);
              }
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
              if (data.onToggleExpand) {
                data.onToggleExpand(data.datasourceName);
              }
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

      {/* Expanded Body: Schema/Resource -> Table/Endpoint -> Column/Field Hierarchy */}
      {isExpanded ? (
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
                {Object.entries(schema.tables || {}).map(([tableName, table]: [string, any]) => {
                  const isGet = tableName.toUpperCase().startsWith('GET');
                  const isPost = tableName.toUpperCase().startsWith('POST');
                  const isPut = tableName.toUpperCase().startsWith('PUT') || tableName.toUpperCase().startsWith('PATCH');
                  const isDelete = tableName.toUpperCase().startsWith('DELETE');
                  const methodColor = isGet ? '#10B981' : isPost ? '#F59E0B' : isPut ? '#6366F1' : isDelete ? '#EF4444' : C.blue;
                  const isHttpEndpoint = isGet || isPost || isPut || isDelete;

                  return (
                  <div key={tableName} style={{ background: 'rgba(255,255,255,0.02)', borderRadius: 4, padding: '6px 8px' }}>
                    <div style={{ display: 'flex', alignItems: 'center', gap: 6, marginBottom: 4 }}>
                      <span style={{ fontSize: 12 }}>{isHttpEndpoint ? '⚡' : '📊'}</span>
                      <span style={{ fontSize: 11, fontWeight: 700, color: methodColor, fontFamily: 'monospace' }}>
                        {tableName}
                      </span>
                      {isHttpEndpoint && (
                        <span style={{
                          fontSize: 9,
                          fontWeight: 700,
                          padding: '1px 4px',
                          borderRadius: 3,
                          background: `${methodColor}22`,
                          color: methodColor,
                          border: `1px solid ${methodColor}44`,
                        }}>
                          {isGet ? 'GET' : isPost ? 'POST' : isPut ? 'PUT' : 'DELETE'}
                        </span>
                      )}
                    </div>

                    <div style={{ display: 'flex', flexDirection: 'column', gap: 3, paddingLeft: 10 }}>
                      {table.columns.map((col: any, colIdx: number) => (
                        <div
                          key={`${schemaName}.${tableName}.${col.name}.${colIdx}`}
                          style={{
                            display: 'flex',
                            alignItems: 'center',
                            justifyContent: 'space-between',
                            gap: 6,
                            padding: '3px 6px',
                            background: 'rgba(0,0,0,0.15)',
                            borderRadius: 4,
                            fontSize: 11,
                          }}
                        >
                          <div style={{ display: 'flex', alignItems: 'center', gap: 5, overflow: 'hidden' }}>
                            <span style={{ color: C.accent, fontSize: 10 }}>🔹</span>
                            <span style={{ fontFamily: 'monospace', fontWeight: 600, color: C.text, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
                              {col.name}
                            </span>
                          </div>
                          <div style={{ display: 'flex', alignItems: 'center', gap: 4, flexShrink: 0 }}>
                            {col.dataType && (
                              <Badge label={col.dataType} color={typeColor(col.dataType)} />
                            )}
                            <Badge
                              label={col.isCore ? 'CORE' : 'CUSTOM'}
                              color={col.isCore ? C.teal : '#ED6C02'}
                            />
                            {col.edgeId && data.onRemoveMapping && (
                              <button
                                className="nodrag nopan"
                                title="Remove column mapping"
                                onClick={(e) => {
                                  e.stopPropagation();
                                  data.onRemoveMapping(col.edgeId);
                                }}
                                style={{
                                  background: 'transparent',
                                  border: 'none',
                                  color: C.danger,
                                  cursor: 'pointer',
                                  padding: '2px 4px',
                                  fontSize: 10,
                                  borderRadius: 3,
                                  display: 'flex',
                                  alignItems: 'center',
                                }}
                              >
                                ✕
                              </button>
                            )}
                          </div>
                        </div>
                      ))}
                    </div>
                  </div>
                );
                })}
              </div>
            </div>
          ))}
        </div>
      ) : (
        <div style={{ padding: '8px 12px', color: C.textMuted, fontSize: 11, fontStyle: 'italic' }}>
          Click Expand to view schema, tables &amp; columns
        </div>
      )}
    </div>
  );
};

// 4. Grouped Business Objects Node (Single expandable card for all connected Business Objects)
const GroupedBusinessObjectsNode: React.FC<{ data: any }> = ({ data }) => {
  const isExpanded = !!data.isExpanded;
  const items: any[] = data.items || [];
  const color = '#A855F7';

  return (
    <div style={{
      background: C.panel,
      border: `1px solid ${isExpanded ? color : 'rgba(168,85,247,0.4)'}`,
      borderRadius: 10,
      minWidth: isExpanded ? 300 : 240,
      maxWidth: 360,
      overflow: 'hidden',
      boxShadow: isExpanded ? '0 0 20px rgba(168,85,247,0.25)' : '0 4px 20px rgba(0,0,0,0.4)',
      transition: 'all 0.2s ease',
    }}>
      {/* Header */}
      <div style={{
        background: isExpanded ? 'rgba(168,85,247,0.18)' : 'rgba(168,85,247,0.08)',
        padding: '10px 12px',
        borderBottom: `1px solid ${C.border}`,
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'space-between',
        gap: 8,
      }}>
        <div style={{ display: 'flex', alignItems: 'center', gap: 8, overflow: 'hidden' }}>
          <span style={{ fontSize: 18 }}>🏢</span>
          <div>
            <div style={{ color: color, fontWeight: 700, fontSize: 12, letterSpacing: '0.04em', textTransform: 'uppercase' }}>
              Business Objects
            </div>
            <div style={{ color: C.textMuted, fontSize: 10, marginTop: 1 }}>
              {items.length} Connected Object{items.length !== 1 ? 's' : ''}
            </div>
          </div>
        </div>
        <button
          className="nodrag nopan"
          onClick={(e) => {
            e.stopPropagation();
            if (data.onToggleExpand) {
              data.onToggleExpand();
            }
          }}
          style={{
            background: isExpanded ? 'rgba(255,255,255,0.1)' : 'rgba(168,85,247,0.2)',
            border: `1px solid ${isExpanded ? C.border : color}`,
            color: isExpanded ? C.text : '#C084FC',
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

      {/* Expanded List of Business Objects */}
      {isExpanded ? (
        <div className="nodrag nopan" style={{ padding: '8px 10px', maxHeight: 300, overflowY: 'auto', display: 'flex', flexDirection: 'column', gap: 6 }}>
          {items.map((bo: any) => (
            <div
              key={bo.id}
              onClick={() => data.onSelectNode && data.onSelectNode(bo.id)}
              style={{
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'space-between',
                gap: 8,
                padding: '6px 10px',
                background: 'rgba(255,255,255,0.03)',
                borderRadius: 6,
                border: '1px solid rgba(255,255,255,0.05)',
                cursor: 'pointer',
                transition: 'all 0.15s ease',
              }}
              onMouseEnter={e => (e.currentTarget.style.background = 'rgba(168,85,247,0.15)', e.currentTarget.style.borderColor = color)}
              onMouseLeave={e => (e.currentTarget.style.background = 'rgba(255,255,255,0.03)', e.currentTarget.style.borderColor = 'rgba(255,255,255,0.05)')}
              title="Click to make this Business Object the focal point"
            >
              <div style={{ display: 'flex', alignItems: 'center', gap: 6, overflow: 'hidden' }}>
                <span style={{ fontSize: 13 }}>🏢</span>
                <span style={{ color: C.text, fontWeight: 600, fontSize: 12, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
                  {bo.node_name || bo.name}
                </span>
              </div>
              <span style={{ fontSize: 10, color: '#C084FC', fontWeight: 600, flexShrink: 0 }}>
                Focus ➔
              </span>
            </div>
          ))}
        </div>
      ) : (
        <div style={{ padding: '8px 12px', color: C.textMuted, fontSize: 11, fontStyle: 'italic' }}>
          {items.slice(0, 3).map((b: any) => b.node_name || b.name).join(', ')}{items.length > 3 ? ` +${items.length - 3} more` : ''}
        </div>
      )}

      <Handle
        type="source"
        position={Position.Right}
        style={{ background: color, width: 8, height: 8, border: `2px solid ${C.panel}` }}
      />
    </div>
  );
};

// 5. Generic Connected Node Fallback
const GenericNode: React.FC<{ data: any }> = ({ data }) => (
  <div style={{
    background: C.panel,
    border: `1px solid ${C.border}`,
    borderRadius: 8,
    minWidth: 180,
    overflow: 'hidden',
    boxShadow: '0 4px 16px rgba(0,0,0,0.3)',
  }}>
    <Handle type="target" position={Position.Left} style={{ background: C.textMuted, width: 7, height: 7 }} />
    <div style={{ padding: '8px 12px' }}>
      <div style={{ color: C.text, fontWeight: 700, fontSize: 12 }}>
        {data.icon || '📄'} {data.label}
      </div>
      {data.subtitle && <div style={{ color: C.textMuted, fontSize: 10, marginTop: 2 }}>{data.subtitle}</div>}
    </div>
    <Handle type="source" position={Position.Right} style={{ background: C.textMuted, width: 7, height: 7 }} />
  </div>
);

const nodeTypes = {
  businessTerm: BusinessTermNode,
  groupedBO: GroupedBusinessObjectsNode,
  semanticTerm: SemanticTermNode,
  datasource: DatasourceNode,
  generic: GenericNode,
  erdTable: GenericNode,
};

const parsePath = (path: string) => {
  if (!path) return { datasource: 'N/A', schema: 'N/A', table: 'N/A', column: 'N/A' };
  
  // Clean string
  const clean = path.trim().replace(/^\/+|\/+$/g, '');

  // Special handling for api_endpoint paths (e.g. "api_endpoint/Action Type" or "/api_endpoint/action_type")
  if (clean.startsWith('api_endpoint/') || clean === 'api_endpoint') {
    const parts = clean.split('/').filter(Boolean);
    return {
      datasource: 'API Endpoints',
      schema: 'REST API',
      table: 'Endpoints',
      column: parts.length > 1 ? parts[1] : 'Endpoint'
    };
  }

  const slashParts = clean.split('/').filter(Boolean);
  if (slashParts.length >= 4) {
    return { datasource: slashParts[0], schema: slashParts[1], table: slashParts[2], column: slashParts[3] };
  } else if (slashParts.length === 3) {
    return { datasource: slashParts[0], schema: slashParts[1], table: slashParts[1], column: slashParts[2] };
  } else if (slashParts.length === 2) {
    return { datasource: slashParts[0], schema: 'public', table: slashParts[0], column: slashParts[1] };
  }

  // Fallback to dot notation
  const dotParts = clean.split('.').filter(Boolean);
  if (dotParts.length === 4) {
    return { datasource: dotParts[0], schema: dotParts[1], table: dotParts[2], column: dotParts[3] };
  } else if (dotParts.length === 3) {
    return { datasource: dotParts[0], schema: 'public', table: dotParts[1], column: dotParts[2] };
  } else if (dotParts.length === 2) {
    return { datasource: dotParts[0], schema: 'public', table: dotParts[0], column: dotParts[1] };
  }

  return { datasource: 'Default Datasource', schema: 'public', table: 'table', column: path };
};

export default function GlossaryExplorer() {
  const { currentTenant, isPlatformOperator, accessLevel } = useAccess();
  const cachedSelection = readCachedSelection();
  const tenantId = currentTenant?.id ?? cachedSelection.tenant?.id;

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
  const tabParam = searchParams.get('tab') as 'properties' | 'technical' | 'relationships' | 'lineage' | null;

  const [searchTerm, setSearchTerm] = useState('');
  const [activeTab, setActiveTab] = useState<'properties' | 'technical' | 'relationships' | 'lineage'>(
    tabParam && ['properties', 'technical', 'relationships', 'lineage'].includes(tabParam) ? tabParam : 'properties'
  );

  useEffect(() => {
    if (tabParam && ['properties', 'technical', 'relationships', 'lineage'].includes(tabParam)) {
      setActiveTab(tabParam);
    }
  }, [tabParam]);
  const [isSemOpen, setIsSemOpen] = useState(true);

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
  const { data: semTermsRaw, loading: semLoading, refetch: refetchSem } = useApiQuery<any[]>(
    `/api/glossary/semantic-terms?tenant_id=${tenantId}`,
    { skip: !tenantId }
  );

  const { data: busTermsRaw, refetch: refetchBus } = useApiQuery<any[]>(
    `/api/glossary/business-terms?tenant_id=${tenantId}`,
    { skip: !tenantId }
  );

  const { data: nodeGraph, loading: graphLoading, refetch: refetchGraph } = useApiQuery<any>(
    `/api/glossary/node-graph?node_id=${selectedId}&tenant_id=${tenantId}`,
    { skip: !(tenantId && selectedId) }
  );

  // Technical assets — dedicated endpoint with proper edge direction
  const { data: technicalAssetsRaw, loading: taLoading, refetch: refetchTa } = useApiQuery<any>(
    `/api/glossary/technical-assets?node_id=${selectedId}&tenant_id=${tenantId}`,
    { skip: !(tenantId && selectedId) }
  );

  // Technical assets UI state
  const [taSearch, setTaSearch] = useState('');
  const [taDsFilter, setTaDsFilter] = useState<string>('ALL');
  const [isAddMappingOpen, setIsAddMappingOpen] = useState(false);
  const [mappingColSearch, setMappingColSearch] = useState('');
  const [selectedColIds, setSelectedColIds] = useState<string[]>([]);

  // Lineage Layer Visibility Toggles
  const [showBOs, setShowBOs] = useState(true);
  const [showCalcs, setShowCalcs] = useState(true);
  const [showDatasources, setShowDatasources] = useState(true);
  const [isBoGroupExpanded, setIsBoGroupExpanded] = useState(false);

  // Lineage Inspector Drawer (for Node Details & Edge Attributes)
  const [inspectDrawer, setInspectDrawer] = useState<{
    open: boolean;
    type: 'node' | 'edge';
    data: any;
  }>({
    open: false,
    type: 'node',
    data: null,
  });

  const { data: allColumns, loading: columnsLoading } = useApiQuery<any[]>(
    `/api/rest/catalog-nodes?node_type_id=a64c1011-16e8-4ddf-b447-363bf8e15c9a&tenant_id=${tenantId}&limit=10000`,
    { skip: !tenantId || !(isGenModalOpen || isAddMappingOpen) }
  );

  const semTerms = useMemo(() => {
    const list = Array.isArray(semTermsRaw) ? semTermsRaw : (semTermsRaw as any)?.data ?? [];
    const sorted = [...list].sort((a, b) => (a.node_name || '').localeCompare(b.node_name || ''));
    if (!searchTerm) return sorted;
    return sorted.filter((t: any) => t.node_name?.toLowerCase().includes(searchTerm.toLowerCase()));
  }, [semTermsRaw, searchTerm]);

  // Dedicated single node query to guarantee selected node data loads even on deep-link / direct URL navigation
  const { data: singleNodeRaw, refetch: refetchSingleNode } = useApiQuery<any>(
    `/api/rest/catalog-nodes/${selectedId}?tenant_id=${tenantId}`,
    { skip: !(tenantId && selectedId) }
  );

  const selectedTerm = useMemo(() => {
    if (!selectedId) return null;
    const semList = Array.isArray(semTermsRaw) ? semTermsRaw : (semTermsRaw as any)?.data ?? [];
    const busList = Array.isArray(busTermsRaw) ? busTermsRaw : (busTermsRaw as any)?.data ?? [];
    let found = semList.find((t: any) => t.id === selectedId);
    if (found) return { ...found, _kind: 'semantic' };
    found = busList.find((t: any) => t.id === selectedId);
    if (found) return { ...found, _kind: 'business' };

    // Fallback to direct single node lookup or nodeGraph
    if (singleNodeRaw && (singleNodeRaw.id === selectedId || singleNodeRaw.node_id === selectedId)) {
      const typeName = (singleNodeRaw.catalog_type_name || singleNodeRaw.type || '').toLowerCase();
      const isBus = typeName.includes('business');
      return { ...singleNodeRaw, _kind: isBus ? 'business' : 'semantic' };
    }
    if (nodeGraph?.node && (nodeGraph.node.id === selectedId || nodeGraph.node.node_id === selectedId)) {
      const typeName = (nodeGraph.node.catalog_type_name || nodeGraph.node.type || '').toLowerCase();
      const isBus = typeName.includes('business');
      return { ...nodeGraph.node, _kind: isBus ? 'business' : 'semantic' };
    }

    return null;
  }, [selectedId, semTermsRaw, busTermsRaw, singleNodeRaw, nodeGraph]);

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
      const res = await apiClient<any>(`/api/glossary/terms?tenant_id=${tenantId}`, {
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
    const raw = Array.isArray(technicalAssetsRaw) 
      ? technicalAssetsRaw 
      : ((technicalAssetsRaw as any)?.data ?? []);
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
    const raw = Array.isArray(technicalAssetsRaw)
      ? technicalAssetsRaw
      : ((technicalAssetsRaw as any)?.data ?? []);
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

  const refreshAllData = useCallback(async () => {
    await Promise.all([
      refetchTa(),
      refetchGraph(),
      refetchSem(),
      refetchBus(),
    ]);
  }, [refetchTa, refetchGraph, refetchSem, refetchBus]);

  // Remove a technical asset mapping (delete has_context edge)
  const removeMapping = async (edgeId: string) => {
    if (!window.confirm('Remove this column mapping from the semantic term?')) return;
    try {
      await apiClient(`/api/glossary/technical-assets/${edgeId}?tenant_id=${tenantId}`, { method: 'DELETE' });
      await refreshAllData();
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
      await refreshAllData();
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

  // Lineage nodes/edges & expand state
  const [nodes, setNodes, onNodesChange] = useNodesState([]);
  const [edges, setEdges, onEdgesChange] = useEdgesState([]);
  const [expandedDs, setExpandedDs] = useState<Record<string, boolean>>({});

  const toggleDsExpand = useCallback((dsName: string) => {
    setExpandedDs(prev => ({
      ...prev,
      [dsName]: !prev[dsName]
    }));
  }, []);

  // Compute datasource hierarchy from technicalAssetsRaw + nodeGraph physical assets
  const dsHierarchy = useMemo(() => {
    const raw = Array.isArray(technicalAssetsRaw) 
      ? technicalAssetsRaw 
      : ((technicalAssetsRaw as any)?.data ?? []);

    const datasources: Record<string, {
      name: string;
      type?: string;
      host?: string;
      totalColumns: number;
      tablesSet: Set<string>;
      schemas: Record<string, {
        name: string;
        tables: Record<string, {
          name: string;
          columns: Array<{
            id: string;
            name: string;
            dataType?: string;
            isCore?: boolean;
            qualifiedPath?: string;
            edgeId?: string;
          }>;
        }>;
      }>;
    }> = {};

    // Helper to add a physical asset to dsHierarchy
    const addAssetToHierarchy = (a: any) => {
      const qPath = a.qualified_path || a.path || '';
      const parsed = parsePath(qPath);
      
      let dsName = a.datasource || a.datasource_id || (parsed.datasource !== 'N/A' ? parsed.datasource : '');
      if (!dsName || dsName === 'default_datasource' || dsName === 'Default Datasource') {
        if (qPath.startsWith('/orm/')) dsName = 'CRM ORM Database';
        else if (qPath.startsWith('/public/')) dsName = 'Northwind Database';
        else if (qPath.startsWith('api_endpoint/') || qPath.startsWith('/api_endpoint/')) dsName = 'REST API';
        else dsName = 'Physical Datasource';
      }

      let schemaName = parsed.schema !== 'N/A' ? parsed.schema : 'public';
      let tableName = parsed.table !== 'N/A' ? parsed.table : (a.table_name || 'default_table');
      let colName = a.node_name || (parsed.column !== 'N/A' ? parsed.column : a.name || 'column');
      const dataType = (a.properties as any)?.data_type || (a.properties as any)?.dataType;

      if (!datasources[dsName]) {
        datasources[dsName] = {
          name: dsName,
          type: a.datasource_type || (dsName.includes('CRM') || dsName.includes('ORM') ? 'Postgres' : 'Postgres'),
          host: a.datasource_host,
          totalColumns: 0,
          tablesSet: new Set<string>(),
          schemas: {}
        };
      }

      const ds = datasources[dsName];
      ds.totalColumns += 1;
      ds.tablesSet.add(`${schemaName}.${tableName}`);

      if (!ds.schemas[schemaName]) {
        ds.schemas[schemaName] = {
          name: schemaName,
          tables: {}
        };
      }

      const schema = ds.schemas[schemaName];
      if (!schema.tables[tableName]) {
        schema.tables[tableName] = {
          name: tableName,
          columns: []
        };
      }

      schema.tables[tableName].columns.push({
        id: a.id || a.node_id,
        name: colName,
        dataType,
        isCore: a.is_core ?? true,
        qualifiedPath: qPath,
        edgeId: a.edge_id,
      });
    };

    raw.forEach(addAssetToHierarchy);

    // Fallback: Check nodeGraph for any physical column nodes connected via has_context
    if (raw.length === 0 && nodeGraph?.edges && nodeGraph.edges.length > 0) {
      const graphNodes = [
        ...(nodeGraph?.connected_nodes ?? []),
        ...(nodeGraph?.nodes ?? [])
      ];
      graphNodes.forEach((n: any) => {
        const nType = (n.catalog_type_name || n.catalog_type || n.node_type || '').toLowerCase();
        const nPath = (n.qualified_path || n.node_name || '');
        const isColumn = ['column', 'table', 'database_column', 'database_table', 'endpoint', 'api_endpoint'].includes(nType)
          || nPath.startsWith('/orm/')
          || nPath.startsWith('/public/')
          || nPath.startsWith('api_endpoint/');
        if (isColumn && n.id !== selectedTerm?.id) {
          addAssetToHierarchy(n);
        }
      });
    }

    return datasources;
  }, [technicalAssetsRaw, nodeGraph, selectedTerm]);

  // Compute upstream business terms connected to selected term
  const upstreamBusinessTerms = useMemo(() => {
    if (!selectedTerm) return [];
    const busList = Array.isArray(busTermsRaw) ? busTermsRaw : (busTermsRaw as any)?.data ?? [];
    const busMap = new Map<string, any>(busList.map((b: any) => [b.id, b]));
    const matchedTerms: any[] = [];
    const seenIds = new Set<string>();

    // 1. Direct parent_id relationship
    if (selectedTerm.parent_id && busMap.has(selectedTerm.parent_id)) {
      const p = busMap.get(selectedTerm.parent_id);
      matchedTerms.push({ ...p, relLabel: 'parent_term' });
      seenIds.add(p.id);
    }

    // 2. Check edges in nodeGraph
    const graphEdges = nodeGraph?.edges ?? [];
    graphEdges.forEach((e: any) => {
      const sourceId = e.source_node_id || e.source;
      const targetId = e.target_node_id || e.target;
      const otherId = sourceId === selectedTerm.id ? targetId : sourceId;
      let otherName = sourceId === selectedTerm.id ? e.target_name : e.source_name;
      let otherType = sourceId === selectedTerm.id ? e.target_node_type : e.source_node_type;
      const otherPath = sourceId === selectedTerm.id ? e.target_path : e.source_path;
      const rawNameOrPath = otherName || otherPath || '';

      // Skip database columns, api_endpoints, and physical assets from upstream column
      const lowerType = (otherType || '').toLowerCase();
      const isColumnOrPhysical = ['column', 'table', 'database_column', 'database_table', 'api_endpoint', 'endpoint'].includes(lowerType)
        || rawNameOrPath.startsWith('/orm/')
        || rawNameOrPath.startsWith('/public/')
        || rawNameOrPath.startsWith('api_endpoint/')
        || rawNameOrPath.startsWith('/api_endpoint/')
        || (rawNameOrPath.startsWith('/') && rawNameOrPath.split('/').filter(Boolean).length >= 3 && !rawNameOrPath.startsWith('/business_object'));

      if (isColumnOrPhysical) {
        return; // Columns, endpoints, and physical assets belong downstream in Technical Assets / Datasources
      }

      // Check for business object prefix or catalog type
      if (rawNameOrPath.startsWith('business_object/') || rawNameOrPath.startsWith('/business_object/') || lowerType.includes('business_object')) {
        otherType = 'business_object';
        if (otherName.includes('/')) {
          otherName = otherName.split('/').pop() || otherName;
        }
      }

      if (busMap.has(otherId) && !seenIds.has(otherId)) {
        const bTerm = busMap.get(otherId);
        const rel = e.relationship_type || e.edge_type_name || e.predicate || 'describes';
        matchedTerms.push({ ...bTerm, node_type: 'business_term', relLabel: rel });
        seenIds.add(otherId);
      } else if (otherId && !seenIds.has(otherId)) {
        const rel = e.relationship_type || e.edge_type_name || e.predicate || 'relates_to';
        matchedTerms.push({
          id: otherId,
          node_name: otherName?.includes('/') ? otherName.split('/').pop() : (otherName || 'Connected Entity'),
          qualified_path: otherPath,
          catalog_type_name: otherType || 'business_term',
          node_type: otherType || 'business_term',
          relLabel: rel,
        });
        seenIds.add(otherId);
      }
    });

    // 3. Fallback: Check connected_nodes in nodeGraph
    const graphNodes = [
      ...(nodeGraph?.connected_nodes ?? []),
      ...(nodeGraph?.nodes ?? [])
    ];
    graphNodes.forEach((n: any) => {
      let nType = (n.catalog_type_name || n.catalog_type || n.node_type || '').toLowerCase();
      const nPath = (n.qualified_path || n.node_name || '');
      const isColumnOrPhysical = ['column', 'table', 'database_column', 'database_table', 'api_endpoint', 'endpoint'].includes(nType)
        || nPath.startsWith('/orm/')
        || nPath.startsWith('/public/')
        || nPath.startsWith('api_endpoint/')
        || nPath.startsWith('/api_endpoint/')
        || (nPath.startsWith('/') && nPath.split('/').filter(Boolean).length >= 3 && !nPath.startsWith('/business_object'));

      if (n.id !== selectedTerm.id && !seenIds.has(n.id) && !isColumnOrPhysical) {
        if (nPath.startsWith('business_object/') || nPath.startsWith('/business_object/') || nType.includes('business_object')) {
          nType = 'business_object';
        }
        matchedTerms.push({
          ...n,
          node_name: n.node_name?.includes('/') ? n.node_name.split('/').pop() : n.node_name,
          node_type: nType || 'business_term',
          catalog_type_name: nType || 'business_term',
          relLabel: 'relates_to'
        });
        seenIds.add(n.id);
      }
    });

    return matchedTerms;
  }, [selectedTerm, busTermsRaw, nodeGraph]);

  const dsList = useMemo(() => Object.values(dsHierarchy), [dsHierarchy]);

  const expandAllDs = useCallback(() => {
    const next: Record<string, boolean> = {};
    dsList.forEach(ds => { next[ds.name] = true; });
    setExpandedDs(next);
  }, [dsList]);

  const collapseAllDs = useCallback(() => {
    setExpandedDs({});
  }, []);

  useEffect(() => {
    if (activeTab !== 'lineage' || !selectedTerm) return;
    const flowNodes: any[] = [];
    const flowEdges: any[] = [];
    
    const centerX = 500;
    const centerY = 280;

    const props = typeof selectedTerm.properties === 'string'
      ? (() => { try { return JSON.parse(selectedTerm.properties); } catch { return {}; } })()
      : (selectedTerm.properties || {});
    
    // 1. Add Center Focal Term Node
    flowNodes.push({
      id: selectedTerm.id,
      type: 'semanticTerm',
      position: { x: centerX, y: centerY },
      data: {
        label: selectedTerm.node_name,
        description: selectedTerm.description,
        path: selectedTerm.qualified_path,
        formula: props.formula || props.expression,
        termType: props.term_type,
        mapped: dsList.length > 0,
        selected: true,
        onInspectNode: (nodeData: any) => setInspectDrawer({ open: true, type: 'node', data: { ...selectedTerm, ...nodeData, properties: props } }),
      }
    });

    // Filter upstream terms based on active visibility toggles
    const filteredUpstream = upstreamBusinessTerms.filter((bt: any) => {
      const typeName = (bt.node_type || bt.catalog_type_name || '').toLowerCase();
      const isBO = typeName.includes('business_object') || typeName.includes('bo');
      const isCalc = typeName.includes('calculated') || (bt.qualified_path || '').includes('Investment ORM') || (bt.qualified_path || '').includes('Northwind') || !!bt.isCalc || bt.relLabel === 'depends_on';
      if (isBO && !showBOs) return false;
      if (isCalc && !showCalcs) return false;
      if (!isBO && !isCalc && !showBOs) return false;
      return true;
    });

    // Segregate upstream terms: Business Objects vs Calculations vs Standard Terms vs Semantic Terms
    const boList: any[] = [];
    const calcList: any[] = [];
    const standardList: any[] = [];

    const isFocalBO = (selectedTerm._kind === 'business') 
      || (selectedTerm.catalog_type_name || '').toLowerCase().includes('business')
      || (selectedTerm.qualified_path || '').startsWith('business_object/')
      || (selectedTerm.qualified_path || '').startsWith('/business_object/');

    filteredUpstream.forEach((bt: any) => {
      const typeName = (bt.node_type || bt.catalog_type_name || '').toLowerCase();
      const isBO = typeName.includes('business_object') || typeName.includes('bo');
      const isCalc = typeName.includes('calculated') || (bt.qualified_path || '').includes('Investment ORM') || (bt.qualified_path || '').includes('Northwind') || !!bt.isCalc || bt.relLabel === 'depends_on';

      if (isBO) {
        boList.push(bt);
      } else if (isCalc) {
        calcList.push(bt);
      } else {
        standardList.push(bt);
      }
    });

    // 2. Add Upstream Nodes (Left Column) with smart grouping and non-overlapping layout
    let currentUpstreamY = Math.max(30, centerY - 140);
    const upstreamNodeEntries: any[] = [];

    // A. Business Objects Group (only if focal node is NOT a Business Object itself)
    if (!isFocalBO && boList.length > 0) {
      if (boList.length > 1) {
        upstreamNodeEntries.push({
          type: 'groupedBO',
          id: 'grouped-business-objects',
          data: {
            items: boList,
            isExpanded: isBoGroupExpanded,
            onToggleExpand: () => setIsBoGroupExpanded(prev => !prev),
            onSelectNode: (newId: string) => setSearchParams({ id: newId, tab: 'lineage' }),
            onInspectNode: (nodeData: any) => setInspectDrawer({ open: true, type: 'node', data: nodeData }),
          },
          relLabel: `${boList.length} Business Objects`,
          edgeColor: '#A855F7',
          height: isBoGroupExpanded ? Math.min(360, 90 + boList.length * 36) : 90,
          edgeData: {
            predicate: 'member_of',
            relationship_type: 'member_of',
            source_type: 'business_object_group',
            target_type: 'semantic_term',
            source_name: `${boList.length} Business Objects`,
            target_name: selectedTerm.node_name,
            items: boList.map(b => b.node_name || b.name),
          }
        });
      } else {
        const bo = boList[0];
        upstreamNodeEntries.push({
          type: 'businessTerm',
          id: `bo-${bo.id}`,
          data: {
            rawId: bo.id,
            id: bo.id,
            label: bo.node_name || bo.name,
            description: bo.description,
            path: bo.qualified_path,
            nodeType: 'business_object',
            catalogTypeName: 'business_object',
            onSelectNode: (newId: string) => setSearchParams({ id: newId, tab: 'lineage' }),
            onInspectNode: (nodeData: any) => setInspectDrawer({ open: true, type: 'node', data: { ...bo, ...nodeData } }),
          },
          relLabel: 'member_of',
          edgeColor: '#A855F7',
          height: 140,
          edgeData: {
            predicate: 'member_of',
            relationship_type: 'member_of',
            source_type: 'business_object',
            target_type: 'semantic_term',
            source_name: bo.node_name || bo.name,
            target_name: selectedTerm.node_name,
          }
        });
      }
    } else if (isFocalBO && boList.length > 0) {
      // If focal node is a Business Object, any connected BOs are related BOs
      boList.forEach((bo: any) => {
        upstreamNodeEntries.push({
          type: 'businessTerm',
          id: `bo-${bo.id}`,
          data: {
            rawId: bo.id,
            id: bo.id,
            label: bo.node_name || bo.name,
            description: bo.description,
            path: bo.qualified_path,
            nodeType: 'business_object',
            catalogTypeName: 'Business Object',
            onSelectNode: (newId: string) => setSearchParams({ id: newId, tab: 'lineage' }),
            onInspectNode: (nodeData: any) => setInspectDrawer({ open: true, type: 'node', data: { ...bo, ...nodeData } }),
          },
          relLabel: 'relates_to',
          edgeColor: '#A855F7',
          height: 140,
          edgeData: {
            predicate: 'relates_to',
            relationship_type: 'relates_to',
            source_type: 'business_object',
            target_type: 'business_object',
            source_name: bo.node_name || bo.name,
            target_name: selectedTerm.node_name,
          }
        });
      });
    }

    // B. Calculations & Dependencies
    calcList.forEach((calc: any) => {
      upstreamNodeEntries.push({
        type: 'businessTerm',
        id: `calc-${calc.id}`,
        data: {
          rawId: calc.id,
          id: calc.id,
          label: calc.node_name || calc.name,
          description: calc.description,
          path: calc.qualified_path,
          formula: calc.formula || (calc.properties && calc.properties.formula),
          nodeType: 'calculated',
          catalogTypeName: 'Calculated Term',
          isCalc: true,
          onSelectNode: (newId: string) => setSearchParams({ id: newId, tab: 'lineage' }),
          onInspectNode: (nodeData: any) => setInspectDrawer({ open: true, type: 'node', data: { ...calc, ...nodeData } }),
        },
        relLabel: 'depends_on',
        edgeColor: '#38BDF8',
        height: 140,
        edgeData: {
          predicate: 'depends_on',
          relationship_type: 'depends_on',
          source_type: 'calculated_term',
          target_type: 'semantic_term',
          source_name: calc.node_name || calc.name,
          target_name: selectedTerm.node_name,
          formula: calc.formula || (calc.properties && calc.properties.formula),
        }
      });
    });

    // C. Standard Terms
    standardList.forEach((st: any) => {
      upstreamNodeEntries.push({
        type: 'businessTerm',
        id: `st-${st.id}`,
        data: {
          rawId: st.id,
          id: st.id,
          label: st.node_name || st.name,
          description: st.description,
          path: st.qualified_path,
          nodeType: st.node_type || st.catalog_type_name || 'business_term',
          catalogTypeName: st.catalog_type_name,
          onSelectNode: (newId: string) => setSearchParams({ id: newId, tab: 'lineage' }),
          onInspectNode: (nodeData: any) => setInspectDrawer({ open: true, type: 'node', data: { ...st, ...nodeData } }),
        },
        relLabel: st.relLabel || 'describes',
        edgeColor: C.teal,
        height: 130,
        edgeData: {
          predicate: st.relLabel || 'describes',
          relationship_type: st.relLabel || 'describes',
          source_type: 'business_term',
          target_type: 'semantic_term',
          source_name: st.node_name || st.name,
          target_name: selectedTerm.node_name,
        }
      });
    });

    // Render upstream nodes with dynamic non-overlapping vertical offsets
    upstreamNodeEntries.forEach((entry: any) => {
      flowNodes.push({
        id: entry.id,
        type: entry.type,
        position: { x: 70, y: currentUpstreamY },
        data: entry.data,
      });

      flowEdges.push({
        id: `edge-${entry.id}-${selectedTerm.id}`,
        source: entry.id,
        target: selectedTerm.id,
        type: 'smoothstep',
        animated: true,
        label: entry.relLabel,
        labelStyle: { fill: entry.edgeColor, fontWeight: 700, fontSize: 10, cursor: 'pointer' },
        labelBgStyle: { fill: C.panel, fillOpacity: 0.95 },
        labelBgPadding: [4, 2],
        labelBgBorderRadius: 4,
        style: { stroke: entry.edgeColor, strokeWidth: 2, cursor: 'pointer' },
        markerEnd: { type: MarkerType.ArrowClosed, color: entry.edgeColor },
        data: entry.edgeData || {
          predicate: entry.relLabel,
          source_id: entry.id,
          target_id: selectedTerm.id,
        }
      });

      currentUpstreamY += entry.height + 30;
    });

    // 3. Add Downstream Datasource Nodes (Right Column)
    if (showDatasources && dsList.length > 0) {
      let currentDsY = Math.max(40, centerY - (dsList.length * 90));

      dsList.forEach((ds) => {
        const isExp = !!expandedDs[ds.name];
        const dsNodeId = `ds-${ds.name}`;

        flowNodes.push({
          id: dsNodeId,
          type: 'datasource',
          position: { x: 920, y: currentDsY },
          data: {
            datasourceName: ds.name,
            datasourceType: ds.type,
            datasourceHost: ds.host,
            totalColumns: ds.totalColumns,
            totalTables: ds.tablesSet.size,
            schemas: ds.schemas,
            isExpanded: isExp,
            onToggleExpand: toggleDsExpand,
            onRemoveMapping: removeMapping,
            onInspectNode: (nodeData: any) => setInspectDrawer({ open: true, type: 'node', data: { ...ds, ...nodeData, node_name: ds.name, catalog_type_name: 'Datasource' } }),
          }
        });

        flowEdges.push({
          id: `edge-${selectedTerm.id}-${dsNodeId}`,
          source: selectedTerm.id,
          target: dsNodeId,
          type: 'smoothstep',
          animated: true,
          label: `${ds.totalColumns} col${ds.totalColumns !== 1 ? 's' : ''}`,
          labelStyle: { fill: C.blue, fontWeight: 700, fontSize: 10 },
          labelBgStyle: { fill: C.panel, fillOpacity: 0.95 },
          labelBgPadding: [4, 2],
          labelBgBorderRadius: 4,
          style: { stroke: C.blue, strokeWidth: 2 },
          markerEnd: { type: MarkerType.ArrowClosed, color: C.blue }
        });

        // Compute height for dynamic layout so expanded nodes don't overlap
        let estimatedHeight = 120;
        if (isExp) {
          let rows = 0;
          Object.values(ds.schemas).forEach(s => {
            rows += 1;
            Object.values(s.tables).forEach(t => {
              rows += 1 + t.columns.length;
            });
          });
          estimatedHeight = Math.min(440, 120 + rows * 28);
        }
        currentDsY += estimatedHeight + 40;
      });
    }

    // 4. Handle other connected graph nodes if present (excluding already processed business terms and columns)
    if (nodeGraph?.edges && nodeGraph.edges.length > 0) {
      const handledIds = new Set<string>([
        selectedTerm.id,
        ...filteredUpstream.map((b: any) => b.id),
        ...upstreamBusinessTerms.map((b: any) => b.id)
      ]);

      const otherNodes = (nodeGraph.connected_nodes ?? nodeGraph.nodes ?? []).filter((n: any) => {
        if (handledIds.has(n.id)) return false;
        const nType = (n.catalog_type_name || n.catalog_type || n.node_type || '').toLowerCase();
        const nPath = (n.qualified_path || n.node_name || '');
        const isTechnicalAsset = ['column', 'table', 'database_column', 'database_table', 'api_endpoint', 'endpoint'].includes(nType)
          || nPath.startsWith('/orm/')
          || nPath.startsWith('/public/')
          || nPath.startsWith('api_endpoint/')
          || nPath.startsWith('/api_endpoint/')
          || (nPath.startsWith('/') && nPath.split('/').filter(Boolean).length >= 3 && !nPath.startsWith('/business_object'));
        
        const isCalcOrDep = nType.includes('calculated') || nPath.includes('Investment ORM') || nPath.includes('Northwind') || (n.properties && n.properties.formula);
        if (isCalcOrDep && !showCalcs) return false;

        return !isTechnicalAsset;
      });

      otherNodes.forEach((n: any, idx: number) => {
        const nodeId = `other-${n.id}`;
        flowNodes.push({
          id: nodeId,
          type: 'generic',
          position: { x: centerX, y: centerY + 220 + idx * 110 },
          data: {
            id: n.id,
            label: n.node_name || n.id,
            subtitle: n.qualified_path || n.catalog_type_name || 'Related Node',
            onInspectNode: (nodeData: any) => setInspectDrawer({ open: true, type: 'node', data: { ...n, ...nodeData } }),
            onSelectNode: (newId: string) => setSearchParams({ id: newId, tab: 'lineage' }),
          }
        });

        flowEdges.push({
          id: `edge-${selectedTerm.id}-${nodeId}`,
          source: selectedTerm.id,
          target: nodeId,
          type: 'smoothstep',
          animated: false,
          label: 'related',
          style: { stroke: C.border, strokeWidth: 1.5, cursor: 'pointer' },
          markerEnd: { type: MarkerType.ArrowClosed, color: C.textMuted },
          data: {
            edge_type: 'related',
            source_id: selectedTerm.id,
            target_id: n.id,
            source_name: selectedTerm.node_name,
            target_name: n.node_name || n.id,
          }
        });
      });
    }

    setNodes(flowNodes);
    setEdges(flowEdges);
  }, [activeTab, selectedTerm, upstreamBusinessTerms, dsList, expandedDs, toggleDsExpand, nodeGraph, showBOs, showCalcs, showDatasources, isBoGroupExpanded]);

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
          <div style={{ background: C.panel, padding: 24, borderRadius: 12, width: 480, border: `1px solid ${C.border}`, maxHeight: '90vh', overflowY: 'auto' }}>
            <h2 style={{ margin: '0 0 16px 0', fontSize: 18, color: C.text }}>New Semantic Term</h2>
            <div style={{ marginBottom: 12 }}>
              <label style={{ fontSize: 11, color: C.textMuted, textTransform: 'uppercase', fontWeight: 700 }}>Term Name</label>
              <input placeholder="e.g. Net Investment Yield or Profit Margin" style={inputStyle} value={semName} onChange={e => setSemName(e.target.value)} />
            </div>
            <div style={{ marginBottom: 12 }}>
              <label style={{ fontSize: 11, color: C.textMuted, textTransform: 'uppercase', fontWeight: 700 }}>Description</label>
              <textarea placeholder="Purpose or definition of this metric" style={{ ...inputStyle, height: 70 }} value={semDesc} onChange={e => setSemDesc(e.target.value)} />
            </div>
            <div style={{ display: 'flex', gap: 12, justifyContent: 'flex-end', marginTop: 16 }}>
              <button onClick={() => setIsCreateSemModalOpen(false)} style={{ background: 'transparent', color: C.text, border: 'none', cursor: 'pointer' }}>Cancel</button>
              <button onClick={createSemantic} style={{ background: C.accent, color: '#0F172A', border: 'none', padding: '8px 18px', borderRadius: 6, cursor: 'pointer', fontWeight: 700 }}>Save Term</button>
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
          <div style={{ padding: '12px 16px' }}>
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
        </div>
      </div>

      {/* Main Panel */}
      <div style={{ flex: 1, display: 'flex', flexDirection: 'column', background: C.bg }}>
        
        {/* Topbar */}
        <div style={{ height: 60, borderBottom: `1px solid ${C.border}`, display: 'flex', alignItems: 'center', padding: '0 24px', justifyContent: 'space-between', background: C.panel }}>
          <div style={{ display: 'flex', alignItems: 'center', gap: 16 }}>
            <h1 style={{ margin: 0, fontSize: 17, fontWeight: 700, letterSpacing: '-0.01em', color: C.text }}>
              Semantic Terms Explorer
            </h1>
            <div style={{ display: 'flex', gap: 8 }}>
              <span style={{
                display: 'inline-flex', alignItems: 'center', padding: '3px 9px',
                borderRadius: 9999, fontSize: 11, fontWeight: 700, letterSpacing: '0.04em',
                color: C.accent, background: 'rgba(99,102,241,0.12)',
                border: `1px solid ${C.accent}44`, fontFamily: 'monospace', textTransform: 'uppercase',
              }}>
                {semTerms.length} Terms
              </span>
              <span style={{
                display: 'inline-flex', alignItems: 'center', padding: '3px 9px',
                borderRadius: 9999, fontSize: 11, fontWeight: 700, letterSpacing: '0.04em',
                color: C.success, background: 'rgba(16,185,129,0.12)',
                border: `1px solid ${C.success}44`, fontFamily: 'monospace', textTransform: 'uppercase',
              }}>
                {semTerms.filter((t: any) => t.is_active !== false).length} Active
              </span>
            </div>
          </div>
          {canCreate && (
            <div style={{ display: 'flex', gap: 10 }}>
              <button 
                onClick={() => setIsGenModalOpen(true)} 
                style={{ display: 'flex', alignItems: 'center', gap: 6, background: '#1E2130', border: `1px solid ${C.border}`, color: C.text, padding: '6px 12px', borderRadius: 6, cursor: 'pointer', fontSize: 13, fontWeight: 600 }}
              >
                <AutoFixIcon sx={{ fontSize: 16 }} /> Generate from Columns
              </button>
              <button 
                onClick={() => setIsCreateSemModalOpen(true)} 
                style={{ display: 'flex', alignItems: 'center', gap: 6, background: C.accent, color: '#0F172A', border: 'none', padding: '6px 14px', borderRadius: 6, cursor: 'pointer', fontSize: 13, fontWeight: 700, boxShadow: C.accentGlow }}
              >
                <AddIcon sx={{ fontSize: 16 }} /> Create Semantic Term
              </button>
            </div>
          )}
        </div>

        {/* Detail View */}
        {!selectedTerm ? (
          <Empty icon="📖" title="Select a term" subtitle="Choose a semantic term from the sidebar to view details." />
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
                      {(() => {
                        const props = typeof selectedTerm.properties === 'string' 
                          ? (() => { try { return JSON.parse(selectedTerm.properties); } catch { return {}; } })() 
                          : (selectedTerm.properties || {});
                        
                        const isCalc = props.term_type === 'calculated' || props.term_type === 'preaggregated' || !!props.formula;
                        if (!isCalc) return null;

                        return (
                          <>
                            <tr style={{ borderBottom: `1px solid ${C.border}`, background: 'rgba(99,102,241,0.03)' }}>
                              <td style={{ padding: '12px 0', color: C.accent, fontWeight: 700 }}>Calculation Classification</td>
                              <td style={{ padding: '12px 0' }}>
                                <Badge label={props.term_type === 'preaggregated' ? '⚡ Pre-Aggregated Metric' : '🧮 Calculated Field (Derived)'} color={C.accent} />
                              </td>
                            </tr>
                            {props.execution_strategy && (
                              <tr style={{ borderBottom: `1px solid ${C.border}` }}>
                                <td style={{ padding: '12px 0', color: C.textMuted }}>Execution Strategy</td>
                                <td style={{ padding: '12px 0' }}>
                                  <Badge 
                                    label={props.execution_strategy === 'sql' ? 'SQL (Deep CTE)' : props.execution_strategy === 'on_the_fly' ? 'On-The-Fly (In-Memory)' : 'Pre-Aggregated (Cube)'} 
                                    color={props.execution_strategy === 'sql' ? C.blue : props.execution_strategy === 'on_the_fly' ? '#10B981' : '#F59E0B'} 
                                  />
                                </td>
                              </tr>
                            )}
                            {props.formula && (
                              <tr style={{ borderBottom: `1px solid ${C.border}` }}>
                                <td style={{ padding: '12px 0', color: C.textMuted }}>Formula Expression</td>
                                <td style={{ padding: '12px 0' }}>
                                  <code style={{
                                    background: 'rgba(0,0,0,0.3)', padding: '4px 10px', borderRadius: 6,
                                    fontFamily: 'monospace', fontSize: 13, color: '#38BDF8', border: `1px solid ${C.border}`
                                  }}>
                                    {props.formula}
                                  </code>
                                </td>
                              </tr>
                            )}
                            {props.aggregation_function && (
                              <tr style={{ borderBottom: `1px solid ${C.border}` }}>
                                <td style={{ padding: '12px 0', color: C.textMuted }}>Default Aggregation</td>
                                <td style={{ padding: '12px 0', fontWeight: 600, color: C.teal }}>{props.aggregation_function}</td>
                              </tr>
                            )}
                            {props.drill_down_dimensions && (
                              <tr style={{ borderBottom: `1px solid ${C.border}` }}>
                                <td style={{ padding: '12px 0', color: C.textMuted }}>Drill-Down Dimensions</td>
                                <td style={{ padding: '12px 0', color: C.textMuted, fontSize: 13 }}>
                                  {props.drill_down_dimensions.split(',').map((dim: string, idx: number) => (
                                    <span key={idx} style={{ marginRight: 6, background: 'rgba(255,255,255,0.05)', padding: '2px 8px', borderRadius: 4, border: `1px solid ${C.border}` }}>
                                      🔍 {dim.trim()}
                                    </span>
                                  ))}
                                </td>
                              </tr>
                            )}
                          </>
                        );
                      })()}
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
                    <div style={{ background: C.panel, borderRadius: 10, border: `1px solid ${C.border}`, overflow: 'hidden', boxShadow: '0 4px 20px rgba(0,0,0,0.25)' }}>
                      <table style={{ width: '100%', borderCollapse: 'collapse', textAlign: 'left' }}>
                        <thead>
                          <tr style={{ borderBottom: `2px solid ${C.border}`, background: 'rgba(255,255,255,0.03)' }}>
                            <th style={{ padding: '12px 18px', fontSize: 11, color: C.textMuted, fontWeight: 700, textTransform: 'uppercase', letterSpacing: '0.06em' }}>Datasource</th>
                            <th style={{ padding: '12px 18px', fontSize: 11, color: C.textMuted, fontWeight: 700, textTransform: 'uppercase', letterSpacing: '0.06em' }}>Table · Schema</th>
                            <th style={{ padding: '12px 18px', fontSize: 11, color: C.textMuted, fontWeight: 700, textTransform: 'uppercase', letterSpacing: '0.06em' }}>Physical Column</th>
                            <th style={{ padding: '12px 18px', fontSize: 11, color: C.textMuted, fontWeight: 700, textTransform: 'uppercase', letterSpacing: '0.06em' }}>Data Type</th>
                            <th style={{ padding: '12px 18px', fontSize: 11, color: C.textMuted, fontWeight: 700, textTransform: 'uppercase', letterSpacing: '0.06em' }}>Source</th>
                            <th style={{ padding: '12px 18px', fontSize: 11, color: C.textMuted, fontWeight: 700, textTransform: 'uppercase', letterSpacing: '0.06em', textAlign: 'right' }}>Action</th>
                          </tr>
                        </thead>
                        <tbody>
                          {technicalAssets.map((a: any, i: number) => {
                            const parsed = parsePath(a.qualified_path);
                            const dsName = a.datasource || a.datasource_id || parsed.datasource || 'Default Datasource';
                            const dsType = (a.datasource_type || '').toLowerCase();
                            const isApi = dsType.includes('api') || dsName.toLowerCase().includes('api');
                            const dsIcon = isApi ? '🌐' : dsType.includes('snowflake') ? '❄️' : '🐘';
                            const tableLabel = [parsed.schema !== 'N/A' ? parsed.schema : null, parsed.table !== 'N/A' ? parsed.table : null].filter(Boolean).join(' · ');
                            const dataType = (a.properties as any)?.data_type || (a.properties as any)?.dataType;
                            const colName = a.node_name || parsed.column || a.name || 'column';

                            return (
                              <tr
                                key={a.edge_id ?? a.id ?? i}
                                style={{ borderBottom: `1px solid ${C.border}`, transition: 'background 0.15s' }}
                                onMouseEnter={e => (e.currentTarget.style.background = 'rgba(99,102,241,0.05)')}
                                onMouseLeave={e => (e.currentTarget.style.background = 'transparent')}
                              >
                                <td style={{ padding: '14px 18px' }}>
                                  <div style={{ display: 'flex', alignItems: 'center', gap: 10 }}>
                                    <span style={{ fontSize: 18 }}>{dsIcon}</span>
                                    <div>
                                      <div style={{ fontSize: 13, fontWeight: 700, color: C.text, fontFamily: 'monospace' }}>{dsName}</div>
                                      {a.datasource_host && <div style={{ fontSize: 11, color: C.textMuted }}>{a.datasource_host}</div>}
                                    </div>
                                  </div>
                                </td>
                                <td style={{ padding: '14px 18px' }}>
                                  <span style={{
                                    fontFamily: 'monospace', fontSize: 12, color: C.blue,
                                    background: 'rgba(96,165,250,0.1)', padding: '3px 8px', borderRadius: 4,
                                    border: '1px solid rgba(96,165,250,0.2)',
                                  }}>
                                    📊 {tableLabel || 'default_table'}
                                  </span>
                                </td>
                                <td style={{ padding: '14px 18px' }}>
                                  <div style={{ display: 'flex', alignItems: 'center', gap: 6 }}>
                                    <span style={{ fontFamily: 'monospace', fontSize: 13, fontWeight: 700, color: C.text }}>
                                      {colName}
                                    </span>
                                  </div>
                                  {a.qualified_path && (
                                    <div style={{ fontSize: 11, color: C.textMuted, fontFamily: 'monospace', marginTop: 2 }}>
                                      {a.qualified_path}
                                    </div>
                                  )}
                                </td>
                                <td style={{ padding: '14px 18px' }}>
                                  {dataType ? <Badge label={dataType} color={typeColor(dataType)} /> : <span style={{ color: C.textMuted, fontSize: 12 }}>—</span>}
                                </td>
                                <td style={{ padding: '14px 18px' }}>
                                  <Badge label={a.is_core ? 'CORE' : 'CUSTOM'} color={a.is_core ? C.teal : '#ED6C02'} />
                                </td>
                                <td style={{ padding: '14px 18px', textAlign: 'right' }}>
                                  <button
                                    onClick={() => removeMapping(a.edge_id)}
                                    style={{
                                      background: 'rgba(239,68,68,0.1)', border: `1px solid ${C.danger}44`,
                                      borderRadius: 6, color: C.danger, cursor: 'pointer',
                                      padding: '5px 12px', fontSize: 12, fontWeight: 600,
                                      transition: 'all 0.15s ease',
                                    }}
                                    onMouseEnter={e => (e.currentTarget.style.background = C.danger, e.currentTarget.style.color = '#fff')}
                                    onMouseLeave={e => (e.currentTarget.style.background = 'rgba(239,68,68,0.1)', e.currentTarget.style.color = C.danger)}
                                  >
                                    ✕ Remove
                                  </button>
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
                    excludeEdgeTypeId="0434ca1a-6543-42d3-9fce-f0b58b5fba34" // has_context
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
                      onDeleted={refreshAllData}
                      onUpdated={refreshAllData}
                    />
                  )}
                </div>
              )}

              {activeTab === 'lineage' && (
                <div style={{ width: '100%', height: '100%', display: 'flex', flexDirection: 'column' }}>
                  {/* Lineage Header / Toolbar */}
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
                    {/* Breadcrumb / Legend & Layer Toggles */}
                    <div style={{ display: 'flex', alignItems: 'center', gap: 12, fontSize: 12, flexWrap: 'wrap' }}>
                      <span style={{ fontSize: 11, fontWeight: 700, color: C.textMuted, textTransform: 'uppercase', letterSpacing: '0.04em' }}>
                        Layers:
                      </span>
                      
                      {/* BO / Terms Toggle */}
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

                      {/* Calculations / Dependencies Toggle */}
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
                          transition: 'all 0.15s ease',
                        }}
                      >
                        <span>🧮</span> Dependencies ({showCalcs ? 'ON' : 'OFF'})
                      </button>

                      {/* Datasources Toggle */}
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
                          transition: 'all 0.15s ease',
                        }}
                      >
                        <span>🗄️</span> Datasources ({showDatasources ? 'ON' : 'OFF'})
                      </button>
                    </div>

                    {/* Action Controls */}
                    {dsList.length > 0 && showDatasources && (
                      <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
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
                            display: 'flex',
                            alignItems: 'center',
                            gap: 4,
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
                      </div>
                    )}
                  </div>

                  {/* Canvas & Sliding Drawer Container */}
                  <div style={{ flex: 1, position: 'relative', overflow: 'hidden' }}>
                    <ReactFlow
                      nodes={nodes}
                      edges={edges}
                      onNodesChange={onNodesChange}
                      onEdgesChange={onEdgesChange}
                      nodeTypes={nodeTypes}
                      fitView
                      attributionPosition="bottom-right"
                      onNodeClick={(_event, node) => {
                        const rawId = node.data?.rawId || node.id.replace(/^(bo-|st-|bt-|calc-|other-|ds-)/, '');
                        if (rawId && rawId !== selectedId) {
                          setSearchParams({ id: rawId, tab: 'lineage' });
                        }
                      }}
                      onEdgeClick={(_event, edge) => {
                        setInspectDrawer({
                          open: true,
                          type: 'edge',
                          data: edge.data || {
                            predicate: edge.label,
                            id: edge.id,
                            source: edge.source,
                            target: edge.target,
                          }
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

                    {/* Sliding Metadata Inspector Drawer */}
                    {inspectDrawer.open && (
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
                        {/* Drawer Header */}
                        <div style={{
                          padding: '16px 20px',
                          borderBottom: `1px solid ${C.border}`,
                          background: 'rgba(255,255,255,0.03)',
                          display: 'flex',
                          alignItems: 'center',
                          justifyContent: 'space-between',
                        }}>
                          <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
                            <span style={{ fontSize: 16 }}>{inspectDrawer.type === 'node' ? 'ℹ️' : '🔗'}</span>
                            <span style={{ fontWeight: 800, fontSize: 13, textTransform: 'uppercase', letterSpacing: '0.05em', color: C.accent }}>
                              {inspectDrawer.type === 'node' ? 'Node Inspector' : 'Edge Attribute Inspector'}
                            </span>
                          </div>
                          <button
                            onClick={() => setInspectDrawer({ open: false, type: 'node', data: null })}
                            style={{
                              background: 'transparent',
                              border: 'none',
                              color: C.textMuted,
                              cursor: 'pointer',
                              fontSize: 16,
                              padding: '4px 8px',
                              borderRadius: 4,
                            }}
                            onMouseEnter={e => (e.currentTarget.style.color = '#fff')}
                            onMouseLeave={e => (e.currentTarget.style.color = C.textMuted)}
                          >
                            ✕
                          </button>
                        </div>

                        {/* Drawer Body */}
                        <div style={{ flex: 1, padding: '20px', overflowY: 'auto', display: 'flex', flexDirection: 'column', gap: 16 }}>
                          {inspectDrawer.type === 'node' ? (
                            <>
                              <div>
                                <div style={{ fontSize: 11, fontWeight: 700, color: C.textMuted, textTransform: 'uppercase', marginBottom: 4 }}>
                                  Node Name
                                </div>
                                <div style={{ fontSize: 16, fontWeight: 800, color: '#fff' }}>
                                  {inspectDrawer.data?.node_name || inspectDrawer.data?.label || inspectDrawer.data?.name}
                                </div>
                              </div>

                              {inspectDrawer.data?.catalog_type_name && (
                                <div>
                                  <div style={{ fontSize: 11, fontWeight: 700, color: C.textMuted, textTransform: 'uppercase', marginBottom: 4 }}>
                                    Type
                                  </div>
                                  <Badge label={inspectDrawer.data.catalog_type_name} color={C.accent} />
                                </div>
                              )}

                              {inspectDrawer.data?.description && (
                                <div>
                                  <div style={{ fontSize: 11, fontWeight: 700, color: C.textMuted, textTransform: 'uppercase', marginBottom: 4 }}>
                                    Description
                                  </div>
                                  <div style={{ fontSize: 13, color: C.text, lineHeight: 1.5, background: 'rgba(255,255,255,0.02)', padding: '8px 12px', borderRadius: 6, border: `1px solid ${C.border}` }}>
                                    {inspectDrawer.data.description}
                                  </div>
                                </div>
                              )}

                              {inspectDrawer.data?.formula && (
                                <div>
                                  <div style={{ fontSize: 11, fontWeight: 700, color: '#38BDF8', textTransform: 'uppercase', marginBottom: 4 }}>
                                    Calculation Formula
                                  </div>
                                  <div style={{ fontSize: 12, fontFamily: 'monospace', color: '#38BDF8', background: 'rgba(56,189,248,0.1)', padding: '8px 12px', borderRadius: 6, border: '1px solid rgba(56,189,248,0.3)' }}>
                                    {inspectDrawer.data.formula}
                                  </div>
                                </div>
                              )}

                              {inspectDrawer.data?.qualified_path && (
                                <div>
                                  <div style={{ fontSize: 11, fontWeight: 700, color: C.textMuted, textTransform: 'uppercase', marginBottom: 4 }}>
                                    Qualified Path
                                  </div>
                                  <div style={{ fontSize: 11, fontFamily: 'monospace', color: C.textMuted, background: 'rgba(0,0,0,0.3)', padding: '6px 10px', borderRadius: 6 }}>
                                    {inspectDrawer.data.qualified_path}
                                  </div>
                                </div>
                              )}

                              {/* Attributes & Properties */}
                              <div>
                                <div style={{ fontSize: 11, fontWeight: 700, color: C.textMuted, textTransform: 'uppercase', marginBottom: 6 }}>
                                  Node Attributes
                                </div>
                                <div style={{ background: 'rgba(0,0,0,0.3)', borderRadius: 6, padding: '10px 12px', fontSize: 11, fontFamily: 'monospace', color: C.textMuted, overflowX: 'auto', maxHeight: 200 }}>
                                  <pre style={{ margin: 0 }}>
                                    {JSON.stringify(inspectDrawer.data?.properties || inspectDrawer.data, null, 2)}
                                  </pre>
                                </div>
                              </div>

                              {/* Make Focal Button */}
                              {inspectDrawer.data?.id && inspectDrawer.data.id !== selectedTerm?.id && (
                                <button
                                  onClick={() => {
                                    setSearchParams({ id: inspectDrawer.data.rawId || inspectDrawer.data.id, tab: 'lineage' });
                                    setInspectDrawer({ open: false, type: 'node', data: null });
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
                                    display: 'flex',
                                    alignItems: 'center',
                                    justifyContent: 'center',
                                    gap: 6,
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
                                  {inspectDrawer.data?.predicate || inspectDrawer.data?.relationship_type || 'relationship'}
                                </div>
                              </div>

                              <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 12 }}>
                                <div>
                                  <div style={{ fontSize: 11, fontWeight: 700, color: C.textMuted, textTransform: 'uppercase', marginBottom: 4 }}>
                                    Source Node
                                  </div>
                                  <div style={{ fontSize: 12, fontWeight: 600, color: '#fff' }}>
                                    {inspectDrawer.data?.source_name || inspectDrawer.data?.source || 'Source'}
                                  </div>
                                </div>
                                <div>
                                  <div style={{ fontSize: 11, fontWeight: 700, color: C.textMuted, textTransform: 'uppercase', marginBottom: 4 }}>
                                    Target Node
                                  </div>
                                  <div style={{ fontSize: 12, fontWeight: 600, color: '#fff' }}>
                                    {inspectDrawer.data?.target_name || inspectDrawer.data?.target || 'Target'}
                                  </div>
                                </div>
                              </div>

                              {/* Edge Properties JSON */}
                              <div>
                                <div style={{ fontSize: 11, fontWeight: 700, color: C.textMuted, textTransform: 'uppercase', marginBottom: 6 }}>
                                  Edge Attributes &amp; Metadata
                                </div>
                                <div style={{ background: 'rgba(0,0,0,0.3)', borderRadius: 6, padding: '10px 12px', fontSize: 11, fontFamily: 'monospace', color: C.textMuted, overflowX: 'auto', maxHeight: 220 }}>
                                  <pre style={{ margin: 0 }}>
                                    {JSON.stringify(inspectDrawer.data, null, 2)}
                                  </pre>
                                </div>
                              </div>
                            </>
                          )}
                        </div>
                      </div>
                    )}
                  </div>
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
        onSave={refreshAllData}
      />
    </div>
  );
}
