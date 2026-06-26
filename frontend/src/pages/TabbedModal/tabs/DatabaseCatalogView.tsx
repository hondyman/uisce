// 2. DatabaseCatalogView.tsx
import React, { useState } from 'react';
import { Node as FlowNode, Edge } from 'reactflow';
import { Button, Dialog } from '@mui/material';
import { AutoAwesome } from '@mui/icons-material';
import { EnhancedSelectedAsset, ColumnData } from '../../../types/SemanticTypes';
import DataCatalogTree from './DataCatalogTree';
import DetailsPane from '../Catalog/CatalogDetailsPane';
import { SemanticMappingWizard } from '../../../components/SemanticMappingWizard';

interface TableNodeData {
  schemaName?: string;
  tableName?: string;
  label?: string;
  isCore?: boolean;
  columns?: ColumnData[];
}

interface DatabaseCatalogViewProps {
  nodes: FlowNode<TableNodeData>[];
  edges: Edge[];
  selectedAsset: EnhancedSelectedAsset | null;
  selectedEdge: Edge | null;
  highlightedItem: string | null;
  searchTerm: string;
  showColumns: boolean;
  isRelationshipPanelOpen: boolean;
  forceLineageType: 'technical' | 'semantic';
  onCloseRelationshipPanel: () => void;
  onAssetSelect: (asset: EnhancedSelectedAsset) => void;
  onColumnCountClick?: (node: FlowNode<TableNodeData>) => void;
  onTotalColumnsClick?: (columns: any[], label?: string) => void;
  onEdgeClick: (event: React.MouseEvent, edge: Edge) => void;
  processedTechnicalData: any;
  processedSemanticData: any;
  hierarchicalData?: any | null;
  preferHierarchical?: boolean;
  onOpenColumnsModal?: (tableLabel: string, columns: any[]) => void;
  datasourceId: string;
  tenantId?: string;
  onRefresh?: () => void; // Callback to refresh database catalog after mappings
}

const DatabaseCatalogView: React.FC<DatabaseCatalogViewProps> = ({
  nodes,
  edges,
  selectedAsset,
  selectedEdge,
  highlightedItem,
  searchTerm,
  isRelationshipPanelOpen,
  forceLineageType,
  onCloseRelationshipPanel,
  onAssetSelect,
  onEdgeClick,
  onColumnCountClick,
  onTotalColumnsClick,
  processedTechnicalData,
  processedSemanticData,
  hierarchicalData,
  preferHierarchical,
  onOpenColumnsModal,
  datasourceId,
  tenantId = 'default',
  onRefresh,
}) => {
  const [wizardOpen, setWizardOpen] = useState(false);

  return (
    <>
      <div className="catalog-tab">
        <div className="catalog-layout">
          <div className="catalog-sidebar">
            <div className="catalog-section-header">
              <h3>Database Assets ({nodes.length})</h3>
              <Button
                variant="outlined"
                size="small"
                startIcon={<AutoAwesome />}
                onClick={() => setWizardOpen(true)}
                sx={{ ml: 'auto' }}
              >
                Map to Semantic Terms
              </Button>
            </div>
          <DataCatalogTree
            nodes={nodes}
            onAssetSelect={onAssetSelect}
            onColumnCountClick={onColumnCountClick}
            onTotalColumnsClick={onTotalColumnsClick}
            searchTerm={searchTerm}
            highlightedItem={highlightedItem}
            showGoldCopyIcon={true}
            hideAssignmentControls
          />
        </div>
        <div className="catalog-main">
          <DetailsPane
            selectedAsset={selectedAsset}
            nodes={nodes}
            edges={edges}
            businessTerms={[]}
            semanticTerms={[]}
            semanticViews={[]}
            onEdgeClick={onEdgeClick}
            isRelationshipPanelOpen={isRelationshipPanelOpen}
            selectedEdge={selectedEdge}
            onCloseRelationshipPanel={onCloseRelationshipPanel}
            processedTechnicalData={processedTechnicalData}
            forceLineageType={forceLineageType}
            processedSemanticData={processedSemanticData}
            onAssetSelect={onAssetSelect}
            onOpenColumnsModal={onOpenColumnsModal}
            hierarchicalData={hierarchicalData}
            preferHierarchical={preferHierarchical}
          />
        </div>
      </div>
    </div>

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
          // Refresh database catalog after mappings are applied
          if (onRefresh) {
            onRefresh();
          }
        }}
      />
    </Dialog>
  </>
  );
};

export default DatabaseCatalogView;