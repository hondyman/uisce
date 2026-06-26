import React, { useState, useMemo } from 'react';
import {
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  Button,
  Box,
  Typography,
  Paper,
  TextField,
  Grid,
  Chip,
  useTheme,
  alpha,
  Tooltip,
} from '@mui/material';
import AddIcon from '@mui/icons-material/Add';
import CheckCircleIcon from '@mui/icons-material/CheckCircle';
import {
  DEFAULT_COLORS,
  type ColorPalette,
  isValidHexColor,
  getColorDistance,
  suggestNextColor,
} from '../utils/colorPalette';

interface ColorPaletteEditorProps {
  open: boolean;
  onClose: () => void;
  usedColors?: string[];
  onColorSelect?: (color: string) => void;
}

export const ColorPaletteEditor: React.FC<ColorPaletteEditorProps> = ({
  open,
  onClose,
  usedColors = [],
  onColorSelect,
}) => {
  const theme = useTheme();
  const [customColors, setCustomColors] = useState<ColorPalette[]>([]);
  const [newColor, setNewColor] = useState('');
  const [selectedColor, setSelectedColor] = useState<string | null>(null);

  const handleAddColor = () => {
    if (!newColor || !isValidHexColor(newColor)) {
      alert('Please enter a valid hex color (e.g., #FF5733)');
      return;
    }

    const customColor: ColorPalette = {
      id: `custom-${Date.now()}`,
      name: newColor,
      hex: newColor,
      isCustom: true,
    };

    setCustomColors([...customColors, customColor]);
    setNewColor('');
  };

  const handleDeleteColor = (colorId: string) => {
    setCustomColors(customColors.filter(c => c.id !== colorId));
  };

  const handleSelectColor = (hex: string) => {
    setSelectedColor(hex);
    if (onColorSelect) {
      onColorSelect(hex);
      onClose();
    }
  };

  const getColorConflict = (hex: string): boolean => {
    return usedColors.some(used => getColorDistance(hex, used) < 100);
  };

  const suggestedColor = useMemo(() => {
    return suggestNextColor(usedColors, DEFAULT_COLORS);
  }, []);

  return (
    <Dialog open={open} onClose={onClose} maxWidth="md" fullWidth>
      <DialogTitle>Color Palette</DialogTitle>
      <DialogContent sx={{ pt: 2, display: 'flex', flexDirection: 'column', gap: 2 }}>
        {/* Suggested Color */}
        <Box>
          <Typography variant="subtitle2" fontWeight="bold" gutterBottom>
            Suggested Color
          </Typography>
          <Box
            sx={{
              display: 'flex',
              gap: 1,
              alignItems: 'center',
              p: 2,
              border: `2px solid ${theme.palette.info.main}`,
              borderRadius: 1,
              bgcolor: alpha(theme.palette.info.main, 0.05),
            }}
          >
            <Box
              sx={{
                width: 40,
                height: 40,
                borderRadius: 1,
                bgcolor: suggestedColor,
                border: `2px solid ${theme.palette.divider}`,
              }}
            />
            <Typography variant="body2" fontFamily="monospace" sx={{ flex: 1 }}>
              {suggestedColor}
            </Typography>
            <Button
              size="small"
              variant="outlined"
              onClick={() => handleSelectColor(suggestedColor)}
            >
              Use
            </Button>
          </Box>
        </Box>

        {/* Default Palette */}
        <Box>
          <Typography variant="subtitle2" fontWeight="bold" gutterBottom>
            Default Colors
          </Typography>
          <Grid container spacing={1}>
            {DEFAULT_COLORS.map((color: ColorPalette) => {
              const isConflicting = getColorConflict(color.hex);
              const isSelected = selectedColor === color.hex;

              return (
                <Grid item xs={6} sm={4} md={3} key={color.id}>
                  <Tooltip
                    title={isConflicting ? 'Too similar to existing color' : color.name}
                  >
                    <Paper
                      elevation={0}
                      sx={{
                        p: 1.5,
                        borderRadius: 2,
                        border: isSelected
                          ? `3px solid ${theme.palette.primary.main}`
                          : `1px solid ${theme.palette.divider}`,
                        bgcolor: 'background.paper',
                        cursor: isConflicting ? 'not-allowed' : 'pointer',
                        opacity: isConflicting ? 0.5 : 1,
                        transition: 'all 0.2s',
                        '&:hover': !isConflicting
                          ? {
                              transform: 'translateY(-2px)',
                              boxShadow: theme.shadows[4],
                            }
                          : {},
                      }}
                      onClick={() => !isConflicting && handleSelectColor(color.hex)}
                    >
                      <Box
                        sx={{
                          width: '100%',
                          height: 60,
                          borderRadius: 1,
                          bgcolor: color.hex,
                          mb: 1,
                          display: 'flex',
                          alignItems: 'center',
                          justifyContent: 'center',
                        }}
                      >
                        {isSelected && (
                          <CheckCircleIcon
                            sx={{
                              color: 'white',
                              filter: 'drop-shadow(0 1px 3px rgba(0,0,0,0.5))',
                            }}
                          />
                        )}
                      </Box>
                      <Typography variant="caption" display="block" align="center">
                        {color.name}
                      </Typography>
                    </Paper>
                  </Tooltip>
                </Grid>
              );
            })}
          </Grid>
        </Box>

        {/* Custom Colors */}
        {customColors.length > 0 && (
          <Box>
            <Typography variant="subtitle2" fontWeight="bold" gutterBottom>
              Custom Colors
            </Typography>
            <Box sx={{ display: 'flex', gap: 1, flexWrap: 'wrap' }}>
              {customColors.map(color => (
                <Chip
                  key={color.id}
                  label={color.hex}
                  onDelete={() => handleDeleteColor(color.id)}
                  icon={
                    <Box
                      sx={{
                        width: 12,
                        height: 12,
                        borderRadius: '50%',
                        bgcolor: color.hex,
                        border: `1px solid ${theme.palette.divider}`,
                      }}
                    />
                  }
                  onClick={() => handleSelectColor(color.hex)}
                  sx={{
                    cursor: 'pointer',
                    border:
                      selectedColor === color.hex
                        ? `2px solid ${theme.palette.primary.main}`
                        : `1px solid ${theme.palette.divider}`,
                  }}
                />
              ))}
            </Box>
          </Box>
        )}

        {/* Add Custom Color */}
        <Box sx={{ borderTop: `1px solid ${theme.palette.divider}`, pt: 2 }}>
          <Typography variant="subtitle2" fontWeight="bold" gutterBottom>
            Add Custom Color
          </Typography>
          <Box sx={{ display: 'flex', gap: 1 }}>
            <TextField
              type="text"
              placeholder="#FF5733"
              value={newColor}
              onChange={e => setNewColor(e.target.value)}
              size="small"
              inputProps={{ pattern: '^#[0-9A-Fa-f]{6}$' }}
              sx={{ flex: 1 }}
            />
            {newColor && isValidHexColor(newColor) && (
              <Box
                sx={{
                  width: 40,
                  height: 40,
                  borderRadius: 1,
                  bgcolor: newColor,
                  border: `1px solid ${theme.palette.divider}`,
                }}
              />
            )}
            <Button
              variant="outlined"
              size="small"
              startIcon={<AddIcon />}
              onClick={handleAddColor}
              disabled={!newColor || !isValidHexColor(newColor)}
            >
              Add
            </Button>
          </Box>
        </Box>
      </DialogContent>
      <DialogActions>
        <Button onClick={onClose}>Close</Button>
      </DialogActions>
    </Dialog>
  );
};
