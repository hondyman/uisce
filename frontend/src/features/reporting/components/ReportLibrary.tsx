import React, { useState, useMemo, useEffect, useCallback } from 'react';
import { useNavigate } from 'react-router-dom';
import {
  Box,
  Card,
  CardContent,
  Typography,
  Button,
  Grid,
  List,
  ListItem,
  ListItemText,
  ListItemIcon,
  ListItemButton,
  IconButton,
  TextField,
  InputAdornment,
  Chip,
  Menu,
  MenuItem,
  Dialog,
  DialogTitle,
  DialogContent,
  DialogContentText,
  DialogActions,
  Breadcrumbs,
  Link,
  Paper,
  Divider,
  Avatar,
  Tooltip,
  Badge,
  Stack,
  ToggleButton,
  ToggleButtonGroup,
  CircularProgress,
  Tabs,
  Tab,
  Select,
  FormControl,
  InputLabel,
  Snackbar,
  Alert,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  Autocomplete,
  AlertTitle,
} from '@mui/material';
import {
  Add as AddIcon,
  Folder as FolderIcon,
  FolderOpen as FolderOpenIcon,
  Description as ReportIcon,
  Search as SearchIcon,
  MoreVert as MoreIcon,
  Star as StarIcon,
  StarBorder as StarBorderIcon,
  Share as ShareIcon,
  Schedule as ScheduleIcon,
  Edit as EditIcon,
  Delete as DeleteIcon,
  FileCopy as DuplicateIcon,
  Refresh as RefreshIcon,
  ViewList as ListViewIcon,
  ViewModule as GridViewIcon,
  AccessTime as RecentIcon,
  Person as PersonIcon,
  Group as GroupIcon,
  Public as PublicIcon,
  VerifiedUser as CoreIcon,
  DriveFileMove as MoveIcon,
  PersonAdd as PersonAddIcon,
  Block as BlockIcon,
  CheckCircle as CheckCircleIcon,
  FolderShared as FolderSharedIcon,
  Outbound as OutboundIcon,
  PlayArrow as PlayIcon,
  Close as CloseIcon,
  Storefront as StorefrontIcon,
  AutoAwesome as AutoAwesomeIcon,
} from '@mui/icons-material';
import { formatDistanceToNow } from 'date-fns';
import { useReportTemplates } from '../../../api/reporting';
import { useFolders } from '../../../api/explorer';
import { useTenant } from '../../../contexts/TenantContext';
import { useShareableUsers, ShareableUser } from '../hooks/useShareableUsers';
import { useReportShares, ShareRecord } from '../hooks/useReportShares';
import { UnifiedBOPickerModal } from '../../../components/common/UnifiedBOPickerModal';

// ============================================================================
// Types & Interfaces
// ============================================================================

export interface Collaborator {
  id: string;
  name: string;
  email: string;
  role: 'viewer';
  added_at: string;
  access_path?: 'direct' | 'entitlement';
  is_active?: boolean;
  organization?: string;
}

export interface SavedReport {
  id: string;
  name: string;
  description?: string;
  folder_id?: string;
  created_by: string;
  created_at: string;
  updated_at: string;
  is_favorite: boolean;
  is_scheduled: boolean;
  is_shared: boolean;
  share_type: 'private' | 'shared_by_me' | 'shared_with_me' | 'team' | 'public';
  shared_by?: string;
  shared_by_email?: string;
  shared_with?: Collaborator[];
  last_run?: string;
  run_count: number;
  tags?: string[];
  is_core?: boolean;
  config: any;
  metadata?: {
    report_key?: string;
    data_bindings?: DataBinding[];
    [key: string]: unknown;
  };
}

export interface DataBinding {
  bo_path: string;
  field_allowlist?: string[];
  measures?: { field: string; aggregation?: string }[];
  dimensions?: string[];
  filters?: { field: string; operator: string; value?: unknown; parameter?: string }[];
}

export interface Folder {
  id: string;
  name: string;
  description?: string;
  parent_id?: string;
  is_core: boolean;
  created_by: string;
  report_count: number;
  created_at?: string;
}

const STORAGE_KEY_CUSTOM_FOLDERS = 'uisce_report_custom_folders_v2';
const STORAGE_KEY_REPORTS = 'uisce_report_library_items_v2';

// Standard Core Folders (seeded from Master Tenant / Gold Copy)
const INITIAL_CORE_FOLDERS: Folder[] = [
  {
    id: 'core-folder-financials',
    name: 'Core Financial Reports',
    description: 'Gold copy standard ledger, balance sheet, and AUM reporting',
    is_core: true,
    created_by: 'Master Tenant',
    report_count: 2,
    created_at: new Date('2026-01-01').toISOString(),
  },
  {
    id: 'core-folder-portfolio',
    name: 'Portfolio Operations',
    description: 'Rebalancing, tax-loss harvesting, and factor attribution',
    is_core: true,
    created_by: 'Master Tenant',
    report_count: 2,
    created_at: new Date('2026-01-01').toISOString(),
  },
  {
    id: 'core-folder-compliance',
    name: 'Compliance & Audit',
    description: 'Regulatory SEC/FINRA audit trails, bitemporal diffs, and change logs',
    is_core: true,
    created_by: 'Master Tenant',
    report_count: 1,
    created_at: new Date('2026-01-01').toISOString(),
  },
  {
    id: 'core-folder-executive',
    name: 'Executive & Board',
    description: 'High-level KPI dashboards and risk exposure metrics',
    is_core: true,
    created_by: 'Master Tenant',
    report_count: 1,
    created_at: new Date('2026-01-01').toISOString(),
  },
];

// Initial Seed Reports
const INITIAL_SEED_REPORTS: SavedReport[] = [
  {
    id: 'rep-core-001',
    name: 'AUM & Fee Revenue Summary',
    description: 'Institutional and wealth management advisory fee schedule breakdown by asset tier.',
    folder_id: 'core-folder-financials',
    created_by: 'Master Tenant',
    created_at: new Date(Date.now() - 30 * 86400000).toISOString(),
    updated_at: new Date(Date.now() - 2 * 86400000).toISOString(),
    is_favorite: true,
    is_scheduled: true,
    is_shared: true,
    share_type: 'shared_by_me',
    shared_with: [
      { id: 'a0000002-0002-0002-0002-000000000002', name: 'Sarah Connor', email: 'sarah.c@alpha-wealth.com', role: 'viewer', added_at: '2026-07-10', access_path: 'direct', is_active: true, organization: 'alpha-wealth' },
      { id: 'a0000003-0003-0003-0003-000000000003', name: 'David Vance', email: 'david.v@alpha-wealth.com', role: 'viewer', added_at: '2026-07-15', access_path: 'direct', is_active: true, organization: 'alpha-wealth' },
    ],
    last_run: new Date(Date.now() - 3600000 * 3).toISOString(),
    run_count: 48,
    is_core: true,
    tags: ['AUM', 'Revenue', 'Monthly'],
    metadata: {
      report_key: 'rep-core-001',
      data_bindings: [
        { bo_path: 'oms.account/institutional', field_allowlist: ['sponsor_id', 'mandate_type', 'fee_schedule_code'], measures: [{ field: 'aum_basis_amount', aggregation: 'SUM' }], dimensions: ['fee_schedule_code'] },
        { bo_path: 'master.sales_ledger/aum_management_fee', field_allowlist: ['aum_basis_amount', 'effective_fee_bps', 'billing_period_end'], measures: [{ field: 'aum_basis_amount', aggregation: 'SUM' }], dimensions: ['fee_schedule_code'] },
      ],
    },
    config: {
      _schemaVersion: 2,
      reportTitle: 'AUM & Fee Revenue Summary',
      groupDefinitions: [],
      sections: [
        {
          id: 's1',
          title: 'AUM & Fee Revenue',
          elements: [
            { id: 'el1', type: 'table', columns: [
              { label: 'Account Type', dimension: 'account_type', width: 180 },
              { label: 'AUM (USD)', measure: 'aum_basis_amount', format: 'currency', width: 140, alignment: 'right' as const },
              { label: 'Fee Schedule', dimension: 'fee_schedule_code', width: 140 },
              { label: 'Eff. Fee (bps)', measure: 'effective_fee_bps', format: 'number', width: 120, alignment: 'right' as const },
              { label: 'YTD Revenue', measure: 'ytd_revenue', format: 'currency', width: 140, alignment: 'right' as const },
            ], totals: {}, banding: 'row', freezePane: true, pagination: true },
          ],
        },
      ],
      sectionConfig: { s1: { visible: true, backgroundColor: '#ffffff' } },
    },
  },
  {
    id: 'rep-core-002',
    name: 'Quarterly LP Distribution Matrix',
    description: 'Alternative investment capital call and cash flow waterfall summary across fund tranches.',
    folder_id: 'core-folder-financials',
    created_by: 'Master Tenant',
    created_at: new Date(Date.now() - 45 * 86400000).toISOString(),
    updated_at: new Date(Date.now() - 10 * 86400000).toISOString(),
    is_favorite: false,
    is_scheduled: false,
    is_shared: false,
    share_type: 'private',
    last_run: new Date(Date.now() - 86400000 * 2).toISOString(),
    run_count: 14,
    is_core: true,
    tags: ['Private Markets', 'Distributions'],
    metadata: {
      report_key: 'rep-core-002',
      data_bindings: [
        { bo_path: 'altinv.alternative_investment/private_equity', field_allowlist: ['investment_name', 'vintage_year', 'committed_capital', 'called_capital', 'dpi', 'rvpi'], measures: [{ field: 'called_capital', aggregation: 'SUM' }, { field: 'dpi', aggregation: 'AVG' }], dimensions: ['vintage_year'] },
        { bo_path: 'cash_flow.settlement/lp_distribution', field_allowlist: ['amount', 'due_date', 'return_of_capital', 'preferred_return'], measures: [{ field: 'amount', aggregation: 'SUM' }], dimensions: ['due_date'] },
      ],
    },
    config: {
      _schemaVersion: 2,
      reportTitle: 'Quarterly LP Distribution Matrix',
      groupDefinitions: [],
      sections: [
        {
          id: 's1',
          title: 'LP Distribution',
          elements: [
            { id: 'el1', type: 'table', columns: [
              { label: 'Fund', dimension: 'investment_name', width: 200 },
              { label: 'Vintage', dimension: 'vintage_year', width: 80 },
              { label: 'Quarter', dimension: 'quarter', width: 80 },
              { label: 'Capital Called', measure: 'called_capital', format: 'currency', width: 140, alignment: 'right' as const },
              { label: 'LP Distribution', measure: 'lp_distribution_amount', format: 'currency', width: 140, alignment: 'right' as const },
              { label: 'DPI', measure: 'dpi', format: 'percent', width: 80, alignment: 'right' as const },
              { label: 'RVPI', measure: 'rvpi', format: 'percent', width: 80, alignment: 'right' as const },
            ], totals: {}, banding: 'row', freezePane: true, pagination: true },
          ],
        },
      ],
      sectionConfig: { s1: { visible: true, backgroundColor: '#ffffff' } },
    },
  },
  {
    id: 'rep-core-003',
    name: 'Portfolio Rebalance & TLH Impact',
    description: 'Automated tax-loss harvesting tracking with pre/post wash sale tracking.',
    folder_id: 'core-folder-portfolio',
    created_by: 'Master Tenant',
    created_at: new Date(Date.now() - 20 * 86400000).toISOString(),
    updated_at: new Date(Date.now() - 1 * 86400000).toISOString(),
    is_favorite: true,
    is_scheduled: true,
    is_shared: false,
    share_type: 'private',
    shared_with: [],
    last_run: new Date(Date.now() - 3600000 * 1).toISOString(),
    run_count: 89,
    is_core: true,
    tags: ['Rebalance', 'TLH', 'Tax'],
    metadata: {
      report_key: 'rep-core-003',
      data_bindings: [
        { bo_path: 'oms.position/settled_long', field_allowlist: ['custody_account_id', 'settled_shares', 'cost_basis_method'], measures: [{ field: 'settled_shares', aggregation: 'SUM' }], dimensions: ['custody_account_id'] },
        { bo_path: 'oms.trade_order/block_parent', field_allowlist: ['allocation_profile_id', 'total_requested_quantity', 'average_price'], measures: [{ field: 'total_requested_quantity', aggregation: 'SUM' }], dimensions: ['allocation_profile_id'] },
      ],
    },
    config: {
      _schemaVersion: 2,
      reportTitle: 'Portfolio Rebalance & TLH Impact',
      groupDefinitions: [],
      sections: [
        {
          id: 's1',
          title: 'Rebalance Impact',
          elements: [
            { id: 'el1', type: 'table', columns: [
              { label: 'Account', dimension: 'account_number', width: 150 },
              { label: 'Security', dimension: 'ticker', width: 100 },
              { label: 'Pre-Qty', measure: 'pre_quantity', format: 'number', width: 100, alignment: 'right' as const },
              { label: 'Post-Qty', measure: 'post_quantity', format: 'number', width: 100, alignment: 'right' as const },
              { label: 'Qty Change', measure: 'quantity_change', format: 'number', width: 100, alignment: 'right' as const },
              { label: 'Realized P&L', measure: 'realized_pnl', format: 'currency', width: 120, alignment: 'right' as const },
              { label: 'Tax Alpha', measure: 'tax_alpha', format: 'currency', width: 120, alignment: 'right' as const },
            ], totals: {}, banding: 'row', freezePane: true, pagination: true },
          ],
        },
      ],
      sectionConfig: { s1: { visible: true, backgroundColor: '#ffffff' } },
    },
  },
  {
    id: 'rep-core-004',
    name: 'Multi-Asset Factor Risk Decomposition',
    description: 'Barra-style risk factor exposures across equities, fixed income, and commodities.',
    folder_id: 'core-folder-portfolio',
    created_by: 'Master Tenant',
    created_at: new Date(Date.now() - 15 * 86400000).toISOString(),
    updated_at: new Date(Date.now() - 5 * 86400000).toISOString(),
    is_favorite: false,
    is_scheduled: false,
    is_shared: true,
    share_type: 'shared_with_me',
    shared_by: 'Elena Rostova (Quantitative Risk)',
    shared_by_email: 'elena.r@alpha-wealth.com',
    last_run: new Date(Date.now() - 3600000 * 12).toISOString(),
    run_count: 32,
    is_core: true,
    tags: ['Risk', 'Factor', 'Analytics'],
    metadata: {
      report_key: 'rep-core-004',
      data_bindings: [
        { bo_path: 'oms.position/settled_long', field_allowlist: ['quantity', 'market_value', 'notional_amount'], measures: [{ field: 'notional_amount', aggregation: 'SUM' }], dimensions: ['subtype_code'] },
        { bo_path: 'oms.security/equity', field_allowlist: ['ticker', 'isin'], measures: [], dimensions: ['ticker'] },
      ],
    },
    config: {
      _schemaVersion: 2,
      reportTitle: 'Multi-Asset Factor Risk Decomposition',
      groupDefinitions: [],
      sections: [
        {
          id: 's1',
          title: 'Factor Exposures',
          elements: [
            { id: 'el1', type: 'table', columns: [
              { label: 'Factor', dimension: 'factor_name', width: 160 },
              { label: 'Instrument Class', dimension: 'instrument_class', width: 140 },
              { label: 'Exposure', measure: 'factor_exposure', format: 'number', width: 120, alignment: 'right' as const },
              { label: 'Contribution', measure: 'risk_contribution', format: 'percent', width: 120, alignment: 'right' as const },
            ], totals: {}, banding: 'row', freezePane: true, pagination: true },
          ],
        },
      ],
      sectionConfig: { s1: { visible: true, backgroundColor: '#ffffff' } },
    },
  },
  {
    id: 'rep-core-005',
    name: 'Bitemporal Change Audit Log',
    description: 'SEC rule compliance audit log tracking system-time vs valid-time entity changes.',
    folder_id: 'core-folder-compliance',
    created_by: 'Master Tenant',
    created_at: new Date(Date.now() - 60 * 86400000).toISOString(),
    updated_at: new Date(Date.now() - 3 * 86400000).toISOString(),
    is_favorite: false,
    is_scheduled: true,
    is_shared: false,
    share_type: 'private',
    last_run: new Date(Date.now() - 86400000 * 4).toISOString(),
    run_count: 19,
    is_core: true,
    tags: ['Audit', 'Compliance'],
    metadata: {
      report_key: 'rep-core-005',
      data_bindings: [
        { bo_path: 'oms.account', field_allowlist: ['created_at', 'updated_at', 'valid_from', 'valid_to'], dimensions: ['account_number', 'created_at', 'updated_at', 'valid_from', 'valid_to'] },
      ],
    },
    config: {
      _schemaVersion: 2,
      reportTitle: 'Bitemporal Change Audit Log',
      groupDefinitions: [],
      sections: [
        {
          id: 's1',
          title: 'Audit Entries',
          elements: [
            { id: 'el1', type: 'table', columns: [
              { label: 'Entity', dimension: 'entity_type', width: 140 },
              { label: 'Entity ID', dimension: 'entity_id', width: 200 },
              { label: 'System Time', dimension: 'system_time', format: 'datetime', width: 150 },
              { label: 'Valid From', dimension: 'valid_from', format: 'datetime', width: 150 },
              { label: 'Valid To', dimension: 'valid_to', format: 'datetime', width: 150 },
              { label: 'Changed By', dimension: 'changed_by', width: 140 },
              { label: 'Action', dimension: 'action', width: 80 },
            ], totals: {}, banding: 'row', freezePane: true, pagination: true },
          ],
        },
      ],
      sectionConfig: { s1: { visible: true, backgroundColor: '#ffffff' } },
    },
  },
  {
    id: 'rep-custom-001',
    name: 'High-Net-Worth Household Allocation',
    description: 'Custom client wealth report highlighting municipal debt yield vs equity momentum.',
    folder_id: 'custom-folder-wealth',
    created_by: 'You',
    created_at: new Date(Date.now() - 5 * 86400000).toISOString(),
    updated_at: new Date(Date.now() - 1 * 86400000).toISOString(),
    is_favorite: true,
    is_scheduled: false,
    is_shared: true,
    share_type: 'shared_by_me',
    shared_with: [
      { id: 'a0000004-0004-0004-0004-000000000004', name: 'Marcus Bell', email: 'marcus.b@alpha-wealth.com', role: 'viewer', added_at: '2026-08-18', access_path: 'direct', is_active: true, organization: 'alpha-wealth' },
      { id: 'a0000005-0005-0005-0005-000000000005', name: 'Sophia Lin', email: 'sophia.l@alpha-wealth.com', role: 'viewer', added_at: '2026-08-20', access_path: 'direct', is_active: true, organization: 'alpha-wealth' },
    ],
    last_run: new Date(Date.now() - 3600000 * 2).toISOString(),
    run_count: 27,
    is_core: false,
    tags: ['HNW', 'Custom', 'Advisory'],
    metadata: {
      report_key: 'rep-custom-001',
      data_bindings: [
        { bo_path: 'oms.account/retail_wealth', field_allowlist: ['tax_id_type', 'accredited_investor_status'], dimensions: ['account_number'] },
        { bo_path: 'oms.position/settled_long', field_allowlist: ['quantity', 'market_value'], measures: [{ field: 'market_value', aggregation: 'SUM' }], dimensions: ['account_id'] },
        { bo_path: 'oms.security/sovereign_debt', field_allowlist: ['ticker', 'coupon_rate', 'credit_rating_sp'], dimensions: ['ticker'] },
      ],
    },
    config: {
      _schemaVersion: 2,
      reportTitle: 'High-Net-Worth Household Allocation',
      groupDefinitions: [],
      sections: [
        {
          id: 's1',
          title: 'Asset Allocation',
          elements: [
            { id: 'el1', type: 'table', columns: [
              { label: 'Client', dimension: 'customer_name', width: 180 },
              { label: 'Asset Class', dimension: 'asset_class', width: 140 },
              { label: 'Market Value', measure: 'market_value', format: 'currency', width: 140, alignment: 'right' as const },
              { label: 'Yield', measure: 'coupon_rate', format: 'percent', width: 100, alignment: 'right' as const },
              { label: 'Credit Rating', dimension: 'credit_rating_sp', width: 100 },
            ], totals: {}, banding: 'row', freezePane: true, pagination: true },
          ],
        },
      ],
      sectionConfig: { s1: { visible: true, backgroundColor: '#ffffff' } },
    },
  },
  {
    id: 'rep-shared-002',
    name: 'Executive Liquidity & Cash Flow Forecast',
    description: 'Rolling 30/60/90 day treasury cash obligations and settlement recon.',
    folder_id: 'core-folder-executive',
    created_by: 'Chief Risk Officer',
    created_at: new Date(Date.now() - 12 * 86400000).toISOString(),
    updated_at: new Date(Date.now() - 2 * 86400000).toISOString(),
    is_favorite: false,
    is_scheduled: false,
    is_shared: true,
    share_type: 'shared_with_me',
    shared_by: 'Jonathan Pierce',
    shared_by_email: 'jonah.pierce@alpha-wealth.com',
    last_run: new Date(Date.now() - 3600000 * 8).toISOString(),
    run_count: 41,
    is_core: true,
    tags: ['Treasury', 'Liquidity'],
    metadata: {
      report_key: 'rep-shared-002',
      data_bindings: [
        { bo_path: 'cash_flow.settlement/dividend', field_allowlist: ['amount', 'due_date', 'currency'], measures: [{ field: 'amount', aggregation: 'SUM' }], dimensions: ['due_date', 'currency'] },
        { bo_path: 'cash_flow.settlement/lp_distribution', field_allowlist: ['amount', 'due_date', 'return_of_capital'], measures: [{ field: 'amount', aggregation: 'SUM' }], dimensions: ['due_date'] },
        { bo_path: 'cash_flow.settlement/capital_call', field_allowlist: ['amount', 'due_date', 'management_fee_portion'], measures: [{ field: 'amount', aggregation: 'SUM' }], dimensions: ['due_date'] },
        { bo_path: 'oms.account/corporate_treasury', field_allowlist: ['wire_limit_daily', 'base_currency'], dimensions: ['base_currency'] },
      ],
    },
    config: {
      _schemaVersion: 2,
      reportTitle: 'Executive Liquidity & Cash Flow Forecast',
      groupDefinitions: [],
      sections: [
        {
          id: 's1',
          title: '30/60/90 Day Forecast',
          elements: [
            { id: 'el1', type: 'table', columns: [
              { label: 'Currency', dimension: 'currency', width: 100 },
              { label: 'Bucket', dimension: 'bucket', width: 100 },
              { label: 'Settlement Type', dimension: 'subtype_code', width: 160 },
              { label: 'Amount', measure: 'amount', format: 'currency', width: 140, alignment: 'right' as const },
              { label: 'Due Date', dimension: 'due_date', format: 'date', width: 120 },
            ], totals: {}, banding: 'row', freezePane: true, pagination: true },
          ],
        },
      ],
      sectionConfig: { s1: { visible: true, backgroundColor: '#ffffff' } },
    },
  },
];

// ============================================================================
// Component
// ============================================================================

export const ReportLibrary: React.FC = () => {
  const navigate = useNavigate();
  const { tenant } = useTenant();

  // --- API Hooks ---
  const { data: apiReports, isLoading: isLoadingReports } = useReportTemplates();
  const { data: apiFolders, isLoading: isLoadingFolders } = useFolders();

  // --- Sharing Hooks ---
  const {
    users: shareableUsers,
    loading: shareableUsersLoading,
    fetchUsers: fetchShareableUsers,
  } = useShareableUsers(tenant?.id);

  const [customFolders, setCustomFolders] = useState<Folder[]>(() => {
    try {
      const saved = localStorage.getItem(STORAGE_KEY_CUSTOM_FOLDERS);
      if (saved) {
        return JSON.parse(saved);
      }
    } catch (e) {
      console.warn('Failed to parse cached custom folders', e);
    }
    return [
      {
        id: 'custom-folder-wealth',
        name: 'Private Wealth Advisory',
        description: 'Bespoke client portfolio packages and tax review docs',
        is_core: false,
        created_by: 'You',
        report_count: 1,
        created_at: new Date('2026-07-20').toISOString(),
      },
      {
        id: 'custom-folder-adhoc',
        name: 'Ad-Hoc Research & Notes',
        description: 'Draft reports and exploratory queries',
        is_core: false,
        created_by: 'You',
        report_count: 0,
        created_at: new Date('2026-08-01').toISOString(),
      },
    ];
  });

  // --- Local Reports State with Storage Cache ---
  const [localReports, setLocalReports] = useState<SavedReport[]>(() => {
    try {
      const saved = localStorage.getItem(STORAGE_KEY_REPORTS);
      if (saved) {
        return JSON.parse(saved);
      }
    } catch (e) {
      console.warn('Failed to parse cached reports', e);
    }
    return INITIAL_SEED_REPORTS;
  });

  // Sync API reports if present
  useEffect(() => {
    if (apiReports && apiReports.length > 0) {
      setLocalReports((prev) => {
        const merged = [...prev];
        apiReports.forEach((apiR) => {
          if (!merged.some((m) => m.id === apiR.id)) {
            merged.push({
              id: apiR.id,
              name: apiR.name,
              description: apiR.description || '',
              folder_id: (apiR.metadata as any)?.folder_id || undefined,
              created_by: (apiR.metadata as any)?.created_by || 'Master Tenant',
              created_at: apiR.createdAt || new Date().toISOString(),
              updated_at: apiR.updatedAt || new Date().toISOString(),
              is_favorite: (apiR.metadata as any)?.is_favorite || false,
              is_scheduled: (apiR.metadata as any)?.is_scheduled || false,
              is_shared: (apiR.metadata as any)?.is_shared || false,
              share_type: (apiR.metadata as any)?.share_type || 'private',
              shared_by: (apiR.metadata as any)?.shared_by,
              shared_with: (apiR.metadata as any)?.shared_with || [],
              last_run: (apiR.metadata as any)?.last_run,
              run_count: (apiR.metadata as any)?.run_count || 0,
              is_core: (apiR.metadata as any)?.is_core || true,
              config: apiR.definition || {},
            });
          }
        });
        return merged;
      });
    }
  }, [apiReports]);

  // Persist custom folders
  useEffect(() => {
    localStorage.setItem(STORAGE_KEY_CUSTOM_FOLDERS, JSON.stringify(customFolders));
  }, [customFolders]);

  // Persist reports
  useEffect(() => {
    localStorage.setItem(STORAGE_KEY_REPORTS, JSON.stringify(localReports));
  }, [localReports]);

  // --- Navigation & Filter State ---
  const [currentFolder, setCurrentFolder] = useState<string | null>(null);
  const [searchQuery, setSearchQuery] = useState('');
  const [viewMode, setViewMode] = useState<'list' | 'grid'>('list');
  const [filterType, setFilterType] = useState<'all' | 'favorites' | 'recent' | 'shared'>('all');
  const [sharedSubFilter, setSharedSubFilter] = useState<'all' | 'shared_by_me' | 'shared_with_me'>('all');

  // --- Selected Report & Menus ---
  const [selectedReport, setSelectedReport] = useState<SavedReport | null>(null);
  const [menuAnchor, setMenuAnchor] = useState<null | HTMLElement>(null);

  // --- Folder Management Modals ---
  const [folderDialogOpen, setFolderDialogOpen] = useState(false);
  const [folderDialogMode, setFolderDialogMode] = useState<'create' | 'edit'>('create');
  const [editingFolderId, setEditingFolderId] = useState<string | null>(null);
  const [folderName, setFolderName] = useState('');
  const [folderDescription, setFolderDescription] = useState('');
  const [deleteFolderDialogOpen, setDeleteFolderDialogOpen] = useState(false);
  const [folderToDelete, setFolderToDelete] = useState<Folder | null>(null);
  const [folderMenuAnchor, setFolderMenuAnchor] = useState<null | HTMLElement>(null);
  const [activeFolderMenu, setActiveFolderMenu] = useState<Folder | null>(null);

  // --- Move to Folder Dialog ---
  const [moveToFolderDialogOpen, setMoveToFolderDialogOpen] = useState(false);
  const [targetFolderId, setTargetFolderId] = useState<string>('none');

  // --- Sharing Dialogs ---
  const [manageShareDialogOpen, setManageShareDialogOpen] = useState(false);
  const [sharedWithMeDialogOpen, setSharedWithMeDialogOpen] = useState(false);
  const [newCollaboratorUser, setNewCollaboratorUser] = useState<ShareableUser | null>(null);
  const [stopShareConfirmOpen, setStopShareConfirmOpen] = useState(false);
  const [shareDialogTab, setShareDialogTab] = useState<'people' | 'activity'>('people');
  const [shareMode, setShareMode] = useState<'people' | 'team'>('people');

  // Mock team/role options for team-sharing mode
  const TEAM_ROLE_OPTIONS: { id: string; name: string; type: 'role' | 'team'; description: string }[] = [
    { id: 'role:portfolio_manager', name: 'All Portfolio Managers', type: 'role', description: 'Portfolio management team' },
    { id: 'role:client_advisor', name: 'All Client Advisors', type: 'role', description: 'Wealth advisory team' },
    { id: 'role:compliance_officer', name: 'All Compliance Officers', type: 'role', description: 'Compliance team' },
    { id: 'role:trade_execution', name: 'All Trade Execution', type: 'role', description: 'Trading desk' },
    { id: 'role:admin', name: 'All Administrators', type: 'role', description: 'Admin users' },
  ];

  // --- Sharing Hooks (must be after selectedReport state) ---
  const {
    shares,
    loading: sharesLoading,
    fetchShares,
    addShare,
    removeShare,
    updateShare,
    stopAllShares,
    cloneReport,
  } = useReportShares(selectedReport?.id);

  // --- New Report Modal ---
  const [newReportModalOpen, setNewReportModalOpen] = useState(false);

  // --- Delete Report Dialog ---
  const [deleteReportConfirmOpen, setDeleteReportConfirmOpen] = useState(false);

  // --- Notification Toast ---
  const [snackbarMessage, setSnackbarMessage] = useState<string | null>(null);
  const [snackbarSeverity, setSnackbarSeverity] = useState<'success' | 'info' | 'warning' | 'error'>('success');

  const showNotification = (msg: string, severity: 'success' | 'info' | 'warning' | 'error' = 'success') => {
    setSnackbarMessage(msg);
    setSnackbarSeverity(severity);
  };

  // Combine Core Folders and Custom Folders, computing dynamic report counts
  const allFolders = useMemo<Folder[]>(() => {
    const combined = [...INITIAL_CORE_FOLDERS, ...customFolders];
    return combined.map((f) => ({
      ...f,
      report_count: localReports.filter((r) => r.folder_id === f.id).length,
    }));
  }, [customFolders, localReports]);

  const coreFolders = useMemo(() => allFolders.filter((f) => f.is_core), [allFolders]);
  const userFolders = useMemo(() => allFolders.filter((f) => !f.is_core), [allFolders]);

  const currentFolderObject = useMemo(() => {
    if (!currentFolder) return null;
    return allFolders.find((f) => f.id === currentFolder) || null;
  }, [currentFolder, allFolders]);

  // Counts for sidebar badges
  const favoritesCount = useMemo(() => localReports.filter((r) => r.is_favorite).length, [localReports]);
  const recentCount = useMemo(() => localReports.filter((r) => Boolean(r.last_run)).length, [localReports]);
  const sharedCount = useMemo(() => localReports.filter((r) => r.is_shared).length, [localReports]);
  const sharedByMeCount = useMemo(() => localReports.filter((r) => r.share_type === 'shared_by_me').length, [localReports]);
  const sharedWithMeCount = useMemo(() => localReports.filter((r) => r.share_type === 'shared_with_me').length, [localReports]);

  // --- Filtering Reports ---
  const filteredReports = useMemo(() => {
    return localReports.filter((report) => {
      // 1. Search Query
      if (searchQuery.trim()) {
        const q = searchQuery.toLowerCase();
        const matchesName = report.name.toLowerCase().includes(q);
        const matchesDesc = (report.description || '').toLowerCase().includes(q);
        const matchesCreator = report.created_by.toLowerCase().includes(q);
        const matchesTags = report.tags ? report.tags.some((t) => t.toLowerCase().includes(q)) : false;
        if (!matchesName && !matchesDesc && !matchesCreator && !matchesTags) {
          return false;
        }
      }

      // 2. Main Navigation Mode
      if (filterType === 'favorites') {
        return report.is_favorite;
      }

      if (filterType === 'recent') {
        return Boolean(report.last_run);
      }

      if (filterType === 'shared') {
        if (!report.is_shared) return false;
        if (sharedSubFilter === 'shared_by_me') return report.share_type === 'shared_by_me';
        if (sharedSubFilter === 'shared_with_me') return report.share_type === 'shared_with_me';
        return true;
      }

      // 3. Folder Filter (when in 'all' view)
      if (currentFolder) {
        return report.folder_id === currentFolder;
      }

      return true;
    }).sort((a, b) => {
      // If in recent view, sort by last_run descending
      if (filterType === 'recent') {
        const timeA = a.last_run ? new Date(a.last_run).getTime() : 0;
        const timeB = b.last_run ? new Date(b.last_run).getTime() : 0;
        return timeB - timeA;
      }
      // Otherwise sort favorites first, then alphabetically
      if (a.is_favorite !== b.is_favorite) {
        return a.is_favorite ? -1 : 1;
      }
      return a.name.localeCompare(b.name);
    });
  }, [localReports, searchQuery, filterType, sharedSubFilter, currentFolder]);

  // --- Action Handlers ---

  const handleMenuOpen = (event: React.MouseEvent<HTMLElement>, report: SavedReport) => {
    event.stopPropagation();
    setSelectedReport(report);
    setMenuAnchor(event.currentTarget);
  };

  const handleMenuClose = () => {
    setMenuAnchor(null);
  };

  // Toggle Favorite
  const handleToggleFavorite = (reportId: string, event?: React.MouseEvent) => {
    if (event) event.stopPropagation();
    setLocalReports((prev) =>
      prev.map((r) => {
        if (r.id === reportId) {
          const nextVal = !r.is_favorite;
          showNotification(nextVal ? `Added "${r.name}" to favorites` : `Removed "${r.name}" from favorites`, 'info');
          return { ...r, is_favorite: nextVal };
        }
        return r;
      })
    );
    if (selectedReport && selectedReport.id === reportId) {
      setSelectedReport((prev) => (prev ? { ...prev, is_favorite: !prev.is_favorite } : null));
    }
  };

  // Execute / Run Report
  const handleRunReport = (report: SavedReport) => {
    handleMenuClose();
    const now = new Date().toISOString();
    setLocalReports((prev) =>
      prev.map((r) => {
        if (r.id === report.id) {
          return { ...r, last_run: now, run_count: r.run_count + 1 };
        }
        return r;
      })
    );
    showNotification(`Executed "${report.name}" successfully`, 'success');
  };

  // Edit Report
  const handleEditReport = (report: SavedReport) => {
    handleMenuClose();
    navigate(`/reports/${report.id}/edit`);
  };

  // Duplicate Report
  const handleDuplicateReport = (report: SavedReport) => {
    handleMenuClose();
    const newReport: SavedReport = {
      ...report,
      id: `rep-copy-${Date.now()}`,
      name: `${report.name} (Copy)`,
      created_by: 'You',
      created_at: new Date().toISOString(),
      updated_at: new Date().toISOString(),
      is_core: false,
      is_shared: false,
      share_type: 'private',
      shared_with: [],
      run_count: 0,
      last_run: undefined,
    };
    setLocalReports((prev) => [newReport, ...prev]);
    showNotification(`Duplicated "${report.name}"`, 'success');
  };

  // Delete Report
  const handleDeleteReportConfirm = () => {
    if (!selectedReport) return;
    setLocalReports((prev) => prev.filter((r) => r.id !== selectedReport.id));
    setDeleteReportConfirmOpen(false);
    showNotification(`Deleted report "${selectedReport.name}"`, 'info');
    setSelectedReport(null);
  };

  // Open Move to Folder Dialog
  const handleOpenMoveDialog = (report: SavedReport) => {
    handleMenuClose();
    setSelectedReport(report);
    setTargetFolderId(report.folder_id || 'none');
    setMoveToFolderDialogOpen(true);
  };

  // Save Move to Folder
  const handleSaveMoveToFolder = () => {
    if (!selectedReport) return;
    const destFolderId = targetFolderId === 'none' ? undefined : targetFolderId;
    setLocalReports((prev) =>
      prev.map((r) => (r.id === selectedReport.id ? { ...r, folder_id: destFolderId } : r))
    );
    const destName = targetFolderId === 'none' ? 'Root (No Folder)' : allFolders.find((f) => f.id === targetFolderId)?.name || 'Folder';
    showNotification(`Moved "${selectedReport.name}" to ${destName}`, 'success');
    setMoveToFolderDialogOpen(false);
    setSelectedReport(null);
  };

  // --- Folder Management Handlers ---

  const handleOpenCreateFolderDialog = () => {
    setFolderDialogMode('create');
    setFolderName('');
    setFolderDescription('');
    setEditingFolderId(null);
    setFolderDialogOpen(true);
  };

  const handleOpenEditFolderDialog = (folder: Folder) => {
    setFolderDialogMode('edit');
    setFolderName(folder.name);
    setFolderDescription(folder.description || '');
    setEditingFolderId(folder.id);
    setFolderMenuAnchor(null);
    setFolderDialogOpen(true);
  };

  const handleSaveFolder = () => {
    if (!folderName.trim()) return;

    if (folderDialogMode === 'create') {
      const newF: Folder = {
        id: `custom-folder-${Date.now()}`,
        name: folderName.trim(),
        description: folderDescription.trim() || undefined,
        is_core: false,
        created_by: 'You',
        report_count: 0,
        created_at: new Date().toISOString(),
      };
      setCustomFolders((prev) => [...prev, newF]);
      showNotification(`Created folder "${newF.name}"`, 'success');
      setCurrentFolder(newF.id);
      setFilterType('all');
    } else if (folderDialogMode === 'edit' && editingFolderId) {
      setCustomFolders((prev) =>
        prev.map((f) =>
          f.id === editingFolderId
            ? { ...f, name: folderName.trim(), description: folderDescription.trim() || undefined }
            : f
        )
      );
      showNotification(`Updated folder "${folderName.trim()}"`, 'success');
    }

    setFolderDialogOpen(false);
    setFolderName('');
    setFolderDescription('');
    setEditingFolderId(null);
  };

  const handleOpenDeleteFolderConfirm = (folder: Folder) => {
    setFolderToDelete(folder);
    setFolderMenuAnchor(null);
    setDeleteFolderDialogOpen(true);
  };

  const handleDeleteFolderConfirm = () => {
    if (!folderToDelete) return;
    // Remove folder and unassign its reports to root
    setCustomFolders((prev) => prev.filter((f) => f.id !== folderToDelete.id));
    setLocalReports((prev) =>
      prev.map((r) => (r.folder_id === folderToDelete.id ? { ...r, folder_id: undefined } : r))
    );
    if (currentFolder === folderToDelete.id) {
      setCurrentFolder(null);
    }
    showNotification(`Deleted folder "${folderToDelete.name}". Contained reports moved to root.`, 'info');
    setDeleteFolderDialogOpen(false);
    setFolderToDelete(null);
  };

  // --- Sharing Handlers ---

  const handleOpenShareManagement = (report: SavedReport) => {
    handleMenuClose();
    setSelectedReport(report);
    if (report.share_type === 'shared_with_me') {
      setSharedWithMeDialogOpen(true);
    } else {
      setManageShareDialogOpen(true);
      if (tenant?.id) fetchShareableUsers();
      fetchShares();
    }
  };

  const handleAddCollaborator = async () => {
    if (!selectedReport || !newCollaboratorUser) return;

    try {
      await addShare(newCollaboratorUser.id, 'view');
      await fetchShares();
      setNewCollaboratorUser(null);
      showNotification(
        `Shared "${selectedReport.name}" with ${newCollaboratorUser.email}`,
        'success'
      );
    } catch (err: any) {
      showNotification(`Failed to share: ${err.message}`, 'error');
    }
  };

  const handleRemoveCollaborator = async (collaboratorId: string) => {
    if (!selectedReport) return;
    try {
      await removeShare(collaboratorId);
      await fetchShares();
      showNotification('Collaborator access removed', 'info');
    } catch (err: any) {
      showNotification(`Failed to remove collaborator: ${err.message}`, 'error');
    }
  };

  // Stop Sharing Completely
  const handleStopSharing = async () => {
    if (!selectedReport) return;
    try {
      await stopAllShares();
      setLocalReports((prev) =>
        prev.map((r) =>
          r.id === selectedReport.id
            ? { ...r, is_shared: false, share_type: 'private', shared_with: [] }
            : r
        )
      );
      setSelectedReport((prev) =>
        prev ? { ...prev, is_shared: false, share_type: 'private', shared_with: [] } : null
      );
      setStopShareConfirmOpen(false);
      setManageShareDialogOpen(false);
      showNotification(`Stopped sharing "${selectedReport.name}". It is now private.`, 'warning');
    } catch (err: any) {
      showNotification(`Failed to stop sharing: ${err.message}`, 'error');
    }
  };

  // Remove report shared with me
  const handleRemoveSharedWithMe = () => {
    if (!selectedReport) return;
    setLocalReports((prev) => prev.filter((r) => r.id !== selectedReport.id));
    setSharedWithMeDialogOpen(false);
    showNotification(`Removed "${selectedReport.name}" from your shared reports`, 'info');
    setSelectedReport(null);
  };

  // Clone shared report and open in builder
  const handleCloneAndEdit = async () => {
    if (!selectedReport) return;
    try {
      const result = await cloneReport();
      if (result) {
        const newReport: SavedReport = {
          ...selectedReport,
          id: result.id,
          name: result.name,
          created_by: 'You',
          created_at: new Date().toISOString(),
          updated_at: new Date().toISOString(),
          is_shared: false,
          share_type: 'private',
          shared_with: [],
          run_count: 0,
          last_run: undefined,
          is_core: false,
        };
        setLocalReports((prev) => [newReport, ...prev]);
        setSharedWithMeDialogOpen(false);
        navigate(`/reports/builder/${result.id}`);
        showNotification(`Cloned — you now own your own editable copy.`, 'success');
      }
    } catch (err: any) {
      showNotification(`Failed to clone: ${err.message}`, 'error');
    }
  };

  return (
    <Box sx={{ p: { xs: 2, md: 3 }, minHeight: '100vh', bgcolor: 'background.default' }}>
      {/* Header */}
      <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start', mb: 3, flexWrap: 'wrap', gap: 2 }}>
        <Box>
          <Box sx={{ display: 'flex', alignItems: 'center', gap: 1.5 }}>
            <Typography variant="h4" fontWeight={700} sx={{ letterSpacing: '-0.02em' }}>
              Report Studio Library
            </Typography>
            <Chip label="Enterprise" size="small" color="primary" variant="outlined" sx={{ fontWeight: 600 }} />
          </Box>
          <Breadcrumbs sx={{ mt: 1 }}>
            <Link
              component="button"
              variant="body2"
              onClick={() => {
                setFilterType('all');
                setCurrentFolder(null);
              }}
              sx={{ cursor: 'pointer', fontWeight: !currentFolder && filterType === 'all' ? 700 : 400 }}
            >
              All Reports
            </Link>
            {filterType === 'favorites' && <Typography variant="body2" color="primary.main" fontWeight={600}>Favorites</Typography>}
            {filterType === 'recent' && <Typography variant="body2" color="primary.main" fontWeight={600}>Recently Executed</Typography>}
            {filterType === 'shared' && (
              <Typography variant="body2" color="primary.main" fontWeight={600}>
                Shared Reports {sharedSubFilter === 'shared_by_me' ? '(Shared by You)' : sharedSubFilter === 'shared_with_me' ? '(Shared with You)' : ''}
              </Typography>
            )}
            {filterType === 'all' && currentFolderObject && (
              <Box sx={{ display: 'flex', alignItems: 'center', gap: 0.5 }}>
                {currentFolderObject.is_core && <CoreIcon fontSize="small" color="primary" sx={{ fontSize: 16 }} />}
                <Typography variant="body2" color="text.primary" fontWeight={600}>
                  {currentFolderObject.name}
                </Typography>
              </Box>
            )}
          </Breadcrumbs>
        </Box>

        <Box sx={{ display: 'flex', gap: 1.5, alignItems: 'center' }}>
          <Button
            variant="outlined"
            startIcon={<StorefrontIcon />}
            onClick={() => navigate('/reports/marketplace')}
            sx={{ textTransform: 'none', fontWeight: 600, borderRadius: 2, borderColor: '#00D4FF', color: '#00D4FF' }}
          >
            Template Marketplace & AI
          </Button>
          <Button
            variant="outlined"
            startIcon={<FolderIcon />}
            onClick={handleOpenCreateFolderDialog}
            sx={{ textTransform: 'none', fontWeight: 600, borderRadius: 2 }}
          >
            New Folder
          </Button>
          <Button
            variant="contained"
            startIcon={<AddIcon />}
            onClick={() => setNewReportModalOpen(true)}
            sx={{ textTransform: 'none', fontWeight: 600, borderRadius: 2, px: 2.5 }}
          >
            New Report
          </Button>
        </Box>
      </Box>

      <Grid container spacing={3}>
        {/* =========================================================================
            SIDEBAR: Filters & Folder Tree
            ========================================================================= */}
        <Grid size={{ xs: 12, md: 3 }}>
          <Card elevation={0} sx={{ border: '1px solid', borderColor: 'divider', borderRadius: 3, overflow: 'hidden' }}>
            <List sx={{ py: 1 }}>
              <ListItemButton
                selected={filterType === 'all' && !currentFolder}
                onClick={() => {
                  setFilterType('all');
                  setCurrentFolder(null);
                }}
                sx={{ borderRadius: 1.5, mx: 1, mb: 0.5 }}
              >
                <ListItemIcon sx={{ minWidth: 36, color: filterType === 'all' && !currentFolder ? 'primary.main' : 'inherit' }}>
                  <ReportIcon />
                </ListItemIcon>
                <ListItemText slotProps={{ primary: { component: 'div' }, secondary: { component: 'div' } }} primary={<Typography variant="body2" fontWeight={filterType === 'all' && !currentFolder ? 700 : 500}>All Reports</Typography>} />
                <Chip label={localReports.length} size="small" variant="outlined" sx={{ height: 20, fontSize: 11 }} />
              </ListItemButton>

              <ListItemButton
                selected={filterType === 'favorites'}
                onClick={() => {
                  setFilterType('favorites');
                  setCurrentFolder(null);
                }}
                sx={{ borderRadius: 1.5, mx: 1, mb: 0.5 }}
              >
                <ListItemIcon sx={{ minWidth: 36, color: filterType === 'favorites' ? 'warning.main' : 'warning.light' }}>
                  <StarIcon />
                </ListItemIcon>
                <ListItemText slotProps={{ primary: { component: 'div' }, secondary: { component: 'div' } }} primary={<Typography variant="body2" fontWeight={filterType === 'favorites' ? 700 : 500}>Favorites</Typography>} />
                {favoritesCount > 0 && (
                  <Chip label={favoritesCount} size="small" color="warning" sx={{ height: 20, fontSize: 11, fontWeight: 700 }} />
                )}
              </ListItemButton>

              <ListItemButton
                selected={filterType === 'recent'}
                onClick={() => {
                  setFilterType('recent');
                  setCurrentFolder(null);
                }}
                sx={{ borderRadius: 1.5, mx: 1, mb: 0.5 }}
              >
                <ListItemIcon sx={{ minWidth: 36, color: filterType === 'recent' ? 'primary.main' : 'inherit' }}>
                  <RecentIcon />
                </ListItemIcon>
                <ListItemText slotProps={{ primary: { component: 'div' }, secondary: { component: 'div' } }} primary={<Typography variant="body2" fontWeight={filterType === 'recent' ? 700 : 500}>Recently Executed</Typography>} />
                {recentCount > 0 && (
                  <Chip label={recentCount} size="small" variant="outlined" sx={{ height: 20, fontSize: 11 }} />
                )}
              </ListItemButton>

              <ListItemButton
                selected={filterType === 'shared'}
                onClick={() => {
                  setFilterType('shared');
                  setCurrentFolder(null);
                }}
                sx={{ borderRadius: 1.5, mx: 1, mb: 0.5 }}
              >
                <ListItemIcon sx={{ minWidth: 36, color: filterType === 'shared' ? 'primary.main' : 'inherit' }}>
                  <ShareIcon />
                </ListItemIcon>
                <ListItemText slotProps={{ primary: { component: 'div' }, secondary: { component: 'div' } }} primary={<Typography variant="body2" fontWeight={filterType === 'shared' ? 700 : 500}>Shared Reports</Typography>} />
                {sharedCount > 0 && (
                  <Chip label={sharedCount} size="small" color="info" sx={{ height: 20, fontSize: 11, fontWeight: 700 }} />
                )}
              </ListItemButton>
            </List>

            <Divider sx={{ my: 1 }} />

            {/* Core Folders Section */}
            <Box sx={{ px: 2, pt: 1, pb: 0.5, display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
              <Box sx={{ display: 'flex', alignItems: 'center', gap: 0.8 }}>
                <CoreIcon color="primary" sx={{ fontSize: 16 }} />
                <Typography variant="caption" fontWeight={700} color="text.secondary" sx={{ letterSpacing: '0.05em', textTransform: 'uppercase' }}>
                  Core Folders (System)
                </Typography>
              </Box>
              <Chip label="Gold Copy" size="small" sx={{ height: 18, fontSize: 10, bgcolor: 'primary.50', color: 'primary.700', fontWeight: 600 }} />
            </Box>

            <List sx={{ py: 0.5 }}>
              {coreFolders.map((folder) => {
                const isSelected = filterType === 'all' && currentFolder === folder.id;
                return (
                  <ListItemButton
                    key={folder.id}
                    selected={isSelected}
                    onClick={() => {
                      setFilterType('all');
                      setCurrentFolder(folder.id);
                    }}
                    sx={{ borderRadius: 1.5, mx: 1, mb: 0.5 }}
                  >
                    <ListItemIcon sx={{ minWidth: 36 }}>
                      {isSelected ? <FolderOpenIcon color="primary" /> : <FolderIcon color="action" />}
                    </ListItemIcon>
                    <ListItemText
                      slotProps={{ primary: { component: 'div' }, secondary: { component: 'div' } }}
                      primary={
                        <Box sx={{ display: 'flex', alignItems: 'center', gap: 0.8 }}>
                          <Typography variant="body2" noWrap fontWeight={isSelected ? 700 : 500} sx={{ maxWidth: 140 }}>
                            {folder.name}
                          </Typography>
                        </Box>
                      }
                      secondary={
                        <Typography variant="caption" color="text.secondary">
                          {folder.report_count} {folder.report_count === 1 ? 'report' : 'reports'}
                        </Typography>
                      }
                    />
                  </ListItemButton>
                );
              })}
            </List>

            <Divider sx={{ my: 1 }} />

            {/* Custom Folders Section */}
            <Box sx={{ px: 2, pt: 1, pb: 0.5, display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
              <Typography variant="caption" fontWeight={700} color="text.secondary" sx={{ letterSpacing: '0.05em', textTransform: 'uppercase' }}>
                My Folders
              </Typography>
              <Tooltip title="Create new folder">
                <IconButton size="small" onClick={handleOpenCreateFolderDialog} sx={{ p: 0.5 }}>
                  <AddIcon fontSize="small" />
                </IconButton>
              </Tooltip>
            </Box>

            <List sx={{ py: 0.5 }}>
              {userFolders.length === 0 ? (
                <Box sx={{ px: 2, py: 2, textAlign: 'center' }}>
                  <Typography variant="caption" color="text.secondary">
                    No custom folders yet. Click "+" to create one.
                  </Typography>
                </Box>
              ) : (
                userFolders.map((folder) => {
                  const isSelected = filterType === 'all' && currentFolder === folder.id;
                  return (
                    <ListItem
                      key={folder.id}
                      disablePadding
                      secondaryAction={
                        <IconButton
                          edge="end"
                          size="small"
                          onClick={(e) => {
                            e.stopPropagation();
                            setActiveFolderMenu(folder);
                            setFolderMenuAnchor(e.currentTarget);
                          }}
                        >
                          <MoreIcon fontSize="small" />
                        </IconButton>
                      }
                      sx={{ px: 1, mb: 0.5 }}
                    >
                      <ListItemButton
                        selected={isSelected}
                        onClick={() => {
                          setFilterType('all');
                          setCurrentFolder(folder.id);
                        }}
                        sx={{ borderRadius: 1.5, pr: 4 }}
                      >
                        <ListItemIcon sx={{ minWidth: 36 }}>
                          {isSelected ? <FolderOpenIcon color="primary" /> : <FolderIcon color="primary" />}
                        </ListItemIcon>
                        <ListItemText
                          slotProps={{ primary: { component: 'div' }, secondary: { component: 'div' } }}
                          primary={
                            <Typography variant="body2" noWrap fontWeight={isSelected ? 700 : 500} sx={{ maxWidth: 120 }}>
                              {folder.name}
                            </Typography>
                          }
                          secondary={
                            <Typography variant="caption" color="text.secondary">
                              {folder.report_count} {folder.report_count === 1 ? 'report' : 'reports'}
                            </Typography>
                          }
                        />
                      </ListItemButton>
                    </ListItem>
                  );
                })
              )}
            </List>
          </Card>
        </Grid>

        {/* =========================================================================
            MAIN CONTENT AREA: Filters Bar & Reports Display
            ========================================================================= */}
        <Grid size={{ xs: 12, md: 9 }}>
          {/* Controls Bar */}
          <Card elevation={0} sx={{ p: 2, mb: 2.5, border: '1px solid', borderColor: 'divider', borderRadius: 3 }}>
            <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', flexWrap: 'wrap', gap: 2 }}>
              <Box sx={{ display: 'flex', alignItems: 'center', gap: 2, flex: 1, minWidth: 260 }}>
                <TextField
                  placeholder="Search by report name, tag, or creator..."
                  size="small"
                  fullWidth
                  value={searchQuery}
                  onChange={(e) => setSearchQuery(e.target.value)}
                  InputProps={{
                    startAdornment: (
                      <InputAdornment position="start">
                        <SearchIcon color="action" />
                      </InputAdornment>
                    ),
                    endAdornment: searchQuery ? (
                      <InputAdornment position="end">
                        <IconButton size="small" onClick={() => setSearchQuery('')}>
                          <CloseIcon fontSize="small" />
                        </IconButton>
                      </InputAdornment>
                    ) : null,
                  }}
                  sx={{ maxWidth: 420 }}
                />
              </Box>

              <Box sx={{ display: 'flex', alignItems: 'center', gap: 1.5 }}>
                <ToggleButtonGroup
                  size="small"
                  value={viewMode}
                  exclusive
                  onChange={(_, v) => v && setViewMode(v)}
                  sx={{ borderRadius: 2 }}
                >
                  <ToggleButton value="list">
                    <ListViewIcon fontSize="small" />
                  </ToggleButton>
                  <ToggleButton value="grid">
                    <GridViewIcon fontSize="small" />
                  </ToggleButton>
                </ToggleButtonGroup>
              </Box>
            </Box>

            {/* Shared Sub-filters if in Shared View */}
            {filterType === 'shared' && (
              <Box sx={{ mt: 2, pt: 1.5, borderTop: '1px solid', borderColor: 'divider', display: 'flex', alignItems: 'center', gap: 1 }}>
                <Typography variant="body2" color="text.secondary" fontWeight={600} sx={{ mr: 1 }}>
                  Filter Shared:
                </Typography>
                <Chip
                  label={`All Shared (${sharedCount})`}
                  size="small"
                  color={sharedSubFilter === 'all' ? 'primary' : 'default'}
                  onClick={() => setSharedSubFilter('all')}
                  sx={{ fontWeight: 600, cursor: 'pointer' }}
                />
                <Chip
                  icon={<OutboundIcon sx={{ fontSize: '16px !important' }} />}
                  label={`Shared by me (${sharedByMeCount})`}
                  size="small"
                  color={sharedSubFilter === 'shared_by_me' ? 'info' : 'default'}
                  onClick={() => setSharedSubFilter('shared_by_me')}
                  sx={{ fontWeight: 600, cursor: 'pointer' }}
                />
                <Chip
                  icon={<FolderSharedIcon sx={{ fontSize: '16px !important' }} />}
                  label={`Shared with me (${sharedWithMeCount})`}
                  size="small"
                  color={sharedSubFilter === 'shared_with_me' ? 'success' : 'default'}
                  onClick={() => setSharedSubFilter('shared_with_me')}
                  sx={{ fontWeight: 600, cursor: 'pointer' }}
                />
              </Box>
            )}
          </Card>

          {/* Folder Description Banner if inside a folder */}
          {currentFolderObject && (
            <Paper
              elevation={0}
              sx={{
                p: 2,
                mb: 2.5,
                borderRadius: 2.5,
                bgcolor: currentFolderObject.is_core ? 'primary.50' : 'grey.50',
                border: '1px solid',
                borderColor: currentFolderObject.is_core ? 'primary.200' : 'divider',
                display: 'flex',
                justifyContent: 'space-between',
                alignItems: 'center',
              }}
            >
              <Box sx={{ display: 'flex', alignItems: 'center', gap: 1.5 }}>
                {currentFolderObject.is_core ? <CoreIcon color="primary" /> : <FolderIcon color="primary" />}
                <Box>
                  <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                    <Typography variant="subtitle1" fontWeight={700} color={currentFolderObject.is_core ? 'primary.900' : 'text.primary'}>
                      {currentFolderObject.name}
                    </Typography>
                    {currentFolderObject.is_core ? (
                      <Chip label="Core Standard" size="small" color="primary" sx={{ height: 20, fontSize: 11, fontWeight: 700 }} />
                    ) : (
                      <Chip label="Custom Folder" size="small" variant="outlined" sx={{ height: 20, fontSize: 11 }} />
                    )}
                  </Box>
                  {currentFolderObject.description && (
                    <Typography variant="body2" color="text.secondary" sx={{ mt: 0.3 }}>
                      {currentFolderObject.description}
                    </Typography>
                  )}
                </Box>
              </Box>

              <Button
                size="small"
                variant="text"
                onClick={() => setCurrentFolder(null)}
                sx={{ textTransform: 'none', fontWeight: 600 }}
              >
                View All Reports
              </Button>
            </Paper>
          )}

          {/* Reports Content */}
          {filteredReports.length === 0 ? (
            <Paper
              elevation={0}
              sx={{
                p: 6,
                textAlign: 'center',
                border: '1px dashed',
                borderColor: 'divider',
                borderRadius: 3,
                bgcolor: 'background.paper',
              }}
            >
              <Avatar sx={{ width: 64, height: 64, bgcolor: 'primary.50', color: 'primary.main', mx: 'auto', mb: 2 }}>
                <ReportIcon sx={{ fontSize: 32 }} />
              </Avatar>
              <Typography variant="h6" fontWeight={700} gutterBottom>
                No reports found
              </Typography>
              <Typography variant="body2" color="text.secondary" sx={{ maxWidth: 420, mx: 'auto', mb: 3 }}>
                {searchQuery
                  ? `No report matched "${searchQuery}". Try a different keyword.`
                  : filterType === 'favorites'
                  ? 'You have not favorited any reports yet. Click the star icon on any report to add it to your favorites.'
                  : filterType === 'recent'
                  ? 'No executed reports recorded yet. Run any report to see it appear in your recent history.'
                  : filterType === 'shared'
                  ? 'No reports are currently shared with the selected criteria.'
                  : currentFolderObject
                  ? `There are no reports inside "${currentFolderObject.name}". Click below to create a report or move existing reports here.`
                  : 'Get started by creating your first custom reporting template.'}
              </Typography>
              <Button
                variant="contained"
                startIcon={<AddIcon />}
                onClick={() => setNewReportModalOpen(true)}
                sx={{ textTransform: 'none', fontWeight: 600, borderRadius: 2 }}
              >
                Create New Report
              </Button>
            </Paper>
          ) : viewMode === 'list' ? (
            /* =========================================================================
               LIST VIEW
               ========================================================================= */
            <Card elevation={0} sx={{ border: '1px solid', borderColor: 'divider', borderRadius: 3, overflow: 'hidden' }}>
              <List disablePadding>
                {filteredReports.map((report, idx) => {
                  const assignedFolder = allFolders.find((f) => f.id === report.folder_id);
                  return (
                    <React.Fragment key={report.id}>
                      <ListItem
                        sx={{
                          py: 1.8,
                          px: 2.5,
                          transition: 'background-color 0.15s ease',
                          '&:hover': { bgcolor: 'action.hover' },
                        }}
                        secondaryAction={
                          <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                            <Tooltip title={report.is_favorite ? 'Remove from favorites' : 'Add to favorites'}>
                              <IconButton
                                size="small"
                                onClick={(e) => handleToggleFavorite(report.id, e)}
                                sx={{ color: report.is_favorite ? 'warning.main' : 'action.disabled' }}
                              >
                                {report.is_favorite ? <StarIcon fontSize="small" /> : <StarBorderIcon fontSize="small" />}
                              </IconButton>
                            </Tooltip>
                            <Tooltip title="Run Report">
                              <IconButton
                                size="small"
                                color="primary"
                                onClick={() => handleRunReport(report)}
                                sx={{ bgcolor: 'primary.50', '&:hover': { bgcolor: 'primary.100' } }}
                              >
                                <PlayIcon fontSize="small" />
                              </IconButton>
                            </Tooltip>
                            <IconButton size="small" onClick={(e) => handleMenuOpen(e, report)}>
                              <MoreIcon fontSize="small" />
                            </IconButton>
                          </Box>
                        }
                      >
                        <ListItemButton
                          onClick={() => handleRunReport(report)}
                          sx={{ p: 0, '&:hover': { bgcolor: 'transparent' }, mr: 10 }}
                        >
                          <ListItemIcon sx={{ minWidth: 44 }}>
                            <Avatar
                              sx={{
                                width: 38,
                                height: 38,
                                bgcolor: report.is_core ? 'primary.50' : 'secondary.50',
                                color: report.is_core ? 'primary.main' : 'secondary.main',
                              }}
                            >
                              <ReportIcon fontSize="small" />
                            </Avatar>
                          </ListItemIcon>
                          <ListItemText
                            slotProps={{ primary: { component: 'div' }, secondary: { component: 'div' } }}
                            primary={
                              <Box sx={{ display: 'flex', alignItems: 'center', gap: 1, flexWrap: 'wrap' }}>
                                <Typography variant="subtitle2" fontWeight={700} sx={{ color: 'text.primary' }}>
                                  {report.name}
                                </Typography>

                                {report.is_core && (
                                  <Chip
                                    icon={<CoreIcon sx={{ fontSize: '13px !important' }} />}
                                    label="Core"
                                    size="small"
                                    color="primary"
                                    variant="outlined"
                                    sx={{ height: 18, fontSize: 10, fontWeight: 700 }}
                                  />
                                )}

                                {assignedFolder && (
                                  <Chip
                                    icon={<FolderIcon sx={{ fontSize: '13px !important' }} />}
                                    label={assignedFolder.name}
                                    size="small"
                                    variant="filled"
                                    sx={{ height: 18, fontSize: 10, bgcolor: 'grey.100' }}
                                  />
                                )}

                                {/* DISTINCT SHARING BADGES */}
                                {report.is_shared && report.share_type === 'shared_by_me' && (
                                  <Tooltip title={`Shared by you with ${report.shared_with?.length || 0} collaborators. Click to manage.`}>
                                    <Chip
                                      icon={<OutboundIcon sx={{ fontSize: '13px !important' }} />}
                                      label={`Shared by you (${report.shared_with?.length || 0})`}
                                      size="small"
                                      color="info"
                                      onClick={(e) => {
                                        e.stopPropagation();
                                        handleOpenShareManagement(report);
                                      }}
                                      sx={{ height: 18, fontSize: 10, fontWeight: 700, cursor: 'pointer' }}
                                    />
                                  </Tooltip>
                                )}

                                {report.is_shared && report.share_type === 'shared_with_me' && (
                                  <Tooltip title={`Shared with you by ${report.shared_by || 'Colleague'}`}>
                                    <Chip
                                      icon={<FolderSharedIcon sx={{ fontSize: '13px !important' }} />}
                                      label={`Shared with you`}
                                      size="small"
                                      color="success"
                                      onClick={(e) => {
                                        e.stopPropagation();
                                        handleOpenShareManagement(report);
                                      }}
                                      sx={{ height: 18, fontSize: 10, fontWeight: 700, cursor: 'pointer' }}
                                    />
                                  </Tooltip>
                                )}
                              </Box>
                            }
                            secondary={
                              <Box sx={{ mt: 0.5 }}>
                                {report.description && (
                                  <Typography variant="body2" color="text.secondary" noWrap sx={{ maxWidth: 650, mb: 0.4 }}>
                                    {report.description}
                                  </Typography>
                                )}
                                <Stack direction="row" spacing={2} sx={{ color: 'text.disabled', fontSize: 12 }}>
                                  <span>{report.run_count} total runs</span>
                                  <span>•</span>
                                  <span>Last run: {report.last_run ? `${formatDistanceToNow(new Date(report.last_run))} ago` : 'Never'}</span>
                                  <span>•</span>
                                  <span>
                                    {report.share_type === 'shared_with_me'
                                      ? `From: ${report.shared_by || report.created_by}`
                                      : `By ${report.created_by}`}
                                  </span>
                                </Stack>
                              </Box>
                            }
                          />
                        </ListItemButton>
                      </ListItem>
                      {idx < filteredReports.length - 1 && <Divider />}
                    </React.Fragment>
                  );
                })}
              </List>
            </Card>
          ) : (
            /* =========================================================================
               GRID VIEW
               ========================================================================= */
            <Grid container spacing={2.5}>
              {filteredReports.map((report) => {
                const assignedFolder = allFolders.find((f) => f.id === report.folder_id);
                return (
                  <Grid key={report.id} size={{ xs: 12, sm: 6, lg: 4 }}>
                    <Card
                      elevation={0}
                      sx={{
                        height: '100%',
                        display: 'flex',
                        flexDirection: 'column',
                        border: '1px solid',
                        borderColor: 'divider',
                        borderRadius: 3,
                        transition: 'all 0.2s ease',
                        '&:hover': {
                          borderColor: 'primary.main',
                          boxShadow: '0 4px 16px rgba(0,0,0,0.06)',
                        },
                      }}
                    >
                      <CardContent sx={{ p: 2.5, flex: 1, display: 'flex', flexDirection: 'column' }}>
                        {/* Top Icon & Quick Actions */}
                        <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start', mb: 2 }}>
                          <Avatar
                            sx={{
                              bgcolor: report.is_core ? 'primary.50' : 'secondary.50',
                              color: report.is_core ? 'primary.main' : 'secondary.main',
                              width: 42,
                              height: 42,
                            }}
                          >
                            <ReportIcon />
                          </Avatar>
                          <Box sx={{ display: 'flex', alignItems: 'center' }}>
                            <Tooltip title={report.is_favorite ? 'Remove from favorites' : 'Add to favorites'}>
                              <IconButton
                                size="small"
                                onClick={(e) => handleToggleFavorite(report.id, e)}
                                sx={{ color: report.is_favorite ? 'warning.main' : 'action.disabled' }}
                              >
                                {report.is_favorite ? <StarIcon fontSize="small" /> : <StarBorderIcon fontSize="small" />}
                              </IconButton>
                            </Tooltip>
                            <IconButton size="small" onClick={(e) => handleMenuOpen(e, report)}>
                              <MoreIcon fontSize="small" />
                            </IconButton>
                          </Box>
                        </Box>

                        {/* Title & Badges */}
                        <Typography variant="subtitle1" fontWeight={700} noWrap gutterBottom title={report.name}>
                          {report.name}
                        </Typography>

                        <Box sx={{ display: 'flex', alignItems: 'center', gap: 0.8, mb: 1.5, flexWrap: 'wrap' }}>
                          {report.is_core && (
                            <Chip
                              icon={<CoreIcon sx={{ fontSize: '13px !important' }} />}
                              label="Core"
                              size="small"
                              color="primary"
                              variant="outlined"
                              sx={{ height: 18, fontSize: 10, fontWeight: 700 }}
                            />
                          )}

                          {assignedFolder && (
                            <Chip
                              icon={<FolderIcon sx={{ fontSize: '13px !important' }} />}
                              label={assignedFolder.name}
                              size="small"
                              variant="filled"
                              sx={{ height: 18, fontSize: 10, bgcolor: 'grey.100' }}
                            />
                          )}

                          {/* DISTINCT SHARING BADGES */}
                          {report.is_shared && report.share_type === 'shared_by_me' && (
                            <Chip
                              icon={<OutboundIcon sx={{ fontSize: '13px !important' }} />}
                              label="Shared by you"
                              size="small"
                              color="info"
                              onClick={(e) => {
                                e.stopPropagation();
                                handleOpenShareManagement(report);
                              }}
                              sx={{ height: 18, fontSize: 10, fontWeight: 700, cursor: 'pointer' }}
                            />
                          )}

                          {report.is_shared && report.share_type === 'shared_with_me' && (
                            <Chip
                              icon={<FolderSharedIcon sx={{ fontSize: '13px !important' }} />}
                              label="Shared with you"
                              size="small"
                              color="success"
                              onClick={(e) => {
                                e.stopPropagation();
                                handleOpenShareManagement(report);
                              }}
                              sx={{ height: 18, fontSize: 10, fontWeight: 700, cursor: 'pointer' }}
                            />
                          )}
                        </Box>

                        {/* Description */}
                        <Typography
                          variant="body2"
                          color="text.secondary"
                          sx={{
                            mb: 2,
                            flex: 1,
                            display: '-webkit-box',
                            WebkitLineClamp: 2,
                            WebkitBoxOrient: 'vertical',
                            overflow: 'hidden',
                            minHeight: 40,
                          }}
                        >
                          {report.description || 'No description provided.'}
                        </Typography>

                        <Divider sx={{ my: 1.5 }} />

                        {/* Card Footer */}
                        <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                          <Box>
                            <Typography variant="caption" color="text.secondary" display="block">
                              Last run: {report.last_run ? `${formatDistanceToNow(new Date(report.last_run))} ago` : 'Never'}
                            </Typography>
                            <Typography variant="caption" color="text.disabled">
                              {report.run_count} executions • By {report.created_by}
                            </Typography>
                          </Box>
                          <Button
                            size="small"
                            variant="contained"
                            startIcon={<PlayIcon />}
                            onClick={() => handleRunReport(report)}
                            sx={{ textTransform: 'none', fontWeight: 600, borderRadius: 1.5 }}
                          >
                            Run
                          </Button>
                        </Box>
                      </CardContent>
                    </Card>
                  </Grid>
                );
              })}
            </Grid>
          )}
        </Grid>
      </Grid>

      {/* =========================================================================
          REPORT ACTION CONTEXT MENU
          ========================================================================= */}
      <Menu
        anchorEl={menuAnchor}
        open={Boolean(menuAnchor)}
        onClose={handleMenuClose}
        transformOrigin={{ horizontal: 'right', vertical: 'top' }}
        anchorOrigin={{ horizontal: 'right', vertical: 'bottom' }}
        PaperProps={{
          elevation: 3,
          sx: { minWidth: 200, borderRadius: 2 },
        }}
      >
        <MenuItem onClick={() => selectedReport && handleRunReport(selectedReport)}>
          <ListItemIcon>
            <PlayIcon fontSize="small" color="primary" />
          </ListItemIcon>
          Run Report
        </MenuItem>

        <MenuItem onClick={() => selectedReport && handleToggleFavorite(selectedReport.id)}>
          <ListItemIcon>
            {selectedReport?.is_favorite ? <StarBorderIcon fontSize="small" /> : <StarIcon fontSize="small" color="warning" />}
          </ListItemIcon>
          {selectedReport?.is_favorite ? 'Remove from Favorites' : 'Add to Favorites'}
        </MenuItem>

        <MenuItem
          onClick={() => selectedReport && handleOpenMoveDialog(selectedReport)}
          sx={{ display: selectedReport?.share_type === 'shared_with_me' ? 'none' : 'flex' }}
        >
          <ListItemIcon>
            <MoveIcon fontSize="small" />
          </ListItemIcon>
          Move to Folder...
        </MenuItem>

        <Divider />

        <MenuItem onClick={() => selectedReport && handleOpenShareManagement(selectedReport)}>
          <ListItemIcon>
            {selectedReport?.share_type === 'shared_with_me' ? (
              <FolderSharedIcon fontSize="small" color="success" />
            ) : (
              <ShareIcon fontSize="small" color="info" />
            )}
          </ListItemIcon>
          {selectedReport?.share_type === 'shared_with_me'
            ? 'View Share Details...'
            : selectedReport?.is_shared
            ? 'Manage Sharing...'
            : 'Share Report...'}
        </MenuItem>

        <MenuItem
          onClick={() => selectedReport && handleEditReport(selectedReport)}
          sx={{ display: selectedReport?.share_type === 'shared_with_me' ? 'none' : 'flex' }}
        >
          <ListItemIcon>
            <EditIcon fontSize="small" />
          </ListItemIcon>
          Edit Definition
        </MenuItem>

        <MenuItem
          onClick={() => selectedReport && handleDuplicateReport(selectedReport)}
          sx={{ display: selectedReport?.share_type === 'shared_with_me' ? 'none' : 'flex' }}
        >
          <ListItemIcon>
            <DuplicateIcon fontSize="small" />
          </ListItemIcon>
          Clone
        </MenuItem>

        <Divider />

        <MenuItem
          onClick={() => {
            handleMenuClose();
            setDeleteReportConfirmOpen(true);
          }}
          sx={{ color: 'error.main', display: selectedReport?.share_type === 'shared_with_me' ? 'none' : 'flex' }}
        >
          <ListItemIcon>
            <DeleteIcon fontSize="small" color="error" />
          </ListItemIcon>
          Delete Report
        </MenuItem>
      </Menu>

      {/* =========================================================================
          FOLDER MENU (Custom Folders Edit / Delete)
          ========================================================================= */}
      <Menu
        anchorEl={folderMenuAnchor}
        open={Boolean(folderMenuAnchor)}
        onClose={() => setFolderMenuAnchor(null)}
        PaperProps={{ elevation: 3, sx: { minWidth: 160, borderRadius: 2 } }}
      >
        <MenuItem onClick={() => activeFolderMenu && handleOpenEditFolderDialog(activeFolderMenu)}>
          <ListItemIcon>
            <EditIcon fontSize="small" />
          </ListItemIcon>
          Rename / Edit
        </MenuItem>
        <MenuItem
          onClick={() => activeFolderMenu && handleOpenDeleteFolderConfirm(activeFolderMenu)}
          sx={{ color: 'error.main' }}
        >
          <ListItemIcon>
            <DeleteIcon fontSize="small" color="error" />
          </ListItemIcon>
          Delete Folder
        </MenuItem>
      </Menu>

      {/* =========================================================================
          NEW / EDIT FOLDER DIALOG
          ========================================================================= */}
      <Dialog
        open={folderDialogOpen}
        onClose={() => setFolderDialogOpen(false)}
        maxWidth="xs"
        fullWidth
        PaperProps={{ sx: { borderRadius: 3 } }}
      >
        <DialogTitle sx={{ fontWeight: 700 }}>
          {folderDialogMode === 'create' ? 'Create New Folder' : 'Edit Folder'}
        </DialogTitle>
        <DialogContent>
          <DialogContentText sx={{ mb: 2 }}>
            Organize your report templates into custom folders for quick access.
          </DialogContentText>
          <TextField
            autoFocus
            label="Folder Name"
            fullWidth
            required
            variant="outlined"
            value={folderName}
            onChange={(e) => setFolderName(e.target.value)}
            sx={{ mb: 2 }}
          />
          <TextField
            label="Description (Optional)"
            fullWidth
            multiline
            rows={2}
            variant="outlined"
            value={folderDescription}
            onChange={(e) => setFolderDescription(e.target.value)}
          />
        </DialogContent>
        <DialogActions sx={{ px: 3, pb: 2.5 }}>
          <Button onClick={() => setFolderDialogOpen(false)} sx={{ textTransform: 'none' }}>
            Cancel
          </Button>
          <Button
            variant="contained"
            onClick={handleSaveFolder}
            disabled={!folderName.trim()}
            sx={{ textTransform: 'none', fontWeight: 600, borderRadius: 2 }}
          >
            {folderDialogMode === 'create' ? 'Create Folder' : 'Save Changes'}
          </Button>
        </DialogActions>
      </Dialog>

      {/* =========================================================================
          DELETE FOLDER CONFIRMATION DIALOG
          ========================================================================= */}
      <Dialog
        open={deleteFolderDialogOpen}
        onClose={() => setDeleteFolderDialogOpen(false)}
        maxWidth="xs"
        fullWidth
        PaperProps={{ sx: { borderRadius: 3 } }}
      >
        <DialogTitle sx={{ fontWeight: 700, color: 'error.main' }}>
          Delete Folder?
        </DialogTitle>
        <DialogContent>
          <DialogContentText>
            Are you sure you want to delete folder <strong>"{folderToDelete?.name}"</strong>?
            <br />
            Reports contained inside will not be deleted; they will be moved to the root level.
          </DialogContentText>
        </DialogContent>
        <DialogActions sx={{ px: 3, pb: 2.5 }}>
          <Button onClick={() => setDeleteFolderDialogOpen(false)} sx={{ textTransform: 'none' }}>
            Cancel
          </Button>
          <Button
            variant="contained"
            color="error"
            onClick={handleDeleteFolderConfirm}
            sx={{ textTransform: 'none', fontWeight: 600, borderRadius: 2 }}
          >
            Delete Folder
          </Button>
        </DialogActions>
      </Dialog>

      {/* =========================================================================
          MOVE REPORT TO FOLDER DIALOG
          ========================================================================= */}
      <Dialog
        open={moveToFolderDialogOpen}
        onClose={() => setMoveToFolderDialogOpen(false)}
        maxWidth="xs"
        fullWidth
        PaperProps={{ sx: { borderRadius: 3 } }}
      >
        <DialogTitle sx={{ fontWeight: 700 }}>
          Move Report to Folder
        </DialogTitle>
        <DialogContent>
          <DialogContentText sx={{ mb: 2 }}>
            Select destination folder for <strong>"{selectedReport?.name}"</strong>:
          </DialogContentText>
          <FormControl fullWidth>
            <InputLabel>Destination Folder</InputLabel>
            <Select
              value={targetFolderId}
              label="Destination Folder"
              onChange={(e) => setTargetFolderId(e.target.value)}
            >
              <MenuItem value="none">
                <em>No Folder (Root Level)</em>
              </MenuItem>
              <Divider />
              <Typography variant="overline" sx={{ px: 2, color: 'text.secondary', display: 'block', pt: 1 }}>
                Core Folders
              </Typography>
              {coreFolders.map((f) => (
                <MenuItem key={f.id} value={f.id}>
                  <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                    <CoreIcon fontSize="small" color="primary" />
                    <span>{f.name}</span>
                  </Box>
                </MenuItem>
              ))}
              <Divider />
              <Typography variant="overline" sx={{ px: 2, color: 'text.secondary', display: 'block', pt: 1 }}>
                My Custom Folders
              </Typography>
              {userFolders.map((f) => (
                <MenuItem key={f.id} value={f.id}>
                  <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                    <FolderIcon fontSize="small" color="action" />
                    <span>{f.name}</span>
                  </Box>
                </MenuItem>
              ))}
            </Select>
          </FormControl>
        </DialogContent>
        <DialogActions sx={{ px: 3, pb: 2.5 }}>
          <Button onClick={() => setMoveToFolderDialogOpen(false)} sx={{ textTransform: 'none' }}>
            Cancel
          </Button>
          <Button
            variant="contained"
            onClick={handleSaveMoveToFolder}
            sx={{ textTransform: 'none', fontWeight: 600, borderRadius: 2 }}
          >
            Move Report
          </Button>
        </DialogActions>
      </Dialog>

      {/* =========================================================================
          MANAGE SHARING DIALOG (Shared by me)
          ========================================================================= */}
      <Dialog
        open={manageShareDialogOpen}
        onClose={() => setManageShareDialogOpen(false)}
        maxWidth="sm"
        fullWidth
        PaperProps={{ sx: { borderRadius: 3 } }}
      >
        <DialogTitle sx={{ fontWeight: 700, display: 'flex', justifyContent: 'space-between', alignItems: 'center', flexDirection: 'column', gap: 1 }}>
          <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', width: '100%' }}>
            <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
              <OutboundIcon color="info" />
              <span>Manage Sharing: {selectedReport?.name}</span>
            </Box>
            <IconButton size="small" onClick={() => setManageShareDialogOpen(false)}>
              <CloseIcon fontSize="small" />
            </IconButton>
          </Box>
          <Tabs
            value={shareDialogTab}
            onChange={(_, v) => setShareDialogTab(v)}
            sx={{ minHeight: 36, '& .MuiTab-root': { minHeight: 36, py: 0.5 } }}
          >
            <Tab value="people" label="People & Access" />
            <Tab value="activity" label="Activity" />
          </Tabs>
        </DialogTitle>
        <DialogContent dividers>
          {shareDialogTab === 'people' && (
            <Box sx={{ display: 'flex', flexDirection: 'column', gap: 2 }}>
              <Box>
                <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', mb: 1 }}>
                  <Typography variant="subtitle2" fontWeight={700}>
                    Add People &amp; Collaborators
                  </Typography>
                  <ToggleButtonGroup
                    size="small"
                    value={shareMode}
                    exclusive
                    onChange={(_, v) => { if (v) setShareMode(v); }}
                    sx={{ '& .MuiToggleButton-root': { py: 0.25, px: 1.5, fontSize: 12, textTransform: 'none' } }}
                  >
                    <ToggleButton value="people">People</ToggleButton>
                    <ToggleButton value="team">Team / Role</ToggleButton>
                  </ToggleButtonGroup>
                </Box>
                {shareMode === 'people' ? (
                  <Grid container spacing={1.5} alignItems="center">
                    <Grid size={{ xs: 12, sm: 9 }}>
                      <Autocomplete
                        size="small"
                        options={shareableUsers}
                        loading={shareableUsersLoading}
                        getOptionLabel={(option) => `${option.name} (${option.email})`}
                        groupBy={(option) => option.access_path === 'entitlement' ? 'Entitlement Access' : 'Direct Access'}
                        isOptionEqualToValue={(option, value) => option.id === value.id}
                        value={newCollaboratorUser}
                        onChange={(_, value) => setNewCollaboratorUser(value)}
                        renderOption={(props, option) => (
                          <Box component="li" {...props} sx={{ display: 'flex', flexDirection: 'column', alignItems: 'flex-start', gap: 0 }}>
                            <Typography variant="body2" fontWeight={600}>
                              {option.name}
                              {!option.is_active && (
                                <Chip label="Inactive" size="small" color="warning" sx={{ ml: 1, height: 16, fontSize: 10 }} />
                              )}
                            </Typography>
                            <Typography variant="caption" color="text.secondary">
                              {option.email} · {option.role} · {option.organization}
                              {option.access_path === 'entitlement' && (
                                <Chip label="Entitlement" size="small" sx={{ ml: 0.5, height: 14, fontSize: 9 }} />
                              )}
                            </Typography>
                          </Box>
                        )}
                        renderInput={(params) => (
                          <TextField
                            {...params}
                            placeholder="Search by name or email..."
                            InputProps={{
                              ...params.InputProps,
                              endAdornment: (
                                <>
                                  {shareableUsersLoading && <CircularProgress size={14} />}
                                  {params.InputProps.endAdornment}
                                </>
                              ),
                            }}
                          />
                        )}
                      />
                    </Grid>
                    <Grid size={{ xs: 12, sm: 3 }}>
                      <Button
                        variant="contained"
                        fullWidth
                        disabled={!newCollaboratorUser}
                        onClick={handleAddCollaborator}
                        startIcon={<PersonAddIcon />}
                        sx={{ height: 40, textTransform: 'none', fontWeight: 600 }}
                      >
                        Share
                      </Button>
                    </Grid>
                  </Grid>
                ) : (
                  <>
                    <Grid container spacing={1.5} alignItems="center">
                      <Grid size={{ xs: 12, sm: 9 }}>
                        <FormControl size="small" fullWidth>
                          <InputLabel>Select team or role...</InputLabel>
                          <Select
                            label="Select team or role..."
                            onChange={(e) => {
                              const selected = TEAM_ROLE_OPTIONS.find((t) => t.id === e.target.value);
                              if (selected) {
                                setNewCollaboratorUser({
                                  id: selected.id,
                                  name: selected.name,
                                  email: selected.id,
                                  role: selected.type,
                                  organization: 'alpha-wealth',
                                  access_path: 'direct',
                                  is_active: true,
                                });
                              }
                            }}
                          >
                            {TEAM_ROLE_OPTIONS.map((t) => (
                              <MenuItem key={t.id} value={t.id}>
                                <Box sx={{ display: 'flex', flexDirection: 'column' }}>
                                  <Typography variant="body2" fontWeight={600}>{t.name}</Typography>
                                  <Typography variant="caption" color="text.secondary">{t.description}</Typography>
                                </Box>
                              </MenuItem>
                            ))}
                          </Select>
                        </FormControl>
                      </Grid>
                      <Grid size={{ xs: 12, sm: 3 }}>
                        <Button
                          variant="contained"
                          fullWidth
                          disabled={!newCollaboratorUser}
                          onClick={handleAddCollaborator}
                          startIcon={<GroupIcon />}
                          sx={{ height: 40, textTransform: 'none', fontWeight: 600 }}
                        >
                          Share
                        </Button>
                      </Grid>
                    </Grid>
                    <Typography variant="caption" color="text.secondary" sx={{ mt: 0.5, display: 'block' }}>
                      Sharing with a role grants access to all current and future members of that role.
                    </Typography>
                  </>
                )}
              </Box>
              <Typography variant="subtitle2" fontWeight={700} gutterBottom>
                Who Has Access ({shares.length})
              </Typography>
              {sharesLoading ? (
                <Box sx={{ display: 'flex', justifyContent: 'center', py: 3 }}>
                  <CircularProgress size={24} />
                </Box>
              ) : (
                <TableContainer component={Paper} variant="outlined" sx={{ borderRadius: 2 }}>
                  <Table size="small">
                    <TableHead sx={{ bgcolor: 'grey.50' }}>
                      <TableRow>
                        <TableCell sx={{ fontWeight: 600 }}>User</TableCell>
                        <TableCell sx={{ fontWeight: 600 }}>Access</TableCell>
                        <TableCell sx={{ fontWeight: 600 }}>Role</TableCell>
                        <TableCell align="right" sx={{ fontWeight: 600 }}>Action</TableCell>
                      </TableRow>
                    </TableHead>
                    <TableBody>
                      <TableRow>
                        <TableCell>
                          <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                            <Avatar sx={{ width: 28, height: 28, bgcolor: 'primary.main', fontSize: 12 }}>You</Avatar>
                            <Box>
                              <Typography variant="body2" fontWeight={600}>You (Owner)</Typography>
                              <Typography variant="caption" color="text.secondary">Creator of this report</Typography>
                            </Box>
                          </Box>
                        </TableCell>
                        <TableCell>
                          <Chip label="Owner" size="small" color="primary" sx={{ height: 20, fontSize: 11 }} />
                        </TableCell>
                        <TableCell>
                          <Typography variant="caption" color="text.secondary">—</Typography>
                        </TableCell>
                        <TableCell align="right">—</TableCell>
                      </TableRow>
                      {shares.map((share) => (
                        <React.Fragment key={share.id}>
                          {!share.is_active && (
                            <TableRow>
                              <TableCell colSpan={4} sx={{ py: 0, borderBottom: 'none' }}>
                                <Alert severity="warning" sx={{ py: 0.5, borderRadius: 1 }}>
                                  <AlertTitle sx={{ fontSize: 11, my: 0 }}>Inactive User</AlertTitle>
                                  {share.recipient_name} ({share.recipient_email}) is inactive. Their shared access is also inactive.
                                </Alert>
                              </TableCell>
                            </TableRow>
                          )}
                          <TableRow>
                            <TableCell>
                              <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                                <Avatar sx={{ width: 28, height: 28, bgcolor: share.is_active ? 'grey.300' : 'grey.200', fontSize: 12, color: 'text.primary', opacity: share.is_active ? 1 : 0.5 }}>
                                  {share.recipient_name.charAt(0)}
                                </Avatar>
                                <Box>
                                  <Box sx={{ display: 'flex', alignItems: 'center', gap: 0.5 }}>
                                    <Typography variant="body2" fontWeight={600} sx={{ opacity: share.is_active ? 1 : 0.5 }}>{share.recipient_name}</Typography>
                                    {share.recipient_organization && share.recipient_organization !== 'alpha-wealth' && (
                                      <Chip label={share.recipient_organization} size="small" sx={{ height: 16, fontSize: 10 }} />
                                    )}
                                  </Box>
                                  <Typography variant="caption" color="text.secondary" sx={{ opacity: share.is_active ? 1 : 0.5 }}>{share.recipient_email}</Typography>
                                </Box>
                              </Box>
                            </TableCell>
                            <TableCell>
                              <Chip
                                label={share.access_path === 'entitlement' ? 'Entitlement' : 'Direct'}
                                size="small"
                                sx={{ height: 20, fontSize: 10, bgcolor: share.access_path === 'entitlement' ? 'purple.50' : 'grey.50', color: share.access_path === 'entitlement' ? 'purple.main' : 'text.secondary' }}
                              />
                            </TableCell>
                            <TableCell>
                              <Chip
                                label={share.permission === 'view' ? 'Can View' : share.permission === 'comment' ? 'Can Comment' : share.permission}
                                size="small"
                                color="default"
                                sx={{ height: 20, fontSize: 11 }}
                              />
                            </TableCell>
                            <TableCell align="right">
                              <Box sx={{ display: 'flex', gap: 0.5, justifyContent: 'flex-end', alignItems: 'center' }}>
                                <Tooltip title={share.watermark ? 'Watermark ON — click to disable' : 'Watermark OFF — click to enable'}>
                                  <IconButton
                                    size="small"
                                    color={share.watermark ? 'info' : 'default'}
                                    onClick={async () => {
                                      try {
                                        await updateShare(share.recipient_id, { watermark: !share.watermark });
                                        await fetchShares();
                                      } catch (err: any) {
                                        showNotification(`Failed to update: ${err.message}`, 'error');
                                      }
                                    }}
                                  >
                                    <AutoAwesomeIcon fontSize="small" />
                                  </IconButton>
                                </Tooltip>
                                <Tooltip title={share.is_suspended ? 'Resume access' : 'Suspend access (temp)'}>
                                  <IconButton
                                    size="small"
                                    color={share.is_suspended ? 'warning' : 'default'}
                                    onClick={async () => {
                                      try {
                                        await updateShare(share.recipient_id, { suspend: !share.is_suspended });
                                        await fetchShares();
                                      } catch (err: any) {
                                        showNotification(`Failed to update: ${err.message}`, 'error');
                                      }
                                    }}
                                  >
                                    {share.is_suspended ? <PlayIcon fontSize="small" /> : <BlockIcon fontSize="small" />}
                                  </IconButton>
                                </Tooltip>
                                <Tooltip title={share.is_active ? 'Remove access' : 'Remove inactive user'}>
                                  <IconButton size="small" color="error" onClick={() => handleRemoveCollaborator(share.recipient_id)}>
                                    <DeleteIcon fontSize="small" />
                                  </IconButton>
                                </Tooltip>
                              </Box>
                            </TableCell>
                          </TableRow>
                        </React.Fragment>
                      ))}
                    </TableBody>
                  </Table>
                </TableContainer>
              )}
            </Box>
          )}
          {shareDialogTab === 'activity' && (
            <Box sx={{ py: 1 }}>
              <Typography variant="body2" color="text.secondary" sx={{ mb: 2 }}>
                Share activity is tracked in the audit log. Actions include share creation, revocation, suspension, and report cloning.
              </Typography>
              <Alert severity="info" sx={{ borderRadius: 2 }}>
                Activity audit log is available in the system administration panel. Go to <strong>Admin &gt; Access Control &gt; Audit Logs</strong> and filter by <strong>Report Sharing</strong> to see a full history of who shared what with whom and when.
              </Alert>
            </Box>
          )}
        </DialogContent>

        <DialogActions sx={{ px: 3, py: 2, display: 'flex', justifyContent: 'space-between' }}>
          <Button
            variant="outlined"
            color="error"
            startIcon={<BlockIcon />}
            onClick={() => setStopShareConfirmOpen(true)}
            sx={{ textTransform: 'none', fontWeight: 600 }}
          >
            Stop Sharing Report
          </Button>

          <Button
            variant="contained"
            onClick={() => setManageShareDialogOpen(false)}
            sx={{ textTransform: 'none', fontWeight: 600, borderRadius: 2 }}
          >
            Done
          </Button>
        </DialogActions>
      </Dialog>

      {/* =========================================================================
          STOP SHARING CONFIRMATION DIALOG
          ========================================================================= */}
      <Dialog
        open={stopShareConfirmOpen}
        onClose={() => setStopShareConfirmOpen(false)}
        maxWidth="xs"
        fullWidth
        PaperProps={{ sx: { borderRadius: 3 } }}
      >
        <DialogTitle sx={{ fontWeight: 700, color: 'error.main' }}>
          Stop Sharing This Report?
        </DialogTitle>
        <DialogContent>
          <DialogContentText>
            This will immediately revoke access for all <strong>{selectedReport?.shared_with?.length || 0}</strong> collaborators.
            The report will be reverted to <strong>Private</strong>.
          </DialogContentText>
        </DialogContent>
        <DialogActions sx={{ px: 3, pb: 2.5 }}>
          <Button onClick={() => setStopShareConfirmOpen(false)} sx={{ textTransform: 'none' }}>
            Cancel
          </Button>
          <Button
            variant="contained"
            color="error"
            onClick={handleStopSharing}
            sx={{ textTransform: 'none', fontWeight: 600, borderRadius: 2 }}
          >
            Confirm Stop Sharing
          </Button>
        </DialogActions>
      </Dialog>

      {/* =========================================================================
          SHARED WITH ME DETAILS DIALOG
          ========================================================================= */}
      <Dialog
        open={sharedWithMeDialogOpen}
        onClose={() => setSharedWithMeDialogOpen(false)}
        maxWidth="xs"
        fullWidth
        PaperProps={{ sx: { borderRadius: 3 } }}
      >
        <DialogTitle sx={{ fontWeight: 700, display: 'flex', alignItems: 'center', gap: 1 }}>
          <FolderSharedIcon color="success" />
          <span>Shared Report Details</span>
        </DialogTitle>
        <DialogContent>
          <Box sx={{ mb: 2 }}>
            <Typography variant="subtitle1" fontWeight={700}>
              {selectedReport?.name}
            </Typography>
            <Typography variant="body2" color="text.secondary" sx={{ mt: 0.5 }}>
              {selectedReport?.description}
            </Typography>
          </Box>
          <Paper variant="outlined" sx={{ p: 2, borderRadius: 2, mb: 2, bgcolor: 'grey.50' }}>
            <Typography variant="caption" color="text.secondary" display="block">
              SHARED BY
            </Typography>
            <Typography variant="body2" fontWeight={600} sx={{ mt: 0.2 }}>
              {selectedReport?.shared_by || selectedReport?.created_by || 'Colleague'}
            </Typography>
            {selectedReport?.shared_by_email && (
              <Typography variant="caption" color="text.secondary">
                {selectedReport?.shared_by_email}
              </Typography>
            )}
            <Divider sx={{ my: 1.5 }} />
            <Typography variant="caption" color="text.secondary" display="block">
              YOUR ACCESS LEVEL
            </Typography>
            <Chip label="Viewer (Can Run & Export)" size="small" color="success" sx={{ height: 22, fontSize: 11, mt: 0.5 }} />
          </Paper>
        </DialogContent>
        <DialogActions sx={{ px: 3, pb: 2.5, display: 'flex', justifyContent: 'space-between' }}>
          <Box>
            <Button
              color="error"
              onClick={handleRemoveSharedWithMe}
              sx={{ textTransform: 'none', fontWeight: 600 }}
            >
              Remove from My Shared
            </Button>
          </Box>
          <Box sx={{ display: 'flex', gap: 1 }}>
            <Button
              variant="outlined"
              startIcon={<DuplicateIcon />}
              onClick={handleCloneAndEdit}
              sx={{ textTransform: 'none', fontWeight: 600, borderRadius: 2 }}
            >
              Clone &amp; Edit
            </Button>
            <Button
              variant="contained"
              onClick={() => setSharedWithMeDialogOpen(false)}
              sx={{ textTransform: 'none', fontWeight: 600, borderRadius: 2 }}
            >
              Close
            </Button>
          </Box>
        </DialogActions>
      </Dialog>

      {/* =========================================================================
          DELETE REPORT CONFIRMATION DIALOG
          ========================================================================= */}
      <Dialog
        open={deleteReportConfirmOpen}
        onClose={() => setDeleteReportConfirmOpen(false)}
        maxWidth="xs"
        fullWidth
        PaperProps={{ sx: { borderRadius: 3 } }}
      >
        <DialogTitle sx={{ fontWeight: 700, color: 'error.main' }}>
          Delete Report?
        </DialogTitle>
        <DialogContent>
          <DialogContentText>
            Are you sure you want to delete <strong>"{selectedReport?.name}"</strong>?
            This action cannot be undone.
          </DialogContentText>
        </DialogContent>
        <DialogActions sx={{ px: 3, pb: 2.5 }}>
          <Button onClick={() => setDeleteReportConfirmOpen(false)} sx={{ textTransform: 'none' }}>
            Cancel
          </Button>
          <Button
            variant="contained"
            color="error"
            onClick={handleDeleteReportConfirm}
            sx={{ textTransform: 'none', fontWeight: 600, borderRadius: 2 }}
          >
            Delete
          </Button>
        </DialogActions>
      </Dialog>

      {/* =========================================================================
          UNIFIED BUSINESS OBJECT PICKER FOR NEW REPORTS
          ========================================================================= */}
      <UnifiedBOPickerModal
        open={newReportModalOpen}
        context="report"
        onClose={() => setNewReportModalOpen(false)}
        onPick={(bo, bindingId, _selectedRelatedBOs, _bindingDetails, selectedSubtypeKey) => {
          setNewReportModalOpen(false);
          navigate(`/reports/builder?bo=${encodeURIComponent(bo.id)}&binding=${encodeURIComponent(bindingId || '')}`, {
            state: { boId: bo.id, bindingId, bo, selectedSubtypeKey },
          });
        }}
      />

      {/* =========================================================================
          TOAST SNACKBAR NOTIFICATION
          ========================================================================= */}
      <Snackbar
        open={Boolean(snackbarMessage)}
        autoHideDuration={3500}
        onClose={() => setSnackbarMessage(null)}
        anchorOrigin={{ vertical: 'bottom', horizontal: 'right' }}
      >
        <Alert
          onClose={() => setSnackbarMessage(null)}
          severity={snackbarSeverity}
          variant="filled"
          sx={{ borderRadius: 2, fontWeight: 600 }}
        >
          {snackbarMessage}
        </Alert>
      </Snackbar>
    </Box>
  );
};