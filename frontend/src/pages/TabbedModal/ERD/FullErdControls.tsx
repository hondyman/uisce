  // React default import not required with new JSX transform

  interface ErdControlsProps {
    zoomLevel: number;
    showColumns: boolean;
    showMiniMap: boolean;
    isExporting: boolean;
    onZoomChange: (zoom: number) => void;
    onToggleColumns: () => void;
    onToggleMiniMap: () => void;
    onFitView: () => void;
    onExportPng: () => void;
  }

  const ErdControls: React.FC<ErdControlsProps> = ({
      zoomLevel,
      showColumns,
      showMiniMap,
      isExporting,
      onZoomChange,
      onToggleColumns,
      onToggleMiniMap,
      onFitView,
      onExportPng
    }) => {
    return (
      <div className="erd-controls">
        <div className="zoom-control">
          <label htmlFor="erd-zoom-range" className="sr-only">Zoom level</label>
          <input
            id="erd-zoom-range"
            aria-label="Zoom level"
            type="range"
            min="0.1"
            max="3"
            step="0.1"
            value={zoomLevel}
            onChange={(e) => onZoomChange(parseFloat((e.target as HTMLInputElement).value))}
          />
          <span>{Math.round(zoomLevel * 100)}%</span>
        </div>
        <button 
          className={`control-btn ${showColumns ? 'active' : ''}`}
          onClick={onToggleColumns}
          title="Toggle Columns"
        >
          📄
        </button>
        <button 
          className={`control-btn ${showMiniMap ? 'active' : ''}`}
          onClick={onToggleMiniMap}
          title="Toggle MiniMap"
        >
          🗺️
        </button>
        <button 
          className="control-btn"
          onClick={onFitView}
          title="Fit View"
        >
          🎯
        </button>
        <button 
          className="control-btn"
          onClick={onExportPng}
          title="Export PNG"
          disabled={isExporting}
        >
          {isExporting ? '⏳' : '📷'}
        </button>
      </div>
    );
  };

  export default ErdControls;