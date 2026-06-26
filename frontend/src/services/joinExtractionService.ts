import { apiGet, apiPost } from '../utils/api';
import { devWarn } from '../utils/devLogger';

export interface JoinSuggestion {
  source_table: string;
  target_table: string;
  source_column: string;
  target_column: string;
  relationship: string;
  join_sql: string;
  description: string;
}

export interface TableJoinDefinition {
  relationship: string;
  sql: string;
}

export interface TableJoinDefinitions {
  [targetTable: string]: TableJoinDefinition;
}

export interface JoinExtractorResponse {
  joins: JoinSuggestion[];
  count: number;
  datasource_id: string;
}

export interface TableJoinResponse {
  table_name: string;
  joins: TableJoinDefinitions;
  count: number;
  datasource_id: string;
}

export interface GeneratedCube {
  name: string;
  sql_table: string;
  title: string;
  description: string;
  public?: boolean;
  dimensions: Record<string, any>;
  measures: Record<string, any>;
  joins: TableJoinDefinitions;
  hierarchies?: Array<{
    name: string;
    title: string;
    levels: Array<{
      name: string;
      title: string;
      dimension: string;
      time_granularity?: string;
    }>;
  }>;
  drill_members?: string[];
}

export interface CubeGenerationResponse {
  cube: GeneratedCube;
  table_name: string;
  datasource_id: string;
}

/**
 * Service for extracting database join relationships and generating Cube.js structures
 */
export class JoinExtractionService {
  /**
   * Extract join suggestions from database foreign key relationships
   */
  async extractJoinSuggestions(datasourceId: string): Promise<JoinExtractorResponse> {
    return await apiGet(`fabric/joins/${datasourceId}`);
  }

  /**
   * Get join definitions for a specific table
   */
  async getTableJoinDefinitions(datasourceId: string, tableName: string): Promise<TableJoinResponse> {
    return await apiGet(`fabric/joins/${datasourceId}/table/${tableName}`);
  }

  /**
   * Generate a complete Cube.js cube definition from a database table
   */
  async generateCubeFromTable(datasourceId: string, tableName: string): Promise<CubeGenerationResponse> {
    return await apiPost('fabric/cubes/generate-from-table', {
      datasource_id: datasourceId,
      table_name: tableName,
    });
  }

  /**
   * Build a join path between two tables using the extracted relationships
   */
  async buildJoinPath(
    datasourceId: string,
    sourceTable: string,
    targetTable: string
  ): Promise<string[]> {
    // Get all join suggestions for the datasource
    const suggestions = await this.extractJoinSuggestions(datasourceId);
    
    // Build a graph of table relationships
    const graph = new Map<string, Set<string>>();
    const edgeMap = new Map<string, JoinSuggestion>();
    
    suggestions.joins.forEach(join => {
      // Add forward edge
      if (!graph.has(join.source_table)) {
        graph.set(join.source_table, new Set());
      }
      graph.get(join.source_table)!.add(join.target_table);
      edgeMap.set(`${join.source_table}->${join.target_table}`, join);
      
      // Add reverse edge for bidirectional traversal
      if (!graph.has(join.target_table)) {
        graph.set(join.target_table, new Set());
      }
      graph.get(join.target_table)!.add(join.source_table);
      edgeMap.set(`${join.target_table}->${join.source_table}`, {
        ...join,
        source_table: join.target_table,
        target_table: join.source_table,
        source_column: join.target_column,
        target_column: join.source_column,
      });
    });

    // BFS to find shortest path
    const queue: { table: string; path: string[] }[] = [{ table: sourceTable, path: [sourceTable] }];
    const visited = new Set<string>();
    
    while (queue.length > 0) {
      const { table, path } = queue.shift()!;
      
      if (table === targetTable) {
        return path;
      }
      
      if (visited.has(table)) {
        continue;
      }
      visited.add(table);
      
      const neighbors = graph.get(table);
      if (neighbors) {
        for (const neighbor of neighbors) {
          if (!visited.has(neighbor)) {
            queue.push({
              table: neighbor,
              path: [...path, neighbor],
            });
          }
        }
      }
    }
    
    // No path found
    return [];
  }

  /**
   * Generate Cube.js join SQL for a path between tables
   */
  async generateJoinSQL(
    datasourceId: string,
    joinPath: string[]
  ): Promise<string[]> {
    if (joinPath.length < 2) {
      return [];
    }

    const suggestions = await this.extractJoinSuggestions(datasourceId);
    const edgeMap = new Map<string, JoinSuggestion>();
    
    suggestions.joins.forEach(join => {
      edgeMap.set(`${join.source_table}->${join.target_table}`, join);
      // Add reverse mapping
      edgeMap.set(`${join.target_table}->${join.source_table}`, {
        ...join,
        source_table: join.target_table,
        target_table: join.source_table,
        source_column: join.target_column,
        target_column: join.source_column,
        join_sql: `{CUBE.${join.target_column}} = {${join.source_table}.${join.source_column}}`,
      });
    });

    const joinStatements: string[] = [];
    
    for (let i = 0; i < joinPath.length - 1; i++) {
      const sourceTable = joinPath[i];
      const targetTable = joinPath[i + 1];
      const edgeKey = `${sourceTable}->${targetTable}`;
      
      const join = edgeMap.get(edgeKey);
      if (join) {
        joinStatements.push(join.join_sql);
      }
    }
    
    return joinStatements;
  }

  /**
   * Validate if a join relationship exists between two tables
   */
  async validateJoinRelationship(
    datasourceId: string,
    sourceTable: string,
    targetTable: string
  ): Promise<boolean> {
    try {
      const joinDefs = await this.getTableJoinDefinitions(datasourceId, sourceTable);
      return targetTable in joinDefs.joins;
    } catch (error) {
      devWarn(`Failed to validate join relationship: ${error}`);
      return false;
    }
  }

  /**
   * Get all available tables that can be joined from a source table
   */
  async getJoinableTargets(datasourceId: string, sourceTable: string): Promise<string[]> {
    try {
      const joinDefs = await this.getTableJoinDefinitions(datasourceId, sourceTable);
      return Object.keys(joinDefs.joins);
    } catch (error) {
      devWarn(`Failed to get joinable targets: ${error}`);
      return [];
    }
  }

  /**
   * Format join relationship for display in UI
   */
  formatJoinDescription(join: JoinSuggestion): string {
    const relationshipType = join.relationship.replace(/_/g, '-');
    return `${join.source_table}.${join.source_column} → ${join.target_table}.${join.target_column} (${relationshipType})`;
  }

  /**
   * Convert database join suggestions to Cube.js join definitions
   */
  convertToCubeJoins(suggestions: JoinSuggestion[]): Record<string, TableJoinDefinition> {
    const cubeJoins: Record<string, TableJoinDefinition> = {};
    
    suggestions.forEach(join => {
      cubeJoins[join.target_table] = {
        relationship: join.relationship,
        sql: join.join_sql,
      };
    });
    
    return cubeJoins;
  }
}

// Export a singleton instance
export const joinExtractionService = new JoinExtractionService();
