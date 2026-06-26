import React, { useState } from 'react';
import { HexColorPicker } from 'react-colorful';
import {
  Box,
  Paper,
  TextField,
  Typography,
  Stack,
  Divider,
  Chip,
  Grid,
} from '@mui/material';
import { isValidHexColor } from '../utils/colorPalette';

// Add react-colorful styles inline
const colorfulStyles = `
  .react-colorful {
    width: 100%;
    height: 200px;
    max-width: 300px;
  }
  
  .react-colorful__picker {
    border-radius: 8px;
    box-shadow: 0 2px 8px rgba(0,0,0,0.1);
  }
  
  .react-colorful__pointer {
    border-radius: 50%;
    width: 16px;
    height: 16px;
    border: 3px solid #fff;
    box-shadow: 0 0 4px rgba(0,0,0,0.3);
  }
  
  .react-colorful__hue,
  .react-colorful__alpha {
    height: 20px;
    border-radius: 4px;
    margin-top: 8px;
  }
`;

// Inject styles on mount
if (typeof document !== 'undefined') {
  const styleEl = document.createElement('style');
  styleEl.textContent = colorfulStyles;
  document.head.appendChild(styleEl);
}

interface ProfessionalColorPickerProps {
  color: string;
  onChange: (color: string) => void;
  label?: string;
  showRecent?: boolean;
}

const DEFAULT_PALETTE = [
  '#3B82F6', '#8B5CF6', '#EC4899', '#F43F5E',
  '#F97316', '#EAB308', '#22C55E', '#10B981',
  '#14B8A6', '#06B6D4', '#0EA5E9', '#6366F1',
  '#000000', '#666666', '#999999', '#FFFFFF',
];

export const ProfessionalColorPicker: React.FC<ProfessionalColorPickerProps> = ({
  color,
  onChange,
  label = 'Color',
  showRecent = true,
}) => {
  const [recentColors, setRecentColors] = useState<string[]>([]);
  const [customInput, setCustomInput] = useState(color);
  const [showAdvanced, setShowAdvanced] = useState(false);

  const handleColorChange = (newColor: string) => {
    if (isValidHexColor(newColor)) {
      onChange(newColor);
      setCustomInput(newColor);
      // Add to recent colors
      if (!recentColors.includes(newColor)) {
        setRecentColors([newColor, ...recentColors].slice(0, 5));
      }
    }
  };

  const handleCustomInput = (value: string) => {
    setCustomInput(value);
    if (isValidHexColor(value)) {
      handleColorChange(value);
    }
  };

  return (
    <Stack spacing={2} sx={{ width: '100%' }}>
      <Box>
        <Typography variant="subtitle2" sx={{ mb: 1, fontWeight: 600 }}>
          {label}
        </Typography>
        
        {/* Current Color Display */}
        <Paper
          sx={{
            display: 'flex',
            alignItems: 'center',
            gap: 2,
            p: 1.5,
            mb: 2,
            backgroundColor: '#f5f5f5',
          }}
        >
          <Box
            sx={{
              width: 50,
              height: 50,
              backgroundColor: color,
              borderRadius: 1,
              border: `2px solid #ddd`,
              cursor: 'pointer',
            }}
          />
          <TextField
            size="small"
            value={customInput}
            onChange={(e) => handleCustomInput(e.target.value)}
            placeholder="#000000"
            inputProps={{ style: { fontFamily: 'monospace', fontSize: '0.9rem' } }}
            sx={{ flex: 1 }}
          />
        </Paper>

        {/* Main Color Picker */}
        <Box
          sx={{
            display: 'flex',
            justifyContent: 'center',
            mb: 2,
            '& .react-colorful': {
              width: '100%',
              maxWidth: 300,
              height: 200,
            },
            '& .react-colorful__picker': {
              borderRadius: '8px',
              boxShadow: '0 2px 8px rgba(0,0,0,0.1)',
            },
            '& .react-colorful__hue': {
              height: 30,
              borderRadius: '4px',
              marginTop: '8px',
            },
            '& .react-colorful__alpha': {
              height: 20,
              borderRadius: '4px',
              marginTop: '8px',
            },
            '& .react-colorful__pointer': {
              borderRadius: '50%',
              width: 16,
              height: 16,
              border: '3px solid #fff',
              boxShadow: '0 0 4px rgba(0,0,0,0.3)',
            },
          }}
        >
          <HexColorPicker color={color} onChange={handleColorChange} />
        </Box>

        {/* Quick Palette */}
        <Box>
          <Typography variant="caption" sx={{ display: 'block', mb: 1, color: '#666' }}>
            Quick Select:
          </Typography>
          <Grid container spacing={1} sx={{ mb: 2 }}>
            {DEFAULT_PALETTE.map((paletteColor) => (
              <Grid item xs={4} sm={3} md={2} key={paletteColor}>
                <Box
                  onClick={() => handleColorChange(paletteColor)}
                  sx={{
                    width: '100%',
                    height: 40,
                    backgroundColor: paletteColor,
                    borderRadius: 1,
                    cursor: 'pointer',
                    border: color === paletteColor ? '3px solid #333' : '2px solid #ddd',
                    transition: 'all 0.2s ease',
                    '&:hover': {
                      transform: 'scale(1.05)',
                      boxShadow: '0 2px 8px rgba(0,0,0,0.2)',
                    },
                  }}
                  title={paletteColor}
                />
              </Grid>
            ))}
          </Grid>
        </Box>

        {/* Recent Colors */}
        {showRecent && recentColors.length > 0 && (
          <>
            <Divider sx={{ my: 1.5 }} />
            <Box>
              <Typography variant="caption" sx={{ display: 'block', mb: 1, color: '#666' }}>
                Recently Used:
              </Typography>
              <Stack direction="row" spacing={1} sx={{ flexWrap: 'wrap' }}>
                {recentColors.map((recentColor) => (
                  <Chip
                    key={recentColor}
                    label={recentColor}
                    onClick={() => handleColorChange(recentColor)}
                    icon={
                      <Box
                        sx={{
                          width: 16,
                          height: 16,
                          backgroundColor: recentColor,
                          borderRadius: '2px',
                          border: '1px solid rgba(0,0,0,0.1)',
                        }}
                      />
                    }
                    variant={color === recentColor ? 'filled' : 'outlined'}
                    size="small"
                    sx={{ cursor: 'pointer' }}
                  />
                ))}
              </Stack>
            </Box>
          </>
        )}
      </Box>
    </Stack>
  );
};
