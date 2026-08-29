import { useState, useEffect } from 'react';
import {
  NavigateNext as ChevronRightIcon,
  NavigateBefore as ChevronLeftIcon,
  ViewColumn as ColumnsIcon,
  Map as MapIcon,
  TrackChanges as TargetIcon,
  Info as InfoIcon,
  ZoomIn as ZoomInIcon,
  ZoomOut as ZoomOutIcon,
  AutoAwesome as SparklesIcon,
  Search as SearchIcon
} from '@mui/icons-material';
import './ErdSidebar.css';

interface ErdSidebarProps {
  zoomLevel: number;
  showColumns: boolean;
  showMiniMap: boolean;
  infoMode: boolean;
  searchTerm?: string;
  onZoomChange: (zoom: number) => void;
  onToggleColumns: () => void;
  onToggleMiniMap: () => void;
  onFitView: () => void;
  onToggleInfoMode: () => void;
  onGenerateMappings?: () => void;
  onSearchChange?: (term: string) => void;
}

const ErdSidebar: React.FC<ErdSidebarProps> = ({
  zoomLevel,
  showColumns,
  showMiniMap,
  infoMode,
  searchTerm = '',
  onZoomChange,
  onToggleColumns,
  onToggleMiniMap,
  onFitView,
  onToggleInfoMode,
  onGenerateMappings,
  onSearchChange,
}) => {
  const [isCollapsed, setIsCollapsed] = useState(() => {
    const saved = localStorage.getItem('erdSidebarCollapsed');
    return saved === 'true';
  });

  useEffect(() => {
    localStorage.setItem('erdSidebarCollapsed', String(isCollapsed));
  }, [isCollapsed]);

  const handleZoomIn = () => {
    onZoomChange(Math.min(3, zoomLevel + 0.1));
  };

  const handleZoomOut = () => {
    onZoomChange(Math.max(0.1, zoomLevel - 0.1));
  };

  return (
    <div className={`erd-sidebar ${isCollapsed ? 'collapsed' : 'expanded'}`}>
      <div className="erd-sidebar-header">
        <button
          className="sidebar-toggle-btn"
          onClick={() => setIsCollapsed(!isCollapsed)}
          title={isCollapsed ? 'Expand Sidebar' : 'Collapse Sidebar'}
          aria-label={isCollapsed ? 'Expand sidebar' : 'Collapse sidebar'}
        >
          {isCollapsed ? <ChevronRightIcon sx={{ fontSize: 20 }} /> : <ChevronLeftIcon sx={{ fontSize: 20 }} />}
        </button>
      </div>

      <div className="erd-sidebar-content">
        {/* Search Input */}
        {!isCollapsed && onSearchChange && (
          <div className="sidebar-search">
            <input
              type="text"
              className="sidebar-search-input"
              placeholder="Search tables..."
              value={searchTerm}
              onChange={(e) => onSearchChange(e.target.value)}
            />
          </div>
        )}
        {isCollapsed && onSearchChange && (
          <div className="sidebar-section">
            <div className="sidebar-control-group">
              <button
                className="sidebar-control-btn"
                onClick={() => setIsCollapsed(false)}
                title="Search"
                aria-label="Open search"
              >
                <SearchIcon sx={{ fontSize: 18 }} />
              </button>
            </div>
          </div>
        )}

        {/* Zoom Controls */}
        <div className="sidebar-section">
          {!isCollapsed && <div className="sidebar-section-label">Zoom</div>}
          <div className="sidebar-control-group">
            <button
              className="sidebar-control-btn"
              onClick={handleZoomIn}
              title="Zoom In"
              aria-label="Zoom in"
            >
              <ZoomInIcon sx={{ fontSize: 18 }} />
              {!isCollapsed && <span>Zoom In</span>}
            </button>
            <button
              className="sidebar-control-btn"
              onClick={handleZoomOut}
              title="Zoom Out"
              aria-label="Zoom out"
            >
              <ZoomOutIcon sx={{ fontSize: 18 }} />
              {!isCollapsed && <span>Zoom Out</span>}
            </button>
            {!isCollapsed && (
              <div className="zoom-display">
                {Math.round(zoomLevel * 100)}%
              </div>
            )}
          </div>
        </div>

        {/* View Controls */}
        <div className="sidebar-section">
          {!isCollapsed && <div className="sidebar-section-label">View</div>}
          <div className="sidebar-control-group">
            <button
              className={`sidebar-control-btn ${showColumns ? 'active' : ''}`}
              onClick={onToggleColumns}
              title="Toggle Columns"
              aria-label="Toggle columns"
            >
              <ColumnsIcon sx={{ fontSize: 18 }} />
              {!isCollapsed && <span>Columns</span>}
            </button>
            <button
              className={`sidebar-control-btn ${showMiniMap ? 'active' : ''}`}
              onClick={onToggleMiniMap}
              title="Toggle Navigator"
              aria-label="Toggle navigator"
            >
              <MapIcon sx={{ fontSize: 18 }} />
              {!isCollapsed && <span>Navigator</span>}
            </button>
            <button
              className="sidebar-control-btn"
              onClick={onFitView}
              title="Fit View"
              aria-label="Fit view"
            >
              <TargetIcon sx={{ fontSize: 18 }} />
              {!isCollapsed && <span>Fit View</span>}
            </button>
          </div>
        </div>

        {/* Info Mode */}
        <div className="sidebar-section">
          {!isCollapsed && <div className="sidebar-section-label">Info</div>}
          <div className="sidebar-control-group">
            <button
              className={`sidebar-control-btn ${infoMode ? 'active' : ''}`}
              onClick={onToggleInfoMode}
              title="Toggle Info Mode"
              aria-label="Toggle info mode"
            >
              <InfoIcon sx={{ fontSize: 18 }} />
              {!isCollapsed && <span>Info Mode</span>}
            </button>
          </div>
        </div>
        {/* Semantic Actions */}
        {onGenerateMappings && (
          <div className="sidebar-section">
            {!isCollapsed && <div className="sidebar-section-label">Semantic</div>}
            <div className="sidebar-control-group">
              <button
                className="sidebar-control-btn"
                onClick={onGenerateMappings}
                title="Auto-Map Terms"
                aria-label="Auto-map semantic terms"
              >
                <SparklesIcon sx={{ fontSize: 18 }} />
                {!isCollapsed && <span>Auto-Map</span>}
              </button>
            </div>
          </div>
        )}
      </div>
    </div>
  );
};

export default ErdSidebar;
