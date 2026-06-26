import React, { useState } from 'react';
import {
    Box,
    Typography,
    Paper,
    Stack,
    TextField,
    Button,
    Divider,
    IconButton,
    Grid,
    Slider
} from '@mui/material';
import {
    Palette as PaletteIcon,
    Save as SaveIcon,
    History as HistoryIcon,
    Refresh as ResetIcon
} from '@mui/icons-material';

export const TenantBrandingEditor: React.FC = () => {
    const [primaryColor, setPrimaryColor] = useState('#3b82f6');
    const [secondaryColor, setSecondaryColor] = useState('#6366f1');
    const [borderRadius, setBorderRadius] = useState(8);

    const handleSave = () => {
        console.log('Saving branding overrides', { primaryColor, secondaryColor, borderRadius });
    };

    return (
        <Box sx={{ p: 4, maxWidth: 800 }}>
            <Box sx={{ display: 'flex', alignItems: 'center', mb: 4 }}>
                <PaletteIcon color="primary" sx={{ mr: 1, fontSize: 32 }} />
                <Typography variant="h5" fontWeight="bold">Tenant Branding Overrides</Typography>
            </Box>

            <Grid container spacing={4}>
                <Grid item xs={12} md={6}>
                    <Paper variant="outlined" sx={{ p: 3, borderRadius: 3 }}>
                        <Typography variant="subtitle2" gutterBottom fontWeight="bold">Colors</Typography>
                        <Stack spacing={3}>
                            <Box>
                                <Typography variant="caption" color="textSecondary" display="block" gutterBottom>Primary Brand Color</Typography>
                                <Box sx={{ display: 'flex', gap: 2 }}>
                                    <TextField 
                                        size="small" 
                                        value={primaryColor} 
                                        onChange={(e) => setPrimaryColor(e.target.value)}
                                        sx={{ flex: 1 }}
                                    />
                                    <Box sx={{ width: 40, height: 40, bgcolor: primaryColor, borderRadius: 1, border: '1px solid #ccc' }} />
                                </Box>
                            </Box>
                            <Box>
                                <Typography variant="caption" color="textSecondary" display="block" gutterBottom>Secondary Color</Typography>
                                <Box sx={{ display: 'flex', gap: 2 }}>
                                    <TextField 
                                        size="small" 
                                        value={secondaryColor} 
                                        onChange={(e) => setSecondaryColor(e.target.value)}
                                        sx={{ flex: 1 }}
                                    />
                                    <Box sx={{ width: 40, height: 40, bgcolor: secondaryColor, borderRadius: 1, border: '1px solid #ccc' }} />
                                </Box>
                            </Box>
                        </Stack>
                    </Paper>
                </Grid>

                <Grid item xs={12} md={6}>
                    <Paper variant="outlined" sx={{ p: 3, borderRadius: 3 }}>
                        <Typography variant="subtitle2" gutterBottom fontWeight="bold">Styling</Typography>
                        <Box sx={{ mt: 2 }}>
                            <Typography variant="caption" color="textSecondary" display="block" gutterBottom>Corner Rounding (px)</Typography>
                            <Slider 
                                value={borderRadius} 
                                onChange={(_, v) => setBorderRadius(v as number)}
                                min={0}
                                max={24}
                                valueLabelDisplay="auto"
                            />
                            <Box sx={{ 
                                mt: 2, 
                                p: 2, 
                                border: '1px solid #ccc', 
                                borderRadius: borderRadius / 4, 
                                textAlign: 'center',
                                bgcolor: 'grey.50'
                            }}>
                                <Typography variant="caption">Preview Box</Typography>
                            </Box>
                        </Box>
                    </Paper>
                </Grid>
            </Grid>

            <Box sx={{ mt: 4, display: 'flex', justifyContent: 'flex-end', gap: 2 }}>
                <Button variant="outlined" startIcon={<ResetIcon />}>Reset to Defaults</Button>
                <Button variant="contained" startIcon={<SaveIcon />} onClick={handleSave}>Save Branding</Button>
            </Box>
        </Box>
    );
};
