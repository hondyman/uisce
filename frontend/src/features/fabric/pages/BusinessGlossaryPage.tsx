import React, { useState, useMemo, useEffect } from 'react';
import { useQuery } from '@apollo/client';
import { useNavigate } from 'react-router-dom';
import { GET_ALL_SEMANTIC_DATA } from '../../../graphql/queries/semantic';
import { useTenant } from '../../../contexts/TenantContext';
import BusinessTermsTree from '../../../components/BusinessTermsTree';
import SemanticFlow from '../../../components/SemanticFlow';
import TermForm from '../../../components/TermForm';
import { EnhancedSelectedAsset } from '../../../types/SemanticTypes';
import { useCreateTerm, useUpdateTerm, useDeleteTerm, CatalogNode } from '../../../api/glossary';
import { IconButton, Tooltip, Snackbar, Alert, Box, Typography } from '@mui/material';
import ProfessionalSearchInput from '../../../components/ProfessionalSearchInput';
import { devDebug, devError } from '../../../utils/devLogger';
import { useTranslation } from 'react-i18next';
import { Add as AddIcon, EditOutlined as EditIcon, DeleteOutline as DeleteIcon } from '@mui/icons-material';
import './BusinessGlossaryPage.css';

const BusinessGlossaryPage: React.FC = () => {
  const { datasource } = useTenant();
  const navigate = useNavigate();
  const [selectedAsset, setSelectedAsset] = useState<EnhancedSelectedAsset | null>(null);
  const [highlightedItem, setHighlightedItem] = useState<string | null>(null);
  const [activeTab, setActiveTab] = useState<'business_terms' | 'semantic_terms'>('business_terms');
  // Search state for typeahead / filtering
  const [searchTerm, setSearchTerm] = useState('');
  const [semanticTermsState, setSemanticTermsState] = useState<any[]>([]);
  // translate helper
  const { t } = useTranslation();

  const { data, loading, error } = useQuery(GET_ALL_SEMANTIC_DATA, {
    variables: {
      datasourceId: datasource?.id || ''
    },
    skip: !datasource?.id,
  });

  // Log query results and handle side effects
  useEffect(() => {
    if (data) {
      devDebug('[BusinessGlossaryPage] Query completed. Full data object:', JSON.stringify(data, null, 2));
      devDebug('[BusinessGlossaryPage] Business terms count:', data?.business_terms?.length);
      devDebug('[BusinessGlossaryPage] Semantic terms count:', data?.semantic_terms?.length);
      devDebug('[BusinessGlossaryPage] Semantic terms:', data?.semantic_terms);
      // Update semantic terms state directly
      if (data?.semantic_terms) {
        setSemanticTermsState(data.semantic_terms);
        devDebug('[BusinessGlossaryPage] Updated semanticTermsState:', data.semantic_terms);
      }
    }
    if (error) {
      devError('[BusinessGlossaryPage] Query error:', error);
    }
  }, [data, error]);

  // Build typeahead suggestions from business + semantic term names - moved after businessTerms is defined

  // Redirect to tenant selection if no datasource is selected
  useEffect(() => {
    if (!loading && !datasource?.id) {
      navigate('/');
    }
  }, [datasource?.id, loading, navigate]);

  const businessTerms = useMemo(() => {
    if (!data?.business_terms) return [];
    devDebug('[BusinessGlossaryPage] businessTerms:', data.business_terms);
    return data.business_terms;
  }, [data]);

  const semanticViews = useMemo(() => {
    if (!data?.semantic_columns) return []; // Using semantic_columns as semantic views for now
    return data.semantic_columns;
  }, [data]);

  const [selectedSemanticTerm, setSelectedSemanticTerm] = useState<any | null>(null);

  // CRUD state
  const [formOpen, setFormOpen] = useState(false);
  const [editingTerm, setEditingTerm] = useState<CatalogNode | null>(null);
  const [formTermType, setFormTermType] = useState<'business_term' | 'semantic_term'>('business_term');
  const [deleteConfirmOpen, setDeleteConfirmOpen] = useState(false);
  const [termToDelete, setTermToDelete] = useState<CatalogNode | null>(null);
  const [snackbar, setSnackbar] = useState<{ open: boolean; message: string; severity: 'success' | 'error' }>({
    open: false,
    message: '',
    severity: 'success',
  });

  // CRUD hooks
  const createTermMutation = useCreateTerm();
  const updateTermMutation = useUpdateTerm();
  const deleteTermMutation = useDeleteTerm();

  const semanticEdges = useMemo(() => {
    return data?.semantic_edges || [];
  }, [data]);

  // Build typeahead suggestions from business + semantic term names
  const typeaheadOptions = useMemo(() => {
    const sems = (data?.semantic_terms || semanticTermsState || []).map((t: any) => ({ id: t.id, label: t.node_name, type: 'semantic_term', node: t }));
    const businesses = (data?.business_terms || businessTerms || []).map((t: any) => ({ id: t.id, label: t.node_name, type: 'business_term', node: t }));
    // Include semantic views too
    const views = (data?.semantic_columns || semanticViews || []).map((t: any) => ({ id: t.id, label: t.node_name, type: 'semantic_view', node: t }));

    // Combine and dedupe by id
    const map: Record<string, any> = {};
    [...sems, ...businesses, ...views].forEach((o) => (map[o.id] = o));
    return Object.values(map);
  }, [data, semanticTermsState, businessTerms, semanticViews]);

  // Log when active tab changes
  useEffect(() => {
    devDebug('[BusinessGlossaryPage] Active tab changed to:', activeTab);
    devDebug('[BusinessGlossaryPage] semanticTermsState:', semanticTermsState);
    if (activeTab === 'semantic_terms') {
      devDebug('[BusinessGlossaryPage] RENDERING SEMANTIC TERMS TAB with', semanticTermsState.length, 'terms');
    }
  }, [activeTab, semanticTermsState]);

  const handleAssetSelect = (asset: EnhancedSelectedAsset) => {
    setSelectedAsset(asset);
    setHighlightedItem(asset.id);
    // clear semantic selection when choosing a new business term
    setSelectedSemanticTerm(null);
  };

  // CRUD handlers
  const handleCreateTerm = (termType: 'business_term' | 'semantic_term') => {
    devDebug('[CRUD] handleCreateTerm called with type:', termType);
    setEditingTerm(null);
    setFormTermType(termType);
    setFormOpen(true);
  };

  const handleEditTerm = (term: CatalogNode) => {
    devDebug('[CRUD] handleEditTerm called with term:', term);
    setEditingTerm(term);
    setFormTermType(term.catalog_type as 'business_term' | 'semantic_term');
    setFormOpen(true);
  };

  const handleDeleteTerm = (term: CatalogNode) => {
    devDebug('[CRUD] handleDeleteTerm called with term:', term);
    setTermToDelete(term);
    setDeleteConfirmOpen(true);
  };

  const handleSaveTerm = async (termData: Partial<CatalogNode>) => {
    devDebug('[CRUD] handleSaveTerm called with data:', termData);
    try {
      if (editingTerm) {
        await updateTermMutation.mutateAsync({
          id: editingTerm.id,
          updates: termData,
        });
        setSnackbar({ open: true, message: 'Term updated successfully', severity: 'success' });
      } else {
        await createTermMutation.mutateAsync({
          ...termData,
          tenant_tenant_instance_id: datasource?.id,
        } as Omit<CatalogNode, 'id' | 'created_at' | 'updated_at'>);
        setSnackbar({ open: true, message: 'Term created successfully', severity: 'success' });
      }
      setFormOpen(false);
    } catch (error) {
      setSnackbar({
        open: true,
        message: `Failed to ${editingTerm ? 'update' : 'create'} term: ${error instanceof Error ? error.message : 'Unknown error'}`,
        severity: 'error'
      });
    }
  };

  const handleConfirmDelete = async () => {
    if (!termToDelete) return;

    try {
      await deleteTermMutation.mutateAsync(termToDelete.id);
      setSnackbar({ open: true, message: 'Term deleted successfully', severity: 'success' });
      setDeleteConfirmOpen(false);
      setTermToDelete(null);

      // Clear selection if the deleted term was selected
      if (selectedAsset?.id === termToDelete.id) {
        setSelectedAsset(null);
      }
    } catch (error) {
      setSnackbar({
        open: true,
        message: `Failed to delete term: ${error instanceof Error ? error.message : 'Unknown error'}`,
        severity: 'error'
      });
    }
  };

  const handleCloseSnackbar = () => {
    setSnackbar({ ...snackbar, open: false });
  };

  if (loading) {
    return (
      <div className="business-glossary-loading">
        <div className="loading-spinner"></div>
        <p>{t('global_loading.business_glossary', 'Loading business glossary...')}</p>
      </div>
    );
  }

  if (error) {
    return (
      <div className="business-glossary-error">
        <h2>{t('global_error.business_glossary', 'Error Loading Business Glossary')}</h2>
        <p>{error.message}</p>
      </div>
    );
  }

  devDebug('[CRUD] Rendering BusinessGlossaryPage with CRUD state:', {
    formOpen,
    editingTerm,
    formTermType,
    deleteConfirmOpen,
    termToDelete,
  });

  

  return (
    <div className="business-glossary-page">
      <div className="business-glossary-content">
        <div className="business-glossary-sidebar">
          {/* Tab Navigation */}
          <div className="glossary-tabs">
            {/* Typeahead search above tabs */}
            <Box className="glossary-search-wrapper">
              <ProfessionalSearchInput
                placeholder={t('global_search.placeholder', 'Search business & semantic terms...')}
                data={typeaheadOptions.map((o: any) => ({ id: `${o.type}-${o.id}`, text: o.label, subtext: o.type === 'business_term' ? 'Business Term' : 'Semantic Term', payload: o }))}
                onSelect={(payload: any) => {
                  if (!payload) return;
                  const selected = payload;
                  setSearchTerm(selected.label || selected.node?.node_name || '');
                  if (selected.type === 'business_term') {
                    setActiveTab('business_terms');
                    handleAssetSelect({ id: `business_term-${selected.id}`, name: selected.label, type: 'business_term', nodeId: selected.id, node: selected.node });
                  } else {
                    setActiveTab('semantic_terms');
                    handleAssetSelect({ id: `semantic_term-${selected.id}`, name: selected.label, type: 'semantic_term', nodeId: selected.id, node: selected.node });
                  }
                }}
                onSearch={(q) => setSearchTerm(q)}
                className="glossary-global-search"
              />
            </Box>
              <div className="tab-with-actions">
              <button 
                className={`tab-button ${activeTab === 'business_terms' ? 'active' : ''}`}
                onClick={() => {
                  setActiveTab('business_terms');
                  setSelectedAsset(null);
                }}
              >
                {t('tab.business_terms', 'Business Terms')} ({businessTerms.length})
              </button>
              <Tooltip title={t('tab.add_business_term', 'Add Business Term')}>
                <IconButton
                  size="small"
                  onClick={() => {
                    devDebug('[CRUD] Add Business Term button clicked');
                    handleCreateTerm('business_term');
                  }}
                  className="tab-action-button"
                >
                  <AddIcon fontSize="small" />
                </IconButton>
              </Tooltip>
            </div>

            <div className="tab-with-actions">
              <button
                className={`tab-button ${activeTab === 'semantic_terms' ? 'active' : ''}`}
                onClick={() => {
                  setActiveTab('semantic_terms');
                  setSelectedAsset(null);
                }}
              >
                {t('tab.semantic_terms', 'Semantic Terms')} ({semanticTermsState.length})
              </button>
              <Tooltip title={t('tab.add_semantic_term', 'Add Semantic Term')}>
                <IconButton
                  size="small"
                  onClick={() => {
                    devDebug('[CRUD] Add Semantic Term button clicked');
                    handleCreateTerm('semantic_term');
                  }}
                  className="tab-action-button"
                >
                  <AddIcon fontSize="small" />
                </IconButton>
              </Tooltip>
            </div>
          </div>

          {activeTab === 'business_terms' && (
            <div className="business-terms-tree-wrapper">
              <BusinessTermsTree
                businessTerms={businessTerms}
                semanticTerms={semanticTermsState}
                semanticViews={semanticViews}
                selectedAsset={selectedAsset}
                onAssetSelect={handleAssetSelect}
                highlightedItem={highlightedItem}
                searchTerm={searchTerm}
                onEditTerm={handleEditTerm}
                onDeleteTerm={handleDeleteTerm}
              />
            </div>
          )}

          {activeTab === 'semantic_terms' && (
            <div className="semantic-terms-list">
              {semanticTermsState.filter((t: any) => (
                !searchTerm || String(t.node_name || '').toLowerCase().includes(searchTerm.toLowerCase()) || String(t.description || '').toLowerCase().includes(searchTerm.toLowerCase())
              )).length === 0 ? (
                <div className="empty-state">{t('terms.no_semantic_terms', 'No semantic terms found')} (state length: {semanticTermsState.length})</div>
              ) : (
                <ul className="terms-list">
                  {semanticTermsState
                    .filter((t: any) => (
                      !searchTerm || String(t.node_name || '').toLowerCase().includes(searchTerm.toLowerCase()) || String(t.description || '').toLowerCase().includes(searchTerm.toLowerCase())
                    ))
                    .map((term: any) => (
                    <li 
                      key={term.id}
                      className={`term-item ${selectedAsset?.nodeId === term.id ? 'selected' : ''}`}
                    >
                      <div 
                        className="term-content"
                        onClick={() => handleAssetSelect({
                          id: term.id,
                          name: term.node_name,
                          type: 'semantic_term',
                          nodeId: term.id,
                          node: term
                        })}
                      >
                        <div className="term-name">{term.node_name}</div>
                        {/* Data type removed from list for a cleaner list view */}
                      </div>
                      <div className="term-actions">
                        <Tooltip title={t('term.edit', 'Edit Term')}>
                          <IconButton
                            size="small"
                            onClick={(e) => {
                              e.stopPropagation();
                              handleEditTerm(term);
                            }}
                          >
                            <EditIcon fontSize="small" />
                          </IconButton>
                        </Tooltip>
                        <Tooltip title={t('term.delete', 'Delete Term')}>
                          <IconButton
                            size="small"
                            onClick={(e) => {
                              e.stopPropagation();
                              handleDeleteTerm(term);
                            }}
                          >
                            <DeleteIcon fontSize="small" />
                          </IconButton>
                        </Tooltip>
                      </div>
                    </li>
                  ))}
                </ul>
              )}
            </div>
          )}
        </div>

        <div className="business-glossary-main">
          {selectedAsset ? (
            <>
              <div className="business-term-details">
                <h2>{selectedAsset.name}</h2>
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
                    {selectedAsset.node?.properties && (
                      <div className="metadata-item">
                        <span className="metadata-label">Properties:</span>
                        <pre className="metadata-value">
                          {JSON.stringify(selectedAsset.node.properties, null, 2)}
                        </pre>
                      </div>
                    )}
                  </div>
                </div>
              </div>

              <div className="semantics-layout below-details-semantics">
                <div className="semantics-center">
                  <SemanticFlow
                    centerNode={selectedAsset.node}
                    allNodes={[selectedAsset.node, ...semanticTermsState]}
                    semanticEdges={semanticEdges}
                    onNodeClick={(t) => setSelectedSemanticTerm(t)}
                  />
                </div>
                <div className="semantics-right">
                  {selectedSemanticTerm ? (
                    <div className="semantic-detail">
                      <h3>{selectedSemanticTerm.node_name}</h3>
                      {selectedSemanticTerm.description && <p>{selectedSemanticTerm.description}</p>}
                      <h4>Properties</h4>
                      <pre>{JSON.stringify(selectedSemanticTerm.properties || {}, null, 2)}</pre>
                    </div>
                  ) : (
                    <div className="semantic-detail empty">Click a semantic term to see details</div>
                  )}
                </div>
              </div>
            </>
          ) : (
            <div className="business-glossary-welcome">
              <div className="welcome-content">
                <h2>Welcome to the Business Glossary</h2>
                <p>
                  Select a business term from the tree on the left to view its details,
                  definitions, and related information.
                </p>
                <div className="welcome-stats">
                  <div className="stat-item">
                    <span className="stat-number">{businessTerms.length}</span>
                    <span className="stat-label">Business Terms</span>
                  </div>
                </div>
              </div>
            </div>
          )}
        </div>
      </div>

      {/* CRUD Components */}
      <TermForm
        open={formOpen}
        onClose={() => setFormOpen(false)}
        onSave={handleSaveTerm}
        term={editingTerm}
        termType={formTermType}
        loading={createTermMutation.isPending || updateTermMutation.isPending}
        disableTypeSelection={true}
      />

      {/* Delete Confirmation Dialog */}
      {deleteConfirmOpen && termToDelete && (
        <div className="delete-confirmation-overlay">
          <div className="delete-confirmation-dialog">
            <h3>Confirm Delete</h3>
            <p>Are you sure you want to delete the term "{termToDelete.node_name}"?</p>
            <p className="delete-warning">This action cannot be undone.</p>
            <div className="delete-actions">
              <button
                onClick={() => setDeleteConfirmOpen(false)}
                disabled={deleteTermMutation.isPending}
              >
                Cancel
              </button>
              <button
                onClick={handleConfirmDelete}
                disabled={deleteTermMutation.isPending}
                className="delete-button"
              >
                {deleteTermMutation.isPending ? 'Deleting...' : 'Delete'}
              </button>
            </div>
          </div>
        </div>
      )}

      {/* Snackbar for notifications */}
      <Snackbar
        open={snackbar.open}
        autoHideDuration={6000}
        onClose={handleCloseSnackbar}
        anchorOrigin={{ vertical: 'bottom', horizontal: 'left' }}
      >
        <Alert onClose={handleCloseSnackbar} severity={snackbar.severity}>
          {snackbar.message}
        </Alert>
      </Snackbar>
    </div>
  );
};

export default BusinessGlossaryPage;