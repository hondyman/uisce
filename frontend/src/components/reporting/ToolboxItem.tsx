import type { FC, ReactNode } from 'react';
import { Paper, Typography } from '@mui/material';
import { alpha } from '@mui/material/styles';
import { useDraggable } from '@dnd-kit/core';

interface ToolboxItemProps {
  type: string;
  icon: ReactNode;
  label: string;
}

const ToolboxItem: FC<ToolboxItemProps> = ({ type, icon, label }) => {
  const { attributes, listeners, setNodeRef, transform, isDragging } = useDraggable({
    id: `toolbox-${type}`,
    data: { type },
  });

  const style = transform ? {
    transform: `translate3d(${transform.x}px, ${transform.y}px, 0)`,
  } : undefined;

  return (
    <div
      ref={setNodeRef}
      style={{ ...style, cursor: isDragging ? 'grabbing' : 'grab' }}
      {...listeners}
      {...attributes}
    >
      <Paper
        variant="outlined"
        sx={{
          p: 2,
          display: 'flex',
          alignItems: 'center',
          gap: 1.5,
          borderStyle: 'dashed',
          borderColor: isDragging ? alpha('#6366f1', 0.6) : alpha('#94a3b8', 0.6),
          bgcolor: isDragging ? alpha('#6366f1', 0.08) : 'linear-gradient(135deg, #667eea 0%, #764ba2 100%)',
          opacity: isDragging ? 0.6 : 1,
          transition: 'all 0.3s cubic-bezier(0.4, 0, 0.2, 1)',
          '&:hover': {
            borderColor: '#6366f1',
            boxShadow: '0 8px 32px rgba(99, 102, 241, 0.3)',
          },
          borderRadius: '12px',
          color: '#2196f3', // Using a distinct blue to test visibility
        }}
      >
        <div style={{ display: 'flex', alignItems: 'center' }}>
          {icon}
        </div>
        <Typography variant="body2" sx={{ fontWeight: 600 }}>
          {label}
        </Typography>
      </Paper>
    </div>
  );
};

export default ToolboxItem;
