import React, { useState, useMemo, useEffect } from 'react';
import { useTenant } from '../../contexts/TenantContext';
import { useAccess } from '../../contexts/AccessContext';
import BusinessTermsTree from '../../components/BusinessTermsTree';
import { EnhancedSelectedAsset } from '../../types/SemanticTypes';
import SemanticFlow from '../../components/SemanticFlow';
import { CatalogNode, useBusinessTerms, useSemanticTerms, useGlossaryEdges } from '../../api/glossary';
import { useNodeTypes } from '../../api/nodeTypes';
import { usePropertyLookupMaps } from '../../hooks/usePropertyLookupMaps';
import { enrichNodesWithTypes } from '../../utils/nodeTypeMapping';
import { Tabs, Tab, Box, IconButton, Tooltip, Chip } from '@mui/material';
import { Add as AddIcon, ArrowBack as ArrowBackIcon, AutoAwesome } from '@mui/icons-material';
import { SemanticEnrichmentWizard } from '../../components/SemanticEnrichmentWizard';
import { RelationshipExplorer } from '../../features/glossary/components/RelationshipExplorer';
import { Button } from '@mui/material';
import './BusinessTermsTab.css';
import { useTranslation } from 'react-i18next';
import { devDebug } from '../../utils/devLogger';

export const BusinessTermsTab: React.FC<{
  scopeTenantId?: string;
  searchTerm?: string;
  onCreateTerm?: () => void;
  onEditTerm?: (term: CatalogNode) => void;
  onDeleteTerm?: (term: CatalogNode) => void;
  selectedBusinessTerm?: CatalogNode | null;
}> = ({ scopeTenantId, searchTerm, onCreateTerm, onEditTerm, onDeleteTerm, selectedBusinessTerm }) => {
  const { datasource, tenant } = useTenant();
  const { currentDatasource } = useAccess();
  const effectiveDatasourceId = currentDatasource?.id ?? datasource?.id;
  const [selectedAsset, setSelectedAsset] = useState<EnhancedSelectedAsset | null>(null);
  const [highlightedItem, setHighlightedItem] = useState<string | null>(null);
  const [activeDetailTab, setActiveDetailTab] = useState<'details' | 'lineage'>('details');
  const [selectedSemanticTerm, setSelectedSemanticTerm] = useState<any | null>(null);
  const [filterType, setFilterType] = useState<'all' | 'with_relationships' | 'without_relationships'>('all');
  const [wizardOpen, setWizardOpen] = useState(false);

  const { t } = useTranslation();

  // REST-only data flow (GraphQL removed per user request — it was returning
  // empty data because the remote Hasura either doesn't have the right
  // datasource_id mapping or wasn't being reached with the correct JWT).
  // useBusinessTerms() hits /api/catalog/nodes (the local semlayer-backend,
  // proxied via Vite) and filters client-side for catalog_type='business_term'.
  // useSemanticTerms() hits /api/glossary/semantic-terms.
  // useGlossaryEdges() fetches business/semantic term edges from
  // /api/glossary/edges.
  // scopeTenantId (if provided) drives the queries; otherwise the active tenant.
  // Semantic/business terms are scoped to the TENANT (no datasource param).
  const { data: rqBusinessTerms, isLoading: rqBusinessLoading, error: rqBusinessError, refetch } = useBusinessTerms({ tenantOverride: scopeTenantId });
  const { data: rqSemanticTerms, isLoading: rqSemanticLoading, error: rqSemanticError } = useSemanticTerms({ tenantOverride: scopeTenantId });
  const { data: rqEdges, isLoading: rqEdgesLoading, refetch: refetchEdges } = useGlossaryEdges({ tenantOverride: scopeTenantId });

  const { data: nodeTypes } = useNodeTypes({ tenantId: scopeTenantId });

  const selectedNodeType = useMemo(() => {
    if (!nodeTypes || !selectedAsset?.node?.node_type_id) return null;
    const match = (nodeTypes as any[]).find(nt => nt.id === selectedAsset.node.node_type_id);
    return match;
  }, [nodeTypes, selectedAsset]);

  // FALLBACK: Find the business_term node type by name (like BusinessTermsTree does)
  // We do this unconditionally so that top-level lookup maps for categories are available
  // even when no business term is currently selected.
  const businessTermNodeTypeByName = useMemo(() => {
    if (!nodeTypes) return null;
    const name = 'business_term';
    return (nodeTypes as any[]).find((nt) => {
      const ntName = String(nt.catalog_type_name || '').toLowerCase();
      return ntName === name || ntName === 'business term' || ntName.includes('business_term') || ntName.includes('business term');
    }) || null;
  }, [nodeTypes]);

  // Use the selectedNodeType if available, otherwise fall back to businessTermNodeTypeByName for lookups
  const effectiveNodeTypeForLookups = selectedAsset?.type === 'business_term'
    ? (businessTermNodeTypeByName || selectedNodeType)
    : (selectedNodeType || businessTermNodeTypeByName);

  // Pre-load lookup maps for properties with lookup_id on the selected nodeType
  const lookupMaps = usePropertyLookupMaps(effectiveNodeTypeForLookups, selectedAsset?.node?.properties);
  // Also fetch top-level lookup maps (no assetProperties) so we have lookup labels
  // available even if cascade parent value isn't present.
  const topLevelLookupMaps = usePropertyLookupMaps(businessTermNodeTypeByName);

  // If the parent page passes in a selected business term (e.g. via parent link click), pick it here
  useEffect(() => {
    if (!selectedBusinessTerm) return;
    setSelectedAsset({
      id: selectedBusinessTerm.id,
      name: selectedBusinessTerm.node_name || 'Untitled',
      type: 'business_term',
      nodeId: selectedBusinessTerm.id,
      node: selectedBusinessTerm,
    });
    setHighlightedItem(selectedBusinessTerm.id);
  }, [selectedBusinessTerm]);

  // REST-only data: business_terms and semantic_terms from /api/catalog/nodes
  // and /api/glossary/semantic-terms. Edges come from /api/glossary/edges.
  const businessTerms = enrichNodesWithTypes(rqBusinessTerms || []);
  const semanticTerms = enrichNodesWithTypes(rqSemanticTerms || []);
  const semanticViews: CatalogNode[] = []; // No REST endpoint wired yet
  const glossaryEdges = rqEdges || [];

  // Debug logging for lookup maps
  useEffect(() => {
    devDebug('[BusinessTermsTab] ====== DEBUG CHECKPOINT ======');
    devDebug('[BusinessTermsTab] tenant?.id:', tenant?.id);
    devDebug('[BusinessTermsTab] datasource?.id:', datasource?.id);
    devDebug('[BusinessTermsTab] nodeTypes count:', (nodeTypes as any[])?.length || 0);
    devDebug('[BusinessTermsTab] businessTerms count:', (rqBusinessTerms || []).length);
    devDebug('[BusinessTermsTab] semanticTerms count:', (rqSemanticTerms || []).length);
    devDebug('[BusinessTermsTab] rqBusinessError:', rqBusinessError?.message);
    devDebug('[BusinessTermsTab] rqSemanticError:', rqSemanticError?.message);
    devDebug('[BusinessTermsTab] selectedAsset?.node?.node_type_id:', selectedAsset?.node?.node_type_id);
    devDebug('[BusinessTermsTab] selectedNodeType exists:', !!selectedNodeType, selectedNodeType?.catalog_type_name);
    devDebug('[BusinessTermsTab] businessTermNodeTypeByName exists:', !!businessTermNodeTypeByName, businessTermNodeTypeByName?.catalog_type_name);
    devDebug('[BusinessTermsTab] effectiveNodeTypeForLookups:', effectiveNodeTypeForLookups?.id, effectiveNodeTypeForLookups?.catalog_type_name);
    if (effectiveNodeTypeForLookups) {
      devDebug('[BusinessTermsTab] effectiveNodeTypeForLookups.properties count:', effectiveNodeTypeForLookups.properties?.length || 0);
      const propsWithLookups = (effectiveNodeTypeForLookups.properties as any[])?.filter(p => p.lookup_id) || [];
      devDebug('[BusinessTermsTab] Properties with lookup_id:', propsWithLookups.map((p: any) => ({name: p.name, lookup_id: p.lookup_id})));
      devDebug('[BusinessTermsTab] lookupMaps keys:', Object.keys(lookupMaps), `(${Object.keys(lookupMaps).length} keys)`);
      devDebug('[BusinessTermsTab] topLevelLookupMaps keys:', Object.keys(topLevelLookupMaps), `(${Object.keys(topLevelLookupMaps).length} keys)`);
      Object.entries(lookupMaps).forEach(([k, v]) => {
        devDebug(`[BusinessTermsTab] lookupMaps['${k}'] size:`, v.size);
        if (v.size > 0) {
          const first3 = Array.from(v.entries()).slice(0, 3);
          devDebug(`[BusinessTermsTab]   First 3 entries:`, first3);
        }
      });
      Object.entries(topLevelLookupMaps).forEach(([k, v]) => {
        devDebug(`[BusinessTermsTab] topLevelLookupMaps['${k}'] size:`, v.size);
        if (v.size > 0) {
          const first3 = Array.from(v.entries()).slice(0, 3);
          devDebug(`[BusinessTermsTab]   First 3 entries:`, first3);
        }
      });
    }
    devDebug('[BusinessTermsTab] ====== END CHECKPOINT ======');
  }, [selectedNodeType, businessTermNodeTypeByName, effectiveNodeTypeForLookups, lookupMaps, topLevelLookupMaps, rqBusinessTerms, rqSemanticTerms, rqBusinessError, rqSemanticError]);

  // Helper to resolve category property values (UUIDs) to labels
  const resolveCategoryValue = (propKeys: string[], val: any): string | null => {
    if (!val) return null;
    const strVal = String(val);
    const uuidRegex = /^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$/;
    if (!uuidRegex.test(strVal)) return strVal; // not a UUID so return as-is

    // Try candidate property names in lookupMaps
    for (const k of propKeys) {
      if (lookupMaps[k] && lookupMaps[k].has(strVal)) {
        return lookupMaps[k].get(strVal) || null;
      }
    }

    // Try any lookup map if candidate keys didn't match
    for (const anyKey of Object.keys(lookupMaps)) {
      if (lookupMaps[anyKey]?.has(strVal)) return lookupMaps[anyKey].get(strVal) || null;
    }
    // Try top-level lookup maps
    for (const anyKey of Object.keys(topLevelLookupMaps)) {
      if (topLevelLookupMaps[anyKey]?.has(strVal)) return topLevelLookupMaps[anyKey].get(strVal) || null;
    }

    // Fallback to nodeNameMap
    if (nodeNameMap[strVal]) {
      return nodeNameMap[strVal];
    }

    // If not found, show short fallback
    return `Unknown (${strVal.substring(0, 8)}...)`;
  };

  const selectedSemanticTermNodeType = useMemo(() => {
    if (!nodeTypes || !selectedSemanticTerm?.node_type_id) return null;
    return (nodeTypes as any[]).find(nt => nt.id === selectedSemanticTerm.node_type_id);
  }, [nodeTypes, selectedSemanticTerm]);

  // Create a mapping of node IDs to node names for both business terms and semantic terms
  const nodeNameMap = useMemo(() => {
    const map: { [key: string]: string } = {};

    // Add business terms to the map
    businessTerms.forEach((term: any) => {
      map[term.id] = term.node_name || 'Unknown Business Term';
    });

    // Add semantic terms to the map
    semanticTerms.forEach((term: any) => {
      map[term.id] = term.node_name || 'Unknown Semantic Term';
    });

    // Add semantic columns to the map (these are the actual semantic terms referenced in edges)
    semanticViews.forEach((column: any) => {
      map[column.id] = column.node_name || 'Unknown Semantic Column';
    });

    // Also include all nodes from glossary edges to ensure complete coverage
    glossaryEdges.forEach((edge: any) => {
      if (edge.source_node_id && !map[edge.source_node_id]) {
        map[edge.source_node_id] = edge.source_node_id.substring(0, 8) + '...';
      }
      if (edge.target_node_id && !map[edge.target_node_id]) {
        map[edge.target_node_id] = edge.target_node_id.substring(0, 8) + '...';
      }
    });

    return map;
  }, [businessTerms, semanticTerms, semanticViews, glossaryEdges]);

  // Create a mapping of node IDs to qualified paths (for fallback in relationships table)
  const nodePathMap = useMemo(() => {
    const map: { [key: string]: string } = {};

    businessTerms.forEach((term: any) => {
      if (term.qualified_path) {
        map[term.id] = term.qualified_path;
      }
    });

    semanticTerms.forEach((term: any) => {
      if (term.qualified_path) {
        map[term.id] = term.qualified_path;
      }
    });

    semanticViews.forEach((column: any) => {
      if (column.qualified_path) {
        map[column.id] = column.qualified_path;
      }
    });

    return map;
  }, [businessTerms, semanticTerms, semanticViews]);

  // Filter semantic edges to only show those connected to the selected business term
  const relatedEdges = useMemo(() => {
    if (!selectedAsset || !glossaryEdges.length) return [];

    return glossaryEdges.filter((edge: any) =>
      edge.source_node_id === selectedAsset.nodeId || edge.target_node_id === selectedAsset.nodeId
    );
  }, [selectedAsset, glossaryEdges]);

  // Calculate statistics for business terms
  const statistics = useMemo(() => {
    if (!businessTerms || businessTerms.length === 0) return { total: 0, withRelationships: 0, withoutRelationships: 0 };

    const total = businessTerms.length;
    const businessTermIds = new Set(businessTerms.map((term: any) => term.id));

    // Find business terms that have relationships (connected to semantic terms)
    const termsWithRelationships = new Set<string>();
    const edges = glossaryEdges || [];
    edges.forEach((edge: any) => {
      if (edge.relationship_type === 'business_term_to_semantic_term') {
        if (businessTermIds.has(edge.source_node_id)) {
          termsWithRelationships.add(edge.source_node_id);
        }
      }
    });

    const withRelationships = termsWithRelationships.size;
    const withoutRelationships = total - withRelationships;

    return { total, withRelationships, withoutRelationships };
  }, [businessTerms, glossaryEdges]);

  const handleAssetSelect = (asset: EnhancedSelectedAsset) => {
    setSelectedAsset(asset);
    setHighlightedItem(asset.id);
  };

  const handleBackToSplash = () => {
    setSelectedAsset(null);
    setHighlightedItem(null);
    setSelectedSemanticTerm(null);
  };

  const handleClearFilter = () => {
    setFilterType('all');
  };

  const anyLoading = (rqBusinessLoading || rqSemanticLoading || rqEdgesLoading);

  if (anyLoading) {
    return <div>{t('global_loading.business_terms', 'Loading business terms...')}</div>;
  }

  return (
    <div className="business-terms-tab-container">
      {/* Header with Add Button */}
      <div className="business-terms-header">
        <h3>{t('tab.business_terms', 'Business Terms')}</h3>
        {onCreateTerm && (
          <Box display="flex" gap={1}>
            <Button
              variant="outlined"
              size="small"
              startIcon={<AutoAwesome />}
              onClick={() => setWizardOpen(true)}
            >
              Enrichment Wizard
            </Button>
            <Tooltip title={t('tab.add_business_term', 'Add New Business Term')}>
              <IconButton
                size="small"
                onClick={onCreateTerm}
                className="add-term-button"
              >
                <AddIcon fontSize="small" />
              </IconButton>
            </Tooltip>
          </Box>
        )}
      </div>

      <div className="business-terms-content">
        <div className="business-terms-sidebar">
          <BusinessTermsTree
            businessTerms={businessTerms}
            semanticTerms={[]}
            semanticViews={[]}
            semanticEdges={glossaryEdges}
            selectedAsset={selectedAsset}
            onAssetSelect={handleAssetSelect}
            highlightedItem={highlightedItem}
            searchTerm={searchTerm}
            filterType={filterType}
            onEditTerm={onEditTerm}
            onDeleteTerm={onDeleteTerm}
            tenantId={scopeTenantId}
          />
        </div>

        <div className="business-terms-main">
          {selectedAsset ? (
            <div className="business-term-details">
              {/* Header with back button and filter chip */}
              <div className="business-term-header">
                <div className="header-left">
                  <Tooltip title={t('back.to_overview', 'Back to overview')}>
                    <IconButton
                      size="small"
                      onClick={handleBackToSplash}
                      className="back-button"
                    >
                      <ArrowBackIcon fontSize="small" />
                    </IconButton>
                  </Tooltip>
                  <h2>{selectedAsset.name}</h2>
                </div>
                {filterType !== 'all' && (
                      <Chip
                      label={`${t('filters.' + (filterType === 'with_relationships' ? 'with_relationships' : 'without_relationships'), filterType === 'with_relationships' ? 'With Relationships' : 'Without Relationships')}`}
                    onDelete={handleClearFilter}
                    size="small"
                    color="primary"
                    variant="outlined"
                  />
                )}
              </div>

              <Box sx={{ borderBottom: 1, borderColor: 'divider', marginBottom: 2 }}>
                <Tabs value={activeDetailTab} onChange={(_, newValue) => setActiveDetailTab(newValue)}>
                  <Tab label="Details" value="details" />
                  <Tab label="Lineage" value="lineage" />
                </Tabs>
              </Box>

              {activeDetailTab === 'details' && (
                <div className="details-tab-content">
                  {selectedAsset.node?.description && (
                    <div className="business-term-description">
                      <h3>Description</h3>
                      <p>{selectedAsset.node.description}</p>
                    </div>
                  )}

                  <div className="business-term-metadata">
                    <h3>Metadata</h3>
                    <div className="metadata-grid">
                      <div className="metadata-item">
                        <span className="metadata-label">Type:</span>
                        <span className="metadata-value">{selectedAsset.type}</span>
                      </div>
                      <div className="metadata-item">
                        <span className="metadata-label">ID:</span>
                        <span className="metadata-value">{selectedAsset.nodeId}</span>
                      </div>
                      {(() => {
                        if (!selectedAsset.node?.properties || Object.keys(selectedAsset.node.properties).length === 0) {
                          devDebug(`[BusinessTermsTab] No properties on selectedAsset.node`);
                          return null;
                        }

                        const assetProperties = selectedAsset.node.properties;
                        devDebug('%c[METADATA RENDER] assetProperties:', 'color: purple; font-weight: bold;', assetProperties);
                        devDebug(`[BusinessTermsTab] assetProperties:`, assetProperties);

                        // Use selectedNodeType if available, otherwise try effectiveNodeTypeForLookups as fallback
                        const nodeTypeForMetadata = selectedNodeType || effectiveNodeTypeForLookups;
                        devDebug('%c[METADATA RENDER] nodeTypeForMetadata:', 'color: purple; font-weight: bold;', nodeTypeForMetadata?.catalog_type_name, 'has', nodeTypeForMetadata?.properties?.length, 'properties');
                        if (nodeTypeForMetadata && nodeTypeForMetadata.properties) {
                          const metadataMap = new Map((nodeTypeForMetadata.properties as any[]).map((p: any) => [p.name, p]));
                          // Filter to only include keys with non-null values
                          const propertyKeys = Object.keys(assetProperties).filter(key =>
                            metadataMap.has(key) && assetProperties[key] !== null && assetProperties[key] !== undefined
                          );

                          devDebug(`[BusinessTermsTab] nodeTypeForMetadata.properties:`, nodeTypeForMetadata.properties);
                          devDebug(`[BusinessTermsTab] assetProperties keys: ${Object.keys(assetProperties).join(', ')}`);
                          devDebug(`[BusinessTermsTab] metadataMap keys: ${Array.from(metadataMap.keys()).join(', ')}`);
                          devDebug(`[BusinessTermsTab] propertyKeys (filtered): ${propertyKeys.join(', ')}`);

                          propertyKeys.sort((a, b) => {
                            const metaA = metadataMap.get(a);
                            const metaB = metadataMap.get(b);
                            return (metaA?.order ?? 999) - (metaB?.order ?? 999);
                          });

                          return propertyKeys.map(key => {
                            const metadata = metadataMap.get(key);
                            const value = assetProperties[key];

                            let displayValue: React.ReactNode = String(value);
                            // If this property maps to a lookup, try to resolve the id to a name
                            const propName = metadata?.name || key;
                            const uuidRegex = /^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$/;

                            // Special handling for 'type' field - resolve to node type name
                            if (key === 'type' && typeof value === 'string' && uuidRegex.test(value) && nodeTypes) {
                              const typeNode = (nodeTypes as any[]).find((nt: any) => nt.id === value);
                              if (typeNode) {
                                displayValue = typeNode.catalog_type_name || typeNode.name || value;
                              }
                            } else if (typeof value === 'string' && uuidRegex.test(value)) {
                              devDebug(`%c[UUID RESOLUTION] key='${key}', value='${value}'`, 'color: red; font-weight: bold;');
                              devDebug(`%c  lookupMaps keys: ${Object.keys(lookupMaps).join(', ')}`, 'color: orange;');
                              devDebug(`%c  topLevelLookupMaps keys: ${Object.keys(topLevelLookupMaps).join(', ')}`, 'color: orange;');
                              devDebug(`\n[BusinessTermsTab] >>>>>> RESOLVING: key='${key}', propName='${propName}', value='${value}'`);
                              devDebug(`[BusinessTermsTab] Available lookupMaps keys:`, Object.keys(lookupMaps));
                              devDebug(`[BusinessTermsTab] Available topLevelLookupMaps keys:`, Object.keys(topLevelLookupMaps));

                              // Try multiple candidate keys for the same property (backwards compatibility)
                              const keyCandidates = [propName, key];
                              if (key.includes('category')) {
                                keyCandidates.push('category', 'category_1', 'category_2', 'category_3', 'category1', 'category2', 'category3', 'category_level_1', 'category_level_2', 'category_level_3', 'sub_category');
                              }
                              devDebug(`[BusinessTermsTab] Candidate keys to try:`, keyCandidates);


                              let resolved = null;
                                                        for (const candidateKey of keyCandidates) {
                                                          devDebug(`[BusinessTermsTab]   Trying candidate: '${candidateKey}'`);
                                                          if (lookupMaps[candidateKey]) {
                                                            const hasIt = lookupMaps[candidateKey].has(value);
                                                            devDebug(`[BusinessTermsTab]     lookupMaps['${candidateKey}'].has('${value}'): ${hasIt}, size: ${lookupMaps[candidateKey].size}`);
                                                            if (hasIt) {
                                                              resolved = lookupMaps[candidateKey].get(value);
                                                              devDebug(`[BusinessTermsTab] ✓ FOUND in lookupMaps['${candidateKey}']: ${resolved}`);
                                                              break;
                                                            }
                                                          }
                                                          // Try top-level lookup maps too
                                                          if (!resolved && topLevelLookupMaps[candidateKey]) {
                                                            const hasIt = topLevelLookupMaps[candidateKey].has(value);
                                                            devDebug(`[BusinessTermsTab]     topLevelLookupMaps['${candidateKey}'].has('${value}'): ${hasIt}, size: ${topLevelLookupMaps[candidateKey].size}`);
                                                            if (hasIt) {
                                                              resolved = topLevelLookupMaps[candidateKey].get(value);
                                                              devDebug(`[BusinessTermsTab] ✓ FOUND in topLevelLookupMaps['${candidateKey}']: ${resolved}`);
                                                              break;
                                                            }
                                                          }
                              }
                                // Try any lookup map key (fallback) in the property maps
                                if (!resolved) {
                                  devDebug(`[BusinessTermsTab] No candidate matched, trying ANY key in lookupMaps...`);
                                  for (const anyKey of Object.keys(lookupMaps)) {
                                    if (lookupMaps[anyKey]?.has(value)) {
                                      resolved = lookupMaps[anyKey].get(value);
                                      devDebug(`[BusinessTermsTab] ✓ FOUND in lookupMaps['${anyKey}'] (any-key): ${resolved}`);
                                      break;
                                    }
                                  }
                                }
                                // Try any top-level lookup map key (fallback)
                                if (!resolved) {
                                  devDebug(`[BusinessTermsTab] Still not resolved, trying ANY key in topLevelLookupMaps...`);
                                  for (const anyKey of Object.keys(topLevelLookupMaps)) {
                                    if (topLevelLookupMaps[anyKey]?.has(value)) {
                                      resolved = topLevelLookupMaps[anyKey].get(value);
                                      devDebug(`[BusinessTermsTab] ✓ FOUND in topLevelLookupMaps['${anyKey}'] (any-key): ${resolved}`);
                                      break;
                                    }
                                  }
                                }
                              // Fallback: if not found in property lookup maps, use nodeNameMap to resolve if available
                              if (!resolved && nodeNameMap[value]) {
                                resolved = nodeNameMap[value];
                                devDebug(`[BusinessTermsTab] ✓ Resolved via nodeNameMap: ${resolved}`);
                              }

                              // If still not resolved and this is a category (or labelled as a category), try the dedicated resolver
                              const labelLower = String(metadata?.label || '').toLowerCase();
                              if (!resolved && (key.toLowerCase().includes('category') || labelLower.includes('category'))) {
                                devDebug(`[BusinessTermsTab] Category field but still unresolved, trying resolveCategoryValue...`);
                                const catResolved = resolveCategoryValue(['category_1', 'category1', 'category_level_1', 'category'], value);
                                if (catResolved && !catResolved.includes('Unknown')) {
                                  resolved = catResolved;
                                  devDebug(`[BusinessTermsTab] ✓ Resolved via resolveCategoryValue: ${resolved}`);
                                }
                              }

                              if (resolved) {
                                devDebug(`[BusinessTermsTab] FINAL RESULT: ${resolved} <<<<<<\n`);
                                displayValue = resolved;
                              } else {
                                devDebug(`[BusinessTermsTab] FINAL RESULT: NOT FOUND, showing Unknown <<<<<<\n`);
                                displayValue = `Unknown (${value.substring(0, 8)}...)`;
                              }
                              if (resolved) displayValue = resolved;
                            }

                            // Handle other data types (not type field, not UUID resolution)
                            if (displayValue === String(value)) {
                              // displayValue wasn't set by special handling above
                              if (metadata?.data_type === 'boolean') {
                                displayValue = value ? 'Yes' : 'No';
                              } else if (typeof value === 'object') {
                                displayValue = <pre>{JSON.stringify(value, null, 2)}</pre>;
                              }
                            }

                            return (
                              <div className="metadata-item" key={key}>
                                <span className="metadata-label">{metadata?.label || key}:</span>
                                <span className="metadata-value">{displayValue}</span>
                              </div>
                            );
                          });
                        }

                        // Fallback for when there's no metadata
                        return Object.entries(assetProperties).map(([key, value]) => (
                          <div className="metadata-item" key={key}>
                            <span className="metadata-label">{key}:</span>
                            <span className="metadata-value">{String(value)}</span>
                          </div>
                        ));
                      })()}
                    </div>
                  </div>

                  {/* Relationships Section — PR 4: migrated to RelationshipExplorer */}
                  <div className="business-term-relationships">
                    <h3>Relationships</h3>
                    {selectedAsset?.node ? (
                      <RelationshipExplorer
                        entityType="business_term"
                        entityId={selectedAsset.nodeId}
                        focalNode={selectedAsset.node}
                        onMutated={refetchEdges}
                      />
                    ) : (
                      <div className="no-relationships">{t('relationships_table.none_found', 'No relationships found for this term')}</div>
                    )}
                  </div>
                </div>
              )}

              {activeDetailTab === 'lineage' && (
                <div className="lineage-tab-content">
                  <div className="lineage-visualization">
                    <SemanticFlow
                      centerNode={selectedAsset.node}
                      allNodes={[...businessTerms, ...semanticTerms, ...semanticViews]}
                      semanticEdges={glossaryEdges}
                      onNodeClick={setSelectedSemanticTerm}
                    />
                  </div>
                  <div className="lineage-properties-sidebar">
                    {selectedSemanticTerm ? (
                      <div className="selected-semantic-detail">
                        <h3>{selectedSemanticTerm.node_name}</h3>
                        {selectedSemanticTerm.description && <p>{selectedSemanticTerm.description}</p>}
                        <h4>Properties</h4>
                        <div className="metadata-grid">
                          {(() => {
                            if (!selectedSemanticTerm.properties || Object.keys(selectedSemanticTerm.properties).length === 0) {
                              return <p>No properties</p>;
                            }

                            const assetProperties = selectedSemanticTerm.properties;

                            if (selectedSemanticTermNodeType && selectedSemanticTermNodeType.properties) {
                              const metadataMap = new Map((selectedSemanticTermNodeType.properties as any[]).map((p: any) => [p.name, p]));
                              const propertyKeys = Object.keys(assetProperties).filter(key => metadataMap.has(key));

                              propertyKeys.sort((a, b) => {
                                const metaA = metadataMap.get(a);
                                const metaB = metadataMap.get(b);
                                return (metaA?.order ?? 999) - (metaB?.order ?? 999);
                              });

                              return propertyKeys.map(key => {
                                const metadata = metadataMap.get(key);
                                const value = assetProperties[key];

                                let displayValue: React.ReactNode = String(value);
                                const uuidRegex = /^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$/;

                                if (typeof value === 'string' && uuidRegex.test(value)) {
                                  // Try to resolve UUID values
                                  let resolved = null;
                                  const propName = metadata?.name || key;

                                  // Special handling for 'type' field - first try to resolve to node type name
                                  if (key === 'type' && nodeTypes) {
                                    const typeNode = (nodeTypes as any[]).find((nt: any) => nt.id === value);
                                    if (typeNode) {
                                      resolved = typeNode.title || typeNode.catalog_type_name || typeNode.name || typeNode.id;
                                    }
                                  }

                                  // If 'type' field wasn't resolved via nodeTypes, try the lookup maps
                                  if (!resolved && (key === 'type' || metadata?.input_type === 'lookup' || metadata?.lookup_id)) {
                                    const keyCandidates = [propName, key, 'type'];
                                    for (const candidateKey of keyCandidates) {
                                      if (lookupMaps[candidateKey] && lookupMaps[candidateKey].has(value)) {
                                        resolved = lookupMaps[candidateKey].get(value);
                                        break;
                                      }
                                      if (topLevelLookupMaps[candidateKey] && topLevelLookupMaps[candidateKey].has(value)) {
                                        resolved = topLevelLookupMaps[candidateKey].get(value);
                                        break;
                                      }
                                    }
                                  }

                                  // General fallback for any UUID field with lookup metadata
                                  if (!resolved && metadata?.input_type === 'lookup' && metadata?.lookup_id) {
                                    const keyCandidates = [propName, key];
                                    for (const candidateKey of keyCandidates) {
                                      if (lookupMaps[candidateKey] && lookupMaps[candidateKey].has(value)) {
                                        resolved = lookupMaps[candidateKey].get(value);
                                        break;
                                      }
                                      if (topLevelLookupMaps[candidateKey] && topLevelLookupMaps[candidateKey].has(value)) {
                                        resolved = topLevelLookupMaps[candidateKey].get(value);
                                        break;
                                      }
                                    }
                                  }

                                  // Fallback: try any lookup map
                                  if (!resolved) {
                                    for (const anyKey of Object.keys(lookupMaps)) {
                                      if (lookupMaps[anyKey]?.has(value)) {
                                        resolved = lookupMaps[anyKey].get(value);
                                        break;
                                      }
                                    }
                                  }

                                  // Fallback: try any top-level lookup map
                                  if (!resolved) {
                                    for (const anyKey of Object.keys(topLevelLookupMaps)) {
                                      if (topLevelLookupMaps[anyKey]?.has(value)) {
                                        resolved = topLevelLookupMaps[anyKey].get(value);
                                        break;
                                      }
                                    }
                                  }

                                  // Fallback: try to resolve any UUID against nodeTypes
                                  if (!resolved && typeof value === 'string' && nodeTypes) {
                                     const typeNode = (nodeTypes as any[]).find((nt: any) => nt.id === value);
                                     if (typeNode) {
                                       resolved = typeNode.title || typeNode.catalog_type_name || typeNode.name || typeNode.id;
                                     }
                                  }

                                  if (resolved) {
                                    displayValue = resolved;
                                  }
                                } else if (value === null || value === undefined) {
                                  displayValue = <span className="not-set">Not set</span>;
                                } else if (metadata?.data_type === 'boolean') {
                                  displayValue = value ? 'Yes' : 'No';
                                } else if (typeof value === 'object') {
                                  displayValue = <pre>{JSON.stringify(value, null, 2)}</pre>;
                                }

                                return (
                                  <div className="metadata-item" key={key}>
                                    <span className="metadata-label">{metadata?.title || metadata?.label || key}:</span>
                                    <span className="metadata-value">{displayValue}</span>
                                  </div>
                                );
                              });
                            }

                            // Fallback for when there's no metadata
                            return Object.entries(assetProperties).map(([key, value]) => (
                              <div className="metadata-item" key={key}>
                                <span className="metadata-label">{key}:</span>
                                <span className="metadata-value">{String(value)}</span>
                              </div>
                            ));
                          })()}
                        </div>
                      </div>
                    ) : (
                      <div className="sidebar-placeholder">
                        <p>Click on a node in the lineage diagram to view its properties here.</p>
                      </div>
                    )}
                  </div>
                </div>
              )}
            </div>
          ) : (
            <div className="statistics-tiles">
              <div
                className={`stat-tile ${filterType === 'all' ? 'active' : ''}`}
                onClick={() => setFilterType('all')}
              >
                <div className="stat-number">{statistics.total}</div>
                <div className="stat-label">{t('stats.total_business_terms', 'Total Business Terms')}</div>
              </div>

              <div
                className={`stat-tile mapped ${filterType === 'with_relationships' ? 'active' : ''}`}
                onClick={() => setFilterType('with_relationships')}
              >
                <div className="stat-number">{statistics.withRelationships}</div>
                <div className="stat-label">{t('stats.with_relationships', 'With Relationships')}</div>
              </div>

              <div
                className={`stat-tile unmapped ${filterType === 'without_relationships' ? 'active' : ''}`}
                onClick={() => setFilterType('without_relationships')}
              >
                <div className="stat-number">{statistics.withoutRelationships}</div>
                <div className="stat-label">{t('stats.without_relationships', 'Without Relationships')}</div>
              </div>
            </div>
          )}
        </div>
      </div>

      <SemanticEnrichmentWizard
        open={wizardOpen}
        onClose={() => setWizardOpen(false)}
        tenantId={scopeTenantId || ''}
        datasourceId={effectiveDatasourceId || ''}
        onSuccess={() => {
          devDebug('SemanticEnrichmentWizard onSuccess triggered');
          refetch();
        }}
      />
    </div>
  );
};

// Provide default export for backwards compatibility with some tests
// eslint-disable-next-line import/no-default-export
export default BusinessTermsTab;