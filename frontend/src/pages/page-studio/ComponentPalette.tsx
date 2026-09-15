import React from 'react';
import { Box, Typography, Card, CardActionArea, Grid } from '@mui/material';
import { useDraggable } from '@dnd-kit/core';
import {
    TableChart as TableIcon,
    ShowChart as ChartIcon,
    Description as FormIcon,
    FormatShapes as LayoutIcon,
    SmartButton as ButtonIcon,
    Notes as DetailIcon,
    Numbers as KPIIcon,
    TextFields as TextIcon,
    FilterAlt as FilterIcon,
    Link as LinkIcon,
    ViewSidebar as PanelIcon,
    Dashboard as TileIcon,
} from '@mui/icons-material';

const COMPONENT_TYPES = [
    { type: 'Row', icon: <LayoutIcon />, group: 'Layout' },
    { type: 'Column', icon: <LayoutIcon />, group: 'Layout' },
    { type: 'Panel', icon: <PanelIcon />, group: 'Layout' },
    { type: 'Text', icon: <TextIcon />, group: 'Data' },
    { type: 'Table', icon: <TableIcon />, group: 'Data' },
    { type: 'LineChart', icon: <ChartIcon />, group: 'Data' },
    { type: 'Form', icon: <FormIcon />, group: 'Data' },
    { type: 'KPIGroup', icon: <KPIIcon />, group: 'Data' },
    { type: 'DetailPanel', icon: <DetailIcon />, group: 'Data' },
    { type: 'Slicer', icon: <FilterIcon />, group: 'Filters & Actions' },
    { type: 'Button', icon: <ButtonIcon />, group: 'Filters & Actions' },
    { type: 'Hyperlink', icon: <LinkIcon />, group: 'Filters & Actions' },
    { type: 'Tile', icon: <TileIcon />, group: 'Filters & Actions' },
];

const GROUP_LABELS: Record<string, string> = {
    Layout: 'Layout',
    Data: 'Data Displays',
    'Filters & Actions': 'Filters & Actions',
};
const GROUP_COLORS: Record<string, string> = {
    Layout: 'primary.main',
    Data: 'secondary.main',
    'Filters & Actions': 'warning.main',
};
const GROUP_ORDER = ['Layout', 'Data', 'Filters & Actions'];

/**
 * A palette tile's own useDraggable instance - one per COMPONENT_TYPES entry.
 * Carries {kind: 'component', componentType} as its drag data, read back in
 * LayoutCanvas.tsx's onDragEnd (via a shared DndContext mounted in
 * PageEditor.tsx) instead of the old dataTransfer.setData('componentType', ...).
 */
const PaletteTile: React.FC<{ type: string; icon: React.ReactNode; color: string }> = ({ type, icon, color }) => {
    const { attributes, listeners, setNodeRef, isDragging } = useDraggable({
        id: `palette-${type}`,
        data: { kind: 'component', componentType: type },
    });

    return (
        <Card variant="outlined" sx={{ borderRadius: 2, opacity: isDragging ? 0.4 : 1 }}>
            <CardActionArea
                ref={setNodeRef}
                {...listeners}
                {...attributes}
                sx={{ p: 1, display: 'flex', flexDirection: 'column', alignItems: 'center', cursor: 'grab' }}
            >
                <Box sx={{ color, mb: 0.5 }}>{icon}</Box>
                <Typography variant="caption" fontWeight="600">{type}</Typography>
            </CardActionArea>
        </Card>
    );
};

const ComponentPalette: React.FC = () => {
    return (
        <Box sx={{ p: 2 }}>
            {GROUP_ORDER.map((group) => (
                <Box key={group} sx={{ mb: 3 }}>
                    <Typography variant="overline" color="textSecondary" fontWeight="bold">{GROUP_LABELS[group]}</Typography>
                    <Grid container spacing={1} sx={{ mt: 0.5 }}>
                        {COMPONENT_TYPES.filter(c => c.group === group).map(c => (
                            <Grid size={6} key={c.type}>
                                <PaletteTile type={c.type} icon={c.icon} color={GROUP_COLORS[group]} />
                            </Grid>
                        ))}
                    </Grid>
                </Box>
            ))}
        </Box>
    );
};

export default ComponentPalette;
