// LineageContainer.tsx - Fixed TypeScript version
import { lazy, Suspense, useEffect } from 'react';
import { devLog, devDebug } from '../../../utils/devLogger';
import { useLineageData, LineageContainerProps } from '../../../services/lineageService';

const DualLineageViewer = lazy(() => import('./DualLineageViewer'));

const LineageContainer: React.FC<LineageContainerProps> = ({ 
  datasourceId, 
  selectedAsset, 
  onAssetClick, 
  onRelationshipClick 
}) => {
  const { technicalData, semanticData, loading, error, refetch } = useLineageData(datasourceId);
     
  // Debug logging
  useEffect(() => {
    devLog('=== LINEAGE CONTAINER DEBUG ===');
    devDebug('Datasource ID:', datasourceId);
    devDebug('Selected Asset:', selectedAsset);
    devDebug('Technical Data:', technicalData);
    devDebug('Semantic Data:', semanticData);
    devDebug('Loading:', loading);
    devDebug('Error:', error);
  }, [datasourceId, selectedAsset, technicalData, semanticData, loading, error]);

  if (loading) {
    return (
      <div className="lineage-status-card">
        <div className="lineage-status-inner">
          <div className="lineage-status-icon">⏳</div>
          <p>Loading lineage data...</p>
        </div>
      </div>
    );
  }

  if (error) {
    return (
      <div className="lineage-status-card lineage-status-error">
        <div className="lineage-status-inner">
          <div className="lineage-status-icon">❌</div>
          <h3 className="lineage-status-title">Error Loading Lineage</h3>
          <p className="lineage-status-desc">{error}</p>
          <button onClick={refetch} className="lineage-retry-btn">Retry</button>
        </div>
      </div>
    );
  }

  return (
    <Suspense fallback={<div className="lineage-status-card"><div className="lineage-status-inner"><div className="lineage-status-icon">🔄</div><p>Loading lineage viewer…</p></div></div>}>
      <DualLineageViewer
        selectedAsset={selectedAsset}
        technicalData={technicalData}
        semanticData={semanticData}
        onAssetClick={onAssetClick}
        onRelationshipClick={onRelationshipClick}
      />
    </Suspense>
  );
};

export default LineageContainer;