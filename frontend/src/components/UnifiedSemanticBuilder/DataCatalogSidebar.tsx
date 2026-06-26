import { FC, useState } from 'react';
import DataCatalogTree from '../../pages/TabbedModal/tabs/DataCatalogTree';
import { Node as FlowNode } from 'reactflow';
import { EnhancedSelectedAsset } from '../../types/SemanticTypes';
import { SelectedColumn } from './types';
import { useTenant } from '../../contexts/TenantContext';
import { useAuthFetch } from '../../utils/authFetch';
import { useGlobalSearch } from '../../contexts/GlobalSearchContext';

interface DataCatalogSidebarProps {
  filteredNodes: FlowNode[];
  searchTerm: string;
  setSearchTerm: (term: string) => void;
  selectedColumn: SelectedColumn | null;
  setSelectedColumn: (column: SelectedColumn | null) => void;
}

export const DataCatalogSidebar: FC<DataCatalogSidebarProps> = ({ 
  filteredNodes, 
  searchTerm, 
  setSearchTerm, 
  selectedColumn, 
  setSelectedColumn, 
}) => { 
  const { searchTerm: ctxTerm, setSearchTerm: ctxSet } = useGlobalSearch();
  const term = typeof searchTerm === 'string' ? searchTerm : ctxTerm;
  const setTerm = setSearchTerm ?? ctxSet;
  const { datasource: _datasource } = useTenant();
  const { authFetch: _authFetch } = useAuthFetch();
  const [selection, setSelection] = useState<Set<string>>(new Set());

  const handleAssetSelect = (asset: EnhancedSelectedAsset) => {
    if (asset.type === 'column') {
      // Store the full asset ID along with other column data
      setSelectedColumn({
        nodeId: asset.nodeId,
        tableName: asset.tableName || '',
        column: asset.column,
        id: asset.id,
      });
      // Clear table selection when a column is selected
      setSelection(new Set());
    } else if (asset.type === 'table') {
      // If a table is selected, clear column selection
      setSelectedColumn(null);
      // Mark the table row as selected in the tree so the actions menu appears
      if (asset.id) {
        setSelection(new Set([asset.id]));
      }
    }
  };

  const handleRowSelectionChange = (sel: Set<string>) => {
    setSelection(new Set(sel));
  };

  // Note: Intentionally no per-row generation in sidebar. Generation is available in the Model Generator screen only.

  // Determine highlighted item for DataCatalogTree
  const highlightedItem = selectedColumn ? selectedColumn.id : null;

  return (
    <div className="data-catalog-sidebar">
      <div className="data-catalog-header">
        <div className="data-catalog-title">
          <span>Data Catalog</span>
        </div>
        <div className="data-catalog-search">
          <input
            type="text"
            placeholder="Search tables and columns..."
            value={term}
            onChange={(e) => setTerm(e.target.value)}
          />
        </div>
      </div>

      <div className="data-catalog-tree-container">
        <DataCatalogTree
          nodes={filteredNodes}
          onAssetSelect={handleAssetSelect}
          searchTerm={searchTerm}
          highlightedItem={highlightedItem}
          multiselect={false}
          selection={selection}
          onSelectionChange={handleRowSelectionChange}
          showGoldCopyIcon={true}
        />
      </div>
    </div>
  );
};

export default DataCatalogSidebar;