import React from 'react';
import { Box, Typography, IconButton, Tooltip } from '@mui/material';
import { Hub as HubIcon } from '@mui/icons-material';
import { useLocation, useNavigate } from 'react-router-dom';
import { stripLocale } from '../../i18n/locales';

interface NavItem {
  key: string;
  label: string;
  path: string;
}

const NAV_ITEMS: NavItem[] = [
  { key: 'query-builder',     label: 'Query Builder',     path: '/query-builder' },
  { key: 'semantic-catalog',  label: 'Semantic Catalog',  path: '/semantic-catalog' },
  { key: 'sql-studio',        label: 'SQL Studio',        path: '/sql-studio' },
  { key: 'pipelines',         label: 'Pipelines',         path: '/pipelines' },
  { key: 'reports',           label: 'Reports',           path: '/reports' },
];

const STORAGE_KEY = 'analytical.workspace-rail-collapsed.v1';

function readPersisted(): boolean {
  try { return localStorage.getItem(STORAGE_KEY) === 'true'; } catch { return false; }
}
function persist(value: boolean) {
  try { localStorage.setItem(STORAGE_KEY, String(value)); } catch { /* ignore */ }
}

interface AnalyticalShellProps {
  children: React.ReactNode;
  collapsed?: boolean;
  onToggleCollapsed?: () => void;
}

export const AnalyticalShell: React.FC<AnalyticalShellProps> = ({ children, collapsed: collapsedProp, onToggleCollapsed }) => {
  const [internalCollapsed, setInternalCollapsed] = React.useState<boolean>(() => readPersisted());
  const collapsed = collapsedProp ?? internalCollapsed;
  const toggle = () => {
    const next = !collapsed;
    if (onToggleCollapsed) onToggleCollapsed();
    else { persist(next); setInternalCollapsed(next); }
  };

  const location = useLocation();
  const navigate = useNavigate();
  const stripped = stripLocale(location.pathname);
  const activeKey = NAV_ITEMS.find((i) => stripped.startsWith(i.path))?.key ?? 'query-builder';

  return (
    <Box sx={{ display: 'flex', flex: 1, minHeight: 0, minWidth: 0 }}>
      <Box
        component="aside"
        sx={{
          width: collapsed ? 60 : 260,
          flexShrink: 0,
          bgcolor: 'var(--mui-bg-lowest)',
          borderRight: '1px solid var(--mui-border)',
          display: 'flex',
          flexDirection: 'column',
          justifyContent: 'space-between',
          transition: 'width 240ms cubic-bezier(0.4, 0, 0.2, 1)',
          overflow: 'hidden',
          height: '100%',
        }}
      >
        <Box>
          <Box
            sx={{
              height: 52,
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'space-between',
              px: 1.5,
              borderBottom: '1px solid var(--mui-border)',
              bgcolor: 'var(--mui-bg-paper)',
              gap: 1,
            }}
          >
            <Box sx={{ display: 'flex', alignItems: 'center', gap: 1.25, overflow: 'hidden' }}>
              <Box
                sx={{
                  width: 28,
                  height: 28,
                  borderRadius: '4px',
                  bgcolor: 'var(--mui-primary-main)',
                  display: 'flex',
                  alignItems: 'center',
                  justifyContent: 'center',
                  flexShrink: 0,
                }}
              >
                <HubIcon sx={{ fontSize: 18, color: '#ffffff' }} />
              </Box>
              {!collapsed && (
                <Box sx={{ display: 'flex', flexDirection: 'column', whiteSpace: 'nowrap' }}>
                  <Typography sx={{ fontWeight: 700, fontSize: '0.9375rem', letterSpacing: '-0.02em', lineHeight: 1, color: 'var(--mui-text-primary)' }}>
                    Ivy
                  </Typography>
                  <Typography
                    className="font-mono"
                    sx={{
                      fontSize: '0.65rem',
                      color: 'var(--mui-primary-light)',
                      textTransform: 'uppercase',
                      letterSpacing: '0.08em',
                      mt: '2px !important',
                    }}
                  >
                    Workbench
                  </Typography>
                </Box>
              )}
            </Box>
            <Tooltip title={collapsed ? 'Expand Workspace Rail' : 'Collapse Workspace Rail'} placement="right">
              <IconButton
                size="small"
                onClick={toggle}
                sx={{
                  p: 0.5,
                  color: 'var(--mui-text-secondary)',
                  '&:hover': { color: 'var(--mui-text-primary)', bgcolor: 'rgba(255,255,255,0.04)' },
                }}
              >
                <Box component="span" sx={{ display: 'flex', alignItems: 'center', transform: collapsed ? 'none' : 'rotate(180deg)', transition: 'transform 240ms' }}>
                  <svg width="18" height="18" viewBox="0 0 24 24" fill="currentColor">
                    <path d="M3 5h18v2H3zm0 12h18v2H3zM3 11h18v2H3z" />
                  </svg>
                </Box>
              </IconButton>
            </Tooltip>
          </Box>

          {!collapsed && (
            <Box sx={{ px: 1.5, py: 1.5 }}>
              <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', px: 1, mb: 0.5 }}>
                <Typography
                  sx={{
                    fontFamily: 'var(--font-data-label)',
                    fontSize: '0.6875rem',
                    fontWeight: 600,
                    textTransform: 'uppercase',
                    letterSpacing: '0.08em',
                    color: 'var(--mui-text-secondary)',
                  }}
                >
                  Analytical Modules
                </Typography>
                <Typography
                  className="font-mono"
                  sx={{
                    fontSize: '0.625rem',
                    color: 'var(--mui-text-secondary)',
                  }}
                >
                  v4.12
                </Typography>
              </Box>
              <Box sx={{ display: 'flex', flexDirection: 'column', gap: 0.5, mt: 1 }}>
                {NAV_ITEMS.map((item) => {
                  const isActive = item.key === activeKey;
                  const button = (
                    <Box
                      key={item.key}
                      role="button"
                      tabIndex={0}
                      onClick={() => navigate(item.path)}
                      onKeyDown={(e) => { if (e.key === 'Enter' || e.key === ' ') navigate(item.path); }}
                      sx={{
                        display: 'flex',
                        alignItems: 'center',
                        px: 1.5,
                        py: 0.75,
                        borderRadius: '4px',
                        cursor: 'pointer',
                        color: isActive ? 'var(--mui-primary-light)' : 'var(--mui-text-secondary)',
                        bgcolor: isActive ? 'var(--mui-bg-subtle)' : 'transparent',
                        fontWeight: isActive ? 600 : 400,
                        fontSize: '0.875rem',
                        transition: 'background-color 0.15s, color 0.15s',
                        userSelect: 'none',
                        '&:hover': { bgcolor: 'var(--mui-bg-subtle)', color: 'var(--mui-text-primary)' },
                      }}
                    >
                      {item.label}
                    </Box>
                  );
                  return collapsed ? (
                    <Tooltip key={item.key} title={item.label} placement="right">
                      {button}
                    </Tooltip>
                  ) : button;
                })}
              </Box>
            </Box>
          )}
        </Box>

        <Box
          sx={{
            p: 1,
            borderTop: '1px solid var(--mui-border)',
            bgcolor: 'var(--mui-bg-lowest)',
            display: 'flex',
            flexDirection: 'column',
            gap: 1,
          }}
        >
          {!collapsed ? (
            <>
              <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', bgcolor: 'var(--mui-bg-paper)', p: '6px 8px', borderRadius: '4px' }}>
                <Box sx={{ display: 'flex', alignItems: 'center', gap: 0.75 }}>
                  <Box sx={{ width: 7, height: 7, borderRadius: '50%', bgcolor: 'var(--mui-primary-light)', animation: 'pulse 2s infinite' }} />
                  <Typography className="font-mono" sx={{ fontSize: '0.6875rem', color: 'var(--mui-text-secondary)', whiteSpace: 'nowrap' }}>
                    Engine v4.12.0
                  </Typography>
                </Box>
                <Typography className="font-mono" sx={{ fontSize: '0.6875rem', color: 'var(--mui-secondary-main)', fontWeight: 600 }}>
                  SYNCED
                </Typography>
              </Box>
              <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', px: 0.5 }}>
                <Typography sx={{ fontFamily: 'var(--font-data-label)', fontSize: '0.6875rem', fontWeight: 600, textTransform: 'uppercase', letterSpacing: '0.08em', color: 'var(--mui-text-secondary)' }}>
                  Region
                </Typography>
                <Typography className="font-mono" sx={{ color: 'var(--mui-text-primary)', fontSize: '0.6875rem' }}>
                  eu-west-1 (prod)
                </Typography>
              </Box>
            </>
          ) : (
            <Tooltip title="Engine v4.12.0 — SYNCED" placement="right">
              <Box sx={{ display: 'flex', justifyContent: 'center' }}>
                <Box sx={{ width: 7, height: 7, borderRadius: '50%', bgcolor: 'var(--mui-primary-light)' }} />
              </Box>
            </Tooltip>
          )}
        </Box>
      </Box>
      <Box sx={{ flex: 1, minWidth: 0, minHeight: 0, display: 'flex', flexDirection: 'column' }}>
        {children}
      </Box>
    </Box>
  );
};

export default AnalyticalShell;
