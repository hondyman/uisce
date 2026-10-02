import React, { useMemo, useState } from 'react';
import {
  Alert, Box, Card, CardActionArea, CardContent, CircularProgress, Grid, IconButton, InputAdornment,
  Paper, Stack, Table, TableBody, TableCell, TableHead, TableRow, TextField, ToggleButton,
  ToggleButtonGroup, Typography,
} from '@mui/material';
import SearchIcon from '@mui/icons-material/Search';
import AddIcon from '@mui/icons-material/Add';
import MoreVertIcon from '@mui/icons-material/MoreVert';
import ViewModuleIcon from '@mui/icons-material/ViewModule';
import ViewListIcon from '@mui/icons-material/ViewList';
import { CoreIcon, CustomIcon } from './CoreCustomIcons';

export type CatalogViewMode = 'tiles' | 'grid';
export type CatalogScope = 'all' | 'core' | 'custom';

export interface CatalogColumn<T> {
  key: string;
  header: string;
  render: (item: T) => React.ReactNode;
  width?: number | string;
}

export interface CatalogListProps<T> {
  items: T[];
  getId: (item: T) => string;
  getTitle: (item: T) => string;
  /** Short line under the title on a tile (e.g. a slug). */
  getSubtitle?: (item: T) => string | undefined;
  getDescription?: (item: T) => string | undefined;
  /** true = gold-copy/core item, false = tenant-created custom item. */
  isCore: (item: T) => boolean;
  /** Extra text matched by search, beyond title/subtitle/description. */
  getSearchText?: (item: T) => string;
  /** Chips/badges shown on a tile. */
  renderTileMeta?: (item: T) => React.ReactNode;
  /** Table columns after the standard core/custom icon + name columns. */
  columns: CatalogColumn<T>[];
  onOpen: (item: T) => void;
  /** Shows a per-item "⋮" button; the caller owns the menu it opens. */
  onActions?: (item: T, anchor: HTMLElement) => void;
  /** Extra toolbar controls (filters, buttons) placed before the toggles. */
  toolbarExtra?: React.ReactNode;
  /** Extra predicate applied after search and scope filtering. */
  filter?: (item: T) => boolean;
  /** localStorage key remembering tile/grid per list; omit to not persist. */
  storageKey?: string;
  defaultView?: CatalogViewMode;
  searchPlaceholder?: string;
  loading?: boolean;
  error?: string | null;
  onDismissError?: () => void;
  emptyMessage?: string;
  noMatchMessage?: string;
  /** Controlled scope, for pages whose own widgets (e.g. KPI cards) also set it. */
  scope?: CatalogScope;
  onScopeChange?: (scope: CatalogScope) => void;
  /** Replaces the default title/subtitle/description search match. */
  matchesSearch?: (item: T, query: string) => boolean;
  /** Adds a dashed "create" tile at the end of the tile view. */
  createLabel?: string;
  onCreate?: () => void;
  /** Controlled search text, for lists that search on the server. */
  search?: string;
  onSearchChange?: (q: string) => void;
  /** true = items are already filtered by search upstream; skip client matching. */
  serverSearch?: boolean;
  /** Rendered under the list (e.g. pagination). */
  footer?: React.ReactNode;
}

function readView(storageKey: string | undefined, fallback: CatalogViewMode): CatalogViewMode {
  if (!storageKey) return fallback;
  try {
    const v = localStorage.getItem(storageKey);
    return v === 'grid' || v === 'tiles' ? v : fallback;
  } catch {
    return fallback;
  }
}

/**
 * The platform's standard list layout: search, All/Core/Custom scope, a
 * tile/grid toggle, and the Core (gold copy) vs Custom (tenant) icon on every
 * item. Pages own their data, filters and action menus; this owns the layout
 * so every catalog looks and behaves the same.
 */
export function CatalogList<T>(props: CatalogListProps<T>): React.ReactElement {
  const {
    items, getId, getTitle, getSubtitle, getDescription, isCore, getSearchText, renderTileMeta, columns,
    onOpen, onActions, toolbarExtra, filter, storageKey, defaultView = 'tiles', searchPlaceholder,
    loading, error, onDismissError, scope: scopeProp, onScopeChange, matchesSearch, createLabel, onCreate, search: searchProp, onSearchChange, serverSearch, footer, emptyMessage = 'Nothing here yet.', noMatchMessage = 'Nothing matches your search.',
  } = props;

  const [searchState, setSearchState] = useState('');
  const search = searchProp ?? searchState;
  const setSearch = (q: string) => { setSearchState(q); onSearchChange?.(q); };
  const [scopeState, setScopeState] = useState<CatalogScope>('all');
  const scope = scopeProp ?? scopeState;
  const setScope = (v: CatalogScope) => { setScopeState(v); onScopeChange?.(v); };
  const [view, setView] = useState<CatalogViewMode>(() => readView(storageKey, defaultView));

  const changeView = (v: CatalogViewMode) => {
    setView(v);
    if (!storageKey) return;
    try { localStorage.setItem(storageKey, v); } catch { /* storage unavailable */ }
  };

  const visible = useMemo(() => {
    const q = search.trim().toLowerCase();
    return items.filter((item) => {
      if (scope === 'core' && !isCore(item)) return false;
      if (scope === 'custom' && isCore(item)) return false;
      if (filter && !filter(item)) return false;
      if (!q || serverSearch) return true;
      if (matchesSearch) return matchesSearch(item, q);
      const hay = [getTitle(item), getSubtitle?.(item), getDescription?.(item), getSearchText?.(item)]
        .filter(Boolean).join(' ').toLowerCase();
      return hay.includes(q);
    });
  }, [items, search, serverSearch, scope, filter, matchesSearch, isCore, getTitle, getSubtitle, getDescription, getSearchText]);

  const kindIcon = (item: T) => (isCore(item) ? <CoreIcon /> : <CustomIcon />);

  const actionsButton = (item: T, sx?: object) => onActions && (
    <IconButton
      size="small"
      sx={sx}
      aria-label="Actions"
      onClick={(e) => { e.stopPropagation(); onActions(item, e.currentTarget); }}
    >
      <MoreVertIcon fontSize="small" />
    </IconButton>
  );

  return (
    <Box>
      <Stack direction="row" spacing={2} alignItems="center" sx={{ mb: 3 }}>
        <TextField
          fullWidth
          placeholder={searchPlaceholder ?? 'Search…'}
          value={search}
          onChange={(e) => setSearch(e.target.value)}
          InputProps={{ startAdornment: <InputAdornment position="start"><SearchIcon fontSize="small" /></InputAdornment> }}
        />
        {toolbarExtra}
        <ToggleButtonGroup exclusive size="small" value={scope} onChange={(_, v) => v && setScope(v)}>
          <ToggleButton value="all" sx={{ textTransform: 'none', px: 2 }}>All</ToggleButton>
          <ToggleButton value="core" sx={{ textTransform: 'none', px: 2 }}>Core</ToggleButton>
          <ToggleButton value="custom" sx={{ textTransform: 'none', px: 2 }}>Custom</ToggleButton>
        </ToggleButtonGroup>
        <ToggleButtonGroup exclusive size="small" value={view} onChange={(_, v) => v && changeView(v)}>
          <ToggleButton value="tiles" aria-label="Tile view"><ViewModuleIcon fontSize="small" /></ToggleButton>
          <ToggleButton value="grid" aria-label="Grid view"><ViewListIcon fontSize="small" /></ToggleButton>
        </ToggleButtonGroup>
      </Stack>

      {loading && <Box sx={{ display: 'flex', justifyContent: 'center', p: 6 }}><CircularProgress /></Box>}
      {error && <Alert severity="error" sx={{ mb: 2 }} onClose={onDismissError}>{error}</Alert>}
      {!loading && visible.length === 0 && (
        <Alert severity="info">{items.length === 0 ? emptyMessage : noMatchMessage}</Alert>
      )}

      {view === 'grid' && visible.length > 0 && (
        <Paper variant="outlined" sx={{ borderRadius: 2, overflowX: 'auto' }}>
          <Table size="small">
            <TableHead>
              <TableRow>
                <TableCell width={48} />
                <TableCell>Name</TableCell>
                {columns.map((c) => <TableCell key={c.key} width={c.width}>{c.header}</TableCell>)}
                {onActions && <TableCell width={48} />}
              </TableRow>
            </TableHead>
            <TableBody>
              {visible.map((item) => (
                <TableRow key={getId(item)} hover sx={{ cursor: 'pointer' }} onClick={() => onOpen(item)}>
                  <TableCell>{kindIcon(item)}</TableCell>
                  <TableCell><Typography variant="body2" fontWeight={600}>{getTitle(item)}</Typography></TableCell>
                  {columns.map((c) => <TableCell key={c.key}>{c.render(item)}</TableCell>)}
                  {onActions && <TableCell>{actionsButton(item)}</TableCell>}
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </Paper>
      )}

      {view === 'tiles' && (
        <Grid container spacing={2}>
          {visible.map((item) => (
            <Grid key={getId(item)} size={{ xs: 12, sm: 6, md: 4 }}>
              <Card variant="outlined" sx={{ borderRadius: 2, height: '100%', position: 'relative' }}>
                {actionsButton(item, { position: 'absolute', top: 6, right: 6, zIndex: 1 })}
                <CardActionArea onClick={() => onOpen(item)} sx={{ height: '100%', p: 0.5 }}>
                  <CardContent>
                    <Stack direction="row" spacing={1} alignItems="center" sx={{ mb: 1, pr: 3 }}>
                      {kindIcon(item)}
                      <Typography variant="subtitle1" fontWeight={700} noWrap>{getTitle(item)}</Typography>
                    </Stack>
                    {getSubtitle?.(item) && (
                      <Typography variant="caption" color="text.secondary" sx={{ display: 'block', mb: 0.5 }}>
                        {getSubtitle(item)}
                      </Typography>
                    )}
                    {getDescription?.(item) && (
                      <Typography variant="body2" color="text.secondary" sx={{ mb: 1.5, display: '-webkit-box', WebkitLineClamp: 2, WebkitBoxOrient: 'vertical', overflow: 'hidden' }}>
                        {getDescription(item)}
                      </Typography>
                    )}
                    {renderTileMeta?.(item)}
                  </CardContent>
                </CardActionArea>
              </Card>
            </Grid>
          ))}
          {createLabel && onCreate && (
            <Grid size={{ xs: 12, sm: 6, md: 4 }}>
              <Card
                variant="outlined"
                onClick={onCreate}
                sx={{ cursor: 'pointer', height: '100%', minHeight: 140, borderRadius: 2, borderStyle: 'dashed', bgcolor: 'action.hover', display: 'flex', alignItems: 'center', justifyContent: 'center', '&:hover': { borderColor: 'primary.main' } }}
              >
                <Stack alignItems="center" spacing={1}>
                  <AddIcon color="primary" />
                  <Typography variant="body2" fontWeight={700}>{createLabel}</Typography>
                </Stack>
              </Card>
            </Grid>
          )}
        </Grid>
      )}
      {footer}
    </Box>
  );
}

export default CatalogList;
