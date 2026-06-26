// No default React import required; keep JSX runtime implicit
import { useDrop } from 'react-dnd';
import './Canvas.css';

export default function Canvas({ onDrop, items }: { onDrop?: (item: any, pos?: any) => void; items?: any[] }) {
  const [{ isOver }, drop] = useDrop(() => ({
    accept: 'TILE',
    drop: (item: any, monitor: any) => {
      const offset = monitor.getClientOffset?.();
      if (onDrop) onDrop(item, offset);
      return { moved: true };
    },
    collect: (m: any) => ({ isOver: !!m.isOver?.() }),
  }), [onDrop]);

  return (
    <div ref={drop} className={`canvas ${isOver ? 'over' : ''}`}>
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
