import { SemanticNode, SemanticLineageData, RawSemanticChart, ReactFlowNode } from '../../../types/SemanticTypes';
import { LINEAGE_LAYOUT, createLineagePosition, getLineageNodeStyle, formatEdgeLabel } from './lineageUtils';
import { devLog, devDebug } from '../../../utils/devLogger';

/**
 * Enhanced buildSemanticLineageLayout function with complete graph traversal
 */
export const buildSemanticLineageLayout = (
  centerNode: SemanticNode,
  data: SemanticLineageData
) => {
  devLog('Building semantic layout for:', centerNode);

  // Local edge type for layout calculations
  type FlowEdge = { id?: string | number; source?: string; target?: string;[k: string]: unknown };

  const layoutNodes: ReactFlowNode[] = [];
  const layoutEdges: FlowEdge[] = [];

  const getEdgeId = (e: FlowEdge | undefined) => (e && e.id != null) ? String(e.id) : '';

  // Handle ReactFlow format
  if ('nodes' in data && data.nodes) {
    devLog('Processing ReactFlow semantic format');
    const allNodes = data.nodes;
    const allEdges = data.edges as FlowEdge[];

    // Debug: Show all node types in the data
    const nodeTypes = new Set(allNodes.map(n => n.data?.nodeType).filter(Boolean));
    devDebug('Available node types in data:', Array.from(nodeTypes));

    // Debug: Show sample nodes of each type
    nodeTypes.forEach(type => {
      const sampleNode = allNodes.find(n => n.data?.nodeType === type);
      devDebug(`Sample ${type} node:`, {
        id: sampleNode?.id,
        label: sampleNode?.data?.label,
        data: sampleNode?.data
      });
    });

    // Debug: Show edges that might connect to database columns
    const dbColumnEdges = allEdges.filter(edge => {
      const targetNode = allNodes.find(n => n.id === edge.target);
      return targetNode?.data?.nodeType?.includes('column');
    });
    devDebug(`Found ${dbColumnEdges.length} edges connecting to column-type nodes:`,
      dbColumnEdges.map(e => ({
        id: e.id,
        source: e.source,
        target: e.target,
        sourceType: allNodes.find(n => n.id === e.source)?.data?.nodeType,
        targetType: allNodes.find(n => n.id === e.target)?.data?.nodeType
      }))
    );



    // Debug: Find all edges from semantic columns to any node type
    const semanticColumnNodes = allNodes.filter(n => n.data?.nodeType === 'semantic_column');
    const semanticColumnIds = semanticColumnNodes.map(n => n.id);
    const edgesFromSemanticColumns = allEdges.filter(e => typeof e.source === 'string' && semanticColumnIds.includes(e.source));
    devDebug(`Found ${edgesFromSemanticColumns.length} edges FROM semantic columns:`,
      edgesFromSemanticColumns.map(e => ({
        id: e.id,
        source: e.source,
        target: e.target,
        sourceLabel: allNodes.find(n => n.id === e.source)?.data?.label,
        targetLabel: allNodes.find(n => n.id === e.target)?.data?.label,
        targetType: allNodes.find(n => n.id === e.target)?.data?.nodeType
      }))
    );

    const centerReactNode = allNodes.find(n =>
      n.id === centerNode.id.toString() ||
      n.data?.label === centerNode.node_name ||
      n.data?.id === centerNode.id.toString()
    );

    if (!centerReactNode) {
      devDebug('Center node not found in ReactFlow nodes');
      return { nodes: [], edges: [] };
    }

    // Helper function to recursively find all connected nodes
    const traverseFromNode = (nodeId: string, visitedNodes: Set<string>, targetTypes: string[], semanticChain: any): ReactFlowNode[] => {
      if (visitedNodes.has(nodeId)) return [];
      visitedNodes.add(nodeId);

      devDebug(`    Traversing from node: ${nodeId}, looking for types: ${targetTypes.join(', ')}`);
      const connectedNodes: ReactFlowNode[] = [];
      const connectedEdges = allEdges.filter(edge => edge.source === nodeId);

      devDebug(`    Found ${connectedEdges.length} outgoing edges`);

      connectedEdges.forEach(edge => {
        const targetNode = allNodes.find(n => n.id === edge.target);
        if (targetNode) {
          devDebug(`    Edge ${edge.id}: ${nodeId} -> ${edge.target} (type: ${targetNode.data?.nodeType})`);

          // Add this edge to our collection
          if (!semanticChain.allEdges.some((e: FlowEdge) => getEdgeId(e) === getEdgeId(edge as FlowEdge))) {
            semanticChain.allEdges.push(edge as FlowEdge);
          }

          // If this is a target type we're looking for, add it
          if (targetTypes.some(type => targetNode.data?.nodeType === type || targetNode.data?.nodeType?.includes(type))) {
            devDebug(`    ✅ Found matching target type: ${targetNode.data?.nodeType}`);
            if (!connectedNodes.some(n => n.id === targetNode.id)) {
              connectedNodes.push(targetNode);
            }
          }

          // Continue traversing from this node to find more connections
          const furtherNodes = edge.target ? traverseFromNode(String(edge.target), visitedNodes, targetTypes, semanticChain) : [];
          connectedNodes.push(...furtherNodes.filter(n => !connectedNodes.some(existing => existing.id === n.id)));
        } else {
          devDebug(`    ⚠️ Target node ${edge.target} not found for edge ${edge.id}`);
        }
      });

      devDebug(`    Traversal from ${nodeId} returned ${connectedNodes.length} nodes`);
      return connectedNodes;
    };

    if (centerNode.node_type === 'business_term') {
      devLog('Building complete semantic chain for business term:', centerNode.node_name);

      // Enhanced traversal to capture ALL connected nodes through the semantic chain
      const semanticChain = {
        businessTerm: centerReactNode,
        semanticTerms: [] as ReactFlowNode[],
        semanticColumns: [] as ReactFlowNode[],
        databaseColumns: [] as ReactFlowNode[],
        allEdges: [] as FlowEdge[]
      };

      // Step 1: Find all semantic terms directly connected to the business term
      const directSemanticTerms = allEdges
        .filter(edge => edge.source === centerReactNode.id &&
          allNodes.find(n => n.id === edge.target)?.data?.nodeType === 'semantic_term')
        .map(edge => allNodes.find(n => n.id === edge.target)!)
        .filter(Boolean);

      // Step 2: Find direct semantic columns from business term
      const directSemanticColumns = allEdges
        .filter(edge => edge.source === centerReactNode.id &&
          allNodes.find(n => n.id === edge.target)?.data?.nodeType === 'semantic_column')
        .map(edge => allNodes.find(n => n.id === edge.target)!)
        .filter(Boolean);

      // Add direct connections and their edges
      semanticChain.semanticTerms.push(...directSemanticTerms);
      semanticChain.semanticColumns.push(...directSemanticColumns);

      // Add direct edges from business term
      semanticChain.allEdges.push(
        ...allEdges.filter(edge =>
          edge.source === centerReactNode.id &&
          (directSemanticTerms.some(st => st.id === edge.target) ||
            directSemanticColumns.some(sc => sc.id === edge.target))
        ) as FlowEdge[]
      );

      // Step 3: For each semantic term, find ALL connected semantic columns AND database columns recursively
      directSemanticTerms.forEach(semanticTerm => {
        const visitedNodes = new Set<string>();

        // 3a. Find semantic columns
        const connectedSemanticColumns = traverseFromNode(
          semanticTerm.id,
          visitedNodes,
          ['semantic_column'],
          semanticChain
        );

        // Add unique semantic columns
        connectedSemanticColumns.forEach(sc => {
          if (!semanticChain.semanticColumns.some(existing => existing.id === sc.id)) {
            semanticChain.semanticColumns.push(sc);
          }
        });

        // 3b. Find direct database columns (MAPS_TO relationship)
        // Reset visited nodes or use a new set to ensure we check this path even if visited for other types
        const dbVisitedNodes = new Set<string>();
        const connectedDatabaseColumns = traverseFromNode(
          semanticTerm.id,
          dbVisitedNodes,
          ['database_column', 'column', 'databaseColumn'],
          semanticChain
        );

        connectedDatabaseColumns.forEach(dc => {
          if (!semanticChain.databaseColumns.some(existing => existing.id === dc.id)) {
            devDebug('  Adding direct database column from semantic term (outgoing):', dc.data?.label);
            semanticChain.databaseColumns.push(dc);
          }
        });

        // Check incoming edges (Column -> Term, e.g. MAPS_TO)
        const incomingEdges = allEdges.filter(edge => edge.target === semanticTerm.id);
        incomingEdges.forEach(edge => {
          const sourceNode = allNodes.find(n => n.id === edge.source);
          if (sourceNode && (
            sourceNode.data?.nodeType === 'database_column' ||
            sourceNode.data?.nodeType === 'column' ||
            (typeof sourceNode.data?.nodeType === 'string' && sourceNode.data.nodeType.includes('column') && !sourceNode.data.nodeType.includes('semantic'))
          )) {
            if (!semanticChain.databaseColumns.some(existing => existing.id === sourceNode.id)) {
              devDebug('  Adding direct database column from semantic term (incoming):', sourceNode.data?.label);
              semanticChain.databaseColumns.push(sourceNode);

              // Add the edge explicitly since traverseFromNode won't catch it
              if (!semanticChain.allEdges.some(e => getEdgeId(e) === getEdgeId(edge as FlowEdge))) {
                semanticChain.allEdges.push(edge as FlowEdge);
              }
            }
          }
        });
      });

      // Step 4: For each semantic column (direct + indirect), find ALL connected database columns
      const allSemanticColumns = [...semanticChain.semanticColumns];
      devLog('Searching for database columns from semantic columns:', allSemanticColumns.length);

      allSemanticColumns.forEach((semanticColumn, index) => {
        devDebug(`Processing semantic column ${index + 1}:`, semanticColumn.data?.label, semanticColumn.id);

        // First, check direct edges from this semantic column
        const directDbEdges = allEdges.filter(edge =>
          edge.source === semanticColumn.id &&
          allNodes.find(n => n.id === edge.target)?.data?.nodeType?.includes('column')
        );

        devDebug(`  Found ${directDbEdges.length} direct database column edges:`, directDbEdges.map(e => ({
          id: e.id,
          target: e.target,
          targetType: allNodes.find(n => n.id === e.target)?.data?.nodeType
        })));

        // Add direct database columns
        directDbEdges.forEach(edge => {
          const dbColumn = allNodes.find(n => n.id === edge.target);
          if (dbColumn && !semanticChain.databaseColumns.some(existing => existing.id === dbColumn.id)) {
            devDebug('  Adding database column:', dbColumn.data?.label, dbColumn.data?.nodeType);
            semanticChain.databaseColumns.push(dbColumn);
            if (!semanticChain.allEdges.some(e => getEdgeId(e) === getEdgeId(edge as FlowEdge))) {
              semanticChain.allEdges.push(edge as FlowEdge);
            }
          }
        });

        // Also use recursive traversal
        const visitedNodes = new Set<string>();
        const connectedDatabaseColumns = traverseFromNode(
          semanticColumn.id,
          visitedNodes,
          ['database_column', 'column', 'databaseColumn'], // Try different possible naming conventions
          semanticChain
        );

        devDebug(`  Recursive traversal found ${connectedDatabaseColumns.length} additional database columns`);

        // Add unique database columns
        connectedDatabaseColumns.forEach(dc => {
          if (!semanticChain.databaseColumns.some(existing => existing.id === dc.id)) {
            devDebug('  Adding recursive database column:', dc.data?.label, dc.data?.nodeType);
            semanticChain.databaseColumns.push(dc);
          }
        });
      });

      devLog('Total database columns found:', semanticChain.databaseColumns.length);

      // Step 5: Ensure we capture all edges in the semantic chain
      // This will include the edge 97d82101-2b84-47a6-9ec0-f93Ofe389c3c you're looking for
      const allChainNodeIds = [
        centerReactNode.id,
        ...semanticChain.semanticTerms.map(n => n.id),
        ...semanticChain.semanticColumns.map(n => n.id),
        ...semanticChain.databaseColumns.map(n => n.id)
      ];

      // Find all edges that connect any two nodes in our chain
      const relevantEdges = allEdges.filter(edge => {
        const s = edge.source != null ? String(edge.source) : '';
        const t = edge.target != null ? String(edge.target) : '';
        return allChainNodeIds.includes(s) && allChainNodeIds.includes(t);
      });

      // Add any missing edges
      relevantEdges.forEach(edge => {
        if (!semanticChain.allEdges.some((e: FlowEdge) => getEdgeId(e) === getEdgeId(edge as FlowEdge))) {
          semanticChain.allEdges.push(edge as FlowEdge);
        }
      });

      devLog('Enhanced semantic chain built:');
      devLog('- Business Term: 1');
      devLog('- Semantic Terms:', semanticChain.semanticTerms.length);
      devLog('- Semantic Columns:', semanticChain.semanticColumns.length);
      devLog('- Database Columns:', semanticChain.databaseColumns.length);
      devLog('- Total Edges:', semanticChain.allEdges.length);
      devLog('- All Edge IDs:', semanticChain.allEdges.map((e: FlowEdge) => getEdgeId(e)));

      // Build layout with proper positioning
      // Center: Business Term (0,0) - Far Right of this group
      const centerPosition = createLineagePosition(0, 0, 1);
      layoutNodes.push({
        id: centerReactNode.id,
        type: 'hoverableNode',
        position: centerPosition,
        data: {
          ...centerReactNode.data,
          isCenter: true,
          style: getLineageNodeStyle(centerReactNode.data?.nodeType, true),
          width: LINEAGE_LAYOUT.nodeWidth,
          height: LINEAGE_LAYOUT.nodeHeight,
        }
      });

      // Level -1: Semantic Terms (Left of Business Term)
      semanticChain.semanticTerms.forEach((node: ReactFlowNode, index: number) => {
        layoutNodes.push({
          id: node.id,
          type: 'hoverableNode',
          position: createLineagePosition(-1, index, semanticChain.semanticTerms.length),
          data: {
            ...node.data,
            direction: 'upstream',
            style: getLineageNodeStyle(node.data?.nodeType, false),
            width: LINEAGE_LAYOUT.nodeWidth,
            height: LINEAGE_LAYOUT.nodeHeight,
          }
        });
      });

      // Level -2: Semantic Columns (Left of Semantic Terms)
      semanticChain.semanticColumns.forEach((node: ReactFlowNode, index: number) => {
        layoutNodes.push({
          id: node.id,
          type: 'hoverableNode',
          position: createLineagePosition(-2, index, semanticChain.semanticColumns.length),
          data: {
            ...node.data,
            direction: 'upstream',
            style: getLineageNodeStyle(node.data?.nodeType, false),
            width: LINEAGE_LAYOUT.nodeWidth,
            height: LINEAGE_LAYOUT.nodeHeight,
          }
        });
      });

      // Level -3: Database Columns (Far Left)
      semanticChain.databaseColumns.forEach((node: ReactFlowNode, index: number) => {
        layoutNodes.push({
          id: node.id,
          type: 'hoverableNode',
          position: createLineagePosition(-3, index, semanticChain.databaseColumns.length),
          data: {
            ...node.data,
            direction: 'upstream',
            style: getLineageNodeStyle(node.data?.nodeType, false),
            width: LINEAGE_LAYOUT.nodeWidth,
            height: LINEAGE_LAYOUT.nodeHeight,
            qualifiedPath: node.data?.qualifiedPath,
          }
        });
      });

      // Add all edges with proper styling
      layoutEdges.push(...semanticChain.allEdges.map((edge: FlowEdge) => ({
        ...edge,
        type: 'smoothstep',
        style: { stroke: '#64748b', strokeWidth: 1 }
      } as FlowEdge)));

    } else if (centerNode.node_type === 'semantic_column') {
      devLog('Building semantic chain for semantic column:', centerReactNode.data?.label);

      const semanticChain = {
        businessTerms: [] as ReactFlowNode[],
        semanticTerms: [] as ReactFlowNode[],
        semanticColumn: centerReactNode,
        databaseColumns: [] as ReactFlowNode[],
        allEdges: [] as FlowEdge[]
      };

      // Build Downstream (to database columns) - using the same enhanced traversal
      const visitedDownstream = new Set<string>();
      const downstreamColumns = traverseFromNode(
        centerReactNode.id,
        visitedDownstream,
        ['database_column', 'column'],
        semanticChain
      );
      semanticChain.databaseColumns.push(...downstreamColumns);

      // Build Upstream (to semantic terms and business terms)
      const upstreamStEdges = allEdges.filter(edge =>
        edge.target === centerReactNode.id &&
        allNodes.find(n => n.id === edge.source)?.data?.nodeType === 'semantic_term'
      );

      upstreamStEdges.forEach(edge => {
        const st = allNodes.find(n => n.id === edge.source);
        if (st && !semanticChain.semanticTerms.some(existing => existing.id === st.id)) {
          semanticChain.semanticTerms.push(st);
          semanticChain.allEdges.push(edge);

          // Find business terms for this semantic term
          const upstreamBtEdges = allEdges.filter(btEdge =>
            btEdge.target === st.id &&
            allNodes.find(n => n.id === btEdge.source)?.data?.nodeType === 'business_term'
          );

          upstreamBtEdges.forEach(btEdge => {
            const bt = allNodes.find(n => n.id === btEdge.source);
            if (bt && !semanticChain.businessTerms.some(existing => existing.id === bt.id)) {
              semanticChain.businessTerms.push(bt);
              semanticChain.allEdges.push(btEdge);
            }
          });
        }
      });

      // Also check for direct business term to semantic column connection
      const upstreamBtDirectEdges = allEdges.filter(edge =>
        edge.target === centerReactNode.id &&
        allNodes.find(n => n.id === edge.source)?.data?.nodeType === 'business_term'
      );

      upstreamBtDirectEdges.forEach(edge => {
        const bt = allNodes.find(n => n.id === edge.source);
        if (bt && !semanticChain.businessTerms.some(existing => existing.id === bt.id)) {
          semanticChain.businessTerms.push(bt);
          semanticChain.allEdges.push(edge);
        }
      });

      devLog('Semantic column chain built:');
      devDebug('- Business Terms:', semanticChain.businessTerms.length);
      devDebug('- Semantic Terms:', semanticChain.semanticTerms.length);
      devDebug('- Database Columns:', semanticChain.databaseColumns.length);
      devDebug('- Total Edges:', semanticChain.allEdges.length);

      // Build layout - Center: Semantic Column (Level 2)
      layoutNodes.push({
        id: centerReactNode.id,
        type: 'hoverableNode',
        position: createLineagePosition(2, 0, 1),
        data: {
          ...centerReactNode.data,
          isCenter: true,
          style: getLineageNodeStyle(centerReactNode.data?.nodeType, true),
          width: LINEAGE_LAYOUT.nodeWidth,
          height: LINEAGE_LAYOUT.nodeHeight,
        }
      });

      // Upstream
      semanticChain.businessTerms.forEach((node, index) => {
        layoutNodes.push({
          id: node.id,
          type: 'hoverableNode',
          position: createLineagePosition(0, index, semanticChain.businessTerms.length),
          data: {
            ...node.data,
            direction: 'upstream',
            style: getLineageNodeStyle(node.data?.nodeType, false),
            width: LINEAGE_LAYOUT.nodeWidth,
            height: LINEAGE_LAYOUT.nodeHeight,
          }
        });
      });

      semanticChain.semanticTerms.forEach((node, index) => {
        layoutNodes.push({
          id: node.id,
          type: 'hoverableNode',
          position: createLineagePosition(1, index, semanticChain.semanticTerms.length),
          data: {
            ...node.data,
            direction: 'upstream',
            style: getLineageNodeStyle(node.data?.nodeType, false),
            width: LINEAGE_LAYOUT.nodeWidth,
            height: LINEAGE_LAYOUT.nodeHeight,
          }
        });
      });

      // Downstream
      semanticChain.databaseColumns.forEach((node, index) => {
        layoutNodes.push({
          id: node.id,
          type: 'hoverableNode',
          position: createLineagePosition(3, index, semanticChain.databaseColumns.length),
          data: {
            ...node.data,
            direction: 'downstream',
            style: getLineageNodeStyle(node.data?.nodeType, false),
            width: LINEAGE_LAYOUT.nodeWidth,
            height: LINEAGE_LAYOUT.nodeHeight,
          }
        });
      });

      layoutEdges.push(...semanticChain.allEdges.map((edge: FlowEdge) => ({
        ...edge,
        type: 'smoothstep'
      } as FlowEdge)));
    }

  } else if ('businessTerms' in data) {
    // Handle original format with basic traversal logic
    devLog('Processing original semantic format');
    const rawData = data as RawSemanticChart;

    if (centerNode.node_type === 'business_term') {
      devLog('Building complete semantic chain for business term (original format)');
      const semanticChain = {
        businessTerm: centerNode,
        semanticTerms: [] as SemanticNode[],
        semanticColumns: [] as SemanticNode[],
        databaseColumns: [] as SemanticNode[],
        allEdges: [] as FlowEdge[]
      };

      const allNodes = [
        ...(rawData.businessTerms || []),
        ...(rawData.semanticTerms || []),
        ...(rawData.semanticColumns || []),
        ...(rawData.databaseColumns || [])
      ];
      const allEdges = (rawData.edges || []) as unknown as FlowEdge[];

      // Existing original format logic with basic traversal
      const directSemanticTerms = allEdges.filter(edge => edge.source_node_id?.toString() === centerNode.id.toString() && allNodes.find(n => n.id.toString() === edge.target_node_id?.toString())?.node_type === 'semantic_term').map(edge => allNodes.find(n => n.id.toString() === edge.target_node_id?.toString())!);
      const directSemanticColumns = allEdges.filter(edge => edge.source_node_id?.toString() === centerNode.id.toString() && allNodes.find(n => n.id.toString() === edge.target_node_id?.toString())?.node_type === 'semantic_column').map(edge => allNodes.find(n => n.id.toString() === edge.target_node_id?.toString())!);

      semanticChain.semanticTerms.push(...directSemanticTerms);
      semanticChain.allEdges.push(...allEdges.filter(edge => edge.source_node_id?.toString() === centerNode.id.toString() && directSemanticTerms.some(st => st.id.toString() === edge.target_node_id?.toString())) as FlowEdge[]);

      semanticChain.semanticColumns.push(...directSemanticColumns);
      semanticChain.allEdges.push(...allEdges.filter(edge => edge.source_node_id?.toString() === centerNode.id.toString() && directSemanticColumns.some(sc => sc.id.toString() === edge.target_node_id?.toString())) as FlowEdge[]);

      directSemanticTerms.forEach(st => {
        const semanticColumns = allEdges.filter(edge => edge.source_node_id?.toString() === st.id.toString() && allNodes.find(n => n.id.toString() === edge.target_node_id?.toString())?.node_type === 'semantic_column').map(edge => allNodes.find(n => n.id.toString() === edge.target_node_id?.toString())!);
        semanticChain.semanticColumns.push(...semanticColumns.filter(sc => sc && !semanticChain.semanticColumns.some(existing => existing.id.toString() === sc.id.toString())));
        semanticChain.allEdges.push(...allEdges.filter(edge => edge.source_node_id?.toString() === st.id.toString() && semanticColumns.some(sc => sc.id.toString() === edge.target_node_id?.toString())) as FlowEdge[]);
      });

      semanticChain.semanticColumns.forEach(sc => {
        const databaseColumns = allEdges.filter(edge => edge.source_node_id?.toString() === sc.id.toString() && edge.relationship_type === 'SemanticToDatabase').map(edge => allNodes.find(n => n.id.toString() === edge.target_node_id?.toString())!);
        semanticChain.databaseColumns.push(...databaseColumns.filter(dc => dc && !semanticChain.databaseColumns.some(existing => existing.id.toString() === dc.id.toString())));
        semanticChain.allEdges.push(...allEdges.filter(edge => edge.source_node_id?.toString() === sc.id.toString() && edge.relationship_type === 'SemanticToDatabase') as FlowEdge[]);
      });

      // Build layout
      layoutNodes.push({ id: centerNode.id.toString(), type: 'hoverableNode', position: createLineagePosition(0, 0, 1), data: { label: centerNode.node_name, nodeType: centerNode.node_type, isCenter: true, description: centerNode.description, style: getLineageNodeStyle(centerNode.node_type, true), width: LINEAGE_LAYOUT.nodeWidth, height: LINEAGE_LAYOUT.nodeHeight } });
      semanticChain.semanticTerms.forEach((node, index) => layoutNodes.push({ id: node.id.toString(), type: 'hoverableNode', position: createLineagePosition(1, index, semanticChain.semanticTerms.length), data: { label: node.node_name, nodeType: node.node_type, description: node.description, direction: 'downstream', style: getLineageNodeStyle(node.node_type, false), width: LINEAGE_LAYOUT.nodeWidth, height: LINEAGE_LAYOUT.nodeHeight } }));
      semanticChain.semanticColumns.forEach((node, index) => layoutNodes.push({ id: node.id.toString(), type: 'hoverableNode', position: createLineagePosition(2, index, semanticChain.semanticColumns.length), data: { label: node.node_name, nodeType: node.node_type, description: node.description, direction: 'downstream', style: getLineageNodeStyle(node.node_type, false), width: LINEAGE_LAYOUT.nodeWidth, height: LINEAGE_LAYOUT.nodeHeight } }));
      semanticChain.databaseColumns.forEach((node, index) => layoutNodes.push({ id: node.id.toString(), type: 'hoverableNode', position: createLineagePosition(3, index, semanticChain.databaseColumns.length), data: { label: node.node_name, nodeType: node.node_type, description: node.description, direction: 'downstream', style: getLineageNodeStyle(node.node_type, false), width: LINEAGE_LAYOUT.nodeWidth, height: LINEAGE_LAYOUT.nodeHeight } }));
      layoutEdges.push(...semanticChain.allEdges.map(edge => {
        const id = edge.id != null ? String(edge.id) : '';
        const source = edge.source_node_id != null ? String(edge.source_node_id) : (edge.source != null ? String(edge.source) : '');
        const target = edge.target_node_id != null ? String(edge.target_node_id) : (edge.target != null ? String(edge.target) : '');
        const label = formatEdgeLabel(edge);
        return { id, source, target, type: 'smoothstep', label, data: edge.properties, style: { stroke: '#64748b', strokeWidth: 1 } } as FlowEdge;
      }));
    }
    // Note: semantic_column case for original format would go here if needed
  }

  return { nodes: layoutNodes, edges: layoutEdges };
};