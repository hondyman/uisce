import { useState, useEffect, useCallback } from 'react';
import { useScope } from '../../contexts/ScopeContext';
import { getRequiredTenantScope } from '../../utils/tenantScope';
import { devWarn, devError } from '../../utils/devLogger';
import resolveApiUrl from '../../utils/resolveApiUrl';

let generatedIdCounter = 0;
const makeUniqueId = (prefix = 'generated') => `${prefix}-${Date.now().toString(36)}-${++generatedIdCounter}`;

export function useSemanticMapper() {
  const [mappings, setMappings] = useState<any[]>([]);
  const [loading, setLoading] = useState(false);
  const [toast, setToast] = useState<{ type: 'success' | 'error'; message: string } | null>(null);
  const [editOriginals, setEditOriginals] = useState<Record<string, any>>({});

  const {
    schemaIds, tableIds, columnIds,
    schemaNames, tableNames, columnNames
  } = useScope();

  const generateMappingsForScope = useCallback(async (): Promise<any[]> => {
    const generatedMappings: any[] = [];
    try {
      // Get tenant scope for database_column objects
      const { tenantId, datasourceId } = getRequiredTenantScope();
      
      if (columnIds.length > 0 && columnNames.length > 0) {
        for (let i = 0; i < columnNames.length; i++) {
          generatedMappings.push({
            id: columnIds[i] ? `generated-${columnIds[i]}` : makeUniqueId(),
            database_column: { 
              schema: schemaNames[0] || 'unknown', 
              table: tableNames[0] || 'unknown', 
              column: columnNames[i], 
              node_id: columnIds[i], 
              data_type: 'unknown',
              tenant_id: tenantId,
              tenant_tenant_instance_id: datasourceId
            },
            semantic_term: null, confidence: 0, selected: false, ignored: false, override: false
          });
        }
      } else if (tableIds.length > 0) {
        for (let i = 0; i < tableIds.length; i++) {
          const tableId = tableIds[i];
          const params = new URLSearchParams({ type: 'column', parent_id: tableId, limit: '1000' });
          const res = await fetch(`/api/catalog/nodes?${params.toString()}`, { credentials: 'include' });
          if (res.ok) {
            const cols = await res.json();
            cols.forEach((col: any) => generatedMappings.push({
              id: col.id ? `generated-${col.id}` : makeUniqueId(),
              database_column: { 
                schema: schemaNames[i] || schemaNames[0] || 'unknown', 
                table: tableNames[i], 
                column: col.node_name, 
                node_id: col.id, 
                data_type: 'unknown',
                tenant_id: tenantId,
                tenant_tenant_instance_id: datasourceId
              },
              semantic_term: null, confidence: 0, selected: false, ignored: false, override: false
            }));
          }
        }
      } else if (schemaIds.length > 0) {
        // This can be very large, so it's often better to prompt the user to select tables.
        // For this refactor, I'll keep the logic but it could be optimized.
      }
    } catch (err) {
      devError('Error generating mappings for scope:', err);
    }
    return generatedMappings;
  }, [columnIds, columnNames, tableIds, tableNames, schemaIds, schemaNames]);

  const loadMappings = useCallback(async () => {
    setLoading(true);
    try {
  const res = await fetch(resolveApiUrl('/api/semantic-mappings'));
      let data: any[] = [];
      if (!res.ok) {
        const text = await res.text().catch(() => '<no body>');
        devError('[useSemanticMapper] Failed to fetch mappings:', res.status, text);
        if (schemaIds.length > 0 || tableIds.length > 0 || columnIds.length > 0) {
          data = await generateMappingsForScope();
        }
      } else {
        data = (await res.json()) || [];
        if (data.length === 0 && (schemaIds.length > 0 || tableIds.length > 0 || columnIds.length > 0)) {
          data = await generateMappingsForScope();
        }
      }
      setMappings(data);
    } catch (err) {
      devError('Failed to load mappings:', err);
      setMappings([]);
    }
    setLoading(false);
  }, [schemaIds, tableIds, columnIds, generateMappingsForScope]);

  useEffect(() => {
    loadMappings();
  }, [loadMappings]);

  const searchSemanticTerms = async (query: string) => {
    if (!query || query.length < 2) return [];
    try {
  const res = await fetch(resolveApiUrl('/api/semantic-terms/search'), {
        method: 'POST', headers: { 'Content-Type': 'application/json' }, credentials: 'include',
        body: JSON.stringify({ query, limit: 10 })
      });
      return (await res.json()) || [];
    } catch (err) {
      devError('Search failed:', err);
      return [];
    }
  };

  const createNewSemanticTerm = async (termName: string) => {
    try {
  const res = await fetch(resolveApiUrl('/api/semantic-terms'), {
        method: 'POST', headers: { 'Content-Type': 'application/json' }, credentials: 'include',
        body: JSON.stringify({ term_name: termName.toUpperCase(), description: `Custom semantic term: ${termName}` })
      });
      if (!res.ok) throw new Error('Failed to create semantic term');
      return await res.json();
    } catch (err) {
      devError('Failed to create semantic term:', err);
      setToast({ type: 'error', message: 'Failed to create new semantic term' });
      return null;
    }
  };

  const applyCustomMapping = async (mapping: any) => {
    try {
  const res = await fetch(resolveApiUrl('/api/semantic-mappings/apply-custom'), {
        method: 'POST', headers: { 'Content-Type': 'application/json', 'X-Tenant-ID': mapping.database_column.tenant_id, 'X-Tenant-Datasource-ID': mapping.database_column.tenant_tenant_instance_id },
        credentials: 'include',
        body: JSON.stringify({ column_node_id: mapping.database_column.node_id, semantic_term_name: mapping.semantic_term })
      });
      if (!res.ok) throw new Error('Failed to apply custom mapping');
      return true;
    } catch (err) {
      devError('Failed to apply custom mapping:', err);
      setToast({ type: 'error', message: 'Failed to apply custom mapping' });
      return false;
    }
  };

  const createEdges = async (selected: any[]) => {
    if (selected.length === 0) return;
    setLoading(true);
    try {
  const res = await fetch(resolveApiUrl('/api/semantic-mappings/edges'), {
        method: 'POST', headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ mappings: selected })
      });
      const data = await res.json();
      setToast({ type: 'success', message: `Created ${data.created_edges || 0} edges.` });
      await loadMappings(); // Reload to get updated edge_exists status
    } catch (err) {
      devError('Failed to create edges:', err);
      setToast({ type: 'error', message: 'Failed to create edges' });
    }
    setLoading(false);
  };

  const replaceMapping = async (mapping: any) => {
    if (!mapping || !mapping.semantic_term_id) return;
    setLoading(true);
    try {
  await fetch(resolveApiUrl('/api/semantic-mappings/replace'), {
        method: 'POST', headers: { 'Content-Type': 'application/json' }, credentials: 'include',
        body: JSON.stringify({ mapping })
      });
      setToast({ type: 'success', message: 'Mapping replaced.' });
      await loadMappings();
    } catch (err) {
      devError('Replace failed:', err);
      setToast({ type: 'error', message: 'Failed to replace mapping' });
    }
    setLoading(false);
  };

  const persistIgnores = async (ignoredMappings: any[]) => {
    try {
  await fetch(resolveApiUrl('/api/semantic-mappings/ignore'), {
        method: 'POST', headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ mappings: ignoredMappings.map((m) => ({ database_column: m.database_column, semantic_term: m.semantic_term })) })
      });
      setToast({ type: 'success', message: 'Ignored suggestion saved' });
    } catch (err) {
      devWarn('Failed to persist ignores:', err);
      setToast({ type: 'error', message: 'Failed to save ignore' });
    }
  };

  return {
    mappings, setMappings, loading, toast, setToast, editOriginals, setEditOriginals,
    loadMappings, searchSemanticTerms, createNewSemanticTerm, applyCustomMapping,
    createEdges, replaceMapping, persistIgnores
  };
}