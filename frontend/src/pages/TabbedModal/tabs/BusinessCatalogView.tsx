// 3. BusinessCatalogView.tsx
import React, { useState, useMemo } from 'react';
import { Edge } from 'reactflow';
import { EnhancedSelectedAsset } from '../../../types/SemanticTypes';
import BusinessTermTree from './BusinessTermTree';
import BusinessTermSearch from './BusinessTermSearch';
import DetailsPane from '../Catalog/CatalogDetailsPane';

interface SemanticAsset {
  id: string;
  node_name: string;
  description: string;
  parent_id?: string;
  properties: Record<string, unknown>;
}

interface BusinessCatalogViewProps {
  businessTerms: SemanticAsset[];
  semanticTerms: SemanticAsset[];
  semanticViews: SemanticAsset[];
  businessEdges: Edge[];
  selectedAsset: EnhancedSelectedAsset | null;
  selectedEdge: Edge | null;
  highlightedItem: string | null;
  searchTerm: string;
  isRelationshipPanelOpen: boolean;
  forceLineageType: 'technical' | 'semantic';
  onCloseRelationshipPanel: () => void;
  onAssetSelect: (asset: EnhancedSelectedAsset) => void;
  onEdgeClick: (event: React.MouseEvent, edge: Edge) => void;
  processedTechnicalData: any;
  processedSemanticData: any;
  hierarchicalData?: any | null;
  preferHierarchical?: boolean;
}

const BusinessCatalogView: React.FC<BusinessCatalogViewProps> = ({
  businessTerms,
  semanticTerms,
  semanticViews,
  businessEdges,
  selectedAsset,
  selectedEdge,
  highlightedItem,
  searchTerm,
  isRelationshipPanelOpen,
  forceLineageType,
  onCloseRelationshipPanel,
  onAssetSelect,
  onEdgeClick,
  processedTechnicalData,
  processedSemanticData,
  hierarchicalData,
  preferHierarchical
}) => {
  const [searchResults, setSearchResults] = useState<SemanticAsset[]>([]);
  const [isSearching, setIsSearching] = useState(false);

  const displayTerms = useMemo(() => {
    return isSearching ? searchResults : businessTerms;
  }, [isSearching, searchResults, businessTerms]);

  const handleSearchResults = (results: SemanticAsset[]) => {
    setSearchResults(results);
    setIsSearching(true);
  };

  const handleClearSearch = () => {
    setSearchResults([]);
    setIsSearching(false);
  };

  const totalBusinessAssets = displayTerms.length;

  return (
    <div className="catalog-tab">
      <div className="catalog-layout">
        <div className="catalog-sidebar">
          <div className="catalog-section-header">
            <h3>Business Terms ({totalBusinessAssets})</h3>
          </div>
          <BusinessTermSearch
            onSearchResults={handleSearchResults}
            onClearSearch={handleClearSearch}
          />
          <BusinessTermTree
            businessTerms={displayTerms}
            onAssetSelect={onAssetSelect}
            searchTerm={searchTerm}
            highlightedItem={highlightedItem}
          />
        </div>
        <div className="catalog-main">
          <DetailsPane
            selectedAsset={selectedAsset}
            nodes={[]}
            edges={businessEdges}
            businessTerms={businessTerms}
            semanticTerms={semanticTerms}
            semanticViews={semanticViews}
            onEdgeClick={onEdgeClick}
            isRelationshipPanelOpen={isRelationshipPanelOpen}
            selectedEdge={selectedEdge}
            onCloseRelationshipPanel={onCloseRelationshipPanel}
            processedTechnicalData={processedTechnicalData}
            forceLineageType={forceLineageType}
            processedSemanticData={processedSemanticData}
            onAssetSelect={onAssetSelect}
            hierarchicalData={hierarchicalData}
            preferHierarchical={preferHierarchical}
          />
        </div>
      </div>
    </div>
  );
};

export default BusinessCatalogView;