import React, { useMemo, useState } from 'react';
import { EnhancedSelectedAsset } from '../types/SemanticTypes';
import { IconButton, Tooltip } from '@mui/material';
import {
  EditOutlined as EditIcon,
  DeleteOutline as DeleteIcon,
  VisibilityOff as VisibilityOffIcon,
  Visibility as VisibilityIcon,
} from '@mui/icons-material';
import './BusinessTermsTree.css'; // Reusing the same CSS for consistent styling
import { useTranslation } from 'react-i18next';
import { CatalogNode } from '../api/glossary';

interface SemanticAsset {
  id: string;
  node_name: string;
  description: string;
  tenant_instance_id?: string;
  is_mapped?: boolean;
  [key: string]: any;
}

interface SemanticTermsTreeProps {
  semanticAssets: SemanticAsset[];
  onAssetSelect: (asset: EnhancedSelectedAsset) => void;
  searchTerm?: string;
  highlightedItem: string | null;
}

const SemanticTermsTree: React.FC<SemanticTermsTreeProps> = ({
  semanticAssets = [],
  onAssetSelect,
  highlightedItem,
  searchTerm = '',
}) => {
  const { t } = useTranslation();
  
  // Ensure semanticAssets is always an array
  const assets = Array.isArray(semanticAssets) ? semanticAssets : [];

  const [showUnlinked, setShowUnlinked] = useState(true);



  const filteredAssets = useMemo(() => {
    let filtered = assets;

    // Apply search filter
    if (searchTerm.trim()) {
      const searchLower = searchTerm.toLowerCase();
      filtered = filtered.filter(asset =>
        asset.node_name.toLowerCase().includes(searchLower) ||
        (asset.description && asset.description.toLowerCase().includes(searchLower))
      );
    }

    // Apply unlinked filter
    if (!showUnlinked) {
      filtered = filtered.filter(asset => asset.is_mapped);
    }

    // Sort alphabetically
    return filtered.sort((a, b) => a.node_name.localeCompare(b.node_name));
  }, [assets, searchTerm, showUnlinked]);

  const handleAssetSelect = (asset: SemanticAsset) => {
    const enhancedAsset: EnhancedSelectedAsset = {
      type: 'semantic_term',
      id: `semantic_term-${asset.id}`,
      nodeId: asset.id,
      name: asset.node_name,
      node: asset,
    };
    onAssetSelect(enhancedAsset);
  };

  const hasAssets = assets.length > 0;
  const hasResults = filteredAssets.length > 0;

  return (
    <div className="business-terms-tree-container">
      <div className="business-tree-header">
        <div className="tree-controls">
          <Tooltip title={showUnlinked ? t('filter.hide_unlinked', 'Hide Unlinked Terms') : t('filter.show_unlinked', 'Show Unlinked Terms')}>
            <IconButton 
              onClick={() => setShowUnlinked(!showUnlinked)} 
              size="small"
              color={showUnlinked ? 'default' : 'primary'}
            >
              {showUnlinked ? <VisibilityIcon /> : <VisibilityOffIcon />}
            </IconButton>
          </Tooltip>
        </div>
      </div>



      <div className="business-tree-content">
        {!hasAssets ? (
          <div className="no-results">
            <div className="no-results-icon">📁</div>
            <h4>{t('no_semantic_assets.title', 'No Semantic Assets')}</h4>
            <p>{t('no_semantic_assets.description', 'No semantic assets are available')}</p>
          </div>
        ) : !hasResults ? (
          <div className="no-results">
            <div className="no-results-icon">🔍</div>
            <h4>{t('no_results.title', 'No assets found')}</h4>
            <p>{t('no_results.description', 'No semantic assets match your search')}</p>
          </div>
        ) : (
          <div className="business-flat-view">
            {filteredAssets.map(asset => {
              const assetId = `semantic_term-${asset.id}`;
              const isSelected = highlightedItem === assetId;
              const isLinked = asset.is_mapped;
              
              return (
                <div
                  key={assetId}
                  className={`business-term-item-flat ${isSelected ? 'selected' : ''}`}
                  onClick={() => handleAssetSelect(asset)}
                >
                  <div className="term-content">
                    <div className="term-header">
                      <span 
                        className="term-name" 
                        style={{ 
                          color: isLinked ? '#2196F3' : '#1e293b',
                          fontWeight: isLinked ? 600 : 400
                        }}
                      >
                        {asset.node_name}
                      </span>
                    </div>
                    {asset.description && (
                      <div className="term-description">{asset.description}</div>
                    )}
                  </div>
                </div>
              );
            })}
          </div>
        )}
      </div>
    </div>
  );
};

export default SemanticTermsTree;