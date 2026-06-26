import type { FC, KeyboardEvent } from 'react';
import { motion } from 'framer-motion';
import { Rnd } from 'react-rnd';
import { Box, Typography, IconButton } from '@mui/material';
import { Trash2 } from 'lucide-react';

interface ReportElementProps {
  id: string;
  type: string;
  position: { x: number; y: number };
  size: { width: number; height: number };
  properties: Record<string, any>;
  onUpdate: (id: string, updates: Partial<any>) => void;
  onDelete: (id: string) => void;
  onSelect: (id: string) => void;
  isSelected: boolean;
}

const ReportElement: FC<ReportElementProps> = ({
  id,
  type,
  position,
  size,
  properties,
  onUpdate,
  onDelete,
  onSelect,
  isSelected,
}) => {
  const renderContent = () => (
    <motion.div initial={{ opacity: 0, scale: 0.8 }} animate={{ opacity: 1, scale: 1 }} transition={{ duration: 0.2 }}>
      <Typography
        variant="body2"
        sx={{
          fontSize: Number(properties.fontSize) || 12,
          textAlign: properties.textAlign || 'left',
          fontWeight: properties.fontWeight || 500,
          color: properties.textColor || '#111827',
          border: `${properties.borderWidth || 0}px solid ${properties.borderColor || 'transparent'}`,
          padding: properties.borderWidth ? '4px' : '0',
        }}
      >
        {String(properties.text ?? 'Sample Text')}
      </Typography>
    </motion.div>
  );

  return (
    <motion.div initial={{ opacity: 0 }} animate={{ opacity: 1 }} transition={{ duration: 0.3 }}>
      <Rnd
        size={size}
        position={position}
        onDragStop={(_, d) => onUpdate(id, { position: { x: d.x, y: d.y } })}
        onResizeStop={(_, _direction, ref, _delta, nextPosition) => {
          onUpdate(id, {
            size: { width: ref.offsetWidth, height: ref.offsetHeight },
            position: nextPosition,
          });
        }}
        bounds="parent"
        onClick={() => onSelect(id)}
        enableUserSelectHack={false}
        tabIndex={0}
        onKeyDown={(e: KeyboardEvent) => {
          if (e.key === 'Delete') onDelete(id);
          if (e.key === 'Enter') onSelect(id);
        }}
        aria-label={`Report element: ${type}`}
      >
        <Box sx={{ width: '100%', height: '100%', border: isSelected ? '2px solid #6366f1' : '1px solid transparent', position: 'relative', transition: 'border 0.15s ease', '&:hover': { border: '1px solid #6366f1', boxShadow: '0 4px 20px rgba(99, 102, 241, 0.3)' }, borderRadius: '8px', overflow: 'hidden' }} onClick={(e) => { e.stopPropagation(); onSelect(id); }}>
          {renderContent()}
          {isSelected && (
            <motion.div initial={{ scale: 0 }} animate={{ scale: 1 }} transition={{ type: 'spring', stiffness: 500, damping: 30 }}>
              <IconButton size="small" sx={{ position: 'absolute', top: -12, right: -12, bgcolor: '#ef4444', color: '#ffffff', '&:hover': { bgcolor: '#dc2626' }, width: 24, height: 24, zIndex: 1000 }} onClick={(e) => { e.stopPropagation(); onDelete(id); }} aria-label="Delete element">
                <Trash2 size={12} />
              </IconButton>
            </motion.div>
          )}
        </Box>
      </Rnd>
    </motion.div>
  );
};

export default ReportElement;
