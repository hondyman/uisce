import * as React from 'react';
import { useState, useMemo, useEffect, useCallback } from 'react';
import { useTheme, IconButton, Tooltip, Collapse } from '@mui/material';
import {
  AutoFixHigh as AutoFixHighIcon,
  Search as SearchIcon,
  Check as CheckIcon,
  Close as CloseIcon,
  Refresh as RefreshIcon,
  ArrowForward as ArrowForwardIcon,
  FlashOn as FlashOnIcon,
  LinkOff as LinkOffIcon,
  Psychology as PsychologyIcon,
  AutoAwesome as AutoAwesomeIcon,
  Warning as WarningIcon,
  Add as AddIcon,
  Hub as HubIcon,
  ExpandMore as ExpandMoreIcon,
  ExpandLess as ExpandLessIcon,
  DeleteForever as DeleteForeverIcon,
} from '@mui/icons-material';
import { useAccess } from '../../contexts/AccessContext';
import { readCachedSelection } from '../../utils/tenantScope';
import { useQuery } from '@tanstack/react-query';
import apiClient from '../../utils/apiClient';
import { useNodeTypes } from '../../api/nodeTypes';
import { useEdgeTypes } from '../../api/edgeTypes';
import type { EdgeType } from '../../types/edgeTypes';
import type {
  CatalogNodeItem,
  MatchSuggestion,
  GraphMappingRow,
  EdgeCardinality,
  RelatedItemCandidate,
  TargetNodeDraft,
  CompositeCluster,
  GovernanceTier,
  VendorAlignment
} from './types';
import {
  generateSuggestionsForNode,
  getEdgeTypeCardinality,
  discoverRelatedItems,
  isGenericColumn,
  extractEntityFromSourceNode,
  buildContextualTermName,
  detectCompositeClusters,
  UNIVERSAL_VALUE_TYPES,
  resolveUniversalParent
} from './utils/graphMatcher';
import { lookupBloombergField } from './constants/financialVendorDictionaries';
import { analyzeSampledValues } from './utils/financialChecksums';
import { runGeminiBatchMapping } from './services/geminiBatchMapper';
import { recordRejection, clearAllRejections, loadRejections } from './services/rejectionStore';

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

const CardinalityBadge: React.FC<{ cardinality: EdgeCardinality }> = ({ cardinality }) => {
  const color = cardinality === '1:1' ? '#F59E0B' : cardinality === '1:N' ? '#A78BFA' : '#60A5FA';
  return (
    <Tooltip title={`Relationship Cardinality: ${cardinality}`}>
      <span style={{
        display: 'inline-flex', alignItems: 'center', padding: '1px 6px',
        borderRadius: 4, fontSize: 10, fontWeight: 800,
        color, background: `${color}18`, border: `1px solid ${color}44`,
        fontFamily: 'monospace',
      }}>
        {cardinality}
      </span>
    </Tooltip>
  );
};

const ConfidencePill: React.FC<{ confidence: number; reason?: string; isAi?: boolean }> = ({ confidence, reason, isAi }) => {
  const color = confidence >= 85 ? '#10B981' : confidence >= 70 ? '#6366F1' : confidence >= 50 ? '#F59E0B' : '#8892A4';
  const label = `${confidence}% Match`;

  return (
    <Tooltip title={reason || 'Confidence score calculated by semantic matching engine'}>
      <span style={{
        display: 'inline-flex', alignItems: 'center', gap: 4, padding: '2px 8px',
        borderRadius: 9999, fontSize: 11, fontWeight: 700,
        color, background: `${color}18`, border: `1px solid ${color}44`,
        fontFamily: 'monospace', cursor: 'help',
      }}>
        <span>{isAi ? '✨' : '⚡'}</span> {label}
      </span>
    </Tooltip>
  );
};

const FilterPill: React.FC<{
  label: string; count?: number; active: boolean; onClick: () => void;
  accentColor?: string; border?: string; textMuted?: string; icon?: React.ReactNode;
}> = ({ label, count, active, onClick, accentColor = '#6366F1', border = 'rgba(255,255,255,0.07)', textMuted = '#8892A4', icon }) => (
  <button onClick={onClick} style={{
    display: 'flex', alignItems: 'center', gap: 6, padding: '6px 12px',
    background: active ? `${accentColor}18` : 'transparent',
    border: `1px solid ${active ? accentColor : border}`,
    borderRadius: 8, cursor: 'pointer', color: active ? accentColor : textMuted,
    fontSize: 12, fontWeight: 600, transition: 'all 0.15s ease',
    boxShadow: active ? `0 0 10px ${accentColor}33` : 'none',
    whiteSpace: 'nowrap',
  }}>
    {icon && <span>{icon}</span>}
    <span>{label}</span>
    {count !== undefined && (
      <span style={{
        fontSize: 10, padding: '1px 5px', borderRadius: 9999,
        background: active ? accentColor : 'rgba(255,255,255,0.1)',
        color: active ? '#0F172A' : textMuted, fontWeight: 700,
      }}>
        {count}
      </span>
    )}
  </button>
);

const Spinner: React.FC<{ accent?: string }> = ({ accent = '#6366F1' }) => (
  <div style={{ display: 'flex', justifyContent: 'center', alignItems: 'center', padding: 48 }}>
    <div style={{
      width: 32, height: 32, border: '3px solid rgba(255,255,255,0.1)',
      borderTop: `3px solid ${accent}`, borderRadius: '50%',
      animation: 'spin 0.8s linear infinite',
    }} />
    <style>{`@keyframes spin { to { transform: rotate(360deg); } }`}</style>
  </div>
);

// ─────────────────────────────────────────────
// Main Component
// ─────────────────────────────────────────────

export default function IntelligentSemanticMapper() {
  const theme = useTheme();
  const isDark = theme.palette.mode === 'dark';

  const C = useMemo(() => ({
    bg: isDark ? '#0A0C12' : '#F8FAFC',
    sidebar: isDark ? '#0F1117' : '#F1F5F9',
    panel: isDark ? '#13161E' : '#FFFFFF',
    panelHover: isDark ? '#1A1E2A' : '#F8FAFC',
    panelElevated: isDark ? '#181C26' : '#F1F5F9',
    border: isDark ? 'rgba(255,255,255,0.07)' : 'rgba(0,0,0,0.08)',
    borderStrong: isDark ? 'rgba(255,255,255,0.12)' : 'rgba(0,0,0,0.14)',
    accent: '#6366F1',
    accentDim: isDark ? 'rgba(99,102,241,0.15)' : 'rgba(99,102,241,0.08)',
    accentGlow: '0 0 20px rgba(99,102,241,0.4)',
    text: isDark ? '#E2E8F0' : '#0F172A',
    textMuted: isDark ? '#8892A4' : '#64748B',
    teal: '#2DD4BF',
    green: '#10B981',
    amber: '#F59E0B',
    red: '#EF4444',
    blue: '#60A5FA',
    purple: '#A78BFA',
  }), [isDark]);

  const { currentTenant, isPlatformOperator, accessLevel } = useAccess();
  const cachedSelection = readCachedSelection();
  const tenantId = currentTenant?.id ?? cachedSelection.tenant?.id;

  const isWriter = isPlatformOperator || accessLevel === 'tenant_admin' || accessLevel === 'platform_operator';

  // 1. Fetch Catalog Node Types
  const { data: nodeTypesRaw, isLoading: nodeTypesLoading } = useNodeTypes({ tenantId });
  const nodeTypes = useMemo(() => Array.isArray(nodeTypesRaw) ? nodeTypesRaw : [], [nodeTypesRaw]);

  // 2. Fetch Catalog Edge Types
  const { data: edgeTypesRaw, isLoading: edgeTypesLoading } = useEdgeTypes(tenantId || '');
  const edgeTypes = useMemo(() => Array.isArray(edgeTypesRaw) ? edgeTypesRaw : [], [edgeTypesRaw]);

  // States: Selection & Filtering
  const [selectedSourceTypeId, setSelectedSourceTypeId] = useState<string>('');
  const [selectedEdgeTypeId, setSelectedEdgeTypeId] = useState<string>('');
  const [searchTerm, setSearchTerm] = useState<string>('');
  const [filterTab, setFilterTab] = useState<'all' | 'unmapped' | 'high_confidence' | 'composite_clusters' | 'vendor_aligned' | 'generic_collisions' | 'needs_review' | 'has_see_also' | 'mapped'>('all');
  
  // Dynamic Manual Overrides & AI state
  const [manualTargetSelections, setManualTargetSelections] = useState<Record<string, CatalogNodeItem | null>>({});
  const [aiSuggestionsMap, setAiSuggestionsMap] = useState<Map<string, MatchSuggestion>>(new Map());
  const [isAiRunning, setIsAiRunning] = useState<boolean>(false);
  const [aiProgress, setAiProgress] = useState<{ processed: number; total: number } | null>(null);
  const [isBatchAutoMapping, setIsBatchAutoMapping] = useState<boolean>(false);
  const [batchAutoMapProgress, setBatchAutoMapProgress] = useState<{ current: number; total: number; currentItem: string } | null>(null);
  const [savingRows, setSavingRows] = useState<Set<string>>(new Set());
  const [toastMessage, setToastMessage] = useState<string | null>(null);
  const [expandedSeeAlso, setExpandedSeeAlso] = useState<Record<string, boolean>>({});
  const [rejectionVersion, setRejectionVersion] = useState<number>(0);
  const [sampleProfiles, setSampleProfiles] = useState<Record<string, { sampleValues: string[]; pattern: string; count: number; description: string; bloombergCandidate?: string }>>({});
  const [loadingProfiles, setLoadingProfiles] = useState<Set<string>>(new Set());

  // Auto-select initial Source Node Type (prefer Column, Database Column, or Business Term)
  useEffect(() => {
    if (!selectedSourceTypeId && nodeTypes.length > 0) {
      const preferred = nodeTypes.find((nt: any) => {
        const name = String(nt.catalog_type_name || '').toLowerCase();
        return name === 'column' || name === 'database_column' || name.includes('column');
      }) || nodeTypes.find((nt: any) => {
        const name = String(nt.catalog_type_name || '').toLowerCase();
        return name === 'business_term' || name.includes('business');
      }) || nodeTypes[0];

      if (preferred) setSelectedSourceTypeId(preferred.id);
    }
  }, [selectedSourceTypeId, nodeTypes]);

  // Available Edge Types for currently selected Source Node Type
  const availableEdgeTypes = useMemo(() => {
    if (!selectedSourceTypeId) return [];
    return edgeTypes.filter((et: EdgeType) => et.subject_node_type_id === selectedSourceTypeId);
  }, [selectedSourceTypeId, edgeTypes]);

  // Auto-select initial Edge Type when availableEdgeTypes changes
  useEffect(() => {
    if (availableEdgeTypes.length > 0) {
      const match = availableEdgeTypes.some((et: EdgeType) => et.id === selectedEdgeTypeId);
      if (!match) {
        setSelectedEdgeTypeId(availableEdgeTypes[0].id);
      }
    } else if (selectedEdgeTypeId !== '') {
      setSelectedEdgeTypeId('');
    }
  }, [availableEdgeTypes, selectedEdgeTypeId]);

  const selectedSourceNodeType = useMemo(() => {
    return nodeTypes.find((nt: any) => nt.id === selectedSourceTypeId) || null;
  }, [nodeTypes, selectedSourceTypeId]);

  const selectedEdgeType = useMemo(() => {
    return edgeTypes.find((et: EdgeType) => et.id === selectedEdgeTypeId) || null;
  }, [edgeTypes, selectedEdgeTypeId]);

  const selectedTargetNodeType = useMemo(() => {
    if (!selectedEdgeType) return null;
    return nodeTypes.find((nt: any) => nt.id === selectedEdgeType.object_node_type_id) || null;
  }, [nodeTypes, selectedEdgeType]);

  const cardinality = useMemo(() => {
    return getEdgeTypeCardinality(selectedEdgeType);
  }, [selectedEdgeType]);

  // 3. Fetch Source Nodes (with fallback to type name if node_type_id returns 0)
  const {
    data: sourceNodesRaw,
    isLoading: sourceNodesLoading,
    refetch: refetchSourceNodes,
  } = useQuery<CatalogNodeItem[]>({
    queryKey: ['mapper-source-nodes', tenantId, selectedSourceTypeId, selectedSourceNodeType?.catalog_type_name],
    queryFn: async () => {
      if (!tenantId || !selectedSourceTypeId) return [];
      
      // 1. Try querying by node_type_id
      let res = await apiClient<any>(`/api/catalog/nodes?node_type_id=${selectedSourceTypeId}&tenant_id=${tenantId}&limit=50000`);
      let list = Array.isArray(res) ? res : ((res as any)?.data ?? []);

      // 2. Fallback: Query by type name
      if ((!list || list.length === 0) && selectedSourceNodeType?.catalog_type_name) {
        res = await apiClient<any>(`/api/catalog/nodes?type=${selectedSourceNodeType.catalog_type_name}&tenant_id=${tenantId}&limit=50000`);
        list = Array.isArray(res) ? res : ((res as any)?.data ?? []);
      }

      // 3. Fallback: Column aliases
      if ((!list || list.length === 0) && (selectedSourceNodeType?.catalog_type_name === 'database_column' || selectedSourceNodeType?.catalog_type_name === 'column')) {
        const altType = selectedSourceNodeType.catalog_type_name === 'database_column' ? 'column' : 'database_column';
        res = await apiClient<any>(`/api/catalog/nodes?type=${altType}&tenant_id=${tenantId}&limit=50000`);
        list = Array.isArray(res) ? res : ((res as any)?.data ?? []);
      }

      return list || [];
    },
    enabled: !!tenantId && !!selectedSourceTypeId,
  });

  // 4. Fetch Target Nodes (with fallback to type name if node_type_id returns 0)
  const {
    data: targetNodesRaw,
    isLoading: targetNodesLoading,
    refetch: refetchTargetNodes,
  } = useQuery<CatalogNodeItem[]>({
    queryKey: ['mapper-target-nodes', tenantId, selectedEdgeType?.object_node_type_id, selectedTargetNodeType?.catalog_type_name],
    queryFn: async () => {
      if (!tenantId || !selectedEdgeType?.object_node_type_id) return [];
      
      let res = await apiClient<any>(`/api/catalog/nodes?node_type_id=${selectedEdgeType.object_node_type_id}&tenant_id=${tenantId}&limit=50000`);
      let list = Array.isArray(res) ? res : ((res as any)?.data ?? []);

      if ((!list || list.length === 0) && selectedTargetNodeType?.catalog_type_name) {
        res = await apiClient<any>(`/api/catalog/nodes?type=${selectedTargetNodeType.catalog_type_name}&tenant_id=${tenantId}&limit=50000`);
        list = Array.isArray(res) ? res : ((res as any)?.data ?? []);
      }

      return list || [];
    },
    enabled: !!tenantId && !!selectedEdgeType?.object_node_type_id,
  });

  // 5. Fetch Existing Edges for this tenant
  const {
    data: edgesRaw,
    isLoading: edgesLoading,
    refetch: refetchEdges,
  } = useQuery<any[]>({
    queryKey: ['mapper-edges', tenantId],
    queryFn: async () => {
      if (!tenantId) return [];
      const res = await apiClient<any>(`/api/glossary/edges?tenant_id=${tenantId}`);
      return Array.isArray(res) ? res : ((res as any)?.data ?? []);
    },
    enabled: !!tenantId,
  });

  const sourceNodes = useMemo(() => Array.isArray(sourceNodesRaw) ? sourceNodesRaw : [], [sourceNodesRaw]);
  const targetNodes = useMemo(() => Array.isArray(targetNodesRaw) ? targetNodesRaw : [], [targetNodesRaw]);
  const edges = useMemo(() => Array.isArray(edgesRaw) ? edgesRaw : [], [edgesRaw]);

  // Target Node Map for lookup
  const targetNodesMap = useMemo(() => {
    const map = new Map<string, CatalogNodeItem>();
    targetNodes.forEach(tn => map.set(tn.id, tn));
    return map;
  }, [targetNodes]);

  // Map of Target Node IDs that are already mapped to a Source Node (for 1:1 conflict detection)
  const targetToSourceMap = useMemo(() => {
    const map = new Map<string, string>(); // targetId -> sourceNodeName
    if (!selectedEdgeType) return map;
    const edgeTypeId = selectedEdgeType.id;
    const edgeTypeName = selectedEdgeType.edge_type_name;

    edges.forEach((e: any) => {
      const isMatch = e.edge_type_id === edgeTypeId || e.edge_type_name === edgeTypeName || e.relationship_type === edgeTypeName;
      if (isMatch) {
        const tId = e.object_node_id || e.target_node_id;
        const sId = e.subject_node_id || e.source_node_id;
        const srcNode = sourceNodes.find(s => s.id === sId);
        if (tId && srcNode) {
          map.set(tId, srcNode.node_name);
        }
      }
    });
    return map;
  }, [edges, selectedEdgeType, sourceNodes]);

  // Existing linked node IDs set per source node
  const linkedNodeIdsBySource = useMemo(() => {
    const map = new Map<string, Set<string>>();
    edges.forEach((e: any) => {
      const sId = e.subject_node_id || e.source_node_id;
      const tId = e.object_node_id || e.target_node_id;
      if (sId && tId) {
        if (!map.has(sId)) map.set(sId, new Set());
        map.get(sId)!.add(tId);
      }
    });
    return map;
  }, [edges]);

  // Detect Multi-Column Composite Semantic Clusters
  const compositeClusters = useMemo(() => {
    return detectCompositeClusters(sourceNodes);
  }, [sourceNodes]);

  // Construct Mapping Rows with Rejection Memory, AI Overrides, and See-Also Discovery
  const mappingRows: GraphMappingRow[] = useMemo(() => {
    if (!selectedEdgeType || sourceNodes.length === 0) return [];

    const edgeTypeId = selectedEdgeType.id;
    const edgeTypeName = selectedEdgeType.edge_type_name;
    const targetTypeId = selectedEdgeType.object_node_type_id;
    const targetTypeName = selectedTargetNodeType?.catalog_type_name || 'term';

    return sourceNodes.map(sourceNode => {
      // Find current mapping
      const matchingEdge = edges.find((e: any) => {
        const isSubject = e.subject_node_id === sourceNode.id || e.source_node_id === sourceNode.id;
        const matchesType = e.edge_type_id === edgeTypeId || e.edge_type_name === edgeTypeName || e.relationship_type === edgeTypeName;
        return isSubject && matchesType;
      });

      let currentTargetNode: CatalogNodeItem | null = null;
      let existingEdgeId: string | undefined = undefined;

      if (matchingEdge) {
        existingEdgeId = matchingEdge.id;
        const targetId = matchingEdge.object_node_id || matchingEdge.target_node_id;
        if (targetId) {
          currentTargetNode = targetNodesMap.get(targetId) || {
            id: targetId,
            node_name: matchingEdge.target_name || matchingEdge.object_name || 'Mapped Node',
            type: matchingEdge.type || 'custom',
          };
        }
      }

      // Check if AI generated suggestion is available
      const aiSuggestion = aiSuggestionsMap.get(sourceNode.id);

      // Local matcher suggestion
      const localResult = generateSuggestionsForNode(
        sourceNode,
        targetNodes,
        tenantId || '',
        45,
        targetTypeId,
        targetTypeName
      );

      const topSuggestion = aiSuggestion || localResult.topSuggestion;
      const alternatives = localResult.alternatives;

      // Manual selection in UI
      const manualSelection = manualTargetSelections[sourceNode.id];
      const selectedTargetNode = manualSelection !== undefined
        ? manualSelection
        : currentTargetNode || topSuggestion?.targetNode || null;

      // Cardinality Conflict Check (1:1 constraint)
      let cardinalityConflict: string | undefined = undefined;
      if (cardinality === '1:1' && selectedTargetNode && !currentTargetNode) {
        const existingClaimedBy = targetToSourceMap.get(selectedTargetNode.id);
        if (existingClaimedBy && existingClaimedBy !== sourceNode.node_name) {
          cardinalityConflict = `1:1 Conflict: Already mapped to "${existingClaimedBy}"`;
        }
      }

      // Discover "See Also" / Related Items across catalog
      const linkedSet = linkedNodeIdsBySource.get(sourceNode.id) || new Set();
      const relatedItems = discoverRelatedItems(sourceNode, targetNodes, linkedSet);

      // Generic Term Disambiguation info
      const isGeneric = topSuggestion?.isGenericCollision ?? isGenericColumn(sourceNode.node_name || '');
      const entityName = topSuggestion?.contextualEntity || extractEntityFromSourceNode(sourceNode);
      const contextualTerm = topSuggestion?.suggestedContextualTerm || (entityName ? buildContextualTermName(entityName, sourceNode.node_name || '') : undefined);

      // Composite cluster membership
      const matchedCluster = compositeClusters.find(c =>
        c.members.some(m => m.sourceNodeId === sourceNode.id || m.sourceColumn.toLowerCase() === (sourceNode.node_name || '').toLowerCase())
      );

      const universalParent = topSuggestion?.universalParentName || (matchedCluster ? matchedCluster.universalParent : resolveUniversalParent(sourceNode.node_name || '')?.name);
      const governanceTier: GovernanceTier = currentTargetNode
        ? (currentTargetNode.type === 'core' ? 'gold_certified' : 'custom')
        : (topSuggestion?.governanceTier || 'draft');

      // Bloomberg Financial Vendor Alignment
      const bbgDirect = lookupBloombergField(sourceNode.node_name || '');
      const vendorAlignment: VendorAlignment | undefined = topSuggestion?.vendorAlignment || (bbgDirect ? {
        vendor: 'BLOOMBERG',
        mnemonic: bbgDirect.mnemonic,
        canonicalTermName: bbgDirect.canonicalTermName,
        category: bbgDirect.category,
        description: bbgDirect.description,
        feedType: bbgDirect.feedType,
      } : undefined);

      return {
        id: sourceNode.id,
        sourceNode,
        edgeType: selectedEdgeType,
        cardinality,
        currentTargetNode,
        existingEdgeId,
        suggestion: topSuggestion,
        alternativeSuggestions: alternatives,
        selectedTargetNode,
        selectedTargetDraft: topSuggestion?.targetDraft || null,
        isMapped: !!currentTargetNode,
        isModified: manualSelection !== undefined,
        isSaving: savingRows.has(sourceNode.id),
        relatedItems,
        cardinalityConflict,
        isGenericCollision: isGeneric,
        contextualEntity: entityName,
        suggestedContextualTerm: contextualTerm,
        compositeCluster: matchedCluster,
        universalParentName: universalParent,
        governanceTier,
        vendorAlignment,
      };
    });
  }, [
    sourceNodes, targetNodes, edges, selectedEdgeType, selectedTargetNodeType,
    targetNodesMap, aiSuggestionsMap, manualTargetSelections, savingRows,
    cardinality, targetToSourceMap, linkedNodeIdsBySource, tenantId, rejectionVersion,
    compositeClusters
  ]);

  // Filtered rows
  const filteredRows = useMemo(() => {
    return mappingRows.filter(row => {
      // Search term filter
      if (searchTerm.trim()) {
        const query = searchTerm.toLowerCase();
        const srcName = (row.sourceNode.node_name || '').toLowerCase();
        const srcPath = (row.sourceNode.qualified_path || '').toLowerCase();
        const tgtName = (row.currentTargetNode?.node_name || row.suggestion?.targetNode?.node_name || '').toLowerCase();
        const bbgMnemonic = (row.vendorAlignment?.mnemonic || '').toLowerCase();
        if (!srcName.includes(query) && !srcPath.includes(query) && !tgtName.includes(query) && !bbgMnemonic.includes(query)) {
          return false;
        }
      }

      // Filter tabs
      if (filterTab === 'unmapped') return !row.isMapped;
      if (filterTab === 'mapped') return row.isMapped;
      if (filterTab === 'high_confidence') return !row.isMapped && (row.suggestion?.confidence ?? 0) >= 80;
      if (filterTab === 'composite_clusters') return !!row.compositeCluster;
      if (filterTab === 'vendor_aligned') return !!row.vendorAlignment;
      if (filterTab === 'generic_collisions') return !row.isMapped && row.isGenericCollision;
      if (filterTab === 'needs_review') return !row.isMapped && (row.suggestion?.confidence ?? 0) >= 50 && (row.suggestion?.confidence ?? 0) < 80;
      if (filterTab === 'has_see_also') return row.relatedItems.length > 0;

      return true;
    });
  }, [mappingRows, searchTerm, filterTab]);

  // Summary Metrics
  const totalCount = mappingRows.length;
  const mappedCount = mappingRows.filter(r => r.isMapped).length;
  const highConfidenceCount = mappingRows.filter(r => !r.isMapped && (r.suggestion?.confidence ?? 0) >= 80).length;
  const genericCollisionsCount = mappingRows.filter(r => !r.isMapped && r.isGenericCollision).length;
  const compositeClustersCount = compositeClusters.length;
  const vendorAlignedCount = mappingRows.filter(r => !!r.vendorAlignment).length;
  const newNodesSuggestedCount = mappingRows.filter(r => !r.isMapped && r.suggestion?.targetDraft?.isNew).length;
  const unmappedCount = totalCount - mappedCount;

  // Actions: Switch single row to contextual entity term
  const handleUseContextualTerm = (row: GraphMappingRow) => {
    if (!row.suggestedContextualTerm) return;
    const targetType = selectedTargetNodeType?.catalog_type_name || 'semantic_term';
    const contextualDraft: CatalogNodeItem = {
      id: `draft-${row.sourceNode.id}`,
      node_name: row.suggestedContextualTerm,
      description: `Contextual ${targetType} for ${row.contextualEntity || ''} ${row.sourceNode.node_name}`,
      catalog_type: targetType,
      catalog_type_name: targetType,
      properties: {
        data_type: row.sourceNode.properties?.data_type || 'string',
        source_origin: row.sourceNode.node_name,
        parent_entity: row.contextualEntity,
        is_contextual_disambiguated: true,
      }
    };

    setManualTargetSelections(prev => ({
      ...prev,
      [row.sourceNode.id]: contextualDraft,
    }));
    setToastMessage(`Switched "${row.sourceNode.node_name}" to contextual term "${row.suggestedContextualTerm}"`);
  };

  // Actions: Remediate all generic columns in batch with table context
  const handleRemediateAllGenericTerms = () => {
    const unmappedGenerics = mappingRows.filter(r => !r.isMapped && r.isGenericCollision && r.suggestedContextualTerm);
    if (unmappedGenerics.length === 0) return;

    const targetType = selectedTargetNodeType?.catalog_type_name || 'semantic_term';
    const updates: Record<string, CatalogNodeItem> = {};

    unmappedGenerics.forEach(r => {
      updates[r.sourceNode.id] = {
        id: `draft-${r.sourceNode.id}`,
        node_name: r.suggestedContextualTerm!,
        description: `Contextual ${targetType} for ${r.contextualEntity || ''} ${r.sourceNode.node_name}`,
        catalog_type: targetType,
        catalog_type_name: targetType,
        properties: {
          data_type: r.sourceNode.properties?.data_type || 'string',
          source_origin: r.sourceNode.node_name,
          parent_entity: r.contextualEntity,
          is_contextual_disambiguated: true,
        }
      };
    });

    setManualTargetSelections(prev => ({ ...prev, ...updates }));
    setToastMessage(`Remediated ${unmappedGenerics.length} generic terms with table context!`);
  };

  // Actions: Auto-Map entire Composite Cluster (e.g. 5 columns in Address bundle conforming to ISO 19160)
  const handleMapCompositeCluster = async (cluster: CompositeCluster) => {
    if (!tenantId || !selectedEdgeType) return;

    let successCount = 0;
    for (const member of cluster.members) {
      const row = mappingRows.find(r => r.sourceNode.id === member.sourceNodeId);
      if (!row || row.isMapped) continue;

      const targetType = selectedTargetNodeType?.catalog_type_name || 'semantic_term';
      const termDraft: CatalogNodeItem = {
        id: `draft-${row.sourceNode.id}`,
        node_name: member.suggestedTermName,
        description: `Composite ${cluster.clusterType} member (${member.subProperty}) for ${cluster.entityName}`,
        catalog_type: targetType,
        catalog_type_name: targetType,
        properties: {
          data_type: row.sourceNode.properties?.data_type || 'string',
          cluster_id: cluster.clusterId,
          cluster_type: cluster.clusterType,
          sub_property: member.subProperty,
          parent_entity: cluster.entityName,
          standard: cluster.standard,
        }
      };

      try {
        await handleAcceptMapping(row, termDraft);
        successCount++;
      } catch (e) {
        console.error(`Failed mapping composite member ${member.sourceColumn}`, e);
      }
    }

    setToastMessage(`✨ Auto-mapped composite cluster "${cluster.compositeTermName}" (${successCount} columns linked to ${cluster.standard})!`);
    await refetchEdges();
  };

  // Actions: Promote a tenant custom term to Gold Copy Core
  const handlePromoteToGoldCore = async (row: GraphMappingRow) => {
    if (!row.currentTargetNode?.id || !tenantId) return;
    try {
      await apiClient(`/api/glossary/terms/${row.currentTargetNode.id}?tenant_id=${tenantId}`, {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          properties: {
            ...(row.currentTargetNode.properties || {}),
            governance_tier: 'gold_certified',
            is_gold_copy: true,
          }
        })
      });

      setToastMessage(`🏆 Promoted "${row.currentTargetNode.node_name}" to Certified Gold Copy Core!`);
      await refetchTargetNodes();
    } catch (e: any) {
      alert(`Could not promote term: ${e?.message || e}`);
    }
  };

  // Actions: Zero-Impact Safe Physical Page Sampling (500 rows)
  const handleProfileColumn = async (node: CatalogNodeItem) => {
    if (!tenantId || loadingProfiles.has(node.id)) return;
    setLoadingProfiles(prev => new Set(prev).add(node.id));

    try {
      const res = await apiClient<any>(`/api/glossary/profile-sample?tenant_id=${tenantId}`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          node_id: node.id,
          node_name: node.node_name,
          qualified_path: node.qualified_path,
          sample_size: 500,
        })
      });

      if (res && Array.isArray(res.sample_values)) {
        const analysis = analyzeSampledValues(res.sample_values);
        setSampleProfiles(prev => ({
          ...prev,
          [node.id]: {
            sampleValues: res.sample_values,
            pattern: analysis.pattern !== 'GENERIC_STRING' ? analysis.pattern : res.pattern_detected,
            count: res.sample_count || 500,
            description: analysis.description,
            bloombergCandidate: analysis.bloombergCandidate || res.bloomberg_candidate,
          }
        }));

        setToastMessage(`⚡ Sampled 500 rows for "${node.node_name}": ${analysis.description}`);
      }
    } catch (e) {
      console.warn('Sample profiling warning:', e);
    } finally {
      setLoadingProfiles(prev => {
        const next = new Set(prev);
        next.delete(node.id);
        return next;
      });
    }
  };

  // Actions: Accept / Save Mapping (handles Auto-Creating Target Node + Edge with JSON properties)
  const handleAcceptMapping = async (row: GraphMappingRow, targetNodeToSave?: CatalogNodeItem, options?: { silent?: boolean }) => {
    if (!tenantId || !selectedEdgeType) return;
    const target = targetNodeToSave || row.selectedTargetNode || row.suggestion?.targetNode;
    if (!target) return;

    if (row.cardinalityConflict) {
      if (!options?.silent) {
        alert(`Cannot map: ${row.cardinalityConflict}`);
      }
      throw new Error(`Cardinality conflict: ${row.cardinalityConflict}`);
    }

    setSavingRows(prev => new Set(prev).add(row.sourceNode.id));

    try {
      let targetId = target.id;

      // 1. If target node is newly suggested draft, check if matching target node exists or create target catalog node first!
      if ((row.suggestion?.targetDraft?.isNew || target.id.startsWith('draft-')) && target.id.startsWith('draft-')) {
        const draft = row.suggestion?.targetDraft || {
          node_name: target.node_name,
          description: target.description,
          catalog_type: target.catalog_type || 'semantic_term',
          properties: target.properties || {},
        };

        // Check if an existing target node already has this name in targetNodes
        const existingNode = targetNodes.find(t => !t.id.startsWith('draft-') && t.node_name.toLowerCase() === draft.node_name.toLowerCase());
        if (existingNode) {
          targetId = existingNode.id;
        } else {
          const catalogType = selectedTargetNodeType?.catalog_type_name === 'business_term'
            ? 'business_term'
            : selectedTargetNodeType?.catalog_type_name === 'business_object'
            ? 'business_object'
            : (draft.catalog_type || 'semantic_term');
          const createdNodeRes = await apiClient<any>(`/api/glossary/terms?tenant_id=${tenantId}`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({
              node_name: draft.node_name,
              description: draft.description || `Auto-created term from ${row.sourceNode.node_name}`,
              catalog_type: catalogType,
              properties: draft.properties || {},
            })
          });

          if (createdNodeRes?.id) {
            targetId = createdNodeRes.id;
            await refetchTargetNodes();
          }
        }
      }

      // 2. Create the Edge with JSON properties attached
      const edgeProperties = {
        transformation: row.suggestion?.edgeDraft?.transformation || 'direct',
        confidence: row.suggestion?.confidence || 100,
        mapping_notes: row.suggestion?.edgeDraft?.mapping_notes || `Mapped from ${row.sourceNode.node_name}`,
        source_column_path: row.sourceNode.qualified_path,
        universal_standard: row.universalParentName,
      };

      await apiClient(`/api/glossary/edges?tenant_id=${tenantId}`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          subject_node_id: row.sourceNode.id,
          object_node_id: targetId,
          edge_type_id: selectedEdgeType.id,
          edge_type_name: selectedEdgeType.edge_type_name,
          properties: edgeProperties,
          tenant_id: tenantId,
        })
      });

      // 3. Create Hierarchical Ontology Edges (SPECIALIZES / BELONGS_TO) if present
      if (row.suggestion?.hierarchicalEdgesToCreate && row.suggestion.hierarchicalEdgesToCreate.length > 0) {
        for (const hEdge of row.suggestion.hierarchicalEdgesToCreate) {
          try {
            let parentTargetNode = targetNodes.find(t => t.node_name.toLowerCase() === hEdge.target_node_name.toLowerCase());
            let parentTargetId = parentTargetNode?.id;

            if (!parentTargetId) {
              const createdParentRes = await apiClient<any>(`/api/glossary/terms?tenant_id=${tenantId}`, {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({
                  node_name: hEdge.target_node_name,
                  description: `Universal semantic archetype: ${hEdge.target_node_name}`,
                  catalog_type: hEdge.target_catalog_type || 'semantic_term',
                  properties: hEdge.properties || {},
                })
              }).catch(() => null);
              parentTargetId = createdParentRes?.id;
            }

            if (parentTargetId && parentTargetId !== targetId) {
              await apiClient(`/api/glossary/edges?tenant_id=${tenantId}`, {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({
                  subject_node_id: targetId,
                  object_node_id: parentTargetId,
                  edge_type_id: hEdge.edge_type_name,
                  edge_type_name: hEdge.edge_type_name,
                  properties: hEdge.properties || {},
                  tenant_id: tenantId,
                })
              }).catch(() => null);
            }
          } catch (e) {
            console.warn(`Could not create hierarchical edge ${hEdge.edge_type_name}:`, e);
          }
        }
      }

      // Clear manual selection override
      setManualTargetSelections(prev => {
        const next = { ...prev };
        delete next[row.sourceNode.id];
        return next;
      });

      if (!options?.silent) {
        setToastMessage(`Mapped "${row.sourceNode.node_name}" → "${target.node_name}"`);
        await refetchEdges();
      }
    } catch (err: any) {
      console.error(err);
      if (!options?.silent) {
        alert(`Failed to save mapping: ${err?.message || 'Unknown error'}`);
      }
      throw err;
    } finally {
      setSavingRows(prev => {
        const next = new Set(prev);
        next.delete(row.sourceNode.id);
        return next;
      });
    }
  };

  // Actions: Reject Suggestion (permanently remembers so never suggested again)
  const handleRejectSuggestion = (row: GraphMappingRow) => {
    if (!tenantId || !row.suggestion) return;
    const targetName = row.suggestion.targetNode.node_name;
    const targetId = row.suggestion.targetNode.id;

    recordRejection(tenantId, row.sourceNode.id, targetName);
    recordRejection(tenantId, row.sourceNode.id, targetId);

    // Clear from AI map if present
    setAiSuggestionsMap(prev => {
      const next = new Map(prev);
      next.delete(row.sourceNode.id);
      return next;
    });

    setToastMessage(`Rejected suggestion "${targetName}". It will not be suggested again.`);
    setRejectionVersion(v => v + 1);
  };

  // Actions: Unlink / Remove Mapping
  const handleUnlink = async (row: GraphMappingRow) => {
    if (!tenantId || !row.existingEdgeId) return;

    setSavingRows(prev => new Set(prev).add(row.sourceNode.id));

    try {
      await apiClient(`/api/glossary/edges/${row.existingEdgeId}?tenant_id=${tenantId}`, {
        method: 'DELETE',
      });

      setToastMessage(`Unlinked "${row.sourceNode.node_name}"`);
      await refetchEdges();
    } catch (err: any) {
      console.error(err);
      alert(`Failed to remove mapping: ${err?.message || 'Unknown error'}`);
    } finally {
      setSavingRows(prev => {
        const next = new Set(prev);
        next.delete(row.sourceNode.id);
        return next;
      });
    }
  };

  // Actions: Link "See Also" Related Item
  const handleLinkSeeAlso = async (sourceNode: CatalogNodeItem, peerItem: RelatedItemCandidate) => {
    if (!tenantId) return;
    try {
      // Find or use see_also edge type
      const seeAlsoEdgeType = edgeTypes.find(et => {
        const name = String(et.edge_type_name || '').toLowerCase();
        return name.includes('see_also') || name.includes('related');
      }) || selectedEdgeType;

      if (!seeAlsoEdgeType) return;

      await apiClient(`/api/glossary/edges?tenant_id=${tenantId}`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          subject_node_id: sourceNode.id,
          object_node_id: peerItem.id,
          edge_type_id: seeAlsoEdgeType.id,
          edge_type_name: 'see_also',
          properties: { relation: 'cross_reference', discovered_by: 'ai_cluster' },
          tenant_id: tenantId,
        })
      });

      setToastMessage(`Linked "${sourceNode.node_name}" ↔ See Also: "${peerItem.node_name}"`);
      await refetchEdges();
    } catch (e: any) {
      console.error(e);
      alert(`Failed to link related item: ${e?.message || 'Unknown error'}`);
    }
  };

  // Actions: Run Gemini AI Batch Suggestions
  const handleRunGeminiBatch = async () => {
    if (!tenantId || !selectedEdgeType || sourceNodes.length === 0) return;
    const unmappedSourceNodes = mappingRows.filter(r => !r.isMapped).map(r => r.sourceNode);
    if (unmappedSourceNodes.length === 0) {
      alert('All items are already mapped!');
      return;
    }

    setIsAiRunning(true);
    setAiProgress({ processed: 0, total: unmappedSourceNodes.length });

    try {
      const batchResults = await runGeminiBatchMapping(
        unmappedSourceNodes,
        targetNodes,
        selectedEdgeType,
        selectedSourceNodeType,
        selectedTargetNodeType,
        tenantId,
        (processed, total) => setAiProgress({ processed, total })
      );

      setAiSuggestionsMap(batchResults);
      setToastMessage(`✨ Gemini AI generated suggestions for ${batchResults.size} items!`);
    } catch (err) {
      console.error(err);
      alert('Error running AI batch mapping');
    } finally {
      setIsAiRunning(false);
      setAiProgress(null);
    }
  };

  // Actions: Batch Auto-Map High Confidence
  const handleAutoMapAllHighConfidence = async () => {
    if (!tenantId || !selectedEdgeType) return;
    const candidates = mappingRows.filter(r => !r.isMapped && (r.suggestion?.confidence ?? 0) >= 80 && !r.cardinalityConflict);
    if (candidates.length === 0) {
      alert('No unmapped items with high confidence (≥80%) and valid cardinality found.');
      return;
    }

    if (!window.confirm(`Auto-map ${candidates.length} items with confidence ≥ 80%?`)) {
      return;
    }

    setIsBatchAutoMapping(true);
    setBatchAutoMapProgress({ current: 0, total: candidates.length, currentItem: '' });

    let successCount = 0;
    let failedCount = 0;
    const errorList: string[] = [];

    try {
      for (let i = 0; i < candidates.length; i++) {
        const row = candidates[i];
        if (!row.suggestion?.targetNode) continue;
        
        setBatchAutoMapProgress({
          current: i + 1,
          total: candidates.length,
          currentItem: row.sourceNode.node_name,
        });

        try {
          await handleAcceptMapping(row, undefined, { silent: true });
          successCount++;
        } catch (e: any) {
          failedCount++;
          const errMsg = e?.message || 'Failed';
          errorList.push(`${row.sourceNode.node_name}: ${errMsg}`);
          console.error(`Failed to map ${row.sourceNode.node_name}`, e);
        }
      }

      if (failedCount === 0) {
        setToastMessage(`✨ Batch complete: Successfully auto-mapped all ${successCount} items!`);
      } else {
        setToastMessage(`⚠️ Batch complete: ${successCount} mapped, ${failedCount} failed.`);
        console.warn(`[Batch Auto-Map Errors]`, errorList);
      }
      await refetchEdges();
      await refetchTargetNodes();
    } finally {
      setIsBatchAutoMapping(false);
      setBatchAutoMapProgress(null);
    }
  };

  const isDataLoading = nodeTypesLoading || edgeTypesLoading || sourceNodesLoading || targetNodesLoading || edgesLoading;

  return (
    <div style={{ display: 'flex', flexDirection: 'column', height: '100%', background: C.bg, color: C.text, overflow: 'hidden' }}>
      
      {/* Toast Notification */}
      {toastMessage && (
        <div style={{
          position: 'fixed', bottom: 24, right: 24, zIndex: 1000,
          background: C.panel, border: `1px solid ${C.teal}`,
          borderRadius: 8, padding: '10px 18px', color: C.text,
          display: 'flex', alignItems: 'center', gap: 10,
          boxShadow: '0 8px 30px rgba(0,0,0,0.5)',
          animation: 'fadeIn 0.2s ease',
        }}>
          <CheckIcon sx={{ color: C.teal, fontSize: 18 }} />
          <span style={{ fontSize: 13, fontWeight: 600 }}>{toastMessage}</span>
          <IconButton size="small" onClick={() => setToastMessage(null)} sx={{ color: C.textMuted, ml: 1 }}>
            <CloseIcon fontSize="small" />
          </IconButton>
        </div>
      )}

      {/* ──────────────── Top Navigation & Config Bar ──────────────── */}
      <div style={{
        background: C.panel,
        borderBottom: `1px solid ${C.border}`,
        padding: '16px 24px',
        display: 'flex',
        flexDirection: 'column',
        gap: 14,
      }}>
        {/* Title & Summary Badges */}
        <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', flexWrap: 'wrap', gap: 16 }}>
          <div style={{ display: 'flex', alignItems: 'center', gap: 12 }}>
            <div style={{
              width: 38, height: 38, borderRadius: 8,
              background: 'rgba(99,102,241,0.15)', border: `1px solid rgba(99,102,241,0.3)`,
              display: 'flex', alignItems: 'center', justifyContent: 'center', color: C.accent
            }}>
              <AutoFixHighIcon sx={{ fontSize: 22 }} />
            </div>
            <div>
              <h1 style={{ margin: 0, fontSize: 18, fontWeight: 700, letterSpacing: '-0.01em', color: C.text }}>
                Intelligent Semantic &amp; Graph Mapper
              </h1>
              <div style={{ fontSize: 12, color: C.textMuted, marginTop: 2 }}>
                Intelligently discover, map, and create graph nodes and edges with Gemini AI batching &amp; cardinality rules
              </div>
            </div>
          </div>

          {/* Outlined Summary Badges */}
          <div style={{ display: 'flex', alignItems: 'center', gap: 8, flexWrap: 'wrap' }}>
            <span style={{
              display: 'inline-flex', alignItems: 'center', padding: '3px 10px',
              borderRadius: 9999, fontSize: 11, fontWeight: 700, letterSpacing: '0.04em',
              color: C.text, background: isDark ? 'rgba(255,255,255,0.06)' : 'rgba(0,0,0,0.05)',
              border: `1px solid ${C.borderStrong}`, fontFamily: 'monospace', textTransform: 'uppercase',
            }}>
              {totalCount} Total Items
            </span>
            {compositeClustersCount > 0 && (
              <span style={{
                display: 'inline-flex', alignItems: 'center', padding: '3px 10px',
                borderRadius: 9999, fontSize: 11, fontWeight: 700, letterSpacing: '0.04em',
                color: C.purple, background: isDark ? 'rgba(168,85,247,0.14)' : 'rgba(168,85,247,0.09)',
                border: `1px solid ${C.purple}66`, fontFamily: 'monospace', textTransform: 'uppercase',
              }}>
                📦 {compositeClustersCount} Composite Clusters
              </span>
            )}
            {vendorAlignedCount > 0 && (
              <span style={{
                display: 'inline-flex', alignItems: 'center', padding: '3px 10px',
                borderRadius: 9999, fontSize: 11, fontWeight: 700, letterSpacing: '0.04em',
                color: '#F97316', background: isDark ? 'rgba(249,115,22,0.14)' : 'rgba(249,115,22,0.09)',
                border: `1px solid rgba(249,115,22,0.45)`, fontFamily: 'monospace', textTransform: 'uppercase',
              }}>
                🟧 {vendorAlignedCount} Bloomberg Aligned
              </span>
            )}
            {genericCollisionsCount > 0 && (
              <span style={{
                display: 'inline-flex', alignItems: 'center', padding: '3px 10px',
                borderRadius: 9999, fontSize: 11, fontWeight: 700, letterSpacing: '0.04em',
                color: C.amber, background: isDark ? 'rgba(245,158,11,0.14)' : 'rgba(245,158,11,0.09)',
                border: `1px solid ${C.amber}66`, fontFamily: 'monospace', textTransform: 'uppercase',
              }}>
                ⚠️ {genericCollisionsCount} Generic Collisions
              </span>
            )}
            <span style={{
              display: 'inline-flex', alignItems: 'center', padding: '3px 10px',
              borderRadius: 9999, fontSize: 11, fontWeight: 700, letterSpacing: '0.04em',
              color: C.green, background: isDark ? 'rgba(16,185,129,0.12)' : 'rgba(16,185,129,0.08)',
              border: `1px solid ${C.green}44`, fontFamily: 'monospace', textTransform: 'uppercase',
            }}>
              {mappedCount} Mapped
            </span>
            <span style={{
              display: 'inline-flex', alignItems: 'center', padding: '3px 10px',
              borderRadius: 9999, fontSize: 11, fontWeight: 700, letterSpacing: '0.04em',
              color: C.purple, background: isDark ? 'rgba(167,139,250,0.12)' : 'rgba(167,139,250,0.08)',
              border: `1px solid ${C.purple}44`, fontFamily: 'monospace', textTransform: 'uppercase',
            }}>
              {highConfidenceCount} High Conf (≥80%)
            </span>
            {newNodesSuggestedCount > 0 && (
              <span style={{
                display: 'inline-flex', alignItems: 'center', padding: '3px 10px',
                borderRadius: 9999, fontSize: 11, fontWeight: 700, letterSpacing: '0.04em',
                color: C.teal, background: isDark ? 'rgba(45,212,191,0.12)' : 'rgba(45,212,191,0.08)',
                border: `1px solid ${C.teal}44`, fontFamily: 'monospace', textTransform: 'uppercase',
              }}>
                {newNodesSuggestedCount} New Suggested
              </span>
            )}
            <span style={{
              display: 'inline-flex', alignItems: 'center', padding: '3px 10px',
              borderRadius: 9999, fontSize: 11, fontWeight: 700, letterSpacing: '0.04em',
              color: C.amber, background: isDark ? 'rgba(245,158,11,0.12)' : 'rgba(245,158,11,0.08)',
              border: `1px solid ${C.amber}44`, fontFamily: 'monospace', textTransform: 'uppercase',
            }}>
              {unmappedCount} Unmapped
            </span>
          </div>
        </div>

        {/* Step 1: Source Node Type & Step 2: Relationship Selector */}
        <div style={{
          display: 'flex', alignItems: 'center', gap: 14, flexWrap: 'wrap',
          padding: '12px 16px', background: C.panelElevated, borderRadius: 10,
          border: `1px solid ${C.border}`,
        }}>
          {/* 1. Source Node Type Selector */}
          <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
            <span style={{ fontSize: 11, fontWeight: 800, color: C.textMuted, letterSpacing: '0.04em', textTransform: 'uppercase' }}>
              1. Source Node:
            </span>
            <select
              value={selectedSourceTypeId}
              onChange={e => {
                setSelectedSourceTypeId(e.target.value);
                setManualTargetSelections({});
                setAiSuggestionsMap(new Map());
              }}
              style={{
                background: C.panel, border: `1px solid ${C.borderStrong}`,
                borderRadius: 6, padding: '7px 12px', color: C.text,
                fontSize: 13, fontWeight: 700, outline: 'none', cursor: 'pointer',
              }}
            >
              {nodeTypes.map((nt: any) => (
                <option key={nt.id} value={nt.id}>
                  {nt.catalog_type_name || nt.name} ({nt.type === 'core' ? 'Core' : 'Custom'})
                </option>
              ))}
            </select>
          </div>

          <ArrowForwardIcon sx={{ color: C.textMuted, fontSize: 16 }} />

          {/* 2. Relationship / Edge Type Selector */}
          <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
            <span style={{ fontSize: 11, fontWeight: 800, color: C.textMuted, letterSpacing: '0.04em', textTransform: 'uppercase' }}>
              2. Relationship:
            </span>
            {availableEdgeTypes.length > 0 ? (
              <select
                value={selectedEdgeTypeId}
                onChange={e => {
                  setSelectedEdgeTypeId(e.target.value);
                  setManualTargetSelections({});
                  setAiSuggestionsMap(new Map());
                }}
                style={{
                  background: C.panel, border: `1px solid ${C.accent}`,
                  borderRadius: 6, padding: '7px 12px', color: C.accent,
                  fontSize: 13, fontWeight: 700, outline: 'none', cursor: 'pointer',
                }}
              >
                {availableEdgeTypes.map((et: EdgeType) => (
                  <option key={et.id} value={et.id}>
                    {et.edge_type_name} → {et.object_node_type_name || 'Target'} ({getEdgeTypeCardinality(et)})
                  </option>
                ))}
              </select>
            ) : (
              <span style={{ fontSize: 12, color: C.textMuted, fontStyle: 'italic' }}>
                No defined relationships for this source node type
              </span>
            )}
          </div>

          {/* Cardinality Badge */}
          {selectedEdgeType && (
            <CardinalityBadge cardinality={cardinality} />
          )}

          {/* Target Node Type Display */}
          {selectedTargetNodeType && (
            <div style={{ display: 'flex', alignItems: 'center', gap: 6, marginLeft: 'auto' }}>
              <span style={{ fontSize: 12, color: C.textMuted }}>Target Node:</span>
              <Badge label={selectedTargetNodeType.catalog_type_name || selectedTargetNodeType.name} color={C.teal} />
            </div>
          )}
        </div>
      </div>

      {/* ──────────────── Toolbar: Filters, Search & AI Actions ──────────────── */}
      <div style={{
        padding: '12px 24px',
        borderBottom: `1px solid ${C.border}`,
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'space-between',
        flexWrap: 'wrap',
        gap: 12,
        background: isDark ? '#0D0F16' : '#F1F5F9',
      }}>
        {/* Filter Pills */}
        <div style={{ display: 'flex', alignItems: 'center', gap: 8, flexWrap: 'wrap' }}>
          <FilterPill
            label="All"
            count={totalCount}
            active={filterTab === 'all'}
            onClick={() => setFilterTab('all')}
            accentColor={C.accent}
            border={C.border}
            textMuted={C.textMuted}
          />
          <FilterPill
            label="Unmapped"
            count={unmappedCount}
            active={filterTab === 'unmapped'}
            onClick={() => setFilterTab('unmapped')}
            accentColor={C.amber}
            border={C.border}
            textMuted={C.textMuted}
          />
          <FilterPill
            label="Composite Clusters"
            count={compositeClustersCount}
            active={filterTab === 'composite_clusters'}
            onClick={() => setFilterTab('composite_clusters')}
            accentColor={C.purple}
            border={C.border}
            textMuted={C.textMuted}
            icon={<AutoAwesomeIcon sx={{ fontSize: 13 }} />}
          />
          <FilterPill
            label="Bloomberg Aligned"
            count={vendorAlignedCount}
            active={filterTab === 'vendor_aligned'}
            onClick={() => setFilterTab('vendor_aligned')}
            accentColor="#F97316"
            border={C.border}
            textMuted={C.textMuted}
            icon={<span style={{ fontSize: 11 }}>🟧</span>}
          />
          <FilterPill
            label="Generic Collisions"
            count={genericCollisionsCount}
            active={filterTab === 'generic_collisions'}
            onClick={() => setFilterTab('generic_collisions')}
            accentColor={C.amber}
            border={C.border}
            textMuted={C.textMuted}
            icon={<WarningIcon sx={{ fontSize: 13 }} />}
          />
          <FilterPill
            label="High Confidence"
            count={highConfidenceCount}
            active={filterTab === 'high_confidence'}
            onClick={() => setFilterTab('high_confidence')}
            accentColor={C.green}
            border={C.border}
            textMuted={C.textMuted}
          />
          <FilterPill
            label="Needs Review"
            count={mappingRows.filter(r => !r.isMapped && (r.suggestion?.confidence ?? 0) >= 50 && (r.suggestion?.confidence ?? 0) < 80).length}
            active={filterTab === 'needs_review'}
            onClick={() => setFilterTab('needs_review')}
            accentColor={C.purple}
            border={C.border}
            textMuted={C.textMuted}
          />
          <FilterPill
            label="See Also Discovered"
            count={mappingRows.filter(r => r.relatedItems.length > 0).length}
            active={filterTab === 'has_see_also'}
            onClick={() => setFilterTab('has_see_also')}
            accentColor={C.blue}
            border={C.border}
            textMuted={C.textMuted}
            icon={<HubIcon sx={{ fontSize: 13 }} />}
          />
          <FilterPill
            label="Mapped"
            count={mappedCount}
            active={filterTab === 'mapped'}
            onClick={() => setFilterTab('mapped')}
            accentColor={C.teal}
            border={C.border}
            textMuted={C.textMuted}
          />
        </div>

        {/* Search, AI Batch & Auto-Map Actions */}
        <div style={{ display: 'flex', alignItems: 'center', gap: 10, flexWrap: 'wrap' }}>
          <div style={{ position: 'relative' }}>
            <SearchIcon sx={{ position: 'absolute', left: 10, top: 8, fontSize: 16, color: C.textMuted }} />
            <input
              placeholder="Filter items..."
              value={searchTerm}
              onChange={e => setSearchTerm(e.target.value)}
              style={{
                background: C.panel, border: `1px solid ${C.borderStrong}`,
                borderRadius: 6, padding: '6px 12px 6px 32px', color: C.text,
                fontSize: 12, outline: 'none', width: 190,
              }}
            />
          </div>

          <button
            onClick={() => {
              refetchSourceNodes();
              refetchTargetNodes();
              refetchEdges();
            }}
            title="Refresh & Re-match"
            style={{
              display: 'flex', alignItems: 'center', gap: 4,
              background: C.panel, border: `1px solid ${C.borderStrong}`,
              color: C.text, padding: '6px 10px', borderRadius: 6,
              cursor: 'pointer', fontSize: 12, fontWeight: 600,
            }}
          >
            <RefreshIcon sx={{ fontSize: 14 }} /> Refresh
          </button>

          {/* Bulk Remediate Generic Columns */}
          {isWriter && genericCollisionsCount > 0 && (
            <button
              onClick={handleRemediateAllGenericTerms}
              style={{
                display: 'flex', alignItems: 'center', gap: 6,
                background: 'rgba(245,158,11,0.15)', color: C.amber,
                border: `1px solid ${C.amber}88`,
                padding: '6px 12px', borderRadius: 6, cursor: 'pointer',
                fontSize: 12, fontWeight: 700, boxShadow: '0 0 12px rgba(245,158,11,0.25)',
              }}
              title="Prefix all generic columns (address, name, status, etc.) with their parent table name"
            >
              <WarningIcon sx={{ fontSize: 15 }} />
              {`Remediate Generic (${genericCollisionsCount})`}
            </button>
          )}

          {/* Gemini AI Batch Suggestion Button */}
          {isWriter && unmappedCount > 0 && (
            <button
              onClick={handleRunGeminiBatch}
              disabled={isAiRunning}
              style={{
                display: 'flex', alignItems: 'center', gap: 6,
                background: 'linear-gradient(135deg, #6366F1 0%, #A855F7 100%)',
                color: '#fff', border: 'none',
                padding: '6px 14px', borderRadius: 6, cursor: 'pointer',
                fontSize: 12, fontWeight: 700, boxShadow: '0 0 16px rgba(168,85,247,0.4)',
                opacity: isAiRunning ? 0.7 : 1,
              }}
            >
              <AutoAwesomeIcon sx={{ fontSize: 15 }} />
              {isAiRunning
                ? `AI Batching (${aiProgress?.processed || 0}/${aiProgress?.total || unmappedCount})...`
                : '✨ AI Batch Suggest (Gemini)'}
            </button>
          )}

          {/* Bulk Auto-Map High Confidence */}
          {isWriter && highConfidenceCount > 0 && (
            <button
              onClick={handleAutoMapAllHighConfidence}
              disabled={isBatchAutoMapping}
              style={{
                display: 'flex', alignItems: 'center', gap: 6,
                background: isBatchAutoMapping ? '#059669' : C.green, color: '#0F172A', border: 'none',
                padding: '6px 14px', borderRadius: 6, cursor: isBatchAutoMapping ? 'not-allowed' : 'pointer',
                fontSize: 12, fontWeight: 700, boxShadow: '0 0 14px rgba(16,185,129,0.4)',
                opacity: isBatchAutoMapping ? 0.85 : 1,
              }}
            >
              <FlashOnIcon sx={{ fontSize: 16, animation: isBatchAutoMapping ? 'pulse 1s infinite' : 'none' }} />
              {isBatchAutoMapping && batchAutoMapProgress
                ? `Auto-Mapping (${batchAutoMapProgress.current}/${batchAutoMapProgress.total})...`
                : `Auto-Map High Conf (${highConfidenceCount})`}
            </button>
          )}
        </div>
      </div>

      {/* ──────────────── Main Mappings List ──────────────── */}
      <div style={{ flex: 1, overflowY: 'auto', padding: '16px 24px' }}>

        {/* Live Batch Auto-Map Progress Banner */}
        {isBatchAutoMapping && batchAutoMapProgress && (
          <div style={{
            background: isDark ? 'linear-gradient(135deg, rgba(16,185,129,0.15) 0%, rgba(6,78,59,0.2) 100%)' : '#ECFDF5',
            border: `1px solid rgba(16,185,129,0.4)`,
            borderRadius: 10, padding: '14px 20px', marginBottom: 16,
            boxShadow: '0 4px 20px rgba(16,185,129,0.15)',
          }}>
            <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginBottom: 8 }}>
              <div style={{ display: 'flex', alignItems: 'center', gap: 10 }}>
                <span style={{ fontSize: 18, animation: 'spin 1.5s linear infinite', display: 'inline-block' }}>⚡</span>
                <div>
                  <span style={{ fontWeight: 800, fontSize: 13, color: isDark ? '#34D399' : '#065F46' }}>
                    Auto-Mapping Batch in Progress...
                  </span>
                  {batchAutoMapProgress.currentItem && (
                    <span style={{ marginLeft: 10, fontSize: 12, color: C.textMuted, fontFamily: 'monospace' }}>
                      Processing: <strong>{batchAutoMapProgress.currentItem}</strong>
                    </span>
                  )}
                </div>
              </div>
              <span style={{ fontWeight: 800, fontSize: 13, color: isDark ? '#34D399' : '#065F46', fontFamily: 'monospace' }}>
                {batchAutoMapProgress.current} / {batchAutoMapProgress.total} ({Math.round((batchAutoMapProgress.current / batchAutoMapProgress.total) * 100)}%)
              </span>
            </div>
            {/* Progress Bar Track */}
            <div style={{ width: '100%', height: 8, background: 'rgba(0,0,0,0.2)', borderRadius: 4, overflow: 'hidden' }}>
              <div style={{
                height: '100%',
                width: `${Math.max(5, (batchAutoMapProgress.current / batchAutoMapProgress.total) * 100)}%`,
                background: 'linear-gradient(90deg, #10B981 0%, #34D399 100%)',
                borderRadius: 4,
                transition: 'width 0.2s ease-in-out',
                boxShadow: '0 0 10px rgba(52,211,153,0.5)',
              }} />
            </div>
          </div>
        )}

        {/* Composite Clusters Discovery Callout Banner */}
        {compositeClusters.length > 0 && (
          <div style={{
            background: isDark ? 'linear-gradient(135deg, rgba(99,102,241,0.12) 0%, rgba(168,85,247,0.08) 100%)' : '#EEF2FF',
            border: `1px solid rgba(99,102,241,0.3)`,
            borderRadius: 10, padding: '12px 18px', marginBottom: 16,
            display: 'flex', alignItems: 'center', justifyContent: 'space-between', flexWrap: 'wrap', gap: 12,
          }}>
            <div style={{ display: 'flex', alignItems: 'center', gap: 10 }}>
              <div style={{
                width: 32, height: 32, borderRadius: 6,
                background: 'rgba(99,102,241,0.2)', color: C.accent,
                display: 'flex', alignItems: 'center', justifyContent: 'center',
              }}>
                <AutoAwesomeIcon sx={{ fontSize: 18 }} />
              </div>
              <div>
                <div style={{ fontWeight: 800, fontSize: 13, color: C.text }}>
                  Composite Ontology Clusters Detected ({compositeClusters.length} Bundles)
                </div>
                <div style={{ fontSize: 11, color: C.textMuted }}>
                  Multi-column semantic structures discovered conforming to global standards (ISO 19160-1 / ISO 4217 / FIBO / W3C)
                </div>
              </div>
            </div>

            {isWriter && (
              <div style={{ display: 'flex', gap: 8, flexWrap: 'wrap' }}>
                {compositeClusters.map(cluster => (
                  <button
                    key={cluster.clusterId}
                    onClick={() => handleMapCompositeCluster(cluster)}
                    style={{
                      display: 'flex', alignItems: 'center', gap: 6,
                      background: 'rgba(99,102,241,0.2)', border: `1px solid ${C.accent}`,
                      color: C.accent, borderRadius: 6, padding: '5px 12px',
                      fontSize: 11, fontWeight: 700, cursor: 'pointer',
                    }}
                  >
                    ⚡ Auto-Map {cluster.compositeTermName} ({cluster.members.length} cols · {cluster.standard.split('/')[0].trim()})
                  </button>
                ))}
              </div>
            )}
          </div>
        )}

        {isDataLoading ? (
          <Spinner accent={C.accent} />
        ) : filteredRows.length === 0 ? (
          <div style={{ textAlign: 'center', padding: 64, color: C.textMuted }}>
            <div style={{ fontSize: 44, opacity: 0.3, marginBottom: 12 }}>🔍</div>
            <div style={{ fontSize: 16, fontWeight: 600, color: C.text }}>No items match current filters</div>
            <div style={{ fontSize: 13, marginTop: 4 }}>
              Try adjusting your search or selecting a different filter tab above.
            </div>
          </div>
        ) : (
          <div style={{ display: 'flex', flexDirection: 'column', gap: 12 }}>
            {filteredRows.map(row => {
              const isMapped = row.isMapped;
              const suggestion = row.suggestion;
              const selectedTarget = row.selectedTargetNode;
              const isSeeAlsoExpanded = expandedSeeAlso[row.id] || false;
              const hasSeeAlso = row.relatedItems.length > 0;
              const isNewDraft = suggestion?.targetDraft?.isNew && selectedTarget?.id?.startsWith('draft-');

              return (
                <div
                  key={row.id}
                  style={{
                    background: C.panel,
                    border: `1px solid ${
                      row.cardinalityConflict
                        ? 'rgba(239,68,68,0.5)'
                        : isMapped
                        ? 'rgba(16,185,129,0.3)'
                        : row.isModified
                        ? C.accent
                        : C.border
                    }`,
                    borderRadius: 10,
                    padding: '14px 18px',
                    display: 'flex',
                    flexDirection: 'column',
                    gap: 10,
                    transition: 'all 0.15s ease',
                    boxShadow: isMapped ? '0 2px 10px rgba(0,0,0,0.15)' : 'none',
                  }}
                >
                  {/* Main Row Content */}
                  <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', gap: 16 }}>
                    
                    {/* Left: Source Node Info */}
                    <div style={{ flex: 1, minWidth: 260, maxWidth: 420 }}>
                      <div style={{ display: 'flex', alignItems: 'center', gap: 6, flexWrap: 'wrap', marginBottom: 4 }}>
                        <span style={{ fontSize: 14 }}>
                          {selectedSourceNodeType?.catalog_type_name === 'column' ? '🔹' : '💼'}
                        </span>
                        <span style={{ fontWeight: 700, fontSize: 14, color: C.text, wordBreak: 'break-word' }}>
                          {row.sourceNode.node_name}
                        </span>
                        {row.sourceNode.properties?.data_type && (
                          <Badge label={row.sourceNode.properties.data_type} color={C.blue} />
                        )}
                        {/* Governance Tier Badge */}
                        <Badge
                          label={row.governanceTier === 'gold_certified' ? '🟢 Gold Core' : row.governanceTier === 'custom' ? '🔵 Custom' : '🟡 Draft'}
                          color={row.governanceTier === 'gold_certified' ? C.green : row.governanceTier === 'custom' ? C.blue : C.amber}
                        />
                        {/* Universal Standard Pill */}
                        {row.universalParentName && (
                          <Badge label={`📐 ${row.universalParentName}`} color={C.teal} />
                        )}
                        {/* Composite Cluster Badge */}
                        {row.compositeCluster && (
                          <Badge label={`📦 ${row.compositeCluster.clusterType}`} color={C.purple} />
                        )}
                        {/* Bloomberg Financial Data Dictionary Badge */}
                        {row.vendorAlignment && (
                          <Tooltip title={`Bloomberg Data License (${row.vendorAlignment.category}): ${row.vendorAlignment.mnemonic} — ${row.vendorAlignment.description}`}>
                            <span style={{
                              display: 'inline-flex', alignItems: 'center', gap: 4,
                              padding: '2px 8px', borderRadius: 4,
                              background: 'rgba(249,115,22,0.15)', border: '1px solid rgba(249,115,22,0.5)',
                              color: '#F97316', fontSize: 10, fontWeight: 800, fontFamily: 'monospace',
                              cursor: 'help',
                            }}>
                              🟧 BBG: {row.vendorAlignment.mnemonic}
                            </span>
                          </Tooltip>
                        )}
                      </div>
                      {row.sourceNode.qualified_path && (
                        <div style={{ fontSize: 11, color: C.textMuted, fontFamily: 'monospace', wordBreak: 'break-all' }}>
                          {row.sourceNode.qualified_path}
                        </div>
                      )}
                      {row.sourceNode.description && (
                        <div style={{ fontSize: 12, color: C.textMuted, marginTop: 4, lineHeight: 1.3 }}>
                          {row.sourceNode.description}
                        </div>
                      )}

                      {/* Safe Physical Page Sample Profiling (500 rows) */}
                      {sampleProfiles[row.sourceNode.id] ? (
                        <div style={{
                          display: 'flex', alignItems: 'center', gap: 6, flexWrap: 'wrap',
                          padding: '4px 8px', borderRadius: 6, marginTop: 6,
                          background: isDark ? 'rgba(59,130,246,0.1)' : 'rgba(59,130,246,0.06)',
                          border: '1px solid rgba(59,130,246,0.25)',
                          fontSize: 11, color: C.text,
                        }}>
                          <span style={{ fontWeight: 700, color: C.blue }}>📊 Sample (500 rows):</span>
                          <span style={{ fontFamily: 'monospace', color: C.textMuted, fontSize: 10 }}>
                            [{sampleProfiles[row.sourceNode.id].sampleValues.slice(0, 3).map(v => `"${v}"`).join(', ')}]
                          </span>
                          <Badge label={`✓ ${sampleProfiles[row.sourceNode.id].pattern}`} color={C.green} />
                          {sampleProfiles[row.sourceNode.id].bloombergCandidate && (
                            <Badge label={`🟧 ${sampleProfiles[row.sourceNode.id].bloombergCandidate}`} color="#F97316" />
                          )}
                        </div>
                      ) : (
                        <div style={{ marginTop: 4 }}>
                          <button
                            onClick={() => handleProfileColumn(row.sourceNode)}
                            disabled={loadingProfiles.has(row.sourceNode.id)}
                            style={{
                              background: 'transparent', border: `1px dashed ${C.borderStrong}`,
                              color: C.textMuted, borderRadius: 4, padding: '2px 7px',
                              fontSize: 10, cursor: 'pointer', display: 'inline-flex', alignItems: 'center', gap: 4,
                            }}
                            title="Perform safe, zero-impact physical page sampling (500 rows) to verify data patterns and checksums"
                          >
                            <span>⚡</span>
                            {loadingProfiles.has(row.sourceNode.id) ? 'Sampling 500 rows...' : 'Sample 500 rows (zero DB impact)'}
                          </button>
                        </div>
                      )}
                    </div>

                    {/* Center: Relationship Indicator & Cardinality */}
                    <div style={{ display: 'flex', flexDirection: 'column', alignItems: 'center', gap: 5, flexShrink: 0 }}>
                      <div style={{
                        display: 'flex', alignItems: 'center', gap: 6,
                        padding: '4px 10px', borderRadius: 9999,
                        background: isDark ? 'rgba(99,102,241,0.12)' : 'rgba(99,102,241,0.08)',
                        border: `1px solid rgba(99,102,241,0.3)`,
                        color: C.accent, fontSize: 11, fontWeight: 700, fontFamily: 'monospace',
                      }}>
                        <span>{selectedEdgeType?.edge_type_name || 'maps_to'}</span>
                        <ArrowForwardIcon sx={{ fontSize: 13 }} />
                      </div>

                      {suggestion && !isMapped && (
                        <ConfidencePill
                          confidence={suggestion.confidence}
                          reason={suggestion.matchReason}
                          isAi={suggestion.matchType === 'gemini_ai'}
                        />
                      )}

                      {row.cardinalityConflict && (
                        <div style={{ display: 'flex', alignItems: 'center', gap: 4, color: C.red, fontSize: 10, fontWeight: 700 }}>
                          <WarningIcon sx={{ fontSize: 12 }} /> {row.cardinalityConflict}
                        </div>
                      )}
                    </div>

                    {/* Right: Target Selection & Actions */}
                    <div style={{ flex: 1, minWidth: 340, display: 'flex', alignItems: 'center', justifyContent: 'flex-end', gap: 12 }}>
                      {isMapped ? (
                        /* Currently Mapped State */
                        <div style={{ display: 'flex', alignItems: 'center', gap: 10 }}>
                          <div style={{
                            display: 'flex', alignItems: 'center', gap: 8,
                            padding: '8px 14px', borderRadius: 8,
                            background: isDark ? 'rgba(16,185,129,0.12)' : 'rgba(16,185,129,0.08)',
                            border: `1px solid ${C.green}44`,
                          }}>
                            <span style={{ fontSize: 14 }}>🧠</span>
                            <div>
                              <div style={{ fontWeight: 700, fontSize: 13, color: C.green }}>
                                {row.currentTargetNode?.node_name || 'Mapped'}
                              </div>
                              <div style={{ fontSize: 10, color: C.textMuted }}>Connected in Catalog Graph</div>
                            </div>
                          </div>

                          {isWriter && row.governanceTier !== 'gold_certified' && (
                            <Tooltip title="Promote this custom term to Certified Gold Copy Core (Master Template)">
                              <button
                                onClick={() => handlePromoteToGoldCore(row)}
                                style={{
                                  background: 'rgba(16,185,129,0.1)', border: `1px solid ${C.green}88`,
                                  color: C.green, padding: '6px 10px', borderRadius: 6,
                                  cursor: 'pointer', display: 'flex', alignItems: 'center', gap: 4,
                                  fontSize: 11, fontWeight: 700,
                                }}
                              >
                                🏆 Promote
                              </button>
                            </Tooltip>
                          )}

                          {isWriter && (
                            <Tooltip title="Unlink / Remove relationship">
                              <button
                                onClick={() => handleUnlink(row)}
                                disabled={row.isSaving}
                                style={{
                                  background: 'transparent', border: `1px solid ${C.red}44`,
                                  color: C.red, padding: '6px 10px', borderRadius: 6,
                                  cursor: 'pointer', display: 'flex', alignItems: 'center', gap: 4,
                                  fontSize: 12, fontWeight: 600,
                                }}
                              >
                                <LinkOffIcon sx={{ fontSize: 15 }} /> Unlink
                              </button>
                            </Tooltip>
                          )}
                        </div>
                      ) : (
                        /* Unmapped / Suggestion State */
                        <div style={{ display: 'flex', alignItems: 'center', gap: 10, width: '100%', justifyContent: 'flex-end' }}>
                          {/* Target Selection Dropdown */}
                          <div style={{ display: 'flex', flexDirection: 'column', gap: 4, minWidth: 240, maxWidth: 300 }}>
                            <select
                              value={selectedTarget?.id || ''}
                              onChange={e => {
                                const val = e.target.value;
                                if (val === suggestion?.targetNode.id) {
                                  setManualTargetSelections(prev => ({
                                    ...prev,
                                    [row.sourceNode.id]: suggestion.targetNode,
                                  }));
                                } else {
                                  const chosen = targetNodesMap.get(val) || null;
                                  setManualTargetSelections(prev => ({
                                    ...prev,
                                    [row.sourceNode.id]: chosen,
                                  }));
                                }
                              }}
                              style={{
                                background: C.panelElevated,
                                border: `1px solid ${row.cardinalityConflict ? C.red : selectedTarget ? C.accent : C.borderStrong}`,
                                borderRadius: 6, padding: '7px 10px', color: C.text,
                                fontSize: 12, fontWeight: 600, outline: 'none', width: '100%',
                              }}
                            >
                              <option value="">-- Select Target --</option>
                              {/* Top Suggestion (Existing or New Draft) */}
                              {suggestion && (
                                <option value={suggestion.targetNode.id}>
                                  {isNewDraft ? '✨ [CREATE NEW] ' : '⭐ '}{suggestion.targetNode.node_name} ({suggestion.confidence}% match)
                                </option>
                              )}
                              {/* Alternative Suggestions */}
                              {row.alternativeSuggestions.map((alt, aIdx) => (
                                <option key={`alt-${aIdx}-${alt.targetNode.id}`} value={alt.targetNode.id}>
                                  {alt.targetDraft?.isNew ? '✨ [NEW] ' : ''}{alt.targetNode.node_name} ({alt.confidence}% match)
                                </option>
                              ))}
                              {/* All Catalog Targets */}
                              <option disabled>──────────</option>
                              {targetNodes.map(tn => (
                                <option key={tn.id} value={tn.id}>
                                  {tn.node_name}
                                </option>
                              ))}
                            </select>

                            {suggestion && (
                              <div style={{ fontSize: 10, color: C.textMuted, paddingLeft: 2, display: 'flex', alignItems: 'center', gap: 4 }}>
                                <span>💡</span>
                                <span style={{ overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
                                  {suggestion.matchReason}
                                </span>
                              </div>
                            )}
                          </div>

                          {/* Accept Button */}
                          {isWriter && selectedTarget && !row.cardinalityConflict && (
                            <button
                              onClick={() => handleAcceptMapping(row, selectedTarget)}
                              disabled={row.isSaving}
                              style={{
                                display: 'flex', alignItems: 'center', gap: 4,
                                background: isNewDraft ? C.teal : C.accent,
                                color: '#0F172A',
                                border: 'none', padding: '8px 14px', borderRadius: 6,
                                cursor: 'pointer', fontSize: 12, fontWeight: 700,
                                boxShadow: isNewDraft ? '0 0 14px rgba(45,212,191,0.4)' : C.accentGlow,
                                flexShrink: 0,
                                opacity: row.isSaving ? 0.7 : 1,
                              }}
                            >
                              {isNewDraft ? <AddIcon sx={{ fontSize: 15 }} /> : <CheckIcon sx={{ fontSize: 15 }} />}
                              {row.isSaving ? 'Saving...' : isNewDraft ? 'Create & Map' : 'Accept'}
                            </button>
                          )}

                          {/* Reject Suggestion Button */}
                          {isWriter && suggestion && (
                            <Tooltip title="Reject suggestion permanently (never suggest again)">
                              <button
                                onClick={() => handleRejectSuggestion(row)}
                                style={{
                                  background: 'transparent', border: `1px solid ${C.borderStrong}`,
                                  color: C.textMuted, padding: '7px 9px', borderRadius: 6,
                                  cursor: 'pointer', display: 'flex', alignItems: 'center',
                                }}
                              >
                                <CloseIcon sx={{ fontSize: 14 }} />
                              </button>
                            </Tooltip>
                          )}
                        </div>
                      )}
                    </div>
                  </div>

                  {/* Contextual Disambiguation Alert & Remediation Callout */}
                  {!isMapped && row.isGenericCollision && row.suggestedContextualTerm && (
                    <div style={{
                      display: 'flex', alignItems: 'center', justifyContent: 'space-between', flexWrap: 'wrap', gap: 8,
                      padding: '8px 12px', borderRadius: 8,
                      background: isDark ? 'rgba(245,158,11,0.08)' : 'rgba(245,158,11,0.05)',
                      border: `1px solid rgba(245,158,11,0.25)`,
                      fontSize: 11,
                    }}>
                      <div style={{ display: 'flex', alignItems: 'center', gap: 6, color: C.amber }}>
                        <WarningIcon sx={{ fontSize: 14, color: C.amber }} />
                        <span>
                          <strong>Generic Column Context:</strong> "{row.sourceNode.node_name}" in table "{row.contextualEntity || 'table'}" — Recommend contextual term <strong style={{ color: C.accent }}>{row.suggestedContextualTerm}</strong> to prevent cross-table collisions.
                        </span>
                      </div>
                      {isWriter && selectedTarget?.node_name !== row.suggestedContextualTerm && (
                        <button
                          onClick={() => handleUseContextualTerm(row)}
                          style={{
                            display: 'flex', alignItems: 'center', gap: 4,
                            background: C.amber, color: '#0F172A', border: 'none',
                            borderRadius: 5, padding: '4px 10px', fontSize: 11, fontWeight: 700,
                            cursor: 'pointer', boxShadow: '0 0 10px rgba(245,158,11,0.3)',
                          }}
                        >
                          ⚡ Switch to Contextual Term ({row.suggestedContextualTerm})
                        </button>
                      )}
                    </div>
                  )}

                  {/* "See Also" / Related Items Discovery Sub-Panel */}
                  {hasSeeAlso && (
                    <div style={{ borderTop: `1px dashed ${C.border}`, paddingTop: 8, marginTop: 4 }}>
                      <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
                        <div style={{ display: 'flex', alignItems: 'center', gap: 6, fontSize: 11, color: C.blue, fontWeight: 700 }}>
                          <HubIcon sx={{ fontSize: 14 }} />
                          <span>Related "See Also" Identifiers Discovered ({row.relatedItems.length}):</span>
                        </div>
                        <button
                          onClick={() => setExpandedSeeAlso(prev => ({ ...prev, [row.id]: !isSeeAlsoExpanded }))}
                          style={{ background: 'transparent', border: 'none', color: C.textMuted, cursor: 'pointer', fontSize: 11, display: 'flex', alignItems: 'center', gap: 2 }}
                        >
                          {isSeeAlsoExpanded ? 'Hide' : 'Show'} {isSeeAlsoExpanded ? <ExpandLessIcon sx={{ fontSize: 14 }} /> : <ExpandMoreIcon sx={{ fontSize: 14 }} />}
                        </button>
                      </div>

                      <Collapse in={isSeeAlsoExpanded}>
                        <div style={{ display: 'flex', gap: 8, flexWrap: 'wrap', marginTop: 8 }}>
                          {row.relatedItems.map(peer => (
                            <div
                              key={peer.id}
                              style={{
                                display: 'flex', alignItems: 'center', gap: 8,
                                padding: '4px 10px', borderRadius: 6,
                                background: isDark ? 'rgba(96,165,250,0.1)' : 'rgba(96,165,250,0.06)',
                                border: `1px solid rgba(96,165,250,0.3)`, fontSize: 12,
                              }}
                            >
                              <span style={{ fontWeight: 700, color: C.blue }}>{peer.node_name}</span>
                              {peer.isAlreadyLinked ? (
                                <Badge label="Linked" color={C.green} />
                              ) : (
                                isWriter && (
                                  <button
                                    onClick={() => handleLinkSeeAlso(row.sourceNode, peer)}
                                    style={{
                                      background: 'rgba(96,165,250,0.2)', border: `1px solid ${C.blue}`,
                                      color: C.blue, borderRadius: 4, padding: '2px 6px',
                                      fontSize: 10, fontWeight: 700, cursor: 'pointer',
                                    }}
                                  >
                                    + Link See-Also
                                  </button>
                                )
                              )}
                            </div>
                          ))}
                        </div>
                      </Collapse>
                    </div>
                  )}
                </div>
              );
            })}
          </div>
        )}
      </div>
    </div>
  );
}
