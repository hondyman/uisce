// src/services/lineageService.ts - Debug version with enhanced logging
import { useState, useEffect, useRef, useCallback } from 'react';

// Import your existing types
import type { 
  SemanticLineageData, 
  SemanticNode,
  SemanticEdge,
  TechnicalLineageData,
  RawSemanticChart,
  ReactFlowSemanticChart,
  EnhancedSelectedAsset
} from '../types/SemanticTypes';

import { devLog, devDebug, devWarn, devError } from '../utils/devLogger';

export interface ReactFlowNode {
  id: string;
  type: string;
  position: { x: number; y: number };
  data: Record<string, unknown>;
}

export interface ReactFlowEdge {
  id: string;
  source: string;
  target: string;
  sourceHandle?: string;
  targetHandle?: string;
  type?: string;
  animated?: boolean;
  label?: string;
  data?: Record<string, unknown>;
}

export interface LineageApiResponse {
  technicalData?: TechnicalLineageData;
  semanticData?: RawSemanticChart;
  error?: string;
}

export interface LineageContainerProps {
  datasourceId: string;
  selectedAsset: EnhancedSelectedAsset | null;
  onAssetClick: (asset: EnhancedSelectedAsset | null) => void;
  onRelationshipClick: (relationship: unknown) => void;
}

export class LineageService {
  private baseUrl: string;

  constructor(baseUrl: string = '/api') {
    this.baseUrl = baseUrl;
  }

  async fetchDualLineageData(datasourceId: string): Promise<{
    technicalData: TechnicalLineageData | null;
    semanticData: SemanticLineageData | null;
  }> {
    try {
  const response = await fetch(`${this.baseUrl}/lineage/dual?datasourceId=${datasourceId}`, { credentials: 'include' });
      
      if (!response.ok) {
        throw new Error(`HTTP error! status: ${response.status}`);
      }

      const data: LineageApiResponse = await response.json();
      
      if (data.error) {
        throw new Error(data.error);
      }

      devLog('=== RAW API RESPONSE ===');
      devLog('Full response:', data);
      devLog('Semantic data keys:', data.semanticData ? Object.keys(data.semanticData) : 'No semantic data');
      
      if (data.semanticData) {
        devDebug('Business terms raw:', data.semanticData.businessTerms);
        devDebug('Semantic terms raw:', data.semanticData.semanticTerms);
        devDebug('Semantic columns raw:', data.semanticData.semanticColumns);
        devDebug('Database columns raw:', data.semanticData.databaseColumns);
        devDebug('Edges raw:', data.semanticData.edges);
      }

      // Parse and transform the data to match expected types
      const result = {
        technicalData: data.technicalData ? this.transformTechnicalData(data.technicalData) : null,
        semanticData: data.semanticData ? this.transformSemanticData(data.semanticData) : null
      };

      devLog('=== TRANSFORMED DATA ===');
      devDebug('Transformed result:', result);
      
      if (result.semanticData && 'businessTerms' in result.semanticData) {
        devDebug('Transformed business terms count:', result.semanticData.businessTerms.length);
        devDebug('Transformed semantic terms count:', result.semanticData.semanticTerms.length);
        devDebug('Transformed semantic columns count:', result.semanticData.semanticColumns.length);
        devDebug('Transformed database columns count:', result.semanticData.databaseColumns.length);
        devDebug('Transformed edges count:', result.semanticData.edges.length);
        
        // Check for Hire Date specifically
        const hireDateBT = result.semanticData.businessTerms.find((bt: SemanticNode) => bt.node_name.includes('Hire Date'));
        const hireDateST = result.semanticData.semanticTerms.find((st: SemanticNode) => st.node_name.includes('EmployeeHireDate'));
        const hireDateSC = result.semanticData.semanticColumns.find((sc: SemanticNode) => sc.node_name.includes('hire_date'));
        const hireDateDC = result.semanticData.databaseColumns.find((dc: SemanticNode) => dc.node_name.includes('hire_date'));
        
        devDebug('=== HIRE DATE SPECIFIC DEBUG ===');
        devDebug('Business Term "Hire Date":', hireDateBT);
        devDebug('Semantic Term "EmployeeHireDate":', hireDateST);
        devDebug('Semantic Column with "hire_date":', hireDateSC);
        devDebug('Database Column "hire_date":', hireDateDC);
        
        const hireDateEdges = result.semanticData.edges.filter((edge: SemanticEdge) => 
          edge.relationship_type === 'defines' || 
          edge.relationship_type === 'implements' || 
          edge.relationship_type === 'maps_to'
        );
        devDebug('Hire Date related edges:', hireDateEdges);
      }
      
      return result;
    } catch (error) {
      devError('Failed to fetch lineage data:', error);
      throw error;
    }
  }

  private transformTechnicalData(rawData: TechnicalLineageData): TechnicalLineageData {
  devLog('=== TECHNICAL DATA TRANSFORMATION ===');
  devDebug('Raw technical data:', rawData);
    
    const result = {
      nodes: rawData.nodes || [],
      edges: rawData.edges || [],
      viewport: rawData.viewport || { x: 0, y: 0, zoom: 1 },
      metadata: {
        chartType: rawData.metadata?.chartType || 'technical',
        databaseNodeCount: rawData.metadata?.totalNodes || rawData.nodes?.length || 0,
        databaseEdgeCount: rawData.metadata?.databaseEdgeCount || rawData.edges?.length || 0
      }
    };
    
  devDebug('Transformed technical data:', result);
    return result;
  }

  private transformSemanticData(rawData: RawSemanticChart | ReactFlowSemanticChart): SemanticLineageData {
  devLog('=== SEMANTIC DATA TRANSFORMATION ===');
  
  // If it's already a ReactFlow chart, return as-is
  if ('nodes' in rawData && 'edges' in rawData) {
    devDebug('Data is already ReactFlow format:', rawData);
    return rawData as ReactFlowSemanticChart;
  }
  
  // Otherwise, transform RawSemanticChart to ReactFlow format
  const rawChart = rawData as RawSemanticChart;
  devDebug('Raw semantic data structure:', {
      businessTerms: rawChart.businessTerms ? rawChart.businessTerms.length : 0,
      semanticTerms: rawChart.semanticTerms ? rawChart.semanticTerms.length : 0,
      semanticColumns: rawData.semanticColumns ? rawData.semanticColumns.length : 0,
      databaseColumns: rawData.databaseColumns ? rawData.databaseColumns.length : 0,
      edges: rawData.edges ? rawData.edges.length : 0
    });

    const transformedBusinessTerms = this.transformSemanticNodes(rawData.businessTerms || [], 'business_term');
    const transformedSemanticTerms = this.transformSemanticNodes(rawData.semanticTerms || [], 'semantic_term');
    const transformedSemanticColumns = this.transformSemanticNodes(rawData.semanticColumns || [], 'semantic_column');
    const transformedDatabaseColumns = this.transformSemanticNodes(rawData.databaseColumns || [], 'database_column');

  devDebug('Transformed node counts:', {
      businessTerms: transformedBusinessTerms.length,
      semanticTerms: transformedSemanticTerms.length,
      semanticColumns: transformedSemanticColumns.length,
      databaseColumns: transformedDatabaseColumns.length
    });

    const result = {
      businessTerms: transformedBusinessTerms,
      semanticTerms: transformedSemanticTerms,
      semanticColumns: transformedSemanticColumns,
      databaseColumns: transformedDatabaseColumns,
      edges: rawData.edges || [],
      viewport: rawData.viewport || { x: 0, y: 0, zoom: 1 },
      metadata: rawData.metadata || {}
    };

  devDebug('Final semantic transformation result:', result);
    return result;
  }

  private transformSemanticNodes(nodes: any[], expectedType?: string): SemanticNode[] {
  devDebug(`=== TRANSFORMING ${expectedType?.toUpperCase()} NODES ===`);
  devDebug('Input nodes:', nodes);
    
    if (!Array.isArray(nodes)) {
      devWarn('Nodes is not an array');
      devDebug('Nodes is not an array:', nodes);
      return [];
    }

    const transformed = nodes.map((node, index) => {
  devDebug(`Transforming node ${index}:`, node);
      
      const transformedNode = {
        id: node.id || node.ID || `generated-${index}`,
        node_type_id: node.nodeTypeId || node.node_type_id || node.NodeTypeID || '',
        node_name: node.nodeName || node.node_name || node.NodeName || node.name || '',
        qualified_path: node.qualifiedPath || node.qualified_path || node.QualifiedPath || '',
        node_type: this.mapNodeType(node.nodeType || node.node_type || node.NodeType || expectedType || ''),
        description: node.description || node.Description || '',
        properties: node.properties || node.Properties || {}
      };
      
  devDebug(`Transformed node ${index}:`, transformedNode);
      return transformedNode;
    });

  devDebug(`Final transformed ${expectedType} nodes:`, transformed);
    return transformed;
  }

  private mapNodeType(rawType: string): "business_term" | "semantic_term" | "semantic_column" | "database_column" | "semantic_model" {
  devDebug('Mapping node type:', rawType);
    
    const lowerType = rawType.toLowerCase();
    
    let mappedType: "business_term" | "semantic_term" | "semantic_column" | "database_column" | "semantic_model";
    
    if (lowerType.includes('business') || lowerType === 'business_term') {
      mappedType = 'business_term';
    } else if (lowerType.includes('semantic_term') || lowerType === 'semantic_term') {
      mappedType = 'semantic_term';
    } else if (lowerType.includes('semantic_column') || lowerType === 'semantic_column') {
      mappedType = 'semantic_column';
    } else if (lowerType.includes('database') || lowerType === 'database_column') {
      mappedType = 'database_column';
    } else if (lowerType.includes('model') || lowerType === 'semantic_model') {
      mappedType = 'semantic_model';
    } else {
      mappedType = 'semantic_term'; // Default fallback
    }
    
  devDebug(`Mapped "${rawType}" to "${mappedType}"`);
    return mappedType;
  }

  async fetchTechnicalLineageData(datasourceId: string): Promise<TechnicalLineageData | null> {
    try {
  const response = await fetch(`${this.baseUrl}/lineage/technical?datasourceId=${datasourceId}`, { credentials: 'include' });
      
      if (!response.ok) {
        throw new Error(`HTTP error! status: ${response.status}`);
      }

      const data: { technicalData?: TechnicalLineageData; error?: string } = await response.json();
      
      if (data.error) {
        throw new Error(data.error);
      }

      return data.technicalData ? this.transformTechnicalData(data.technicalData) : null;
    } catch (error) {
      devError('Failed to fetch technical lineage data:', error);
      throw error;
    }
  }

  async fetchSemanticLineageData(datasourceId: string): Promise<SemanticLineageData | null> {
    try {
  const response = await fetch(`${this.baseUrl}/lineage/semantic?datasourceId=${datasourceId}`, { credentials: 'include' });
      
      if (!response.ok) {
        throw new Error(`HTTP error! status: ${response.status}`);
      }

      const data: { semanticData?: SemanticLineageData; error?: string } = await response.json();
      
      if (data.error) {
        throw new Error(data.error);
      }

      return data.semanticData ? this.transformSemanticData(data.semanticData) : null;
    } catch (error) {
      devError('Failed to fetch semantic lineage data:', error);
      throw error;
    }
  }

  // Business term search and validation methods
  async searchBusinessTerms(params: {
    query?: string;
    category?: string;
    status?: string;
    tags?: string[];
    limit?: number;
    offset?: number;
  }): Promise<{ business_terms: any[]; total: number }> {
    const queryParams = new URLSearchParams();
    if (params.query) queryParams.append('query', params.query);
    if (params.category) queryParams.append('category', params.category);
    if (params.status) queryParams.append('status', params.status);
    if (params.tags) params.tags.forEach(tag => queryParams.append('tags', tag));
    if (params.limit) queryParams.append('limit', params.limit.toString());
    if (params.offset) queryParams.append('offset', params.offset.toString());

  const response = await fetch(`${this.baseUrl}/business-terms?${queryParams}`, { credentials: 'include' });
    if (!response.ok) {
      throw new Error(`Failed to search business terms: ${response.statusText}`);
    }
    return response.json();
  }

  async validateBusinessTerms(businessTerms: any[]): Promise<{
    valid: boolean;
    errors: string[];
    warnings: string[];
  }> {
    const response = await fetch(`${this.baseUrl}/business-terms/validate`, {
      method: 'POST',
      credentials: 'include',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({ business_terms: businessTerms }),
    });

    if (!response.ok) {
      throw new Error(`Failed to validate business terms: ${response.statusText}`);
    }
    return response.json();
  }
}

// Hook for using the lineage service
export const useLineageData = (datasourceId: string) => {
  const [technicalData, setTechnicalData] = useState<TechnicalLineageData | null>(null);
  const [semanticData, setSemanticData] = useState<SemanticLineageData | null>(null);
  const [loading, setLoading] = useState<boolean>(false);
  const [error, setError] = useState<string | null>(null);

  // Keep a stable service instance across renders
  const lineageServiceRef = useRef<LineageService | null>(null);
  if (!lineageServiceRef.current) lineageServiceRef.current = new LineageService();

  const fetchData = useCallback(async () => {
    if (!datasourceId) return;

    setLoading(true);
    setError(null);

    try {
      const data = await lineageServiceRef.current!.fetchDualLineageData(datasourceId);
      setTechnicalData(data.technicalData);
      setSemanticData(data.semanticData);
    } catch (err) {
      const errorMessage = err instanceof Error ? err.message : 'Unknown error occurred';
      setError(errorMessage);
      devError('Error fetching lineage data:', err);
    } finally {
      setLoading(false);
    }
  }, [datasourceId]);

  useEffect(() => {
    fetchData();
  }, [fetchData]);

  return {
    technicalData,
    semanticData,
    loading,
    error,
    refetch: fetchData
  };
};

export default LineageService;