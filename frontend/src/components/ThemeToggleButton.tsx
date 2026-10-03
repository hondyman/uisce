import { useState } from 'react';
import type { FC, MouseEvent } from 'react';
import { Moon, Sun, Monitor } from 'lucide-react';
import { useTheme } from '../contexts/ThemeContext';
import { IconButton, Tooltip, Menu, MenuItem, ListSubheader, Divider, Box } from '@mui/material';
import { THEME_STYLES } from '../theme/themeStyles';

interface ThemeToggleButtonProps {
  /**
   * Optional CSS class for styling
   */
  className?: string;
  /**
   * Show as dropdown menu instead of simple toggle
   */
  showMenu?: boolean;
}

/**
 * Theme toggle button component that supports light, dark, and system preferences.
 * 
 * Usage:
 * ```tsx
 * <ThemeToggleButton />
 * <ThemeToggleButton showMenu />
 * ```
 */
export const ThemeToggleButton: FC<ThemeToggleButtonProps> = ({ 
  className,
  showMenu = true 
}) => {
  const { theme, effectiveTheme, setTheme, style, setStyle } = useTheme();
  const [anchorEl, setAnchorEl] = useState<null | HTMLElement>(null);

  const handleMenuOpen = (event: MouseEvent<HTMLElement>) => {
    setAnchorEl(event.currentTarget);
  };

  const handleMenuClose = () => {
    setAnchorEl(null);
  };

  const handleThemeSelect = (newTheme: 'light' | 'dark' | 'system') => {
    setTheme(newTheme);
    handleMenuClose();
  };

  const getIcon = () => {
    switch (effectiveTheme) {
      case 'dark':
        return <Moon className="w-5 h-5" />;
      case 'light':
        return <Sun className="w-5 h-5" />;
      default:
        return <Sun className="w-5 h-5" />;
    }
  };

  const getLabel = () => {
    switch (theme) {
      case 'light':
        return 'Light mode';
      case 'dark':
        return 'Dark mode';
      case 'system':
        return 'System preference';
      default:
        return 'Toggle theme';
    }
  };

  if (showMenu) {
    return (
      <>
        <Tooltip title={getLabel()}>
          <IconButton 
            onClick={handleMenuOpen}
            className={className}
            aria-label="theme-toggle"
          >
            {getIcon()}
          </IconButton>
        </Tooltip>
        <Menu
          anchorEl={anchorEl}
          open={Boolean(anchorEl)}
          onClose={handleMenuClose}
          anchorOrigin={{
            vertical: 'bottom',
            horizontal: 'right',
          }}
          transformOrigin={{
            vertical: 'top',
            horizontal: 'right',
          }}
        >
          <ListSubheader disableSticky>Style</ListSubheader>
          {THEME_STYLES.map((s) => (
            <MenuItem key={s.id} selected={style === s.id} onClick={() => { setStyle(s.id); handleMenuClose(); }}>
              <Box component="span" sx={{ width: 12, height: 12, borderRadius: '50%', bgcolor: s.accent, mr: 1.5, flexShrink: 0 }} />
              {s.label}
            </MenuItem>
          ))}
          <Divider />
          <ListSubheader disableSticky>Mode</ListSubheader>
          <MenuItem 
            selected={theme === 'light'}
            onClick={() => handleThemeSelect('light')}
          >
            <Sun className="w-4 h-4 mr-2" />
            Light
          </MenuItem>
          <MenuItem 
            selected={theme === 'dark'}
            onClick={() => handleThemeSelect('dark')}
          >
            <Moon className="w-4 h-4 mr-2" />
            Dark
          </MenuItem>
          <MenuItem 
            selected={theme === 'system'}
            onClick={() => handleThemeSelect('system')}
          >
            <Monitor className="w-4 h-4 mr-2" />
            System
          </MenuItem>
        </Menu>
      </>
    );
  }

  // Simple toggle between light and dark
  return (
    <Tooltip title={getLabel()}>
      <IconButton 
        onClick={() => {
          setTheme(effectiveTheme === 'dark' ? 'light' : 'dark');
        }}
        className={className}
        aria-label="theme-toggle"
      >
        {getIcon()}
      </IconButton>
    </Tooltip>
  );
};

export default ThemeToggleButton;
