// React default import not required with new JSX transform
import { useDrag } from 'react-dnd';

interface Props {
  typeName: 'dimension' | 'measure' | 'filter' | 'join';
  label: string;
  description: string;
  icon: React.ReactNode;
  onAdd: (type: 'dimension' | 'measure' | 'filter' | 'join') => void;
  horizontal?: boolean;
  enableDrag?: boolean;
  onTooltipShow: (label: string, description: string, rect: DOMRect) => void;
  onTooltipHide: () => void;
}

const PaletteItem: React.FC<Props> = ({ typeName, label, description, icon, onAdd, horizontal = false, enableDrag = false, onTooltipShow, onTooltipHide }) => {
  const [{ isDragging }, drag] = useDrag(() => ({
    type: 'palette-item',
    canDrag: () => enableDrag,
    item: { type: typeName, isCore: false },
  collect: (m: any) => ({ isDragging: !!m.isDragging() })
  }), [typeName, enableDrag]);

  return (
    <button
      ref={drag}
      className={`palette-icon-btn colored ${typeName} ${isDragging ? 'dragging' : ''}`}
      type="button"
  onClick={(e) => { e.stopPropagation(); onAdd(typeName); }}
  aria-label={`${label} ${typeName}`}
  title={description || label}
  onMouseEnter={(e) => { onTooltipShow(label, description, (e.currentTarget as HTMLElement).getBoundingClientRect()); }}
  onMouseLeave={() => onTooltipHide()}
  onFocus={(e) => { onTooltipShow(label, description, (e.currentTarget as HTMLElement).getBoundingClientRect()); }}
  onBlur={() => onTooltipHide()}
    >
      {icon}
      {!horizontal && <span className="label">{label}</span>}
      {horizontal && <span className="sr-only">{label}</span>}
    </button>
  );
};

export default PaletteItem;
