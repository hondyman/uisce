import React, { useEffect, useState, lazy, Suspense } from 'react';
import styles from './DatasourceGrid.module.css';
import { Box, IconButton, Tooltip, TextField, InputAdornment, Snackbar } from '@mui/material';
import EditIcon from '@mui/icons-material/Edit';
import DeleteIcon from '@mui/icons-material/Delete';
import PlayArrowIcon from '@mui/icons-material/PlayArrow';
import OpenInNewIcon from '@mui/icons-material/OpenInNew';
import SearchIcon from '@mui/icons-material/Search';
import CloseIcon from '@mui/icons-material/Close';
import CheckCircleIcon from '@mui/icons-material/CheckCircle';
import WarningAmberIcon from '@mui/icons-material/WarningAmber';
import ContentCopyIcon from '@mui/icons-material/ContentCopy';
import type { DataSource, Product, TenantInstance } from '../types';

interface DatasourceGridProps {
  tenant: any;
  product: Product;
  datasources: any[]; // prepared by parent
  onSelect: (tenant: TenantInstance, product: Product, datasource: DataSource) => void;
  onRunScanner: (datasource: DataSource) => void;
  onEditDatasource: (datasource: DataSource) => void;
  onDeleteDatasource: (datasourceId: string) => void;
  telemetryHook?: (event: string, payload?: any) => void;
  // optional injection used by tests to avoid loading the real DataGrid
  dataGridModule?: any;
}

const DatasourceGrid: React.FC<DatasourceGridProps> = (props) => {
  const { tenant, product, datasources, onSelect, onRunScanner, onEditDatasource, onDeleteDatasource } = props;
  const [DataGridModule, setDataGridModule] = useState<any>(props.dataGridModule || null);
  const [filterText, setFilterText] = useState<string>('');
  const [selectedRowId, setSelectedRowId] = useState<string | null>(null);
  const [focusedRowIndex, setFocusedRowIndex] = useState<number>(-1);
  const [copiedOpen, setCopiedOpen] = useState(false);
  const [copyAnnouncement, setCopyAnnouncement] = useState<string>('');
  const prevFocusedRowId = React.useRef<string | null>(null);
  const DatasourceDetailsDrawer = lazy(() => import('./DatasourceDetailsDrawer')) as unknown as React.ComponentType<any>;

  useEffect(() => {
    if (props.dataGridModule) return;
    let mounted = true;
    import('@mui/x-data-grid').then((mod) => {
      if (mounted) setDataGridModule(mod);
    });
    return () => { mounted = false; };
  }, [props.dataGridModule]);

  // move DOM focus to the currently selected/focused row so screen readers and
  // keyboard users get a visible focus target. We add a temporary tabIndex so
  // the row becomes focusable and remove it from the previous row.
  useEffect(() => {
    const currentId = selectedRowId;
    try {
      // remove tabIndex from previous
      if (prevFocusedRowId.current && prevFocusedRowId.current !== currentId) {
        const prevEl = document.querySelector(`.MuiDataGrid-row[data-id="${prevFocusedRowId.current}"]`) as HTMLElement | null;
        if (prevEl) prevEl.removeAttribute('tabindex');
      }
      if (currentId) {
        const el = document.querySelector(`.MuiDataGrid-row[data-id="${currentId}"]`) as HTMLElement | null;
        if (el) {
          el.setAttribute('tabindex', '-1');
          el.focus();
          // add a CSS class so styling is in CSS instead of inline styles
          el.classList.add(styles.rowFocusOutline);
          const handleBlur = () => { el.classList.remove(styles.rowFocusOutline); el.removeEventListener('blur', handleBlur); };
          el.addEventListener('blur', handleBlur);
        }
      }
      prevFocusedRowId.current = currentId || null;
    } catch (e) {
      // defensive: DOM may not be ready yet
    }
  }, [selectedRowId]);

  if (!DataGridModule) {
    return <div className="centered-loader">Loading data sources...</div>;
  }

  const DataGrid = DataGridModule.DataGrid as any;

  const rows = datasources.map(ds => ({
    id: ds.id != null && typeof ds.id !== 'object' ? String(ds.id) : '',
    source_name: ds.source_name,
    name: ds.alpha_datasource?.datasource_name,
    type: ds.alpha_datasource?.datasource_type,
    status: ds.is_active ? 'Active' : 'Inactive',
    fullDatasource: ds
  }));

  const filteredRows = rows.filter(r => {
    if (!filterText) return true;
    const t = filterText.toLowerCase();
    return (String(r.name || '').toLowerCase().includes(t) || String(r.source_name || '').toLowerCase().includes(t) || String(r.type || '').toLowerCase().includes(t));
  });

  const columns = [
    {
      field: 'source_name',
      headerName: 'Source Name',
      flex: 1,
      sortable: true,
      renderCell: (params: any) => (
        <Tooltip title={params.value || ''}>
          <Box component="span" sx={{ whiteSpace: 'nowrap', overflow: 'hidden', textOverflow: 'ellipsis' }}>{params.value}</Box>
        </Tooltip>
      )
    },
    {
      field: 'name',
      headerName: 'Datasource',
      flex: 2,
      sortable: true,
      renderCell: (params: any) => (
        <Tooltip title={params.row.name || ''}>
          <Box component="span" sx={{ whiteSpace: 'nowrap', overflow: 'hidden', textOverflow: 'ellipsis' }}>{params.row.name}</Box>
        </Tooltip>
      )
    },
    { field: 'type', headerName: 'Type', width: 120, sortable: true },
    { field: 'status', headerName: 'Status', width: 80, renderCell: (params: any) => (
      <Tooltip title={params.value}>
        {params.value === 'Active' ? <CheckCircleIcon color="success" /> : <WarningAmberIcon color="warning" />}
      </Tooltip>
    ) },
    { field: 'actions', headerName: 'Actions', width: 160, renderCell: (params: any) => (
      <Box className="ds-actions" sx={{ display: 'flex', gap: 1 }}>
        <Tooltip title="Open">
          <IconButton aria-label={`open datasource ${String(params.row.id)}`} size="small" onClick={() => { onSelect(tenant, product, params.row.fullDatasource); setFocusedRowIndex(params.api.getRowIndex(params.id)); props.telemetryHook?.('datasource.open.click', { datasourceId: params.row.id }); }}>
            <OpenInNewIcon fontSize="small" />
          </IconButton>
        </Tooltip>
        <Tooltip title="Run">
          <IconButton aria-label="run" size="small" onClick={() => onRunScanner(params.row.fullDatasource)}>
            <PlayArrowIcon fontSize="small" />
          </IconButton>
        </Tooltip>
        <Tooltip title="Copy ID">
          <IconButton
            aria-label={`copy datasource id ${String(params.row.id)}`}
            size="small"
            onClick={async () => {
              try {
                const toCopy = String(params.row.id || params.row.fullDatasource?.id || '');
                await navigator.clipboard.writeText(toCopy);
                setCopiedOpen(true);
                setCopyAnnouncement(`Copied datasource id ${toCopy}`);
                props.telemetryHook?.('datasource.copy', { datasourceId: params.row.id });
                // clear announcement after a short delay
                setTimeout(() => setCopyAnnouncement(''), 2000);
              } catch (e) {
                const { devError } = require('../utils/devLogger');
                devError('copy failed', e);
                setCopyAnnouncement('Failed to copy datasource id');
                props.telemetryHook?.('datasource.copy_failed', { datasourceId: params.row.id, error: String(e) });
                setTimeout(() => setCopyAnnouncement(''), 2000);
              }
            }}
          >
            <ContentCopyIcon fontSize="small" />
          </IconButton>
        </Tooltip>
        <Tooltip title="Edit">
          <IconButton aria-label="edit" size="small" onClick={() => onEditDatasource(params.row.fullDatasource)}>
            <EditIcon fontSize="small" />
          </IconButton>
        </Tooltip>
        <Tooltip title="Unassign">
          <IconButton aria-label="unassign" size="small" color="error" onClick={() => onDeleteDatasource(params.row.id)}>
            <DeleteIcon fontSize="small" />
          </IconButton>
        </Tooltip>
      </Box>
    )}
  ];

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (!filteredRows.length) return;
    if (e.key === 'ArrowDown') {
      e.preventDefault();
      setFocusedRowIndex((prev) => {
        const next = Math.min(prev + 1, filteredRows.length - 1);
        setSelectedRowId(filteredRows[next]?.id || null);
        return next;
      });
    } else if (e.key === 'ArrowUp') {
      e.preventDefault();
      setFocusedRowIndex((prev) => {
        const next = Math.max(prev - 1, 0);
        setSelectedRowId(filteredRows[next]?.id || null);
        return next;
      });
    } else if (e.key === 'Enter' || e.key === ' ') {
      e.preventDefault();
      const row = filteredRows[focusedRowIndex];
      if (row) {
        onSelect(tenant, product, row.fullDatasource);
        props.telemetryHook?.('datasource.open.keyboard', { datasourceId: row.id });
      }
    }
  };

  return (
    <Box sx={{ height: '100%', position: 'relative', display: 'flex', flexDirection: 'column', gap: 1 }} onKeyDown={handleKeyDown} tabIndex={0}>
      <TextField
        size="small"
        placeholder="Filter datasources..."
        value={filterText}
        onChange={(e) => setFilterText(e.target.value)}
        InputProps={{
          startAdornment: (
            <InputAdornment position="start">
              <SearchIcon fontSize="small" />
            </InputAdornment>
          ),
          endAdornment: filterText ? (
            <InputAdornment position="end">
              <IconButton size="small" onClick={() => setFilterText('')} aria-label="clear filter"><CloseIcon fontSize="small" /></IconButton>
            </InputAdornment>
          ) : undefined,
        }}
      />

      <Box role="region" aria-label="datasource grid" sx={{ flex: 1, '& .MuiDataGrid-row .ds-actions': { opacity: 0.08, transition: 'opacity 150ms' }, '& .MuiDataGrid-row:hover .ds-actions': { opacity: 1 }, '& .MuiDataGrid-row:hover': { backgroundColor: 'action.hover' } }}>
        <DataGrid
          rows={filteredRows}
          columns={columns}
          hideFooter
          density="compact"
      getRowClassName={(params: any) => params.id === selectedRowId ? 'Mui-selected' : ''}
      onRowClick={(params: any) => { setSelectedRowId(String(params.id)); setFocusedRowIndex(params.api.getRowIndex(params.id)); props.telemetryHook?.('datasource.row.click', { datasourceId: params.id }); onSelect(tenant, product, params.row.fullDatasource); }}
        />
      </Box>

      <Snackbar open={copiedOpen} autoHideDuration={1600} onClose={() => setCopiedOpen(false)} message="Copied to clipboard" />

      {/* aria-live region for screen reader announcements when copying */}
      <div aria-live="polite" aria-atomic="true" className="sr-only">
        {copyAnnouncement}
      </div>

      <Suspense fallback={null}>
        {/* lazy detail panel: only loads when user opens rows (via Open action) */}
        {/* Note: we don't control parent navigation — this is an inline detail preview */}
        {/* The drawer shows when selectedRowId is set */}
        {selectedRowId && <DatasourceDetailsDrawer open={!!selectedRowId} datasource={rows.find(r => r.id === selectedRowId)?.fullDatasource} onClose={() => setSelectedRowId(null)} />}
      </Suspense>
    </Box>
  );
};

export default DatasourceGrid;
