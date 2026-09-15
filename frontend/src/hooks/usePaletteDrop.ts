import { useDroppable } from '@dnd-kit/core';

const usePaletteDrop = (activeWorkspaceTab: 'model'|'extension') => {
  const { isOver: dndIsOver, setNodeRef } = useDroppable({
    id: 'palette-drop',
    disabled: activeWorkspaceTab !== 'model',
  });

  const isOver = activeWorkspaceTab === 'model' && dndIsOver;

  return { isOver, drop: setNodeRef } as const;
};

export default usePaletteDrop;
