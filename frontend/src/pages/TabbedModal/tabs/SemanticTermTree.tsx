import React from 'react';
import BusinessTermTree from './BusinessTermTree';
import { EnhancedSelectedAsset } from '../../../types/SemanticTypes';

interface SemanticTermTreeProps {
  semanticAssets: any[];
  onAssetSelect: (asset: EnhancedSelectedAsset) => void;
  searchTerm: string;
  highlightedItem: string | null;
}

const SemanticTermTree: React.FC<SemanticTermTreeProps> = ({ semanticAssets, onAssetSelect, searchTerm, highlightedItem }) => {
  // Reuse the BusinessTermTree implementation but pass semanticAssets as businessTerms
  return (
    <BusinessTermTree
      businessTerms={semanticAssets}
      onAssetSelect={onAssetSelect}
      searchTerm={searchTerm}
      highlightedItem={highlightedItem}
    />
  );
};

export default SemanticTermTree;
