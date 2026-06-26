/**
 * Utilities for working with cube join paths, dimensions, and measures
 * 
 * This module provides functionality to:
 * 1. Extract available join paths from a selected cube
 * 2. Pull dimensions/measures from joined tables
 * 3. Handle includes: "*" patterns
 * 4. Generate view configurations with proper join path references
 */

import type { ModelCatalogNode } from '../types/model';
import { devError } from './devLogger';

export interface JoinPath {
  path: string;
  targetCube: string;
  relationship: string;
  description?: string;
}

export interface CubeMember {
  name: string;
  type: 'dimension' | 'measure';
  sql?: string;
  title?: string;
  description?: string;
  dataType?: string;
  joinPath?: string;
}

export interface JoinPathReference {
  joinPath: string;
  includes: string[] | '*';
  excludes?: string[];
  prefix?: boolean;
  alias?: string;
}

/**
 * Extracts available join paths from a cube's resolved configuration
 */
export function extractJoinPaths(model: ModelCatalogNode): JoinPath[] {
  if (!model?.resolved_config) return [];

  try {
    const config = typeof model.resolved_config === 'string' 
      ? JSON.parse(model.resolved_config) 
      : model.resolved_config;

    const joinPaths: JoinPath[] = [];

    // Handle both single cube and multiple cubes in config
    const cubes = config.cubes || (config.name ? [config] : []);
    
    cubes.forEach((cube: any) => {
      if (cube.joins && typeof cube.joins === 'object') {
        Object.entries(cube.joins).forEach(([joinName, joinDef]: [string, any]) => {
          joinPaths.push({
            path: joinName,
            targetCube: joinName,
            relationship: joinDef.relationship || 'many_to_one',
            description: joinDef.description || `Join to ${joinName}`
          });
        });
      }
    });

    return joinPaths;
  } catch (error) {
    try { devError('Error extracting join paths:', error); } catch {}
    return [];
  }
}

/**
 * Extracts dimensions and measures from a cube
 */
export function extractCubeMembers(model: ModelCatalogNode, joinPath?: string): CubeMember[] {
  if (!model?.resolved_config) return [];

  try {
    const config = typeof model.resolved_config === 'string' 
      ? JSON.parse(model.resolved_config) 
      : model.resolved_config;

    const members: CubeMember[] = [];
    
    // Handle both single cube and multiple cubes in config
    const cubes = config.cubes || (config.name ? [config] : []);
    
    cubes.forEach((cube: any) => {
      // Extract dimensions
      if (cube.dimensions && typeof cube.dimensions === 'object') {
        Object.entries(cube.dimensions).forEach(([dimName, dimDef]: [string, any]) => {
          members.push({
            name: dimName,
            type: 'dimension',
            sql: dimDef.sql,
            title: dimDef.title || formatTitle(dimName),
            description: dimDef.description,
            dataType: dimDef.type || 'string',
            joinPath
          });
        });
      }

      // Extract measures
      if (cube.measures && typeof cube.measures === 'object') {
        Object.entries(cube.measures).forEach(([measureName, measureDef]: [string, any]) => {
          members.push({
            name: measureName,
            type: 'measure',
            sql: measureDef.sql,
            title: measureDef.title || formatTitle(measureName),
            description: measureDef.description,
            dataType: measureDef.type || 'number',
            joinPath
          });
        });
      }
    });

    return members;
  } catch (error) {
    try { devError('Error extracting cube members:', error); } catch {}
    return [];
  }
}

/**
 * Gets all available dimensions and measures from a cube including joined tables
 */
export function getAllAvailableMembers(model: ModelCatalogNode, allModels: ModelCatalogNode[]): {
  mainCube: CubeMember[];
  joinedCubes: { [joinPath: string]: CubeMember[] };
} {
  const result = {
    mainCube: extractCubeMembers(model),
    joinedCubes: {} as { [joinPath: string]: CubeMember[] }
  };

  // Get join paths from the main cube
  const joinPaths = extractJoinPaths(model);

  // For each join path, find the target cube and extract its members
  joinPaths.forEach(join => {
    const targetModel = allModels.find(m => 
      m.model_key.includes(join.targetCube) || 
      m.display_name?.toLowerCase().includes(join.targetCube.toLowerCase())
    );

    if (targetModel) {
      result.joinedCubes[join.path] = extractCubeMembers(targetModel, join.path);
    }
  });

  return result;
}

/**
 * Generates a view configuration with proper join path references
 */
export function generateViewConfig(options: {
  baseCube: string;
  joinPathReferences: JoinPathReference[];
  name: string;
  title?: string;
  description?: string;
}): any {
  const { baseCube, joinPathReferences, name, title, description } = options;

  return {
    name,
    title: title || formatTitle(name),
    description: description || `View combining ${baseCube} with joined tables`,
    cubes: joinPathReferences.map(ref => ({
      join_path: ref.joinPath,
      includes: ref.includes,
      ...(ref.excludes && { excludes: ref.excludes }),
      ...(ref.prefix && { prefix: true }),
      ...(ref.alias && { alias: ref.alias })
    }))
  };
}

/**
 * Expands includes: "*" to explicit member names
 */
export function expandIncludesAll(members: CubeMember[], excludes: string[] = []): string[] {
  return members
    .map(m => m.name)
    .filter(name => !excludes.includes(name));
}

/**
 * Creates dimension declarations for joined tables
 */
export function createJoinedDimensions(options: {
  joinPath: string;
  members: CubeMember[];
  prefix?: string;
}): any[] {
  const { joinPath, members, prefix } = options;
  
  return members
    .filter(m => m.type === 'dimension')
    .map(dim => {
      const name = prefix ? `${prefix}_${dim.name}` : dim.name;
      return {
        name,
        sql: `${joinPath}.${dim.name}`,
        type: dim.dataType,
        title: dim.title || formatTitle(name),
        description: dim.description || `${dim.name} from ${joinPath}`
      };
    });
}

/**
 * Creates measure declarations for joined tables
 */
export function createJoinedMeasures(options: {
  joinPath: string;
  members: CubeMember[];
  prefix?: string;
}): any[] {
  const { joinPath, members, prefix } = options;
  
  return members
    .filter(m => m.type === 'measure')
    .map(measure => {
      const name = prefix ? `${prefix}_${measure.name}` : measure.name;
      return {
        name,
        sql: measure.sql?.replace(/\${CUBE}/g, `\${${joinPath}}`),
        type: measure.dataType,
        title: measure.title || formatTitle(name),
        description: measure.description || `${measure.name} from ${joinPath}`
      };
    });
}

/**
 * Formats a name into a proper title
 */
export function formatTitle(name: string): string {
  return name
    .replace(/_/g, ' ')
    .replace(/\b\w/g, l => l.toUpperCase());
}

/**
 * Validates join path syntax
 */
export function validateJoinPath(joinPath: string): { isValid: boolean; error?: string } {
  if (!joinPath || joinPath.trim() === '') {
    return { isValid: false, error: 'Join path cannot be empty' };
  }

  // Basic validation - join paths can be simple names or dot-separated paths
  if (!/^[a-zA-Z][a-zA-Z0-9_]*(\.[a-zA-Z][a-zA-Z0-9_]*)*$/.test(joinPath.trim())) {
    return { 
      isValid: false, 
      error: 'Join path must contain only letters, numbers, underscores, and dots for nested paths' 
    };
  }

  return { isValid: true };
}

/**
 * Example patterns for regional data aggregation
 */
export function generateRegionalAggregationExample(baseCube: string): {
  yaml: string;
  description: string;
} {
  return {
    description: `
Pattern for aggregating regional data from multiple joined tables.
This keeps ${baseCube} as the anchor and pulls in region-specific balances.
`,
    yaml: `
views:
  - name: ${baseCube}_regional_view
    title: ${formatTitle(baseCube)} Regional View
    description: Regional aggregation view for ${baseCube}
    cubes:
      # Main cube as anchor
      - join_path: ${baseCube}
        includes: "*"
        
      # Regional tables with explicit balance fields
      - join_path: account_north
        includes:
          - name: balance
            alias: north_balance
            title: "North Region Balance"
            
      - join_path: account_south
        includes:
          - name: balance
            alias: south_balance
            title: "South Region Balance"
            
      - join_path: account_east
        includes:
          - name: balance
            alias: east_balance
            title: "East Region Balance"
            
      - join_path: account_west
        includes:
          - name: balance
            alias: west_balance
            title: "West Region Balance"

# Alternative: Use includes: "*" to bring in all columns
views:
  - name: ${baseCube}_all_regions_view
    title: ${formatTitle(baseCube)} All Regions View
    cubes:
      - join_path: ${baseCube}
        includes: "*"
      - join_path: account_north
        includes: "*"
        prefix: true  # Prefixes all fields with "account_north_"
      - join_path: account_south
        includes: "*"
        prefix: true
      - join_path: account_east
        includes: "*"
        prefix: true
      - join_path: account_west
        includes: "*"
        prefix: true
`.trim()
  };
}
