  import React, { useState, useMemo } from 'react';
  import {
    Box,
    Tabs,
    Tab,
    Typography,
    Container,
    Alert,
    Snackbar,
    IconButton,
    Tooltip,
  } from '@mui/material';
  // AddIcon intentionally removed (unused)
  import { useTenant } from '../../contexts/TenantContext';
  import { SemanticTermsTab } from './SemanticTermsTab';
  import { BusinessTermsTab } from './BusinessTermsTab';
  import { CalculationTermsTab } from './CalculationTermsTab';
  import TermForm from '../../components/TermForm';
  import { useCreateTerm, useUpdateTerm, useDeleteTerm, useAllSemanticData, CatalogNode, glossaryKeys } from '../../api/glossary';
import { useQueryClient } from '@tanstack/react-query';
  import { devDebug } from '../../utils/devLogger';
  import BookIcon from '@mui/icons-material/Book';
  import CloseIcon from '@mui/icons-material/Close';
  import CategoryIcon from '@mui/icons-material/Category';
  import ProfessionalSearchInput from '../../components/ProfessionalSearchInput';
  import './BusinessGlossaryPage.css';
  import { useQuery } from '@apollo/client';
  import { GET_SEMANTIC_LINEAGE_CHART, transformChartData } from '../../graphql/queries/semantic';

  interface TabPanelProps {
    children?: React.ReactNode;
    index: number;
    value: number;
  }

  function TabPanel(props: TabPanelProps) {
    const { children, value, index, ...other } = props;

    return (
      <Box
        role="tabpanel"
        hidden={value !== index}
        id={`glossary-tabpanel-${index}`}
        aria-labelledby={`glossary-tab-${index}`}
        sx={{ height: '100%', display: value === index ? 'flex' : 'none', flexDirection: 'column' }}
        {...other}
      >
        {value === index && <Box sx={{ height: '100%', flex: 1 }}>{children}</Box>}
      </Box>
    );
  }

  export const BusinessGlossaryPage: React.FC = () => {
    const [currentTab, setCurrentTab] = useState(0);
    const [searchTerm, setSearchTerm] = useState('');
    const [externalSelectedBusinessTerm, setExternalSelectedBusinessTerm] = useState<CatalogNode | null>(null);
    const { tenant, datasource } = useTenant();

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

    const handleNavigateToBusinessTerm = (term: CatalogNode) => {
      // Switch to Business Terms tab and set the external selection so the tab highlights it
      setExternalSelectedBusinessTerm(term);
      setCurrentTab(0); // Switch to Business Terms tab
    };

    // CRUD hooks
    const createTermMutation = useCreateTerm();
    const updateTermMutation = useUpdateTerm();
    const deleteTermMutation = useDeleteTerm();

    // Get data for search suggestions and refetch function
    const { data } = useAllSemanticData();
    const queryClient = useQueryClient();

    // Fetch full lineage for mapped/unmapped calculation
    const { data: metricData } = useQuery(GET_SEMANTIC_LINEAGE_CHART, {
        variables: { datasourceId: datasource?.id || '' },
        skip: !datasource?.id,
        fetchPolicy: 'cache-and-network',
    });

    const [transformedLineage, setTransformedLineage] = useState<any>(null);

    React.useEffect(() => {
        if (metricData?.tenant_chart?.[0]?.chart) {
            try {
                const result = transformChartData(metricData.tenant_chart[0].chart);
                setTransformedLineage(result);
            } catch (e) {
                console.error('Failed to transform chart data in Glossary', e);
            }
        }
    }, [metricData]);

    // Create search data combining business terms and semantic terms
    const searchData = useMemo((): any[] => {
      const results: any[] = [];
      
      // Add business terms
      if (data?.business_terms) {
        data.business_terms.forEach((term: any) => {
          results.push({
            id: `business-${term.id}`,
            text: term.node_name,
            subtext: term.description || 'Business Term',
            payload: { type: 'business_term', data: term }
          });
        });
      }
      
      // Add semantic terms
      if (data?.semantic_terms) {
        data.semantic_terms.forEach((term: any) => {
          results.push({
            id: `semantic-${term.id}`,
            text: term.node_name,
            subtext: 'Semantic Term',
            payload: { type: 'semantic_term', data: term }
          });
        });
      }
      
      return results;
    }, [data]);

    const handleTabChange = (_event: React.SyntheticEvent, newValue: number) => {
      setCurrentTab(newValue);
    };

    const handleSearchSelect = (payload: any) => {
      if (payload) {
        // Switch to appropriate tab based on selection
        if (payload.type === 'business_term') {
          setCurrentTab(0); // Business Terms tab
        } else if (payload.type === 'semantic_term') {
          setCurrentTab(1); // Semantic Terms tab
        }
      }
    };

    // Check if tenant scope is selected
    const hasTenantScope = !!(tenant?.id && datasource?.id);

    // CRUD handlers
    const handleCreateTerm = (termType: 'business_term' | 'semantic_term') => {
      setEditingTerm(null);
      setFormTermType(termType);
      setFormOpen(true);
    };

    const handleEditTerm = (term: CatalogNode) => {
      setEditingTerm(term);
      setFormTermType(term.catalog_type as 'business_term' | 'semantic_term');
      setFormOpen(true);
    };

    const handleDeleteTerm = (term: CatalogNode) => {
      setTermToDelete(term);
      setDeleteConfirmOpen(true);
    };

    const handleSaveTerm = async (termData: Partial<CatalogNode>) => {
      try {
        devDebug('[BusinessGlossaryPage.handleSaveTerm] CALLED with termData:', termData);
        devDebug('[BusinessGlossaryPage.handleSaveTerm] parent_id in termData:', termData.parent_id);
        devDebug('[BusinessGlossaryPage.handleSaveTerm] editingTerm:', editingTerm?.id);
        
        if (editingTerm) {
          devDebug('[BusinessGlossaryPage.handleSaveTerm] About to call updateTermMutation.mutateAsync');
          const updatePayload = {
            id: editingTerm.id,
            updates: termData,
          };
          devDebug('[BusinessGlossaryPage.handleSaveTerm] Update payload:', JSON.stringify(updatePayload, null, 2));
          
          const updatedNode = await updateTermMutation.mutateAsync(updatePayload);
        devDebug('[BusinessGlossaryPage.handleSaveTerm] updateTermMutation completed successfully');
        setSnackbar({ open: true, message: 'Term updated successfully', severity: 'success' });
        // Update external selected term to reflect changes
        setExternalSelectedBusinessTerm(updatedNode);
        // Invalidate specific glossary queries using the correct query keys
        await queryClient.invalidateQueries({ queryKey: glossaryKeys.semanticTerms() });
        await queryClient.invalidateQueries({ queryKey: glossaryKeys.businessTerms() });
        await queryClient.invalidateQueries({ queryKey: glossaryKeys.edges() });
        devDebug('[BusinessGlossaryPage.handleSaveTerm] Query cache invalidated');
        } else {
          devDebug('[BusinessGlossaryPage.handleSaveTerm] About to call createTermMutation.mutateAsync');
          const createdNode = await createTermMutation.mutateAsync({
            ...termData,
            tenant_datasource_id: datasource?.id,
          } as Omit<CatalogNode, 'id' | 'created_at' | 'updated_at'>);
        devDebug('[BusinessGlossaryPage.handleSaveTerm] createTermMutation completed successfully');
        setSnackbar({ open: true, message: 'Term created successfully', severity: 'success' });
        // Set newly created term as selected
        setExternalSelectedBusinessTerm(createdNode);
        // Invalidate specific glossary queries using the correct query keys
        await queryClient.invalidateQueries({ queryKey: glossaryKeys.semanticTerms() });
        await queryClient.invalidateQueries({ queryKey: glossaryKeys.businessTerms() });
        await queryClient.invalidateQueries({ queryKey: glossaryKeys.edges() });
        devDebug('[BusinessGlossaryPage.handleSaveTerm] Query cache invalidated');
        }
        setFormOpen(false);
      } catch (error: any) {
        console.error('[BusinessGlossaryPage.handleSaveTerm] Error caught:', error);
        // If this is a server-side validation error, rethrow so the form can
        // display field-level errors inline. Otherwise display a generic snackbar.
        if (error && error.validation_errors) {
          throw error;
        }

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
        // Clear external selected term if it was deleted
        if (externalSelectedBusinessTerm?.id === termToDelete.id) {
          setExternalSelectedBusinessTerm(null);
        }
        // Invalidate specific glossary queries using the correct query keys
        await queryClient.invalidateQueries({ queryKey: glossaryKeys.semanticTerms() });
        await queryClient.invalidateQueries({ queryKey: glossaryKeys.businessTerms() });
        await queryClient.invalidateQueries({ queryKey: glossaryKeys.edges() });
        devDebug('[BusinessGlossaryPage.handleConfirmDelete] Query cache invalidated');
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

    if (!hasTenantScope) {
      return (
        <Box sx={{ p: 4 }}>
          <Alert severity="warning">
            Please select a tenant and datasource from the tenant picker to access the Business Glossary.
          </Alert>
        </Box>
      );
    }

    return (
      <Box
        sx={{
          display: 'flex',
          flexDirection: 'column',
          height: '100%',
          bgcolor: 'background.default',
        }}
      >
        {/* Header Section */}
        <Box
          sx={{
            bgcolor: 'background.paper',
            borderBottom: 1,
            borderColor: 'divider',
            boxShadow: 1,
          }}
        >
          {/* Reduced header height */}
          <Container maxWidth="xl" sx={{ py: 1 }}>
            <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', gap: 1, mb: 1 }}>
              <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                <BookIcon sx={{ fontSize: 20, color: 'primary.main' }} />
                <Box>
                  <Typography variant="h5" sx={{ fontWeight: '600' }}>
                    Business Glossary
                  </Typography>
                  <Typography variant="body2" color="textSecondary">
                    Manage semantic and business terms
                  </Typography>
                </Box>
              </Box>

              {/* Global Search - moved to right side */}
              <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                <Box sx={{ width: '400px' }}>
                <ProfessionalSearchInput
                  placeholder="Search business terms and semantic terms..."
                  data={searchData}
                  onSelect={handleSearchSelect}
                  onSearch={setSearchTerm}
                  className="glossary-global-search"
                />
                </Box>
                {/* Clear / reset button to quickly reset the typeahead */}
                {searchTerm && (
                  <Tooltip title="Clear search">
                    <IconButton size="small" onClick={() => { setSearchTerm(''); handleSearchSelect(undefined); }}>
                      <CloseIcon fontSize="small" />
                    </IconButton>
                  </Tooltip>
                )}
              </Box>
            </Box>

            {/* Tab Navigation (Business Terms first) */}
            <Tabs
              value={currentTab}
              onChange={handleTabChange}
              sx={{
                borderBottom: 1,
                borderColor: 'divider',
              }}
            >
              <Tab
                label="Business Terms"
                id="glossary-tab-0"
                aria-controls="glossary-tabpanel-0"
                icon={<BookIcon sx={{ mr: 1 }} />}
                iconPosition="start"
              />
              <Tab
                label="Semantic Terms"
                id="glossary-tab-1"
                aria-controls="glossary-tabpanel-1"
                icon={<CategoryIcon sx={{ mr: 1 }} />}
                iconPosition="start"
              />
              <Tab
                label="Calculated Values"
                id="glossary-tab-2"
                aria-controls="glossary-tabpanel-2"
                icon={<CategoryIcon sx={{ mr: 1 }} />}
                iconPosition="start"
              />
            </Tabs>
          </Container>
        </Box>

        {/* Content Area */}
        <Box sx={{ flex: 1, display: 'flex', flexDirection: 'column', minHeight: 0 }}>
          <TabPanel value={currentTab} index={0}>
            <div className="glossary-embed">
              <BusinessTermsTab
                searchTerm={searchTerm}
                onCreateTerm={() => handleCreateTerm('business_term')}
                onEditTerm={handleEditTerm}
                onDeleteTerm={handleDeleteTerm}
                selectedBusinessTerm={externalSelectedBusinessTerm}
              />
            </div>
          </TabPanel>
          <TabPanel value={currentTab} index={1}>
            <div className="glossary-embed">
              <SemanticTermsTab
                searchTerm={searchTerm}
                onCreateTerm={() => handleCreateTerm('semantic_term')}
                onEditTerm={handleEditTerm}
                onDeleteTerm={handleDeleteTerm}
                onNavigateToBusinessTerm={handleNavigateToBusinessTerm}
                semanticData={transformedLineage} // Pass the full lineage data
              />
            </div>
          </TabPanel>
          <TabPanel value={currentTab} index={2}>
            <div className="glossary-embed">
              <CalculationTermsTab
                searchTerm={searchTerm}
                onCreateTerm={() => handleCreateTerm('semantic_term')}
                onEditTerm={handleEditTerm}
                onDeleteTerm={handleDeleteTerm}
              />
            </div>
          </TabPanel>
        </Box>

        {/* Term Form Dialog */}
        <TermForm
          open={formOpen}
          onClose={() => setFormOpen(false)}
          onSave={handleSaveTerm}
          term={editingTerm}
          termType={formTermType}
          // When creating or editing terms from the glossary tabs we don't want
          // the user to change the type - it's fixed by which tab they opened the
          // form from. Pass disableTypeSelection so the `Type` select is not
          // editable. This covers both create-from-tab and edit-from-tab.
          disableTypeSelection={true}
        />

        {/* Delete Confirmation Dialog */}
        {deleteConfirmOpen && termToDelete && (
          <div className="delete-confirmation-overlay">
            <div className="delete-confirmation-dialog">
              <h3>Confirm Delete</h3>
              <p>Are you sure you want to delete the term "{termToDelete.node_name}"?</p>
              <p className="delete-warning">This action cannot be undone.</p>
              <div className="delete-confirmation-actions">
                <button
                  className="cancel-button"
                  onClick={() => setDeleteConfirmOpen(false)}
                >
                  Cancel
                </button>
                <button
                  className="delete-button"
                  onClick={handleConfirmDelete}
                >
                  Delete
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
          <Alert onClose={handleCloseSnackbar} severity={snackbar.severity} sx={{ width: '100%' }}>
            {snackbar.message}
          </Alert>
        </Snackbar>
      </Box>
    );
  };
