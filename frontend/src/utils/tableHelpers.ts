export const getTableIdFromVal = (v: any): string => {
  if (!v) return '';
  if (typeof v === 'string') return v;
  // common id fields in catalog_node results
  return String(v.id || v.node_id || v.node_name || v.fetchKey || v.qualified_path || v.qualifiedPath || '');
};

export const getTableLabelFromVal = (v: any): string => {
  if (!v) return '';
  if (typeof v === 'string') return v;
  // prefer friendly labels if available
  return String(v.label || v.node_name || v.nodeName || v.tableName || v.qualified_path || v.qualifiedPath || v.id || '');
};

export default getTableIdFromVal;
