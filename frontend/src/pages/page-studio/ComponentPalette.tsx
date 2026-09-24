import React from 'react';
import { Box, Typography, Card, CardActionArea, Grid } from '@mui/material';
import { useDraggable } from '@dnd-kit/core';
import {
    TableChart as TableIcon,
    ShowChart as ChartIcon,
    Description as FormIcon,
    FormatShapes as LayoutIcon,
    SmartButton as ButtonIcon,
    Numbers as KPIIcon,
    TextFields as TextIcon,
    FilterAlt as FilterIcon,
    Link as LinkIcon,
    ViewSidebar as PanelIcon,
    Dashboard as TileIcon,
} from '@mui/icons-material';

/**
 * A palette entry. `availableIn` is the studio discriminator — omitted
 * defaults to `['page']` (every widget below this comment, unchanged).
 * `acceptsRelatedBODrop` is a renderer *capability* flag, not a renderer:
 * it says a widget can be the target of a related-BO drag/drop
 * (`ensureRelatedDataSource`/`masterFilter`, studio-core/binding), not that
 * one is wired up yet. Extending in place here (not extracting to
 * studio-core) is deliberate: this file, like `types/pageStudio.ts`, would
 * be a HIGH-risk multi-file move per this repo's GitNexus policy (same
 * class of risk 1.1 hit with `buildBODataSource`) — Report Studio should
 * cross-module import from here when it exists, and the file moves to
 * studio-core only as its own impact-analyzed ticket once there's a real
 * second consumer. See HANDOFF_REPORT_BUILDER_SPINE_PLAN.md ticket 1.3.
 */
interface WidgetDefinition {
    type: string;
    icon: React.ReactNode;
    group: string;
    /** Which studios offer this widget in their palette. Omitted = ['page']. */
    availableIn?: ('page' | 'report')[];
    acceptsRelatedBODrop?: boolean;
}

const COMPONENT_TYPES: WidgetDefinition[] = [
    { type: 'Row', icon: <LayoutIcon />, group: 'Layout' },
    { type: 'Column', icon: <LayoutIcon />, group: 'Layout' },
    { type: 'Panel', icon: <PanelIcon />, group: 'Layout' },
    { type: 'Text', icon: <TextIcon />, group: 'Data' },
    { type: 'Table', icon: <TableIcon />, group: 'Data' },
    { type: 'LineChart', icon: <ChartIcon />, group: 'Data' },
    { type: 'Form', icon: <FormIcon />, group: 'Data' },
    { type: 'KPIGroup', icon: <KPIIcon />, group: 'Data' },
    { type: 'Slicer', icon: <FilterIcon />, group: 'Filters & Actions' },
    { type: 'Button', icon: <ButtonIcon />, group: 'Filters & Actions' },
    { type: 'FixCommand', icon: <ButtonIcon />, group: 'Filters & Actions' },
    { type: 'Hyperlink', icon: <LinkIcon />, group: 'Filters & Actions' },
    { type: 'Tile', icon: <TileIcon />, group: 'Filters & Actions' },
    // Report Studio only — band types. Not shown in Page Studio's palette
    // (filtered below) and not rendered by PageComponentRenderer; they
    // become live once Phase 3 builds ReportCanvas/ReportBandDesigner.
    { type: 'Band', icon: <PanelIcon />, group: 'Report Bands', availableIn: ['report'], acceptsRelatedBODrop: true },
    { type: 'GroupHeader', icon: <LayoutIcon />, group: 'Report Bands', availableIn: ['report'] },
    { type: 'GroupFooter', icon: <LayoutIcon />, group: 'Report Bands', availableIn: ['report'] },
    { type: 'PageHeader', icon: <LayoutIcon />, group: 'Report Bands', availableIn: ['report'] },
    { type: 'PageFooter', icon: <LayoutIcon />, group: 'Report Bands', availableIn: ['report'] },
    { type: 'FieldText', icon: <TextIcon />, group: 'Report Bands', availableIn: ['report'] },
    { type: 'Image', icon: <TileIcon />, group: 'Report Bands', availableIn: ['report'] },
    { type: 'Barcode', icon: <TileIcon />, group: 'Report Bands', availableIn: ['report'] },
];

const GROUP_LABELS: Record<string, string> = {
    Layout: 'Layout',
    Data: 'Data Displays',
    'Filters & Actions': 'Filters & Actions',
    'Report Bands': 'Report Bands',
};
const GROUP_COLORS: Record<string, string> = {
    Layout: 'primary.main',
    Data: 'secondary.main',
    'Filters & Actions': 'warning.main',
    'Report Bands': 'info.main',
};
const GROUP_ORDER = ['Layout', 'Data', 'Filters & Actions', 'Report Bands'];

/** Page Studio's palette guard: report-only widgets never appear here, full stop. */
const PAGE_STUDIO_COMPONENT_TYPES = COMPONENT_TYPES.filter((c) => (c.availableIn ?? ['page']).includes('page'));

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
            {GROUP_ORDER.filter((group) => PAGE_STUDIO_COMPONENT_TYPES.some((c) => c.group === group)).map((group) => (
                <Box key={group} sx={{ mb: 3 }}>
                    <Typography variant="overline" color="textSecondary" fontWeight="bold">{GROUP_LABELS[group]}</Typography>
                    <Grid container spacing={1} sx={{ mt: 0.5 }}>
                        {PAGE_STUDIO_COMPONENT_TYPES.filter(c => c.group === group).map(c => (
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
