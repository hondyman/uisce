import { useState, useEffect, useCallback } from 'react';
import { useScope } from '../../contexts/ScopeContext';
import { getRequiredTenantScope } from '../../utils/tenantScope';
import { devDebug, devWarn, devError } from '../../utils/devLogger';
import resolveApiUrl from '../../utils/resolveApiUrl';
import type { Mapping, DatabaseColumn, SemanticTerm } from './types';

let generatedIdCounter = 0;
const makeUniqueId = (prefix = 'generated') => `${prefix}-${Date.now().toString(36)}-${++generatedIdCounter}`;

export function useSemanticMapper() {
  const [mappings, setMappings] = useState<Mapping[]>([]);
  const [loading, setLoading] = useState(false);
  const [toast, setToast] = useState<{ type: 'success' | 'error' | 'info'; message: string } | null>(null);
  const [editOriginals, setEditOriginals] = useState<Record<string, Partial<Mapping>>>({});

  const {
    schemaIds, tableIds, columnIds,
    schemaNames, tableNames, columnNames
  } = useScope();

  const loadMappings = useCallback(async () => {
    setLoading(true);
    try {
      const generateMappingsForScope = async (): Promise<Mapping[]> => {
        const generatedMappings: Mapping[] = [];
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
                  } as DatabaseColumn,
                  semantic_term: null, confidence: 0, selected: false, ignored: false, override: false
                } as Mapping));
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
      };

      // Add cache-busting timestamp to force fresh data
      const finalUrlObj = new URL(resolveApiUrl('/api/semantic-mappings'));
      finalUrlObj.searchParams.set('_t', String(Date.now()));
      try {
        if ((import.meta as any).env?.DEV) {
          devDebug('[useSemanticMapper] Fetching mappings from URL:', finalUrlObj.toString());
        }
      } catch (e) { }
      let data: Mapping[] = [];
      const { tenantId, datasourceId } = getRequiredTenantScope();

      // Parallel fetch: Valid mappings (catalog) + Pending mappings (local DB)
      const [catalogRes, pendingRes] = await Promise.all([
        fetch(finalUrlObj.toString(), { cache: 'no-store' }),
        fetch(`/api/semantic-mapping/wizard/pending?tenant_id=${tenantId}&tenant_instance_id=${datasourceId}`, { cache: 'no-store' }).catch(() => null)
      ]);

      if (!catalogRes.ok) {
        // Log backend error body (may be plain text) to help debugging
        const text = await catalogRes.text().catch(() => '<no body>');
        devError('[useSemanticMapper] Failed to fetch mappings:', catalogRes.status, text);
        // If we have a scope available, fall back to generating mappings client-side
        if (schemaIds.length > 0 || tableIds.length > 0 || columnIds.length > 0) {
          data = await generateMappingsForScope();
        } else {
          data = [];
        }
      } else {
        const json = await catalogRes.json();
        data = json.mappings || [];
      }

      // Merge pending items
      if (pendingRes && pendingRes.ok) {
        const pendingData = await pendingRes.json();
        const pendingMappings = pendingData.pending_approvals || [];

        // Create a map of existing column IDs to avoid duplicates (prefer existing mapping if edge exists)
        const existingMap = new Map(data.map(m => [m.database_column.node_id, m]));

        pendingMappings.forEach((p: any) => {
          // Only add if not already mapped (edge_exists=true takes precedence)
          const existing = existingMap.get(p.column_id);
          if (!existing || !existing.edge_exists) {
            const pendingMapping: Mapping = {
              id: `pending-${p.id}`,
              database_column: {
                node_id: p.column_id,
                column: p.column_name,
                table: 'unknown', // Backend might need to return this, or we infer/ignore for now
                schema: 'unknown',
                tenant_id: p.tenant_id,
                tenant_tenant_instance_id: p.tenant_instance_id
              },
              semantic_term: p.suggested_semantic_term,
              confidence: p.confidence,
              match_reason: p.reasoning,
              edge_exists: false,
              is_pending: true,
              selected: false,
              override: false
            };

            // If we have an existing skeleton (from scope generation), update it
            if (existing) {
              Object.assign(existing, pendingMapping);
            } else {
              data.push(pendingMapping);
            }
          }
        });
      }
      devDebug('[useSemanticMapper] Loaded mappings:', data.length, 'mappings');
      if (data.length > 0 && (import.meta as any).env?.DEV) {
        devDebug('[useSemanticMapper] Sample mapping:', {
          column: data[0]?.database_column?.column,
          semantic_term: data[0]?.semantic_term,
          semantic_term_id: data[0]?.semantic_term_id,
          edge_exists: data[0]?.edge_exists
        });
      }
      if (data.length === 0 && (schemaIds.length > 0 || tableIds.length > 0 || columnIds.length > 0)) {
        data = await generateMappingsForScope();
      }
      setMappings(data);
    } catch (err) {
      devError('Failed to load mappings:', err);
      setMappings([]);
    }
    setLoading(false);
  }, [schemaIds.join(','), tableIds.join(','), columnIds.join(','), schemaNames.join(','), tableNames.join(','), columnNames.join(',')]);

  useEffect(() => {
    loadMappings();
  }, [loadMappings]);

  const searchSemanticTerms = async (query: string): Promise<SemanticTerm[]> => {
    if (!query || query.length < 2) return [];
    try {
      // Search for both original and uppercase versions to handle case sensitivity
      const originalUrl = resolveApiUrl('/api/semantic-terms/search');
      const uppercaseUrl = resolveApiUrl('/api/semantic-terms/search');
      try {
        if ((import.meta as any).env?.DEV) {
          devDebug('[useSemanticMapper] Searching semantic terms; urls:', originalUrl);
        }
      } catch (e) { }

      const [originalResults, uppercaseResults] = await Promise.all([
        fetch(originalUrl, {
          method: 'POST', headers: { 'Content-Type': 'application/json' }, credentials: 'include',
          body: JSON.stringify({ query, limit: 10 })
        }),
        fetch(uppercaseUrl, {
          method: 'POST', headers: { 'Content-Type': 'application/json' }, credentials: 'include',
          body: JSON.stringify({ query: query.toUpperCase(), limit: 10 })
        })
      ]);

      let originalData: any[] = [];
      let uppercaseData: any[] = [];
      if (!originalResults.ok) {
        const text = await originalResults.text().catch(() => '<no body>');
        devError('[useSemanticMapper] semantic-terms search failed:', originalResults.status, text);
      } else {
        originalData = (await originalResults.json()) || [];
      }
      if (!uppercaseResults.ok) {
        const text = await uppercaseResults.text().catch(() => '<no body>');
        devError('[useSemanticMapper] semantic-terms uppercase search failed:', uppercaseResults.status, text);
      } else {
        uppercaseData = (await uppercaseResults.json()) || [];
      }

      // Combine and deduplicate results
      const combined = [...originalData, ...uppercaseData];
      const unique = combined.filter((item, index, self) =>
        index === self.findIndex(t => t.node_id === item.node_id)
      );

      return unique;
    } catch (err) {
      devError('Search failed:', err);
      return [];
    }
  };

  const createNewSemanticTerm = async (termName: string): Promise<SemanticTerm | null> => {
    try {
      const url = resolveApiUrl('/api/semantic-terms');
      try { if ((import.meta as any).env?.DEV) { devDebug('[useSemanticMapper] Creating semantic term; url:', url); } } catch (e) { }
      const res = await fetch(url, {
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

  const applyCustomMapping = async (mapping: Mapping) => {
    try {
      const headers: Record<string, string> = { 'Content-Type': 'application/json' };
      if (mapping.database_column.tenant_id) headers['X-Tenant-ID'] = mapping.database_column.tenant_id;
      if (mapping.database_column.tenant_tenant_instance_id) headers['X-Tenant-Datasource-ID'] = mapping.database_column.tenant_tenant_instance_id;
      const url = resolveApiUrl('/api/semantic-mappings/apply-custom');
      try { if ((import.meta as any).env?.DEV) { devDebug('[useSemanticMapper] Applying custom mapping; url:', url); } } catch (e) { }
      const res = await fetch(url, {
        method: 'POST', headers, credentials: 'include',
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

  const createEdges = async (selected: Mapping[]) => {
    if (selected.length === 0) {
      devWarn('[useSemanticMapper] createEdges called with 0 selected mappings');
      return;
    }
    setLoading(true);

    // Check if any mappings are overrides (edge already exists or skipped=true)
    const hasOverrides = selected.some(m => m.override || m.edge_exists);

    try {
      let totalCreated = 0;
      let totalDeleted = 0;

      // If we have overrides, use the replace endpoint for each mapping
      if (hasOverrides) {
        devDebug('[useSemanticMapper] Detected override scenario, using replace endpoint');

        for (const mapping of selected) {
          const url = resolveApiUrl('/api/semantic-mappings/replace');

          devDebug('[useSemanticMapper] Replacing edge:', {
            url,
            column: mapping.database_column.column,
            semantic_term: mapping.semantic_term,
            semantic_term_id: mapping.semantic_term_id,
            override: mapping.override,
            edge_exists: mapping.edge_exists
          });

          const payload = { mapping };
          devDebug('[useSemanticMapper] Replace payload:', JSON.stringify(payload, null, 2));

          const res = await fetch(url, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            credentials: 'include',
            body: JSON.stringify(payload)
          });

          if (!res.ok) {
            const errorText = await res.text();
            devError('[useSemanticMapper] Replace failed:', errorText);
            throw new Error(`HTTP ${res.status}: ${errorText}`);
          }

          const data = await res.json();
          devDebug('[useSemanticMapper] Replace response:', data);

          // Backend returns "created" and "deleted", not "created_edges" and "deleted_edges"
          totalCreated += data.created || 0;
          totalDeleted += data.deleted || 0;
        }

        // Show appropriate message based on results
        if (totalCreated > 0) {
          setToast({
            type: 'success',
            message: `Replaced ${totalCreated} edge${totalCreated !== 1 ? 's' : ''} (deleted ${totalDeleted} old edge${totalDeleted !== 1 ? 's' : ''}).`
          });
        } else {
          setToast({
            type: 'error',
            message: `Failed to create edges. Check logs for details.`
          });
        }
        await loadMappings();
        setLoading(false);
        return;
      }

      // Otherwise, use the bulk create endpoint
      const url = resolveApiUrl('/api/semantic-mappings/edges');

      // Get tenant scope for headers
      let tenantId = '';
      let datasourceId = '';
      try {
        const scope = getRequiredTenantScope();
        tenantId = scope.tenantId;
        datasourceId = scope.datasourceId;
      } catch (err) {
        devWarn('[useSemanticMapper] Could not get tenant scope for headers:', err);
      }

      // Log the request details
      devDebug('[useSemanticMapper] Creating edges:', {
        url,
        count: selected.length,
        tenantId,
        datasourceId,
        mappings: selected.map(m => ({
          column: m.database_column.column,
          semantic_term: m.semantic_term,
          semantic_term_id: m.semantic_term_id,
          tenant_id: m.database_column.tenant_id,
          tenant_instance_id: m.database_column.tenant_tenant_instance_id
        }))
      });

      const payload = { mappings: selected };
      devDebug('[useSemanticMapper] Request payload:', JSON.stringify(payload, null, 2));

      const headers: Record<string, string> = { 'Content-Type': 'application/json' };
      if (tenantId) headers['X-Tenant-ID'] = tenantId;
      if (datasourceId) headers['X-Tenant-Datasource-ID'] = datasourceId;

      const res = await fetch(url, {
        method: 'POST',
        headers,
        credentials: 'include',
        body: JSON.stringify(payload)
      });

      devDebug('[useSemanticMapper] Response status:', res.status, res.statusText);

      if (!res.ok) {
        const errorText = await res.text();
        devError('[useSemanticMapper] Error response:', errorText);
        throw new Error(`HTTP ${res.status}: ${errorText}`);
      }

      const data = await res.json();
      devDebug('[useSemanticMapper] Response data:', data);

      // CRITICAL: Log per-mapping details to see why edges weren't created
      if (data.per_mapping_results && data.per_mapping_results.length > 0) {
        devDebug('[useSemanticMapper] ⚠️ Per-mapping results:', JSON.stringify(data.per_mapping_results, null, 2));
        data.per_mapping_results.forEach((result: any, idx: number) => {
          if (result.error) {
            devError(`[useSemanticMapper] ❌ Mapping ${idx} FAILED:`, result.error);
          } else if (result.skipped) {
            devWarn(`[useSemanticMapper] ⏭️  Mapping ${idx} SKIPPED (already exists)`);
          } else if (result.created_edge) {
            devDebug(`[useSemanticMapper] ✅ Mapping ${idx} created successfully`);
          } else {
            devWarn(`[useSemanticMapper] ⚠️  Mapping ${idx} unknown state:`, result);
          }
        });
      } else {
        devError('[useSemanticMapper] ❌ NO per_mapping_results in response!');
      }

      setToast({ type: 'success', message: `Created ${data.created_edges || 0} edges.` });
      await loadMappings(); // Reload to get updated edge_exists status
    } catch (err) {
      devError('[useSemanticMapper] Failed to create edges:', err);
      setToast({ type: 'error', message: `Failed to create edges: ${err instanceof Error ? err.message : 'Unknown error'}` });
    }
    setLoading(false);
  };

  const replaceMapping = async (mapping: Mapping) => {
    if (!mapping || !mapping.semantic_term_id) return;
    setLoading(true);
    try {
      const url = resolveApiUrl('/api/semantic-mappings/replace');
      try { if ((import.meta as any).env?.DEV) { devDebug('[useSemanticMapper] Replacing mapping; url:', url); } } catch (e) { }

      // Get tenant scope for headers
      let tenantId = '';
      let datasourceId = '';
      try {
        const scope = getRequiredTenantScope();
        tenantId = scope.tenantId;
        datasourceId = scope.datasourceId;
      } catch (err) {
        devWarn('[useSemanticMapper] Could not get tenant scope for headers:', err);
      }

      const headers: Record<string, string> = { 'Content-Type': 'application/json' };
      if (tenantId) headers['X-Tenant-ID'] = tenantId;
      if (datasourceId) headers['X-Tenant-Datasource-ID'] = datasourceId;

      await fetch(url, {
        method: 'POST', headers, credentials: 'include',
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

  const persistIgnores = async (ignoredMappings: Mapping[]) => {
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

  const loadSemanticTerms = async (): Promise<SemanticTerm[]> => {
    try {
      const url = resolveApiUrl('/api/semantic-terms');
      const res = await fetch(url, { credentials: 'include' });
      if (!res.ok) throw new Error('Failed to load semantic terms');
      return await res.json();
    } catch (err) {
      devError('Failed to load semantic terms:', err);
      setToast({ type: 'error', message: 'Failed to load semantic terms' });
      return [];
    }
  };

  const loadBusinessTerms = async (): Promise<SemanticTerm[]> => {
    try {
      const url = resolveApiUrl('/api/business-terms');
      const res = await fetch(url, { credentials: 'include' });
      if (!res.ok) throw new Error('Failed to load business terms');
      return await res.json();
    } catch (err) {
      devError('Failed to load business terms:', err);
      setToast({ type: 'error', message: 'Failed to load business terms' });
      return [];
    }
  };

  const searchBusinessTerms = async (query: string): Promise<SemanticTerm[]> => {
    if (!query || query.length < 2) return [];
    try {
      const url = resolveApiUrl('/api/business-terms/search');
      const res = await fetch(url, {
        method: 'POST', headers: { 'Content-Type': 'application/json' }, credentials: 'include',
        body: JSON.stringify({ query, limit: 10 })
      });
      if (!res.ok) throw new Error('Failed to search business terms');
      return await res.json();
    } catch (err) {
      devError('Search failed:', err);
      return [];
    }
  };

  const createNewBusinessTerm = async (termName: string): Promise<SemanticTerm | null> => {
    try {
      const url = resolveApiUrl('/api/business-terms');
      // Backend expects a `term_name` and a `properties` object
      const payload = {
        term_name: termName.toUpperCase(),
        properties: { description: `Custom business term: ${termName}` }
      };
      const res = await fetch(url, {
        method: 'POST', headers: { 'Content-Type': 'application/json' }, credentials: 'include',
        body: JSON.stringify(payload)
      });
      if (!res.ok) throw new Error('Failed to create business term');
      return await res.json();
    } catch (err) {
      devError('Failed to create business term:', err);
      setToast({ type: 'error', message: 'Failed to create new business term' });
      return null;
    }
  };

  const createBusinessTermEdge = async (semanticTermId: string, businessTermId: string) => {
    try {
      const url = resolveApiUrl('/api/business-term-edges');
      const res = await fetch(url, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        credentials: 'include',
        body: JSON.stringify({
          subject_node_id: businessTermId, // Business term as subject
          object_node_id: semanticTermId,  // Semantic term as object
          edge_type_id: '3be9d6ae-1598-4628-a3dd-b606921a9193',
          relationship_type: 'business_term_to_semantic_term'
        })
      });
      if (!res.ok) throw new Error('Failed to create business term edge');
      await res.json();
      setToast({ type: 'success', message: `Created business term edge.` });
    } catch (err) {
      devError('Failed to create business term edge:', err);
      setToast({ type: 'error', message: 'Failed to create business term edge' });
    }
  };

  const updateBusinessTerm = async (termNodeId: string, updates: Record<string, any>): Promise<boolean> => {
    try {
      const url = resolveApiUrl(`/api/business-terms/${termNodeId}`);
      const res = await fetch(url, {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        credentials: 'include',
        body: JSON.stringify(updates)
      });

      if (!res.ok) {
        const errorText = await res.text();
        throw new Error(`HTTP ${res.status}: ${errorText}`);
      }

      await res.json(); // Consume response
      setToast({ type: 'success', message: 'Business term updated successfully' });
      return true;
    } catch (err) {
      devError('Failed to update business term:', err);
      setToast({ type: 'error', message: `Failed to update business term: ${err instanceof Error ? err.message : 'Unknown error'}` });
      return false;
    }
  };

  const upsertBusinessTermAndEdge = async (businessTermName: string, semanticTermId: string) => {
    try {
      const url = resolveApiUrl('/api/semantic-mappings/upsert-business-term-edge');
      const res = await fetch(url, {
        method: 'POST', headers: { 'Content-Type': 'application/json' }, credentials: 'include',
        body: JSON.stringify({ business_term_name: businessTermName, semantic_term_id: semanticTermId })
      });
      if (!res.ok) throw new Error('Failed to upsert business term and edge');
      return await res.json();
    } catch (err) {
      devError('Failed to upsert business term and edge:', err);
      setToast({ type: 'error', message: 'Failed to upsert business term and edge' });
      return null;
    }
  };

  const deleteBusinessTerm = async (termNodeId: string): Promise<boolean> => {
    try {
      const url = resolveApiUrl(`/api/business-terms/${termNodeId}`);
      const res = await fetch(url, {
        method: 'DELETE',
        credentials: 'include',
      });

      if (!res.ok) {
        const errorText = await res.text();
        throw new Error(`HTTP ${res.status}: ${errorText}`);
      }

      setToast({ type: 'success', message: 'Business term deleted successfully' });
      return true;
    } catch (err) {
      devError('Failed to delete business term:', err);
      setToast({ type: 'error', message: `Failed to delete business term: ${err instanceof Error ? err.message : 'Unknown error'}` });
      return false;
    }
  };

  const deleteBusinessTermEdge = async (edgeId: string): Promise<boolean> => {
    try {
      const url = resolveApiUrl(`/api/business-term-edges/${edgeId}`);
      const res = await fetch(url, {
        method: 'DELETE',
        credentials: 'include',
      });

      if (!res.ok) {
        const errorText = await res.text();
        throw new Error(`HTTP ${res.status}: ${errorText}`);
      }

      setToast({ type: 'success', message: 'Business term mapping deleted successfully' });
      return true;
    } catch (err) {
      devError('Failed to delete business term edge:', err);
      setToast({ type: 'error', message: `Failed to delete mapping: ${err instanceof Error ? err.message : 'Unknown error'}` });
      return false;
    }
  };

  const deleteBusinessTermEdgeByTerms = async (semanticTermId: string, businessTermId: string): Promise<boolean> => {
    try {
      const urlObj = new URL(resolveApiUrl('/api/business-term-edges'));
      urlObj.searchParams.set('semantic_term_id', semanticTermId);
      urlObj.searchParams.set('business_term_id', businessTermId);
      const res = await fetch(urlObj.toString(), {
        method: 'DELETE',
        credentials: 'include',
      });

      if (!res.ok) {
        const errorText = await res.text();
        throw new Error(`HTTP ${res.status}: ${errorText}`);
      }

      setToast({ type: 'success', message: 'Business term mapping deleted successfully' });
      return true;
    } catch (err) {
      devError('Failed to delete business term edge:', err);
      setToast({ type: 'error', message: `Failed to delete mapping: ${err instanceof Error ? err.message : 'Unknown error'}` });
      return false;
    }
  };

  const recordSuggestionFeedback = async (
    semanticTermId: string,
    businessTermName: string,
    action: 'accept' | 'reject',
    businessTermId?: string,
    confidence?: number,
    reason?: string
  ): Promise<boolean> => {
    try {
      const url = resolveApiUrl('/api/business-term/suggestion-feedback');
      const res = await fetch(url, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        credentials: 'include',
        body: JSON.stringify({
          semantic_term_id: semanticTermId,
          business_term_id: businessTermId,
          business_term_name: businessTermName,
          action,
          confidence,
          reason
        })
      });

      if (!res.ok) {
        const errorText = await res.text();
        throw new Error(`HTTP ${res.status}: ${errorText}`);
      }

      return true;
    } catch (err) {
      devError('Failed to record suggestion feedback:', err);
      // Don't show toast for feedback errors - fail silently
      return false;
    }
  };

  return {
    mappings, setMappings, loading, toast, setToast, editOriginals, setEditOriginals,
    loadMappings, searchSemanticTerms, createNewSemanticTerm, applyCustomMapping,
    createEdges, replaceMapping, persistIgnores,
    loadSemanticTerms, loadBusinessTerms, searchBusinessTerms, createNewBusinessTerm,
    updateBusinessTerm, createBusinessTermEdge, upsertBusinessTermAndEdge,
    deleteBusinessTerm, deleteBusinessTermEdge, deleteBusinessTermEdgeByTerms,
    recordSuggestionFeedback
  };
}