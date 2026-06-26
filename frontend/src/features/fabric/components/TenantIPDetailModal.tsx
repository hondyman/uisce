import { useMemo, useState } from 'react';
import {
  Dialog,
  DialogContent,
  DialogActions,
  TextField,
  Button,
  Box,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  Paper,
  Typography,
  IconButton,
  Tooltip,
  Menu,
  MenuItem,
  ListItemIcon,
  ListItemText,
  
} from '@mui/material';
import ModalHeader from '../../../components/ModalHeader';
import { Download as DownloadIcon, Search as SearchIcon, Add as AddIcon, Edit as EditIcon, Delete as DeleteIcon } from '@mui/icons-material';
import { IPWhitelistEntry, Tenant } from '../types/ipWhitelist';
import { exportToCSV, exportToJSON } from '../utils/exportUtils';
import IPAddEditDialog from './IPAddEditDialog';
import { useIPWhitelistAPI } from '../hooks/useIPWhitelist';

interface TenantIPDetailModalProps {
  open: boolean;
  onClose: () => void;
  tenant: Tenant | null;
  ipAddresses: IPWhitelistEntry[];
  tenants: Tenant[];
  onChanged?: () => void | Promise<void>;
}

const TenantIPDetailModal: React.FC<TenantIPDetailModalProps> = ({ open, onClose, tenant, ipAddresses, tenants, onChanged }) => {
  const [search, setSearch] = useState('');
  const [exportMenuAnchor, setExportMenuAnchor] = useState<null | HTMLElement>(null);
  const [addOpen, setAddOpen] = useState(false);
  const [editOpen, setEditOpen] = useState(false);
  const [selectedEntry, setSelectedEntry] = useState<IPWhitelistEntry | null>(null);
  const api = useIPWhitelistAPI();

  const filtered = useMemo(() => {
    const q = search.trim().toLowerCase();
    if (!q) return ipAddresses;
    return ipAddresses.filter(ip =>
      ip.ipAddress.toLowerCase().includes(q) ||
      (ip.label || '').toLowerCase().includes(q) ||
      (ip.description || '').toLowerCase().includes(q)
    );
  }, [search, ipAddresses]);

  const handleOpenExport = (e: React.MouseEvent<HTMLElement>) => setExportMenuAnchor(e.currentTarget);
  const handleCloseExport = () => setExportMenuAnchor(null);

  const exportDataRows = useMemo(() => {
    return filtered.map(ip => ({
      tenantId: tenant?.id || '',
      tenantName: tenant?.displayName || '',
      ipAddress: ip.ipAddress,
      label: ip.label || '',
      description: ip.description || '',
      createdAt: ip.createdAt || '',
      updatedAt: ip.updatedAt || ''
    }));
  }, [filtered, tenant]);

  const doExportCSV = () => {
    exportToCSV(exportDataRows, `${(tenant?.displayName || 'tenant').replace(/\s+/g, '-')}-ips`);
    handleCloseExport();
  };
  const doExportJSON = () => {
    // Wrap to match JSON structure
    exportToJSON({
      exportDate: new Date().toISOString(),
      totalEntries: exportDataRows.length,
      totalTenants: 1,
      data: exportDataRows
    }, `${(tenant?.displayName || 'tenant').replace(/\s+/g, '-')}-ips`);
    handleCloseExport();
  };

  const handleAddSave = async (ipData: { ipAddress: string; label?: string; description?: string; tenantIds: string[]; allTenants?: boolean; }) => {
    if (!tenant) return;
    // Prevent duplicates for this tenant
    const normalize = (s: string) => s.trim().toLowerCase();
    if (ipAddresses.some(e => normalize(e.ipAddress) === normalize(ipData.ipAddress))) {
      throw new Error('This IP address already exists for this tenant.');
    }
    const primary = ipData.allTenants ? '__ALL_TENANTS__' : tenant.id;
    const additional = ipData.allTenants ? [] : (ipData.tenantIds.filter(id => id !== tenant.id));
    await api.addIPWhitelist(primary, ipData.ipAddress, ipData.label, ipData.description, additional, { allTenants: ipData.allTenants });
    await onChanged?.();
  };

  const handleEditSave = async (ipData: { ipAddress: string; label?: string; description?: string; tenantIds: string[]; allTenants?: boolean; }) => {
    if (!selectedEntry) return;
    await api.updateIPAssignments(
      selectedEntry.ipAddress,
      selectedEntry.tenantIds[0] || '',
      ipData.tenantIds,
      { allTenants: ipData.allTenants, prevTenantIds: selectedEntry.tenantIds }
    );
    await onChanged?.();
  };

  const handleRemove = async (ip: IPWhitelistEntry) => {
    if (!tenant) return;
    await api.removeIPWhitelist(tenant.id, ip.ipAddress);
    await onChanged?.();
  };

  return (
    <Dialog open={open} onClose={onClose} maxWidth="md" fullWidth>
      <ModalHeader title={tenant ? `IP Addresses for ${tenant.displayName}` : 'IP Addresses'} onClose={onClose} />
      <DialogContent>
        <Box sx={{ display: 'flex', alignItems: 'center', gap: 1, mb: 2 }}>
          <Tooltip title="Add IP"><span>
            <IconButton onClick={() => { setSelectedEntry(null); setAddOpen(true); }} aria-label="Add IP">
              <AddIcon />
            </IconButton>
          </span></Tooltip>
          <TextField
            size="small"
            placeholder="Search IP, label, description..."
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            InputProps={{ startAdornment: <SearchIcon sx={{ mr: 1, color: 'action.active' }} /> }}
            fullWidth
          />
          <Tooltip title="Export">
            <IconButton onClick={handleOpenExport} aria-label="Export">
              <DownloadIcon />
            </IconButton>
          </Tooltip>
          <Menu anchorEl={exportMenuAnchor} open={Boolean(exportMenuAnchor)} onClose={handleCloseExport}>
            <MenuItem onClick={doExportCSV}>
              <ListItemIcon>
                <DownloadIcon fontSize="small" />
              </ListItemIcon>
              <ListItemText>Export CSV (filtered)</ListItemText>
            </MenuItem>
            <MenuItem onClick={doExportJSON}>
              <ListItemIcon>
                <DownloadIcon fontSize="small" />
              </ListItemIcon>
              <ListItemText>Export JSON (filtered)</ListItemText>
            </MenuItem>
          </Menu>
        </Box>

        <Typography variant="body2" color="text.secondary" sx={{ mb: 1 }}>
          Showing {filtered.length} of {ipAddresses.length} IPs
        </Typography>

        <TableContainer component={Paper}>
          <Table size="small">
            <TableHead>
              <TableRow>
                <TableCell>IP Address</TableCell>
                <TableCell>Label</TableCell>
                <TableCell>Description</TableCell>
                <TableCell align="right">Actions</TableCell>
              </TableRow>
            </TableHead>
            <TableBody>
              {filtered.length === 0 ? (
                <TableRow>
                  <TableCell colSpan={4} align="center">No IP addresses match your search</TableCell>
                </TableRow>
              ) : (
                filtered.map((ip, idx) => (
                  <TableRow key={`${ip.ipAddress}-${idx}`}>
                    <TableCell>
                      <Typography variant="body2" fontFamily="monospace">{ip.ipAddress}</Typography>
                    </TableCell>
                    <TableCell>{ip.label || '—'}</TableCell>
                    <TableCell>
                      <Typography variant="body2" sx={{ maxWidth: 420, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
                        {ip.description || '—'}
                      </Typography>
                    </TableCell>
                    <TableCell align="right">
                      <Tooltip title="Edit"><span>
                        <IconButton size="small" onClick={() => { setSelectedEntry(ip); setEditOpen(true); }} aria-label="Edit">
                          <EditIcon fontSize="small" />
                        </IconButton>
                      </span></Tooltip>
                      <Tooltip title="Remove"><span>
                        <IconButton size="small" color="error" onClick={() => handleRemove(ip)} aria-label="Remove">
                          <DeleteIcon fontSize="small" />
                        </IconButton>
                      </span></Tooltip>
                    </TableCell>
                  </TableRow>
                ))
              )}
            </TableBody>
          </Table>
        </TableContainer>
      </DialogContent>
      <DialogActions>
        <Button onClick={onClose}>Close</Button>
      </DialogActions>

      {/* Add / Edit dialogs */}
      <IPAddEditDialog
        open={addOpen}
        onClose={() => setAddOpen(false)}
        onSave={handleAddSave}
        tenants={tenants}
        initialTenantId={tenant?.id}
      />
      <IPAddEditDialog
        open={editOpen}
        onClose={() => setEditOpen(false)}
        onSave={handleEditSave}
        tenants={tenants}
        editingEntry={selectedEntry || undefined}
      />
    </Dialog>
  );
};

export default TenantIPDetailModal;
