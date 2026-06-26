// React default import not required with new JSX transform
import { useState, useEffect, useRef } from 'react';
import { MiniMap, useReactFlow } from 'reactflow';

const ErdMinimap: React.FC = () => {
  const { zoomIn, zoomOut } = useReactFlow();
  
  // Load saved position from localStorage or use default
  const [position, setPosition] = useState(() => {
    const saved = localStorage.getItem('erdMinimapPosition');
    return saved ? JSON.parse(saved) : { x: window.innerWidth - 300, y: 20 };
  });
  
  const [isDragging, setIsDragging] = useState(false);
  const [dragOffset, setDragOffset] = useState({ x: 0, y: 0 });
  const minimapRef = useRef<HTMLDivElement>(null);

  // Save position to localStorage whenever it changes
  useEffect(() => {
    localStorage.setItem('erdMinimapPosition', JSON.stringify(position));
  }, [position]);

  const handleMouseDown = (e: React.MouseEvent) => {
    // Only start drag if clicking on the minimap container (not the MiniMap itself)
    if (e.target === minimapRef.current || (e.target as HTMLElement).classList.contains('minimap-drag-handle')) {
      setIsDragging(true);
      setDragOffset({
        x: e.clientX - position.x,
        y: e.clientY - position.y,
      });
      e.preventDefault();
    }
  };

  useEffect(() => {
    const handleMouseMove = (e: MouseEvent) => {
      if (isDragging) {
        setPosition({
          x: e.clientX - dragOffset.x,
          y: e.clientY - dragOffset.y,
        });
      }
    };

    const handleMouseUp = () => {
      setIsDragging(false);
    };

    if (isDragging) {
      document.addEventListener('mousemove', handleMouseMove);
      document.addEventListener('mouseup', handleMouseUp);
    }

    return () => {
      document.removeEventListener('mousemove', handleMouseMove);
      document.removeEventListener('mouseup', handleMouseUp);
    };
  }, [isDragging, dragOffset]);

  const handleWheel = (event: React.WheelEvent) => {
    event.preventDefault();
    event.stopPropagation();
    
    if (event.deltaY < 0) {
      zoomIn();
    } else {
      zoomOut();
    }
  };

  return (
    <>
      <style>
        {`
          .draggable-minimap {
            position: fixed !important;
            width: 280px !important;
            height: 200px !important;
            background: rgba(255, 255, 255, 0.95) !important;
            border-radius: 8px !important;
            box-shadow: 0 4px 12px rgba(0, 0, 0, 0.15) !important;
            backdrop-filter: blur(10px) !important;
            overflow: hidden !important;
            z-index: 1000 !important;
            border: 0.5px solid rgba(0, 0, 0, 0.08) !important;
            transition: box-shadow 0.2s ease !important;
            cursor: ${isDragging ? 'grabbing' : 'grab'} !important;
          }
          
          .draggable-minimap:hover {
            box-shadow: 0 6px 16px rgba(0, 0, 0, 0.2) !important;
          }
          
          .draggable-minimap .react-flow__minimap {
            border: none !important;
            background: transparent !important;
            border-radius: 8px !important;
            cursor: default !important;
          }
          
          .draggable-minimap .react-flow__minimap-mask {
            fill: rgba(59, 130, 246, 0.15) !important;
            stroke: #3b82f6 !important;
            stroke-width: 2px !important;
            rx: 4 !important;
          }

          .draggable-minimap .react-flow__minimap-node {
            fill: #374151 !important;
            stroke: #1f2937 !important;
            stroke-width: 1.5px !important;
            opacity: 0.8 !important;
            cursor: pointer !important;
            transition: all 0.2s ease !important;
          }

          .draggable-minimap .react-flow__minimap-node:hover {
            fill: #3b82f6 !important;
            opacity: 1 !important;
          }
          
          .minimap-drag-handle {
            position: absolute;
            top: 0;
            left: 0;
            right: 0;
            height: 30px;
            cursor: grab;
            z-index: 1001;
            background: linear-gradient(to bottom, rgba(0,0,0,0.05), transparent);
          }
          
          .minimap-drag-handle:active {
            cursor: grabbing;
          }
        `}
      </style>

      <div 
        ref={minimapRef}
        className="draggable-minimap"
        style={{
          left: `${position.x}px`,
          top: `${position.y}px`,
        }}
        onMouseDown={handleMouseDown}
        onWheel={handleWheel}
      >
        <div className="minimap-drag-handle" />
        <MiniMap pannable zoomable />
      </div>
    </>
  );
};

export default ErdMinimap;