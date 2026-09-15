import React, { useState } from 'react';
import { Box, Tabs, Tab, Stack, Typography, IconButton, Tooltip } from '@mui/material';
import { DndContext, useDraggable } from '@dnd-kit/core';
import AddIcon from '@mui/icons-material/Add';
import BOFieldsPalette, { type BOField } from '../reporting/BOFieldsPalette';
import { RelatedObjectsPalette } from './RelatedObjectsPalette';
import { CoreFieldBadge } from './CoreFieldBadge';
import { WIDGET_REGISTRY } from './widgetRegistry';
import type { ContainerWidgetType, RelationshipResult } from './pageStudioTypes';

export interface PageStudioPaletteProps {
  selectedBO: any;
  relatedBOs?: any[];
  selectedSubtypeKeys: string[];
  relationships: RelationshipResult[] | null;
  onAddFieldToCanvas: (field: BOField) => void;
  onAddAllAsTable: (fields: BOField[]) => void;
  onAddWidget: (widgetType: ContainerWidgetType) => void;
  onAddRelatedObject: (relationship: RelationshipResult) => void;
}

export const PageStudioPalette: React.FC<PageStudioPaletteProps> = ({
  selectedBO,
  relatedBOs,
  relationships,
  onAddFieldToCanvas,
  onAddAllAsTable,
  onAddWidget,
  onAddRelatedObject,
}) => {
  const [tab, setTab] = useState<'fields' | 'widgets' | 'related'>('fields');

  return (
    <DndContext>
    <Box sx={{ display: 'flex', flexDirection: 'column', height: '100%', width: 340, borderRight: '1px solid rgba(255,255,255,0.08)' }}>
      <Tabs value={tab} onChange={(_, v) => setTab(v)} variant="fullWidth">
        <Tab value="fields" label="BO Fields" />
        <Tab value="widgets" label="Widgets" />
        <Tab value="related" label="Related" />
      </Tabs>
      <Box sx={{ flex: 1, overflowY: 'auto' }}>
        {tab === 'fields' ? (
          <BOFieldsPalette
            selectedBO={selectedBO}
            relatedBOs={relatedBOs}
            selectedSubtypeKey={null}
            onAddFieldToCanvas={onAddFieldToCanvas}
            onAddAllAsTable={onAddAllAsTable}
            mode="design"
          />
        ) : tab === 'related' ? (
          <RelatedObjectsPalette relationships={relationships} onAddRelatedObject={onAddRelatedObject} />
        ) : (
          <Stack spacing={1} sx={{ p: 1.5 }}>
            {Object.values(WIDGET_REGISTRY)
              .filter((w) => w.type !== 'field' && w.type !== 'relatedObject')
              .map((w) => (
                <WidgetTile key={w.type} widgetType={w.type} label={w.label} description={w.description} Icon={w.Icon} isCoreDefault={w.isCoreDefault} onAdd={() => onAddWidget(w.type as ContainerWidgetType)} />
              ))}
          </Stack>
        )}
      </Box>
    </Box>
    </DndContext>
  );
};

/** One draggable widget-type tile - see PageStudioPalette's `widgets` tab. */
const WidgetTile: React.FC<{
  widgetType: string;
  label: string;
  description: string;
  Icon: React.ElementType;
  isCoreDefault?: boolean;
  onAdd: () => void;
}> = ({ widgetType, label, description, Icon, isCoreDefault, onAdd }) => {
  const { attributes, listeners, setNodeRef, isDragging } = useDraggable({
    id: `widget-${widgetType}`,
    data: { kind: 'widget', widgetType },
  });
  return (
    <Box
      ref={setNodeRef}
      {...listeners}
      {...attributes}
      sx={{
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'space-between',
        p: 1,
        bgcolor: 'rgba(255,255,255,0.03)',
        border: '1px solid rgba(255,255,255,0.08)',
        borderRadius: 1,
        cursor: 'grab',
        opacity: isDragging ? 0.4 : 1,
        '&:hover': { borderColor: '#00D4FF', bgcolor: 'rgba(0,212,255,0.06)' },
      }}
    >
      <Stack direction="row" alignItems="center" gap={1} sx={{ minWidth: 0 }}>
        <Icon fontSize="small" color="action" />
        <Box sx={{ minWidth: 0 }}>
          <Stack direction="row" alignItems="center" gap={0.5}>
            <Typography variant="body2" fontWeight={700} noWrap>{label}</Typography>
            <CoreFieldBadge isCore={isCoreDefault} size={12} />
          </Stack>
          <Typography variant="caption" color="text.secondary" noWrap display="block">{description}</Typography>
        </Box>
      </Stack>
      <Tooltip title={`Add ${label}`}>
        <IconButton size="small" onClick={onAdd}>
          <AddIcon fontSize="small" />
        </IconButton>
      </Tooltip>
    </Box>
  );
};

export default PageStudioPalette;
