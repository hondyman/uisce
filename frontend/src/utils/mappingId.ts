import type { Mapping } from '../components/semantic-mapper/types';

export function getColumnUniqueId(column: any) {
  if (!column) return 'unknown-unknown-unknown';
  return `${column.schema || 'default'}-${column.table}-${column.column}`;
}

export function getMappingUniqueId(mapping: Mapping) {
  if (!mapping) return 'unknown-unknown-unknown';
  return mapping.id || getColumnUniqueId(mapping.database_column);
}

export default getMappingUniqueId;
