import React, { useMemo, useState } from 'react';
import { EnhancedSelectedAsset } from '../types/SemanticTypes';
import { IconButton, Tooltip, Checkbox } from '@mui/material';
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
  onEditTerm?: (term: CatalogNode) => void;
  onDeleteTerm?: (term: CatalogNode) => void;
  selectedIds?: string[];
  onSelectionChange?: (ids: string[]) => void;
  onDeleteMultiple?: (ids: string[]) => void;
}

const SemanticTermsTree: React.FC<SemanticTermsTreeProps> = ({
  semanticAssets = [],
  onAssetSelect,
  highlightedItem,
  searchTerm = '',
  onEditTerm,
  onDeleteTerm,
  selectedIds = [],
  onSelectionChange,
  onDeleteMultiple,
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

  const isSelectable = !!onSelectionChange;
  const filteredIds = useMemo(() => filteredAssets.map(a => a.id), [filteredAssets]);
  const allSelected = isSelectable && filteredIds.length > 0 && filteredIds.every(id => selectedIds.includes(id));
  const someSelected = isSelectable && filteredIds.some(id => selectedIds.includes(id)) && !allSelected;

  const handleToggleSelectAll = () => {
    if (!onSelectionChange) return;
    if (allSelected) {
      onSelectionChange(selectedIds.filter(id => !filteredIds.includes(id)));
    } else {
      const next = new Set(selectedIds);
      filteredIds.forEach(id => next.add(id));
      onSelectionChange(Array.from(next));
    }
  };

  const handleToggleItem = (assetId: string) => {
    if (!onSelectionChange) return;
    if (selectedIds.includes(assetId)) {
      onSelectionChange(selectedIds.filter(id => id !== assetId));
    } else {
      onSelectionChange([...selectedIds, assetId]);
    }
  };

  return (
    <div className="business-terms-tree-container">
      <div className="business-tree-header">
        <div className="tree-controls">
          {isSelectable && (
            <Tooltip title={t('select.all_visible', 'Select all visible')}>
              <Checkbox
                size="small"
                checked={allSelected}
                indeterminate={someSelected}
                onChange={handleToggleSelectAll}
                inputProps={{ 'aria-label': t('select.all_visible', 'Select all visible') }}
              />
            </Tooltip>
          )}
          <Tooltip title={showUnlinked ? t('filter.hide_unlinked', 'Hide Unlinked Terms') : t('filter.show_unlinked', 'Show Unlinked Terms')}>
            <IconButton
              onClick={() => setShowUnlinked(!showUnlinked)}
              size="small"
              color={showUnlinked ? 'default' : 'primary'}
            >
              {showUnlinked ? <VisibilityIcon /> : <VisibilityOffIcon />}
            </IconButton>
          </Tooltip>
          {onDeleteMultiple && selectedIds.length > 0 && (
            <Tooltip title={t('term.delete_selected', 'Delete selected terms')}>
              <IconButton
                size="small"
                color="error"
                onClick={() => onDeleteMultiple(selectedIds)}
              >
                <DeleteIcon fontSize="small" />
              </IconButton>
            </Tooltip>
          )}
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
              const isHighlighted = highlightedItem === assetId;
              const isLinked = asset.is_mapped;
              const isItemSelected = selectedIds.includes(asset.id);

              return (
                <div
                  key={assetId}
                  className={`business-term-item-flat ${isHighlighted ? 'selected' : ''}`}
                  onClick={() => handleAssetSelect(asset)}
                >
                  {isSelectable && (
                    <Checkbox
                      size="small"
                      checked={isItemSelected}
                      onClick={(e) => {
                        e.stopPropagation();
                        handleToggleItem(asset.id);
                      }}
                      inputProps={{ 'aria-label': t('select.term', 'Select term') }}
                    />
                  )}
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
                  <div className="term-actions">
                    {onEditTerm && (
                      <Tooltip title={t('term.edit', 'Edit Term')}>
                        <IconButton
                          size="small"
                          onClick={(e) => {
                            e.stopPropagation();
                            onEditTerm(asset as CatalogNode);
                          }}
                        >
                          <EditIcon fontSize="small" />
                        </IconButton>
                      </Tooltip>
                    )}
                    {onDeleteTerm && (
                      <Tooltip title={t('term.delete', 'Delete Term')}>
                        <IconButton
                          size="small"
                          onClick={(e) => {
                            e.stopPropagation();
                            onDeleteTerm(asset as CatalogNode);
                          }}
                        >
                          <DeleteIcon fontSize="small" />
                        </IconButton>
                      </Tooltip>
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