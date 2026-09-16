import React, { useCallback, useEffect, useState } from 'react';
import {
  Box, Table, TableHead, TableBody, TableRow, TableCell, CircularProgress, Alert, Typography,
  IconButton, Tooltip, Button, Stack, Dialog, DialogTitle, DialogContent, DialogActions, TextField,
  TablePagination, InputAdornment,
} from '@mui/material';
import SearchIcon from '@mui/icons-material/Search';
import VisibilityIcon from '@mui/icons-material/Visibility';
import EditIcon from '@mui/icons-material/Edit';
import DeleteIcon from '@mui/icons-material/Delete';
import AddIcon from '@mui/icons-material/Add';
import apiClient from '../../utils/apiClient';
import { fetchBOTerms } from '../../features/query-builder/services/queryBuilderApi';

/** Excel-style table presentation, set via PropertiesPanel's Table Style section on component.props.tableStyle. */
export interface TableStyleOptions {
  /** Alternating row background, like Excel's "banded rows" table style. */
  banded?: boolean;
  /** Odd-row band color when banded is on. Defaults to the theme's neutral hover tint. */
  bandColor?: string;
  /** Even-row band color when banded is on. Defaults to transparent (matches the page background). */
  bandColorAlt?: string;
  headerBg?: string;
  headerColor?: string;
  /** Draws a border around every cell instead of just row dividers. */
  gridLines?: boolean;
  dense?: boolean;
}

interface BODirectTableProps {
  boId: string;
  /** Shown as a heading above the table so multiple Table widgets on one page are distinguishable - e.g. "Order" vs "Order Allocation". */
  title?: string;
  /** Page size before the user changes it via the pagination control. */
  limit?: number;
  tableStyle?: TableStyleOptions;
  /** Physical column names to show, in order; omitted/empty shows every column the /data endpoint returns. Set via PropertiesPanel's Columns section. */
  visibleColumns?: string[];
  /** Master-detail child scoping: when both are set, only rows where filterField = filterValue load - see BusinessObjectDataSourceConfig.masterFilter. */
  filterField?: string;
  filterValue?: string;
  /**
   * Master-detail selection: when set, clicking a row calls this instead of
   * opening the view dialog, and the selected row is highlighted. The
   * view/edit/delete icon buttons still work as before (they stop
   * propagation so they don't also trigger selection).
   */
  onRowSelect?: (row: Record<string, unknown>) => void;
  /** Row id (row.id) currently selected via onRowSelect, for highlighting. */
  selectedRowId?: string | null;
}

interface BODataResponse {
  total: number;
  page: number;
  limit: number;
  columns: string[];
  rows: Record<string, unknown>[];
}

// System-managed columns: never shown as editable inputs in the add/edit
// dialog. The backend (create/update trigger, DB default) owns these; the
// central validation engine still enforces everything else on submit.
const READONLY_COLUMNS = new Set(['id', 'created_at', 'updated_at']);

type DialogMode = 'add' | 'edit' | 'view' | null;

const DEBOUNCE_MS = 400;

// Renders a Table widget straight off GET/POST/PUT/DELETE
// /api/business-objects/{id}/data - the same endpoint Business Object
// Manager's Records & ORM CRUD tab uses and the one proven to return real
// rows. The semantic-query path (ReportWidgetRenderer -> executeQuery ->
// boresolver.BOSQLGenerator) that every other widget type still goes
// through is for aggregated chart/KPI/gauge queries, a different job than
// a flat record grid - Table widgets take this more direct path instead.
//
// /data already supports search/page/limit/sortBy/sortDir server-side
// (business_object_handlers.go's QueryBORecords) - this component drives
// those params directly rather than re-implementing search or pagination
// client-side over a fixed-size page of rows.
const BODirectTable: React.FC<BODirectTableProps> = ({
  boId, title, limit: initialLimit = 25, tableStyle, visibleColumns,
  filterField, filterValue, onRowSelect, selectedRowId,
}) => {
  const [data, setData] = useState<BODataResponse | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [labels, setLabels] = useState<Record<string, string>>({});

  const [searchInput, setSearchInput] = useState('');
  const [search, setSearch] = useState('');
  const [page, setPage] = useState(0); // MUI TablePagination is 0-indexed; the API is 1-indexed.
  const [rowsPerPage, setRowsPerPage] = useState(initialLimit);

  // Case-insensitive: business_object_fields.technical_name (the source for
  // fetchBOTerms' termKey, e.g. "avg_price") is lowercase, but /data's
  // columns come back upper-cased (e.g. "AVG_PRICE") - a case-sensitive
  // lookup missed every column, silently falling back to the raw physical
  // name for the entire table instead of the semantic display name.
  const labelFor = useCallback((column: string) => labels[column.toLowerCase()] || column, [labels]);

  const [dialogMode, setDialogMode] = useState<DialogMode>(null);
  const [activeRecordId, setActiveRecordId] = useState<string | null>(null);
  const [formValues, setFormValues] = useState<Record<string, string>>({});
  const [saving, setSaving] = useState(false);
  const [formError, setFormError] = useState<string | null>(null);
  const [deleteTarget, setDeleteTarget] = useState<Record<string, unknown> | null>(null);
  const [deleting, setDeleting] = useState(false);

  // Debounce the search box: reload on pause-in-typing (typeahead), not on
  // every keystroke, and reset to page 1 since a new search invalidates
  // whatever page the user was on.
  useEffect(() => {
    const handle = setTimeout(() => {
      setPage(0);
      setSearch(searchInput);
    }, DEBOUNCE_MS);
    return () => clearTimeout(handle);
  }, [searchInput]);

  const load = useCallback(() => {
    setLoading(true);
    setError(null);
    const params = new URLSearchParams({ page: String(page + 1), limit: String(rowsPerPage) });
    if (search) params.set('search', search);
    if (filterField && filterValue) {
      params.set('filterField', filterField);
      params.set('filterValue', filterValue);
    }
    return apiClient<BODataResponse>(`/business-objects/${encodeURIComponent(boId)}/data?${params.toString()}`)
      .then((res) => setData(res))
      .catch((err) => setError(err instanceof Error ? err.message : 'Failed to load records'))
      .finally(() => setLoading(false));
  }, [boId, page, rowsPerPage, search, filterField, filterValue]);

  useEffect(() => {
    load();
  }, [load]);

  // Column headers and form labels should read as business language
  // (semantic term display names, e.g. "SecuritiesID") rather than the
  // physical column names the /data endpoint returns (e.g. "sec_id") -
  // terms without a display_name already fall back to field_name server
  // side, so termKey (the physical name) -> displayName is a complete map.
  useEffect(() => {
    let cancelled = false;
    fetchBOTerms(boId, '')
      .then((terms) => {
        if (cancelled) return;
        const map: Record<string, string> = {};
        terms.forEach((t) => { if (t.termKey) map[t.termKey.toLowerCase()] = t.displayName || t.termKey; });
        setLabels(map);
      })
      .catch(() => { if (!cancelled) setLabels({}); });
    return () => { cancelled = true; };
  }, [boId]);

  const editableColumns = (data?.columns || []).filter((c) => !READONLY_COLUMNS.has(c));

  const openAdd = () => {
    const blank: Record<string, string> = {};
    editableColumns.forEach((c) => { blank[c] = ''; });
    setFormValues(blank);
    setActiveRecordId(null);
    setFormError(null);
    setDialogMode('add');
  };

  const openEditOrView = (row: Record<string, unknown>, mode: 'edit' | 'view') => {
    const values: Record<string, string> = {};
    (data?.columns || []).forEach((c) => {
      const v = row[c];
      values[c] = v === null || v === undefined ? '' : String(v);
    });
    setFormValues(values);
    setActiveRecordId(String(row.id ?? ''));
    setFormError(null);
    setDialogMode(mode);
  };

  const closeDialog = () => setDialogMode(null);

  const handleSave = async () => {
    setSaving(true);
    setFormError(null);
    const record: Record<string, string> = {};
    editableColumns.forEach((c) => { record[c] = formValues[c]; });
    try {
      if (dialogMode === 'add') {
        await apiClient(`/business-objects/${encodeURIComponent(boId)}/data`, {
          method: 'POST',
          body: JSON.stringify({ record }),
        });
      } else if (dialogMode === 'edit' && activeRecordId) {
        await apiClient(`/business-objects/${encodeURIComponent(boId)}/data/${encodeURIComponent(activeRecordId)}`, {
          method: 'PUT',
          body: JSON.stringify({ record }),
        });
      }
      closeDialog();
      await load();
    } catch (err) {
      setFormError(err instanceof Error ? err.message : 'Failed to save record');
    } finally {
      setSaving(false);
    }
  };

  const handleDelete = async () => {
    if (!deleteTarget) return;
    const recordId = String(deleteTarget.id ?? '');
    setDeleting(true);
    try {
      await apiClient(`/business-objects/${encodeURIComponent(boId)}/data/${encodeURIComponent(recordId)}`, {
        method: 'DELETE',
      });
      setDeleteTarget(null);
      await load();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to delete record');
    } finally {
      setDeleting(false);
    }
  };

  const banded = tableStyle?.banded ?? false;
  const bandColor = tableStyle?.bandColor || 'action.hover';
  const bandColorAlt = tableStyle?.bandColorAlt || 'transparent';
  const gridLines = tableStyle?.gridLines ?? false;
  const headerBg = tableStyle?.headerBg || '#1a2332';
  const headerColor = tableStyle?.headerColor || '#ffffff';
  const cellBorderSx = gridLines ? { border: '1px solid', borderColor: 'divider' } : undefined;

  // visibleColumns holds physical names as stored on the field (lower/mixed
  // case, e.g. from technical_name) while /data's columns come back
  // upper-cased - same case mismatch as labelFor, same fix.
  const visibleSet = visibleColumns && visibleColumns.length > 0
    ? new Set(visibleColumns.map((c) => c.toLowerCase()))
    : null;
  const displayColumns = (data?.columns || []).filter((c) => !visibleSet || visibleSet.has(c.toLowerCase()));

  // A child table configured with a masterFilter but no master record
  // selected yet has nothing valid to show - loading with no filter would
  // dump every child row across every master record, not "none selected".
  const awaitingMasterSelection = !!filterField && !filterValue;

  return (
    <Box sx={{ width: '100%', maxWidth: '100%' }}>
      {title && <Typography variant="subtitle2" fontWeight={700} sx={{ mb: 1 }}>{title}</Typography>}
      {awaitingMasterSelection ? (
        <Alert severity="info" sx={{ m: 1 }}>Select a record above to see related rows.</Alert>
      ) : (
      <>
      <Stack direction="row" justifyContent="space-between" alignItems="center" sx={{ mb: 1, gap: 1 }}>
        <TextField
          size="small"
          placeholder="Search…"
          value={searchInput}
          onChange={(e) => setSearchInput(e.target.value)}
          sx={{ maxWidth: 280 }}
          InputProps={{ startAdornment: <InputAdornment position="start"><SearchIcon fontSize="small" /></InputAdornment> }}
        />
        <Button size="small" variant="contained" startIcon={<AddIcon />} onClick={openAdd}>
          Add Record
        </Button>
      </Stack>

      {loading && (
        <Box sx={{ display: 'flex', justifyContent: 'center', p: 3 }}><CircularProgress size={20} /></Box>
      )}
      {!loading && error && <Alert severity="error" sx={{ m: 1 }}>{error}</Alert>}
      {!loading && !error && data && data.rows.length === 0 && (
        <Alert severity="info" sx={{ m: 1 }}>{search ? 'No records match your search.' : 'No records yet.'}</Alert>
      )}
      {!loading && !error && data && data.rows.length > 0 && (
        <Box sx={{ width: '100%', maxWidth: '100%', overflowX: 'auto' }}>
          <Table size={tableStyle?.dense ? 'small' : 'medium'}>
            <TableHead>
              <TableRow>
                {displayColumns.map((col) => (
                  <TableCell key={col} sx={{ fontWeight: 700, bgcolor: headerBg, color: headerColor, whiteSpace: 'nowrap', ...cellBorderSx }}>
                    {labelFor(col)}
                  </TableCell>
                ))}
                <TableCell align="right" sx={{ fontWeight: 700, bgcolor: headerBg, color: headerColor, whiteSpace: 'nowrap', ...cellBorderSx }}>
                  Actions
                </TableCell>
              </TableRow>
            </TableHead>
            <TableBody>
              {data.rows.map((row, i) => {
                const rowId = String(row.id ?? '');
                const isSelected = !!onRowSelect && !!selectedRowId && rowId === selectedRowId;
                return (
                <TableRow
                  key={String(row.id ?? i)}
                  hover
                  selected={isSelected}
                  onClick={onRowSelect ? () => onRowSelect(row) : undefined}
                  sx={{
                    cursor: onRowSelect ? 'pointer' : undefined,
                    ...(banded ? { bgcolor: i % 2 === 1 ? bandColor : bandColorAlt } : undefined),
                  }}
                >
                  {displayColumns.map((col) => (
                    <TableCell key={col} sx={cellBorderSx}>
                      {row[col] === null || row[col] === undefined ? '—' : String(row[col])}
                    </TableCell>
                  ))}
                  <TableCell align="right" sx={cellBorderSx} onClick={(e) => e.stopPropagation()}>
                    <Tooltip title="View">
                      <IconButton size="small" onClick={() => openEditOrView(row, 'view')}>
                        <VisibilityIcon fontSize="small" />
                      </IconButton>
                    </Tooltip>
                    <Tooltip title="Edit">
                      <IconButton size="small" onClick={() => openEditOrView(row, 'edit')}>
                        <EditIcon fontSize="small" />
                      </IconButton>
                    </Tooltip>
                    <Tooltip title="Delete">
                      <IconButton size="small" onClick={() => setDeleteTarget(row)}>
                        <DeleteIcon fontSize="small" />
                      </IconButton>
                    </Tooltip>
                  </TableCell>
                </TableRow>
                );
              })}
            </TableBody>
          </Table>
          <TablePagination
            component="div"
            count={data.total}
            page={page}
            onPageChange={(_, newPage) => setPage(newPage)}
            rowsPerPage={rowsPerPage}
            onRowsPerPageChange={(e) => { setRowsPerPage(parseInt(e.target.value, 10)); setPage(0); }}
            rowsPerPageOptions={[10, 25, 50, 100]}
          />
        </Box>
      )}
      </>
      )}

      <Dialog open={dialogMode !== null} onClose={closeDialog} maxWidth="sm" fullWidth>
        <DialogTitle>
          {dialogMode === 'add' ? 'Add Record' : dialogMode === 'edit' ? 'Edit Record' : 'View Record'}
        </DialogTitle>
        <DialogContent>
          <Stack spacing={2} sx={{ mt: 1 }}>
            {formError && <Alert severity="error">{formError}</Alert>}
            {dialogMode === 'view'
              ? (data?.columns || []).map((col) => (
                  <TextField key={col} label={labelFor(col)} value={formValues[col] ?? ''} fullWidth disabled />
                ))
              : editableColumns.map((col) => (
                  <TextField
                    key={col}
                    label={labelFor(col)}
                    value={formValues[col] ?? ''}
                    onChange={(e) => setFormValues((prev) => ({ ...prev, [col]: e.target.value }))}
                    fullWidth
                  />
                ))}
          </Stack>
        </DialogContent>
        <DialogActions>
          <Button onClick={closeDialog}>{dialogMode === 'view' ? 'Close' : 'Cancel'}</Button>
          {dialogMode !== 'view' && (
            <Button variant="contained" onClick={handleSave} disabled={saving}>
              {saving ? 'Saving…' : 'Save'}
            </Button>
          )}
        </DialogActions>
      </Dialog>

      <Dialog open={!!deleteTarget} onClose={() => setDeleteTarget(null)}>
        <DialogTitle>Delete this record?</DialogTitle>
        <DialogContent>
          <Typography variant="body2">This can't be undone.</Typography>
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setDeleteTarget(null)}>Cancel</Button>
          <Button color="error" variant="contained" onClick={handleDelete} disabled={deleting}>
            {deleting ? 'Deleting…' : 'Delete'}
          </Button>
        </DialogActions>
      </Dialog>
    </Box>
  );
};

export default BODirectTable;
