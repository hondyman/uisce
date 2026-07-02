import React, { useState, useEffect } from 'react';
import { useLocation } from 'react-router-dom';
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
  AccountBalance as PortfolioIcon,
  Category as CategoryIcon,
  Security as SecurityIcon,
  Assessment as AssessmentIcon,
  SystemUpdateAlt as SystemUpdateAltIcon,
  Settings as SettingsIcon,
  Notifications as NotificationsIcon,
  QueryStats as QueryStatsIcon,
  Schema as SchemaIcon,
  Policy as PolicyIcon,
  Build as BuildIcon,
  Timeline as TimelineIcon,
  KeyboardArrowDown as KeyboardArrowDownIcon,
  CheckCircle as CheckCircleIcon,
  Api as ApiIcon,
  AccountCircle as AccountCircleIcon,
  Logout as LogoutIcon,
  ManageAccounts as ManageAccountsIcon,
  AutoFixHigh as AutoFixHighIcon,
  AutoAwesome as AIIcon,
  Store as StoreIcon,
  Storage as StorageIcon,
  PlayCircleOutline as PlayCircleOutlineIcon,
  SupervisorAccount as SupervisorAccountIcon,
  Code as CodeIcon,
  Speed as SpeedIcon,
  AccountTree as AccountTreeIcon,
  Layers as LayersIcon,
  Event as EventIcon
} from '@mui/icons-material';
import {
  Lock as LockIcon,
  Warning as WarningIcon,
  Shield as ShieldIcon,
  Group as GroupIcon,
  PersonAdd as PersonAddIcon,
  LockOpen as LockOpenIcon,
  Groups as GroupsIcon
} from '@mui/icons-material';
import { useTenant } from '../contexts/TenantContext';
import { useAccess } from '../contexts/AccessContext';
import useBlockableNavigate from './RouteBlocker/useBlockableNavigate';
import { useAuth } from '../contexts/AuthContext';
import ScopeBadge from './ScopeBadge';
import TenantSwitcher from './TenantSwitcher';
import TenantTreeView from './TenantTreeView';
import LanguageSelector from './LanguageSelector';
import Dialog from '@mui/material/Dialog';
import DialogTitle from '@mui/material/DialogTitle';
import DialogContent from '@mui/material/DialogContent';
import DialogActions from '@mui/material/DialogActions';
import { Tenant } from '../types';

interface NavigationItem {
  label: string;
  path: string;
  icon: React.ReactNode;
  description?: string;
  badge?: {
    label: string;
    color?: 'default' | 'primary' | 'secondary' | 'error' | 'info' | 'success' | 'warning';
  };
}

interface NavigationMenu {
  label: string;
  icon: React.ReactNode;
  items: NavigationItem[];
}

interface CategoryConfig {
  label: string;
  key: 'tenants' | 'catalog' | 'weave' | 'studio' | 'workflow' | 'intelligence' | 'entity' | 'calendar';
  icon: React.ReactNode;
  defaultPath: string; // Navigate here when category is selected
  color: {
    primary: string;
    light: string;
    dark: string;
    background: string;
  };
  menus: NavigationMenu[];
}

const categoryConfigs: CategoryConfig[] = [
  // ═══════════════════════════════════════════════════════════════════════════
  {
    label: 'Platform',
    key: 'tenants',
    icon: <SettingsIcon />,
    defaultPath: '/fabric/tenants',
    color: {
      primary: '#607D8B',
      light: '#ECEFF1',
      dark: '#455A64',
      background: 'rgba(96, 125, 139, 0.06)'
    },
    menus: [
      {
        label: 'Organization',
        icon: <BusinessIcon />,
        items: [
          { label: 'Tenants', path: '/fabric/tenants', icon: <BusinessIcon />, description: 'Manage platform tenants' },
          { label: 'Users & Roles', path: '/admin/rbac/users', icon: <PersonAddIcon />, description: 'Manage users and role assignments' },
          { label: 'Teams', path: '/admin/rbac/teams', icon: <GroupsIcon />, description: 'Team structure' },
          { label: 'User Tenants', path: '/admin/rbac/user-tenants', icon: <SupervisorAccountIcon />, description: 'User tenant assignments' },
        ]
      },
      {
        label: 'Security',
        icon: <SecurityIcon />,
        items: [
          { label: 'Access Rules', path: '/security/access-rules', icon: <LockIcon />, description: 'Row & column security' },
          { label: 'IDP Mappings', path: '/security/idp-mappings', icon: <SecurityIcon />, description: 'Identity Provider mappings' },
          { label: 'Roles & Permissions', path: '/admin/rbac/roles', icon: <ShieldIcon />, description: 'RBAC management' },
          { label: 'Delegations', path: '/admin/rbac/delegations', icon: <SupervisorAccountIcon />, description: 'Approval delegations' },
          { label: 'Field Permissions', path: '/admin/rbac/field-permissions', icon: <LockOpenIcon />, description: 'Field-level security' },
          { label: 'IP Whitelist', path: '/fabric/ip-whitelist', icon: <SecurityIcon />, description: 'Network access rules' },
          { label: 'Secrets', path: '/secrets/config', icon: <LockIcon />, description: 'Secrets management' },
          { label: 'Secrets Audit', path: '/secrets/audit', icon: <ShieldIcon />, description: 'Audit secret changes' },
          { label: 'Secrets Monitoring', path: '/secrets/monitoring', icon: <AssessmentIcon />, description: 'Monitor secrets usage' },
          { label: 'JIT Requests', path: '/jit-request', icon: <LockOpenIcon />, description: 'Just-in-time access' },
          { label: 'Access Explanation', path: '/access-explanation', icon: <SecurityIcon />, description: 'Access debug explain' },
        ]
      },
      {
        label: 'System',
        icon: <SystemUpdateAltIcon />,
        items: [
          { label: 'Audit History', path: '/setup/audit', icon: <TimelineIcon />, description: 'Bitemporal audit trail', badge: { label: 'New', color: 'success' } },
          { label: 'Audit Plane', path: '/audit', icon: <TimelineIcon />, description: 'Immutable audit & snapshots', badge: { label: 'New', color: 'success' } },
          { label: 'Audit Explorer', path: '/core/audit-explorer', icon: <TimelineIcon />, description: 'Explore audit records' },
          { label: 'Fabric Audit', path: '/fabric/audit-logs', icon: <TimelineIcon />, description: 'Platform audit logs' },
          { label: 'Fabric Settings', path: '/fabric/settings', icon: <SettingsIcon />, description: 'Platform settings' },
          { label: 'LLM Config', path: '/admin/llm', icon: <AutoFixHighIcon />, description: 'AI model configuration' },
          { label: 'Seeding', path: '/admin/seeding', icon: <SystemUpdateAltIcon />, description: 'Rule seeding' },
          { label: 'Temporal Ops', path: '/admin/temporal-ops', icon: <PlayCircleOutlineIcon />, description: 'Workflow engine' },
        ]
      }
    ]
  },

  // ═══════════════════════════════════════════════════════════════════════════
  // CATALOG - Data discovery, glossary, and lineage
  // ═══════════════════════════════════════════════════════════════════════════
  {
    label: 'Catalog',
    key: 'catalog',
    icon: <CategoryIcon />,
    defaultPath: '/core/glossary',
    color: {
      primary: '#2196F3',
      light: '#E3F2FD',
      dark: '#1976D2',
      background: 'rgba(33, 150, 243, 0.08)'
    },
    menus: [
      {
        label: 'Glossary',
        icon: <CategoryIcon />,
        items: [
          { label: 'Business Glossary', path: '/core/glossary', icon: <CategoryIcon />, description: 'Terms & definitions' },
          { label: 'Abbreviations', path: '/core/abbreviations', icon: <CategoryIcon />, description: 'Standard abbreviations' },
          { label: 'Data Domains', path: '/core/domains', icon: <CategoryIcon />, description: 'Domain ownership' },
        ]
      },
      {
        label: 'Discovery',
        icon: <StorageIcon />,
        items: [
          { label: 'Schema Explorer', path: '/schema-explorer', icon: <StorageIcon />, description: 'Database tables & columns' },
          { label: 'Metadata Explorer', path: '/metadata-explorer', icon: <LayersIcon />, description: 'Unified BOs & Views' },
          { label: 'Node Types', path: '/catalog/node-types', icon: <SchemaIcon />, description: 'Metadata structures' },
          { label: 'Edge Types', path: '/catalog/edge-types', icon: <AccountTreeIcon />, description: 'Relationship types' },
          { label: 'Catalog Setup', path: '/core/catalog-setup', icon: <SettingsIcon />, description: 'Configure catalog' },
          { label: 'Catalog Test', path: '/core/catalog-setup-test', icon: <SettingsIcon />, description: 'Test catalog setup' },
          { label: 'AI Term Suggestions', path: '/catalog/ai-suggestions', icon: <AutoFixHighIcon />, description: 'Suggested terms' },
          { label: 'Bundle Explorer', path: '/bundle-explorer', icon: <SchemaIcon />, description: 'Explore semantic bundles' },
          { label: 'BO Explorer', path: '/core/business-objects', icon: <BusinessIcon />, description: 'Business Objects setup' },
        ]
      },
      {
        label: 'Lineage',
        icon: <TimelineIcon />,
        items: [
          { label: 'Lineage Explorer', path: '/lineage', icon: <TimelineIcon />, description: 'Data flow visualization', badge: { label: 'New', color: 'success' } },
          { label: 'Semantic Mapper', path: '/core/semantic-mapper', icon: <AutoFixHighIcon />, description: 'Column-to-term mapping' },
          { label: 'Impact Analysis', path: '/lineage/impact', icon: <WarningIcon />, description: 'Change impact' },
        ]
      }
    ]
  },

  // ═══════════════════════════════════════════════════════════════════════════
  // BUILD - Semantic layer development
  // ═══════════════════════════════════════════════════════════════════════════
  {
    label: 'Build',
    key: 'weave',
    icon: <BuildIcon />,
    defaultPath: '/business-objects',
    color: {
      primary: '#9C27B0',
      light: '#F3E5F5',
      dark: '#7B1FA2',
      background: 'rgba(156, 39, 176, 0.08)'
    },
    menus: [
      {
        label: 'Models',
        icon: <BuildIcon />,
        items: [
          { label: 'Business Objects', path: '/business-objects', icon: <BusinessIcon />, description: 'Core entities' },
          { label: 'Views Catalog', path: '/views', icon: <AssessmentIcon />, description: 'Semantic views' },
          { label: 'Bundles', path: '/fabric/bundles', icon: <CategoryIcon />, description: 'Curated bundles', badge: { label: 'AI', color: 'info' } },
        ]
      },
      {
        label: 'Rules',
        icon: <CheckCircleIcon />,
        items: [
          { label: 'Validation Rules', path: '/core/validation-rules', icon: <CheckCircleIcon />, description: 'Data validations' },
          { label: 'Calculated Fields', path: '/core/calculated-fields', icon: <QueryStatsIcon />, description: 'Field calculations' },
          { label: 'Expressions', path: '/reports/expressions', icon: <CodeIcon />, description: 'Starlark expressions' },
          { label: 'Calculations Library', path: '/fabric/calculations', icon: <QueryStatsIcon />, description: 'Core calculation logic' },
        ]
      },
      {
        label: 'Quality',
        icon: <CheckCircleIcon />,
        items: [
          { label: 'Flow Builder', path: '/core/flow-builder', icon: <TimelineIcon />, description: 'Visual pipeline builder', badge: { label: 'New', color: 'success' } },
          { label: 'Uisce Builder', path: '/core/uisce-builder', icon: <BuildIcon />, description: 'Advanced UI builder' },
          { label: 'Run Validations', path: '/core/validation', icon: <CheckCircleIcon />, description: 'Execute validations' },
          { label: 'Marketplace', path: '/marketplace', icon: <StoreIcon />, description: 'Components library' },
          { label: 'UI Components', path: '/marketplace/components', icon: <CodeIcon />, description: 'Component marketplace' },
        ]
      }
    ]
  },

  // ═══════════════════════════════════════════════════════════════════════════
  // STUDIO - Low-code development tools (NEW)
  // ═══════════════════════════════════════════════════════════════════════════
  {
    label: 'Studio',
    key: 'studio' as any,
    icon: <CodeIcon />,
    defaultPath: '/api-studio',
    color: {
      primary: '#E91E63',
      light: '#FCE4EC',
      dark: '#C2185B',
      background: 'rgba(233, 30, 99, 0.08)'
    },
    menus: [
      {
        label: 'API Studio',
        icon: <ApiIcon />,
        items: [
          { label: 'API Designer', path: '/api-studio', icon: <ApiIcon />, description: 'Visual API builder', badge: { label: 'New', color: 'success' } },
          { label: 'API Catalog', path: '/api-catalog', icon: <ApiIcon />, description: 'Published APIs' },
        ]
      },
      {
        label: 'Page Studio',
        icon: <BuildIcon />,
        items: [
          { label: 'Page Designer', path: '/page-studio', icon: <BuildIcon />, description: 'Visual UI builder', badge: { label: 'New', color: 'success' } },
          { label: 'Dynamic UI', path: '/dynamic-ui', icon: <BuildIcon />, description: 'Form generator' },
          { label: 'Custom Components', path: '/fabric/custom-components', icon: <BuildIcon />, description: 'Reusable components' },
        ]
      },
      {
        label: 'Workflow Studio',
        icon: <AccountTreeIcon />,
        items: [
          { label: 'Process Designer', path: '/client-portal/workflow-studio', icon: <AccountTreeIcon />, description: 'Workflow builder' },
          { label: 'Business Rules', path: '/client-portal/rules-editor', icon: <PolicyIcon />, description: 'Rule editor' },
          { label: 'Workflow Designer', path: '/core/workflow-designer', icon: <TimelineIcon />, description: 'Legacy designer' },
        ]
      }
    ]
  },

  // ═══════════════════════════════════════════════════════════════════════════
  // OPERATIONS - Scheduling, orchestration, and process management
  // ═══════════════════════════════════════════════════════════════════════════
  {
    label: 'Operations',
    key: 'workflow',
    icon: <PlayCircleOutlineIcon />,
    defaultPath: '/scheduler-intelligence',
    color: {
      primary: '#00695C',
      light: '#E0F2F1',
      dark: '#004D40',
      background: 'rgba(0, 105, 92, 0.06)'
    },
    menus: [
      {
        label: 'Scheduler',
        icon: <TimelineIcon />,
        items: [
          { label: 'Intelligence Console', path: '/scheduler-intelligence', icon: <AutoFixHighIcon />, description: 'AI-powered scheduler', badge: { label: 'AI', color: 'info' } },
          { label: 'Jobs', path: '/scheduler/jobs', icon: <TimelineIcon />, description: 'Job definitions' },
          { label: 'Executions', path: '/scheduler/executions', icon: <PlayCircleOutlineIcon />, description: 'Run history' },
          { label: 'Calendars', path: '/scheduler/calendars', icon: <CategoryIcon />, description: 'Business calendars' },
        ]
      },
      {
        label: 'Workflows',
        icon: <AccountTreeIcon />,
        items: [
          { label: 'BP Console', path: '/bp-console', icon: <SpeedIcon />, description: 'Orchestration monitor', badge: { label: 'Live', color: 'success' } },
          { label: 'Instance Explorer', path: '/bp-console/instances', icon: <TimelineIcon />, description: 'Debug workflows' },
          { label: 'Work Queues', path: '/bp-console/queues', icon: <AssessmentIcon />, description: 'Queue management' },
          { label: 'Process Catalog', path: '/core/process-catalog', icon: <SchemaIcon />, description: 'Process definitions' },
        ]
      },
      {
        label: 'Governance',
        icon: <PolicyIcon />,
        items: [
          { label: 'ChangeSets', path: '/governance/changesets', icon: <PolicyIcon />, description: 'Change management', badge: { label: 'New', color: 'success' } },
          { label: 'Compliance', path: '/governance/compliance', icon: <SecurityIcon />, description: 'Risk dashboard', badge: { label: 'New', color: 'primary' } },
          { label: 'Approvals', path: '/core/approval-workflows', icon: <CheckCircleIcon />, description: 'Approval workflows' },
          { label: 'Approval Inbox', path: '/core/approval-inbox', icon: <CheckCircleIcon />, description: 'Inbox for approvals' },
          { label: 'SLA Dashboard', path: '/core/sla-dashboard', icon: <TimelineIcon />, description: 'Monitor SLA logic' },
          { label: 'Notifications', path: '/core/notifications', icon: <NotificationsIcon />, description: 'Notification center' },
          { label: 'Notification Templates', path: '/core/notifications/templates', icon: <NotificationsIcon />, description: 'Message templates' },
          { label: 'Notification Prefs', path: '/core/notifications/preferences', icon: <SettingsIcon />, description: 'User preferences' },
        ]
      }
    ]
  },

  // ═══════════════════════════════════════════════════════════════════════════
  // INTELLIGENCE - AI, optimization, and observability (NEW)
  // ═══════════════════════════════════════════════════════════════════════════
  {
    label: 'Intelligence',
    key: 'intelligence' as any,
    icon: <AutoFixHighIcon />,
    defaultPath: '/intelligence',
    color: {
      primary: '#673AB7',
      light: '#EDE7F6',
      dark: '#512DA8',
      background: 'rgba(103, 58, 183, 0.08)'
    },
    menus: [
      {
        label: 'Optimization',
        icon: <SpeedIcon />,
        items: [
          { label: 'Dashboard', path: '/intelligence', icon: <SpeedIcon />, description: 'AI optimization hub', badge: { label: 'New', color: 'success' } },
          { label: 'ASO Center', path: '/optimization', icon: <SpeedIcon />, description: 'Query optimization' },
          { label: 'Index Advisor', path: '/intelligence/index-advisor', icon: <StorageIcon />, description: 'AI index recommendations', badge: { label: 'AI', color: 'info' } },
          { label: 'Storage Tiering', path: '/intelligence/storage', icon: <LayersIcon />, description: 'Intelligent tiering' },
          { label: 'Preaggregations', path: '/fabric/preaggregations', icon: <SpeedIcon />, description: 'Fabric preaggregations' },
        ]
      },
      {
        label: 'Observability',
        icon: <QueryStatsIcon />,
        items: [
          { label: 'Metrics Dashboard', path: '/observability', icon: <QueryStatsIcon />, description: 'Platform metrics' },
          { label: 'SLO Dashboard', path: '/observability/slos', icon: <AssessmentIcon />, description: 'SLO tracking' },
          { label: 'Data Quality', path: '/intelligence/data-quality', icon: <CheckCircleIcon />, description: 'Quality monitoring', badge: { label: 'New', color: 'success' } },
        ]
      },
      {
        label: 'AI Copilot',
        icon: <AutoFixHighIcon />,
        items: [
          { label: 'Natural Language', path: '/nlq', icon: <AutoFixHighIcon />, description: 'Ask questions', badge: { label: 'AI', color: 'info' } },
          { label: 'Global Intelligence', path: '/global-intelligence', icon: <AIIcon />, description: 'Cross-platform AI assistant', badge: { label: 'New', color: 'success' } },
          { label: 'Scenario Analysis', path: '/analytics/scenario-analysis', icon: <TimelineIcon />, description: 'What-if scenarios' },
          { label: 'Portfolio Rebalancer', path: '/analytics/rebalancer', icon: <AutoFixHighIcon />, description: 'AI rebalancing' },
          { label: 'Simulation Workspace', path: '/simulation', icon: <TimelineIcon />, description: 'Run simulations', badge: { label: 'New', color: 'info' } },
          { label: 'Rebalancing Wizard', path: '/simulation/rebalance', icon: <AutoFixHighIcon />, description: 'AI guided rebalancing' },
          { label: 'Scenario Comparison', path: '/simulation/compare', icon: <AssessmentIcon />, description: 'Compare scenario outcomes' },
        ]
      }
    ]
  },

  // ═══════════════════════════════════════════════════════════════════════════
  // CONSUME - Reporting, analytics, and dashboards
  // ═══════════════════════════════════════════════════════════════════════════
  {
    label: 'Consume',
    key: 'entity',
    icon: <AssessmentIcon />,
    defaultPath: '/reports/library',
    color: {
      primary: '#FF9800',
      light: '#FFF3E0',
      dark: '#F57C00',
      background: 'rgba(255, 152, 0, 0.08)'
    },
    menus: [
      {
        label: 'Reports',
        icon: <AssessmentIcon />,
        items: [
          { label: 'Report Library', path: '/reports/library', icon: <AssessmentIcon />, description: 'Saved reports' },
          { label: 'Report Builder', path: '/reports/builder', icon: <BuildIcon />, description: 'Create reports' },
          { label: 'Data Explorer', path: '/reports/queries', icon: <StorageIcon />, description: 'Query builder', badge: { label: 'New', color: 'success' } },
          { label: 'Semantic Models', path: '/reports/models', icon: <CategoryIcon />, description: 'Data models' },
        ]
      },
      {
        label: 'Analytics',
        icon: <TimelineIcon />,
        items: [
          { label: 'Factor Analytics', path: '/analytics/factors', icon: <AssessmentIcon />, description: 'Factor exposure' },
          { label: 'Fixed Income', path: '/fixed-income', icon: <TimelineIcon />, description: 'Bond analytics' },
          { label: 'Private Markets', path: '/private-markets', icon: <BusinessIcon />, description: 'PE/VC analytics' },
        ]
      },
      {
        label: 'Dashboards',
        icon: <AssessmentIcon />,
        items: [
          { label: 'Advisor Dashboard', path: '/analytics/advisor-dashboard', icon: <SupervisorAccountIcon />, description: 'Advisor view' },
          { label: 'Portfolio Master', path: '/analytics/portfolio-master', icon: <PortfolioIcon />, description: 'Gold copy & performance' },
          { label: 'Security Master', path: '/analytics/security-master', icon: <AssessmentIcon />, description: 'Instrument MDM & lineage' },
          { label: 'Crypto Portfolio', path: '/crypto/portfolio', icon: <TimelineIcon />, description: 'Digital assets' },
          { label: 'Wealth Feed', path: '/wealth/feed', icon: <NotificationsIcon />, description: 'Activity feed' },
          { label: 'Fabric Dashboard', path: '/fabric/dashboard', icon: <AssessmentIcon />, description: 'General dashboard view' },
        ]
      }
    ]
  },

  // ═══════════════════════════════════════════════════════════════════════════
  // CALENDAR - Calendar synchronization and management
  // ═══════════════════════════════════════════════════════════════════════════
  {
    label: 'Calendar',
    key: 'calendar',
    icon: <EventIcon />,
    defaultPath: '/calendar',
    color: {
      primary: '#009688',
      light: '#E0F2F1',
      dark: '#00796B',
      background: 'rgba(0, 150, 136, 0.08)'
    },
    menus: [
      {
        label: 'Management',
        icon: <EventIcon />,
        items: [
          { label: 'Calendar Dashboard', path: '/calendar', icon: <EventIcon />, description: 'View events and sync status' },
          { label: 'Sync Conflicts', path: '/calendar/conflicts', icon: <WarningIcon />, description: 'Resolve synchronization conflicts' },
        ]
      }
    ]
  }
];

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

  const [categoryMenuAnchorEl, setCategoryMenuAnchorEl] = useState<null | HTMLElement>(null);
  // Default to Tenants category on initial load
  const [selectedCategory, setSelectedCategory] = useState<'tenants' | 'catalog' | 'weave' | 'studio' | 'workflow' | 'intelligence' | 'entity' | 'calendar' | null>('tenants');
  const [menuAnchorEl, setMenuAnchorEl] = useState<null | HTMLElement>(null);
  const [activeMenu, setActiveMenu] = useState<string | null>(null);
  const [settingsAnchorEl, setSettingsAnchorEl] = useState<null | HTMLElement>(null);
  const [tenantSelectorOpen, setTenantSelectorOpen] = useState(false);
  const [tenants, setTenants] = useState<Tenant[]>([]);

  const buttonRefs = React.useRef<Record<string, HTMLButtonElement | null>>({});

  // Get current category config
  const currentCategory = selectedCategory ? categoryConfigs.find(c => c.key === selectedCategory) : null;

  // Handle category selection from dropdown - navigate to default page
  const handleCategorySelect = (categoryKey: 'tenants' | 'catalog' | 'weave' | 'studio' | 'workflow' | 'intelligence' | 'entity' | 'calendar') => {
    setSelectedCategory(categoryKey);
    setCategoryMenuAnchorEl(null);
    
    // Navigate to the category's default page
    const category = categoryConfigs.find(c => c.key === categoryKey);
    if (category?.defaultPath) {
      navigate(category.defaultPath);
    }
  };

  // Handle opening menu from top nav
  const handleMenuOpen = (menuLabel: string) => {
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
    // fetch tenants when dialog opens
    let mounted = true;
    fetch('/api/tenants')
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
      const instanceProducts = selectedInstance.tenant_products || [];
      const selectedProduct = instanceProducts[0] || null;
      const selectedDatasource = selectedProduct?.tenant_product_datasources?.[0] || null;
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
    return location.pathname === path;
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
            <Typography variant="h6" component="div" sx={{ fontWeight: 'bold' }}>
              SemLayer
            </Typography>
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
            {categoryConfigs.map((cat) => (
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
            <ThemeToggleButton showMenu />

            <NotificationBell />

            <IconButton color="inherit" onClick={handleSettingsOpen} aria-label="Settings and account menu">
              <SettingsIcon />
            </IconButton>
          </Box>
        </Toolbar>
      </AppBar>

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
            
            return (
              <MenuItem
                key={item.path}
                component={BlockableLink}
                {...{ to: item.path }}
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
