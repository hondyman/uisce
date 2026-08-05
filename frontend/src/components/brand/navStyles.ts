import { useMemo } from 'react';
import { useTheme } from '../../contexts/ThemeContext';

export interface NavStyles {
  isDark: boolean;
  appBar: {
    bgcolor: string;
    borderBottom: string;
    boxShadow: string;
  };
  appBarAccent: React.CSSProperties;
  toolbar: React.CSSProperties;
  searchInput: React.CSSProperties;
  categoryBtn: (isActive: boolean) => React.CSSProperties;
  categoryBtnDot: (color: string) => React.CSSProperties;
  menuPaper: {
    bgcolor: string;
    border: string;
    boxShadow: string;
    backdropFilter: string;
    borderRadius: number;
  };
  menuItem: (isSelected: boolean) => React.CSSProperties;
  menuItemLeftBar: React.CSSProperties;
  sidebar: React.CSSProperties;
  sidebarItem: (isActive: boolean) => React.CSSProperties;
  sidebarRail: React.CSSProperties;
  mobileTopBar: React.CSSProperties;
  hoverTransition: React.CSSProperties;
  activeTransition: React.CSSProperties;
}

export function useNavStyles(): NavStyles {
  const { effectiveTheme } = useTheme();
  const isDark = effectiveTheme === 'dark';

  return useMemo(() => {
    if (isDark) {
      return {
        isDark: true,
        appBar: {
          bgcolor: 'var(--nav-appbar-bg)',
          borderBottom: 'var(--nav-appbar-border)',
          boxShadow: 'none',
        },
        appBarAccent: {
          position: 'absolute' as const,
          bottom: 0,
          left: 0,
          right: 0,
          height: '1px',
          background: 'linear-gradient(90deg, transparent 0%, var(--nav-accent) 40%, var(--nav-accent) 60%, transparent 100%)',
          opacity: 0.5,
          pointerEvents: 'none' as const,
        },
        toolbar: {
          flexWrap: 'wrap' as const,
          gap: 1,
          alignItems: 'center',
          background: 'var(--nav-bg)',
        },
        searchInput: {
          bgcolor: 'rgba(255,255,255,0.06)',
          borderRadius: '20px',
          border: '1px solid rgba(255,255,255,0.08)',
          color: 'var(--nav-text)',
          '& .MuiOutlinedInput-notchedOutline': { border: 'none' },
          '&:hover .MuiOutlinedInput-notchedOutline': { border: '1px solid var(--nav-border-accent)' },
          '&.Mui-focused .MuiOutlinedInput-notchedOutline': { border: '1px solid var(--nav-accent)' },
          fontSize: '0.875rem',
          height: '38px',
          pl: 1.5,
          pr: 1.5,
        },
        categoryBtn: (isActive: boolean) => ({
          textTransform: 'none' as const,
          fontWeight: isActive ? 600 : 400,
          color: isActive ? 'var(--nav-accent)' : 'var(--nav-text)',
          borderBottom: isActive ? '2px solid var(--nav-accent)' : '2px solid transparent',
          bgcolor: isActive ? 'var(--nav-accent-muted)' : 'transparent',
          borderRadius: '6px 6px 0 0',
          px: 1.5,
          py: 0.75,
          transition: 'all 150ms cubic-bezier(0.4,0,0.2,1)',
          '&:hover': { bgcolor: 'var(--nav-hover-fill)', color: 'var(--nav-text)' },
        }),
        categoryBtnDot: (color: string) => ({
          width: 6,
          height: 6,
          borderRadius: '50%',
          bgcolor: color,
          flexShrink: 0,
          boxShadow: `0 0 0 2px ${color}22`,
        }),
        menuPaper: {
          bgcolor: 'var(--nav-glass-bg)',
          backdropFilter: 'blur(20px) saturate(180%)',
          WebkitBackdropFilter: 'blur(20px) saturate(180%)',
          border: '1px solid var(--nav-glass-border)',
          boxShadow: 'var(--nav-menu-shadow)',
          borderRadius: 2,
          minWidth: 280,
          maxWidth: '90vw',
          overflow: 'hidden',
        },
        menuItem: (isSelected: boolean) => ({
          display: 'flex',
          alignItems: 'center',
          gap: 1.5,
          py: 1.5,
          px: 2,
          borderLeft: isSelected ? '3px solid var(--nav-accent)' : '3px solid transparent',
          bgcolor: isSelected ? 'var(--nav-item-active)' : 'transparent',
          color: isSelected ? 'var(--nav-accent)' : 'var(--nav-item-text)',
          transition: 'all 150ms cubic-bezier(0.4,0,0.2,1)',
          '&:hover': {
            bgcolor: 'var(--nav-item-hover)',
            color: 'var(--nav-item-text)',
          },
          '&.Mui-disabled': { opacity: 0.5 },
        }),
        menuItemLeftBar: {
          borderLeft: '3px solid var(--nav-accent)',
          bgcolor: 'var(--nav-item-active)',
          color: 'var(--nav-accent)',
        },
        sidebar: {
          bgcolor: 'var(--nav-sidebar-bg)',
          borderRight: 'var(--nav-sidebar-border)',
          boxShadow: 'none',
        },
        sidebarItem: (isActive: boolean) => ({
          display: 'flex',
          alignItems: 'center',
          gap: '0.75rem',
          px: '0.875rem',
          py: '0.75rem',
          mx: 1,
          borderRadius: '8px',
          borderLeft: isActive ? '4px solid var(--nav-rail-accent)' : '4px solid transparent',
          bgcolor: isActive ? 'var(--nav-item-active)' : 'transparent',
          color: isActive ? 'var(--nav-accent)' : 'var(--nav-item-text)',
          transition: 'all 150ms cubic-bezier(0.4,0,0.2,1)',
          '&:hover': {
            bgcolor: isActive ? 'var(--nav-item-active)' : 'var(--nav-item-hover)',
          },
          cursor: 'pointer',
          textDecoration: 'none',
          width: 'calc(100% - 8px)',
          boxSizing: 'border-box',
        }),
        sidebarRail: {
          width: '4px',
          bgcolor: 'var(--nav-rail-accent)',
          borderRadius: '0 2px 2px 0',
          flexShrink: 0,
        },
        mobileTopBar: {
          bgcolor: 'var(--nav-appbar-bg)',
          borderBottom: 'var(--nav-appbar-border)',
          boxShadow: 'none',
          backdropFilter: 'blur(16px)',
          WebkitBackdropFilter: 'blur(16px)',
        },
        hoverTransition: {
          transition: 'all 150ms cubic-bezier(0.4,0,0.2,1)',
        },
        activeTransition: {
          transition: 'all 200ms cubic-bezier(0.4,0,0.2,1)',
        },
      };
    }

    // Light mode
    return {
      isDark: false,
      appBar: {
        bgcolor: 'var(--nav-appbar-bg)',
        borderBottom: 'var(--nav-appbar-border)',
        boxShadow: 'none',
      },
      appBarAccent: {
        position: 'absolute' as const,
        bottom: 0,
        left: 0,
        right: 0,
        height: '2px',
        background: 'linear-gradient(90deg, transparent 0%, var(--nav-accent) 40%, var(--nav-accent) 60%, transparent 100%)',
        opacity: 0.7,
        pointerEvents: 'none' as const,
      },
      toolbar: {
        flexWrap: 'wrap' as const,
        gap: 1,
        alignItems: 'center',
        background: 'var(--nav-bg)',
      },
      searchInput: {
        bgcolor: 'rgba(0,0,0,0.04)',
        borderRadius: '20px',
        border: '1px solid transparent',
        color: 'var(--nav-text)',
        '& .MuiOutlinedInput-notchedOutline': { border: 'none' },
        '&:hover .MuiOutlinedInput-notchedOutline': { border: '1px solid var(--nav-border-accent)' },
        '&.Mui-focused .MuiOutlinedInput-notchedOutline': { border: '1px solid var(--nav-accent)' },
        fontSize: '0.875rem',
        height: '38px',
        pl: 1.5,
        pr: 1.5,
      },
      categoryBtn: (isActive: boolean) => ({
        textTransform: 'none' as const,
        fontWeight: isActive ? 600 : 400,
        color: isActive ? 'var(--nav-accent)' : 'var(--nav-text)',
        borderBottom: isActive ? '2px solid var(--nav-accent)' : '2px solid transparent',
        bgcolor: isActive ? 'var(--nav-accent-muted)' : 'transparent',
        borderRadius: '6px 6px 0 0',
        px: 1.5,
        py: 0.75,
        transition: 'all 150ms cubic-bezier(0.4,0,0.2,1)',
        '&:hover': { bgcolor: 'var(--nav-hover-fill)', color: 'var(--nav-text)' },
      }),
      categoryBtnDot: (color: string) => ({
        width: 6,
        height: 6,
        borderRadius: '50%',
        bgcolor: color,
        flexShrink: 0,
        boxShadow: `0 0 0 2px ${color}18`,
      }),
      menuPaper: {
        bgcolor: 'var(--nav-menu-bg)',
        border: '1px solid var(--nav-menu-border)',
        boxShadow: 'var(--nav-menu-shadow)',
        borderRadius: 2,
        minWidth: 280,
        maxWidth: '90vw',
        overflow: 'hidden',
      },
      menuItem: (isSelected: boolean) => ({
        display: 'flex',
        alignItems: 'center',
        gap: 1.5,
        py: 1.5,
        px: 2,
        borderLeft: isSelected ? '3px solid var(--nav-accent)' : '3px solid transparent',
        bgcolor: isSelected ? 'var(--nav-item-active)' : 'transparent',
        color: isSelected ? 'var(--nav-accent)' : 'var(--nav-item-text)',
        transition: 'all 150ms cubic-bezier(0.4,0,0.2,1)',
        '&:hover': {
          bgcolor: 'var(--nav-item-hover)',
          color: 'var(--nav-item-text)',
        },
        '&.Mui-disabled': { opacity: 0.5 },
      }),
      menuItemLeftBar: {
        borderLeft: '3px solid var(--nav-accent)',
        bgcolor: 'var(--nav-item-active)',
        color: 'var(--nav-accent)',
      },
      sidebar: {
        bgcolor: 'var(--nav-sidebar-bg)',
        borderRight: 'var(--nav-sidebar-border)',
        boxShadow: 'none',
      },
      sidebarItem: (isActive: boolean) => ({
        display: 'flex',
        alignItems: 'center',
        gap: '0.75rem',
        px: '0.875rem',
        py: '0.75rem',
        mx: 1,
        borderRadius: '8px',
        borderLeft: isActive ? '4px solid var(--nav-rail-accent)' : '4px solid transparent',
        bgcolor: isActive ? 'var(--nav-item-active)' : 'transparent',
        color: isActive ? 'var(--nav-accent)' : 'var(--nav-item-text)',
        transition: 'all 150ms cubic-bezier(0.4,0,0.2,1)',
        '&:hover': {
          bgcolor: isActive ? 'var(--nav-item-active)' : 'var(--nav-item-hover)',
        },
        cursor: 'pointer',
        textDecoration: 'none',
        width: 'calc(100% - 8px)',
        boxSizing: 'border-box',
      }),
      sidebarRail: {
        width: '4px',
        bgcolor: 'var(--nav-rail-accent)',
        borderRadius: '0 2px 2px 0',
        flexShrink: 0,
      },
      mobileTopBar: {
        bgcolor: 'var(--nav-appbar-bg)',
        borderBottom: 'var(--nav-appbar-border)',
        boxShadow: 'none',
        backdropFilter: 'blur(12px)',
        WebkitBackdropFilter: 'blur(12px)',
      },
      hoverTransition: {
        transition: 'all 150ms cubic-bezier(0.4,0,0.2,1)',
      },
      activeTransition: {
        transition: 'all 200ms cubic-bezier(0.4,0,0.2,1)',
      },
    };
  }, [isDark]) as NavStyles;
}
