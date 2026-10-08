import React, { useState, useEffect, useMemo } from 'react';
import { useLocation } from 'react-router-dom';
import { apiFetch } from '../lib/apiClient';
import { stripLocale } from '../i18n/locales';
import BlockableLink from './RouteBlocker/BlockableLink';
import { NotificationBell } from './Notifications/NotificationBell';
import {
  AppBar,
  Toolbar,
  Typography,
  Button,
  Menu,
  MenuItem,
  Box,
  Chip,
  IconButton,
  useTheme,
  Divider,
  ListItemIcon,
  ListItemText,
  Stack as _Stack
} from '@mui/material';
import { ThemeToggleButton } from './ThemeToggleButton';
import {
  Business as BusinessIcon,
  CorporateFare as CorporateFareIcon,
  AccountBalance as PortfolioIcon,
  Category as CategoryIcon,
  Security as SecurityIcon,
  Assessment as AssessmentIcon,
  SystemUpdateAlt as SystemUpdateAltIcon,
  Settings as SettingsIcon,
  Palette as PaletteIcon,
  Notifications as NotificationsIcon,
  QueryStats as QueryStatsIcon,
  Schema as SchemaIcon,
  Policy as PolicyIcon,
  Build as BuildIcon,
  Timeline as TimelineIcon,
  Translate as TranslateIcon,
  EventRepeat as EventRepeatIcon,
  KeyboardArrowDown as KeyboardArrowDownIcon,
  CheckCircle as CheckCircleIcon,
  Api as ApiIcon,
  MenuBook as MenuBookIcon,
  CreateNewFolder as CreateNewFolderIcon,
  Folder as FolderIcon,
  AccountCircle as AccountCircleIcon,
  Logout as LogoutIcon,
  ManageAccounts as ManageAccountsIcon,
  AutoFixHigh as AutoFixHighIcon,
  AutoAwesome as AIIcon,
  Storage as StorageIcon,
  PlayCircleOutline as PlayCircleOutlineIcon,
  SupervisorAccount as SupervisorAccountIcon,
  Code as CodeIcon,
  Speed as SpeedIcon,
  AccountTree as AccountTreeIcon,
  Layers as LayersIcon,
  Extension as ExtensionIcon
} from '@mui/icons-material';
import {
  Lock as LockIcon,
  Warning as WarningIcon,
  Shield as ShieldIcon,
  Group as GroupIcon,
  PersonAdd as PersonAddIcon,
  LockOpen as LockOpenIcon,
  Groups as GroupsIcon,
  MonitorHeart as MonitorHeartIcon
} from '@mui/icons-material';
import { useTenant } from '../contexts/TenantContext';
import { useAccess } from '../contexts/AccessContext';
import { useOrganizationEntitlement } from '../contexts/useOrganizationEntitlement';
import useBlockableNavigate from './RouteBlocker/useBlockableNavigate';
import { useAuth } from '../contexts/AuthContext';
import { MenuCardsPage, MenuCardItem } from './ui/MenuCardsPage';
import { useRouteAliases } from '../hooks/useRouteAliases';
import { useMenuTree } from '../hooks/useMenuTree';
import { mapDesignerTreeToCategories } from './navigationTree';
import { ViewModule as ViewModuleIcon, Menu as MenuListIcon } from '@mui/icons-material';
import ScopeBadge from './ScopeBadge';
import TenantSwitcher from './TenantSwitcher';
import RegionPicker from './RegionPicker';
import TenantTreeView from './TenantTreeView';
import DescriptionIcon from '@mui/icons-material/Description';
import LanguageSelector from './LanguageSelector';
import Dialog from '@mui/material/Dialog';
import DialogTitle from '@mui/material/DialogTitle';
import DialogContent from '@mui/material/DialogContent';
import DialogActions from '@mui/material/DialogActions';
import { Tenant } from '../types';
import { IvyLogo } from './brand/IvyLogo';
import { useCapabilities } from '../hooks/useCapabilities';

export interface NavigationItem {
  label: string;
  path: string;
  icon: React.ReactNode;
  description?: string;
  badge?: {
    label: string;
    color?: 'default' | 'primary' | 'secondary' | 'error' | 'info' | 'success' | 'warning';
  };
  /** Backend ABAC capability required to render this item (e.g. "menu:platform"). */
  requiredCapability?: string;
  /**
   * target_profile_key this item is restricted to, e.g. "PLATFORM_OPERATOR".
   * Resolved from a Menu Designer node's required_entitlement. Distinct from
   * requiredCapability: that is a menu:* action key, this is a profile.
   */
  requiredEntitlement?: string;
}

export interface NavigationMenu {
  label: string;
  icon: React.ReactNode;
  items: NavigationItem[];
  /** Backend ABAC capability required to render this menu group. */
  requiredCapability?: string;
  /** Profile required to render this group; see NavigationItem.requiredEntitlement. */
  requiredEntitlement?: string;
}

export interface CategoryConfig {
  label: string;
  /** The Menu Designer root node key. */
  key: string;
  icon: React.ReactNode;
  defaultPath: string; // Navigate here when category is selected
  color: {
    primary: string;
    light: string;
    dark: string;
    background: string;
  };
  menus: NavigationMenu[];
  /** Backend ABAC capability required to render this top-level category. */
  requiredCapability?: string;
  /** Profile required to render this category; see NavigationItem.requiredEntitlement. */
  requiredEntitlement?: string;
}

// Menu content is not defined here: it is read from the Menu Designer tree (navigation_menu_nodes)
// and mapped by mapDesignerTreeToCategories (see navigationTree.ts).

/**
 * Filter navigation config against the backend capability map.  The frontend
 * never inspects roles directly; it only asks "does the backend allow this
 * capability?".
 *
 * When the backend capability feed hasn't loaded yet (or is missing keys for
 * a freshly-provisioned user), we still want to show admin menus to users we
 * already know are platform operators / global admins — otherwise the entire
 * Organization / Security / System groups disappear and the user can't
 * navigate at all.  Pass `isPlatformOperator=true` to fall back to "show
 * every capability-gated menu" instead of hiding them all.
 *
 * The Organization submenu is additionally gated by the operator_role claim
 * (global_admin / helpdesk / professional_services).  Users with no
 * organization entitlement never see the submenu; users with read-only
 * entitlement see the submenu and the page itself enforces read-only mode.
 */
export function filterNavigationByCapabilities(
  categories: CategoryConfig[],
  capabilities: Record<string, boolean> | undefined,
  isPlatformOperator: boolean = false,
  organizationAccess: { isVisible: boolean; canRead: boolean; canWrite: boolean } = {
    isVisible: false,
    canRead: false,
    canWrite: false,
  },
  // The caller's resolved profile, used to enforce Menu Designer
  // required_entitlement (a target_profile_key). See useResolvedProfile below.
  resolvedProfile: string | undefined = undefined
): CategoryConfig[] {
  // Platform operators bypass the capability gate entirely — they are trusted
  // to see admin navigation.  This is safe because the canAccess() check in
  // the menu-item renderer will still block individual routes they lack
  // scope for, with a clear reason.
  if (isPlatformOperator) {
    return categories;
  }

  // A node gated to a profile is visible only to callers resolved to that
  // profile. BASE_USER is the default (what navigation_menu_handler.go stores
  // when the field is blank), and an empty value means ungated.
  //
  // Fails closed: when the profile is still unknown (undefined) we cannot prove
  // the caller holds it, so restricted nodes stay hidden. An unrecognized
  // value that no IAM row can ever produce also hides the node — a typo in the
  // Menu Designer must never authorize everyone.
  const hasProfile = (entitlement?: string): boolean => {
    if (!entitlement || entitlement === 'BASE_USER') return true;
    if (!resolvedProfile) return false;
    return entitlement === resolvedProfile;
  };

  // Strip the Organization submenu for users who do not have any
  // organization entitlement.  The Platform category itself remains visible
  // because it also contains the Security and System submenus.
  const stripOrganization = (cat: CategoryConfig): CategoryConfig => ({
    ...cat,
    menus: cat.menus.filter((menu) => {
      if (menu.requiredCapability === 'menu:organization' && !organizationAccess.isVisible) {
        return false;
      }
      return true;
    }),
  });

  if (!capabilities) {
    // If entitlements have not loaded yet (or AuthContext hasn't yet exposed
    // them — see the entitlements-property-on-AuthContext TS error), fall
    // back to showing only menus that require no capability.  This prevents a
    // flash of unauthorized UI for non-admin users.
    return categories
      .filter((cat) => !cat.requiredCapability && hasProfile(cat.requiredEntitlement))
      .map(stripOrganization)
      .map((cat) => ({
        ...cat,
        menus: cat.menus
          .filter((menu) => !menu.requiredCapability && hasProfile(menu.requiredEntitlement))
          .map((menu) => ({
            ...menu,
            items: menu.items.filter(
              (item) => !item.requiredCapability && hasProfile(item.requiredEntitlement)
            ),
          }))
          .filter((menu) => menu.items.length > 0),
      }))
      .filter((cat) => cat.menus.length > 0);
  }

  const hasCap = (cap?: string) => (cap ? !!capabilities[cap] : true);

  return categories
    .filter((cat) => hasCap(cat.requiredCapability) && hasProfile(cat.requiredEntitlement))
    .map(stripOrganization)
    .map((cat) => ({
      ...cat,
      menus: cat.menus
        .filter((menu) => hasCap(menu.requiredCapability) && hasProfile(menu.requiredEntitlement))
        .map((menu) => ({
          ...menu,
          items: menu.items.filter(
            (item) => hasCap(item.requiredCapability) && hasProfile(item.requiredEntitlement)
          ),
        }))
        .filter((menu) => menu.items.length > 0),
    }))
    .filter((cat) => cat.menus.length > 0);
}

interface MainNavigationProps {
  // No longer needed - ThemeToggleButton handles theme internally
}

export const MainNavigation: React.FC<MainNavigationProps> = () => {
  const theme = useTheme();
  const location = useLocation();
  const navigate = useBlockableNavigate();
  const { tenant, product, datasource, isSelected, setSelection } = useTenant();
  const { isPlatformOperator, accessLevel, canAccess, scope, scopeDescription } = useAccess();
  // compact scope summary for very small screens
  const scopeSummary = `${tenant?.display_name || tenant?.name || ''}${product ? ` · ${product.alpha_product?.product_name || 'Product'}` : ''}${datasource ? ` · ${datasource.source_name || 'Source'}` : ''}`;
  const { user, logout } = useAuth();
  const organizationAccess = useOrganizationEntitlement();
  const { capabilities, profile: resolvedProfile } = useCapabilities();

  // Capability-filtered navigation config.  The backend decides which menus
  // the user is allowed to see; the frontend only renders the allowed subset.
  // Platform operators (global admins) bypass the capability gate so they can
  // always navigate to admin sections; per-route access is still gated by
  // canAccess() inside the dropdown renderer.
  const { aliases: routeAliases } = useRouteAliases();
  const menuTree = useMenuTree();
  const designerCategories = useMemo(
    () => mapDesignerTreeToCategories(menuTree, routeAliases),
    [menuTree, routeAliases],
  );
  const baseCategoryConfigs = useMemo(
    () =>
      filterNavigationByCapabilities(
        designerCategories,
        capabilities,
        isPlatformOperator,
        organizationAccess,
        resolvedProfile,
      ),
    [designerCategories, capabilities, isPlatformOperator, organizationAccess, resolvedProfile],
  );

  const filteredCategoryConfigs = baseCategoryConfigs;

  const [categoryMenuAnchorEl, setCategoryMenuAnchorEl] = useState<null | HTMLElement>(null);
  // Default to Tenants category on initial load
  const [selectedCategory, setSelectedCategory] = useState<string | null>(null);
  // Nav Mode preference: 'dropdown' or 'cards'
  const [navMode, setNavMode] = useState<'dropdown' | 'cards'>(() => {
    return (localStorage.getItem('app-nav-mode-preference') as 'dropdown' | 'cards') || 'dropdown';
  });
  const [activeCardsMenu, setActiveCardsMenu] = useState<{ title: string; items: MenuCardItem[]; icon?: React.ReactNode } | null>(null);

  const toggleNavMode = () => {
    const next = navMode === 'dropdown' ? 'cards' : 'dropdown';
    setNavMode(next);
    localStorage.setItem('app-nav-mode-preference', next);
  };
  const [menuAnchorEl, setMenuAnchorEl] = useState<null | HTMLElement>(null);
  const [activeMenu, setActiveMenu] = useState<string | null>(null);
  const [settingsAnchorEl, setSettingsAnchorEl] = useState<null | HTMLElement>(null);
  const [tenantSelectorOpen, setTenantSelectorOpen] = useState(false);
  const [tenants, setTenants] = useState<Tenant[]>([]);

  const buttonRefs = React.useRef<Record<string, HTMLButtonElement | null>>({});

  // Get current category config
  const currentCategory = selectedCategory ? filteredCategoryConfigs.find(c => c.key === selectedCategory) : null;

  // Handle category selection from dropdown - navigate to default page
  const handleCategorySelect = (categoryKey: string) => {
    setSelectedCategory(categoryKey);
    setCategoryMenuAnchorEl(null);

    // Navigate to the category's default page
    const category = filteredCategoryConfigs.find(c => c.key === categoryKey);
    if (category?.defaultPath) {
      navigate(category.defaultPath);
    }
  };

  // Keep selectedCategory in sync with the current URL route if a category owns it
  useEffect(() => {
    const currentPath = stripLocale(location.pathname);
    if (!currentPath || currentPath === '/') {
      if (!selectedCategory && filteredCategoryConfigs.length > 0) {
        setSelectedCategory(filteredCategoryConfigs[0].key);
      }
      return;
    }

    // Check if any category contains the current route or prefix
    const matchingCat = filteredCategoryConfigs.find((c) =>
      c.menus.some((m) =>
        m.items.some((i) => {
          if (i.path === currentPath) return true;
          // Sub-routes like /build/cubes/new matching /build/cubes
          if (i.path !== '/' && currentPath.startsWith(i.path)) return true;
          return false;
        })
      )
    );

    if (matchingCat) {
      if (selectedCategory !== matchingCat.key) {
        setSelectedCategory(matchingCat.key);
      }
    } else if (!selectedCategory && filteredCategoryConfigs.length > 0) {
      setSelectedCategory(filteredCategoryConfigs[0].key);
    } else if (
      selectedCategory &&
      filteredCategoryConfigs.length > 0 &&
      !filteredCategoryConfigs.some((c) => c.key === selectedCategory)
    ) {
      const fallback = filteredCategoryConfigs[0];
      setSelectedCategory(fallback.key);
      if (fallback.defaultPath) {
        navigate(fallback.defaultPath);
      }
    }
  }, [location.pathname, filteredCategoryConfigs, selectedCategory, navigate]);

  // Handle opening menu from top nav
  const handleMenuOpen = (menuLabel: string) => {
    if (navMode === 'cards' && currentCategory) {
      const menuGroup = currentCategory.menus.find(m => m.label === menuLabel);
      if (menuGroup) {
        const cardItems: MenuCardItem[] = menuGroup.items.map(item => ({
          id: item.path,
          label: item.label,
          description: item.description,
          icon: item.icon,
          to: item.path,
        }));
        setActiveCardsMenu({
          title: `${currentCategory.label} · ${menuGroup.label}`,
          items: cardItems,
          icon: menuGroup.icon,
        });
        return;
      }
    }

    const el = buttonRefs.current[menuLabel] || null;
    setMenuAnchorEl(el);
    setActiveMenu(menuLabel);
  };

  const handleMenuClose = () => {
    setMenuAnchorEl(null);
    setActiveMenu(null);
  };

  const handleCategoryMenuOpen = (event: React.MouseEvent<HTMLButtonElement>) => {
    setCategoryMenuAnchorEl(event.currentTarget);
  };

  const handleCategoryMenuClose = () => {
    setCategoryMenuAnchorEl(null);
  };

  const handleSettingsOpen = (event: React.MouseEvent<HTMLButtonElement>) => {
    setSettingsAnchorEl(event.currentTarget);
  };

  const handleSettingsClose = () => {
    setSettingsAnchorEl(null);
  };

  useEffect(() => {
    if (!tenantSelectorOpen) return;
    let mounted = true;
    apiFetch('/api/tenants/all')
      .then(r => r.json())
      .then((data: Tenant[]) => { if (mounted) setTenants(data || []); })
      .catch(() => { if (mounted) setTenants([]); });
    return () => { mounted = false; };
  }, [tenantSelectorOpen]);

  const handleTenantSelect = (item: Tenant | any) => {
    // If a TenantInstance was selected, item may be instance; normalize
    let selectedTenant: Tenant | null = null;
    let selectedInstance: any = null;
    if (item && 'tenant_instances' in item) {
      selectedTenant = item as Tenant;
    } else if (item && item.tenant_id) {
      // instance
      selectedInstance = item;
      selectedTenant = tenants.find(t => t.id === item.tenant_id) || null;
    }

    // Try to pick a product and datasource if available on the instance
    if (selectedInstance) {
      const instanceProducts = selectedInstance.tenant_products || selectedInstance.products || [];
      const selectedProduct = instanceProducts[0] || null;
      const selectedDatasource = selectedProduct?.tenant_product_datasources?.[0] || selectedProduct?.datasources?.[0] || null;
      if (selectedTenant && selectedProduct && selectedDatasource) {
        setSelection(selectedTenant, selectedProduct, selectedDatasource);
      }
    }
    setTenantSelectorOpen(false);
  };

  const handleLogout = async () => {
    try {
      await logout();
      handleSettingsClose();
      // allow the route blocker to intercept programmatic navigation
      void navigate('/login', { replace: true });
    } catch (e) {
      // fallback redirect
      window.location.href = '/login';
    }
  };

  const isCurrentPath = (path: string) => {
    return stripLocale(location.pathname) === path;
  };

  return (
    <>
      <AppBar position="static" elevation={1} sx={{
        bgcolor: theme.palette.mode === 'dark' ? 'rgba(20,20,25,0.85)' : undefined,
        backdropFilter: theme.palette.mode === 'dark' ? 'blur(6px)' : undefined,
        borderBottom: theme.palette.mode === 'dark' ? '1px solid rgba(255,255,255,0.08)' : undefined
      }}>
  <Toolbar className="app-top-nav" sx={{ flexWrap: 'wrap', gap: 1, alignItems: 'center' }}>
          {/* Logo/Brand with Category Selector */}
          <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
            <IvyLogo variant="full" size="sm" asLink />
            <Button
              color="inherit"
              endIcon={<KeyboardArrowDownIcon />}
              onClick={handleCategoryMenuOpen}
              sx={{
                textTransform: 'none',
                fontWeight: 'normal',
                ml: 1
              }}
            >
              {currentCategory?.label}
            </Button>
          </Box>

          {/* Category Selection Menu */}
          <Menu
            anchorEl={categoryMenuAnchorEl}
            open={Boolean(categoryMenuAnchorEl)}
            onClose={handleCategoryMenuClose}
          >
            {filteredCategoryConfigs.map((cat) => (
              <MenuItem
                key={cat.key}
                onClick={() => handleCategorySelect(cat.key)}
                selected={selectedCategory === cat.key}
                sx={{
                  backgroundColor: selectedCategory === cat.key ? `${cat.color.light}` : 'transparent',
                  color: selectedCategory === cat.key ? cat.color.primary : 'inherit'
                }}
              >
                {cat.icon}
                <Typography sx={{ ml: 1, display: 'inline-flex', alignItems: 'center' }}>{cat.label}</Typography>
                {/* Highlight Tenants option in the category selector */}
                {cat.key === 'tenants' && (
                  <Chip
                    size="small"
                    label="Setup"
                    sx={{
                      ml: 1,
                      bgcolor: cat.color.light,
                      color: cat.color.primary,
                      fontWeight: 'bold'
                    }}
                  />
                )}
              </MenuItem>
            ))}
          </Menu>



          {/* Category-Specific Menus */}
          {currentCategory && (
            <Box className="category-menus" sx={{ display: 'flex', gap: 1, flexWrap: 'wrap', alignItems: 'center' }}>
              {currentCategory.menus.map((menu) => (
                <Button
                  key={menu.label}
                  color="inherit"
                  ref={(el: HTMLButtonElement | null) => { buttonRefs.current[menu.label] = el; }}
                  onClick={() => handleMenuOpen(menu.label)}
                  endIcon={<KeyboardArrowDownIcon />}
                  sx={{
                    textTransform: 'none',
                    fontWeight: activeMenu === menu.label ? 'bold' : 'normal',
                    backgroundColor: activeMenu === menu.label
                      ? `${currentCategory.color.primary}20`
                      : 'transparent',
                    color: activeMenu === menu.label ? currentCategory.color.primary : 'inherit',
                    borderBottomWidth: activeMenu === menu.label ? '2px' : '0px',
                    borderBottomStyle: 'solid',
                    borderBottomColor: activeMenu === menu.label ? currentCategory.color.primary : 'transparent',
                    // Extra visual emphasis for Tenants category menus
                    ...(currentCategory.key === 'tenants' ? { boxShadow: `0 0 0 3px ${currentCategory.color.background}`, borderRadius: 1 } : {}),
                    '&:hover': {
                      backgroundColor: `${currentCategory.color.primary}10`
                    }
                  }}
                >
                  {menu.icon}
                  <Typography variant="body2" sx={{ ml: 0.5 }}>
                    {menu.label}
                  </Typography>
                </Button>
              ))}
            </Box>
          )}

          {/* Current Selection Display - New TenantSwitcher */}
          <Box sx={{ display: 'flex', alignItems: 'center', gap: 1, mr: { xs: 0, md: 2 } }}>
            {/* Access level indicator for platform operators */}
            {isPlatformOperator && (
              <Chip
                icon={<SupervisorAccountIcon />}
                label="Operator"
                size="small"
                color="warning"
                variant="outlined"
                sx={{ display: { xs: 'none', sm: 'inline-flex' } }}
              />
            )}
            {/* New unified tenant/scope switcher */}
            <TenantSwitcher compact={false} />
            <RegionPicker />
          </Box>

          {/* Spacer */}
          <Box sx={{ flexGrow: 1 }} />

          {/* Scope badge (positioned absolute via CSS) */}
          <div className="scope-badge">
            <ScopeBadge />
          </div>

          {/* Quick Actions */}
          <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
            <LanguageSelector />
            <IconButton
              color="inherit"
              onClick={toggleNavMode}
              aria-label={`Switch to ${navMode === 'dropdown' ? 'Card' : 'Dropdown'} View`}
              title={`Navigation View: ${navMode === 'dropdown' ? 'Dropdown Menus' : 'Cards Grid'} (click to toggle)`}
            >
              {navMode === 'dropdown' ? <ViewModuleIcon /> : <MenuListIcon />}
            </IconButton>
            <ThemeToggleButton showMenu />

            <NotificationBell />

            <IconButton color="inherit" onClick={handleSettingsOpen} aria-label="Settings and account menu">
              <SettingsIcon />
            </IconButton>
          </Box>
        </Toolbar>
      </AppBar>

      {/* Menu Cards Page Overlay when NavMode is 'cards' */}
      {activeCardsMenu && (
        <MenuCardsPage
          title={activeCardsMenu.title}
          items={activeCardsMenu.items}
          icon={activeCardsMenu.icon}
          onClose={() => setActiveCardsMenu(null)}
        />
      )}

      {/* Menu Items Dropdown */}
      <Menu
        anchorEl={menuAnchorEl}
        open={Boolean(menuAnchorEl)}
        onClose={handleMenuClose}
        anchorOrigin={{
          vertical: 'bottom',
          horizontal: 'left',
        }}
        transformOrigin={{
          vertical: 'top',
          horizontal: 'left',
        }}
        PaperProps={{
          sx: {
            minWidth: 280,
            maxWidth: '90vw',
            borderRadius: 1,
            boxShadow: theme.palette.mode === 'dark'
              ? '0 10px 30px rgba(0,0,0,0.4)'
              : '0 10px 30px rgba(15,23,42,0.12)',
          }
        }}
      >
        {currentCategory && activeMenu && (
          // Render menu items as an array (Menu does not accept a Fragment as direct child)
          (currentCategory.menus.find(m => m.label === activeMenu)?.items ?? []).map((item: NavigationItem) => {
            const selected = isCurrentPath(item.path);
            const { allowed, reason } = canAccess(item.path);
            
            // Show disabled items with lock icon if not allowed
            if (!allowed) {
              const isScopeReason = reason?.includes('Please select a');
              return (
                <MenuItem
                  key={item.path}
                  disabled
                  sx={{
                    py: 1.5,
                    px: 2,
                    display: 'flex',
                    alignItems: 'center',
                    gap: 1.5,
                    opacity: 0.7,
                    borderLeft: isScopeReason ? '3px solid transparent' : 'none',
                    '&.Mui-disabled': {
                      color: 'text.secondary',
                    }
                  }}
                >
                  <Box sx={{ color: isScopeReason ? 'warning.main' : 'text.disabled', display: 'flex' }}>
                    {isScopeReason ? <WarningIcon fontSize="small" /> : <LockIcon fontSize="small" />}
                  </Box>
                  <Box sx={{ flex: 1 }}>
                    <Typography variant="body2" color="text.secondary" fontWeight={500}>
                      {item.label}
                    </Typography>
                    <Typography 
                      variant="caption" 
                      color={isScopeReason ? 'warning.main' : 'text.disabled'}
                      sx={{ display: 'block', fontWeight: isScopeReason ? 600 : 400 }}
                    >
                      {reason || 'Access restricted'}
                    </Typography>
                  </Box>
                </MenuItem>
              );
            }
            
            let scopedPath = item.path;
            if (tenant?.id) {
              const iId = scope?.instanceId || datasource?.id;
              if (item.path === '/fabric/tenants') {
                scopedPath = `/tenants/${tenant.id}`;
                if (iId) {
                  scopedPath += `?instanceId=${iId}`;
                }
              } else {
                const qs = new URLSearchParams();
                qs.set('tenantId', tenant.id);
                if (iId) {
                  qs.set('instanceId', iId);
                }
                const hasQuery = item.path.includes('?');
                scopedPath = `${item.path}${hasQuery ? '&' : '?'}${qs.toString()}`;
              }
            }

            return (
              <MenuItem
                key={item.path}
                component={BlockableLink}
                {...{ to: scopedPath }}
                onClick={handleMenuClose}
                selected={selected}
                sx={{
                  py: 1.5,
                  px: 2,
                  display: 'flex',
                  alignItems: 'center',
                  gap: 1.5,
                  backgroundColor: selected 
                    ? `${currentCategory.color.primary}20` 
                    : 'transparent',
                  color: selected ? currentCategory.color.primary : 'inherit',
                  borderLeft: selected ? `3px solid ${currentCategory.color.primary}` : 'none',
                  '&:hover': {
                    backgroundColor: `${currentCategory.color.primary}10`
                  }
                }}
              >
                <Box sx={{ color: currentCategory.color.primary, display: 'flex' }}>
                  {item.icon}
                </Box>
                <Box sx={{ flex: 1 }}>
                  <Typography variant="body2" fontWeight={selected ? 600 : 500}>
                    {item.label}
                  </Typography>
                  {item.description && (
                    <Typography variant="caption" color="text.secondary">
                      {item.description}
                    </Typography>
                  )}
                </Box>
                {item.badge && (
                  <Chip
                    size="small"
                    label={item.badge.label}
                    color={item.badge.color ?? 'default'}
                  />
                )}
              </MenuItem>
            );
          })
        )}
      </Menu>

      {/* Tenant Selector Dialog (opened from combined chip) */}
      <Dialog open={tenantSelectorOpen} onClose={() => setTenantSelectorOpen(false)} maxWidth="sm" fullWidth>
        <DialogTitle>Select Tenant / Instance</DialogTitle>
        <DialogContent dividers>
          <TenantTreeView tenants={tenants} onSelect={handleTenantSelect} onAddInstance={() => {}} onShowProducts={() => {}} />
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setTenantSelectorOpen(false)}>Close</Button>
        </DialogActions>
      </Dialog>

      {/* Settings / Account Menu */}
      <Menu
        anchorEl={settingsAnchorEl}
        open={Boolean(settingsAnchorEl)}
        onClose={handleSettingsClose}
        anchorOrigin={{ vertical: 'bottom', horizontal: 'right' }}
        transformOrigin={{ vertical: 'top', horizontal: 'right' }}
        PaperProps={{ sx: { minWidth: 220, bgcolor: theme.palette.mode === 'dark' ? 'rgba(30,30,35,0.95)' : undefined } }}
      >
        <MenuItem disabled sx={{ opacity: 1, cursor: 'default' }}>
          <ListItemIcon>
            <AccountCircleIcon fontSize="small" />
          </ListItemIcon>
          <ListItemText
            primary={user?.name || 'User'}
            secondary={user?.email || ''}
            primaryTypographyProps={{ variant: 'body2', fontWeight: 'bold' }}
            secondaryTypographyProps={{ variant: 'caption' }}
          />
        </MenuItem>
        <Divider sx={{ my: 0.5 }} />
        {/* Appearance (theme style + mode) is a per-user preference, so it is always offered here. */}
        <MenuItem onClick={() => { handleSettingsClose(); void navigate('/fabric/settings'); }} data-testid="appearance-item">
          <ListItemIcon>
            <PaletteIcon fontSize="small" />
          </ListItemIcon>
          <ListItemText
            primary="Appearance"
            secondary="Theme style and mode"
            primaryTypographyProps={{ variant: 'body2', fontWeight: 'bold' }}
            secondaryTypographyProps={{ variant: 'caption' }}
          />
        </MenuItem>
        {/* IP Whitelist menu item above Sign Out */}
  <MenuItem onClick={() => { handleSettingsClose(); void navigate('/fabric/ip-whitelist'); }} data-testid="ip-whitelist-item">
          <ListItemIcon>
            <SecurityIcon fontSize="small" />
          </ListItemIcon>
          <ListItemText
            primary="IP Whitelist"
            primaryTypographyProps={{ variant: 'body2', fontWeight: 'bold', color: 'warning.main' }}
          />
        </MenuItem>
        <Divider sx={{ my: 0.5 }} />
        <MenuItem onClick={handleLogout}>
          <ListItemIcon>
            <LogoutIcon fontSize="small" />
          </ListItemIcon>
            <ListItemText
              primary="Sign Out"
              primaryTypographyProps={{ variant: 'body2' }}
            />
        </MenuItem>
      </Menu>
    </>
  );
};
