import { IconButton, Tooltip, Chip, Tabs, Tab, Box, ToggleButtonGroup, ToggleButton, Button, Alert } from '@mui/material';
import React, { useState, useMemo, useEffect } from 'react';
import { useTenant } from '../../contexts/TenantContext';
import { useAccess } from '../../contexts/AccessContext';
import { EnhancedSelectedAsset } from '../../types/SemanticTypes'; 
import { CatalogNode, useAllSemanticData, useAllSemanticDataQuery } from '../../api/glossary';
import { useNodeTypes } from '../../api/nodeTypes';
import { usePropertyLookupMaps } from '../../hooks/usePropertyLookupMaps';
import { useTranslation } from 'react-i18next';
import { devDebug } from '../../utils/devLogger';
import { Add as AddIcon, ArrowBack as ArrowBackIcon, FilterList as FilterListIcon, Settings as SettingsIcon } from '@mui/icons-material';
import SemanticTermsTree from '../../components/SemanticTermsTree';
import SemanticTermDetails from '../../pages/TabbedModal/tabs/SemanticTermDetails';
import { ScopeSelectorDialog } from '../../components/ScopeSelectorDialog';
import './SemanticTermsTab.css';

export const SemanticTermsTab: React.FC<{ 
  searchTerm?: string;
  onCreateTerm?: () => void;
  onEditTerm?: (term: CatalogNode) => void;
  onDeleteTerm?: (term: CatalogNode) => void;
  // Optional callback for navigating to the business term detail view
  onNavigateToBusinessTerm?: (term: CatalogNode) => void;
  // Optional lineage data for advanced mapping checks
  semanticData?: any;
  technicalData?: any;
}> = ({ searchTerm, onCreateTerm, onEditTerm, onDeleteTerm, onNavigateToBusinessTerm, semanticData, technicalData }) => {
  const { tenant, datasource } = useTenant();
  const { isPlatformOperator } = useAccess();
  const [selectedAsset, setSelectedAsset] = useState<EnhancedSelectedAsset | null>(null);
  const [filterType, setFilterType] = useState<'all' | 'mapped' | 'unmapped'>('all');
  const [highlightedItem, setHighlightedItem] = useState<string | null>(null);
  const [selectedTermIds, setSelectedTermIds] = useState<string[]>([]);
  const [scopeSelectorOpen, setScopeSelectorOpen] = useState(false);

  const { data, error } = useAllSemanticData();
  const { t } = useTranslation();

  // When data changes, refresh the selected asset if it's currently being viewed
  useEffect(() => {
    if (selectedAsset?.nodeId && data?.semantic_terms) {
      const updatedTerm = data.semantic_terms.find((term: any) => term.id === selectedAsset.nodeId);
      if (updatedTerm) {
        devDebug('[SemanticTermsTab] Refreshing selectedAsset from updated data');
        setSelectedAsset({
          id: updatedTerm.id,
          name: updatedTerm.node_name || 'Untitled',
          type: 'semantic_term',
          nodeId: updatedTerm.id,
          node: updatedTerm,
        });
      }
    }
  }, [data?.semantic_terms]);

  // Calculate statistics using is_mapped property from backend
  const statistics = useMemo(() => {
    const terms = (semanticData?.semantic_terms || data?.semantic_terms || []);
    
    // Improved logic using backend 'is_mapped' flag
    let mappedCount = 0;
    const mappedIds = new Set<string>();

    terms.forEach((term: any) => {
      // Use explicit is_mapped flag if available, otherwise check edges (legacy fallback)
      let isMapped = term.is_mapped;
      
      if (isMapped === undefined) {
         // Fallback calculation
         const allEdges = [
             ...(semanticData?.semantic_edges || []), 
             ...(data?.semantic_edges || []),
             ...(technicalData?.edges || [])
         ];
         isMapped = allEdges.some((edge: any) => 
           edge.target_node_id === term.id || edge.source_node_id === term.id
         );
      }

      if (isMapped) {
        mappedCount++;
        mappedIds.add(term.id);
      }
    });

    const total = terms.length;
    const unmapped = total - mappedCount;

    return { total, mapped: mappedCount, unmapped, mappedIds };
  }, [data, semanticData, technicalData]);

  const semanticTerms = useMemo(() => {
    const srcTerms = (semanticData?.semantic_terms || data?.semantic_terms || []);
    if (!srcTerms) return [];
    
    let terms = srcTerms;

    if (filterType !== 'all') {
      const mappedIds = statistics.mappedIds;
      
      switch (filterType) {
        case 'mapped':
          terms = terms.filter((term: any) => 
            term.is_mapped !== undefined ? term.is_mapped : mappedIds.has(term.id)
          );
          break;
        case 'unmapped':
          terms = terms.filter((term: any) => 
            term.is_mapped !== undefined ? !term.is_mapped : !mappedIds.has(term.id)
          );
          break;
      }
    }

    return terms;
  }, [data, semanticData, filterType, statistics]);


  const handleAssetSelect = (asset: EnhancedSelectedAsset) => {
    setSelectedAsset(asset);
    setHighlightedItem(asset.id);
  };

  const handleBackToSplash = () => {
    setSelectedAsset(null);
    setHighlightedItem(null);
  };

  const handleClearFilter = () => {
    setFilterType('all');
  };

  // Debug logging to track component state - MUST be before any conditional returns
  useEffect(() => {
    console.info('[SemanticTermsTab] render snapshot', {
      selectedAsset: selectedAsset?.name,
      selectedAssetId: selectedAsset?.nodeId,
      semanticTermsCount: semanticTerms?.length,
      edgesCount: data?.semantic_edges?.length,
      filterType,
      statistics
    });
  });

  const anyLoading = !data;

  // Show empty state if no scope is selected for platform operators
  if (!tenant || !datasource) {
    return (
      <div className="semantic-terms-empty-state" style={{
        display: 'flex',
        flexDirection: 'column',
        alignItems: 'center',
        justifyContent: 'center',
        height: '100%',
        padding: '40px',
        textAlign: 'center'
      }}>
        <SettingsIcon sx={{ fontSize: 64, color: 'action.disabled', mb: 2 }} />
        <h2 style={{ color: '#666', marginBottom: '12px' }}>
          {isPlatformOperator ? 'Select Operating Scope' : 'No Scope Available'}
        </h2>
        <p style={{ color: '#999', marginBottom: '24px', maxWidth: '400px' }}>
          {isPlatformOperator 
            ? 'Please select a tenant and datasource to view and manage semantic terms.'
            : 'Your user account does not have access to any tenants. Contact your administrator.'}
        </p>
        {isPlatformOperator && (
          <Button 
            variant="contained" 
            color="primary"
            onClick={() => setScopeSelectorOpen(true)}
            startIcon={<SettingsIcon />}
          >
            Select Tenant & Datasource
          </Button>
        )}
        <ScopeSelectorDialog 
          open={scopeSelectorOpen} 
          onClose={() => setScopeSelectorOpen(false)} 
        />
      </div>
    );
  }

  if (anyLoading) {
    return (
      <div className="semantic-terms-loading">
        <div className="loading-spinner"></div>
        <p>Loading semantic terms...</p>
      </div>
    );
  }

  const hasFallbackData = (Array.isArray(data?.semantic_terms) && data.semantic_terms.length > 0);

  if (error && !hasFallbackData) {
    return (
      <div className="semantic-terms-error">
        <h2>Error Loading Semantic Terms</h2>
        <p>{error instanceof Error ? error.message : String(error)}</p>
      </div>
    );
  }



  const handleBulkDelete = async (ids: string[]) => {
    if (confirm(t('confirm.delete_multiple', 'Are you sure you want to delete {{count}} terms?', { count: ids.length }))) {
        if (onDeleteTerm) {
            // Since the parent only provides single delete, we iterate. 
            // Ideally should be a bulk API, but this works for now.
            // We need to find the node objects
            const termsToDelete = (semanticTerms || []).filter((t: CatalogNode) => ids.includes(t.id));
            
            // Execute sequentially to avoid overwhelming
            for (const term of termsToDelete) {
                await onDeleteTerm(term);
            }
            setSelectedTermIds([]);
        }
    }
  };



  return (
    <div className="business-terms-tab-container">
      {/* Header with Add Button */}
      <div className="business-terms-header">
        <h3>{t('tab.semantic_terms', 'Semantic Terms')}</h3>
        <div className="header-actions">
          <ToggleButtonGroup
            value={filterType}
            exclusive
            onChange={(_, newVal) => { if (newVal) setFilterType(newVal); }}
            size="small"
            aria-label="mapping filter"
            style={{ marginRight: '8px', height: '32px' }}
          >
            <ToggleButton value="all" aria-label="all">
              {t('filter.all', 'All')}
            </ToggleButton>
            <ToggleButton value="mapped" aria-label="mapped">
               {t('filter.mapped', 'Mapped')}
            </ToggleButton>
            <ToggleButton value="unmapped" aria-label="unmapped">
               {t('filter.unmapped', 'Unmapped')}
            </ToggleButton>
          </ToggleButtonGroup>
          {onCreateTerm && (
            <Tooltip title={t('tab.add_semantic_term', 'Add New Semantic Term')}>
              <IconButton
                size="small"
                onClick={onCreateTerm}
                className="add-term-button"
              >
                <AddIcon fontSize="small" />
              </IconButton>
            </Tooltip>
          )}
        </div>
      </div>

      <div className="business-terms-content">
        <div className="business-terms-sidebar">
          <SemanticTermsTree
            semanticAssets={(semanticTerms || []).map((term: CatalogNode) => ({
              ...term,
              node_name: term.node_name || 'Untitled'
            })) as any}
            onAssetSelect={handleAssetSelect}
            highlightedItem={highlightedItem}
            searchTerm={searchTerm}
            onEditTerm={onEditTerm}
            onDeleteTerm={onDeleteTerm}
            selectedIds={selectedTermIds}
            onSelectionChange={setSelectedTermIds}
            onDeleteMultiple={handleBulkDelete}
          />
        </div>

        <div className="business-terms-main">
          {selectedAsset ? (
            <SemanticTermDetails
              asset={selectedAsset}
              semanticData={{
                business_terms: data?.business_terms || [],
                semantic_terms: data?.semantic_terms || [],
                semantic_edges: data?.semantic_edges || [],
                semantic_columns: (data as any)?.semantic_columns || [],
                node_types: data?.node_types || []
              }}
              technicalData={technicalData}
              allEdges={data?.semantic_edges || []}
              allNodes={data?.all_nodes || []}
              datasourceId={datasource?.id}
              onRefresh={() => {
                // Trigger a refetch of semantic data
                // The useAllSemanticData hook should handle this automatically
                devDebug('[SemanticTermsTab] Refresh triggered from SemanticTermDetails');
              }}
              onAssetSelect={(newAsset) => {
                // Handle navigation to related assets
                if (newAsset.type === 'business_term' && onNavigateToBusinessTerm) {
                  // Navigate to business term in parent page
                  const businessTerm = data?.business_terms?.find((bt: any) => bt.id === newAsset.nodeId);
                  if (businessTerm) {
                    onNavigateToBusinessTerm(businessTerm);
                  }
                } else {
                  // Select the asset locally
                  setSelectedAsset(newAsset);
                  setHighlightedItem(newAsset.id);
                }
              }}
            />
          ) : (
            <div className="statistics-tiles">
              <div
                className={`stat-tile ${filterType === 'all' ? 'active' : ''}`}
                onClick={() => setFilterType('all')}
              >
                <div className="stat-number">{statistics.total}</div>
                <div className="stat-label">{t('stats.total_semantic_terms', 'Total Semantic Terms')}</div>
              </div>

              <div
                className={`stat-tile mapped ${filterType === 'mapped' ? 'active' : ''}`}
                onClick={() => setFilterType('mapped')}
              >
                <div className="stat-number">{statistics.mapped}</div>
                <div className="stat-label">{t('stats.mapped_to_business', 'Mapped to Business Terms')}</div>
              </div>

              <div
                className={`stat-tile unmapped ${filterType === 'unmapped' ? 'active' : ''}`}
                onClick={() => setFilterType('unmapped')}
              >
                <div className="stat-number">{statistics.unmapped}</div>
                <div className="stat-label">{t('stats.unmapped_from_business', 'Unmapped from Business Terms')}</div>
              </div>
            </div>
          )}
        </div>
      </div>
    </div>
  );
};

export default SemanticTermsTab;