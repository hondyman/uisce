// utils/exportUtils.ts
import { Node as FlowNode } from 'reactflow';

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