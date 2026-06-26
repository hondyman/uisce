// services/exportService.ts
import { Node as FlowNode, Edge } from 'reactflow';
import { devError } from '../utils/devLogger';

export interface ExportOptions {
  format: 'csv' | 'json' | 'xml';
  delimiter?: string;
  includeRelationships: boolean;
  includeIndexes: boolean;
  includeComments: boolean;
  selectedSchemas: string[];
  selectedTables: string[];
  exportScope: 'all' | 'schemas' | 'tables';
}

export interface ExportIndex {
  name?: string;
  type?: string;
  columns?: string[];
  isUnique?: boolean;
  unique?: boolean;
  isPrimary?: boolean;
}

export interface ExportColumn {
  name?: string;
  type?: string;
  dataType?: string;
  maxLength?: string | number;
  length?: string | number;
  isNullable?: boolean;
  isPrimaryKey?: boolean;
  primary_key?: boolean;
  isForeignKey?: boolean;
  foreign_key?: boolean;
  defaultValue?: string;
  default_value?: string;
  comment?: string;
  description?: string;
  indexes?: ExportIndex[];
}

export interface ExportMetadata {
  exportedAt: string;
  exportedBy: string;
  version: string;
  totalSchemas: number;
  totalTables: number;
  totalColumns: number;
  totalRelationships: number;
  exportOptions: {
    format: string;
    scope: string;
    includeRelationships: boolean;
    includeIndexes: boolean;
    includeComments: boolean;
  };
}

export interface JSONExportData {
  metadata: {
    exportedAt: string;
    exportedBy: string;
    version: string;
    totalSchemas: number;
    totalTables: number;
    totalColumns: number;
    totalRelationships: number;
    exportOptions: {
      format: string;
      scope: string;
      includeRelationships: boolean;
      includeIndexes: boolean;
      includeComments: boolean;
      delimiter?: string;
    };
  };
  schemas: Record<string, {
    name: string;
    tableCount: number;
    tables: Record<string, JSONExportTable>;
  }>;
  relationships?: JSONExportRelationship[];
}

export interface JSONExportSchema {
  name: string;
  comment?: string;
  tables: JSONExportTable[];
}

export interface JSONExportTable {
  name: string;
  comment?: string;
  nodeId: string;
  position: { x: number; y: number };
  columnCount: number;
  columns: JSONExportColumn[];
  indexes?: ExportIndex[];
}

export interface JSONExportColumn {
  position: number;
  name: string;
  dataType: string;
  maxLength?: string | number;
  isNullable: boolean;
  isPrimaryKey: boolean;
  isForeignKey: boolean;
  defaultValue?: string;
  comment?: string;
  indexes?: ExportIndex[];
}

export interface JSONExportRelationship {
  id: string;
  constraintName: string;
  constraintType: string;
  sourceTable: {
    name: string;
    schema: string;
    nodeId: string;
    column: string;
  };
  targetTable: {
    name: string;
    schema: string;
    nodeId: string;
    column: string;
  };
  metadata?: unknown;
}

export interface ExportTable {
  schema: string;
  name: string;
  comment?: string;
  columns: ExportColumn[];
  indexes?: ExportIndex[];
}

// Utility functions
export const getNodeSchema = (node: FlowNode): string => {
  const tableName = node.data?.label || 'Unknown Table';
  
  if (node.data?.schema) return node.data.schema;
  if (node.data?.schemaName) return node.data.schemaName;
  if (node.data?.table_schema) return node.data.table_schema;
  if (node.data?.database_schema) return node.data.database_schema;
  if (node.data?.owner) return node.data.owner;
  if (node.data?.namespace) return node.data.namespace;
  if (node.data?.database) return node.data.database;
  
  if (tableName && tableName.includes('.')) {
    const parts = tableName.split('.');
    if (parts.length >= 2) {
      return parts[0];
    }
  }
  
  return 'public';
};

export const getTableName = (node: FlowNode): string => {
  const fullName = node.data?.label || 'Unknown Table';
  
  if (node.data?.displayName) {
    return node.data.displayName;
  }
  
  if (fullName.includes('.')) {
    const parts = fullName.split('.');
    if (parts.length >= 2) {
      return parts.slice(1).join('.');
    }
  }
  
  return fullName;
};

// CSV Export
export const exportToCSV = (
  filteredNodes: FlowNode[], 
  filteredEdges: Edge[], 
  exportOptions: ExportOptions
): string => {
  const delimiter = exportOptions.delimiter || ',';
  const rows: string[] = [];
  
  // Define headers
  const headers = [
    'Schema Name',
    'Table Name', 
    'Column Name',
    'Data Type',
    'Max Length',
    'Is Nullable',
    'Is Primary Key',
    'Is Foreign Key',
    'Default Value',
    'Column Position'
  ];
  
  // Add optional headers
  if (exportOptions.includeComments) {
    headers.push('Column Comment', 'Table Comment');
  }
  
  if (exportOptions.includeIndexes) {
    headers.push('Index Name', 'Index Type', 'Is Unique');
  }
  
  if (exportOptions.includeRelationships) {
    headers.push(
      'Referenced Schema',
      'Referenced Table', 
      'Referenced Column',
      'Constraint Name',
      'Constraint Type'
    );
  }
  
  rows.push(headers.join(delimiter));
  
  // Process each node
  filteredNodes.forEach(node => {
    const schema = getNodeSchema(node);
    const tableName = getTableName(node);
    const columns = node.data?.columns || [];
    const tableComment = node.data?.comment || node.data?.description || '';
    
    if (columns.length === 0) {
      // Handle tables with no columns
      const row = [
        `"${schema}"`,
        `"${tableName}"`,
        '""', '""', '""', '""', 'false', 'false', '""', '0'
      ];
      
      if (exportOptions.includeComments) {
        row.push('""', `"${tableComment}"`);
      }
      
      if (exportOptions.includeIndexes) {
        row.push('""', '""', 'false');
      }
      
      if (exportOptions.includeRelationships) {
        row.push('""', '""', '""', '""', '""');
      }
      
      rows.push(row.join(delimiter));
    } else {
      // Process each column
      columns.forEach((column: ExportColumn, index: number) => {
        const row = [
          `"${schema}"`,
          `"${tableName}"`,
          `"${column.name || ''}"`,
          `"${column.type || column.dataType || ''}"`,
          `"${column.maxLength || column.length || ''}"`,
          `"${column.isNullable !== false ? 'true' : 'false'}"`,
          `"${column.isPrimaryKey || column.primary_key ? 'true' : 'false'}"`,
          `"${column.isForeignKey || column.foreign_key ? 'true' : 'false'}"`,
          `"${column.defaultValue || column.default_value || ''}"`,
          `"${index + 1}"`
        ];
        
        if (exportOptions.includeComments) {
          row.push(
            `"${column.comment || column.description || ''}"`,
            `"${tableComment}"`
          );
        }
        
        if (exportOptions.includeIndexes) {
          const indexInfo = column.indexes || [];
          const indexName = indexInfo.length > 0 ? indexInfo[0].name : '';
          const indexType = indexInfo.length > 0 ? indexInfo[0].type : '';
          const isUnique = indexInfo.length > 0 ? indexInfo[0].unique : false;
          
          row.push(
            `"${indexName}"`,
            `"${indexType}"`,
            `"${isUnique ? 'true' : 'false'}"`
          );
        }
        
        if (exportOptions.includeRelationships) {
          const relatedEdge = filteredEdges.find(edge => 
            (edge.source === node.id && edge.sourceHandle === column.name) ||
            (edge.target === node.id && edge.targetHandle === column.name)
          );
          
          if (relatedEdge) {
            const referencedNode = filteredNodes.find(n => 
              n.id === (relatedEdge.source === node.id ? relatedEdge.target : relatedEdge.source)
            );
            
            const referencedColumn = relatedEdge.source === node.id 
              ? relatedEdge.targetHandle 
              : relatedEdge.sourceHandle;
            
            row.push(
              `"${referencedNode ? getNodeSchema(referencedNode) : ''}"`,
              `"${referencedNode ? getTableName(referencedNode) : ''}"`,
              `"${referencedColumn || ''}"`,
              `"${relatedEdge.data?.constraintName || relatedEdge.data?.name || ''}"`,
              `"${relatedEdge.type || 'FOREIGN_KEY'}"`
            );
          } else {
            row.push('""', '""', '""', '""', '""');
          }
        }
        
        rows.push(row.join(delimiter));
      });
    }
  });
  
  return rows.join('\n');
};

// JSON Export
export const exportToJSON = (
  filteredNodes: FlowNode[], 
  filteredEdges: Edge[], 
  exportOptions: ExportOptions
): string => {
  const data: JSONExportData = {
    metadata: {
      exportedAt: new Date().toISOString(),
      exportedBy: 'Data Catalog Export Tool',
      version: '2.0',
      totalSchemas: new Set(filteredNodes.map(n => getNodeSchema(n))).size,
      totalTables: filteredNodes.length,
      totalColumns: filteredNodes.reduce((sum, n) => sum + (n.data?.columns?.length || 0), 0),
      totalRelationships: filteredEdges.length,
      exportOptions: {
        format: exportOptions.format,
        scope: exportOptions.exportScope,
        includeRelationships: exportOptions.includeRelationships,
        includeIndexes: exportOptions.includeIndexes,
        includeComments: exportOptions.includeComments,
        delimiter: exportOptions.delimiter
      }
    },
    schemas: {} as Record<string, { name: string; tableCount: number; tables: Record<string, JSONExportTable> }>
  };
  
  // Group nodes by schema
  filteredNodes.forEach(node => {
    const schema = getNodeSchema(node);
    const tableName = getTableName(node);
    
    if (!data.schemas[schema]) {
      data.schemas[schema] = {
        name: schema,
        tableCount: 0,
        tables: {}
      };
    }
    
    data.schemas[schema].tableCount++;
    
    const tableData: JSONExportTable = {
      name: tableName,
      nodeId: node.id,
      position: node.position,
      columnCount: node.data?.columns?.length || 0,
      columns: (node.data?.columns || []).map((col: ExportColumn, index: number) => {
        const columnData: JSONExportColumn = {
          position: index + 1,
          name: col.name || '',
          dataType: col.type || col.dataType || 'unknown',
          maxLength: col.maxLength || col.length,
          isNullable: col.isNullable !== false,
          isPrimaryKey: col.isPrimaryKey || col.primary_key || false,
          isForeignKey: col.isForeignKey || col.foreign_key || false,
          defaultValue: col.defaultValue || col.default_value
        };
        
        if (exportOptions.includeComments) {
          columnData.comment = col.comment || col.description || '';
        }
        
        if (exportOptions.includeIndexes) {
          columnData.indexes = col.indexes || [];
        }
        
        return columnData;
      })
    };
    
    if (exportOptions.includeComments) {
      tableData.comment = node.data?.comment || node.data?.description || '';
    }
    
    if (exportOptions.includeIndexes) {
      tableData.indexes = node.data?.indexes || [];
    }
    
    data.schemas[schema].tables[tableName] = tableData;
  });
  
  // Add relationships if requested
  if (exportOptions.includeRelationships) {
    data.relationships = filteredEdges.map(edge => {
      const sourceNode = filteredNodes.find(n => n.id === edge.source);
      const targetNode = filteredNodes.find(n => n.id === edge.target);
      
      return {
        id: edge.id,
        constraintName: edge.data?.constraintName || edge.data?.name || '',
        constraintType: edge.type || 'FOREIGN_KEY',
        sourceTable: {
          name: sourceNode ? getTableName(sourceNode) : 'Unknown',
          schema: sourceNode ? getNodeSchema(sourceNode) : 'default',
          nodeId: edge.source,
          column: edge.sourceHandle || ''
        },
        targetTable: {
          name: targetNode ? getTableName(targetNode) : 'Unknown',
          schema: targetNode ? getNodeSchema(targetNode) : 'default',
          nodeId: edge.target,
          column: edge.targetHandle || ''
        },
        metadata: edge.data || {}
      };
    });
  }
  
  return JSON.stringify(data, null, 2);
};

// XML Export
export const exportToXML = (
  filteredNodes: FlowNode[], 
  filteredEdges: Edge[], 
  exportOptions: ExportOptions
): string => {
  const escapeXML = (str: string) => {
    return str.replace(/[<>&'"]/g, (c) => {
      switch (c) {
        case '<': return '&lt;';
        case '>': return '&gt;';
        case '&': return '&amp;';
        case "'": return '&apos;';
        case '"': return '&quot;';
        default: return c;
      }
    });
  };
  
  let xml = '<?xml version="1.0" encoding="UTF-8"?>\n';
  xml += '<database>\n';
  
  // Metadata
  xml += '  <metadata>\n';
  xml += `    <exportedAt>${new Date().toISOString()}</exportedAt>\n`;
  xml += `    <exportedBy>Data Catalog Export Tool</exportedBy>\n`;
  xml += `    <version>2.0</version>\n`;
  xml += `    <totalSchemas>${new Set(filteredNodes.map(n => getNodeSchema(n))).size}</totalSchemas>\n`;
  xml += `    <totalTables>${filteredNodes.length}</totalTables>\n`;
  xml += `    <totalColumns>${filteredNodes.reduce((sum, n) => sum + (n.data?.columns?.length || 0), 0)}</totalColumns>\n`;
  xml += `    <totalRelationships>${filteredEdges.length}</totalRelationships>\n`;
  xml += '    <exportOptions>\n';
  xml += `      <format>${exportOptions.format}</format>\n`;
  xml += `      <scope>${exportOptions.exportScope}</scope>\n`;
  xml += `      <includeRelationships>${exportOptions.includeRelationships}</includeRelationships>\n`;
  xml += `      <includeIndexes>${exportOptions.includeIndexes}</includeIndexes>\n`;
  xml += `      <includeComments>${exportOptions.includeComments}</includeComments>\n`;
  xml += '    </exportOptions>\n';
  xml += '  </metadata>\n';
  
  // Schemas
  xml += '  <schemas>\n';
  
  const schemaData: { [key: string]: FlowNode[] } = {};
  filteredNodes.forEach(node => {
    const schema = getNodeSchema(node);
    if (!schemaData[schema]) schemaData[schema] = [];
    schemaData[schema].push(node);
  });
  
  Object.entries(schemaData).forEach(([schemaName, schemaNodes]) => {
    xml += `    <schema name="${escapeXML(schemaName)}" tableCount="${schemaNodes.length}">\n`;
    xml += '      <tables>\n';
    
    schemaNodes.forEach(node => {
      const tableName = getTableName(node);
      const columnCount = node.data?.columns?.length || 0;
      const tableComment = node.data?.comment || node.data?.description || '';
      
      xml += `        <table name="${escapeXML(tableName)}" nodeId="${escapeXML(node.id)}" columnCount="${columnCount}"`;
      
      if (exportOptions.includeComments && tableComment) {
        xml += ` comment="${escapeXML(tableComment)}"`;
      }
      
      xml += '>\n';
      xml += `          <position x="${node.position.x}" y="${node.position.y}" />\n`;
      
      // Indexes (if included)
      if (exportOptions.includeIndexes && node.data?.indexes) {
        xml += '          <indexes>\n';
        node.data.indexes.forEach((index: ExportIndex) => {
          xml += `            <index name="${escapeXML(index.name || '')}" `;
          xml += `type="${escapeXML(index.type || '')}" `;
          xml += `unique="${index.unique || false}" `;
          xml += `columns="${escapeXML((index.columns || []).join(','))}" />\n`;
        });
        xml += '          </indexes>\n';
      }
      
      // Columns
      xml += '          <columns>\n';
      (node.data?.columns || []).forEach((col: ExportColumn, index: number) => {
        xml += `            <column position="${index + 1}" `;
        xml += `name="${escapeXML(col.name || '')}" `;
        xml += `dataType="${escapeXML(col.type || col.dataType || '')}" `;
        xml += `maxLength="${escapeXML(String(col.maxLength || col.length || ''))}" `;
        xml += `isNullable="${col.isNullable !== false}" `;
        xml += `isPrimaryKey="${col.isPrimaryKey || col.primary_key || false}" `;
        xml += `isForeignKey="${col.isForeignKey || col.foreign_key || false}" `;
        xml += `defaultValue="${escapeXML(col.defaultValue || col.default_value || '')}"`;
        
        if (exportOptions.includeComments) {
          xml += ` comment="${escapeXML(col.comment || col.description || '')}"`;
        }
        
        xml += '>\n';
        
        // Column indexes (if included)
        if (exportOptions.includeIndexes && col.indexes) {
          col.indexes.forEach((index: ExportIndex) => {
            xml += `              <index name="${escapeXML(index.name || '')}" `;
            xml += `type="${escapeXML(index.type || '')}" `;
            xml += `unique="${index.unique || false}" />\n`;
          });
        }
        
        xml += '            </column>\n';
      });
      xml += '          </columns>\n';
      xml += '        </table>\n';
    });
    
    xml += '      </tables>\n';
    xml += '    </schema>\n';
  });
  
  xml += '  </schemas>\n';
  
  // Relationships (if included)
  if (exportOptions.includeRelationships) {
    xml += '  <relationships>\n';
    
    filteredEdges.forEach(edge => {
      const sourceNode = filteredNodes.find(n => n.id === edge.source);
      const targetNode = filteredNodes.find(n => n.id === edge.target);
      
      xml += '    <relationship ';
      xml += `id="${escapeXML(edge.id)}" `;
      xml += `constraintName="${escapeXML(edge.data?.constraintName || edge.data?.name || '')}" `;
      xml += `constraintType="${escapeXML(edge.type || 'FOREIGN_KEY')}">\n`;
      xml += `      <source table="${escapeXML(sourceNode ? getTableName(sourceNode) : 'Unknown')}" `;
      xml += `schema="${escapeXML(sourceNode ? getNodeSchema(sourceNode) : 'default')}" `;
      xml += `nodeId="${escapeXML(edge.source)}" `;
      xml += `column="${escapeXML(edge.sourceHandle || '')}" />\n`;
      xml += `      <target table="${escapeXML(targetNode ? getTableName(targetNode) : 'Unknown')}" `;
      xml += `schema="${escapeXML(targetNode ? getNodeSchema(targetNode) : 'default')}" `;
      xml += `nodeId="${escapeXML(edge.target)}" `;
      xml += `column="${escapeXML(edge.targetHandle || '')}" />\n`;
      
      // Additional metadata
      if (edge.data && Object.keys(edge.data).length > 0) {
        xml += '      <metadata>\n';
        Object.entries(edge.data).forEach(([key, value]) => {
          if (key !== 'constraintName' && key !== 'name') {
            xml += `        <${escapeXML(key)}>${escapeXML(String(value || ''))}</${escapeXML(key)}>\n`;
          }
        });
        xml += '      </metadata>\n';
      }
      
      xml += '    </relationship>\n';
    });
    
    xml += '  </relationships>\n';
  }
  
  xml += '</database>';
  return xml;
};

// Main export function
export const exportData = async (
  nodes: FlowNode[],
  edges: Edge[],
  exportOptions: ExportOptions
): Promise<void> => {
  try {
    // Filter nodes based on export scope
    let filteredNodes = nodes;
    
    if (exportOptions.exportScope === 'schemas') {
      filteredNodes = nodes.filter(node => {
        const schema = getNodeSchema(node);
        return exportOptions.selectedSchemas.includes(schema);
      });
    } else if (exportOptions.exportScope === 'tables') {
      filteredNodes = nodes.filter(node => 
        exportOptions.selectedTables.includes(node.id)
      );
    }
    
    // Filter edges to only include those between filtered nodes
    const nodeIds = new Set(filteredNodes.map(n => n.id));
    const filteredEdges = edges.filter(edge => 
      nodeIds.has(edge.source) && nodeIds.has(edge.target)
    );
    
    // Generate content based on format
    let content = '';
    let mimeType = '';
    let fileExtension = '';
    
    switch (exportOptions.format) {
      case 'csv':
        content = exportToCSV(filteredNodes, filteredEdges, exportOptions);
        mimeType = 'text/csv;charset=utf-8;';
        fileExtension = 'csv';
        break;
      case 'json':
        content = exportToJSON(filteredNodes, filteredEdges, exportOptions);
        mimeType = 'application/json;charset=utf-8;';
        fileExtension = 'json';
        break;
      case 'xml':
        content = exportToXML(filteredNodes, filteredEdges, exportOptions);
        mimeType = 'application/xml;charset=utf-8;';
        fileExtension = 'xml';
        break;
      default:
        throw new Error(`Unsupported export format: ${exportOptions.format}`);
    }
    
    // Generate filename
    const timestamp = new Date().toISOString().split('T')[0];
    const scopePrefix = exportOptions.exportScope === 'all' ? 'full' : 
                       exportOptions.exportScope === 'schemas' ? `${exportOptions.selectedSchemas.length}schemas` :
                       `${exportOptions.selectedTables.length}tables`;
    
    const filename = `data-catalog-${scopePrefix}-${timestamp}.${fileExtension}`;
    
    // Create and download file
    const blob = new Blob([content], { type: mimeType });
    const url = URL.createObjectURL(blob);
    const link = document.createElement('a');
    link.href = url;
    link.download = filename;
    link.style.display = 'none';
    
    document.body.appendChild(link);
    link.click();
    document.body.removeChild(link);
    URL.revokeObjectURL(url);
    
  } catch (error) {
    devError('Export failed:', error);
    throw error;
  }
};

// Helper function to estimate file size
export const estimateFileSize = (
  nodes: FlowNode[],
  format: 'csv' | 'json' | 'xml'
): string => {
  const totalColumns = nodes.reduce((sum, n) => sum + (n.data?.columns?.length || 0), 0);
  
  let estimatedBytes = 0;
  
  switch (format) {
    case 'csv':
      // Rough estimate: ~100 bytes per column (including headers and data)
      estimatedBytes = totalColumns * 100;
      break;
    case 'json':
      // JSON is more verbose: ~200 bytes per column
      estimatedBytes = totalColumns * 200;
      break;
    case 'xml':
      // XML is most verbose: ~300 bytes per column
      estimatedBytes = totalColumns * 300;
      break;
  }
  
  // Convert to MB
  const sizeInMB = estimatedBytes / (1024 * 1024);
  
  if (sizeInMB < 0.1) {
    return '< 0.1';
  } else if (sizeInMB < 1) {
    return sizeInMB.toFixed(1);
  } else {
    return Math.round(sizeInMB).toString();
  }
};
