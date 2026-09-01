import React from 'react';
import { Box, Typography, Card, CardActionArea, Grid, Chip } from '@mui/material';
import { 
    TableChart as TableIcon, 
    ShowChart as ChartIcon, 
    Description as FormIcon, 
    FormatShapes as LayoutIcon,
    SmartButton as ButtonIcon,
    Notes as DetailIcon,
    Numbers as KPIIcon,
    ViewColumn as GridIcon,
    GridView as CardGridIcon,
    AccountTree as TreeIcon,
    DynamicForm as DynamicFormIcon
} from '@mui/icons-material';

const COMPONENT_TYPES = [
    { type: 'Row', label: 'Row Container', icon: <LayoutIcon />, group: 'Layout', desc: 'Horizontal flex container' },
    { type: 'Column', label: 'Column Container', icon: <LayoutIcon />, group: 'Layout', desc: 'Vertical stack container' },
    { type: 'BO_FORM', label: 'Entity Form', icon: <DynamicFormIcon />, group: 'Forms & CRUD', desc: 'Type-aware CRUD field form' },
    { type: 'Table', label: 'Data Grid', icon: <TableIcon />, group: 'Data Displays', desc: 'Tabular grid with sorting & filtering' },
    { type: 'LineChart', label: 'Analytics Chart', icon: <ChartIcon />, group: 'Data Displays', desc: 'Interactive ECharts visual' },
    { type: 'KPIGroup', label: 'KPI Metric Cards', icon: <KPIIcon />, group: 'Data Displays', desc: 'Aggregated metric summaries' },
    { type: 'DetailPanel', label: 'Master-Detail Panel', icon: <DetailIcon />, group: 'Data Displays', desc: 'Side-by-side drilldown' },
];

const ComponentPalette: React.FC = () => {
    const handleDragStart = (e: React.DragEvent, type: string) => {
        e.dataTransfer.setData('componentType', type);
        e.dataTransfer.effectAllowed = 'copy';
    };

    const groups = ['Layout', 'Forms & CRUD', 'Data Displays'];

    return (
        <Box sx={{ p: 1.5 }}>
            {groups.map((grp) => {
                const items = COMPONENT_TYPES.filter(c => c.group === grp);
                if (items.length === 0) return null;
                return (
                    <Box key={grp} sx={{ mb: 2.5 }}>
                        <Typography 
                            variant="caption" 
                            sx={{ 
                                fontWeight: 700, 
                                letterSpacing: '0.04em', 
                                color: '#38BDF8', 
                                textTransform: 'uppercase', 
                                fontSize: '0.68rem',
                                display: 'block',
                                mb: 1 
                            }}
                        >
                            {grp}
                        </Typography>
                        <Grid container spacing={1}>
                            {items.map((c) => (
                                <Grid item xs={12} key={c.type}>
                                    <Card 
                                        variant="outlined" 
                                        sx={{ 
                                            bgcolor: '#0F172A', 
                                            borderColor: '#1E293B',
                                            borderRadius: 1.5,
                                            transition: 'all 0.15s ease',
                                            '&:hover': {
                                                borderColor: '#0284C7',
                                                bgcolor: 'rgba(2, 132, 199, 0.08)',
                                            }
                                        }}
                                    >
                                        <CardActionArea 
                                            draggable 
                                            onDragStart={(e) => handleDragStart(e, c.type)}
                                            sx={{ p: 1.2, display: 'flex', alignItems: 'center', justifyContent: 'flex-start', gap: 1.5 }}
                                        >
                                            <Box sx={{ color: '#38BDF8', display: 'flex', alignItems: 'center', p: 0.5, bgcolor: '#1E293B', borderRadius: 1 }}>
                                                {c.icon}
                                            </Box>
                                            <Box sx={{ minWidth: 0, flex: 1 }}>
                                                <Typography variant="body2" fontWeight="700" sx={{ color: '#F8FAFC', fontSize: '0.75rem' }}>
                                                    {c.label || c.type}
                                                </Typography>
                                                <Typography variant="caption" sx={{ color: '#94A3B8', fontSize: '0.65rem', display: 'block' }}>
                                                    {c.desc}
                                                </Typography>
                                            </Box>
                                        </CardActionArea>
                                    </Card>
                                </Grid>
                            ))}
                        </Grid>
                    </Box>
                );
            })}
        </Box>
    );
};

export default ComponentPalette;
