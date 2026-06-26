import React from 'react';
import { Box, Typography, Card, CardActionArea, Grid, IconButton } from '@mui/material';
import { 
    TableChart as TableIcon, 
    ShowChart as ChartIcon, 
    Description as FormIcon, 
    FormatShapes as LayoutIcon,
    SmartButton as ButtonIcon,
    Notes as DetailIcon,
    Numbers as KPIIcon
} from '@mui/icons-material';

const COMPONENT_TYPES = [
    { type: 'Row', icon: <LayoutIcon />, group: 'Layout' },
    { type: 'Column', icon: <LayoutIcon />, group: 'Layout' },
    { type: 'Table', icon: <TableIcon />, group: 'Data' },
    { type: 'LineChart', icon: <ChartIcon />, group: 'Data' },
    { type: 'Form', icon: <FormIcon />, group: 'Data' },
    { type: 'KPIGroup', icon: <KPIIcon />, group: 'Data' },
    { type: 'DetailPanel', icon: <DetailIcon />, group: 'Data' },
];

const ComponentPalette: React.FC = () => {
    const handleDragStart = (e: React.DragEvent, type: string) => {
        e.dataTransfer.setData('componentType', type);
    };

    return (
        <Box sx={{ p: 2 }}>
            <Typography variant="overline" color="textSecondary" fontWeight="bold">Layout</Typography>
            <Grid container spacing={1} sx={{ mb: 3, mt: 0.5 }}>
                {COMPONENT_TYPES.filter(c => c.group === 'Layout').map(c => (
                    <Grid item xs={6} key={c.type}>
                        <Card variant="outlined" sx={{ borderRadius: 2 }}>
                            <CardActionArea 
                                draggable 
                                onDragStart={(e) => handleDragStart(e, c.type)}
                                sx={{ p: 1, display: 'flex', flexDirection: 'column', alignItems: 'center' }}
                            >
                                <Box sx={{ color: 'primary.main', mb: 0.5 }}>{c.icon}</Box>
                                <Typography variant="caption" fontWeight="600">{c.type}</Typography>
                            </CardActionArea>
                        </Card>
                    </Grid>
                ))}
            </Grid>

            <Typography variant="overline" color="textSecondary" fontWeight="bold">Data Displays</Typography>
            <Grid container spacing={1} sx={{ mt: 0.5 }}>
                {COMPONENT_TYPES.filter(c => c.group === 'Data').map(c => (
                    <Grid item xs={6} key={c.type}>
                        <Card variant="outlined" sx={{ borderRadius: 2 }}>
                            <CardActionArea 
                                draggable 
                                onDragStart={(e) => handleDragStart(e, c.type)}
                                sx={{ p: 1, display: 'flex', flexDirection: 'column', alignItems: 'center' }}
                            >
                                <Box sx={{ color: 'secondary.main', mb: 0.5 }}>{c.icon}</Box>
                                <Typography variant="caption" fontWeight="600">{c.type}</Typography>
                            </CardActionArea>
                        </Card>
                    </Grid>
                ))}
            </Grid>
        </Box>
    );
};

export default ComponentPalette;
