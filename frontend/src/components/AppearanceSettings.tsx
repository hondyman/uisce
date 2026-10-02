import React from 'react';
import { Box, Card, CardActionArea, CardContent, Grid, ToggleButton, ToggleButtonGroup, Typography } from '@mui/material';
import CheckCircleIcon from '@mui/icons-material/CheckCircle';
import { useTheme, type Theme as ThemeMode } from '../contexts/ThemeContext';
import { THEME_STYLES } from '../theme/themeStyles';

/** Pick the visual style (Uisce, Ivy, ...) and light/dark/system mode. Saved per browser. */
export const AppearanceSettings: React.FC = () => {
  const { theme: mode, setTheme, style, setStyle } = useTheme();

  return (
    <Box>
      <Typography variant="h6" fontWeight={700} gutterBottom>Appearance</Typography>
      <Typography variant="caption" color="text.secondary" sx={{ display: 'block', mb: 2 }}>
        Choose a style and colour mode. Your choice is saved in this browser.
      </Typography>

      <Grid container spacing={2} sx={{ mb: 3 }}>
        {THEME_STYLES.map((s) => {
          const selected = style === s.id;
          return (
            <Grid key={s.id} size={{ xs: 12, sm: 6, md: 4 }}>
              <Card variant="outlined" sx={{ borderColor: selected ? 'primary.main' : 'divider', borderWidth: selected ? 2 : 1 }}>
                <CardActionArea onClick={() => setStyle(s.id)} aria-pressed={selected} aria-label={`${s.label} style`}>
                  <CardContent>
                    <Box sx={{ display: 'flex', alignItems: 'center', gap: 1.5 }}>
                      <Box sx={{ width: 28, height: 28, borderRadius: '50%', bgcolor: s.accent, flexShrink: 0 }} />
                      <Box sx={{ flex: 1, minWidth: 0 }}>
                        <Typography variant="subtitle1" fontWeight={700}>{s.label}</Typography>
                        <Typography variant="caption" color="text.secondary">{s.description}</Typography>
                      </Box>
                      {selected && <CheckCircleIcon color="primary" fontSize="small" />}
                    </Box>
                  </CardContent>
                </CardActionArea>
              </Card>
            </Grid>
          );
        })}
      </Grid>

      <ToggleButtonGroup exclusive size="small" value={mode} onChange={(_, v: ThemeMode | null) => v && setTheme(v)} aria-label="Colour mode">
        <ToggleButton value="light" sx={{ textTransform: 'none', px: 2 }}>Light</ToggleButton>
        <ToggleButton value="dark" sx={{ textTransform: 'none', px: 2 }}>Dark</ToggleButton>
        <ToggleButton value="system" sx={{ textTransform: 'none', px: 2 }}>System</ToggleButton>
      </ToggleButtonGroup>
    </Box>
  );
};

export default AppearanceSettings;
