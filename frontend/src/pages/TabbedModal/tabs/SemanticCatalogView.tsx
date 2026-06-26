import React, { useState, useMemo } from 'react';
import { Box, Typography, Paper, Button, Dialog, CircularProgress, Tooltip, Alert } from '@mui/material';
import { AutoAwesome, AccountTree as LineageIcon, Refresh as RefreshIcon, Fullscreen as FullscreenIcon, Add as AddIcon } from '@mui/icons-material';
import { useChartRefresh } from '../../../hooks/useChartRefresh';
import { EnhancedSelectedAsset } from '../../../types/SemanticTypes';
import SemanticTermsTree from '../../../components/SemanticTermsTree';
import SemanticTermDetails from './SemanticTermDetails';
import { SemanticMappingWizard } from '../../../components/SemanticMappingWizard';
import AddSemanticTermDialog from '../../../components/AddSemanticTermDialog';
import { CatalogNode, useDeleteSemanticTerm } from '../../../api/glossary';

interface SemanticCatalogViewProps {
  semanticAssets: any[];
  selectedAsset: EnhancedSelectedAsset | null;
  searchTerm: string;
  highlightedItem: string | null;
  onAssetSelect: (asset: EnhancedSelectedAsset) => void;
  semanticData: any;
  technicalData: any;
  datasourceId: string;
  tenantId?: string;
  onRefresh?: () => void; // Callback to refresh semantic terms after mappings
  onToggleFullScreen?: () => void;
  isFullScreen?: boolean;
}

const SemanticCatalogView: React.FC<SemanticCatalogViewProps> = ({
  semanticAssets = [],
  selectedAsset,
  searchTerm,
  highlightedItem,
  onAssetSelect,
  semanticData,
  technicalData,
  datasourceId,
  tenantId = 'default',
  onRefresh,
  onToggleFullScreen,
  isFullScreen = false,
}) => {
  const [wizardOpen, setWizardOpen] = useState(false);
  const [addTermOpen, setAddTermOpen] = useState(false);
  
  const deleteTermMutation = useDeleteSemanticTerm();

  // Chart refresh hook
  const { refreshCharts, isRefreshing } = useChartRefresh({
    datasourceId,
    onSuccess: () => {
      // Trigger data refresh after chart regeneration
      if (onRefresh) {
        onRefresh();
      }
    },
  });

  // Filter assets to only show those that have edges (mapped)
  const mappedAssets = useMemo(() => {
    if (!semanticData?.semantic_edges && !semanticData?.edges) return semanticAssets;
    
    // Combine edges sources if available
    const edges = [...(semanticData?.semantic_edges || []), ...(semanticData?.edges || [])];
    const mappedNodeIds = new Set<string>();
    
    edges.forEach((edge: any) => {
        if (edge.source_node_id) mappedNodeIds.add(edge.source_node_id);
        if (edge.target_node_id) mappedNodeIds.add(edge.target_node_id);
    });

    // Specifically filter Semantic Terms. Business Terms might strictly not have edges to columns directly but to terms.
    // For now, applying the rule: "any semantic term shown should have at least one edge record"
    return semanticAssets.filter(asset => {
        // preserve business terms or other types, usually we only want to filter "semantic_term" type strict
        if (asset.type === 'business_term') return true; 

        // check if asset id is in mapped set
        return mappedNodeIds.has(asset.id);
    });
  }, [semanticAssets, semanticData]);

  const handleEditTerm = (term: CatalogNode) => {
      // In this view, we might need a dedicated edit dialog or rely on the details panel
      // For now, just selecting it effectively "edits" via details panel
      onAssetSelect({
          type: 'semantic_term',
          id: `semantic_term-${term.id}`,
          nodeId: term.id,
          name: term.node_name,
          node: term
      });
  };

  const handleDeleteTerm = async (term: CatalogNode) => {
      if (confirm(`Are you sure you want to delete ${term.node_name}?`)) {
          console.log('[SemanticCatalogView] Deleting term:', term.id);
          try {
             await deleteTermMutation.mutateAsync(term.id);
             if (onRefresh) onRefresh();
          } catch (e) {
              console.error('Failed to delete term', e);
              alert('Failed to delete term');
          }
      }
  };



  return (
    <>
      <Box sx={{ display: 'flex', height: '100%', overflow: 'hidden', bgcolor: 'background.default' }}>
        {/* Sidebar - Term Tree */}
        <Box
          sx={{
            width: 380,
            minWidth: 380,
            borderRight: 1,
            borderColor: 'divider',
            bgcolor: 'background.paper',
            display: 'flex',
            flexDirection: 'column',
            height: '100%',
          }}
        >
          {/* Header with wizard and regenerate buttons */}
          <Box sx={{ p: 2, borderBottom: 1, borderColor: 'divider', display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
            <Typography variant="h6" fontWeight={600}>Semantic Terms</Typography>
            <Box sx={{ display: 'flex', gap: 1 }}>
              <Tooltip title="Add New Semantic Term">
                <Button
                  variant="outlined"
                  size="small"
                  sx={{ minWidth: 40, width: 40, px: 0 }}
                  onClick={() => setAddTermOpen(true)}
                >
                  <AddIcon />
                </Button>
              </Tooltip>
              <Tooltip title="Generate Mappings Wizard">
                <Button
                  variant="outlined"
                  size="small"
                  sx={{ minWidth: 40, width: 40, px: 0 }}
                  onClick={() => setWizardOpen(true)}
                >
                  <AutoAwesome />
                </Button>
              </Tooltip>
            </Box>
          </Box>
          
          {semanticAssets.length > mappedAssets.length && (
              <Alert severity="info" sx={{ mx: 1, mt: 1, py: 0 }}>
                  Hiding {semanticAssets.length - mappedAssets.length} unmapped terms
              </Alert>
          )}

          <SemanticTermsTree
            semanticAssets={mappedAssets}
            onAssetSelect={onAssetSelect}
            searchTerm={searchTerm}
            highlightedItem={highlightedItem}
          />
        </Box>

      {/* Main Content - Details & Lineage */}
      <Box sx={{ flex: 1, height: '100%', overflow: 'hidden', bgcolor: '#f8fafc' }}>
        {selectedAsset ? (
          <SemanticTermDetails
            asset={selectedAsset}
            semanticData={semanticData}
            technicalData={technicalData}
            allEdges={semanticData?.semantic_edges || semanticData?.edges || []}
            allNodes={semanticData?.all_nodes || [
                ...(semanticData?.business_terms || []),
                ...(semanticData?.semantic_terms || []),
                ...(semanticData?.semantic_columns || []),
                ...(technicalData?.nodes || [])
            ]}
            datasourceId={datasourceId}
            onRefresh={onRefresh}
            onAssetSelect={onAssetSelect}
          />
        ) : (
          <Box
            sx={{
              height: '100%',
              display: 'flex',
              flexDirection: 'column',
              alignItems: 'center',
              justifyContent: 'center',
              color: 'text.secondary',
              p: 3,
            }}
          >
            <Paper
              elevation={0}
              sx={{
                p: 6,
                display: 'flex',
                flexDirection: 'column',
                alignItems: 'center',
                bgcolor: 'transparent',
              }}
            >
              <Box
                sx={{
                  width: 80,
                  height: 80,
                  borderRadius: '50%',
                  bgcolor: 'action.hover',
                  display: 'flex',
                  alignItems: 'center',
                  justifyContent: 'center',
                  mb: 3,
                }}
              >
                <LineageIcon sx={{ fontSize: 40, color: 'text.disabled' }} />
              </Box>
              <Typography variant="h5" fontWeight={600} gutterBottom>
                Select a Business Term
              </Typography>
              <Typography variant="body1" textAlign="center" sx={{ maxWidth: 400, color: 'text.secondary' }}>
                Browse the glossary and select a term to view its properties, definition, and data lineage relationships.
              </Typography>
            </Paper>
          </Box>
        )}
      </Box>
    </Box>

    {/* Semantic Mapping Wizard Dialog */}
    <Dialog
      open={wizardOpen}
      onClose={() => setWizardOpen(false)}
      maxWidth="xl"
      fullWidth
      PaperProps={{
        sx: { height: '90vh' }
      }}
    >
      <SemanticMappingWizard
        tenantId={tenantId}
        datasourceId={datasourceId}
        onClose={() => setWizardOpen(false)}
        onMappingsApplied={() => {
          // Trigger chart regeneration to ensure lineage is updated with new mappings
          refreshCharts();
        }}
      />
    </Dialog>

    {/* Add Semantic Term Dialog */}
    <AddSemanticTermDialog
        open={addTermOpen}
        onClose={() => {
          setAddTermOpen(false);
          if (onRefresh) onRefresh();
        }}
        tenantId={tenantId}
        datasourceId={datasourceId}
    />
    </>
  );
};

export default SemanticCatalogView;
