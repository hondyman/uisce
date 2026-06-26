import React, { useState } from 'react';
import {
  Box,
  Typography,
  Card,
  CardContent,
  Tabs,
  Tab,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  Paper,
  Chip,
  Divider,
  Button,
  IconButton,
  Tooltip,
  CircularProgress,
  ToggleButton,
  ToggleButtonGroup
} from '@mui/material';
import {
  Description as DescriptionIcon,
  AccountTree as LineageIcon,
  Label as LabelIcon,
  CheckCircle as MappedIcon,
  RadioButtonUnchecked as UnmappedIcon,
  AddLink as AddLinkIcon,
  Delete as DeleteIcon,
  Edit as EditIcon,
  CompareArrows as RelationshipIcon,
  Hub as GraphIcon,
  Chat as ChatIcon,
  ViewSidebar as SidebarIcon,
  Close as CloseIcon,
  ArrowUpward as ArrowUpwardIcon,
  ArrowDownward as ArrowDownwardIcon,
  SwapVert as SwapVertIcon,
} from '@mui/icons-material';

const TextIcon = DescriptionIcon;
import { EnhancedSelectedAsset } from '../../../types/SemanticTypes';
import AddEdgeDialog from '../../../components/AddEdgeDialog';
import EditSemanticTermDialog from '../../../components/EditSemanticTermDialog';
import { useNodeTypes } from '../../../api/nodeTypes';
import { apiFetch, getTenantHeaders } from '../../../lib/apiClient';

// Use standard require/import for DualLineageViewer to allow it to be constrained
// We'll wrap it in a container that forces height
import DualLineageViewer from '../Catalog/DualLineageViewer';
import { devDebug } from '../../../utils/devLogger';
import { Gavel as RuleIcon } from '@mui/icons-material';
import ValidationRuleEditor from '../../../components/validation/ValidationRuleEditor';
import { UnifiedLineageTab } from '../../../features/impact-analysis/components/UnifiedLineageTab';
import { ImpactExplanation } from '../../../features/impact-analysis/components/ImpactExplanation';
import { ImpactQA } from '../../../features/impact-analysis/components/ImpactQA';
import { formatEdgeLabel } from '../Catalog/lineageUtils';

interface SemanticTermDetailsProps {
  asset: EnhancedSelectedAsset;
  semanticData: any;
  technicalData: any;
  allEdges?: any[]; // Pass edges to find generic relationships
  allNodes?: any[];
  datasourceId?: string;
  onRefresh?: () => void;
  onAssetSelect?: (asset: EnhancedSelectedAsset) => void;
}

interface TermData {
  id: string;
  node_name: string;
  description?: string;
  properties?: any;
}

interface TabPanelProps {
  children?: React.ReactNode;
  index: number;
  value: number;
}

function CustomTabPanel(props: TabPanelProps) {
  const { children, value, index, ...other } = props;

  return (
    <div
      role="tabpanel"
      hidden={value !== index}
      id={`semantic-tabpanel-${index}`}
      aria-labelledby={`semantic-tab-${index}`}
      style={{ height: '100%', display: value === index ? 'block' : 'none' }}
      {...other}
    >
      {value === index && (
        <Box sx={{ p: 0, height: '100%' }}>
          {children}
        </Box>
      )}
    </div>
  );
}

const SemanticTermDetails: React.FC<SemanticTermDetailsProps> = ({
  asset,
  semanticData,
  technicalData,
  allEdges = [],
  allNodes = [],
  datasourceId,
  onRefresh,
  onAssetSelect
}) => {
  const [tabValue, setTabValue] = useState(0);
  const [isSemanticFullScreen, setIsSemanticFullScreen] = useState(false);
  const [addEdgeDialogOpen, setAddEdgeDialogOpen] = useState(false);
  const [editTermDialogOpen, setEditTermDialogOpen] = useState(false);
  const [dynamicLineage, setDynamicLineage] = useState<any>(null);
  const [refreshTrigger, setRefreshTrigger] = useState(0);
  const [termData, setTermData] = useState<TermData | null>(null);
  const [lineageType, setLineageType] = useState<'sql' | 'cypher'>('sql');
  
  // Sidebar and AI Assistant state
  const [sidebarOpen, setSidebarOpen] = useState(false);
  const [sidebarMode, setSidebarMode] = useState<'explanation' | 'assistant'>('explanation');
  const [highlightedNodeIds, setHighlightedNodeIds] = useState<string[]>([]);
  const [directionMode, setDirectionMode] = useState<'upstream' | 'downstream' | 'both'>('both');
  
  // Fetch node types for resolving node_type_id to name
  const tenantId = asset.node?.tenant_id || 'default';
  const { data: nodeTypes } = useNodeTypes(tenantId);

  // Fetch term data from API
  const fetchTermData = React.useCallback(async () => {
    if (!asset?.nodeId) return;
    
    try {
      const response = await apiFetch(`/api/glossary/terms/${asset.nodeId}`);
      
      if (response.ok) {
        const data = await response.json();
        setTermData(data);
      }
    } catch (error) {
      console.error('[SemanticTermDetails] Failed to fetch term data:', error);
    }
  }, [asset?.nodeId]);

  // Callback to refresh lineage data
  const refreshLineage = React.useCallback(() => {
    setRefreshTrigger(prev => prev + 1);
    // Also trigger parent refresh to reload term data
    if (onRefresh) {
      onRefresh();
    }
    // Refetch term data to update UI
    fetchTermData();
  }, [onRefresh, fetchTermData]);

  React.useEffect(() => {
    if (asset?.nodeId) {
      console.log('[SemanticTermDetails] Fetching lineage for nodeId:', asset.nodeId);
      const fetchLineage = async () => {
        try {
          const url = `/api/lineage/node/${asset.nodeId}/graph?depth=3`;
          console.log('[SemanticTermDetails] Calling:', url);
          const res = await apiFetch(url);
          console.log('[SemanticTermDetails] Response status:', res.status);
          if (res.ok) {
            const data = await res.json();
            
            // Build adjacency and reverse adjacency maps for bi-directional traversal
            const adjacency: Record<string, string[]> = {};
            const reverseAdjacency: Record<string, string[]> = {};
            
            (data.edges || []).forEach((e: any) => {
              const src = e.from_id || e.source;
              const dst = e.to_id || e.target;
              if (src && dst) {
                if (!adjacency[src]) adjacency[src] = [];
                adjacency[src].push(dst);
                
                if (!reverseAdjacency[dst]) reverseAdjacency[dst] = [];
                reverseAdjacency[dst].push(src);
              }
            });

            // Bi-directional BFS to determine depths relative to center node
            const depths: Record<string, number> = {};
            const queue = [{ id: asset.nodeId, depth: 0 }];
            const visited = new Set<string>([asset.nodeId]);
            depths[asset.nodeId] = 0;

            while (queue.length > 0) {
              const { id, depth } = queue.shift()!;
              
              // Traverse downstream
              if (adjacency[id]) {
                adjacency[id].forEach(childId => {
                  if (!visited.has(childId)) {
                    visited.add(childId);
                    depths[childId] = depth + 1;
                    queue.push({ id: childId, depth: depth + 1 });
                  }
                });
              }
              
              // Traverse upstream
              if (reverseAdjacency[id]) {
                reverseAdjacency[id].forEach(parentId => {
                  if (!visited.has(parentId)) {
                    visited.add(parentId);
                    depths[parentId] = depth - 1;
                    queue.push({ id: parentId, depth: depth - 1 });
                  }
                });
              }
            }

            const nodesAtDepth: Record<number, number> = {};

            const nodes = (data.nodes || []).map((n: any) => {
              const meta = typeof n.metadata === 'string' ? JSON.parse(n.metadata) : (n.metadata || {});
              const depth = depths[n.id] ?? 0;
              const indexAtDepth = nodesAtDepth[depth] || 0;
              nodesAtDepth[depth] = indexAtDepth + 1;

              const label = n.name || n.label || meta.label || n.properties?.node_name || n.id;
              
              return {
                id: n.id,
                type: 'hoverableNode',
                position: { x: depth * 300, y: indexAtDepth * 150 },
                data: {
                  label,
                  nodeType: n.type,
                  description: meta.description || n.properties?.description,
                  properties: n.properties || meta,
                  parent_name: meta.parent_name || n.properties?.parent_name,
                  node_type_id: meta.node_type_id || n.properties?.node_type_id,
                  isCore: meta.is_core === 'true' || meta.is_core === true || meta.isCore === true,
                  qualifiedPath: meta.qualified_path || meta.qualifiedPath || n.properties?.qualified_path || n.properties?.qualifiedPath,
                  isCenter: n.id === asset.nodeId
                }
              };
            });

            // Create a map to resolve numeric IDs to UUIDs if nodes use UUIDs but edges use numeric IDs
            const idMap = new Map<string, string>();
            (data.nodes || []).forEach((n: any) => {
              // Assume n.id is the canonical ID (likely UUID from observed behavior)
              const canonicalId = n.id;
              
              // Try to find the numeric ID that might be used in edges
              const meta = typeof n.metadata === 'string' ? JSON.parse(n.metadata) : (n.metadata || {});
              const props = n.properties || {};
              
              // Check various properties where the internal graph ID might be stored
              const potentialIds = [
                meta.id, 
                meta.graph_id, 
                meta.node_id,
                props.id, 
                props.graph_id,
                props.node_id
              ];
              
              potentialIds.forEach(pid => {
                if (pid) idMap.set(String(pid), canonicalId);
              });
              
              // Also map the canonical ID to itself just in case
              idMap.set(String(canonicalId), canonicalId);
            });

            const edges = (data.edges || []).map((e: any) => {
              const rawSource = e.from_id || e.source;
              const rawTarget = e.to_id || e.target;
              
              // Resolve IDs using the map, falling back to raw values
              const source = idMap.get(String(rawSource)) || rawSource;
              const target = idMap.get(String(rawTarget)) || rawTarget;

              const edge = {
                id: e.id || `${source}-${target}`,
                source,
                target,
                label: e.type || e.label || e.edge_type_name,
                type: 'smoothstep', // Ensure uniform edge type that supports styling
                animated: false,
                style: { 
                  strokeWidth: 2,
                  stroke: '#64748b' // Slate gray color for edges
                },
                markerEnd: {
                  type: 'arrowclosed',
                  width: 20,
                  height: 20,
                  color: '#64748b'
                },
                labelStyle: { 
                  fill: '#475569',
                  fontWeight: 500,
                  fontSize: 12
                },
                labelBgStyle: { 
                  fill: '#ffffff',
                  fillOpacity: 0.9
                },
                labelBgPadding: [4, 4],
                labelBgBorderRadius: 4,
                data: {
                  relationship_type: e.type || e.label || e.edge_type_name,
                  properties: e.metadata ? (typeof e.metadata === 'string' ? JSON.parse(e.metadata) : e.metadata) : {}
                }
              };
              return edge;
            });
            
            console.log('[SemanticTermDetails] Constructed edges:', edges.length, edges[0]);
            console.log('[SemanticTermDetails] Constructed nodes:', nodes.length, nodes[0]);

            setDynamicLineage({ nodes, edges });
          } else {
            console.error('[SemanticTermDetails] Lineage fetch failed:', res.status, await res.text());
          }
        } catch (error) {
          console.error('[SemanticTermDetails] Error fetching lineage:', error);
        }
      };
      fetchLineage();
    } else {
      console.log('[SemanticTermDetails] No nodeId, skipping lineage fetch');
    }
  }, [asset?.nodeId, lineageType, refreshTrigger]);

  const handleTabChange = (_event: React.SyntheticEvent, newValue: number) => {
    setTabValue(newValue);
  };

  const properties = asset.node?.properties || {};
  const isMapped = properties.mapped === true;

  // Filter edges for this term
  // IMPORTANT: Always use allEdges (static props) because dynamicLineage.edges use ReactFlow-generated
  // numeric IDs for source/target, while we need to match against the original database UUIDs
  const termEdges = React.useMemo(() => {
    return allEdges.filter(e => 
      e.source_node_id === asset.nodeId || e.target_node_id === asset.nodeId ||
      e.source === asset.nodeId || e.target === asset.nodeId ||
      e.from_id === asset.nodeId || e.to_id === asset.nodeId
    );
  }, [allEdges, asset.nodeId]);

  // Visible console log to confirm component/render data in dev builds
  React.useEffect(() => {
    console.info('[SemanticTermDetails] render snapshot', {
      tabValue,
      termEdgesCount: termEdges?.length,
      dynamicLineageNodes: dynamicLineage?.nodes?.length,
      dynamicLineageEdges: dynamicLineage?.edges?.length,
      allNodesCount: allNodes?.length,
      assetId: asset?.nodeId,
    });
  });

  const handleDeleteEdge = async (edgeId: string) => {
    if (!window.confirm("Are you sure you want to delete this relationship?")) return;
    try {
      const response = await fetch(`/api/glossary/edges/${edgeId}`, {
        method: 'DELETE',
        headers: {
          'X-Tenant-ID': asset.node?.tenant_id || 'default', 
          'X-Tenant-Datasource-ID': datasourceId || asset.node?.tenant_datasource_id || ''
        }
      });
      if (response.ok) {
        refreshLineage();
        if (onRefresh) {
            onRefresh();
        }
      } else {
        alert("Failed to delete relationship");
      }
    } catch (e) {
      console.error(e);
      alert("Error deleting relationship");
    }
  };

  const handleDeleteTerm = async () => {
    if (!window.confirm(`Are you sure you want to delete term "${asset.name}"?`)) return;
    try {
        const tenantId = asset.node?.tenant_id || 'default';
        const dsId = datasourceId || asset.node?.tenant_datasource_id || '';
        
        const response = await fetch(`/api/glossary/terms/${asset.nodeId}?tenant_id=${tenantId}&datasource_id=${dsId}`, {
            method: 'DELETE',
            headers: {
                'X-Tenant-ID': tenantId,
                'X-Tenant-Datasource-ID': dsId
            }
        });
        if (response.ok) {
            refreshLineage();
            if (onRefresh) {
                onRefresh();
            }
        } else {
            alert("Failed to delete term");
        }
    } catch (e) {
        console.error(e);
        alert("Error deleting term");
    }
  };

  // Helper to get node name
  const getNodeName = (id: string) => {
    // Try dynamic nodes first
    if (dynamicLineage && dynamicLineage.nodes) {
        const node = dynamicLineage.nodes.find((n: any) => n.id === id);
        if (node) return node.data?.label || node.name || node.id;
    }
    // Fallback to static props
    const node = allNodes.find(n => n.id === id);
    return node ? (node.node_name || node.name || node.data?.label || id) : id;
  };
  
  // Helper to get node type name from node_type_id
  const getNodeTypeName = (nodeTypeId: string | undefined) => {
    if (!nodeTypeId) return 'Unknown';
    
    // Combine types from hook and passed semanticData
    const allAvailableTypes = [
      ...(nodeTypes || []),
      ...(semanticData?.node_types || [])
    ];
    
    // Debug logging for node type resolution
    if (allAvailableTypes.length > 0) {
      const found = allAvailableTypes.find((nt: any) => nt.id === nodeTypeId);
      if (!found) {
        console.warn(`[SemanticTermDetails] Node type ID ${nodeTypeId} not found in ${allAvailableTypes.length} types`);
        // Log first few types to see structure
        console.log('[SemanticTermDetails] Available types sample:', allAvailableTypes.slice(0, 3));
      } else {
        console.log(`[SemanticTermDetails] Found node type for ${nodeTypeId}:`, found.catalog_type_name);
      }
    }
    
    if (allAvailableTypes.length === 0) return nodeTypeId;
    
    const nodeType = allAvailableTypes.find((nt: any) => nt.id === nodeTypeId);
    return nodeType?.catalog_type_name || nodeTypeId;
  };

  // Render a nice table for properties
  const renderProperties = () => {
    const propEntries = Object.entries(properties).filter(([key]) => 
      key !== 'mapped' && key !== 'description' && !key.startsWith('__')
    );

    if (propEntries.length === 0) {
      return (
        <Typography variant="body2" color="text.secondary" sx={{ p: 2, fontStyle: 'italic' }}>
          No additional properties defined.
        </Typography>
      );
    }

    // Get property schema from node type for descriptions
    const semanticTermType = nodeTypes?.find((nt: any) => nt.catalog_type_name === 'semantic_term');
    const propertySchema = (semanticTermType?.properties || {}) as Record<string, any>;

    return (
      <TableContainer component={Paper} elevation={0} variant="outlined" sx={{ mt: 2 }}>
        <Table size="small">
          <TableHead>
            <TableRow sx={{ bgcolor: 'action.hover' }}>
              <TableCell sx={{ fontWeight: 600 }}>Property</TableCell>
              <TableCell sx={{ fontWeight: 600 }}>Value</TableCell>
            </TableRow>
          </TableHead>
          <TableBody>
            {propEntries.map(([key, value]) => {
              const schema = propertySchema[key];
              const description = schema?.description || `Property: ${key}`;
              
              return (
                <TableRow key={key} hover>
                  <Tooltip title={description} placement="left" arrow>
                    <TableCell 
                      component="th" 
                      scope="row" 
                      sx={{ 
                        color: 'text.secondary', 
                        width: '30%',
                        cursor: 'help',
                        '&:hover': {
                          color: 'primary.main',
                          fontWeight: 500
                        }
                      }}
                    >
                      {key.replace(/_/g, ' ')}
                    </TableCell>
                  </Tooltip>
                  <TableCell>
                    {typeof value === 'object' ? JSON.stringify(value) : String(value)}
                  </TableCell>
                </TableRow>
              );
            })}
          </TableBody>
        </Table>
      </TableContainer>
    );
  };

  return (
    <>
      <Box sx={{ height: '100%', display: 'flex', flexDirection: 'column', bgcolor: '#f8fafc' }}>
      {/* Header */}
      <Paper elevation={0} sx={{ p: 3, borderBottom: 1, borderColor: 'divider', bgcolor: 'white' }}>
        <Box sx={{ display: 'flex', alignItems: 'flex-start', justifyContent: 'space-between' }}>
          <Box sx={{ display: 'flex', gap: 2 }}>
            <Box
              sx={{
                width: 48,
                height: 48,
                borderRadius: 2,
                bgcolor: 'primary.main',
                color: 'white',
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'center',
                boxShadow: 2
              }}
            >
              <LabelIcon fontSize="medium" />
            </Box>
            <Box>
              <Typography variant="h5" fontWeight={700} color="text.primary">
                {termData?.node_name || asset.name}
              </Typography>
              <Box sx={{ display: 'flex', alignItems: 'center', gap: 1, mt: 0.5 }}>
                <Typography variant="body2" color="text.secondary">
                  {asset.type === 'business_term' ? 'Business Term' : 'Semantic Term'}
                </Typography>
                <Divider orientation="vertical" flexItem variant="middle" />
                <Chip
                  icon={isMapped ? <MappedIcon /> : <UnmappedIcon />}
                  label={isMapped ? 'Mapped' : 'Unmapped'}
                  size="small"
                  color={isMapped ? 'success' : 'default'}
                  variant="outlined"
                  sx={{ height: 20, fontSize: '0.7rem' }}
                />
              </Box>
            </Box>
          </Box>
          <Box sx={{ display: 'flex', gap: 1 }}>
            <Tooltip title="Edit Term">
                <IconButton onClick={() => setEditTermDialogOpen(true)} color="primary" size="small">
                    <EditIcon />
                </IconButton>
            </Tooltip>
            <Tooltip title="Delete Term">
                <IconButton onClick={handleDeleteTerm} color="error" size="small">
                    <DeleteIcon />
                </IconButton>
            </Tooltip>
            <Tooltip title="Add Relationship">
                <IconButton onClick={() => setAddEdgeDialogOpen(true)} color="primary">
                    <AddLinkIcon />
                </IconButton>
            </Tooltip>
          </Box>
        </Box>


        {/* Description */}
        {(termData?.description || asset.node?.description) && (
          <Box sx={{ mt: 3, p: 2, bgcolor: 'action.hover', borderRadius: 2, border: 1, borderColor: 'divider' }}>
            <Typography variant="body2" color="text.primary" sx={{ fontStyle: 'italic', lineHeight: 1.6 }}>
              "{termData?.description || asset.node.description}"
            </Typography>
          </Box>
        )}
      </Paper>

      {/* Tabs */}
      <Box sx={{ borderBottom: 1, borderColor: 'divider', bgcolor: 'white' }}>
        <Tabs value={tabValue} onChange={handleTabChange} aria-label="term details tabs">
          <Tab 
            icon={<DescriptionIcon fontSize="small" />} 
            iconPosition="start" 
            label="Overview" 
          />
          <Tab 
            icon={<LineageIcon fontSize="small" />} 
            iconPosition="start" 
            label="Lineage" 
          />
          <Tab 
            icon={<RuleIcon fontSize="small" />} 
            iconPosition="start" 
            label="Validation Rules" 
          />
        </Tabs>
      </Box>

      {/* Content */}
      <Box sx={{ flex: 1, overflow: 'hidden', p: 0 }}>
        {/* Overview Tab */}
        <CustomTabPanel value={tabValue} index={0}>
          <Box sx={{ p: 3, height: '100%', overflow: 'auto' }}>
            
            {/* Parent Business Term (if applicable) */}
            {asset.node?.parent_id && (
              <Card variant="outlined" sx={{ borderRadius: 2, mb: 3 }}>
                <CardContent>
                  <Typography variant="h6" gutterBottom display="flex" alignItems="center">
                    <LabelIcon sx={{ mr: 1, color: 'text.secondary' }} />
                    Parent Business Term
                  </Typography>
                  <Divider sx={{ mb: 2 }} />
                  {(() => {
                    // Find parent business term from allNodes
                    const parentTerm = allNodes.find((n: any) => n.id === asset.node.parent_id);
                    
                    if (parentTerm) {
                      return (
                        <Box
                          sx={{
                            p: 2,
                            bgcolor: 'action.hover',
                            borderRadius: 1,
                            cursor: 'pointer',
                            transition: 'all 0.2s',
                            '&:hover': {
                              bgcolor: 'action.selected',
                              transform: 'translateX(4px)'
                            }
                          }}
                          onClick={() => {
                            if (onAssetSelect) {
                              onAssetSelect({
                                type: 'business_term',
                                id: parentTerm.id,
                                nodeId: parentTerm.id,
                                name: parentTerm.node_name || 'Untitled',
                                node: parentTerm
                              });
                            }
                          }}
                        >
                          <Typography variant="body1" fontWeight={600} color="primary.main">
                            {parentTerm.node_name || 'Untitled'}
                          </Typography>
                          {parentTerm.description && (
                            <Typography variant="body2" color="text.secondary" sx={{ mt: 0.5 }}>
                              {parentTerm.description}
                            </Typography>
                          )}
                        </Box>
                      );
                    }
                    
                    return (
                      <Typography variant="body2" color="text.secondary" sx={{ fontStyle: 'italic' }}>
                        Parent term not found (ID: {asset.node.parent_id})
                      </Typography>
                    );
                  })()}
                </CardContent>
              </Card>
            )}
            
            {/* Relationships Card */}
            <Card variant="outlined" sx={{ borderRadius: 2, mb: 3 }}>
                <CardContent>
                 <Typography variant="h6" gutterBottom display="flex" alignItems="center">
                  <RelationshipIcon sx={{ mr: 1, color: 'text.secondary' }} />
                  Relationships
                </Typography>
                <Divider sx={{ mb: 2 }} />
                {termEdges.length === 0 ? (
                    <Typography variant="body2" color="text.secondary" sx={{ fontStyle: 'italic' }}>
                        No relationships defined.
                    </Typography>
                ) : (
                    <TableContainer component={Paper} elevation={0} variant="outlined">
                        <Table size="small">
                            <TableHead>
                                <TableRow sx={{ bgcolor: 'action.hover' }}>
                                    <TableCell sx={{ fontWeight: 600, width: '60px' }}>Dir</TableCell>
                                    <TableCell sx={{ fontWeight: 600 }}>Predicate</TableCell>
                                    <TableCell sx={{ fontWeight: 600 }}>Node Type</TableCell>
                                    <TableCell sx={{ fontWeight: 600 }}>Path</TableCell>
                                    <TableCell sx={{ fontWeight: 600, width: '100px', textAlign: 'right' }}>Actions</TableCell>
                                </TableRow>
                            </TableHead>
                            <TableBody>
                                {(() => {
                                  // Debug: Log all available data to understand the structure
                                  // Debug: Log all available data to understand the structure
                                  // console.log('[SemanticTermDetails] Relationships Debug:', ...);
                                  
                                  return termEdges.map((edge: any) => {
                                    const sourceId = edge.source_node_id || edge.source || edge.from_id;
                                    const targetId = edge.target_node_id || edge.target || edge.to_id;
                                    const isSource = sourceId === asset.nodeId;
                                    const otherId = isSource ? targetId : sourceId;
                                    const otherName = getNodeName(otherId);
                                    
                                    const otherNode = dynamicLineage?.nodes?.find((n: any) => n.id === otherId);
                                    const staticNode = allNodes.find(n => n.id === otherId);
                                    
                                    // Debug: Log the lookup for this specific edge if it fails to resolve properly
                                    if (!otherNode && !staticNode) {
                                      console.warn(`[SemanticTermDetails] Edge target ${otherId} not found in nodes`);
                                    }

                                    // Helper to check if a string is a UUID
                                    const isUUID = (str: string) => /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i.test(str);

                                    // Get node type name - prioritize static node data which has correct types
                                    let nodeTypeName = 'Unknown';
                                    
                                    // Try static node first (most reliable)
                                    if (staticNode) {
                                      if (staticNode.catalog_type_name && !isUUID(staticNode.catalog_type_name)) {
                                        nodeTypeName = staticNode.catalog_type_name;
                                      } else if (staticNode.catalog_type && !isUUID(staticNode.catalog_type)) {
                                        nodeTypeName = staticNode.catalog_type;
                                      } else if (staticNode.node_type_id) {
                                        nodeTypeName = getNodeTypeName(staticNode.node_type_id);
                                      } else if (staticNode.catalog_type_name && isUUID(staticNode.catalog_type_name)) {
                                        nodeTypeName = getNodeTypeName(staticNode.catalog_type_name);
                                      }
                                    }
                                    
                                    // Fallback to dynamic node if static didn't work (or if static resolved to generic "Node")
                                    if ((nodeTypeName === 'Unknown' || nodeTypeName === 'Node') && otherNode) {
                                      let dynamicType = 'Unknown';
                                      
                                      if (otherNode?.data?.node_type_name && !isUUID(otherNode.data.node_type_name)) {
                                        dynamicType = otherNode.data.node_type_name;
                                      } else if (otherNode?.data?.nodeType && !isUUID(otherNode.data.nodeType)) {
                                        dynamicType = otherNode.data.nodeType;
                                      } else if (otherNode?.data?.node_type_id) {
                                        dynamicType = getNodeTypeName(otherNode.data.node_type_id);
                                      } else if (otherNode?.data?.node_type_name && isUUID(otherNode.data.node_type_name)) {
                                        dynamicType = getNodeTypeName(otherNode.data.node_type_name);
                                      }
                                      
                                      // Only overwrite if we found a better specific type (not just "Node" unless that's all we have)
                                      if (dynamicType !== 'Unknown' && dynamicType !== 'Node') {
                                        nodeTypeName = dynamicType;
                                      } else if (nodeTypeName === 'Unknown' && dynamicType === 'Node') {
                                        nodeTypeName = 'Node';
                                      }
                                    }
                                    
                                    const parentName = otherNode?.data?.parent_name || staticNode?.parent_name || '';
                                    
                                    // Debug logging to trace why paths might be missing (emit and console)
                                    const debugPayload = {
                                      edge,
                                      sourceId,
                                      targetId,
                                      otherId,
                                      otherName,
                                      nodeTypeName,
                                      otherNode,
                                      staticNode,
                                      otherNodeQualified:
                                        otherNode?.data?.qualified_path ||
                                        otherNode?.data?.qualifiedPath ||
                                        otherNode?.data?.properties?.qualified_path ||
                                        otherNode?.data?.properties?.qualifiedPath,
                                      staticNodeQualified:
                                        staticNode?.qualified_path ||
                                        staticNode?.qualifiedPath ||
                                        staticNode?.properties?.qualified_path ||
                                        staticNode?.data?.qualifiedPath,
                                    };
                                    devDebug('[SemanticTermDetails] relationship row', debugPayload);
                                    // Also log to console to ensure visibility when dev logger listeners are not attached
                                    if (typeof console !== 'undefined' && console.debug) {
                                      console.debug('[SemanticTermDetails] relationship row', debugPayload);
                                    }

                                    // Robust check for qualified path - check all possible locations
                                    const qualifiedPath = 
                                        otherNode?.data?.qualified_path || 
                                        otherNode?.data?.qualifiedPath ||
                                        otherNode?.data?.properties?.qualified_path || 
                                        otherNode?.data?.properties?.qualifiedPath ||
                                        staticNode?.qualified_path || 
                                        staticNode?.qualifiedPath ||
                                        staticNode?.properties?.qualified_path ||
                                        staticNode?.data?.qualifiedPath ||
                                        staticNode?.node_name;

                                    const pathDisplay = qualifiedPath || (parentName ? `${parentName}.${otherName}` : otherName);
                                    
                                    // Extract relationship type/predicate using centralized formatter
                                    const relType = formatEdgeLabel(edge);
                                    
                                    // Determine directionality relative to the visual flow (Upstream -> Center -> Downstream)
                                    // Use explicit badges instead of ambiguous arrows
                                    const flowDir = isSource ? 'Downstream' : 'Upstream';
                                    const flowColor = isSource ? 'info' : 'secondary'; // Blue for forward (output), Purple/Pink for input
                                    const flowIcon = isSource ? <ArrowDownwardIcon fontSize="inherit" /> : <ArrowUpwardIcon fontSize="inherit" />; // Use existing imports
                                    
                                    return (
                                        <TableRow key={edge.id || `${sourceId}-${targetId}`} hover>
                                            <TableCell>
                                                <Chip 
                                                    label={flowDir} 
                                                    size="small" 
                                                    color={flowColor} 
                                                    variant="outlined" 
                                                    icon={flowIcon}
                                                    sx={{ height: 20, fontSize: '0.7rem', '& .MuiChip-icon': { fontSize: '0.9rem' } }}
                                                />
                                            </TableCell>
                                            <TableCell sx={{ fontSize: '0.85rem' }}>
                                                {relType}
                                            </TableCell>
                                            <TableCell sx={{ fontSize: '0.85rem' }}>
                                                {nodeTypeName || 'Unknown'}
                                            </TableCell>
                                            <TableCell 
                                                sx={{ 
                                                    fontSize: '0.85rem', 
                                                    color: 'primary.main',
                                                    cursor: 'pointer',
                                                    '&:hover': {
                                                        textDecoration: 'underline'
                                                    }
                                                }}
                                                onClick={() => {
                                                    const otherAsset: EnhancedSelectedAsset = {
                                                        type: nodeTypeName as any, // nodeTypeName might not match the strict type union
                                                        id: otherId,
                                                        nodeId: otherId,
                                                        name: otherName,
                                                        node: otherNode || staticNode
                                                    };
                                                    onAssetSelect?.(otherAsset);
                                                }}
                                            >
                                                <a style={{ color: 'inherit', textDecoration: 'inherit' }}>
                                                    {pathDisplay}
                                                </a>
                                            </TableCell>
                                            <TableCell align="right">
                                                <Box sx={{ display: 'flex', justifyContent: 'flex-end', gap: 0.5 }}>
                                                    <Tooltip title="Delete Relationship">
                                                        <IconButton 
                                                            size="small" 
                                                            color="error"
                                                            onClick={(e) => {
                                                                e.stopPropagation();
                                                                handleDeleteEdge(edge.id || edge.edge_id);
                                                            }}
                                                        >
                                                            <DeleteIcon fontSize="small" />
                                                        </IconButton>
                                                    </Tooltip>
                                                </Box>
                                            </TableCell>
                                        </TableRow>
                                    );
                                  });
                                })()}
                            </TableBody>
                        </Table>
                    </TableContainer>
                )}
                </CardContent>
            </Card>

            <Card variant="outlined" sx={{ borderRadius: 2 }}>
              <CardContent>
                <Typography variant="h6" gutterBottom display="flex" alignItems="center">
                  <DescriptionIcon sx={{ mr: 1, color: 'text.secondary' }} />
                  Term Properties
                </Typography>
                <Divider sx={{ mb: 2 }} />
                {renderProperties()}
              </CardContent>
            </Card>

          </Box>
        </CustomTabPanel>

        {/* Lineage & Relationships Tab */}
        <CustomTabPanel value={tabValue} index={1}>
          <Box sx={{ p: 3, height: '100%', overflow: 'auto' }}>
            
            {/* Engine Toggle and Graph */}
            <Card variant="outlined" sx={{ borderRadius: 2 }}>
              <CardContent sx={{ p: 0, '&:last-child': { pb: 0 } }}>
                <Box sx={{ p: 2, display: 'flex', alignItems: 'center', justifyContent: 'space-between', borderBottom: 1, borderColor: 'divider' }}>
                    <Box sx={{ display: 'flex', alignItems: 'center', gap: 2 }}>
                      {/* Direction Toggle */}
                      <ToggleButtonGroup
                        value={directionMode}
                        exclusive
                        onChange={(_e, val) => val && setDirectionMode(val)}
                        size="small"
                        sx={{ 
                          height: 32,
                          '& .MuiToggleButton-root': {
                            px: 1.5,
                            py: 0,
                            fontSize: '0.75rem',
                            textTransform: 'none',
                            border: '1px solid #e2e8f0',
                            '&.Mui-selected': {
                              backgroundColor: 'primary.main',
                              color: 'white',
                              '&:hover': {
                                backgroundColor: 'primary.dark'
                              }
                            }
                          }
                        }}
                      >
                        <ToggleButton value="upstream">
                          <ArrowUpwardIcon sx={{ fontSize: 14, mr: 0.5 }} />
                          Lineage
                        </ToggleButton>
                        <ToggleButton value="both">
                          <SwapVertIcon sx={{ fontSize: 14, mr: 0.5 }} />
                          Both
                        </ToggleButton>
                        <ToggleButton value="downstream">
                          <ArrowDownwardIcon sx={{ fontSize: 14, mr: 0.5 }} />
                          Impact
                        </ToggleButton>
                      </ToggleButtonGroup>
                      <LineageIcon sx={{ mr: 1, color: 'text.secondary' }} />
                      <Typography variant="h6">Visual Lineage Graph</Typography>
                    </Box>
                </Box>
                <Box sx={{ height: '600px', width: '100%', position: 'relative', display: 'flex' }}>
                  {/* Floating Controls Sidebar - Integrated from Impact View */}
                  <Box 
                    sx={{ 
                      position: 'absolute', 
                      right: sidebarOpen ? '320px' : '16px', 
                      top: '16px', 
                      zIndex: 1100,
                      display: 'flex',
                      flexDirection: 'column',
                      gap: 1,
                      transition: 'right 0.3s ease'
                    }}
                  >
                    <Tooltip title="Lineage Graph" placement="left">
                      <IconButton 
                        size="small" 
                        sx={{ 
                          bgcolor: !sidebarOpen ? 'primary.main' : 'white',
                          color: !sidebarOpen ? 'white' : 'text.secondary',
                          boxShadow: 2,
                          '&:hover': { bgcolor: !sidebarOpen ? 'primary.dark' : 'action.hover' }
                        }}
                        onClick={() => setSidebarOpen(false)}
                      >
                        <GraphIcon fontSize="small" />
                      </IconButton>
                    </Tooltip>
                    
                    <Tooltip title="Explanation" placement="left">
                      <IconButton 
                        size="small" 
                        sx={{ 
                          bgcolor: sidebarOpen && sidebarMode === 'explanation' ? 'primary.main' : 'white',
                          color: sidebarOpen && sidebarMode === 'explanation' ? 'white' : 'text.secondary',
                          boxShadow: 2,
                          '&:hover': { bgcolor: sidebarOpen && sidebarMode === 'explanation' ? 'primary.dark' : 'action.hover' }
                        }}
                        onClick={() => {
                          setSidebarMode('explanation');
                          setSidebarOpen(true);
                        }}
                      >
                        <TextIcon fontSize="small" />
                      </IconButton>
                    </Tooltip>
                    
                    <Tooltip title="AI Assistant" placement="left">
                      <IconButton 
                        size="small" 
                        sx={{ 
                          bgcolor: sidebarOpen && sidebarMode === 'assistant' ? 'primary.main' : 'white',
                          color: sidebarOpen && sidebarMode === 'assistant' ? 'white' : 'text.secondary',
                          boxShadow: 2,
                          '&:hover': { bgcolor: sidebarOpen && sidebarMode === 'assistant' ? 'primary.dark' : 'action.hover' }
                        }}
                        onClick={() => {
                          setSidebarMode('assistant');
                          setSidebarOpen(true);
                        }}
                      >
                        <ChatIcon fontSize="small" />
                      </IconButton>
                    </Tooltip>
                  </Box>

                  <Box sx={{ flex: 1, position: 'relative' }}>
                    <DualLineageViewer
                      selectedAsset={asset}
                      technicalData={technicalData}
                      semanticData={dynamicLineage || semanticData}
                      forceLineageType="semantic"
                      isFullScreen={isSemanticFullScreen}
                      onToggleFullScreen={() => setIsSemanticFullScreen(!isSemanticFullScreen)}
                      onAssetClick={(relatedAsset) => {
                        devDebug('Clicked related asset:', relatedAsset);
                      }}
                      highlightedNodes={highlightedNodeIds}
                      directionMode={directionMode}
                    />
                  </Box>

                  {/* Integrated Sidebar */}
                  {sidebarOpen && (
                    <Box 
                      sx={{ 
                        width: '300px', 
                        height: '100%', 
                        bgcolor: 'white', 
                        borderLeft: 1, 
                        borderColor: 'divider',
                        display: 'flex',
                        flexDirection: 'column',
                        zIndex: 1050
                      }}
                    >
                      <Box sx={{ p: 1.5, display: 'flex', alignItems: 'center', justifyContent: 'space-between', borderBottom: 1, borderColor: 'divider' }}>
                        <Typography variant="subtitle2" fontWeight={700}>
                          {sidebarMode === 'explanation' ? 'Lineage Explanation' : 'AI Assistant'}
                        </Typography>
                        <IconButton size="small" onClick={() => setSidebarOpen(false)}>
                          <CloseIcon fontSize="small" />
                        </IconButton>
                      </Box>
                      <Box sx={{ flex: 1, overflow: 'auto', p: 0 }}>
                        {sidebarMode === 'explanation' ? (
                          <ImpactExplanation 
                            nodeType={asset.type as any} 
                            nodeId={asset.nodeId} 
                            directionMode={directionMode}
                          />
                        ) : (
                          <ImpactQA 
                            nodeType={asset.type as any} 
                            nodeId={asset.nodeId} 
                            directionMode={directionMode}
                            onHighlightNodes={setHighlightedNodeIds}
                          />
                        )}
                      </Box>
                    </Box>
                  )}
                </Box>
              </CardContent>
            </Card>
          </Box>
        </CustomTabPanel>

        {/* Validation Rules Tab */}
        <CustomTabPanel value={tabValue} index={2}>
            <Box sx={{ p: 2, height: '100%', overflow: 'auto' }}>
                <ValidationRuleEditor 
                    contextEntity={termData?.node_name || asset.name}
                    // We don't restrict field, assuming rules apply to term as an entity or fields within it? 
                    // Actually, for a term, maybe 'step_name' is typically the term if it's a leaf?
                    // Let's pass just contextEntity for now to show all rules for this term/entity.
                />
            </Box>
        </CustomTabPanel>

        {/* Lineage & Impact Tab */}
        <CustomTabPanel value={tabValue} index={3}>
            <Box sx={{ height: '100%', overflow: 'hidden' }}>
                <UnifiedLineageTab 
                    nodeType="semantic_term" 
                    nodeId={asset.nodeId}
                    initialDirection="both"
                />
            </Box>
        </CustomTabPanel>
      </Box>
      </Box>
      <AddEdgeDialog
        open={addEdgeDialogOpen}
        onClose={() => setAddEdgeDialogOpen(false)}
        sourceNodeId={asset.nodeId}
        sourceNodeType={asset.type}
        onEdgeAdded={() => {
          refreshLineage();
          if (onRefresh) onRefresh();
        }}
      />
      <EditSemanticTermDialog
        open={editTermDialogOpen}
        onClose={() => setEditTermDialogOpen(false)}
        term={{
          id: asset.nodeId,
          name: asset.name,
          description: asset.node?.description,
          properties: asset.node?.properties || {},
          tenant_id: asset.node?.tenant_id,
          tenant_datasource_id: asset.node?.tenant_datasource_id,
        }}
        onSave={() => {
          refreshLineage();
          if (onRefresh) onRefresh();
        }}
      />

      {/* Edit Edge Dialog - REMOVED - edit relationship was not needed */}
    </>
  );
};

export default SemanticTermDetails;