import type { FC, ReactNode } from 'react';
import { Paper, Typography } from '@mui/material';
import { alpha } from '@mui/material/styles';
import { useDraggable } from '@dnd-kit/core';

interface ToolboxItemProps {
  type: string;
  icon: ReactNode;
  label: string;
  payload?: Record<string, unknown>;
}

const ToolboxItem: FC<ToolboxItemProps & { onAdd?: (type: string, payload?: Record<string, unknown>) => void }> = ({
  type,
  icon,
  label,
  payload,
  onAdd,
}) => {
  const { attributes, listeners, setNodeRef, transform, isDragging } = useDraggable({
    id: `toolbox-${type}`,
    data: { type, isToolboxItem: true, ...(payload ? { payload } : {}) },
  });

  const style = transform ? {
    transform: `translate3d(${transform.x}px, ${transform.y}px, 0)`,
  } : undefined;

  return (
    <div
      ref={setNodeRef}
      style={{ ...style, cursor: isDragging ? 'grabbing' : 'grab', marginBottom: 8 }}
      {...listeners}
      {...attributes}
      onClick={() => onAdd && onAdd(type, payload)}
    >
      <Paper
        variant="outlined"
        sx={{
          p: 1.5,
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'space-between',
          borderStyle: 'solid',
          borderColor: isDragging ? 'primary.main' : 'divider',
          bgcolor: isDragging ? alpha('#6366f1', 0.08) : 'background.paper',
          opacity: isDragging ? 0.6 : 1,
          transition: 'all 0.2s ease',
          '&:hover': {
            borderColor: 'primary.main',
            bgcolor: 'action.hover',
            boxShadow: '0 4px 16px rgba(99, 102, 241, 0.15)',
          },
          borderRadius: '8px',
          cursor: 'grab',
        }}
      >
        <div style={{ display: 'flex', alignItems: 'center', gap: 10 }}>
          {icon}
          <Typography variant="body2" sx={{ fontWeight: 600, fontSize: '0.8rem' }}>
            {label}
          </Typography>
        </div>
      </Paper>
    </div>
  );
};

export default ToolboxItem;
