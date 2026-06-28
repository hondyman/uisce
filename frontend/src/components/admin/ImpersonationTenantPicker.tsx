/**
 * ImpersonationTenantPicker
 *
 * A two-pane picker for the Global Admin → Tenant Impersonation flow.
 *
 *   ┌──────────────────────────────────────────────────────────────────┐
 *   │  RECENT SESSIONS  (last 5)                                     │
 *   ├─────────────────────────┬────────────────────────────────────────┤
 *   │  Tenant search list     │  Scope tree (instance → product →     │
 *   │  (debounced, paginated)│  datasource) for the selected tenant   │
 *   └─────────────────────────┴────────────────────────────────────────┘
 *
 * On selection, emits an ImpersonationScope (kind + id) which the parent
 * passes to AssumeContextParams.scope.
 *
 * NOTE: This picker reuses the SCOPE NAVIGATION UI PATTERN from
 * ScopeSelectorDialog (breadcrumbs + List of instances/products/datasources)
 * but is a separate, focused component because:
 *   1. The audience is different: GLOBAL ADMIN picking any tenant, not a
 *      regular user picking among their accessible tenants.
 *   2. The data source is different: /api/admin/tenants/search (admin-only)
 *      vs /api/tenants/accessible (user-scoped).
 *   3. The output is different: an ImpersonationScope that gets audited,
 *      not a change to the operating scope in AccessContext.
 */

import React, { useState, useEffect, useMemo, useCallback, useRef } from 'react';
import {
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  Button,
  Box,
  Typography,
  TextField,
  List,
  ListItemButton,
  ListItemIcon,
  ListItemText,
  IconButton,
  Breadcrumbs,
  Link,
  Chip,
  InputAdornment,
  CircularProgress,
  Alert,
} from '@mui/material';
import {
  Search as SearchIcon,
  Business as BusinessIcon,
  Dns as InstanceIcon,
  Inventory as ProductIcon,
  Storage as DatasourceIcon,
  ChevronRight as ChevronRightIcon,
  History as HistoryIcon,
  Clear as ClearIcon,
} from '@mui/icons-material';
import {
  type ImpersonationScope,
  type ImpersonationScopeKind,
  type RecentSession,
  ScopeTenant,
  ScopeInstance,
  ScopeProduct,
  ScopeDatasource,
} from '../../contexts/ImpersonationContext';

const API_BASE = import.meta.env.VITE_API_URL ?? '/api';
const DEBOUNCE_MS = 250;
const DEFAULT_SEARCH_LIMIT = 20;

// -----------------------------------------------------------------------------
// Backend types matching /api/admin/tenants/search and /scope
// -----------------------------------------------------------------------------

interface TenantSearchRow {
  id: string;
  name: string;
  code?: string;
  region?: string;
  plan?: string;
  is_suspended: boolean;
  instance_count: number;
}

interface ScopeNode {
  id: string;
  name: string;
  type: 'instance' | 'product' | 'datasource';
  children?: ScopeNode[];
}

interface TenantScopeResponse {
  tenant_id: string;
  instances: ScopeNode[];
}

// -----------------------------------------------------------------------------
// API helpers
// -----------------------------------------------------------------------------

async function searchTenants(
  query: string,
  token: string,
  signal: AbortSignal,
): Promise<TenantSearchRow[]> {
  const url = new URL(`${API_BASE}/admin/tenants/search`, window.location.origin);
  if (query) url.searchParams.set('q', query);
  url.searchParams.set('limit', String(DEFAULT_SEARCH_LIMIT));
  const resp = await fetch(url.toString(), {
    headers: { Authorization: `Bearer ${token}` },
    signal,
  });
  if (!resp.ok) throw new Error(`search failed (${resp.status})`);
  const data = await resp.json();
  return data.results ?? [];
}

async function fetchTenantScope(
  tenantID: string,
  token: string,
  signal: AbortSignal,
): Promise<TenantScopeResponse> {
  const resp = await fetch(`${API_BASE}/admin/tenants/${tenantID}/scope`, {
    headers: { Authorization: `Bearer ${token}` },
    signal,
  });
  if (!resp.ok) throw new Error(`scope fetch failed (${resp.status})`);
  return resp.json();
}

// -----------------------------------------------------------------------------
// Component
// -----------------------------------------------------------------------------

export interface ImpersonationTenantPickerProps {
  open: boolean;
  onClose: () => void;
  /** Admin JWT (from useAuth). */
  adminToken: string;
  /** Recent sessions from ImpersonationContext. */
  recentSessions: RecentSession[];
  /** Clear-history callback. */
  onClearRecentSessions: () => void;
  /** Called when the admin confirms a tenant + scope. */
  onSelect: (tenant: { id: string; name: string }, scope: ImpersonationScope) => void;
  /** Pre-selected tenant (e.g. when the user clicked "Assume Context" on a row). */
  initialTenant?: { id: string; name: string } | null;
}

export const ImpersonationTenantPicker: React.FC<ImpersonationTenantPickerProps> = ({
  open,
  onClose,
  adminToken,
  recentSessions,
  onClearRecentSessions,
  onSelect,
  initialTenant,
}) => {
  // ---------------------------------------------------------------------------
  // Search state (left pane)
  // ---------------------------------------------------------------------------
  const [query, setQuery] = useState('');
  const [results, setResults] = useState<TenantSearchRow[]>([]);
  const [searching, setSearching] = useState(false);
  const [searchError, setSearchError] = useState<string | null>(null);

  // ---------------------------------------------------------------------------
  // Selected tenant + scope tree (right pane)
  // ---------------------------------------------------------------------------
  const [selectedTenant, setSelectedTenant] = useState<{ id: string; name: string } | null>(
    initialTenant ?? null,
  );
  const [scope, setScope] = useState<ImpersonationScope | null>(
    initialTenant ? { kind: ScopeTenant, id: initialTenant.id } : null,
  );
  const [scopeTree, setScopeTree] = useState<ScopeNode[]>([]);
  const [scopeLoading, setScopeLoading] = useState(false);
  const [scopeError, setScopeError] = useState<string | null>(null);

  // ---------------------------------------------------------------------------
  // Debounced search
  // ---------------------------------------------------------------------------
  const abortRef = useRef<AbortController | null>(null);
  const debounceRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  const runSearch = useCallback(
    async (q: string) => {
      // Cancel any in-flight request.
      abortRef.current?.abort();
      const ctrl = new AbortController();
      abortRef.current = ctrl;

      setSearching(true);
      setSearchError(null);
      try {
        const rows = await searchTenants(q, adminToken, ctrl.signal);
        setResults(rows);
      } catch (err) {
        if ((err as Error).name !== 'AbortError') {
          setSearchError((err as Error).message);
          setResults([]);
        }
      } finally {
        if (abortRef.current === ctrl) setSearching(false);
      }
    },
    [adminToken],
  );

  useEffect(() => {
    if (!open) return;
    if (debounceRef.current) clearTimeout(debounceRef.current);
    debounceRef.current = setTimeout(() => {
      void runSearch(query);
    }, DEBOUNCE_MS);
    return () => {
      if (debounceRef.current) clearTimeout(debounceRef.current);
    };
  }, [query, open, runSearch]);

  // ---------------------------------------------------------------------------
  // Scope tree fetch when tenant is selected
  // ---------------------------------------------------------------------------
  useEffect(() => {
    if (!open || !selectedTenant) {
      setScopeTree([]);
      setScope(null);
      return;
    }
    // Default scope = tenant-wide when a tenant is first picked.
    setScope({ kind: ScopeTenant, id: selectedTenant.id });
    const ctrl = new AbortController();
    setScopeLoading(true);
    setScopeError(null);
    fetchTenantScope(selectedTenant.id, adminToken, ctrl.signal)
      .then((data) => setScopeTree(data.instances ?? []))
      .catch((err) => {
        if ((err as Error).name !== 'AbortError') {
          setScopeError((err as Error).message);
          setScopeTree([]);
        }
      })
      .finally(() => setScopeLoading(false));
    return () => ctrl.abort();
  }, [selectedTenant, open, adminToken]);

  // ---------------------------------------------------------------------------
  // Reset when modal closes
  // ---------------------------------------------------------------------------
  useEffect(() => {
    if (!open) {
      setQuery('');
      setResults([]);
      setSearchError(null);
      setSelectedTenant(initialTenant ?? null);
      setScope(initialTenant ? { kind: ScopeTenant, id: initialTenant.id } : null);
      setScopeTree([]);
      setScopeError(null);
      abortRef.current?.abort();
    }
  }, [open, initialTenant]);

  // ---------------------------------------------------------------------------
  // Handlers
  // ---------------------------------------------------------------------------
  const handlePickTenant = (row: TenantSearchRow) => {
    setSelectedTenant({ id: row.id, name: row.name });
  };

  const handlePickRecent = (session: RecentSession) => {
    setSelectedTenant({ id: session.tenantId, name: session.tenantName });
  };

  const handlePickScopeNode = (node: ScopeNode) => {
    // Map the node type to a scope kind.
    let kind: ImpersonationScopeKind;
    switch (node.type) {
      case 'instance':
        kind = ScopeInstance;
        break;
      case 'product':
        kind = ScopeProduct;
        break;
      case 'datasource':
        kind = ScopeDatasource;
        break;
      default:
        kind = ScopeTenant;
    }
    setScope({ kind, id: node.id });
  };

  const handleClearScope = () => {
    if (!selectedTenant) return;
    setScope({ kind: ScopeTenant, id: selectedTenant.id });
  };

  const handleConfirm = () => {
    if (!selectedTenant || !scope) return;
    onSelect(selectedTenant, scope);
  };

  // ---------------------------------------------------------------------------
  // Render
  // ---------------------------------------------------------------------------
  return (
    <Dialog open={open} onClose={onClose} maxWidth="lg" fullWidth PaperProps={{ sx: { height: '85vh' } }}>
      <DialogTitle sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
        <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
          <BusinessIcon color="primary" />
          <Typography variant="h6">Assume Tenant Context</Typography>
        </Box>
        <IconButton onClick={onClose} size="small" aria-label="close">
          <ClearIcon />
        </IconButton>
      </DialogTitle>

      <DialogContent dividers sx={{ p: 0, display: 'flex', flexDirection: 'column' }}>
        {/* RECENT SESSIONS STRIP */}
        {recentSessions.length > 0 && (
          <Box sx={{ px: 2, py: 1, borderBottom: 1, borderColor: 'divider', bgcolor: 'action.hover' }}>
            <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', mb: 0.5 }}>
              <Typography variant="caption" color="text.secondary" sx={{ display: 'flex', alignItems: 'center', gap: 0.5 }}>
                <HistoryIcon fontSize="inherit" />
                RECENT SESSIONS
              </Typography>
              <Button size="small" onClick={onClearRecentSessions}>Clear</Button>
            </Box>
            <Box sx={{ display: 'flex', gap: 1, flexWrap: 'wrap' }}>
              {recentSessions.map((s) => (
                <Chip
                  key={s.tenantId}
                  label={s.tenantName}
                  size="small"
                  onClick={() => handlePickRecent(s)}
                  color={selectedTenant?.id === s.tenantId ? 'primary' : 'default'}
                  variant={selectedTenant?.id === s.tenantId ? 'filled' : 'outlined'}
                />
              ))}
            </Box>
          </Box>
        )}

        <Box sx={{ flex: 1, display: 'flex', overflow: 'hidden' }}>
          {/* LEFT PANE: tenant search */}
          <Box
            sx={{
              width: '40%',
              borderRight: 1,
              borderColor: 'divider',
              display: 'flex',
              flexDirection: 'column',
              overflow: 'hidden',
            }}
          >
            <Box sx={{ p: 2, borderBottom: 1, borderColor: 'divider' }}>
              <TextField
                fullWidth
                size="small"
                placeholder="Search tenants by name or code..."
                value={query}
                onChange={(e) => setQuery(e.target.value)}
                autoFocus
                InputProps={{
                  startAdornment: (
                    <InputAdornment position="start">
                      <SearchIcon color="action" />
                    </InputAdornment>
                  ),
                  endAdornment: searching ? (
                    <InputAdornment position="end">
                      <CircularProgress size={16} />
                    </InputAdornment>
                  ) : null,
                }}
              />
            </Box>

            <Box sx={{ flex: 1, overflow: 'auto' }}>
              {searchError && (
                <Box sx={{ p: 2 }}>
                  <Alert severity="error">{searchError}</Alert>
                </Box>
              )}

              {!searchError && results.length === 0 && !searching && (
                <Box sx={{ p: 4, textAlign: 'center', color: 'text.secondary' }}>
                  <Typography variant="body2">
                    {query ? 'No tenants match your search.' : 'Type to search across all tenants.'}
                  </Typography>
                </Box>
              )}

              <List sx={{ pt: 0 }}>
                {results.map((row) => (
                  <ListItemButton
                    key={row.id}
                    onClick={() => handlePickTenant(row)}
                    selected={selectedTenant?.id === row.id}
                    sx={{ py: 1.5, borderBottom: 1, borderColor: 'divider' }}
                  >
                    <ListItemIcon>
                      <BusinessIcon color={row.is_suspended ? 'disabled' : (selectedTenant?.id === row.id ? 'primary' : 'inherit')} />
                    </ListItemIcon>
                    <ListItemText
                      primary={row.name}
                      secondary={
                        <Box sx={{ display: 'flex', gap: 0.5, mt: 0.25, flexWrap: 'wrap' }}>
                          {row.code && <Chip label={row.code} size="small" variant="outlined" />}
                          {row.region && <Chip label={row.region} size="small" variant="outlined" />}
                          {row.plan && <Chip label={row.plan} size="small" />}
                          <Typography variant="caption" color="text.secondary">
                            {row.instance_count} instance{row.instance_count === 1 ? '' : 's'}
                          </Typography>
                          {row.is_suspended && (
                            <Chip label="SUSPENDED" size="small" color="error" />
                          )}
                        </Box>
                      }
                    />
                    <ChevronRightIcon color="action" />
                  </ListItemButton>
                ))}
              </List>
            </Box>
          </Box>

          {/* RIGHT PANE: scope tree */}
          <Box sx={{ flex: 1, display: 'flex', flexDirection: 'column', overflow: 'hidden' }}>
            {!selectedTenant ? (
              <Box sx={{ p: 4, textAlign: 'center', color: 'text.secondary' }}>
                <Typography variant="body1">Pick a tenant from the left to choose a scope.</Typography>
              </Box>
            ) : (
              <>
                <Box sx={{ px: 2, py: 1.5, bgcolor: 'action.hover', borderBottom: 1, borderColor: 'divider' }}>
                  <Breadcrumbs separator={<ChevronRightIcon fontSize="small" />}>
                    <Link
                      component="button"
                      variant="body2"
                      onClick={handleClearScope}
                      sx={{ textDecoration: 'none', fontWeight: scope?.kind === ScopeTenant ? 'bold' : 'normal' }}
                      color={scope?.kind === ScopeTenant ? 'primary' : 'text.secondary'}
                    >
                      {selectedTenant.name} (full)
                    </Link>
                    {scope && scope.kind !== ScopeTenant && (
                      <Typography variant="body2" color="primary" sx={{ fontWeight: 'bold' }}>
                        {scope.kind}: {scope.id.slice(0, 8)}…
                      </Typography>
                    )}
                  </Breadcrumbs>
                </Box>

                <Box sx={{ flex: 1, overflow: 'auto' }}>
                  {scopeLoading && (
                    <Box sx={{ p: 4, textAlign: 'center' }}>
                      <CircularProgress />
                    </Box>
                  )}
                  {scopeError && (
                    <Box sx={{ p: 2 }}>
                      <Alert severity="error">Failed to load scope: {scopeError}</Alert>
                    </Box>
                  )}
                  {!scopeLoading && !scopeError && scopeTree.length === 0 && (
                    <Box sx={{ p: 4, textAlign: 'center', color: 'text.secondary' }}>
                      <Typography variant="body2">
                        This tenant has no instances configured. The scope will be tenant-wide.
                      </Typography>
                    </Box>
                  )}
                  {!scopeLoading && scopeTree.length > 0 && (
                    <ScopeTreeView
                      nodes={scopeTree}
                      onPick={handlePickScopeNode}
                      selectedId={scope?.kind === ScopeTenant ? '' : scope?.id ?? ''}
                    />
                  )}
                </Box>
              </>
            )}
          </Box>
        </Box>
      </DialogContent>

      <DialogActions sx={{ p: 2, justifyContent: 'space-between' }}>
        <Box>
          <Typography variant="caption" color="text.secondary" sx={{ display: 'block' }}>
            Selected scope
          </Typography>
          <Typography variant="body2" sx={{ fontWeight: 500 }}>
            {!selectedTenant
              ? '—'
              : scope?.kind === ScopeTenant
              ? `${selectedTenant.name} (full tenant access)`
              : `${selectedTenant.name} → ${scope?.kind}: ${scope?.id.slice(0, 8) ?? '…'}…`}
          </Typography>
        </Box>
        <Box sx={{ display: 'flex', gap: 1 }}>
          <Button onClick={onClose}>Cancel</Button>
          <Button
            variant="contained"
            color="primary"
            onClick={handleConfirm}
            disabled={!selectedTenant || !scope}
          >
            Continue
          </Button>
        </Box>
      </DialogActions>
    </Dialog>
  );
};

// =============================================================================
// Reusable scope-tree view
// =============================================================================

interface ScopeTreeViewProps {
  nodes: ScopeNode[];
  onPick: (node: ScopeNode) => void;
  selectedId: string;
}

function ScopeTreeView({ nodes, onPick, selectedId }: ScopeTreeViewProps) {
  const flat = useMemo(() => flattenScopeTree(nodes, 0), [nodes]);

  return (
    <List sx={{ pt: 0 }}>
      {flat.map(({ node, depth }) => {
        const Icon =
          node.type === 'instance'
            ? InstanceIcon
            : node.type === 'product'
            ? ProductIcon
            : DatasourceIcon;
        const isSelected = node.id === selectedId;

        return (
          <ListItemButton
            key={`${node.type}:${node.id}`}
            onClick={() => onPick(node)}
            selected={isSelected}
            sx={{
              pl: 2 + depth * 3,
              py: 1.25,
              borderBottom: 1,
              borderColor: 'divider',
            }}
          >
            <ListItemIcon sx={{ minWidth: 36 }}>
              <Icon color={isSelected ? 'primary' : 'inherit'} fontSize="small" />
            </ListItemIcon>
            <ListItemText
              primary={node.name}
              secondary={node.type}
              primaryTypographyProps={{ fontWeight: isSelected ? 'bold' : 'normal' }}
            />
          </ListItemButton>
        );
      })}
    </List>
  );
}

function flattenScopeTree(
  nodes: ScopeNode[],
  depth: number,
): Array<{ node: ScopeNode; depth: number }> {
  const out: Array<{ node: ScopeNode; depth: number }> = [];
  for (const n of nodes) {
    out.push({ node: n, depth });
    if (n.children?.length) {
      out.push(...flattenScopeTree(n.children, depth + 1));
    }
  }
  return out;
}

export default ImpersonationTenantPicker;