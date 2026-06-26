// React default import not required with new JSX transform
import { useState } from 'react';

interface ErdControlsProps {
  zoomLevel: number;
  showColumns: boolean;
  showMiniMap: boolean;
  isExporting: boolean;
  isFullScreen?: boolean;
  onZoomChange: (zoom: number) => void;
  onToggleColumns: () => void;
  onToggleMiniMap: () => void;
  onFitView: () => void;
  onExportPng: () => void;
  onExportSvg?: () => void;
  onExportPdf?: () => void;
  onToggleFullScreen?: () => void;
}

const ErdControls: React.FC<ErdControlsProps> = ({
  zoomLevel,
  showColumns,
  showMiniMap,
  isExporting,
  isFullScreen = false,
  onZoomChange,
  onToggleColumns,
  onToggleMiniMap,
  onFitView,
  onExportPng,
  onExportSvg,
  onExportPdf,
  onToggleFullScreen
}) => {
  const [showExportMenu, setShowExportMenu] = useState(false);

  return (
    <div className="erd-controls">
      <div className="zoom-control">
        <label htmlFor="erd-zoom" className="sr-only">Zoom level</label>
        <input
          id="erd-zoom"
          type="range"
          min={0.1}
          max={3}
          step={0.1}
          value={zoomLevel}
          onChange={(e) => onZoomChange(parseFloat(e.target.value))}
        />
        <span aria-live="polite" className="ml-2">{Math.round(zoomLevel * 100)}%</span>
      </div>

      <button
        className={`control-btn ${showColumns ? 'active' : ''}`}
        onClick={onToggleColumns}
        title="Toggle Columns"
        aria-label="Toggle columns"
      >
        📄
      </button>

      <button
        className={`control-btn ${showMiniMap ? 'active' : ''}`}
        onClick={onToggleMiniMap}
        title="Toggle MiniMap"
        aria-label="Toggle minimap"
      >
        🗺️
      </button>

      <button
        className="control-btn"
        onClick={onFitView}
        title="Fit View"
        aria-label="Fit view"
      >
        🎯
      </button>

      {/* Fullscreen Toggle */}
      {onToggleFullScreen && (
        <button
          className={`control-btn ${isFullScreen ? 'active' : ''}`}
          onClick={onToggleFullScreen}
          title={isFullScreen ? "Exit Fullscreen" : "Fullscreen"}
          aria-label={isFullScreen ? "Exit fullscreen" : "Enter fullscreen"}
        >
          {isFullScreen ? '⛶' : '⛶'}
        </button>
      )}

      {/* Export Dropdown */}
      <div className="export-dropdown-container">
        <button
          aria-label="Export"
          aria-haspopup="true"
          aria-expanded={showExportMenu}
          className="control-btn"
          onClick={() => setShowExportMenu(!showExportMenu)}
          title="Export Diagram"
          disabled={isExporting}
        >
          {isExporting ? '⏳' : '📥'}
        </button>
        {showExportMenu && (
          <div className="export-dropdown-menu">
            <button
              onClick={() => { onExportPng(); setShowExportMenu(false); }}
              className="export-menu-item"
            >
              📷 Export PNG
            </button>
            {onExportSvg && (
              <button
                onClick={() => { onExportSvg(); setShowExportMenu(false); }}
                className="export-menu-item"
              >
                🎨 Export SVG
              </button>
            )}
            {onExportPdf && (
              <button
                onClick={() => { onExportPdf(); setShowExportMenu(false); }}
                className="export-menu-item"
              >
                📄 Export PDF
              </button>
            )}
          </div>
        )}
      </div>
    </div>
  );
};

export default ErdControls;