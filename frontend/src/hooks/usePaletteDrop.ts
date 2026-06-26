import { useDrop } from 'react-dnd';

const usePaletteDrop = (activeWorkspaceTab: 'model'|'extension') => {
  const [{ isOver }, drop] = useDrop(() => ({
    accept: 'palette-item',
    canDrop: () => activeWorkspaceTab === 'model',
    drop: (_item: any) => {},
    collect: (monitor: any) => ({ isOver: monitor.canDrop() && !!monitor.isOver() }),
  }), [activeWorkspaceTab]);

  return { isOver, drop } as const;
};

export default usePaletteDrop;
