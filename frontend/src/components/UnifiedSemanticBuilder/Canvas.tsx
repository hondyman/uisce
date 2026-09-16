// No default React import required; keep JSX runtime implicit
import { useDroppable, useDndMonitor, type DragEndEvent } from '@dnd-kit/core';
import './Canvas.css';

const CANVAS_DROPPABLE_ID = 'unified-semantic-canvas';

export default function Canvas({ onDrop, items }: { onDrop?: (item: any, pos?: any) => void; items?: any[] }) {
  const { isOver, setNodeRef } = useDroppable({ id: CANVAS_DROPPABLE_ID });

  useDndMonitor({
    onDragEnd(event: DragEndEvent) {
      const { active, over } = event;
      if (over?.id === CANVAS_DROPPABLE_ID && onDrop) {
        onDrop(active.data.current, active.rect.current.translated);
      }
    },
  });

  return (
    <div ref={setNodeRef} className={`canvas ${isOver ? 'over' : ''}`}>
      <div className="canvas-help">Drop tiles here to compose your cube</div>
      <div className="canvas-grid">
        {items && items.map((it, idx) => (
          <div key={it.id || idx} className={`canvas-tile ${it.origin === 'core' ? 'core-linked' : 'custom'}`}>
            <div className="canvas-tile-title">{it.kind || it.title}</div>
            <div className="canvas-tile-actions">
              <button className="btn btn-xs">Edit</button>
            </div>
          </div>
        ))}
      </div>
    </div>
  );
}
